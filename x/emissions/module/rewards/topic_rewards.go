package rewards

import (
	"context"
	"fmt"
	"math"

	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// The amount of emission rewards to be distributed to a topic
// E_{t,i} = f_{t,i}*E_i
// f_{t,i} is the reward fraction for that topic
// E_i is the reward emission total for that epoch
func GetTopicReward(
	topicRewardFraction float64,
	totalReward float64,
) float64 {
	return topicRewardFraction * totalReward
}

// The reward fraction for a topic
// normalize the topic reward weight
// f{t,i} = (1 - f_v) * (w_{t,i}) / (∑_t w_{t,i})
// where f_v is a global parameter set that controls the
// fraction of total reward emissions for cosmos network validators
// w_{t,i} is the weight of topic t
// and the sum is naturally the total of all the weights for all topics
func GetTopicRewardFraction(
	f_v float64,
	topicWeight float64,
	totalWeight float64,
) float64 {
	return f_v * topicWeight / totalWeight
}

// Return the target weight of a topic
// ^w_{t,i} = S^{μ}_{t,i} * P^{ν}_{t,i}
// where S_{t,i} is the stake of of topic t in the last reward epoch i
// and P_{t,i} is the fee revenue collected for performing inference
// requests for topic t in the last reward epoch i
// μ, ν are global constants with fiduciary values of 0.5 and 0.5
func GetTargetWeight(
	topicStake float64,
	topicFeeRevenue float64,
	stakeImportance float64,
	feeImportance float64,
) float64 {
	s := math.Pow(topicStake, stakeImportance)
	p := math.Pow(topicFeeRevenue, feeImportance)
	return s * p
}

// iterates through every active topic
// computes its target weight,
// then exponential moving average
// to get weight. Returns the total sum as well as a
// slice of all of the weights
func GetActiveTopicWeights(
	ctx context.Context,
	k keeper.Keeper,
	activeTopics []types.Topic,
) (weights []float64, sumWeight float64, err error) {
	alphaTopic, err := k.GetParamsTopicRewardAlpha(ctx)
	if err != nil {
		fmt.Println("alpha error")
		return []float64{}, 0.0, err
	}
	currentFeeRevenueEpoch, err := k.GetFeeRevenueEpoch(ctx)
	if err != nil {
		fmt.Println("epoch error")
		return []float64{}, 0.0, err
	}
	stakeImportance, feeImportance, err := k.GetParamsStakeAndFeeRevenueImportance(ctx)
	if err != nil {
		fmt.Println("importance error")
		return []float64{}, 0.0, err
	}
	sumWeight = 0.0
	weights = make([]float64, len(activeTopics))
	for i, topic := range activeTopics {
		topicStake, err := k.GetTopicStake(ctx, topic.Id)
		if err != nil {
			fmt.Println("stake error")
			return []float64{}, 0.0, err
		}
		topicStakeFloat64, _ := topicStake.BigInt().Float64()
		topicFeeRevenue, err := k.GetTopicFeeRevenue(ctx, topic.Id)
		if err != nil {
			fmt.Println("revenue error")
			return []float64{}, 0.0, err
		}
		feeRevenueFloat := 0.0
		if topicFeeRevenue.Epoch == currentFeeRevenueEpoch {
			feeRevenueFloat, _ = topicFeeRevenue.Revenue.BigInt().Float64()
		}
		targetWeight := GetTargetWeight(
			topicStakeFloat64,
			feeRevenueFloat,
			stakeImportance,
			feeImportance,
		)
		previousTopicWeight, err := k.GetPreviousTopicWeight(ctx, topic.Id)
		if err != nil {
			fmt.Println("previous topic weight error")
			return []float64{}, 0.0, err
		}
		previousWeight := 0.0
		if previousTopicWeight.Epoch == currentFeeRevenueEpoch-1 {
			previousWeight = previousTopicWeight.Weight
		}
		weight, err := ExponentialMovingAverage(alphaTopic, targetWeight, previousWeight)
		if err != nil {
			fmt.Println("ema error")
			return []float64{}, 0.0, err
		}
		weights[i] = weight
		sumWeight += weight
	}
	return weights, sumWeight, nil
}

// after rewards work is done you must update the
// previous topic weight in order for EMAs to use it
func SetPreviousTopicWeights(
	ctx context.Context,
	k keeper.Keeper,
	topics []types.Topic,
	topicWeights []float64,
) error {
	currentEpoch, err := k.GetFeeRevenueEpoch(ctx)
	if err != nil {
		fmt.Println("epoch error")
		return err
	}
	if len(topics) != len(topicWeights) {
		fmt.Println("length error")
		return types.ErrSliceLengthMismatch
	}
	for i, tw := range topicWeights {
		ptw := types.PreviousTopicWeight{
			Epoch:  currentEpoch,
			Weight: tw,
		}
		topicId := topics[i].Id
		err := k.SetPreviousTopicWeight(ctx, topicId, ptw)
		if err != nil {
			fmt.Println("set previous weight error")
			return err
		}
	}
	return nil
}
