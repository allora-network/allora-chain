package testutil

import (
	gomock "github.com/golang/mock/gomock"
)

// MockUpshotKeeper is a mock of UpshotKeeper interface.
type MockUpshotKeeper struct {
	ctrl     *gomock.Controller
	recorder *MockUpshotKeeperMockRecorder
}

// MockUpshotKeeperMockRecorder is the mock recorder for MockUpshotKeeper.
type MockUpshotKeeperMockRecorder struct {
	mock *MockUpshotKeeper
}

// NewMockUpshotKeeper creates a new mock instance.
func NewMockUpshotKeeper(ctrl *gomock.Controller) *MockUpshotKeeper {
	mock := &MockUpshotKeeper{ctrl: ctrl}
	mock.recorder = &MockUpshotKeeperMockRecorder{mock}
	return mock
}
