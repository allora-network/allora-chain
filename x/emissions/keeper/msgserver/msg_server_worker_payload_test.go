package msgserver_test

import (
	"bytes"
	"fmt"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/testutil/sims"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestMsgInsertBulkWorkerPayload() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	//workerAddr := sdk.AccAddress(PKS[0].Address()).String()
	inferencerAddr := sdk.AccAddress(PKS[0].Address()).String()
	inferencerAddr1 := sdk.AccAddress(PKS[0].Address()).String()
	inferencerAddr2 := sdk.AccAddress(PKS[0].Address()).String()
	forecasterAddr := sdk.AccAddress(PKS[0].Address()).String()

	test := sdk.AccAddress(simtestutil.CreateTestPubKeys(1)[0].Address())
	// Create a MsgInsertBulkReputerPayload message
	workerMsg := &types.MsgInsertBulkWorkerPayload{
		Sender:  test.String(),
		Nonce:   &types.Nonce{1},
		TopicId: 1,
		Inferences: []*types.Inference{
			{
				TopicId:   1,
				Worker:    inferencerAddr,
				Value:     alloraMath.NewDecFromInt64(100),
				Signature: []byte("Inferences"),
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
				Signature: []byte("Forecasts"),
			},
		},
	}
	var buffer bytes.Buffer
	buffer.WriteString(workerMsg.Sender)
	te := sims.NewPubKeyFromHex(buffer.String())
	fmt.Println("te", te.Address().String())
	_, err := msgServer.InsertBulkWorkerPayload(ctx, workerMsg)
	require.NoError(err, "InsertBulkWorkerPayload should not return an error")
}
