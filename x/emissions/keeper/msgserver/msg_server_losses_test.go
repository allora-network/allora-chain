package msgserver_test

import (
	"encoding/hex"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (s *MsgServerTestSuite) getBasicReputerPayload(
	reputerAddr sdk.AccAddress,
	workerAddr sdk.AccAddress,
	block types.BlockHeight,
) (
	reputerValueBundle types.ValueBundle,
	expectedInferences types.Inferences,
	expectedForecasts types.Forecasts,
	topicId uint64,
	reputerNonce types.Nonce,
	workerNonce types.Nonce,
) {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	keeper := s.emissionsKeeper

	params, err := keeper.GetParams(ctx)
	require.NoError(err)

	minStakeScaled := params.RequiredMinimumStake.Mul(inference_synthesis.CosmosIntOneE18())

	topicId = s.commonStakingSetup(ctx, reputerAddr.String(), workerAddr.String(), minStakeScaled)
	s.MintTokensToAddress(reputerAddr, params.RequiredMinimumStake)

	addStakeMsg := &types.MsgAddStake{
		Sender:  reputerAddr.String(),
		TopicId: topicId,
		Amount:  minStakeScaled,
	}

	_, err = msgServer.AddStake(ctx, addStakeMsg)
	s.Require().NoError(err)

	reputerNonce = types.Nonce{
		BlockHeight: block,
	}
	workerNonce = types.Nonce{
		BlockHeight: block,
	}

	keeper.AddWorkerNonce(ctx, topicId, &workerNonce)
	keeper.FulfillWorkerNonce(ctx, topicId, &workerNonce)
	keeper.AddReputerNonce(ctx, topicId, &reputerNonce, &workerNonce)

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
			ReputerNonce: &reputerNonce,
			WorkerNonce:  &workerNonce,
		},
	}

	return reputerValueBundle, expectedInferences, expectedForecasts, topicId, reputerNonce, workerNonce
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
	topicId uint64,
	reputerNonce *types.Nonce,
	workerNonce *types.Nonce,
) error {
	ctx, msgServer := s.ctx, s.msgServer
	valueBundleSignature := s.signValueBundle(reputerValueBundle, reputerPrivateKey)

	// Create a MsgInsertBulkReputerPayload message
	lossesMsg := &types.MsgInsertBulkReputerPayload{
		Sender:  reputerAddr.String(),
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: reputerNonce,
			WorkerNonce:  workerNonce,
		},
		ReputerValueBundles: []*types.ReputerValueBundle{
			{
				ValueBundle: reputerValueBundle,
				Signature:   valueBundleSignature,
				Pubkey:      hex.EncodeToString(reputerPublicKeyBytes),
			},
		},
	}

	_, err := msgServer.InsertBulkReputerPayload(ctx, lossesMsg)
	return err
}

func (s *MsgServerTestSuite) TestMsgInsertBulkReputerPayload() {
	ctx := s.ctx
	require := s.Require()
	keeper := s.emissionsKeeper

	block := types.BlockHeight(1)

	reputerPrivateKey := secp256k1.GenPrivKey()
	reputerPublicKeyBytes := reputerPrivateKey.PubKey().Bytes()
	reputerAddr := sdk.AccAddress(reputerPrivateKey.PubKey().Address())

	workerPrivateKey := secp256k1.GenPrivKey()
	workerAddr := sdk.AccAddress(workerPrivateKey.PubKey().Address())

	reputerValueBundle, expectedInferences, expectedForecasts, topicId, reputerNonce, workerNonce := s.getBasicReputerPayload(reputerAddr, workerAddr, block)

	err := keeper.InsertForecasts(ctx, topicId, types.Nonce{BlockHeight: int64(block)}, expectedForecasts)
	require.NoError(err)

	err = keeper.InsertInferences(ctx, topicId, types.Nonce{BlockHeight: block}, expectedInferences)
	require.NoError(err)

	err = s.constructAndInsertReputerPayload(reputerAddr, reputerPrivateKey, reputerPublicKeyBytes, &reputerValueBundle, topicId, &reputerNonce, &workerNonce)
	require.NoError(err)
}

func (s *MsgServerTestSuite) TestInsertingReputerPayloadWithMismatchedTopicIdsIsIgnored() {
	ctx := s.ctx
	require := s.Require()
	keeper := s.emissionsKeeper

	block := types.BlockHeight(1)

	reputerPrivateKey := secp256k1.GenPrivKey()
	reputerPublicKeyBytes := reputerPrivateKey.PubKey().Bytes()
	reputerAddr := sdk.AccAddress(reputerPrivateKey.PubKey().Address())

	workerPrivateKey := secp256k1.GenPrivKey()
	workerAddr := sdk.AccAddress(workerPrivateKey.PubKey().Address())

	reputerValueBundle, expectedInferences, expectedForecasts, topicId, reputerNonce, workerNonce := s.getBasicReputerPayload(reputerAddr, workerAddr, block)

	// BEGIN MODIFICATION
	reputerValueBundle.TopicId = topicId + 1
	// END MODIFICATION

	err := keeper.InsertForecasts(ctx, topicId, types.Nonce{BlockHeight: int64(block)}, expectedForecasts)
	require.NoError(err)

	err = keeper.InsertInferences(ctx, topicId, types.Nonce{BlockHeight: block}, expectedInferences)
	require.NoError(err)

	err = s.constructAndInsertReputerPayload(
		reputerAddr,
		reputerPrivateKey,
		reputerPublicKeyBytes,
		&reputerValueBundle,
		topicId,
		&reputerNonce,
		&workerNonce,
	)

	require.ErrorIs(err, types.ErrNoValidBundles)
}

func (s *MsgServerTestSuite) TestInsertingReputerPayloadWithMismatchedReputerNonceIsIgnored() {
	ctx := s.ctx
	require := s.Require()
	keeper := s.emissionsKeeper

	block := types.BlockHeight(1)

	reputerPrivateKey := secp256k1.GenPrivKey()
	reputerPublicKeyBytes := reputerPrivateKey.PubKey().Bytes()
	reputerAddr := sdk.AccAddress(reputerPrivateKey.PubKey().Address())

	workerPrivateKey := secp256k1.GenPrivKey()
	workerAddr := sdk.AccAddress(workerPrivateKey.PubKey().Address())

	reputerValueBundle, expectedInferences, expectedForecasts, topicId, reputerNonce, workerNonce := s.getBasicReputerPayload(reputerAddr, workerAddr, block)

	// BEGIN MODIFICATION
	reputerValueBundle.ReputerRequestNonce.ReputerNonce.BlockHeight = 123
	// END MODIFICATION

	err := keeper.InsertForecasts(ctx, topicId, types.Nonce{BlockHeight: block}, expectedForecasts)
	require.NoError(err)

	err = keeper.InsertInferences(ctx, topicId, types.Nonce{BlockHeight: block}, expectedInferences)
	require.NoError(err)

	err = s.constructAndInsertReputerPayload(
		reputerAddr,
		reputerPrivateKey,
		reputerPublicKeyBytes,
		&reputerValueBundle,
		topicId,
		&reputerNonce,
		&workerNonce,
	)

	require.ErrorIs(err, types.ErrNoValidBundles)
}

func (s *MsgServerTestSuite) TestInsertingReputerPayloadWithUnregisteredReputerIsIgnored() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	keeper := s.emissionsKeeper

	block := types.BlockHeight(1)

	reputerPrivateKey := secp256k1.GenPrivKey()
	reputerPublicKeyBytes := reputerPrivateKey.PubKey().Bytes()
	reputerAddr := sdk.AccAddress(reputerPrivateKey.PubKey().Address())

	workerPrivateKey := secp256k1.GenPrivKey()
	workerAddr := sdk.AccAddress(workerPrivateKey.PubKey().Address())

	reputerValueBundle, expectedInferences, expectedForecasts, topicId, reputerNonce, workerNonce := s.getBasicReputerPayload(reputerAddr, workerAddr, block)

	// BEGIN MODIFICATION
	unregisterMsg := &types.MsgRemoveRegistration{
		Sender:    reputerAddr.String(),
		TopicId:   topicId,
		IsReputer: true,
	}

	_, err := msgServer.RemoveRegistration(ctx, unregisterMsg)
	require.NoError(err)
	// END MODIFICATION

	err = keeper.InsertForecasts(ctx, topicId, types.Nonce{BlockHeight: block}, expectedForecasts)
	require.NoError(err)

	err = keeper.InsertInferences(ctx, topicId, types.Nonce{BlockHeight: block}, expectedInferences)
	require.NoError(err)

	err = s.constructAndInsertReputerPayload(
		reputerAddr,
		reputerPrivateKey,
		reputerPublicKeyBytes,
		&reputerValueBundle,
		topicId,
		&reputerNonce,
		&workerNonce,
	)

	require.ErrorIs(err, types.ErrNoValidBundles)
}

func (s *MsgServerTestSuite) TestInsertingReputerPayloadWithUnderstakeReputerIsIgnored() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	keeper := s.emissionsKeeper

	reputerPrivateKey := secp256k1.GenPrivKey()
	reputerPublicKeyBytes := reputerPrivateKey.PubKey().Bytes()
	reputerAddr := sdk.AccAddress(reputerPrivateKey.PubKey().Address())

	workerPrivateKey := secp256k1.GenPrivKey()
	workerAddr := sdk.AccAddress(workerPrivateKey.PubKey().Address())

	// BEGIN MODIFICATION
	block := ctx.BlockHeight()

	reputerValueBundle, expectedInferences, expectedForecasts, topicId, reputerNonce, workerNonce := s.getBasicReputerPayload(reputerAddr, workerAddr, block)

	reputerStake, err := keeper.GetStakeReputerAuthority(ctx, topicId, reputerAddr.String())
	require.NoError(err)

	params, err := keeper.GetParams(ctx)
	require.NoError(err)
	removalDelay := params.RemoveStakeDelayWindow

	startRemoveStakeMsg := &types.MsgRemoveStake{
		Sender:  reputerAddr.String(),
		TopicId: topicId,
		Amount:  reputerStake,
	}

	_, err = msgServer.RemoveStake(ctx, startRemoveStakeMsg)
	require.NoError(err)

	block = block + removalDelay
	ctx = ctx.WithBlockHeight(block)

	// run the end block to force the removal of stake to go through
	s.appModule.EndBlock(ctx)

	// END MODIFICATION

	err = keeper.InsertForecasts(ctx, topicId, types.Nonce{BlockHeight: block}, expectedForecasts)
	require.NoError(err)

	err = keeper.InsertInferences(ctx, topicId, types.Nonce{BlockHeight: block}, expectedInferences)
	require.NoError(err, "InsertInferences should not return an error")

	err = s.constructAndInsertReputerPayload(
		reputerAddr,
		reputerPrivateKey,
		reputerPublicKeyBytes,
		&reputerValueBundle,
		topicId,
		&reputerNonce,
		&workerNonce,
	)

	require.ErrorIs(err, types.ErrNoValidBundles)
}

func (s *MsgServerTestSuite) TestInsertingReputerPayloadWithMissingInferencesIsIgnored() {
	ctx := s.ctx
	require := s.Require()
	keeper := s.emissionsKeeper

	block := types.BlockHeight(1)

	reputerPrivateKey := secp256k1.GenPrivKey()
	reputerPublicKeyBytes := reputerPrivateKey.PubKey().Bytes()
	reputerAddr := sdk.AccAddress(reputerPrivateKey.PubKey().Address())

	workerPrivateKey := secp256k1.GenPrivKey()
	workerAddr := sdk.AccAddress(workerPrivateKey.PubKey().Address())

	reputerValueBundle, _, expectedForecasts, topicId, reputerNonce, workerNonce := s.getBasicReputerPayload(reputerAddr, workerAddr, block)

	err := keeper.InsertForecasts(ctx, topicId, types.Nonce{BlockHeight: block}, expectedForecasts)
	require.NoError(err)

	// BEGIN MODIFICATION
	// err = keeper.InsertInferences(ctx, topicId, types.Nonce{BlockHeight: block}, expectedInferences)
	// require.NoError(err, "InsertInferences should not return an error")
	// END MODIFICATION

	err = s.constructAndInsertReputerPayload(
		reputerAddr,
		reputerPrivateKey,
		reputerPublicKeyBytes,
		&reputerValueBundle,
		topicId,
		&reputerNonce,
		&workerNonce,
	)

	require.ErrorIs(err, types.ErrNoValidBundles)
}

func (s *MsgServerTestSuite) TestInsertingReputerPayloadWithIncorrectBaseWorkerNonceIsIgnored() {
	ctx := s.ctx
	require := s.Require()
	keeper := s.emissionsKeeper

	block := types.BlockHeight(1)

	reputerPrivateKey := secp256k1.GenPrivKey()
	reputerPublicKeyBytes := reputerPrivateKey.PubKey().Bytes()
	reputerAddr := sdk.AccAddress(reputerPrivateKey.PubKey().Address())

	workerPrivateKey := secp256k1.GenPrivKey()
	workerAddr := sdk.AccAddress(workerPrivateKey.PubKey().Address())

	reputerValueBundle, expectedInferences, expectedForecasts, topicId, reputerNonce, workerNonce := s.getBasicReputerPayload(reputerAddr, workerAddr, block)

	// BEGIN MODIFICATION
	reputerNonce.BlockHeight = block + 1
	// END MODIFICATION

	err := keeper.InsertForecasts(ctx, topicId, types.Nonce{BlockHeight: int64(block)}, expectedForecasts)
	require.NoError(err)

	err = keeper.InsertInferences(ctx, topicId, types.Nonce{BlockHeight: block}, expectedInferences)
	require.NoError(err)

	err = s.constructAndInsertReputerPayload(
		reputerAddr,
		reputerPrivateKey,
		reputerPublicKeyBytes,
		&reputerValueBundle,
		topicId,
		&reputerNonce,
		&workerNonce,
	)

	require.ErrorIs(err, types.ErrNonceAlreadyFulfilled)
}

func (s *MsgServerTestSuite) TestMsgInsertBulkReputerPayloadInvalid() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	keeper := s.emissionsKeeper

	reputerPrivateKey := secp256k1.GenPrivKey()
	reputerPublicKeyBytes := reputerPrivateKey.PubKey().Bytes()
	reputerAddr := sdk.AccAddress(reputerPrivateKey.PubKey().Address())

	workerPrivateKey := secp256k1.GenPrivKey()
	workerAddr := sdk.AccAddress(workerPrivateKey.PubKey().Address())

	params, err := keeper.GetParams(ctx)
	require.NoError(err)

	minStakeScaled := params.RequiredMinimumStake.Mul(inference_synthesis.CosmosIntOneE18())

	topicId := s.commonStakingSetup(ctx, reputerAddr.String(), workerAddr.String(), minStakeScaled)

	s.MintTokensToAddress(reputerAddr, params.RequiredMinimumStake)

	addStakeMsg := &types.MsgAddStake{
		Sender:  reputerAddr.String(),
		TopicId: topicId,
		Amount:  minStakeScaled,
	}

	_, err = msgServer.AddStake(ctx, addStakeMsg)
	s.Require().NoError(err)

	reputerNonce := &types.Nonce{
		BlockHeight: 2,
	}
	workerNonce := &types.Nonce{
		BlockHeight: 1,
	}

	keeper.AddWorkerNonce(ctx, topicId, workerNonce)
	keeper.FulfillWorkerNonce(ctx, topicId, workerNonce)
	keeper.AddReputerNonce(ctx, topicId, reputerNonce, workerNonce)

	// add in inference and forecast data
	block := types.BlockHeight(1)
	expectedInferences := types.Inferences{
		Inferences: []*types.Inference{
			{
				Value:   alloraMath.NewDecFromInt64(1), // Assuming NewDecFromInt64 exists and is appropriate
				Inferer: workerAddr.String(),
			},
		},
	}

	nonce := types.Nonce{BlockHeight: block} // Assuming block type cast to int64 if needed
	err = keeper.InsertInferences(ctx, topicId, nonce, expectedInferences)
	require.NoError(err, "InsertInferences should not return an error")

	expectedForecasts := types.Forecasts{
		Forecasts: []*types.Forecast{
			{
				TopicId:    topicId,
				Forecaster: workerAddr.String(),
			},
		},
	}

	nonce = types.Nonce{BlockHeight: int64(block)}
	err = keeper.InsertForecasts(ctx, topicId, nonce, expectedForecasts)
	s.Require().NoError(err)

	reputerValueBundle := &types.ValueBundle{
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
			ReputerNonce: reputerNonce,
			WorkerNonce:  workerNonce,
		},
	}

	src := make([]byte, 0)
	src, err = reputerValueBundle.XXX_Marshal(src, true)
	require.NoError(err, "Marshall reputer value bundle should not return an error")

	valueBundleSignature, err := reputerPrivateKey.Sign(src)
	require.NoError(err, "Sign should not return an error")

	// Create a MsgInsertBulkReputerPayload message
	lossesMsg := &types.MsgInsertBulkReputerPayload{
		Sender:  reputerAddr.String(),
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: reputerNonce,
			WorkerNonce:  workerNonce,
		},
		ReputerValueBundles: []*types.ReputerValueBundle{
			{
				ValueBundle: reputerValueBundle,
				Signature:   valueBundleSignature,
				Pubkey:      hex.EncodeToString(reputerPublicKeyBytes),
			},
		},
	}

	// Send to the wrong topic should error
	lossesMsg.TopicId = topicId + 999
	_, err = msgServer.InsertBulkReputerPayload(ctx, lossesMsg)
	require.ErrorIs(err, sdkerrors.ErrNotFound)

	// Fix topic
	lossesMsg.TopicId = topicId

	// Send bundle with one-out inference with just 1 inferer in topic-block
	lossesMsg.ReputerValueBundles[0].ValueBundle.OneOutInfererValues = []*types.WithheldWorkerAttributedValue{
		{
			Worker: workerAddr.String(),
			Value:  alloraMath.NewDecFromInt64(100),
		},
	}
	_, err = msgServer.InsertBulkReputerPayload(ctx, lossesMsg)
	require.ErrorIs(err, types.ErrNoValidBundles)
}

func (s *MsgServerTestSuite) TestMsgInsertHugeBulkReputerPayloadFails() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	keeper := s.emissionsKeeper

	reputerPrivateKey := secp256k1.GenPrivKey()
	reputerPublicKeyBytes := reputerPrivateKey.PubKey().Bytes()
	reputerAddr := sdk.AccAddress(reputerPrivateKey.PubKey().Address())

	workerPrivateKey := secp256k1.GenPrivKey()
	workerAddr := sdk.AccAddress(workerPrivateKey.PubKey().Address())

	params, err := keeper.GetParams(ctx)
	require.NoError(err)

	minStakeScaled := params.RequiredMinimumStake.Mul(inference_synthesis.CosmosIntOneE18())

	topicId := s.commonStakingSetup(ctx, reputerAddr.String(), workerAddr.String(), minStakeScaled)

	s.MintTokensToAddress(reputerAddr, params.RequiredMinimumStake)

	addStakeMsg := &types.MsgAddStake{
		Sender:  reputerAddr.String(),
		TopicId: topicId,
		Amount:  minStakeScaled,
	}

	_, err = msgServer.AddStake(ctx, addStakeMsg)
	s.Require().NoError(err)

	reputerNonce := &types.Nonce{
		BlockHeight: 2,
	}
	workerNonce := &types.Nonce{
		BlockHeight: 1,
	}

	keeper.AddWorkerNonce(ctx, topicId, workerNonce)
	keeper.FulfillWorkerNonce(ctx, topicId, workerNonce)
	keeper.AddReputerNonce(ctx, topicId, reputerNonce, workerNonce)

	// add in inference and forecast data
	block := types.BlockHeight(1)
	expectedInferences := types.Inferences{
		Inferences: []*types.Inference{
			{
				Value:   alloraMath.NewDecFromInt64(1), // Assuming NewDecFromInt64 exists and is appropriate
				Inferer: workerAddr.String(),
			},
		},
	}

	nonce := types.Nonce{BlockHeight: block} // Assuming block type cast to int64 if needed
	err = keeper.InsertInferences(ctx, topicId, nonce, expectedInferences)
	require.NoError(err, "InsertInferences should not return an error")

	expectedForecasts := types.Forecasts{
		Forecasts: []*types.Forecast{
			{
				TopicId:    topicId,
				Forecaster: workerAddr.String(),
			},
		},
	}

	nonce = types.Nonce{BlockHeight: int64(block)}
	err = keeper.InsertForecasts(ctx, topicId, nonce, expectedForecasts)
	s.Require().NoError(err)

	reputerValueBundle := &types.ValueBundle{
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
		NaiveValue: alloraMath.NewDecFromInt64(100),
		OneOutInfererValues: []*types.WithheldWorkerAttributedValue{
			{
				Worker: workerAddr.String(),
				Value:  alloraMath.NewDecFromInt64(100),
			},
		},
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
			ReputerNonce: reputerNonce,
			WorkerNonce:  workerNonce,
		},
	}

	src := make([]byte, 0)
	src, err = reputerValueBundle.XXX_Marshal(src, true)
	require.NoError(err, "Marshall reputer value bundle should not return an error")

	valueBundleSignature, err := reputerPrivateKey.Sign(src)
	require.NoError(err, "Sign should not return an error")

	reputerValueBundles := []*types.ReputerValueBundle{}
	for i := 0; i < 1000000; i++ {
		reputerValueBundles = append(
			reputerValueBundles,
			&types.ReputerValueBundle{
				ValueBundle: reputerValueBundle,
				Signature:   valueBundleSignature,
				Pubkey:      hex.EncodeToString(reputerPublicKeyBytes),
			},
		)
	}

	// Create a MsgInsertBulkReputerPayload message
	lossesMsg := &types.MsgInsertBulkReputerPayload{
		Sender:  reputerAddr.String(),
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: reputerNonce,
			WorkerNonce:  workerNonce,
		},
		ReputerValueBundles: reputerValueBundles,
	}

	_, err = msgServer.InsertBulkReputerPayload(ctx, lossesMsg)
	require.Error(err, types.ErrQueryTooLarge)
}

func (s *MsgServerTestSuite) TestMsgInsertBulkReputerPayloadUpdateTopicCommit() {
	ctx := s.ctx
	require := s.Require()
	keeper := s.emissionsKeeper
	block := types.BlockHeight(1)

	reputerPrivateKey := secp256k1.GenPrivKey()
	reputerPublicKeyBytes := reputerPrivateKey.PubKey().Bytes()
	reputerAddr := sdk.AccAddress(reputerPrivateKey.PubKey().Address())
	workerPrivateKey := secp256k1.GenPrivKey()
	workerAddr := sdk.AccAddress(workerPrivateKey.PubKey().Address())
	reputerValueBundle, expectedInferences, expectedForecasts, topicId, reputerNonce, workerNonce := s.getBasicReputerPayload(reputerAddr, workerAddr, block)
	err := keeper.InsertForecasts(ctx, topicId, types.Nonce{BlockHeight: int64(block)}, expectedForecasts)
	require.NoError(err)
	err = keeper.InsertInferences(ctx, topicId, types.Nonce{BlockHeight: block}, expectedInferences)
	require.NoError(err)

	blockHeight := sdk.UnwrapSDKContext(ctx).BlockHeight()
	err = s.constructAndInsertReputerPayload(
		reputerAddr,
		reputerPrivateKey,
		reputerPublicKeyBytes,
		&reputerValueBundle,
		topicId,
		&reputerNonce,
		&workerNonce,
	)
	require.NoError(err, "InsertReputerPayload should not return an error")

	lastCommit, err := keeper.GetTopicLastCommit(ctx, topicId, types.ActorType_REPUTER)
	require.NoError(err, "GetTopicLastCommit should not return an error")

	require.Equal(blockHeight, lastCommit.BlockHeight, "BlockHeight should be same")
	require.Equal(reputerValueBundle.Reputer, lastCommit.Actor, "Actor should be same")
	require.Equal(reputerValueBundle.ReputerRequestNonce.ReputerNonce, lastCommit.Nonce, "Nonce should be same")
}
