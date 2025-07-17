package keeper

import (
	"context"
	"strconv"
	"time"

	"cosmossdk.io/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	schedulertypes "github.com/allora-network/allora-chain/x/scheduler/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	TaskUnbondStake      = emissionstypes.ModuleName + ":unbond_stake"
	TaskCloseWorkerNonce = emissionstypes.ModuleName + ":close_worker_nonce"
)

func (k *Keeper) TaskHandlers() schedulertypes.TaskHandlers {
	return schedulertypes.TaskHandlers{
		schedulertypes.NewTaskHandler(
			TaskUnbondStake,
			nil,
			nil,
			func(ctx context.Context, id schedulertypes.TaskID, args *emissionstypes.UnbondStakeArgs, runCount uint64) error {
				return nil
			},
		),
		schedulertypes.NewTaskHandler(
			TaskCloseWorkerNonce,
			nil,
			nil,
			func(ctx context.Context, id schedulertypes.TaskID, args *emissionstypes.CloseWorkerNonceArgs, runCount uint64) error {
				return nil
			},
		),
	}
}

func (k *Keeper) ScheduleUnbondStakeTask(
	ctx context.Context,
	delegator sdk.AccAddress,
	amount math.Int,
	when time.Time,
) error {
	return k.schedulerKeeper.ScheduleTask(
		ctx,
		TaskUnbondStake,
		schedulertypes.TaskID(delegator.String()),
		&emissionstypes.UnbondStakeArgs{Delegator: delegator.String(), Amount: amount},
		when,
	)
}

func (k *Keeper) ScheduleCloseWorkerNonceTask(
	ctx context.Context,
	topicID emissionstypes.TopicId,
	nonce *emissionstypes.Nonce,
	when time.Time,
) error {
	return k.schedulerKeeper.SchedulePeriodicTask(
		ctx,
		TaskCloseWorkerNonce,
		schedulertypes.TaskID(strconv.FormatUint(topicID, 10)),
		&emissionstypes.CloseWorkerNonceArgs{TopicId: topicID, Nonce: nonce},
		when,
		time.Hour*24,
	)
}
