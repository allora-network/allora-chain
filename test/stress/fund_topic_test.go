package stress_test

import (
	cosmosMath "cosmossdk.io/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"
)

func FundTopic(m TestMetadata, topicId uint64, address string, account cosmosaccount.Account, amount int64) error {
	txResp, err := m.n.Client.BroadcastTx(
		m.ctx,
		account,
		&emissionstypes.MsgFundTopic{
			Sender:  address,
			TopicId: topicId,
			Amount:  cosmosMath.NewInt(amount),
		},
	)
	if err != nil {
		return err
	}
	_, err = m.n.Client.WaitForTx(m.ctx, txResp.TxHash)
	if err != nil {
		return err
	}
	resp := &emissionstypes.MsgFundTopicResponse{}
	err = txResp.Decode(resp)
	if err != nil {
		return err
	}
	return nil
}
