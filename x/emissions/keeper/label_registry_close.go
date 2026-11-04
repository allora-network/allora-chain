package keeper

import (
	"context"
	"sort"

	errorsmod "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// CompactRegistryAndRemapInferences freezes the close-time view of an epoch
// label registry. It validates the temporary first-seen registry, filters out
// labels that have no active non-default value, compacts surviving label IDs to
// 1..L, and remaps active inference vectors from temporary label positions into
// that final compact ID space.
func CompactRegistryAndRemapInferences(
	topic types.Topic,
	nonce types.BlockHeight,
	tempRegistry types.EpochLabelRegistry,
	activeInferences []*types.Inference,
	maxLabelBytes uint64,
) (types.EpochLabelRegistry, *types.Inferences, error) {
	if err := validateEpochLabelRegistry(
		topic.Id,
		topic.LabelCaseSensitive,
		nonce,
		tempRegistry,
		maxLabelBytes,
	); err != nil {
		return types.EpochLabelRegistry{}, nil, err
	}
	for _, inference := range activeInferences {
		if inference == nil {
			return types.EpochLabelRegistry{}, nil, errorsmod.Wrap(sdkerrors.ErrLogic, "active inference is nil")
		}
		if err := validateActiveInferenceForClose(topic, nonce, inference.Inferer, *inference); err != nil {
			return types.EpochLabelRegistry{}, nil, err
		}
	}
	sortedActive := append([]*types.Inference(nil), activeInferences...)
	sort.Slice(sortedActive, func(i, j int) bool { return sortedActive[i].Inferer < sortedActive[j].Inferer })

	switch topic.OutputArity {
	case types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE:
		registry := types.EpochLabelRegistry{
			TopicId: topic.Id,
			//nolint:gosec // nonce is non-negative (validated above)
			EpochId: uint64(nonce),
			Labels: []*types.TopicLabel{
				{Id: types.SingleArityCanonicalLabelID, Name: types.SingleArityCanonicalLabel},
			},
		}
		out := make([]*types.Inference, 0, len(sortedActive))
		for _, inference := range sortedActive {
			value := topic.LabelDefaultValue
			if len(inference.Values) > 0 {
				value = inference.Values[0]
			}
			out = append(out, copyInferenceWithValues(inference, []alloraMath.Dec{value}))
		}
		return registry, &types.Inferences{Inferences: out}, nil
	case types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI:
		// fall through
	default:
		return types.EpochLabelRegistry{}, nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "output_arity is invalid")
	}

	if len(tempRegistry.Labels) == 0 {
		return types.EpochLabelRegistry{}, nil, types.ErrEpochLabelRegistryEmpty
	}
	used := activeNonDefaultLabelMask(tempRegistry, sortedActive, topic.LabelDefaultValue)
	finalLabels := make([]*types.TopicLabel, 0, len(tempRegistry.Labels))
	tempToFinal := make([]int, len(tempRegistry.Labels))
	for i := range tempToFinal {
		tempToFinal[i] = -1
	}
	for tempIdx, lbl := range tempRegistry.Labels {
		if !used[tempIdx] {
			continue
		}
		//nolint:gosec // finalLabels length is bounded by tempRegistry.Labels length.
		finalID := LabelId(len(finalLabels) + 1)
		finalLabels = append(finalLabels, &types.TopicLabel{Id: finalID, Name: lbl.Name})
		tempToFinal[tempIdx] = labelSlot(finalID)
	}
	if len(finalLabels) == 0 {
		return types.EpochLabelRegistry{}, nil, types.ErrEpochLabelRegistryEmpty
	}
	finalRegistry := types.EpochLabelRegistry{
		TopicId: topic.Id,
		//nolint:gosec // nonce is non-negative (validated above)
		EpochId: uint64(nonce),
		Labels:  finalLabels,
	}
	remapped := make([]*types.Inference, 0, len(sortedActive))
	for _, inference := range sortedActive {
		values := make([]alloraMath.Dec, len(finalLabels))
		for i := range values {
			values[i] = topic.LabelDefaultValue
		}
		for tempIdx, finalIdx := range tempToFinal {
			if finalIdx < 0 {
				continue
			}
			if tempIdx < len(inference.Values) {
				values[finalIdx] = inference.Values[tempIdx]
			}
		}
		remapped = append(remapped, copyInferenceWithValues(inference, values))
	}

	return finalRegistry, &types.Inferences{Inferences: remapped}, nil
}

func activeNonDefaultLabelMask(
	tempRegistry types.EpochLabelRegistry,
	activeInferences []*types.Inference,
	labelDefaultValue alloraMath.Dec,
) []bool {
	used := make([]bool, len(tempRegistry.Labels))
	for tempIdx := range tempRegistry.Labels {
		for _, inference := range activeInferences {
			if inference == nil || tempIdx >= len(inference.Values) {
				continue
			}
			if !inference.Values[tempIdx].Equal(labelDefaultValue) {
				used[tempIdx] = true
				break
			}
		}
	}
	return used
}

func validateActiveInferenceForClose(
	topic types.Topic,
	nonce types.BlockHeight,
	inferer ActorId,
	in types.Inference,
) error {
	if inferer == "" {
		return errorsmod.Wrap(sdkerrors.ErrLogic, "active inference has empty inferer")
	}
	if in.TopicId != topic.Id {
		return errorsmod.Wrapf(
			sdkerrors.ErrLogic,
			"active inference topic mismatch for %s: got %d expected %d",
			inferer,
			in.TopicId,
			topic.Id,
		)
	}
	if in.BlockHeight != nonce {
		return errorsmod.Wrapf(
			sdkerrors.ErrLogic,
			"active inference nonce mismatch for %s: got %d expected %d",
			inferer,
			in.BlockHeight,
			nonce,
		)
	}
	if in.Inferer != inferer {
		return errorsmod.Wrapf(
			sdkerrors.ErrLogic,
			"active inference inferer mismatch: got %s expected %s",
			in.Inferer,
			inferer,
		)
	}
	return nil
}

func copyInferenceWithValues(inference *types.Inference, values []alloraMath.Dec) *types.Inference {
	copiedValues := append([]alloraMath.Dec(nil), values...)
	return &types.Inference{
		TopicId:     inference.TopicId,
		BlockHeight: inference.BlockHeight,
		Inferer:     inference.Inferer,
		Values:      copiedValues,
		ExtraData:   inference.ExtraData,
		Proof:       inference.Proof,
	}
}

// FinalizeInferencesAndRegistryAtClose loads active temporary inferences and
// returns the final compact registry with committed inferences aligned to it.
func (k *WorkerKeeper) FinalizeInferencesAndRegistryAtClose(
	ctx context.Context,
	topic types.Topic,
	nonce types.BlockHeight,
	activeInfererAddresses []ActorId,
) (*types.Inferences, types.EpochLabelRegistry, error) {
	activeInferences, err := k.LoadActiveInfererInferencesForClose(ctx, topic, nonce, activeInfererAddresses)
	if err != nil {
		return nil, types.EpochLabelRegistry{}, err
	}
	tempRegistry, err := k.topicKeeper.GetEpochLabelRegistry(ctx, topic.Id, nonce)
	if err != nil {
		return nil, types.EpochLabelRegistry{}, err
	}
	params, err := k.paramsKeeper.GetParams(ctx)
	if err != nil {
		return nil, types.EpochLabelRegistry{}, errorsmod.Wrap(err, "error getting params for epoch label registry finalization")
	}
	finalRegistry, inferences, err := CompactRegistryAndRemapInferences(
		topic,
		nonce,
		tempRegistry,
		activeInferences,
		params.MaxCanonicalLabelByteLength,
	)
	if err != nil {
		return nil, types.EpochLabelRegistry{}, err
	}
	return inferences, finalRegistry, nil
}
