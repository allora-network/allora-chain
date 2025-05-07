package msgserver_test

import (
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	alloraMath "github.com/allora-network/allora-chain/math"
	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

const block = types.BlockHeight(1)

func (s *MsgServerTestSuite) setUpMsgReputerPayload(
	reputer string,
	reputerAddr sdk.AccAddress,
	worker string,
	workerAddr sdk.AccAddress,
) (
	reputerValueBundle types.InputValueBundle,
	expectedInferences types.Inferences,
	expectedForecasts types.Forecasts,
	topicId uint64,
) {
	require := s.Require()
	keeper := s.EmissionsKeeper()

	params, err := keeper.GetParams(s.Ctx())
	require.NoError(err)

	minStakeScaled := params.RequiredMinimumStake.Mul(inferencesynthesis.CosmosIntOneE18())

	topicId = s.commonStakingSetup(s.Ctx(), reputer, reputerAddr, worker, workerAddr, minStakeScaled)
	s.MintTokensToAddress(reputerAddr, params.RequiredMinimumStake)

	addStakeMsg := &types.AddStakeRequest{
		Sender:  reputerAddr.String(),
		TopicId: topicId,
		Amount:  minStakeScaled,
	}

	_, err = s.EmissionsMsgServer().AddStake(s.Ctx(), addStakeMsg)
	s.Require().NoError(err)

	workerNonce := types.Nonce{
		BlockHeight: block,
	}

	err = keeper.AddWorkerNonce(s.Ctx(), topicId, &workerNonce)
	require.NoError(err)
	_, err = keeper.FulfillWorkerNonce(s.Ctx(), topicId, &workerNonce)
	require.NoError(err)
	err = keeper.AddReputerNonce(s.Ctx(), topicId, &workerNonce)
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

	reputerValueBundle = types.InputValueBundle{
		TopicId:             topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{ReputerNonce: &workerNonce},
		Reputer:             reputerAddr.String(),
		ExtraData:           nil,
		CombinedValue:       alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
		InfererValues: []*types.InputWorkerAttributedValue{
			{
				Worker: workerAddr.String(),
				Value:  alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
			},
		},
		ForecasterValues: []*types.InputWorkerAttributedValue{
			{
				Worker: workerAddr.String(),
				Value:  alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
			},
		},
		NaiveValue:          alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
		OneOutInfererValues: []*types.InputWithheldWorkerAttributedValue{},
		OneOutForecasterValues: []*types.InputWithheldWorkerAttributedValue{
			{
				Worker: workerAddr.String(),
				Value:  alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
			},
		},
		OneInForecasterValues: []*types.InputWorkerAttributedValue{
			{
				Worker: workerAddr.String(),
				Value:  alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
			},
		},
		OneOutInfererForecasterValues: nil,
	}

	return reputerValueBundle, expectedInferences, expectedForecasts, topicId
}

func (s *MsgServerTestSuite) signInputValueBundle(InputReputerValueBundle *types.InputValueBundle, privateKey secp256k1.PrivKey) []byte {
	require := s.Require()
	reputerValueBundle, err := types.NewValueBundleFromInput(InputReputerValueBundle)
	require.NoError(err, "Convert should not return an error")
	return s.signValueBundle(reputerValueBundle, privateKey)
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
	reputerValueBundle *types.InputValueBundle,
) error {
	valueBundleSignature := s.signInputValueBundle(reputerValueBundle, reputerPrivateKey)

	// Create a InsertReputerPayloadRequest message
	lossesMsg := &types.InsertReputerPayloadRequest{
		Sender: reputerAddr.String(),
		ReputerValueBundle: &types.InputReputerValueBundle{
			ValueBundle: reputerValueBundle,
			Signature:   valueBundleSignature,
			Pubkey:      reputerPublicKeyHex,
		},
	}

	_, err := s.EmissionsMsgServer().InsertReputerPayload(s.Ctx(), lossesMsg)
	return err
}

func (s *MsgServerTestSuite) TestMsgInsertReputerPayloadFailsEarlyWindowAndWhitelistCheck() {
	ctx := s.Ctx()
	require := s.Require()
	keeper := s.EmissionsKeeper()

	reputerPrivateKey := s.PrivKeys()[0]
	reputerPublicKeyHex := s.PubKeyHexStr()[0]
	reputerAddr := s.Addrs()[0]
	reputer := s.AddrsStr()[0]

	workerAddr := s.Addrs()[1]
	worker := s.AddrsStr()[1]

	reputerValueBundle, expectedInferences, expectedForecasts, topicId := s.setUpMsgReputerPayload(reputer, reputerAddr, worker, workerAddr)

	err := keeper.InsertActiveForecasts(ctx, topicId, block, expectedForecasts)
	require.NoError(err)

	err = keeper.InsertActiveInferences(ctx, topicId, block, expectedInferences)
	require.NoError(err)

	topic, err := s.EmissionsKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)

	// Prior to the ground truth lag, should not allow reputer payload
	newBlockheight := block + topic.GroundTruthLag - 1
	s.WithBlockHeight(newBlockheight)

	err = s.constructAndInsertReputerPayload(reputerAddr, reputerPrivateKey, reputerPublicKeyHex, &reputerValueBundle)
	require.ErrorIs(err, types.ErrReputerNonceWindowNotAvailable)

	// Valid reputer nonce window, end
	newBlockheight = block + topic.GroundTruthLag*2 + 1
	s.WithBlockHeight(newBlockheight)
	err = s.constructAndInsertReputerPayload(reputerAddr, reputerPrivateKey, reputerPublicKeyHex, &reputerValueBundle)
	require.ErrorIs(err, types.ErrReputerNonceWindowNotAvailable)

	// Remove reputer from whitelist
	err = keeper.RemoveFromGlobalWhitelist(ctx, reputerAddr.String())
	require.NoError(err)
	err = keeper.RemoveFromTopicReputerWhitelist(ctx, topicId, reputerAddr.String())
	require.NoError(err)

	newBlockheight = block + topic.GroundTruthLag*2
	s.WithBlockHeight(newBlockheight)
	err = s.constructAndInsertReputerPayload(reputerAddr, reputerPrivateKey, reputerPublicKeyHex, &reputerValueBundle)
	require.ErrorIs(err, types.ErrNotPermittedToSubmitReputerPayload)

	// Add reputer to whitelist so they could submit payload again
	err = keeper.AddToTopicReputerWhitelist(ctx, topicId, reputerAddr.String())
	require.NoError(err)

	// Valid reputer nonce window, end
	err = s.constructAndInsertReputerPayload(reputerAddr, reputerPrivateKey, reputerPublicKeyHex, &reputerValueBundle)
	require.NoError(err)
}

func (s *MsgServerTestSuite) TestMsgInsertReputerPayloadReputerNotMatchSignature() {
	ctx := s.Ctx()
	require := s.Require()
	keeper := s.EmissionsKeeper()

	reputerPrivateKey := s.PrivKeys()[0]
	reputerAddr := s.Addrs()[0]
	reputer := s.AddrsStr()[0]
	reputerPublicKeyHex := s.PubKeyHexStr()[0]
	workerAddr := s.Addrs()[1]
	worker := s.AddrsStr()[1]

	reputerValueBundle, expectedInferences, expectedForecasts, topicId := s.setUpMsgReputerPayload(reputer, reputerAddr, worker, workerAddr)

	err := keeper.InsertActiveForecasts(ctx, topicId, block, expectedForecasts)
	require.NoError(err)

	err = keeper.InsertActiveInferences(ctx, topicId, block, expectedInferences)
	require.NoError(err)

	topic, err := s.EmissionsKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)

	// Prior to the ground truth lag, should not allow reputer payload
	newBlockheight := block + topic.GroundTruthLag - 1
	s.WithBlockHeight(newBlockheight)

	reputerValueBundle.Reputer = s.AddrsStr()[3]
	valueBundleSignature := s.signInputValueBundle(&reputerValueBundle, reputerPrivateKey)

	// Create a InsertReputerPayloadRequest message
	lossesMsg := &types.InsertReputerPayloadRequest{
		Sender: reputerAddr.String(),
		ReputerValueBundle: &types.InputReputerValueBundle{
			ValueBundle: &reputerValueBundle,
			Signature:   valueBundleSignature,
			Pubkey:      reputerPublicKeyHex,
		},
	}

	_, err = s.EmissionsMsgServer().InsertReputerPayload(ctx, lossesMsg)
	require.ErrorIs(err, sdkerrors.ErrUnauthorized)
}
