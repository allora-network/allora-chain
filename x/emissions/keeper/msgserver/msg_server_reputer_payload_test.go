package msgserver_test

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/allora-network/allora-chain/test/testutil"
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

func (s *MsgServerTestSuite) TestMsgInsertReputerPayloadReputerNotMatchSignature() {
	reputerIndexes := testutil.ReturnIndexes(0, 1)
	reputerAddr := s.Addrs(reputerIndexes[0])
	reputerPrivateKey := s.PrivKeys(0)
	reputerPublicKeyHex := s.PubKeyHexStr(0)
	topicId := uint64(1)

	unauthReputer := s.AddrsStr(3)
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
