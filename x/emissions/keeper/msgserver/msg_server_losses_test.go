package msgserver_test

import (
	"encoding/hex"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// createReputerValueBundle constructs and returns a pointer to a types.ValueBundle for given inputs.
func createReputerValueBundle(topicId uint64, reputerAddr sdk.AccAddress, workerAddr sdk.AccAddress, value int64) *types.ValueBundle {
	return &types.ValueBundle{
		TopicId:       topicId,
		Reputer:       reputerAddr.String(),
		CombinedValue: alloraMath.NewDecFromInt64(value),
		InfererValues: []*types.WorkerAttributedValue{
			{
				Worker: workerAddr.String(),
				Value:  alloraMath.NewDecFromInt64(value),
			},
		},
		ForecasterValues: []*types.WorkerAttributedValue{
			{
				Worker: workerAddr.String(),
				Value:  alloraMath.NewDecFromInt64(value),
			},
		},
		NaiveValue: alloraMath.NewDecFromInt64(value),
		OneOutInfererValues: []*types.WithheldWorkerAttributedValue{
			{
				Worker: workerAddr.String(),
				Value:  alloraMath.NewDecFromInt64(value),
			},
		},
		OneOutForecasterValues: []*types.WithheldWorkerAttributedValue{
			{
				Worker: workerAddr.String(),
				Value:  alloraMath.NewDecFromInt64(value),
			},
		},
		OneInForecasterValues: []*types.WorkerAttributedValue{
			{
				Worker: workerAddr.String(),
				Value:  alloraMath.NewDecFromInt64(value),
			},
		},
	}
}

func (s *KeeperTestSuite) TestMsgInsertBulkReputerPayload() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	keeper := s.emissionsKeeper

	reputerPrivateKey := secp256k1.GenPrivKey()
	reputerPublicKeyBytes := reputerPrivateKey.PubKey().Bytes()
	reputerAddr := sdk.AccAddress(reputerPrivateKey.PubKey().Address())

	workerPrivateKey := secp256k1.GenPrivKey()
	workerAddr := sdk.AccAddress(workerPrivateKey.PubKey().Address())

	minStake, err := keeper.GetParamsRequiredMinimumStake(ctx)
	require.NoError(err)

	minStakeScaled := minStake.Mul(inference_synthesis.CosmosUintOneE18())

	topicId := s.commonStakingSetup(ctx, reputerAddr, workerAddr, minStakeScaled)

	addStakeMsg := &types.MsgAddStake{
		Sender:  reputerAddr.String(),
		TopicId: topicId,
		Amount:  minStakeScaled,
	}

	_, err = msgServer.AddStake(ctx, addStakeMsg)
	s.Require().NoError(err)

	reputerNonce := &types.Nonce{
		BlockHeight: 2,
	}
	workerNonce := &types.Nonce{
		BlockHeight: 1,
	}

	keeper.AddWorkerNonce(ctx, topicId, workerNonce)
	keeper.FulfillWorkerNonce(ctx, topicId, workerNonce)
	keeper.AddReputerNonce(ctx, topicId, reputerNonce, workerNonce)

	// add in inference and forecast data
	block := types.BlockHeight(1)
	expectedInferences := types.Inferences{
		Inferences: []*types.Inference{
			{
				Value:   alloraMath.NewDecFromInt64(1), // Assuming NewDecFromInt64 exists and is appropriate
				Inferer: workerAddr.String(),
			},
		},
	}

	nonce := types.Nonce{BlockHeight: block} // Assuming block type cast to int64 if needed
	err = keeper.InsertInferences(ctx, topicId, nonce, expectedInferences)
	require.NoError(err, "InsertInferences should not return an error")

	expectedForecasts := types.Forecasts{
		Forecasts: []*types.Forecast{
			{
				TopicId:    topicId,
				Forecaster: workerAddr.String(),
			},
		},
	}

	nonce = types.Nonce{BlockHeight: int64(block)}
	err = keeper.InsertForecasts(ctx, topicId, nonce, expectedForecasts)
	s.Require().NoError(err)

	reputerValueBundle := createReputerValueBundle(topicId, reputerAddr, workerAddr, 100)

	src := make([]byte, 0)
	src, err = reputerValueBundle.XXX_Marshal(src, true)
	require.NoError(err, "Marshall reputer value bundle should not return an error")

	valueBundleSignature, err := reputerPrivateKey.Sign(src)
	require.NoError(err, "Sign should not return an error")

	// Create a MsgInsertBulkReputerPayload message
	lossesMsg := &types.MsgInsertBulkReputerPayload{
		Sender:  reputerAddr.String(),
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: reputerNonce,
			WorkerNonce:  workerNonce,
		},
		ReputerValueBundles: []*types.ReputerValueBundle{
			{
				ValueBundle: reputerValueBundle,
				Signature:   valueBundleSignature,
				Pubkey:      hex.EncodeToString(reputerPublicKeyBytes),
			},
		},
	}

	_, err = msgServer.InsertBulkReputerPayload(ctx, lossesMsg)
	require.NoError(err, "InsertBulkReputerPayload should not return an error")
}

func (s *KeeperTestSuite) TestMsgInsertBulkReputerPayloadContinueStatements() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	keeper := s.emissionsKeeper

	reputerPrivateKey := secp256k1.GenPrivKey()
	reputerPublicKeyBytes := reputerPrivateKey.PubKey().Bytes()
	reputerAddr := sdk.AccAddress(reputerPrivateKey.PubKey().Address())

	workerPrivateKey := secp256k1.GenPrivKey()
	workerAddr := sdk.AccAddress(workerPrivateKey.PubKey().Address())

	minStake, err := keeper.GetParamsRequiredMinimumStake(ctx)
	require.NoError(err)

	minStakeScaled := minStake.Mul(inference_synthesis.CosmosUintOneE18())

	topicId := s.commonStakingSetup(ctx, reputerAddr, workerAddr, minStakeScaled)

	addStakeMsg := &types.MsgAddStake{
		Sender:  reputerAddr.String(),
		TopicId: topicId,
		Amount:  minStakeScaled,
	}

	_, err = msgServer.AddStake(ctx, addStakeMsg)
	s.Require().NoError(err)

	reputerNonce := &types.Nonce{
		BlockHeight: 2,
	}
	workerNonce := &types.Nonce{
		BlockHeight: 1,
	}

	keeper.AddWorkerNonce(ctx, topicId, workerNonce)
	keeper.FulfillWorkerNonce(ctx, topicId, workerNonce)
	keeper.AddReputerNonce(ctx, topicId, reputerNonce, workerNonce)

	// add in inference and forecast data
	block := types.BlockHeight(1)
	expectedInferences := types.Inferences{
		Inferences: []*types.Inference{
			{
				Value:   alloraMath.NewDecFromInt64(1), // Assuming NewDecFromInt64 exists and is appropriate
				Inferer: workerAddr.String(),
			},
		},
	}

	nonce := types.Nonce{BlockHeight: block} // Assuming block type cast to int64 if needed
	err = keeper.InsertInferences(ctx, topicId, nonce, expectedInferences)
	require.NoError(err, "InsertInferences should not return an error")

	expectedForecasts := types.Forecasts{
		Forecasts: []*types.Forecast{
			{
				TopicId:    topicId,
				Forecaster: workerAddr.String(),
			},
		},
	}

	nonce = types.Nonce{BlockHeight: int64(block)}
	err = keeper.InsertForecasts(ctx, topicId, nonce, expectedForecasts)
	s.Require().NoError(err)

	reputerValueBundle1 := createReputerValueBundle(topicId, reputerAddr, workerAddr, 100)
	reputerValueBundle2 := createReputerValueBundle(topicId+1, reputerAddr, workerAddr, 100)

	src1 := make([]byte, 0)
	src1, err = reputerValueBundle1.XXX_Marshal(src1, true)
	require.NoError(err, "Marshall reputer value bundle should not return an error")

	valueBundleSignature1, err := reputerPrivateKey.Sign(src1)
	require.NoError(err, "Sign should not return an error")

	src2 := make([]byte, 0)
	src2, err = reputerValueBundle2.XXX_Marshal(src2, true)
	require.NoError(err, "Marshall reputer value bundle should not return an error")

	valueBundleSignature2, err := reputerPrivateKey.Sign(src2)
	require.NoError(err, "Sign should not return an error")

	// Create a MsgInsertBulkReputerPayload message
	lossesMsg := &types.MsgInsertBulkReputerPayload{
		Sender:  reputerAddr.String(),
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: reputerNonce,
			WorkerNonce:  workerNonce,
		},
		ReputerValueBundles: []*types.ReputerValueBundle{
			{
				ValueBundle: reputerValueBundle1,
				Signature:   valueBundleSignature1,
				Pubkey:      hex.EncodeToString(reputerPublicKeyBytes),
			},
			{
				ValueBundle: reputerValueBundle2,
				Signature:   valueBundleSignature2,
				Pubkey:      hex.EncodeToString(reputerPublicKeyBytes),
			},
		},
	}

	_, err = msgServer.InsertBulkReputerPayload(ctx, lossesMsg)
	require.NoError(err, "InsertBulkReputerPayload should not return an error")
}
