package integration_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/stretchr/testify/require"
)

func checkIfAdmin(m TestMetadata, address string) bool {
	paramsReq := &emissionstypes.QueryIsWhitelistAdminRequest{
		Address: address,
	}
	p, err := m.n.QueryEmissions.IsWhitelistAdmin(
		m.ctx,
		paramsReq,
	)
	require.NoError(m.t, err)
	require.NotNil(m.t, p)
	return p.IsAdmin
}

// Test that whitelisted admin can successfully update params and others cannot
func UpdateParamsChecks(m TestMetadata) emissionstypes.Params {
	// Ensure Alice is in the whitelist and Bob is not
	require.True(m.t, checkIfAdmin(m, m.n.AliceAddr))
	require.False(m.t, checkIfAdmin(m, m.n.BobAddr))

	// Should succeed for Alice because she's a whitelist admin
	newSharpness := alloraMath.NewDecFinite(1, 99)
	input := []alloraMath.Dec{newSharpness}
	updateParamRequest := &emissionstypes.MsgUpdateParams{
		Sender: m.n.AliceAddr,
		Params: &emissionstypes.OptionalParams{
			Sharpness: input,
		},
	}
	txResp, err := m.n.Client.BroadcastTx(m.ctx, m.n.AliceAcc, updateParamRequest)
	require.NoError(m.t, err)
	_, err = m.n.Client.WaitForTx(m.ctx, txResp.TxHash)
	require.NoError(m.t, err)

	// Should fail for Bob because he's not a whitelist admin
	input = []alloraMath.Dec{alloraMath.NewDecFinite(1, 2)}
	updateParamRequest = &emissionstypes.MsgUpdateParams{
		Sender: m.n.BobAddr,
		Params: &emissionstypes.OptionalParams{
			Sharpness: input,
		},
	}
	txResp, err = m.n.Client.BroadcastTx(m.ctx, m.n.BobAcc, updateParamRequest)
	require.Error(m.t, err)
	// Check that error is due to Bob not being a whitelist admin
	require.Contains(m.t, err.Error(), "not whitelist admin")

	// Check that the sharpness was updated by Alice successfully
	updatedParams := GetEmissionsParams(m)
	require.Equal(m.t, updatedParams.Sharpness.String(), newSharpness.String())
	return updatedParams
}
