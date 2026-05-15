package queryserver_test

import (
	"fmt"

	cosmosMath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/utils/ptr"
	synth "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *QueryServerTestSuite) TestGetInferencesAtBlock() {
	ctx := s.Ctx()
	queryServer := s.EmissionsQueryServer()
	topicId := uint64(1)
	blockHeight := types.BlockHeight(100)
	expectedInferences := types.Inferences{
		Inferences: []*types.Inference{
			{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Inferer:     "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
				Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(1)},
			},
			{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Inferer:     "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
				Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(2)},
			},
		},
	}

	nonce := types.Nonce{BlockHeight: blockHeight}
	err := s.WorkerKeeper().InsertActiveInferences(ctx, topicId, nonce.BlockHeight, expectedInferences)
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

//nolint:exhaustruct
func (s *QueryServerTestSuite) TestGetWorkerLatestInputInferenceByTopicId() {
	ctx := s.Ctx()
	queryServer := s.EmissionsQueryServer()
	topicId := s.CreateTopic()
	nonce := types.BlockHeight(42)
	worker := s.AddrsStr(0)

	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)
	topic.OutputArity = types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI
	topic.LabelDefaultValue = alloraMath.ZeroDec()
	s.Require().NoError(s.TopicKeeper().SetTopic(ctx, topicId, topic))
	s.Require().NoError(s.TopicKeeper().SetEpochLabelRegistry(ctx, types.EpochLabelRegistry{
		TopicId: topicId,
		EpochId: uint64(nonce),
		Labels: []*types.TopicLabel{
			{Id: 1, Name: "a"},
			{Id: 2, Name: "b"},
			{Id: 3, Name: "c"},
		},
	}))
	inference := types.Inference{
		TopicId:     topicId,
		BlockHeight: nonce,
		Inferer:     worker,
		Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString("0.1"), alloraMath.MustNewDecFromString("0.2")},
		ExtraData:   []byte("query"),
		Proof:       "proof",
	}
	s.Require().NoError(s.WorkerKeeper().InsertInference(ctx, topicId, inference))

	latestInput, err := queryServer.GetWorkerLatestInputInferenceByTopicId(ctx, &types.GetWorkerLatestInputInferenceByTopicIdRequest{
		TopicId:       topicId,
		WorkerAddress: worker,
	})
	s.Require().NoError(err)
	s.Require().Equal(topicId, latestInput.LatestInputInference.TopicId)
	s.Require().Equal(nonce, latestInput.LatestInputInference.BlockHeight)
	s.Require().Equal(worker, latestInput.LatestInputInference.Inferer)
	s.Require().Equal([]byte("query"), latestInput.LatestInputInference.ExtraData)
	s.Require().Equal("proof", latestInput.LatestInputInference.Proof)
	s.Require().Len(latestInput.LatestInputInference.Values, 2)
	s.Require().Equal("a", latestInput.LatestInputInference.Values[0].Label)
	s.Require().Equal("0.1", latestInput.LatestInputInference.Values[0].Value.String())
	s.Require().Equal("b", latestInput.LatestInputInference.Values[1].Label)
	s.Require().Equal("0.2", latestInput.LatestInputInference.Values[1].Value.String())
}

//nolint:exhaustruct
func (s *QueryServerTestSuite) TestGetNetworkInferencesAtBlock() {
	queryServer := s.EmissionsQueryServer()
	require := s.Require()

	topicId := uint64(1)
	blockHeight := int64(10)

	// Set up test data
	reputers := make([]string, 5)
	reputerIndexes := testutil.ReturnIndexes(5, 5)
	for i, idx := range reputerIndexes {
		reputers[i] = s.AddrsStr(idx)
	}

	simpleNonce := types.Nonce{BlockHeight: blockHeight}
	reputerRequestNonce := &types.ReputerRequestNonce{
		ReputerNonce: &types.Nonce{BlockHeight: blockHeight},
	}

	// Create and set value bundles
	valueBundleData := []struct {
		reputer       string
		combinedValue string
	}{
		{reputers[0], ".0000117005278862668"},
		{reputers[1], ".00000962701954026944"},
		{reputers[2], ".0000256948644008351"},
		{reputers[3], ".0000123986052417188"},
		{reputers[4], ".0000115363240547692"},
	}

	var reputerValueBundles types.LossBundles
	for _, data := range valueBundleData {
		valueBundle := &types.ValueBundle{
			TopicId:             topicId,
			Reputer:             data.reputer,
			ReputerRequestNonce: reputerRequestNonce,
			CombinedValue:       alloraMath.MustNewDecFromString(data.combinedValue),
			InfererValues: []*types.WorkerAttributedValue{
				{
					Worker: "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
					Value:  alloraMath.MustNewDecFromString("1"),
				},
			},
			NaiveValue: alloraMath.MustNewDecFromString(data.combinedValue),
		}
		reputerValueBundles = append(reputerValueBundles, valueBundle)
	}

	err := s.ReputerLossKeeper().InsertActiveReputerLosses(s.Ctx(), topicId, blockHeight, reputerValueBundles)
	require.NoError(err)

	// Set stakes
	stakeValues := []string{
		"210535101370326000000000",
		"216697093951021000000000",
		"161740241803855000000000",
		"394848305052250000000000",
		"206169717590569000000000",
	}

	for i, stakeValue := range stakeValues {
		stake, ok := cosmosMath.NewIntFromString(stakeValue)
		require.True(ok)
		err = s.StakingKeeper().AddReputerStake(s.Ctx(), topicId, reputers[i], stake)
		require.NoError(err)
	}

	// Set inferences
	inferenceValues := []string{
		"-0.035995138925040600",
		"-0.07333303938740420",
		"-0.1495482917094790",
		"-0.12952123274063800",
		"-0.0703055329498285",
	}

	var infs []*types.Inference
	for i, value := range inferenceValues {
		infs = append(infs, &types.Inference{
			Inferer:     reputers[i],
			Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString(value)},
			TopicId:     topicId,
			BlockHeight: blockHeight,
		})
	}

	inferences := types.Inferences{Inferences: infs}
	err = s.WorkerKeeper().InsertActiveInferences(s.Ctx(), topicId, simpleNonce.BlockHeight, inferences)
	require.NoError(err)

	// Set block height and update epoch
	s.WithBlockHeight(blockHeight + 10)
	err = s.TopicKeeper().UpdateTopicEpochLastEnded(s.Ctx(), topicId, blockHeight+10)
	require.NoError(err)

	// Test query
	req := &types.GetNetworkInferencesAtBlockRequest{
		TopicId:                  topicId,
		BlockHeightLastInference: blockHeight,
	}
	response, err := queryServer.GetNetworkInferencesAtBlock(s.Ctx(), req)
	require.NoError(err)
	require.NotNil(response, "Response should not be nil")
}

//nolint:exhaustruct
func (s *QueryServerTestSuite) TestGetLatestNetworkInferences() {
	queryServer := s.EmissionsQueryServer()
	keeper := *s.EmissionsKeeper()
	require := s.Require()

	// Setup test data
	topicId := uint64(1)
	topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
	require.NoError(err)

	epochLength := topic.EpochLength
	epochLastEnded := int64(1)
	lossBlockHeight := epochLastEnded
	inferenceBlockHeight := epochLastEnded + epochLength

	lossNonce := types.Nonce{BlockHeight: lossBlockHeight}
	inferenceNonce := types.Nonce{BlockHeight: inferenceBlockHeight}
	reputerLossRequestNonce := &types.ReputerRequestNonce{ReputerNonce: &lossNonce}

	// Setup workers and forecasters
	workers := []string{
		"allo1s8sar766d54wzlmqhwpwdv0unzjfusjydg3l3j",
		"allo1rp9026g0ppp9nwdtzvxpqqhl43yqplrj7pmnhq",
		"allo1cmdyvyqgzudlf0ep2nht333a057wg9vwfek7tq",
		"allo1cr5usf94ph9w2lpeqfjkv3eyuspv47c0zx3nye",
		"allo19dvpcsqqer4xy7cdh4s3gtm460z6xpe2hzlf5s",
	}

	forecasters := []string{
		"allo13hh468ghmmyfjrdwqn567j29wq8sh6pnwff0cn",
		"allo1nxqgvyt6ggu3dz7uwe8p22sac6v2v8sayhwqvz",
		"allo1a0sc83cls78g4j5qey5er9zzpjpva4x935aajk",
	}

	// Set loss bundle
	s.WithBlockHeight(lossBlockHeight)
	err = s.ReputerLossKeeper().InsertNetworkLossBundleAtBlock(s.Ctx(), topicId, lossBlockHeight, types.ValueBundle{
		TopicId:             topicId,
		ReputerRequestNonce: reputerLossRequestNonce,
		Reputer:             s.AddrsStr(8),
		CombinedValue:       alloraMath.MustNewDecFromString("0.00001342819294865661936622664543402969"),
		InfererValues: []*types.WorkerAttributedValue{
			{
				Worker: "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
				Value:  alloraMath.MustNewDecFromString("1"),
			},
		},
		NaiveValue: alloraMath.MustNewDecFromString("0.00001342819294865661936622664543402969"),
	})
	require.NoError(err)

	// Set worker commits and regrets
	err = s.TopicKeeper().SetWorkerTopicLastCommit(s.Ctx(), topicId, inferenceBlockHeight, &inferenceNonce)
	require.NoError(err)

	s.WithBlockHeight(inferenceBlockHeight)

	getWorkerRegretValue := func(value string) types.TimestampedValue {
		return types.TimestampedValue{
			BlockHeight: inferenceBlockHeight,
			Value:       alloraMath.MustNewDecFromString(value),
		}
	}

	// Set worker regrets
	for i, worker := range workers {
		err = s.RegretsKeeper().SetInfererNetworkRegret(s.Ctx(), topicId, worker, getWorkerRegretValue(fmt.Sprintf("0.%d", i+1)))
		require.NoError(err)
	}

	// Set forecaster regrets
	for i, forecaster := range forecasters {
		err = s.RegretsKeeper().SetForecasterNetworkRegret(s.Ctx(), topicId, forecaster, getWorkerRegretValue(fmt.Sprintf("0.%d", i+1)))
		require.NoError(err)
	}

	// Set inferences and forecasts
	inferences := getInferencesForBlockHeight(workers, inferenceBlockHeight, topicId)
	err = s.WorkerKeeper().InsertActiveInferences(s.Ctx(), topicId, inferenceNonce.BlockHeight, inferences)
	require.NoError(err)

	_, err = keeper.GetTopicKeeper().RegisterEpochLabel(
		s.Ctx(),
		topic.Id,
		topic.LabelCaseSensitive,
		inferenceNonce.BlockHeight,
		"y",
	)
	require.NoError(err)

	forecasts := getForecastsForBlockHeight(workers, forecasters, inferenceBlockHeight, topicId)
	err = s.WorkerKeeper().InsertActiveForecasts(s.Ctx(), topicId, inferenceNonce.BlockHeight, forecasts)
	require.NoError(err)

	// Update epoch
	err = s.TopicKeeper().UpdateTopicEpochLastEnded(s.Ctx(), topicId, inferenceBlockHeight)
	require.NoError(err)

	// Calculate network inferences
	networkInferences, err := synth.GetNetworkInferences(
		sdk.UnwrapSDKContext(s.Ctx()),
		keeper,
		topicId,
		&inferenceNonce.BlockHeight,
		ptr.To(inferences),
		ptr.To(forecasts),
		false,
	)
	require.NoError(err)

	err = keeper.InsertNetworkInferenceBundle(s.Ctx(), topicId, inferenceNonce.BlockHeight, *networkInferences.NetworkInferences)
	require.NoError(err)

	// Test query
	response, err := queryServer.GetLatestNetworkInferences(s.Ctx(), &types.GetLatestNetworkInferencesRequest{
		TopicId: topicId,
	})
	require.NoError(err)
	require.NotNil(response, "Response should not be nil")
	require.Equal(len(response.NetworkInferences.ForecasterValues), len(forecasters))
	require.Equal(len(response.NetworkInferences.InfererValues), len(workers))
}

func (s *QueryServerTestSuite) TestIsWorkerNonceUnfulfilled() {
	ctx := s.Ctx()
	topicId := uint64(1)
	newNonce := &types.Nonce{BlockHeight: 42}

	req := &types.IsWorkerNonceUnfulfilledRequest{
		TopicId:     topicId,
		BlockHeight: newNonce.BlockHeight,
	}
	response, err := s.EmissionsQueryServer().IsWorkerNonceUnfulfilled(s.Ctx(), req)
	s.Require().NoError(err)
	s.Require().NotNil(response, "Response should not be nil")
	s.Require().False(response.IsWorkerNonceUnfulfilled)

	// Set worker nonce
	err = s.NonceKeeper().AddWorkerNonce(ctx, topicId, newNonce)
	s.Require().NoError(err)

	response, err = s.EmissionsQueryServer().IsWorkerNonceUnfulfilled(s.Ctx(), req)
	s.Require().NoError(err)
	s.Require().NotNil(response, "Response should not be nil")
	s.Require().True(response.IsWorkerNonceUnfulfilled)
}

func (s *QueryServerTestSuite) TestGetUnfulfilledWorkerNonces() {
	ctx := s.Ctx()
	topicId := uint64(1)

	// Initially, ensure no unfulfilled nonces exist
	req := &types.GetUnfulfilledWorkerNoncesRequest{
		TopicId: topicId,
	}
	response, err := s.EmissionsQueryServer().GetUnfulfilledWorkerNonces(s.Ctx(), req)
	s.Require().NoError(err)
	s.Require().NotNil(response, "Response should not be nil")
	s.Require().Len(response.Nonces.Nonces, 0, "Initial unfulfilled nonces should be empty")

	// Set multiple worker nonces
	nonceValues := []int64{42, 43, 44}
	for _, val := range nonceValues {
		err = s.NonceKeeper().AddWorkerNonce(ctx, topicId, &types.Nonce{BlockHeight: val})
		s.Require().NoError(err, "Failed to add worker nonce")
	}

	// Retrieve and verify the nonces
	response, err = s.EmissionsQueryServer().GetUnfulfilledWorkerNonces(s.Ctx(), req)
	s.Require().NoError(err, "Error retrieving nonces after adding")
	s.Require().Len(response.Nonces.Nonces, len(nonceValues), "Should match the number of added nonces")

	// Check that all the expected nonces are present and correct
	for i, nonce := range response.Nonces.Nonces {
		s.Require().Equal(nonceValues[len(nonceValues)-i-1], nonce.BlockHeight, "Nonce value should match the expected value")
	}
}

func (s *QueryServerTestSuite) TestGetInfererNetworkRegret() {
	ctx := s.Ctx()
	topicId := uint64(1)
	worker := s.AddrsStr(1)
	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}
	emptyRegret := types.TimestampedValue{
		BlockHeight: 0,
		Value:       alloraMath.NewDecFromInt64(0),
	}

	req := &types.GetInfererNetworkRegretRequest{
		TopicId: topicId,
		ActorId: worker,
	}
	response, err := s.EmissionsQueryServer().GetInfererNetworkRegret(s.Ctx(), req)
	s.Require().NoError(err)
	s.Require().Equal(response.Regret, &emptyRegret)

	// Set Inferer Network Regret
	err = s.RegretsKeeper().SetInfererNetworkRegret(ctx, topicId, worker, regret)
	s.Require().NoError(err)

	// Get Inferer Network Regret
	response, err = s.EmissionsQueryServer().GetInfererNetworkRegret(s.Ctx(), req)
	s.Require().NoError(err)
	s.Require().Equal(response.Regret, &regret)
}

func (s *QueryServerTestSuite) TestGetForecasterNetworkRegret() {
	ctx := s.Ctx()
	topicId := uint64(1)
	worker := s.AddrsStr(1)
	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}
	emptyRegret := types.TimestampedValue{
		BlockHeight: 0,
		Value:       alloraMath.NewDecFromInt64(0),
	}

	req := &types.GetForecasterNetworkRegretRequest{
		TopicId: topicId,
		Worker:  worker,
	}
	response, err := s.EmissionsQueryServer().GetForecasterNetworkRegret(s.Ctx(), req)
	s.Require().NoError(err)
	s.Require().Equal(response.Regret, &emptyRegret)

	// Set Forecaster Network Regret
	err = s.RegretsKeeper().SetForecasterNetworkRegret(ctx, topicId, worker, regret)
	s.Require().NoError(err)

	// Get Forecaster Network Regret
	response, err = s.EmissionsQueryServer().GetForecasterNetworkRegret(s.Ctx(), req)
	s.Require().NoError(err)
	s.Require().Equal(response.Regret, &regret)
}

func (s *QueryServerTestSuite) TestGetOneInForecasterNetworkRegret() {
	ctx := s.Ctx()
	topicId := uint64(1)
	forecaster := s.AddrsStr(3)
	inferer := s.AddrsStr(1)
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
	response, err := s.EmissionsQueryServer().GetOneInForecasterNetworkRegret(s.Ctx(), req)
	s.Require().NoError(err)
	s.Require().Equal(response.Regret, &emptyRegret)

	// Set One In Forecaster Network Regret
	err = s.RegretsKeeper().SetOneInForecasterNetworkRegret(ctx, topicId, forecaster, inferer, regret)
	s.Require().NoError(err)

	// Get One In Forecaster Network Regret
	response, err = s.EmissionsQueryServer().GetOneInForecasterNetworkRegret(s.Ctx(), req)
	s.Require().NoError(err)
	s.Require().Equal(response.Regret, &regret)
}

func (s *QueryServerTestSuite) TestGetLatestTopicInferences() {
	ctx := s.Ctx()

	topicId := uint64(1)

	// Initially, there should be no inferences, so we expect an empty result
	req := &types.GetLatestTopicInferencesRequest{
		TopicId: topicId,
	}
	response, err := s.EmissionsQueryServer().GetLatestTopicInferences(ctx, req)
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
		Inferer:     s.AddrsStr(1),
		Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString("10")},
		ExtraData:   []byte("data1"),
		Proof:       "proof1",
	}
	inferences1 := types.Inferences{
		Inferences: []*types.Inference{&newInference1},
	}
	nonce1 := types.Nonce{BlockHeight: blockHeight1}
	err = s.WorkerKeeper().InsertActiveInferences(ctx, topicId, nonce1.BlockHeight, inferences1)
	s.Require().NoError(err, "Inserting first set of inferences should not fail")

	// Insert second set of inferences
	blockHeight2 := types.BlockHeight(12346)
	newInference2 := types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeight2,
		Inferer:     s.AddrsStr(2),
		Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString("20")},
		ExtraData:   []byte("data2"),
		Proof:       "proof2",
	}
	inferences2 := types.Inferences{
		Inferences: []*types.Inference{&newInference2},
	}
	nonce2 := types.Nonce{BlockHeight: blockHeight2}
	err = s.WorkerKeeper().InsertActiveInferences(ctx, topicId, nonce2.BlockHeight, inferences2)
	s.Require().NoError(err, "Inserting second set of inferences should not fail")

	// Retrieve the latest inferences
	response, err = s.EmissionsQueryServer().GetLatestTopicInferences(ctx, req)
	s.Require().NoError(err)
	latestInferences := response.Inferences
	latestBlockHeight := response.BlockHeight
	s.Require().NoError(err, "Retrieving latest inferences should not fail")
	s.Require().Equal(&inferences2, latestInferences, "Latest inferences should match the second inserted set")
	s.Require().Equal(blockHeight2, latestBlockHeight, "Latest block height should match the second inserted set")
}

func (s *QueryServerTestSuite) TestGetLatestNetworkInferencesWithMissingInferences() {
	queryServer := s.EmissionsQueryServer()

	require := s.Require()
	topicId := uint64(1)
	topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
	require.NoError(err)

	epochLength := topic.EpochLength
	epochLastEnded := int64(1) // topic.EpochLastEnded , but 0 is not a valid nonce
	blockHeights := map[string]int64{
		"loss":       epochLastEnded,
		"inference":  epochLastEnded + epochLength,
		"inference2": epochLastEnded + 2*epochLength,
	}

	nonces := map[string]*types.Nonce{
		"loss":       {BlockHeight: blockHeights["loss"]},
		"inference2": {BlockHeight: blockHeights["inference2"]},
	}

	reputerLossRequestNonce := &types.ReputerRequestNonce{ReputerNonce: nonces["loss"]}

	s.WithBlockHeight(blockHeights["loss"])

	// Set Loss bundles
	//nolint:exhaustruct
	lossBundle := types.ValueBundle{
		CombinedValue:       alloraMath.MustNewDecFromString("0.00001342819294865661936622664543402969"),
		ReputerRequestNonce: reputerLossRequestNonce,
		TopicId:             topicId,
		Reputer:             s.AddrsStr(0),
		NaiveValue:          alloraMath.MustNewDecFromString("0.00001342819294865661936622664543402969"),
		InfererValues: []*types.WorkerAttributedValue{
			{
				Worker: "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
				Value:  alloraMath.MustNewDecFromString("1"),
			},
		},
	}
	err = s.ReputerLossKeeper().InsertNetworkLossBundleAtBlock(s.Ctx(), topicId, blockHeights["loss"], lossBundle)
	require.NoError(err)

	// Set Inferences
	s.WithBlockHeight(blockHeights["inference"])

	getWorkerRegretValue := func(value string) types.TimestampedValue {
		return types.TimestampedValue{
			BlockHeight: blockHeights["inference"],
			Value:       alloraMath.MustNewDecFromString(value),
		}
	}

	workers := make([]string, 5)
	forecasters := make([]string, 3)
	for i := range workers {
		workers[i] = s.AddrsStr(i + 1)
	}
	for i := range forecasters {
		forecasters[i] = s.AddrsStr(i + 6)
	}

	// Set network regrets
	for i, worker := range workers {
		err = s.RegretsKeeper().SetInfererNetworkRegret(s.Ctx(), topicId, worker, getWorkerRegretValue(alloraMath.NewDecFromInt64(int64(i+1)/10).String()))
		require.NoError(err)
	}

	for i, forecaster := range forecasters {
		err = s.RegretsKeeper().SetForecasterNetworkRegret(s.Ctx(), topicId, forecaster, getWorkerRegretValue(alloraMath.NewDecFromInt64(int64(i+1)/10).String()))
		require.NoError(err)
	}

	// Insert active inferences and forecasts
	err = s.WorkerKeeper().InsertActiveInferences(s.Ctx(), topicId, nonces["inference2"].BlockHeight, getInferencesForBlockHeight(workers, blockHeights["inference2"], topicId))
	s.Require().NoError(err)

	err = s.WorkerKeeper().InsertActiveForecasts(s.Ctx(), topicId, nonces["inference2"].BlockHeight, getForecastsForBlockHeight(workers, forecasters, blockHeights["inference2"], topicId))
	require.NoError(err)

	// Update epoch topic epoch last ended
	err = s.TopicKeeper().UpdateTopicEpochLastEnded(s.Ctx(), topicId, blockHeights["inference2"])
	require.NoError(err)

	// Test querying the server
	req := &types.GetLatestNetworkInferencesRequest{
		TopicId: topicId,
	}
	response, err := queryServer.GetLatestNetworkInferences(s.Ctx(), req)

	require.NoError(err)
	require.NotNil(response)
	require.Equal(response.NetworkInferences.Nonce, int64(0))
}

// Helper function to generate inferences
func getInferencesForBlockHeight(workers []string, blockHeight int64, topicId uint64) types.Inferences {
	var inferences []*types.Inference
	inferenceValues := []string{
		"-0.035995138925040600",
		"-0.07333303938740420",
		"-0.1495482917094790",
		"-0.12952123274063800",
		"-0.0703055329498285",
	}

	for i, worker := range workers {
		//nolint:exhaustruct
		inferences = append(inferences, &types.Inference{
			Inferer:     worker,
			Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString(inferenceValues[i])},
			TopicId:     topicId,
			BlockHeight: blockHeight,
		})
	}

	return types.Inferences{Inferences: inferences}
}

// Helper function to generate forecasts
func getForecastsForBlockHeight(workers []string, forecasters []string, blockHeight int64, topicId uint64) types.Forecasts {
	forecastValues := [][]string{
		{"0.1", "0.2", "0.3", "0.4", "0.5"},
		{"0.5", "0.4", "0.3", "0.2", "0.1"},
		{"0.2", "0.3", "0.4", "0.3", "0.2"},
	}

	var forecasts []*types.Forecast
	for i, forecaster := range forecasters {
		var elements []*types.ForecastElement
		for j, worker := range workers {
			elements = append(elements, &types.ForecastElement{
				Inferer: worker,
				Value:   alloraMath.MustNewDecFromString(forecastValues[i][j]),
			})
		}
		//nolint:exhaustruct
		forecasts = append(forecasts, &types.Forecast{
			Forecaster:       forecaster,
			ForecastElements: elements,
			TopicId:          topicId,
			BlockHeight:      blockHeight,
		})
	}

	return types.Forecasts{Forecasts: forecasts}
}
