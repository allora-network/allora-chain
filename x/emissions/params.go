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
		MinTopicUnmetDemand:         cosmosMath.NewUint(100),       // total unmet demand for a topic < this => don't run inference solicatation or weight-adjustment
		MaxTopicsPerBlock:           uint64(2048),                  // max number of topics to run cadence for per block
		MinRequestUnmetDemand:       cosmosMath.NewUint(1),         // delete requests if they have below this demand remaining
		MaxMissingInferencePercent:  uint64(20),                    // if a worker has this percentage of inferences missing, they are penalized
		RequiredMinimumStake:        cosmosMath.NewUint(100),       // minimum stake required to be a worker or reputer
		RemoveStakeDelayWindow:      uint64(60 * 60 * 24 * 5),      // 5 days in seconds
		MinRequestCadence:           uint64(10),                    // 10 seconds
		MinWeightCadence:            uint64(60 * 60),               // 1 hour in seconds
		MaxInferenceRequestValidity: uint64(60 * 60 * 24 * 7 * 52), // 52 weeks approximately 1 year in seconds
		MaxRequestCadence:           uint64(60 * 60 * 24 * 7 * 52), // 52 weeks approximately 1 year in seconds
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

func DefaultParamsMinWeightCadence() uint64 {
	return DefaultParams().MinWeightCadence
}

func DefaultParamsMaxInferenceRequestValidity() uint64 {
	return DefaultParams().MaxInferenceRequestValidity
}

func DefaultParamsMaxRequestCadence() uint64 {
	return DefaultParams().MaxRequestCadence
}

// Validate does the sanity check on the params.
func (p Params) Validate() error {
	// Sanity check goes here.
	return nil
}
