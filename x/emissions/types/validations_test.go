package types

import (
	"testing"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/stretchr/testify/require"
)

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
