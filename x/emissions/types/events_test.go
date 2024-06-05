package types_test

import (
	"strconv"
	"testing"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
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
	require.Equal(t, types.EventTypeInfererScoresSet, event.Type)

	attributes := event.Attributes
	require.Len(t, attributes, 4)

	require.Equal(t, types.AttributeKeyTopicId, string(attributes[0].Key))
	require.Equal(t, strconv.FormatUint(scores[0].TopicId, 10), string(attributes[0].Value))

	require.Equal(t, types.AttributeKeyBlockHeight, string(attributes[1].Key))
	require.Equal(t, "10", string(attributes[1].Value))

	require.Equal(t, types.AttributeKeyAddresses, string(attributes[2].Key))
	require.Equal(t, `["address1","address2"]`, string(attributes[2].Value))

	require.Equal(t, types.AttributeKeyScores, string(attributes[3].Key))
	require.Equal(t, `["100","200"]`, string(attributes[3].Value))
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
	require.Equal(t, types.EventTypeForecasterScoresSet, event.Type)

	attributes := event.Attributes
	require.Len(t, attributes, 4)

	require.Equal(t, types.AttributeKeyTopicId, string(attributes[0].Key))
	require.Equal(t, strconv.FormatUint(scores[0].TopicId, 10), string(attributes[0].Value))

	require.Equal(t, types.AttributeKeyBlockHeight, string(attributes[1].Key))
	require.Equal(t, "10", string(attributes[1].Value))

	require.Equal(t, types.AttributeKeyAddresses, string(attributes[2].Key))
	require.Equal(t, `["address1","address2"]`, string(attributes[2].Value))

	require.Equal(t, types.AttributeKeyScores, string(attributes[3].Key))
	require.Equal(t, `["100","200"]`, string(attributes[3].Value))
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
	require.Equal(t, types.EventTypeReputerScoresSet, event.Type)

	attributes := event.Attributes
	require.Len(t, attributes, 4)

	require.Equal(t, types.AttributeKeyTopicId, string(attributes[0].Key))
	require.Equal(t, strconv.FormatUint(scores[0].TopicId, 10), string(attributes[0].Value))

	require.Equal(t, types.AttributeKeyBlockHeight, string(attributes[1].Key))
	require.Equal(t, "10", string(attributes[1].Value))

	require.Equal(t, types.AttributeKeyAddresses, string(attributes[2].Key))
	require.Equal(t, `["address1","address2"]`, string(attributes[2].Value))

	require.Equal(t, types.AttributeKeyScores, string(attributes[3].Key))
	require.Equal(t, `["100","200"]`, string(attributes[3].Value))
}

func TestEmitNewReputerScoresSetEventWithNoScores(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	scores := []types.Score{}

	types.EmitNewReputerScoresSetEvent(ctx, scores)

	events := ctx.EventManager().Events()
	require.Len(t, events, 0)
}

func TestEmitNewInfererRewardsSettledEvent(t *testing.T) {
	topicId1 := uint64(1)
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	blockHeight := int64(10)
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

	types.EmitNewInfererRewardsSettledEvent(ctx, blockHeight, rewards)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, types.EventTypeInfererRewardsSettled, event.Type)

	attributes := event.Attributes
	require.Len(t, attributes, 4)

	require.Equal(t, types.AttributeKeyTopicId, string(attributes[0].Key))
	require.Equal(t, strconv.FormatUint(topicId1, 10), string(attributes[0].Value))

	require.Equal(t, types.AttributeKeyBlockHeight, string(attributes[1].Key))
	require.Equal(t, "10", string(attributes[1].Value))

	require.Equal(t, types.AttributeKeyAddresses, string(attributes[2].Key))
	require.Equal(t, `["address1","address2"]`, string(attributes[2].Value))

	require.Equal(t, types.AttributeKeyRewards, string(attributes[3].Key))
	require.Equal(t, `["100","200"]`, string(attributes[3].Value))
}

func TestEmitNewInfererRewardsSettledEventEmptyScores(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	blockHeight := int64(10)
	rewards := []types.TaskReward{}

	types.EmitNewInfererRewardsSettledEvent(ctx, blockHeight, rewards)

	events := ctx.EventManager().Events()
	require.Len(t, events, 0)
}

func TestEmitNewForecasterRewardsSettledEvent(t *testing.T) {
	blockHeight := int64(10)
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

	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	types.EmitNewForecasterRewardsSettledEvent(ctx, blockHeight, rewards)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, types.EventTypeForecasterRewardsSettled, event.Type)

	attributes := event.Attributes
	require.Len(t, attributes, 4)

	require.Equal(t, types.AttributeKeyTopicId, string(attributes[0].Key))
	require.Equal(t, strconv.FormatUint(rewards[0].TopicId, 10), string(attributes[0].Value))

	require.Equal(t, types.AttributeKeyBlockHeight, string(attributes[1].Key))
	require.Equal(t, "10", string(attributes[1].Value))

	require.Equal(t, types.AttributeKeyAddresses, string(attributes[2].Key))
	require.Equal(t, `["address1","address2"]`, string(attributes[2].Value))

	require.Equal(t, types.AttributeKeyRewards, string(attributes[3].Key))
	require.Equal(t, `["100","200"]`, string(attributes[3].Value))
}

func TestEmitNewForecasterRewardsSettledEventEmptyScores(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	blockHeight := int64(10)
	rewards := []types.TaskReward{}

	types.EmitNewForecasterRewardsSettledEvent(ctx, blockHeight, rewards)

	events := ctx.EventManager().Events()
	require.Len(t, events, 0)
}

func TestEmitNewReputerAndDelegatorRewardsSettledEvent(t *testing.T) {
	topicId1 := uint64(1)
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	blockHeight := int64(10)
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

	types.EmitNewReputerAndDelegatorRewardsSettledEvent(ctx, blockHeight, rewards)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, types.EventTypeReputerAndDelegatorRewardsSettled, event.Type)

	attributes := event.Attributes
	require.Len(t, attributes, 4)

	require.Equal(t, types.AttributeKeyTopicId, string(attributes[0].Key))
	require.Equal(t, strconv.FormatUint(topicId1, 10), string(attributes[0].Value))

	require.Equal(t, types.AttributeKeyBlockHeight, string(attributes[1].Key))
	require.Equal(t, "10", string(attributes[1].Value))

	require.Equal(t, types.AttributeKeyAddresses, string(attributes[2].Key))
	require.Equal(t, `["address1","address2"]`, string(attributes[2].Value))

	require.Equal(t, types.AttributeKeyRewards, string(attributes[3].Key))
	require.Equal(t, `["100","200"]`, string(attributes[3].Value))
}

func TestEmitNewReputerAndDelegatorRewardsSettledEventEmptyScores(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	blockHeight := int64(10)
	rewards := []types.TaskReward{}

	types.EmitNewReputerAndDelegatorRewardsSettledEvent(ctx, blockHeight, rewards)

	events := ctx.EventManager().Events()
	require.Len(t, events, 0)
}
