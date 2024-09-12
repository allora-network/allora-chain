package stress_test

import (
	"context"

	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// register alice as a reputer in topic 1, then check success
func RegisterReputerForTopic(
	m testCommon.TestConfig,
	reputer NameAccountAndAddress,
	topicId uint64,
) error {
	ctx := context.Background()

	registerReputerRequest := &emissionstypes.RegisterRequest{
		Sender:    reputer.aa.addr,
		Owner:     reputer.aa.addr,
		TopicId:   topicId,
		IsReputer: true,
	}
	txResp, err := m.Client.BroadcastTx(ctx, reputer.aa.acc, registerReputerRequest)
	if err != nil {
		return err
	}
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	if err != nil {
		return err
	}
	registerAliceResponse := &emissionstypes.RegisterResponse{}
	err = txResp.Decode(registerAliceResponse)
	if err != nil {
		return err
	}
	incrementCountReputers()
	return nil
}

// register bob as worker in topic 1, then check success
func RegisterWorkerForTopic(
	m testCommon.TestConfig,
	worker NameAccountAndAddress,
	topicId uint64,
) error {
	ctx := context.Background()
	registerWorkerRequest := &emissionstypes.RegisterRequest{
		Sender:    worker.aa.addr,
		Owner:     worker.aa.addr,
		TopicId:   topicId,
		IsReputer: false,
	}
	txResp, err := m.Client.BroadcastTx(ctx, worker.aa.acc, registerWorkerRequest)
	if err != nil {
		return err
	}
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	if err != nil {
		return err
	}
	registerBobResponse := &emissionstypes.RegisterResponse{}
	err = txResp.Decode(registerBobResponse)
	if err != nil {
		return err
	}
	incrementCountWorkers()
	return nil
}
