package keeper_test

import (
	cosmossdk_io_math "cosmossdk.io/math"

	alloraMath "github.com/allora-network/allora-chain/math"
	emissionstestutil "github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// newFreshGenesisSuite creates an isolated keeper test suite with a fresh store/context.
// Round-trip genesis tests use it to verify InitGenesis rebuilds state instead of reusing existing state.
func (s *KeeperTestSuite) newFreshGenesisSuite() *KeeperTestSuite {
	fresh := &KeeperTestSuite{
		TestSuite: emissionstestutil.NewTestSuite("emissions_keeper"),
	}
	fresh.SetT(s.T())
	fresh.SetupTest()
	return fresh
}

// at minimum test that an import can be done from an export without error
func (s *KeeperTestSuite) TestImportExportGenesisNoError() {
	testAddr := s.AddrsStr(0)
	err := s.WhitelistsKeeper().AddWhitelistAdmin(s.Ctx(), testAddr)
	s.Require().NoError(err)

	err = s.StakingKeeper().SetTopicStake(s.Ctx(), 1, cosmossdk_io_math.OneInt())
	s.Require().NoError(err)
	genesisState, err := s.EmissionsKeeper().ExportGenesis(s.Ctx())
	s.Require().NoError(err)

	fresh := s.newFreshGenesisSuite()

	err = fresh.EmissionsKeeper().InitGenesis(fresh.Ctx(), genesisState)
	s.Require().NoError(err)

	for _, addr := range types.DefaultCoreTeamAddresses() {
		admin, err := fresh.WhitelistsKeeper().IsWhitelistAdmin(fresh.Ctx(), addr)
		s.Require().NoError(err)
		s.Require().Equal(admin, true)
	}
	admin, err := fresh.WhitelistsKeeper().IsWhitelistAdmin(fresh.Ctx(), testAddr)
	s.Require().NoError(err)
	s.Require().Equal(admin, true)
}

func (s *KeeperTestSuite) TestGenesisRoundTripComprehensive() {
	ctx := s.Ctx()
	topicId := uint64(1)
	blockHeight := int64(100)
	addr0 := s.AddrsStr(0)
	addr1 := s.AddrsStr(1)
	addr2 := s.AddrsStr(2)

	// Set up a topic first (required for many fields)
	topic := types.Topic{
		Id:                       topicId,
		Creator:                  addr0,
		Metadata:                 "test-topic",
		LossMethod:               "mse",
		EpochLastEnded:           0,
		EpochLength:              12,
		GroundTruthLag:           12,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:            false,
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		InitialRegret:            alloraMath.ZeroDec(),
		WorkerSubmissionWindow:   10,
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		CNorm:                    alloraMath.ZeroDec(),
		TopicType:                types.TopicType_TOPIC_TYPE_REGRESSION,
		OutputArity:              types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
		RequireUnity:             false,
		UnityTolerance:           alloraMath.Dec{},
		MaxLabelsPerSubmission:   types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:           nil,
		LabelDefaultValue:        alloraMath.ZeroDec(),
		LabelCaseSensitive:       false,
	}
	err := s.TopicKeeper().SetTopic(ctx, topicId, topic)
	s.Require().NoError(err)
	err = s.TopicKeeper().ActivateTopic(ctx, topicId)
	s.Require().NoError(err)

	// PARAMS — use existing defaults, they're already set

	// TOPIC
	err = s.TopicKeeper().SetTopicRewardNonce(ctx, topicId, blockHeight)
	s.Require().NoError(err)
	err = s.TopicKeeper().SetPreviousTopicWeight(ctx, topicId, alloraMath.MustNewDecFromString("0.5"))
	s.Require().NoError(err)
	err = s.TopicKeeper().SetLastDripBlock(ctx, topicId, blockHeight)
	s.Require().NoError(err)
	err = s.TopicKeeper().SetWorkerTopicLastCommit(ctx, topicId, blockHeight, &types.Nonce{BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = s.TopicKeeper().SetReputerTopicLastCommit(ctx, topicId, blockHeight, &types.Nonce{BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = s.TopicKeeper().SetTopicToNextPossibleChurningBlock(ctx, topicId, blockHeight+10)
	s.Require().NoError(err)
	err = s.TopicKeeper().SetLastMedianInferences(ctx, topicId, alloraMath.MustNewDecFromString("1.5"))
	s.Require().NoError(err)
	err = s.TopicKeeper().SetMadInferences(ctx, topicId, alloraMath.MustNewDecFromString("0.3"))
	s.Require().NoError(err)
	err = s.TopicKeeper().SetTotalSumPreviousTopicWeights(ctx, alloraMath.MustNewDecFromString("100.0"))
	s.Require().NoError(err)
	label := "testlabel"
	params, err := s.ParamsKeeper().GetParams(ctx)
	s.Require().NoError(err)
	_, _, err = s.TopicKeeper().RegisterEpochLabels(
		ctx,
		topicId,
		false,
		blockHeight,
		[]string{label},
		params.MaxCanonicalLabelByteLength,
		params.MaxEpochLabelRegistrySize,
	)
	s.Require().NoError(err)

	// WORKER
	err = s.WorkerKeeper().SetTopicWorker(ctx, topicId, addr0)
	s.Require().NoError(err)
	err = s.WorkerKeeper().AddActiveInferer(ctx, topicId, addr0)
	s.Require().NoError(err)
	err = s.WorkerKeeper().AddActiveForecaster(ctx, topicId, addr1)
	s.Require().NoError(err)

	// REPUTER LOSS
	err = s.ReputerLossKeeper().SetTopicReputer(ctx, topicId, addr1)
	s.Require().NoError(err)
	err = s.ReputerLossKeeper().AddActiveReputer(ctx, topicId, addr1)
	s.Require().NoError(err)

	// STAKING
	err = s.StakingKeeper().SetTotalStake(ctx, cosmossdk_io_math.NewInt(1000))
	s.Require().NoError(err)
	err = s.StakingKeeper().SetTopicStake(ctx, topicId, cosmossdk_io_math.NewInt(500))
	s.Require().NoError(err)
	err = s.StakingKeeper().SetStakeReputerAuthority(ctx, topicId, addr1, cosmossdk_io_math.NewInt(200))
	s.Require().NoError(err)
	err = s.StakingKeeper().SetDelegateRewardPerShare(ctx, topicId, addr1, alloraMath.MustNewDecFromString("1.5"))
	s.Require().NoError(err)

	// SCORES
	err = s.ScoresKeeper().SetInfererScoreEma(ctx, topicId, addr0, types.Score{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Address:     addr0,
		Score:       alloraMath.MustNewDecFromString("0.9"),
	})
	s.Require().NoError(err)
	err = s.ScoresKeeper().SetForecasterScoreEma(ctx, topicId, addr1, types.Score{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Address:     addr1,
		Score:       alloraMath.MustNewDecFromString("0.8"),
	})
	s.Require().NoError(err)
	err = s.ScoresKeeper().SetReputerScoreEma(ctx, topicId, addr1, types.Score{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Address:     addr1,
		Score:       alloraMath.MustNewDecFromString("0.85"),
	})
	s.Require().NoError(err)
	err = s.ScoresKeeper().SetListeningCoefficient(ctx, topicId, addr1, types.ListeningCoefficient{
		Coefficient: alloraMath.MustNewDecFromString("0.5"),
	})
	s.Require().NoError(err)
	err = s.ScoresKeeper().SetPreviousReputerRewardFraction(ctx, topicId, addr1, alloraMath.MustNewDecFromString("0.3"))
	s.Require().NoError(err)
	err = s.ScoresKeeper().SetPreviousInferenceRewardFraction(ctx, topicId, addr0, alloraMath.MustNewDecFromString("0.4"))
	s.Require().NoError(err)
	err = s.ScoresKeeper().SetPreviousForecastRewardFraction(ctx, topicId, addr1, alloraMath.MustNewDecFromString("0.3"))
	s.Require().NoError(err)
	err = s.ScoresKeeper().SetPreviousPercentageRewardToStakedReputers(ctx, alloraMath.MustNewDecFromString("0.25"))
	s.Require().NoError(err)
	err = s.ScoresKeeper().SetPreviousForecasterScoreRatio(ctx, topicId, alloraMath.MustNewDecFromString("0.7"))
	s.Require().NoError(err)
	err = s.ScoresKeeper().SetPreviousTopicQuantileInfererScoreEma(ctx, topicId, alloraMath.MustNewDecFromString("0.5"))
	s.Require().NoError(err)
	err = s.ScoresKeeper().SetPreviousTopicQuantileForecasterScoreEma(ctx, topicId, alloraMath.MustNewDecFromString("0.6"))
	s.Require().NoError(err)
	err = s.ScoresKeeper().SetPreviousTopicQuantileReputerScoreEma(ctx, topicId, alloraMath.MustNewDecFromString("0.7"))
	s.Require().NoError(err)
	err = s.ScoresKeeper().SetLowestInfererScoreEma(ctx, topicId, types.Score{
		TopicId: topicId, BlockHeight: blockHeight, Address: addr0,
		Score: alloraMath.MustNewDecFromString("0.1"),
	})
	s.Require().NoError(err)
	err = s.ScoresKeeper().SetLowestForecasterScoreEma(ctx, topicId, types.Score{
		TopicId: topicId, BlockHeight: blockHeight, Address: addr1,
		Score: alloraMath.MustNewDecFromString("0.2"),
	})
	s.Require().NoError(err)
	err = s.ScoresKeeper().SetLowestReputerScoreEma(ctx, topicId, types.Score{
		TopicId: topicId, BlockHeight: blockHeight, Address: addr1,
		Score: alloraMath.MustNewDecFromString("0.15"),
	})
	s.Require().NoError(err)

	// REGRETS
	err = s.RegretsKeeper().SetInfererNetworkRegret(ctx, topicId, addr0, types.TimestampedValue{
		BlockHeight: blockHeight, Value: alloraMath.MustNewDecFromString("0.1"),
	})
	s.Require().NoError(err)
	err = s.RegretsKeeper().SetNaiveInfererNetworkRegret(ctx, topicId, addr0, types.TimestampedValue{
		BlockHeight: blockHeight, Value: alloraMath.MustNewDecFromString("0.2"),
	})
	s.Require().NoError(err)
	err = s.RegretsKeeper().SetForecasterNetworkRegret(ctx, topicId, addr1, types.TimestampedValue{
		BlockHeight: blockHeight, Value: alloraMath.MustNewDecFromString("0.15"),
	})
	s.Require().NoError(err)
	err = s.RegretsKeeper().SetOneOutInfererInfererNetworkRegret(ctx, topicId, addr0, addr1, types.TimestampedValue{
		BlockHeight: blockHeight, Value: alloraMath.MustNewDecFromString("0.05"),
	})
	s.Require().NoError(err)
	err = s.RegretsKeeper().SetOneOutForecasterInfererNetworkRegret(ctx, topicId, addr1, addr0, types.TimestampedValue{
		BlockHeight: blockHeight, Value: alloraMath.MustNewDecFromString("0.06"),
	})
	s.Require().NoError(err)
	err = s.RegretsKeeper().SetOneOutForecasterForecasterNetworkRegret(ctx, topicId, addr1, addr2, types.TimestampedValue{
		BlockHeight: blockHeight, Value: alloraMath.MustNewDecFromString("0.07"),
	})
	s.Require().NoError(err)
	err = s.RegretsKeeper().SetOneInForecasterNetworkRegret(ctx, topicId, addr1, addr0, types.TimestampedValue{
		BlockHeight: blockHeight, Value: alloraMath.MustNewDecFromString("0.08"),
	})
	s.Require().NoError(err)

	// WEIGHTS
	err = s.WeightsKeeper().SetLatestRegretScale(ctx, topicId, alloraMath.MustNewDecFromString("2.0"))
	s.Require().NoError(err)
	err = s.WeightsKeeper().SetLatestInfererWeight(ctx, topicId, addr0, alloraMath.MustNewDecFromString("0.6"))
	s.Require().NoError(err)
	err = s.WeightsKeeper().SetLatestForecasterWeight(ctx, topicId, addr1, alloraMath.MustNewDecFromString("0.4"))
	s.Require().NoError(err)

	// WHITELISTS
	err = s.WhitelistsKeeper().AddWhitelistAdmin(ctx, addr0)
	s.Require().NoError(err)
	err = s.WhitelistsKeeper().AddToGlobalWhitelist(ctx, addr0)
	s.Require().NoError(err)
	err = s.WhitelistsKeeper().AddToGlobalWorkerWhitelist(ctx, addr1)
	s.Require().NoError(err)
	err = s.WhitelistsKeeper().AddToGlobalReputerWhitelist(ctx, addr2)
	s.Require().NoError(err)
	err = s.WhitelistsKeeper().AddToTopicCreatorWhitelist(ctx, addr0)
	s.Require().NoError(err)

	// KEEPER-LEVEL
	err = s.EmissionsKeeper().SetRewardCurrentBlockEmission(ctx, cosmossdk_io_math.NewInt(999))
	s.Require().NoError(err)

	// EXPORT
	genesisState, err := s.EmissionsKeeper().ExportGenesis(ctx)
	s.Require().NoError(err)
	s.Require().NotNil(genesisState)

	// Verify key exported fields before re-import
	s.Require().True(genesisState.TotalStake.Equal(cosmossdk_io_math.NewInt(1000)))
	s.Require().Len(genesisState.TopicStake, 1)
	s.Require().True(genesisState.TopicStake[0].Int.Equal(cosmossdk_io_math.NewInt(500)))
	s.Require().True(genesisState.RewardCurrentBlockEmission.Equal(cosmossdk_io_math.NewInt(999)))
	s.Require().NotEmpty(genesisState.ActiveTopics)
	s.Require().True(genesisState.TotalSumPreviousTopicWeights.Equal(alloraMath.MustNewDecFromString("100.0")))

	fresh := s.newFreshGenesisSuite()

	// RE-IMPORT on a fresh store to ensure InitGenesis actually reconstructs state.
	err = fresh.EmissionsKeeper().InitGenesis(fresh.Ctx(), genesisState)
	s.Require().NoError(err)

	// VERIFY round-trip by exporting again and comparing key fields
	genesisState2, err := fresh.EmissionsKeeper().ExportGenesis(fresh.Ctx())
	s.Require().NoError(err)

	// Params
	s.Require().Equal(genesisState.Params, genesisState2.Params)

	// Topic
	s.Require().Equal(genesisState.NextTopicId, genesisState2.NextTopicId)
	s.Require().Len(genesisState2.Topics, len(genesisState.Topics))
	s.Require().NotEmpty(genesisState2.ActiveTopics)

	// Labels
	s.Require().Equal(len(genesisState2.TopicLabelRegistries), 1)
	s.Require().Equal(len(genesisState2.TopicLabelRegistries), len(genesisState.TopicLabelRegistries))
	s.Require().Len(genesisState2.TopicLabelRegistries[0].EpochLabelRegistry.Labels, 1)
	s.Require().Equal(
		len(genesisState2.TopicLabelRegistries[0].EpochLabelRegistry.Labels),
		len(genesisState.TopicLabelRegistries[0].EpochLabelRegistry.Labels),
	)
	s.Require().Equal(
		genesisState2.TopicLabelRegistries[0].EpochLabelRegistry.Labels[0].Name,
		genesisState.TopicLabelRegistries[0].EpochLabelRegistry.Labels[0].Name,
	)

	// Staking
	s.Require().True(genesisState2.TotalStake.Equal(genesisState.TotalStake))
	s.Require().Len(genesisState2.TopicStake, len(genesisState.TopicStake))

	// Scores
	s.Require().Len(genesisState2.InfererScoreEmas, len(genesisState.InfererScoreEmas))
	s.Require().Len(genesisState2.ForecasterScoreEmas, len(genesisState.ForecasterScoreEmas))
	s.Require().Len(genesisState2.ReputerScoreEmas, len(genesisState.ReputerScoreEmas))
	s.Require().Len(genesisState2.ReputerListeningCoefficient, len(genesisState.ReputerListeningCoefficient))

	// Regrets
	s.Require().Len(genesisState2.LatestInfererNetworkRegrets, len(genesisState.LatestInfererNetworkRegrets))
	s.Require().Len(genesisState2.LatestForecasterNetworkRegrets, len(genesisState.LatestForecasterNetworkRegrets))
	s.Require().Len(genesisState2.LatestNaiveInfererNetworkRegrets, len(genesisState.LatestNaiveInfererNetworkRegrets))
	s.Require().Len(genesisState2.LatestOneOutInfererInfererNetworkRegrets, len(genesisState.LatestOneOutInfererInfererNetworkRegrets))

	// Weights
	s.Require().Len(genesisState2.LatestRegretStdNorm, len(genesisState.LatestRegretStdNorm))
	s.Require().Len(genesisState2.LatestInfererWeights, len(genesisState.LatestInfererWeights))
	s.Require().Len(genesisState2.LatestForecasterWeights, len(genesisState.LatestForecasterWeights))

	// Whitelists
	s.Require().NotEmpty(genesisState2.WhitelistAdmins)
	s.Require().NotEmpty(genesisState2.GlobalWhitelist)

	// Keeper-level
	s.Require().True(genesisState2.RewardCurrentBlockEmission.Equal(genesisState.RewardCurrentBlockEmission))
	s.Require().True(genesisState2.TotalSumPreviousTopicWeights.Equal(genesisState.TotalSumPreviousTopicWeights))
}

func (s *KeeperTestSuite) TestGenesisSubKeeperParamsRoundTrip() {
	ctx := s.Ctx()

	params, err := s.ParamsKeeper().GetParams(ctx)
	s.Require().NoError(err)

	genesisState, err := s.EmissionsKeeper().ExportGenesis(ctx)
	s.Require().NoError(err)
	s.Require().Equal(params, genesisState.Params)

	fresh := s.newFreshGenesisSuite()

	err = fresh.EmissionsKeeper().InitGenesis(fresh.Ctx(), genesisState)
	s.Require().NoError(err)

	params2, err := fresh.ParamsKeeper().GetParams(fresh.Ctx())
	s.Require().NoError(err)
	s.Require().Equal(params, params2)
}

func (s *KeeperTestSuite) TestGenesisSubKeeperStakingRoundTrip() {
	ctx := s.Ctx()
	topicId := uint64(1)

	err := s.StakingKeeper().SetTotalStake(ctx, cosmossdk_io_math.NewInt(5000))
	s.Require().NoError(err)
	err = s.StakingKeeper().SetTopicStake(ctx, topicId, cosmossdk_io_math.NewInt(2000))
	s.Require().NoError(err)
	err = s.StakingKeeper().SetStakeReputerAuthority(ctx, topicId, s.AddrsStr(1), cosmossdk_io_math.NewInt(1000))
	s.Require().NoError(err)
	err = s.StakingKeeper().SetDelegateRewardPerShare(ctx, topicId, s.AddrsStr(1), alloraMath.MustNewDecFromString("2.5"))
	s.Require().NoError(err)

	genesisState, err := s.EmissionsKeeper().ExportGenesis(ctx)
	s.Require().NoError(err)

	s.Require().True(genesisState.TotalStake.Equal(cosmossdk_io_math.NewInt(5000)))
	s.Require().Len(genesisState.TopicStake, 1)
	s.Require().True(genesisState.TopicStake[0].Int.Equal(cosmossdk_io_math.NewInt(2000)))
	s.Require().Len(genesisState.StakeReputerAuthority, 1)
	s.Require().Len(genesisState.DelegateRewardPerShare, 1)

	fresh := s.newFreshGenesisSuite()

	err = fresh.EmissionsKeeper().InitGenesis(fresh.Ctx(), genesisState)
	s.Require().NoError(err)

	totalStake, err := fresh.StakingKeeper().GetTotalStake(fresh.Ctx())
	s.Require().NoError(err)
	s.Require().True(totalStake.Equal(cosmossdk_io_math.NewInt(5000)))

	topicStake, err := fresh.StakingKeeper().GetTopicStake(fresh.Ctx(), topicId)
	s.Require().NoError(err)
	s.Require().True(topicStake.Equal(cosmossdk_io_math.NewInt(2000)))
}

func (s *KeeperTestSuite) TestGenesisSubKeeperRegretsRoundTrip() {
	ctx := s.Ctx()
	topicId := uint64(1)
	addr0 := s.AddrsStr(0)
	addr1 := s.AddrsStr(1)

	err := s.RegretsKeeper().SetInfererNetworkRegret(ctx, topicId, addr0, types.TimestampedValue{
		BlockHeight: 100, Value: alloraMath.MustNewDecFromString("0.123"),
	})
	s.Require().NoError(err)
	err = s.RegretsKeeper().SetForecasterNetworkRegret(ctx, topicId, addr1, types.TimestampedValue{
		BlockHeight: 100, Value: alloraMath.MustNewDecFromString("0.456"),
	})
	s.Require().NoError(err)
	err = s.RegretsKeeper().SetOneOutInfererInfererNetworkRegret(ctx, topicId, addr0, addr1, types.TimestampedValue{
		BlockHeight: 100, Value: alloraMath.MustNewDecFromString("0.789"),
	})
	s.Require().NoError(err)
	err = s.RegretsKeeper().SetOneInForecasterNetworkRegret(ctx, topicId, addr1, addr0, types.TimestampedValue{
		BlockHeight: 100, Value: alloraMath.MustNewDecFromString("0.321"),
	})
	s.Require().NoError(err)

	genesisState, err := s.EmissionsKeeper().ExportGenesis(ctx)
	s.Require().NoError(err)

	s.Require().Len(genesisState.LatestInfererNetworkRegrets, 1)
	s.Require().Len(genesisState.LatestForecasterNetworkRegrets, 1)
	s.Require().Len(genesisState.LatestOneOutInfererInfererNetworkRegrets, 1)
	s.Require().Len(genesisState.LatestOneInForecasterNetworkRegrets, 1)

	fresh := s.newFreshGenesisSuite()

	err = fresh.EmissionsKeeper().InitGenesis(fresh.Ctx(), genesisState)
	s.Require().NoError(err)

	genesisState2, err := fresh.EmissionsKeeper().ExportGenesis(fresh.Ctx())
	s.Require().NoError(err)

	s.Require().Len(genesisState2.LatestInfererNetworkRegrets, 1)
	s.Require().Equal(
		genesisState.LatestInfererNetworkRegrets[0].TimestampedValue.Value,
		genesisState2.LatestInfererNetworkRegrets[0].TimestampedValue.Value,
	)
}

func (s *KeeperTestSuite) TestGenesisSubKeeperWhitelistsRoundTrip() {
	ctx := s.Ctx()
	addr0 := s.AddrsStr(0)
	addr1 := s.AddrsStr(1)

	err := s.WhitelistsKeeper().AddWhitelistAdmin(ctx, addr0)
	s.Require().NoError(err)
	err = s.WhitelistsKeeper().AddToGlobalWhitelist(ctx, addr1)
	s.Require().NoError(err)
	err = s.WhitelistsKeeper().AddToGlobalWorkerWhitelist(ctx, addr0)
	s.Require().NoError(err)
	err = s.WhitelistsKeeper().AddToGlobalReputerWhitelist(ctx, addr1)
	s.Require().NoError(err)
	err = s.WhitelistsKeeper().AddToGlobalAdminWhitelist(ctx, addr0)
	s.Require().NoError(err)
	err = s.WhitelistsKeeper().AddToTopicCreatorWhitelist(ctx, addr1)
	s.Require().NoError(err)
	err = s.WhitelistsKeeper().AddToTopicWorkerWhitelist(ctx, 1, addr0)
	s.Require().NoError(err)
	err = s.WhitelistsKeeper().AddToTopicReputerWhitelist(ctx, 1, addr1)
	s.Require().NoError(err)
	err = s.WhitelistsKeeper().EnableTopicWorkerWhitelist(ctx, 1)
	s.Require().NoError(err)
	err = s.WhitelistsKeeper().EnableTopicReputerWhitelist(ctx, 1)
	s.Require().NoError(err)

	genesisState, err := s.EmissionsKeeper().ExportGenesis(ctx)
	s.Require().NoError(err)

	s.Require().NotEmpty(genesisState.WhitelistAdmins)
	s.Require().NotEmpty(genesisState.GlobalWhitelist)
	s.Require().NotEmpty(genesisState.GlobalWorkerWhitelist)
	s.Require().NotEmpty(genesisState.GlobalReputerWhitelist)
	s.Require().NotEmpty(genesisState.GlobalAdminWhitelist)
	s.Require().NotEmpty(genesisState.TopicCreatorWhitelist)
	s.Require().NotEmpty(genesisState.TopicWorkerWhitelist)
	s.Require().NotEmpty(genesisState.TopicReputerWhitelist)
	s.Require().NotEmpty(genesisState.TopicWorkerWhitelistEnabled)
	s.Require().NotEmpty(genesisState.TopicReputerWhitelistEnabled)

	fresh := s.newFreshGenesisSuite()

	err = fresh.EmissionsKeeper().InitGenesis(fresh.Ctx(), genesisState)
	s.Require().NoError(err)

	isAdmin, err := fresh.WhitelistsKeeper().IsWhitelistAdmin(fresh.Ctx(), addr0)
	s.Require().NoError(err)
	s.Require().True(isAdmin)

	isGlobal, err := fresh.WhitelistsKeeper().IsWhitelistedGlobalActor(fresh.Ctx(), addr1)
	s.Require().NoError(err)
	s.Require().True(isGlobal)
}

func (s *KeeperTestSuite) TestGenesisSubKeeperWeightsRoundTrip() {
	ctx := s.Ctx()
	topicId := uint64(1)
	addr0 := s.AddrsStr(0)
	addr1 := s.AddrsStr(1)

	err := s.WeightsKeeper().SetLatestRegretScale(ctx, topicId, alloraMath.MustNewDecFromString("3.14"))
	s.Require().NoError(err)
	err = s.WeightsKeeper().SetLatestInfererWeight(ctx, topicId, addr0, alloraMath.MustNewDecFromString("0.7"))
	s.Require().NoError(err)
	err = s.WeightsKeeper().SetLatestForecasterWeight(ctx, topicId, addr1, alloraMath.MustNewDecFromString("0.3"))
	s.Require().NoError(err)

	genesisState, err := s.EmissionsKeeper().ExportGenesis(ctx)
	s.Require().NoError(err)

	s.Require().Len(genesisState.LatestRegretStdNorm, 1)
	s.Require().Len(genesisState.LatestInfererWeights, 1)
	s.Require().Len(genesisState.LatestForecasterWeights, 1)

	fresh := s.newFreshGenesisSuite()

	err = fresh.EmissionsKeeper().InitGenesis(fresh.Ctx(), genesisState)
	s.Require().NoError(err)

	genesisState2, err := fresh.EmissionsKeeper().ExportGenesis(fresh.Ctx())
	s.Require().NoError(err)

	s.Require().Len(genesisState2.LatestRegretStdNorm, 1)
	s.Require().True(genesisState2.LatestRegretStdNorm[0].Dec.Equal(alloraMath.MustNewDecFromString("3.14")))
}

func (s *KeeperTestSuite) TestGenesisDefaultStateRoundTrip() {
	genesisState := types.NewGenesisState()

	fresh := s.newFreshGenesisSuite()

	err := fresh.EmissionsKeeper().InitGenesis(fresh.Ctx(), genesisState)
	s.Require().NoError(err)

	exported, err := fresh.EmissionsKeeper().ExportGenesis(fresh.Ctx())
	s.Require().NoError(err)
	s.Require().NotNil(exported)

	s.Require().Equal(genesisState.Params, exported.Params)
}

func (s *KeeperTestSuite) TestGenesisSubKeeperNoncesRoundTrip() {
	ctx := s.Ctx()
	topicId := uint64(1)

	err := s.NonceKeeper().AddWorkerNonce(ctx, topicId, &types.Nonce{BlockHeight: 50})
	s.Require().NoError(err)
	err = s.NonceKeeper().AddWorkerNonce(ctx, topicId, &types.Nonce{BlockHeight: 60})
	s.Require().NoError(err)
	err = s.NonceKeeper().AddReputerNonce(ctx, topicId, &types.Nonce{BlockHeight: 50})
	s.Require().NoError(err)

	genesisState, err := s.EmissionsKeeper().ExportGenesis(ctx)
	s.Require().NoError(err)

	s.Require().Len(genesisState.UnfulfilledWorkerNonces, 1)
	s.Require().Len(genesisState.UnfulfilledWorkerNonces[0].Nonces.Nonces, 2)
	s.Require().Len(genesisState.UnfulfilledReputerNonces, 1)

	fresh := s.newFreshGenesisSuite()

	err = fresh.EmissionsKeeper().InitGenesis(fresh.Ctx(), genesisState)
	s.Require().NoError(err)

	genesisState2, err := fresh.EmissionsKeeper().ExportGenesis(fresh.Ctx())
	s.Require().NoError(err)

	s.Require().Len(genesisState2.UnfulfilledWorkerNonces, 1)
	s.Require().Len(genesisState2.UnfulfilledWorkerNonces[0].Nonces.Nonces, 2)
	s.Require().Equal(
		genesisState.UnfulfilledWorkerNonces[0].Nonces.Nonces[0].BlockHeight,
		genesisState2.UnfulfilledWorkerNonces[0].Nonces.Nonces[0].BlockHeight,
	)
}

func (s *KeeperTestSuite) TestGenesisSubKeeperScoresRoundTrip() {
	ctx := s.Ctx()
	topicId := uint64(1)
	blockHeight := int64(100)
	addr0 := s.AddrsStr(0)
	addr1 := s.AddrsStr(1)

	err := s.ScoresKeeper().SetInfererScoreEma(ctx, topicId, addr0, types.Score{
		TopicId: topicId, BlockHeight: blockHeight, Address: addr0,
		Score: alloraMath.MustNewDecFromString("0.95"),
	})
	s.Require().NoError(err)
	err = s.ScoresKeeper().SetForecasterScoreEma(ctx, topicId, addr1, types.Score{
		TopicId: topicId, BlockHeight: blockHeight, Address: addr1,
		Score: alloraMath.MustNewDecFromString("0.85"),
	})
	s.Require().NoError(err)
	err = s.ScoresKeeper().SetListeningCoefficient(ctx, topicId, addr1, types.ListeningCoefficient{
		Coefficient: alloraMath.MustNewDecFromString("0.5"),
	})
	s.Require().NoError(err)
	err = s.ScoresKeeper().SetPreviousPercentageRewardToStakedReputers(ctx, alloraMath.MustNewDecFromString("0.28"))
	s.Require().NoError(err)
	err = s.ScoresKeeper().SetPreviousForecasterScoreRatio(ctx, topicId, alloraMath.MustNewDecFromString("0.65"))
	s.Require().NoError(err)

	genesisState, err := s.EmissionsKeeper().ExportGenesis(ctx)
	s.Require().NoError(err)

	s.Require().Len(genesisState.InfererScoreEmas, 1)
	s.Require().Len(genesisState.ForecasterScoreEmas, 1)
	s.Require().Len(genesisState.ReputerListeningCoefficient, 1)
	s.Require().True(genesisState.PreviousPercentageRewardToStakedReputers.Equal(
		alloraMath.MustNewDecFromString("0.28"),
	))

	fresh := s.newFreshGenesisSuite()

	err = fresh.EmissionsKeeper().InitGenesis(fresh.Ctx(), genesisState)
	s.Require().NoError(err)

	genesisState2, err := fresh.EmissionsKeeper().ExportGenesis(fresh.Ctx())
	s.Require().NoError(err)

	s.Require().Len(genesisState2.InfererScoreEmas, 1)
	s.Require().True(genesisState2.InfererScoreEmas[0].Score.Score.Equal(
		alloraMath.MustNewDecFromString("0.95"),
	))
	s.Require().True(genesisState2.PreviousPercentageRewardToStakedReputers.Equal(
		alloraMath.MustNewDecFromString("0.28"),
	))
	s.Require().Len(genesisState2.PreviousForecasterScoreRatio, 1)
	s.Require().True(genesisState2.PreviousForecasterScoreRatio[0].Dec.Equal(
		alloraMath.MustNewDecFromString("0.65"),
	))
}

func (s *KeeperTestSuite) TestGenesisSubKeeperTopicRoundTrip() {
	ctx := s.Ctx()
	topicId := uint64(1)
	blockHeight := int64(100)
	addr0 := s.AddrsStr(0)

	topic := types.Topic{
		Id:                       topicId,
		Creator:                  addr0,
		Metadata:                 "test-topic-topic-keeper",
		LossMethod:               "mse",
		EpochLastEnded:           0,
		EpochLength:              12,
		GroundTruthLag:           12,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:            false,
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		InitialRegret:            alloraMath.ZeroDec(),
		WorkerSubmissionWindow:   10,
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		CNorm:                    alloraMath.ZeroDec(),
		TopicType:                types.TopicType_TOPIC_TYPE_REGRESSION,
		OutputArity:              types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
		RequireUnity:             false,
		UnityTolerance:           alloraMath.Dec{},
		MaxLabelsPerSubmission:   types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:           nil,
		LabelDefaultValue:        alloraMath.ZeroDec(),
		LabelCaseSensitive:       false,
	}
	err := s.TopicKeeper().SetTopic(ctx, topicId, topic)
	s.Require().NoError(err)
	err = s.TopicKeeper().ActivateTopic(ctx, topicId)
	s.Require().NoError(err)
	err = s.TopicKeeper().SetTopicRewardNonce(ctx, topicId, blockHeight)
	s.Require().NoError(err)
	err = s.TopicKeeper().SetPreviousTopicWeight(ctx, topicId, alloraMath.MustNewDecFromString("1.5"))
	s.Require().NoError(err)
	err = s.TopicKeeper().SetLastDripBlock(ctx, topicId, blockHeight)
	s.Require().NoError(err)
	err = s.TopicKeeper().SetLastMedianInferences(ctx, topicId, alloraMath.MustNewDecFromString("2.5"))
	s.Require().NoError(err)
	err = s.TopicKeeper().SetMadInferences(ctx, topicId, alloraMath.MustNewDecFromString("0.5"))
	s.Require().NoError(err)
	err = s.TopicKeeper().SetTotalSumPreviousTopicWeights(ctx, alloraMath.MustNewDecFromString("50.0"))
	s.Require().NoError(err)

	genesisState, err := s.EmissionsKeeper().ExportGenesis(ctx)
	s.Require().NoError(err)

	s.Require().NotEmpty(genesisState.Topics)
	s.Require().NotEmpty(genesisState.ActiveTopics)
	s.Require().Len(genesisState.TopicRewardNonce, 1)
	s.Require().Len(genesisState.PreviousTopicWeight, 1)
	s.Require().Len(genesisState.LastDripBlock, 1)
	s.Require().Len(genesisState.LastMedianInferences, 1)
	s.Require().True(genesisState.TotalSumPreviousTopicWeights.Equal(alloraMath.MustNewDecFromString("50.0")))

	fresh := s.newFreshGenesisSuite()

	err = fresh.EmissionsKeeper().InitGenesis(fresh.Ctx(), genesisState)
	s.Require().NoError(err)

	genesisState2, err := fresh.EmissionsKeeper().ExportGenesis(fresh.Ctx())
	s.Require().NoError(err)

	s.Require().Equal(genesisState.NextTopicId, genesisState2.NextTopicId)
	s.Require().Len(genesisState2.Topics, len(genesisState.Topics))
	s.Require().Equal(genesisState.Topics[0].Topic.Metadata, genesisState2.Topics[0].Topic.Metadata)
	s.Require().True(genesisState2.TotalSumPreviousTopicWeights.Equal(alloraMath.MustNewDecFromString("50.0")))
	s.Require().True(genesisState2.LastMedianInferences[0].Dec.Equal(alloraMath.MustNewDecFromString("2.5")))
}

func (s *KeeperTestSuite) TestGenesisSubKeeperWorkerRoundTrip() {
	ctx := s.Ctx()
	topicId := uint64(1)
	addr0 := s.AddrsStr(0)
	addr1 := s.AddrsStr(1)

	err := s.WorkerKeeper().SetTopicWorker(ctx, topicId, addr0)
	s.Require().NoError(err)
	err = s.WorkerKeeper().AddActiveInferer(ctx, topicId, addr0)
	s.Require().NoError(err)
	err = s.WorkerKeeper().AddActiveForecaster(ctx, topicId, addr1)
	s.Require().NoError(err)

	genesisState, err := s.EmissionsKeeper().ExportGenesis(ctx)
	s.Require().NoError(err)

	s.Require().Len(genesisState.TopicWorkers, 1)
	s.Require().Equal(addr0, genesisState.TopicWorkers[0].ActorId)
	s.Require().Len(genesisState.ActiveInferers, 1)
	s.Require().Len(genesisState.ActiveForecasters, 1)

	fresh := s.newFreshGenesisSuite()

	err = fresh.EmissionsKeeper().InitGenesis(fresh.Ctx(), genesisState)
	s.Require().NoError(err)

	genesisState2, err := fresh.EmissionsKeeper().ExportGenesis(fresh.Ctx())
	s.Require().NoError(err)

	s.Require().Len(genesisState2.TopicWorkers, 1)
	s.Require().Equal(addr0, genesisState2.TopicWorkers[0].ActorId)
	s.Require().Len(genesisState2.ActiveInferers, 1)
	s.Require().Equal(addr0, genesisState2.ActiveInferers[0].ActorId)
	s.Require().Len(genesisState2.ActiveForecasters, 1)
	s.Require().Equal(addr1, genesisState2.ActiveForecasters[0].ActorId)
}

func (s *KeeperTestSuite) TestGenesisSubKeeperReputerLossRoundTrip() {
	ctx := s.Ctx()
	topicId := uint64(1)
	addr1 := s.AddrsStr(1)

	err := s.ReputerLossKeeper().SetTopicReputer(ctx, topicId, addr1)
	s.Require().NoError(err)
	err = s.ReputerLossKeeper().AddActiveReputer(ctx, topicId, addr1)
	s.Require().NoError(err)

	genesisState, err := s.EmissionsKeeper().ExportGenesis(ctx)
	s.Require().NoError(err)

	s.Require().Len(genesisState.TopicReputers, 1)
	s.Require().Equal(addr1, genesisState.TopicReputers[0].ActorId)
	s.Require().Len(genesisState.ActiveReputers, 1)

	fresh := s.newFreshGenesisSuite()

	err = fresh.EmissionsKeeper().InitGenesis(fresh.Ctx(), genesisState)
	s.Require().NoError(err)

	genesisState2, err := fresh.EmissionsKeeper().ExportGenesis(fresh.Ctx())
	s.Require().NoError(err)

	s.Require().Len(genesisState2.TopicReputers, 1)
	s.Require().Equal(addr1, genesisState2.TopicReputers[0].ActorId)
	s.Require().Len(genesisState2.ActiveReputers, 1)
	s.Require().Equal(addr1, genesisState2.ActiveReputers[0].ActorId)
}

func (s *KeeperTestSuite) TestInitGenesisRejectsMalformedNestedNilFields() {
	testCases := []struct {
		name      string
		mutate    func(*types.GenesisState)
		wantError string
	}{
		{
			name: "scores inferer block scores nil",
			mutate: func(gs *types.GenesisState) {
				gs.InfererScoresByBlock = []*types.TopicIdBlockHeightScores{{TopicId: 1, BlockHeight: 10, Scores: nil}}
			},
			wantError: "inferer scores by block cannot be nil",
		},
		{
			name: "staking delegated stake info nil",
			mutate: func(gs *types.GenesisState) {
				gs.DelegatedStakes = []*types.TopicIdDelegatorReputerDelegatorInfo{{TopicId: 1, Delegator: "a", Reputer: "b", DelegatorInfo: nil}}
			},
			wantError: "delegated stake info cannot be nil",
		},
		{
			name: "reputer offchain node nil",
			mutate: func(gs *types.GenesisState) {
				gs.Reputers = []*types.LibP2PKeyAndOffchainNode{{LibP2PKey: "peer", OffchainNode: nil}}
			},
			wantError: "reputer info cannot be nil",
		},
		{
			name: "worker inference nil",
			mutate: func(gs *types.GenesisState) {
				gs.Inferences = []*types.TopicIdActorIdInference{{TopicId: 1, ActorId: s.AddrsStr(0), Inference: nil}}
			},
			wantError: "inference cannot be nil",
		},
		{
			name: "all inferences nil wrapper",
			mutate: func(gs *types.GenesisState) {
				gs.AllInferences = []*types.TopicIdBlockHeightInferences{{TopicId: 1, BlockHeight: 10, Inferences: nil}}
			},
			wantError: "all inferences cannot be nil",
		},
		{
			name: "nonces worker nonces nil",
			mutate: func(gs *types.GenesisState) {
				gs.UnfulfilledWorkerNonces = []*types.TopicIdAndNonces{{TopicId: 1, Nonces: nil}}
			},
			wantError: "unfulfilled worker nonces cannot be nil",
		},
		{
			name: "nonces reputer nonces nil",
			mutate: func(gs *types.GenesisState) {
				gs.UnfulfilledReputerNonces = []*types.TopicIdAndReputerRequestNonces{{TopicId: 1, ReputerRequestNonces: nil}}
			},
			wantError: "unfulfilled reputer nonces cannot be nil",
		},
		{
			name: "topic inner topic nil",
			mutate: func(gs *types.GenesisState) {
				gs.Topics = []*types.TopicIdAndTopic{{TopicId: 1, Topic: nil}}
			},
			wantError: "topic cannot be nil",
		},
		{
			name: "topic worker commit nil",
			mutate: func(gs *types.GenesisState) {
				gs.TopicLastWorkerCommit = []*types.TopicIdTimestampedActorNonce{{TopicId: 1, TimestampedActorNonce: nil}}
			},
			wantError: "topic last worker commit cannot be nil",
		},
		{
			name: "topic active topics nil list",
			mutate: func(gs *types.GenesisState) {
				gs.BlockToActiveTopics = []*types.BlockHeightTopicIds{{BlockHeight: 10, TopicIds: nil}}
			},
			wantError: "block to active topics cannot be nil",
		},
		{
			name: "topic lowest active topic weight nil",
			mutate: func(gs *types.GenesisState) {
				gs.BlockToLowestActiveTopicWeight = []*types.BlockHeightTopicIdWeightPair{{BlockHeight: 10, TopicWeight: nil}}
			},
			wantError: "block to lowest active topic weight cannot be nil",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			fresh := s.newFreshGenesisSuite()
			genesisState := types.NewGenesisState()
			tc.mutate(genesisState)

			err := fresh.EmissionsKeeper().InitGenesis(fresh.Ctx(), genesisState)
			s.Require().Error(err)
			s.Require().Contains(err.Error(), tc.wantError)
		})
	}
}

// TestGenesisInferenceRoundTrip covers the round-trip of the live temporary
// inferences store through ExportGenesis -> InitGenesis on a fresh suite.
//
//nolint:exhaustruct
func (s *KeeperTestSuite) TestGenesisInferenceRoundTrip() {
	ctx := s.Ctx()
	topicId := uint64(1)
	nonceHeight := int64(42)
	inferer := s.AddrsStr(0)

	inference := types.Inference{
		TopicId:     topicId,
		BlockHeight: nonceHeight,
		Inferer:     inferer,
		Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString("0.5"), alloraMath.MustNewDecFromString("0.3")},
		ExtraData:   []byte("gdata"),
	}
	s.Require().NoError(
		s.WorkerKeeper().InsertInference(ctx, topicId, inference),
	)

	exported, err := s.EmissionsKeeper().ExportGenesis(ctx)
	s.Require().NoError(err)
	s.Require().Len(exported.Inferences, 1,
		"exported genesis should include the temporary inference")

	fresh := s.newFreshGenesisSuite()
	s.Require().NoError(fresh.EmissionsKeeper().InitGenesis(fresh.Ctx(), exported))

	got, err := fresh.WorkerKeeper().GetWorkerLatestInferenceByTopicId(fresh.Ctx(), topicId, inferer)
	s.Require().NoError(err)
	s.Require().Equal(topicId, got.TopicId)
	s.Require().Equal(inferer, got.Inferer)
	s.Require().Equal(nonceHeight, got.BlockHeight)
	s.Require().Len(got.Values, 2)
	s.Require().Equal("0.5", got.Values[0].String())
	s.Require().Equal("0.3", got.Values[1].String())
	s.Require().Equal([]byte("gdata"), got.ExtraData)
}

//nolint:exhaustruct
func (s *KeeperTestSuite) TestGenesisMultiInferenceRoundTripWithTopicLabelRegistry() {
	ctx := s.Ctx()
	topicId := s.CreateTopic()
	nonceHeight := types.BlockHeight(42)
	inferer := s.AddrsStr(0)

	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)
	topic.OutputArity = types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI
	topic.LabelDefaultValue = alloraMath.ZeroDec()
	s.Require().NoError(s.TopicKeeper().SetTopic(ctx, topicId, topic))
	registry := types.EpochLabelRegistry{
		TopicId: topicId,
		EpochId: uint64(nonceHeight),
		Labels: []*types.TopicLabel{
			{Id: 1, Name: "a"},
			{Id: 2, Name: "b"},
			{Id: 3, Name: "c"},
		},
	}
	s.Require().NoError(s.TopicKeeper().SetEpochLabelRegistry(ctx, registry))
	inference := types.Inference{
		TopicId:     topicId,
		BlockHeight: nonceHeight,
		Inferer:     inferer,
		Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString("0.5"), alloraMath.MustNewDecFromString("0.3")},
		ExtraData:   []byte("multi"),
	}
	s.Require().NoError(s.WorkerKeeper().InsertInference(ctx, topicId, inference))

	exported, err := s.EmissionsKeeper().ExportGenesis(ctx)
	s.Require().NoError(err)
	s.Require().Len(exported.Inferences, 1)
	s.Require().Len(exported.TopicLabelRegistries, 1)

	fresh := s.newFreshGenesisSuite()
	s.Require().NoError(fresh.EmissionsKeeper().InitGenesis(fresh.Ctx(), exported))

	gotRegistry, err := fresh.TopicKeeper().GetEpochLabelRegistry(fresh.Ctx(), topicId, nonceHeight)
	s.Require().NoError(err)
	s.Require().Equal(registry, gotRegistry)
	freshTopic, err := fresh.TopicKeeper().GetTopic(fresh.Ctx(), topicId)
	s.Require().NoError(err)
	gotInput, err := fresh.WorkerKeeper().GetWorkerLatestInputInferenceByTopicId(fresh.Ctx(), freshTopic, inferer)
	s.Require().NoError(err)
	s.Require().Len(gotInput.Values, 2)
	s.Require().Equal("a", gotInput.Values[0].Label)
	s.Require().Equal("0.5", gotInput.Values[0].Value.String())
	s.Require().Equal("b", gotInput.Values[1].Label)
	s.Require().Equal("0.3", gotInput.Values[1].Value.String())
}

//nolint:exhaustruct
func (s *KeeperTestSuite) TestGenesisMultiInferenceRequiresSufficientTopicLabelRegistry() {
	ctx := s.Ctx()
	topicId := s.CreateTopic()
	nonceHeight := types.BlockHeight(42)
	inferer := s.AddrsStr(0)

	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)
	topic.OutputArity = types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI
	topic.LabelDefaultValue = alloraMath.ZeroDec()
	s.Require().NoError(s.TopicKeeper().SetTopic(ctx, topicId, topic))
	inference := types.Inference{
		TopicId:     topicId,
		BlockHeight: nonceHeight,
		Inferer:     inferer,
		Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString("0.5")},
	}
	s.Require().NoError(s.WorkerKeeper().InsertInference(ctx, topicId, inference))

	exported, err := s.EmissionsKeeper().ExportGenesis(ctx)
	s.Require().NoError(err)
	exported.TopicLabelRegistries = nil

	fresh := s.newFreshGenesisSuite()
	err = fresh.EmissionsKeeper().InitGenesis(fresh.Ctx(), exported)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "live MULTI inference is incompatible with topic label registry")
}
