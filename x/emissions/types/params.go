package types

import (
	fmt "fmt"

	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
)

type BlockHeight = int64

// DefaultParams returns default module parameters.
func DefaultParams() Params {
	return Params{
		Version:                         "0.0.3",                                   // version of the protocol should be in lockstep with github release tag version
		MinTopicWeight:                  alloraMath.MustNewDecFromString("100"),    // total weight for a topic < this => don't run inference solicatation or loss update
		MaxTopicsPerBlock:               uint64(128),                               // max number of topics to run cadence for per block
		RequiredMinimumStake:            cosmosMath.NewInt(100),                    // minimum stake required to be a worker or reputer
		RemoveStakeDelayWindow:          int64(60 * 60 * 24 * 7 * 3),               // 3 weeks in seconds
		MinEpochLength:                  1,                                         // 1 block
		BetaEntropy:                     alloraMath.MustNewDecFromString("0.25"),   // controls resilience of reward payouts against copycat workers
		LearningRate:                    alloraMath.MustNewDecFromString("0.05"),   // speed of gradient descent
		GradientDescentMaxIters:         uint64(10),                                // max iterations on gradient desc
		MaxGradientThreshold:            alloraMath.MustNewDecFromString("0.001"),  // gradient descent stops when gradient falls below this
		MinStakeFraction:                alloraMath.MustNewDecFromString("0.5"),    // minimum fraction of stake that should be listened to when setting consensus listening coefficients
		Epsilon:                         alloraMath.MustNewDecFromString("0.0001"), // 0 threshold to prevent div by 0 and 0-approximation errors
		MaxUnfulfilledWorkerRequests:    uint64(100),                               // maximum number of outstanding nonces for worker requests per topic from the chain; needs to be bigger to account for varying topic ground truth lag
		MaxUnfulfilledReputerRequests:   uint64(100),                               // maximum number of outstanding nonces for reputer requests per topic from the chain; needs to be bigger to account for varying topic ground truth lag
		TopicRewardStakeImportance:      alloraMath.MustNewDecFromString("0.5"),    // importance of stake in determining rewards for a topic
		TopicRewardFeeRevenueImportance: alloraMath.MustNewDecFromString("0.5"),    // importance of fee revenue in determining rewards for a topic
		TopicRewardAlpha:                alloraMath.MustNewDecFromString("0.5"),    // alpha for topic reward calculation; coupled with blocktime, or how often rewards are calculated
		TaskRewardAlpha:                 alloraMath.MustNewDecFromString("0.1"),    // alpha for task reward calculation used to calculate  ~U_ij, ~V_ik, ~W_im
		ValidatorsVsAlloraPercentReward: alloraMath.MustNewDecFromString("0.25"),   // 25% rewards go to cosmos network validators
		MaxSamplesToScaleScores:         uint64(10),                                // maximum number of previous scores to store and use for standard deviation calculation
		MaxTopInferersToReward:          uint64(24),                                // max this many top inferers by score are rewarded for a topic
		MaxTopForecastersToReward:       uint64(6),                                 // max this many top forecasters by score are rewarded for a topic
		MaxTopReputersToReward:          uint64(12),                                // max this many top reputers by score are rewarded for a topic
		CreateTopicFee:                  cosmosMath.NewInt(10),                     // topic registration fee
		MaxRetriesToFulfilNoncesWorker:  int64(1),                                  // max throttle of simultaneous unfulfilled worker requests
		MaxRetriesToFulfilNoncesReputer: int64(3),                                  // max throttle of simultaneous unfulfilled reputer requests
		RegistrationFee:                 cosmosMath.NewInt(6),                      // how much workers and reputers must pay to register per topic
		DefaultPageLimit:                  uint64(100),                               // how many topics to return per page during churn of requests
		MaxPageLimit:                        uint64(1000),                              // max limit for pagination
		MinEpochLengthRecordLimit:       int64(3),                                  // minimum number of epochs to keep records for a topic
		MaxSerializedMsgLength:          int64(1000 * 1000),                        // maximum size of data to msg and query server in bytes
		BlocksPerMonth:                  DefaultParamsBlocksPerMonth(),             // ~5 seconds block time, 6311520 per year, 525960 per month
		PRewardInference:                alloraMath.NewDecFromInt64(1),             // fiducial value for rewards calculation
		PRewardForecast:                 alloraMath.NewDecFromInt64(3),             // fiducial value for rewards calculation
		PRewardReputer:                  alloraMath.NewDecFromInt64(3),             // fiducial value for rewards calculation
		CRewardInference:                alloraMath.MustNewDecFromString("0.75"),   // fiducial value for rewards calculation
		CRewardForecast:                 alloraMath.MustNewDecFromString("0.75"),   // fiducial value for rewards calculation
		FTolerance:                      alloraMath.MustNewDecFromString("0.01"),   // fiducial value for rewards calculation
		CNorm:                           alloraMath.MustNewDecFromString("0.75"),   // fiducial value for inference synthesis
	}
}

func DefaultParamsVersion() string {
	return DefaultParams().Version
}

func DefaultParamsMinTopicUnmetDemand() alloraMath.Dec {
	return DefaultParams().MinTopicWeight
}

func DefaultParamsMaxTopicsPerBlock() uint64 {
	return DefaultParams().MaxTopicsPerBlock
}

func DefaultParamsRequiredMinimumStake() cosmosMath.Int {
	return DefaultParams().RequiredMinimumStake
}

func DefaultParamsRemoveStakeDelayWindow() BlockHeight {
	return DefaultParams().RemoveStakeDelayWindow
}

func DefaultParamsMinEpochLength() BlockHeight {
	return DefaultParams().MinEpochLength
}

func DefaultParamsBetaEntropy() alloraMath.Dec {
	return DefaultParams().BetaEntropy
}

func DefaultParamsLearningRate() alloraMath.Dec {
	return DefaultParams().LearningRate
}

func DefaultParamsGradientDescentMaxIters() uint64 {
	return DefaultParams().GradientDescentMaxIters
}

func DefaultParamsMaxGradientThreshold() alloraMath.Dec {
	return DefaultParams().MaxGradientThreshold
}

func DefaultParamsMinStakeFraction() alloraMath.Dec {
	return DefaultParams().MinStakeFraction
}

func DefaultParamsEpsilon() alloraMath.Dec {
	return DefaultParams().Epsilon
}

func DefaultParamsMaxUnfulfilledWorkerRequestNonces() uint64 {
	return DefaultParams().MaxUnfulfilledWorkerRequests
}

func DefaultParamsMaxUnfulfilledReputerRequestNonces() uint64 {
	return DefaultParams().MaxUnfulfilledReputerRequests
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

func DefaultParamsValidatorsVsAlloraPercentReward() alloraMath.Dec {
	return DefaultParams().ValidatorsVsAlloraPercentReward
}

func DefaultParamsMaxSamplesToScaleScores() uint64 {
	return DefaultParams().MaxSamplesToScaleScores
}

func DefaultParamsCreateTopicFee() cosmosMath.Int {
	return DefaultParams().CreateTopicFee
}

func DefaultParamsMaxTopInferersToReward() uint64 {
	return DefaultParams().MaxTopInferersToReward
}

func DefaultParamsMaxTopForecastersToReward() uint64 {
	return DefaultParams().MaxTopForecastersToReward
}

func DefaultParamsMaxTopReputersToReward() uint64 {
	return DefaultParams().MaxTopReputersToReward
}

func DefaultParamsMaxRetriesToFulfilNoncesWorker() int64 {
	return DefaultParams().MaxRetriesToFulfilNoncesWorker
}

func DefaultParamsMaxRetriesToFulfilNoncesReputer() int64 {
	return DefaultParams().MaxRetriesToFulfilNoncesReputer
}

func DefaultParamsRegistrationFee() cosmosMath.Int {
	return DefaultParams().RegistrationFee
}

func DefaultParamsDefaultLimit() uint64 {
	return DefaultParams().DefaultPageLimit
}

func DefaultParamsMaxLimit() uint64 {
	return DefaultParams().MaxPageLimit
}

func DefaultParamsMinEpochLengthRecordLimit() int64 {
	return DefaultParams().MinEpochLengthRecordLimit
}

// ~5 seconds block time, 6311520 per year, 525960 per month
func DefaultParamsBlocksPerMonth() uint64 {
	return uint64(525960)
}

func DefaultParamsPRewardInference() alloraMath.Dec {
	return DefaultParams().PRewardInference
}

func DefaultParamsPRewardForecast() alloraMath.Dec {
	return DefaultParams().PRewardForecast
}

func DefaultParamsPRewardReputer() alloraMath.Dec {
	return DefaultParams().PRewardReputer
}

func DefaultParamsCRewardInference() alloraMath.Dec {
	return DefaultParams().CRewardInference
}

func DefaultParamsCRewardForecast() alloraMath.Dec {
	return DefaultParams().CRewardForecast
}

func DefaultParamsFTolerance() alloraMath.Dec {
	return DefaultParams().FTolerance
}

func DefaultParamsCNorm() alloraMath.Dec {
	return DefaultParams().CNorm
}

// Validate does the sanity check on the params.
func (p Params) Validate() error {
	if err := validateBlocksPerMonth(p.BlocksPerMonth); err != nil {
		return err
	}
	return nil
}

func validateBlocksPerMonth(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v == 0 {
		return fmt.Errorf("blocks per month must be positive: %d", v)
	}

	return nil
}
