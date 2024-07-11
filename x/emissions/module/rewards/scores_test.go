package rewards_test

import (
	"encoding/hex"
	"math/rand"
	"strconv"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// this test is timing out more than 30 seconds
// need to figure out why later
// func (s *RewardsTestSuite) TestGetReputersScores() {
// 	topidId := uint64(1)
// 	block := int64(1003)

// 	// Generate reputers data for tests
// 	reportedLosses, err := mockReputersScoresTestData(s, topidId, block)
// 	s.Require().NoError(err)

// 	// Generate new reputer scores
// 	scores, err := rewards.GenerateReputerScores(
// 		s.ctx,
// 		s.emissionsKeeper,
// 		topidId,
// 		block,
// 		reportedLosses,
// 	)
// 	s.Require().NoError(err)

// 	expectedScores := []alloraMath.Dec{
// 		alloraMath.MustNewDecFromString("17.98648"),
// 		alloraMath.MustNewDecFromString("20.32339"),
// 		alloraMath.MustNewDecFromString("26.44637"),
// 		alloraMath.MustNewDecFromString("11.17804"),
// 		alloraMath.MustNewDecFromString("14.93222"),
// 	}
// 	for i, reputerScore := range scores {
// 		scoreDelta, err := reputerScore.Score.Sub(expectedScores[i])
// 		s.Require().NoError(err)
// 		deltaTightness := scoreDelta.Abs().Cmp(alloraMath.MustNewDecFromString("0.001"))
// 		if !(deltaTightness == alloraMath.LessThan || deltaTightness == alloraMath.EqualTo) {
// 			s.Fail("Expected reward is not equal to the actual reward")
// 		}
// 	}
// }

func (s *RewardsTestSuite) TestGetInferenceScores() {
	topicId := uint64(1)
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
		deltaTightness := scoreDelta.Abs().
			Cmp(alloraMath.MustNewDecFromString("0.00001"))
		if !(deltaTightness == alloraMath.LessThan || deltaTightness == alloraMath.EqualTo) {
			s.Fail("Expected reward is not equal to the actual reward")
		}
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
	topicId := uint64(1)
	block := int64(1003)

	// Generate workers data for tests
	reportedLosses, err := mockNetworkLosses(s, topicId, block)
	s.Require().NoError(err)

	// Get inference scores
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
		deltaTightness := delta.Abs().Cmp(alloraMath.MustNewDecFromString("0.00001"))
		if !(deltaTightness == alloraMath.LessThan || deltaTightness == alloraMath.EqualTo) {
			s.Fail("Expected reward is not equal to the actual reward")
		}
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

func GenerateReputerLatestScores(s *RewardsTestSuite, reputers []sdk.AccAddress, blockHeight int64, topicId uint64) error {
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

func PrepareMockLosses(reputersCount int, workersCount int) (
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
func GenerateLossBundles(s *RewardsTestSuite, blockHeight int64, topicId uint64, reputers []sdk.AccAddress) types.ReputerValueBundles {
	workers := []sdk.AccAddress{
		s.addrs[5],
		s.addrs[6],
		s.addrs[7],
		s.addrs[8],
		s.addrs[9],
	}
	reputersLosses := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("0.01127"),
		alloraMath.MustNewDecFromString("0.01791"),
		alloraMath.MustNewDecFromString("0.01404"),
		alloraMath.MustNewDecFromString("0.02318"),
		alloraMath.MustNewDecFromString("0.01251"),
	}
	reputersInfererLosses := [][]alloraMath.Dec{
		{
			alloraMath.MustNewDecFromString("0.0112"),
			alloraMath.MustNewDecFromString("0.00231"),
			alloraMath.MustNewDecFromString("0.02274"),
			alloraMath.MustNewDecFromString("0.01299"),
			alloraMath.MustNewDecFromString("0.02515"),
		},
		{
			alloraMath.MustNewDecFromString("0.01635"),
			alloraMath.MustNewDecFromString("0.00179"),
			alloraMath.MustNewDecFromString("0.03396"),
			alloraMath.MustNewDecFromString("0.0153"),
			alloraMath.MustNewDecFromString("0.01988"),
		},
		{
			alloraMath.MustNewDecFromString("0.01345"),
			alloraMath.MustNewDecFromString("0.00209"),
			alloraMath.MustNewDecFromString("0.03249"),
			alloraMath.MustNewDecFromString("0.01688"),
			alloraMath.MustNewDecFromString("0.02126"),
		},
		{
			alloraMath.MustNewDecFromString("0.01675"),
			alloraMath.MustNewDecFromString("0.00318"),
			alloraMath.MustNewDecFromString("0.02623"),
			alloraMath.MustNewDecFromString("0.02734"),
			alloraMath.MustNewDecFromString("0.03526"),
		},
		{
			alloraMath.MustNewDecFromString("0.02093"),
			alloraMath.MustNewDecFromString("0.00213"),
			alloraMath.MustNewDecFromString("0.02462"),
			alloraMath.MustNewDecFromString("0.0203"),
			alloraMath.MustNewDecFromString("0.03115"),
		},
	}
	reputersForecasterLosses := [][]alloraMath.Dec{
		{
			alloraMath.MustNewDecFromString("0.0185"),
			alloraMath.MustNewDecFromString("0.01018"),
			alloraMath.MustNewDecFromString("0.02105"),
			alloraMath.MustNewDecFromString("0.01041"),
			alloraMath.MustNewDecFromString("0.0183"),
		},
		{
			alloraMath.MustNewDecFromString("0.00962"),
			alloraMath.MustNewDecFromString("0.01191"),
			alloraMath.MustNewDecFromString("0.01616"),
			alloraMath.MustNewDecFromString("0.01417"),
			alloraMath.MustNewDecFromString("0.01216"),
		},
		{
			alloraMath.MustNewDecFromString("0.01338"),
			alloraMath.MustNewDecFromString("0.0116"),
			alloraMath.MustNewDecFromString("0.01605"),
			alloraMath.MustNewDecFromString("0.0133"),
			alloraMath.MustNewDecFromString("0.01407"),
		},
		{
			alloraMath.MustNewDecFromString("0.02733"),
			alloraMath.MustNewDecFromString("0.01697"),
			alloraMath.MustNewDecFromString("0.01619"),
			alloraMath.MustNewDecFromString("0.01925"),
			alloraMath.MustNewDecFromString("0.02018"),
		},
		{
			alloraMath.MustNewDecFromString("0.01"),
			alloraMath.MustNewDecFromString("0.01545"),
			alloraMath.MustNewDecFromString("0.01785"),
			alloraMath.MustNewDecFromString("0.01662"),
			alloraMath.MustNewDecFromString("0.01156"),
		},
	}
	reputersNaiveLosses := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("0.0116"),
		alloraMath.MustNewDecFromString("0.01428"),
		alloraMath.MustNewDecFromString("0.01441"),
		alloraMath.MustNewDecFromString("0.01594"),
		alloraMath.MustNewDecFromString("0.01705"),
	}
	reputersInfererOneOutLosses := [][]alloraMath.Dec{
		{
			alloraMath.MustNewDecFromString("0.0148"),
			alloraMath.MustNewDecFromString("0.01046"),
			alloraMath.MustNewDecFromString("0.01192"),
			alloraMath.MustNewDecFromString("0.01381"),
			alloraMath.MustNewDecFromString("0.01687"),
		},
		{
			alloraMath.MustNewDecFromString("0.01043"),
			alloraMath.MustNewDecFromString("0.01308"),
			alloraMath.MustNewDecFromString("0.01455"),
			alloraMath.MustNewDecFromString("0.01607"),
			alloraMath.MustNewDecFromString("0.01205"),
		},
		{
			alloraMath.MustNewDecFromString("0.01339"),
			alloraMath.MustNewDecFromString("0.01053"),
			alloraMath.MustNewDecFromString("0.01424"),
			alloraMath.MustNewDecFromString("0.01428"),
			alloraMath.MustNewDecFromString("0.01446"),
		},
		{
			alloraMath.MustNewDecFromString("0.01674"),
			alloraMath.MustNewDecFromString("0.02944"),
			alloraMath.MustNewDecFromString("0.01796"),
			alloraMath.MustNewDecFromString("0.02187"),
			alloraMath.MustNewDecFromString("0.01895"),
		},
		{
			alloraMath.MustNewDecFromString("0.01049"),
			alloraMath.MustNewDecFromString("0.02068"),
			alloraMath.MustNewDecFromString("0.01573"),
			alloraMath.MustNewDecFromString("0.01487"),
			alloraMath.MustNewDecFromString("0.02639"),
		},
	}
	reputersForecasterOneOutLosses := [][]alloraMath.Dec{
		{
			alloraMath.MustNewDecFromString("0.01136"),
			alloraMath.MustNewDecFromString("0.01185"),
			alloraMath.MustNewDecFromString("0.01568"),
			alloraMath.MustNewDecFromString("0.00949"),
			alloraMath.MustNewDecFromString("0.01339"),
		},
		{
			alloraMath.MustNewDecFromString("0.01357"),
			alloraMath.MustNewDecFromString("0.01108"),
			alloraMath.MustNewDecFromString("0.01633"),
			alloraMath.MustNewDecFromString("0.01208"),
			alloraMath.MustNewDecFromString("0.01278"),
		},
		{
			alloraMath.MustNewDecFromString("0.01805"),
			alloraMath.MustNewDecFromString("0.01229"),
			alloraMath.MustNewDecFromString("0.01586"),
			alloraMath.MustNewDecFromString("0.01234"),
			alloraMath.MustNewDecFromString("0.01513"),
		},
		{
			alloraMath.MustNewDecFromString("0.01637"),
			alloraMath.MustNewDecFromString("0.01594"),
			alloraMath.MustNewDecFromString("0.01608"),
			alloraMath.MustNewDecFromString("0.02203"),
			alloraMath.MustNewDecFromString("0.01486"),
		},
		{
			alloraMath.MustNewDecFromString("0.01981"),
			alloraMath.MustNewDecFromString("0.02123"),
			alloraMath.MustNewDecFromString("0.02134"),
			alloraMath.MustNewDecFromString("0.0217"),
			alloraMath.MustNewDecFromString("0.01177"),
		},
	}
	reputersOneInNaiveLosses := [][]alloraMath.Dec{
		{
			alloraMath.MustNewDecFromString("0.01588"),
			alloraMath.MustNewDecFromString("0.01012"),
			alloraMath.MustNewDecFromString("0.01467"),
			alloraMath.MustNewDecFromString("0.0128"),
			alloraMath.MustNewDecFromString("0.01234"),
		},
		{
			alloraMath.MustNewDecFromString("0.01239"),
			alloraMath.MustNewDecFromString("0.01023"),
			alloraMath.MustNewDecFromString("0.01712"),
			alloraMath.MustNewDecFromString("0.0116"),
			alloraMath.MustNewDecFromString("0.01639"),
		},
		{
			alloraMath.MustNewDecFromString("0.01419"),
			alloraMath.MustNewDecFromString("0.01497"),
			alloraMath.MustNewDecFromString("0.01629"),
			alloraMath.MustNewDecFromString("0.01514"),
			alloraMath.MustNewDecFromString("0.01133"),
		},
		{
			alloraMath.MustNewDecFromString("0.01936"),
			alloraMath.MustNewDecFromString("0.01518"),
			alloraMath.MustNewDecFromString("0.018"),
			alloraMath.MustNewDecFromString("0.02212"),
			alloraMath.MustNewDecFromString("0.02259"),
		},
		{
			alloraMath.MustNewDecFromString("0.01602"),
			alloraMath.MustNewDecFromString("0.01194"),
			alloraMath.MustNewDecFromString("0.0153"),
			alloraMath.MustNewDecFromString("0.0199"),
			alloraMath.MustNewDecFromString("0.01673"),
		},
	}

	var reputerValueBundles types.ReputerValueBundles
	for i, reputer := range reputers {
		valueBundle := &types.ValueBundle{
			TopicId: topicId,
			ReputerRequestNonce: &types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{
					BlockHeight: blockHeight,
				},
			},
			Reputer:                reputer.String(),
			CombinedValue:          reputersLosses[i],
			NaiveValue:             reputersNaiveLosses[i],
			InfererValues:          make([]*types.WorkerAttributedValue, len(workers)),
			ForecasterValues:       make([]*types.WorkerAttributedValue, len(workers)),
			OneOutInfererValues:    make([]*types.WithheldWorkerAttributedValue, len(workers)),
			OneOutForecasterValues: make([]*types.WithheldWorkerAttributedValue, len(workers)),
			OneInForecasterValues:  make([]*types.WorkerAttributedValue, len(workers)),
		}

		for j, worker := range workers {
			valueBundle.InfererValues[j] = &types.WorkerAttributedValue{Worker: worker.String(), Value: reputersInfererLosses[i][j]}
			valueBundle.ForecasterValues[j] = &types.WorkerAttributedValue{Worker: worker.String(), Value: reputersForecasterLosses[i][j]}
			valueBundle.OneOutInfererValues[j] = &types.WithheldWorkerAttributedValue{Worker: worker.String(), Value: reputersInfererOneOutLosses[i][j]}
			valueBundle.OneOutForecasterValues[j] = &types.WithheldWorkerAttributedValue{Worker: worker.String(), Value: reputersForecasterOneOutLosses[i][j]}
			valueBundle.OneInForecasterValues[j] = &types.WorkerAttributedValue{Worker: worker.String(), Value: reputersOneInNaiveLosses[i][j]}
		}

		sig, err := GenerateReputerSignature(s, valueBundle, reputer)
		s.Require().NoError(err)

		bundle := &types.ReputerValueBundle{
			Pubkey:      GetAccPubKey(s, reputer),
			Signature:   sig,
			ValueBundle: valueBundle,
		}
		reputerValueBundles.ReputerValueBundles = append(reputerValueBundles.ReputerValueBundles, bundle)
	}

	return reputerValueBundles
}

func GenerateHugeLossBundles(s *RewardsTestSuite, blockHeight int64, topicId uint64, reputers []sdk.AccAddress, workers []sdk.AccAddress) types.ReputerValueBundles {

	var (
		reputersLosses,
		reputersInfererLosses,
		reputersForecasterLosses,
		reputersNaiveLosses,
		reputersInfererOneOutLosses,
		reputersForecasterOneOutLosses,
		reputersOneInNaiveLosses = PrepareMockLosses(len(reputers), len(workers))
	)

	var reputerValueBundles types.ReputerValueBundles
	for i, reputer := range reputers {
		valueBundle := &types.ValueBundle{
			TopicId: topicId,
			ReputerRequestNonce: &types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{
					BlockHeight: blockHeight,
				},
			},
			Reputer:                reputer.String(),
			CombinedValue:          reputersLosses[i],
			NaiveValue:             reputersNaiveLosses[i],
			InfererValues:          make([]*types.WorkerAttributedValue, len(workers)),
			ForecasterValues:       make([]*types.WorkerAttributedValue, len(workers)),
			OneOutInfererValues:    make([]*types.WithheldWorkerAttributedValue, len(workers)),
			OneOutForecasterValues: make([]*types.WithheldWorkerAttributedValue, len(workers)),
			OneInForecasterValues:  make([]*types.WorkerAttributedValue, len(workers)),
		}

		for j, worker := range workers {
			valueBundle.InfererValues[j] = &types.WorkerAttributedValue{Worker: worker.String(), Value: reputersInfererLosses[i][j]}
			valueBundle.ForecasterValues[j] = &types.WorkerAttributedValue{Worker: worker.String(), Value: reputersForecasterLosses[i][j]}
			valueBundle.OneOutInfererValues[j] = &types.WithheldWorkerAttributedValue{Worker: worker.String(), Value: reputersInfererOneOutLosses[i][j]}
			valueBundle.OneOutForecasterValues[j] = &types.WithheldWorkerAttributedValue{Worker: worker.String(), Value: reputersForecasterOneOutLosses[i][j]}
			valueBundle.OneInForecasterValues[j] = &types.WorkerAttributedValue{Worker: worker.String(), Value: reputersOneInNaiveLosses[i][j]}
		}

		sig, err := GenerateReputerSignature(s, valueBundle, reputer)
		s.Require().NoError(err)

		bundle := &types.ReputerValueBundle{
			Pubkey:      GetAccPubKey(s, reputer),
			Signature:   sig,
			ValueBundle: valueBundle,
		}
		reputerValueBundles.ReputerValueBundles = append(reputerValueBundles.ReputerValueBundles, bundle)
	}

	return reputerValueBundles
}

func GenerateHugeWorkerDataBundles(s *RewardsTestSuite, blockHeight int64, topicId uint64, workers []sdk.AccAddress) []*types.WorkerDataBundle {
	var inferences []*types.WorkerDataBundle
	for _, worker := range workers {
		workerInferenceForecastBundle := &types.InferenceForecastBundle{
			Inference: &types.Inference{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Inferer:     worker.String(),
				Value:       alloraMath.MustNewDecFromString(strconv.FormatInt(int64(rand.Intn(1000)+1), 10)),
			},
			Forecast: &types.Forecast{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Forecaster:  worker.String(),
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: s.addrs[26].String(),
						Value:   alloraMath.MustNewDecFromString(strconv.FormatInt(int64(rand.Intn(1000)+1), 10)),
					},
					{
						Inferer: s.addrs[27].String(),
						Value:   alloraMath.MustNewDecFromString(strconv.FormatInt(int64(rand.Intn(1000)+1), 10)),
					},
				},
			},
		}
		workerSig, err := GenerateWorkerSignature(s, workerInferenceForecastBundle, worker)
		s.Require().NoError(err)
		workerBundle := &types.WorkerDataBundle{
			Worker:                             worker.String(),
			InferenceForecastsBundle:           workerInferenceForecastBundle,
			InferencesForecastsBundleSignature: workerSig,
			Pubkey:                             GetAccPubKey(s, worker),
		}
		inferences = append(inferences, workerBundle)
	}
	return inferences
}
func GenerateReputerSignature(s *RewardsTestSuite, valueBundle *types.ValueBundle, account sdk.AccAddress) ([]byte, error) {
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

func GenerateWorkerSignature(s *RewardsTestSuite, valueBundle *types.InferenceForecastBundle, account sdk.AccAddress) ([]byte, error) {
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

func GetAccPubKey(s *RewardsTestSuite, address sdk.AccAddress) string {
	pk := s.privKeys[address.String()].PubKey().Bytes()
	return hex.EncodeToString(pk)
}

func GenerateWorkerDataBundles(s *RewardsTestSuite, blockHeight int64, topicId uint64) []*types.WorkerDataBundle {
	var inferences []*types.WorkerDataBundle
	worker1Addr := s.addrs[5]
	worker2Addr := s.addrs[6]
	worker3Addr := s.addrs[7]
	worker4Addr := s.addrs[8]
	worker5Addr := s.addrs[9]

	// inference and forecast data - worker 1
	worker1InferenceForecastBundle := &types.InferenceForecastBundle{
		Inference: &types.Inference{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Inferer:     worker1Addr.String(),
			Value:       alloraMath.MustNewDecFromString("0.01127"),
		},
		Forecast: &types.Forecast{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Forecaster:  worker1Addr.String(),
			ForecastElements: []*types.ForecastElement{
				{
					Inferer: s.addrs[6].String(),
					Value:   alloraMath.MustNewDecFromString("0.01127"),
				},
				{
					Inferer: s.addrs[7].String(),
					Value:   alloraMath.MustNewDecFromString("0.01127"),
				},
			},
		},
	}
	worker1Sig, err := GenerateWorkerSignature(s, worker1InferenceForecastBundle, worker1Addr)
	s.Require().NoError(err)
	worker1Bundle := &types.WorkerDataBundle{
		Worker:                             worker1Addr.String(),
		InferenceForecastsBundle:           worker1InferenceForecastBundle,
		InferencesForecastsBundleSignature: worker1Sig,
		Pubkey:                             GetAccPubKey(s, worker1Addr),
	}
	inferences = append(inferences, worker1Bundle)
	// inference and forecast data - worker 2
	worker2InferenceForecastBundle := &types.InferenceForecastBundle{
		Inference: &types.Inference{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Inferer:     worker2Addr.String(),
			Value:       alloraMath.MustNewDecFromString("0.01791"),
		},
		Forecast: &types.Forecast{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Forecaster:  worker2Addr.String(),
			ForecastElements: []*types.ForecastElement{
				{
					Inferer: s.addrs[7].String(),
					Value:   alloraMath.MustNewDecFromString("0.01791"),
				},
				{
					Inferer: s.addrs[8].String(),
					Value:   alloraMath.MustNewDecFromString("0.01791"),
				},
			},
		},
	}
	worker2Sig, err := GenerateWorkerSignature(s, worker2InferenceForecastBundle, worker2Addr)
	s.Require().NoError(err)
	worker2Bundle := &types.WorkerDataBundle{
		Worker:                             worker2Addr.String(),
		InferenceForecastsBundle:           worker2InferenceForecastBundle,
		InferencesForecastsBundleSignature: worker2Sig,
		Pubkey:                             GetAccPubKey(s, worker2Addr),
	}
	inferences = append(inferences, worker2Bundle)
	// inference and forecast data - worker 3
	worker3InferenceForecastBundle := &types.InferenceForecastBundle{
		Inference: &types.Inference{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Inferer:     worker3Addr.String(),
			Value:       alloraMath.MustNewDecFromString("0.01404"),
		},
		Forecast: &types.Forecast{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Forecaster:  worker3Addr.String(),
			ForecastElements: []*types.ForecastElement{
				{
					Inferer: s.addrs[8].String(),
					Value:   alloraMath.MustNewDecFromString("0.01404"),
				},
				{
					Inferer: s.addrs[9].String(),
					Value:   alloraMath.MustNewDecFromString("0.01404"),
				},
			},
		},
	}
	worker3Sig, err := GenerateWorkerSignature(s, worker3InferenceForecastBundle, worker3Addr)
	s.Require().NoError(err)
	worker3Bundle := &types.WorkerDataBundle{
		Worker:                             worker3Addr.String(),
		InferenceForecastsBundle:           worker3InferenceForecastBundle,
		InferencesForecastsBundleSignature: worker3Sig,
		Pubkey:                             GetAccPubKey(s, worker3Addr),
	}
	inferences = append(inferences, worker3Bundle)
	// inference and forecast data - worker 4
	worker4InferenceForecastBundle := &types.InferenceForecastBundle{
		Inference: &types.Inference{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Inferer:     worker4Addr.String(),
			Value:       alloraMath.MustNewDecFromString("0.02318"),
		},
		Forecast: &types.Forecast{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Forecaster:  worker4Addr.String(),
			ForecastElements: []*types.ForecastElement{
				{
					Inferer: s.addrs[9].String(),
					Value:   alloraMath.MustNewDecFromString("0.02318"),
				},
				{
					Inferer: s.addrs[0].String(),
					Value:   alloraMath.MustNewDecFromString("0.02318"),
				},
			},
		},
	}
	worker4Sig, err := GenerateWorkerSignature(s, worker4InferenceForecastBundle, worker4Addr)
	s.Require().NoError(err)
	worker4Bundle := &types.WorkerDataBundle{
		Worker:                             worker4Addr.String(),
		InferenceForecastsBundle:           worker4InferenceForecastBundle,
		InferencesForecastsBundleSignature: worker4Sig,
		Pubkey:                             GetAccPubKey(s, worker4Addr),
	}
	inferences = append(inferences, worker4Bundle)
	// inference and forecast data - worker 5
	worker5InferenceForecastBundle := &types.InferenceForecastBundle{
		Inference: &types.Inference{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Inferer:     worker5Addr.String(),
			Value:       alloraMath.MustNewDecFromString("0.01251"),
		},
		Forecast: &types.Forecast{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Forecaster:  worker5Addr.String(),
			ForecastElements: []*types.ForecastElement{
				{
					Inferer: s.addrs[0].String(),
					Value:   alloraMath.MustNewDecFromString("0.01251"),
				},
				{
					Inferer: s.addrs[1].String(),
					Value:   alloraMath.MustNewDecFromString("0.01251"),
				},
			},
		},
	}
	worker5Sig, err := GenerateWorkerSignature(s, worker5InferenceForecastBundle, worker5Addr)
	s.Require().NoError(err)
	worker5Bundle := &types.WorkerDataBundle{
		Worker:                             worker5Addr.String(),
		InferenceForecastsBundle:           worker5InferenceForecastBundle,
		InferencesForecastsBundleSignature: worker5Sig,
		Pubkey:                             GetAccPubKey(s, worker5Addr),
	}
	inferences = append(inferences, worker5Bundle)

	return inferences
}

func GenerateMoreInferencesDataBundles(s *RewardsTestSuite, blockHeight int64, topicId uint64) []*types.WorkerDataBundle {
	var newInferences []*types.WorkerDataBundle
	oldForecaster := s.addrs[5]
	worker1Addr := s.addrs[10]
	worker2Addr := s.addrs[11]

	worker1InferenceForecastBundle := &types.InferenceForecastBundle{
		Inference: &types.Inference{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Inferer:     worker1Addr.String(),
			Value:       alloraMath.MustNewDecFromString("0.01251"),
		},
		Forecast: &types.Forecast{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Forecaster:  oldForecaster.String(),
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
		},
	}
	worker1Sig, err := GenerateWorkerSignature(s, worker1InferenceForecastBundle, worker1Addr)
	s.Require().NoError(err)
	worker1Bundle := &types.WorkerDataBundle{
		Worker:                             worker1Addr.String(),
		InferenceForecastsBundle:           worker1InferenceForecastBundle,
		InferencesForecastsBundleSignature: worker1Sig,
		Pubkey:                             GetAccPubKey(s, worker1Addr),
	}
	newInferences = append(newInferences, worker1Bundle)

	worker2InferenceForecastBundle := &types.InferenceForecastBundle{
		Inference: &types.Inference{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Inferer:     worker2Addr.String(),
			Value:       alloraMath.MustNewDecFromString("0.01251"),
		},
		Forecast: &types.Forecast{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Forecaster:  oldForecaster.String(),
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
		},
	}
	worker2Sig, err := GenerateWorkerSignature(s, worker2InferenceForecastBundle, worker2Addr)
	s.Require().NoError(err)
	worker2Bundle := &types.WorkerDataBundle{
		Worker:                             worker2Addr.String(),
		InferenceForecastsBundle:           worker2InferenceForecastBundle,
		InferencesForecastsBundleSignature: worker2Sig,
		Pubkey:                             GetAccPubKey(s, worker2Addr),
	}
	newInferences = append(newInferences, worker2Bundle)

	return newInferences
}

func GenerateMoreForecastersDataBundles(s *RewardsTestSuite, blockHeight int64, topicId uint64) []*types.WorkerDataBundle {
	var newForecasts []*types.WorkerDataBundle
	oldInferencer1 := s.addrs[5]
	oldInferencer2 := s.addrs[6]
	worker1Addr := s.addrs[10]
	worker2Addr := s.addrs[11]

	worker1InferenceForecastBundle := &types.InferenceForecastBundle{
		Inference: &types.Inference{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Inferer:     oldInferencer1.String(),
			Value:       alloraMath.MustNewDecFromString("0.01251"),
		},
		Forecast: &types.Forecast{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Forecaster:  worker1Addr.String(),
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
		},
	}
	worker1Sig, err := GenerateWorkerSignature(s, worker1InferenceForecastBundle, oldInferencer1)
	s.Require().NoError(err)
	worker1Bundle := &types.WorkerDataBundle{
		Worker:                             oldInferencer1.String(),
		InferenceForecastsBundle:           worker1InferenceForecastBundle,
		InferencesForecastsBundleSignature: worker1Sig,
		Pubkey:                             GetAccPubKey(s, oldInferencer1),
	}
	newForecasts = append(newForecasts, worker1Bundle)

	worker2InferenceForecastBundle := &types.InferenceForecastBundle{
		Inference: &types.Inference{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Inferer:     oldInferencer2.String(),
			Value:       alloraMath.MustNewDecFromString("0.01251"),
		},
		Forecast: &types.Forecast{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Forecaster:  worker2Addr.String(),
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
		},
	}
	worker2Sig, err := GenerateWorkerSignature(s, worker2InferenceForecastBundle, oldInferencer2)
	s.Require().NoError(err)
	worker2Bundle := &types.WorkerDataBundle{
		Worker:                             oldInferencer2.String(),
		InferenceForecastsBundle:           worker2InferenceForecastBundle,
		InferencesForecastsBundleSignature: worker2Sig,
		Pubkey:                             GetAccPubKey(s, oldInferencer2),
	}
	newForecasts = append(newForecasts, worker2Bundle)

	return newForecasts
}

type TestWorkerValue struct {
	Address sdk.AccAddress
	Value   string
}

func GenerateSimpleWorkerDataBundles(
	s *RewardsTestSuite,
	topicId uint64,
	blockHeight int64,
	workerValues []TestWorkerValue,
	infererAddrs []sdk.AccAddress,
) []*types.WorkerDataBundle {
	require := s.Require()
	if len(workerValues) < 2 {
		require.Fail("workerValues must have at least 2 elements")
	}
	if len(infererAddrs) < 2 {
		require.Fail("infererAddrs must have at least 2 elements")
	}

	var inferences []*types.WorkerDataBundle

	infererIndex := 0

	getInfererIndex := func() int {
		if infererIndex >= len(infererAddrs) {
			infererIndex = 0
		}
		currentInfererIndex := infererIndex
		infererIndex++
		return currentInfererIndex
	}

	for i, workerValue := range workerValues {
		newWorkerInferenceForecastBundle := &types.InferenceForecastBundle{
			Inference: &types.Inference{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Inferer:     workerValue.Address.String(),
				Value:       alloraMath.MustNewDecFromString(workerValues[i].Value),
			},
			Forecast: &types.Forecast{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Forecaster:  workerValue.Address.String(),
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: infererAddrs[getInfererIndex()].String(),
						Value:   alloraMath.MustNewDecFromString(workerValues[i].Value),
					},
					{
						Inferer: infererAddrs[getInfererIndex()].String(),
						Value:   alloraMath.MustNewDecFromString(workerValues[i].Value),
					},
				},
			},
		}
		workerSig, err := GenerateWorkerSignature(s, newWorkerInferenceForecastBundle, workerValue.Address)
		s.Require().NoError(err)
		workerBundle := &types.WorkerDataBundle{
			Worker:                             workerValue.Address.String(),
			InferenceForecastsBundle:           newWorkerInferenceForecastBundle,
			InferencesForecastsBundleSignature: workerSig,
			Pubkey:                             GetAccPubKey(s, workerValue.Address),
		}
		inferences = append(inferences, workerBundle)
	}

	return inferences
}

func GenerateSimpleLossBundles(
	s *RewardsTestSuite,
	topicId uint64,
	blockHeight int64,
	workerValues []TestWorkerValue,
	reputerValues []TestWorkerValue,
	workerZeroAddress sdk.AccAddress,
	workerZeroOneOutInfererValue string,
	workerZeroInfererValue string,
) types.ReputerValueBundles {

	var reputerValueBundles types.ReputerValueBundles
	for _, reputer := range reputerValues {

		var countValues int
		if len(workerValues) < len(reputerValues) {
			countValues = len(workerValues)
		} else {
			countValues = len(reputerValues)
		}

		valueBundle := &types.ValueBundle{
			TopicId: topicId,
			ReputerRequestNonce: &types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{
					BlockHeight: blockHeight,
				},
			},
			Reputer:                reputer.Address.String(),
			CombinedValue:          alloraMath.MustNewDecFromString(reputer.Value),
			NaiveValue:             alloraMath.MustNewDecFromString(reputer.Value),
			InfererValues:          make([]*types.WorkerAttributedValue, countValues),
			ForecasterValues:       make([]*types.WorkerAttributedValue, countValues),
			OneOutInfererValues:    make([]*types.WithheldWorkerAttributedValue, countValues),
			OneOutForecasterValues: make([]*types.WithheldWorkerAttributedValue, countValues),
			OneInForecasterValues:  make([]*types.WorkerAttributedValue, countValues),
		}

		for j, worker := range workerValues {
			if j < len(reputerValues) {
				if worker.Address.Equals(workerZeroAddress) {
					valueBundle.InfererValues[j] = &types.WorkerAttributedValue{Worker: worker.Address.String(), Value: alloraMath.MustNewDecFromString(workerZeroInfererValue)}
				} else {
					valueBundle.InfererValues[j] = &types.WorkerAttributedValue{Worker: worker.Address.String(), Value: alloraMath.MustNewDecFromString(reputerValues[j].Value)}
				}
				valueBundle.ForecasterValues[j] = &types.WorkerAttributedValue{Worker: worker.Address.String(), Value: alloraMath.MustNewDecFromString(reputerValues[j].Value)}
				if worker.Address.Equals(workerZeroAddress) {
					valueBundle.OneOutInfererValues[j] = &types.WithheldWorkerAttributedValue{Worker: worker.Address.String(), Value: alloraMath.MustNewDecFromString(workerZeroOneOutInfererValue)}
				} else {
					valueBundle.OneOutInfererValues[j] = &types.WithheldWorkerAttributedValue{Worker: worker.Address.String(), Value: alloraMath.MustNewDecFromString(reputerValues[j].Value)}
				}
				valueBundle.OneOutForecasterValues[j] = &types.WithheldWorkerAttributedValue{Worker: worker.Address.String(), Value: alloraMath.MustNewDecFromString(reputerValues[j].Value)}
				valueBundle.OneInForecasterValues[j] = &types.WorkerAttributedValue{Worker: worker.Address.String(), Value: alloraMath.MustNewDecFromString(reputerValues[j].Value)}
			}
		}

		sig, err := GenerateReputerSignature(s, valueBundle, reputer.Address)
		s.Require().NoError(err)

		bundle := &types.ReputerValueBundle{
			Pubkey:      GetAccPubKey(s, reputer.Address),
			Signature:   sig,
			ValueBundle: valueBundle,
		}
		reputerValueBundles.ReputerValueBundles = append(reputerValueBundles.ReputerValueBundles, bundle)
	}

	return reputerValueBundles
}
