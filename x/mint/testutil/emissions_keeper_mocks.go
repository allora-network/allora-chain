package testutil

import (
	context "context"
	reflect "reflect"

	math "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	gomock "github.com/golang/mock/gomock"
)

// MockEmissionsKeeper is a mock of EmissionsKeeper interface.
type MockEmissionsKeeper struct {
	ctrl     *gomock.Controller
	recorder *MockEmissionsKeeperMockRecorder
}

// MockStakingKeeperMockRecorder is the mock recorder for MockStakingKeeper.
type MockEmissionsKeeperMockRecorder struct {
	mock *MockEmissionsKeeper
}

// NewEmissionsEmissionsKeeper creates a new mock instance.
func NewMockEmissionsKeeper(ctrl *gomock.Controller) *MockEmissionsKeeper {
	mock := &MockEmissionsKeeper{ctrl: ctrl}
	mock.recorder = &MockEmissionsKeeperMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEmissionsKeeper) EXPECT() *MockEmissionsKeeperMockRecorder {
	return m.recorder
}

func (m *MockEmissionsKeeper) GetTotalStake(ctx context.Context) (math.Int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTotalStake", ctx)
	ret0, _ := ret[0].(math.Int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTotalStake indicates an expected call of GetTotalStake.
func (mr *MockEmissionsKeeperMockRecorder) GetTotalStake(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTotalStake", reflect.TypeOf((*MockEmissionsKeeper)(nil).GetTotalStake), ctx)
}

func (m *MockEmissionsKeeper) GetParams(ctx context.Context) (emissionstypes.Params, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetParams", ctx)
	ret0, _ := ret[0].(emissionstypes.Params)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockEmissionsKeeperMockRecorder) GetParams(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetParams", reflect.TypeOf((*MockEmissionsKeeper)(nil).GetParams), ctx)
}

func (m *MockEmissionsKeeper) SetParams(ctx context.Context, params emissionstypes.Params) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetParams", ctx, params)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockEmissionsKeeperMockRecorder) SetParams(ctx, params interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetParams", reflect.TypeOf((*MockEmissionsKeeper)(nil).SetParams), ctx, params)
}

func (mr *MockEmissionsKeeperMockRecorder) GetParamsBlocksPerMonth(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetParamsBlocksPerMonth", reflect.TypeOf((*MockEmissionsKeeper)(nil).GetParamsBlocksPerMonth), ctx)
}

func (m *MockEmissionsKeeper) GetParamsBlocksPerMonth(ctx context.Context) (uint64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetParamsBlocksPerMonth", ctx)
	ret0, _ := ret[0].(uint64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockEmissionsKeeperMockRecorder) GetPreviousPercentageRewardToStakedReputers(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPreviousPercentageRewardToStakedReputers", reflect.TypeOf((*MockEmissionsKeeper)(nil).GetPreviousPercentageRewardToStakedReputers), ctx)
}

func (m *MockEmissionsKeeper) GetPreviousPercentageRewardToStakedReputers(ctx context.Context) (alloraMath.Dec, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPreviousPercentageRewardToStakedReputers", ctx)
	ret0, _ := ret[0].(alloraMath.Dec)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockEmissionsKeeperMockRecorder) IsWhitelistAdmin(ctx interface{}, admin interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsWhitelistAdmin", reflect.TypeOf((*MockEmissionsKeeper)(nil).IsWhitelistAdmin), ctx, admin)
}

func (m *MockEmissionsKeeper) IsWhitelistAdmin(ctx context.Context, admin string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsWhitelistAdmin", ctx, admin)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}
