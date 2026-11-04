package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

func NewNonceKeeper(
	cdc codec.BinaryCodec,
	sb *collections.SchemaBuilder,
	topicKeeper *TopicKeeper,
	paramsKeeper *ParamsKeeper,
) *NonceKeeper {
	return &NonceKeeper{
		unfulfilledWorkerNonces:  collections.NewMap(sb, types.UnfulfilledWorkerNoncesKey, "unfulfilled_worker_nonces", collections.Uint64Key, codec.CollValue[types.Nonces](cdc)),
		unfulfilledReputerNonces: collections.NewMap(sb, types.UnfulfilledReputerNoncesKey, "unfulfilled_reputer_nonces", collections.Uint64Key, codec.CollValue[types.ReputerRequestNonces](cdc)),
		openWorkerWindows:        collections.NewMap(sb, types.OpenWorkerWindowsKey, "open_worker_windows", collections.Int64Key, codec.CollValue[types.TopicIds](cdc)),
		topicKeeper:              topicKeeper,
		paramsKeeper:             paramsKeeper,
	}
}

type NonceKeeper struct {
	// map of open worker nonce windows for topics on particular block heights
	openWorkerWindows collections.Map[BlockHeight, types.TopicIds]

	// map of (topic) -> unfulfilled nonces
	unfulfilledWorkerNonces collections.Map[TopicId, types.Nonces]

	// map of (topic) -> unfulfilled nonces
	unfulfilledReputerNonces collections.Map[TopicId, types.ReputerRequestNonces]

	// topic keeper
	topicKeeper *TopicKeeper

	// params keeper
	paramsKeeper *ParamsKeeper
}

// GetWorkerWindowTopicIds returns the TopicIds for a given BlockHeight.
// If no TopicIds are found for the BlockHeight, it returns an empty slice.
func (k *NonceKeeper) GetWorkerWindowTopicIds(ctx sdk.Context, height BlockHeight) types.TopicIds {
	topicIds, err := k.openWorkerWindows.Get(ctx, height)
	if err != nil {
		return types.TopicIds{TopicIds: []TopicId{}}
	}
	return topicIds
}

// AddWorkerWindowTopicId appends a new TopicId to the list of TopicIds for a given BlockHeight.
// If no entry exists for the BlockHeight, it creates a new entry with the TopicId.
func (k *NonceKeeper) AddWorkerWindowTopicId(ctx sdk.Context, height BlockHeight, topicId TopicId) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "error validating topic id")
	}
	if err := types.ValidateBlockHeight(height); err != nil {
		return errorsmod.Wrap(err, "error validating block height")
	}
	topicIds, err := k.openWorkerWindows.Get(ctx, height)
	if err != nil {
		topicIds = types.TopicIds{
			TopicIds: []types.TopicId{},
		}
	}
	topicIds.TopicIds = append(topicIds.TopicIds, topicId)
	err = k.openWorkerWindows.Set(ctx, height, topicIds)
	return errorsmod.Wrap(err, "error setting open worker windows")
}

func (k *NonceKeeper) DeleteWorkerWindowBlockHeight(ctx sdk.Context, height BlockHeight) error {
	return k.openWorkerWindows.Remove(ctx, height)
}

// Attempts to fulfill an unfulfilled nonce.
// If the nonce is present, then it is removed from the unfulfilled nonces and this function returns true.
// If the nonce is not present, then the function returns false.
func (k *NonceKeeper) FulfillWorkerNonce(ctx context.Context, topicId TopicId, nonce *types.Nonce) (bool, error) {
	if err := types.ValidateTopicId(topicId); err != nil {
		return false, errorsmod.Wrap(err, "error validating topic id")
	}
	if nonce == nil {
		return false, errors.New("nil worker nonce provided")
	}
	if err := nonce.Validate(); err != nil {
		return false, errorsmod.Wrap(err, "error validating nonce")
	}
	unfulfilledNonces, err := k.GetUnfulfilledWorkerNonces(ctx, topicId)
	if err != nil {
		return false, errorsmod.Wrap(err, "error getting unfulfilled worker nonces")
	}

	// Check if the nonce is present in the unfulfilled nonces
	for i, n := range unfulfilledNonces.Nonces {
		if n == nil {
			return false, errorsmod.Wrapf(types.ErrInvalidValue, "nil worker nonce entry in stored unfulfilled nonces for topic %d", topicId)
		}
		if n.BlockHeight == nonce.BlockHeight {
			// Remove the nonce from the unfulfilled nonces
			unfulfilledNonces.Nonces = append(unfulfilledNonces.Nonces[:i], unfulfilledNonces.Nonces[i+1:]...)
			if err := unfulfilledNonces.Validate(); err != nil {
				return false, errorsmod.Wrap(err, "error validating unfulfilled nonces")
			}
			err := k.unfulfilledWorkerNonces.Set(ctx, topicId, unfulfilledNonces)
			if err != nil {
				return false, errorsmod.Wrap(err, "error setting unfulfilled worker nonces")
			}
			return true, nil
		}
	}

	// If the nonce is not present in the unfulfilled nonces
	return false, nil
}

// Attempts to fulfill an unfulfilled nonce.
// If the nonce is present, then it is removed from the unfulfilled nonces and this function returns true.
// If the nonce is not present, then the function returns false.
func (k *NonceKeeper) FulfillReputerNonce(ctx context.Context, topicId TopicId, nonce *types.Nonce) (bool, error) {
	if err := types.ValidateTopicId(topicId); err != nil {
		return false, errorsmod.Wrap(err, "error validating topic id")
	}
	if nonce == nil {
		return false, errors.New("nil reputer nonce provided")
	}
	if err := nonce.Validate(); err != nil {
		return false, errorsmod.Wrap(err, "error validating nonce")
	}
	unfulfilledNonces, err := k.GetUnfulfilledReputerNonces(ctx, topicId)
	if err != nil {
		return false, errorsmod.Wrap(err, "error getting unfulfilled reputer nonces")
	}

	// Check if the nonce is present in the unfulfilled nonces
	for i, n := range unfulfilledNonces.Nonces {
		if n == nil || n.ReputerNonce == nil {
			return false, errorsmod.Wrapf(types.ErrInvalidValue, "nil reputer nonce entry in stored unfulfilled nonces for topic %d", topicId)
		}
		if n.ReputerNonce.BlockHeight == nonce.BlockHeight {
			// Remove the nonce from the unfulfilled nonces
			unfulfilledNonces.Nonces = append(unfulfilledNonces.Nonces[:i], unfulfilledNonces.Nonces[i+1:]...)
			if err := unfulfilledNonces.Validate(); err != nil {
				return false, errorsmod.Wrap(err, "error validating unfulfilled reputer nonces")
			}
			err := k.unfulfilledReputerNonces.Set(ctx, topicId, unfulfilledNonces)
			if err != nil {
				return false, errorsmod.Wrap(err, "error setting unfulfilled reputer nonces")
			}
			return true, nil
		}
	}

	// If the nonce is not present in the unfulfilled nonces
	return false, nil
}

// True if nonce is unfulfilled, false otherwise.
func (k *NonceKeeper) IsWorkerNonceUnfulfilled(ctx context.Context, topicId TopicId, nonce *types.Nonce) (isUnfulfilled bool, err error) {
	if nonce == nil {
		return false, errors.New("nil worker nonce provided")
	}
	// Get the latest unfulfilled nonces
	unfulfilledNonces, err := k.GetUnfulfilledWorkerNonces(ctx, topicId)
	if err != nil {
		return false, errorsmod.Wrap(err, "error getting unfulfilled worker nonces")
	}

	// Check if the nonce is present in the unfulfilled nonces
	for _, n := range unfulfilledNonces.Nonces {
		if n == nil {
			return false, errorsmod.Wrapf(types.ErrInvalidValue, "nil worker nonce entry in stored unfulfilled nonces for topic %d", topicId)
		}
		if n.BlockHeight == nonce.BlockHeight {
			return true, nil
		}
	}

	return false, nil
}

// True if nonce is unfulfilled, false otherwise.
func (k *NonceKeeper) IsReputerNonceUnfulfilled(ctx context.Context, topicId TopicId, nonce *types.Nonce) (isUnfulfilled bool, err error) {
	if nonce == nil {
		return false, errors.New("nil reputer nonce provided")
	}
	// Get the latest unfulfilled nonces
	unfulfilledNonces, err := k.GetUnfulfilledReputerNonces(ctx, topicId)
	if err != nil {
		return false, errorsmod.Wrap(err, "error getting unfulfilled reputer nonces")
	}

	// Check if the nonce is present in the unfulfilled nonces
	for _, n := range unfulfilledNonces.Nonces {
		if n == nil || n.ReputerNonce == nil {
			return false, errorsmod.Wrapf(types.ErrInvalidValue, "nil reputer nonce entry in stored unfulfilled nonces for topic %d", topicId)
		}
		if n.ReputerNonce.BlockHeight == nonce.BlockHeight {
			return true, nil
		}
	}

	return false, nil
}

// Adds a nonce to the unfulfilled nonces for the topic if it is not yet added (idempotent).
// If the max number of nonces is reached, then the function removes the oldest nonce and adds the new nonce.
func (k *NonceKeeper) AddWorkerNonce(ctx context.Context, topicId TopicId, nonce *types.Nonce) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "error validating topic id")
	}
	if nonce == nil {
		return errors.New("nil worker nonce provided")
	}
	if err := nonce.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating nonce")
	}
	nonces, err := k.GetUnfulfilledWorkerNonces(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "error getting unfulfilled worker nonces")
	}

	// Check that input nonce is not already contained in the nonces of this topic
	for _, n := range nonces.Nonces {
		if n == nil {
			return errorsmod.Wrapf(types.ErrInvalidValue, "nil worker nonce entry in stored unfulfilled nonces for topic %d", topicId)
		}
		if n.BlockHeight == nonce.BlockHeight {
			return nil
		}
	}
	nonces.Nonces = append([]*types.Nonce{nonce}, nonces.Nonces...)

	moduleParams, err := k.paramsKeeper.GetParams(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "error getting module params")
	}
	maxUnfulfilledRequests := moduleParams.MaxUnfulfilledWorkerRequests

	lenNonces := uint64(len(nonces.Nonces))
	if lenNonces > maxUnfulfilledRequests {
		nonces.Nonces = nonces.Nonces[:maxUnfulfilledRequests]
	}

	if err := nonces.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating unfulfilled worker nonces")
	}
	return k.unfulfilledWorkerNonces.Set(ctx, topicId, nonces)
}

// Adds a nonce to the unfulfilled nonces for the topic if it is not yet added (idempotent).
// If the max number of nonces is reached, then the function removes the oldest nonce and adds the new nonce.
func (k *NonceKeeper) AddReputerNonce(ctx context.Context, topicId TopicId, nonce *types.Nonce) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "error validating topic id")
	}
	if nonce == nil {
		return errors.New("nil reputer's nonce provided")
	}
	if err := nonce.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating nonce")
	}
	nonces, err := k.GetUnfulfilledReputerNonces(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "error getting unfulfilled reputer nonces")
	}

	// Check that input nonce is not already contained in the nonces of this topic
	// nor that the `associatedWorkerNonce` is already associated with a worker request
	for _, n := range nonces.Nonces {
		if n == nil || n.ReputerNonce == nil {
			return errorsmod.Wrapf(types.ErrInvalidValue, "nil reputer nonce entry in stored unfulfilled nonces for topic %d", topicId)
		}
		// Do nothing if nonce is already in the list
		if n.ReputerNonce.BlockHeight == nonce.BlockHeight {
			return nil
		}
	}
	reputerRequestNonce := &types.ReputerRequestNonce{
		ReputerNonce: nonce,
	}
	nonces.Nonces = append([]*types.ReputerRequestNonce{reputerRequestNonce}, nonces.Nonces...)

	moduleParams, err := k.paramsKeeper.GetParams(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "error getting module params")
	}
	maxUnfulfilledRequests := moduleParams.MaxUnfulfilledReputerRequests
	lenNonces := uint64(len(nonces.Nonces))
	if lenNonces > maxUnfulfilledRequests {
		nonces.Nonces = nonces.Nonces[:maxUnfulfilledRequests]
	}
	if err := nonces.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating unfulfilled reputer nonces")
	}
	return k.unfulfilledReputerNonces.Set(ctx, topicId, nonces)
}

func (k *NonceKeeper) GetUnfulfilledWorkerNonces(ctx context.Context, topicId TopicId) (types.Nonces, error) {
	nonces, err := k.unfulfilledWorkerNonces.Get(ctx, topicId)
	if errors.Is(err, collections.ErrNotFound) {
		return types.Nonces{Nonces: nil}, nil
	} else if err != nil {
		return types.Nonces{Nonces: nil}, errorsmod.Wrap(err, "error getting unfulfilled worker nonces")
	}
	return nonces, nil
}

func (k *NonceKeeper) GetUnfulfilledReputerNonces(ctx context.Context, topicId TopicId) (types.ReputerRequestNonces, error) {
	nonces, err := k.unfulfilledReputerNonces.Get(ctx, topicId)
	if errors.Is(err, collections.ErrNotFound) {
		return types.ReputerRequestNonces{Nonces: nil}, nil
	} else if err != nil {
		return types.ReputerRequestNonces{Nonces: nil}, errorsmod.Wrap(err, "error getting unfulfilled reputer nonces")
	}
	return nonces, nil
}

// GetOpenReputerSubmissionWindows returns only the reputer nonces that are currently open for submission.
// A nonce is considered open if the current block height is within its submission window.
func (k *NonceKeeper) GetOpenReputerSubmissionWindows(ctx context.Context, topicId TopicId) (types.ReputerRequestNonces, error) {
	// Get current block height
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentBlockHeight := sdkCtx.BlockHeight()

	// Get topic to access window parameters
	topic, err := k.topicKeeper.GetTopic(ctx, topicId)
	if err != nil {
		return types.ReputerRequestNonces{Nonces: nil}, errorsmod.Wrapf(err, "error getting topic %v", topicId)
	}

	// Get all unfulfilled reputer nonces
	unfulfilledNonces, err := k.GetUnfulfilledReputerNonces(ctx, topicId)
	if err != nil {
		return types.ReputerRequestNonces{Nonces: nil}, errorsmod.Wrap(err, "error getting unfulfilled reputer nonces")
	}

	// Filter nonces that are within the open submission window
	openNonces := []*types.ReputerRequestNonce{}
	for _, nonce := range unfulfilledNonces.Nonces {
		if nonce == nil {
			return types.ReputerRequestNonces{Nonces: nil}, errorsmod.Wrapf(types.ErrInvalidValue, "nil reputer nonce entry in stored unfulfilled nonces for topic %d", topicId)
		}
		isOpen, err := BlockWithinReputerSubmissionWindowOfNonce(topic, *nonce, currentBlockHeight)
		if err != nil {
			return types.ReputerRequestNonces{Nonces: nil}, errorsmod.Wrap(err, "error checking reputer submission window")
		}
		if isOpen {
			openNonces = append(openNonces, nonce)
		}
	}

	return types.ReputerRequestNonces{Nonces: openNonces}, nil
}

// GetOpenWorkerSubmissionWindows returns only the worker nonces that are currently open for submission.
// A nonce is considered open if the current block height is within its submission window.
func (k *NonceKeeper) GetOpenWorkerSubmissionWindows(ctx context.Context, topicId TopicId) (types.Nonces, error) {
	// Get current block height
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentBlockHeight := sdkCtx.BlockHeight()

	// Get topic to access window parameters
	topic, err := k.topicKeeper.GetTopic(ctx, topicId)
	if err != nil {
		return types.Nonces{Nonces: nil}, errorsmod.Wrapf(err, "error getting topic %v", topicId)
	}

	// Get all unfulfilled worker nonces
	unfulfilledNonces, err := k.GetUnfulfilledWorkerNonces(ctx, topicId)
	if err != nil {
		return types.Nonces{Nonces: nil}, errorsmod.Wrap(err, "error getting unfulfilled worker nonces")
	}

	// Filter nonces that are within the open submission window
	openNonces := []*types.Nonce{}
	for _, nonce := range unfulfilledNonces.Nonces {
		if nonce == nil {
			return types.Nonces{Nonces: nil}, errorsmod.Wrapf(types.ErrInvalidValue, "nil worker nonce entry in stored unfulfilled nonces for topic %d", topicId)
		}
		isOpen, err := BlockWithinWorkerSubmissionWindowOfNonce(topic, *nonce, currentBlockHeight)
		if err != nil {
			return types.Nonces{Nonces: nil}, errorsmod.Wrap(err, "error checking worker submission window")
		}
		if isOpen {
			openNonces = append(openNonces, nonce)
		}
	}

	return types.Nonces{Nonces: openNonces}, nil
}

func (k *NonceKeeper) DeleteUnfulfilledWorkerNonces(ctx context.Context, topicId TopicId) error {
	return k.unfulfilledWorkerNonces.Remove(ctx, topicId)
}

func (k *NonceKeeper) DeleteUnfulfilledReputerNonces(ctx context.Context, topicId TopicId) error {
	return k.unfulfilledReputerNonces.Remove(ctx, topicId)
}

func (k *NonceKeeper) PruneWorkerNonces(ctx context.Context, topicId uint64, blockHeightThreshold int64) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "error validating topic id")
	}
	nonces, err := k.GetUnfulfilledWorkerNonces(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "error getting unfulfilled worker nonces")
	}

	// Filter Nonces based on block_height
	filteredNonces := make([]*types.Nonce, 0)
	for _, nonce := range nonces.Nonces {
		if nonce == nil {
			return errorsmod.Wrapf(types.ErrInvalidValue, "nil worker nonce entry in stored unfulfilled nonces for topic %d", topicId)
		}
		if nonce.BlockHeight >= blockHeightThreshold {
			filteredNonces = append(filteredNonces, nonce)
		}
	}

	// Update nonces in the map
	nonces.Nonces = filteredNonces
	if err := nonces.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating unfulfilled worker nonces")
	}
	if err := k.unfulfilledWorkerNonces.Set(ctx, topicId, nonces); err != nil {
		return errorsmod.Wrap(err, "error setting unfulfilled worker nonces")
	}

	return nil
}

func (k *NonceKeeper) PruneReputerNonces(ctx context.Context, topicId uint64, blockHeightThreshold int64) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "error validating topic id")
	}
	nonces, err := k.GetUnfulfilledReputerNonces(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "error getting unfulfilled reputer nonces")
	}

	// Filter Nonces based on block_height
	filteredNonces := make([]*types.ReputerRequestNonce, 0)
	for _, nonce := range nonces.Nonces {
		if nonce == nil || nonce.ReputerNonce == nil {
			return errorsmod.Wrapf(types.ErrInvalidValue, "nil reputer nonce entry in stored unfulfilled nonces for topic %d", topicId)
		}
		if nonce.ReputerNonce.BlockHeight >= blockHeightThreshold {
			filteredNonces = append(filteredNonces, nonce)
		}
	}

	if len(filteredNonces) == 0 {
		return k.unfulfilledReputerNonces.Remove(ctx, topicId)
	}

	// Update nonces in the map
	nonces.Nonces = filteredNonces
	if err := nonces.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating unfulfilled reputer nonces")
	}
	if err := k.unfulfilledReputerNonces.Set(ctx, topicId, nonces); err != nil {
		return errorsmod.Wrap(err, "error setting unfulfilled reputer nonces")
	}

	return nil
}
