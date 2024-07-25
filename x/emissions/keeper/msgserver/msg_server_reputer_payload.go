package msgserver

import (
	"context"
	"github.com/allora-network/allora-chain/x/emissions/types"
	mintTypes "github.com/allora-network/allora-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// A tx function that accepts a individual loss and possibly returns an error
func (ms msgServer) InsertReputerPayload(ctx context.Context, msg *types.MsgInsertReputerPayload) (*types.MsgInsertReputerPayloadResponse, error) {
	err := checkInputLength(ctx, ms, msg)
	if err != nil {
		return nil, err
	}

	// Check if the topic exists
	topicExists, err := ms.k.TopicExists(ctx, msg.TopicId)
	if err != nil || !topicExists {
		return nil, types.ErrInvalidTopicId
	}

	// Check if the window time has passed: if blockheight > nonce.BlockHeight + topic.WorkerSubmissionWindow
	blockHeight := sdk.UnwrapSDKContext(ctx).BlockHeight()
	nonce := msg.ReputerRequestNonce

	topicId := msg.TopicId
	topic, err := ms.k.GetTopic(ctx, msg.TopicId)
	if err != nil {
		return nil, types.ErrInvalidTopicId
	}

	if blockHeight > topic.EpochLastEnded+topic.GroundTruthLag {
		return nil, types.ErrWorkerNonceWindowNotAvailable
	}

	if err := msg.ReputerValueBundles.Validate(); err != nil {
		return nil, types.ErrInvalidWorkerData
	}

	hasEnoughBal, fee, err := ms.CheckBalanceForSendingDataFee(ctx, msg.Sender)
	if err != nil {
		return nil, err
	}
	if !hasEnoughBal {
		return nil, types.ErrDataSenderNotEnoughDenom
	}

	// Before creating topic, transfer fee amount from creator to ecosystem bucket
	err = ms.k.SendCoinsFromAccountToModule(ctx, msg.Sender, mintTypes.EcosystemModuleName, sdk.NewCoins(fee))
	if err != nil {
		return nil, err
	}

	// Get all inferences from this topic, nonce
	lossBundles, err := ms.k.GetReputerLossBundlesAtBlock(ctx, topicId, nonce.ReputerNonce.BlockHeight)
	if err != nil {
		return nil, err
	}
	// Append this individual inference to all inferences
	lossValue := msg.ReputerValueBundles

	inferencesToInsert := types.ReputerValueBundles{
		ReputerValueBundles: append(lossBundles.ReputerValueBundles, lossValue),
	}
	err = ms.k.InsertReputerLossBundlesAtBlock(ctx, topicId, nonce.ReputerNonce.BlockHeight, inferencesToInsert)
	if err != nil {
		return nil, err
	}

	return &types.MsgInsertReputerPayloadResponse{}, nil
}
