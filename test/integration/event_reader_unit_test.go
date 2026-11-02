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
	event := endBlockEvent(t, emissionstypes.NewTopicRewardForfeitedNoRewardNonceEventBase(7, 100, cosmosMath.NewInt(42)))

	_, err := sdk.ParseTypedEvent(event)
	require.Error(t, err, "the SDK parser rejects the mode attribute it appends itself")

	decoded, err := decodeTypedEvent(event)
	require.NoError(t, err)
	forfeit, ok := decoded.(*emissionstypes.EventTopicRewardForfeitedNoRewardNonce)
	require.True(t, ok)
	require.Equal(t, uint64(7), forfeit.TopicId)
	require.Equal(t, int64(100), forfeit.Epoch)
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

func TestDecodeTypedEventIgnoresOtherModules(t *testing.T) {
	decoded, err := decodeTypedEvent(abci.Event{Type: "transfer", Attributes: nil})
	require.NoError(t, err)
	require.Nil(t, decoded)
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

func TestFirstFingerprintDifference(t *testing.T) {
	require.Empty(t, firstFingerprintDifference([]string{"a"}, []string{"a"}))
	require.Contains(t, firstFingerprintDifference([]string{"a"}, []string{"b"}), "event 0 differs")
	require.Contains(t, firstFingerprintDifference([]string{"a"}, nil), "event counts differ")
}

func TestMissingCoverage(t *testing.T) {
	require.Equal(t, []string{"b"}, missingCoverage(map[string]int{"a": 1}, []string{"a", "b"}))
}
