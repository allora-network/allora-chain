package msgserver

import (
	"context"
	"fmt"

	"errors"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	"encoding/hex"

	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/allora-network/allora-chain/x/emissions/keeper"
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
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := sdkCtx.BlockHeight()

	// Call the bundles self validation method
	if err := validateReputerValueBundle(msg.ReputerValueBundle); err != nil {
		return nil, errorsmod.Wrapf(err,
			"Error validating reputer value bundle for block height %d", blockHeight)
	}

	nonce := msg.ReputerValueBundle.ValueBundle.ReputerRequestNonce
	topicId := msg.ReputerValueBundle.ValueBundle.TopicId
	reputer := msg.ReputerValueBundle.ValueBundle.Reputer

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
		return nil, errorsmod.Wrapf(types.ErrInvalidReputerData, "Reputer not registered in topic")
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
	if blockHeight < nonce.ReputerNonce.BlockHeight+topic.GroundTruthLag ||
		blockHeight > nonce.ReputerNonce.BlockHeight+topic.GroundTruthLag*2 {
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
		return nil, errorsmod.Wrapf(types.ErrInvalidReputerData,
			"Reputer must have minimum stake in order to participate in topic")
	}

	// Before activating topic, transfer fee amount from creator to ecosystem bucket
	err = sendEffectiveRevenueActivateTopicIfWeightSufficient(ctx, ms, msg.Sender, topicId, moduleParams.DataSendingFee)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error sending effective revenue")
	}

	// active set management: now we decide if we will accept this reputer payload
	// based on whether this reputer is in the active set or not
	// get all the current reputer loss bundles we've accepted at this nonce epoch
	existingReputerLossBundles, err := ms.k.GetReputerLossBundlesAtBlock(ctx, topicId, nonce.ReputerNonce.BlockHeight)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error getting existing reputer loss bundles")
	}

	// if the number of reputer loss bundles is less than the max number of reputers to reward,
	// we can just accept this reputer payload straight away
	if uint64(len(existingReputerLossBundles.ReputerValueBundles)) < moduleParams.MaxTopReputersToReward {
		filteredLossBundle, err := filterReputerBundle(ctx, ms.k, topicId, *nonce, *msg.ReputerValueBundle)
		if err != nil {
			return nil, errorsmod.Wrap(err, "error filtering reputer bundle")
		}
		err = ms.k.UpsertReputerBundle(ctx, topicId, nonce.ReputerNonce.BlockHeight, filteredLossBundle)
		if err != nil {
			return nil, errorsmod.Wrap(err, "error upserting reputer loss bundle")
		}
	} else {
		// find the lowest scored reputer currently in this epoch
		// and see if our score is higher than theirs, if so we can replace them

		// get the lowest reputer score and index from all of the loss bundles this epoch
		lowScore, lowScoreIndex, err := lowestReputerScoreEma(ctx, ms.k, topicId, *existingReputerLossBundles)
		if err != nil {
			return nil, errorsmod.Wrap(err, "error getting low score from all loss bundles")
		}
		// get score of current reputer
		reputerScore, err := ms.k.GetReputerScoreEma(ctx, topicId, msg.ReputerValueBundle.ValueBundle.Reputer)
		if err != nil {
			return nil, errorsmod.Wrap(err, "error getting reputer score ema")
		}
		if lowScore.Score.Gt(reputerScore.Score) && lowScore.Address != msg.ReputerValueBundle.ValueBundle.Reputer {
			sdkCtx.Logger().Debug(
				fmt.Sprintf(
					"Reputer does not meet threshold for active set inclusion, ignoring.\n"+
						" Reputer: %s\nScore Ema %s\nThreshold %s\nTopicId: %d\nNonce: %d",
					msg.ReputerValueBundle.ValueBundle.Reputer,
					reputerScore.Score.String(),
					lowScore.Score.String(),
					topicId,
					nonce.ReputerNonce.BlockHeight))
		} else {
			// limit reputation loss bundles to only those that describe
			// the top inferers and forecasters
			// also remove duplicate loss bundles
			filteredLossBundle, err := filterReputerBundle(ctx, ms.k, topicId, *nonce, *msg.ReputerValueBundle)
			if err != nil {
				return nil, errorsmod.Wrap(err, "error filtering reputer bundle")
			}

			// we are kicking out the lowest scoring forecaster and replacing them with the new forecaster
			err = ms.k.ReplaceReputerValueBundles(
				ctx,
				topicId,
				*nonce.ReputerNonce,
				*existingReputerLossBundles,
				lowScoreIndex,
				*filteredLossBundle)
			if err != nil {
				return nil, errorsmod.Wrap(err, "error replacing reputer value bundles")
			}
		}
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

	pk, err := hex.DecodeString(bundle.Pubkey)
	if err != nil || len(pk) != secp256k1.PubKeySize {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "invalid pubkey")
	}
	pubkey := secp256k1.PubKey(pk)
	pubKeyConvertedToAddress := sdk.AccAddress(pubkey.Address().Bytes()).String()

	if bundle.ValueBundle.Reputer != pubKeyConvertedToAddress {
		return errorsmod.Wrapf(types.ErrUnauthorized, "Reputer does not match pubkey")
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

	src := make([]byte, 0)
	src, _ = bundle.ValueBundle.XXX_Marshal(src, true)
	if !pubkey.VerifySignature(src, bundle.Signature) {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "signature verification failed")
	}

	return nil
}

// gets the lowest reputer score from all of the loss bundles
// no validation is done that the loss bundles are of len > 0
// because by the time this is called, that should be guaranteed
// by the caller
func lowestReputerScoreEma(
	ctx context.Context,
	k keeper.Keeper,
	topicId TopicId,
	lossBundles types.ReputerValueBundles,
) (lowScore types.Score, lowScoreIndex int, err error) {
	lowScoreIndex = 0
	lowScore, err = k.GetReputerScoreEma(ctx, topicId, lossBundles.ReputerValueBundles[0].ValueBundle.Reputer)
	if err != nil {
		return types.Score{}, lowScoreIndex, err
	}
	for index, extLossBundle := range lossBundles.ReputerValueBundles {
		extScore, err := k.GetReputerScoreEma(ctx, topicId, extLossBundle.ValueBundle.Reputer)
		if err != nil {
			return types.Score{}, lowScoreIndex, err
		}
		if lowScore.Score.Gt(extScore.Score) {
			lowScore = extScore
			lowScoreIndex = index
		}
	}
	return lowScore, lowScoreIndex, nil
}

// filter loss bundles
// remove duplicates
// remove any loss bundles that describe reputation for
// any inferer a forecaster that is not in the active set
func filterReputerBundle(
	ctx context.Context,
	k keeper.Keeper,
	topicId TopicId,
	reputerRequestNonce types.ReputerRequestNonce,
	reputerValueBundle types.ReputerValueBundle,
) (*types.ReputerValueBundle, error) {
	// Get the accepted inferers of the associated worker response payload
	inferences, err := k.GetInferencesAtBlock(ctx, topicId, reputerRequestNonce.ReputerNonce.BlockHeight)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "no inferences found at block height")
		} else {
			return nil, err
		}
	}
	activeInferers := make(map[string]struct{})
	for _, inference := range inferences.Inferences {
		activeInferers[inference.Inferer] = struct{}{}
	}

	// Get the accepted forecasters of the associated worker response payload
	forecasts, err := k.GetForecastsAtBlock(ctx, topicId, reputerRequestNonce.ReputerNonce.BlockHeight)
	if err != nil {
		return nil, err
	}
	activeForecasters := make(map[string]struct{})
	for _, forecast := range forecasts.Forecasts {
		activeForecasters[forecast.Forecaster] = struct{}{}
	}

	// Filter out values submitted by unaccepted workers

	acceptedInfererValues := make([]*types.WorkerAttributedValue, 0)
	infererSeen := make(map[string]struct{})
	for _, workerVal := range reputerValueBundle.ValueBundle.InfererValues {
		if _, isActive := activeInferers[workerVal.Worker]; isActive {
			if _, seen := infererSeen[workerVal.Worker]; !seen {
				acceptedInfererValues = append(acceptedInfererValues, workerVal)
				infererSeen[workerVal.Worker] = struct{}{} // Mark as seen => no duplicates
			}
		}
	}

	acceptedForecasterValues := make([]*types.WorkerAttributedValue, 0)
	forecasterSeen := make(map[string]struct{})
	for _, workerVal := range reputerValueBundle.ValueBundle.ForecasterValues {
		if _, isActive := activeForecasters[workerVal.Worker]; isActive {
			if _, seen := forecasterSeen[workerVal.Worker]; !seen {
				acceptedForecasterValues = append(acceptedForecasterValues, workerVal)
				forecasterSeen[workerVal.Worker] = struct{}{} // Mark as seen => no duplicates
			}
		}
	}

	acceptedOneOutInfererValues := make([]*types.WithheldWorkerAttributedValue, 0)
	// If 1 or fewer inferers, there's no one-out inferer data to receive
	if len(acceptedInfererValues) > 1 {
		oneOutInfererSeen := make(map[string]struct{})
		for _, workerVal := range reputerValueBundle.ValueBundle.OneOutInfererValues {
			if _, isActive := activeInferers[workerVal.Worker]; isActive {
				if _, seen := oneOutInfererSeen[workerVal.Worker]; !seen {
					acceptedOneOutInfererValues = append(acceptedOneOutInfererValues, workerVal)
					oneOutInfererSeen[workerVal.Worker] = struct{}{} // Mark as seen => no duplicates
				}
			}
		}
	}

	acceptedOneOutForecasterValues := make([]*types.WithheldWorkerAttributedValue, 0)
	oneOutForecasterSeen := make(map[string]struct{})
	for _, workerVal := range reputerValueBundle.ValueBundle.OneOutForecasterValues {
		if _, isActive := activeForecasters[workerVal.Worker]; isActive {
			if _, seen := oneOutForecasterSeen[workerVal.Worker]; !seen {
				acceptedOneOutForecasterValues = append(acceptedOneOutForecasterValues, workerVal)
				oneOutForecasterSeen[workerVal.Worker] = struct{}{} // Mark as seen => no duplicates
			}
		}
	}

	acceptedOneInForecasterValues := make([]*types.WorkerAttributedValue, 0)
	oneInForecasterSeen := make(map[string]struct{})
	for _, workerVal := range reputerValueBundle.ValueBundle.OneInForecasterValues {
		if _, isActive := activeForecasters[workerVal.Worker]; isActive {
			if _, seen := oneInForecasterSeen[workerVal.Worker]; !seen {
				acceptedOneInForecasterValues = append(acceptedOneInForecasterValues, workerVal)
				oneInForecasterSeen[workerVal.Worker] = struct{}{} // Mark as seen => no duplicates
			}
		}
	}

	acceptedReputerValueBundle := &types.ReputerValueBundle{
		ValueBundle: &types.ValueBundle{
			TopicId:                reputerValueBundle.ValueBundle.TopicId,
			ReputerRequestNonce:    reputerValueBundle.ValueBundle.ReputerRequestNonce,
			Reputer:                reputerValueBundle.ValueBundle.Reputer,
			ExtraData:              reputerValueBundle.ValueBundle.ExtraData,
			InfererValues:          acceptedInfererValues,
			ForecasterValues:       acceptedForecasterValues,
			OneOutInfererValues:    acceptedOneOutInfererValues,
			OneOutForecasterValues: acceptedOneOutForecasterValues,
			OneInForecasterValues:  acceptedOneInForecasterValues,
			NaiveValue:             reputerValueBundle.ValueBundle.NaiveValue,
			CombinedValue:          reputerValueBundle.ValueBundle.CombinedValue,
		},
		Signature: reputerValueBundle.Signature,
	}

	return acceptedReputerValueBundle, nil
}
