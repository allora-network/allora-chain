package stress_test

import (
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
	"github.com/stretchr/testify/require"
)

// get the emissions params from outside the chain
func GetEmissionsParams(m TestMetadata) emissionstypes.Params {
	paramsReq := &emissionstypes.QueryParamsRequest{}
	p, err := m.n.Client.QueryEmissions().Params(
		m.ctx,
		paramsReq,
	)
	require.NoError(m.t, err)
	require.NotNil(m.t, p)
	return p.Params
}

// get the mint params from outside the chain
func GetMintParams(m TestMetadata) minttypes.Params {
	paramsReq := &minttypes.QueryParamsRequest{}
	p, err := m.n.Client.QueryMint().Params(
		m.ctx,
		paramsReq,
	)
	require.NoError(m.t, err)
	require.NotNil(m.t, p)
	return p.Params
}
