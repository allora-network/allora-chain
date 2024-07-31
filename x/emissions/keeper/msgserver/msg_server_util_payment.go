package msgserver

import (
	"context"
	"strconv"

	"cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
	appParams "github.com/allora-network/allora-chain/app/params"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

type TopicId = uint64
type Allo = cosmosMath.Int

func activateTopicIfWeightAtLeastGlobalMin(
	ctx context.Context,
	ms msgServer,
	topicId TopicId,
	newRevenue Allo,
	newStake Allo,
) error {
	isActivated, err := ms.k.IsTopicActive(ctx, topicId)
	if err != nil {
		return errors.Wrapf(err, "error getting topic activation status")
	}
	if !isActivated {
		params, err := ms.k.GetParams(ctx)
		if err != nil {
			return errors.Wrapf(err, "error getting params")
		}
		topic, err := ms.k.GetTopic(ctx, topicId)
		if err != nil {
			return errors.Wrapf(err, "error getting topic")
		}

		newTopicWeight, _, err := ms.k.GetCurrentTopicWeight(
			ctx,
			topicId,
			topic.EpochLength,
			params.TopicRewardAlpha,
			params.TopicRewardStakeImportance,
			params.TopicRewardFeeRevenueImportance,
			newRevenue,
			newStake,
		)
		if err != nil {
			return errors.Wrapf(err, "error getting current topic weight")
		}

		if newTopicWeight.Gte(params.MinTopicWeight) {
			err = ms.k.ActivateTopic(ctx, topicId)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Check if user has enough balance to send the fee, then send the fee to EcoSystem bucket
// insufficientBalanceErrorMsg is appended to error message if sender has insufficient balance
func checkBalanceAndSendFee(
	ctx context.Context,
	ms msgServer,
	sender string,
	topicId TopicId,
	amount Allo,
	insufficientBalanceErrorMsg string,
) error {
	accAddress, err := sdk.AccAddressFromBech32(sender)
	if err != nil {
		return err
	}
	balance := ms.k.GetBankBalance(ctx, accAddress, appParams.DefaultBondDenom)
	fee := sdk.NewCoin(balance.Denom, amount)

	if balance.IsLT(fee) {
		errMsg := insufficientBalanceErrorMsg + " " + strconv.FormatUint(topicId, 10) + " sender " + sender
		return errors.Wrapf(sdkerrors.ErrInsufficientFunds, errMsg)
	}

	err = ms.k.SendCoinsFromAccountToModule(ctx, sender, minttypes.EcosystemModuleName, sdk.NewCoins(fee))
	if err != nil {
		return err
	}

	return nil
}

// Does 4 things:
// 1. Checks if sender has enough balance to send the fee
// 2. Sends coins from sender to mint module Ecosystem bucket
// 3. Adds the amount to the topic's effective revenue
// 4. Activates the topic if the weight is at least the global minimum for active topics
// insufficientBalanceErrorMsg is appended to error message if sender has insufficient balance
// Assumes the topic already exists
func sendEffectiveRevenueActivateTopicIfWeightSufficient(
	ctx context.Context,
	ms msgServer,
	sender string,
	topicId TopicId,
	amount Allo,
	insufficientBalanceErrorMsg string,
) error {
	err := checkBalanceAndSendFee(ctx, ms, sender, topicId, amount, insufficientBalanceErrorMsg)
	if err != nil {
		return err
	}

	err = ms.k.AddTopicFeeRevenue(ctx, topicId, amount)
	if err != nil {
		return err
	}

	err = activateTopicIfWeightAtLeastGlobalMin(ctx, ms, topicId, amount, cosmosMath.ZeroInt())
	return err
}
