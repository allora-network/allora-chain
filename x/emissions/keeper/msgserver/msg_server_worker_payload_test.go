package msgserver_test

import (
	"encoding/hex"
	"errors"

	"cosmossdk.io/collections"
	cosmosMath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
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

	err = s.NonceKeeper().AddWorkerNonce(ctx, topic, &nonce)
	s.Require().NoError(err)
	err = s.WorkerKeeper().InsertWorker(ctx, topic, worker, workerInfo)
	s.Require().NoError(err)

	for _, idx := range workerIndexes {
		err = s.WorkerKeeper().InsertWorker(ctx, topic, s.AddrsStr(idx), workerInfo)
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
					// Default test fixture targets a SINGLE-arity topic, so
					// we leave Values nil and let NormalizeInputInference
					// fall through to the scalar path. Tests that want a
					// MULTI submission mutate Values in-place after building
					// the fixture.
					Values: nil,
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
			InferencesForecastsBundleSignature: []byte{},
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

	err := s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), topicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &workerMsg)
	require.NoError(err, "InsertWorkerPayload should not return an error")

	inference, err := s.WorkerKeeper().GetWorkerLatestInferenceByTopicId(s.Ctx(), topicId, workerMsg.WorkerDataBundle.Worker)
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
	err = s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err, "InsertWorkerPayload should not return an error after adding worker to whitelist")

	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &workerMsg)
	require.NoError(err)

	forecasts, err := s.WorkerKeeper().GetWorkerLatestForecastByTopicId(s.Ctx(), topicId, workerMsg.WorkerDataBundle.Worker)
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
	err := s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &workerMsg)
	require.NoError(err)

	inferences, err := s.WorkerKeeper().GetWorkerLatestInferenceByTopicId(s.Ctx(), topicId, workerMsg.WorkerDataBundle.Worker)
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
	err := s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
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
	err := s.WhitelistsKeeper().AddToTopicWorkerWhitelist(ctx, workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
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
	err := s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), topicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)
	err = s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
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
	err = s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
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
	_ = s.WhitelistsKeeper().AddWhitelistAdmin(s.Ctx(), adminAddr.String())

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

	_ = s.ScoresKeeper().SetInfererScoreEma(s.Ctx(), topicId, inferer1, score1)
	_ = s.ScoresKeeper().SetInfererScoreEma(s.Ctx(), topicId, inferer2, score2)
	_ = s.ScoresKeeper().SetInfererScoreEma(s.Ctx(), topicId, inferer3, score3)
	_ = s.ScoresKeeper().SetInfererScoreEma(s.Ctx(), topicId, inferer4, score4)

	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)
	param, _ := s.ParamsKeeper().GetParams(s.Ctx())
	s.WithBlockHeight(workerBlockHeight)

	// Add worker to topic whitelist
	err = s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &workerMsg)
	require.NoError(err, "InsertWorkerPayload should not return an error")

	forecasts, err := s.WorkerKeeper().GetWorkerLatestForecastByTopicId(s.Ctx(), topicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

	require.Equal(uint64(len(forecasts.ForecastElements)), param.MaxElementsPerForecast)
	require.Equal(forecasts.ForecastElements[0].Inferer, inferer1)
	require.Equal(forecasts.ForecastElements[1].Inferer, inferer2)
	require.Equal(forecasts.ForecastElements[2].Inferer, inferer4)
}

// forecasterPayloadEvents returns all EventInsertForecasterPayload typed events
// currently on the SDK context's event manager. It reconstructs each event via
// sdk.ParseTypedEvent so the assertion stays agnostic to the proto package
// version (e.g. v10 -> v11 bumps require no test changes).
func forecasterPayloadEvents(ctx sdk.Context) []*types.EventInsertForecasterPayload {
	out := make([]*types.EventInsertForecasterPayload, 0)
	for _, ev := range ctx.EventManager().Events() {
		msg, err := sdk.ParseTypedEvent(abci.Event(ev))
		if err != nil {
			continue
		}
		if e, ok := msg.(*types.EventInsertForecasterPayload); ok {
			out = append(out, e)
		}
	}
	return out
}

// TestMsgInsertWorkerPayloadNoForecasterEventWhenAllFiltered pins the contract
// that a forecaster payload event is emitted only when forecast elements were
// actually stored. With MaxElementsPerForecast == 0, FindTopNByScoreDesc selects
// no inferers, every element is filtered out, AppendForecast is skipped, and no
// event must be emitted - mirroring the admission-gated inference path.
func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadNoForecasterEventWhenAllFiltered() {
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()
	adminPrivateKey := secp256k1.GenPrivKey()
	adminAddr := sdk.AccAddress(adminPrivateKey.PubKey().Address())
	_ = s.WhitelistsKeeper().AddWhitelistAdmin(s.Ctx(), adminAddr.String())

	newParams := &types.OptionalParams{ //nolint:exhaustruct
		MaxElementsPerForecast: []uint64{0},
		// rest not updated
	}
	_, err := s.EmissionsMsgServer().UpdateParams(s.Ctx(), &types.UpdateParamsRequest{
		Sender: adminAddr.String(),
		Params: newParams,
	})
	require.NoError(err, "UpdateParams should not return an error")

	blockHeight := int64(1)
	workerBlockHeight := blockHeight + 10800
	workerMsg, topicId := s.setUpMsgInsertWorkerPayloadWithBlockHeight(workerPrivateKey, workerBlockHeight)
	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)
	s.WithBlockHeight(workerBlockHeight)

	err = s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), topicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &workerMsg)
	require.NoError(err, "InsertWorkerPayload should not return an error")

	// Nothing was stored...
	require.Equal(0, s.getCountForecastsAtBlock(topicId, workerBlockHeight),
		"no forecast should be stored when all elements are filtered out")
	// ...so no forecaster payload event should have been emitted.
	require.Empty(forecasterPayloadEvents(s.Ctx()),
		"no forecaster payload event should be emitted when nothing is stored")
}

// TestMsgInsertWorkerPayloadEmitsForecasterEventReflectingStoredElements guards
// against over-correcting the gating fix: when forecast elements are accepted,
// exactly one event must be emitted and it must advertise the actually-stored
// (filtered) elements rather than the original unfiltered submission.
func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadEmitsForecasterEventReflectingStoredElements() {
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()
	workerMsg, topicId := s.setUpMsgInsertWorkerPayload(workerPrivateKey)
	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)
	blockHeight := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight
	s.WithBlockHeight(blockHeight)

	err := s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), topicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &workerMsg)
	require.NoError(err, "InsertWorkerPayload should not return an error")

	stored, err := s.WorkerKeeper().GetWorkerLatestForecastByTopicId(s.Ctx(), topicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)
	require.NotEmpty(stored.ForecastElements)

	events := forecasterPayloadEvents(s.Ctx())
	require.Len(events, 1, "exactly one forecaster payload event should be emitted on a successful store")

	storedInferers := make([]string, 0, len(stored.ForecastElements))
	for _, el := range stored.ForecastElements {
		storedInferers = append(storedInferers, el.Inferer)
	}
	require.ElementsMatch(storedInferers, events[0].InfererAddresses,
		"event must advertise exactly the stored (filtered) forecast elements")
}

func (s *MsgServerTestSuite) getCountForecastsAtBlock(topicId uint64, blockHeight int64) int {
	forecastsAtBlock, err := s.WorkerKeeper().GetForecastsAtBlock(s.Ctx(), topicId, blockHeight)
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
	err := s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), originalTopicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)
	err = s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), newTopicId, workerMsg.WorkerDataBundle.Worker)
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
	err = s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
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
	err := s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)

	blockHeight := forecast.BlockHeight
	s.WithBlockHeight(blockHeight)
	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &workerMsg)
	require.NoError(err, "InsertWorkerPayload should not return an error")

	storedForecasts, err := s.WorkerKeeper().GetWorkerLatestForecastByTopicId(s.Ctx(), topicId, workerMsg.WorkerDataBundle.Worker)
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
	workerPrivateKey := s.PrivKeys(workerIndexes[0])
	workerPubKeyBytes := s.PubKeyHexStr(workerIndexes[0])

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
			InferencesForecastsBundleSignature: []byte{},
		},
	}

	src := make([]byte, 0)
	src, err := workerMsg.WorkerDataBundle.InferenceForecastsBundle.XXX_Marshal(src, true)
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
	topicId := uint64(1)
	nonce := types.Nonce{BlockHeight: 1}

	// Mock setup for addresses
	worker := s.AddrsStr(1)
	inferer := s.AddrsStr(2)
	forecaster := s.AddrsStr(3)
	inferer2 := s.AddrsStr(4)

	// Create a InsertWorkerPayloadRequest message
	//nolint:exhaustruct
	workerMsg := &types.InsertWorkerPayloadRequest{
		Sender: worker,
		WorkerDataBundle: &types.InputWorkerDataBundle{
			Worker:  inferer,
			TopicId: topicId,
			Nonce:   &nonce,
			InferenceForecastsBundle: &types.InputInferenceForecastBundle{
				Inference: &types.InputInference{
					TopicId:     topicId,
					BlockHeight: nonce.BlockHeight,
					Inferer:     inferer,
					Value:       alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
				},
				Forecast: &types.InputForecast{
					TopicId:     topicId,
					BlockHeight: nonce.BlockHeight,
					Forecaster:  forecaster,
					ForecastElements: []*types.InputForecastElement{
						{
							Inferer: inferer,
							Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
						},
						{
							Inferer: inferer2,
							Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
						},
					},
				},
			},
			InferencesForecastsBundleSignature: []byte{},
		},
	}

	_, err := msgServer.InsertWorkerPayload(ctx, workerMsg)
	require.ErrorIs(err, sdkerrors.ErrInvalidRequest)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadWithLowScoreForecastsAreRejected() {
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()
	adminPrivateKey := secp256k1.GenPrivKey()
	adminAddr := sdk.AccAddress(adminPrivateKey.PubKey().Address())
	_ = s.WhitelistsKeeper().AddWhitelistAdmin(s.Ctx(), adminAddr.String())

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

	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)
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
		_ = s.ScoresKeeper().SetInfererScoreEma(s.Ctx(), topicId, inferers[i], scores[i])
	}

	blockHeight = blockHeight + workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight
	s.WithBlockHeight(blockHeight)

	// Add worker to topic whitelist
	err = s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &workerMsg)
	require.NoError(err, "InsertWorkerPayload should not return an error even if the forecast elements are below the threshold")

	forecastsAtBlock, err := s.WorkerKeeper().GetWorkerLatestForecastByTopicId(s.Ctx(), topicId, workerMsg.WorkerDataBundle.Worker)
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
	err := s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
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
	workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.Forecaster = s.AddrsStr(3)

	// Add worker to topic whitelist
	err := s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
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
	workerMsg.WorkerDataBundle.Worker = s.AddrsStr(3)
	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)
	blockHeight := workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight
	s.WithBlockHeight(blockHeight)

	// Add worker to topic whitelist
	err := s.WhitelistsKeeper().AddToGlobalWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &workerMsg)
	require.ErrorIs(err, sdkerrors.ErrUnauthorized)
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayloadForecastIncludesSelf() {
	require := s.Require()

	workerPrivateKey := secp256k1.GenPrivKey()
	adminPrivateKey := secp256k1.GenPrivKey()
	adminAddr := sdk.AccAddress(adminPrivateKey.PubKey().Address())
	_ = s.WhitelistsKeeper().AddWhitelistAdmin(s.Ctx(), adminAddr.String())

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

	workerMsg = s.signMsgInsertWorkerPayload(workerMsg, workerPrivateKey)
	s.WithBlockHeight(blockHeight)

	// Set up scores for both inferers
	score1 := types.Score{TopicId: topicId, BlockHeight: blockHeight, Address: workerAddr, Score: alloraMath.NewDecFromInt64(95)}
	score2 := types.Score{TopicId: topicId, BlockHeight: blockHeight, Address: otherInferer, Score: alloraMath.NewDecFromInt64(90)}

	_ = s.ScoresKeeper().SetInfererScoreEma(s.Ctx(), topicId, workerAddr, score1)
	_ = s.ScoresKeeper().SetInfererScoreEma(s.Ctx(), topicId, otherInferer, score2)

	blockHeight = blockHeight + workerMsg.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight
	s.WithBlockHeight(blockHeight)

	// Add worker to topic whitelist
	err = s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
	require.NoError(err)

	// Submit and verify the payload
	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &workerMsg)
	require.NoError(err, "InsertWorkerPayload should succeed when forecaster includes self in forecast")

	// Verify the stored forecast
	forecastsAtBlock, err := s.WorkerKeeper().GetWorkerLatestForecastByTopicId(s.Ctx(), topicId, workerMsg.WorkerDataBundle.Worker)
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
	err := s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), workerMsg.WorkerDataBundle.TopicId, workerMsg.WorkerDataBundle.Worker)
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
	isRegistered, err := s.WorkerKeeper().IsWorkerRegisteredInTopic(s.Ctx(), topicId, infererToUnregister)
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

//nolint:exhaustruct
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
			// SINGLE topics now require the canonical label "y" (or an
			// entirely-missing Values slice, falling through to the scalar
			// path). See NormalizeInputInference + the plan's "SINGLE/MULTI
			// arity" section.
			name:         "SINGLE_values_len1_overrides_scalar",
			arity:        types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			requireUnity: false,
			unityTol:     "0",
			mutate: func(m *types.InsertWorkerPayloadRequest) {
				m.WorkerDataBundle.InferenceForecastsBundle.Inference.Value = alloraMath.MustNewBoundedExp40DecFromString("999")
				m.WorkerDataBundle.InferenceForecastsBundle.Inference.Values = []*types.InputLabeledValue{
					{Label: "y", Value: alloraMath.MustNewBoundedExp40DecFromString("7")},
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
			wantErrIs: types.ErrTooManyLabelsPerSubmission,
		},
		{
			// An empty label in Values is rejected at the InsertWorkerPayload
			// boundary by ValidateWithLimits -> CanonicalLabelName ("empty
			// after trimming"), before NormalizeInputInference's permissive
			// empty-label scalar branch can apply. Distinct from the scalar
			// Value path (SINGLE_values_empty_uses_scalar), which is accepted.
			name:         "SINGLE_empty_label_rejected",
			arity:        types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			requireUnity: false,
			unityTol:     "0",
			mutate: func(m *types.InsertWorkerPayloadRequest) {
				m.WorkerDataBundle.InferenceForecastsBundle.Inference.Values = []*types.InputLabeledValue{
					{Label: "", Value: alloraMath.MustNewBoundedExp40DecFromString("1")},
				}
			},
			wantErr:   true,
			wantErrIs: types.ErrInvalidLabelName,
		},
		{
			// A single non-"y" canonical label passes ValidateWithLimits (no
			// whitelist, valid charset) but is rejected by the single-arity
			// guard in validateWorkerInferenceLabels: SINGLE topics only accept
			// the canonical label "y" (or the scalar path).
			name:         "SINGLE_non_y_label_rejected",
			arity:        types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			requireUnity: false,
			unityTol:     "0",
			mutate: func(m *types.InsertWorkerPayloadRequest) {
				m.WorkerDataBundle.InferenceForecastsBundle.Inference.Values = []*types.InputLabeledValue{
					{Label: "x", Value: alloraMath.MustNewBoundedExp40DecFromString("1")},
				}
			},
			wantErr:   true,
			wantErrIs: sdkerrors.ErrInvalidRequest,
		},
		{
			// Under the default caseSensitive=false topic, submitted
			// uppercase labels "A"/"B" lowercase to "a"/"b" and are
			// refcounted under the canonical form.
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
			wantReg:       []string{"a", "b"},
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
			wantReg:       []string{"a", "b"},
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
			// Duplicate labels are rejected by
			// InputInference.ValidateWithLimits with ErrInvalidLabelName
			// (wrapped by msgserver). Under caseSensitive=false, "A" and
			// "a" also collapse to the same canonical form and dedupe.
			name:         "MULTI_duplicate_label_rejected",
			arity:        types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			requireUnity: false,
			unityTol:     "0",
			mutate: func(m *types.InsertWorkerPayloadRequest) {
				m.WorkerDataBundle.InferenceForecastsBundle.Inference.Values = []*types.InputLabeledValue{
					{Label: "A", Value: alloraMath.MustNewBoundedExp40DecFromString("0.1")},
					{Label: "a", Value: alloraMath.MustNewBoundedExp40DecFromString("0.2")},
				}
			},
			wantErr:   true,
			wantErrIs: types.ErrInvalidLabelName,
		},
		{
			// Empty/whitespace-only labels are rejected by
			// CanonicalLabelName with ErrInvalidLabelName.
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
			wantErrIs: types.ErrInvalidLabelName,
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
			msg = s.signMsgInsertWorkerPayload(msg, workerPrivateKey)

			s.WithBlockHeight(nonce)
			s.setTopicArityAndUnity(topicId, c.arity, c.requireUnity, c.unityTol)

			err := s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), topicId, msg.WorkerDataBundle.Worker)
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

			// The normalized temporary inference is staged only when the worker
			// is admitted. MULTI values are aligned to temporary first-seen ids.
			got, err := s.WorkerKeeper().GetWorkerLatestInferenceByTopicId(s.Ctx(), topicId, msg.WorkerDataBundle.Worker)
			s.Require().NoError(err)
			s.Require().Equal(len(c.wantValuesStr), len(got.Values))
			for i := range c.wantValuesStr {
				s.Require().Equal(c.wantValuesStr[i], got.Values[i].String())
			}

			if c.arity == types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE {
				return
			}

			reg, err := s.TopicKeeper().GetEpochLabelRegistry(s.Ctx(), topicId, nonce)
			s.Require().NoError(err)
			s.Require().Len(reg.Labels, len(c.wantReg))
			for i, label := range c.wantReg {
				s.Require().Equal(label, reg.Labels[i].Name)
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
	msg1 = s.signMsgInsertWorkerPayload(msg1, pk1)

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

	msg2 = s.signMsgInsertWorkerPayload(msg2, pk2)

	err := s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), topicId, msg1.WorkerDataBundle.Worker)
	s.Require().NoError(err)
	err = s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), topicId, msg2.WorkerDataBundle.Worker)
	s.Require().NoError(err)

	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &msg1)
	s.Require().NoError(err)

	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &msg2)
	s.Require().NoError(err)

	// Finalize the final registry as CloseWorkerNonce would, and
	// assert that the projected Values slices are (a) aligned to the
	// frozen registry and (b) carry zeros for labels the worker did not
	// submit.
	topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)
	tempReg, err := s.TopicKeeper().GetEpochLabelRegistry(s.Ctx(), topicId, nonce)
	s.Require().NoError(err)
	s.Require().Equal(4, len(tempReg.Labels))
	s.Require().Equal("a", tempReg.Labels[0].Name)
	s.Require().Equal("b", tempReg.Labels[1].Name)
	s.Require().Equal("c", tempReg.Labels[2].Name)
	s.Require().Equal("d", tempReg.Labels[3].Name)

	finalized, reg, err := s.WorkerKeeper().FinalizeInferencesAndRegistryAtClose(
		s.Ctx(), topic, nonce,
		[]string{msg1.WorkerDataBundle.Worker, msg2.WorkerDataBundle.Worker},
	)
	s.Require().NoError(err)
	s.Require().Equal(tempReg, reg)
	byWorker := map[string]*types.Inference{}
	for _, inf := range finalized.Inferences {
		byWorker[inf.Inferer] = inf
	}
	got1 := byWorker[msg1.WorkerDataBundle.Worker]
	s.Require().NotNil(got1)
	s.Require().Equal(4, len(got1.Values))
	s.Require().Equal("1", got1.Values[0].String())
	s.Require().Equal("2", got1.Values[1].String())
	s.Require().Equal("3", got1.Values[2].String())
	s.Require().Equal("0", got1.Values[3].String())

	got2 := byWorker[msg2.WorkerDataBundle.Worker]
	s.Require().NotNil(got2)
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
	msg1 = s.signMsgInsertWorkerPayload(msg1, pk1)

	// register nonce2 for the same topic (epoch 2)
	err := s.NonceKeeper().AddWorkerNonce(s.Ctx(), topicId, &types.Nonce{BlockHeight: nonce2})
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
	msg2 = s.signMsgInsertWorkerPayload(msg2, pk2)

	// whitelist both workers
	err = s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), topicId, msg1.WorkerDataBundle.Worker)
	s.Require().NoError(err)
	err = s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), topicId, msg2.WorkerDataBundle.Worker)
	s.Require().NoError(err)

	// submit worker1 @ nonce1
	s.WithBlockHeight(nonce1)
	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &msg1)
	s.Require().NoError(err)

	got1, err := s.WorkerKeeper().GetWorkerLatestInferenceByTopicId(s.Ctx(), topicId, msg1.WorkerDataBundle.Worker)
	s.Require().NoError(err)
	s.Require().Equal(2, len(got1.Values))
	s.Require().Equal("1", got1.Values[0].String())
	s.Require().Equal("2", got1.Values[1].String())

	// submit worker2 @ nonce2
	s.WithBlockHeight(nonce2)
	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &msg2)
	s.Require().NoError(err)

	got2, err := s.WorkerKeeper().GetWorkerLatestInferenceByTopicId(s.Ctx(), topicId, msg2.WorkerDataBundle.Worker)
	s.Require().NoError(err)
	s.Require().Equal(3, len(got2.Values))
	s.Require().Equal("10", got2.Values[0].String())
	s.Require().Equal("20", got2.Values[1].String())
	s.Require().Equal("30", got2.Values[2].String())

	topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)
	_, err = s.WorkerKeeper().LoadActiveInfererInferencesForClose(
		s.Ctx(),
		topic,
		nonce1,
		[]string{msg2.WorkerDataBundle.Worker},
	)
	s.Require().Error(err)
	s.Require().ErrorIs(err, sdkerrors.ErrLogic)
}

func (s *MsgServerTestSuite) TestLoadActiveInfererInferencesForClose_MissingTemporaryInference() {
	s.SetupTest()
	pk := secp256k1.GenPrivKey()
	_, topicId := s.setUpMsgInsertWorkerPayload(pk) // proven helper → real topic
	topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)

	missingInferer := s.AddrsStr(9) // active in the set, never staged

	_, err = s.WorkerKeeper().LoadActiveInfererInferencesForClose(
		s.Ctx(), topic, int64(1), []string{missingInferer},
	)
	s.Require().Error(err)
	s.Require().ErrorIs(err, sdkerrors.ErrLogic)
	s.Require().Contains(err.Error(), "missing temporary inference for active inferer")
}

func (s *MsgServerTestSuite) TestMsgInsertWorkerPayload_NotAdmittedDoesNotStageInput() {
	s.SetupTest()

	nonce := int64(1)
	pk := secp256k1.GenPrivKey()
	msg, topicId := s.setUpMsgInsertWorkerPayload(pk)
	s.WithBlockHeight(nonce)
	msg.WorkerDataBundle.Nonce.BlockHeight = nonce
	msg.WorkerDataBundle.InferenceForecastsBundle.Inference.BlockHeight = nonce
	msg.WorkerDataBundle.InferenceForecastsBundle.Forecast = nil
	msg = s.signMsgInsertWorkerPayload(msg, pk)

	params := types.DefaultParams()
	params.MaxTopInferersToReward = 1
	s.Require().NoError(s.ParamsKeeper().SetParams(s.Ctx(), params))

	activeInferer := s.AddrsStr(9)
	activeScore := types.Score{
		TopicId:     topicId,
		BlockHeight: nonce,
		Address:     activeInferer,
		Score:       alloraMath.NewDecFromInt64(100),
	}
	s.Require().NoError(s.ScoresKeeper().SetInfererScoreEma(s.Ctx(), topicId, activeInferer, activeScore))
	s.Require().NoError(s.ScoresKeeper().SetLowestInfererScoreEma(s.Ctx(), topicId, activeScore))
	s.Require().NoError(s.WorkerKeeper().AddActiveInferer(s.Ctx(), topicId, activeInferer))
	s.Require().NoError(s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), topicId, msg.WorkerDataBundle.Worker))

	_, err := s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &msg)
	s.Require().NoError(err)

	_, err = s.WorkerKeeper().GetWorkerLatestInferenceByTopicId(s.Ctx(), topicId, msg.WorkerDataBundle.Worker)
	s.Require().True(errors.Is(err, collections.ErrNotFound))

	isActive, err := s.WorkerKeeper().IsActiveInferer(s.Ctx(), topicId, msg.WorkerDataBundle.Worker)
	s.Require().NoError(err)
	s.Require().False(isActive)
}

//nolint:exhaustruct
func (s *MsgServerTestSuite) TestMsgInsertWorkerPayload_SingleArityValidationBeforePlanning() {
	type labeledValue struct {
		label string
		value string
	}
	type tc struct {
		name             string
		scalarValue      string
		values           []labeledValue
		makeNonAdmitted  bool
		wantErrIs        error
		wantStoredScalar string
	}

	cases := []tc{
		{
			name:             "scalar_payload_still_succeeds",
			scalarValue:      "100",
			wantStoredScalar: "100",
		},
		{
			name: "canonical_y_label_payload_still_succeeds",
			values: []labeledValue{
				{label: "  Y  ", value: "7"},
			},
			wantStoredScalar: "7",
		},
		{
			name: "two_labeled_values_rejected_before_non_admitted_score_update",
			values: []labeledValue{
				{label: "y", value: "1"},
				{label: "z", value: "2"},
			},
			makeNonAdmitted: true,
			wantErrIs:       types.ErrTooManyLabelsPerSubmission,
		},
		{
			name: "non_y_single_label_rejected_before_non_admitted_score_update",
			values: []labeledValue{
				{label: "x", value: "1"},
			},
			makeNonAdmitted: true,
			wantErrIs:       sdkerrors.ErrInvalidRequest,
		},
	}

	makeWorkerNonAdmitted := func(topicId uint64, worker string, nonce types.BlockHeight) {
		params := types.DefaultParams()
		params.MaxTopInferersToReward = 1
		s.Require().NoError(s.ParamsKeeper().SetParams(s.Ctx(), params))

		activeInferer := s.AddrsStr(9)
		activeScore := types.Score{
			TopicId:     topicId,
			BlockHeight: nonce,
			Address:     activeInferer,
			Score:       alloraMath.NewDecFromInt64(100),
		}
		s.Require().NoError(s.ScoresKeeper().SetInfererScoreEma(s.Ctx(), topicId, activeInferer, activeScore))
		s.Require().NoError(s.ScoresKeeper().SetLowestInfererScoreEma(s.Ctx(), topicId, activeScore))
		s.Require().NoError(s.WorkerKeeper().AddActiveInferer(s.Ctx(), topicId, activeInferer))
		score, err := s.ScoresKeeper().GetInfererScoreEma(s.Ctx(), topicId, worker)
		s.Require().NoError(err)
		s.Require().Equal(types.BlockHeight(0), score.BlockHeight)
	}

	for _, c := range cases {
		s.Run(c.name, func() {
			s.SetupTest()

			workerPrivateKey := secp256k1.GenPrivKey()
			msg, topicId := s.setUpMsgInsertWorkerPayload(workerPrivateKey)
			nonce := msg.WorkerDataBundle.Nonce.BlockHeight
			s.WithBlockHeight(nonce)
			s.setTopicArityAndUnity(topicId, types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE, false, "0")

			inference := msg.WorkerDataBundle.InferenceForecastsBundle.Inference
			if c.scalarValue != "" {
				inference.Value = alloraMath.MustNewBoundedExp40DecFromString(c.scalarValue)
			}
			inference.Values = make([]*types.InputLabeledValue, 0, len(c.values))
			for _, value := range c.values {
				inference.Values = append(inference.Values, &types.InputLabeledValue{
					Label: value.label,
					Value: alloraMath.MustNewBoundedExp40DecFromString(value.value),
				})
			}
			if len(c.values) == 0 {
				inference.Values = nil
			}
			msg.WorkerDataBundle.InferenceForecastsBundle.Forecast = nil
			if c.makeNonAdmitted {
				makeWorkerNonAdmitted(topicId, msg.WorkerDataBundle.Worker, nonce)
			}

			msg = s.signMsgInsertWorkerPayload(msg, workerPrivateKey)
			s.Require().NoError(s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), topicId, msg.WorkerDataBundle.Worker))

			_, err := s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &msg)
			if c.wantErrIs != nil {
				s.Require().Error(err)
				s.Require().ErrorIs(err, c.wantErrIs)

				score, scoreErr := s.ScoresKeeper().GetInfererScoreEma(s.Ctx(), topicId, msg.WorkerDataBundle.Worker)
				s.Require().NoError(scoreErr)
				s.Require().Equal(types.BlockHeight(0), score.BlockHeight)

				isActive, activeErr := s.WorkerKeeper().IsActiveInferer(s.Ctx(), topicId, msg.WorkerDataBundle.Worker)
				s.Require().NoError(activeErr)
				s.Require().False(isActive)
				_, inferenceErr := s.WorkerKeeper().GetWorkerLatestInferenceByTopicId(s.Ctx(), topicId, msg.WorkerDataBundle.Worker)
				s.Require().True(errors.Is(inferenceErr, collections.ErrNotFound), "expected no stored inference, got %v", inferenceErr)
				return
			}

			s.Require().NoError(err)
			stored, err := s.WorkerKeeper().GetWorkerLatestInferenceByTopicId(s.Ctx(), topicId, msg.WorkerDataBundle.Worker)
			s.Require().NoError(err)
			s.Require().Len(stored.Values, 1)
			s.Require().Equal(c.wantStoredScalar, stored.Values[0].String())
		})
	}
}

//nolint:exhaustruct
func (s *MsgServerTestSuite) TestMsgInsertWorkerPayload_LabelRegistryAdmissionPlanning() {
	type labeledValue struct {
		label string
		value string
	}
	type tc struct {
		name               string
		values             []labeledValue
		keepForecast       bool
		setup              func(topicId uint64, msg *types.InsertWorkerPayloadRequest)
		wantErrIs          error
		wantRegistry       []string
		wantActive         bool
		wantInference      bool
		wantForecast       bool
		wantCandidateScore bool
	}

	cases := []tc{
		{
			name: "low_score_never_admitted_succeeds_without_growing_registry",
			values: []labeledValue{
				{label: "large-new-label", value: "1"},
			},
			setup: func(topicId uint64, msg *types.InsertWorkerPayloadRequest) {
				params := types.DefaultParams()
				params.MaxTopInferersToReward = 1
				s.Require().NoError(s.ParamsKeeper().SetParams(s.Ctx(), params))
				activeInferer := s.AddrsStr(9)
				activeScore := types.Score{
					TopicId:     topicId,
					BlockHeight: 1,
					Address:     activeInferer,
					Score:       alloraMath.NewDecFromInt64(100),
				}
				s.Require().NoError(s.ScoresKeeper().SetInfererScoreEma(s.Ctx(), topicId, activeInferer, activeScore))
				s.Require().NoError(s.ScoresKeeper().SetLowestInfererScoreEma(s.Ctx(), topicId, activeScore))
				s.Require().NoError(s.WorkerKeeper().AddActiveInferer(s.Ctx(), topicId, activeInferer))
				msg.WorkerDataBundle.InferenceForecastsBundle.Forecast = nil
			},
			wantCandidateScore: true,
		},
		{
			name: "admitted_payload_registers_labels_and_stores_inference",
			values: []labeledValue{
				{label: "admitted-label", value: "1"},
			},
			setup: func(_ uint64, msg *types.InsertWorkerPayloadRequest) {
				msg.WorkerDataBundle.InferenceForecastsBundle.Forecast = nil
			},
			wantRegistry:  []string{"admitted-label"},
			wantActive:    true,
			wantInference: true,
		},
		{
			name: "admitted_by_eviction_registers_labels_and_replaces_lowest_active",
			values: []labeledValue{
				{label: "replacement-label", value: "1"},
			},
			setup: func(topicId uint64, msg *types.InsertWorkerPayloadRequest) {
				params := types.DefaultParams()
				params.MaxTopInferersToReward = 1
				s.Require().NoError(s.ParamsKeeper().SetParams(s.Ctx(), params))
				activeInferer := s.AddrsStr(9)
				activeScore := types.Score{
					TopicId:     topicId,
					BlockHeight: 1,
					Address:     activeInferer,
					Score:       alloraMath.NewDecFromInt64(10),
				}
				candidateScore := types.Score{
					TopicId:     topicId,
					BlockHeight: 1,
					Address:     msg.WorkerDataBundle.Worker,
					Score:       alloraMath.NewDecFromInt64(100),
				}
				s.Require().NoError(s.ScoresKeeper().SetPreviousTopicQuantileInfererScoreEma(s.Ctx(), topicId, alloraMath.NewDecFromInt64(1000)))
				s.Require().NoError(s.ScoresKeeper().SetInfererScoreEma(s.Ctx(), topicId, activeInferer, activeScore))
				s.Require().NoError(s.ScoresKeeper().SetInfererScoreEma(s.Ctx(), topicId, msg.WorkerDataBundle.Worker, candidateScore))
				s.Require().NoError(s.ScoresKeeper().SetLowestInfererScoreEma(s.Ctx(), topicId, activeScore))
				s.Require().NoError(s.WorkerKeeper().AddActiveInferer(s.Ctx(), topicId, activeInferer))
				s.Require().NoError(s.WorkerKeeper().InsertInference(s.Ctx(), topicId, types.Inference{
					TopicId:     topicId,
					BlockHeight: msg.WorkerDataBundle.Nonce.BlockHeight,
					Inferer:     activeInferer,
					Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(1)},
				}))
				msg.WorkerDataBundle.InferenceForecastsBundle.Forecast = nil
			},
			wantRegistry:  []string{"replacement-label"},
			wantActive:    true,
			wantInference: true,
		},
		{
			name: "all_default_multi_rejected_when_worker_would_be_admitted",
			values: []labeledValue{
				{label: "default-label", value: "0"},
			},
			setup: func(_ uint64, msg *types.InsertWorkerPayloadRequest) {
				msg.WorkerDataBundle.InferenceForecastsBundle.Forecast = nil
			},
			wantErrIs: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "all_default_multi_from_never_admitted_worker_is_noop_without_registry_growth",
			values: []labeledValue{
				{label: "default-label", value: "0"},
			},
			setup: func(topicId uint64, msg *types.InsertWorkerPayloadRequest) {
				params := types.DefaultParams()
				params.MaxTopInferersToReward = 1
				s.Require().NoError(s.ParamsKeeper().SetParams(s.Ctx(), params))
				activeInferer := s.AddrsStr(9)
				activeScore := types.Score{
					TopicId:     topicId,
					BlockHeight: 1,
					Address:     activeInferer,
					Score:       alloraMath.NewDecFromInt64(100),
				}
				s.Require().NoError(s.ScoresKeeper().SetInfererScoreEma(s.Ctx(), topicId, activeInferer, activeScore))
				s.Require().NoError(s.ScoresKeeper().SetLowestInfererScoreEma(s.Ctx(), topicId, activeScore))
				s.Require().NoError(s.WorkerKeeper().AddActiveInferer(s.Ctx(), topicId, activeInferer))
				msg.WorkerDataBundle.InferenceForecastsBundle.Forecast = nil
			},
			wantCandidateScore: true,
		},
		{
			name: "validation_failure_still_happens_before_planning",
			values: []labeledValue{
				{label: "a", value: "1"},
				{label: "b", value: "1"},
			},
			setup: func(topicId uint64, msg *types.InsertWorkerPayloadRequest) {
				topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
				s.Require().NoError(err)
				topic.MaxLabelsPerSubmission = 1
				s.Require().NoError(s.TopicKeeper().SetTopic(s.Ctx(), topicId, topic))
				msg.WorkerDataBundle.InferenceForecastsBundle.Forecast = nil
			},
			wantErrIs: types.ErrTooManyLabelsPerSubmission,
		},
		{
			name: "mixed_non_admitted_inference_still_processes_forecast",
			values: []labeledValue{
				{label: "large-new-label", value: "1"},
			},
			keepForecast: true,
			setup: func(topicId uint64, _ *types.InsertWorkerPayloadRequest) {
				params := types.DefaultParams()
				params.MaxTopInferersToReward = 1
				s.Require().NoError(s.ParamsKeeper().SetParams(s.Ctx(), params))
				activeInferer := s.AddrsStr(9)
				activeScore := types.Score{
					TopicId:     topicId,
					BlockHeight: 1,
					Address:     activeInferer,
					Score:       alloraMath.NewDecFromInt64(100),
				}
				s.Require().NoError(s.ScoresKeeper().SetInfererScoreEma(s.Ctx(), topicId, activeInferer, activeScore))
				s.Require().NoError(s.ScoresKeeper().SetLowestInfererScoreEma(s.Ctx(), topicId, activeScore))
				s.Require().NoError(s.WorkerKeeper().AddActiveInferer(s.Ctx(), topicId, activeInferer))
			},
			wantForecast:       true,
			wantCandidateScore: true,
		},
		{
			name: "label_not_in_whitelist_rejected_before_planning",
			values: []labeledValue{
				{label: "bear", value: "1"},
			},
			setup: func(topicId uint64, msg *types.InsertWorkerPayloadRequest) {
				// LabelWhitelist canonicalization is applied by SetTopic/UpdateTopic
				// in production; "bull" is already canonical, so set it directly here.
				topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
				s.Require().NoError(err)
				topic.LabelWhitelist = []string{"bull"}
				s.Require().NoError(s.TopicKeeper().SetTopic(s.Ctx(), topicId, topic))
				msg.WorkerDataBundle.InferenceForecastsBundle.Forecast = nil
			},
			wantErrIs: types.ErrLabelNotInWhitelist,
		},
	}

	for _, c := range cases {
		s.Run(c.name, func() {
			s.SetupTest()
			workerPrivateKey := secp256k1.GenPrivKey()
			nonce := int64(10)
			msg, topicId := s.setUpMsgInsertWorkerPayload(workerPrivateKey)
			s.Require().NoError(s.NonceKeeper().AddWorkerNonce(s.Ctx(), topicId, &types.Nonce{BlockHeight: nonce}))
			s.WithBlockHeight(nonce)
			s.setTopicArityAndUnity(topicId, types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI, false, "0")
			msg.WorkerDataBundle.Nonce.BlockHeight = nonce
			msg.WorkerDataBundle.InferenceForecastsBundle.Inference.BlockHeight = nonce
			msg.WorkerDataBundle.InferenceForecastsBundle.Inference.Values = make([]*types.InputLabeledValue, 0, len(c.values))
			for _, value := range c.values {
				msg.WorkerDataBundle.InferenceForecastsBundle.Inference.Values = append(
					msg.WorkerDataBundle.InferenceForecastsBundle.Inference.Values,
					&types.InputLabeledValue{
						Label: value.label,
						Value: alloraMath.MustNewBoundedExp40DecFromString(value.value),
					},
				)
			}
			if msg.WorkerDataBundle.InferenceForecastsBundle.Forecast != nil {
				msg.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight = nonce
			}
			if c.setup != nil {
				c.setup(topicId, &msg)
			}
			if !c.keepForecast {
				msg.WorkerDataBundle.InferenceForecastsBundle.Forecast = nil
			}
			msg = s.signMsgInsertWorkerPayload(msg, workerPrivateKey)
			s.Require().NoError(s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), topicId, msg.WorkerDataBundle.Worker))

			registryBefore, err := s.TopicKeeper().GetEpochLabelRegistry(s.Ctx(), topicId, nonce)
			s.Require().NoError(err)
			s.Require().Empty(registryBefore.Labels)

			_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &msg)
			if c.wantErrIs != nil {
				s.Require().Error(err)
				s.Require().ErrorIs(err, c.wantErrIs)
			} else {
				s.Require().NoError(err)
			}

			registryAfter, err := s.TopicKeeper().GetEpochLabelRegistry(s.Ctx(), topicId, nonce)
			s.Require().NoError(err)
			s.Require().Len(registryAfter.Labels, len(c.wantRegistry))
			for i, label := range c.wantRegistry {
				s.Require().Equal(label, registryAfter.Labels[i].Name)
			}

			isActive, err := s.WorkerKeeper().IsActiveInferer(s.Ctx(), topicId, msg.WorkerDataBundle.Worker)
			s.Require().NoError(err)
			s.Require().Equal(c.wantActive, isActive)
			_, err = s.WorkerKeeper().GetWorkerLatestInferenceByTopicId(s.Ctx(), topicId, msg.WorkerDataBundle.Worker)
			if c.wantInference {
				s.Require().NoError(err)
			} else {
				s.Require().True(errors.Is(err, collections.ErrNotFound), "expected no stored inference, got %v", err)
			}
			_, err = s.WorkerKeeper().GetWorkerLatestForecastByTopicId(s.Ctx(), topicId, msg.WorkerDataBundle.Worker)
			if c.wantForecast {
				s.Require().NoError(err)
			} else {
				s.Require().True(errors.Is(err, collections.ErrNotFound), "expected no stored forecast, got %v", err)
			}
			score, err := s.ScoresKeeper().GetInfererScoreEma(s.Ctx(), topicId, msg.WorkerDataBundle.Worker)
			s.Require().NoError(err)
			if c.wantCandidateScore {
				s.Require().Equal(nonce, score.BlockHeight)
			} else if c.wantErrIs != nil {
				s.Require().Equal(types.BlockHeight(0), score.BlockHeight)
			}
		})
	}
}

// TestMsgInsertWorkerPayload_Multi_RegistrySaturationRejectsNewLabel exercises the
// #947 epoch label registry cap (MaxEpochLabelRegistrySize) end-to-end through
// InsertWorkerPayload. Keeper-level tests cover RegisterEpochLabels directly; this
// asserts that the only worker-payload path that grows the registry (an admitted
// worker reaching NormalizeInputInference) is bounded by the cap. With the cap at 2,
// a first admitted payload fills the registry with two labels, and a second admitted
// payload that introduces a third label is rejected with ErrEpochLabelRegistrySaturated
// without partially writing the new label.
func (s *MsgServerTestSuite) TestMsgInsertWorkerPayload_Multi_RegistrySaturationRejectsNewLabel() {
	s.SetupTest()

	pk1 := secp256k1.GenPrivKey()
	pk2 := secp256k1.GenPrivKey()
	nonce := int64(1)

	// Both setups run FullTopicSetup; configure arity and the cap afterwards so a
	// re-run cannot reset them.
	msg1, topicId := s.setUpMsgInsertWorkerPayload(pk1)
	msg2, _ := s.setUpMsgInsertWorkerPayload(pk2)
	s.WithBlockHeight(nonce)
	s.setTopicArityAndUnity(topicId, types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI, false, "0")

	// Cap the epoch label registry at two labels.
	params := types.DefaultParams()
	params.MaxEpochLabelRegistrySize = 2
	s.Require().NoError(s.ParamsKeeper().SetParams(s.Ctx(), params))

	// worker1 fills the registry to the cap with labels a, b.
	msg1.WorkerDataBundle.Nonce.BlockHeight = nonce
	msg1.WorkerDataBundle.InferenceForecastsBundle.Inference.BlockHeight = nonce
	msg1.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight = nonce
	msg1.WorkerDataBundle.InferenceForecastsBundle.Inference.Values = []*types.InputLabeledValue{
		{Label: "a", Value: alloraMath.MustNewBoundedExp40DecFromString("1")},
		{Label: "b", Value: alloraMath.MustNewBoundedExp40DecFromString("2")},
	}
	msg1 = s.signMsgInsertWorkerPayload(msg1, pk1)

	// worker2 is admitted (MaxTopInferersToReward default leaves room) and introduces
	// a third label c; a is idempotent and does not by itself grow the registry.
	msg2.WorkerDataBundle.TopicId = topicId
	msg2.WorkerDataBundle.Nonce.BlockHeight = nonce
	msg2.WorkerDataBundle.InferenceForecastsBundle.Inference.TopicId = topicId
	msg2.WorkerDataBundle.InferenceForecastsBundle.Forecast.TopicId = topicId
	msg2.WorkerDataBundle.InferenceForecastsBundle.Inference.BlockHeight = nonce
	msg2.WorkerDataBundle.InferenceForecastsBundle.Forecast.BlockHeight = nonce
	msg2.WorkerDataBundle.InferenceForecastsBundle.Inference.Values = []*types.InputLabeledValue{
		{Label: "a", Value: alloraMath.MustNewBoundedExp40DecFromString("10")},
		{Label: "c", Value: alloraMath.MustNewBoundedExp40DecFromString("30")},
	}
	msg2 = s.signMsgInsertWorkerPayload(msg2, pk2)

	err := s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), topicId, msg1.WorkerDataBundle.Worker)
	s.Require().NoError(err)
	err = s.WhitelistsKeeper().AddToTopicWorkerWhitelist(s.Ctx(), topicId, msg2.WorkerDataBundle.Worker)
	s.Require().NoError(err)

	// First payload succeeds; registry sits at the cap.
	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &msg1)
	s.Require().NoError(err)
	reg, err := s.TopicKeeper().GetEpochLabelRegistry(s.Ctx(), topicId, nonce)
	s.Require().NoError(err)
	s.Require().Equal(2, len(reg.Labels))

	// Second admitted payload introduces a third label -> saturation.
	_, err = s.EmissionsMsgServer().InsertWorkerPayload(s.Ctx(), &msg2)
	s.Require().Error(err)
	s.Require().True(
		errors.Is(err, types.ErrEpochLabelRegistrySaturated),
		"expected ErrEpochLabelRegistrySaturated, got %v", err,
	)

	// Registry is unchanged: the rejected label was not partially written.
	reg, err = s.TopicKeeper().GetEpochLabelRegistry(s.Ctx(), topicId, nonce)
	s.Require().NoError(err)
	s.Require().Equal(2, len(reg.Labels))
	s.Require().Equal("a", reg.Labels[0].Name)
	s.Require().Equal("b", reg.Labels[1].Name)
}

func (s *MsgServerTestSuite) setTopicArityAndUnity(
	topicId uint64,
	outputArity types.TopicOutputArity,
	requireUnity bool,
	unityTol string,
) {
	ctx := s.Ctx()
	k := s.TopicKeeper()

	topic, err := k.GetTopic(ctx, topicId)
	s.Require().NoError(err)

	topic.OutputArity = outputArity
	topic.RequireUnity = requireUnity
	topic.UnityTolerance = alloraMath.MustNewDecFromString(unityTol)

	err = k.SetTopic(ctx, topicId, topic)
	s.Require().NoError(err)
}
