package msgserver

import (
	"context"
	"fmt"

	"encoding/hex"

	errorsmod "cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

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
	moduleParams, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "Error getting params for sender: %v", &msg.Sender)
	}
	err = sendEffectiveRevenueActivateTopicIfWeightSufficient(ctx, ms, msg.Sender, topicId, moduleParams.DataSendingFee)
	if err != nil {
		return nil, err
	}

	// get the existing inferences at this block height already
	// we'll use this later for both inferences
	// and for filtering forecasts about inferences we're
	// currently including at this moment
	existingInferences, err := ms.k.GetInferencesAtBlock(ctx, topicId, nonce.BlockHeight)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "Error getting existing inferences at block")
	}

	// Inferences
	if msg.WorkerDataBundle.InferenceForecastsBundle.Inference != nil {
		inference := msg.WorkerDataBundle.InferenceForecastsBundle.Inference
		// inference cannot be nil
		if inference == nil {
			return nil, errorsmod.Wrapf(types.ErrNoValidInferences, "Inference not found")
		}
		// inference topic id must match bundle topic id
		if inference.TopicId != msg.WorkerDataBundle.TopicId {
			return nil, errorsmod.Wrapf(types.ErrInvalidTopicId,
				"inferer not using the same topic as bundle")
		}
		// inferer must be registered in topic
		isInfererRegistered, err := ms.k.IsWorkerRegisteredInTopic(ctx, topicId, inference.Inferer)
		if err != nil {
			return nil, errorsmod.Wrapf(err,
				"error checking if inferer address is registered in this topic")
		}
		if !isInfererRegistered {
			return nil, errorsmod.Wrapf(types.ErrAddressNotRegistered,
				"inferer address is not registered in this topic")
		}

		// if we have hit the maximum number of inferers for the topic,
		// inferer must have a score higher than the lowest score this epoch window.
		// if they do not their inference is not accepted
		// if they do, their inference is accepted, and they replace the person with the lowest score

		// if we haven't yet reached the cap of inferers, we can upsert
		if uint64(len(existingInferences.Inferences)) < moduleParams.MaxTopInferersToReward {
			err = ms.k.UpsertInference(ctx, topicId, *nonce, inference)
			if err != nil {
				return nil, errorsmod.Wrapf(err, "Error upserting inference")
			}
		} else {
			lowestInfererScore,
				indexOfLowestScoreInExistingInferences,
				err := lowestInfererScoreEma(ctx, ms.k, topicId, *existingInferences)
			if err != nil {
				return nil, errorsmod.Wrapf(err, "Error getting lowest inferer score")
			}
			infererEmaScore, err := ms.k.GetInfererScoreEma(ctx, topicId, inference.Inferer)
			if err != nil {
				return nil, errorsmod.Wrapf(err, "Error getting inferer score ema")
			}
			// if the new inference is the lowest score and it's the same as this Inferer, we allow the inferer
			// to edit their inference via upsert.
			// if the lowest score is greater than this candidate inference,
			// we ignore it. The forecast could still be accepted, so we don't throw an error here
			if lowestInfererScore.Score.Gt(infererEmaScore.Score) && lowestInfererScore.Address != inference.Inferer {
				sdkCtx.Logger().Debug(
					fmt.Sprintf(
						"Inference does not meet threshold for active set inclusion, ignoring.\n"+
							" Inferer: %s\nScore Ema %s\nThreshold %s\nTopicId: %d\nNonce: %d",
						inference.Inferer,
						infererEmaScore.Score.String(),
						lowestInfererScore.Score.String(),
						topicId,
						nonce.BlockHeight))
			} else {
				// we are kicking out the lowest scoring inferer and replacing them with the new inferer
				err = ms.k.ReplaceInference(
					ctx,
					topicId,
					*nonce,
					*existingInferences,
					indexOfLowestScoreInExistingInferences,
					*inference)
				if err != nil {
					return nil, errorsmod.Wrapf(err, "Error replacing inference")
				}
			}
		}
	}

	// Forecasts
	if msg.WorkerDataBundle.InferenceForecastsBundle.Forecast != nil {
		forecast := msg.WorkerDataBundle.InferenceForecastsBundle.Forecast
		if len(forecast.ForecastElements) == 0 {
			return nil, errorsmod.Wrapf(types.ErrNoValidForecastElements, "No valid forecast elements found in Forecast")
		}
		if forecast.TopicId != msg.WorkerDataBundle.TopicId {
			return nil, errorsmod.Wrapf(types.ErrInvalidTopicId, "forecaster not using the same topic as bundle")
		}
		isForecasterRegistered, err := ms.k.IsWorkerRegisteredInTopic(ctx, topicId, forecast.Forecaster)
		if err != nil {
			return nil, errorsmod.Wrapf(err,
				"error checking if forecaster address is registered in this topic")
		}
		if !isForecasterRegistered {
			return nil, errorsmod.Wrapf(types.ErrAddressNotRegistered,
				"forecaster address is not registered in this topic")
		}

		// Only the top forecasters can insert forecasts
		// if we have hit the maximum number of forecasters for the topic,
		// the forecaster must have a score higher than the lowest score this epoch window.
		// if they do not their forecast is not accepted
		// if they do, their forecast is accepted, and they replace the person with the lowest score

		// get the existing forecasts at this block height
		existingForecasts, err := ms.k.GetForecastsAtBlock(ctx, topicId, nonce.BlockHeight)
		if err != nil {
			return nil, errorsmod.Wrapf(err, "Error getting existing forecasts at block")
		}

		// if we have not yet hit the maximum number of forecasters for the topic,
		// we can upsert the forecast
		if uint64(len(existingForecasts.Forecasts)) < moduleParams.MaxTopForecastersToReward {
			// limit forecast elements to only those that describe the top inferers
			// note that this is the current top inferers, which could change before
			// the worker nonce is done
			// also remove duplicate forecast elements
			forecast = filterForecastElementsToTopInferers(forecast, existingInferences)

			if len(forecast.ForecastElements) > 0 {
				err = ms.k.UpsertForecast(ctx, topicId, *nonce, forecast)
				if err != nil {
					return nil, errorsmod.Wrapf(err,
						"Error upserting forecast")
				}
			}
		} else {
			// if we have hit the maximum number of forecasters for the topic,
			// we need to see if this forecaster is good enough to break into the active set
			lowestForecasterScore,
				indexOfLowestScoreInExistingForecasts,
				err := lowestForecasterScoreEma(ctx, ms.k, topicId, *existingForecasts)
			if err != nil {
				return nil, errorsmod.Wrapf(err, "Error getting lowest forecaster score")
			}
			forecasterEmaScore, err := ms.k.GetForecasterScoreEma(ctx, topicId, forecast.Forecaster)
			if err != nil {
				return nil, errorsmod.Wrapf(err, "Error getting forecaster score ema")
			}
			if lowestForecasterScore.Score.Gt(forecasterEmaScore.Score) && lowestForecasterScore.Address != forecast.Forecaster {
				sdkCtx.Logger().Debug(
					fmt.Sprintf(
						"Forecast does not meet threshold for active set inclusion, ignoring.\n"+
							" Forecaster: %s\nScore Ema %s\nThreshold %s\nTopicId: %d\nNonce: %d",
						forecast.Forecaster,
						forecasterEmaScore.Score.String(),
						lowestForecasterScore.Score.String(),
						topicId,
						nonce.BlockHeight))
			} else {
				// limit forecast elements to only those that describe the top inferers
				// note that this is the current top inferers, which could change before
				// the worker nonce is done
				// also remove duplicate forecast elements
				forecast = filterForecastElementsToTopInferers(forecast, existingInferences)

				// we are kicking out the lowest scoring forecaster and replacing them with the new forecaster
				err = ms.k.ReplaceForecast(
					ctx,
					topicId,
					*nonce,
					*existingForecasts,
					indexOfLowestScoreInExistingForecasts,
					*forecast)
				if err != nil {
					return nil, errorsmod.Wrap(err, "error replacing forecasts")
				}
			}
		}
	}
	return &types.MsgInsertWorkerPayloadResponse{}, nil
}

// Validate top level then elements of the bundle
func validateWorkerDataBundle(bundle *types.WorkerDataBundle) error {
	if bundle == nil {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "worker data bundle cannot be nil")
	}
	if bundle.Nonce == nil {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "worker data bundle nonce cannot be nil")
	}
	if len(bundle.Worker) == 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "worker cannot be empty")
	}
	if len(bundle.Pubkey) == 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "public key cannot be empty")
	}
	if len(bundle.InferencesForecastsBundleSignature) == 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "signature cannot be empty")
	}
	if bundle.InferenceForecastsBundle == nil {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "inference forecasts bundle cannot be nil")
	}

	// Validate the inference and forecast of the bundle
	if bundle.InferenceForecastsBundle.Inference == nil && bundle.InferenceForecastsBundle.Forecast == nil {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "inference and forecast cannot both be nil")
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
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "signature verification failed")
	}
	pubkey := secp256k1.PubKey(pk)

	src := make([]byte, 0)
	src, _ = bundle.InferenceForecastsBundle.XXX_Marshal(src, true)
	if !pubkey.VerifySignature(src, bundle.InferencesForecastsBundleSignature) {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "signature verification failed")
	}

	return nil
}

// Validate forecast
func validateForecast(forecast *types.Forecast) error {
	if forecast == nil {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "forecast cannot be nil")
	}
	if forecast.BlockHeight < 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "forecast block height cannot be negative")
	}
	if len(forecast.Forecaster) == 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "forecaster cannot be empty")
	}
	if len(forecast.ForecastElements) == 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "at least one forecast element must be provided")
	}
	for _, elem := range forecast.ForecastElements {
		_, err := sdk.AccAddressFromBech32(elem.Inferer)
		if err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid inferer address (%s)", err)
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
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "inference cannot be nil")
	}
	_, err := sdk.AccAddressFromBech32(inference.Inferer)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid inferer address (%s)", err)
	}
	if inference.BlockHeight < 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "inference block height cannot be negative")
	}
	if err := validateDec(inference.Value); err != nil {
		return err
	}
	return nil
}

// find the lowest score of anyone who has been accepted into this epoch thus far
// because the number of active inferers is capped at a relatively small number,
// O(n) over the full list is fine here
func lowestInfererScoreEma(
	ctx context.Context,
	k keeper.Keeper,
	topicId uint64,
	existingInferences types.Inferences,
) (lowestScore types.Score, indexInExistingInferences int, err error) {
	// we should only ever be calling this function
	// when we've found that the number of inferers is at the cap,
	// so it's impossible to have 0 existing inferences
	if len(existingInferences.Inferences) == 0 {
		return types.Score{}, 0, errorsmod.Wrapf(types.ErrNoValidInferences, "No valid inferences found while looking for EMA")
	}
	indexInExistingInferences = 0
	firstInferer := existingInferences.Inferences[indexInExistingInferences].Inferer
	lowestScore, err = k.GetInfererScoreEma(ctx, topicId, firstInferer)
	if err != nil {
		return types.Score{}, 0, errorsmod.Wrapf(err, "Error getting existing inferer score at block")
	}
	for i, existingInference := range existingInferences.Inferences {
		existingInfererScore, err := k.GetInfererScoreEma(ctx, topicId, existingInference.Inferer)
		if err != nil {
			return types.Score{}, 0, errorsmod.Wrapf(err, "Error getting existing inferer score at block")
		}
		if existingInfererScore.Score.Lt(lowestScore.Score) {
			indexInExistingInferences = i
			lowestScore = existingInfererScore
		}
	}
	return lowestScore, indexInExistingInferences, nil
}

// return the top inferers from the existing inferences.
// only top inferers can insert inferences,
// therefore the existing list contains the top inferers
func getTopInferersFromExistingInferences(existingInferences *types.Inferences) (topInferers map[string]struct{}) {
	topInferers = make(map[string]struct{})
	for _, inference := range existingInferences.Inferences {
		topInferers[inference.Inferer] = struct{}{}
	}
	return topInferers
}

// filterForecastElementsToTopInferers filters the forecast elements to only include the top inferers
// and also removes duplicate forecast elements
func filterForecastElementsToTopInferers(
	forecast *types.Forecast,
	existingInferences *types.Inferences,
) (filteredForecast *types.Forecast) {
	filteredForecast = &types.Forecast{
		TopicId:          forecast.TopicId,
		BlockHeight:      forecast.BlockHeight,
		Forecaster:       forecast.Forecaster,
		ForecastElements: make([]*types.ForecastElement, 0),
		ExtraData:        forecast.ExtraData,
	}
	topInferers := getTopInferersFromExistingInferences(existingInferences)
	acceptedForecastElements := make([]*types.ForecastElement, 0)
	seenInferers := make(map[string]struct{})
	for _, el := range forecast.ForecastElements {
		_, alreadySeen := seenInferers[el.Inferer]
		_, isTopInferer := topInferers[el.Inferer]
		if !alreadySeen && isTopInferer {
			acceptedForecastElements = append(acceptedForecastElements, el)
			seenInferers[el.Inferer] = struct{}{}
		}
	}
	filteredForecast.ForecastElements = acceptedForecastElements
	return filteredForecast
}

// find the lowest score of anyone who has been accepted into this epoch thus far
// because the number of active forecasters is capped at a relatively small number,
// O(n) over the full list is fine here
func lowestForecasterScoreEma(
	ctx context.Context,
	k keeper.Keeper,
	topicId uint64,
	existingForecasts types.Forecasts,
) (lowestScore types.Score, indexInExistingInferences int, err error) {
	// we should only ever be calling this function
	// when we've found that the number of inferers is at the cap,
	// so it's impossible to have 0 existing inferences
	if len(existingForecasts.Forecasts) == 0 {
		return types.Score{}, 0, errorsmod.Wrapf(types.ErrNoValidForecastElements, "No valid forecasts found while looking for EMA")
	}
	indexInExistingInferences = 0
	firstForecaster := existingForecasts.Forecasts[indexInExistingInferences].Forecaster
	lowestScore, err = k.GetForecasterScoreEma(ctx, topicId, firstForecaster)
	if err != nil {
		return types.Score{}, 0, errorsmod.Wrapf(err, "Error getting existing forecaster score at block")
	}
	for i, existingForecast := range existingForecasts.Forecasts {
		existingForecasterScore, err := k.GetForecasterScoreEma(ctx, topicId, existingForecast.Forecaster)
		if err != nil {
			return types.Score{}, 0, errorsmod.Wrapf(err, "Error getting existing forecaster score at block")
		}
		if existingForecasterScore.Score.Lt(lowestScore.Score) {
			indexInExistingInferences = i
			lowestScore = existingForecasterScore
		}
	}
	return lowestScore, indexInExistingInferences, nil
}
