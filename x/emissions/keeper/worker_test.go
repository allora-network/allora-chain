package keeper_test

import (
	"fmt"
	"strings"

	cosmosMath "cosmossdk.io/math"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *KeeperTestSuite) TestGetInferencesAtBlock() {
	ctx := s.Ctx()
	k := s.WorkerKeeper()
	topicId := uint64(1)
	block := types.BlockHeight(100)
	expectedInferences := types.Inferences{
		Inferences: []*types.Inference{
			{
				TopicId:     topicId,
				BlockHeight: block,
				Value:       alloraMath.NewDecFromInt64(1), // Assuming NewDecFromInt64 exists and is appropriate
				Inferer:     "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
			},
			{
				TopicId:     topicId,
				BlockHeight: block,
				Value:       alloraMath.NewDecFromInt64(2),
				Inferer:     "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
			},
		},
	}

	// Assume InsertActiveInferences correctly sets up inferences
	nonce := types.Nonce{BlockHeight: block} // Assuming block type cast to int64 if needed
	err := k.InsertActiveInferences(ctx, topicId, nonce.BlockHeight, expectedInferences)
	s.Require().NoError(err)

	// Retrieve inferences
	actualInferences, err := k.GetInferencesAtBlock(ctx, topicId, block, false)
	s.Require().NoError(err)
	s.Require().Equal(&expectedInferences, actualInferences)
}

func (s *KeeperTestSuite) TestGetInferencesAtBlockOutlierResistant() {
	ctx := s.Ctx()
	k := s.WorkerKeeper()
	topicId := uint64(1)
	block := types.BlockHeight(100)
	// Force setting values of MAD and last_median to 10
	err := s.TopicKeeper().SetLastMedianInferences(ctx, topicId, alloraMath.NewDecFromInt64(150))
	s.Require().NoError(err)
	err = s.TopicKeeper().SetMadInferences(ctx, topicId, alloraMath.NewDecFromInt64(10))
	s.Require().NoError(err)

	// Create a set of inferences with a high value to test outlier resistant filtering
	expectedInferences := types.Inferences{
		Inferences: []*types.Inference{
			{
				TopicId:     topicId,
				BlockHeight: block,
				Value:       alloraMath.NewDecFromInt64(100), // Assuming NewDecFromInt64 exists and is appropriate
				Inferer:     "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
			},
			{
				TopicId:     topicId,
				BlockHeight: block,
				Value:       alloraMath.NewDecFromInt64(200),
				Inferer:     "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
			},
			{
				TopicId:     topicId,
				BlockHeight: block,
				Value:       alloraMath.NewDecFromInt64(10000),
				Inferer:     "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
			},
		},
	}

	// Insert directly as active inferences
	nonce := types.Nonce{BlockHeight: block}
	err = k.InsertActiveInferences(ctx, topicId, nonce.BlockHeight, expectedInferences)
	s.Require().NoError(err)

	// Confirm the non-or keeps all inferences
	actualInferences, err := k.GetInferencesAtBlock(ctx, topicId, block, false)
	s.Require().NoError(err)
	s.Require().Len(actualInferences.Inferences, 3)

	actualInferences, err = k.GetInferencesAtBlock(ctx, topicId, block, true)
	s.Require().NoError(err)
	s.Require().Len(actualInferences.Inferences, 2)
	s.Require().Equal(alloraMath.NewDecFromInt64(100), actualInferences.Inferences[0].Value)
	s.Require().Equal(alloraMath.NewDecFromInt64(200), actualInferences.Inferences[1].Value)

}

func (s *KeeperTestSuite) TestGetLatestTopicInferences() {
	ctx := s.Ctx()
	k := s.WorkerKeeper()

	topicId := uint64(1)

	// Initially, there should be no inferences, so we expect an empty result
	emptyInferences, emptyBlockHeight, err := k.GetLatestTopicInferences(ctx, topicId, false)
	s.Require().NoError(err, "Retrieving latest inferences when none exist should not result in an error")
	s.Require().Equal(&types.Inferences{Inferences: []*types.Inference{}}, emptyInferences, "Expected no inferences initially")
	s.Require().Equal(types.BlockHeight(0), emptyBlockHeight, "Expected block height to be zero initially")

	// Insert first set of inferences
	blockHeight1 := types.BlockHeight(12345)
	newInference1 := types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeight1,
		Inferer:     "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5",
		Value:       alloraMath.MustNewDecFromString("10"),
		ExtraData:   []byte("data1"),
		Proof:       "proof1",
	}
	inferences1 := types.Inferences{
		Inferences: []*types.Inference{&newInference1},
	}
	nonce1 := types.Nonce{BlockHeight: blockHeight1}
	err = k.InsertActiveInferences(ctx, topicId, nonce1.BlockHeight, inferences1)
	s.Require().NoError(err, "Inserting first set of inferences should not fail")

	// Insert second set of inferences
	blockHeight2 := types.BlockHeight(12346)
	newInference2 := types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeight2,
		Inferer:     "allo1dwxj49n0t5969uj4zfuemxg8a2ty85njn9xy9t",
		Value:       alloraMath.MustNewDecFromString("20"),
		ExtraData:   []byte("data2"),
		Proof:       "proof2",
	}
	inferences2 := types.Inferences{
		Inferences: []*types.Inference{&newInference2},
	}
	nonce2 := types.Nonce{BlockHeight: blockHeight2}
	err = k.InsertActiveInferences(ctx, topicId, nonce2.BlockHeight, inferences2)
	s.Require().NoError(err, "Inserting second set of inferences should not fail")

	// Retrieve the latest inferences
	latestInferences, latestBlockHeight, err := k.GetLatestTopicInferences(ctx, topicId, false)
	s.Require().NoError(err, "Retrieving latest inferences should not fail")
	s.Require().Equal(&inferences2, latestInferences, "Latest inferences should match the second inserted set")
	s.Require().Equal(blockHeight2, latestBlockHeight, "Latest block height should match the second inserted set")
}

func (s *KeeperTestSuite) TestGetWorkerLatestInferenceByTopicId() {
	ctx := s.Ctx()
	k := s.WorkerKeeper()

	topicId := uint64(1)
	workerAccStr := "allo1xy0pf5hq85j873glav6aajkvtennmg3fpu3cec"

	_, err := k.GetWorkerLatestInferenceByTopicId(ctx, topicId, workerAccStr)
	s.Require().Error(err, "Retrieving an inference that does not exist should result in an error")

	blockHeight1 := int64(12345)
	newInference1 := types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeight1,
		Inferer:     workerAccStr,
		Value:       alloraMath.MustNewDecFromString("10"),
		ExtraData:   []byte("data"),
		Proof:       "proof123",
	}
	err = k.InsertInference(ctx, topicId, newInference1)
	s.Require().NoError(err, "Inserting inferences should not fail")

	blockHeight2 := int64(12346)
	newInference2 := types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeight2,
		Inferer:     workerAccStr,
		Value:       alloraMath.MustNewDecFromString("10"),
		ExtraData:   []byte("data"),
		Proof:       "proof123",
	}
	err = k.InsertInference(ctx, topicId, newInference2)
	s.Require().NoError(err, "Inserting inferences should not fail")

	retrievedInference, err := k.GetWorkerLatestInferenceByTopicId(ctx, topicId, workerAccStr)
	s.Require().NoError(err, "Retrieving an existing inference should not fail")
	s.Require().Equal(newInference2, retrievedInference, "Retrieved inference should match the inserted one")
}

func (s *KeeperTestSuite) TestGetForecastsAtBlock() {
	ctx := s.Ctx()
	k := s.WorkerKeeper()
	topicId := uint64(1)
	block := types.BlockHeight(100)
	expectedForecasts := types.Forecasts{
		Forecasts: []*types.Forecast{
			{
				TopicId:     topicId,
				BlockHeight: block,
				Forecaster:  "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
						Value:   alloraMath.MustNewDecFromString("1"),
					},
					{
						Inferer: "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
						Value:   alloraMath.MustNewDecFromString("2"),
					},
				},
			},
			{
				TopicId:     topicId,
				BlockHeight: block,
				Forecaster:  "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
						Value:   alloraMath.MustNewDecFromString("3"),
					},
					{
						Inferer: "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
						Value:   alloraMath.MustNewDecFromString("4"),
					},
				},
			},
		},
	}

	// Assume InsertActiveForecasts correctly sets up forecasts
	nonce := types.Nonce{BlockHeight: block}
	err := k.InsertActiveForecasts(ctx, topicId, nonce.BlockHeight, expectedForecasts)
	s.Require().NoError(err)

	// Retrieve forecasts
	actualForecasts, err := k.GetForecastsAtBlock(ctx, topicId, block)
	s.Require().NoError(err)
	s.Require().Equal(&expectedForecasts, actualForecasts)
}

func (s *KeeperTestSuite) TestInsertWorker() {
	ctx := s.Ctx()
	k := s.WorkerKeeper()
	worker := "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5"
	topicId := uint64(401)

	// Define sample OffchainNode information for a worker
	workerInfo := types.OffchainNode{
		Owner:       "allo1wmvlvr82nlnu2y6hewgjwex30spyqgzvjhc80h",
		NodeAddress: worker,
	}

	// Attempt to insert the worker for multiple topics
	err := k.InsertWorker(ctx, topicId, worker, workerInfo)
	s.Require().NoError(err)

	node, err := k.GetWorkerInfo(ctx, worker)

	s.Require().NoError(err)
	s.Require().Equal(workerInfo.Owner, node.Owner)
	s.Require().Equal(workerInfo.NodeAddress, node.NodeAddress)
}

func (s *KeeperTestSuite) TestUpdateWorkerOwner() {
	ctx := s.Ctx()
	k := s.WorkerKeeper()

	topicId := uint64(402)
	worker := s.AddrsStr(0)
	initialOwner := s.AddrsStr(1)
	newOwner := s.AddrsStr(2)
	nonRegisteredWorker := s.AddrsStr(3)

	err := k.InsertWorker(ctx, topicId, worker, types.OffchainNode{
		NodeAddress: worker,
		Owner:       initialOwner,
	})
	s.Require().NoError(err)

	oldOwner, err := k.UpdateWorkerOwner(ctx, worker, newOwner)
	s.Require().NoError(err)
	s.Require().Equal(initialOwner, oldOwner)

	node, err := k.GetWorkerInfo(ctx, worker)
	s.Require().NoError(err)
	s.Require().Equal(newOwner, node.Owner)

	_, err = k.UpdateWorkerOwner(ctx, nonRegisteredWorker, newOwner)
	s.Require().ErrorIs(err, types.ErrAddressNotRegistered)
}

func (s *KeeperTestSuite) TestRemoveWorker() {
	ctx := s.Ctx()
	k := s.WorkerKeeper()
	worker := "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5"
	topicId := uint64(401) // Assume the worker is associated with this topicId initially

	// Define sample OffchainNode information for a worker
	workerInfo := types.OffchainNode{
		Owner:       "allo1wmvlvr82nlnu2y6hewgjwex30spyqgzvjhc80h",
		NodeAddress: "allo195jgulwj7vd292m0fth5gwqu4r2447dnarunmx",
	}

	// Insert the worker
	insertErr := k.InsertWorker(ctx, topicId, worker, workerInfo)
	s.Require().NoError(insertErr, "Failed to insert worker initially")

	// Verify the worker is registered in the topic
	isRegisteredPre, preErr := k.IsWorkerRegisteredInTopic(ctx, topicId, worker)
	s.Require().NoError(preErr, "Failed to check worker registration before removal")
	s.Require().True(isRegisteredPre, "Worker should be registered in the topic before removal")

	// Perform the removal
	removeErr := k.RemoveWorker(ctx, topicId, worker)
	s.Require().NoError(removeErr, "Failed to remove worker")

	// Verify the worker is no longer registered in the topic
	isRegisteredPost, postErr := k.IsWorkerRegisteredInTopic(ctx, topicId, worker)
	s.Require().NoError(postErr, "Failed to check worker registration after removal")
	s.Require().False(isRegisteredPost, "Worker should not be registered in the topic after removal")
}

func (s *KeeperTestSuite) TestAppendInference() {
	ctx := s.Ctx()
	k := s.WorkerKeeper()
	// Topic IDs
	topicId := s.CreateTopic()
	nonce := types.Nonce{BlockHeight: 10}
	blockHeightInferences := int64(10)

	// Set previous topic quantile inferer score ema
	err := s.ScoresKeeper().SetPreviousTopicQuantileInfererScoreEma(ctx, topicId, alloraMath.MustNewDecFromString("1000"))
	s.Require().NoError(err)

	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)

	worker1 := "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5"
	worker2 := "allo1dwxj49n0t5969uj4zfuemxg8a2ty85njn9xy9t"
	worker3 := "allo1wha0sj6pldjwjasc0pm28htgpqa5uf69kafe5y"
	worker4 := "allo19n8h9zwsqawpfn9qk9v773zj6rcjmqt28a2gyd"
	worker5 := "allo18d5wvwlhc08kfu27l25q9zr2shhanlh9fvt225"
	ogWorker2Score := alloraMath.MustNewDecFromString("90")

	score1 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker1, Score: alloraMath.NewDecFromInt64(95)}
	score2 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker2, Score: ogWorker2Score}
	score3 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker3, Score: alloraMath.NewDecFromInt64(99)}
	score4 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker4, Score: alloraMath.NewDecFromInt64(91)}
	score5 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker5, Score: alloraMath.NewDecFromInt64(96)}
	err = s.ScoresKeeper().SetInfererScoreEma(ctx, topicId, worker1, score1)
	s.Require().NoError(err)
	err = s.ScoresKeeper().SetInfererScoreEma(ctx, topicId, worker2, score2)
	s.Require().NoError(err)
	err = s.ScoresKeeper().SetInfererScoreEma(ctx, topicId, worker3, score3)
	s.Require().NoError(err)
	err = s.ScoresKeeper().SetInfererScoreEma(ctx, topicId, worker4, score4)
	s.Require().NoError(err)
	err = s.ScoresKeeper().SetInfererScoreEma(ctx, topicId, worker5, score5)
	s.Require().NoError(err)

	// Ensure that the number of top inferers is capped at the max top inferers to reward
	// New high-score entrant should replace earlier low-score entrant
	params := types.DefaultParams()
	params.MaxTopInferersToReward = 4
	err = s.ParamsKeeper().SetParams(ctx, params)
	s.Require().NoError(err)

	allInferences := types.Inferences{
		Inferences: []*types.Inference{
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.52")},
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.71")},
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker3, Value: alloraMath.MustNewDecFromString("0.71")},
		},
	}
	for _, inference := range allInferences.Inferences {
		err = k.AppendInference(ctx, topic, nonce.BlockHeight, inference, params.MaxTopInferersToReward)
		s.Require().NoError(err)
	}

	blockHeightInferences = blockHeightInferences + topic.EpochLength
	newInference := types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeightInferences,
		Inferer:     worker4,
		Value:       alloraMath.MustNewDecFromString("0.52"),
		ExtraData:   nil,
		Proof:       "",
	}
	err = k.AppendInference(ctx, topic, nonce.BlockHeight, &newInference, params.MaxTopInferersToReward)
	s.Require().NoError(err)
	activeInferers, err := k.GetActiveInferersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopInferersToReward, uint64(len(activeInferers)))

	blockHeightInferences = blockHeightInferences + topic.EpochLength
	newInference2 := types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeightInferences,
		Inferer:     worker5,
		Value:       alloraMath.MustNewDecFromString("0.52"),
		ExtraData:   nil,
		Proof:       "",
	}

	err = k.AppendInference(ctx, topic, nonce.BlockHeight, &newInference2, params.MaxTopInferersToReward)
	s.Require().NoError(err)
	activeInferers, err = k.GetActiveInferersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopInferersToReward, uint64(len(activeInferers)))

	// New high-score entrant should replace earlier low-score entrant
	worker5OgScore, err := s.ScoresKeeper().GetInfererScoreEma(ctx, topicId, worker5)
	s.Require().NoError(err)
	worker5Found := false
	for _, address := range activeInferers {
		if address == worker5 {
			worker5Found = true
		}
	}
	s.Require().True(worker5Found)

	// Ensure EMA score of active set is not yet updated
	// This will happen later during epoch reward calculation, not here
	worker5NewScore, err := s.ScoresKeeper().GetInfererScoreEma(ctx, topicId, worker5)
	s.Require().NoError(err)
	// EMA score should be updated higher because saved topic quantile ema is higher
	s.Require().True(worker5OgScore.Score.Equal(worker5NewScore.Score))
	// EMA score should be updated with the new time of update given that it was updated then
	s.Require().Equal(worker5OgScore.BlockHeight, worker5NewScore.BlockHeight)

	// Ensure EMA score of actor moved to passive set is updated
	updatedWorker2Score, err := s.ScoresKeeper().GetInfererScoreEma(ctx, topicId, worker2)
	s.Require().NoError(err)
	// EMA score should be updated higher because saved topic quantile ema is higher
	updatedWorker2ScoreVal, err := updatedWorker2Score.Score.Int64()
	s.Require().NoError(err)
	ogWorker2ScoreVal, err := ogWorker2Score.Int64()
	s.Require().NoError(err)
	worker5OgScoreVal, err := worker5OgScore.Score.Int64()
	s.Require().NoError(err)
	s.Require().Greater(updatedWorker2ScoreVal, ogWorker2ScoreVal, "worker2 score should go up given large ema value")
	s.Require().Greater(updatedWorker2ScoreVal, worker5OgScoreVal, "worker2 could not overtake worker5, but not in this epoch")
	// EMA score should be updated with the new time of update given that it was updated then
	s.Require().Equal(nonce.BlockHeight, updatedWorker2Score.BlockHeight)

	// Ensure passive set participant can't update their score within the same epoch
	blockHeightInferences = blockHeightInferences + 1 // within the same epoch => no update
	newInference2 = types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeightInferences,
		Inferer:     worker2,
		Value:       alloraMath.MustNewDecFromString("0.52"),
		ExtraData:   nil,
		Proof:       "",
	}
	err = k.AppendInference(ctx, topic, nonce.BlockHeight, &newInference2, params.MaxTopInferersToReward)
	s.Require().Error(err, types.ErrCantUpdateEmaMoreThanOncePerWindow.Error())
	// Confirm no change in EMA score
	updateAttemptForWorker2, err := s.ScoresKeeper().GetInfererScoreEma(ctx, topicId, worker2)
	s.Require().NoError(err)
	updateAttemptForWorker2Val, err := updateAttemptForWorker2.Score.Int64()
	s.Require().NoError(err)
	s.Require().Equal(updateAttemptForWorker2Val, updatedWorker2ScoreVal, "unchanged score")
	s.Require().Equal(updateAttemptForWorker2.BlockHeight, updatedWorker2Score.BlockHeight, "unchanged height")
}

func getNewAddress() string {
	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	return addr.String()
}

func (s *KeeperTestSuite) TestAppendInferenceWithResetActiveWorkers() {
	ctx := s.Ctx()
	k := s.WorkerKeeper()
	topicId := s.CreateTopic(testutil.WithEpochLength(10801), testutil.WithGroundTruthLag(10801))
	nonce := types.Nonce{BlockHeight: 10}
	blockHeightInferences := int64(10)

	// Set previous topic quantile inferer score ema
	err := s.ScoresKeeper().SetPreviousTopicQuantileInfererScoreEma(ctx, topicId, alloraMath.MustNewDecFromString("1000"))
	s.Require().NoError(err)

	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)

	// Create workers
	workers := make([]string, 6)
	for i := range workers {
		workers[i] = getNewAddress()
	}

	// Set scores for workers 2-6
	for i := 1; i < len(workers); i++ {
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: 2,
			Address:     workers[i],
			Score:       alloraMath.NewDecFromInt64(int64(91 + i)),
		}
		err = s.ScoresKeeper().SetInfererScoreEma(ctx, topicId, workers[i], score)
		s.Require().NoError(err)
	}

	params := types.DefaultParams()
	params.MaxTopInferersToReward = 4
	err = s.ParamsKeeper().SetParams(ctx, params)
	s.Require().NoError(err)

	formatValue := func(val float64) string {
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", val), "0"), ".")
	}

	// Create first set of inferences
	inferences := make([]*types.Inference, 5)
	for i := range inferences {
		//nolint:exhaustruct
		inferences[i] = &types.Inference{
			TopicId:     topicId,
			BlockHeight: blockHeightInferences,
			Inferer:     workers[i],
			Value:       alloraMath.MustNewDecFromString(formatValue(0.11 + float64(i)*0.01)),
		}
	}

	allInferences := types.Inferences{Inferences: inferences}
	for _, inference := range allInferences.Inferences {
		err = k.AppendInference(ctx, topic, nonce.BlockHeight, inference, params.MaxTopInferersToReward)
		s.Require().NoError(err)
	}

	activeInferers, err := k.GetActiveInferersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopInferersToReward, uint64(len(activeInferers)))

	lowestEmaScore, found, err := s.ScoresKeeper().GetLowestInfererScoreEma(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(lowestEmaScore.Address, workers[1])

	err = k.ResetActiveWorkersForTopic(ctx, topicId)
	s.Require().NoError(err)

	activeInferers, err = k.GetActiveInferersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Empty(activeInferers)

	lowestEmaScore, found, err = s.ScoresKeeper().GetLowestInfererScoreEma(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(lowestEmaScore.Address, workers[1])

	// Create second set of inferences
	blockHeightInferences = blockHeightInferences + topic.EpochLength
	inferences = make([]*types.Inference, 5)
	for i := 1; i <= len(inferences); i++ {
		//nolint:exhaustruct
		inferences[i-1] = &types.Inference{
			TopicId:     topicId,
			BlockHeight: blockHeightInferences,
			Inferer:     workers[i],
			Value:       alloraMath.MustNewDecFromString(formatValue(0.22 + float64(i-1)*0.01)),
		}
	}

	allInferences = types.Inferences{Inferences: inferences}
	nonce.BlockHeight++
	for _, inference := range allInferences.Inferences {
		err = k.AppendInference(ctx, topic, nonce.BlockHeight, inference, params.MaxTopInferersToReward)
		s.Require().NoError(err)
	}

	activeInferers, err = k.GetActiveInferersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopInferersToReward, uint64(len(activeInferers)))

	lowestEmaScore, found, err = s.ScoresKeeper().GetLowestInfererScoreEma(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(lowestEmaScore.Address, workers[2])
}

func mockUninitializedParams() types.Params {
	return types.Params{
		Version:                             "v2",
		MinTopicWeight:                      alloraMath.MustNewDecFromString("0"),
		RequiredMinimumStake:                cosmosMath.NewInt(0),
		RemoveStakeDelayWindow:              0,
		MinEpochLength:                      1,
		BetaEntropy:                         alloraMath.MustNewDecFromString("0"),
		LearningRate:                        alloraMath.MustNewDecFromString("0.0001"),
		GradientDescentMaxIters:             uint64(100),
		MaxGradientThreshold:                alloraMath.MustNewDecFromString("0.0001"),
		MinStakeFraction:                    alloraMath.MustNewDecFromString("0"),
		EpsilonReputer:                      alloraMath.MustNewDecFromString("0.0001"),
		EpsilonSafeDiv:                      alloraMath.MustNewDecFromString("0.0001"),
		MaxUnfulfilledWorkerRequests:        uint64(0),
		MaxUnfulfilledReputerRequests:       uint64(0),
		TopicRewardStakeImportance:          alloraMath.MustNewDecFromString("0"),
		TopicRewardFeeRevenueImportance:     alloraMath.MustNewDecFromString("0"),
		TopicRewardAlpha:                    alloraMath.MustNewDecFromString("0.1"),
		TaskRewardAlpha:                     alloraMath.MustNewDecFromString("0.1"),
		ValidatorsVsAlloraPercentReward:     alloraMath.MustNewDecFromString("0"),
		MaxSamplesToScaleScores:             uint64(10),
		MaxTopInferersToReward:              uint64(0),
		MaxTopForecastersToReward:           uint64(0),
		MaxTopReputersToReward:              uint64(0),
		CreateTopicFee:                      cosmosMath.NewInt(0),
		RegistrationFee:                     cosmosMath.NewInt(0),
		DefaultPageLimit:                    uint64(0),
		MaxPageLimit:                        uint64(0),
		MinEpochLengthRecordLimit:           int64(0),
		MaxSerializedMsgLength:              int64(0),
		BlocksPerMonth:                      uint64(864000),
		PRewardInference:                    alloraMath.NewDecFromInt64(1),
		PRewardForecast:                     alloraMath.NewDecFromInt64(1),
		PRewardReputer:                      alloraMath.NewDecFromInt64(1),
		CRewardInference:                    alloraMath.MustNewDecFromString("0.1"),
		CRewardForecast:                     alloraMath.MustNewDecFromString("0.1"),
		CNorm:                               alloraMath.ZeroDec(), // deprecated global c_norm kept for compat
		HalfMaxProcessStakeRemovalsEndBlock: uint64(1),
		DataSendingFee:                      cosmosMath.NewInt(0),
		MaxElementsPerForecast:              uint64(0),
		MaxActiveTopicsPerBlock:             uint64(0),
		MaxStringLength:                     uint64(0),
		InitialRegretQuantile:               alloraMath.ZeroDec(),
		PNormSafeDiv:                        alloraMath.ZeroDec(),
		GlobalWhitelistEnabled:              true,
		TopicCreatorWhitelistEnabled:        true,
		MinExperiencedWorkerRegrets:         uint64(10),
		InferenceOutlierDetectionThreshold:  alloraMath.MustNewDecFromString("11"),
		InferenceOutlierDetectionAlpha:      alloraMath.MustNewDecFromString("0.2"),
		LambdaInitialScore:                  alloraMath.MustNewDecFromString("2"),
		GlobalWorkerWhitelistEnabled:        true,
		GlobalReputerWhitelistEnabled:       true,
		GlobalAdminWhitelistAppended:        true,
		MaxWhitelistInputArrayLength:        uint64(10),
		MinWeightThresholdForStdnorm:        alloraMath.MustNewDecFromString("0.000001"),
	}
}

func (s *KeeperTestSuite) TestAppendForecast() {
	ctx := s.Ctx()
	k := s.WorkerKeeper()
	topicId := s.CreateTopic()
	nonce := types.Nonce{BlockHeight: 10}
	blockHeightInferences := int64(10)

	workers := make([]string, 5)
	for i := range workers {
		workers[i] = s.AddrsStr(i)
	}

	// Set initial scores
	scores := []int64{95, 90, 99, 91, 96}
	for i, worker := range workers {
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: 2,
			Address:     worker,
			Score:       alloraMath.NewDecFromInt64(scores[i]),
		}
		err := s.ScoresKeeper().SetForecasterScoreEma(ctx, topicId, worker, score)
		s.Require().NoError(err)
	}

	params := mockUninitializedParams()
	params.MaxTopForecastersToReward = 4
	err := s.ParamsKeeper().SetParams(ctx, params)
	s.Require().NoError(err)

	//nolint:exhaustruct
	newForecast := types.Forecast{
		TopicId:     topicId,
		BlockHeight: blockHeightInferences,
		ForecastElements: []*types.ForecastElement{
			{
				Inferer: workers[0],
				Value:   alloraMath.MustNewDecFromString("0.52"),
			},
			{
				Inferer: workers[1],
				Value:   alloraMath.MustNewDecFromString("0.52"),
			},
		},
	}

	// Create forecasts for first 4 workers
	forecasts := make([]*types.Forecast, 4)
	for i := range forecasts {
		forecast := newForecast
		forecast.Forecaster = workers[i]
		forecasts[i] = &forecast
	}

	allForecasts := types.Forecasts{Forecasts: forecasts}

	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)

	// Append initial forecasts
	for _, forecast := range allForecasts.Forecasts {
		err = k.AppendForecast(ctx, topic, nonce.BlockHeight, forecast, params.MaxTopForecastersToReward)
		s.Require().NoError(err)
	}

	activeForecasters, err := k.GetActiveForecastersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopForecastersToReward, uint64(len(activeForecasters)))

	// Try to add forecast for the fifth worker
	blockHeightInferences = blockHeightInferences + topic.EpochLength
	newForecast.Forecaster = workers[4]
	newForecast.BlockHeight = blockHeightInferences

	// MaxTopForecastersToReward is 4, so this should not increase active forecasters
	err = k.AppendForecast(ctx, topic, nonce.BlockHeight, &newForecast, params.MaxTopForecastersToReward)
	s.Require().NoError(err)

	activeForecasters, err = k.GetActiveForecastersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopForecastersToReward, uint64(len(activeForecasters)))
}

func (s *KeeperTestSuite) TestAppendForecastWithResetActiveForecasters() {
	ctx := s.Ctx()
	k := s.WorkerKeeper()
	topicId := s.CreateTopic()
	nonce := types.Nonce{BlockHeight: 10}
	blockHeightInferences := int64(10)

	// Create workers and set their scores
	workers := make([]string, 5)
	scores := []int64{95, 90, 99, 91, 96}
	for i := range workers {
		workers[i] = s.AddrsStr(i)
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: 2,
			Address:     workers[i],
			Score:       alloraMath.NewDecFromInt64(scores[i]),
		}
		err := s.ScoresKeeper().SetForecasterScoreEma(ctx, topicId, workers[i], score)
		s.Require().NoError(err)
	}

	// Set params
	params := mockUninitializedParams()
	params.MaxTopForecastersToReward = 4
	err := s.ParamsKeeper().SetParams(ctx, params)
	s.Require().NoError(err)

	// Create forecast elements template
	forecastElements := []*types.ForecastElement{
		{
			Inferer: workers[0],
			Value:   alloraMath.MustNewDecFromString("0.52"),
		},
		{
			Inferer: workers[1],
			Value:   alloraMath.MustNewDecFromString("0.52"),
		},
	}

	// Create forecasts for all workers
	allForecasts := types.Forecasts{Forecasts: make([]*types.Forecast, len(workers))}
	for i, worker := range workers {
		//nolint:exhaustruct
		allForecasts.Forecasts[i] = &types.Forecast{
			TopicId:          topicId,
			BlockHeight:      blockHeightInferences,
			Forecaster:       worker,
			ForecastElements: forecastElements,
		}
	}

	// Append initial forecasts and verify
	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)

	for _, forecast := range allForecasts.Forecasts {
		err = k.AppendForecast(ctx, topic, nonce.BlockHeight, forecast, params.MaxTopForecastersToReward)
		s.Require().NoError(err)
	}

	activeForecasters, err := k.GetActiveForecastersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopForecastersToReward, uint64(len(activeForecasters)))

	// Reset active forecasters and verify
	err = k.ResetActiveWorkersForTopic(ctx, topicId)
	s.Require().NoError(err)

	activeForecasters, err = k.GetActiveForecastersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Empty(activeForecasters)

	// Re-append forecasts and verify again
	topic, err = s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)

	nonce.BlockHeight++
	for _, forecast := range allForecasts.Forecasts {
		err = k.AppendForecast(ctx, topic, nonce.BlockHeight, forecast, params.MaxTopForecastersToReward)
		s.Require().NoError(err)
	}

	activeForecasters, err = k.GetActiveForecastersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopForecastersToReward, uint64(len(activeForecasters)))
}

func (s *KeeperTestSuite) TestActiveInfererFunctions() {
	ctx := s.Ctx()
	k := s.WorkerKeeper()
	topicId := uint64(1)
	inferer := s.AddrsStr(0)

	err := k.AddActiveInferer(ctx, topicId, inferer)
	s.Require().NoError(err)

	isActive, err := k.IsActiveInferer(ctx, topicId, inferer)
	s.Require().NoError(err)
	s.Require().True(isActive)

	activeInferers, err := k.GetActiveInferersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Len(activeInferers, 1)
	s.Require().Equal(inferer, activeInferers[0])
}

func (s *KeeperTestSuite) TestActiveForecasterFunctions() {
	ctx := s.Ctx()
	k := s.WorkerKeeper()
	topicId := uint64(1)
	forecaster := s.AddrsStr(1)

	err := k.AddActiveForecaster(ctx, topicId, forecaster)
	s.Require().NoError(err)

	isActive, err := k.IsActiveForecaster(ctx, topicId, forecaster)
	s.Require().NoError(err)
	s.Require().True(isActive)

	activeForecasters, err := k.GetActiveForecastersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Len(activeForecasters, 1)
	s.Require().Equal(forecaster, activeForecasters[0])

	err = k.RemoveActiveForecaster(ctx, topicId, forecaster)
	s.Require().NoError(err)

	isActive, err = k.IsActiveForecaster(ctx, topicId, forecaster)
	s.Require().NoError(err)
	s.Require().False(isActive)
}

func (s *KeeperTestSuite) TestRemoveForecast() {
	ctx := s.Ctx()
	k := s.WorkerKeeper()
	topicId := s.CreateTopic()
	forecaster := s.AddrsStr(0)

	// Create a forecast
	forecast := types.Forecast{
		TopicId:     topicId,
		BlockHeight: 100,
		Forecaster:  forecaster,
		ForecastElements: []*types.ForecastElement{
			{
				Inferer: "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
				Value:   alloraMath.MustNewDecFromString("1"),
			},
			{
				Inferer: "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
				Value:   alloraMath.MustNewDecFromString("2"),
			},
		},
		ExtraData: []byte("data"),
	}

	// Insert the forecast
	err := k.InsertForecast(ctx, topicId, forecast)
	s.Require().NoError(err)

	// Verify the forecast was added
	retrievedForecast, err := k.GetWorkerLatestForecastByTopicId(ctx, topicId, forecaster)
	s.Require().NoError(err)
	s.Require().Equal(forecast, retrievedForecast)

	// Remove the forecast
	err = k.RemoveForecast(ctx, topicId, forecaster)
	s.Require().NoError(err)

	// Verify the forecast was removed
	_, err = k.GetWorkerLatestForecastByTopicId(ctx, topicId, forecaster)
	s.Require().Error(err) // Expect an error since the forecast should be removed
}

func (s *KeeperTestSuite) TestRemoveInference() {
	ctx := s.Ctx()
	k := s.WorkerKeeper()
	topicId := s.CreateTopic()
	inferer := s.AddrsStr(0)

	// Create an inference
	inference := types.Inference{
		TopicId:     topicId,
		BlockHeight: 100,
		Value:       alloraMath.NewDecFromInt64(1),
		Inferer:     inferer,
		ExtraData:   []byte("data"),
		Proof:       "",
	}

	// Insert the inference
	err := k.InsertInference(ctx, topicId, inference)
	s.Require().NoError(err)

	// Verify the inference was added
	retrievedInference, err := k.GetWorkerLatestInferenceByTopicId(ctx, topicId, inferer)
	s.Require().NoError(err)
	s.Require().Equal(inference, retrievedInference)

	// Remove the inference
	err = k.RemoveInference(ctx, topicId, inferer)
	s.Require().NoError(err)

	// Verify the inference was removed
	_, err = k.GetWorkerLatestInferenceByTopicId(ctx, topicId, inferer)
	s.Require().Error(err) // Expect an error since the inference should be removed
}
