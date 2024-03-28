package msgserver

import (
	"context"
	"fmt"

	cosmoserrors "cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Registers a new network participant to the network for the first time
func (ms msgServer) Register(ctx context.Context, msg *types.MsgRegister) (*types.MsgRegisterResponse, error) {
	if msg.GetLibP2PKey() == "" {
		return nil, types.ErrLibP2PKeyRequired
	}
	address, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}
	// require funds to be at least greater than the minimum stake
	requiredMinimumStake, err := ms.k.GetParamsRequiredMinimumStake(ctx)
	if err != nil {
		return nil, err
	}
	// check user existing stake
	addressExistingStake := cosmosMath.NewUint(0)
	for _, topicId := range msg.TopicIds {
		addressExistingStakeInTopic, err := ms.k.GetStakeOnTopicFromReputer(ctx, topicId, address)
		if err != nil {
			return nil, err
		}
		addressExistingStake = addressExistingStake.Add(addressExistingStakeInTopic)
	}
	// check if the user has enough funds to register
	totalAddressStake := addressExistingStake.Add(msg.GetInitialStake())
	if totalAddressStake.LT(requiredMinimumStake) {
		return nil, cosmoserrors.Wrapf(types.ErrInsufficientStakeToRegister,
			"required minimum stake: %s, existing address stake: %s, initial stake: %s",
			requiredMinimumStake, addressExistingStake, msg.GetInitialStake())
	}
	// check if topics exists and if address is already registered in any of them
	registeredTopicIds, err := ms.k.GetRegisteredTopicIdsByAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	if len(registeredTopicIds) > 0 {
		return nil, types.ErrAddressAlreadyRegisteredInATopic
	}

	for _, topicId := range msg.TopicIds {
		// check if topic exists
		topicExists, err := ms.k.TopicExists(ctx, topicId)
		if !topicExists {
			return nil, types.ErrTopicDoesNotExist
		} else if err != nil {
			return nil, err
		}
	}

	// move the tokens from the creator to the module account
	// then add the stake to the total, topicTotal, and 3 staking tracking maps
	moveFundsAddStake(ctx, ms, address, msg)

	nodeInfo := types.OffchainNode{
		NodeAddress:  msg.Creator,
		LibP2PKey:    msg.LibP2PKey,
		MultiAddress: msg.MultiAddress,
	}
	if msg.IsReputer {
		// add node to topicReputers
		// add node to reputers
		err = ms.k.InsertReputer(ctx, msg.TopicIds, address, nodeInfo)
		if err != nil {
			return nil, err
		}
	} else {
		if msg.Owner == "" {
			return nil, types.ErrOwnerCannotBeEmpty
		}
		nodeInfo.Owner = msg.Owner
		nodeInfo.NodeId = msg.Owner + "|" + msg.LibP2PKey

		// add node to topicWorkers
		// add node to workers
		err = ms.k.InsertWorker(ctx, msg.TopicIds, address, nodeInfo)
		if err != nil {
			return nil, err
		}
	}

	return &types.MsgRegisterResponse{
		Success: true,
		Message: "Node successfully registered",
	}, nil
}

// Add additional topics after initial reputer or worker registration
func (ms msgServer) AddNewRegistration(ctx context.Context, msg *types.MsgAddNewRegistration) (*types.MsgAddNewRegistrationResponse, error) {
	// check if topics exists and if address is already registered in any of them
	address, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}
	// check if topic exists
	topicExists, err := ms.k.TopicExists(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	} else if !topicExists {
		return nil, types.ErrTopicDoesNotExist
	}
	registeredTopicIds, err := ms.k.GetRegisteredTopicIdsByAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	if len(registeredTopicIds) == 0 {
		return nil, types.ErrAddressIsNotRegisteredInAnyTopic
	}

	// copy overall staking power of the wallet to the topic stake
	totalAddressStake, err := ms.k.GetStakeOnTopicFromReputer(ctx, msg.TopicId, address)
	if err != nil {
		return nil, err
	}

	// add to topic stake
	err = ms.k.AddStake(ctx, msg.GetTopicId(), address, totalAddressStake)
	if err != nil {
		return nil, err
	}

	nodeInfo := types.OffchainNode{
		NodeAddress:  msg.Creator,
		LibP2PKey:    msg.LibP2PKey,
		MultiAddress: msg.MultiAddress,
	}

	if msg.IsReputer {
		// get topics where the users is registered as reputer
		reputerRegisteredTopicIds, err := ms.k.GetRegisteredTopicIdByReputerAddress(ctx, address)
		if err != nil {
			return nil, err
		}
		for _, topicIdRegistered := range reputerRegisteredTopicIds {
			if topicIdRegistered == msg.TopicId {
				return nil, types.ErrReputerAlreadyRegisteredInTopic
			}
		}

		// add node to topicReputers
		err = ms.k.InsertReputer(ctx, []uint64{msg.TopicId}, address, nodeInfo)
		if err != nil {
			return nil, err
		}
	} else {
		// get topics where the users is registered as worker
		reputerRegisteredTopicIds, err := ms.k.GetRegisteredTopicIdsByWorkerAddress(ctx, address)
		if err != nil {
			return nil, err
		}
		for _, topicIdRegistered := range reputerRegisteredTopicIds {
			if topicIdRegistered == msg.TopicId {
				return nil, types.ErrReputerAlreadyRegisteredInTopic
			}
		}

		// add node to topicWorkers
		err = ms.k.InsertWorker(ctx, []uint64{msg.TopicId}, address, nodeInfo)
		if err != nil {
			return nil, err
		}
	}

	return &types.MsgAddNewRegistrationResponse{
		Success: true,
		Message: fmt.Sprintf("Node successfully registered in topic %d", msg.TopicId),
	}, nil
}

// Remove registration from a topic
func (ms msgServer) RemoveRegistration(ctx context.Context, msg *types.MsgRemoveRegistration) (*types.MsgRemoveRegistrationResponse, error) {
	// check if topic exists
	topicExists, err := ms.k.TopicExists(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	} else if !topicExists {
		return nil, types.ErrTopicDoesNotExist
	}

	// Check if the address is registered in the specified topic
	address, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}
	registeredTopicIds, err := ms.k.GetRegisteredTopicIdsByAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	isRegisteredInTopic := false
	for _, topicId := range registeredTopicIds {
		if topicId == msg.TopicId {
			isRegisteredInTopic = true
			break
		}
	}
	if !isRegisteredInTopic {
		return nil, types.ErrAddressIsNotRegisteredInThisTopic
	}

	// remove overall staking power of the wallet to the topic stake
	totalAddressStake, err := ms.k.GetStakeOnTopicFromReputer(ctx, msg.TopicId, address)
	if err != nil {
		return nil, err
	}

	// remove from topic stake
	err = ms.k.RemoveStake(ctx, msg.TopicId, address, totalAddressStake)
	if err != nil {
		return nil, err
	}

	// Proceed based on whether the address is registered as a reputer or worker
	if msg.IsReputer {
		// Remove the reputer registration from the topic
		err = ms.k.RemoveReputer(ctx, msg.TopicId, address)
		if err != nil {
			return nil, err
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

///
/// PRIVATE
///

// checks if a node is registered in the system and if it is,
// returns whether said node is a reputer or a worker
func checkNodeRegistered(ctx context.Context, ms msgServer, node sdk.AccAddress) error {

	reputerRegisteredTopicIds, err := ms.k.GetRegisteredTopicIdByReputerAddress(ctx, node)
	if err != nil {
		return err
	}
	if len(reputerRegisteredTopicIds) > 0 {
		return nil
	}
	workerRegisteredTopicIds, err := ms.k.GetRegisteredTopicIdsByWorkerAddress(ctx, node)
	if err != nil {
		return err
	}
	if len(workerRegisteredTopicIds) > 0 {
		return nil
	}
	return types.ErrAddressNotRegistered
}
