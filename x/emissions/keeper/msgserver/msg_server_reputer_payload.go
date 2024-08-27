package msgserver

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	"encoding/hex"

	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

// A tx function that accepts a individual loss and possibly returns an error
func (ms msgServer) InsertReputerPayload(ctx context.Context, msg *types.MsgInsertReputerPayload) (*types.MsgInsertReputerPayloadResponse, error) {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	err = checkInputLength(ctx, ms, msg)
	if err != nil {
		return nil, err
	}

	blockHeight := sdk.UnwrapSDKContext(ctx).BlockHeight()

	// Call the bundles self validation method
	if err := validateReputerValueBundle(msg.ReputerValueBundle); err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidWorkerData,
			"Error validating reputer value bundle: %v", err)
	}

	nonce := msg.ReputerValueBundle.ValueBundle.ReputerRequestNonce
	topicId := msg.ReputerValueBundle.ValueBundle.TopicId
	reputer := msg.Sender

	// the reputer in the bundle must match the sender of this transaction
	if reputer != msg.ReputerValueBundle.ValueBundle.Reputer {
		return nil, errorsmod.Wrapf(types.ErrInvalidWorkerData,
			"Reputer cannot upload value bundle for another reputer")
	}

	// Check if the topic exists
	topicExists, err := ms.k.TopicExists(ctx, topicId)
	if err != nil || !topicExists {
		return nil, types.ErrInvalidTopicId
	}

	// Check that the reputer is registered in the topic
	isReputerRegistered, err := ms.k.IsReputerRegisteredInTopic(ctx, topicId, reputer)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "Error checking if reputer is registered in topic")
	}
	if !isReputerRegistered {
		return nil, errorsmod.Wrapf(types.ErrInvalidWorkerData,
			"Reputer not registered in topic")
	}

	// Check if the worker nonce is fulfilled
	workerNonceUnfulfilled, err := ms.k.IsWorkerNonceUnfulfilled(ctx, topicId, nonce.ReputerNonce)
	if err != nil {
		return nil, err
	}
	// Returns an error if unfulfilled worker nonce exists
	if workerNonceUnfulfilled {
		return nil, types.ErrNonceStillUnfulfilled
	}

	// Check if the reputer nonce is unfulfilled
	reputerNonceUnfulfilled, err := ms.k.IsReputerNonceUnfulfilled(ctx, topicId, nonce.ReputerNonce)
	if err != nil {
		return nil, err
	}
	// If the reputer nonce is already fulfilled, return an error
	if !reputerNonceUnfulfilled {
		return nil, types.ErrUnfulfilledNonceNotFound
	}

	topic, err := ms.k.GetTopic(ctx, topicId)
	if err != nil {
		return nil, types.ErrInvalidTopicId
	}

	// Check if the ground truth lag has passed: if blockheight > nonce.BlockHeight + topic.GroundTruthLag
	if blockHeight < nonce.ReputerNonce.BlockHeight+topic.GroundTruthLag {
		return nil, types.ErrReputerNonceWindowNotAvailable
	}

	moduleParams, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "Error getting params for sender: %v", &msg.Sender)
	}

	// reputer must have minimum stake in order to participate in topic
	reputerStake, err := ms.k.GetStakeReputerAuthority(ctx, topicId, reputer)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "Error getting reputer stake for sender: %v", &msg.Sender)
	}
	if reputerStake.LT(moduleParams.RequiredMinimumStake) {
		return nil, errorsmod.Wrapf(types.ErrInvalidWorkerData,
			"Reputer must have minimum stake in order to participate in topic")
	}

	// Before activating topic, transfer fee amount from creator to ecosystem bucket
	err = sendEffectiveRevenueActivateTopicIfWeightSufficient(ctx, ms, msg.Sender, topicId, moduleParams.DataSendingFee)
	if err != nil {
		return nil, err
	}

	err = ms.k.UpsertReputerLoss(ctx, topicId, nonce.ReputerNonce.BlockHeight, msg.ReputerValueBundle)
	if err != nil {
		return nil, err
	}

	return &types.MsgInsertReputerPayloadResponse{}, nil
}

// validateWorkerAttributedValue validates a WorkerAttributedValue
func validateWorkerAttributedValue(workerValue *types.WorkerAttributedValue) error {
	_, err := sdk.AccAddressFromBech32(workerValue.Worker)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid worker address (%s)", err)
	}

	if err := validateDec(workerValue.Value); err != nil {
		return err
	}

	return nil
}

// validateWithheldWorkerAttributedValue validates a WithheldWorkerAttributedValue
func validateWithheldWorkerAttributedValue(withheldWorkerValue *types.WithheldWorkerAttributedValue) error {
	_, err := sdk.AccAddressFromBech32(withheldWorkerValue.Worker)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid withheld worker address (%s)", err)
	}

	if err := validateDec(withheldWorkerValue.Value); err != nil {
		return err
	}

	return nil
}

// validateReputerValueBundle validates a ReputerValueBundle
func validateReputerValueBundle(bundle *types.ReputerValueBundle) error {
	if bundle.ValueBundle == nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "value bundle cannot be nil")
	}

	_, err := sdk.AccAddressFromBech32(bundle.ValueBundle.Reputer)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid reputer address (%s)", err)
	}

	if bundle.ValueBundle.ReputerRequestNonce == nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "value bundle's reputer request nonce cannot be nil")
	}

	if err := validateDec(bundle.ValueBundle.CombinedValue); err != nil {
		return err
	}

	if err := validateDec(bundle.ValueBundle.NaiveValue); err != nil {
		return err
	}

	for _, infererValue := range bundle.ValueBundle.InfererValues {
		if err := validateWorkerAttributedValue(infererValue); err != nil {
			return err
		}
	}

	for _, forecasterValue := range bundle.ValueBundle.ForecasterValues {
		if err := validateWorkerAttributedValue(forecasterValue); err != nil {
			return err
		}
	}

	for _, oneOutInfererValue := range bundle.ValueBundle.OneOutInfererValues {
		if err := validateWithheldWorkerAttributedValue(oneOutInfererValue); err != nil {
			return err
		}
	}

	for _, oneOutForecasterValue := range bundle.ValueBundle.OneOutForecasterValues {
		if err := validateWithheldWorkerAttributedValue(oneOutForecasterValue); err != nil {
			return err
		}
	}

	for _, oneInForecasterValue := range bundle.ValueBundle.OneInForecasterValues {
		if err := validateWorkerAttributedValue(oneInForecasterValue); err != nil {
			return err
		}
	}

	for _, oneOutInfererForecaster := range bundle.ValueBundle.OneOutInfererForecasterValues {
		_, err := sdk.AccAddressFromBech32(oneOutInfererForecaster.Forecaster)
		if err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid forecaster address in OneOutInfererForecasterValues (%s)", err)
		}
		for _, oneOutInfererValue := range oneOutInfererForecaster.OneOutInfererValues {
			if err := validateWithheldWorkerAttributedValue(oneOutInfererValue); err != nil {
				return err
			}
		}
	}

	pk, err := hex.DecodeString(bundle.Pubkey)
	if err != nil || len(pk) != secp256k1.PubKeySize {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "signature verification failed")
	}
	pubkey := secp256k1.PubKey(pk)

	src := make([]byte, 0)
	src, _ = bundle.ValueBundle.XXX_Marshal(src, true)
	if !pubkey.VerifySignature(src, bundle.Signature) {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "signature verification failed")
	}

	return nil
}
