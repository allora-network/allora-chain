package queryserver

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	state "github.com/allora-network/allora-chain/x/emissions"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Get timestamp of the last rewards update
func (qs queryServer) GetLastRewardsUpdate(ctx context.Context, req *state.QueryLastRewardsUpdateRequest) (*state.QueryLastRewardsUpdateResponse, error) {
	lastRewardsUpdate, err := qs.k.lastRewardsUpdate.Get(ctx)
	if err != nil {
		return nil, err
	}
	return &state.QueryLastRewardsUpdateResponse{LastRewardsUpdate: lastRewardsUpdate}, nil
}

// TotalStake defines the handler for the Query/TotalStake RPC method.
func (qs queryServer) GetTotalStake(ctx context.Context, req *state.QueryTotalStakeRequest) (*state.QueryTotalStakeResponse, error) {
	totalStake, err := qs.k.totalStake.Get(ctx)
	if err != nil {
		return nil, err
	}
	return &state.QueryTotalStakeResponse{Amount: totalStake}, nil
}

// Get the amount of token rewards that have accumulated this epoch
func (qs queryServer) GetAccumulatedEpochRewards(ctx context.Context, req *state.QueryAccumulatedEpochRewardsRequest) (*state.QueryAccumulatedEpochRewardsResponse, error) {
	emissions, err := qs.k.CalculateAccumulatedEmissions(ctx)
	if err != nil {
		return nil, err
	}
	return &state.QueryAccumulatedEpochRewardsResponse{Amount: emissions}, nil
}

// GetReputerStakeList retrieves a list of stakes for a given account address.
func (qs queryServer) GetReputerStakeList(ctx context.Context, req *state.QueryReputerStakeListRequest) (*state.QueryReputerStakeListResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "request cannot be nil")
	}

	address, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, err
	}

	stakes, err := qs.k.GetStakePlacementsForReputer(ctx, address)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var stakePointers []*state.StakeInfo
	for _, stake := range stakes {
		stakePointers = append(stakePointers, &stake)
	}

	return &state.QueryReputerStakeListResponse{Stakes: stakePointers}, nil
}
