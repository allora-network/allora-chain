package types

import (
	"cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	alloramath "github.com/allora-network/allora-chain/math"
)

func (workerValue *WorkerAttributedValue) GetValue() alloramath.Dec {
	return workerValue.Value
}

func (withheldWorkerValue *WithheldWorkerAttributedValue) GetValue() alloramath.Dec {
	return withheldWorkerValue.Value
}

func (rvb ReputerValueBundles) ToLossBundles(topicId uint64, block int64) (LossBundles, error) {
	lossBundles := make(LossBundles, len(rvb.ReputerValueBundles))
	for i, reputerValueBundle := range rvb.ReputerValueBundles {
		if reputerValueBundle == nil {
			return nil, errors.Wrapf(
				sdkerrors.ErrInvalidRequest,
				"nil reputer value bundle at index %d for topic %d at block %d",
				i, topicId, block,
			)
		}
		if reputerValueBundle.ValueBundle == nil {
			return nil, errors.Wrapf(
				sdkerrors.ErrInvalidRequest,
				"nil value bundle at index %d for topic %d at block %d",
				i, topicId, block,
			)
		}
		lossBundles[i] = reputerValueBundle.ValueBundle
	}
	return lossBundles, nil
}

func (rvb *ReputerValueBundles) Validate() error {
	if rvb == nil || rvb.ReputerValueBundles == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "ReputerValueBundles cannot be nil")
	}
	lbs := make(LossBundles, len(rvb.ReputerValueBundles))
	for i := range rvb.ReputerValueBundles {
		if rvb.ReputerValueBundles[i] == nil {
			return errors.Wrapf(sdkerrors.ErrInvalidRequest, "reputer_value_bundle is nil at index %d", i)
		}
		if rvb.ReputerValueBundles[i].ValueBundle == nil {
			return errors.Wrapf(sdkerrors.ErrInvalidRequest, "value_bundle is nil at index %d", i)
		}
		lbs[i] = rvb.ReputerValueBundles[i].ValueBundle
	}
	return lbs.Validate()
}
