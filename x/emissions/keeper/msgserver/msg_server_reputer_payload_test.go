package msgserver_test

import (
	"encoding/hex"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *MsgServerTestSuite) setUpMsgReputerPayload(
	reputerAddr sdk.AccAddress,
	workerAddr sdk.AccAddress,
	block types.BlockHeight,
) (
	reputerValueBundle types.ValueBundle,
	expectedInferences types.Inferences,
	expectedForecasts types.Forecasts,
	topicId uint64,
	reputerNonce types.Nonce,
	workerNonce types.Nonce,
) {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	keeper := s.emissionsKeeper

	params, err := keeper.GetParams(ctx)
	require.NoError(err)

	minStakeScaled := params.RequiredMinimumStake.Mul(inference_synthesis.CosmosIntOneE18())

	topicId = s.commonStakingSetup(ctx, reputerAddr.String(), workerAddr.String(), minStakeScaled)
	s.MintTokensToAddress(reputerAddr, params.RequiredMinimumStake)

	addStakeMsg := &types.MsgAddStake{
		Sender:  reputerAddr.String(),
		TopicId: topicId,
		Amount:  minStakeScaled,
	}

	_, err = msgServer.AddStake(ctx, addStakeMsg)
	s.Require().NoError(err)

	reputerNonce = types.Nonce{
		BlockHeight: block,
	}
	workerNonce = types.Nonce{
		BlockHeight: block,
	}

	keeper.AddWorkerNonce(ctx, topicId, &workerNonce)
	keeper.FulfillWorkerNonce(ctx, topicId, &workerNonce)
	keeper.AddReputerNonce(ctx, topicId, &reputerNonce)

	// add in inference and forecast data
	expectedInferences = types.Inferences{
		Inferences: []*types.Inference{
			{
				Value:   alloraMath.NewDecFromInt64(1), // Assuming NewDecFromInt64 exists and is appropriate
				Inferer: workerAddr.String(),
			},
		},
	}

	expectedForecasts = types.Forecasts{
		Forecasts: []*types.Forecast{
			{
				TopicId:    topicId,
				Forecaster: workerAddr.String(),
			},
		},
	}

	reputerValueBundle = types.ValueBundle{
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
			ReputerNonce: &reputerNonce,
		},
	}

	return reputerValueBundle, expectedInferences, expectedForecasts, topicId, reputerNonce, workerNonce
}

func (s *MsgServerTestSuite) signValueBundle(reputerValueBundle *types.ValueBundle, privateKey secp256k1.PrivKey) []byte {
	require := s.Require()
	src := make([]byte, 0)
	src, err := reputerValueBundle.XXX_Marshal(src, true)
	require.NoError(err, "Marshall reputer value bundle should not return an error")

	valueBundleSignature, err := privateKey.Sign(src)
	require.NoError(err, "Sign should not return an error")

	return valueBundleSignature
}

func (s *MsgServerTestSuite) constructAndInsertReputerPayload(
	reputerAddr sdk.AccAddress,
	reputerPrivateKey secp256k1.PrivKey,
	reputerPublicKeyBytes []byte,
	reputerValueBundle *types.ValueBundle,
	topicId uint64,
	reputerNonce *types.Nonce,
) error {
	ctx, msgServer := s.ctx, s.msgServer
	valueBundleSignature := s.signValueBundle(reputerValueBundle, reputerPrivateKey)

	// Create a MsgInsertReputerPayload message
	lossesMsg := &types.MsgInsertReputerPayload{
		Sender: reputerAddr.String(),
		ReputerValueBundle: &types.ReputerValueBundle{
			ValueBundle: reputerValueBundle,
			Signature:   valueBundleSignature,
			Pubkey:      hex.EncodeToString(reputerPublicKeyBytes),
		},
	}

	_, err := msgServer.InsertReputerPayload(ctx, lossesMsg)
	return err
}

func (s *MsgServerTestSuite) TestMsgInsertReputerPayload() {
	ctx := s.ctx
	require := s.Require()
	keeper := s.emissionsKeeper

	block := types.BlockHeight(1)

	reputerPrivateKey := secp256k1.GenPrivKey()
	reputerPublicKeyBytes := reputerPrivateKey.PubKey().Bytes()
	reputerAddr := sdk.AccAddress(reputerPrivateKey.PubKey().Address())

	workerPrivateKey := secp256k1.GenPrivKey()
	workerAddr := sdk.AccAddress(workerPrivateKey.PubKey().Address())

	reputerValueBundle, expectedInferences, expectedForecasts, topicId, reputerNonce, _ := s.setUpMsgReputerPayload(reputerAddr, workerAddr, block)

	err := keeper.InsertForecasts(ctx, topicId, types.Nonce{BlockHeight: int64(block)}, expectedForecasts)
	require.NoError(err)

	err = keeper.InsertInferences(ctx, topicId, types.Nonce{BlockHeight: block}, expectedInferences)
	require.NoError(err)

	topic, err := s.emissionsKeeper.GetTopic(s.ctx, topicId)
	s.Require().NoError(err)

	newBlockheight := block + topic.GroundTruthLag
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(newBlockheight)

	err = s.constructAndInsertReputerPayload(reputerAddr, reputerPrivateKey, reputerPublicKeyBytes, &reputerValueBundle, topicId, &reputerNonce)
	require.NoError(err)
}
