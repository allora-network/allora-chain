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

type NodeExists int8

const (
	isWorker NodeExists = iota
	isReputer
	isNotFound
)

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

	// TODO: Add after demo
	// if msg.InferenceCadence < 60 {
	// 	return nil, fmt.Errorf("inference cadence must be at least 60 seconds (1 minute)")
	// }

	// if msg.WeightCadence < 10800 {
	// 	return nil, fmt.Errorf("weight cadence must be at least 10800 seconds (3 hours)")
	// }

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
// Reputer Registration signs up a new reputer
// to be a reputer for a given topicId on the network
func (ms msgServer) RegisterReputer(ctx context.Context, msg *state.MsgRegisterReputer) (*state.MsgRegisterReputerResponse, error) {
	err := validateRegistrationCommon(ctx, ms, msg)
	if err != nil {
		return nil, err
	}

	// check if topics exists and if reputer is already registered in any of them
	reputerAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}
	topicsIds, err := ms.k.GetRegisteredTopicsIdsByReputerAddress(ctx, reputerAddr)
	if err != nil {
		return nil, err
	}

	for _, topicId := range msg.TopicsIds {
		// check if topic exists
		exist, err := ms.k.TopicExists(ctx, topicId)
		if !exist {
			return nil, state.ErrTopicDoesNotExist
		} else if err != nil {
			return nil, err
		}

		// check if reputer is already registered in the topic
		for _, topicIdRegistered := range topicsIds {
			if topicIdRegistered == topicId {
				return nil, state.ErrReputerAlreadyRegisteredInTopic
			}
		}
	}

	if len(topicsIds) == 0 {
		// move the tokens from the creator to the module account
		// then add the stake to the total, topicTotal, and 3 staking tracking maps
		moveFundsAddStake(ctx, ms, reputerAddr, msg)
	} else {
		// add overall staking power of the wallet to the topics stakes
		moveFundsAddStakeToTopics(ctx, ms, reputerAddr, msg)
	}

	// add node to the data structures that track the nodes:
	// add node to topicReputers
	// add node to reputers
	reputerInfo := state.OffchainNode{
		TopicsIds:    msg.TopicsIds,
		LibP2PKey:    msg.LibP2PKey,
		MultiAddress: msg.MultiAddress,
	}

	err = ms.k.InsertReputer(ctx, msg.TopicsIds, reputerAddr, reputerInfo)
	if err != nil {
		return nil, err
	}

	// Return a successful response
	return &state.MsgRegisterReputerResponse{
		Success: true,
		Message: "Reputer node successfully registered",
	}, nil
}

// Reputer Registration signs up a new reputer
// to be a reputer for a given topicId on the network
func (ms msgServer) RegisterWorker(ctx context.Context, msg *state.MsgRegisterWorker) (*state.MsgRegisterWorkerResponse, error) {
	err := validateRegistrationCommon(ctx, ms, msg)
	if err != nil {
		return nil, err
	}
	// check if topics exists and if worker is already registered in any of them
	workerAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}
	topicsIds, err := ms.k.GetRegisteredTopicsIdsByWorkerAddress(ctx, workerAddr)
	if err != nil {
		return nil, err
	}

	for _, topicId := range msg.TopicsIds {
		// check if topic exists
		exist, err := ms.k.TopicExists(ctx, topicId)
		if !exist {
			return nil, state.ErrTopicDoesNotExist
		} else if err != nil {
			return nil, err
		}

		// check if worker is already registered in the topic
		for _, topicIdRegistered := range topicsIds {
			if topicIdRegistered == topicId {
				return nil, state.ErrWorkerAlreadyRegisteredInTopic
			}
		}
	}

	if len(topicsIds) == 0 {
		// move the tokens from the creator to the module account
		// then add the stake to the total, topicTotal, and 3 staking tracking maps
		moveFundsAddStake(ctx, ms, workerAddr, msg)
	} else {
		// add overall staking power of the wallet to the topics stakes
		moveFundsAddStakeToTopics(ctx, ms, workerAddr, msg)
	}

	// add node to the data structures that track the nodes:
	// add node to topicReputers
	// add node to reputers
	workerInfo := state.OffchainNode{
		NodeAddress:  msg.Creator,
		TopicsIds:    msg.TopicsIds,
		LibP2PKey:    msg.LibP2PKey,
		MultiAddress: msg.MultiAddress,
		Owner:        msg.Owner,
		NodeId:       msg.Owner + "|" + msg.LibP2PKey,
	}

	err = ms.k.InsertWorker(ctx, msg.TopicsIds, workerAddr, workerInfo)
	if err != nil {
		return nil, err
	}

	// Return a successful response
	return &state.MsgRegisterWorkerResponse{
		Success: true,
		Message: "Worker node successfully registered",
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
	_, err = checkNodeRegistered(ctx, ms, senderAddr)
	if err != nil {
		return nil, err
	}

	// 2. check the target exists and is registered
	_, err = checkNodeRegistered(ctx, ms, targetAddr)
	if err != nil {
		return nil, err
	}

	// 3. check the sender has enough funds to add the stake
	// bank module does this for us in module SendCoins / subUnlockedCoins so we don't need to check
	// 4. send the funds
	amountInt := cosmosMath.NewIntFromBigInt(msg.Amount.BigInt())
	coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amountInt))
	ms.k.bankKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, state.AlloraStakingModuleName, coins)

	// 5. get target topics registrated
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
		_, err = checkNodeRegistered(ctx, ms, targetAddr)
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
	_, err = checkNodeRegistered(ctx, ms, senderAddr)
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
		_, err = checkNodeRegistered(ctx, ms, targetAddr)
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

		// 5. push to the stake removal object
		stakeRemoval.Placements = append(stakeRemoval.Placements, &state.StakeRemovalPlacement{
			TopicsIds: topicsIds,
			Target:    stakePlacement.Target,
			Amount:    stakePlacement.Amount,
		})
	}
	// 6. if no errors have occured and the removal is valid, add the stake removal to the delayed queue
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
		exists, err := ms.k.TopicExists(ctx, request.TopicId)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, state.ErrInvalidTopicId
		}
		requestId, err := request.GetRequestId()
		if err != nil {
			return nil, err
		}
		// 2. check the request isn't already in the mempool
		exists, err = ms.k.IsRequestInMempool(ctx, request.TopicId, requestId)
		if err != nil {
			return nil, err
		}
		if exists {
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
		err = ms.k.SetFunds(ctx, requestId, request.BidAmount)
		if err != nil {
			return nil, err
		}
		// 12. Write request state into the mempool state
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
type RegistrationMessage interface {
	GetTopicsIds() []uint64
	GetLibP2PKey() string
	GetInitialStake() cosmosMath.Uint
	GetCreator() string
}

func validateRegistrationCommon[M RegistrationMessage](ctx context.Context, ms msgServer, msg M) error {
	// Validate the message contents
	if msg.GetLibP2PKey() == "" {
		return state.ErrLibP2PKeyRequired
	}

	// require funds to be at least greater than the minimum stake
	if msg.GetInitialStake().LT(cosmosMath.NewUint(REQUIRED_MINIMUM_STAKE)) {
		return state.ErrInsufficientStakeToRegister
	}
	return nil
}

func moveFundsAddStake[M RegistrationMessage](ctx context.Context, ms msgServer, nodeAddr sdk.AccAddress, msg M) error {
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

func moveFundsAddStakeToTopics[M RegistrationMessage](ctx context.Context, ms msgServer, nodeAddr sdk.AccAddress, msg M) error {
	totalAddressStake, err := ms.k.GetStakePlacedUponTarget(ctx, nodeAddr)
	if err != nil {
		return err
	}

	// add to topic stake
	err = ms.k.AddStakeToTopics(ctx, msg.GetTopicsIds(), totalAddressStake)
	if err != nil {
		return err
	}

	return nil
}

// checks if a node is registered in the system and if it is,
// returns whether said node is a reputer or a worker
func checkNodeRegistered(ctx context.Context, ms msgServer, node sdk.AccAddress) (NodeExists, error) {
	nodeIsReputer, err := ms.k.IsReputerRegistered(ctx, node)
	if err != nil {
		return isNotFound, err
	}
	if nodeIsReputer {
		return isReputer, nil
	}
	nodeIsWorker, err := ms.k.IsWorkerRegistered(ctx, node)
	if err != nil {
		return isNotFound, err
	}
	if nodeIsWorker {
		return isWorker, nil
	}
	return isNotFound, state.ErrAddressNotRegistered
}
