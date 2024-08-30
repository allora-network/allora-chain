package app

import (
	_ "embed"
	"io"
	"math/big"
	"os"
	"path/filepath"

	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	dbm "github.com/cosmos/cosmos-db"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"

	"cosmossdk.io/core/appconfig"
	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	"cosmossdk.io/math"

	storetypes "cosmossdk.io/store/types"
	circuitkeeper "cosmossdk.io/x/circuit/keeper"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	emissionsKeeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
	mintkeeper "github.com/allora-network/allora-chain/x/mint/keeper"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/codec/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	consensuskeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	icacontrollerkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/keeper"
	icahostkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/keeper"
	ibcfeekeeper "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/keeper"
	ibcfeetypes "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/types"
	ibctransferkeeper "github.com/cosmos/ibc-go/v8/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
	ibctestingtypes "github.com/cosmos/ibc-go/v8/testing/types"

	"cosmossdk.io/api/cosmos/crypto/ed25519"
	_ "cosmossdk.io/api/cosmos/tx/config/v1" // import for side-effects
	_ "cosmossdk.io/x/circuit"               // import for side-effects
	_ "cosmossdk.io/x/upgrade"
	_ "github.com/allora-network/allora-chain/x/emissions/module"
	_ "github.com/allora-network/allora-chain/x/mint/module" // import for side-effects
	_ "github.com/cosmos/cosmos-sdk/x/auth"                  // import for side-effects
	_ "github.com/cosmos/cosmos-sdk/x/auth/tx/config"        // import for side-effects
	_ "github.com/cosmos/cosmos-sdk/x/authz/module"          // import for side-effects
	_ "github.com/cosmos/cosmos-sdk/x/bank"                  // import for side-effects
	_ "github.com/cosmos/cosmos-sdk/x/consensus"             // import for side-effects
	_ "github.com/cosmos/cosmos-sdk/x/distribution"          // import for side-effects
	_ "github.com/cosmos/cosmos-sdk/x/params"                // import for side-effects
	_ "github.com/cosmos/cosmos-sdk/x/slashing"              // import for side-effects
	_ "github.com/cosmos/cosmos-sdk/x/staking"               // import for side-effects
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
	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	txConfig          client.TxConfig
	interfaceRegistry codectypes.InterfaceRegistry

	// keepers
	AccountKeeper         authkeeper.AccountKeeper
	AuthzKeeper           authzkeeper.Keeper
	CircuitBreakerKeeper  circuitkeeper.Keeper
	BankKeeper            bankkeeper.Keeper
	StakingKeeper         *stakingkeeper.Keeper
	DistrKeeper           distrkeeper.Keeper
	ConsensusParamsKeeper consensuskeeper.Keeper
	MintKeeper            mintkeeper.Keeper
	GovKeeper             *govkeeper.Keeper
	EmissionsKeeper       emissionsKeeper.Keeper
	ParamsKeeper          paramskeeper.Keeper
	UpgradeKeeper         *upgradekeeper.Keeper
	SlashingKeeper        slashingkeeper.Keeper

	// IBC
	IBCKeeper           *ibckeeper.Keeper // IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
	CapabilityKeeper    *capabilitykeeper.Keeper
	IBCFeeKeeper        ibcfeekeeper.Keeper
	ICAControllerKeeper icacontrollerkeeper.Keeper
	ICAHostKeeper       icahostkeeper.Keeper
	TransferKeeper      ibctransferkeeper.Keeper

	// Scoped IBC
	ScopedIBCKeeper           capabilitykeeper.ScopedKeeper
	ScopedIBCTransferKeeper   capabilitykeeper.ScopedKeeper
	ScopedICAControllerKeeper capabilitykeeper.ScopedKeeper
	ScopedICAHostKeeper       capabilitykeeper.ScopedKeeper

	// simulation manager
	sm *module.SimulationManager
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
		app        = &AlloraApp{}
		appBuilder *runtime.AppBuilder
	)

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
	); err != nil {
		return nil, err
	}

	app.App = appBuilder.Build(db, traceStore, baseAppOptions...)

	// Register legacy modules
	app.registerIBCModules()

	// register streaming services
	if err := app.RegisterStreamingServices(appOpts, app.kvStoreKeys()); err != nil {
		return nil, err
	}

	/****  Module Options ****/
	app.ModuleManager.SetOrderPreBlockers(
		upgradetypes.ModuleName,
	)

	//begin_blockers: [capability, distribution, staking, mint, ibc, transfer, genutil, interchainaccounts, feeibc]
	//end_blockers: [staking, ibc, transfer, capability, genutil, interchainaccounts, feeibc, emissions]
	app.ModuleManager.SetOrderBeginBlockers(
		capabilitytypes.ModuleName,
		distrtypes.ModuleName,
		slashingtypes.ModuleName,
		stakingtypes.ModuleName,
		upgradetypes.ModuleName,
		minttypes.ModuleName,
		ibcexported.ModuleName,
		ibctransfertypes.ModuleName,
		genutiltypes.ModuleName,
		authz.ModuleName,
		icatypes.ModuleName,
		ibcfeetypes.ModuleName,
	)
	app.ModuleManager.SetOrderEndBlockers(
		govtypes.ModuleName,
		stakingtypes.ModuleName,
		ibcexported.ModuleName,
		ibctransfertypes.ModuleName,
		capabilitytypes.ModuleName,
		genutiltypes.ModuleName,
		icatypes.ModuleName,
		ibcfeetypes.ModuleName,
		emissions.ModuleName,
	)

	// create the simulation manager and define the order of the modules for deterministic simulations
	// NOTE: this is not required apps that don't use the simulator for fuzz testing transactions
	app.sm = module.NewSimulationManagerFromAppModules(app.ModuleManager.Modules, make(map[string]module.AppModuleSimulation, 0))
	app.sm.RegisterStoreDecoders()

	topicsHandler := NewTopicsHandler(app.EmissionsKeeper)
	app.SetPrepareProposal(topicsHandler.PrepareProposalHandler())

	app.setupUpgradeHandlers()

	app.SetInitChainer(func(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
		app.UpgradeKeeper.SetModuleVersionMap(ctx, app.ModuleManager.GetVersionMap())
		return app.App.InitChainer(ctx, req)
	})

	if err := app.Load(loadLatest); err != nil {
		return nil, err
	}

	return app, nil
}

// LegacyAmino returns AlloraApp's amino codec.
func (app *AlloraApp) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

// AppCodec returns App's app codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *AlloraApp) AppCodec() codec.Codec {
	return app.appCodec
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

// ibctesting.TestingApp compatibility
func (app *AlloraApp) GetBaseApp() *baseapp.BaseApp {
	return app.App.BaseApp
}

// ibctesting.TestingApp compatibility
func (app *AlloraApp) GetStakingKeeper() ibctestingtypes.StakingKeeper {
	return app.StakingKeeper
}

// ibctesting.TestingApp compatibility
func (app *AlloraApp) GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper {
	return app.ScopedIBCKeeper
}

// ibctesting.TestingApp compatibility
func (app *AlloraApp) GetTxConfig() client.TxConfig {
	return app.txConfig
}

// ibctesting.TestingApp compatibility
func (app *AlloraApp) LastCommitID() storetypes.CommitID {
	return app.BaseApp.LastCommitID()
}

// ibctesting.TestingApp compatibility
func (app *AlloraApp) LastBlockHeight() int64 {
	return app.BaseApp.LastBlockHeight()
}

// InitAppForTestnet initializes the app for testnet
func InitAppForTestnet(app *AlloraApp, newValAddr []byte, newValPubKey crypto.PubKey, newOperatorAddress, upgradeToTrigger string) (*AlloraApp, error) {
	ctx := app.BaseApp.NewUncachedContext(true, tmproto.Header{})

	// Implement the required changes
	err := initStaking(app, ctx, newValPubKey, newOperatorAddress)
	if err != nil {
		return nil, err
	}
	err = initDistribution(app, ctx, newOperatorAddress)
	if err != nil {
		return nil, err
	}
	err = initSlashing(app, ctx, newValAddr)
	if err != nil {
		return nil, err
	}
	err = initBank(app, ctx)
	if err != nil {
		return nil, err
	}

	return app, nil
}

func initStaking(app *AlloraApp, ctx sdk.Context, newValPubKey crypto.PubKey, newOperatorAddress string) error {
	// Create Validator struct for our new validator
	pubkey := &ed25519.PubKey{Key: newValPubKey.Bytes()}
	pubkeyAny, err := types.NewAnyWithValue(pubkey)
	if err != nil {
		return err
	}
	_, bz, err := bech32.DecodeAndConvert(newOperatorAddress)
	if err != nil {
		return err
	}
	bech32Addr, err := bech32.ConvertAndEncode("alloravaloper", bz)
	if err != nil {
		return err
	}

	newVal := stakingtypes.Validator{
		OperatorAddress: bech32Addr,
		ConsensusPubkey: pubkeyAny,
		Jailed:          false,
		Status:          stakingtypes.Bonded,
		Tokens:          math.NewInt(900000000000000),
		DelegatorShares: math.LegacyNewDec(10000000),
		Description:     stakingtypes.Description{Moniker: "Devnet Validator"},
		Commission: stakingtypes.Commission{
			CommissionRates: stakingtypes.CommissionRates{
				Rate:          math.LegacyNewDec(5),
				MaxRate:       math.LegacyNewDec(10),
				MaxChangeRate: math.LegacyNewDec(5),
			},
		},
		MinSelfDelegation: math.OneInt(),
	}

	// Remove all validators from power store
	stakingKey := app.GetKey(stakingtypes.ModuleName)
	stakingStore := ctx.KVStore(stakingKey)
	iterator, err := app.StakingKeeper.ValidatorsPowerStoreIterator(ctx)
	if err != nil {
		return err
	}
	for ; iterator.Valid(); iterator.Next() {
		stakingStore.Delete(iterator.Key())
	}
	iterator.Close()

	// Remove all valdiators from last validators store
	iterator, err = app.StakingKeeper.LastValidatorsIterator(ctx)
	if err != nil {
		return err
	}
	for ; iterator.Valid(); iterator.Next() {
		addr := sdk.ValAddress(stakingtypes.AddressFromLastValidatorPowerKey(iterator.Key()))
		app.StakingKeeper.DeleteLastValidatorPower(ctx, addr)
	}
	iterator.Close()

	// Add our validator to power and last validators store
	err = app.StakingKeeper.SetValidator(ctx, newVal)
	if err != nil {
		return err
	}
	err = app.StakingKeeper.SetValidatorByConsAddr(ctx, newVal)
	if err != nil {
		return err
	}
	err = app.StakingKeeper.SetValidatorByPowerIndex(ctx, newVal)
	if err != nil {
		return err
	}
	err = app.StakingKeeper.SetLastValidatorPower(ctx, sdk.ValAddress(newVal.GetOperator()), 0)
	if err != nil {
		return err
	}
	err = app.StakingKeeper.Hooks().AfterValidatorCreated(ctx, sdk.ValAddress(newVal.GetOperator()))
	if err != nil {
		return err
	}

	return nil
}

func initDistribution(app *AlloraApp, ctx sdk.Context, newOperatorAddress string) error {
	err := app.DistrKeeper.SetValidatorHistoricalRewards(ctx, sdk.ValAddress(newOperatorAddress), 0, distrtypes.NewValidatorHistoricalRewards(sdk.DecCoins{}, 1))
	if err != nil {
		return err
	}
	err = app.DistrKeeper.SetValidatorCurrentRewards(ctx, sdk.ValAddress(newOperatorAddress), distrtypes.NewValidatorCurrentRewards(sdk.DecCoins{}, 1))
	if err != nil {
		return err
	}
	err = app.DistrKeeper.SetValidatorAccumulatedCommission(ctx, sdk.ValAddress(newOperatorAddress), distrtypes.InitialValidatorAccumulatedCommission())
	if err != nil {
		return err
	}
	err = app.DistrKeeper.SetValidatorOutstandingRewards(ctx, sdk.ValAddress(newOperatorAddress), distrtypes.ValidatorOutstandingRewards{Rewards: sdk.DecCoins{}})
	if err != nil {
		return err
	}
	return nil
}

func initSlashing(app *AlloraApp, ctx sdk.Context, newValAddr []byte) error {
	newConsAddr := sdk.ConsAddress(newValAddr)
	signingInfo := slashingtypes.ValidatorSigningInfo{
		Address:     newConsAddr.String(),
		StartHeight: ctx.BlockHeight(),
		Tombstoned:  false,
	}
	err := app.SlashingKeeper.SetValidatorSigningInfo(ctx, newConsAddr, signingInfo)
	if err != nil {
		return err
	}
	return nil
}

func initBank(app *AlloraApp, ctx sdk.Context) error {
	defaultCoins := sdk.NewCoins(sdk.NewInt64Coin("uallo", 1000000000000))
	testAccounts := []string{
		"allo1az9ru7rnkaqhgcr4h6tpvtr3arem2whva74cev",
		// Add more test accounts as needed
	}
	for _, accAddr := range testAccounts {
		addr, err := sdk.AccAddressFromBech32(accAddr)
		if err != nil {
			return err
		}
		err = app.BankKeeper.MintCoins(ctx, minttypes.ModuleName, defaultCoins)
		if err != nil {
			return err
		}
		err = app.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, addr, defaultCoins)
		if err != nil {
			return err
		}
	}
	return nil
}
