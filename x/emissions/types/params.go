package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
)

// DefaultParams returns default module parameters.
func DefaultParams() Params {
	return Params{
		Version:                             "v2",                                         // version of the protocol should be in lockstep with github release tag version
		MinTopicWeight:                      alloraMath.MustNewDecFromString("100"),       // total weight for a topic < this => don't run inference solicatation or loss update
		RequiredMinimumStake:                cosmosMath.NewInt(10000),                     // minimum stake required to be a worker or reputer
		RemoveStakeDelayWindow:              int64((60 * 60 * 24 * 7 * 3) / 3),            // ~approx 3 weeks assuming 3 second block time, number of blocks to wait before finalizing a stake withdrawal
		MinEpochLength:                      12,                                           // shortest number of blocks per epoch topics are allowed to set as their cadence
		BetaEntropy:                         alloraMath.MustNewDecFromString("0.25"),      // controls resilience of reward payouts against copycat workers
		LearningRate:                        alloraMath.MustNewDecFromString("0.05"),      // speed of gradient descent
		GradientDescentMaxIters:             uint64(10),                                   // max iterations on gradient descent
		MaxGradientThreshold:                alloraMath.MustNewDecFromString("0.001"),     // gradient descent stops when gradient falls below this
		MinStakeFraction:                    alloraMath.MustNewDecFromString("0.5"),       // minimum fraction of stake that should be listened to when setting consensus listening coefficients
		EpsilonReputer:                      alloraMath.MustNewDecFromString("0.01"),      // a small tolerance quantity used to cap reputer scores at infinitesimally close proximities
		EpsilonSafeDiv:                      alloraMath.MustNewDecFromString("0.0000001"), // a small tolerance quantity used to cap division by zero
		MaxUnfulfilledWorkerRequests:        uint64(100),                                  // maximum number of outstanding nonces for worker requests per topic from the chain; needs to be bigger to account for varying topic ground truth lag
		MaxUnfulfilledReputerRequests:       uint64(100),                                  // maximum number of outstanding nonces for reputer requests per topic from the chain; needs to be bigger to account for varying topic ground truth lag
		TopicRewardStakeImportance:          alloraMath.MustNewDecFromString("0.5"),       // importance of stake in determining rewards for a topic
		TopicRewardFeeRevenueImportance:     alloraMath.MustNewDecFromString("0.5"),       // importance of fee revenue in determining rewards for a topic
		TopicRewardAlpha:                    alloraMath.MustNewDecFromString("0.5"),       // alpha for topic reward calculation; coupled with blocktime, or how often rewards are calculated
		TaskRewardAlpha:                     alloraMath.MustNewDecFromString("0.1"),       // alpha for task reward calculation used to calculate  ~U_ij, ~V_ik, ~W_im
		ValidatorsVsAlloraPercentReward:     alloraMath.MustNewDecFromString("0.25"),      // 25% rewards go to cosmos network validators
		MaxSamplesToScaleScores:             uint64(10),                                   // maximum number of previous scores to store and use for standard deviation calculation
		MaxTopInferersToReward:              uint64(32),                                   // max this many top inferers by score are rewarded for a topic
		MaxTopForecastersToReward:           uint64(6),                                    // max this many top forecasters by score are rewarded for a topic
		MaxTopReputersToReward:              uint64(6),                                    // max this many top reputers by score are rewarded for a topic
		CreateTopicFee:                      cosmosMath.NewInt(75000),                     // topic registration fee
		RegistrationFee:                     cosmosMath.NewInt(200),                       // how much workers and reputers must pay to register per topic
		DefaultPageLimit:                    uint64(100),                                  // how many topics to return per page during churn of requests
		MaxPageLimit:                        uint64(1000),                                 // max limit for pagination
		MinEpochLengthRecordLimit:           int64(3),                                     // minimum number of epochs to keep records for a topic
		MaxSerializedMsgLength:              int64(1000 * 1000),                           // maximum size of data to msg and query server in bytes
		BlocksPerMonth:                      uint64(864000),                               // ~3 seconds block time, assuming 30 days in a month 60 * 60 * 24 * 30 / 3
		PRewardInference:                    alloraMath.NewDecFromInt64(1),                // fiducial value for rewards calculation
		PRewardForecast:                     alloraMath.NewDecFromInt64(3),                // fiducial value for rewards calculation
		PRewardReputer:                      alloraMath.NewDecFromInt64(3),                // fiducial value for rewards calculation
		CRewardInference:                    alloraMath.MustNewDecFromString("0.75"),      // fiducial value for rewards calculation
		CRewardForecast:                     alloraMath.MustNewDecFromString("0.75"),      // fiducial value for rewards calculation
		CNorm:                               alloraMath.MustNewDecFromString("0.75"),      // fiducial value for inference synthesis
		HalfMaxProcessStakeRemovalsEndBlock: uint64(40),                                   // half of the max number of stake removals to process at the end of the block, set this too big and blocks require too much time to process, slowing down consensus
		DataSendingFee:                      cosmosMath.NewInt(10),                        // how much workers and reputers must pay to send payload
		MaxElementsPerForecast:              uint64(12),                                   // top forecast elements by score
		MaxActiveTopicsPerBlock:             uint64(1),                                    // maximum number of active topics per block
		MaxStringLength:                     uint64(255),                                  // maximum length of strings uploaded to the chain
		RegretPercentile:                    alloraMath.MustNewDecFromString("0.25"),      // percentile value for getting regret
		PnormSafeDiv:                        alloraMath.MustNewDecFromString("8.25"),      // pnorm divide value to calculate offset with cnorm
	}
}

// Validate does the sanity check on the params.
func (p Params) Validate() error {
	if err := validateVersion(p.Version); err != nil {
		return errorsmod.Wrap(err, "params validation failure: version")
	}
	if err := validateMinTopicWeight(p.MinTopicWeight); err != nil {
		return errorsmod.Wrap(err, "params validation failure: min topic weight")
	}
	if err := validateRequiredMinimumStake(p.RequiredMinimumStake); err != nil {
		return errorsmod.Wrap(err, "params validation failure: required minimum stake")
	}
	if err := validateRemoveStakeDelayWindow(p.RemoveStakeDelayWindow); err != nil {
		return errorsmod.Wrap(err, "params validation failure: remove stake delay window")
	}
	if err := validateMinEpochLength(p.MinEpochLength); err != nil {
		return errorsmod.Wrap(err, "params validation failure: min epoch length")
	}
	if err := validateBetaEntropy(p.BetaEntropy); err != nil {
		return errorsmod.Wrap(err, "params validation failure: beta entropy")
	}
	if err := validateLearningRate(p.LearningRate); err != nil {
		return errorsmod.Wrap(err, "params validation failure: learning rate")
	}
	if err := validateGradientDescentMaxIters(p.GradientDescentMaxIters); err != nil {
		return errorsmod.Wrap(err, "params validation failure: gradient descent max iters")
	}
	if err := validateMaxGradientThreshold(p.MaxGradientThreshold); err != nil {
		return errorsmod.Wrap(err, "params validation failure: max gradient threshold")
	}
	if err := validateMinStakeFraction(p.MinStakeFraction); err != nil {
		return errorsmod.Wrap(err, "params validation failure: min stake fraction")
	}
	if err := validateEpsilonReputer(p.EpsilonReputer); err != nil {
		return errorsmod.Wrap(err, "params validation failure: epsilon reputer")
	}
	if err := validateEpsilonSafeDiv(p.EpsilonSafeDiv); err != nil {
		return errorsmod.Wrap(err, "params validation failure: epsilon safe div")
	}
	if err := validateMaxUnfulfilledWorkerRequests(p.MaxUnfulfilledWorkerRequests); err != nil {
		return errorsmod.Wrap(err, "params validation failure: max unfulfilled worker requests")
	}
	if err := validateMaxUnfulfilledReputerRequests(p.MaxUnfulfilledReputerRequests); err != nil {
		return errorsmod.Wrap(err, "params validation failure: max unfulfilled reputer requests")
	}
	if err := validateTopicRewardStakeImportance(p.TopicRewardStakeImportance); err != nil {
		return errorsmod.Wrap(err, "params validation failure: topic reward stake importance")
	}
	if err := validateTopicRewardFeeRevenueImportance(p.TopicRewardFeeRevenueImportance); err != nil {
		return errorsmod.Wrap(err, "params validation failure: topic reward fee revenue importance")
	}
	if err := validateTopicRewardAlpha(p.TopicRewardAlpha); err != nil {
		return errorsmod.Wrap(err, "params validation failure: topic reward alpha")
	}
	if err := validateTaskRewardAlpha(p.TaskRewardAlpha); err != nil {
		return errorsmod.Wrap(err, "params validation failure: task reward alpha")
	}
	if err := validateValidatorsVsAlloraPercentReward(p.ValidatorsVsAlloraPercentReward); err != nil {
		return errorsmod.Wrap(err, "params validation failure: validators vs allora percent reward")
	}
	if err := validateMaxSamplesToScaleScores(p.MaxSamplesToScaleScores); err != nil {
		return errorsmod.Wrap(err, "params validation failure: max samples to scale scores")
	}
	if err := validateMaxTopInferersToReward(p.MaxTopInferersToReward); err != nil {
		return errorsmod.Wrap(err, "params validation failure: max top inferers to reward")
	}
	if err := validateMaxTopForecastersToReward(p.MaxTopForecastersToReward); err != nil {
		return errorsmod.Wrap(err, "params validation failure: max top forecasters to reward")
	}
	if err := validateMaxTopReputersToReward(p.MaxTopReputersToReward); err != nil {
		return errorsmod.Wrap(err, "params validation failure: max top reputers to reward")
	}
	if err := validateCreateTopicFee(p.CreateTopicFee); err != nil {
		return errorsmod.Wrap(err, "params validation failure: create topic fee")
	}
	if err := validateRegistrationFee(p.RegistrationFee); err != nil {
		return errorsmod.Wrap(err, "params validation failure: registration fee")
	}
	if err := validateDefaultPageLimit(p.DefaultPageLimit); err != nil {
		return errorsmod.Wrap(err, "params validation failure: default page limit")
	}
	if err := validateMaxPageLimit(p.MaxPageLimit); err != nil {
		return errorsmod.Wrap(err, "params validation failure: max page limit")
	}
	if err := validateMinEpochLengthRecordLimit(p.MinEpochLengthRecordLimit); err != nil {
		return errorsmod.Wrap(err, "params validation failure: min epoch length record limit")
	}
	if err := validateMaxSerializedMsgLength(p.MaxSerializedMsgLength); err != nil {
		return errorsmod.Wrap(err, "params validation failure: max serialized msg length")
	}
	if err := ValidateBlocksPerMonth(p.BlocksPerMonth); err != nil {
		return errorsmod.Wrap(err, "params validation failure: blocks per month")
	}
	if err := validatePRewardInference(p.PRewardInference); err != nil {
		return errorsmod.Wrap(err, "params validation failure: p reward inference")
	}
	if err := validatePRewardForecast(p.PRewardForecast); err != nil {
		return errorsmod.Wrap(err, "params validation failure: p reward forecast")
	}
	if err := validatePRewardReputer(p.PRewardReputer); err != nil {
		return errorsmod.Wrap(err, "params validation failure: p reward reputer")
	}
	if err := validateCRewardInference(p.CRewardInference); err != nil {
		return errorsmod.Wrap(err, "params validation failure: c reward inference")
	}
	if err := validateCRewardForecast(p.CRewardForecast); err != nil {
		return errorsmod.Wrap(err, "params validation failure: c reward forecast")
	}
	if err := validateCNorm(p.CNorm); err != nil {
		return errorsmod.Wrap(err, "params validation failure: c norm")
	}
	if err := validateHalfMaxProcessStakeRemovalsEndBlock(p.HalfMaxProcessStakeRemovalsEndBlock); err != nil {
		return errorsmod.Wrap(err, "params validation failure: half max process stake removals end block")
	}
	if err := validateDataSendingFee(p.DataSendingFee); err != nil {
		return errorsmod.Wrap(err, "params validation failure: data sending fee")
	}
	if err := validateMaxElementsPerForecast(p.MaxElementsPerForecast); err != nil {
		return errorsmod.Wrap(err, "params validation failure: max elements per forecast")
	}
	if err := validateMaxActiveTopicsPerBlock(p.MaxActiveTopicsPerBlock); err != nil {
		return errorsmod.Wrap(err, "params validation failure: max active topics per block")
	}
	if err := validateMaxStringLength(p.MaxStringLength); err != nil {
		return errorsmod.Wrap(err, "params validation failure: max string length")
	}
	if err := validateRegretPercentile(p.RegretPercentile); err != nil {
		return errorsmod.Wrap(err, "params validation failure: regret percentile")
	}
	if err := validatePnormSafeDiv(p.PnormSafeDiv); err != nil {
		return errorsmod.Wrap(err, "params validation failure: pnorm safe div")
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
	if err := ValidateDec(i); err != nil {
		return err
	}
	if i.IsNegative() {
		return ErrValidationMustBeGreaterthanZero
	}
	return nil
}

// Minimum stake required to be a worker or reputer.
// Should be >= 0.
func validateRequiredMinimumStake(i cosmosMath.Int) error {
	if err := ValidateSdkIntRepresentingMonetaryValue(i); err != nil {
		return errorsmod.Wrap(err, ErrValidationMustBeGreaterthanZero.Error())
	}
	return nil
}

// Number of blocks to enforce stake withdrawal delay.
// Should be >= 0.
func validateRemoveStakeDelayWindow(i int64) error {
	if i < 0 {
		return ErrValidationMustBeGreaterthanZero
	}
	return nil
}

// Minimum number of blocks per epoch a topic can set.
// Should be >= 0.
func validateMinEpochLength(i BlockHeight) error {
	if i < 1 {
		return ErrValidationMustBeGreaterthanZero
	}
	return nil
}

// controls resilience of reward payouts against copycat workers
// Should be 0 <= i <= 1
func validateBetaEntropy(i alloraMath.Dec) error {
	if err := ValidateDec(i); err != nil {
		return err
	}
	if !isAlloraDecBetweenZeroAndOneInclusive(i) {
		return ErrValidationMustBeBetweenZeroAndOne
	}
	return nil
}

// Speed of gradient descent.
// Should be 0 < x < 1
func validateLearningRate(i alloraMath.Dec) error {
	if err := ValidateDec(i); err != nil {
		return err
	}
	if !isAlloraDecBetweenZeroAndOneExclusive(i) {
		return ErrValidationMustBeBetweenZeroAndOne
	}
	return nil
}

// Max iterations on gradient descent.
// Should be positive non zero number, i > 0
func validateGradientDescentMaxIters(i uint64) error {
	if i == 0 {
		return ErrValidationMustBeGreaterthanZero
	}
	return nil
}

// Gradient descent stops when gradient falls below this.
// Should be 0 < i < 1
func validateMaxGradientThreshold(i alloraMath.Dec) error {
	if err := ValidateDec(i); err != nil {
		return err
	}
	if !isAlloraDecBetweenZeroAndOneExclusive(i) {
		return ErrValidationMustBeBetweenZeroAndOne
	}
	return nil
}

// minimum fraction of stake that should be listened to when setting consensus listening coefficients.
// Should be between 0 and 1.
func validateMinStakeFraction(i alloraMath.Dec) error {
	if err := ValidateDec(i); err != nil {
		return err
	}
	if !isAlloraDecBetweenZeroAndOneInclusive(i) {
		return ErrValidationMustBeBetweenZeroAndOne
	}
	return nil
}

// Small tolerance quantity used to cap reputer scores at infinitesimally close proximities.
// Should be close to zero, but not zero. i > 0
func validateEpsilonReputer(i alloraMath.Dec) error {
	if err := ValidateDec(i); err != nil {
		return err
	}
	if i.Lte(alloraMath.ZeroDec()) {
		return ErrValidationMustBeGreaterthanZero
	}
	return nil
}

// Small tolerance quantity used to cap division by zero.
func validateEpsilonSafeDiv(i alloraMath.Dec) error {
	if err := ValidateDec(i); err != nil {
		return err
	}
	if i.Lte(alloraMath.ZeroDec()) {
		return ErrValidationMustBeGreaterthanZero
	}
	return nil
}

// fiducial value for rewards calculation
// should be x > 0
func validatePRewardInference(i alloraMath.Dec) error {
	if err := ValidateDec(i); err != nil {
		return err
	}
	if i.Lte(alloraMath.ZeroDec()) {
		return ErrValidationMustBeGreaterthanZero
	}
	return nil
}

// fiducial value for rewards calculation
// should be x > 0
func validatePRewardForecast(i alloraMath.Dec) error {
	if err := ValidateDec(i); err != nil {
		return err
	}
	if i.Lte(alloraMath.ZeroDec()) {
		return ErrValidationMustBeGreaterthanZero
	}
	return nil
}

// fiducial value for rewards calculation
// should be x > 0
func validatePRewardReputer(i alloraMath.Dec) error {
	if err := ValidateDec(i); err != nil {
		return err
	}
	if i.Lte(alloraMath.ZeroDec()) {
		return ErrValidationMustBeGreaterthanZero
	}
	return nil
}

// fiducial value for rewards calculation
// should be x > 0
func validateCRewardInference(i alloraMath.Dec) error {
	if err := ValidateDec(i); err != nil {
		return err
	}
	if i.Lte(alloraMath.ZeroDec()) {
		return ErrValidationMustBeGreaterthanZero
	}
	return nil
}

// fiducial value for rewards calculation
// should be x > 0
func validateCRewardForecast(i alloraMath.Dec) error {
	if err := ValidateDec(i); err != nil {
		return err
	}
	if i.Lte(alloraMath.ZeroDec()) {
		return ErrValidationMustBeGreaterthanZero
	}
	return nil
}

// fiducial value for inference synthesis
// should be x > 0
func validateCNorm(i alloraMath.Dec) error {
	if err := ValidateDec(i); err != nil {
		return err
	}
	if i.Lte(alloraMath.ZeroDec()) {
		return ErrValidationMustBeGreaterthanZero
	}
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
	if err := ValidateDec(i); err != nil {
		return err
	}
	if !isAlloraDecBetweenZeroAndOneInclusive(i) {
		return ErrValidationMustBeBetweenZeroAndOne
	}
	return nil
}

// importance of fee revenue in determining rewards for a topic.
// should be between 0 and 1.
func validateTopicRewardFeeRevenueImportance(i alloraMath.Dec) error {
	if err := ValidateDec(i); err != nil {
		return err
	}
	if !isAlloraDecBetweenZeroAndOneInclusive(i) {
		return ErrValidationMustBeBetweenZeroAndOne
	}
	return nil
}

// alpha for topic reward calculation; coupled with blocktime, or how often rewards are calculated
// should be 0 < x < 1
func validateTopicRewardAlpha(i alloraMath.Dec) error {
	if err := ValidateDec(i); err != nil {
		return err
	}
	if !isAlloraDecBetweenZeroAndOneExclusive(i) {
		return ErrValidationMustBeBetweenZeroAndOne
	}
	return nil
}

// alpha for task reward calculation used to calculate  ~U_ij, ~V_ik, ~W_im
// should be 0 < x <= 1 (note the difference on both sides!)
func validateTaskRewardAlpha(i alloraMath.Dec) error {
	if err := ValidateDec(i); err != nil {
		return err
	}
	if !isAlloraDecBetweenZeroAndOneExclusive(i) {
		return ErrValidationMustBeBetweenZeroAndOne
	}
	return nil
}

// percent reward to go to cosmos network validators.
// Should be a value between 0 and 1.
func validateValidatorsVsAlloraPercentReward(i alloraMath.Dec) error {
	if err := ValidateDec(i); err != nil {
		return err
	}
	if !isAlloraDecBetweenZeroAndOneInclusive(i) {
		return ErrValidationMustBeBetweenZeroAndOne
	}
	return nil
}

// maximum number of previous scores to store and use for standard deviation calculation
// Should be greater than zero. Enforced by conditional + uint type
func validateMaxSamplesToScaleScores(i uint64) error {
	if i == 0 {
		return ErrValidationMustBeGreaterthanZero
	}
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

// max this many top forecast elements per forecast
// Should be zero or positive. Enforced by uint type
func validateMaxElementsPerForecast(_ uint64) error {
	return nil
}

// maximum number of active topics per block
// Should be zero or positive. Enforced by uint type
func validateMaxActiveTopicsPerBlock(_ uint64) error { return nil }

// max this many top reputers by score are rewarded for a topic
// Should be zero or positive. Enforced by uint type
func validateMaxTopReputersToReward(_ uint64) error {
	return nil
}

// topic registration fee
// must be positive or zero
func validateCreateTopicFee(i cosmosMath.Int) error {
	if err := ValidateSdkIntRepresentingMonetaryValue(i); err != nil {
		return errorsmod.Wrap(err, ErrValidationMustBeGreaterthanZero.Error())
	}
	return nil
}

// How much workers and reputers must pay to register per topic.
// Should be non-negative.
func validateRegistrationFee(i cosmosMath.Int) error {
	if err := ValidateSdkIntRepresentingMonetaryValue(i); err != nil {
		return errorsmod.Wrap(err, ErrValidationMustBeGreaterthanZero.Error())
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
		return ErrValidationMustBeGreaterthanZero
	}
	return nil
}

// maximum size of data to msg and query server in bytes
// Should be non-negative.
func validateMaxSerializedMsgLength(i int64) error {
	if i < 0 {
		return ErrValidationMustBeGreaterthanZero
	}
	return nil
}

// Number of blocks in a month.
// should be a number on the order of 525,960
func ValidateBlocksPerMonth(i uint64) error {
	if i == 0 {
		return fmt.Errorf("blocks per month must be positive: %d", i)
	}
	return nil
}

// this value should be greater than or equal to 1
func validateHalfMaxProcessStakeRemovalsEndBlock(i uint64) error {
	if i == 0 {
		return ErrValidationMustBeGreaterthanZero
	}
	return nil
}

// the maximum length of the metadata string when creating a new topic
// should be non-negative, enforced by uint type
func validateMaxStringLength(_ uint64) error {
	return nil
}

func validateRegretPercentile(i alloraMath.Dec) error {
	if err := ValidateDec(i); err != nil {
		return err
	}
	if !isAlloraDecBetweenZeroAndOneInclusive(i) {
		return ErrValidationMustBeBetweenZeroAndOne
	}
	return nil
}

func validatePnormSafeDiv(i alloraMath.Dec) error {
	if err := ValidateDec(i); err != nil {
		return err
	}
	return nil
}

// Whether an alloraDec is between the value of [0, 1] inclusive
func isAlloraDecBetweenZeroAndOneInclusive(a alloraMath.Dec) bool {
	return a.Gte(alloraMath.ZeroDec()) && a.Lte(alloraMath.OneDec())
}

// Whether an alloraDec is between the value of (0, 1) exclusive
func isAlloraDecBetweenZeroAndOneExclusive(a alloraMath.Dec) bool {
	return a.Gt(alloraMath.ZeroDec()) && a.Lt(alloraMath.OneDec())
}

// Whether an alloraDec is between the values of [0, 1)
// inclusive on 0 and exclusive on 1
func isAlloraDecZeroOrLessThanOne(a alloraMath.Dec) bool {
	return a.Gte(alloraMath.ZeroDec()) && a.Lt(alloraMath.OneDec())
}

// How much workers and reputers must pay to send data.
// Should be non-negative.
func validateDataSendingFee(i cosmosMath.Int) error {
	if err := ValidateSdkIntRepresentingMonetaryValue(i); err != nil {
		return errorsmod.Wrap(err, ErrValidationMustBeGreaterthanZero.Error())
	}
	return nil
}
