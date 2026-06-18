package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

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
