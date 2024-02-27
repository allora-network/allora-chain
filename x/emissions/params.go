package emissions

import (
	cosmosMath "cosmossdk.io/math"
)

// DefaultParams returns default module parameters.
func DefaultParams() Params {
	return Params{
		Version:                             "0.0.3",                       // version of the protocol should be in lockstep with github release tag version
		EpochLength:                         int64(5),                      // length of an "epoch" for rewards payouts in blocks
		EmissionsPerEpoch:                   cosmosMath.NewInt(1000),       // default amount of tokens to issue per epoch
		MinTopicUnmetDemand:                 cosmosMath.NewUint(100),       // total unmet demand for a topic < this => don't run inference solicatation or weight-adjustment
		MaxTopicsPerBlock:                   uint64(1000),                  // max number of topics to run cadence for per block
		MinRequestUnmetDemand:               cosmosMath.NewUint(1),         // delete requests if they have below this demand remaining
		MaxAllowableMissingInferencePercent: uint64(10),                    // if a worker has this percentage of inferences missing, they are penalized
		RequiredMinimumStake:                cosmosMath.NewUint(1),         // minimum stake required to be a worker
		RemoveStakeDelayWindow:              uint64(172800),                // 2 days in seconds
		MinFastestAllowedCadence:            uint64(60),                    // 1 minute in seconds
		MaxInferenceRequestValidity:         uint64(60 * 60 * 24 * 7 * 24), // 24 weeks approximately 6 months in seconds
		MaxSlowestAllowedCadence:            uint64(60 * 60 * 24 * 7 * 24), // 24 weeks approximately 6 months in seconds
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

func DefaultParamsMaxAllowableMissingInferencePercent() uint64 {
	return DefaultParams().MaxAllowableMissingInferencePercent
}

func DefaultParamsRequiredMinimumStake() cosmosMath.Uint {
	return DefaultParams().RequiredMinimumStake
}

func DefaultParamsRemoveStakeDelayWindow() uint64 {
	return DefaultParams().RemoveStakeDelayWindow
}

func DefaultParamsMinFastestAllowedCadence() uint64 {
	return DefaultParams().MinFastestAllowedCadence
}

func DefaultParamsMaxInferenceRequestValidity() uint64 {
	return DefaultParams().MaxInferenceRequestValidity
}

func DefaultParamsMaxSlowestAllowedCadence() uint64 {
	return DefaultParams().MaxSlowestAllowedCadence
}

// Validate does the sanity check on the params.
func (p Params) Validate() error {
	// Sanity check goes here.
	return nil
}
