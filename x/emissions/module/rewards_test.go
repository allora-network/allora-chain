package module_test

import (
	"fmt"

	cosmosMath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/allora-network/allora-chain/app/params"
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

	allReputersStakes := []float64{50, 150}

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
	consensus, err := module.GetStakeWeightedLossMatrix(adjustedStakes, [][]float64{reputer1AllReportedLosses, reputer2AllReportedLosses})
	s.NoError(err, "Error getting consensus")

	// Get reputer scores
	reputer1Score, err := module.GetConsensusScore(reputer1AllReportedLosses, consensus)
	s.NoError(err, "Error getting reputer1Score")
	s.NotEqual(0, reputer1Score, "Expected reputer1Score to be non-zero")

	reputer2Score, err := module.GetConsensusScore(reputer2AllReportedLosses, consensus)
	s.NoError(err, "Error getting reputer2Score")
	s.NotEqual(0, reputer2Score, "Expected reputer2Score to be non-zero")
}

func (s *ModuleTestSuite) TestGetWorkerScoreForecastTask() {

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

	// Add a valueBundle for each reputer
	var reputersValueBundles []*types.ReputerValueBundle
	reputer1ValueBundle := types.ReputerValueBundle{
		Reputer: reputers[0].String(),
		ValueBundle: &types.ValueBundle{
			TopicId:       topicId,
			CombinedValue: 85,
			ForecasterValues: []*types.WorkerAttributedValue{
				{
					Worker: workers[0].String(),
					Value:  90,
				},
				{
					Worker: workers[1].String(),
					Value:  100,
				},
			},
			NaiveValue: 100,
			// Increased loss when removing for worker 1
			OneOutValues: []*types.WorkerAttributedValue{
				{
					Worker: workers[0].String(),
					Value:  115,
				},
				{
					Worker: workers[1].String(),
					Value:  100,
				},
			},
			OneInNaiveValues: []*types.WorkerAttributedValue{
				{
					Worker: workers[0].String(),
					Value:  90,
				},
				{
					Worker: workers[1].String(),
					Value:  100,
				},
			},
		},
	}
	reputersValueBundles = append(reputersValueBundles, &reputer1ValueBundle)
	reputer2ValueBundle := types.ReputerValueBundle{
		Reputer: reputers[1].String(),
		ValueBundle: &types.ValueBundle{
			TopicId:       topicId,
			CombinedValue: 80,
			ForecasterValues: []*types.WorkerAttributedValue{
				{
					Worker: workers[0].String(),
					Value:  85,
				},
				{
					Worker: workers[1].String(),
					Value:  100,
				},
			},
			NaiveValue: 90,
			// Increased loss when removing for worker 1
			OneOutValues: []*types.WorkerAttributedValue{
				{
					Worker: workers[0].String(),
					Value:  120,
				},
				{
					Worker: workers[1].String(),
					Value:  100,
				},
			},
			OneInNaiveValues: []*types.WorkerAttributedValue{
				{
					Worker: workers[0].String(),
					Value:  85,
				},
				{
					Worker: workers[1].String(),
					Value:  100,
				},
			},
		},
	}
	reputersValueBundles = append(reputersValueBundles, &reputer2ValueBundle)
	timeNow := s.ctx.BlockHeight()

	err = s.emissionsKeeper.InsertValueBundles(s.ctx, topicId, timeNow, types.ReputerValueBundles{ReputerValueBundles: reputersValueBundles})
	s.NoError(err, "Error adding valueBundles")

	// Get ValueBundles
	valueBundles, err := s.emissionsKeeper.GetValueBundles(s.ctx, topicId, timeNow)
	s.NoError(err, "Error getting valueBundles")

	// Get reputers stakes and reported losses for each worker
	var reputersStakes []float64
	var reputersReportedCombinedLosses []float64
	var reputersNaiveReportedLosses []float64

	var reputersWorker1ReportedOneOutLosses []float64
	var reputersWorker2ReportedOneOutLosses []float64

	var reputersWorker1ReportedOneInNaiveLosses []float64
	var reputersWorker2ReportedOneInNaiveLosses []float64

	for _, valueBundle := range valueBundles.ReputerValueBundles {
		reputerAddr, err := sdk.AccAddressFromBech32(valueBundle.Reputer)
		s.NoError(err, "Error getting reputerAddr")

		reputerStake, err := s.emissionsKeeper.GetStakeOnTopicFromReputer(s.ctx, topicId, reputerAddr)
		s.NoError(err, "Error getting reputerStake")

		reputerStakeFloat := float64(reputerStake.BigInt().Int64())
		reputersStakes = append(reputersStakes, reputerStakeFloat)
		reputersReportedCombinedLosses = append(reputersReportedCombinedLosses, valueBundle.ValueBundle.CombinedValue)
		reputersNaiveReportedLosses = append(reputersNaiveReportedLosses, valueBundle.ValueBundle.NaiveValue)

		// Add OneOutLosses
		for _, workerLoss := range valueBundle.ValueBundle.OneOutValues {
			if workerLoss.Worker == workers[0].String() {
				reputersWorker1ReportedOneOutLosses = append(reputersWorker1ReportedOneOutLosses, workerLoss.Value)
			} else if workerLoss.Worker == workers[1].String() {
				reputersWorker2ReportedOneOutLosses = append(reputersWorker2ReportedOneOutLosses, workerLoss.Value)
			}
		}

		// Add OneInNaiveLosses
		for _, workerLoss := range valueBundle.ValueBundle.OneInNaiveValues {
			if workerLoss.Worker == workers[0].String() {
				reputersWorker1ReportedOneInNaiveLosses = append(reputersWorker1ReportedOneInNaiveLosses, workerLoss.Value)
			} else if workerLoss.Worker == workers[1].String() {
				reputersWorker2ReportedOneInNaiveLosses = append(reputersWorker2ReportedOneInNaiveLosses, workerLoss.Value)
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

func (s *ModuleTestSuite) TestGetStakeWeightedLoss() {

	// Create a topic
	topicIds, err := mockCreateTopics(s, 1)
	s.NoError(err, "Error creating topic")
	topicId := topicIds[0]
	// Create and register 2 reputers in topic
	reputers, err := mockSomeReputers(s, topicId)
	s.NoError(err, "Error creating reputers")

	// Add a valueBundle for each reputer
	losses := []float64{150, 250}

	var newValueBundles []*types.ReputerValueBundle
	for i, reputer := range reputers {
		valueBundle := types.ReputerValueBundle{
			Reputer: reputer.String(),
			ValueBundle: &types.ValueBundle{
				CombinedValue: losses[i],
			},
		}
		newValueBundles = append(newValueBundles, &valueBundle)
	}

	timeNow := s.ctx.BlockHeight()

	err = s.emissionsKeeper.InsertValueBundles(s.ctx, topicId, timeNow, types.ReputerValueBundles{ReputerValueBundles: newValueBundles})
	s.NoError(err, "Error adding valueBundle for reputer")

	var reputersStakes []float64
	var reputersReportedLosses []float64

	// Get ValueBundles
	valueBundles, err := s.emissionsKeeper.GetValueBundles(s.ctx, topicId, timeNow)
	s.NoError(err, "Error getting valueBundles")

	// Get stakes and reported losses
	for _, valueBundle := range valueBundles.ReputerValueBundles {
		reputerAddr, err := sdk.AccAddressFromBech32(valueBundle.Reputer)
		s.NoError(err, "Error getting reputerAddr")

		reputerStake, err := s.emissionsKeeper.GetStakeOnTopicFromReputer(s.ctx, topicId, reputerAddr)
		s.NoError(err, "Error getting reputerStake")

		reputerStakeFloat, _ := reputerStake.BigInt().Float64()
		reputersStakes = append(reputersStakes, reputerStakeFloat)
		reputersReportedLosses = append(reputersReportedLosses, valueBundle.ValueBundle.CombinedValue)
	}

	expectedStakeWeightedLoss, err := module.GetStakeWeightedLoss(reputersStakes, reputersReportedLosses)
	s.NoError(err, "Error getting stakeWeightedLoss")
	s.NotEqual(0, expectedStakeWeightedLoss, "Expected stakeWeightedLoss to be non-zero")
}

/// HELPER FUNCTIONS

const (
	reputer1StartAmount = 1337
	reputer2StartAmount = 6969
	worker1StartAmount  = 4242
	worker2StartAmount  = 1111
)

// mock mint coins to participants
func mockMintRewardCoins(s *ModuleTestSuite, amount []cosmosMath.Int, target []sdk.AccAddress) error {
	if len(amount) != len(target) {
		return fmt.Errorf("amount and target must be the same length")
	}
	for i, addr := range target {
		coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amount[i]))
		s.bankKeeper.MintCoins(s.ctx, types.AlloraStakingAccountName, coins)
		s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.AlloraStakingAccountName, addr, coins)
	}
	return nil
}

// give some reputers coins, have them stake those coins
func mockSomeReputers(s *ModuleTestSuite, topicId uint64) ([]sdk.AccAddress, error) {
	reputerAddrs := []sdk.AccAddress{
		s.addrs[0],
		s.addrs[1],
	}
	reputerAmounts := []cosmosMath.Int{
		cosmosMath.NewInt(reputer1StartAmount),
		cosmosMath.NewInt(reputer2StartAmount),
	}
	err := mockMintRewardCoins(
		s,
		reputerAmounts,
		reputerAddrs,
	)
	if err != nil {
		return nil, err
	}
	_, err = s.msgServer.Register(s.ctx, &types.MsgRegister{
		Creator:      reputerAddrs[0].String(),
		LibP2PKey:    "libp2pkeyReputer1",
		MultiAddress: "multiaddressReputer1",
		TopicIds:     []uint64{topicId},
		InitialStake: cosmosMath.NewUintFromBigInt(reputerAmounts[0].BigInt()),
		IsReputer:    true,
	})
	if err != nil {
		return nil, err
	}
	_, err = s.msgServer.Register(s.ctx, &types.MsgRegister{
		Creator:      reputerAddrs[1].String(),
		LibP2PKey:    "libp2pkeyReputer2",
		MultiAddress: "multiaddressReputer2",
		TopicIds:     []uint64{topicId},
		InitialStake: cosmosMath.NewUintFromBigInt(reputerAmounts[1].BigInt()),
		IsReputer:    true,
	})
	if err != nil {
		return nil, err
	}
	return reputerAddrs, nil
}

// give some workers coins, have them stake those coins
func mockSomeWorkers(s *ModuleTestSuite, topicId uint64) ([]sdk.AccAddress, error) {
	workerAddrs := []sdk.AccAddress{
		s.addrs[2],
		s.addrs[3],
	}
	workerAmounts := []cosmosMath.Int{
		cosmosMath.NewInt(worker1StartAmount),
		cosmosMath.NewInt(worker2StartAmount),
	}
	err := mockMintRewardCoins(
		s,
		workerAmounts,
		workerAddrs,
	)
	if err != nil {
		return nil, err
	}
	_, err = s.msgServer.Register(s.ctx, &types.MsgRegister{
		Creator:      workerAddrs[0].String(),
		LibP2PKey:    "libp2pkeyWorker1",
		MultiAddress: "multiaddressWorker1",
		TopicIds:     []uint64{topicId},
		InitialStake: cosmosMath.NewUintFromBigInt(workerAmounts[0].BigInt()),
		Owner:        workerAddrs[0].String(),
	})
	if err != nil {
		return nil, err
	}
	_, err = s.msgServer.Register(s.ctx, &types.MsgRegister{
		Creator:      workerAddrs[1].String(),
		LibP2PKey:    "libp2pkeyWorker2",
		MultiAddress: "multiaddressWorker2",
		TopicIds:     []uint64{topicId},
		InitialStake: cosmosMath.NewUintFromBigInt(workerAmounts[1].BigInt()),
		Owner:        workerAddrs[1].String(),
	})
	if err != nil {
		return nil, err
	}
	return workerAddrs, nil
}

// create a topic
func mockCreateTopics(s *ModuleTestSuite, numToCreate uint64) ([]uint64, error) {
	ret := make([]uint64, 0)
	var i uint64
	for i = 0; i < numToCreate; i++ {
		topicMessage := types.MsgCreateNewTopic{
			Creator:          s.addrsStr[0],
			Metadata:         "metadata",
			LossLogic:        "logic",
			LossMethod:       "whatever",
			InferenceLogic:   "morelogic",
			InferenceMethod:  "whatever2",
			EpochLength:      10800,
			DefaultArg:       "default",
			Pnorm:            2,
			AlphaRegret:      "0.1",
			PrewardReputer:   "0.1",
			PrewardInference: "0.1",
			PrewardForecast:  "0.1",
			FTolerance:       "0.1",
		}

		response, err := s.msgServer.CreateNewTopic(s.ctx, &topicMessage)
		if err != nil {
			return nil, err
		}
		ret = append(ret, response.TopicId)
	}
	return ret, nil
}
