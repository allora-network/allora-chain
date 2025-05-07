package msgserver_test

import (
	"strings"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// Topics tests

func (s *MsgServerTestSuite) TestMsgCreateNewTopicAndWhitelistCheck() {
	require := s.Require()

	senderAddr := s.Addrs()[0]
	sender := s.AddrsStr()[0]

	s.MintTokensToAddress(senderAddr, types.DefaultParams().CreateTopicFee)

	// Should fail if sender is not whitelisted
	err := s.EmissionsKeeper().RemoveFromTopicCreatorWhitelist(s.Ctx(), sender)
	require.NoError(err)
	err = s.EmissionsKeeper().RemoveFromGlobalWhitelist(s.Ctx(), sender)
	require.NoError(err)

	newTopicMsg := s.MockTopicMsg()
	result, err := s.EmissionsMsgServer().CreateNewTopic(s.Ctx(), newTopicMsg)
	require.ErrorIs(err, types.ErrNotPermittedToCreateTopic)
	require.Nil(result)

	// Add sender to whitelist
	err = s.EmissionsKeeper().AddToTopicCreatorWhitelist(s.Ctx(), sender)
	require.NoError(err)

	// Should now succeed
	result, err = s.EmissionsMsgServer().CreateNewTopic(s.Ctx(), newTopicMsg)
	require.NoError(err)
	require.NotNil(result)

	activeTopics, err := s.EmissionsKeeper().GetActiveTopicIdsAtBlock(s.Ctx(), 10800)
	require.NoError(err)
	found := false
	for _, topicId := range activeTopics.TopicIds {
		if topicId == result.TopicId {
			found = true
			break
		}
	}
	require.False(found, "Added topic found in active topics")

	enabled, err := s.EmissionsKeeper().IsTopicWorkerWhitelistEnabled(s.Ctx(), result.TopicId)
	require.NoError(err)
	require.True(enabled, "Topic worker whitelist should be enabled")

	enabled, err = s.EmissionsKeeper().IsTopicReputerWhitelistEnabled(s.Ctx(), result.TopicId)
	require.NoError(err)
	require.True(enabled, "Topic reputer whitelist should be enabled")
}

func (s *MsgServerTestSuite) TestMsgCreateNewTopicWithEpsilonZeroFails() {
	require := s.Require()

	senderAddr := s.Addrs()[0]

	// Create a CreateNewTopicRequest message
	newTopicMsg := s.MockTopicMsg()
	newTopicMsg.Epsilon = alloraMath.MustNewDecFromString("0")

	s.MintTokensToAddress(senderAddr, types.DefaultParams().CreateTopicFee)

	result, err := s.EmissionsMsgServer().CreateNewTopic(s.Ctx(), newTopicMsg)
	require.Error(err)
	require.True(strings.Contains(err.Error(), "epsilon must be greater than"))
	s.Require().Nil(result)
}

func (s *MsgServerTestSuite) TestUpdateTopicEpochLastEnded() {
	ctx := s.Ctx()
	require := s.Require()
	topicPrev := uint64(1)

	// Mock setup for topic
	inferenceTs := int64(20)

	err := s.EmissionsKeeper().UpdateTopicEpochLastEnded(ctx, topicPrev, inferenceTs)
	require.NoError(err, "UpdateTopicEpochLastEnded should not return an error")

	topic, err := s.EmissionsKeeper().GetTopic(s.Ctx(), topicPrev)
	s.Require().NoError(err)
	s.Require().NotNil(topic)
	s.Require().Equal(topic.EpochLastEnded, inferenceTs)
}

func (s *MsgServerTestSuite) TestMsgCreateNewTopicTooLongMetadataFails() {
	require := s.Require()

	senderAddr := s.Addrs()[0]
	newTopicMsg := s.MockTopicMsg()
	newTopicMsg.Metadata = strings.Repeat("a", 257)

	s.MintTokensToAddress(senderAddr, types.DefaultParams().CreateTopicFee)

	result, err := s.EmissionsMsgServer().CreateNewTopic(s.Ctx(), newTopicMsg)
	require.Error(err)
	require.Nil(result)
	require.ErrorContains(err, "metadata invalid")
}

func (s *MsgServerTestSuite) TestMsgCreateNewTopicTooLongLossMethodFails() {
	require := s.Require()

	senderAddr := s.Addrs()[0]
	newTopicMsg := s.MockTopicMsg()
	newTopicMsg.LossMethod = strings.Repeat("a", 257)

	s.MintTokensToAddress(senderAddr, types.DefaultParams().CreateTopicFee)

	result, err := s.EmissionsMsgServer().CreateNewTopic(s.Ctx(), newTopicMsg)
	require.Error(err)
	require.Nil(result)
	require.ErrorContains(err, "loss method invalid")
}
