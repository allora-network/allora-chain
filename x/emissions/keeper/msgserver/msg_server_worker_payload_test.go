package msgserver_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestMsgInsertBulkWorkerPayload() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	workerAddr := sdk.AccAddress(PKS[0].Address()).String()
	inferencerAddr := sdk.AccAddress(PKS[0].Address()).String()
	forecasterAddr1 := sdk.AccAddress(PKS[0].Address()).String()
	forecasterAddr2 := sdk.AccAddress(PKS[0].Address()).String()

	// Create a MsgInsertBulkReputerPayload message
	workerMsg := &types.MsgInsertBulkWorkerPayload{
		Sender:  workerAddr,
		Nonce:   &types.Nonce{1},
		TopicId: 1,
		Inferences: []*types.Inference{
			{
				TopicId: 1,
				Worker:  inferencerAddr,
				Value:   alloraMath.NewDecFromInt64(100),
			},
		},
		Forecasts: []*types.Forecast{
			{
				TopicId:    1,
				Forecaster: "",
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: forecasterAddr1,
						Value:   alloraMath.NewDecFromInt64(200),
					},
					{
						Inferer: forecasterAddr2,
						Value:   alloraMath.NewDecFromInt64(300),
					},
				},
			},
		},
		Signature: []byte("Test"),
	}

	_, err := msgServer.InsertBulkWorkerPayload(ctx, workerMsg)
	require.NoError(err, "InsertBulkWorkerPayload should not return an error")
}
