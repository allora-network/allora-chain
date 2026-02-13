package keeper

import (
	"context"
	"errors"
	"strings"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func NewReputerLossKeeper(
	cdc codec.BinaryCodec,
	sb *collections.SchemaBuilder,
	scoresKeeper *ScoresKeeper,
	topicKeeper *TopicKeeper,
	actorPenaltiesKeeper *ActorPenaltiesKeeper,
) *ReputerLossKeeper {
	return &ReputerLossKeeper{
		reputers:             collections.NewMap(sb, types.ReputerNodesKey, "reputer_nodes", collections.StringKey, codec.CollValue[types.OffchainNode](cdc)),
		topicReputers:        collections.NewKeySet(sb, types.TopicReputersKey, "topic_reputers", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey)),
		activeReputers:       collections.NewKeySet(sb, types.ActiveReputersKey, "active_reputers", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey)),
		networkLossBundles:   collections.NewMap(sb, types.NetworkLossBundlesKey, "value_bundles_network", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.ValueBundle](cdc)),
		lossBundles:          collections.NewMap(sb, types.LossBundlesKey, "loss_bundles", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.ReputerValueBundle](cdc)),
		allLossBundles:       collections.NewMap(sb, types.AllLossBundlesKey, "value_bundles_all", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.ReputerValueBundles](cdc)),
		scoresKeeper:         scoresKeeper,
		topicKeeper:          topicKeeper,
		actorPenaltiesKeeper: actorPenaltiesKeeper,
	}
}

type ReputerLossKeeper struct {
	// map of reputer id to node data about that reputer
	reputers collections.Map[ActorId, types.OffchainNode]
	// for a topic, what is every reputer node that has registered to it?
	topicReputers collections.KeySet[collections.Pair[TopicId, ActorId]]
	// active reputers for a topic
	activeReputers collections.KeySet[collections.Pair[TopicId, ActorId]]
	// map of (topic, reputer) -> reputer loss
	lossBundles collections.Map[collections.Pair[TopicId, Reputer], types.ReputerValueBundle]
	// map of (topic, block_height) -> ReputerValueBundles (1 per reputer active at that time)
	allLossBundles collections.Map[collections.Pair[TopicId, BlockHeight], types.ReputerValueBundles]
	// map of (topic, block_height) -> ValueBundle (1 network wide bundle per timestep)
	networkLossBundles collections.Map[collections.Pair[TopicId, BlockHeight], types.ValueBundle]
	// scores keeper
	scoresKeeper *ScoresKeeper
	// topic keeper
	topicKeeper *TopicKeeper
	// actor penalties keeper
	actorPenaltiesKeeper *ActorPenaltiesKeeper
}

// True if reputer is registered in topic, else False
func (k *ReputerLossKeeper) IsReputerRegisteredInTopic(ctx context.Context, topicId TopicId, reputer ActorId) (bool, error) {
	topicKey := collections.Join(topicId, reputer)
	return k.topicReputers.Has(ctx, topicKey)
}

// Adds a new reputer to the reputer tracking data structures, reputers and topicReputers
func (k *ReputerLossKeeper) InsertReputer(ctx context.Context, topicId TopicId, reputer ActorId, reputerInfo types.OffchainNode) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateBech32(reputer); err != nil {
		return errorsmod.Wrap(err, "reputer validation failed")
	}
	if err := reputerInfo.Validate(); err != nil {
		return errorsmod.Wrap(err, "reputer info validation failed")
	}
	topicKey := collections.Join(topicId, reputer)
	err := k.topicReputers.Set(ctx, topicKey)
	if err != nil {
		return errorsmod.Wrap(err, "error setting topic reputer")
	}
	err = k.reputers.Set(ctx, reputer, reputerInfo)
	if err != nil {
		return errorsmod.Wrap(err, "error setting reputer")
	}
	return nil
}

// Remove a reputer to the reputer tracking data structures and topicReputers
func (k *ReputerLossKeeper) RemoveReputer(ctx context.Context, topicId TopicId, reputer ActorId) error {
	topicKey := collections.Join(topicId, reputer)
	err := k.topicReputers.Remove(ctx, topicKey)
	if err != nil {
		return errorsmod.Wrap(err, "error removing topic reputer")
	}
	return nil
}

func (k *ReputerLossKeeper) GetReputerInfo(ctx sdk.Context, reputerKey ActorId) (types.OffchainNode, error) {
	return k.reputers.Get(ctx, reputerKey)
}

// UpdateReputerOwner updates the payout owner associated with a reputer node.
// Returns:
//   - (string) old owner
func (k *ReputerLossKeeper) UpdateReputerOwner(ctx context.Context, reputer ActorId, newOwner string) (string, error) {
	if err := types.ValidateBech32(reputer); err != nil {
		return "", errorsmod.Wrap(err, "reputer validation failed")
	}
	if err := types.ValidateBech32(newOwner); err != nil {
		return "", errorsmod.Wrap(err, "new owner validation failed")
	}
	nodeInfo, err := k.reputers.Get(ctx, reputer)
	if errors.Is(err, collections.ErrNotFound) {
		return "", errorsmod.Wrapf(types.ErrAddressNotRegistered, "reputer %s", reputer)
	} else if err != nil {
		return "", errorsmod.Wrap(err, "error getting reputer info")
	}
	oldOwner := nodeInfo.Owner
	nodeInfo.Owner = newOwner
	if err := nodeInfo.Validate(); err != nil {
		return "", errorsmod.Wrap(err, "reputer info validation failed")
	}
	if err := k.reputers.Set(ctx, reputer, nodeInfo); err != nil {
		return "", errorsmod.Wrap(err, "error setting reputer info")
	}
	return oldOwner, nil
}

// AddActiveReputer adds a reputer to the active reputers set for a topic
func (k *ReputerLossKeeper) AddActiveReputer(ctx context.Context, topicId TopicId, reputer ActorId) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "invalid topic id")
	}
	if err := types.ValidateBech32(reputer); err != nil {
		return errorsmod.Wrap(err, "invalid reputer address")
	}
	key := collections.Join(topicId, reputer)
	return k.activeReputers.Set(ctx, key)
}

// IsActiveReputer checks if a reputer is in the active reputers set for a topic
func (k *ReputerLossKeeper) IsActiveReputer(ctx context.Context, topicId TopicId, reputer ActorId) (bool, error) {
	key := collections.Join(topicId, reputer)
	return k.activeReputers.Has(ctx, key)
}

// RemoveActiveReputer removes a reputer from the active reputers set for a topic
func (k *ReputerLossKeeper) RemoveActiveReputer(ctx context.Context, topicId TopicId, reputer ActorId) error {
	key := collections.Join(topicId, reputer)
	return k.activeReputers.Remove(ctx, key)
}

// GetActiveReputersForTopic returns all active reputers for a specific topic
func (k *ReputerLossKeeper) GetActiveReputersForTopic(ctx context.Context, topicId TopicId) ([]ActorId, error) {
	var reputers []ActorId
	rng := collections.NewPrefixedPairRange[TopicId, ActorId](topicId)
	err := k.activeReputers.Walk(ctx, rng, func(key collections.Pair[TopicId, ActorId]) (bool, error) {
		reputers = append(reputers, key.K2())
		return false, nil
	})
	if err != nil {
		return nil, errorsmod.Wrap(err, "error walking active reputers")
	}
	return reputers, nil
}

// ResetActiveReputersForTopic resets the active reputers for a topic
func (k *ReputerLossKeeper) ResetActiveReputersForTopic(ctx context.Context, topicId TopicId) error {
	reputerRange := collections.NewPrefixedPairRange[TopicId, ActorId](topicId)
	if err := k.activeReputers.Clear(ctx, reputerRange); err != nil {
		return errorsmod.Wrap(err, "error clearing active reputers")
	}
	return nil
}

// Append loss bundle for a topic and blockHeight
func (k *ReputerLossKeeper) AppendReputerLoss(
	ctx sdk.Context,
	topic types.Topic,
	moduleParams types.Params,
	nonceBlockHeight BlockHeight,
	reputerLoss *types.LossBundle,
) error {
	if reputerLoss == nil {
		return errors.New("invalid reputerLoss bundle: reputer is empty or nil")
	}

	// Check if the reputer already submitted the loss
	isActive, err := k.IsActiveReputer(ctx, topic.Id, reputerLoss.Reputer)
	if err != nil {
		return errorsmod.Wrap(err, "error checking if reputer already submitted loss")
	} else if isActive {
		return errors.New("reputer loss already submitted")
	}

	previousEmaScore, err := k.scoresKeeper.GetReputerScoreEma(ctx, topic.Id, reputerLoss.Reputer)
	if err != nil {
		return errorsmod.Wrapf(err, "Error getting reputer score ema")
	}
	// Only calc and save if there's a new update
	if previousEmaScore.BlockHeight >= nonceBlockHeight {
		return types.ErrCantUpdateEmaMoreThanOncePerWindow
	}

	// Check if the reputer is new and set initial EMA score
	firstSubmission := false
	if previousEmaScore.BlockHeight == 0 {
		firstSubmission = true
		initialEmaScore, err := k.scoresKeeper.GetTopicInitialReputerEmaScore(ctx, topic.Id)
		if err != nil {
			return errorsmod.Wrap(err, "error getting topic initial ema score")
		}
		err = k.scoresKeeper.SetReputerScoreEma(ctx, topic.Id, reputerLoss.Reputer, types.Score{
			TopicId:     topic.Id,
			Address:     reputerLoss.Reputer,
			BlockHeight: nonceBlockHeight,
			Score:       initialEmaScore,
		})
		if err != nil {
			return errorsmod.Wrap(err, "error setting initial reputer score ema")
		}
	} else {
		// If not new: Penalise the reputer if needed
		previousEmaScore, err = k.actorPenaltiesKeeper.ApplyLivenessPenaltyToReputer(ctx, topic, nonceBlockHeight, previousEmaScore)
		if err != nil {
			return errorsmod.Wrap(err, "error trying to penalise reputer")
		}

		// Update score nonce for liveness tracking
		previousEmaScore.BlockHeight = nonceBlockHeight
		err = k.scoresKeeper.SetReputerScoreEma(ctx, topic.Id, previousEmaScore.Address, previousEmaScore)
		if err != nil {
			return errorsmod.Wrap(err, "error setting penalised reputer score ema")
		}
	}

	lowestEmaScore, _, err := k.scoresKeeper.GetLowestReputerScoreEma(ctx, topic.Id)
	if err != nil {
		return errorsmod.Wrap(err, "error getting lowest reputer score ema")
	}

	reputerAddresses, err := k.GetActiveReputersForTopic(ctx, topic.Id)
	if err != nil {
		return errorsmod.Wrap(err, "error getting active reputers for topic")
	}
	// If there are less than maxTopReputersToReward, add the current reputer
	if uint64(len(reputerAddresses)) < moduleParams.MaxTopReputersToReward {
		if uint64(len(reputerAddresses)) == 0 || lowestEmaScore.Score.Gt(previousEmaScore.Score) {
			err = k.scoresKeeper.SetLowestReputerScoreEma(ctx, topic.Id, previousEmaScore)
			if err != nil {
				return errorsmod.Wrap(err, "error setting lowest reputer score ema")
			}
		}

		err = k.AddActiveReputer(ctx, topic.Id, reputerLoss.Reputer)
		if err != nil {
			return errorsmod.Wrap(err, "error adding active reputer")
		}
		return k.InsertReputerLoss(ctx, topic.Id, *reputerLoss)
	}

	if previousEmaScore.Score.Gt(lowestEmaScore.Score) {
		// Update EMA score for the lowest score reputer, who is not the current reputer
		err = k.scoresKeeper.CalcAndSaveReputerScoreEmaWithLastSavedTopicQuantile(
			ctx,
			topic,
			nonceBlockHeight,
			lowestEmaScore,
		)
		if err != nil {
			return errorsmod.Wrap(err, "error calculating and saving reputer score ema with last saved topic quantile")
		}

		// Check if the reputer with lowest score is active before removing it, because remove will not fail if the reputer is not active
		isActive, err := k.IsActiveReputer(ctx, topic.Id, lowestEmaScore.Address)
		if err != nil {
			return errorsmod.Wrap(err, "error checking if reputer is active")
		}
		if !isActive {
			return errors.New("reputer with lowest score is not active")
		}

		// Remove reputer with lowest score
		err = k.RemoveActiveReputer(ctx, topic.Id, lowestEmaScore.Address)
		if err != nil {
			return errorsmod.Wrap(err, "error removing active reputer")
		}
		// Remove reputer loss from reputer with lowest score
		err = k.RemoveReputerLoss(ctx, topic.Id, lowestEmaScore.Address)
		if err != nil {
			return errorsmod.Wrap(err, "error removing reputer loss from reputer")
		}
		// Add new active reputer
		err = k.AddActiveReputer(ctx, topic.Id, reputerLoss.Reputer)
		if err != nil {
			return errorsmod.Wrap(err, "error adding active reputer")
		}
		// Calculate new lowest score with updated reputerAddresses
		err = k.scoresKeeper.UpdateLowestScoreFromReputerAddresses(ctx, topic.Id, reputerAddresses, reputerLoss.Reputer, lowestEmaScore.Address)
		if err != nil {
			return errorsmod.Wrap(err, "error getting low score from all reputer losses")
		}
		return k.InsertReputerLoss(ctx, topic.Id, *reputerLoss)
	} else {
		// Update EMA score for the current reputer, who is the lowest score reputer
		if !firstSubmission {
			err = k.scoresKeeper.CalcAndSaveReputerScoreEmaWithLastSavedTopicQuantile(ctx, topic, nonceBlockHeight, previousEmaScore)
			if err != nil {
				return errorsmod.Wrap(err, "error calculating and saving reputer score ema with last saved topic quantile")
			}
		}
	}
	return nil
}

// GetReputerLatestLossByTopicId
func (k *ReputerLossKeeper) GetReputerLatestLossByTopicId(
	ctx context.Context,
	topicId TopicId,
	reputer ActorId,
) (types.LossBundle, error) {
	key := collections.Join(topicId, reputer)
	valueBundle, err := k.lossBundles.Get(ctx, key)
	if err != nil {
		return types.LossBundle{}, err
	}
	return *valueBundle.GetValueBundle(), err
}

// InsertReputerLoss inserts a reputer loss for a specific topic
func (k *ReputerLossKeeper) InsertReputerLoss(
	ctx context.Context,
	topicId TopicId,
	reputerLoss types.LossBundle,
) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	err := reputerLoss.Validate()
	if err != nil {
		return errorsmod.Wrap(err, "reputer loss is invalid")
	}
	key := collections.Join(topicId, reputerLoss.Reputer)
	return k.lossBundles.Set(ctx, key, types.ReputerValueBundle{
		ValueBundle: &reputerLoss,
	})
}

// RemoveReputerLoss removes a reputer loss for a specific topic
func (k *ReputerLossKeeper) RemoveReputerLoss(
	ctx context.Context,
	topicId TopicId,
	reputer ActorId,
) error {
	key := collections.Join(topicId, reputer)
	return k.lossBundles.Remove(ctx, key)
}

// Insert a loss bundle for a topic and timestamp but do it in the CloseReputerNonce, so reputer loss bundles are known to be validated
// and not validated again (this is due to poor data type design choices, see PROTO-2369)
func (k *ReputerLossKeeper) InsertActiveReputerLosses(
	ctx context.Context,
	topicId TopicId,
	block BlockHeight,
	reputerLossBundles types.LossBundles,
) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateBlockHeight(block); err != nil {
		return errorsmod.Wrap(err, "block height validation failed")
	}
	if err := reputerLossBundles.Validate(); err != nil {
		// in the singular case of a bundle that has been filtered, we allow
		// the signature validation to fail. This is secure because we already do
		// bundle validation in CloseReputerNonce before calling this function
		// however this should be well and truly fixed by PROTO-2369
		if !strings.Contains(err.Error(), "signature verification failed") {
			return errorsmod.Wrap(err, "reputer loss bundles validation failed")
		}
	}
	key := collections.Join(topicId, block)
	valueBundles := make([]*types.ReputerValueBundle, len(reputerLossBundles))
	for i := range reputerLossBundles {
		valueBundles[i] = &types.ReputerValueBundle{
			ValueBundle: reputerLossBundles[i],
		}
	}
	return k.allLossBundles.Set(ctx, key, types.ReputerValueBundles{ReputerValueBundles: valueBundles})
}

// Get loss bundles for a topic/timestamp
func (k *ReputerLossKeeper) GetReputerLossBundlesAtBlock(ctx context.Context, topicId TopicId, block BlockHeight) (types.LossBundles, error) {
	key := collections.Join(topicId, block)
	reputerLossBundles, err := k.allLossBundles.Get(ctx, key)

	if errors.Is(err, collections.ErrNotFound) {
		return nil, nil
	} else if err != nil {
		return nil, errorsmod.Wrap(err, "error getting reputer loss bundles at block")
	}
	lossBundles := make(types.LossBundles, len(reputerLossBundles.ReputerValueBundles))
	for i := range reputerLossBundles.ReputerValueBundles {
		lossBundles[i] = reputerLossBundles.ReputerValueBundles[i].GetValueBundle()
	}
	return lossBundles, nil
}

// Insert a network loss bundle for a topic and block.
func (k *ReputerLossKeeper) InsertNetworkLossBundleAtBlock(
	ctx context.Context,
	topicId TopicId,
	block BlockHeight,
	lossBundle types.ValueBundle,
) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateBlockHeight(block); err != nil {
		return errorsmod.Wrap(err, "block height validation failed")
	}
	if err := lossBundle.Validate(); err != nil {
		return errorsmod.Wrap(err, "loss bundle validation failed")
	}
	key := collections.Join(topicId, block)
	return k.networkLossBundles.Set(ctx, key, lossBundle)
}

// A function that accepts a topicId and returns the network LossBundle at the block or error
func (k *ReputerLossKeeper) GetNetworkLossBundleAtBlock(ctx context.Context, topicId TopicId, block BlockHeight) (*types.ValueBundle, error) {
	key := collections.Join(topicId, block)
	lossBundle, err := k.networkLossBundles.Get(ctx, key)

	if errors.Is(err, collections.ErrNotFound) {
		return &types.ValueBundle{
			TopicId: topicId,
			ReputerRequestNonce: &types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{
					BlockHeight: 0,
				},
			},
			Reputer:                       "",
			ExtraData:                     nil,
			CombinedValue:                 alloraMath.ZeroDec(),
			InfererValues:                 nil,
			ForecasterValues:              nil,
			NaiveValue:                    alloraMath.ZeroDec(),
			OneOutInfererValues:           nil,
			OneOutForecasterValues:        nil,
			OneInForecasterValues:         nil,
			OneOutInfererForecasterValues: nil,
		}, nil
	} else if err != nil {
		return nil, errorsmod.Wrap(err, "error getting network loss bundle at block")
	}
	return &lossBundle, nil
}

// Returns the latest network loss bundle for a given topic id.
func (k *ReputerLossKeeper) GetLatestNetworkLossBundle(ctx context.Context, topicId TopicId) (*types.ValueBundle, error) {
	rng := collections.NewPrefixedPairRange[TopicId, BlockHeight](topicId).Descending()
	iter, err := k.networkLossBundles.Iterate(ctx, rng)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error iterating over network loss bundles")
	}
	defer iter.Close()

	if iter.Valid() {
		keyValue, err := iter.KeyValue()
		if err != nil {
			return nil, errorsmod.Wrap(err, "error getting key value")
		}
		return &keyValue.Value, nil
	}

	return nil, types.ErrNotFound
}

// ResetReputersIndividualSubmissionsForTopic resets the reputer individual submissions for a topic
func (k *ReputerLossKeeper) ResetReputersIndividualSubmissionsForTopic(ctx context.Context, topicId TopicId) error {
	reputerRange := collections.NewPrefixedPairRange[TopicId, ActorId](topicId)
	if err := k.lossBundles.Clear(ctx, reputerRange); err != nil {
		return errorsmod.Wrap(err, "error clearing reputer losses")
	}
	return nil
}

func (k *ReputerLossKeeper) PruneLossBundles(ctx context.Context, blockRange *collections.PairRange[uint64, int64]) error {
	return k.allLossBundles.Clear(ctx, blockRange)
}

func (k *ReputerLossKeeper) PruneNetworkLosses(ctx context.Context, blockRange *collections.PairRange[uint64, int64]) error {
	return k.networkLossBundles.Clear(ctx, blockRange)
}
