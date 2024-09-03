package types

import (
	"testing"

	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/stretchr/testify/require"
)

func TestDefaultParams(t *testing.T) {
	expectedParams := Params{
		Version:                             "v2",
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
		PRewardInference:                    alloraMath.NewDecFromInt64(1),
		PRewardForecast:                     alloraMath.NewDecFromInt64(3),
		PRewardReputer:                      alloraMath.NewDecFromInt64(3),
		CRewardInference:                    alloraMath.MustNewDecFromString("0.75"),
		CRewardForecast:                     alloraMath.MustNewDecFromString("0.75"),
		CNorm:                               alloraMath.MustNewDecFromString("0.75"),
		HalfMaxProcessStakeRemovalsEndBlock: uint64(40),
		DataSendingFee:                      cosmosMath.NewInt(10),
		MaxElementsPerForecast:              uint64(12),
		MaxActiveTopicsPerBlock:             uint64(1),
	}

	params := DefaultParams()

	require.Equal(t, expectedParams, params)
}
