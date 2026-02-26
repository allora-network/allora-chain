package keeper_test

import (
	"context"

	"github.com/allora-network/allora-chain/x/emissions/testutil"
)

//nolint:exhaustruct
func (s *KeeperTestSuite) TestWhitelistOperations() {
	ctx := s.Ctx()
	k := s.WhitelistsKeeper()
	topicId := uint64(1)
	address := "allo1wmvlvr82nlnu2y6hewgjwex30spyqgzvjhc80h"
	nonExistentAddr := "allo1w6uwgrv77szudkve7g84uazuhyw6j4q9hdqelv"
	invalidAddr := "invalid"

	testCases := []struct {
		name                 string
		setupWhitelist       func(context.Context, string) error
		setupTopicWhitelist  func(context.Context, uint64, string) error
		checkWhitelist       func(context.Context, string) (bool, error)
		checkTopicWhitelist  func(context.Context, uint64, string) (bool, error)
		removeWhitelist      func(context.Context, string) error
		removeTopicWhitelist func(context.Context, uint64, string) error
		needsTopicId         bool
	}{
		{
			name:            "Admin whitelist operations",
			setupWhitelist:  k.AddWhitelistAdmin,
			checkWhitelist:  k.IsWhitelistAdmin,
			removeWhitelist: k.RemoveWhitelistAdmin,
		},
		{
			name:            "Global whitelist operations",
			setupWhitelist:  k.AddToGlobalWhitelist,
			checkWhitelist:  k.IsWhitelistedGlobalActor,
			removeWhitelist: k.RemoveFromGlobalWhitelist,
		},
		{
			name:            "Topic creator whitelist operations",
			setupWhitelist:  k.AddToTopicCreatorWhitelist,
			checkWhitelist:  k.IsWhitelistedTopicCreator,
			removeWhitelist: k.RemoveFromTopicCreatorWhitelist,
		},
		{
			name:                 "Topic worker whitelist operations",
			setupTopicWhitelist:  k.AddToTopicWorkerWhitelist,
			checkTopicWhitelist:  k.IsWhitelistedTopicWorker,
			removeTopicWhitelist: k.RemoveFromTopicWorkerWhitelist,
			needsTopicId:         true,
		},
		{
			name:                 "Topic reputer whitelist operations",
			setupTopicWhitelist:  k.AddToTopicReputerWhitelist,
			checkTopicWhitelist:  k.IsWhitelistedTopicReputer,
			removeTopicWhitelist: k.RemoveFromTopicReputerWhitelist,
			needsTopicId:         true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Test basic operations
			var err error
			if tc.setupTopicWhitelist != nil {
				err = tc.setupTopicWhitelist(ctx, topicId, address)
			} else {
				err = tc.setupWhitelist(ctx, address)
			}
			s.Require().NoError(err)

			var isWhitelisted bool
			if tc.checkTopicWhitelist != nil {
				isWhitelisted, err = tc.checkTopicWhitelist(ctx, topicId, address)
			} else {
				isWhitelisted, err = tc.checkWhitelist(ctx, address)
			}
			s.Require().NoError(err)
			s.Require().True(isWhitelisted)

			// Test removal
			if tc.removeTopicWhitelist != nil {
				err = tc.removeTopicWhitelist(ctx, topicId, address)
			} else {
				err = tc.removeWhitelist(ctx, address)
			}
			s.Require().NoError(err)

			// Verify removal
			if tc.checkTopicWhitelist != nil {
				isWhitelisted, err = tc.checkTopicWhitelist(ctx, topicId, address)
			} else {
				isWhitelisted, err = tc.checkWhitelist(ctx, address)
			}
			s.Require().NoError(err)
			s.Require().False(isWhitelisted)

			// Test removing non-existent entry (should not error)
			if tc.removeTopicWhitelist != nil {
				err = tc.removeTopicWhitelist(ctx, topicId, nonExistentAddr)
			} else {
				err = tc.removeWhitelist(ctx, nonExistentAddr)
			}
			s.Require().NoError(err)

			// Test invalid address (should error)
			if tc.setupTopicWhitelist != nil {
				err = tc.setupTopicWhitelist(ctx, topicId, invalidAddr)
			} else {
				err = tc.setupWhitelist(ctx, invalidAddr)
			}
			s.Require().Error(err)
			s.Require().Contains(err.Error(), "error validating admin id")

			if tc.removeTopicWhitelist != nil {
				err = tc.removeTopicWhitelist(ctx, topicId, invalidAddr)
			} else {
				err = tc.removeWhitelist(ctx, invalidAddr)
			}
			s.Require().Error(err)
			s.Require().Contains(err.Error(), "error validating admin id")
		})
	}
}

//nolint:exhaustruct
func (s *KeeperTestSuite) TestWhitelistEnableDisableOperations() {
	ctx := s.Ctx()
	k := s.WhitelistsKeeper()
	topicId := uint64(2)

	testCases := []struct {
		name    string
		enable  func(context.Context, uint64) error
		check   func(context.Context, uint64) (bool, error)
		disable func(context.Context, uint64) error
	}{
		{
			name:    "Topic worker whitelist enable/disable",
			enable:  k.EnableTopicWorkerWhitelist,
			check:   k.IsTopicWorkerWhitelistEnabled,
			disable: k.DisableTopicWorkerWhitelist,
		},
		{
			name:    "Topic reputer whitelist enable/disable",
			enable:  k.EnableTopicReputerWhitelist,
			check:   k.IsTopicReputerWhitelistEnabled,
			disable: k.DisableTopicReputerWhitelist,
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Initially should be disabled
			enabled, err := tc.check(ctx, topicId)
			s.Require().NoError(err)
			s.Require().False(enabled)

			// Test enabling
			err = tc.enable(ctx, topicId)
			s.Require().NoError(err)

			enabled, err = tc.check(ctx, topicId)
			s.Require().NoError(err)
			s.Require().True(enabled)

			// Test disabling
			err = tc.disable(ctx, topicId)
			s.Require().NoError(err)

			enabled, err = tc.check(ctx, topicId)
			s.Require().NoError(err)
			s.Require().False(enabled)

			// Test disabling when already disabled (should not error)
			err = tc.disable(ctx, topicId)
			s.Require().NoError(err)

			enabled, err = tc.check(ctx, topicId)
			s.Require().NoError(err)
			s.Require().False(enabled)
		})
	}
}

func (s *KeeperTestSuite) TestWhitelistEnabledOperations() {
	ctx := s.Ctx()
	k := s.WhitelistsKeeper()
	address := "allo1wmvlvr82nlnu2y6hewgjwex30spyqgzvjhc80h"

	testCases := []struct {
		name         string
		setupAddr    func(context.Context, string) error
		checkEnabled func(context.Context, string) (bool, error)
		cleanupAddr  func(context.Context, string) error
	}{
		{
			name:         "IsEnabledGlobalActor",
			setupAddr:    k.AddToGlobalWhitelist,
			checkEnabled: k.IsEnabledGlobalActor,
			cleanupAddr:  k.RemoveFromGlobalWhitelist,
		},
		{
			name:         "IsEnabledWhitelistedTopicCreator",
			setupAddr:    k.AddToTopicCreatorWhitelist,
			checkEnabled: k.IsEnabledWhitelistedTopicCreator,
			cleanupAddr:  k.RemoveFromTopicCreatorWhitelist,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Initially should be false (not whitelisted)
			enabled, err := tc.checkEnabled(ctx, address)
			s.Require().NoError(err)
			s.Require().False(enabled)

			// Add to whitelist
			err = tc.setupAddr(ctx, address)
			s.Require().NoError(err)

			// Should now be enabled
			enabled, err = tc.checkEnabled(ctx, address)
			s.Require().NoError(err)
			s.Require().True(enabled)

			// Clean up
			err = tc.cleanupAddr(ctx, address)
			s.Require().NoError(err)
		})
	}
}

func (s *KeeperTestSuite) TestTopicWhitelistEnabledOperations() {
	ctx := s.Ctx()
	k := s.WhitelistsKeeper()
	address := "allo1wmvlvr82nlnu2y6hewgjwex30spyqgzvjhc80h"
	topicId := uint64(1)

	testCases := []struct {
		name           string
		enableList     func(context.Context, uint64) error
		disableList    func(context.Context, uint64) error
		addToList      func(context.Context, uint64, string) error
		removeFromList func(context.Context, uint64, string) error
		checkEnabled   func(context.Context, uint64, string) (bool, error)
	}{
		{
			name:           "IsEnabledTopicWorker",
			enableList:     k.EnableTopicWorkerWhitelist,
			disableList:    k.DisableTopicWorkerWhitelist,
			addToList:      k.AddToTopicWorkerWhitelist,
			removeFromList: k.RemoveFromTopicWorkerWhitelist,
			checkEnabled:   k.IsEnabledTopicWorker,
		},
		{
			name:           "IsEnabledTopicReputer",
			enableList:     k.EnableTopicReputerWhitelist,
			disableList:    k.DisableTopicReputerWhitelist,
			addToList:      k.AddToTopicReputerWhitelist,
			removeFromList: k.RemoveFromTopicReputerWhitelist,
			checkEnabled:   k.IsEnabledTopicReputer,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Clean state first
			_ = tc.removeFromList(ctx, topicId, address)
			_ = tc.disableList(ctx, topicId)

			// When whitelist disabled, should be enabled for everyone
			enabled, err := tc.checkEnabled(ctx, topicId, address)
			s.Require().NoError(err)
			s.Require().True(enabled)

			// Enable whitelist
			err = tc.enableList(ctx, topicId)
			s.Require().NoError(err)

			// When whitelist enabled but not in list, should be false
			enabled, err = tc.checkEnabled(ctx, topicId, address)
			s.Require().NoError(err)
			s.Require().False(enabled)

			// Add to whitelist
			err = tc.addToList(ctx, topicId, address)
			s.Require().NoError(err)

			// Should now be enabled
			enabled, err = tc.checkEnabled(ctx, topicId, address)
			s.Require().NoError(err)
			s.Require().True(enabled)

			// Clean up
			_ = tc.removeFromList(ctx, topicId, address)
			_ = tc.disableList(ctx, topicId)
		})
	}
}

func (s *KeeperTestSuite) TestPermissionOperations() {
	ctx := s.Ctx()
	k := s.WhitelistsKeeper()
	address := "allo1wmvlvr82nlnu2y6hewgjwex30spyqgzvjhc80h"

	testCases := []struct {
		name         string
		setupAdmin   func(context.Context, string) error
		checkPerm    func(context.Context, string) (bool, error)
		cleanupAdmin func(context.Context, string) error
	}{
		{
			name:         "CanUpdateAllGlobalWhitelists",
			setupAdmin:   k.AddWhitelistAdmin,
			checkPerm:    k.CanUpdateAllGlobalWhitelists,
			cleanupAdmin: k.RemoveWhitelistAdmin,
		},
		{
			name:         "CanUpdateParams",
			setupAdmin:   k.AddWhitelistAdmin,
			checkPerm:    k.CanUpdateParams,
			cleanupAdmin: k.RemoveWhitelistAdmin,
		},
		{
			name:         "CanUpdateTopicCreatorWhitelist",
			setupAdmin:   k.AddWhitelistAdmin,
			checkPerm:    k.CanUpdateTopicCreatorWhitelist,
			cleanupAdmin: k.RemoveWhitelistAdmin,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Initially should be false (not admin)
			canUpdate, err := tc.checkPerm(ctx, address)
			s.Require().NoError(err)
			s.Require().False(canUpdate)

			// Add admin
			err = tc.setupAdmin(ctx, address)
			s.Require().NoError(err)

			// Should now have permission
			canUpdate, err = tc.checkPerm(ctx, address)
			s.Require().NoError(err)
			s.Require().True(canUpdate)

			// Clean up
			err = tc.cleanupAdmin(ctx, address)
			s.Require().NoError(err)
		})
	}
}

func (s *KeeperTestSuite) TestTopicPermissionCanUpdateTopicWhitelist() {
	ctx := s.Ctx()
	k := s.WhitelistsKeeper()
	address := "allo1wmvlvr82nlnu2y6hewgjwex30spyqgzvjhc80h"
	topicId := s.CreateTopic(testutil.WithEpochLength(60))

	// Get topic creator for comparison
	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)

	// Topic creator should have permission
	canUpdate, err := k.CanUpdateTopicWhitelist(ctx, topicId, topic.Creator)
	s.Require().NoError(err)
	s.Require().True(canUpdate)

	// Non-creator should not have permission
	canUpdate, err = k.CanUpdateTopicWhitelist(ctx, topicId, address)
	s.Require().NoError(err)
	s.Require().False(canUpdate)

	// Admin should have permission
	err = k.AddWhitelistAdmin(ctx, address)
	s.Require().NoError(err)

	canUpdate, err = k.CanUpdateTopicWhitelist(ctx, topicId, address)
	s.Require().NoError(err)
	s.Require().True(canUpdate)

	// Clean up
	err = k.RemoveWhitelistAdmin(ctx, address)
	s.Require().NoError(err)
}

//nolint:exhaustruct
func (s *KeeperTestSuite) TestWhitelistBasedPermissions() {
	ctx := s.Ctx()
	k := s.WhitelistsKeeper()
	topicCreator := s.AddrsStr(0)
	topicWorker := s.AddrsStr(1)
	topicReputer := s.AddrsStr(2)
	globalActor := s.AddrsStr(3)
	nonWhitelistedAddr := "allo1b6uwgrv77szudkve7g84uazuhyw6j4q9hdqely"
	topicId := uint64(2)

	testCases := []struct {
		name             string
		setupWhitelist   func(context.Context, string) error
		setupWhitelist1  func(context.Context, uint64, string) error
		enableWhitelist  func(context.Context, uint64) error
		disableWhitelist func(context.Context, uint64) error
		checkPermission  func(context.Context, uint64, string) (bool, error)
		checkPermission1 func(context.Context, string) (bool, error) // For CanCreateTopic
		whitelistedAddr  string
		hasEnableDisable bool
	}{
		{
			name:             "CanCreateTopic",
			setupWhitelist:   k.AddToTopicCreatorWhitelist,
			checkPermission1: k.CanCreateTopic,
			whitelistedAddr:  topicCreator,
			hasEnableDisable: false,
		},
		{
			name:             "CanSubmitWorkerPayload",
			setupWhitelist1:  k.AddToTopicWorkerWhitelist,
			enableWhitelist:  k.EnableTopicWorkerWhitelist,
			disableWhitelist: k.DisableTopicWorkerWhitelist,
			checkPermission:  k.CanSubmitWorkerPayload,
			whitelistedAddr:  topicWorker,
			hasEnableDisable: true,
		},
		{
			name:             "CanSubmitReputerPayload",
			setupWhitelist1:  k.AddToTopicReputerWhitelist,
			enableWhitelist:  k.EnableTopicReputerWhitelist,
			disableWhitelist: k.DisableTopicReputerWhitelist,
			checkPermission:  k.CanSubmitReputerPayload,
			whitelistedAddr:  topicReputer,
			hasEnableDisable: true,
		},
		{
			name:             "CanAddReputerStake",
			setupWhitelist1:  k.AddToTopicReputerWhitelist,
			enableWhitelist:  k.EnableTopicReputerWhitelist,
			disableWhitelist: k.DisableTopicReputerWhitelist,
			checkPermission:  k.CanAddReputerStake,
			whitelistedAddr:  topicReputer,
			hasEnableDisable: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup whitelists
			if tc.setupWhitelist != nil {
				err := tc.setupWhitelist(ctx, tc.whitelistedAddr)
				s.Require().NoError(err)
			} else {
				err := tc.setupWhitelist1(ctx, topicId, tc.whitelistedAddr)
				s.Require().NoError(err)
			}

			if tc.hasEnableDisable {
				err := k.AddToGlobalWhitelist(ctx, globalActor)
				s.Require().NoError(err)
				// When whitelist disabled, all should have permission
				can, err := tc.checkPermission(ctx, topicId, nonWhitelistedAddr)
				s.Require().NoError(err)
				s.Require().True(can)

				// Enable whitelist
				err = tc.enableWhitelist(ctx, topicId)
				s.Require().NoError(err)

				// Non-whitelisted should not have permission
				can, err = tc.checkPermission(ctx, topicId, nonWhitelistedAddr)
				s.Require().NoError(err)
				s.Require().False(can)

				// Whitelisted actor should have permission
				can, err = tc.checkPermission(ctx, topicId, tc.whitelistedAddr)
				s.Require().NoError(err)
				s.Require().True(can)

				// Global actor should have permission
				can, err = tc.checkPermission(ctx, topicId, globalActor)
				s.Require().NoError(err)
				s.Require().True(can)

				// Disable whitelist
				err = tc.disableWhitelist(ctx, topicId)
				s.Require().NoError(err)

				// All should have permission again
				can, err = tc.checkPermission(ctx, topicId, nonWhitelistedAddr)
				s.Require().NoError(err)
				s.Require().True(can)
			} else {
				// For topic creator (no enable/disable), non-whitelisted should not have permission
				can, err := tc.checkPermission1(ctx, nonWhitelistedAddr)
				s.Require().NoError(err)
				s.Require().False(can)

				// Whitelisted should have permission
				can, err = tc.checkPermission1(ctx, tc.whitelistedAddr)
				s.Require().NoError(err)
				s.Require().True(can)
			}
		})
	}
}
