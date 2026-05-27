package keeper_test

import (
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

// TestGetWorkerLatestInferenceByTopicId exercises the live temporary inference
// store used during the worker submission window.
func (s *KeeperTestSuite) TestGetWorkerLatestInferenceByTopicId() {
	ctx := s.Ctx()
	k := s.WorkerKeeper()

	topicId := uint64(1)
	workerAccStr := "allo1xy0pf5hq85j873glav6aajkvtennmg3fpu3cec"

	_, err := k.GetWorkerLatestInferenceByTopicId(ctx, topicId, workerAccStr)
	s.Require().Error(err, "Retrieving an inference that does not exist should result in an error")

	blockHeight1 := int64(12345)
	firstInference := types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeight1,
		Inferer:     workerAccStr,
		Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString("10")},
		ExtraData:   []byte("data"),
		Proof:       "proof123",
	}
	err = k.InsertInference(ctx, topicId, firstInference)
	s.Require().NoError(err, "Staging the first inference should not fail")

	blockHeight2 := int64(12346)
	secondInference := types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeight2,
		Inferer:     workerAccStr,
		Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString("11")},
		ExtraData:   []byte("data"),
		Proof:       "proof123",
	}
	err = k.InsertInference(ctx, topicId, secondInference)
	s.Require().NoError(err, "Overwriting the staged inference should not fail")

	retrievedInference, err := k.GetWorkerLatestInferenceByTopicId(ctx, topicId, workerAccStr)
	s.Require().NoError(err, "Retrieving an existing inference should not fail")
	s.Require().Equal(secondInference, retrievedInference, "Retrieved inference should match the latest staged inference")
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
		admitted, err := k.AppendInference(ctx, topic, nonce.BlockHeight, inference, params.MaxTopInferersToReward)
		s.Require().NoError(err)
		s.Require().True(admitted)
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
	admitted, err := k.AppendInference(ctx, topic, nonce.BlockHeight, &newInference, params.MaxTopInferersToReward)
	s.Require().NoError(err)
	s.Require().True(admitted)
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

	admitted, err = k.AppendInference(ctx, topic, nonce.BlockHeight, &newInference2, params.MaxTopInferersToReward)
	s.Require().NoError(err)
	s.Require().True(admitted)
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
	_, err = k.AppendInference(ctx, topic, nonce.BlockHeight, &newInference2, params.MaxTopInferersToReward)
	s.Require().Error(err, types.ErrCantUpdateEmaMoreThanOncePerWindow.Error())
	// Confirm no change in EMA score
	updateAttemptForWorker2, err := s.ScoresKeeper().GetInfererScoreEma(ctx, topicId, worker2)
	s.Require().NoError(err)
	updateAttemptForWorker2Val, err := updateAttemptForWorker2.Score.Int64()
	s.Require().NoError(err)
	s.Require().Equal(updateAttemptForWorker2Val, updatedWorker2ScoreVal, "unchanged score")
	s.Require().Equal(updateAttemptForWorker2.BlockHeight, updatedWorker2Score.BlockHeight, "unchanged height")
}

//nolint:exhaustruct
func (s *KeeperTestSuite) TestPlanAppendInference() {
	type tc struct {
		name      string
		setup     func(ctx sdk.Context, topicId uint64, inferer string) uint64
		wantKind  keeper.InferenceAdmissionKind
		wantFirst bool
		wantErr   bool
	}

	cases := []tc{
		{
			name: "already_active_inferer_errors",
			setup: func(ctx sdk.Context, topicId uint64, inferer string) uint64 {
				s.Require().NoError(s.WorkerKeeper().AddActiveInferer(ctx, topicId, inferer))
				return 1
			},
			wantErr: true,
		},
		{
			name:      "first_submission_with_open_slot_plans_admission",
			setup:     func(sdk.Context, uint64, string) uint64 { return 2 },
			wantKind:  keeper.InferenceAdmissionOpenSlot,
			wantFirst: true,
		},
		{
			name: "experienced_worker_with_open_slot_plans_admission",
			setup: func(ctx sdk.Context, topicId uint64, inferer string) uint64 {
				score := types.Score{TopicId: topicId, BlockHeight: 1, Address: inferer, Score: alloraMath.NewDecFromInt64(50)}
				s.Require().NoError(s.ScoresKeeper().SetInfererScoreEma(ctx, topicId, inferer, score))
				return 2
			},
			wantKind: keeper.InferenceAdmissionOpenSlot,
		},
		{
			name: "full_active_set_and_lower_score_plans_not_admitted",
			setup: func(ctx sdk.Context, topicId uint64, inferer string) uint64 {
				activeInferer := s.AddrsStr(1)
				activeScore := types.Score{TopicId: topicId, BlockHeight: 1, Address: activeInferer, Score: alloraMath.NewDecFromInt64(100)}
				candidateScore := types.Score{TopicId: topicId, BlockHeight: 1, Address: inferer, Score: alloraMath.NewDecFromInt64(10)}
				s.Require().NoError(s.ScoresKeeper().SetInfererScoreEma(ctx, topicId, activeInferer, activeScore))
				s.Require().NoError(s.ScoresKeeper().SetInfererScoreEma(ctx, topicId, inferer, candidateScore))
				s.Require().NoError(s.ScoresKeeper().SetLowestInfererScoreEma(ctx, topicId, activeScore))
				s.Require().NoError(s.WorkerKeeper().AddActiveInferer(ctx, topicId, activeInferer))
				return 1
			},
			wantKind: keeper.InferenceAdmissionNotAdmitted,
		},
		{
			name: "full_active_set_and_higher_score_plans_eviction",
			setup: func(ctx sdk.Context, topicId uint64, inferer string) uint64 {
				activeInferer := s.AddrsStr(1)
				activeScore := types.Score{TopicId: topicId, BlockHeight: 1, Address: activeInferer, Score: alloraMath.NewDecFromInt64(10)}
				candidateScore := types.Score{TopicId: topicId, BlockHeight: 1, Address: inferer, Score: alloraMath.NewDecFromInt64(100)}
				s.Require().NoError(s.ScoresKeeper().SetInfererScoreEma(ctx, topicId, activeInferer, activeScore))
				s.Require().NoError(s.ScoresKeeper().SetInfererScoreEma(ctx, topicId, inferer, candidateScore))
				s.Require().NoError(s.ScoresKeeper().SetLowestInfererScoreEma(ctx, topicId, activeScore))
				s.Require().NoError(s.WorkerKeeper().AddActiveInferer(ctx, topicId, activeInferer))
				return 1
			},
			wantKind: keeper.InferenceAdmissionEvictLowest,
		},
		{
			name:      "zero_max_top_inferers_plans_not_admitted",
			setup:     func(sdk.Context, uint64, string) uint64 { return 0 },
			wantKind:  keeper.InferenceAdmissionNotAdmitted,
			wantFirst: true,
		},
	}

	for _, c := range cases {
		s.Run(c.name, func() {
			s.SetupTest()
			ctx := s.Ctx()
			topicId := s.CreateTopic()
			topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
			s.Require().NoError(err)
			nonce := types.BlockHeight(10)
			inferer := s.AddrsStr(0)

			maxTopInferersToReward := c.setup(ctx, topicId, inferer)
			activeBefore, err := s.WorkerKeeper().GetActiveInferersForTopic(ctx, topicId)
			s.Require().NoError(err)
			scoreBefore, err := s.ScoresKeeper().GetInfererScoreEma(ctx, topicId, inferer)
			s.Require().NoError(err)
			registryBefore, err := s.TopicKeeper().GetEpochLabelRegistry(ctx, topicId, nonce)
			s.Require().NoError(err)

			plan, err := s.WorkerKeeper().PlanAppendInference(ctx, topic, nonce, inferer, maxTopInferersToReward)
			if c.wantErr {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)
			s.Require().Equal(c.wantKind, plan.Kind)
			s.Require().Equal(c.wantFirst, plan.FirstSubmission)
			s.Require().Equal(inferer, plan.Inferer)

			activeAfter, err := s.WorkerKeeper().GetActiveInferersForTopic(ctx, topicId)
			s.Require().NoError(err)
			s.Require().Equal(activeBefore, activeAfter)
			scoreAfter, err := s.ScoresKeeper().GetInfererScoreEma(ctx, topicId, inferer)
			s.Require().NoError(err)
			s.Require().Equal(scoreBefore, scoreAfter)
			registryAfter, err := s.TopicKeeper().GetEpochLabelRegistry(ctx, topicId, nonce)
			s.Require().NoError(err)
			s.Require().Equal(registryBefore, registryAfter)
		})
	}
}

//nolint:exhaustruct
func (s *KeeperTestSuite) TestCommitPlannedInference() {
	makeInference := func(topicId uint64, nonce types.BlockHeight, inferer string, value string) *types.Inference {
		return &types.Inference{
			TopicId:     topicId,
			BlockHeight: nonce,
			Inferer:     inferer,
			Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString(value)},
		}
	}

	type tc struct {
		name            string
		setup           func(ctx sdk.Context, topicId uint64, nonce types.BlockHeight, inferer string) (keeper.InferenceAdmissionPlan, *types.Inference)
		assert          func(ctx sdk.Context, topicId uint64, inferer string)
		wantErr         bool
		errIsReq        bool
		wantErrContains string
	}

	cases := []tc{
		{
			name: "open_slot_adds_active_inferer_inserts_inference_and_updates_lowest",
			setup: func(_ sdk.Context, topicId uint64, nonce types.BlockHeight, inferer string) (keeper.InferenceAdmissionPlan, *types.Inference) {
				score := types.Score{TopicId: topicId, BlockHeight: nonce, Address: inferer, Score: alloraMath.NewDecFromInt64(50)}
				return keeper.InferenceAdmissionPlan{
					Kind:             keeper.InferenceAdmissionOpenSlot,
					Inferer:          inferer,
					PreviousEmaScore: score,
					WorkerAddresses:  []string{},
					FirstSubmission:  true,
				}, makeInference(topicId, nonce, inferer, "1")
			},
			assert: func(ctx sdk.Context, topicId uint64, inferer string) {
				isActive, err := s.WorkerKeeper().IsActiveInferer(ctx, topicId, inferer)
				s.Require().NoError(err)
				s.Require().True(isActive)
				got, err := s.WorkerKeeper().GetWorkerLatestInferenceByTopicId(ctx, topicId, inferer)
				s.Require().NoError(err)
				s.Require().Equal("1", got.Values[0].String())
				lowest, found, err := s.ScoresKeeper().GetLowestInfererScoreEma(ctx, topicId)
				s.Require().NoError(err)
				s.Require().True(found)
				s.Require().Equal(inferer, lowest.Address)
			},
		},
		{
			name: "eviction_removes_lowest_active_inference_and_adds_candidate",
			setup: func(ctx sdk.Context, topicId uint64, nonce types.BlockHeight, inferer string) (keeper.InferenceAdmissionPlan, *types.Inference) {
				evicted := s.AddrsStr(1)
				evictedScore := types.Score{TopicId: topicId, BlockHeight: 1, Address: evicted, Score: alloraMath.NewDecFromInt64(10)}
				candidateScore := types.Score{TopicId: topicId, BlockHeight: nonce, Address: inferer, Score: alloraMath.NewDecFromInt64(100)}
				s.Require().NoError(s.ScoresKeeper().SetPreviousTopicQuantileInfererScoreEma(ctx, topicId, alloraMath.NewDecFromInt64(1000)))
				s.Require().NoError(s.ScoresKeeper().SetInfererScoreEma(ctx, topicId, evicted, evictedScore))
				s.Require().NoError(s.ScoresKeeper().SetLowestInfererScoreEma(ctx, topicId, evictedScore))
				s.Require().NoError(s.WorkerKeeper().AddActiveInferer(ctx, topicId, evicted))
				s.Require().NoError(s.WorkerKeeper().InsertInference(ctx, topicId, *makeInference(topicId, nonce, evicted, "0.1")))
				return keeper.InferenceAdmissionPlan{
					Kind:             keeper.InferenceAdmissionEvictLowest,
					Inferer:          inferer,
					PreviousEmaScore: candidateScore,
					LowestEmaScore:   evictedScore,
					WorkerAddresses:  []string{evicted},
				}, makeInference(topicId, nonce, inferer, "1")
			},
			assert: func(ctx sdk.Context, topicId uint64, inferer string) {
				isActive, err := s.WorkerKeeper().IsActiveInferer(ctx, topicId, inferer)
				s.Require().NoError(err)
				s.Require().True(isActive)
				evictedActive, err := s.WorkerKeeper().IsActiveInferer(ctx, topicId, s.AddrsStr(1))
				s.Require().NoError(err)
				s.Require().False(evictedActive)
				_, err = s.WorkerKeeper().GetWorkerLatestInferenceByTopicId(ctx, topicId, s.AddrsStr(1))
				s.Require().Error(err)
				got, err := s.WorkerKeeper().GetWorkerLatestInferenceByTopicId(ctx, topicId, inferer)
				s.Require().NoError(err)
				s.Require().Equal("1", got.Values[0].String())
			},
		},
		{
			name: "open_slot_keeps_existing_lowest_when_candidate_score_is_higher",
			setup: func(ctx sdk.Context, topicId uint64, nonce types.BlockHeight, inferer string) (keeper.InferenceAdmissionPlan, *types.Inference) {
				activeInferer := s.AddrsStr(1)
				lowestScore := types.Score{TopicId: topicId, BlockHeight: 1, Address: activeInferer, Score: alloraMath.NewDecFromInt64(10)}
				candidateScore := types.Score{TopicId: topicId, BlockHeight: nonce, Address: inferer, Score: alloraMath.NewDecFromInt64(50)}
				s.Require().NoError(s.ScoresKeeper().SetInfererScoreEma(ctx, topicId, activeInferer, lowestScore))
				s.Require().NoError(s.ScoresKeeper().SetLowestInfererScoreEma(ctx, topicId, lowestScore))
				s.Require().NoError(s.WorkerKeeper().AddActiveInferer(ctx, topicId, activeInferer))
				return keeper.InferenceAdmissionPlan{
					Kind:             keeper.InferenceAdmissionOpenSlot,
					Inferer:          inferer,
					PreviousEmaScore: candidateScore,
					LowestEmaScore:   lowestScore,
					WorkerAddresses:  []string{activeInferer},
				}, makeInference(topicId, nonce, inferer, "1")
			},
			assert: func(ctx sdk.Context, topicId uint64, inferer string) {
				isActive, err := s.WorkerKeeper().IsActiveInferer(ctx, topicId, inferer)
				s.Require().NoError(err)
				s.Require().True(isActive)
				lowest, found, err := s.ScoresKeeper().GetLowestInfererScoreEma(ctx, topicId)
				s.Require().NoError(err)
				s.Require().True(found)
				s.Require().Equal(s.AddrsStr(1), lowest.Address)
				s.Require().Equal("10", lowest.Score.String())
			},
		},
		{
			name: "not_admitted_existing_worker_updates_passive_ema_without_inference",
			setup: func(ctx sdk.Context, topicId uint64, nonce types.BlockHeight, inferer string) (keeper.InferenceAdmissionPlan, *types.Inference) {
				score := types.Score{TopicId: topicId, BlockHeight: nonce, Address: inferer, Score: alloraMath.NewDecFromInt64(10)}
				s.Require().NoError(s.ScoresKeeper().SetPreviousTopicQuantileInfererScoreEma(ctx, topicId, alloraMath.NewDecFromInt64(1000)))
				return keeper.InferenceAdmissionPlan{
					Kind:             keeper.InferenceAdmissionNotAdmitted,
					Inferer:          inferer,
					PreviousEmaScore: score,
					FirstSubmission:  false,
				}, nil
			},
			assert: func(ctx sdk.Context, topicId uint64, inferer string) {
				isActive, err := s.WorkerKeeper().IsActiveInferer(ctx, topicId, inferer)
				s.Require().NoError(err)
				s.Require().False(isActive)
				_, err = s.WorkerKeeper().GetWorkerLatestInferenceByTopicId(ctx, topicId, inferer)
				s.Require().Error(err)
				score, err := s.ScoresKeeper().GetInfererScoreEma(ctx, topicId, inferer)
				s.Require().NoError(err)
				s.Require().Equal(types.BlockHeight(10), score.BlockHeight)
				s.Require().True(score.Score.Gt(alloraMath.NewDecFromInt64(10)))
			},
		},
		{
			name: "not_admitted_first_submission_sets_initial_ema_without_inference",
			setup: func(_ sdk.Context, topicId uint64, nonce types.BlockHeight, inferer string) (keeper.InferenceAdmissionPlan, *types.Inference) {
				score := types.Score{TopicId: topicId, BlockHeight: nonce, Address: inferer, Score: alloraMath.NewDecFromInt64(25)}
				return keeper.InferenceAdmissionPlan{
					Kind:             keeper.InferenceAdmissionNotAdmitted,
					Inferer:          inferer,
					PreviousEmaScore: score,
					FirstSubmission:  true,
				}, nil
			},
			assert: func(ctx sdk.Context, topicId uint64, inferer string) {
				score, err := s.ScoresKeeper().GetInfererScoreEma(ctx, topicId, inferer)
				s.Require().NoError(err)
				s.Require().Equal("25", score.Score.String())
				_, err = s.WorkerKeeper().GetWorkerLatestInferenceByTopicId(ctx, topicId, inferer)
				s.Require().Error(err)
			},
		},
		{
			name: "mismatched_inference_and_plan_inferer_errors",
			setup: func(_ sdk.Context, topicId uint64, nonce types.BlockHeight, inferer string) (keeper.InferenceAdmissionPlan, *types.Inference) {
				score := types.Score{TopicId: topicId, BlockHeight: nonce, Address: inferer, Score: alloraMath.NewDecFromInt64(50)}
				return keeper.InferenceAdmissionPlan{
					Kind:             keeper.InferenceAdmissionOpenSlot,
					Inferer:          inferer,
					PreviousEmaScore: score,
				}, makeInference(topicId, nonce, s.AddrsStr(1), "1")
			},
			wantErr:  true,
			errIsReq: true,
		},
		{
			name: "open_slot_plan_with_nil_inference_errors",
			setup: func(_ sdk.Context, topicId uint64, nonce types.BlockHeight, inferer string) (keeper.InferenceAdmissionPlan, *types.Inference) {
				score := types.Score{TopicId: topicId, BlockHeight: nonce, Address: inferer, Score: alloraMath.NewDecFromInt64(50)}
				return keeper.InferenceAdmissionPlan{
					Kind:             keeper.InferenceAdmissionOpenSlot,
					Inferer:          inferer,
					PreviousEmaScore: score,
				}, nil
			},
			wantErr:  true,
			errIsReq: true,
		},
		{
			name: "eviction_plan_with_nil_inference_errors",
			setup: func(_ sdk.Context, topicId uint64, nonce types.BlockHeight, inferer string) (keeper.InferenceAdmissionPlan, *types.Inference) {
				score := types.Score{TopicId: topicId, BlockHeight: nonce, Address: inferer, Score: alloraMath.NewDecFromInt64(50)}
				return keeper.InferenceAdmissionPlan{
					Kind:             keeper.InferenceAdmissionEvictLowest,
					Inferer:          inferer,
					PreviousEmaScore: score,
				}, nil
			},
			wantErr:  true,
			errIsReq: true,
		},
		{
			name: "unknown_plan_kind_errors",
			setup: func(_ sdk.Context, topicId uint64, nonce types.BlockHeight, inferer string) (keeper.InferenceAdmissionPlan, *types.Inference) {
				score := types.Score{TopicId: topicId, BlockHeight: nonce, Address: inferer, Score: alloraMath.NewDecFromInt64(50)}
				return keeper.InferenceAdmissionPlan{
					Kind:             keeper.InferenceAdmissionKind(255),
					Inferer:          inferer,
					PreviousEmaScore: score,
				}, nil
			},
			wantErr:  true,
			errIsReq: true,
		},
		{
			name: "empty_plan_inferer_errors",
			setup: func(_ sdk.Context, topicId uint64, nonce types.BlockHeight, _ string) (keeper.InferenceAdmissionPlan, *types.Inference) {
				score := types.Score{TopicId: topicId, BlockHeight: nonce, Address: "", Score: alloraMath.NewDecFromInt64(50)}
				return keeper.InferenceAdmissionPlan{
					Kind:             keeper.InferenceAdmissionNotAdmitted,
					Inferer:          "",
					PreviousEmaScore: score,
					FirstSubmission:  true,
				}, nil
			},
			wantErr:  true,
			errIsReq: true,
		},
		{
			name: "planned_score_address_mismatch_errors",
			setup: func(_ sdk.Context, topicId uint64, nonce types.BlockHeight, inferer string) (keeper.InferenceAdmissionPlan, *types.Inference) {
				score := types.Score{TopicId: topicId, BlockHeight: nonce, Address: s.AddrsStr(1), Score: alloraMath.NewDecFromInt64(50)}
				return keeper.InferenceAdmissionPlan{
					Kind:             keeper.InferenceAdmissionNotAdmitted,
					Inferer:          inferer,
					PreviousEmaScore: score,
					FirstSubmission:  true,
				}, nil
			},
			wantErr:  true,
			errIsReq: true,
		},
		{
			name: "eviction_plan_with_inactive_lowest_inferer_errors",
			setup: func(ctx sdk.Context, topicId uint64, nonce types.BlockHeight, inferer string) (keeper.InferenceAdmissionPlan, *types.Inference) {
				inactiveLowest := s.AddrsStr(1)
				lowestScore := types.Score{TopicId: topicId, BlockHeight: 1, Address: inactiveLowest, Score: alloraMath.NewDecFromInt64(10)}
				candidateScore := types.Score{TopicId: topicId, BlockHeight: nonce, Address: inferer, Score: alloraMath.NewDecFromInt64(100)}
				s.Require().NoError(s.ScoresKeeper().SetPreviousTopicQuantileInfererScoreEma(ctx, topicId, alloraMath.NewDecFromInt64(1000)))
				s.Require().NoError(s.ScoresKeeper().SetInfererScoreEma(ctx, topicId, inactiveLowest, lowestScore))
				s.Require().NoError(s.ScoresKeeper().SetLowestInfererScoreEma(ctx, topicId, lowestScore))
				return keeper.InferenceAdmissionPlan{
					Kind:             keeper.InferenceAdmissionEvictLowest,
					Inferer:          inferer,
					PreviousEmaScore: candidateScore,
					LowestEmaScore:   lowestScore,
					WorkerAddresses:  []string{inactiveLowest},
				}, makeInference(topicId, nonce, inferer, "1")
			},
			wantErr:         true,
			wantErrContains: "inferer with lowest score is not active",
		},
		{
			name: "eviction_missing_evicted_inference_documents_remove_behavior",
			setup: func(ctx sdk.Context, topicId uint64, nonce types.BlockHeight, inferer string) (keeper.InferenceAdmissionPlan, *types.Inference) {
				evicted := s.AddrsStr(1)
				evictedScore := types.Score{TopicId: topicId, BlockHeight: 1, Address: evicted, Score: alloraMath.NewDecFromInt64(10)}
				candidateScore := types.Score{TopicId: topicId, BlockHeight: nonce, Address: inferer, Score: alloraMath.NewDecFromInt64(100)}
				s.Require().NoError(s.ScoresKeeper().SetPreviousTopicQuantileInfererScoreEma(ctx, topicId, alloraMath.NewDecFromInt64(1000)))
				s.Require().NoError(s.ScoresKeeper().SetInfererScoreEma(ctx, topicId, evicted, evictedScore))
				s.Require().NoError(s.ScoresKeeper().SetLowestInfererScoreEma(ctx, topicId, evictedScore))
				s.Require().NoError(s.WorkerKeeper().AddActiveInferer(ctx, topicId, evicted))
				return keeper.InferenceAdmissionPlan{
					Kind:             keeper.InferenceAdmissionEvictLowest,
					Inferer:          inferer,
					PreviousEmaScore: candidateScore,
					LowestEmaScore:   evictedScore,
					WorkerAddresses:  []string{evicted},
				}, makeInference(topicId, nonce, inferer, "1")
			},
			assert: func(ctx sdk.Context, topicId uint64, inferer string) {
				isActive, err := s.WorkerKeeper().IsActiveInferer(ctx, topicId, inferer)
				s.Require().NoError(err)
				s.Require().True(isActive)
				evictedActive, err := s.WorkerKeeper().IsActiveInferer(ctx, topicId, s.AddrsStr(1))
				s.Require().NoError(err)
				s.Require().False(evictedActive)
				got, err := s.WorkerKeeper().GetWorkerLatestInferenceByTopicId(ctx, topicId, inferer)
				s.Require().NoError(err)
				s.Require().Equal("1", got.Values[0].String())
			},
		},
		{
			name: "malformed_planned_score_errors_before_outcome_side_effects",
			setup: func(_ sdk.Context, topicId uint64, _ types.BlockHeight, inferer string) (keeper.InferenceAdmissionPlan, *types.Inference) {
				score := types.Score{TopicId: topicId, BlockHeight: -1, Address: inferer, Score: alloraMath.NewDecFromInt64(50)}
				return keeper.InferenceAdmissionPlan{
					Kind:             keeper.InferenceAdmissionOpenSlot,
					Inferer:          inferer,
					PreviousEmaScore: score,
				}, makeInference(topicId, 10, inferer, "1")
			},
			assert: func(ctx sdk.Context, topicId uint64, inferer string) {
				isActive, err := s.WorkerKeeper().IsActiveInferer(ctx, topicId, inferer)
				s.Require().NoError(err)
				s.Require().False(isActive)
				_, err = s.WorkerKeeper().GetWorkerLatestInferenceByTopicId(ctx, topicId, inferer)
				s.Require().Error(err)
			},
			wantErr: true,
		},
		{
			name: "open_slot_malformed_inference_errors_after_adding_active_inferer",
			setup: func(_ sdk.Context, topicId uint64, nonce types.BlockHeight, inferer string) (keeper.InferenceAdmissionPlan, *types.Inference) {
				score := types.Score{TopicId: topicId, BlockHeight: nonce, Address: inferer, Score: alloraMath.NewDecFromInt64(50)}
				inference := makeInference(topicId, nonce, inferer, "1")
				inference.Values = nil
				return keeper.InferenceAdmissionPlan{
					Kind:             keeper.InferenceAdmissionOpenSlot,
					Inferer:          inferer,
					PreviousEmaScore: score,
					WorkerAddresses:  []string{},
					FirstSubmission:  true,
				}, inference
			},
			assert: func(ctx sdk.Context, topicId uint64, inferer string) {
				isActive, err := s.WorkerKeeper().IsActiveInferer(ctx, topicId, inferer)
				s.Require().NoError(err)
				s.Require().True(isActive)
				_, err = s.WorkerKeeper().GetWorkerLatestInferenceByTopicId(ctx, topicId, inferer)
				s.Require().Error(err)
			},
			wantErr: true,
		},
	}

	for _, c := range cases {
		s.Run(c.name, func() {
			s.SetupTest()
			ctx := s.Ctx()
			topicId := s.CreateTopic()
			topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
			s.Require().NoError(err)
			nonce := types.BlockHeight(10)
			inferer := s.AddrsStr(0)

			plan, inference := c.setup(ctx, topicId, nonce, inferer)
			err = s.WorkerKeeper().CommitPlannedInference(ctx, topic, nonce, inference, plan)
			if c.wantErr {
				s.Require().Error(err)
				if c.errIsReq {
					s.Require().True(errorsmod.IsOf(err, sdkerrors.ErrInvalidRequest))
				}
				if c.wantErrContains != "" {
					s.Require().ErrorContains(err, c.wantErrContains)
				}
				if c.assert != nil {
					c.assert(ctx, topicId, inferer)
				}
				return
			}
			s.Require().NoError(err)
			c.assert(ctx, topicId, inferer)
		})
	}
}

//nolint:exhaustruct
func (s *KeeperTestSuite) TestAppendInferenceWrapper() {
	type tc struct {
		name      string
		setup     func(ctx sdk.Context, topicId uint64, nonce types.BlockHeight, inferer string) uint64
		wantAdmit bool
		wantErr   bool
	}

	cases := []tc{
		{
			name:    "nil_inference_errors",
			setup:   func(sdk.Context, uint64, types.BlockHeight, string) uint64 { return 1 },
			wantErr: true,
		},
		{
			name:      "open_slot_admits_and_stores_inference",
			setup:     func(sdk.Context, uint64, types.BlockHeight, string) uint64 { return 1 },
			wantAdmit: true,
		},
		{
			name: "full_active_set_low_score_returns_false_without_storing_inference",
			setup: func(ctx sdk.Context, topicId uint64, _ types.BlockHeight, inferer string) uint64 {
				activeInferer := s.AddrsStr(1)
				activeScore := types.Score{TopicId: topicId, BlockHeight: 1, Address: activeInferer, Score: alloraMath.NewDecFromInt64(100)}
				candidateScore := types.Score{TopicId: topicId, BlockHeight: 1, Address: inferer, Score: alloraMath.NewDecFromInt64(10)}
				s.Require().NoError(s.ScoresKeeper().SetPreviousTopicQuantileInfererScoreEma(ctx, topicId, alloraMath.NewDecFromInt64(1000)))
				s.Require().NoError(s.ScoresKeeper().SetInfererScoreEma(ctx, topicId, activeInferer, activeScore))
				s.Require().NoError(s.ScoresKeeper().SetInfererScoreEma(ctx, topicId, inferer, candidateScore))
				s.Require().NoError(s.ScoresKeeper().SetLowestInfererScoreEma(ctx, topicId, activeScore))
				s.Require().NoError(s.WorkerKeeper().AddActiveInferer(ctx, topicId, activeInferer))
				return 1
			},
		},
	}

	for _, c := range cases {
		s.Run(c.name, func() {
			s.SetupTest()
			ctx := s.Ctx()
			topicId := s.CreateTopic()
			topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
			s.Require().NoError(err)
			nonce := types.BlockHeight(10)
			inferer := s.AddrsStr(0)
			maxTopInferersToReward := c.setup(ctx, topicId, nonce, inferer)

			var inference *types.Inference
			if c.name != "nil_inference_errors" {
				inference = &types.Inference{
					TopicId:     topicId,
					BlockHeight: nonce,
					Inferer:     inferer,
					Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(1)},
				}
			}

			admitted, err := s.WorkerKeeper().AppendInference(ctx, topic, nonce, inference, maxTopInferersToReward)
			if c.wantErr {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)
			s.Require().Equal(c.wantAdmit, admitted)
			_, err = s.WorkerKeeper().GetWorkerLatestInferenceByTopicId(ctx, topicId, inferer)
			if c.wantAdmit {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
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
		admitted, err := k.AppendInference(ctx, topic, nonce.BlockHeight, inference, params.MaxTopInferersToReward)
		s.Require().NoError(err)
		s.Require().True(admitted)
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
		admitted, err := k.AppendInference(ctx, topic, nonce.BlockHeight, inference, params.MaxTopInferersToReward)
		s.Require().NoError(err)
		s.Require().True(admitted)
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
		MaxCanonicalLabelByteLength:         64,
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

// TestRemoveInference verifies that RemoveInference scrubs the worker's
// temporary inference from the live WSW store.
func (s *KeeperTestSuite) TestRemoveInference() {
	ctx := s.Ctx()
	k := s.WorkerKeeper()
	topicId := s.CreateTopic()
	inferer := s.AddrsStr(0)

	inference := types.Inference{
		TopicId:     topicId,
		BlockHeight: 100,
		Inferer:     inferer,
		Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(1)},
		ExtraData:   []byte("data"),
		Proof:       "",
	}

	err := k.InsertInference(ctx, topicId, inference)
	s.Require().NoError(err)

	retrievedInference, err := k.GetWorkerLatestInferenceByTopicId(ctx, topicId, inferer)
	s.Require().NoError(err)
	s.Require().Equal(topicId, retrievedInference.TopicId)
	s.Require().Equal(int64(100), retrievedInference.BlockHeight)
	s.Require().Equal(inferer, retrievedInference.Inferer)
	s.Require().Equal("1", retrievedInference.Values[0].String())

	err = k.RemoveInference(ctx, topicId, inferer)
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

			got, err := s.WorkerKeeper().NewInferenceForecastBundleFromInput(s.Ctx(), topic, validInference.BlockHeight, tt.input)
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

// TestNormalizeInputInference covers the contract for
// NormalizeInputInference: callers MUST have already canonicalized labels
// (via InputInference.ValidateWithLimits). Normalize registers non-default
// labels in the temporary ELR, scatters values into that temporary
// id space, and enforces SINGLE/RequireUnity rules. CloseWorkerNonce later
// filters that temporary ELR and remaps active vectors into final compact ids.
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

		scalarValue       string
		labelDefaultValue string
		labeled           []labeledInput

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
			name:  "MULTI_uses_temporary_first_seen_order",
			arity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			nonce: 1,
			labeled: []labeledInput{
				{label: "b", value: "0.7"},
				{label: "a", value: "0.3"},
			},
			wantValuesStr: []string{"0.7", "0.3"},
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
			name:              "MULTI_rejects_when_all_labels_equal_default_value",
			arity:             types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			nonce:             1,
			labelDefaultValue: "0.5",
			labeled: []labeledInput{
				{label: "a", value: "0.5"},
				{label: "b", value: "0.5"},
			},
			wantErr:   true,
			wantErrIs: sdkerrors.ErrInvalidRequest,
		},
		{
			name:  "MULTI_accepts_one_non_default_label_and_ignores_default_labels",
			arity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			nonce: 1,
			labeled: []labeledInput{
				{label: "a", value: "1"},
				{label: "b", value: "0"},
			},
			wantValuesStr: []string{"1"},
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
			if c.labelDefaultValue != "" {
				topic.LabelDefaultValue = alloraMath.MustNewDecFromString(c.labelDefaultValue)
			}
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

			got, err := s.WorkerKeeper().NormalizeInputInference(ctx, topic, c.nonce, in)
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

// TestCompactRegistryAndRemapInferences exercises the close-time materializer
// that filters the temporary ELR to active non-default labels and remaps dense
// vectors into compact final ids.
//
//nolint:exhaustruct
func (s *KeeperTestSuite) TestCompactRegistryAndRemapInferences() {
	topic := types.Topic{
		Id:                1,
		OutputArity:       types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
		LabelDefaultValue: alloraMath.ZeroDec(),
	} //nolint:exhaustruct
	nonce := types.BlockHeight(7)
	maxLabelBytes := types.DefaultParams().MaxCanonicalLabelByteLength
	tempRegistry := types.EpochLabelRegistry{
		TopicId: 1,
		EpochId: uint64(nonce),
		Labels: []*types.TopicLabel{
			{Id: 1, Name: "b"},
			{Id: 2, Name: "a"},
			{Id: 3, Name: "c"},
			{Id: 4, Name: "d"},
		},
	}

	cases := []struct {
		name       string
		active     []*types.Inference
		wantLabels []string
		wantValues map[string][]string
		wantReuse  bool
		wantErrIs  error
	}{
		{
			name: "compacts_and_remaps_filtered_middle_label",
			active: []*types.Inference{
				{TopicId: 1, BlockHeight: nonce, Inferer: s.AddrsStr(0), Values: decs("1", "2", "0", "4")},
				{TopicId: 1, BlockHeight: nonce, Inferer: s.AddrsStr(1), Values: decs("0", "5")},
			},
			wantLabels: []string{"b", "a", "d"},
			wantValues: map[string][]string{
				s.AddrsStr(0): {"1", "2", "4"},
				s.AddrsStr(1): {"0", "5", "0"},
			},
		},
		{
			name: "reuses_temp_registry_when_all_labels_remain_used_and_pads_short_vectors",
			active: []*types.Inference{
				{TopicId: 1, BlockHeight: nonce, Inferer: s.AddrsStr(0), Values: decs("1")},
				{TopicId: 1, BlockHeight: nonce, Inferer: s.AddrsStr(1), Values: decs("0", "2", "3", "4")},
			},
			wantLabels: []string{"b", "a", "c", "d"},
			wantValues: map[string][]string{
				s.AddrsStr(0): {"1", "0", "0", "0"},
				s.AddrsStr(1): {"0", "2", "3", "4"},
			},
			wantReuse: true,
		},
		{
			name: "errors_when_no_active_non_default_labels_remain",
			active: []*types.Inference{
				{TopicId: 1, BlockHeight: nonce, Inferer: s.AddrsStr(0), Values: decs("0", "0")},
			},
			wantErrIs: types.ErrEpochLabelRegistryEmpty,
		},
		{
			name: "errors_when_active_inference_nil_before_sort",
			active: []*types.Inference{
				{TopicId: 1, BlockHeight: nonce, Inferer: s.AddrsStr(0), Values: decs("1")},
				nil,
			},
			wantErrIs: sdkerrors.ErrLogic,
		},
	}

	for _, c := range cases {
		s.Run(c.name, func() {
			reg, got, reused, err := keeper.CompactRegistryAndRemapInferences(
				topic,
				nonce,
				tempRegistry,
				c.active,
				maxLabelBytes,
			)
			if c.wantErrIs != nil {
				s.Require().True(errorsmod.IsOf(err, c.wantErrIs), "expected error to be %v, got %v", c.wantErrIs, err)
				return
			}
			s.Require().NoError(err)
			s.Require().Equal(c.wantReuse, reused)
			gotLabels := make([]string, 0, len(reg.Labels))
			for _, label := range reg.Labels {
				gotLabels = append(gotLabels, label.Name)
			}
			s.Require().Equal(c.wantLabels, gotLabels)
			for _, inference := range got.Inferences {
				want := c.wantValues[inference.Inferer]
				s.Require().Len(inference.Values, len(want))
				for i := range want {
					s.Require().Equal(want[i], inference.Values[i].String())
				}
			}
		})
	}
}

func (s *KeeperTestSuite) TestMaterializeInputInferenceFromLabelRegistry() {
	nonce := types.BlockHeight(7)
	inferer := s.AddrsStr(0)
	maxLabelBytes := types.DefaultParams().MaxCanonicalLabelByteLength
	baseInference := types.Inference{
		TopicId:     1,
		BlockHeight: nonce,
		Inferer:     inferer,
		ExtraData:   []byte("extra"),
		Proof:       "proof",
		Values:      nil,
	}
	var multiTopic types.Topic
	multiTopic.Id = 1
	multiTopic.OutputArity = types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI
	multiTopic.LabelDefaultValue = alloraMath.ZeroDec()

	var singleTopic types.Topic
	singleTopic.Id = 1
	singleTopic.OutputArity = types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE

	emptyRegistry := types.EpochLabelRegistry{
		TopicId: 0,
		EpochId: 0,
		Labels:  nil,
	}
	registry := types.EpochLabelRegistry{
		TopicId: 1,
		EpochId: uint64(nonce),
		Labels: []*types.TopicLabel{
			{Id: 1, Name: "a"},
			{Id: 2, Name: "b"},
			{Id: 3, Name: "c"},
		},
	}

	cases := []struct {
		name            string
		topic           types.Topic
		registry        types.EpochLabelRegistry
		values          []alloraMath.Dec
		wantScalar      string
		wantLabels      []string
		wantValues      []string
		wantErrContains string
	}{
		{
			name:            "single_value_uses_scalar_field",
			topic:           singleTopic,
			registry:        emptyRegistry,
			values:          decs("0.25"),
			wantScalar:      "0.25",
			wantLabels:      nil,
			wantValues:      nil,
			wantErrContains: "",
		},
		{
			name:            "single_empty_values_rejected",
			topic:           singleTopic,
			registry:        emptyRegistry,
			values:          nil,
			wantScalar:      "",
			wantLabels:      nil,
			wantValues:      nil,
			wantErrContains: "inference values cannot be empty",
		},
		{
			name:            "single_rejects_multiple_values",
			topic:           singleTopic,
			registry:        emptyRegistry,
			values:          decs("0.25", "0.75"),
			wantScalar:      "",
			wantLabels:      nil,
			wantValues:      nil,
			wantErrContains: "expected at most 1",
		},
		{
			name:            "multi_uses_registry_prefix_when_registry_has_grown",
			topic:           multiTopic,
			registry:        registry,
			values:          decs("0.1", "0.2"),
			wantScalar:      "",
			wantLabels:      []string{"a", "b"},
			wantValues:      []string{"0.1", "0.2"},
			wantErrContains: "",
		},
		{
			name:            "multi_empty_vector_rejected",
			topic:           multiTopic,
			registry:        types.EpochLabelRegistry{TopicId: 1, EpochId: uint64(nonce), Labels: nil},
			values:          nil,
			wantScalar:      "",
			wantLabels:      nil,
			wantValues:      nil,
			wantErrContains: "inference values cannot be empty",
		},
		{
			name:            "multi_rejects_vector_longer_than_registry",
			topic:           multiTopic,
			registry:        types.EpochLabelRegistry{TopicId: 1, EpochId: uint64(nonce), Labels: []*types.TopicLabel{{Id: 1, Name: "a"}}},
			values:          decs("0.1", "0.2"),
			wantScalar:      "",
			wantLabels:      nil,
			wantValues:      nil,
			wantErrContains: "temporary registry has 1 labels",
		},
		{
			name:            "multi_rejects_non_contiguous_ids",
			topic:           multiTopic,
			registry:        types.EpochLabelRegistry{TopicId: 1, EpochId: uint64(nonce), Labels: []*types.TopicLabel{{Id: 2, Name: "a"}}},
			values:          decs("0.1"),
			wantScalar:      "",
			wantLabels:      nil,
			wantValues:      nil,
			wantErrContains: "expected 1",
		},
		{
			name:            "multi_rejects_nil_label",
			topic:           multiTopic,
			registry:        types.EpochLabelRegistry{TopicId: 1, EpochId: uint64(nonce), Labels: []*types.TopicLabel{nil}},
			values:          decs("0.1"),
			wantScalar:      "",
			wantLabels:      nil,
			wantValues:      nil,
			wantErrContains: "is nil",
		},
		{
			name:            "multi_rejects_out_of_range_bounded_value",
			topic:           multiTopic,
			registry:        registry,
			values:          decs("1e41"),
			wantScalar:      "",
			wantLabels:      nil,
			wantValues:      nil,
			wantErrContains: "out of bounded range",
		},
	}

	for _, c := range cases {
		s.Run(c.name, func() {
			inference := baseInference
			inference.Values = c.values

			got, err := keeper.MaterializeInputInferenceFromLabelRegistry(
				c.topic,
				c.registry,
				inference,
				maxLabelBytes,
			)
			if c.wantErrContains != "" {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), c.wantErrContains)
				return
			}
			s.Require().NoError(err)
			s.Require().Equal(inference.TopicId, got.TopicId)
			s.Require().Equal(inference.BlockHeight, got.BlockHeight)
			s.Require().Equal(inference.Inferer, got.Inferer)
			s.Require().Equal(inference.ExtraData, got.ExtraData)
			s.Require().Equal(inference.Proof, got.Proof)
			if c.topic.OutputArity == types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE {
				s.Require().Equal(c.wantScalar, got.Value.String())
				s.Require().Empty(got.Values)
				return
			}
			s.Require().Len(got.Values, len(c.wantLabels))
			for i := range c.wantLabels {
				s.Require().Equal(c.wantLabels[i], got.Values[i].Label)
				s.Require().Equal(c.wantValues[i], got.Values[i].Value.String())
			}
		})
	}
}

func (s *KeeperTestSuite) TestSetEpochLabelRegistryUsesLiveLabelByteCap() {
	ctx := s.Ctx()
	topicId := s.CreateTopic()
	nonce := types.BlockHeight(7)

	params := types.DefaultParams()
	params.MaxCanonicalLabelByteLength = 3
	s.Require().NoError(s.ParamsKeeper().SetParams(ctx, params))

	err := s.TopicKeeper().SetEpochLabelRegistry(ctx, types.EpochLabelRegistry{
		TopicId: topicId,
		EpochId: uint64(nonce),
		Labels: []*types.TopicLabel{
			{Id: 1, Name: "abcd"},
		},
	})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "label exceeds 3 bytes")

	err = s.TopicKeeper().SetEpochLabelRegistry(ctx, types.EpochLabelRegistry{
		TopicId: topicId,
		EpochId: uint64(nonce),
		Labels: []*types.TopicLabel{
			{Id: 1, Name: "abc"},
		},
	})
	s.Require().NoError(err)
}

func (s *KeeperTestSuite) TestMaterializersUsePassedLabelByteCap() {
	nonce := types.BlockHeight(7)
	topic, _ := s.setupMultiTopic()
	registry := types.EpochLabelRegistry{
		TopicId: topic.Id,
		EpochId: uint64(nonce),
		Labels: []*types.TopicLabel{
			{Id: 1, Name: "abcd"},
		},
	}
	inference := types.Inference{
		TopicId:     topic.Id,
		BlockHeight: nonce,
		Inferer:     s.AddrsStr(0),
		Values:      decs("0.1"),
		ExtraData:   nil,
		Proof:       "",
	}

	_, err := keeper.MaterializeInputInferenceFromLabelRegistry(topic, registry, inference, 3)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "label exceeds 3 bytes")

	_, _, _, err = keeper.CompactRegistryAndRemapInferences(
		topic,
		nonce,
		registry,
		[]*types.Inference{&inference},
		3,
	)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "label exceeds 3 bytes")
}

func decs(values ...string) []alloraMath.Dec {
	out := make([]alloraMath.Dec, 0, len(values))
	for _, value := range values {
		out = append(out, alloraMath.MustNewDecFromString(value))
	}
	return out
}

// setupMultiTopic creates a topic and flips it to MULTI arity (with
// RequireUnity disabled). Returns the topic and its id.
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

func (s *KeeperTestSuite) TestRegisterEpochLabel_FirstSeenIdempotent() {
	ctx := s.Ctx()
	tk := s.TopicKeeper()

	topic, topicId := s.setupMultiTopic()
	nonce := types.BlockHeight(7)

	idB, err := tk.RegisterEpochLabel(ctx, topicId, topic.LabelCaseSensitive, nonce, "b")
	s.Require().NoError(err)
	idA, err := tk.RegisterEpochLabel(ctx, topicId, topic.LabelCaseSensitive, nonce, "a")
	s.Require().NoError(err)
	idBAgain, err := tk.RegisterEpochLabel(ctx, topicId, topic.LabelCaseSensitive, nonce, "b")
	s.Require().NoError(err)
	s.Require().Equal(keeper.LabelId(1), idB)
	s.Require().Equal(keeper.LabelId(2), idA)
	s.Require().Equal(idB, idBAgain)

	reg, err := tk.GetEpochLabelRegistry(ctx, topicId, nonce)
	s.Require().NoError(err)
	s.Require().Len(reg.Labels, 2)
	s.Require().Equal(uint32(1), reg.Labels[0].Id)
	s.Require().Equal("b", reg.Labels[0].Name)
	s.Require().Equal(uint32(2), reg.Labels[1].Id)
	s.Require().Equal("a", reg.Labels[1].Name)

	gotID, ok, err := tk.GetEpochLabelId(ctx, topicId, nonce, "a")
	s.Require().NoError(err)
	s.Require().True(ok)
	s.Require().Equal(keeper.LabelId(2), gotID)

	gotName, ok, err := tk.GetEpochLabelName(ctx, topicId, nonce, keeper.LabelId(1))
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

func (s *KeeperTestSuite) TestRegisterEpochLabel_UsesTopicCaseSensitivity() {
	ctx := s.Ctx()
	tk := s.TopicKeeper()
	nonce := types.BlockHeight(7)

	caseInsensitiveTopic, caseInsensitiveTopicID := s.setupMultiTopic()
	_, err := tk.RegisterEpochLabel(
		ctx,
		caseInsensitiveTopicID,
		caseInsensitiveTopic.LabelCaseSensitive,
		nonce,
		"Cat",
	)
	s.Require().Error(err)
	s.Require().ErrorContains(err, "label name must be canonical")

	lowerID, err := tk.RegisterEpochLabel(
		ctx,
		caseInsensitiveTopicID,
		caseInsensitiveTopic.LabelCaseSensitive,
		nonce,
		"cat",
	)
	s.Require().NoError(err)
	s.Require().Equal(keeper.LabelId(1), lowerID)

	caseSensitiveTopic, caseSensitiveTopicID := s.setupMultiTopic()
	caseSensitiveTopic.LabelCaseSensitive = true
	s.Require().NoError(tk.SetTopic(ctx, caseSensitiveTopicID, caseSensitiveTopic))
	caseSensitiveTopic, err = tk.GetTopic(ctx, caseSensitiveTopicID)
	s.Require().NoError(err)

	upperID, err := tk.RegisterEpochLabel(
		ctx,
		caseSensitiveTopicID,
		caseSensitiveTopic.LabelCaseSensitive,
		nonce,
		"Cat",
	)
	s.Require().NoError(err)
	s.Require().Equal(keeper.LabelId(1), upperID)

	distinctLowerID, err := tk.RegisterEpochLabel(
		ctx,
		caseSensitiveTopicID,
		caseSensitiveTopic.LabelCaseSensitive,
		nonce,
		"cat",
	)
	s.Require().NoError(err)
	s.Require().Equal(keeper.LabelId(2), distinctLowerID)
}

func (s *KeeperTestSuite) TestRegisterEpochLabel_UsesCanonicalByteLimit() {
	ctx := s.Ctx()
	tk := s.TopicKeeper()

	params, err := s.ParamsKeeper().GetParams(ctx)
	s.Require().NoError(err)
	params.MaxCanonicalLabelByteLength = 3
	s.Require().NoError(s.ParamsKeeper().SetParams(ctx, params))

	topic, topicID := s.setupMultiTopic()
	nonce := types.BlockHeight(7)

	_, err = tk.RegisterEpochLabel(ctx, topicID, topic.LabelCaseSensitive, nonce, "abcd")
	s.Require().Error(err)
	s.Require().ErrorContains(err, "label exceeds 3 bytes")

	id, err := tk.RegisterEpochLabel(ctx, topicID, topic.LabelCaseSensitive, nonce, "abc")
	s.Require().NoError(err)
	s.Require().Equal(keeper.LabelId(1), id)
}

func (s *KeeperTestSuite) TestMaterializeInferencesAndRegistryAtClose_DoesNotPersistFinalRegistry() {
	ctx := s.Ctx()
	topic, topicId := s.setupMultiTopic()
	nonce := types.BlockHeight(7)
	remaining := s.AddrsStr(0)
	evicted := s.AddrsStr(1)
	s.Require().NoError(s.TopicKeeper().SetEpochLabelRegistry(ctx, types.EpochLabelRegistry{
		TopicId: topicId,
		EpochId: uint64(nonce),
		Labels: []*types.TopicLabel{
			{Id: 1, Name: "a"},
			{Id: 2, Name: "b"},
			{Id: 3, Name: "c"},
			{Id: 4, Name: "d"},
		},
	}))
	s.Require().NoError(s.WorkerKeeper().InsertInference(ctx, topicId, types.Inference{
		TopicId:     topicId,
		BlockHeight: nonce,
		Inferer:     remaining,
		Values:      decs("1", "2", "0", "4"),
		ExtraData:   nil,
		Proof:       "",
	}))
	s.Require().NoError(s.WorkerKeeper().InsertInference(ctx, topicId, types.Inference{
		TopicId:     topicId,
		BlockHeight: nonce,
		Inferer:     evicted,
		Values:      decs("0", "0", "3", "0"),
		ExtraData:   nil,
		Proof:       "",
	}))

	got, reg, reused, err := s.WorkerKeeper().MaterializeInferencesAndRegistryAtClose(
		ctx,
		topic,
		nonce,
		[]string{remaining},
	)
	s.Require().NoError(err)
	s.Require().False(reused)
	s.Require().Len(reg.Labels, 3)
	s.Require().Equal([]string{"a", "b", "d"}, []string{reg.Labels[0].Name, reg.Labels[1].Name, reg.Labels[2].Name})
	s.Require().Len(got.Inferences, 1)
	s.Require().Equal([]string{"1", "2", "4"}, []string{
		got.Inferences[0].Values[0].String(),
		got.Inferences[0].Values[1].String(),
		got.Inferences[0].Values[2].String(),
	})

	stored, err := s.TopicKeeper().GetEpochLabelRegistry(ctx, topicId, nonce)
	s.Require().NoError(err)
	s.Require().Len(stored.Labels, 4)
	s.Require().Equal([]string{"a", "b", "c", "d"}, []string{
		stored.Labels[0].Name,
		stored.Labels[1].Name,
		stored.Labels[2].Name,
		stored.Labels[3].Name,
	})
}
