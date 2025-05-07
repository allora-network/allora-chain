package rewards_test

import (
	"math/rand"
	"strconv"

	cosmosMath "cosmossdk.io/math"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/test/testutil"
	"github.com/allora-network/allora-chain/utils/fn"
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

func (s *RewardsTestSuite) generateLossBundles(
	blockHeight int64,
	topicId uint64,
	reputerIndexes []int,
	reputerValues ...map[string]string,
) (reputerValueBundles types.InputReputerValueBundles) {
	rv := len(reputerValues)
	hasReputerValues := rv > 0
	if hasReputerValues && len(reputerIndexes) != rv {
		panic("invalid reputer values length")
	}

	networkInferences, err := s.emissionsKeeper.GetLatestNetworkInferences(s.ctx, topicId, false)
	s.Require().NoError(err)

	deltaVal := alloraMath.MustNewDecFromString("0.01988")

	for i, reputerIndex := range reputerIndexes {
		var val alloraMath.Dec
		combinedVal := alloraMath.MustNewDecFromString("0.1") // , _ := networkInferences.CombinedValue.Sub(deltaVal)

		valueBundle := &types.InputValueBundle{
			TopicId: topicId,
			ReputerRequestNonce: &types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{
					BlockHeight: blockHeight,
				},
			},
			Reputer:       s.addrsStr[reputerIndex],
			ExtraData:     nil,
			CombinedValue: alloraMath.MustNewBoundedExp40Dec(combinedVal),
			NaiveValue:    alloraMath.MustNewBoundedExp40Dec(combinedVal),
			InfererValues: fn.Map(networkInferences.InfererValues, func(inf *types.WorkerAttributedValue) *types.InputWorkerAttributedValue {
				val, _ = inf.Value.Sub(deltaVal)
				if hasReputerValues {
					val = alloraMath.MustNewDecFromString(reputerValues[i][inf.Worker])
				}
				return &types.InputWorkerAttributedValue{Worker: inf.Worker, Value: alloraMath.MustNewBoundedExp40Dec(val)}
			}),
			ForecasterValues: fn.Map(networkInferences.ForecasterValues, func(inf *types.WorkerAttributedValue) *types.InputWorkerAttributedValue {
				val, _ = inf.Value.Sub(deltaVal)
				if hasReputerValues {
					val = alloraMath.MustNewDecFromString(reputerValues[i][inf.Worker])
				}
				return &types.InputWorkerAttributedValue{Worker: inf.Worker, Value: alloraMath.MustNewBoundedExp40Dec(val)}
			}),
			OneOutInfererValues: fn.Map(networkInferences.OneOutInfererValues, func(inf *types.WithheldWorkerAttributedValue) *types.InputWithheldWorkerAttributedValue {
				val, _ = inf.Value.Sub(deltaVal)
				if hasReputerValues {
					val = alloraMath.MustNewDecFromString(reputerValues[i][inf.Worker])
				}
				return &types.InputWithheldWorkerAttributedValue{Worker: inf.Worker, Value: alloraMath.MustNewBoundedExp40Dec(val)}
			}),
			OneOutForecasterValues: fn.Map(networkInferences.OneOutForecasterValues, func(inf *types.WithheldWorkerAttributedValue) *types.InputWithheldWorkerAttributedValue {
				val, _ = inf.Value.Sub(deltaVal)
				if hasReputerValues {
					val = alloraMath.MustNewDecFromString(reputerValues[i][inf.Worker])
				}
				return &types.InputWithheldWorkerAttributedValue{Worker: inf.Worker, Value: alloraMath.MustNewBoundedExp40Dec(val)}
			}),
			OneInForecasterValues: fn.Map(networkInferences.OneInForecasterValues, func(inf *types.WorkerAttributedValue) *types.InputWorkerAttributedValue {
				val, _ = inf.Value.Sub(deltaVal)
				if hasReputerValues {
					val = alloraMath.MustNewDecFromString(reputerValues[i][inf.Worker])
				}
				return &types.InputWorkerAttributedValue{Worker: inf.Worker, Value: alloraMath.MustNewBoundedExp40Dec(val)}
			}),
			OneOutInfererForecasterValues: fn.Map(networkInferences.OneOutInfererForecasterValues, func(inf *types.OneOutInfererForecasterValues) *types.InputOneOutInfererForecasterValues {
				return &types.InputOneOutInfererForecasterValues{Forecaster: inf.Forecaster, OneOutInfererValues: fn.Map(inf.OneOutInfererValues, func(inf *types.WithheldWorkerAttributedValue) *types.InputWithheldWorkerAttributedValue {
					val, _ = inf.Value.Sub(deltaVal)
					if hasReputerValues {
						val = alloraMath.MustNewDecFromString(reputerValues[i][inf.Worker])
					}
					return &types.InputWithheldWorkerAttributedValue{Worker: inf.Worker, Value: alloraMath.MustNewCappedBoundedExp40Dec(val)}
				})}
			}),
		}

		if len(valueBundle.ForecasterValues) > 0 {
			infererValues := make([]*types.InputWorkerAttributedValue, len(valueBundle.InfererValues))
			for i, inf := range valueBundle.InfererValues {
				infererValues[i] = &types.InputWorkerAttributedValue{
					Worker: inf.Worker,
					Value:  inf.Value,
				}
			}
			valueBundle.ForecasterValues = infererValues
			valueBundle.ForecasterValues[0].Value = valueBundle.CombinedValue
			valueBundle.OneOutInfererValues[0].Value = valueBundle.CombinedValue
			valueBundle.OneOutForecasterValues = valueBundle.OneOutInfererValues
			valueBundle.OneOutForecasterValues[0].Value = valueBundle.CombinedValue
			valueBundle.OneInForecasterValues = valueBundle.ForecasterValues
			valueBundle.OneInForecasterValues[0].Value = valueBundle.CombinedValue
		}
		valueBundle.OneOutInfererForecasterValues = nil

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

func generateWorkerDataBundles(s *RewardsTestSuite, blockHeight int64, topicId uint64, workerIndexes []int, workerValues ...TestWorkerValue) []*types.InputWorkerDataBundle {
	lwv := len(workerValues)
	hasWorkerValues := lwv > 0
	if hasWorkerValues && len(workerIndexes) != lwv {
		panic("invalid worker values length")
	}
	var bundles []*types.InputWorkerDataBundle
	totalAddresses := len(s.addrsStr)

	for i, workerIdx := range workerIndexes {
		// Generate random inference value between 0.01 and 0.025
		inferenceValueStr := workerValues[i].Value
		if !hasWorkerValues {
			rand.Seed(int64(workerIdx) + blockHeight)
			inferenceValueStr = strconv.FormatFloat(0.01+rand.Float64()*0.015, 'f', 5, 64)
		}

		// Select forecast targets (next two workers in sequence, wrapping if needed)
		forecastTargets := []int{
			(workerIdx + 0) % totalAddresses,
			(workerIdx + 1) % totalAddresses,
			(workerIdx + 2) % totalAddresses,
		}

		// Create forecast elements
		forecastElements := []*types.InputForecastElement{
			{
				Inferer: s.addrs[forecastTargets[0]].String(),
				Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString(inferenceValueStr)),
			},
			{
				Inferer: s.addrs[forecastTargets[1]].String(),
				Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString(inferenceValueStr)),
			},
			{
				Inferer: s.addrs[forecastTargets[2]].String(),
				Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString(inferenceValueStr)),
			},
		}

		// Create inference-forecast bundle
		inferenceForecastBundle := &types.InputInferenceForecastBundle{
			Inference: &types.InputInference{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Inferer:     s.addrsStr[workerIdx],
				Value:       alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString(inferenceValueStr)),
				ExtraData:   nil,
				Proof:       "",
			},
			Forecast: &types.InputForecast{
				TopicId:          topicId,
				BlockHeight:      blockHeight,
				Forecaster:       s.addrsStr[workerIdx],
				ForecastElements: forecastElements,
				ExtraData:        nil,
			},
		}

		// Sign the bundle
		signature, err := signInputInferenceForecastBundle(inferenceForecastBundle, s.privKeys[workerIdx])
		s.Require().NoError(err)

		// Create the complete worker data bundle
		bundle := &types.InputWorkerDataBundle{
			Worker:                             s.addrsStr[workerIdx],
			Nonce:                              &types.Nonce{BlockHeight: blockHeight},
			TopicId:                            topicId,
			InferenceForecastsBundle:           inferenceForecastBundle,
			InferencesForecastsBundleSignature: signature,
			Pubkey:                             s.pubKeyHexStr[workerIdx],
		}

		bundles = append(bundles, bundle)
	}

	return bundles
}

type TestWorkerValue struct {
	Index int
	Value string
}

func generateSimpleWorkerDataBundles(
	s *RewardsTestSuite,
	topicId uint64,
	nonce int64,
	blockHeight int64,
	workerValues []TestWorkerValue,
	infererIndexes []int,
) []*types.InputWorkerDataBundle {
	require := s.Require()
	if len(workerValues) < 2 {
		require.Fail("workerValues must have at least 2 elements")
	}
	if len(infererIndexes) < 2 {
		require.Fail("infererIndexes must have at least 2 elements")
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
			OneOutInfererForecasterValues: nil,
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
