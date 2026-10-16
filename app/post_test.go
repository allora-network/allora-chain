package app

import (
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestFeePathSimGasCompensationDecorator(t *testing.T) {
	decorator := newFeePathSimGasCompensationDecorator()

	tests := []struct {
		name         string
		simulate     bool
		success      bool
		wantConsumed uint64
	}{
		{name: "simulate after success", simulate: true, success: true, wantConsumed: feePathSimGasCompensation},
		{name: "simulate after failure", simulate: true, success: false, wantConsumed: feePathSimGasCompensation},
		{name: "deliver after success", simulate: false, success: true, wantConsumed: 0},
		{name: "deliver after failure", simulate: false, success: false, wantConsumed: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := testutil.DefaultContext(
				storetypes.NewKVStoreKey("test"),
				storetypes.NewTransientStoreKey("test-transient"),
			).WithGasMeter(storetypes.NewGasMeter(1_000_000_000))

			nextCalled := false
			next := func(ctx sdk.Context, _ sdk.Tx, _, _ bool) (sdk.Context, error) {
				nextCalled = true
				return ctx, nil
			}

			newCtx, err := decorator.PostHandle(ctx, nil, tc.simulate, tc.success, next)

			require.NoError(t, err)
			require.True(t, nextCalled)
			require.Equal(t, tc.wantConsumed, newCtx.GasMeter().GasConsumed())
		})
	}
}
