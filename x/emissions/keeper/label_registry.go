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

// ActiveInfererInput is the staged raw submission for an inferer that is in
// the final active set at CloseWorkerNonce time.
type ActiveInfererInput struct {
	Inferer ActorId
	Input   types.InputInference
}

// BuildFinalEpochLabelRegistryFromActiveSet materializes the EpochLabelRegistry
// for (topicId, nonce) from the staged inputs of the final active inferer set.
// Labels are deduplicated, lex-sorted deterministically, assigned ids 1..L,
// and the registry is persisted.
//
// SINGLE topics short-circuit to a 1-entry registry of the canonical label
// "y" so that close-time projection is uniform across arities.
//
// Returns types.ErrEpochLabelRegistryEmpty if a MULTI topic has no labels from
// active inferers; the caller surfaces that as no-qualified-inferers (no
// registry is written in that case).
func (k *TopicKeeper) BuildFinalEpochLabelRegistryFromActiveSet(
	ctx context.Context,
	topic types.Topic,
	nonce types.BlockHeight,
	activeInputs []ActiveInfererInput,
) (types.EpochLabelRegistry, error) {
	if err := types.ValidateTopicId(topic.Id); err != nil {
		return types.EpochLabelRegistry{}, errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateBlockHeight(nonce); err != nil {
		return types.EpochLabelRegistry{}, errorsmod.Wrap(err, "nonce block height validation failed")
	}

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
		if err := k.topicLabelRegistry.Set(ctx, collections.Join(topic.Id, nonce), registry); err != nil {
			return types.EpochLabelRegistry{}, errorsmod.Wrap(err, "error setting topic label registry")
		}
		return registry, nil
	case types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI:
		// fall through
	default:
		return types.EpochLabelRegistry{}, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "output_arity is invalid")
	}

	labelsByName := make(map[string]struct{})
	for _, activeInput := range activeInputs {
		if err := validateActiveInfererInputForClose(topic, nonce, activeInput); err != nil {
			return types.EpochLabelRegistry{}, err
		}
		if len(activeInput.Input.Values) == 0 {
			return types.EpochLabelRegistry{}, errorsmod.Wrapf(
				sdkerrors.ErrLogic,
				"multi-arity active input for %s has no labeled values",
				activeInput.Inferer,
			)
		}
		for _, lv := range activeInput.Input.Values {
			if lv == nil {
				return types.EpochLabelRegistry{}, errorsmod.Wrapf(
					sdkerrors.ErrLogic,
					"multi-arity active input for %s has nil labeled value",
					activeInput.Inferer,
				)
			}
			canonicalLabel, err := types.CanonicalLabelName(lv.Label)
			if err != nil {
				return types.EpochLabelRegistry{}, errorsmod.Wrapf(
					sdkerrors.ErrLogic,
					"multi-arity active input for %s has invalid label %q",
					activeInput.Inferer,
					lv.Label,
				)
			}
			if canonicalLabel != lv.Label {
				return types.EpochLabelRegistry{}, errorsmod.Wrapf(
					sdkerrors.ErrLogic,
					"multi-arity active input for %s has non-canonical label %q",
					activeInput.Inferer,
					lv.Label,
				)
			}
			labelsByName[lv.Label] = struct{}{}
		}
	}
	if len(labelsByName) == 0 {
		return types.EpochLabelRegistry{}, types.ErrEpochLabelRegistryEmpty
	}
	labels := make([]string, 0, len(labelsByName))
	for label := range labelsByName {
		labels = append(labels, label)
	}
	sort.Strings(labels)

	entries := make([]*types.TopicLabel, len(labels))
	for i, name := range labels {
		//nolint:gosec // bounded by len(labels), from active inferers times label cap
		entries[i] = &types.TopicLabel{Id: LabelId(i + 1), Name: name}
	}
	registry := types.EpochLabelRegistry{
		TopicId: topic.Id,
		//nolint:gosec // nonce is non-negative (validated above)
		EpochId: uint64(nonce),
		Labels:  entries,
	}
	if err := k.topicLabelRegistry.Set(ctx, collections.Join(topic.Id, nonce), registry); err != nil {
		return types.EpochLabelRegistry{}, errorsmod.Wrap(err, "error setting topic label registry")
	}
	return registry, nil
}

func validateActiveInfererInputForClose(
	topic types.Topic,
	nonce types.BlockHeight,
	activeInput ActiveInfererInput,
) error {
	if activeInput.Inferer == "" {
		return errorsmod.Wrap(sdkerrors.ErrLogic, "active inferer input has empty inferer")
	}
	in := activeInput.Input
	if in.TopicId != topic.Id {
		return errorsmod.Wrapf(
			sdkerrors.ErrLogic,
			"active input topic mismatch for %s: got %d expected %d",
			activeInput.Inferer,
			in.TopicId,
			topic.Id,
		)
	}
	if in.BlockHeight != nonce {
		return errorsmod.Wrapf(
			sdkerrors.ErrLogic,
			"active input nonce mismatch for %s: got %d expected %d",
			activeInput.Inferer,
			in.BlockHeight,
			nonce,
		)
	}
	if in.Inferer != activeInput.Inferer {
		return errorsmod.Wrapf(
			sdkerrors.ErrLogic,
			"active input inferer mismatch: got %s expected %s",
			in.Inferer,
			activeInput.Inferer,
		)
	}
	return nil
}

// GetWorkersLatestInferencesByTopicIdValuesMaterializedAtClose projects each
// staged active InputInference into a committed types.Inference whose Values
// slice is aligned to the provided (already-frozen) EpochLabelRegistry.
//
// SINGLE topics produce a single-element Values slice from InputInference.Value
// (labeled value is also accepted if present and unique).
// MULTI topics scatter each labeled value into the index derived from the
// registry (ids are 1-based). Labels submitted by the worker that are not
// present in the registry are treated as errors (should be impossible if the
// caller built the registry from the same active inputs).
func (k *WorkerKeeper) GetWorkersLatestInferencesByTopicIdValuesMaterializedAtClose(
	ctx context.Context,
	topic types.Topic,
	nonce types.BlockHeight,
	activeInputs []ActiveInfererInput,
	registry types.EpochLabelRegistry,
) (*types.Inferences, error) {
	_ = ctx
	if err := types.ValidateBlockHeight(nonce); err != nil {
		return nil, errorsmod.Wrap(err, "nonce block height validation failed")
	}
	if registry.TopicId != topic.Id {
		return nil, errorsmod.Wrapf(sdkerrors.ErrLogic, "registry topic mismatch: got %d expected %d", registry.TopicId, topic.Id)
	}
	if registry.EpochId != uint64(nonce) {
		return nil, errorsmod.Wrapf(sdkerrors.ErrLogic, "registry epoch mismatch: got %d expected %d", registry.EpochId, nonce)
	}

	active := make([]*types.Inference, 0, len(activeInputs))
	switch topic.OutputArity {
	case types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE:
		for _, activeInput := range activeInputs {
			if err := validateActiveInfererInputForClose(topic, nonce, activeInput); err != nil {
				return nil, err
			}
			in := activeInput.Input
			var dec alloraMath.Dec
			switch {
			case len(in.Values) == 1:
				dec = in.Values[0].Value.ToDec()
			case len(in.Values) == 0:
				dec = in.Value.ToDec()
			default:
				return nil, errorsmod.Wrapf(sdkerrors.ErrLogic, "single-arity input inference has %d labeled values", len(in.Values))
			}
			active = append(active, &types.Inference{
				TopicId:     in.TopicId,
				BlockHeight: in.BlockHeight,
				Inferer:     in.Inferer,
				Values:      []alloraMath.Dec{dec},
				ExtraData:   in.ExtraData,
				Proof:       in.Proof,
			})
		}
	case types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI:
		L := len(registry.Labels)
		if L == 0 {
			return nil, types.ErrEpochLabelRegistryEmpty
		}
		idByLabel := make(map[string]int, L)
		for _, lbl := range registry.Labels {
			if lbl == nil {
				continue
			}
			idByLabel[lbl.Name] = int(lbl.Id) - 1
		}
		zero := alloraMath.ZeroDec()
		for _, activeInput := range activeInputs {
			if err := validateActiveInfererInputForClose(topic, nonce, activeInput); err != nil {
				return nil, err
			}
			in := activeInput.Input
			values := make([]alloraMath.Dec, L)
			for i := range values {
				values[i] = zero
			}
			for _, lv := range in.Values {
				if lv == nil {
					continue
				}
				idx, ok := idByLabel[lv.Label]
				if !ok {
					return nil, errorsmod.Wrapf(sdkerrors.ErrLogic, "label %q from worker %s not in frozen registry", lv.Label, activeInput.Inferer)
				}
				if idx < 0 || idx >= L {
					return nil, errorsmod.Wrapf(sdkerrors.ErrLogic, "label %q id out of range", lv.Label)
				}
				values[idx] = lv.Value.ToDec()
			}
			active = append(active, &types.Inference{
				TopicId:     in.TopicId,
				BlockHeight: in.BlockHeight,
				Inferer:     in.Inferer,
				Values:      values,
				ExtraData:   in.ExtraData,
				Proof:       in.Proof,
			})
		}
	default:
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "output_arity is invalid")
	}

	sort.Slice(active, func(i, j int) bool { return active[i].Inferer < active[j].Inferer })
	return &types.Inferences{Inferences: active}, nil
}
