package msgserver_test

import (
	cosmosMath "cosmossdk.io/math"
	"encoding/hex"
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
	reputerAddr := sdk.AccAddress(PKS[4].Address())
	forecasterAddr := sdk.AccAddress(PKS[5].Address())
	registrationInitialStake := cosmosMath.NewUint(100)

	// Create topic 0 and register reputer in it
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)
	s.emissionsKeeper.AddWorkerNonce(ctx, 0, &types.Nonce{1})

	// Create a MsgInsertBulkReputerPayload message
	workerMsg := &types.MsgInsertBulkWorkerPayload{
		Sender:  workerAddr.String(),
		Nonce:   &types.Nonce{1},
		TopicId: 0,
		WorkerDataBundles: []*types.WorkerDataBundle{
			{
				Worker: inferencerAddr.String(),
				InferenceForecastsBundle: &types.InferenceForecastBundle{
					Inference: &types.Inference{
						TopicId:     0,
						BlockHeight: 1,
						Inferer:     inferencerAddr1.String(),
						Value:       alloraMath.NewDecFromInt64(100),
					},
					Forecast: &types.Forecast{
						TopicId:     0,
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
				InferencesForecastsBundleSignature: []byte("InferenceForecastBundle Signature"),
				Pubkey:                             "Worker Pubkey",
			},
		},
	}

	workerMsg.WorkerDataBundles[0].InferencesForecastsBundleSignature, _ = hex.DecodeString("6182c6115df6c6d2c603797f7ed4ca882eb7bc8c0f1536803b9d117bb22933b578c6e093219f5c1bbe41ae2da895cbd7079e37840720f0c352541f10c162334c")
	workerMsg.WorkerDataBundles[0].Pubkey = "031defa76703f22f4db7590df684052d2ae52ad693981304087b906a628c747996"
	_, err := msgServer.InsertBulkWorkerPayload(ctx, workerMsg)
	require.NoError(err, "InsertBulkWorkerPayload should not return an error")
}

func (s *KeeperTestSuite) TestMsgInsertBulkWorkerPayloadVerifyFailed() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	workerAddr := sdk.AccAddress(PKS[0].Address())
	inferencerAddr := sdk.AccAddress(PKS[1].Address())
	inferencerAddr1 := sdk.AccAddress(PKS[2].Address())
	inferencerAddr2 := sdk.AccAddress(PKS[3].Address())
	reputerAddr := sdk.AccAddress(PKS[4].Address())
	forecasterAddr := sdk.AccAddress(PKS[5].Address())
	registrationInitialStake := cosmosMath.NewUint(100)

	// Create topic 0 and register reputer in it
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)
	s.emissionsKeeper.AddWorkerNonce(ctx, 0, &types.Nonce{1})

	// Create a MsgInsertBulkReputerPayload message
	workerMsg := &types.MsgInsertBulkWorkerPayload{
		Sender:  workerAddr.String(),
		Nonce:   &types.Nonce{1},
		TopicId: 0,
		WorkerDataBundles: []*types.WorkerDataBundle{
			{
				Worker: inferencerAddr.String(),
				InferenceForecastsBundle: &types.InferenceForecastBundle{
					Inference: &types.Inference{
						TopicId:     0,
						BlockHeight: 1,
						Inferer:     inferencerAddr1.String(),
						Value:       alloraMath.NewDecFromInt64(100),
					},
					Forecast: &types.Forecast{
						TopicId:     0,
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
				InferencesForecastsBundleSignature: []byte("InferenceForecastBundle Signature"),
				Pubkey:                             "Worker Pubkey",
			},
		},
	}

	_, err := msgServer.InsertBulkWorkerPayload(ctx, workerMsg)
	require.ErrorIs(err, types.ErrSignatureVerificationFailed)
}

func (s *KeeperTestSuite) TestMsgInsertBulkWorkerAlreadyFullfilledNonce() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	workerAddr := sdk.AccAddress(PKS[0].Address())
	inferencerAddr := sdk.AccAddress(PKS[1].Address())
	inferencerAddr1 := sdk.AccAddress(PKS[2].Address())
	inferencerAddr2 := sdk.AccAddress(PKS[3].Address())
	reputerAddr := sdk.AccAddress(PKS[4].Address())
	forecasterAddr := sdk.AccAddress(PKS[5].Address())
	registrationInitialStake := cosmosMath.NewUint(100)

	// Create topic 0 and register reputer in it
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)
	s.emissionsKeeper.AddWorkerNonce(ctx, 0, &types.Nonce{1})

	// Create a MsgInsertBulkReputerPayload message
	workerMsg := &types.MsgInsertBulkWorkerPayload{
		Sender:  workerAddr.String(),
		Nonce:   &types.Nonce{1},
		TopicId: 0,
		WorkerDataBundles: []*types.WorkerDataBundle{
			{
				Worker: inferencerAddr.String(),
				InferenceForecastsBundle: &types.InferenceForecastBundle{
					Inference: &types.Inference{
						TopicId:     0,
						BlockHeight: 1,
						Inferer:     inferencerAddr1.String(),
						Value:       alloraMath.NewDecFromInt64(100),
					},
					Forecast: &types.Forecast{
						TopicId:     0,
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
				InferencesForecastsBundleSignature: []byte("InferenceForecastBundle Signature"),
				Pubkey:                             "Worker Pubkey",
			},
		},
	}

	workerMsg.WorkerDataBundles[0].InferencesForecastsBundleSignature, _ = hex.DecodeString("6182c6115df6c6d2c603797f7ed4ca882eb7bc8c0f1536803b9d117bb22933b578c6e093219f5c1bbe41ae2da895cbd7079e37840720f0c352541f10c162334c")
	workerMsg.WorkerDataBundles[0].Pubkey = "031defa76703f22f4db7590df684052d2ae52ad693981304087b906a628c747996"
	_, err := msgServer.InsertBulkWorkerPayload(ctx, workerMsg)
	_, err = msgServer.InsertBulkWorkerPayload(ctx, workerMsg)
	require.ErrorIs(err, types.ErrNonceAlreadyFulfilled)
}
