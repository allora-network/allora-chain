package app

import (
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// measuredFeePathSimGasGap is the worst-case fee-path cost that simulation
// cannot measure, derived from mainnet: the simulated vs delivered gas gap of
// a single-signer MsgSend (32,682) plus the first-time tip recipient account
// creation (12,953). The integration harness measures a much smaller gap
// (smaller amounts mean fewer bytes written), so tuning the compensation down
// to satisfy CI silently re-breaks gas estimation on mainnet. Lowering the
// compensation below this floor requires re-deriving the value from mainnet
// measurements first.
const measuredFeePathSimGasGap uint64 = 45_635

// TestFeePathSimGasCompensationCoversMeasuredGap guards the compensation
// against downward retuning: it must cover the measured worst-case fee-path
// cost, not just the gap observable in the test harness.
func TestFeePathSimGasCompensationCoversMeasuredGap(t *testing.T) {
	require.GreaterOrEqual(t, FeePathSimGasCompensation, measuredFeePathSimGasGap)
}

func TestFeePathSimGasCompensationDecorator(t *testing.T) {
	decorator := newFeePathSimGasCompensationDecorator()

	tests := []struct {
		name         string
		simulate     bool
		success      bool
		wantConsumed uint64
	}{
		{name: "simulate after success", simulate: true, success: true, wantConsumed: FeePathSimGasCompensation},
		{name: "simulate after failure", simulate: true, success: false, wantConsumed: FeePathSimGasCompensation},
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
