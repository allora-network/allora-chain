package keeper

import (
	"context"

	"cosmossdk.io/errors"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (k *NonceKeeper) InitGenesis(ctx context.Context, data *types.GenesisState) error {
	// OpenWorkerWindows
	for _, blockHeightAndListOfTopicIds := range data.OpenWorkerWindows {
		if blockHeightAndListOfTopicIds != nil {
			topicIds := types.TopicIds{TopicIds: blockHeightAndListOfTopicIds.TopicIds}
			for _, topicId := range topicIds.TopicIds {
				if err := types.ValidateTopicId(topicId); err != nil {
					return errors.Wrap(err, "error validating topic id")
				}
			}
			if err := types.ValidateBlockHeight(blockHeightAndListOfTopicIds.BlockHeight); err != nil {
				return errors.Wrap(err, "error validating block height")
			}
			if err := k.openWorkerWindows.Set(
				ctx,
				blockHeightAndListOfTopicIds.BlockHeight,
				topicIds,
			); err != nil {
				return errors.Wrap(err, "error setting openWorkerWindows")
			}
		}
	}

	// UnfulfilledWorkerNonces
	for _, topicIdAndNonces := range data.UnfulfilledWorkerNonces {
		if topicIdAndNonces != nil {
			if err := topicIdAndNonces.Nonces.Validate(); err != nil {
				return errors.Wrap(err, "error validating unfulfilled worker nonces")
			}
			if err := k.unfulfilledWorkerNonces.Set(ctx, topicIdAndNonces.TopicId, *topicIdAndNonces.Nonces); err != nil {
				return errors.Wrap(err, "error setting unfulfilledWorkerNonces")
			}
		}
	}

	// UnfulfilledReputerNonces
	for _, topicIdAndReputerRequestNonces := range data.UnfulfilledReputerNonces {
		if topicIdAndReputerRequestNonces != nil {
			if err := topicIdAndReputerRequestNonces.ReputerRequestNonces.Validate(); err != nil {
				return errors.Wrap(err, "error validating unfulfilled reputer nonces")
			}
			if err := k.unfulfilledReputerNonces.Set(ctx, topicIdAndReputerRequestNonces.TopicId, *topicIdAndReputerRequestNonces.ReputerRequestNonces); err != nil {
				return errors.Wrap(err, "error setting unfulfilledReputerNonces")
			}
		}
	}

	return nil
}

func (k *NonceKeeper) ExportGenesis(ctx context.Context, data *types.GenesisState) error {
	// openWorkerWindows
	openWorkerWindows := make([]*types.BlockHeightAndTopicIds, 0)
	openWorkerWindowsIter, err := k.openWorkerWindows.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate open worker windows")
	}
	for ; openWorkerWindowsIter.Valid(); openWorkerWindowsIter.Next() {
		keyValue, err := openWorkerWindowsIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: openWorkerWindowsIter")
		}
		openWorkerWindows = append(openWorkerWindows, &types.BlockHeightAndTopicIds{
			BlockHeight: keyValue.Key,
			TopicIds:    keyValue.Value.TopicIds,
		})
	}
	data.OpenWorkerWindows = openWorkerWindows

	// unfulfilledWorkerNonces
	unfulfilledWorkerNonces := make([]*types.TopicIdAndNonces, 0)
	unfulfilledWorkerNoncesIter, err := k.unfulfilledWorkerNonces.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate unfulfilled worker nonces")
	}
	for ; unfulfilledWorkerNoncesIter.Valid(); unfulfilledWorkerNoncesIter.Next() {
		keyValue, err := unfulfilledWorkerNoncesIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: unfulfilledWorkerNoncesIter")
		}
		unfulfilledWorkerNonces = append(unfulfilledWorkerNonces, &types.TopicIdAndNonces{
			TopicId: keyValue.Key,
			Nonces:  &keyValue.Value,
		})
	}
	data.UnfulfilledWorkerNonces = unfulfilledWorkerNonces

	// unfulfilledReputerNonces
	unfulfilledReputerNonces := make([]*types.TopicIdAndReputerRequestNonces, 0)
	unfulfilledReputerNoncesIter, err := k.unfulfilledReputerNonces.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate unfulfilled reputer nonces")
	}
	for ; unfulfilledReputerNoncesIter.Valid(); unfulfilledReputerNoncesIter.Next() {
		keyValue, err := unfulfilledReputerNoncesIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: unfulfilledReputerNoncesIter")
		}
		value := keyValue.Value
		unfulfilledReputerNonces = append(unfulfilledReputerNonces, &types.TopicIdAndReputerRequestNonces{
			TopicId:              keyValue.Key,
			ReputerRequestNonces: &value,
		})
	}
	data.UnfulfilledReputerNonces = unfulfilledReputerNonces

	return nil
}
