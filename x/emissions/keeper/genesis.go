package keeper

import (
	"context"

	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// InitGenesis initializes the module state from a genesis state.
func (k *Keeper) InitGenesis(ctx context.Context, data *types.GenesisState) error {
	// ensure the module account exists
	stakingModuleAccount := k.authKeeper.GetModuleAccount(ctx, types.AlloraStakingAccountName)
	k.authKeeper.SetModuleAccount(ctx, stakingModuleAccount)
	alloraRewardsModuleAccount := k.authKeeper.GetModuleAccount(ctx, types.AlloraRewardsAccountName)
	k.authKeeper.SetModuleAccount(ctx, alloraRewardsModuleAccount)
	alloraPendingRewardsModuleAccount := k.authKeeper.GetModuleAccount(ctx, types.AlloraPendingRewardForDelegatorAccountName)
	k.authKeeper.SetModuleAccount(ctx, alloraPendingRewardsModuleAccount)
	if err := k.SetParams(ctx, data.Params); err != nil {
		return err
	}
	if err := k.SetTotalStake(ctx, cosmosMath.ZeroInt()); err != nil {
		return err
	}
	// reserve topic ID 0 for future use
	if _, err := k.IncrementTopicId(ctx); err != nil {
		return err
	}

	// add core team to the whitelists
	if err := k.addCoreTeamToWhitelists(ctx, data.CoreTeamAddresses); err != nil {
		return err
	}

	// For mint module inflation rate calculation set the initial
	// "previous percentage of rewards that went to staked reputers" to 30%
	if err := k.SetPreviousPercentageRewardToStakedReputers(ctx, alloraMath.MustNewDecFromString("0.3")); err != nil {
		return err
	}

	return nil
}

// ExportGenesis exports the module state to a genesis state.
func (k *Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	nextTopicId, err := k.nextTopicId.Peek(ctx)
	if err != nil {
		return nil, err
	}

	topicsIter, err := k.topics.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	topics := make([]*types.TopicIdAndTopic, 0)
	for ; topicsIter.Valid(); topicsIter.Next() {
		keyValue, err := topicsIter.KeyValue()
		if err != nil {
			return nil, err
		}
		value := keyValue.Value
		topic := types.TopicIdAndTopic{
			TopicId: keyValue.Key,
			Topic:   &value,
		}
		topics = append(topics, &topic)
	}

	activeTopics := make([]uint64, 0)
	activeTopicsIter, err := k.activeTopics.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; activeTopicsIter.Valid(); activeTopicsIter.Next() {
		key, err := activeTopicsIter.Key()
		if err != nil {
			return nil, err
		}
		activeTopics = append(activeTopics, key)
	}

	churnableTopics := make([]uint64, 0)
	churnableTopicsIter, err := k.churnableTopics.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; churnableTopicsIter.Valid(); churnableTopicsIter.Next() {
		key, err := churnableTopicsIter.Key()
		if err != nil {
			return nil, err
		}
		churnableTopics = append(churnableTopics, key)
	}

	rewardableTopics := make([]uint64, 0)
	rewardableTopicsIter, err := k.rewardableTopics.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; rewardableTopicsIter.Valid(); rewardableTopicsIter.Next() {
		key, err := rewardableTopicsIter.Key()
		if err != nil {
			return nil, err
		}
		rewardableTopics = append(rewardableTopics, key)
	}

	topicWorkers := make([]*types.TopicAndActorId, 0)
	topicWorkersIter, err := k.topicWorkers.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; topicWorkersIter.Valid(); topicWorkersIter.Next() {
		key, err := topicWorkersIter.Key()
		if err != nil {
			return nil, err
		}
		topicIdAndActorId := types.TopicAndActorId{
			TopicId: key.K1(),
			ActorId: key.K2(),
		}
		topicWorkers = append(topicWorkers, &topicIdAndActorId)
	}

	topicReputers := make([]*types.TopicAndActorId, 0)
	topicReputersIter, err := k.topicReputers.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; topicReputersIter.Valid(); topicReputersIter.Next() {
		key, err := topicReputersIter.Key()
		if err != nil {
			return nil, err
		}
		topicIdAndActorId := types.TopicAndActorId{
			TopicId: key.K1(),
			ActorId: key.K2(),
		}
		topicReputers = append(topicReputers, &topicIdAndActorId)
	}

	topicRewardNonce := make([]*types.TopicIdAndBlockHeight, 0)
	topicRewardNonceIter, err := k.topicRewardNonce.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; topicRewardNonceIter.Valid(); topicRewardNonceIter.Next() {
		keyValue, err := topicRewardNonceIter.KeyValue()
		if err != nil {
			return nil, err
		}
		topicIdAndBlockHeight := types.TopicIdAndBlockHeight{
			TopicId:     keyValue.Key,
			BlockHeight: keyValue.Value,
		}
		topicRewardNonce = append(topicRewardNonce, &topicIdAndBlockHeight)
	}

	infererScoresByBlock := make([]*types.TopicIdBlockHeightScores, 0)
	infererScoresByBlockIter, err := k.infererScoresByBlock.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; infererScoresByBlockIter.Valid(); infererScoresByBlockIter.Next() {
		keyValue, err := infererScoresByBlockIter.KeyValue()
		if err != nil {
			return nil, err
		}
		value := keyValue.Value
		topicIdBlockHeightScores := types.TopicIdBlockHeightScores{
			TopicId:     keyValue.Key.K1(),
			BlockHeight: keyValue.Key.K2(),
			Scores:      &value,
		}
		infererScoresByBlock = append(infererScoresByBlock, &topicIdBlockHeightScores)
	}

	forecasterScoresByBlock := make([]*types.TopicIdBlockHeightScores, 0)
	forecasterScoresByBlockIter, err := k.forecasterScoresByBlock.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; forecasterScoresByBlockIter.Valid(); forecasterScoresByBlockIter.Next() {
		keyValue, err := forecasterScoresByBlockIter.KeyValue()
		if err != nil {
			return nil, err
		}
		value := keyValue.Value
		topicIdBlockHeightScores := types.TopicIdBlockHeightScores{
			TopicId:     keyValue.Key.K1(),
			BlockHeight: keyValue.Key.K2(),
			Scores:      &value,
		}
		forecasterScoresByBlock = append(forecasterScoresByBlock, &topicIdBlockHeightScores)
	}

	reputerScoresByBlock := make([]*types.TopicIdBlockHeightScores, 0)
	reputerScoresByBlockIter, err := k.reputerScoresByBlock.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; reputerScoresByBlockIter.Valid(); reputerScoresByBlockIter.Next() {
		keyValue, err := reputerScoresByBlockIter.KeyValue()
		if err != nil {
			return nil, err
		}
		value := keyValue.Value
		topicIdBlockHeightScores := types.TopicIdBlockHeightScores{
			TopicId:     keyValue.Key.K1(),
			BlockHeight: keyValue.Key.K2(),
			Scores:      &value,
		}
		reputerScoresByBlock = append(reputerScoresByBlock, &topicIdBlockHeightScores)
	}

	latestInfererScoresByWorker := make([]*types.TopicIdActorIdScore, 0)
	latestInfererScoresByWorkerIter, err := k.latestInfererScoresByWorker.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; latestInfererScoresByWorkerIter.Valid(); latestInfererScoresByWorkerIter.Next() {
		keyValue, err := latestInfererScoresByWorkerIter.KeyValue()
		if err != nil {
			return nil, err
		}
		value := keyValue.Value
		topicIdActorIdScore := types.TopicIdActorIdScore{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Score:   &value,
		}
		latestInfererScoresByWorker = append(latestInfererScoresByWorker, &topicIdActorIdScore)
	}

	latestForecasterScoresByWorker := make([]*types.TopicIdActorIdScore, 0)
	latestForecasterScoresByWorkerIter, err := k.latestForecasterScoresByWorker.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; latestForecasterScoresByWorkerIter.Valid(); latestForecasterScoresByWorkerIter.Next() {
		keyValue, err := latestForecasterScoresByWorkerIter.KeyValue()
		if err != nil {
			return nil, err
		}
		value := keyValue.Value
		topicIdActorIdScore := types.TopicIdActorIdScore{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Score:   &value,
		}
		latestForecasterScoresByWorker = append(latestForecasterScoresByWorker, &topicIdActorIdScore)
	}

	latestReputerScoresByReputer := make([]*types.TopicIdActorIdScore, 0)
	latestReputerScoresByReputerIter, err := k.latestReputerScoresByReputer.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; latestReputerScoresByReputerIter.Valid(); latestReputerScoresByReputerIter.Next() {
		keyValue, err := latestReputerScoresByReputerIter.KeyValue()
		if err != nil {
			return nil, err
		}
		value := keyValue.Value
		topicIdActorIdScore := types.TopicIdActorIdScore{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Score:   &value,
		}
		latestReputerScoresByReputer = append(latestReputerScoresByReputer, &topicIdActorIdScore)
	}

	reputerListeningCoefficient := make([]*types.TopicIdActorIdListeningCoefficient, 0)
	reputerListeningCoefficientIter, err := k.reputerListeningCoefficient.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; reputerListeningCoefficientIter.Valid(); reputerListeningCoefficientIter.Next() {
		keyValue, err := reputerListeningCoefficientIter.KeyValue()
		if err != nil {
			return nil, err
		}
		value := keyValue.Value
		topicIdActorIdListeningCoefficient := types.TopicIdActorIdListeningCoefficient{
			TopicId:              keyValue.Key.K1(),
			ActorId:              keyValue.Key.K2(),
			ListeningCoefficient: &value,
		}
		reputerListeningCoefficient = append(reputerListeningCoefficient, &topicIdActorIdListeningCoefficient)
	}

	previousReputerRewardFraction := make([]*types.TopicIdActorIdDec, 0)
	previousReputerRewardFractionIter, err := k.previousReputerRewardFraction.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; previousReputerRewardFractionIter.Valid(); previousReputerRewardFractionIter.Next() {
		keyValue, err := previousReputerRewardFractionIter.KeyValue()
		if err != nil {
			return nil, err
		}
		topicIdActorIdDec := types.TopicIdActorIdDec{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Dec:     keyValue.Value,
		}
		previousReputerRewardFraction = append(previousReputerRewardFraction, &topicIdActorIdDec)
	}

	previousInferenceRewardFraction := make([]*types.TopicIdActorIdDec, 0)
	previousInferenceRewardFractionIter, err := k.previousInferenceRewardFraction.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; previousInferenceRewardFractionIter.Valid(); previousInferenceRewardFractionIter.Next() {
		keyValue, err := previousInferenceRewardFractionIter.KeyValue()
		if err != nil {
			return nil, err
		}
		topicIdActorIdDec := types.TopicIdActorIdDec{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Dec:     keyValue.Value,
		}
		previousInferenceRewardFraction = append(previousInferenceRewardFraction, &topicIdActorIdDec)
	}

	previousForecastRewardFraction := make([]*types.TopicIdActorIdDec, 0)
	previousForecastRewardFractionIter, err := k.previousForecastRewardFraction.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; previousForecastRewardFractionIter.Valid(); previousForecastRewardFractionIter.Next() {
		keyValue, err := previousForecastRewardFractionIter.KeyValue()
		if err != nil {
			return nil, err
		}
		topicIdActorIdDec := types.TopicIdActorIdDec{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Dec:     keyValue.Value,
		}
		previousForecastRewardFraction = append(previousForecastRewardFraction, &topicIdActorIdDec)
	}

	totalStake, err := k.totalStake.Get(ctx)
	if err != nil {
		return nil, err
	}

	// Fill in the values from keeper.go

	// topicStake
	topicStake := make([]*types.TopicIdAndInt, 0)
	topicStakeIter, err := k.topicStake.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; topicStakeIter.Valid(); topicStakeIter.Next() {
		keyValue, err := topicStakeIter.KeyValue()
		if err != nil {
			return nil, err
		}
		topicIdAndInt := types.TopicIdAndInt{
			TopicId: keyValue.Key,
			Int:     keyValue.Value,
		}
		topicStake = append(topicStake, &topicIdAndInt)
	}

	// stakeReputerAuthority
	stakeReputerAuthority := make([]*types.TopicIdActorIdInt, 0)
	stakeReputerAuthorityIter, err := k.stakeReputerAuthority.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; stakeReputerAuthorityIter.Valid(); stakeReputerAuthorityIter.Next() {
		keyValue, err := stakeReputerAuthorityIter.KeyValue()
		if err != nil {
			return nil, err
		}
		topicIdActorIdInt := types.TopicIdActorIdInt{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Int:     keyValue.Value,
		}
		stakeReputerAuthority = append(stakeReputerAuthority, &topicIdActorIdInt)
	}

	// stakeSumFromDelegator
	stakeSumFromDelegator := make([]*types.TopicIdActorIdInt, 0)
	stakeSumFromDelegatorIter, err := k.stakeSumFromDelegator.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; stakeSumFromDelegatorIter.Valid(); stakeSumFromDelegatorIter.Next() {
		keyValue, err := stakeSumFromDelegatorIter.KeyValue()
		if err != nil {
			return nil, err
		}
		topicIdActorIdInt := types.TopicIdActorIdInt{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Int:     keyValue.Value,
		}
		stakeSumFromDelegator = append(stakeSumFromDelegator, &topicIdActorIdInt)
	}

	// delegatedStakes
	delegatedStakes := make([]*types.TopicIdDelegatorReputerDelegatorInfo, 0)
	delegatedStakesIter, err := k.delegatedStakes.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; delegatedStakesIter.Valid(); delegatedStakesIter.Next() {
		keyValue, err := delegatedStakesIter.KeyValue()
		if err != nil {
			return nil, err
		}
		value := keyValue.Value
		topicIdDelegatorReputerDelegatorInfo := types.TopicIdDelegatorReputerDelegatorInfo{
			TopicId:       keyValue.Key.K1(),
			Delegator:     keyValue.Key.K2(),
			Reputer:       keyValue.Key.K3(),
			DelegatorInfo: &value,
		}
		delegatedStakes = append(delegatedStakes, &topicIdDelegatorReputerDelegatorInfo)
	}

	// stakeFromDelegatorsUponReputer
	stakeFromDelegatorsUponReputer := make([]*types.TopicIdActorIdInt, 0)
	stakeFromDelegatorsUponReputerIter, err := k.stakeFromDelegatorsUponReputer.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; stakeFromDelegatorsUponReputerIter.Valid(); stakeFromDelegatorsUponReputerIter.Next() {
		keyValue, err := stakeFromDelegatorsUponReputerIter.KeyValue()
		if err != nil {
			return nil, err
		}
		topicIdActorIdInt := types.TopicIdActorIdInt{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Int:     keyValue.Value,
		}
		stakeFromDelegatorsUponReputer = append(stakeFromDelegatorsUponReputer, &topicIdActorIdInt)
	}

	// delegateRewardPerShare
	delegateRewardPerShare := make([]*types.TopicIdActorIdDec, 0)
	delegateRewardPerShareIter, err := k.delegateRewardPerShare.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; delegateRewardPerShareIter.Valid(); delegateRewardPerShareIter.Next() {
		keyValue, err := delegateRewardPerShareIter.KeyValue()
		if err != nil {
			return nil, err
		}
		topicIdActorIdDec := types.TopicIdActorIdDec{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Dec:     keyValue.Value,
		}
		delegateRewardPerShare = append(delegateRewardPerShare, &topicIdActorIdDec)
	}

	// stakeRemovalsByBlock
	stakeRemovalsByBlock := make([]*types.BlockHeightTopicIdReputerStakeRemovalInfo, 0)
	stakeRemovalsByBlockIter, err := k.stakeRemovalsByBlock.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; stakeRemovalsByBlockIter.Valid(); stakeRemovalsByBlockIter.Next() {
		keyValue, err := stakeRemovalsByBlockIter.KeyValue()
		if err != nil {
			return nil, err
		}
		value := keyValue.Value
		blockHeightTopicIdReputerStakeRemovalInfo := types.BlockHeightTopicIdReputerStakeRemovalInfo{
			BlockHeight:      keyValue.Key.K1(),
			TopicId:          keyValue.Key.K2(),
			StakeRemovalInfo: &value,
		}
		stakeRemovalsByBlock = append(stakeRemovalsByBlock, &blockHeightTopicIdReputerStakeRemovalInfo)
	}

	// stakeRemovalsByActor
	stakeRemovalsByActor := make([]*types.ActorIdTopicIdBlockHeight, 0)
	stakeRemovalsByActorIter, err := k.stakeRemovalsByActor.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; stakeRemovalsByActorIter.Valid(); stakeRemovalsByActorIter.Next() {
		key, err := stakeRemovalsByActorIter.Key()
		if err != nil {
			return nil, err
		}
		actorIdTopicIdBlockHeight := types.ActorIdTopicIdBlockHeight{
			ActorId:     key.K1(),
			TopicId:     key.K2(),
			BlockHeight: key.K3(),
		}
		stakeRemovalsByActor = append(stakeRemovalsByActor, &actorIdTopicIdBlockHeight)
	}

	// delegateStakeRemovalsByBlock
	delegateStakeRemovalsByBlock := make([]*types.BlockHeightTopicIdDelegatorReputerDelegateStakeRemovalInfo, 0)
	delegateStakeRemovalsByBlockIter, err := k.delegateStakeRemovalsByBlock.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; delegateStakeRemovalsByBlockIter.Valid(); delegateStakeRemovalsByBlockIter.Next() {
		keyValue, err := delegateStakeRemovalsByBlockIter.KeyValue()
		if err != nil {
			return nil, err
		}
		value := keyValue.Value
		blockHeightTopicIdDelegatorReputerDelegateStakeRemovalInfo := types.BlockHeightTopicIdDelegatorReputerDelegateStakeRemovalInfo{
			BlockHeight:              keyValue.Key.K1(),
			TopicId:                  keyValue.Key.K2(),
			DelegateStakeRemovalInfo: &value,
		}
		delegateStakeRemovalsByBlock = append(delegateStakeRemovalsByBlock, &blockHeightTopicIdDelegatorReputerDelegateStakeRemovalInfo)
	}

	// delegateStakeRemovalsByActor
	delegateStakeRemovalsByActor := make([]*types.DelegatorReputerTopicIdBlockHeight, 0)
	delegateStakeRemovalsByActorIter, err := k.delegateStakeRemovalsByActor.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; delegateStakeRemovalsByActorIter.Valid(); delegateStakeRemovalsByActorIter.Next() {
		key, err := delegateStakeRemovalsByActorIter.Key()
		if err != nil {
			return nil, err
		}
		delegatorReputerTopicIdBlockHeight := types.DelegatorReputerTopicIdBlockHeight{
			Delegator:   key.K1(),
			Reputer:     key.K2(),
			TopicId:     key.K3(),
			BlockHeight: key.K4(),
		}
		delegateStakeRemovalsByActor = append(delegateStakeRemovalsByActor, &delegatorReputerTopicIdBlockHeight)
	}

	// inferences
	inferences := make([]*types.TopicIdActorIdInference, 0)
	inferencesIter, err := k.inferences.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; inferencesIter.Valid(); inferencesIter.Next() {
		keyValue, err := inferencesIter.KeyValue()
		if err != nil {
			return nil, err
		}
		value := keyValue.Value
		topicIdActorIdInference := types.TopicIdActorIdInference{
			TopicId:   keyValue.Key.K1(),
			ActorId:   keyValue.Key.K2(),
			Inference: &value,
		}
		inferences = append(inferences, &topicIdActorIdInference)
	}

	// forecasts
	forecasts := make([]*types.TopicIdActorIdForecast, 0)
	forecastsIter, err := k.forecasts.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; forecastsIter.Valid(); forecastsIter.Next() {
		keyValue, err := forecastsIter.KeyValue()
		if err != nil {
			return nil, err
		}
		value := keyValue.Value
		topicIdActorIdForecast := types.TopicIdActorIdForecast{
			TopicId:  keyValue.Key.K1(),
			ActorId:  keyValue.Key.K2(),
			Forecast: &value,
		}
		forecasts = append(forecasts, &topicIdActorIdForecast)
	}

	// workers
	workers := make([]*types.LibP2PKeyAndOffchainNode, 0)
	workersIter, err := k.workers.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; workersIter.Valid(); workersIter.Next() {
		keyValue, err := workersIter.KeyValue()
		if err != nil {
			return nil, err
		}
		value := keyValue.Value
		libP2PKeyAndOffchainNode := types.LibP2PKeyAndOffchainNode{
			LibP2PKey:    keyValue.Key,
			OffchainNode: &value,
		}
		workers = append(workers, &libP2PKeyAndOffchainNode)
	}

	// reputers
	reputers := make([]*types.LibP2PKeyAndOffchainNode, 0)
	reputersIter, err := k.reputers.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; reputersIter.Valid(); reputersIter.Next() {
		keyValue, err := reputersIter.KeyValue()
		if err != nil {
			return nil, err
		}
		libP2PKeyAndOffchainNode := types.LibP2PKeyAndOffchainNode{
			LibP2PKey:    keyValue.Key,
			OffchainNode: &keyValue.Value,
		}
		reputers = append(reputers, &libP2PKeyAndOffchainNode)
	}

	// topicFeeRevenue
	topicFeeRevenue := make([]*types.TopicIdAndInt, 0)
	topicFeeRevenueIter, err := k.topicFeeRevenue.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; topicFeeRevenueIter.Valid(); topicFeeRevenueIter.Next() {
		keyValue, err := topicFeeRevenueIter.KeyValue()
		if err != nil {
			return nil, err
		}
		topicIdAndInt := types.TopicIdAndInt{
			TopicId: keyValue.Key,
			Int:     keyValue.Value,
		}
		topicFeeRevenue = append(topicFeeRevenue, &topicIdAndInt)
	}

	// previousTopicWeight
	previousTopicWeight := make([]*types.TopicIdAndDec, 0)
	previousTopicWeightIter, err := k.previousTopicWeight.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; previousTopicWeightIter.Valid(); previousTopicWeightIter.Next() {
		keyValue, err := previousTopicWeightIter.KeyValue()
		if err != nil {
			return nil, err
		}
		topicIdAndDec := types.TopicIdAndDec{
			TopicId: keyValue.Key,
			Dec:     keyValue.Value,
		}
		previousTopicWeight = append(previousTopicWeight, &topicIdAndDec)
	}

	// allInferences
	allInferences := make([]*types.TopicIdBlockHeightInferences, 0)
	allInferencesIter, err := k.allInferences.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; allInferencesIter.Valid(); allInferencesIter.Next() {
		keyValue, err := allInferencesIter.KeyValue()
		if err != nil {
			return nil, err
		}
		value := keyValue.Value
		topicIdBlockHeightInferences := types.TopicIdBlockHeightInferences{
			TopicId:     keyValue.Key.K1(),
			BlockHeight: keyValue.Key.K2(),
			Inferences:  &value,
		}
		allInferences = append(allInferences, &topicIdBlockHeightInferences)
	}

	// allForecasts
	allForecasts := make([]*types.TopicIdBlockHeightForecasts, 0)
	allForecastsIter, err := k.allForecasts.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; allForecastsIter.Valid(); allForecastsIter.Next() {
		keyValue, err := allForecastsIter.KeyValue()
		if err != nil {
			return nil, err
		}
		value := keyValue.Value
		topicIdBlockHeightForecasts := types.TopicIdBlockHeightForecasts{
			TopicId:     keyValue.Key.K1(),
			BlockHeight: keyValue.Key.K2(),
			Forecasts:   &value,
		}
		allForecasts = append(allForecasts, &topicIdBlockHeightForecasts)
	}

	// allLossBundles
	allLossBundles := make([]*types.TopicIdBlockHeightReputerValueBundles, 0)
	allLossBundlesIter, err := k.allLossBundles.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; allLossBundlesIter.Valid(); allLossBundlesIter.Next() {
		keyValue, err := allLossBundlesIter.KeyValue()
		if err != nil {
			return nil, err
		}
		value := keyValue.Value
		topicIdBlockHeightValueBundles := types.TopicIdBlockHeightReputerValueBundles{
			TopicId:             keyValue.Key.K1(),
			BlockHeight:         keyValue.Key.K2(),
			ReputerValueBundles: &value,
		}
		allLossBundles = append(allLossBundles, &topicIdBlockHeightValueBundles)
	}

	// networkLossBundles
	networkLossBundles := make([]*types.TopicIdBlockHeightValueBundles, 0)
	networkLossBundlesIter, err := k.networkLossBundles.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; networkLossBundlesIter.Valid(); networkLossBundlesIter.Next() {
		keyValue, err := networkLossBundlesIter.KeyValue()
		if err != nil {
			return nil, err
		}
		value := keyValue.Value
		topicIdBlockHeightValueBundles := types.TopicIdBlockHeightValueBundles{
			TopicId:     keyValue.Key.K1(),
			BlockHeight: keyValue.Key.K2(),
			ValueBundle: &value,
		}
		networkLossBundles = append(networkLossBundles, &topicIdBlockHeightValueBundles)
	}

	// previousPercentageRewardToStakedReputers
	previousPercentageRewardToStakedReputers, err := k.previousPercentageRewardToStakedReputers.Get(ctx)
	if err != nil {
		return nil, err
	}

	// unfulfilledWorkerNonces
	unfulfilledWorkerNonces := make([]*types.TopicIdAndNonces, 0)
	unfulfilledWorkerNoncesIter, err := k.unfulfilledWorkerNonces.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; unfulfilledWorkerNoncesIter.Valid(); unfulfilledWorkerNoncesIter.Next() {
		keyValue, err := unfulfilledWorkerNoncesIter.KeyValue()
		if err != nil {
			return nil, err
		}
		topicIdAndNonces := types.TopicIdAndNonces{
			TopicId: keyValue.Key,
			Nonces:  &keyValue.Value,
		}
		unfulfilledWorkerNonces = append(unfulfilledWorkerNonces, &topicIdAndNonces)
	}

	// unfulfilledReputerNonces
	unfulfilledReputerNonces := make([]*types.TopicIdAndReputerRequestNonces, 0)
	unfulfilledReputerNoncesIter, err := k.unfulfilledReputerNonces.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; unfulfilledReputerNoncesIter.Valid(); unfulfilledReputerNoncesIter.Next() {
		keyValue, err := unfulfilledReputerNoncesIter.KeyValue()
		if err != nil {
			return nil, err
		}
		value := keyValue.Value
		topicIdAndReputerRequestNonces := types.TopicIdAndReputerRequestNonces{
			TopicId:              keyValue.Key,
			ReputerRequestNonces: &value,
		}
		unfulfilledReputerNonces = append(unfulfilledReputerNonces, &topicIdAndReputerRequestNonces)
	}

	latestInfererNetworkRegrets := make([]*types.TopicIdActorIdTimeStampedValue, 0)
	latestInfererNetworkRegretsIter, err := k.latestInfererNetworkRegrets.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; latestInfererNetworkRegretsIter.Valid(); latestInfererNetworkRegretsIter.Next() {
		keyValue, err := latestInfererNetworkRegretsIter.KeyValue()
		if err != nil {
			return nil, err
		}
		topicIdActorIdTimeStampedValue := types.TopicIdActorIdTimeStampedValue{
			TopicId:          keyValue.Key.K1(),
			ActorId:          keyValue.Key.K2(),
			TimestampedValue: &keyValue.Value,
		}
		latestInfererNetworkRegrets = append(latestInfererNetworkRegrets, &topicIdActorIdTimeStampedValue)
	}

	latestForecasterNetworkRegrets := make([]*types.TopicIdActorIdTimeStampedValue, 0)
	latestForecasterNetworkRegretsIter, err := k.latestForecasterNetworkRegrets.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; latestForecasterNetworkRegretsIter.Valid(); latestForecasterNetworkRegretsIter.Next() {
		keyValue, err := latestForecasterNetworkRegretsIter.KeyValue()
		if err != nil {
			return nil, err
		}
		topicIdActorIdTimeStampedValue := types.TopicIdActorIdTimeStampedValue{
			TopicId:          keyValue.Key.K1(),
			ActorId:          keyValue.Key.K2(),
			TimestampedValue: &keyValue.Value,
		}
		latestForecasterNetworkRegrets = append(latestForecasterNetworkRegrets, &topicIdActorIdTimeStampedValue)
	}

	latestOneInForecasterNetworkRegrets := make([]*types.TopicIdActorIdActorIdTimeStampedValue, 0)
	latestOneInForecasterNetworkRegretsIter, err := k.latestOneInForecasterNetworkRegrets.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; latestOneInForecasterNetworkRegretsIter.Valid(); latestOneInForecasterNetworkRegretsIter.Next() {
		keyValue, err := latestOneInForecasterNetworkRegretsIter.KeyValue()
		if err != nil {
			return nil, err
		}
		topicIdActorIdActorIdTimeStampedValue := types.TopicIdActorIdActorIdTimeStampedValue{
			TopicId:          keyValue.Key.K1(),
			ActorId1:         keyValue.Key.K2(),
			ActorId2:         keyValue.Key.K3(),
			TimestampedValue: &keyValue.Value,
		}
		latestOneInForecasterNetworkRegrets = append(latestOneInForecasterNetworkRegrets, &topicIdActorIdActorIdTimeStampedValue)
	}

	latestOneInForecasterSelfNetworkRegrets := make([]*types.TopicIdActorIdTimeStampedValue, 0)
	latestOneInForecasterSelfNetworkRegretsIter, err := k.latestOneInForecasterSelfNetworkRegrets.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; latestOneInForecasterSelfNetworkRegretsIter.Valid(); latestOneInForecasterSelfNetworkRegretsIter.Next() {
		keyValue, err := latestOneInForecasterSelfNetworkRegretsIter.KeyValue()
		if err != nil {
			return nil, err
		}
		topicIdActorIdTimeStampedValue := types.TopicIdActorIdTimeStampedValue{
			TopicId:          keyValue.Key.K1(),
			ActorId:          keyValue.Key.K2(),
			TimestampedValue: &keyValue.Value,
		}
		latestOneInForecasterSelfNetworkRegrets = append(latestOneInForecasterSelfNetworkRegrets, &topicIdActorIdTimeStampedValue)
	}

	coreTeamAddresses := make([]string, 0)
	coreTeamAddressesIter, err := k.whitelistAdmins.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; coreTeamAddressesIter.Valid(); coreTeamAddressesIter.Next() {
		key, err := coreTeamAddressesIter.Key()
		if err != nil {
			return nil, err
		}
		coreTeamAddresses = append(coreTeamAddresses, key)
	}

	topicLastWorkerCommit := make([]*types.TopicIdTimestampedActorNonce, 0)
	topicLastWorkerCommitIter, err := k.topicLastWorkerCommit.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; topicLastWorkerCommitIter.Valid(); topicLastWorkerCommitIter.Next() {
		keyValue, err := topicLastWorkerCommitIter.KeyValue()
		if err != nil {
			return nil, err
		}
		topicIdTimestampedActorNonce := types.TopicIdTimestampedActorNonce{
			TopicId:               keyValue.Key,
			TimestampedActorNonce: &keyValue.Value,
		}
		topicLastWorkerCommit = append(topicLastWorkerCommit, &topicIdTimestampedActorNonce)
	}

	topicLastReputerCommit := make([]*types.TopicIdTimestampedActorNonce, 0)
	topicLastReputerCommitIter, err := k.topicLastReputerCommit.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; topicLastReputerCommitIter.Valid(); topicLastReputerCommitIter.Next() {
		keyValue, err := topicLastReputerCommitIter.KeyValue()
		if err != nil {
			return nil, err
		}
		topicIdTimestampedActorNonce := types.TopicIdTimestampedActorNonce{
			TopicId:               keyValue.Key,
			TimestampedActorNonce: &keyValue.Value,
		}
		topicLastReputerCommit = append(topicLastReputerCommit, &topicIdTimestampedActorNonce)
	}

	topicLastWorkerPayload := make([]*types.TopicIdTimestampedActorNonce, 0)
	topicLastWorkerPayloadIter, err := k.topicLastWorkerPayload.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; topicLastWorkerPayloadIter.Valid(); topicLastWorkerPayloadIter.Next() {
		keyValue, err := topicLastWorkerPayloadIter.KeyValue()
		if err != nil {
			return nil, err
		}
		topicIdTimestampedActorNonce := types.TopicIdTimestampedActorNonce{
			TopicId:               keyValue.Key,
			TimestampedActorNonce: &keyValue.Value,
		}
		topicLastWorkerPayload = append(topicLastWorkerPayload, &topicIdTimestampedActorNonce)
	}

	topicLastReputerPayload := make([]*types.TopicIdTimestampedActorNonce, 0)
	topicLastReputerPayloadIter, err := k.topicLastReputerPayload.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; topicLastReputerPayloadIter.Valid(); topicLastReputerPayloadIter.Next() {
		keyValue, err := topicLastReputerPayloadIter.KeyValue()
		if err != nil {
			return nil, err
		}
		topicIdTimestampedActorNonce := types.TopicIdTimestampedActorNonce{
			TopicId:               keyValue.Key,
			TimestampedActorNonce: &keyValue.Value,
		}
		topicLastReputerPayload = append(topicLastReputerPayload, &topicIdTimestampedActorNonce)
	}

	return &types.GenesisState{
		Params:                                   moduleParams,
		NextTopicId:                              nextTopicId,
		Topics:                                   topics,
		ActiveTopics:                             activeTopics,
		ChurnableTopics:                          churnableTopics,
		RewardableTopics:                         rewardableTopics,
		TopicWorkers:                             topicWorkers,
		TopicReputers:                            topicReputers,
		TopicRewardNonce:                         topicRewardNonce,
		InfererScoresByBlock:                     infererScoresByBlock,
		ForecasterScoresByBlock:                  forecasterScoresByBlock,
		ReputerScoresByBlock:                     reputerScoresByBlock,
		LatestInfererScoresByWorker:              latestInfererScoresByWorker,
		LatestForecasterScoresByWorker:           latestForecasterScoresByWorker,
		LatestReputerScoresByReputer:             latestReputerScoresByReputer,
		ReputerListeningCoefficient:              reputerListeningCoefficient,
		PreviousReputerRewardFraction:            previousReputerRewardFraction,
		PreviousInferenceRewardFraction:          previousInferenceRewardFraction,
		PreviousForecastRewardFraction:           previousForecastRewardFraction,
		TotalStake:                               totalStake,
		TopicStake:                               topicStake,
		StakeReputerAuthority:                    stakeReputerAuthority,
		StakeSumFromDelegator:                    stakeSumFromDelegator,
		DelegatedStakes:                          delegatedStakes,
		StakeFromDelegatorsUponReputer:           stakeFromDelegatorsUponReputer,
		DelegateRewardPerShare:                   delegateRewardPerShare,
		StakeRemovalsByBlock:                     stakeRemovalsByBlock,
		StakeRemovalsByActor:                     stakeRemovalsByActor,
		DelegateStakeRemovalsByBlock:             delegateStakeRemovalsByBlock,
		DelegateStakeRemovalsByActor:             delegateStakeRemovalsByActor,
		Inferences:                               inferences,
		Forecasts:                                forecasts,
		Workers:                                  workers,
		Reputers:                                 reputers,
		TopicFeeRevenue:                          topicFeeRevenue,
		PreviousTopicWeight:                      previousTopicWeight,
		AllInferences:                            allInferences,
		AllForecasts:                             allForecasts,
		AllLossBundles:                           allLossBundles,
		NetworkLossBundles:                       networkLossBundles,
		PreviousPercentageRewardToStakedReputers: previousPercentageRewardToStakedReputers,
		UnfulfilledWorkerNonces:                  unfulfilledWorkerNonces,
		UnfulfilledReputerNonces:                 unfulfilledReputerNonces,
		LatestInfererNetworkRegrets:              latestInfererNetworkRegrets,
		LatestForecasterNetworkRegrets:           latestForecasterNetworkRegrets,
		LatestOneInForecasterNetworkRegrets:      latestOneInForecasterNetworkRegrets,
		LatestOneInForecasterSelfNetworkRegrets:  latestOneInForecasterSelfNetworkRegrets,
		CoreTeamAddresses:                        coreTeamAddresses,
		TopicLastWorkerCommit:                    topicLastWorkerCommit,
		TopicLastReputerCommit:                   topicLastReputerCommit,
		TopicLastWorkerPayload:                   topicLastWorkerPayload,
		TopicLastReputerPayload:                  topicLastReputerPayload,
	}, nil
}

func (k *Keeper) addCoreTeamToWhitelists(ctx context.Context, coreTeamAddresses []string) error {
	for _, addr := range coreTeamAddresses {
		k.AddWhitelistAdmin(ctx, addr)
	}
	return nil
}
