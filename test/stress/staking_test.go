package stress_test

import (
	cosmosMath "cosmossdk.io/math"
	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"
)

// broadcast tx to register reputer in topic, then check success
func stakeReputer(
	m testCommon.TestConfig,
	topicId uint64,
	address string,
	account cosmosaccount.Account,
	stakeToAdd uint64,
) error {
	addStake := &emissionstypes.MsgAddStake{
		Sender:  address,
		TopicId: topicId,
		Amount:  cosmosMath.NewIntFromUint64(stakeToAdd),
	}
	txResp, err := m.Client.BroadcastTx(m.Ctx, account, addStake)
	if err != nil {
		return err
	}
	_, err = m.Client.WaitForTx(m.Ctx, txResp.TxHash)
	if err != nil {
		return err
	}

	return nil
}
