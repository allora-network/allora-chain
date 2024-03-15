package msgserver_test

import (
	"time"

	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ########################################
// #           Topics tests              #
// ########################################

func (s *KeeperTestSuite) TestMsgCreateNewTopic() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Create a MsgCreateNewTopic message
	newTopicMsg := &types.MsgCreateNewTopic{
		Creator:          sdk.AccAddress(PKS[0].Address()).String(),
		Metadata:         "Some metadata for the new topic",
		LossLogic:        "logic",
		LossCadence:      10800,
		InferenceLogic:   "Ilogic",
		InferenceMethod:  "Imethod",
		InferenceCadence: 60,
		DefaultArg:       "ETH",
	}

	_, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err, "CreateTopic fails on first creation")

	result, err := s.emissionsKeeper.GetNumTopics(s.ctx)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Equal(result, uint64(1), "Topic count after first topic is not 1.")
}

func (s *KeeperTestSuite) TestMsgCreateNewTopicInvalidUnauthorized() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Create a MsgCreateNewTopic message
	newTopicMsg := &types.MsgCreateNewTopic{
		Creator:          nonAdminAccounts[0].String(),
		Metadata:         "Some metadata for the new topic",
		LossLogic:        "logic",
		LossCadence:      10800,
		InferenceLogic:   "Ilogic",
		InferenceMethod:  "Imethod",
		InferenceCadence: 60,
		DefaultArg:       "ETH",
	}

	_, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.ErrorIs(err, types.ErrNotInTopicCreationWhitelist, "CreateTopic should return an error")
}

func (s *KeeperTestSuite) TestMsgReactivateTopic() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	topicCreator := sdk.AccAddress(PKS[0].Address()).String()
	s.CreateOneTopic()

	// Deactivate topic
	s.emissionsKeeper.InactivateTopic(ctx, 0)

	// Set unmet demand for topic
	s.emissionsKeeper.SetTopicUnmetDemand(ctx, 0, cosmosMath.NewUint(100))

	// Create a MsgCreateNewTopic message
	reactivateTopicMsg := &types.MsgReactivateTopic{
		Sender:  topicCreator,
		TopicId: 0,
	}

	_, err := msgServer.ReactivateTopic(ctx, reactivateTopicMsg)
	require.NoError(err, "ReactivateTopic should not return an error")

	// Check if topic is active
	topic, err := s.emissionsKeeper.GetTopic(ctx, 0)
	require.NoError(err)
	require.True(topic.Active, "Topic should be active")
}

func (s *KeeperTestSuite) TestMsgReactivateTopicInvalidNotEnoughDemand() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	topicCreator := sdk.AccAddress(PKS[0].Address()).String()
	s.CreateOneTopic()

	// Deactivate topic
	s.emissionsKeeper.InactivateTopic(ctx, 0)

	// Create a MsgCreateNewTopic message
	reactivateTopicMsg := &types.MsgReactivateTopic{
		Sender:  topicCreator,
		TopicId: 0,
	}

	_, err := msgServer.ReactivateTopic(ctx, reactivateTopicMsg)
	require.ErrorIs(err, types.ErrTopicNotEnoughDemand, "ReactivateTopic should return an error")
}

func (s *KeeperTestSuite) TestUpdateTopicLossUpdateLastRan() {
	ctx := s.ctx
	require := s.Require()
	s.CreateOneTopic()

	// Mock setup for topic
	topicId := uint64(0)
	inferenceTs := uint64(time.Now().UTC().Unix())

	err := s.emissionsKeeper.UpdateTopicLossUpdateLastRan(ctx, topicId, inferenceTs)
	require.NoError(err, "UpdateTopicLossUpdateLastRan should not return an error")

	result, err := s.emissionsKeeper.GetTopicLossCalcLastRan(s.ctx, topicId)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Equal(result, inferenceTs)
}
