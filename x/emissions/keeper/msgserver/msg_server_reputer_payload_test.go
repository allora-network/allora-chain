package msgserver_test

import (
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *MsgServerTestSuite) TestMsgInsertReputerPayloadFailsEarlyWindowAndWhitelistCheck() {
	reputerIndexes := testutil.ReturnIndexes(0, 1)
	workerIndexes := testutil.ReturnIndexes(1, 1)
	reputerAddr := s.Addrs(reputerIndexes[0])
	reputerValues := s.GetReputerValuesFromIndexes(reputerIndexes, workerIndexes, "0.1")

	topic := s.FullTopicSetup(workerIndexes, reputerIndexes)

	nonce, _, _ := s.EmissionsKeeper().GetNextPossibleChurningBlockByTopicId(s.Ctx(), topic.Id)
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
	err = s.EmissionsKeeper().RemoveFromGlobalWhitelist(s.Ctx(), reputerAddr.String())
	s.Require().NoError(err)
	err = s.EmissionsKeeper().RemoveFromTopicReputerWhitelist(s.Ctx(), topic.Id, reputerAddr.String())
	s.Require().NoError(err)

	newBlockheight = nonce + topic.GroundTruthLag*2
	s.WithBlockHeight(newBlockheight)
	err = s.InsertReputerLossBundle(topic.GetId(), nonce, reputerIndexes)
	s.Require().ErrorIs(err, types.ErrNotPermittedToSubmitReputerPayload)

	// Add reputer to whitelist so they could submit payload again
	err = s.EmissionsKeeper().AddToTopicReputerWhitelist(s.Ctx(), topic.Id, reputerAddr.String())
	s.Require().NoError(err)

	// Valid reputer nonce window, end
	err = s.InsertReputerLossBundle(topic.GetId(), nonce, reputerIndexes)
	s.Require().NoError(err)
}
