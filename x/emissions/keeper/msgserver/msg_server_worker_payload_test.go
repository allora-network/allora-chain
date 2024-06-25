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

func (s *MsgServerTestSuite) setUpMsgInsertBulkWorkerPayload(
	workerPrivateKey secp256k1.PrivKey,

) (types.MsgInsertBulkWorkerPayload, uint64) {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	nonce := types.Nonce{BlockHeight: 1}
	topicId := uint64(0)

	// Define sample OffchainNode information for a worker
	workerInfo := types.OffchainNode{
		LibP2PKey:    "worker-libp2p-key-sample",
		MultiAddress: "worker-multi-address-sample",
		Owner:        "worker-owner-sample",
		NodeAddress:  "worker-node-address-sample",
		NodeId:       "worker-node-id-sample",
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
	keeper.AddWorkerNonce(ctx, 0, &nonce)
	keeper.InsertWorker(ctx, topicId, InfererAddr, workerInfo)
	keeper.InsertWorker(ctx, topicId, ForecasterAddr, workerInfo)
	s.emissionsKeeper.SetTopic(ctx, topicId, types.Topic{Id: topicId})

	// Create a MsgInsertBulkWorkerPayload message
	workerMsg := types.MsgInsertBulkWorkerPayload{
		Sender:  workerAddr,
		Nonce:   &nonce,
		TopicId: topicId,
		WorkerDataBundles: []*types.WorkerDataBundle{
			{
				Worker: InfererAddr,
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
		},
	}

	return workerMsg, topicId
}

func (s *MsgServerTestSuite) signMsgInsertBulkWorkerPayload(workerMsg types.MsgInsertBulkWorkerPayload, workerPrivateKey secp256k1.PrivKey) types.MsgInsertBulkWorkerPayload {
	require := s.Require()

	workerPublicKeyBytes := workerPrivateKey.PubKey().Bytes()

	src := make([]byte, 0)
	src, err := workerMsg.WorkerDataBundles[0].InferenceForecastsBundle.XXX_Marshal(src, true)
	require.NoError(err, "Marshall reputer value bundle should not return an error")

	sig, err := workerPrivateKey.Sign(src)
	require.NoError(err, "Sign should not return an error")
	workerMsg.WorkerDataBundles[0].InferencesForecastsBundleSignature = sig
	workerMsg.WorkerDataBundles[0].Pubkey = hex.EncodeToString(workerPublicKeyBytes)

	return workerMsg
}

func (s *MsgServerTestSuite) TestMsgInsertBulkWorkerPayload() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()

	workerMsg, topicId := s.setUpMsgInsertBulkWorkerPayload(workerPrivateKey)

	workerMsg = s.signMsgInsertBulkWorkerPayload(workerMsg, workerPrivateKey)

	blockHeight := workerMsg.WorkerDataBundles[0].InferenceForecastsBundle.Forecast.BlockHeight

	forecastsCount0 := s.getCountForecastsAtBlock(topicId, blockHeight)

	_, err := msgServer.InsertBulkWorkerPayload(ctx, &workerMsg)
	require.NoError(err, "InsertBulkWorkerPayload should not return an error")

	forecastsCount1 := s.getCountForecastsAtBlock(topicId, blockHeight)

	require.Equal(forecastsCount0, 0)
	require.Equal(forecastsCount1, 1)
}

func (s *MsgServerTestSuite) TestMsgInsertBulkWorkerPayloadFailsWithoutWorkerDataBundle() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()

	workerMsg, _ := s.setUpMsgInsertBulkWorkerPayload(workerPrivateKey)

	// BEGIN MODIFICATION
	workerMsg.WorkerDataBundles = make([]*types.WorkerDataBundle, 0)
	// workerMsg = s.signMsgInsertBulkWorkerPayload(workerMsg, workerPrivateKey)
	// END MODIFICATION

	_, err := msgServer.InsertBulkWorkerPayload(ctx, &workerMsg)
	require.Error(err)
}

func (s *MsgServerTestSuite) TestMsgInsertBulkWorkerPayloadFailsWithMismatchedTopicId() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()

	workerMsg, _ := s.setUpMsgInsertBulkWorkerPayload(workerPrivateKey)

	// BEGIN MODIFICATION
	workerMsg.WorkerDataBundles[0].InferenceForecastsBundle.Inference.TopicId = 1
	// END MODIFICATION

	workerMsg = s.signMsgInsertBulkWorkerPayload(workerMsg, workerPrivateKey)

	_, err := msgServer.InsertBulkWorkerPayload(ctx, &workerMsg)
	require.Error(err, types.ErrNoValidBundles)
}

func (s *MsgServerTestSuite) TestMsgInsertBulkWorkerPayloadFailsWithUnregisteredInferer() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()

	workerMsg, topicId := s.setUpMsgInsertBulkWorkerPayload(workerPrivateKey)

	// BEGIN MODIFICATION
	inferer := workerMsg.WorkerDataBundles[0].InferenceForecastsBundle.Inference.Inferer

	unregisterMsg := &types.MsgRemoveRegistration{
		Sender:    inferer,
		TopicId:   topicId,
		IsReputer: false,
	}

	_, err := msgServer.RemoveRegistration(ctx, unregisterMsg)
	require.NoError(err)

	// END MODIFICATION

	workerMsg = s.signMsgInsertBulkWorkerPayload(workerMsg, workerPrivateKey)

	_, err = msgServer.InsertBulkWorkerPayload(ctx, &workerMsg)
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

func (s *MsgServerTestSuite) TestMsgInsertBulkWorkerPayloadFailsWithMismatchedForecastTopicId() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()

	workerMsg, _ := s.setUpMsgInsertBulkWorkerPayload(workerPrivateKey)

	// BEGIN MODIFICATION
	originalTopicId := workerMsg.WorkerDataBundles[0].InferenceForecastsBundle.Forecast.TopicId
	workerMsg.WorkerDataBundles[0].InferenceForecastsBundle.Forecast.TopicId = 123
	// END MODIFICATION

	workerMsg = s.signMsgInsertBulkWorkerPayload(workerMsg, workerPrivateKey)

	blockHeight := workerMsg.WorkerDataBundles[0].InferenceForecastsBundle.Forecast.BlockHeight

	forecastsCount0 := s.getCountForecastsAtBlock(originalTopicId, blockHeight)

	_, err := msgServer.InsertBulkWorkerPayload(ctx, &workerMsg)
	require.NoError(err)

	forecastsCount1 := s.getCountForecastsAtBlock(originalTopicId, blockHeight)

	require.Equal(forecastsCount0, 0)
	require.Equal(forecastsCount1, 0)
}

func (s *MsgServerTestSuite) TestMsgInsertBulkWorkerPayloadFailsWithUnregisteredForecaster() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()

	workerMsg, topicId := s.setUpMsgInsertBulkWorkerPayload(workerPrivateKey)

	// BEGIN MODIFICATION
	forecaster := workerMsg.WorkerDataBundles[0].InferenceForecastsBundle.Forecast.Forecaster

	unregisterMsg := &types.MsgRemoveRegistration{
		Sender:    forecaster,
		TopicId:   topicId,
		IsReputer: false,
	}

	_, err := msgServer.RemoveRegistration(ctx, unregisterMsg)
	require.NoError(err)

	// END MODIFICATION

	blockHeight := workerMsg.WorkerDataBundles[0].InferenceForecastsBundle.Forecast.BlockHeight

	forecastsCount0 := s.getCountForecastsAtBlock(topicId, blockHeight)

	workerMsg = s.signMsgInsertBulkWorkerPayload(workerMsg, workerPrivateKey)

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
		LibP2PKey:    "worker-libp2p-key-sample",
		MultiAddress: "worker-multi-address-sample",
		Owner:        "worker-owner-sample",
		NodeAddress:  "worker-node-address-sample",
		NodeId:       "worker-node-id-sample",
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

	// Create a MsgInsertBulkWorkerPayload message
	workerMsg := &types.MsgInsertBulkWorkerPayload{
		Sender:  workerAddr,
		Nonce:   &nonce,
		TopicId: topicId,
		WorkerDataBundles: []*types.WorkerDataBundle{
			{
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
		},
	}

	src := make([]byte, 0)
	src, err := workerMsg.WorkerDataBundles[0].InferenceForecastsBundle.XXX_Marshal(src, true)
	require.NoError(err, "Marshall reputer value bundle should not return an error")

	sig, err := workerPrivateKey.Sign(src)
	require.NoError(err, "Sign should not return an error")
	workerMsg.WorkerDataBundles[0].InferencesForecastsBundleSignature = sig
	workerMsg.WorkerDataBundles[0].Pubkey = hex.EncodeToString(workerPublicKeyBytes)
	_, err = msgServer.InsertBulkWorkerPayload(ctx, workerMsg)
	require.Error(err, types.ErrQueryTooLarge)
}

func (s *MsgServerTestSuite) TestMsgInsertBulkWorkerPayloadVerifyFailed() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	keeper := s.emissionsKeeper
	topicId := uint64(0)
	nonce := types.Nonce{BlockHeight: 1}

	// Define sample OffchainNode information for a worker
	workerInfo := types.OffchainNode{
		LibP2PKey:    "worker-libp2p-key-sample",
		MultiAddress: "worker-multi-address-sample",
		Owner:        "worker-owner-sample",
		NodeAddress:  "worker-node-address-sample",
		NodeId:       "worker-node-id-sample",
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

	// Create a MsgInsertBulkWorkerPayload message
	workerMsg := &types.MsgInsertBulkWorkerPayload{
		Sender:  workerAddr,
		Nonce:   &nonce,
		TopicId: topicId,
		WorkerDataBundles: []*types.WorkerDataBundle{
			{
				Worker: InfererAddr,
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
				InferencesForecastsBundleSignature: []byte("Signature"),
				Pubkey:                             "Failed Pubkey",
			},
		},
	}

	_, err := msgServer.InsertBulkWorkerPayload(ctx, workerMsg)
	require.ErrorIs(err, types.ErrNoValidBundles)
}

func (s *MsgServerTestSuite) TestMsgInsertBulkWorkerAlreadyFullfilledNonce() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	keeper := s.emissionsKeeper
	topicId := uint64(0)
	nonce := types.Nonce{BlockHeight: 1}

	// Define sample OffchainNode information for a worker
	workerInfo := types.OffchainNode{
		LibP2PKey:    "worker-libp2p-key-sample",
		MultiAddress: "worker-multi-address-sample",
		Owner:        "worker-owner-sample",
		NodeAddress:  "worker-node-address-sample",
		NodeId:       "worker-node-id-sample",
	}
	// Mock setup for addresses

	reputerPrivateKey := secp256k1.GenPrivKey()
	reputerAddr := sdk.AccAddress(reputerPrivateKey.PubKey().Address()).String()

	workerPrivateKey := secp256k1.GenPrivKey()
	workerPublicKeyBytes := workerPrivateKey.PubKey().Bytes()
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

	// Create a MsgInsertBulkWorkerPayload message
	workerMsg := &types.MsgInsertBulkWorkerPayload{
		Sender:  workerAddr,
		Nonce:   &nonce,
		TopicId: topicId,
		WorkerDataBundles: []*types.WorkerDataBundle{
			{
				Worker: InfererAddr,
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
		},
	}

	src := make([]byte, 0)
	src, err := workerMsg.WorkerDataBundles[0].InferenceForecastsBundle.XXX_Marshal(src, true)
	require.NoError(err, "Marshall reputer value bundle should not return an error")

	sig, err := workerPrivateKey.Sign(src)
	require.NoError(err, "Sign should not return an error")
	workerMsg.WorkerDataBundles[0].InferencesForecastsBundleSignature = sig
	workerMsg.WorkerDataBundles[0].Pubkey = hex.EncodeToString(workerPublicKeyBytes)

	_, err = msgServer.InsertBulkWorkerPayload(ctx, workerMsg)
	require.NoError(err)
	_, err = msgServer.InsertBulkWorkerPayload(ctx, workerMsg)
	require.ErrorIs(err, types.ErrNonceAlreadyFulfilled)
}
