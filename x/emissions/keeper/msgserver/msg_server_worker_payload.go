package msgserver

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	actorutils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Output a new set of inferences where only 1 inference per registerd inferer is kept,
// ignore the rest. In particular, take the first inference from each registered inferer
// and none from any unregistered inferer.
// Signatures, anti-synil procedures, and "skimming of only the top few workers by score
// descending" should be done here.
func verifyAndInsertInferencesFromTopInferers(
	ctx sdk.Context,
	ms msgServer,
	blockHeight int64,
	topicId uint64,
	maxTopWorkersToReward uint64,
	topicQuantile alloraMath.Dec,
	alphaRegret alloraMath.Dec,
	nonce types.Nonce,
	workerDataBundles []*types.WorkerDataBundle,
) (map[string]bool, error) {
	inferencesByInferer := make(map[string]*types.Inference)
	infererScoreEmas := make(map[string]types.Score)
	errors := make(map[string]string)
	if len(workerDataBundles) == 0 {
		return nil, types.ErrNoValidBundles
	}
	for _, workerDataBundle := range workerDataBundles {
		/// Do filters first, then consider the inferenes for inclusion
		/// Do filters on the per payload first, then on each inferer
		/// All filters should be done in order of increasing computational complexity

		if err := workerDataBundle.Validate(); err != nil {
			errors[workerDataBundle.Worker] = "Validate: Invalid worker data bundle"
			continue // Ignore only invalid worker data bundles
		}
		/// If we do PoX-like anti-sybil procedure, would go here

		inference := workerDataBundle.InferenceForecastsBundle.Inference

		// Check if the topic and nonce are correct
		if inference.TopicId != topicId ||
			inference.BlockHeight != nonce.BlockHeight {
			errors[workerDataBundle.Worker] = "Worker data bundle does not match topic or nonce"
			continue
		}

		/// Now do filters on each inferer
		// Ensure that we only have one inference per inferer. If not, we just take the first one
		if _, ok := inferencesByInferer[inference.Inferer]; !ok {
			// Check if the inferer is registered
			isInfererRegistered, err := ms.k.IsWorkerRegisteredInTopic(ctx, topicId, inference.Inferer)
			if err != nil {
				errors[workerDataBundle.Worker] = "Err to check if worker is registered in topic"
				continue
			}
			if !isInfererRegistered {
				errors[workerDataBundle.Worker] = "Inferer is not registered"
				continue
			}

			// Get the latest score for each inferer => only take top few by score descending
			latestScore, err := ms.k.GetInfererScoreEma(ctx, topicId, inference.Inferer)
			if err != nil {
				errors[workerDataBundle.Worker] = "Latest score not found"
				continue
			}
			/// Filtering done now, now write what we must for inclusion
			infererScoreEmas[inference.Inferer] = latestScore
			inferencesByInferer[inference.Inferer] = inference
		}
	}

	/// If we pseudo-random sample from the non-sybil set of reputers, we would do it here
	topInferers, allInferersSorted := FindTopNByScoreDesc(maxTopWorkersToReward, infererScoreEmas, nonce.BlockHeight)
	// There is an edge case when all reputers are random.
	// Technically we should sort by stake with pseudo-random tiebreaker, however this adds unnecessary complexity
	// given how rare this possibility is. Futhermore, score ultimately may matter more than stake.

	// Build list of inferences that pass all filters
	// AND are from top performing inferers among those who have submitted inferences in this batch
	inferencesFromTopInferers := make([]*types.Inference, 0)
	acceptedInferers := make(map[string]bool, 0)
	for _, worker := range topInferers {
		acceptedInferers[worker] = true
		inferencesFromTopInferers = append(inferencesFromTopInferers, inferencesByInferer[worker])
	}

	if len(inferencesFromTopInferers) == 0 {
		return nil, types.ErrNoValidBundles
	}

	err := ms.UpdateScoresOfPassiveActorsWithActiveQuantile(
		ctx,
		blockHeight,
		maxTopWorkersToReward,
		topicId,
		alphaRegret,
		topicQuantile,
		infererScoreEmas,
		topInferers,
		allInferersSorted,
		acceptedInferers,
		types.ActorType_INFERER,
	)

	// Ensure deterministic ordering of inferences
	sort.Slice(inferencesFromTopInferers, func(i, j int) bool {
		return inferencesFromTopInferers[i].Inferer < inferencesFromTopInferers[j].Inferer
	})

	// Store the final list of inferences
	inferencesToInsert := types.Inferences{
		Inferences: inferencesFromTopInferers,
	}
	err = ms.k.InsertInferences(ctx, topicId, nonce, inferencesToInsert)
	if err != nil {
		return nil, err
	}

	return acceptedInferers, nil
}

// Output a new set of forecasts where only 1 forecast per registerd forecaster is kept,
// ignore the rest. In particular, take the first forecast from each registered forecaster
// and none from any unregistered forecaster.
// Signatures, anti-synil procedures, and "skimming of only the top few workers by score
// descending" should be done here.
func verifyAndInsertForecastsFromTopForecasters(
	ctx sdk.Context,
	ms msgServer,
	blockHeight int64,
	topicId uint64,
	maxTopWorkersToReward uint64,
	topicQuantile alloraMath.Dec,
	alphaRegret alloraMath.Dec,
	nonce types.Nonce,
	workerDataBundle []*types.WorkerDataBundle,
	// Inferers in the current batch, assumed to have passed VerifyAndInsertInferencesFromTopInferers() filters
	acceptedInferersOfBatch map[string]bool,
) error {
	forecastsByForecaster := make(map[string]*types.Forecast)
	forecasterScoreEmas := make(map[string]types.Score)
	for _, workerDataBundle := range workerDataBundle {
		/// Do filters first, then consider the inferenes for inclusion
		/// Do filters on the per payload first, then on each forecaster
		/// All filters should be done in order of increasing computational complexity

		if err := workerDataBundle.Validate(); err != nil {
			continue // Ignore only invalid worker data bundles
		}

		/// If we do PoX-like anti-sybil procedure, would go here

		forecast := workerDataBundle.InferenceForecastsBundle.Forecast
		// Check that the forecast exist, is for the correct topic, and is for the correct nonce
		if forecast == nil ||
			forecast.TopicId != topicId ||
			forecast.BlockHeight != nonce.BlockHeight {
			continue
		}

		/// Now do filters on each forecaster
		// Ensure that we only have one forecast per forecaster. If not, we just take the first one
		if _, ok := forecastsByForecaster[forecast.Forecaster]; !ok {
			// Check if the forecaster is registered
			isForecasterRegistered, err := ms.k.IsWorkerRegisteredInTopic(ctx, topicId, forecast.Forecaster)
			if err != nil {
				continue
			}
			if !isForecasterRegistered {
				continue
			}

			// Examine forecast elements to verify that they're for inferers in the current set.
			// We assume that set of inferers has been verified above.
			// We keep what we can, ignoring the forecaster and their contribution (forecast) entirely
			// if they're left with no valid forecast elements.
			acceptedForecastElements := make([]*types.ForecastElement, 0)
			for _, el := range forecast.ForecastElements {
				if _, ok := acceptedInferersOfBatch[el.Inferer]; ok {
					acceptedForecastElements = append(acceptedForecastElements, el)
				}
			}

			// Discard if empty
			if len(acceptedForecastElements) == 0 {
				continue
			}

			/// Filtering done now, now write what we must for inclusion

			// Get the latest score for each forecaster => only take top few by score descending
			latestScoreEma, err := ms.k.GetForecasterScoreEma(ctx, topicId, forecast.Forecaster)
			if err != nil {
				continue
			}
			forecasterScoreEmas[forecast.Forecaster] = latestScoreEma
			forecastsByForecaster[forecast.Forecaster] = forecast
		}
	}

	/// If we pseudo-random sample from the non-sybil set of reputers, we would do it here
	topForecasters, allForecastersSorted := FindTopNByScoreDesc(maxTopWorkersToReward, forecasterScoreEmas, nonce.BlockHeight)

	// Build list of forecasts that pass all filters
	// AND are from top performing forecasters among those who have submitted forecasts in this batch
	forecastsFromTopForecasters := make([]*types.Forecast, 0)
	forecasterToIsTop := make(map[string]bool, 0)
	for _, worker := range topForecasters {
		forecastsFromTopForecasters = append(forecastsFromTopForecasters, forecastsByForecaster[worker])
		forecasterToIsTop[worker] = true
	}

	err := ms.UpdateScoresOfPassiveActorsWithActiveQuantile(
		ctx,
		blockHeight,
		maxTopWorkersToReward,
		topicId,
		alphaRegret,
		topicQuantile,
		forecasterScoreEmas,
		topForecasters,
		allForecastersSorted,
		forecasterToIsTop,
		types.ActorType_FORECASTER,
	)

	// Though less than ideal because it produces less-acurate network inferences,
	// it is fine if no forecasts are accepted
	// => no need to check len(forecastsFromTopForecasters) == 0

	// Ensure deterministic ordering
	sort.Slice(forecastsFromTopForecasters, func(i, j int) bool {
		return forecastsFromTopForecasters[i].Forecaster < forecastsFromTopForecasters[j].Forecaster
	})
	// Store the final list of forecasts
	forecastsToInsert := types.Forecasts{
		Forecasts: forecastsFromTopForecasters,
	}
	err = ms.k.InsertForecasts(ctx, topicId, nonce, forecastsToInsert)
	if err != nil {
		return err
	}

	return nil
}

// A tx function that accepts a list of forecasts and possibly returns an error
// Need to call this once per forecaster per topic inference solicitation round because protobuf does not nested repeated fields
func (ms msgServer) InsertWorkerPayload(ctx context.Context, msg *types.MsgInsertWorkerPayload) (*types.MsgInsertWorkerPayloadResponse, error) {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	err = checkInputLength(ctx, ms, msg)
	if err != nil {
		return nil, err
	}

	blockHeight := sdk.UnwrapSDKContext(ctx).BlockHeight()

	if err := msg.WorkerDataBundle.Validate(); err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidWorkerData,
			"Worker invalid data for block: %d", blockHeight)
	}

	// Check if the topic exists and get its parameters
	topic, err := ms.k.GetTopic(ctx, msg.TopicId)
	if err != nil {
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

	// Check if the window time is open
	if blockHeight < nonce.BlockHeight ||
		blockHeight > nonce.BlockHeight+topic.WorkerSubmissionWindow {
		return nil, errorsmod.Wrapf(
			types.ErrWorkerNonceWindowNotAvailable,
			"Worker window not open for topic: %d, current block %d , nonce block height: %d , start window: %d, end window: %d",
			topicId, blockHeight, nonce.BlockHeight, nonce.BlockHeight+topic.WorkerSubmissionWindow, nonce.BlockHeight+topic.GroundTruthLag,
		)
	}
	moduleParams, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := sdkCtx.BlockHeight()

	acceptedInferers, err := verifyAndInsertInferencesFromTopInferers(
		sdkCtx,
		ms,
		blockHeight,
		msg.TopicId,
		moduleParams.MaxTopInferersToReward,
		topic.ActiveInfererQuantile,
		topic.AlphaRegret,
		*msg.Nonce,
		msg.WorkerDataBundles,
	)
	if err != nil {
		return nil, err
	}

	err = verifyAndInsertForecastsFromTopForecasters(
		sdkCtx,
		ms,
		blockHeight,
		msg.TopicId,
		moduleParams.MaxTopForecastersToReward,
		topic.ActiveForecasterQuantile,
		topic.AlphaRegret,
		*msg.Nonce,
		msg.WorkerDataBundles,
		acceptedInferers,
	)
	if err != nil {
		return nil, err
	}
	// Update the unfulfilled worker nonce
	_, err = ms.k.FulfillWorkerNonce(ctx, msg.TopicId, msg.Nonce)
	if err != nil {
		return nil, err
	}

	workerNonce := &types.Nonce{
		BlockHeight: msg.Nonce.BlockHeight - topic.EpochLength,
	}
	sdkCtx.Logger().Debug(fmt.Sprintf("InsertBulkWorkerPayload workerNonce %d", workerNonce.BlockHeight))

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
		err = ms.k.AppendInference(ctx, topicId, *nonce, inference)
		if err != nil {
			return nil, errorsmod.Wrapf(err, "Error appending inference")
		}
	err = ms.k.SetTopicLastCommit(ctx, topic.Id, blockHeight, msg.Nonce, msg.Sender, types.ActorType_INFERER)
	if err != nil {
		return nil, err
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

		latestScoresForForecastedInferers := make(map[string]types.Score)
		// Remove duplicate forecast element
		acceptedForecastElements := make([]*types.ForecastElement, 0)
		seenInferers := make(map[string]bool)

		for _, el := range forecast.ForecastElements {
			score, err := ms.k.GetLatestInfererScore(ctx, forecast.TopicId, el.Inferer)
			if err != nil {
				continue
			}
			latestScoresForForecastedInferers[el.Inferer] = score
		}

		moduleParams, err := ms.k.GetParams(ctx)
		if err != nil {
			return nil, err
		}
		_, topNInferer := actorutils.FindTopNByScoreDesc(moduleParams.MaxElementsPerForecast, latestScoresForForecastedInferers, forecast.BlockHeight)

		for _, el := range forecast.ForecastElements {
			if !seenInferers[el.Inferer] && topNInferer[el.Inferer] {
				acceptedForecastElements = append(acceptedForecastElements, el)
				seenInferers[el.Inferer] = true
			}
		}
		forecast.ForecastElements = acceptedForecastElements
		err = ms.k.AppendForecast(ctx, topicId, *nonce, forecast)
		if err != nil {
			return nil, errorsmod.Wrapf(err,
				"Error appending forecast")
		}
	}
	return &types.MsgInsertWorkerPayloadResponse{}, nil
}
