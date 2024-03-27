package types

import (
	cosmosMath "cosmossdk.io/math"
)

type BLOCK_NUMBER = int64

// DefaultParams returns default module parameters.
func DefaultParams() Params {
	return Params{
		Version:                       "0.0.3",                                   // version of the protocol should be in lockstep with github release tag version
		RewardCadence:                 int64(600),                                // length of an "epoch" for rewards payouts in blocks
		MinTopicUnmetDemand:           cosmosMath.NewUint(100),                   // total unmet demand for a topic < this => don't run inference solicatation or loss update
		MaxTopicsPerBlock:             uint64(2048),                              // max number of topics to run cadence for per block
		MinRequestUnmetDemand:         cosmosMath.NewUint(1),                     // delete requests if they have below this demand remaining
		MaxMissingInferencePercent:    cosmosMath.LegacyMustNewDecFromStr("0.2"), // if a worker has this percentage of inferences missing, they are penalized
		RequiredMinimumStake:          cosmosMath.NewUint(100),                   // minimum stake required to be a worker or reputer
		RemoveStakeDelayWindow:        int64(60 * 60 * 24),                       // 1 day in seconds
		MinEpochLength:                1,                                         // 1 block
		MaxInferenceRequestValidity:   int64(6 * 60 * 24 * 7 * 52),               // approximately 1 year in number of blocks
		MaxRequestCadence:             int64(6 * 60 * 24 * 7 * 52),               // approximately 1 year in number of blocks
		PercentRewardsReputersWorkers: cosmosMath.LegacyMustNewDecFromStr("0.5"), // 50% of rewards go to workers and reputers, 50% to cosmos validators
		Sharpness:                     20.0,                                      // controls going from stake-weighted consensus at low values to majority vote of above-average stake holders at high values
		BetaEntropy:                   float32(0.25),                             // controls resilience of reward payouts against copycat workers
		DcoefAbs:                      float32(0.001),                            // delta for numerical differentiation
		LearningRate:                  0.01,                                      // speed of gradient descent
		MaxGradientThreshold:          float32(0.001),                            // gradient descent stops when gradient falls below this
		MinStakeFraction:              float32(0.5),                              // minimum fraction of stake that should be listened to when setting consensus listening coefficients
		MaxWorkersPerTopicRequest:     uint64(20),                                // maximum number of workers that can be assigned to a single inference request
		MaxReputersPerTopicRequest:    uint64(20),                                // maximum number of reputers that can be assigned to a single loss request
	}
}

func DefaultParamsVersion() string {
	return DefaultParams().Version
}

func DefaultParamsEpochLength() BLOCK_NUMBER {
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

func DefaultParamsMaxMissingInferencePercent() cosmosMath.LegacyDec {
	return DefaultParams().MaxMissingInferencePercent
}

func DefaultParamsRequiredMinimumStake() cosmosMath.Uint {
	return DefaultParams().RequiredMinimumStake
}

func DefaultParamsRemoveStakeDelayWindow() BLOCK_NUMBER {
	return DefaultParams().RemoveStakeDelayWindow
}

func DefaultParamsMinEpochLength() BLOCK_NUMBER {
	return DefaultParams().MinEpochLength
}

func DefaultParamsMaxInferenceRequestValidity() BLOCK_NUMBER {
	return DefaultParams().MaxInferenceRequestValidity
}

func DefaultParamsMaxRequestCadence() BLOCK_NUMBER {
	return DefaultParams().MaxRequestCadence
}

func DefaultParamsSharpness() float64 {
	return DefaultParams().Sharpness
}

func DefaultParamsBetaEntropy() float32 {
	return DefaultParams().BetaEntropy
}

func DefaultParamsDcoefAbs() float32 {
	return DefaultParams().DcoefAbs
}

func DefaultParamsLearningRate() float64 {
	return DefaultParams().LearningRate
}

func DefaultParamsMaxGradientThreshold() float32 {
	return DefaultParams().MaxGradientThreshold
}

func DefaultParamsMinStakeFraction() float32 {
	return DefaultParams().MinStakeFraction
}

func DefaultParamsMaxWorkersPerTopicRequest() uint64 {
	return DefaultParams().MaxWorkersPerTopicRequest
}

func DefaultParamsMaxReputersPerTopicRequest() uint64 {
	return DefaultParams().MaxReputersPerTopicRequest
}

// Validate does the sanity check on the params.
func (p Params) Validate() error {
	// Sanity check goes here.
	return nil
}
