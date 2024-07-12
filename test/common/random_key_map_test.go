package testcommon_test

import (
	"math/rand"
	"testing"

	testcommon "github.com/allora-network/allora-chain/test/common"
)

func TestRandomKeyMap_Upsert(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	rkm := testcommon.NewRandomKeyMap[int, string](r)
	// Insert some elements into the map
	keys := []int{1, 2, 3, 4, 5}
	values := []string{"one", "two", "three", "four", "five"}
	for i, key := range keys {
		rkm.Upsert(key, values[i])
	}
	// Verify that the inserted elements exist in the map
	for i, key := range keys {
		value, exists := rkm.Get(key)
		if !exists {
			t.Errorf("Expected key %d to exist in the map, but it doesn't", key)
		}
		if value != values[i] {
			t.Errorf("Expected value %s for key %d, but got %s", values[i], key, value)
		}
	}

	// Update an existing element
	keyToUpdate := 3
	newValue := "updated"
	rkm.Upsert(keyToUpdate, newValue)
	updatedValue, exists := rkm.Get(keyToUpdate)
	if !exists {
		t.Errorf("Expected key %d to exist in the map after update, but it doesn't", keyToUpdate)
	}
	if updatedValue != newValue {
		t.Errorf("Expected value %s for key %d after update, but got %s", newValue, keyToUpdate, updatedValue)
	}
	// Verify that the length of the map has stayed the same
	expectedLen := len(keys)
	actualLen := rkm.Len()
	if actualLen != expectedLen {
		t.Errorf("Expected map length to be %d after insertion, but got %d", expectedLen, actualLen)
	}

	// Insert a new element
	newKey := 6
	newValue = "six"
	rkm.Upsert(newKey, newValue)
	updatedValue, exists = rkm.Get(newKey)
	if !exists {
		t.Errorf("Expected key %d to exist in the map after insertion, but it doesn't", newKey)
	}
	if updatedValue != newValue {
		t.Errorf("Expected value %s for key %d after insertion, but got %s", newValue, newKey, updatedValue)
	}

	// Verify that the length of the map has increased by 1
	expectedLen = len(keys) + 1
	actualLen = rkm.Len()
	if actualLen != expectedLen {
		t.Errorf("Expected map length to be %d after insertion, but got %d", expectedLen, actualLen)
	}
}

func TestRandomKeyMap_Delete(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	rkm := testcommon.NewRandomKeyMap[int, string](r)

	// Insert some elements into the map
	keys := []int{1, 2, 3, 4, 5}
	values := []string{"one", "two", "three", "four", "five"}
	for i, key := range keys {
		rkm.Upsert(key, values[i])
	}

	// Delete an existing element
	keyToDelete := 3
	rkm.Delete(keyToDelete)

	// Verify that the deleted element is no longer in the map
	_, exists := rkm.Get(keyToDelete)
	if exists {
		t.Errorf("Expected key %d to be deleted, but it still exists in the map", keyToDelete)
	}

	// Verify that the length of the map has decreased by 1
	expectedLen := len(keys) - 1
	actualLen := rkm.Len()
	if actualLen != expectedLen {
		t.Errorf("Expected map length to be %d, but got %d", expectedLen, actualLen)
	}

	// Delete a non-existing element
	nonExistingKey := 6
	rkm.Delete(nonExistingKey)

	// Verify that the map remains unchanged
	actualLen = rkm.Len()
	if actualLen != expectedLen {
		t.Errorf("Expected map length to be %d, but got %d", expectedLen, actualLen)
	}
}

func TestRandomKeyMap_Get(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	rkm := testcommon.NewRandomKeyMap[int, string](r)
	// Insert some elements into the map
	keys := []int{1, 2, 3, 4, 5}
	values := []string{"one", "two", "three", "four", "five"}
	for i, key := range keys {
		rkm.Upsert(key, values[i])
	}
	// Get an existing element
	keyToGet := 3
	valueToGet := "three"
	value, exists := rkm.Get(keyToGet)
	if !exists {
		t.Errorf("Expected key %d to exist in the map, but it doesn't", keyToGet)
	}
	if value != valueToGet {
		t.Errorf("Expected value %s for key %d, but got %s", valueToGet, keyToGet, value)
	}
	// Get a non-existing element
	nonExistingKey := 6
	_, exists = rkm.Get(nonExistingKey)
	if exists {
		t.Errorf("Expected key %d to not exist in the map, but it does", nonExistingKey)
	}
}

func TestRandomKeyMap_RandomKey(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	rkm := testcommon.NewRandomKeyMap[int, string](r)
	// Insert some elements into the map
	keys := []int{1, 2, 3, 4, 5}
	values := []string{"one", "two", "three", "four", "five"}
	for i, key := range keys {
		rkm.Upsert(key, values[i])
	}
	// Get a random key from the map
	randomKeyPtr, err := rkm.RandomKey()
	if err != nil || randomKeyPtr == nil {
		t.Errorf("Expected to get a random key from the map, but got an error or nil pointer")
	}
	randomKey := *randomKeyPtr
	// Verify that the random key is one of the keys in the map
	found := false
	for _, key := range keys {
		if key == randomKey {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected random key to be one of %v, but got %v", keys, randomKey)
	}
}

func TestRandomKeyMap_Len(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	rkm := testcommon.NewRandomKeyMap[int, string](r)
	// Insert some elements into the map
	keys := []int{1, 2, 3, 4, 5}
	values := []string{"one", "two", "three", "four", "five"}
	for i, key := range keys {
		rkm.Upsert(key, values[i])
	}
	// Verify the initial length of the map
	expectedLen := len(keys)
	actualLen := rkm.Len()
	if actualLen != expectedLen {
		t.Errorf("Expected map length to be %d, but got %d", expectedLen, actualLen)
	}
	// Delete an element and verify the length decreases by 1
	keyToDelete := 3
	rkm.Delete(keyToDelete)
	expectedLen--
	actualLen = rkm.Len()
	if actualLen != expectedLen {
		t.Errorf("Expected map length to be %d, but got %d", expectedLen, actualLen)
	}
	// Insert a new element and verify the length increases by 1
	newKey := 6
	newValue := "six"
	rkm.Upsert(newKey, newValue)
	expectedLen++
	actualLen = rkm.Len()
	if actualLen != expectedLen {
		t.Errorf("Expected map length to be %d, but got %d", expectedLen, actualLen)
	}
}

func TestRandomKeyMap_GetAll(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	rkm := testcommon.NewRandomKeyMap[int, string](r)
	// Insert some elements into the map
	keys := []int{1, 2, 3, 4, 5}
	values := []string{"one", "two", "three", "four", "five"}
	for i, key := range keys {
		rkm.Upsert(key, values[i])
	}
	// Get all elements from the map
	allValues := rkm.GetAll()
	// Verify that the length of the returned slice is equal to the length of the map
	expectedLen := len(keys)
	actualLen := len(allValues)
	if actualLen != expectedLen {
		t.Errorf("Expected slice length to be %d, but got %d", expectedLen, actualLen)
	}
	// Verify that all values in the returned slice exist in the map
	for _, value := range allValues {
		found := false
		for i := 0; i < len(values); i++ {
			if value == values[i] {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected value %s to exist in the map, but it doesn't", value)
		}
	}
}

func TestRandomKeyMap_Filter(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	rkm := testcommon.NewRandomKeyMap[int, string](r)
	// Insert some elements into the map
	keys := []int{1, 2, 3, 4, 5}
	values := []string{"one", "two", "three", "four", "five"}
	for i, key := range keys {
		rkm.Upsert(key, values[i])
	}
	// Filter the map to get elements with even keys
	filteredKeys, filteredValues := rkm.Filter(func(k int) bool {
		return k%2 == 0
	})
	// Verify that the length of the filtered slice is correct
	expectedLen := 2
	actualLenKeys := len(filteredKeys)
	actualLenValues := len(filteredValues)
	if actualLenKeys != expectedLen || actualLenValues != expectedLen {
		t.Errorf("Expected filtered slice length to be %d, but got %d, %d", expectedLen, actualLenKeys, actualLenValues)
	}
	// verify that the filtered keys are correct
	expectedKeys := []int{2, 4}
	for i, key := range filteredKeys {
		if key != expectedKeys[i] {
			t.Errorf("Expected filtered key at index %d to be %d, but got %d", i, expectedKeys[i], key)
		}
	}
	// Verify that the filtered values are correct
	expectedValues := []string{"two", "four"}
	for i, value := range filteredValues {
		if value != expectedValues[i] {
			t.Errorf("Expected filtered value at index %d to be %s, but got %s", i, expectedValues[i], value)
		}
	}
	// Filter the map to get elements with keys greater than 5
	filteredKeys, filteredValues = rkm.Filter(func(k int) bool {
		return k > 5
	})
	// Verify that the length of the filtered slice is correct
	expectedLen = 0
	actualLenKeys = len(filteredKeys)
	actualLenValues = len(filteredValues)
	if actualLenKeys != expectedLen || actualLenValues != expectedLen {
		t.Errorf("Expected filtered slice length to be %d, but got %d, %d", expectedLen, actualLenKeys, actualLenValues)
	}
}
