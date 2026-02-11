package testutil

import (
	"context"
	"reflect"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gomock "github.com/golang/mock/gomock"
)

// MockMintKeeper is a mock of MintKeeper interface.
type MockMintKeeper struct {
	ctrl     *gomock.Controller
	recorder *MockMintKeeperMockRecorder
}

// MockStakingKeeperMockRecorder is the mock recorder for MockStakingKeeper.
type MockMintKeeperMockRecorder struct {
	mock *MockMintKeeper
}

// NewMintMintKeeper creates a new mock instance.
func NewMockMintKeeper(ctrl *gomock.Controller) *MockMintKeeper {
	mock := &MockMintKeeper{ctrl: ctrl}
	mock.recorder = &MockMintKeeperMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMintKeeper) EXPECT() *MockMintKeeperMockRecorder {
	return m.recorder
}

// CosmosValidatorStakedSupply(ctx context.Context) (math.Int, error)
func (m *MockMintKeeper) CosmosValidatorStakedSupply(ctx context.Context) (math.Int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CosmosValidatorStakedSupply", ctx)
	ret0, _ := ret[0].(math.Int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockMintKeeperMockRecorder) CosmosValidatorStakedSupply(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CosmosValidatorStakedSupply", reflect.TypeOf((*MockMintKeeper)(nil).CosmosValidatorStakedSupply), ctx)
}

// GetEmissionsKeeperTotalStake(ctx context.Context) (math.Int, error)
func (m *MockMintKeeper) GetEmissionsKeeperTotalStake(ctx context.Context) (math.Int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEmissionsKeeperTotalStake", ctx)
	ret0, _ := ret[0].(math.Int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockMintKeeperMockRecorder) GetEmissionsKeeperTotalStake(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEmissionsKeeperTotalStake", reflect.TypeOf((*MockMintKeeper)(nil).GetEmissionsKeeperTotalStake), ctx)
}

// GetTotalCurrTokenSupply(ctx context.Context) sdk.Coin
func (m *MockMintKeeper) GetTotalCurrTokenSupply(ctx context.Context) sdk.Coin {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTotalCurrTokenSupply", ctx)
	ret0, _ := ret[0].(sdk.Coin)
	return ret0
}

func (mr *MockMintKeeperMockRecorder) GetTotalCurrTokenSupply(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTotalCurrTokenSupply", reflect.TypeOf((*MockMintKeeper)(nil).GetTotalCurrTokenSupply), ctx)
}

// GetPreviousPercentageRewardToStakedReputers(ctx context.Context) (math.LegacyDec, error)
func (m *MockMintKeeper) GetPreviousPercentageRewardToStakedReputers(ctx context.Context) (math.LegacyDec, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPreviousPercentageRewardToStakedReputers", ctx)
	ret0, _ := ret[0].(math.LegacyDec)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockMintKeeperMockRecorder) GetPreviousPercentageRewardToStakedReputers(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPreviousPercentageRewardToStakedReputers", reflect.TypeOf((*MockMintKeeper)(nil).GetPreviousPercentageRewardToStakedReputers), ctx)
}

// GetPreviousRewardEmissionPerUnitStakedToken(ctx context.Context) (math.LegacyDec, error)
func (m *MockMintKeeper) GetPreviousRewardEmissionPerUnitStakedToken(ctx context.Context) (math.LegacyDec, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPreviousRewardEmissionPerUnitStakedToken", ctx)
	ret0, _ := ret[0].(math.LegacyDec)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockMintKeeperMockRecorder) GetPreviousRewardEmissionPerUnitStakedToken(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPreviousRewardEmissionPerUnitStakedToken", reflect.TypeOf((*MockMintKeeper)(nil).GetPreviousRewardEmissionPerUnitStakedToken), ctx)
}
