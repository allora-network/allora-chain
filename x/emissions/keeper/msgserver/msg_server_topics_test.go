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
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	senderAddr := s.addrs[0]
	sender := s.addrsStr[0]

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

	// Update topic with new values
	updateTopicMsg := &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		Metadata:               []string{"Updated metadata"},
		LossMethod:             []string{"mae"},
		GroundTruthLag:         []int64{21600},
		WorkerSubmissionWindow: []int64{20},
	}

	updateResult, err := msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.NoError(err)
	require.NotNil(updateResult)

	// Verify that the pending update was stored
	hasPending, err := s.emissionsKeeper.HasPendingTopicUpdate(ctx, topicId)
	require.NoError(err)
	require.True(hasPending, "Should have pending topic update")

	// Verify the pending update has the correct values
	pendingTopic, err := s.emissionsKeeper.GetPendingTopicUpdate(ctx, topicId)
	require.NoError(err)
	require.Equal("Updated metadata", pendingTopic.Metadata)
	require.Equal("mae", pendingTopic.LossMethod)
	require.Equal(int64(21600), pendingTopic.GroundTruthLag)
	require.Equal(int64(20), pendingTopic.WorkerSubmissionWindow)

	// Verify other fields remained unchanged from original
	require.Equal(int64(10800), pendingTopic.EpochLength)
	require.Equal(alloraMath.NewDecFromInt64(3), pendingTopic.PNorm)
	require.Equal(alloraMath.NewDecFromInt64(1), pendingTopic.AlphaRegret)
}

func (s *MsgServerTestSuite) TestUpdateTopicNotTopicCreator() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	senderAddr := s.addrs[0]
	sender := s.addrsStr[0]
	otherUser := s.addrsStr[1]

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
		Sender:                 otherUser,
		TopicId:                topicId,
		Metadata:               []string{"Updated metadata"},
		LossMethod:             nil,
		GroundTruthLag:         nil,
		WorkerSubmissionWindow: nil,
	}

	updateResult, err := msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.ErrorIs(err, types.ErrNotTopicCreator)
	require.Nil(updateResult)

	// Verify no pending update was created
	hasPending, err := s.emissionsKeeper.HasPendingTopicUpdate(ctx, topicId)
	require.NoError(err)
	require.False(hasPending, "Should not have pending topic update")
}

func (s *MsgServerTestSuite) TestUpdateTopicNonexistentTopic() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	sender := s.addrsStr[0]
	nonexistentTopicId := uint64(999)

	updateTopicMsg := &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                nonexistentTopicId,
		Metadata:               []string{"Updated metadata"},
		LossMethod:             nil,
		GroundTruthLag:         nil,
		WorkerSubmissionWindow: nil,
	}

	updateResult, err := msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.Error(err)
	require.Nil(updateResult)
	require.ErrorIs(err, types.ErrTopicDoesNotExist)
}

func (s *MsgServerTestSuite) TestUpdateTopicValidationErrors() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	senderAddr := s.addrs[0]
	sender := s.addrsStr[0]

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

	// Test zero worker submission window
	updateTopicMsg := &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		WorkerSubmissionWindow: []int64{0},
		Metadata:               nil,
		LossMethod:             nil,
		GroundTruthLag:         nil,
	}
	updateResult, err := msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.Error(err)
	require.Nil(updateResult)
	require.ErrorIs(err, types.ErrInvalidValue)

	// Test too long metadata
	updateTopicMsg = &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		Metadata:               []string{strings.Repeat("a", 257)},
		LossMethod:             nil,
		GroundTruthLag:         nil,
		WorkerSubmissionWindow: nil,
	}
	updateResult, err = msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.Error(err)
	require.Nil(updateResult)
	require.ErrorContains(err, "metadata invalid")
}

func (s *MsgServerTestSuite) TestUpdateTopicNoChanges() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	senderAddr := s.addrs[0]
	sender := s.addrsStr[0]

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
		Sender:                 sender,
		TopicId:                topicId,
		Metadata:               nil,
		LossMethod:             nil,
		GroundTruthLag:         nil,
		WorkerSubmissionWindow: nil,
	}

	updateResult, err := msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.ErrorIs(err, types.ErrNoUpdateFields)
	require.Nil(updateResult)

	// Verify no pending update was created
	hasPending, err := s.emissionsKeeper.HasPendingTopicUpdate(ctx, topicId)
	require.NoError(err)
	require.False(hasPending, "Should not have pending topic update")
}

func (s *MsgServerTestSuite) TestUpdateTopicPartialUpdate() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	senderAddr := s.addrs[0]
	sender := s.addrsStr[0]

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

	// Update only metadata
	updateTopicMsg := &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		Metadata:               []string{"Updated metadata only"},
		LossMethod:             nil,
		GroundTruthLag:         nil,
		WorkerSubmissionWindow: nil,
	}

	updateResult, err := msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.NoError(err)
	require.NotNil(updateResult)

	// Verify the pending update has correct values
	pendingTopic, err := s.emissionsKeeper.GetPendingTopicUpdate(ctx, topicId)
	require.NoError(err)
	require.Equal("Updated metadata only", pendingTopic.Metadata)

	// Verify other fields remained unchanged from original
	originalTopic, err := s.emissionsKeeper.GetTopic(ctx, topicId)
	require.NoError(err)
	require.Equal(originalTopic.LossMethod, pendingTopic.LossMethod)
	require.Equal(originalTopic.EpochLength, pendingTopic.EpochLength)
	require.Equal(originalTopic.GroundTruthLag, pendingTopic.GroundTruthLag)
	require.Equal(originalTopic.PNorm, pendingTopic.PNorm)
	require.Equal(originalTopic.AlphaRegret, pendingTopic.AlphaRegret)
}

func (s *MsgServerTestSuite) TestUpdateTopicValidationInvalidFields() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	sender := s.addrsStr[0]
	topicId := s.CreateOneTopic().Id

	// Test empty loss method
	updateTopicMsg := &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		LossMethod:             []string{""},
		Metadata:               nil,
		GroundTruthLag:         nil,
		WorkerSubmissionWindow: nil,
	}
	updateResult, err := msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.Error(err)
	require.Nil(updateResult)
	require.ErrorIs(err, types.ErrInvalidValue)

	// Test too long loss method
	updateTopicMsg = &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		LossMethod:             []string{strings.Repeat("a", 257)},
		Metadata:               nil,
		GroundTruthLag:         nil,
		WorkerSubmissionWindow: nil,
	}
	updateResult, err = msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.Error(err)
	require.Nil(updateResult)
	require.ErrorContains(err, "loss method invalid")

	// Test zero worker submission window
	updateTopicMsg = &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		WorkerSubmissionWindow: []int64{0},
		Metadata:               nil,
		LossMethod:             nil,
		GroundTruthLag:         nil,
	}
	updateResult, err = msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.Error(err)
	require.Nil(updateResult)
	require.ErrorIs(err, types.ErrInvalidValue)

	// Test too long metadata
	updateTopicMsg = &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		Metadata:               []string{strings.Repeat("a", 257)},
		LossMethod:             nil,
		GroundTruthLag:         nil,
		WorkerSubmissionWindow: nil,
	}
	updateResult, err = msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.Error(err)
	require.Nil(updateResult)
	require.ErrorContains(err, "metadata invalid")
}

func (s *MsgServerTestSuite) TestUpdateTopicCrossFieldValidation() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	senderAddr := s.addrs[0]
	sender := s.addrsStr[0]

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

	// Test updating ground truth lag to be lower than existing epoch length
	updateTopicMsg := &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		GroundTruthLag:         []int64{5000}, // Lower than original epoch length (10800)
		Metadata:               nil,
		LossMethod:             nil,
		WorkerSubmissionWindow: nil,
	}
	updateResult, err := msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.Error(err)
	require.Nil(updateResult)
	require.ErrorIs(err, types.ErrInvalidValue)

	// Test updating worker submission window to be higher than existing epoch length
	updateTopicMsg = &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		WorkerSubmissionWindow: []int64{15000}, // Higher than original epoch length (10800)
		Metadata:               nil,
		LossMethod:             nil,
		GroundTruthLag:         nil,
	}
	updateResult, err = msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.Error(err)
	require.Nil(updateResult)
	require.ErrorIs(err, types.ErrInvalidValue)
}
