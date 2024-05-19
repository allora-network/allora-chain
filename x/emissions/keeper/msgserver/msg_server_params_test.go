package msgserver_test

import (
	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestUpdateParams() {
	ctx, msgServer := s.ctx, s.msgServer
	keeper := s.emissionsKeeper
	require := s.Require()

	// Setup a sender address
	adminPrivateKey := secp256k1.GenPrivKey()
	adminAddr := sdk.AccAddress(adminPrivateKey.PubKey().Address())

	keeper.AddWhitelistAdmin(ctx, adminAddr.String())

	existingParams, err := keeper.GetParams(ctx)
	require.NoError(err)

	// New parameters to update
	newParams := &types.OptionalParams{
		MaxTopicsPerBlock: []uint64{20},
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

	require.Equal(uint64(20), updatedParams.MaxTopicsPerBlock)

	require.Equal(existingParams.Version, updatedParams.Version)
}

func (s *KeeperTestSuite) TestUpdateAllParams() {
	ctx, msgServer := s.ctx, s.msgServer
	keeper := s.emissionsKeeper
	require := s.Require()

	adminPrivateKey := secp256k1.GenPrivKey()
	adminAddr := sdk.AccAddress(adminPrivateKey.PubKey().Address())

	keeper.AddWhitelistAdmin(ctx, adminAddr.String())

	newParams := &types.OptionalParams{
		Version:                         []string{"1234"},
		MinTopicWeight:                  []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
		MaxTopicsPerBlock:               []uint64{1234},
		MaxMissingInferencePercent:      []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
		RequiredMinimumStake:            []cosmosMath.Uint{cosmosMath.NewUint(1234)},
		RemoveStakeDelayWindow:          []int64{1234},
		MinEpochLength:                  []int64{1234},
		BetaEntropy:                     []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
		LearningRate:                    []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
		MaxGradientThreshold:            []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
		MinStakeFraction:                []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
		Epsilon:                         []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
		PInferenceSynthesis:             []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
		PRewardSpread:                   []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
		AlphaRegret:                     []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
		MaxUnfulfilledWorkerRequests:    []uint64{1234},
		MaxUnfulfilledReputerRequests:   []uint64{1234},
		TopicRewardStakeImportance:      []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
		TopicRewardFeeRevenueImportance: []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
		TopicRewardAlpha:                []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
		TaskRewardAlpha:                 []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
		ValidatorsVsAlloraPercentReward: []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
		MaxSamplesToScaleScores:         []uint64{1234},
		MaxTopWorkersToReward:           []uint64{1234},
		MaxTopReputersToReward:          []uint64{1234},
		CreateTopicFee:                  []cosmosMath.Int{cosmosMath.NewInt(1234)},
		SigmoidA:                        []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
		SigmoidB:                        []alloraMath.Dec{alloraMath.NewDecFromInt64(1234)},
		GradientDescentMaxIters:         []uint64{1234},
		MaxRetriesToFulfilNoncesWorker:  []int64{1234},
		MaxRetriesToFulfilNoncesReputer: []int64{1234},
		TopicPageLimit:                  []uint64{1234},
		MaxTopicPages:                   []uint64{1234},
		RegistrationFee:                 []cosmosMath.Int{cosmosMath.NewInt(1234)},
		DefaultLimit:                    []uint64{1234},
		MaxLimit:                        []uint64{1234},
		MinEpochLengthRecordLimit:       []int64{1234},
		MaxSerializedMsgLength:          []int64{1234},
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
	require.Equal(newParams.MaxTopicsPerBlock[0], updatedParams.MaxTopicsPerBlock)
	require.Equal(newParams.MaxMissingInferencePercent[0], updatedParams.MaxMissingInferencePercent)
	require.Equal(newParams.RequiredMinimumStake[0], updatedParams.RequiredMinimumStake)
	require.Equal(newParams.RemoveStakeDelayWindow[0], updatedParams.RemoveStakeDelayWindow)
	require.Equal(newParams.MinEpochLength[0], updatedParams.MinEpochLength)
	require.Equal(newParams.BetaEntropy[0], updatedParams.BetaEntropy)
	require.Equal(newParams.LearningRate[0], updatedParams.LearningRate)
	require.Equal(newParams.MaxGradientThreshold[0], updatedParams.MaxGradientThreshold)
	require.Equal(newParams.MinStakeFraction[0], updatedParams.MinStakeFraction)
	require.Equal(newParams.Epsilon[0], updatedParams.Epsilon)
	require.Equal(newParams.PInferenceSynthesis[0], updatedParams.PInferenceSynthesis)
	require.Equal(newParams.PRewardSpread[0], updatedParams.PRewardSpread)
	require.Equal(newParams.AlphaRegret[0], updatedParams.AlphaRegret)
	require.Equal(newParams.MaxUnfulfilledWorkerRequests[0], updatedParams.MaxUnfulfilledWorkerRequests)
	require.Equal(newParams.MaxUnfulfilledReputerRequests[0], updatedParams.MaxUnfulfilledReputerRequests)
	require.Equal(newParams.TopicRewardStakeImportance[0], updatedParams.TopicRewardStakeImportance)
	require.Equal(newParams.TopicRewardFeeRevenueImportance[0], updatedParams.TopicRewardFeeRevenueImportance)
	require.Equal(newParams.TopicRewardAlpha[0], updatedParams.TopicRewardAlpha)
	require.Equal(newParams.TaskRewardAlpha[0], updatedParams.TaskRewardAlpha)
	require.Equal(newParams.ValidatorsVsAlloraPercentReward[0], updatedParams.ValidatorsVsAlloraPercentReward)
	require.Equal(newParams.MaxSamplesToScaleScores[0], updatedParams.MaxSamplesToScaleScores)
	require.Equal(newParams.MaxTopWorkersToReward[0], updatedParams.MaxTopWorkersToReward)
	require.Equal(newParams.MaxTopReputersToReward[0], updatedParams.MaxTopReputersToReward)
	require.Equal(newParams.CreateTopicFee[0], updatedParams.CreateTopicFee)
	require.Equal(newParams.SigmoidA[0], updatedParams.SigmoidA)
	require.Equal(newParams.SigmoidB[0], updatedParams.SigmoidB)
	require.Equal(newParams.GradientDescentMaxIters[0], updatedParams.GradientDescentMaxIters)
	require.Equal(newParams.MaxRetriesToFulfilNoncesWorker[0], updatedParams.MaxRetriesToFulfilNoncesWorker)
	require.Equal(newParams.MaxRetriesToFulfilNoncesReputer[0], updatedParams.MaxRetriesToFulfilNoncesReputer)
	require.Equal(newParams.TopicPageLimit[0], updatedParams.TopicPageLimit)
	require.Equal(newParams.MaxTopicPages[0], updatedParams.MaxTopicPages)
	require.Equal(newParams.RegistrationFee[0], updatedParams.RegistrationFee)
	require.Equal(newParams.DefaultLimit[0], updatedParams.DefaultLimit)
	require.Equal(newParams.MaxLimit[0], updatedParams.MaxLimit)
	require.Equal(newParams.MinEpochLengthRecordLimit[0], updatedParams.MinEpochLengthRecordLimit)
	require.Equal(newParams.MaxSerializedMsgLength[0], updatedParams.MaxSerializedMsgLength)
}

func (s *KeeperTestSuite) TestUpdateParamsNonWhitelistedUser() {
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
