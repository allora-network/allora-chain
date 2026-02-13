package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/utils/fn"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func NewWorkerKeeper(
	cdc codec.BinaryCodec,
	sb *collections.SchemaBuilder,
	topicKeeper *TopicKeeper,
	scoresKeeper *ScoresKeeper,
	paramsKeeper *ParamsKeeper,
	actorPenaltiesKeeper *ActorPenaltiesKeeper,
) *WorkerKeeper {
	return &WorkerKeeper{
		topicWorkers:         collections.NewKeySet(sb, types.TopicWorkersKey, "topic_workers", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey)),
		activeInferers:       collections.NewKeySet(sb, types.ActiveInferersKey, "active_inferers", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey)),
		activeForecasters:    collections.NewKeySet(sb, types.ActiveForecastersKey, "active_forecasters", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey)),
		workers:              collections.NewMap(sb, types.WorkerNodesKey, "worker_nodes", collections.StringKey, codec.CollValue[types.OffchainNode](cdc)),
		inferences:           collections.NewMap(sb, types.InferencesKey, "inferences", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.Inference](cdc)),
		forecasts:            collections.NewMap(sb, types.ForecastsKey, "forecasts", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.Forecast](cdc)),
		allInferences:        collections.NewMap(sb, types.AllInferencesKey, "inferences_all", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.Inferences](cdc)),
		allForecasts:         collections.NewMap(sb, types.AllForecastsKey, "forecasts_all", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.Forecasts](cdc)),
		topicKeeper:          topicKeeper,
		scoresKeeper:         scoresKeeper,
		paramsKeeper:         paramsKeeper,
		actorPenaltiesKeeper: actorPenaltiesKeeper,
	}
}

type WorkerKeeper struct {
	// for a topic, what is every worker node that has registered to it?
	topicWorkers collections.KeySet[collections.Pair[TopicId, ActorId]]
	// active topic inferers for a topic
	activeInferers collections.KeySet[collections.Pair[TopicId, ActorId]]
	// active topic forecasters for a topic
	activeForecasters collections.KeySet[collections.Pair[TopicId, ActorId]]
	// map of worker id to node data about that worker
	workers collections.Map[ActorId, types.OffchainNode]
	// map of (topic, worker) -> inference
	inferences collections.Map[collections.Pair[TopicId, ActorId], types.Inference]
	// map of (topic, worker) -> forecast[]
	forecasts collections.Map[collections.Pair[TopicId, ActorId], types.Forecast]
	// map of (topic, block_height) -> Inference
	allInferences collections.Map[collections.Pair[TopicId, BlockHeight], types.Inferences]
	// map of (topic, block_height) -> Forecast
	allForecasts collections.Map[collections.Pair[TopicId, BlockHeight], types.Forecasts]
	// topic keeper
	topicKeeper *TopicKeeper
	// scores keeper
	scoresKeeper *ScoresKeeper
	// params keeper
	paramsKeeper *ParamsKeeper
	// actor penalties keeper
	actorPenaltiesKeeper *ActorPenaltiesKeeper
}

// Adds a new worker to the worker tracking data structures, workers and topicWorkers
func (k *WorkerKeeper) InsertWorker(ctx context.Context, topicId TopicId, worker ActorId, workerInfo types.OffchainNode) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateBech32(worker); err != nil {
		return errorsmod.Wrap(err, "worker validation failed")
	}
	if err := workerInfo.Validate(); err != nil {
		return errorsmod.Wrap(err, "worker info validation failed")
	}
	topicKey := collections.Join(topicId, worker)
	err := k.topicWorkers.Set(ctx, topicKey)
	if err != nil {
		return errorsmod.Wrap(err, "error setting topic worker")
	}
	err = k.workers.Set(ctx, worker, workerInfo)
	if err != nil {
		return errorsmod.Wrap(err, "error setting worker")
	}
	return nil
}

// Remove a worker to the worker tracking data structures and topicWorkers
func (k *WorkerKeeper) RemoveWorker(ctx context.Context, topicId TopicId, worker ActorId) error {
	topicKey := collections.Join(topicId, worker)
	err := k.topicWorkers.Remove(ctx, topicKey)
	if err != nil {
		return errorsmod.Wrap(err, "error removing topic worker")
	}
	return nil
}

func (k *WorkerKeeper) GetWorkerInfo(ctx sdk.Context, workerKey ActorId) (types.OffchainNode, error) {
	return k.workers.Get(ctx, workerKey)
}

// UpdateWorkerOwner updates the payout owner associated with a worker node.
// Returns:
//   - (string) old owner
func (k *WorkerKeeper) UpdateWorkerOwner(ctx context.Context, worker ActorId, newOwner string) (string, error) {
	if err := types.ValidateBech32(worker); err != nil {
		return "", errorsmod.Wrap(err, "worker validation failed")
	}
	if err := types.ValidateBech32(newOwner); err != nil {
		return "", errorsmod.Wrap(err, "new owner validation failed")
	}
	nodeInfo, err := k.workers.Get(ctx, worker)

	if errors.Is(err, collections.ErrNotFound) {
		return "", errorsmod.Wrapf(types.ErrAddressNotRegistered, "worker %s", worker)
	} else if err != nil {
		return "", errorsmod.Wrap(err, "error getting worker info")
	}
	oldOwner := nodeInfo.Owner
	nodeInfo.Owner = newOwner
	if err := nodeInfo.Validate(); err != nil {
		return "", errorsmod.Wrap(err, "worker info validation failed")
	}
	if err := k.workers.Set(ctx, worker, nodeInfo); err != nil {
		return "", errorsmod.Wrap(err, "error setting worker info")
	}
	return oldOwner, nil
}

// AddActiveInferer adds an inferer to the active inferers set for a topic
func (k *WorkerKeeper) AddActiveInferer(ctx context.Context, topicId TopicId, inferer ActorId) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "invalid topic id")
	}
	if err := types.ValidateBech32(inferer); err != nil {
		return errorsmod.Wrap(err, "invalid inferer address")
	}
	key := collections.Join(topicId, inferer)
	return k.activeInferers.Set(ctx, key)
}

// IsActiveInferer checks if an inferer is in the active inferers set for a topic
func (k *WorkerKeeper) IsActiveInferer(ctx context.Context, topicId TopicId, inferer ActorId) (bool, error) {
	key := collections.Join(topicId, inferer)
	return k.activeInferers.Has(ctx, key)
}

// RemoveActiveInferer removes an inferer from the active inferers set for a topic
func (k *WorkerKeeper) RemoveActiveInferer(ctx context.Context, topicId TopicId, inferer ActorId) error {
	key := collections.Join(topicId, inferer)
	return k.activeInferers.Remove(ctx, key)
}

// GetActiveInferersForTopic returns all active inferers for a specific topic
func (k *WorkerKeeper) GetActiveInferersForTopic(ctx context.Context, topicId TopicId) ([]ActorId, error) {
	var inferers []ActorId
	rng := collections.NewPrefixedPairRange[TopicId, ActorId](topicId)
	err := k.activeInferers.Walk(ctx, rng, func(key collections.Pair[TopicId, ActorId]) (bool, error) {
		inferers = append(inferers, key.K2())
		return false, nil
	})
	if err != nil {
		return nil, errorsmod.Wrap(err, "error walking active inferers")
	}
	return inferers, nil
}

// AddActiveForecaster adds a forecaster to the active forecasters set for a topic
func (k *WorkerKeeper) AddActiveForecaster(ctx context.Context, topicId TopicId, forecaster ActorId) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "invalid topic id")
	}
	if err := types.ValidateBech32(forecaster); err != nil {
		return errorsmod.Wrap(err, "invalid forecaster address")
	}
	key := collections.Join(topicId, forecaster)
	return k.activeForecasters.Set(ctx, key)
}

// IsActiveForecaster checks if a forecaster is in the active forecasters set for a topic
func (k *WorkerKeeper) IsActiveForecaster(ctx context.Context, topicId TopicId, forecaster ActorId) (bool, error) {
	key := collections.Join(topicId, forecaster)
	return k.activeForecasters.Has(ctx, key)
}

// RemoveActiveForecaster removes a forecaster from the active forecasters set for a topic
func (k *WorkerKeeper) RemoveActiveForecaster(ctx context.Context, topicId TopicId, forecaster ActorId) error {
	key := collections.Join(topicId, forecaster)
	return k.activeForecasters.Remove(ctx, key)
}

// GetActiveForecastersForTopic returns all active forecasters for a specific topic
func (k *WorkerKeeper) GetActiveForecastersForTopic(ctx context.Context, topicId TopicId) ([]ActorId, error) {
	var forecasters []ActorId
	rng := collections.NewPrefixedPairRange[TopicId, ActorId](topicId)
	err := k.activeForecasters.Walk(ctx, rng, func(key collections.Pair[TopicId, ActorId]) (bool, error) {
		forecasters = append(forecasters, key.K2())
		return false, nil
	})
	if err != nil {
		return nil, errorsmod.Wrap(err, "error walking active forecasters")
	}
	return forecasters, nil
}

// ResetActiveWorkersForTopic resets the active workers for a topic
func (k *WorkerKeeper) ResetActiveWorkersForTopic(ctx context.Context, topicId TopicId) error {
	infererRange := collections.NewPrefixedPairRange[TopicId, ActorId](topicId)
	if err := k.activeInferers.Clear(ctx, infererRange); err != nil {
		return errorsmod.Wrap(err, "error clearing active inferers")
	}
	forecasterRange := collections.NewPrefixedPairRange[TopicId, ActorId](topicId)
	if err := k.activeForecasters.Clear(ctx, forecasterRange); err != nil {
		return errorsmod.Wrap(err, "error clearing active forecasters")
	}

	return nil
}

// ResetWorkersIndividualSubmissionsForTopic resets the inferer individual submissions for a topic
func (k *WorkerKeeper) ResetWorkersIndividualSubmissionsForTopic(ctx context.Context, topicId TopicId) error {
	infererRange := collections.NewPrefixedPairRange[TopicId, ActorId](topicId)
	if err := k.inferences.Clear(ctx, infererRange); err != nil {
		return errorsmod.Wrap(err, "error clearing inferences")
	}
	forecasterRange := collections.NewPrefixedPairRange[TopicId, ActorId](topicId)
	if err := k.forecasts.Clear(ctx, forecasterRange); err != nil {
		return errorsmod.Wrap(err, "error clearing forecasts")
	}

	return nil
}

func (k *WorkerKeeper) GetInferencesAtBlock(ctx context.Context, topicId TopicId, block BlockHeight, outlierResistant bool) (*types.Inferences, error) {
	key := collections.Join(topicId, block)
	inferences, err := k.allInferences.Get(ctx, key)
	if errors.Is(err, collections.ErrNotFound) {
		return &types.Inferences{Inferences: []*types.Inference{}}, nil
	} else if err != nil {
		return nil, errorsmod.Wrap(err, "error getting inferences at block")
	}

	if outlierResistant {
		filteredInferences, err := k.topicKeeper.FilterOutlierResistantInferences(ctx, topicId, inferences)
		if err != nil {
			return nil, errorsmod.Wrap(err, "error filtering outlier resistant inferences")
		}
		return &filteredInferences, nil
	}
	return &inferences, nil
}

// GetLatestTopicInferences retrieves the latest topic inferences and its block height.
func (k *WorkerKeeper) GetLatestTopicInferences(ctx context.Context, topicId TopicId, outlierResistant bool) (*types.Inferences, BlockHeight, error) {
	rng := collections.NewPrefixedPairRange[TopicId, BlockHeight](topicId).Descending()

	iter, err := k.allInferences.Iterate(ctx, rng)
	if err != nil {
		return nil, 0, errorsmod.Wrap(err, "error iterating over inferences")
	}
	defer iter.Close()

	inferences := &types.Inferences{
		Inferences: make([]*types.Inference, 0),
	}
	var blockHeight int64 = 0

	if iter.Valid() {
		keyValue, err := iter.KeyValue()
		if err != nil {
			return nil, 0, errorsmod.Wrap(err, "error getting key value")
		}
		inferences = &keyValue.Value
		blockHeight = keyValue.Key.K2()

		if outlierResistant {
			filteredInferences, err := k.topicKeeper.FilterOutlierResistantInferences(ctx, topicId, *inferences)
			if err != nil {
				return nil, 0, errorsmod.Wrap(err, "error filtering outlier resistant inferences")
			}
			inferences = &filteredInferences
		}
	}

	return inferences, blockHeight, nil
}

// UpdateNetworkInferencesOutlierMetrics recalculates the network inferences outlier metrics
// (MAD and median) for a given topic and with inferences from the given block height
func (k *WorkerKeeper) UpdateNetworkInferencesOutlierMetrics(
	ctx sdk.Context,
	topicId TopicId,
	inferenceBlockHeight BlockHeight,
) error {
	ctx.Logger().Debug("Updating network inferences outlier metrics", "topicId", topicId, "blockHeight", inferenceBlockHeight)
	// Get all inferences at the block height
	inferences, err := k.GetInferencesAtBlock(ctx, topicId, inferenceBlockHeight, false)
	if err != nil {
		return errorsmod.Wrap(err, "while getting inferences")
	}
	if len(inferences.Inferences) == 0 {
		// If there are no inferences, do not update the metrics
		ctx.Logger().Info("no inferences found, skipping update of outlier metrics", "topicId", topicId, "blockHeight", inferenceBlockHeight)
		return nil
	}

	// Create an array of the values
	values := fn.Map(inferences.Inferences[:], func(inf *types.Inference) alloraMath.Dec { return inf.Value })

	// Calculate MAD (median absolute deviation)
	mad, median, err := alloraMath.MedianAbsoluteDeviation(values)
	if err != nil {
		return errorsmod.Wrap(err, "while calculating MAD")
	}
	// Validate mad and median
	if err := types.ValidateDec(mad); err != nil {
		return errorsmod.Wrap(err, "mad is not valid")
	}
	if err := types.ValidateDec(median); err != nil {
		return errorsmod.Wrap(err, "median is not valid")
	}

	var newMad alloraMath.Dec
	// Get current mad
	previousMad, err := k.topicKeeper.GetMadInferences(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "error getting last mad")
	}
	if previousMad.IsZero() {
		// if zero, set to current mad
		newMad = mad
	} else {
		// Get alpha from params
		params, err := k.paramsKeeper.GetParams(ctx)
		if err != nil {
			return errorsmod.Wrap(err, "error getting params")
		}
		alpha := params.InferenceOutlierDetectionAlpha

		// Calculate EMA of MAD
		newMad, err = alloraMath.CalcEma(alpha, mad, previousMad, false)
		if err != nil {
			return errorsmod.Wrap(err, "error calculating ema of mad")
		}
	}

	ctx.Logger().Info("Setting new outlier-resistant mad", "newMad", newMad, "median", median, "topicId", topicId)

	// Set last mad inferences
	err = k.topicKeeper.SetMadInferences(ctx, topicId, newMad)
	if err != nil {
		return errorsmod.Wrap(err, "error setting last mad inferences")
	}

	// Set last median inferences
	err = k.topicKeeper.SetLastMedianInferences(ctx, topicId, median)
	if err != nil {
		return errorsmod.Wrap(err, "error setting last median inferences")
	}

	return nil
}

func (k *WorkerKeeper) GetForecastsAtBlock(ctx context.Context, topicId TopicId, block BlockHeight) (*types.Forecasts, error) {
	key := collections.Join(topicId, block)
	forecasts, err := k.allForecasts.Get(ctx, key)
	if errors.Is(err, collections.ErrNotFound) {
		return &types.Forecasts{Forecasts: []*types.Forecast{}}, nil
	} else if err != nil {
		return nil, errorsmod.Wrap(err, "error getting forecasts at block")
	}
	return &forecasts, nil
}

// Append individual inference for a topic/block
func (k *WorkerKeeper) AppendInference(
	ctx sdk.Context,
	topic types.Topic,
	nonceBlockHeight BlockHeight,
	inference *types.Inference,
	maxTopInferersToReward uint64,
) error {
	// Check if the inferers already submitted the inference
	isActive, err := k.IsActiveInferer(ctx, topic.Id, inference.Inferer)
	if err != nil {
		return errorsmod.Wrap(err, "error checking if worker already submitted inference")
	} else if isActive {
		return errors.New("inference already submitted")
	}

	// Get previous EMA score for the current inferer
	previousEmaScore, err := k.scoresKeeper.GetInfererScoreEma(ctx, topic.Id, inference.Inferer)
	if err != nil {
		return errorsmod.Wrapf(err, "Error getting inferer score ema")
	}
	// Only calc and save if there's a new update
	if previousEmaScore.BlockHeight >= nonceBlockHeight {
		return types.ErrCantUpdateEmaMoreThanOncePerWindow
	}

	// Check if the inferer is new and set initial EMA score
	firstSubmission := false
	if previousEmaScore.BlockHeight == 0 {
		firstSubmission = true
		initialEmaScore, err := k.scoresKeeper.GetTopicInitialInfererEmaScore(ctx, topic.Id)
		if err != nil {
			return errorsmod.Wrap(err, "error getting topic initial ema score")
		}
		previousEmaScore = types.Score{
			TopicId:     topic.Id,
			Address:     inference.Inferer,
			BlockHeight: nonceBlockHeight,
			Score:       initialEmaScore,
		}
		err = k.scoresKeeper.SetInfererScoreEma(ctx, topic.Id, inference.Inferer, previousEmaScore)
		if err != nil {
			return errorsmod.Wrap(err, "error setting initial inferer score ema")
		}
	} else {
		// If not new: Penalise the inferer if needed
		previousEmaScore, err = k.actorPenaltiesKeeper.ApplyLivenessPenaltyToInferer(ctx, topic, nonceBlockHeight, previousEmaScore)
		if err != nil {
			return errorsmod.Wrap(err, "error trying to penalise inferer")
		}

		// Update score nonce for liveness tracking
		previousEmaScore.BlockHeight = nonceBlockHeight
		err = k.scoresKeeper.SetInfererScoreEma(ctx, topic.Id, inference.Inferer, previousEmaScore)
		if err != nil {
			return errorsmod.Wrap(err, "error setting penalised inferer score ema")
		}
	}

	// Get lowest inferer score ema for the topic
	lowestEmaScore, _, err := k.scoresKeeper.GetLowestInfererScoreEma(ctx, topic.Id)
	if err != nil {
		return errorsmod.Wrap(err, "error getting lowest inferer score ema")
	}

	// Get active inferers for topic
	workerAddresses, err := k.GetActiveInferersForTopic(ctx, topic.Id)
	if err != nil {
		return errorsmod.Wrap(err, "error getting active inferers for topic")
	}

	// If there are less than maxTopInferersToReward, add the current inferer, update the lowest inferer score ema if needed, and return
	if uint64(len(workerAddresses)) < maxTopInferersToReward {
		// Update lowest inferer score ema if needed
		if uint64(len(workerAddresses)) == 0 || lowestEmaScore.Score.Gt(previousEmaScore.Score) {
			err = k.scoresKeeper.SetLowestInfererScoreEma(ctx, topic.Id, previousEmaScore)
			if err != nil {
				return errorsmod.Wrap(err, "error setting lowest inferer score ema")
			}
		}

		err = k.AddActiveInferer(ctx, topic.Id, inference.Inferer)
		if err != nil {
			return errorsmod.Wrap(err, "error adding active inferer")
		}
		return k.InsertInference(ctx, topic.Id, *inference)
	}

	// Else ...
	// Checks if the inferer's previous EMA score is greater than the lowest EMA score
	if previousEmaScore.Score.Gt(lowestEmaScore.Score) {
		// Update EMA score for the lowest score inferer, who is not the current inferer
		err = k.scoresKeeper.CalcAndSaveInfererScoreEmaWithLastSavedTopicQuantile(
			ctx,
			topic,
			nonceBlockHeight,
			lowestEmaScore,
		)
		if err != nil {
			return errorsmod.Wrap(err, "error calculating and saving inferer score ema with last saved topic quantile")
		}

		// Check if the inferer with lowest score is active before removing it, because remove will not fail if the inferer is not active
		isActive, err := k.IsActiveInferer(ctx, topic.Id, lowestEmaScore.Address)
		if err != nil {
			return errorsmod.Wrap(err, "error checking if inferer is active")
		}
		if !isActive {
			return errors.New("inferer with lowest score is not active")
		}

		// Remove inferer with lowest score
		err = k.RemoveActiveInferer(ctx, topic.Id, lowestEmaScore.Address)
		if err != nil {
			return errorsmod.Wrap(err, "error removing active inferer")
		}
		// Remove inference from inferer with lowest score
		err = k.RemoveInference(ctx, topic.Id, lowestEmaScore.Address)
		if err != nil {
			return errorsmod.Wrap(err, "error removing inference from inferer")
		}
		// Add new active inferer
		err = k.AddActiveInferer(ctx, topic.Id, inference.Inferer)
		if err != nil {
			return errorsmod.Wrap(err, "error adding active inferer")
		}
		// Calculate new lowest score with updated infererAddresses
		err = k.scoresKeeper.UpdateLowestScoreFromInfererAddresses(ctx, topic.Id, workerAddresses, inference.Inferer, lowestEmaScore.Address)
		if err != nil {
			return errorsmod.Wrap(err, "error getting low score from all inferences")
		}
		return k.InsertInference(ctx, topic.Id, *inference)
	} else {
		// Update EMA score for the current inferer, who is the lowest score inferer
		if !firstSubmission { // Only update if not a new inferer
			err = k.scoresKeeper.CalcAndSaveInfererScoreEmaWithLastSavedTopicQuantile(ctx, topic, nonceBlockHeight, previousEmaScore)
			if err != nil {
				return errorsmod.Wrap(err, "error calculating and saving inferer score ema with last saved topic quantile")
			}
		}
	}
	return nil
}

// Insert an inference for a specific topic
func (k *WorkerKeeper) InsertInference(
	ctx context.Context,
	topicId TopicId,
	inference types.Inference,
) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	err := inference.Validate()
	if err != nil {
		return errorsmod.Wrap(err, "inference is invalid")
	}
	key := collections.Join(topicId, inference.Inferer)
	return k.inferences.Set(ctx, key, inference)
}

// RemoveInference removes an inference from a inferer
func (k *WorkerKeeper) RemoveInference(
	ctx context.Context,
	topicId TopicId,
	inferer ActorId,
) error {
	key := collections.Join(topicId, inferer)
	return k.inferences.Remove(ctx, key)
}

// Insert a complete set of inferences for a topic/block.
func (k *WorkerKeeper) InsertActiveInferences(
	ctx context.Context,
	topicId TopicId,
	nonceBlockHeight BlockHeight,
	inferences types.Inferences,
) error {
	key := collections.Join(topicId, nonceBlockHeight)
	return k.allInferences.Set(ctx, key, inferences)
}

// Append individual forecast for a topic/block
func (k *WorkerKeeper) AppendForecast(
	ctx sdk.Context,
	topic types.Topic,
	nonceBlockHeight BlockHeight,
	forecast *types.Forecast,
	maxTopForecastersToReward uint64,
) error {
	// Check if the forecaster already submitted the forecast
	isActive, err := k.IsActiveForecaster(ctx, topic.Id, forecast.Forecaster)
	if err != nil {
		return errorsmod.Wrap(err, "error checking if forecaster already submitted forecast")
	} else if isActive {
		return errors.New("forecast already submitted")
	}

	previousEmaScore, err := k.scoresKeeper.GetForecasterScoreEma(ctx, topic.Id, forecast.Forecaster)
	if err != nil {
		return errorsmod.Wrapf(err, "Error getting forecaster score ema")
	}
	// Only calc and save if there's a new update
	if previousEmaScore.BlockHeight >= nonceBlockHeight {
		return types.ErrCantUpdateEmaMoreThanOncePerWindow
	}

	// Check if the forecaster is new and set initial EMA score
	firstSubmission := false
	if previousEmaScore.BlockHeight == 0 {
		firstSubmission = true
		initialEmaScore, err := k.scoresKeeper.GetTopicInitialForecasterEmaScore(ctx, topic.Id)
		if err != nil {
			return errorsmod.Wrap(err, "error getting topic initial ema score")
		}
		err = k.scoresKeeper.SetForecasterScoreEma(ctx, topic.Id, forecast.Forecaster, types.Score{
			TopicId:     topic.Id,
			Address:     forecast.Forecaster,
			BlockHeight: nonceBlockHeight,
			Score:       initialEmaScore,
		})
		if err != nil {
			return errorsmod.Wrap(err, "error setting forecaster score ema")
		}
	} else {
		// If not new: Penalise the forecaster if needed
		previousEmaScore, err = k.actorPenaltiesKeeper.ApplyLivenessPenaltyToForecaster(ctx, topic, nonceBlockHeight, previousEmaScore)
		if err != nil {
			return errorsmod.Wrap(err, "error trying to penalise forecaster")
		}

		// Update score nonce for liveness tracking
		previousEmaScore.BlockHeight = nonceBlockHeight
		err = k.scoresKeeper.SetForecasterScoreEma(ctx, topic.Id, previousEmaScore.Address, previousEmaScore)
		if err != nil {
			return errorsmod.Wrap(err, "error setting penalised forecaster score ema")
		}
	}

	lowestEmaScore, _, err := k.scoresKeeper.GetLowestForecasterScoreEma(ctx, topic.Id)
	if err != nil {
		return errorsmod.Wrap(err, "error getting lowest forecaster score ema")
	}

	forecasterAddresses, err := k.GetActiveForecastersForTopic(ctx, topic.Id)
	if err != nil {
		return errorsmod.Wrap(err, "error getting active forecasters for topic")
	}

	// If there are less than maxTopForecastersToReward, add the current forecaster
	if uint64(len(forecasterAddresses)) < maxTopForecastersToReward {
		if uint64(len(forecasterAddresses)) == 0 || lowestEmaScore.Score.Gt(previousEmaScore.Score) {
			err = k.scoresKeeper.SetLowestForecasterScoreEma(ctx, topic.Id, previousEmaScore)
			if err != nil {
				return errorsmod.Wrap(err, "error setting lowest forecaster score ema")
			}
		}

		err = k.AddActiveForecaster(ctx, topic.Id, forecast.Forecaster)
		if err != nil {
			return errorsmod.Wrap(err, "error adding active forecaster")
		}
		return k.InsertForecast(ctx, topic.Id, *forecast)
	}

	if previousEmaScore.Score.Gt(lowestEmaScore.Score) {
		// Update EMA score for the lowest score forecaster, who is not the current forecaster
		err = k.scoresKeeper.CalcAndSaveForecasterScoreEmaWithLastSavedTopicQuantile(
			ctx,
			topic,
			nonceBlockHeight,
			lowestEmaScore,
		)
		if err != nil {
			return errorsmod.Wrap(err, "error calculating and saving forecaster score ema with last saved topic quantile")
		}

		// Check if the forecaster with lowest score is active before removing it, because remove will not fail if the forecaster is not active
		isActive, err := k.IsActiveForecaster(ctx, topic.Id, lowestEmaScore.Address)
		if err != nil {
			return errorsmod.Wrap(err, "error checking if forecaster is active")
		}
		if !isActive {
			return errors.New("forecaster with lowest score is not active")
		}

		// Remove forecaster with lowest score
		err = k.RemoveActiveForecaster(ctx, topic.Id, lowestEmaScore.Address)
		if err != nil {
			return errorsmod.Wrap(err, "error removing active forecaster")
		}
		// Remove forecast from forecaster with lowest score
		err = k.RemoveForecast(ctx, topic.Id, lowestEmaScore.Address)
		if err != nil {
			return errorsmod.Wrap(err, "error removing forecast from forecaster")
		}
		// Add new active forecaster
		err = k.AddActiveForecaster(ctx, topic.Id, forecast.Forecaster)
		if err != nil {
			return errorsmod.Wrap(err, "error adding active forecaster")
		}
		// Calculate new lowest score with updated forecasterAddresses
		err = k.scoresKeeper.UpdateLowestScoreFromForecasterAddresses(ctx, topic.Id, forecasterAddresses, forecast.Forecaster, lowestEmaScore.Address)
		if err != nil {
			return errorsmod.Wrap(err, "error getting low score from all forecasts")
		}
		return k.InsertForecast(ctx, topic.Id, *forecast)
	} else {
		// Update EMA score for the current forecaster, who is the lowest score forecaster
		if !firstSubmission {
			err = k.scoresKeeper.CalcAndSaveForecasterScoreEmaWithLastSavedTopicQuantile(ctx, topic, nonceBlockHeight, previousEmaScore)
			if err != nil {
				return errorsmod.Wrap(err, "error calculating and saving forecaster score ema with last saved topic quantile")
			}
		}
	}
	return nil
}

// InsertForecast inserts a forecast for a specific topic
func (k *WorkerKeeper) InsertForecast(
	ctx context.Context,
	topicId TopicId,
	forecast types.Forecast,
) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	err := forecast.Validate()
	if err != nil {
		return errorsmod.Wrap(err, "forecast is invalid")
	}
	key := collections.Join(topicId, forecast.Forecaster)
	return k.forecasts.Set(ctx, key, forecast)
}

// RemoveForecast removes a forecast from a forecaster
func (k *WorkerKeeper) RemoveForecast(
	ctx context.Context,
	topicId TopicId,
	forecaster ActorId,
) error {
	key := collections.Join(topicId, forecaster)
	return k.forecasts.Remove(ctx, key)
}

// Insert a complete set of forecasts for a topic/block.
func (k *WorkerKeeper) InsertActiveForecasts(
	ctx context.Context,
	topicId TopicId,
	nonceBlockHeight BlockHeight,
	forecasts types.Forecasts,
) error {
	key := collections.Join(topicId, nonceBlockHeight)
	return k.allForecasts.Set(ctx, key, forecasts)
}

func (k *WorkerKeeper) GetWorkerLatestInferenceByTopicId(
	ctx context.Context,
	topicId TopicId,
	worker ActorId,
) (types.Inference, error) {
	key := collections.Join(topicId, worker)
	return k.inferences.Get(ctx, key)
}

func (k *WorkerKeeper) GetWorkerLatestForecastByTopicId(
	ctx context.Context,
	topicId TopicId,
	worker ActorId,
) (types.Forecast, error) {
	key := collections.Join(topicId, worker)
	return k.forecasts.Get(ctx, key)
}

// True if worker is registered in topic, else False
func (k *WorkerKeeper) IsWorkerRegisteredInTopic(ctx context.Context, topicId TopicId, worker ActorId) (bool, error) {
	topicKey := collections.Join(topicId, worker)
	return k.topicWorkers.Has(ctx, topicKey)
}

func (k *WorkerKeeper) PruneInferences(ctx context.Context, blockRange *collections.PairRange[uint64, int64]) error {
	return k.allInferences.Clear(ctx, blockRange)
}

func (k *WorkerKeeper) PruneForecasts(ctx context.Context, blockRange *collections.PairRange[uint64, int64]) error {
	return k.allForecasts.Clear(ctx, blockRange)
}
