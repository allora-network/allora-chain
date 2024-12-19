package keeper_test

func (s *KeeperTestSuite) TestWhitelistAdminOperations() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	adminAddress := "allo1wmvlvr82nlnu2y6hewgjwex30spyqgzvjhc80h"

	// Test Adding to whitelist
	err := keeper.AddWhitelistAdmin(ctx, adminAddress)
	s.Require().NoError(err, "Adding whitelist admin should not fail")

	// Test Checking whitelist
	isAdmin, err := keeper.IsWhitelistAdmin(ctx, adminAddress)
	s.Require().NoError(err, "Checking if address is an admin should not fail")
	s.Require().True(isAdmin, "Address should be an admin after being added")

	// Test Removing from whitelist
	err = keeper.RemoveWhitelistAdmin(ctx, adminAddress)
	s.Require().NoError(err, "Removing whitelist admin should not fail")

	// Verify removal
	isAdmin, err = keeper.IsWhitelistAdmin(ctx, adminAddress)
	s.Require().NoError(err, "Checking admin status after removal should not fail")
	s.Require().False(isAdmin, "Address should not be an admin after being removed")

	// Test invalid address
	invalidAddr := "invalid"
	err = keeper.AddWhitelistAdmin(ctx, invalidAddr)
	s.Require().Error(err)
	err = keeper.RemoveWhitelistAdmin(ctx, invalidAddr)
	s.Require().Error(err)
}

func (s *KeeperTestSuite) TestGlobalWhitelistOperations() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	testAddr := "allo1wmvlvr82nlnu2y6hewgjwex30spyqgzvjhc80h"

	// Test global whitelist operations
	isWhitelisted, err := keeper.IsWhitelistedGlobalActor(ctx, testAddr)
	s.Require().NoError(err)
	s.Require().False(isWhitelisted)

	err = keeper.AddToGlobalWhitelist(ctx, testAddr)
	s.Require().NoError(err)

	isWhitelisted, err = keeper.IsWhitelistedGlobalActor(ctx, testAddr)
	s.Require().NoError(err)
	s.Require().True(isWhitelisted)

	err = keeper.RemoveFromGlobalWhitelist(ctx, testAddr)
	s.Require().NoError(err)

	// Test invalid address
	invalidAddr := "invalid"
	err = keeper.AddToGlobalWhitelist(ctx, invalidAddr)
	s.Require().Error(err)
	err = keeper.RemoveFromGlobalWhitelist(ctx, invalidAddr)
	s.Require().Error(err)
}

func (s *KeeperTestSuite) TestRemoveWhitelistAdmin() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	// Test removing non-existent admin
	nonExistentAdmin := "allo1w6uwgrv77szudkve7g84uazuhyw6j4q9hdqelv"
	err := keeper.RemoveWhitelistAdmin(ctx, nonExistentAdmin)
	s.Require().NoError(err)

	// Test removing existing admin
	admin := "allo14s7gd09y7mkje8547ukm0c8gjnd3hak7v3fwz6"
	err = keeper.AddWhitelistAdmin(ctx, admin)
	s.Require().NoError(err)

	err = keeper.RemoveWhitelistAdmin(ctx, admin)
	s.Require().NoError(err)

	// Verify admin was removed
	has, err := keeper.IsWhitelistAdmin(ctx, admin)
	s.Require().NoError(err)
	s.Require().False(has)

	// Test invalid bech32 address
	invalidAdmin := "invalid-address"
	err = keeper.RemoveWhitelistAdmin(ctx, invalidAdmin)
	s.Require().Error(err)
}

func (s *KeeperTestSuite) TestTopicCreatorWhitelistOperations() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	testAddr := "allo1wmvlvr82nlnu2y6hewgjwex30spyqgzvjhc80h"

	// Test topic creator whitelist operations
	isWhitelisted, err := keeper.IsWhitelistedTopicCreator(ctx, testAddr)
	s.Require().NoError(err)
	s.Require().False(isWhitelisted)

	err = keeper.AddToTopicCreatorWhitelist(ctx, testAddr)
	s.Require().NoError(err)

	isWhitelisted, err = keeper.IsWhitelistedTopicCreator(ctx, testAddr)
	s.Require().NoError(err)
	s.Require().True(isWhitelisted)

	err = keeper.RemoveFromTopicCreatorWhitelist(ctx, testAddr)
	s.Require().NoError(err)

	// Test invalid address
	invalidAddr := "invalid"
	err = keeper.AddToTopicCreatorWhitelist(ctx, invalidAddr)
	s.Require().Error(err)
	err = keeper.RemoveFromTopicCreatorWhitelist(ctx, invalidAddr)
	s.Require().Error(err)
}

func (s *KeeperTestSuite) TestTopicWorkerWhitelistOperations() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	testAddr := "allo1wmvlvr82nlnu2y6hewgjwex30spyqgzvjhc80h"
	topicId := uint64(1)

	// Test topic worker whitelist operations
	isWhitelisted, err := keeper.IsWhitelistedTopicWorker(ctx, topicId, testAddr)
	s.Require().NoError(err)
	s.Require().False(isWhitelisted)

	err = keeper.AddToTopicWorkerWhitelist(ctx, topicId, testAddr)
	s.Require().NoError(err)

	isWhitelisted, err = keeper.IsWhitelistedTopicWorker(ctx, topicId, testAddr)
	s.Require().NoError(err)
	s.Require().True(isWhitelisted)

	err = keeper.RemoveFromTopicWorkerWhitelist(ctx, topicId, testAddr)
	s.Require().NoError(err)

	// Test invalid address
	invalidAddr := "invalid"
	err = keeper.AddToTopicWorkerWhitelist(ctx, topicId, invalidAddr)
	s.Require().Error(err)
	err = keeper.RemoveFromTopicWorkerWhitelist(ctx, topicId, invalidAddr)
	s.Require().Error(err)
}

func (s *KeeperTestSuite) TestTopicReputerWhitelistOperations() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	testAddr := "allo1wmvlvr82nlnu2y6hewgjwex30spyqgzvjhc80h"
	topicId := uint64(1)

	// Test topic reputer whitelist operations
	isWhitelisted, err := keeper.IsWhitelistedTopicReputer(ctx, topicId, testAddr)
	s.Require().NoError(err)
	s.Require().False(isWhitelisted)

	err = keeper.AddToTopicReputerWhitelist(ctx, topicId, testAddr)
	s.Require().NoError(err)

	isWhitelisted, err = keeper.IsWhitelistedTopicReputer(ctx, topicId, testAddr)
	s.Require().NoError(err)
	s.Require().True(isWhitelisted)

	err = keeper.RemoveFromTopicReputerWhitelist(ctx, topicId, testAddr)
	s.Require().NoError(err)

	// Test invalid address
	invalidAddr := "invalid"
	err = keeper.AddToTopicReputerWhitelist(ctx, topicId, invalidAddr)
	s.Require().Error(err)
	err = keeper.RemoveFromTopicReputerWhitelist(ctx, topicId, invalidAddr)
	s.Require().Error(err)
}

func (s *KeeperTestSuite) TestIsTopicWhitelistEnabled() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	enabled, err := keeper.IsTopicWorkerWhitelistEnabled(ctx, topicId)
	s.Require().NoError(err)
	s.Require().False(enabled)

	err = keeper.EnableTopicWorkerWhitelist(ctx, topicId)
	s.Require().NoError(err)

	enabled, err = keeper.IsTopicWorkerWhitelistEnabled(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(enabled)
}

func (s *KeeperTestSuite) TestIsEnabledGlobalActor() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	testAddr := "allo1wmvlvr82nlnu2y6hewgjwex30spyqgzvjhc80h"

	enabled, err := keeper.IsEnabledGlobalActor(ctx, testAddr)
	s.Require().NoError(err)
	s.Require().False(enabled)

	err = keeper.AddToGlobalWhitelist(ctx, testAddr)
	s.Require().NoError(err)

	enabled, err = keeper.IsEnabledGlobalActor(ctx, testAddr)
	s.Require().NoError(err)
	s.Require().True(enabled)
}

func (s *KeeperTestSuite) TestDisableTopicWorkerWhitelist() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	// Test disabling when not enabled
	err := keeper.DisableTopicWorkerWhitelist(ctx, topicId)
	s.Require().NoError(err, "Disabling non-enabled whitelist should not error")

	enabled, err := keeper.IsTopicWorkerWhitelistEnabled(ctx, topicId)
	s.Require().NoError(err)
	s.Require().False(enabled, "Whitelist should remain disabled")

	// Enable whitelist first
	err = keeper.EnableTopicWorkerWhitelist(ctx, topicId)
	s.Require().NoError(err)

	enabled, err = keeper.IsTopicWorkerWhitelistEnabled(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(enabled, "Whitelist should be enabled")

	// Test disabling when enabled
	err = keeper.DisableTopicWorkerWhitelist(ctx, topicId)
	s.Require().NoError(err, "Disabling enabled whitelist should not error")

	enabled, err = keeper.IsTopicWorkerWhitelistEnabled(ctx, topicId)
	s.Require().NoError(err)
	s.Require().False(enabled, "Whitelist should be disabled")
}

func (s *KeeperTestSuite) TestDisableTopicReputerWhitelist() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	// Test disabling when not enabled
	err := keeper.DisableTopicReputerWhitelist(ctx, topicId)
	s.Require().NoError(err, "Disabling non-enabled whitelist should not error")

	enabled, err := keeper.IsTopicReputerWhitelistEnabled(ctx, topicId)
	s.Require().NoError(err)
	s.Require().False(enabled, "Whitelist should remain disabled")

	// Enable whitelist first
	err = keeper.EnableTopicReputerWhitelist(ctx, topicId)
	s.Require().NoError(err)

	enabled, err = keeper.IsTopicReputerWhitelistEnabled(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(enabled, "Whitelist should be enabled")

	// Test disabling when enabled
	err = keeper.DisableTopicReputerWhitelist(ctx, topicId)
	s.Require().NoError(err, "Disabling enabled whitelist should not error")

	enabled, err = keeper.IsTopicReputerWhitelistEnabled(ctx, topicId)
	s.Require().NoError(err)
	s.Require().False(enabled, "Whitelist should be disabled")
}

func (s *KeeperTestSuite) TestRemoveFromGlobalWhitelist() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	// Test removing non-existent actor
	nonExistentActor := "allo1w6uwgrv77szudkve7g84uazuhyw6j4q9hdqelv"
	err := keeper.RemoveFromGlobalWhitelist(ctx, nonExistentActor)
	s.Require().NoError(err)

	// Test removing existing actor
	actor := "allo14s7gd09y7mkje8547ukm0c8gjnd3hak7v3fwz6"
	err = keeper.AddToGlobalWhitelist(ctx, actor)
	s.Require().NoError(err)

	err = keeper.RemoveFromGlobalWhitelist(ctx, actor)
	s.Require().NoError(err)

	// Verify actor was removed
	isWhitelisted, err := keeper.IsWhitelistedGlobalActor(ctx, actor)
	s.Require().NoError(err)
	s.Require().False(isWhitelisted)

	// Test invalid bech32 address
	invalidActor := "invalid-address"
	err = keeper.RemoveFromGlobalWhitelist(ctx, invalidActor)
	s.Require().Error(err)
}

func (s *KeeperTestSuite) TestRemoveFromTopicCreatorWhitelist() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	actor := "allo1w6uwgrv77szudkve7g84uazuhyw6j4q9hdqelv" // Replace with a valid Bech32 actor ID for testing

	// Test case: Actor is not in the whitelist
	err := keeper.RemoveFromTopicCreatorWhitelist(ctx, actor)
	s.Require().NoError(err, "Expected no error when actor is not in the whitelist")

	// Test case: Actor is in the whitelist
	err = keeper.AddToTopicCreatorWhitelist(ctx, actor)
	s.Require().NoError(err)

	err = keeper.RemoveFromTopicCreatorWhitelist(ctx, actor)
	s.Require().NoError(err)

	// Verify actor is removed
	isWhitelisted, err := keeper.IsWhitelistedTopicCreator(ctx, actor)
	s.Require().NoError(err)
	s.Require().False(isWhitelisted)

	// Test case: Invalid Bech32 actor ID
	invalidActor := "invalidActorId"
	err = keeper.RemoveFromTopicCreatorWhitelist(ctx, invalidActor)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "error validating admin id", "Expected validation error message")
}

func (s *KeeperTestSuite) TestRemoveFromTopicWorkerWhitelist() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	// Test removing non-existent worker
	nonExistentWorker := "allo1w6uwgrv77szudkve7g84uazuhyw6j4q9hdqelv"
	err := keeper.RemoveFromTopicWorkerWhitelist(ctx, topicId, nonExistentWorker)
	s.Require().NoError(err, "Expected no error when worker is not in the whitelist")

	// Test removing existing worker
	worker := "allo14s7gd09y7mkje8547ukm0c8gjnd3hak7v3fwz6"
	err = keeper.AddToTopicWorkerWhitelist(ctx, topicId, worker)
	s.Require().NoError(err)

	err = keeper.RemoveFromTopicWorkerWhitelist(ctx, topicId, worker)
	s.Require().NoError(err)

	// Verify worker was removed
	isWhitelisted, err := keeper.IsWhitelistedTopicWorker(ctx, topicId, worker)
	s.Require().NoError(err)
	s.Require().False(isWhitelisted)

	// Test invalid bech32 address
	invalidWorker := "invalid-address"
	err = keeper.RemoveFromTopicWorkerWhitelist(ctx, topicId, invalidWorker)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "error validating admin id")
}

func (s *KeeperTestSuite) TestRemoveFromTopicReputerWhitelist() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	// Test removing non-existent reputer
	nonExistentReputer := "allo1w6uwgrv77szudkve7g84uazuhyw6j4q9hdqelv"
	err := keeper.RemoveFromTopicReputerWhitelist(ctx, topicId, nonExistentReputer)
	s.Require().NoError(err, "Expected no error when reputer is not in the whitelist")

	// Test removing existing reputer
	reputer := "allo14s7gd09y7mkje8547ukm0c8gjnd3hak7v3fwz6"
	err = keeper.AddToTopicReputerWhitelist(ctx, topicId, reputer)
	s.Require().NoError(err)

	err = keeper.RemoveFromTopicReputerWhitelist(ctx, topicId, reputer)
	s.Require().NoError(err)

	// Verify reputer was removed
	isWhitelisted, err := keeper.IsWhitelistedTopicReputer(ctx, topicId, reputer)
	s.Require().NoError(err)
	s.Require().False(isWhitelisted)

	// Test invalid bech32 address
	invalidReputer := "invalid-address"
	err = keeper.RemoveFromTopicReputerWhitelist(ctx, topicId, invalidReputer)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "error validating admin id")
}

func (s *KeeperTestSuite) TestIsEnabledWhitelistedTopicCreator() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	testAddr := "allo1wmvlvr82nlnu2y6hewgjwex30spyqgzvjhc80h"

	enabled, err := keeper.IsEnabledWhitelistedTopicCreator(ctx, testAddr)
	s.Require().NoError(err)
	s.Require().False(enabled)

	err = keeper.AddToTopicCreatorWhitelist(ctx, testAddr)
	s.Require().NoError(err)

	enabled, err = keeper.IsEnabledWhitelistedTopicCreator(ctx, testAddr)
	s.Require().NoError(err)
	s.Require().True(enabled)
}

func (s *KeeperTestSuite) TestIsEnabledTopicWorker() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	testAddr := "allo1wmvlvr82nlnu2y6hewgjwex30spyqgzvjhc80h"
	topicId := uint64(1)

	err := keeper.RemoveFromTopicWorkerWhitelist(ctx, topicId, testAddr)
	s.Require().NoError(err)

	err = keeper.DisableTopicWorkerWhitelist(ctx, topicId)
	s.Require().NoError(err)

	enabled, err := keeper.IsEnabledTopicWorker(ctx, topicId, testAddr)
	s.Require().NoError(err)
	s.Require().True(enabled)

	err = keeper.EnableTopicWorkerWhitelist(ctx, topicId)
	s.Require().NoError(err)

	enabled, err = keeper.IsEnabledTopicWorker(ctx, topicId, testAddr)
	s.Require().NoError(err)
	s.Require().False(enabled)

	err = keeper.AddToTopicWorkerWhitelist(ctx, topicId, testAddr)
	s.Require().NoError(err)

	enabled, err = keeper.IsEnabledTopicWorker(ctx, topicId, testAddr)
	s.Require().NoError(err)
	s.Require().True(enabled)
}

func (s *KeeperTestSuite) TestIsEnabledTopicReputer() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	testAddr := "allo1wmvlvr82nlnu2y6hewgjwex30spyqgzvjhc80h"
	topicId := uint64(1)

	err := keeper.RemoveFromTopicReputerWhitelist(ctx, topicId, testAddr)
	s.Require().NoError(err)

	err = keeper.DisableTopicReputerWhitelist(ctx, topicId)
	s.Require().NoError(err)

	enabled, err := keeper.IsEnabledTopicReputer(ctx, topicId, testAddr)
	s.Require().NoError(err)
	s.Require().True(enabled)

	err = keeper.EnableTopicReputerWhitelist(ctx, topicId)
	s.Require().NoError(err)

	enabled, err = keeper.IsEnabledTopicReputer(ctx, topicId, testAddr)
	s.Require().NoError(err)
	s.Require().False(enabled)

	err = keeper.AddToTopicReputerWhitelist(ctx, topicId, testAddr)
	s.Require().NoError(err)

	enabled, err = keeper.IsEnabledTopicReputer(ctx, topicId, testAddr)
	s.Require().NoError(err)
	s.Require().True(enabled)
}

func (s *KeeperTestSuite) TestCanUpdateGlobalWhitelists() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	testAddr := "allo1wmvlvr82nlnu2y6hewgjwex30spyqgzvjhc80h"

	can, err := keeper.CanUpdateAllGlobalWhitelists(ctx, testAddr)
	s.Require().NoError(err)
	s.Require().False(can)

	err = keeper.AddWhitelistAdmin(ctx, testAddr)
	s.Require().NoError(err)

	can, err = keeper.CanUpdateAllGlobalWhitelists(ctx, testAddr)
	s.Require().NoError(err)
	s.Require().True(can)
}

func (s *KeeperTestSuite) TestCanUpdateParams() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	testAddr := "allo1wmvlvr82nlnu2y6hewgjwex30spyqgzvjhc80h"

	can, err := keeper.CanUpdateParams(ctx, testAddr)
	s.Require().NoError(err)
	s.Require().False(can)

	err = keeper.AddWhitelistAdmin(ctx, testAddr)
	s.Require().NoError(err)

	can, err = keeper.CanUpdateParams(ctx, testAddr)
	s.Require().NoError(err)
	s.Require().True(can)
}

func (s *KeeperTestSuite) TestCanUpdateTopicWhitelist() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	testAddr := "allo1wmvlvr82nlnu2y6hewgjwex30spyqgzvjhc80h"
	topicId := s.CreateOneTopic(60)
	topic, err := keeper.GetTopic(ctx, topicId)
	s.Require().NoError(err)

	can, err := keeper.CanUpdateTopicWhitelist(ctx, topicId, topic.Creator)
	s.Require().NoError(err)
	s.Require().True(can)

	can, err = keeper.CanUpdateTopicWhitelist(ctx, topicId, testAddr)
	s.Require().NoError(err)
	s.Require().False(can)

	err = keeper.AddWhitelistAdmin(ctx, testAddr)
	s.Require().NoError(err)

	can, err = keeper.CanUpdateTopicWhitelist(ctx, topicId, testAddr)
	s.Require().NoError(err)
	s.Require().True(can)
}

func (s *KeeperTestSuite) TestCanCreateTopicWithWhitelist() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	testAddr := "allo1wmvlvr82nlnu2y6hewgjwex30spyqgzvjhc80h"

	can, err := keeper.CanCreateTopic(ctx, testAddr)
	s.Require().NoError(err)
	s.Require().False(can)

	err = keeper.AddToTopicCreatorWhitelist(ctx, testAddr)
	s.Require().NoError(err)

	can, err = keeper.CanCreateTopic(ctx, testAddr)
	s.Require().NoError(err)
	s.Require().True(can)
}

func (s *KeeperTestSuite) TestCanSubmitWorkerPayloadWithWhitelist() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	enabledTopicWorker := "allo1wmvlvr82nlnu2y6hewgjwex30spyqgzvjhc80h"
	enabledGlobalActor := "allo14s7gd09y7mkje8547ukm0c8gjnd3hak7v3fwz6"
	neitherAddr := "allo1w6uwgrv77szudkve7g84uazuhyw6j4q9hdqelv"
	topicId := uint64(1)

	err := keeper.EnableTopicWorkerWhitelist(ctx, topicId)
	s.Require().NoError(err)

	can, err := keeper.CanSubmitWorkerPayload(ctx, topicId, neitherAddr)
	s.Require().NoError(err)
	s.Require().False(can)

	err = keeper.AddToTopicWorkerWhitelist(ctx, topicId, enabledTopicWorker)
	s.Require().NoError(err)

	can, err = keeper.CanSubmitWorkerPayload(ctx, topicId, enabledTopicWorker)
	s.Require().NoError(err)
	s.Require().True(can)

	err = keeper.AddToGlobalWhitelist(ctx, enabledGlobalActor)
	s.Require().NoError(err)

	can, err = keeper.CanSubmitWorkerPayload(ctx, topicId, enabledGlobalActor)
	s.Require().NoError(err)
	s.Require().True(can)

	err = keeper.DisableTopicWorkerWhitelist(ctx, topicId)
	s.Require().NoError(err)

	can, err = keeper.CanSubmitWorkerPayload(ctx, topicId, neitherAddr)
	s.Require().NoError(err)
	s.Require().True(can)
}

func (s *KeeperTestSuite) TestCanSubmitReputerPayloadWithWhitelist() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	enabledTopicReputer := "allo1wmvlvr82nlnu2y6hewgjwex30spyqgzvjhc80h"
	enabledGlobalActor := "allo14s7gd09y7mkje8547ukm0c8gjnd3hak7v3fwz6"
	neitherAddr := "allo1w6uwgrv77szudkve7g84uazuhyw6j4q9hdqelv"
	topicId := uint64(1)

	err := keeper.EnableTopicReputerWhitelist(ctx, topicId)
	s.Require().NoError(err)

	can, err := keeper.CanSubmitReputerPayload(ctx, topicId, neitherAddr)
	s.Require().NoError(err)
	s.Require().False(can)

	err = keeper.AddToTopicReputerWhitelist(ctx, topicId, enabledTopicReputer)
	s.Require().NoError(err)

	can, err = keeper.CanSubmitReputerPayload(ctx, topicId, enabledTopicReputer)
	s.Require().NoError(err)
	s.Require().True(can)

	err = keeper.AddToGlobalWhitelist(ctx, enabledGlobalActor)
	s.Require().NoError(err)

	can, err = keeper.CanSubmitReputerPayload(ctx, topicId, enabledGlobalActor)
	s.Require().NoError(err)
	s.Require().True(can)

	err = keeper.DisableTopicReputerWhitelist(ctx, topicId)
	s.Require().NoError(err)

	can, err = keeper.CanSubmitReputerPayload(ctx, topicId, neitherAddr)
	s.Require().NoError(err)
	s.Require().True(can)
}

func (s *KeeperTestSuite) TestCanAddReputerStakeWithWhitelist() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	enabledTopicReputer := "allo1wmvlvr82nlnu2y6hewgjwex30spyqgzvjhc80h"
	enabledGlobalActor := "allo14s7gd09y7mkje8547ukm0c8gjnd3hak7v3fwz6"
	neitherAddr := "allo1w6uwgrv77szudkve7g84uazuhyw6j4q9hdqelv"
	topicId := uint64(1)

	err := keeper.EnableTopicReputerWhitelist(ctx, topicId)
	s.Require().NoError(err)

	can, err := keeper.CanAddReputerStake(ctx, topicId, neitherAddr)
	s.Require().NoError(err)
	s.Require().False(can)

	err = keeper.AddToTopicReputerWhitelist(ctx, topicId, enabledTopicReputer)
	s.Require().NoError(err)

	can, err = keeper.CanAddReputerStake(ctx, topicId, enabledTopicReputer)
	s.Require().NoError(err)
	s.Require().True(can)

	err = keeper.AddToGlobalWhitelist(ctx, enabledGlobalActor)
	s.Require().NoError(err)

	can, err = keeper.CanAddReputerStake(ctx, topicId, enabledGlobalActor)
	s.Require().NoError(err)
	s.Require().True(can)

	err = keeper.DisableTopicReputerWhitelist(ctx, topicId)
	s.Require().NoError(err)

	can, err = keeper.CanAddReputerStake(ctx, topicId, neitherAddr)
	s.Require().NoError(err)
	s.Require().True(can)
}

func (s *KeeperTestSuite) TestEnableDisableTopicWorkerWhitelist() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	// Initially should be disabled
	enabled, err := keeper.IsTopicWorkerWhitelistEnabled(ctx, topicId)
	s.Require().NoError(err)
	s.Require().False(enabled)

	// Test enabling
	err = keeper.EnableTopicWorkerWhitelist(ctx, topicId)
	s.Require().NoError(err)

	enabled, err = keeper.IsTopicWorkerWhitelistEnabled(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(enabled)

	// Test disabling
	err = keeper.DisableTopicWorkerWhitelist(ctx, topicId)
	s.Require().NoError(err)

	enabled, err = keeper.IsTopicWorkerWhitelistEnabled(ctx, topicId)
	s.Require().NoError(err)
	s.Require().False(enabled)
}

func (s *KeeperTestSuite) TestEnableDisableTopicReputerWhitelist() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	// Initially should be disabled
	enabled, err := keeper.IsTopicReputerWhitelistEnabled(ctx, topicId)
	s.Require().NoError(err)
	s.Require().False(enabled)

	// Test enabling
	err = keeper.EnableTopicReputerWhitelist(ctx, topicId)
	s.Require().NoError(err)

	enabled, err = keeper.IsTopicReputerWhitelistEnabled(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(enabled)

	// Test disabling
	err = keeper.DisableTopicReputerWhitelist(ctx, topicId)
	s.Require().NoError(err)

	enabled, err = keeper.IsTopicReputerWhitelistEnabled(ctx, topicId)
	s.Require().NoError(err)
	s.Require().False(enabled)
}

func (s *KeeperTestSuite) TestCanUpdateTopicCreatorWhitelist() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	// Test non-admin actor
	nonAdmin := "allo1w6uwgrv77szudkve7g84uazuhyw6j4q9hdqelv"
	canUpdate, err := keeper.CanUpdateTopicCreatorWhitelist(ctx, nonAdmin)
	s.Require().NoError(err)
	s.Require().False(canUpdate)

	// Test admin actor
	admin := "allo14s7gd09y7mkje8547ukm0c8gjnd3hak7v3fwz6"
	err = keeper.AddWhitelistAdmin(ctx, admin)
	s.Require().NoError(err)

	canUpdate, err = keeper.CanUpdateTopicCreatorWhitelist(ctx, admin)
	s.Require().NoError(err)
	s.Require().True(canUpdate)
}
