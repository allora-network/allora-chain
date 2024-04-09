package msgserver

import (
	"context"
	"fmt"

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

func (k msgServer) TestDecMessage(ctx context.Context, msg *types.DecMessage) (*types.DecMessage, error) {
	fmt.Println("TestDecMessage")
	fmt.Println(msg.Value.String())
	fmt.Println("")
	return msg, nil
}
