package types

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	alloraMath "github.com/allora-network/allora-chain/math"
)

func validCreateNewTopicRequest() *CreateNewTopicRequest {
	return &CreateNewTopicRequest{
		MaxTopInferersToReward:   0,
		Creator:                  "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
		Metadata:                 "metadata",
		LossMethod:               "mse",
		EpochLength:              10800,
		GroundTruthLag:           10800,
		PNorm:                    alloraMath.MustNewDecFromString("3"),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:            false,
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		WorkerSubmissionWindow:   10,
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		EnableWorkerWhitelist:    false,
		EnableReputerWhitelist:   false,
		CNorm:                    alloraMath.MustNewDecFromString("0.75"),
		TopicType:                TopicType_TOPIC_TYPE_REGRESSION,
		OutputArity:              TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
		RequireUnity:             false,
		UnityTolerance:           alloraMath.Dec{},
		MaxLabelsPerSubmission:   DefaultMaxLabelsPerSubmission,
		LabelDefaultValue:        alloraMath.ZeroDec(),
		LabelCaseSensitive:       false,
		LabelWhitelist:           nil,
	}
}

func TestCreateNewTopicRequest_Validate(t *testing.T) {
	const maxStringLen uint64 = 256

	tests := []struct {
		name        string
		mutate      func(*CreateNewTopicRequest)
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid request",
			mutate:      func(*CreateNewTopicRequest) {},
			wantErr:     false,
			errContains: "",
		},
		{
			name: "label whitelist above max",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.LabelWhitelist = make([]string, DefaultMaxTopicLabelWhitelistSize+1)
			},
			wantErr:     true,
			errContains: "topic label whitelist size",
		},
		{
			name: "empty loss method",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.LossMethod = ""
			},
			wantErr:     true,
			errContains: "loss method invalid",
		},
		{
			name: "loss method too long",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.LossMethod = strings.Repeat("a", int(maxStringLen)+1)
			},
			wantErr:     true,
			errContains: "loss method invalid",
		},
		{
			name: "metadata too long",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.Metadata = strings.Repeat("a", int(maxStringLen)+1)
			},
			wantErr:     true,
			errContains: "metadata invalid",
		},
		{
			name: "zero epoch length",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.EpochLength = 0
			},
			wantErr:     true,
			errContains: "epoch length must be greater than zero",
		},
		{
			name: "negative epoch length",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.EpochLength = -1
			},
			wantErr:     true,
			errContains: "epoch length must be greater than zero",
		},
		{
			name: "zero worker submission window",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.WorkerSubmissionWindow = 0
			},
			wantErr:     true,
			errContains: "worker submission window must be greater than zero",
		},
		{
			name: "negative worker submission window",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.WorkerSubmissionWindow = -1
			},
			wantErr:     true,
			errContains: "worker submission window must be greater than zero",
		},
		{
			name: "worker submission window greater than epoch length",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.WorkerSubmissionWindow = msg.EpochLength + 1
			},
			wantErr:     true,
			errContains: "worker submission window cannot be higher than epoch length",
		},
		{
			name: "ground truth lag lower than epoch length",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.GroundTruthLag = msg.EpochLength - 1
			},
			wantErr:     true,
			errContains: "ground truth lag cannot be lower than epoch length",
		},
		{
			name: "zero alpha regret",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.AlphaRegret = alloraMath.ZeroDec()
			},
			wantErr:     true,
			errContains: "alpha regret must be greater than 0 and less than or equal to 1",
		},
		{
			name: "negative alpha regret",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.AlphaRegret = alloraMath.MustNewDecFromString("-0.1")
			},
			wantErr:     true,
			errContains: "alpha regret must be greater than 0 and less than or equal to 1",
		},
		{
			name: "alpha regret greater than one",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.AlphaRegret = alloraMath.MustNewDecFromString("1.1")
			},
			wantErr:     true,
			errContains: "alpha regret must be greater than 0 and less than or equal to 1",
		},
		{
			name: "p-norm below one",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.PNorm = alloraMath.MustNewDecFromString("0.9")
			},
			wantErr:     true,
			errContains: "p-norm must be between 1 and 10",
		},
		{
			name: "p-norm above ten",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.PNorm = alloraMath.MustNewDecFromString("10.1")
			},
			wantErr:     true,
			errContains: "p-norm must be between 1 and 10",
		},
		{
			name: "c-norm below negative one hundred",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.CNorm = alloraMath.MustNewDecFromString("-100.1")
			},
			wantErr:     true,
			errContains: "c_norm must be between -100 and 100",
		},
		{
			name: "c-norm above one hundred",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.CNorm = alloraMath.MustNewDecFromString("100.1")
			},
			wantErr:     true,
			errContains: "c_norm must be between -100 and 100",
		},
		{
			name: "zero epsilon",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.Epsilon = alloraMath.ZeroDec()
			},
			wantErr:     true,
			errContains: "epsilon must be greater than 0",
		},
		{
			name: "negative epsilon",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.Epsilon = alloraMath.MustNewDecFromString("-0.1")
			},
			wantErr:     true,
			errContains: "epsilon must be greater than 0",
		},
		{
			name: "negative merit sortition alpha",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.MeritSortitionAlpha = alloraMath.MustNewDecFromString("-0.1")
			},
			wantErr:     true,
			errContains: "merit sortition alpha must be greater than or equal to 0 and less than 1",
		},
		{
			name: "zero merit sortition alpha",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.MeritSortitionAlpha = alloraMath.ZeroDec()
			},
			wantErr:     false,
			errContains: "",
		},
		{
			name: "merit sortition alpha equal to one",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.MeritSortitionAlpha = alloraMath.OneDec()
			},
			wantErr:     true,
			errContains: "merit sortition alpha must be greater than or equal to 0 and less than 1",
		},
		{
			name: "negative active inferer quantile",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.ActiveInfererQuantile = alloraMath.MustNewDecFromString("-0.1")
			},
			wantErr:     true,
			errContains: "active inferer quantile must be between 0 and 1 inclusive",
		},
		{
			name: "active inferer quantile equal to zero",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.ActiveInfererQuantile = alloraMath.ZeroDec()
			},
			wantErr:     false,
			errContains: "",
		},
		{
			name: "active inferer quantile equal to one",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.ActiveInfererQuantile = alloraMath.OneDec()
			},
			wantErr:     false,
			errContains: "",
		},
		{
			name: "active inferer quantile greater than one",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.ActiveInfererQuantile = alloraMath.MustNewDecFromString("1.1")
			},
			wantErr:     true,
			errContains: "active inferer quantile must be between 0 and 1 inclusive",
		},
		{
			name: "negative active forecaster quantile",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.ActiveForecasterQuantile = alloraMath.MustNewDecFromString("-0.1")
			},
			wantErr:     true,
			errContains: "active forecaster quantile must be between 0 and 1 inclusive",
		},
		{
			name: "active forecaster quantile equal to zero",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.ActiveForecasterQuantile = alloraMath.ZeroDec()
			},
			wantErr:     false,
			errContains: "",
		},
		{
			name: "active forecaster quantile equal to one",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.ActiveForecasterQuantile = alloraMath.OneDec()
			},
			wantErr:     false,
			errContains: "",
		},
		{
			name: "active forecaster quantile greater than one",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.ActiveForecasterQuantile = alloraMath.MustNewDecFromString("1.1")
			},
			wantErr:     true,
			errContains: "active forecaster quantile must be between 0 and 1 inclusive",
		},
		{
			name: "negative active reputer quantile",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.ActiveReputerQuantile = alloraMath.MustNewDecFromString("-0.1")
			},
			wantErr:     true,
			errContains: "active reputer quantile must be between 0 and 1 inclusive",
		},
		{
			name: "active reputer quantile equal to zero",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.ActiveReputerQuantile = alloraMath.ZeroDec()
			},
			wantErr:     false,
			errContains: "",
		},
		{
			name: "active reputer quantile equal to one",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.ActiveReputerQuantile = alloraMath.OneDec()
			},
			wantErr:     false,
			errContains: "",
		},
		{
			name: "active reputer quantile greater than one",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.ActiveReputerQuantile = alloraMath.MustNewDecFromString("1.1")
			},
			wantErr:     true,
			errContains: "active reputer quantile must be between 0 and 1 inclusive",
		},
		{
			name: "unspecified topic_type",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.TopicType = TopicType_TOPIC_TYPE_UNSPECIFIED
			},
			wantErr:     true,
			errContains: "topic_type is invalid",
		},
		{
			name: "unspecified output_arity",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.OutputArity = TopicOutputArity_TOPIC_OUTPUT_ARITY_UNSPECIFIED
			},
			wantErr:     true,
			errContains: "output_arity is invalid",
		},
		{
			name: "require_unity with SINGLE arity",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.OutputArity = TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE
				msg.RequireUnity = true
			},
			wantErr:     true,
			errContains: "require_unity MUST be false when output_arity is SINGLE",
		},
		{
			name: "require_unity with unity tolerance NaN",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.OutputArity = TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI
				msg.RequireUnity = true
				msg.UnityTolerance = alloraMath.NewNaN()
			},
			wantErr:     true,
			errContains: "unity_tolerance must be in",
		},
		{
			name: "require_unity with unity tolerance above max",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.OutputArity = TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI
				msg.RequireUnity = true
				msg.UnityTolerance = alloraMath.MustNewDecFromString("1")
			},
			wantErr:     true,
			errContains: "unity_tolerance must be in",
		},
		{
			name: "require_unity with nonzero label_default_value",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.OutputArity = TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI
				msg.RequireUnity = true
				msg.UnityTolerance = alloraMath.MustNewDecFromString("0.005")
				msg.LabelDefaultValue = alloraMath.OneDec()
			},
			wantErr:     true,
			errContains: "label_default_value must be zero when require_unity is true",
		},
		{
			name: "valid require_unity with MULTI arity",
			mutate: func(msg *CreateNewTopicRequest) {
				msg.OutputArity = TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI
				msg.RequireUnity = true
				msg.UnityTolerance = alloraMath.MustNewDecFromString("0.005")
				msg.LabelDefaultValue = alloraMath.ZeroDec()
			},
			wantErr:     false,
			errContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := validCreateNewTopicRequest()
			tt.mutate(msg)

			err := msg.Validate(maxStringLen, DefaultMaxTopicLabelWhitelistSize)
			if tt.wantErr {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestInputInference_Validate(t *testing.T) {
	tests := []struct {
		name    string
		input   *InputInference
		wantErr bool
	}{
		{
			name: "valid inference",
			input: &InputInference{
				TopicId:     1,
				BlockHeight: 100,
				Inferer:     "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy", // valid bech32 address
				Value:       alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
				Values:      []*InputLabeledValue{{Label: "", Value: mustNewBoundedExp40Dec(t, "1")}},
				ExtraData:   nil,
				Proof:       "",
			},
			wantErr: false,
		},
		{
			name:    "nil inference",
			input:   nil,
			wantErr: true,
		},
		{
			name: "invalid topic id",
			input: &InputInference{
				TopicId:     0, // invalid
				Inferer:     "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
				BlockHeight: 100,
				Value:       alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
				Values:      []*InputLabeledValue{{Label: "", Value: mustNewBoundedExp40Dec(t, "1")}},
				ExtraData:   nil,
				Proof:       "",
			},
			wantErr: true,
		},
		{
			name: "invalid inferer address",
			input: &InputInference{
				TopicId:     1,
				Inferer:     "invalid",
				BlockHeight: 100,
				Value:       alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
				Values:      []*InputLabeledValue{{Label: "", Value: mustNewBoundedExp40Dec(t, "1")}},
				ExtraData:   nil,
				Proof:       "",
			},
			wantErr: true,
		},
		{
			name: "negative block height",
			input: &InputInference{
				TopicId:     1,
				Inferer:     "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
				BlockHeight: -1,
				Value:       alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
				Values:      []*InputLabeledValue{{Label: "", Value: mustNewBoundedExp40Dec(t, "1")}},
				ExtraData:   nil,
				Proof:       "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestInputForecast_Validate(t *testing.T) {
	validElement := &InputForecastElement{
		Inferer: "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
		Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
	}

	tests := []struct {
		name    string
		input   *InputForecast
		wantErr bool
	}{
		{
			name: "valid forecast",
			input: &InputForecast{
				TopicId:          1,
				Forecaster:       "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
				BlockHeight:      100,
				ForecastElements: []*InputForecastElement{validElement},
				ExtraData:        nil,
			},
			wantErr: false,
		},
		{
			name:    "nil forecast",
			input:   nil,
			wantErr: true,
		},
		{
			name: "empty forecast elements",
			input: &InputForecast{
				TopicId:          1,
				Forecaster:       "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
				BlockHeight:      100,
				ForecastElements: []*InputForecastElement{},
				ExtraData:        nil,
			},
			wantErr: true,
		},
		{
			name: "invalid forecast element",
			input: &InputForecast{
				TopicId:     1,
				Forecaster:  "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
				BlockHeight: 100,
				ForecastElements: []*InputForecastElement{{
					Inferer: "invalid",
					Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
				}},
				ExtraData: nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestInputValueBundle_Validate(t *testing.T) {
	validInfererValue := &InputWorkerAttributedValue{
		Worker: "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
		Value:  alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
	}

	tests := []struct {
		name    string
		input   *InputValueBundle
		wantErr bool
	}{
		{
			name: "valid value bundle",
			input: &InputValueBundle{
				TopicId:                       1,
				ReputerRequestNonce:           &ReputerRequestNonce{ReputerNonce: &Nonce{BlockHeight: 100}},
				Reputer:                       "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
				CombinedValue:                 alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
				NaiveValue:                    alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
				InfererValues:                 []*InputWorkerAttributedValue{validInfererValue},
				ForecasterValues:              nil,
				OneOutInfererValues:           nil,
				OneOutForecasterValues:        nil,
				OneInForecasterValues:         nil,
				OneOutInfererForecasterValues: nil,
				ExtraData:                     nil,
			},
			wantErr: false,
		},
		{
			name:    "nil value bundle",
			input:   nil,
			wantErr: true,
		},
		{
			name: "nil reputer request nonce",
			input: &InputValueBundle{
				TopicId:                       1,
				ReputerRequestNonce:           nil,
				Reputer:                       "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
				CombinedValue:                 alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
				NaiveValue:                    alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
				InfererValues:                 []*InputWorkerAttributedValue{validInfererValue},
				ForecasterValues:              nil,
				OneOutInfererValues:           nil,
				OneOutForecasterValues:        nil,
				OneInForecasterValues:         nil,
				OneOutInfererForecasterValues: nil,
				ExtraData:                     nil,
			},
			wantErr: true,
		},
		{
			name: "zero block height in reputer request nonce",
			input: &InputValueBundle{
				TopicId:                       1,
				ReputerRequestNonce:           &ReputerRequestNonce{ReputerNonce: &Nonce{BlockHeight: 0}},
				Reputer:                       "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
				CombinedValue:                 alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
				NaiveValue:                    alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
				InfererValues:                 []*InputWorkerAttributedValue{validInfererValue},
				ForecasterValues:              nil,
				OneOutInfererValues:           nil,
				OneOutForecasterValues:        nil,
				OneInForecasterValues:         nil,
				OneOutInfererForecasterValues: nil,
				ExtraData:                     nil,
			},
			wantErr: true,
		},
		{
			name: "nil inferer values",
			input: &InputValueBundle{
				TopicId:                       1,
				ReputerRequestNonce:           &ReputerRequestNonce{ReputerNonce: &Nonce{BlockHeight: 100}},
				Reputer:                       "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
				CombinedValue:                 alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
				NaiveValue:                    alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
				InfererValues:                 nil,
				ForecasterValues:              nil,
				OneOutInfererValues:           nil,
				OneOutForecasterValues:        nil,
				OneInForecasterValues:         nil,
				OneOutInfererForecasterValues: nil,
				ExtraData:                     nil,
			},
			wantErr: true,
		},
		{
			name: "empty inferer values",
			input: &InputValueBundle{
				TopicId:                       1,
				ReputerRequestNonce:           &ReputerRequestNonce{ReputerNonce: &Nonce{BlockHeight: 100}},
				Reputer:                       "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
				CombinedValue:                 alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
				NaiveValue:                    alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
				InfererValues:                 []*InputWorkerAttributedValue{},
				ForecasterValues:              nil,
				OneOutInfererValues:           nil,
				OneOutForecasterValues:        nil,
				OneInForecasterValues:         nil,
				OneOutInfererForecasterValues: nil,
				ExtraData:                     nil,
			},
			wantErr: true,
		},
		{
			name: "negative combined value valid",
			input: &InputValueBundle{
				TopicId:                       1,
				ReputerRequestNonce:           &ReputerRequestNonce{ReputerNonce: &Nonce{BlockHeight: 100}},
				Reputer:                       "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
				CombinedValue:                 alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("-1")),
				NaiveValue:                    alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
				InfererValues:                 []*InputWorkerAttributedValue{validInfererValue},
				ForecasterValues:              nil,
				OneOutInfererValues:           nil,
				OneOutForecasterValues:        nil,
				OneInForecasterValues:         nil,
				OneOutInfererForecasterValues: nil,
				ExtraData:                     nil,
			},
			wantErr: false,
		},
		{
			name: "empty combined value means 0 thus valid",
			input: &InputValueBundle{
				TopicId:                       1,
				ReputerRequestNonce:           &ReputerRequestNonce{ReputerNonce: &Nonce{BlockHeight: 100}},
				Reputer:                       "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
				CombinedValue:                 alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("")),
				NaiveValue:                    alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
				InfererValues:                 []*InputWorkerAttributedValue{validInfererValue},
				ForecasterValues:              nil,
				OneOutInfererValues:           nil,
				OneOutForecasterValues:        nil,
				OneInForecasterValues:         nil,
				OneOutInfererForecasterValues: nil,
				ExtraData:                     nil,
			},
			wantErr: false,
		},
		{
			name: "negative naive value valid",
			input: &InputValueBundle{
				TopicId:                       1,
				ReputerRequestNonce:           &ReputerRequestNonce{ReputerNonce: &Nonce{BlockHeight: 100}},
				Reputer:                       "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
				CombinedValue:                 alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
				NaiveValue:                    alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("-1")),
				InfererValues:                 []*InputWorkerAttributedValue{validInfererValue},
				ForecasterValues:              nil,
				OneOutInfererValues:           nil,
				OneOutForecasterValues:        nil,
				OneInForecasterValues:         nil,
				OneOutInfererForecasterValues: nil,
				ExtraData:                     nil,
			},
			wantErr: false,
		},
		{
			name: "empty naive value means 0 thus valid",
			input: &InputValueBundle{
				TopicId:                       1,
				ReputerRequestNonce:           &ReputerRequestNonce{ReputerNonce: &Nonce{BlockHeight: 100}},
				Reputer:                       "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
				CombinedValue:                 alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
				NaiveValue:                    alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("")),
				InfererValues:                 []*InputWorkerAttributedValue{validInfererValue},
				ForecasterValues:              nil,
				OneOutInfererValues:           nil,
				OneOutForecasterValues:        nil,
				OneInForecasterValues:         nil,
				OneOutInfererForecasterValues: nil,
				ExtraData:                     nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestInputWorkerAttributedValue_Validate(t *testing.T) {
	tests := []struct {
		name    string
		input   *InputWorkerAttributedValue
		wantErr bool
	}{
		{
			name: "valid worker attributed value",
			input: &InputWorkerAttributedValue{
				Worker: "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
				Value:  alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
			},
			wantErr: false,
		},
		{
			name:    "nil worker attributed value",
			input:   nil,
			wantErr: true,
		},
		{
			name: "invalid worker address",
			input: &InputWorkerAttributedValue{
				Worker: "invalid",
				Value:  alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestInputForecastElement_Validate(t *testing.T) {
	tests := []struct {
		name    string
		input   *InputForecastElement
		wantErr bool
	}{
		{
			name: "valid forecast element",
			input: &InputForecastElement{
				Inferer: "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
				Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
			},
			wantErr: false,
		},
		{
			name:    "nil forecast element",
			input:   nil,
			wantErr: true,
		},
		{
			name: "invalid inferer address",
			input: &InputForecastElement{
				Inferer: "invalid",
				Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestInputInferenceForecastBundle_Validate(t *testing.T) {
	validInference := &InputInference{
		TopicId:     1,
		Inferer:     "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
		BlockHeight: 100,
		Value:       alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
		Values:      []*InputLabeledValue{{Label: "", Value: mustNewBoundedExp40Dec(t, "1")}},
		ExtraData:   nil,
		Proof:       "",
	}

	validForecast := &InputForecast{
		TopicId:     1,
		Forecaster:  "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
		BlockHeight: 100,
		ForecastElements: []*InputForecastElement{{
			Inferer: "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
			Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
		}},
		ExtraData: nil,
	}

	tests := []struct {
		name    string
		input   *InputInferenceForecastBundle
		wantErr bool
	}{
		{
			name: "valid bundle with both inference and forecast",
			input: &InputInferenceForecastBundle{
				Inference: validInference,
				Forecast:  validForecast,
			},
			wantErr: false,
		},
		{
			name: "valid bundle with only inference",
			input: &InputInferenceForecastBundle{
				Inference: validInference,
				Forecast:  nil,
			},
			wantErr: false,
		},
		{
			name: "valid bundle with only forecast",
			input: &InputInferenceForecastBundle{
				Inference: nil,
				Forecast:  validForecast,
			},
			wantErr: false,
		},
		{
			name: "invalid - both inference and forecast are nil",
			input: &InputInferenceForecastBundle{
				Inference: nil,
				Forecast:  nil,
			},
			wantErr: true,
		},
		{
			name:    "invalid - nil bundle",
			input:   nil,
			wantErr: true,
		},
		{
			name: "invalid - inference with validation error",
			input: &InputInferenceForecastBundle{
				Inference: &InputInference{
					TopicId:     0, // invalid topic id
					Inferer:     "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
					BlockHeight: 100,
					Value:       alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
					Values:      []*InputLabeledValue{{Label: "", Value: mustNewBoundedExp40Dec(t, "1")}},
					ExtraData:   nil,
					Proof:       "",
				},
				Forecast: nil,
			},
			wantErr: true,
		},
		{
			name: "invalid - forecast with validation error",
			input: &InputInferenceForecastBundle{
				Inference: nil,
				Forecast: &InputForecast{
					TopicId:          0, // invalid topic id
					Forecaster:       "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
					BlockHeight:      100,
					ForecastElements: []*InputForecastElement{},
					ExtraData:        nil,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.wantErr {
				require.Error(t, err)
				if tt.input != nil && tt.input.Inference == nil && tt.input.Forecast == nil {
					require.ErrorContains(t, err, "inference and forecast cannot both be nil")
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTopicValidate(t *testing.T) {
	p := validParams()

	longStr := func(n int) string {
		b := make([]byte, n)
		for i := range b {
			b[i] = 'a'
		}
		return string(b)
	}

	maxTol := alloraMath.MustNewDecFromString(maxTopicUnityTolerance)

	tests := []struct {
		name        string
		mutate      func(*Topic, *Params)
		wantErr     bool
		errContains string
	}{
		{
			name:    "ok",
			mutate:  func(_ *Topic, _ *Params) {},
			wantErr: false, errContains: "",
		},
		{
			name:    "id zero reserved",
			mutate:  func(tp *Topic, _ *Params) { tp.Id = 0 },
			wantErr: true, errContains: "topic id zero is reserved",
		},
		{
			name:    "creator invalid",
			mutate:  func(tp *Topic, _ *Params) { tp.Creator = "not-bech32" },
			wantErr: true, errContains: "topic creator address invalid",
		},
		{
			name: "metadata too long",
			//nolint:gosec
			mutate:  func(tp *Topic, p *Params) { tp.Metadata = longStr(int(p.MaxStringLength) + 1) },
			wantErr: true, errContains: "topic metadata invalid",
		},
		{
			name:    "loss method empty",
			mutate:  func(tp *Topic, _ *Params) { tp.LossMethod = "" },
			wantErr: true, errContains: "topic loss method invalid",
		},
		{
			name:    "epoch last ended negative",
			mutate:  func(tp *Topic, _ *Params) { tp.EpochLastEnded = -1 },
			wantErr: true, errContains: "topic epoch last ended cannot be negative",
		},
		{
			name:    "epoch length <= 0",
			mutate:  func(tp *Topic, _ *Params) { tp.EpochLength = 0 },
			wantErr: true, errContains: "topic epoch length must be greater than zero",
		},
		{
			name:    "epoch length < min",
			mutate:  func(tp *Topic, p *Params) { tp.EpochLength = p.MinEpochLength - 1 },
			wantErr: true, errContains: "topic epoch length must be greater than minimum epoch length",
		},
		{
			name:    "worker submission window zero",
			mutate:  func(tp *Topic, _ *Params) { tp.WorkerSubmissionWindow = 0 },
			wantErr: true, errContains: "topic worker submission window must be greater than zero",
		},
		{
			name:    "worker submission window > epoch length",
			mutate:  func(tp *Topic, _ *Params) { tp.WorkerSubmissionWindow = tp.EpochLength + 1 },
			wantErr: true, errContains: "topic worker submission window cannot be higher than epoch length",
		},
		{
			name:    "ground truth lag <= 0",
			mutate:  func(tp *Topic, _ *Params) { tp.GroundTruthLag = 0 },
			wantErr: true, errContains: "topic ground truth lag must be greater than zero",
		},
		{
			name:    "ground truth lag < epoch length",
			mutate:  func(tp *Topic, _ *Params) { tp.GroundTruthLag = tp.EpochLength - 1 },
			wantErr: true, errContains: "topic ground truth lag cannot be lower than epoch length",
		},
		{
			name: "ground truth lag too high",
			mutate: func(tp *Topic, p *Params) {
				//nolint:gosec
				tp.GroundTruthLag = int64(p.MaxUnfulfilledReputerRequests)*tp.EpochLength + 1
			},
			wantErr: true, errContains: "topic ground truth lag cannot be higher than max unfulfilled reputer requests",
		},
		{
			name:    "alpha regret NaN invalid",
			mutate:  func(tp *Topic, _ *Params) { tp.AlphaRegret = alloraMath.NewNaN() },
			wantErr: true, errContains: "topic alpha regret is invalid",
		},
		{
			name:    "alpha regret out of range (<=0)",
			mutate:  func(tp *Topic, _ *Params) { tp.AlphaRegret = alloraMath.MustNewDecFromString("0") },
			wantErr: true, errContains: "topic alpha regret must be greater than 0 and less than or equal to 1",
		},
		{
			name:    "alpha regret out of range (>1)",
			mutate:  func(tp *Topic, _ *Params) { tp.AlphaRegret = alloraMath.MustNewDecFromString("1.0001") },
			wantErr: true, errContains: "topic alpha regret must be greater than 0 and less than or equal to 1",
		},
		{
			name:    "p-norm out of range low",
			mutate:  func(tp *Topic, _ *Params) { tp.PNorm = alloraMath.MustNewDecFromString("0.9") },
			wantErr: true, errContains: "topic p-norm must be between 1 and 10",
		},
		{
			name:    "p-norm out of range high",
			mutate:  func(tp *Topic, _ *Params) { tp.PNorm = alloraMath.MustNewDecFromString("11") },
			wantErr: true, errContains: "topic p-norm must be between 1 and 10",
		},
		{
			name:    "epsilon <= 0",
			mutate:  func(tp *Topic, _ *Params) { tp.Epsilon = alloraMath.MustNewDecFromString("0") },
			wantErr: true, errContains: "topic epsilon must be greater than 0",
		},
		{
			name:    "merit sortition alpha > 1",
			mutate:  func(tp *Topic, _ *Params) { tp.MeritSortitionAlpha = alloraMath.MustNewDecFromString("1.1") },
			wantErr: true, errContains: "topic merit sortition alpha must be greater than or equal to 0 and less than 1",
		},
		{
			name:    "active inferer quantile < 0",
			mutate:  func(tp *Topic, _ *Params) { tp.ActiveInfererQuantile = alloraMath.MustNewDecFromString("-0.1") },
			wantErr: true, errContains: "topic active inferer quantile must be between 0 and 1 inclusive",
		},
		{
			name:    "active forecaster quantile > 1",
			mutate:  func(tp *Topic, _ *Params) { tp.ActiveForecasterQuantile = alloraMath.MustNewDecFromString("1.1") },
			wantErr: true, errContains: "topic active forecaster quantile must be between 0 and 1 inclusive",
		},
		{
			name:    "active reputer quantile > 1",
			mutate:  func(tp *Topic, _ *Params) { tp.ActiveReputerQuantile = alloraMath.MustNewDecFromString("1.1") },
			wantErr: true, errContains: "topic active reputer quantile must be between 0 and 1 inclusive",
		},
		{
			name:    "c_norm out of range",
			mutate:  func(tp *Topic, _ *Params) { tp.CNorm = alloraMath.MustNewDecFromString("100.1") },
			wantErr: true, errContains: "topic c_norm must be between -100 and 100",
		},
		{
			name: "require_unity not allowed for SINGLE arity",
			mutate: func(tp *Topic, _ *Params) {
				tp.OutputArity = TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE
				tp.RequireUnity = true
			},
			wantErr: true, errContains: "topic require_unity MUST be false when output_arity is SINGLE",
		},
		{
			name: "require_unity with unity tolerance NaN",
			mutate: func(tp *Topic, _ *Params) {
				tp.RequireUnity = true
				tp.UnityTolerance = alloraMath.NewNaN()
			},
			wantErr: true, errContains: "unity_tolerance must be in",
		},
		{
			name: "require_unity with unity tolerance < 0",
			mutate: func(tp *Topic, _ *Params) {
				tp.RequireUnity = true
				tp.UnityTolerance = alloraMath.MustNewDecFromString("-0.1")
			},
			wantErr: true, errContains: "unity_tolerance must be in",
		},
		{
			name: "require_unity with unity tolerance > max",
			mutate: func(tp *Topic, _ *Params) {
				tp.RequireUnity = true
				tp.UnityTolerance, _ = maxTol.Add(alloraMath.MustNewDecFromString("0.0000000001"))
			},
			wantErr: true, errContains: "unity_tolerance must be in",
		},
		{
			name: "require_unity false ignores unity tolerance (even NaN)",
			mutate: func(tp *Topic, _ *Params) {
				tp.RequireUnity = false
				tp.UnityTolerance = alloraMath.NewNaN()
			},
			wantErr: false, errContains: "",
		},
		{
			name: "label whitelist above max",
			mutate: func(tp *Topic, p *Params) {
				p.MaxTopicLabelWhitelistSize = 1
				tp.LabelWhitelist = []string{"bear", "bull"}
			},
			wantErr: true, errContains: "topic label_whitelist is invalid",
		},
		{
			name:    "max top inferers one ok",
			mutate:  func(tp *Topic, _ *Params) { tp.MaxTopInferersToReward = 1 },
			wantErr: false, errContains: "",
		},
		{
			name:    "max top inferers equals global ceiling ok",
			mutate:  func(tp *Topic, p *Params) { tp.MaxTopInferersToReward = p.MaxTopInferersToReward },
			wantErr: false, errContains: "",
		},
		{
			// Zero is accepted for migration safety: pre-field topics decode as
			// zero and the shipped v15 migration validates them before v16
			// backfills the cap. The >=1 invariant is enforced at write time.
			name:    "max top inferers zero accepted (migration safety)",
			mutate:  func(tp *Topic, _ *Params) { tp.MaxTopInferersToReward = 0 },
			wantErr: false, errContains: "",
		},
		{
			name:    "max top inferers above global accepted at rest",
			mutate:  func(tp *Topic, p *Params) { tp.MaxTopInferersToReward = p.MaxTopInferersToReward + 1 },
			wantErr: false, errContains: "",
		},
		{
			name: "max top inferers above shrunk global accepted at rest",
			mutate: func(tp *Topic, p *Params) {
				p.MaxTopInferersToReward = 5
				tp.MaxTopInferersToReward = 6
			},
			wantErr: false, errContains: "",
		},
		{
			name: "max top inferers ceiling tracks params (at shrunk global ok)",
			mutate: func(tp *Topic, p *Params) {
				p.MaxTopInferersToReward = 5
				tp.MaxTopInferersToReward = 5
			},
			wantErr: false, errContains: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pp := p
			topic := validTopic(pp)

			tc.mutate(&topic, &pp)

			err := topic.Validate(pp)
			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					require.ErrorContains(t, err, tc.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func validTopic(p Params) Topic {
	creator := "allo10ljt8c3nuky75nnca4gnt9sqaxrmmyaz4d520a"

	return Topic{
		Id:                       1,
		Creator:                  creator,
		Metadata:                 "meta",
		LossMethod:               "mse",
		EpochLastEnded:           0,
		EpochLength:              p.MinEpochLength,
		GroundTruthLag:           p.MinEpochLength * 2,
		PNorm:                    alloraMath.MustNewDecFromString("3.0"),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.5"),
		AllowNegative:            false,
		Epsilon:                  alloraMath.MustNewDecFromString("0.0001"),
		InitialRegret:            alloraMath.MustNewDecFromString("0.0"),
		WorkerSubmissionWindow:   p.MinEpochLength / 2,
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.5"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.5"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.5"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.5"),
		CNorm:                    alloraMath.MustNewDecFromString("0"),
		TopicType:                TopicType_TOPIC_TYPE_REGRESSION,
		OutputArity:              TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
		RequireUnity:             false,
		UnityTolerance:           alloraMath.MustNewDecFromString("0.1"),
		MaxLabelsPerSubmission:   8,
		LabelWhitelist:           nil,
		LabelDefaultValue:        alloraMath.ZeroDec(),
		LabelCaseSensitive:       false,
		MaxTopInferersToReward:   10,
	}
}

//nolint:exhaustruct
func validParams() Params {
	return Params{
		MaxStringLength:               256,
		MinEpochLength:                10,
		MaxUnfulfilledReputerRequests: 5,
		MaxTopicLabelWhitelistSize:    DefaultMaxTopicLabelWhitelistSize,
		MaxCanonicalLabelByteLength:   64,
		MaxEpochLabelRegistrySize:     DefaultMaxEpochLabelRegistrySize,
		MaxWhitelistInputArrayLength:  2000,
		MaxTopInferersToReward:        32,
	}
}

// baseValidInput returns a minimal InputInference whose basic fields pass
// InputInference.Validate(); each sub-test overrides Values as needed.
//
//nolint:exhaustruct
func baseValidInput(t *testing.T) *InputInference {
	t.Helper()
	return &InputInference{
		TopicId:     1,
		BlockHeight: 100,
		Inferer:     "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
		Value:       alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
	}
}

func TestEffectiveMaxTopInferersToReward(t *testing.T) {
	cases := []struct {
		name                                 string
		topicCap, globalMin, globalMax, want uint64
	}{
		{"zero uses global max", 0, 1, 32, 32},
		{"within range preserved", 10, 1, 32, 10},
		{"equal to global max", 32, 1, 32, 32},
		{"above global max clamped", 40, 1, 32, 32},
		{"below global min raised", 3, 10, 32, 10},
		{"equal to global min", 10, 10, 32, 10},
		{"zero with a floor still uses global max", 0, 10, 32, 32},
		{"no floor", 10, 0, 32, 10},
		// With the default floor: below it is raised, within the range is kept,
		// above the ceiling is clamped.
		{"default floor raises a low cap", 3, 5, 32, 5},
		{"default floor keeps a cap above it", 6, 5, 32, 6},
		{"default floor with cap above the ceiling", 33, 5, 32, 32},
		// The ceiling is applied last, so an inverted global range can never
		// yield a cap above the global max.
		{"inverted global range yields global max", 3, 40, 32, 32},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, EffectiveMaxTopInferersToReward(c.topicCap, c.globalMin, c.globalMax))
		})
	}
}

// TestInputInference_ValidateWithLimits_EffectiveCap covers the cap
// enforcement: exactly cap labels ok, one over cap rejected, zero cap
// rejected defensively.

func TestInputInference_ValidateWithLimits_EffectiveCap(t *testing.T) {
	in := baseValidInput(t)
	in.Values = []*InputLabeledValue{
		{Label: "a", Value: mustNewBoundedExp40Dec(t, "1")},
		{Label: "b", Value: mustNewBoundedExp40Dec(t, "2")},
	}
	require.NoError(t, in.ValidateWithLimits(2, nil, 64, false))
	require.Error(t, in.ValidateWithLimits(1, nil, 64, false))
	require.Error(t, in.ValidateWithLimits(0, nil, 64, false))
}

// TestInputInference_ValidateWithLimits_CanonicalizesAndDedupes covers the
// canonicalization of label names and the post-canon dedupe check, plus
// ensures the canonical form is written back to the input slice in place.
func TestInputInference_ValidateWithLimits_CanonicalizesAndDedupes(t *testing.T) {
	in := baseValidInput(t)
	in.Values = []*InputLabeledValue{
		{Label: "  y  ", Value: mustNewBoundedExp40Dec(t, "1")},
	}
	require.NoError(t, in.ValidateWithLimits(4, nil, 64, false))
	require.Equal(t, "y", in.Values[0].Label, "label should be canonicalized in place")

	// Upper vs lower case collapse under the default caseSensitive=false.
	dup := baseValidInput(t)
	dup.Values = []*InputLabeledValue{
		{Label: "Cat", Value: mustNewBoundedExp40Dec(t, "1")},
		{Label: "cat", Value: mustNewBoundedExp40Dec(t, "2")},
	}
	require.Error(t, dup.ValidateWithLimits(4, nil, 64, false))

	// Under caseSensitive=true, the same two inputs are distinct.
	distinct := baseValidInput(t)
	distinct.Values = []*InputLabeledValue{
		{Label: "Cat", Value: mustNewBoundedExp40Dec(t, "1")},
		{Label: "cat", Value: mustNewBoundedExp40Dec(t, "2")},
	}
	require.NoError(t, distinct.ValidateWithLimits(4, nil, 64, true))
	require.Equal(t, "Cat", distinct.Values[0].Label, "case preserved under caseSensitive")
	require.Equal(t, "cat", distinct.Values[1].Label, "case preserved under caseSensitive")
}

// TestInputInference_ValidateWithLimits_Whitelist covers the whitelist
// check: nil whitelist means unrestricted; a non-nil whitelist requires
// membership post-canonicalization; and a canonical whitelist entry matches
// a non-canonical input.
func TestInputInference_ValidateWithLimits_Whitelist(t *testing.T) {
	okWhitelist := map[string]struct{}{"y": {}, "n": {}}

	in := baseValidInput(t)
	in.Values = []*InputLabeledValue{
		{Label: "y", Value: mustNewBoundedExp40Dec(t, "1")},
	}
	require.NoError(t, in.ValidateWithLimits(4, nil, 64, false), "nil whitelist is unrestricted")
	require.NoError(t, in.ValidateWithLimits(4, okWhitelist, 64, false))

	bad := baseValidInput(t)
	bad.Values = []*InputLabeledValue{
		{Label: "other", Value: mustNewBoundedExp40Dec(t, "1")},
	}
	require.Error(t, bad.ValidateWithLimits(4, okWhitelist, 64, false))

	// Whitelist canonical form matches a whitespace-wrapped submission.
	wrap := baseValidInput(t)
	wrap.Values = []*InputLabeledValue{
		{Label: "  y  ", Value: mustNewBoundedExp40Dec(t, "1")},
	}
	require.NoError(t, wrap.ValidateWithLimits(4, okWhitelist, 64, false))

	// Case-sensitive: "Y" must match "Y" and must NOT match a lowercase
	// whitelist entry.
	csWhitelist := map[string]struct{}{"Y": {}}
	cs := baseValidInput(t)
	cs.Values = []*InputLabeledValue{
		{Label: "Y", Value: mustNewBoundedExp40Dec(t, "1")},
	}
	require.NoError(t, cs.ValidateWithLimits(4, csWhitelist, 64, true))

	csMismatch := baseValidInput(t)
	csMismatch.Values = []*InputLabeledValue{
		{Label: "y", Value: mustNewBoundedExp40Dec(t, "1")},
	}
	require.Error(t, csMismatch.ValidateWithLimits(4, csWhitelist, 64, true))
}

// TestInputInference_ValidateWithLimits_RejectsInvalidLabel covers each of
// the canonicalizer's rejection paths when called through
// ValidateWithLimits, including nil LabeledValue entries.
func TestInputInference_ValidateWithLimits_RejectsInvalidLabel(t *testing.T) {
	in := baseValidInput(t)
	in.Values = []*InputLabeledValue{
		{Label: "   ", Value: mustNewBoundedExp40Dec(t, "1")},
	}
	require.Error(t, in.ValidateWithLimits(4, nil, 64, false))

	// Disallowed charset: zero-width space is rejected as an illegal rune.
	in2 := baseValidInput(t)
	in2.Values = []*InputLabeledValue{
		{Label: "fo\u200bo", Value: mustNewBoundedExp40Dec(t, "1")},
	}
	require.Error(t, in2.ValidateWithLimits(4, nil, 64, false))

	in3 := baseValidInput(t)
	in3.Values = []*InputLabeledValue{nil}
	require.Error(t, in3.ValidateWithLimits(4, nil, 64, false))

	// Punctuation outside the allowed separator set is rejected.
	in4 := baseValidInput(t)
	in4.Values = []*InputLabeledValue{
		{Label: "bad!", Value: mustNewBoundedExp40Dec(t, "1")},
	}
	require.Error(t, in4.ValidateWithLimits(4, nil, 64, false))
}
