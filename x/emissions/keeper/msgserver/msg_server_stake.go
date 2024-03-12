package msgserver

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	state "github.com/allora-network/allora-chain/x/emissions"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

///
/// STAKE
///

// Function for reputers or workers to call to add stake to an existing stake position.
func (ms msgServer) AddStake(ctx context.Context, msg *state.MsgAddStake) (*state.MsgAddStakeResponse, error) {
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
	err = checkNodeRegistered(ctx, ms, targetAddr)
	if err != nil {
		return nil, err
	}

	// 3. check the sender has enough funds to add the stake
	// bank module does this for us in module SendCoins / subUnlockedCoins so we don't need to check
	// 4. send the funds
	amountInt := cosmosMath.NewIntFromBigInt(msg.Amount.BigInt())
	coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amountInt))
	ms.k.bankKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, state.AlloraStakingModuleName, coins)

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

	return &state.MsgAddStakeResponse{}, nil
}

// StartRemoveStake kicks off a stake removal process. Stake Removals are placed into a delayed queue.
// once the withdrawal delay has passed then ConfirmRemoveStake can be called to remove the stake.
// if a stake removal is not confirmed within a certain time period, the stake removal becomes invalid
// and one must start the stake removal process again and wait the delay again.
func (ms msgServer) StartRemoveStake(ctx context.Context, msg *state.MsgStartRemoveStake) (*state.MsgStartRemoveStakeResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// Check the sender is registered
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	stakeRemoval := state.StakeRemoval{
		TimestampRemovalStarted: uint64(sdkCtx.BlockTime().Unix()),
		Placements:              make([]*state.StakeRemovalPlacement, 0),
	}
	for _, stakePlacement := range msg.PlacementsRemove {
		// Check the sender has enough stake already placed on the topic to remove the stake
		stakePlaced, err := ms.k.GetStakeOnTopicFromReputer(ctx, stakePlacement.TopicId, senderAddr)
		if err != nil {
			return nil, err
		}
		if stakePlaced.LT(stakePlacement.Amount) {
			return nil, state.ErrInsufficientStakeToRemove
		}

		// If user is still registered in the topic check that the stake is greater than the minimum required
		requiredMinimumStake, err := ms.k.GetParamsRequiredMinimumStake(ctx)
		if err != nil {
			return nil, err
		}
		if stakePlaced.Sub(stakePlacement.Amount).LT(requiredMinimumStake) {
			return nil, state.ErrInsufficientStakeAfterRemoval
		}

		// Push to the stake removal object
		stakeRemoval.Placements = append(stakeRemoval.Placements, &state.StakeRemovalPlacement{
			TopicId: stakePlacement.TopicId,
			Amount:  stakePlacement.Amount,
		})
	}
	// If no errors have occured and the removal is valid, add the stake removal to the delayed queue
	err = ms.k.SetStakeRemovalQueueForAddress(ctx, senderAddr, stakeRemoval)
	if err != nil {
		return nil, err
	}
	return &state.MsgStartRemoveStakeResponse{}, nil
}

// Function for reputers or workers to call to remove stake from an existing stake position.
func (ms msgServer) ConfirmRemoveStake(ctx context.Context, msg *state.MsgConfirmRemoveStake) (*state.MsgConfirmRemoveStakeResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// pull the stake removal from the delayed queue
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	stakeRemoval, err := ms.k.GetStakeRemovalQueueByAddress(ctx, senderAddr)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, state.ErrConfirmRemoveStakeNoRemovalStarted
		}
		return nil, err
	}
	// check the timestamp is valid
	timeNow := uint64(sdkCtx.BlockTime().Unix())
	if stakeRemoval.TimestampRemovalStarted > timeNow {
		return nil, state.ErrConfirmRemoveStakeTooEarly
	}
	delayWindow, err := ms.k.GetParamsRemoveStakeDelayWindow(ctx)
	if err != nil {
		return nil, err
	}
	if stakeRemoval.TimestampRemovalStarted+delayWindow < timeNow {
		return nil, state.ErrConfirmRemoveStakeTooLate
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
		ms.k.bankKeeper.SendCoinsFromModuleToAccount(ctx, state.AlloraStakingModuleName, senderAddr, coins)

		// 7. update the stake data structures
		err = ms.k.RemoveStake(ctx, stakePlacement.TopicId, senderAddr, stakePlacement.Amount)
		if err != nil {
			return nil, err
		}
	}
	return &state.MsgConfirmRemoveStakeResponse{}, nil
}

// StartRemoveAllStake kicks off a stake removal process. Stake Removals are placed into a delayed queue.
// once the withdrawal delay has passed then ConfirmRemoveStake can be called to remove the stake.
// if a stake removal is not confirmed within a certain time period, the stake removal becomes invalid
// and one must start the stake removal process again and wait the delay again.
// RemoveAllStake is just a convenience wrapper around StartRemoveStake.
func (ms msgServer) StartRemoveAllStake(ctx context.Context, msg *state.MsgStartRemoveAllStake) (*state.MsgStartRemoveAllStakeResponse, error) {
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	stakePlacements, err := ms.k.GetStakePlacementsForReputer(ctx, senderAddr)
	if err != nil {
		return nil, err
	}
	msgRemoveStake := &state.MsgStartRemoveStake{
		Sender:           msg.Sender,
		PlacementsRemove: make([]*state.StakePlacement, 0),
	}
	for _, stakePlacement := range stakePlacements {
		msgRemoveStake.PlacementsRemove = append(msgRemoveStake.PlacementsRemove, &state.StakePlacement{
			TopicId: stakePlacement.TopicId,
			Amount:  stakePlacement.Amount,
		})
	}
	_, err = ms.StartRemoveStake(ctx, msgRemoveStake)
	if err != nil {
		return nil, err
	}
	return &state.MsgStartRemoveAllStakeResponse{}, nil
}

///
/// PRIVATE
///

// Making common interfaces available to protobuf messages
func moveFundsAddStake(
	ctx context.Context,
	ms msgServer,
	nodeAddr sdk.AccAddress,
	msg *state.MsgRegister) error {
	// move funds
	initialStakeInt := cosmosMath.NewIntFromBigInt(msg.GetInitialStake().BigInt())
	amount := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, initialStakeInt))
	err := ms.k.bankKeeper.SendCoinsFromAccountToModule(ctx, nodeAddr, state.AlloraStakingModuleName, amount)
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

