package app

import (
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	gogojsonpb "github.com/cosmos/gogoproto/jsonpb"
	"github.com/stretchr/testify/require"

	emissionsv7 "github.com/allora-network/allora-chain/x/emissions/api/emissions/v7"
	emissionsv8 "github.com/allora-network/allora-chain/x/emissions/api/emissions/v8"
	mintv2 "github.com/allora-network/allora-chain/x/mint/api/mint/v2"
)

// rendersOverGateway reports whether a tx carrying msg would render as JSON over
// the REST gateway. The gateway marshals query responses with a jsonpb marshaler
// whose AnyResolver is the app interface registry, not the query registry
// (server/api New: gateway.JSONPb{AnyResolver: clientCtx.InterfaceRegistry}), and
// it resolves each message Any through that registry rather than any cached value.
// The Any is rebuilt from TypeUrl+Value to mirror what the gateway sees after the
// gRPC round-trip.
func rendersOverGateway(t *testing.T, a *AlloraApp, msg sdk.Msg) bool {
	t.Helper()
	packed, err := codectypes.NewAnyWithValue(msg)
	require.NoError(t, err)
	//nolint:exhaustruct // wire form: only TypeUrl+Value survive the gRPC boundary
	wire := &codectypes.Any{TypeUrl: packed.TypeUrl, Value: packed.Value}
	//nolint:exhaustruct // minimal response tx; only the message Any drives resolution
	tx := &txtypes.Tx{Body: &txtypes.TxBody{Messages: []*codectypes.Any{wire}}}
	//nolint:exhaustruct // only the marshaling options the gateway sets are relevant
	m := &gogojsonpb.Marshaler{OrigName: true, EmitDefaults: true, AnyResolver: a.interfaceRegistry}
	_, err = m.MarshalToString(tx)
	return err == nil
}

// The REST gateway resolves response messages through the app registry, so its
// visible surface is exactly the consensus-registered types. A historical payload
// registered for consensus (v9) renders; every query-only addition (mint.v2 and the
// v7/v8 whitelist txs) does not. The negative cases are the regression guard: if one
// starts rendering, it was added to the app registry, which widens what the
// consensus decoder accepts. The positive case guards that the read fix still
// reaches the REST surface that the original bug was reported on.
func TestGatewayRendersConsensusTypesOnly(t *testing.T) {
	a := sharedApp(t)

	require.True(t, rendersOverGateway(t, a, historicalV9Worker()),
		"a historical v9 tx must render over the REST gateway")

	//nolint:exhaustruct // only Sender is needed to resolve the type URL
	require.False(t, rendersOverGateway(t, a, &mintv2.UpdateParamsRequest{Sender: "allo1sender"}),
		"query-only mint.v2 must not render over the REST gateway")
	//nolint:exhaustruct // only the signer fields are needed
	require.False(t, rendersOverGateway(t, a, &emissionsv7.AddToGlobalWorkerWhitelistRequest{Sender: "allo1sender", Address: "allo1addr"}),
		"query-only v7 whitelist tx must not render over the REST gateway")
	//nolint:exhaustruct // only the signer fields are needed
	require.False(t, rendersOverGateway(t, a, &emissionsv8.BulkRemoveFromTopicReputerWhitelistRequest{Sender: "allo1sender"}),
		"query-only v8 whitelist tx must not render over the REST gateway")
}
