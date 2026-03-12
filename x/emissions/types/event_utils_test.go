package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	alloraMath "github.com/allora-network/allora-chain/math"
)

func Test_convertNetworkInferenceBundleToEvent(t *testing.T) {
	dec := func(s string) alloraMath.Dec {
		return alloraMath.MustNewDecFromString(s)
	}

	tests := []struct {
		name          string
		in            NetworkInferenceBundle
		outlierResist bool
		want          *EventNetworkInferenceBundle
	}{
		{
			name:          "basic conversion",
			outlierResist: false,
			in: NetworkInferenceBundle{
				TopicId: 1,
				Nonce:   42,
				CombinedValue: []*LabeledValue{
					{LabelName: "a", Value: dec("1")},
					{LabelName: "b", Value: dec("2")},
				},
				NaiveValue: []*LabeledValue{
					{LabelName: "a", Value: dec("9")},
					{LabelName: "b", Value: dec("8")},
				},
				InfererValues: []*WorkerInference{
					{
						Worker: "inferer1",
						Values: []*LabeledValue{
							{LabelName: "a", Value: dec("10")},
							{LabelName: "b", Value: dec("20")},
						},
					},
				},
				ForecasterValues: []*WorkerInference{
					{
						Worker: "forecaster1",
						Values: []*LabeledValue{
							{LabelName: "a", Value: dec("3")},
							{LabelName: "b", Value: dec("4")},
						},
					},
				},
				OneOutInfererValues: []*OneOutInfererValue{
					{
						WithheldInferer: "inferer1",
						CombinedInference: []*LabeledValue{
							{LabelName: "a", Value: dec("100")},
							{LabelName: "b", Value: dec("200")},
						},
					},
				},
				OneOutForecasterValues: []*OneOutForecasterValue{
					{
						WithheldForecaster: "forecaster1",
						CombinedInference: []*LabeledValue{
							{LabelName: "a", Value: dec("300")},
							{LabelName: "b", Value: dec("400")},
						},
					},
				},
				OneInForecasterValues: []*OneInForecasterValue{
					{
						Forecaster: "forecaster1",
						CombinedInference: []*LabeledValue{
							{LabelName: "a", Value: dec("500")},
							{LabelName: "b", Value: dec("600")},
						},
					},
				},
				OneOutInfererForecasterValues: []*OneOutInfererForecasterValue{
					{
						Forecaster:      "forecaster1",
						WithheldInferer: "inferer1",
						CombinedInference: []*LabeledValue{
							{LabelName: "a", Value: dec("7")},
							{LabelName: "b", Value: dec("8")},
						},
					},
				},
			},
			want: &EventNetworkInferenceBundle{
				TopicId:             1,
				Nonce:               42,
				LabelNames:          []string{"a", "b"},
				InfererAddresses:    []string{"inferer1"},
				ForecasterAddresses: []string{"forecaster1"},
				CombinedValue:       alloraMath.DecArray{dec("1"), dec("2")},
				NaiveValue:          alloraMath.DecArray{dec("9"), dec("8")},
				InfererValues: alloraMath.DecMatrix{
					{dec("10"), dec("20")},
				},
				ForecasterValues: alloraMath.DecMatrix{
					{dec("3"), dec("4")},
				},
				OneOutInfererValues: alloraMath.DecMatrix{
					{dec("100"), dec("200")},
				},
				OneOutForecasterValues: alloraMath.DecMatrix{
					{dec("300"), dec("400")},
				},
				OneInForecasterValues: alloraMath.DecMatrix{
					{dec("500"), dec("600")},
				},
				OneOutInfererForecasterValues: []alloraMath.DecMatrix{
					{
						{dec("7"), dec("8")},
					},
				},
				OutlierResistant: false,
			},
		},

		{
			name:          "empty bundle",
			outlierResist: false,
			in: NetworkInferenceBundle{
				TopicId:                       0,
				Nonce:                         0,
				CombinedValue:                 nil,
				InfererValues:                 nil,
				ForecasterValues:              nil,
				NaiveValue:                    nil,
				OneOutInfererValues:           nil,
				OneOutForecasterValues:        nil,
				OneInForecasterValues:         nil,
				OneOutInfererForecasterValues: nil,
			},
			want: &EventNetworkInferenceBundle{
				TopicId:                       0,
				Nonce:                         0,
				LabelNames:                    nil,
				InfererAddresses:              nil,
				ForecasterAddresses:           nil,
				CombinedValue:                 nil,
				NaiveValue:                    nil,
				InfererValues:                 nil,
				ForecasterValues:              nil,
				OneOutInfererValues:           nil,
				OneOutForecasterValues:        nil,
				OneInForecasterValues:         nil,
				OneOutInfererForecasterValues: nil,
				OutlierResistant:              false,
			},
		},

		{
			name:          "multiple inferers and forecasters preserves order",
			outlierResist: true,
			in: NetworkInferenceBundle{
				TopicId: 9,
				Nonce:   7,
				CombinedValue: []*LabeledValue{
					{LabelName: "x", Value: dec("1")},
					{LabelName: "y", Value: dec("2")},
					{LabelName: "z", Value: dec("3")},
				},
				InfererValues: []*WorkerInference{
					{
						Worker: "i2",
						Values: []*LabeledValue{
							{LabelName: "x", Value: dec("20")},
							{LabelName: "y", Value: dec("21")},
							{LabelName: "z", Value: dec("22")},
						},
					},
					{
						Worker: "i1",
						Values: []*LabeledValue{
							{LabelName: "x", Value: dec("10")},
							{LabelName: "y", Value: dec("11")},
							{LabelName: "z", Value: dec("12")},
						},
					},
				},
				ForecasterValues: []*WorkerInference{
					{
						Worker: "fB",
						Values: []*LabeledValue{
							{LabelName: "x", Value: dec("5")},
							{LabelName: "y", Value: dec("6")},
							{LabelName: "z", Value: dec("7")},
						},
					},
					{
						Worker: "fA",
						Values: []*LabeledValue{
							{LabelName: "x", Value: dec("1")},
							{LabelName: "y", Value: dec("2")},
							{LabelName: "z", Value: dec("3")},
						},
					},
				},
				NaiveValue:                    nil,
				OneOutInfererValues:           nil,
				OneOutForecasterValues:        nil,
				OneInForecasterValues:         nil,
				OneOutInfererForecasterValues: nil,
			},
			want: &EventNetworkInferenceBundle{
				TopicId:             9,
				Nonce:               7,
				LabelNames:          []string{"x", "y", "z"},
				InfererAddresses:    []string{"i2", "i1"},
				ForecasterAddresses: []string{"fB", "fA"},
				CombinedValue:       alloraMath.DecArray{dec("1"), dec("2"), dec("3")},
				NaiveValue:          nil,
				InfererValues: alloraMath.DecMatrix{
					{dec("20"), dec("21"), dec("22")},
					{dec("10"), dec("11"), dec("12")},
				},
				ForecasterValues: alloraMath.DecMatrix{
					{dec("5"), dec("6"), dec("7")},
					{dec("1"), dec("2"), dec("3")},
				},
				OneOutInfererValues:           nil,
				OneOutForecasterValues:        nil,
				OneInForecasterValues:         nil,
				OneOutInfererForecasterValues: nil,
				OutlierResistant:              true,
			},
		},

		{
			name:          "one out inferer forecaster groups by forecaster and preserves first-seen forecaster order",
			outlierResist: false,
			in: NetworkInferenceBundle{
				TopicId: 1,
				Nonce:   1,
				CombinedValue: []*LabeledValue{
					{LabelName: "a", Value: dec("0")},
					{LabelName: "b", Value: dec("0")},
				},
				InfererValues:          nil,
				ForecasterValues:       nil,
				NaiveValue:             nil,
				OneOutInfererValues:    nil,
				OneOutForecasterValues: nil,
				OneInForecasterValues:  nil,
				OneOutInfererForecasterValues: []*OneOutInfererForecasterValue{
					{
						Forecaster:      "f2",
						WithheldInferer: "i1",
						CombinedInference: []*LabeledValue{
							{LabelName: "a", Value: dec("21")},
							{LabelName: "b", Value: dec("22")},
						},
					},
					{
						Forecaster:      "f1",
						WithheldInferer: "i1",
						CombinedInference: []*LabeledValue{
							{LabelName: "a", Value: dec("11")},
							{LabelName: "b", Value: dec("12")},
						},
					},
					{
						Forecaster:      "f2",
						WithheldInferer: "i2",
						CombinedInference: []*LabeledValue{
							{LabelName: "a", Value: dec("23")},
							{LabelName: "b", Value: dec("24")},
						},
					},
				},
			},
			want: &EventNetworkInferenceBundle{
				TopicId:             1,
				Nonce:               1,
				LabelNames:          []string{"a", "b"},
				InfererAddresses:    nil,
				ForecasterAddresses: nil,
				CombinedValue: alloraMath.DecArray{
					dec("0"), dec("0"),
				},
				NaiveValue:             nil,
				InfererValues:          nil,
				ForecasterValues:       nil,
				OneOutInfererValues:    nil,
				OneOutForecasterValues: nil,
				OneInForecasterValues:  nil,
				OneOutInfererForecasterValues: []alloraMath.DecMatrix{
					{
						{dec("21"), dec("22")},
						{dec("23"), dec("24")},
					},
					{
						{dec("11"), dec("12")},
					},
				},
				OutlierResistant: false,
			},
		},

		{
			name:          "outlier resistant flag is propagated",
			outlierResist: true,
			in: NetworkInferenceBundle{
				TopicId: 2,
				Nonce:   3,
				CombinedValue: []*LabeledValue{
					{LabelName: "a", Value: dec("1")},
				},
				InfererValues:                 nil,
				ForecasterValues:              nil,
				NaiveValue:                    nil,
				OneOutInfererValues:           nil,
				OneOutForecasterValues:        nil,
				OneInForecasterValues:         nil,
				OneOutInfererForecasterValues: nil,
			},
			want: &EventNetworkInferenceBundle{
				TopicId:                       2,
				Nonce:                         3,
				LabelNames:                    []string{"a"},
				InfererAddresses:              nil,
				ForecasterAddresses:           nil,
				CombinedValue:                 alloraMath.DecArray{dec("1")},
				NaiveValue:                    nil,
				InfererValues:                 nil,
				ForecasterValues:              nil,
				OneOutInfererValues:           nil,
				OneOutForecasterValues:        nil,
				OneInForecasterValues:         nil,
				OneOutInfererForecasterValues: nil,
				OutlierResistant:              true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertNetworkInferenceBundleToEvent(tt.in, tt.outlierResist)
			require.Equal(t, tt.want, got)
		})
	}
}
