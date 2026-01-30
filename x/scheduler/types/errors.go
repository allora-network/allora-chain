package types

import "cosmossdk.io/errors"

var (
	ErrInvalidTaskHandler       = errors.Register(ModuleName, 2, "invalid task handler")
	ErrInvalidTask              = errors.Register(ModuleName, 3, "invalid task")
	ErrInvalidTaskArguments     = errors.Register(ModuleName, 4, "invalid task arguments")
	ErrTaskArbitrage            = errors.Register(ModuleName, 5, "task arbitrage failed")
	ErrTaskExecution            = errors.Register(ModuleName, 6, "task execution failed")
	ErrTaskAlreadyExists        = errors.Register(ModuleName, 7, "task already exists")
	ErrTaskHandlerAlreadyExists = errors.Register(ModuleName, 8, "task handler already exists")
)
