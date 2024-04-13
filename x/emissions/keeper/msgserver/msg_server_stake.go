package msgserver

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Function for reputers to call to add stake to an existing stake position.
func (ms msgServer) AddStake(ctx context.Context, msg *types.MsgAddStake) (*types.MsgAddStakeResponse, error) {
	// 1. check the sender is registered
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	targetAddr, err := sdk.AccAddressFromBech32(msg.StakeTarget)
	if err != nil {
		return nil, err
	}
	err = checkNodeRegistered(ctx, ms, senderAddr)
	if err != nil {
		return nil, err
	}

	// 2. check the target exists and is registered
	isReputerRegistered, err := ms.k.IsReputerRegisteredInTopic(ctx, msg.TopicId, targetAddr)
	if err != nil {
		return nil, err
	}
	if !isReputerRegistered {
		return nil, types.ErrReputerNotRegistered
	}

	// 3. check the sender has enough funds to add the stake
	// bank module does this for us in module SendCoins / subUnlockedCoins so we don't need to check
	// 4. send the funds
	amountInt := cosmosMath.NewIntFromBigInt(msg.Amount.BigInt())
	coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amountInt))
	ms.k.SendCoinsFromAccountToModule(ctx, senderAddr, types.AlloraStakingAccountName, coins)

	// 5. get target topics Registerd
	TopicIds, err := ms.k.GetRegisteredTopicIdsByAddress(ctx, targetAddr)
	if err != nil {
		return nil, err
	}

	// 6. update the stake data structures, spread the stake across all topics evenly
	amountToStake := cosmosMath.NewUintFromBigInt(msg.Amount.BigInt()).Quo(cosmosMath.NewUint(uint64(len(TopicIds))))
	for _, topicId := range TopicIds {
		err = ms.k.AddStake(ctx, topicId, senderAddr, amountToStake)
		if err != nil {
			return nil, err
		}
	}

	return &types.MsgAddStakeResponse{}, nil
}

// StartRemoveStake kicks off a stake removal process. Stake Removals are placed into a delayed queue.
// once the withdrawal delay has passed then ConfirmRemoveStake can be called to remove the stake.
// if a stake removal is not confirmed within a certain time period, the stake removal becomes invalid
// and one must start the stake removal process again and wait the delay again.
func (ms msgServer) StartRemoveStake(ctx context.Context, msg *types.MsgStartRemoveStake) (*types.MsgStartRemoveStakeResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// Check the sender is registered
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	stakeRemoval := types.StakeRemoval{
		BlockRemovalStarted: sdkCtx.BlockHeight(),
		Placements:          make([]*types.StakePlacement, 0),
	}
	for _, stakePlacement := range msg.PlacementsRemove {
		// Check the sender has enough stake already placed on the topic to remove the stake
		stakePlaced, err := ms.k.GetStakeOnTopicFromReputer(ctx, stakePlacement.TopicId, senderAddr)
		if err != nil {
			return nil, err
		}
		if stakePlaced.LT(stakePlacement.Amount) {
			return nil, types.ErrInsufficientStakeToRemove
		}

		// If user is still registered in the topic check that the stake is greater than the minimum required
		requiredMinimumStake, err := ms.k.GetParamsRequiredMinimumStake(ctx)
		if err != nil {
			return nil, err
		}
		if stakePlaced.Sub(stakePlacement.Amount).LT(requiredMinimumStake) {
			return nil, types.ErrInsufficientStakeAfterRemoval
		}

		// Push to the stake removal object
		stakeRemoval.Placements = append(stakeRemoval.Placements, &types.StakePlacement{
			TopicId: stakePlacement.TopicId,
			Amount:  stakePlacement.Amount,
		})
	}
	// If no errors have occured and the removal is valid, add the stake removal to the delayed queue
	err = ms.k.SetStakeRemovalQueueForAddress(ctx, senderAddr, stakeRemoval)
	if err != nil {
		return nil, err
	}
	return &types.MsgStartRemoveStakeResponse{}, nil
}

// Function for reputers or workers to call to remove stake from an existing stake position.
func (ms msgServer) ConfirmRemoveStake(ctx context.Context, msg *types.MsgConfirmRemoveStake) (*types.MsgConfirmRemoveStakeResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// pull the stake removal from the delayed queue
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	stakeRemoval, err := ms.k.GetStakeRemovalQueueByAddress(ctx, senderAddr)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, types.ErrConfirmRemoveStakeNoRemovalStarted
		}
		return nil, err
	}
	// check the timestamp is valid
	currentBlock := sdkCtx.BlockHeight()
	if stakeRemoval.BlockRemovalStarted > currentBlock {
		return nil, types.ErrConfirmRemoveStakeTooEarly
	}
	delayWindow, err := ms.k.GetParamsRemoveStakeDelayWindow(ctx)
	if err != nil {
		return nil, err
	}
	if stakeRemoval.BlockRemovalStarted+delayWindow < currentBlock {
		return nil, types.ErrConfirmRemoveStakeTooLate
	}
	// skip checking all the data is valid
	// the data should be valid because it was checked when the stake removal was started
	// send the money
	for _, stakePlacement := range stakeRemoval.Placements {
		// 5. check the module has enough funds to send back to the sender
		// bank module does this for us in module SendCoins / subUnlockedCoins so we don't need to check
		// 6. send the funds
		amountInt := cosmosMath.NewIntFromBigInt(stakePlacement.Amount.BigInt())
		coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amountInt))
		ms.k.SendCoinsFromModuleToAccount(ctx, types.AlloraStakingAccountName, senderAddr, coins)

		// 7. update the stake data structures
		err = ms.k.RemoveStake(ctx, stakePlacement.TopicId, senderAddr, stakePlacement.Amount)
		if err != nil {
			return nil, err
		}
	}
	return &types.MsgConfirmRemoveStakeResponse{}, nil
}

// StartRemoveAllStake kicks off a stake removal process. Stake Removals are placed into a delayed queue.
// once the withdrawal delay has passed then ConfirmRemoveStake can be called to remove the stake.
// if a stake removal is not confirmed within a certain time period, the stake removal becomes invalid
// and one must start the stake removal process again and wait the delay again.
// RemoveAllStake is just a convenience wrapper around StartRemoveStake.
func (ms msgServer) StartRemoveAllStake(ctx context.Context, msg *types.MsgStartRemoveAllStake) (*types.MsgStartRemoveAllStakeResponse, error) {
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	stakePlacements, err := ms.k.GetStakePlacementsByReputer(ctx, senderAddr)
	if err != nil {
		return nil, err
	}
	msgRemoveStake := &types.MsgStartRemoveStake{
		Sender:           msg.Sender,
		PlacementsRemove: make([]*types.StakePlacement, 0),
	}
	for _, stakePlacement := range stakePlacements {
		msgRemoveStake.PlacementsRemove = append(msgRemoveStake.PlacementsRemove, &types.StakePlacement{
			TopicId: stakePlacement.TopicId,
			Amount:  stakePlacement.Amount,
		})
	}
	_, err = ms.StartRemoveStake(ctx, msgRemoveStake)
	if err != nil {
		return nil, err
	}
	return &types.MsgStartRemoveAllStakeResponse{}, nil
}

// Delegates a stake to a reputer. Sender need not be registered to delegate stake.
func (ms msgServer) DelegateStake(ctx context.Context, msg *types.MsgDelegateStake) (*types.MsgDelegateStakeResponse, error) {
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	// Check the target reputer exists and is registered
	targetAddr, err := sdk.AccAddressFromBech32(msg.Reputer)
	if err != nil {
		return nil, err
	}
	isRegistered, err := ms.k.IsReputerRegisteredInTopic(ctx, msg.TopicId, sdk.AccAddress(targetAddr))
	if err != nil {
		return nil, err
	}
	if !isRegistered {
		return nil, types.ErrReputerNotRegistered
	}

	// Check the sender has enough funds to delegate the stake
	// bank module does this for us in module SendCoins / subUnlockedCoins so we don't need to check here
	// Send the funds
	amountInt := cosmosMath.NewIntFromBigInt(msg.Amount.BigInt())
	coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amountInt))
	ms.k.SendCoinsFromAccountToModule(ctx, senderAddr, types.AlloraStakingAccountName, coins)

	// Update the stake data structures
	err = ms.k.AddStake(ctx, msg.TopicId, targetAddr, msg.Amount)
	if err != nil {
		return nil, err
	}

	err = ms.k.AddDelegatedStake(ctx, msg.TopicId, senderAddr, targetAddr, msg.Amount)
	if err != nil {
		return nil, err
	}

	return &types.MsgDelegateStakeResponse{}, nil
}

// StartRemoveDelegatedStake kicks off a stake removal process. Stake Removals are placed into a delayed queue.
// once the withdrawal delay has passed then ConfirmRemoveStake can be called to remove the stake.
// if a stake removal is not confirmed within a certain time period, the stake removal becomes invalid
// and one must start the stake removal process again and wait the delay again.
func (ms msgServer) StartRemoveDelegatedStake(ctx context.Context, msg *types.MsgStartRemoveDelegatedStake) (*types.MsgStartRemoveDelegatedStakeResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// Check the sender is registered
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	stakeRemoval := types.DelegatedStakeRemoval{
		BlockRemovalStarted: sdkCtx.BlockHeight(),
		Placements:          make([]*types.DelegatedStakePlacement, 0),
	}

	for _, stakeAntiPlacement := range msg.PlacementsToRemove {
		// Check the sender has enough stake already placed on the topic to remove the stake
		stakePlaced, err := ms.k.GetStakeOnTopicFromReputer(ctx, stakeAntiPlacement.TopicId, senderAddr)
		if err != nil {
			return nil, err
		}
		if stakePlaced.LT(stakeAntiPlacement.Amount) {
			return nil, types.ErrInsufficientStakeToRemove
		}

		// Push to the stake removal object
		stakeRemoval.Placements = append(stakeRemoval.Placements, &types.DelegatedStakePlacement{
			TopicId: stakeAntiPlacement.TopicId,
			Reputer: stakeAntiPlacement.Reputer,
			Amount:  stakeAntiPlacement.Amount,
		})

		// If no errors have occured and the removal is valid, add the stake removal to the delayed queue
		err = ms.k.SetDelegatedStakeRemovalQueueForAddress(ctx, senderAddr, stakeRemoval)
		if err != nil {
			return nil, err
		}
	}
	return &types.MsgStartRemoveDelegatedStakeResponse{}, nil
}

// Function for delegators to call to remove stake from an existing delegated stake position.
func (ms msgServer) ConfirmRemoveDelegatedStake(ctx context.Context, msg *types.MsgConfirmDelegatedRemoveStake) (*types.MsgConfirmRemoveDelegatedStakeResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// Pull the stake removal from the delayed queue
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	stakeRemoval, err := ms.k.GetDelegatedStakeRemovalQueueByAddress(ctx, senderAddr)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, types.ErrConfirmRemoveStakeNoRemovalStarted
		}
		return nil, err
	}
	// check the block it should start is valid
	currentBlock := sdkCtx.BlockHeight()
	if stakeRemoval.BlockRemovalStarted > currentBlock {
		return nil, types.ErrConfirmRemoveStakeTooEarly
	}
	delayWindow, err := ms.k.GetParamsRemoveStakeDelayWindow(ctx)
	if err != nil {
		return nil, err
	}
	if stakeRemoval.BlockRemovalStarted+delayWindow < currentBlock {
		return nil, types.ErrConfirmRemoveStakeTooLate
	}
	// Skip checking all the data is valid
	// the data should be valid because it was checked when the stake removal was started
	// send the money
	for _, stakePlacement := range stakeRemoval.Placements {
		// Check the module has enough funds to send back to the sender
		// bank module does this for us in module SendCoins / subUnlockedCoins so we don't need to check
		// Send the funds
		amountInt := cosmosMath.NewIntFromBigInt(stakePlacement.Amount.BigInt())
		coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amountInt))
		ms.k.SendCoinsFromModuleToAccount(ctx, types.AlloraStakingAccountName, senderAddr, coins)

		// Update the stake data structures
		err = ms.k.RemoveDelegatedStake(ctx, stakePlacement.TopicId, senderAddr, sdk.AccAddress(stakePlacement.Reputer), stakePlacement.Amount)
		if err != nil {
			return nil, err
		}
	}
	return &types.MsgConfirmRemoveDelegatedStakeResponse{}, nil
}

///
/// PRIVATE
///

// Making common interfaces available to protobuf messages
func moveFundsAddStake(
	ctx context.Context,
	ms msgServer,
	nodeAddr sdk.AccAddress,
	msg *types.MsgRegister) error {
	// move funds
	initialStakeInt := cosmosMath.NewIntFromBigInt(msg.GetInitialStake().BigInt())
	amount := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, initialStakeInt))
	err := ms.k.SendCoinsFromAccountToModule(ctx, nodeAddr, types.AlloraStakingAccountName, amount)
	if err != nil {
		return err
	}

	// add stake to each topic
	for _, topicId := range msg.TopicIds {
		err = ms.k.AddStake(ctx, topicId, nodeAddr, msg.GetInitialStake())
		if err != nil {
			return err
		}
	}

	return nil
}
