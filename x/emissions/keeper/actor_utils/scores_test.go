package actorutils_test

import (
	"encoding/hex"

	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/test/testutil"
	actorutils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *ActorUtilsTestSuite) TestGetReputersScoresFromCsv() {
	epochGet := testutil.GetSimulatedValuesGetterForEpochs()
	epoch300Get := epochGet[300]
	epoch301Get := epochGet[301]
	block := int64(1003)

	newTopicMsg := &types.MsgCreateNewTopic{
		Creator:                s.addrs[0].String(),
		Metadata:               "test",
		LossMethod:             "mse",
		EpochLength:            10800,
		GroundTruthLag:         10800,
		WorkerSubmissionWindow: 10,
		PNorm:                  alloraMath.NewDecFromInt64(3),
		AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:          true,
		Epsilon:                alloraMath.MustNewDecFromString("0.01"),
	}
	res, err := s.msgServer.CreateNewTopic(s.ctx, newTopicMsg)
	s.Require().NoError(err)
	topicId := res.TopicId

	reputer0 := s.addrs[0].String()
	reputer1 := s.addrs[1].String()
	reputer2 := s.addrs[2].String()
	reputer3 := s.addrs[3].String()
	reputer4 := s.addrs[4].String()
	reputerAddresses := []string{reputer0, reputer1, reputer2, reputer3, reputer4}

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
		infererAddresses,
		forecasterAddresses,
		reputerAddresses,
		epoch301Get,
	)
	s.Require().NoError(err)

	// Generate new reputer scores
	scores, err := actorutils.GenerateReputerScores(
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

func mockNetworkLosses(s *ActorUtilsTestSuite, topicId uint64, block int64) (types.ValueBundle, error) {
	oneOutInfererLosses := []*types.WithheldWorkerAttributedValue{
		{
			Worker: s.addrs[0].String(),
			Value:  alloraMath.MustNewDecFromString("0.01327"),
		},
		{
			Worker: s.addrs[1].String(),
			Value:  alloraMath.MustNewDecFromString("0.01302"),
		},
		{
			Worker: s.addrs[2].String(),
			Value:  alloraMath.MustNewDecFromString("0.0136"),
		},
		{
			Worker: s.addrs[3].String(),
			Value:  alloraMath.MustNewDecFromString("0.01491"),
		},
		{
			Worker: s.addrs[4].String(),
			Value:  alloraMath.MustNewDecFromString("0.01686"),
		},
	}

	oneOutForecasterLosses := []*types.WithheldWorkerAttributedValue{
		{
			Worker: s.addrs[0].String(),
			Value:  alloraMath.MustNewDecFromString("0.01402"),
		},
		{
			Worker: s.addrs[1].String(),
			Value:  alloraMath.MustNewDecFromString("0.01316"),
		},
		{
			Worker: s.addrs[2].String(),
			Value:  alloraMath.MustNewDecFromString("0.01657"),
		},
		{
			Worker: s.addrs[3].String(),
			Value:  alloraMath.MustNewDecFromString("0.0124"),
		},
		{
			Worker: s.addrs[4].String(),
			Value:  alloraMath.MustNewDecFromString("0.01341"),
		},
	}

	oneInNaiveLosses := []*types.WorkerAttributedValue{
		{
			Worker: s.addrs[0].String(),
			Value:  alloraMath.MustNewDecFromString("0.01529"),
		},
		{
			Worker: s.addrs[1].String(),
			Value:  alloraMath.MustNewDecFromString("0.01141"),
		},
		{
			Worker: s.addrs[2].String(),
			Value:  alloraMath.MustNewDecFromString("0.01562"),
		},
		{
			Worker: s.addrs[3].String(),
			Value:  alloraMath.MustNewDecFromString("0.01444"),
		},
		{
			Worker: s.addrs[4].String(),
			Value:  alloraMath.MustNewDecFromString("0.01396"),
		},
	}

	networkLosses := types.ValueBundle{
		TopicId:                topicId,
		CombinedValue:          alloraMath.MustNewDecFromString("0.013481256018186383"),
		NaiveValue:             alloraMath.MustNewDecFromString("0.01344474872292"),
		OneOutInfererValues:    oneOutInfererLosses,
		OneOutForecasterValues: oneOutForecasterLosses,
		OneInForecasterValues:  oneInNaiveLosses,
	}

	// Persist network losses
	err := s.emissionsKeeper.InsertNetworkLossBundleAtBlock(s.ctx, topicId, block, networkLosses)
	if err != nil {
		return types.ValueBundle{}, err
	}

	return networkLosses, nil
}

func (s *ActorUtilsTestSuite) TestGetInferenceScores() {
	topicId := uint64(1)
	block := int64(1003)

	// Generate workers data for tests
	reportedLosses, err := mockNetworkLosses(s, topicId, block)
	s.Require().NoError(err)

	// Get inference scores
	scores, err := actorutils.GenerateInferenceScores(
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
		deltaTightness := scoreDelta.Abs().
			Cmp(alloraMath.MustNewDecFromString("0.00001"))
		if !(deltaTightness == alloraMath.LessThan || deltaTightness == alloraMath.EqualTo) {
			s.Fail("Expected reward is not equal to the actual reward")
		}
	}
}

func (s *ActorUtilsTestSuite) TestGetInferenceScoresFromCsv() {
	epochGet := testutil.GetSimulatedValuesGetterForEpochs()
	for i := 300; i < 305; i++ {
		epoch3Get := epochGet[i]
		topicId := uint64(1)
		block := int64(i)

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

		reportedLosses, err := testutil.GetNetworkLossFromCsv(topicId, infererAddresses, forecasterAddresses, epoch3Get)
		s.Require().NoError(err)

		scores, err := actorutils.GenerateInferenceScores(
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
}

func mockSimpleNetworkLosses(
	s *ActorUtilsTestSuite,
	topicId uint64,
	block int64,
	worker0Value string,
) (types.ValueBundle, error) {
	genericLossesWithheld := []*types.WithheldWorkerAttributedValue{
		{
			Worker: s.addrs[0].String(),
			Value:  alloraMath.MustNewDecFromString(worker0Value),
		},
		{
			Worker: s.addrs[1].String(),
			Value:  alloraMath.MustNewDecFromString("0.3"),
		},
		{
			Worker: s.addrs[2].String(),
			Value:  alloraMath.MustNewDecFromString("0.4"),
		},
	}

	genericLosses := []*types.WorkerAttributedValue{
		{
			Worker: s.addrs[0].String(),
			Value:  alloraMath.MustNewDecFromString(worker0Value),
		},
		{
			Worker: s.addrs[1].String(),
			Value:  alloraMath.MustNewDecFromString("0.3"),
		},
		{
			Worker: s.addrs[2].String(),
			Value:  alloraMath.MustNewDecFromString("0.4"),
		},
	}

	networkLosses := types.ValueBundle{
		TopicId:                topicId,
		CombinedValue:          alloraMath.MustNewDecFromString("0.05"),
		NaiveValue:             alloraMath.MustNewDecFromString("0.05"),
		OneOutInfererValues:    genericLossesWithheld,
		OneOutForecasterValues: genericLossesWithheld,
		OneInForecasterValues:  genericLosses,
	}

	err := s.emissionsKeeper.InsertNetworkLossBundleAtBlock(s.ctx, topicId, block, networkLosses)
	if err != nil {
		return types.ValueBundle{}, err
	}

	return networkLosses, nil
}

// In this test we run two trials of generating inference scores, the first with lower one out losses
// and the second with higher one out losses.
// We then compare the resulting scores and check that the higher one out losses result in higher scores.
func (s *ActorUtilsTestSuite) TestHigherOneOutLossesHigherInferenceScore() {
	topicId := uint64(1)
	block0 := int64(1003)
	require := s.Require()

	networkLosses0, err := mockSimpleNetworkLosses(s, topicId, block0, "0.1")
	require.NoError(err)

	scores0, err := actorutils.GenerateInferenceScores(
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

	scores1, err := actorutils.GenerateInferenceScores(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		block1,
		networkLosses1,
	)
	require.NoError(err)

	require.True(scores0[0].Score.Lt(scores1[0].Score))
}

func (s *ActorUtilsTestSuite) TestGetForecastScores() {
	topicId := uint64(1)
	block := int64(1003)

	// Generate workers data for tests
	reportedLosses, err := mockNetworkLosses(s, topicId, block)
	s.Require().NoError(err)

	scores, err := actorutils.GenerateForecastScores(
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
		deltaTightness := delta.Abs().Cmp(alloraMath.MustNewDecFromString("0.00001"))
		if !(deltaTightness == alloraMath.LessThan || deltaTightness == alloraMath.EqualTo) {
			s.Fail("Expected reward is not equal to the actual reward")
		}
	}
}

func (s *ActorUtilsTestSuite) TestGetForecasterScoresFromCsv() {
	epochGet := testutil.GetSimulatedValuesGetterForEpochs()
	epoch3Get := epochGet[300]
	topicId := uint64(1)
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

	reportedLosses, err := testutil.GetNetworkLossFromCsv(topicId, infererAddresses, forecasterAddresses, epoch3Get)
	s.Require().NoError(err)

	scores, err := actorutils.GenerateForecastScores(
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
func (s *ActorUtilsTestSuite) TestHigherOneOutLossesHigherForecastScore() {
	topicId := uint64(1)
	block0 := int64(1003)
	require := s.Require()

	networkLosses0, err := mockSimpleNetworkLosses(s, topicId, block0, "0.1")
	require.NoError(err)

	scores0, err := actorutils.GenerateForecastScores(
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
	scores1, err := actorutils.GenerateForecastScores(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		block1,
		networkLosses1,
	)
	require.NoError(err)

	require.True(scores0[0].Score.Lt(scores1[0].Score))
}

func (s *ActorUtilsTestSuite) TestEnsureAllWorkersPresent() {
	// Setup
	allWorkers := map[string]struct{}{
		"worker1": {},
		"worker2": {},
		"worker3": {},
		"worker4": {},
	}

	values := []*types.WorkerAttributedValue{
		{Worker: "worker1", Value: alloraMath.NewDecFromInt64(100)},
		{Worker: "worker3", Value: alloraMath.NewDecFromInt64(300)},
	}

	expectedValues := map[string]string{
		"worker1": "100",
		"worker2": "NaN",
		"worker3": "300",
		"worker4": "NaN",
	}

	// Act
	updatedValues := actorutils.EnsureAllWorkersPresent(values, allWorkers)

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

func (s *ActorUtilsTestSuite) TestEnsureAllWorkersPresentWithheld() {
	// Setup
	allWorkers := map[string]struct{}{
		"worker1": {},
		"worker2": {},
		"worker3": {},
		"worker4": {},
	}

	values := []*types.WithheldWorkerAttributedValue{
		{Worker: "worker1", Value: alloraMath.NewDecFromInt64(100)},
		{Worker: "worker3", Value: alloraMath.NewDecFromInt64(300)},
	}

	expectedValues := map[string]string{
		"worker1": "100",
		"worker2": "NaN",
		"worker3": "300",
		"worker4": "NaN",
	}

	// Act
	updatedValues := actorutils.EnsureAllWorkersPresentWithheld(values, allWorkers)

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

func GenerateReputerLatestScores(s *ActorUtilsTestSuite, reputers []sdk.AccAddress, blockHeight int64, topicId uint64) error {
	var scores = []alloraMath.Dec{
		alloraMath.MustNewDecFromString("17.53436"),
		alloraMath.MustNewDecFromString("20.29489"),
		alloraMath.MustNewDecFromString("24.26994"),
		alloraMath.MustNewDecFromString("11.36754"),
		alloraMath.MustNewDecFromString("15.21749"),
	}
	for i, reputerAddr := range reputers {
		scoreToAdd := types.Score{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Address:     reputerAddr.String(),
			Score:       scores[i],
		}
		err := s.emissionsKeeper.InsertReputerScore(s.ctx, topicId, blockHeight, scoreToAdd)
		if err != nil {
			return err
		}
	}

	return nil
}

func GenerateReputerSignature(s *ActorUtilsTestSuite, valueBundle *types.ValueBundle, account sdk.AccAddress) ([]byte, error) {
	protoBytesIn := make([]byte, 0)
	protoBytesIn, err := valueBundle.XXX_Marshal(protoBytesIn, true)
	if err != nil {
		return nil, err
	}

	privKey := s.privKeys[account.String()]
	sig, err := privKey.Sign(protoBytesIn)
	if err != nil {
		return nil, err
	}

	return sig, nil
}

func GenerateWorkerSignature(s *ActorUtilsTestSuite, valueBundle *types.InferenceForecastBundle, account sdk.AccAddress) ([]byte, error) {
	protoBytesIn := make([]byte, 0)
	protoBytesIn, err := valueBundle.XXX_Marshal(protoBytesIn, true)
	if err != nil {
		return nil, err
	}

	privKey := s.privKeys[account.String()]
	sig, err := privKey.Sign(protoBytesIn)
	if err != nil {
		return nil, err
	}

	return sig, nil
}

func GetAccPubKey(s *ActorUtilsTestSuite, address sdk.AccAddress) string {
	pk := s.privKeys[address.String()].PubKey().Bytes()
	return hex.EncodeToString(pk)
}
