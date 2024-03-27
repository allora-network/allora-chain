package msgserver_test

import (
	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestMsgSetLosses() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address()).String()
	workerAddr := sdk.AccAddress(PKS[1].Address()).String()

	// Create a MsgSetLosses message
	lossesMsg := &types.MsgSetLosses{
		Sender: reputerAddr,
		ValueBundles: []*types.ValueBundle{
			{
				TopicId:      1,
				Reputer:      reputerAddr,
				CombinedLoss: cosmosMath.NewUint(100),
				InfererLosses: []*types.WorkerAttributedValue{
					{
						Worker: workerAddr,
						Value:  cosmosMath.NewUint(100),
					},
				},
				ForecasterLosses: []*types.WorkerAttributedValue{
					{
						Worker: workerAddr,
						Value:  cosmosMath.NewUint(100),
					},
				},
				NaiveLoss: cosmosMath.NewUint(100),
				OneOutLosses: []*types.WorkerAttributedValue{
					{
						Worker: workerAddr,
						Value:  cosmosMath.NewUint(100),
					},
				},
				OneInNaiveLosses: []*types.WorkerAttributedValue{
					{
						Worker: workerAddr,
						Value:  cosmosMath.NewUint(100),
					},
				},
			},
		},
	}

	_, err := msgServer.InsertLosses(ctx, lossesMsg)
	require.NoError(err, "InsertLosses should not return an error")
}

func (s *KeeperTestSuite) TestMsgSetLossesInvalidUnauthorized() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := nonAdminAccounts[0].String()
	workerAddr := sdk.AccAddress(PKS[1].Address()).String()

	// Create a MsgSetLosses message
	lossesMsg := &types.MsgSetLosses{
		Sender: reputerAddr,
		ValueBundles: []*types.ValueBundle{
			{
				TopicId:      1,
				Reputer:      reputerAddr,
				CombinedLoss: cosmosMath.NewUint(100),
				InfererLosses: []*types.WorkerAttributedValue{
					{
						Worker: workerAddr,
						Value:  cosmosMath.NewUint(100),
					},
				},
				ForecasterLosses: []*types.WorkerAttributedValue{
					{
						Worker: workerAddr,
						Value:  cosmosMath.NewUint(100),
					},
				},
				NaiveLoss: cosmosMath.NewUint(100),
				OneOutLosses: []*types.WorkerAttributedValue{
					{
						Worker: workerAddr,
						Value:  cosmosMath.NewUint(100),
					},
				},
				OneInNaiveLosses: []*types.WorkerAttributedValue{
					{
						Worker: workerAddr,
						Value:  cosmosMath.NewUint(100),
					},
				},
			},
		},
	}

	_, err := msgServer.InsertLosses(ctx, lossesMsg)
	require.ErrorIs(err, types.ErrNotInReputerWhitelist, "InsertLosses should return an error")
}
