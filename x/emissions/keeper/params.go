package keeper

import (
	"context"
	"encoding/binary"
	"errors"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/codec"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

func NewParamsKeeper(cdc codec.BinaryCodec, sb *collections.SchemaBuilder) *ParamsKeeper {
	return &ParamsKeeper{
		params: collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
	}
}

type ParamsKeeper struct {
	params collections.Item[types.Params]
}

func (k *ParamsKeeper) SetParams(ctx context.Context, params types.Params) error {
	if err := params.Validate(); err != nil {
		return errorsmod.Wrap(err, "failed to set params")
	}
	return k.params.Set(ctx, params)
}

func (k *ParamsKeeper) GetParams(ctx context.Context) (types.Params, error) {
	ret, err := k.params.Get(ctx)
	if errors.Is(err, collections.ErrNotFound) {
		return types.DefaultParams(), nil
	} else if err != nil {
		return types.Params{}, errorsmod.Wrap(err, "error getting params")
	}
	return ret, nil
}

// Convert pagination.key from []bytes to uint64, if pagination is nil or [], len = 0
// Get the limit from the pagination request, within acceptable bounds and defaulting as necessary
func (k *ParamsKeeper) CalcAppropriatePaginationForUint64Cursor(ctx context.Context, pagination *types.SimpleCursorPaginationRequest) (uint64, uint64, error) {
	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return uint64(0), uint64(0), errorsmod.Wrap(err, "error getting params")
	}
	limit := moduleParams.DefaultPageLimit
	cursor := uint64(0)

	if pagination != nil {
		if len(pagination.Key) > 0 {
			if len(pagination.Key) != 8 {
				return uint64(0), uint64(0), errorsmod.Wrapf(
					sdkerrors.ErrInvalidRequest,
					"invalid pagination key length %d, expected 8",
					len(pagination.Key),
				)
			}
			cursor = binary.BigEndian.Uint64(pagination.Key)
		}
		if pagination.Limit > 0 {
			limit = pagination.Limit
		}
		if limit > moduleParams.MaxPageLimit {
			limit = moduleParams.MaxPageLimit
		}
	}

	return limit, cursor, nil
}
