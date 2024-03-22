package module_test

import (
	"time"

	cosmosMath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/allora-network/allora-chain/x/emissions/module"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *ModuleTestSuite) TestGetReputerScore() {
	// Mock data with 2 reputers reporting loss of 2 workers
	// [0] - Reputer 1
	// [1] - Reputer 2
	reputersReportedCombinedLosses := []float64{85, 80}
	reputersReportedNaiveLosses := []float64{100, 90}
	reputersWorker1ReportedInferenceLosses := []float64{90, 90}
	reputersWorker2ReportedInferenceLosses := []float64{100, 100}
	reputersWorker1ReportedForecastLosses := []float64{90, 85}
	reputersWorker2ReportedForecastLosses := []float64{100, 100}
	reputersWorker1ReportedOneOutLosses := []float64{115, 120}
	reputersWorker2ReportedOneOutLosses := []float64{100, 100}
	reputersWorker1ReportedOneInNaiveLosses := []float64{90, 85}
	reputersWorker2ReportedOneInNaiveLosses := []float64{100, 100}

	// Matrix for reported losses
	reportedLossesMatrix := [][]float64{
		reputersReportedCombinedLosses,
		reputersReportedNaiveLosses,
		reputersWorker1ReportedInferenceLosses,
		reputersWorker2ReportedInferenceLosses,
		reputersWorker1ReportedForecastLosses,
		reputersWorker2ReportedForecastLosses,
		reputersWorker1ReportedOneOutLosses,
		reputersWorker2ReportedOneOutLosses,
		reputersWorker1ReportedOneInNaiveLosses,
		reputersWorker2ReportedOneInNaiveLosses,
	}

	allReputersStakes := []float64{50, 150}

	// TODO: Implement get listening coefficients function
	// Get listening coefficients
	listeningCoefficient := 0.18
	allListeningCoefficients := []float64{listeningCoefficient, 0.63}

	// Get adjusted stakes
	var adjustedStakes []float64
	for _, reputerStake := range allReputersStakes {
		adjustedStake, err := module.GetAdjustedStake(reputerStake, allReputersStakes, listeningCoefficient, allListeningCoefficients, float64(2))
		s.NoError(err, "Error getting adjustedStake")
		adjustedStakes = append(adjustedStakes, adjustedStake)
	}

	// Get consensus loss vector
	consensus, err := module.GetStakeWeightedLossMatrix(adjustedStakes, reportedLossesMatrix)
	s.NoError(err, "Error getting consensus")

	// Get reputer scores
	reputer1AllReportedLosses := []float64{
		reputersReportedCombinedLosses[0],
		reputersReportedNaiveLosses[0],
		reputersWorker1ReportedInferenceLosses[0],
		reputersWorker2ReportedInferenceLosses[0],
		reputersWorker1ReportedForecastLosses[0],
		reputersWorker2ReportedForecastLosses[0],
		reputersWorker1ReportedOneOutLosses[0],
		reputersWorker2ReportedOneOutLosses[0],
		reputersWorker1ReportedOneInNaiveLosses[0],
		reputersWorker2ReportedOneInNaiveLosses[0],
	}
	reputer1Score, err := module.GetConsensusScore(reputer1AllReportedLosses, consensus)
	s.NoError(err, "Error getting reputer1Score")
	s.NotEqual(0, reputer1Score, "Expected reputer1Score to be non-zero")

	reputer2AllReportedLosses := []float64{
		reputersReportedCombinedLosses[1],
		reputersReportedNaiveLosses[1],
		reputersWorker1ReportedInferenceLosses[1],
		reputersWorker2ReportedInferenceLosses[1],
		reputersWorker1ReportedForecastLosses[1],
		reputersWorker2ReportedForecastLosses[1],
		reputersWorker1ReportedOneOutLosses[1],
		reputersWorker2ReportedOneOutLosses[1],
		reputersWorker1ReportedOneInNaiveLosses[1],
		reputersWorker2ReportedOneInNaiveLosses[1],
	}
	
	reputer2Score, err := module.GetConsensusScore(reputer2AllReportedLosses, consensus)
	s.NoError(err, "Error getting reputer2Score")
	s.NotEqual(0, reputer2Score, "Expected reputer2Score to be non-zero")
}

func (s *ModuleTestSuite) TestGetWorkerScoreForecastTask() {
	timeNow := uint64(time.Now().UTC().Unix())

	// Create a topic
	topicIds, err := mockCreateTopics(s, 1)
	s.NoError(err, "Error creating topic")
	topicId := topicIds[0]

	// Create and register 2 reputers in topic
	reputers, err := mockSomeReputers(s, topicId)
	s.NoError(err, "Error creating reputers")

	// Create and register 2 workers in topic
	workers, err := mockSomeWorkers(s, topicId)
	s.NoError(err, "Error creating workers")

	// Add a lossBundle for each reputer
	var reputersLossBundles []*types.LossBundle
	reputer1LossBundle := types.LossBundle{
		TopicId: topicId,
		Reputer: reputers[0].String(),
		CombinedLoss: cosmosMath.NewUint(85),
		ForecasterLosses: []*types.WorkerAttributedLoss{
			{
				Worker: workers[0].String(),
				Value: cosmosMath.NewUint(90),
			},
			{
				Worker: workers[1].String(),
				Value: cosmosMath.NewUint(100),
			},
		},
		NaiveLoss: cosmosMath.NewUint(100),
		// Increased loss when removing for worker 1
		OneOutLosses: []*types.WorkerAttributedLoss{
			{
				Worker: workers[0].String(),
				Value: cosmosMath.NewUint(115),
			},
			{
				Worker: workers[1].String(),
				Value: cosmosMath.NewUint(100),
			},
		},
		OneInNaiveLosses: []*types.WorkerAttributedLoss{
			{
				Worker: workers[0].String(),
				Value: cosmosMath.NewUint(90),
			},
			{
				Worker: workers[1].String(),
				Value: cosmosMath.NewUint(100),
			},
		},
	}
	reputersLossBundles = append(reputersLossBundles, &reputer1LossBundle)
	reputer2LossBundle := types.LossBundle{
		TopicId: topicId,
		Reputer: reputers[1].String(),
		CombinedLoss: cosmosMath.NewUint(80),
		ForecasterLosses: []*types.WorkerAttributedLoss{
			{
				Worker: workers[0].String(),
				Value: cosmosMath.NewUint(85),
			},
			{
				Worker: workers[1].String(),
				Value: cosmosMath.NewUint(100),
			},
		},
		NaiveLoss: cosmosMath.NewUint(90),
		// Increased loss when removing for worker 1
		OneOutLosses: []*types.WorkerAttributedLoss{
			{
				Worker: workers[0].String(),
				Value: cosmosMath.NewUint(120),
			},
			{
				Worker: workers[1].String(),
				Value: cosmosMath.NewUint(100),
			},
		},
		OneInNaiveLosses: []*types.WorkerAttributedLoss{
			{
				Worker: workers[0].String(),
				Value: cosmosMath.NewUint(85),
			},
			{
				Worker: workers[1].String(),
				Value: cosmosMath.NewUint(100),
			},
		},
	}
	reputersLossBundles = append(reputersLossBundles, &reputer2LossBundle)

	err = s.emissionsKeeper.InsertLossBundles(s.ctx, topicId, timeNow, types.LossBundles{LossBundles: reputersLossBundles})
	s.NoError(err, "Error adding lossBundles")

	// Get LossBundles
	lossBundles, err := s.emissionsKeeper.GetLossBundles(s.ctx, topicId, timeNow)
	s.NoError(err, "Error getting lossBundles")

	// Get reputers stakes and reported losses for each worker
	var reputersStakes []float64
	var reputersReportedCombinedLosses []float64
	var reputersNaiveReportedLosses []float64

	var reputersWorker1ReportedOneOutLosses []float64
	var reputersWorker2ReportedOneOutLosses []float64

	var reputersWorker1ReportedOneInNaiveLosses []float64
	var reputersWorker2ReportedOneInNaiveLosses []float64

	for _, lossBundle := range lossBundles.LossBundles {
		reputerAddr, err := sdk.AccAddressFromBech32(lossBundle.Reputer)
		s.NoError(err, "Error getting reputerAddr")

		reputerStake, err := s.emissionsKeeper.GetStakeOnTopicFromReputer(s.ctx, topicId, reputerAddr)
		s.NoError(err, "Error getting reputerStake")

		reputerStakeFloat := float64(reputerStake.BigInt().Int64())
		reputersStakes = append(reputersStakes, reputerStakeFloat)
		reputersReportedCombinedLosses = append(reputersReportedCombinedLosses, float64(lossBundle.CombinedLoss.BigInt().Int64()))
		reputersNaiveReportedLosses = append(reputersNaiveReportedLosses, float64(lossBundle.NaiveLoss.BigInt().Int64()))

		// Add OneOutLosses
		for _, workerLoss := range lossBundle.OneOutLosses {
			if workerLoss.Worker == workers[0].String() {
				reputersWorker1ReportedOneOutLosses = append(reputersWorker1ReportedOneOutLosses, float64(workerLoss.Value.BigInt().Int64()))
			} else if workerLoss.Worker == workers[1].String() {
				reputersWorker2ReportedOneOutLosses = append(reputersWorker2ReportedOneOutLosses, float64(workerLoss.Value.BigInt().Int64()))
			}
		}

		// Add OneInNaiveLosses
		for _, workerLoss := range lossBundle.OneInNaiveLosses {
			if workerLoss.Worker == workers[0].String() {
				reputersWorker1ReportedOneInNaiveLosses = append(reputersWorker1ReportedOneInNaiveLosses, float64(workerLoss.Value.BigInt().Int64()))
			} else if workerLoss.Worker == workers[1].String() {
				reputersWorker2ReportedOneInNaiveLosses = append(reputersWorker2ReportedOneInNaiveLosses, float64(workerLoss.Value.BigInt().Int64()))
			}
		}
	}

	// Get Stake Weighted Loss - Network Inference Loss - (L_i)
	networkStakeWeightedLoss, err := module.GetStakeWeightedLoss(reputersStakes, reputersReportedCombinedLosses)
	s.NoError(err, "Error getting stakeWeightedLoss")
	s.NotEqual(0, networkStakeWeightedLoss, "Expected worker1StakeWeightedLoss to be non-zero")

	// Get Stake Weighted Loss - Naive Loss - (L^-_i)
	networkStakeWeightedNaiveLoss, err := module.GetStakeWeightedLoss(reputersStakes, reputersNaiveReportedLosses)
	s.NoError(err, "Error getting stakeWeightedLoss")
	s.NotEqual(0, networkStakeWeightedNaiveLoss, "Expected worker1StakeWeightedNaiveLoss to be non-zero")

	// Get Stake Weighted Loss - OneOut Loss - (L^-_i)
	worker1StakeWeightedOneOutLoss, err := module.GetStakeWeightedLoss(reputersStakes, reputersWorker1ReportedOneOutLosses)
	s.NoError(err, "Error getting stakeWeightedLoss")
	s.NotEqual(0, worker1StakeWeightedOneOutLoss, "Expected worker1StakeWeightedOneOutLoss to be non-zero")

	worker2StakeWeightedOneOutLoss, err := module.GetStakeWeightedLoss(reputersStakes, reputersWorker2ReportedOneOutLosses)
	s.NoError(err, "Error getting stakeWeightedLoss")
	s.NotEqual(0, worker2StakeWeightedOneOutLoss, "Expected worker2StakeWeightedOneOutLoss to be non-zero")

	// Get Stake Weighted Loss - OneInNaive Loss - (L^+_ki)
	worker1StakeWeightedOneInNaiveLoss, err := module.GetStakeWeightedLoss(reputersStakes, reputersWorker1ReportedOneInNaiveLosses)
	s.NoError(err, "Error getting stakeWeightedLoss")
	s.NotEqual(0, worker1StakeWeightedOneInNaiveLoss, "Expected worker1StakeWeightedOneInNaiveLoss to be non-zero")

	worker2StakeWeightedOneInNaiveLoss, err := module.GetStakeWeightedLoss(reputersStakes, reputersWorker2ReportedOneInNaiveLosses)
	s.NoError(err, "Error getting stakeWeightedLoss")
	s.NotEqual(0, worker2StakeWeightedOneInNaiveLoss, "Expected worker2StakeWeightedOneInNaiveLoss to be non-zero")

	// Get Worker Score - OneOut Score (L^-_ki - L_i)
	worker1OneOutScore := module.GetWorkerScore(networkStakeWeightedLoss, worker1StakeWeightedOneOutLoss)
	s.NotEqual(0, worker1OneOutScore, "Expected worker1Score to be non-zero")

	worker2OneOutScore := module.GetWorkerScore(networkStakeWeightedLoss, worker2StakeWeightedOneOutLoss)
	s.NotEqual(0, worker2OneOutScore, "Expected worker2Score to be non-zero")

	// Get Worker Score - OneIn Score (L^-_i - L^+_ki)
	worker1ScoreOneInNaive := module.GetWorkerScore(networkStakeWeightedNaiveLoss, worker1StakeWeightedOneInNaiveLoss)
	s.NotEqual(0, worker1ScoreOneInNaive, "Expected worker1ScoreOneInNaive to be non-zero")

	worker2ScoreOneInNaive := module.GetWorkerScore(networkStakeWeightedNaiveLoss, worker2StakeWeightedOneInNaiveLoss)
	s.NotEqual(0, worker2ScoreOneInNaive, "Expected worker2ScoreOneInNaive to be non-zero")

	// Get Worker 1 Final Score - Forecast Task (T_ik)
	worker1FinalScore := module.GetFinalWorkerScoreForecastTask(worker1ScoreOneInNaive, worker1OneOutScore, module.GetfUniqueAgg(2))
	s.NotEqual(0, worker1FinalScore, "Expected worker1FinalScore to be non-zero")

	// Get Worker 2 Final Score - Forecast Task (T_ik)
	worker2FinalScore := module.GetFinalWorkerScoreForecastTask(worker2ScoreOneInNaive, worker2OneOutScore, module.GetfUniqueAgg(2))
	s.NotEqual(0, worker2FinalScore, "Expected worker2FinalScore to be non-zero")
}

func (s *ModuleTestSuite) TestGetWorkerScoreInferenceTask() {
	timeNow := uint64(time.Now().UTC().Unix())

	// Create a topic
	topicIds, err := mockCreateTopics(s, 1)
	s.NoError(err, "Error creating topic")
	topicId := topicIds[0]

	// Create and register 2 reputers in topic
	reputers, err := mockSomeReputers(s, topicId)
	s.NoError(err, "Error creating reputers")

	// Create and register 2 workers in topic
	workers, err := mockSomeWorkers(s, topicId)
	s.NoError(err, "Error creating workers")

	// Add a lossBundle for each reputer
	var reputersLossBundles []*types.LossBundle
	reputer1LossBundle := types.LossBundle{
		TopicId: topicId,
		Reputer: reputers[0].String(),
		CombinedLoss: cosmosMath.NewUint(85),
		// Increased loss when removing for worker 1
		OneOutLosses: []*types.WorkerAttributedLoss{
			{
				Worker: workers[0].String(),
				Value: cosmosMath.NewUint(115),
			},
			{
				Worker: workers[1].String(),
				Value: cosmosMath.NewUint(100),
			},
		},
	}
	reputersLossBundles = append(reputersLossBundles, &reputer1LossBundle)
	reputer2LossBundle := types.LossBundle{
		TopicId: topicId,
		Reputer: reputers[1].String(),
		CombinedLoss: cosmosMath.NewUint(80),
		// Increased loss when removing for worker 1
		OneOutLosses: []*types.WorkerAttributedLoss{
			{
				Worker: workers[0].String(),
				Value: cosmosMath.NewUint(120),
			},
			{
				Worker: workers[1].String(),
				Value: cosmosMath.NewUint(100),
			},
		},
	}
	reputersLossBundles = append(reputersLossBundles, &reputer2LossBundle)

	err = s.emissionsKeeper.InsertLossBundles(s.ctx, topicId, timeNow, types.LossBundles{LossBundles: reputersLossBundles})
	s.NoError(err, "Error adding lossBundle for worker")

	// Get LossBundles
	lossBundles, err := s.emissionsKeeper.GetLossBundles(s.ctx, topicId, timeNow)
	s.NoError(err, "Error getting lossBundles")

	// Get reputers stakes and reported losses for each worker
	var reputersStakes []float64
	var reputersCombinedReportedLosses []float64

	var reputersWorker1ReportedOneOutLosses []float64
	var reputersWorker2ReportedOneOutLosses []float64

	for _, lossBundle := range lossBundles.LossBundles {
		reputerAddr, err := sdk.AccAddressFromBech32(lossBundle.Reputer)
		s.NoError(err, "Error getting reputerAddr")

		reputerStake, err := s.emissionsKeeper.GetStakeOnTopicFromReputer(s.ctx, topicId, reputerAddr)
		s.NoError(err, "Error getting reputerStake")

		reputerStakeFloat := float64(reputerStake.BigInt().Int64())
		reputersStakes = append(reputersStakes, reputerStakeFloat)
		reputersCombinedReportedLosses = append(reputersCombinedReportedLosses, float64(lossBundle.CombinedLoss.BigInt().Int64()))

		// Add OneOutLosses
		for _, workerLoss := range lossBundle.OneOutLosses {
			if workerLoss.Worker == workers[0].String() {
				reputersWorker1ReportedOneOutLosses = append(reputersWorker1ReportedOneOutLosses, float64(workerLoss.Value.BigInt().Int64()))
			} else if workerLoss.Worker == workers[1].String() {
				reputersWorker2ReportedOneOutLosses = append(reputersWorker2ReportedOneOutLosses, float64(workerLoss.Value.BigInt().Int64()))
			}
		}
	}

	// Get Stake Weighted Loss - Network Inference Loss - (L_i)
	networkStakeWeightedLoss, err := module.GetStakeWeightedLoss(reputersStakes, reputersCombinedReportedLosses)
	s.NoError(err, "Error getting stakeWeightedLoss")
	s.NotEqual(0, networkStakeWeightedLoss, "Expected worker1StakeWeightedLoss to be non-zero")

	// Get Stake Weighted Loss - OneOut Loss - (L^-_ji)
	worker1StakeWeightedOneOutLoss, err := module.GetStakeWeightedLoss(reputersStakes, reputersWorker1ReportedOneOutLosses)
	s.NoError(err, "Error getting stakeWeightedLoss")
	s.NotEqual(0, worker1StakeWeightedOneOutLoss, "Expected worker1StakeWeightedOneOutLoss to be non-zero")

	worker2StakeWeightedOneOutLoss, err := module.GetStakeWeightedLoss(reputersStakes, reputersWorker2ReportedOneOutLosses)
	s.NoError(err, "Error getting stakeWeightedLoss")
	s.NotEqual(0, worker2StakeWeightedOneOutLoss, "Expected worker2StakeWeightedOneOutLoss to be non-zero")

	// Get Worker Score - OneOut Score - (Tij)
	worker1Score := module.GetWorkerScore(networkStakeWeightedLoss, worker1StakeWeightedOneOutLoss)
	s.NotEqual(0, worker1Score, "Expected worker1Score to be non-zero")

	worker2Score := module.GetWorkerScore(networkStakeWeightedLoss, worker2StakeWeightedOneOutLoss)
	s.NotEqual(0, worker2Score, "Expected worker2Score to be non-zero")
}

func (s *ModuleTestSuite) TestGetStakeWeightedLoss() {
	timeNow := uint64(time.Now().UTC().Unix())

	// Create a topic
	topicIds, err := mockCreateTopics(s, 1)
	s.NoError(err, "Error creating topic")
	topicId := topicIds[0]

	// Create and register 2 reputers in topic
	reputers, err := mockSomeReputers(s, topicId)
	s.NoError(err, "Error creating reputers")

	// Add a lossBundle for each reputer
	losses := []cosmosMath.Uint{cosmosMath.NewUint(150), cosmosMath.NewUint(250)}

	var newLossBundles []*types.LossBundle
	for i, reputer := range reputers {
		lossBundle := types.LossBundle{
			Reputer:      reputer.String(),
			CombinedLoss: losses[i],
		}
		newLossBundles = append(newLossBundles, &lossBundle)
	}

	err = s.emissionsKeeper.InsertLossBundles(s.ctx, topicId, timeNow, types.LossBundles{LossBundles: newLossBundles})
	s.NoError(err, "Error adding lossBundle for reputer")

	var reputersStakes []float64
	var reputersReportedLosses []float64

	// Get LossBundles
	lossBundles, err := s.emissionsKeeper.GetLossBundles(s.ctx, topicId, timeNow)
	s.NoError(err, "Error getting lossBundles")

	// Get stakes and reported losses
	for _, lossBundle := range lossBundles.LossBundles {
		reputerAddr, err := sdk.AccAddressFromBech32(lossBundle.Reputer)
		s.NoError(err, "Error getting reputerAddr")

		reputerStake, err := s.emissionsKeeper.GetStakeOnTopicFromReputer(s.ctx, topicId, reputerAddr)
		s.NoError(err, "Error getting reputerStake")

		reputerStakeFloat, _ := reputerStake.BigInt().Float64()
		reputersStakes = append(reputersStakes, reputerStakeFloat)
		reputersReportedLosses = append(reputersReportedLosses, float64(lossBundle.CombinedLoss.BigInt().Int64()))
	}

	expectedStakeWeightedLoss, err := module.GetStakeWeightedLoss(reputersStakes, reputersReportedLosses)
	s.NoError(err, "Error getting stakeWeightedLoss")
	s.NotEqual(0, expectedStakeWeightedLoss, "Expected stakeWeightedLoss to be non-zero")
}
