package v5_test

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

	v5 "github.com/allora-network/allora-chain/x/mint/migrations/v5"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	storetypes "cosmossdk.io/store/types"
	oldV4Types "github.com/allora-network/allora-chain/x/mint/migrations/v5/oldtypes"
	cosmostestutil "github.com/cosmos/cosmos-sdk/testutil"
)

type MintV5MigrationTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	ctx          sdk.Context
	storeService store.KVStoreService
	mintKeeper   *keeper.Keeper
}

func TestMintV5MigrationTestSuite(t *testing.T) {
	suite.Run(t, new(MintV5MigrationTestSuite))
}

func (s *MintV5MigrationTestSuite) SetupTest() {
	encCfg := moduletestutil.MakeTestEncodingConfig(mint.AppModule{}) // nolint: exhaustruct
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

// In this test we check that the mint module params have been migrated
// and the expected new fields are added and set to true:
// EmissionEnabled
func (s *MintV5MigrationTestSuite) TestMigrateParams() {
	storageService := s.mintKeeper.GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.ctx))
	cdc := s.mintKeeper.GetBinaryCodec()

	defaultParams := minttypes.DefaultParams()
	paramsOld := oldV4Types.Params{
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
	}

	store.Set(minttypes.ParamsKey, cdc.MustMarshal(&paramsOld))

	// Run migration
	err := v5.MigrateStore(s.ctx, *s.mintKeeper)
	s.Require().NoError(err)

	paramsExpected := defaultParams
	paramsExpected.EmissionEnabled = true

	// TO BE ADDED:
	// - EmissionEnabled - set to true

	params, err := s.mintKeeper.GetParams(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(paramsExpected.MintDenom, params.MintDenom)
	s.Require().Equal(paramsExpected.MaxSupply, params.MaxSupply)
	s.Require().Equal(paramsExpected.FEmission, params.FEmission)
	s.Require().Equal(paramsExpected.OneMonthSmoothingDegree, params.OneMonthSmoothingDegree)
	s.Require().Equal(paramsExpected.EcosystemTreasuryPercentOfTotalSupply, params.EcosystemTreasuryPercentOfTotalSupply)
	s.Require().Equal(paramsExpected.FoundationTreasuryPercentOfTotalSupply, params.FoundationTreasuryPercentOfTotalSupply)
	s.Require().Equal(paramsExpected.ParticipantsPercentOfTotalSupply, params.ParticipantsPercentOfTotalSupply)
	s.Require().Equal(paramsExpected.InvestorsPercentOfTotalSupply, params.InvestorsPercentOfTotalSupply)
	s.Require().Equal(paramsExpected.TeamPercentOfTotalSupply, params.TeamPercentOfTotalSupply)
	s.Require().Equal(paramsExpected.MaximumMonthlyPercentageYield, params.MaximumMonthlyPercentageYield)
	s.Require().Equal(paramsExpected.InvestorsPreseedPercentOfTotalSupply, params.InvestorsPreseedPercentOfTotalSupply)
	s.Require().Equal(paramsExpected.EmissionEnabled, params.EmissionEnabled)
}
