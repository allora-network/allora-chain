package module

import (
	"math/big"
	"testing"
	"time"

	"cosmossdk.io/core/header"
	cosmosMath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/app/params"
	state "github.com/allora-network/allora-chain/x/emissions"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/stretchr/testify/suite"
)

type RewardsCalcTestSuite struct {
	suite.Suite

	ctx             sdk.Context
	authKeeper      keeper.AccountKeeper
	bankKeeper      keeper.BankKeeper
	emissionsKeeper keeper.Keeper
	key             *storetypes.KVStoreKey
}

func (s *RewardsCalcTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey("emissions")
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithHeaderInfo(header.Info{Time: time.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig()
	addressCodec := address.NewBech32Codec(params.Bech32PrefixAccAddr)

	s.ctx = ctx
	s.emissionsKeeper = keeper.NewKeeper(encCfg.Codec, addressCodec, storeService, s.authKeeper, s.bankKeeper)
	s.key = key
}

func TestRewardsCalcTestSuite(t *testing.T) {
	suite.Run(t, new(RewardsCalcTestSuite))
}

func (s *RewardsCalcTestSuite) TestSumMapValuesEmptyMap() {
	values := make(map[string]*Float)
	expectedSum := *(big.NewFloat(0))

	result := sumMapValues(values)

	s.Require().Equal(expectedSum, result, "Sum function returned incorrect result: expected 0, got %s", result.String())
}

func (s *RewardsCalcTestSuite) TestSumMapValuesSingle() {
	values := map[string]*Float{
		"address1": big.NewFloat(1.5),
	}

	expectedSum := *(big.NewFloat(1.5))

	result := sumMapValues(values)

	s.Require().Equal(expectedSum, result, "Sum function returned incorrect result: expected 1.5, got %s", result.String())
}

func (s *RewardsCalcTestSuite) TestSumMapValuesMultiple() {
	// Create a map of values for testing
	values := map[string]*Float{
		"address1": big.NewFloat(1.5),
		"address2": big.NewFloat(2.5),
		"address3": big.NewFloat(3.5),
	}

	// Calculate the expected sum
	expectedSum := *(big.NewFloat(7.5))

	// Call the sum function
	result := sumMapValues(values)

	// Check if the result matches the expected sum

	s.Require().Equal(expectedSum, result, "Sum function returned incorrect result")
}

func (s *RewardsCalcTestSuite) TestScalarMultiply() {
	matrix := map[string]*Float{
		"address1": big.NewFloat(1.5),
		"address2": big.NewFloat(2.5),
		"address3": big.NewFloat(3.5),
		"address4": big.NewFloat(4.5),
	}

	var scalar *Float = big.NewFloat(2.3)

	e1 := cosmosMath.NewUint(3)  // 3.45
	e2 := cosmosMath.NewUint(5)  // 5.75
	e3 := cosmosMath.NewUint(8)  // 8.05
	e4 := cosmosMath.NewUint(10) // 10.35
	expected := map[string]*Uint{
		"address1": &e1,
		"address2": &e2,
		"address3": &e3,
		"address4": &e4,
	}

	result, err := scalarMultiply(matrix, scalar)

	s.Require().NoError(err, "ScalarMultiply returned an error")
	s.Require().Equal(expected, result, "ScalarMultiply returned incorrect result")
}

func (s *RewardsCalcTestSuite) TestScalarMultiplyFailNegative() {
	matrix := map[string]*Float{
		"address1": big.NewFloat(1.5),
	}

	var scalar *Float = big.NewFloat(-2)

	var expected map[string]*Uint = nil

	result, err := scalarMultiply(matrix, scalar)
	s.Require().Equal(state.ErrScalarMultiplyNegative, err, "ScalarMultiply returned incorrect error")
	s.Require().Equal(expected, result, "ScalarMultiply returned incorrect result")
}

func (s *RewardsCalcTestSuite) TestNormalizeEmptyMap() {
	matrix := make(map[string]*Float)

	expected := make(map[string]*Float)

	result, err := normalize(matrix)
	s.Require().NoError(err, "Normalize returned an error")
	s.Require().Equal(expected, result, "Normalize returned incorrect result")
}

func (s *RewardsCalcTestSuite) TestNormalizeSingle() {
	matrix := map[string]*Float{
		"address1": big.NewFloat(1),
	}

	expected := map[string]*Float{
		"address1": big.NewFloat(1),
	}

	result, err := normalize(matrix)
	s.Require().NoError(err, "Normalize returned an error")
	resetMapFloatAccuracies(result)
	s.Require().Equal(expected, result, "Normalize returned incorrect result")
}

func (s *RewardsCalcTestSuite) TestNormalizeSingleZero() {
	matrix := map[string]*Float{
		"address1": big.NewFloat(0),
	}

	result, err := normalize(matrix)
	var expected map[string]*Float = nil
	s.Require().Equal(err, state.ErrDivideMapValuesByZero)
	s.Require().Equal(expected, result, "Normalize returned incorrect result")
}

func (s *RewardsCalcTestSuite) TestNormalizeMultipleZero() {
	matrix := map[string]*Float{
		"address1": big.NewFloat(0),
		"address2": big.NewFloat(0),
		"address3": big.NewFloat(0),
		"address4": big.NewFloat(0),
	}

	result, err := normalize(matrix)
	var expected map[string]*Float = nil
	s.Require().Equal(err, state.ErrDivideMapValuesByZero)
	s.Require().Equal(expected, result, "Normalize returned incorrect result")
}

func (s *RewardsCalcTestSuite) TestNormalizeMultiple() {
	// 1+2+3+4 sum equals 10
	matrix := map[string]*Float{
		"address1": big.NewFloat(1),
		"address2": big.NewFloat(2),
		"address3": big.NewFloat(3),
		"address4": big.NewFloat(4),
	}

	// since normalize divides each value by its sum, the expected result is
	// the value divided by 10
	expected := map[string]*Float{
		"address1": big.NewFloat(0.1),
		"address2": big.NewFloat(0.2),
		"address3": big.NewFloat(0.3),
		"address4": big.NewFloat(0.4),
	}

	result, err := normalize(matrix)
	s.Require().NoError(err, "Normalize returned an error")
	resetMapFloatAccuracies(result)
	s.Require().Equal(expected, result, "Normalize returned incorrect result")
}

func (s *RewardsCalcTestSuite) TestNormalizeBondDeltasSimple() {
	row1 := map[string]*Float{
		"address1": big.NewFloat(1),
		"address2": big.NewFloat(3),
	}
	row2 := map[string]*Float{
		"address3": big.NewFloat(5),
		"address4": big.NewFloat(5),
	}
	sampleBondDeltas := map[string]map[string]*Float{
		"address5": row1,
		"address6": row2,
	}

	expectedRow1 := map[string]*Float{
		"address1": big.NewFloat(0.25),
		"address2": big.NewFloat(0.75),
	}
	expectedRow2 := map[string]*Float{
		"address3": big.NewFloat(0.5),
		"address4": big.NewFloat(0.5),
	}
	expected := map[string]map[string]*Float{
		"address5": expectedRow1,
		"address6": expectedRow2,
	}

	result, err := normalizeBondDeltas(sampleBondDeltas)

	s.Require().NoError(err, "NormalizeBondDeltas returned an error")
	resetDoubleMapFloatAccuracies(result)
	s.Require().Equal(expected, result, "NormalizeBondDeltas returned incorrect result")
}

func (s *RewardsCalcTestSuite) TestNormalizeBondDeltasEmpty() {
	sampleBondDeltas := make(map[string]map[string]*Float)

	expected := make(map[string]map[string]*Float)

	result, err := normalizeBondDeltas(sampleBondDeltas)

	s.Require().NoError(err, "NormalizeBondDeltas returned an error")
	s.Require().Equal(expected, result, "NormalizeBondDeltas returned incorrect result")
}

func (s *RewardsCalcTestSuite) TestNormalizeBondDeltasFailZero() {
	sampleBondDeltas := map[string]map[string]*Float{
		"address1": {
			"address2": big.NewFloat(1),
			"address3": big.NewFloat(2),
		},
		"address4": {
			"address5": big.NewFloat(0),
			"address6": big.NewFloat(0),
		},
	}

	var expected map[string]map[string]*Float = nil

	result, err := normalizeBondDeltas(sampleBondDeltas)

	s.Require().Equal(err, state.ErrDivideMapValuesByZero)
	s.Require().Equal(expected, result, "NormalizeBondDeltas returned incorrect result")
}

func (s *RewardsCalcTestSuite) TestElementWiseProductSimple() {
	aUint := cosmosMath.NewUint(2)
	matrix := map[string]map[string]*Uint{
		"address1": {
			"address2": &aUint,
			"address3": &aUint,
		},
		"address4": {
			"address5": &aUint,
			"address6": &aUint,
		},
	}

	vector := map[string]*Float{
		"address1": big.NewFloat(3),
		"address4": big.NewFloat(4),
	}

	expected := map[string]map[string]*Float{
		"address1": {
			"address2": big.NewFloat(6),
			"address3": big.NewFloat(6),
		},
		"address4": {
			"address5": big.NewFloat(8),
			"address6": big.NewFloat(8),
		},
	}

	result := elementWiseProduct(matrix, vector)

	resetDoubleMapFloatAccuracies(result)
	s.Require().Equal(expected, result, "ElementWiseProduct returned incorrect result")
}

func (s *RewardsCalcTestSuite) TestElementWiseProductVectorEmpty() {
	aUint := cosmosMath.NewUint(2)
	matrix := map[string]map[string]*Uint{
		"address1": {
			"address2": &aUint,
			"address3": &aUint,
		},
		"address4": {
			"address5": &aUint,
			"address6": &aUint,
		},
	}

	vector := make(map[string]*Float)

	expected := map[string]map[string]*Float{
		"address1": {},
		"address4": {},
	}

	result := elementWiseProduct(matrix, vector)
	s.Require().Equal(expected, result, "ElementWiseProduct returned incorrect result")
}

func (s *RewardsCalcTestSuite) TestElementWiseProductMatrixEmpty() {
	matrix := make(map[string]map[string]*Uint)
	vector := map[string]*Float{
		"address1": big.NewFloat(3),
		"address4": big.NewFloat(4),
	}

	expected := make(map[string]map[string]*Float)

	result := elementWiseProduct(matrix, vector)

	s.Require().Equal(expected, result, "ElementWiseProduct returned incorrect result")
}

func (s *RewardsCalcTestSuite) TestElementWiseProductVectorZero() {
	aUint := cosmosMath.NewUint(2)
	matrix := map[string]map[string]*Uint{
		"address1": {
			"address2": &aUint,
			"address3": &aUint,
		},
		"address4": {
			"address5": &aUint,
			"address6": &aUint,
		},
	}

	vector := map[string]*Float{
		"address1": big.NewFloat(3),
		"address4": big.NewFloat(0),
	}

	expected := map[string]map[string]*Float{
		"address1": {
			"address2": big.NewFloat(6),
			"address3": big.NewFloat(6),
		},
		"address4": {
			"address5": big.NewFloat(0),
			"address6": big.NewFloat(0),
		},
	}

	result := elementWiseProduct(matrix, vector)

	resetDoubleMapFloatAccuracies(result)
	s.Require().Equal(expected, result, "ElementWiseProduct returned incorrect result")
}

func (s *RewardsCalcTestSuite) TestMatMulFloatSimple() {

	// vector = { 1, 2 }
	// matrix = { { 1, 2, 3 }, { 4, 5, 6 } }
	// output = { 1*1 + 2*4, 1*2 + 2*5, 1*3 + 6*2}
	// output = { 9, 12, 15 }
	matrix := map[string]map[string]*Float{
		"address1": {
			"address2": big.NewFloat(1),
			"address3": big.NewFloat(2),
			"address4": big.NewFloat(3),
		},
		"address5": {
			"address2": big.NewFloat(4),
			"address3": big.NewFloat(5),
			"address4": big.NewFloat(6),
		},
	}

	vector := map[string]*Float{
		"address1": big.NewFloat(1),
		"address5": big.NewFloat(2),
	}

	expected := map[string]*Float{
		"address2": big.NewFloat(9),
		"address3": big.NewFloat(12),
		"address4": big.NewFloat(15),
	}

	result := matmul(matrix, vector)
	resetMapFloatAccuracies(result)
	s.Require().Equal(expected, result, "MatMul returned incorrect result")
}

func (s *RewardsCalcTestSuite) TestMatMulFloatEmpty() {
	matrix := map[string]map[string]*Float{
		"address1": {
			"address2": big.NewFloat(1),
			"address3": big.NewFloat(2),
			"address4": big.NewFloat(3),
		},
		"address5": {
			"address2": big.NewFloat(4),
			"address3": big.NewFloat(5),
			"address4": big.NewFloat(6),
		},
	}

	vector := make(map[string]*Float)
	expected := make(map[string]*Float)

	result := matmul(matrix, vector)
	s.Require().Equal(expected, result, "MatMul returned incorrect result")
}

func (s *RewardsCalcTestSuite) TestMatMulUintSimple() {

	// vector = { 8, 6 }
	// matrix = { { 4, 1, 2 }, { 3, 7, 9 } }
	// output = { 8*4 + 6*3, 8*1 + 6*7, 8*2 + 6*9}
	// output = { 50, 50, 70 }
	m1 := cosmosMath.NewUint(4)
	m2 := cosmosMath.NewUint(1)
	m3 := cosmosMath.NewUint(2)
	m4 := cosmosMath.NewUint(3)
	m5 := cosmosMath.NewUint(7)
	m6 := cosmosMath.NewUint(9)
	matrix := map[string]map[string]*cosmosMath.Uint{
		"address1": {
			"address2": &m1,
			"address3": &m2,
			"address4": &m3,
		},
		"address5": {
			"address2": &m4,
			"address3": &m5,
			"address4": &m6,
		},
	}

	vector := map[string]*Float{
		"address1": big.NewFloat(8),
		"address5": big.NewFloat(6),
	}

	expected := map[string]*Float{
		"address2": big.NewFloat(50),
		"address3": big.NewFloat(50),
		"address4": big.NewFloat(70),
	}

	result := matmul(matrix, vector)
	resetMapFloatAccuracies(result)
	s.Require().Equal(expected, result, "MatMul returned incorrect result")
}

func (s *RewardsCalcTestSuite) TestMapAddSimple() {
	val1 := cosmosMath.NewUint(1)
	val2 := cosmosMath.NewUint(2)
	a := map[string]*Uint{
		"a": &val1,
		"b": &val1,
		"c": &val1,
	}
	b := map[string]*Uint{
		"a": &val1,
		"b": &val2,
		"d": &val2,
	}

	expec1 := cosmosMath.NewUint(2)
	expec2 := cosmosMath.NewUint(3)
	expec3 := cosmosMath.NewUint(1)
	expected := map[string]*Uint{
		"a": &expec1,
		"b": &expec2,
		"c": &expec3,
		"d": &expec1,
	}

	result := mapAdd(a, b)

	s.Require().Equal(expected, result, "MapAdd returned incorrect result")
}

func (s *RewardsCalcTestSuite) TestMapAddEmptyA() {
	a := make(map[string]*Uint)
	val1 := cosmosMath.NewUint(1)
	val2 := cosmosMath.NewUint(2)
	b := map[string]*Uint{
		"a": &val1,
		"b": &val2,
		"d": &val2,
	}

	expected := map[string]*Uint{
		"a": &val1,
		"b": &val2,
		"d": &val2,
	}

	result := mapAdd(a, b)

	s.Require().Equal(expected, result, "MapAdd returned incorrect result")
}

func (s *RewardsCalcTestSuite) TestMapAddEmptyB() {
	val1 := cosmosMath.NewUint(1)
	a := map[string]*Uint{
		"a": &val1,
		"b": &val1,
		"c": &val1,
	}
	b := make(map[string]*Uint)

	expected := map[string]*Uint{
		"a": &val1,
		"b": &val1,
		"c": &val1,
	}

	result := mapAdd(a, b)

	s.Require().Equal(expected, result, "MapAdd returned incorrect result")
}

func (s *RewardsCalcTestSuite) TestMapAddEmptyBoth() {
	a := make(map[string]*Uint)
	b := make(map[string]*Uint)
	expected := make(map[string]*Uint)
	result := mapAdd(a, b)
	s.Require().Equal(expected, result, "MapAdd returned incorrect result")
}

/*************************************************/
/*      Helper functions for testing             */
/*************************************************/

// Big.Float keeps an Accuracy field which contains an enum
// reporting whether the conversion to float was exact or not
// here we reset the accuracy to the default of Exact
// regardless of what it was before, so that unit tests
// can compare the results of normalize
// since we don't do any checking of the accuracy anywhere else
// this is probably fine, but we should some day consider
// the effect of precision loss on the accuracy of the results
func resetMapFloatAccuracies(m map[string]*Float) {
	for key, value := range m {
		m[key] = value.SetMode(value.Mode())
	}
}

// see comment on resetMapFloatAccuracies
func resetDoubleMapFloatAccuracies(m map[string]map[string]*Float) {
	for key, value := range m {
		resetMapFloatAccuracies(value)
		m[key] = value
	}
}
