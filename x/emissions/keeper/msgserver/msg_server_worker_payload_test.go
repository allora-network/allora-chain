package msgserver_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
)

func (s *KeeperTestSuite) TestMsgInsertBulkWorkerPayload() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	workerAddr := sdk.AccAddress(PKS[0].Address()).String()
	inferencerAddr := sdk.AccAddress(PKS[0].Address()).String()
	inferencerAddr1 := sdk.AccAddress(PKS[0].Address()).String()
	inferencerAddr2 := sdk.AccAddress(PKS[0].Address()).String()
	forecasterAddr := sdk.AccAddress(PKS[0].Address()).String()

	// Create a MsgInsertBulkReputerPayload message
	workerMsg := &types.MsgInsertBulkWorkerPayload{
		Sender:  workerAddr,
		Nonce:   &types.Nonce{1},
		TopicId: 1,
		Inferences: []*types.Inference{
			{
				TopicId:   1,
				Worker:    inferencerAddr,
				Value:     alloraMath.NewDecFromInt64(100),
				Signature: []byte("Inference Signature"),
			},
		},
		Forecasts: []*types.Forecast{
			{
				TopicId:    1,
				Forecaster: forecasterAddr,
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: inferencerAddr1,
						Value:   alloraMath.NewDecFromInt64(200),
					},
					{
						Inferer: inferencerAddr2,
						Value:   alloraMath.NewDecFromInt64(300),
					},
				},
			},
			Signature: []byte("Forecast Signature"),
		},
		Signature:      []byte("Inferences + Forecasts Signature"),
		NonceSignature: []byte("Nonce Signature"),
	}
	senderAddr, err := sdk.AccAddressFromBech32(workerMsg.Sender)
	if err != nil {
		s.Require().Error(err)
	}
	s.authKeeper.EXPECT().GetAccount(gomock.Any(), senderAddr)
	_, err = msgServer.InsertBulkWorkerPayload(ctx, workerMsg)
	require.NoError(err, "InsertBulkWorkerPayload should not return an error")
}
