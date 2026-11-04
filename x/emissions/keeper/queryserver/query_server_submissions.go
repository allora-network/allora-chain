package queryserver

import (
	"context"
	"time"

	"github.com/allora-network/allora-chain/x/emissions/metrics"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (qs queryServer) GetOpenReputerSubmissionWindows(ctx context.Context, req *types.GetOpenReputerSubmissionWindowsRequest) (_ *types.GetOpenReputerSubmissionWindowsResponse, err error) {
	defer metrics.RecordMetrics("GetOpenReputerSubmissionWindows", time.Now(), &err)

	openNonces, err := qs.nk.GetOpenReputerSubmissionWindows(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.GetOpenReputerSubmissionWindowsResponse{
		Nonces: &openNonces,
	}, nil
}

func (qs queryServer) GetOpenWorkerSubmissionWindows(ctx context.Context, req *types.GetOpenWorkerSubmissionWindowsRequest) (_ *types.GetOpenWorkerSubmissionWindowsResponse, err error) {
	defer metrics.RecordMetrics("GetOpenWorkerSubmissionWindows", time.Now(), &err)

	openNonces, err := qs.nk.GetOpenWorkerSubmissionWindows(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.GetOpenWorkerSubmissionWindowsResponse{
		Nonces: &openNonces,
	}, nil
}
