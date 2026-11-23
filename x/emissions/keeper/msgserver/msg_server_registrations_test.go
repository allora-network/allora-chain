package msgserver_test

import (
	"context"

	cosmosMath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *MsgServerTestSuite) TestRegistration() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	topicId := uint64(1)

	// Setup topic once
	err := s.TopicKeeper().ActivateTopic(ctx, topicId, types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_UNSPECIFIED, alloraMath.ZeroDec(), alloraMath.ZeroDec())
	s.Require().NoError(err)

	testCases := []struct {
		name      string
		isReputer bool
		checkReg  func(context.Context, uint64, string) (bool, error)
	}{
		{
			name:      "Register reputer",
			isReputer: true,
			checkReg:  s.ReputerLossKeeper().IsReputerRegisteredInTopic,
		},
		{
			name:      "Register worker",
			isReputer: false,
			checkReg:  s.WorkerKeeper().IsWorkerRegisteredInTopic,
		},
	}

	for i, tc := range testCases {
		s.Run(tc.name, func() {
			addr := s.Addrs(i)

			// Fund the account
			moduleParams, _ := s.ParamsKeeper().GetParams(ctx)
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

func (s *MsgServerTestSuite) TestRegisterEmitsFundTopicEvent() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	topicId := uint64(1)

	err := s.TopicKeeper().ActivateTopic(ctx, topicId, types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_UNSPECIFIED, alloraMath.ZeroDec(), alloraMath.ZeroDec())
	s.Require().NoError(err)

	moduleParams, err := s.ParamsKeeper().GetParams(ctx)
	s.Require().NoError(err)

	testCases := []struct {
		name      string
		isReputer bool
		source    types.TopicFundSource
	}{
		{
			name:      "worker",
			isReputer: false,
			source:    types.TopicFundSource_TOPIC_FUND_SOURCE_WORKER_REGISTRATION,
		},
		{
			name:      "reputer",
			isReputer: true,
			source:    types.TopicFundSource_TOPIC_FUND_SOURCE_REPUTER_REGISTRATION,
		},
	}

	for i, tc := range testCases {
		s.Run(tc.name, func() {
			addr := s.Addrs(i)
			s.FundAccount(moduleParams.RegistrationFee.Int64(), addr)

			revenueBefore, err := s.TopicKeeper().GetTopicFeeRevenue(ctx, topicId)
			s.Require().NoError(err)
			ecosystemBefore := s.ecosystemBalance(ctx)

			callCtx := ctx.WithEventManager(sdk.NewEventManager())
			_, err = msgServer.Register(callCtx, &types.RegisterRequest{
				Sender:    addr.String(),
				TopicId:   topicId,
				IsReputer: tc.isReputer,
				Owner:     addr.String(),
			})
			s.Require().NoError(err)

			events := fundTopicEvents(callCtx)
			s.Require().Len(events, 1)
			s.Require().Equal(topicId, events[0].TopicId)
			s.Require().Equal(addr.String(), events[0].Funder)
			s.Require().True(moduleParams.RegistrationFee.Equal(events[0].Amount))
			s.Require().True(revenueBefore.Add(moduleParams.RegistrationFee).Equal(events[0].NewRevenue))
			s.Require().Equal(tc.source, events[0].Source)
			s.Require().True(s.ecosystemBalance(ctx).Equal(ecosystemBefore.Add(moduleParams.RegistrationFee)))
		})
	}
}

// A registration fee on an inactive topic emits the registration, then the
// credit, then the status change, and the topic becomes active.
func (s *MsgServerTestSuite) TestRegisterFeeActivatesInactiveTopic() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	moduleParams, err := s.ParamsKeeper().GetParams(ctx)
	s.Require().NoError(err)
	created, err := msgServer.CreateNewTopic(ctx, s.MockTopicMsg())
	s.Require().NoError(err)

	testCases := []struct {
		name      string
		topicId   uint64
		addrIdx   int
		isReputer bool
	}{
		{name: "worker", topicId: 1, addrIdx: 4, isReputer: false},
		{name: "reputer", topicId: created.TopicId, addrIdx: 5, isReputer: true},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.Require().NoError(s.StakingKeeper().AddReputerStake(ctx, tc.topicId, s.AddrsStr(1), cosmosMath.NewInt(500000)))
			_, err := s.TopicKeeper().AddTopicFeeRevenue(ctx, tc.topicId, cosmosMath.NewInt(500000))
			s.Require().NoError(err)
			active, err := s.TopicKeeper().IsTopicActive(ctx, tc.topicId)
			s.Require().NoError(err)
			s.Require().False(active)

			addr := s.Addrs(tc.addrIdx)
			s.FundAccount(moduleParams.RegistrationFee.Int64(), addr)

			callCtx := ctx.WithEventManager(sdk.NewEventManager())
			_, err = msgServer.Register(callCtx, &types.RegisterRequest{
				Sender:    addr.String(),
				TopicId:   tc.topicId,
				IsReputer: tc.isReputer,
				Owner:     addr.String(),
			})
			s.Require().NoError(err)

			var registeredAt, fundAt, statusAt int
			seq := 0
			for _, ev := range callCtx.EventManager().Events() {
				parsed, err := sdk.ParseTypedEvent(abci.Event(ev))
				if err != nil {
					continue
				}
				seq++
				switch parsed.(type) {
				case *types.EventWorkerRegistered, *types.EventReputerRegistered:
					registeredAt = seq
				case *types.EventFundTopic:
					fundAt = seq
				case *types.EventTopicStatusChanged:
					statusAt = seq
				}
			}
			s.Require().Positive(registeredAt)
			s.Require().Positive(fundAt)
			s.Require().Positive(statusAt)
			s.Require().Less(registeredAt, fundAt, "the registration event precedes the credit")
			s.Require().Less(fundAt, statusAt, "the credit event precedes activation")

			active, err = s.TopicKeeper().IsTopicActive(ctx, tc.topicId)
			s.Require().NoError(err)
			s.Require().True(active)
		})
	}
}

func (s *MsgServerTestSuite) TestRegisterZeroFeeEmitsNoFundEvent() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	topicId := uint64(1)

	err := s.TopicKeeper().ActivateTopic(ctx, topicId, types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_UNSPECIFIED, alloraMath.ZeroDec(), alloraMath.ZeroDec())
	s.Require().NoError(err)

	_, err = msgServer.UpdateParams(ctx, &types.UpdateParamsRequest{
		Sender: s.AddrsStr(0),
		Params: &types.OptionalParams{ //nolint:exhaustruct
			RegistrationFee: []cosmosMath.Int{cosmosMath.ZeroInt()},
		},
	})
	s.Require().NoError(err)

	addr := s.Addrs(1)
	revenueBefore, err := s.TopicKeeper().GetTopicFeeRevenue(ctx, topicId)
	s.Require().NoError(err)

	callCtx := ctx.WithEventManager(sdk.NewEventManager())
	_, err = msgServer.Register(callCtx, &types.RegisterRequest{
		Sender:    addr.String(),
		TopicId:   topicId,
		IsReputer: false,
		Owner:     addr.String(),
	})
	s.Require().NoError(err)
	s.Require().Empty(fundTopicEvents(callCtx))

	revenueAfter, err := s.TopicKeeper().GetTopicFeeRevenue(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(revenueBefore.Equal(revenueAfter))

	registered, err := s.WorkerKeeper().IsWorkerRegisteredInTopic(ctx, topicId, addr.String())
	s.Require().NoError(err)
	s.Require().True(registered)
}

func (s *MsgServerTestSuite) TestRemoveRegistration() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	topicId := uint64(1)

	// Setup topic once
	err := s.TopicKeeper().ActivateTopic(ctx, topicId, types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_UNSPECIFIED, alloraMath.ZeroDec(), alloraMath.ZeroDec())
	s.Require().NoError(err)

	testCases := []struct {
		name      string
		isReputer bool
		checkReg  func(context.Context, uint64, string) (bool, error)
	}{
		{
			name:      "Remove reputer registration",
			isReputer: true,
			checkReg:  s.ReputerLossKeeper().IsReputerRegisteredInTopic,
		},
		{
			name:      "Remove worker registration",
			isReputer: false,
			checkReg:  s.WorkerKeeper().IsWorkerRegisteredInTopic,
		},
	}

	for i, tc := range testCases {
		s.Run(tc.name, func() {
			addr := s.Addrs(i)

			// Setup: Register first
			moduleParams, _ := s.ParamsKeeper().GetParams(ctx)
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
				err := s.TopicKeeper().ActivateTopic(ctx, topicId, types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_UNSPECIFIED, alloraMath.ZeroDec(), alloraMath.ZeroDec())
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
				err := s.TopicKeeper().ActivateTopic(ctx, topicId, types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_UNSPECIFIED, alloraMath.ZeroDec(), alloraMath.ZeroDec())
				s.Require().NoError(err)

				addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
				registrationInitialStake := cosmosMath.NewInt(100)

				// Add some stake but no funds to pay fees
				err = s.StakingKeeper().AddReputerStake(ctx, topicId, addr.String(), registrationInitialStake.QuoRaw(2))
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
				err := s.TopicKeeper().ActivateTopic(ctx, topicId, types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_UNSPECIFIED, alloraMath.ZeroDec(), alloraMath.ZeroDec())
				s.Require().NoError(err)

				blockedReputer := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

				return &types.RegisterRequest{
					Sender:    blockedReputer.String(),
					TopicId:   topicId,
					IsReputer: true,
					Owner:     blockedReputer.String(),
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
