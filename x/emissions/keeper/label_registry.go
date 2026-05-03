package keeper

import (
	"context"
	"sort"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// SetEpochLabelRegistry overwrites the registry for a topic epoch.
func (k *TopicKeeper) SetEpochLabelRegistry(
	ctx context.Context,
	registry types.EpochLabelRegistry,
) error {
	if err := types.ValidateTopicId(registry.TopicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	//nolint:gosec // EpochId was originally produced from a validated non-negative block height.
	nonce := types.BlockHeight(registry.EpochId)
	if err := types.ValidateBlockHeight(nonce); err != nil {
		return errorsmod.Wrap(err, "nonce block height validation failed")
	}
	return k.topicLabelRegistry.Set(ctx, collections.Join(registry.TopicId, nonce), registry)
}

// MaterializeFinalEpochLabelRegistry filters a temporary first-seen registry to
// active non-default labels, compacts IDs to 1..L, and returns active
// inferences aligned to that final registry.
func MaterializeFinalEpochLabelRegistry(
	topic types.Topic,
	nonce types.BlockHeight,
	tempRegistry types.EpochLabelRegistry,
	activeInferences []*types.Inference,
) (types.EpochLabelRegistry, *types.Inferences, bool, error) {
	if err := validateTemporaryRegistry(topic, nonce, tempRegistry); err != nil {
		return types.EpochLabelRegistry{}, nil, false, err
	}
	for _, inference := range activeInferences {
		if inference == nil {
			return types.EpochLabelRegistry{}, nil, false, errorsmod.Wrap(sdkerrors.ErrLogic, "active inference is nil")
		}
		if err := validateActiveInferenceForClose(topic, nonce, inference.Inferer, *inference); err != nil {
			return types.EpochLabelRegistry{}, nil, false, err
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
				{Id: 1, Name: "y"},
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
		return registry, &types.Inferences{Inferences: out}, true, nil
	case types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI:
		// fall through
	default:
		return types.EpochLabelRegistry{}, nil, false, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "output_arity is invalid")
	}

	if len(tempRegistry.Labels) == 0 {
		return types.EpochLabelRegistry{}, nil, false, types.ErrEpochLabelRegistryEmpty
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
		tempToFinal[tempIdx] = int(finalID) - 1
	}
	if len(finalLabels) == 0 {
		return types.EpochLabelRegistry{}, nil, false, types.ErrEpochLabelRegistryEmpty
	}
	finalRegistry := types.EpochLabelRegistry{
		TopicId: topic.Id,
		//nolint:gosec // nonce is non-negative (validated above)
		EpochId: uint64(nonce),
		Labels:  finalLabels,
	}
	reusedTemporary := CanReuseTemporaryRegistryAsFinal(tempRegistry, sortedActive, topic.LabelDefaultValue)

	materialized := make([]*types.Inference, 0, len(sortedActive))
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
		materialized = append(materialized, copyInferenceWithValues(inference, values))
	}

	return finalRegistry, &types.Inferences{Inferences: materialized}, reusedTemporary, nil
}

// CanReuseTemporaryRegistryAsFinal returns true when every temporary label is
// used by at least one active inference with a non-default value.
func CanReuseTemporaryRegistryAsFinal(
	tempRegistry types.EpochLabelRegistry,
	activeInferences []*types.Inference,
	labelDefaultValue alloraMath.Dec,
) bool {
	if len(tempRegistry.Labels) == 0 {
		return false
	}
	used := activeNonDefaultLabelMask(tempRegistry, activeInferences, labelDefaultValue)
	for _, isUsed := range used {
		if !isUsed {
			return false
		}
	}
	return true
}

func epochLabelRegistriesEqual(a, b types.EpochLabelRegistry) bool {
	if a.TopicId != b.TopicId || a.EpochId != b.EpochId || len(a.Labels) != len(b.Labels) {
		return false
	}
	for i := range a.Labels {
		if a.Labels[i] == nil || b.Labels[i] == nil {
			return a.Labels[i] == b.Labels[i]
		}
		if a.Labels[i].Id != b.Labels[i].Id || a.Labels[i].Name != b.Labels[i].Name {
			return false
		}
	}
	return true
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

func validateTemporaryRegistry(
	topic types.Topic,
	nonce types.BlockHeight,
	registry types.EpochLabelRegistry,
) error {
	if err := types.ValidateTopicId(topic.Id); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateBlockHeight(nonce); err != nil {
		return errorsmod.Wrap(err, "nonce block height validation failed")
	}
	if registry.TopicId != topic.Id {
		return errorsmod.Wrapf(sdkerrors.ErrLogic, "registry topic mismatch: got %d expected %d", registry.TopicId, topic.Id)
	}
	if registry.EpochId != uint64(nonce) {
		return errorsmod.Wrapf(sdkerrors.ErrLogic, "registry epoch mismatch: got %d expected %d", registry.EpochId, nonce)
	}
	for i, lbl := range registry.Labels {
		if lbl == nil {
			return errorsmod.Wrapf(sdkerrors.ErrLogic, "registry label at index %d is nil", i)
		}
		//nolint:gosec // registry length is bounded by per-submission caps and active windows.
		expectedID := LabelId(i + 1)
		if lbl.Id != expectedID {
			return errorsmod.Wrapf(sdkerrors.ErrLogic, "registry label %q has id %d expected %d", lbl.Name, lbl.Id, expectedID)
		}
	}
	return nil
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

// GetWorkersLatestInferencesByTopicIdValuesMaterializedAtClose loads active
// temporary inferences, freezes/overwrites the final registry, and returns the
// committed inferences aligned to final compact ids.
func (k *WorkerKeeper) GetWorkersLatestInferencesByTopicIdValuesMaterializedAtClose(
	ctx context.Context,
	topic types.Topic,
	nonce types.BlockHeight,
	activeInfererAddresses []ActorId,
) (*types.Inferences, types.EpochLabelRegistry, bool, error) {
	activeInferences, err := k.LoadActiveInfererInferencesForClose(ctx, topic, nonce, activeInfererAddresses)
	if err != nil {
		return nil, types.EpochLabelRegistry{}, false, err
	}
	tempRegistry, err := k.topicKeeper.GetEpochLabelRegistry(ctx, topic.Id, nonce)
	if err != nil {
		return nil, types.EpochLabelRegistry{}, false, err
	}
	finalRegistry, inferences, reusedTemporary, err := MaterializeFinalEpochLabelRegistry(topic, nonce, tempRegistry, activeInferences)
	if err != nil {
		return nil, types.EpochLabelRegistry{}, false, err
	}
	if !reusedTemporary || !epochLabelRegistriesEqual(tempRegistry, finalRegistry) {
		if err := k.topicKeeper.SetEpochLabelRegistry(ctx, finalRegistry); err != nil {
			return nil, types.EpochLabelRegistry{}, false, errorsmod.Wrap(err, "error setting final epoch label registry")
		}
	}
	return inferences, finalRegistry, reusedTemporary, nil
}
