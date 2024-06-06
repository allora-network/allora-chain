package stress_test

import (
	"math/rand"
	"strconv"

	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// register alice as a reputer in topic 1, then check success
func RegisterReputerForTopic(
	m testCommon.TestConfig,
	reputer testCommon.NameAccountAndAddress,
	topicId uint64,
) error {

	registerReputerRequest := &emissionstypes.MsgRegister{
		Sender:       reputer.Aa.Addr,
		Owner:        reputer.Aa.Addr,
		LibP2PKey:    "reputerkey" + strconv.Itoa(rand.Intn(10000000000)),
		MultiAddress: "reputermultiaddress",
		TopicId:      topicId,
		IsReputer:    true,
	}
	txResp, err := m.Client.BroadcastTx(m.Ctx, reputer.Aa.Acc, registerReputerRequest)
	if err != nil {
		return err
	}
	_, err = m.Client.WaitForTx(m.Ctx, txResp.TxHash)
	m.T.Log("Create Topic TxHash", txResp.TxHash)
	if err != nil {
		return err
	}
	registerAliceResponse := &emissionstypes.MsgRegisterResponse{}
	err = txResp.Decode(registerAliceResponse)
	if err != nil {
		return err
	}
	incrementCountReputers()
	return nil
}

// register bob as worker in topic 1, then check sucess
func RegisterWorkerForTopic(
	m testCommon.TestConfig,
	worker testCommon.NameAccountAndAddress,
	topicId uint64,
) error {
	registerWorkerRequest := &emissionstypes.MsgRegister{
		Sender:       worker.Aa.Addr,
		Owner:        worker.Aa.Addr,
		LibP2PKey:    "workerkey" + strconv.Itoa(rand.Intn(10000000000)),
		MultiAddress: "workermultiaddress",
		TopicId:      topicId,
		IsReputer:    false,
	}
	txResp, err := m.Client.BroadcastTx(m.Ctx, worker.Aa.Acc, registerWorkerRequest)
	if err != nil {
		return err
	}
	_, err = m.Client.WaitForTx(m.Ctx, txResp.TxHash)
	if err != nil {
		return err
	}
	registerBobResponse := &emissionstypes.MsgRegisterResponse{}
	err = txResp.Decode(registerBobResponse)
	if err != nil {
		return err
	}
	incrementCountWorkers()
	return nil
}
