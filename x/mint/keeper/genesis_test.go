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
	genesisState.Minter = types.NewMinter(math.LegacyNewDecWithPrec(20, 2), math.LegacyNewDec(1))
	genesisState.Params = types.NewParams(
		"testDenom",
		math.LegacyNewDecWithPrec(15, 2),
		math.LegacyNewDecWithPrec(22, 2),
		math.LegacyNewDecWithPrec(9, 2),
		math.LegacyNewDecWithPrec(69, 2),
		uint64(60*60*8766/5),
		math.NewUintFromString("1000000000000000000000000000"),
		uint64(25246080),
		math.NewUintFromString("2831000000000000000000"),
	)
	genesisState.PreviousReward = types.DefaultPreviousReward()

	s.keeper.InitGenesis(s.sdkCtx, s.accountKeeper, genesisState)

	minter, err := s.keeper.Minter.Get(s.sdkCtx)
	s.Require().Equal(genesisState.Minter, minter)
	s.Require().NoError(err)

	invalidCtx := testutil.DefaultContextWithDB(s.T(), s.key, storetypes.NewTransientStoreKey("transient_test"))
	_, err = s.keeper.Minter.Get(invalidCtx.Ctx)
	s.Require().ErrorIs(err, collections.ErrNotFound)

	params, err := s.keeper.Params.Get(s.sdkCtx)
	s.Require().Equal(genesisState.Params, params)
	s.Require().NoError(err)

	previousRewards, err := s.keeper.PreviousReward.Get(s.sdkCtx)
	s.Require().True(genesisState.PreviousReward.Equal(previousRewards))
	s.Require().NoError(err)

	genesisState2 := s.keeper.ExportGenesis(s.sdkCtx)
	// got to check the fields are equal one by one because the
	// bigint params screw .Equal up
	s.Require().Equal(genesisState.Minter, genesisState2.Minter)
	s.Require().Equal(genesisState.Params, genesisState2.Params)
	s.Require().True(genesisState.PreviousReward.Equal(genesisState2.PreviousReward))
}
