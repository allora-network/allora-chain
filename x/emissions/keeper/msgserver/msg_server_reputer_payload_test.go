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
	require := s.Require()

	reputerIndexes := testutil.ReturnIndexes(0, 1)
	workerIndexes := testutil.ReturnIndexes(1, 3)
	reputerPrivateKey := s.PrivKeys(0)
	reputerAddr := s.Addrs(0)
	reputerPublicKeyHex := s.PubKeyHexStr(0)
	workerValues := testutil.GetWorkerValuesFromIndexes(workerIndexes, "0.1")
	reputerValues := s.GetReputerValuesFromIndexes(reputerIndexes, workerIndexes, "0.1")

	// Pre-emptive topic pass
	topicId, _ := s.FullTopicPass(workerIndexes, reputerIndexes)
	topic, err := s.EmissionsKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)

	testCases := []struct {
		name               string
		setupBundle        func(bundle *types.InputValueBundle)
		setupNetworkValues func(valueBundle *types.ValueBundle)
		expectedError      string
	}{
		{
			name: "Different inferer sets - missing worker",
			setupBundle: func(bundle *types.InputValueBundle) {
				bundle.InfererValues = bundle.InfererValues[1:]
			},
			expectedError: "worker sets don't match - different unique workers",
		},
		{
			name: "Different forecaster sets - missing worker",
			setupBundle: func(bundle *types.InputValueBundle) {
				bundle.ForecasterValues = bundle.ForecasterValues[1:]
			},
			expectedError: "worker sets don't match - different unique workers",
		},
		{
			name: "Different inferer sets - different unique worker",
			setupBundle: func(bundle *types.InputValueBundle) {
				bundle.InfererValues[0] = &types.InputWorkerAttributedValue{
					Worker: s.AddrsStr(4),
					Value:  bundle.InfererValues[0].Value,
				}
			},
			expectedError: "worker mismatch: expected",
		},
		{
			name: "Different forecaster sets - different unique worker",
			setupBundle: func(bundle *types.InputValueBundle) {
				bundle.ForecasterValues[0] = &types.InputWorkerAttributedValue{
					Worker: s.AddrsStr(4),
					Value:  bundle.ForecasterValues[0].Value,
				}
			},
			expectedError: "worker mismatch: expected",
		},
		{
			name: "Same inferer workers but different frequencies",
			setupBundle: func(bundle *types.InputValueBundle) {
				bundle.InfererValues[1] = &types.InputWorkerAttributedValue{
					Worker: bundle.InfererValues[0].Worker,
					Value:  bundle.InfererValues[0].Value,
				}
			},
			expectedError: "worker mismatch: expected",
		},
		{
			name: "Same forecaster workers but different frequencies",
			setupBundle: func(bundle *types.InputValueBundle) {
				bundle.ForecasterValues[1] = &types.InputWorkerAttributedValue{
					Worker: bundle.ForecasterValues[0].Worker,
					Value:  bundle.ForecasterValues[0].Value,
				}
			},
			expectedError: "worker mismatch: expected",
		},
		{
			name: "Invalid OneOutInfererValues - different worker",
			setupBundle: func(bundle *types.InputValueBundle) {
				bundle.OneOutInfererValues[0].Worker = s.AddrsStr(4)
			},
			expectedError: "worker mismatch: expected",
		},
		{
			name: "Invalid OneOutForecasterValues - different worker",
			setupBundle: func(bundle *types.InputValueBundle) {
				bundle.OneOutForecasterValues[0].Worker = s.AddrsStr(4)
			},
			expectedError: "worker mismatch: expected",
		},
		{
			name: "Invalid OneInForecasterValues - different worker",
			setupBundle: func(bundle *types.InputValueBundle) {
				bundle.OneInForecasterValues[0].Worker = s.AddrsStr(4)
			},
			expectedError: "worker mismatch: expected",
		},
		{
			name: "Invalid OneOutInfererForecasterValues - different forecaster",
			setupBundle: func(bundle *types.InputValueBundle) {
				bundle.OneOutInfererForecasterValues[0].Forecaster = s.AddrsStr(4)
			},
			expectedError: "worker mismatch: expected",
		},
		{
			name: "Invalid OneOutInfererForecasterValues - different inferer",
			setupBundle: func(bundle *types.InputValueBundle) {
				bundle.OneOutInfererForecasterValues[0].OneOutInfererValues[0].Worker = s.AddrsStr(4)
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
			// get next nonce
			nonce, _, err := s.EmissionsKeeper().GetNextPossibleChurningBlockByTopicId(s.Ctx(), topicId)
			s.Require().NoError(err)
			s.WithBlockHeight(nonce)
			s.EndBlock()

			// get unfulfilled worker nonces
			workerNonces, err := s.EmissionsKeeper().GetUnfulfilledWorkerNonces(s.Ctx(), topic.Id)
			s.Require().NoError(err)
			s.Require().True(len(workerNonces.Nonces) > 0)

			// setup inferences
			workerNonce := workerNonces.Nonces[0].BlockHeight
			s.SetupInferences(topicId, workerNonce, workerIndexes, workerValues...)
			wswBlock := nonce + topic.WorkerSubmissionWindow
			s.WithBlockHeight(wswBlock)
			s.EndBlock()

			// modify network inferences if needed
			if tc.setupNetworkValues != nil {
				inferenceBundle, err := s.EmissionsKeeper().GetLatestNetworkInferences(s.Ctx(), topicId, false)
				s.Require().NoError(err)
				tc.setupNetworkValues(inferenceBundle)
				err = s.EmissionsKeeper().InsertNetworkInferences(s.Ctx(), topicId, nonce, *inferenceBundle)
				require.NoError(err)
			}

			// get unfulfilled reputer nonces
			reputerNonces, err := s.EmissionsKeeper().GetUnfulfilledReputerNonces(s.Ctx(), topicId)
			s.Require().NoError(err)
			s.Require().True(len(reputerNonces.Nonces) > 0)

			reputerNonce := reputerNonces.Nonces[0].ReputerNonce.BlockHeight
			reputerTxBlockHeight := reputerNonce + topic.GroundTruthLag + 1
			s.WithBlockHeight(reputerTxBlockHeight)

			genReputerBundleOptions := []testutil.Option{
				testutil.WithReputerValues(reputerValues),
			}

			if tc.setupNetworkValues != nil {
				genReputerBundleOptions = append(genReputerBundleOptions, testutil.WithSkipNetworkInferences())
			}

			// generate reputer value bundle
			reputerValueBundle := s.GenerateLossBundles(reputerNonce, topicId, reputerIndexes, genReputerBundleOptions...).ReputerValueBundles[0].ValueBundle

			// modify reputer value bundle if needed
			if tc.setupBundle != nil {
				tc.setupBundle(reputerValueBundle)
			}

			// generate and insert reputer payload
			valueBundleSignature := s.SignInputValueBundle(reputerValueBundle, reputerPrivateKey)
			lossesMsg := &types.InsertReputerPayloadRequest{
				Sender: reputerAddr.String(),
				ReputerValueBundle: &types.InputReputerValueBundle{
					ValueBundle: reputerValueBundle,
					Signature:   valueBundleSignature,
					Pubkey:      reputerPublicKeyHex,
				},
			}
			_, err = s.EmissionsMsgServer().InsertReputerPayload(s.Ctx(), lossesMsg)

			// check for expected error
			if tc.expectedError != "" {
				require.Error(err)
				require.Contains(err.Error(), tc.expectedError)
			} else {
				require.NoError(err)
			}

			// end blocker to finalize the epoch
			rewardsBlockHeight := nonce + topic.GroundTruthLag + topic.EpochLength
			s.WithBlockHeight(rewardsBlockHeight)
			s.EndBlock()
		})
	}
}
