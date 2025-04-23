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
	reputerAddr sdk.AccAddress,
	workerAddr ...sdk.AccAddress,
) (
	reputerValueBundle types.InputValueBundle,
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

	topicId = s.commonStakingSetup(ctx, reputerAddr, minStakeScaled, workerAddr...)
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

	inferences := make([]*types.Inference, 0, len(workerAddr))
	forecasts := make([]*types.Forecast, 0, len(workerAddr))
	infererValues := make([]*types.InputWorkerAttributedValue, 0, len(workerAddr))
	forecasterValues := make([]*types.InputWorkerAttributedValue, 0, len(workerAddr))
	oneOutForecasterValues := make([]*types.InputWithheldWorkerAttributedValue, 0, len(workerAddr))
	oneInForecasterValues := make([]*types.InputWorkerAttributedValue, 0, len(workerAddr))

	for _, worker := range workerAddr {
		inferences = append(inferences, &types.Inference{
			TopicId:     topicId,
			BlockHeight: block,
			Value:       alloraMath.NewDecFromInt64(1),
			Inferer:     worker.String(),
		})

		forecastElements := make([]*types.ForecastElement, 0, len(workerAddr))
		for _, inferWorker := range workerAddr {
			forecastElements = append(forecastElements, &types.ForecastElement{
				Inferer: inferWorker.String(),
				Value:   alloraMath.NewDecFromInt64(1),
			})
		}

		forecasts = append(forecasts, &types.Forecast{
			TopicId:          topicId,
			BlockHeight:      block,
			Forecaster:       worker.String(),
			ForecastElements: forecastElements,
		})

		infererValues = append(infererValues, &types.InputWorkerAttributedValue{
			Worker: worker.String(),
			Value:  alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
		})

		forecasterValues = append(forecasterValues, &types.InputWorkerAttributedValue{
			Worker: worker.String(),
			Value:  alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
		})

		oneOutForecasterValues = append(oneOutForecasterValues, &types.InputWithheldWorkerAttributedValue{
			Worker: worker.String(),
			Value:  alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
		})

		oneInForecasterValues = append(oneInForecasterValues, &types.InputWorkerAttributedValue{
			Worker: worker.String(),
			Value:  alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
		})
	}

	expectedInferences = types.Inferences{
		Inferences: inferences,
	}

	expectedForecasts = types.Forecasts{
		Forecasts: forecasts,
	}

	reputerValueBundle = types.InputValueBundle{
		TopicId:                       topicId,
		ReputerRequestNonce:           &types.ReputerRequestNonce{ReputerNonce: &workerNonce},
		Reputer:                       reputerAddr.String(),
		ExtraData:                     nil,
		CombinedValue:                 alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
		InfererValues:                 infererValues,
		ForecasterValues:              forecasterValues,
		NaiveValue:                    alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
		OneOutInfererValues:           []*types.InputWithheldWorkerAttributedValue{},
		OneOutForecasterValues:        oneOutForecasterValues,
		OneInForecasterValues:         oneInForecasterValues,
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
	ctx, msgServer := s.ctx, s.msgServer
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

	_, err := msgServer.InsertReputerPayload(ctx, lossesMsg)
	return err
}

func (s *MsgServerTestSuite) TestMsgInsertReputerPayloadFailsEarlyWindowAndWhitelistCheck() {
	ctx := s.ctx
	require := s.Require()
	keeper := s.emissionsKeeper

	reputerPrivateKey := s.privKeys[0]
	reputerPublicKeyHex := s.pubKeyHexStr[0]
	reputerAddr := s.addrs[0]

	workerAddr := s.addrs[1]

	reputerValueBundle, expectedInferences, expectedForecasts, topicId := s.setUpMsgReputerPayload(reputerAddr, workerAddr)

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

	// Remove reputer from whitelist
	err = keeper.RemoveFromGlobalWhitelist(ctx, reputerAddr.String())
	require.NoError(err)
	err = keeper.RemoveFromTopicReputerWhitelist(ctx, topicId, reputerAddr.String())
	require.NoError(err)

	newBlockheight = block + topic.GroundTruthLag*2
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(newBlockheight)
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
	ctx := s.ctx
	require := s.Require()
	keeper := s.emissionsKeeper

	reputerPrivateKey := s.privKeys[0]
	reputerAddr := s.addrs[0]
	reputerPublicKeyHex := s.pubKeyHexStr[0]
	workerAddr := s.addrs[1]

	reputerValueBundle, expectedInferences, expectedForecasts, topicId := s.setUpMsgReputerPayload(reputerAddr, workerAddr)

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

	_, err = s.msgServer.InsertReputerPayload(ctx, lossesMsg)
	require.ErrorIs(err, sdkerrors.ErrUnauthorized)
}

func (s *MsgServerTestSuite) TestMsgInsertReputerPayloadWorkerAddressValidation() {
	testCases := []struct {
		name            string
		setupBundle     func(bundle types.InputValueBundle) types.InputValueBundle
		setupInferences func(inferences types.Inferences) types.Inferences
		expectedError   string
	}{
		{
			name: "Different worker sets - missing worker",
			setupBundle: func(bundle types.InputValueBundle) types.InputValueBundle {
				bundle.InfererValues = bundle.InfererValues[1:]
				return bundle
			},
			setupInferences: func(inferences types.Inferences) types.Inferences {
				return inferences
			},
			expectedError: "worker sets don't match",
		},
		{
			name: "Different worker sets - different unique worker",
			setupBundle: func(bundle types.InputValueBundle) types.InputValueBundle {
				modifiedBundle := bundle
				for i, infVal := range bundle.InfererValues {
					modifiedBundle.InfererValues[i] = &types.InputWorkerAttributedValue{
						Worker: infVal.Worker,
						Value:  infVal.Value,
					}
				}
				modifiedBundle.InfererValues[0].Worker = s.addrsStr[4]
				return modifiedBundle
			},
			setupInferences: func(inferences types.Inferences) types.Inferences {
				return inferences
			},
			expectedError: "worker frequency mismatch",
		},
		{
			name: "Same workers but different frequencies",
			setupBundle: func(bundle types.InputValueBundle) types.InputValueBundle {
				return bundle
			},
			setupInferences: func(inferences types.Inferences) types.Inferences {
				duplicateInferences := inferences
				duplicateInferences.Inferences[1] = duplicateInferences.Inferences[0]
				return duplicateInferences
			},
			expectedError: "worker frequency mismatch",
		},
		{
			name: "Valid worker sets match",
			setupBundle: func(bundle types.InputValueBundle) types.InputValueBundle {
				return bundle
			},
			setupInferences: func(inferences types.Inferences) types.Inferences {
				return inferences
			},
			expectedError: "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			require := s.Require()
			keeper := s.emissionsKeeper

			reputerPrivateKey := s.privKeys[0]
			reputerAddr := s.addrs[0]
			reputerPublicKeyHex := s.pubKeyHexStr[0]
			workerAddr1 := s.addrs[1]
			workerAddr2 := s.addrs[2]
			workerAddr3 := s.addrs[3]

			reputerValueBundle, expectedInferences, expectedForecasts, topicId := s.setUpMsgReputerPayload(reputerAddr, workerAddr1, workerAddr2, workerAddr3)

			err := keeper.InsertActiveForecasts(s.ctx, topicId, block, expectedForecasts)
			require.NoError(err)

			modifiedInferences := tc.setupInferences(expectedInferences)
			err = keeper.InsertActiveInferences(s.ctx, topicId, block, modifiedInferences)
			require.NoError(err)

			topic, err := s.emissionsKeeper.GetTopic(s.ctx, topicId)
			require.NoError(err)

			newBlockheight := block + topic.GroundTruthLag
			s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(newBlockheight)

			modifiedBundle := tc.setupBundle(reputerValueBundle)
			valueBundleSignature := s.signInputValueBundle(&modifiedBundle, reputerPrivateKey)

			lossesMsg := &types.InsertReputerPayloadRequest{
				Sender: reputerAddr.String(),
				ReputerValueBundle: &types.InputReputerValueBundle{
					ValueBundle: &modifiedBundle,
					Signature:   valueBundleSignature,
					Pubkey:      reputerPublicKeyHex,
				},
			}

			_, err = s.msgServer.InsertReputerPayload(s.ctx, lossesMsg)

			if tc.expectedError != "" {
				require.Error(err)
				require.Contains(err.Error(), tc.expectedError)
			} else {
				require.NoError(err)
			}
		})
	}
}
