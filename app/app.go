package app

import (
	_ "embed"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/core/appconfig"
	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
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
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	consensuskeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	ibctransferkeeper "github.com/cosmos/ibc-go/v8/modules/apps/transfer/keeper" //nolint:staticcheck
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibcclienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types" // TODO move to new non-deprecated package
	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/allora-network/allora-chain/inflation"
	emissionsKeeper "github.com/allora-network/allora-chain/x/emissions/keeper"

	_ "cosmossdk.io/api/cosmos/tx/config/v1" // import for side-effects
	_ "github.com/allora-network/allora-chain/x/emissions/module"
	_ "github.com/cosmos/cosmos-sdk/x/auth"           // import for side-effects
	_ "github.com/cosmos/cosmos-sdk/x/auth/tx/config" // import for side-effects
	_ "github.com/cosmos/cosmos-sdk/x/bank"           // import for side-effects
	_ "github.com/cosmos/cosmos-sdk/x/consensus"      // import for side-effects
	_ "github.com/cosmos/cosmos-sdk/x/mint"           // import for side-effects
	_ "github.com/cosmos/cosmos-sdk/x/staking"        // import for side-effects
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
	BankKeeper            bankkeeper.Keeper
	StakingKeeper         *stakingkeeper.Keeper
	ConsensusParamsKeeper consensuskeeper.Keeper
	MintKeeper            mintkeeper.Keeper
	emissionsKeeper       emissionsKeeper.Keeper

	// simulation manager
	sm *module.SimulationManager
}

func init() {
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
				minttypes.InflationCalculationFn(inflation.CustomInflationCalculation),
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
		&app.ConsensusParamsKeeper,
		&app.MintKeeper,
		&app.emissionsKeeper,
	); err != nil {
		return nil, err
	}

	app.App = appBuilder.Build(db, traceStore, baseAppOptions...)

	// register streaming services
	if err := app.RegisterStreamingServices(appOpts, app.kvStoreKeys()); err != nil {
		return nil, err
	}

	/****  Module Options ****/

	// create the simulation manager and define the order of the modules for deterministic simulations
	// NOTE: this is not required apps that don't use the simulator for fuzz testing transactions
	app.sm = module.NewSimulationManagerFromAppModules(app.ModuleManager.Modules, make(map[string]module.AppModuleSimulation, 0))
	app.sm.RegisterStoreDecoders()

	app.SetInitChainer(app.InitChainer)

	if err := app.Load(loadLatest); err != nil {
		return nil, err
	}

	return app, nil
}

const (
	// TypeUnrecognized means coin type is unrecognized
	TypeUnrecognized = iota
	// TypeGeneralMessage is a pure message
	TypeGeneralMessage
	// TypeGeneralMessageWithToken is a general message with token
	TypeGeneralMessageWithToken
	// TypeSendToken is a direct token transfer
	TypeSendToken
)

type AxelarBody struct {
	DestinationChain   string     `json:"destination_chain"`
	DestinationAddress string     `json:"destination_address"`
	Payload            []byte     `json:"payload"`
	Type               int64      `json:"type"`
	Fee                *AxelarFee `json:"fee"`
}

type AxelarFee struct {
	Amount    string `json:"amount"`
	Recipient string `json:"recipient"`
}

type msgServer struct {
	ibcTransferK ibctransferkeeper.Keeper
}

const AxelarGMPAcc = "axelar1dv4u5k73pzqrxlzujxg3qp8kvc3pje7jtdvu72npnt5zhq05ejcsn5qme5"

/*
// TODO. figure out why this doesn't compile
// app/app.go:207:9: cannot use &msgServer{…} (value of type *msgServer) as
// "cosmossdk.io/x/circuit/types".MsgServer value in return statement: *msgServer
// does not implement "cosmossdk.io/x/circuit/types".MsgServer (missing method AuthorizeCircuitBreaker)

func NewMsgServerImpl(ibcTransferK ibctransferkeeper.Keeper) types.MsgServer {
	return &msgServer{
		ibcTransferK: ibcTransferK,
	}
}
*/

func (k msgServer) BuildAndSendIBCDataMessage(
	goCtx sdk.Context,
	msg struct {
		DestinationChain   string
		DestinationAddress string
		IbcChannel         string
		Sender             sdk.AccAddress
		ExecutorAccount    string
		FeeTokenAndAmount  sdk.Coin
		bytesData          string
	}) error {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// build payload that can be decoded by solidity
	bytesDataType, err := abi.NewType("bytes", "bytes", nil)
	if err != nil {
		return err
	}

	payload, err := abi.Arguments{{Type: bytesDataType}}.Pack(msg.bytesData)
	if err != nil {
		return err
	}

	axelarMemo := AxelarBody{
		DestinationChain:   msg.DestinationChain,
		DestinationAddress: msg.DestinationAddress,
		Payload:            payload,
		Type:               TypeGeneralMessage,
		Fee: &AxelarFee{
			Amount:    msg.FeeTokenAndAmount.Amount.String(),
			Recipient: msg.ExecutorAccount,
		},
	}

	axelarMemoJson, err := json.Marshal(axelarMemo)
	if err != nil {
		return err
	}

	transferMsg := ibctransfertypes.NewMsgTransfer(
		ibctransfertypes.PortID,
		msg.IbcChannel,
		msg.FeeTokenAndAmount,
		string(msg.Sender),
		AxelarGMPAcc,
		ibcclienttypes.ZeroHeight(),
		uint64(ctx.BlockTime().Add(6*time.Hour).UnixNano()),
		string(axelarMemoJson),
	)
	_, err = k.ibcTransferK.Transfer(goCtx, transferMsg)
	if err != nil {
		return err
	}

	return nil
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

func (app *AlloraApp) kvStoreKeys() map[string]*storetypes.KVStoreKey {
	keys := make(map[string]*storetypes.KVStoreKey)
	for _, k := range app.GetStoreKeys() {
		if kv, ok := k.(*storetypes.KVStoreKey); ok {
			keys[kv.Name()] = kv
		}
	}

	return keys
}

// SimulationManager implements the SimulationApp interface
func (app *AlloraApp) SimulationManager() *module.SimulationManager {
	return app.sm
}

// InitChainer updates at chain initialization
func (app *AlloraApp) InitChainer(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	var genesisState map[string]json.RawMessage
	if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}

	var (
		initialBlockRewardBTC = 50
		blocksPerYear         = 6311520 //TODO: Check Block Time --> BTC is actually 52560 (10min)
		totalALLORA           = 21000000
		initialProvisions     = math.LegacyNewDec(int64(initialBlockRewardBTC * blocksPerYear))
		initialInflation      = initialProvisions.QuoInt64(int64(totalALLORA))
	)

	app.MintKeeper.Minter.Set(ctx, minttypes.Minter{
		Inflation:        initialInflation,
		AnnualProvisions: initialProvisions,
	})

	return app.ModuleManager.InitGenesis(ctx, app.appCodec, genesisState)
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
