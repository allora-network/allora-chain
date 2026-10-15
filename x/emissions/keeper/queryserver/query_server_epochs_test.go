package queryserver_test

import (
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *QueryServerTestSuite) TestGetEpochAndGetTopicEpochs() {
	ctx := s.Ctx()
	queryServer := s.EmissionsQueryServer()

	topicID := s.CreateTopic(
		testutil.WithEpochLength(60),
		testutil.WithGroundTruthLag(60),
		testutil.WithWorkerSubmissionWindow(10),
	)
	s.Require().NoError(s.EmissionsKeeper().StartNewEpoch(ctx, topicID))

	lastNonce, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, topicID)
	s.Require().NoError(err)
	s.Require().True(found)

	got, err := queryServer.GetEpoch(ctx, &types.GetEpochRequest{
		TopicId: topicID,
		Nonce:   uint64(lastNonce),
	})
	s.Require().NoError(err)
	s.Require().NotNil(got.Epoch)
	s.Require().Equal(topicID, got.Epoch.TopicId)
	s.Require().Equal(lastNonce, got.Epoch.Nonce)
	s.Require().Equal(types.EpochState_WORKER_SUBMISSION, got.Epoch.State)
	s.Require().Equal(ctx.BlockHeight(), got.Epoch.StartBlockHeight)

	list, err := queryServer.GetTopicEpochs(ctx, &types.GetTopicEpochsRequest{TopicId: topicID})
	s.Require().NoError(err)
	s.Require().Len(list.Epochs, 1)
	s.Require().Equal(lastNonce, list.Epochs[0].Nonce)

	// Start a second epoch so the topic has two live records.
	s.Require().NoError(s.EmissionsKeeper().StartNewEpoch(ctx, topicID))
	list, err = queryServer.GetTopicEpochs(ctx, &types.GetTopicEpochsRequest{TopicId: topicID})
	s.Require().NoError(err)
	s.Require().Len(list.Epochs, 2)
	s.Require().True(list.Epochs[0].Nonce < list.Epochs[1].Nonce)
}

func (s *QueryServerTestSuite) TestGetEpochNotFound() {
	ctx := s.Ctx()
	queryServer := s.EmissionsQueryServer()
	topicID := s.CreateTopic(
		testutil.WithEpochLength(60),
		testutil.WithGroundTruthLag(60),
		testutil.WithWorkerSubmissionWindow(10),
	)

	_, err := queryServer.GetEpoch(ctx, &types.GetEpochRequest{
		TopicId: topicID,
		Nonce:   uint64(types.ZeroNonce().NextNonce()),
	})
	s.Require().Error(err)
	s.Require().Equal(codes.NotFound, status.Code(err))

	list, err := queryServer.GetTopicEpochs(ctx, &types.GetTopicEpochsRequest{TopicId: topicID})
	s.Require().NoError(err)
	s.Require().Empty(list.Epochs)
}
