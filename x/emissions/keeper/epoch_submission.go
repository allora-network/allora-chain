package keeper

import (
	"context"
	"time"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

// GetEpochByLegacyNonce returns the live epoch for a topic whose StartBlockHeight matches
// the legacy BlockHeight nonce. Completed/cancelled epochs are not in the store.
func (k *Keeper) GetEpochByLegacyNonce(
	ctx context.Context,
	topicID TopicId,
	legacyNonce types.Nonce,
) (types.Epoch, bool, error) {
	rng := collections.NewPrefixedPairRange[TopicId, types.NonceV2](topicID)
	iter, err := k.epochs.Iterate(ctx, rng)
	if err != nil {
		return types.Epoch{}, false, err
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		epoch, err := iter.Value()
		if err != nil {
			return types.Epoch{}, false, err
		}
		if epoch.StartBlockHeight == legacyNonce.BlockHeight {
			return epoch, true, nil
		}
	}
	return types.Epoch{}, false, nil
}

// TimeWithinWindow reports whether now is inside [OpenAt, CloseAt] inclusive.
// Matches the inclusive bounds used by height-based submission windows.
func TimeWithinWindow(now time.Time, window *types.Window) bool {
	if window == nil {
		return false
	}
	return !now.Before(window.OpenAt) && !now.After(window.CloseAt)
}

// CheckWorkerSubmissionWindow enforces Epoch wall-clock / FSM state when a live epoch
// matches the legacy nonce. found is false when no live epoch matches (caller should
// use the height-based window check).
func (k *Keeper) CheckWorkerSubmissionWindow(
	ctx context.Context,
	topicID TopicId,
	legacyNonce types.Nonce,
) (found bool, err error) {
	epoch, found, err := k.GetEpochByLegacyNonce(ctx, topicID, legacyNonce)
	if err != nil || !found {
		return found, err
	}

	if epoch.State != types.EpochState_WORKER_SUBMISSION {
		return true, errorsmod.Wrapf(
			types.ErrWorkerNonceWindowNotAvailable,
			"epoch not in worker submission state for topic %d legacy nonce %d (state=%s)",
			topicID, legacyNonce.BlockHeight, epoch.State,
		)
	}

	now := sdk.UnwrapSDKContext(ctx).BlockTime()
	if !TimeWithinWindow(now, epoch.WorkerSubmissionWindow) {
		return true, errorsmod.Wrapf(
			types.ErrWorkerNonceWindowNotAvailable,
			"worker wall-clock window not open for topic %d legacy nonce %d: now=%s open=%s close=%s",
			topicID, legacyNonce.BlockHeight, now.UTC(),
			epoch.WorkerSubmissionWindow.OpenAt.UTC(),
			epoch.WorkerSubmissionWindow.CloseAt.UTC(),
		)
	}
	return true, nil
}

// CheckReputerSubmissionWindow enforces Epoch wall-clock / FSM state when a live epoch
// matches the legacy nonce. found is false when no live epoch matches (caller should
// use the height-based window check).
func (k *Keeper) CheckReputerSubmissionWindow(
	ctx context.Context,
	topicID TopicId,
	legacyNonce types.Nonce,
) (found bool, err error) {
	epoch, found, err := k.GetEpochByLegacyNonce(ctx, topicID, legacyNonce)
	if err != nil || !found {
		return found, err
	}

	if epoch.State != types.EpochState_REPUTER_SUBMISSION {
		return true, errorsmod.Wrapf(
			types.ErrReputerNonceWindowNotAvailable,
			"epoch not in reputer submission state for topic %d legacy nonce %d (state=%s)",
			topicID, legacyNonce.BlockHeight, epoch.State,
		)
	}

	now := sdk.UnwrapSDKContext(ctx).BlockTime()
	if !TimeWithinWindow(now, epoch.ReputerSubmissionWindow) {
		return true, errorsmod.Wrapf(
			types.ErrReputerNonceWindowNotAvailable,
			"reputer wall-clock window not open for topic %d legacy nonce %d: now=%s open=%s close=%s",
			topicID, legacyNonce.BlockHeight, now.UTC(),
			epoch.ReputerSubmissionWindow.OpenAt.UTC(),
			epoch.ReputerSubmissionWindow.CloseAt.UTC(),
		)
	}
	return true, nil
}
