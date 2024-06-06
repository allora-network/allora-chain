package simulation

import (
	"fmt"
	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"strconv"
	"time"
)

type ACTOR_NAME = string
type NameToAccountMap map[ACTOR_NAME]testCommon.AccountAndAddress

const secondsInAMonth = 2592000
const INFERER_TYPE = "Inferer"
const FORECASTER_TYPE = "Forecaster"
const REPUTER_TYPE = "Reputer"

// get the emissions params from outside the chain
func getEmissionsParams(m testCommon.TestConfig) (emissionstypes.Params, error) {
	paramsReq := &emissionstypes.QueryParamsRequest{}
	p, err := m.Client.QueryEmissions().Params(
		m.Ctx,
		paramsReq,
	)
	return p.Params, err
}

// get the mint params from outside the chain
func getMintParams(m testCommon.TestConfig) (minttypes.Params, error) {
	paramsReq := &minttypes.QueryParamsRequest{}
	p, err := m.Client.QueryMint().Params(
		m.Ctx,
		paramsReq,
	)
	return p.Params, err
}

func getActorsAccountName(actorType string, seed int, actorIndex int) string {
	return "simulation" + strconv.Itoa(seed) + "_" + actorType + strconv.Itoa(actorIndex)
}

// return the approximate block time in seconds
func getApproximateBlockTimeSeconds(m testCommon.TestConfig) time.Duration {
	emissionsParams, err := getEmissionsParams(m)
	if err != nil {
		return time.Duration(0) * time.Second
	}
	blocksPerMonth := emissionsParams.GetBlocksPerMonth()
	return time.Duration(secondsInAMonth/blocksPerMonth) * time.Second
}

// get the token holdings of an address from the bank module
func getAccountBalance(
	m testCommon.TestConfig,
	queryClient banktypes.QueryClient,
	address string,
) (*sdktypes.Coin, error) {
	req := &banktypes.QueryAllBalancesRequest{
		Address:    address,
		Pagination: &query.PageRequest{Limit: 1},
	}

	res, err := queryClient.AllBalances(m.Ctx, req)
	if err != nil {
		return nil, err
	}

	if len(res.Balances) > 0 {
		return &res.Balances[0], nil
	}
	return nil, fmt.Errorf("no balance found for address: %s", address)
}

// get the token holdings of an address from the bank module
func getMultiAccountsBalance(
	m testCommon.TestConfig,
	queryClient banktypes.QueryClient,
	address string,
) (*sdktypes.Coin, error) {
	req := &banktypes.QueryAllBalancesRequest{
		Address:    address,
		Pagination: &query.PageRequest{Limit: 1},
	}

	res, err := queryClient.AllBalances(m.Ctx, req)
	if err != nil {
		return nil, err
	}

	if len(res.Balances) > 0 {
		return &res.Balances[0], nil
	}
	return nil, fmt.Errorf("no balance found for address: %s", address)
}

func getNetworkInferencesAtBlock(
	m testCommon.TestConfig,
	topicId uint64,
	blockHeightLastInference,
	blockHeightLastReward int64,
) *emissionstypes.ValueBundle {
	query := &emissionstypes.QueryNetworkInferencesAtBlockRequest{
		TopicId:                  topicId,
		BlockHeightLastInference: blockHeightLastInference,
		BlockHeightLastReward:    blockHeightLastReward,
	}
	txResp, err := m.Client.QueryEmissions().GetNetworkInferencesAtBlock(m.Ctx, query)
	if err != nil {
		m.T.Log("Error query for getting network inferences at block: ", err)
		return &emissionstypes.ValueBundle{}
	}

	return txResp.NetworkInferences
}
func getNetworkLossBundleAtBlock(
	m testCommon.TestConfig,
	topicId uint64,
	blockHeight int64,
) *emissionstypes.ValueBundle {
	query := &emissionstypes.QueryNetworkLossBundleAtBlockRequest{
		TopicId:     topicId,
		BlockHeight: blockHeight,
	}
	txResp, err := m.Client.QueryEmissions().GetNetworkLossBundleAtBlock(m.Ctx, query)
	if err != nil {
		m.T.Log("Error query for getting network inferences at block: ", err)
		return &emissionstypes.ValueBundle{}
	}

	return txResp.LossBundle
}
