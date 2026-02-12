package msgserver_test

import (
	"strings"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

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
		{
			name: "Fails with CNorm below -100",
			setup: func() *types.CreateNewTopicRequest {
				err := keeper.AddToTopicCreatorWhitelist(ctx, senderAddr.String())
				s.Require().NoError(err)

				msg := s.MockTopicMsg()
				msg.CNorm = alloraMath.MustNewDecFromString("-101")
				return msg
			},
			expectedError: "c_norm must be between -100 and 100",
			expectSuccess: false,
		},
		{
			name: "Fails with CNorm above 100",
			setup: func() *types.CreateNewTopicRequest {
				err := keeper.AddToTopicCreatorWhitelist(ctx, senderAddr.String())
				s.Require().NoError(err)

				msg := s.MockTopicMsg()
				msg.CNorm = alloraMath.MustNewDecFromString("101")
				return msg
			},
			expectedError: "c_norm must be between -100 and 100",
			expectSuccess: false,
		},
		{
			name: "Success with valid CNorm value",
			setup: func() *types.CreateNewTopicRequest {
				err := keeper.AddToTopicCreatorWhitelist(ctx, senderAddr.String())
				s.Require().NoError(err)

				msg := s.MockTopicMsg()
				msg.CNorm = alloraMath.MustNewDecFromString("0.75")
				return msg
			},
			postCheck: func(topicId uint64) {
				topic, err := keeper.GetTopic(ctx, topicId)
				s.Require().NoError(err)
				s.Require().Equal(alloraMath.MustNewDecFromString("0.75"), topic.CNorm)
			},
			expectSuccess: true,
		},
		{
			name: "Fails when require_unity true with SINGLE output_arity",
			setup: func() *types.CreateNewTopicRequest {
				_ = keeper.AddToTopicCreatorWhitelist(ctx, senderAddr.String())
				msg := s.MockTopicMsg()
				msg.OutputArity = types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE
				msg.RequireUnity = true
				msg.UnityTolerance = alloraMath.MustNewDecFromString("0.1")
				return msg
			},
			expectedError: "require_unity MUST be false when output_arity is SINGLE",
			expectSuccess: false,
		},
		{
			name: "Fails when require_unity true and unity_tolerance <= 0",
			setup: func() *types.CreateNewTopicRequest {
				_ = keeper.AddToTopicCreatorWhitelist(ctx, senderAddr.String())
				msg := s.MockTopicMsg()
				msg.OutputArity = types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI
				msg.RequireUnity = true
				msg.UnityTolerance = alloraMath.ZeroDec()
				return msg
			},
			expectedError: "unity_tolerance must be in",
			expectSuccess: false,
		},
		{
			name: "Fails when require_unity true and unity_tolerance > max",
			setup: func() *types.CreateNewTopicRequest {
				_ = keeper.AddToTopicCreatorWhitelist(ctx, senderAddr.String())
				msg := s.MockTopicMsg()
				msg.RequireUnity = true
				msg.UnityTolerance, _ = alloraMath.MustNewDecFromString("0.01").Add(alloraMath.MustNewDecFromString("0.0000000001"))
				return msg
			},
			expectedError: "unity_tolerance must be in",
			expectSuccess: false,
		},
		{
			name: "Success when require_unity true and unity_tolerance valid",
			setup: func() *types.CreateNewTopicRequest {
				_ = keeper.AddToTopicCreatorWhitelist(ctx, senderAddr.String())
				msg := s.MockTopicMsg()
				msg.RequireUnity = true
				msg.UnityTolerance = alloraMath.MustNewDecFromString("0.01")
				return msg
			},
			postCheck: func(topicId uint64) {
				topic, err := keeper.GetTopic(ctx, topicId)
				s.Require().NoError(err)
				s.Require().True(topic.RequireUnity)
				s.Require().Equal(alloraMath.MustNewDecFromString("0.01"), topic.UnityTolerance)
				s.Require().Equal(types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI, topic.OutputArity)
			},
			expectSuccess: true,
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
		CNorm:                    alloraMath.MustNewDecFromString("0.75"),
	}

	createResult, err := msgServer.CreateNewTopic(ctx, createTopicMsg)
	require.NoError(err)
	require.NotNil(createResult)
	topicId := createResult.TopicId

	originalTopic, err := s.EmissionsKeeper().GetTopic(ctx, topicId)
	require.NoError(err)

	// Update topic with new values
	updateTopicMsg := &types.UpdateTopicRequest{
		Sender:              sender,
		TopicId:             topicId,
		Metadata:            "Updated metadata",
		LossMethod:          "mae",
		AlphaRegret:         alloraMath.NewDecFromInt64(1),
		MeritSortitionAlpha: alloraMath.MustNewDecFromString("0.1"),
		PNorm:               alloraMath.NewDecFromInt64(3),
		CNorm:               alloraMath.MustNewDecFromString("0.75"),
	}

	updateResult, err := msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.NoError(err)
	require.NotNil(updateResult)

	// Verify topic updated
	updatedTopic, err := s.EmissionsKeeper().GetTopic(ctx, topicId)
	require.NoError(err)
	require.Equal("Updated metadata", updatedTopic.Metadata)
	require.Equal("mae", updatedTopic.LossMethod)

	require.Equal(originalTopic.OutputArity, updatedTopic.OutputArity)
	require.Equal(originalTopic.RequireUnity, updatedTopic.RequireUnity)
	require.Equal(originalTopic.UnityTolerance, updatedTopic.UnityTolerance)
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
		CNorm:                    alloraMath.MustNewDecFromString("0.75"),
	}

	createResult, err := msgServer.CreateNewTopic(ctx, createTopicMsg)
	require.NoError(err)
	topicId := createResult.TopicId

	// Try to update topic with different user
	updateTopicMsg := &types.UpdateTopicRequest{
		Sender:              otherUser,
		TopicId:             topicId,
		Metadata:            "Updated metadata",
		LossMethod:          "mse",
		AlphaRegret:         alloraMath.NewDecFromInt64(1),
		MeritSortitionAlpha: alloraMath.MustNewDecFromString("0.1"),
		PNorm:               alloraMath.NewDecFromInt64(3),
		CNorm:               alloraMath.MustNewDecFromString("0.75"),
	}

	updateResult, err := msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.ErrorIs(err, sdkerrors.ErrUnauthorized)
	require.Nil(updateResult)
}

func (s *MsgServerTestSuite) TestUpdateTopicNonexistentTopic() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	sender := s.AddrsStr(0)
	nonexistentTopicId := uint64(999)

	updateTopicMsg := &types.UpdateTopicRequest{
		Sender:              sender,
		TopicId:             nonexistentTopicId,
		Metadata:            "Updated metadata",
		LossMethod:          "mse",
		AlphaRegret:         alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha: alloraMath.MustNewDecFromString("0.1"),
		PNorm:               alloraMath.MustNewDecFromString("3.0"),
		CNorm:               alloraMath.MustNewDecFromString("0.75"),
	}

	updateResult, err := msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.Error(err)
	require.Nil(updateResult)
	require.ErrorIs(err, types.ErrTopicDoesNotExist)
}

func (s *MsgServerTestSuite) TestUpdateTopicValidationInvalidFields() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	sender := s.AddrsStr(0)
	topicId := s.CreateTopic()

	// Test empty loss method
	updateTopicMsg := &types.UpdateTopicRequest{
		Sender:              sender,
		TopicId:             topicId,
		Metadata:            "valid metadata",
		LossMethod:          "",
		AlphaRegret:         alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha: alloraMath.MustNewDecFromString("0.1"),
		PNorm:               alloraMath.MustNewDecFromString("3.0"),
		CNorm:               alloraMath.MustNewDecFromString("0.75"),
	}
	updateResult, err := msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.Error(err)
	require.Nil(updateResult)
	require.ErrorContains(err, "loss method invalid")

	// Test too long loss method
	updateTopicMsg = &types.UpdateTopicRequest{
		Sender:              sender,
		TopicId:             topicId,
		Metadata:            "valid metadata",
		LossMethod:          strings.Repeat("a", 257),
		AlphaRegret:         alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha: alloraMath.MustNewDecFromString("0.1"),
		PNorm:               alloraMath.MustNewDecFromString("3.0"),
		CNorm:               alloraMath.MustNewDecFromString("0.75"),
	}
	updateResult, err = msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.Error(err)
	require.Nil(updateResult)
	require.ErrorContains(err, "loss method invalid")

	// Test too long metadata
	updateTopicMsg = &types.UpdateTopicRequest{
		Sender:              sender,
		TopicId:             topicId,
		Metadata:            strings.Repeat("a", 257),
		LossMethod:          "mse",
		AlphaRegret:         alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha: alloraMath.MustNewDecFromString("0.1"),
		PNorm:               alloraMath.MustNewDecFromString("3.0"),
		CNorm:               alloraMath.MustNewDecFromString("0.75"),
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
		CNorm:                    alloraMath.MustNewDecFromString("0.75"),
	}

	createResult, err := msgServer.CreateNewTopic(ctx, createTopicMsg)
	require.NoError(err)
	topicId := createResult.TopicId

	// Get original topic to verify initial state
	originalTopic, err := s.EmissionsKeeper().GetTopic(ctx, topicId)
	require.NoError(err)
	require.Equal("original metadata", originalTopic.Metadata)
	require.Equal("mse", originalTopic.LossMethod)

	// Test successful update
	updateTopicMsg := &types.UpdateTopicRequest{
		Sender:              sender,
		TopicId:             topicId,
		Metadata:            "updated metadata",
		LossMethod:          "mae",
		AlphaRegret:         alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha: alloraMath.MustNewDecFromString("0.1"),
		PNorm:               alloraMath.MustNewDecFromString("3.0"),
		CNorm:               alloraMath.MustNewDecFromString("0.75"),
	}

	updateResult, err := msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.NoError(err)
	require.NotNil(updateResult)

	// Verify topic updated and only allowed fields changed
	updatedTopic, err := s.EmissionsKeeper().GetTopic(ctx, topicId)
	require.NoError(err)
	require.Equal("updated metadata", updatedTopic.Metadata)
	require.Equal("mae", updatedTopic.LossMethod)
	// Verify restricted fields remain unchanged
	require.Equal(originalTopic.GroundTruthLag, updatedTopic.GroundTruthLag)
	require.Equal(originalTopic.WorkerSubmissionWindow, updatedTopic.WorkerSubmissionWindow)
	require.Equal(originalTopic.EpochLength, updatedTopic.EpochLength)
}

func (s *MsgServerTestSuite) TestUpdateTopicNumericParams() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	senderAddr := s.Addrs(0)
	sender := s.AddrsStr(0)

	s.MintTokensToAddress(senderAddr, types.DefaultParams().CreateTopicFee)
	createTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  sender,
		Metadata:                 "Original metadata",
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
		CNorm:                    alloraMath.MustNewDecFromString("0.75"),
	}
	createResult, err := msgServer.CreateNewTopic(ctx, createTopicMsg)
	require.NoError(err)
	topicId := createResult.TopicId

	updateTopicMsg := &types.UpdateTopicRequest{
		Sender:              sender,
		TopicId:             topicId,
		Metadata:            "Original metadata",
		LossMethod:          "mse",
		AlphaRegret:         alloraMath.MustNewDecFromString("0.25"),
		MeritSortitionAlpha: alloraMath.MustNewDecFromString("0.3"),
		PNorm:               alloraMath.MustNewDecFromString("3.5"),
		CNorm:               alloraMath.MustNewDecFromString("0.75"),
	}

	_, err = msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.NoError(err)

	updatedTopic, err := s.EmissionsKeeper().GetTopic(ctx, topicId)
	require.NoError(err)
	require.Equal(alloraMath.MustNewDecFromString("0.25"), updatedTopic.AlphaRegret)
	require.Equal(alloraMath.MustNewDecFromString("0.3"), updatedTopic.MeritSortitionAlpha)
	require.Equal(alloraMath.MustNewDecFromString("3.5"), updatedTopic.PNorm)

	// Test updating CNorm with valid values
	updateTopicMsg = &types.UpdateTopicRequest{
		Sender:              sender,
		TopicId:             topicId,
		Metadata:            "Original metadata",
		LossMethod:          "mse",
		AlphaRegret:         alloraMath.MustNewDecFromString("0.25"),
		MeritSortitionAlpha: alloraMath.MustNewDecFromString("0.3"),
		PNorm:               alloraMath.MustNewDecFromString("3.5"),
		CNorm:               alloraMath.MustNewDecFromString("50.5"),
	}
	_, err = msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.NoError(err)

	updatedTopic, err = s.EmissionsKeeper().GetTopic(ctx, topicId)
	require.NoError(err)
	require.Equal(alloraMath.MustNewDecFromString("50.5"), updatedTopic.CNorm)

	// Test updating CNorm to boundary value -100
	updateTopicMsg = &types.UpdateTopicRequest{
		Sender:              sender,
		TopicId:             topicId,
		Metadata:            "Original metadata",
		LossMethod:          "mse",
		AlphaRegret:         alloraMath.MustNewDecFromString("0.25"),
		MeritSortitionAlpha: alloraMath.MustNewDecFromString("0.3"),
		PNorm:               alloraMath.MustNewDecFromString("3.5"),
		CNorm:               alloraMath.MustNewDecFromString("-100"),
	}
	_, err = msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.NoError(err)

	updatedTopic, err = s.EmissionsKeeper().GetTopic(ctx, topicId)
	require.NoError(err)
	require.Equal(alloraMath.MustNewDecFromString("-100"), updatedTopic.CNorm)

	// Test updating CNorm to boundary value 100
	updateTopicMsg = &types.UpdateTopicRequest{
		Sender:              sender,
		TopicId:             topicId,
		Metadata:            "Original metadata",
		LossMethod:          "mse",
		AlphaRegret:         alloraMath.MustNewDecFromString("0.25"),
		MeritSortitionAlpha: alloraMath.MustNewDecFromString("0.3"),
		PNorm:               alloraMath.MustNewDecFromString("3.5"),
		CNorm:               alloraMath.MustNewDecFromString("100"),
	}
	_, err = msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.NoError(err)

	updatedTopic, err = s.EmissionsKeeper().GetTopic(ctx, topicId)
	require.NoError(err)
	require.Equal(alloraMath.MustNewDecFromString("100"), updatedTopic.CNorm)

	// Add a fulfilled nonce (window closed) and ensure updates still allowed
	s.Require().NoError(s.EmissionsKeeper().AddWorkerNonce(ctx, topicId, &types.Nonce{BlockHeight: 1}))
	// Close window by moving block height beyond worker submission window
	s.WithBlockHeight(50)
	ctx = s.Ctx()
	updateTopicMsg = &types.UpdateTopicRequest{
		Sender:              sender,
		TopicId:             topicId,
		Metadata:            "Original metadata",
		LossMethod:          "mse",
		AlphaRegret:         alloraMath.MustNewDecFromString("0.25"),
		MeritSortitionAlpha: alloraMath.MustNewDecFromString("0.4"),
		PNorm:               alloraMath.MustNewDecFromString("3.5"),
		CNorm:               alloraMath.MustNewDecFromString("100"),
	}
	_, err = msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.NoError(err)
}

func (s *MsgServerTestSuite) TestUpdateTopicNumericParamsInvalid() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	sender := s.AddrsStr(0)
	topicId := s.CreateTopic()

	// Invalid alpha_regret (<=0)
	updateTopicMsg := &types.UpdateTopicRequest{
		Sender:              sender,
		TopicId:             topicId,
		Metadata:            "metadata",
		LossMethod:          "mse",
		AlphaRegret:         alloraMath.ZeroDec(),
		MeritSortitionAlpha: alloraMath.MustNewDecFromString("0.1"),
		PNorm:               alloraMath.MustNewDecFromString("3.0"),
		CNorm:               alloraMath.MustNewDecFromString("0.75"),
	}
	_, err := msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.ErrorContains(err, "alpha regret")

	// Invalid merit_sortition_alpha (>1)
	updateTopicMsg = &types.UpdateTopicRequest{
		Sender:              sender,
		TopicId:             topicId,
		Metadata:            "metadata",
		LossMethod:          "mse",
		AlphaRegret:         alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha: alloraMath.MustNewDecFromString("1.1"),
		PNorm:               alloraMath.MustNewDecFromString("3.0"),
		CNorm:               alloraMath.MustNewDecFromString("0.75"),
	}
	_, err = msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.ErrorContains(err, "merit sortition alpha")

	// Invalid p_norm (below range)
	updateTopicMsg = &types.UpdateTopicRequest{
		Sender:              sender,
		TopicId:             topicId,
		Metadata:            "metadata",
		LossMethod:          "mse",
		AlphaRegret:         alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha: alloraMath.MustNewDecFromString("0.1"),
		PNorm:               alloraMath.MustNewDecFromString("2.0"),
		CNorm:               alloraMath.MustNewDecFromString("0.75"),
	}
	_, err = msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.ErrorContains(err, "p-norm")

	// Invalid c_norm (below -100)
	updateTopicMsg = &types.UpdateTopicRequest{
		Sender:              sender,
		TopicId:             topicId,
		Metadata:            "metadata",
		LossMethod:          "mse",
		AlphaRegret:         alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha: alloraMath.MustNewDecFromString("0.1"),
		PNorm:               alloraMath.MustNewDecFromString("3.0"),
		CNorm:               alloraMath.MustNewDecFromString("-101"),
	}
	_, err = msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.ErrorContains(err, "c_norm")

	// Invalid c_norm (above 100)
	updateTopicMsg = &types.UpdateTopicRequest{
		Sender:              sender,
		TopicId:             topicId,
		Metadata:            "metadata",
		LossMethod:          "mse",
		AlphaRegret:         alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha: alloraMath.MustNewDecFromString("0.1"),
		PNorm:               alloraMath.MustNewDecFromString("3.0"),
		CNorm:               alloraMath.MustNewDecFromString("101"),
	}
	_, err = msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.ErrorContains(err, "c_norm")
}

func (s *MsgServerTestSuite) TestUpdateTopicMeritSortitionBlockedWhenWorkerWindowOpen() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	senderAddr := s.Addrs(0)
	sender := s.AddrsStr(0)

	s.WithBlockHeight(10)
	s.MintTokensToAddress(senderAddr, types.DefaultParams().CreateTopicFee)
	createTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  sender,
		Metadata:                 "Original metadata",
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
		CNorm:                    alloraMath.MustNewDecFromString("0.75"),
	}
	createResult, err := msgServer.CreateNewTopic(ctx, createTopicMsg)
	require.NoError(err)
	topicId := createResult.TopicId
	require.NoError(s.EmissionsKeeper().ActivateTopic(ctx, topicId))

	// Add a nonce whose window covers the current block.
	newerNonce := &types.Nonce{BlockHeight: 5}
	require.NoError(s.EmissionsKeeper().AddWorkerNonce(ctx, topicId, newerNonce))

	// Ensure current block is within the nonce window
	s.WithBlockHeight(6)
	ctx = s.Ctx()

	updateTopicMsg := &types.UpdateTopicRequest{
		Sender:              sender,
		TopicId:             topicId,
		Metadata:            "Original metadata",
		LossMethod:          "mse",
		AlphaRegret:         alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha: alloraMath.MustNewDecFromString("0.3"),
		PNorm:               alloraMath.MustNewDecFromString("3.0"),
		CNorm:               alloraMath.MustNewDecFromString("0.75"),
	}
	_, err = msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.ErrorIs(err, types.ErrWorkerNonceWindowNotAvailable)

	// Verify no changes were applied
	current, err := s.EmissionsKeeper().GetTopic(ctx, topicId)
	require.NoError(err)
	require.Equal(alloraMath.MustNewDecFromString("0.1"), current.MeritSortitionAlpha)
}

func (s *MsgServerTestSuite) TestUpdateTopicMeritSortitionInactiveIgnoresWindow() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	senderAddr := s.Addrs(0)
	sender := s.AddrsStr(0)

	// Topic inactive by default
	s.MintTokensToAddress(senderAddr, types.DefaultParams().CreateTopicFee)
	createTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  sender,
		Metadata:                 "Original metadata",
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
		CNorm:                    alloraMath.MustNewDecFromString("0.75"),
	}
	createResult, err := msgServer.CreateNewTopic(ctx, createTopicMsg)
	require.NoError(err)
	topicId := createResult.TopicId

	// Add a nonce whose window covers current block
	nonce := &types.Nonce{BlockHeight: 5}
	require.NoError(s.EmissionsKeeper().AddWorkerNonce(ctx, topicId, nonce))
	s.WithBlockHeight(6)
	ctx = s.Ctx()

	updateTopicMsg := &types.UpdateTopicRequest{
		Sender:              sender,
		TopicId:             topicId,
		Metadata:            "Original metadata",
		LossMethod:          "mse",
		AlphaRegret:         alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha: alloraMath.MustNewDecFromString("0.3"),
		PNorm:               alloraMath.MustNewDecFromString("3.0"),
		CNorm:               alloraMath.MustNewDecFromString("0.75"),
	}
	_, err = msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.NoError(err)

	current, err := s.EmissionsKeeper().GetTopic(ctx, topicId)
	require.NoError(err)
	require.Equal(alloraMath.MustNewDecFromString("0.3"), current.MeritSortitionAlpha)
}

func (s *MsgServerTestSuite) TestUpdateTopicUnityFields() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	sender := s.AddrsStr(0)
	senderAddr := s.Addrs(0)

	s.MintTokensToAddress(senderAddr, types.DefaultParams().CreateTopicFee)
	create := s.MockTopicMsg()
	create.Creator = sender
	create.RequireUnity = false
	create.OutputArity = types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI
	create.UnityTolerance = alloraMath.MustNewDecFromString("0.1")
	res, err := msgServer.CreateNewTopic(ctx, create)
	require.NoError(err)
	topicId := res.TopicId

	// valid update
	upd := &types.UpdateTopicRequest{
		Sender: sender, TopicId: topicId,
		Metadata: "x", LossMethod: "mse",
		AlphaRegret:         alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha: alloraMath.MustNewDecFromString("0.1"),
		PNorm:               alloraMath.MustNewDecFromString("3.0"),
		CNorm:               alloraMath.MustNewDecFromString("0.75"),

		RequireUnity:   true,
		UnityTolerance: alloraMath.MustNewDecFromString("0.01"),
	}
	_, err = msgServer.UpdateTopic(ctx, upd)
	require.NoError(err)

	topic, err := s.EmissionsKeeper().GetTopic(ctx, topicId)
	require.NoError(err)
	require.True(topic.RequireUnity)
	require.Equal(alloraMath.MustNewDecFromString("0.01"), topic.UnityTolerance)
}
