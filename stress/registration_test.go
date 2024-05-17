package stress_test

import (
	"math/rand"
	"strconv"

	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"
)

// register alice as a reputer in topic 1, then check success
func RegisterReputerForTopic(m TestMetadata, address string, account cosmosaccount.Account, topicId uint64) error {

	registerReputerRequest := &emissionstypes.MsgRegister{
		Sender:       address,
		Owner:        address,
		LibP2PKey:    "reputerkey" + strconv.Itoa(rand.Intn(10000000000)),
		MultiAddress: "reputermultiaddress",
		TopicId:      1,
		IsReputer:    true,
	}
	txResp, err := m.n.Client.BroadcastTx(m.ctx, account, registerReputerRequest)
	if err != nil {
		return err
	}
	_, err = m.n.Client.WaitForTx(m.ctx, txResp.TxHash)
	if err != nil {
		return err
	}
	registerAliceResponse := &emissionstypes.MsgRegisterResponse{}
	err = txResp.Decode(registerAliceResponse)
	if err != nil {
		return err
	}
	return nil
}

// register bob as worker in topic 1, then check sucess
func RegisterWorkerForTopic(m TestMetadata, address string, account cosmosaccount.Account, topicId uint64) error {
	registerWorkerRequest := &emissionstypes.MsgRegister{
		Sender:       address,
		Owner:        address,
		LibP2PKey:    "workerkey" + strconv.Itoa(rand.Intn(10000000000)),
		MultiAddress: "workermultiaddress",
		TopicId:      topicId,
		IsReputer:    false,
	}
	txResp, err := m.n.Client.BroadcastTx(m.ctx, account, registerWorkerRequest)
	if err != nil {
		return err
	}
	_, err = m.n.Client.WaitForTx(m.ctx, txResp.TxHash)
	if err != nil {
		return err
	}
	registerBobResponse := &emissionstypes.MsgRegisterResponse{}
	err = txResp.Decode(registerBobResponse)
	if err != nil {
		return err
	}
	return nil
}
