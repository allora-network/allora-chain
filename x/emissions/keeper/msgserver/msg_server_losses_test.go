package msgserver_test

import (
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestMsgInsertLosses() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address()).String()
	workerAddr := sdk.AccAddress(PKS[1].Address()).String()

	// Create a MsgInsertLosses message
	lossesMsg := &types.MsgInsertLosses{
		Sender: reputerAddr,
		ReputerValueBundles: []*types.ReputerValueBundle{
			{
				Reputer: reputerAddr,
				ValueBundle: &types.ValueBundle{
					TopicId:       1,
					CombinedValue: 100,
					InfererValues: []*types.WorkerAttributedValue{
						{
							Worker: workerAddr,
							Value:  100,
						},
					},
					ForecasterValues: []*types.WorkerAttributedValue{
						{
							Worker: workerAddr,
							Value:  100,
						},
					},
					NaiveValue: 100,
					OneOutInfererValues: []*types.WithheldWorkerAttributedValue{
						{
							Worker: workerAddr,
							Value:  100,
						},
					},
					OneOutForecasterValues: []*types.WithheldWorkerAttributedValue{
						{
							Worker: workerAddr,
							Value:  100,
						},
					},
					OneInForecasterValues: []*types.WorkerAttributedValue{
						{
							Worker: workerAddr,
							Value:  100,
						},
					},
				},
			},
		},
	}

	_, err := msgServer.InsertLosses(ctx, lossesMsg)
	require.NoError(err, "InsertLosses should not return an error")
}

func (s *KeeperTestSuite) TestMsgInsertLossesInvalidUnauthorized() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := nonAdminAccounts[0].String()
	workerAddr := sdk.AccAddress(PKS[1].Address()).String()

	// Create a MsgInsertLosses message
	lossesMsg := &types.MsgInsertLosses{
		Sender: reputerAddr,
		ReputerValueBundles: []*types.ReputerValueBundle{
			{
				Reputer: reputerAddr,
				ValueBundle: &types.ValueBundle{
					TopicId:       1,
					CombinedValue: 100,
					InfererValues: []*types.WorkerAttributedValue{
						{
							Worker: workerAddr,
							Value:  100,
						},
					},
					ForecasterValues: []*types.WorkerAttributedValue{
						{
							Worker: workerAddr,
							Value:  100,
						},
					},
					NaiveValue: 100,
					OneOutInfererValues: []*types.WithheldWorkerAttributedValue{
						{
							Worker: workerAddr,
							Value:  100,
						},
					},
					OneOutForecasterValues: []*types.WithheldWorkerAttributedValue{
						{
							Worker: workerAddr,
							Value:  100,
						},
					},
					OneInForecasterValues: []*types.WorkerAttributedValue{
						{
							Worker: workerAddr,
							Value:  100,
						},
					},
				},
			},
		},
	}

	_, err := msgServer.InsertLosses(ctx, lossesMsg)
	require.ErrorIs(err, types.ErrNotInReputerWhitelist, "InsertLosses should return an error")
}
