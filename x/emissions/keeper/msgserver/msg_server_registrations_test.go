package msgserver_test

import (
	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	"github.com/allora-network/allora-chain/x/emissions/types"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestMsgRegisterReputer() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address())
	creatorAddress := sdk.AccAddress(PKS[1].Address())
	topic1 := types.Topic{Id: 1, Creator: creatorAddress.String()}

	// Topic register
	s.emissionsKeeper.SetTopic(ctx, 1, topic1)
	s.emissionsKeeper.ActivateTopic(ctx, 1)
	// Reputer register
	registerMsg := &types.MsgRegister{
		Sender:       reputerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicId:      1,
		IsReputer:    true,
		Owner:        "Reputer",
	}

	mintAmount := sdk.NewCoins(sdk.NewInt64Coin(params.DefaultBondDenom, 100))
	s.bankKeeper.MintCoins(ctx, minttypes.ModuleName, mintAmount)
	err := s.bankKeeper.SendCoinsFromModuleToAccount(
		ctx,
		minttypes.ModuleName,
		reputerAddr,
		mintAmount,
	)
	require.NoError(err, "SendCoinsFromModuleToAccount should not return an error")
	_, err = msgServer.Register(ctx, registerMsg)
	require.NoError(err, "Registering reputer should not return an error")
}
func (s *KeeperTestSuite) TestMsgRegisterReputerInvalidLibP2PKey() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	topicId := uint64(0)

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address())

	// Topic does not exist
	registerMsg := &types.MsgRegister{
		Sender:       reputerAddr.String(),
		Owner:        reputerAddr.String(),
		LibP2PKey:    "",
		MultiAddress: "test",
		TopicId:      topicId,
		IsReputer:    true,
	}
	_, err := msgServer.Register(ctx, registerMsg)
	require.ErrorIs(err, types.ErrLibP2PKeyRequired, "Register should return an error")
}

func (s *KeeperTestSuite) TestMsgRegisterReputerInsufficientBalance() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	topicId := uint64(0)

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address())
	topic1 := types.Topic{Id: topicId, Creator: reputerAddr.String()}
	s.emissionsKeeper.SetTopic(ctx, topicId, topic1)
	s.emissionsKeeper.ActivateTopic(ctx, 1)
	// Zero initial stake

	// Topic does not exist
	registerMsg := &types.MsgRegister{
		Sender:       reputerAddr.String(),
		Owner:        reputerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicId:      topicId,
		IsReputer:    true,
	}
	_, err := msgServer.Register(ctx, registerMsg)
	require.ErrorIs(types.ErrTopicRegistrantNotEnoughDenom, err, "Register should return an error")
}

func (s *KeeperTestSuite) TestMsgRegisterReputerInsufficientDenom() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	topicId := s.CreateOneTopic()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address())
	registrationInitialStake := cosmosMath.NewUint(100)

	// Register Reputer
	reputerRegMsg := &types.MsgRegister{
		Sender:       reputerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicId:      topicId,
		IsReputer:    true,
	}

	s.emissionsKeeper.AddStake(ctx, topicId, reputerAddr, registrationInitialStake.QuoUint64(2))

	// Try to register without any funds to pay fees
	_, err := msgServer.Register(ctx, reputerRegMsg)
	require.ErrorIs(err, types.ErrTopicRegistrantNotEnoughDenom, "Register should return an error")
}

func (s *KeeperTestSuite) TestMsgRegisterReputerInvalidTopicNotExist() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	topicId := uint64(0)

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address())

	// Topic does not exist
	registerMsg := &types.MsgRegister{
		Sender:       reputerAddr.String(),
		Owner:        reputerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicId:      topicId,
		IsReputer:    true,
	}
	_, err := msgServer.Register(ctx, registerMsg)
	require.ErrorIs(err, types.ErrTopicDoesNotExist, "Register should return an error")
}
