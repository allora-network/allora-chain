package v6_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/golang/mock/gomock"

	"cosmossdk.io/core/store"
	minttestutil "github.com/allora-network/allora-chain/x/mint/testutil"

	"github.com/allora-network/allora-chain/x/mint/keeper"
	mint "github.com/allora-network/allora-chain/x/mint/module"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	v6 "github.com/allora-network/allora-chain/x/mint/migrations/v6"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	storetypes "cosmossdk.io/store/types"
	oldV5Types "github.com/allora-network/allora-chain/x/mint/migrations/v6/oldtypes"
	cosmostestutil "github.com/cosmos/cosmos-sdk/testutil"
)

type MintV6MigrationTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	ctx          sdk.Context
	storeService store.KVStoreService
	mintKeeper   *keeper.Keeper
}

func TestMintV6MigrationTestSuite(t *testing.T) {
	suite.Run(t, new(MintV6MigrationTestSuite))
}

func (s *MintV6MigrationTestSuite) SetupTest() {
	encCfg := moduletestutil.MakeTestEncodingConfig(mint.AppModule{}) //nolint:exhaustruct
	key := storetypes.NewKVStoreKey(minttypes.StoreKey)
	storeService := runtime.NewKVStoreService(key)
	s.storeService = storeService
	testCtx := cosmostestutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))

	// gomock initializations
	s.ctrl = gomock.NewController(s.T())
	accountKeeper := minttestutil.NewMockAccountKeeper(s.ctrl)
	bankKeeper := minttestutil.NewMockBankKeeper(s.ctrl)
	emissionsKeeper := minttestutil.NewMockEmissionsKeeper(s.ctrl)
	stakingKeeper := minttestutil.NewMockStakingKeeper(s.ctrl)
	accountKeeper.EXPECT().GetModuleAddress(minttypes.ModuleName).Return(authtypes.NewModuleAddress(minttypes.ModuleName))
	mintKeeper := keeper.NewKeeper(
		encCfg.Codec,
		storeService,
		stakingKeeper,
		accountKeeper,
		bankKeeper,
		emissionsKeeper,
		authtypes.FeeCollectorName,
	)

	s.ctx = testCtx.Ctx
	s.storeService = storeService
	s.mintKeeper = &mintKeeper
}

// The migration keeps every existing param and adds the investors and team
// vesting delay with a value of zero.
func (s *MintV6MigrationTestSuite) TestMigrateParams() {
	storageService := s.mintKeeper.GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.ctx))
	cdc := s.mintKeeper.GetBinaryCodec()

	defaultParams := minttypes.DefaultParams()
	defaultParams.EmissionEnabled = true
	paramsOld := oldV5Types.Params{
		MintDenom:                              defaultParams.MintDenom,
		MaxSupply:                              defaultParams.MaxSupply,
		FEmission:                              defaultParams.FEmission,
		OneMonthSmoothingDegree:                defaultParams.OneMonthSmoothingDegree,
		EcosystemTreasuryPercentOfTotalSupply:  defaultParams.EcosystemTreasuryPercentOfTotalSupply,
		FoundationTreasuryPercentOfTotalSupply: defaultParams.FoundationTreasuryPercentOfTotalSupply,
		ParticipantsPercentOfTotalSupply:       defaultParams.ParticipantsPercentOfTotalSupply,
		InvestorsPercentOfTotalSupply:          defaultParams.InvestorsPercentOfTotalSupply,
		TeamPercentOfTotalSupply:               defaultParams.TeamPercentOfTotalSupply,
		MaximumMonthlyPercentageYield:          defaultParams.MaximumMonthlyPercentageYield,
		InvestorsPreseedPercentOfTotalSupply:   defaultParams.InvestorsPreseedPercentOfTotalSupply,
		EmissionEnabled:                        true,
	}
	store.Set(minttypes.ParamsKey, cdc.MustMarshal(&paramsOld))

	err := v6.MigrateStore(s.ctx, *s.mintKeeper)
	s.Require().NoError(err)

	params, err := s.mintKeeper.GetParams(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(defaultParams, params)
	s.Require().Equal(uint64(0), params.InvestorsTeamVestingDelayMonths)
}

// A store without params cannot be migrated.
func (s *MintV6MigrationTestSuite) TestMigrateParamsMissing() {
	err := v6.MigrateStore(s.ctx, *s.mintKeeper)
	s.Require().Error(err)
}
