package stress_test

import (
	cosmosMath "cosmossdk.io/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"
)

// register alice as a reputer in topic 1, then check success
func StakeReputer(m TestMetadata, topicId uint64, address string, account cosmosaccount.Account, stakeToAdd uint64) error {
	addStake := &emissionstypes.MsgAddStake{
		Sender:  address,
		TopicId: topicId,
		Amount:  cosmosMath.NewUint(stakeToAdd),
	}
	txResp, err := m.n.Client.BroadcastTx(m.ctx, account, addStake)
	if err != nil {
		return err
	}
	_, err = m.n.Client.WaitForTx(m.ctx, txResp.TxHash)
	if err != nil {
		return err
	}

	return nil
}
