package msgserver

import (
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

type msgServer struct {
	k keeper.Keeper
}

var _ types.MsgServer = msgServer{}

// NewMsgServerImpl returns an implementation of the module MsgServer interface.
func NewMsgServerImpl(keeper keeper.Keeper) types.MsgServer {
	return &msgServer{k: keeper}
}

func (ms msgServer) CheckInputLength(ctx context.Context, inputLength int) error {
	params, err := ms.k.GetParams(ctx)
	if err != nil {
		return err
	}

	// Check the length of the serialized message
	if int64(inputLength) > params.MaxSerializedMsgLength {
		return types.ErrQueryTooLarge
	}

	return nil
}
