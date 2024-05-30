package msgserver

import (
	"context"

	"cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/app/params"
	"github.com/allora-network/allora-chain/x/emissions/types"
	mintTypes "github.com/allora-network/allora-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (ms msgServer) CreateNewTopic(
	ctx context.Context,
	msg *types.MsgCreateNewTopic,
) (*types.MsgCreateNewTopicResponse, error) {
	if err := msg.Validate(); err != nil {
		return nil, err
	}

	hasEnoughBal, fee, _ := ms.CheckAddressHasBalanceForTopicCreationFee(ctx, msg.Creator)
	if !hasEnoughBal {
		return nil, errors.Wrapf(
			sdkerrors.ErrInsufficientFunds,
			"sender has insufficient balance to cover topic creation fee",
		)
	}

	id, err := ms.k.GetNextTopicId(ctx)
	if err != nil {
		return nil, err
	}

	params, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	if msg.EpochLength < params.MinEpochLength {
		return nil, types.ErrTopicCadenceBelowMinimum
	}

	// Before creating topic, transfer fee amount from creator to ecosystem bucket
	err = ms.k.SendCoinsFromAccountToModule(
		ctx,
		msg.Creator,
		mintTypes.EcosystemModuleName,
		sdk.NewCoins(fee),
	)
	if err != nil {
		return nil, err
	}

	topic := types.Topic{
		Id:              id,
		Creator:         msg.Creator,
		Metadata:        msg.Metadata,
		LossLogic:       msg.LossLogic,
		LossMethod:      msg.LossMethod,
		InferenceLogic:  msg.InferenceLogic,
		InferenceMethod: msg.InferenceMethod,
		EpochLastEnded:  0,
		EpochLength:     msg.EpochLength,
		GroundTruthLag:  msg.GroundTruthLag,
		DefaultArg:      msg.DefaultArg,
		PNorm:           msg.PNorm,
		AlphaRegret:     msg.AlphaRegret,
		AllowNegative:   msg.AllowNegative,
	}
	_, err = ms.k.IncrementTopicId(ctx)
	if err != nil {
		return nil, err
	}
	if err := ms.k.SetTopic(ctx, id, topic); err != nil {
		return nil, err
	}
	// Rather than set latest weight-adjustment timestamp of a topic to 0
	// we do nothing, since no value in the map means zero

	return &types.MsgCreateNewTopicResponse{TopicId: id}, nil
}

func (ms msgServer) CheckAddressHasBalanceForTopicCreationFee(
	ctx context.Context,
	address string,
) (bool, sdk.Coin, error) {
	moduleParams, err := ms.k.GetParams(ctx)
	if err != nil {
		return false, sdk.Coin{}, err
	}
	fee := sdk.NewCoin(params.DefaultBondDenom, moduleParams.CreateTopicFee)
	accAddress, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return false, sdk.Coin{}, err
	}
	balance := ms.k.BankKeeper().GetBalance(ctx, accAddress, fee.Denom)
	return balance.IsGTE(fee), fee, nil
}
