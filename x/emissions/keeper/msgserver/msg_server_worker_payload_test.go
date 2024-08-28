package msgserver_test

import (
	"encoding/hex"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	InfererAddr := getNewAddress()
	Inferer2Addr := getNewAddress()
	Inferer3Addr := getNewAddress()
	Inferer4Addr := getNewAddress()
	ForecasterAddr := getNewAddress()

	workerAddr := sdk.AccAddress(workerPrivateKey.PubKey().Address()).String()

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
	err = keeper.InsertWorker(ctx, topicId, ForecasterAddr, workerInfo)
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
					Inferer:     Inferer2Addr,
					Value:       alloraMath.NewDecFromInt64(100),
				},
				Forecast: &types.Forecast{
					TopicId:     topicId,
					BlockHeight: nonce.BlockHeight,
					Forecaster:  ForecasterAddr,
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

	workerPrivateKey := secp256k1.GenPrivKey()

	workerMsg, topicId := s.setUpMsgInsertWorkerPayload(workerPrivateKey)

	workerMsg.WorkerDataBundle.InferenceForecastsBundle.Inference = nil
	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)

	blockHeight := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight
	ctx = ctx.WithBlockHeight(blockHeight)

	_, err := msgServer.InsertWorkerPayload(ctx, &workerMsg)
	require.NoError(err)

	forecasts, err := s.emissionsKeeper.GetForecastsAtBlock(ctx, topicId, blockHeight)
	require.NoError(err)
	require.Equal(len(forecasts.Forecasts[0].ForecastElements), 4)
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
	require.ErrorIs(err, types.ErrInvalidWorkerData)
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
	require.ErrorIs(err, types.ErrInvalidWorkerData)
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

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadWithFewTopElementsPerForecast() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()
	adminPrivateKey := secp256k1.GenPrivKey()
	adminAddr := sdk.AccAddress(adminPrivateKey.PubKey().Address())
	_ = s.emissionsKeeper.AddWhitelistAdmin(s.ctx, adminAddr.String())

	newParams := &types.OptionalParams{
		MaxElementsPerForecast: []uint64{3},
	}

	updateMsg := &types.MsgUpdateParams{
		Sender: adminAddr.String(),
		Params: newParams,
	}

	_, err := s.msgServer.UpdateParams(s.ctx, updateMsg)
	require.NoError(err, "UpdateParams should not return an error")

	workerMsg, topicId := s.setUpMsgInsertWorkerPayload(workerPrivateKey)

	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)

	blockHeight := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight

	param, _ := s.emissionsKeeper.GetParams(ctx)

	ctx = ctx.WithBlockHeight(blockHeight)

	inferer1 := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.ForecastElements[0].Inferer
	inferer2 := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.ForecastElements[1].Inferer
	inferer3 := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.ForecastElements[2].Inferer
	inferer4 := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.ForecastElements[3].Inferer

	score1 := types.Score{TopicId: topicId, BlockHeight: blockHeight, Address: inferer1, Score: alloraMath.NewDecFromInt64(95)}
	score2 := types.Score{TopicId: topicId, BlockHeight: blockHeight, Address: inferer2, Score: alloraMath.NewDecFromInt64(90)}
	score3 := types.Score{TopicId: topicId, BlockHeight: blockHeight, Address: inferer3, Score: alloraMath.NewDecFromInt64(80)}
	score4 := types.Score{TopicId: topicId, BlockHeight: blockHeight, Address: inferer4, Score: alloraMath.NewDecFromInt64(99)}

	_ = s.emissionsKeeper.SetInfererScoreEma(ctx, topicId, inferer1, score1)
	_ = s.emissionsKeeper.SetInfererScoreEma(ctx, topicId, inferer2, score2)
	_ = s.emissionsKeeper.SetInfererScoreEma(ctx, topicId, inferer3, score3)
	_ = s.emissionsKeeper.SetInfererScoreEma(ctx, topicId, inferer4, score4)

	_, err = msgServer.InsertWorkerPayload(ctx, &workerMsg)
	require.NoError(err, "InsertWorkerPayload should not return an error")

	forecasts, err := s.emissionsKeeper.GetForecastsAtBlock(ctx, topicId, blockHeight)

	require.NoError(err)

	require.Equal(uint64(len(forecasts.Forecasts[0].ForecastElements)), param.MaxElementsPerForecast)
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
	require.ErrorIs(err, types.ErrInvalidWorkerData)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadWithLowScoreForecastsAreRejected() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	keeper := s.emissionsKeeper

	workerPrivateKey := secp256k1.GenPrivKey()
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

	workerMsg, topicId := s.setUpMsgInsertWorkerPayload(workerPrivateKey)

	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)

	blockHeight := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight
	ctx = ctx.WithBlockHeight(blockHeight)

	inferer1 := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.ForecastElements[0].Inferer
	inferer2 := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.ForecastElements[1].Inferer
	inferer3 := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.ForecastElements[2].Inferer
	inferer4 := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.ForecastElements[3].Inferer

	score1 := types.Score{TopicId: topicId, BlockHeight: blockHeight, Address: inferer1, Score: alloraMath.NewDecFromInt64(95)}
	score2 := types.Score{TopicId: topicId, BlockHeight: blockHeight, Address: inferer2, Score: alloraMath.NewDecFromInt64(90)}
	score3 := types.Score{TopicId: topicId, BlockHeight: blockHeight, Address: inferer3, Score: alloraMath.NewDecFromInt64(80)}
	score4 := types.Score{TopicId: topicId, BlockHeight: blockHeight, Address: inferer4, Score: alloraMath.NewDecFromInt64(50)}

	_ = keeper.SetInfererScoreEma(ctx, topicId, inferer1, score1)
	_ = keeper.SetInfererScoreEma(ctx, topicId, inferer2, score2)
	_ = keeper.SetInfererScoreEma(ctx, topicId, inferer3, score3)
	_ = keeper.SetInfererScoreEma(ctx, topicId, inferer4, score4)

	_, err = msgServer.InsertWorkerPayload(ctx, &workerMsg)
	require.NoError(err, "InsertWorkerPayload should not return an error even if the forecast elements are below the threshold")

	forecastsCount0 := s.getCountForecastsAtBlock(topicId, blockHeight)
	require.Equal(forecastsCount0, 1)
	forecastsAtBlock, err := keeper.GetForecastsAtBlock(ctx, topicId, blockHeight)
	require.NoError(err)
	require.Equal(len(forecastsAtBlock.Forecasts[0].ForecastElements), 3)
	require.Equal(forecastsAtBlock.Forecasts[0].ForecastElements[0].Inferer, inferer1)
	require.Equal(forecastsAtBlock.Forecasts[0].ForecastElements[1].Inferer, inferer2)
	require.Equal(forecastsAtBlock.Forecasts[0].ForecastElements[2].Inferer, inferer3)
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

	workerPrivateKey := secp256k1.GenPrivKey()

	workerMsg, topicId := s.setUpMsgInsertWorkerPayload(workerPrivateKey)
	// BEGIN MODIFICATION
	workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.ForecastElements[0].Value = alloraMath.NewDecFromInt64(100)
	// END MODIFICATION
	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)

	blockHeight := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight
	ctx = ctx.WithBlockHeight(blockHeight)

	_, err := msgServer.InsertWorkerPayload(ctx, &workerMsg)
	require.NoError(err, "InsertWorkerPayload should not return an error")

	forecasts, err := keeper.GetForecastsAtBlock(ctx, topicId, blockHeight)
	require.NoError(err)
	require.Equal(len(forecasts.Forecasts[0].ForecastElements), 4)
	require.Equal(forecasts.Forecasts[0].ForecastElements[0].Value, alloraMath.NewDecFromInt64(100))

	// Repeat the same forecast with a different forecast value and check if it overwrites the previous value
	// BEGIN MODIFICATION
	workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.ForecastElements[0].Value = alloraMath.NewDecFromInt64(200)
	// END MODIFICATION
	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)

	_, err = msgServer.InsertWorkerPayload(ctx, &workerMsg)
	require.NoError(err, "InsertWorkerPayload should not return an error")

	forecasts, err = keeper.GetForecastsAtBlock(ctx, topicId, blockHeight)
	require.NoError(err)
	require.Equal(len(forecasts.Forecasts[0].ForecastElements), 4)
	require.Equal(forecasts.Forecasts[0].ForecastElements[0].Value, alloraMath.NewDecFromInt64(200))
}
