package app

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	feemarketpost "github.com/skip-mev/feemarket/x/feemarket/post"
)

// PostHandlerOptions are the options required for constructing a FeeMarket PostHandler.
type PostHandlerOptions struct {
	AccountKeeper   feemarketpost.AccountKeeper
	BankKeeper      feemarketpost.BankKeeper
	FeeMarketKeeper feemarketpost.FeeMarketKeeper
}

// FeePathSimGasCompensation is the gas charged in simulate mode to account
// for the fee market fee path, which simulation cannot measure. During
// simulation the fee market runs its fee handling on zero-value dummy coins,
// so the module-level bank sends it performs at deliver (the ante escrow, the
// fee deduction to the fee collector and the tip payout to the block
// proposer) plus their events consume almost no gas, while the feemarket post
// handler only compensates a single bank send via BankSendGasConsumption.
// Charging this amount keeps gas estimation derived from simulation at or
// above the real execution cost of a transaction.
//
// The value is the measured difference between simulated and delivered gas
// of a single-signer MsgSend on mainnet (32,682) plus the measured cost of
// creating a first-time tip recipient account (12,953), rounded up. The gas
// meter is infinite in simulate mode, so charging it can never abort a
// simulation. TestFeePathSimGasCompensationCoversMeasuredGap pins the
// measured minimum: if the fee market fee path changes, re-derive the value
// from measurements rather than from the test harness, whose gap is smaller.
const FeePathSimGasCompensation uint64 = 46_000

// feePathSimGasCompensationDecorator charges FeePathSimGasCompensation gas in
// simulate mode so that simulated gas covers the deliver-time cost of the fee
// market fee path. Outside simulate mode it is a no-op: deliver-time gas and
// therefore consensus are unchanged.
type feePathSimGasCompensationDecorator struct{}

// newFeePathSimGasCompensationDecorator returns a new fee path simulation gas
// compensation decorator.
func newFeePathSimGasCompensationDecorator() feePathSimGasCompensationDecorator {
	return feePathSimGasCompensationDecorator{}
}

// PostHandle charges the compensation gas when simulating and always calls
// the next post handler.
func (feePathSimGasCompensationDecorator) PostHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate, success bool,
	next sdk.PostHandler,
) (sdk.Context, error) {
	if simulate {
		ctx.GasMeter().ConsumeGas(FeePathSimGasCompensation, "fee market fee path simulation gas compensation")
	}

	return next(ctx, tx, simulate, success)
}

// NewPostHandler returns a PostHandler chain with the fee deduct decorator.
func NewPostHandler(options PostHandlerOptions) (sdk.PostHandler, error) {
	if !UseFeeMarketDecorator {
		return nil, nil //nolint:nilnil
	}

	if options.AccountKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "account keeper is required for post builder")
	}

	if options.BankKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "bank keeper is required for post builder")
	}

	if options.FeeMarketKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "feemarket keeper is required for post builder")
	}

	postDecorators := []sdk.PostDecorator{
		feemarketpost.NewFeeMarketDeductDecorator(
			options.AccountKeeper,
			options.BankKeeper,
			options.FeeMarketKeeper,
		),
		newFeePathSimGasCompensationDecorator(),
	}

	return sdk.ChainPostDecorators(postDecorators...), nil
}
