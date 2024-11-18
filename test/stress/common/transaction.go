package testcommon

import (
	"context"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
)

func BuildAndSignTransaction(
	ctx context.Context,
	txParams TransactionParams,
	sequence uint64,
	encodingConfig moduletestutil.TestEncodingConfig,
	msgs ...sdktypes.Msg,
) ([]byte, error) {
	// Create a new TxBuilder
	txBuilder := encodingConfig.TxConfig.NewTxBuilder()

	var memo string

	// Construct the message based on the message type
	var err error

	// Set the message and other transaction parameters
	if err := txBuilder.SetMsgs(msgs...); err != nil {
		return nil, err
	}

	// Estimate gas limit
	totalTxSize := 0
	for _, msg := range msgs {
		totalTxSize += len(msg.String())
	}
	gasLimit := uint64(int64(totalTxSize)*int64(txParams.Config.GasPerByte) + int64(txParams.Config.BaseGas))
	txBuilder.SetGasLimit(gasLimit)

	// Calculate fee
	gasPrice := sdk.NewDecCoinFromDec(
		txParams.Config.Denom,
		sdkmath.LegacyNewDecWithPrec(int64(txParams.Config.GasLow), int64(txParams.Config.GasPrecision)),
	)
	feeAmount := gasPrice.Amount.MulInt64(int64(gasLimit)).RoundInt()
	feeCoin := sdk.NewCoin(txParams.Config.Denom, feeAmount)
	txBuilder.SetFeeAmount(sdk.NewCoins(feeCoin))

	// Set memo and timeout height
	txBuilder.SetMemo(memo)
	txBuilder.SetTimeoutHeight(0)

	// Set up signature
	sigV2 := signing.SignatureV2{
		PubKey:   txParams.PubKey,
		Sequence: sequence,
		Data: &signing.SingleSignatureData{
			SignMode: signing.SignMode_SIGN_MODE_DIRECT,
		},
	}

	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return nil, err
	}

	signerData := authsigning.SignerData{
		ChainID:       txParams.ChainID,
		AccountNumber: txParams.AccNum,
		Sequence:      sequence,
	}

	// Sign the transaction with the private key
	sigV2, err = tx.SignWithPrivKey(
		ctx,
		signing.SignMode_SIGN_MODE_DIRECT,
		signerData,
		txBuilder,
		txParams.PrivKey,
		encodingConfig.TxConfig,
		sequence,
	)
	if err != nil {
		return nil, err
	}

	// Set the signed signature back to the txBuilder
	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return nil, err
	}

	// Encode the transaction
	txBytes, err := encodingConfig.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return nil, err
	}

	return txBytes, nil
}
