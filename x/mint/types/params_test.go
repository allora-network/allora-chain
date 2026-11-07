package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/allora-network/allora-chain/x/mint/types"
)

func TestDefaultParamsHaveNoVestingDelay(t *testing.T) {
	params := types.DefaultParams()
	require.Equal(t, uint64(0), params.InvestorsTeamVestingDelayMonths)
	require.NoError(t, params.Validate())
}

func TestParamsValidateInvestorsTeamVestingDelayMonths(t *testing.T) {
	tests := []struct {
		name    string
		months  uint64
		wantErr bool
	}{
		{name: "zero", months: 0, wantErr: false},
		{name: "six months", months: 6, wantErr: false},
		{name: "at the cap", months: types.MaxInvestorsTeamVestingDelayMonths, wantErr: false},
		{name: "above the cap", months: types.MaxInvestorsTeamVestingDelayMonths + 1, wantErr: true},
		{name: "absurdly large", months: ^uint64(0), wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			params := types.DefaultParams()
			params.InvestorsTeamVestingDelayMonths = tc.months
			err := params.Validate()
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
