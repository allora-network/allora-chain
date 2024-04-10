package msgserver

import (
	"context"
	"strconv"

	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (ms msgServer) UpdateParams(ctx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	isAdmin, err := ms.k.IsWhitelistAdmin(ctx, sender)
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
	if len(newParams.RewardCadence) == 1 {
		existingParams.RewardCadence = newParams.RewardCadence[0]
	}
	if len(newParams.MinTopicUnmetDemand) == 1 {
		existingParams.MinTopicUnmetDemand = newParams.MinTopicUnmetDemand[0]
	}
	if len(newParams.MaxTopicsPerBlock) == 1 {
		existingParams.MaxTopicsPerBlock = newParams.MaxTopicsPerBlock[0]
	}
	if len(newParams.MinRequestUnmetDemand) == 1 {
		existingParams.MinRequestUnmetDemand = newParams.MinRequestUnmetDemand[0]
	}
	if len(newParams.MaxMissingInferencePercent) == 1 {
		maxMissingInferencePercent, err := strconv.ParseFloat(newParams.MaxMissingInferencePercent[0], 64)
		if err != nil {
			return nil, err
		}
		existingParams.MaxMissingInferencePercent = maxMissingInferencePercent
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
	if len(newParams.MaxInferenceRequestValidity) == 1 {
		existingParams.MaxInferenceRequestValidity = newParams.MaxInferenceRequestValidity[0]
	}
	if len(newParams.MaxRequestCadence) == 1 {
		existingParams.MaxRequestCadence = newParams.MaxRequestCadence[0]
	}
	if len(newParams.PercentRewardsReputersWorkers) == 1 {
		percentRewardsReputersWorkers, err := strconv.ParseFloat(newParams.PercentRewardsReputersWorkers[0], 64)
		if err != nil {
			return nil, err
		}
		existingParams.PercentRewardsReputersWorkers = percentRewardsReputersWorkers
	}
	if len(newParams.Sharpness) == 1 {
		sharpness, err := strconv.ParseFloat(newParams.Sharpness[0], 64)
		if err != nil {
			return nil, err
		}
		existingParams.Sharpness = sharpness
	}
	if len(newParams.BetaEntropy) == 1 {
		betaEntropy, err := strconv.ParseFloat(newParams.BetaEntropy[0], 64)
		if err != nil {
			return nil, err
		}
		existingParams.BetaEntropy = betaEntropy
	}
	if len(newParams.DcoefAbs) == 1 {
		dcoefAbs, err := strconv.ParseFloat(newParams.DcoefAbs[0], 64)
		if err != nil {
			return nil, err
		}
		existingParams.DcoefAbs = dcoefAbs
	}
	if len(newParams.LearningRate) == 1 {
		learningRate, err := strconv.ParseFloat(newParams.LearningRate[0], 64)
		if err != nil {
			return nil, err
		}
		existingParams.LearningRate = learningRate
	}
	if len(newParams.MaxGradientThreshold) == 1 {
		maxGradientThreshold, err := strconv.ParseFloat(newParams.MaxGradientThreshold[0], 64)
		if err != nil {
			return nil, err
		}
		existingParams.MaxGradientThreshold = maxGradientThreshold
	}
	if len(newParams.MinStakeFraction) == 1 {
		minStakeFraction, err := strconv.ParseFloat(newParams.MinStakeFraction[0], 64)
		if err != nil {
			return nil, err
		}
		existingParams.MinStakeFraction = minStakeFraction
	}
	if len(newParams.MaxWorkersPerTopicRequest) == 1 {
		existingParams.MaxWorkersPerTopicRequest = newParams.MaxWorkersPerTopicRequest[0]
	}
	if len(newParams.MaxReputersPerTopicRequest) == 1 {
		existingParams.MaxReputersPerTopicRequest = newParams.MaxReputersPerTopicRequest[0]
	}
	if len(newParams.Epsilon) == 1 {
		epsilon, err := strconv.ParseFloat(newParams.Epsilon[0], 64)
		if err != nil {
			return nil, err
		}
		existingParams.Epsilon = epsilon
	}
	if len(newParams.PInferenceSynthesis) == 1 {
		pInferenceSynthesis, err := strconv.ParseFloat(newParams.PInferenceSynthesis[0], 64)
		if err != nil {
			return nil, err
		}
		existingParams.PInferenceSynthesis = pInferenceSynthesis
	}
	if len(newParams.AlphaRegret) == 1 {
		alphaRegret, err := strconv.ParseFloat(newParams.AlphaRegret[0], 64)
		if err != nil {
			return nil, err
		}
		existingParams.AlphaRegret = alphaRegret
	}
	if len(newParams.MaxUnfulfilledWorkerRequests) == 1 {
		existingParams.MaxUnfulfilledWorkerRequests = newParams.MaxUnfulfilledWorkerRequests[0]
	}
	if len(newParams.MaxUnfulfilledReputerRequests) == 1 {
		existingParams.MaxUnfulfilledReputerRequests = newParams.MaxUnfulfilledReputerRequests[0]
	}
	if len(newParams.NumberExpectedInfernceSybils) == 1 {
		existingParams.NumberExpectedInfernceSybils = newParams.NumberExpectedInfernceSybils[0]
	}
	if len(newParams.SybilTaxExponent) == 1 {
		existingParams.SybilTaxExponent = newParams.SybilTaxExponent[0]
	}
	if len(newParams.TopicRewardStakeImportance) == 1 {
		topicRewardStakeImportance, err := strconv.ParseFloat(newParams.TopicRewardStakeImportance[0], 64)
		if err != nil {
			return nil, err
		}
		existingParams.TopicRewardStakeImportance = topicRewardStakeImportance
	}
	if len(newParams.TopicRewardFeeRevenueImportance) == 1 {
		topicRewardFeeRevenueImportance, err := strconv.ParseFloat(newParams.TopicRewardFeeRevenueImportance[0], 64)
		if err != nil {
			return nil, err
		}
		existingParams.TopicRewardFeeRevenueImportance = topicRewardFeeRevenueImportance
	}
	if len(newParams.TopicRewardAlpha) == 1 {
		topicRewardAlpha, err := strconv.ParseFloat(newParams.TopicRewardAlpha[0], 64)
		if err != nil {
			return nil, err
		}
		existingParams.TopicRewardAlpha = topicRewardAlpha
	}
	if len(newParams.ValidatorsVsAlloraPercentReward) == 1 {
		existingParams.ValidatorsVsAlloraPercentReward = newParams.ValidatorsVsAlloraPercentReward[0]
	}
	if len(newParams.MaxSamplesToScaleScores) == 1 {
		existingParams.MaxSamplesToScaleScores = newParams.MaxSamplesToScaleScores[0]
	}
	err = ms.k.SetParams(ctx, existingParams)
	if err != nil {
		return nil, err
	}
	return &types.MsgUpdateParamsResponse{}, nil
}
