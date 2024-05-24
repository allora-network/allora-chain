package msgserver

import (
	"context"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (ms msgServer) UpdateParams(ctx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if err := ms.k.ValidateStringIsBech32(msg.Sender); err != nil {
		return nil, err
	}
	isAdmin, err := ms.k.IsWhitelistAdmin(ctx, msg.Sender)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, types.ErrNotWhitelistAdmin
	}
	existingParams, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	// every option is a repeated field, so we interpret an empty array as "make no change"
	newParams := msg.Params
	if len(newParams.Version) == 1 {
		existingParams.Version = newParams.Version[0]
	}
	if len(newParams.MinTopicWeight) == 1 {
		existingParams.MinTopicWeight = newParams.MinTopicWeight[0]
	}
	if len(newParams.MaxTopicsPerBlock) == 1 {
		existingParams.MaxTopicsPerBlock = newParams.MaxTopicsPerBlock[0]
	}
	if len(newParams.MaxMissingInferencePercent) == 1 {
		existingParams.MaxMissingInferencePercent = newParams.MaxMissingInferencePercent[0]
	}
	if len(newParams.RequiredMinimumStake) == 1 {
		existingParams.RequiredMinimumStake = newParams.RequiredMinimumStake[0]
	}
	if len(newParams.RemoveStakeDelayWindow) == 1 {
		existingParams.RemoveStakeDelayWindow = newParams.RemoveStakeDelayWindow[0]
	}
	if len(newParams.MinEpochLength) == 1 {
		existingParams.MinEpochLength = newParams.MinEpochLength[0]
	}
	if len(newParams.BetaEntropy) == 1 {
		existingParams.BetaEntropy = newParams.BetaEntropy[0]
	}
	if len(newParams.LearningRate) == 1 {
		existingParams.LearningRate = newParams.LearningRate[0]
	}
	if len(newParams.GradientDescentMaxIters) == 1 {
		existingParams.GradientDescentMaxIters = newParams.GradientDescentMaxIters[0]
	}
	if len(newParams.MaxGradientThreshold) == 1 {
		existingParams.MaxGradientThreshold = newParams.MaxGradientThreshold[0]
	}
	if len(newParams.MinStakeFraction) == 1 {
		existingParams.MinStakeFraction = newParams.MinStakeFraction[0]
	}
	if len(newParams.Epsilon) == 1 {
		existingParams.Epsilon = newParams.Epsilon[0]
	}
	if len(newParams.PInferenceSynthesis) == 1 {
		existingParams.PInferenceSynthesis = newParams.PInferenceSynthesis[0]
	}
	if len(newParams.PRewardSpread) == 1 {
		existingParams.PRewardSpread = newParams.PRewardSpread[0]
	}
	if len(newParams.AlphaRegret) == 1 {
		existingParams.AlphaRegret = newParams.AlphaRegret[0]
	}
	if len(newParams.MaxUnfulfilledWorkerRequests) == 1 {
		existingParams.MaxUnfulfilledWorkerRequests = newParams.MaxUnfulfilledWorkerRequests[0]
	}
	if len(newParams.MaxUnfulfilledReputerRequests) == 1 {
		existingParams.MaxUnfulfilledReputerRequests = newParams.MaxUnfulfilledReputerRequests[0]
	}
	if len(newParams.TopicRewardStakeImportance) == 1 {
		existingParams.TopicRewardStakeImportance = newParams.TopicRewardStakeImportance[0]
	}
	if len(newParams.TopicRewardFeeRevenueImportance) == 1 {
		existingParams.TopicRewardFeeRevenueImportance = newParams.TopicRewardFeeRevenueImportance[0]
	}
	if len(newParams.TopicRewardAlpha) == 1 {
		existingParams.TopicRewardAlpha = newParams.TopicRewardAlpha[0]
	}
	if len(newParams.TaskRewardAlpha) == 1 {
		existingParams.TaskRewardAlpha = newParams.TaskRewardAlpha[0]
	}
	if len(newParams.ValidatorsVsAlloraPercentReward) == 1 {
		existingParams.ValidatorsVsAlloraPercentReward = newParams.ValidatorsVsAlloraPercentReward[0]
	}
	if len(newParams.MaxSamplesToScaleScores) == 1 {
		existingParams.MaxSamplesToScaleScores = newParams.MaxSamplesToScaleScores[0]
	}
	if len(newParams.MaxTopWorkersToReward) == 1 {
		existingParams.MaxTopWorkersToReward = newParams.MaxTopWorkersToReward[0]
	}
	if len(newParams.MaxTopReputersToReward) == 1 {
		existingParams.MaxTopReputersToReward = newParams.MaxTopReputersToReward[0]
	}
	if len(newParams.CreateTopicFee) == 1 {
		existingParams.CreateTopicFee = newParams.CreateTopicFee[0]
	}
	if len(newParams.SigmoidA) == 1 {
		existingParams.SigmoidA = newParams.SigmoidA[0]
	}
	if len(newParams.SigmoidB) == 1 {
		existingParams.SigmoidB = newParams.SigmoidB[0]
	}
	if len(newParams.MaxRetriesToFulfilNoncesWorker) == 1 {
		existingParams.MaxRetriesToFulfilNoncesWorker = newParams.MaxRetriesToFulfilNoncesWorker[0]
	}
	if len(newParams.MaxRetriesToFulfilNoncesReputer) == 1 {
		existingParams.MaxRetriesToFulfilNoncesReputer = newParams.MaxRetriesToFulfilNoncesReputer[0]
	}
	if len(newParams.TopicPageLimit) == 1 {
		existingParams.TopicPageLimit = newParams.TopicPageLimit[0]
	}
	if len(newParams.MaxTopicPages) == 1 {
		existingParams.MaxTopicPages = newParams.MaxTopicPages[0]
	}
	if len(newParams.RegistrationFee) == 1 {
		existingParams.RegistrationFee = newParams.RegistrationFee[0]
	}
	if len(newParams.DefaultLimit) == 1 {
		existingParams.DefaultLimit = newParams.DefaultLimit[0]
	}
	if len(newParams.MaxLimit) == 1 {
		existingParams.MaxLimit = newParams.MaxLimit[0]
	}
	if len(newParams.MinEpochLengthRecordLimit) == 1 {
		existingParams.MinEpochLengthRecordLimit = newParams.MinEpochLengthRecordLimit[0]
	}
	if len(newParams.MaxSerializedMsgLength) == 1 {
		existingParams.MaxSerializedMsgLength = newParams.MaxSerializedMsgLength[0]
	}
	if len(newParams.BlocksPerMonth) == 1 {
		existingParams.BlocksPerMonth = newParams.BlocksPerMonth[0]
	}
	err = ms.k.SetParams(ctx, existingParams)
	if err != nil {
		return nil, err
	}
	return &types.MsgUpdateParamsResponse{}, nil
}
