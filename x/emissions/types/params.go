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
		RemoveStakeDelayWindow:          int64(60 * 60 * 24 * 7 * 3),               // 3 weeks in seconds, number of seconds to wait before finalizing a stake withdrawal
		MinEpochLength:                  1,                                         // 1 block, the shortest number of blocks per epoch topics are allowed to set as their cadence
		BetaEntropy:                     alloraMath.MustNewDecFromString("0.25"),   // controls resilience of reward payouts against copycat workers
		LearningRate:                    alloraMath.MustNewDecFromString("0.05"),   // speed of gradient descent
		GradientDescentMaxIters:         uint64(10),                                // max iterations on gradient descent
		MaxGradientThreshold:            alloraMath.MustNewDecFromString("0.001"),  // gradient descent stops when gradient falls below this
		MinStakeFraction:                alloraMath.MustNewDecFromString("0.5"),    // minimum fraction of stake that should be listened to when setting consensus listening coefficients
		Epsilon:                         alloraMath.MustNewDecFromString("0.0001"), // 0 threshold to prevent div by 0 and 0-approximation errors
		PInferenceSynthesis:             alloraMath.NewDecFromInt64(2),             // free parameter used in the gradient function phi' for inference synthesis
		PRewardSpread:                   alloraMath.NewDecFromInt64(1),             // fiducial value = 1; Exponent for W_i total reward allocated to reputers per timestep
		AlphaRegret:                     alloraMath.MustNewDecFromString("0.1"),    // how much to weight the most recent log-loss differences in regret EMA update
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
		SigmoidA:                        alloraMath.NewDecFromInt64(8),             // sigmoid function parameter, a = 8
		SigmoidB:                        alloraMath.MustNewDecFromString("0.5"),    // sigmoid function parameter, b = 0.5
		MaxRetriesToFulfilNoncesWorker:  int64(1),                                  // max throttle of simultaneous unfulfilled worker requests
		MaxRetriesToFulfilNoncesReputer: int64(3),                                  // max throttle of simultaneous unfulfilled reputer requests
		RegistrationFee:                 cosmosMath.NewInt(6),                      // how much workers and reputers must pay to register per topic
		DefaultPageLimit:                uint64(100),                               // default limit for pagination
		MaxPageLimit:                    uint64(1000),                              // max limit for pagination
		MinEpochLengthRecordLimit:       int64(3),                                  // minimum number of epochs to keep records for a topic
		MaxSerializedMsgLength:          int64(1000 * 1000),                        // maximum size of data to msg and query server in bytes
		BlocksPerMonth:                  uint64(525960),                            // ~5 seconds block time, 6311520 per year, 525960 per month
	}
}

// Validate does the sanity check on the params.
func (p Params) Validate() error {
	if err := validateVersion(p.Version); err != nil {
		return err
	}
	if err := validateMinTopicWeight(p.MinTopicWeight); err != nil {
		return err
	}
	if err := validateMaxTopicsPerBlock(p.MaxTopicsPerBlock); err != nil {
		return err
	}
	if err := validateRequiredMinimumStake(p.RequiredMinimumStake); err != nil {
		return err
	}
	if err := validateRemoveStakeDelayWindow(p.RemoveStakeDelayWindow); err != nil {
		return err
	}
	if err := validateMinEpochLength(p.MinEpochLength); err != nil {
		return err
	}
	if err := validateBetaEntropy(p.BetaEntropy); err != nil {
		return err
	}
	if err := validateLearningRate(p.LearningRate); err != nil {
		return err
	}
	if err := validateGradientDescentMaxIters(p.GradientDescentMaxIters); err != nil {
		return err
	}
	if err := validateMaxGradientThreshold(p.MaxGradientThreshold); err != nil {
		return err
	}
	if err := validateMinStakeFraction(p.MinStakeFraction); err != nil {
		return err
	}
	if err := validateEpsilon(p.Epsilon); err != nil {
		return err
	}
	if err := validatePInferenceSynthesis(p.PInferenceSynthesis); err != nil {
		return err
	}
	if err := validatePRewardSpread(p.PRewardSpread); err != nil {
		return err
	}
	if err := validateAlphaRegret(p.AlphaRegret); err != nil {
		return err
	}
	if err := validateMaxUnfulfilledWorkerRequests(p.MaxUnfulfilledWorkerRequests); err != nil {
		return err
	}
	if err := validateMaxUnfulfilledReputerRequests(p.MaxUnfulfilledReputerRequests); err != nil {
		return err
	}
	if err := validateTopicRewardStakeImportance(p.TopicRewardStakeImportance); err != nil {
		return err
	}
	if err := validateTopicRewardFeeRevenueImportance(p.TopicRewardFeeRevenueImportance); err != nil {
		return err
	}
	if err := validateTopicRewardAlpha(p.TopicRewardAlpha); err != nil {
		return err
	}
	if err := validateTaskRewardAlpha(p.TaskRewardAlpha); err != nil {
		return err
	}
	if err := validateValidatorsVsAlloraPercentReward(p.ValidatorsVsAlloraPercentReward); err != nil {
		return err
	}
	if err := validateMaxSamplesToScaleScores(p.MaxSamplesToScaleScores); err != nil {
		return err
	}
	if err := validateMaxTopInferersToReward(p.MaxTopInferersToReward); err != nil {
		return err
	}
	if err := validateMaxTopForecastersToReward(p.MaxTopForecastersToReward); err != nil {
		return err
	}
	if err := validateMaxTopReputersToReward(p.MaxTopReputersToReward); err != nil {
		return err
	}
	if err := validateCreateTopicFee(p.CreateTopicFee); err != nil {
		return err
	}
	if err := validateSigmoidA(p.SigmoidA); err != nil {
		return err
	}
	if err := validateSigmoidB(p.SigmoidB); err != nil {
		return err
	}
	if err := validateMaxRetriesToFulfilNoncesWorker(p.MaxRetriesToFulfilNoncesWorker); err != nil {
		return err
	}
	if err := validateMaxRetriesToFulfilNoncesReputer(p.MaxRetriesToFulfilNoncesReputer); err != nil {
		return err
	}
	if err := validateRegistrationFee(p.RegistrationFee); err != nil {
		return err
	}
	if err := validateDefaultPageLimit(p.DefaultPageLimit); err != nil {
		return err
	}
	if err := validateMaxPageLimit(p.MaxPageLimit); err != nil {
		return err
	}
	if err := validateMinEpochLengthRecordLimit(p.MinEpochLengthRecordLimit); err != nil {
		return err
	}
	if err := validateMaxSerializedMsgLength(p.MaxSerializedMsgLength); err != nil {
		return err
	}
	if err := validateBlocksPerMonth(p.BlocksPerMonth); err != nil {
		return err
	}

	return nil
}

// Version of the protocol should be in lockstep with github release tag version.
// Should be between 1 and 32 characters. We do not enforce semver or a specific format.
func validateVersion(v string) error {
	lenV := len(v)
	if v == "" || lenV == 0 {
		return ErrValidationVersionEmpty
	}
	if lenV > 32 {
		return ErrValidationVersionTooLong
	}
	return nil
}

// Total weight for a topic < this => don't run inference solicatation or loss update.
// Should be >= 0
func validateMinTopicWeight(i alloraMath.Dec) error {
	if i.IsNegative() {
		return ErrValidationMustBeNonNegative
	}
	return nil
}

// Max number of topics to run cadence for per block.
// Should be >= 0, uint enforces this.
func validateMaxTopicsPerBlock(_ uint64) error {
	return nil
}

// Minimum stake required to be a worker or reputer.
// Should be >= 0.
func validateRequiredMinimumStake(i cosmosMath.Int) error {
	if i.IsNegative() {
		return ErrValidationMustBeNonNegative
	}
	return nil
}

// Number of seconds to enforce stake withdrawal delay.
// Should be >= 0.
func validateRemoveStakeDelayWindow(i int64) error {
	if i < 0 {
		return ErrValidationMustBeNonNegative
	}
	return nil
}

// Minumum number of blocks per epoch a topic can set.
// Should be >= 0.
func validateMinEpochLength(i BlockHeight) error {
	if i < 0 {
		return ErrValidationMustBeNonNegative
	}
	return nil
}

// controls resilience of reward payouts against copycat workers
// Should be ??? (between zero and one always?)
func validateBetaEntropy(i alloraMath.Dec) error {
	return nil
}

// Speed of gradient descent.
// Should be ?? (between zero and one always?)
func validateLearningRate(i alloraMath.Dec) error {
	return nil
}

// Max iterations on gradient descent.
// Should be ?? (positive non zero number? positive or zero?)
func validateGradientDescentMaxIters(i uint64) error {
	return nil
}

// Gradient descent stops when gradient falls below this.
// Should be ??? (between zero and one always?)
func validateMaxGradientThreshold(i alloraMath.Dec) error {
	return nil
}

// minimum fraction of stake that should be listened to when setting consensus listening coefficients.
// Should be between 0 and 1.
func validateMinStakeFraction(i alloraMath.Dec) error {
	if !isAlloraDecBetweenZeroAndOne(i) {
		return ErrValidationMustBeBetweenZeroAndOne
	}
	return nil
}

// 0 threshold to prevent div by 0 and 0-approximation errors.
// Should be close to zero, but not zero.
func validateEpsilon(i alloraMath.Dec) error {
	if i.Lte(alloraMath.ZeroDec()) {
		return ErrValidationMustBeGreaterthanZero
	}
	return nil
}

// free parameter used in the gradient function phi' for inference synthesis
// should be ??? (positive number?)
func validatePInferenceSynthesis(i alloraMath.Dec) error {
	return nil
}

// fiducial value = 1; Exponent for W_i total reward allocated to reputers per timestep
// should be ??? (positive number?)
func validatePRewardSpread(i alloraMath.Dec) error {
	return nil
}

// how much to weight the most recent log-loss differences in regret EMA update
// should be ??? (between zero and one always?)
func validateAlphaRegret(i alloraMath.Dec) error {
	return nil
}

// maximum number of outstanding nonces for worker requests per topic from the chain
// Should be zero or positive. Enforced by uint type
func validateMaxUnfulfilledWorkerRequests(_ uint64) error {
	return nil
}

// maximum number of outstanding nonces for reputer requests per topic from the chain
// Should be zero or positive. Enforced by uint type
func validateMaxUnfulfilledReputerRequests(_ uint64) error {
	return nil
}

// importance of stake in determining rewards for a topic.
// should be between 0 and 1.
func validateTopicRewardStakeImportance(i alloraMath.Dec) error {
	if !isAlloraDecBetweenZeroAndOne(i) {
		return ErrValidationMustBeBetweenZeroAndOne
	}
	return nil
}

// importance of fee revenue in determining rewards for a topic.
// should be between 0 and 1.
func validateTopicRewardFeeRevenueImportance(i alloraMath.Dec) error {
	if !isAlloraDecBetweenZeroAndOne(i) {
		return ErrValidationMustBeBetweenZeroAndOne
	}
	return nil
}

// alpha for topic reward calculation; coupled with blocktime, or how often rewards are calculated
// should be ??? (between zero and one always?)
func validateTopicRewardAlpha(i alloraMath.Dec) error {
	return nil
}

// alpha for task reward calculation used to calculate  ~U_ij, ~V_ik, ~W_im
// should be ??? (between zero and one always?)
func validateTaskRewardAlpha(i alloraMath.Dec) error {
	return nil
}

// percent reward to go to cosmos network validators.
// Should be a value between 0 and 1.
func validateValidatorsVsAlloraPercentReward(i alloraMath.Dec) error {
	if !isAlloraDecBetweenZeroAndOne(i) {
		return ErrValidationMustBeBetweenZeroAndOne
	}
	return nil
}

// maximum number of previous scores to store and use for standard deviation calculation
// Should be zero or positive. Enforced by uint type
func validateMaxSamplesToScaleScores(_ uint64) error {
	return nil
}

// max this many top workers by score are rewarded for a topic
// Should be zero or positive. Enforced by uint type
func validateMaxTopInferersToReward(_ uint64) error {
	return nil
}

// max this many top workers by score are rewarded for a topic
// Should be zero or positive. Enforced by uint type
func validateMaxTopForecastersToReward(_ uint64) error {
	return nil
}

// max this many top reputers by score are rewarded for a topic
// Should be zero or positive. Enforced by uint type
func validateMaxTopReputersToReward(_ uint64) error {
	return nil
}

// topic registration fee
// must be positive or zero
func validateCreateTopicFee(i cosmosMath.Int) error {
	if i.IsNegative() {
		return ErrValidationMustBeNonNegative
	}
	return nil
}

// sigmoid function parameter, a = 8
// should be ??? (positive number?)
func validateSigmoidA(i alloraMath.Dec) error {
	return nil
}

// sigmoid function parameter, b = 0.5
// should be ??? (between zero and one always? positive number??)
func validateSigmoidB(i alloraMath.Dec) error {
	return nil
}

// max throttle of simultaneous unfulfilled worker requests.
// Should be non negative.
func validateMaxRetriesToFulfilNoncesWorker(i int64) error {
	if i < 0 {
		return ErrValidationMustBeNonNegative
	}
	return nil
}

// max throttle of simultaneous unfulfilled reputer requests.
// Should be non negative.
func validateMaxRetriesToFulfilNoncesReputer(i int64) error {
	if i < 0 {
		return ErrValidationMustBeNonNegative
	}
	return nil
}

// How much workers and reputers must pay to register per topic.
// Should be non-negative.
func validateRegistrationFee(i cosmosMath.Int) error {
	if i.IsNegative() {
		return ErrValidationMustBeNonNegative
	}
	return nil
}

// default limit for pagination
// should be non-negative, enforced by uint type
func validateDefaultPageLimit(_ uint64) error {
	return nil
}

// max limit for pagination
// should be non-negative, enforced by uint type
func validateMaxPageLimit(_ uint64) error {
	return nil
}

// minimum number of epochs to keep records for a topic
// Should be non-negative.
func validateMinEpochLengthRecordLimit(i int64) error {
	if i < 0 {
		return ErrValidationMustBeNonNegative
	}
	return nil
}

// maximum size of data to msg and query server in bytes
// Should be non-negative.
func validateMaxSerializedMsgLength(i int64) error {
	if i < 0 {
		return ErrValidationMustBeNonNegative
	}
	return nil
}

// Number of blocks in a month.
// should be a number on the order of 525,960
func validateBlocksPerMonth(i uint64) error {
	if i == 0 {
		return fmt.Errorf("blocks per month must be positive: %d", i)
	}
	return nil
}

// Whether an alloraDec is between the value of [0, 1] inclusive
func isAlloraDecBetweenZeroAndOne(a alloraMath.Dec) bool {
	return a.Gte(alloraMath.ZeroDec()) && a.Lte(alloraMath.OneDec())
}
