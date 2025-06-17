package msgserver_test

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *MsgServerTestSuite) TestMsgInsertReputerPayloadFailsEarlyWindowAndWhitelistCheck() {
	reputerIndexes := testutil.ReturnIndexes(0, 1)
	workerIndexes := testutil.ReturnIndexes(1, 1)
	reputerAddr := s.Addrs(reputerIndexes[0])
	reputerValues := s.GetReputerValuesFromIndexes(reputerIndexes, workerIndexes, "0.1")

	topic := s.FullTopicSetup(workerIndexes, reputerIndexes)

	nonce, _, _ := s.EmissionsKeeper().GetNextPossibleChurningBlockByTopicId(s.Ctx(), topic.Id)
	s.WithBlockHeight(nonce)
	s.EndBlock()

	s.SetupInferences(topic.Id, nonce, workerIndexes)
	newBlockheight := nonce + topic.WorkerSubmissionWindow
	s.WithBlockHeight(newBlockheight)
	s.CloseWorkerNonce(topic, types.Nonce{BlockHeight: nonce})

	// Prior to the ground truth lag, should not allow reputer payload
	newBlockheight = nonce + topic.GroundTruthLag - 1
	s.WithBlockHeight(newBlockheight)

	err := s.InsertReputerLossBundle(
		topic.GetId(),
		nonce,
		reputerIndexes,
		testutil.WithReputerValues(reputerValues),
		testutil.WithSkipNetworkInferences(),
	)
	s.Require().ErrorIs(err, types.ErrReputerNonceWindowNotAvailable)

	// Valid reputer nonce window, end
	newBlockheight = nonce + topic.GroundTruthLag*2 + 1
	s.WithBlockHeight(newBlockheight)
	err = s.InsertReputerLossBundle(topic.GetId(), nonce, reputerIndexes)
	s.Require().ErrorIs(err, types.ErrReputerNonceWindowNotAvailable)

	// Remove reputer from whitelist
	err = s.EmissionsKeeper().RemoveFromGlobalWhitelist(s.Ctx(), reputerAddr.String())
	s.Require().NoError(err)
	err = s.EmissionsKeeper().RemoveFromTopicReputerWhitelist(s.Ctx(), topic.Id, reputerAddr.String())
	s.Require().NoError(err)

	newBlockheight = nonce + topic.GroundTruthLag*2
	s.WithBlockHeight(newBlockheight)
	err = s.InsertReputerLossBundle(topic.GetId(), nonce, reputerIndexes)
	s.Require().ErrorIs(err, types.ErrNotPermittedToSubmitReputerPayload)

	// Add reputer to whitelist so they could submit payload again
	err = s.EmissionsKeeper().AddToTopicReputerWhitelist(s.Ctx(), topic.Id, reputerAddr.String())
	s.Require().NoError(err)

	// Valid reputer nonce window, end
	err = s.InsertReputerLossBundle(topic.GetId(), nonce, reputerIndexes)
	s.Require().NoError(err)
}

func (s *MsgServerTestSuite) TestMsgInsertReputerPayloadReputerNotMatchSignature() {
	reputerIndexes := testutil.ReturnIndexes(0, 1)
	reputerAddr := s.Addrs(reputerIndexes[0])
	reputerPrivateKey := s.PrivKeys(0)
	reputerPublicKeyHex := s.PubKeyHexStr(0)
	topicId := uint64(1)

	unauthReputer := s.AddrsStr(3)
	//nolint:exhaustruct
	inputValueBundle := &types.InputValueBundle{
		TopicId:             topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{ReputerNonce: &types.Nonce{BlockHeight: 1}},
		Reputer:             unauthReputer,
		InfererValues:       []*types.InputWorkerAttributedValue{{Worker: s.AddrsStr(0)}},
	}
	valueBundleSignature := s.SignInputValueBundle(inputValueBundle, reputerPrivateKey)

	// Create a InsertReputerPayloadRequest message
	lossesMsg := &types.InsertReputerPayloadRequest{
		Sender: reputerAddr.String(),
		ReputerValueBundle: &types.InputReputerValueBundle{
			ValueBundle: inputValueBundle,
			Signature:   valueBundleSignature,
			Pubkey:      reputerPublicKeyHex,
		},
	}

	_, err := s.EmissionsMsgServer().InsertReputerPayload(s.Ctx(), lossesMsg)
	s.Require().ErrorIs(err, sdkerrors.ErrUnauthorized)
}

// nolint: exhaustruct
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
