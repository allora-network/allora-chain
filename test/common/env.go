package testcommon

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

// helper function to convert a string to a RpcConnectionType
func StringToRpcConnectionType(t *testing.T, value string) RpcConnectionType {
	t.Helper()
	if value == "SingleRpc" {
		return SingleRpc
	} else if value == "RoundRobin" {
		return RoundRobin
	} else if value == "RandomBasedOnDeterministicSeed" {
		return RandomBasedOnDeterministicSeed
	} else {
		t.Fatal("Unknown RpcConnectionType: ", value)
	}
	return SingleRpc
}

// helper function to look up the rpc mode from the environment variable key
func LookupRpcMode(t *testing.T, key string) (rpcMode RpcConnectionType, found bool) {
	t.Helper()
	value, found := os.LookupEnv(key)
	if !found {
		return SingleRpc, false
	}
	rpcMode = StringToRpcConnectionType(t, value)
	return rpcMode, true
}

// helper function to look up the rpc mode from the environment variable key
// and return a passed-in default value if the environment variable is not set
func LookupRpcModeWithDefault(t *testing.T, key string, defaultValue RpcConnectionType) RpcConnectionType {
	t.Helper()
	rpcMode, found := LookupRpcMode(t, key)
	if !found {
		return defaultValue
	}
	return rpcMode
}

// helper function to look up an integer from the environment variable key
func LookupEnvInt(t *testing.T, key string) (envInt int, found bool) {
	t.Helper()
	value, found := os.LookupEnv(key)
	if !found {
		return 0, false
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		t.Fatal("Error converting string to int: ", err)
	}
	return intValue, true
}

// helper function to look up an integer from the environment variable key
// and return a passed-in default value if the environment variable is not set
func LookupEnvIntWithDefault(t *testing.T, key string, defaultValue int) int {
	t.Helper()
	value, found := LookupEnvInt(t, key)
	if !found {
		return defaultValue
	}
	return value
}

// helper function to look up a boolean from the environment variable key
func LookupEnvBool(t *testing.T, key string) (envBool bool, found bool) {
	t.Helper()
	value, found := os.LookupEnv(key)
	if !found {
		return false, false
	}
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		t.Fatal("Error converting string to bool: ", err)
	}
	return boolValue, true
}

// helper function to look up a boolean from the environment variable key
// and return a passed-in default value if the environment variable is not set
func LookupEnvBoolWithDefault(t *testing.T, key string, defaultValue bool) bool {
	t.Helper()
	value, found := LookupEnvBool(t, key)
	if !found {
		return defaultValue
	}
	return value
}

// helper function to look up a string array from the environment variable key
func LookupEnvStringArray(t *testing.T, key string) (envStringArray []string, found bool) {
	t.Helper()
	value, found := os.LookupEnv(key)
	if !found {
		return nil, false
	}
	valueArr := strings.Split(value, `,`)
	return valueArr, true
}

// helper function to look up a string array from the environment variable key
// and return a passed-in default value if the environment variable is not set
func LookupEnvStringArrayWithDefault(t *testing.T, key string, defaultValue []string) []string {
	t.Helper()
	envStringArray, found := LookupEnvStringArray(t, key)
	if !found {
		return defaultValue
	}
	return envStringArray
}
