package msgserver_test

import (
	cosmosMath "cosmossdk.io/math"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

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
	workerAddr := sdk.AccAddress(workerPrivateKey.PubKey().Address())
	worker := workerAddr.String()

	reputerIndexes := testutil.ReturnIndexes(0, 1)
	workerIndexes := testutil.ReturnIndexes(1, 3)

	// Define sample OffchainNode information for a worker
	workerInfo := types.OffchainNode{
		Owner:       worker,
		NodeAddress: worker,
	}

	// Create topic 0 and register reputer in it
	s.FullTopicSetup(workerIndexes, reputerIndexes)
	workerInitialBalanceCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(11000)))
	err := s.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.AlloraStakingAccountName, workerAddr, workerInitialBalanceCoins)
	s.Require().NoError(err, "Sending coins should not return an error")

	err = keeper.AddWorkerNonce(ctx, topic, &nonce)
	s.Require().NoError(err)
	err = keeper.InsertWorker(ctx, topic, worker, workerInfo)
	s.Require().NoError(err)

	for _, idx := range workerIndexes {
		err = keeper.InsertWorker(ctx, topic, s.AddrsStr(idx), workerInfo)
		s.Require().NoError(err)
	}

	// Create a InsertWorkerPayloadRequest message
	//nolint:exhaustruct
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
					Value:       alloraMath.MustNewBoundedExp40DecFromString("100"),
					Values: []*types.InputLabeledValue{
						{
							Label: "whatever",
							Value: alloraMath.MustNewBoundedExp40DecFromString("100"),
						},
					},
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
							Inferer: s.AddrsStr(workerIndexes[0]),
							Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(101)),
						},
						{
							Inferer: s.AddrsStr(workerIndexes[1]),
							Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(102)),
						},
						{
							Inferer: s.AddrsStr(workerIndexes[2]),
							Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(103)),
						},
					},
				},
			},
		},
	}

	return workerMsg, topic
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayload() {
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()
	workerMsg, topicId := s.setUpMsgInsertWorkerPayload(workerPrivateKey)
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
	// END MODIFICATION

	// Add worker to topic whitelist
	err := s.EmissionsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &workerMsg)
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

	newParams := &types.OptionalParams{ //nolint:exhaustruct
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

	nonce := types.Nonce{BlockHeight: 1}
	topicId := uint64(1)
	// Mock setup for addresses
	workerIndexes := testutil.ReturnIndexes(1, 1)
	worker := s.AddrsStr(workerIndexes[0])

	forecastElements := []*types.InputForecastElement{}
	for i := 0; i < 1000000; i++ {
		forecastElements = append(forecastElements, &types.InputForecastElement{
			Inferer: worker,
			Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
		})
	}

	// Create a InsertWorkerPayloadRequest message
	//nolint:exhaustruct
	workerMsg := &types.InsertWorkerPayloadRequest{
		Sender: worker,
		WorkerDataBundle: &types.InputWorkerDataBundle{
			TopicId: topicId,
			Worker:  worker,
			Nonce:   &nonce,
			InferenceForecastsBundle: &types.InputInferenceForecastBundle{
				Inference: &types.InputInference{
					TopicId:     topicId,
					BlockHeight: nonce.BlockHeight,
					Inferer:     worker,
					Value:       alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
				},
				Forecast: &types.InputForecast{
					TopicId:          topicId,
					BlockHeight:      nonce.BlockHeight,
					Forecaster:       worker,
					ForecastElements: forecastElements,
				},
			},
		},
	}

	_, err := msgServer.InsertWorkerPayload(ctx, workerMsg)
	require.ErrorIs(err, types.ErrQueryTooLarge)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadVerifyFailed() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()
	topicId := uint64(1)
	nonce := types.Nonce{BlockHeight: 1}

	// Mock setup for addresses
	worker := s.AddrsStr(1)
	inferer := s.AddrsStr(2)

	// Create a InsertWorkerPayloadRequest message
	//nolint:exhaustruct
	workerMsg := &types.InsertWorkerPayloadRequest{
		Sender: worker,
		WorkerDataBundle: &types.InputWorkerDataBundle{
			Worker:  inferer,
			TopicId: topicId,
			Nonce:   &nonce,
			InferenceForecastsBundle: &types.InputInferenceForecastBundle{
				Inference: nil,
				Forecast:  nil,
			},
		},
	}

	_, err := msgServer.InsertWorkerPayload(ctx, workerMsg)
	require.ErrorIs(err, sdkerrors.ErrInvalidRequest)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadWithLowScoreForecastsAreRejected() {
	require := s.Require()
	keeper := s.EmissionsKeeper()

	workerPrivateKey := secp256k1.GenPrivKey()
	adminPrivateKey := secp256k1.GenPrivKey()
	adminAddr := sdk.AccAddress(adminPrivateKey.PubKey().Address())
	_ = keeper.AddWhitelistAdmin(s.Ctx(), adminAddr.String())

	newParams := &types.OptionalParams{ //nolint:exhaustruct
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

	s.WithBlockHeight(blockHeight)

	inferers := make([]string, 4)
	scores := make([]types.Score, 4)
	scoreValues := []int64{95, 90, 80, 50}

	for i := range inferers {
		inferers[i] = workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.ForecastElements[i].Inferer
		scores[i] = types.Score{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Address:     inferers[i],
			Score:       alloraMath.NewDecFromInt64(scoreValues[i]),
		}
		_ = keeper.SetInfererScoreEma(s.Ctx(), topicId, inferers[i], scores[i])
	}

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

	for i, forecastElement := range forecastsAtBlock.ForecastElements {
		require.Equal(forecastElement.Inferer, inferers[i])
	}
}

// test that the inferer address inside the bundle matches the signature on the payload message
func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadInfererNotMatchSignature() {
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()
	workerMsg, _ := s.setUpMsgInsertWorkerPayload(workerPrivateKey)
	workerMsg.WorkerDataBundle.InferenceForecastsBundle.Inference.Inferer = s.AddrsStr(3)

	// Add worker to topic whitelist
	err := s.EmissionsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

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
	workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.Forecaster = s.AddrsStr(3)

	// Add worker to topic whitelist
	err := s.EmissionsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

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
	workerMsg.WorkerDataBundle.Worker = s.AddrsStr(3)
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
	newParams := &types.OptionalParams{ //nolint:exhaustruct
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

	// Set the block height context so that we're within the worker submission window
	s.WithBlockHeight(blockHeight)

	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &workerMsg)
	require.NoError(err, "InsertWorkerPayload should succeed with an unregistered inferer")
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayload_NormalizeInference() {
	type tc struct {
		name          string
		arity         types.TopicOutputArity
		requireUnity  bool
		unityTol      string
		mutate        func(*types.InsertWorkerPayloadRequest)
		wantErr       bool
		wantErrIs     error
		wantValuesStr []string
		wantReg       []string
	}

	cases := []tc{
		{
			name:         "SINGLE_values_len1_overrides_scalar",
			arity:        types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			requireUnity: false,
			unityTol:     "0",
			mutate: func(m *types.InsertWorkerPayloadRequest) {
				m.WorkerDataBundle.InferenceForecastsBundle.Inference.Value = alloraMath.MustNewBoundedExp40DecFromString("999")
				m.WorkerDataBundle.InferenceForecastsBundle.Inference.Values = []*types.InputLabeledValue{
					{Label: "x", Value: alloraMath.MustNewBoundedExp40DecFromString("7")},
				}
			},
			wantValuesStr: []string{"7"},
			wantReg:       nil,
		},
		{
			name:         "SINGLE_values_empty_uses_scalar",
			arity:        types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			requireUnity: false,
			unityTol:     "0",
			mutate: func(m *types.InsertWorkerPayloadRequest) {
				m.WorkerDataBundle.InferenceForecastsBundle.Inference.Value = alloraMath.MustNewBoundedExp40DecFromString("42")
				m.WorkerDataBundle.InferenceForecastsBundle.Inference.Values = nil
			},
			wantValuesStr: []string{"42"},
			wantReg:       nil,
		},
		{
			name:         "SINGLE_values_len2_rejected",
			arity:        types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			requireUnity: false,
			unityTol:     "0",
			mutate: func(m *types.InsertWorkerPayloadRequest) {
				m.WorkerDataBundle.InferenceForecastsBundle.Inference.Values = []*types.InputLabeledValue{
					{Label: "x", Value: alloraMath.MustNewBoundedExp40DecFromString("1")},
					{Label: "y", Value: alloraMath.MustNewBoundedExp40DecFromString("2")},
				}
			},
			wantErr:   true,
			wantErrIs: sdkerrors.ErrInvalidRequest,
		},
		{
			name:         "MULTI_registers_labels_and_aligns",
			arity:        types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			requireUnity: false,
			unityTol:     "0",
			mutate: func(m *types.InsertWorkerPayloadRequest) {
				m.WorkerDataBundle.InferenceForecastsBundle.Inference.Values = []*types.InputLabeledValue{
					{Label: "A", Value: alloraMath.MustNewBoundedExp40DecFromString("0.2")},
					{Label: "B", Value: alloraMath.MustNewBoundedExp40DecFromString("0.8")},
				}
			},
			wantValuesStr: []string{"0.2", "0.8"},
			wantReg:       []string{"A", "B"},
		},
		{
			name:         "MULTI_require_unity_ok",
			arity:        types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			requireUnity: true,
			unityTol:     "0.000001",
			mutate: func(m *types.InsertWorkerPayloadRequest) {
				m.WorkerDataBundle.InferenceForecastsBundle.Inference.Values = []*types.InputLabeledValue{
					{Label: "A", Value: alloraMath.MustNewBoundedExp40DecFromString("0.2")},
					{Label: "B", Value: alloraMath.MustNewBoundedExp40DecFromString("0.8")},
				}
			},
			wantValuesStr: []string{"0.2", "0.8"},
			wantReg:       []string{"A", "B"},
		},
		{
			name:         "MULTI_require_unity_rejected",
			arity:        types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			requireUnity: true,
			unityTol:     "0.01",
			mutate: func(m *types.InsertWorkerPayloadRequest) {
				m.WorkerDataBundle.InferenceForecastsBundle.Inference.Values = []*types.InputLabeledValue{
					{Label: "A", Value: alloraMath.MustNewBoundedExp40DecFromString("0.2")},
					{Label: "B", Value: alloraMath.MustNewBoundedExp40DecFromString("0.7")},
				}
			},
			wantErr:   true,
			wantErrIs: sdkerrors.ErrInvalidRequest,
		},
		{
			name:         "MULTI_duplicate_label_rejected",
			arity:        types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			requireUnity: false,
			unityTol:     "0",
			mutate: func(m *types.InsertWorkerPayloadRequest) {
				m.WorkerDataBundle.InferenceForecastsBundle.Inference.Values = []*types.InputLabeledValue{
					{Label: "A", Value: alloraMath.MustNewBoundedExp40DecFromString("0.1")},
					{Label: "A", Value: alloraMath.MustNewBoundedExp40DecFromString("0.2")},
				}
			},
			wantErr:   true,
			wantErrIs: sdkerrors.ErrInvalidRequest,
		},
		{
			name:         "MULTI_empty_label_rejected",
			arity:        types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			requireUnity: false,
			unityTol:     "0",
			mutate: func(m *types.InsertWorkerPayloadRequest) {
				m.WorkerDataBundle.InferenceForecastsBundle.Inference.Values = []*types.InputLabeledValue{
					{Label: "   ", Value: alloraMath.MustNewBoundedExp40DecFromString("0.1")},
				}
			},
			wantErr:   true,
			wantErrIs: sdkerrors.ErrInvalidRequest,
		},
	}

	for _, c := range cases {
		s.Run(c.name, func() {
			s.SetupTest()

			workerPrivateKey := secp256k1.GenPrivKey()
			nonce := int64(1)

			msg, topicId := s.setUpMsgInsertWorkerPayload(workerPrivateKey)
			if c.mutate != nil {
				c.mutate(&msg)
			}

			s.WithBlockHeight(nonce)
			s.setTopicArityAndUnity(topicId, c.arity, c.requireUnity, c.unityTol)

			err := s.EmissionsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), topicId, msg.WorkerDataBundle.Worker)
			s.Require().NoError(err)

			_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &msg)
			if c.wantErr {
				if c.wantErrIs != nil {
					s.Require().ErrorIs(err, c.wantErrIs)
				} else {
					s.Require().Error(err)
				}
				return
			}
			s.Require().NoError(err)

			got, err := s.EmissionsKeeper().GetWorkerLatestInferenceByTopicId(s.Ctx(), topicId, msg.WorkerDataBundle.Worker)
			s.Require().NoError(err)
			s.Require().NotNil(got)
			s.Require().Equal(len(c.wantValuesStr), len(got.Values))
			for i := range c.wantValuesStr {
				s.Require().Equal(c.wantValuesStr[i], got.Values[i].String())
			}

			reg, err := s.EmissionsKeeper().GetEpochLabelRegistry(s.Ctx(), topicId, nonce)
			s.Require().NoError(err)

			if c.arity == types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE {
				s.Require().Equal(1, len(reg.Labels))
				return
			}

			s.Require().Equal(len(c.wantReg), len(reg.Labels))
			for i := range c.wantReg {
				s.Require().Equal(c.wantReg[i], reg.Labels[i].Name)
			}
		})
	}
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayload_Multi_TwoWorkersSameNonce_UnionAndAlignment() {
	s.SetupTest()

	pk1 := secp256k1.GenPrivKey()
	pk2 := secp256k1.GenPrivKey()

	nonce := int64(1)

	msg1, topicId := s.setUpMsgInsertWorkerPayload(pk1)
	s.WithBlockHeight(nonce)
	s.setTopicArityAndUnity(topicId, types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI, false, "0")

	msg1.WorkerDataBundle.Nonce.BlockHeight = nonce
	msg1.WorkerDataBundle.InferenceForecastsBundle.Inference.BlockHeight = nonce
	msg1.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight = nonce
	msg1.WorkerDataBundle.InferenceForecastsBundle.Inference.Values = []*types.InputLabeledValue{
		{Label: "a", Value: alloraMath.MustNewBoundedExp40DecFromString("1")},
		{Label: "b", Value: alloraMath.MustNewBoundedExp40DecFromString("2")},
		{Label: "c", Value: alloraMath.MustNewBoundedExp40DecFromString("3")},
	}

	msg2, _ := s.setUpMsgInsertWorkerPayload(pk2)
	msg2.WorkerDataBundle.TopicId = topicId
	msg2.WorkerDataBundle.Nonce.BlockHeight = nonce
	msg2.WorkerDataBundle.InferenceForecastsBundle.Inference.TopicId = topicId
	msg2.WorkerDataBundle.InferenceForecastsBundle.Forecast.TopicId = topicId
	msg2.WorkerDataBundle.InferenceForecastsBundle.Inference.BlockHeight = nonce
	msg2.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight = nonce
	msg2.WorkerDataBundle.InferenceForecastsBundle.Inference.Values = []*types.InputLabeledValue{
		{Label: "a", Value: alloraMath.MustNewBoundedExp40DecFromString("10")},
		{Label: "b", Value: alloraMath.MustNewBoundedExp40DecFromString("20")},
		{Label: "d", Value: alloraMath.MustNewBoundedExp40DecFromString("40")},
	}

	err := s.EmissionsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), topicId, msg1.WorkerDataBundle.Worker)
	s.Require().NoError(err)
	err = s.EmissionsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), topicId, msg2.WorkerDataBundle.Worker)
	s.Require().NoError(err)

	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &msg1)
	s.Require().NoError(err)

	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &msg2)
	s.Require().NoError(err)

	reg, err := s.EmissionsKeeper().GetEpochLabelRegistry(s.Ctx(), topicId, nonce)
	s.Require().NoError(err)
	s.Require().Equal(4, len(reg.Labels))
	s.Require().Equal("a", reg.Labels[0].Name)
	s.Require().Equal("b", reg.Labels[1].Name)
	s.Require().Equal("c", reg.Labels[2].Name)
	s.Require().Equal("d", reg.Labels[3].Name)

	got1, err := s.EmissionsKeeper().GetWorkerLatestInferenceByTopicId(s.Ctx(), topicId, msg1.WorkerDataBundle.Worker)
	s.Require().NoError(err)
	s.Require().Equal(3, len(got1.Values))
	s.Require().Equal("1", got1.Values[0].String())
	s.Require().Equal("2", got1.Values[1].String())
	s.Require().Equal("3", got1.Values[2].String())

	got2, err := s.EmissionsKeeper().GetWorkerLatestInferenceByTopicId(s.Ctx(), topicId, msg2.WorkerDataBundle.Worker)
	s.Require().NoError(err)
	s.Require().Equal(4, len(got2.Values))
	s.Require().Equal("10", got2.Values[0].String())
	s.Require().Equal("20", got2.Values[1].String())
	s.Require().Equal("0", got2.Values[2].String())
	s.Require().Equal("40", got2.Values[3].String())
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayload_Multi_NoCrossEpochRegistryLeakage() {
	s.SetupTest()

	pk1 := secp256k1.GenPrivKey()
	pk2 := secp256k1.GenPrivKey()

	nonce1 := int64(1)
	nonce2 := int64(200)

	// worker1 @ nonce1
	msg1, topicId := s.setUpMsgInsertWorkerPayload(pk1)
	s.WithBlockHeight(nonce1)
	s.setTopicArityAndUnity(topicId, types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI, false, "0")

	msg1.WorkerDataBundle.Nonce.BlockHeight = nonce1
	msg1.WorkerDataBundle.InferenceForecastsBundle.Inference.BlockHeight = nonce1
	msg1.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight = nonce1
	msg1.WorkerDataBundle.InferenceForecastsBundle.Inference.Values = []*types.InputLabeledValue{
		{Label: "a", Value: alloraMath.MustNewBoundedExp40DecFromString("1")},
		{Label: "b", Value: alloraMath.MustNewBoundedExp40DecFromString("2")},
	}

	// register nonce2 for the same topic (epoch 2)
	err := s.EmissionsKeeper().AddWorkerNonce(s.Ctx(), topicId, &types.Nonce{BlockHeight: nonce2})
	s.Require().NoError(err)

	// worker2 @ nonce2 (must be a different worker because activeInferers is (topic,inferer) only)
	msg2, _ := s.setUpMsgInsertWorkerPayload(pk2)
	msg2.WorkerDataBundle.TopicId = topicId
	msg2.WorkerDataBundle.InferenceForecastsBundle.Inference.TopicId = topicId
	msg2.WorkerDataBundle.InferenceForecastsBundle.Forecast.TopicId = topicId

	msg2.WorkerDataBundle.Nonce.BlockHeight = nonce2
	msg2.WorkerDataBundle.InferenceForecastsBundle.Inference.BlockHeight = nonce2
	msg2.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight = nonce2
	msg2.WorkerDataBundle.InferenceForecastsBundle.Inference.Values = []*types.InputLabeledValue{
		{Label: "a", Value: alloraMath.MustNewBoundedExp40DecFromString("10")},
		{Label: "b", Value: alloraMath.MustNewBoundedExp40DecFromString("20")},
		{Label: "c", Value: alloraMath.MustNewBoundedExp40DecFromString("30")},
	}

	// whitelist both workers
	err = s.EmissionsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), topicId, msg1.WorkerDataBundle.Worker)
	s.Require().NoError(err)
	err = s.EmissionsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), topicId, msg2.WorkerDataBundle.Worker)
	s.Require().NoError(err)

	// submit worker1 @ nonce1
	s.WithBlockHeight(nonce1)
	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &msg1)
	s.Require().NoError(err)

	reg1, err := s.EmissionsKeeper().GetEpochLabelRegistry(s.Ctx(), topicId, nonce1)
	s.Require().NoError(err)
	s.Require().Equal(2, len(reg1.Labels))
	s.Require().Equal("a", reg1.Labels[0].Name)
	s.Require().Equal("b", reg1.Labels[1].Name)

	got1, err := s.EmissionsKeeper().GetWorkerLatestInferenceByTopicId(s.Ctx(), topicId, msg1.WorkerDataBundle.Worker)
	s.Require().NoError(err)
	s.Require().NotNil(got1)
	s.Require().Equal(2, len(got1.Values))
	s.Require().Equal("1", got1.Values[0].String())
	s.Require().Equal("2", got1.Values[1].String())

	// submit worker2 @ nonce2
	s.WithBlockHeight(nonce2)
	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &msg2)
	s.Require().NoError(err)

	reg2, err := s.EmissionsKeeper().GetEpochLabelRegistry(s.Ctx(), topicId, nonce2)
	s.Require().NoError(err)
	s.Require().Equal(3, len(reg2.Labels))
	s.Require().Equal("a", reg2.Labels[0].Name)
	s.Require().Equal("b", reg2.Labels[1].Name)
	s.Require().Equal("c", reg2.Labels[2].Name)

	got2, err := s.EmissionsKeeper().GetWorkerLatestInferenceByTopicId(s.Ctx(), topicId, msg2.WorkerDataBundle.Worker)
	s.Require().NoError(err)
	s.Require().NotNil(got2)
	s.Require().Equal(3, len(got2.Values))
	s.Require().Equal("10", got2.Values[0].String())
	s.Require().Equal("20", got2.Values[1].String())
	s.Require().Equal("30", got2.Values[2].String())

	// ensure epoch1 registry did not change after epoch2 submission
	reg1Again, err := s.EmissionsKeeper().GetEpochLabelRegistry(s.Ctx(), topicId, nonce1)
	s.Require().NoError(err)
	s.Require().Equal(2, len(reg1Again.Labels))
	s.Require().Equal("a", reg1Again.Labels[0].Name)
	s.Require().Equal("b", reg1Again.Labels[1].Name)
}

func (s *MsgServerTestSuite) setTopicArityAndUnity(
	topicId uint64,
	outputArity types.TopicOutputArity,
	requireUnity bool,
	unityTol string,
) {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	topic, err := k.GetTopic(ctx, topicId)
	s.Require().NoError(err)

	topic.OutputArity = outputArity
	topic.RequireUnity = requireUnity
	topic.UnityTolerance = alloraMath.MustNewDecFromString(unityTol)

	err = k.SetTopic(ctx, topicId, topic)
	s.Require().NoError(err)
}
