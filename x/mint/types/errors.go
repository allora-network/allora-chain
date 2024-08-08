package types

import "cosmossdk.io/errors"

var (
	// ERROR 1 IS RESERVED BY COSMOS-SDK PACKAGE
	ErrNegativeTargetEmissionPerToken                  = errors.Register(ModuleName, 2, "negative target emission per token")
	ErrInvalidPreviousRewardEmissionPerUnitStakedToken = errors.Register(ModuleName, 3, "invalid previous reward")
	ErrInvalidEcosystemTokensMinted                    = errors.Register(ModuleName, 4, "invalid ecosystem tokens minted")
	ErrZeroDenominator                                 = errors.Register(ModuleName, 5, "zero denominator")
	ErrNegativeCirculatingSupply                       = errors.Register(ModuleName, 6, "circulating supply cannot be negative")
	ErrNotFound                                        = errors.Register(ModuleName, 7, "not found")
	ErrUnauthorized                                    = errors.Register(ModuleName, 8, "unauthorized message signer")
)
