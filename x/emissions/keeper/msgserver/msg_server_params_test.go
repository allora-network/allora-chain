package msgserver_test

import (
	cosmosMath "cosmossdk.io/math"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *MsgServerTestSuite) TestUpdateAllParams() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	keeper := s.EmissionsKeeper()
	require := s.Require()

	adminPrivateKey := secp256k1.GenPrivKey()
	adminAddr := sdk.AccAddress(adminPrivateKey.PubKey().Address())

	err := keeper.AddWhitelistAdmin(ctx, adminAddr.String())
	require.NoError(err)

	newParams := &types.OptionalParams{ //nolint:exhaustruct
		Version:                             []string{"1234"},
		MinTopicWeight:                      []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
		RequiredMinimumStake:                []cosmosMath.Int{cosmosMath.NewInt(1234)},
		RemoveStakeDelayWindow:              []int64{1234},
		MinEpochLength:                      []int64{1234},
		BetaEntropy:                         []alloraMath.Dec{alloraMath.MustNewDecFromString(".1234")},
		LearningRate:                        []alloraMath.Dec{alloraMath.MustNewDecFromString(".1234")},
		GradientDescentMaxIters:             []uint64{1234},
		MaxGradientThreshold:                []alloraMath.Dec{alloraMath.MustNewDecFromString(".1234")},
		MinStakeFraction:                    []alloraMath.Dec{alloraMath.MustNewDecFromString(".1234")},
		EpsilonReputer:                      []alloraMath.Dec{alloraMath.MustNewDecFromString(".1234")},
		MaxUnfulfilledWorkerRequests:        []uint64{1234},
		MaxUnfulfilledReputerRequests:       []uint64{1234},
		TopicRewardStakeImportance:          []alloraMath.Dec{alloraMath.MustNewDecFromString(".1234")},
		TopicRewardFeeRevenueImportance:     []alloraMath.Dec{alloraMath.MustNewDecFromString(".1234")},
		TopicRewardAlpha:                    []alloraMath.Dec{alloraMath.MustNewDecFromString(".1234")},
		TaskRewardAlpha:                     []alloraMath.Dec{alloraMath.MustNewDecFromString(".1234")},
		ValidatorsVsAlloraPercentReward:     []alloraMath.Dec{alloraMath.MustNewDecFromString(".1234")},
		MaxSamplesToScaleScores:             []uint64{1234},
		MaxTopInferersToReward:              []uint64{1234},
		MaxTopForecastersToReward:           []uint64{1234},
		MaxTopReputersToReward:              []uint64{1234},
		CreateTopicFee:                      []cosmosMath.Int{cosmosMath.NewInt(1234)},
		RegistrationFee:                     []cosmosMath.Int{cosmosMath.NewInt(1234)},
		DefaultPageLimit:                    []uint64{1234},
		MaxPageLimit:                        []uint64{1234},
		MinEpochLengthRecordLimit:           []int64{1234},
		MaxSerializedMsgLength:              []int64{1234},
		PRewardInference:                    []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
		PRewardForecast:                     []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
		PRewardReputer:                      []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
		CRewardInference:                    []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
		CRewardForecast:                     []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
		BlocksPerMonth:                      []uint64{1234},
		HalfMaxProcessStakeRemovalsEndBlock: []uint64{1234},
		DataSendingFee:                      []cosmosMath.Int{cosmosMath.NewInt(1234)},
		EpsilonSafeDiv:                      []alloraMath.Dec{alloraMath.MustNewDecFromString(".1234")},
		MaxElementsPerForecast:              []uint64{1234},
		MaxActiveTopicsPerBlock:             []uint64{1234},
		MaxStringLength:                     []uint64{1234},
		InitialRegretQuantile:               []alloraMath.Dec{alloraMath.ZeroDec()},
		PNormSafeDiv:                        []alloraMath.Dec{alloraMath.ZeroDec()},
		GlobalWhitelistEnabled:              []bool{true},
		TopicCreatorWhitelistEnabled:        []bool{true},
		MinExperiencedWorkerRegrets:         []uint64{1234},
		InferenceOutlierDetectionThreshold:  []alloraMath.Dec{alloraMath.MustNewDecFromString("11")},
		InferenceOutlierDetectionAlpha:      []alloraMath.Dec{alloraMath.MustNewDecFromString("0.2")},
		LambdaInitialScore:                  []alloraMath.Dec{alloraMath.MustNewDecFromString("2")},
		GlobalWorkerWhitelistEnabled:        []bool{true},
		GlobalReputerWhitelistEnabled:       []bool{true},
		GlobalAdminWhitelistAppended:        []bool{true},
		MaxWhitelistInputArrayLength:        []uint64{10},
		MinWeightThresholdForStdnorm:        []alloraMath.Dec{alloraMath.MustNewDecFromString("0.000001")},
		MaxLabelsPerSubmission:              []uint64{7},
		MaxWhitelistLabelSize:               []uint64{9},
	}

	updateMsg := &types.UpdateParamsRequest{
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
	require.Equal(newParams.BlocksPerMonth[0], updatedParams.BlocksPerMonth)
	require.Equal(newParams.HalfMaxProcessStakeRemovalsEndBlock[0], updatedParams.HalfMaxProcessStakeRemovalsEndBlock)
	require.Equal(newParams.DataSendingFee[0], updatedParams.DataSendingFee)
	require.Equal(newParams.EpsilonSafeDiv[0], updatedParams.EpsilonSafeDiv)
	require.Equal(newParams.MaxElementsPerForecast[0], updatedParams.MaxElementsPerForecast)
	require.Equal(newParams.MaxActiveTopicsPerBlock[0], updatedParams.MaxActiveTopicsPerBlock)
	require.Equal(newParams.MaxStringLength[0], updatedParams.MaxStringLength)
	require.Equal(newParams.InitialRegretQuantile[0], updatedParams.InitialRegretQuantile)
	require.Equal(newParams.PNormSafeDiv[0], updatedParams.PNormSafeDiv)
	require.Equal(newParams.GlobalWhitelistEnabled[0], updatedParams.GlobalWhitelistEnabled)
	require.Equal(newParams.TopicCreatorWhitelistEnabled[0], updatedParams.TopicCreatorWhitelistEnabled)
	require.Equal(newParams.MinExperiencedWorkerRegrets[0], updatedParams.MinExperiencedWorkerRegrets)
	require.Equal(newParams.InferenceOutlierDetectionThreshold[0], updatedParams.InferenceOutlierDetectionThreshold)
	require.Equal(newParams.InferenceOutlierDetectionAlpha[0], updatedParams.InferenceOutlierDetectionAlpha)
	require.Equal(newParams.LambdaInitialScore[0], updatedParams.LambdaInitialScore)
	require.Equal(newParams.GlobalWorkerWhitelistEnabled[0], updatedParams.GlobalWorkerWhitelistEnabled)
	require.Equal(newParams.GlobalReputerWhitelistEnabled[0], updatedParams.GlobalReputerWhitelistEnabled)
	require.Equal(newParams.GlobalAdminWhitelistAppended[0], updatedParams.GlobalAdminWhitelistAppended)
	require.Equal(newParams.MaxWhitelistInputArrayLength[0], updatedParams.MaxWhitelistInputArrayLength)
	require.Equal(newParams.MinWeightThresholdForStdnorm[0], updatedParams.MinWeightThresholdForStdnorm)
	require.Equal(newParams.MaxLabelsPerSubmission[0], updatedParams.MaxLabelsPerSubmission)
	require.Equal(newParams.MaxWhitelistLabelSize[0], updatedParams.MaxWhitelistLabelSize)
}

func (s *MsgServerTestSuite) TestUpdateParamsNonWhitelistedUser() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	// Setup a non-whitelisted sender address
	nonAdminPrivateKey := secp256k1.GenPrivKey()
	nonAdminAddr := sdk.AccAddress(nonAdminPrivateKey.PubKey().Address())

	// Define new parameters to update
	newParams := &types.OptionalParams{ //nolint:exhaustruct
		Version: []string{"2.0"}, // example of changing the version
		// don't update the other params
	}

	// Creating the UpdateParamsRequest message with a non-whitelisted user
	updateMsg := &types.UpdateParamsRequest{
		Sender: nonAdminAddr.String(),
		Params: newParams,
	}

	// Attempt to call UpdateParams with a non-whitelisted sender
	response, err := msgServer.UpdateParams(ctx, updateMsg)

	// Expect an error since the sender is not whitelisted
	require.Nil(response, "Response should be nil when access is denied")
	require.ErrorIs(err, types.ErrNotPermittedToUpdateParams, "Expected an error for non-whitelisted sender")
}

func (s *MsgServerTestSuite) TestUpdateParams_LabelLimitOptionalBehavior() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	k := s.EmissionsKeeper()
	require := s.Require()

	adminPrivateKey := secp256k1.GenPrivKey()
	adminAddr := sdk.AccAddress(adminPrivateKey.PubKey().Address())
	require.NoError(k.AddWhitelistAdmin(ctx, adminAddr.String()))

	before, err := k.GetParams(ctx)
	require.NoError(err)

	_, err = msgServer.UpdateParams(ctx, &types.UpdateParamsRequest{
		Sender: adminAddr.String(),
		Params: &types.OptionalParams{ //nolint:exhaustruct // Added nolint because update tests intentionally set only the field being patched.
			MaxLabelsPerSubmission: []uint64{before.MaxLabelsPerSubmission + 1},
			MaxWhitelistLabelSize:  []uint64{before.MaxWhitelistLabelSize + 1},
		},
	})
	require.NoError(err)

	afterEpochOnly, err := k.GetParams(ctx)
	require.NoError(err)
	require.Equal(before.MaxLabelsPerSubmission+1, afterEpochOnly.MaxLabelsPerSubmission)
	require.Equal(before.MaxWhitelistLabelSize+1, afterEpochOnly.MaxWhitelistLabelSize)

	_, err = msgServer.UpdateParams(ctx, &types.UpdateParamsRequest{
		Sender: adminAddr.String(),
		Params: &types.OptionalParams{}, //nolint:exhaustruct // Added nolint because empty OptionalParams is required to verify no-op update behavior.
	})
	require.NoError(err)

	afterNoChange, err := k.GetParams(ctx)
	require.NoError(err)
	require.Equal(afterEpochOnly.MaxLabelsPerSubmission, afterNoChange.MaxLabelsPerSubmission)
	require.Equal(afterEpochOnly.MaxWhitelistLabelSize, afterNoChange.MaxWhitelistLabelSize)
}
