package types

import (
	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
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
		MaxMissingInferencePercent:      alloraMath.MustNewDecFromString("0.2"),     // if a worker has this percentage of inferences missing, they are penalized
		RequiredMinimumStake:            cosmosMath.NewUint(100),                    // minimum stake required to be a worker or reputer
		RemoveStakeDelayWindow:          int64(60 * 60 * 24),                        // 1 day in seconds
		MinEpochLength:                  1,                                          // 1 block
		MaxInferenceRequestValidity:     int64(6 * 60 * 24 * 7 * 52),                // approximately 1 year in number of blocks
		MaxRequestCadence:               int64(6 * 60 * 24 * 7 * 52),                // approximately 1 year in number of blocks
		PercentRewardsReputersWorkers:   alloraMath.MustNewDecFromString("0.75"),    // 75% of rewards go to workers and reputers, 25% to cosmos validators
		Sharpness:                       alloraMath.MustNewDecFromString("20"),      // controls going from stake-weighted consensus at low values to majority vote of above-average stake holders at high values
		BetaEntropy:                     alloraMath.MustNewDecFromString("0.25"),    // controls resilience of reward payouts against copycat workers
		DcoefAbs:                        alloraMath.MustNewDecFromString("0.001"),   // delta for numerical differentiation
		LearningRate:                    alloraMath.MustNewDecFromString("0.01"),    // speed of gradient descent
		MaxGradientThreshold:            alloraMath.MustNewDecFromString("0.001"),   // gradient descent stops when gradient falls below this
		MinStakeFraction:                alloraMath.MustNewDecFromString("0.5"),     // minimum fraction of stake that should be listened to when setting consensus listening coefficients
		MaxWorkersPerTopicRequest:       uint64(20),                                 // maximum number of workers that can be assigned to a single inference request
		MaxReputersPerTopicRequest:      uint64(20),                                 // maximum number of reputers that can be assigned to a single loss request
		Epsilon:                         alloraMath.MustNewDecFromString("0.0001"),  // 0 threshold to prevent div by 0 and 0-approximation errors
		PInferenceSynthesis:             alloraMath.MustNewDecFromString("2"),       // free parameter used in the gradient function phi' for inference synthesis
		AlphaRegret:                     alloraMath.MustNewDecFromString("0.1"),     // how much to weight the most recent log-loss differences in regret EMA update
		MaxUnfulfilledWorkerRequests:    uint64(100),                                // maximum number of outstanding nonces for worker requests from the chain
		MaxUnfulfilledReputerRequests:   uint64(100),                                // maximum number of outstanding nonces for reputer requests from the chain
		NumberOfClientsForTax:           uint64(10),                                 // global number for calculation tax of worker reward
		ParameterForTax:                 uint64(3),                                  // global parameter for calculation tax of worker reward
		TopicRewardStakeImportance:      alloraMath.MustNewDecFromString("0.5"),     // importance of stake in determining rewards for a topic
		TopicRewardFeeRevenueImportance: alloraMath.MustNewDecFromString("0.5"),     // importance of fee revenue in determining rewards for a topic
		TopicRewardAlpha:                alloraMath.MustNewDecFromString("0.5"),     // alpha for topic reward calculation
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

func DefaultParamsMaxMissingInferencePercent() alloraMath.Dec {
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

func DefaultParamsPercentRewardsReputersWorkers() alloraMath.Dec {
	return DefaultParams().PercentRewardsReputersWorkers
}

func DefaultParamsSharpness() alloraMath.Dec {
	return DefaultParams().Sharpness
}

func DefaultParamsBetaEntropy() alloraMath.Dec {
	return DefaultParams().BetaEntropy
}

func DefaultParamsDcoefAbs() alloraMath.Dec {
	return DefaultParams().DcoefAbs
}

func DefaultParamsLearningRate() alloraMath.Dec {
	return DefaultParams().LearningRate
}

func DefaultParamsMaxGradientThreshold() alloraMath.Dec {
	return DefaultParams().MaxGradientThreshold
}

func DefaultParamsMinStakeFraction() alloraMath.Dec {
	return DefaultParams().MinStakeFraction
}

func DefaultParamsMaxWorkersPerTopicRequest() uint64 {
	return DefaultParams().MaxWorkersPerTopicRequest
}

func DefaultParamsMaxReputersPerTopicRequest() uint64 {
	return DefaultParams().MaxReputersPerTopicRequest
}

func DefaultParamsEpsilon() alloraMath.Dec {
	return DefaultParams().Epsilon
}

func DefaultParamsPInferenceSynthesis() alloraMath.Dec {
	return DefaultParams().PInferenceSynthesis
}

func DefaultParamsAlphaRegret() alloraMath.Dec {
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

func DefaultParamsTopicRewardStakeImportance() alloraMath.Dec {
	return DefaultParams().TopicRewardStakeImportance
}

func DefaultParamsTopicRewardFeeRevenueImportance() alloraMath.Dec {
	return DefaultParams().TopicRewardFeeRevenueImportance
}

func DefaultParamsTopicRewardAlpha() alloraMath.Dec {
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
