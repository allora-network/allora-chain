package emissions

import (
	cosmosMath "cosmossdk.io/math"
)

// DefaultParams returns default module parameters.
func DefaultParams() Params {
	return Params{
		Version:                       "0.0.3",                       // version of the protocol should be in lockstep with github release tag version
		EpochLength:                   int64(600),                    // length of an "epoch" for rewards payouts in blocks
		MinTopicUnmetDemand:           cosmosMath.NewUint(100),       // total unmet demand for a topic < this => don't run inference solicatation or weight-adjustment
		MaxTopicsPerBlock:             uint64(2048),                  // max number of topics to run cadence for per block
		MinRequestUnmetDemand:         cosmosMath.NewUint(1),         // delete requests if they have below this demand remaining
		MaxMissingInferencePercent:    uint64(20),                    // if a worker has this percentage of inferences missing, they are penalized
		RequiredMinimumStake:          cosmosMath.NewUint(100),       // minimum stake required to be a worker or reputer
		RemoveStakeDelayWindow:        uint64(60 * 60 * 24),          // 1 day in seconds
		MinRequestCadence:             uint64(10),                    // 10 seconds
		MinWeightCadence:              uint64(60 * 60),               // 1 hour in seconds
		MaxInferenceRequestValidity:   uint64(60 * 60 * 24 * 7 * 52), // 52 weeks approximately 1 year in seconds
		MaxRequestCadence:             uint64(60 * 60 * 24 * 7 * 52), // 52 weeks approximately 1 year in seconds
		PercentRewardsReputersWorkers: uint64(50),                    // 50% of rewards go to workers and reputers, 50% to cosmos validators
	}
}

func DefaultParamsVersion() string {
	return DefaultParams().Version
}

func DefaultParamsEpochLength() int64 {
	return DefaultParams().EpochLength
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
