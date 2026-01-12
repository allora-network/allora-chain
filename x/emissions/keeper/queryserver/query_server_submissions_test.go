package queryserver_test

import (
	testutil "github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *QueryServerTestSuite) TestGetOpenReputerSubmissionWindows() {
	keeper := s.EmissionsKeeper()
	queryServer := s.EmissionsQueryServer()
	topicId := s.CreateTopic(
		testutil.WithEpochLength(100),
		testutil.WithGroundTruthLag(100),
		testutil.WithWorkerSubmissionWindow(50),
	)

	// Initially, no open nonces
	req := &types.GetOpenReputerSubmissionWindowsRequest{
		TopicId: topicId,
	}
	response, err := queryServer.GetOpenReputerSubmissionWindows(s.Ctx(), req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Empty(response.Nonces.Nonces, "Initially should have no open nonces")

	// Add nonces at different block heights
	// Nonce 1: block 1000, will be open at block 1100-1200
	nonce1 := &types.Nonce{BlockHeight: 1000}
	err = keeper.AddReputerNonce(s.Ctx(), topicId, nonce1)
	s.Require().NoError(err)

	// Nonce 2: block 2000, will be open at block 2100-2200
	nonce2 := &types.Nonce{BlockHeight: 2000}
	err = keeper.AddReputerNonce(s.Ctx(), topicId, nonce2)
	s.Require().NoError(err)

	// Set block height to 1150 (within window for nonce1, outside for nonce2)
	s.WithBlockHeight(1150)
	response, err = queryServer.GetOpenReputerSubmissionWindows(s.Ctx(), req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Nonces.Nonces, 1, "Should have one open nonce")
	s.Require().Equal(int64(1000), response.Nonces.Nonces[0].ReputerNonce.BlockHeight, "Should be nonce1")

	// Set block height to 2150 (within window for nonce2, outside for nonce1)
	s.WithBlockHeight(2150)
	response, err = queryServer.GetOpenReputerSubmissionWindows(s.Ctx(), req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Nonces.Nonces, 1, "Should have one open nonce")
	s.Require().Equal(int64(2000), response.Nonces.Nonces[0].ReputerNonce.BlockHeight, "Should be nonce2")

	// Set block height to 1050 (before window opens for nonce1)
	s.WithBlockHeight(1050)
	response, err = queryServer.GetOpenReputerSubmissionWindows(s.Ctx(), req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Empty(response.Nonces.Nonces, "Should have no open nonces before window opens")

	// Set block height to 1200 (exactly at window end for nonce1, inclusive)
	s.WithBlockHeight(1200)
	response, err = queryServer.GetOpenReputerSubmissionWindows(s.Ctx(), req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Nonces.Nonces, 1, "Should have one open nonce at window end")
	s.Require().Equal(int64(1000), response.Nonces.Nonces[0].ReputerNonce.BlockHeight, "Should be nonce1")
}

func (s *QueryServerTestSuite) TestGetOpenWorkerSubmissionWindows() {
	keeper := s.EmissionsKeeper()
	queryServer := s.EmissionsQueryServer()
	topicId := s.CreateTopic(
		testutil.WithEpochLength(100),
		testutil.WithGroundTruthLag(100),
		testutil.WithWorkerSubmissionWindow(50),
	)

	// Initially, no open nonces
	req := &types.GetOpenWorkerSubmissionWindowsRequest{
		TopicId: topicId,
	}
	response, err := queryServer.GetOpenWorkerSubmissionWindows(s.Ctx(), req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Empty(response.Nonces.Nonces, "Initially should have no open nonces")

	// Add nonces at different block heights
	// Nonce 1: block 1000, will be open at block 1000-1050
	nonce1 := &types.Nonce{BlockHeight: 1000}
	err = keeper.AddWorkerNonce(s.Ctx(), topicId, nonce1)
	s.Require().NoError(err)

	// Nonce 2: block 2000, will be open at block 2000-2050
	nonce2 := &types.Nonce{BlockHeight: 2000}
	err = keeper.AddWorkerNonce(s.Ctx(), topicId, nonce2)
	s.Require().NoError(err)

	// Set block height to 1025 (within window for nonce1, outside for nonce2)
	s.WithBlockHeight(1025)
	response, err = queryServer.GetOpenWorkerSubmissionWindows(s.Ctx(), req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Nonces.Nonces, 1, "Should have one open nonce")
	s.Require().Equal(int64(1000), response.Nonces.Nonces[0].BlockHeight, "Should be nonce1")

	// Set block height to 2025 (within window for nonce2, outside for nonce1)
	s.WithBlockHeight(2025)
	response, err = queryServer.GetOpenWorkerSubmissionWindows(s.Ctx(), req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Nonces.Nonces, 1, "Should have one open nonce")
	s.Require().Equal(int64(2000), response.Nonces.Nonces[0].BlockHeight, "Should be nonce2")

	// Set block height to 999 (before window opens for nonce1)
	s.WithBlockHeight(999)
	response, err = queryServer.GetOpenWorkerSubmissionWindows(s.Ctx(), req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Empty(response.Nonces.Nonces, "Should have no open nonces before window opens")

	// Set block height to 1000 (exactly at window start for nonce1, inclusive)
	s.WithBlockHeight(1000)
	response, err = queryServer.GetOpenWorkerSubmissionWindows(s.Ctx(), req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Nonces.Nonces, 1, "Should have one open nonce at window start")
	s.Require().Equal(int64(1000), response.Nonces.Nonces[0].BlockHeight, "Should be nonce1")

	// Set block height to 1050 (exactly at window end for nonce1, inclusive)
	s.WithBlockHeight(1050)
	response, err = queryServer.GetOpenWorkerSubmissionWindows(s.Ctx(), req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Nonces.Nonces, 1, "Should have one open nonce at window end")
	s.Require().Equal(int64(1000), response.Nonces.Nonces[0].BlockHeight, "Should be nonce1")
}

func (s *QueryServerTestSuite) TestGetOpenReputerSubmissionWindowsWithMultipleNoncesInWindow() {
	keeper := s.EmissionsKeeper()
	queryServer := s.EmissionsQueryServer()
	topicId := s.CreateTopic(
		testutil.WithEpochLength(100),
		testutil.WithGroundTruthLag(100),
		testutil.WithWorkerSubmissionWindow(50),
	)

	// Add multiple nonces that will be open at the same time
	nonce1 := &types.Nonce{BlockHeight: 1000} // Open at 1100-1200
	nonce2 := &types.Nonce{BlockHeight: 1100} // Open at 1200-1300
	err := keeper.AddReputerNonce(s.Ctx(), topicId, nonce1)
	s.Require().NoError(err)
	err = keeper.AddReputerNonce(s.Ctx(), topicId, nonce2)
	s.Require().NoError(err)

	req := &types.GetOpenReputerSubmissionWindowsRequest{
		TopicId: topicId,
	}

	// Set block height to 1200 (both nonces are open - nonce1 at end, nonce2 at start)
	s.WithBlockHeight(1200)
	response, err := queryServer.GetOpenReputerSubmissionWindows(s.Ctx(), req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Nonces.Nonces, 2, "Should have two open nonces")
	// Check both are present
	nonceHeights := []int64{
		response.Nonces.Nonces[0].ReputerNonce.BlockHeight,
		response.Nonces.Nonces[1].ReputerNonce.BlockHeight,
	}
	s.Require().Contains(nonceHeights, int64(1000), "Should contain nonce1")
	s.Require().Contains(nonceHeights, int64(1100), "Should contain nonce2")
}

func (s *QueryServerTestSuite) TestGetOpenReputerSubmissionWindowsWithNonExistentTopic() {
	queryServer := s.EmissionsQueryServer()
	nonExistentTopicId := uint64(99999)

	req := &types.GetOpenReputerSubmissionWindowsRequest{
		TopicId: nonExistentTopicId,
	}
	response, err := queryServer.GetOpenReputerSubmissionWindows(s.Ctx(), req)
	s.Require().Error(err)
	s.Require().Nil(response)
}

func (s *QueryServerTestSuite) TestGetOpenWorkerSubmissionWindowsWithNonExistentTopic() {
	queryServer := s.EmissionsQueryServer()
	nonExistentTopicId := uint64(99999)

	req := &types.GetOpenWorkerSubmissionWindowsRequest{
		TopicId: nonExistentTopicId,
	}
	response, err := queryServer.GetOpenWorkerSubmissionWindows(s.Ctx(), req)
	s.Require().Error(err)
	s.Require().Nil(response)
}
