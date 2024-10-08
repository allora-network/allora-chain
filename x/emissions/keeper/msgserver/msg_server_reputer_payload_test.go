package msgserver_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const block = types.BlockHeight(1)

func (s *MsgServerTestSuite) setUpMsgReputerPayload(
	reputer string,
	reputerAddr sdk.AccAddress,
	worker string,
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

	topicId = s.commonStakingSetup(ctx, reputer, reputerAddr, worker, workerAddr, minStakeScaled)
	s.MintTokensToAddress(reputerAddr, params.RequiredMinimumStake)

	addStakeMsg := &types.AddStakeRequest{
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
				TopicId:     topicId,
				BlockHeight: block,
				Value:       alloraMath.NewDecFromInt64(1), // Assuming NewDecFromInt64 exists and is appropriate
				Inferer:     workerAddr.String(),
			},
		},
	}

	expectedForecasts = types.Forecasts{
		Forecasts: []*types.Forecast{
			{
				TopicId:     topicId,
				BlockHeight: block,
				Forecaster:  workerAddr.String(),
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: workerAddr.String(),
						Value:   alloraMath.NewDecFromInt64(1),
					},
				},
			},
		},
	}

	reputerValueBundle = types.ValueBundle{
		TopicId:             topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{ReputerNonce: &workerNonce},
		Reputer:             reputerAddr.String(),
		ExtraData:           nil,
		CombinedValue:       alloraMath.NewDecFromInt64(100),
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
		OneOutInfererForecasterValues: nil,
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
	reputerPublicKeyHex string,
	reputerValueBundle *types.ValueBundle,
) error {
	ctx, msgServer := s.ctx, s.msgServer
	valueBundleSignature := s.signValueBundle(reputerValueBundle, reputerPrivateKey)

	// Create a InsertReputerPayloadRequest message
	lossesMsg := &types.InsertReputerPayloadRequest{
		Sender: reputerAddr.String(),
		ReputerValueBundle: &types.ReputerValueBundle{
			ValueBundle: reputerValueBundle,
			Signature:   valueBundleSignature,
			Pubkey:      reputerPublicKeyHex,
		},
	}

	_, err := msgServer.InsertReputerPayload(ctx, lossesMsg)
	return err
}

func (s *MsgServerTestSuite) TestMsgInsertReputerPayloadFailsEarlyWindow() {
	ctx := s.ctx
	require := s.Require()
	keeper := s.emissionsKeeper

	reputerPrivateKey := s.privKeys[0]
	reputerPublicKeyHex := s.pubKeyHexStr[0]
	reputerAddr := s.addrs[0]
	reputer := s.addrsStr[0]

	workerAddr := s.addrs[1]
	worker := s.addrsStr[1]

	reputerValueBundle, expectedInferences, expectedForecasts, topicId := s.setUpMsgReputerPayload(reputer, reputerAddr, worker, workerAddr)

	err := keeper.InsertActiveForecasts(ctx, topicId, block, expectedForecasts)
	require.NoError(err)

	err = keeper.InsertActiveInferences(ctx, topicId, block, expectedInferences)
	require.NoError(err)

	topic, err := s.emissionsKeeper.GetTopic(s.ctx, topicId)
	s.Require().NoError(err)

	// Prior to the ground truth lag, should not allow reputer payload
	newBlockheight := block + topic.GroundTruthLag - 1
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(newBlockheight)

	err = s.constructAndInsertReputerPayload(reputerAddr, reputerPrivateKey, reputerPublicKeyHex, &reputerValueBundle)
	require.ErrorIs(err, types.ErrReputerNonceWindowNotAvailable)

	// Valid reputer nonce window, end
	newBlockheight = block + topic.GroundTruthLag*2 + 1
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(newBlockheight)
	err = s.constructAndInsertReputerPayload(reputerAddr, reputerPrivateKey, reputerPublicKeyHex, &reputerValueBundle)
	require.ErrorIs(err, types.ErrReputerNonceWindowNotAvailable)

	// Valid reputer nonce window, end
	newBlockheight = block + topic.GroundTruthLag*2
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(newBlockheight)
	err = s.constructAndInsertReputerPayload(reputerAddr, reputerPrivateKey, reputerPublicKeyHex, &reputerValueBundle)
	require.NoError(err)
}

func (s *MsgServerTestSuite) TestMsgInsertReputerPayloadReputerNotMatchSignature() {
	ctx := s.ctx
	require := s.Require()
	keeper := s.emissionsKeeper

	reputerPrivateKey := s.privKeys[0]
	reputerAddr := s.addrs[0]
	reputer := s.addrsStr[0]
	reputerPublicKeyHex := s.pubKeyHexStr[0]
	workerAddr := s.addrs[1]
	worker := s.addrsStr[1]

	reputerValueBundle, expectedInferences, expectedForecasts, topicId := s.setUpMsgReputerPayload(reputer, reputerAddr, worker, workerAddr)

	err := keeper.InsertActiveForecasts(ctx, topicId, block, expectedForecasts)
	require.NoError(err)

	err = keeper.InsertActiveInferences(ctx, topicId, block, expectedInferences)
	require.NoError(err)

	topic, err := s.emissionsKeeper.GetTopic(s.ctx, topicId)
	s.Require().NoError(err)

	// Prior to the ground truth lag, should not allow reputer payload
	newBlockheight := block + topic.GroundTruthLag - 1
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(newBlockheight)

	reputerValueBundle.Reputer = s.addrsStr[3]
	valueBundleSignature := s.signValueBundle(&reputerValueBundle, reputerPrivateKey)

	// Create a InsertReputerPayloadRequest message
	lossesMsg := &types.InsertReputerPayloadRequest{
		Sender: reputerAddr.String(),
		ReputerValueBundle: &types.ReputerValueBundle{
			ValueBundle: &reputerValueBundle,
			Signature:   valueBundleSignature,
			Pubkey:      reputerPublicKeyHex,
		},
	}

	_, err = s.msgServer.InsertReputerPayload(ctx, lossesMsg)
	require.ErrorIs(err, sdkerrors.ErrUnauthorized)
}
