package types

import "cosmossdk.io/collections"

// these might need to be unique across the whole module space
// certain tests were failing in weird ways when they were not
// e.g. emissions starts at 0, so maybe there was a conflict
// with using the same integer for the keys for multiple keepers
var (
	ParamsKey                                              = collections.NewPrefix(138)
	PreviousRewardEmissionPerUnitStakedTokenNumeratorKey   = collections.NewPrefix(139)
	PreviousRewardEmissionPerUnitStakedTokenDenominatorKey = collections.NewPrefix(140)
	EcosystemTokensMintedKey                               = collections.NewPrefix(141)
)

const (
	// module name
	ModuleName = "mint"
	// ecosystem module account name
	EcosystemModuleName = "ecosystem"

	// StoreKey is the default store key for mint
	StoreKey = ModuleName

	// GovModuleName duplicates the gov module's name to avoid a cyclic dependency with x/gov.
	// It should be synced with the gov module's name if it is ever changed.
	// See: https://github.com/cosmos/cosmos-sdk/blob/b62a28aac041829da5ded4aeacfcd7a42873d1c8/x/gov/types/keys.go#L9
	GovModuleName = "gov"
)
