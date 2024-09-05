package integration_test

import (
	"context"

	alloraMath "github.com/allora-network/allora-chain/math"
	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/stretchr/testify/require"
)

func checkIfAdmin(m testCommon.TestConfig, address string) bool {
	ctx := context.Background()
	paramsReq := &emissionstypes.QueryIsWhitelistAdminRequest{
		Address: address,
	}
	p, err := m.Client.QueryEmissions().IsWhitelistAdmin(
		ctx,
		paramsReq,
	)
	require.NoError(m.T, err)
	require.NotNil(m.T, p)
	return p.IsAdmin
}

// Test that whitelisted admin can successfully update params and others cannot
func UpdateParamsChecks(m testCommon.TestConfig) {
	ctx := context.Background()
	// Ensure Alice is in the whitelist and Bob is not
	require.True(m.T, checkIfAdmin(m, m.AliceAddr))
	require.False(m.T, checkIfAdmin(m, m.BobAddr))

	// Keep old params to revert back to
	oldParams := GetEmissionsParams(m)
	oldEpsilonReputer := oldParams.EpsilonReputer

	// Should succeed for Alice because she's a whitelist admin
	newEpsilonReputer := alloraMath.NewDecFinite(1, 99)
	input := []alloraMath.Dec{newEpsilonReputer}
	updateParamRequest := &emissionstypes.MsgServiceUpdateParamsRequest{
		Sender: m.AliceAddr,
		Params: &emissionstypes.OptionalParams{
			EpsilonReputer: input,
			// These are set for subsequent tests
			MaxTopReputersToReward: []uint64{24},
			MinEpochLength:         []int64{1},
		},
	}
	txResp, err := m.Client.BroadcastTx(ctx, m.AliceAcc, updateParamRequest)
	require.NoError(m.T, err)
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	require.NoError(m.T, err)

	// Should fail for Bob because he's not a whitelist admin
	input = []alloraMath.Dec{alloraMath.NewDecFinite(1, 2)}
	updateParamRequest = &emissionstypes.MsgServiceUpdateParamsRequest{
		Sender: m.BobAddr,
		Params: &emissionstypes.OptionalParams{
			EpsilonReputer: input,
		},
	}
	_, err = m.Client.BroadcastTx(ctx, m.BobAcc, updateParamRequest)
	require.Error(m.T, err)
	// Check that error is due to Bob not being a whitelist admin
	require.Contains(m.T, err.Error(), "not whitelist admin")

	// Check that the epsilon was updated by Alice successfully
	updatedParams := GetEmissionsParams(m)
	require.Equal(m.T, updatedParams.EpsilonReputer.String(), newEpsilonReputer.String())

	// Set the epsilon reputer back to the original value
	input = []alloraMath.Dec{oldEpsilonReputer}
	updateParamRequest = &emissionstypes.MsgServiceUpdateParamsRequest{
		Sender: m.AliceAddr,
		Params: &emissionstypes.OptionalParams{
			EpsilonReputer: input,
		},
	}
	txResp, err = m.Client.BroadcastTx(ctx, m.AliceAcc, updateParamRequest)
	require.NoError(m.T, err)
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	require.NoError(m.T, err)
}
