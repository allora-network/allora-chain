package msgserver

import (
	"context"

	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (ms msgServer) IsTopicMempoolFull(ctx context.Context, topicId uint64) (bool, error) {
	maxMempoolSize, err := ms.k.GetParamsMaxRequestsPerTopic(ctx)
	if err != nil {
		return false, err
	}
	mempoolSize, err := ms.k.GetTopicMempoolRequestCount(ctx, topicId)
	if err != nil {
		return false, err
	}
	return mempoolSize >= maxMempoolSize, nil
}

func (ms msgServer) RequestInference(ctx context.Context, msg *types.MsgRequestInference) (*types.MsgRequestInferenceResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	request := types.CreateNewInferenceRequestFromListItem(msg.Sender, msg.Request)
	// 1. check the topic is valid
	topicExists, err := ms.k.TopicExists(ctx, request.TopicId)
	if err != nil {
		return nil, err
	}
	if !topicExists {
		return nil, types.ErrInvalidTopicId
	}
	requestId, err := request.GetRequestId()
	if err != nil {
		return nil, err
	}
	// 2. check the request isn't already in the mempool
	requestExists, err := ms.k.IsRequestInMempool(ctx, requestId)
	if err != nil {
		return nil, err
	}
	if requestExists {
		return nil, types.ErrInferenceRequestAlreadyInMempool
	}
	// Check if the topic mempool is full
	topicMempoolFull, err := ms.IsTopicMempoolFull(ctx, request.TopicId)
	if err != nil {
		return nil, err
	}
	if topicMempoolFull {
		return nil, types.ErrTopicMempoolAtCapacity
	}
	// 3. Check the BidAmount is greater than the price per request
	if request.BidAmount.LT(request.MaxPricePerInference) {
		return nil, types.ErrInferenceRequestBidAmountLessThanPrice
	}
	// 4. Check the block valid until is in the future
	currentBlock := sdkCtx.BlockHeight()
	if request.BlockValidUntil < currentBlock {
		return nil, types.ErrInferenceRequestBlockValidUntilInPast
	}
	// 5. Check the block validity is no more than the maximum allowed time in the future
	maxInferenceRequestValidity, err := ms.k.GetParamsMaxInferenceRequestValidity(ctx)
	if err != nil {
		return nil, err
	}
	if request.BlockValidUntil > currentBlock+maxInferenceRequestValidity {
		return nil, types.ErrInferenceRequestBlockValidUntilTooFarInFuture
	}
	if request.Cadence != 0 {
		// 6. Check the cadence is either 0, or greater than the minimum fastest cadence allowed
		minFastestAllowedCadence, err := ms.k.GetParamsMinEpochLength(ctx)
		if err != nil {
			return nil, err
		}
		if request.Cadence < minFastestAllowedCadence {
			return nil, types.ErrInferenceRequestCadenceTooFast
		}
		// 7. Check the cadence is no more than the maximum allowed slowest cadence
		maxSlowestAllowedCadence, err := ms.k.GetParamsMaxRequestCadence(ctx)
		if err != nil {
			return nil, err
		}
		if request.Cadence > maxSlowestAllowedCadence {
			return nil, types.ErrInferenceRequestCadenceTooSlow
		}
	}
	// 8. Check the cadence is not greater than the block valid until
	if currentBlock+request.Cadence > request.BlockValidUntil {
		return nil, types.ErrInferenceRequestWillNeverBeScheduled
	}
	MinRequestUnmetDemand, err := ms.k.GetParamsMinRequestUnmetDemand(ctx)
	if err != nil {
		return nil, err
	}
	// Check that the request isn't spam by checking that the amount of funds it bids is greater than a global minimum demand per request
	if request.BidAmount.LT(MinRequestUnmetDemand) {
		return nil, types.ErrInferenceRequestBidAmountTooLow
	}
	// 9. Check sender has funds to pay for the inference request
	// bank module does this for us in module SendCoins / subUnlockedCoins so we don't need to check
	// 10. Send funds
	senderAddr, err := sdk.AccAddressFromBech32(request.Sender)
	if err != nil {
		return nil, err
	}
	amountInt := cosmosMath.NewIntFromBigInt(request.BidAmount.BigInt())
	coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amountInt))
	err = ms.k.SendCoinsFromAccountToModule(ctx, senderAddr, types.AlloraRequestsAccountName, coins)
	if err != nil {
		return nil, err
	}
	// 11. record the number of tokens sent to the module account
	err = ms.k.SetRequestDemand(ctx, requestId, request.BidAmount)
	if err != nil {
		return nil, err
	}
	// 12. Write request state into the mempool state
	request.BlockLastChecked = currentBlock
	unmetDemand, err := ms.k.AddToMempool(ctx, *request)
	if err != nil {
		return nil, err
	}
	// 13. Activate topic if meet demand
	isActivated, err := ms.k.IsTopicActive(ctx, request.TopicId)
	if err != nil {
		return nil, err
	}
	if !isActivated {
		minTopicUnmentDemand, err := ms.k.GetParamsMinTopicUnmetDemand(ctx)
		if err != nil {
			return nil, err
		}
		minTopicUnmetDemandUint := cosmosMath.NewUintFromString(minTopicUnmentDemand.String())

		if unmetDemand.GTE(minTopicUnmetDemandUint) {
			_ = ms.k.ActivateTopic(ctx, request.TopicId)
		}
	}
	return &types.MsgRequestInferenceResponse{RequestId: requestId}, nil
}
