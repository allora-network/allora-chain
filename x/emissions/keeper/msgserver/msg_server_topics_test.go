package msgserver_test

import (
	"strings"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

/// Topics tests

func (s *MsgServerTestSuite) TestMsgCreateNewTopic() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	senderAddr := sdk.AccAddress(PKS[0].Address())
	sender := senderAddr.String()

	// Create a MsgCreateNewTopic message
	newTopicMsg := &types.MsgCreateNewTopic{
		Creator:         sender,
		Metadata:        "Some metadata for the new topic",
		LossLogic:       "logic",
		LossMethod:      "method",
		EpochLength:     10800,
		InferenceLogic:  "Ilogic",
		InferenceMethod: "Imethod",
		DefaultArg:      "ETH",
		AlphaRegret:     alloraMath.NewDecFromInt64(1),
		PNorm:           alloraMath.NewDecFromInt64(3),
		Epsilon:         alloraMath.MustNewDecFromString("0.01"),
	}

	s.MintTokensToAddress(senderAddr, types.DefaultParams().CreateTopicFee)

	// s.PrepareForCreateTopic(newTopicMsg.Creator)
	result, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err)
	s.Require().NotNil(result)

	pagination := &types.SimpleCursorPaginationRequest{
		Limit: 100,
	}
	activeTopics, _, err := s.emissionsKeeper.GetIdsOfActiveTopics(s.ctx, pagination)
	require.NoError(err)
	found := false
	for _, topicId := range activeTopics {
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

	senderAddr := sdk.AccAddress(PKS[0].Address())
	sender := senderAddr.String()

	// Create a MsgCreateNewTopic message
	newTopicMsg := &types.MsgCreateNewTopic{
		Creator:         sender,
		Metadata:        "Some metadata for the new topic",
		LossLogic:       "logic",
		LossMethod:      "method",
		EpochLength:     10800,
		InferenceLogic:  "Ilogic",
		InferenceMethod: "Imethod",
		DefaultArg:      "ETH",
		AlphaRegret:     alloraMath.NewDecFromInt64(1),
		PNorm:           alloraMath.NewDecFromInt64(3),
		Epsilon:         alloraMath.MustNewDecFromString("0"),
	}

	s.MintTokensToAddress(senderAddr, types.DefaultParams().CreateTopicFee)

	result, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.Error(err)
	require.True(strings.Contains(err.Error(), "epsilon must be greater than"))
	s.Require().Nil(result)
}

func (s *MsgServerTestSuite) TestUpdateTopicLossUpdateLastRan() {
	ctx := s.ctx
	require := s.Require()
	topicId := s.CreateOneTopic()

	// Mock setup for topic
	inferenceTs := int64(0x0)

	err := s.emissionsKeeper.UpdateTopicEpochLastEnded(ctx, topicId, inferenceTs)
	require.NoError(err, "UpdateTopicEpochLastEnded should not return an error")

	result, err := s.emissionsKeeper.GetTopicEpochLastEnded(s.ctx, topicId)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Equal(result, inferenceTs)
}
