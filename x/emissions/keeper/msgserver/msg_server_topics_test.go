package msgserver_test

import (
	"strings"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

/// Topics tests

func (s *MsgServerTestSuite) TestMsgCreateNewTopic() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	senderAddr := s.addrs[0]
	sender := s.addrsStr[0]

	// Create a CreateNewTopicRequest message
	newTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  sender,
		Metadata:                 "Some metadata for the new topic",
		LossMethod:               "mse",
		EpochLength:              10800,
		GroundTruthLag:           10800,
		WorkerSubmissionWindow:   10,
		AlphaRegret:              alloraMath.NewDecFromInt64(1),
		PNorm:                    alloraMath.NewDecFromInt64(3),
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
	}

	s.MintTokensToAddress(senderAddr, types.DefaultParams().CreateTopicFee)

	// s.PrepareForCreateTopic(newTopicMsg.Creator)
	result, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err)
	s.Require().NotNil(result)

	activeTopics, err := s.emissionsKeeper.GetActiveTopicIdsAtBlock(s.ctx, 10800)
	require.NoError(err)
	found := false
	for _, topicId := range activeTopics.TopicIds {
		if topicId == result.TopicId {
			found = true
			break
		}
	}
	require.False(found, "Added topic found in active topics")
}

func (s *MsgServerTestSuite) TestMsgCreateNewTopicWithEpsilonZeroFails() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	senderAddr := s.addrs[0]
	sender := s.addrsStr[0]

	// Create a CreateNewTopicRequest message
	newTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  sender,
		Metadata:                 "Some metadata for the new topic",
		LossMethod:               "mse",
		EpochLength:              10800,
		GroundTruthLag:           10800,
		WorkerSubmissionWindow:   10,
		AlphaRegret:              alloraMath.NewDecFromInt64(1),
		PNorm:                    alloraMath.NewDecFromInt64(3),
		Epsilon:                  alloraMath.MustNewDecFromString("0"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
	}

	s.MintTokensToAddress(senderAddr, types.DefaultParams().CreateTopicFee)

	result, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.Error(err)
	require.True(strings.Contains(err.Error(), "epsilon must be greater than"))
	s.Require().Nil(result)
}

func (s *MsgServerTestSuite) TestUpdateTopicEpochLastEnded() {
	ctx := s.ctx
	require := s.Require()
	topicPrev := s.CreateOneTopic()

	// Mock setup for topic
	inferenceTs := int64(20)

	err := s.emissionsKeeper.UpdateTopicEpochLastEnded(ctx, topicPrev.Id, inferenceTs)
	require.NoError(err, "UpdateTopicEpochLastEnded should not return an error")

	topic, err := s.emissionsKeeper.GetTopic(s.ctx, topicPrev.Id)
	s.Require().NoError(err)
	s.Require().NotNil(topic)
	s.Require().Equal(topic.EpochLastEnded, inferenceTs)
}

func (s *MsgServerTestSuite) TestMsgCreateNewTopicTooLongMetadataFails() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	senderAddr := s.addrs[0]
	sender := s.addrsStr[0]

	newTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  sender,
		LossMethod:               "mse",
		EpochLength:              10800,
		GroundTruthLag:           10800,
		WorkerSubmissionWindow:   10,
		AlphaRegret:              alloraMath.NewDecFromInt64(1),
		PNorm:                    alloraMath.NewDecFromInt64(3),
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		Metadata:                 strings.Repeat("a", 257),
	}

	s.MintTokensToAddress(senderAddr, types.DefaultParams().CreateTopicFee)

	result, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.Error(err)
	require.Nil(result)
	require.ErrorContains(err, "metadata invalid")
}

func (s *MsgServerTestSuite) TestMsgCreateNewTopicTooLongLossMethodFails() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	senderAddr := s.addrs[0]
	sender := s.addrsStr[0]

	newTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  sender,
		Metadata:                 "Some metadata for the new topic",
		EpochLength:              10800,
		GroundTruthLag:           10800,
		WorkerSubmissionWindow:   10,
		AlphaRegret:              alloraMath.NewDecFromInt64(1),
		PNorm:                    alloraMath.NewDecFromInt64(3),
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		LossMethod:               strings.Repeat("a", 257),
	}

	s.MintTokensToAddress(senderAddr, types.DefaultParams().CreateTopicFee)

	result, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.Error(err)
	require.Nil(result)
	require.ErrorContains(err, "loss method invalid")
}
