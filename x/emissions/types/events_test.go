package types_test

import (
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
	require.Equal(t, "emissions.v1.EventScoresSet", event.Type)

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
	require.Len(t, events, 0)
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
	require.Equal(t, "emissions.v1.EventScoresSet", event.Type)

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
	require.Len(t, events, 0)
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
	require.Equal(t, "emissions.v1.EventScoresSet", event.Type)

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
	require.Len(t, events, 0)
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
	require.Equal(t, "emissions.v1.EventRewardsSettled", event.Type)

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
	require.Len(t, events, 0)
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
	require.Equal(t, "emissions.v1.EventRewardsSettled", event.Type)

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
	require.Len(t, events, 0)
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
	require.Equal(t, "emissions.v1.EventRewardsSettled", event.Type)

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
	require.Len(t, events, 0)
}
