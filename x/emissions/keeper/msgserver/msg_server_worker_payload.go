package msgserver

import (
	"context"

	"encoding/hex"

	errorsmod "cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/errors"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// A tx function that accepts a individual inference and forecast and possibly returns an error
// Need to call this once per forecaster per topic inference solicitation round because protobuf does not nested repeated fields
func (ms msgServer) InsertWorkerPayload(ctx context.Context, msg *types.MsgInsertWorkerPayload) (*types.MsgInsertWorkerPayloadResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := sdkCtx.BlockHeight()
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	err = checkInputLength(ctx, ms, msg)
	if err != nil {
		return nil, err
	}

	if err := validateWorkerDataBundle(msg.WorkerDataBundle); err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidWorkerData,
			"Worker invalid data for block: %d", blockHeight)
	}

	nonce := msg.WorkerDataBundle.Nonce
	topicId := msg.WorkerDataBundle.TopicId

	// Check if the topic exists
	topicExists, err := ms.k.TopicExists(ctx, topicId)
	if err != nil || !topicExists {
		return nil, types.ErrInvalidTopicId
	}
	// Check if the nonce is unfulfilled
	nonceUnfulfilled, err := ms.k.IsWorkerNonceUnfulfilled(ctx, topicId, nonce)
	if err != nil {
		return nil, err
	}
	// If the nonce is already fulfilled, return an error
	if !nonceUnfulfilled {
		return nil, types.ErrUnfulfilledNonceNotFound
	}

	topic, err := ms.k.GetTopic(ctx, topicId)
	if err != nil {
		return nil, types.ErrInvalidTopicId
	}

	// Check if the window time is open
	if blockHeight < nonce.BlockHeight ||
		blockHeight > nonce.BlockHeight+topic.WorkerSubmissionWindow {
		return nil, errorsmod.Wrapf(
			types.ErrWorkerNonceWindowNotAvailable,
			"Worker window not open for topic: %d, current block %d , nonce block height: %d , start window: %d, end window: %d",
			topicId, blockHeight, nonce.BlockHeight, nonce.BlockHeight+topic.WorkerSubmissionWindow, nonce.BlockHeight+topic.GroundTruthLag,
		)
	}

	// Before creating topic, transfer fee amount from creator to ecosystem bucket
	params, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "Error getting params for sender: %v", &msg.Sender)
	}
	err = sendEffectiveRevenueActivateTopicIfWeightSufficient(ctx, ms, msg.Sender, topicId, params.DataSendingFee)
	if err != nil {
		return nil, err
	}

	if msg.WorkerDataBundle.InferenceForecastsBundle.Inference != nil {
		inference := msg.WorkerDataBundle.InferenceForecastsBundle.Inference
		if inference == nil {
			return nil, errorsmod.Wrapf(err, "Inference not found")
		}
		if inference.TopicId != msg.WorkerDataBundle.TopicId {
			return nil, errorsmod.Wrapf(err,
				"Error inferer not use same topic")
		}
		isInfererRegistered, err := ms.k.IsWorkerRegisteredInTopic(ctx, topicId, inference.Inferer)
		if err != nil {
			return nil, errorsmod.Wrapf(err,
				"Error inferer address is not registered in this topic")
		}
		if !isInfererRegistered {
			return nil, errorsmod.Wrapf(err,
				"Error inferer address is not registered in this topic")
		}
		err = ms.k.UpsertInference(ctx, topicId, *nonce, inference)
		if err != nil {
			return nil, errorsmod.Wrapf(err, "Error appending inference")
		}
	}

	// Append this individual inference to all inferences
	if msg.WorkerDataBundle.InferenceForecastsBundle.Forecast != nil {
		forecast := msg.WorkerDataBundle.InferenceForecastsBundle.Forecast
		if forecast.TopicId != msg.WorkerDataBundle.TopicId {
			return nil, errorsmod.Wrapf(err,
				"Error forecaster not use same topic")
		}
		isForecasterRegistered, err := ms.k.IsWorkerRegisteredInTopic(ctx, topicId, forecast.Forecaster)
		if err != nil {
			return nil, errorsmod.Wrapf(err,
				"Error forecaster address is not registered in this topic")
		}
		if !isForecasterRegistered {
			return nil, errorsmod.Wrapf(err,
				"Error forecaster address is not registered in this topic")
		}

		// Remove duplicate forecast element
		acceptedForecastElements := make([]*types.ForecastElement, 0)
		seenInferers := make(map[string]bool)
		for _, el := range forecast.ForecastElements {
			notAlreadySeen := !seenInferers[el.Inferer]
			if notAlreadySeen {
				acceptedForecastElements = append(acceptedForecastElements, el)
				seenInferers[el.Inferer] = true
			}
		}
		forecast.ForecastElements = acceptedForecastElements
		err = ms.k.UpsertForecast(ctx, topicId, *nonce, forecast)
		if err != nil {
			return nil, errorsmod.Wrapf(err,
				"Error appending forecast")
		}
	}
	return &types.MsgInsertWorkerPayloadResponse{}, nil
}

// Validate top level then elements of the bundle
func validateWorkerDataBundle(bundle *types.WorkerDataBundle) error {
	if bundle == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "worker data bundle cannot be nil")
	}
	if bundle.Nonce == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "worker data bundle nonce cannot be nil")
	}
	if len(bundle.Worker) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "worker cannot be empty")
	}
	if len(bundle.Pubkey) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "public key cannot be empty")
	}
	if len(bundle.InferencesForecastsBundleSignature) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "signature cannot be empty")
	}
	if bundle.InferenceForecastsBundle == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "inference forecasts bundle cannot be nil")
	}

	// Validate the inference and forecast of the bundle
	if bundle.InferenceForecastsBundle.Inference == nil && bundle.InferenceForecastsBundle.Forecast == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "inference and forecast cannot both be nil")
	}
	if bundle.InferenceForecastsBundle.Inference != nil {
		if err := validateInference(bundle.InferenceForecastsBundle.Inference); err != nil {
			return err
		}
	}
	if bundle.InferenceForecastsBundle.Forecast != nil {
		if err := validateForecast(bundle.InferenceForecastsBundle.Forecast); err != nil {
			return err
		}
	}

	// Check signature from the bundle, throw if invalid!
	pk, err := hex.DecodeString(bundle.Pubkey)
	if err != nil || len(pk) != secp256k1.PubKeySize {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "signature verification failed")
	}
	pubkey := secp256k1.PubKey(pk)

	src := make([]byte, 0)
	src, _ = bundle.InferenceForecastsBundle.XXX_Marshal(src, true)
	if !pubkey.VerifySignature(src, bundle.InferencesForecastsBundleSignature) {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "signature verification failed")
	}

	return nil
}

// Validate forecast
func validateForecast(forecast *types.Forecast) error {
	if forecast == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "forecast cannot be nil")
	}
	if forecast.BlockHeight < 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "forecast block height cannot be negative")
	}
	if len(forecast.Forecaster) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "forecaster cannot be empty")
	}
	if len(forecast.ForecastElements) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "at least one forecast element must be provided")
	}
	for _, elem := range forecast.ForecastElements {
		_, err := sdk.AccAddressFromBech32(elem.Inferer)
		if err != nil {
			return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid inferer address (%s)", err)
		}
		if err := validateDec(elem.Value); err != nil {
			return err
		}
	}

	return nil
}

// Validate inference
func validateInference(inference *types.Inference) error {
	if inference == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "inference cannot be nil")
	}
	_, err := sdk.AccAddressFromBech32(inference.Inferer)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid inferer address (%s)", err)
	}
	if inference.BlockHeight < 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "inference block height cannot be negative")
	}
	if err := validateDec(inference.Value); err != nil {
		return err
	}
	return nil
}
