package msgserver

import (
	"context"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/allora-network/allora-chain/app/params"
	"github.com/allora-network/allora-chain/x/emissions/metrics"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// Register registers a new network participant to the network as worker or reputer
func (ms msgServer) Register(ctx context.Context, msg *types.RegisterRequest) (_ *types.RegisterResponse, err error) {
	defer metrics.RecordMetrics("Register", time.Now(), &err)

	err = msg.Validate()
	if err != nil {
		return nil, err
	}

	topicExists, err := ms.tk.TopicExists(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	}
	if !topicExists {
		return nil, types.ErrTopicDoesNotExist
	}

	if msg.IsReputer {
		isRegistered, err := ms.rlk.IsReputerRegisteredInTopic(ctx, msg.TopicId, msg.Sender)
		if err != nil {
			return nil, err
		}
		if isRegistered {
			return nil, errorsmod.Wrapf(types.ErrAddressAlreadyRegisteredInATopic, "reputer is already registered in this topic")
		}
	} else {
		isRegistered, err := ms.wk.IsWorkerRegisteredInTopic(ctx, msg.TopicId, msg.Sender)
		if err != nil {
			return nil, err
		}
		if isRegistered {
			return nil, errorsmod.Wrapf(types.ErrAddressAlreadyRegisteredInATopic, "worker is already registered in this topic")
		}
	}

	params, err := ms.pk.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	err = sendEffectiveRevenueActivateTopicIfWeightSufficient(ctx, ms, msg.Sender, msg.TopicId, params.RegistrationFee)
	if err != nil {
		return nil, err
	}

	nodeInfo := types.OffchainNode{
		NodeAddress: msg.Sender,
		Owner:       msg.Owner,
	}

	if msg.IsReputer {
		err = ms.rlk.InsertReputer(ctx, msg.TopicId, msg.Sender, nodeInfo)
		if err != nil {
			return nil, err
		}
		types.EmitNewReputerRegisteredEvent(ctx, msg.TopicId, msg.Sender, msg.Owner)
	} else {
		err = ms.wk.InsertWorker(ctx, msg.TopicId, msg.Sender, nodeInfo)
		if err != nil {
			return nil, err
		}
		types.EmitNewWorkerRegisteredEvent(ctx, msg.TopicId, msg.Sender, msg.Owner)
	}

	return &types.RegisterResponse{}, nil // nolint:exhaustruct // due to deprecated fields
}

// RemoveRegistration removes registration from a topic for worker or reputer
func (ms msgServer) RemoveRegistration(ctx context.Context, msg *types.RemoveRegistrationRequest) (_ *types.RemoveRegistrationResponse, err error) {
	defer metrics.RecordMetrics("RemoveRegistration", time.Now(), &err)

	err = msg.Validate()
	if err != nil {
		return nil, err
	}
	// Check if topic exists
	topicExists, err := ms.tk.TopicExists(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	}
	if !topicExists {
		return nil, types.ErrTopicDoesNotExist
	}

	// Proceed based on whether requester is removing their reputer or worker registration
	if msg.IsReputer {
		isRegisteredInTopic, err := ms.rlk.IsReputerRegisteredInTopic(ctx, msg.TopicId, msg.Sender)
		if err != nil {
			return nil, err
		}

		if !isRegisteredInTopic {
			return nil, types.ErrAddressIsNotRegisteredInThisTopic
		}

		// Remove the reputer registration from the topic
		err = ms.rlk.RemoveReputer(ctx, msg.TopicId, msg.Sender)
		if err != nil {
			return nil, err
		}
		types.EmitNewReputerUnregisteredEvent(ctx, msg.TopicId, msg.Sender)
	} else {
		isRegisteredInTopic, err := ms.wk.IsWorkerRegisteredInTopic(ctx, msg.TopicId, msg.Sender)
		if err != nil {
			return nil, err
		}

		if !isRegisteredInTopic {
			return nil, types.ErrAddressIsNotRegisteredInThisTopic
		}

		// Remove the worker registration from the topic
		err = ms.wk.RemoveWorker(ctx, msg.TopicId, msg.Sender)
		if err != nil {
			return nil, err
		}
		types.EmitNewWorkerUnregisteredEvent(ctx, msg.TopicId, msg.Sender)
	}

	// Return a successful response
	return &types.RemoveRegistrationResponse{}, nil // nolint:exhaustruct // due to deprecated fields
}

// CheckBalanceForRegistration checks if the account has enough balance to register
func (ms msgServer) CheckBalanceForRegistration(ctx context.Context, address string) (success bool, fee sdk.Coin, err error) {
	defer metrics.RecordMetrics("CheckBalanceForRegistration", time.Now(), &err)

	moduleParams, err := ms.pk.GetParams(ctx)
	if err != nil {
		return false, sdk.Coin{}, err
	}
	fee = sdk.NewCoin(params.DefaultBondDenom, moduleParams.RegistrationFee)
	accAddress, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return false, fee, err
	}
	balance := ms.bk.GetBankBalance(ctx, accAddress, fee.Denom)
	success = balance.IsGTE(fee)
	return success, fee, err
}
