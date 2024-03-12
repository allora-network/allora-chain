package emissions

import (
	cosmosMath "cosmossdk.io/math"
)

// DefaultParams returns default module parameters.
func DefaultParams() Params {
	return Params{
		Version:                     "0.0.3",                       // version of the protocol should be in lockstep with github release tag version
		EpochLength:                 int64(600),                    // length of an "epoch" for rewards payouts in blocks
		EmissionsPerEpoch:           cosmosMath.NewInt(1000),       // default amount of tokens to issue per epoch
		MinTopicUnmetDemand:         cosmosMath.NewUint(100),       // total unmet demand for a topic < this => don't run inference solicatation or loss update
		MaxTopicsPerBlock:           uint64(2048),                  // max number of topics to run cadence for per block
		MinRequestUnmetDemand:       cosmosMath.NewUint(1),         // delete requests if they have below this demand remaining
		MaxMissingInferencePercent:  uint64(20),                    // if a worker has this percentage of inferences missing, they are penalized
		RequiredMinimumStake:        cosmosMath.NewUint(100),       // minimum stake required to be a worker or reputer
		RemoveStakeDelayWindow:      uint64(60 * 60 * 24),          // 1 day in seconds
		MinRequestCadence:           uint64(10),                    // 10 seconds
		MinLossCadence:              uint64(60 * 60),               // 1 hour in seconds
		MaxInferenceRequestValidity: uint64(60 * 60 * 24 * 7 * 52), // 52 weeks approximately 1 year in seconds
		MaxRequestCadence:           uint64(60 * 60 * 24 * 7 * 52), // 52 weeks approximately 1 year in seconds
		Sharpness:                   uint64(20),                    // controls going from stake-weighted consensus at low values to majority vote of above-average stake holders at high values
		BetaEntropy:                 float32(0.25),                 // controls resilience of reward payouts against copycat workers
		DcoefAbs:                    float32(0.001),                // delta for numerical differentiation
		LearningRate:                float32(0.05),                 // speed of gradient descent
		MaxGradientThreshold:        float32(0.001),                // gradient descent stops when gradient falls below this
		MinStakeFraction:            float32(0.5),                  // minimum fraction of stake that should be listened to when setting consensus listening coefficients
	}
}

func DefaultParamsVersion() string {
	return DefaultParams().Version
}

func DefaultParamsEpochLength() int64 {
	return DefaultParams().EpochLength
}

func DefaultParamsEmissionsPerEpoch() cosmosMath.Int {
	return DefaultParams().EmissionsPerEpoch
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

func DefaultParamsMaxMissingInferencePercent() uint64 {
	return DefaultParams().MaxMissingInferencePercent
}

func DefaultParamsRequiredMinimumStake() cosmosMath.Uint {
	return DefaultParams().RequiredMinimumStake
}

func DefaultParamsRemoveStakeDelayWindow() uint64 {
	return DefaultParams().RemoveStakeDelayWindow
}

func DefaultParamsMinRequestCadence() uint64 {
	return DefaultParams().MinRequestCadence
}

func DefaultParamsMinLossCadence() uint64 {
	return DefaultParams().MinLossCadence
}

func DefaultParamsMaxInferenceRequestValidity() uint64 {
	return DefaultParams().MaxInferenceRequestValidity
}

func DefaultParamsMaxRequestCadence() uint64 {
	return DefaultParams().MaxRequestCadence
}

func DefaultParamsSharpness() uint64 {
	return DefaultParams().Sharpness
}

func DefaultParamsBetaEntropy() float32 {
	return DefaultParams().BetaEntropy
}

func DefaultParamsDcoefAbs() float32 {
	return DefaultParams().DcoefAbs
}

func DefaultParamsLearningRate() float32 {
	return DefaultParams().LearningRate
}

func DefaultParamsMaxGradientThreshold() float32 {
	return DefaultParams().MaxGradientThreshold
}

func DefaultParamsMinStakeFraction() float32 {
	return DefaultParams().MinStakeFraction
}

// Validate does the sanity check on the params.
func (p Params) Validate() error {
	// Sanity check goes here.
	return nil
}
