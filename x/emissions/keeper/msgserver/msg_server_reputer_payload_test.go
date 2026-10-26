package msgserver_test

import (
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// reputerPayloadEvents returns all EventInsertReputerPayload typed events
// currently on the SDK context's event manager. It reconstructs each event via
// sdk.ParseTypedEvent so the assertion stays agnostic to the proto package
// version.
func reputerPayloadEvents(ctx sdk.Context) []*types.EventInsertReputerPayload {
	out := make([]*types.EventInsertReputerPayload, 0)
	for _, ev := range ctx.EventManager().Events() {
		msg, err := sdk.ParseTypedEvent(abci.Event(ev))
		if err != nil {
			continue
		}
		if e, ok := msg.(*types.EventInsertReputerPayload); ok {
			out = append(out, e)
		}
	}
	return out
}

func (s *MsgServerTestSuite) TestMsgInsertReputerPayloadFailsEarlyWindowAndWhitelistCheck() {
	reputerIndexes := testutil.ReturnIndexes(0, 1)
	workerIndexes := testutil.ReturnIndexes(1, 1)
	reputerAddr := s.Addrs(reputerIndexes[0])
	reputerValues := s.GetReputerValuesFromIndexes(reputerIndexes, workerIndexes, "0.1")

	topic := s.FullTopicSetup(workerIndexes, reputerIndexes)

	nonce, _, _ := s.TopicKeeper().GetNextPossibleChurningBlockByTopicId(s.Ctx(), topic.Id)
	s.WithBlockHeight(nonce)
	s.EndBlock()

	s.SetupInferences(topic.Id, nonce, workerIndexes)
	newBlockheight := nonce + topic.WorkerSubmissionWindow
	s.WithBlockHeight(newBlockheight)
	s.CloseWorkerNonce(topic, types.Nonce{BlockHeight: nonce})

	// Prior to the ground truth lag, should not allow reputer payload
	newBlockheight = nonce + topic.GroundTruthLag - 1
	s.WithBlockHeight(newBlockheight)

	err := s.InsertReputerLossBundle(
		topic.GetId(),
		nonce,
		reputerIndexes,
		testutil.WithReputerValues(reputerValues),
		testutil.WithSkipNetworkInferences(),
	)
	s.Require().ErrorIs(err, types.ErrReputerNonceWindowNotAvailable)

	// Valid reputer nonce window, end
	newBlockheight = nonce + topic.GroundTruthLag*2 + 1
	s.WithBlockHeight(newBlockheight)
	err = s.InsertReputerLossBundle(topic.GetId(), nonce, reputerIndexes)
	s.Require().ErrorIs(err, types.ErrReputerNonceWindowNotAvailable)

	// Remove reputer from whitelist
	err = s.WhitelistsKeeper().RemoveFromGlobalWhitelist(s.Ctx(), reputerAddr.String())
	s.Require().NoError(err)
	err = s.WhitelistsKeeper().RemoveFromTopicReputerWhitelist(s.Ctx(), topic.Id, reputerAddr.String())
	s.Require().NoError(err)

	newBlockheight = nonce + topic.GroundTruthLag*2
	s.WithBlockHeight(newBlockheight)
	err = s.InsertReputerLossBundle(topic.GetId(), nonce, reputerIndexes)
	s.Require().ErrorIs(err, types.ErrNotPermittedToSubmitReputerPayload)

	// Add reputer to whitelist so they could submit payload again
	err = s.WhitelistsKeeper().AddToTopicReputerWhitelist(s.Ctx(), topic.Id, reputerAddr.String())
	s.Require().NoError(err)

	// Valid reputer nonce window, end
	err = s.InsertReputerLossBundle(topic.GetId(), nonce, reputerIndexes)
	s.Require().NoError(err)
}

// TestMsgInsertReputerPayloadEmitsAdmissionOutcome pins the admission_outcome
// of EventInsertReputerPayload for both a stored submission and a submission
// rejected by sortition (active set full, score not above the lowest).
func (s *MsgServerTestSuite) TestMsgInsertReputerPayloadEmitsAdmissionOutcome() {
	reputerIndexes := testutil.ReturnIndexes(0, 2)
	workerIndexes := testutil.ReturnIndexes(2, 1)

	topic := s.FullTopicSetup(workerIndexes, reputerIndexes)

	// Cap the active reputer set at one so the second submission cannot fit.
	params := types.DefaultParams()
	params.MaxTopReputersToReward = 1
	s.Require().NoError(s.ParamsKeeper().SetParams(s.Ctx(), params))

	nonce, _, _ := s.TopicKeeper().GetNextPossibleChurningBlockByTopicId(s.Ctx(), topic.Id)
	s.WithBlockHeight(nonce)
	s.EndBlock()

	s.SetupInferences(topic.Id, nonce, workerIndexes)
	s.WithBlockHeight(nonce + topic.WorkerSubmissionWindow)
	s.CloseWorkerNonce(topic, types.Nonce{BlockHeight: nonce})
	s.WithBlockHeight(nonce + topic.GroundTruthLag*2)

	s.Require().NoError(s.InsertReputerLossBundle(topic.GetId(), nonce, testutil.ReturnIndexes(0, 1)))
	s.Require().NoError(s.InsertReputerLossBundle(topic.GetId(), nonce, testutil.ReturnIndexes(1, 1)))

	events := reputerPayloadEvents(s.Ctx())
	s.Require().Len(events, 2)
	s.Require().Equal(s.AddrsStr(0), events[0].Reputer)
	s.Require().Equal(types.PayloadAdmissionOutcome_PAYLOAD_ADMISSION_OUTCOME_ADMITTED, events[0].AdmissionOutcome)
	s.Require().Equal(s.AddrsStr(1), events[1].Reputer)
	s.Require().Equal(types.PayloadAdmissionOutcome_PAYLOAD_ADMISSION_OUTCOME_NOT_ADMITTED, events[1].AdmissionOutcome)
}

func (s *MsgServerTestSuite) TestMsgInsertReputerPayloadReputerNotMatchSignature() {
	reputerIndexes := testutil.ReturnIndexes(0, 1)
	reputerAddr := s.Addrs(reputerIndexes[0])
	reputerPrivateKey := s.PrivKeys(0)
	reputerPublicKeyHex := s.PubKeyHexStr(0)
	topicId := uint64(1)

	unauthReputer := s.AddrsStr(3)
	//nolint:exhaustruct
	inputValueBundle := &types.InputValueBundle{
		TopicId:             topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{ReputerNonce: &types.Nonce{BlockHeight: 1}},
		Reputer:             unauthReputer,
		InfererValues:       []*types.InputWorkerAttributedValue{{Worker: s.AddrsStr(0)}},
	}
	valueBundleSignature := s.SignInputValueBundle(inputValueBundle, reputerPrivateKey)

	// Create a InsertReputerPayloadRequest message
	lossesMsg := &types.InsertReputerPayloadRequest{
		Sender: reputerAddr.String(),
		ReputerValueBundle: &types.InputReputerValueBundle{
			ValueBundle: inputValueBundle,
			Signature:   valueBundleSignature,
			Pubkey:      reputerPublicKeyHex,
		},
	}

	_, err := s.EmissionsMsgServer().InsertReputerPayload(s.Ctx(), lossesMsg)
	s.Require().ErrorIs(err, sdkerrors.ErrUnauthorized)
}
