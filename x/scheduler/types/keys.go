package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name.
	ModuleName = "scheduler"

	// StoreKey defines the primary module store key.
	StoreKey = ModuleName
)

var (
	TasksKeyPrefix       = collections.NewPrefix(0) // TasksKeyPrefix stores tasks.
	TasksByTypeKeyPrefix = collections.NewPrefix(1) // TasksByTypeKeyPrefix stores task ids per task type.
	TasksSchedulePrefix  = collections.NewPrefix(2) // TasksRunKeyPrefix stores tasks scheduled run.
)
