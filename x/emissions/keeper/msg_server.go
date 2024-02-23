package keeper

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cosmossdk.io/collections"
	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	state "github.com/allora-network/allora-chain/x/emissions"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const REQUIRED_MINIMUM_STAKE = 1
const DELAY_WINDOW = 172800                                        // 48 hours in seconds
const MIN_FASTEST_ALLOWED_CADENCE = 60                             // 1 minute in seconds
const MAX_INFERENCE_REQUEST_VALIDITY = 60 * 60 * 24 * 7 * 24       // 24 weeks approximately 6 months in seconds
const MAX_SLOWEST_ALLOWED_CADENCE = MAX_INFERENCE_REQUEST_VALIDITY // 24 weeks approximately 6 months in seconds

type msgServer struct {
	k Keeper
}

var _ state.MsgServer = msgServer{}

// NewMsgServerImpl returns an implementation of the module MsgServer interface.
func NewMsgServerImpl(keeper Keeper) state.MsgServer {
	return &msgServer{k: keeper}
}

func (ms msgServer) CreateNewTopic(ctx context.Context, msg *state.MsgCreateNewTopic) (*state.MsgCreateNewTopicResponse, error) {
	id, err := ms.k.GetNumTopics(ctx)
	if err != nil {
		return nil, err
	}

	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	if msg.InferenceCadence < 60 {
		return nil, state.ErrInferenceCadenceBelowMinimum
	}

	if msg.WeightCadence < 10800 {
		return nil, state.ErrWeightCadenceBelowMinimum
	}

	topic := state.Topic{
		Id:               id,
		Creator:          creator.String(),
		Metadata:         msg.Metadata,
		WeightLogic:      msg.WeightLogic,
		WeightMethod:     msg.WeightMethod,
		WeightCadence:    msg.WeightCadence,
		WeightLastRan:    0,
		InferenceLogic:   msg.InferenceLogic,
		InferenceMethod:  msg.InferenceMethod,
		InferenceCadence: msg.InferenceCadence,
		InferenceLastRan: 0,
		Active:           true,
		DefaultArg:       msg.DefaultArg,
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

func (ms msgServer) SetWeights(ctx context.Context, msg *state.MsgSetWeights) (*state.MsgSetWeightsResponse, error) {
	//
	// WHITELIST CHECK SENDER
	//

	for _, weightEntry := range msg.Weights {

		fmt.Println("Topic: ", weightEntry.TopicId, "| Reputer: ", weightEntry.Reputer, "| Worker: ", weightEntry.Worker, "| Weight: ", weightEntry.Weight)

		reputerAddr := sdk.AccAddress(weightEntry.Reputer)
		workerAddr := sdk.AccAddress(weightEntry.Worker)

		err := ms.k.SetWeight(ctx, weightEntry.TopicId, reputerAddr, workerAddr, weightEntry.Weight)
		if err != nil {
			return nil, err
		}
	}

	return &state.MsgSetWeightsResponse{}, nil
}

func (ms msgServer) SetInferences(ctx context.Context, msg *state.MsgSetInferences) (*state.MsgSetInferencesResponse, error) {
	for _, inferenceEntry := range msg.Inferences {
		workerAddr := sdk.AccAddress(inferenceEntry.Worker)

		err := ms.k.SetInference(ctx, inferenceEntry.TopicId, workerAddr, *inferenceEntry)
		if err != nil {
			return nil, err
		}
	}

	return &state.MsgSetInferencesResponse{}, nil
}

// T1: a tx function that accepts a list of inferences and possibly returns an error
func (ms msgServer) ProcessInferences(ctx context.Context, msg *state.MsgProcessInferences) (*state.MsgProcessInferencesResponse, error) {
	inferences := msg.Inferences
	// Group inferences by topicId - Create a map to store the grouped inferences
	groupedInferences := make(map[uint64][]*state.Inference)

	// Iterate through the array and group by topic_id
	for _, inference := range inferences {
		groupedInferences[inference.TopicId] = append(groupedInferences[inference.TopicId], inference)
	}

	actualTimestamp := uint64(time.Now().UTC().Unix())

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

// ########################################
// #           Node Registration          #
// ########################################

// Registers a new network participant to the network for the first time
func (ms msgServer) Register(ctx context.Context, msg *state.MsgRegister) (*state.MsgRegisterResponse, error) {
	if msg.GetLibP2PKey() == "" {
		return nil, state.ErrLibP2PKeyRequired
	}
	// require funds to be at least greater than the minimum stake
	if msg.GetInitialStake().LT(cosmosMath.NewUint(REQUIRED_MINIMUM_STAKE)) {
		return nil, state.ErrInsufficientStakeToRegister
	}
	// check if topics exists and if address is already registered in any of them
	address, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}
	registeredTopicsIds, err := ms.k.GetRegisteredTopicsIdsByAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	if len(registeredTopicsIds) > 0 {
		return nil, state.ErrAddressAlreadyRegisteredInATopic
	}

	for _, topicId := range msg.TopicsIds {
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
		err = ms.k.InsertReputer(ctx, msg.TopicsIds, address, nodeInfo)
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
		err = ms.k.InsertWorker(ctx, msg.TopicsIds, address, nodeInfo)
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
	registeredTopicsIds, err := ms.k.GetRegisteredTopicsIdsByAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	if len(registeredTopicsIds) == 0 {
		return nil, state.ErrAddressIsNotRegisteredInAnyTopic
	}

	// copy overall staking power of the wallet to the topic stake
	totalAddressStake, err := ms.k.GetStakePlacedUponTarget(ctx, address)
	if err != nil {
		return nil, err
	}

	// add to topic stake
	err = ms.k.AddStakeToTopics(ctx, []uint64{msg.GetTopicId()}, totalAddressStake)
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
		reputerRegisteredTopicsIds, err := ms.k.GetRegisteredTopicsIdsByReputerAddress(ctx, address)
		if err != nil {
			return nil, err
		}
		for _, topicIdRegistered := range reputerRegisteredTopicsIds {
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
		reputerRegisteredTopicsIds, err := ms.k.GetRegisteredTopicsIdsByWorkerAddress(ctx, address)
		if err != nil {
			return nil, err
		}
		for _, topicIdRegistered := range reputerRegisteredTopicsIds {
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
	registeredTopicsIds, err := ms.k.GetRegisteredTopicsIdsByAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	isRegisteredInTopic := false
	for _, topicId := range registeredTopicsIds {
		if topicId == msg.TopicId {
			isRegisteredInTopic = true
			break
		}
	}
	if !isRegisteredInTopic {
		return nil, state.ErrAddressIsNotRegisteredInThisTopic
	}

	// remove overall staking power of the wallet to the topic stake
	totalAddressStake, err := ms.k.GetStakePlacedUponTarget(ctx, address)
	if err != nil {
		return nil, err
	}

	// remove from topic stake
	err = ms.k.RemoveStakeFromTopics(ctx, []uint64{msg.TopicId}, totalAddressStake)
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
	topicsIds, err := ms.k.GetRegisteredTopicsIdsByAddress(ctx, targetAddr)
	if err != nil {
		return nil, err
	}

	// 6. update the stake data structures
	err = ms.k.AddStake(ctx, topicsIds, msg.Sender, msg.StakeTarget, msg.Amount)
	if err != nil {
		return nil, err
	}

	return &state.MsgAddStakeResponse{}, nil
}

func (ms msgServer) ModifyStake(ctx context.Context, msg *state.MsgModifyStake) (*state.MsgModifyStakeResponse, error) {
	// 1. check the sender is registered
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	// 2. For all stake befores, check the sum is less than or equal to this sender's existing stake
	// 3. For all stake befores, check that the bond is greater than or equal to the amount being removed
	senderTotalStake, err := ms.k.GetDelegatorStake(ctx, senderAddr)
	if err != nil {
		return nil, err
	}
	beforeSum := cosmosMath.NewUint(0)
	for _, stakeBefore := range msg.PlacementsRemove {
		beforeSum = beforeSum.Add(stakeBefore.Amount)
		targetAddr, err := sdk.AccAddressFromBech32(stakeBefore.Target)
		if err != nil {
			return nil, err
		}
		bond, err := ms.k.GetBond(ctx, senderAddr, targetAddr)
		if err != nil {
			return nil, err
		}
		if bond.LT(stakeBefore.Amount) {
			return nil, state.ErrModifyStakeBeforeBondLessThanAmountModified
		}
	}
	if senderTotalStake.LT(beforeSum) {
		return nil, state.ErrModifyStakeBeforeSumGreaterThanSenderStake
	}
	// 4. For all stake afters, check that the target is a valid signed up participant
	// 5. For all stake afters, check that the sum is equal to the sum of stake befores
	afterSum := cosmosMath.NewUint(0)
	for _, stakeAfter := range msg.PlacementsAdd {
		targetAddr, err := sdk.AccAddressFromBech32(stakeAfter.Target)
		if err != nil {
			return nil, err
		}
		afterSum = afterSum.Add(stakeAfter.Amount)
		err = checkNodeRegistered(ctx, ms, targetAddr)
		if err != nil {
			return nil, err
		}
	}
	if !afterSum.Equal(beforeSum) {
		return nil, state.ErrModifyStakeSumBeforeNotEqualToSumAfter
	}

	// Update the stake data structures
	// 6. For all stake befores, remove the stake
	// 7. For all stake afters, add the stake to the existing stake position
	for _, stakeBefore := range msg.PlacementsRemove {
		targetAddr, err := sdk.AccAddressFromBech32(stakeBefore.Target)
		if err != nil {
			return nil, err
		}
		err = ms.k.SubStakePlacement(ctx, senderAddr, targetAddr, stakeBefore.Amount)
		if err != nil {
			return nil, err
		}
		err = ms.k.SubStakePlacedUponTarget(ctx, targetAddr, stakeBefore.Amount)
		if err != nil {
			return nil, err
		}

		topicsIds, err := ms.k.GetRegisteredTopicsIdsByAddress(ctx, targetAddr)
		if err != nil {
			return nil, err
		}

		err = ms.k.RemoveStakeFromTopics(ctx, topicsIds, stakeBefore.Amount)
		if err != nil {
			return nil, err
		}
	}
	for _, stakeAfter := range msg.PlacementsAdd {
		targetAddr, err := sdk.AccAddressFromBech32(stakeAfter.Target)
		if err != nil {
			return nil, err
		}
		err = ms.k.AddStakePlacement(ctx, senderAddr, targetAddr, stakeAfter.Amount)
		if err != nil {
			return nil, err
		}
		err = ms.k.AddStakePlacedUponTarget(ctx, targetAddr, stakeAfter.Amount)
		if err != nil {
			return nil, err
		}

		topicsIds, err := ms.k.GetRegisteredTopicsIdsByAddress(ctx, targetAddr)
		if err != nil {
			return nil, err
		}

		err = ms.k.AddStakeToTopics(ctx, topicsIds, stakeAfter.Amount)
		if err != nil {
			return nil, err
		}
	}

	return &state.MsgModifyStakeResponse{}, nil
}

// StartRemoveStake kicks off a stake removal process. Stake Removals are placed into a delayed queue.
// once the withdrawal delay has passed then ConfirmRemoveStake can be called to remove the stake.
// if a stake removal is not confirmed within a certain time period, the stake removal becomes invalid
// and one must start the stake removal process again and wait the delay again.
func (ms msgServer) StartRemoveStake(ctx context.Context, msg *state.MsgStartRemoveStake) (*state.MsgStartRemoveStakeResponse, error) {
	// 1. check the sender is registered
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	stakeRemoval := state.StakeRemoval{
		TimestampRemovalStarted: uint64(time.Now().UTC().Unix()),
		Placements:              make([]*state.StakeRemovalPlacement, 0),
	}
	for _, stakePlacement := range msg.PlacementsRemove {
		// 2. check the target exists and is registered
		targetAddr, err := sdk.AccAddressFromBech32(stakePlacement.Target)
		if err != nil {
			return nil, err
		}

		// 3. check the sender has enough stake already placed on the target to remove the stake
		stakePlaced, err := ms.k.GetBond(ctx, senderAddr, targetAddr)
		if err != nil {
			return nil, err
		}
		if stakePlaced.LT(stakePlacement.Amount) {
			return nil, state.ErrInsufficientStakeToRemove
		}

		// 4. get topics ids where the target is registered
		topicsIds, err := ms.k.GetRegisteredTopicsIdsByAddress(ctx, targetAddr)
		if err != nil {
			return nil, err
		}

		// 6. If user is removing stake from themselves and he still registered in topics
		//  check that the stake is greater than the minimum required
		if senderAddr.String() == targetAddr.String() &&
			stakePlaced.Sub(stakePlacement.Amount).LT(cosmosMath.NewUint(REQUIRED_MINIMUM_STAKE)) &&
			len(topicsIds) > 0 {
			return nil, state.ErrInsufficientStakeAfterRemoval
		}

		// 7. push to the stake removal object
		stakeRemoval.Placements = append(stakeRemoval.Placements, &state.StakeRemovalPlacement{
			TopicsIds: topicsIds,
			Target:    stakePlacement.Target,
			Amount:    stakePlacement.Amount,
		})
	}
	// 8. if no errors have occured and the removal is valid, add the stake removal to the delayed queue
	err = ms.k.SetStakeRemovalQueueForDelegator(ctx, senderAddr, stakeRemoval)
	if err != nil {
		return nil, err
	}
	return &state.MsgStartRemoveStakeResponse{}, nil
}

// Function for reputers or workers to call to remove stake from an existing stake position.
func (ms msgServer) ConfirmRemoveStake(ctx context.Context, msg *state.MsgConfirmRemoveStake) (*state.MsgConfirmRemoveStakeResponse, error) {
	// pull the stake removal from the delayed queue
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	stakeRemoval, err := ms.k.GetStakeRemovalQueueForDelegator(ctx, senderAddr)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, state.ErrConfirmRemoveStakeNoRemovalStarted
		}
		return nil, err
	}
	// check the timestamp is valid
	timeNow := uint64(time.Now().UTC().Unix())
	if stakeRemoval.TimestampRemovalStarted > timeNow {
		return nil, state.ErrConfirmRemoveStakeTooEarly
	}
	if stakeRemoval.TimestampRemovalStarted+DELAY_WINDOW < timeNow {
		return nil, state.ErrConfirmRemoveStakeTooLate
	}
	// skip checking all the data is valid
	// the data should be valid because it was checked when the stake removal was started
	// send the money
	for _, stakePlacement := range stakeRemoval.Placements {
		targetAddr, err := sdk.AccAddressFromBech32(stakePlacement.Target)
		if err != nil {
			return nil, err
		}
		// 5. check the module has enough funds to send back to the sender
		// bank module does this for us in module SendCoins / subUnlockedCoins so we don't need to check
		// 6. send the funds
		amountInt := cosmosMath.NewIntFromBigInt(stakePlacement.Amount.BigInt())
		coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amountInt))
		ms.k.bankKeeper.SendCoinsFromModuleToAccount(ctx, state.AlloraStakingModuleName, senderAddr, coins)

		// 7. update the stake data structures
		err = ms.k.RemoveStakeFromBond(ctx, stakePlacement.TopicsIds, senderAddr, targetAddr, stakePlacement.Amount)
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
	targets, amounts, err := ms.k.GetAllBondsForDelegator(ctx, senderAddr)
	if err != nil {
		return nil, err
	}
	msgRemoveStake := &state.MsgStartRemoveStake{
		Sender:           msg.Sender,
		PlacementsRemove: make([]*state.StakePlacement, 0),
	}
	for i, target := range targets {
		msgRemoveStake.PlacementsRemove = append(msgRemoveStake.PlacementsRemove, &state.StakePlacement{
			Target: target.String(),
			Amount: amounts[i],
		})
	}
	_, err = ms.StartRemoveStake(ctx, msgRemoveStake)
	if err != nil {
		return nil, err
	}
	return &state.MsgStartRemoveAllStakeResponse{}, nil
}

func (ms msgServer) RequestInference(ctx context.Context, msg *state.MsgRequestInference) (*state.MsgRequestInferenceResponse, error) {
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
		timeNow := uint64(time.Now().UTC().Unix())
		if request.TimestampValidUntil < timeNow {
			return nil, state.ErrInferenceRequestTimestampValidUntilInPast
		}
		// 5. Check the timestamp validity is no more than the maximum allowed time in the future
		if request.TimestampValidUntil > timeNow+MAX_INFERENCE_REQUEST_VALIDITY {
			return nil, state.ErrInferenceRequestTimestampValidUntilTooFarInFuture
		}
		if request.Cadence != 0 {
			// 6. Check the cadence is either 0, or greater than the minimum fastest cadence allowed
			if request.Cadence < MIN_FASTEST_ALLOWED_CADENCE {
				return nil, state.ErrInferenceRequestCadenceTooFast
			}
			// 7. Check the cadence is no more than the maximum allowed slowest cadence
			if request.Cadence > MAX_SLOWEST_ALLOWED_CADENCE {
				return nil, state.ErrInferenceRequestCadenceTooSlow
			}
		}
		// 8. Check the cadence is not greater than the timestamp valid until
		if timeNow+request.Cadence > request.TimestampValidUntil {
			return nil, state.ErrInferenceRequestWillNeverBeScheduled
		}
		// Check that the request isn't spam by checking that the amount of funds it bids is greater than a global minimum demand per request
		if request.BidAmount.LT(cosmosMath.NewUint(MIN_UNMET_DEMAND)) {
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
		coins := sdk.NewCoins(sdk.NewCoin(params.BaseCoinUnit, amountInt))
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

// ########################################
// #           Private Functions          #
// ########################################

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

	// for now we will force initial stake deposits to be placed upon oneself.
	// add to total stake
	// add to topic stake
	// add to stakeOwnedByDelegator
	// add to stakePlacement
	// add to stakePlacedUponTarget
	err = ms.k.AddStake(ctx, msg.GetTopicsIds(), msg.GetCreator(), msg.GetCreator(), msg.GetInitialStake())
	if err != nil {
		return err
	}

	return nil
}

// checks if a node is registered in the system and if it is,
// returns whether said node is a reputer or a worker
func checkNodeRegistered(ctx context.Context, ms msgServer, node sdk.AccAddress) error {

	reputerRegisteredTopicsIds, err := ms.k.GetRegisteredTopicsIdsByReputerAddress(ctx, node)
	if err != nil {
		return err
	}
	if len(reputerRegisteredTopicsIds) > 0 {
		return nil
	}
	workerRegisteredTopicsIds, err := ms.k.GetRegisteredTopicsIdsByWorkerAddress(ctx, node)
	if err != nil {
		return err
	}
	if len(workerRegisteredTopicsIds) > 0 {
		return nil
	}
	return state.ErrAddressNotRegistered
}
