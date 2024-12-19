package app

import (
	"context"
	_ "embed"
	"encoding/hex"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"

	"cosmossdk.io/core/appconfig"
	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	dbm "github.com/cosmos/cosmos-db"

	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/app/keepers"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
	metrics "github.com/hashicorp/go-metrics"

	_ "cosmossdk.io/api/cosmos/tx/config/v1"                      // import for side-effects
	_ "cosmossdk.io/x/circuit"                                    // import for side-effects
	_ "cosmossdk.io/x/evidence"                                   // import for side-effects
	_ "cosmossdk.io/x/feegrant/module"                            // import for side-effects
	_ "cosmossdk.io/x/upgrade"                                    // import for side-effects
	_ "github.com/allora-network/allora-chain/x/emissions/module" // import for side-effects
	_ "github.com/allora-network/allora-chain/x/mint/module"      // import for side-effects
	_ "github.com/cosmos/cosmos-sdk/x/auth"                       // import for side-effects
	_ "github.com/cosmos/cosmos-sdk/x/auth/tx/config"             // import for side-effects
	_ "github.com/cosmos/cosmos-sdk/x/authz/module"               // import for side-effects
	_ "github.com/cosmos/cosmos-sdk/x/bank"                       // import for side-effects
	_ "github.com/cosmos/cosmos-sdk/x/consensus"                  // import for side-effects
	_ "github.com/cosmos/cosmos-sdk/x/distribution"               // import for side-effects
	_ "github.com/cosmos/cosmos-sdk/x/params"                     // import for side-effects
	_ "github.com/cosmos/cosmos-sdk/x/slashing"                   // import for side-effects
	_ "github.com/cosmos/cosmos-sdk/x/staking"                    // import for side-effects

	"github.com/allora-network/allora-chain/health"
)

// DefaultNodeHome default home directories for the application daemon
var DefaultNodeHome string

//go:embed app.yaml
var AppConfigYAML []byte

var (
	_ runtime.AppI            = (*AlloraApp)(nil)
	_ servertypes.Application = (*AlloraApp)(nil)
)

// AlloraApp extends an ABCI application, but with most of its parameters exported.
// They are exported for convenience in creating helper functions, as object
// capabilities aren't needed for testing.
type AlloraApp struct {
	*runtime.App
	keepers.AppKeepers

	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	txConfig          client.TxConfig
	interfaceRegistry codectypes.InterfaceRegistry

	// simulation manager
	sm *module.SimulationManager

	// Nurse
	nurse *health.Nurse
}

func init() {
	sdk.DefaultPowerReduction = math.NewIntFromBigInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	DefaultNodeHome = filepath.Join(userHomeDir, ".allorad")
}

// AppConfig returns the default app config.
func AppConfig() depinject.Config {
	return depinject.Configs(
		appconfig.LoadYAML(AppConfigYAML),
		depinject.Supply(
			// supply custom module basics
			map[string]module.AppModuleBasic{
				genutiltypes.ModuleName: genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
				govtypes.ModuleName: gov.NewAppModuleBasic(
					[]govclient.ProposalHandler{
						paramsclient.ProposalHandler,
					},
				),
			},
		),
	)
}

// NewAlloraApp returns a reference to an initialized AlloraApp.
func NewAlloraApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) (*AlloraApp, error) {
	var (
		app        = &AlloraApp{} //nolint:exhaustruct
		appBuilder *runtime.AppBuilder
	)

	// Initialize nurse if provided with a config TOML path
	nurseCfgPath := os.Getenv("NURSE_TOML_PATH")
	if nurseCfgPath != "" {
		nurseCfg := health.MustReadConfigTOML(nurseCfgPath)
		nurseCfg.Logger = logger
		app.nurse = health.NewNurse(nurseCfg)

		err := app.nurse.Start()
		if err != nil {
			return nil, err
		}
	}

	if err := depinject.Inject(
		depinject.Configs(
			AppConfig(),
			depinject.Supply(
				logger,
				appOpts,
			),
		),

		&appBuilder,
		&app.appCodec,
		&app.legacyAmino,
		&app.txConfig,
		&app.interfaceRegistry,
		&app.AccountKeeper,
		&app.BankKeeper,
		&app.FeeGrantKeeper,
		&app.StakingKeeper,
		&app.SlashingKeeper,
		&app.DistrKeeper,
		&app.ConsensusParamsKeeper,
		&app.MintKeeper,
		&app.GovKeeper,
		&app.EmissionsKeeper,
		&app.UpgradeKeeper,
		&app.ParamsKeeper,
		&app.AuthzKeeper,
		&app.CircuitBreakerKeeper,
		&app.EvidenceKeeper,
	); err != nil {
		return nil, err
	}

	baseAppOptions = append(baseAppOptions, baseapp.SetOptimisticExecution())
	app.App = appBuilder.Build(db, traceStore, baseAppOptions...)

	// Register legacy modules
	app.registerLegacyModules()

	// Register feemarket module
	app.registerFeeMarketModule()

	// register streaming services
	if err := app.RegisterStreamingServices(appOpts, app.kvStoreKeys()); err != nil {
		return nil, err
	}

	// create the simulation manager and define the order of the modules for deterministic simulations
	// NOTE: this is not required apps that don't use the simulator for fuzz testing transactions
	app.sm = module.NewSimulationManagerFromAppModules(app.ModuleManager.Modules, make(map[string]module.AppModuleSimulation, 0))
	app.sm.RegisterStoreDecoders()

	app.setupUpgradeHandlers(&app.AppKeepers)
	app.setupUpgradeStoreLoaders()

	app.SetInitChainer(func(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
		err := app.UpgradeKeeper.SetModuleVersionMap(ctx, app.ModuleManager.GetVersionMap())
		if err != nil {
			return nil, errors.Wrap(err, "failed to set module version map")
		}
		return app.App.InitChainer(ctx, req)
	})

	// Create a global ante handler that will be called on each transaction when
	// proposals are being built and verified.
	anteHandlerOptions := ante.HandlerOptions{
		AccountKeeper:          app.AccountKeeper,
		BankKeeper:             app.BankKeeper,
		FeegrantKeeper:         app.FeeGrantKeeper,
		SigGasConsumer:         ante.DefaultSigVerificationGasConsumer,
		SignModeHandler:        app.txConfig.SignModeHandler(),
		TxFeeChecker:           nil,
		ExtensionOptionChecker: nil,
	}

	anteOptions := AnteHandlerOptions{
		BaseOptions:     anteHandlerOptions,
		AccountKeeper:   app.AccountKeeper,
		BankKeeper:      app.BankKeeper,
		FeeMarketKeeper: app.FeeMarketKeeper,
		CircuitKeeper:   &app.CircuitBreakerKeeper,
	}
	anteHandler, err := NewAnteHandler(anteOptions)
	if err != nil {
		panic(err)
	}

	postHandlerOptions := PostHandlerOptions{
		AccountKeeper:   app.AccountKeeper,
		BankKeeper:      app.BankKeeper,
		FeeMarketKeeper: app.FeeMarketKeeper,
	}
	postHandler, err := NewPostHandler(postHandlerOptions)
	if err != nil {
		panic(err)
	}

	// set ante and post handlers
	app.SetAnteHandler(anteHandler)
	app.SetPostHandler(postHandler)

	if err := app.Load(loadLatest); err != nil {
		return nil, err
	}

	return app, nil
}

// LegacyAmino returns AlloraApp's amino codec.
func (app *AlloraApp) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

// GetKey returns the KVStoreKey for the provided store key.
func (app *AlloraApp) GetKey(storeKey string) *storetypes.KVStoreKey {
	sk := app.UnsafeFindStoreKey(storeKey)
	kvStoreKey, ok := sk.(*storetypes.KVStoreKey)
	if !ok {
		return nil
	}
	return kvStoreKey
}

// GetMemKey returns the MemoryStoreKey for the provided store key.
func (app *AlloraApp) GetMemKey(storeKey string) *storetypes.MemoryStoreKey {
	key, ok := app.UnsafeFindStoreKey(storeKey).(*storetypes.MemoryStoreKey)
	if !ok {
		return nil
	}

	return key
}

func (app *AlloraApp) kvStoreKeys() map[string]*storetypes.KVStoreKey {
	keys := make(map[string]*storetypes.KVStoreKey)
	for _, k := range app.GetStoreKeys() {
		if kv, ok := k.(*storetypes.KVStoreKey); ok {
			keys[kv.Name()] = kv
		}
	}

	return keys
}

// GetSubspace returns a param subspace for a given module name.
func (app *AlloraApp) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := app.ParamsKeeper.GetSubspace(moduleName)
	return subspace
}

// GetIBCKeeper returns the IBC keeper.
func (app *AlloraApp) GetIBCKeeper() *ibckeeper.Keeper {
	return app.IBCKeeper
}

// GetCapabilityScopedKeeper returns the capability scoped keeper.
func (app *AlloraApp) GetCapabilityScopedKeeper(moduleName string) capabilitykeeper.ScopedKeeper {
	return app.CapabilityKeeper.ScopeToModule(moduleName)
}

// SimulationManager implements the SimulationApp interface
func (app *AlloraApp) SimulationManager() *module.SimulationManager {
	return app.sm
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *AlloraApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	app.App.RegisterAPIRoutes(apiSvr, apiConfig)
	// register swagger API in app.go so that other applications can override easily
	if err := server.RegisterSwaggerAPI(apiSvr.ClientCtx, apiSvr.Router, apiConfig.Swagger); err != nil {
		panic(err)
	}
}

func (app *AlloraApp) PrepareProposal(req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
	app.Logger().Debug("CONSENSUS EVENT", "event", "PrepareProposal", "module", "allora_abci_metrics")
	startTime := time.Now()
	defer func() {
		app.Logger().Debug("CONSENSUS EVENT", "event", "End", "module", "allora_abci_metrics")
		logMisbehaviors(req.Misbehavior, "prepare", "proposal")
		metrics.MeasureSince([]string{"allora", "prepare", "proposal", "ms"}, startTime.UTC())
	}()

	res, err := app.App.PrepareProposal(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (app *AlloraApp) ProcessProposal(req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
	app.Logger().Debug("CONSENSUS EVENT", "event", "ProcessProposal", "module", "allora_abci_metrics")
	startTime := time.Now()
	defer func() {
		app.Logger().Debug("CONSENSUS EVENT", "event", "End", "module", "allora_abci_metrics")
		logMisbehaviors(req.Misbehavior, "process", "proposal")
		metrics.MeasureSince([]string{"allora", "process", "proposal", "ms"}, startTime.UTC())
	}()

	res, err := app.App.ProcessProposal(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (app *AlloraApp) ExtendVote(ctx context.Context, req *abci.RequestExtendVote) (*abci.ResponseExtendVote, error) {
	app.Logger().Debug("CONSENSUS EVENT", "event", "ExtendVote", "module", "allora_abci_metrics")
	startTime := time.Now()
	defer func() {
		app.Logger().Debug("CONSENSUS EVENT", "event", "End", "module", "allora_abci_metrics")
		logMisbehaviors(req.Misbehavior, "extend", "vote")
		metrics.MeasureSince([]string{"allora", "extend", "vote", "ms"}, startTime.UTC())
	}()

	res, err := app.App.ExtendVote(ctx, req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (app *AlloraApp) VerifyVoteExtension(req *abci.RequestVerifyVoteExtension) (resp *abci.ResponseVerifyVoteExtension, err error) {
	app.Logger().Debug("CONSENSUS EVENT", "event", "VerifyVoteExtension", "module", "allora_abci_metrics")
	startTime := time.Now()
	defer func() {
		app.Logger().Debug("CONSENSUS EVENT", "event", "End", "module", "allora_abci_metrics")
		metrics.MeasureSince([]string{"allora", "verify", "vote", "extension", "ms"}, startTime.UTC())
	}()

	res, err := app.App.VerifyVoteExtension(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (app *AlloraApp) FinalizeBlock(req *abci.RequestFinalizeBlock) (*abci.ResponseFinalizeBlock, error) {
	app.Logger().Debug("CONSENSUS EVENT", "event", "FinalizeBlock", "module", "allora_abci_metrics")
	startTime := time.Now()
	defer func() {
		app.Logger().Debug("CONSENSUS EVENT", "event", "End", "module", "allora_abci_metrics")
		logMisbehaviors(req.Misbehavior, "finalize", "block")
		metrics.SetGauge([]string{"allora", "finalize", "block", "height"}, float32(req.Height))
		metrics.MeasureSince([]string{"allora", "finalize", "block", "ms"}, startTime.UTC())
	}()

	res, err := app.App.FinalizeBlock(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (app *AlloraApp) Commit() (*abci.ResponseCommit, error) {
	startTime := time.Now()
	app.Logger().Debug("CONSENSUS EVENT", "event", "Commit", "module", "allora_abci_metrics")
	defer func() {
		app.Logger().Debug("CONSENSUS EVENT", "event", "End", "duration", time.Since(startTime).Milliseconds(), "module", "allora_abci_metrics")
		metrics.MeasureSince([]string{"allora", "commit", "ms"}, startTime.UTC())
	}()

	res, err := app.App.Commit()
	if err != nil {
		return nil, err
	}

	return res, nil
}

func logMisbehaviors(mbs []abci.Misbehavior, keys ...string) {
	for _, misbehavior := range mbs {
		var typ string
		switch misbehavior.GetType() {
		case abci.MisbehaviorType_UNKNOWN:
			typ = "unknown"
		case abci.MisbehaviorType_DUPLICATE_VOTE:
			typ = "duplicate_vote"
		case abci.MisbehaviorType_LIGHT_CLIENT_ATTACK:
			typ = "light_client_attack"
		}
		metrics.IncrCounterWithLabels(
			append(append([]string{"allora"}, keys...), "misbehavior"),
			float32(1),
			[]metrics.Label{
				telemetry.NewLabel("validator", sdk.ValAddress(misbehavior.Validator.Address).String()),
				telemetry.NewLabel("validator_hex", hex.EncodeToString(misbehavior.Validator.Address)),
				telemetry.NewLabel("validator_power", strconv.FormatInt(misbehavior.Validator.Power, 10)),
				telemetry.NewLabel("misbehavior_type", typ),
			},
		)
	}
}
