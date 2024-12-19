package msgserver_test

import (
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *MsgServerTestSuite) TestAddWhitelistAdmin() {
	ctx := s.ctx
	require := s.Require()
	msgServer := s.msgServer

	adminAddr := s.addrsStr[0]
	newAdminAddr := nonAdminAccounts[0].String()

	// Verify that newAdminAddr is not a whitelist admin
	isWhitelistAdmin, err := s.emissionsKeeper.IsWhitelistAdmin(ctx, newAdminAddr)
	require.NoError(err, "IsWhitelistAdmin should not return an error")
	require.False(isWhitelistAdmin, "newAdminAddr should not be a whitelist admin")

	// Attempt to add newAdminAddr to whitelist by adminAddr
	msg := &types.AddToWhitelistAdminRequest{
		Sender:  adminAddr,
		Address: newAdminAddr,
	}

	_, err = msgServer.AddToWhitelistAdmin(ctx, msg)
	require.NoError(err, "Adding to whitelist admin should succeed")

	// Verify that newAdminAddr is now a whitelist admin
	isWhitelistAdmin, err = s.emissionsKeeper.IsWhitelistAdmin(ctx, newAdminAddr)
	require.NoError(err, "IsWhitelistAdmin should not return an error")
	require.True(isWhitelistAdmin, "newAdminAddr should be a whitelist admin")
}

func (s *MsgServerTestSuite) TestAddWhitelistAdminInvalidUnauthorized() {
	ctx := s.ctx
	require := s.Require()

	nonAdminAddr := nonAdminAccounts[0]
	targetAddr := s.addrsStr[1]

	// Attempt to add targetAddr to whitelist by nonAdminAddr
	msg := &types.AddToWhitelistAdminRequest{
		Sender:  nonAdminAddr.String(),
		Address: targetAddr,
	}

	_, err := s.msgServer.AddToWhitelistAdmin(ctx, msg)
	require.ErrorIs(err, types.ErrNotPermittedToUpdateWhitelistAdmins, "Should fail due to unauthorized access")
}

func (s *MsgServerTestSuite) TestRemoveWhitelistAdmin() {
	ctx := s.ctx
	require := s.Require()
	msgServer := s.msgServer

	adminAddr := s.addrsStr[0]
	adminToRemove := s.addrsStr[1]

	// Attempt to remove adminToRemove from the whitelist by adminAddr
	removeMsg := &types.RemoveFromWhitelistAdminRequest{
		Sender:  adminAddr,
		Address: adminToRemove,
	}
	_, err := msgServer.RemoveFromWhitelistAdmin(ctx, removeMsg)
	require.NoError(err, "Removing from whitelist admin should succeed")

	// Verify that adminToRemove is no longer a whitelist admin
	isWhitelistAdmin, err := s.emissionsKeeper.IsWhitelistAdmin(ctx, adminToRemove)
	require.NoError(err, "IsWhitelistAdmin check should not return an error")
	require.False(isWhitelistAdmin, "adminToRemove should not be a whitelist admin anymore")
}

func (s *MsgServerTestSuite) TestRemoveWhitelistAdminInvalidUnauthorized() {
	ctx := s.ctx
	require := s.Require()

	nonAdminAddr := nonAdminAccounts[0]

	// Attempt to remove an admin from whitelist by nonAdminAddr
	msg := &types.RemoveFromWhitelistAdminRequest{
		Sender:  nonAdminAddr.String(),
		Address: s.addrsStr[0],
	}

	_, err := s.msgServer.RemoveFromWhitelistAdmin(ctx, msg)
	require.ErrorIs(err, types.ErrNotPermittedToUpdateWhitelistAdmins, "Should fail due to unauthorized access")
}

func (s *MsgServerTestSuite) TestAddToGlobalWhitelist() {
	ctx := s.ctx
	require := s.Require()
	msgServer := s.msgServer

	adminAddr := s.addrsStr[0]
	targetAddr := s.addrsStr[1]

	// Add targetAddr to global whitelist by adminAddr
	msg := &types.AddToGlobalWhitelistRequest{
		Sender:  adminAddr,
		Address: targetAddr,
	}
	_, err := msgServer.AddToGlobalWhitelist(ctx, msg)
	require.NoError(err, "Adding to global whitelist should succeed")

	// Verify targetAddr is now in global whitelist
	isWhitelisted, err := s.emissionsKeeper.IsWhitelistedGlobalActor(ctx, targetAddr)
	require.NoError(err, "IsWhitelistedGlobalActor check should not return an error")
	require.True(isWhitelisted, "targetAddr should be in global whitelist")
}

func (s *MsgServerTestSuite) TestAddToGlobalWhitelistInvalidUnauthorized() {
	ctx := s.ctx
	require := s.Require()

	nonAdminAddr := nonAdminAccounts[0]
	targetAddr := s.addrsStr[1]

	// Attempt to add targetAddr to global whitelist by nonAdminAddr
	msg := &types.AddToGlobalWhitelistRequest{
		Sender:  nonAdminAddr.String(),
		Address: targetAddr,
	}

	_, err := s.msgServer.AddToGlobalWhitelist(ctx, msg)
	require.ErrorIs(err, types.ErrNotPermittedToUpdateGlobalWhitelist, "Should fail due to unauthorized access")
}

func (s *MsgServerTestSuite) TestRemoveFromGlobalWhitelist() {
	ctx := s.ctx
	require := s.Require()
	msgServer := s.msgServer

	adminAddr := s.addrsStr[0]
	targetAddr := s.addrsStr[1]

	// First add targetAddr to global whitelist
	addMsg := &types.AddToGlobalWhitelistRequest{
		Sender:  adminAddr,
		Address: targetAddr,
	}
	_, err := msgServer.AddToGlobalWhitelist(ctx, addMsg)
	require.NoError(err, "Adding to global whitelist should succeed")

	// Remove targetAddr from global whitelist
	removeMsg := &types.RemoveFromGlobalWhitelistRequest{
		Sender:  adminAddr,
		Address: targetAddr,
	}
	_, err = msgServer.RemoveFromGlobalWhitelist(ctx, removeMsg)
	require.NoError(err, "Removing from global whitelist should succeed")

	// Verify targetAddr is no longer in global whitelist
	isWhitelisted, err := s.emissionsKeeper.IsWhitelistedGlobalActor(ctx, targetAddr)
	require.NoError(err, "IsWhitelistedGlobalActor check should not return an error")
	require.False(isWhitelisted, "targetAddr should not be in global whitelist")
}

func (s *MsgServerTestSuite) TestRemoveFromGlobalWhitelistInvalidUnauthorized() {
	ctx := s.ctx
	require := s.Require()

	nonAdminAddr := nonAdminAccounts[0]
	targetAddr := s.addrsStr[1]

	// Attempt to remove targetAddr from global whitelist by nonAdminAddr
	msg := &types.RemoveFromGlobalWhitelistRequest{
		Sender:  nonAdminAddr.String(),
		Address: targetAddr,
	}

	_, err := s.msgServer.RemoveFromGlobalWhitelist(ctx, msg)
	require.ErrorIs(err, types.ErrNotPermittedToUpdateGlobalWhitelist, "Should fail due to unauthorized access")
}

func (s *MsgServerTestSuite) TestAddToGlobalWorkerWhitelist() {
	ctx := s.ctx
	require := s.Require()
	msgServer := s.msgServer

	adminAddr := s.addrsStr[0]
	targetAddr := s.addrsStr[1]

	// Add targetAddr to global worker whitelist
	msg := &types.AddToGlobalWorkerWhitelistRequest{
		Sender:  adminAddr,
		Address: targetAddr,
	}
	_, err := msgServer.AddToGlobalWorkerWhitelist(ctx, msg)
	require.NoError(err, "Adding to global worker whitelist should succeed")

	// Verify targetAddr is in global worker whitelist
	isWhitelisted, err := s.emissionsKeeper.IsWhitelistedGlobalWorker(ctx, targetAddr)
	require.NoError(err, "IsWhitelistedGlobalWorker check should not return an error")
	require.True(isWhitelisted, "targetAddr should be in global worker whitelist")
}

func (s *MsgServerTestSuite) TestAddToGlobalWorkerWhitelistInvalidUnauthorized() {
	ctx := s.ctx
	require := s.Require()

	nonAdminAddr := nonAdminAccounts[0]
	targetAddr := s.addrsStr[1]

	// Attempt to add targetAddr to global worker whitelist by nonAdminAddr
	msg := &types.AddToGlobalWorkerWhitelistRequest{
		Sender:  nonAdminAddr.String(),
		Address: targetAddr,
	}

	_, err := s.msgServer.AddToGlobalWorkerWhitelist(ctx, msg)
	require.ErrorIs(err, types.ErrNotPermittedToUpdateGlobalWhitelist, "Should fail due to unauthorized access")
}

func (s *MsgServerTestSuite) TestRemoveFromGlobalWorkerWhitelist() {
	ctx := s.ctx
	require := s.Require()
	msgServer := s.msgServer

	adminAddr := s.addrsStr[0]
	targetAddr := s.addrsStr[1]

	// First add targetAddr to global worker whitelist
	addMsg := &types.AddToGlobalWorkerWhitelistRequest{
		Sender:  adminAddr,
		Address: targetAddr,
	}
	_, err := msgServer.AddToGlobalWorkerWhitelist(ctx, addMsg)
	require.NoError(err, "Adding to global worker whitelist should succeed")

	// Remove targetAddr from global worker whitelist
	removeMsg := &types.RemoveFromGlobalWorkerWhitelistRequest{
		Sender:  adminAddr,
		Address: targetAddr,
	}
	_, err = msgServer.RemoveFromGlobalWorkerWhitelist(ctx, removeMsg)
	require.NoError(err, "Removing from global worker whitelist should succeed")

	// Verify targetAddr is no longer in global worker whitelist
	isWhitelisted, err := s.emissionsKeeper.IsWhitelistedGlobalWorker(ctx, targetAddr)
	require.NoError(err, "IsWhitelistedGlobalWorker check should not return an error")
	require.False(isWhitelisted, "targetAddr should not be in global worker whitelist")
}

func (s *MsgServerTestSuite) TestAddToGlobalReputerWhitelist() {
	ctx := s.ctx
	require := s.Require()
	msgServer := s.msgServer

	adminAddr := s.addrsStr[0]
	targetAddr := s.addrsStr[1]

	// Add targetAddr to global reputer whitelist
	msg := &types.AddToGlobalReputerWhitelistRequest{
		Sender:  adminAddr,
		Address: targetAddr,
	}
	_, err := msgServer.AddToGlobalReputerWhitelist(ctx, msg)
	require.NoError(err, "Adding to global reputer whitelist should succeed")

	// Verify targetAddr is in global reputer whitelist
	isWhitelisted, err := s.emissionsKeeper.IsWhitelistedGlobalReputer(ctx, targetAddr)
	require.NoError(err, "IsWhitelistedGlobalReputer check should not return an error")
	require.True(isWhitelisted, "targetAddr should be in global reputer whitelist")
}

func (s *MsgServerTestSuite) TestRemoveFromGlobalReputerWhitelist() {
	ctx := s.ctx
	require := s.Require()
	msgServer := s.msgServer

	adminAddr := s.addrsStr[0]
	targetAddr := s.addrsStr[1]

	// First add targetAddr to global reputer whitelist
	addMsg := &types.AddToGlobalReputerWhitelistRequest{
		Sender:  adminAddr,
		Address: targetAddr,
	}
	_, err := msgServer.AddToGlobalReputerWhitelist(ctx, addMsg)
	require.NoError(err, "Adding to global reputer whitelist should succeed")

	// Remove targetAddr from global reputer whitelist
	removeMsg := &types.RemoveFromGlobalReputerWhitelistRequest{
		Sender:  adminAddr,
		Address: targetAddr,
	}
	_, err = msgServer.RemoveFromGlobalReputerWhitelist(ctx, removeMsg)
	require.NoError(err, "Removing from global reputer whitelist should succeed")

	// Verify targetAddr is no longer in global reputer whitelist
	isWhitelisted, err := s.emissionsKeeper.IsWhitelistedGlobalReputer(ctx, targetAddr)
	require.NoError(err, "IsWhitelistedGlobalReputer check should not return an error")
	require.False(isWhitelisted, "targetAddr should not be in global reputer whitelist")
}

func (s *MsgServerTestSuite) TestAddToGlobalAdminWhitelist() {
	ctx := s.ctx
	require := s.Require()
	msgServer := s.msgServer

	adminAddr := s.addrsStr[0]
	targetAddr := s.addrsStr[1]

	// Add targetAddr to global admin whitelist
	msg := &types.AddToGlobalAdminWhitelistRequest{
		Sender:  adminAddr,
		Address: targetAddr,
	}
	_, err := msgServer.AddToGlobalAdminWhitelist(ctx, msg)
	require.NoError(err, "Adding to global admin whitelist should succeed")

	// Verify targetAddr is in global admin whitelist
	canUpdate, err := s.emissionsKeeper.CanUpdateAllGlobalWhitelists(ctx, targetAddr)
	require.NoError(err, "CanUpdateAllGlobalWhitelists check should not return an error")
	require.True(canUpdate, "targetAddr should be in global admin whitelist")
}

func (s *MsgServerTestSuite) TestRemoveFromGlobalAdminWhitelist() {
	ctx := s.ctx
	require := s.Require()
	msgServer := s.msgServer

	adminAddr := s.addrsStr[0]
	targetAddr := s.addrsStr[1]

	// First add targetAddr to global admin whitelist
	addMsg := &types.AddToGlobalAdminWhitelistRequest{
		Sender:  adminAddr,
		Address: targetAddr,
	}
	_, err := msgServer.AddToGlobalAdminWhitelist(ctx, addMsg)
	require.NoError(err, "Adding to global admin whitelist should succeed")

	// Verify targetAddr is in global admin whitelist before removal
	isWhitelisted, err := s.emissionsKeeper.IsWhitelistedGlobalAdmin(ctx, targetAddr)
	require.NoError(err, "IsWhitelistedGlobalAdmin check should not return an error")
	require.True(isWhitelisted, "targetAddr should be in global admin whitelist before removal")

	// Remove targetAddr from global admin whitelist
	removeMsg := &types.RemoveFromGlobalAdminWhitelistRequest{
		Sender:  adminAddr,
		Address: targetAddr,
	}
	_, err = msgServer.RemoveFromGlobalAdminWhitelist(ctx, removeMsg)
	require.NoError(err, "Removing from global admin whitelist should succeed")

	// Verify targetAddr is no longer in global admin whitelist
	isWhitelisted, err = s.emissionsKeeper.IsWhitelistedGlobalAdmin(ctx, targetAddr)
	require.NoError(err, "IsWhitelistedGlobalAdmin check should not return an error")
	require.False(isWhitelisted, "targetAddr should not be in global admin whitelist")
}

func (s *MsgServerTestSuite) TestBulkAddToGlobalWorkerWhitelist() {
	ctx := s.ctx
	require := s.Require()
	msgServer := s.msgServer
	adminAddr := s.addrsStr[0]

	// First add some addresses
	addresses := []string{
		nonAdminAccounts[0].String(),
		nonAdminAccounts[1].String(),
		nonAdminAccounts[2].String(),
	}

	// Add addresses to whitelist
	msg := &types.BulkAddToGlobalWorkerWhitelistRequest{
		Sender:    adminAddr,
		Addresses: addresses,
	}
	_, err := msgServer.BulkAddToGlobalWorkerWhitelist(ctx, msg)
	require.NoError(err, "Bulk adding to global worker whitelist should succeed")

	// Verify all addresses were added
	for _, addr := range addresses {
		isWhitelisted, err := s.emissionsKeeper.IsWhitelistedGlobalWorker(ctx, addr)
		require.NoError(err)
		require.True(isWhitelisted, "Address should be whitelisted")
	}

	// Set max length parameter
	params, err := s.emissionsKeeper.GetParams(ctx)
	require.NoError(err)
	params.MaxWhitelistInputArrayLength = 3
	err = s.emissionsKeeper.SetParams(ctx, params)
	require.NoError(err)

	// Try adding more than max length
	tooManyAddresses := make([]string, params.MaxWhitelistInputArrayLength+1)
	for i := range tooManyAddresses {
		tooManyAddresses[i] = nonAdminAccounts[i].String()
	}

	msg = &types.BulkAddToGlobalWorkerWhitelistRequest{
		Sender:    adminAddr,
		Addresses: tooManyAddresses,
	}
	_, err = msgServer.BulkAddToGlobalWorkerWhitelist(ctx, msg)
	require.ErrorIs(err, types.ErrMaxWhitelistInputArrayLengthExceeded)
}

func (s *MsgServerTestSuite) TestBulkRemoveFromGlobalWorkerWhitelist() {
	ctx := s.ctx
	require := s.Require()
	msgServer := s.msgServer
	adminAddr := s.addrsStr[0]

	// First add some addresses
	addresses := []string{
		nonAdminAccounts[0].String(),
		nonAdminAccounts[1].String(),
		nonAdminAccounts[2].String(),
	}

	addMsg := &types.BulkAddToGlobalWorkerWhitelistRequest{
		Sender:    adminAddr,
		Addresses: addresses,
	}
	_, err := msgServer.BulkAddToGlobalWorkerWhitelist(ctx, addMsg)
	require.NoError(err)

	// Remove addresses
	removeMsg := &types.BulkRemoveFromGlobalWorkerWhitelistRequest{
		Sender:    adminAddr,
		Addresses: addresses,
	}
	_, err = msgServer.BulkRemoveFromGlobalWorkerWhitelist(ctx, removeMsg)
	require.NoError(err, "Bulk removing from global worker whitelist should succeed")

	// Verify addresses were removed
	for _, addr := range addresses {
		isWhitelisted, err := s.emissionsKeeper.IsWhitelistedGlobalWorker(ctx, addr)
		require.NoError(err)
		require.False(isWhitelisted, "Address should not be whitelisted")
	}

	// Set max length parameter
	params, err := s.emissionsKeeper.GetParams(ctx)
	require.NoError(err)
	params.MaxWhitelistInputArrayLength = 3
	err = s.emissionsKeeper.SetParams(ctx, params)
	require.NoError(err)

	// Try adding more than max length
	tooManyAddresses := make([]string, params.MaxWhitelistInputArrayLength+1)
	for i := range tooManyAddresses {
		tooManyAddresses[i] = nonAdminAccounts[i].String()
	}

	msg := &types.BulkAddToGlobalWorkerWhitelistRequest{
		Sender:    adminAddr,
		Addresses: tooManyAddresses,
	}
	_, err = msgServer.BulkAddToGlobalWorkerWhitelist(ctx, msg)
	require.ErrorIs(err, types.ErrMaxWhitelistInputArrayLengthExceeded)
}

func (s *MsgServerTestSuite) TestBulkAddToGlobalReputerWhitelist() {
	ctx := s.ctx
	require := s.Require()
	msgServer := s.msgServer
	adminAddr := s.addrsStr[0]

	// First add some addresses
	addresses := []string{
		nonAdminAccounts[0].String(),
		nonAdminAccounts[1].String(),
		nonAdminAccounts[2].String(),
	}

	// Add addresses to whitelist
	msg := &types.BulkAddToGlobalReputerWhitelistRequest{
		Sender:    adminAddr,
		Addresses: addresses,
	}
	_, err := msgServer.BulkAddToGlobalReputerWhitelist(ctx, msg)
	require.NoError(err, "Bulk adding to global reputer whitelist should succeed")

	// Verify all addresses were added
	for _, addr := range addresses {
		isWhitelisted, err := s.emissionsKeeper.IsWhitelistedGlobalReputer(ctx, addr)
		require.NoError(err)
		require.True(isWhitelisted, "Address should be whitelisted")
	}

	// Set max length parameter
	params, err := s.emissionsKeeper.GetParams(ctx)
	require.NoError(err)
	params.MaxWhitelistInputArrayLength = 3
	err = s.emissionsKeeper.SetParams(ctx, params)
	require.NoError(err)

	// Try adding more than max length
	tooManyAddresses := make([]string, params.MaxWhitelistInputArrayLength+1)
	for i := range tooManyAddresses {
		tooManyAddresses[i] = nonAdminAccounts[i].String()
	}

	msg = &types.BulkAddToGlobalReputerWhitelistRequest{
		Sender:    adminAddr,
		Addresses: tooManyAddresses,
	}
	_, err = msgServer.BulkAddToGlobalReputerWhitelist(ctx, msg)
	require.ErrorIs(err, types.ErrMaxWhitelistInputArrayLengthExceeded)
}

func (s *MsgServerTestSuite) TestBulkRemoveFromGlobalReputerWhitelist() {
	ctx := s.ctx
	require := s.Require()
	msgServer := s.msgServer
	adminAddr := s.addrsStr[0]

	// First add some addresses
	addresses := []string{
		nonAdminAccounts[0].String(),
		nonAdminAccounts[1].String(),
		nonAdminAccounts[2].String(),
	}

	addMsg := &types.BulkAddToGlobalReputerWhitelistRequest{
		Sender:    adminAddr,
		Addresses: addresses,
	}
	_, err := msgServer.BulkAddToGlobalReputerWhitelist(ctx, addMsg)
	require.NoError(err)

	// Remove addresses
	removeMsg := &types.BulkRemoveFromGlobalReputerWhitelistRequest{
		Sender:    adminAddr,
		Addresses: addresses,
	}
	_, err = msgServer.BulkRemoveFromGlobalReputerWhitelist(ctx, removeMsg)
	require.NoError(err, "Bulk removing from global reputer whitelist should succeed")

	// Verify addresses were removed
	for _, addr := range addresses {
		isWhitelisted, err := s.emissionsKeeper.IsWhitelistedGlobalReputer(ctx, addr)
		require.NoError(err)
		require.False(isWhitelisted, "Address should not be whitelisted")
	}

	// Set max length parameter
	params, err := s.emissionsKeeper.GetParams(ctx)
	require.NoError(err)
	params.MaxWhitelistInputArrayLength = 3
	err = s.emissionsKeeper.SetParams(ctx, params)
	require.NoError(err)

	// Try adding more than max length
	tooManyAddresses := make([]string, params.MaxWhitelistInputArrayLength+1)
	for i := range tooManyAddresses {
		tooManyAddresses[i] = nonAdminAccounts[i].String()
	}

	msg := &types.BulkAddToGlobalReputerWhitelistRequest{
		Sender:    adminAddr,
		Addresses: tooManyAddresses,
	}
	_, err = msgServer.BulkAddToGlobalReputerWhitelist(ctx, msg)
	require.ErrorIs(err, types.ErrMaxWhitelistInputArrayLengthExceeded)
}

func (s *MsgServerTestSuite) TestBulkAddToTopicWorkerWhitelist() {
	ctx := s.ctx
	require := s.Require()
	msgServer := s.msgServer
	adminAddr := s.addrsStr[0]
	topicId := s.CreateOneTopic().Id

	// First add some addresses
	addresses := []string{
		nonAdminAccounts[0].String(),
		nonAdminAccounts[1].String(),
		nonAdminAccounts[2].String(),
	}

	addMsg := &types.BulkAddToTopicWorkerWhitelistRequest{
		Sender:    adminAddr,
		Addresses: addresses,
		TopicId:   topicId,
	}
	_, err := msgServer.BulkAddToTopicWorkerWhitelist(ctx, addMsg)
	require.NoError(err)

	// Verify addresses were added
	for _, addr := range addresses {
		isWhitelisted, err := s.emissionsKeeper.IsWhitelistedTopicWorker(ctx, topicId, addr)
		require.NoError(err)
		require.True(isWhitelisted, "Address should be whitelisted")
	}

	// Set max length parameter
	params, err := s.emissionsKeeper.GetParams(ctx)
	require.NoError(err)
	params.MaxWhitelistInputArrayLength = 3
	err = s.emissionsKeeper.SetParams(ctx, params)
	require.NoError(err)

	// Try adding more than max length
	tooManyAddresses := make([]string, params.MaxWhitelistInputArrayLength+1)
	for i := range tooManyAddresses {
		tooManyAddresses[i] = nonAdminAccounts[i].String()
	}

	msg := &types.BulkAddToTopicWorkerWhitelistRequest{
		Sender:    adminAddr,
		Addresses: tooManyAddresses,
		TopicId:   topicId,
	}
	_, err = msgServer.BulkAddToTopicWorkerWhitelist(ctx, msg)
	require.ErrorIs(err, types.ErrMaxWhitelistInputArrayLengthExceeded)
}

func (s *MsgServerTestSuite) TestBulkRemoveFromTopicWorkerWhitelist() {
	ctx := s.ctx
	require := s.Require()
	msgServer := s.msgServer
	adminAddr := s.addrsStr[0]
	topicId := s.CreateOneTopic().Id

	// First add some addresses
	addresses := []string{
		nonAdminAccounts[0].String(),
		nonAdminAccounts[1].String(),
		nonAdminAccounts[2].String(),
	}

	addMsg := &types.BulkAddToTopicWorkerWhitelistRequest{
		Sender:    adminAddr,
		Addresses: addresses,
		TopicId:   topicId,
	}
	_, err := msgServer.BulkAddToTopicWorkerWhitelist(ctx, addMsg)
	require.NoError(err)

	// Remove addresses
	removeMsg := &types.BulkRemoveFromTopicWorkerWhitelistRequest{
		Sender:    adminAddr,
		Addresses: addresses,
		TopicId:   topicId,
	}
	_, err = msgServer.BulkRemoveFromTopicWorkerWhitelist(ctx, removeMsg)
	require.NoError(err, "Bulk removing from topic worker whitelist should succeed")

	// Verify addresses were removed
	for _, addr := range addresses {
		isWhitelisted, err := s.emissionsKeeper.IsWhitelistedTopicWorker(ctx, topicId, addr)
		require.NoError(err)
		require.False(isWhitelisted, "Address should not be whitelisted")
	}

	// Set max length parameter
	params, err := s.emissionsKeeper.GetParams(ctx)
	require.NoError(err)
	params.MaxWhitelistInputArrayLength = 3
	err = s.emissionsKeeper.SetParams(ctx, params)
	require.NoError(err)

	// Try adding more than max length
	tooManyAddresses := make([]string, params.MaxWhitelistInputArrayLength+1)
	for i := range tooManyAddresses {
		tooManyAddresses[i] = nonAdminAccounts[i].String()
	}

	msg := &types.BulkRemoveFromTopicWorkerWhitelistRequest{
		Sender:    adminAddr,
		Addresses: tooManyAddresses,
		TopicId:   topicId,
	}
	_, err = msgServer.BulkRemoveFromTopicWorkerWhitelist(ctx, msg)
	require.ErrorIs(err, types.ErrMaxWhitelistInputArrayLengthExceeded)
}

func (s *MsgServerTestSuite) TestBulkAddToTopicReputerWhitelist() {
	ctx := s.ctx
	require := s.Require()
	msgServer := s.msgServer
	adminAddr := s.addrsStr[0]
	topicId := s.CreateOneTopic().Id

	// First add some addresses
	addresses := []string{
		nonAdminAccounts[0].String(),
		nonAdminAccounts[1].String(),
		nonAdminAccounts[2].String(),
	}

	addMsg := &types.BulkAddToTopicReputerWhitelistRequest{
		Sender:    adminAddr,
		Addresses: addresses,
		TopicId:   topicId,
	}
	_, err := msgServer.BulkAddToTopicReputerWhitelist(ctx, addMsg)
	require.NoError(err)

	// Verify addresses were added
	for _, addr := range addresses {
		isWhitelisted, err := s.emissionsKeeper.IsWhitelistedTopicReputer(ctx, topicId, addr)
		require.NoError(err)
		require.True(isWhitelisted, "Address should be whitelisted")
	}

	// Set max length parameter
	params, err := s.emissionsKeeper.GetParams(ctx)
	require.NoError(err)
	params.MaxWhitelistInputArrayLength = 3
	err = s.emissionsKeeper.SetParams(ctx, params)
	require.NoError(err)

	// Try adding more than max length
	tooManyAddresses := make([]string, params.MaxWhitelistInputArrayLength+1)
	for i := range tooManyAddresses {
		tooManyAddresses[i] = nonAdminAccounts[i].String()
	}

	msg := &types.BulkAddToTopicReputerWhitelistRequest{
		Sender:    adminAddr,
		Addresses: tooManyAddresses,
		TopicId:   topicId,
	}
	_, err = msgServer.BulkAddToTopicReputerWhitelist(ctx, msg)
	require.ErrorIs(err, types.ErrMaxWhitelistInputArrayLengthExceeded)
}

func (s *MsgServerTestSuite) TestBulkRemoveFromTopicReputerWhitelist() {
	ctx := s.ctx
	require := s.Require()
	msgServer := s.msgServer
	adminAddr := s.addrsStr[0]
	topicId := s.CreateOneTopic().Id

	// First add some addresses
	addresses := []string{
		nonAdminAccounts[0].String(),
		nonAdminAccounts[1].String(),
		nonAdminAccounts[2].String(),
	}

	addMsg := &types.BulkAddToTopicReputerWhitelistRequest{
		Sender:    adminAddr,
		Addresses: addresses,
		TopicId:   topicId,
	}
	_, err := msgServer.BulkAddToTopicReputerWhitelist(ctx, addMsg)
	require.NoError(err)

	// Remove addresses
	removeMsg := &types.BulkRemoveFromTopicReputerWhitelistRequest{
		Sender:    adminAddr,
		Addresses: addresses,
		TopicId:   topicId,
	}
	_, err = msgServer.BulkRemoveFromTopicReputerWhitelist(ctx, removeMsg)
	require.NoError(err, "Bulk removing from topic reputer whitelist should succeed")

	// Verify addresses were removed
	for _, addr := range addresses {
		isWhitelisted, err := s.emissionsKeeper.IsWhitelistedTopicReputer(ctx, topicId, addr)
		require.NoError(err)
		require.False(isWhitelisted, "Address should not be whitelisted")
	}

	// Set max length parameter
	params, err := s.emissionsKeeper.GetParams(ctx)
	require.NoError(err)
	params.MaxWhitelistInputArrayLength = 3
	err = s.emissionsKeeper.SetParams(ctx, params)
	require.NoError(err)

	// Try adding more than max length
	tooManyAddresses := make([]string, params.MaxWhitelistInputArrayLength+1)
	for i := range tooManyAddresses {
		tooManyAddresses[i] = nonAdminAccounts[i].String()
	}

	msg := &types.BulkRemoveFromTopicReputerWhitelistRequest{
		Sender:    adminAddr,
		Addresses: tooManyAddresses,
		TopicId:   topicId,
	}
	_, err = msgServer.BulkRemoveFromTopicReputerWhitelist(ctx, msg)
	require.ErrorIs(err, types.ErrMaxWhitelistInputArrayLengthExceeded)
}

func (s *MsgServerTestSuite) TestEnableTopicWorkerWhitelist() {
	ctx := s.ctx
	require := s.Require()
	msgServer := s.msgServer
	adminAddr := s.addrsStr[0]
	topicId := s.CreateOneTopic().Id

	// Enable whitelist for topic
	msg := &types.EnableTopicWorkerWhitelistRequest{
		Sender:  adminAddr,
		TopicId: topicId,
	}
	_, err := msgServer.EnableTopicWorkerWhitelist(ctx, msg)
	require.NoError(err, "Enabling topic whitelist should succeed")

	// Verify topic whitelist is enabled
	isEnabled, err := s.emissionsKeeper.IsTopicWorkerWhitelistEnabled(ctx, topicId)
	require.NoError(err, "IsTopicWorkerWhitelistEnabled check should not return an error")
	require.True(isEnabled, "Topic worker whitelist should be enabled")
}

func (s *MsgServerTestSuite) TestEnableTopicWorkerWhitelistInvalidUnauthorized() {
	ctx := s.ctx
	require := s.Require()
	nonAdminAddr := nonAdminAccounts[0]
	topicId := s.CreateOneTopic().Id

	// Attempt to enable whitelist for topic by nonAdminAddr
	msg := &types.EnableTopicWorkerWhitelistRequest{
		Sender:  nonAdminAddr.String(),
		TopicId: topicId,
	}

	_, err := s.msgServer.EnableTopicWorkerWhitelist(ctx, msg)
	require.ErrorIs(err, types.ErrNotPermittedToUpdateTopicWhitelist, "Should fail due to unauthorized access")
}

func (s *MsgServerTestSuite) TestEnableTopicWorkerWhitelistTopicDoesNotExist() {
	ctx := s.ctx
	require := s.Require()
	nonAdminAddr := nonAdminAccounts[0]
	topicId := uint64(1000)

	// Attempt to enable whitelist for topic by nonAdminAddr
	msg := &types.EnableTopicWorkerWhitelistRequest{
		Sender:  nonAdminAddr.String(),
		TopicId: topicId,
	}

	_, err := s.msgServer.EnableTopicWorkerWhitelist(ctx, msg)
	require.ErrorIs(err, types.ErrTopicDoesNotExist, "Should fail due to topic not existing")
}

func (s *MsgServerTestSuite) TestEnableTopicReputerWhitelist() {
	ctx := s.ctx
	require := s.Require()
	msgServer := s.msgServer
	adminAddr := s.addrsStr[0]
	topicId := s.CreateOneTopic().Id

	// Enable whitelist for topic
	msg := &types.EnableTopicReputerWhitelistRequest{
		Sender:  adminAddr,
		TopicId: topicId,
	}
	_, err := msgServer.EnableTopicReputerWhitelist(ctx, msg)
	require.NoError(err, "Enabling topic whitelist should succeed")

	// Verify topic whitelist is enabled
	isEnabled, err := s.emissionsKeeper.IsTopicReputerWhitelistEnabled(ctx, topicId)
	require.NoError(err, "IsTopicReputerWhitelistEnabled check should not return an error")
	require.True(isEnabled, "Topic reputer whitelist should be enabled")
}

func (s *MsgServerTestSuite) TestEnableTopicReputerWhitelistInvalidUnauthorized() {
	ctx := s.ctx
	require := s.Require()
	nonAdminAddr := nonAdminAccounts[0]
	topicId := s.CreateOneTopic().Id

	// Attempt to enable whitelist for topic by nonAdminAddr
	msg := &types.EnableTopicReputerWhitelistRequest{
		Sender:  nonAdminAddr.String(),
		TopicId: topicId,
	}

	_, err := s.msgServer.EnableTopicReputerWhitelist(ctx, msg)
	require.ErrorIs(err, types.ErrNotPermittedToUpdateTopicWhitelist, "Should fail due to unauthorized access")
}

func (s *MsgServerTestSuite) TestEnableTopicReputerWhitelistTopicDoesNotExist() {
	ctx := s.ctx
	require := s.Require()
	nonAdminAddr := nonAdminAccounts[0]
	topicId := uint64(1000)

	// Attempt to enable whitelist for topic by nonAdminAddr
	msg := &types.EnableTopicReputerWhitelistRequest{
		Sender:  nonAdminAddr.String(),
		TopicId: topicId,
	}

	_, err := s.msgServer.EnableTopicReputerWhitelist(ctx, msg)
	require.ErrorIs(err, types.ErrTopicDoesNotExist, "Should fail due to topic not existing")
}

func (s *MsgServerTestSuite) TestDisableTopicWorkerWhitelist() {
	ctx := s.ctx
	require := s.Require()
	msgServer := s.msgServer
	adminAddr := s.addrsStr[0]
	topicId := s.CreateOneTopic().Id

	// First enable whitelist for topic
	enableMsg := &types.EnableTopicWorkerWhitelistRequest{
		Sender:  adminAddr,
		TopicId: topicId,
	}
	_, err := msgServer.EnableTopicWorkerWhitelist(ctx, enableMsg)
	require.NoError(err, "Enabling topic whitelist should succeed")

	// Disable whitelist for topic
	disableMsg := &types.DisableTopicWorkerWhitelistRequest{
		Sender:  adminAddr,
		TopicId: topicId,
	}
	_, err = msgServer.DisableTopicWorkerWhitelist(ctx, disableMsg)
	require.NoError(err, "Disabling topic whitelist should succeed")

	// Verify topic whitelist is disabled
	isEnabled, err := s.emissionsKeeper.IsTopicWorkerWhitelistEnabled(ctx, topicId)
	require.NoError(err, "IsTopicWorkerWhitelistEnabled check should not return an error")
	require.False(isEnabled, "Topic whitelist should be disabled")
}

func (s *MsgServerTestSuite) TestDisableTopicWorkerWhitelistInvalidUnauthorized() {
	ctx := s.ctx
	require := s.Require()
	nonAdminAddr := nonAdminAccounts[0]
	topicId := s.CreateOneTopic().Id

	// Attempt to disable whitelist for topic by nonAdminAddr
	msg := &types.DisableTopicWorkerWhitelistRequest{
		Sender:  nonAdminAddr.String(),
		TopicId: topicId,
	}

	_, err := s.msgServer.DisableTopicWorkerWhitelist(ctx, msg)
	require.ErrorIs(err, types.ErrNotPermittedToUpdateTopicWhitelist, "Should fail due to unauthorized access")
}

func (s *MsgServerTestSuite) TestDisableTopicWorkerWhitelistTopicDoesNotExist() {
	ctx := s.ctx
	require := s.Require()
	nonAdminAddr := nonAdminAccounts[0]
	topicId := uint64(1000)

	// Attempt to disable whitelist for topic by nonAdminAddr
	msg := &types.DisableTopicWorkerWhitelistRequest{
		Sender:  nonAdminAddr.String(),
		TopicId: topicId,
	}

	_, err := s.msgServer.DisableTopicWorkerWhitelist(ctx, msg)
	require.ErrorIs(err, types.ErrTopicDoesNotExist, "Should fail due to topic not existing")
}

func (s *MsgServerTestSuite) TestDisableTopicReputerWhitelist() {
	ctx := s.ctx
	require := s.Require()
	msgServer := s.msgServer
	adminAddr := s.addrsStr[0]
	topicId := s.CreateOneTopic().Id

	// First enable whitelist for topic
	enableMsg := &types.EnableTopicReputerWhitelistRequest{
		Sender:  adminAddr,
		TopicId: topicId,
	}
	_, err := msgServer.EnableTopicReputerWhitelist(ctx, enableMsg)
	require.NoError(err, "Enabling topic whitelist should succeed")

	// Disable whitelist for topic
	disableMsg := &types.DisableTopicReputerWhitelistRequest{
		Sender:  adminAddr,
		TopicId: topicId,
	}
	_, err = msgServer.DisableTopicReputerWhitelist(ctx, disableMsg)
	require.NoError(err, "Disabling topic whitelist should succeed")

	// Verify topic whitelist is disabled
	isEnabled, err := s.emissionsKeeper.IsTopicReputerWhitelistEnabled(ctx, topicId)
	require.NoError(err, "IsTopicReputerWhitelistEnabled check should not return an error")
	require.False(isEnabled, "Topic reputer whitelist should be disabled")
}

func (s *MsgServerTestSuite) TestDisableTopicReputerWhitelistInvalidUnauthorized() {
	ctx := s.ctx
	require := s.Require()
	nonAdminAddr := nonAdminAccounts[0]
	topicId := s.CreateOneTopic().Id

	// Attempt to disable whitelist for topic by nonAdminAddr
	msg := &types.DisableTopicReputerWhitelistRequest{
		Sender:  nonAdminAddr.String(),
		TopicId: topicId,
	}

	_, err := s.msgServer.DisableTopicReputerWhitelist(ctx, msg)
	require.ErrorIs(err, types.ErrNotPermittedToUpdateTopicWhitelist, "Should fail due to unauthorized access")
}

func (s *MsgServerTestSuite) TestDisableTopicReputerWhitelistTopicDoesNotExist() {
	ctx := s.ctx
	require := s.Require()
	nonAdminAddr := nonAdminAccounts[0]
	topicId := uint64(1000)

	// Attempt to disable whitelist for topic by nonAdminAddr
	msg := &types.DisableTopicReputerWhitelistRequest{
		Sender:  nonAdminAddr.String(),
		TopicId: topicId,
	}

	_, err := s.msgServer.DisableTopicReputerWhitelist(ctx, msg)
	require.ErrorIs(err, types.ErrTopicDoesNotExist, "Should fail due to topic not existing")
}

func (s *MsgServerTestSuite) TestAddToTopicWorkerWhitelist() {
	ctx := s.ctx
	require := s.Require()
	adminAddr := s.addrsStr[0]
	topicId := s.CreateOneTopic().Id
	targetAddr := s.addrsStr[1]

	msg := &types.AddToTopicWorkerWhitelistRequest{
		Sender:  adminAddr,
		TopicId: topicId,
		Address: targetAddr,
	}
	_, err := s.msgServer.AddToTopicWorkerWhitelist(ctx, msg)
	require.NoError(err, "Adding to topic worker whitelist should succeed")
}

func (s *MsgServerTestSuite) TestAddToTopicWorkerWhitelistInvalidUnauthorized() {
	ctx := s.ctx
	require := s.Require()
	nonAdminAddr := nonAdminAccounts[0]
	topicId := s.CreateOneTopic().Id
	targetAddr := s.addrsStr[1]

	msg := &types.AddToTopicWorkerWhitelistRequest{
		Sender:  nonAdminAddr.String(),
		TopicId: topicId,
		Address: targetAddr,
	}

	_, err := s.msgServer.AddToTopicWorkerWhitelist(ctx, msg)
	require.ErrorIs(err, types.ErrNotPermittedToUpdateTopicWorkerWhitelist, "Should fail due to unauthorized access")
}

func (s *MsgServerTestSuite) TestAddToTopicWorkerWhitelistTopicDoesNotExist() {
	ctx := s.ctx
	require := s.Require()
	nonAdminAddr := nonAdminAccounts[0]
	topicId := uint64(1000)
	targetAddr := s.addrsStr[1]

	msg := &types.AddToTopicWorkerWhitelistRequest{
		Sender:  nonAdminAddr.String(),
		TopicId: topicId,
		Address: targetAddr,
	}

	_, err := s.msgServer.AddToTopicWorkerWhitelist(ctx, msg)
	require.ErrorIs(err, types.ErrTopicDoesNotExist, "Should fail due to topic not existing")
}

func (s *MsgServerTestSuite) TestRemoveFromTopicWorkerWhitelist() {
	ctx := s.ctx
	require := s.Require()
	adminAddr := s.addrsStr[0]
	topicId := s.CreateOneTopic().Id
	targetAddr := s.addrsStr[1]

	msg := &types.RemoveFromTopicWorkerWhitelistRequest{
		Sender:  adminAddr,
		TopicId: topicId,
		Address: targetAddr,
	}
	_, err := s.msgServer.RemoveFromTopicWorkerWhitelist(ctx, msg)
	require.NoError(err, "Removing from topic worker whitelist should succeed")
}

func (s *MsgServerTestSuite) TestRemoveFromTopicWorkerWhitelistTopicDoesNotExist() {
	ctx := s.ctx
	require := s.Require()
	nonAdminAddr := nonAdminAccounts[0]
	topicId := uint64(1000)
	targetAddr := s.addrsStr[1]

	msg := &types.RemoveFromTopicWorkerWhitelistRequest{
		Sender:  nonAdminAddr.String(),
		TopicId: topicId,
		Address: targetAddr,
	}

	_, err := s.msgServer.RemoveFromTopicWorkerWhitelist(ctx, msg)
	require.ErrorIs(err, types.ErrTopicDoesNotExist, "Should fail due to topic not existing")
}

func (s *MsgServerTestSuite) TestAddToTopicReputerWhitelist() {
	ctx := s.ctx
	require := s.Require()
	adminAddr := s.addrsStr[0]
	topicId := s.CreateOneTopic().Id
	targetAddr := s.addrsStr[1]

	msg := &types.AddToTopicReputerWhitelistRequest{
		Sender:  adminAddr,
		TopicId: topicId,
		Address: targetAddr,
	}
	_, err := s.msgServer.AddToTopicReputerWhitelist(ctx, msg)
	require.NoError(err, "Adding to topic reputer whitelist should succeed")
}

func (s *MsgServerTestSuite) TestAddToTopicReputerWhitelistInvalidUnauthorized() {
	ctx := s.ctx
	require := s.Require()
	nonAdminAddr := nonAdminAccounts[0]
	topicId := s.CreateOneTopic().Id
	targetAddr := s.addrsStr[1]

	msg := &types.AddToTopicReputerWhitelistRequest{
		Sender:  nonAdminAddr.String(),
		TopicId: topicId,
		Address: targetAddr,
	}

	_, err := s.msgServer.AddToTopicReputerWhitelist(ctx, msg)
	require.ErrorIs(err, types.ErrNotPermittedToUpdateTopicReputerWhitelist, "Should fail due to unauthorized access")
}

func (s *MsgServerTestSuite) TestAddToTopicReputerWhitelistTopicDoesNotExist() {
	ctx := s.ctx
	require := s.Require()
	nonAdminAddr := nonAdminAccounts[0]
	topicId := uint64(1000)
	targetAddr := s.addrsStr[1]

	msg := &types.AddToTopicReputerWhitelistRequest{
		Sender:  nonAdminAddr.String(),
		TopicId: topicId,
		Address: targetAddr,
	}

	_, err := s.msgServer.AddToTopicReputerWhitelist(ctx, msg)
	require.ErrorIs(err, types.ErrTopicDoesNotExist, "Should fail due to topic not existing")
}

func (s *MsgServerTestSuite) TestRemoveFromTopicReputerWhitelist() {
	ctx := s.ctx
	require := s.Require()
	adminAddr := s.addrsStr[0]
	topicId := s.CreateOneTopic().Id
	targetAddr := s.addrsStr[1]

	msg := &types.RemoveFromTopicReputerWhitelistRequest{
		Sender:  adminAddr,
		TopicId: topicId,
		Address: targetAddr,
	}
	_, err := s.msgServer.RemoveFromTopicReputerWhitelist(ctx, msg)
	require.NoError(err, "Removing from topic reputer whitelist should succeed")
}

func (s *MsgServerTestSuite) TestRemoveFromTopicReputerWhitelistTopicDoesNotExist() {
	ctx := s.ctx
	require := s.Require()
	nonAdminAddr := nonAdminAccounts[0]
	topicId := uint64(1000)
	targetAddr := s.addrsStr[1]

	msg := &types.RemoveFromTopicReputerWhitelistRequest{
		Sender:  nonAdminAddr.String(),
		TopicId: topicId,
		Address: targetAddr,
	}

	_, err := s.msgServer.RemoveFromTopicReputerWhitelist(ctx, msg)
	require.ErrorIs(err, types.ErrTopicDoesNotExist, "Should fail due to topic not existing")
}
