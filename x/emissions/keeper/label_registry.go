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

// labelSlot maps a 1-based epoch-label id (assigned sequentially by
// RegisterEpochLabels) to its 0-based index in a dense inference Values vector.
// The id space and this mapping are owned by the epoch label registry, so this
// primitive lives here and is shared by NormalizeInputInference and close-time
// compaction.
func labelSlot(id LabelId) int { return int(id) - 1 }

// DenormalizeInferenceToInput rebuilds the input-shaped view
// of a live WSW dense inference using the open-window temporary registry passed
// by the caller. "Temporary" is a lifecycle contract: before CloseWorkerNonce,
// MULTI inference values are aligned to first-seen labels in this registry; the
// final compact registry is finalized later from active non-default labels.
//
// For MULTI topics, only the first len(inference.Values) temporary labels are
// projected back into InputLabeledValue entries. The temporary registry may have
// grown after the worker submitted, so later labels are intentionally ignored.
func DenormalizeInferenceToInput(
	topic types.Topic,
	epochLabelRegistry types.EpochLabelRegistry,
	inference types.Inference,
	maxLabelBytes uint64,
) (*types.InputInference, error) {
	if err := inference.Validate(); err != nil {
		return nil, errorsmod.Wrap(err, "inference validation failed")
	}
	if inference.TopicId != topic.Id {
		return nil, errorsmod.Wrapf(
			sdkerrors.ErrLogic,
			"inference topic mismatch: got %d expected %d",
			inference.TopicId,
			topic.Id,
		)
	}

	nonce := inference.BlockHeight
	switch topic.OutputArity {
	case types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE:
		return denormalizeSingleInferenceToInput(inference)
	case types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI:
		return denormalizeMultiInferenceToInput(topic, nonce, epochLabelRegistry, inference, maxLabelBytes)
	default:
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "output_arity is invalid")
	}
}

func denormalizeSingleInferenceToInput(inference types.Inference) (*types.InputInference, error) {
	if len(inference.Values) > 1 {
		return nil, errorsmod.Wrapf(
			sdkerrors.ErrLogic,
			"single-arity inference has %d values, expected at most 1",
			len(inference.Values),
		)
	}
	if len(inference.Values) == 0 {
		return nil, errorsmod.Wrapf(
			sdkerrors.ErrLogic,
			"single-arity inference must contain inferences, has none",
		)
	}
	boundedValue, err := alloraMath.NewBoundedExp40Dec(inference.Values[0])
	if err != nil {
		return nil, errorsmod.Wrap(err, "single-arity inference value is out of bounded range")
	}
	return &types.InputInference{
		TopicId:     inference.TopicId,
		BlockHeight: inference.BlockHeight,
		Inferer:     inference.Inferer,
		Value:       boundedValue,
		Values:      nil,
		ExtraData:   inference.ExtraData,
		Proof:       inference.Proof,
	}, nil
}

func denormalizeMultiInferenceToInput(
	topic types.Topic,
	nonce types.BlockHeight,
	labelRegistry types.EpochLabelRegistry,
	inference types.Inference,
	maxLabelBytes uint64,
) (*types.InputInference, error) {
	if err := validateEpochLabelRegistry(
		topic.Id,
		topic.LabelCaseSensitive,
		nonce,
		labelRegistry,
		maxLabelBytes,
	); err != nil {
		return nil, err
	}
	if len(inference.Values) > len(labelRegistry.Labels) {
		return nil, errorsmod.Wrapf(
			sdkerrors.ErrLogic,
			"inference has %d values but temporary registry has %d labels",
			len(inference.Values),
			len(labelRegistry.Labels),
		)
	}

	values := make([]*types.InputLabeledValue, 0, len(inference.Values))
	for i, value := range inference.Values {
		boundedValue, err := alloraMath.NewBoundedExp40Dec(value)
		if err != nil {
			return nil, errorsmod.Wrapf(err, "multi-arity inference value at index %d is out of bounded range", i)
		}
		values = append(values, &types.InputLabeledValue{
			Label: labelRegistry.Labels[i].Name,
			Value: boundedValue,
		})
	}
	return &types.InputInference{
		TopicId:     inference.TopicId,
		BlockHeight: inference.BlockHeight,
		Inferer:     inference.Inferer,
		Value:       alloraMath.MustNewBoundedExp40Dec(alloraMath.ZeroDec()),
		Values:      values,
		ExtraData:   inference.ExtraData,
		Proof:       inference.Proof,
	}, nil
}

// SetEpochLabelRegistry overwrites the registry for a topic epoch.
func (k *TopicKeeper) SetEpochLabelRegistry(
	ctx context.Context,
	registry types.EpochLabelRegistry,
) error {
	//nolint:gosec // EpochId was originally produced from a validated non-negative block height.
	nonce := types.BlockHeight(registry.EpochId)
	topic, err := k.GetTopic(ctx, registry.TopicId)
	if err != nil {
		return errorsmod.Wrap(err, "failed to get topic for epoch label registry validation")
	}
	params, err := k.paramsKeeper.GetParams(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "failed to get params for epoch label registry validation")
	}
	if err := validateEpochLabelRegistry(
		registry.TopicId,
		topic.LabelCaseSensitive,
		nonce,
		registry,
		params.MaxCanonicalLabelByteLength,
	); err != nil {
		return err
	}
	return k.topicLabelRegistry.Set(ctx, collections.Join(registry.TopicId, nonce), registry)
}

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

// validateEpochLabelRegistry guards the stored registry invariant for a topic epoch.
// It ensures identity, label IDs, canonical names, and uniqueness stay consistent.
func validateEpochLabelRegistry(
	topicID types.TopicId,
	labelCaseSensitive bool,
	nonce types.BlockHeight,
	registry types.EpochLabelRegistry,
	maxLabelBytes uint64,
) error {
	if err := types.ValidateTopicId(topicID); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateBlockHeight(nonce); err != nil {
		return errorsmod.Wrap(err, "nonce block height validation failed")
	}
	if registry.TopicId != topicID {
		return errorsmod.Wrapf(sdkerrors.ErrLogic, "registry topic mismatch: got %d expected %d", registry.TopicId, topicID)
	}
	if registry.EpochId != uint64(nonce) {
		return errorsmod.Wrapf(sdkerrors.ErrLogic, "registry epoch mismatch: got %d expected %d", registry.EpochId, nonce)
	}
	// NOTE: the registry size cap (Params.MaxEpochLabelRegistrySize) is enforced
	// only at the growth point (RegisterEpochLabels). It is intentionally NOT
	// re-checked here: this validation runs on already-persisted registries
	// (read/close/import), and re-checking a mutable governance param against
	// historical data would let a lowered cap retroactively invalidate valid
	// state (e.g. aborting genesis import)
	seen := make(map[string]struct{}, len(registry.Labels))
	for i, lbl := range registry.Labels {
		if lbl == nil {
			return errorsmod.Wrapf(sdkerrors.ErrLogic, "registry label at index %d is nil", i)
		}
		//nolint:gosec // registry length is bounded at registration by Params.MaxEpochLabelRegistrySize.
		expectedID := LabelId(i + 1)
		if lbl.Id != expectedID {
			return errorsmod.Wrapf(sdkerrors.ErrLogic, "registry label %q has id %d expected %d", lbl.Name, lbl.Id, expectedID)
		}
		if err := types.EnsureCanonicalLabelName(lbl.Name, maxLabelBytes, labelCaseSensitive); err != nil {
			return errorsmod.Wrapf(err, "registry label at index %d is invalid", i)
		}
		if _, ok := seen[lbl.Name]; ok {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "registry label at index %d is duplicated: %q", i, lbl.Name)
		}
		seen[lbl.Name] = struct{}{}
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
