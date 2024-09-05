package stress_test

import (
	"context"

	cosmosMath "cosmossdk.io/math"
	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// broadcast tx to register reputer in topic, then check success
func stakeReputer(
	m testCommon.TestConfig,
	topicId uint64,
	reputer NameAccountAndAddress,
	stakeToAdd uint64,
) error {
	addStake := &emissionstypes.MsgAddStake{
		Sender:  reputer.aa.addr,
		TopicId: topicId,
		Amount:  cosmosMath.NewIntFromUint64(stakeToAdd),
	}
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, reputer.aa.acc, addStake)
	if err != nil {
		return err
	}
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	if err != nil {
		return err
	}

	return nil
}
