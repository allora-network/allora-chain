package queryserver_test

import (
	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestGetInferencesAtBlock() {
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
	ctx := s.ctx
	queryServer := s.queryServer
	require := s.Require()

	keeper := s.emissionsKeeper

	reputer0 := "allo1m5v6rgjtxh4xszrrzqacwjh4ve6r0za2gxx9qr"
	reputer1 := "allo1e7cj9839ht2xm8urynqs5279hrvqd8neusvp2x"
	reputer2 := "allo1k9ss0xfer54nyack5678frl36e5g3rj2yzxtfj"
	reputer3 := "allo18ljxewge4vqrkk09tm5heldqg25yj8d9ekgkw5"
	reputer4 := "allo1k36ljvn8z0u49sagdg46p75psgreh23kdjn3l0"

	reputer0Acc := sdk.AccAddress(reputer0)
	reputer1Acc := sdk.AccAddress(reputer1)
	reputer2Acc := sdk.AccAddress(reputer2)
	reputer3Acc := sdk.AccAddress(reputer3)
	reputer4Acc := sdk.AccAddress(reputer4)

	topicId := uint64(1)
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

	err = keeper.AddStake(s.ctx, topicId, reputer0Acc, cosmosMath.NewUintFromString("210535101370326000000000"))
	require.NoError(err)
	err = keeper.AddStake(s.ctx, topicId, reputer1Acc, cosmosMath.NewUintFromString("216697093951021000000000"))
	require.NoError(err)
	err = keeper.AddStake(s.ctx, topicId, reputer2Acc, cosmosMath.NewUintFromString("161740241803855000000000"))
	require.NoError(err)
	err = keeper.AddStake(s.ctx, topicId, reputer3Acc, cosmosMath.NewUintFromString("394848305052250000000000"))
	require.NoError(err)
	err = keeper.AddStake(s.ctx, topicId, reputer4Acc, cosmosMath.NewUintFromString("206169717590569000000000"))
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

	// Test querying the server
	req := &types.QueryNetworkInferencesAtBlockRequest{
		TopicId:     topicId,
		BlockHeight: blockHeight,
	}
	response, err := queryServer.GetNetworkInferencesAtBlock(ctx, req)
	require.NoError(err, "Query should not return an error")
	require.NotNil(response, "Response should not be nil")
	require.Equal(blockHeight, response.BlockHeight, "The returned block height should match the requested block height")
}
