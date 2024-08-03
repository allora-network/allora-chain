package msgserver

import (
	"context"
	"fmt"

	"github.com/allora-network/allora-chain/app/params"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Registers a new network participant to the network for the first time for worker or reputer
func (ms msgServer) Register(ctx context.Context, msg *types.MsgRegister) (*types.MsgRegisterResponse, error) {
	if err := msg.Validate(); err != nil {
		return nil, err
	}

	topicExists, err := ms.k.TopicExists(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	}
	if !topicExists {
		return nil, types.ErrTopicDoesNotExist
	}

	params, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	err = sendEffectiveRevenueActivateTopicIfWeightSufficient(ctx, ms, msg.Sender, msg.TopicId, params.RegistrationFee, "register")
	if err != nil {
		return nil, err
	}

	nodeInfo := types.OffchainNode{
		NodeAddress: msg.Sender,
		Owner:       msg.Owner,
	}

	if msg.IsReputer {
		err = ms.k.InsertReputer(ctx, msg.TopicId, msg.Sender, nodeInfo)
		if err != nil {
			return nil, err
		}
	} else {
		err = ms.k.InsertWorker(ctx, msg.TopicId, msg.Sender, nodeInfo)
		if err != nil {
			return nil, err
		}
	}

	return &types.MsgRegisterResponse{
		Success: true,
		Message: "Node successfully registered",
	}, nil
}

// Remove registration from a topic for worker or reputer
func (ms msgServer) RemoveRegistration(ctx context.Context, msg *types.MsgRemoveRegistration) (*types.MsgRemoveRegistrationResponse, error) {
	// Check if topic exists
	topicExists, err := ms.k.TopicExists(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	}
	if !topicExists {
		return nil, types.ErrTopicDoesNotExist
	}

	// Proceed based on whether requester is removing their reputer or worker registration
	if msg.IsReputer {
		isRegisteredInTopic, err := ms.k.IsReputerRegisteredInTopic(ctx, msg.TopicId, msg.Sender)
		if err != nil {
			return nil, err
		}

		if !isRegisteredInTopic {
			return nil, types.ErrAddressIsNotRegisteredInThisTopic
		}

		// Remove the reputer registration from the topic
		err = ms.k.RemoveReputer(ctx, msg.TopicId, msg.Sender)
		if err != nil {
			return nil, err
		}

	} else {
		isRegisteredInTopic, err := ms.k.IsWorkerRegisteredInTopic(ctx, msg.TopicId, msg.Sender)
		if err != nil {
			return nil, err
		}

		if !isRegisteredInTopic {
			return nil, types.ErrAddressIsNotRegisteredInThisTopic
		}

		// Remove the worker registration from the topic
		err = ms.k.RemoveWorker(ctx, msg.TopicId, msg.Sender)
		if err != nil {
			return nil, err
		}
	}

	// Return a successful response
	return &types.MsgRemoveRegistrationResponse{
		Success: true,
		Message: fmt.Sprintf("Node successfully removed from topic %d", msg.TopicId),
	}, nil
}

func (ms msgServer) CheckBalanceForRegistration(ctx context.Context, address string) (bool, sdk.Coin, error) {
	moduleParams, err := ms.k.GetParams(ctx)
	if err != nil {
		return false, sdk.Coin{}, err
	}
	fee := sdk.NewCoin(params.DefaultBondDenom, moduleParams.RegistrationFee)
	accAddress, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return false, fee, err
	}
	balance := ms.k.GetBankBalance(ctx, accAddress, fee.Denom)
	return balance.IsGTE(fee), fee, nil
}
