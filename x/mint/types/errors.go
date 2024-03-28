package types

import "cosmossdk.io/errors"

var ErrInvalidSigner = errors.Register(ModuleName, 1, "expected authority account as only signer for proposal message")
var ErrNegativeCirculatingSupply = errors.Register(ModuleName, 2, "negative circulating supply")
var ErrNegativeTargetEmissionPerToken = errors.Register(ModuleName, 3, "negative target emission per token")
