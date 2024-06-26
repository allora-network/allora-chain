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

var _ types.MsgServer = msgServer{}

// NewMsgServerImpl returns an implementation of the module MsgServer interface.
func NewMsgServerImpl(keeper keeper.Keeper) types.MsgServer {
	return &msgServer{k: keeper}
}

func checkInputLength(ctx context.Context, ms msgServer, msg proto.Message) error {
	params, err := ms.k.GetParams(ctx)
	if err != nil {
		return err
	}

	serializedMsg, err := proto.Marshal(msg)
	if err != nil {
		return types.ErrFailedToSerializePayload
	}

	// Check the length of the serialized message
	if int64(len(serializedMsg)) > params.MaxSerializedMsgLength {
		return types.ErrQueryTooLarge
	}

	return nil
}
