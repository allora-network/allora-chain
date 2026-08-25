package app

import (
	"context"
	"errors"
	"fmt"

	txsigning "cosmossdk.io/x/tx/signing"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/module"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	gogoproto "github.com/cosmos/gogoproto/proto"
	protov2 "google.golang.org/protobuf/proto"

	mintv2 "github.com/allora-network/allora-chain/x/mint/api/mint/v2"
)

// Compile-time guards for the two exported contracts these types must satisfy.
// The read path also asserts intoAny (AsAny) and protoTxProvider (GetProtoTx) on
// historicalTx structurally; those are covered by TestHistoricalTxImplementsReadInterfaces.
var (
	_ sdk.Tx                = historicalTx{}      //nolint:exhaustruct // interface assertion only
	_ txtypes.ServiceServer = compositeTxServer{} //nolint:exhaustruct // interface assertion only
)

// The consensus tx decoder (app.txConfig) rejects pre-upgrade payloads: after the
// v10 proto bump, unknownproto can no longer resolve historical nested message
// names in the process-global gogo registry. Widening the consensus decoder to
// accept them would be consensus-breaking (a rejected tx and an accepted-then-failed
// tx write different state), so instead the historical-read RPCs get their own
// tolerant decoder over a separate registry. Nothing here touches consensus state.

// historicalTx is the minimal sdk.Tx the tolerant decoder returns for a payload
// the strict decoder rejects. It exposes exactly the three shapes the read-only
// tx service asserts: sdk.Tx, intoAny (AsAny), and protoTxProvider (GetProtoTx).
type historicalTx struct {
	tx *txtypes.Tx
}

// GetMsgs returns the already-unpacked messages; the tolerant decoder unpacks the
// Anys during Unmarshal, so the cached values are present.
func (h historicalTx) GetMsgs() []sdk.Msg { return h.tx.GetMsgs() }

// GetMsgsV2 is never called on the read path; fail loudly rather than return a
// half-built value if that ever changes.
func (h historicalTx) GetMsgsV2() ([]protov2.Message, error) {
	return nil, fmt.Errorf("historicalTx: GetMsgsV2 is not supported on the read-only query path")
}

// GetProtoTx satisfies protoTxProvider, used by GetBlockWithTxs.
func (h historicalTx) GetProtoTx() *txtypes.Tx { return h.tx }

// AsAny satisfies intoAny, used by GetTx/GetTxsEvent. UnsafePackAny caches the
// concrete *txtypes.Tx that mkTxResult type-asserts, without an error return
// AsAny cannot surface.
func (h historicalTx) AsAny() *codectypes.Any { return codectypes.UnsafePackAny(h.tx) }

// newHistoricalTxDecoder decodes committed txs for the query path. It tries the
// strict SDK decoder first, so every tx that decodes today keeps the identical
// code path and concrete type; only when that fails (a historical payload) does
// it fall back to a plain unmarshal that skips unknownproto's nested-field walk.
func newHistoricalTxDecoder(cdc codec.Codec) sdk.TxDecoder {
	strict := authtx.DefaultTxDecoder(cdc)
	return func(txBytes []byte) (sdk.Tx, error) {
		// Current-format txs: unchanged behavior and unchanged return type.
		tx, strictErr := strict(txBytes)
		if strictErr == nil {
			return tx, nil
		}

		// Both decoders failing means the payload is malformed rather than
		// historical, and the strict error is the one that carries the reason.
		fail := func(err error) (sdk.Tx, error) {
			return nil, sdkerrors.ErrTxDecode.Wrap(errors.Join(err, strictErr).Error())
		}

		// Historical payloads: unmarshal the envelope directly. Any resolution
		// still goes through cdc's registry, so an unknown type URL still errors
		// here; only the strict unknown-field rejection is skipped.
		var raw txtypes.TxRaw
		if err := cdc.Unmarshal(txBytes, &raw); err != nil {
			return fail(err)
		}
		var body txtypes.TxBody
		if err := cdc.Unmarshal(raw.BodyBytes, &body); err != nil {
			return fail(err)
		}
		// An Any with an empty type URL unmarshals without error but caches no
		// value, which makes GetMsgs panic; unknown non-empty URLs already fail above.
		for _, msg := range body.Messages {
			if msg == nil || msg.GetCachedValue() == nil {
				return fail(errors.New("message Any has no resolvable type URL"))
			}
		}
		var authInfo txtypes.AuthInfo
		if err := cdc.Unmarshal(raw.AuthInfoBytes, &authInfo); err != nil {
			return fail(err)
		}
		return historicalTx{tx: &txtypes.Tx{
			Body:       &body,
			AuthInfo:   &authInfo,
			Signatures: raw.Signatures,
		}}, nil
	}
}

// buildQueryDecodingPath constructs the read-only registry and tx config and
// stores them on the app. Called once at construction, after every module is on
// ModuleManager. It never mutates app.interfaceRegistry or the gogo registry.
func (app *AlloraApp) buildQueryDecodingPath() error {
	reg, err := app.buildQueryInterfaceRegistry()
	if err != nil {
		return err
	}
	app.queryInterfaceRegistry = reg

	// The tolerant decoder and the tx config share one codec so Any resolution is
	// consistent across decode and (gRPC) response marshaling.
	queryCodec := codec.NewProtoCodec(reg)
	//nolint:exhaustruct // only the decoder and sign modes are overridden; the rest default
	txConfig, err := authtx.NewTxConfigWithOptions(queryCodec, authtx.ConfigOptions{
		EnabledSignModes: authtx.DefaultSignModes,
		ProtoDecoder:     newHistoricalTxDecoder(queryCodec),
	})
	if err != nil {
		return err
	}
	app.queryTxConfig = txConfig
	return nil
}

// buildQueryInterfaceRegistry mirrors the app's registry (same address codecs,
// same modules) into a fresh registry, then adds the historical tx types that are
// not registered for consensus. Kept separate so these additions can never widen
// what the consensus decoder accepts.
func (app *AlloraApp) buildQueryInterfaceRegistry() (codectypes.InterfaceRegistry, error) {
	// Reuse the app's exact address codecs so signer resolution matches.
	sc := app.interfaceRegistry.SigningContext()
	reg, err := codectypes.NewInterfaceRegistryWithOptions(codectypes.InterfaceRegistryOptions{
		ProtoFiles: gogoproto.HybridResolver,
		//nolint:exhaustruct // the remaining signing options default exactly as the app's registry
		SigningOptions: txsigning.Options{
			AddressCodec:          sc.AddressCodec(),
			ValidatorAddressCodec: sc.ValidatorAddressCodec(),
		},
	})
	if err != nil {
		return nil, err
	}

	// std registers sdk.Msg/sdk.Tx/cryptotypes.PubKey as interfaces; without it
	// RegisterImplementations below has nothing to attach to.
	std.RegisterInterfaces(reg)

	// Re-run every module's RegisterInterfaces against the new registry. This
	// covers auth/bank/gov/emissions v2-v9/mint v1beta1 and the IBC/feemarket
	// modules added imperatively, exactly as SetupAppBuilder does for consensus.
	module.NewBasicManagerFromManager(app.ModuleManager, CustomModuleBasics()).RegisterInterfaces(reg)

	// The historical extras. mint.v2 is the only tx-bearing version not registered
	// anywhere for consensus (the mint tx lineage is v1beta1 -> v2 -> v5; v3 is
	// events-only and v4 is query/genesis-only, so neither declares an sdk.Msg).
	// This is a query-registry-only addition and does not touch consensus; verified
	// by TestMintV2QueryRegistryDoesNotAffectConsensus.
	mintv2.RegisterInterfaces(reg)

	if err := reg.SigningContext().Validate(); err != nil {
		return nil, err
	}
	return reg, nil
}

// compositeTxServer serves the tx service with two backends: the strict, embedded
// server handles every method by default (and any method a future SDK adds), while
// only the historical-read methods are delegated to the tolerant server.
type compositeTxServer struct {
	txtypes.ServiceServer                       // strict; default for all methods
	tolerant              txtypes.ServiceServer // tolerant; historical reads only
}

// GetTx reads a committed tx by hash; route to the tolerant server so pre-upgrade
// payloads decode.
func (s compositeTxServer) GetTx(ctx context.Context, req *txtypes.GetTxRequest) (*txtypes.GetTxResponse, error) {
	return s.tolerant.GetTx(ctx, req)
}

// GetTxsEvent searches committed txs; route to the tolerant server.
func (s compositeTxServer) GetTxsEvent(ctx context.Context, req *txtypes.GetTxsEventRequest) (*txtypes.GetTxsEventResponse, error) {
	return s.tolerant.GetTxsEvent(ctx, req)
}

// GetBlockWithTxs decodes every tx in a committed block; route to the tolerant server.
func (s compositeTxServer) GetBlockWithTxs(ctx context.Context, req *txtypes.GetBlockWithTxsRequest) (*txtypes.GetBlockWithTxsResponse, error) {
	return s.tolerant.GetBlockWithTxs(ctx, req)
}

// RegisterTxService overrides the promoted runtime.App method. It registers a
// composite tx service: strict for user-supplied and non-decoding methods
// (including TxDecode, which asserts the SDK's own concrete type), tolerant only
// for historical reads. It must NOT also call the embedded method, or the gRPC
// router panics on duplicate service registration.
func (app *AlloraApp) RegisterTxService(clientCtx client.Context) {
	// Strict server: the app's own client context, unchanged consensus behavior.
	strict := authtx.NewTxServer(clientCtx, app.Simulate, app.interfaceRegistry)

	// Tolerant server: same context but with the read-only tx config swapped in.
	tolerantCtx := clientCtx.WithTxConfig(app.queryTxConfig)
	tolerant := authtx.NewTxServer(tolerantCtx, app.Simulate, app.queryInterfaceRegistry)

	txtypes.RegisterServiceServer(
		app.GRPCQueryRouter(),
		compositeTxServer{ServiceServer: strict, tolerant: tolerant},
	)
}
