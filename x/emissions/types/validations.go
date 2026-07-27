package types

import (
	"encoding/hex"

	"cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/utils"
)

const (
	validationMinLossMethodLength       = 1
	validationMinPositiveIntValue int64 = 1
	validationMinEpochLastEnded   int64 = 0

	validationZeroDecString     = "0"
	validationOneDecString      = "1"
	validationPNormMaxDecString = "10"
	validationCNormMinDecString = "-100"
	validationCNormMaxDecString = "100"
)

var (
	validationZeroDec     = alloraMath.MustNewDecFromString(validationZeroDecString)
	validationOneDec      = alloraMath.MustNewDecFromString(validationOneDecString)
	validationPNormMaxDec = alloraMath.MustNewDecFromString(validationPNormMaxDecString)
	validationCNormMinDec = alloraMath.MustNewDecFromString(validationCNormMinDecString)
	validationCNormMaxDec = alloraMath.MustNewDecFromString(validationCNormMaxDecString)

	reputerValueBundleBufferPool       = utils.NewBytesPool(1024, 0)
	inferenceForecastsBundleBufferPool = utils.NewBytesPool(1024, 0)
)

// / EXTERNAL TYPE VALIDATIONS

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

// ValidateDecs checks if the list of values are all valid Dec by our standards
func ValidateDecs(values []alloraMath.Dec) error {
	for i, val := range values {
		if err := ValidateDec(val); err != nil {
			return errors.Wrapf(err, "values[%d] cannot be %s", i, val.String())
		}
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

// / PRIMITIVE TYPE VALIDATIONS

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

// / EMISSIONS TYPES PACKAGE VALIDATIONS

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

// INFERENCES AND FORECASTS

// validate that the contents of an inference follow the expected format
func validateInferenceContents(topicId uint64, inferer string, blockHeight BlockHeight) error {
	if err := ValidateTopicId(topicId); err != nil {
		return errors.Wrap(err, "topic id is invalid")
	}
	if err := ValidateBech32(inferer); err != nil {
		return errors.Wrap(err, "inferer address is invalid")
	}
	if err := ValidateBlockHeight(blockHeight); err != nil {
		return errors.Wrap(err, "block height is invalid")
	}
	return nil
}

// validate that an inference follows the expected format
func (inference *Inference) Validate() error {
	if inference == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "inference cannot be nil")
	}
	if err := validateInferenceContents(inference.TopicId, inference.Inferer, inference.BlockHeight); err != nil {
		return errors.Wrap(err, "inference contents are invalid")
	}
	if len(inference.Values) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "inference values cannot be empty")
	}
	if err := ValidateDecs(inference.Values); err != nil {
		return errors.Wrap(err, "inference values are invalid")
	}
	// ExtraData not validated as it is not used by the chain
	// Proof not validated as it is not used by the chain
	return nil
}

// validate that an inference follows the expected format
func (inputInference *InputInference) Validate() error {
	if inputInference == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "inference cannot be nil")
	}
	if err := validateInferenceContents(inputInference.TopicId, inputInference.Inferer, inputInference.BlockHeight); err != nil {
		return errors.Wrap(err, "inference contents are invalid")
	}
	// ExtraData not validated as it is not used by the chain
	// Proof not validated as it is not used by the chain
	return nil
}

// ValidateWithLimits validates an InputInference against the per-topic and
// per-submission constraints that apply to the worker payload path:
//
//   - labelCap is the topic MaxLabelsPerSubmission value applied to this
//     payload. It is the maximum number of distinct canonical labels the
//     submission may carry. Must be >= 1.
//   - whitelist, when non-empty, is the set of canonical labels accepted by the
//     topic. Nil and empty whitelists both mean unrestricted because repeated
//     fields do not preserve a nil-vs-empty distinction across serialization.
//   - maxLabelBytes is Params.MaxCanonicalLabelByteLength; it is threaded
//     into CanonicalLabelName as the per-label byte cap.
//   - labelCaseSensitive is the topic's LabelCaseSensitive flag; it is
//     threaded into CanonicalLabelName so submission-time canonicalization
//     matches the whitelist canonicalization the keeper applied in
//     SetTopic/UpdateTopic.
//
// In addition to the basic InputInference.Validate() checks, this function:
//
//   - canonicalizes each labeled value's Label via CanonicalLabelName and
//     rewrites the value in place so downstream consumers see canonical
//     bytes only;
//   - rejects duplicates after canonicalization (ErrInvalidLabelName);
//   - enforces the label cap (ErrTooManyLabelsPerSubmission);
//   - enforces whitelist membership post-canonicalization
//     (ErrLabelNotInWhitelist).
//
// The canonicalized slice is left in its submitted order so temporary ELR
// registration can preserve first-seen label order during the WSW.
func (inputInference *InputInference) ValidateWithLimits(
	labelCap uint64,
	whitelist map[string]struct{},
	maxLabelBytes uint64,
	labelCaseSensitive bool,
) error {
	if err := inputInference.Validate(); err != nil {
		return err
	}
	if labelCap == 0 {
		// Defensive: topic validation rejects zero, so a zero here indicates a
		// programming error rather than user input.
		return errors.Wrap(ErrValidationMustBeGreaterthanZero,
			"per-submission label cap must be >= 1")
	}
	n := uint64(len(inputInference.Values))
	if n > labelCap {
		return errors.Wrapf(ErrTooManyLabelsPerSubmission,
			"submission has %d labels, per-topic cap is %d", n, labelCap)
	}
	seen := make(map[string]struct{}, len(inputInference.Values))
	for i, lv := range inputInference.Values {
		if lv == nil {
			return errors.Wrapf(sdkerrors.ErrInvalidRequest,
				"input labeled value at index %d is nil", i)
		}
		c, err := CanonicalLabelName(lv.Label, maxLabelBytes, labelCaseSensitive)
		if err != nil {
			return errors.Wrapf(err, "input labeled value at index %d", i)
		}
		if _, dup := seen[c]; dup {
			return errors.Wrapf(ErrInvalidLabelName,
				"duplicate label after canonicalization at index %d: %q", i, c)
		}
		seen[c] = struct{}{}
		if len(whitelist) > 0 {
			if _, ok := whitelist[c]; !ok {
				return errors.Wrapf(ErrLabelNotInWhitelist,
					"label %q is not in topic label whitelist", c)
			}
		}
		// BoundedExp40Dec is intrinsically bounded on decode; convert to
		// alloraMath.Dec and reuse ValidateDec so the validator only rejects
		// explicitly-NaN values (the same rule applied to scalar Inference.Value).
		if err := ValidateDec(lv.Value.ToDec()); err != nil {
			return errors.Wrapf(err, "input labeled value %q", c)
		}
		// Rewrite in place to the canonical form so temporary registry
		// registration and whitelist lookups use canonical bytes only.
		lv.Label = c
	}
	return nil
}

func ValidateForecastContents(topicId uint64, forecaster string, blockHeight BlockHeight) error {
	if err := ValidateTopicId(topicId); err != nil {
		return errors.Wrap(err, "forecast topic id is invalid")
	}
	if err := ValidateBech32(forecaster); err != nil {
		return errors.Wrap(err, "forecast forecaster address is invalid")
	}
	if err := ValidateBlockHeight(blockHeight); err != nil {
		return errors.Wrap(err, "forecast block height is invalid")
	}
	return nil
}

// validate that a forecast follows the expected format
func (forecast *Forecast) Validate() error {
	if forecast == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "forecast cannot be nil")
	}
	if err := ValidateForecastContents(forecast.TopicId, forecast.Forecaster, forecast.BlockHeight); err != nil {
		return errors.Wrap(err, "forecast contents are invalid")
	}
	if len(forecast.ForecastElements) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "at least one forecast element must be provided")
	}
	for _, elem := range forecast.ForecastElements {
		if err := elem.Validate(); err != nil {
			return errors.Wrap(err, "forecast element is invalid")
		}
	}
	// ExtraData not validated as it is not used by the chain
	return nil
}

func (inputForecast *InputForecast) Validate() error {
	if inputForecast == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "forecast cannot be nil")
	}
	if err := ValidateForecastContents(inputForecast.TopicId, inputForecast.Forecaster, inputForecast.BlockHeight); err != nil {
		return errors.Wrap(err, "forecast contents are invalid")
	}
	if len(inputForecast.ForecastElements) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "at least one forecast element must be provided")
	}
	for _, elem := range inputForecast.ForecastElements {
		if err := elem.Validate(); err != nil {
			return errors.Wrap(err, "forecast element is invalid")
		}
	}
	// ExtraData not validated as it is not used by the chain
	return nil
}

func ValidateForecastElementContents(inferer string) error {
	if err := ValidateBech32(inferer); err != nil {
		return errors.Wrap(err, "forecast element inferer address is invalid")
	}
	return nil
}

func (inputForecastElement *InputForecastElement) Validate() error {
	if inputForecastElement == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "forecast element cannot be nil")
	}
	if err := ValidateForecastElementContents(inputForecastElement.Inferer); err != nil {
		return errors.Wrap(err, "forecast element contents are invalid")
	}
	return nil
}

// validate that a forecast element follows the expected format
func (forecastElement *ForecastElement) Validate() error {
	if forecastElement == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "forecast element cannot be nil")
	}
	if err := ValidateForecastElementContents(forecastElement.Inferer); err != nil {
		return errors.Wrap(err, "forecast element contents are invalid")
	}
	if err := ValidateDec(forecastElement.Value); err != nil {
		return errors.Wrap(err, "forecast element value is invalid")
	}
	return nil
}

func (inputInferenceForecastBundle *InputInferenceForecastBundle) Validate() error {
	if inputInferenceForecastBundle == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "inference forecast bundle cannot be nil")
	}

	if inputInferenceForecastBundle.Inference == nil && inputInferenceForecastBundle.Forecast == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "inference and forecast cannot both be nil")
	}

	if inputInferenceForecastBundle.Inference != nil {
		if err := inputInferenceForecastBundle.Inference.Validate(); err != nil {
			return errors.Wrap(err, "inference is invalid")
		}
	}
	if inputInferenceForecastBundle.Forecast != nil {
		if err := inputInferenceForecastBundle.Forecast.Validate(); err != nil {
			return errors.Wrap(err, "forecast is invalid")
		}
	}

	return nil
}

func (bundle *InputWorkerDataBundle) Validate() error {
	if bundle == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "worker data bundle cannot be nil")
	}
	if bundle.Nonce == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "worker data bundle nonce cannot be nil")
	}
	if err := bundle.Nonce.Validate(); err != nil {
		return errors.Wrap(err, "worker data bundle nonce is invalid")
	}
	// Additional validation on zero height for bundles
	if bundle.Nonce.BlockHeight <= 0 {
		return errors.Wrap(sdkerrors.ErrInvalidType, "worker data bundle nonce block height must be greater than 0")
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
		// Validate against the current bundle
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
		if bundle.TopicId != bundle.InferenceForecastsBundle.Inference.TopicId {
			return errors.Wrapf(sdkerrors.ErrInvalidRequest, "inference topic id %d does not match bundle topic id %d", bundle.InferenceForecastsBundle.Inference.TopicId, bundle.TopicId)
		}
		if bundle.Nonce.BlockHeight != bundle.InferenceForecastsBundle.Inference.BlockHeight {
			return errors.Wrapf(sdkerrors.ErrInvalidRequest, "inference block height %d does not match bundle block height %d", bundle.InferenceForecastsBundle.Inference.BlockHeight, bundle.Nonce.BlockHeight)
		}
	}
	if bundle.InferenceForecastsBundle.Forecast != nil {
		if err := bundle.InferenceForecastsBundle.Forecast.Validate(); err != nil {
			return err
		}
		// Validate against the current bundle
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
		if bundle.TopicId != bundle.InferenceForecastsBundle.Forecast.TopicId {
			return errors.Wrapf(sdkerrors.ErrInvalidRequest, "forecast topic id %d does not match bundle topic id %d", bundle.InferenceForecastsBundle.Forecast.TopicId, bundle.TopicId)
		}
		if bundle.Nonce.BlockHeight != bundle.InferenceForecastsBundle.Forecast.BlockHeight {
			return errors.Wrapf(sdkerrors.ErrInvalidRequest, "forecast block height %d does not match bundle block height %d", bundle.InferenceForecastsBundle.Forecast.BlockHeight, bundle.Nonce.BlockHeight)
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
		if bundle.Worker != bundle.InferenceForecastsBundle.Inference.Inferer {
			return errors.Wrapf(sdkerrors.ErrUnauthorized,
				"Inference.Inferer %s does not match worker address %s",
				bundle.InferenceForecastsBundle.Inference.Inferer, bundle.Worker)
		}
		if bundle.TopicId != bundle.InferenceForecastsBundle.Inference.TopicId {
			return errors.Wrapf(sdkerrors.ErrInvalidRequest, "inference topic id %d does not match bundle topic id %d", bundle.InferenceForecastsBundle.Inference.TopicId, bundle.TopicId)
		}
		if bundle.Nonce.BlockHeight != bundle.InferenceForecastsBundle.Inference.BlockHeight {
			return errors.Wrapf(sdkerrors.ErrInvalidRequest, "inference block height %d does not match bundle block height %d", bundle.InferenceForecastsBundle.Inference.BlockHeight, bundle.Nonce.BlockHeight)
		}
	}
	if bundle.InferenceForecastsBundle.Forecast != nil {
		if err := bundle.InferenceForecastsBundle.Forecast.Validate(); err != nil {
			return err
		}
		if bundle.Worker != bundle.InferenceForecastsBundle.Forecast.Forecaster {
			return errors.Wrapf(sdkerrors.ErrUnauthorized,
				"Forecast.Forecaster %s does not match worker address %s",
				bundle.InferenceForecastsBundle.Forecast.Forecaster, bundle.Worker)
		}
		if bundle.TopicId != bundle.InferenceForecastsBundle.Forecast.TopicId {
			return errors.Wrapf(sdkerrors.ErrInvalidRequest, "forecast topic id %d does not match bundle topic id %d", bundle.InferenceForecastsBundle.Forecast.TopicId, bundle.TopicId)
		}
		if bundle.Nonce.BlockHeight != bundle.InferenceForecastsBundle.Forecast.BlockHeight {
			return errors.Wrapf(sdkerrors.ErrInvalidRequest, "forecast block height %d does not match bundle block height %d", bundle.InferenceForecastsBundle.Forecast.BlockHeight, bundle.Nonce.BlockHeight)
		}
	}

	return nil
}

// REPUTER VALIDATIONS

func (bundle *InputValueBundle) Validate() error {
	if bundle == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "value bundle cannot be nil")
	}
	if err := ValidateTopicId(bundle.TopicId); err != nil {
		return errors.Wrap(err, "value bundle topic id is invalid")
	}
	if bundle.ReputerRequestNonce == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "value bundle reputer request nonce cannot be nil")
	}
	if err := bundle.ReputerRequestNonce.Validate(); err != nil {
		return errors.Wrap(err, "value bundle reputer request nonce is invalid")
	}
	// Additional validation on zero height for bundles
	if bundle.ReputerRequestNonce.ReputerNonce.BlockHeight <= 0 {
		return errors.Wrap(sdkerrors.ErrInvalidType, "value bundle reputer request nonce block height must be greater than 0")
	}
	if err := ValidateBech32(bundle.Reputer); err != nil {
		return errors.Wrap(err, "value bundle reputer address is invalid")
	}
	// extraData is not checked as it is not used by the chain

	if len(bundle.InfererValues) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "value bundle inferer values cannot be nil")
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
	if err := bundle.ReputerRequestNonce.Validate(); err != nil {
		return errors.Wrap(err, "value bundle reputer request nonce is invalid")
	}
	// Additional validation on zero height for bundles
	if bundle.ReputerRequestNonce.ReputerNonce.BlockHeight <= 0 {
		return errors.Wrap(sdkerrors.ErrInvalidType, "value bundle reputer request nonce block height must be greater than 0")
	}
	if err := ValidateBech32(bundle.Reputer); err != nil {
		return errors.Wrap(err, "value bundle reputer address is invalid")
	}
	// extraData is not checked as it is not used by the chain

	if err := ValidateDec(bundle.CombinedValue); err != nil {
		return errors.Wrap(err, "value bundle combined value is invalid")
	}

	if len(bundle.InfererValues) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "value bundle inferer values cannot be nil")
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

func FromLabeledValues(lvals []*LabeledValue) (out alloraMath.DecArray) {
	out = make(alloraMath.DecArray, len(lvals))
	for i := range lvals {
		out[i] = lvals[i].Value
	}
	return
}

// validate that a network inference bundle follows the expected format
func (bundle *NetworkInferenceBundle) Validate() error {
	if bundle == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "value bundle cannot be nil")
	}
	if err := ValidateTopicId(bundle.TopicId); err != nil {
		return errors.Wrap(err, "value bundle topic id is invalid")
	}
	if err := ValidateBlockHeight(bundle.Nonce); err != nil {
		return errors.Wrap(err, "value bundle nonce is invalid")
	}
	if bundle.Nonce <= 0 {
		return errors.Wrap(sdkerrors.ErrInvalidType, "value bundle reputer request nonce block height must be greater than 0")
	}

	combinedValue := FromLabeledValues(bundle.CombinedValue)

	if err := ValidateDecs(combinedValue); err != nil {
		return errors.Wrap(err, "value bundle combined value is invalid")
	}

	if len(bundle.InfererValues) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "value bundle inferer values cannot be nil")
	}

	// nil values for bundle.InfererValues are interpreted to mean that there
	// are no inferer values for this bundle, and are allowed
	for _, infererValue := range bundle.InfererValues {
		vals := FromLabeledValues(infererValue.GetValues())
		if err := ValidateDecs(vals); err != nil {
			return errors.Wrap(err, "value bundle inferer value is invalid")
		}
	}

	// nil values for bundle.ForecasterValues are interpreted to mean that there
	// are no forecaster values for this bundle, and are allowed
	for _, forecasterValue := range bundle.ForecasterValues {
		vals := FromLabeledValues(forecasterValue.GetValues())
		if err := ValidateDecs(vals); err != nil {
			return errors.Wrap(err, "value bundle forecaster value is invalid")
		}
	}

	naiveValue := FromLabeledValues(bundle.NaiveValue)
	if err := ValidateDecs(naiveValue); err != nil {
		return errors.Wrap(err, "value bundle naive value is invalid")
	}

	// nil values for bundle.OneOutInfererValues are interpreted to mean that there
	// are no one out inferer values for this bundle, and are allowed
	for _, oneOutInfererValue := range bundle.OneOutInfererValues {
		vals := FromLabeledValues(oneOutInfererValue.GetCombinedInference())
		if err := ValidateDecs(vals); err != nil {
			return errors.Wrap(err, "value bundle one out inferer value is invalid")
		}
	}

	// nil values for bundle.OneOutForecasterValues are interpreted to mean that there
	// are no one out forecaster values for this bundle, and are allowed
	for _, oneOutForecasterValue := range bundle.OneOutForecasterValues {
		vals := FromLabeledValues(oneOutForecasterValue.GetCombinedInference())
		if err := ValidateDecs(vals); err != nil {
			return errors.Wrap(err, "value bundle one out forecaster value is invalid")
		}
	}

	// nil values for bundle.OneInForecasterValues are interpreted to mean that there
	// are no one in forecaster values for this bundle, and are allowed
	for _, oneInForecasterValue := range bundle.OneInForecasterValues {
		vals := FromLabeledValues(oneInForecasterValue.GetCombinedInference())
		if err := ValidateDecs(vals); err != nil {
			return errors.Wrap(err, "value bundle one in forecaster value is invalid")
		}
	}

	// nil values for bundle.OneOutInfererForecasterValues are interpreted to mean that there
	// are no one out inferer forecaster values for this bundle, and are allowed
	for _, oneOutInfererForecaster := range bundle.OneOutInfererForecasterValues {
		vals := FromLabeledValues(oneOutInfererForecaster.GetCombinedInference())
		if err := ValidateDecs(vals); err != nil {
			return errors.Wrap(err, "value bundle one out inferer forecaster value is invalid")
		}
	}
	return nil
}

// validate that a reputer value bundle follows the expected format
func (bundle *InputReputerValueBundle) Validate() error {
	if bundle == nil {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "reputer value bundle cannot be nil")
	}
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

// validate that a reputer value bundle follows the expected format
func (bundle *ReputerValueBundle) Validate() error {
	if bundle == nil {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "reputer value bundle cannot be nil")
	}
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

// validate that a worker attributed value follows the expected format
func (inputWorkerValue *InputWorkerAttributedValue) Validate() error {
	if inputWorkerValue == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "input worker attributed value cannot be nil")
	}
	_, err := sdk.AccAddressFromBech32(inputWorkerValue.Worker)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid worker address (%s)", err)
	}

	return nil
}

// validate that a worker attributed value follows the expected format
func (workerValue *WorkerAttributedValue) Validate() error {
	if workerValue == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "worker attributed value cannot be nil")
	}
	_, err := sdk.AccAddressFromBech32(workerValue.Worker)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid worker address (%s)", err)
	}

	if err := ValidateDec(workerValue.Value); err != nil {
		return err
	}

	return nil
}

func (inputWithheldWorkerValue *InputWithheldWorkerAttributedValue) Validate() error {
	if inputWithheldWorkerValue == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "input withheld worker attributed value cannot be nil")
	}
	_, err := sdk.AccAddressFromBech32(inputWithheldWorkerValue.Worker)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid withheld worker address (%s)", err)
	}

	return nil
}

// validate that a withheld worker attributed value follows the expected format
func (withheldWorkerValue *WithheldWorkerAttributedValue) Validate() error {
	if withheldWorkerValue == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "withheld worker attributed value cannot be nil")
	}
	_, err := sdk.AccAddressFromBech32(withheldWorkerValue.Worker)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid withheld worker address (%s)", err)
	}

	if err := ValidateDec(withheldWorkerValue.Value); err != nil {
		return err
	}

	return nil
}

func (inputOneOutInfererForecasterValues *InputOneOutInfererForecasterValues) Validate() error {
	if inputOneOutInfererForecasterValues == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "input one out inferer forecaster values cannot be nil")
	}
	if err := ValidateBech32(inputOneOutInfererForecasterValues.Forecaster); err != nil {
		return errors.Wrap(err, "one out inferer forecaster values forecaster is invalid")
	}
	if inputOneOutInfererForecasterValues.OneOutInfererValues == nil {
		return errors.Wrap(sdkerrors.ErrInvalidType, "one out inferer forecaster values one out inferer values cannot be nil")
	}
	for _, oneOutInfererValue := range inputOneOutInfererForecasterValues.OneOutInfererValues {
		if err := oneOutInfererValue.Validate(); err != nil {
			return errors.Wrap(err, "one out inferer forecaster values one out inferer value is invalid")
		}
	}
	return nil
}

// validate that a types.OneOutInfererForecasterValues follows the expected format
func (oneOutInfererForecasterValues *OneOutInfererForecasterValues) Validate() error {
	if oneOutInfererForecasterValues == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "one out inferer forecaster values cannot be nil")
	}
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

// validate that a types.InputReputerValueBundles follows the expected format
func (inputReputerValueBundles *InputReputerValueBundles) Validate() error {
	if inputReputerValueBundles == nil {
		return errors.Wrapf(sdkerrors.ErrInvalidType, "reputer value bundles cannot be nil")
	}
	for i, reputerValueBundle := range inputReputerValueBundles.ReputerValueBundles {
		if err := reputerValueBundle.Validate(); err != nil {
			return errors.Wrapf(err, "reputer value bundle at index %d is invalid", i)
		}
	}
	return nil
}

// validate that a types.ReputerValueBundles follows the expected format
func (bundle LossBundles) Validate() error {
	if bundle == nil {
		return errors.Wrapf(sdkerrors.ErrInvalidType, "loss bundles cannot be nil")
	}
	for i, reputerValueBundle := range bundle {
		if err := reputerValueBundle.Validate(); err != nil {
			return errors.Wrapf(err, "loss bundle at index %d is invalid", i)
		}
	}
	return nil
}

const (
	maxTopicUnityTolerance = "0.01"
)

// EffectiveMaxTopInferersToReward resolves a per-topic cap against the global:
// 0 means "use the global", and any value is clamped to the global ceiling.
func EffectiveMaxTopInferersToReward(topicCap, globalMax uint64) uint64 {
	if topicCap == 0 {
		return globalMax
	}
	return min(topicCap, globalMax)
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
	if len(topic.LossMethod) < validationMinLossMethodLength || uint64(len(topic.LossMethod)) > params.MaxStringLength {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic loss method invalid")
	}
	if topic.EpochLastEnded < validationMinEpochLastEnded {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic epoch last ended cannot be negative")
	}
	if topic.EpochLength < validationMinPositiveIntValue {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic epoch length must be greater than zero")
	}
	if topic.EpochLength < params.MinEpochLength {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic epoch length must be greater than minimum epoch length")
	}
	if topic.WorkerSubmissionWindow < validationMinPositiveIntValue {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic worker submission window must be greater than zero")
	}
	if topic.GroundTruthLag < validationMinPositiveIntValue {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic ground truth lag must be greater than zero")
	}
	if topic.GroundTruthLag < topic.EpochLength {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic ground truth lag cannot be lower than epoch length")
	}
	if uint64(topic.GroundTruthLag) > params.MaxUnfulfilledReputerRequests*uint64(topic.EpochLength) {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic ground truth lag cannot be higher than max unfulfilled reputer requests")
	}
	if topic.WorkerSubmissionWindow < validationMinPositiveIntValue {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic worker submission window must be greater than zero")
	}
	if topic.WorkerSubmissionWindow > topic.EpochLength {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic worker submission window cannot be higher than epoch length")
	}
	if err := ValidateDec(topic.AlphaRegret); err != nil {
		return errors.Wrap(err, "topic alpha regret is invalid")
	}
	if topic.AlphaRegret.Lte(validationZeroDec) || topic.AlphaRegret.Gt(validationOneDec) {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic alpha regret must be greater than 0 and less than or equal to 1")
	}
	if err := ValidateDec(topic.PNorm); err != nil {
		return errors.Wrap(err, "topic p-norm is invalid")
	}
	if topic.PNorm.Lt(validationOneDec) || topic.PNorm.Gt(validationPNormMaxDec) {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic p-norm must be between 1 and 10")
	}
	if err := ValidateDec(topic.Epsilon); err != nil {
		return errors.Wrap(err, "topic epsilon is invalid")
	}
	if topic.Epsilon.Lte(validationZeroDec) {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic epsilon must be greater than 0")
	}
	// no validation on AllowNegative because either it is true or false
	// and both are valid values
	//	AllowNegative            bool
	if err := ValidateDec(topic.MeritSortitionAlpha); err != nil {
		return errors.Wrap(err, "topic merit sortition alpha is invalid")
	}
	if !isAlloraDecZeroOrLessThanOne(topic.MeritSortitionAlpha) {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic merit sortition alpha must be greater than or equal to 0 and less than 1")
	}
	if err := ValidateDec(topic.ActiveInfererQuantile); err != nil {
		return errors.Wrap(err, "topic active inferer quantile is invalid")
	}
	if !isAlloraDecBetweenZeroAndOneInclusive(topic.ActiveInfererQuantile) {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic active inferer quantile must be between 0 and 1 inclusive")
	}
	if err := ValidateDec(topic.ActiveForecasterQuantile); err != nil {
		return errors.Wrap(err, "topic active forecaster quantile is invalid")
	}
	if !isAlloraDecBetweenZeroAndOneInclusive(topic.ActiveForecasterQuantile) {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic active forecaster quantile must be between 0 and 1 inclusive")
	}
	if err := ValidateDec(topic.ActiveReputerQuantile); err != nil {
		return errors.Wrap(err, "topic active reputer quantile is invalid")
	}
	if !isAlloraDecBetweenZeroAndOneInclusive(topic.ActiveReputerQuantile) {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic active reputer quantile must be between 0 and 1 inclusive")
	}
	if err := ValidateDec(topic.CNorm); err != nil {
		return errors.Wrap(err, "topic c_norm is invalid")
	}
	if topic.CNorm.Lt(validationCNormMinDec) || topic.CNorm.Gt(validationCNormMaxDec) {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic c_norm must be between -100 and 100")
	}
	// The per-topic inferer cap is not bounded by the global here: the global
	// bounds admission via min(cap, global), not storage. Zero and above-global
	// stored values are valid at rest.
	if err := ValidateClassificationConsistency(
		topic.TopicType,
		topic.OutputArity,
		topic.RequireUnity,
		topic.UnityTolerance,
		topic.LabelDefaultValue,
	); err != nil {
		return err
	}
	if err := ValidateMaxLabelsPerSubmission(topic.MaxLabelsPerSubmission); err != nil {
		return errors.Wrap(err, "topic max_labels_per_submission is invalid")
	}
	if err := ValidateTopicLabelWhitelistSize(topic.LabelWhitelist, params.MaxTopicLabelWhitelistSize); err != nil {
		return errors.Wrap(err, "topic label_whitelist is invalid")
	}

	return nil
}

// ValidateClassificationConsistency enforces the topic_type / output_arity /
// unity invariants shared by Topic.Validate (run by SetTopic before
// persistence) and CreateNewTopicRequest.Validate, so the message layer and
// the keeper layer cannot drift. Field validity is checked before the
// cross-field consistency rules that depend on it.
//
// UpdateTopicRequest validates only the classification field it can mutate
// (currently label_default_value); topic_type/output_arity/require_unity are
// immutable after creation and absent from that message, so their consistency
// is enforced by Topic.Validate on the merged topic in the keeper.
func ValidateClassificationConsistency(
	topicType TopicType,
	outputArity TopicOutputArity,
	requireUnity bool,
	unityTolerance alloraMath.Dec,
	labelDefaultValue alloraMath.Dec,
) error {
	if topicType <= TopicType_TOPIC_TYPE_UNSPECIFIED || topicType > TopicType_TOPIC_TYPE_CLASSIFICATION {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic_type is invalid")
	}
	if outputArity <= TopicOutputArity_TOPIC_OUTPUT_ARITY_UNSPECIFIED || outputArity > TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI {
		return errors.Wrap(sdkerrors.ErrInvalidType, "output_arity is invalid")
	}
	if outputArity == TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE && requireUnity {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic require_unity MUST be false when output_arity is SINGLE")
	}
	if requireUnity &&
		(unityTolerance.IsNaN() ||
			unityTolerance.Lt(alloraMath.ZeroDec()) ||
			unityTolerance.Gt(alloraMath.MustNewDecFromString(maxTopicUnityTolerance))) {
		return errors.Wrapf(sdkerrors.ErrInvalidType,
			"unity_tolerance must be in (0, %s] when require_unity is true", maxTopicUnityTolerance)
	}
	if err := ValidateDec(labelDefaultValue); err != nil {
		return errors.Wrap(err, "topic label_default_value is invalid")
	}
	if requireUnity && !labelDefaultValue.IsZero() {
		return errors.Wrap(sdkerrors.ErrInvalidType, "topic label_default_value must be zero when require_unity is true")
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

// / PROTOBUF MESSAGE VALIDATIONS

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

// validate that a transfer actor ownership request follows the expected format
func (msg *TransferActorOwnershipRequest) Validate() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid sender address (%s)", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.NewOwner); err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid new owner address (%s)", err)
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
	if err := ValidateSdkIntRepresentingMonetaryValue(amount); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidCoins, "amount is not valid")
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
func (msg *CreateNewTopicRequest) Validate(maxStringLen uint64, maxTopicLabelWhitelistSize uint64) error {
	if err := ValidateBech32(msg.Creator); err != nil {
		return errors.Wrap(err, "invalid msg Creator address")
	}

	if len(msg.LossMethod) < validationMinLossMethodLength || uint64(len(msg.LossMethod)) > maxStringLen {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "loss method invalid")
	}
	if msg.EpochLength < validationMinPositiveIntValue {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "epoch length must be greater than zero")
	}
	if msg.WorkerSubmissionWindow < validationMinPositiveIntValue {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "worker submission window must be greater than zero")
	}
	if msg.GroundTruthLag < msg.EpochLength {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "ground truth lag cannot be lower than epoch length")
	}
	if msg.WorkerSubmissionWindow > msg.EpochLength {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "worker submission window cannot be higher than epoch length")
	}
	if msg.AlphaRegret.Lte(validationZeroDec) || msg.AlphaRegret.Gt(validationOneDec) || ValidateDec(msg.AlphaRegret) != nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "alpha regret must be greater than 0 and less than or equal to 1")
	}
	if msg.PNorm.Lt(validationOneDec) || msg.PNorm.Gt(validationPNormMaxDec) || ValidateDec(msg.PNorm) != nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "p-norm must be between 1 and 10")
	}
	if msg.CNorm.Lt(validationCNormMinDec) || msg.CNorm.Gt(validationCNormMaxDec) || ValidateDec(msg.CNorm) != nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "c_norm must be between -100 and 100")
	}
	if msg.Epsilon.Lte(validationZeroDec) || msg.Epsilon.IsNaN() || !msg.Epsilon.IsFinite() {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "epsilon must be greater than 0")
	}
	if uint64(len(msg.Metadata)) > maxStringLen {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "metadata invalid")
	}
	// no validation on AllowNegative because either it is true or false
	// and both are valid values
	//	AllowNegative            bool

	if !isAlloraDecZeroOrLessThanOne(msg.MeritSortitionAlpha) {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "merit sortition alpha must be greater than or equal to 0 and less than 1")
	}
	if !isAlloraDecBetweenZeroAndOneInclusive(msg.ActiveInfererQuantile) {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "active inferer quantile must be between 0 and 1 inclusive")
	}
	if !isAlloraDecBetweenZeroAndOneInclusive(msg.ActiveForecasterQuantile) {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "active forecaster quantile must be between 0 and 1 inclusive")
	}
	if !isAlloraDecBetweenZeroAndOneInclusive(msg.ActiveReputerQuantile) {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "active reputer quantile must be between 0 and 1 inclusive")
	}
	if err := ValidateClassificationConsistency(
		msg.TopicType,
		msg.OutputArity,
		msg.RequireUnity,
		msg.UnityTolerance,
		msg.LabelDefaultValue,
	); err != nil {
		return err
	}
	if err := ValidateMaxLabelsPerSubmission(msg.MaxLabelsPerSubmission); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, err.Error())
	}
	if err := ValidateTopicLabelWhitelistSize(msg.LabelWhitelist, maxTopicLabelWhitelistSize); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, err.Error())
	}

	return nil
}

// Validate checks if the given UpdateTopicRequest is valid
func (msg *UpdateTopicRequest) Validate(maxStringLen uint64, maxTopicLabelWhitelistSize uint64) error {
	if err := ValidateBech32(msg.Sender); err != nil {
		return errors.Wrap(err, "invalid msg Sender address")
	}
	if len(msg.LossMethod) < validationMinLossMethodLength || uint64(len(msg.LossMethod)) > maxStringLen {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "loss method invalid")
	}
	if msg.AlphaRegret.Lte(validationZeroDec) || msg.AlphaRegret.Gt(validationOneDec) || ValidateDec(msg.AlphaRegret) != nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "alpha regret must be greater than 0 and less than or equal to 1")
	}
	if msg.PNorm.Lt(validationOneDec) || msg.PNorm.Gt(validationPNormMaxDec) || ValidateDec(msg.PNorm) != nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "p-norm must be between 1 and 10")
	}
	if msg.CNorm.Lt(validationCNormMinDec) || msg.CNorm.Gt(validationCNormMaxDec) || ValidateDec(msg.CNorm) != nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "c_norm must be between -100 and 100")
	}
	if uint64(len(msg.Metadata)) > maxStringLen {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "metadata invalid")
	}
	if !isAlloraDecZeroOrLessThanOne(msg.MeritSortitionAlpha) {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "merit sortition alpha must be greater than or equal to 0 and less than 1")
	}
	// label_default_value is the only classification field UpdateTopicRequest can
	// mutate; topic_type/output_arity/require_unity are immutable after creation
	// and absent here, so the require_unity coupling is enforced by Topic.Validate
	// on the merged topic in the keeper. If those fields become mutable on update,
	// switch this to ValidateClassificationConsistency.
	if err := ValidateDec(msg.LabelDefaultValue); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "label_default_value is invalid")
	}
	if err := ValidateMaxLabelsPerSubmission(msg.MaxLabelsPerSubmission); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, err.Error())
	}
	if err := ValidateTopicLabelWhitelistSize(msg.LabelWhitelist, maxTopicLabelWhitelistSize); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, err.Error())
	}

	return nil
}

// ValidateInferenceValues verifies that the inference values are consistent
// with the provided epoch label registry.
func ValidateInferenceValues(iv InferenceValues, labels []*TopicLabel) error {
	want := len(labels)
	if len(iv) != want {
		return errors.Wrapf(
			sdkerrors.ErrLogic,
			"inference values length mismatch: got=%d want=%d",
			len(iv), want,
		)
	}
	for i := range iv {
		if iv[i].IsNaN() || !iv[i].IsFinite() {
			return errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid inference value at idx=%d", i)
		}
	}
	return nil
}

func ValidateStringIsBech32(actor string) error {
	_, err := sdk.AccAddressFromBech32(actor)
	if err != nil {
		return errors.Wrap(err, "error validating actor id")
	}
	return nil
}
