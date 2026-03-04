package keeper

import (
	"context"
	"sort"

	errorsmod "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

type InferenceValues struct {
	TopicId     uint64
	BlockHeight int64
	Worker      string
	Values      alloraMath.DecArray
}

func (iv InferenceValues) Len() int { return len(iv.Values) }

// ValidateAgainstRegistry verifies that the inference values are consistent
// with the provided epoch label registry.
func (iv InferenceValues) ValidateAgainstRegistry(reg types.EpochLabelRegistry) error {
	want := len(reg.GetLabels())
	if len(iv.Values) != want {
		return errorsmod.Wrapf(
			sdkerrors.ErrLogic,
			"inference values length mismatch: got=%d want=%d",
			len(iv.Values), want,
		)
	}
	for i := range iv.Values {
		if iv.Values[i].IsNaN() || !iv.Values[i].IsFinite() {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid inference value at idx=%d", i)
		}
	}
	return nil
}

// InferenceValuesFromProto converts a stored Inference proto into the internal
// InferenceValues representation used by math code.
func (k *Keeper) InferenceValuesFromProto(
	ctx context.Context,
	topic types.Topic,
	nonce BlockHeight,
	inf *types.Inference,
) (InferenceValues, error) {
	if inf == nil {
		return InferenceValues{}, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "inference is nil")
	}

	switch topic.OutputArity {
	case types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE:
		if len(inf.Values) > 1 {
			return InferenceValues{}, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "single-arity inference accepts at most one value")
		}

		var dec alloraMath.Dec
		if len(inf.Values) == 1 {
			dec = inf.Values[0]
			if !inf.Value.Equal(dec) {
				return InferenceValues{}, errorsmod.Wrap(
					sdkerrors.ErrInvalidRequest,
					"single-arity inference scalar and array mismatch",
				)
			}
		} else {
			dec = inf.Value
		}

		if dec.IsNaN() || !dec.IsFinite() {
			return InferenceValues{}, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "invalid scalar inference value")
		}

		return InferenceValues{
			TopicId:     inf.TopicId,
			BlockHeight: inf.BlockHeight,
			Worker:      inf.Inferer,
			Values:      alloraMath.DecArray{dec},
		}, nil
	case types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI:
		reg, err := k.GetEpochLabelRegistry(ctx, inf.TopicId, nonce)
		if err != nil {
			return InferenceValues{}, err
		}
		regLen := len(reg.GetLabels())
		if regLen == 0 {
			return InferenceValues{}, errorsmod.Wrap(sdkerrors.ErrLogic, "epoch label registry is empty for multi-arity")
		}
		if len(inf.Values) == 0 {
			return InferenceValues{}, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "multi-arity inference requires values")
		}
		if len(inf.Values) > regLen {
			return InferenceValues{}, errorsmod.Wrapf(
				sdkerrors.ErrLogic,
				"multi-arity inference length exceeds registry: got=%d reg=%d",
				len(inf.Values), regLen,
			)
		}

		zero := alloraMath.ZeroDec()
		out := make(alloraMath.DecArray, regLen)
		for i := range out {
			out[i] = zero
		}
		copy(out, inf.Values)

		iv := InferenceValues{
			TopicId:     inf.TopicId,
			BlockHeight: inf.BlockHeight,
			Worker:      inf.Inferer,
			Values:      out,
		}
		if err := iv.ValidateAgainstRegistry(reg); err != nil {
			return InferenceValues{}, err
		}
		return iv, nil
	default:
		return InferenceValues{}, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "output_arity is invalid")
	}
}

// ToLabeledValues converts the internal InferenceValues representation into
// a slice of LabeledValue suitable for RPC responses or event emission.
func (iv InferenceValues) ToLabeledValues(reg types.EpochLabelRegistry) ([]*types.LabeledValue, error) {
	want := len(reg.GetLabels())
	if len(iv.Values) != want {
		return nil, errorsmod.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"inference values length mismatch: got=%d want=%d",
			len(iv.Values), want,
		)
	}
	out := make([]*types.LabeledValue, 0, len(iv.Values))
	for i, v := range iv.Values {
		lbl := reg.GetLabels()[i]
		if lbl == nil {
			return nil, errorsmod.Wrapf(sdkerrors.ErrLogic, "nil label in registry at idx=%d", i)
		}
		out = append(out, &types.LabeledValue{
			LabelId:   lbl.Id,
			LabelName: lbl.Name,
			Value:     v,
		})
	}
	return out, nil
}

// InferenceValuesFromInferencesSnapshot converts a stored Inferences snapshot
// (e.g. from allInferences KV store) into a slice of internal InferenceValues.
func (k *Keeper) InferenceValuesFromInferencesSnapshot(
	ctx context.Context,
	topic types.Topic,
	nonce BlockHeight,
	inferences *types.Inferences,
) ([]InferenceValues, error) {
	if inferences == nil || len(inferences.Inferences) == 0 {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "inferences is nil/empty")
	}

	reg, err := k.GetEpochLabelRegistry(ctx, topic.Id, nonce)
	if err != nil {
		return nil, err
	}
	regLen := len(reg.GetLabels())

	out := make([]InferenceValues, 0, len(inferences.Inferences))
	for _, inf := range inferences.Inferences {
		if inf == nil {
			return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "nil inference in snapshot")
		}
		iv, err := k.InferenceValuesFromProto(ctx, topic, nonce, inf)
		if err != nil {
			return nil, err
		}
		if topic.OutputArity == types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI && len(iv.Values) != regLen {
			return nil, errorsmod.Wrapf(
				sdkerrors.ErrLogic,
				"snapshot inference not padded to registry: worker=%s got=%d want=%d",
				iv.Worker, len(iv.Values), regLen,
			)
		}
		out = append(out, iv)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Worker < out[j].Worker })
	return out, nil
}

// GetInferenceValuesAtBlock retrieves all stored inferences for a given
// (topicId, blockHeight) and converts them into internal InferenceValues.
func (k *Keeper) GetInferenceValuesAtBlock(
	ctx context.Context,
	topic types.Topic,
	nonce types.BlockHeight,
	outlierResistant bool,
) ([]InferenceValues, error) {
	infs, err := k.GetInferencesAtBlock(ctx, topic.Id, nonce, outlierResistant)
	if err != nil {
		return nil, err
	}
	return k.InferenceValuesFromInferencesSnapshot(ctx, topic, nonce, infs)
}

// GetLatestInferenceValues retrieves the most recent stored inferences
// for a topic and converts them into internal InferenceValues.
func (k *Keeper) GetLatestInferenceValues(
	ctx context.Context,
	topic types.Topic,
	outlierResistant bool,
) ([]InferenceValues, types.BlockHeight, error) {
	infs, bh, err := k.GetLatestTopicInferences(ctx, topic.Id, outlierResistant)
	if err != nil {
		return nil, 0, err
	}
	ivs, err := k.InferenceValuesFromInferencesSnapshot(ctx, topic, bh, infs)
	if err != nil {
		return nil, 0, err
	}
	return ivs, bh, nil
}
