package rewards_test

import (
	"math/rand"
	"strconv"

	cosmosMath "cosmossdk.io/math"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/test/testutil"
	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *RewardsTestSuite) TestGetReputersScoresFromCsv() {
	epochGet := testutil.GetSimulatedValuesGetterForEpochs()
	epoch300Get := epochGet[300]
	epoch301Get := epochGet[301]
	block := int64(1003)

	newTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  s.addrs[0].String(),
		Metadata:                 "test",
		LossMethod:               "mse",
		EpochLength:              10800,
		GroundTruthLag:           10800,
		WorkerSubmissionWindow:   10,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:            true,
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		EnableWorkerWhitelist:    true,
		EnableReputerWhitelist:   true,
	}
	res, err := s.msgServer.CreateNewTopic(s.ctx, newTopicMsg)
	s.Require().NoError(err)
	topicId := res.TopicId

	reputer0 := s.addrsStr[13]
	reputer1 := s.addrsStr[14]
	reputer2 := s.addrsStr[15]
	reputer3 := s.addrsStr[16]
	reputer4 := s.addrsStr[17]
	reputerAddresses := []string{reputer0, reputer1, reputer2, reputer3, reputer4}

	inferer0 := s.addrsStr[5]
	inferer1 := s.addrsStr[6]
	inferer2 := s.addrsStr[7]
	inferer3 := s.addrsStr[8]
	inferer4 := s.addrsStr[9]
	infererAddresses := []string{inferer0, inferer1, inferer2, inferer3, inferer4}

	forecaster0 := s.addrsStr[10]
	forecaster1 := s.addrsStr[11]
	forecaster2 := s.addrsStr[12]
	forecasterAddresses := []string{forecaster0, forecaster1, forecaster2}

	reputers := []testutil.ReputerKey{
		{
			Address:    s.addrsStr[13],
			PrivateKey: s.privKeys[13],
			PubKeyHex:  s.pubKeyHexStr[13],
		},
		{
			Address:    s.addrsStr[14],
			PrivateKey: s.privKeys[14],
			PubKeyHex:  s.pubKeyHexStr[14],
		},
		{
			Address:    s.addrsStr[15],
			PrivateKey: s.privKeys[15],
			PubKeyHex:  s.pubKeyHexStr[15],
		},
		{
			Address:    s.addrsStr[16],
			PrivateKey: s.privKeys[16],
			PubKeyHex:  s.pubKeyHexStr[16],
		},
		{
			Address:    s.addrsStr[17],
			PrivateKey: s.privKeys[17],
			PubKeyHex:  s.pubKeyHexStr[17],
		},
	}

	cosmosOneE18 := inferencesynthesis.CosmosIntOneE18()
	cosmosOneE18Dec, err := alloraMath.NewDecFromSdkInt(cosmosOneE18)
	s.Require().NoError(err)

	reputer0Stake, err := epoch301Get("reputer_stake_0").Mul(cosmosOneE18Dec)
	s.Require().NoError(err)
	reputer0StakeInt, err := reputer0Stake.BigInt()
	s.Require().NoError(err)
	reputer1Stake, err := epoch301Get("reputer_stake_1").Mul(cosmosOneE18Dec)
	s.Require().NoError(err)
	reputer1StakeInt, err := reputer1Stake.BigInt()
	s.Require().NoError(err)
	reputer2Stake, err := epoch301Get("reputer_stake_2").Mul(cosmosOneE18Dec)
	s.Require().NoError(err)
	reputer2StakeInt, err := reputer2Stake.BigInt()
	s.Require().NoError(err)
	reputer3Stake, err := epoch301Get("reputer_stake_3").Mul(cosmosOneE18Dec)
	s.Require().NoError(err)
	reputer3StakeInt, err := reputer3Stake.BigInt()
	s.Require().NoError(err)
	reputer4Stake, err := epoch301Get("reputer_stake_4").Mul(cosmosOneE18Dec)
	s.Require().NoError(err)
	reputer4StakeInt, err := reputer4Stake.BigInt()
	s.Require().NoError(err)

	var stakes = []cosmosMath.Int{
		cosmosMath.NewIntFromBigInt(reputer0StakeInt),
		cosmosMath.NewIntFromBigInt(reputer1StakeInt),
		cosmosMath.NewIntFromBigInt(reputer2StakeInt),
		cosmosMath.NewIntFromBigInt(reputer3StakeInt),
		cosmosMath.NewIntFromBigInt(reputer4StakeInt),
	}
	var coefficients = []alloraMath.Dec{
		epoch300Get("reputer_listening_coefficient_0"),
		epoch300Get("reputer_listening_coefficient_1"),
		epoch300Get("reputer_listening_coefficient_2"),
		epoch300Get("reputer_listening_coefficient_3"),
		epoch300Get("reputer_listening_coefficient_4"),
	}
	for i, addr := range reputerAddresses {
		addrBech, err := sdk.AccAddressFromBech32(addr)
		s.Require().NoError(err)

		s.MintTokensToAddress(addrBech, stakes[i])

		err = s.emissionsKeeper.AddReputerStake(s.ctx, topicId, addr, stakes[i])
		s.Require().NoError(err)

		err = s.emissionsKeeper.SetListeningCoefficient(
			s.ctx,
			topicId,
			addr,
			types.ListeningCoefficient{Coefficient: coefficients[i]},
		)
		s.Require().NoError(err)
	}

	reportedLosses, err := testutil.GetReputersDataFromCsv(
		topicId,
		block,
		infererAddresses,
		forecasterAddresses,
		reputers,
		epoch301Get,
	)
	s.Require().NoError(err)

	// Generate new reputer scores
	scores, err := rewards.GenerateReputerScores(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		block,
		reportedLosses,
	)
	s.Require().NoError(err)

	expectedScores := []alloraMath.Dec{
		epoch301Get("reputer_score_0"),
		epoch301Get("reputer_score_1"),
		epoch301Get("reputer_score_2"),
		epoch301Get("reputer_score_3"),
		epoch301Get("reputer_score_4"),
	}
	for i, reputerScore := range scores {
		testutil.InEpsilon5(s.T(), reputerScore.Score, expectedScores[i].String())
	}
}

func (s *RewardsTestSuite) TestGetInferenceScores() {
	newTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  s.addrs[0].String(),
		Metadata:                 "test",
		LossMethod:               "mse",
		EpochLength:              10800,
		GroundTruthLag:           10800,
		WorkerSubmissionWindow:   10,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:            true,
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		EnableWorkerWhitelist:    true,
		EnableReputerWhitelist:   true,
	}
	res, err := s.msgServer.CreateNewTopic(s.ctx, newTopicMsg)
	s.Require().NoError(err)
	topicId := res.TopicId
	block := int64(1003)

	// Generate workers data for tests
	reportedLosses, err := mockNetworkLosses(s, topicId, block)
	s.Require().NoError(err)

	// Get inference scores
	scores, err := rewards.GenerateInferenceScores(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		block,
		reportedLosses,
	)
	s.Require().NoError(err)

	expectedScores := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("-0.00021125601"),
		alloraMath.MustNewDecFromString("-0.000461256018"),
		alloraMath.MustNewDecFromString("0.0001187439"),
		alloraMath.MustNewDecFromString("0.0014287439"),
		alloraMath.MustNewDecFromString("0.00337874398"),
	}
	for i, reputerScore := range scores {
		scoreDelta, err := reputerScore.Score.Sub(expectedScores[i])
		s.Require().NoError(err)
		absScoreDelta, err := scoreDelta.Abs()
		s.Require().NoError(err)
		deltaTightness := absScoreDelta.
			Cmp(alloraMath.MustNewDecFromString("0.00001"))
		if !(deltaTightness == alloraMath.LessThan || deltaTightness == alloraMath.EqualTo) {
			s.Fail("Expected reward is not equal to the actual reward")
		}
	}
}

func (s *RewardsTestSuite) TestGenerateInferenceScores_AllNilOneOutAndOneIn() {
	newTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  s.addrs[0].String(),
		Metadata:                 "test",
		LossMethod:               "mse",
		EpochLength:              10800,
		GroundTruthLag:           10800,
		WorkerSubmissionWindow:   10,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:            true,
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		EnableWorkerWhitelist:    true,
		EnableReputerWhitelist:   true,
	}
	res, err := s.msgServer.CreateNewTopic(s.ctx, newTopicMsg)
	s.Require().NoError(err)
	topicId := res.TopicId
	block := int64(2001)

	// Generate normal network losses, then set all one-out/one-in fields to nil
	reportedLosses, err := mockNetworkLosses(s, topicId, block)
	s.Require().NoError(err)

	reportedLosses.OneOutInfererValues = nil
	reportedLosses.OneOutForecasterValues = nil
	reportedLosses.OneInForecasterValues = nil

	scores, err := rewards.GenerateInferenceScores(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		block,
		reportedLosses,
	)
	// Should not panic, should return empty slice and no error
	s.Require().NoError(err)
	s.Require().Len(scores, 0, "Expected no scores when all one-out/one-in values are nil")
}

func (s *RewardsTestSuite) TestGetInferenceScoresFromCsv() {
	epochGet := testutil.GetSimulatedValuesGetterForEpochs()
	epoch3Get := epochGet[300]
	newTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  s.addrs[0].String(),
		Metadata:                 "test",
		LossMethod:               "mse",
		EpochLength:              10800,
		GroundTruthLag:           10800,
		WorkerSubmissionWindow:   10,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:            true,
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		EnableWorkerWhitelist:    true,
		EnableReputerWhitelist:   true,
	}
	res, err := s.msgServer.CreateNewTopic(s.ctx, newTopicMsg)
	s.Require().NoError(err)
	topicId := res.TopicId
	block := int64(1003)

	inferer0 := s.addrs[5].String()
	inferer1 := s.addrs[6].String()
	inferer2 := s.addrs[7].String()
	inferer3 := s.addrs[8].String()
	inferer4 := s.addrs[9].String()
	infererAddresses := []string{inferer0, inferer1, inferer2, inferer3, inferer4}

	forecaster0 := s.addrs[10].String()
	forecaster1 := s.addrs[11].String()
	forecaster2 := s.addrs[12].String()
	forecasterAddresses := []string{forecaster0, forecaster1, forecaster2}

	reputer0 := s.addrs[13].String()

	reportedLosses, err := testutil.GetNetworkLossFromCsv(
		topicId,
		block,
		infererAddresses,
		forecasterAddresses,
		reputer0,
		epoch3Get,
	)
	s.Require().NoError(err)

	scores, err := rewards.GenerateInferenceScores(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		block,
		reportedLosses,
	)
	s.Require().NoError(err)

	expectedScores := []alloraMath.Dec{
		epoch3Get("inferer_score_0"),
		epoch3Get("inferer_score_1"),
		epoch3Get("inferer_score_2"),
		epoch3Get("inferer_score_3"),
		epoch3Get("inferer_score_4"),
	}
	for i, infererScore := range scores {
		testutil.InEpsilon5(s.T(), infererScore.Score, expectedScores[i].String())
	}
}

// In this test we run two trials of generating inference scores, the first with lower one out losses
// and the second with higher one out losses.
// We then compare the resulting scores and check that the higher one out losses result in higher scores.
func (s *RewardsTestSuite) TestHigherOneOutLossesHigherInferenceScore() {
	newTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  s.addrs[0].String(),
		Metadata:                 "test",
		LossMethod:               "mse",
		EpochLength:              10800,
		GroundTruthLag:           10800,
		WorkerSubmissionWindow:   10,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:            true,
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		EnableWorkerWhitelist:    true,
		EnableReputerWhitelist:   true,
	}
	res, err := s.msgServer.CreateNewTopic(s.ctx, newTopicMsg)
	s.Require().NoError(err)
	topicId := res.TopicId
	block0 := int64(1003)
	require := s.Require()

	networkLosses0, err := mockSimpleNetworkLosses(s, topicId, block0, "0.1")
	require.NoError(err)

	scores0, err := rewards.GenerateInferenceScores(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		block0,
		networkLosses0,
	)
	require.NoError(err)

	block1 := block0 + 1

	networkLosses1, err := mockSimpleNetworkLosses(s, topicId, block1, "0.2")
	require.NoError(err)

	scores1, err := rewards.GenerateInferenceScores(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		block1,
		networkLosses1,
	)
	require.NoError(err)

	require.True(scores0[0].Score.Lt(scores1[0].Score))
}

func (s *RewardsTestSuite) TestGetForecastScores() {
	block := int64(1003)
	newTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  s.addrs[0].String(),
		Metadata:                 "test",
		LossMethod:               "mse",
		EpochLength:              10800,
		GroundTruthLag:           10800,
		WorkerSubmissionWindow:   10,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:            true,
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		EnableWorkerWhitelist:    true,
		EnableReputerWhitelist:   true,
	}
	res, err := s.msgServer.CreateNewTopic(s.ctx, newTopicMsg)
	s.Require().NoError(err)
	topicId := res.TopicId

	// Generate workers data for tests
	reportedLosses, err := mockNetworkLosses(s, topicId, block)
	s.Require().NoError(err)

	scores, err := rewards.GenerateForecastScores(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		block,
		reportedLosses,
	)
	s.Require().NoError(err)

	expectedScores := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("0.000389744278"),
		alloraMath.MustNewDecFromString("-0.00017400572"),
		alloraMath.MustNewDecFromString("0.0027597442"),
		alloraMath.MustNewDecFromString("-0.001075880"),
		alloraMath.MustNewDecFromString("-0.000099005721"),
	}
	for i, reputerScore := range scores {
		delta, err := reputerScore.Score.Sub(expectedScores[i])
		s.Require().NoError(err)
		absScoreDelta, err := delta.Abs()
		s.Require().NoError(err)
		deltaTightness := absScoreDelta.Cmp(alloraMath.MustNewDecFromString("0.00001"))
		if !(deltaTightness == alloraMath.LessThan || deltaTightness == alloraMath.EqualTo) {
			s.Fail("Expected reward is not equal to the actual reward")
		}
	}
}

func (s *RewardsTestSuite) TestGetForecasterScoresFromCsv() {
	epochGet := testutil.GetSimulatedValuesGetterForEpochs()
	epoch3Get := epochGet[300]
	newTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  s.addrs[0].String(),
		Metadata:                 "test",
		LossMethod:               "mse",
		EpochLength:              10800,
		GroundTruthLag:           10800,
		WorkerSubmissionWindow:   10,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:            true,
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		EnableWorkerWhitelist:    true,
		EnableReputerWhitelist:   true,
	}
	res, err := s.msgServer.CreateNewTopic(s.ctx, newTopicMsg)
	s.Require().NoError(err)
	topicId := res.TopicId
	block := int64(1003)

	inferer0 := s.addrs[5].String()
	inferer1 := s.addrs[6].String()
	inferer2 := s.addrs[7].String()
	inferer3 := s.addrs[8].String()
	inferer4 := s.addrs[9].String()
	infererAddresses := []string{inferer0, inferer1, inferer2, inferer3, inferer4}

	forecaster0 := s.addrs[10].String()
	forecaster1 := s.addrs[11].String()
	forecaster2 := s.addrs[12].String()
	forecasterAddresses := []string{forecaster0, forecaster1, forecaster2}

	reputer0 := s.addrsStr[13]

	reportedLosses, err := testutil.GetNetworkLossFromCsv(
		topicId,
		block,
		infererAddresses,
		forecasterAddresses,
		reputer0,
		epoch3Get,
	)
	s.Require().NoError(err)

	scores, err := rewards.GenerateForecastScores(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		block,
		reportedLosses,
	)
	s.Require().NoError(err)

	expectedScores := []alloraMath.Dec{
		epoch3Get("forecaster_score_0"),
		epoch3Get("forecaster_score_1"),
		epoch3Get("forecaster_score_2"),
	}
	for i, forecasterScore := range scores {
		testutil.InEpsilon5(s.T(), forecasterScore.Score, expectedScores[i].String())
	}
}

// In this test we run two trials of generating forecast scores, the first with lower one out losses
// and the second with higher one out losses.
// We then compare the resulting forecaster scores and check that the higher one out losses result
// in higher scores.
func (s *RewardsTestSuite) TestHigherOneOutLossesHigherForecastScore() {
	newTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  s.addrs[0].String(),
		Metadata:                 "test",
		LossMethod:               "mse",
		EpochLength:              10800,
		GroundTruthLag:           10800,
		WorkerSubmissionWindow:   10,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:            true,
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		EnableWorkerWhitelist:    true,
		EnableReputerWhitelist:   true,
	}
	res, err := s.msgServer.CreateNewTopic(s.ctx, newTopicMsg)
	s.Require().NoError(err)
	topicId := res.TopicId
	block0 := int64(1003)
	require := s.Require()

	networkLosses0, err := mockSimpleNetworkLosses(s, topicId, block0, "0.1")
	require.NoError(err)

	scores0, err := rewards.GenerateForecastScores(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		block0,
		networkLosses0,
	)
	require.NoError(err)

	block1 := block0 + 1

	networkLosses1, err := mockSimpleNetworkLosses(s, topicId, block1, "0.2")
	require.NoError(err)

	// Get inference scores
	scores1, err := rewards.GenerateForecastScores(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		block1,
		networkLosses1,
	)
	require.NoError(err)

	require.True(scores0[0].Score.Lt(scores1[0].Score))
}

func (s *RewardsTestSuite) TestEnsureAllWorkersPresent() {
	// Setup
	allWorkers := map[string]struct{}{
		s.addrsStr[1]: {},
		s.addrsStr[2]: {},
		"worker3":     {},
		"worker4":     {},
	}

	values := []*types.WorkerAttributedValue{
		{Worker: s.addrsStr[1], Value: alloraMath.NewDecFromInt64(100)},
		{Worker: "worker3", Value: alloraMath.NewDecFromInt64(300)},
	}

	expectedValues := map[string]string{
		s.addrsStr[1]: "100",
		s.addrsStr[2]: "NaN",
		"worker3":     "300",
		"worker4":     "NaN",
	}

	// Act
	updatedValues := rewards.EnsureAllWorkersPresent(values, allWorkers)

	// Assert
	if len(updatedValues) != len(allWorkers) {
		s.Fail("Incorrect number of workers returned")
	}

	for _, val := range updatedValues {
		expectedVal, ok := expectedValues[val.Worker]
		if !ok {
			s.Fail("Unexpected worker found:", val.Worker)
			continue
		}
		if expectedVal == "NaN" {
			if !val.Value.IsNaN() {
				s.Failf("expected NaN but did not get it for worker %s", val.Worker)
			}
		} else if val.Value.String() != expectedVal {
			s.Failf("Value mismatch for worker %s: got %s, want %s", val.Worker, val.Value.String(), expectedVal)
		}
	}
}

func (s *RewardsTestSuite) TestEnsureAllWorkersPresentWithheld() {
	// Setup
	allWorkers := map[string]struct{}{
		s.addrsStr[1]: {},
		s.addrsStr[2]: {},
		"worker3":     {},
		"worker4":     {},
	}

	values := []*types.WithheldWorkerAttributedValue{
		{Worker: s.addrsStr[1], Value: alloraMath.NewDecFromInt64(100)},
		{Worker: "worker3", Value: alloraMath.NewDecFromInt64(300)},
	}

	expectedValues := map[string]string{
		s.addrsStr[1]: "100",
		s.addrsStr[2]: "NaN",
		"worker3":     "300",
		"worker4":     "NaN",
	}

	// Act
	updatedValues := rewards.EnsureAllWorkersPresentWithheld(values, allWorkers)

	// Assert
	if len(updatedValues) != len(allWorkers) {
		s.Fail("Incorrect number of workers returned")
	}

	for _, val := range updatedValues {
		expectedVal, ok := expectedValues[val.Worker]
		if !ok {
			s.Fail("Unexpected worker found:", val.Worker)
			continue
		}
		if expectedVal == "NaN" {
			if !val.Value.IsNaN() {
				s.Failf("expected NaN but did not get it for worker %s", val.Worker)
			}
		} else if val.Value.String() != expectedVal {
			s.Failf("Value mismatch for worker %s: got %s, want %s", val.Worker, val.Value.String(), expectedVal)
		}
	}
}

func (s *RewardsTestSuite) TestEnsureWorkerPresenceConsistency() {
	// Create sample input where reputer1 has fewer workers
	reportedLosses := types.ReputerValueBundles{
		ReputerValueBundles: []*types.ReputerValueBundle{
			{
				Pubkey: "allo12vgd3fhvghc94e6kmnv02yw2jar3a5zu3jgfh2",
				ValueBundle: &types.ValueBundle{
					TopicId:   1,
					ExtraData: nil,
					ReputerRequestNonce: &types.ReputerRequestNonce{
						ReputerNonce: &types.Nonce{BlockHeight: 100},
					},
					Reputer:       "allo12vgd3fhvghc94e6kmnv02yw2jar3a5zu3jgfh2",
					CombinedValue: alloraMath.NewDecFromInt64(100),
					NaiveValue:    alloraMath.NewDecFromInt64(100),
					InfererValues: []*types.WorkerAttributedValue{
						{Worker: s.addrsStr[1], Value: alloraMath.NewDecFromInt64(100)},
						{Worker: s.addrsStr[2], Value: alloraMath.NewDecFromInt64(200)},
					},
					ForecasterValues: []*types.WorkerAttributedValue{
						{Worker: s.addrsStr[1], Value: alloraMath.NewDecFromInt64(300)},
					},
					OneOutInfererValues: []*types.WithheldWorkerAttributedValue{
						{Worker: s.addrsStr[1], Value: alloraMath.NewDecFromInt64(100)},
						{Worker: s.addrsStr[2], Value: alloraMath.NewDecFromInt64(200)},
					},
					OneOutForecasterValues: []*types.WithheldWorkerAttributedValue{
						{Worker: s.addrsStr[1], Value: alloraMath.NewDecFromInt64(300)},
					},
					OneInForecasterValues: []*types.WorkerAttributedValue{
						{Worker: s.addrsStr[2], Value: alloraMath.NewDecFromInt64(400)},
					},
					OneOutInfererForecasterValues: []*types.OneOutInfererForecasterValues{
						{
							Forecaster: "allo13kenskkx7e0v253m3kcgwfc67cmx00fgwpgj6h",
							OneOutInfererValues: []*types.WithheldWorkerAttributedValue{
								{Worker: s.addrsStr[1], Value: alloraMath.NewDecFromInt64(500)},
							},
						},
					},
				},
			},
			{
				Pubkey: "reputer2",
				ValueBundle: &types.ValueBundle{
					TopicId:   1,
					ExtraData: nil,
					ReputerRequestNonce: &types.ReputerRequestNonce{
						ReputerNonce: &types.Nonce{BlockHeight: 100},
					},
					Reputer:       "reputer2",
					CombinedValue: alloraMath.NewDecFromInt64(100),
					NaiveValue:    alloraMath.NewDecFromInt64(100),
					InfererValues: []*types.WorkerAttributedValue{
						{Worker: "worker5", Value: alloraMath.NewDecFromInt64(100)},
					},
					ForecasterValues: nil,
					OneOutInfererValues: []*types.WithheldWorkerAttributedValue{
						{Worker: s.addrsStr[1], Value: alloraMath.NewDecFromInt64(100)},
						{Worker: s.addrsStr[2], Value: alloraMath.NewDecFromInt64(200)},
						{Worker: "worker3", Value: alloraMath.NewDecFromInt64(300)},
						{Worker: "worker4", Value: alloraMath.NewDecFromInt64(400)},
					},
					OneOutForecasterValues: []*types.WithheldWorkerAttributedValue{
						{Worker: s.addrsStr[1], Value: alloraMath.NewDecFromInt64(500)},
						{Worker: "worker3", Value: alloraMath.NewDecFromInt64(600)},
					},
					OneInForecasterValues: []*types.WorkerAttributedValue{
						{Worker: s.addrsStr[2], Value: alloraMath.NewDecFromInt64(700)},
						{Worker: "worker4", Value: alloraMath.NewDecFromInt64(800)},
					},
					OneOutInfererForecasterValues: []*types.OneOutInfererForecasterValues{
						{
							Forecaster: "allo13kenskkx7e0v253m3kcgwfc67cmx00fgwpgj6h",
							OneOutInfererValues: []*types.WithheldWorkerAttributedValue{
								{Worker: "worker6", Value: alloraMath.NewDecFromInt64(1000)},
							},
						},
						{
							Forecaster: "forecaster2",
							OneOutInfererValues: []*types.WithheldWorkerAttributedValue{
								{Worker: "worker3", Value: alloraMath.NewDecFromInt64(900)},
								{Worker: "worker4", Value: alloraMath.NewDecFromInt64(1000)},
							},
						},
					},
				},
			},
		},
	}

	// Flatten losses and compare lengths before processing
	reputer1Losses := rewards.ExtractValues(reportedLosses.ReputerValueBundles[0].ValueBundle)
	reputer2Losses := rewards.ExtractValues(reportedLosses.ReputerValueBundles[1].ValueBundle)
	s.Require().NotEqual(len(reputer1Losses), len(reputer2Losses), "Initial lengths should not be equal")

	// Run the function under test
	updatedLosses := rewards.EnsureWorkerPresence(reportedLosses)

	// Flatten losses and compare lengths after processing
	reputer1Losses = rewards.ExtractValues(updatedLosses.ReputerValueBundles[0].ValueBundle)
	reputer2Losses = rewards.ExtractValues(updatedLosses.ReputerValueBundles[1].ValueBundle)

	// Ensure the lengths are equal
	s.Require().Equal(len(reputer1Losses), len(reputer2Losses), "Lengths should be equal after processing")
}

func prepareMockLosses(reputersCount int, workersCount int) (
	reputersLosses []alloraMath.Dec,
	reputersInfererLosses [][]alloraMath.Dec,
	reputersForecasterLosses [][]alloraMath.Dec,
	reputersNaiveLosses []alloraMath.Dec,
	reputersInfererOneOutLosses [][]alloraMath.Dec,
	reputersForecasterOneOutLosses [][]alloraMath.Dec,
	reputersOneInNaiveLosses [][]alloraMath.Dec,
) {
	rnd := rand.New(rand.NewSource(20))
	for i := 0; i < reputersCount; i++ {
		reputersLosses = append(reputersLosses, alloraMath.MustNewDecFromString(strconv.FormatFloat(float64(rnd.Intn(1000)+1), 'f', -1, 64)))
		reputersNaiveLosses = append(reputersNaiveLosses, alloraMath.MustNewDecFromString(strconv.FormatFloat(float64(rnd.Intn(1000)+1), 'f', -1, 64)))
		var infererLosses = make([]alloraMath.Dec, 0)
		var forecasterLosses = make([]alloraMath.Dec, 0)
		var infererOneOutLosses = make([]alloraMath.Dec, 0)
		var forecasterOneOutLosses = make([]alloraMath.Dec, 0)
		var oneInNaiveLosses = make([]alloraMath.Dec, 0)
		for j := 0; j < workersCount; j++ {
			infererLosses = append(infererLosses, alloraMath.MustNewDecFromString(strconv.FormatFloat(float64(rnd.Intn(1000)+1), 'f', -1, 64)))
			forecasterLosses = append(forecasterLosses, alloraMath.MustNewDecFromString(strconv.FormatFloat(float64(rnd.Intn(1000)+1), 'f', -1, 64)))
			infererOneOutLosses = append(infererOneOutLosses, alloraMath.MustNewDecFromString(strconv.FormatFloat(float64(rnd.Intn(1000)+1), 'f', -1, 64)))
			forecasterOneOutLosses = append(forecasterOneOutLosses, alloraMath.MustNewDecFromString(strconv.FormatFloat(float64(rnd.Intn(1000)+1), 'f', -1, 64)))
			oneInNaiveLosses = append(oneInNaiveLosses, alloraMath.MustNewDecFromString(strconv.FormatFloat(float64(rnd.Intn(1000)+1), 'f', -1, 64)))
		}
		reputersInfererLosses = append(reputersInfererLosses, infererLosses)
		reputersForecasterLosses = append(reputersForecasterLosses, forecasterLosses)
		reputersInfererOneOutLosses = append(reputersInfererOneOutLosses, infererOneOutLosses)
		reputersForecasterOneOutLosses = append(reputersForecasterOneOutLosses, forecasterOneOutLosses)
		reputersOneInNaiveLosses = append(reputersOneInNaiveLosses, oneInNaiveLosses)
	}
	return reputersLosses,
		reputersInfererLosses,
		reputersForecasterLosses,
		reputersNaiveLosses,
		reputersInfererOneOutLosses,
		reputersForecasterOneOutLosses,
		reputersOneInNaiveLosses
}

func generateLossBundles(s *RewardsTestSuite, blockHeight int64, topicId uint64, reputerIndexes []int, workerIndexes []int) types.InputReputerValueBundles {
	if len(workerIndexes) != 5 {
		panic("workerIndexes length must be 5")
	}
	workers := []sdk.AccAddress{
		s.addrs[workerIndexes[0]],
		s.addrs[workerIndexes[1]],
		s.addrs[workerIndexes[2]],
		s.addrs[workerIndexes[3]],
		s.addrs[workerIndexes[4]],
	}
	reputersLosses := []alloraMath.BoundedExp40Dec{
		alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01127")),
		alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01791")),
		alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01404")),
		alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.02318")),
		alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01251")),
	}
	reputersInfererLosses := [][]alloraMath.BoundedExp40Dec{
		{
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.0112")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.00231")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.02274")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01299")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.02515")),
		},
		{
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01635")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.00179")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.03396")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.0153")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01988")),
		},
		{
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01345")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.00209")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.03249")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01688")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.02126")),
		},
		{
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01675")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.00318")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.02623")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.02734")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.03526")),
		},
		{
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.02093")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.00213")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.02462")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.0203")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.03115")),
		},
	}
	reputersForecasterLosses := [][]alloraMath.BoundedExp40Dec{
		{
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.0185")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01018")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.02105")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01041")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.0183")),
		},
		{
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.00962")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01191")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01616")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01417")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01216")),
		},
		{
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01338")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.0116")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01605")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.0133")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01407")),
		},
		{
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.02733")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01697")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01619")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01925")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.02018")),
		},
		{
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01545")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01785")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01662")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01156")),
		},
	}
	reputersNaiveLosses := []alloraMath.BoundedExp40Dec{
		alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.0116")),
		alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01428")),
		alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01441")),
		alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01594")),
		alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01705")),
	}
	reputersInfererOneOutLosses := [][]alloraMath.BoundedExp40Dec{
		{
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.0148")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01046")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01192")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01381")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01687")),
		},
		{
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01043")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01308")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01455")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01607")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01205")),
		},
		{
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01339")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01053")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01424")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01428")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01446")),
		},
		{
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01674")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.02944")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01796")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.02187")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01895")),
		},
		{
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01049")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.02068")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01573")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01487")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.02639")),
		},
	}
	reputersForecasterOneOutLosses := [][]alloraMath.BoundedExp40Dec{
		{
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01136")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01185")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01568")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.00949")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01339")),
		},
		{
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01357")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01108")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01633")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01208")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01278")),
		},
		{
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01805")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01229")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01586")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01234")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01513")),
		},
		{
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01637")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01594")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01608")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.02203")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01486")),
		},
		{
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01981")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.02123")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.02134")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.0217")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01177")),
		},
	}
	reputersOneInNaiveLosses := [][]alloraMath.BoundedExp40Dec{
		{
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01588")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01012")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01467")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.0128")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01234")),
		},
		{
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01239")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01023")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01712")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.0116")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01639")),
		},
		{
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01419")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01497")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01629")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01514")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01133")),
		},
		{
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01936")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01518")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.018")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.02212")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.02259")),
		},
		{
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01602")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01194")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.0153")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.0199")),
			alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01673")),
		},
	}

	var reputerValueBundles types.InputReputerValueBundles
	for i, reputerIndex := range reputerIndexes {
		valueBundle := &types.InputValueBundle{
			TopicId: topicId,
			ReputerRequestNonce: &types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{
					BlockHeight: blockHeight,
				},
			},
			Reputer:                       s.addrsStr[reputerIndex],
			ExtraData:                     nil,
			CombinedValue:                 reputersLosses[i],
			NaiveValue:                    reputersNaiveLosses[i],
			InfererValues:                 make([]*types.InputWorkerAttributedValue, len(workers)),
			ForecasterValues:              make([]*types.InputWorkerAttributedValue, len(workers)),
			OneOutInfererValues:           make([]*types.InputWithheldWorkerAttributedValue, len(workers)),
			OneOutForecasterValues:        make([]*types.InputWithheldWorkerAttributedValue, len(workers)),
			OneInForecasterValues:         make([]*types.InputWorkerAttributedValue, len(workers)),
			OneOutInfererForecasterValues: make([]*types.InputOneOutInfererForecasterValues, len(workers)),
		}

		for j, worker := range workers {
			valueBundle.InfererValues[j] = &types.InputWorkerAttributedValue{Worker: worker.String(), Value: reputersInfererLosses[i][j]}
			valueBundle.ForecasterValues[j] = &types.InputWorkerAttributedValue{Worker: worker.String(), Value: reputersForecasterLosses[i][j]}
			valueBundle.OneOutInfererValues[j] = &types.InputWithheldWorkerAttributedValue{Worker: worker.String(), Value: reputersInfererOneOutLosses[i][j]}
			valueBundle.OneOutForecasterValues[j] = &types.InputWithheldWorkerAttributedValue{Worker: worker.String(), Value: reputersForecasterOneOutLosses[i][j]}
			valueBundle.OneInForecasterValues[j] = &types.InputWorkerAttributedValue{Worker: worker.String(), Value: reputersOneInNaiveLosses[i][j]}
		}
		for j, worker := range workers {
			valueBundle.OneOutInfererForecasterValues[j] = &types.InputOneOutInfererForecasterValues{Forecaster: worker.String(), OneOutInfererValues: valueBundle.OneOutInfererValues}
		}

		sig, err := signInputValueBundle(valueBundle, s.privKeys[reputerIndex])
		s.Require().NoError(err)

		bundle := &types.InputReputerValueBundle{
			Pubkey:      s.pubKeyHexStr[reputerIndex],
			Signature:   sig,
			ValueBundle: valueBundle,
		}
		reputerValueBundles.ReputerValueBundles = append(reputerValueBundles.ReputerValueBundles, bundle)
	}

	return reputerValueBundles
}

func generateHugeLossBundles(
	s *RewardsTestSuite,
	blockHeight int64,
	topicId uint64,
	reputerIndexes,
	workerIndexes,
	forecasterIndexes []int,
) types.InputReputerValueBundles {
	var (
		reputersLosses,
		reputersInfererLosses,
		reputersForecasterLosses,
		reputersNaiveLosses,
		reputersInfererOneOutLosses,
		reputersForecasterOneOutLosses,
		reputersOneInNaiveLosses = prepareMockLosses(len(reputerIndexes), len(workerIndexes))
	)

	var reputerValueBundles types.InputReputerValueBundles
	for i, reputerIndex := range reputerIndexes {
		valueBundle := &types.InputValueBundle{
			TopicId: topicId,
			ReputerRequestNonce: &types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{
					BlockHeight: blockHeight,
				},
			},
			ExtraData:                     nil,
			Reputer:                       s.addrsStr[reputerIndex],
			CombinedValue:                 alloraMath.MustNewBoundedExp40Dec(reputersLosses[i]),
			NaiveValue:                    alloraMath.MustNewBoundedExp40Dec(reputersNaiveLosses[i]),
			InfererValues:                 make([]*types.InputWorkerAttributedValue, len(workerIndexes)),
			ForecasterValues:              make([]*types.InputWorkerAttributedValue, len(forecasterIndexes)),
			OneOutInfererValues:           make([]*types.InputWithheldWorkerAttributedValue, len(workerIndexes)),
			OneOutForecasterValues:        make([]*types.InputWithheldWorkerAttributedValue, len(forecasterIndexes)),
			OneInForecasterValues:         make([]*types.InputWorkerAttributedValue, len(forecasterIndexes)),
			OneOutInfererForecasterValues: make([]*types.InputOneOutInfererForecasterValues, len(forecasterIndexes)),
		}

		for j, workerIndex := range workerIndexes {
			valueBundle.InfererValues[j] = &types.InputWorkerAttributedValue{Worker: s.addrsStr[workerIndex], Value: alloraMath.MustNewBoundedExp40Dec(reputersInfererLosses[i][j])}
			valueBundle.OneOutInfererValues[j] = &types.InputWithheldWorkerAttributedValue{Worker: s.addrsStr[workerIndex], Value: alloraMath.MustNewBoundedExp40Dec(reputersInfererOneOutLosses[i][j])}
		}

		for j, workerIndex := range forecasterIndexes {
			valueBundle.ForecasterValues[j] = &types.InputWorkerAttributedValue{Worker: s.addrsStr[workerIndex], Value: alloraMath.MustNewBoundedExp40Dec(reputersForecasterLosses[i][j])}
			valueBundle.OneOutForecasterValues[j] = &types.InputWithheldWorkerAttributedValue{Worker: s.addrsStr[workerIndex], Value: alloraMath.MustNewBoundedExp40Dec(reputersForecasterOneOutLosses[i][j])}
			valueBundle.OneInForecasterValues[j] = &types.InputWorkerAttributedValue{Worker: s.addrsStr[workerIndex], Value: alloraMath.MustNewBoundedExp40Dec(reputersOneInNaiveLosses[i][j])}
			valueBundle.OneOutInfererForecasterValues[j] = &types.InputOneOutInfererForecasterValues{Forecaster: s.addrsStr[workerIndex], OneOutInfererValues: valueBundle.OneOutInfererValues}
		}

		sig, err := signInputValueBundle(valueBundle, s.privKeys[reputerIndex])
		s.Require().NoError(err)

		bundle := &types.InputReputerValueBundle{
			Pubkey:      s.pubKeyHexStr[reputerIndex],
			Signature:   sig,
			ValueBundle: valueBundle,
		}
		reputerValueBundles.ReputerValueBundles = append(reputerValueBundles.ReputerValueBundles, bundle)
	}

	return reputerValueBundles
}

func generateHugeWorkerDataBundles(
	s *RewardsTestSuite,
	blockHeight int64,
	topicId uint64,
	workerIndexes []int,
) []*types.InputWorkerDataBundle {
	var inferences []*types.InputWorkerDataBundle
	for _, workerIndex := range workerIndexes {
		workerInferenceForecastBundle := &types.InputInferenceForecastBundle{
			Inference: &types.InputInference{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Inferer:     s.addrsStr[workerIndex],
				Value:       alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString(strconv.FormatInt(int64(rand.Intn(1000)+1), 10))),
				ExtraData:   nil,
				Proof:       "",
			},
			Forecast: &types.InputForecast{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Forecaster:  s.addrsStr[workerIndex],
				ForecastElements: []*types.InputForecastElement{
					{
						Inferer: s.addrs[26].String(),
						Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString(strconv.FormatInt(int64(rand.Intn(1000)+1), 10))),
					},
					{
						Inferer: s.addrs[27].String(),
						Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString(strconv.FormatInt(int64(rand.Intn(1000)+1), 10))),
					},
				},
				ExtraData: nil,
			},
		}
		workerSig, err := signInputInferenceForecastBundle(workerInferenceForecastBundle, s.privKeys[workerIndex])
		s.Require().NoError(err)
		workerBundle := &types.InputWorkerDataBundle{
			Worker:                             s.addrsStr[workerIndex],
			TopicId:                            topicId,
			Nonce:                              &types.Nonce{BlockHeight: blockHeight},
			InferenceForecastsBundle:           workerInferenceForecastBundle,
			InferencesForecastsBundleSignature: workerSig,
			Pubkey:                             s.pubKeyHexStr[workerIndex],
		}
		inferences = append(inferences, workerBundle)
	}
	return inferences
}

func signValueBundle(valueBundle *types.ValueBundle, privateKey secp256k1.PrivKey) ([]byte, error) {
	src := make([]byte, 0)
	src, err := valueBundle.XXX_Marshal(src, true)
	if err != nil {
		return nil, err
	}

	valueBundleSignature, err := privateKey.Sign(src)
	if err != nil {
		return nil, err
	}

	return valueBundleSignature, nil
}

func signInputValueBundle(InputValueBundle *types.InputValueBundle, privateKey secp256k1.PrivKey) ([]byte, error) {
	valueBundle, err := types.NewValueBundleFromInput(InputValueBundle)
	if err != nil {
		return nil, err
	}
	return signValueBundle(valueBundle, privateKey)
}

func signInferenceForecastBundle(
	inferenceForecastBundle *types.InferenceForecastBundle,
	privateKey secp256k1.PrivKey,
) ([]byte, error) {
	src := make([]byte, 0)
	src, err := inferenceForecastBundle.XXX_Marshal(src, true)
	if err != nil {
		return nil, err
	}

	sig, err := privateKey.Sign(src)
	if err != nil {
		return nil, err
	}

	return sig, nil
}

func signInputInferenceForecastBundle(
	InputInferenceForecastBundle *types.InputInferenceForecastBundle,
	privateKey secp256k1.PrivKey,
) ([]byte, error) {
	bundle, err := types.NewInferenceForecastBundleFromInput(InputInferenceForecastBundle)
	if err != nil {
		return nil, err
	}
	return signInferenceForecastBundle(bundle, privateKey)
}

func generateWorkerDataBundles(s *RewardsTestSuite, blockHeight int64, topicId uint64) []*types.InputWorkerDataBundle {
	var inferences []*types.InputWorkerDataBundle
	worker1 := 5
	worker2 := 6
	worker3 := 7
	worker4 := 8
	worker5 := 9

	// inference and forecast data - worker 1
	worker1InferenceForecastBundle := &types.InputInferenceForecastBundle{
		Inference: &types.InputInference{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Inferer:     s.addrsStr[worker1],
			Value:       alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01127")),
			ExtraData:   nil,
			Proof:       "",
		},
		Forecast: &types.InputForecast{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Forecaster:  s.addrsStr[worker1],
			ForecastElements: []*types.InputForecastElement{
				{
					Inferer: s.addrs[6].String(),
					Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01127")),
				},
				{
					Inferer: s.addrs[7].String(),
					Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01127")),
				},
			},
			ExtraData: nil,
		},
	}
	worker1Sig, err := signInputInferenceForecastBundle(worker1InferenceForecastBundle, s.privKeys[worker1])
	s.Require().NoError(err)
	worker1Bundle := &types.InputWorkerDataBundle{
		Worker:                             s.addrsStr[worker1],
		Nonce:                              &types.Nonce{BlockHeight: blockHeight},
		TopicId:                            topicId,
		InferenceForecastsBundle:           worker1InferenceForecastBundle,
		InferencesForecastsBundleSignature: worker1Sig,
		Pubkey:                             s.pubKeyHexStr[worker1],
	}
	inferences = append(inferences, worker1Bundle)
	// inference and forecast data - worker 2
	worker2InferenceForecastBundle := &types.InputInferenceForecastBundle{
		Inference: &types.InputInference{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Inferer:     s.addrsStr[worker2],
			Value:       alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01791")),
			ExtraData:   nil,
			Proof:       "",
		},
		Forecast: &types.InputForecast{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Forecaster:  s.addrsStr[worker2],
			ForecastElements: []*types.InputForecastElement{
				{
					Inferer: s.addrs[7].String(),
					Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01791")),
				},
				{
					Inferer: s.addrs[8].String(),
					Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01791")),
				},
			},
			ExtraData: nil,
		},
	}
	worker2Sig, err := signInputInferenceForecastBundle(worker2InferenceForecastBundle, s.privKeys[worker2])
	s.Require().NoError(err)
	worker2Bundle := &types.InputWorkerDataBundle{
		Worker:                             s.addrsStr[worker2],
		Nonce:                              &types.Nonce{BlockHeight: blockHeight},
		TopicId:                            topicId,
		InferenceForecastsBundle:           worker2InferenceForecastBundle,
		InferencesForecastsBundleSignature: worker2Sig,
		Pubkey:                             s.pubKeyHexStr[worker2],
	}
	inferences = append(inferences, worker2Bundle)
	// inference and forecast data - worker 3
	worker3InferenceForecastBundle := &types.InputInferenceForecastBundle{
		Inference: &types.InputInference{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Inferer:     s.addrsStr[worker3],
			Value:       alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01404")),
			ExtraData:   nil,
			Proof:       "",
		},
		Forecast: &types.InputForecast{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Forecaster:  s.addrsStr[worker3],
			ForecastElements: []*types.InputForecastElement{
				{
					Inferer: s.addrs[8].String(),
					Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01404")),
				},
				{
					Inferer: s.addrs[9].String(),
					Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01404")),
				},
			},
			ExtraData: nil,
		},
	}
	worker3Sig, err := signInputInferenceForecastBundle(worker3InferenceForecastBundle, s.privKeys[worker3])
	s.Require().NoError(err)
	worker3Bundle := &types.InputWorkerDataBundle{
		Worker:                             s.addrsStr[worker3],
		Nonce:                              &types.Nonce{BlockHeight: blockHeight},
		TopicId:                            topicId,
		InferenceForecastsBundle:           worker3InferenceForecastBundle,
		InferencesForecastsBundleSignature: worker3Sig,
		Pubkey:                             s.pubKeyHexStr[worker3],
	}
	inferences = append(inferences, worker3Bundle)
	// inference and forecast data - worker 4
	worker4InferenceForecastBundle := &types.InputInferenceForecastBundle{
		Inference: &types.InputInference{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Inferer:     s.addrsStr[worker4],
			Value:       alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.02318")),
			ExtraData:   nil,
			Proof:       "",
		},
		Forecast: &types.InputForecast{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Forecaster:  s.addrsStr[worker4],
			ForecastElements: []*types.InputForecastElement{
				{
					Inferer: s.addrs[9].String(),
					Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.02318")),
				},
				{
					Inferer: s.addrs[0].String(),
					Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.02318")),
				},
			},
			ExtraData: nil,
		},
	}
	worker4Sig, err := signInputInferenceForecastBundle(worker4InferenceForecastBundle, s.privKeys[worker4])
	s.Require().NoError(err)
	worker4Bundle := &types.InputWorkerDataBundle{
		Worker:                             s.addrsStr[worker4],
		Nonce:                              &types.Nonce{BlockHeight: blockHeight},
		TopicId:                            topicId,
		InferenceForecastsBundle:           worker4InferenceForecastBundle,
		InferencesForecastsBundleSignature: worker4Sig,
		Pubkey:                             s.pubKeyHexStr[worker4],
	}
	inferences = append(inferences, worker4Bundle)
	// inference and forecast data - worker 5
	worker5InferenceForecastBundle := &types.InputInferenceForecastBundle{
		Inference: &types.InputInference{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Inferer:     s.addrsStr[worker5],
			Value:       alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01251")),
			ExtraData:   nil,
			Proof:       "",
		},
		Forecast: &types.InputForecast{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Forecaster:  s.addrsStr[worker5],
			ForecastElements: []*types.InputForecastElement{
				{
					Inferer: s.addrs[0].String(),
					Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01251")),
				},
				{
					Inferer: s.addrs[1].String(),
					Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString("0.01251")),
				},
			},
			ExtraData: nil,
		},
	}
	worker5Sig, err := signInputInferenceForecastBundle(worker5InferenceForecastBundle, s.privKeys[worker5])
	s.Require().NoError(err)
	worker5Bundle := &types.InputWorkerDataBundle{
		Worker:                             s.addrsStr[worker5],
		Nonce:                              &types.Nonce{BlockHeight: blockHeight},
		TopicId:                            topicId,
		InferenceForecastsBundle:           worker5InferenceForecastBundle,
		InferencesForecastsBundleSignature: worker5Sig,
		Pubkey:                             s.pubKeyHexStr[worker5],
	}
	inferences = append(inferences, worker5Bundle)

	return inferences
}

/* to be rewritten in PROTO-3088
func generateMoreInferencesDataBundles(s *RewardsTestSuite, blockHeight int64, topicId uint64) []*types.WorkerDataBundle {
	var newInferences []*types.WorkerDataBundle
	worker1 := 13
	worker2 := 14

	worker1InferenceForecastBundle := &types.InferenceForecastBundle{
		Inference: &types.Inference{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Inferer:     s.addrsStr[worker1],
			Value:       alloraMath.MustNewDecFromString("0.01251"),
			ExtraData:   nil,
			Proof:       "",
		},
		Forecast: &types.Forecast{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Forecaster:  s.addrsStr[worker1],
			ForecastElements: []*types.ForecastElement{
				{
					Inferer: s.addrs[7].String(),
					Value:   alloraMath.MustNewDecFromString("0.01251"),
				},
				{
					Inferer: s.addrs[8].String(),
					Value:   alloraMath.MustNewDecFromString("0.01251"),
				},
			},
			ExtraData: nil,
		},
	}
	worker1Sig, err := signInferenceForecastBundle(worker1InferenceForecastBundle, s.privKeys[worker1])
	s.Require().NoError(err)
	worker1Bundle := &types.WorkerDataBundle{
		Worker:                             s.addrsStr[worker1],
		Nonce:                              &types.Nonce{BlockHeight: blockHeight},
		TopicId:                            topicId,
		InferenceForecastsBundle:           worker1InferenceForecastBundle,
		InferencesForecastsBundleSignature: worker1Sig,
		Pubkey:                             s.pubKeyHexStr[worker1],
	}
	newInferences = append(newInferences, worker1Bundle)

	worker2InferenceForecastBundle := &types.InferenceForecastBundle{
		Inference: &types.Inference{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Inferer:     s.addrsStr[worker2],
			Value:       alloraMath.MustNewDecFromString("10000"),
			ExtraData:   nil,
			Proof:       "",
		},
		Forecast: &types.Forecast{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Forecaster:  s.addrsStr[worker2],
			ForecastElements: []*types.ForecastElement{
				{
					Inferer: s.addrs[5].String(),
					Value:   alloraMath.MustNewDecFromString("0.01251"),
				},
				{
					Inferer: s.addrs[6].String(),
					Value:   alloraMath.MustNewDecFromString("0.01251"),
				},
			},
			ExtraData: nil,
		},
	}
	worker2Sig, err := signInferenceForecastBundle(worker2InferenceForecastBundle, s.privKeys[worker2])
	s.Require().NoError(err)
	worker2Bundle := &types.WorkerDataBundle{
		Worker:                             s.addrsStr[worker2],
		Nonce:                              &types.Nonce{BlockHeight: blockHeight},
		TopicId:                            topicId,
		InferenceForecastsBundle:           worker2InferenceForecastBundle,
		InferencesForecastsBundleSignature: worker2Sig,
		Pubkey:                             s.pubKeyHexStr[worker2],
	}
	newInferences = append(newInferences, worker2Bundle)

	return newInferences
}
*/

/* to be rewritten in PROTO-3088
func generateMoreForecastersDataBundles(s *RewardsTestSuite, blockHeight int64, topicId uint64) []*types.WorkerDataBundle {
	var newForecasts []*types.WorkerDataBundle
	worker1 := 13
	worker2 := 14

	worker1InferenceForecastBundle := &types.InferenceForecastBundle{
		Inference: &types.Inference{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Inferer:     s.addrsStr[worker1],
			Value:       alloraMath.MustNewDecFromString("0.01251"),
			ExtraData:   nil,
			Proof:       "",
		},
		Forecast: &types.Forecast{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Forecaster:  s.addrsStr[worker1],
			ForecastElements: []*types.ForecastElement{
				{
					Inferer: s.addrs[7].String(),
					Value:   alloraMath.MustNewDecFromString("0.01251"),
				},
				{
					Inferer: s.addrs[8].String(),
					Value:   alloraMath.MustNewDecFromString("0.01251"),
				},
			},
			ExtraData: nil,
		},
	}
	worker1Sig, err := signInferenceForecastBundle(worker1InferenceForecastBundle, s.privKeys[worker1])
	s.Require().NoError(err)
	worker1Bundle := &types.WorkerDataBundle{
		Worker:                             s.addrsStr[worker1],
		Nonce:                              &types.Nonce{BlockHeight: blockHeight},
		TopicId:                            topicId,
		InferenceForecastsBundle:           worker1InferenceForecastBundle,
		InferencesForecastsBundleSignature: worker1Sig,
		Pubkey:                             s.pubKeyHexStr[worker1],
	}
	newForecasts = append(newForecasts, worker1Bundle)

	worker2InferenceForecastBundle := &types.InferenceForecastBundle{
		Inference: &types.Inference{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Inferer:     s.addrsStr[worker2],
			Value:       alloraMath.MustNewDecFromString("0.01251"),
			ExtraData:   nil,
			Proof:       "",
		},
		Forecast: &types.Forecast{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Forecaster:  s.addrsStr[worker2],
			ForecastElements: []*types.ForecastElement{
				{
					Inferer: s.addrs[5].String(),
					Value:   alloraMath.MustNewDecFromString("0.01251"),
				},
				{
					Inferer: s.addrs[6].String(),
					Value:   alloraMath.MustNewDecFromString("0.01251"),
				},
			},
			ExtraData: nil,
		},
	}
	worker2Sig, err := signInferenceForecastBundle(worker2InferenceForecastBundle, s.privKeys[worker2])
	s.Require().NoError(err)
	worker2Bundle := &types.WorkerDataBundle{
		Worker:                             s.addrsStr[worker2],
		Nonce:                              &types.Nonce{BlockHeight: blockHeight},
		TopicId:                            topicId,
		InferenceForecastsBundle:           worker2InferenceForecastBundle,
		InferencesForecastsBundleSignature: worker2Sig,
		Pubkey:                             s.pubKeyHexStr[worker2],
	}
	newForecasts = append(newForecasts, worker2Bundle)

	return newForecasts
}
*/

type TestWorkerValue struct {
	Index int
	Value string
}

func generateSimpleWorkerDataBundles(
	s *RewardsTestSuite,
	topicId uint64,
	nonce int64,
	blockHeight int64,
	infererIndexes []int,
) []*types.InputWorkerDataBundle {
	require := s.Require()
	if len(infererIndexes) < 2 {
		require.Fail("workerValues must have at least 2 elements")
	}
	if len(infererIndexes) < 2 {
		require.Fail("infererIndexes must have at least 2 elements")
	}

	workerValues := make([]TestWorkerValue, len(infererIndexes))
	for i, index := range infererIndexes {
		workerValues[i] = TestWorkerValue{
			Index: index,
			Value: "100",
		}
	}

	var inferences []*types.InputWorkerDataBundle

	infererIndex := 0

	getInfererIndex := func() int {
		if infererIndex >= len(infererIndexes) {
			infererIndex = 0
		}
		currentInfererIndex := infererIndex
		infererIndex++
		return currentInfererIndex
	}

	for _, workerValue := range workerValues {
		newWorkerInferenceForecastBundle := &types.InputInferenceForecastBundle{
			Inference: &types.InputInference{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Inferer:     s.addrsStr[workerValue.Index],
				Value:       alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString(workerValue.Value)),
				ExtraData:   nil,
				Proof:       "",
			},
			Forecast: &types.InputForecast{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Forecaster:  s.addrsStr[workerValue.Index],
				ForecastElements: []*types.InputForecastElement{
					{
						Inferer: s.addrsStr[infererIndexes[getInfererIndex()]],
						Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString(workerValue.Value)),
					},
					{
						Inferer: s.addrsStr[infererIndexes[getInfererIndex()]],
						Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString(workerValue.Value)),
					},
				},
				ExtraData: nil,
			},
		}
		workerSig, err := signInputInferenceForecastBundle(newWorkerInferenceForecastBundle, s.privKeys[workerValue.Index])
		s.Require().NoError(err)
		workerBundle := &types.InputWorkerDataBundle{
			Worker:                             s.addrsStr[workerValue.Index],
			Nonce:                              &types.Nonce{BlockHeight: nonce},
			TopicId:                            topicId,
			InferenceForecastsBundle:           newWorkerInferenceForecastBundle,
			InferencesForecastsBundleSignature: workerSig,
			Pubkey:                             s.pubKeyHexStr[workerValue.Index],
		}
		inferences = append(inferences, workerBundle)
	}

	return inferences
}

func generateSimpleLossBundles(
	s *RewardsTestSuite,
	topicId uint64,
	nonce int64,
	workerValues []TestWorkerValue,
	reputerValues []TestWorkerValue,
	workerZeroAddress sdk.AccAddress,
	workerZeroOneOutInfererValue string,
	workerZeroInfererValue string,
) types.InputReputerValueBundles {
	var reputerValueBundles types.InputReputerValueBundles
	for _, reputer := range reputerValues {
		var countValues int
		if len(workerValues) < len(reputerValues) {
			countValues = len(workerValues)
		} else {
			countValues = len(reputerValues)
		}

		valueBundle := &types.InputValueBundle{
			TopicId: topicId,
			ReputerRequestNonce: &types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{
					BlockHeight: nonce,
				},
			},
			Reputer:                       s.addrsStr[reputer.Index],
			ExtraData:                     nil,
			CombinedValue:                 alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString(reputer.Value)),
			NaiveValue:                    alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString(reputer.Value)),
			InfererValues:                 make([]*types.InputWorkerAttributedValue, countValues),
			ForecasterValues:              make([]*types.InputWorkerAttributedValue, countValues),
			OneOutInfererValues:           make([]*types.InputWithheldWorkerAttributedValue, countValues),
			OneOutForecasterValues:        make([]*types.InputWithheldWorkerAttributedValue, countValues),
			OneInForecasterValues:         make([]*types.InputWorkerAttributedValue, countValues),
			OneOutInfererForecasterValues: make([]*types.InputOneOutInfererForecasterValues, countValues),
		}

		for j, worker := range workerValues {
			if j < len(reputerValues) {
				if s.addrs[worker.Index].Equals(workerZeroAddress) {
					valueBundle.InfererValues[j] = &types.InputWorkerAttributedValue{Worker: s.addrsStr[worker.Index], Value: alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString(workerZeroInfererValue))}
				} else {
					valueBundle.InfererValues[j] = &types.InputWorkerAttributedValue{Worker: s.addrsStr[worker.Index], Value: alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString(reputerValues[j].Value))}
				}
				valueBundle.ForecasterValues[j] = &types.InputWorkerAttributedValue{Worker: s.addrsStr[worker.Index], Value: alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString(reputerValues[j].Value))}
				if s.addrs[worker.Index].Equals(workerZeroAddress) {
					valueBundle.OneOutInfererValues[j] = &types.InputWithheldWorkerAttributedValue{Worker: s.addrsStr[worker.Index], Value: alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString(workerZeroOneOutInfererValue))}
				} else {
					valueBundle.OneOutInfererValues[j] = &types.InputWithheldWorkerAttributedValue{Worker: s.addrsStr[worker.Index], Value: alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString(reputerValues[j].Value))}
				}
				valueBundle.OneOutForecasterValues[j] = &types.InputWithheldWorkerAttributedValue{Worker: s.addrsStr[worker.Index], Value: alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString(reputerValues[j].Value))}
				valueBundle.OneInForecasterValues[j] = &types.InputWorkerAttributedValue{Worker: s.addrsStr[worker.Index], Value: alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString(reputerValues[j].Value))}
			}
		}
		for j, worker := range workerValues {
			if j < len(reputerValues) {
				valueBundle.OneOutInfererForecasterValues[j] = &types.InputOneOutInfererForecasterValues{Forecaster: s.addrsStr[worker.Index], OneOutInfererValues: valueBundle.OneOutInfererValues}
			}
		}

		sig, err := signInputValueBundle(valueBundle, s.privKeys[reputer.Index])
		s.Require().NoError(err)

		bundle := &types.InputReputerValueBundle{
			Pubkey:      s.pubKeyHexStr[reputer.Index],
			Signature:   sig,
			ValueBundle: valueBundle,
		}
		reputerValueBundles.ReputerValueBundles = append(reputerValueBundles.ReputerValueBundles, bundle)
	}

	return reputerValueBundles
}

func (s *RewardsTestSuite) TestGenerateReputerScoresWithZeroListeningCoefficients() {
	// Create a new topic
	newTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  s.addrs[0].String(),
		Metadata:                 "test",
		LossMethod:               "mse",
		EpochLength:              10800,
		GroundTruthLag:           10800,
		WorkerSubmissionWindow:   10,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:            true,
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		EnableWorkerWhitelist:    true,
		EnableReputerWhitelist:   true,
	}
	res, err := s.msgServer.CreateNewTopic(s.ctx, newTopicMsg)
	s.Require().NoError(err)
	topicId := res.TopicId
	block := int64(1003)

	// Set up reputer with zero listening coefficient
	reputer := s.addrsStr[13]
	stake := cosmosMath.NewInt(1000000000)

	// Mint tokens and add stake
	addrBech, err := sdk.AccAddressFromBech32(reputer)
	s.Require().NoError(err)
	s.MintTokensToAddress(addrBech, stake)
	err = s.emissionsKeeper.AddReputerStake(s.ctx, topicId, reputer, stake)
	s.Require().NoError(err)

	// Set zero listening coefficient
	err = s.emissionsKeeper.SetListeningCoefficient(
		s.ctx,
		topicId,
		reputer,
		types.ListeningCoefficient{Coefficient: alloraMath.ZeroDec()},
	)
	s.Require().NoError(err)

	// Create reported losses
	reportedLosses := types.ReputerValueBundles{
		ReputerValueBundles: []*types.ReputerValueBundle{
			{
				Pubkey: s.pubKeyHexStr[13],
				ValueBundle: &types.ValueBundle{
					TopicId: topicId,
					ReputerRequestNonce: &types.ReputerRequestNonce{
						ReputerNonce: &types.Nonce{BlockHeight: block},
					},
					Reputer:       reputer,
					CombinedValue: alloraMath.MustNewDecFromString("3.8"),
					NaiveValue:    alloraMath.MustNewDecFromString("3.8"),
					InfererValues: []*types.WorkerAttributedValue{
						{
							Worker: s.addrsStr[5],
							Value:  alloraMath.MustNewDecFromString("3.81"),
						},
						{
							Worker: s.addrsStr[6],
							Value:  alloraMath.MustNewDecFromString("3.82"),
						},
					},
					ForecasterValues:              nil,
					OneOutInfererValues:           nil,
					OneOutForecasterValues:        nil,
					OneInForecasterValues:         nil,
					OneOutInfererForecasterValues: nil,
					ExtraData:                     nil,
				},
			},
		},
	}

	// Sign the value bundle
	sig, err := signValueBundle(reportedLosses.ReputerValueBundles[0].ValueBundle, s.privKeys[13])
	s.Require().NoError(err)
	reportedLosses.ReputerValueBundles[0].Signature = sig

	// Get params and set epsilon reputer
	params := types.DefaultParams()
	params.EpsilonReputer = alloraMath.MustNewDecFromString("0.1")
	err = s.emissionsKeeper.SetParams(s.ctx, params)
	s.Require().NoError(err)

	// Generate scores
	scores, err := rewards.GenerateReputerScores(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		block,
		reportedLosses,
	)
	s.Require().NoError(err)
	s.Require().Len(scores, 1)

	// Verify that the listening coefficient was updated to epsilon reputer value
	coefficient, err := s.emissionsKeeper.GetListeningCoefficient(s.ctx, topicId, reputer)
	s.Require().NoError(err)
	s.Require().True(coefficient.Coefficient.Equal(params.EpsilonReputer))
}

func (s *RewardsTestSuite) TestCalculateTopicInitialEmaScore() {
	// Setup test scores
	scores := []types.Score{
		{
			TopicId:     1,
			BlockHeight: 1000,
			Address:     s.addrs[0].String(),
			Score:       alloraMath.MustNewDecFromString("0.5"),
		},
		{
			TopicId:     1,
			BlockHeight: 1000,
			Address:     s.addrs[1].String(),
			Score:       alloraMath.MustNewDecFromString("0.3"),
		},
		{
			TopicId:     1,
			BlockHeight: 1000,
			Address:     s.addrs[2].String(),
			Score:       alloraMath.MustNewDecFromString("0.1"),
		},
		{
			TopicId:     1,
			BlockHeight: 1000,
			Address:     s.addrs[3].String(),
			Score:       alloraMath.MustNewDecFromString("0.4"),
		},
		{
			TopicId:     1,
			BlockHeight: 1000,
			Address:     s.addrs[4].String(),
			Score:       alloraMath.MustNewDecFromString("0.2"),
		},
	}

	// Calculate initial EMA score
	initialScore, err := rewards.CalculateTopicInitialEmaScore(s.ctx, s.emissionsKeeper, scores)
	s.Require().NoError(err)

	// Get lambda from params
	params, err := s.emissionsKeeper.GetParams(s.ctx)
	s.Require().NoError(err)
	lambda := params.LambdaInitialScore

	// Calculate expected score manually
	// Standard deviation ≈ 0.1581139
	stdDev := alloraMath.MustNewDecFromString("0.1581139")
	lambdaStdDev, err := lambda.Mul(stdDev)
	s.Require().NoError(err)

	// Lowest score is 0.1
	lowestScore := alloraMath.MustNewDecFromString("0.1")
	expectedScore, err := lowestScore.Sub(lambdaStdDev)
	s.Require().NoError(err)

	// Verify result matches expected
	diff, err := initialScore.Sub(expectedScore)
	s.Require().NoError(err)
	absDiff, err := diff.Abs()
	s.Require().NoError(err)
	s.Require().True(absDiff.Lt(alloraMath.MustNewDecFromString("0.000001")))
}

func (s *RewardsTestSuite) TestCalculateTopicInitialEmaScoreEdgeCases() {
	testCases := []struct {
		name          string
		scores        []types.Score
		expectedError bool
		expectedScore string
	}{
		{
			name:          "empty scores",
			scores:        []types.Score{},
			expectedError: false,
			expectedScore: "0", // Returns zero when no scores
		},
		{
			name: "single score",
			scores: []types.Score{
				{Score: alloraMath.MustNewDecFromString("0.5")}, // nolint:exhaustruct
			},
			expectedError: false,
			expectedScore: "0.5", // With single score, no std dev calculation possible, returns the score
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			initialScore, err := rewards.CalculateTopicInitialEmaScore(s.ctx, s.emissionsKeeper, tc.scores)

			if tc.expectedError {
				s.Require().Error(err)
				return
			}

			s.Require().NoError(err)
			expectedScore := alloraMath.MustNewDecFromString(tc.expectedScore)
			diff, err := initialScore.Sub(expectedScore)
			s.Require().NoError(err)
			absDiff, err := diff.Abs()
			s.Require().NoError(err)
			s.Require().True(
				absDiff.Lt(alloraMath.MustNewDecFromString("0.000001")),
				"Expected %s but got %s", expectedScore, initialScore,
			)
		})
	}
}
