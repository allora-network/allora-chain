package msgserver_test

import (
	"time"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *MsgServerTestSuite) TestMsgInsertReputerPayloadFailsEarlyWindowAndWhitelistCheck() {
	reputerIndexes := testutil.ReturnIndexes(0, 1)
	workerIndexes := testutil.ReturnIndexes(1, 1)
	reputerAddr := s.Addrs(reputerIndexes[0])
	reputerValues := s.GetReputerValuesFromIndexes(reputerIndexes, workerIndexes, "0.1")

	// Payload nonces must be a positive block height. Activation opens the epoch at the current height.
	s.WithBlockHeight(1)
	topic := s.FullTopicSetup(workerIndexes, reputerIndexes)
	nonceV2, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(s.Ctx(), topic.Id)
	s.Require().NoError(err)
	s.Require().True(found)
	epoch, err := s.EmissionsKeeper().GetEpoch(s.Ctx(), topic.Id, nonceV2)
	s.Require().NoError(err)
	nonce := epoch.StartBlockHeight

	s.SetupInferences(topic.Id, nonce, workerIndexes)
	s.WithBlockTime(epoch.WorkerSubmissionWindow.CloseAt.Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(s.Ctx()))

	// Ground-truth lag has not elapsed: the reputer window is still closed.
	err = s.InsertReputerLossBundle(
		topic.GetId(),
		nonce,
		reputerIndexes,
		testutil.WithReputerValues(reputerValues),
		testutil.WithSkipNetworkInferences(),
	)
	s.Require().ErrorIs(err, types.ErrReputerNonceWindowNotAvailable)

	s.WithBlockTime(epoch.ReputerSubmissionWindow.OpenAt.Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(s.Ctx()))

	// Past the reputer close, without running the close task, the window check rejects the payload.
	s.WithBlockTime(epoch.ReputerSubmissionWindow.CloseAt.Add(time.Nanosecond))
	err = s.InsertReputerLossBundle(
		topic.GetId(),
		nonce,
		reputerIndexes,
		testutil.WithReputerValues(reputerValues),
		testutil.WithSkipNetworkInferences(),
	)
	s.Require().ErrorIs(err, types.ErrReputerNonceWindowNotAvailable)

	err = s.WhitelistsKeeper().RemoveFromGlobalWhitelist(s.Ctx(), reputerAddr.String())
	s.Require().NoError(err)
	err = s.WhitelistsKeeper().RemoveFromTopicReputerWhitelist(s.Ctx(), topic.Id, reputerAddr.String())
	s.Require().NoError(err)

	s.WithBlockTime(epoch.ReputerSubmissionWindow.OpenAt.Add(time.Nanosecond))
	err = s.InsertReputerLossBundle(
		topic.GetId(),
		nonce,
		reputerIndexes,
		testutil.WithReputerValues(reputerValues),
		testutil.WithSkipNetworkInferences(),
	)
	s.Require().ErrorIs(err, types.ErrNotPermittedToSubmitReputerPayload)

	err = s.WhitelistsKeeper().AddToTopicReputerWhitelist(s.Ctx(), topic.Id, reputerAddr.String())
	s.Require().NoError(err)

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
