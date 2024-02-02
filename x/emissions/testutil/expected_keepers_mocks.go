package testutil

import (
	context "context"
	reflect "reflect"
	gomock "github.com/golang/mock/gomock"
	types0 "github.com/cosmos/cosmos-sdk/types"
	types "github.com/cosmos/cosmos-sdk/x/auth/types"
	address "cosmossdk.io/core/address"
)


type MockBankKeeper struct {
	ctrl     *gomock.Controller
	recorder *MockBankKeeperMockRecorder
}

// MockBankKeeperMockRecorder is the mock recorder for MockBankKeeper.
type MockBankKeeperMockRecorder struct {
	mock *MockBankKeeper
}


// MockUpshotKeeper is a mock of UpshotKeeper interface.
func NewMockBankKeeper(ctrl *gomock.Controller) *MockBankKeeper {
	mock := &MockBankKeeper{ctrl: ctrl}
	mock.recorder = &MockBankKeeperMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBankKeeper) EXPECT() *MockBankKeeperMockRecorder {
	return m.recorder
}

// BurnCoins mocks base method.
func (m *MockBankKeeper) BurnCoins(arg0 context.Context, arg1 []byte, arg2 types0.Coins) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BurnCoins", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// BurnCoins indicates an expected call of BurnCoins.
func (mr *MockBankKeeperMockRecorder) BurnCoins(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BurnCoins", reflect.TypeOf((*MockBankKeeper)(nil).BurnCoins), arg0, arg1, arg2)
}

// DelegateCoinsFromAccountToModule mocks base method.
func (m *MockBankKeeper) DelegateCoinsFromAccountToModule(ctx context.Context, senderAddr types0.AccAddress, recipientModule string, amt types0.Coins) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DelegateCoinsFromAccountToModule", ctx, senderAddr, recipientModule, amt)
	ret0, _ := ret[0].(error)
	return ret0
}

// DelegateCoinsFromAccountToModule indicates an expected call of DelegateCoinsFromAccountToModule.
func (mr *MockBankKeeperMockRecorder) DelegateCoinsFromAccountToModule(ctx, senderAddr, recipientModule, amt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DelegateCoinsFromAccountToModule", reflect.TypeOf((*MockBankKeeper)(nil).DelegateCoinsFromAccountToModule), ctx, senderAddr, recipientModule, amt)
}

// GetAllBalances mocks base method.
func (m *MockBankKeeper) GetAllBalances(ctx context.Context, addr types0.AccAddress) types0.Coins {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAllBalances", ctx, addr)
	ret0, _ := ret[0].(types0.Coins)
	return ret0
}

// GetAllBalances indicates an expected call of GetAllBalances.
func (mr *MockBankKeeperMockRecorder) GetAllBalances(ctx, addr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAllBalances", reflect.TypeOf((*MockBankKeeper)(nil).GetAllBalances), ctx, addr)
}

// GetBalance mocks base method.
func (m *MockBankKeeper) GetBalance(ctx context.Context, addr types0.AccAddress, denom string) types0.Coin {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBalance", ctx, addr, denom)
	ret0, _ := ret[0].(types0.Coin)
	return ret0
}

// GetBalance indicates an expected call of GetBalance.
func (mr *MockBankKeeperMockRecorder) GetBalance(ctx, addr, denom interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBalance", reflect.TypeOf((*MockBankKeeper)(nil).GetBalance), ctx, addr, denom)
}

// GetSupply mocks base method.
func (m *MockBankKeeper) GetSupply(ctx context.Context, denom string) types0.Coin {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSupply", ctx, denom)
	ret0, _ := ret[0].(types0.Coin)
	return ret0
}

// GetSupply indicates an expected call of GetSupply.
func (mr *MockBankKeeperMockRecorder) GetSupply(ctx, denom interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSupply", reflect.TypeOf((*MockBankKeeper)(nil).GetSupply), ctx, denom)
}

// LockedCoins mocks base method.
func (m *MockBankKeeper) LockedCoins(ctx context.Context, addr types0.AccAddress) types0.Coins {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LockedCoins", ctx, addr)
	ret0, _ := ret[0].(types0.Coins)
	return ret0
}

// LockedCoins indicates an expected call of LockedCoins.
func (mr *MockBankKeeperMockRecorder) LockedCoins(ctx, addr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LockedCoins", reflect.TypeOf((*MockBankKeeper)(nil).LockedCoins), ctx, addr)
}

// SendCoinsFromAccountToModule mocks base method.
func (m *MockBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr types0.AccAddress, recipientModule string, amt types0.Coins) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendCoinsFromAccountToModule", ctx, senderAddr, recipientModule, amt)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendCoinsFromAccountToModule indicates an expected call of SendCoinsFromAccountToModule.
func (mr *MockBankKeeperMockRecorder) SendCoinsFromAccountToModule(ctx, senderAddr, recipientModule, amt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendCoinsFromAccountToModule", reflect.TypeOf((*MockBankKeeper)(nil).SendCoinsFromAccountToModule), ctx, senderAddr, recipientModule, amt)
}

// SendCoinsFromModuleToModule mocks base method.
func (m *MockBankKeeper) SendCoinsFromModuleToModule(ctx context.Context, senderPool, recipientPool string, amt types0.Coins) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendCoinsFromModuleToModule", ctx, senderPool, recipientPool, amt)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendCoinsFromModuleToModule indicates an expected call of SendCoinsFromModuleToModule.
func (mr *MockBankKeeperMockRecorder) SendCoinsFromModuleToModule(ctx, senderPool, recipientPool, amt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendCoinsFromModuleToModule", reflect.TypeOf((*MockBankKeeper)(nil).SendCoinsFromModuleToModule), ctx, senderPool, recipientPool, amt)
}

// SpendableCoins mocks base method.
func (m *MockBankKeeper) SpendableCoins(ctx context.Context, addr types0.AccAddress) types0.Coins {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SpendableCoins", ctx, addr)
	ret0, _ := ret[0].(types0.Coins)
	return ret0
}

// SpendableCoins indicates an expected call of SpendableCoins.
func (mr *MockBankKeeperMockRecorder) SpendableCoins(ctx, addr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SpendableCoins", reflect.TypeOf((*MockBankKeeper)(nil).SpendableCoins), ctx, addr)
}

// UndelegateCoinsFromModuleToAccount mocks base method.
func (m *MockBankKeeper) UndelegateCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr types0.AccAddress, amt types0.Coins) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UndelegateCoinsFromModuleToAccount", ctx, senderModule, recipientAddr, amt)
	ret0, _ := ret[0].(error)
	return ret0
}

// UndelegateCoinsFromModuleToAccount indicates an expected call of UndelegateCoinsFromModuleToAccount.
func (mr *MockBankKeeperMockRecorder) UndelegateCoinsFromModuleToAccount(ctx, senderModule, recipientAddr, amt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UndelegateCoinsFromModuleToAccount", reflect.TypeOf((*MockBankKeeper)(nil).UndelegateCoinsFromModuleToAccount), ctx, senderModule, recipientAddr, amt)
}

// BlockedAddr mocks base method.
func (m *MockBankKeeper) BlockedAddr(addr types0.AccAddress) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BlockedAddr", addr)
	ret0, _ := ret[0].(bool)
	return ret0
}

// BlockedAddr indicates an expected call of BlockedAddr.
func (mr *MockBankKeeperMockRecorder) BlockedAddr(addr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BlockedAddr", reflect.TypeOf((*MockBankKeeper)(nil).BlockedAddr), addr)
}

// MintCoins mocks base method.
func (m *MockBankKeeper) MintCoins(ctx context.Context, moduleName string, amt types0.Coins) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MintCoins", ctx, moduleName, amt)
	ret0, _ := ret[0].(error)
	return ret0
}

// MintCoins indicates an expected call of MintCoins.
func (mr *MockBankKeeperMockRecorder) MintCoins(ctx, moduleName, amt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MintCoins", reflect.TypeOf((*MockBankKeeper)(nil).MintCoins), ctx, moduleName, amt)
}

// SendCoinsFromModuleToAccount mocks base method.
func (m *MockBankKeeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr types0.AccAddress, amt types0.Coins) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendCoinsFromModuleToAccount", ctx, senderModule, recipientAddr, amt)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendCoinsFromModuleToAccount indicates an expected call of SendCoinsFromModuleToAccount.
func (mr *MockBankKeeperMockRecorder) SendCoinsFromModuleToAccount(ctx, senderModule, recipientAddr, amt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendCoinsFromModuleToAccount", reflect.TypeOf((*MockBankKeeper)(nil).SendCoinsFromModuleToAccount), ctx, senderModule, recipientAddr, amt)
}

// MockAccountKeeper is a mock of AccountKeeper interface.
type MockAccountKeeper struct {
	ctrl     *gomock.Controller
	recorder *MockAccountKeeperMockRecorder
}

// MockAccountKeeperMockRecorder is the mock recorder for MockAccountKeeper.
type MockAccountKeeperMockRecorder struct {
	mock *MockAccountKeeper
}

// NewMockAccountKeeper creates a new mock instance.
func NewMockAccountKeeper(ctrl *gomock.Controller) *MockAccountKeeper {
	mock := &MockAccountKeeper{ctrl: ctrl}
	mock.recorder = &MockAccountKeeperMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAccountKeeper) EXPECT() *MockAccountKeeperMockRecorder {
	return m.recorder
}

// AddressCodec mocks base method.
func (m *MockAccountKeeper) AddressCodec() address.Codec {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddressCodec")
	ret0, _ := ret[0].(address.Codec)
	return ret0
}

// AddressCodec indicates an expected call of AddressCodec.
func (mr *MockAccountKeeperMockRecorder) AddressCodec() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddressCodec", reflect.TypeOf((*MockAccountKeeper)(nil).AddressCodec))
}

// GetAccount mocks base method.
func (m *MockAccountKeeper) GetAccount(ctx context.Context, addr types0.AccAddress) types0.AccountI {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAccount", ctx, addr)
	ret0, _ := ret[0].(types0.AccountI)
	return ret0
}

// GetAccount indicates an expected call of GetAccount.
func (mr *MockAccountKeeperMockRecorder) GetAccount(ctx, addr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAccount", reflect.TypeOf((*MockAccountKeeper)(nil).GetAccount), ctx, addr)
}

// GetAllAccounts mocks base method.
func (m *MockAccountKeeper) GetAllAccounts(ctx context.Context) []types0.AccountI {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAllAccounts", ctx)
	ret0, _ := ret[0].([]types0.AccountI)
	return ret0
}

// GetAllAccounts indicates an expected call of GetAllAccounts.
func (mr *MockAccountKeeperMockRecorder) GetAllAccounts(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAllAccounts", reflect.TypeOf((*MockAccountKeeper)(nil).GetAllAccounts), ctx)
}

// GetModuleAccount mocks base method.
func (m *MockAccountKeeper) GetModuleAccount(ctx context.Context, moduleName string) types0.ModuleAccountI {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModuleAccount", ctx, moduleName)
	ret0, _ := ret[0].(types0.ModuleAccountI)
	return ret0
}

// GetModuleAccount indicates an expected call of GetModuleAccount.
func (mr *MockAccountKeeperMockRecorder) GetModuleAccount(ctx, moduleName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModuleAccount", reflect.TypeOf((*MockAccountKeeper)(nil).GetModuleAccount), ctx, moduleName)
}

// GetModuleAccountAndPermissions mocks base method.
func (m *MockAccountKeeper) GetModuleAccountAndPermissions(ctx context.Context, moduleName string) (types0.ModuleAccountI, []string) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModuleAccountAndPermissions", ctx, moduleName)
	ret0, _ := ret[0].(types0.ModuleAccountI)
	ret1, _ := ret[1].([]string)
	return ret0, ret1
}

// GetModuleAccountAndPermissions indicates an expected call of GetModuleAccountAndPermissions.
func (mr *MockAccountKeeperMockRecorder) GetModuleAccountAndPermissions(ctx, moduleName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModuleAccountAndPermissions", reflect.TypeOf((*MockAccountKeeper)(nil).GetModuleAccountAndPermissions), ctx, moduleName)
}

// GetModuleAddress mocks base method.
func (m *MockAccountKeeper) GetModuleAddress(moduleName string) types0.AccAddress {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModuleAddress", moduleName)
	ret0, _ := ret[0].(types0.AccAddress)
	return ret0
}

// GetModuleAddress indicates an expected call of GetModuleAddress.
func (mr *MockAccountKeeperMockRecorder) GetModuleAddress(moduleName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModuleAddress", reflect.TypeOf((*MockAccountKeeper)(nil).GetModuleAddress), moduleName)
}

// GetModuleAddressAndPermissions mocks base method.
func (m *MockAccountKeeper) GetModuleAddressAndPermissions(moduleName string) (types0.AccAddress, []string) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModuleAddressAndPermissions", moduleName)
	ret0, _ := ret[0].(types0.AccAddress)
	ret1, _ := ret[1].([]string)
	return ret0, ret1
}

// GetModuleAddressAndPermissions indicates an expected call of GetModuleAddressAndPermissions.
func (mr *MockAccountKeeperMockRecorder) GetModuleAddressAndPermissions(moduleName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModuleAddressAndPermissions", reflect.TypeOf((*MockAccountKeeper)(nil).GetModuleAddressAndPermissions), moduleName)
}

// GetModulePermissions mocks base method.
func (m *MockAccountKeeper) GetModulePermissions() map[string]types.PermissionsForAddress {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModulePermissions")
	ret0, _ := ret[0].(map[string]types.PermissionsForAddress)
	return ret0
}

// GetModulePermissions indicates an expected call of GetModulePermissions.
func (mr *MockAccountKeeperMockRecorder) GetModulePermissions() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModulePermissions", reflect.TypeOf((*MockAccountKeeper)(nil).GetModulePermissions))
}

// HasAccount mocks base method.
func (m *MockAccountKeeper) HasAccount(ctx context.Context, addr types0.AccAddress) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HasAccount", ctx, addr)
	ret0, _ := ret[0].(bool)
	return ret0
}

// HasAccount indicates an expected call of HasAccount.
func (mr *MockAccountKeeperMockRecorder) HasAccount(ctx, addr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HasAccount", reflect.TypeOf((*MockAccountKeeper)(nil).HasAccount), ctx, addr)
}

// IterateAccounts mocks base method.
func (m *MockAccountKeeper) IterateAccounts(ctx context.Context, process func(types0.AccountI) bool) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "IterateAccounts", ctx, process)
}

// IterateAccounts indicates an expected call of IterateAccounts.
func (mr *MockAccountKeeperMockRecorder) IterateAccounts(ctx, process interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IterateAccounts", reflect.TypeOf((*MockAccountKeeper)(nil).IterateAccounts), ctx, process)
}

// NewAccount mocks base method.
func (m *MockAccountKeeper) NewAccount(arg0 context.Context, arg1 types0.AccountI) types0.AccountI {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewAccount", arg0, arg1)
	ret0, _ := ret[0].(types0.AccountI)
	return ret0
}

// NewAccount indicates an expected call of NewAccount.
func (mr *MockAccountKeeperMockRecorder) NewAccount(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewAccount", reflect.TypeOf((*MockAccountKeeper)(nil).NewAccount), arg0, arg1)
}

// NewAccountWithAddress mocks base method.
func (m *MockAccountKeeper) NewAccountWithAddress(ctx context.Context, addr types0.AccAddress) types0.AccountI {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewAccountWithAddress", ctx, addr)
	ret0, _ := ret[0].(types0.AccountI)
	return ret0
}

// NewAccountWithAddress indicates an expected call of NewAccountWithAddress.
func (mr *MockAccountKeeperMockRecorder) NewAccountWithAddress(ctx, addr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewAccountWithAddress", reflect.TypeOf((*MockAccountKeeper)(nil).NewAccountWithAddress), ctx, addr)
}

// SetAccount mocks base method.
func (m *MockAccountKeeper) SetAccount(ctx context.Context, acc types0.AccountI) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetAccount", ctx, acc)
}

// SetAccount indicates an expected call of SetAccount.
func (mr *MockAccountKeeperMockRecorder) SetAccount(ctx, acc interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetAccount", reflect.TypeOf((*MockAccountKeeper)(nil).SetAccount), ctx, acc)
}

// SetModuleAccount mocks base method.
func (m *MockAccountKeeper) SetModuleAccount(ctx context.Context, macc types0.ModuleAccountI) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetModuleAccount", ctx, macc)
}

// SetModuleAccount indicates an expected call of SetModuleAccount.
func (mr *MockAccountKeeperMockRecorder) SetModuleAccount(ctx, macc interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetModuleAccount", reflect.TypeOf((*MockAccountKeeper)(nil).SetModuleAccount), ctx, macc)
}

// ValidatePermissions mocks base method.
func (m *MockAccountKeeper) ValidatePermissions(macc types0.ModuleAccountI) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ValidatePermissions", macc)
	ret0, _ := ret[0].(error)
	return ret0
}

// ValidatePermissions indicates an expected call of ValidatePermissions.
func (mr *MockAccountKeeperMockRecorder) ValidatePermissions(macc interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ValidatePermissions", reflect.TypeOf((*MockAccountKeeper)(nil).ValidatePermissions), macc)
}
