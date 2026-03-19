package types

import (
	"os"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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
			boundedDec := tt.input.Value.ToDec()
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
			combinedValue := tt.input.CombinedValue.ToDec()
			require.NoError(t, err)
			require.True(t, combinedValue.Equal(got.CombinedValue))

			naiveValue := tt.input.NaiveValue.ToDec()
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

func TestConvertInferenceValuesFromProto(t *testing.T) {
	topicId := uint64(1)
	nonce := int64(1)

	w1 := "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5"
	w2 := "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve"

	mustDec := func(x string) math.Dec { return math.MustNewDecFromString(x) }

	type tc struct {
		name      string
		arity     TopicOutputArity
		labels    []*TopicLabel
		inf       *Inference
		wantErrIs error
		wantVals  []string
	}

	cases := []tc{
		{
			name:      "nil_inference_rejected",
			arity:     TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			labels:    nil,
			inf:       nil,
			wantErrIs: sdkerrors.ErrInvalidRequest,
		},
		{
			name:  "SINGLE_scalar_only_ok",
			arity: TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			inf: &Inference{
				TopicId:     topicId,
				BlockHeight: nonce,
				Inferer:     w1,
				Values:      []math.Dec{mustDec("42")},
			},
			wantVals: []string{"42"},
		},
		{
			name:  "SINGLE_values_len1_equal_scalar_ok",
			arity: TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			inf: &Inference{
				TopicId:     topicId,
				BlockHeight: nonce,
				Inferer:     w1,
				Values:      []math.Dec{mustDec("7")},
			},
			wantVals: []string{"7"},
		},
		{
			name:  "SINGLE_values_len_gt_1_rejected",
			arity: TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			inf: &Inference{
				TopicId:     topicId,
				BlockHeight: nonce,
				Inferer:     w1,
				Values:      []math.Dec{mustDec("1"), mustDec("2")},
			},
			wantErrIs: sdkerrors.ErrInvalidRequest,
		},
		{
			name:   "MULTI_empty_registry_rejected",
			arity:  TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			labels: []*TopicLabel{},
			inf: &Inference{
				TopicId:     topicId,
				BlockHeight: nonce,
				Inferer:     w1,
				Values:      []math.Dec{mustDec("1")},
			},
			wantErrIs: sdkerrors.ErrLogic,
		},
		{
			name:  "MULTI_empty_values_rejected",
			arity: TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			labels: []*TopicLabel{{
				Id:   1,
				Name: "a",
			}},
			inf: &Inference{
				TopicId:     topicId,
				BlockHeight: nonce,
				Inferer:     w1,
				Values:      nil,
			},
			wantErrIs: sdkerrors.ErrInvalidRequest,
		},
		{
			name:  "MULTI_values_len_gt_registry_rejected",
			arity: TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			labels: []*TopicLabel{
				{
					Id:   1,
					Name: "a",
				}, {
					Id:   2,
					Name: "b",
				},
			},
			inf: &Inference{
				TopicId:     topicId,
				BlockHeight: nonce,
				Inferer:     w1,
				Values:      []math.Dec{mustDec("1"), mustDec("2"), mustDec("3")},
			},
			wantErrIs: sdkerrors.ErrLogic,
		},
		{
			name:  "MULTI_exact_len_ok_no_padding",
			arity: TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			labels: []*TopicLabel{
				{
					Id:   1,
					Name: "a",
				}, {
					Id:   2,
					Name: "b",
				}, {
					Id:   3,
					Name: "c",
				},
			},
			inf: &Inference{
				TopicId:     topicId,
				BlockHeight: nonce,
				Inferer:     w2,
				Values:      []math.Dec{mustDec("10"), mustDec("20"), mustDec("30")},
			},
			wantVals: []string{"10", "20", "30"},
		},
		{
			name:  "MULTI_shorter_len_pads_to_registry_len",
			arity: TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			labels: []*TopicLabel{
				{
					Id:   1,
					Name: "a",
				}, {
					Id:   2,
					Name: "b",
				}, {
					Id:   3,
					Name: "c",
				}, {
					Id:   4,
					Name: "d",
				}, {
					Id:   5,
					Name: "e",
				},
			},
			inf: &Inference{
				TopicId:     topicId,
				BlockHeight: nonce,
				Inferer:     w1,
				Values:      []math.Dec{mustDec("9"), mustDec("8")}, // => [9,8,0,0,0]
			},
			wantVals: []string{"9", "8", "0", "0", "0"},
		},
		{
			name:  "MULTI_rejects_invalid_value_in_values",
			arity: TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			labels: []*TopicLabel{
				{
					Id:   1,
					Name: "a",
				}, {
					Id:   2,
					Name: "b",
				}, {
					Id:   3,
					Name: "c",
				},
			},
			inf: &Inference{
				TopicId:     topicId,
				BlockHeight: nonce,
				Inferer:     w1,
				Values:      []math.Dec{mustDec("1"), math.NewNaN(), mustDec("3")},
			},
			wantErrIs: sdkerrors.ErrInvalidRequest,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			nonce = int64(1)

			inf := (*Inference)(nil)
			if c.inf != nil {
				inf = c.inf
			}

			got, err := ConvertInferenceValuesFromProto(c.arity, c.labels, inf)

			if c.wantErrIs != nil {
				require.ErrorIs(t, err, c.wantErrIs)
				return
			}
			require.NoError(t, err)

			require.Equal(t, len(c.wantVals), len(got))
			for i := range c.wantVals {
				require.Equal(t, c.wantVals[i], got[i].String())
			}
		})
	}
}
