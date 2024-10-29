package newstress_test

import (
	"context"
	"fmt"

	testcommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	proto "github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
)

// registerWorkers registers numWorkers as workers in topicId
func registerWorkers(
	m *testcommon.TestConfig,
	actors []Actor,
	topicId uint64,
	data *SimulationData,
	numWorkers int,
) error {
	workerRequests := make([]proto.Message, numWorkers)
	signers := make([]signingtypes.SignatureV2, numWorkers)
	for i := 0; i < numWorkers; i++ {
		worker := actors[i]
		workerRequest := &emissionstypes.RegisterRequest{
			Sender:    worker.addr,
			Owner:     worker.addr,
			IsReputer: false,
			TopicId:   topicId,
		}
		src, err := workerRequest.XXX_Marshal(nil, true)
		require.NoError(m.T, err, "Marshall worker request should not return an error")

		workerRequestSignature, pubKey, err := m.Client.Context().Keyring.Sign(worker.name, src, signing.SignMode_SIGN_MODE_DIRECT)
		require.NoError(m.T, err, "Sign should not return an error")

		signers[i] = signingtypes.SignatureV2{
			PubKey: pubKey,
			Data: &signingtypes.SingleSignatureData{
				SignMode:  signingtypes.SignMode_SIGN_MODE_DIRECT,
				Signature: workerRequestSignature,
			},
		}

		workerRequests[i] = workerRequest
		fmt.Println("Registering worker: ", worker.name, "in topic: ", topicId, "with address: ", worker.addr)
		data.addWorkerRegistration(topicId, worker)
	}

	fmt.Println("Building tx with workers in topic: ", topicId)
	builder := m.Client.Context().TxConfig.NewTxBuilder()
	builder.SetMsgs(workerRequests...)
	builder.SetSignatures(signers...)
	tx := builder.GetTx()
	fmt.Println("Encoding tx with workers in topic: ", topicId)
	txBytes, err := m.Client.Context().TxConfig.TxEncoder()(tx)
	require.NoError(m.T, err, "TxEncoder should not return an error")
	fmt.Println("Broadcasting tx with workers in topic: ", topicId)

	txResp, err := m.Client.Context().BroadcastTxSync(txBytes)
	if err != nil {
		fmt.Println("Error broadcasting tx: ", err)
		return err
	}
	fmt.Printf("Transaction broadcast successful. TxHash: %s\n", txResp.TxHash)

	return nil
}

func registerReputers(
	m *testcommon.TestConfig,
	actors []Actor,
	topicId uint64,
	data *SimulationData,
	numReputers int,
) error {
	fmt.Println("Registering reputers in topic: ", topicId)
	ctx := context.Background()
	// Register each reputer in separate transactions
	for i := 0; i < numReputers; i++ {
		reputer := actors[i]
		request := &emissionstypes.RegisterRequest{
			Sender:    reputer.addr,
			Owner:     reputer.addr,
			IsReputer: true,
			TopicId:   topicId,
		}
		fmt.Println("Registering reputer: ", reputer.name, "in topic: ", topicId, "with address: ", reputer.addr)
		
		// Broadcast transaction with the reputer's account
		txResp, err := m.Client.BroadcastTx(ctx, reputer.acc, request)
		if err != nil {
			return err
		}
		_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
		if err != nil {
			return err
		}
		
		data.addReputerRegistration(topicId, reputer)
	}
	return nil
}