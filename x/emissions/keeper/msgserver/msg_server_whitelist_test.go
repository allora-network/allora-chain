package msgserver_test

import (
	"context"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *MsgServerTestSuite) TestWhitelistAdminOperations() {
	ctx := s.Ctx()
	msgServer := s.EmissionsMsgServer()
	keeper := s.EmissionsKeeper()

	adminAddr := s.AddrsStr(0)
	targetAddr := s.AddrsStr(1)
	nonAdminAddr := nonAdminAccounts[0].String()

	testCases := []struct {
		name          string
		execute       func() error
		verifyResult  func()
		expectedError error
	}{
		// Whitelist Admin Operations
		{
			name: "Add whitelist admin - unauthorized",
			execute: func() error {
				msg := &types.AddToWhitelistAdminRequest{
					Sender:  nonAdminAddr,
					Address: targetAddr,
				}
				_, err := msgServer.AddToWhitelistAdmin(ctx, msg)
				return err
			},
			expectedError: types.ErrNotPermittedToUpdateWhitelistAdmins,
		},
		{
			name: "Remove whitelist admin - unauthorized",
			execute: func() error {
				msg := &types.RemoveFromWhitelistAdminRequest{
					Sender:  nonAdminAddr,
					Address: adminAddr,
				}
				_, err := msgServer.RemoveFromWhitelistAdmin(ctx, msg)
				return err
			},
			expectedError: types.ErrNotPermittedToUpdateWhitelistAdmins,
		},
		{
			name: "Add whitelist admin - success",
			execute: func() error {
				msg := &types.AddToWhitelistAdminRequest{
					Sender:  adminAddr,
					Address: nonAdminAddr,
				}
				_, err := msgServer.AddToWhitelistAdmin(ctx, msg)
				return err
			},
			verifyResult: func() {
				isAdmin, err := keeper.IsWhitelistAdmin(ctx, nonAdminAddr)
				s.Require().NoError(err)
				s.Require().True(isAdmin)
			},
		},
		{
			name: "Remove whitelist admin - success",
			execute: func() error {
				msg := &types.RemoveFromWhitelistAdminRequest{
					Sender:  adminAddr,
					Address: targetAddr,
				}
				_, err := msgServer.RemoveFromWhitelistAdmin(ctx, msg)
				return err
			},
			verifyResult: func() {
				isAdmin, err := keeper.IsWhitelistAdmin(ctx, targetAddr)
				s.Require().NoError(err)
				s.Require().False(isAdmin)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.execute()

			if tc.expectedError != nil {
				s.Require().ErrorIs(err, tc.expectedError)
			} else {
				s.Require().NoError(err)
				if tc.verifyResult != nil {
					tc.verifyResult()
				}
			}
		})
	}
}

func (s *MsgServerTestSuite) TestGlobalWhitelistOperations() {
	ctx := s.Ctx()
	msgServer := s.EmissionsMsgServer()
	keeper := s.EmissionsKeeper()

	adminAddr := s.AddrsStr(0)
	targetAddr := s.AddrsStr(1)
	nonAdminAddr := nonAdminAccounts[0].String()

	testCases := []struct {
		name          string
		whitelistType string // "global", "worker", "reputer", "admin"
		operation     string // "add" or "remove"
		setupMsg      func() any
		execute       func(any) error
		verify        func(context.Context, string) (bool, error)
		expectedError error
	}{
		// Global Whitelist
		{
			name:          "Add to global whitelist - success",
			whitelistType: "global",
			operation:     "add",
			setupMsg: func() any {
				return &types.AddToGlobalWhitelistRequest{
					Sender:  adminAddr,
					Address: targetAddr,
				}
			},
			execute: func(msg any) error {
				_, err := msgServer.AddToGlobalWhitelist(ctx, msg.(*types.AddToGlobalWhitelistRequest))
				return err
			},
			verify: keeper.IsWhitelistedGlobalActor,
		},
		{
			name:          "Add to global whitelist - unauthorized",
			whitelistType: "global",
			operation:     "add",
			setupMsg: func() any {
				return &types.AddToGlobalWhitelistRequest{
					Sender:  nonAdminAddr,
					Address: targetAddr,
				}
			},
			execute: func(msg any) error {
				_, err := msgServer.AddToGlobalWhitelist(ctx, msg.(*types.AddToGlobalWhitelistRequest))
				return err
			},
			expectedError: types.ErrNotPermittedToUpdateGlobalWhitelist,
		},
		{
			name:          "Remove from global whitelist - success",
			whitelistType: "global",
			operation:     "remove",
			setupMsg: func() any {
				// First add to whitelist
				addMsg := &types.AddToGlobalWhitelistRequest{
					Sender:  adminAddr,
					Address: targetAddr,
				}
				_, err := msgServer.AddToGlobalWhitelist(ctx, addMsg)
				s.Require().NoError(err)

				return &types.RemoveFromGlobalWhitelistRequest{
					Sender:  adminAddr,
					Address: targetAddr,
				}
			},
			execute: func(msg any) error {
				_, err := msgServer.RemoveFromGlobalWhitelist(ctx, msg.(*types.RemoveFromGlobalWhitelistRequest))
				return err
			},
			verify: keeper.IsWhitelistedGlobalActor,
		},
		{
			name:          "Remove from global whitelist - unauthorized",
			whitelistType: "global",
			operation:     "remove",
			setupMsg: func() any {
				return &types.RemoveFromGlobalWhitelistRequest{
					Sender:  nonAdminAddr,
					Address: targetAddr,
				}
			},
			execute: func(msg any) error {
				_, err := msgServer.RemoveFromGlobalWhitelist(ctx, msg.(*types.RemoveFromGlobalWhitelistRequest))
				return err
			},
			expectedError: types.ErrNotPermittedToUpdateGlobalWhitelist,
		},

		// Global Worker Whitelist
		{
			name:          "Add to global worker whitelist - success",
			whitelistType: "worker",
			operation:     "add",
			setupMsg: func() any {
				return &types.AddToGlobalWorkerWhitelistRequest{
					Sender:  adminAddr,
					Address: targetAddr,
				}
			},
			execute: func(msg any) error {
				_, err := msgServer.AddToGlobalWorkerWhitelist(ctx, msg.(*types.AddToGlobalWorkerWhitelistRequest))
				return err
			},
			verify: keeper.IsWhitelistedGlobalWorker,
		},
		{
			name:          "Add to global worker whitelist - unauthorized",
			whitelistType: "worker",
			operation:     "add",
			setupMsg: func() any {
				return &types.AddToGlobalWorkerWhitelistRequest{
					Sender:  nonAdminAddr,
					Address: targetAddr,
				}
			},
			execute: func(msg any) error {
				_, err := msgServer.AddToGlobalWorkerWhitelist(ctx, msg.(*types.AddToGlobalWorkerWhitelistRequest))
				return err
			},
			expectedError: types.ErrNotPermittedToUpdateGlobalWhitelist,
		},
		{
			name:          "Remove from global worker whitelist - success",
			whitelistType: "worker",
			operation:     "remove",
			setupMsg: func() any {
				// First add to whitelist
				addMsg := &types.AddToGlobalWorkerWhitelistRequest{
					Sender:  adminAddr,
					Address: targetAddr,
				}
				_, err := msgServer.AddToGlobalWorkerWhitelist(ctx, addMsg)
				s.Require().NoError(err)

				return &types.RemoveFromGlobalWorkerWhitelistRequest{
					Sender:  adminAddr,
					Address: targetAddr,
				}
			},
			execute: func(msg any) error {
				_, err := msgServer.RemoveFromGlobalWorkerWhitelist(ctx, msg.(*types.RemoveFromGlobalWorkerWhitelistRequest))
				return err
			},
			verify: keeper.IsWhitelistedGlobalWorker,
		},

		// Global Reputer Whitelist
		{
			name:          "Add to global reputer whitelist - success",
			whitelistType: "reputer",
			operation:     "add",
			setupMsg: func() any {
				return &types.AddToGlobalReputerWhitelistRequest{
					Sender:  adminAddr,
					Address: targetAddr,
				}
			},
			execute: func(msg any) error {
				_, err := msgServer.AddToGlobalReputerWhitelist(ctx, msg.(*types.AddToGlobalReputerWhitelistRequest))
				return err
			},
			verify: keeper.IsWhitelistedGlobalReputer,
		},
		{
			name:          "Remove from global reputer whitelist - success",
			whitelistType: "reputer",
			operation:     "remove",
			setupMsg: func() any {
				// First add to whitelist
				addMsg := &types.AddToGlobalReputerWhitelistRequest{
					Sender:  adminAddr,
					Address: targetAddr,
				}
				_, err := msgServer.AddToGlobalReputerWhitelist(ctx, addMsg)
				s.Require().NoError(err)

				return &types.RemoveFromGlobalReputerWhitelistRequest{
					Sender:  adminAddr,
					Address: targetAddr,
				}
			},
			execute: func(msg any) error {
				_, err := msgServer.RemoveFromGlobalReputerWhitelist(ctx, msg.(*types.RemoveFromGlobalReputerWhitelistRequest))
				return err
			},
			verify: keeper.IsWhitelistedGlobalReputer,
		},

		// Global Admin Whitelist
		{
			name:          "Add to global admin whitelist - success",
			whitelistType: "admin",
			operation:     "add",
			setupMsg: func() any {
				return &types.AddToGlobalAdminWhitelistRequest{
					Sender:  adminAddr,
					Address: targetAddr,
				}
			},
			execute: func(msg any) error {
				_, err := msgServer.AddToGlobalAdminWhitelist(ctx, msg.(*types.AddToGlobalAdminWhitelistRequest))
				return err
			},
			verify: keeper.CanUpdateAllGlobalWhitelists,
		},
		{
			name:          "Remove from global admin whitelist - success",
			whitelistType: "admin",
			operation:     "remove",
			setupMsg: func() any {
				// First add to whitelist
				addMsg := &types.AddToGlobalAdminWhitelistRequest{
					Sender:  adminAddr,
					Address: targetAddr,
				}
				_, err := msgServer.AddToGlobalAdminWhitelist(ctx, addMsg)
				s.Require().NoError(err)

				return &types.RemoveFromGlobalAdminWhitelistRequest{
					Sender:  adminAddr,
					Address: targetAddr,
				}
			},
			execute: func(msg any) error {
				_, err := msgServer.RemoveFromGlobalAdminWhitelist(ctx, msg.(*types.RemoveFromGlobalAdminWhitelistRequest))
				return err
			},
			verify: keeper.IsWhitelistedGlobalAdmin,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			msg := tc.setupMsg()
			err := tc.execute(msg)

			if tc.expectedError != nil {
				s.Require().ErrorIs(err, tc.expectedError)
			} else {
				s.Require().NoError(err)
				if tc.verify != nil {
					isWhitelisted, err := tc.verify(ctx, targetAddr)
					s.Require().NoError(err)
					if tc.operation == "add" {
						s.Require().True(isWhitelisted)
					} else {
						s.Require().False(isWhitelisted)
					}
				}
			}
		})
	}
}

func (s *MsgServerTestSuite) TestBulkWhitelistOperations() {
	ctx := s.Ctx()
	msgServer := s.EmissionsMsgServer()
	keeper := s.EmissionsKeeper()
	adminAddr := s.AddrsStr(0)

	addresses := []string{
		nonAdminAccounts[0].String(),
		nonAdminAccounts[1].String(),
		nonAdminAccounts[2].String(),
	}

	// Set max array length for testing
	params, err := keeper.GetParams(ctx)
	s.Require().NoError(err)
	params.MaxWhitelistInputArrayLength = 3
	err = keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	tooManyAddresses := make([]string, params.MaxWhitelistInputArrayLength+1)
	for i := range tooManyAddresses {
		tooManyAddresses[i] = nonAdminAccounts[i].String()
	}

	testCases := []struct {
		name          string
		whitelistType string // "worker", "reputer"
		isGlobal      bool
		operation     string // "add" or "remove"
		addresses     []string
		topicId       uint64
		execute       func() error
		verify        func(context.Context, string) (bool, error)
		expectedError error
	}{
		// Global Worker Bulk Operations
		{
			name:          "Bulk add to global worker whitelist - success",
			whitelistType: "worker",
			isGlobal:      true,
			operation:     "add",
			addresses:     addresses,
			execute: func() error {
				msg := &types.BulkAddToGlobalWorkerWhitelistRequest{
					Sender:    adminAddr,
					Addresses: addresses,
				}
				_, err := msgServer.BulkAddToGlobalWorkerWhitelist(ctx, msg)
				return err
			},
			verify: keeper.IsWhitelistedGlobalWorker,
		},
		{
			name:          "Bulk add to global worker whitelist - too many addresses",
			whitelistType: "worker",
			isGlobal:      true,
			operation:     "add",
			addresses:     tooManyAddresses,
			execute: func() error {
				msg := &types.BulkAddToGlobalWorkerWhitelistRequest{
					Sender:    adminAddr,
					Addresses: tooManyAddresses,
				}
				_, err := msgServer.BulkAddToGlobalWorkerWhitelist(ctx, msg)
				return err
			},
			expectedError: types.ErrMaxWhitelistInputArrayLengthExceeded,
		},
		{
			name:          "Bulk remove from global worker whitelist - success",
			whitelistType: "worker",
			isGlobal:      true,
			operation:     "remove",
			addresses:     addresses,
			execute: func() error {
				// First add addresses
				addMsg := &types.BulkAddToGlobalWorkerWhitelistRequest{
					Sender:    adminAddr,
					Addresses: addresses,
				}
				_, err = msgServer.BulkAddToGlobalWorkerWhitelist(ctx, addMsg)
				s.Require().NoError(err)

				// Then remove
				msg := &types.BulkRemoveFromGlobalWorkerWhitelistRequest{
					Sender:    adminAddr,
					Addresses: addresses,
				}
				_, err = msgServer.BulkRemoveFromGlobalWorkerWhitelist(ctx, msg)
				return err
			},
			verify: keeper.IsWhitelistedGlobalWorker,
		},

		// Global Reputer Bulk Operations
		{
			name:          "Bulk add to global reputer whitelist - success",
			whitelistType: "reputer",
			isGlobal:      true,
			operation:     "add",
			addresses:     addresses,
			execute: func() error {
				msg := &types.BulkAddToGlobalReputerWhitelistRequest{
					Sender:    adminAddr,
					Addresses: addresses,
				}
				_, err := msgServer.BulkAddToGlobalReputerWhitelist(ctx, msg)
				return err
			},
			verify: keeper.IsWhitelistedGlobalReputer,
		},
		{
			name:          "Bulk remove from global reputer whitelist - success",
			whitelistType: "reputer",
			isGlobal:      true,
			operation:     "remove",
			addresses:     addresses,
			execute: func() error {
				// First add addresses
				addMsg := &types.BulkAddToGlobalReputerWhitelistRequest{
					Sender:    adminAddr,
					Addresses: addresses,
				}
				_, err = msgServer.BulkAddToGlobalReputerWhitelist(ctx, addMsg)
				s.Require().NoError(err)

				// Then remove
				msg := &types.BulkRemoveFromGlobalReputerWhitelistRequest{
					Sender:    adminAddr,
					Addresses: addresses,
				}
				_, err = msgServer.BulkRemoveFromGlobalReputerWhitelist(ctx, msg)
				return err
			},
			verify: keeper.IsWhitelistedGlobalReputer,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.execute()

			if tc.expectedError != nil {
				s.Require().ErrorIs(err, tc.expectedError)
			} else {
				s.Require().NoError(err)
				if tc.verify != nil {
					for _, addr := range tc.addresses {
						isWhitelisted, err := tc.verify(ctx, addr)
						s.Require().NoError(err)
						if tc.operation == "add" {
							s.Require().True(isWhitelisted)
						} else {
							s.Require().False(isWhitelisted)
						}
					}
				}
			}
		})
	}
}

func (s *MsgServerTestSuite) TestTopicWhitelistOperations() {
	ctx := s.Ctx()
	msgServer := s.EmissionsMsgServer()
	keeper := s.EmissionsKeeper()

	adminAddr := s.AddrsStr(0)
	targetAddr := s.AddrsStr(1)
	nonAdminAddr := nonAdminAccounts[0].String()
	topicId := s.CreateTopic()
	nonExistentTopicId := uint64(1000)

	testCases := []struct {
		name          string
		whitelistType string // "worker" or "reputer"
		operation     string // "enable", "disable", "add", "remove"
		topicId       uint64
		setupMsg      func() any
		execute       func(any) error
		verify        func() (bool, error)
		expectedError error
	}{
		// Topic Worker Whitelist Operations
		{
			name:          "Enable topic worker whitelist - success",
			whitelistType: "worker",
			operation:     "enable",
			topicId:       topicId,
			setupMsg: func() any {
				return &types.EnableTopicWorkerWhitelistRequest{
					Sender:  adminAddr,
					TopicId: topicId,
				}
			},
			execute: func(msg any) error {
				_, err := msgServer.EnableTopicWorkerWhitelist(ctx, msg.(*types.EnableTopicWorkerWhitelistRequest))
				return err
			},
			verify: func() (bool, error) {
				return keeper.IsTopicWorkerWhitelistEnabled(ctx, topicId)
			},
		},
		{
			name:          "Enable topic worker whitelist - unauthorized",
			whitelistType: "worker",
			operation:     "enable",
			topicId:       topicId,
			setupMsg: func() any {
				return &types.EnableTopicWorkerWhitelistRequest{
					Sender:  nonAdminAddr,
					TopicId: topicId,
				}
			},
			execute: func(msg any) error {
				_, err := msgServer.EnableTopicWorkerWhitelist(ctx, msg.(*types.EnableTopicWorkerWhitelistRequest))
				return err
			},
			expectedError: types.ErrNotPermittedToUpdateTopicWhitelist,
		},
		{
			name:          "Enable topic worker whitelist - topic does not exist",
			whitelistType: "worker",
			operation:     "enable",
			topicId:       nonExistentTopicId,
			setupMsg: func() any {
				return &types.EnableTopicWorkerWhitelistRequest{
					Sender:  nonAdminAddr,
					TopicId: nonExistentTopicId,
				}
			},
			execute: func(msg any) error {
				_, err := msgServer.EnableTopicWorkerWhitelist(ctx, msg.(*types.EnableTopicWorkerWhitelistRequest))
				return err
			},
			expectedError: types.ErrTopicDoesNotExist,
		},
		{
			name:          "Disable topic worker whitelist - success",
			whitelistType: "worker",
			operation:     "disable",
			topicId:       topicId,
			setupMsg: func() any {
				// First enable
				enableMsg := &types.EnableTopicWorkerWhitelistRequest{
					Sender:  adminAddr,
					TopicId: topicId,
				}
				_, err := msgServer.EnableTopicWorkerWhitelist(ctx, enableMsg)
				s.Require().NoError(err)

				return &types.DisableTopicWorkerWhitelistRequest{
					Sender:  adminAddr,
					TopicId: topicId,
				}
			},
			execute: func(msg any) error {
				_, err := msgServer.DisableTopicWorkerWhitelist(ctx, msg.(*types.DisableTopicWorkerWhitelistRequest))
				return err
			},
			verify: func() (bool, error) {
				return keeper.IsTopicWorkerWhitelistEnabled(ctx, topicId)
			},
		},
		{
			name:          "Add to topic worker whitelist - success",
			whitelistType: "worker",
			operation:     "add",
			topicId:       topicId,
			setupMsg: func() any {
				return &types.AddToTopicWorkerWhitelistRequest{
					Sender:  adminAddr,
					TopicId: topicId,
					Address: targetAddr,
				}
			},
			execute: func(msg any) error {
				_, err := msgServer.AddToTopicWorkerWhitelist(ctx, msg.(*types.AddToTopicWorkerWhitelistRequest))
				return err
			},
		},
		{
			name:          "Add to topic worker whitelist - unauthorized",
			whitelistType: "worker",
			operation:     "add",
			topicId:       topicId,
			setupMsg: func() any {
				return &types.AddToTopicWorkerWhitelistRequest{
					Sender:  nonAdminAddr,
					TopicId: topicId,
					Address: targetAddr,
				}
			},
			execute: func(msg any) error {
				_, err := msgServer.AddToTopicWorkerWhitelist(ctx, msg.(*types.AddToTopicWorkerWhitelistRequest))
				return err
			},
			expectedError: types.ErrNotPermittedToUpdateTopicWorkerWhitelist,
		},
		{
			name:          "Remove from topic worker whitelist - success",
			whitelistType: "worker",
			operation:     "remove",
			topicId:       topicId,
			setupMsg: func() any {
				return &types.RemoveFromTopicWorkerWhitelistRequest{
					Sender:  adminAddr,
					TopicId: topicId,
					Address: targetAddr,
				}
			},
			execute: func(msg any) error {
				_, err := msgServer.RemoveFromTopicWorkerWhitelist(ctx, msg.(*types.RemoveFromTopicWorkerWhitelistRequest))
				return err
			},
		},

		// Topic Reputer Whitelist Operations (similar pattern)
		{
			name:          "Enable topic reputer whitelist - success",
			whitelistType: "reputer",
			operation:     "enable",
			topicId:       topicId,
			setupMsg: func() any {
				return &types.EnableTopicReputerWhitelistRequest{
					Sender:  adminAddr,
					TopicId: topicId,
				}
			},
			execute: func(msg any) error {
				_, err := msgServer.EnableTopicReputerWhitelist(ctx, msg.(*types.EnableTopicReputerWhitelistRequest))
				return err
			},
			verify: func() (bool, error) {
				return keeper.IsTopicReputerWhitelistEnabled(ctx, topicId)
			},
		},
		{
			name:          "Disable topic reputer whitelist - success",
			whitelistType: "reputer",
			operation:     "disable",
			topicId:       topicId,
			setupMsg: func() any {
				// First enable
				enableMsg := &types.EnableTopicReputerWhitelistRequest{
					Sender:  adminAddr,
					TopicId: topicId,
				}
				_, err := msgServer.EnableTopicReputerWhitelist(ctx, enableMsg)
				s.Require().NoError(err)

				return &types.DisableTopicReputerWhitelistRequest{
					Sender:  adminAddr,
					TopicId: topicId,
				}
			},
			execute: func(msg any) error {
				_, err := msgServer.DisableTopicReputerWhitelist(ctx, msg.(*types.DisableTopicReputerWhitelistRequest))
				return err
			},
			verify: func() (bool, error) {
				return keeper.IsTopicReputerWhitelistEnabled(ctx, topicId)
			},
		},
		{
			name:          "Add to topic reputer whitelist - success",
			whitelistType: "reputer",
			operation:     "add",
			topicId:       topicId,
			setupMsg: func() any {
				return &types.AddToTopicReputerWhitelistRequest{
					Sender:  adminAddr,
					TopicId: topicId,
					Address: targetAddr,
				}
			},
			execute: func(msg any) error {
				_, err := msgServer.AddToTopicReputerWhitelist(ctx, msg.(*types.AddToTopicReputerWhitelistRequest))
				return err
			},
		},
		{
			name:          "Remove from topic reputer whitelist - success",
			whitelistType: "reputer",
			operation:     "remove",
			topicId:       topicId,
			setupMsg: func() any {
				return &types.RemoveFromTopicReputerWhitelistRequest{
					Sender:  adminAddr,
					TopicId: topicId,
					Address: targetAddr,
				}
			},
			execute: func(msg any) error {
				_, err := msgServer.RemoveFromTopicReputerWhitelist(ctx, msg.(*types.RemoveFromTopicReputerWhitelistRequest))
				return err
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			msg := tc.setupMsg()
			err := tc.execute(msg)

			if tc.expectedError != nil {
				s.Require().ErrorIs(err, tc.expectedError)
			} else {
				s.Require().NoError(err)
				if tc.verify != nil {
					result, err := tc.verify()
					s.Require().NoError(err)
					if tc.operation == "enable" {
						s.Require().True(result)
					} else if tc.operation == "disable" {
						s.Require().False(result)
					}
				}
			}
		})
	}
}

func (s *MsgServerTestSuite) TestBulkTopicWhitelistOperations() {
	ctx := s.Ctx()
	msgServer := s.EmissionsMsgServer()
	keeper := s.EmissionsKeeper()
	adminAddr := s.AddrsStr(0)
	topicId := s.CreateTopic()

	addresses := []string{
		nonAdminAccounts[0].String(),
		nonAdminAccounts[1].String(),
		nonAdminAccounts[2].String(),
	}

	// Set max array length for testing
	params, err := keeper.GetParams(ctx)
	s.Require().NoError(err)
	params.MaxWhitelistInputArrayLength = 3
	err = keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	tooManyAddresses := make([]string, params.MaxWhitelistInputArrayLength+1)
	for i := range tooManyAddresses {
		tooManyAddresses[i] = nonAdminAccounts[i].String()
	}

	testCases := []struct {
		name          string
		whitelistType string // "worker" or "reputer"
		operation     string // "add" or "remove"
		addresses     []string
		execute       func() error
		verify        func(string) (bool, error)
		expectedError error
	}{
		// Topic Worker Bulk Operations
		{
			name:          "Bulk add to topic worker whitelist - success",
			whitelistType: "worker",
			operation:     "add",
			addresses:     addresses,
			execute: func() error {
				msg := &types.BulkAddToTopicWorkerWhitelistRequest{
					Sender:    adminAddr,
					Addresses: addresses,
					TopicId:   topicId,
				}
				_, err = msgServer.BulkAddToTopicWorkerWhitelist(ctx, msg)
				return err
			},
			verify: func(addr string) (bool, error) {
				return keeper.IsWhitelistedTopicWorker(ctx, topicId, addr)
			},
		},
		{
			name:          "Bulk add to topic worker whitelist - too many addresses",
			whitelistType: "worker",
			operation:     "add",
			addresses:     tooManyAddresses,
			execute: func() error {
				msg := &types.BulkAddToTopicWorkerWhitelistRequest{
					Sender:    adminAddr,
					Addresses: tooManyAddresses,
					TopicId:   topicId,
				}
				_, err = msgServer.BulkAddToTopicWorkerWhitelist(ctx, msg)
				return err
			},
			expectedError: types.ErrMaxWhitelistInputArrayLengthExceeded,
		},
		{
			name:          "Bulk remove from topic worker whitelist - success",
			whitelistType: "worker",
			operation:     "remove",
			addresses:     addresses,
			execute: func() error {
				// First add addresses
				addMsg := &types.BulkAddToTopicWorkerWhitelistRequest{
					Sender:    adminAddr,
					Addresses: addresses,
					TopicId:   topicId,
				}
				_, err = msgServer.BulkAddToTopicWorkerWhitelist(ctx, addMsg)
				s.Require().NoError(err)

				// Then remove
				msg := &types.BulkRemoveFromTopicWorkerWhitelistRequest{
					Sender:    adminAddr,
					Addresses: addresses,
					TopicId:   topicId,
				}
				_, err = msgServer.BulkRemoveFromTopicWorkerWhitelist(ctx, msg)
				return err
			},
			verify: func(addr string) (bool, error) {
				return keeper.IsWhitelistedTopicWorker(ctx, topicId, addr)
			},
		},
		{
			name:          "Bulk remove from topic worker whitelist - too many addresses",
			whitelistType: "worker",
			operation:     "remove",
			addresses:     tooManyAddresses,
			execute: func() error {
				msg := &types.BulkRemoveFromTopicWorkerWhitelistRequest{
					Sender:    adminAddr,
					Addresses: tooManyAddresses,
					TopicId:   topicId,
				}
				_, err = msgServer.BulkRemoveFromTopicWorkerWhitelist(ctx, msg)
				return err
			},
			expectedError: types.ErrMaxWhitelistInputArrayLengthExceeded,
		},

		// Topic Reputer Bulk Operations
		{
			name:          "Bulk add to topic reputer whitelist - success",
			whitelistType: "reputer",
			operation:     "add",
			addresses:     addresses,
			execute: func() error {
				msg := &types.BulkAddToTopicReputerWhitelistRequest{
					Sender:    adminAddr,
					Addresses: addresses,
					TopicId:   topicId,
				}
				_, err := msgServer.BulkAddToTopicReputerWhitelist(ctx, msg)
				return err
			},
			verify: func(addr string) (bool, error) {
				return keeper.IsWhitelistedTopicReputer(ctx, topicId, addr)
			},
		},
		{
			name:          "Bulk add to topic reputer whitelist - too many addresses",
			whitelistType: "reputer",
			operation:     "add",
			addresses:     tooManyAddresses,
			execute: func() error {
				msg := &types.BulkAddToTopicReputerWhitelistRequest{
					Sender:    adminAddr,
					Addresses: tooManyAddresses,
					TopicId:   topicId,
				}
				_, err = msgServer.BulkAddToTopicReputerWhitelist(ctx, msg)
				return err
			},
			expectedError: types.ErrMaxWhitelistInputArrayLengthExceeded,
		},
		{
			name:          "Bulk remove from topic reputer whitelist - success",
			whitelistType: "reputer",
			operation:     "remove",
			addresses:     addresses,
			execute: func() error {
				// First add addresses
				addMsg := &types.BulkAddToTopicReputerWhitelistRequest{
					Sender:    adminAddr,
					Addresses: addresses,
					TopicId:   topicId,
				}
				_, err = msgServer.BulkAddToTopicReputerWhitelist(ctx, addMsg)
				s.Require().NoError(err)

				// Then remove
				msg := &types.BulkRemoveFromTopicReputerWhitelistRequest{
					Sender:    adminAddr,
					Addresses: addresses,
					TopicId:   topicId,
				}
				_, err = msgServer.BulkRemoveFromTopicReputerWhitelist(ctx, msg)
				return err
			},
			verify: func(addr string) (bool, error) {
				return keeper.IsWhitelistedTopicReputer(ctx, topicId, addr)
			},
		},
		{
			name:          "Bulk remove from topic reputer whitelist - too many addresses",
			whitelistType: "reputer",
			operation:     "remove",
			addresses:     tooManyAddresses,
			execute: func() error {
				msg := &types.BulkRemoveFromTopicReputerWhitelistRequest{
					Sender:    adminAddr,
					Addresses: tooManyAddresses,
					TopicId:   topicId,
				}
				_, err = msgServer.BulkRemoveFromTopicReputerWhitelist(ctx, msg)
				return err
			},
			expectedError: types.ErrMaxWhitelistInputArrayLengthExceeded,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.execute()

			if tc.expectedError != nil {
				s.Require().ErrorIs(err, tc.expectedError)
			} else {
				s.Require().NoError(err)
				if tc.verify != nil {
					for _, addr := range tc.addresses {
						isWhitelisted, err := tc.verify(addr)
						s.Require().NoError(err)
						if tc.operation == "add" {
							s.Require().True(isWhitelisted)
						} else {
							s.Require().False(isWhitelisted)
						}
					}
				}
			}
		})
	}
}
