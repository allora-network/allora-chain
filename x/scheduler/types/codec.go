package types

import "cosmossdk.io/collections/codec"

var (
	TaskIDKey = codec.NewStringKeyCodec[TaskID]()
)
