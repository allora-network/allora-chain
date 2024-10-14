package cmd

import (
	"fmt"
	"io"
	"strings"
	"time"

	"cosmossdk.io/errors"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/libs/bytes"
	tmos "github.com/cometbft/cometbft/libs/os"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client/flags"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"

	"github.com/allora-network/allora-chain/app"
)

const (
	valVotingPower int64 = 900000000000000
)

var (
	flagAccountsToFund = "accounts-to-fund"
)

type valArgs struct {
	newValAddr         bytes.HexBytes
	newOperatorAddress string
	newValPubKey       crypto.PubKey
	accountsToFund     []sdk.AccAddress
	upgradeToTrigger   string
	homeDir            string
}

func NewInPlaceTestnetCmd(addStartFlags servertypes.ModuleInitFlags) *cobra.Command {
	cmd := server.InPlaceTestnetCreator(newTestnetApp)
	addStartFlags(cmd)
	cmd.Short = "Updates chain's application and consensus state with provided validator info and starts the node"
	cmd.Long = `The test command modifies both application and consensus stores within a local mainnet node and starts the node,
with the aim of facilitating testing procedures. This command replaces existing validator data with updated information,
thereby removing the old validator set and introducing a new set suitable for local testing purposes. By altering the state extracted from the mainnet node,
it enables developers to configure their local environments to reflect mainnet conditions more accurately.`

	cmd.Example = fmt.Sprintf(`%sd in-place-testnet testing-1 allovaloper1xepena2sx9yzdwjtln0c5ycznw8vvzmt7ktnk5 --home $HOME/.%sd/data --accounts-to-fund="allo1xepena2sx9yzdwjtln0c5ycznw8vvzmtfjds2c"`, "allora", "allora")

	cmd.Flags().String(flagAccountsToFund, "", "Comma-separated list of account addresses that will be funded for testing purposes")
	return cmd
}

// newTestnetApp starts by running the normal newApp method. From there, the app interface returned is modified in order
// for a testnet to be created from the provided app.
func newTestnetApp(logger log.Logger, db dbm.DB, traceStore io.Writer, appOpts servertypes.AppOptions) servertypes.Application {
	// Create an app and type cast to an App
	newApp := newApp(logger, db, traceStore, appOpts)
	testApp, ok := newApp.(*app.AlloraApp)
	if !ok {
		panic("app created from newApp is not of type App")
	}

	// Get command args
	args, err := getCommandArgs(appOpts)
	if err != nil {
		panic(errors.Wrap(err, "newTestnetApp(): failed to get command args"))
	}

	return initAppForTestnet(testApp, args)
}

func initAppForTestnet(app *app.AlloraApp, args valArgs) *app.AlloraApp {
	// Required Changes:
	ctx := app.BaseApp.NewUncachedContext(true, tmproto.Header{}) //nolint:exhaustruct // cosmos-sdk logic

	pubkey := &ed25519.PubKey{Key: args.newValPubKey.Bytes()}
	pubkeyAny, err := codectypes.NewAnyWithValue(pubkey)
	if err != nil {
		tmos.Exit(errors.Wrap(err, "initAppForTestnet(): failed to create pubkey any").Error())
	}

	// STAKING
	//

	// Create Validator struct for our new validator.
	_, bz, err := bech32.DecodeAndConvert(args.newOperatorAddress)
	if err != nil {
		tmos.Exit(errors.Wrap(err, "initAppForTestnet(): failed to decode operator address").Error())
	}
	bech32Addr, err := bech32.ConvertAndEncode("allovaloper", bz)
	if err != nil {
		tmos.Exit(errors.Wrap(err, "initAppForTestnet(): failed to convert and encode operator address").Error())
	}
	newVal := stakingtypes.Validator{
		OperatorAddress: bech32Addr,
		ConsensusPubkey: pubkeyAny,
		Jailed:          false,
		Status:          stakingtypes.Bonded,
		Tokens:          math.NewInt(9e18),
		DelegatorShares: math.LegacyNewDec(1e18),
		Description: stakingtypes.Description{
			Moniker:         "Testnet Validator",
			Identity:        "",
			Website:         "",
			SecurityContact: "",
			Details:         "",
		},
		Commission: stakingtypes.Commission{
			CommissionRates: stakingtypes.CommissionRates{
				Rate:          math.LegacyMustNewDecFromStr("0.05"),
				MaxRate:       math.LegacyMustNewDecFromStr("0.1"),
				MaxChangeRate: math.LegacyMustNewDecFromStr("0.05"),
			},
			UpdateTime: time.Time{},
		},
		MinSelfDelegation:       math.OneInt(),
		UnbondingHeight:         0,
		UnbondingTime:           time.Time{},
		UnbondingOnHoldRefCount: 0,
		UnbondingIds:            []uint64{},
	}

	// Remove all validators from power store
	stakingKey := app.GetKey(stakingtypes.ModuleName)
	stakingStore := ctx.KVStore(stakingKey)
	validatorsPowerStoreIterator, err := app.StakingKeeper.ValidatorsPowerStoreIterator(ctx)
	if err != nil {
		tmos.Exit(errors.Wrap(err, "initAppForTestnet(): failed to get validators power store iterator").Error())
	}
	defer validatorsPowerStoreIterator.Close()
	for ; validatorsPowerStoreIterator.Valid(); validatorsPowerStoreIterator.Next() {
		stakingStore.Delete(validatorsPowerStoreIterator.Key())
	}
	validatorsPowerStoreIterator.Close()

	// Remove all valdiators from last validators store
	lastValidatorsIterator, err := app.StakingKeeper.LastValidatorsIterator(ctx)
	if err != nil {
		tmos.Exit(errors.Wrap(err, "initAppForTestnet(): failed to get last validators iterator").Error())
	}
	defer lastValidatorsIterator.Close()
	for ; lastValidatorsIterator.Valid(); lastValidatorsIterator.Next() {
		stakingStore.Delete(lastValidatorsIterator.Key())
	}
	lastValidatorsIterator.Close()

	// Remove all validators from validators store
	validatorsIterator := storetypes.KVStorePrefixIterator(stakingStore, stakingtypes.ValidatorsKey)
	defer validatorsIterator.Close()
	for ; validatorsIterator.Valid(); validatorsIterator.Next() {
		stakingStore.Delete(validatorsIterator.Key())
	}
	validatorsIterator.Close()

	// Remove all validators from unbonding queue
	validatorQueueIterator := storetypes.KVStorePrefixIterator(stakingStore, stakingtypes.ValidatorQueueKey)
	defer validatorQueueIterator.Close()
	for ; validatorQueueIterator.Valid(); validatorQueueIterator.Next() {
		stakingStore.Delete(validatorQueueIterator.Key())
	}
	validatorQueueIterator.Close()

	// Add our validator to power and last validators store
	err = app.StakingKeeper.SetValidator(ctx, newVal)
	if err != nil {
		tmos.Exit(errors.Wrap(err, "initAppForTestnet(): failed to set validator").Error())
	}
	err = app.StakingKeeper.SetValidatorByConsAddr(ctx, newVal)
	if err != nil {
		tmos.Exit(errors.Wrap(err, "initAppForTestnet(): failed to set validator by consensus address").Error())
	}
	err = app.StakingKeeper.SetValidatorByPowerIndex(ctx, newVal)
	if err != nil {
		tmos.Exit(errors.Wrap(err, "initAppForTestnet(): failed to set validator by power index").Error())
	}
	valAddr, err := sdk.ValAddressFromBech32(newVal.GetOperator())
	if err != nil {
		tmos.Exit(errors.Wrap(err, "initAppForTestnet(): failed to convert validator address to bytes").Error())
	}
	err = app.StakingKeeper.SetLastValidatorPower(ctx, valAddr, 9e18)
	if err != nil {
		tmos.Exit(errors.Wrap(err, "initAppForTestnet(): failed to set last validator power").Error())
	}
	err = app.StakingKeeper.Hooks().AfterValidatorCreated(ctx, valAddr)
	if err != nil {
		panic(errors.Wrap(err, "initAppForTestnet(): failed to call after validator created hook").Error())
	}

	// DISTRIBUTION
	//

	valAddr, err = sdk.ValAddressFromBech32(newVal.GetOperator())
	if err != nil {
		tmos.Exit(errors.Wrap(err, "initAppForTestnet(): failed to convert validator address to bytes").Error())
	}
	// Initialize records for this validator across all distribution stores
	err = app.DistrKeeper.SetValidatorHistoricalRewards(ctx, valAddr, 0, distrtypes.NewValidatorHistoricalRewards(sdk.DecCoins{}, 1))
	if err != nil {
		tmos.Exit(errors.Wrap(err, "initAppForTestnet(): failed to set validator historical rewards").Error())
	}
	err = app.DistrKeeper.SetValidatorCurrentRewards(ctx, valAddr, distrtypes.NewValidatorCurrentRewards(sdk.DecCoins{}, 1))
	if err != nil {
		tmos.Exit(errors.Wrap(err, "initAppForTestnet(): failed to set validator current rewards").Error())
	}
	err = app.DistrKeeper.SetValidatorAccumulatedCommission(ctx, valAddr, distrtypes.InitialValidatorAccumulatedCommission())
	if err != nil {
		tmos.Exit(errors.Wrap(err, "initAppForTestnet(): failed to set validator accumulated commission").Error())
	}
	err = app.DistrKeeper.SetValidatorOutstandingRewards(ctx, valAddr, distrtypes.ValidatorOutstandingRewards{Rewards: sdk.DecCoins{}})
	if err != nil {
		tmos.Exit(errors.Wrap(err, "initAppForTestnet(): failed to set validator outstanding rewards").Error())
	}

	// SLASHING
	//

	// Set validator signing info for our new validator.
	newConsAddr := sdk.ConsAddress(args.newValAddr.Bytes())
	newValidatorSigningInfo := slashingtypes.ValidatorSigningInfo{
		Address:             newConsAddr.String(),
		StartHeight:         app.LastBlockHeight() - 1,
		Tombstoned:          false,
		IndexOffset:         0,
		JailedUntil:         time.Time{},
		MissedBlocksCounter: 0,
	}
	err = app.SlashingKeeper.SetValidatorSigningInfo(ctx, newConsAddr, newValidatorSigningInfo)
	if err != nil {
		tmos.Exit(errors.Wrap(err, "initAppForTestnet(): failed to set validator signing info").Error())
	}

	// BANK
	//
	bondDenom, err := app.StakingKeeper.BondDenom(ctx)
	if err != nil {
		tmos.Exit(errors.Wrap(err, "initAppForTestnet(): failed to get bond denom").Error())
	}

	defaultCoins := sdk.NewCoins(sdk.NewInt64Coin(bondDenom, 9e18))

	// Fund local accounts
	for _, account := range args.accountsToFund {
		err := app.BankKeeper.MintCoins(ctx, minttypes.ModuleName, defaultCoins)
		if err != nil {
			tmos.Exit(errors.Wrap(err, "initAppForTestnet(): failed to mint coins").Error())
		}
		err = app.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, account, defaultCoins)
		if err != nil {
			tmos.Exit(errors.Wrap(err, "initAppForTestnet(): failed to send coins from module to account").Error())
		}
	}

	if args.upgradeToTrigger != "" {
		upgradePlan := upgradetypes.Plan{
			Name:   args.upgradeToTrigger,
			Height: app.LastBlockHeight() + 10,
		}
		err = app.UpgradeKeeper.ScheduleUpgrade(ctx, upgradePlan)
		if err != nil {
			tmos.Exit(errors.Wrap(err, "initAppForTestnet(): failed to schedule upgrade").Error())
		}
	}

	return app
}

// parse the input flags and returns valArgs
func getCommandArgs(appOpts servertypes.AppOptions) (valArgs, error) {
	newValAddr, ok := appOpts.Get(server.KeyNewValAddr).(bytes.HexBytes)
	if !ok {
		panic("newValAddr is not of type bytes.HexBytes")
	}
	newValPubKey, ok := appOpts.Get(server.KeyUserPubKey).(crypto.PubKey)
	if !ok {
		panic("newValPubKey is not of type crypto.PubKey")
	}
	newOperatorAddress, ok := appOpts.Get(server.KeyNewOpAddr).(string)
	if !ok {
		panic("newOperatorAddress is not of type string")
	}
	upgradeToTrigger, ok := appOpts.Get(server.KeyTriggerTestnetUpgrade).(string)
	if !ok {
		panic("upgradeToTrigger is not of type string")
	}

	// validate  and set accounts to fund
	accountsString := cast.ToString(appOpts.Get(flagAccountsToFund))

	accountsToFund := []sdk.AccAddress{}
	for _, account := range strings.Split(accountsString, ",") {
		if account != "" {
			addr, err := sdk.AccAddressFromBech32(account)
			if err != nil {
				return valArgs{}, fmt.Errorf("invalid bech32 address format %w", err)
			}
			accountsToFund = append(accountsToFund, addr)
		}
	}

	// home dir
	homeDir := cast.ToString(appOpts.Get(flags.FlagHome))
	if homeDir == "" {
		return valArgs{}, fmt.Errorf("invalid home dir")
	}

	return valArgs{
		newValAddr:         newValAddr,
		newOperatorAddress: newOperatorAddress,
		newValPubKey:       newValPubKey,
		accountsToFund:     accountsToFund,
		upgradeToTrigger:   upgradeToTrigger,
		homeDir:            homeDir,
	}, nil
}
