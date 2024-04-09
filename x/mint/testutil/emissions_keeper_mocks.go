package testutil

import (
	context "context"
	reflect "reflect"

	math "cosmossdk.io/math"
	gomock "github.com/golang/mock/gomock"
)

// MockEmissionsKeeper is a mock of BankKeeper interface.
type MockEmissionsKeeper struct {
	ctrl     *gomock.Controller
	recorder *MockEmissionsKeeperMockRecorder
}

// MockStakingKeeperMockRecorder is the mock recorder for MockStakingKeeper.
type MockEmissionsKeeperMockRecorder struct {
	mock *MockEmissionsKeeper
}

// NewEmissionsBankKeeper creates a new mock instance.
func NewMockEmissionsKeeper(ctrl *gomock.Controller) *MockEmissionsKeeper {
	mock := &MockEmissionsKeeper{ctrl: ctrl}
	mock.recorder = &MockEmissionsKeeperMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEmissionsKeeper) EXPECT() *MockEmissionsKeeperMockRecorder {
	return m.recorder
}

func (m *MockEmissionsKeeper) GetTotalStake(ctx context.Context) (math.Uint, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTotalStake", ctx)
	ret0, _ := ret[0].(math.Uint)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTotalStake indicates an expected call of GetTotalStake.
func (mr *MockEmissionsKeeperMockRecorder) GetTotalStake(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTotalStake", reflect.TypeOf((*MockEmissionsKeeper)(nil).GetTotalStake), ctx)
}

func (m *MockEmissionsKeeper) GetParamsValidatorsVsAlloraPercentReward(ctx context.Context) (math.LegacyDec, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetParamsValidatorsVsAlloraPercentReward", ctx)
	ret0, _ := ret[0].(math.LegacyDec)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockEmissionsKeeperMockRecorder) GetParamsValidatorsVsAlloraPercentReward(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetParamsValidatorsVsAlloraPercentReward", reflect.TypeOf((*MockEmissionsKeeper)(nil).GetParamsValidatorsVsAlloraPercentReward), ctx)
}
