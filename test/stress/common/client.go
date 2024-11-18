package testcommon

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	cometrpc "github.com/cometbft/cometbft/rpc/client/http"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
)

var cdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

type StressClient struct {
	client *cometrpc.HTTP
}

var (
	clients    = make(map[string]*StressClient)
	clientsMux sync.RWMutex
)

func GetClient(rpcEndpoint string) (*StressClient, error) {
	clientsMux.RLock()
	if client, exists := clients[rpcEndpoint]; exists {
		clientsMux.RUnlock()
		return client, nil
	}
	clientsMux.RUnlock()

	// If client doesn't exist, acquire write lock and create it
	clientsMux.Lock()
	defer clientsMux.Unlock()

	// Double-check after acquiring write lock
	if client, exists := clients[rpcEndpoint]; exists {
		return client, nil
	}

	// Create new client
	cmtCli, err := cometrpc.New(rpcEndpoint, "/websocket")
	if err != nil {
		return nil, err
	}

	client := &StressClient{
		client: cmtCli,
	}
	clients[rpcEndpoint] = client
	return client, nil
}

func (b *StressClient) BroadcastTx(txBytes []byte) (*coretypes.ResultBroadcastTx, error) {
	ctx := context.Background()

	t := tmtypes.Tx(txBytes)
	res, err := b.client.BroadcastTxSync(ctx, t)
	if err != nil {
		return nil, err
	}

	if res.Code != 0 {
		return res, fmt.Errorf("broadcast error code %d: %s", res.Code, res.Log)
	}

	return res, nil
}

// SendTransactionViaRPC sends a transaction using the provided TransactionParams and sequence number.
func SendTransactionViaRPC(txParams TransactionParams, sequence uint64, msgs ...sdktypes.Msg) (*coretypes.ResultBroadcastTx, string, error) {
	encodingConfig := moduletestutil.MakeTestEncodingConfig()
	encodingConfig.Codec = cdc

	ctx := context.Background()

	// Build and sign the transaction
	txBytes, err := BuildAndSignTransaction(ctx, txParams, sequence, encodingConfig, msgs...)
	if err != nil {
		return nil, "", err
	}

	// Broadcast the transaction via RPC
	resp, err := Transaction(txBytes, txParams.NodeURL)
	if err != nil {
		return resp, string(txBytes), fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	return resp, string(txBytes), nil
}

// Transaction broadcasts the transaction bytes to the given RPC endpoint.
func Transaction(txBytes []byte, rpcEndpoint string) (*coretypes.ResultBroadcastTx, error) {
	client, err := GetClient(rpcEndpoint)
	if err != nil {
		return nil, err
	}

	return client.BroadcastTx(txBytes)
}

// Loop handles the main transaction broadcasting logic
func SendDataWithRetry(
	txParams TransactionParams,
	msgs ...sdktypes.Msg,
) (*coretypes.ResultBroadcastTx, uint64, error) {
	sequence := txParams.Sequence
	maxRetries := int64(3)
	retryDelay := int64(1)
	accountNumberMismatch := true

	for retryCount := int64(0); retryCount <= maxRetries; retryCount++ {
		currentSequence := sequence

		if accountNumberMismatch {
			seq, accnum, err := GetAccountInfo(txParams.AcctAddress, txParams.Config)
			if err != nil {
				delay := calculateLinearBackoffDelay(retryDelay, retryCount)
				time.Sleep(delay)
				continue
			}
			txParams.Sequence = seq
			txParams.AccNum = accnum
			accountNumberMismatch = false
		}

		resp, _, err := SendTransactionViaRPC(txParams, currentSequence, msgs...)
		if err != nil {
			// if sequence mismatch, handle it and retry
			if resp != nil && resp.Code == 32 {
				resp, newSeq, err := handleSequenceMismatch(txParams, sequence, err, msgs...)
				if err == nil {
					sequence = newSeq
					return resp, sequence, nil
				}
				continue
			}
			// if mempool is full, retry
			if strings.Contains(err.Error(), "mempool is full") {
				delay := calculateLinearBackoffDelay(retryDelay, retryCount)
				fmt.Printf("Mempool is full, retrying in %d seconds...\n", delay)
				time.Sleep(delay)
				continue
			}
			continue
		}
		sequence++
		return resp, sequence, nil
	}

	return nil, sequence, nil
}

// handleSequenceMismatch handles the case where a transaction fails due to sequence mismatch
func handleSequenceMismatch(txParams TransactionParams, sequence uint64, err error, msgs ...sdktypes.Msg) (*coretypes.ResultBroadcastTx, uint64, error) {
	expectedSeq, parseErr := ExtractExpectedSequence(err.Error())
	if parseErr != nil {
		fmt.Printf("Failed to parse expected sequence: %v\n", parseErr)
		return nil, sequence, nil
	}

	// fmt.Printf("Set sequence to expected value %d due to mismatch\n", expectedSeq)

	resp, _, err := SendTransactionViaRPC(txParams, expectedSeq, msgs...)
	if err != nil {
		return nil, expectedSeq, err
	}

	return resp, expectedSeq + 1, nil
}

// Function to extract the expected sequence number from the error message
func ExtractExpectedSequence(errMsg string) (uint64, error) {
	// Parse the error message to extract the expected sequence number
	// Example error message:
	// "account sequence mismatch, expected 42, got 41: incorrect account sequence"
	if !strings.Contains(errMsg, "account sequence mismatch") {
		return 0, fmt.Errorf("unexpected error message format: %s", errMsg)
	}

	index := strings.Index(errMsg, "expected ")
	if index == -1 {
		return 0, errors.New("expected sequence not found in error message")
	}

	start := index + len("expected ")
	rest := errMsg[start:]
	parts := strings.SplitN(rest, ",", 2)
	if len(parts) < 1 {
		return 0, errors.New("failed to split expected sequence from error message")
	}

	expectedSeqStr := strings.TrimSpace(parts[0])
	expectedSeq, err := strconv.ParseUint(expectedSeqStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse expected sequence number: %v", err)
	}

	return expectedSeq, nil
}

func calculateLinearBackoffDelay(baseDelay int64, retryCount int64) time.Duration {
	if retryCount == 0 {
		retryCount = 1
	}
	return time.Duration(baseDelay*retryCount) * time.Second
}
