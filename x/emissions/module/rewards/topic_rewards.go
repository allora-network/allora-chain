package rewards

import "math"

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
