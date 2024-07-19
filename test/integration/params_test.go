package integration_test

import (
	"context"

	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
	"github.com/stretchr/testify/require"
)

// get the emissions params from outside the chain
func GetEmissionsParams(m testCommon.TestConfig) emissionstypes.Params {
	ctx := context.Background()
	paramsReq := &emissionstypes.QueryParamsRequest{}
	p, err := m.Client.QueryEmissions().Params(
		ctx,
		paramsReq,
	)
	require.NoError(m.T, err)
	require.NotNil(m.T, p)
	return p.Params
}

// get the mint params from outside the chain
func GetMintParams(m testCommon.TestConfig) minttypes.Params {
	ctx := context.Background()
	paramsReq := &minttypes.QueryParamsRequest{}
	p, err := m.Client.QueryMint().Params(
		ctx,
		paramsReq,
	)
	require.NoError(m.T, err)
	require.NotNil(m.T, p)
	return p.Params
}

// Test that we can get params from the chain
func GetParams(m testCommon.TestConfig) {
	m.T.Log("--- Test Getting Emissions Params ---")
	GetEmissionsParams(m)
	m.T.Log("--- Test Getting Mint Params ---")
	GetMintParams(m)
}
