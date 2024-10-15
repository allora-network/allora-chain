package msgserver

import (
	"context"

	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/gogo/protobuf/proto"
)

type msgServer struct {
	k keeper.Keeper
}

var _ types.MsgServiceServer = msgServer{k: keeper.Keeper{}}

// NewMsgServerImpl returns an implementation of the module MsgServer interface.
func NewMsgServerImpl(keeper keeper.Keeper) types.MsgServiceServer {
	return &msgServer{k: keeper}
}

func checkInputLength(ctx context.Context, ms msgServer, msg proto.Message) error {
	params, err := ms.k.GetParams(ctx)
	if err != nil {
		return err
	}

	size := proto.Size(msg)

	// Check the length of the serialized message
	if int64(size) > params.MaxSerializedMsgLength {
		return types.ErrQueryTooLarge
	}

	return nil
}
