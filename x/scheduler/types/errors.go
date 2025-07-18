package types

import "cosmossdk.io/errors"

var (
	ErrInvalidTaskHandler   = errors.Register(ModuleName, 2, "invalid task handler")
	ErrInvalidTask          = errors.Register(ModuleName, 3, "invalid task")
	ErrInvalidTaskArguments = errors.Register(ModuleName, 4, "invalid task arguments")
)
