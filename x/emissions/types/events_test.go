package types_test

import (
	"encoding/json"
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
