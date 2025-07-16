package types

import "cosmossdk.io/collections/codec"

var (
	TaskIDKey = codec.NewStringKeyCodec[TaskID]()
)

// TaskID denotes a unique identifier for a scheduled task, only one task with a given ID can be scheduled at a time,
// but an identifier may be reused later one.
type TaskID string

// Equal implements the gogo proto custom type interface.
func (t TaskID) Equal(other TaskID) bool {
	return t == other
}

// Size implements the gogo proto custom type interface.
func (t TaskID) Size() int {
	return len(t)
}

// MarshalTo implements the gogo proto custom type interface.
func (t TaskID) MarshalTo(to []byte) (int, error) {
	return copy(to, t), nil
}

// Unmarshal implements the gogo proto custom type interface.
func (t *TaskID) Unmarshal(from []byte) error {
	*t = TaskID(from)
	return nil
}
