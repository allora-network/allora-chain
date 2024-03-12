package msgserver_test

import (
	"fmt"

	state "github.com/allora-network/allora-chain/x/emissions"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestAddWhitelistAdmin() {
	ctx := s.ctx
	require := s.Require()
	msgServer := s.msgServer

	adminAddr := sdk.AccAddress(PKS[0].Address())
	newAdminAddr := nonAdminAccounts[0]

	// Verify that newAdminAddr is not a whitelist admin
	isWhitelistAdmin, err := s.emissionsKeeper.IsWhitelistAdmin(ctx, newAdminAddr)
	require.NoError(err, "IsWhitelistAdmin should not return an error")
	require.False(isWhitelistAdmin, "newAdminAddr should not be a whitelist admin")

	// Attempt to add newAdminAddr to whitelist by adminAddr
	msg := &state.MsgAddToWhitelistAdmin{
		Sender:  adminAddr.String(),
		Address: newAdminAddr.String(),
	}

	_, err = msgServer.AddToWhitelistAdmin(ctx, msg)
	require.NoError(err, "Adding to whitelist admin should succeed")

	// Verify that newAdminAddr is now a whitelist admin
	isWhitelistAdmin, err = s.emissionsKeeper.IsWhitelistAdmin(ctx, newAdminAddr)
	require.NoError(err, "IsWhitelistAdmin should not return an error")
	require.True(isWhitelistAdmin, "newAdminAddr should be a whitelist admin")
}

func (s *KeeperTestSuite) TestAddWhitelistAdminInvalidUnauthorized() {
	ctx := s.ctx
	require := s.Require()

	nonAdminAddr := nonAdminAccounts[0]
	targetAddr := sdk.AccAddress(PKS[1].Address())

	// Attempt to add targetAddr to whitelist by nonAdminAddr
	msg := &state.MsgAddToWhitelistAdmin{
		Sender:  nonAdminAddr.String(),
		Address: targetAddr.String(),
	}

	_, err := s.msgServer.AddToWhitelistAdmin(ctx, msg)
	require.ErrorIs(err, state.ErrNotWhitelistAdmin, "Should fail due to unauthorized access")
}

func (s *KeeperTestSuite) TestRemoveWhitelistAdmin() {
	ctx := s.ctx
	require := s.Require()
	msgServer := s.msgServer

	adminAddr := sdk.AccAddress(PKS[0].Address())
	adminToRemove := sdk.AccAddress(PKS[1].Address())

	// Attempt to remove adminToRemove from the whitelist by adminAddr
	removeMsg := &state.MsgRemoveFromWhitelistAdmin{
		Sender:  adminAddr.String(),
		Address: adminToRemove.String(),
	}
	_, err := msgServer.RemoveFromWhitelistAdmin(ctx, removeMsg)
	require.NoError(err, "Removing from whitelist admin should succeed")

	// Verify that adminToRemove is no longer a whitelist admin
	isWhitelistAdmin, err := s.emissionsKeeper.IsWhitelistAdmin(ctx, adminToRemove)
	require.NoError(err, "IsWhitelistAdmin check should not return an error")
	fmt.Println(isWhitelistAdmin)
	require.False(isWhitelistAdmin, "adminToRemove should not be a whitelist admin anymore")
}

func (s *KeeperTestSuite) TestRemoveWhitelistAdminInvalidUnauthorized() {
	ctx := s.ctx
	require := s.Require()

	nonAdminAddr := nonAdminAccounts[0]

	// Attempt to remove an admin from whitelist by nonAdminAddr
	msg := &state.MsgRemoveFromWhitelistAdmin{
		Sender:  nonAdminAddr.String(),
		Address: Addr.String(),
	}

	_, err := s.msgServer.RemoveFromWhitelistAdmin(ctx, msg)
	require.ErrorIs(err, state.ErrNotWhitelistAdmin, "Should fail due to unauthorized access")
}

func (s *KeeperTestSuite) TestAddToTopicCreationWhitelist() {
	ctx := s.ctx
	require := s.Require()

	adminAddr := sdk.AccAddress(PKS[0].Address())
	newAddr := nonAdminAccounts[0]

	// Attempt to add newAddr to the topic creation whitelist by adminAddr
	msg := &state.MsgAddToTopicCreationWhitelist{
		Sender:  adminAddr.String(),
		Address: newAddr.String(),
	}

	_, err := s.msgServer.AddToTopicCreationWhitelist(ctx, msg)
	require.NoError(err, "Adding to topic creation whitelist should succeed")

	// Verify newAddr is now in the topic creation whitelist
	isInWhitelist, err := s.emissionsKeeper.IsInTopicCreationWhitelist(ctx, newAddr)
	require.NoError(err, "IsInTopicCreationWhitelist should not return an error")
	require.True(isInWhitelist, "newAddr should be in the topic creation whitelist")
}

func (s *KeeperTestSuite) TestAddToTopicCreationWhitelistInvalidUnauthorized() {
	ctx := s.ctx
	require := s.Require()

	nonAdminAddr := nonAdminAccounts[0]
	newAddr := nonAdminAccounts[1]

	// Attempt to add addressToAdd to the topic creation whitelist by nonAdminAddr
	msg := &state.MsgAddToTopicCreationWhitelist{
		Sender:  nonAdminAddr.String(),
		Address: newAddr.String(),
	}

	_, err := s.msgServer.AddToTopicCreationWhitelist(ctx, msg)
	require.ErrorIs(err, state.ErrNotWhitelistAdmin, "Non-admin should not be able to add to the topic creation whitelist")
}

func (s *KeeperTestSuite) TestRemoveFromTopicCreationWhitelist() {
	ctx := s.ctx
	require := s.Require()

	adminAddr := sdk.AccAddress(PKS[0].Address())
	addressToRemove := sdk.AccAddress(PKS[1].Address())

	// Attempt to remove addressToRemove from the topic creation whitelist by adminAddr
	removeFromWhitelistMsg := &state.MsgRemoveFromTopicCreationWhitelist{
		Sender:  adminAddr.String(),
		Address: addressToRemove.String(),
	}
	_, err := s.msgServer.RemoveFromTopicCreationWhitelist(ctx, removeFromWhitelistMsg)
	require.NoError(err, "Removing from topic creation whitelist should succeed")

	// Verify if addressToRemove is no longer in the topic creation whitelist
	isInWhitelist, err := s.emissionsKeeper.IsInTopicCreationWhitelist(ctx, addressToRemove)
	require.NoError(err, "IsInTopicCreationWhitelist check should not return an error")
	require.False(isInWhitelist, "addressToRemove should no longer be in the topic creation whitelist")
}

func (s *KeeperTestSuite) TestRemoveFromTopicCreationWhitelistInvalidUnauthorized() {
	ctx := s.ctx
	require := s.Require()

	nonAdminAddr := nonAdminAccounts[0]
	addressToRemove := nonAdminAccounts[1]

	// Attempt to remove addressToRemove from the topic creation whitelist by nonAdminAddr
	msg := &state.MsgRemoveFromTopicCreationWhitelist{
		Sender:  nonAdminAddr.String(),
		Address: addressToRemove.String(),
	}

	_, err := s.msgServer.RemoveFromTopicCreationWhitelist(ctx, msg)
	require.ErrorIs(err, state.ErrNotWhitelistAdmin, "Non-admin should not be able to remove from the topic creation whitelist")
}

func (s *KeeperTestSuite) TestAddToReputerWhitelist() {
	ctx := s.ctx
	require := s.Require()

	adminAddr := sdk.AccAddress(PKS[0].Address())
	newAddr := nonAdminAccounts[0]

	// Attempt to add newAddr to the weight setting whitelist by adminAddr
	msg := &state.MsgAddToReputerWhitelist{
		Sender:  adminAddr.String(),
		Address: newAddr.String(),
	}

	_, err := s.msgServer.AddToReputerWhitelist(ctx, msg)
	require.NoError(err, "Adding to weight setting whitelist should succeed")

	// Verify if newAddr is now in the weight setting whitelist
	isInWhitelist, err := s.emissionsKeeper.IsInReputerWhitelist(ctx, newAddr)
	require.NoError(err, "IsInReputerWhitelist check should not return an error")
	require.True(isInWhitelist, "newAddr should be in the weight setting whitelist")
}

func (s *KeeperTestSuite) TestAddToReputerWhitelistInvalidUnauthorized() {
	ctx := s.ctx
	require := s.Require()

	nonAdminAddr := nonAdminAccounts[0]
	newAddr := nonAdminAccounts[1]

	// Attempt to add addressToAdd to the weight setting whitelist by nonAdminAddr
	msg := &state.MsgAddToReputerWhitelist{
		Sender:  nonAdminAddr.String(),
		Address: newAddr.String(),
	}

	_, err := s.msgServer.AddToReputerWhitelist(ctx, msg)
	require.ErrorIs(err, state.ErrNotWhitelistAdmin, "Non-admin should not be able to add to the weight setting whitelist")
}

func (s *KeeperTestSuite) TestRemoveFromReputerWhitelist() {
	ctx := s.ctx
	require := s.Require()

	adminAddr := sdk.AccAddress(PKS[0].Address())
	addressToRemove := sdk.AccAddress(PKS[1].Address())

	// Attempt to remove addressToRemove from the weight setting whitelist by adminAddr
	removeFromWhitelistMsg := &state.MsgRemoveFromReputerWhitelist{
		Sender:  adminAddr.String(),
		Address: addressToRemove.String(),
	}
	_, err := s.msgServer.RemoveFromReputerWhitelist(ctx, removeFromWhitelistMsg)
	require.NoError(err, "Removing from weight setting whitelist should succeed")

	// Verify if addressToRemove is no longer in the weight setting whitelist
	isInWhitelist, err := s.emissionsKeeper.IsInReputerWhitelist(ctx, addressToRemove)
	require.NoError(err, "IsInReputerWhitelist check should not return an error")
	require.False(isInWhitelist, "addressToRemove should no longer be in the weight setting whitelist")
}

func (s *KeeperTestSuite) TestRemoveFromReputerWhitelistInvalidUnauthorized() {
	ctx := s.ctx
	require := s.Require()

	nonAdminAddr := nonAdminAccounts[0]
	addressToRemove := nonAdminAccounts[1]

	// Attempt to remove addressToRemove from the weight setting whitelist by nonAdminAddr
	msg := &state.MsgRemoveFromReputerWhitelist{
		Sender:  nonAdminAddr.String(),
		Address: addressToRemove.String(),
	}

	_, err := s.msgServer.RemoveFromReputerWhitelist(ctx, msg)
	require.ErrorIs(err, state.ErrNotWhitelistAdmin, "Non-admin should not be able to remove from the weight setting whitelist")
}
