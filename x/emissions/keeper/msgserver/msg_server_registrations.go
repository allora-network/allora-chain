package msgserver

import (
	"context"
	"fmt"

	"github.com/allora-network/allora-chain/app/params"
	"github.com/allora-network/allora-chain/x/emissions/types"
	mintTypes "github.com/allora-network/allora-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Registers a new network participant to the network for the first time
func (ms msgServer) Register(ctx context.Context, msg *types.MsgRegister) (*types.MsgRegisterResponse, error) {
	if msg.GetLibP2PKey() == "" {
		return nil, types.ErrLibP2PKeyRequired
	}
	address, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Check if topic exists
	topicExists, err := ms.k.TopicExists(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	}
	if !topicExists {
		return nil, types.ErrTopicDoesNotExist
	}

	hasEnoughBal, fee, _ := ms.CheckAddressHasBalanceForTopicCreationFee(ctx, address)
	if !hasEnoughBal {
		return nil, types.ErrTopicRegistrantNotEnoughDenom
	}

	// Before creating topic, transfer fee amount from creator to ecosystem bucket
	err = ms.k.SendCoinsFromAccountToModule(ctx, address, mintTypes.EcosystemModuleName, sdk.NewCoins(fee))
	if err != nil {
		return nil, err
	}

	nodeInfo := types.OffchainNode{
		NodeAddress:  msg.Sender,
		LibP2PKey:    msg.LibP2PKey,
		MultiAddress: msg.MultiAddress,
	}
	nodeInfo.Owner = msg.Owner
	nodeInfo.NodeId = msg.Owner + "|" + msg.LibP2PKey
	if msg.IsReputer {
		// add node to topicReputers
		// add node to reputers
		err = ms.k.InsertReputer(ctx, msg.TopicId, address, nodeInfo)
		if err != nil {
			return nil, err
		}
	} else {
		if msg.Owner == "" {
			return nil, types.ErrOwnerCannotBeEmpty
		}
		// add node to topicWorkers
		// add node to workers
		err = ms.k.InsertWorker(ctx, msg.TopicId, address, nodeInfo)
		if err != nil {
			return nil, err
		}
	}

	return &types.MsgRegisterResponse{
		Success: true,
		Message: "Node successfully registered",
	}, nil
}

// Remove registration from a topic
func (ms msgServer) RemoveRegistration(ctx context.Context, msg *types.MsgRemoveRegistration) (*types.MsgRemoveRegistrationResponse, error) {
	// Check if topic exists
	topicExists, err := ms.k.TopicExists(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	}
	if !topicExists {
		return nil, types.ErrTopicDoesNotExist
	}

	// Check if the address is registered in the specified topic
	address, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Proceed based on whether requester is removing their reputer or worker registration
	if msg.IsReputer {
		// Remove the reputer registration from the topic
		err = ms.k.RemoveReputer(ctx, msg.TopicId, address)
		if err != nil {
			return nil, err
		}

		isRegisteredInTopic, err := ms.k.IsReputerRegisteredInTopic(ctx, msg.TopicId, address)
		if err != nil {
			return nil, err
		}

		if !isRegisteredInTopic {
			return nil, types.ErrAddressIsNotRegisteredInThisTopic
		}
	} else {
		// Remove the worker registration from the topic
		err = ms.k.RemoveWorker(ctx, msg.TopicId, address)
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

func (ms msgServer) CheckBalanceForRegistration(ctx context.Context, address sdk.AccAddress) (bool, sdk.Coin, error) {
	amountInt, err := ms.k.GetParamsRegistrationFee(ctx)
	if err != nil {
		return false, sdk.Coin{}, err
	}
	fee := sdk.NewCoin(params.DefaultBondDenom, amountInt)
	balance := ms.k.BankKeeper().GetBalance(ctx, address, fee.Denom)
	return balance.IsGTE(fee), fee, nil
}
