package types

import (
	"os"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/allora-network/allora-chain/math"
)

func TestMain(m *testing.M) {
	// Set custom address prefixes
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("allo", "allopub")
	config.Seal()

	os.Exit(m.Run())
}

func TestInputInferenceConvert(t *testing.T) {
	tests := []struct {
		name    string
		input   *InputInference
		wantErr bool
	}{
		{
			name:    "nil input",
			input:   nil,
			wantErr: true,
		},
		{
			name: "valid input",
			input: &InputInference{
				TopicId:     1,
				BlockHeight: 100,
				Inferer:     "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
				Value:       mustNewBoundedExp40Dec(t, "1.23"),
				Values:      []math.BoundedExp40Dec{mustNewBoundedExp40Dec(t, "1.23")},
				ExtraData:   []byte("extra"),
				Proof:       "proof",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewInferenceFromInput(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.input == nil {
				require.Nil(t, got)
				return
			}
			require.Equal(t, tt.input.TopicId, got.TopicId)
			require.Equal(t, tt.input.BlockHeight, got.BlockHeight)
			require.Equal(t, tt.input.Inferer, got.Inferer)
			require.Equal(t, tt.input.ExtraData, got.ExtraData)
			require.Equal(t, tt.input.Proof, got.Proof)
			// Check value conversion
			dec, _ := tt.input.Value.ToDec()
			require.True(t, dec.Equal(got.Value))
			for i := range got.Values {
				decv, _ := tt.input.Values[i].ToDec()
				require.True(t, decv.Equal(got.Values[i]))
			}
		})
	}
}

func TestInputForecastElementConvert(t *testing.T) {
	tests := []struct {
		name    string
		input   *InputForecastElement
		wantErr bool
	}{
		{
			name:    "nil input",
			input:   nil,
			wantErr: true,
		},
		{
			name: "valid input",
			input: &InputForecastElement{
				Inferer: "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
				Value:   mustNewBoundedExp40Dec(t, "1.23"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewForecastElementFromInput(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.input == nil {
				require.Nil(t, got)
				return
			}
			require.Equal(t, tt.input.Inferer, got.Inferer)
			boundedDec, err := tt.input.Value.ToDec()
			require.NoError(t, err)
			require.True(t, boundedDec.Equal(got.Value))
		})
	}
}

func TestInputForecastConvert(t *testing.T) {
	tests := []struct {
		name    string
		input   *InputForecast
		wantErr bool
	}{
		{
			name:    "nil input",
			input:   nil,
			wantErr: true,
		},
		{
			name: "valid input",
			input: &InputForecast{
				TopicId:     1,
				BlockHeight: 100,
				Forecaster:  "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5",
				ForecastElements: []*InputForecastElement{
					{
						Inferer: "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
						Value:   mustNewBoundedExp40Dec(t, "1.23"),
					},
					{
						Inferer: "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
						Value:   mustNewBoundedExp40Dec(t, "4.56"),
					},
				},
				ExtraData: []byte("extra"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewForecastFromInput(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.input == nil {
				require.Nil(t, got)
				return
			}
			require.Equal(t, tt.input.TopicId, got.TopicId)
			require.Equal(t, tt.input.BlockHeight, got.BlockHeight)
			require.Equal(t, tt.input.Forecaster, got.Forecaster)
			require.Equal(t, tt.input.ExtraData, got.ExtraData)
			require.Equal(t, len(tt.input.ForecastElements), len(got.ForecastElements))
		})
	}
}

func TestInputInferenceForecastBundleConvert(t *testing.T) {
	validInference := &InputInference{
		TopicId:     1,
		BlockHeight: 100,
		Inferer:     "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
		Value:       mustNewBoundedExp40Dec(t, "1.23"),
		ExtraData:   []byte("extra"),
		Proof:       "proof",
	}

	validForecast := &InputForecast{
		TopicId:     1,
		BlockHeight: 100,
		Forecaster:  "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5",
		ForecastElements: []*InputForecastElement{
			{
				Inferer: "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
				Value:   mustNewBoundedExp40Dec(t, "1.23"),
			},
		},
		ExtraData: []byte("extra"),
	}

	tests := []struct {
		name    string
		input   *InputInferenceForecastBundle
		wantErr bool
	}{
		{
			name:    "nil input",
			input:   nil,
			wantErr: true,
		},
		{
			name: "valid input",
			input: &InputInferenceForecastBundle{
				Inference: validInference,
				Forecast:  validForecast,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewInferenceForecastBundleFromInput(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.input == nil {
				require.Nil(t, got)
				return
			}
			require.NotNil(t, got.Inference)
			require.NotNil(t, got.Forecast)
		})
	}
}

// Helper function to create BoundedExp40Dec for testing
func mustNewBoundedExp40Dec(t *testing.T, s string) math.BoundedExp40Dec {
	t.Helper()
	dec, err := math.NewBoundedExp40DecFromString(s)
	require.NoError(t, err)
	return dec
}

// Add more test functions for other types...
// TestInputWorkerDataBundleConvert
// TestInputWorkerAttributedValueConvert
// TestInputWithheldWorkerAttributedValueConvert
// TestInputOneOutInfererForecasterValuesConvert
// TestInputValueBundleConvert
// TestInputReputerValueBundleConvert

// Example of a complex test:
func TestInputValueBundleConvert(t *testing.T) {
	tests := []struct {
		name    string
		input   *InputValueBundle
		wantErr bool
	}{
		{
			name:    "nil input",
			input:   nil,
			wantErr: true,
		},
		{
			name: "valid input with all fields",
			input: &InputValueBundle{
				TopicId: 1,
				ReputerRequestNonce: &ReputerRequestNonce{
					ReputerNonce: &Nonce{
						BlockHeight: 1,
					},
				},
				Reputer:       "allo1xy0pf5hq85j873glav6aajkvtennmg3fpu3cec",
				ExtraData:     []byte("extra"),
				CombinedValue: mustNewBoundedExp40Dec(t, "1.23"),
				NaiveValue:    mustNewBoundedExp40Dec(t, "4.56"),
				InfererValues: []*InputWorkerAttributedValue{
					{
						Worker: "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
						Value:  mustNewBoundedExp40Dec(t, "1.23"),
					},
				},
				ForecasterValues: []*InputWorkerAttributedValue{
					{
						Worker: "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
						Value:  mustNewBoundedExp40Dec(t, "4.56"),
					},
				},
				OneOutInfererValues: []*InputWithheldWorkerAttributedValue{
					{
						Worker: "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
						Value:  mustNewBoundedExp40Dec(t, "7.89"),
					},
				},
				OneOutForecasterValues: []*InputWithheldWorkerAttributedValue{
					{
						Worker: "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
						Value:  mustNewBoundedExp40Dec(t, "7.89"),
					},
				},
				OneInForecasterValues: []*InputWorkerAttributedValue{
					{
						Worker: "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
						Value:  mustNewBoundedExp40Dec(t, "7.89"),
					},
				},
				OneOutInfererForecasterValues: []*InputOneOutInfererForecasterValues{
					{
						Forecaster: "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
						OneOutInfererValues: []*InputWithheldWorkerAttributedValue{
							{
								Worker: "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
								Value:  mustNewBoundedExp40Dec(t, "7.89"),
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewValueBundleFromInput(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.input == nil {
				require.Nil(t, got)
				return
			}

			// Verify all fields were converted correctly
			require.Equal(t, tt.input.TopicId, got.TopicId)
			require.Equal(t, tt.input.Reputer, got.Reputer)
			require.Equal(t, tt.input.ExtraData, got.ExtraData)

			// Check decimal conversions
			combinedValue, err := tt.input.CombinedValue.ToDec()
			require.NoError(t, err)
			require.True(t, combinedValue.Equal(got.CombinedValue))

			naiveValue, err := tt.input.NaiveValue.ToDec()
			require.NoError(t, err)
			require.True(t, naiveValue.Equal(got.NaiveValue))

			// Check slices
			require.Equal(t, len(tt.input.InfererValues), len(got.InfererValues))
			require.Equal(t, len(tt.input.ForecasterValues), len(got.ForecasterValues))
			require.Equal(t, len(tt.input.OneOutInfererValues), len(got.OneOutInfererValues))
			// Add more slice checks as needed
		})
	}
}
