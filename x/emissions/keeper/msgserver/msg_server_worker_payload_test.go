package msgserver_test

import (
	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestMsgInsertBulkWorkerPayload() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	workerAddr := sdk.AccAddress(PKS[0].Address())
	inferencerAddr := sdk.AccAddress(PKS[1].Address())
	inferencerAddr1 := sdk.AccAddress(PKS[2].Address())
	inferencerAddr2 := sdk.AccAddress(PKS[3].Address())
	forecasterAddr := sdk.AccAddress(PKS[4].Address())
	reputerAddr := sdk.AccAddress(PKS[5].Address())

	registrationInitialStake := cosmosMath.NewUint(100)

	// Create topic 0 and register reputer in it
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

	// Try to register again
	registerMsg := &types.MsgRegister{
		Creator:      reputerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicIds:     []uint64{0},
		InitialStake: registrationInitialStake,
		IsReputer:    false,
	}
	_, err := msgServer.Register(ctx, registerMsg)

	// Create a MsgInsertBulkReputerPayload message
	workerMsg := &types.MsgInsertBulkWorkerPayload{
		Sender:  workerAddr.String(),
		Nonce:   &types.Nonce{1},
		TopicId: 1,
		WorkerDataBundles: []*types.WorkerDataBundle{
			{
				Worker: inferencerAddr.String(),
				InferenceForecastsBundle: &types.InferenceForecastBundle{
					Inference: &types.Inference{
						TopicId:     1,
						BlockHeight: 1,
						Inferer:     inferencerAddr1.String(),
						Value:       alloraMath.NewDecFromInt64(100),
					},
					Forecast: &types.Forecast{
						TopicId:     1,
						BlockHeight: 10,
						Forecaster:  forecasterAddr.String(),
						ForecastElements: []*types.ForecastElement{
							{
								Inferer: inferencerAddr2.String(),
								Value:   alloraMath.NewDecFromInt64(100),
							},
							{
								Inferer: inferencerAddr2.String(),
								Value:   alloraMath.NewDecFromInt64(100),
							},
						},
					},
				},
				InferencesForecastsBundleSignature: []byte("Signature"),
			},
		},
	}
	_, err = msgServer.InsertBulkWorkerPayload(ctx, workerMsg)
	require.NoError(err, "InsertBulkWorkerPayload should not return an error")
}
