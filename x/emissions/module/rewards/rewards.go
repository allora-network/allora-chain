package rewards

import (
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func EmitRewards(ctx sdk.Context, k keeper.Keeper, activeTopics []types.Topic) error {
	// Get Total Emissions/ Fees Collected

	// Get Total Allocation

	// Get Distribution of Rewards per Topic
	for _, topic := range activeTopics {
		topicStake, err := k.GetTopicStake(ctx, topic.Id)
		if err != nil {
			return err
		}
		topicStakeFloat64, _ := topicStake.BigInt().Float64()
		topicFeeRevenue, err := k.GetTopicFeeRevenue(ctx, topic.Id)
		if err != nil {
			return err
		}
		currentFeeRevenueEpoch, err := k.GetFeeRevenueEpoch(ctx)
		if err != nil {
			return err
		}
		var feeRevenueFloat float64
		if topicFeeRevenue.Epoch != currentFeeRevenueEpoch {
			feeRevenueFloat = 0
		} else {
			feeRevenueFloat, _ = topicFeeRevenue.Revenue.BigInt().Float64()
		}
		stakeImportance, feeImportance, err := k.GetParamsStakeAndFeeRevenueImportance(ctx)
		if err != nil {
			return err
		}
		targetWeight := GetTargetWeight(
			topicStakeFloat64,
			feeRevenueFloat,
			stakeImportance,
			feeImportance,
		)
	}

	// Get Distribution of Rewards per Worker - Inference Task

	// Get Distribution of Rewards per Worker - Forecast Task

	// Get Distribution of Rewards per Reputer

	// Pay out rewards

	return nil
}
