package migrations_test

import (
	"testing"

	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/mint/keeper"
	"github.com/allora-network/allora-chain/x/mint/migrations"
	mint "github.com/allora-network/allora-chain/x/mint/module"
	minttestutil "github.com/allora-network/allora-chain/x/mint/testutil"
	"github.com/allora-network/allora-chain/x/mint/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	storetypes "cosmossdk.io/store/types"
	cosmostestutil "github.com/cosmos/cosmos-sdk/testutil"
)

type MigrationTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	mintKeeper      keeper.Keeper
	ctx             sdk.Context
	stakingKeeper   *minttestutil.MockStakingKeeper
	bankKeeper      *minttestutil.MockBankKeeper
	emissionsKeeper *minttestutil.MockEmissionsKeeper
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(MigrationTestSuite))
}

func (s *MigrationTestSuite) SetupTest() {
	encCfg := moduletestutil.MakeTestEncodingConfig(mint.AppModuleBasic{})
	key := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(key)
	testCtx := cosmostestutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	s.ctx = testCtx.Ctx

	// gomock initializations
	s.ctrl = gomock.NewController(s.T())
	accountKeeper := minttestutil.NewMockAccountKeeper(s.ctrl)
	bankKeeper := minttestutil.NewMockBankKeeper(s.ctrl)
	stakingKeeper := minttestutil.NewMockStakingKeeper(s.ctrl)
	emissionsKeeper := minttestutil.NewMockEmissionsKeeper(s.ctrl)

	accountKeeper.EXPECT().GetModuleAddress(types.ModuleName).Return(sdk.AccAddress{})

	s.mintKeeper = keeper.NewKeeper(
		encCfg.Codec,
		storeService,
		stakingKeeper,
		accountKeeper,
		bankKeeper,
		emissionsKeeper,
		authtypes.FeeCollectorName,
	)
	s.stakingKeeper = stakingKeeper
	s.bankKeeper = bankKeeper
	s.emissionsKeeper = emissionsKeeper
}

func (s *MigrationTestSuite) TestMigrate() {

	storageService := s.mintKeeper.GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.ctx))
	cdc := s.mintKeeper.GetBinaryCodec()

	paramsOld := OldParams{
		MintDenom:                              "test",
		MaxSupply:                              cosmosMath.NewInt(120),
		FEmission:                              cosmosMath.LegacyMustNewDecFromStr("0.1337"),
		OneMonthSmoothingDegree:                cosmosMath.LegacyMustNewDecFromStr("0.1337"),
		EcosystemTreasuryPercentOfTotalSupply:  cosmosMath.LegacyMustNewDecFromStr("0.1337"),
		FoundationTreasuryPercentOfTotalSupply: cosmosMath.LegacyMustNewDecFromStr("0.1337"),
		ParticipantsPercentOfTotalSupply:       cosmosMath.LegacyMustNewDecFromStr("0.1337"),
		InvestorsPercentOfTotalSupply:          cosmosMath.LegacyMustNewDecFromStr("0.1337"),
		TeamPercentOfTotalSupply:               cosmosMath.LegacyMustNewDecFromStr("0.1337"),
		MaximumMonthlyPercentageYield:          cosmosMath.LegacyMustNewDecFromStr("0.1337"),
	}

	store.Set(types.ParamsKey, cdc.MustMarshal(&paramsOld))

	// Run migration
	err := migrations.V1ToV2(s.ctx, s.mintKeeper)
	s.Require().NoError(err)

	defaultParams := types.DefaultParams()
	paramsExpected := types.Params{
		MintDenom:                              paramsOld.MintDenom,
		MaxSupply:                              paramsOld.MaxSupply,
		FEmission:                              paramsOld.FEmission,
		OneMonthSmoothingDegree:                paramsOld.OneMonthSmoothingDegree,
		EcosystemTreasuryPercentOfTotalSupply:  paramsOld.EcosystemTreasuryPercentOfTotalSupply,
		FoundationTreasuryPercentOfTotalSupply: paramsOld.FoundationTreasuryPercentOfTotalSupply,
		ParticipantsPercentOfTotalSupply:       paramsOld.ParticipantsPercentOfTotalSupply,
		InvestorsPercentOfTotalSupply:          paramsOld.InvestorsPercentOfTotalSupply,
		TeamPercentOfTotalSupply:               paramsOld.TeamPercentOfTotalSupply,
		MaximumMonthlyPercentageYield:          paramsOld.MaximumMonthlyPercentageYield,
		InvestorsPreseedPercentOfTotalSupply:   defaultParams.InvestorsPreseedPercentOfTotalSupply,
	}

	params, err := s.mintKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(paramsExpected.MintDenom, params.MintDenom)
	s.Require().True(paramsExpected.MaxSupply.Equal(params.MaxSupply), "%s != %s", paramsExpected.MaxSupply, params.MaxSupply)
	s.Require().True(paramsExpected.FEmission.Equal(params.FEmission), "%s != %s", paramsExpected.FEmission, params.FEmission)
	s.Require().True(paramsExpected.OneMonthSmoothingDegree.Equal(params.OneMonthSmoothingDegree), "%s != %s", paramsExpected.OneMonthSmoothingDegree, params.OneMonthSmoothingDegree)
	s.Require().True(paramsExpected.EcosystemTreasuryPercentOfTotalSupply.Equal(params.EcosystemTreasuryPercentOfTotalSupply), "%s != %s", paramsExpected.EcosystemTreasuryPercentOfTotalSupply, params.EcosystemTreasuryPercentOfTotalSupply)
	s.Require().True(paramsExpected.FoundationTreasuryPercentOfTotalSupply.Equal(params.FoundationTreasuryPercentOfTotalSupply), "%s != %s", paramsExpected.FoundationTreasuryPercentOfTotalSupply, params.FoundationTreasuryPercentOfTotalSupply)
	s.Require().True(paramsExpected.ParticipantsPercentOfTotalSupply.Equal(params.ParticipantsPercentOfTotalSupply), "%s != %s", paramsExpected.ParticipantsPercentOfTotalSupply, params.ParticipantsPercentOfTotalSupply)
	s.Require().True(paramsExpected.InvestorsPercentOfTotalSupply.Equal(params.InvestorsPercentOfTotalSupply), "%s != %s", paramsExpected.InvestorsPercentOfTotalSupply, params.InvestorsPercentOfTotalSupply)
	s.Require().True(paramsExpected.TeamPercentOfTotalSupply.Equal(params.TeamPercentOfTotalSupply), "%s != %s", paramsExpected.TeamPercentOfTotalSupply, params.TeamPercentOfTotalSupply)
	s.Require().True(paramsExpected.MaximumMonthlyPercentageYield.Equal(params.MaximumMonthlyPercentageYield), "%s != %s", paramsExpected.MaximumMonthlyPercentageYield, params.MaximumMonthlyPercentageYield)
	s.Require().True(paramsExpected.InvestorsPreseedPercentOfTotalSupply.Equal(params.InvestorsPreseedPercentOfTotalSupply), "%s != %s", paramsExpected.InvestorsPreseedPercentOfTotalSupply, params.InvestorsPreseedPercentOfTotalSupply)
	s.Require().Equal(paramsExpected, params)

}
