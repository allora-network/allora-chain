package msgserver_test

import (
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	alloraMath "github.com/allora-network/allora-chain/math"
	actorutils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
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
	expectedOneOutInfererValues,
	expectedOneOutForecasterValues []*types.WithheldWorkerAttributedValue,
	expectedOneInForecasterValues []*types.WorkerAttributedValue,
	expectedOneOutInfererForecasterValues []*types.OneOutInfererForecasterValues,
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
	oneOutInfererValues := make([]*types.InputWithheldWorkerAttributedValue, 0, len(workerAddr))
	oneOutForecasterValues := make([]*types.InputWithheldWorkerAttributedValue, 0, len(workerAddr))
	oneInForecasterValues := make([]*types.InputWorkerAttributedValue, 0, len(workerAddr))
	oneOutInfererForecasterValues := make([]*types.InputOneOutInfererForecasterValues, 0, len(workerAddr))

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

		expectedOneOutForecasterValues = append(expectedOneOutForecasterValues, &types.WithheldWorkerAttributedValue{
			Worker: worker.String(),
			Value:  alloraMath.NewDecFromInt64(100),
		})

		oneInForecasterValues = append(oneInForecasterValues, &types.InputWorkerAttributedValue{
			Worker: worker.String(),
			Value:  alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
		})

		expectedOneInForecasterValues = append(expectedOneInForecasterValues, &types.WorkerAttributedValue{
			Worker: worker.String(),
			Value:  alloraMath.NewDecFromInt64(100),
		})

		oneOutInfererValues = append(oneOutInfererValues, &types.InputWithheldWorkerAttributedValue{
			Worker: worker.String(),
			Value:  alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
		})

		expectedOneOutInfererValues = append(expectedOneOutInfererValues, &types.WithheldWorkerAttributedValue{
			Worker: worker.String(),
			Value:  alloraMath.NewDecFromInt64(100),
		})

		infererOneOutValues := make([]*types.InputWithheldWorkerAttributedValue, 0, len(workerAddr))
		for _, inferWorker := range workerAddr {
			infererOneOutValues = append(infererOneOutValues, &types.InputWithheldWorkerAttributedValue{
				Worker: inferWorker.String(),
				Value:  alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
			})
		}

		oneOutInfererForecasterValues = append(oneOutInfererForecasterValues, &types.InputOneOutInfererForecasterValues{
			Forecaster:          worker.String(),
			OneOutInfererValues: infererOneOutValues,
		})

		expectedOneOutInfererForecasterValues = append(expectedOneOutInfererForecasterValues, &types.OneOutInfererForecasterValues{
			Forecaster:          worker.String(),
			OneOutInfererValues: convertWithheldWorkerValues(infererOneOutValues),
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
		OneOutInfererValues:           oneOutInfererValues,
		OneOutForecasterValues:        oneOutForecasterValues,
		OneInForecasterValues:         oneInForecasterValues,
		OneOutInfererForecasterValues: oneOutInfererForecasterValues,
	}

	return reputerValueBundle,
		expectedInferences,
		expectedForecasts,
		expectedOneOutInfererValues,
		expectedOneOutForecasterValues,
		expectedOneInForecasterValues,
		expectedOneOutInfererForecasterValues,
		topicId
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

	reputerValueBundle, expectedInferences, expectedForecasts, _, _, _, _, topicId := s.setUpMsgReputerPayload(reputerAddr, workerAddr)
	reputerValueBundle.ForecasterValues = nil              // would require existing previous network losses
	reputerValueBundle.OneOutForecasterValues = nil        // would require existing previous network losses
	reputerValueBundle.OneOutInfererValues = nil           // would require existing previous network losses
	reputerValueBundle.OneInForecasterValues = nil         // would require existing previous network losses
	reputerValueBundle.OneOutInfererForecasterValues = nil // would require existing previous network losses

	err := s.emissionsKeeper.AddActiveInferer(ctx, topicId, workerAddr.String())
	s.Require().NoError(err)

	// Insert unfullfiled nonces
	err = s.emissionsKeeper.AddWorkerNonce(s.ctx, topicId, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)
	err = s.emissionsKeeper.AddReputerNonce(s.ctx, topicId, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)

	err = keeper.InsertActiveForecasts(ctx, topicId, block, expectedForecasts)
	require.NoError(err)

	err = keeper.InsertInference(ctx, topicId, *expectedInferences.Inferences[0])
	require.NoError(err)

	topic, err := s.emissionsKeeper.GetTopic(s.ctx, topicId)
	s.Require().NoError(err)

	// Move to end of worker submission window
	s.ctx = s.ctx.WithBlockHeight(block + topic.WorkerSubmissionWindow)
	err = actorutils.CloseWorkerNonce(&s.emissionsKeeper, s.ctx, topic, *reputerValueBundle.ReputerRequestNonce.ReputerNonce)
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

	reputerValueBundle, expectedInferences, expectedForecasts, _, _, _, _, topicId := s.setUpMsgReputerPayload(reputerAddr, workerAddr)

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
		name               string
		setupBundle        func(bundle types.InputValueBundle) types.InputValueBundle
		setupNetworkValues func(valueBundle *types.ValueBundle)
		expectedError      string
	}{
		{
			name: "Different inferer sets - missing worker",
			setupBundle: func(bundle types.InputValueBundle) types.InputValueBundle {
				bundle.InfererValues = bundle.InfererValues[1:]
				return bundle
			},
			expectedError: "worker sets don't match - different unique workers",
		},
		{
			name: "Different forecaster sets - missing worker",
			setupBundle: func(bundle types.InputValueBundle) types.InputValueBundle {
				bundle.ForecasterValues = bundle.ForecasterValues[1:]
				return bundle
			},
			expectedError: "worker sets don't match - different unique workers",
		},
		{
			name: "Different inferer sets - different unique worker",
			setupBundle: func(bundle types.InputValueBundle) types.InputValueBundle {
				bundle.InfererValues[0] = &types.InputWorkerAttributedValue{
					Worker: s.addrsStr[4],
					Value:  bundle.InfererValues[0].Value,
				}
				return bundle
			},
			expectedError: "worker mismatch: expected",
		},
		{
			name: "Different forecaster sets - different unique worker",
			setupBundle: func(bundle types.InputValueBundle) types.InputValueBundle {
				bundle.ForecasterValues[0] = &types.InputWorkerAttributedValue{
					Worker: s.addrsStr[4],
					Value:  bundle.ForecasterValues[0].Value,
				}
				return bundle
			},
			expectedError: "worker mismatch: expected",
		},
		{
			name: "Same inferer workers but different frequencies",
			setupBundle: func(bundle types.InputValueBundle) types.InputValueBundle {
				bundle.InfererValues[1] = &types.InputWorkerAttributedValue{
					Worker: bundle.InfererValues[0].Worker,
					Value:  bundle.InfererValues[0].Value,
				}
				return bundle
			},
			expectedError: "worker mismatch: expected",
		},
		{
			name: "Same forecaster workers but different frequencies",
			setupBundle: func(bundle types.InputValueBundle) types.InputValueBundle {
				bundle.ForecasterValues[1] = &types.InputWorkerAttributedValue{
					Worker: bundle.ForecasterValues[0].Worker,
					Value:  bundle.ForecasterValues[0].Value,
				}
				return bundle
			},
			expectedError: "worker mismatch: expected",
		},
		{
			name: "Invalid OneOutInfererValues - different worker",
			setupBundle: func(bundle types.InputValueBundle) types.InputValueBundle {
				bundle.OneOutInfererValues[0].Worker = s.addrsStr[4]
				return bundle
			},
			expectedError: "worker mismatch: expected",
		},
		{
			name: "Invalid OneOutForecasterValues - different worker",
			setupBundle: func(bundle types.InputValueBundle) types.InputValueBundle {
				bundle.OneOutForecasterValues[0].Worker = s.addrsStr[4]
				return bundle
			},
			expectedError: "worker mismatch: expected",
		},
		{
			name: "Invalid OneInForecasterValues - different worker",
			setupBundle: func(bundle types.InputValueBundle) types.InputValueBundle {
				bundle.OneInForecasterValues[0].Worker = s.addrsStr[4]
				return bundle
			},
			expectedError: "worker mismatch: expected",
		},
		{
			name: "Invalid OneOutInfererForecasterValues - different forecaster",
			setupBundle: func(bundle types.InputValueBundle) types.InputValueBundle {
				bundle.OneOutInfererForecasterValues[0].Forecaster = s.addrsStr[4]
				return bundle
			},
			expectedError: "worker mismatch: expected",
		},
		{
			name: "Invalid OneOutInfererForecasterValues - different inferer",
			setupBundle: func(bundle types.InputValueBundle) types.InputValueBundle {
				bundle.OneOutInfererForecasterValues[0].OneOutInfererValues[0].Worker = s.addrsStr[4]
				return bundle
			},
			expectedError: "worker mismatch: expected",
		},
		{
			name: "Modified network inferences worker frequency",
			setupNetworkValues: func(valueBundle *types.ValueBundle) {
				valueBundle.InfererValues[1] = &types.WorkerAttributedValue{
					Worker: valueBundle.InfererValues[0].Worker,
					Value:  valueBundle.InfererValues[0].Value,
				}
			},
			expectedError: "worker mismatch: expected",
		},
		{
			name: "Modified network forecasts worker frequency",
			setupNetworkValues: func(valueBundle *types.ValueBundle) {
				valueBundle.InfererValues[1] = &types.WorkerAttributedValue{
					Worker: valueBundle.ForecasterValues[0].Worker,
					Value:  valueBundle.ForecasterValues[0].Value,
				}
			},
			expectedError: "worker mismatch: expected",
		},
		{
			name: "Modified network OneOutInfererValues worker frequency",
			setupNetworkValues: func(valueBundle *types.ValueBundle) {
				valueBundle.OneOutInfererValues[1] = &types.WithheldWorkerAttributedValue{
					Worker: valueBundle.OneOutInfererValues[0].Worker,
					Value:  valueBundle.OneOutInfererValues[0].Value,
				}
			},
			expectedError: "worker mismatch: expected",
		},
		{
			name: "Modified network OneOutForecasterValues worker frequency",
			setupNetworkValues: func(valueBundle *types.ValueBundle) {
				valueBundle.OneOutForecasterValues[1] = &types.WithheldWorkerAttributedValue{
					Worker: valueBundle.OneOutForecasterValues[0].Worker,
					Value:  valueBundle.OneOutForecasterValues[0].Value,
				}
			},
			expectedError: "worker mismatch: expected",
		},
		{
			name: "Modified network OneInForecasterValues worker frequency",
			setupNetworkValues: func(valueBundle *types.ValueBundle) {
				valueBundle.OneInForecasterValues[1] = &types.WorkerAttributedValue{
					Worker: valueBundle.OneInForecasterValues[0].Worker,
					Value:  valueBundle.OneInForecasterValues[0].Value,
				}
			},
			expectedError: "worker mismatch: expected",
		},
		{
			name: "Modified network OneOutInfererForecasterValues forecaster frequency",
			setupNetworkValues: func(valueBundle *types.ValueBundle) {
				valueBundle.OneOutInfererForecasterValues[1] = &types.OneOutInfererForecasterValues{
					Forecaster:          valueBundle.OneOutInfererForecasterValues[0].Forecaster,
					OneOutInfererValues: valueBundle.OneOutInfererForecasterValues[0].OneOutInfererValues,
				}
			},
			expectedError: "worker mismatch: expected",
		},
		{
			name: "Modified network OneOutInfererForecasterValues inferer frequency",
			setupNetworkValues: func(valueBundle *types.ValueBundle) {
				valueBundle.OneOutInfererForecasterValues[0].OneOutInfererValues[1] = &types.WithheldWorkerAttributedValue{
					Worker: valueBundle.OneOutInfererForecasterValues[0].OneOutInfererValues[0].Worker,
					Value:  valueBundle.OneOutInfererForecasterValues[0].OneOutInfererValues[0].Value,
				}
			},
			expectedError: "worker mismatch: expected",
		},
		{
			name:          "Valid worker sets match with all fields populated",
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

			reputerValueBundle,
				expectedInferences,
				expectedForecasts,
				oneOutInfererValues,
				oneOutForecasterValues,
				oneInForecasterValues,
				oneOutInfererForecasterValues,
				topicId := s.setUpMsgReputerPayload(reputerAddr, workerAddr1, workerAddr2, workerAddr3)
			inferenceValues := convertInferencesToWorkerValues(expectedInferences)
			forecastValues := convertForecastsToWorkerValues(expectedForecasts)

			valueBundle := types.ValueBundle{
				TopicId: topicId,
				ReputerRequestNonce: &types.ReputerRequestNonce{
					ReputerNonce: &types.Nonce{BlockHeight: block},
				},
				Reputer:                       reputerAddr.String(),
				InfererValues:                 inferenceValues,
				ForecasterValues:              forecastValues,
				OneOutInfererValues:           oneOutInfererValues,
				OneOutForecasterValues:        oneOutForecasterValues,
				OneInForecasterValues:         oneInForecasterValues,
				OneOutInfererForecasterValues: oneOutInfererForecasterValues,
			}

			if tc.setupNetworkValues != nil {
				tc.setupNetworkValues(&valueBundle)
			}

			err := keeper.InsertNetworkInferences(s.ctx, topicId, block, valueBundle)
			require.NoError(err)

			topic, err := s.emissionsKeeper.GetTopic(s.ctx, topicId)
			require.NoError(err)

			newBlockheight := block + topic.GroundTruthLag
			s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(newBlockheight)

			if tc.setupBundle != nil {
				reputerValueBundle = tc.setupBundle(reputerValueBundle)
			}

			valueBundleSignature := s.signInputValueBundle(&reputerValueBundle, reputerPrivateKey)

			lossesMsg := &types.InsertReputerPayloadRequest{
				Sender: reputerAddr.String(),
				ReputerValueBundle: &types.InputReputerValueBundle{
					ValueBundle: &reputerValueBundle,
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

func convertInferencesToWorkerValues(inferences types.Inferences) []*types.WorkerAttributedValue {
	workerValues := make([]*types.WorkerAttributedValue, len(inferences.Inferences))
	for i, infVal := range inferences.Inferences {
		workerValues[i] = &types.WorkerAttributedValue{
			Worker: infVal.Inferer,
			Value:  infVal.Value,
		}
	}
	return workerValues
}

func convertForecastsToWorkerValues(forecasts types.Forecasts) []*types.WorkerAttributedValue {
	workerValues := make([]*types.WorkerAttributedValue, len(forecasts.Forecasts))
	for i, forecastVal := range forecasts.Forecasts {
		workerValues[i] = &types.WorkerAttributedValue{
			Worker: forecastVal.Forecaster,
			Value:  alloraMath.NewDecFromInt64(1),
		}
	}
	return workerValues
}

func convertWithheldWorkerValues(values []*types.InputWithheldWorkerAttributedValue) []*types.WithheldWorkerAttributedValue {
	workerValues := make([]*types.WithheldWorkerAttributedValue, len(values))
	for i, val := range values {
		dec, _ := val.Value.ToDec()
		workerValues[i] = &types.WithheldWorkerAttributedValue{
			Worker: val.Worker,
			Value:  dec,
		}
	}
	return workerValues
}
