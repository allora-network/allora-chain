package keeper

import (
	"context"
	"errors"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

// CloseNonceFn closes a worker or reputer nonce via the legacy height-keyed path.
// Injected from outside the keeper package to avoid an import cycle with actor_utils.
type CloseNonceFn func(ctx sdk.Context, topic types.Topic, nonce types.Nonce) error

type epochCloseHandlers struct {
	closeWorker  CloseNonceFn
	closeReputer CloseNonceFn
}

// SetEpochCloseHandlers wires CloseWorkerNonce / CloseReputerNonce implementations used by
// FSM transition side effects. Must be called after NewKeeper (module/testutil).
func (k *Keeper) SetEpochCloseHandlers(closeWorker, closeReputer CloseNonceFn) {
	if k.epochCloseHandlers == nil {
		k.epochCloseHandlers = &epochCloseHandlers{}
	}
	k.epochCloseHandlers.closeWorker = closeWorker
	k.epochCloseHandlers.closeReputer = closeReputer
}

func (k *Keeper) openWorkerWindow(ctx context.Context, epoch types.Epoch) error {
	topic, err := k.topicKeeper.GetTopic(ctx, epoch.TopicId)
	if err != nil {
		return err
	}

	legacyNonce := epoch.LegacyNonce()
	if err := k.nonceKeeper.AddWorkerNonce(ctx, epoch.TopicId, &legacyNonce); err != nil {
		return errorsmod.Wrap(err, "open worker window: add worker nonce")
	}

	windowEndBlock := legacyNonce.BlockHeight + topic.WorkerSubmissionWindow
	types.EmitNewWorkerSubmissionWindowOpenedEvent(ctx, epoch.TopicId, legacyNonce.BlockHeight, windowEndBlock)
	return nil
}

func (k *Keeper) closeWorkerWindow(ctx context.Context, epoch types.Epoch) error {
	if k.epochCloseHandlers == nil || k.epochCloseHandlers.closeWorker == nil {
		return fmt.Errorf("close worker window: close handler not configured")
	}

	topic, err := k.topicKeeper.GetTopic(ctx, epoch.TopicId)
	if err != nil {
		return err
	}

	legacyNonce := epoch.LegacyNonce()
	err = k.epochCloseHandlers.closeWorker(sdk.UnwrapSDKContext(ctx), topic, legacyNonce)
	if err != nil && !errors.Is(err, types.ErrNoQualifiedInferers) {
		return errorsmod.Wrap(err, "close worker window")
	}
	return nil
}

func (k *Keeper) openReputerWindow(ctx context.Context, epoch types.Epoch) error {
	topic, err := k.topicKeeper.GetTopic(ctx, epoch.TopicId)
	if err != nil {
		return err
	}

	legacyNonce := epoch.LegacyNonce()
	// CloseWorkerNonce normally adds the reputer nonce; ensure it exists if worker
	// close soft-failed (e.g. no qualified inferers).
	if err := k.nonceKeeper.AddReputerNonce(ctx, epoch.TopicId, &legacyNonce); err != nil {
		return errorsmod.Wrap(err, "open reputer window: add reputer nonce")
	}

	extraLag := types.TopicExtraLag(topic)
	windowEndBlock := legacyNonce.BlockHeight + topic.GroundTruthLag + extraLag + topic.EpochLength
	types.EmitNewReputerSubmissionWindowOpenedEvent(ctx, epoch.TopicId, legacyNonce.BlockHeight, windowEndBlock)
	return nil
}

func (k *Keeper) closeReputerWindow(ctx context.Context, epoch types.Epoch) error {
	if k.epochCloseHandlers == nil || k.epochCloseHandlers.closeReputer == nil {
		return fmt.Errorf("close reputer window: close handler not configured")
	}

	topic, err := k.topicKeeper.GetTopic(ctx, epoch.TopicId)
	if err != nil {
		return err
	}

	legacyNonce := epoch.LegacyNonce()
	err = k.epochCloseHandlers.closeReputer(sdk.UnwrapSDKContext(ctx), topic, legacyNonce)
	if err != nil {
		// Soft-fail empty / already-closed sets so the FSM can still complete during
		// the parallel-run period when submissions may be absent.
		if errors.Is(err, sdkerrors.ErrNotFound) ||
			errors.Is(err, types.ErrNotFound) ||
			errors.Is(err, types.ErrUnfulfilledNonceNotFound) {
			sdk.UnwrapSDKContext(ctx).Logger().Info(
				"close reputer window soft-failed; continuing epoch lifecycle",
				"topicId", epoch.TopicId,
				"legacyNonce", legacyNonce.BlockHeight,
				"error", err,
			)
			return nil
		}
		return errorsmod.Wrap(err, "close reputer window")
	}
	return nil
}

func (k *Keeper) completeEpoch(ctx context.Context, epoch types.Epoch) error {
	// Losses/weights/TopicRewardNonce are produced in closeReputerWindow via CloseReputerNonce
	// when submissions exist. During the parallel EndBlocker period, coin payout + prune remain
	// in rewards.EmitRewards so we do not delete TopicRewardNonce here (that would race EndBlocker).
	// Mark the topic rewardable as a signal for reward selection.
	if err := k.topicKeeper.SetRewardableTopic(ctx, epoch.TopicId); err != nil {
		return errorsmod.Wrap(err, "complete epoch: set rewardable topic")
	}
	return nil
}

func (k *Keeper) cancelEpoch(ctx context.Context, epoch types.Epoch) error {
	legacyNonce := epoch.LegacyNonce()

	workerUnfulfilled, err := k.nonceKeeper.IsWorkerNonceUnfulfilled(ctx, epoch.TopicId, &legacyNonce)
	if err != nil {
		return err
	}
	if workerUnfulfilled {
		if _, err := k.nonceKeeper.FulfillWorkerNonce(ctx, epoch.TopicId, &legacyNonce); err != nil {
			return errorsmod.Wrap(err, "cancel epoch: fulfill worker nonce")
		}
		types.EmitNewWorkerSubmissionWindowClosedEvent(ctx, epoch.TopicId, legacyNonce.BlockHeight)
	}

	reputerUnfulfilled, err := k.nonceKeeper.IsReputerNonceUnfulfilled(ctx, epoch.TopicId, &legacyNonce)
	if err != nil {
		return err
	}
	if reputerUnfulfilled {
		if _, err := k.nonceKeeper.FulfillReputerNonce(ctx, epoch.TopicId, &legacyNonce); err != nil {
			return errorsmod.Wrap(err, "cancel epoch: fulfill reputer nonce")
		}
		types.EmitNewReputerSubmissionWindowClosedEvent(ctx, epoch.TopicId, legacyNonce.BlockHeight)
	}

	return k.unscheduleEpochLifecycle(ctx, epoch)
}
