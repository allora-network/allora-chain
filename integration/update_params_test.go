package integration_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/stretchr/testify/require"
)

// test that we can create topics and that the resultant topics are what we asked for
func UpdateParams(m TestMetadata) emissionstypes.Params {
	newSharpness := alloraMath.NewDecFinite(1, 999999)
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

	updatedParams := GetEmissionsParams(m)
	require.NoError(m.t, err)
	require.Equal(m.t, updatedParams.Sharpness.String(), newSharpness.String())
	return updatedParams
}
