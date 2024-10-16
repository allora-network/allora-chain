package msgserver

import (
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

func checkInputLength(maxSerializedMsgLength int64, msg proto.Message) error {
	size := proto.Size(msg)

	// Check the length of the serialized message
	if int64(size) > maxSerializedMsgLength {
		return types.ErrQueryTooLarge
	}

	return nil
}
