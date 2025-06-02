package msgserver_test

import (
	"context"

	cosmosMath "cosmossdk.io/math"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	alloratestutil "github.com/allora-network/allora-chain/test/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *MsgServerTestSuite) TestRegistration() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	topicId := uint64(1)

	// Setup topic once
	err := s.EmissionsKeeper().ActivateTopic(ctx, topicId)
	s.Require().NoError(err)

	testCases := []struct {
		name      string
		isReputer bool
		checkReg  func(context.Context, uint64, string) (bool, error)
	}{
		{
			name:      "Register reputer",
			isReputer: true,
			checkReg:  s.EmissionsKeeper().IsReputerRegisteredInTopic,
		},
		{
			name:      "Register worker",
			isReputer: false,
			checkReg:  s.EmissionsKeeper().IsWorkerRegisteredInTopic,
		},
	}

	for i, tc := range testCases {
		s.Run(tc.name, func() {
			addr := s.Addrs(i)

			// Fund the account
			moduleParams, _ := s.EmissionsKeeper().GetParams(ctx)
			s.FundAccount(moduleParams.RegistrationFee.Int64(), addr)

			// Check not registered
			isRegistered, err := tc.checkReg(ctx, topicId, addr.String())
			s.Require().NoError(err)
			s.Require().False(isRegistered)

			// Register
			registerMsg := &types.RegisterRequest{
				Sender:    addr.String(),
				TopicId:   topicId,
				IsReputer: tc.isReputer,
				Owner:     addr.String(),
			}
			_, err = msgServer.Register(ctx, registerMsg)
			s.Require().NoError(err)

			// Check registered
			isRegistered, err = tc.checkReg(ctx, topicId, addr.String())
			s.Require().NoError(err)
			s.Require().True(isRegistered)
		})
	}
}

func (s *MsgServerTestSuite) TestRemoveRegistration() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	topicId := uint64(1)

	// Setup topic once
	err := s.EmissionsKeeper().ActivateTopic(ctx, topicId)
	s.Require().NoError(err)

	testCases := []struct {
		name      string
		isReputer bool
		checkReg  func(context.Context, uint64, string) (bool, error)
	}{
		{
			name:      "Remove reputer registration",
			isReputer: true,
			checkReg:  s.EmissionsKeeper().IsReputerRegisteredInTopic,
		},
		{
			name:      "Remove worker registration",
			isReputer: false,
			checkReg:  s.EmissionsKeeper().IsWorkerRegisteredInTopic,
		},
	}

	for i, tc := range testCases {
		s.Run(tc.name, func() {
			addr := s.Addrs(i)

			// Setup: Register first
			moduleParams, _ := s.EmissionsKeeper().GetParams(ctx)
			s.FundAccount(moduleParams.RegistrationFee.Int64(), addr)

			registerMsg := &types.RegisterRequest{
				Sender:    addr.String(),
				TopicId:   topicId,
				IsReputer: tc.isReputer,
				Owner:     addr.String(),
			}
			_, err = msgServer.Register(ctx, registerMsg)
			s.Require().NoError(err)

			// Verify registered
			isRegistered, err := tc.checkReg(ctx, topicId, addr.String())
			s.Require().NoError(err)
			s.Require().True(isRegistered)

			// Remove registration
			unregisterMsg := &types.RemoveRegistrationRequest{
				Sender:    addr.String(),
				TopicId:   topicId,
				IsReputer: tc.isReputer,
			}
			_, err = msgServer.RemoveRegistration(ctx, unregisterMsg)
			s.Require().NoError(err)

			// Verify unregistered
			isRegistered, err = tc.checkReg(ctx, topicId, addr.String())
			s.Require().NoError(err)
			s.Require().False(isRegistered)
		})
	}
}

func (s *MsgServerTestSuite) TestRegistrationErrors() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()

	testCases := []struct {
		name          string
		setup         func() *types.RegisterRequest
		expectedError error
	}{
		{
			name: "Topic does not exist",
			setup: func() *types.RegisterRequest {
				addr := s.Addrs(0)
				return &types.RegisterRequest{
					Sender:    addr.String(),
					Owner:     addr.String(),
					TopicId:   uint64(999), // non-existent topic
					IsReputer: true,
				}
			},
			expectedError: types.ErrTopicDoesNotExist,
		},
		{
			name: "Insufficient balance for registration fee",
			setup: func() *types.RegisterRequest {
				topicId := uint64(1)
				err := s.EmissionsKeeper().ActivateTopic(ctx, topicId)
				s.Require().NoError(err)

				addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
				s.MintTokensToAddress(addr, cosmosMath.NewInt(1)) // Not enough for registration fee

				return &types.RegisterRequest{
					Sender:    addr.String(),
					Owner:     addr.String(),
					TopicId:   topicId,
					IsReputer: true,
				}
			},
			expectedError: sdkerrors.ErrInsufficientFunds,
		},
		{
			name: "Insufficient funds with partial stake",
			setup: func() *types.RegisterRequest {
				topicId := uint64(1)
				err := s.EmissionsKeeper().ActivateTopic(ctx, topicId)
				s.Require().NoError(err)

				addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
				registrationInitialStake := cosmosMath.NewInt(100)

				// Add some stake but no funds to pay fees
				err = s.EmissionsKeeper().AddReputerStake(ctx, topicId, addr.String(), registrationInitialStake.QuoRaw(2))
				s.Require().NoError(err)

				return &types.RegisterRequest{
					Sender:    addr.String(),
					TopicId:   topicId,
					IsReputer: true,
					Owner:     addr.String(),
				}
			},
			expectedError: sdkerrors.ErrInsufficientFunds,
		},
		{
			name: "Blacklisted address cannot register",
			setup: func() *types.RegisterRequest {
				topicId := uint64(1)
				err := s.EmissionsKeeper().ActivateTopic(ctx, topicId)
				s.Require().NoError(err)

				_, _, _, accs := alloratestutil.GenerateTestAccounts(1)
				blockedReputer := accs[0]

				return &types.RegisterRequest{
					Sender:    blockedReputer,
					TopicId:   topicId,
					IsReputer: true,
					Owner:     blockedReputer,
				}
			},
			expectedError: sdkerrors.ErrInsufficientFunds,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			msg := tc.setup()
			_, err := msgServer.Register(ctx, msg)
			s.Require().ErrorIs(err, tc.expectedError)
		})
	}
}
