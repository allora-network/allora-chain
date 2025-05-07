package module_test

import (
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	emissionstestutil "github.com/allora-network/allora-chain/test/testutil"
	"github.com/allora-network/allora-chain/x/emissions/module"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// nolint: exhaustruct
func TestRemoveStakes(t *testing.T) {
	ctx := testutil.DefaultContext(
		storetypes.NewKVStoreKey(emissionstypes.ModuleName),
		storetypes.NewTransientStoreKey("transient_test"),
	)

	testCases := []struct {
		name          string
		currentBlock  int64
		stakeRemovals []emissionstypes.StakeRemovalInfo
		limitHit      bool
	}{
		{
			name:          "no stake removals",
			stakeRemovals: []emissionstypes.StakeRemovalInfo{},
		},
		{
			name:         "one stake removal",
			currentBlock: 2000,
			stakeRemovals: []emissionstypes.StakeRemovalInfo{
				{
					BlockRemovalStarted:   1500,
					TopicId:               1,
					Reputer:               "reputer",
					Amount:                math.NewInt(100),
					BlockRemovalCompleted: 2000,
				},
			},
		},
		{
			name:         "two stake removals",
			currentBlock: 2000,
			stakeRemovals: []emissionstypes.StakeRemovalInfo{
				{
					BlockRemovalStarted:   1500,
					TopicId:               1,
					Reputer:               "reputer1",
					Amount:                math.NewInt(100),
					BlockRemovalCompleted: 2000,
				},
				{
					BlockRemovalStarted:   1500,
					TopicId:               2,
					Reputer:               "reputer2",
					Amount:                math.NewInt(200),
					BlockRemovalCompleted: 2000,
				},
			},
		},
		{
			name:         "stake removals previously completed",
			currentBlock: 2000,
			stakeRemovals: []emissionstypes.StakeRemovalInfo{
				{
					BlockRemovalStarted:   1000,
					TopicId:               1,
					Reputer:               "reputer1",
					Amount:                math.NewInt(100),
					BlockRemovalCompleted: 1500,
				},
				{
					BlockRemovalStarted:   500,
					TopicId:               2,
					Reputer:               "reputer2",
					Amount:                math.NewInt(200),
					BlockRemovalCompleted: 1000,
				},
			},
		},
		{
			name:         "limit hit",
			currentBlock: 2000,
			stakeRemovals: []emissionstypes.StakeRemovalInfo{
				{
					BlockRemovalStarted:   1000,
					TopicId:               1,
					Reputer:               "reputer1",
					Amount:                math.NewInt(100),
					BlockRemovalCompleted: 1500,
				},
				{
					BlockRemovalStarted:   1500,
					TopicId:               2,
					Reputer:               "reputer2",
					Amount:                math.NewInt(200),
					BlockRemovalCompleted: 2000,
				},
			},
			limitHit: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			kMock := emissionstestutil.NewMockEmissionsKeeper(ctrl)
			kMock.EXPECT().GetStakeRemovalsUpUntilBlock(gomock.Any(), tc.currentBlock, uint64(2)).Return(tc.stakeRemovals, tc.limitHit, nil)
			for _, removal := range tc.stakeRemovals {
				kMock.EXPECT().RemoveReputerStake(gomock.Any(), removal.BlockRemovalCompleted, removal.TopicId, removal.Reputer, removal.Amount).Return(nil)
				kMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), emissionstypes.AlloraStakingAccountName, removal.Reputer, types.NewCoins(types.NewCoin("uallo", removal.Amount))).Return(nil)
			}

			require.NoError(t, module.RemoveStakes(ctx, tc.currentBlock, kMock, 2))
		})
	}
}

// nolint: exhaustruct
func TestRemoveDelegateStakes(t *testing.T) {
	ctx := testutil.DefaultContext(
		storetypes.NewKVStoreKey(emissionstypes.ModuleName),
		storetypes.NewTransientStoreKey("transient_test"),
	)

	testCases := []struct {
		name          string
		currentBlock  int64
		stakeRemovals []emissionstypes.DelegateStakeRemovalInfo
		limitHit      bool
	}{
		{
			name:          "no stake removals",
			stakeRemovals: []emissionstypes.DelegateStakeRemovalInfo{},
		},
		{
			name:         "one stake removal",
			currentBlock: 2000,
			stakeRemovals: []emissionstypes.DelegateStakeRemovalInfo{
				{
					BlockRemovalStarted:   1500,
					TopicId:               1,
					Reputer:               "reputer",
					Delegator:             "delegator",
					Amount:                math.NewInt(100),
					BlockRemovalCompleted: 2000,
				},
			},
		},
		{
			name:         "two stake removals",
			currentBlock: 2000,
			stakeRemovals: []emissionstypes.DelegateStakeRemovalInfo{
				{
					BlockRemovalStarted:   1500,
					TopicId:               1,
					Reputer:               "reputer1",
					Delegator:             "delegator1",
					Amount:                math.NewInt(100),
					BlockRemovalCompleted: 2000,
				},
				{
					BlockRemovalStarted:   1500,
					TopicId:               2,
					Reputer:               "reputer2",
					Delegator:             "delegator2",
					Amount:                math.NewInt(200),
					BlockRemovalCompleted: 2000,
				},
			},
		},
		{
			name:         "stake removals previously completed",
			currentBlock: 2000,
			stakeRemovals: []emissionstypes.DelegateStakeRemovalInfo{
				{
					BlockRemovalStarted:   1000,
					TopicId:               1,
					Reputer:               "reputer1",
					Delegator:             "delegator1",
					Amount:                math.NewInt(100),
					BlockRemovalCompleted: 1500,
				},
				{
					BlockRemovalStarted:   500,
					TopicId:               2,
					Reputer:               "reputer2",
					Delegator:             "delegator2",
					Amount:                math.NewInt(200),
					BlockRemovalCompleted: 1000,
				},
			},
		},
		{
			name:         "limit hit",
			currentBlock: 2000,
			stakeRemovals: []emissionstypes.DelegateStakeRemovalInfo{
				{
					BlockRemovalStarted:   1000,
					TopicId:               1,
					Reputer:               "reputer1",
					Delegator:             "delegator1",
					Amount:                math.NewInt(100),
					BlockRemovalCompleted: 1500,
				},
				{
					BlockRemovalStarted:   1500,
					TopicId:               2,
					Reputer:               "reputer2",
					Delegator:             "delegator2",
					Amount:                math.NewInt(200),
					BlockRemovalCompleted: 2000,
				},
			},
			limitHit: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			kMock := emissionstestutil.NewMockEmissionsKeeper(ctrl)
			kMock.EXPECT().GetDelegateStakeRemovalsUpUntilBlock(gomock.Any(), tc.currentBlock, uint64(2)).Return(tc.stakeRemovals, tc.limitHit, nil)
			for _, removal := range tc.stakeRemovals {
				kMock.EXPECT().RemoveDelegateStake(gomock.Any(), removal.BlockRemovalCompleted, removal.TopicId, removal.Delegator, removal.Reputer, removal.Amount).Return(nil)
				kMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), emissionstypes.AlloraStakingAccountName, removal.Delegator, types.NewCoins(types.NewCoin("uallo", removal.Amount))).Return(nil)
			}

			require.NoError(t, module.RemoveDelegateStakes(ctx, tc.currentBlock, kMock, 2))
		})
	}
}
