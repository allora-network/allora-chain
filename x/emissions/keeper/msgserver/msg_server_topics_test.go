package msgserver_test

import (
	"strings"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// Topics tests
//
//nolint:exhaustruct
func (s *MsgServerTestSuite) TestCreateNewTopic() {
	ctx := s.Ctx()
	msgServer := s.EmissionsMsgServer()
	keeper := s.EmissionsKeeper()
	senderAddr := s.Addrs(0)

	testCases := []struct {
		name          string
		setup         func() *types.CreateNewTopicRequest
		postCheck     func(topicId uint64)
		expectedError string
		expectSuccess bool
	}{
		{
			name: "Fails when sender not whitelisted",
			setup: func() *types.CreateNewTopicRequest {
				// Ensure sender is not whitelisted
				err := keeper.RemoveFromTopicCreatorWhitelist(ctx, senderAddr.String())
				s.Require().NoError(err)
				err = keeper.RemoveFromGlobalWhitelist(ctx, senderAddr.String())
				s.Require().NoError(err)

				return s.MockTopicMsg()
			},
			expectedError: types.ErrNotPermittedToCreateTopic.Error(),
			expectSuccess: false,
		},
		{
			name: "Success when sender is whitelisted",
			setup: func() *types.CreateNewTopicRequest {
				// Add sender to whitelist
				err := keeper.AddToTopicCreatorWhitelist(ctx, senderAddr.String())
				s.Require().NoError(err)

				return s.MockTopicMsg()
			},
			postCheck: func(topicId uint64) {
				// Check topic is not in active topics yet
				activeTopics, err := keeper.GetActiveTopicIdsAtBlock(ctx, 10800)
				s.Require().NoError(err)
				found := false
				for _, id := range activeTopics.TopicIds {
					if id == topicId {
						found = true
						break
					}
				}
				s.Require().False(found, "Added topic found in active topics")

				// Check worker whitelist is enabled
				enabled, err := keeper.IsTopicWorkerWhitelistEnabled(ctx, topicId)
				s.Require().NoError(err)
				s.Require().True(enabled, "Topic worker whitelist should be enabled")

				// Check reputer whitelist is enabled
				enabled, err = keeper.IsTopicReputerWhitelistEnabled(ctx, topicId)
				s.Require().NoError(err)
				s.Require().True(enabled, "Topic reputer whitelist should be enabled")
			},
			expectSuccess: true,
		},
		{
			name: "Fails with epsilon zero",
			setup: func() *types.CreateNewTopicRequest {
				// Add to whitelist to avoid that error
				err := keeper.AddToTopicCreatorWhitelist(ctx, senderAddr.String())
				s.Require().NoError(err)

				msg := s.MockTopicMsg()
				msg.Epsilon = alloraMath.MustNewDecFromString("0")
				return msg
			},
			expectedError: "epsilon must be greater than",
			expectSuccess: false,
		},
		{
			name: "Fails with too long metadata",
			setup: func() *types.CreateNewTopicRequest {
				// Add to whitelist
				err := keeper.AddToTopicCreatorWhitelist(ctx, senderAddr.String())
				s.Require().NoError(err)

				msg := s.MockTopicMsg()
				msg.Metadata = strings.Repeat("a", 257)
				return msg
			},
			expectedError: "metadata invalid",
			expectSuccess: false,
		},
		{
			name: "Fails with too long loss method",
			setup: func() *types.CreateNewTopicRequest {
				// Add to whitelist
				err := keeper.AddToTopicCreatorWhitelist(ctx, senderAddr.String())
				s.Require().NoError(err)

				msg := s.MockTopicMsg()
				msg.LossMethod = strings.Repeat("a", 257)
				return msg
			},
			expectedError: "loss method invalid",
			expectSuccess: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.MintTokensToAddress(senderAddr, types.DefaultParams().CreateTopicFee)
			msg := tc.setup()

			result, err := msgServer.CreateNewTopic(ctx, msg)

			if tc.expectSuccess {
				s.Require().NoError(err)
				s.Require().NotNil(result)
				if tc.postCheck != nil {
					tc.postCheck(result.TopicId)
				}
			} else {
				s.Require().Error(err)
				s.Require().Nil(result)
				if tc.expectedError != "" {
					s.Require().ErrorContains(err, tc.expectedError)
				}
			}
		})
	}
}

func (s *MsgServerTestSuite) TestUpdateTopicEpochLastEnded() {
	ctx := s.Ctx()
	keeper := s.EmissionsKeeper()
	topicId := uint64(1)
	inferenceTs := int64(20)

	err := keeper.UpdateTopicEpochLastEnded(ctx, topicId, inferenceTs)
	s.Require().NoError(err, "UpdateTopicEpochLastEnded should not return an error")

	topic, err := keeper.GetTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().NotNil(topic)
	s.Require().Equal(inferenceTs, topic.EpochLastEnded)
}

func (s *MsgServerTestSuite) TestUpdateTopicSuccess() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	senderAddr := s.Addrs(0)
	sender := s.AddrsStr(0)

	// Create a topic first
	s.MintTokensToAddress(senderAddr, types.DefaultParams().CreateTopicFee)
	createTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  sender,
		Metadata:                 "Original metadata",
		LossMethod:               "mse",
		EpochLength:              10800,
		GroundTruthLag:           10800,
		WorkerSubmissionWindow:   10,
		AllowNegative:            false,
		AlphaRegret:              alloraMath.NewDecFromInt64(1),
		PNorm:                    alloraMath.NewDecFromInt64(3),
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		EnableWorkerWhitelist:    false,
		EnableReputerWhitelist:   false,
	}

	createResult, err := msgServer.CreateNewTopic(ctx, createTopicMsg)
	require.NoError(err)
	require.NotNil(createResult)
	topicId := createResult.TopicId

	// Activate so updates are deferred to epoch end (pending path)
	require.NoError(s.EmissionsKeeper().ActivateTopic(ctx, topicId))

	// Update topic with new values
	updateTopicMsg := &types.UpdateTopicRequest{
		Sender:     sender,
		TopicId:    topicId,
		Metadata:   []string{"Updated metadata"},
		LossMethod: []string{"mae"},
	}

	updateResult, err := msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.NoError(err)
	require.NotNil(updateResult)

	// Verify that the pending update was stored
	hasPending, err := s.EmissionsKeeper().HasPendingTopicUpdate(ctx, topicId)
	require.NoError(err)
	require.True(hasPending, "Should have pending topic update")

	// Verify the pending update has the correct values
	pendingTopic, err := s.EmissionsKeeper().GetPendingTopicUpdate(ctx, topicId)
	require.NoError(err)
	require.Equal("Updated metadata", pendingTopic.Metadata)
	require.Equal("mae", pendingTopic.LossMethod)
}

func (s *MsgServerTestSuite) TestUpdateTopicNotTopicCreator() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	senderAddr := s.Addrs(0)
	sender := s.AddrsStr(0)
	otherUser := s.AddrsStr(1)

	// Create a topic
	s.MintTokensToAddress(senderAddr, types.DefaultParams().CreateTopicFee)
	createTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  sender,
		Metadata:                 "Original metadata",
		LossMethod:               "mse",
		EpochLength:              10800,
		GroundTruthLag:           10800,
		WorkerSubmissionWindow:   10,
		AllowNegative:            false,
		AlphaRegret:              alloraMath.NewDecFromInt64(1),
		PNorm:                    alloraMath.NewDecFromInt64(3),
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		EnableWorkerWhitelist:    false,
		EnableReputerWhitelist:   false,
	}

	createResult, err := msgServer.CreateNewTopic(ctx, createTopicMsg)
	require.NoError(err)
	topicId := createResult.TopicId

	// Try to update topic with different user
	updateTopicMsg := &types.UpdateTopicRequest{
		Sender:     otherUser,
		TopicId:    topicId,
		Metadata:   []string{"Updated metadata"},
		LossMethod: nil,
	}

	updateResult, err := msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.ErrorIs(err, types.ErrNotPermittedToModifyTopic)
	require.Nil(updateResult)
}

func (s *MsgServerTestSuite) TestUpdateTopicNonexistentTopic() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	sender := s.AddrsStr(0)
	nonexistentTopicId := uint64(999)

	updateTopicMsg := &types.UpdateTopicRequest{
		Sender:     sender,
		TopicId:    nonexistentTopicId,
		Metadata:   []string{"Updated metadata"},
		LossMethod: nil,
	}

	updateResult, err := msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.Error(err)
	require.Nil(updateResult)
	require.ErrorIs(err, types.ErrTopicDoesNotExist)
}

func (s *MsgServerTestSuite) TestUpdateTopicNoChanges() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	senderAddr := s.Addrs(0)
	sender := s.AddrsStr(0)

	// Create a topic
	s.MintTokensToAddress(senderAddr, types.DefaultParams().CreateTopicFee)
	createTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  sender,
		Metadata:                 "Original metadata",
		LossMethod:               "mse",
		EpochLength:              10800,
		GroundTruthLag:           10800,
		WorkerSubmissionWindow:   10,
		AllowNegative:            false,
		AlphaRegret:              alloraMath.NewDecFromInt64(1),
		PNorm:                    alloraMath.NewDecFromInt64(3),
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		EnableWorkerWhitelist:    false,
		EnableReputerWhitelist:   false,
	}

	createResult, err := msgServer.CreateNewTopic(ctx, createTopicMsg)
	require.NoError(err)
	topicId := createResult.TopicId

	// Try to update with no fields set
	updateTopicMsg := &types.UpdateTopicRequest{
		Sender:     sender,
		TopicId:    topicId,
		Metadata:   nil,
		LossMethod: nil,
	}

	updateResult, err := msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.ErrorIs(err, types.ErrNoUpdateFields)
	require.Nil(updateResult)
}

func (s *MsgServerTestSuite) TestUpdateTopicValidationInvalidFields() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	sender := s.AddrsStr(0)
	topicId := s.CreateTopic()

	// Test empty loss method
	updateTopicMsg := &types.UpdateTopicRequest{
		Sender:     sender,
		TopicId:    topicId,
		Metadata:   nil,
		LossMethod: []string{""},
	}
	updateResult, err := msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.Error(err)
	require.Nil(updateResult)
	require.ErrorContains(err, "loss method invalid")

	// Test too long loss method
	updateTopicMsg = &types.UpdateTopicRequest{
		Sender:     sender,
		TopicId:    topicId,
		Metadata:   nil,
		LossMethod: []string{strings.Repeat("a", 257)},
	}
	updateResult, err = msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.Error(err)
	require.Nil(updateResult)
	require.ErrorContains(err, "loss method invalid")

	// Test too long metadata
	updateTopicMsg = &types.UpdateTopicRequest{
		Sender:     sender,
		TopicId:    topicId,
		Metadata:   []string{strings.Repeat("a", 257)},
		LossMethod: nil,
	}
	updateResult, err = msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.Error(err)
	require.Nil(updateResult)
	require.ErrorContains(err, "metadata invalid")
}

func (s *MsgServerTestSuite) TestUpdateTopicSuccessfulUpdate() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	senderAddr := s.Addrs(0)
	sender := s.AddrsStr(0)

	// Fund the sender
	s.FundAccount(1000000, senderAddr)

	// Create a topic first
	createTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  sender,
		Metadata:                 "original metadata",
		LossMethod:               "mse",
		EpochLength:              100,
		GroundTruthLag:           100,
		WorkerSubmissionWindow:   10,
		AllowNegative:            false,
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		PNorm:                    alloraMath.MustNewDecFromString("3.0"),
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		EnableWorkerWhitelist:    false,
		EnableReputerWhitelist:   false,
	}

	createResult, err := msgServer.CreateNewTopic(ctx, createTopicMsg)
	require.NoError(err)
	topicId := createResult.TopicId

	// Get original topic to verify initial state
	originalTopic, err := s.EmissionsKeeper().GetTopic(ctx, topicId)
	require.NoError(err)
	require.Equal("original metadata", originalTopic.Metadata)
	require.Equal("mse", originalTopic.LossMethod)

	// Test successful update of allowed fields
	require.NoError(s.EmissionsKeeper().ActivateTopic(ctx, topicId))
	updateTopicMsg := &types.UpdateTopicRequest{
		Sender:     sender,
		TopicId:    topicId,
		Metadata:   []string{"updated metadata"},
		LossMethod: []string{"mae"},
	}

	updateResult, err := msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.NoError(err)
	require.NotNil(updateResult)

	// Verify pending update was created
	hasPending, err := s.EmissionsKeeper().HasPendingTopicUpdate(ctx, topicId)
	require.NoError(err)
	require.True(hasPending)

	// Get pending update and verify only allowed fields were changed
	pendingTopic, err := s.EmissionsKeeper().GetPendingTopicUpdate(ctx, topicId)
	require.NoError(err)
	require.Equal("updated metadata", pendingTopic.Metadata)
	require.Equal("mae", pendingTopic.LossMethod)
	// Verify restricted fields remain unchanged
	require.Equal(originalTopic.GroundTruthLag, pendingTopic.GroundTruthLag)
	require.Equal(originalTopic.WorkerSubmissionWindow, pendingTopic.WorkerSubmissionWindow)
	require.Equal(originalTopic.EpochLength, pendingTopic.EpochLength)
}

func (s *MsgServerTestSuite) TestUpdateTopicInactiveAppliesImmediately() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	senderAddr := s.Addrs(0)
	sender := s.AddrsStr(0)

	// Create a topic (starts inactive); ensure inactive explicitly if needed
	s.MintTokensToAddress(senderAddr, types.DefaultParams().CreateTopicFee)
	createTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  sender,
		Metadata:                 "Original metadata",
		LossMethod:               "mse",
		EpochLength:              10800,
		GroundTruthLag:           10800,
		WorkerSubmissionWindow:   10,
		AllowNegative:            false,
		AlphaRegret:              alloraMath.NewDecFromInt64(1),
		PNorm:                    alloraMath.NewDecFromInt64(3),
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		EnableWorkerWhitelist:    false,
		EnableReputerWhitelist:   false,
	}
	createResult, err := msgServer.CreateNewTopic(ctx, createTopicMsg)
	require.NoError(err)
	topicId := createResult.TopicId

	// Apply update; should be immediate (no pending)
	updateMsg := &types.UpdateTopicRequest{
		Sender:     sender,
		TopicId:    topicId,
		Metadata:   []string{"Updated metadata"},
		LossMethod: []string{"mae"},
	}
	_, err = msgServer.UpdateTopic(ctx, updateMsg)
	require.NoError(err)

	hasPending, err := s.EmissionsKeeper().HasPendingTopicUpdate(ctx, topicId)
	require.NoError(err)
	require.False(hasPending)

	updatedTopic, err := s.EmissionsKeeper().GetTopic(ctx, topicId)
	require.NoError(err)
	require.Equal("Updated metadata", updatedTopic.Metadata)
	require.Equal("mae", updatedTopic.LossMethod)
}

func (s *MsgServerTestSuite) TestUpdateTopicReplacesPendingUpdate() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	senderAddr := s.Addrs(0)
	sender := s.AddrsStr(0)

	// Create a topic
	s.MintTokensToAddress(senderAddr, types.DefaultParams().CreateTopicFee)
	createTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  sender,
		Metadata:                 "Original metadata",
		LossMethod:               "mse",
		EpochLength:              10800,
		GroundTruthLag:           10800,
		WorkerSubmissionWindow:   10,
		AllowNegative:            false,
		AlphaRegret:              alloraMath.NewDecFromInt64(1),
		PNorm:                    alloraMath.NewDecFromInt64(3),
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		EnableWorkerWhitelist:    false,
		EnableReputerWhitelist:   false,
	}
	createResult, err := msgServer.CreateNewTopic(ctx, createTopicMsg)
	require.NoError(err)
	topicId := createResult.TopicId

	// Ensure topic active so updates go to pending
	require.NoError(s.EmissionsKeeper().ActivateTopic(ctx, topicId))

	// First update changes metadata only
	firstUpdate := &types.UpdateTopicRequest{
		Sender:     sender,
		TopicId:    topicId,
		Metadata:   []string{"Updated metadata"},
		LossMethod: nil,
	}
	_, err = msgServer.UpdateTopic(ctx, firstUpdate)
	require.NoError(err)

	pending, err := s.EmissionsKeeper().GetPendingTopicUpdate(ctx, topicId)
	require.NoError(err)
	require.Equal("Updated metadata", pending.Metadata)
	require.Equal("mse", pending.LossMethod)

	// Second update (same epoch) replaces the pending update instead of merging
	secondUpdate := &types.UpdateTopicRequest{
		Sender:     sender,
		TopicId:    topicId,
		Metadata:   nil,
		LossMethod: []string{"mae"},
	}
	_, err = msgServer.UpdateTopic(ctx, secondUpdate)
	require.NoError(err)

	pending, err = s.EmissionsKeeper().GetPendingTopicUpdate(ctx, topicId)
	require.NoError(err)
	// Metadata falls back to original because the second update replaces the prior pending change
	require.Equal("Original metadata", pending.Metadata)
	require.Equal("mae", pending.LossMethod)
}
