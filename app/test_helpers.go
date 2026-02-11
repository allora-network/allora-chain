package app

import (
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
	ibctestingtypes "github.com/cosmos/ibc-go/v8/testing/types"
)

var _ ibctesting.TestingApp = &AlloraApp{} //nolint:exhaustruct

func (app *AlloraApp) GetBaseApp() *baseapp.BaseApp {
	return app.App.BaseApp
}

func (app *AlloraApp) GetStakingKeeper() ibctestingtypes.StakingKeeper {
	return app.StakingKeeper
}

func (app *AlloraApp) GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper {
	return app.ScopedIBCKeeper
}

func (app *AlloraApp) GetTxConfig() client.TxConfig {
	return app.txConfig
}

func (app *AlloraApp) AppCodec() codec.Codec {
	return app.appCodec
}
