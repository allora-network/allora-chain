package msgserver_test

import (
	"encoding/hex"

	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func getNewAddress() (sdk.AccAddress, string) {
	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	return addr, addr.String()
}

func (s *MsgServerTestSuite) setUpMsgInsertWorkerPayload(
	workerPrivateKey secp256k1.PrivKey,
) (types.InsertWorkerPayloadRequest, uint64) {
	return s.setUpMsgInsertWorkerPayloadWithBlockHeight(workerPrivateKey, 1)
}
func (s *MsgServerTestSuite) setUpMsgInsertWorkerPayloadWithBlockHeight(
	workerPrivateKey secp256k1.PrivKey,
	blockHeight int64,
) (types.InsertWorkerPayloadRequest, uint64) {
	ctx := s.Ctx()
	keeper := s.EmissionsKeeper()
	nonce := types.Nonce{BlockHeight: blockHeight}
	topic := uint64(1)

	// Mock setup for addresses
	reputerAddr, reputer := getNewAddress()
	workerAddr := sdk.AccAddress(workerPrivateKey.PubKey().Address())
	worker := workerAddr.String()
	_, Inferer2 := getNewAddress()
	_, Inferer3 := getNewAddress()
	_, Inferer4 := getNewAddress()

	// Define sample OffchainNode information for a worker
	workerInfo := types.OffchainNode{
		Owner:       worker,
		NodeAddress: worker,
	}

	moduleParams, err := keeper.GetParams(ctx)
	s.Require().NoError(err)

	// Create topic 0 and register reputer in it
	s.commonStakingSetup(ctx, reputer, reputerAddr, worker, workerAddr, moduleParams.RegistrationFee)
	err = keeper.AddWorkerNonce(ctx, topic, &nonce)
	s.Require().NoError(err)
	err = keeper.InsertWorker(ctx, topic, worker, workerInfo)
	s.Require().NoError(err)
	err = keeper.InsertWorker(ctx, topic, Inferer2, workerInfo)
	s.Require().NoError(err)
	err = keeper.InsertWorker(ctx, topic, Inferer3, workerInfo)
	s.Require().NoError(err)
	err = keeper.InsertWorker(ctx, topic, Inferer4, workerInfo)
	s.Require().NoError(err)

	// Create a InsertWorkerPayloadRequest message
	workerMsg := types.InsertWorkerPayloadRequest{
		Sender: worker,
		WorkerDataBundle: &types.InputWorkerDataBundle{
			Worker:  worker,
			Nonce:   &nonce,
			TopicId: topic,
			InferenceForecastsBundle: &types.InputInferenceForecastBundle{
				Inference: &types.InputInference{
					TopicId:     topic,
					BlockHeight: nonce.BlockHeight,
					Inferer:     worker,
					Value:       alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
					ExtraData:   nil,
					Proof:       "",
				},
				Forecast: &types.InputForecast{
					TopicId:     topic,
					BlockHeight: nonce.BlockHeight,
					Forecaster:  worker,
					ForecastElements: []*types.InputForecastElement{
						{
							Inferer: worker,
							Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
						},
						{
							Inferer: Inferer2,
							Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(101)),
						},
						{
							Inferer: Inferer3,
							Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(102)),
						},
						{
							Inferer: Inferer4,
							Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(103)),
						},
					},
					ExtraData: nil,
				},
			},
			InferencesForecastsBundleSignature: []byte{},
			Pubkey:                             "",
		},
	}

	return workerMsg, topic
}
func (s *MsgServerTestSuite) signMsgInsertWorkerPayload(workerMsg types.InsertWorkerPayloadRequest, workerPrivateKey secp256k1.PrivKey) types.InsertWorkerPayloadRequest {
	require := s.Require()

	workerPublicKeyBytes := workerPrivateKey.PubKey().Bytes()

	src := make([]byte, 0)
	src, err := workerMsg.WorkerDataBundle.InferenceForecastsBundle.XXX_Marshal(src, true)
	require.NoError(err, "Marshall reputer value bundle should not return an error")

	sig, err := workerPrivateKey.Sign(src)
	require.NoError(err, "Sign should not return an error")
	workerMsg.WorkerDataBundle.InferencesForecastsBundleSignature = sig
	workerMsg.WorkerDataBundle.Pubkey = hex.EncodeToString(workerPublicKeyBytes)

	return workerMsg
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayload() {
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()
	workerMsg, topicId := s.setUpMsgInsertWorkerPayload(workerPrivateKey)
	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)
	blockHeight := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight

	s.WithBlockHeight(blockHeight)

	err := s.EmissionsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), topicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &workerMsg)
	require.NoError(err, "InsertWorkerPayload should not return an error")

	inference, err := s.EmissionsKeeper().GetWorkerLatestInferenceByTopicId(s.Ctx(), topicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)
	require.NotNil(inference)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadNotFailsWithNilInference() {
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()
	workerMsg, topicId := s.setUpMsgInsertWorkerPayload(workerPrivateKey)

	workerMsg.WorkerDataBundle.InferenceForecastsBundle.Inference = nil
	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)

	blockHeight := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight
	s.WithBlockHeight(blockHeight)

	_, err := s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &workerMsg)
	require.ErrorIs(err, types.ErrNotPermittedToSubmitWorkerPayload)

	// Add worker to topic whitelist
	err = s.EmissionsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err, "InsertWorkerPayload should not return an error after adding worker to whitelist")

	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &workerMsg)
	require.NoError(err)

	forecasts, err := s.EmissionsKeeper().GetWorkerLatestForecastByTopicId(s.Ctx(), topicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)
	require.Equal(len(forecasts.ForecastElements), 4)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadNotFailsWithNilForecast() {
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()
	workerMsg, topicId := s.setUpMsgInsertWorkerPayload(workerPrivateKey)

	workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast = nil
	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)

	blockHeight := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Inference.BlockHeight
	s.WithBlockHeight(blockHeight)

	// Add worker to topic whitelist
	err := s.EmissionsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &workerMsg)
	require.NoError(err)

	inferences, err := s.EmissionsKeeper().GetWorkerLatestInferenceByTopicId(s.Ctx(), topicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)
	require.NotNil(inferences)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadFailsWithNilInferenceAndForecast() {
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()
	workerMsg, _ := s.setUpMsgInsertWorkerPayload(workerPrivateKey)
	blockHeight := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight
	s.WithBlockHeight(blockHeight)
	// BEGIN MODIFICATION
	workerMsg.WorkerDataBundle.InferenceForecastsBundle.Inference = nil
	workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast = nil
	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)
	// END MODIFICATION

	// Add worker to topic whitelist
	err := s.EmissionsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &workerMsg)
	require.ErrorIs(err, sdkerrors.ErrInvalidRequest)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadFailsWithoutSignature() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()
	workerMsg, _ := s.setUpMsgInsertWorkerPayload(workerPrivateKey)

	// BEGIN MODIFICATION
	// workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)
	// END MODIFICATION

	// Add worker to topic whitelist
	err := s.EmissionsKeeper().AddToTopicWorkerWhitelist(ctx, workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

	_, err = msgServer.InsertWorkerPayload(ctx, &workerMsg)
	require.ErrorIs(err, sdkerrors.ErrInvalidRequest)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadFailsWithMismatchedTopicId() {
	require := s.Require()

	topicId := uint64(123)
	workerPrivateKey := secp256k1.GenPrivKey()
	workerMsg, _ := s.setUpMsgInsertWorkerPayload(workerPrivateKey)

	// Add worker to topic whitelist
	err := s.EmissionsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), topicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)
	err = s.EmissionsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

	// BEGIN MODIFICATION
	workerMsg.WorkerDataBundle.InferenceForecastsBundle.Inference.TopicId = topicId
	// END MODIFICATION

	blockHeight := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Inference.BlockHeight
	s.WithBlockHeight(blockHeight)
	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)

	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &workerMsg)
	require.ErrorIs(err, sdkerrors.ErrInvalidRequest)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadFailsWithUnregisteredInferer() {
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()
	workerMsg, topicId := s.setUpMsgInsertWorkerPayload(workerPrivateKey)

	// BEGIN MODIFICATION
	inferer := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Inference.Inferer

	unregisterMsg := &types.RemoveRegistrationRequest{
		Sender:    inferer,
		TopicId:   topicId,
		IsReputer: false,
	}

	_, err := s.EmissionsMsgServer().RemoveRegistration(s.Ctx(), unregisterMsg)
	require.NoError(err)
	// END MODIFICATION

	// Add worker to topic whitelist
	err = s.EmissionsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)
	blockHeight := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Inference.BlockHeight
	s.WithBlockHeight(blockHeight)

	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &workerMsg)
	require.ErrorIs(err, types.ErrAddressNotRegistered)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadWithFewTopElementsPerForecast() {
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()
	adminPrivateKey := secp256k1.GenPrivKey()
	adminAddr := sdk.AccAddress(adminPrivateKey.PubKey().Address())
	_ = s.EmissionsKeeper().AddWhitelistAdmin(s.Ctx(), adminAddr.String())

	newParams := &types.OptionalParams{
		MaxElementsPerForecast: []uint64{3},
		// rest not updated
	}

	updateMsg := &types.UpdateParamsRequest{
		Sender: adminAddr.String(),
		Params: newParams,
	}

	_, err := s.EmissionsMsgServer().UpdateParams(s.Ctx(), updateMsg)
	require.NoError(err, "UpdateParams should not return an error")

	blockHeight := int64(1)
	workerBlockHeight := blockHeight + 10800
	workerMsg, topicId := s.setUpMsgInsertWorkerPayloadWithBlockHeight(workerPrivateKey, workerBlockHeight)

	inferer1 := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.ForecastElements[0].Inferer
	inferer2 := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.ForecastElements[1].Inferer
	inferer3 := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.ForecastElements[2].Inferer
	inferer4 := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.ForecastElements[3].Inferer

	score1 := types.Score{TopicId: topicId, BlockHeight: blockHeight, Address: inferer1, Score: alloraMath.NewDecFromInt64(95)}
	score2 := types.Score{TopicId: topicId, BlockHeight: blockHeight, Address: inferer2, Score: alloraMath.NewDecFromInt64(90)}
	score3 := types.Score{TopicId: topicId, BlockHeight: blockHeight, Address: inferer3, Score: alloraMath.NewDecFromInt64(80)}
	score4 := types.Score{TopicId: topicId, BlockHeight: blockHeight, Address: inferer4, Score: alloraMath.NewDecFromInt64(99)}

	_ = s.EmissionsKeeper().SetInfererScoreEma(s.Ctx(), topicId, inferer1, score1)
	_ = s.EmissionsKeeper().SetInfererScoreEma(s.Ctx(), topicId, inferer2, score2)
	_ = s.EmissionsKeeper().SetInfererScoreEma(s.Ctx(), topicId, inferer3, score3)
	_ = s.EmissionsKeeper().SetInfererScoreEma(s.Ctx(), topicId, inferer4, score4)

	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)
	param, _ := s.EmissionsKeeper().GetParams(s.Ctx())
	s.WithBlockHeight(workerBlockHeight)

	// Add worker to topic whitelist
	err = s.EmissionsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &workerMsg)
	require.NoError(err, "InsertWorkerPayload should not return an error")

	forecasts, err := s.EmissionsKeeper().GetWorkerLatestForecastByTopicId(s.Ctx(), topicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

	require.Equal(uint64(len(forecasts.ForecastElements)), param.MaxElementsPerForecast)
	require.Equal(forecasts.ForecastElements[0].Inferer, inferer1)
	require.Equal(forecasts.ForecastElements[1].Inferer, inferer2)
	require.Equal(forecasts.ForecastElements[2].Inferer, inferer4)
}

func (s *MsgServerTestSuite) getCountForecastsAtBlock(topicId uint64, blockHeight int64) int {
	forecastsAtBlock, err := s.EmissionsKeeper().GetForecastsAtBlock(s.Ctx(), topicId, blockHeight)
	if err != nil {
		return 0
	}
	return len(forecastsAtBlock.Forecasts)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadFailsWithMismatchedForecastTopicId() {
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()
	workerMsg, _ := s.setUpMsgInsertWorkerPayload(workerPrivateKey)

	// BEGIN MODIFICATION
	originalTopicId := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.TopicId
	newTopicId := uint64(123)
	workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.TopicId = newTopicId
	// END MODIFICATION

	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)
	blockHeight := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight
	forecastsCount0 := s.getCountForecastsAtBlock(originalTopicId, blockHeight)
	require.Equal(forecastsCount0, 0)

	// Enable topic whitelists
	err := s.EmissionsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), originalTopicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)
	err = s.EmissionsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), newTopicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

	s.WithBlockHeight(blockHeight)
	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &workerMsg)
	require.ErrorIs(err, sdkerrors.ErrInvalidRequest)

	forecastsCount1 := s.getCountForecastsAtBlock(originalTopicId, blockHeight)
	require.Equal(forecastsCount1, 0)

	// Also not added on the changed topicId
	forecastsCountNew := s.getCountForecastsAtBlock(newTopicId, blockHeight)
	require.Equal(forecastsCountNew, 0)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadFailsWithUnregisteredForecaster() {
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()
	workerMsg, topicId := s.setUpMsgInsertWorkerPayload(workerPrivateKey)

	// BEGIN MODIFICATION
	forecaster := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.Forecaster

	unregisterMsg := &types.RemoveRegistrationRequest{
		Sender:    forecaster,
		TopicId:   topicId,
		IsReputer: false,
	}

	_, err := s.EmissionsMsgServer().RemoveRegistration(s.Ctx(), unregisterMsg)
	require.NoError(err)

	// END MODIFICATION

	blockHeight := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight

	// Add worker to topic whitelist
	err = s.EmissionsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

	forecastsCount0 := s.getCountForecastsAtBlock(topicId, blockHeight)
	require.Equal(forecastsCount0, 0)

	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)

	s.WithBlockHeight(blockHeight)
	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &workerMsg)
	require.ErrorIs(err, types.ErrAddressNotRegistered)

	forecastsCount1 := s.getCountForecastsAtBlock(topicId, blockHeight)
	require.Equal(forecastsCount1, 0)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadFiltersDuplicateForecastElements() {
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()
	workerMsg, topicId := s.setUpMsgInsertWorkerPayload(workerPrivateKey)

	// BEGIN MODIFICATION
	forecast := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast
	originalElement := forecast.ForecastElements[0]
	duplicateElement := &types.InputForecastElement{
		Inferer: originalElement.Inferer,
		Value:   originalElement.Value,
	}
	forecast.ForecastElements = append(forecast.ForecastElements, duplicateElement)
	// END MODIFICATION

	// Add worker to topic whitelist
	err := s.EmissionsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)

	blockHeight := forecast.BlockHeight
	s.WithBlockHeight(blockHeight)
	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &workerMsg)
	require.NoError(err, "InsertWorkerPayload should not return an error")

	storedForecasts, err := s.EmissionsKeeper().GetWorkerLatestForecastByTopicId(s.Ctx(), topicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err, "GetForecastsAtBlock should not return an error")
	require.NotZero(len(storedForecasts.ForecastElements), "ForecastElements should not be empty")

	infererMap := make(map[string]bool)
	for _, el := range storedForecasts.ForecastElements {
		_, exists := infererMap[el.Inferer]
		require.False(exists, "Each inferer should appear only once in ForecastElements")
		infererMap[el.Inferer] = true
	}
}

func (s *MsgServerTestSuite) TestInsertingHugeBundleWorkerPayloadFails() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()
	keeper := s.EmissionsKeeper()
	nonce := types.Nonce{BlockHeight: 1}

	// Mock setup for addresses
	reputer := s.AddrsStr()[0]
	reputerAddr := s.Addrs()[0]
	worker := s.AddrsStr()[1]
	workerPrivateKey := s.PrivKeys()[1]
	workerPubKeyBytes := s.PubKeyHexStr()[1]
	workerAddr := s.Addrs()[1]
	InfererAddr := s.AddrsStr()[1]
	ForecasterAddr := s.AddrsStr()[1]

	// Define sample OffchainNode information for a worker
	workerInfo := types.OffchainNode{
		Owner:       worker,
		NodeAddress: worker,
	}

	moduleParams, err := keeper.GetParams(ctx)
	require.NoError(err)

	// Create topic 0 and register reputer in it
	topicId := s.commonStakingSetup(ctx, reputer, reputerAddr, worker, workerAddr, moduleParams.RegistrationFee)
	err = keeper.AddWorkerNonce(ctx, topicId, &nonce)
	require.NoError(err)
	err = keeper.InsertWorker(ctx, topicId, InfererAddr, workerInfo)
	require.NoError(err)
	err = keeper.InsertWorker(ctx, topicId, ForecasterAddr, workerInfo)
	require.NoError(err)

	forecastElements := []*types.InputForecastElement{}
	for i := 0; i < 1000000; i++ {
		forecastElements = append(forecastElements, &types.InputForecastElement{
			Inferer: InfererAddr,
			Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
		})
	}

	// Create a InsertWorkerPayloadRequest message
	workerMsg := &types.InsertWorkerPayloadRequest{
		Sender: worker,
		WorkerDataBundle: &types.InputWorkerDataBundle{
			TopicId: topicId,
			Worker:  InfererAddr,
			Nonce:   &nonce,
			InferenceForecastsBundle: &types.InputInferenceForecastBundle{
				Inference: &types.InputInference{
					TopicId:     topicId,
					BlockHeight: nonce.BlockHeight,
					Inferer:     InfererAddr,
					Value:       alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
					ExtraData:   nil,
					Proof:       "",
				},
				Forecast: &types.InputForecast{
					TopicId:          topicId,
					BlockHeight:      nonce.BlockHeight,
					Forecaster:       ForecasterAddr,
					ForecastElements: forecastElements,
					ExtraData:        nil,
				},
			},
			InferencesForecastsBundleSignature: []byte(""),
			Pubkey:                             "",
		},
	}

	src := make([]byte, 0)
	src, err = workerMsg.WorkerDataBundle.InferenceForecastsBundle.XXX_Marshal(src, true)
	require.NoError(err, "Marshall reputer value bundle should not return an error")

	sig, err := workerPrivateKey.Sign(src)
	require.NoError(err, "Sign should not return an error")
	workerMsg.WorkerDataBundle.InferencesForecastsBundleSignature = sig
	workerMsg.WorkerDataBundle.Pubkey = workerPubKeyBytes
	_, err = msgServer.InsertWorkerPayload(ctx, workerMsg)
	require.ErrorIs(err, types.ErrQueryTooLarge)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadVerifyFailed() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()
	keeper := s.EmissionsKeeper()
	topicId := uint64(1)
	nonce := types.Nonce{BlockHeight: 1}

	// Mock setup for addresses
	reputer := s.AddrsStr()[0]
	reputerAddr := s.Addrs()[0]
	worker := s.AddrsStr()[1]
	workerAddr := s.Addrs()[1]
	Inferer := s.AddrsStr()[2]
	Forecaster := s.AddrsStr()[3]
	Inferer2 := s.AddrsStr()[4]

	// Define sample OffchainNode information for a worker
	workerInfo := types.OffchainNode{
		Owner:       worker,
		NodeAddress: worker,
	}

	moduleParams, err := keeper.GetParams(ctx)
	require.NoError(err)

	// Create topic 0 and register reputer in it
	s.commonStakingSetup(ctx, reputer, reputerAddr, worker, workerAddr, moduleParams.RegistrationFee)
	err = keeper.AddWorkerNonce(ctx, topicId, &nonce)
	require.NoError(err)
	err = keeper.InsertWorker(ctx, topicId, Inferer, workerInfo)
	require.NoError(err)
	err = keeper.InsertWorker(ctx, topicId, Forecaster, workerInfo)
	require.NoError(err)

	// Create a InsertWorkerPayloadRequest message
	workerMsg := &types.InsertWorkerPayloadRequest{
		Sender: worker,
		WorkerDataBundle: &types.InputWorkerDataBundle{
			Worker:  Inferer,
			TopicId: topicId,
			Nonce:   &nonce,
			InferenceForecastsBundle: &types.InputInferenceForecastBundle{
				Inference: &types.InputInference{
					TopicId:     topicId,
					BlockHeight: nonce.BlockHeight,
					Inferer:     Inferer,
					Value:       alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
					ExtraData:   nil,
					Proof:       "",
				},
				Forecast: &types.InputForecast{
					TopicId:     topicId,
					BlockHeight: nonce.BlockHeight,
					Forecaster:  Forecaster,
					ForecastElements: []*types.InputForecastElement{
						{
							Inferer: Inferer,
							Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
						},
						{
							Inferer: Inferer2,
							Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
						},
					},
					ExtraData: nil,
				},
			},
			InferencesForecastsBundleSignature: []byte(""),
			Pubkey:                             "",
		},
	}

	_, err = msgServer.InsertWorkerPayload(ctx, workerMsg)
	require.ErrorIs(err, sdkerrors.ErrInvalidRequest)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadWithLowScoreForecastsAreRejected() {
	require := s.Require()
	keeper := s.EmissionsKeeper()

	workerPrivateKey := secp256k1.GenPrivKey()
	adminPrivateKey := secp256k1.GenPrivKey()
	adminAddr := sdk.AccAddress(adminPrivateKey.PubKey().Address())
	_ = keeper.AddWhitelistAdmin(s.Ctx(), adminAddr.String())

	newParams := &types.OptionalParams{
		MaxElementsPerForecast: []uint64{3},
		// rest not updated
	}

	updateMsg := &types.UpdateParamsRequest{
		Sender: adminAddr.String(),
		Params: newParams,
	}

	_, err := s.EmissionsMsgServer().UpdateParams(s.Ctx(), updateMsg)
	require.NoError(err, "UpdateParams should not return an error")

	blockHeight := int64(1)
	inferenceBlockHeight := blockHeight + 10800
	workerMsg, topicId := s.setUpMsgInsertWorkerPayloadWithBlockHeight(workerPrivateKey, inferenceBlockHeight)

	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)
	s.WithBlockHeight(blockHeight)
	inferer1 := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.ForecastElements[0].Inferer
	inferer2 := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.ForecastElements[1].Inferer
	inferer3 := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.ForecastElements[2].Inferer
	inferer4 := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.ForecastElements[3].Inferer

	score1 := types.Score{TopicId: topicId, BlockHeight: blockHeight, Address: inferer1, Score: alloraMath.NewDecFromInt64(95)}
	score2 := types.Score{TopicId: topicId, BlockHeight: blockHeight, Address: inferer2, Score: alloraMath.NewDecFromInt64(90)}
	score3 := types.Score{TopicId: topicId, BlockHeight: blockHeight, Address: inferer3, Score: alloraMath.NewDecFromInt64(80)}
	score4 := types.Score{TopicId: topicId, BlockHeight: blockHeight, Address: inferer4, Score: alloraMath.NewDecFromInt64(50)}

	_ = keeper.SetInfererScoreEma(s.Ctx(), topicId, inferer1, score1)
	_ = keeper.SetInfererScoreEma(s.Ctx(), topicId, inferer2, score2)
	_ = keeper.SetInfererScoreEma(s.Ctx(), topicId, inferer3, score3)
	_ = keeper.SetInfererScoreEma(s.Ctx(), topicId, inferer4, score4)

	blockHeight = blockHeight + workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight
	s.WithBlockHeight(blockHeight)

	// Add worker to topic whitelist
	err = s.EmissionsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &workerMsg)
	require.NoError(err, "InsertWorkerPayload should not return an error even if the forecast elements are below the threshold")

	forecastsAtBlock, err := keeper.GetWorkerLatestForecastByTopicId(s.Ctx(), topicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)
	require.Equal(len(forecastsAtBlock.ForecastElements), 3)
	require.Equal(forecastsAtBlock.ForecastElements[0].Inferer, inferer1)
	require.Equal(forecastsAtBlock.ForecastElements[1].Inferer, inferer2)
	require.Equal(forecastsAtBlock.ForecastElements[2].Inferer, inferer3)
}

// test that the inferer address inside the bundle matches the signature on the payload message
func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadInfererNotMatchSignature() {
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()
	workerMsg, _ := s.setUpMsgInsertWorkerPayload(workerPrivateKey)
	workerMsg.WorkerDataBundle.InferenceForecastsBundle.Inference.Inferer = s.AddrsStr()[3]

	// Add worker to topic whitelist
	err := s.EmissionsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)
	blockHeight := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight
	s.WithBlockHeight(blockHeight)
	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &workerMsg)
	require.ErrorIs(err, sdkerrors.ErrUnauthorized)
}

// test that the forecaster address inside the bundle matches the signature on the payload message
func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadForecasterNotMatchSignature() {
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()
	workerMsg, _ := s.setUpMsgInsertWorkerPayload(workerPrivateKey)
	workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.Forecaster = s.AddrsStr()[3]

	// Add worker to topic whitelist
	err := s.EmissionsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)
	blockHeight := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight
	s.WithBlockHeight(blockHeight)
	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &workerMsg)
	require.ErrorIs(err, sdkerrors.ErrUnauthorized)
}

// test that the worker field on the bundle matches the signature on the payload message
func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadWorkerNotMatchSignature() {
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()

	workerMsg, _ := s.setUpMsgInsertWorkerPayload(workerPrivateKey)
	workerMsg.WorkerDataBundle.Worker = s.AddrsStr()[3]
	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)
	blockHeight := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight
	s.WithBlockHeight(blockHeight)

	// Add worker to topic whitelist
	err := s.EmissionsKeeper().AddToGlobalWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &workerMsg)
	require.ErrorIs(err, sdkerrors.ErrUnauthorized)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadForecastIncludesSelf() {
	require := s.Require()
	keeper := s.EmissionsKeeper()

	workerPrivateKey := secp256k1.GenPrivKey()
	adminPrivateKey := secp256k1.GenPrivKey()
	adminAddr := sdk.AccAddress(adminPrivateKey.PubKey().Address())
	_ = keeper.AddWhitelistAdmin(s.Ctx(), adminAddr.String())

	// Set up params similar to other tests
	newParams := &types.OptionalParams{ // nolint: exhaustruct
		MaxElementsPerForecast: []uint64{3},
		// not updated params remain nil
	}

	updateMsg := &types.UpdateParamsRequest{
		Sender: adminAddr.String(),
		Params: newParams,
	}

	_, err := s.EmissionsMsgServer().UpdateParams(s.Ctx(), updateMsg)
	require.NoError(err, "UpdateParams should not return an error")

	blockHeight := int64(1)
	inferenceBlockHeight := blockHeight + 10800
	workerMsg, topicId := s.setUpMsgInsertWorkerPayloadWithBlockHeight(workerPrivateKey, inferenceBlockHeight)

	// Modify the forecast to include the worker itself as an inferer
	workerAddr := workerMsg.WorkerDataBundle.Worker
	otherInferer := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.ForecastElements[1].Inferer

	workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.ForecastElements = []*types.InputForecastElement{
		{
			Inferer: workerAddr, // Worker forecasting for self
			Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
		},
		{
			Inferer: otherInferer,
			Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(150)),
		},
	}

	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)
	s.WithBlockHeight(blockHeight)

	// Set up scores for both inferers
	score1 := types.Score{TopicId: topicId, BlockHeight: blockHeight, Address: workerAddr, Score: alloraMath.NewDecFromInt64(95)}
	score2 := types.Score{TopicId: topicId, BlockHeight: blockHeight, Address: otherInferer, Score: alloraMath.NewDecFromInt64(90)}

	_ = keeper.SetInfererScoreEma(s.Ctx(), topicId, workerAddr, score1)
	_ = keeper.SetInfererScoreEma(s.Ctx(), topicId, otherInferer, score2)

	blockHeight = blockHeight + workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight
	s.WithBlockHeight(blockHeight)

	// Add worker to topic whitelist
	err = s.EmissionsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

	// Submit and verify the payload
	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &workerMsg)
	require.NoError(err, "InsertWorkerPayload should succeed when forecaster includes self in forecast")

	// Verify the stored forecast
	forecastsAtBlock, err := keeper.GetWorkerLatestForecastByTopicId(s.Ctx(), topicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)
	require.Equal(2, len(forecastsAtBlock.ForecastElements))
	require.Equal(workerAddr, forecastsAtBlock.ForecastElements[0].Inferer)
	require.Equal(otherInferer, forecastsAtBlock.ForecastElements[1].Inferer)
}
func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadSucceedsWithUnregisteredForecastedInferer() {
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()

	// Set up a worker payload message and get the topic ID
	workerMsg, topicId := s.setUpMsgInsertWorkerPayload(workerPrivateKey)

	// Count the number of forecast elements before we make any changes
	originalElementCount := len(workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.ForecastElements)
	require.Greater(originalElementCount, 1, "Test needs multiple forecast elements")

	// Select one of the inferers from the forecast elements to unregister
	infererToUnregister := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.ForecastElements[1].Inferer

	blockHeight := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight

	// Whitelist the worker for the topic so they can submit forecasts
	err := s.EmissionsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

	// Unregister the inferer
	unregisterMsg := &types.RemoveRegistrationRequest{
		Sender:    infererToUnregister,
		TopicId:   topicId,
		IsReputer: false,
	}
	_, err = s.EmissionsMsgServer().RemoveRegistration(s.Ctx(), unregisterMsg)
	require.NoError(err)

	// Verify that the inferer was successfully unregistered from the topic
	isRegistered, err := s.EmissionsKeeper().IsWorkerRegisteredInTopic(s.Ctx(), topicId, infererToUnregister)
	require.NoError(err)
	require.False(isRegistered, "Inferer should be unregistered")

	// IMPORTANT: Verify that there are no forecasts for this topic/block at the start of the test
	// This establishes a clean baseline to verify our test's effects
	forecastsCount0 := s.getCountForecastsAtBlock(topicId, blockHeight)
	require.Equal(forecastsCount0, 0, "No forecasts should exist before submitting the payload")

	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)

	// Set the block height context so that we're within the worker submission window
	s.WithBlockHeight(blockHeight)

	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &workerMsg)
	require.NoError(err, "InsertWorkerPayload should succeed with an unregistered inferer")
}
