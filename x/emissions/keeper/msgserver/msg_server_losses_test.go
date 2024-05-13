package msgserver_test

import (
	"encoding/hex"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

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

	reputerValueBundle := &types.ValueBundle{
		TopicId:       topicId,
		Reputer:       reputerAddr.String(),
		CombinedValue: alloraMath.NewDecFromInt64(100),
		InfererValues: []*types.WorkerAttributedValue{
			{
				Worker: workerAddr.String(),
				Value:  alloraMath.NewDecFromInt64(100),
			},
		},
		ForecasterValues: []*types.WorkerAttributedValue{
			{
				Worker: workerAddr.String(),
				Value:  alloraMath.NewDecFromInt64(100),
			},
		},
		NaiveValue:          alloraMath.NewDecFromInt64(100),
		OneOutInfererValues: []*types.WithheldWorkerAttributedValue{},
		OneOutForecasterValues: []*types.WithheldWorkerAttributedValue{
			{
				Worker: workerAddr.String(),
				Value:  alloraMath.NewDecFromInt64(100),
			},
		},
		OneInForecasterValues: []*types.WorkerAttributedValue{
			{
				Worker: workerAddr.String(),
				Value:  alloraMath.NewDecFromInt64(100),
			},
		},
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: reputerNonce,
			WorkerNonce:  workerNonce,
		},
	}

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

func (s *KeeperTestSuite) TestMsgInsertBulkReputerPayloadInvalid() {
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

	reputerValueBundle := &types.ValueBundle{
		TopicId:       topicId,
		Reputer:       reputerAddr.String(),
		CombinedValue: alloraMath.NewDecFromInt64(100),
		InfererValues: []*types.WorkerAttributedValue{
			{
				Worker: workerAddr.String(),
				Value:  alloraMath.NewDecFromInt64(100),
			},
		},
		ForecasterValues: []*types.WorkerAttributedValue{
			{
				Worker: workerAddr.String(),
				Value:  alloraMath.NewDecFromInt64(100),
			},
		},
		NaiveValue:             alloraMath.NewDecFromInt64(100),
		OneOutInfererValues:    []*types.WithheldWorkerAttributedValue{},
		OneOutForecasterValues: []*types.WithheldWorkerAttributedValue{
			{
				Worker: workerAddr.String(),
				Value:  alloraMath.NewDecFromInt64(100),
			},
		},
		OneInForecasterValues: []*types.WorkerAttributedValue{
			{
				Worker: workerAddr.String(),
				Value:  alloraMath.NewDecFromInt64(100),
			},
		},
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: reputerNonce,
			WorkerNonce:  workerNonce,
		},
	}

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

	// Send to the wrong topic should error
	lossesMsg.TopicId = topicId + 999
	_, err = msgServer.InsertBulkReputerPayload(ctx, lossesMsg)
	require.ErrorIs(err, types.ErrTopicDoesNotExist)

	// Fix topic
	lossesMsg.TopicId = topicId

	// Send bundle with one-out inference with just 1 inferer in topic-block
	lossesMsg.ReputerValueBundles[0].ValueBundle.OneOutInfererValues = []*types.WithheldWorkerAttributedValue{
		{
			Worker: workerAddr.String(),
			Value:  alloraMath.NewDecFromInt64(100),
		},
	}
	_, err = msgServer.InsertBulkReputerPayload(ctx, lossesMsg)
	require.ErrorIs(err, sdkerrors.ErrInvalidRequest)
}
