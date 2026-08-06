package keeper

import (
	"context"
	"sort"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	schedulertypes "github.com/allora-network/allora-chain/x/scheduler/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// arbitrateByTopicWeight keeps at most MaxActiveTopicsPerBlock distinct topics'
// tasks executing this BeginBlock. Remaining tasks are rescheduled for the next
// BeginBlock (explicit ScheduleAt(now); never cancelled/dropped).
//
// Selection matches EndBlocker topic skimming: highest weight first, topic ID
// ascending on ties. Weights are computed read-only via GetTopicWeightFromTopicId
// (no drip/inactivation). Scheduler BeginBlock cannot wait for EndBlocker weight
// updates in the same block; this uses current stake/revenue-derived weights.
func (k *Keeper) arbitrateByTopicWeight(
	ctx context.Context,
	taskTopicIDs map[schedulertypes.TaskID]TopicId,
) (map[schedulertypes.TaskID]schedulertypes.ArbitrageDecision, error) {
	if len(taskTopicIDs) == 0 {
		return nil, nil //nolint:nilnil // no decisions; all tasks execute
	}

	params, err := k.paramsKeeper.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	maxTopics := params.MaxActiveTopicsPerBlock

	weights := make(map[TopicId]*alloraMath.Dec)
	for _, topicID := range taskTopicIDs { //nolint:maprange // collect unique topics before sorting
		if _, ok := weights[topicID]; ok {
			continue
		}
		weight, err := k.topicKeeper.GetTopicWeightFromTopicId(ctx, topicID)
		if err != nil {
			return nil, err
		}
		w := weight
		weights[topicID] = &w
	}

	if uint64(len(weights)) <= maxTopics {
		return nil, nil //nolint:nilnil // under cap; all execute
	}

	topTopics := skimTopTopicIDsByWeightDesc(weights, maxTopics)
	topSet := make(map[TopicId]struct{}, len(topTopics))
	for _, topicID := range topTopics {
		topSet[topicID] = struct{}{}
	}

	now := sdk.UnwrapSDKContext(ctx).BlockTime()
	decisions := make(map[schedulertypes.TaskID]schedulertypes.ArbitrageDecision)
	for taskID, topicID := range taskTopicIDs { //nolint:maprange // decision map keyed by task id
		if _, ok := topSet[topicID]; ok {
			continue
		}
		decisions[taskID] = schedulertypes.ArbitrageDecision{
			Action: schedulertypes.ArbitrageActionReschedule,
			// Explicit ScheduleAt: empty reschedule options are a no-op in the scheduler.
			// Interval (if any) is preserved by RescheduleTask.
			RescheduleOpts: []schedulertypes.SchedulingOption{
				schedulertypes.ScheduleAt(now),
			},
		}
	}
	return decisions, nil
}

// skimTopTopicIDsByWeightDesc returns up to n topic IDs sorted by weight descending,
// tie-breaking with ascending topic ID (same rules as rewards.SkimTopTopicsByWeightDesc).
func skimTopTopicIDsByWeightDesc(weights map[TopicId]*alloraMath.Dec, n uint64) []TopicId {
	topicIDs := make([]TopicId, 0, len(weights))
	for topicID := range weights { //nolint:maprange // sort after collect
		topicIDs = append(topicIDs, topicID)
	}
	sort.Slice(topicIDs, func(i, j int) bool {
		if (*weights[topicIDs[i]]).Equal(*weights[topicIDs[j]]) {
			return topicIDs[i] < topicIDs[j]
		}
		return (*weights[topicIDs[i]]).Gt(*weights[topicIDs[j]])
	})
	if uint64(len(topicIDs)) > n {
		topicIDs = topicIDs[:n]
	}
	return topicIDs
}

func (k *Keeper) arbitrateStartNewEpochTasks(
	ctx context.Context,
	tasks []schedulertypes.Invocation[*types.StartNewEpochTaskArgs],
) (map[schedulertypes.TaskID]schedulertypes.ArbitrageDecision, error) {
	taskTopicIDs := make(map[schedulertypes.TaskID]TopicId, len(tasks))
	for _, task := range tasks {
		taskTopicIDs[task.TaskID] = task.Args.TopicId
	}
	return k.arbitrateByTopicWeight(ctx, taskTopicIDs)
}

func (k *Keeper) arbitrateCompleteEpochTasks(
	ctx context.Context,
	tasks []schedulertypes.Invocation[*types.EpochTransitionTaskArgs],
) (map[schedulertypes.TaskID]schedulertypes.ArbitrageDecision, error) {
	taskTopicIDs := make(map[schedulertypes.TaskID]TopicId, len(tasks))
	for _, task := range tasks {
		taskTopicIDs[task.TaskID] = task.Args.TopicId
	}
	return k.arbitrateByTopicWeight(ctx, taskTopicIDs)
}
