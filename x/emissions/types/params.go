package types

import (
	"fmt"

	"cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
)

// DefaultParams returns default module parameters.
func DefaultParams() Params {
	return Params{
		Version:                         "0.0.3",
		MinTopicWeight:                  alloraMath.MustNewDecFromString("100"),
		MaxTopicsPerBlock:               uint64(16),
		RequiredMinimumStake:            cosmosMath.NewInt(100),
		RemoveStakeDelayWindow:          int64((60 * 60 * 24 * 7 * 3) / 5),
		MinEpochLength:                  12,
		BetaEntropy:                     alloraMath.MustNewDecFromString("0.25"),
		LearningRate:                    alloraMath.MustNewDecFromString("0.05"),
		GradientDescentMaxIters:         uint64(10),
		MaxGradientThreshold:            alloraMath.MustNewDecFromString("0.001"),
		MinStakeFraction:                alloraMath.MustNewDecFromString("0.5"),
		EpsilonReputer:                  alloraMath.MustNewDecFromString("0.01"),
		MaxUnfulfilledWorkerRequests:    uint64(100),
		MaxUnfulfilledReputerRequests:   uint64(100),
		TopicRewardStakeImportance:      alloraMath.MustNewDecFromString("0.5"),
		TopicRewardFeeRevenueImportance: alloraMath.MustNewDecFromString("0.5"),
		TopicRewardAlpha:                alloraMath.MustNewDecFromString("0.5"),
		TaskRewardAlpha:                 alloraMath.MustNewDecFromString("0.1"),
		ValidatorsVsAlloraPercentReward: alloraMath.MustNewDecFromString("0.25"),
		MaxSamplesToScaleScores:         uint64(10),
		MaxTopInferersToReward:          uint64(48),
		MaxTopForecastersToReward:       uint64(6),
		MaxTopReputersToReward:          uint64(12),
		MaxActiveInferersQuantile:       alloraMath.MustNewDecFromString("0.25"),
		MaxActiveForecastersQuantile:    alloraMath.MustNewDecFromString("0.25"),
		MaxActiveReputersQuantile:       alloraMath.MustNewDecFromString("0.25"),
		CreateTopicFee:                  cosmosMath.NewInt(10),
		MaxRetriesToFulfilNoncesWorker:  int64(1),
		MaxRetriesToFulfilNoncesReputer: int64(3),
		RegistrationFee:                 cosmosMath.NewInt(10),
		DefaultPageLimit:                uint64(100),
		MaxPageLimit:                    uint64(1000),
		MinEpochLengthRecordLimit:       int64(3),
		MaxSerializedMsgLength:          int64(1000 * 1000),
		BlocksPerMonth:                  uint64(525960),
		PRewardInference:                alloraMath.NewDecFromInt64(1),
		PRewardForecast:                 alloraMath.NewDecFromInt64(3),
		PRewardReputer:                  alloraMath.NewDecFromInt64(3),
		CRewardInference:                alloraMath.MustNewDecFromString("0.75"),
		CRewardForecast:                 alloraMath.MustNewDecFromString("0.75"),
		CNorm:                           alloraMath.MustNewDecFromString("0.75"),
		TopicFeeRevenueDecayRate:        alloraMath.MustNewDecFromString("0.025"),
		MinEffectiveTopicRevenue:        alloraMath.MustNewDecFromString("0.01"),
	}
}

// Validate does the stateless sanity check on the params.
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
	if err := validateEpsilonReputer(p.EpsilonReputer); err != nil {
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
	if err := validateMaxTopInferersToReward(p.MaxTopInferersToReward, p.MaxActiveInferersQuantile); err != nil {
		return err
	}
	if err := validateMaxTopForecastersToReward(p.MaxTopForecastersToReward, p.MaxActiveForecastersQuantile); err != nil {
		return err
	}
	if err := validateMaxTopReputersToReward(p.MaxTopReputersToReward, p.MaxActiveReputersQuantile); err != nil {
		return err
	}
	if err := validateMaxActiveInferersQuantile(p.MaxActiveInferersQuantile, p.MaxTopInferersToReward); err != nil {
		return err
	}
	if err := validateMaxActiveForecastersQuantile(p.MaxActiveForecastersQuantile, p.MaxTopForecastersToReward); err != nil {
		return err
	}
	if err := validateMaxActiveReputersQuantile(p.MaxActiveReputersQuantile, p.MaxTopReputersToReward); err != nil {
		return err
	}
	if err := validateCreateTopicFee(p.CreateTopicFee); err != nil {
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
	if err := validatePRewardInference(p.PRewardInference); err != nil {
		return err
	}
	if err := validatePRewardForecast(p.PRewardForecast); err != nil {
		return err
	}
	if err := validatePRewardReputer(p.PRewardReputer); err != nil {
		return err
	}
	if err := validateCRewardInference(p.CRewardInference); err != nil {
		return err
	}
	if err := validateCRewardForecast(p.CRewardForecast); err != nil {
		return err
	}
	if err := validateCNorm(p.CNorm); err != nil {
		return err
	}
	if err := validateTopicFeeRevenueDecayRate(p.TopicFeeRevenueDecayRate); err != nil {
		return err
	}
	if err := validateMinEffectiveTopicRevenue(p.MinEffectiveTopicRevenue); err != nil {
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
		return ErrValidationMustBeGreaterthanZero
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
		return ErrValidationMustBeGreaterthanZero
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

// Minumum number of blocks per epoch a topic can set.
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
	if !isAlloraDecBetweenZeroAndOneInclusive(i) {
		return ErrValidationMustBeBetweenZeroAndOne
	}
	return nil
}

// Speed of gradient descent.
// Should be 0 < x < 1
func validateLearningRate(i alloraMath.Dec) error {
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
	if !isAlloraDecBetweenZeroAndOneExclusive(i) {
		return ErrValidationMustBeBetweenZeroAndOne
	}
	return nil
}

// minimum fraction of stake that should be listened to when setting consensus listening coefficients.
// Should be between 0 and 1.
func validateMinStakeFraction(i alloraMath.Dec) error {
	if !isAlloraDecBetweenZeroAndOneInclusive(i) {
		return ErrValidationMustBeBetweenZeroAndOne
	}
	return nil
}

// Small tolerance quantity used to cap reputer scores at infinitesimally close proximities.
// Should be close to zero, but not zero. i > 0
func validateEpsilonReputer(i alloraMath.Dec) error {
	if i.Lte(alloraMath.ZeroDec()) {
		return ErrValidationMustBeGreaterthanZero
	}
	return nil
}

// fiducial value for rewards calculation
// should be x > 0
func validatePRewardInference(i alloraMath.Dec) error {
	if i.Lte(alloraMath.ZeroDec()) {
		return ErrValidationMustBeGreaterthanZero
	}
	return nil
}

// fiducial value for rewards calculation
// should be x > 0
func validatePRewardForecast(i alloraMath.Dec) error {
	if i.Lte(alloraMath.ZeroDec()) {
		return ErrValidationMustBeGreaterthanZero
	}
	return nil
}

// fiducial value for rewards calculation
// should be x > 0
func validatePRewardReputer(i alloraMath.Dec) error {
	if i.Lte(alloraMath.ZeroDec()) {
		return ErrValidationMustBeGreaterthanZero
	}
	return nil
}

// fiducial value for rewards calculation
// should be x > 0
func validateCRewardInference(i alloraMath.Dec) error {
	if i.Lte(alloraMath.ZeroDec()) {
		return ErrValidationMustBeGreaterthanZero
	}
	return nil
}

// fiducial value for rewards calculation
// should be x > 0
func validateCRewardForecast(i alloraMath.Dec) error {
	if i.Lte(alloraMath.ZeroDec()) {
		return ErrValidationMustBeGreaterthanZero
	}
	return nil
}

// fiducial value for inference synthesis
// should be x > 0
func validateCNorm(i alloraMath.Dec) error {
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
	if !isAlloraDecBetweenZeroAndOneInclusive(i) {
		return ErrValidationMustBeBetweenZeroAndOne
	}
	return nil
}

// importance of fee revenue in determining rewards for a topic.
// should be between 0 and 1.
func validateTopicRewardFeeRevenueImportance(i alloraMath.Dec) error {
	if !isAlloraDecBetweenZeroAndOneInclusive(i) {
		return ErrValidationMustBeBetweenZeroAndOne
	}
	return nil
}

// alpha for topic reward calculation; coupled with blocktime, or how often rewards are calculated
// should be 0 < x < 1
func validateTopicRewardAlpha(i alloraMath.Dec) error {
	if !isAlloraDecBetweenZeroAndOneExclusive(i) {
		return ErrValidationMustBeBetweenZeroAndOne
	}
	return nil
}

// alpha for task reward calculation used to calculate  ~U_ij, ~V_ik, ~W_im
// should be 0 < x <= 1 (note the difference on both sides!)
func validateTaskRewardAlpha(i alloraMath.Dec) error {
	if i.Lte(alloraMath.ZeroDec()) || i.Gt(alloraMath.OneDec()) {
		return ErrValidationMustBeBetweenZeroAndOne
	}
	return nil
}

// percent reward to go to cosmos network validators.
// Should be a value between 0 and 1.
func validateValidatorsVsAlloraPercentReward(i alloraMath.Dec) error {
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

// 1/MaxTop<Actor>sToReward should be less than maxActive<Actor>sQuantile
// and the topic <Actor>Quantile should fit between them (greater than former and less than or equal to latter).
func validateMaxTopActorVsActiveActorQuantile(maxTopActorsToReward uint64, maxActiveActorsQuantile alloraMath.Dec, errMsg *errors.Error) error {
	oneOverMax, err := alloraMath.InvUint64(maxTopActorsToReward)
	if err != nil {
		return err
	}
	if oneOverMax.Gte(maxActiveActorsQuantile) {
		return errMsg
	}
	return nil
}

func validateActiveActorQuantile(maxTopActorsToReward uint64, maxActiveActorsQuantile alloraMath.Dec, errMsg *errors.Error) error {
	// Check quantile is over maxTopActorsToReward
	err := validateMaxTopActorVsActiveActorQuantile(
		maxTopActorsToReward,
		maxActiveActorsQuantile,
		errMsg,
	)
	if err != nil {
		return err
	}
	// Check quantile does not exceed nor equal 100%
	if maxActiveActorsQuantile.Gte(alloraMath.MustNewDecFromString("100")) {
		return ErrMustBeBelow100Percent
	}
	return nil
}

// max this many top workers by score are rewarded for a topic
// Should be zero or positive. Enforced by uint type
func validateMaxTopInferersToReward(maxTopInferersToReward uint64, maxActiveInferersQuantile alloraMath.Dec) error {
	return validateMaxTopActorVsActiveActorQuantile(
		maxTopInferersToReward,
		maxActiveInferersQuantile,
		ErrMaxTopInferersIncompatibleWithActiveQuantile,
	)
}

// max this many top workers by score are rewarded for a topic
// Should be zero or positive. Enforced by uint type
func validateMaxTopForecastersToReward(maxTopForecastersToReward uint64, maxActiveForecastersQuantile alloraMath.Dec) error {
	return validateMaxTopActorVsActiveActorQuantile(
		maxTopForecastersToReward,
		maxActiveForecastersQuantile,
		ErrMaxTopForecastersIncompatibleWithActiveQuantile,
	)
}

// max this many top reputers by score are rewarded for a topic
// Should be zero or positive. Enforced by uint type
func validateMaxTopReputersToReward(maxTopReputersToReward uint64, maxActiveReputersQuantile alloraMath.Dec) error {
	return validateMaxTopActorVsActiveActorQuantile(
		maxTopReputersToReward,
		maxActiveReputersQuantile,
		ErrMaxTopReputersIncompatibleWithActiveQuantile,
	)
}

func validateMaxActiveInferersQuantile(maxActiveInferersQuantile alloraMath.Dec, maxTopInferersToReward uint64) error {
	return validateActiveActorQuantile(
		maxTopInferersToReward,
		maxActiveInferersQuantile,
		ErrMaxTopInferersIncompatibleWithActiveQuantile,
	)
}

func validateMaxActiveForecastersQuantile(maxActiveForecastersQuantile alloraMath.Dec, maxTopForecastersToReward uint64) error {
	return validateActiveActorQuantile(
		maxTopForecastersToReward,
		maxActiveForecastersQuantile,
		ErrMaxTopForecastersIncompatibleWithActiveQuantile,
	)
}

func validateMaxActiveReputersQuantile(maxActiveReputersQuantile alloraMath.Dec, maxTopReputersToReward uint64) error {
	return validateActiveActorQuantile(
		maxTopReputersToReward,
		maxActiveReputersQuantile,
		ErrMaxTopReputersIncompatibleWithActiveQuantile,
	)
}

// topic registration fee
// must be positive or zero
func validateCreateTopicFee(i cosmosMath.Int) error {
	if i.IsNegative() {
		return ErrValidationMustBeGreaterthanZero
	}
	return nil
}

// max throttle of simultaneous unfulfilled worker requests.
// Should be non negative.
func validateMaxRetriesToFulfilNoncesWorker(i int64) error {
	if i < 0 {
		return ErrValidationMustBeGreaterthanZero
	}
	return nil
}

// max throttle of simultaneous unfulfilled reputer requests.
// Should be non negative.
func validateMaxRetriesToFulfilNoncesReputer(i int64) error {
	if i < 0 {
		return ErrValidationMustBeGreaterthanZero
	}
	return nil
}

// How much workers and reputers must pay to register per topic.
// Should be non-negative.
func validateRegistrationFee(i cosmosMath.Int) error {
	if i.IsNegative() {
		return ErrValidationMustBeGreaterthanZero
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
func validateBlocksPerMonth(i uint64) error {
	if i == 0 {
		return fmt.Errorf("blocks per month must be positive: %d", i)
	}
	return nil
}

// Percent by which effecive topic fee used in weight calculation drips
// Should be a value between 0 and 1.
func validateTopicFeeRevenueDecayRate(i alloraMath.Dec) error {
	if !isAlloraDecBetweenZeroAndOneInclusive(i) {
		return ErrValidationMustBeBetweenZeroAndOne
	}
	return nil
}

// We no stop dripping from the topic's effective revenue when the topic's effective revenue is below this
// Should be > 0
func validateMinEffectiveTopicRevenue(i alloraMath.Dec) error {
	if i.Lte(alloraMath.ZeroDec()) {
		return ErrValidationMustBeGreaterthanZero
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

func validateActiveActorTopicQuantile(
	maxTopActorsToReward uint64,
	maxActiveActorsQuantile alloraMath.Dec,
	topicActiveActorsQuantile alloraMath.Dec,
	errMsg *errors.Error,
) error {
	// Check topic quantile is over maxTopActorsToReward
	err := validateMaxTopActorVsActiveActorQuantile(
		maxTopActorsToReward,
		topicActiveActorsQuantile,
		errMsg,
	)
	if err != nil {
		return err
	}
	// Check topic quantile does not exceed the global max
	if topicActiveActorsQuantile.Gt(maxActiveActorsQuantile) {
		return ErrMustBeBelow100Percent
	}
	return nil
}

func (p Params) ValidateTopicActiveInfererQuantile(topicQuantile alloraMath.Dec) error {
	return validateActiveActorTopicQuantile(
		p.MaxTopInferersToReward,
		p.MaxActiveInferersQuantile,
		topicQuantile,
		ErrMaxTopInferersIncompatibleWithActiveQuantile,
	)
}

func (p Params) ValidateTopicActiveForecasterQuantile(topicQuantile alloraMath.Dec) error {
	return validateActiveActorTopicQuantile(
		p.MaxTopForecastersToReward,
		p.MaxActiveForecastersQuantile,
		topicQuantile,
		ErrMaxTopForecastersIncompatibleWithActiveQuantile,
	)
}

func (p Params) ValidateTopicActiveReputerQuantile(topicQuantile alloraMath.Dec) error {
	return validateActiveActorTopicQuantile(
		p.MaxTopReputersToReward,
		p.MaxActiveReputersQuantile,
		topicQuantile,
		ErrMaxTopReputersIncompatibleWithActiveQuantile,
	)
}
