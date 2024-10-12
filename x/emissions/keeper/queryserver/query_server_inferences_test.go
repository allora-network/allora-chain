package queryserver_test

import (
	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *QueryServerTestSuite) TestGetInferencesAtBlock() {
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
				BlockHeight: blockHeight,
				Inferer:     "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
				Value:       alloraMath.NewDecFromInt64(1),
			},
			{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Inferer:     "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
				Value:       alloraMath.NewDecFromInt64(2),
			},
		},
	}

	nonce := types.Nonce{BlockHeight: blockHeight}
	err := keeper.InsertActiveInferences(ctx, topicId, nonce.BlockHeight, expectedInferences)
	s.Require().NoError(err)

	results, err := queryServer.GetInferencesAtBlock(
		ctx,
		&types.GetInferencesAtBlockRequest{
			TopicId:     topicId,
			BlockHeight: blockHeight,
		},
	)

	s.Require().NoError(err)
	s.Require().Equal(&expectedInferences, results.Inferences)
}

func (s *QueryServerTestSuite) TestGetWorkerLatestInferenceByTopicId() {
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
		&types.GetWorkerLatestInferenceByTopicIdRequest{
			TopicId:       999, // non-existent topic
			WorkerAddress: workerAddress,
		},
	)
	s.Require().Error(err, "Should return an error for non-existent topic")

	// Testing non-existent worker
	_, err = queryServer.GetWorkerLatestInferenceByTopicId(
		ctx,
		&types.GetWorkerLatestInferenceByTopicIdRequest{
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
		ExtraData:   nil,
		Proof:       "",
	}
	err = keeper.InsertInference(ctx, topicId, inference)
	s.Require().NoError(err, "Inserting inferences should succeed")

	// Testing successful retrieval
	response, err := queryServer.GetWorkerLatestInferenceByTopicId(
		ctx,
		&types.GetWorkerLatestInferenceByTopicIdRequest{
			TopicId:       topicId,
			WorkerAddress: workerAddress,
		},
	)
	s.Require().NoError(err, "Retrieving latest inference should succeed")
	s.Require().NotNil(response.LatestInference, "Response should contain a latest inference")
	s.Require().Equal(&inference, response.LatestInference, "The latest inference should match the expected data")
}

func (s *QueryServerTestSuite) TestGetNetworkInferencesAtBlock() {
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	require := s.Require()
	topicId := s.CreateOneTopic()

	reputer0 := s.addrsStr[5]
	reputer1 := s.addrsStr[6]
	reputer2 := s.addrsStr[7]
	reputer3 := s.addrsStr[8]
	reputer4 := s.addrsStr[9]

	blockHeight := int64(10)

	simpleNonce := types.Nonce{BlockHeight: blockHeight}
	reputerRequestNonce := &types.ReputerRequestNonce{
		ReputerNonce: &types.Nonce{BlockHeight: blockHeight},
	}

	// Set Loss bundles

	valueBundle1 := types.ValueBundle{
		TopicId:                       topicId,
		Reputer:                       reputer0,
		ExtraData:                     nil,
		ReputerRequestNonce:           reputerRequestNonce,
		CombinedValue:                 alloraMath.MustNewDecFromString(".0000117005278862668"),
		InfererValues:                 nil,
		ForecasterValues:              nil,
		NaiveValue:                    alloraMath.MustNewDecFromString(".0000117005278862668"),
		OneOutInfererValues:           nil,
		OneOutForecasterValues:        nil,
		OneInForecasterValues:         nil,
		OneOutInfererForecasterValues: nil,
	}
	signature1 := s.signValueBundle(&valueBundle1, s.privKeys[5])

	valueBundle2 := types.ValueBundle{
		TopicId:                       topicId,
		Reputer:                       reputer1,
		ExtraData:                     nil,
		CombinedValue:                 alloraMath.MustNewDecFromString(".00000962701954026944"),
		ReputerRequestNonce:           reputerRequestNonce,
		InfererValues:                 nil,
		ForecasterValues:              nil,
		NaiveValue:                    alloraMath.MustNewDecFromString(".00000962701954026944"),
		OneOutInfererValues:           nil,
		OneOutForecasterValues:        nil,
		OneInForecasterValues:         nil,
		OneOutInfererForecasterValues: nil,
	}
	signature2 := s.signValueBundle(&valueBundle2, s.privKeys[6])
	valueBundle3 := types.ValueBundle{
		Reputer:                       reputer2,
		ExtraData:                     nil,
		CombinedValue:                 alloraMath.MustNewDecFromString(".0000256948644008351"),
		ReputerRequestNonce:           reputerRequestNonce,
		TopicId:                       topicId,
		InfererValues:                 nil,
		ForecasterValues:              nil,
		NaiveValue:                    alloraMath.MustNewDecFromString(".0000256948644008351"),
		OneOutInfererValues:           nil,
		OneOutForecasterValues:        nil,
		OneInForecasterValues:         nil,
		OneOutInfererForecasterValues: nil,
	}
	signature3 := s.signValueBundle(&valueBundle3, s.privKeys[7])
	valueBundle4 := types.ValueBundle{
		Reputer:                       reputer3,
		ExtraData:                     nil,
		CombinedValue:                 alloraMath.MustNewDecFromString(".0000123986052417188"),
		ReputerRequestNonce:           reputerRequestNonce,
		TopicId:                       topicId,
		InfererValues:                 nil,
		ForecasterValues:              nil,
		NaiveValue:                    alloraMath.MustNewDecFromString(".0000123986052417188"),
		OneOutInfererValues:           nil,
		OneOutForecasterValues:        nil,
		OneInForecasterValues:         nil,
		OneOutInfererForecasterValues: nil,
	}
	signature4 := s.signValueBundle(&valueBundle4, s.privKeys[8])
	valueBundle5 := types.ValueBundle{
		Reputer:                       reputer4,
		ExtraData:                     nil,
		CombinedValue:                 alloraMath.MustNewDecFromString(".0000115363240547692"),
		ReputerRequestNonce:           reputerRequestNonce,
		TopicId:                       topicId,
		InfererValues:                 nil,
		ForecasterValues:              nil,
		NaiveValue:                    alloraMath.MustNewDecFromString(".0000115363240547692"),
		OneOutInfererValues:           nil,
		OneOutForecasterValues:        nil,
		OneInForecasterValues:         nil,
		OneOutInfererForecasterValues: nil,
	}
	signature5 := s.signValueBundle(&valueBundle5, s.privKeys[9])
	reputerLossBundles := types.ReputerValueBundles{
		ReputerValueBundles: []*types.ReputerValueBundle{
			{ValueBundle: &valueBundle1, Signature: signature1, Pubkey: s.pubKeyHexStr[5]},
			{ValueBundle: &valueBundle2, Signature: signature2, Pubkey: s.pubKeyHexStr[6]},
			{ValueBundle: &valueBundle3, Signature: signature3, Pubkey: s.pubKeyHexStr[7]},
			{ValueBundle: &valueBundle4, Signature: signature4, Pubkey: s.pubKeyHexStr[8]},
			{ValueBundle: &valueBundle5, Signature: signature5, Pubkey: s.pubKeyHexStr[9]},
		},
	}

	err := keeper.InsertActiveReputerLosses(s.ctx, topicId, blockHeight, reputerLossBundles)
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

	err = keeper.InsertActiveInferences(s.ctx, topicId, simpleNonce.BlockHeight, inferences)
	s.Require().NoError(err)

	// Set actual block
	s.ctx = s.ctx.WithBlockHeight(blockHeight + 10)

	// Update epoch topic epoch last ended
	err = keeper.UpdateTopicEpochLastEnded(s.ctx, topicId, blockHeight+10)
	require.NoError(err)

	// Test querying the server
	req := &types.GetNetworkInferencesAtBlockRequest{
		TopicId:                  topicId,
		BlockHeightLastInference: blockHeight,
	}
	response, err := queryServer.GetNetworkInferencesAtBlock(s.ctx, req)
	require.NoError(err)
	require.NotNil(response, "Response should not be nil")
}

func (s *QueryServerTestSuite) TestGetLatestNetworkInferences() {
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	require := s.Require()
	topicId := s.CreateOneTopic()
	topic, err := keeper.GetTopic(s.ctx, topicId)
	require.NoError(err)

	epochLength := topic.EpochLength
	epochLastEnded := topic.EpochLastEnded

	lossBlockHeight := epochLastEnded
	inferenceBlockHeight := epochLastEnded + epochLength

	lossNonce := types.Nonce{BlockHeight: lossBlockHeight}
	inferenceNonce := types.Nonce{BlockHeight: inferenceBlockHeight}

	reputerLossRequestNonce := &types.ReputerRequestNonce{ReputerNonce: &lossNonce}

	s.ctx = s.ctx.WithBlockHeight(lossBlockHeight)

	// Set Loss bundles
	err = keeper.InsertNetworkLossBundleAtBlock(s.ctx, topicId, lossBlockHeight, types.ValueBundle{
		TopicId:                       topicId,
		ReputerRequestNonce:           reputerLossRequestNonce,
		ExtraData:                     nil,
		Reputer:                       s.addrsStr[8],
		CombinedValue:                 alloraMath.MustNewDecFromString("0.00001342819294865661936622664543402969"),
		InfererValues:                 nil,
		ForecasterValues:              nil,
		NaiveValue:                    alloraMath.MustNewDecFromString("0.00001342819294865661936622664543402969"),
		OneOutInfererValues:           nil,
		OneOutForecasterValues:        nil,
		OneInForecasterValues:         nil,
		OneOutInfererForecasterValues: nil,
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

	err = keeper.SetInfererNetworkRegret(s.ctx, topicId, worker0, getWorkerRegretValue("0.1"))
	require.NoError(err)
	err = keeper.SetInfererNetworkRegret(s.ctx, topicId, worker1, getWorkerRegretValue("0.2"))
	require.NoError(err)
	err = keeper.SetInfererNetworkRegret(s.ctx, topicId, worker2, getWorkerRegretValue("0.3"))
	require.NoError(err)
	err = keeper.SetInfererNetworkRegret(s.ctx, topicId, worker3, getWorkerRegretValue("0.4"))
	require.NoError(err)
	err = keeper.SetInfererNetworkRegret(s.ctx, topicId, worker4, getWorkerRegretValue("0.5"))
	require.NoError(err)

	err = keeper.SetForecasterNetworkRegret(s.ctx, topicId, forecaster0, getWorkerRegretValue("0.1"))
	require.NoError(err)
	err = keeper.SetForecasterNetworkRegret(s.ctx, topicId, forecaster1, getWorkerRegretValue("0.2"))
	require.NoError(err)
	err = keeper.SetForecasterNetworkRegret(s.ctx, topicId, forecaster2, getWorkerRegretValue("0.3"))
	require.NoError(err)

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

	err = keeper.InsertActiveInferences(s.ctx, topicId, inferenceNonce.BlockHeight, inferences)
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

	err = keeper.InsertActiveForecasts(s.ctx, topicId, inferenceNonce.BlockHeight, forecasts)
	require.NoError(err)

	// Update epoch topic epoch last ended
	err = keeper.UpdateTopicEpochLastEnded(s.ctx, topicId, inferenceBlockHeight)
	require.NoError(err)

	// Test querying the server
	req := &types.GetLatestNetworkInferencesRequest{
		TopicId: topicId,
	}
	response, err := queryServer.GetLatestNetworkInferences(s.ctx, req)
	require.NoError(err)
	require.NotNil(response, "Response should not be nil")

	require.Equal(len(response.InfererWeights), 5)
	require.Equal(len(response.ForecasterWeights), 3)
	require.Equal(len(response.ForecastImpliedInferences), 3)
}

func (s *QueryServerTestSuite) TestIsWorkerNonceUnfulfilled() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	newNonce := &types.Nonce{BlockHeight: 42}

	req := &types.IsWorkerNonceUnfulfilledRequest{
		TopicId:     topicId,
		BlockHeight: newNonce.BlockHeight,
	}
	response, err := s.queryServer.IsWorkerNonceUnfulfilled(s.ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response, "Response should not be nil")
	s.Require().False(response.IsWorkerNonceUnfulfilled)

	// Set worker nonce
	err = keeper.AddWorkerNonce(ctx, topicId, newNonce)
	s.Require().NoError(err)

	response, err = s.queryServer.IsWorkerNonceUnfulfilled(s.ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response, "Response should not be nil")
	s.Require().True(response.IsWorkerNonceUnfulfilled)
}

func (s *QueryServerTestSuite) TestGetUnfulfilledWorkerNonces() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	// Initially, ensure no unfulfilled nonces exist
	req := &types.GetUnfulfilledWorkerNoncesRequest{
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

func (s *QueryServerTestSuite) TestGetInfererNetworkRegret() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := s.CreateOneTopic()
	worker := s.addrsStr[1]
	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}
	emptyRegret := types.TimestampedValue{
		BlockHeight: 0,
		Value:       alloraMath.NewDecFromInt64(0),
	}

	req := &types.GetInfererNetworkRegretRequest{
		TopicId: topicId,
		ActorId: worker,
	}
	response, err := s.queryServer.GetInfererNetworkRegret(s.ctx, req)
	s.Require().NoError(err)
	s.Require().Equal(response.Regret, &emptyRegret)

	// Set Inferer Network Regret
	err = keeper.SetInfererNetworkRegret(ctx, topicId, worker, regret)
	s.Require().NoError(err)

	// Get Inferer Network Regret
	response, err = s.queryServer.GetInfererNetworkRegret(s.ctx, req)
	s.Require().NoError(err)
	s.Require().Equal(response.Regret, &regret)
}

func (s *QueryServerTestSuite) TestGetForecasterNetworkRegret() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := s.CreateOneTopic()
	worker := s.addrsStr[1]
	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}
	emptyRegret := types.TimestampedValue{
		BlockHeight: 0,
		Value:       alloraMath.NewDecFromInt64(0),
	}

	req := &types.GetForecasterNetworkRegretRequest{
		TopicId: topicId,
		Worker:  worker,
	}
	response, err := s.queryServer.GetForecasterNetworkRegret(s.ctx, req)
	s.Require().NoError(err)
	s.Require().Equal(response.Regret, &emptyRegret)

	// Set Forecaster Network Regret
	err = keeper.SetForecasterNetworkRegret(ctx, topicId, worker, regret)
	s.Require().NoError(err)

	// Get Forecaster Network Regret
	response, err = s.queryServer.GetForecasterNetworkRegret(s.ctx, req)
	s.Require().NoError(err)
	s.Require().Equal(response.Regret, &regret)
}

func (s *QueryServerTestSuite) TestGetOneInForecasterNetworkRegret() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := s.CreateOneTopic()
	forecaster := s.addrsStr[3]
	inferer := s.addrsStr[1]
	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}
	emptyRegret := types.TimestampedValue{
		BlockHeight: 0,
		Value:       alloraMath.NewDecFromInt64(0),
	}

	req := &types.GetOneInForecasterNetworkRegretRequest{
		TopicId:    topicId,
		Forecaster: forecaster,
		Inferer:    inferer,
	}
	response, err := s.queryServer.GetOneInForecasterNetworkRegret(s.ctx, req)
	s.Require().NoError(err)
	s.Require().Equal(response.Regret, &emptyRegret)

	// Set One In Forecaster Network Regret
	err = keeper.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster, inferer, regret)
	s.Require().NoError(err)

	// Get One In Forecaster Network Regret
	response, err = s.queryServer.GetOneInForecasterNetworkRegret(s.ctx, req)
	s.Require().NoError(err)
	s.Require().Equal(response.Regret, &regret)
}

func (s *QueryServerTestSuite) TestGetLatestTopicInferences() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	topicId := s.CreateOneTopic()

	// Initially, there should be no inferences, so we expect an empty result
	req := &types.GetLatestTopicInferencesRequest{
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
		TopicId:     topicId,
		BlockHeight: blockHeight1,
		Inferer:     s.addrsStr[1],
		Value:       alloraMath.MustNewDecFromString("10"),
		ExtraData:   []byte("data1"),
		Proof:       "proof1",
	}
	inferences1 := types.Inferences{
		Inferences: []*types.Inference{&newInference1},
	}
	nonce1 := types.Nonce{BlockHeight: blockHeight1}
	err = keeper.InsertActiveInferences(ctx, topicId, nonce1.BlockHeight, inferences1)
	s.Require().NoError(err, "Inserting first set of inferences should not fail")

	// Insert second set of inferences
	blockHeight2 := types.BlockHeight(12346)
	newInference2 := types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeight2,
		Inferer:     s.addrsStr[2],
		Value:       alloraMath.MustNewDecFromString("20"),
		ExtraData:   []byte("data2"),
		Proof:       "proof2",
	}
	inferences2 := types.Inferences{
		Inferences: []*types.Inference{&newInference2},
	}
	nonce2 := types.Nonce{BlockHeight: blockHeight2}
	err = keeper.InsertActiveInferences(ctx, topicId, nonce2.BlockHeight, inferences2)
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

func (s *QueryServerTestSuite) TestGetLatestAvailableNetworkInference() {
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	require := s.Require()
	topicId := s.CreateOneTopic()
	topic, err := keeper.GetTopic(s.ctx, topicId)
	require.NoError(err)

	epochLength := topic.EpochLength
	epochLastEnded := topic.EpochLastEnded

	lossBlockHeight := epochLastEnded + epochLength
	inferenceBlockHeight := lossBlockHeight + epochLength
	inferenceBlockHeight2 := inferenceBlockHeight + epochLength

	lossNonce := types.Nonce{BlockHeight: lossBlockHeight}
	inferenceNonce := types.Nonce{BlockHeight: inferenceBlockHeight}
	inferenceNonce2 := types.Nonce{BlockHeight: inferenceBlockHeight2}

	reputerLossRequestNonce := &types.ReputerRequestNonce{ReputerNonce: &lossNonce}

	s.ctx = s.ctx.WithBlockHeight(lossBlockHeight)

	lossBundle := types.ValueBundle{
		CombinedValue:                 alloraMath.MustNewDecFromString("0.00001342819294865661936622664543402969"),
		ReputerRequestNonce:           reputerLossRequestNonce,
		TopicId:                       topicId,
		Reputer:                       s.addrsStr[0],
		ExtraData:                     nil,
		NaiveValue:                    alloraMath.MustNewDecFromString("0.00001342819294865661936622664543402969"),
		InfererValues:                 nil,
		ForecasterValues:              nil,
		OneOutInfererValues:           nil,
		OneOutForecasterValues:        nil,
		OneInForecasterValues:         nil,
		OneOutInfererForecasterValues: nil,
	}
	// Set Loss bundles
	err = keeper.InsertNetworkLossBundleAtBlock(s.ctx, topicId, lossBlockHeight, lossBundle)
	require.NoError(err)

	err = keeper.SetReputerTopicLastCommit(s.ctx, topicId, lossBlockHeight, &lossNonce)
	s.Require().NoError(err)

	// Set Inferences
	s.ctx = s.ctx.WithBlockHeight(inferenceBlockHeight)

	getWorkerRegretValue := func(value string) types.TimestampedValue {
		return types.TimestampedValue{
			BlockHeight: inferenceBlockHeight,
			Value:       alloraMath.MustNewDecFromString(value),
		}
	}

	worker0 := s.addrsStr[1]
	worker1 := s.addrsStr[2]
	worker2 := s.addrsStr[3]
	worker3 := s.addrsStr[4]
	worker4 := s.addrsStr[5]

	forecaster0 := s.addrsStr[6]
	forecaster1 := s.addrsStr[7]
	forecaster2 := s.addrsStr[8]

	err = keeper.SetInfererNetworkRegret(s.ctx, topicId, worker0, getWorkerRegretValue("0.1"))
	require.NoError(err)
	err = keeper.SetInfererNetworkRegret(s.ctx, topicId, worker1, getWorkerRegretValue("0.2"))
	require.NoError(err)
	err = keeper.SetInfererNetworkRegret(s.ctx, topicId, worker2, getWorkerRegretValue("0.3"))
	require.NoError(err)
	err = keeper.SetInfererNetworkRegret(s.ctx, topicId, worker3, getWorkerRegretValue("0.4"))
	require.NoError(err)
	err = keeper.SetInfererNetworkRegret(s.ctx, topicId, worker4, getWorkerRegretValue("0.5"))
	require.NoError(err)

	err = keeper.SetForecasterNetworkRegret(s.ctx, topicId, forecaster0, getWorkerRegretValue("0.1"))
	require.NoError(err)
	err = keeper.SetForecasterNetworkRegret(s.ctx, topicId, forecaster1, getWorkerRegretValue("0.2"))
	require.NoError(err)
	err = keeper.SetForecasterNetworkRegret(s.ctx, topicId, forecaster2, getWorkerRegretValue("0.3"))
	require.NoError(err)

	getInferencesForBlockHeight := func(blockHeight int64) types.Inferences {
		return types.Inferences{
			Inferences: []*types.Inference{
				{
					Inferer:     worker0,
					Value:       alloraMath.MustNewDecFromString("-0.035995138925040600"),
					TopicId:     topicId,
					BlockHeight: blockHeight,
				},
				{
					Inferer:     worker1,
					Value:       alloraMath.MustNewDecFromString("-0.07333303938740420"),
					TopicId:     topicId,
					BlockHeight: blockHeight,
				},
				{
					Inferer:     worker2,
					Value:       alloraMath.MustNewDecFromString("-0.1495482917094790"),
					TopicId:     topicId,
					BlockHeight: blockHeight,
				},
				{
					Inferer:     worker3,
					Value:       alloraMath.MustNewDecFromString("-0.12952123274063800"),
					TopicId:     topicId,
					BlockHeight: blockHeight,
				},
				{
					Inferer:     worker4,
					Value:       alloraMath.MustNewDecFromString("-0.0703055329498285"),
					TopicId:     topicId,
					BlockHeight: blockHeight,
				},
			},
		}
	}

	getForecastsForBlockHeight := func(blockHeight int64) types.Forecasts {
		return types.Forecasts{
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
					BlockHeight: blockHeight,
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
					BlockHeight: blockHeight,
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
					BlockHeight: blockHeight,
				},
			},
		}
	}

	// insert inferences and forecasts 1
	err = keeper.InsertActiveInferences(s.ctx, topicId, inferenceNonce.BlockHeight, getInferencesForBlockHeight(inferenceBlockHeight))
	s.Require().NoError(err)

	err = keeper.InsertActiveForecasts(s.ctx, topicId, inferenceNonce.BlockHeight, getForecastsForBlockHeight(inferenceBlockHeight))
	require.NoError(err)

	err = keeper.SetWorkerTopicLastCommit(s.ctx, topicId, inferenceBlockHeight, &inferenceNonce)
	s.Require().NoError(err)

	// insert inferences and forecasts 2
	err = keeper.InsertActiveInferences(s.ctx, topicId, inferenceNonce2.BlockHeight, getInferencesForBlockHeight(inferenceBlockHeight2))
	s.Require().NoError(err)

	err = keeper.InsertActiveForecasts(s.ctx, topicId, inferenceNonce2.BlockHeight, getForecastsForBlockHeight(inferenceBlockHeight2))
	require.NoError(err)

	err = keeper.SetWorkerTopicLastCommit(s.ctx, topicId, inferenceBlockHeight2, &inferenceNonce2)
	s.Require().NoError(err)

	// Update epoch topic epoch last ended
	err = keeper.UpdateTopicEpochLastEnded(s.ctx, topicId, inferenceBlockHeight2)
	require.NoError(err)

	// Test querying the server
	req := &types.GetLatestAvailableNetworkInferencesRequest{
		TopicId: topicId,
	}
	response, err := queryServer.GetLatestAvailableNetworkInferences(s.ctx, req)
	require.NoError(err)
	require.NotNil(response, "Response should not be nil")

	// should be 4 since we would be looking at inferences from a previous block
	require.Equal(len(response.InfererWeights), 5)
	require.Equal(len(response.ForecasterWeights), 3)
	require.Equal(len(response.ForecastImpliedInferences), 3)
	require.Equal(len(response.ConfidenceIntervalRawPercentiles), 5)
	require.Equal(len(response.ConfidenceIntervalValues), 5)

	require.Equal(response.InferenceBlockHeight, inferenceBlockHeight2)
	require.Equal(response.LossBlockHeight, lossBlockHeight)
}

func (s *QueryServerTestSuite) TestTestGetLatestAvailableNetworkInferenceWithMissingInferences() {
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	require := s.Require()
	topicId := s.CreateOneTopic()
	topic, err := keeper.GetTopic(s.ctx, topicId)
	require.NoError(err)

	epochLength := topic.EpochLength
	epochLastEnded := topic.EpochLastEnded

	lossBlockHeight := epochLastEnded
	inferenceBlockHeight := epochLastEnded + epochLength
	inferenceBlockHeight2 := inferenceBlockHeight + epochLength

	lossNonce := types.Nonce{BlockHeight: lossBlockHeight}
	// inferenceNonce := types.Nonce{BlockHeight: inferenceBlockHeight}
	inferenceNonce2 := types.Nonce{BlockHeight: inferenceBlockHeight2}

	reputerLossRequestNonce := &types.ReputerRequestNonce{ReputerNonce: &lossNonce}

	s.ctx = s.ctx.WithBlockHeight(lossBlockHeight)

	// Set Loss bundles
	lossBundle := types.ValueBundle{
		CombinedValue:                 alloraMath.MustNewDecFromString("0.00001342819294865661936622664543402969"),
		ReputerRequestNonce:           reputerLossRequestNonce,
		TopicId:                       topicId,
		Reputer:                       s.addrsStr[0],
		ExtraData:                     nil,
		NaiveValue:                    alloraMath.MustNewDecFromString("0.00001342819294865661936622664543402969"),
		InfererValues:                 nil,
		ForecasterValues:              nil,
		OneOutInfererValues:           nil,
		OneOutForecasterValues:        nil,
		OneInForecasterValues:         nil,
		OneOutInfererForecasterValues: nil,
	}
	err = keeper.InsertNetworkLossBundleAtBlock(s.ctx, topicId, lossBlockHeight, lossBundle)
	require.NoError(err)

	// Set Inferences
	s.ctx = s.ctx.WithBlockHeight(inferenceBlockHeight)

	getWorkerRegretValue := func(value string) types.TimestampedValue {
		return types.TimestampedValue{
			BlockHeight: inferenceBlockHeight,
			Value:       alloraMath.MustNewDecFromString(value),
		}
	}

	worker0 := s.addrsStr[1]
	worker1 := s.addrsStr[2]
	worker2 := s.addrsStr[3]
	worker3 := s.addrsStr[4]
	worker4 := s.addrsStr[5]

	forecaster0 := s.addrsStr[6]
	forecaster1 := s.addrsStr[7]
	forecaster2 := s.addrsStr[8]

	err = keeper.SetInfererNetworkRegret(s.ctx, topicId, worker0, getWorkerRegretValue("0.1"))
	require.NoError(err)
	err = keeper.SetInfererNetworkRegret(s.ctx, topicId, worker1, getWorkerRegretValue("0.2"))
	require.NoError(err)
	err = keeper.SetInfererNetworkRegret(s.ctx, topicId, worker2, getWorkerRegretValue("0.3"))
	require.NoError(err)
	err = keeper.SetInfererNetworkRegret(s.ctx, topicId, worker3, getWorkerRegretValue("0.4"))
	require.NoError(err)
	err = keeper.SetInfererNetworkRegret(s.ctx, topicId, worker4, getWorkerRegretValue("0.5"))
	require.NoError(err)

	err = keeper.SetForecasterNetworkRegret(s.ctx, topicId, forecaster0, getWorkerRegretValue("0.1"))
	require.NoError(err)
	err = keeper.SetForecasterNetworkRegret(s.ctx, topicId, forecaster1, getWorkerRegretValue("0.2"))
	require.NoError(err)
	err = keeper.SetForecasterNetworkRegret(s.ctx, topicId, forecaster2, getWorkerRegretValue("0.3"))
	require.NoError(err)

	getInferencesForBlockHeight := func(blockHeight int64) types.Inferences {
		return types.Inferences{
			Inferences: []*types.Inference{
				{
					Inferer:     worker0,
					Value:       alloraMath.MustNewDecFromString("-0.035995138925040600"),
					TopicId:     topicId,
					BlockHeight: blockHeight,
				},
				{
					Inferer:     worker1,
					Value:       alloraMath.MustNewDecFromString("-0.07333303938740420"),
					TopicId:     topicId,
					BlockHeight: blockHeight,
				},
				{
					Inferer:     worker2,
					Value:       alloraMath.MustNewDecFromString("-0.1495482917094790"),
					TopicId:     topicId,
					BlockHeight: blockHeight,
				},
				{
					Inferer:     worker3,
					Value:       alloraMath.MustNewDecFromString("-0.12952123274063800"),
					TopicId:     topicId,
					BlockHeight: blockHeight,
				},
				{
					Inferer:     worker4,
					Value:       alloraMath.MustNewDecFromString("-0.0703055329498285"),
					TopicId:     topicId,
					BlockHeight: blockHeight,
				},
			},
		}
	}

	getForecastsForBlockHeight := func(blockHeight int64) types.Forecasts {
		return types.Forecasts{
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
					BlockHeight: blockHeight,
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
					BlockHeight: blockHeight,
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
					BlockHeight: blockHeight,
				},
			},
		}
	}

	// dont insert inferences at the blockheight that matches with the losses

	err = keeper.InsertActiveInferences(s.ctx, topicId, inferenceNonce2.BlockHeight, getInferencesForBlockHeight(inferenceBlockHeight2))
	s.Require().NoError(err)

	err = keeper.InsertActiveForecasts(s.ctx, topicId, inferenceNonce2.BlockHeight, getForecastsForBlockHeight(inferenceBlockHeight2))
	require.NoError(err)

	// Update epoch topic epoch last ended
	err = keeper.UpdateTopicEpochLastEnded(s.ctx, topicId, inferenceBlockHeight2)
	require.NoError(err)

	// Test querying the server
	req := &types.GetLatestAvailableNetworkInferencesRequest{
		TopicId: topicId,
	}
	_, err = queryServer.GetLatestAvailableNetworkInferences(s.ctx, req)
	require.Error(err)
}
