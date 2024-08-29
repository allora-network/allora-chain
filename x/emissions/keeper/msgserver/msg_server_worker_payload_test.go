package msgserver_test

import (
	"encoding/hex"

	cosmosMath "cosmossdk.io/math"
	chainParams "github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func getNewAddress() string {
	return sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
}

func (s *MsgServerTestSuite) setUpMsgInsertWorkerPayload(
	workerPrivateKey secp256k1.PrivKey,
) (types.MsgInsertWorkerPayload, uint64) {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	nonce := types.Nonce{BlockHeight: 1}
	topicId := s.CreateOneTopic()

	// Define sample OffchainNode information for a worker
	workerInfo := types.OffchainNode{
		Owner:       "worker-owner-sample",
		NodeAddress: "worker-node-address-sample",
	}

	// Mock setup for addresses
	reputerAddr := getNewAddress()
	workerAddr := sdk.AccAddress(workerPrivateKey.PubKey().Address()).String()
	InfererAddr := workerAddr
	Inferer2Addr := getNewAddress()
	Inferer3Addr := getNewAddress()
	Inferer4Addr := getNewAddress()

	moduleParams, err := keeper.GetParams(ctx)
	s.Require().NoError(err)

	// Create topic 0 and register reputer in it
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, moduleParams.RegistrationFee)
	err = keeper.AddWorkerNonce(ctx, topicId, &nonce)
	s.Require().NoError(err)
	err = keeper.InsertWorker(ctx, topicId, InfererAddr, workerInfo)
	s.Require().NoError(err)
	err = keeper.InsertWorker(ctx, topicId, Inferer2Addr, workerInfo)
	s.Require().NoError(err)

	topic, _ := s.emissionsKeeper.GetTopic(ctx, topicId)
	err = s.emissionsKeeper.SetTopic(ctx, topicId, topic)
	s.Require().NoError(err)

	// Create a MsgInsertWorkerPayload message
	workerMsg := types.MsgInsertWorkerPayload{
		Sender: workerAddr,
		WorkerDataBundle: &types.WorkerDataBundle{
			Worker:  InfererAddr,
			Nonce:   &nonce,
			TopicId: topicId,
			InferenceForecastsBundle: &types.InferenceForecastBundle{
				Inference: &types.Inference{
					TopicId:     topicId,
					BlockHeight: nonce.BlockHeight,
					Inferer:     workerAddr,
					Value:       alloraMath.NewDecFromInt64(100),
				},
				Forecast: &types.Forecast{
					TopicId:     topicId,
					BlockHeight: nonce.BlockHeight,
					Forecaster:  workerAddr,
					ForecastElements: []*types.ForecastElement{
						{
							Inferer: InfererAddr,
							Value:   alloraMath.NewDecFromInt64(100),
						},
						{
							Inferer: Inferer2Addr,
							Value:   alloraMath.NewDecFromInt64(101),
						},
						{
							Inferer: Inferer3Addr,
							Value:   alloraMath.NewDecFromInt64(102),
						},
						{
							Inferer: Inferer4Addr,
							Value:   alloraMath.NewDecFromInt64(103),
						},
					},
				},
			},
		},
	}

	return workerMsg, topicId
}

// set up multiple worker msgs for multiple workers
func (s *MsgServerTestSuite) setUpMsgInsertWorkerPayloadFourWorkers(
	workerPrivateKey1 secp256k1.PrivKey,
	workerPrivateKey2 secp256k1.PrivKey,
	workerPrivateKey3 secp256k1.PrivKey,
	workerPrivateKey4 secp256k1.PrivKey,
) (
	inferencePayloads []types.MsgInsertWorkerPayload,
	forecastPayloads []types.MsgInsertWorkerPayload,
	topicId uint64,
) {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	nonce := types.Nonce{BlockHeight: 1}
	topicId = s.CreateOneTopic()

	// Define sample OffchainNode information for a worker
	workerInfo := types.OffchainNode{
		Owner:       "worker-owner-sample",
		NodeAddress: "worker-node-address-sample",
	}

	// Mock setup for addresses
	reputerAddr := getNewAddress()
	worker1 := sdk.AccAddress(workerPrivateKey1.PubKey().Address())
	worker1Addr := worker1.String()
	worker2 := sdk.AccAddress(workerPrivateKey2.PubKey().Address())
	worker2Addr := worker2.String()
	worker3 := sdk.AccAddress(workerPrivateKey3.PubKey().Address())
	worker3Addr := worker3.String()
	worker4 := sdk.AccAddress(workerPrivateKey4.PubKey().Address())
	worker4Addr := worker4.String()

	moduleParams, err := keeper.GetParams(ctx)
	s.Require().NoError(err)

	// Create topic 0 and register actors in it
	s.commonStakingSetup(ctx, reputerAddr, worker1.String(), moduleParams.RegistrationFee)
	// give the other workers some coins too
	workerInitialBalanceCoins := sdk.NewCoins(sdk.NewCoin(chainParams.DefaultBondDenom, cosmosMath.NewInt(11000)))
	mintCoins := sdk.NewCoins(sdk.NewCoin(chainParams.DefaultBondDenom, cosmosMath.NewInt(11000*3)))
	err = s.bankKeeper.MintCoins(ctx, types.AlloraStakingAccountName, mintCoins)
	s.Require().NoError(err, "Minting coins should not return an error")
	err = s.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.AlloraStakingAccountName, worker2, workerInitialBalanceCoins)
	s.Require().NoError(err, "Sending coins should not return an error")
	err = s.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.AlloraStakingAccountName, worker3, workerInitialBalanceCoins)
	s.Require().NoError(err, "Sending coins should not return an error")
	err = s.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.AlloraStakingAccountName, worker4, workerInitialBalanceCoins)
	s.Require().NoError(err, "Sending coins should not return an error")
	// Register The other workers that didn't get registered in commonStakingSetup
	workerRegMsg := &types.MsgRegister{
		Sender:  worker2Addr,
		Owner:   worker2Addr,
		TopicId: topicId,
	}
	_, err = s.msgServer.Register(ctx, workerRegMsg)
	s.Require().NoError(err, "Registering worker should not return an error")
	workerRegMsg = &types.MsgRegister{
		Sender:  worker3Addr,
		Owner:   worker3Addr,
		TopicId: topicId,
	}
	_, err = s.msgServer.Register(ctx, workerRegMsg)
	s.Require().NoError(err, "Registering worker should not return an error")
	workerRegMsg = &types.MsgRegister{
		Sender:  worker4Addr,
		Owner:   worker4Addr,
		TopicId: topicId,
	}
	_, err = s.msgServer.Register(ctx, workerRegMsg)
	s.Require().NoError(err, "Registering worker should not return an error")

	err = keeper.AddWorkerNonce(ctx, topicId, &nonce)
	s.Require().NoError(err)
	err = keeper.InsertWorker(ctx, topicId, worker1Addr, workerInfo)
	s.Require().NoError(err)
	err = keeper.InsertWorker(ctx, topicId, worker2Addr, workerInfo)
	s.Require().NoError(err)
	err = keeper.InsertWorker(ctx, topicId, worker3Addr, workerInfo)
	s.Require().NoError(err)
	err = keeper.InsertWorker(ctx, topicId, worker4Addr, workerInfo)
	s.Require().NoError(err)

	topic, _ := s.emissionsKeeper.GetTopic(ctx, topicId)
	err = s.emissionsKeeper.SetTopic(ctx, topicId, topic)
	s.Require().NoError(err)

	worker1Value := alloraMath.NewDecFromInt64(100)
	worker2Value := alloraMath.NewDecFromInt64(101)
	worker3Value := alloraMath.NewDecFromInt64(102)
	worker4Value := alloraMath.NewDecFromInt64(103)

	inferencePayloads = []types.MsgInsertWorkerPayload{
		{
			Sender: worker1Addr,
			WorkerDataBundle: &types.WorkerDataBundle{
				Worker:  worker1Addr,
				Nonce:   &nonce,
				TopicId: topicId,
				InferenceForecastsBundle: &types.InferenceForecastBundle{
					Inference: &types.Inference{
						TopicId:     topicId,
						BlockHeight: nonce.BlockHeight,
						Inferer:     worker1Addr,
						Value:       worker1Value,
					},
					Forecast: nil,
				},
			},
		},
		{
			Sender: worker2Addr,
			WorkerDataBundle: &types.WorkerDataBundle{
				Worker:  worker2Addr,
				Nonce:   &nonce,
				TopicId: topicId,
				InferenceForecastsBundle: &types.InferenceForecastBundle{
					Inference: &types.Inference{
						TopicId:     topicId,
						BlockHeight: nonce.BlockHeight,
						Inferer:     worker2Addr,
						Value:       worker2Value,
					},
					Forecast: nil,
				},
			},
		},
		{
			Sender: worker3Addr,
			WorkerDataBundle: &types.WorkerDataBundle{
				Worker:  worker3Addr,
				Nonce:   &nonce,
				TopicId: topicId,
				InferenceForecastsBundle: &types.InferenceForecastBundle{
					Inference: &types.Inference{
						TopicId:     topicId,
						BlockHeight: nonce.BlockHeight,
						Inferer:     worker3Addr,
						Value:       worker3Value,
					},
					Forecast: nil,
				},
			},
		},
		{
			Sender: worker4Addr,
			WorkerDataBundle: &types.WorkerDataBundle{
				Worker:  worker4Addr,
				Nonce:   &nonce,
				TopicId: topicId,
				InferenceForecastsBundle: &types.InferenceForecastBundle{
					Inference: &types.Inference{
						TopicId:     topicId,
						BlockHeight: nonce.BlockHeight,
						Inferer:     worker4Addr,
						Value:       worker4Value,
					},
					Forecast: nil,
				},
			},
		},
	}

	forecastPayloads = []types.MsgInsertWorkerPayload{
		{
			Sender: worker1Addr,
			WorkerDataBundle: &types.WorkerDataBundle{
				Worker:  worker1Addr,
				Nonce:   &nonce,
				TopicId: topicId,
				InferenceForecastsBundle: &types.InferenceForecastBundle{
					Inference: nil,
					Forecast: &types.Forecast{
						TopicId:     topicId,
						BlockHeight: nonce.BlockHeight,
						Forecaster:  worker1Addr,
						ForecastElements: []*types.ForecastElement{
							{
								Inferer: worker1Addr,
								Value:   worker1Value,
							},
							{
								Inferer: worker2Addr,
								Value:   worker2Value,
							},
							{
								Inferer: worker3Addr,
								Value:   worker3Value,
							},
							{
								Inferer: worker4Addr,
								Value:   worker4Value,
							},
						},
					},
				},
			},
		},
		{
			Sender: worker2Addr,
			WorkerDataBundle: &types.WorkerDataBundle{
				Worker:  worker2Addr,
				Nonce:   &nonce,
				TopicId: topicId,
				InferenceForecastsBundle: &types.InferenceForecastBundle{
					Inference: nil,
					Forecast: &types.Forecast{
						TopicId:     topicId,
						BlockHeight: nonce.BlockHeight,
						Forecaster:  worker2Addr,
						ForecastElements: []*types.ForecastElement{
							{
								Inferer: worker1Addr,
								Value:   worker1Value,
							},
							{
								Inferer: worker2Addr,
								Value:   worker2Value,
							},
							{
								Inferer: worker3Addr,
								Value:   worker3Value,
							},
							{
								Inferer: worker4Addr,
								Value:   worker4Value,
							},
						},
					},
				},
			},
		},
		{
			Sender: worker3Addr,
			WorkerDataBundle: &types.WorkerDataBundle{
				Worker:  worker3Addr,
				Nonce:   &nonce,
				TopicId: topicId,
				InferenceForecastsBundle: &types.InferenceForecastBundle{
					Inference: nil,
					Forecast: &types.Forecast{
						TopicId:     topicId,
						BlockHeight: nonce.BlockHeight,
						Forecaster:  worker3Addr,
						ForecastElements: []*types.ForecastElement{
							{
								Inferer: worker1Addr,
								Value:   worker1Value,
							},
							{
								Inferer: worker2Addr,
								Value:   worker2Value,
							},
							{
								Inferer: worker3Addr,
								Value:   worker3Value,
							},
							{
								Inferer: worker4Addr,
								Value:   worker4Value,
							},
						},
					},
				},
			},
		},
		{
			Sender: worker4Addr,
			WorkerDataBundle: &types.WorkerDataBundle{
				Worker:  worker4Addr,
				Nonce:   &nonce,
				TopicId: topicId,
				InferenceForecastsBundle: &types.InferenceForecastBundle{
					Inference: nil,
					Forecast: &types.Forecast{
						TopicId:     topicId,
						BlockHeight: nonce.BlockHeight,
						Forecaster:  worker4Addr,
						ForecastElements: []*types.ForecastElement{
							{
								Inferer: worker1Addr,
								Value:   worker1Value,
							},
							{
								Inferer: worker2Addr,
								Value:   worker2Value,
							},
							{
								Inferer: worker3Addr,
								Value:   worker3Value,
							},
							{
								Inferer: worker4Addr,
								Value:   worker4Value,
							},
						},
					},
				},
			},
		},
	}

	return inferencePayloads, forecastPayloads, topicId
}

// sign the MsgInsertWorkerPayload message with
// the private key of the worker
func (s *MsgServerTestSuite) signMsgInsertWorkerPayload(
	workerMsg types.MsgInsertWorkerPayload,
	workerPrivateKey secp256k1.PrivKey,
) types.MsgInsertWorkerPayload {
	require := s.Require()

	workerPublicKeyBytes := workerPrivateKey.PubKey().Bytes()

	src := make([]byte, 0)
	src, err := workerMsg.WorkerDataBundle.InferenceForecastsBundle.XXX_Marshal(src, true)
	require.NoError(err, "Marshall reputer value bundle should not return an error")

	sig, err := workerPrivateKey.Sign(src)
	require.NoError(err, "Sign should not return an error")
	workerMsg.WorkerDataBundle.InferencesForecastsBundleSignature = sig
	workerMsg.WorkerDataBundle.Pubkey = hex.EncodeToString(workerPublicKeyBytes)

	return workerMsg
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayload() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()

	workerMsg, topicId := s.setUpMsgInsertWorkerPayload(workerPrivateKey)

	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)

	blockHeight := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight

	forecastsCount0 := s.getCountForecastsAtBlock(topicId, blockHeight)

	ctx = ctx.WithBlockHeight(blockHeight)
	_, err := msgServer.InsertWorkerPayload(ctx, &workerMsg)
	require.NoError(err, "InsertWorkerPayload should not return an error")

	forecastsCount1 := s.getCountForecastsAtBlock(topicId, blockHeight)

	require.Equal(forecastsCount0, 0)
	require.Equal(forecastsCount1, 1)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadNotFailsWithNilInference() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	worker1PrivateKey := secp256k1.GenPrivKey()
	worker2PrivateKey := secp256k1.GenPrivKey()
	worker3PrivateKey := secp256k1.GenPrivKey()
	worker4PrivateKey := secp256k1.GenPrivKey()

	inferMsgs, forecastMsgs, topicId := s.setUpMsgInsertWorkerPayloadFourWorkers(
		worker1PrivateKey, worker2PrivateKey, worker3PrivateKey, worker4PrivateKey)

	// in order for the insertion to do something, there need to be some actual
	// inferences in place to forecast upon
	worker1MsgInference := s.signMsgInsertWorkerPayload(inferMsgs[0], worker1PrivateKey)
	blockHeight := forecastMsgs[0].WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight
	ctx = ctx.WithBlockHeight(blockHeight)
	_, err := msgServer.InsertWorkerPayload(ctx, &worker1MsgInference)
	require.NoError(err)
	worker2MsgInference := s.signMsgInsertWorkerPayload(inferMsgs[1], worker2PrivateKey)
	_, err = msgServer.InsertWorkerPayload(ctx, &worker2MsgInference)
	require.NoError(err)
	worker3MsgInference := s.signMsgInsertWorkerPayload(inferMsgs[2], worker3PrivateKey)
	_, err = msgServer.InsertWorkerPayload(ctx, &worker3MsgInference)
	require.NoError(err)
	worker4MsgInference := s.signMsgInsertWorkerPayload(inferMsgs[3], worker4PrivateKey)
	_, err = msgServer.InsertWorkerPayload(ctx, &worker4MsgInference)
	require.NoError(err)

	worker1MsgForecast := s.signMsgInsertWorkerPayload(forecastMsgs[0], worker1PrivateKey)
	_, err = msgServer.InsertWorkerPayload(ctx, &worker1MsgForecast)
	require.NoError(err)
	worker2MsgForecast := s.signMsgInsertWorkerPayload(forecastMsgs[1], worker2PrivateKey)
	_, err = msgServer.InsertWorkerPayload(ctx, &worker2MsgForecast)
	require.NoError(err)
	worker3MsgForecast := s.signMsgInsertWorkerPayload(forecastMsgs[2], worker3PrivateKey)
	_, err = msgServer.InsertWorkerPayload(ctx, &worker3MsgForecast)
	require.NoError(err)
	worker4MsgForecast := s.signMsgInsertWorkerPayload(forecastMsgs[3], worker4PrivateKey)
	_, err = msgServer.InsertWorkerPayload(ctx, &worker4MsgForecast)
	require.NoError(err)

	forecasts, err := s.emissionsKeeper.GetForecastsAtBlock(ctx, topicId, blockHeight)
	require.NoError(err)
	require.Len(forecasts.Forecasts, 4)
	require.Len(forecasts.Forecasts[0].ForecastElements, 4)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadNotFailsWithNilForecast() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()

	workerMsg, topicId := s.setUpMsgInsertWorkerPayload(workerPrivateKey)

	workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast = nil
	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)

	blockHeight := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Inference.BlockHeight
	ctx = ctx.WithBlockHeight(blockHeight)

	_, err := msgServer.InsertWorkerPayload(ctx, &workerMsg)
	require.NoError(err)

	inferences, err := s.emissionsKeeper.GetInferencesAtBlock(ctx, topicId, blockHeight)
	require.NoError(err)
	require.Equal(len(inferences.Inferences), 1)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadFailsWithNilInferenceAndForecast() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()

	workerMsg, _ := s.setUpMsgInsertWorkerPayload(workerPrivateKey)
	blockHeight := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight
	ctx = ctx.WithBlockHeight(blockHeight)
	// BEGIN MODIFICATION
	workerMsg.WorkerDataBundle.InferenceForecastsBundle.Inference = nil
	workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast = nil
	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)
	// END MODIFICATION

	_, err := msgServer.InsertWorkerPayload(ctx, &workerMsg)
	require.ErrorIs(err, sdkerrors.ErrInvalidRequest)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadFailsWithoutSignature() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()

	workerMsg, _ := s.setUpMsgInsertWorkerPayload(workerPrivateKey)

	// BEGIN MODIFICATION
	// workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)
	// END MODIFICATION

	_, err := msgServer.InsertWorkerPayload(ctx, &workerMsg)
	require.ErrorIs(err, sdkerrors.ErrInvalidRequest)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadFailsWithMismatchedTopicId() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()

	workerMsg, _ := s.setUpMsgInsertWorkerPayload(workerPrivateKey)

	// BEGIN MODIFICATION
	workerMsg.WorkerDataBundle.InferenceForecastsBundle.Inference.TopicId = 123
	// END MODIFICATION

	blockHeight := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Inference.BlockHeight
	ctx = ctx.WithBlockHeight(blockHeight)
	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)

	_, err := msgServer.InsertWorkerPayload(ctx, &workerMsg)
	require.ErrorIs(err, types.ErrInvalidTopicId)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadFailsWithUnregisteredInferer() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()

	workerMsg, topicId := s.setUpMsgInsertWorkerPayload(workerPrivateKey)

	// BEGIN MODIFICATION
	inferer := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Inference.Inferer

	unregisterMsg := &types.MsgRemoveRegistration{
		Sender:    inferer,
		TopicId:   topicId,
		IsReputer: false,
	}

	_, err := msgServer.RemoveRegistration(ctx, unregisterMsg)
	require.NoError(err)

	// END MODIFICATION
	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)
	blockHeight := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Inference.BlockHeight
	ctx = ctx.WithBlockHeight(blockHeight)

	_, err = msgServer.InsertWorkerPayload(ctx, &workerMsg)
	require.ErrorIs(err, types.ErrAddressNotRegistered)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerActiveSetBounds() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	worker1PrivateKey := secp256k1.GenPrivKey()
	worker2PrivateKey := secp256k1.GenPrivKey()
	worker3PrivateKey := secp256k1.GenPrivKey()
	worker4PrivateKey := secp256k1.GenPrivKey()

	adminPrivateKey := secp256k1.GenPrivKey()
	adminAddr := sdk.AccAddress(adminPrivateKey.PubKey().Address())
	_ = s.emissionsKeeper.AddWhitelistAdmin(s.ctx, adminAddr.String())

	newParams := &types.OptionalParams{
		MaxTopInferersToReward:    []uint64{3},
		MaxTopForecastersToReward: []uint64{3},
	}

	updateMsg := &types.MsgUpdateParams{
		Sender: adminAddr.String(),
		Params: newParams,
	}

	_, err := s.msgServer.UpdateParams(s.ctx, updateMsg)
	require.NoError(err, "UpdateParams should not return an error")

	inferMsgs, forecastMsgs, topicId := s.setUpMsgInsertWorkerPayloadFourWorkers(
		worker1PrivateKey, worker2PrivateKey, worker3PrivateKey, worker4PrivateKey)

	blockHeight := inferMsgs[0].WorkerDataBundle.InferenceForecastsBundle.Inference.BlockHeight

	param, _ := s.emissionsKeeper.GetParams(ctx)

	ctx = ctx.WithBlockHeight(blockHeight)

	inferer1 := inferMsgs[0].WorkerDataBundle.InferenceForecastsBundle.Inference.Inferer
	inferer2 := inferMsgs[1].WorkerDataBundle.InferenceForecastsBundle.Inference.Inferer
	inferer3 := inferMsgs[2].WorkerDataBundle.InferenceForecastsBundle.Inference.Inferer
	inferer4 := inferMsgs[3].WorkerDataBundle.InferenceForecastsBundle.Inference.Inferer

	score1 := types.Score{TopicId: topicId, BlockHeight: blockHeight, Address: inferer1, Score: alloraMath.NewDecFromInt64(95)}
	score2 := types.Score{TopicId: topicId, BlockHeight: blockHeight, Address: inferer2, Score: alloraMath.NewDecFromInt64(90)}
	score3 := types.Score{TopicId: topicId, BlockHeight: blockHeight, Address: inferer3, Score: alloraMath.NewDecFromInt64(80)}
	score4 := types.Score{TopicId: topicId, BlockHeight: blockHeight, Address: inferer4, Score: alloraMath.NewDecFromInt64(99)}

	_ = s.emissionsKeeper.SetInfererScoreEma(ctx, topicId, inferer1, score1)
	_ = s.emissionsKeeper.SetInfererScoreEma(ctx, topicId, inferer2, score2)
	_ = s.emissionsKeeper.SetInfererScoreEma(ctx, topicId, inferer3, score3)
	_ = s.emissionsKeeper.SetInfererScoreEma(ctx, topicId, inferer4, score4)

	inferMsg1 := s.signMsgInsertWorkerPayload(inferMsgs[0], worker1PrivateKey)
	_, err = msgServer.InsertWorkerPayload(ctx, &inferMsg1)
	require.NoError(err, "InsertWorkerPayload should not return an error")
	inferences, err := s.emissionsKeeper.GetInferencesAtBlock(ctx, topicId, blockHeight)
	require.NoError(err)
	require.Len(inferences.Inferences, 1)
	inferMsg2 := s.signMsgInsertWorkerPayload(inferMsgs[1], worker2PrivateKey)
	_, err = msgServer.InsertWorkerPayload(ctx, &inferMsg2)
	require.NoError(err, "InsertWorkerPayload should not return an error")
	inferences, err = s.emissionsKeeper.GetInferencesAtBlock(ctx, topicId, blockHeight)
	require.NoError(err)
	require.Len(inferences.Inferences, 2)
	inferMsg3 := s.signMsgInsertWorkerPayload(inferMsgs[2], worker3PrivateKey)
	_, err = msgServer.InsertWorkerPayload(ctx, &inferMsg3)
	require.NoError(err, "InsertWorkerPayload should not return an error")
	inferences, err = s.emissionsKeeper.GetInferencesAtBlock(ctx, topicId, blockHeight)
	require.NoError(err)
	require.Len(inferences.Inferences, 3)
	inferMsg4 := s.signMsgInsertWorkerPayload(inferMsgs[3], worker4PrivateKey)
	_, err = msgServer.InsertWorkerPayload(ctx, &inferMsg4)
	require.NoError(err, "InsertWorkerPayload should not return an error")
	inferences, err = s.emissionsKeeper.GetInferencesAtBlock(ctx, topicId, blockHeight)
	require.NoError(err)
	require.Len(inferences.Inferences, 3)

	forecastMsg1 := s.signMsgInsertWorkerPayload(forecastMsgs[0], worker1PrivateKey)
	_, err = msgServer.InsertWorkerPayload(ctx, &forecastMsg1)
	require.NoError(err, "InsertWorkerPayload should not return an error")
	forecasts, err := s.emissionsKeeper.GetForecastsAtBlock(ctx, topicId, blockHeight)
	require.NoError(err)
	require.Len(forecasts.Forecasts, 1)
	forecastMsg2 := s.signMsgInsertWorkerPayload(forecastMsgs[1], worker2PrivateKey)
	_, err = msgServer.InsertWorkerPayload(ctx, &forecastMsg2)
	require.NoError(err, "InsertWorkerPayload should not return an error")
	forecasts, err = s.emissionsKeeper.GetForecastsAtBlock(ctx, topicId, blockHeight)
	require.NoError(err)
	require.Len(forecasts.Forecasts, 2)
	forecastMsg3 := s.signMsgInsertWorkerPayload(forecastMsgs[2], worker3PrivateKey)
	_, err = msgServer.InsertWorkerPayload(ctx, &forecastMsg3)
	require.NoError(err, "InsertWorkerPayload should not return an error")
	forecasts, err = s.emissionsKeeper.GetForecastsAtBlock(ctx, topicId, blockHeight)
	require.NoError(err)
	require.Len(forecasts.Forecasts, 3)
	forecastMsg4 := s.signMsgInsertWorkerPayload(forecastMsgs[3], worker4PrivateKey)
	_, err = msgServer.InsertWorkerPayload(ctx, &forecastMsg4)
	require.NoError(err, "InsertWorkerPayload should not return an error")
	forecasts, err = s.emissionsKeeper.GetForecastsAtBlock(ctx, topicId, blockHeight)
	require.NoError(err)

	require.Len(forecasts.Forecasts, int(param.MaxTopForecastersToReward)) //nolint:gosec //G115: integer overflow conversion uint64 -> int
	require.Equal(forecasts.Forecasts[0].ForecastElements[0].Inferer, inferer1)
	require.Equal(forecasts.Forecasts[0].ForecastElements[1].Inferer, inferer2)
	require.Equal(forecasts.Forecasts[0].ForecastElements[2].Inferer, inferer4)
}

func (s *MsgServerTestSuite) getCountForecastsAtBlock(topicId uint64, blockHeight int64) int {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	forecastsAtBlock, err := keeper.GetForecastsAtBlock(ctx, topicId, blockHeight)
	if err != nil {
		return 0
	}
	return len(forecastsAtBlock.Forecasts)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadFailsWithMismatchedForecastTopicId() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()

	workerMsg, _ := s.setUpMsgInsertWorkerPayload(workerPrivateKey)

	// BEGIN MODIFICATION
	originalTopicId := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.TopicId
	workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.TopicId = 123
	// END MODIFICATION

	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)

	blockHeight := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight

	forecastsCount0 := s.getCountForecastsAtBlock(originalTopicId, blockHeight)
	require.Equal(forecastsCount0, 0)

	ctx = ctx.WithBlockHeight(blockHeight)
	_, err := msgServer.InsertWorkerPayload(ctx, &workerMsg)
	require.ErrorIs(err, types.ErrInvalidTopicId)

	forecastsCount1 := s.getCountForecastsAtBlock(originalTopicId, blockHeight)
	require.Equal(forecastsCount1, 0)

	// Also not added on the changed topicId
	forecastsCountNew := s.getCountForecastsAtBlock(123, blockHeight)
	require.Equal(forecastsCountNew, 0)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadFailsWithUnregisteredForecaster() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()

	workerMsg, topicId := s.setUpMsgInsertWorkerPayload(workerPrivateKey)

	// BEGIN MODIFICATION
	forecaster := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.Forecaster

	unregisterMsg := &types.MsgRemoveRegistration{
		Sender:    forecaster,
		TopicId:   topicId,
		IsReputer: false,
	}

	_, err := msgServer.RemoveRegistration(ctx, unregisterMsg)
	require.NoError(err)

	// END MODIFICATION

	blockHeight := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight

	forecastsCount0 := s.getCountForecastsAtBlock(topicId, blockHeight)
	require.Equal(forecastsCount0, 0)

	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)

	ctx = ctx.WithBlockHeight(blockHeight)
	_, err = msgServer.InsertWorkerPayload(ctx, &workerMsg)
	require.ErrorIs(err, types.ErrAddressNotRegistered)

	forecastsCount1 := s.getCountForecastsAtBlock(topicId, blockHeight)
	require.Equal(forecastsCount1, 0)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadFiltersDuplicateForecastElements() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()
	workerMsg, topicId := s.setUpMsgInsertWorkerPayload(workerPrivateKey)

	// BEGIN MODIFICATION
	forecast := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast
	originalElement := forecast.ForecastElements[0]
	duplicateElement := &types.ForecastElement{
		Inferer: originalElement.Inferer,
		Value:   originalElement.Value,
	}
	forecast.ForecastElements = append(forecast.ForecastElements, duplicateElement)
	// END MODIFICATION

	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)

	blockHeight := forecast.BlockHeight
	forecastsCount0 := s.getCountForecastsAtBlock(topicId, blockHeight)

	ctx = ctx.WithBlockHeight(blockHeight)
	_, err := msgServer.InsertWorkerPayload(ctx, &workerMsg)
	require.NoError(err, "InsertWorkerPayload should not return an error")

	// Check the forecast count to ensure duplicates were filtered out
	forecastsCount1 := s.getCountForecastsAtBlock(topicId, blockHeight)
	require.Equal(forecastsCount0+1, forecastsCount1, "Forecast count should increase by one")

	storedForecasts, err := s.emissionsKeeper.GetForecastsAtBlock(ctx, topicId, blockHeight)
	require.NoError(err, "GetForecastsAtBlock should not return an error")

	for _, forecast := range storedForecasts.Forecasts {
		infererMap := make(map[string]bool)
		for _, el := range forecast.ForecastElements {
			_, exists := infererMap[el.Inferer]
			require.False(exists, "Each inferer should appear only once in ForecastElements")
			infererMap[el.Inferer] = true
		}
	}
}

func (s *MsgServerTestSuite) TestInsertingHugeBundleWorkerPayloadFails() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	keeper := s.emissionsKeeper
	topicId := uint64(0)
	nonce := types.Nonce{BlockHeight: 1}

	// Define sample OffchainNode information for a worker
	workerInfo := types.OffchainNode{
		Owner:       "worker-owner-sample",
		NodeAddress: "worker-node-address-sample",
	}
	// Mock setup for addresses

	reputerPrivateKey := secp256k1.GenPrivKey()
	reputerAddr := sdk.AccAddress(reputerPrivateKey.PubKey().Address()).String()

	workerPrivateKey := secp256k1.GenPrivKey()
	workerPublicKeyBytes := workerPrivateKey.PubKey().Bytes()
	workerAddr := sdk.AccAddress(workerPrivateKey.PubKey().Address()).String()

	InfererPrivateKey := secp256k1.GenPrivKey()
	InfererAddr := sdk.AccAddress(InfererPrivateKey.PubKey().Address()).String()

	ForecasterPrivateKey := secp256k1.GenPrivKey()
	ForecasterAddr := sdk.AccAddress(ForecasterPrivateKey.PubKey().Address()).String()

	moduleParams, err := keeper.GetParams(ctx)
	require.NoError(err)

	// Create topic 0 and register reputer in it
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, moduleParams.RegistrationFee)
	err = keeper.AddWorkerNonce(ctx, 0, &nonce)
	require.NoError(err)
	err = keeper.InsertWorker(ctx, topicId, InfererAddr, workerInfo)
	require.NoError(err)
	err = keeper.InsertWorker(ctx, topicId, ForecasterAddr, workerInfo)
	require.NoError(err)
	err = s.emissionsKeeper.SetTopic(ctx, topicId, types.Topic{Id: topicId})
	require.NoError(err)

	forecastElements := []*types.ForecastElement{}
	for i := 0; i < 1000000; i++ {
		forecastElements = append(forecastElements, &types.ForecastElement{
			Inferer: InfererAddr,
			Value:   alloraMath.NewDecFromInt64(100),
		})
	}

	// Create a MsgInsertWorkerPayload message
	workerMsg := &types.MsgInsertWorkerPayload{
		Sender: workerAddr,
		WorkerDataBundle: &types.WorkerDataBundle{
			Worker: InfererAddr,
			InferenceForecastsBundle: &types.InferenceForecastBundle{
				Inference: &types.Inference{
					TopicId:     topicId,
					BlockHeight: nonce.BlockHeight,
					Inferer:     InfererAddr,
					Value:       alloraMath.NewDecFromInt64(100),
				},
				Forecast: &types.Forecast{
					TopicId:          0,
					BlockHeight:      nonce.BlockHeight,
					Forecaster:       ForecasterAddr,
					ForecastElements: forecastElements,
				},
			},
		},
	}

	src := make([]byte, 0)
	src, err = workerMsg.WorkerDataBundle.InferenceForecastsBundle.XXX_Marshal(src, true)
	require.NoError(err, "Marshall reputer value bundle should not return an error")

	sig, err := workerPrivateKey.Sign(src)
	require.NoError(err, "Sign should not return an error")
	workerMsg.WorkerDataBundle.InferencesForecastsBundleSignature = sig
	workerMsg.WorkerDataBundle.Pubkey = hex.EncodeToString(workerPublicKeyBytes)
	_, err = msgServer.InsertWorkerPayload(ctx, workerMsg)
	require.ErrorIs(err, types.ErrQueryTooLarge)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadVerifyFailed() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	keeper := s.emissionsKeeper
	topicId := uint64(0)
	nonce := types.Nonce{BlockHeight: 1}

	// Define sample OffchainNode information for a worker
	workerInfo := types.OffchainNode{
		Owner:       "worker-owner-sample",
		NodeAddress: "worker-node-address-sample",
	}
	// Mock setup for addresses

	reputerPrivateKey := secp256k1.GenPrivKey()
	reputerAddr := sdk.AccAddress(reputerPrivateKey.PubKey().Address()).String()

	workerPrivateKey := secp256k1.GenPrivKey()
	workerAddr := sdk.AccAddress(workerPrivateKey.PubKey().Address()).String()

	InfererPrivateKey := secp256k1.GenPrivKey()
	InfererAddr := sdk.AccAddress(InfererPrivateKey.PubKey().Address()).String()

	Inferer2PrivateKey := secp256k1.GenPrivKey()
	Inferer2Addr := sdk.AccAddress(Inferer2PrivateKey.PubKey().Address()).String()

	ForecasterPrivateKey := secp256k1.GenPrivKey()
	ForecasterAddr := sdk.AccAddress(ForecasterPrivateKey.PubKey().Address()).String()

	moduleParams, err := keeper.GetParams(ctx)
	require.NoError(err)

	// Create topic 0 and register reputer in it
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, moduleParams.RegistrationFee)
	err = keeper.AddWorkerNonce(ctx, 0, &nonce)
	require.NoError(err)
	err = keeper.InsertWorker(ctx, topicId, InfererAddr, workerInfo)
	require.NoError(err)
	err = keeper.InsertWorker(ctx, topicId, ForecasterAddr, workerInfo)
	require.NoError(err)
	err = s.emissionsKeeper.SetTopic(ctx, topicId, types.Topic{Id: topicId})
	require.NoError(err)

	// Create a MsgInsertWorkerPayload message
	workerMsg := &types.MsgInsertWorkerPayload{
		Sender: workerAddr,
		WorkerDataBundle: &types.WorkerDataBundle{
			Worker:  InfererAddr,
			TopicId: topicId,
			Nonce:   &nonce,
			InferenceForecastsBundle: &types.InferenceForecastBundle{
				Inference: &types.Inference{
					TopicId:     topicId,
					BlockHeight: nonce.BlockHeight,
					Inferer:     InfererAddr,
					Value:       alloraMath.NewDecFromInt64(100),
				},
				Forecast: &types.Forecast{
					TopicId:     0,
					BlockHeight: nonce.BlockHeight,
					Forecaster:  ForecasterAddr,
					ForecastElements: []*types.ForecastElement{
						{
							Inferer: InfererAddr,
							Value:   alloraMath.NewDecFromInt64(100),
						},
						{
							Inferer: Inferer2Addr,
							Value:   alloraMath.NewDecFromInt64(100),
						},
					},
				},
			},
		},
	}

	_, err = msgServer.InsertWorkerPayload(ctx, workerMsg)
	require.ErrorIs(err, sdkerrors.ErrInvalidRequest)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadCapOnMaxElementsPerForecast() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	keeper := s.emissionsKeeper

	workerPrivateKey := secp256k1.GenPrivKey()
	workerPrivateKey2 := secp256k1.GenPrivKey()
	workerPrivateKey3 := secp256k1.GenPrivKey()
	workerPrivateKey4 := secp256k1.GenPrivKey()
	adminPrivateKey := secp256k1.GenPrivKey()
	adminAddr := sdk.AccAddress(adminPrivateKey.PubKey().Address())
	_ = keeper.AddWhitelistAdmin(s.ctx, adminAddr.String())

	newParams := &types.OptionalParams{
		MaxElementsPerForecast: []uint64{3},
	}

	updateMsg := &types.MsgUpdateParams{
		Sender: adminAddr.String(),
		Params: newParams,
	}

	_, err := s.msgServer.UpdateParams(s.ctx, updateMsg)
	require.NoError(err, "UpdateParams should not return an error")

	inferMsgs, forecastMsgs, topicId := s.setUpMsgInsertWorkerPayloadFourWorkers(workerPrivateKey, workerPrivateKey2, workerPrivateKey3, workerPrivateKey4)

	blockHeight := inferMsgs[0].WorkerDataBundle.InferenceForecastsBundle.Inference.BlockHeight
	ctx = ctx.WithBlockHeight(blockHeight)

	worker1 := inferMsgs[0].WorkerDataBundle.InferenceForecastsBundle.Inference.Inferer
	worker2 := inferMsgs[1].WorkerDataBundle.InferenceForecastsBundle.Inference.Inferer
	worker3 := inferMsgs[2].WorkerDataBundle.InferenceForecastsBundle.Inference.Inferer
	worker4 := inferMsgs[3].WorkerDataBundle.InferenceForecastsBundle.Inference.Inferer

	score1 := types.Score{TopicId: topicId, BlockHeight: blockHeight, Address: worker1, Score: alloraMath.NewDecFromInt64(95)}
	score2 := types.Score{TopicId: topicId, BlockHeight: blockHeight, Address: worker2, Score: alloraMath.NewDecFromInt64(90)}
	score3 := types.Score{TopicId: topicId, BlockHeight: blockHeight, Address: worker3, Score: alloraMath.NewDecFromInt64(80)}
	score4 := types.Score{TopicId: topicId, BlockHeight: blockHeight, Address: worker4, Score: alloraMath.NewDecFromInt64(50)}
	_ = keeper.SetInfererScoreEma(ctx, topicId, worker1, score1)
	_ = keeper.SetInfererScoreEma(ctx, topicId, worker2, score2)
	_ = keeper.SetInfererScoreEma(ctx, topicId, worker3, score3)
	_ = keeper.SetInfererScoreEma(ctx, topicId, worker4, score4)

	inferMsgs[0] = s.signMsgInsertWorkerPayload(inferMsgs[0], workerPrivateKey)
	_, err = msgServer.InsertWorkerPayload(ctx, &inferMsgs[0])
	require.NoError(err)
	inferMsgs[1] = s.signMsgInsertWorkerPayload(inferMsgs[1], workerPrivateKey2)
	_, err = msgServer.InsertWorkerPayload(ctx, &inferMsgs[1])
	require.NoError(err)
	inferMsgs[2] = s.signMsgInsertWorkerPayload(inferMsgs[2], workerPrivateKey3)
	_, err = msgServer.InsertWorkerPayload(ctx, &inferMsgs[2])
	require.NoError(err)
	inferMsgs[3] = s.signMsgInsertWorkerPayload(inferMsgs[3], workerPrivateKey4)
	_, err = msgServer.InsertWorkerPayload(ctx, &inferMsgs[3])
	require.NoError(err)

	forecastMsgs[0] = s.signMsgInsertWorkerPayload(forecastMsgs[0], workerPrivateKey)
	_, err = msgServer.InsertWorkerPayload(ctx, &forecastMsgs[0])
	require.NoError(err)
	forecastMsgs[1] = s.signMsgInsertWorkerPayload(forecastMsgs[1], workerPrivateKey2)
	_, err = msgServer.InsertWorkerPayload(ctx, &forecastMsgs[1])
	require.NoError(err)
	forecastMsgs[2] = s.signMsgInsertWorkerPayload(forecastMsgs[2], workerPrivateKey3)
	_, err = msgServer.InsertWorkerPayload(ctx, &forecastMsgs[2])
	require.NoError(err)
	forecastMsgs[3] = s.signMsgInsertWorkerPayload(forecastMsgs[3], workerPrivateKey4)
	_, err = msgServer.InsertWorkerPayload(ctx, &forecastMsgs[3])
	require.NoError(err)

	forecastsCount0 := s.getCountForecastsAtBlock(topicId, blockHeight)
	require.Equal(forecastsCount0, 4)
	forecastsAtBlock, err := keeper.GetForecastsAtBlock(ctx, topicId, blockHeight)
	require.NoError(err)
	require.Len(forecastsAtBlock.Forecasts, 4)
	require.Len(forecastsAtBlock.Forecasts[0].ForecastElements, 3)
	require.Len(forecastsAtBlock.Forecasts[1].ForecastElements, 3)
	require.Len(forecastsAtBlock.Forecasts[2].ForecastElements, 3)
	require.Len(forecastsAtBlock.Forecasts[3].ForecastElements, 3)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadWithInferencesRepeatedlyOverwritesPreviousValue() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	keeper := s.emissionsKeeper

	workerPrivateKey := secp256k1.GenPrivKey()

	workerMsg, topicId := s.setUpMsgInsertWorkerPayload(workerPrivateKey)
	// BEGIN MODIFICATION
	workerMsg.WorkerDataBundle.InferenceForecastsBundle.Inference.Value = alloraMath.NewDecFromInt64(100)
	// END MODIFICATION
	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)

	blockHeight := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Inference.BlockHeight
	ctx = ctx.WithBlockHeight(blockHeight)

	_, err := msgServer.InsertWorkerPayload(ctx, &workerMsg)
	require.NoError(err, "InsertWorkerPayload should not return an error")

	inferences, err := keeper.GetInferencesAtBlock(ctx, topicId, blockHeight)
	require.NoError(err)
	require.Equal(len(inferences.Inferences), 1)
	require.Equal(inferences.Inferences[0].Value, alloraMath.NewDecFromInt64(100))

	// Repeat the same inference with a different inference value and check if it overwrites the previous value
	// BEGIN MODIFICATION
	workerMsg.WorkerDataBundle.InferenceForecastsBundle.Inference.Value = alloraMath.NewDecFromInt64(200)
	// END MODIFICATION
	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)

	_, err = msgServer.InsertWorkerPayload(ctx, &workerMsg)
	require.NoError(err, "InsertWorkerPayload should not return an error")

	inferences, err = keeper.GetInferencesAtBlock(ctx, topicId, blockHeight)
	require.NoError(err)
	require.Equal(len(inferences.Inferences), 1)
	require.Equal(inferences.Inferences[0].Value, alloraMath.NewDecFromInt64(200))
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadWithForecastRepeatedlyOverwritesPreviousValue() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	keeper := s.emissionsKeeper

	worker1PrivateKey := secp256k1.GenPrivKey()
	worker2PrivateKey := secp256k1.GenPrivKey()
	worker3PrivateKey := secp256k1.GenPrivKey()
	worker4PrivateKey := secp256k1.GenPrivKey()

	inferMsgs, forecastMsgs, topicId := s.setUpMsgInsertWorkerPayloadFourWorkers(worker1PrivateKey, worker2PrivateKey, worker3PrivateKey, worker4PrivateKey)

	// upload an inference so forecasts can be created
	infer1Msg := s.signMsgInsertWorkerPayload(inferMsgs[0], worker1PrivateKey)
	blockHeight := infer1Msg.WorkerDataBundle.InferenceForecastsBundle.Inference.BlockHeight
	ctx = ctx.WithBlockHeight(blockHeight)
	_, err := msgServer.InsertWorkerPayload(ctx, &infer1Msg)
	require.NoError(err, "InsertWorkerPayload should not return an error")

	// upload a forecast to later overwrite
	forecast1Msg := s.signMsgInsertWorkerPayload(forecastMsgs[0], worker1PrivateKey)
	_, err = msgServer.InsertWorkerPayload(ctx, &forecast1Msg)
	require.NoError(err, "InsertWorkerPayload should not return an error")

	// upload the same forecast again with a different value
	forecast1Msg.WorkerDataBundle.InferenceForecastsBundle.Forecast.ForecastElements[0].Value = alloraMath.NewDecFromInt64(200)
	forecast1Msg = s.signMsgInsertWorkerPayload(forecast1Msg, worker1PrivateKey)
	_, err = msgServer.InsertWorkerPayload(ctx, &forecast1Msg)
	require.NoError(err, "InsertWorkerPayload should not return an error")

	forecasts, err := keeper.GetForecastsAtBlock(ctx, topicId, blockHeight)
	require.NoError(err)
	require.Len(forecasts.Forecasts[0].ForecastElements, 1)
	require.Equal(forecasts.Forecasts[0].ForecastElements[0].Value, alloraMath.NewDecFromInt64(200))
}
