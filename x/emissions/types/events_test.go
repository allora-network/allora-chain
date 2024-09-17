package types_test

import (
	"encoding/json"
	"strconv"
	"testing"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

const (
	AttributeKeyActorType   = "actor_type"
	AttributeKeyTopicId     = "topic_id"
	AttributeKeyBlockHeight = "block_height"
	AttributeKeyAddresses   = "addresses"
	AttributeKeyScores      = "scores"
	AttributeKeyRewards     = "rewards"
	AttributeKeyValueBundle = "value_bundle"
)

func TestEmitNewInfererScoresSetEventWithScores(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	scores := []types.Score{
		{
			TopicId:     uint64(1),
			BlockHeight: int64(10),
			Address:     "address1",
			Score:       alloraMath.NewDecFromInt64(100),
		},
		{
			TopicId:     uint64(1),
			BlockHeight: int64(10),
			Address:     "address2",
			Score:       alloraMath.NewDecFromInt64(200),
		},
	}

	types.EmitNewInfererScoresSetEvent(ctx, scores)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "emissions.v3.EventScoresSet", event.Type)

	attributes := event.Attributes
	require.Len(t, attributes, 5)

	val, exists := event.GetAttribute(AttributeKeyActorType)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "INFERER")

	val, exists = event.GetAttribute(AttributeKeyTopicId)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "1")

	val, exists = event.GetAttribute(AttributeKeyBlockHeight)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "10")

	val, exists = event.GetAttribute(AttributeKeyAddresses)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), `["address1","address2"]`)

	val, exists = event.GetAttribute(AttributeKeyScores)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), `["100","200"]`)
}

func TestEmitNewInfererScoresSetEventWithNoScores(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	scores := []types.Score{}

	types.EmitNewInfererScoresSetEvent(ctx, scores)

	events := ctx.EventManager().Events()
	require.Empty(t, events)
}

func TestEmitNewForecasterScoresSetEventWithScores(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	scores := []types.Score{
		{
			TopicId:     uint64(1),
			BlockHeight: int64(10),
			Address:     "address1",
			Score:       alloraMath.NewDecFromInt64(100),
		},
		{
			TopicId:     uint64(1),
			BlockHeight: int64(10),
			Address:     "address2",
			Score:       alloraMath.NewDecFromInt64(200),
		},
	}

	types.EmitNewForecasterScoresSetEvent(ctx, scores)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "emissions.v3.EventScoresSet", event.Type)

	attributes := event.Attributes
	require.Len(t, attributes, 5)

	val, exists := event.GetAttribute(AttributeKeyActorType)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "FORECASTER")

	val, exists = event.GetAttribute(AttributeKeyTopicId)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "1")

	val, exists = event.GetAttribute(AttributeKeyBlockHeight)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "10")

	val, exists = event.GetAttribute(AttributeKeyAddresses)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), `["address1","address2"]`)

	val, exists = event.GetAttribute(AttributeKeyScores)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), `["100","200"]`)
}

func TestEmitNewForecasterScoresSetEventWithNoScores(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	scores := []types.Score{}

	types.EmitNewForecasterScoresSetEvent(ctx, scores)

	events := ctx.EventManager().Events()
	require.Empty(t, events)
}

func TestEmitNewReputerScoresSetEventWithScores(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	scores := []types.Score{
		{
			TopicId:     uint64(1),
			BlockHeight: int64(10),
			Address:     "address1",
			Score:       alloraMath.NewDecFromInt64(100),
		},
		{
			TopicId:     uint64(1),
			BlockHeight: int64(10),
			Address:     "address2",
			Score:       alloraMath.NewDecFromInt64(200),
		},
	}

	types.EmitNewReputerScoresSetEvent(ctx, scores)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "emissions.v3.EventScoresSet", event.Type)

	attributes := event.Attributes
	require.Len(t, attributes, 5)

	val, exists := event.GetAttribute(AttributeKeyActorType)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "REPUTER")

	val, exists = event.GetAttribute(AttributeKeyTopicId)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "1")

	val, exists = event.GetAttribute(AttributeKeyBlockHeight)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "10")

	val, exists = event.GetAttribute(AttributeKeyAddresses)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), `["address1","address2"]`)

	val, exists = event.GetAttribute(AttributeKeyScores)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), `["100","200"]`)
}

func TestEmitNewReputerScoresSetEventWithNoScores(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	scores := []types.Score{}

	types.EmitNewReputerScoresSetEvent(ctx, scores)

	events := ctx.EventManager().Events()
	require.Empty(t, events)
}

func TestEmitNewInfererRewardsSettledEventWithRewards(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	rewards := []types.TaskReward{
		{
			TopicId: uint64(1),
			Address: "address1",
			Reward:  alloraMath.NewDecFromInt64(100),
		},
		{
			TopicId: uint64(1),
			Address: "address2",
			Reward:  alloraMath.NewDecFromInt64(200),
		},
	}

	types.EmitNewInfererRewardsSettledEvent(ctx, types.BlockHeight(10), rewards)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "emissions.v3.EventRewardsSettled", event.Type)

	attributes := event.Attributes
	require.Len(t, attributes, 5)

	val, exists := event.GetAttribute(AttributeKeyActorType)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "INFERER")

	val, exists = event.GetAttribute(AttributeKeyTopicId)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "1")

	val, exists = event.GetAttribute(AttributeKeyBlockHeight)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "10")

	val, exists = event.GetAttribute(AttributeKeyAddresses)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), `["address1","address2"]`)

	val, exists = event.GetAttribute(AttributeKeyRewards)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), `["100","200"]`)
}

func TestEmitNewInfererRewardsSettledEventWithNoRewards(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	rewards := []types.TaskReward{}

	types.EmitNewInfererRewardsSettledEvent(ctx, types.BlockHeight(10), rewards)

	events := ctx.EventManager().Events()
	require.Empty(t, events)
}

func TestEmitNewForecasterRewardsSettledEventWithRewards(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	rewards := []types.TaskReward{
		{
			TopicId: uint64(1),
			Address: "address1",
			Reward:  alloraMath.NewDecFromInt64(100),
		},
		{
			TopicId: uint64(1),
			Address: "address2",
			Reward:  alloraMath.NewDecFromInt64(200),
		},
	}

	types.EmitNewForecasterRewardsSettledEvent(ctx, types.BlockHeight(10), rewards)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "emissions.v3.EventRewardsSettled", event.Type)

	attributes := event.Attributes
	require.Len(t, attributes, 5)

	val, exists := event.GetAttribute(AttributeKeyActorType)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "FORECASTER")

	val, exists = event.GetAttribute(AttributeKeyTopicId)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "1")

	val, exists = event.GetAttribute(AttributeKeyBlockHeight)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "10")

	val, exists = event.GetAttribute(AttributeKeyAddresses)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), `["address1","address2"]`)

	val, exists = event.GetAttribute(AttributeKeyRewards)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), `["100","200"]`)
}

func TestEmitNewForecasterRewardsSettledEventWithNoRewards(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	rewards := []types.TaskReward{}

	types.EmitNewForecasterRewardsSettledEvent(ctx, types.BlockHeight(10), rewards)

	events := ctx.EventManager().Events()
	require.Empty(t, events)
}

func TestEmitNewReputerAndDelegatorRewardsSettledEventWithRewards(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	rewards := []types.TaskReward{
		{
			TopicId: uint64(1),
			Address: "address1",
			Reward:  alloraMath.NewDecFromInt64(100),
		},
		{
			TopicId: uint64(1),
			Address: "address2",
			Reward:  alloraMath.NewDecFromInt64(200),
		},
	}

	types.EmitNewReputerAndDelegatorRewardsSettledEvent(ctx, types.BlockHeight(10), rewards)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "emissions.v3.EventRewardsSettled", event.Type)

	attributes := event.Attributes
	require.Len(t, attributes, 5)

	val, exists := event.GetAttribute(AttributeKeyActorType)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "REPUTER")

	val, exists = event.GetAttribute(AttributeKeyTopicId)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "1")

	val, exists = event.GetAttribute(AttributeKeyBlockHeight)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "10")

	val, exists = event.GetAttribute(AttributeKeyAddresses)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), `["address1","address2"]`)

	val, exists = event.GetAttribute(AttributeKeyRewards)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), `["100","200"]`)
}

func TestEmitNewReputerAndDelegatorRewardsSettledEventWithNoRewards(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	rewards := []types.TaskReward{}

	types.EmitNewReputerAndDelegatorRewardsSettledEvent(ctx, types.BlockHeight(10), rewards)

	events := ctx.EventManager().Events()
	require.Empty(t, events)
}

func TestEmitNewNetworkLossSetEvent(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	topicId := uint64(1)
	blockHeight := int64(10)
	loss := types.ValueBundle{
		CombinedValue:          alloraMath.MustNewDecFromString("10"),
		NaiveValue:             alloraMath.MustNewDecFromString("20"),
		InfererValues:          []*types.WorkerAttributedValue{{Worker: "TestInferer", Value: alloraMath.MustNewDecFromString("0.0112")}, {Worker: "TestInferer1", Value: alloraMath.MustNewDecFromString("0.0112")}},
		ForecasterValues:       []*types.WorkerAttributedValue{{Worker: "TestForecaster", Value: alloraMath.MustNewDecFromString("0.0112")}, {Worker: "TestForecaster1", Value: alloraMath.MustNewDecFromString("0.0112")}},
		OneOutInfererValues:    []*types.WithheldWorkerAttributedValue{{Worker: "TestInferer2", Value: alloraMath.MustNewDecFromString("0.0112")}, {Worker: "TestInferer3", Value: alloraMath.MustNewDecFromString("0.0112")}},
		OneOutForecasterValues: []*types.WithheldWorkerAttributedValue{{Worker: "TestForecaster3", Value: alloraMath.MustNewDecFromString("0.0112")}, {Worker: "TestForecaster4", Value: alloraMath.MustNewDecFromString("0.0112")}},
		OneInForecasterValues:  []*types.WorkerAttributedValue{{Worker: "TestForecaster5", Value: alloraMath.MustNewDecFromString("0.0112")}, {Worker: "TestForecaster6", Value: alloraMath.MustNewDecFromString("0.0112")}},
	}

	types.EmitNewNetworkLossSetEvent(ctx, topicId, blockHeight, loss)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "emissions.v3.EventNetworkLossSet", event.Type)

	attributes := event.Attributes
	require.Len(t, attributes, 3)

	var result types.ValueBundle
	val, exists := event.GetAttribute(AttributeKeyValueBundle)
	require.True(t, exists)
	_ = json.Unmarshal([]byte(val.GetValue()), &result)
	require.Equal(t, loss.CombinedValue, result.CombinedValue)
	require.Equal(t, loss.NaiveValue, result.NaiveValue)
	require.Equal(t, loss.InfererValues, result.InfererValues)
	require.Equal(t, loss.ForecasterValues, result.ForecasterValues)
	require.Equal(t, loss.OneOutInfererValues, result.OneOutInfererValues)
	require.Equal(t, loss.OneOutForecasterValues, result.OneOutForecasterValues)
	require.Equal(t, loss.OneInForecasterValues, result.OneInForecasterValues)
}

func TestEmitNewForecastTaskSetEvent(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	topicId := uint64(1)
	loss := types.ValueBundle{
		CombinedValue:          alloraMath.MustNewDecFromString("10"),
		NaiveValue:             alloraMath.MustNewDecFromString("20"),
		InfererValues:          []*types.WorkerAttributedValue{{Worker: "TestInferer", Value: alloraMath.MustNewDecFromString("0.0112")}, {Worker: "TestInferer1", Value: alloraMath.MustNewDecFromString("0.0112")}},
		ForecasterValues:       []*types.WorkerAttributedValue{{Worker: "TestForecaster", Value: alloraMath.MustNewDecFromString("0.0112")}, {Worker: "TestForecaster1", Value: alloraMath.MustNewDecFromString("0.0112")}},
		OneOutInfererValues:    []*types.WithheldWorkerAttributedValue{{Worker: "TestInferer2", Value: alloraMath.MustNewDecFromString("0.0112")}, {Worker: "TestInferer3", Value: alloraMath.MustNewDecFromString("0.0112")}},
		OneOutForecasterValues: []*types.WithheldWorkerAttributedValue{{Worker: "TestForecaster3", Value: alloraMath.MustNewDecFromString("0.0112")}, {Worker: "TestForecaster4", Value: alloraMath.MustNewDecFromString("0.0112")}},
		OneInForecasterValues:  []*types.WorkerAttributedValue{{Worker: "TestForecaster5", Value: alloraMath.MustNewDecFromString("0.0112")}, {Worker: "TestForecaster6", Value: alloraMath.MustNewDecFromString("0.0112")}},
	}

	score, err := loss.NaiveValue.Sub(loss.CombinedValue)
	require.NoError(t, err)

	types.EmitNewForecastTaskScoreSetEvent(ctx, topicId, score)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "emissions.v4.EventForecastTaskScoreSet", event.Type)

	require.Contains(t, event.Attributes[0].Key, "score")
	require.Contains(t, event.Attributes[0].Value, "10")

	require.Contains(t, event.Attributes[1].Key, "topic_id")
	require.Contains(t, event.Attributes[1].Value, "1")
}

func TestNewLastCommitSetEvent(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	topicId1 := uint64(1)
	topicId2 := uint64(2)
	workerHeight := int64(10)
	worker2Height := int64(20)
	reputerHeight := int64(30)
	types.EmitNewWorkerLastCommitSetEvent(ctx, topicId1, workerHeight, &types.Nonce{BlockHeight: workerHeight - 5})
	types.EmitNewWorkerLastCommitSetEvent(ctx, topicId1, worker2Height, &types.Nonce{BlockHeight: worker2Height - 5})
	types.EmitNewReputerLastCommitSetEvent(ctx, topicId2, reputerHeight, &types.Nonce{BlockHeight: reputerHeight - 5})

	events := ctx.EventManager().Events()
	require.Len(t, events, 3)

	require.Equal(t, "emissions.v4.EventWorkerLastCommitSet", events[0].Type)
	require.Equal(t, "emissions.v4.EventWorkerLastCommitSet", events[1].Type)
	require.Equal(t, "emissions.v4.EventReputerLastCommitSet", events[2].Type)

	require.Contains(t, events[0].Attributes[0].Key, "block_height")
	require.Contains(t, events[0].Attributes[1].Key, "nonce")
	require.Contains(t, events[0].Attributes[2].Key, "topic_id")
	require.Contains(t, events[0].Attributes[0].Value, "10")
	require.Contains(t, events[0].Attributes[1].Value, "{\"block_height\":\"5\"}")
	require.Contains(t, events[0].Attributes[2].Value, "1")

	require.Contains(t, events[1].Attributes[0].Key, "block_height")
	require.Contains(t, events[1].Attributes[1].Key, "nonce")
	require.Contains(t, events[1].Attributes[2].Key, "topic_id")
	require.Contains(t, events[1].Attributes[0].Value, "20")
	require.Contains(t, events[1].Attributes[1].Value, "{\"block_height\":\"15\"}")
	require.Contains(t, events[1].Attributes[2].Value, "1")

	require.Contains(t, events[2].Attributes[0].Key, "block_height")
	require.Contains(t, events[2].Attributes[1].Key, "nonce")
	require.Contains(t, events[2].Attributes[2].Key, "topic_id")
	require.Contains(t, events[2].Attributes[0].Value, "30")
	require.Contains(t, events[2].Attributes[1].Value, "{\"block_height\":\"25\"}")
	require.Contains(t, events[2].Attributes[2].Value, "2")
}

func TestEmitNewTopicRewardsSetEvent(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	var topicIds = []uint64{1, 2, 3, 4, 5}
	topicRewards := make(map[uint64]*alloraMath.Dec)
	for index, id := range topicIds {
		reward := alloraMath.MustNewDecFromString(strconv.Itoa(10 * index))
		topicRewards[id] = &reward
	}

	types.EmitNewTopicRewardSetEvent(ctx, topicRewards)
	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	require.Equal(t, "emissions.v4.EventTopicRewardsSet", events[0].Type)
	require.Contains(t, events[0].Attributes[0].Key, "rewards")
	require.Contains(t, events[0].Attributes[0].Value, `["0","10","20","30","40"]`)
	require.Contains(t, events[0].Attributes[1].Key, "topic_ids")
	require.Contains(t, events[0].Attributes[1].Value, `["1","2","3","4","5"]`)
}
