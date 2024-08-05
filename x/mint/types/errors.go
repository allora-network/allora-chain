package types

import "cosmossdk.io/errors"

var (
	ErrUnauthorized                                    = errors.Register(ModuleName, 1, "unauthorized message signer")
	ErrNegativeTargetEmissionPerToken                  = errors.Register(ModuleName, 2, "negative target emission per token")
	ErrInvalidPreviousRewardEmissionPerUnitStakedToken = errors.Register(ModuleName, 3, "invalid previous reward")
	ErrInvalidEcosystemTokensMinted                    = errors.Register(ModuleName, 4, "invalid ecosystem tokens minted")
	ErrZeroDenominator                                 = errors.Register(ModuleName, 5, "zero denominator")
	ErrNegativeCirculatingSupply                       = errors.Register(ModuleName, 6, "circulating supply cannot be negative")
)
