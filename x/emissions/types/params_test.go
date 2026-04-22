package types

import (
	"testing"

	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/stretchr/testify/require"
)

func TestDefaultParams(t *testing.T) {
	expectedParams := Params{
		Version:                             "v10",
		MinTopicWeight:                      alloraMath.MustNewDecFromString("100"),
		RequiredMinimumStake:                cosmosMath.NewInt(10000),
		RemoveStakeDelayWindow:              int64((60 * 60 * 24 * 7 * 3) / 3),
		MinEpochLength:                      12,
		BetaEntropy:                         alloraMath.MustNewDecFromString("0.25"),
		LearningRate:                        alloraMath.MustNewDecFromString("0.05"),
		GradientDescentMaxIters:             uint64(10),
		MaxGradientThreshold:                alloraMath.MustNewDecFromString("0.001"),
		MinStakeFraction:                    alloraMath.MustNewDecFromString("0.5"),
		EpsilonReputer:                      alloraMath.MustNewDecFromString("0.01"),
		EpsilonSafeDiv:                      alloraMath.MustNewDecFromString("0.0000001"),
		MaxUnfulfilledWorkerRequests:        uint64(100),
		MaxUnfulfilledReputerRequests:       uint64(100),
		TopicRewardStakeImportance:          alloraMath.MustNewDecFromString("0.5"),
		TopicRewardFeeRevenueImportance:     alloraMath.MustNewDecFromString("0.5"),
		TopicRewardAlpha:                    alloraMath.MustNewDecFromString("0.5"),
		TaskRewardAlpha:                     alloraMath.MustNewDecFromString("0.1"),
		ValidatorsVsAlloraPercentReward:     alloraMath.MustNewDecFromString("0.25"),
		MaxSamplesToScaleScores:             uint64(10),
		MaxTopInferersToReward:              uint64(32),
		MaxTopForecastersToReward:           uint64(6),
		MaxTopReputersToReward:              uint64(6),
		CreateTopicFee:                      cosmosMath.NewInt(75000),
		RegistrationFee:                     cosmosMath.NewInt(200),
		DefaultPageLimit:                    uint64(100),
		MaxPageLimit:                        uint64(1000),
		MinEpochLengthRecordLimit:           int64(3),
		MaxSerializedMsgLength:              int64(1000 * 1000),
		BlocksPerMonth:                      uint64(864000),
		PRewardInference:                    alloraMath.NewDecFromInt64(3),
		PRewardForecast:                     alloraMath.NewDecFromInt64(3),
		PRewardReputer:                      alloraMath.NewDecFromInt64(1),
		CRewardInference:                    alloraMath.MustNewDecFromString("0.75"),
		CRewardForecast:                     alloraMath.MustNewDecFromString("0.75"),
		CNorm:                               alloraMath.MustNewDecFromString("0.75"),
		HalfMaxProcessStakeRemovalsEndBlock: uint64(40),
		DataSendingFee:                      cosmosMath.NewInt(10),
		MaxElementsPerForecast:              uint64(12),
		MaxActiveTopicsPerBlock:             uint64(1),
		MaxStringLength:                     uint64(255),
		InitialRegretQuantile:               alloraMath.MustNewDecFromString("0.25"),
		PNormSafeDiv:                        alloraMath.MustNewDecFromString("8.25"),
		GlobalWhitelistEnabled:              true,
		TopicCreatorWhitelistEnabled:        true,
		MinExperiencedWorkerRegrets:         uint64(10),
		InferenceOutlierDetectionThreshold:  alloraMath.MustNewDecFromString("11"),
		InferenceOutlierDetectionAlpha:      alloraMath.MustNewDecFromString("0.2"),
		LambdaInitialScore:                  alloraMath.MustNewDecFromString("2"),
		GlobalWorkerWhitelistEnabled:        true,
		GlobalReputerWhitelistEnabled:       true,
		GlobalAdminWhitelistAppended:        true,
		MaxWhitelistInputArrayLength:        uint64(2000),
		MinWeightThresholdForStdnorm:        alloraMath.MustNewDecFromString("0.000001"),
		MaxLabelsPerSubmission:              DefaultMaxLabelsPerSubmission,
		MaxCanonicalLabelByteLength:         64,
	}

	params := DefaultParams()

	require.Equal(t, expectedParams, params)
}

// TestValidateMaxLabelsPerSubmission exercises the explicit bound on the
// new module parameter: 0 is rejected, 1 is accepted (smallest positive
// cap), DefaultMaxLabelsPerSubmission round-trips validation, the upper
// bound is inclusive, and one-above-bound is rejected.
func TestValidateMaxLabelsPerSubmission(t *testing.T) {
	cases := []struct {
		name string
		in   uint64
		ok   bool
	}{
		{name: "zero rejected", in: 0, ok: false},
		{name: "one ok", in: 1, ok: true},
		{name: "default ok", in: DefaultMaxLabelsPerSubmission, ok: true},
		{name: "upper bound ok", in: MaxMaxLabelsPerSubmission, ok: true},
		{name: "above upper rejected", in: MaxMaxLabelsPerSubmission + 1, ok: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateMaxLabelsPerSubmission(tc.in)
			if tc.ok {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

// TestParamsValidate_RejectsZeroMaxLabelsPerSubmission ensures that
// Params.Validate does not silently accept a zero cap. The v15 migration
// backfills zero to DefaultMaxLabelsPerSubmission, so a zero at
// Params.Validate time indicates a genuinely invalid state.
func TestParamsValidate_RejectsZeroMaxLabelsPerSubmission(t *testing.T) {
	p := DefaultParams()
	p.MaxLabelsPerSubmission = 0
	require.Error(t, p.Validate())
}

// TestValidateMaxCanonicalLabelByteLength exercises the explicit bound on
// the canonical label byte cap: 0 is rejected, 1 is accepted (smallest
// positive cap), the default round-trips, the upper bound is inclusive,
// and one-above-bound is rejected. The migration backfills zero to the
// default, so Params.Validate must reject the zero value.
func TestValidateMaxCanonicalLabelByteLength(t *testing.T) {
	cases := []struct {
		name string
		in   uint64
		ok   bool
	}{
		{name: "zero rejected", in: 0, ok: false},
		{name: "one ok", in: MinMaxCanonicalLabelByteLength, ok: true},
		{name: "module-initial 64 ok", in: 64, ok: true},
		{name: "upper bound ok", in: MaxMaxCanonicalLabelByteLength, ok: true},
		{name: "above upper rejected", in: MaxMaxCanonicalLabelByteLength + 1, ok: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateMaxCanonicalLabelByteLength(tc.in)
			if tc.ok {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

// TestParamsValidate_RejectsZeroMaxCanonicalLabelByteLength documents that
// Params.Validate rejects a zero cap; the v15 migration is the only code
// path that may observe zero and it backfills to the module-initial 64
// before calling Validate.
func TestParamsValidate_RejectsZeroMaxCanonicalLabelByteLength(t *testing.T) {
	p := DefaultParams()
	p.MaxCanonicalLabelByteLength = 0
	require.Error(t, p.Validate())
}
