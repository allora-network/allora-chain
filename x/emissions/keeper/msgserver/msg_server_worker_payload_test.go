package msgserver_test

import (
	"encoding/hex"

	cosmosMath "cosmossdk.io/math"
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
	topicId := uint64(0)

	// Define sample OffchainNode information for a worker
	workerInfo := types.OffchainNode{
		Owner:       "worker-owner-sample",
		NodeAddress: "worker-node-address-sample",
	}

	// Mock setup for addresses
	reputerAddr := getNewAddress()
	InfererAddr := getNewAddress()
	Inferer2Addr := getNewAddress()
	ForecasterAddr := getNewAddress()

	workerAddr := sdk.AccAddress(workerPrivateKey.PubKey().Address()).String()

	registrationInitialStake := cosmosMath.NewInt(100)

	// Create topic 0 and register reputer in it
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)
	keeper.AddWorkerNonce(ctx, topicId, &nonce)
	keeper.InsertWorker(ctx, topicId, InfererAddr, workerInfo)
	keeper.InsertWorker(ctx, topicId, Inferer2Addr, workerInfo)
	keeper.InsertWorker(ctx, topicId, ForecasterAddr, workerInfo)
	s.emissionsKeeper.SetTopic(ctx, topicId, types.Topic{Id: topicId})

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
					},
				},
			},
		},
	}

	return workerMsg, topicId
}

func (s *MsgServerTestSuite) signMsgInsertWorkerPayload(workerMsg types.MsgInsertWorkerPayload, workerPrivateKey secp256k1.PrivKey) types.MsgInsertWorkerPayload {
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

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadFailsWithNilInference() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()

	workerMsg, _ := s.setUpMsgInsertWorkerPayload(workerPrivateKey)

	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)

	workerMsg.WorkerDataBundle.InferenceForecastsBundle.Inference = nil

	_, err := msgServer.InsertWorkerPayload(ctx, &workerMsg)
	require.Error(err)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadFailsWithoutWorkerDataBundle() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()

	workerMsg, _ := s.setUpMsgInsertWorkerPayload(workerPrivateKey)

	// BEGIN MODIFICATION
	// workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)
	// END MODIFICATION

	_, err := msgServer.InsertWorkerPayload(ctx, &workerMsg)
	require.Error(err)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadFailsWithMismatchedTopicId() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()

	workerMsg, _ := s.setUpMsgInsertWorkerPayload(workerPrivateKey)

	// BEGIN MODIFICATION
	workerMsg.WorkerDataBundle.InferenceForecastsBundle.Inference.TopicId = 1
	// END MODIFICATION

	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)

	_, err := msgServer.InsertWorkerPayload(ctx, &workerMsg)
	require.Error(err, types.ErrNoValidBundles)
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

	_, err = msgServer.InsertWorkerPayload(ctx, &workerMsg)
	require.Error(err, types.ErrNoValidBundles)
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

	ctx = ctx.WithBlockHeight(blockHeight)
	_, err := msgServer.InsertWorkerPayload(ctx, &workerMsg)
	require.NoError(err)

	forecastsCount1 := s.getCountForecastsAtBlock(originalTopicId, blockHeight)

	require.Equal(forecastsCount0, 0)
	require.Equal(forecastsCount1, 0)
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

	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)

	forecastsCount1 := s.getCountForecastsAtBlock(topicId, blockHeight)

	require.Equal(forecastsCount0, 0)
	require.Equal(forecastsCount1, 0)
}

func (s *MsgServerTestSuite) TestInsertingHugeBulkWorkerPayloadFails() {
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

	registrationInitialStake := cosmosMath.NewInt(100)

	// Create topic 0 and register reputer in it
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)
	keeper.AddWorkerNonce(ctx, 0, &nonce)
	keeper.InsertWorker(ctx, topicId, InfererAddr, workerInfo)
	keeper.InsertWorker(ctx, topicId, ForecasterAddr, workerInfo)
	s.emissionsKeeper.SetTopic(ctx, topicId, types.Topic{Id: topicId})

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
	src, err := workerMsg.WorkerDataBundle.InferenceForecastsBundle.XXX_Marshal(src, true)
	require.NoError(err, "Marshall reputer value bundle should not return an error")

	sig, err := workerPrivateKey.Sign(src)
	require.NoError(err, "Sign should not return an error")
	workerMsg.WorkerDataBundle.InferencesForecastsBundleSignature = sig
	workerMsg.WorkerDataBundle.Pubkey = hex.EncodeToString(workerPublicKeyBytes)
	_, err = msgServer.InsertWorkerPayload(ctx, workerMsg)
	require.Error(err, types.ErrQueryTooLarge)
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

	registrationInitialStake := cosmosMath.NewInt(100)

	// Create topic 0 and register reputer in it
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)
	keeper.AddWorkerNonce(ctx, 0, &nonce)
	keeper.InsertWorker(ctx, topicId, InfererAddr, workerInfo)
	keeper.InsertWorker(ctx, topicId, ForecasterAddr, workerInfo)
	s.emissionsKeeper.SetTopic(ctx, topicId, types.Topic{Id: topicId})

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

	_, err := msgServer.InsertWorkerPayload(ctx, workerMsg)
	require.Error(err, types.ErrNoValidBundles)
}
