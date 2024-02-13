package keeper

import (
	"context"
	"fmt"
	"time"

	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	state "github.com/allora-network/allora-chain/x/emissions"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const REQUIRED_MINIMUM_STAKE = 1

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
		Active:           msg.Active,
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
	fmt.Println("Processing updated weights")

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

// Sets a timestamp for a topic
func (ms msgServer) SetLatestInferencesTimestamp(ctx context.Context, msg *state.MsgSetLatestInferencesTimestamp) (*state.MsgSetLatestInferencesTimestampResponse, error) {
	topic := msg.TopicId
	inference_timestamp := msg.InferenceTimestamp

	// Update the map with the new timestamp for the topic
	if err := ms.k.SetLatestInferenceTimestamp(ctx, topic, inference_timestamp); err != nil {
		return nil, err
	}

	// Return an empty response as the operation was successful
	return &state.MsgSetLatestInferencesTimestampResponse{}, nil
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
	fmt.Println("Processing inferences for timestamp: ", actualTimestamp)

	// Update all_inferences
	for topicId, inferences := range groupedInferences {
		inferences := &state.Inferences{
			Inferences: inferences,
		}
		err := ms.k.InsertInferences(ctx, topicId, actualTimestamp, *inferences)
		if err != nil {
			return nil, err
		}
		for _, inference := range inferences.Inferences {
			fmt.Println("Topic: ", topicId, "| Inference: ", inference.Value.String(), "| Worker: ", inference.Worker)
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
	// check the reputer isn't already registered
	reputerAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}
	reputerExists, err := ms.k.IsReputerRegistered(ctx, reputerAddr)
	if err != nil {
		return nil, err
	}
	if reputerExists {
		return nil, state.ErrReputerAlreadyRegistered
	}

	// move the tokens from the creator to the module account
	// then add the stake to the total, topicTotal, and 3 staking tracking maps
	moveFundsAddStake(ctx, ms, reputerAddr, msg)

	// add node to the data structures that track the nodes:
	// add node to topicReputers
	// add node to reputers
	reputerInfo := state.OffchainNode{
		TopicId:      msg.TopicId,
		LibP2PKey:    msg.LibP2PKey,
		MultiAddress: msg.MultiAddress,
	}

	err = ms.k.InsertReputer(ctx, msg.TopicId, reputerAddr, reputerInfo)
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
	// check the worker isn't already registered
	workerAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}
	workerExists, err := ms.k.IsWorkerRegistered(ctx, workerAddr)
	if err != nil {
		return nil, err
	}
	if workerExists {
		return nil, state.ErrWorkerAlreadyRegistered
	}

	// move the tokens from the creator to the module account
	// then add the stake to the total, topicTotal, and 3 staking tracking maps
	moveFundsAddStake(ctx, ms, workerAddr, msg)

	// add node to the data structures that track the nodes:
	// add node to topicReputers
	// add node to reputers
	workerInfo := state.OffchainNode{
		NodeAddress:  msg.Creator,
		TopicId:      msg.TopicId,
		LibP2PKey:    msg.LibP2PKey,
		MultiAddress: msg.MultiAddress,
		Owner:        msg.Owner,
		NodeId:       msg.Owner + "|" + msg.LibP2PKey,
	}

	err = ms.k.InsertWorker(ctx, msg.TopicId, workerAddr, workerInfo)
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
	senderAddr, targetAddr, err := unMarshalSenderAndTargetAddrs(msg.Sender, msg.StakeTarget)
	if err != nil {
		return nil, err
	}
	senderNodeType, err := checkNodeRegistered(ctx, ms, senderAddr)
	if err != nil {
		return nil, err
	}
	// 2. check the target exists and is registered
	targetNodeType, err := checkNodeRegistered(ctx, ms, targetAddr)
	if err != nil {
		return nil, err
	}

	// 3. check target and sender are signed up for the same topic. err == nil if they are
	topicId, err := checkSenderAndTargetSameTopic(ctx, ms, senderAddr, senderNodeType, targetAddr, targetNodeType)
	if err != nil {
		return nil, err
	}

	// 4. check the sender has enough funds to add the stake
	// bank module does this for us in module SendCoins / subUnlockedCoins so we don't need to check
	// 5. send the funds
	amountInt := cosmosMath.NewIntFromBigInt(msg.Amount.BigInt())
	coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amountInt))
	ms.k.bankKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, state.ModuleName, coins)

	// 6. update the stake data structures
	err = ms.k.AddStake(ctx, topicId, msg.Sender, msg.StakeTarget, msg.Amount)
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
	}

	return &state.MsgModifyStakeResponse{}, nil
}

// Function for reputers or workers to call to remove stake from an existing stake position.
func (ms msgServer) RemoveStake(ctx context.Context, msg *state.MsgRemoveStake) (*state.MsgRemoveStakeResponse, error) {
	// 1. check the sender is registered
	senderAddr, targetAddr, err := unMarshalSenderAndTargetAddrs(msg.Sender, msg.StakeTarget)
	if err != nil {
		return nil, err
	}
	senderNodeType, err := checkNodeRegistered(ctx, ms, senderAddr)
	if err != nil {
		return nil, err
	}

	// 2. check the target exists and is registered
	targetNodeType, err := checkNodeRegistered(ctx, ms, targetAddr)
	if err != nil {
		return nil, err
	}

	// 3. check target and sender are signed up for the same topic. err == nil if they are
	topicId, err := checkSenderAndTargetSameTopic(ctx, ms, senderAddr, senderNodeType, targetAddr, targetNodeType)
	if err != nil {
		return nil, err
	}

	// 4. check the sender has enough stake already placed on the target to remove the stake
	stakePlaced, err := ms.k.GetBond(ctx, senderAddr, targetAddr)
	if err != nil {
		return nil, err
	}
	if stakePlaced.LT(msg.Amount) {
		return nil, state.ErrInsufficientStakeToRemove
	}

	// 5. check the module has enough funds to send back to the sender
	// bank module does this for us in module SendCoins / subUnlockedCoins so we don't need to check
	// 6. send the funds
	amountInt := cosmosMath.NewIntFromBigInt(msg.Amount.BigInt())
	coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amountInt))
	ms.k.bankKeeper.SendCoinsFromModuleToAccount(ctx, state.ModuleName, senderAddr, coins)

	// 7. update the stake data structures
	err = ms.k.RemoveStakeFromBond(ctx, topicId, senderAddr, targetAddr, msg.Amount)
	if err != nil {
		return nil, err
	}
	return &state.MsgRemoveStakeResponse{}, nil
}

// Function for a reputer or a worker to pull out all stake from a topic entirely
func (ms msgServer) RemoveAllStake(ctx context.Context, msg *state.MsgRemoveAllStake) (*state.MsgRemoveAllStakeResponse, error) {
	// 1. check the sender is registered
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	fmt.Println(senderAddr.String())
	senderType, err := checkNodeRegistered(ctx, ms, senderAddr)
	if err != nil {
		return nil, err
	}
	// 2. Get the topic id this sender participates in
	var topicId uint64
	if senderType == isReputer {
		nodeInfo, err := ms.k.GetReputer(ctx, senderAddr)
		if err != nil {
			return nil, err
		}
		topicId = nodeInfo.TopicId
	} else {
		nodeInfo, err := ms.k.GetWorker(ctx, senderAddr)
		if err != nil {
			return nil, err
		}
		topicId = nodeInfo.TopicId
	}
	// 2. Get all stake positions for this node
	targets, amounts, err := ms.k.GetAllBondsForDelegator(ctx, senderAddr)
	if err != nil {
		return nil, err
	}
	// 3. Check the module has enough funds to send back to the sender
	//    The bank module does this for us in module SendCoins / subUnlockedCoins so we don't need to check
	// 4. Get sender total stake
	senderStake, err := ms.k.GetDelegatorStake(ctx, senderAddr)
	if err != nil {
		return nil, err
	}
	// 5. Check the sender has enough stake to remove
	if senderStake.IsZero() {
		return nil, state.ErrNoStakeToRemove
	}
	// 6. Send the funds to the sender
	senderStakeInt := cosmosMath.NewIntFromBigInt(senderStake.BigInt())
	coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, senderStakeInt))
	ms.k.bankKeeper.SendCoinsFromModuleToAccount(ctx, state.ModuleName, senderAddr, coins)
	// 7. Update the topicStake data structure (no underflow checks since data comes from chain)
	topicStake, err := ms.k.GetTopicStake(ctx, topicId)
	if err != nil {
		return nil, err
	}
	err = ms.k.SetTopicStake(ctx, topicId, topicStake.Sub(senderStake))
	if err != nil {
		return nil, err
	}
	// 8. Update the totalStake data structure (no underflow checks since data comes from chain)
	totalStake, err := ms.k.GetTotalStake(ctx)
	if err != nil {
		return nil, err
	}
	err = ms.k.SetTotalStake(ctx, totalStake.Sub(senderStake))
	if err != nil {
		return nil, err
	}
	// 9. For every stake position, update the stake data structures
	for i := 0; i < len(targets); i++ {
		target := targets[i]
		amount := amounts[i]
		err = ms.k.RemoveStakeFromBondMissingTotalOrTopicStake(ctx, topicId, senderAddr, target, amount)
		if err != nil {
			return nil, err
		}
	}
	return &state.MsgRemoveAllStakeResponse{}, nil
}

// ########################################
// #           Private Functions          #
// ########################################

// Making common interfaces available to protobuf messages
type RegistrationMessage interface {
	GetTopicId() uint64
	GetLibP2PKey() string
	GetInitialStake() cosmosMath.Uint
	GetCreator() string
}

func validateRegistrationCommon[M RegistrationMessage](ctx context.Context, ms msgServer, msg M) error {
	// Validate the message contents
	if msg.GetLibP2PKey() == "" {
		return state.ErrLibP2PKeyRequired
	}
	// check the topic specified is a valid topic
	numTopics, err := ms.k.GetNumTopics(ctx)
	if err != nil {
		return err
	}
	if msg.GetTopicId() >= numTopics { // topic id is 0 indexed
		return state.ErrInvalidTopicId
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
	err := ms.k.bankKeeper.SendCoinsFromAccountToModule(ctx, nodeAddr, state.ModuleName, amount)
	if err != nil {
		return err
	}

	// for now we will force initial stake deposits to be placed upon oneself.
	// add to total stake
	// add to topic stake
	// add to stakeOwnedByDelegator
	// add to stakePlacement
	// add to stakePlacedUponTarget
	err = ms.k.AddStake(ctx, msg.GetTopicId(), msg.GetCreator(), msg.GetCreator(), msg.GetInitialStake())
	if err != nil {
		return err
	}

	return nil
}

// convert bech32 address strings from protobuf network traffic to sdk.AccAddress
func unMarshalSenderAndTargetAddrs(sender string, target string) (sdk.AccAddress, sdk.AccAddress, error) {
	senderAddr, err := sdk.AccAddressFromBech32(sender)
	if err != nil {
		return nil, nil, err
	}
	targetAddr, err := sdk.AccAddressFromBech32(target)
	if err != nil {
		return nil, nil, err
	}
	return senderAddr, targetAddr, nil
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

// checks if the sender and target are signed up for the same topic
// if they are, returns that topic id
func checkSenderAndTargetSameTopic(
	ctx context.Context,
	ms msgServer,
	senderAddr sdk.AccAddress,
	senderType NodeExists,
	targetAddr sdk.AccAddress,
	targetType NodeExists) (TOPIC_ID, error) {

	var senderTopicId uint64
	if senderType == isReputer {
		nodeInfo, err := ms.k.GetReputer(ctx, senderAddr)
		if err != nil {
			return 0, err
		}
		senderTopicId = nodeInfo.TopicId
	} else {
		nodeInfo, err := ms.k.GetWorker(ctx, senderAddr)
		if err != nil {
			return 0, err
		}
		senderTopicId = nodeInfo.TopicId
	}

	var targetTopicId uint64
	if targetType == isReputer {
		nodeInfo, err := ms.k.GetReputer(ctx, targetAddr)
		if err != nil {
			return 0, err
		}
		targetTopicId = nodeInfo.TopicId
	} else {
		nodeInfo, err := ms.k.GetWorker(ctx, targetAddr)
		if err != nil {
			return 0, err
		}
		targetTopicId = nodeInfo.TopicId
	}

	// only success case
	if senderTopicId == targetTopicId {
		return senderTopicId, nil
	}

	return 0, state.ErrTopicIdOfStakerAndTargetDoNotMatch
}
