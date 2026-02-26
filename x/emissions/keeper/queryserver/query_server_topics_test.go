package queryserver_test

import (
	cosmosMath "cosmossdk.io/math"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *QueryServerTestSuite) TestGetNextTopicId() {
	ctx := s.Ctx()
	queryServer := s.EmissionsQueryServer()
	keeper := s.TopicKeeper()

	// Get the initial next topic ID
	initialNextTopicId, err := keeper.GetNextTopicId(ctx)
	s.Require().NoError(err, "Fetching the initial next topic ID should not fail")

	topicsToCreate := 5
	for range topicsToCreate {
		s.CreateTopic()
	}

	req := &types.GetNextTopicIdRequest{}

	response, err := queryServer.GetNextTopicId(ctx, req)
	s.Require().NoError(err, "GetNextTopicId should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	expectedNextTopicId := initialNextTopicId + uint64(topicsToCreate)
	s.Require().Equal(expectedNextTopicId, response.NextTopicId, "The next topic ID should match the expected value after topic creation")
}

func (s *QueryServerTestSuite) TestGetTopic() {
	ctx := s.Ctx()
	queryServer := s.EmissionsQueryServer()
	keeper := s.TopicKeeper()

	topicId, err := keeper.GetNextTopicId(ctx)
	s.Require().NoError(err)
	metadata := "metadata"
	req := &types.GetTopicRequest{TopicId: topicId}

	// Setting up a new topic
	newTopic := s.MockTopic()
	newTopic.Id = topicId
	newTopic.Metadata = metadata
	err = keeper.SetTopic(ctx, topicId, newTopic)
	s.Require().NoError(err, "Setting a new topic should not fail")

	// Test retrieving an existing topic
	response, err := queryServer.GetTopic(ctx, req)
	s.Require().NoError(err, "Retrieving an existing topic should not fail")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().NotNil(response.Topic, "The response's Topic should not be nil")
	s.Require().Equal(newTopic, *response.Topic, "Retrieved topic should match the set topic")
	s.Require().Equal(metadata, response.Topic.Metadata, "The metadata of the retrieved topic should match")
}

func (s *QueryServerTestSuite) TestGetLatestCommit() {
	ctx := s.Ctx()
	queryServer := s.EmissionsQueryServer()
	keeper := s.TopicKeeper()
	blockHeight := 100
	nonce := types.Nonce{
		BlockHeight: 95,
	}

	topic := s.MockTopic()
	_ = keeper.SetReputerTopicLastCommit(
		ctx,
		topic.Id,
		int64(blockHeight),
		&nonce,
	)

	req := &types.GetTopicLastReputerCommitInfoRequest{
		TopicId: topic.Id,
	}

	response, err := queryServer.GetTopicLastReputerCommitInfo(ctx, req)
	s.Require().NoError(err, "GetTopicLastReputerCommitInfo should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().Equal(int64(blockHeight), response.LastCommit.BlockHeight, "Retrieved blockheight should match")
	s.Require().Equal(&nonce, response.LastCommit.Nonce, "The metadata of the retrieved nonce should match")

	topic2 := s.MockTopic()
	topic2.Id = 2
	blockHeight = 101
	nonce = types.Nonce{
		BlockHeight: 98,
	}

	_ = keeper.SetWorkerTopicLastCommit(
		ctx,
		topic2.Id,
		int64(blockHeight),
		&nonce,
	)

	req2 := &types.GetTopicLastWorkerCommitInfoRequest{
		TopicId: topic2.Id,
	}

	response2, err := queryServer.GetTopicLastWorkerCommitInfo(ctx, req2)
	s.Require().NoError(err, "GetTopicLastWorkerCommitInfo should not produce an error")
	s.Require().NotNil(response2, "The response should not be nil")
	s.Require().Equal(int64(blockHeight), response2.LastCommit.BlockHeight, "Retrieved blockheight should match")
	s.Require().Equal(&nonce, response2.LastCommit.Nonce, "The metadata of the retrieved nonce should match")
}

func (s *QueryServerTestSuite) TestGetSetDeleteTopicRewardNonce() {
	ctx := s.Ctx()
	keeper := s.TopicKeeper()
	topicId := uint64(1)

	// Test Get on an unset topicId, should return 0
	req := &types.GetTopicRewardNonceRequest{
		TopicId: topicId,
	}
	response, err := s.EmissionsQueryServer().GetTopicRewardNonce(ctx, req)
	s.Require().NoError(err, "Getting an unset topic reward nonce should not fail")
	nonce := response.Nonce
	s.Require().Equal(int64(0), nonce, "Nonce for an unset topicId should be 0")

	// Test Set
	expectedNonce := int64(12345)
	err = keeper.SetTopicRewardNonce(ctx, topicId, expectedNonce)
	s.Require().NoError(err, "Setting topic reward nonce should not fail")

	// Test Get after Set, should return the set value
	response, err = s.EmissionsQueryServer().GetTopicRewardNonce(ctx, req)
	s.Require().NoError(err, "Getting set topic reward nonce should not fail")
	nonce = response.Nonce
	s.Require().Equal(expectedNonce, nonce, "Nonce should match the value set earlier")

	// Test Delete
	err = keeper.DeleteTopicRewardNonce(ctx, topicId)
	s.Require().NoError(err, "Deleting topic reward nonce should not fail")

	// Test Get after Delete, should return 0
	response, err = s.EmissionsQueryServer().GetTopicRewardNonce(ctx, req)
	s.Require().NoError(err, "Getting deleted topic reward nonce should not fail")
	nonce = response.Nonce
	s.Require().Equal(int64(0), nonce, "Nonce should be 0 after deletion")
}

func (s *QueryServerTestSuite) TestGetPreviousTopicWeight() {
	ctx := s.Ctx()
	keeper := s.TopicKeeper()
	topicId := uint64(1)

	// Set previous topic weight
	weightToSet := alloraMath.NewDecFromInt64(10)
	err := keeper.SetPreviousTopicWeight(ctx, topicId, weightToSet)
	s.Require().NoError(err, "Setting previous topic weight should not fail")

	// Get the previously set topic weight
	req := &types.GetPreviousTopicWeightRequest{TopicId: topicId}
	response, err := s.EmissionsQueryServer().GetPreviousTopicWeight(ctx, req)
	s.Require().NoError(err, "Getting previous topic weight should not fail")

	retrievedWeight := response.Weight
	s.Require().Equal(weightToSet, retrievedWeight, "Retrieved weight should match the set weight")
}

func (s *QueryServerTestSuite) TestTopicExists() {
	ctx := s.Ctx()
	keeper := s.TopicKeeper()

	// Test a topic ID that does not exist
	nonExistentTopicId := uint64(999) // Assuming this ID has not been used
	req := &types.TopicExistsRequest{TopicId: nonExistentTopicId}
	response, err := s.EmissionsQueryServer().TopicExists(ctx, req)
	s.Require().NoError(err, "Checking existence for a non-existent topic should not fail")
	exists := response.Exists
	s.Require().False(exists, "No topic should exist for an unused topic ID")

	// Create a topic to test existence
	existentTopicId, err := keeper.IncrementTopicId(ctx)
	s.Require().NoError(err, "Incrementing topic ID should not fail")

	newTopic := s.MockTopic()
	newTopic.Id = existentTopicId

	err = keeper.SetTopic(ctx, existentTopicId, newTopic)
	s.Require().NoError(err, "Setting a new topic should not fail")

	// Test the newly created topic ID
	req = &types.TopicExistsRequest{TopicId: existentTopicId}
	response, err = s.EmissionsQueryServer().TopicExists(ctx, req)
	s.Require().NoError(err, "Checking existence for an existent topic should not fail")
	exists = response.Exists
	s.Require().True(exists, "Topic should exist for a newly created topic ID")
}

func (s *QueryServerTestSuite) TestIsTopicActive() {
	ctx := s.Ctx()
	keeper := s.TopicKeeper()
	topicId := uint64(3)

	// Assume topic initially active
	initialTopic := s.MockTopic()
	initialTopic.Id = topicId
	_ = keeper.SetTopic(ctx, topicId, initialTopic)

	// Activate the topic
	err := keeper.ActivateTopic(ctx, topicId)
	s.Require().NoError(err, "Reactivating topic should not fail")

	// Check if topic is active
	req := &types.IsTopicActiveRequest{TopicId: topicId}
	response, err := s.EmissionsQueryServer().IsTopicActive(ctx, req)
	s.Require().NoError(err, "Getting topic should not fail after reactivation")

	topicActive := response.IsActive
	s.Require().True(topicActive, "Topic should be active again")

	// Inactivate the topic
	err = keeper.InactivateTopic(ctx, topicId)
	s.Require().NoError(err, "Inactivating topic should not fail")

	// Check if topic is inactive
	req = &types.IsTopicActiveRequest{TopicId: topicId}
	response, err = s.EmissionsQueryServer().IsTopicActive(ctx, req)
	s.Require().NoError(err, "Getting topic should not fail after inactivation")
	topicActive = response.IsActive
	s.Require().False(topicActive, "Topic should be inactive")

	// Activate the topic
	err = keeper.ActivateTopic(ctx, topicId)
	s.Require().NoError(err, "Reactivating topic should not fail")

	// Check if topic is active again
	req = &types.IsTopicActiveRequest{TopicId: topicId}
	response, err = s.EmissionsQueryServer().IsTopicActive(ctx, req)
	s.Require().NoError(err, "Getting topic should not fail after reactivation")
	topicActive = response.IsActive
	s.Require().True(topicActive, "Topic should be active again")
}

func (s *QueryServerTestSuite) TestGetTopicFeeRevenue() {
	ctx := s.Ctx()
	keeper := s.TopicKeeper()
	topicId := uint64(2)

	// Test getting revenue for a topic with no existing revenue
	req := &types.GetTopicFeeRevenueRequest{TopicId: topicId}
	response, err := s.EmissionsQueryServer().GetTopicFeeRevenue(ctx, req)
	s.Require().NoError(err, "Should not error when revenue does not exist")
	feeRev := response.FeeRevenue
	s.Require().Equal(cosmosMath.ZeroInt(), feeRev, "Revenue should be zero for non-existing entries")

	// Setup a topic with some revenue
	initialRevenue := cosmosMath.NewInt(100)
	initialRevenueInt := cosmosMath.NewInt(100)
	err = keeper.AddTopicFeeRevenue(ctx, topicId, initialRevenue)
	s.Require().NoError(err, "Adding revenue should not fail")

	// Test getting revenue for a topic with existing revenue
	req = &types.GetTopicFeeRevenueRequest{TopicId: topicId}
	response, err = s.EmissionsQueryServer().GetTopicFeeRevenue(ctx, req)
	s.Require().NoError(err, "Should not error when retrieving existing revenue")
	feeRev = response.FeeRevenue
	s.Require().Equal(feeRev.String(), initialRevenueInt.String(), "Revenue should match the initial setup")
}

func (s *QueryServerTestSuite) TestGetWorkerSubmissionWindowStatus() {
	ctx := s.Ctx()
	queryServer := s.EmissionsQueryServer()
	topicId := uint64(1)
	workerAddress := s.AddrsStr(0)

	// Set params to prevent global whitelist bypass
	params := types.DefaultParams()
	params.MaxUnfulfilledReputerRequests = uint64(300)
	params.GlobalWorkerWhitelistEnabled = true
	params.GlobalReputerWhitelistEnabled = true
	err := s.ParamsKeeper().SetParams(ctx, params)
	s.Require().NoError(err)

	// Create topic
	topic := s.MockTopic()
	topic.Id = topicId
	topic.WorkerSubmissionWindow = 10
	topic.EpochLength = 20
	topic.GroundTruthLag = 30

	err = s.TopicKeeper().SetTopic(ctx, topicId, topic)
	s.Require().NoError(err)

	// Enable worker whitelist for testing
	err = s.WhitelistsKeeper().EnableTopicWorkerWhitelist(ctx, topicId)
	s.Require().NoError(err)

	// Test with no address provided
	req := &types.GetWorkerSubmissionWindowStatusRequest{
		TopicId: topicId,
		Address: "",
	}
	response, err := queryServer.GetWorkerSubmissionWindowStatus(ctx, req)
	s.Require().NoError(err, "Should not error with no address")
	s.Require().False(response.IsOpen, "Window should not be open with no active nonces")
	s.Require().False(response.IsRegistered, "Should be false when no address provided")
	s.Require().False(response.IsWhitelisted, "Should be false when no address provided")

	// Test with invalid address
	req.Address = "invalid_address"
	_, err = queryServer.GetWorkerSubmissionWindowStatus(ctx, req)
	s.Require().Error(err, "Should error with invalid address format")

	// Add worker to global whitelist first so they can be tested
	err = s.WhitelistsKeeper().AddToGlobalWorkerWhitelist(ctx, workerAddress)
	s.Require().NoError(err)

	// Test with valid unregistered address
	req.Address = workerAddress
	response, err = queryServer.GetWorkerSubmissionWindowStatus(ctx, req)
	s.Require().NoError(err)
	s.Require().False(response.IsRegistered)
	s.Require().True(response.IsWhitelisted) // Can submit via global whitelist

	// Register the worker using MsgServer
	moduleParams, _ := s.ParamsKeeper().GetParams(ctx)
	s.FundAccount(moduleParams.RegistrationFee.Int64(), s.Addrs(0))
	registerMsg := &types.RegisterRequest{
		Sender:    workerAddress,
		TopicId:   topicId,
		IsReputer: false,
		Owner:     workerAddress,
	}
	_, err = s.EmissionsMsgServer().Register(ctx, registerMsg)
	s.Require().NoError(err)

	// Verify registration
	response, err = queryServer.GetWorkerSubmissionWindowStatus(ctx, req)
	s.Require().NoError(err)
	s.Require().True(response.IsRegistered)

	// Add worker to whitelist for deterministic test setup
	err = s.WhitelistsKeeper().AddToTopicWorkerWhitelist(ctx, topicId, workerAddress)
	s.Require().NoError(err)

	// Fund and activate topic
	currentBlock := int64(0)
	err = s.StakingKeeper().AddReputerStake(ctx, topicId, s.AddrsStr(1), cosmosMath.NewInt(500000))
	s.Require().NoError(err)

	funderAddr := s.Addrs(2)
	s.FundTopic(topicId, funderAddr, cosmosMath.NewInt(10000))

	isActive, err := s.TopicKeeper().IsTopicActive(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(isActive)

	err = s.TopicKeeper().UpdateTopicEpochLastEnded(ctx, topicId, int64(0))
	s.Require().NoError(err)

	// Create multiple overlapping worker nonces to test "latest active nonce" selection
	// Set current block to have multiple active windows
	currentBlock = int64(5)
	s.WithBlockHeight(currentBlock)
	ctx = s.Ctx()

	// Create multiple nonces with overlapping windows
	nonce1 := &types.Nonce{BlockHeight: 0}  // Window [0, 10] - includes block 5
	nonce2 := &types.Nonce{BlockHeight: 2}  // Window [2, 12] - includes block 5 (more recent)
	nonce3 := &types.Nonce{BlockHeight: 15} // Window [15, 25] - future window

	err = s.NonceKeeper().AddWorkerNonce(ctx, topicId, nonce1)
	s.Require().NoError(err)
	err = s.NonceKeeper().AddWorkerNonce(ctx, topicId, nonce2)
	s.Require().NoError(err)
	err = s.NonceKeeper().AddWorkerNonce(ctx, topicId, nonce3)
	s.Require().NoError(err)

	unfulfilledNonces, err := s.NonceKeeper().GetUnfulfilledWorkerNonces(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Len(unfulfilledNonces.Nonces, 3)

	// Test that it returns the LATEST active nonce (nonce2 - most recent that includes current block)
	response, err = queryServer.GetWorkerSubmissionWindowStatus(ctx, req)
	s.Require().NoError(err)
	s.Require().True(response.IsOpen)
	s.Require().Equal(nonce2.BlockHeight, response.CurrentNonceBlockHeight) // Should be latest active

	// Verify window calculations for the latest active window
	expectedWindowStart := nonce2.BlockHeight
	expectedWindowEnd := nonce2.BlockHeight + topic.WorkerSubmissionWindow
	s.Require().Equal(expectedWindowStart, response.WindowStartBlock)
	s.Require().Equal(expectedWindowEnd, response.WindowEndBlock)

	// Verify next window calculations - based on epoch schedule from EpochLastEnded=0
	// With EpochLastEnded=0 and EpochLength=20, next epoch is at block 20
	expectedNextStart := int64(20) // Next epoch boundary
	expectedNextEnd := int64(30)   // expectedNextStart + WorkerSubmissionWindow(10)

	s.Require().Equal(expectedNextStart, response.NextWindowStartBlock)
	s.Require().Equal(expectedNextEnd, response.NextWindowEndBlock)

	// Verify topic is active
	s.Require().True(response.IsTopicActive)
}

func (s *QueryServerTestSuite) TestGetReputerSubmissionWindowStatus() {
	ctx := s.Ctx()
	queryServer := s.EmissionsQueryServer()
	topicId := uint64(1)
	reputerAddress := s.AddrsStr(0)
	var currentBlock int64

	params := types.DefaultParams()
	params.MaxUnfulfilledReputerRequests = uint64(300)
	params.GlobalWorkerWhitelistEnabled = true
	params.GlobalReputerWhitelistEnabled = true
	err := s.ParamsKeeper().SetParams(ctx, params)
	s.Require().NoError(err)

	// Create topic
	topic := s.MockTopic()
	topic.Id = topicId
	topic.WorkerSubmissionWindow = 10
	topic.EpochLength = 20
	topic.GroundTruthLag = 30

	err = s.TopicKeeper().SetTopic(ctx, topicId, topic)
	s.Require().NoError(err)

	// Enable reputer whitelist
	err = s.WhitelistsKeeper().EnableTopicReputerWhitelist(ctx, topicId)
	s.Require().NoError(err)

	// Test with no address provided
	req := &types.GetReputerSubmissionWindowStatusRequest{
		TopicId: topicId,
		Address: "",
	}
	response, err := queryServer.GetReputerSubmissionWindowStatus(ctx, req)
	s.Require().NoError(err, "Should not error with no address")
	s.Require().False(response.IsOpen, "Window should not be open with no active nonces")
	s.Require().False(response.IsRegistered, "Should be false when no address provided")
	s.Require().False(response.IsWhitelisted, "Should be false when no address provided")

	// Test with invalid address
	req.Address = "invalid_address"
	_, err = queryServer.GetReputerSubmissionWindowStatus(ctx, req)
	s.Require().Error(err, "Should error with invalid address format")

	// Add reputer to global whitelist first so they can be tested
	err = s.WhitelistsKeeper().AddToGlobalReputerWhitelist(ctx, reputerAddress)
	s.Require().NoError(err)

	// Test with valid unregistered address
	req.Address = reputerAddress
	response, err = queryServer.GetReputerSubmissionWindowStatus(ctx, req)
	s.Require().NoError(err)
	s.Require().False(response.IsRegistered)
	s.Require().True(response.IsWhitelisted) // Can submit via global whitelist

	// Register the reputer using MsgServer
	moduleParams, _ := s.ParamsKeeper().GetParams(ctx)
	s.FundAccount(moduleParams.RegistrationFee.Int64(), s.Addrs(0))
	registerMsg := &types.RegisterRequest{
		Sender:    reputerAddress,
		TopicId:   topicId,
		IsReputer: true,
		Owner:     reputerAddress,
	}
	_, err = s.EmissionsMsgServer().Register(ctx, registerMsg)
	s.Require().NoError(err)

	// Verify registration
	response, err = queryServer.GetReputerSubmissionWindowStatus(ctx, req)
	s.Require().NoError(err)
	s.Require().True(response.IsRegistered)

	// Add reputer to whitelist for deterministic test setup
	err = s.WhitelistsKeeper().AddToTopicReputerWhitelist(ctx, topicId, reputerAddress)
	s.Require().NoError(err)

	// Fund and activate topic
	err = s.StakingKeeper().AddReputerStake(ctx, topicId, s.AddrsStr(1), cosmosMath.NewInt(500000))
	s.Require().NoError(err)

	funderAddr := s.Addrs(2)
	s.FundTopic(topicId, funderAddr, cosmosMath.NewInt(10000))

	isActive, err := s.TopicKeeper().IsTopicActive(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(isActive)

	// Create multiple reputer nonces to test "latest active nonce" selection
	// Reputer windows: [nonce + GroundTruthLag, nonce + GroundTruthLag + extraLag + EpochLength]
	// extraLag = EpochLength - (GroundTruthLag % EpochLength) = 20 - (30 % 20) = 10

	reputerNonce1 := &types.Nonce{BlockHeight: 0}  // Window [30, 60] (0+30 to 0+30+10+20)
	reputerNonce2 := &types.Nonce{BlockHeight: 5}  // Window [35, 65] (5+30 to 5+30+10+20)
	reputerNonce3 := &types.Nonce{BlockHeight: 20} // Window [50, 80] (20+30 to 20+30+10+20)

	err = s.NonceKeeper().AddReputerNonce(ctx, topicId, reputerNonce1)
	s.Require().NoError(err)
	err = s.NonceKeeper().AddReputerNonce(ctx, topicId, reputerNonce2)
	s.Require().NoError(err)
	err = s.NonceKeeper().AddReputerNonce(ctx, topicId, reputerNonce3)
	s.Require().NoError(err)

	// Set current block to be within multiple reputer windows
	currentBlock = int64(40) // Within windows [30,60] and [35,65]
	s.WithBlockHeight(currentBlock)
	ctx = s.Ctx()

	// Test that it returns the LATEST active reputer nonce (nonce2 - most recent that includes current block)
	response, err = queryServer.GetReputerSubmissionWindowStatus(ctx, req)
	s.Require().NoError(err)
	s.Require().True(response.IsOpen)
	s.Require().Equal(reputerNonce2.BlockHeight, response.CurrentNonceBlockHeight) // Should be latest active
	s.Require().Equal(int64(35), response.WindowStartBlock)                        // 5 + 30
	s.Require().Equal(int64(65), response.WindowEndBlock)                          // 35 + 10 + 20

	// Next window: [20+30, 20+30+10+20] = [50, 80]
	expectedNextStart := int64(50) // From reputerNonce3 (BlockHeight=20)
	expectedNextEnd := int64(80)

	response, err = queryServer.GetReputerSubmissionWindowStatus(ctx, req)
	s.Require().NoError(err)
	s.Require().Equal(expectedNextStart, response.NextWindowStartBlock)
	s.Require().Equal(expectedNextEnd, response.NextWindowEndBlock)
}
