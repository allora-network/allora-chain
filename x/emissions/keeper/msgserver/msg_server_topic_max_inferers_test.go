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
		MaxTopInferersToReward: types.DefaultParams().MaxTopInferersToReward,
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
		MaxTopInferersToReward: types.DefaultParams().MaxTopInferersToReward,
	}
}

// A request below the floor is rejected, 0 included.
func (s *MsgServerTestSuite) TestCreateTopicMaxTopInferersZeroRejected() {
	sender := s.AddrsStr(0)
	_, err := s.createTopicWithCap(sender, 0)
	s.Require().ErrorIs(err, types.ErrTopicMaxTopInferersToRewardTooSmall)
}

// The range is read from the live params, not from the module defaults.
func (s *MsgServerTestSuite) TestCreateTopicMaxTopInferersRangeReadsLiveGlobal() {
	ctx := s.Ctx()
	params, err := s.EmissionsKeeper().GetParams(ctx)
	s.Require().NoError(err)
	params.MaxTopInferersToReward = 20
	s.Require().NoError(s.EmissionsKeeper().SetParams(ctx, params))

	sender := s.AddrsStr(0)
	_, err = s.createTopicWithCap(sender, 21)
	s.Require().ErrorIs(err, types.ErrTopicMaxTopInferersToRewardTooBig)

	topicId, err := s.createTopicWithCap(sender, 20)
	s.Require().NoError(err)
	got, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(uint64(20), got.MaxTopInferersToReward)
}

// explicit values at either end of the global range are stored verbatim.
func (s *MsgServerTestSuite) TestCreateTopicMaxTopInferersExplicitValues() {
	sender := s.AddrsStr(0)
	floor := types.DefaultParams().MinTopInferersToReward
	global := types.DefaultParams().MaxTopInferersToReward

	idFloor, err := s.createTopicWithCap(sender, floor)
	s.Require().NoError(err)
	gotFloor, err := s.TopicKeeper().GetTopic(s.Ctx(), idFloor)
	s.Require().NoError(err)
	s.Require().Equal(floor, gotFloor.MaxTopInferersToReward)

	idMax, err := s.createTopicWithCap(sender, global)
	s.Require().NoError(err)
	gotMax, err := s.TopicKeeper().GetTopic(s.Ctx(), idMax)
	s.Require().NoError(err)
	s.Require().Equal(global, gotMax.MaxTopInferersToReward)
}

// The stored cap is the requested value and does not move with the globals.
func (s *MsgServerTestSuite) TestCreateTopicMaxTopInferersStoredVerbatimAndPinned() {
	ctx := s.Ctx()
	sender := s.AddrsStr(0)
	requested := uint64(12) // strictly inside the default [5, 32] range

	topicId, err := s.createTopicWithCap(sender, requested)
	s.Require().NoError(err)
	got, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(requested, got.MaxTopInferersToReward)

	for _, ceiling := range []uint64{64, 20} {
		params := types.DefaultParams()
		params.MaxTopInferersToReward = ceiling
		s.Require().NoError(s.ParamsKeeper().SetParams(ctx, params))

		got, err = s.TopicKeeper().GetTopic(ctx, topicId)
		s.Require().NoError(err)
		s.Require().Equal(requested, got.MaxTopInferersToReward)
		s.Require().Equal(requested, types.EffectiveMaxTopInferersToReward(
			got.MaxTopInferersToReward, params.MinTopInferersToReward, ceiling))
	}
}

// an explicit value above the global ceiling is rejected.
func (s *MsgServerTestSuite) TestCreateTopicMaxTopInferersAboveGlobalRejected() {
	sender := s.AddrsStr(0)
	global := types.DefaultParams().MaxTopInferersToReward
	_, err := s.createTopicWithCap(sender, global+1)
	s.Require().ErrorIs(err, types.ErrTopicMaxTopInferersToRewardTooBig)
}

// any value below the global floor is rejected, 0 included.
func (s *MsgServerTestSuite) TestCreateTopicMaxTopInferersBelowGlobalMinRejected() {
	sender := s.AddrsStr(0)
	params := types.DefaultParams()
	params.MinTopInferersToReward = 10
	s.Require().NoError(s.ParamsKeeper().SetParams(s.Ctx(), params))

	_, err := s.createTopicWithCap(sender, 9)
	s.Require().ErrorIs(err, types.ErrTopicMaxTopInferersToRewardTooSmall)

	_, err = s.createTopicWithCap(sender, 0)
	s.Require().ErrorIs(err, types.ErrTopicMaxTopInferersToRewardTooSmall)
}

// UpdateTopic applies the same floor as creation.
func (s *MsgServerTestSuite) TestUpdateTopicMaxTopInferersBelowGlobalMinRejected() {
	sender := s.AddrsStr(0)
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	topicId := s.createTopicForWSWTests(sender)

	params := types.DefaultParams()
	params.MinTopInferersToReward = 10
	s.Require().NoError(s.ParamsKeeper().SetParams(ctx, params))

	msg := s.baseWSWUpdateMsg(sender, topicId)
	msg.MaxTopInferersToReward = 9
	_, err := msgServer.UpdateTopic(ctx, msg)
	s.Require().ErrorIs(err, types.ErrTopicMaxTopInferersToRewardTooSmall)
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

// UpdateTopic applies the same floor as creation, 0 included.
func (s *MsgServerTestSuite) TestUpdateTopicMaxTopInferersZeroRejected() {
	sender := s.AddrsStr(0)
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	topicId, err := s.createTopicWithCap(sender, 10) // inactive, no window
	s.Require().NoError(err)

	msg := s.baseCapUpdateMsg(sender, topicId)
	msg.MaxTopInferersToReward = 0
	_, err = msgServer.UpdateTopic(ctx, msg)
	s.Require().ErrorIs(err, types.ErrTopicMaxTopInferersToRewardTooSmall)

	// The rejection wrote nothing.
	got, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(uint64(10), got.MaxTopInferersToReward)
}

// Resubmitting the stored cap is a no-op for the open-window guard, also after
// the global ceiling has moved.
func (s *MsgServerTestSuite) TestUpdateTopicMaxTopInferersResubmitSameCapDuringWindowAllowed() {
	sender := s.AddrsStr(0)
	// A live topic with a worker submission window currently open.
	ctx, topicId := s.setupActiveTopicWithOpenWSW(sender)
	msgServer := s.EmissionsMsgServer()

	stored, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(types.DefaultParams().MaxTopInferersToReward, stored.MaxTopInferersToReward)

	// Governance raises the ceiling; the topic keeps its own value.
	params := types.DefaultParams()
	params.MaxTopInferersToReward = stored.MaxTopInferersToReward + 8
	s.Require().NoError(s.ParamsKeeper().SetParams(ctx, params))

	msg := s.baseWSWUpdateMsg(sender, topicId)
	msg.MaxTopInferersToReward = stored.MaxTopInferersToReward
	_, err = msgServer.UpdateTopic(ctx, msg)
	s.Require().NoError(err)

	got, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(stored.MaxTopInferersToReward, got.MaxTopInferersToReward)
}

// A cap above the old ceiling is accepted once the ceiling has been raised.
func (s *MsgServerTestSuite) TestUpdateTopicMaxTopInferersCanRaiseCapAfterCeilingRaised() {
	sender := s.AddrsStr(0)
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	created := types.DefaultParams().MaxTopInferersToReward
	topicId, err := s.createTopicWithCap(sender, created) // inactive: no window
	s.Require().NoError(err)

	// Governance raises the ceiling out from under the topic.
	raised := created + 8
	params := types.DefaultParams()
	params.MaxTopInferersToReward = raised
	s.Require().NoError(s.ParamsKeeper().SetParams(ctx, params))

	// The stored cap did not move by itself.
	got, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(created, got.MaxTopInferersToReward)

	msg := s.baseCapUpdateMsg(sender, topicId)
	msg.MaxTopInferersToReward = raised
	_, err = msgServer.UpdateTopic(ctx, msg)
	s.Require().NoError(err)

	got, err = s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(raised, got.MaxTopInferersToReward)
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
	params.MinTopInferersToReward = 0
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

// Admission raises a stored cap below the global floor, so a slot stays open and
// the worker is admitted. Without the floor the stored cap of 1 would be full.
func (s *MsgServerTestSuite) TestAdmissionRaisesCapToGlobalFloor() {
	s.SetupTest()
	nonce := int64(1)
	pk := secp256k1.GenPrivKey()
	msg, topicId := s.setUpMsgInsertWorkerPayload(pk)
	s.WithBlockHeight(nonce)
	msg.WorkerDataBundle.Nonce.BlockHeight = nonce
	msg.WorkerDataBundle.InferenceForecastsBundle.Inference.BlockHeight = nonce
	msg.WorkerDataBundle.InferenceForecastsBundle.Forecast = nil
	msg = s.signMsgInsertWorkerPayload(msg, pk)

	// Store a cap of 1 on the topic, then set a global floor above it.
	topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)
	topic.MaxTopInferersToReward = 1
	s.Require().NoError(s.TopicKeeper().SetTopic(s.Ctx(), topicId, topic))

	params := types.DefaultParams()
	params.MinTopInferersToReward = 5
	s.Require().NoError(s.ParamsKeeper().SetParams(s.Ctx(), params))

	// One active inferer already fills the topic's own cap of 1.
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

	isActive, err := s.WorkerKeeper().IsActiveInferer(s.Ctx(), topicId, msg.WorkerDataBundle.Worker)
	s.Require().NoError(err)
	s.Require().True(isActive)
}
