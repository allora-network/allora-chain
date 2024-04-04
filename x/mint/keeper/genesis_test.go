package keeper_test

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/x/mint/keeper"
	mint "github.com/allora-network/allora-chain/x/mint/module"
	minttestutil "github.com/allora-network/allora-chain/x/mint/testutil"
	"github.com/allora-network/allora-chain/x/mint/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
)

var minterAcc = authtypes.NewEmptyModuleAccount(types.ModuleName, authtypes.Minter)

type GenesisTestSuite struct {
	suite.Suite

	sdkCtx        sdk.Context
	keeper        keeper.Keeper
	cdc           codec.BinaryCodec
	accountKeeper types.AccountKeeper
	key           *storetypes.KVStoreKey
}

func TestGenesisTestSuite(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}

func (s *GenesisTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	encCfg := moduletestutil.MakeTestEncodingConfig(mint.AppModuleBasic{})

	// gomock initializations
	ctrl := gomock.NewController(s.T())
	s.cdc = codec.NewProtoCodec(encCfg.InterfaceRegistry)
	s.sdkCtx = testCtx.Ctx
	s.key = key

	stakingKeeper := minttestutil.NewMockStakingKeeper(ctrl)
	accountKeeper := minttestutil.NewMockAccountKeeper(ctrl)
	bankKeeper := minttestutil.NewMockBankKeeper(ctrl)
	emissionsKeeper := minttestutil.NewMockEmissionsKeeper(ctrl)
	s.accountKeeper = accountKeeper
	accountKeeper.EXPECT().GetModuleAddress(minterAcc.Name).Return(minterAcc.GetAddress())
	accountKeeper.EXPECT().GetModuleAccount(s.sdkCtx, minterAcc.Name).Return(minterAcc)

	s.keeper = keeper.NewKeeper(s.cdc, runtime.NewKVStoreService(key), stakingKeeper, accountKeeper, bankKeeper, emissionsKeeper, "", "")
}

func (s *GenesisTestSuite) TestImportExportGenesis() {
	genesisState := types.DefaultGenesisState()
	maxSupply, ok := math.NewIntFromString("1000000000000000000000000000")
	if !ok {
		panic("invalid number")
	}
	genesisState.Params = types.NewParams(
		"testDenom",
		uint64(60/5*60*24*30),
		uint64(2),
		maxSupply,
		math.NewInt(15),
		math.NewInt(1000),
		math.NewInt(1),
		math.NewInt(100),
		math.NewInt(3675),
		math.NewInt(10000),
	)
	genesisState.PreviousRewardEmissionPerUnitStakedToken = types.DefaultPreviousRewardEmissionPerUnitStakedToken()
	genesisState.PreviousBlockEmission = types.DefaultPreviousBlockEmission()
	genesisState.EcosystemTokensMinted = types.DefaultEcosystemTokensMinted()

	s.keeper.InitGenesis(s.sdkCtx, s.accountKeeper, genesisState)

	invalidCtx := testutil.DefaultContextWithDB(s.T(), s.key, storetypes.NewTransientStoreKey("transient_test"))
	_, err := s.keeper.EcosystemTokensMinted.Get(invalidCtx.Ctx)
	s.Require().ErrorIs(err, collections.ErrNotFound)

	params, err := s.keeper.Params.Get(s.sdkCtx)
	s.Require().Equal(genesisState.Params, params)
	s.Require().NoError(err)

	previousRewardsN, err := s.keeper.PreviousRewardEmissionPerUnitStakedToken.Get(s.sdkCtx)
	s.Require().NoError(err)
	s.Require().True(genesisState.PreviousRewardEmissionPerUnitStakedToken.Equal(previousRewardsN))
	s.Require().NoError(err)

	ecosystemTokensMinted, err := s.keeper.EcosystemTokensMinted.Get(s.sdkCtx)
	s.Require().True(genesisState.EcosystemTokensMinted.Equal(ecosystemTokensMinted))
	s.Require().NoError(err)

	genesisState2 := s.keeper.ExportGenesis(s.sdkCtx)
	// got to check the fields are equal one by one because the
	// bigint params screw .Equal up
	s.Require().Equal(genesisState.Params, genesisState2.Params)
	s.Require().True(genesisState.PreviousRewardEmissionPerUnitStakedToken.Equal(genesisState2.PreviousRewardEmissionPerUnitStakedToken))
	s.Require().True(genesisState.EcosystemTokensMinted.Equal(genesisState2.EcosystemTokensMinted))
}
