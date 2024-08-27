package msgserver_test

import (
	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *MsgServerTestSuite) TestUpdateAllParams() {
	ctx, msgServer := s.ctx, s.msgServer
	keeper := s.emissionsKeeper
	require := s.Require()

	adminPrivateKey := secp256k1.GenPrivKey()
	adminAddr := sdk.AccAddress(adminPrivateKey.PubKey().Address())

	err := keeper.AddWhitelistAdmin(ctx, adminAddr.String())
	require.NoError(err)

	newParams := &types.OptionalParams{
		Version:                         []string{"1234"},
		MinTopicWeight:                  []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
		RequiredMinimumStake:            []cosmosMath.Int{cosmosMath.NewInt(1234)},
		RemoveStakeDelayWindow:          []int64{1234},
		MinEpochLength:                  []int64{1234},
		BetaEntropy:                     []alloraMath.Dec{alloraMath.MustNewDecFromString(".1234")},
		LearningRate:                    []alloraMath.Dec{alloraMath.MustNewDecFromString(".1234")},
		GradientDescentMaxIters:         []uint64{1234},
		MaxGradientThreshold:            []alloraMath.Dec{alloraMath.MustNewDecFromString(".1234")},
		MinStakeFraction:                []alloraMath.Dec{alloraMath.MustNewDecFromString(".1234")},
		EpsilonReputer:                  []alloraMath.Dec{alloraMath.MustNewDecFromString(".1234")},
		MaxUnfulfilledWorkerRequests:    []uint64{1234},
		MaxUnfulfilledReputerRequests:   []uint64{1234},
		TopicRewardStakeImportance:      []alloraMath.Dec{alloraMath.MustNewDecFromString(".1234")},
		TopicRewardFeeRevenueImportance: []alloraMath.Dec{alloraMath.MustNewDecFromString(".1234")},
		TopicRewardAlpha:                []alloraMath.Dec{alloraMath.MustNewDecFromString(".1234")},
		TaskRewardAlpha:                 []alloraMath.Dec{alloraMath.MustNewDecFromString(".1234")},
		ValidatorsVsAlloraPercentReward: []alloraMath.Dec{alloraMath.MustNewDecFromString(".1234")},
		MaxSamplesToScaleScores:         []uint64{1234},
		MaxTopInferersToReward:          []uint64{1234},
		MaxTopForecastersToReward:       []uint64{1234},
		MaxTopReputersToReward:          []uint64{1234},
		CreateTopicFee:                  []cosmosMath.Int{cosmosMath.NewInt(1234)},
		RegistrationFee:                 []cosmosMath.Int{cosmosMath.NewInt(1234)},
		DefaultPageLimit:                []uint64{1234},
		MaxPageLimit:                    []uint64{1234},
		MinEpochLengthRecordLimit:       []int64{1234},
		MaxSerializedMsgLength:          []int64{1234},
		PRewardInference:                []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
		PRewardForecast:                 []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
		PRewardReputer:                  []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
		CRewardInference:                []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
		CRewardForecast:                 []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
		CNorm:                           []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
	}

	updateMsg := &types.MsgUpdateParams{
		Sender: adminAddr.String(),
		Params: newParams,
	}

	response, err := msgServer.UpdateParams(ctx, updateMsg)
	require.NoError(err)
	require.NotNil(response)

	updatedParams, err := keeper.GetParams(ctx)
	require.NoError(err)

	require.Equal(newParams.Version[0], updatedParams.Version)
	require.Equal(newParams.MinTopicWeight[0], updatedParams.MinTopicWeight)
	require.Equal(newParams.RequiredMinimumStake[0], updatedParams.RequiredMinimumStake)
	require.Equal(newParams.RemoveStakeDelayWindow[0], updatedParams.RemoveStakeDelayWindow)
	require.Equal(newParams.MinEpochLength[0], updatedParams.MinEpochLength)
	require.Equal(newParams.BetaEntropy[0], updatedParams.BetaEntropy)
	require.Equal(newParams.LearningRate[0], updatedParams.LearningRate)
	require.Equal(newParams.MaxGradientThreshold[0], updatedParams.MaxGradientThreshold)
	require.Equal(newParams.MinStakeFraction[0], updatedParams.MinStakeFraction)
	require.Equal(newParams.EpsilonReputer[0], updatedParams.EpsilonReputer)
	require.Equal(newParams.MaxUnfulfilledWorkerRequests[0], updatedParams.MaxUnfulfilledWorkerRequests)
	require.Equal(newParams.MaxUnfulfilledReputerRequests[0], updatedParams.MaxUnfulfilledReputerRequests)
	require.Equal(newParams.TopicRewardStakeImportance[0], updatedParams.TopicRewardStakeImportance)
	require.Equal(newParams.TopicRewardFeeRevenueImportance[0], updatedParams.TopicRewardFeeRevenueImportance)
	require.Equal(newParams.TopicRewardAlpha[0], updatedParams.TopicRewardAlpha)
	require.Equal(newParams.TaskRewardAlpha[0], updatedParams.TaskRewardAlpha)
	require.Equal(newParams.ValidatorsVsAlloraPercentReward[0], updatedParams.ValidatorsVsAlloraPercentReward)
	require.Equal(newParams.MaxSamplesToScaleScores[0], updatedParams.MaxSamplesToScaleScores)
	require.Equal(newParams.MaxTopInferersToReward[0], updatedParams.MaxTopInferersToReward)
	require.Equal(newParams.MaxTopForecastersToReward[0], updatedParams.MaxTopForecastersToReward)
	require.Equal(newParams.MaxTopReputersToReward[0], updatedParams.MaxTopReputersToReward)
	require.Equal(newParams.CreateTopicFee[0], updatedParams.CreateTopicFee)
	require.Equal(newParams.GradientDescentMaxIters[0], updatedParams.GradientDescentMaxIters)
	require.Equal(newParams.RegistrationFee[0], updatedParams.RegistrationFee)
	require.Equal(newParams.DefaultPageLimit[0], updatedParams.DefaultPageLimit)
	require.Equal(newParams.MaxPageLimit[0], updatedParams.MaxPageLimit)
	require.Equal(newParams.MinEpochLengthRecordLimit[0], updatedParams.MinEpochLengthRecordLimit)
	require.Equal(newParams.MaxSerializedMsgLength[0], updatedParams.MaxSerializedMsgLength)
	require.Equal(newParams.PRewardInference[0], updatedParams.PRewardInference)
	require.Equal(newParams.PRewardForecast[0], updatedParams.PRewardForecast)
	require.Equal(newParams.PRewardReputer[0], updatedParams.PRewardReputer)
	require.Equal(newParams.CRewardInference[0], updatedParams.CRewardInference)
	require.Equal(newParams.CRewardForecast[0], updatedParams.CRewardForecast)
	require.Equal(newParams.CNorm[0], updatedParams.CNorm)
}

func (s *MsgServerTestSuite) TestUpdateParamsNonWhitelistedUser() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Setup a non-whitelisted sender address
	nonAdminPrivateKey := secp256k1.GenPrivKey()
	nonAdminAddr := sdk.AccAddress(nonAdminPrivateKey.PubKey().Address())

	// Define new parameters to update
	newParams := &types.OptionalParams{
		Version: []string{"2.0"}, // example of changing the version
	}

	// Creating the MsgUpdateParams message with a non-whitelisted user
	updateMsg := &types.MsgUpdateParams{
		Sender: nonAdminAddr.String(),
		Params: newParams,
	}

	// Attempt to call UpdateParams with a non-whitelisted sender
	response, err := msgServer.UpdateParams(ctx, updateMsg)

	// Expect an error since the sender is not whitelisted
	require.Nil(response, "Response should be nil when access is denied")
	require.Error(err, types.ErrNotWhitelistAdmin, "Expected an error for non-whitelisted sender")
}
