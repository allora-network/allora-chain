package queryserver

import (
	"context"
	"time"

	"cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/x/emissions/metrics"

	"github.com/allora-network/allora-chain/x/emissions/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// Params defines the handler for the Query/Params RPC method.
func (qs queryServer) IsWhitelistAdmin(ctx context.Context, req *types.IsWhitelistAdminRequest) (_ *types.IsWhitelistAdminResponse, err error) {
	defer metrics.RecordMetrics("IsWhitelistAdmin", time.Now(), &err)
	if err := qs.k.ValidateStringIsBech32(req.Address); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid address: %s", err)
	}

	isAdmin, err := qs.k.IsWhitelistAdmin(ctx, req.Address)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting whitelist admin")
	}

	return &types.IsWhitelistAdminResponse{IsAdmin: isAdmin}, nil
}

func (qs queryServer) IsWhitelistedGlobalWorker(ctx context.Context, req *types.IsWhitelistedGlobalWorkerRequest) (_ *types.IsWhitelistedGlobalWorkerResponse, err error) {
	defer metrics.RecordMetrics("IsWhitelistedGlobalWorker", time.Now(), &err)
	if err := qs.k.ValidateStringIsBech32(req.Address); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid address: %s", err)
	}

	val, err := qs.k.IsWhitelistedGlobalWorker(ctx, req.Address)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting whitelisted global worker")
	}

	return &types.IsWhitelistedGlobalWorkerResponse{IsWhitelistedGlobalWorker: val}, nil
}

func (qs queryServer) IsWhitelistedGlobalReputer(ctx context.Context, req *types.IsWhitelistedGlobalReputerRequest) (_ *types.IsWhitelistedGlobalReputerResponse, err error) {
	defer metrics.RecordMetrics("IsWhitelistedGlobalReputer", time.Now(), &err)
	if err := qs.k.ValidateStringIsBech32(req.Address); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid address: %s", err)
	}

	val, err := qs.k.IsWhitelistedGlobalReputer(ctx, req.Address)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting whitelisted global reputer")
	}

	return &types.IsWhitelistedGlobalReputerResponse{IsWhitelistedGlobalReputer: val}, nil
}

func (qs queryServer) IsWhitelistedGlobalAdmin(ctx context.Context, req *types.IsWhitelistedGlobalAdminRequest) (_ *types.IsWhitelistedGlobalAdminResponse, err error) {
	defer metrics.RecordMetrics("IsWhitelistedGlobalAdmin", time.Now(), &err)
	if err := qs.k.ValidateStringIsBech32(req.Address); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid address: %s", err)
	}

	val, err := qs.k.IsWhitelistedGlobalAdmin(ctx, req.Address)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting whitelisted global admin")
	}

	return &types.IsWhitelistedGlobalAdminResponse{IsWhitelistedGlobalAdmin: val}, nil
}

func (qs queryServer) IsTopicWorkerWhitelistEnabled(ctx context.Context, req *types.IsTopicWorkerWhitelistEnabledRequest) (_ *types.IsTopicWorkerWhitelistEnabledResponse, err error) {
	defer metrics.RecordMetrics("IsTopicWorkerWhitelistEnabled", time.Now(), &err)

	val, err := qs.k.IsTopicWorkerWhitelistEnabled(ctx, req.TopicId)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting topic worker whitelist enabled")
	}

	return &types.IsTopicWorkerWhitelistEnabledResponse{IsTopicWorkerWhitelistEnabled: val}, nil
}

func (qs queryServer) IsTopicReputerWhitelistEnabled(ctx context.Context, req *types.IsTopicReputerWhitelistEnabledRequest) (_ *types.IsTopicReputerWhitelistEnabledResponse, err error) {
	defer metrics.RecordMetrics("IsTopicReputerWhitelistEnabled", time.Now(), &err)

	val, err := qs.k.IsTopicReputerWhitelistEnabled(ctx, req.TopicId)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting topic reputer whitelist enabled")
	}

	return &types.IsTopicReputerWhitelistEnabledResponse{IsTopicReputerWhitelistEnabled: val}, nil
}

func (qs queryServer) IsWhitelistedTopicCreator(ctx context.Context, req *types.IsWhitelistedTopicCreatorRequest) (_ *types.IsWhitelistedTopicCreatorResponse, err error) {
	defer metrics.RecordMetrics("IsWhitelistedTopicCreator", time.Now(), &err)
	if err := qs.k.ValidateStringIsBech32(req.Address); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid address: %s", err)
	}

	val, err := qs.k.IsWhitelistedTopicCreator(ctx, req.Address)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting whitelisted topic creator")
	}

	return &types.IsWhitelistedTopicCreatorResponse{IsWhitelistedTopicCreator: val}, nil
}

func (qs queryServer) IsWhitelistedGlobalActor(ctx context.Context, req *types.IsWhitelistedGlobalActorRequest) (_ *types.IsWhitelistedGlobalActorResponse, err error) {
	defer metrics.RecordMetrics("IsWhitelistedGlobalActor", time.Now(), &err)
	if err := qs.k.ValidateStringIsBech32(req.Address); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid address: %s", err)
	}

	val, err := qs.k.IsWhitelistedGlobalActor(ctx, req.Address)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting whitelisted global actor")
	}

	return &types.IsWhitelistedGlobalActorResponse{IsWhitelistedGlobalActor: val}, nil
}

func (qs queryServer) IsWhitelistedTopicWorker(ctx context.Context, req *types.IsWhitelistedTopicWorkerRequest) (_ *types.IsWhitelistedTopicWorkerResponse, err error) {
	defer metrics.RecordMetrics("IsWhitelistedTopicWorker", time.Now(), &err)
	if err := qs.k.ValidateStringIsBech32(req.Address); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid address: %s", err)
	}

	val, err := qs.k.IsWhitelistedTopicWorker(ctx, req.TopicId, req.Address)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting whitelisted topic worker")
	}

	return &types.IsWhitelistedTopicWorkerResponse{IsWhitelistedTopicWorker: val}, nil
}

func (qs queryServer) IsWhitelistedTopicReputer(ctx context.Context, req *types.IsWhitelistedTopicReputerRequest) (_ *types.IsWhitelistedTopicReputerResponse, err error) {
	defer metrics.RecordMetrics("IsWhitelistedTopicReputer", time.Now(), &err)
	if err := qs.k.ValidateStringIsBech32(req.Address); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid address: %s", err)
	}

	val, err := qs.k.IsWhitelistedTopicReputer(ctx, req.TopicId, req.Address)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting whitelisted topic reputer")
	}

	return &types.IsWhitelistedTopicReputerResponse{IsWhitelistedTopicReputer: val}, nil
}

func (qs queryServer) CanUpdateAllGlobalWhitelists(ctx context.Context, req *types.CanUpdateAllGlobalWhitelistsRequest) (_ *types.CanUpdateAllGlobalWhitelistsResponse, err error) {
	defer metrics.RecordMetrics("CanUpdateAllGlobalWhitelists", time.Now(), &err)
	if err := qs.k.ValidateStringIsBech32(req.Address); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid address: %s", err)
	}

	val, err := qs.k.CanUpdateAllGlobalWhitelists(ctx, req.Address)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting can update global whitelists")
	}

	return &types.CanUpdateAllGlobalWhitelistsResponse{CanUpdateAllGlobalWhitelists: val}, nil
}

func (qs queryServer) CanUpdateGlobalWorkerWhitelist(ctx context.Context, req *types.CanUpdateGlobalWorkerWhitelistRequest) (_ *types.CanUpdateGlobalWorkerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("CanUpdateGlobalWorkerWhitelist", time.Now(), &err)
	if err := qs.k.ValidateStringIsBech32(req.Address); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid address: %s", err)
	}

	val, err := qs.k.CanUpdateGlobalWorkerWhitelist(ctx, req.Address)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting can update global worker whitelist")
	}

	return &types.CanUpdateGlobalWorkerWhitelistResponse{CanUpdateGlobalWorkerWhitelist: val}, nil
}

func (qs queryServer) CanUpdateGlobalReputerWhitelist(ctx context.Context, req *types.CanUpdateGlobalReputerWhitelistRequest) (_ *types.CanUpdateGlobalReputerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("CanUpdateGlobalReputerWhitelist", time.Now(), &err)
	if err := qs.k.ValidateStringIsBech32(req.Address); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid address: %s", err)
	}

	val, err := qs.k.CanUpdateGlobalReputerWhitelist(ctx, req.Address)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting can update global reputer whitelist")
	}

	return &types.CanUpdateGlobalReputerWhitelistResponse{CanUpdateGlobalReputerWhitelist: val}, nil
}

func (qs queryServer) CanUpdateParams(ctx context.Context, req *types.CanUpdateParamsRequest) (_ *types.CanUpdateParamsResponse, err error) {
	defer metrics.RecordMetrics("CanUpdateParams", time.Now(), &err)
	if err := qs.k.ValidateStringIsBech32(req.Address); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid address: %s", err)
	}

	val, err := qs.k.CanUpdateParams(ctx, req.Address)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting can update params")
	}

	return &types.CanUpdateParamsResponse{CanUpdateParams: val}, nil
}

func (qs queryServer) CanUpdateTopicWhitelist(ctx context.Context, req *types.CanUpdateTopicWhitelistRequest) (_ *types.CanUpdateTopicWhitelistResponse, err error) {
	defer metrics.RecordMetrics("CanUpdateTopicWhitelist", time.Now(), &err)
	if err := qs.k.ValidateStringIsBech32(req.Address); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid address: %s", err)
	}

	val, err := qs.k.CanUpdateTopicWhitelist(ctx, req.TopicId, req.Address)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting can update topic whitelist")
	}

	return &types.CanUpdateTopicWhitelistResponse{CanUpdateTopicWhitelist: val}, nil
}

func (qs queryServer) CanCreateTopic(ctx context.Context, req *types.CanCreateTopicRequest) (_ *types.CanCreateTopicResponse, err error) {
	defer metrics.RecordMetrics("CanCreateTopic", time.Now(), &err)
	if err := qs.k.ValidateStringIsBech32(req.Address); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid address: %s", err)
	}

	val, err := qs.k.CanCreateTopic(ctx, req.Address)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting can create topic")
	}

	return &types.CanCreateTopicResponse{CanCreateTopic: val}, nil
}

func (qs queryServer) CanSubmitWorkerPayload(ctx context.Context, req *types.CanSubmitWorkerPayloadRequest) (_ *types.CanSubmitWorkerPayloadResponse, err error) {
	defer metrics.RecordMetrics("CanSubmitWorkerPayload", time.Now(), &err)
	if err := qs.k.ValidateStringIsBech32(req.Address); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid address: %s", err)
	}

	val, err := qs.k.CanSubmitWorkerPayload(ctx, req.TopicId, req.Address)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting can submit worker payload")
	}

	return &types.CanSubmitWorkerPayloadResponse{CanSubmitWorkerPayload: val}, nil
}

func (qs queryServer) CanSubmitReputerPayload(ctx context.Context, req *types.CanSubmitReputerPayloadRequest) (_ *types.CanSubmitReputerPayloadResponse, err error) {
	defer metrics.RecordMetrics("CanSubmitReputerPayload", time.Now(), &err)
	if err := qs.k.ValidateStringIsBech32(req.Address); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid address: %s", err)
	}

	val, err := qs.k.CanSubmitReputerPayload(ctx, req.TopicId, req.Address)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting can submit reputer payload")
	}

	return &types.CanSubmitReputerPayloadResponse{CanSubmitReputerPayload: val}, nil
}
