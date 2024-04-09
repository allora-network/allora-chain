package types

import (
	cosmosMath "cosmossdk.io/math"
)

type BlockHeight = int64

// DefaultParams returns default module parameters.
func DefaultParams() Params {
	return Params{
		Version:                         "0.0.3",                                    // version of the protocol should be in lockstep with github release tag version
		RewardCadence:                   int64(600),                                 // length of an "epoch" for rewards payouts in blocks
		MinTopicUnmetDemand:             cosmosMath.NewUint(100),                    // total unmet demand for a topic < this => don't run inference solicatation or loss update
		MaxTopicsPerBlock:               uint64(2048),                               // max number of topics to run cadence for per block
		MinRequestUnmetDemand:           cosmosMath.NewUint(1),                      // delete requests if they have below this demand remaining
		MaxMissingInferencePercent:      float64(0.2),                               // if a worker has this percentage of inferences missing, they are penalized
		RequiredMinimumStake:            cosmosMath.NewUint(100),                    // minimum stake required to be a worker or reputer
		RemoveStakeDelayWindow:          int64(60 * 60 * 24),                        // 1 day in seconds
		MinEpochLength:                  1,                                          // 1 block
		MaxInferenceRequestValidity:     int64(6 * 60 * 24 * 7 * 52),                // approximately 1 year in number of blocks
		MaxRequestCadence:               int64(6 * 60 * 24 * 7 * 52),                // approximately 1 year in number of blocks
		PercentRewardsReputersWorkers:   float64(0.75),                              // 75% of rewards go to workers and reputers, 25% to cosmos validators
		Sharpness:                       20.0,                                       // controls going from stake-weighted consensus at low values to majority vote of above-average stake holders at high values
		BetaEntropy:                     float64(0.25),                              // controls resilience of reward payouts against copycat workers
		DcoefAbs:                        float64(0.001),                             // delta for numerical differentiation
		LearningRate:                    0.01,                                       // speed of gradient descent
		MaxGradientThreshold:            float64(0.001),                             // gradient descent stops when gradient falls below this
		MinStakeFraction:                float64(0.5),                               // minimum fraction of stake that should be listened to when setting consensus listening coefficients
		MaxWorkersPerTopicRequest:       uint64(20),                                 // maximum number of workers that can be assigned to a single inference request
		MaxReputersPerTopicRequest:      uint64(20),                                 // maximum number of reputers that can be assigned to a single loss request
		Epsilon:                         float64(0.0001),                            // 0 threshold to prevent div by 0 and 0-approximation errors
		PInferenceSynthesis:             float64(2),                                 // free parameter used in the gradient function phi' for inference synthesis
		AlphaRegret:                     float64(0.1),                               // how much to weight the most recent log-loss differences in regret EMA update
		MaxUnfulfilledWorkerRequests:    uint64(100),                                // maximum number of outstanding nonces for worker requests from the chain
		MaxUnfulfilledReputerRequests:   uint64(100),                                // maximum number of outstanding nonces for reputer requests from the chain
		NumberOfClientsForTax:           uint64(10),                                 // global number for calculation tax of worker reward
		ParameterForTax:                 uint64(3),                                  // global parameter for calculation tax of worker reward
		TopicRewardStakeImportance:      float64(0.5),                               // importance of stake in determining rewards for a topic
		TopicRewardFeeRevenueImportance: float64(0.5),                               // importance of fee revenue in determining rewards for a topic
		TopicRewardAlpha:                float64(0.5),                               // alpha for topic reward calculation
		ValidatorsVsAlloraPercentReward: cosmosMath.LegacyMustNewDecFromStr("0.25"), // 25% rewards go to cosmos network validators
	}
}

func DefaultParamsVersion() string {
	return DefaultParams().Version
}

func DefaultParamsEpochLength() BlockHeight {
	return DefaultParams().RewardCadence
}

func DefaultParamsMinTopicUnmetDemand() cosmosMath.Uint {
	return DefaultParams().MinTopicUnmetDemand
}

func DefaultParamsMaxTopicsPerBlock() uint64 {
	return DefaultParams().MaxTopicsPerBlock
}

func DefaultParamsMinRequestUnmetDemand() cosmosMath.Uint {
	return DefaultParams().MinRequestUnmetDemand
}

func DefaultParamsMaxMissingInferencePercent() float64 {
	return DefaultParams().MaxMissingInferencePercent
}

func DefaultParamsRequiredMinimumStake() cosmosMath.Uint {
	return DefaultParams().RequiredMinimumStake
}

func DefaultParamsRemoveStakeDelayWindow() BlockHeight {
	return DefaultParams().RemoveStakeDelayWindow
}

func DefaultParamsMinEpochLength() BlockHeight {
	return DefaultParams().MinEpochLength
}

func DefaultParamsMaxInferenceRequestValidity() BlockHeight {
	return DefaultParams().MaxInferenceRequestValidity
}

func DefaultParamsMaxRequestCadence() BlockHeight {
	return DefaultParams().MaxRequestCadence
}

func DefaultParamsPercentRewardsReputersWorkers() float64 {
	return DefaultParams().PercentRewardsReputersWorkers
}

func DefaultParamsSharpness() float64 {
	return DefaultParams().Sharpness
}

func DefaultParamsBetaEntropy() float64 {
	return DefaultParams().BetaEntropy
}

func DefaultParamsDcoefAbs() float64 {
	return DefaultParams().DcoefAbs
}

func DefaultParamsLearningRate() float64 {
	return DefaultParams().LearningRate
}

func DefaultParamsMaxGradientThreshold() float64 {
	return DefaultParams().MaxGradientThreshold
}

func DefaultParamsMinStakeFraction() float64 {
	return DefaultParams().MinStakeFraction
}

func DefaultParamsMaxWorkersPerTopicRequest() uint64 {
	return DefaultParams().MaxWorkersPerTopicRequest
}

func DefaultParamsMaxReputersPerTopicRequest() uint64 {
	return DefaultParams().MaxReputersPerTopicRequest
}

func DefaultParamsEpsilon() float64 {
	return DefaultParams().Epsilon
}

func DefaultParamsPInferenceSynthesis() float64 {
	return DefaultParams().PInferenceSynthesis
}

func DefaultParamsAlphaRegret() float64 {
	return DefaultParams().AlphaRegret
}

func DefaultParamsMaxUnfulfilledWorkerRequestNonces() uint64 {
	return DefaultParams().MaxUnfulfilledWorkerRequests
}

func DefaultParamsMaxUnfulfilledReputerRequestNonces() uint64 {
	return DefaultParams().MaxUnfulfilledReputerRequests
}

func DefaultParamsNumberOfClientsForTax() uint64 {
	return DefaultParams().NumberOfClientsForTax
}

func DefaultParameterForTax() uint64 {
	return DefaultParams().ParameterForTax
}

func DefaultParamsTopicRewardStakeImportance() float64 {
	return DefaultParams().TopicRewardStakeImportance
}

func DefaultParamsTopicRewardFeeRevenueImportance() float64 {
	return DefaultParams().TopicRewardFeeRevenueImportance
}

func DefaultParamsTopicRewardAlpha() float64 {
	return DefaultParams().TopicRewardAlpha
}

func DefaultParamsValidatorsVsAlloraPercentReward() cosmosMath.LegacyDec {
	return DefaultParams().ValidatorsVsAlloraPercentReward
}

// Validate does the sanity check on the params.
func (p Params) Validate() error {
	// Sanity check goes here.
	return nil
}
