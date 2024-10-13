package msgserver

import (
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

type msgServer struct {
	k keeper.Keeper
}

var _ types.MsgServiceServer = msgServer{k: keeper.Keeper{}}

// NewMsgServerImpl returns an implementation of the module MsgServer interface.
func NewMsgServerImpl(keeper keeper.Keeper) types.MsgServiceServer {
	return &msgServer{k: keeper}
}
