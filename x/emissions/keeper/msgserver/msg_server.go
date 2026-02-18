package msgserver

import (
	"github.com/gogo/protobuf/proto"

	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

type msgServer struct {
	pk  *keeper.ParamsKeeper
	tk  *keeper.TopicKeeper
	wlk *keeper.WhitelistsKeeper
	rlk *keeper.ReputerLossKeeper
	bk  *keeper.BankingKeeper
	sk  *keeper.StakingKeeper
	sck *keeper.ScoresKeeper
	wk  *keeper.WorkerKeeper
	nk  *keeper.NonceKeeper
}

var _ types.MsgServiceServer = msgServer{}

// NewMsgServerImpl returns an implementation of the module MsgServer interface.
func NewMsgServerImpl(
	k keeper.Keeper,
) types.MsgServiceServer {
	return &msgServer{
		pk:  k.GetParamsKeeper(),
		tk:  k.GetTopicKeeper(),
		wlk: k.GetWhitelistsKeeper(),
		rlk: k.GetReputerLossKeeper(),
		bk:  k.GetBankingKeeper(),
		sk:  k.GetStakingKeeper(),
		sck: k.GetScoresKeeper(),
		wk:  k.GetWorkerKeeper(),
		nk:  k.GetNonceKeeper(),
	}
}

func checkInputLength(maxSerializedMsgLength int64, msg proto.Message) error {
	size := proto.Size(msg)

	// Check the length of the serialized message
	if int64(size) > maxSerializedMsgLength {
		return types.ErrQueryTooLarge
	}

	return nil
}
