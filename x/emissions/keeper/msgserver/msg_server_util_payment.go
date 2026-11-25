package msgserver

import (
	"context"

	"cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	appParams "github.com/allora-network/allora-chain/app/params"
	"github.com/allora-network/allora-chain/x/emissions/types"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
)

type TopicId = uint64
type Allo = cosmosMath.Int

func (ms msgServer) activateTopicIfWeightAtLeastGlobalMin(
	ctx context.Context,
	topicId TopicId,
) error {
	isActivated, err := ms.tk.IsTopicActive(ctx, topicId)
	if err != nil {
		return errors.Wrapf(err, "error getting topic activation status")
	}
	if !isActivated {
		params, err := ms.pk.GetParams(ctx)
		if err != nil {
			return errors.Wrapf(err, "error getting params")
		}
		topic, err := ms.tk.GetTopic(ctx, topicId)
		if err != nil {
			return errors.Wrapf(err, "error getting topic")
		}

		newTopicWeight, _, _, err := ms.tk.GetCurrentTopicWeight(
			ctx,
			topicId,
			topic.EpochLength,
			params.TopicRewardAlpha,
			params.TopicRewardStakeImportance,
			params.TopicRewardFeeRevenueImportance,
			params.BlocksPerMonth,
		)
		if err != nil {
			return errors.Wrapf(err, "error getting current topic weight")
		}

		if newTopicWeight.Gte(params.MinTopicWeight) {
			err = ms.tk.ActivateTopic(ctx, topicId, types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_ACTIVATED_BY_FUNDING, newTopicWeight, params.MinTopicWeight)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Sends the fee to the ecosystem bucket and adds it to the topic's fee revenue.
// A zero amount skips the bank transfer and the revenue write, and returns the
// stored total with credited false. A nonzero amount is stored and credited is
// true. Callers emit EventFundTopic when credited is true. FundTopic emits
// either way, including a zero amount.
func (ms msgServer) sendEffectiveRevenue(
	ctx context.Context,
	sender string,
	topicId TopicId,
	amount Allo,
) (newRevenue cosmosMath.Int, credited bool, err error) {
	if amount.IsZero() {
		newRevenue, err = ms.tk.GetTopicFeeRevenue(ctx, topicId)
		if err != nil {
			return cosmosMath.Int{}, false, err
		}
		return newRevenue, false, nil
	}

	if err = ms.checkBalanceAndSendFee(ctx, sender, amount); err != nil {
		return cosmosMath.Int{}, false, err
	}
	newRevenue, err = ms.tk.AddTopicFeeRevenue(ctx, topicId, amount)
	if err != nil {
		return cosmosMath.Int{}, false, err
	}
	return newRevenue, true, nil
}

// Check if user has enough balance to send the fee, then send the fee to EcoSystem bucket
func (ms msgServer) checkBalanceAndSendFee(ctx context.Context, sender string, amount Allo) error {
	if amount.IsZero() {
		return nil
	}

	accAddress, err := sdk.AccAddressFromBech32(sender)
	if err != nil {
		return err
	}
	balance := ms.bk.GetBalance(ctx, accAddress, appParams.DefaultBondDenom)
	fee := sdk.NewCoin(balance.Denom, amount)

	if balance.IsLT(fee) {
		return errors.Wrapf(sdkerrors.ErrInsufficientFunds, "sender has insufficient balance to cover fees")
	}

	err = ms.bk.SendCoinsFromAccountToModule(ctx, sender, minttypes.EcosystemModuleName, sdk.NewCoins(fee))
	if err != nil {
		return err
	}

	return nil
}
