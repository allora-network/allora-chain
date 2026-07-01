package keeper

import (
	"context"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

// labelSlot maps a 1-based epoch-label id (assigned sequentially by
// RegisterEpochLabels) to its 0-based index in a dense inference Values vector.
// The id space and this mapping are owned by the epoch label registry, so this
// primitive lives here and is shared by NormalizeInputInference and close-time
// compaction.
func labelSlot(id LabelId) int { return int(id) - 1 }

// SetEpochLabelRegistry overwrites the registry for a topic epoch.
func (k *TopicKeeper) SetEpochLabelRegistry(
	ctx context.Context,
	registry types.EpochLabelRegistry,
) error {
	//nolint:gosec // EpochId was originally produced from a validated non-negative block height.
	nonce := types.BlockHeight(registry.EpochId)
	topic, err := k.GetTopic(ctx, registry.TopicId)
	if err != nil {
		return errorsmod.Wrap(err, "failed to get topic for epoch label registry validation")
	}
	params, err := k.paramsKeeper.GetParams(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "failed to get params for epoch label registry validation")
	}
	if err := validateEpochLabelRegistry(
		registry.TopicId,
		topic.LabelCaseSensitive,
		nonce,
		registry,
		params.MaxCanonicalLabelByteLength,
	); err != nil {
		return err
	}
	return k.topicLabelRegistry.Set(ctx, collections.Join(registry.TopicId, nonce), registry)
}

// validateEpochLabelRegistry guards the stored registry invariant for a topic epoch.
// It ensures identity, label IDs, canonical names, and uniqueness stay consistent.
func validateEpochLabelRegistry(
	topicID types.TopicId,
	labelCaseSensitive bool,
	nonce types.BlockHeight,
	registry types.EpochLabelRegistry,
	maxLabelBytes uint64,
) error {
	if err := types.ValidateTopicId(topicID); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateBlockHeight(nonce); err != nil {
		return errorsmod.Wrap(err, "nonce block height validation failed")
	}
	if registry.TopicId != topicID {
		return errorsmod.Wrapf(sdkerrors.ErrLogic, "registry topic mismatch: got %d expected %d", registry.TopicId, topicID)
	}
	if registry.EpochId != uint64(nonce) {
		return errorsmod.Wrapf(sdkerrors.ErrLogic, "registry epoch mismatch: got %d expected %d", registry.EpochId, nonce)
	}
	// NOTE: the registry size cap (Params.MaxEpochLabelRegistrySize) is enforced
	// only at the growth point (RegisterEpochLabels). It is intentionally NOT
	// re-checked here: this validation runs on already-persisted registries
	// (read/close/import), and re-checking a mutable governance param against
	// historical data would let a lowered cap retroactively invalidate valid
	// state (e.g. aborting genesis import)
	seen := make(map[string]struct{}, len(registry.Labels))
	for i, lbl := range registry.Labels {
		if lbl == nil {
			return errorsmod.Wrapf(sdkerrors.ErrLogic, "registry label at index %d is nil", i)
		}
		//nolint:gosec // registry length is bounded at registration by Params.MaxEpochLabelRegistrySize.
		expectedID := LabelId(i + 1)
		if lbl.Id != expectedID {
			return errorsmod.Wrapf(sdkerrors.ErrLogic, "registry label %q has id %d expected %d", lbl.Name, lbl.Id, expectedID)
		}
		if err := types.EnsureCanonicalLabelName(lbl.Name, maxLabelBytes, labelCaseSensitive); err != nil {
			return errorsmod.Wrapf(err, "registry label at index %d is invalid", i)
		}
		if _, ok := seen[lbl.Name]; ok {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "registry label at index %d is duplicated: %q", i, lbl.Name)
		}
		seen[lbl.Name] = struct{}{}
	}
	return nil
}
