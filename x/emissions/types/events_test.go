package types_test

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"

	cosmosMath "cosmossdk.io/math"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

const (
	AttributeKeyActorType     = "actor_type"
	AttributeKeyTopicId       = "topic_id"
	AttributeKeyBlockHeight   = "block_height"
	AttributeKeyBlockHeightTx = "block_height_tx"
	AttributeKeyAddresses     = "addresses"
	AttributeKeyScores        = "scores"
	AttributeKeyRewards       = "rewards"
	AttributeKeyValueBundle   = "bundle"
	AttributeKeyCoefficients  = "coefficients"
	AttributeKeyRegrets       = "regrets"
	AttributeKeyRegret        = "regret"
	AttributeKeyWeights       = "weights"
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

	types.EmitNewActorScoresSetEvent(ctx, types.ActorType_ACTOR_TYPE_INFERER_UNSPECIFIED, 10, scores)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "emissions.v10.EventScoresSet", event.Type)

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

	types.EmitNewActorScoresSetEvent(ctx, types.ActorType_ACTOR_TYPE_INFERER_UNSPECIFIED, 10, scores)

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

	types.EmitNewActorScoresSetEvent(ctx, types.ActorType_ACTOR_TYPE_FORECASTER, 10, scores)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "emissions.v10.EventScoresSet", event.Type)

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

	types.EmitNewActorScoresSetEvent(ctx, types.ActorType_ACTOR_TYPE_FORECASTER, 10, scores)

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

	types.EmitNewActorScoresSetEvent(ctx, types.ActorType_ACTOR_TYPE_REPUTER, 10, scores)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "emissions.v10.EventScoresSet", event.Type)

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

	types.EmitNewActorScoresSetEvent(ctx, types.ActorType_ACTOR_TYPE_REPUTER, 10, scores)

	events := ctx.EventManager().Events()
	require.Empty(t, events)
}

func TestEmitNewTopicUpdatedEvent(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())

	topic := types.Topic{
		Id:                       1,
		Creator:                  "creator",
		EpochLastEnded:           100,
		InitialRegret:            alloraMath.MustNewDecFromString("0.01"),
		Metadata:                 "updated metadata",
		LossMethod:               "mse",
		EpochLength:              100,
		GroundTruthLag:           100,
		WorkerSubmissionWindow:   10,
		AllowNegative:            false,
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		PNorm:                    alloraMath.MustNewDecFromString("3.0"),
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		CNorm:                    alloraMath.MustNewDecFromString("0.75"),
		TopicType:                types.TopicType_TOPIC_TYPE_REGRESSION,
		OutputArity:              types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
		RequireUnity:             false,
		UnityTolerance:           alloraMath.Dec{},
	}

	types.EmitNewTopicUpdatedEvent(ctx, topic)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)
	require.Equal(t, "emissions.v10.EventTopicUpdated", events[0].Type)

	attributes := events[0].Attributes
	val, exists := events[0].GetAttribute(AttributeKeyTopicId)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "1")

	foundTopicId := false
	foundTopic := false
	for _, attr := range attributes {
		if strings.Contains(attr.Key, "topic_id") {
			require.Equal(t, "\"1\"", attr.Value)
			foundTopicId = true
		}
		if attr.Key == "topic" {
			require.Contains(t, attr.Value, "updated metadata")
			require.Contains(t, attr.Value, "\"id\":\"1\"")
			foundTopic = true
		}
	}
	require.True(t, foundTopic, "expected topic attribute to be present in event")
	require.True(t, foundTopicId, "expected topic_id attribute to be present in event")
}

func TestEmitNewInfererRewardsSettledEventWithRewards(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	rewards := []types.TaskReward{
		{
			TopicId: uint64(1),
			Address: "address1",
			Reward:  alloraMath.NewDecFromInt64(100),
			Type:    types.WorkerInferenceRewardType,
		},
		{
			TopicId: uint64(1),
			Address: "address2",
			Reward:  alloraMath.NewDecFromInt64(200),
			Type:    types.WorkerInferenceRewardType,
		},
	}

	types.EmitNewInfererRewardsSettledEvent(ctx, types.BlockHeight(10), types.BlockHeight(20), rewards)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "emissions.v10.EventRewardsSettled", event.Type)

	attributes := event.Attributes
	require.Len(t, attributes, 6)

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

	types.EmitNewInfererRewardsSettledEvent(ctx, types.BlockHeight(10), types.BlockHeight(20), rewards)

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
			Type:    types.WorkerForecastRewardType,
		},
		{
			TopicId: uint64(1),
			Address: "address2",
			Reward:  alloraMath.NewDecFromInt64(200),
			Type:    types.WorkerForecastRewardType,
		},
	}

	types.EmitNewForecasterRewardsSettledEvent(ctx, types.BlockHeight(10), types.BlockHeight(20), rewards)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "emissions.v10.EventRewardsSettled", event.Type)

	attributes := event.Attributes
	require.Len(t, attributes, 6)

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

	types.EmitNewForecasterRewardsSettledEvent(ctx, types.BlockHeight(10), types.BlockHeight(20), rewards)

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
			Type:    types.ReputerAndDelegatorRewardType,
		},
		{
			TopicId: uint64(1),
			Address: "address2",
			Reward:  alloraMath.NewDecFromInt64(200),
			Type:    types.ReputerAndDelegatorRewardType,
		},
	}

	types.EmitNewReputerAndDelegatorRewardsSettledEvent(ctx, types.BlockHeight(10), types.BlockHeight(20), rewards)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "emissions.v10.EventRewardsSettled", event.Type)

	attributes := event.Attributes
	require.Len(t, attributes, 6)

	val, exists := event.GetAttribute(AttributeKeyActorType)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "REPUTER")

	val, exists = event.GetAttribute(AttributeKeyTopicId)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "1")

	val, exists = event.GetAttribute(AttributeKeyBlockHeight)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "10")

	val, exists = event.GetAttribute(AttributeKeyBlockHeightTx)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "20")

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

	types.EmitNewReputerAndDelegatorRewardsSettledEvent(ctx, types.BlockHeight(10), types.BlockHeight(20), rewards)

	events := ctx.EventManager().Events()
	require.Empty(t, events)
}

func TestEmitNewNetworkLossSetEvent(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	topicId := uint64(1)
	blockHeight := int64(10)
	loss := types.ValueBundle{
		TopicId:                topicId,
		ReputerRequestNonce:    &types.ReputerRequestNonce{ReputerNonce: &types.Nonce{BlockHeight: blockHeight}},
		Reputer:                "TestReputer",
		ExtraData:              nil,
		CombinedValue:          alloraMath.MustNewDecFromString("10"),
		NaiveValue:             alloraMath.MustNewDecFromString("20"),
		InfererValues:          []*types.WorkerAttributedValue{{Worker: "TestInferer", Value: alloraMath.MustNewDecFromString("0.0112")}, {Worker: "TestInferer1", Value: alloraMath.MustNewDecFromString("0.0112")}},
		ForecasterValues:       []*types.WorkerAttributedValue{{Worker: "TestForecaster", Value: alloraMath.MustNewDecFromString("0.0112")}, {Worker: "TestForecaster1", Value: alloraMath.MustNewDecFromString("0.0112")}},
		OneOutInfererValues:    []*types.WithheldWorkerAttributedValue{{Worker: "TestInferer2", Value: alloraMath.MustNewDecFromString("0.0112")}, {Worker: "TestInferer3", Value: alloraMath.MustNewDecFromString("0.0112")}},
		OneOutForecasterValues: []*types.WithheldWorkerAttributedValue{{Worker: "TestForecaster3", Value: alloraMath.MustNewDecFromString("0.0112")}, {Worker: "TestForecaster4", Value: alloraMath.MustNewDecFromString("0.0112")}},
		OneInForecasterValues:  []*types.WorkerAttributedValue{{Worker: "TestForecaster5", Value: alloraMath.MustNewDecFromString("0.0112")}, {Worker: "TestForecaster6", Value: alloraMath.MustNewDecFromString("0.0112")}},
		OneOutInfererForecasterValues: []*types.OneOutInfererForecasterValues{
			{
				Forecaster: "allo1e7l2cyn29c8jmama9hs0n3weq4tg6pqdzsl249",
				OneOutInfererValues: []*types.WithheldWorkerAttributedValue{
					{
						Worker: "TestInferer",
						Value:  alloraMath.MustNewDecFromString("0.0112"),
					},
				},
			},
		},
	}

	types.EmitNewNetworkLossSetEvent(ctx, loss)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "emissions.v10.EventNetworkLossSet", event.Type)

	attributes := event.Attributes
	require.Len(t, attributes, 5)
	val, exists := event.GetAttribute(AttributeKeyValueBundle)
	require.True(t, exists)
	assertEventValueBundle(t, val.GetValue(), loss)
}

func TestEmitNewNetworkInferencesEvent(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	topicId := uint64(1)
	nonce := int64(10)

	dec := alloraMath.MustNewDecFromString

	networkInferences := types.NetworkInferenceBundle{
		TopicId: topicId,
		Nonce:   nonce,
		CombinedValue: []*types.LabeledValue{
			{LabelId: 0, LabelName: "y", Value: dec("10")},
		},
		NaiveValue: []*types.LabeledValue{
			{LabelId: 0, LabelName: "y", Value: dec("20")},
		},
		InfererValues: []*types.WorkerInference{
			{
				Worker: "allo1e7l2cyn29c8jmama9hs0n3weq4tg6pqdzsl249",
				Values: []*types.LabeledValue{
					{LabelId: 0, LabelName: "y", Value: dec("0.0112")},
				},
			},
			{
				Worker: "allo1xqxnuvvegql2n4xtx52n3fay4t5gkv3wr8etca",
				Values: []*types.LabeledValue{
					{LabelId: 0, LabelName: "y", Value: dec("0.0112")},
				},
			},
		},
		ForecasterValues: []*types.WorkerInference{
			{
				Worker: "allo1e7l2cyn29c8jmama9hs0n3weq4tg6pqdzsl249",
				Values: []*types.LabeledValue{
					{LabelId: 0, LabelName: "y", Value: dec("0.0112")},
				},
			},
			{
				Worker: "allo1xqxnuvvegql2n4xtx52n3fay4t5gkv3wr8etca",
				Values: []*types.LabeledValue{
					{LabelId: 0, LabelName: "y", Value: dec("0.0112")},
				},
			},
		},
		OneOutInfererValues: []*types.OneOutInfererValue{
			{
				WithheldInferer: "allo1e7l2cyn29c8jmama9hs0n3weq4tg6pqdzsl249",
				CombinedInference: []*types.LabeledValue{
					{LabelId: 0, LabelName: "y", Value: dec("0.0112")},
				},
			},
			{
				WithheldInferer: "allo1xqxnuvvegql2n4xtx52n3fay4t5gkv3wr8etca",
				CombinedInference: []*types.LabeledValue{
					{LabelId: 0, LabelName: "y", Value: dec("0.0112")},
				},
			},
		},
		OneOutForecasterValues: []*types.OneOutForecasterValue{
			{
				WithheldForecaster: "allo1e7l2cyn29c8jmama9hs0n3weq4tg6pqdzsl249",
				CombinedInference: []*types.LabeledValue{
					{LabelId: 0, LabelName: "y", Value: dec("0.0112")},
				},
			},
			{
				WithheldForecaster: "allo1xqxnuvvegql2n4xtx52n3fay4t5gkv3wr8etca",
				CombinedInference: []*types.LabeledValue{
					{LabelId: 0, LabelName: "y", Value: dec("0.0112")},
				},
			},
		},
		OneInForecasterValues: []*types.OneInForecasterValue{
			{
				Forecaster: "allo1e7l2cyn29c8jmama9hs0n3weq4tg6pqdzsl249",
				CombinedInference: []*types.LabeledValue{
					{LabelId: 0, LabelName: "y", Value: dec("0.0112")},
				},
			},
			{
				Forecaster: "allo1xqxnuvvegql2n4xtx52n3fay4t5gkv3wr8etca",
				CombinedInference: []*types.LabeledValue{
					{LabelId: 0, LabelName: "y", Value: dec("0.0112")},
				},
			},
		},
		OneOutInfererForecasterValues: []*types.OneOutInfererForecasterValue{
			{
				Forecaster:      "allo1e7l2cyn29c8jmama9hs0n3weq4tg6pqdzsl249",
				WithheldInferer: "allo1e7l2cyn29c8jmama9hs0n3weq4tg6pqdzsl249",
				CombinedInference: []*types.LabeledValue{
					{LabelId: 0, LabelName: "y", Value: dec("0.0112")},
				},
			},
		},
	}

	types.EmitNewNetworkInferencesEvent(ctx, networkInferences)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "emissions.v10.EventNetworkInferenceBundle", event.Type)

	attributes := event.Attributes
	require.Len(t, attributes, 14)

	assertEventNetworkInferenceBundle(t, event, networkInferences, false)
}

func assertEventNetworkInferenceBundle(
	t *testing.T,
	event sdk.Event,
	bundle types.NetworkInferenceBundle,
	outlierResistant bool,
) {
	t.Helper()

	attr := func(key string) string {
		v, ok := event.GetAttribute(key)
		require.True(t, ok, "missing attribute %s", key)
		s := v.GetValue()
		if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
			s = s[1 : len(s)-1]
		}
		return s
	}

	gotTopicID, err := strconv.ParseUint(attr("topic_id"), 10, 64)
	require.NoError(t, err, "topic_id parse")
	require.Equal(t, bundle.GetTopicId(), gotTopicID)

	gotNonce, err := strconv.ParseInt(attr("nonce"), 10, 64)
	require.NoError(t, err, "nonce parse")
	require.Equal(t, bundle.GetNonce(), gotNonce)

	gotOutlier, err := strconv.ParseBool(attr("outlier_resistant"))
	require.NoError(t, err, "outlier_resistant parse")
	require.Equal(t, outlierResistant, gotOutlier)

	unmarshal := func(key string, dst any) {
		t.Helper()
		s := attr(key)
		require.NoError(t, json.Unmarshal([]byte(s), dst), "unmarshal %s: %q", key, s)
	}

	decArrayFromLV := func(vals []*types.LabeledValue) alloraMath.DecArray {
		out := make(alloraMath.DecArray, len(vals))
		for i := range vals {
			out[i] = vals[i].Value
		}
		return out
	}

	decMatrixFromWI := func(ws []*types.WorkerInference) alloraMath.DecMatrix {
		out := make(alloraMath.DecMatrix, len(ws))
		for i := range ws {
			out[i] = decArrayFromLV(ws[i].GetValues())
		}
		return out
	}

	eqDec := func(a, b alloraMath.Dec) bool {
		if a.IsNaN() && b.IsNaN() {
			return true
		}
		return a.Equal(b)
	}
	requireDecArrayEq := func(msg string, got, want alloraMath.DecArray) {
		t.Helper()
		require.Equal(t, len(want), len(got), "%s: len", msg)
		for i := range want {
			require.Truef(t, eqDec(got[i], want[i]), "%s[%d] got=%v want=%v", msg, i, got[i], want[i])
		}
	}
	requireDecMatrixEq := func(msg string, got, want alloraMath.DecMatrix) {
		t.Helper()
		require.Equal(t, len(want), len(got), "%s: rows", msg)
		for r := range want {
			requireDecArrayEq(fmt.Sprintf("%s[%d]", msg, r), got[r], want[r])
		}
	}

	var gotLabelNames []string
	unmarshal("label_names", &gotLabelNames)

	wantLabelNames := make([]string, len(bundle.GetCombinedValue()))
	for i, lv := range bundle.GetCombinedValue() {
		wantLabelNames[i] = lv.GetLabelName()
	}
	require.Equal(t, wantLabelNames, gotLabelNames)

	var gotCombined alloraMath.DecArray
	unmarshal("combined_value", &gotCombined)
	requireDecArrayEq("combined_value", gotCombined, decArrayFromLV(bundle.GetCombinedValue()))

	var gotNaive alloraMath.DecArray
	unmarshal("naive_value", &gotNaive)
	requireDecArrayEq("naive_value", gotNaive, decArrayFromLV(bundle.GetNaiveValue()))

	var gotInfererAddrs []string
	unmarshal("inferer_addresses", &gotInfererAddrs)

	var gotInfererVals alloraMath.DecMatrix
	unmarshal("inferer_values", &gotInfererVals)

	wantInfererAddrs := make([]string, len(bundle.GetInfererValues()))
	for i, w := range bundle.GetInfererValues() {
		wantInfererAddrs[i] = w.GetWorker()
	}
	require.Equal(t, wantInfererAddrs, gotInfererAddrs)
	requireDecMatrixEq("inferer_values", gotInfererVals, decMatrixFromWI(bundle.GetInfererValues()))

	var gotForecasterAddrs []string
	unmarshal("forecaster_addresses", &gotForecasterAddrs)

	var gotForecasterVals alloraMath.DecMatrix
	unmarshal("forecaster_values", &gotForecasterVals)

	wantForecasterAddrs := make([]string, len(bundle.GetForecasterValues()))
	for i, w := range bundle.GetForecasterValues() {
		wantForecasterAddrs[i] = w.GetWorker()
	}
	require.Equal(t, wantForecasterAddrs, gotForecasterAddrs)
	requireDecMatrixEq("forecaster_values", gotForecasterVals, decMatrixFromWI(bundle.GetForecasterValues()))

	var gotOOI alloraMath.DecMatrix
	unmarshal("one_out_inferer_values", &gotOOI)
	wantOOI := make(alloraMath.DecMatrix, len(bundle.GetOneOutInfererValues()))
	for i, v := range bundle.GetOneOutInfererValues() {
		wantOOI[i] = decArrayFromLV(v.GetCombinedInference())
	}
	requireDecMatrixEq("one_out_inferer_values", gotOOI, wantOOI)

	var gotOOF alloraMath.DecMatrix
	unmarshal("one_out_forecaster_values", &gotOOF)
	wantOOF := make(alloraMath.DecMatrix, len(bundle.GetOneOutForecasterValues()))
	for i, v := range bundle.GetOneOutForecasterValues() {
		wantOOF[i] = decArrayFromLV(v.GetCombinedInference())
	}
	requireDecMatrixEq("one_out_forecaster_values", gotOOF, wantOOF)

	var gotOIF alloraMath.DecMatrix
	unmarshal("one_in_forecaster_values", &gotOIF)
	wantOIF := make(alloraMath.DecMatrix, len(bundle.GetOneInForecasterValues()))
	for i, v := range bundle.GetOneInForecasterValues() {
		wantOIF[i] = decArrayFromLV(v.GetCombinedInference())
	}
	requireDecMatrixEq("one_in_forecaster_values", gotOIF, wantOIF)

	var gotOOIF []alloraMath.DecMatrix
	unmarshal("one_out_inferer_forecaster_values", &gotOOIF)

	order := make([]string, 0)
	rowsByForecaster := make(map[string][]alloraMath.DecArray)

	for _, x := range bundle.GetOneOutInfererForecasterValues() {
		if x == nil {
			continue
		}
		fc := x.GetForecaster()
		if _, ok := rowsByForecaster[fc]; !ok {
			order = append(order, fc)
		}
		rowsByForecaster[fc] = append(rowsByForecaster[fc], decArrayFromLV(x.GetCombinedInference()))
	}

	wantOOIF := make([]alloraMath.DecMatrix, 0, len(order))
	for _, fc := range order {
		rows := rowsByForecaster[fc]
		m := make(alloraMath.DecMatrix, len(rows))
		for i := range rows {
			m[i] = rows[i]
		}
		wantOOIF = append(wantOOIF, m)
	}

	require.Equal(t, len(wantOOIF), len(gotOOIF), "ooif group count")
	for gi := range wantOOIF {
		requireDecMatrixEq(fmt.Sprintf("ooif[%d]", gi), gotOOIF[gi], wantOOIF[gi])
	}
}

//nolint:exhaustruct
func TestValueBundleToEventValueBundleBase(t *testing.T) {
	const maxInferers = 32

	d := func(s string) alloraMath.Dec { return alloraMath.MustNewDecFromString(s) }
	nan := func() alloraMath.Dec { return alloraMath.NewNaN() }

	buildFullRowWA := func(n int) []*types.WorkerAttributedValue {
		out := make([]*types.WorkerAttributedValue, 0, n)
		for i := 1; i <= n; i++ {
			out = append(out, &types.WorkerAttributedValue{
				Worker: fmt.Sprintf("inf%d", i), Value: d(fmt.Sprintf("%d", i)),
			})
		}
		return out
	}
	buildFullRowOOIF := func(n int) []*types.WithheldWorkerAttributedValue {
		out := make([]*types.WithheldWorkerAttributedValue, 0, n)
		for i := 1; i <= n; i++ {
			out = append(out, &types.WithheldWorkerAttributedValue{
				Worker: fmt.Sprintf("inf%d", i), Value: d(fmt.Sprintf("%d", i)),
			})
		}
		return out
	}

	tests := []struct {
		name string
		in   *types.ValueBundle
		exp  *types.EventValueBundle
	}{
		{
			name: "0 actors (empty everything)",
			in:   &types.ValueBundle{},
			exp:  &types.EventValueBundle{},
		},
		{
			name: "1 actor (inferer or forecaster), full row",
			in: &types.ValueBundle{
				ExtraData:              []byte("extra"),
				CombinedValue:          alloraMath.MustNewDecFromString("0.0112"),
				NaiveValue:             alloraMath.MustNewDecFromString("0.0113"),
				InfererValues:          []*types.WorkerAttributedValue{{Worker: "inf1", Value: d("1")}},
				ForecasterValues:       []*types.WorkerAttributedValue{{Worker: "F1", Value: d("10")}},
				OneOutInfererValues:    []*types.WithheldWorkerAttributedValue{{Worker: "inf1", Value: d("100")}},
				OneOutForecasterValues: []*types.WithheldWorkerAttributedValue{{Worker: "F1", Value: d("1000")}},
				OneInForecasterValues:  []*types.WorkerAttributedValue{{Worker: "F1", Value: d("7")}},
				OneOutInfererForecasterValues: []*types.OneOutInfererForecasterValues{{
					OneOutInfererValues: []*types.WithheldWorkerAttributedValue{{Worker: "inf1", Value: d("3")}},
				}},
			},
			exp: &types.EventValueBundle{
				ExtraData:                     []byte("extra"),
				CombinedValue:                 alloraMath.MustNewDecFromString("0.0112"),
				NaiveValue:                    alloraMath.MustNewDecFromString("0.0113"),
				InfererAddresses:              []string{"inf1"},
				ForecasterAddresses:           []string{"F1"},
				InfererValues:                 []alloraMath.Dec{d("1")},
				ForecasterValues:              []alloraMath.Dec{d("10")},
				OneOutInfererValues:           []alloraMath.Dec{d("100")},
				OneOutForecasterValues:        []alloraMath.Dec{d("1000")},
				OneInForecasterValues:         []alloraMath.Dec{d("7")},
				OneOutInfererForecasterValues: []alloraMath.DecArray{{d("3")}},
			},
		},
		{
			name: "1 actor; OOIF missing -> NaN padded",
			in: &types.ValueBundle{
				ExtraData:                     []byte("extra"),
				CombinedValue:                 alloraMath.MustNewDecFromString("0.0112"),
				NaiveValue:                    alloraMath.MustNewDecFromString("0.0113"),
				InfererValues:                 []*types.WorkerAttributedValue{{Worker: "inf1", Value: d("1")}},
				ForecasterValues:              []*types.WorkerAttributedValue{{Worker: "F1", Value: d("10")}},
				OneOutInfererForecasterValues: []*types.OneOutInfererForecasterValues{{ /* empty row */ }},
			},
			exp: &types.EventValueBundle{
				ExtraData:                     []byte("extra"),
				CombinedValue:                 alloraMath.MustNewDecFromString("0.0112"),
				NaiveValue:                    alloraMath.MustNewDecFromString("0.0113"),
				InfererAddresses:              []string{"inf1"},
				ForecasterAddresses:           []string{"F1"},
				InfererValues:                 []alloraMath.Dec{d("1")},
				ForecasterValues:              []alloraMath.Dec{d("10")},
				OneOutInfererForecasterValues: []alloraMath.DecArray{{nan()}},
			},
		},
		{
			name: "3 inferers; missing FIRST -> NaN then values",
			in: &types.ValueBundle{
				ExtraData:        []byte("extra"),
				CombinedValue:    alloraMath.MustNewDecFromString("0.0112"),
				NaiveValue:       alloraMath.MustNewDecFromString("0.0113"),
				InfererValues:    []*types.WorkerAttributedValue{{"inf1", d("1")}, {"inf2", d("2")}, {"inf3", d("3")}},
				ForecasterValues: []*types.WorkerAttributedValue{{"F1", d("42")}},
				OneOutInfererForecasterValues: []*types.OneOutInfererForecasterValues{{
					OneOutInfererValues: []*types.WithheldWorkerAttributedValue{
						{"inf2", d("12")}, {"inf3", d("13")},
					},
				}},
			},
			exp: &types.EventValueBundle{
				ExtraData:                     []byte("extra"),
				CombinedValue:                 alloraMath.MustNewDecFromString("0.0112"),
				NaiveValue:                    alloraMath.MustNewDecFromString("0.0113"),
				InfererAddresses:              []string{"inf1", "inf2", "inf3"},
				ForecasterAddresses:           []string{"F1"},
				InfererValues:                 []alloraMath.Dec{d("1"), d("2"), d("3")},
				ForecasterValues:              []alloraMath.Dec{d("42")},
				OneOutInfererForecasterValues: []alloraMath.DecArray{{nan(), d("12"), d("13")}},
			},
		},
		{
			name: "3 inferers; missing MIDDLE -> NaN padded",
			in: &types.ValueBundle{
				ExtraData:        []byte("extra"),
				CombinedValue:    alloraMath.MustNewDecFromString("0.0112"),
				NaiveValue:       alloraMath.MustNewDecFromString("0.0113"),
				InfererValues:    []*types.WorkerAttributedValue{{"A", d("1")}, {"B", d("2")}, {"C", d("3")}},
				ForecasterValues: []*types.WorkerAttributedValue{{"F1", d("1")}},
				OneOutInfererForecasterValues: []*types.OneOutInfererForecasterValues{{
					OneOutInfererValues: []*types.WithheldWorkerAttributedValue{
						{"A", d("10")}, {"C", d("30")},
					},
				}},
			},
			exp: &types.EventValueBundle{
				ExtraData:                     []byte("extra"),
				CombinedValue:                 alloraMath.MustNewDecFromString("0.0112"),
				NaiveValue:                    alloraMath.MustNewDecFromString("0.0113"),
				InfererAddresses:              []string{"A", "B", "C"},
				ForecasterAddresses:           []string{"F1"},
				InfererValues:                 []alloraMath.Dec{d("1"), d("2"), d("3")},
				ForecasterValues:              []alloraMath.Dec{d("1")},
				OneOutInfererForecasterValues: []alloraMath.DecArray{{d("10"), nan(), d("30")}},
			},
		},
		{
			name: "3 inferers; missing LAST -> NaN padded",
			in: &types.ValueBundle{
				ExtraData:        []byte("extra"),
				CombinedValue:    alloraMath.MustNewDecFromString("0.0112"),
				NaiveValue:       alloraMath.MustNewDecFromString("0.0113"),
				InfererValues:    []*types.WorkerAttributedValue{{"A", d("1")}, {"B", d("2")}, {"C", d("3")}},
				ForecasterValues: []*types.WorkerAttributedValue{{"F1", d("1")}},
				OneOutInfererForecasterValues: []*types.OneOutInfererForecasterValues{{
					OneOutInfererValues: []*types.WithheldWorkerAttributedValue{
						{"A", d("10")}, {"B", d("20")},
					},
				}},
			},
			exp: &types.EventValueBundle{
				ExtraData:                     []byte("extra"),
				CombinedValue:                 alloraMath.MustNewDecFromString("0.0112"),
				NaiveValue:                    alloraMath.MustNewDecFromString("0.0113"),
				InfererAddresses:              []string{"A", "B", "C"},
				ForecasterAddresses:           []string{"F1"},
				InfererValues:                 []alloraMath.Dec{d("1"), d("2"), d("3")},
				ForecasterValues:              []alloraMath.Dec{d("1")},
				OneOutInfererForecasterValues: []alloraMath.DecArray{{d("10"), d("20"), nan()}},
			},
		},
		{
			name: "multiple forecasters, different gaps; order preserved",
			in: &types.ValueBundle{
				ExtraData:        []byte("extra"),
				CombinedValue:    alloraMath.MustNewDecFromString("0.0112"),
				NaiveValue:       alloraMath.MustNewDecFromString("0.0113"),
				InfererValues:    []*types.WorkerAttributedValue{{"inf1", d("1")}, {"inf2", d("2")}, {"inf3", d("3")}},
				ForecasterValues: []*types.WorkerAttributedValue{{"F1", d("9")}, {"F2", d("8")}, {"F3", d("7")}},
				OneOutInfererForecasterValues: []*types.OneOutInfererForecasterValues{
					{OneOutInfererValues: []*types.WithheldWorkerAttributedValue{{"inf1", d("11")}, {"inf3", d("13")}}},
					{OneOutInfererValues: []*types.WithheldWorkerAttributedValue{{"inf2", d("22")}}},
					{OneOutInfererValues: buildFullRowOOIF(3)}, // inf1..3 => 1..3
				},
			},
			exp: &types.EventValueBundle{
				ExtraData:           []byte("extra"),
				CombinedValue:       alloraMath.MustNewDecFromString("0.0112"),
				NaiveValue:          alloraMath.MustNewDecFromString("0.0113"),
				InfererAddresses:    []string{"inf1", "inf2", "inf3"},
				ForecasterAddresses: []string{"F1", "F2", "F3"},
				InfererValues:       []alloraMath.Dec{d("1"), d("2"), d("3")},
				ForecasterValues:    []alloraMath.Dec{d("9"), d("8"), d("7")},
				OneOutInfererForecasterValues: []alloraMath.DecArray{
					{d("11"), nan(), d("13")},
					{nan(), d("22"), nan()},
					{d("1"), d("2"), d("3")},
				},
			},
		},
		{
			name: "unknown inferer ignored; padding still matches known inferers",
			in: &types.ValueBundle{
				ExtraData:        []byte("extra"),
				CombinedValue:    alloraMath.MustNewDecFromString("0.0112"),
				NaiveValue:       alloraMath.MustNewDecFromString("0.0113"),
				InfererValues:    []*types.WorkerAttributedValue{{"X", d("1")}, {"Y", d("2")}},
				ForecasterValues: []*types.WorkerAttributedValue{{"F", d("9")}},
				OneOutInfererForecasterValues: []*types.OneOutInfererForecasterValues{
					{OneOutInfererValues: []*types.WithheldWorkerAttributedValue{{"X", d("5")}, {"Z", d("999")}}},
				},
			},
			exp: &types.EventValueBundle{
				ExtraData:                     []byte("extra"),
				CombinedValue:                 alloraMath.MustNewDecFromString("0.0112"),
				NaiveValue:                    alloraMath.MustNewDecFromString("0.0113"),
				InfererAddresses:              []string{"X", "Y"},
				ForecasterAddresses:           []string{"F"},
				InfererValues:                 []alloraMath.Dec{d("1"), d("2")},
				ForecasterValues:              []alloraMath.Dec{d("9")},
				OneOutInfererForecasterValues: []alloraMath.DecArray{{d("5"), nan()}},
			},
		},
		{
			name: "N actors (no gaps)",
			in: &types.ValueBundle{
				ExtraData:                     []byte("extra"),
				CombinedValue:                 alloraMath.MustNewDecFromString("0.0112"),
				NaiveValue:                    alloraMath.MustNewDecFromString("0.0113"),
				InfererValues:                 buildFullRowWA(4),
				ForecasterValues:              []*types.WorkerAttributedValue{{Worker: "F", Value: d("0")}},
				OneOutInfererForecasterValues: []*types.OneOutInfererForecasterValues{{OneOutInfererValues: buildFullRowOOIF(4)}},
			},
			exp: &types.EventValueBundle{
				ExtraData:                     []byte("extra"),
				CombinedValue:                 alloraMath.MustNewDecFromString("0.0112"),
				NaiveValue:                    alloraMath.MustNewDecFromString("0.0113"),
				InfererAddresses:              []string{"inf1", "inf2", "inf3", "inf4"},
				ForecasterAddresses:           []string{"F"},
				InfererValues:                 []alloraMath.Dec{d("1"), d("2"), d("3"), d("4")},
				ForecasterValues:              []alloraMath.Dec{d("0")},
				OneOutInfererForecasterValues: []alloraMath.DecArray{{d("1"), d("2"), d("3"), d("4")}},
			},
		},
		{
			name: "N = max actors (no gaps)",
			in: &types.ValueBundle{
				ExtraData:                     []byte("extra"),
				CombinedValue:                 alloraMath.MustNewDecFromString("0.0112"),
				NaiveValue:                    alloraMath.MustNewDecFromString("0.0113"),
				InfererValues:                 buildFullRowWA(maxInferers),
				ForecasterValues:              []*types.WorkerAttributedValue{{Worker: "F", Value: d("0")}},
				OneOutInfererForecasterValues: []*types.OneOutInfererForecasterValues{{OneOutInfererValues: buildFullRowOOIF(maxInferers)}},
			},
			exp: func() *types.EventValueBundle {
				addrs := make([]string, 0, maxInferers)
				vals := make([]alloraMath.Dec, 0, maxInferers)
				row := make([]alloraMath.Dec, 0, maxInferers)
				for i := 1; i <= maxInferers; i++ {
					addrs = append(addrs, fmt.Sprintf("inf%d", i))
					vals = append(vals, d(fmt.Sprintf("%d", i)))
					row = append(row, d(fmt.Sprintf("%d", i)))
				}
				return &types.EventValueBundle{
					ExtraData:                     []byte("extra"),
					CombinedValue:                 alloraMath.MustNewDecFromString("0.0112"),
					NaiveValue:                    alloraMath.MustNewDecFromString("0.0113"),
					InfererAddresses:              addrs,
					ForecasterAddresses:           []string{"F"},
					InfererValues:                 vals,
					ForecasterValues:              []alloraMath.Dec{d("0")},
					OneOutInfererForecasterValues: []alloraMath.DecArray{row},
				}
			}(),
		},
		{
			name: "permuted input order; output uses infererAddresses order",
			in: &types.ValueBundle{
				ExtraData:        []byte("extra"),
				CombinedValue:    alloraMath.MustNewDecFromString("0.0112"),
				NaiveValue:       alloraMath.MustNewDecFromString("0.0113"),
				InfererValues:    []*types.WorkerAttributedValue{{"B", d("2")}, {"A", d("1")}, {"C", d("3")}},
				ForecasterValues: []*types.WorkerAttributedValue{{"F1", d("9")}},
				OneOutInfererForecasterValues: []*types.OneOutInfererForecasterValues{
					{OneOutInfererValues: []*types.WithheldWorkerAttributedValue{{"C", d("33")}, {"A", d("11")}}},
				},
			},
			exp: &types.EventValueBundle{
				ExtraData:                     []byte("extra"),
				CombinedValue:                 alloraMath.MustNewDecFromString("0.0112"),
				NaiveValue:                    alloraMath.MustNewDecFromString("0.0113"),
				InfererAddresses:              []string{"B", "A", "C"},
				ForecasterAddresses:           []string{"F1"},
				InfererValues:                 []alloraMath.Dec{d("2"), d("1"), d("3")},
				ForecasterValues:              []alloraMath.Dec{d("9")},
				OneOutInfererForecasterValues: []alloraMath.DecArray{{nan(), d("11"), d("33")}},
			},
		},
		{
			name: "actors exist but OOIF rows nil",
			in: &types.ValueBundle{
				ExtraData:                     []byte("extra"),
				CombinedValue:                 alloraMath.MustNewDecFromString("0.0112"),
				NaiveValue:                    alloraMath.MustNewDecFromString("0.0113"),
				InfererValues:                 []*types.WorkerAttributedValue{{"inf1", d("1")}, {"inf2", d("2")}},
				ForecasterValues:              []*types.WorkerAttributedValue{{"F1", d("9")}, {"F2", d("8")}},
				OneOutInfererForecasterValues: nil,
			},
			exp: &types.EventValueBundle{
				ExtraData:                     []byte("extra"),
				CombinedValue:                 alloraMath.MustNewDecFromString("0.0112"),
				NaiveValue:                    alloraMath.MustNewDecFromString("0.0113"),
				InfererAddresses:              []string{"inf1", "inf2"},
				ForecasterAddresses:           []string{"F1", "F2"},
				InfererValues:                 []alloraMath.Dec{d("1"), d("2")},
				ForecasterValues:              []alloraMath.Dec{d("9"), d("8")},
				OneOutInfererForecasterValues: nil,
			},
		},
		{
			name: "all auxiliary vectors present; OOIF mixed gaps",
			in: &types.ValueBundle{
				ExtraData:              []byte("extra"),
				CombinedValue:          alloraMath.MustNewDecFromString("0.0112"),
				NaiveValue:             alloraMath.MustNewDecFromString("0.0113"),
				InfererValues:          []*types.WorkerAttributedValue{{"i1", d("1")}, {"i2", d("2")}, {"i3", d("3")}, {"i4", d("4")}},
				ForecasterValues:       []*types.WorkerAttributedValue{{"f1", d("10")}, {"f2", d("20")}},
				OneOutInfererValues:    []*types.WithheldWorkerAttributedValue{{"i1", d("100")}, {"i2", d("200")}, {"i3", d("300")}, {"i4", d("400")}},
				OneOutForecasterValues: []*types.WithheldWorkerAttributedValue{{"f1", d("1000")}, {"f2", d("2000")}},
				OneInForecasterValues:  []*types.WorkerAttributedValue{{"f1", d("7")}, {"f2", d("8")}},
				OneOutInfererForecasterValues: []*types.OneOutInfererForecasterValues{
					{OneOutInfererValues: []*types.WithheldWorkerAttributedValue{{"i1", d("11")}, {"i4", d("14")}}},
					{OneOutInfererValues: []*types.WithheldWorkerAttributedValue{{"i2", d("22")}, {"i3", d("23")}}},
				},
			},
			exp: &types.EventValueBundle{
				ExtraData:              []byte("extra"),
				CombinedValue:          alloraMath.MustNewDecFromString("0.0112"),
				NaiveValue:             alloraMath.MustNewDecFromString("0.0113"),
				InfererAddresses:       []string{"i1", "i2", "i3", "i4"},
				ForecasterAddresses:    []string{"f1", "f2"},
				InfererValues:          []alloraMath.Dec{d("1"), d("2"), d("3"), d("4")},
				ForecasterValues:       []alloraMath.Dec{d("10"), d("20")},
				OneOutInfererValues:    []alloraMath.Dec{d("100"), d("200"), d("300"), d("400")},
				OneOutForecasterValues: []alloraMath.Dec{d("1000"), d("2000")},
				OneInForecasterValues:  []alloraMath.Dec{d("7"), d("8")},
				OneOutInfererForecasterValues: []alloraMath.DecArray{
					{d("11"), nan(), nan(), d("14")},
					{nan(), d("22"), d("23"), nan()},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.exp, types.ValueBundleToEventValueBundleBase(tc.in))
		})
	}
}

func assertEventValueBundle(t *testing.T, val string, bundle types.ValueBundle) {
	t.Helper()

	var got types.EventValueBundle
	require.NoError(t, json.Unmarshal([]byte(val), &got))

	require.True(t, bundle.CombinedValue.Equal(got.CombinedValue))
	require.True(t, bundle.NaiveValue.Equal(got.NaiveValue))

	require.Equal(t, len(bundle.InfererValues), len(got.InfererValues))
	for i := range bundle.InfererValues {
		require.True(t, bundle.InfererValues[i].Value.Equal(got.InfererValues[i]))
	}
	require.Equal(t, len(bundle.ForecasterValues), len(got.ForecasterValues))
	for i := range bundle.ForecasterValues {
		require.True(t, bundle.ForecasterValues[i].Value.Equal(got.ForecasterValues[i]))
	}
	require.Equal(t, len(bundle.OneOutInfererValues), len(got.OneOutInfererValues))
	for i := range bundle.OneOutInfererValues {
		require.True(t, bundle.OneOutInfererValues[i].Value.Equal(got.OneOutInfererValues[i]))
	}
	require.Equal(t, len(bundle.OneOutForecasterValues), len(got.OneOutForecasterValues))
	for i := range bundle.OneOutForecasterValues {
		require.True(t, bundle.OneOutForecasterValues[i].Value.Equal(got.OneOutForecasterValues[i]))
	}
	require.Equal(t, len(bundle.OneInForecasterValues), len(got.OneInForecasterValues))
	for i := range bundle.OneInForecasterValues {
		require.True(t, bundle.OneInForecasterValues[i].Value.Equal(got.OneInForecasterValues[i]))
	}

	colIndex := make(map[string]int, len(got.InfererAddresses))
	for j, addr := range got.InfererAddresses {
		colIndex[addr] = j
	}

	require.Equal(t, len(bundle.OneOutInfererForecasterValues), len(got.OneOutInfererForecasterValues))

	for i := range bundle.OneOutInfererForecasterValues {
		expected := make([]alloraMath.Dec, len(got.InfererAddresses))
		for j := range expected {
			expected[j] = alloraMath.NewNaN()
		}

		for _, iv := range bundle.OneOutInfererForecasterValues[i].OneOutInfererValues {
			if j, ok := colIndex[iv.Worker]; ok {
				expected[j] = iv.Value
			}
		}

		require.Equal(t, len(expected), len(got.OneOutInfererForecasterValues[i]),
			"row %d length must equal number of inferers", i)

		// handle case where nan is added to complete the matrix
		for j := range expected {
			ge := got.OneOutInfererForecasterValues[i][j]
			ex := expected[j]
			if ex.IsNaN() && ge.IsNaN() {
				continue
			}
			require.Truef(t, ge.Equal(ex), "row %d col %d: got %v want %v", i, j, ge, ex)
		}
	}
}

func TestEmitNewForecastTaskSetEvent(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	topicId := uint64(1)
	nonce := int64(10)
	CombinedValue := alloraMath.MustNewDecFromString("10")
	NaiveValue := alloraMath.MustNewDecFromString("20")

	score, err := NaiveValue.Sub(CombinedValue)
	require.NoError(t, err)

	types.EmitNewForecastTaskUtilityScoreSetEvent(ctx, topicId, score, nonce)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "emissions.v10.EventForecastTaskScoreSet", event.Type)

	// Check that we have the expected number of attributes
	require.Len(t, event.Attributes, 3)

	// Create a map to check attributes regardless of order
	attributes := make(map[string]string)
	for _, attr := range event.Attributes {
		attributes[attr.Key] = attr.Value
	}

	// Verify all expected attributes are present with correct values
	require.Contains(t, attributes, "topic_id")
	require.Equal(t, "\"1\"", attributes["topic_id"])

	require.Contains(t, attributes, "score")
	require.Equal(t, "\"10\"", attributes["score"])

	require.Contains(t, attributes, "nonce_block_height")
	require.Equal(t, "\"10\"", attributes["nonce_block_height"])
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

	require.Equal(t, "emissions.v10.EventWorkerLastCommitSet", events[0].Type)
	require.Equal(t, "emissions.v10.EventWorkerLastCommitSet", events[1].Type)
	require.Equal(t, "emissions.v10.EventReputerLastCommitSet", events[2].Type)

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

	require.Equal(t, "emissions.v10.EventTopicRewardsSet", events[0].Type)
	require.Contains(t, events[0].Attributes[0].Key, "rewards")
	require.Contains(t, events[0].Attributes[0].Value, `["0","10","20","30","40"]`)
	require.Contains(t, events[0].Attributes[1].Key, "topic_ids")
	require.Contains(t, events[0].Attributes[1].Value, `["1","2","3","4","5"]`)
}

func TestEmitNewEMAScoresSetEventWithScores(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	activeArr := make(map[string]bool)
	emaScores := []types.Score{
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

	activeArr[emaScores[0].Address] = true
	activeArr[emaScores[1].Address] = false
	types.EmitNewActorEMAScoresSetEvent(ctx, types.ActorType_ACTOR_TYPE_INFERER_UNSPECIFIED, int64(10), emaScores, activeArr)
	activeArr[emaScores[0].Address] = false
	activeArr[emaScores[1].Address] = false
	types.EmitNewActorEMAScoresSetEvent(ctx, types.ActorType_ACTOR_TYPE_FORECASTER, int64(10), emaScores, activeArr)
	activeArr[emaScores[0].Address] = true
	activeArr[emaScores[1].Address] = true
	types.EmitNewActorEMAScoresSetEvent(ctx, types.ActorType_ACTOR_TYPE_REPUTER, int64(10), emaScores, activeArr)

	events := ctx.EventManager().Events()
	require.Len(t, events, 3)

	event := events[0]
	require.Equal(t, "emissions.v10.EventEMAScoresSet", event.Type)

	require.Contains(t, events[0].Attributes[0].Key, "actor_type")
	require.Contains(t, events[0].Attributes[0].Value, "\"ACTOR_TYPE_INFERER_UNSPECIFIED\"")
	require.Contains(t, events[0].Attributes[1].Key, "addresses")
	require.Contains(t, events[0].Attributes[1].Value, "[\"address1\",\"address2\"]")
	require.Contains(t, events[0].Attributes[2].Key, "is_active")
	require.Contains(t, events[0].Attributes[2].Value, "[true,false]")
	require.Contains(t, events[0].Attributes[3].Key, "nonce")
	require.Contains(t, events[0].Attributes[3].Value, "\"10\"")
	require.Contains(t, events[0].Attributes[4].Key, "scores")
	require.Contains(t, events[0].Attributes[4].Value, "[\"100\",\"200\"]")
	require.Contains(t, events[0].Attributes[5].Key, "topic_id")
	require.Contains(t, events[0].Attributes[5].Value, "\"1\"")

	require.Contains(t, events[1].Attributes[0].Key, "actor_type")
	require.Contains(t, events[1].Attributes[0].Value, "\"ACTOR_TYPE_FORECASTER\"")
	require.Contains(t, events[1].Attributes[2].Key, "is_active")
	require.Contains(t, events[1].Attributes[2].Value, "[false,false]")

	require.Contains(t, events[2].Attributes[0].Key, "actor_type")
	require.Contains(t, events[2].Attributes[0].Value, "\"ACTOR_TYPE_REPUTER\"")
	require.Contains(t, events[2].Attributes[2].Key, "is_active")
	require.Contains(t, events[2].Attributes[2].Value, "[true,true]")
}

func TestEmitNewListeningCoefficientsSetEvent(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())

	actorType := types.ActorType_ACTOR_TYPE_REPUTER
	topicID := uint64(1)
	blockHeight := int64(10)
	addresses := []string{"address1", "address2"}
	coefficients := []alloraMath.Dec{alloraMath.NewDecFromInt64(100), alloraMath.NewDecFromInt64(200)}

	types.EmitNewListeningCoefficientsSetEvent(ctx, actorType, topicID, blockHeight, addresses, coefficients)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "emissions.v10.EventListeningCoefficientsSet", event.Type)

	attributes := event.Attributes
	require.Len(t, attributes, 5)

	val, exists := event.GetAttribute(AttributeKeyActorType)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "ACTOR_TYPE_REPUTER")

	val, exists = event.GetAttribute(AttributeKeyTopicId)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "1")

	val, exists = event.GetAttribute(AttributeKeyBlockHeight)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "10")

	val, exists = event.GetAttribute(AttributeKeyAddresses)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), `["address1","address2"]`)

	val, exists = event.GetAttribute(AttributeKeyCoefficients)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), `["100","200"]`)
}

func TestEmitNewReputerScoresSetEventWithNoAddresses(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	actorType := types.ActorType_ACTOR_TYPE_REPUTER
	topicID := uint64(1)
	blockHeight := int64(10)
	addresses := []string{}
	coefficients := []alloraMath.Dec{}

	types.EmitNewListeningCoefficientsSetEvent(ctx, actorType, topicID, blockHeight, addresses, coefficients)

	events := ctx.EventManager().Events()
	require.Empty(t, events)
}

func TestEmitNewReputerScoresSetEventWithNoCoefficients(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	actorType := types.ActorType_ACTOR_TYPE_REPUTER
	topicID := uint64(1)
	blockHeight := int64(10)
	addresses := []string{"address1", "address2"}
	coefficients := []alloraMath.Dec{}

	types.EmitNewListeningCoefficientsSetEvent(ctx, actorType, topicID, blockHeight, addresses, coefficients)

	events := ctx.EventManager().Events()
	require.Empty(t, events)
}

func TestEmitNewInfererNetworkRegretSetEvent(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	topicID := uint64(1)
	blockHeight := int64(10)
	addresses := []string{"address1", "address2"}
	regrets := []alloraMath.Dec{alloraMath.NewDecFromInt64(100), alloraMath.NewDecFromInt64(200)}

	types.EmitNewInfererNetworkRegretSetEvent(ctx, topicID, blockHeight, addresses, regrets)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "emissions.v10.EventInfererNetworkRegretSet", event.Type)

	val, exists := event.GetAttribute(AttributeKeyTopicId)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "1")

	val, exists = event.GetAttribute(AttributeKeyBlockHeight)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "10")

	val, exists = event.GetAttribute(AttributeKeyAddresses)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), `["address1","address2"]`)

	val, exists = event.GetAttribute(AttributeKeyRegrets)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), `["100","200"]`)
}

func TestEmitNewInfererNetworkRegretSetEventWithNoAddresses(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	topicID := uint64(1)
	blockHeight := int64(10)
	addresses := []string{}
	regrets := []alloraMath.Dec{alloraMath.NewDecFromInt64(100), alloraMath.NewDecFromInt64(200)}

	types.EmitNewInfererNetworkRegretSetEvent(ctx, topicID, blockHeight, addresses, regrets)
	events := ctx.EventManager().Events()
	require.Empty(t, events)
}

func TestEmitNewInfererNetworkRegretSetEventWithNoRegrets(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	topicID := uint64(1)
	blockHeight := int64(10)
	addresses := []string{"address1", "address2"}
	regrets := []alloraMath.Dec{}

	types.EmitNewInfererNetworkRegretSetEvent(ctx, topicID, blockHeight, addresses, regrets)

	events := ctx.EventManager().Events()
	require.Empty(t, events)
}

func TestEmitNewForecasterNetworkRegretSetEvent(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	topicID := uint64(1)
	blockHeight := int64(10)
	addresses := []string{"address1", "address2"}
	regrets := []alloraMath.Dec{alloraMath.NewDecFromInt64(100), alloraMath.NewDecFromInt64(200)}

	types.EmitNewForecasterNetworkRegretSetEvent(ctx, topicID, blockHeight, addresses, regrets)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "emissions.v10.EventForecasterNetworkRegretSet", event.Type)

	val, exists := event.GetAttribute(AttributeKeyTopicId)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "1")

	val, exists = event.GetAttribute(AttributeKeyBlockHeight)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "10")

	val, exists = event.GetAttribute(AttributeKeyAddresses)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), `["address1","address2"]`)

	val, exists = event.GetAttribute(AttributeKeyRegrets)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), `["100","200"]`)
}

func TestEmitNewForecasterNetworkRegretSetEventWithNoAddresses(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	topicID := uint64(1)
	blockHeight := int64(10)
	addresses := []string{}
	regrets := []alloraMath.Dec{alloraMath.NewDecFromInt64(100), alloraMath.NewDecFromInt64(200)}

	types.EmitNewForecasterNetworkRegretSetEvent(ctx, topicID, blockHeight, addresses, regrets)
	events := ctx.EventManager().Events()
	require.Empty(t, events)
}

func TestEmitNewForecasterNetworkRegretSetEventWithNoRegrets(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	topicID := uint64(1)
	blockHeight := int64(10)
	addresses := []string{"address1", "address2"}
	regrets := []alloraMath.Dec{}

	types.EmitNewForecasterNetworkRegretSetEvent(ctx, topicID, blockHeight, addresses, regrets)
	events := ctx.EventManager().Events()
	require.Empty(t, events)
}

func TestEmitNewNaiveInfererNetworkRegretSetEvent(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	topicID := uint64(1)
	blockHeight := int64(10)
	addresses := []string{"address1", "address2"}
	regrets := []alloraMath.Dec{alloraMath.NewDecFromInt64(100), alloraMath.NewDecFromInt64(200)}

	types.EmitNewNaiveInfererNetworkRegretSetEvent(ctx, topicID, blockHeight, addresses, regrets)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "emissions.v10.EventNaiveInfererNetworkRegretSet", event.Type)

	val, exists := event.GetAttribute(AttributeKeyTopicId)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "1")

	val, exists = event.GetAttribute(AttributeKeyBlockHeight)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "10")

	val, exists = event.GetAttribute(AttributeKeyAddresses)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), `["address1","address2"]`)

	val, exists = event.GetAttribute(AttributeKeyRegrets)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), `["100","200"]`)
}

func TestEmitNewNaiveInfererNetworkRegretSetEventWithNoAddresses(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	topicID := uint64(1)
	blockHeight := int64(10)
	addresses := []string{}
	regrets := []alloraMath.Dec{alloraMath.NewDecFromInt64(100), alloraMath.NewDecFromInt64(200)}

	types.EmitNewNaiveInfererNetworkRegretSetEvent(ctx, topicID, blockHeight, addresses, regrets)
	events := ctx.EventManager().Events()
	require.Empty(t, events)
}

func TestEmitNewNaiveInfererNetworkRegretSetEventWithNoRegrets(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	topicID := uint64(1)
	blockHeight := int64(10)
	addresses := []string{"address1", "address2"}
	regrets := []alloraMath.Dec{}

	types.EmitNewNaiveInfererNetworkRegretSetEvent(ctx, topicID, blockHeight, addresses, regrets)
	events := ctx.EventManager().Events()
	require.Empty(t, events)
}

func TestEmitNewTopicInitialRegretSetEvent(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	topicID := uint64(1)
	blockHeight := int64(10)
	regret := alloraMath.NewDecFromInt64(100)

	types.EmitNewTopicInitialRegretSetEvent(ctx, topicID, blockHeight, regret)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "emissions.v10.EventTopicInitialRegretSet", event.Type)

	val, exists := event.GetAttribute(AttributeKeyTopicId)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "1")

	val, exists = event.GetAttribute(AttributeKeyBlockHeight)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "10")

	val, exists = event.GetAttribute(AttributeKeyRegret)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "100")
}

func TestEmitPreviousPercentageRewardToStakedReputersSetEvent(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	blockHeight := int64(10)
	percentage := alloraMath.MustNewDecFromString("0.75")

	types.EmitNewPreviousPercentageRewardToStakedReputersSetEvent(ctx, blockHeight, percentage)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "emissions.v10.EventPreviousPercentageRewardToStakedReputersSet", event.Type)

	val, exists := event.GetAttribute(AttributeKeyBlockHeight)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), strconv.FormatInt(blockHeight, 10))

	val, exists = event.GetAttribute("percentage")
	require.True(t, exists)
	require.Contains(t, val.GetValue(), percentage.String())
}

func TestEmitNewInfererWeightsSetEvent(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	topicId := uint64(1)
	blockHeight := int64(10)
	addresses := []string{"address1", "address2"}
	weights := []alloraMath.Dec{alloraMath.MustNewDecFromString("0.5"), alloraMath.MustNewDecFromString("0.3")}

	types.EmitNewInfererWeightsSetEvent(ctx, topicId, blockHeight, addresses, weights)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "emissions.v10.EventInfererWeightsSet", event.Type)

	val, exists := event.GetAttribute(AttributeKeyTopicId)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "1")

	val, exists = event.GetAttribute(AttributeKeyBlockHeight)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "10")

	val, exists = event.GetAttribute(AttributeKeyAddresses)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), `["address1","address2"]`)

	val, exists = event.GetAttribute(AttributeKeyWeights)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), `["0.5","0.3"]`)
}

func TestEmitNewInfererWeightsSetEventWithNoAddresses(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	topicId := uint64(1)
	blockHeight := int64(10)
	addresses := []string{}
	weights := []alloraMath.Dec{alloraMath.MustNewDecFromString("0.5"), alloraMath.MustNewDecFromString("0.3")}

	types.EmitNewInfererWeightsSetEvent(ctx, topicId, blockHeight, addresses, weights)
	events := ctx.EventManager().Events()
	require.Empty(t, events)
}

func TestEmitNewInfererWeightsSetEventWithNoWeights(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	topicId := uint64(1)
	blockHeight := int64(10)
	addresses := []string{"address1", "address2"}
	weights := []alloraMath.Dec{}

	types.EmitNewInfererWeightsSetEvent(ctx, topicId, blockHeight, addresses, weights)
	events := ctx.EventManager().Events()
	require.Empty(t, events)
}

func TestEmitNewForecasterWeightsSetEvent(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	topicId := uint64(1)
	blockHeight := int64(10)
	addresses := []string{"address1", "address2"}
	weights := []alloraMath.Dec{alloraMath.MustNewDecFromString("0.5"), alloraMath.MustNewDecFromString("0.3")}

	types.EmitNewForecasterWeightsSetEvent(ctx, topicId, blockHeight, addresses, weights)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "emissions.v10.EventForecasterWeightsSet", event.Type)

	val, exists := event.GetAttribute(AttributeKeyTopicId)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "1")

	val, exists = event.GetAttribute(AttributeKeyBlockHeight)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "10")

	val, exists = event.GetAttribute(AttributeKeyAddresses)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), `["address1","address2"]`)

	val, exists = event.GetAttribute(AttributeKeyWeights)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), `["0.5","0.3"]`)
}

func TestEmitNewForecasterWeightsSetEventWithNoAddresses(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	topicId := uint64(1)
	blockHeight := int64(10)
	addresses := []string{}
	weights := []alloraMath.Dec{alloraMath.MustNewDecFromString("0.5"), alloraMath.MustNewDecFromString("0.3")}

	types.EmitNewForecasterWeightsSetEvent(ctx, topicId, blockHeight, addresses, weights)
	events := ctx.EventManager().Events()
	require.Empty(t, events)
}

func TestEmitNewForecasterWeightsSetEventWithNoWeights(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	topicId := uint64(1)
	blockHeight := int64(10)
	addresses := []string{"address1", "address2"}
	weights := []alloraMath.Dec{}

	types.EmitNewForecasterWeightsSetEvent(ctx, topicId, blockHeight, addresses, weights)
	events := ctx.EventManager().Events()
	require.Empty(t, events)
}

func TestEmitNewNetworkInferenceInfererWeightsSetEvent(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	topicId := uint64(1)
	blockHeight := int64(10)
	addresses := []string{"address1", "address2"}
	weights := []alloraMath.Dec{alloraMath.MustNewDecFromString("0.5"), alloraMath.MustNewDecFromString("0.3")}

	types.EmitNewNetworkInferenceInfererWeightsSetEvent(ctx, topicId, blockHeight, addresses, weights)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "emissions.v10.EventNetworkInferenceInfererWeightsSet", event.Type)

	val, exists := event.GetAttribute(AttributeKeyTopicId)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "1")

	val, exists = event.GetAttribute("nonce_block_height")
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "10")

	val, exists = event.GetAttribute(AttributeKeyAddresses)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), `["address1","address2"]`)

	val, exists = event.GetAttribute(AttributeKeyWeights)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), `["0.5","0.3"]`)
}

func TestEmitNewNetworkInferenceForecasterWeightsSetEvent(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	topicId := uint64(1)
	blockHeight := int64(10)
	addresses := []string{"address1", "address2"}
	weights := []alloraMath.Dec{alloraMath.MustNewDecFromString("0.5"), alloraMath.MustNewDecFromString("0.3")}

	types.EmitNewNetworkInferenceForecasterWeightsSetEvent(ctx, topicId, blockHeight, addresses, weights)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "emissions.v10.EventNetworkInferenceForecasterWeightsSet", event.Type)

	val, exists := event.GetAttribute(AttributeKeyTopicId)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "1")

	val, exists = event.GetAttribute("nonce_block_height")
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "10")

	val, exists = event.GetAttribute(AttributeKeyAddresses)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), `["address1","address2"]`)

	val, exists = event.GetAttribute(AttributeKeyWeights)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), `["0.5","0.3"]`)
}

func TestEmitNewTopicFeeRevenueDrippedEvent(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	topicId := uint64(1)
	oldRevenue := cosmosMath.NewInt(1000)
	newRevenue := cosmosMath.NewInt(950)
	dripAmount := cosmosMath.NewInt(50)

	types.EmitNewTopicFeeRevenueDrippedEvent(ctx, topicId, oldRevenue, newRevenue, dripAmount)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "emissions.v10.EventTopicFeeRevenueDripped", event.Type)

	val, exists := event.GetAttribute(AttributeKeyTopicId)
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "1")

	val, exists = event.GetAttribute("old_revenue")
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "1000")

	val, exists = event.GetAttribute("new_revenue")
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "950")

	val, exists = event.GetAttribute("drip_amount")
	require.True(t, exists)
	require.Contains(t, val.GetValue(), "50")
}
