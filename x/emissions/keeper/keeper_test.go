package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/core/header"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
	state "github.com/upshot-tech/protocol-state-machine-module"
	"github.com/upshot-tech/protocol-state-machine-module/keeper"
)

type KeeperTestSuite struct {
	suite.Suite

	ctx          sdk.Context
	authKeeper   keeper.AccountKeeper
	bankKeeper   keeper.BankKeeper
	upshotKeeper keeper.Keeper
	msgServer    state.MsgServer
	mockCtrl     *gomock.Controller
	key          *storetypes.KVStoreKey
}

func (s *KeeperTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey("upshot")
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithHeaderInfo(header.Info{Time: time.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig()
	addressCodec := address.NewBech32Codec("cosmos")
	ctrl := gomock.NewController(s.T())

	s.ctx = ctx
	s.upshotKeeper = keeper.NewKeeper(encCfg.Codec, addressCodec, storeService, s.authKeeper, s.bankKeeper)
	s.msgServer = keeper.NewMsgServerImpl(s.upshotKeeper)
	s.mockCtrl = ctrl
	s.key = key
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}
