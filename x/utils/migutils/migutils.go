package migutils

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
)

// Deletes all keys in the store with the given keyPrefix `maxPageSize` keys at a time
func SafelyClearWholeMap(store storetypes.KVStore, keyPrefix []byte, maxPageSize uint64) error {
	s := prefix.NewStore(store, keyPrefix)

	// `clearPage` deletes `maxPageSize` keys at a time
	clearPage := func() (bool, error) {
		// Gather keys to eventually delete
		iterator := s.Iterator(nil, nil)
		defer iterator.Close()

		keysToDelete := make([][]byte, 0)
		count := uint64(0)
		for ; iterator.Valid(); iterator.Next() {
			if count >= maxPageSize {
				break
			}

			keysToDelete = append(keysToDelete, iterator.Key())
			count++
		}
		err := iterator.Close()
		if err != nil {
			return false, errorsmod.Wrap(err, "while closing iterator in `safelyClearWholeMap`")
		}

		// Delete the keys
		for _, key := range keysToDelete {
			s.Delete(key)
		}

		// If no keys to delete, break => Exit whole function
		more := len(keysToDelete) > 0
		return more, nil
	}

	// Loop until all keys are deleted.
	// Unbounded not best practice but we are sure that the number of keys will be limited
	// and not deleting all keys means "poison" will remain in the store.
	for {
		more, err := clearPage()
		if err != nil {
			return err
		} else if !more {
			break
		}
	}
	return nil
}
