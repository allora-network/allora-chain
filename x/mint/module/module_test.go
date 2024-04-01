package mint_test

import (
	"testing"
	"time"

	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	cosmosMath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/app/params"
	state "github.com/allora-network/allora-chain/x/emissions"

	"github.com/allora-network/allora-chain/x/mint/keeper"
	mint "github.com/allora-network/allora-chain/x/mint/module"
	"github.com/allora-network/allora-chain/x/mint/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"

	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/suite"
)

const (
	multiPerm  = "multiple permissions account"
	randomPerm = "random permission"
)

type MintModuleTestSuite struct {
	suite.Suite

	ctx           sdk.Context
	accountKeeper types.AccountKeeper
	stakingKeeper types.StakingKeeper
	bankKeeper    types.BankKeeper
	appModule     mint.AppModule
	mintKeeper    keeper.Keeper
}

// SetupTest setups a new test, to be run before each test case
func (s *MintModuleTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(key)
	encCfg := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, staking.AppModuleBasic{}, bank.AppModuleBasic{}, mint.AppModuleBasic{})
	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithHeaderInfo(header.Info{Time: time.Now()})

	maccPerms := map[string][]string{
		"fee_collector":                 nil,
		"mint":                          {"minter"},
		state.AlloraStakingAccountName:  {"burner", "minter", "staking"},
		state.AlloraRequestsAccountName: {"burner", "minter", "staking"},
		"bonded_tokens_pool":            {"burner", "staking"},
		"not_bonded_tokens_pool":        {"burner", "staking"},
		multiPerm:                       {"burner", "minter", "staking"},
		randomPerm:                      {"random"},
	}

	accountKeeper := authkeeper.NewAccountKeeper(
		encCfg.Codec,
		storeService,
		authtypes.ProtoBaseAccount,
		maccPerms,
		authcodec.NewBech32Codec(params.Bech32PrefixAccAddr),
		params.Bech32PrefixAccAddr,
		authtypes.NewModuleAddress("gov").String(),
	)

	bankKeeper := bankkeeper.NewBaseKeeper(
		encCfg.Codec,
		storeService,
		accountKeeper,
		map[string]bool{},
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		log.NewNopLogger(),
	)

	stakingKeeper := stakingkeeper.NewKeeper(
		encCfg.Codec,
		storeService,
		accountKeeper,
		bankKeeper,
		authtypes.NewModuleAddress("gov").String(),
		addresscodec.NewBech32Codec(sdk.Bech32PrefixValAddr),
		addresscodec.NewBech32Codec(sdk.Bech32PrefixConsAddr),
	)

	mintKeeper := keeper.NewKeeper(
		encCfg.Codec,
		storeService,
		stakingKeeper,
		accountKeeper,
		bankKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress("gov").String(),
	)

	s.ctx = ctx
	s.accountKeeper = accountKeeper
	s.bankKeeper = bankKeeper
	s.stakingKeeper = stakingKeeper
	s.mintKeeper = mintKeeper

	appModule := mint.NewAppModule(encCfg.Codec, s.mintKeeper, s.accountKeeper)
	defaultGenesis := appModule.DefaultGenesis(encCfg.Codec)
	appModule.InitGenesis(ctx, encCfg.Codec, defaultGenesis)
	s.appModule = appModule
}

func TestMintModuleTestSuite(t *testing.T) {
	suite.Run(t, new(MintModuleTestSuite))
}

func (s *MintModuleTestSuite) TestMintingBelowMaxSupplyInHalvingBlock() {
	defaultMintModuleParams := types.DefaultParams()
	s.mintKeeper.Params.Set(s.ctx, defaultMintModuleParams)

	// Fetch updated minter and params
	minterBeforeUpdate, err := s.mintKeeper.Minter.Get(s.ctx)
	s.Require().NoError(err)
	paramBeforeUpdate, err := s.mintKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	expectedBlockProvision := paramBeforeUpdate.CurrentBlockProvision.QuoUint64(2)

	// Set block height and existing supply for the test case
	blockHeight := int64(25246080)
	existingSupply := cosmosMath.NewIntFromBigInt(cosmosMath.NewUintFromString("50000000000000000000000000").BigInt()) // Equivalent 50 million * 1e18 uallo

	ctx := s.ctx.WithBlockHeight(blockHeight)
	s.mintKeeper.MintCoins(ctx, sdk.NewCoins(sdk.NewCoin(params.BaseCoinUnit, existingSupply)))

	// Call BeginBlocker
	err = mint.BeginBlocker(ctx, s.mintKeeper)
	s.Require().NoError(err)

	// Fetch updated minter and params
	minter, err := s.mintKeeper.Minter.Get(ctx)
	s.Require().NoError(err)
	params, err := s.mintKeeper.Params.Get(ctx)
	s.Require().NoError(err)

	// Verify minting occurred and current block provision remains as set
	s.Require().Less(minter.Inflation.String(), minterBeforeUpdate.Inflation.String(), "New Inflation should be less than previous value")
	s.Require().Equal(minterBeforeUpdate.AnnualProvisions.QuoInt64(2), minter.AnnualProvisions, "Annual should equal half of previous value")
	s.Require().Equal(expectedBlockProvision.String(), params.CurrentBlockProvision.String(), "CurrentBlockProvision should equal half of previous value")
}

func (s *MintModuleTestSuite) TestMintingAtMaxSupply() {
	defaultMintModuleParams := types.DefaultParams()
	s.mintKeeper.Params.Set(s.ctx, defaultMintModuleParams)

	// Set block height and existing supply for the test case
	blockHeight := int64(2)
	existingSupply := cosmosMath.NewIntFromBigInt(defaultMintModuleParams.MaxSupply.BigInt())
	expectedProvision := cosmosMath.NewUint(0) // Expecting no minting, hence provision should be zero.

	ctx := s.ctx.WithBlockHeight(blockHeight)
	s.mintKeeper.MintCoins(ctx, sdk.NewCoins(sdk.NewCoin(params.BaseCoinUnit, existingSupply)))

	// Call BeginBlocker
	err := mint.BeginBlocker(ctx, s.mintKeeper)
	s.Require().NoError(err)

	// Fetch updated minter and params
	minter, err := s.mintKeeper.Minter.Get(ctx)
	s.Require().NoError(err)
	params, err := s.mintKeeper.Params.Get(ctx)
	s.Require().NoError(err)

	// Verify no minting occurred and current block provision is zero
	s.Require().Equal(cosmosMath.LegacyZeroDec(), minter.Inflation, "Inflation should be zero")
	s.Require().Equal(cosmosMath.LegacyZeroDec(), minter.AnnualProvisions, "Annual provisions should be zero")
	s.Require().Equal(expectedProvision.String(), params.CurrentBlockProvision.String(), "CurrentBlockProvision should be zero")
}
