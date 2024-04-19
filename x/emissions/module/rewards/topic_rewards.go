package rewards

import (
	"context"
	"fmt"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// The amount of emission rewards to be distributed to a topic
// E_{t,i} = f_{t,i}*E_i
// f_{t,i} is the reward fraction for that topic
// E_i is the reward emission total for that epoch
func GetTopicReward(
	topicRewardFraction alloraMath.Dec,
	totalReward alloraMath.Dec,
) (alloraMath.Dec, error) {
	return topicRewardFraction.Mul(totalReward)
}

// The reward fraction for a topic
// normalize the topic reward weight
// f{t,i} = (1 - f_v) * (w_{t,i}) / (∑_t w_{t,i})
// where f_v is a global parameter set that controls the
// fraction of total reward emissions for cosmos network validators
// we don't use f_v here, because by the time the emissions module runs
// the validator rewards have already been distributed to the fee_collector account
// (this is done in the mint and then distribution module)
// w_{t,i} is the weight of topic t
// and the sum is naturally the total of all the weights for all topics
func GetTopicRewardFraction(
	//	f_v alloraMath.Dec,
	topicWeight alloraMath.Dec,
	totalWeight alloraMath.Dec,
) (alloraMath.Dec, error) {
	return topicWeight.Quo(totalWeight)
}

// Return the target weight of a topic
// ^w_{t,i} = S^{μ}_{t,i} * P^{ν}_{t,i}
// where S_{t,i} is the stake of of topic t in the last reward epoch i
// and P_{t,i} is the fee revenue collected for performing inference
// requests for topic t in the last reward epoch i
// μ, ν are global constants with fiduciary values of 0.5 and 0.5
func GetTargetWeight(
	topicStake alloraMath.Dec,
	topicFeeRevenue alloraMath.Dec,
	stakeImportance alloraMath.Dec,
	feeImportance alloraMath.Dec,
) (alloraMath.Dec, error) {
	s, err := alloraMath.Pow(topicStake, stakeImportance)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	p, err := alloraMath.Pow(topicFeeRevenue, feeImportance)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	return s.Mul(p)
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
) (weights []alloraMath.Dec, sumWeight alloraMath.Dec, err error) {
	alphaTopic, err := k.GetParamsTopicRewardAlpha(ctx)
	if err != nil {
		fmt.Println("alpha error")
		return []alloraMath.Dec{}, alloraMath.Dec{}, err
	}
	currentFeeRevenueEpoch, err := k.GetFeeRevenueEpoch(ctx)
	if err != nil {
		fmt.Println("epoch error")
		return []alloraMath.Dec{}, alloraMath.Dec{}, err
	}
	stakeImportance, feeImportance, err := k.GetParamsStakeAndFeeRevenueImportance(ctx)
	if err != nil {
		fmt.Println("importance error")
		return []alloraMath.Dec{}, alloraMath.Dec{}, err
	}
	sumWeight = alloraMath.ZeroDec()
	weights = make([]alloraMath.Dec, len(activeTopics))
	for i, topic := range activeTopics {
		topicStake, err := k.GetTopicStake(ctx, topic.Id)
		if err != nil {
			fmt.Println("stake error")
			return []alloraMath.Dec{}, alloraMath.Dec{}, err
		}
		topicStakeDec, err := alloraMath.NewDecFromSdkUint(topicStake)
		if err != nil {
			fmt.Println("dec error")
			return []alloraMath.Dec{}, alloraMath.Dec{}, err
		}
		topicFeeRevenue, err := k.GetTopicFeeRevenue(ctx, topic.Id)
		if err != nil {
			fmt.Println("revenue error")
			return []alloraMath.Dec{}, alloraMath.Dec{}, err
		}
		feeRevenue := alloraMath.ZeroDec()
		if topicFeeRevenue.Epoch == currentFeeRevenueEpoch {
			feeRevenue, err = alloraMath.NewDecFromSdkInt(topicFeeRevenue.Revenue)
			if err != nil {
				fmt.Println("dec error")
				return []alloraMath.Dec{}, alloraMath.Dec{}, err
			}
		}
		targetWeight, err := GetTargetWeight(
			topicStakeDec,
			feeRevenue,
			stakeImportance,
			feeImportance,
		)
		if err != nil {
			fmt.Println("target weight error")
			return []alloraMath.Dec{}, alloraMath.Dec{}, err
		}
		previousTopicWeight, noPrior, err := k.GetPreviousTopicWeight(ctx, topic.Id)
		if err != nil {
			fmt.Println("previous topic weight error")
			return []alloraMath.Dec{}, alloraMath.Dec{}, err
		}
		previousWeight := alloraMath.ZeroDec()
		if previousTopicWeight.Epoch == currentFeeRevenueEpoch-1 {
			previousWeight = previousTopicWeight.Weight
		}
		weight, err := alloraMath.CalcEma(alphaTopic, targetWeight, previousWeight, noPrior)
		if err != nil {
			fmt.Println("ema error")
			return []alloraMath.Dec{}, alloraMath.Dec{}, err
		}
		weights[i] = weight
		sumWeight, err = sumWeight.Add(weight)
		if err != nil {
			return []alloraMath.Dec{}, alloraMath.Dec{}, err
		}
	}
	return weights, sumWeight, nil
}

// after rewards work is done you must update the
// previous topic weight in order for EMAs to use it
func SetPreviousTopicWeights(
	ctx context.Context,
	k keeper.Keeper,
	topics []types.Topic,
	topicWeights []alloraMath.Dec,
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
