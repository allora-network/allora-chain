package types_test

import (
	"testing"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/stretchr/testify/suite"
)

type ValidationsTestSuite struct {
	suite.Suite
}

func TestValidationsTestSuite(t *testing.T) {
	suite.Run(t, new(ValidationsTestSuite))
}

func (s *ValidationsTestSuite) TestValidateIncomingDataDec() {
	testCases := []struct {
		name        string
		value       string
		exponent    uint64
		expectError bool
	}{
		// Zero value tests
		{
			name:        "zero value is valid",
			value:       "0",
			exponent:    40,
			expectError: false,
		},

		// Positive boundary tests
		{
			name:        "positive value at max boundary (1e+40)",
			value:       "1e+40",
			exponent:    40,
			expectError: false,
		},
		{
			name:        "positive value above max boundary (1e+41)",
			value:       "1e+41",
			exponent:    40,
			expectError: true,
		},
		{
			name:        "positive value below max boundary (1e+39)",
			value:       "1e+39",
			exponent:    40,
			expectError: false,
		},

		// Negative boundary tests
		{
			name:        "negative value at min boundary (-1e+40)",
			value:       "-1e+40",
			exponent:    40,
			expectError: false,
		},
		{
			name:        "negative value below min boundary (-1e+41)",
			value:       "-1e+41",
			exponent:    40,
			expectError: true,
		},
		{
			name:        "negative value above min boundary (-1e+39)",
			value:       "-1e+39",
			exponent:    40,
			expectError: false,
		},

		// Precision boundary tests
		{
			name:        "positive value at min precision boundary (1e-40)",
			value:       "1e-40",
			exponent:    40,
			expectError: false,
		},
		{
			name:        "positive value below min precision boundary (1e-41)",
			value:       "1e-41",
			exponent:    40,
			expectError: true,
		},
		{
			name:        "negative value at min precision boundary (-1e-40)",
			value:       "-1e-40",
			exponent:    40,
			expectError: false,
		},
		{
			name:        "negative value below min precision boundary (-1e-41)",
			value:       "-1e-41",
			exponent:    40,
			expectError: true,
		},

		// Normal value tests
		{
			name:        "normal positive decimal",
			value:       "123.456",
			exponent:    40,
			expectError: false,
		},
		{
			name:        "normal negative decimal",
			value:       "-123.456",
			exponent:    40,
			expectError: false,
		},

		// Extreme value tests
		{
			name:        "extremely large positive value",
			value:       "1e+100000",
			exponent:    40,
			expectError: true,
		},
		{
			name:        "extremely large negative value",
			value:       "-1e+100000",
			exponent:    40,
			expectError: true,
		},
		{
			name:        "extremely small positive value",
			value:       "1e-100000",
			exponent:    40,
			expectError: true,
		},
		{
			name:        "extremely small negative value",
			value:       "-1e-100000",
			exponent:    40,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			params := types.Params{
				DataLimitExponent: tc.exponent,
			}

			value, err := alloraMath.NewDecFromString(tc.value)
			s.Require().NoError(err, "Error creating decimal from string")

			err = types.ValidateIncomingDataDec(value, params)

			if tc.expectError {
				s.Require().Error(err, "Expected validation error for value %s", tc.value)
			} else {
				s.Require().NoError(err, "Expected no validation error for value %s", tc.value)
			}
		})
	}
}
