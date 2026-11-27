package integration_test

import (
	"testing"

	cosmosMath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	alloraMath "github.com/allora-network/allora-chain/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// endBlockEvent renders a typed event the way /block_results returns it.
func endBlockEvent(t *testing.T, base interface {
	Reset()
	String() string
	ProtoMessage()
}) abci.Event {
	t.Helper()
	event, err := sdk.TypedEventToEvent(base)
	require.NoError(t, err)
	rendered := abci.Event(event)
	rendered.Attributes = append(rendered.Attributes, abci.EventAttribute{Key: modeAttribute, Value: "EndBlock", Index: false})
	return rendered
}

func TestDecodeTypedEventHandlesEndBlockMode(t *testing.T) {
	event := endBlockEvent(t, emissionstypes.NewTopicRewardForfeitedEventBase(7, 100, emissionstypes.TopicRewardForfeitReason_TOPIC_REWARD_FORFEIT_REASON_NO_REWARD_NONCE, 0, cosmosMath.NewInt(42)))

	_, err := sdk.ParseTypedEvent(event)
	require.Error(t, err, "the SDK parser rejects the mode attribute it appends itself")

	decoded, err := decodeTypedEvent(event)
	require.NoError(t, err)
	forfeit, ok := decoded.(*emissionstypes.EventTopicRewardForfeited)
	require.True(t, ok)
	require.Equal(t, uint64(7), forfeit.TopicId)
	require.Equal(t, int64(100), forfeit.Epoch)
	require.Equal(t, emissionstypes.TopicRewardForfeitReason_TOPIC_REWARD_FORFEIT_REASON_NO_REWARD_NONCE, forfeit.Reason)
	require.True(t, forfeit.Amount.Equal(cosmosMath.NewInt(42)))
}

func TestDecodeTypedEventKeepsZeroValueEnum(t *testing.T) {
	event := endBlockEvent(t, emissionstypes.NewNonceExpiredEventBase(
		3, 90, emissionstypes.ActorType_ACTOR_TYPE_INFERER_UNSPECIFIED, emissionstypes.NonceExpiredReason_NONCE_EXPIRED_REASON_CLOSE_FAILED,
	))
	decoded, err := decodeTypedEvent(event)
	require.NoError(t, err)
	expired, ok := decoded.(*emissionstypes.EventNonceExpired)
	require.True(t, ok)
	require.Equal(t, emissionstypes.ActorType_ACTOR_TYPE_INFERER_UNSPECIFIED, expired.ActorType)
	require.Equal(t, emissionstypes.NonceExpiredReason_NONCE_EXPIRED_REASON_CLOSE_FAILED, expired.Reason)
}

func TestIsTypedEventTypeCoversOnlyChainModules(t *testing.T) {
	require.True(t, isTypedEventType("emissions.v10.EventAddStake"))
	require.True(t, isTypedEventType("mint.v3.EventEmissionInfo"))
	require.False(t, isTypedEventType("transfer"))
}

func TestDecodeTypedEventReportsUnknownEmissionsType(t *testing.T) {
	_, err := decodeTypedEvent(abci.Event{Type: "emissions.v10.EventThatDoesNotExist", Attributes: nil})
	require.Error(t, err)
}

func TestSplitBlockEventsUsesModeAttribute(t *testing.T) {
	begin := endBlockEvent(t, emissionstypes.NewTopicStatusChangedEventBase(
		1, true, emissionstypes.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_REACTIVATED,
		alloraMath.OneDec(), alloraMath.OneDec(), alloraMath.OneDec(),
	))
	begin.Attributes[len(begin.Attributes)-1].Value = "BeginBlock"
	end := endBlockEvent(t, emissionstypes.NewNonceExpiredEventBase(
		1, 2, emissionstypes.ActorType_ACTOR_TYPE_REPUTER, emissionstypes.NonceExpiredReason_NONCE_EXPIRED_REASON_CLOSE_FAILED,
	))

	gotBegin, gotEnd := splitBlockEvents([]abci.Event{begin, end})
	require.Equal(t, []abci.Event{begin}, gotBegin)
	require.Equal(t, []abci.Event{end}, gotEnd)
}

func TestEventFingerprintIsUnambiguous(t *testing.T) {
	joined := abci.Event{Type: "t", Attributes: []abci.EventAttribute{{Key: "a", Value: "b|c=d", Index: false}}}
	split := abci.Event{Type: "t", Attributes: []abci.EventAttribute{{Key: "a", Value: "b", Index: false}, {Key: "c", Value: "d", Index: false}}}
	require.NotEqual(t, eventFingerprint([]abci.Event{joined}), eventFingerprint([]abci.Event{split}))
}

func TestFirstFingerprintDifference(t *testing.T) {
	require.Empty(t, firstFingerprintDifference([]string{"a"}, []string{"a"}))
	require.Contains(t, firstFingerprintDifference([]string{"a"}, []string{"b"}), "event 0 differs")
	require.Contains(t, firstFingerprintDifference([]string{"a"}, nil), "event counts differ")
}

func TestRequireDecEqualSeparatesNaNFromZero(t *testing.T) {
	audit := &eventAudit{} //nolint:exhaustruct_v5 // the comparison only records violations
	audit.requireDecEqual(1, "nan vs zero", alloraMath.NewNaN(), alloraMath.ZeroDec())
	audit.requireDecEqual(1, "zero vs nan", alloraMath.ZeroDec(), alloraMath.NewNaN())
	require.Len(t, audit.violations, 2)

	audit.violations = nil
	audit.requireDecEqual(1, "nan vs nan", alloraMath.NewNaN(), alloraMath.NewNaN())
	require.Empty(t, audit.violations)
}

func TestTakeTransferConsumesOneMovement(t *testing.T) {
	one := cosmosMath.NewInt(1)
	transfers := []bankTransfer{{sender: "a", recipient: "b", amount: one}}

	remaining, moved := takeTransfer(transfers, "a", "b", cosmosMath.NewInt(2))
	require.False(t, moved)
	require.Len(t, remaining, 1)

	remaining, moved = takeTransfer(remaining, "a", "b", one)
	require.True(t, moved)
	require.Empty(t, remaining)

	_, moved = takeTransfer(remaining, "a", "b", one)
	require.False(t, moved)
}

func TestMissingCoverage(t *testing.T) {
	require.Equal(t, []string{"b"}, missingCoverage(map[string]int{"a": 1}, []string{"a", "b"}))
}

func TestMissingWeightEvents(t *testing.T) {
	bundleOf := func(inferers, forecasters int) *emissionstypes.EventNetworkInferenceBundle {
		bundle := new(emissionstypes.EventNetworkInferenceBundle)
		bundle.InfererValues = make(alloraMath.DecMatrix, inferers)
		bundle.ForecasterValues = make(alloraMath.DecMatrix, forecasters)
		return bundle
	}
	infererOnly := networkWeights{infererAddresses: []string{"a", "b"}, infererWeights: []alloraMath.Dec{alloraMath.OneDec(), alloraMath.OneDec()}, forecasterAddresses: nil, forecasterWeights: nil}
	both := networkWeights{infererAddresses: infererOnly.infererAddresses, infererWeights: infererOnly.infererWeights, forecasterAddresses: []string{"f"}, forecasterWeights: []alloraMath.Dec{alloraMath.OneDec()}}
	none := networkWeights{infererAddresses: nil, infererWeights: nil, forecasterAddresses: nil, forecasterWeights: nil}
	infererEvent, forecasterEvent := "EventNetworkInferenceInfererWeightsSet", "EventNetworkInferenceForecasterWeightsSet"

	for name, tc := range map[string]struct {
		bundle   *emissionstypes.EventNetworkInferenceBundle
		weights  networkWeights
		allNew   bool
		wantMiss []string
	}{
		"single inferer needs no weights":                {bundleOf(1, 0), none, false, []string{}},
		"two inferers without inferer weights":           {bundleOf(2, 0), none, false, []string{infererEvent}},
		"two inferers with inferer weights":              {bundleOf(2, 0), infererOnly, false, []string{}},
		"forecast rows without forecaster weights":       {bundleOf(2, 1), infererOnly, false, []string{forecasterEvent}},
		"forecast rows with both weights":                {bundleOf(2, 1), both, false, []string{}},
		"all-new epoch needs no forecaster weights":      {bundleOf(2, 1), infererOnly, true, []string{}},
		"nothing emitted for forecasts and two inferers": {bundleOf(2, 1), none, false, []string{infererEvent, forecasterEvent}},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.wantMiss, missingWeightEvents(tc.bundle, tc.weights, tc.allNew))
		})
	}
}
