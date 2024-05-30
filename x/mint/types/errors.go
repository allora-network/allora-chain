package types

import "cosmossdk.io/errors"

var (
	ErrInvalidSigner = errors.Register(
		ModuleName,
		1,
		"expected authority account as only signer for proposal message",
	)
	ErrNegativeTargetEmissionPerToken = errors.Register(
		ModuleName,
		2,
		"negative target emission per token",
	)
	ErrInvalidPreviousRewardEmissionPerUnitStakedToken = errors.Register(
		ModuleName,
		3,
		"invalid previous reward",
	)
	ErrInvalidEcosystemTokensMinted = errors.Register(
		ModuleName,
		4,
		"invalid ecosystem tokens minted",
	)
	ErrZeroDenominator = errors.Register(
		ModuleName,
		5,
		"zero denominator",
	)
)
