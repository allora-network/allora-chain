package msgserver_test

import (
	"encoding/hex"

	alloraMath "github.com/allora-network/allora-chain/math"
	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const block = types.BlockHeight(1)

func (s *MsgServerTestSuite) setUpMsgReputerPayload(
	reputerAddr sdk.AccAddress,
	workerAddr sdk.AccAddress,
) (
	reputerValueBundle types.ValueBundle,
	expectedInferences types.Inferences,
	expectedForecasts types.Forecasts,
	topicId uint64,
) {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	keeper := s.emissionsKeeper

	params, err := keeper.GetParams(ctx)
	require.NoError(err)

	minStakeScaled := params.RequiredMinimumStake.Mul(inferencesynthesis.CosmosIntOneE18())

	topicId = s.commonStakingSetup(ctx, reputerAddr.String(), workerAddr.String(), minStakeScaled)
	s.MintTokensToAddress(reputerAddr, params.RequiredMinimumStake)

	addStakeMsg := &types.MsgAddStake{
		Sender:  reputerAddr.String(),
		TopicId: topicId,
		Amount:  minStakeScaled,
	}

	_, err = msgServer.AddStake(ctx, addStakeMsg)
	s.Require().NoError(err)

	workerNonce := types.Nonce{
		BlockHeight: block,
	}

	err = keeper.AddWorkerNonce(ctx, topicId, &workerNonce)
	require.NoError(err)
	_, err = keeper.FulfillWorkerNonce(ctx, topicId, &workerNonce)
	require.NoError(err)
	err = keeper.AddReputerNonce(ctx, topicId, &workerNonce)
	require.NoError(err)

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
			ReputerNonce: &workerNonce,
		},
	}

	return reputerValueBundle, expectedInferences, expectedForecasts, topicId
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

func (s *MsgServerTestSuite) TestMsgInsertReputerPayloadFailsEarlyWindow() {
	ctx := s.ctx
	require := s.Require()
	keeper := s.emissionsKeeper

	reputerPrivateKey := secp256k1.GenPrivKey()
	reputerPublicKeyBytes := reputerPrivateKey.PubKey().Bytes()
	reputerAddr := sdk.AccAddress(reputerPrivateKey.PubKey().Address())

	workerPrivateKey := secp256k1.GenPrivKey()
	workerAddr := sdk.AccAddress(workerPrivateKey.PubKey().Address())

	reputerValueBundle, expectedInferences, expectedForecasts, topicId := s.setUpMsgReputerPayload(reputerAddr, workerAddr)

	err := keeper.InsertForecasts(ctx, topicId, types.Nonce{BlockHeight: block}, expectedForecasts)
	require.NoError(err)

	err = keeper.InsertInferences(ctx, topicId, types.Nonce{BlockHeight: block}, expectedInferences)
	require.NoError(err)

	topic, err := s.emissionsKeeper.GetTopic(s.ctx, topicId)
	s.Require().NoError(err)

	// Prior to the ground truth lag, should not allow reputer payload
	newBlockheight := block + topic.GroundTruthLag - 1
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(newBlockheight)

	err = s.constructAndInsertReputerPayload(reputerAddr, reputerPrivateKey, reputerPublicKeyBytes, &reputerValueBundle)
	require.ErrorIs(err, types.ErrReputerNonceWindowNotAvailable)

	// Valid reputer nonce window, start
	newBlockheight = block + topic.GroundTruthLag
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(newBlockheight)

	err = s.constructAndInsertReputerPayload(reputerAddr, reputerPrivateKey, reputerPublicKeyBytes, &reputerValueBundle)
	require.NoError(err)

	// Valid reputer nonce window, end
	newBlockheight = block + topic.GroundTruthLag*2
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(newBlockheight)
	err = s.constructAndInsertReputerPayload(reputerAddr, reputerPrivateKey, reputerPublicKeyBytes, &reputerValueBundle)
	require.NoError(err)

	// Valid reputer nonce window, end
	newBlockheight = block + topic.GroundTruthLag*2 + 1
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(newBlockheight)
	err = s.constructAndInsertReputerPayload(reputerAddr, reputerPrivateKey, reputerPublicKeyBytes, &reputerValueBundle)
	require.ErrorIs(err, types.ErrReputerNonceWindowNotAvailable)
}

func (s *MsgServerTestSuite) TestMsgInsertReputerPayloadReputerNotMatchSignature() {
	ctx := s.ctx
	require := s.Require()
	keeper := s.emissionsKeeper

	reputerPrivateKey := secp256k1.GenPrivKey()
	reputerPublicKeyBytes := reputerPrivateKey.PubKey().Bytes()
	reputerAddr := sdk.AccAddress(reputerPrivateKey.PubKey().Address())

	workerPrivateKey := secp256k1.GenPrivKey()
	workerAddr := sdk.AccAddress(workerPrivateKey.PubKey().Address())

	reputerValueBundle, expectedInferences, expectedForecasts, topicId := s.setUpMsgReputerPayload(reputerAddr, workerAddr)

	err := keeper.InsertForecasts(ctx, topicId, types.Nonce{BlockHeight: block}, expectedForecasts)
	require.NoError(err)

	err = keeper.InsertInferences(ctx, topicId, types.Nonce{BlockHeight: block}, expectedInferences)
	require.NoError(err)

	topic, err := s.emissionsKeeper.GetTopic(s.ctx, topicId)
	s.Require().NoError(err)

	// Prior to the ground truth lag, should not allow reputer payload
	newBlockheight := block + topic.GroundTruthLag - 1
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(newBlockheight)

	reputerValueBundle.Reputer = s.addrsStr[3]
	valueBundleSignature := s.signValueBundle(&reputerValueBundle, reputerPrivateKey)

	// Create a MsgInsertReputerPayload message
	lossesMsg := &types.MsgInsertReputerPayload{
		Sender: reputerAddr.String(),
		ReputerValueBundle: &types.ReputerValueBundle{
			ValueBundle: &reputerValueBundle,
			Signature:   valueBundleSignature,
			Pubkey:      hex.EncodeToString(reputerPublicKeyBytes),
		},
	}

	_, err = s.msgServer.InsertReputerPayload(ctx, lossesMsg)
	require.ErrorIs(err, sdkerrors.ErrUnauthorized)
}
