package msgserver_test

import (
	"errors"

	"cosmossdk.io/collections"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// Tests for the per-topic max_top_inferers_to_reward parameter: creation
// default-resolution and bounds, the UpdateTopic worker-submission-window guard
// and change-detection ordering, and the live-global admission clamp.

// createTopicWithCap creates a simple SINGLE-arity regression topic with an
// explicit max_top_inferers_to_reward request value and returns (topicId, err).
//
//nolint:exhaustruct
func (s *MsgServerTestSuite) createTopicWithCap(sender string, capValue uint64) (uint64, error) {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	senderAddr, err := sdk.AccAddressFromBech32(sender)
	s.Require().NoError(err)
	s.MintTokensToAddress(senderAddr, types.DefaultParams().CreateTopicFee)
	create := &types.CreateNewTopicRequest{
		Creator:                  sender,
		Metadata:                 "cap test",
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
		OutputArity:              types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
		RequireUnity:             false,
		UnityTolerance:           alloraMath.Dec{},
		MaxLabelsPerSubmission:   types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:           nil,
		LabelDefaultValue:        alloraMath.ZeroDec(),
		MaxTopInferersToReward:   capValue,
	}
	resp, err := msgServer.CreateNewTopic(ctx, create)
	if err != nil {
		return 0, err
	}
	return resp.TopicId, nil
}

// baseCapUpdateMsg builds an UpdateTopicRequest whose fields match a topic made
// by createTopicWithCap, so only the field a test overrides actually changes.
//
//nolint:exhaustruct
func (s *MsgServerTestSuite) baseCapUpdateMsg(sender string, topicId uint64) *types.UpdateTopicRequest {
	return &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		Metadata:               "cap test",
		LossMethod:             "mse",
		AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha:    alloraMath.MustNewDecFromString("0.1"),
		PNorm:                  alloraMath.MustNewDecFromString("3.0"),
		CNorm:                  alloraMath.MustNewDecFromString("0.75"),
		MaxLabelsPerSubmission: types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:         nil,
		LabelDefaultValue:      alloraMath.ZeroDec(),
	}
}

// baseWSWUpdateMsg builds an UpdateTopicRequest whose fields match the topic
// created by createTopicForWSWTests, so only the overridden field changes.
//
//nolint:exhaustruct
func (s *MsgServerTestSuite) baseWSWUpdateMsg(sender string, topicId uint64) *types.UpdateTopicRequest {
	return &types.UpdateTopicRequest{
		Sender:                 sender,
		TopicId:                topicId,
		Metadata:               "wsw test",
		LossMethod:             "mse",
		AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha:    alloraMath.MustNewDecFromString("0.1"),
		PNorm:                  alloraMath.MustNewDecFromString("3.0"),
		CNorm:                  alloraMath.MustNewDecFromString("0.75"),
		MaxLabelsPerSubmission: 4,
		LabelWhitelist:         []string{"a", "b", "c"},
		LabelDefaultValue:      alloraMath.ZeroDec(),
	}
}

// request value 0 stores the global default.
func (s *MsgServerTestSuite) TestCreateTopicMaxTopInferersDefaultsToGlobal() {
	sender := s.AddrsStr(0)
	topicId, err := s.createTopicWithCap(sender, 0)
	s.Require().NoError(err)
	got, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)
	s.Require().Equal(types.DefaultParams().MaxTopInferersToReward, got.MaxTopInferersToReward)
}

// request value 0 resolves against the live global, not a compile-time constant.
func (s *MsgServerTestSuite) TestCreateTopicMaxTopInferersDefaultReadsLiveGlobal() {
	ctx := s.Ctx()
	params, err := s.EmissionsKeeper().GetParams(ctx)
	s.Require().NoError(err)
	params.MaxTopInferersToReward = 20
	s.Require().NoError(s.EmissionsKeeper().SetParams(ctx, params))

	sender := s.AddrsStr(0)
	topicId, err := s.createTopicWithCap(sender, 0)
	s.Require().NoError(err)
	got, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(uint64(20), got.MaxTopInferersToReward)
}

// explicit values within [1, global] are stored verbatim.
func (s *MsgServerTestSuite) TestCreateTopicMaxTopInferersExplicitValues() {
	sender := s.AddrsStr(0)
	global := types.DefaultParams().MaxTopInferersToReward

	idOne, err := s.createTopicWithCap(sender, 1)
	s.Require().NoError(err)
	gotOne, err := s.TopicKeeper().GetTopic(s.Ctx(), idOne)
	s.Require().NoError(err)
	s.Require().Equal(uint64(1), gotOne.MaxTopInferersToReward)

	idMax, err := s.createTopicWithCap(sender, global)
	s.Require().NoError(err)
	gotMax, err := s.TopicKeeper().GetTopic(s.Ctx(), idMax)
	s.Require().NoError(err)
	s.Require().Equal(global, gotMax.MaxTopInferersToReward)
}

// an explicit value above the global ceiling is rejected.
func (s *MsgServerTestSuite) TestCreateTopicMaxTopInferersAboveGlobalRejected() {
	sender := s.AddrsStr(0)
	global := types.DefaultParams().MaxTopInferersToReward
	_, err := s.createTopicWithCap(sender, global+1)
	s.Require().ErrorIs(err, types.ErrTopicMaxTopInferersToRewardTooBig)
}

// with no open worker submission window, the cap can be changed.
func (s *MsgServerTestSuite) TestUpdateTopicMaxTopInferersAllowedWhenNoWindow() {
	sender := s.AddrsStr(0)
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	topicId := s.createTopicForWSWTests(sender) // created inactive, no open nonce
	msg := s.baseWSWUpdateMsg(sender, topicId)
	msg.MaxTopInferersToReward = 10
	_, err := msgServer.UpdateTopic(ctx, msg)
	s.Require().NoError(err)
	got, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(uint64(10), got.MaxTopInferersToReward)
}

// changing only the cap while a window is open is rejected and names the field.
func (s *MsgServerTestSuite) TestUpdateTopicMaxTopInferersBlockedWhenWorkerWindowOpen() {
	sender := s.AddrsStr(0)
	ctx, topicId := s.setupActiveTopicWithOpenWSW(sender)
	msgServer := s.EmissionsMsgServer()
	msg := s.baseWSWUpdateMsg(sender, topicId)
	msg.MaxTopInferersToReward = 10 // changed from the global default
	_, err := msgServer.UpdateTopic(ctx, msg)
	s.Require().ErrorIs(err, types.ErrWorkerNonceWindowNotAvailable)
	s.Require().ErrorContains(err, "max_top_inferers_to_reward")
	got, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(types.DefaultParams().MaxTopInferersToReward, got.MaxTopInferersToReward)
}

// an explicit no-op (same value) during an open window is allowed.
func (s *MsgServerTestSuite) TestUpdateTopicMaxTopInferersUnchangedDuringWindow() {
	sender := s.AddrsStr(0)
	ctx, topicId := s.setupActiveTopicWithOpenWSW(sender)
	msgServer := s.EmissionsMsgServer()
	msg := s.baseWSWUpdateMsg(sender, topicId)
	msg.MaxTopInferersToReward = types.DefaultParams().MaxTopInferersToReward // same as stored
	_, err := msgServer.UpdateTopic(ctx, msg)
	s.Require().NoError(err)
}

// sending 0 during an open window resolves to the current global (== stored),
// so it is a no-op and is allowed. This proves default-resolution runs before
// change-detection.
func (s *MsgServerTestSuite) TestUpdateTopicMaxTopInferersZeroResolvesBeforeChangeDetectionDuringWindow() {
	sender := s.AddrsStr(0)
	ctx, topicId := s.setupActiveTopicWithOpenWSW(sender)
	msgServer := s.EmissionsMsgServer()
	msg := s.baseWSWUpdateMsg(sender, topicId) // MaxTopInferersToReward left 0
	_, err := msgServer.UpdateTopic(ctx, msg)
	s.Require().NoError(err)
	got, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(types.DefaultParams().MaxTopInferersToReward, got.MaxTopInferersToReward)
}

// with no open window, sending 0 resets the cap to the global default.
func (s *MsgServerTestSuite) TestUpdateTopicMaxTopInferersZeroResetsToGlobalDefault() {
	sender := s.AddrsStr(0)
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	topicId, err := s.createTopicWithCap(sender, 10) // inactive
	s.Require().NoError(err)
	msg := s.baseCapUpdateMsg(sender, topicId) // MaxTopInferersToReward left 0 -> reset to default
	_, err = msgServer.UpdateTopic(ctx, msg)
	s.Require().NoError(err)
	got, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(types.DefaultParams().MaxTopInferersToReward, got.MaxTopInferersToReward)
}

// changing the cap and another guarded field during a window lists both.
func (s *MsgServerTestSuite) TestUpdateTopicMaxTopInferersAndMeritBlockedListsBoth() {
	sender := s.AddrsStr(0)
	ctx, topicId := s.setupActiveTopicWithOpenWSW(sender)
	msgServer := s.EmissionsMsgServer()
	msg := s.baseWSWUpdateMsg(sender, topicId)
	msg.MeritSortitionAlpha = alloraMath.MustNewDecFromString("0.3") // changed
	msg.MaxTopInferersToReward = 10                                  // changed
	_, err := msgServer.UpdateTopic(ctx, msg)
	s.Require().ErrorIs(err, types.ErrWorkerNonceWindowNotAvailable)
	s.Require().ErrorContains(err, "merit_sortition_alpha")
	s.Require().ErrorContains(err, "max_top_inferers_to_reward")
}

// updating the cap above the global ceiling is rejected.
func (s *MsgServerTestSuite) TestUpdateTopicMaxTopInferersAboveGlobalRejected() {
	sender := s.AddrsStr(0)
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	topicId := s.createTopicForWSWTests(sender)
	msg := s.baseWSWUpdateMsg(sender, topicId)
	msg.MaxTopInferersToReward = types.DefaultParams().MaxTopInferersToReward + 1
	_, err := msgServer.UpdateTopic(ctx, msg)
	s.Require().ErrorIs(err, types.ErrTopicMaxTopInferersToRewardTooBig)
}

// TestAdmissionClampsToLiveGlobalCeiling proves the live-ceiling clamp: a topic
// keeps its high per-topic cap while governance lowers the global below it.
// Admission must clamp to the live global, so the single active slot is full and
// a lower-scoring worker is not admitted. Without the clamp the topic's frozen
// higher cap would leave an open slot and wrongly admit the worker.
func (s *MsgServerTestSuite) TestAdmissionClampsToLiveGlobalCeiling() {
	s.SetupTest()
	nonce := int64(1)
	pk := secp256k1.GenPrivKey()
	msg, topicId := s.setUpMsgInsertWorkerPayload(pk)
	s.WithBlockHeight(nonce)
	msg.WorkerDataBundle.Nonce.BlockHeight = nonce
	msg.WorkerDataBundle.InferenceForecastsBundle.Inference.BlockHeight = nonce
	msg.WorkerDataBundle.InferenceForecastsBundle.Forecast = nil
	msg = s.signMsgInsertWorkerPayload(msg, pk)

	// The topic keeps its high per-topic cap (the default resolved at creation).
	topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)
	s.Require().Equal(types.DefaultParams().MaxTopInferersToReward, topic.MaxTopInferersToReward)

	// Governance lowers the global below the topic's frozen cap.
	params := types.DefaultParams()
	params.MaxTopInferersToReward = 1
	s.Require().NoError(s.ParamsKeeper().SetParams(s.Ctx(), params))

	activeInferer := s.AddrsStr(9)
	activeScore := types.Score{
		TopicId:     topicId,
		BlockHeight: nonce,
		Address:     activeInferer,
		Score:       alloraMath.NewDecFromInt64(100),
	}
	s.Require().NoError(s.ScoresKeeper().SetInfererScoreEma(s.Ctx(), topicId, activeInferer, activeScore))
	s.Require().NoError(s.ScoresKeeper().SetLowestInfererScoreEma(s.Ctx(), topicId, activeScore))
	s.Require().NoError(s.WorkerKeeper().AddActiveInferer(s.Ctx(), topicId, activeInferer))
	s.Require().NoError(s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), topicId, msg.WorkerDataBundle.Worker))

	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &msg)
	s.Require().NoError(err)

	// Clamped to global=1, the single slot is full and the low-score worker is
	// not admitted: no stored inference and not in the active set.
	_, err = s.WorkerKeeper().GetWorkerLatestInferenceByTopicId(s.Ctx(), topicId, msg.WorkerDataBundle.Worker)
	s.Require().True(errors.Is(err, collections.ErrNotFound))
	isActive, err := s.WorkerKeeper().IsActiveInferer(s.Ctx(), topicId, msg.WorkerDataBundle.Worker)
	s.Require().NoError(err)
	s.Require().False(isActive)
}
