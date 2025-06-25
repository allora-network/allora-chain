package keeper

import (
	"context"
	"strconv"
	"time"

	"cosmossdk.io/math"
	actorutils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	schedulertypes "github.com/allora-network/allora-chain/x/scheduler/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
)

const (
	TaskUnbondStake      = emissionstypes.ModuleName + "#unbond_stake"
	TaskCloseWorkerNonce = emissionstypes.ModuleName + "#close_worker_nonce"
)

func (k *Keeper) TaskSpecs() schedulertypes.TaskSpecs {
	return schedulertypes.TaskSpecs{
		{
			Name:     TaskUnbondStake,
			Priority: 100,
			ArgsType: (*emissionstypes.UnbondStakeArgs)(nil),
			TaskHandler: schedulertypes.NewTaskHandlerFromFuncs(
				nil,
				func(ctx context.Context, _ schedulertypes.TaskID, args proto.Message, _ uint64) error {
					unbondArgs := args.(*emissionstypes.UnbondStakeArgs)
					return nil
				},
			),
		},
		{
			Name:     TaskCloseWorkerNonce,
			Priority: 100,
			ArgsType: (*emissionstypes.CloseWorkerNonceArgs)(nil),
			TaskHandler: schedulertypes.NewTaskHandlerFromFuncs(
				nil,
				func(ctx context.Context, _ schedulertypes.TaskID, args proto.Message, _ uint64) error {
					closeArgs := args.(*emissionstypes.CloseWorkerNonceArgs)

					topic, err := k.GetTopic(ctx, closeArgs.TopicId)
					if err != nil {
						return err
					}

					return actorutils.CloseWorkerNonce(k, sdk.UnwrapSDKContext(ctx), topic, *closeArgs.Nonce)
				},
			),
		},
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
	return k.schedulerKeeper.ScheduleTask(
		ctx,
		TaskUnbondStake,
		schedulertypes.TaskID(strconv.FormatUint(topicID, 10)),
		&emissionstypes.CloseWorkerNonceArgs{TopicId: topicID, Nonce: nonce},
		when,
	)
}
