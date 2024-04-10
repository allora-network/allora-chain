package rewards

import (
	"fmt"

	alloraMath "github.com/allora-network/allora-chain/math"
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
	totalRewardDec, err := alloraMath.NewDecFromSdkInt(totalReward)

	// Get Distribution of Rewards per Topic
	weights, sumWeight, err := GetActiveTopicWeights(ctx, k, activeTopics)
	if err != nil {
		fmt.Println("weights error")
		return err
	}
	validatorsVsAlloraPercentReward, err := k.GetParamsValidatorsVsAlloraPercentReward(ctx)
	if err != nil {
		fmt.Println("percent error")
		return err
	}
	f_v, err := alloraMath.NewDecFromSdkLegacyDec(validatorsVsAlloraPercentReward)
	if err != nil {
		fmt.Println("legacyDec conversion error")
		return err
	}
	topicRewards := make([]alloraMath.Dec, len(activeTopics))
	for i := range weights {
		topicWeight := weights[i]
		topicRewardFraction, err := GetTopicRewardFraction(f_v, topicWeight, sumWeight)
		if err != nil {
			fmt.Println("reward fraction error")
			return err
		}
		topicReward, err := GetTopicReward(topicRewardFraction, totalRewardDec)
		if err != nil {
			fmt.Println("reward error")
			return err
		}
		topicRewards[i] = topicReward
	}

	// 	// Get Distribution of Rewards per Worker - Inference Task

	// 	// Get Distribution of Rewards per Worker - Forecast Task

	// 	// Get Distribution of Rewards per Reputer

	// 	// Pay out rewards

	SetPreviousTopicWeights(ctx, k, activeTopics, weights)
	return nil
}
