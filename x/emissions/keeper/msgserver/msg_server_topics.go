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

func (ms msgServer) CreateNewTopic(ctx context.Context, msg *types.MsgCreateNewTopic) (*types.MsgCreateNewTopicResponse, error) {
	// Check if the sender is in the topic creation whitelist
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}
	isTopicCreator, err := ms.k.IsInTopicCreationWhitelist(ctx, creator)
	if err != nil {
		return nil, err
	}
	if !isTopicCreator {
		return nil, types.ErrNotInTopicCreationWhitelist
	}

	hasEnoughBal, fee, _ := ms.CheckAddressHasBalanceForTopicCreationFee(ctx, creator)
	if !hasEnoughBal {
		return nil, errors.Wrapf(sdkerrors.ErrInsufficientFunds, "sender has insufficient balance to cover topic creation fee")
	}

	id, err := ms.k.GetNextTopicId(ctx)
	if err != nil {
		return nil, err
	}

	fastestCadence, err := ms.k.GetParamsMinEpochLength(ctx)
	if err != nil {
		return nil, err
	}
	if msg.EpochLength < fastestCadence {
		return nil, types.ErrTopicCadenceBelowMinimum
	}

	// Before creating topic, transfer fee amount from creator to ecosystem bucket
	err = ms.k.SendCoinsFromAccountToModule(ctx, creator, mintTypes.EcosystemModuleName, sdk.NewCoins(fee))
	if err != nil {
		return nil, err
	}

	topic := types.Topic{
		Id:               id,
		Creator:          creator.String(),
		Metadata:         msg.Metadata,
		LossLogic:        msg.LossLogic,
		LossMethod:       msg.LossMethod,
		InferenceLogic:   msg.InferenceLogic,
		InferenceMethod:  msg.InferenceMethod,
		EpochLastEnded:   0,
		EpochLength:      msg.EpochLength,
		GroundTruthLag:   msg.GroundTruthLag,
		DefaultArg:       msg.DefaultArg,
		Pnorm:            msg.Pnorm,
		AlphaRegret:      msg.AlphaRegret,
		PrewardReputer:   msg.PrewardReputer,
		PrewardInference: msg.PrewardInference,
		PrewardForecast:  msg.PrewardForecast,
		FTolerance:       msg.FTolerance,
	}
	_, err = ms.k.IncrementTopicId(ctx)
	if err != nil {
		return nil, err
	}
	if err := ms.k.SetTopic(ctx, id, topic); err != nil {
		return nil, err
	}
	if err = ms.k.ActivateTopic(ctx, id); err != nil {
		return nil, err
	}
	// Rather than set latest weight-adjustment timestamp of a topic to 0
	// we do nothing, since no value in the map means zero

	return &types.MsgCreateNewTopicResponse{TopicId: id}, nil
}

func (ms msgServer) CheckAddressHasBalanceForTopicCreationFee(ctx context.Context, address sdk.AccAddress) (bool, sdk.Coin, error) {
	amountInt, err := ms.k.GetParamsTopicCreationFee(ctx)
	if err != nil {
		return false, sdk.Coin{}, err
	}
	fee := sdk.NewCoin(params.DefaultBondDenom, amountInt)
	balance := ms.k.BankKeeper().GetBalance(ctx, address, fee.Denom)
	return balance.IsGTE(fee), fee, nil
}
