package types

import (
	"testing"

	"github.com/allora-network/allora-chain/math"
	"github.com/stretchr/testify/require"
)

func TestBoundedInferenceConvert(t *testing.T) {
	tests := []struct {
		name    string
		input   *BoundedInference
		wantErr bool
	}{
		{
			name:    "nil input",
			input:   nil,
			wantErr: false,
		},
		{
			name: "valid input",
			input: &BoundedInference{
				TopicId:     1,
				BlockHeight: 100,
				Inferer:     "inferer1",
				Value:       mustNewBoundedExp40Dec(t, "1.23"),
				ExtraData:   []byte("extra"),
				Proof:       "proof",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.input.Convert()
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
			boundedDec, err := tt.input.Value.ToDec()
			require.NoError(t, err)
			require.True(t, boundedDec.Equal(got.Value))
		})
	}
}

func TestBoundedForecastElementConvert(t *testing.T) {
	tests := []struct {
		name    string
		input   *BoundedForecastElement
		wantErr bool
	}{
		{
			name:    "nil input",
			input:   nil,
			wantErr: false,
		},
		{
			name: "valid input",
			input: &BoundedForecastElement{
				Inferer: "inferer1",
				Value:   mustNewBoundedExp40Dec(t, "1.23"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.input.Convert()
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

func TestBoundedForecastConvert(t *testing.T) {
	tests := []struct {
		name    string
		input   *BoundedForecast
		wantErr bool
	}{
		{
			name:    "nil input",
			input:   nil,
			wantErr: false,
		},
		{
			name: "valid input",
			input: &BoundedForecast{
				TopicId:     1,
				BlockHeight: 100,
				Forecaster:  "forecaster1",
				ForecastElements: []*BoundedForecastElement{
					{
						Inferer: "inferer1",
						Value:   mustNewBoundedExp40Dec(t, "1.23"),
					},
					{
						Inferer: "inferer2",
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
			got, err := tt.input.Convert()
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

func TestBoundedInferenceForecastBundleConvert(t *testing.T) {
	validInference := &BoundedInference{
		TopicId:     1,
		BlockHeight: 100,
		Inferer:     "inferer1",
		Value:       mustNewBoundedExp40Dec(t, "1.23"),
		ExtraData:   []byte("extra"),
		Proof:       "proof",
	}

	validForecast := &BoundedForecast{
		TopicId:     1,
		BlockHeight: 100,
		Forecaster:  "forecaster1",
		ForecastElements: []*BoundedForecastElement{
			{
				Inferer: "inferer1",
				Value:   mustNewBoundedExp40Dec(t, "1.23"),
			},
		},
		ExtraData: []byte("extra"),
	}

	tests := []struct {
		name    string
		input   *BoundedInferenceForecastBundle
		wantErr bool
	}{
		{
			name:    "nil input",
			input:   nil,
			wantErr: false,
		},
		{
			name: "valid input",
			input: &BoundedInferenceForecastBundle{
				Inference: validInference,
				Forecast:  validForecast,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.input.Convert()
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
	dec, err := math.NewBoundedExp40DecFromString(s)
	require.NoError(t, err)
	return dec
}

// Add more test functions for other types...
// TestBoundedWorkerDataBundleConvert
// TestBoundedWorkerAttributedValueConvert
// TestBoundedWithheldWorkerAttributedValueConvert
// TestBoundedOneOutInfererForecasterValuesConvert
// TestBoundedValueBundleConvert
// TestBoundedReputerValueBundleConvert

// Example of a complex test:
func TestBoundedValueBundleConvert(t *testing.T) {
	tests := []struct {
		name    string
		input   *BoundedValueBundle
		wantErr bool
	}{
		{
			name:    "nil input",
			input:   nil,
			wantErr: false,
		},
		{
			name: "valid input with all fields",
			input: &BoundedValueBundle{
				TopicId:       1,
				Reputer:       "reputer1",
				ExtraData:     []byte("extra"),
				CombinedValue: mustNewBoundedExp40Dec(t, "1.23"),
				NaiveValue:    mustNewBoundedExp40Dec(t, "4.56"),
				InfererValues: []*BoundedWorkerAttributedValue{
					{
						Worker: "worker1",
						Value:  mustNewBoundedExp40Dec(t, "1.23"),
					},
				},
				ForecasterValues: []*BoundedWorkerAttributedValue{
					{
						Worker: "worker2",
						Value:  mustNewBoundedExp40Dec(t, "4.56"),
					},
				},
				OneOutInfererValues: []*BoundedWithheldWorkerAttributedValue{
					{
						Worker: "worker3",
						Value:  mustNewBoundedExp40Dec(t, "7.89"),
					},
				},
				// Add more fields as needed
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.input.Convert()
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
