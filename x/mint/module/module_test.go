package mint_test

import (
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	cosmosMath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/app/params"
	state "github.com/allora-network/allora-chain/x/emissions"

	"github.com/allora-network/allora-chain/x/mint/module"
	"github.com/allora-network/allora-chain/x/mint/keeper"
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
		"fee_collector":                nil,
		"mint":                         {"minter"},
		state.AlloraStakingModuleName:  {"burner", "minter", "staking"},
		state.AlloraRequestsModuleName: {"burner", "minter", "staking"},
		"bonded_tokens_pool":           {"burner", "staking"},
		"not_bonded_tokens_pool":       {"burner", "staking"},
		multiPerm:                      {"burner", "minter", "staking"},
		randomPerm:                     {"random"},
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

func TestAbciTestSuite(t *testing.T) {
	fmt.Println("Running AbciTestSuite")
	a := new(MintModuleTestSuite)
	fmt.Println("Running AbciTestSuite", a)
	suite.Run(t, new(MintModuleTestSuite))
}

func (s *MintModuleTestSuite) TestBeginBlockerMinting() {

	defaultMintModuleParams := types.DefaultParams()
	s.mintKeeper.Params.Set(s.ctx, defaultMintModuleParams)

	// Define test cases
	testCases := []struct {
		name           string
		blockHeight    int64
		existingSupply cosmosMath.Int
		expectedMint   bool
	}{
		{
			name:           "Minting below max supply",
			blockHeight:    1,
			existingSupply: cosmosMath.NewIntFromBigInt(cosmosMath.NewUintFromString("500000000000000000000000000").BigInt()), // 50% of max supply
			expectedMint:   true,
		},
		{
			name:           "Minting at max supply",
			blockHeight:    2,
			existingSupply: mint.MaxSupply,
			expectedMint:   false,
		},
	}

	for _, tc := range testCases {
		// Setup each test case
		ctx := s.ctx.WithBlockHeight(tc.blockHeight)

		s.mintKeeper.MintCoins(ctx, sdk.NewCoins(sdk.NewCoin(params.BaseCoinUnit, tc.existingSupply)))

		// Call BeginBlocker
		err := mint.BeginBlocker(ctx, s.mintKeeper)
		s.Require().NoError(err)

		// Fetch updated minter and params
		minter, err := s.mintKeeper.Minter.Get(ctx)
		s.Require().NoError(err)

		if tc.expectedMint {
			// Verify minting occurred
			s.Require().NotEqual(cosmosMath.LegacyZeroDec(), minter.Inflation, "Inflation should not be zero")
			s.Require().NotEqual(cosmosMath.LegacyZeroDec(), minter.AnnualProvisions, "Annual provisions should not be zero")
		} else {
			// Verify no minting occurred
			s.Require().Equal(cosmosMath.LegacyZeroDec(), minter.Inflation, "Inflation should be zero")
			s.Require().Equal(cosmosMath.LegacyZeroDec(), minter.AnnualProvisions, "Annual provisions should be zero")
		}
	}
}
