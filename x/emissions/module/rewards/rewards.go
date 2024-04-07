package rewards

import (
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func EmitRewards(ctx sdk.Context, k keeper.Keeper, activeTopics []types.Topic) error {
	// Get Total Emissions/ Fees Collected

	// Get Total Allocation
	totalReward := k.BankKeeper().GetBalance(
		ctx,
		sdk.AccAddress(types.AlloraRewardsAccountName),
		sdk.DefaultBondDenom).Amount
	totalRewardFloat, _ := totalReward.BigInt().Float64()

	// Get Distribution of Rewards per Topic
	weights, sumWeight, err := GetActiveTopicWeights(ctx, k, activeTopics)
	if err != nil {
		return err
	}
	validatorsVsAlloraPercentReward, err := k.GetParamValidatorsVsAlloraPercentReward(ctx)
	if err != nil {
		return err
	}
	f_v := validatorsVsAlloraPercentReward.MustFloat64()
	topicRewards := make([]float64, len(activeTopics))
	for i := range weights {
		topicWeight := weights[i]
		topicRewardFraction := GetTopicRewardFraction(f_v, topicWeight, sumWeight)
		topicReward := GetTopicReward(topicRewardFraction, totalRewardFloat)
		topicRewards[i] = topicReward
	}

	// Get Distribution of Rewards per Worker - Inference Task

	// Get Distribution of Rewards per Worker - Forecast Task

	// Get Distribution of Rewards per Reputer

	// Pay out rewards

	return nil
}
