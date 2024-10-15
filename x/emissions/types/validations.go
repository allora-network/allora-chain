package types

import (
	"encoding/hex"

	"cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/utils"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	reputerValueBundleBufferPool       = utils.NewBytesPool(1024, 0)
	inferenceForecastsBundleBufferPool = utils.NewBytesPool(1024, 0)
)

/// EXTERNAL TYPE VALIDATIONS

// ValidateDec checks if the given value is a valid Dec by our standards
func ValidateDec(value alloraMath.Dec) error {
	if value.IsNaN() {
		return errors.Wrap(sdkerrors.ErrInvalidType, "value cannot be NaN")
	}

	if !value.IsFinite() {
		return errors.Wrap(sdkerrors.ErrInvalidType, "value must be finite")
	}

	return nil
}

// ValidateSdkInt checks if the given value is a valid cosmosMath.Int
// according to our needs / standards
func ValidateSdkInt(value cosmosMath.Int) error {
	if value.IsNil() {
		return errors.Wrap(sdkerrors.ErrInvalidType, "value cannot be nil")
	}

	return nil
}

// ValidateSdkIntRepresentingMonetaryAmount checks if the given value is a valid cosmosMath.Int
// according to our needs / standards
func ValidateSdkIntRepresentingMonetaryValue(value cosmosMath.Int) error {
	if err := ValidateSdkInt(value); err != nil {
		return errors.Wrap(err, "value is not a valid cosmosMath.Int")
	}
	if value.IsNegative() {
		return errors.Wrap(sdkerrors.ErrInvalidType, "monetary values cannot be negative")
	}
	return nil
}

// ValidateBech32 checks if the given value is a valid bech32 address
func ValidateBech32(value string) error {
	// AccAddressFromBech32 returns an error if the address is not valid
	// also checks if value is empty string
	_, err := sdk.AccAddressFromBech32(value)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid bech32 address (%s)", err)
	}

	return nil
}

/// PRIMITIVE TYPE VALIDATIONS

// ValidateBlockHeight checks if the given value is a valid block height
func ValidateBlockHeight(value BlockHeight) error {
	if value < 0 {
		return errors.Wrap(sdkerrors.ErrInvalidType, "block height must be greater than or equal to 0")
	}
	return nil
}

// ValidateTopicId checks if the given value is a valid topic id
func ValidateTopicId(value TopicId) error {
	if value == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic id zero is reserved")
	}
	return nil
}

/// EMISSIONS TYPES PACKAGE VALIDATIONS

// Validate performs basic genesis state validation returning an error upon any
func (gs *GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	// Ensure that the core team addresses are valid
	for _, addr := range gs.CoreTeamAddresses {
		_, err := sdk.AccAddressFromBech32(addr)
		if err != nil {
			return err
		}
	}

	return nil
}

// validate that an inference follows the expected format
func (inference *Inference) Validate() error {
	if inference == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "inference cannot be nil")
	}
	if err := ValidateTopicId(inference.TopicId); err != nil {
		return errors.Wrap(err, "inference topic id is invalid")
	}
	if err := ValidateBech32(inference.Inferer); err != nil {
		return errors.Wrap(err, "inference inferer address is invalid")
	}
	if err := ValidateBlockHeight(inference.BlockHeight); err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid inferer address (%s)", err)
	}
	if err := ValidateDec(inference.Value); err != nil {
		return errors.Wrap(err, "inference value is invalid")
	}
	// ExtraData not validated as it is not used by the chain
	// Proof not validated as it is not used by the chain
	return nil
}

// validate that a forecast follows the expected format
func (forecast *Forecast) Validate() error {
	if forecast == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "forecast cannot be nil")
	}
	if err := ValidateTopicId(forecast.TopicId); err != nil {
		return errors.Wrap(err, "forecast topic id is invalid")
	}
	if err := ValidateBlockHeight(forecast.BlockHeight); err != nil {
		return errors.Wrap(err, "forecast block height is invalid")
	}
	if err := ValidateBech32(forecast.Forecaster); err != nil {
		return errors.Wrap(err, "forecast forecaster address is invalid")
	}
	if len(forecast.ForecastElements) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "at least one forecast element must be provided")
	}
	for _, elem := range forecast.ForecastElements {
		if err := ValidateBech32(elem.Inferer); err != nil {
			return errors.Wrap(err, "forecast inferer address is invalid")
		}
		if err := ValidateDec(elem.Value); err != nil {
			return errors.Wrap(err, "forecast value is invalid")
		}
	}
	// ExtraData not validated as it is not used by the chain
	return nil
}

// validate that a worker data bundle follows the expected format
func (bundle *WorkerDataBundle) Validate() error {
	if bundle == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "worker data bundle cannot be nil")
	}
	if bundle.Nonce == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "worker data bundle nonce cannot be nil")
	}
	if len(bundle.Worker) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "worker cannot be empty")
	}
	if len(bundle.Pubkey) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "public key cannot be empty")
	}
	pk, err := hex.DecodeString(bundle.Pubkey)
	if err != nil || len(pk) != secp256k1.PubKeySize {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "invalid pubkey")
	}
	pubkey := secp256k1.PubKey(pk)
	pubKeyConvertedToAddress := sdk.AccAddress(pubkey.Address().Bytes()).String()

	if len(bundle.InferencesForecastsBundleSignature) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "signature cannot be empty")
	}
	if bundle.InferenceForecastsBundle == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "inference forecasts bundle cannot be nil")
	}

	// Validate the inference and forecast of the bundle
	if bundle.InferenceForecastsBundle.Inference == nil && bundle.InferenceForecastsBundle.Forecast == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "inference and forecast cannot both be nil")
	}
	if bundle.InferenceForecastsBundle.Inference != nil {
		if err := bundle.InferenceForecastsBundle.Inference.Validate(); err != nil {
			return err
		}
		if bundle.InferenceForecastsBundle.Inference.Inferer != pubKeyConvertedToAddress {
			return errors.Wrapf(sdkerrors.ErrUnauthorized,
				"Inference.Inferer %s does not match pubkey %s",
				bundle.InferenceForecastsBundle.Inference.Inferer, pubKeyConvertedToAddress)
		}
		if bundle.Worker != bundle.InferenceForecastsBundle.Inference.Inferer {
			return errors.Wrapf(sdkerrors.ErrUnauthorized,
				"Inference.Inferer %s does not match worker address %s",
				bundle.InferenceForecastsBundle.Inference.Inferer, bundle.Worker)
		}
	}
	if bundle.InferenceForecastsBundle.Forecast != nil {
		if err := bundle.InferenceForecastsBundle.Forecast.Validate(); err != nil {
			return err
		}
		if bundle.InferenceForecastsBundle.Forecast.Forecaster != pubKeyConvertedToAddress {
			return errors.Wrapf(sdkerrors.ErrUnauthorized,
				"Forecast.Forecaster %s does not match pubkey %s",
				bundle.InferenceForecastsBundle.Forecast.Forecaster, pubKeyConvertedToAddress)
		}
		if bundle.Worker != bundle.InferenceForecastsBundle.Forecast.Forecaster {
			return errors.Wrapf(sdkerrors.ErrUnauthorized,
				"Forecast.Forecaster %s does not match worker address %s",
				bundle.InferenceForecastsBundle.Forecast.Forecaster, bundle.Worker)
		}
	}

	// Check signature from the bundle, throw if invalid!
	buf := inferenceForecastsBundleBufferPool.Get()
	defer inferenceForecastsBundleBufferPool.Put(buf)
	marshaled, err := bundle.InferenceForecastsBundle.XXX_Marshal(buf, true)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "failed to marshal inference forecasts bundle: %s", err)
	}
	if !pubkey.VerifySignature(marshaled, bundle.InferencesForecastsBundleSignature) {
		return errors.Wrap(sdkerrors.ErrUnauthorized, "signature verification failed")
	}
	// Source: https://docs.cosmos.network/v0.46/basics/accounts.html#addresses
	if pubKeyConvertedToAddress != bundle.Worker {
		return errors.Wrap(sdkerrors.ErrUnauthorized, "worker address does not match signature")
	}

	return nil
}

// validate that a value bundle follows the expected format
func (bundle *ValueBundle) Validate() error {
	if bundle == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "value bundle cannot be nil")
	}
	if err := ValidateTopicId(bundle.TopicId); err != nil {
		return errors.Wrap(err, "value bundle topic id is invalid")
	}
	if bundle.ReputerRequestNonce == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "value bundle reputer request nonce cannot be nil")
	}
	if err := ValidateBech32(bundle.Reputer); err != nil {
		return errors.Wrap(err, "value bundle reputer address is invalid")
	}
	// extraData is not checked as it is not used by the chain

	if err := ValidateDec(bundle.CombinedValue); err != nil {
		return errors.Wrap(err, "value bundle combined value is invalid")
	}

	// nil values for bundle.InfererValues are interpreted to mean that there
	// are no inferer values for this bundle, and are allowed
	for _, infererValue := range bundle.InfererValues {
		if err := infererValue.Validate(); err != nil {
			return errors.Wrap(err, "value bundle inferer value is invalid")
		}
	}

	// nil values for bundle.ForecasterValues are interpreted to mean that there
	// are no forecaster values for this bundle, and are allowed
	for _, forecasterValue := range bundle.ForecasterValues {
		if err := forecasterValue.Validate(); err != nil {
			return errors.Wrap(err, "value bundle forecaster value is invalid")
		}
	}

	if err := ValidateDec(bundle.NaiveValue); err != nil {
		return errors.Wrap(err, "value bundle naive value is invalid")
	}

	// nil values for bundle.OneOutInfererValues are interpreted to mean that there
	// are no one out inferer values for this bundle, and are allowed
	for _, oneOutInfererValue := range bundle.OneOutInfererValues {
		if err := oneOutInfererValue.Validate(); err != nil {
			return errors.Wrap(err, "value bundle one out inferer value is invalid")
		}
	}

	// nil values for bundle.OneOutForecasterValues are interpreted to mean that there
	// are no one out forecaster values for this bundle, and are allowed
	for _, oneOutForecasterValue := range bundle.OneOutForecasterValues {
		if err := oneOutForecasterValue.Validate(); err != nil {
			return errors.Wrap(err, "value bundle one out forecaster value is invalid")
		}
	}

	// nil values for bundle.OneInForecasterValues are interpreted to mean that there
	// are no one in forecaster values for this bundle, and are allowed
	for _, oneInForecasterValue := range bundle.OneInForecasterValues {
		if err := oneInForecasterValue.Validate(); err != nil {
			return errors.Wrap(err, "value bundle one in forecaster value is invalid")
		}
	}

	// nil values for bundle.OneOutInfererForecasterValues are interpreted to mean that there
	// are no one out inferer forecaster values for this bundle, and are allowed
	for _, oneOutInfererForecaster := range bundle.OneOutInfererForecasterValues {
		if err := oneOutInfererForecaster.Validate(); err != nil {
			return errors.Wrap(err, "value bundle one out inferer forecaster value is invalid")
		}
	}
	return nil
}

// validate that a reputer value bundle follows the expected format
func (bundle *ReputerValueBundle) Validate() error {
	if bundle.ValueBundle == nil {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "value bundle cannot be nil")
	}
	pk, err := hex.DecodeString(bundle.Pubkey)
	if err != nil || len(pk) != secp256k1.PubKeySize {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid pubkey %d", len(pk))
	}
	pubkey := secp256k1.PubKey(pk)
	pubKeyConvertedToAddress := sdk.AccAddress(pubkey.Address().Bytes()).String()

	if bundle.ValueBundle.Reputer != pubKeyConvertedToAddress {
		return errors.Wrapf(sdkerrors.ErrUnauthorized, "Reputer does not match pubkey")
	}

	// validate the value bundle
	if err := bundle.ValueBundle.Validate(); err != nil {
		return errors.Wrap(err, "value bundle is invalid")
	}

	buf := reputerValueBundleBufferPool.Get()
	defer reputerValueBundleBufferPool.Put(buf)
	marshaled, err := bundle.ValueBundle.XXX_Marshal(buf, true)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "failed to marshal value bundle: %s", err)
	}
	if !pubkey.VerifySignature(marshaled, bundle.Signature) {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "signature verification failed")
	}

	return nil
}

// Validate checks if the given Topic is valid
func (topic Topic) Validate(params Params) error {
	if topic.Id == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic id zero is reserved")
	}
	if err := ValidateBech32(topic.Creator); err != nil {
		return errors.Wrap(err, "topic creator address invalid")
	}
	if uint64(len(topic.Metadata)) > params.MaxStringLength {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic metadata invalid")
	}
	if len(topic.LossMethod) == 0 || uint64(len(topic.LossMethod)) > params.MaxStringLength {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic loss method invalid")
	}
	if topic.EpochLastEnded < 0 {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic epoch last ended cannot be negative")
	}
	if topic.EpochLength <= 0 {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic epoch length must be greater than zero")
	}
	if topic.EpochLength < params.MinEpochLength {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic epoch length must be greater than minimum epoch length")
	}
	if topic.WorkerSubmissionWindow == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic worker submission window must be greater than zero")
	}
	if topic.GroundTruthLag <= 0 {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic ground truth lag must be greater than zero")
	}
	if topic.GroundTruthLag < topic.EpochLength {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic ground truth lag cannot be lower than epoch length")
	}
	if uint64(topic.GroundTruthLag) > params.MaxUnfulfilledReputerRequests*uint64(topic.EpochLength) {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic ground truth lag cannot be higher than max unfulfilled reputer requests")
	}
	if topic.WorkerSubmissionWindow <= 0 {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic worker submission window must be greater than zero")
	}
	if topic.WorkerSubmissionWindow > topic.EpochLength {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic worker submission window cannot be higher than epoch length")
	}
	if err := ValidateDec(topic.AlphaRegret); err != nil {
		return errors.Wrap(err, "topic alpha regret is invalid")
	}
	if topic.AlphaRegret.Lte(alloraMath.ZeroDec()) || topic.AlphaRegret.Gt(alloraMath.OneDec()) {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic alpha regret must be greater than 0 and less than or equal to 1")
	}
	if err := ValidateDec(topic.PNorm); err != nil {
		return errors.Wrap(err, "topic p-norm is invalid")
	}
	if topic.PNorm.Lt(alloraMath.MustNewDecFromString("2.5")) || topic.PNorm.Gt(alloraMath.MustNewDecFromString("4.5")) {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic p-norm must be between 2.5 and 4.5")
	}
	if err := ValidateDec(topic.Epsilon); err != nil {
		return errors.Wrap(err, "topic epsilon is invalid")
	}
	if topic.Epsilon.Lte(alloraMath.ZeroDec()) {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic epsilon must be greater than 0")
	}
	// no validation on AllowNegative because either it is true or false
	// and both are valid values
	//	AllowNegative            bool
	if err := ValidateDec(topic.MeritSortitionAlpha); err != nil {
		return errors.Wrap(err, "topic merit sortition alpha is invalid")
	}
	if !isAlloraDecZeroOrLessThanOne(topic.MeritSortitionAlpha) {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic merit sortition alpha must be between 0 and 1 inclusive")
	}
	if err := ValidateDec(topic.ActiveInfererQuantile); err != nil {
		return errors.Wrap(err, "topic active inferer quantile is invalid")
	}
	if !isAlloraDecZeroOrLessThanOne(topic.ActiveInfererQuantile) {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic active inferer quantile must be between 0 and 1 inclusive")
	}
	if err := ValidateDec(topic.ActiveForecasterQuantile); err != nil {
		return errors.Wrap(err, "topic active forecaster quantile is invalid")
	}
	if !isAlloraDecZeroOrLessThanOne(topic.ActiveForecasterQuantile) {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic active forecaster quantile must be between 0 and 1 inclusive")
	}
	if err := ValidateDec(topic.ActiveReputerQuantile); err != nil {
		return errors.Wrap(err, "topic active reputer quantile is invalid")
	}
	if !isAlloraDecZeroOrLessThanOne(topic.ActiveReputerQuantile) {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic active reputer quantile must be between 0 and 1 inclusive")
	}

	return nil
}

// validate that a worker attributed value follows the expected format
func (workerValue *WorkerAttributedValue) Validate() error {
	_, err := sdk.AccAddressFromBech32(workerValue.Worker)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid worker address (%s)", err)
	}

	if err := ValidateDec(workerValue.Value); err != nil {
		return err
	}

	return nil
}

// validate that a withheld worker attributed value follows the expected format
func (withheldWorkerValue *WithheldWorkerAttributedValue) Validate() error {
	_, err := sdk.AccAddressFromBech32(withheldWorkerValue.Worker)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid withheld worker address (%s)", err)
	}

	if err := ValidateDec(withheldWorkerValue.Value); err != nil {
		return err
	}

	return nil
}

// validate that a types.OneOutInfererForecasterValues follows the expected format
func (oneOutInfererForecasterValues *OneOutInfererForecasterValues) Validate() error {
	if err := ValidateBech32(oneOutInfererForecasterValues.Forecaster); err != nil {
		return errors.Wrap(err, "one out inferer forecaster values forecaster is invalid")
	}
	if oneOutInfererForecasterValues.OneOutInfererValues == nil {
		return errors.Wrap(sdkerrors.ErrInvalidType, "one out inferer forecaster values one out inferer values cannot be nil")
	}
	for _, oneOutInfererValue := range oneOutInfererForecasterValues.OneOutInfererValues {
		if err := oneOutInfererValue.Validate(); err != nil {
			return errors.Wrap(err, "one out inferer forecaster values one out inferer value is invalid")
		}
	}
	return nil
}

// validate that a types.ReputerValueBundles follows the expected format
func (bundle *ReputerValueBundles) Validate() error {
	if bundle.ReputerValueBundles == nil {
		return errors.Wrapf(sdkerrors.ErrInvalidType, "reputer value bundles cannot be nil")
	}
	for i, reputerValueBundle := range bundle.ReputerValueBundles {
		if err := reputerValueBundle.Validate(); err != nil {
			return errors.Wrapf(err, "reputer value bundle at index %d is invalid", i)
		}
	}
	return nil
}

// validate that a types.Nonce has the expected format
func (nonce *Nonce) Validate() error {
	if err := ValidateBlockHeight(nonce.BlockHeight); err != nil {
		return errors.Wrap(err, "nonce block height is invalid")
	}
	return nil
}

// validate that a types.Nonces has the expected format, for each nonce in the Nonces
func (nonces *Nonces) Validate() error {
	for i, nonce := range nonces.Nonces {
		if err := nonce.Validate(); err != nil {
			return errors.Wrapf(err, "nonces nonce at index %d is invalid", i)
		}
	}
	return nil
}

// validate that a types.ReputerRequestNonce has the expected format
func (nonce *ReputerRequestNonce) Validate() error {
	if nonce.ReputerNonce == nil {
		return errors.Wrap(sdkerrors.ErrInvalidType, "reputer request nonce reputer nonce cannot be nil")
	}
	if err := nonce.ReputerNonce.Validate(); err != nil {
		return errors.Wrap(err, "reputer request nonce reputer nonce is invalid")
	}
	return nil
}

// validate that a types.ReputerRequestNonces has the expected format, for each nonce in the ReputerRequestNonces
func (nonces *ReputerRequestNonces) Validate() error {
	if nonces.Nonces == nil {
		return errors.Wrap(sdkerrors.ErrInvalidType, "reputer request nonces cannot be nil")
	}
	for i, nonce := range nonces.Nonces {
		if err := nonce.Validate(); err != nil {
			return errors.Wrapf(err, "reputer request nonces nonce at index %d is invalid", i)
		}
	}
	return nil
}

// validate that a types.Score has the expected format
func (score *Score) Validate() error {
	if err := ValidateTopicId(score.TopicId); err != nil {
		return errors.Wrap(err, "score topic id is invalid")
	}
	if err := ValidateBlockHeight(score.BlockHeight); err != nil {
		return errors.Wrap(err, "score block height is invalid")
	}
	if err := ValidateBech32(score.Address); err != nil {
		return errors.Wrap(err, "score address is invalid")
	}
	if err := ValidateDec(score.Score); err != nil {
		return errors.Wrap(err, "score score decimal is invalid")
	}
	return nil
}

// validate that a types.Scores has the expected format, for each score in the Scores
func (scores *Scores) Validate() error {
	for i, score := range scores.Scores {
		if err := score.Validate(); err != nil {
			return errors.Wrapf(err, "scores: score at index %d is invalid", i)
		}
	}
	return nil
}

// validate that a types.ListeningCoefficient has the expected format
func (listeningCoefficient *ListeningCoefficient) Validate() error {
	if err := ValidateDec(listeningCoefficient.Coefficient); err != nil {
		return errors.Wrap(err, "listening coefficient coefficient decimal is invalid")
	}
	return nil
}

// validate that a types.TimestampedValue has the expected format
func (timestampedValue *TimestampedValue) Validate() error {
	if err := ValidateBlockHeight(timestampedValue.BlockHeight); err != nil {
		return errors.Wrap(err, "timestamped value block height is invalid")
	}
	if err := ValidateDec(timestampedValue.Value); err != nil {
		return errors.Wrap(err, "timestamped value value decimal is invalid")
	}
	return nil
}

// validate a DelegatorInfo follows expected format
func (delegatorInfo *DelegatorInfo) Validate() error {
	if err := ValidateDec(delegatorInfo.Amount); err != nil {
		return errors.Wrap(err, "delegatorInfo amount is invalid")
	}
	if err := ValidateDec(delegatorInfo.RewardDebt); err != nil {
		return errors.Wrap(err, "delegatorInfo reward debt is invalid")
	}
	return nil
}

// validate that a StakeRemovalInfo follows expected format
func (stakeRemovalInfo *StakeRemovalInfo) Validate() error {
	if err := ValidateTopicId(stakeRemovalInfo.TopicId); err != nil {
		return errors.Wrap(err, "stakeRemovalInfo topic id is invalid")
	}
	if err := ValidateBech32(stakeRemovalInfo.Reputer); err != nil {
		return errors.Wrap(err, "stakeRemovalInfo reputer is invalid")
	}
	if err := ValidateSdkIntRepresentingMonetaryValue(stakeRemovalInfo.Amount); err != nil {
		return errors.Wrap(err, "stakeRemovalInfo amount is invalid")
	}
	if err := ValidateBlockHeight(stakeRemovalInfo.BlockRemovalStarted); err != nil {
		return errors.Wrap(err, "stakeRemovalInfo block removal started is invalid")
	}
	if err := ValidateBlockHeight(stakeRemovalInfo.BlockRemovalCompleted); err != nil {
		return errors.Wrap(err, "stakeRemovalInfo block removal completed is invalid")
	}
	if stakeRemovalInfo.BlockRemovalStarted > stakeRemovalInfo.BlockRemovalCompleted {
		return errors.Wrap(sdkerrors.ErrInvalidRequest,
			"stakeRemovalInfo block removal started cannot be greater than block removal completed")
	}
	return nil
}

// validate that a DelegateStakeRemovalInfo follows expected format
func (delegateStakeRemovalInfo *DelegateStakeRemovalInfo) Validate() error {
	if err := ValidateTopicId(delegateStakeRemovalInfo.TopicId); err != nil {
		return errors.Wrap(err, "delegateStakeRemovalInfo topic id is invalid")
	}
	if err := ValidateBech32(delegateStakeRemovalInfo.Delegator); err != nil {
		return errors.Wrap(err, "delegateStakeRemovalInfo delegator is invalid")
	}
	if err := ValidateBech32(delegateStakeRemovalInfo.Reputer); err != nil {
		return errors.Wrap(err, "delegateStakeRemovalInfo reputer is invalid")
	}
	if err := ValidateSdkIntRepresentingMonetaryValue(delegateStakeRemovalInfo.Amount); err != nil {
		return errors.Wrap(err, "delegateStakeRemovalInfo amount is invalid")
	}
	if err := ValidateBlockHeight(delegateStakeRemovalInfo.BlockRemovalStarted); err != nil {
		return errors.Wrap(err, "delegateStakeRemovalInfo block removal started is invalid")
	}
	if err := ValidateBlockHeight(delegateStakeRemovalInfo.BlockRemovalCompleted); err != nil {
		return errors.Wrap(err, "delegateStakeRemovalInfo block removal completed is invalid")
	}
	if delegateStakeRemovalInfo.BlockRemovalStarted > delegateStakeRemovalInfo.BlockRemovalCompleted {
		return errors.Wrap(sdkerrors.ErrInvalidRequest,
			"delegateStakeRemovalInfo block removal started cannot be greater than block removal completed")
	}
	return nil
}

// validate that a OffchainNode follows the expected format
func (oc *OffchainNode) Validate() error {
	if err := ValidateBech32(oc.NodeAddress); err != nil {
		return errors.Wrap(err, "offchain node node address is invalid")
	}
	if err := ValidateBech32(oc.Owner); err != nil {
		return errors.Wrap(err, "offchain node owner is invalid")
	}
	return nil
}

/// PROTOBUF MESSAGE VALIDATIONS

// validate that a register request follows the expected format
func (msg *RegisterRequest) Validate() error {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid sender address (%s)", err)
	}
	_, err = sdk.AccAddressFromBech32(msg.Owner)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid owner address (%s)", err)
	}

	return nil
}

// validate that a remove registration request follows the expected format
func (msg *RemoveRegistrationRequest) Validate() error {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid sender address (%s)", err)
	}
	return nil
}

// stakeValidateHelper validates the provided addresses and amount.
// The `allowZeroAmount` parameter determines whether zero amounts are acceptable.
// In some cases, such as cancel or reward operations, zero amounts are valid, as they indicate no specific stake or value transfer is expected.
// In other cases, such as adding or removing stakes, the amount must be positive because a zero value doesn't make sense in the context of increasing or decreasing a stake.
func stakeValidateHelper(addr []string, amount cosmosMath.Int, allowZeroAmount bool) error {
	if amount.IsNegative() {
		return errors.Wrapf(sdkerrors.ErrInvalidCoins, "amount must be non-negative: %s", amount.String())
	}
	if !allowZeroAmount && amount.IsZero() {
		return errors.Wrapf(sdkerrors.ErrInvalidCoins, "amount must be positive: %s", amount.String())
	}
	for _, ad := range addr {
		_, err := sdk.AccAddressFromBech32(ad)
		if err != nil {
			return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid address (%s)", ad)
		}
	}
	return nil
}

// validate that an add stake request follows the expected format
func (msg *AddStakeRequest) Validate() error {
	return stakeValidateHelper([]string{msg.Sender}, msg.Amount, false)
}

// validate that a remove stake request follows the expected format
func (msg *RemoveStakeRequest) Validate() error {
	return stakeValidateHelper([]string{msg.Sender}, msg.Amount, false)
}

// validate that a delegate stake request follows the expected format
func (msg *DelegateStakeRequest) Validate() error {
	if msg.Reputer == msg.Sender {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "cannot self delegate")
	}

	return stakeValidateHelper([]string{msg.Sender, msg.Reputer}, msg.Amount, false)
}

// validate that a remove delegate stake request follows the expected format
func (msg *RemoveDelegateStakeRequest) Validate() error {
	return stakeValidateHelper([]string{msg.Sender, msg.Reputer}, msg.Amount, false)
}

// validate that a cancel remove delegate stake request follows the expected format
func (msg *CancelRemoveDelegateStakeRequest) Validate() error {
	return stakeValidateHelper([]string{msg.Sender}, cosmosMath.ZeroInt(), true)
}

// validate that a reward delegate stake request follows the expected format
func (msg *RewardDelegateStakeRequest) Validate() error {
	return stakeValidateHelper([]string{msg.Sender, msg.Reputer}, cosmosMath.ZeroInt(), true)
}

// validate that a cancel remove stake request follows the expected format
func (msg *CancelRemoveStakeRequest) Validate() error {
	return stakeValidateHelper([]string{msg.Sender}, cosmosMath.ZeroInt(), true)
}

// Validate checks if the given CreateNewTopicRequest is valid
func (msg *CreateNewTopicRequest) Validate(maxStringLen uint64) error {
	if err := ValidateBech32(msg.Creator); err != nil {
		return errors.Wrap(err, "invalid msg Creator address")
	}

	if len(msg.LossMethod) == 0 || uint64(len(msg.LossMethod)) > maxStringLen {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "loss method invalid")
	}
	if msg.EpochLength <= 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "epoch length must be greater than zero")
	}
	if msg.WorkerSubmissionWindow == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "worker submission window must be greater than zero")
	}
	if msg.GroundTruthLag < msg.EpochLength {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "ground truth lag cannot be lower than epoch length")
	}
	if msg.WorkerSubmissionWindow > msg.EpochLength {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "worker submission window cannot be higher than epoch length")
	}
	if msg.AlphaRegret.Lte(alloraMath.ZeroDec()) || msg.AlphaRegret.Gt(alloraMath.OneDec()) || ValidateDec(msg.AlphaRegret) != nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "alpha regret must be greater than 0 and less than or equal to 1")
	}
	if msg.PNorm.Lt(alloraMath.MustNewDecFromString("2.5")) || msg.PNorm.Gt(alloraMath.MustNewDecFromString("4.5")) || ValidateDec(msg.PNorm) != nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "p-norm must be between 2.5 and 4.5")
	}
	if msg.Epsilon.Lte(alloraMath.ZeroDec()) || msg.Epsilon.IsNaN() || !msg.Epsilon.IsFinite() {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "epsilon must be greater than 0")
	}
	if uint64(len(msg.Metadata)) > maxStringLen {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "metadata invalid")
	}
	// no validation on AllowNegative because either it is true or false
	// and both are valid values
	//	AllowNegative            bool

	if !isAlloraDecZeroOrLessThanOne(msg.MeritSortitionAlpha) {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "merit sortition alpha must be between 0 and 1 inclusive")
	}
	if !isAlloraDecZeroOrLessThanOne(msg.ActiveInfererQuantile) {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "active inferer quantile must be between 0 and 1 inclusive")
	}
	if !isAlloraDecZeroOrLessThanOne(msg.ActiveForecasterQuantile) {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "active forecaster quantile must be between 0 and 1 inclusive")
	}
	if !isAlloraDecZeroOrLessThanOne(msg.ActiveReputerQuantile) {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "active reputer quantile must be between 0 and 1 inclusive")
	}

	return nil
}
