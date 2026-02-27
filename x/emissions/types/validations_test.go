package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	alloraMath "github.com/allora-network/allora-chain/math"
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
			mutate:  func(tp *Topic, _ *Params) { tp.PNorm = alloraMath.MustNewDecFromString("2.4") },
			wantErr: true, errContains: "topic p-norm must be between 2.5 and 4.5",
		},
		{
			name:    "p-norm out of range high",
			mutate:  func(tp *Topic, _ *Params) { tp.PNorm = alloraMath.MustNewDecFromString("4.6") },
			wantErr: true, errContains: "topic p-norm must be between 2.5 and 4.5",
		},
		{
			name:    "epsilon <= 0",
			mutate:  func(tp *Topic, _ *Params) { tp.Epsilon = alloraMath.MustNewDecFromString("0") },
			wantErr: true, errContains: "topic epsilon must be greater than 0",
		},
		{
			name:    "merit sortition alpha > 1",
			mutate:  func(tp *Topic, _ *Params) { tp.MeritSortitionAlpha = alloraMath.MustNewDecFromString("1.1") },
			wantErr: true, errContains: "topic merit sortition alpha must be between 0 and 1 inclusive",
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
			name: "require_unity with unity tolerance <= 0",
			mutate: func(tp *Topic, _ *Params) {
				tp.RequireUnity = true
				tp.UnityTolerance = alloraMath.MustNewDecFromString("0")
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
	}
}

//nolint:exhaustruct
func validParams() Params {
	return Params{
		MaxStringLength:               256,
		MinEpochLength:                10,
		MaxUnfulfilledReputerRequests: 5,
	}
}
