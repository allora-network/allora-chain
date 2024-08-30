package testcommon

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

func LookupRpcMode(t *testing.T, key string, defaultValue RpcConnectionType) RpcConnectionType {
	t.Helper()
	value, found := os.LookupEnv(key)
	if !found {
		return defaultValue
	}
	if value == "SingleRpc" {
		return SingleRpc
	} else if value == "RoundRobin" {
		return RoundRobin
	} else if value == "RandomBasedOnDeterministicSeed" {
		return RandomBasedOnDeterministicSeed
	} else {
		t.Fatal("Unknown RpcConnectionType: ", value)
	}
	return defaultValue
}

func LookupEnvInt(t *testing.T, key string, defaultValue int) int {
	t.Helper()
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
	t.Helper()
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

func LookupEnvStringArray(key string, defaultValue []string) []string {
	value, found := os.LookupEnv(key)
	if !found {
		return defaultValue
	}
	valueArr := strings.Split(value, `,`)
	return valueArr
}
