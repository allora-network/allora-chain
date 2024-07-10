package queryserver_test

import (
	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestGetInferencesAtBlock() {
	s.CreateOneTopic()
	ctx := s.ctx
	keeper := s.emissionsKeeper
	queryServer := s.queryServer
	topicId := uint64(1)
	blockHeight := types.BlockHeight(100)
	expectedInferences := types.Inferences{
		Inferences: []*types.Inference{
			{
				TopicId:     topicId,
				BlockHeight: int64(blockHeight),
				Inferer:     "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
				Value:       alloraMath.NewDecFromInt64(1),
			},
			{
				TopicId:     topicId,
				BlockHeight: int64(blockHeight),
				Inferer:     "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
				Value:       alloraMath.NewDecFromInt64(2),
			},
		},
	}

	nonce := types.Nonce{BlockHeight: int64(blockHeight)}
	err := keeper.InsertInferences(ctx, topicId, nonce, expectedInferences)
	s.Require().NoError(err)

	results, err := queryServer.GetInferencesAtBlock(
		ctx,
		&types.QueryInferencesAtBlockRequest{
			TopicId:     topicId,
			BlockHeight: int64(blockHeight),
		},
	)

	s.Require().NoError(err)
	s.Require().Equal(&expectedInferences, results.Inferences)
}

func (s *KeeperTestSuite) TestGetWorkerLatestInferenceByTopicId() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	queryServer := s.queryServer

	topicId := s.CreateOneTopic()
	workerAddress := "allo1xy0pf5hq85j873glav6aajkvtennmg3fpu3cec"
	wrongWorkerAddress := "invalidAddress"

	_, err := sdk.AccAddressFromBech32(workerAddress)
	s.Require().NoError(err, "The worker address should be valid and convertible")

	// Testing non-existent topic
	_, err = queryServer.GetWorkerLatestInferenceByTopicId(
		ctx,
		&types.QueryWorkerLatestInferenceRequest{
			TopicId:       999, // non-existent topic
			WorkerAddress: workerAddress,
		},
	)
	s.Require().Error(err, "Should return an error for non-existent topic")

	// Testing non-existent worker
	_, err = queryServer.GetWorkerLatestInferenceByTopicId(
		ctx,
		&types.QueryWorkerLatestInferenceRequest{
			TopicId:       topicId,
			WorkerAddress: wrongWorkerAddress,
		},
	)
	s.Require().Error(err, "Should return an error for non-existent worker address")

	// Assume a correct insertion happened
	blockHeight := int64(100)
	inference := types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Inferer:     workerAddress,
		Value:       alloraMath.MustNewDecFromString("123.456"),
	}
	inferences := types.Inferences{
		Inferences: []*types.Inference{&inference},
	}
	nonce := types.Nonce{BlockHeight: blockHeight}
	err = keeper.InsertInferences(ctx, topicId, nonce, inferences)
	s.Require().NoError(err, "Inserting inferences should succeed")

	// Testing successful retrieval
	response, err := queryServer.GetWorkerLatestInferenceByTopicId(
		ctx,
		&types.QueryWorkerLatestInferenceRequest{
			TopicId:       topicId,
			WorkerAddress: workerAddress,
		},
	)
	s.Require().NoError(err, "Retrieving latest inference should succeed")
	s.Require().NotNil(response.LatestInference, "Response should contain a latest inference")
	s.Require().Equal(&inference, response.LatestInference, "The latest inference should match the expected data")
}

func (s *KeeperTestSuite) TestGetNetworkInferencesAtBlock() {
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	require := s.Require()
	topicId := s.CreateOneTopic()

	reputer0 := "allo1m5v6rgjtxh4xszrrzqacwjh4ve6r0za2gxx9qr"
	reputer1 := "allo1e7cj9839ht2xm8urynqs5279hrvqd8neusvp2x"
	reputer2 := "allo1k9ss0xfer54nyack5678frl36e5g3rj2yzxtfj"
	reputer3 := "allo18ljxewge4vqrkk09tm5heldqg25yj8d9ekgkw5"
	reputer4 := "allo1k36ljvn8z0u49sagdg46p75psgreh23kdjn3l0"

	blockHeight := int64(10)

	simpleNonce := types.Nonce{BlockHeight: blockHeight}
	reputerRequestNonce := &types.ReputerRequestNonce{
		ReputerNonce: &types.Nonce{BlockHeight: blockHeight},
	}

	// Set Loss bundles

	reputerLossBundles := types.ReputerValueBundles{
		ReputerValueBundles: []*types.ReputerValueBundle{
			{
				ValueBundle: &types.ValueBundle{
					Reputer:             reputer0,
					CombinedValue:       alloraMath.MustNewDecFromString(".0000117005278862668"),
					ReputerRequestNonce: reputerRequestNonce,
					TopicId:             topicId,
				},
			},
			{
				ValueBundle: &types.ValueBundle{
					Reputer:             reputer1,
					CombinedValue:       alloraMath.MustNewDecFromString(".00000962701954026944"),
					ReputerRequestNonce: reputerRequestNonce,
					TopicId:             topicId,
				},
			},
			{
				ValueBundle: &types.ValueBundle{
					Reputer:             reputer2,
					CombinedValue:       alloraMath.MustNewDecFromString(".0000256948644008351"),
					ReputerRequestNonce: reputerRequestNonce,
					TopicId:             topicId,
				},
			},
			{
				ValueBundle: &types.ValueBundle{
					Reputer:             reputer3,
					CombinedValue:       alloraMath.MustNewDecFromString(".0000123986052417188"),
					ReputerRequestNonce: reputerRequestNonce,
					TopicId:             topicId,
				},
			},
			{
				ValueBundle: &types.ValueBundle{
					Reputer:             reputer4,
					CombinedValue:       alloraMath.MustNewDecFromString(".0000115363240547692"),
					ReputerRequestNonce: reputerRequestNonce,
					TopicId:             topicId,
				},
			},
		},
	}

	err := keeper.InsertReputerLossBundlesAtBlock(s.ctx, topicId, blockHeight, reputerLossBundles)
	require.NoError(err)

	// Set Stake
	stake0, ok := cosmosMath.NewIntFromString("210535101370326000000000")
	s.Require().True(ok)
	stake1, ok := cosmosMath.NewIntFromString("216697093951021000000000")
	s.Require().True(ok)
	stake2, ok := cosmosMath.NewIntFromString("161740241803855000000000")
	s.Require().True(ok)
	stake3, ok := cosmosMath.NewIntFromString("394848305052250000000000")
	s.Require().True(ok)
	stake4, ok := cosmosMath.NewIntFromString("206169717590569000000000")
	s.Require().True(ok)
	err = keeper.AddReputerStake(s.ctx, topicId, reputer0, stake0)
	require.NoError(err)
	err = keeper.AddReputerStake(s.ctx, topicId, reputer1, stake1)
	require.NoError(err)
	err = keeper.AddReputerStake(s.ctx, topicId, reputer2, stake2)
	require.NoError(err)
	err = keeper.AddReputerStake(s.ctx, topicId, reputer3, stake3)
	require.NoError(err)
	err = keeper.AddReputerStake(s.ctx, topicId, reputer4, stake4)
	require.NoError(err)

	// Set Inferences

	inferences := types.Inferences{
		Inferences: []*types.Inference{
			{
				Inferer:     reputer0,
				Value:       alloraMath.MustNewDecFromString("-0.035995138925040600"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Inferer:     reputer1,
				Value:       alloraMath.MustNewDecFromString("-0.07333303938740420"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Inferer:     reputer2,
				Value:       alloraMath.MustNewDecFromString("-0.1495482917094790"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Inferer:     reputer3,
				Value:       alloraMath.MustNewDecFromString("-0.12952123274063800"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Inferer:     reputer4,
				Value:       alloraMath.MustNewDecFromString("-0.0703055329498285"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
		},
	}

	err = keeper.InsertInferences(s.ctx, topicId, simpleNonce, inferences)
	s.Require().NoError(err)

	// Set actual block
	s.ctx = s.ctx.WithBlockHeight(blockHeight + 10)

	// Update epoch topic epoch last ended
	err = keeper.UpdateTopicEpochLastEnded(s.ctx, topicId, blockHeight+10)
	require.NoError(err)

	// Test querying the server
	req := &types.QueryNetworkInferencesAtBlockRequest{
		TopicId:                  topicId,
		BlockHeightLastInference: blockHeight,
		BlockHeightLastReward:    blockHeight,
	}
	response, err := queryServer.GetNetworkInferencesAtBlock(s.ctx, req)
	require.NoError(err)
	require.NotNil(response, "Response should not be nil")
}

func (s *KeeperTestSuite) TestGetLatestNetworkInferences() {
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	require := s.Require()
	topicId := s.CreateOneTopic()
	topic, err := keeper.GetTopic(s.ctx, topicId)
	require.NoError(err)

	epochLength := topic.EpochLength
	epochLastEnded := topic.EpochLastEnded

	lossBlockHeight := int64(epochLastEnded)
	inferenceBlockHeight := int64(epochLastEnded + epochLength)

	lossNonce := types.Nonce{BlockHeight: lossBlockHeight}
	inferenceNonce := types.Nonce{BlockHeight: inferenceBlockHeight}

	reputerLossRequestNonce := &types.ReputerRequestNonce{ReputerNonce: &lossNonce}

	s.ctx = s.ctx.WithBlockHeight(lossBlockHeight)

	// Set Loss bundles
	err = keeper.InsertNetworkLossBundleAtBlock(s.ctx, topicId, lossBlockHeight, types.ValueBundle{
		CombinedValue:       alloraMath.MustNewDecFromString("0.00001342819294865661936622664543402969"),
		ReputerRequestNonce: reputerLossRequestNonce,
		TopicId:             topicId,
	})
	require.NoError(err)

	// Set Inferences
	s.ctx = s.ctx.WithBlockHeight(inferenceBlockHeight)

	getWorkerRegretValue := func(value string) types.TimestampedValue {
		return types.TimestampedValue{
			BlockHeight: inferenceBlockHeight,
			Value:       alloraMath.MustNewDecFromString(value),
		}
	}

	worker0 := "allo1s8sar766d54wzlmqhwpwdv0unzjfusjydg3l3j"
	worker1 := "allo1rp9026g0ppp9nwdtzvxpqqhl43yqplrj7pmnhq"
	worker2 := "allo1cmdyvyqgzudlf0ep2nht333a057wg9vwfek7tq"
	worker3 := "allo1cr5usf94ph9w2lpeqfjkv3eyuspv47c0zx3nye"
	worker4 := "allo19dvpcsqqer4xy7cdh4s3gtm460z6xpe2hzlf5s"

	forecaster0 := "allo13hh468ghmmyfjrdwqn567j29wq8sh6pnwff0cn"
	forecaster1 := "allo1nxqgvyt6ggu3dz7uwe8p22sac6v2v8sayhwqvz"
	forecaster2 := "allo1a0sc83cls78g4j5qey5er9zzpjpva4x935aajk"

	keeper.SetInfererNetworkRegret(s.ctx, topicId, worker0, getWorkerRegretValue("0.1"))
	keeper.SetInfererNetworkRegret(s.ctx, topicId, worker1, getWorkerRegretValue("0.2"))
	keeper.SetInfererNetworkRegret(s.ctx, topicId, worker2, getWorkerRegretValue("0.3"))
	keeper.SetInfererNetworkRegret(s.ctx, topicId, worker3, getWorkerRegretValue("0.4"))
	keeper.SetInfererNetworkRegret(s.ctx, topicId, worker4, getWorkerRegretValue("0.5"))

	keeper.SetForecasterNetworkRegret(s.ctx, topicId, forecaster0, getWorkerRegretValue("0.1"))
	keeper.SetForecasterNetworkRegret(s.ctx, topicId, forecaster1, getWorkerRegretValue("0.2"))
	keeper.SetForecasterNetworkRegret(s.ctx, topicId, forecaster2, getWorkerRegretValue("0.3"))

	inferences := types.Inferences{
		Inferences: []*types.Inference{
			{
				Inferer:     worker0,
				Value:       alloraMath.MustNewDecFromString("-0.035995138925040600"),
				TopicId:     topicId,
				BlockHeight: inferenceBlockHeight,
			},
			{
				Inferer:     worker1,
				Value:       alloraMath.MustNewDecFromString("-0.07333303938740420"),
				TopicId:     topicId,
				BlockHeight: inferenceBlockHeight,
			},
			{
				Inferer:     worker2,
				Value:       alloraMath.MustNewDecFromString("-0.1495482917094790"),
				TopicId:     topicId,
				BlockHeight: inferenceBlockHeight,
			},
			{
				Inferer:     worker3,
				Value:       alloraMath.MustNewDecFromString("-0.12952123274063800"),
				TopicId:     topicId,
				BlockHeight: inferenceBlockHeight,
			},
			{
				Inferer:     worker4,
				Value:       alloraMath.MustNewDecFromString("-0.0703055329498285"),
				TopicId:     topicId,
				BlockHeight: inferenceBlockHeight,
			},
		},
	}

	err = keeper.InsertInferences(s.ctx, topicId, inferenceNonce, inferences)
	s.Require().NoError(err)

	// Set Forecasts
	forecasts := types.Forecasts{
		Forecasts: []*types.Forecast{
			{
				Forecaster: forecaster0,
				ForecastElements: []*types.ForecastElement{
					{Inferer: worker0, Value: alloraMath.MustNewDecFromString("0.1")},
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.2")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.3")},
					{Inferer: worker3, Value: alloraMath.MustNewDecFromString("0.4")},
					{Inferer: worker4, Value: alloraMath.MustNewDecFromString("0.5")},
				},
				TopicId:     topicId,
				BlockHeight: inferenceBlockHeight,
			},
			{
				Forecaster: forecaster1,
				ForecastElements: []*types.ForecastElement{
					{Inferer: worker0, Value: alloraMath.MustNewDecFromString("0.5")},
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.4")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.3")},
					{Inferer: worker3, Value: alloraMath.MustNewDecFromString("0.2")},
					{Inferer: worker4, Value: alloraMath.MustNewDecFromString("0.1")},
				},
				TopicId:     topicId,
				BlockHeight: inferenceBlockHeight,
			},
			{
				Forecaster: forecaster2,
				ForecastElements: []*types.ForecastElement{
					{Inferer: worker0, Value: alloraMath.MustNewDecFromString("0.2")},
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.3")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.4")},
					{Inferer: worker3, Value: alloraMath.MustNewDecFromString("0.3")},
					{Inferer: worker4, Value: alloraMath.MustNewDecFromString("0.2")},
				},
				TopicId:     topicId,
				BlockHeight: inferenceBlockHeight,
			},
		},
	}

	err = keeper.InsertForecasts(s.ctx, topicId, inferenceNonce, forecasts)
	require.NoError(err)

	// Update epoch topic epoch last ended
	err = keeper.UpdateTopicEpochLastEnded(s.ctx, topicId, inferenceBlockHeight)
	require.NoError(err)

	// Test querying the server
	req := &types.QueryLatestNetworkInferencesAtBlockRequest{
		TopicId: topicId,
	}
	response, err := queryServer.GetLatestNetworkInference(s.ctx, req)
	require.NoError(err)
	require.NotNil(response, "Response should not be nil")

	require.Equal(len(response.InfererWeights), 5)
	require.Equal(len(response.ForecasterWeights), 3)
	require.Equal(len(response.ForecastImpliedInferences), 3)
}

func (s *KeeperTestSuite) TestGetIsWorkerNonceUnfulfilled() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	newNonce := &types.Nonce{BlockHeight: 42}

	req := &types.QueryIsWorkerNonceUnfulfilledRequest{
		TopicId:     topicId,
		BlockHeight: newNonce.BlockHeight,
	}
	response, err := s.queryServer.GetIsWorkerNonceUnfulfilled(s.ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response, "Response should not be nil")
	s.Require().False(response.IsWorkerNonceUnfulfilled)

	// Set worker nonce
	err = keeper.AddWorkerNonce(ctx, topicId, newNonce)
	s.Require().NoError(err)

	response, err = s.queryServer.GetIsWorkerNonceUnfulfilled(s.ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response, "Response should not be nil")
	s.Require().True(response.IsWorkerNonceUnfulfilled)
}

func (s *KeeperTestSuite) TestGetUnfulfilledWorkerNonces() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	// Initially, ensure no unfulfilled nonces exist
	req := &types.QueryUnfulfilledWorkerNoncesRequest{
		TopicId: topicId,
	}
	response, err := s.queryServer.GetUnfulfilledWorkerNonces(s.ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response, "Response should not be nil")
	s.Require().Len(response.Nonces.Nonces, 0, "Initial unfulfilled nonces should be empty")

	// Set multiple worker nonces
	nonceValues := []int64{42, 43, 44}
	for _, val := range nonceValues {
		err = keeper.AddWorkerNonce(ctx, topicId, &types.Nonce{BlockHeight: val})
		s.Require().NoError(err, "Failed to add worker nonce")
	}

	// Retrieve and verify the nonces
	response, err = s.queryServer.GetUnfulfilledWorkerNonces(s.ctx, req)
	s.Require().NoError(err, "Error retrieving nonces after adding")
	s.Require().Len(response.Nonces.Nonces, len(nonceValues), "Should match the number of added nonces")

	// Check that all the expected nonces are present and correct
	for i, nonce := range response.Nonces.Nonces {
		s.Require().Equal(nonceValues[len(nonceValues)-i-1], nonce.BlockHeight, "Nonce value should match the expected value")
	}
}

func (s *KeeperTestSuite) TestGetInfererNetworkRegret() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	worker := "worker-address"
	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}
	emptyRegret := types.TimestampedValue{
		BlockHeight: 0,
		Value:       alloraMath.NewDecFromInt64(0),
	}

	req := &types.QueryInfererNetworkRegretRequest{
		TopicId: topicId,
		ActorId: worker,
	}
	response, err := s.queryServer.GetInfererNetworkRegret(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(response.NotFound)
	s.Require().Equal(response.Regret, &emptyRegret)

	// Set Inferer Network Regret
	err = keeper.SetInfererNetworkRegret(ctx, topicId, worker, regret)
	s.Require().NoError(err)

	// Get Inferer Network Regret
	response, err = s.queryServer.GetInfererNetworkRegret(s.ctx, req)
	s.Require().NoError(err)
	s.Require().Equal(response.Regret, &regret)
	s.Require().False(response.NotFound)
}

func (s *KeeperTestSuite) TestGetForecasterNetworkRegret() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	worker := "worker-address"
	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}
	emptyRegret := types.TimestampedValue{
		BlockHeight: 0,
		Value:       alloraMath.NewDecFromInt64(0),
	}

	req := &types.QueryForecasterNetworkRegretRequest{
		TopicId: topicId,
		Worker:  worker,
	}
	response, err := s.queryServer.GetForecasterNetworkRegret(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(response.NotFound)
	s.Require().Equal(response.Regret, &emptyRegret)

	// Set Forecaster Network Regret
	err = keeper.SetForecasterNetworkRegret(ctx, topicId, worker, regret)
	s.Require().NoError(err)

	// Get Forecaster Network Regret
	response, err = s.queryServer.GetForecasterNetworkRegret(s.ctx, req)
	s.Require().NoError(err)
	s.Require().Equal(response.Regret, &regret)
	s.Require().False(response.NotFound)
}

func (s *KeeperTestSuite) TestGetOneInForecasterNetworkRegret() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	forecaster := "forecaster-address"
	inferer := "inferer-address"
	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}
	emptyRegret := types.TimestampedValue{
		BlockHeight: 0,
		Value:       alloraMath.NewDecFromInt64(0),
	}

	req := &types.QueryOneInForecasterNetworkRegretRequest{
		TopicId:    topicId,
		Forecaster: forecaster,
		Inferer:    inferer,
	}
	response, err := s.queryServer.GetOneInForecasterNetworkRegret(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(response.NotFound)
	s.Require().Equal(response.Regret, &emptyRegret)

	// Set One In Forecaster Network Regret
	err = keeper.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster, inferer, regret)
	s.Require().NoError(err)

	// Get One In Forecaster Network Regret
	response, err = s.queryServer.GetOneInForecasterNetworkRegret(s.ctx, req)
	s.Require().NoError(err)
	s.Require().Equal(response.Regret, &regret)
	s.Require().False(response.NotFound)
}

func (s *KeeperTestSuite) TestGetOneInForecasterSelfNetworkRegret() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	forecaster := "forecaster-address"
	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}
	emptyRegret := types.TimestampedValue{
		BlockHeight: 0,
		Value:       alloraMath.NewDecFromInt64(0),
	}

	req := &types.QueryOneInForecasterSelfNetworkRegretRequest{
		TopicId:    topicId,
		Forecaster: forecaster,
	}
	response, err := s.queryServer.GetOneInForecasterSelfNetworkRegret(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(response.NotFound)
	s.Require().Equal(response.Regret, &emptyRegret)

	// Set One In Forecaster Self Network Regret
	err = keeper.SetOneInForecasterSelfNetworkRegret(ctx, topicId, forecaster, regret)
	s.Require().NoError(err)

	// Get One In Forecaster Self Network Regret
	response, err = s.queryServer.GetOneInForecasterSelfNetworkRegret(s.ctx, req)
	s.Require().NoError(err)
	s.Require().Equal(response.Regret, &regret)
	s.Require().False(response.NotFound)
}

func (s *KeeperTestSuite) TestGetLatestTopicInferences() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	topicId := s.CreateOneTopic()

	// Initially, there should be no inferences, so we expect an empty result
	req := &types.QueryLatestTopicInferencesRequest{
		TopicId: topicId,
	}
	response, err := s.queryServer.GetLatestTopicInferences(ctx, req)
	s.Require().NoError(err)
	emptyInferences := response.Inferences
	emptyBlockHeight := response.BlockHeight

	s.Require().NoError(err, "Retrieving latest inferences when none exist should not result in an error")
	s.Require().Equal(&types.Inferences{Inferences: []*types.Inference{}}, emptyInferences, "Expected no inferences initially")
	s.Require().Equal(types.BlockHeight(0), emptyBlockHeight, "Expected block height to be zero initially")

	// Insert first set of inferences
	blockHeight1 := types.BlockHeight(12345)
	newInference1 := types.Inference{
		TopicId:     uint64(topicId),
		BlockHeight: blockHeight1,
		Inferer:     "worker1",
		Value:       alloraMath.MustNewDecFromString("10"),
		ExtraData:   []byte("data1"),
		Proof:       "proof1",
	}
	inferences1 := types.Inferences{
		Inferences: []*types.Inference{&newInference1},
	}
	nonce1 := types.Nonce{BlockHeight: blockHeight1}
	err = keeper.InsertInferences(ctx, topicId, nonce1, inferences1)
	s.Require().NoError(err, "Inserting first set of inferences should not fail")

	// Insert second set of inferences
	blockHeight2 := types.BlockHeight(12346)
	newInference2 := types.Inference{
		TopicId:     uint64(topicId),
		BlockHeight: blockHeight2,
		Inferer:     "worker2",
		Value:       alloraMath.MustNewDecFromString("20"),
		ExtraData:   []byte("data2"),
		Proof:       "proof2",
	}
	inferences2 := types.Inferences{
		Inferences: []*types.Inference{&newInference2},
	}
	nonce2 := types.Nonce{BlockHeight: blockHeight2}
	err = keeper.InsertInferences(ctx, topicId, nonce2, inferences2)
	s.Require().NoError(err, "Inserting second set of inferences should not fail")

	// Retrieve the latest inferences
	response, err = s.queryServer.GetLatestTopicInferences(ctx, req)
	s.Require().NoError(err)
	latestInferences := response.Inferences
	latestBlockHeight := response.BlockHeight
	s.Require().NoError(err, "Retrieving latest inferences should not fail")
	s.Require().Equal(&inferences2, latestInferences, "Latest inferences should match the second inserted set")
	s.Require().Equal(blockHeight2, latestBlockHeight, "Latest block height should match the second inserted set")
}
