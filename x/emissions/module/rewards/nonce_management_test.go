package rewards_test

import (
	"cosmossdk.io/collections"
	cosmosMath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	alloraMath "github.com/allora-network/allora-chain/math"
	actorutils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// Test defer execution of CloseReputerNonce
func (s *RewardsTestSuite) TestCloseReputerNonceTest_DeferExecWhenError() {
	currentBlockHeight := int64(10)
	s.ctx = s.ctx.WithBlockHeight(currentBlockHeight)

	// Create topic
	createTopicReq := &types.CreateNewTopicRequest{
		Creator:                  s.addrsStr[0],
		Metadata:                 "test-topic-close-nonce",
		LossMethod:               "mse",
		EpochLength:              10800,
		AllowNegative:            false,
		GroundTruthLag:           10800,
		WorkerSubmissionWindow:   10,
		AlphaRegret:              alloraMath.NewDecFromInt64(1),
		PNorm:                    alloraMath.NewDecFromInt64(3),
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		EnableWorkerWhitelist:    true,
		EnableReputerWhitelist:   true,
	}
	res, err := s.msgServer.CreateNewTopic(s.ctx, createTopicReq)
	s.Require().NoError(err)
	topicId := res.TopicId

	// Reputer Addresses
	reputerIndexes := returnIndexes(0, 5)
	workerIndexes := returnIndexes(5, 5)

	// Register and add stakes for reputers
	for _, index := range reputerIndexes {
		registerReq := &types.RegisterRequest{
			Sender:    s.addrsStr[index],
			TopicId:   topicId,
			IsReputer: true,
			Owner:     s.addrsStr[index],
		}
		_, err = s.msgServer.Register(s.ctx, registerReq)
		s.Require().NoError(err)
	}

	cosmosOneE18 := inferencesynthesis.CosmosIntOneE18()
	stakeAmount := cosmosMath.NewInt(100).Mul(cosmosOneE18)

	for _, index := range reputerIndexes {
		s.MintTokensToAddress(s.addrs[index], stakeAmount)
		_, err := s.msgServer.AddStake(s.ctx, &types.AddStakeRequest{
			Sender:  s.addrsStr[index],
			Amount:  stakeAmount,
			TopicId: topicId,
		})
		s.Require().NoError(err)
	}

	// Register workers
	for _, index := range workerIndexes {
		registerReq := &types.RegisterRequest{
			Sender:    s.addrsStr[index],
			TopicId:   topicId,
			IsReputer: false,
			Owner:     s.addrsStr[index],
		}
		_, err = s.msgServer.Register(s.ctx, registerReq)
		s.Require().NoError(err)
	}

	// Insert unfullfiled nonces
	err = s.emissionsKeeper.AddWorkerNonce(s.ctx, topicId, &types.Nonce{
		BlockHeight: currentBlockHeight,
	})
	s.Require().NoError(err)
	err = s.emissionsKeeper.AddReputerNonce(s.ctx, topicId, &types.Nonce{
		BlockHeight: currentBlockHeight,
	})
	s.Require().NoError(err)

	workerValues := make([]TestWorkerValue, len(workerIndexes))
	for i, index := range workerIndexes {
		workerValues[i] = TestWorkerValue{
			Index: index,
			Value: "100",
		}
	}

	getIndexesFromValues := func(values []TestWorkerValue) []int {
		indexes := make([]int, 0)
		for _, value := range values {
			indexes = append(indexes, value.Index)
		}
		return indexes
	}

	workerIndexesFromValues := getIndexesFromValues(workerValues)

	// Insert inference from workers
	inferenceBundles := generateSimpleWorkerDataBundles(s, topicId, currentBlockHeight, currentBlockHeight, workerValues, workerIndexesFromValues)
	for _, payload := range inferenceBundles {
		_, err = s.msgServer.InsertWorkerPayload(s.ctx, &types.InsertWorkerPayloadRequest{
			Sender:           payload.Worker,
			WorkerDataBundle: payload,
		})
		s.Require().NoError(err)
	}

	topic, err := s.emissionsKeeper.GetTopic(s.ctx, topicId)
	s.Require().NoError(err)

	// Move to end of worker submission window
	s.ctx = s.ctx.WithBlockHeight(currentBlockHeight + topic.WorkerSubmissionWindow)
	err = actorutils.CloseWorkerNonce(&s.emissionsKeeper, s.ctx, topic, *inferenceBundles[0].Nonce)
	s.Require().NoError(err)

	newBlockheight := currentBlockHeight + topic.GroundTruthLag
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(newBlockheight)
	// Trigger end block - rewards distribution
	err = s.emissionsKeeper.SetRewardCurrentBlockEmission(s.ctx, cosmosMath.NewInt(100))
	s.Require().NoError(err)
	err = s.emissionsAppModule.EndBlock(s.ctx)
	s.Require().NoError(err)

	// Insert loss bundle from reputer
	// Use different indexes to enforce different workers are used
	// This will trigger an error and test if the defer execution of CloseReputerNonce works properly
	workerIndexes = returnIndexes(10, 5)
	lossBundles := s.generateLossBundles(currentBlockHeight, topicId, reputerIndexes)
	for _, payload := range lossBundles.ReputerValueBundles {
		_, _ = s.emissionsKeeper.FulfillWorkerNonce(s.ctx, topicId, payload.ValueBundle.ReputerRequestNonce.ReputerNonce)
		_ = s.emissionsKeeper.AddReputerNonce(s.ctx, topicId, payload.ValueBundle.ReputerRequestNonce.ReputerNonce)
		_, err = s.msgServer.InsertReputerPayload(s.ctx, &types.InsertReputerPayloadRequest{
			Sender:             payload.ValueBundle.Reputer,
			ReputerValueBundle: payload,
		})
		s.Require().NoError(err)
	}

	// before closing the nonce, the nonce should be unfulfilled
	unfulfilled, err := s.emissionsKeeper.IsReputerNonceUnfulfilled(s.ctx, topicId, lossBundles.ReputerValueBundles[0].ValueBundle.ReputerRequestNonce.ReputerNonce)
	s.Require().NoError(err)
	s.Require().True(unfulfilled)

	// before closing the nonce, the active reputers for topic should not be
	activeReputers, err := s.emissionsKeeper.GetActiveReputersForTopic(s.ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(len(reputerIndexes), len(activeReputers))

	// before closing the nonce, the submissions for the topic should not be empty
	for _, bundle := range lossBundles.ReputerValueBundles {
		submissions, err := s.emissionsKeeper.GetReputerLatestLossByTopicId(s.ctx, topicId, bundle.ValueBundle.Reputer)
		s.Require().NoError(err)
		s.Require().NotNil(submissions)
	}

	err = actorutils.CloseReputerNonce(
		&s.emissionsKeeper, s.ctx, topic,
		*lossBundles.ReputerValueBundles[0].ValueBundle.ReputerRequestNonce.ReputerNonce,
	)
	s.Require().Error(err)

	// Check if reputer nonce is fulfilled
	unfulfilled, err = s.emissionsKeeper.IsReputerNonceUnfulfilled(s.ctx, topicId, lossBundles.ReputerValueBundles[0].ValueBundle.ReputerRequestNonce.ReputerNonce)
	s.Require().NoError(err)
	s.Require().False(unfulfilled)

	// Check if the active reputers for topic have been reset
	activeReputers, err = s.emissionsKeeper.GetActiveReputersForTopic(s.ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(0, len(activeReputers))

	// Check if the submissions for the topic have been reset
	for _, bundle := range lossBundles.ReputerValueBundles {
		_, err := s.emissionsKeeper.GetReputerLatestLossByTopicId(s.ctx, topicId, bundle.ValueBundle.Reputer)
		s.Require().ErrorIs(err, collections.ErrNotFound)
	}
}

// Test defer execution of CloseWorkerNonce
func (s *RewardsTestSuite) TestCloseWorkerNonce_DeferExecWhenError() {
	currentBlockHeight := int64(20)
	s.ctx = s.ctx.WithBlockHeight(currentBlockHeight)

	// Create topic
	createTopicReq := &types.CreateNewTopicRequest{
		Creator:                  s.addrsStr[0],
		Metadata:                 "test-topic-close-worker-nonce",
		LossMethod:               "mse",
		EpochLength:              10800,
		AllowNegative:            false,
		GroundTruthLag:           10800,
		WorkerSubmissionWindow:   10,
		AlphaRegret:              alloraMath.NewDecFromInt64(1),
		PNorm:                    alloraMath.NewDecFromInt64(3),
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		EnableWorkerWhitelist:    true,
		EnableReputerWhitelist:   true,
	}
	res, err := s.msgServer.CreateNewTopic(s.ctx, createTopicReq)
	s.Require().NoError(err)
	topicId := res.TopicId

	workerIndexes := returnIndexes(0, 5)

	// Register workers
	for _, index := range workerIndexes {
		registerReq := &types.RegisterRequest{
			Sender:    s.addrsStr[index],
			TopicId:   topicId,
			IsReputer: false,
			Owner:     s.addrsStr[index],
		}
		_, err = s.msgServer.Register(s.ctx, registerReq)
		s.Require().NoError(err)
	}

	// Insert unfulfilled worker nonce
	err = s.emissionsKeeper.AddWorkerNonce(s.ctx, topicId, &types.Nonce{
		BlockHeight: currentBlockHeight,
	})
	s.Require().NoError(err)

	workerValues := make([]TestWorkerValue, len(workerIndexes))
	for i, index := range workerIndexes {
		workerValues[i] = TestWorkerValue{
			Index: index,
			Value: "100",
		}
	}

	getIndexesFromValues := func(values []TestWorkerValue) []int {
		indexes := make([]int, 0)
		for _, value := range values {
			indexes = append(indexes, value.Index)
		}
		return indexes
	}

	workerIndexesFromValues := getIndexesFromValues(workerValues)

	// Insert inference from workers
	inferenceBundles := generateSimpleWorkerDataBundles(s, topicId, currentBlockHeight, currentBlockHeight, workerValues, workerIndexesFromValues)
	for _, payload := range inferenceBundles {
		_, err = s.msgServer.InsertWorkerPayload(s.ctx, &types.InsertWorkerPayloadRequest{
			Sender:           payload.Worker,
			WorkerDataBundle: payload,
		})
		s.Require().NoError(err)
	}

	topic, err := s.emissionsKeeper.GetTopic(s.ctx, topicId)
	s.Require().NoError(err)

	// Move to end of worker submission window
	s.ctx = s.ctx.WithBlockHeight(currentBlockHeight + topic.WorkerSubmissionWindow)

	// Before closing, check nonce is unfulfilled, active workers exist, and submissions exist
	unfulfilled, err := s.emissionsKeeper.IsWorkerNonceUnfulfilled(s.ctx, topicId, inferenceBundles[0].Nonce)
	s.Require().NoError(err)
	s.Require().True(unfulfilled)

	activeInferers, err := s.emissionsKeeper.GetActiveInferersForTopic(s.ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(len(workerIndexes), len(activeInferers))

	for _, bundle := range inferenceBundles {
		submissions, err := s.emissionsKeeper.GetWorkerLatestInferenceByTopicId(s.ctx, topicId, bundle.Worker)
		s.Require().NoError(err)
		s.Require().NotNil(submissions)
	}

	// Enforcing that the active inferer to create an error
	enforcedInferer := s.addrsStr[10]
	err = s.emissionsKeeper.AddActiveInferer(s.ctx, topicId, enforcedInferer)
	s.Require().NoError(err)

	// Call CloseWorkerNonce, expecting an error
	err = actorutils.CloseWorkerNonce(&s.emissionsKeeper, s.ctx, topic, *inferenceBundles[0].Nonce)
	s.Require().Error(err)

	// After closing, check nonce is fulfilled, active workers are reset, and submissions are cleared
	unfulfilled, err = s.emissionsKeeper.IsWorkerNonceUnfulfilled(s.ctx, topicId, inferenceBundles[0].Nonce)
	s.Require().NoError(err)
	s.Require().False(unfulfilled)

	activeInferers, err = s.emissionsKeeper.GetActiveInferersForTopic(s.ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(0, len(activeInferers))

	for _, bundle := range inferenceBundles {
		_, err := s.emissionsKeeper.GetWorkerLatestInferenceByTopicId(s.ctx, topicId, bundle.Worker)
		s.Require().ErrorIs(err, collections.ErrNotFound)
	}
}
