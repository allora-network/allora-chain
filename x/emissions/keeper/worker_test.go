package keeper_test

import (
	"context"
	"fmt"
	"strings"

	errorsmod "cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// toInputInference is a test helper that adapts a legacy scalar *Inference
// (still used by many older tests that predate Epoch Label Registry v2) to
// the *InputInference signature AppendInference now takes. Callers that need
// multi-label or boundary-tested payloads should build InputInference
// literally instead.
func toInputInference(inf *types.Inference) *types.InputInference {
	if inf == nil {
		return nil
	}
	var v alloraMath.BoundedExp40Dec
	if len(inf.Values) > 0 {
		v = alloraMath.MustNewBoundedExp40Dec(inf.Values[0])
	}
	return &types.InputInference{
		TopicId:     inf.TopicId,
		BlockHeight: inf.BlockHeight,
		Inferer:     inf.Inferer,
		Value:       v,
		Values:      nil,
		ExtraData:   inf.ExtraData,
		Proof:       inf.Proof,
	}
}

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
				Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(1)}, // Assuming NewDecFromInt64 exists and is appropriate
				Inferer:     "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
			},
			{
				TopicId:     topicId,
				BlockHeight: block,
				Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(2)},
				Inferer:     "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
			},
		},
	}

	// Assume InsertActiveInferences correctly sets up inferences
	nonce := types.Nonce{BlockHeight: block} // Assuming block type cast to int64 if needed
	err := k.InsertActiveInferences(ctx, topicId, nonce.BlockHeight, expectedInferences)
	s.Require().NoError(err)

	topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)

	// Retrieve inferences
	actualInferences, err := k.GetInferencesAtBlock(ctx, topic, block, false)
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
				Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(100)}, // Assuming NewDecFromInt64 exists and is appropriate
				Inferer:     "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
			},
			{
				TopicId:     topicId,
				BlockHeight: block,
				Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(200)},
				Inferer:     "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
			},
			{
				TopicId:     topicId,
				BlockHeight: block,
				Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(10000)},
				Inferer:     "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
			},
		},
	}

	// Insert directly as active inferences
	nonce := types.Nonce{BlockHeight: block}
	err = k.InsertActiveInferences(ctx, topicId, nonce.BlockHeight, expectedInferences)
	s.Require().NoError(err)

	topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)

	// Confirm the non-or keeps all inferences
	actualInferences, err := k.GetInferencesAtBlock(ctx, topic, block, false)
	s.Require().NoError(err)
	s.Require().Len(actualInferences.Inferences, 3)

	actualInferences, err = k.GetInferencesAtBlock(ctx, topic, block, true)
	s.Require().NoError(err)
	s.Require().Len(actualInferences.Inferences, 2)
	s.Require().Equal(alloraMath.NewDecFromInt64(100), actualInferences.Inferences[0].Values[0])
	s.Require().Equal(alloraMath.NewDecFromInt64(200), actualInferences.Inferences[1].Values[0])

}

func (s *KeeperTestSuite) TestGetLatestTopicInferences() {
	ctx := s.Ctx()
	k := s.WorkerKeeper()

	topicId := uint64(1)
	topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)

	// Initially, there should be no inferences, so we expect an empty result
	emptyInferences, emptyBlockHeight, err := k.GetLatestTopicInferences(ctx, topic, false)
	s.Require().NoError(err, "Retrieving latest inferences when none exist should not result in an error")
	s.Require().Equal(&types.Inferences{Inferences: []*types.Inference{}}, emptyInferences, "Expected no inferences initially")
	s.Require().Equal(types.BlockHeight(0), emptyBlockHeight, "Expected block height to be zero initially")

	// Insert first set of inferences
	blockHeight1 := types.BlockHeight(12345)
	newInference1 := types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeight1,
		Inferer:     "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5",
		Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString("10")},
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
		Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString("20")},
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
	latestInferences, latestBlockHeight, err := k.GetLatestTopicInferences(ctx, topic, false)
	s.Require().NoError(err, "Retrieving latest inferences should not fail")
	s.Require().Equal(&inferences2, latestInferences, "Latest inferences should match the second inserted set")
	s.Require().Equal(blockHeight2, latestBlockHeight, "Latest block height should match the second inserted set")
}

// TestGetWorkerLatestInferenceByTopicId exercises the v2 shim: reads now come
// from workerLatestInputInferences (populated by SetWorkerLatestInputInference),
// not the legacy per-worker inferences store. The shim projects an
// InputInference into a best-effort committed Inference so existing callers
// keep compiling.
func (s *KeeperTestSuite) TestGetWorkerLatestInferenceByTopicId() {
	ctx := s.Ctx()
	k := s.WorkerKeeper()

	topicId := uint64(1)
	workerAccStr := "allo1xy0pf5hq85j873glav6aajkvtennmg3fpu3cec"

	_, err := k.GetWorkerLatestInferenceByTopicId(ctx, topicId, workerAccStr)
	s.Require().Error(err, "Retrieving an inference that does not exist should result in an error")

	blockHeight1 := int64(12345)
	firstInput := types.InputInference{
		TopicId:     topicId,
		BlockHeight: blockHeight1,
		Inferer:     workerAccStr,
		Value:       alloraMath.MustNewBoundedExp40DecFromString("10"),
		Values:      nil,
		ExtraData:   []byte("data"),
		Proof:       "proof123",
	}
	err = k.SetWorkerLatestInputInference(ctx, topicId, workerAccStr, firstInput)
	s.Require().NoError(err, "Staging the first input inference should not fail")

	blockHeight2 := int64(12346)
	secondInput := types.InputInference{
		TopicId:     topicId,
		BlockHeight: blockHeight2,
		Inferer:     workerAccStr,
		Value:       alloraMath.MustNewBoundedExp40DecFromString("10"),
		Values:      nil,
		ExtraData:   []byte("data"),
		Proof:       "proof123",
	}
	err = k.SetWorkerLatestInputInference(ctx, topicId, workerAccStr, secondInput)
	s.Require().NoError(err, "Overwriting the staged input inference should not fail")

	retrievedInference, err := k.GetWorkerLatestInferenceByTopicId(ctx, topicId, workerAccStr)
	s.Require().NoError(err, "Retrieving an existing inference should not fail")
	s.Require().Equal(types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeight2,
		Inferer:     workerAccStr,
		Values:      []alloraMath.Dec{secondInput.Value.ToDec()},
		ExtraData:   []byte("data"),
		Proof:       "proof123",
	}, retrievedInference, "Retrieved inference should match the latest staged input")
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
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker1, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("0.52")}},
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker2, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("0.71")}},
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker3, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("0.71")}},
		},
	}
	for _, inference := range allInferences.Inferences {
		err = k.AppendInference(ctx, topic, nonce.BlockHeight, toInputInference(inference), params.MaxTopInferersToReward)
		s.Require().NoError(err)
	}

	blockHeightInferences = blockHeightInferences + topic.EpochLength
	newInference := types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeightInferences,
		Inferer:     worker4,
		Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString("0.52")},
		ExtraData:   nil,
		Proof:       "",
	}
	err = k.AppendInference(ctx, topic, nonce.BlockHeight, toInputInference(&newInference), params.MaxTopInferersToReward)
	s.Require().NoError(err)
	activeInferers, err := k.GetActiveInferersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopInferersToReward, uint64(len(activeInferers)))

	blockHeightInferences = blockHeightInferences + topic.EpochLength
	newInference2 := types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeightInferences,
		Inferer:     worker5,
		Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString("0.52")},
		ExtraData:   nil,
		Proof:       "",
	}

	err = k.AppendInference(ctx, topic, nonce.BlockHeight, toInputInference(&newInference2), params.MaxTopInferersToReward)
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
		Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString("0.52")},
		ExtraData:   nil,
		Proof:       "",
	}
	err = k.AppendInference(ctx, topic, nonce.BlockHeight, toInputInference(&newInference2), params.MaxTopInferersToReward)
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
			Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString(formatValue(0.11 + float64(i)*0.01))},
		}
	}

	allInferences := types.Inferences{Inferences: inferences}
	for _, inference := range allInferences.Inferences {
		err = k.AppendInference(ctx, topic, nonce.BlockHeight, toInputInference(inference), params.MaxTopInferersToReward)
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
			Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString(formatValue(0.22 + float64(i-1)*0.01))},
		}
	}

	allInferences = types.Inferences{Inferences: inferences}
	nonce.BlockHeight++
	for _, inference := range allInferences.Inferences {
		err = k.AppendInference(ctx, topic, nonce.BlockHeight, toInputInference(inference), params.MaxTopInferersToReward)
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
		MaxLabelsPerSubmission:              types.DefaultMaxLabelsPerSubmission,
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

// TestRemoveInference verifies that RemoveInference scrubs the worker's raw
// staged InputInference from the workerLatestInputInferences store, not the
// legacy deprecated inferences store. The compatibility shim on
// GetWorkerLatestInferenceByTopicId projects that raw row, so reading it back
// after removal must error.
func (s *KeeperTestSuite) TestRemoveInference() {
	ctx := s.Ctx()
	k := s.WorkerKeeper()
	topicId := s.CreateTopic()
	inferer := s.AddrsStr(0)

	input := types.InputInference{
		TopicId:     topicId,
		BlockHeight: 100,
		Inferer:     inferer,
		Value:       alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(1)),
		Values:      nil,
		ExtraData:   []byte("data"),
		Proof:       "",
	}

	err := k.SetWorkerLatestInputInference(ctx, topicId, inferer, input)
	s.Require().NoError(err)

	retrievedInference, err := k.GetWorkerLatestInferenceByTopicId(ctx, topicId, inferer)
	s.Require().NoError(err)
	s.Require().Equal(topicId, retrievedInference.TopicId)
	s.Require().Equal(int64(100), retrievedInference.BlockHeight)
	s.Require().Equal(inferer, retrievedInference.Inferer)
	s.Require().Equal(1, len(retrievedInference.Values))
	s.Require().Equal("1", retrievedInference.Values[0].String())

	err = k.RemoveWorkerLatestInputInference(ctx, topicId, inferer)
	s.Require().NoError(err)

	_, err = k.GetWorkerLatestInferenceByTopicId(ctx, topicId, inferer)
	s.Require().Error(err)
}

func (s *KeeperTestSuite) TestNewInferenceForecastBundleFromInput() {
	validInference := &types.InputInference{
		TopicId:     1,
		BlockHeight: 100,
		Inferer:     "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
		Value:       alloraMath.MustNewBoundedExp40DecFromString("1.23"),
		Values:      []*types.InputLabeledValue{{Label: "", Value: alloraMath.MustNewBoundedExp40DecFromString("1.23")}},
		ExtraData:   []byte("extra"),
		Proof:       "proof",
	}

	validForecast := &types.InputForecast{
		TopicId:     1,
		BlockHeight: 100,
		Forecaster:  "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5",
		ForecastElements: []*types.InputForecastElement{
			{
				Inferer: "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
				Value:   alloraMath.MustNewBoundedExp40DecFromString("1.23"),
			},
		},
		ExtraData: []byte("extra"),
	}

	tests := []struct {
		name    string
		input   *types.InputInferenceForecastBundle
		wantErr bool
	}{
		{
			name: "valid input",
			input: &types.InputInferenceForecastBundle{
				Inference: validInference,
				Forecast:  validForecast,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			topic, err := s.TopicKeeper().GetTopic(s.Ctx(), validInference.TopicId)
			s.Require().NoError(err)

			got, err := keeper.NormalizeInferenceForecastBundle(topic, tt.input)
			if tt.wantErr {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)
			if tt.input == nil {
				s.Require().Nil(got)
				return
			}
			s.Require().NotNil(got.Inference)
			s.Require().NotNil(got.Forecast)
		})
	}
}

// TestNormalizeInputInference covers the v2 contract for
// NormalizeInputInference: callers MUST have already canonicalized labels
// (via InputInference.ValidateWithLimits); Normalize is only responsible for
// (a) SINGLE-arity scalar/labeled selection, (b) MULTI-arity alphabetical
// sorting, and (c) RequireUnity enforcement. It no longer registers labels
// or pads against a pre-existing registry - that happens at close time via
// BuildFinalEpochLabelRegistryFromActiveSet.
//
//nolint:exhaustruct
func (s *KeeperTestSuite) TestNormalizeInputInference() {
	type labeledInput struct {
		label string
		value string
	}
	type tc struct {
		name         string
		arity        types.TopicOutputArity
		requireUnity bool
		unityTol     string
		nonce        int64

		scalarValue string
		labeled     []labeledInput

		wantErr   bool
		wantErrIs error

		wantValuesStr []string
	}

	cases := []tc{
		{
			name:          "SINGLE_uses_scalar_when_no_labeled",
			arity:         types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			nonce:         1,
			scalarValue:   "42",
			wantValuesStr: []string{"42"},
		},
		{
			name:          "SINGLE_accepts_canonical_y_label",
			arity:         types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			nonce:         1,
			scalarValue:   "999",
			labeled:       []labeledInput{{label: "y", value: "7"}},
			wantValuesStr: []string{"7"},
		},
		{
			name:        "SINGLE_rejects_non_y_label",
			arity:       types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			nonce:       1,
			scalarValue: "1",
			labeled:     []labeledInput{{label: "x", value: "1"}},
			wantErr:     true,
			wantErrIs:   sdkerrors.ErrInvalidRequest,
		},
		{
			name:        "SINGLE_rejects_when_labeled_len_gt_1",
			arity:       types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			nonce:       1,
			scalarValue: "1",
			labeled: []labeledInput{
				{label: "y", value: "1"},
				{label: "z", value: "2"},
			},
			wantErr:   true,
			wantErrIs: sdkerrors.ErrInvalidRequest,
		},
		{
			name:      "MULTI_requires_labeled_values",
			arity:     types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			nonce:     1,
			labeled:   nil,
			wantErr:   true,
			wantErrIs: sdkerrors.ErrInvalidRequest,
		},
		{
			name:  "MULTI_sorts_alphabetically",
			arity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			nonce: 1,
			labeled: []labeledInput{
				{label: "b", value: "0.7"},
				{label: "a", value: "0.3"},
			},
			// Output is sorted by canonical label name.
			wantValuesStr: []string{"0.3", "0.7"},
		},
		{
			name:         "MULTI_require_unity_ok",
			arity:        types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			requireUnity: true,
			unityTol:     "0.000001",
			nonce:        1,
			labeled: []labeledInput{
				{label: "a", value: "0.2"},
				{label: "b", value: "0.8"},
			},
			wantValuesStr: []string{"0.2", "0.8"},
		},
		{
			name:         "MULTI_require_unity_rejected_outside_tol",
			arity:        types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			requireUnity: true,
			unityTol:     "0.01",
			nonce:        1,
			labeled: []labeledInput{
				{label: "a", value: "0.2"},
				{label: "b", value: "0.7"},
			},
			wantErr:   true,
			wantErrIs: sdkerrors.ErrInvalidRequest,
		},
		{
			// Normalize is defensive: if a caller forgets to canonicalize,
			// Normalize rejects empty labels rather than silently building
			// an invalid intermediate Inference.
			name:      "MULTI_defensive_rejects_empty_label",
			arity:     types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			nonce:     1,
			labeled:   []labeledInput{{label: "", value: "0.5"}},
			wantErr:   true,
			wantErrIs: sdkerrors.ErrInvalidRequest,
		},
	}

	for _, c := range cases {
		s.Run(c.name, func() {
			s.SetupTest()

			ctx := s.Ctx()
			tk := s.TopicKeeper()

			topicId := s.CreateTopic()
			topic, err := tk.GetTopic(ctx, topicId)
			s.Require().NoError(err)

			topic.OutputArity = c.arity
			topic.RequireUnity = c.requireUnity
			tol := c.unityTol
			if tol == "" {
				tol = "0"
			}
			topic.UnityTolerance = alloraMath.MustNewDecFromString(tol)
			s.Require().NoError(tk.SetTopic(ctx, topicId, topic))

			scalar := c.scalarValue
			if scalar == "" {
				scalar = "0"
			}
			in := &types.InputInference{
				TopicId:     topicId,
				BlockHeight: c.nonce,
				Inferer:     "inferer",
				Value:       alloraMath.MustNewBoundedExp40DecFromString(scalar),
			}
			if c.labeled != nil {
				in.Values = make([]*types.InputLabeledValue, 0, len(c.labeled))
				for _, lv := range c.labeled {
					in.Values = append(in.Values, &types.InputLabeledValue{
						Label: lv.label,
						Value: alloraMath.MustNewBoundedExp40DecFromString(lv.value),
					})
				}
			}

			got, err := keeper.NormalizeInputInference(topic, in)
			if c.wantErr {
				s.Require().Error(err)
				if c.wantErrIs != nil {
					s.Require().True(errorsmod.IsOf(err, c.wantErrIs), "expected error to be %v, got %v", c.wantErrIs, err)
				}
				return
			}
			s.Require().NoError(err)
			s.Require().NotNil(got)

			s.Require().Equal(len(c.wantValuesStr), len(got.Values))
			for i := range c.wantValuesStr {
				s.Require().Equal(c.wantValuesStr[i], got.Values[i].String())
			}
		})
	}
}

// TestGetWorkersLatestInferencesByTopicIdValuesMaterializedAtClose exercises
// the v2 close-time materializer. It replaces the deleted
// TestGetWorkersLatestInferencesByTopicIdValuesPadded: instead of reading
// from a legacy committed-Inference store and padding against a registry
// that was registered during the WSW, the materializer projects each
// worker's raw staged InputInference through a registry that was freshly
// built from activeInfererLabelRefCount.
//
//nolint:exhaustruct
func (s *KeeperTestSuite) TestGetWorkersLatestInferencesByTopicIdValuesPadded() {
	topicId := keeper.TopicId(1)
	nonce := int64(1)

	type tc struct {
		name         string
		outputArity  types.TopicOutputArity
		setup        func(ctx context.Context, k *keeper.Keeper, topicId uint64, nonce int64, w1, w2 string)
		workersOrder []int
		wantErrIs    error
		wantValues   map[int][]string
	}

	mustBounded := func(x string) alloraMath.BoundedExp40Dec {
		return alloraMath.MustNewBoundedExp40DecFromString(x)
	}

	setTopic := func(ctx context.Context, k *keeper.Keeper, topicId uint64, arity types.TopicOutputArity) {
		topic, err := k.GetTopicKeeper().GetTopic(ctx, topicId)
		s.Require().NoError(err)
		topic.OutputArity = arity
		topic.RequireUnity = false
		topic.UnityTolerance = alloraMath.ZeroDec()
		err = k.GetTopicKeeper().SetTopic(ctx, topicId, topic)
		s.Require().NoError(err)
	}

	// stageInput stages a raw InputInference and bumps each label's refcount
	// in one shot, mimicking what the msgserver does when a worker is
	// admitted into the active set.
	stageInput := func(
		ctx context.Context,
		k *keeper.Keeper,
		topicId uint64,
		nonce int64,
		inferer string,
		scalar alloraMath.BoundedExp40Dec,
		values []*types.InputLabeledValue,
	) {
		in := types.InputInference{
			TopicId:     topicId,
			BlockHeight: nonce,
			Inferer:     inferer,
			Value:       scalar,
			Values:      values,
		}
		err := k.GetWorkerKeeper().SetWorkerLatestInputInference(ctx, topicId, inferer, in)
		s.Require().NoError(err)
		labels := make([]string, 0, len(values))
		for _, lv := range values {
			if lv == nil || lv.Label == "" {
				continue
			}
			labels = append(labels, lv.Label)
		}
		if len(labels) > 0 {
			err := k.GetTopicKeeper().IncrementLabelRefCount(ctx, topicId, nonce, labels)
			s.Require().NoError(err)
		}
	}

	cases := []tc{
		{
			name:        "SINGLE_scalar_only_populates_values_len1",
			outputArity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			setup: func(ctx context.Context, k *keeper.Keeper, topicId uint64, nonce int64, w1, w2 string) {
				stageInput(ctx, k, topicId, nonce, w1, mustBounded("42"), nil)
			},
			workersOrder: []int{0},
			wantValues: map[int][]string{
				0: {"42"},
			},
		},
		{
			name:        "SINGLE_values_len1_used_over_scalar",
			outputArity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			setup: func(ctx context.Context, k *keeper.Keeper, topicId uint64, nonce int64, w1, w2 string) {
				stageInput(ctx, k, topicId, nonce, w1, mustBounded("999"), []*types.InputLabeledValue{
					{Label: "y", Value: mustBounded("7")},
				})
			},
			workersOrder: []int{0},
			wantValues: map[int][]string{
				0: {"7"},
			},
		},
		{
			name:        "SINGLE_rejects_values_len_gt_1",
			outputArity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			setup: func(ctx context.Context, k *keeper.Keeper, topicId uint64, nonce int64, w1, w2 string) {
				stageInput(ctx, k, topicId, nonce, w1, mustBounded("1"), []*types.InputLabeledValue{
					{Label: "y", Value: mustBounded("1")},
					{Label: "z", Value: mustBounded("2")},
				})
			},
			workersOrder: []int{0},
			wantErrIs:    sdkerrors.ErrLogic,
		},
		{
			name:        "SINGLE_two_workers_scalar_only_sorted_by_inferer",
			outputArity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			setup: func(ctx context.Context, k *keeper.Keeper, topicId uint64, nonce int64, w1, w2 string) {
				stageInput(ctx, k, topicId, nonce, w1, mustBounded("42"), nil)
				stageInput(ctx, k, topicId, nonce, w2, mustBounded("7"), nil)
			},
			workersOrder: []int{1, 0},
			wantValues: map[int][]string{
				0: {"42"},
				1: {"7"},
			},
		},
		{
			name:        "MULTI_registry_union_pads_missing_labels_to_zero",
			outputArity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			setup: func(ctx context.Context, k *keeper.Keeper, topicId uint64, nonce int64, w1, w2 string) {
				// w1 sees only {a,b,c}, w2 sees {a,b,d}. After close-time
				// lex-sort of the union, the registry is [a,b,c,d] with
				// ids 1..4; each worker's missing labels are filled with 0.
				stageInput(ctx, k, topicId, nonce, w1, mustBounded("0"), []*types.InputLabeledValue{
					{Label: "a", Value: mustBounded("1")},
					{Label: "b", Value: mustBounded("2")},
					{Label: "c", Value: mustBounded("3")},
				})
				stageInput(ctx, k, topicId, nonce, w2, mustBounded("0"), []*types.InputLabeledValue{
					{Label: "a", Value: mustBounded("10")},
					{Label: "b", Value: mustBounded("20")},
					{Label: "d", Value: mustBounded("40")},
				})
			},
			workersOrder: []int{1, 0},
			wantValues: map[int][]string{
				0: {"1", "2", "3", "0"},
				1: {"10", "20", "0", "40"},
			},
		},
		{
			name:        "MULTI_empty_active_set_errors_empty_registry",
			outputArity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			setup: func(ctx context.Context, k *keeper.Keeper, topicId uint64, nonce int64, w1, w2 string) {
				// Nothing staged, nothing incremented => refcount store
				// empty => BuildFinalEpochLabelRegistryFromActiveSet returns
				// types.ErrEpochLabelRegistryEmpty which this test consumes
				// indirectly.
			},
			workersOrder: []int{0},
			wantErrIs:    types.ErrEpochLabelRegistryEmpty,
		},
	}

	for _, c := range cases {
		s.Run(c.name, func() {
			s.SetupTest()

			ctx := s.Ctx()
			k := s.EmissionsKeeper()

			w1 := s.AddrsStr(0)
			w2 := s.AddrsStr(1)
			workers := []string{w1, w2}

			setTopic(ctx, k, topicId, c.outputArity)

			if c.setup != nil {
				c.setup(ctx, k, topicId, nonce, w1, w2)
			}

			reqWorkers := make([]string, 0, len(c.workersOrder))
			for _, idx := range c.workersOrder {
				reqWorkers = append(reqWorkers, workers[idx])
			}

			topic, err := k.GetTopicKeeper().GetTopic(ctx, topicId)
			s.Require().NoError(err)

			reg, err := k.GetTopicKeeper().BuildFinalEpochLabelRegistryFromActiveSet(ctx, topic, nonce)
			if c.wantErrIs != nil && errorsmod.IsOf(err, c.wantErrIs) {
				return
			}
			s.Require().NoError(err)

			got, err := k.GetWorkerKeeper().GetWorkersLatestInferencesByTopicIdValuesMaterializedAtClose(ctx, topic, nonce, reqWorkers, reg)
			if c.wantErrIs != nil {
				s.Require().ErrorIs(err, c.wantErrIs)
				return
			}
			s.Require().NoError(err)
			s.Require().NotNil(got)

			for workerIdx, want := range c.wantValues {
				addr := workers[workerIdx]
				var found *types.Inference
				for _, inf := range got.Inferences {
					if inf.Inferer == addr {
						found = inf
						break
					}
				}
				s.Require().NotNil(found)
				s.Require().Equal(len(want), len(found.Values))
				for i := range want {
					s.Require().Equal(want[i], found.Values[i].String())
				}
			}
		})
	}
}

// setupMultiTopic creates a topic and flips it to MULTI arity (with
// RequireUnity disabled) so that BuildFinalEpochLabelRegistryFromActiveSet
// runs through the MULTI branch. Returns the topic and its id.
func (s *KeeperTestSuite) setupMultiTopic() (types.Topic, uint64) {
	ctx := s.Ctx()
	tk := s.TopicKeeper()
	topicId := s.CreateTopic()
	topic, err := tk.GetTopic(ctx, topicId)
	s.Require().NoError(err)
	topic.OutputArity = types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI
	topic.RequireUnity = false
	topic.UnityTolerance = alloraMath.ZeroDec()
	s.Require().NoError(tk.SetTopic(ctx, topicId, topic))
	return topic, topicId
}

// TestBuildFinalEpochLabelRegistryFromActiveSet_MultiDuplicateLabelDedupes
// pins the invariant that BuildFinalEpochLabelRegistryFromActiveSet produces
// exactly one registry entry when multiple workers stage the same canonical
// label. In v1 this was enforced by RegisterEpochLabel's scan-and-reuse loop;
// in v2 it falls out of the refcount store's keying (topicId, nonce, label)
// so each IncrementLabelRefCount bumps a single row instead of appending.
func (s *KeeperTestSuite) TestBuildFinalEpochLabelRegistryFromActiveSet_MultiDuplicateLabelDedupes() {
	ctx := s.Ctx()
	tk := s.TopicKeeper()

	topic, topicId := s.setupMultiTopic()
	nonce := types.BlockHeight(7)

	s.Require().NoError(tk.IncrementLabelRefCount(ctx, topicId, nonce, []string{"UP"}))
	s.Require().NoError(tk.IncrementLabelRefCount(ctx, topicId, nonce, []string{"UP"}))

	reg, err := tk.BuildFinalEpochLabelRegistryFromActiveSet(ctx, topic, nonce)
	s.Require().NoError(err)
	s.Require().Len(reg.Labels, 1)
	s.Require().Equal(uint32(1), reg.Labels[0].Id)
	s.Require().Equal("UP", reg.Labels[0].Name)

	gotID, ok, err := tk.GetEpochLabelId(ctx, topicId, nonce, "UP")
	s.Require().NoError(err)
	s.Require().True(ok)
	s.Require().Equal(keeper.LabelId(1), gotID)
}

// TestBuildFinalEpochLabelRegistryFromActiveSet_MultiLookupHelpersRoundTrip
// exercises GetEpochLabelId / GetEpochLabelName against a freshly materialized
// registry. The "hit" cases pin the id<->name round-trip for canonical labels
// in lex order, and the "miss" cases pin the no-error / ok=false contract that
// the materializer relies on when a worker submits labels that were not part
// of the frozen registry.
func (s *KeeperTestSuite) TestBuildFinalEpochLabelRegistryFromActiveSet_MultiLookupHelpersRoundTrip() {
	ctx := s.Ctx()
	tk := s.TopicKeeper()

	topic, topicId := s.setupMultiTopic()
	nonce := types.BlockHeight(7)

	s.Require().NoError(tk.IncrementLabelRefCount(ctx, topicId, nonce, []string{"a", "b"}))

	reg, err := tk.BuildFinalEpochLabelRegistryFromActiveSet(ctx, topic, nonce)
	s.Require().NoError(err)
	s.Require().Len(reg.Labels, 2)
	s.Require().Equal(uint32(1), reg.Labels[0].Id)
	s.Require().Equal("a", reg.Labels[0].Name)
	s.Require().Equal(uint32(2), reg.Labels[1].Id)
	s.Require().Equal("b", reg.Labels[1].Name)

	gotID, ok, err := tk.GetEpochLabelId(ctx, topicId, nonce, "a")
	s.Require().NoError(err)
	s.Require().True(ok)
	s.Require().Equal(keeper.LabelId(1), gotID)

	gotName, ok, err := tk.GetEpochLabelName(ctx, topicId, nonce, keeper.LabelId(2))
	s.Require().NoError(err)
	s.Require().True(ok)
	s.Require().Equal("b", gotName)

	_, ok, err = tk.GetEpochLabelId(ctx, topicId, nonce, "MISSING")
	s.Require().NoError(err)
	s.Require().False(ok)

	_, ok, err = tk.GetEpochLabelName(ctx, topicId, nonce, keeper.LabelId(999))
	s.Require().NoError(err)
	s.Require().False(ok)
}

// TestBuildFinalEpochLabelRegistryFromActiveSet_ReleasesLabelsOnEviction
// simulates the active->inactive eviction path that
// trackActiveInfererLabelsOnRemoval triggers inside AppendInference. Two
// MULTI workers are admitted, then the second worker's labels are
// decremented directly (mimicking eviction within the WSW). BuildFinal must
// surface only the remaining workers' labels - in particular, the label
// that was unique to the evicted worker must not appear.
func (s *KeeperTestSuite) TestBuildFinalEpochLabelRegistryFromActiveSet_ReleasesLabelsOnEviction() {
	ctx := s.Ctx()
	tk := s.TopicKeeper()

	topic, topicId := s.setupMultiTopic()
	nonce := types.BlockHeight(7)

	s.Require().NoError(tk.IncrementLabelRefCount(ctx, topicId, nonce, []string{"a", "b", "c"}))
	s.Require().NoError(tk.IncrementLabelRefCount(ctx, topicId, nonce, []string{"a", "b", "d"}))

	s.Require().NoError(tk.DecrementLabelRefCount(ctx, topicId, nonce, []string{"a", "b", "d"}))

	countD, err := tk.GetLabelRefCount(ctx, topicId, nonce, "d")
	s.Require().NoError(err)
	s.Require().Equal(uint64(0), countD)

	visited := make(map[string]uint64)
	s.Require().NoError(tk.IterateLabelsForNonce(ctx, topicId, nonce, func(label string, c uint64) (bool, error) {
		visited[label] = c
		return false, nil
	}))
	s.Require().NotContains(visited, "d", "evicted-worker-unique label must be deleted, not zero-valued")

	reg, err := tk.BuildFinalEpochLabelRegistryFromActiveSet(ctx, topic, nonce)
	s.Require().NoError(err)
	s.Require().Len(reg.Labels, 3)
	s.Require().Equal("a", reg.Labels[0].Name)
	s.Require().Equal(uint32(1), reg.Labels[0].Id)
	s.Require().Equal("b", reg.Labels[1].Name)
	s.Require().Equal(uint32(2), reg.Labels[1].Id)
	s.Require().Equal("c", reg.Labels[2].Name)
	s.Require().Equal(uint32(3), reg.Labels[2].Id)
}

// TestTrackActiveInfererLabelsOnAdmission_NoopForSingleArity pins that the
// admission-side refcount tracker (IncrementStagedLabelRefCount) is a
// no-op on SINGLE topics. The invariant matters because in v2 SINGLE
// topics short-circuit BuildFinal to a 1-entry {Id:1, Name:"y"} registry
// without ever consulting activeInfererLabelRefCount; writing refcount
// rows for SINGLE would be wasted state and could mislead future
// MULTI-only callers that iterate the refcount store.
func (s *KeeperTestSuite) TestTrackActiveInfererLabelsOnAdmission_NoopForSingleArity() {
	ctx := s.Ctx()
	tk := s.TopicKeeper()
	wk := s.WorkerKeeper()

	topicId := s.CreateTopic()
	topic, err := tk.GetTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE, topic.OutputArity)

	nonce := types.BlockHeight(7)
	inferer := s.AddrsStr(0)

	in := types.InputInference{
		TopicId:     topicId,
		BlockHeight: nonce,
		Inferer:     inferer,
		ExtraData:   nil,
		Proof:       "",
		Value:       alloraMath.MustNewBoundedExp40DecFromString("1"),
		Values: []*types.InputLabeledValue{
			{Label: "y", Value: alloraMath.MustNewBoundedExp40DecFromString("1")},
		},
	}
	s.Require().NoError(wk.SetWorkerLatestInputInference(ctx, topicId, inferer, in))

	s.Require().NoError(wk.IncrementStagedLabelRefCount(ctx, topic, nonce, inferer))

	count, err := tk.GetLabelRefCount(ctx, topicId, nonce, "y")
	s.Require().NoError(err)
	s.Require().Equal(uint64(0), count, "SINGLE topic must not write refcount rows on admission")
}

// TestBuildFinalEpochLabelRegistryFromActiveSet_InvalidInputs exercises the
// three fast-fail validation branches at the top of the materializer so
// regressions that silently accept malformed (topic, nonce) cannot go
// unnoticed.
func (s *KeeperTestSuite) TestBuildFinalEpochLabelRegistryFromActiveSet_InvalidInputs() {
	ctx := s.Ctx()
	tk := s.TopicKeeper()

	{
		bad := types.Topic{Id: 0, OutputArity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE} //nolint:exhaustruct
		_, err := tk.BuildFinalEpochLabelRegistryFromActiveSet(ctx, bad, 7)
		s.Require().Error(err)
	}

	{
		topic, _ := s.setupMultiTopic()
		_, err := tk.BuildFinalEpochLabelRegistryFromActiveSet(ctx, topic, -1)
		s.Require().Error(err)
	}

	{
		topicId := s.CreateTopic()
		topic, err := tk.GetTopic(ctx, topicId)
		s.Require().NoError(err)
		topic.OutputArity = types.TopicOutputArity_TOPIC_OUTPUT_ARITY_UNSPECIFIED
		_, err = tk.BuildFinalEpochLabelRegistryFromActiveSet(ctx, topic, 7)
		s.Require().Error(err)
		s.Require().ErrorIs(err, sdkerrors.ErrInvalidRequest)
	}
}

// TestBuildFinalEpochLabelRegistryFromActiveSet_IdempotentRebuild pins two
// consensus-relevant determinism guarantees:
//  1. Running BuildFinal twice against an unchanged refcount store yields
//     byte-equal registries (same label order, same id assignment). Any
//     non-determinism here would break replay.
//  2. After a label is decremented to zero, a rebuild produces a compacted
//     registry with ids reassigned 1..L (no "hole" at the removed label's
//     former slot).
func (s *KeeperTestSuite) TestBuildFinalEpochLabelRegistryFromActiveSet_IdempotentRebuild() {
	ctx := s.Ctx()
	tk := s.TopicKeeper()

	topic, topicId := s.setupMultiTopic()
	nonce := types.BlockHeight(7)

	s.Require().NoError(tk.IncrementLabelRefCount(ctx, topicId, nonce, []string{"a", "b", "c"}))

	reg1, err := tk.BuildFinalEpochLabelRegistryFromActiveSet(ctx, topic, nonce)
	s.Require().NoError(err)
	reg2, err := tk.BuildFinalEpochLabelRegistryFromActiveSet(ctx, topic, nonce)
	s.Require().NoError(err)
	s.Require().Equal(reg1, reg2, "repeated BuildFinal with unchanged refcount must be byte-equal")

	s.Require().Len(reg1.Labels, 3)
	s.Require().Equal([]string{"a", "b", "c"},
		[]string{reg1.Labels[0].Name, reg1.Labels[1].Name, reg1.Labels[2].Name})
	s.Require().Equal([]uint32{1, 2, 3},
		[]uint32{reg1.Labels[0].Id, reg1.Labels[1].Id, reg1.Labels[2].Id})

	s.Require().NoError(tk.DecrementLabelRefCount(ctx, topicId, nonce, []string{"b"}))

	reg3, err := tk.BuildFinalEpochLabelRegistryFromActiveSet(ctx, topic, nonce)
	s.Require().NoError(err)
	s.Require().Len(reg3.Labels, 2)
	s.Require().Equal("a", reg3.Labels[0].Name)
	s.Require().Equal(uint32(1), reg3.Labels[0].Id)
	s.Require().Equal("c", reg3.Labels[1].Name)
	s.Require().Equal(uint32(2), reg3.Labels[1].Id, "ids must be reassigned contiguously after compaction")
}
