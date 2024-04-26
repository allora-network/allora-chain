package msgserver_test

import (
	"encoding/hex"

	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestMsgInsertBulkWorkerPayload() {
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
	reputerAddr := sdk.AccAddress(reputerPrivateKey.PubKey().Address())

	workerPrivateKey := secp256k1.GenPrivKey()
	workerPublicKeyBytes := workerPrivateKey.PubKey().Bytes()
	workerAddr := sdk.AccAddress(workerPrivateKey.PubKey().Address())

	InfererPrivateKey := secp256k1.GenPrivKey()
	InfererAddr := sdk.AccAddress(InfererPrivateKey.PubKey().Address())

	Inferer2PrivateKey := secp256k1.GenPrivKey()
	Inferer2Addr := sdk.AccAddress(Inferer2PrivateKey.PubKey().Address())

	ForecasterPrivateKey := secp256k1.GenPrivKey()
	ForecasterAddr := sdk.AccAddress(ForecasterPrivateKey.PubKey().Address())

	registrationInitialStake := cosmosMath.NewUint(100)

	// Create topic 0 and register reputer in it
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)
	keeper.AddWorkerNonce(ctx, 0, &nonce)
	keeper.InsertWorker(ctx, topicId, InfererAddr, workerInfo)
	keeper.InsertWorker(ctx, topicId, ForecasterAddr, workerInfo)
	s.emissionsKeeper.SetTopic(ctx, topicId, types.Topic{Id: topicId})

	// Create a MsgInsertBulkReputerPayload message
	workerMsg := &types.MsgInsertBulkWorkerPayload{
		Sender:  workerAddr.String(),
		Nonce:   &nonce,
		TopicId: topicId,
		WorkerDataBundles: []*types.WorkerDataBundle{
			{
				Worker: InfererAddr.String(),
				InferenceForecastsBundle: &types.InferenceForecastBundle{
					Inference: &types.Inference{
						TopicId:     topicId,
						BlockHeight: nonce.BlockHeight,
						Inferer:     InfererAddr.String(),
						Value:       alloraMath.NewDecFromInt64(100),
					},
					Forecast: &types.Forecast{
						TopicId:     0,
						BlockHeight: nonce.BlockHeight,
						Forecaster:  ForecasterAddr.String(),
						ForecastElements: []*types.ForecastElement{
							{
								Inferer: InfererAddr.String(),
								Value:   alloraMath.NewDecFromInt64(100),
							},
							{
								Inferer: Inferer2Addr.String(),
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
	require.NoError(err, "InsertBulkWorkerPayload should not return an error")
}

func (s *KeeperTestSuite) TestMsgInsertBulkWorkerPayloadVerifyFailed() {
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
	reputerAddr := sdk.AccAddress(reputerPrivateKey.PubKey().Address())

	workerPrivateKey := secp256k1.GenPrivKey()
	workerAddr := sdk.AccAddress(workerPrivateKey.PubKey().Address())

	InfererPrivateKey := secp256k1.GenPrivKey()
	InfererAddr := sdk.AccAddress(InfererPrivateKey.PubKey().Address())

	Inferer2PrivateKey := secp256k1.GenPrivKey()
	Inferer2Addr := sdk.AccAddress(Inferer2PrivateKey.PubKey().Address())

	ForecasterPrivateKey := secp256k1.GenPrivKey()
	ForecasterAddr := sdk.AccAddress(ForecasterPrivateKey.PubKey().Address())

	registrationInitialStake := cosmosMath.NewUint(100)

	// Create topic 0 and register reputer in it
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)
	keeper.AddWorkerNonce(ctx, 0, &nonce)
	keeper.InsertWorker(ctx, topicId, InfererAddr, workerInfo)
	keeper.InsertWorker(ctx, topicId, ForecasterAddr, workerInfo)
	s.emissionsKeeper.SetTopic(ctx, topicId, types.Topic{Id: topicId})

	// Create a MsgInsertBulkReputerPayload message
	workerMsg := &types.MsgInsertBulkWorkerPayload{
		Sender:  workerAddr.String(),
		Nonce:   &nonce,
		TopicId: topicId,
		WorkerDataBundles: []*types.WorkerDataBundle{
			{
				Worker: InfererAddr.String(),
				InferenceForecastsBundle: &types.InferenceForecastBundle{
					Inference: &types.Inference{
						TopicId:     topicId,
						BlockHeight: nonce.BlockHeight,
						Inferer:     InfererAddr.String(),
						Value:       alloraMath.NewDecFromInt64(100),
					},
					Forecast: &types.Forecast{
						TopicId:     0,
						BlockHeight: nonce.BlockHeight,
						Forecaster:  ForecasterAddr.String(),
						ForecastElements: []*types.ForecastElement{
							{
								Inferer: InfererAddr.String(),
								Value:   alloraMath.NewDecFromInt64(100),
							},
							{
								Inferer: Inferer2Addr.String(),
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
	require.ErrorIs(err, types.ErrSignatureVerificationFailed)
}

func (s *KeeperTestSuite) TestMsgInsertBulkWorkerAlreadyFullfilledNonce() {
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
	reputerAddr := sdk.AccAddress(reputerPrivateKey.PubKey().Address())

	workerPrivateKey := secp256k1.GenPrivKey()
	workerPublicKeyBytes := workerPrivateKey.PubKey().Bytes()
	workerAddr := sdk.AccAddress(workerPrivateKey.PubKey().Address())

	InfererPrivateKey := secp256k1.GenPrivKey()
	InfererAddr := sdk.AccAddress(InfererPrivateKey.PubKey().Address())

	Inferer2PrivateKey := secp256k1.GenPrivKey()
	Inferer2Addr := sdk.AccAddress(Inferer2PrivateKey.PubKey().Address())

	ForecasterPrivateKey := secp256k1.GenPrivKey()
	ForecasterAddr := sdk.AccAddress(ForecasterPrivateKey.PubKey().Address())

	registrationInitialStake := cosmosMath.NewUint(100)

	// Create topic 0 and register reputer in it
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)
	keeper.AddWorkerNonce(ctx, 0, &nonce)
	keeper.InsertWorker(ctx, topicId, InfererAddr, workerInfo)
	keeper.InsertWorker(ctx, topicId, ForecasterAddr, workerInfo)
	s.emissionsKeeper.SetTopic(ctx, topicId, types.Topic{Id: topicId})

	// Create a MsgInsertBulkReputerPayload message
	workerMsg := &types.MsgInsertBulkWorkerPayload{
		Sender:  workerAddr.String(),
		Nonce:   &nonce,
		TopicId: topicId,
		WorkerDataBundles: []*types.WorkerDataBundle{
			{
				Worker: InfererAddr.String(),
				InferenceForecastsBundle: &types.InferenceForecastBundle{
					Inference: &types.Inference{
						TopicId:     topicId,
						BlockHeight: nonce.BlockHeight,
						Inferer:     InfererAddr.String(),
						Value:       alloraMath.NewDecFromInt64(100),
					},
					Forecast: &types.Forecast{
						TopicId:     0,
						BlockHeight: nonce.BlockHeight,
						Forecaster:  ForecasterAddr.String(),
						ForecastElements: []*types.ForecastElement{
							{
								Inferer: InfererAddr.String(),
								Value:   alloraMath.NewDecFromInt64(100),
							},
							{
								Inferer: Inferer2Addr.String(),
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
	_, err = msgServer.InsertBulkWorkerPayload(ctx, workerMsg)
	require.ErrorIs(err, types.ErrNonceAlreadyFulfilled)
}
