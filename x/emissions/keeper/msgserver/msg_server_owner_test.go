package msgserver_test

import (
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *MsgServerTestSuite) TestUpdateOwnerSuccess() {
	ctx := s.Ctx()
	msgServer := s.EmissionsMsgServer()

	testCases := []struct {
		name      string
		isReputer bool
	}{
		{name: "reputer owner updated", isReputer: true},
		{name: "worker owner updated", isReputer: false},
	}

	for i, tc := range testCases {
		s.Run(tc.name, func() {
			sender := s.AddrsStr(i + 1)
			oldOwner := s.AddrsStr(i + 10)
			newOwner := s.AddrsStr(i + 20)
			topicID := uint64(100 + i)

			if tc.isReputer {
				err := s.EmissionsKeeper().InsertReputer(ctx, topicID, sender, types.OffchainNode{
					NodeAddress: sender,
					Owner:       oldOwner,
				})
				s.Require().NoError(err)
			} else {
				err := s.EmissionsKeeper().InsertWorker(ctx, topicID, sender, types.OffchainNode{
					NodeAddress: sender,
					Owner:       oldOwner,
				})
				s.Require().NoError(err)
			}

			resp, err := msgServer.UpdateOwner(ctx, &types.UpdateOwnerRequest{
				Sender:    sender,
				NewOwner:  newOwner,
				IsReputer: tc.isReputer,
			})
			s.Require().NoError(err)
			s.Require().True(resp.Success)

			if tc.isReputer {
				node, err := s.EmissionsKeeper().GetReputerInfo(s.Ctx(), sender)
				s.Require().NoError(err)
				s.Require().Equal(newOwner, node.Owner)
			} else {
				node, err := s.EmissionsKeeper().GetWorkerInfo(s.Ctx(), sender)
				s.Require().NoError(err)
				s.Require().Equal(newOwner, node.Owner)
			}
		})
	}
}

func (s *MsgServerTestSuite) TestUpdateOwnerValidationErrors() {
	ctx := s.Ctx()
	msgServer := s.EmissionsMsgServer()

	testCases := []struct {
		name string
		msg  *types.UpdateOwnerRequest
	}{
		{
			name: "invalid sender address",
			msg: &types.UpdateOwnerRequest{
				Sender:    "invalid",
				NewOwner:  s.AddrsStr(1),
				IsReputer: true,
			},
		},
		{
			name: "invalid new owner",
			msg: &types.UpdateOwnerRequest{
				Sender:    s.AddrsStr(1),
				NewOwner:  "invalid",
				IsReputer: true,
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			_, err := msgServer.UpdateOwner(ctx, tc.msg)
			s.Require().Error(err)
			s.Require().ErrorContains(err, "invalid")
		})
	}
}

func (s *MsgServerTestSuite) TestUpdateOwnerAddressNotRegistered() {
	ctx := s.Ctx()
	msgServer := s.EmissionsMsgServer()

	addr := s.AddrsStr(5)
	owner := s.AddrsStr(6)
	topicID := uint64(501)

	// Register only as worker.
	err := s.EmissionsKeeper().InsertWorker(ctx, topicID, addr, types.OffchainNode{
		NodeAddress: addr,
		Owner:       owner,
	})
	s.Require().NoError(err)

	_, err = msgServer.UpdateOwner(ctx, &types.UpdateOwnerRequest{
		Sender:    addr,
		NewOwner:  s.AddrsStr(7),
		IsReputer: true,
	})
	s.Require().ErrorIs(err, types.ErrAddressNotRegistered)

	// Register only as reputer.
	addr2 := s.AddrsStr(8)
	err = s.EmissionsKeeper().InsertReputer(ctx, topicID+1, addr2, types.OffchainNode{
		NodeAddress: addr2,
		Owner:       owner,
	})
	s.Require().NoError(err)

	_, err = msgServer.UpdateOwner(ctx, &types.UpdateOwnerRequest{
		Sender:    addr2,
		NewOwner:  s.AddrsStr(9),
		IsReputer: false,
	})
	s.Require().ErrorIs(err, types.ErrAddressNotRegistered)
}

func (s *MsgServerTestSuite) TestUpdateOwnerInvariantMismatch() {
	ctx := s.Ctx()
	msgServer := s.EmissionsMsgServer()

	reputerAddr := s.AddrsStr(11)
	mismatchedNode := s.AddrsStr(12)
	err := s.EmissionsKeeper().InsertReputer(ctx, 700, reputerAddr, types.OffchainNode{
		NodeAddress: mismatchedNode,
		Owner:       s.AddrsStr(13),
	})
	s.Require().NoError(err)

	_, err = msgServer.UpdateOwner(ctx, &types.UpdateOwnerRequest{
		Sender:    reputerAddr,
		NewOwner:  s.AddrsStr(14),
		IsReputer: true,
	})
	s.Require().ErrorIs(err, types.ErrInvariantFailure)

	workerAddr := s.AddrsStr(15)
	mismatchedWorkerNode := s.AddrsStr(16)
	err = s.EmissionsKeeper().InsertWorker(ctx, 701, workerAddr, types.OffchainNode{
		NodeAddress: mismatchedWorkerNode,
		Owner:       s.AddrsStr(17),
	})
	s.Require().NoError(err)

	_, err = msgServer.UpdateOwner(ctx, &types.UpdateOwnerRequest{
		Sender:    workerAddr,
		NewOwner:  s.AddrsStr(18),
		IsReputer: false,
	})
	s.Require().ErrorIs(err, types.ErrInvariantFailure)
}

func (s *MsgServerTestSuite) TestUpdateOwnerRoleSpecificity() {
	ctx := s.Ctx()
	msgServer := s.EmissionsMsgServer()

	addr := s.AddrsStr(19)
	reputerOwner := s.AddrsStr(20)
	workerOwner := s.AddrsStr(21)
	newReputerOwner := s.AddrsStr(22)
	newWorkerOwner := s.AddrsStr(23)

	err := s.EmissionsKeeper().InsertReputer(ctx, 800, addr, types.OffchainNode{
		NodeAddress: addr,
		Owner:       reputerOwner,
	})
	s.Require().NoError(err)

	err = s.EmissionsKeeper().InsertWorker(ctx, 801, addr, types.OffchainNode{
		NodeAddress: addr,
		Owner:       workerOwner,
	})
	s.Require().NoError(err)

	_, err = msgServer.UpdateOwner(ctx, &types.UpdateOwnerRequest{
		Sender:    addr,
		NewOwner:  newReputerOwner,
		IsReputer: true,
	})
	s.Require().NoError(err)

	storedReputer, err := s.EmissionsKeeper().GetReputerInfo(s.Ctx(), addr)
	s.Require().NoError(err)
	s.Require().Equal(newReputerOwner, storedReputer.Owner)

	// Worker owner should remain unchanged.
	storedWorker, err := s.EmissionsKeeper().GetWorkerInfo(s.Ctx(), addr)
	s.Require().NoError(err)
	s.Require().Equal(workerOwner, storedWorker.Owner)

	_, err = msgServer.UpdateOwner(ctx, &types.UpdateOwnerRequest{
		Sender:    addr,
		NewOwner:  newWorkerOwner,
		IsReputer: false,
	})
	s.Require().NoError(err)

	storedWorker, err = s.EmissionsKeeper().GetWorkerInfo(s.Ctx(), addr)
	s.Require().NoError(err)
	s.Require().Equal(newWorkerOwner, storedWorker.Owner)

	// Reputer owner should remain as previously updated value.
	storedReputer, err = s.EmissionsKeeper().GetReputerInfo(s.Ctx(), addr)
	s.Require().NoError(err)
	s.Require().Equal(newReputerOwner, storedReputer.Owner)
}
