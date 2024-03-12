package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	cosmoserrors "cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	state "github.com/allora-network/allora-chain/x/emissions"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type msgServer struct {
	k Keeper
}

var _ state.MsgServer = msgServer{}

// NewMsgServerImpl returns an implementation of the module MsgServer interface.
func NewMsgServerImpl(keeper Keeper) state.MsgServer {
	return &msgServer{k: keeper}
}

func (ms msgServer) UpdateParams(ctx context.Context, msg *state.MsgUpdateParams) (*state.MsgUpdateParamsResponse, error) {
	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	isAdmin, err := ms.k.IsWhitelistAdmin(ctx, sender)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, state.ErrNotWhitelistAdmin
	}
	existingParams, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	// every option is a repeated field, so we interpret an empty array as "make no change"
	newParams := msg.Params
	if len(newParams.Version) == 1 {
		existingParams.Version = newParams.Version[0]
	}
	if len(newParams.EpochLength) == 1 {
		existingParams.EpochLength = newParams.EpochLength[0]
	}
	if len(newParams.EmissionsPerEpoch) == 1 {
		existingParams.EmissionsPerEpoch = newParams.EmissionsPerEpoch[0]
	}
	if len(newParams.MinTopicUnmetDemand) == 1 {
		existingParams.MinTopicUnmetDemand = newParams.MinTopicUnmetDemand[0]
	}
	if len(newParams.MaxTopicsPerBlock) == 1 {
		existingParams.MaxTopicsPerBlock = newParams.MaxTopicsPerBlock[0]
	}
	if len(newParams.MinRequestUnmetDemand) == 1 {
		existingParams.MinRequestUnmetDemand = newParams.MinRequestUnmetDemand[0]
	}
	if len(newParams.MaxMissingInferencePercent) == 1 {
		existingParams.MaxMissingInferencePercent = newParams.MaxMissingInferencePercent[0]
	}
	if len(newParams.RequiredMinimumStake) == 1 {
		existingParams.RequiredMinimumStake = newParams.RequiredMinimumStake[0]
	}
	if len(newParams.RemoveStakeDelayWindow) == 1 {
		existingParams.RemoveStakeDelayWindow = newParams.RemoveStakeDelayWindow[0]
	}
	if len(newParams.MinRequestCadence) == 1 {
		existingParams.MinRequestCadence = newParams.MinRequestCadence[0]
	}
	if len(newParams.MinLossCadence) == 1 {
		existingParams.MinLossCadence = newParams.MinLossCadence[0]
	}
	if len(newParams.MaxInferenceRequestValidity) == 1 {
		existingParams.MaxInferenceRequestValidity = newParams.MaxInferenceRequestValidity[0]
	}
	if len(newParams.MaxRequestCadence) == 1 {
		existingParams.MaxRequestCadence = newParams.MaxRequestCadence[0]
	}
	err = ms.k.SetParams(ctx, existingParams)
	if err != nil {
		return nil, err
	}
	return &state.MsgUpdateParamsResponse{}, nil
}

// A tx function that accepts a list of inferences and possibly returns an error
func (ms msgServer) ProcessInferences(ctx context.Context, msg *state.MsgProcessInferences) (*state.MsgProcessInferencesResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	inferences := msg.Inferences
	// Group inferences by topicId - Create a map to store the grouped inferences
	groupedInferences := make(map[uint64][]*state.Inference)

	// Iterate through the array and group by topic_id
	for _, inference := range inferences {
		groupedInferences[inference.TopicId] = append(groupedInferences[inference.TopicId], inference)
	}

	actualTimestamp := uint64(sdkCtx.BlockTime().Unix())

	// Update all_inferences
	for topicId, inferences := range groupedInferences {
		inferences := &state.Inferences{
			Inferences: inferences,
		}
		err := ms.k.InsertInferences(ctx, topicId, actualTimestamp, *inferences)
		if err != nil {
			return nil, err
		}
	}

	// Return an empty response as the operation was successful
	return &state.MsgProcessInferencesResponse{}, nil
}

// A tx function that accepts a list of forecasts and possibly returns an error
// Need to call this once per forecaster per topic inference solicitation round because protobuf does not nested repeated fields
func (ms msgServer) ProcessForecasts(ctx context.Context, msg *state.MsgProcessForecasts) (*state.MsgProcessForecastsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	forecasts := msg.Forecasts
	// Group inferences by topicId - Create a map to store the grouped inferences
	groupedForecasts := make(map[uint64][]*state.Forecast)

	// Iterate through the array and group by topic_id
	for _, forecast := range forecasts {
		groupedForecasts[forecast.TopicId] = append(groupedForecasts[forecast.TopicId], forecast)
	}

	actualTimestamp := uint64(sdkCtx.BlockTime().Unix())

	// Update all_inferences
	for topicId, forecasts := range groupedForecasts {
		forecasts := &state.Forecasts{
			Forecasts: forecasts,
		}
		err := ms.k.InsertForecasts(ctx, topicId, actualTimestamp, *forecasts)
		if err != nil {
			return nil, err
		}
	}

	// Return an empty response as the operation was successful
	return &state.MsgProcessForecastsResponse{}, nil
}

// Called by reputer to submit their assessment of the quality of workers' work compared to ground truth
func (ms msgServer) InsertLosses(ctx context.Context, msg *state.MsgSetLosses) (*state.MsgSetLossesResponse, error) {
	// Check if the sender is in the weight setting whitelist
	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	isLossSetter, err := ms.k.IsInReputerWhitelist(ctx, sender)
	if err != nil {
		return nil, err
	}
	if !isLossSetter {
		return nil, state.ErrNotInReputerWhitelist
	}

	// Iterate through the array to ensure each reputer is in the whitelist
	// Group loss bundles by topicId - Create a map to store the grouped loss bundles
	groupedBundles := make(map[uint64][]*state.LossBundle)
	for _, lossBundle := range msg.LossBundles {
		reputer, err := sdk.AccAddressFromBech32(lossBundle.Reputer)
		if err != nil {
			return nil, err
		}
		isLossSetter, err := ms.k.IsInReputerWhitelist(ctx, reputer)
		if err != nil {
			return nil, err
		}
		if isLossSetter {
			groupedBundles[lossBundle.TopicId] = append(groupedBundles[lossBundle.TopicId], lossBundle)
		}
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	actualTimestamp := uint64(sdkCtx.BlockTime().Unix())

	for topicId, lossBundles := range groupedBundles {
		bundles := &state.LossBundles{
			LossBundles: lossBundles,
		}
		err = ms.k.InsertLossBudles(ctx, topicId, actualTimestamp, *bundles)
		if err != nil {
			return nil, err
		}
	}

	/**
	 * TODO calculate eq14,15, and possibly ep9-11
	 */

	return &state.MsgSetLossesResponse{}, nil
}

///
/// NODE REGISTRATION
///

// Registers a new network participant to the network for the first time
func (ms msgServer) Register(ctx context.Context, msg *state.MsgRegister) (*state.MsgRegisterResponse, error) {
	if msg.GetLibP2PKey() == "" {
		return nil, state.ErrLibP2PKeyRequired
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
		return nil, cosmoserrors.Wrapf(state.ErrInsufficientStakeToRegister,
			"required minimum stake: %s, existing address stake: %s, initial stake: %s",
			requiredMinimumStake, addressExistingStake, msg.GetInitialStake())
	}
	// check if topics exists and if address is already registered in any of them
	registeredTopicIds, err := ms.k.GetRegisteredTopicIdsByAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	if len(registeredTopicIds) > 0 {
		return nil, state.ErrAddressAlreadyRegisteredInATopic
	}

	for _, topicId := range msg.TopicIds {
		// check if topic exists
		topicExists, err := ms.k.TopicExists(ctx, topicId)
		if !topicExists {
			return nil, state.ErrTopicDoesNotExist
		} else if err != nil {
			return nil, err
		}
	}

	// move the tokens from the creator to the module account
	// then add the stake to the total, topicTotal, and 3 staking tracking maps
	moveFundsAddStake(ctx, ms, address, msg)

	nodeInfo := state.OffchainNode{
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
			return nil, state.ErrOwnerCannotBeEmpty
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

	return &state.MsgRegisterResponse{
		Success: true,
		Message: "Node successfully registered",
	}, nil
}

// Add additional topics after initial reputer or worker registration
func (ms msgServer) AddNewRegistration(ctx context.Context, msg *state.MsgAddNewRegistration) (*state.MsgAddNewRegistrationResponse, error) {
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
		return nil, state.ErrTopicDoesNotExist
	}
	registeredTopicIds, err := ms.k.GetRegisteredTopicIdsByAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	if len(registeredTopicIds) == 0 {
		return nil, state.ErrAddressIsNotRegisteredInAnyTopic
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

	nodeInfo := state.OffchainNode{
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
				return nil, state.ErrReputerAlreadyRegisteredInTopic
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
				return nil, state.ErrReputerAlreadyRegisteredInTopic
			}
		}

		// add node to topicWorkers
		err = ms.k.InsertWorker(ctx, []uint64{msg.TopicId}, address, nodeInfo)
		if err != nil {
			return nil, err
		}
	}

	return &state.MsgAddNewRegistrationResponse{
		Success: true,
		Message: fmt.Sprintf("Node successfully registered in topic %d", msg.TopicId),
	}, nil
}

// Remove registration from a topic
func (ms msgServer) RemoveRegistration(ctx context.Context, msg *state.MsgRemoveRegistration) (*state.MsgRemoveRegistrationResponse, error) {
	// check if topic exists
	topicExists, err := ms.k.TopicExists(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	} else if !topicExists {
		return nil, state.ErrTopicDoesNotExist
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
		return nil, state.ErrAddressIsNotRegisteredInThisTopic
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
	return &state.MsgRemoveRegistrationResponse{
		Success: true,
		Message: fmt.Sprintf("Node successfully removed from topic %d", msg.TopicId),
	}, nil
}

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
/// REQUESTS
///

func (ms msgServer) RequestInference(ctx context.Context, msg *state.MsgRequestInference) (*state.MsgRequestInferenceResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	for _, requestItem := range msg.Requests {
		request := state.CreateNewInferenceRequestFromListItem(msg.Sender, requestItem)
		// 1. check the topic is valid
		topicExists, err := ms.k.TopicExists(ctx, request.TopicId)
		if err != nil {
			return nil, err
		}
		if !topicExists {
			return nil, state.ErrInvalidTopicId
		}
		requestId, err := request.GetRequestId()
		if err != nil {
			return nil, err
		}
		// 2. check the request isn't already in the mempool
		requestExists, err := ms.k.IsRequestInMempool(ctx, request.TopicId, requestId)
		if err != nil {
			return nil, err
		}
		if requestExists {
			return nil, state.ErrInferenceRequestAlreadyInMempool
		}
		// 3. Check the BidAmount is greater than the price per request
		if request.BidAmount.LT(request.MaxPricePerInference) {
			return nil, state.ErrInferenceRequestBidAmountLessThanPrice
		}
		// 4. Check the timestamp valid until is in the future
		timeNow := uint64(sdkCtx.BlockTime().Unix())
		if request.TimestampValidUntil < timeNow {
			return nil, state.ErrInferenceRequestTimestampValidUntilInPast
		}
		// 5. Check the timestamp validity is no more than the maximum allowed time in the future
		maxInferenceRequestValidity, err := ms.k.GetParamsMaxInferenceRequestValidity(ctx)
		if err != nil {
			return nil, err
		}
		if request.TimestampValidUntil > timeNow+maxInferenceRequestValidity {
			return nil, state.ErrInferenceRequestTimestampValidUntilTooFarInFuture
		}
		if request.Cadence != 0 {
			// 6. Check the cadence is either 0, or greater than the minimum fastest cadence allowed
			minFastestAllowedCadence, err := ms.k.GetParamsMinRequestCadence(ctx)
			if err != nil {
				return nil, err
			}
			if request.Cadence < minFastestAllowedCadence {
				return nil, state.ErrInferenceRequestCadenceTooFast
			}
			// 7. Check the cadence is no more than the maximum allowed slowest cadence
			maxSlowestAllowedCadence, err := ms.k.GetParamsMaxRequestCadence(ctx)
			if err != nil {
				return nil, err
			}
			if request.Cadence > maxSlowestAllowedCadence {
				return nil, state.ErrInferenceRequestCadenceTooSlow
			}
		}
		// 8. Check the cadence is not greater than the timestamp valid until
		if timeNow+request.Cadence > request.TimestampValidUntil {
			return nil, state.ErrInferenceRequestWillNeverBeScheduled
		}
		MinRequestUnmetDemand, err := ms.k.GetParamsMinRequestUnmetDemand(ctx)
		if err != nil {
			return nil, err
		}
		// Check that the request isn't spam by checking that the amount of funds it bids is greater than a global minimum demand per request
		if request.BidAmount.LT(MinRequestUnmetDemand) {
			return nil, state.ErrInferenceRequestBidAmountTooLow
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
		err = ms.k.bankKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, state.AlloraRequestsModuleName, coins)
		if err != nil {
			return nil, err
		}
		// 11. record the number of tokens sent to the module account
		err = ms.k.SetRequestDemand(ctx, requestId, request.BidAmount)
		if err != nil {
			return nil, err
		}
		// 12. Write request state into the mempool state
		request.LastChecked = timeNow
		err = ms.k.AddToMempool(ctx, *request)
		if err != nil {
			return nil, err
		}
	}
	return &state.MsgRequestInferenceResponse{}, nil
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
	return state.ErrAddressNotRegistered
}

///
/// TOPICS
///

func (ms msgServer) CreateNewTopic(ctx context.Context, msg *state.MsgCreateNewTopic) (*state.MsgCreateNewTopicResponse, error) {
	fmt.Println("CreateNewTopic called with: ", msg)
	// Check if the sender is in the topic creation whitelist
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}
	isTopicCreator, err := ms.k.IsInTopicCreationWhitelist(ctx, creator)
	if err != nil {
		return nil, err
	}
	if !isTopicCreator {
		return nil, state.ErrNotInTopicCreationWhitelist
	}

	id, err := ms.k.GetNumTopics(ctx)
	if err != nil {
		return nil, err
	}

	fastestCadence, err := ms.k.GetParamsMinRequestCadence(ctx)
	if err != nil {
		return nil, err
	}
	if msg.InferenceCadence < fastestCadence {
		return nil, state.ErrInferenceCadenceBelowMinimum
	}

	weightFastestCadence, err := ms.k.GetParamsMinLossCadence(ctx)
	if err != nil {
		return nil, err
	}
	if msg.LossCadence < weightFastestCadence {
		return nil, state.ErrLossCadenceBelowMinimum
	}

	topic := state.Topic{
		Id:                     id,
		Creator:                creator.String(),
		Metadata:               msg.Metadata,
		LossLogic:              msg.LossLogic,
		LossMethod:             msg.LossMethod,
		LossCadence:            msg.LossCadence,
		LossLastRan:            0,
		InferenceLogic:         msg.InferenceLogic,
		InferenceMethod:        msg.InferenceMethod,
		InferenceCadence:       msg.InferenceCadence,
		InferenceLastRan:       0,
		Active:                 true,
		DefaultArg:             msg.DefaultArg,
		Pnorm:                  msg.Pnorm,
		AlphaRegret:            msg.AlphaRegret,
		PrewardReputer:         msg.PrewardReputer,
		PrewardInference:       msg.PrewardInference,
		PrewardForecast:        msg.PrewardForecast,
		FTolerance:             msg.FTolerance,
		Subsidy:                0,   // Can later be updated by a Foundation member
		SubsidizedRewardEpochs: 0,   // Can later be updated by a Foundation member
		FTreasury:              0.5, // Can later be updated by a Foundation member
	}
	_, err = ms.k.IncrementTopicId(ctx)
	if err != nil {
		return nil, err
	}
	if err := ms.k.SetTopic(ctx, id, topic); err != nil {
		return nil, err
	}
	// Rather than set latest weight-adjustment timestamp of a topic to 0
	// we do nothing, since no value in the map means zero

	return &state.MsgCreateNewTopicResponse{TopicId: id}, nil
}

func (ms msgServer) ReactivateTopic(ctx context.Context, msg *state.MsgReactivateTopic) (*state.MsgReactivateTopicResponse, error) {
	// Check that the topic has enough demand to be reactivated
	unmetDemand, err := ms.k.GetTopicUnmetDemand(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	}

	minTopicUnmentDemand, err := ms.k.GetParamsMinTopicUnmetDemand(ctx)
	if err != nil {
		return nil, err
	}
	// If the topic does not have enough demand, return an error
	if unmetDemand.LT(minTopicUnmentDemand) {
		return nil, state.ErrTopicNotEnoughDemand
	}

	// If the topic has enough demand, reactivate it
	err = ms.k.ReactivateTopic(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	}
	return &state.MsgReactivateTopicResponse{Success: true}, nil
}

///
/// FOUNDATION TOPIC MANAGEMENT
///

// Modfies topic subsidy and subsidized_reward_epochs properties
// Can only be called by a whitelisted foundation member
func (ms msgServer) ModifyTopicSubsidy(ctx context.Context, msg *state.MsgModifyTopicSubsidyAndSubsidizedRewardEpochs) (*state.MsgModifyTopicSubsidyAndSubsidizedRewardEpochsResponse, error) {
	// Check that sender is in the foundation whitelist
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	isAdmin, err := ms.k.IsInFoundationWhitelist(ctx, senderAddr)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, state.ErrNotWhitelistAdmin
	}
	// Modify the topic subsidy + F_treasury
	err = ms.k.SetTopicSubsidy(ctx, msg.TopicId, msg.Subsidy)
	if err != nil {
		return nil, err
	}
	err = ms.k.SetTopicSubsidizedRewardEpochs(ctx, msg.TopicId, msg.SubsidizedRewardEpochs)
	if err != nil {
		return nil, err
	}
	return &state.MsgModifyTopicSubsidyAndSubsidizedRewardEpochsResponse{Success: true}, nil
}

func (ms msgServer) ModifyTopicFTreasury(ctx context.Context, msg *state.MsgModifyTopicFTreasury) (*state.MsgModifyTopicFTreasuryResponse, error) {
	// Check that sender is in the foundation whitelist
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	isAdmin, err := ms.k.IsInFoundationWhitelist(ctx, senderAddr)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, state.ErrNotWhitelistAdmin
	}
	// Modify the topic subsidy + F_treasury
	err = ms.k.SetTopicFTreasury(ctx, msg.TopicId, float32(msg.FTreasury))
	if err != nil {
		return nil, err
	}
	return &state.MsgModifyTopicFTreasuryResponse{Success: true}, nil
}

///
/// WHITELIST
///

func (ms msgServer) AddToWhitelistAdmin(ctx context.Context, msg *state.MsgAddToWhitelistAdmin) (*state.MsgAddToWhitelistAdminResponse, error) {
	// Check that sender is also a whitelist admin
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	targetAddr, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return nil, err
	}
	isAdmin, err := ms.k.IsWhitelistAdmin(ctx, senderAddr)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, state.ErrNotWhitelistAdmin
	}
	// Add the address to the whitelist
	err = ms.k.AddWhitelistAdmin(ctx, targetAddr)
	if err != nil {
		return nil, err
	}
	return &state.MsgAddToWhitelistAdminResponse{}, nil
}

func (ms msgServer) RemoveFromWhitelistAdmin(ctx context.Context, msg *state.MsgRemoveFromWhitelistAdmin) (*state.MsgRemoveFromWhitelistAdminResponse, error) {
	// Check that sender is also a whitelist admin
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	targetAddr, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return nil, err
	}
	isAdmin, err := ms.k.IsWhitelistAdmin(ctx, senderAddr)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, state.ErrNotWhitelistAdmin
	}
	// Remove the address from the whitelist
	err = ms.k.RemoveWhitelistAdmin(ctx, targetAddr)
	if err != nil {
		return nil, err
	}
	return &state.MsgRemoveFromWhitelistAdminResponse{}, nil
}

func (ms msgServer) AddToTopicCreationWhitelist(ctx context.Context, msg *state.MsgAddToTopicCreationWhitelist) (*state.MsgAddToTopicCreationWhitelistResponse, error) {
	// Check that sender is also a whitelist admin
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	targetAddr, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return nil, err
	}
	isAdmin, err := ms.k.IsWhitelistAdmin(ctx, senderAddr)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, state.ErrNotWhitelistAdmin
	}
	// Add the address to the whitelist
	err = ms.k.AddToTopicCreationWhitelist(ctx, targetAddr)
	if err != nil {
		return nil, err
	}
	return &state.MsgAddToTopicCreationWhitelistResponse{}, nil
}

func (ms msgServer) RemoveFromTopicCreationWhitelist(ctx context.Context, msg *state.MsgRemoveFromTopicCreationWhitelist) (*state.MsgRemoveFromTopicCreationWhitelistResponse, error) {
	// Check that sender is also a whitelist admin
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	targetAddr, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return nil, err
	}
	isAdmin, err := ms.k.IsWhitelistAdmin(ctx, senderAddr)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, state.ErrNotWhitelistAdmin
	}
	// Remove the address from the whitelist
	err = ms.k.RemoveFromTopicCreationWhitelist(ctx, targetAddr)
	if err != nil {
		return nil, err
	}
	return &state.MsgRemoveFromTopicCreationWhitelistResponse{}, nil
}

func (ms msgServer) AddToReputerWhitelist(ctx context.Context, msg *state.MsgAddToReputerWhitelist) (*state.MsgAddToReputerWhitelistResponse, error) {
	// Check that sender is also a whitelist admin
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	targetAddr, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return nil, err
	}
	isAdmin, err := ms.k.IsWhitelistAdmin(ctx, senderAddr)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, state.ErrNotWhitelistAdmin
	}
	// Add the address to the whitelist
	err = ms.k.AddToReputerWhitelist(ctx, targetAddr)
	if err != nil {
		return nil, err
	}
	return &state.MsgAddToReputerWhitelistResponse{}, nil
}

func (ms msgServer) RemoveFromReputerWhitelist(ctx context.Context, msg *state.MsgRemoveFromReputerWhitelist) (*state.MsgRemoveFromReputerWhitelistResponse, error) {
	// Check that sender is also a whitelist admin
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	targetAddr, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return nil, err
	}
	isAdmin, err := ms.k.IsWhitelistAdmin(ctx, senderAddr)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, state.ErrNotWhitelistAdmin
	}
	// Remove the address from the whitelist
	err = ms.k.RemoveFromReputerWhitelist(ctx, targetAddr)
	if err != nil {
		return nil, err
	}
	return &state.MsgRemoveFromReputerWhitelistResponse{}, nil
}

func (ms msgServer) AddToFoundationWhitelist(ctx context.Context, msg *state.MsgAddToFoundationWhitelist) (*state.MsgAddToFoundationWhitelistResponse, error) {
	// Check that sender is also a whitelist admin
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	targetAddr, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return nil, err
	}
	isAdmin, err := ms.k.IsWhitelistAdmin(ctx, senderAddr)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, state.ErrNotWhitelistAdmin
	}
	// Add the address to the foundation whitelist
	err = ms.k.AddToFoundationWhitelist(ctx, targetAddr)
	if err != nil {
		return nil, err
	}
	return &state.MsgAddToFoundationWhitelistResponse{}, nil
}

func (ms msgServer) RemoveFromFoundationWhitelist(ctx context.Context, msg *state.MsgRemoveFromFoundationWhitelist) (*state.MsgRemoveFromFoundationWhitelistResponse, error) {
	// Check that sender is also a whitelist admin
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	targetAddr, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return nil, err
	}
	isAdmin, err := ms.k.IsWhitelistAdmin(ctx, senderAddr)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, state.ErrNotWhitelistAdmin
	}
	// Remove the address from the foundation whitelist
	err = ms.k.RemoveFromFoundationWhitelist(ctx, targetAddr)
	if err != nil {
		return nil, err
	}
	return &state.MsgRemoveFromFoundationWhitelistResponse{}, nil
}
