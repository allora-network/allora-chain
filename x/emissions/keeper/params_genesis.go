package keeper

import (
	"context"

	"cosmossdk.io/errors"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (k *ParamsKeeper) InitGenesis(ctx context.Context, data *types.GenesisState) error {
	if err := k.SetParams(ctx, data.Params); err != nil {
		return errors.Wrap(err, "error setting params")
	}
	return nil
}

func (k *ParamsKeeper) ExportGenesis(ctx context.Context, data *types.GenesisState) error {
	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get module params")
	}
	data.Params = moduleParams
	return nil
}
