package rewards_test

import (
	"fmt"

	cosmosMath "cosmossdk.io/math"
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
	topicId := uint64(1)

	reputerAddresses := make([]string, 5)
	for i := range len(reputerAddresses) {
		reputerAddresses[i] = s.AddrsStr(13 + i)
	}
	infererAddresses := make([]string, 5)
	for i := range len(infererAddresses) {
		infererAddresses[i] = s.AddrsStr(5 + i)
	}
	forecasterAddresses := make([]string, 3)
	for i := range len(forecasterAddresses) {
		forecasterAddresses[i] = s.AddrsStr(10 + i)
	}
	reputers := make([]testutil.ReputerKey, 5)
	for i := range len(reputers) {
		reputers[i] = testutil.ReputerKey{
			Address:    s.AddrsStr(13 + i),
			PrivateKey: s.PrivKeys(13 + i),
			PubKeyHex:  s.PubKeyHexStr(13 + i),
		}
	}

	cosmosOneE18 := inferencesynthesis.CosmosIntOneE18()
	cosmosOneE18Dec, err := alloraMath.NewDecFromSdkInt(cosmosOneE18)
	s.Require().NoError(err)

	stakes := make([]cosmosMath.Int, 5)
	for i := range len(stakes) {
		reputerStake, err := epoch301Get(fmt.Sprintf("reputer_stake_%d", i)).Mul(cosmosOneE18Dec)
		s.Require().NoError(err)
		reputerStakeInt, err := reputerStake.BigInt()
		s.Require().NoError(err)
		stakes[i] = cosmosMath.NewIntFromBigInt(reputerStakeInt)
	}

	coefficients := make([]alloraMath.Dec, 5)
	for i := range len(coefficients) {
		coefficients[i] = epoch300Get(fmt.Sprintf("reputer_listening_coefficient_%d", i))
	}

	for i, addr := range reputerAddresses {
		addrBech, err := sdk.AccAddressFromBech32(addr)
		s.Require().NoError(err)

		s.MintTokensToAddress(addrBech, stakes[i])

		err = s.StakingKeeper().AddReputerStake(s.Ctx(), topicId, addr, stakes[i])
		s.Require().NoError(err)

		err = s.ScoresKeeper().SetListeningCoefficient(
			s.Ctx(),
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

	scores, err := rewards.GenerateReputerScores(
		s.Ctx(),
		*s.EmissionsKeeper(),
		topicId,
		block,
		reportedLosses,
	)
	s.Require().NoError(err)

	expectedScores := make([]alloraMath.Dec, 5)
	for i := range len(expectedScores) {
		expectedScores[i] = epoch301Get(fmt.Sprintf("reputer_score_%d", i))
	}

	for i, reputerScore := range scores {
		testutil.InEpsilon5(s.T(), reputerScore.Score, expectedScores[i].String())
	}
}

func (s *RewardsTestSuite) TestGetInferenceScores() {
	topicId := uint64(1)
	block := int64(1003)

	// Generate workers data for tests
	reportedLosses, err := mockNetworkLosses(s, topicId, block)
	s.Require().NoError(err)

	// Get inference scores
	scores, err := rewards.GenerateInferenceScores(
		s.Ctx(),
		*s.EmissionsKeeper(),
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
	topicId := uint64(1)
	block := int64(2001)

	// Generate normal network losses, then set all one-out/one-in fields to nil
	reportedLosses, err := mockNetworkLosses(s, topicId, block)
	s.Require().NoError(err)

	reportedLosses.OneOutInfererValues = nil
	reportedLosses.OneOutForecasterValues = nil
	reportedLosses.OneInForecasterValues = nil

	scores, err := rewards.GenerateInferenceScores(
		s.Ctx(),
		*s.EmissionsKeeper(),
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
	topicId := uint64(1)
	block := int64(1003)

	infererAddresses := make([]string, 5)
	for i := range infererAddresses {
		infererAddresses[i] = s.AddrsStr(i + 5)
	}
	forecasterAddresses := make([]string, 3)
	for i := range forecasterAddresses {
		forecasterAddresses[i] = s.AddrsStr(i + 10)
	}
	reputer0 := s.AddrsStr(13)

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
		s.Ctx(),
		*s.EmissionsKeeper(),
		topicId,
		block,
		reportedLosses,
	)
	s.Require().NoError(err)

	expectedScores := make([]alloraMath.Dec, 5)
	for i := range expectedScores {
		expectedScores[i] = epoch3Get(fmt.Sprintf("inferer_score_%d", i))
	}

	for i, infererScore := range scores {
		testutil.InEpsilon5(s.T(), infererScore.Score, expectedScores[i].String())
	}
}

// In this test we run two trials of generating inference scores, the first with lower one out losses
// and the second with higher one out losses.
// We then compare the resulting scores and check that the higher one out losses result in higher scores.
func (s *RewardsTestSuite) TestHigherOneOutLossesHigherInferenceScore() {
	topicId := uint64(1)
	block0 := int64(1003)
	require := s.Require()

	networkLosses0, err := mockSimpleNetworkLosses(s, topicId, block0, "0.1")
	require.NoError(err)

	scores0, err := rewards.GenerateInferenceScores(
		s.Ctx(),
		*s.EmissionsKeeper(),
		topicId,
		block0,
		networkLosses0,
	)
	require.NoError(err)

	block1 := block0 + 1

	networkLosses1, err := mockSimpleNetworkLosses(s, topicId, block1, "0.2")
	require.NoError(err)

	scores1, err := rewards.GenerateInferenceScores(
		s.Ctx(),
		*s.EmissionsKeeper(),
		topicId,
		block1,
		networkLosses1,
	)
	require.NoError(err)

	require.True(scores0[0].Score.Lt(scores1[0].Score))
}

func (s *RewardsTestSuite) TestGetForecastScores() {
	block := int64(1003)
	topicId := uint64(1)

	// Generate workers data for tests
	reportedLosses, err := mockNetworkLosses(s, topicId, block)
	s.Require().NoError(err)

	scores, err := rewards.GenerateForecastScores(
		s.Ctx(),
		*s.EmissionsKeeper(),
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
	topicId := uint64(1)
	block := int64(1003)

	infererAddresses := make([]string, 5)
	for i := range infererAddresses {
		infererAddresses[i] = s.AddrsStr(i + 5)
	}
	forecasterAddresses := make([]string, 3)
	for i := range forecasterAddresses {
		forecasterAddresses[i] = s.AddrsStr(i + 10)
	}
	reputer0 := s.AddrsStr(13)

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
		s.Ctx(),
		*s.EmissionsKeeper(),
		topicId,
		block,
		reportedLosses,
	)
	s.Require().NoError(err)

	expectedScores := make([]alloraMath.Dec, 3)
	for i := range expectedScores {
		expectedScores[i] = epoch3Get(fmt.Sprintf("forecaster_score_%d", i))
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
	topicId := uint64(1)
	block0 := int64(1003)
	require := s.Require()

	networkLosses0, err := mockSimpleNetworkLosses(s, topicId, block0, "0.1")
	require.NoError(err)

	scores0, err := rewards.GenerateForecastScores(
		s.Ctx(),
		*s.EmissionsKeeper(),
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
		s.Ctx(),
		*s.EmissionsKeeper(),
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
		s.AddrsStr(1): {},
		s.AddrsStr(2): {},
		"worker3":     {},
		"worker4":     {},
	}

	values := []*types.WorkerAttributedValue{
		{Worker: s.AddrsStr(1), Value: alloraMath.NewDecFromInt64(100)},
		{Worker: "worker3", Value: alloraMath.NewDecFromInt64(300)},
	}

	expectedValues := map[string]string{
		s.AddrsStr(1): "100",
		s.AddrsStr(2): "NaN",
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
		s.AddrsStr(1): {},
		s.AddrsStr(2): {},
		"worker3":     {},
		"worker4":     {},
	}

	values := []*types.WithheldWorkerAttributedValue{
		{Worker: s.AddrsStr(1), Value: alloraMath.NewDecFromInt64(100)},
		{Worker: "worker3", Value: alloraMath.NewDecFromInt64(300)},
	}

	expectedValues := map[string]string{
		s.AddrsStr(1): "100",
		s.AddrsStr(2): "NaN",
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
						{Worker: s.AddrsStr(1), Value: alloraMath.NewDecFromInt64(100)},
						{Worker: s.AddrsStr(2), Value: alloraMath.NewDecFromInt64(200)},
					},
					ForecasterValues: []*types.WorkerAttributedValue{
						{Worker: s.AddrsStr(1), Value: alloraMath.NewDecFromInt64(300)},
					},
					OneOutInfererValues: []*types.WithheldWorkerAttributedValue{
						{Worker: s.AddrsStr(1), Value: alloraMath.NewDecFromInt64(100)},
						{Worker: s.AddrsStr(2), Value: alloraMath.NewDecFromInt64(200)},
					},
					OneOutForecasterValues: []*types.WithheldWorkerAttributedValue{
						{Worker: s.AddrsStr(1), Value: alloraMath.NewDecFromInt64(300)},
					},
					OneInForecasterValues: []*types.WorkerAttributedValue{
						{Worker: s.AddrsStr(2), Value: alloraMath.NewDecFromInt64(400)},
					},
					OneOutInfererForecasterValues: []*types.OneOutInfererForecasterValues{
						{
							Forecaster: "allo13kenskkx7e0v253m3kcgwfc67cmx00fgwpgj6h",
							OneOutInfererValues: []*types.WithheldWorkerAttributedValue{
								{Worker: s.AddrsStr(1), Value: alloraMath.NewDecFromInt64(500)},
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
						{Worker: s.AddrsStr(1), Value: alloraMath.NewDecFromInt64(100)},
						{Worker: s.AddrsStr(2), Value: alloraMath.NewDecFromInt64(200)},
						{Worker: "worker3", Value: alloraMath.NewDecFromInt64(300)},
						{Worker: "worker4", Value: alloraMath.NewDecFromInt64(400)},
					},
					OneOutForecasterValues: []*types.WithheldWorkerAttributedValue{
						{Worker: s.AddrsStr(1), Value: alloraMath.NewDecFromInt64(500)},
						{Worker: "worker3", Value: alloraMath.NewDecFromInt64(600)},
					},
					OneInForecasterValues: []*types.WorkerAttributedValue{
						{Worker: s.AddrsStr(2), Value: alloraMath.NewDecFromInt64(700)},
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

func (s *RewardsTestSuite) TestGenerateReputerScoresWithZeroListeningCoefficients() {
	topicId := uint64(1)
	block := int64(1003)

	// Set up reputer with zero listening coefficient
	reputer := s.AddrsStr(13)
	stake := cosmosMath.NewInt(1000000000)

	// Mint tokens and add stake
	addrBech, err := sdk.AccAddressFromBech32(reputer)
	s.Require().NoError(err)
	s.MintTokensToAddress(addrBech, stake)
	err = s.StakingKeeper().AddReputerStake(s.Ctx(), topicId, reputer, stake)
	s.Require().NoError(err)

	// Set zero listening coefficient
	err = s.ScoresKeeper().SetListeningCoefficient(
		s.Ctx(),
		topicId,
		reputer,
		types.ListeningCoefficient{Coefficient: alloraMath.ZeroDec()},
	)
	s.Require().NoError(err)

	// Create reported losses
	//nolint:exhaustruct
	reportedLosses := types.ReputerValueBundles{
		ReputerValueBundles: []*types.ReputerValueBundle{
			{
				Pubkey: s.PubKeyHexStr(13),
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
							Worker: s.AddrsStr(5),
							Value:  alloraMath.MustNewDecFromString("3.81"),
						},
						{
							Worker: s.AddrsStr(6),
							Value:  alloraMath.MustNewDecFromString("3.82"),
						},
					},
				},
			},
		},
	}

	// Sign the value bundle
	sig := s.SignValueBundle(reportedLosses.ReputerValueBundles[0].ValueBundle, s.PrivKeys(13))
	s.Require().NoError(err)
	reportedLosses.ReputerValueBundles[0].Signature = sig

	// Get params and set epsilon reputer
	params := types.DefaultParams()
	params.EpsilonReputer = alloraMath.MustNewDecFromString("0.1")
	err = s.ParamsKeeper().SetParams(s.Ctx(), params)
	s.Require().NoError(err)

	// Generate scores
	scores, err := rewards.GenerateReputerScores(
		s.Ctx(),
		*s.EmissionsKeeper(),
		topicId,
		block,
		reportedLosses,
	)
	s.Require().NoError(err)
	s.Require().Len(scores, 1)

	// Verify that the listening coefficient was updated to epsilon reputer value
	coefficient, err := s.ScoresKeeper().GetListeningCoefficient(s.Ctx(), topicId, reputer)
	s.Require().NoError(err)
	s.Require().True(coefficient.Coefficient.Equal(params.EpsilonReputer))
}

func (s *RewardsTestSuite) TestCalculateTopicInitialEmaScore() {
	// Setup test scores
	scores := []types.Score{
		{
			TopicId:     1,
			BlockHeight: 1000,
			Address:     s.AddrsStr(0),
			Score:       alloraMath.MustNewDecFromString("0.5"),
		},
		{
			TopicId:     1,
			BlockHeight: 1000,
			Address:     s.AddrsStr(1),
			Score:       alloraMath.MustNewDecFromString("0.3"),
		},
		{
			TopicId:     1,
			BlockHeight: 1000,
			Address:     s.AddrsStr(2),
			Score:       alloraMath.MustNewDecFromString("0.1"),
		},
		{
			TopicId:     1,
			BlockHeight: 1000,
			Address:     s.AddrsStr(3),
			Score:       alloraMath.MustNewDecFromString("0.4"),
		},
		{
			TopicId:     1,
			BlockHeight: 1000,
			Address:     s.AddrsStr(4),
			Score:       alloraMath.MustNewDecFromString("0.2"),
		},
	}

	// Calculate initial EMA score
	initialScore, err := rewards.CalculateTopicInitialEmaScore(s.Ctx(), s.ParamsKeeper(), scores)
	s.Require().NoError(err)

	// Get lambda from params
	params, err := s.ParamsKeeper().GetParams(s.Ctx())
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
			initialScore, err := rewards.CalculateTopicInitialEmaScore(s.Ctx(), s.ParamsKeeper(), tc.scores)

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

func getQuantileScoreFromScores(scores []types.Score, quantile alloraMath.Dec) (alloraMath.Dec, error) {
	decScores := make([]alloraMath.Dec, 0, len(scores))
	for _, score := range scores {
		decScores = append(decScores, score.Score)
	}
	return alloraMath.GetQuantileOfDecs(decScores, quantile)
}

func (s *RewardsTestSuite) TestInfererQuantileStoredFromEmaScores() {
	topicId := uint64(1)
	block0 := int64(4000)
	block1 := int64(4001)

	networkLosses0, err := mockSimpleNetworkLosses(s, topicId, block0, "0.1")
	s.Require().NoError(err)
	_, err = rewards.GenerateInferenceScores(s.Ctx(), *s.EmissionsKeeper(), topicId, block0, networkLosses0)
	s.Require().NoError(err)

	networkLosses1, err := mockSimpleNetworkLosses(s, topicId, block1, "0.2")
	s.Require().NoError(err)
	instantScores, err := rewards.GenerateInferenceScores(s.Ctx(), *s.EmissionsKeeper(), topicId, block1, networkLosses1)
	s.Require().NoError(err)

	topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)
	storedQuantile, err := s.ScoresKeeper().GetPreviousTopicQuantileInfererScoreEma(s.Ctx(), topicId)
	s.Require().NoError(err)

	emaScores := make([]types.Score, 0, len(instantScores))
	for _, score := range instantScores {
		emaScore, err := s.ScoresKeeper().GetInfererScoreEma(s.Ctx(), topicId, score.Address)
		s.Require().NoError(err)
		emaScores = append(emaScores, emaScore)
	}

	expectedEmaQuantile, err := getQuantileScoreFromScores(emaScores, topic.ActiveInfererQuantile)
	s.Require().NoError(err)
	s.Require().True(storedQuantile.Equal(expectedEmaQuantile))
}

func (s *RewardsTestSuite) TestForecasterQuantileStoredFromEmaScores() {
	topicId := uint64(1)
	block0 := int64(5000)
	block1 := int64(5001)

	networkLosses0, err := mockSimpleNetworkLosses(s, topicId, block0, "0.1")
	s.Require().NoError(err)
	_, err = rewards.GenerateForecastScores(s.Ctx(), *s.EmissionsKeeper(), topicId, block0, networkLosses0)
	s.Require().NoError(err)

	networkLosses1, err := mockSimpleNetworkLosses(s, topicId, block1, "0.2")
	s.Require().NoError(err)
	instantScores, err := rewards.GenerateForecastScores(s.Ctx(), *s.EmissionsKeeper(), topicId, block1, networkLosses1)
	s.Require().NoError(err)

	topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)
	storedQuantile, err := s.ScoresKeeper().GetPreviousTopicQuantileForecasterScoreEma(s.Ctx(), topicId)
	s.Require().NoError(err)

	emaScores := make([]types.Score, 0, len(instantScores))
	for _, score := range instantScores {
		emaScore, err := s.ScoresKeeper().GetForecasterScoreEma(s.Ctx(), topicId, score.Address)
		s.Require().NoError(err)
		emaScores = append(emaScores, emaScore)
	}

	expectedEmaQuantile, err := getQuantileScoreFromScores(emaScores, topic.ActiveForecasterQuantile)
	s.Require().NoError(err)
	s.Require().True(storedQuantile.Equal(expectedEmaQuantile))
}

func (s *RewardsTestSuite) TestReputerQuantileStoredFromEmaScores() {
	epochGet := testutil.GetSimulatedValuesGetterForEpochs()
	epoch300Get := epochGet[300]
	epoch301Get := epochGet[301]
	epoch302Get := epochGet[302]
	topicId := uint64(1)
	block0 := int64(6000)
	block1 := int64(6001)

	reputerAddresses := make([]string, 5)
	for i := range len(reputerAddresses) {
		reputerAddresses[i] = s.AddrsStr(13 + i)
	}
	infererAddresses := make([]string, 5)
	for i := range len(infererAddresses) {
		infererAddresses[i] = s.AddrsStr(5 + i)
	}
	forecasterAddresses := make([]string, 3)
	for i := range len(forecasterAddresses) {
		forecasterAddresses[i] = s.AddrsStr(10 + i)
	}
	reputers := make([]testutil.ReputerKey, 5)
	for i := range len(reputers) {
		reputers[i] = testutil.ReputerKey{
			Address:    s.AddrsStr(13 + i),
			PrivateKey: s.PrivKeys(13 + i),
			PubKeyHex:  s.PubKeyHexStr(13 + i),
		}
	}

	cosmosOneE18 := inferencesynthesis.CosmosIntOneE18()
	cosmosOneE18Dec, err := alloraMath.NewDecFromSdkInt(cosmosOneE18)
	s.Require().NoError(err)
	for i, addr := range reputerAddresses {
		reputerStake, err := epoch301Get(fmt.Sprintf("reputer_stake_%d", i)).Mul(cosmosOneE18Dec)
		s.Require().NoError(err)
		reputerStakeInt, err := reputerStake.BigInt()
		s.Require().NoError(err)
		stake := cosmosMath.NewIntFromBigInt(reputerStakeInt)

		addrBech, err := sdk.AccAddressFromBech32(addr)
		s.Require().NoError(err)
		s.MintTokensToAddress(addrBech, stake)
		err = s.StakingKeeper().AddReputerStake(s.Ctx(), topicId, addr, stake)
		s.Require().NoError(err)
		err = s.ScoresKeeper().SetListeningCoefficient(
			s.Ctx(),
			topicId,
			addr,
			types.ListeningCoefficient{Coefficient: epoch300Get(fmt.Sprintf("reputer_listening_coefficient_%d", i))},
		)
		s.Require().NoError(err)
	}

	reportedLosses0, err := testutil.GetReputersDataFromCsv(
		topicId,
		block0,
		infererAddresses,
		forecasterAddresses,
		reputers,
		epoch301Get,
	)
	s.Require().NoError(err)
	_, err = rewards.GenerateReputerScores(s.Ctx(), *s.EmissionsKeeper(), topicId, block0, reportedLosses0)
	s.Require().NoError(err)

	reportedLosses1, err := testutil.GetReputersDataFromCsv(
		topicId,
		block1,
		infererAddresses,
		forecasterAddresses,
		reputers,
		epoch302Get,
	)
	s.Require().NoError(err)
	instantScores, err := rewards.GenerateReputerScores(s.Ctx(), *s.EmissionsKeeper(), topicId, block1, reportedLosses1)
	s.Require().NoError(err)

	topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)
	storedQuantile, err := s.ScoresKeeper().GetPreviousTopicQuantileReputerScoreEma(s.Ctx(), topicId)
	s.Require().NoError(err)

	emaScores := make([]types.Score, 0, len(instantScores))
	for _, score := range instantScores {
		emaScore, err := s.ScoresKeeper().GetReputerScoreEma(s.Ctx(), topicId, score.Address)
		s.Require().NoError(err)
		emaScores = append(emaScores, emaScore)
	}

	expectedEmaQuantile, err := getQuantileScoreFromScores(emaScores, topic.ActiveReputerQuantile)
	s.Require().NoError(err)
	s.Require().True(storedQuantile.Equal(expectedEmaQuantile))
}
