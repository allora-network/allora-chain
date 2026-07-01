package keeper

import (
	"context"
	"errors"
	"sort"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// InferenceAdmissionKind is the read-only admission outcome computed before an
// input inference is normalized into the temporary ELR label space.
type InferenceAdmissionKind uint8

const (
	// InferenceAdmissionNotAdmitted preserves passive score/liveness side
	// effects, but does not store an inference.
	InferenceAdmissionNotAdmitted InferenceAdmissionKind = iota
	// InferenceAdmissionOpenSlot admits the inferer without evicting anyone.
	InferenceAdmissionOpenSlot
	// InferenceAdmissionEvictLowest admits the inferer by replacing the current
	// lowest-scoring active inferer.
	InferenceAdmissionEvictLowest
)

// InferenceAdmissionPlan carries the exact admission decision and score snapshot
// that CommitAdmittedInference / CommitNonAdmittedInference apply. Planning must
// not mutate active sets, inference stores, score stores, or the temporary epoch
// label registry.
type InferenceAdmissionPlan struct {
	Kind             InferenceAdmissionKind
	Inferer          ActorId
	PreviousEmaScore types.Score
	LowestEmaScore   types.Score
	WorkerAddresses  []ActorId
	FirstSubmission  bool
}

func newInferenceAdmissionPlan(inferer ActorId) InferenceAdmissionPlan {
	return InferenceAdmissionPlan{
		Kind:    InferenceAdmissionNotAdmitted,
		Inferer: inferer,
		PreviousEmaScore: types.Score{
			TopicId:     0,
			BlockHeight: 0,
			Address:     "",
			Score:       alloraMath.ZeroDec(),
		},
		LowestEmaScore: types.Score{
			TopicId:     0,
			BlockHeight: 0,
			Address:     "",
			Score:       alloraMath.ZeroDec(),
		},
		WorkerAddresses: nil,
		FirstSubmission: false,
	}
}

// Admitted reports whether a planned inference is allowed to be normalized and
// stored. Non-admitted plans may still commit score/liveness side effects.
func (p InferenceAdmissionPlan) Admitted() bool {
	return p.Kind == InferenceAdmissionOpenSlot || p.Kind == InferenceAdmissionEvictLowest
}

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
	// map of (topic, worker) -> latest active inference for the open WSW
	inferences collections.Map[collections.Pair[TopicId, ActorId], types.Inference]
	// map of (topic, worker) -> forecast[]
	forecasts collections.Map[collections.Pair[TopicId, ActorId], types.Forecast]
	// map of (topic, block_height) -> Inference
	// allInferences is the committed per-epoch snapshot of accepted inferences,
	// keyed by (topicId, nonceBlockHeight); pruned after rewards are paid.
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
	if err := workerInfo.Validate(); err != nil {
		return errorsmod.Wrap(err, "worker info validation failed")
	}
	err := k.SetTopicWorker(ctx, topicId, worker)
	if err != nil {
		return errorsmod.Wrap(err, "error setting topic worker")
	}
	err = k.workers.Set(ctx, worker, workerInfo)
	if err != nil {
		return errorsmod.Wrap(err, "error setting worker")
	}
	return nil
}

// Set a topic worker
func (k *WorkerKeeper) SetTopicWorker(ctx context.Context, topicId TopicId, worker ActorId) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateBech32(worker); err != nil {
		return errorsmod.Wrap(err, "worker validation failed")
	}
	topicKey := collections.Join(topicId, worker)
	err := k.topicWorkers.Set(ctx, topicKey)
	if err != nil {
		return errorsmod.Wrap(err, "error setting topic worker")
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

// ResetWorkersIndividualSubmissionsForTopic resets worker submissions for a topic.
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

func (k *WorkerKeeper) GetInferencesAtBlock(ctx context.Context, topic types.Topic, block BlockHeight, outlierResistant bool) (*types.Inferences, error) {
	key := collections.Join(topic.Id, block)
	inferences, err := k.allInferences.Get(ctx, key)
	if errors.Is(err, collections.ErrNotFound) {
		return &types.Inferences{Inferences: []*types.Inference{}}, nil
	} else if err != nil {
		return nil, errorsmod.Wrap(err, "error getting inferences at block")
	}

	if outlierResistant {
		filteredInferences, err := k.topicKeeper.FilterOutlierResistantInferences(ctx, topic, inferences)
		if err != nil {
			return nil, errorsmod.Wrap(err, "error filtering outlier resistant inferences")
		}
		return &filteredInferences, nil
	}
	return &inferences, nil
}

// GetLatestTopicInferences retrieves the latest topic inferences and its block height.
func (k *WorkerKeeper) GetLatestTopicInferences(ctx context.Context, topic types.Topic, outlierResistant bool) (*types.Inferences, BlockHeight, error) {
	rng := collections.NewPrefixedPairRange[TopicId, BlockHeight](topic.Id).Descending()

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
			filteredInferences, err := k.topicKeeper.FilterOutlierResistantInferences(ctx, topic, *inferences)
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
	topic types.Topic,
	inferenceBlockHeight BlockHeight,
) error {
	ctx.Logger().Debug("Updating network inferences outlier metrics", "topicId", topic.Id, "blockHeight", inferenceBlockHeight)
	// Get all inferences at the block height
	inferences, err := k.GetInferencesAtBlock(ctx, topic, inferenceBlockHeight, false)
	if err != nil {
		return errorsmod.Wrap(err, "while getting inferences")
	}
	if len(inferences.Inferences) == 0 {
		// If there are no inferences, do not update the metrics
		ctx.Logger().Info("no inferences found, skipping update of outlier metrics", "topicId", topic.Id, "blockHeight", inferenceBlockHeight)
		return nil
	}

	// Create an array of the values
	values := make([]alloraMath.Dec, 0, len(inferences.Inferences))

	for _, inf := range inferences.Inferences {
		norm, err := inferenceOutlierScore(inf.Values)
		if err != nil {
			return errorsmod.Wrap(err, "inference norm failed")
		}

		values = append(values, norm)
	}

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
	previousMad, err := k.topicKeeper.GetMadInferences(ctx, topic.Id)
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

	ctx.Logger().Info("Setting new outlier-resistant mad", "newMad", newMad, "median", median, "topicId", topic.Id)

	// Set last mad inferences
	err = k.topicKeeper.SetMadInferences(ctx, topic.Id, newMad)
	if err != nil {
		return errorsmod.Wrap(err, "error setting last mad inferences")
	}

	// Set last median inferences
	err = k.topicKeeper.SetLastMedianInferences(ctx, topic.Id, median)
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

// PlanInferenceAdmission computes the admission decision and score snapshot for
// an inferer. It is read-only: no active-set, inference, score, or epoch label
// registry writes happen here. Apply the returned plan with the commit methods.
func (k *WorkerKeeper) PlanInferenceAdmission(
	ctx sdk.Context,
	topic types.Topic,
	nonceBlockHeight BlockHeight,
	inferer ActorId,
	maxTopInferersToReward uint64,
) (InferenceAdmissionPlan, error) {
	plan := newInferenceAdmissionPlan(inferer)
	// Check if the inferers already submitted the inference
	isActive, err := k.IsActiveInferer(ctx, topic.Id, inferer)
	if err != nil {
		return InferenceAdmissionPlan{}, errorsmod.Wrap(err, "error checking if worker already submitted inference")
	} else if isActive {
		return InferenceAdmissionPlan{}, errors.New("inference already submitted")
	}

	// Get previous EMA score for the current inferer
	previousEmaScore, err := k.scoresKeeper.GetInfererScoreEma(ctx, topic.Id, inferer)
	if err != nil {
		return InferenceAdmissionPlan{}, errorsmod.Wrapf(err, "Error getting inferer score ema")
	}
	// Only calc and save if there's a new update
	if previousEmaScore.BlockHeight >= nonceBlockHeight {
		return InferenceAdmissionPlan{}, types.ErrCantUpdateEmaMoreThanOncePerWindow
	}

	// Check if the inferer is new and set initial EMA score
	firstSubmission := false
	if previousEmaScore.BlockHeight == 0 {
		firstSubmission = true
		initialEmaScore, err := k.scoresKeeper.GetTopicInitialInfererEmaScore(ctx, topic.Id)
		if err != nil {
			return InferenceAdmissionPlan{}, errorsmod.Wrap(err, "error getting topic initial ema score")
		}
		previousEmaScore = types.Score{
			TopicId:     topic.Id,
			Address:     inferer,
			BlockHeight: nonceBlockHeight,
			Score:       initialEmaScore,
		}
	} else {
		// If not new: Penalise the inferer if needed
		previousEmaScore, err = k.actorPenaltiesKeeper.CalculateLivenessPenaltyToInferer(ctx, topic, nonceBlockHeight, previousEmaScore)
		if err != nil {
			return InferenceAdmissionPlan{}, errorsmod.Wrap(err, "error trying to penalise inferer")
		}

		// Update score nonce for liveness tracking
		previousEmaScore.BlockHeight = nonceBlockHeight
	}

	// Get lowest inferer score ema for the topic
	lowestEmaScore, _, err := k.scoresKeeper.GetLowestInfererScoreEma(ctx, topic.Id)
	if err != nil {
		return InferenceAdmissionPlan{}, errorsmod.Wrap(err, "error getting lowest inferer score ema")
	}

	// Get active inferers for topic
	workerAddresses, err := k.GetActiveInferersForTopic(ctx, topic.Id)
	if err != nil {
		return InferenceAdmissionPlan{}, errorsmod.Wrap(err, "error getting active inferers for topic")
	}

	plan.PreviousEmaScore = previousEmaScore
	plan.LowestEmaScore = lowestEmaScore
	plan.WorkerAddresses = workerAddresses
	plan.FirstSubmission = firstSubmission

	// If there are less than maxTopInferersToReward, add the current inferer, update the lowest inferer score ema if needed, and return
	if uint64(len(workerAddresses)) < maxTopInferersToReward {
		plan.Kind = InferenceAdmissionOpenSlot
		return plan, nil
	}

	if previousEmaScore.Score.Gt(lowestEmaScore.Score) {
		plan.Kind = InferenceAdmissionEvictLowest
	}
	return plan, nil
}

// CommitAdmittedInference applies an admitted plan: it writes the shared score
// snapshot and stores the (already normalized and validated) inference, evicting
// the lowest active inferer when the plan calls for it. inference MUST be non-nil
// and plan.Admitted() MUST be true.
func (k *WorkerKeeper) CommitAdmittedInference(
	ctx sdk.Context,
	topic types.Topic,
	nonceBlockHeight BlockHeight,
	inference *types.Inference,
	plan InferenceAdmissionPlan,
) error {
	if !plan.Admitted() {
		return errorsmod.Wrapf(sdkerrors.ErrLogic, "plan is not admitted (kind %d); use CommitNonAdmittedInference", plan.Kind)
	}
	if err := validateInferenceAdmissionPlan(inference, plan); err != nil {
		return err
	}
	if err := k.commitPlannedInfererScore(ctx, topic, plan); err != nil {
		return err
	}

	switch plan.Kind {
	case InferenceAdmissionOpenSlot:
		return k.commitOpenSlotInferencePlan(ctx, topic, inference, plan)
	case InferenceAdmissionEvictLowest:
		return k.commitEvictionInferencePlan(ctx, topic, nonceBlockHeight, inference, plan)
	}
	return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "unexpected admitted kind: %d", plan.Kind)
}

// CommitNonAdmittedInference applies a not-admitted plan: it writes only the
// passive score/liveness side effects and stores no inference, so it takes no
// inference argument. plan.Admitted() MUST be false.
func (k *WorkerKeeper) CommitNonAdmittedInference(
	ctx sdk.Context,
	topic types.Topic,
	nonceBlockHeight BlockHeight,
	plan InferenceAdmissionPlan,
) error {
	if plan.Admitted() {
		return errorsmod.Wrapf(sdkerrors.ErrLogic, "plan is admitted (kind %d); use CommitAdmittedInference", plan.Kind)
	}
	if err := validateInferenceAdmissionPlan(nil, plan); err != nil {
		return err
	}
	if err := k.commitPlannedInfererScore(ctx, topic, plan); err != nil {
		return err
	}
	return k.commitNotAdmittedInferencePlan(ctx, topic, nonceBlockHeight, plan)
}

func validateInferenceAdmissionPlan(inference *types.Inference, plan InferenceAdmissionPlan) error {
	if plan.Inferer == "" {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "inferer is empty")
	}
	if inference != nil && inference.Inferer != plan.Inferer {
		return errorsmod.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"inference inferer %s does not match admission plan inferer %s",
			inference.Inferer,
			plan.Inferer,
		)
	}
	if plan.PreviousEmaScore.Address != plan.Inferer {
		return errorsmod.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"planned EMA score address %s does not match inferer %s",
			plan.PreviousEmaScore.Address,
			plan.Inferer,
		)
	}
	switch plan.Kind {
	case InferenceAdmissionOpenSlot, InferenceAdmissionEvictLowest:
		if inference == nil {
			return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "admitted inference is nil")
		}
	case InferenceAdmissionNotAdmitted:
		// Non-admitted plans still commit score/liveness side effects, but no
		// inference is required because it will not be stored.
	default:
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "unknown inference admission kind: %d", plan.Kind)
	}
	return nil
}

func (k *WorkerKeeper) commitPlannedInfererScore(
	ctx sdk.Context,
	topic types.Topic,
	plan InferenceAdmissionPlan,
) error {
	// Shared EMA snapshot for every admission outcome. Not-admitted (non-first)
	// then overwrites this same key with the passive-set quantile EMA.
	if err := k.scoresKeeper.SetInfererScoreEma(ctx, topic.Id, plan.Inferer, plan.PreviousEmaScore); err != nil {
		if plan.FirstSubmission {
			return errorsmod.Wrap(err, "error setting initial inferer score ema")
		}
		return errorsmod.Wrap(err, "error setting penalised inferer score ema")
	}
	return nil
}

func (k *WorkerKeeper) commitOpenSlotInferencePlan(
	ctx sdk.Context,
	topic types.Topic,
	inference *types.Inference,
	plan InferenceAdmissionPlan,
) error {
	// Update lowest inferer score ema if needed
	if uint64(len(plan.WorkerAddresses)) == 0 || plan.LowestEmaScore.Score.Gt(plan.PreviousEmaScore.Score) {
		err := k.scoresKeeper.SetLowestInfererScoreEma(ctx, topic.Id, plan.PreviousEmaScore)
		if err != nil {
			return errorsmod.Wrap(err, "error setting lowest inferer score ema")
		}
	}

	err := k.AddActiveInferer(ctx, topic.Id, inference.Inferer)
	if err != nil {
		return errorsmod.Wrap(err, "error adding active inferer")
	}
	if err := k.InsertInference(ctx, topic.Id, *inference); err != nil {
		return errorsmod.Wrap(err, "error inserting inference")
	}
	return nil
}

func (k *WorkerKeeper) commitEvictionInferencePlan(
	ctx sdk.Context,
	topic types.Topic,
	nonceBlockHeight BlockHeight,
	inference *types.Inference,
	plan InferenceAdmissionPlan,
) error {
	// Update EMA score for the lowest score inferer, who is not the current inferer
	err := k.scoresKeeper.CalcAndSaveInfererScoreEmaWithLastSavedTopicQuantile(
		ctx,
		topic,
		nonceBlockHeight,
		plan.LowestEmaScore,
	)
	if err != nil {
		return errorsmod.Wrap(err, "error calculating and saving inferer score ema with last saved topic quantile")
	}

	// RemoveActiveInferer is a no-op for inactive inferers, so verify the
	// planned eviction target is active first.
	isActive, err := k.IsActiveInferer(ctx, topic.Id, plan.LowestEmaScore.Address)
	if err != nil {
		return errorsmod.Wrap(err, "error checking if inferer is active")
	}
	if !isActive {
		return errors.New("inferer with lowest score is not active")
	}

	// Remove inferer with lowest score
	err = k.RemoveActiveInferer(ctx, topic.Id, plan.LowestEmaScore.Address)
	if err != nil {
		return errorsmod.Wrap(err, "error removing active inferer")
	}
	// Clear the evicted worker's temporary inference so it can't leak into
	// close-time finalization.
	if err := k.RemoveInference(ctx, topic.Id, plan.LowestEmaScore.Address); err != nil {
		return errorsmod.Wrap(err, "error removing evicted inferer inference")
	}
	// Add new active inferer
	err = k.AddActiveInferer(ctx, topic.Id, inference.Inferer)
	if err != nil {
		return errorsmod.Wrap(err, "error adding active inferer")
	}
	// Calculate new lowest score with updated infererAddresses
	err = k.scoresKeeper.UpdateLowestScoreFromInfererAddresses(ctx, topic.Id, plan.WorkerAddresses, inference.Inferer, plan.LowestEmaScore.Address)
	if err != nil {
		return errorsmod.Wrap(err, "error getting low score from all inferences")
	}
	if err := k.InsertInference(ctx, topic.Id, *inference); err != nil {
		return errorsmod.Wrap(err, "error inserting inference")
	}
	return nil
}

func (k *WorkerKeeper) commitNotAdmittedInferencePlan(
	ctx sdk.Context,
	topic types.Topic,
	nonceBlockHeight BlockHeight,
	plan InferenceAdmissionPlan,
) error {
	// Passive-set quantile EMA for the current (lowest-scoring) inferer;
	// intentionally re-writes the snapshot from commitPlannedInfererScore.
	if !plan.FirstSubmission { // Only update if not a new inferer
		err := k.scoresKeeper.CalcAndSaveInfererScoreEmaWithLastSavedTopicQuantile(ctx, topic, nonceBlockHeight, plan.PreviousEmaScore)
		if err != nil {
			return errorsmod.Wrap(err, "error calculating and saving inferer score ema with last saved topic quantile")
		}
	}
	return nil
}

// InsertInference inserts an active WSW inference for a specific topic.
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

// RemoveInference removes an inference from a inferer.
func (k *WorkerKeeper) RemoveInference(
	ctx context.Context,
	topicId TopicId,
	inferer ActorId,
) error {
	key := collections.Join(topicId, inferer)
	return k.inferences.Remove(ctx, key)
}

// GetWorkerLatestInferenceByTopicId returns the live WSW inference for
// (topicId, inferer). Returns collections.ErrNotFound if the inferer has not
// been admitted in the current WSW.
func (k *WorkerKeeper) GetWorkerLatestInferenceByTopicId(
	ctx context.Context,
	topicId TopicId,
	inferer ActorId,
) (types.Inference, error) {
	key := collections.Join(topicId, inferer)
	return k.inferences.Get(ctx, key)
}

// GetWorkerLatestInputInferenceByTopicId denormalizes the live WSW dense
// inference into its canonical input-shaped view using the temporary ELR.
func (k *WorkerKeeper) GetWorkerLatestInputInferenceByTopicId(
	ctx context.Context,
	topic types.Topic,
	inferer ActorId,
) (*types.InputInference, error) {
	inference, err := k.GetWorkerLatestInferenceByTopicId(ctx, topic.Id, inferer)
	if err != nil {
		return nil, err
	}
	registry, err := k.topicKeeper.GetEpochLabelRegistry(ctx, topic.Id, inference.BlockHeight)
	if err != nil {
		return nil, err
	}
	params, err := k.paramsKeeper.GetParams(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error getting params for input inference denormalization")
	}
	return DenormalizeInferenceToInput(
		topic,
		registry,
		inference,
		params.MaxCanonicalLabelByteLength,
	)
}

// LoadActiveInfererInferencesForClose reads the temporary dense inferences for
// the final active inferer set. Inferences are returned sorted by inferer so
// close-time registry construction and finalization are deterministic.
func (k *WorkerKeeper) LoadActiveInfererInferencesForClose(
	ctx context.Context,
	topic types.Topic,
	nonce BlockHeight,
	workers []ActorId,
) ([]*types.Inference, error) {
	sortedWorkers := append([]ActorId(nil), workers...)
	sort.Strings(sortedWorkers)

	activeInferences := make([]*types.Inference, 0, len(sortedWorkers))
	for _, inferer := range sortedWorkers {
		in, err := k.GetWorkerLatestInferenceByTopicId(ctx, topic.Id, inferer)
		if err != nil {
			if errors.Is(err, collections.ErrNotFound) {
				return nil, errorsmod.Wrapf(
					sdkerrors.ErrLogic,
					"missing temporary inference for active inferer %s",
					inferer,
				)
			}
			return nil, errorsmod.Wrapf(err, "error reading temporary inference for active inferer %s", inferer)
		}
		if err := validateActiveInferenceForClose(topic, nonce, inferer, in); err != nil {
			return nil, err
		}
		inference := in
		activeInferences = append(activeInferences, &inference)
	}
	return activeInferences, nil
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

func (k *WorkerKeeper) GetWorkerLatestForecastByTopicId(
	ctx context.Context,
	topicId TopicId,
	worker ActorId,
) (types.Forecast, error) {
	key := collections.Join(topicId, worker)
	return k.forecasts.Get(ctx, key)
}

// NormalizeInputInference converts a worker-submitted InputInference into a
// dense temporary *types.Inference aligned to the WSW temporary
// EpochLabelRegistry.
//
// Preconditions: the caller (msgserver) must have already canonicalized and
// validated `in` via (*InputInference).ValidateWithLimits, which enforces
// canonical labels, post-canon dedupe, the effective per-submission cap, and
// the topic whitelist.
//
// SINGLE: Values = [scalar]. Accepts both the scalar Value field and a
//
//	1-element labeled Values list (the latter wins if present and the label
//	is canonical "y"; mismatched single-value labels are rejected).
//
// MULTI: registers non-default submitted labels in first-seen order, initializes
//
//	the dense vector with topic.LabelDefaultValue, and scatters non-default
//	values into their temporary registry ids.
//
// The unity check (topic.RequireUnity) runs here because it depends only on
// the submitted labeled values, not on the registry.
func (k *WorkerKeeper) NormalizeInputInference(
	ctx context.Context,
	topic types.Topic,
	nonce BlockHeight,
	in *types.InputInference,
) (*types.Inference, error) {
	if in == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "inference is nil")
	}

	switch topic.OutputArity {
	case types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE:
		if len(in.Values) > 1 {
			return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "single-arity inference accepts at most one value")
		}
		var dec alloraMath.Dec
		if len(in.Values) == 1 {
			lv := in.Values[0]
			if lv == nil {
				return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "nil labeled value")
			}
			// The canonical label for a SINGLE-arity submission is always "y".
			// Accept a missing label (legacy scalar path), accept "y", reject
			// anything else so that operators using SINGLE topics can't smuggle
			// in extra labels.
			if err := types.ValidateSingleArityLabel(lv.Label); err != nil {
				return nil, err
			}
			dec = lv.Value.ToDec()
		} else {
			dec = in.Value.ToDec()
		}
		if dec.IsNaN() || !dec.IsFinite() {
			return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "invalid scalar inference value")
		}
		params, err := k.paramsKeeper.GetParams(ctx)
		if err != nil {
			return nil, errorsmod.Wrap(err, "failed to get params for inference normalization")
		}
		if _, _, err := k.topicKeeper.RegisterEpochLabels(
			ctx,
			topic.Id,
			topic.LabelCaseSensitive,
			nonce,
			[]string{types.SingleArityCanonicalLabel},
			params.MaxCanonicalLabelByteLength,
			params.MaxEpochLabelRegistrySize,
		); err != nil {
			return nil, errorsmod.Wrap(err, "failed to register single-arity label")
		}
		return &types.Inference{
			TopicId:     in.TopicId,
			BlockHeight: in.BlockHeight,
			Inferer:     in.Inferer,
			Values:      []alloraMath.Dec{dec},
			ExtraData:   in.ExtraData,
			Proof:       in.Proof,
		}, nil
	case types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI:
		// fall through
	default:
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "output_arity is invalid")
	}

	if len(in.Values) == 0 {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "multi-arity inference requires labeled values")
	}

	submitted := make([]*types.InputLabeledValue, 0, len(in.Values))
	for _, lv := range in.Values {
		if lv == nil {
			return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "nil labeled value")
		}
		if lv.Label == "" {
			return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "label must be canonicalized before Normalize")
		}
		submitted = append(submitted, lv)
	}
	sumSubmitted := alloraMath.ZeroDec()
	// A "scattered" value is one submitted value paired with the registry id it
	// will occupy. Only non-default labels are registered: a value equal to
	// topic.LabelDefaultValue is dropped here because in the dense vector it is
	// indistinguishable from a label that was never submitted. labelsToRegister[i]
	// and valuesToScatter[i] are appended in lockstep so they stay positionally
	// aligned with the ids RegisterEpochLabels returns.
	type scatteredValue struct {
		id    LabelId
		value alloraMath.Dec
	}
	labelsToRegister := make([]string, 0, len(submitted))
	valuesToScatter := make([]alloraMath.Dec, 0, len(submitted))
	for _, lv := range submitted {
		dec := lv.Value.ToDec()
		if dec.IsNaN() || !dec.IsFinite() {
			return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "label %s has invalid value", lv.Label)
		}
		var err error
		sumSubmitted, err = sumSubmitted.Add(dec)
		if err != nil {
			return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "failed to sum submitted value: %s", err)
		}
		if !dec.Equal(topic.LabelDefaultValue) {
			labelsToRegister = append(labelsToRegister, lv.Label)
			valuesToScatter = append(valuesToScatter, dec)
		}
	}
	if len(labelsToRegister) == 0 {
		return nil, errorsmod.Wrap(
			sdkerrors.ErrInvalidRequest,
			"multi-arity inference requires at least one non-default label value",
		)
	}
	params, err := k.paramsKeeper.GetParams(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to get params for inference normalization")
	}
	registeredIDs, registry, err := k.topicKeeper.RegisterEpochLabels(
		ctx,
		topic.Id,
		topic.LabelCaseSensitive,
		nonce,
		labelsToRegister,
		params.MaxCanonicalLabelByteLength,
		params.MaxEpochLabelRegistrySize,
	)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to register inference labels")
	}
	scattered := make([]scatteredValue, 0, len(registeredIDs))
	for i, id := range registeredIDs {
		scattered = append(scattered, scatteredValue{id: id, value: valuesToScatter[i]})
	}

	if topic.RequireUnity {
		diff, err := sumSubmitted.Sub(alloraMath.OneDec())
		if err != nil {
			return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "failed to compute unity diff: %v", err)
		}
		diff, err = diff.Abs()
		if err != nil {
			return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "failed to abs unity diff: %v", err)
		}
		if diff.Gt(topic.UnityTolerance) {
			return nil, errorsmod.Wrapf(
				sdkerrors.ErrInvalidRequest,
				"require_unity violated: sum=%s tol=%s",
				sumSubmitted.String(), topic.UnityTolerance.String(),
			)
		}
	}

	// The dense vector is sized to the WHOLE epoch registry (shared by every
	// inferer at this topic+nonce and only growing), not to this submission's
	// label count, so any slot we don't scatter into stays LabelDefaultValue.
	values := make([]alloraMath.Dec, len(registry.Labels))
	for i := range values {
		values[i] = topic.LabelDefaultValue
	}
	// Scatter each value into its registry slot. Ids are 1-based and dense by
	// construction, so this bounds check is an internal invariant (ErrLogic),
	// not a user error.
	for _, item := range scattered {
		if item.id == 0 || int(item.id) > len(values) {
			return nil, errorsmod.Wrapf(sdkerrors.ErrLogic, "registered label id %d out of range", item.id)
		}
		values[labelSlot(item.id)] = item.value
	}

	return &types.Inference{
		TopicId:     in.TopicId,
		BlockHeight: in.BlockHeight,
		Inferer:     in.Inferer,
		Values:      values,
		ExtraData:   in.ExtraData,
		Proof:       in.Proof,
	}, nil
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

func inferenceOutlierScore(values []alloraMath.Dec) (alloraMath.Dec, error) {
	if len(values) == 0 {
		return alloraMath.Dec{}, errorsmod.Wrap(sdkerrors.ErrLogic, "inference has empty values")
	}

	if len(values) == 1 {
		return values[0], nil
	}

	sumSquares := alloraMath.ZeroDec()
	for _, v := range values {
		vv, err := v.Mul(v)
		if err != nil {
			return alloraMath.Dec{}, errorsmod.Wrap(err, "error squaring inference value")
		}
		sumSquares, err = sumSquares.Add(vv)
		if err != nil {
			return alloraMath.Dec{}, errorsmod.Wrap(err, "error accumulating squared inference values")
		}
	}

	score, err := sumSquares.Sqrt()
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrap(err, "error taking sqrt of inference norm")
	}
	return score, nil
}
