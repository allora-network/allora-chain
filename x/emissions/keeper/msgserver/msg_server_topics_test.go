package msgserver_test

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
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
				err := s.WhitelistsKeeper().RemoveFromTopicCreatorWhitelist(ctx, senderAddr.String())
				s.Require().NoError(err)
				err = s.WhitelistsKeeper().RemoveFromGlobalWhitelist(ctx, senderAddr.String())
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
				err := s.WhitelistsKeeper().AddToTopicCreatorWhitelist(ctx, senderAddr.String())
				s.Require().NoError(err)

				return s.MockTopicMsg()
			},
			postCheck: func(topicId uint64) {
				// Check topic is not in active topics yet
				activeTopics, err := s.TopicKeeper().GetActiveTopicIdsAtBlock(ctx, 10800)
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
				enabled, err := s.WhitelistsKeeper().IsTopicWorkerWhitelistEnabled(ctx, topicId)
				s.Require().NoError(err)
				s.Require().True(enabled, "Topic worker whitelist should be enabled")

				// Check reputer whitelist is enabled
				enabled, err = s.WhitelistsKeeper().IsTopicReputerWhitelistEnabled(ctx, topicId)
				s.Require().NoError(err)
				s.Require().True(enabled, "Topic reputer whitelist should be enabled")
			},
			expectSuccess: true,
		},
		{
			name: "Fails with epsilon zero",
			setup: func() *types.CreateNewTopicRequest {
				// Add to whitelist to avoid that error
				err := s.WhitelistsKeeper().AddToTopicCreatorWhitelist(ctx, senderAddr.String())
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
				err := s.WhitelistsKeeper().AddToTopicCreatorWhitelist(ctx, senderAddr.String())
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
				err := s.WhitelistsKeeper().AddToTopicCreatorWhitelist(ctx, senderAddr.String())
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
				err := s.WhitelistsKeeper().AddToTopicCreatorWhitelist(ctx, senderAddr.String())
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
				err := s.WhitelistsKeeper().AddToTopicCreatorWhitelist(ctx, senderAddr.String())
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
				err := s.WhitelistsKeeper().AddToTopicCreatorWhitelist(ctx, senderAddr.String())
				s.Require().NoError(err)

				msg := s.MockTopicMsg()
				msg.CNorm = alloraMath.MustNewDecFromString("0.75")
				return msg
			},
			postCheck: func(topicId uint64) {
				topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
				s.Require().NoError(err)
				s.Require().Equal(alloraMath.MustNewDecFromString("0.75"), topic.CNorm)
			},
			expectSuccess: true,
		},
		{
			name: "Fails with zero max labels per submission",
			setup: func() *types.CreateNewTopicRequest {
				err := s.WhitelistsKeeper().AddToTopicCreatorWhitelist(ctx, senderAddr.String())
				s.Require().NoError(err)

				msg := s.MockTopicMsg()
				msg.MaxLabelsPerSubmission = 0
				return msg
			},
			expectedError: "max labels per submission",
			expectSuccess: false,
		},
		{
			name: "Fails with max labels per submission above max",
			setup: func() *types.CreateNewTopicRequest {
				err := s.WhitelistsKeeper().AddToTopicCreatorWhitelist(ctx, senderAddr.String())
				s.Require().NoError(err)

				msg := s.MockTopicMsg()
				msg.MaxLabelsPerSubmission = types.MaxMaxLabelsPerSubmission + 1
				return msg
			},
			expectedError: "max labels per submission",
			expectSuccess: false,
		},
		{
			name: "Fails when require_unity true with SINGLE output_arity",
			setup: func() *types.CreateNewTopicRequest {
				_ = s.WhitelistsKeeper().AddToTopicCreatorWhitelist(ctx, senderAddr.String())
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
			name: "Fails when require_unity true and unity_tolerance < 0",
			setup: func() *types.CreateNewTopicRequest {
				_ = s.WhitelistsKeeper().AddToTopicCreatorWhitelist(ctx, senderAddr.String())
				msg := s.MockTopicMsg()
				msg.OutputArity = types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI
				msg.RequireUnity = true
				msg.UnityTolerance = alloraMath.MustNewDecFromString("-0.1")
				return msg
			},
			expectedError: "unity_tolerance must be in",
			expectSuccess: false,
		},
		{
			name: "Fails when require_unity true and unity_tolerance > max",
			setup: func() *types.CreateNewTopicRequest {
				_ = s.WhitelistsKeeper().AddToTopicCreatorWhitelist(ctx, senderAddr.String())
				msg := s.MockTopicMsg()
				msg.OutputArity = types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI
				msg.RequireUnity = true
				msg.UnityTolerance, _ = alloraMath.MustNewDecFromString("0.01").Add(alloraMath.MustNewDecFromString("0.0000000001"))
				return msg
			},
			expectedError: "unity_tolerance must be in",
			expectSuccess: false,
		},
		{
			name: "Fails when require_unity true and label_default_value nonzero",
			setup: func() *types.CreateNewTopicRequest {
				_ = s.WhitelistsKeeper().AddToTopicCreatorWhitelist(ctx, senderAddr.String())
				msg := s.MockTopicMsg()
				msg.OutputArity = types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI
				msg.RequireUnity = true
				msg.UnityTolerance = alloraMath.MustNewDecFromString("0.01")
				msg.LabelDefaultValue = alloraMath.OneDec()
				return msg
			},
			expectedError: "label_default_value must be zero when require_unity is true",
			expectSuccess: false,
		},
		{
			name: "Success when require_unity true and unity_tolerance valid",
			setup: func() *types.CreateNewTopicRequest {
				_ = s.WhitelistsKeeper().AddToTopicCreatorWhitelist(ctx, senderAddr.String())
				msg := s.MockTopicMsg()
				msg.OutputArity = types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI
				msg.RequireUnity = true
				msg.UnityTolerance = alloraMath.MustNewDecFromString("0.01")
				return msg
			},
			postCheck: func(topicId uint64) {
				topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
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
	topicId := uint64(1)
	inferenceTs := int64(20)

	err := s.TopicKeeper().UpdateTopicEpochLastEnded(ctx, topicId, inferenceTs)
	s.Require().NoError(err, "UpdateTopicEpochLastEnded should not return an error")

	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
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
		TopicType:                types.TopicType_TOPIC_TYPE_REGRESSION,
		OutputArity:              types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
		RequireUnity:             false,
		UnityTolerance:           alloraMath.Dec{},
		MaxLabelsPerSubmission:   types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:           nil,
		LabelDefaultValue:        alloraMath.ZeroDec(),
	}

	createResult, err := msgServer.CreateNewTopic(ctx, createTopicMsg)
	require.NoError(err)
	require.NotNil(createResult)
	topicId := createResult.TopicId

	originalTopic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	require.NoError(err)

	// Update topic with new values
	updateTopicMsg := &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		Metadata:               "Updated metadata",
		LossMethod:             "mae",
		AlphaRegret:            alloraMath.NewDecFromInt64(1),
		MeritSortitionAlpha:    alloraMath.MustNewDecFromString("0.1"),
		PNorm:                  alloraMath.NewDecFromInt64(3),
		CNorm:                  alloraMath.MustNewDecFromString("0.75"),
		RequireUnity:           false,
		UnityTolerance:         alloraMath.Dec{},
		MaxLabelsPerSubmission: types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:         nil,
		LabelDefaultValue:      alloraMath.ZeroDec(),
	}

	updateResult, err := msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.NoError(err)
	require.NotNil(updateResult)

	// Verify topic updated
	updatedTopic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	require.NoError(err)
	require.Equal("Updated metadata", updatedTopic.Metadata)
	require.Equal("mae", updatedTopic.LossMethod)

	require.Equal(originalTopic.OutputArity, updatedTopic.OutputArity)
	require.Equal(originalTopic.RequireUnity, updatedTopic.RequireUnity)
	require.Equal(originalTopic.UnityTolerance, updatedTopic.UnityTolerance)
}

func (s *MsgServerTestSuite) TestUpdateTopicRejectsMaxLabelsPerSubmissionOutOfRange() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	testCases := []struct {
		name string
		cap  uint64
	}{
		{
			name: "zero",
			cap:  0,
		},
		{
			name: "above max",
			cap:  types.MaxMaxLabelsPerSubmission + 1,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			senderAddr := s.Addrs(0)
			sender := s.AddrsStr(0)

			s.MintTokensToAddress(senderAddr, types.DefaultParams().CreateTopicFee)
			createTopicMsg := s.MockTopicMsg()

			createResult, err := msgServer.CreateNewTopic(ctx, createTopicMsg)
			require.NoError(err)
			require.NotNil(createResult)

			originalTopic, err := s.TopicKeeper().GetTopic(ctx, createResult.TopicId)
			require.NoError(err)

			updateTopicMsg := &types.UpdateTopicRequest{
				Sender:                 sender,
				TopicId:                createResult.TopicId,
				Metadata:               originalTopic.Metadata,
				LossMethod:             originalTopic.LossMethod,
				AlphaRegret:            originalTopic.AlphaRegret,
				MeritSortitionAlpha:    originalTopic.MeritSortitionAlpha,
				PNorm:                  originalTopic.PNorm,
				CNorm:                  originalTopic.CNorm,
				RequireUnity:           originalTopic.RequireUnity,
				UnityTolerance:         originalTopic.UnityTolerance,
				MaxLabelsPerSubmission: tc.cap,
				LabelWhitelist:         originalTopic.LabelWhitelist,
				LabelDefaultValue:      originalTopic.LabelDefaultValue,
			}

			updateResult, err := msgServer.UpdateTopic(ctx, updateTopicMsg)
			require.Error(err)
			require.Nil(updateResult)
			require.ErrorContains(err, "max labels per submission")

			got, err := s.TopicKeeper().GetTopic(ctx, createResult.TopicId)
			require.NoError(err)
			require.Equal(originalTopic.MaxLabelsPerSubmission, got.MaxLabelsPerSubmission)
		})
	}
}

func (s *MsgServerTestSuite) TestUpdateTopicRejectsNonzeroLabelDefaultValueWhenRequireUnity() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	senderAddr := s.Addrs(0)
	sender := s.AddrsStr(0)

	s.MintTokensToAddress(senderAddr, types.DefaultParams().CreateTopicFee)
	createTopicMsg := s.MockTopicMsg()
	createTopicMsg.OutputArity = types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI
	createTopicMsg.RequireUnity = true
	createTopicMsg.UnityTolerance = alloraMath.MustNewDecFromString("0.01")
	createTopicMsg.LabelDefaultValue = alloraMath.ZeroDec()

	createResult, err := msgServer.CreateNewTopic(ctx, createTopicMsg)
	require.NoError(err)
	require.NotNil(createResult)

	updateTopicMsg := &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                createResult.TopicId,
		Metadata:               "Updated metadata",
		LossMethod:             "mae",
		AlphaRegret:            alloraMath.NewDecFromInt64(1),
		MeritSortitionAlpha:    alloraMath.MustNewDecFromString("0.1"),
		PNorm:                  alloraMath.NewDecFromInt64(3),
		CNorm:                  alloraMath.MustNewDecFromString("0.75"),
		RequireUnity:           false,
		UnityTolerance:         alloraMath.Dec{},
		MaxLabelsPerSubmission: types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:         nil,
		LabelDefaultValue:      alloraMath.OneDec(),
	}

	updateResult, err := msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.Error(err)
	require.Nil(updateResult)
	require.ErrorContains(err, "label_default_value must be zero when require_unity is true")
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
		TopicType:                types.TopicType_TOPIC_TYPE_REGRESSION,
		OutputArity:              types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
		RequireUnity:             false,
		UnityTolerance:           alloraMath.Dec{},
		MaxLabelsPerSubmission:   types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:           nil,
		LabelDefaultValue:        alloraMath.ZeroDec(),
	}

	createResult, err := msgServer.CreateNewTopic(ctx, createTopicMsg)
	require.NoError(err)
	topicId := createResult.TopicId

	// Try to update topic with different user
	updateTopicMsg := &types.UpdateTopicRequest{
		Sender:                 otherUser,
		TopicId:                topicId,
		Metadata:               "Updated metadata",
		LossMethod:             "mse",
		AlphaRegret:            alloraMath.NewDecFromInt64(1),
		MeritSortitionAlpha:    alloraMath.MustNewDecFromString("0.1"),
		PNorm:                  alloraMath.NewDecFromInt64(3),
		CNorm:                  alloraMath.MustNewDecFromString("0.75"),
		RequireUnity:           false,
		UnityTolerance:         alloraMath.Dec{},
		MaxLabelsPerSubmission: types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:         nil,
		LabelDefaultValue:      alloraMath.ZeroDec(),
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
		Sender:                 sender,
		TopicId:                nonexistentTopicId,
		Metadata:               "Updated metadata",
		LossMethod:             "mse",
		AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha:    alloraMath.MustNewDecFromString("0.1"),
		PNorm:                  alloraMath.MustNewDecFromString("3.0"),
		CNorm:                  alloraMath.MustNewDecFromString("0.75"),
		RequireUnity:           false,
		UnityTolerance:         alloraMath.Dec{},
		MaxLabelsPerSubmission: types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:         nil,
		LabelDefaultValue:      alloraMath.ZeroDec(),
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
		Sender:                 sender,
		TopicId:                topicId,
		Metadata:               "valid metadata",
		LossMethod:             "",
		AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha:    alloraMath.MustNewDecFromString("0.1"),
		PNorm:                  alloraMath.MustNewDecFromString("3.0"),
		CNorm:                  alloraMath.MustNewDecFromString("0.75"),
		RequireUnity:           false,
		UnityTolerance:         alloraMath.Dec{},
		MaxLabelsPerSubmission: types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:         nil,
		LabelDefaultValue:      alloraMath.ZeroDec(),
	}
	updateResult, err := msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.Error(err)
	require.Nil(updateResult)
	require.ErrorContains(err, "loss method invalid")

	// Test too long loss method
	updateTopicMsg = &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		Metadata:               "valid metadata",
		LossMethod:             strings.Repeat("a", 257),
		AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha:    alloraMath.MustNewDecFromString("0.1"),
		PNorm:                  alloraMath.MustNewDecFromString("3.0"),
		CNorm:                  alloraMath.MustNewDecFromString("0.75"),
		RequireUnity:           false,
		UnityTolerance:         alloraMath.Dec{},
		MaxLabelsPerSubmission: types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:         nil,
		LabelDefaultValue:      alloraMath.ZeroDec(),
	}
	updateResult, err = msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.Error(err)
	require.Nil(updateResult)
	require.ErrorContains(err, "loss method invalid")

	// Test too long metadata
	updateTopicMsg = &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		Metadata:               strings.Repeat("a", 257),
		LossMethod:             "mse",
		AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha:    alloraMath.MustNewDecFromString("0.1"),
		PNorm:                  alloraMath.MustNewDecFromString("3.0"),
		CNorm:                  alloraMath.MustNewDecFromString("0.75"),
		RequireUnity:           false,
		UnityTolerance:         alloraMath.Dec{},
		MaxLabelsPerSubmission: types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:         nil,
		LabelDefaultValue:      alloraMath.ZeroDec(),
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
		TopicType:                types.TopicType_TOPIC_TYPE_REGRESSION,
		OutputArity:              types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
		RequireUnity:             false,
		UnityTolerance:           alloraMath.Dec{},
		MaxLabelsPerSubmission:   types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:           nil,
		LabelDefaultValue:        alloraMath.ZeroDec(),
	}

	createResult, err := msgServer.CreateNewTopic(ctx, createTopicMsg)
	require.NoError(err)
	topicId := createResult.TopicId

	// Get original topic to verify initial state
	originalTopic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	require.NoError(err)
	require.Equal("original metadata", originalTopic.Metadata)
	require.Equal("mse", originalTopic.LossMethod)

	// Test successful update
	updateTopicMsg := &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		Metadata:               "updated metadata",
		LossMethod:             "mae",
		AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha:    alloraMath.MustNewDecFromString("0.1"),
		PNorm:                  alloraMath.MustNewDecFromString("3.0"),
		CNorm:                  alloraMath.MustNewDecFromString("0.75"),
		RequireUnity:           false,
		UnityTolerance:         alloraMath.Dec{},
		MaxLabelsPerSubmission: types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:         nil,
		LabelDefaultValue:      alloraMath.ZeroDec(),
	}

	updateResult, err := msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.NoError(err)
	require.NotNil(updateResult)

	// Verify topic updated and only allowed fields changed
	updatedTopic, err := s.TopicKeeper().GetTopic(ctx, topicId)
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
		TopicType:                types.TopicType_TOPIC_TYPE_REGRESSION,
		OutputArity:              types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
		RequireUnity:             false,
		UnityTolerance:           alloraMath.Dec{},
		MaxLabelsPerSubmission:   types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:           nil,
		LabelDefaultValue:        alloraMath.ZeroDec(),
	}
	createResult, err := msgServer.CreateNewTopic(ctx, createTopicMsg)
	require.NoError(err)
	topicId := createResult.TopicId

	updateTopicMsg := &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		Metadata:               "Original metadata",
		LossMethod:             "mse",
		AlphaRegret:            alloraMath.MustNewDecFromString("0.25"),
		MeritSortitionAlpha:    alloraMath.MustNewDecFromString("0.3"),
		PNorm:                  alloraMath.MustNewDecFromString("3.5"),
		CNorm:                  alloraMath.MustNewDecFromString("0.75"),
		RequireUnity:           false,
		UnityTolerance:         alloraMath.Dec{},
		MaxLabelsPerSubmission: types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:         nil,
		LabelDefaultValue:      alloraMath.ZeroDec(),
	}

	_, err = msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.NoError(err)

	updatedTopic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	require.NoError(err)
	require.Equal(alloraMath.MustNewDecFromString("0.25"), updatedTopic.AlphaRegret)
	require.Equal(alloraMath.MustNewDecFromString("0.3"), updatedTopic.MeritSortitionAlpha)
	require.Equal(alloraMath.MustNewDecFromString("3.5"), updatedTopic.PNorm)

	// Test updating CNorm with valid values
	updateTopicMsg = &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		Metadata:               "Original metadata",
		LossMethod:             "mse",
		AlphaRegret:            alloraMath.MustNewDecFromString("0.25"),
		MeritSortitionAlpha:    alloraMath.MustNewDecFromString("0.3"),
		PNorm:                  alloraMath.MustNewDecFromString("3.5"),
		CNorm:                  alloraMath.MustNewDecFromString("50.5"),
		RequireUnity:           false,
		UnityTolerance:         alloraMath.Dec{},
		MaxLabelsPerSubmission: types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:         nil,
		LabelDefaultValue:      alloraMath.ZeroDec(),
	}
	_, err = msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.NoError(err)

	updatedTopic, err = s.TopicKeeper().GetTopic(ctx, topicId)
	require.NoError(err)
	require.Equal(alloraMath.MustNewDecFromString("50.5"), updatedTopic.CNorm)

	// Test updating CNorm to boundary value -100
	updateTopicMsg = &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		Metadata:               "Original metadata",
		LossMethod:             "mse",
		AlphaRegret:            alloraMath.MustNewDecFromString("0.25"),
		MeritSortitionAlpha:    alloraMath.MustNewDecFromString("0.3"),
		PNorm:                  alloraMath.MustNewDecFromString("3.5"),
		CNorm:                  alloraMath.MustNewDecFromString("-100"),
		RequireUnity:           false,
		UnityTolerance:         alloraMath.Dec{},
		MaxLabelsPerSubmission: types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:         nil,
		LabelDefaultValue:      alloraMath.ZeroDec(),
	}
	_, err = msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.NoError(err)

	updatedTopic, err = s.TopicKeeper().GetTopic(ctx, topicId)
	require.NoError(err)
	require.Equal(alloraMath.MustNewDecFromString("-100"), updatedTopic.CNorm)

	// Test updating CNorm to boundary value 100
	updateTopicMsg = &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		Metadata:               "Original metadata",
		LossMethod:             "mse",
		AlphaRegret:            alloraMath.MustNewDecFromString("0.25"),
		MeritSortitionAlpha:    alloraMath.MustNewDecFromString("0.3"),
		PNorm:                  alloraMath.MustNewDecFromString("3.5"),
		CNorm:                  alloraMath.MustNewDecFromString("100"),
		RequireUnity:           false,
		UnityTolerance:         alloraMath.Dec{},
		MaxLabelsPerSubmission: types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:         nil,
		LabelDefaultValue:      alloraMath.ZeroDec(),
	}
	_, err = msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.NoError(err)

	updatedTopic, err = s.TopicKeeper().GetTopic(ctx, topicId)
	require.NoError(err)
	require.Equal(alloraMath.MustNewDecFromString("100"), updatedTopic.CNorm)

	// Add a fulfilled nonce (window closed) and ensure updates still allowed
	s.Require().NoError(s.NonceKeeper().AddWorkerNonce(ctx, topicId, &types.Nonce{BlockHeight: 1}))
	// Close window by moving block height beyond worker submission window
	s.WithBlockHeight(50)
	ctx = s.Ctx()
	updateTopicMsg = &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		Metadata:               "Original metadata",
		LossMethod:             "mse",
		AlphaRegret:            alloraMath.MustNewDecFromString("0.25"),
		MeritSortitionAlpha:    alloraMath.MustNewDecFromString("0.4"),
		PNorm:                  alloraMath.MustNewDecFromString("3.5"),
		CNorm:                  alloraMath.MustNewDecFromString("100"),
		RequireUnity:           false,
		UnityTolerance:         alloraMath.Dec{},
		MaxLabelsPerSubmission: types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:         nil,
		LabelDefaultValue:      alloraMath.ZeroDec(),
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
		Sender:                 sender,
		TopicId:                topicId,
		Metadata:               "metadata",
		LossMethod:             "mse",
		AlphaRegret:            alloraMath.ZeroDec(),
		MeritSortitionAlpha:    alloraMath.MustNewDecFromString("0.1"),
		PNorm:                  alloraMath.MustNewDecFromString("3.0"),
		CNorm:                  alloraMath.MustNewDecFromString("0.75"),
		RequireUnity:           false,
		UnityTolerance:         alloraMath.Dec{},
		MaxLabelsPerSubmission: types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:         nil,
		LabelDefaultValue:      alloraMath.ZeroDec(),
	}
	_, err := msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.ErrorContains(err, "alpha regret")

	// Invalid merit_sortition_alpha (>1)
	updateTopicMsg = &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		Metadata:               "metadata",
		LossMethod:             "mse",
		AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha:    alloraMath.MustNewDecFromString("1.1"),
		PNorm:                  alloraMath.MustNewDecFromString("3.0"),
		CNorm:                  alloraMath.MustNewDecFromString("0.75"),
		RequireUnity:           false,
		UnityTolerance:         alloraMath.Dec{},
		MaxLabelsPerSubmission: types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:         nil,
		LabelDefaultValue:      alloraMath.ZeroDec(),
	}
	_, err = msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.ErrorContains(err, "merit sortition alpha")

	// Invalid p_norm (below range)
	updateTopicMsg = &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		Metadata:               "metadata",
		LossMethod:             "mse",
		AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha:    alloraMath.MustNewDecFromString("0.1"),
		PNorm:                  alloraMath.MustNewDecFromString("2.0"),
		CNorm:                  alloraMath.MustNewDecFromString("0.75"),
		RequireUnity:           false,
		UnityTolerance:         alloraMath.Dec{},
		MaxLabelsPerSubmission: types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:         nil,
		LabelDefaultValue:      alloraMath.ZeroDec(),
	}
	_, err = msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.ErrorContains(err, "p-norm")

	// Invalid c_norm (below -100)
	updateTopicMsg = &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		Metadata:               "metadata",
		LossMethod:             "mse",
		AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha:    alloraMath.MustNewDecFromString("0.1"),
		PNorm:                  alloraMath.MustNewDecFromString("3.0"),
		CNorm:                  alloraMath.MustNewDecFromString("-101"),
		RequireUnity:           false,
		UnityTolerance:         alloraMath.Dec{},
		MaxLabelsPerSubmission: types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:         nil,
		LabelDefaultValue:      alloraMath.ZeroDec(),
	}
	_, err = msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.ErrorContains(err, "c_norm")

	// Invalid c_norm (above 100)
	updateTopicMsg = &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		Metadata:               "metadata",
		LossMethod:             "mse",
		AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha:    alloraMath.MustNewDecFromString("0.1"),
		PNorm:                  alloraMath.MustNewDecFromString("3.0"),
		CNorm:                  alloraMath.MustNewDecFromString("101"),
		RequireUnity:           false,
		UnityTolerance:         alloraMath.Dec{},
		MaxLabelsPerSubmission: types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:         nil,
		LabelDefaultValue:      alloraMath.ZeroDec(),
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
		TopicType:                types.TopicType_TOPIC_TYPE_REGRESSION,
		OutputArity:              types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
		RequireUnity:             false,
		UnityTolerance:           alloraMath.Dec{},
		MaxLabelsPerSubmission:   types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:           nil,
		LabelDefaultValue:        alloraMath.ZeroDec(),
	}
	createResult, err := msgServer.CreateNewTopic(ctx, createTopicMsg)
	require.NoError(err)
	topicId := createResult.TopicId
	require.NoError(s.TopicKeeper().ActivateTopic(ctx, topicId))

	// Add a nonce whose window covers the current block.
	newerNonce := &types.Nonce{BlockHeight: 5}
	require.NoError(s.NonceKeeper().AddWorkerNonce(ctx, topicId, newerNonce))

	// Ensure current block is within the nonce window
	s.WithBlockHeight(6)
	ctx = s.Ctx()

	updateTopicMsg := &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		Metadata:               "Original metadata",
		LossMethod:             "mse",
		AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha:    alloraMath.MustNewDecFromString("0.3"),
		PNorm:                  alloraMath.MustNewDecFromString("3.0"),
		CNorm:                  alloraMath.MustNewDecFromString("0.75"),
		RequireUnity:           false,
		UnityTolerance:         alloraMath.Dec{},
		MaxLabelsPerSubmission: types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:         nil,
		LabelDefaultValue:      alloraMath.ZeroDec(),
	}
	_, err = msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.ErrorIs(err, types.ErrWorkerNonceWindowNotAvailable)

	// Verify no changes were applied
	current, err := s.TopicKeeper().GetTopic(ctx, topicId)
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
		TopicType:                types.TopicType_TOPIC_TYPE_REGRESSION,
		OutputArity:              types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
		RequireUnity:             false,
		UnityTolerance:           alloraMath.Dec{},
		MaxLabelsPerSubmission:   types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:           nil,
		LabelDefaultValue:        alloraMath.ZeroDec(),
	}
	createResult, err := msgServer.CreateNewTopic(ctx, createTopicMsg)
	require.NoError(err)
	topicId := createResult.TopicId

	// Add a nonce whose window covers current block
	nonce := &types.Nonce{BlockHeight: 5}
	require.NoError(s.NonceKeeper().AddWorkerNonce(ctx, topicId, nonce))
	s.WithBlockHeight(6)
	ctx = s.Ctx()

	updateTopicMsg := &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		Metadata:               "Original metadata",
		LossMethod:             "mse",
		AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha:    alloraMath.MustNewDecFromString("0.3"),
		PNorm:                  alloraMath.MustNewDecFromString("3.0"),
		CNorm:                  alloraMath.MustNewDecFromString("0.75"),
		RequireUnity:           false,
		UnityTolerance:         alloraMath.Dec{},
		MaxLabelsPerSubmission: types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:         nil,
		LabelDefaultValue:      alloraMath.ZeroDec(),
	}
	_, err = msgServer.UpdateTopic(ctx, updateTopicMsg)
	require.NoError(err)

	current, err := s.TopicKeeper().GetTopic(ctx, topicId)
	require.NoError(err)
	require.Equal(alloraMath.MustNewDecFromString("0.3"), current.MeritSortitionAlpha)
}

// createTopicForWSWTests builds a topic that is active with a worker
// submission window currently covering the block height set by the caller.
// Returns the topic id.
//
//nolint:exhaustruct
func (s *MsgServerTestSuite) createTopicForWSWTests(sender string) uint64 {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()
	senderAddr, err := sdk.AccAddressFromBech32(sender)
	require.NoError(err)
	s.MintTokensToAddress(senderAddr, types.DefaultParams().CreateTopicFee)
	create := &types.CreateNewTopicRequest{
		Creator:                  sender,
		Metadata:                 "wsw test",
		LossMethod:               "mse",
		EpochLength:              100,
		GroundTruthLag:           100,
		WorkerSubmissionWindow:   10,
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		PNorm:                    alloraMath.MustNewDecFromString("3.0"),
		AllowNegative:            false,
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		EnableWorkerWhitelist:    false,
		EnableReputerWhitelist:   false,
		CNorm:                    alloraMath.MustNewDecFromString("0.75"),
		TopicType:                types.TopicType_TOPIC_TYPE_REGRESSION,
		OutputArity:              types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
		RequireUnity:             false,
		UnityTolerance:           alloraMath.Dec{},
		MaxLabelsPerSubmission:   4,
		LabelWhitelist:           []string{"a", "b", "c"},
		LabelDefaultValue:        alloraMath.ZeroDec(),
	}
	resp, err := msgServer.CreateNewTopic(ctx, create)
	require.NoError(err)
	return resp.TopicId
}

// TestSetTopicCanonicalizesLabelWhitelist asserts that
// CreateNewTopic canonicalizes the whitelist in place: trimmed, NFC-normalized,
// and deduplicated.
//
//nolint:exhaustruct
func (s *MsgServerTestSuite) TestSetTopicCanonicalizesLabelWhitelist() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	sender := s.AddrsStr(0)
	s.MintTokensToAddress(s.Addrs(0), types.DefaultParams().CreateTopicFee)
	create := &types.CreateNewTopicRequest{
		Creator:                  sender,
		Metadata:                 "canon whitelist",
		LossMethod:               "mse",
		EpochLength:              100,
		GroundTruthLag:           100,
		WorkerSubmissionWindow:   10,
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		PNorm:                    alloraMath.MustNewDecFromString("3.0"),
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		CNorm:                    alloraMath.MustNewDecFromString("0.75"),
		TopicType:                types.TopicType_TOPIC_TYPE_REGRESSION,
		OutputArity:              types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
		RequireUnity:             false,
		UnityTolerance:           alloraMath.Dec{},
		MaxLabelsPerSubmission:   types.DefaultMaxLabelsPerSubmission,
		// NFD form (e + combining acute), plus whitespace padding; a
		// canonical duplicate should be rejected with ErrInvalidLabelName.
		LabelWhitelist:    []string{"  foo  ", "e\u0301", "\u00e9"},
		LabelDefaultValue: alloraMath.ZeroDec(),
	}
	_, err := msgServer.CreateNewTopic(ctx, create)
	require.ErrorIs(err, types.ErrInvalidLabelName, "canonical duplicate must be rejected at SetTopic")

	create.LabelWhitelist = []string{"  foo  ", "e\u0301"}
	resp, err := msgServer.CreateNewTopic(ctx, create)
	require.NoError(err)
	stored, err := s.TopicKeeper().GetTopic(ctx, resp.TopicId)
	require.NoError(err)
	require.Equal([]string{"foo", "\u00e9"}, stored.LabelWhitelist,
		"whitelist must be persisted in canonical (NFC+trimmed) form")
}

// TestUpdateTopicMaxLabelsBlockedWhenWorkerWindowOpen asserts the generalized
// WSW lock now also guards max_labels_per_submission mutations.
//
//nolint:exhaustruct
func (s *MsgServerTestSuite) TestUpdateTopicMaxLabelsBlockedWhenWorkerWindowOpen() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	sender := s.AddrsStr(0)
	s.WithBlockHeight(10)
	topicId := s.createTopicForWSWTests(sender)
	require.NoError(s.TopicKeeper().ActivateTopic(ctx, topicId))

	require.NoError(s.NonceKeeper().AddWorkerNonce(ctx, topicId, &types.Nonce{BlockHeight: 5}))
	s.WithBlockHeight(6)
	ctx = s.Ctx()

	msg := &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		Metadata:               "wsw test",
		LossMethod:             "mse",
		AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha:    alloraMath.MustNewDecFromString("0.1"),
		PNorm:                  alloraMath.MustNewDecFromString("3.0"),
		CNorm:                  alloraMath.MustNewDecFromString("0.75"),
		RequireUnity:           false,
		UnityTolerance:         alloraMath.Dec{},
		MaxLabelsPerSubmission: 8, // changed -> must be rejected
		LabelWhitelist:         []string{"a", "b", "c"},
		LabelDefaultValue:      alloraMath.ZeroDec(),
	}
	_, err := msgServer.UpdateTopic(ctx, msg)
	require.ErrorIs(err, types.ErrWorkerNonceWindowNotAvailable)
	got, err := s.TopicKeeper().GetTopic(ctx, topicId)
	require.NoError(err)
	require.Equal(uint64(4), got.MaxLabelsPerSubmission,
		"MaxLabelsPerSubmission must not have changed while WSW was open")
}

// TestUpdateTopicWhitelistBlockedWhenWorkerWindowOpen asserts the generalized
// WSW lock now also guards label_whitelist mutations.
//
//nolint:exhaustruct
func (s *MsgServerTestSuite) TestUpdateTopicWhitelistBlockedWhenWorkerWindowOpen() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	sender := s.AddrsStr(0)
	s.WithBlockHeight(10)
	topicId := s.createTopicForWSWTests(sender)
	require.NoError(s.TopicKeeper().ActivateTopic(ctx, topicId))

	require.NoError(s.NonceKeeper().AddWorkerNonce(ctx, topicId, &types.Nonce{BlockHeight: 5}))
	s.WithBlockHeight(6)
	ctx = s.Ctx()

	msg := &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		Metadata:               "wsw test",
		LossMethod:             "mse",
		AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha:    alloraMath.MustNewDecFromString("0.1"),
		PNorm:                  alloraMath.MustNewDecFromString("3.0"),
		CNorm:                  alloraMath.MustNewDecFromString("0.75"),
		RequireUnity:           false,
		UnityTolerance:         alloraMath.Dec{},
		MaxLabelsPerSubmission: 4,
		LabelWhitelist:         []string{"a", "b"}, // changed: dropped "c"
		LabelDefaultValue:      alloraMath.ZeroDec(),
	}
	_, err := msgServer.UpdateTopic(ctx, msg)
	require.ErrorIs(err, types.ErrWorkerNonceWindowNotAvailable)
	got, err := s.TopicKeeper().GetTopic(ctx, topicId)
	require.NoError(err)
	require.Equal([]string{"a", "b", "c"}, got.LabelWhitelist,
		"LabelWhitelist must not have changed while WSW was open")
}

// TestUpdateTopicWhitelistAllowedAfterWSWClosed confirms the WSW lock is
// time-bounded: once the submission window has closed for the outstanding
// nonce, whitelist/cap mutations are accepted again (and the whitelist is
// canonicalized by SetTopic on its way in).
//
//nolint:exhaustruct
func (s *MsgServerTestSuite) TestUpdateTopicWhitelistAllowedAfterWSWClosed() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	sender := s.AddrsStr(0)
	s.WithBlockHeight(10)
	topicId := s.createTopicForWSWTests(sender)
	require.NoError(s.TopicKeeper().ActivateTopic(ctx, topicId))

	require.NoError(s.NonceKeeper().AddWorkerNonce(ctx, topicId, &types.Nonce{BlockHeight: 5}))
	// 5 + 10 = 15, so block 16 is strictly past the window.
	s.WithBlockHeight(16)
	ctx = s.Ctx()

	msg := &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		Metadata:               "wsw test",
		LossMethod:             "mse",
		AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha:    alloraMath.MustNewDecFromString("0.1"),
		PNorm:                  alloraMath.MustNewDecFromString("3.0"),
		CNorm:                  alloraMath.MustNewDecFromString("0.75"),
		RequireUnity:           false,
		UnityTolerance:         alloraMath.Dec{},
		MaxLabelsPerSubmission: 8,
		LabelWhitelist:         []string{"  a  ", "e\u0301"},
		LabelDefaultValue:      alloraMath.ZeroDec(),
	}
	_, err := msgServer.UpdateTopic(ctx, msg)
	require.NoError(err)
	got, err := s.TopicKeeper().GetTopic(ctx, topicId)
	require.NoError(err)
	require.Equal(uint64(8), got.MaxLabelsPerSubmission)
	require.Equal([]string{"a", "\u00e9"}, got.LabelWhitelist,
		"whitelist must be canonicalized on UpdateTopic once the WSW has closed")
}

// TestUpdateTopicRejectsOutOfRangeMaxLabelsPerSubmission documents that
// UpdateTopic treats max_labels_per_submission as the submitted value, not as
// "unchanged" or "use module default". Values outside the allowed cap range
// are rejected and the topic remains unchanged.
//
//nolint:exhaustruct
func (s *MsgServerTestSuite) TestUpdateTopicRejectsOutOfRangeMaxLabelsPerSubmission() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	cases := []struct {
		name string
		cap  uint64
		err  error
	}{
		{name: "zero", cap: 0, err: types.ErrValidationMustBeGreaterthanZero},
		{name: "above max", cap: types.MaxMaxLabelsPerSubmission + 1, err: types.ErrInvalidValue},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			sender := s.AddrsStr(0)
			topicId := s.createTopicForWSWTests(sender)

			msg := &types.UpdateTopicRequest{
				Sender:                 sender,
				TopicId:                topicId,
				Metadata:               "full payload clear",
				LossMethod:             "mse",
				AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
				MeritSortitionAlpha:    alloraMath.MustNewDecFromString("0.1"),
				PNorm:                  alloraMath.MustNewDecFromString("3.0"),
				CNorm:                  alloraMath.MustNewDecFromString("0.75"),
				RequireUnity:           false,
				UnityTolerance:         alloraMath.Dec{},
				MaxLabelsPerSubmission: tc.cap,
				LabelWhitelist:         nil,
			}
			_, err := msgServer.UpdateTopic(ctx, msg)
			require.ErrorIs(err, tc.err)

			got, err := s.TopicKeeper().GetTopic(ctx, topicId)
			require.NoError(err)
			require.Equal(uint64(4), got.MaxLabelsPerSubmission,
				"rejected max_labels_per_submission must not modify the topic")
			require.Equal([]string{"a", "b", "c"}, got.LabelWhitelist,
				"rejected max_labels_per_submission must not clear the whitelist")
		})
	}
}

// TestUpdateTopicFullPayloadClearsLabelWhitelist documents that UpdateTopic is
// a full-payload operation: a nil label_whitelist replaces the existing topic
// whitelist with unrestricted, rather than preserving the old whitelist.
//
//nolint:exhaustruct
func (s *MsgServerTestSuite) TestUpdateTopicFullPayloadClearsLabelWhitelist() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	sender := s.AddrsStr(0)
	topicId := s.createTopicForWSWTests(sender)

	msg := &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		Metadata:               "full payload clear",
		LossMethod:             "mse",
		AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha:    alloraMath.MustNewDecFromString("0.1"),
		PNorm:                  alloraMath.MustNewDecFromString("3.0"),
		CNorm:                  alloraMath.MustNewDecFromString("0.75"),
		RequireUnity:           false,
		UnityTolerance:         alloraMath.Dec{},
		MaxLabelsPerSubmission: 8,
		LabelWhitelist:         nil,
	}
	_, err := msgServer.UpdateTopic(ctx, msg)
	require.NoError(err)

	got, err := s.TopicKeeper().GetTopic(ctx, topicId)
	require.NoError(err)
	require.Equal(uint64(8), got.MaxLabelsPerSubmission)
	require.Empty(got.LabelWhitelist,
		"nil label_whitelist must clear the existing whitelist under full-payload UpdateTopic")
}

// TestUpdateTopicWSWLockBlocksWhenOlderNonceStillWithinWindow pins the
// generalization from the v14 "newest unfulfilled nonce only" WSW guard to
// the current "any unfulfilled nonce" guard. With WorkerSubmissionWindow=10
// and two outstanding worker nonces, the topic has two open WSWs. A param
// update must be rejected if *any* of them is currently open, regardless
// of whether the open one is the newest. At block 12 the newest nonce
// (200) is not yet inside its submission window [200, 210], while the
// older nonce (5) is still inside its window [5, 15] and covers block 12.
// A regression to the newest-only check would therefore accept the
// update; the generalized check must reject it.
//
//nolint:exhaustruct
func (s *MsgServerTestSuite) TestUpdateTopicWSWLockBlocksWhenOlderNonceStillWithinWindow() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	sender := s.AddrsStr(0)
	s.WithBlockHeight(10)
	topicId := s.createTopicForWSWTests(sender)
	require.NoError(s.TopicKeeper().ActivateTopic(ctx, topicId))

	require.NoError(s.NonceKeeper().AddWorkerNonce(ctx, topicId, &types.Nonce{BlockHeight: 5}))
	require.NoError(s.NonceKeeper().AddWorkerNonce(ctx, topicId, &types.Nonce{BlockHeight: 200}))
	s.WithBlockHeight(12)
	ctx = s.Ctx()

	msg := &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		Metadata:               "wsw test",
		LossMethod:             "mse",
		AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha:    alloraMath.MustNewDecFromString("0.1"),
		PNorm:                  alloraMath.MustNewDecFromString("3.0"),
		CNorm:                  alloraMath.MustNewDecFromString("0.75"),
		RequireUnity:           false,
		UnityTolerance:         alloraMath.Dec{},
		MaxLabelsPerSubmission: 8, // changed -> must be rejected
		LabelWhitelist:         []string{"a", "b", "c"},
		LabelDefaultValue:      alloraMath.ZeroDec(),
	}
	_, err := msgServer.UpdateTopic(ctx, msg)
	require.ErrorIs(err, types.ErrWorkerNonceWindowNotAvailable,
		"UpdateTopic must reject mutations while any unfulfilled nonce is within its WSW, not only the newest")
	got, err := s.TopicKeeper().GetTopic(ctx, topicId)
	require.NoError(err)
	require.Equal(uint64(4), got.MaxLabelsPerSubmission,
		"MaxLabelsPerSubmission must not change while an older nonce's WSW remains open")
}
