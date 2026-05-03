package types

import (
	"strings"
	"testing"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/stretchr/testify/require"
)

func validCreateNewTopicRequest() *CreateNewTopicRequest {
	return &CreateNewTopicRequest{
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := validCreateNewTopicRequest()
			tt.mutate(msg)

			err := msg.Validate(maxStringLen)
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
				Inferer:     "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy", // valid bech32 address
				BlockHeight: 100,
				Value:       alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("1")),
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
