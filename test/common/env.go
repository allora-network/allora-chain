package testcommon

import (
	"os"
	"strconv"
	"testing"
)

func LookupEnvInt(t *testing.T, key string, defaultValue int) int {
	value, found := os.LookupEnv(key)
	if !found {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		t.Fatal("Error converting string to int: ", err)
	}
	return intValue
}

func LookupEnvBool(t *testing.T, key string, defaultValue bool) bool {
	value, found := os.LookupEnv(key)
	if !found {
		return defaultValue
	}
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		t.Fatal("Error converting string to bool: ", err)
	}
	return boolValue
}
