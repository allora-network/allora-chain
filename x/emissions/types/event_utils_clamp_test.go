package types

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	alloraMath "github.com/allora-network/allora-chain/math"
)

// Shared vectors: an input mix and its clamped expectation (window is 1e±40).
var (
	cTiny   = alloraMath.MustNewDecFromString("1.5e-37962")
	cHuge   = alloraMath.MustNewDecFromString("1e50")
	cMid    = alloraMath.MustNewDecFromString("0.5")
	cNeg    = alloraMath.MustNewDecFromString("-1.5e-37962")
	cMin    = alloraMath.MustNewDecFromString("1e-40")
	cMax    = alloraMath.MustNewDecFromString("1e40")
	cNegMin = alloraMath.MustNewDecFromString("-1e-40")
)

func requireDecsEqual(t *testing.T, got, want []alloraMath.Dec) {
	t.Helper()
	require.Equal(t, len(want), len(got))
	for i := range want {
		require.Truef(t, got[i].Equal(want[i]), "index %d: got %s want %s", i, got[i].String(), want[i].String())
	}
}

// Every event constructor carrying a []Dec value field must clamp it.
//
//nolint:exhaustruct,forcetypeassert // partial inputs + event type assertions are intentional in tests
func TestEventClamp_SliceFields(t *testing.T) {
	actor := ActorType_ACTOR_TYPE_INFERER_UNSPECIFIED
	addrs := []string{"a", "b", "c", "d"}
	mix := []alloraMath.Dec{cTiny, cHuge, cMid, cNeg}
	want := []alloraMath.Dec{cMin, cMax, cMid, cNegMin}

	scoresFrom := func(vs []alloraMath.Dec) []Score {
		out := make([]Score, len(vs))
		for i, v := range vs {
			out[i] = Score{TopicId: 1, BlockHeight: 1, Address: addrs[i], Score: v}
		}
		return out
	}
	rewardsFrom := func(vs []alloraMath.Dec) []TaskReward {
		out := make([]TaskReward, len(vs))
		for i, v := range vs {
			out[i] = TaskReward{TopicId: 1, Address: addrs[i], Reward: v}
		}
		return out
	}
	topicRewards := map[uint64]*alloraMath.Dec{}
	for i := range mix {
		v := mix[i]
		topicRewards[uint64(i+1)] = &v
	}

	cases := []struct {
		name string
		got  []alloraMath.Dec
	}{
		{"ListeningCoefficientsSet", NewListeningCoefficientsSetEventBase(1, 1, addrs, actor, mix).(*EventListeningCoefficientsSet).Coefficients},
		{"InfererNetworkRegretSet", NewInfererNetworkRegretSetEventBase(1, 1, addrs, mix).(*EventInfererNetworkRegretSet).Regrets},
		{"ForecasterNetworkRegretSet", NewForecasterNetworkRegretSetEventBase(1, 1, addrs, mix).(*EventForecasterNetworkRegretSet).Regrets},
		{"NaiveInfererNetworkRegretSet", NewNaiveInfererNetworkRegretSetEventBase(1, 1, addrs, mix).(*EventNaiveInfererNetworkRegretSet).Regrets},
		{"InfererWeightsSet", NewInfererWeightsSetEventBase(1, 1, addrs, mix).(*EventInfererWeightsSet).Weights},
		{"ForecasterWeightsSet", NewForecasterWeightsSetEventBase(1, 1, addrs, mix).(*EventForecasterWeightsSet).Weights},
		{"NetworkInferenceInfererWeightsSet", NewNetworkInferenceInfererWeightsSetEventBase(1, 1, addrs, mix).(*EventNetworkInferenceInfererWeightsSet).Weights},
		{"NetworkInferenceForecasterWeightsSet", NewNetworkInferenceForecasterWeightsSetEventBase(1, 1, addrs, mix).(*EventNetworkInferenceForecasterWeightsSet).Weights},
		{"NetworkInferenceInfererRegretsUsedSet", NewNetworkInferenceInfererRegretsUsedSetEventBase(1, 1, addrs, mix).(*EventNetworkInferenceInfererRegretsUsedSet).Regrets},
		{"NetworkInferenceForecasterRegretsUsedSet", NewNetworkInferenceForecasterRegretsUsedSetEventBase(1, 1, addrs, mix).(*EventNetworkInferenceForecasterRegretsUsedSet).Regrets},
		{"ScoresSet", NewScoresSetEventBase(actor, scoresFrom(mix)).(*EventScoresSet).Scores},
		{"EMAScoresSet", NewEMAScoresSetEventBase(actor, 1, scoresFrom(mix), map[string]bool{}).(*EventEMAScoresSet).Scores},
		{"RewardsSettled", NewRewardsSetEventBase(actor, 1, 1, rewardsFrom(mix)).(*EventRewardsSettled).Rewards},
		{"TopicRewardsSet", NewTopicRewardSetEventBase(topicRewards).(*EventTopicRewardsSet).Rewards},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { requireDecsEqual(t, tc.got, want) })
	}
}

// Every event constructor carrying a scalar Dec value field must clamp it.
//
//nolint:forcetypeassert // event type assertions are intentional in tests
func TestEventClamp_ScalarFields(t *testing.T) {
	actor := ActorType_ACTOR_TYPE_INFERER_UNSPECIFIED
	one := math.NewInt(1)
	cases := []struct {
		name string
		got  alloraMath.Dec
		want alloraMath.Dec
	}{
		{"ForecastTaskScoreSet/tiny", NewForecastTaskScoreSetEventBase(1, cTiny, 1).(*EventForecastTaskScoreSet).Score, cMin},
		{"TopicInitialRegretSet/huge", NewTopicInitialRegretSetEventBase(1, 1, cHuge).(*EventTopicInitialRegretSet).Regret, cMax},
		{"TopicInitialEmaScoreSet/negTiny", NewTopicInitialEmaScoreSetEventBase(actor, 1, 1, cNeg).(*EventTopicInitialEmaScoreSet).Score, cNegMin},
		{"RegretStdNormSet/tiny", NewRegretStdNormSetEventBase(1, 1, cTiny).(*EventRegretStdNormSet).Stdnorm, cMin},
		{"PreviousPercentageReward/inRange", NewPreviousPercentageRewardToStakedReputersSetEventBase(1, cMid).(*EventPreviousPercentageRewardToStakedReputersSet).Percentage, cMid},
		{"RewardDelegateStake/huge", NewRewardDelegateStakeEventBase(1, "r", "d", cHuge).(*EventRewardDelegateStake).Amount, cMax},
		{"TopicWeightUpdated/tiny", NewTopicWeightUpdatedEventBase(1, cTiny, one, one).(*EventTopicWeightUpdated).NewWeight, cMin},
		{"DelegateRewardShareUpdated/tiny", NewDelegateRewardShareUpdatedEventBase(1, "r", cTiny).(*EventDelegateRewardShareUpdated).RewardPerShare, cMin},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Truef(t, tc.got.Equal(tc.want), "got %s want %s", tc.got.String(), tc.want.String())
		})
	}
}

// Bundle path: arrays and matrices clamp; NaN passes through; empty stays nil.
//
//nolint:exhaustruct
func TestEventClamp_NetworkInferenceBundle(t *testing.T) {
	nan := alloraMath.NewNaN()
	labeled := func(vs ...alloraMath.Dec) []*LabeledValue {
		out := make([]*LabeledValue, len(vs))
		for i, v := range vs {
			out[i] = &LabeledValue{LabelName: "l", Value: v}
		}
		return out
	}
	in := NetworkInferenceBundle{
		TopicId:       1,
		Nonce:         1,
		CombinedValue: labeled(cTiny, cHuge, cMid, cNeg, nan),
		InfererValues: []*WorkerInference{{Worker: "i1", Values: labeled(cTiny, cHuge)}},
	}
	ev := convertNetworkInferenceBundleToEvent(in, false)

	requireDecsEqual(t, ev.CombinedValue[:4], []alloraMath.Dec{cMin, cMax, cMid, cNegMin})
	require.True(t, ev.CombinedValue[4].IsNaN(), "NaN must pass through")
	requireDecsEqual(t, ev.InfererValues[0], []alloraMath.Dec{cMin, cMax})

	empty := convertNetworkInferenceBundleToEvent(NetworkInferenceBundle{}, false)
	require.Nil(t, empty.CombinedValue)
	require.Nil(t, empty.InfererValues)
}

// ValueBundle path: scalars, slices, and the NaN-filled one-out matrix.
//
//nolint:exhaustruct
func TestEventClamp_ValueBundle(t *testing.T) {
	wav := func(w string, v alloraMath.Dec) *WorkerAttributedValue {
		return &WorkerAttributedValue{Worker: w, Value: v}
	}
	bundle := &ValueBundle{
		CombinedValue: cTiny,
		NaiveValue:    cHuge,
		InfererValues: []*WorkerAttributedValue{wav("i1", cTiny), wav("i2", cMid)},
		OneOutInfererForecasterValues: []*OneOutInfererForecasterValues{
			{
				Forecaster:          "f1",
				OneOutInfererValues: []*WithheldWorkerAttributedValue{{Worker: "i1", Value: cTiny}}, // i2 missing -> NaN
			},
		},
	}
	evb := ValueBundleToEventValueBundleBase(bundle)

	require.True(t, evb.CombinedValue.Equal(cMin))
	require.True(t, evb.NaiveValue.Equal(cMax))
	requireDecsEqual(t, evb.InfererValues, []alloraMath.Dec{cMin, cMid})

	row := evb.OneOutInfererForecasterValues[0]
	require.True(t, row[0].Equal(cMin), "present cell must be clamped")
	require.True(t, row[1].IsNaN(), "missing inferer must stay NaN")
}
