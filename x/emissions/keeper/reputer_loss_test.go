package keeper_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *KeeperTestSuite) TestInsertActiveReputerLosses() {
	ctx := s.Ctx()
	require := s.Require()
	topicId := uint64(1)
	block := types.BlockHeight(100)

	//nolint:exhaustruct
	lossBundles := types.LossBundles{
		&types.ValueBundle{
			TopicId: topicId,
			ReputerRequestNonce: &types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{BlockHeight: block},
			},
			Reputer:       s.AddrsStr(0),
			ExtraData:     []byte("data"),
			CombinedValue: alloraMath.MustNewDecFromString("123"),
			InfererValues: s.createDefaultInfererValues(),
			NaiveValue:    alloraMath.MustNewDecFromString("123"),
		},
	}

	// Test inserting data
	err := s.ReputerLossKeeper().InsertActiveReputerLosses(ctx, topicId, block, lossBundles)
	require.NoError(err, "InsertActiveReputerLosses should not return an error")

	// Retrieve data to verify insertion
	result, err := s.ReputerLossKeeper().GetReputerLossBundlesAtBlock(ctx, topicId, block)
	require.NoError(err)
	require.NotNil(result)
	require.Equal(lossBundles, result, "Retrieved data should match inserted data")
}

func (s *KeeperTestSuite) TestGetReputerLossBundlesAtBlock() {
	ctx := s.Ctx()
	require := s.Require()
	topicId := uint64(1)
	block := types.BlockHeight(100)

	// Test getting data before any insert, should return error or nil
	result, err := s.ReputerLossKeeper().GetReputerLossBundlesAtBlock(ctx, topicId, block)
	require.NoError(err)
	require.Empty(result, "Result should be empty for non-existent data")
}

func (s *KeeperTestSuite) TestInsertNetworkLossBundleAtBlock() {
	ctx := s.Ctx()
	require := s.Require()
	topicId := uint64(1)
	block := types.BlockHeight(100)
	//nolint:exhaustruct
	lossBundle := types.ValueBundle{
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{BlockHeight: block},
		},
		Reputer:       "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
		ExtraData:     []byte("data"),
		CombinedValue: alloraMath.MustNewDecFromString("123"),
		InfererValues: s.createDefaultInfererValues(),
		NaiveValue:    alloraMath.MustNewDecFromString("123"),
	}

	err := s.ReputerLossKeeper().InsertNetworkLossBundleAtBlock(ctx, topicId, block, lossBundle)
	require.NoError(err, "InsertNetworkLossBundleAtBlock should not return an error")

	// Verify the insertion
	result, err := s.ReputerLossKeeper().GetNetworkLossBundleAtBlock(ctx, topicId, block)
	require.NoError(err)
	require.NotNil(result)
	require.Equal(&lossBundle, result, "Retrieved data should match inserted data")
}

// this unit test needs to be completely rewritten, PROTO-2575
func (s *KeeperTestSuite) TestGetNetworkLossBundleAtBlock() {
	ctx := s.Ctx()
	require := s.Require()
	topicId := uint64(1)
	block := types.BlockHeight(100)

	// Attempt to retrieve before insertion
	result, err := s.ReputerLossKeeper().GetNetworkLossBundleAtBlock(ctx, topicId, block)
	require.NoError(err, "Should not return error for empty loss bundle")
	require.NotNil(result)
	require.Equal(topicId, result.TopicId, "Result should be not be nil for non-existent data")
}

func (s *KeeperTestSuite) TestGetLatestNetworkLossBundle() {
	ctx := s.Ctx()
	k := s.ReputerLossKeeper()
	topicId := s.CreateTopic()

	// Initially, there should be no loss bundle, so we expect a zero result
	emptyLossBundle, err := k.GetLatestNetworkLossBundle(ctx, topicId)
	s.Require().ErrorIs(err, types.ErrNotFound)
	s.Require().Nil(emptyLossBundle, "Expected no network loss bundle initially")

	// Insert first network loss bundle
	blockHeight1 := types.BlockHeight(100)
	//nolint:exhaustruct
	lossBundle1 := types.ValueBundle{
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{BlockHeight: blockHeight1},
		},
		Reputer:       "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
		ExtraData:     []byte("data"),
		CombinedValue: alloraMath.MustNewDecFromString("123"),
		InfererValues: s.createDefaultInfererValues(),
		NaiveValue:    alloraMath.MustNewDecFromString("123"),
	}
	err = k.InsertNetworkLossBundleAtBlock(ctx, topicId, blockHeight1, lossBundle1)
	s.Require().NoError(err, "Inserting first network loss bundle should not fail")

	// Insert second network loss bundle
	blockHeight2 := types.BlockHeight(200)
	//nolint:exhaustruct
	lossBundle2 := types.ValueBundle{
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{BlockHeight: blockHeight2},
		},
		Reputer:       "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
		ExtraData:     []byte("data"),
		CombinedValue: alloraMath.MustNewDecFromString("456"),
		InfererValues: s.createDefaultInfererValues(),
		NaiveValue:    alloraMath.MustNewDecFromString("123"),
	}
	err = k.InsertNetworkLossBundleAtBlock(ctx, topicId, blockHeight2, lossBundle2)
	s.Require().NoError(err, "Inserting second network loss bundle should not fail")

	// Retrieve the latest network loss bundle
	latestLossBundle, err := k.GetLatestNetworkLossBundle(ctx, topicId)
	s.Require().NoError(err, "Retrieving latest network loss bundle should not fail")
	s.Require().Equal(&lossBundle2, latestLossBundle, "Latest network loss bundle should match the second inserted set")
}

func (s *KeeperTestSuite) TestInsertReputer() {
	ctx := s.Ctx()
	k := s.ReputerLossKeeper()
	reputer := s.AddrsStr(0)
	topicId := uint64(501)

	// Define sample OffchainNode information for a reputer
	reputerInfo := types.OffchainNode{
		Owner:       s.AddrsStr(1),
		NodeAddress: s.AddrsStr(2),
	}

	// Attempt to insert the reputer for multiple topics
	err := k.InsertReputer(ctx, topicId, reputer, reputerInfo)
	s.Require().NoError(err)

	// Optionally check if reputer is registered in each topic using an assumed IsReputerRegisteredInTopic method
	isRegistered, regErr := k.IsReputerRegisteredInTopic(ctx, topicId, reputer)
	s.Require().NoError(regErr, "Checking reputer registration should not fail")
	s.Require().True(isRegistered, "Reputer should be registered in each topic")
}

func (s *KeeperTestSuite) TestUpdateReputerOwner() {
	ctx := s.Ctx()
	k := s.ReputerLossKeeper()
	topicId := uint64(502)
	reputer := s.AddrsStr(4)
	initialOwner := s.AddrsStr(5)
	newOwner := s.AddrsStr(6)
	nonRegisteredReputer := s.AddrsStr(7)

	err := k.InsertReputer(ctx, topicId, reputer, types.OffchainNode{
		NodeAddress: reputer,
		Owner:       initialOwner,
	})
	s.Require().NoError(err)

	oldOwner, err := k.UpdateReputerOwner(ctx, reputer, newOwner)
	s.Require().NoError(err)
	s.Require().Equal(initialOwner, oldOwner)

	stored, err := k.GetReputerInfo(ctx, reputer)
	s.Require().NoError(err)
	s.Require().Equal(newOwner, stored.Owner)

	_, err = k.UpdateReputerOwner(ctx, nonRegisteredReputer, newOwner)
	s.Require().ErrorIs(err, types.ErrAddressNotRegistered)
}

func (s *KeeperTestSuite) TestGetReputerInfo() {
	ctx := s.Ctx()
	reputer := "allo17srupely9uux7axep5shdsezva4znz6g30ntdw"
	topicId := uint64(501)
	k := s.ReputerLossKeeper()
	reputerInfo := types.OffchainNode{
		Owner:       s.AddrsStr(2),
		NodeAddress: reputer,
	}

	err := k.InsertReputer(ctx, topicId, reputer, reputerInfo)
	s.Require().NoError(err)

	actualReputer, err := k.GetReputerInfo(ctx, reputer)
	s.Require().NoError(err)
	s.Require().Equal(reputerInfo, actualReputer)

	nonExistentKey := "nonExistentKey123"
	_, err = k.GetReputerInfo(ctx, nonExistentKey)
	s.Require().Error(err)
}

func (s *KeeperTestSuite) TestRemoveReputer() {
	ctx := s.Ctx()
	k := s.ReputerLossKeeper()
	reputer := "allo17srupely9uux7axep5shdsezva4znz6g30ntdw"
	topicId := uint64(501)

	// Pre-setup: Insert the reputer for initial setup
	err := k.InsertReputer(ctx, topicId, reputer, types.OffchainNode{
		Owner:       s.AddrsStr(1),
		NodeAddress: reputer,
	})
	s.Require().NoError(err, "InsertReputer failed during setup")

	// Verify the reputer is registered in the topic
	isRegisteredPre, preErr := k.IsReputerRegisteredInTopic(ctx, topicId, reputer)
	s.Require().NoError(preErr, "Failed to check reputer registration before removal")
	s.Require().True(isRegisteredPre, "Reputer should be registered in the topic before removal")

	// Perform the removal
	removeErr := k.RemoveReputer(ctx, topicId, reputer)
	s.Require().NoError(removeErr, "Failed to remove reputer")

	// Verify the reputer is no longer registered in the topic
	isRegisteredPost, postErr := k.IsReputerRegisteredInTopic(ctx, topicId, reputer)
	s.Require().NoError(postErr, "Failed to check reputer registration after removal")
	s.Require().False(isRegisteredPost, "Reputer should not be registered in the topic after removal")
}

func (s *KeeperTestSuite) TestAppendReputerLoss() {
	ctx := s.Ctx()
	k := s.ReputerLossKeeper()
	topicId := s.CreateTopic()
	blockHeight := int64(10)
	nonce := types.Nonce{BlockHeight: blockHeight}
	reputerRequestNonce := &types.ReputerRequestNonce{
		ReputerNonce: &types.Nonce{BlockHeight: blockHeight},
	}

	// Create reputers and set their scores
	reputers := make([]string, 5)
	scores := []int64{95, 90, 99, 91, 96}
	for i := range reputers {
		reputers[i] = s.AddrsStr(i)
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: 2,
			Address:     reputers[i],
			Score:       alloraMath.NewDecFromInt64(scores[i]),
		}
		err := s.ScoresKeeper().SetReputerScoreEma(ctx, topicId, reputers[i], score)
		s.Require().NoError(err)
	}

	// Set params
	params := types.DefaultParams()
	params.MaxTopReputersToReward = 4
	err := s.ParamsKeeper().SetParams(ctx, params)
	s.Require().NoError(err)

	// Create value bundles for all reputers
	reputerValueBundles := make([]*types.ReputerValueBundle, len(reputers))
	for i, reputer := range reputers {
		//nolint:exhaustruct
		valueBundle := types.ValueBundle{
			Reputer:             reputer,
			CombinedValue:       alloraMath.MustNewDecFromString(".0000256948644008351"),
			ReputerRequestNonce: reputerRequestNonce,
			TopicId:             topicId,
			InfererValues:       s.createDefaultInfererValues(),
			NaiveValue:          alloraMath.MustNewDecFromString("0.0"),
		}
		signature := s.SignValueBundle(&valueBundle, s.PrivKeys(i))
		reputerValueBundles[i] = &types.ReputerValueBundle{
			ValueBundle: &valueBundle,
			Signature:   signature,
			Pubkey:      s.PubKeyHexStr(i),
		}
	}

	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)

	// Append first three reputer losses
	for i := 0; i < 3; i++ {
		err = k.AppendReputerLoss(ctx, topic, params, nonce.BlockHeight, reputerValueBundles[i])
		s.Require().NoError(err)
	}

	// Add fourth reputer and verify active reputers count
	err = k.AppendReputerLoss(ctx, topic, params, nonce.BlockHeight, reputerValueBundles[3])
	s.Require().NoError(err)
	activeReputers, err := k.GetActiveReputersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopReputersToReward, uint64(len(activeReputers)))

	// Add fifth reputer and verify active reputers count remains at max
	err = k.AppendReputerLoss(ctx, topic, params, nonce.BlockHeight, reputerValueBundles[4])
	s.Require().NoError(err)
	activeReputers, err = k.GetActiveReputersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopReputersToReward, uint64(len(activeReputers)))
}

func (s *KeeperTestSuite) TestAppendReputerLossWithResetActiveReputers() {
	ctx := s.Ctx()
	k := s.ReputerLossKeeper()
	topicId := s.CreateTopic()
	blockHeight := int64(10)
	nonce := types.Nonce{BlockHeight: blockHeight}
	reputerRequestNonce := &types.ReputerRequestNonce{
		ReputerNonce: &types.Nonce{BlockHeight: blockHeight},
	}

	// Setup reputers and their scores
	reputers := make([]string, 5)
	scores := []int64{95, 90, 99, 91, 96}
	for i := range reputers {
		reputers[i] = s.AddrsStr(i)
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: 2,
			Address:     reputers[i],
			Score:       alloraMath.NewDecFromInt64(scores[i]),
		}
		err := s.ScoresKeeper().SetReputerScoreEma(ctx, topicId, reputers[i], score)
		s.Require().NoError(err)
	}

	// Set params
	params := types.DefaultParams()
	params.MaxTopReputersToReward = 4
	err := s.ParamsKeeper().SetParams(ctx, params)
	s.Require().NoError(err)

	// Create reputer value bundles
	reputerValueBundles := make([]*types.ReputerValueBundle, len(reputers))
	for i, reputer := range reputers {
		//nolint:exhaustruct
		valueBundle := types.ValueBundle{
			Reputer:             reputer,
			CombinedValue:       alloraMath.MustNewDecFromString(".0000256948644008351"),
			ReputerRequestNonce: reputerRequestNonce,
			TopicId:             topicId,
			InfererValues:       s.createDefaultInfererValues(),
			NaiveValue:          alloraMath.MustNewDecFromString("0.0"),
		}
		signature := s.SignValueBundle(&valueBundle, s.PrivKeys(i))
		reputerValueBundles[i] = &types.ReputerValueBundle{
			ValueBundle: &valueBundle,
			Signature:   signature,
			Pubkey:      s.PubKeyHexStr(i),
		}
	}

	allReputerLosses := types.ReputerValueBundles{
		ReputerValueBundles: reputerValueBundles,
	}

	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)

	// First round: append all reputer losses
	for _, reputerValueBundle := range allReputerLosses.ReputerValueBundles {
		err = k.AppendReputerLoss(ctx, topic, params, nonce.BlockHeight, reputerValueBundle)
		s.Require().NoError(err)
	}

	// Verify active reputers count
	activeReputers, err := k.GetActiveReputersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopReputersToReward, uint64(len(activeReputers)))

	// Reset active reputers
	err = k.ResetActiveReputersForTopic(ctx, topicId)
	s.Require().NoError(err)

	activeReputers, err = k.GetActiveReputersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Empty(activeReputers)

	// Second round: append all reputer losses again
	nonce.BlockHeight++
	for _, reputerValueBundle := range allReputerLosses.ReputerValueBundles {
		err = k.AppendReputerLoss(ctx, topic, params, nonce.BlockHeight, reputerValueBundle)
		s.Require().NoError(err)
	}

	// Verify active reputers count again
	activeReputers, err = k.GetActiveReputersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopReputersToReward, uint64(len(activeReputers)))
}

func (s *KeeperTestSuite) TestActiveReputerFunctions() {
	ctx := s.Ctx()
	k := s.ReputerLossKeeper()
	topicId := uint64(1)
	reputer := s.AddrsStr(3)

	err := k.AddActiveReputer(ctx, topicId, reputer)
	s.Require().NoError(err)

	isActive, err := k.IsActiveReputer(ctx, topicId, reputer)
	s.Require().NoError(err)
	s.Require().True(isActive)

	activeReputers, err := k.GetActiveReputersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Len(activeReputers, 1)
	s.Require().Equal(reputer, activeReputers[0])

	err = k.RemoveActiveReputer(ctx, topicId, reputer)
	s.Require().NoError(err)

	isActive, err = k.IsActiveReputer(ctx, topicId, reputer)
	s.Require().NoError(err)
	s.Require().False(isActive)
}

func (s *KeeperTestSuite) TestRemoveReputerLoss() {
	ctx := s.Ctx()
	k := s.ReputerLossKeeper()
	topicId := s.CreateTopic()
	reputer := s.AddrsStr(0)

	// Create a reputer loss bundle
	//nolint:exhaustruct
	valueBundle := types.ValueBundle{
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{BlockHeight: 100},
		},
		Reputer:       reputer,
		ExtraData:     []byte("data"),
		CombinedValue: alloraMath.MustNewDecFromString("123"),
		InfererValues: s.createDefaultInfererValues(),
		NaiveValue:    alloraMath.MustNewDecFromString("123"),
	}
	signature := s.SignValueBundle(&valueBundle, s.PrivKeys(0))
	reputerLossBundle := types.ReputerValueBundle{
		ValueBundle: &valueBundle,
		Signature:   signature,
		Pubkey:      s.PubKeyHexStr(0),
	}

	// Insert the reputer loss bundle
	err := k.InsertReputerLoss(ctx, topicId, reputerLossBundle)
	s.Require().NoError(err)

	// Verify the reputer loss was added
	retrievedLoss, err := k.GetReputerLatestLossByTopicId(ctx, topicId, reputer)
	s.Require().NoError(err)
	s.Require().Equal(valueBundle, retrievedLoss)

	// Remove the reputer loss
	err = k.RemoveReputerLoss(ctx, topicId, reputer)
	s.Require().NoError(err)

	// Verify the reputer loss was removed
	_, err = k.GetReputerLatestLossByTopicId(ctx, topicId, reputer)
	s.Require().Error(err) // Expect an error since the loss should be removed
}
