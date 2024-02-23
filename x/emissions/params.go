package emissions

import (
	cosmosMath "cosmossdk.io/math"
)

// DefaultParams returns default module parameters.
func DefaultParams() Params {
	return Params{
		Version:                             "0.0.3",                 // version of the protocol should be in lockstep with github release tag version
		EpochLength:                         5,                       // length of an "epoch" for rewards payouts in blocks
		EmissionsPerEpoch:                   cosmosMath.NewInt(1000), // default amount of tokens to issue per epoch
		MinTopicUnmetDemand:                 cosmosMath.NewUint(100), // total unmet demand for a topic < this => don't run inference solicatation or weight-adjustment
		MaxTopicsPerBlock:                   1000,                    // max number of topics to run cadence for per block
		PriceChangePercent:                  0.1,                     // how much the price changes per block
		MinRequestUnmetDemand:               cosmosMath.NewUint(1),   // delete requests if they have below this demand remaining
		MaxAllowableMissingInferencePercent: 10,                      // if a worker has this percentage of inferences missing, they are penalized
		RequiredMinimumStake:                cosmosMath.NewUint(1),   // minimum stake required to be a worker
		RemoveStakeDelayWindow:              172800,                  // 2 days in seconds
		MinFastestAllowedCadence:            60,                      // 1 minute in seconds
		MaxInferenceRequestValidity:         60 * 60 * 24 * 7 * 24,   // 24 weeks approximately 6 months in seconds
		MaxSlowestAllowedCadence:            60 * 60 * 24 * 7 * 24,   // 24 weeks approximately 6 months in seconds
	}
}

// Validate does the sanity check on the params.
func (p Params) Validate() error {
	// Sanity check goes here.
	return nil
}
