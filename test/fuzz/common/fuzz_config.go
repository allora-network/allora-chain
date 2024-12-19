package fuzzcommon

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	testcommon "github.com/allora-network/allora-chain/test/common"
)

type SimulationMode string

const (
	Behave    SimulationMode = "behave"
	Fuzz      SimulationMode = "fuzz"
	Alternate SimulationMode = "alternate"
	Manual    SimulationMode = "manual"
)

// full list of all possible transitions
type TransitionWeights struct {
	CreateTopic                     uint8 `json:"createTopic"`
	FundTopic                       uint8 `json:"fundTopic"`
	RegisterWorker                  uint8 `json:"registerWorker"`
	RegisterReputer                 uint8 `json:"registerReputer"`
	StakeAsReputer                  uint8 `json:"stakeAsReputer"`
	DelegateStake                   uint8 `json:"delegateStake"`
	CollectDelegatorRewards         uint8 `json:"collectDelegatorRewards"`
	DoInferenceAndReputation        uint8 `json:"doInferenceAndReputation"`
	UnregisterWorker                uint8 `json:"unregisterWorker"`
	UnregisterReputer               uint8 `json:"unregisterReputer"`
	UnstakeAsReputer                uint8 `json:"unstakeAsReputer"`
	UndelegateStake                 uint8 `json:"undelegateStake"`
	CancelStakeRemoval              uint8 `json:"cancelStakeRemoval"`
	CancelDelegateStakeRemoval      uint8 `json:"cancelDelegateStakeRemoval"`
	AddToAdminWhitelist             uint8 `json:"addToAdminWhitelist"`
	RemoveFromAdminWhitelist        uint8 `json:"removeFromAdminWhitelist"`
	AddToGlobalWhitelist            uint8 `json:"addToGlobalWhitelist"`
	RemoveFromGlobalWhitelist       uint8 `json:"removeFromGlobalWhitelist"`
	AddToTopicCreatorWhitelist      uint8 `json:"addToTopicCreatorWhitelist"`
	RemoveFromTopicCreatorWhitelist uint8 `json:"removeFromTopicCreatorWhitelist"`
	EnableTopicWorkerWhitelist      uint8 `json:"enableTopicWorkerWhitelist"`
	DisableTopicWorkerWhitelist     uint8 `json:"disableTopicWorkerWhitelist"`
	EnableTopicReputerWhitelist     uint8 `json:"enableTopicReputerWhitelist"`
	DisableTopicReputerWhitelist    uint8 `json:"disableTopicReputerWhitelist"`
	AddToTopicWorkerWhitelist       uint8 `json:"addToTopicWorkerWhitelist"`
	RemoveFromTopicWorkerWhitelist  uint8 `json:"removeFromTopicWorkerWhitelist"`
	AddToTopicReputerWhitelist      uint8 `json:"addToTopicReputerWhitelist"`
	RemoveFromTopicReputerWhitelist uint8 `json:"removeFromTopicReputerWhitelist"`
}

// InitialSetup holds configuration elements for the initial setup
type InitialSetup struct {
	NumAdminWhitelist        int `json:"numAdminWhitelist"`
	NumGlobalWhitelist       int `json:"numGlobalWhitelist"`
	NumTopicCreatorWhitelist int `json:"numTopicCreatorWhitelist"`
}

// struct that holds config from test/fuzz/.config.json
type FuzzConfigJson struct {
	TransitionWeights TransitionWeights `json:"transitionWeights"`
	Seed              int               `json:"seed"`
	RpcMode           string            `json:"rpcMode"`
	RpcUrls           []string          `json:"rpcUrls"`
	MaxIterations     int               `json:"maxIterations"`
	Mode              SimulationMode    `json:"mode"`
	AlternateWeight   int               `json:"alternateWeight"`
	NumActors         int               `json:"numActors"`
	EpochLength       int               `json:"epochLength"`
	InitialSetup      InitialSetup      `json:"initialSetup"`
}

// struct that holds the config for fuzz tests
// which includes slightly different config from integration/stress tests
type FuzzConfig struct {
	Seed              int
	RpcMode           testcommon.RpcConnectionType
	RpcEndpoints      []string
	MaxIterations     int
	NumActors         int
	EpochLength       int
	Mode              SimulationMode
	AlternateWeight   int
	TestConfig        *testcommon.TestConfig
	TransitionWeights TransitionWeights
	InitialSetup      InitialSetup
}

// helper function to look up the simulation mode from the environment variable key
func lookupEnvSimulationMode() (SimulationMode, bool) {
	simulationModeStr, found := os.LookupEnv("MODE")
	if !found {
		return Behave, false
	}
	simulationModeStr = strings.ToLower(simulationModeStr)
	switch simulationModeStr {
	case "behave":
		return Behave, true
	case "fuzz":
		return Fuzz, true
	case "alternate":
		return Alternate, true
	case "manual":
		return Manual, true
	default:
		return Behave, false
	}
}

// Return the "weight" aka, the percentage probability of each transition
// assuming a perfectly random distribution when picking transitions
// i.e. below, the probability of picking createTopic is 2%
func GetHardCodedTransitionWeights() TransitionWeights {
	return TransitionWeights{
		CreateTopic:                     2,
		FundTopic:                       7,
		RegisterWorker:                  4,
		RegisterReputer:                 4,
		StakeAsReputer:                  5,
		DelegateStake:                   5,
		CollectDelegatorRewards:         5,
		DoInferenceAndReputation:        20,
		UnregisterWorker:                4,
		UnregisterReputer:               4,
		UnstakeAsReputer:                5,
		UndelegateStake:                 5,
		CancelStakeRemoval:              0,
		CancelDelegateStakeRemoval:      0,
		AddToAdminWhitelist:             1,
		RemoveFromAdminWhitelist:        1,
		AddToGlobalWhitelist:            1,
		RemoveFromGlobalWhitelist:       1,
		AddToTopicCreatorWhitelist:      1,
		RemoveFromTopicCreatorWhitelist: 1,
		EnableTopicWorkerWhitelist:      1,
		DisableTopicWorkerWhitelist:     1,
		EnableTopicReputerWhitelist:     1,
		DisableTopicReputerWhitelist:    1,
		AddToTopicWorkerWhitelist:       5,
		RemoveFromTopicWorkerWhitelist:  5,
		AddToTopicReputerWhitelist:      5,
		RemoveFromTopicReputerWhitelist: 5,
	}
}

func GetHardCodedInitialSetup() InitialSetup {
	return InitialSetup{
		NumAdminWhitelist:        2,
		NumGlobalWhitelist:       4,
		NumTopicCreatorWhitelist: 2,
	}
}

// Iterate over a struct and check that the sum of its fields is 100
func IterateAndCheckSum(structToCheck interface{}) (fieldsSum uint64, ok bool) {
	vw := reflect.ValueOf(structToCheck)
	fieldsSum = uint64(0)
	fields := reflect.VisibleFields(reflect.TypeOf(structToCheck))
	for _, field := range fields {
		fieldValue := vw.FieldByName(field.Name).Uint()
		fieldsSum += fieldValue
	}
	if fieldsSum != 100 {
		return fieldsSum, false
	}
	return fieldsSum, true
}

// Return a FuzzConfig struct for this fuzz session
// The order of priority is as follows:
// 1. env vars
// 2. test/fuzz/config.json
// 3. default values hard coded in this function
func GetFuzzConfig(t *testing.T) FuzzConfig {
	t.Helper()
	// hard coded defaults
	seed := 1
	rpcMode := testcommon.SingleRpc
	rpcEndpoints := []string{"http://localhost:26657"}
	maxIterations := 1000
	numActors := 100
	epochLength := 12
	mode := Alternate
	alternateWeight := 20
	transitionWeights := GetHardCodedTransitionWeights()
	initialSetup := GetHardCodedInitialSetup()

	// get values from config.json
	var jsonConfig *FuzzConfigJson = nil
	configJsonFile, err := os.ReadFile(".config.json")
	if err != nil {
		jsonConfig = nil
	} else {
		unmarshalJson := FuzzConfigJson{} //nolint: exhaustruct // this is how unmarshalling works
		err = json.Unmarshal(configJsonFile, &unmarshalJson)
		if err != nil {
			jsonConfig = nil
			t.Log("Error unmarshalling config.json, using default values: ", err)
		} else {
			jsonConfig = &unmarshalJson
		}
	}

	// apply values from config.json over hardcoded defaults
	if jsonConfig != nil {
		t.Log("Using values from config.json")
		seed = jsonConfig.Seed
		rpcMode = testcommon.StringToRpcConnectionType(t, jsonConfig.RpcMode)
		rpcEndpoints = jsonConfig.RpcUrls
		maxIterations = jsonConfig.MaxIterations
		numActors = jsonConfig.NumActors
		epochLength = jsonConfig.EpochLength
		mode = jsonConfig.Mode
		alternateWeight = jsonConfig.AlternateWeight
		transitionWeights = jsonConfig.TransitionWeights
		initialSetup = jsonConfig.InitialSetup
	} else {
		t.Log("No config.json found, proceeding without config.json values")
	}

	// lastly, apply values from env vars over config.json values and hardcoded defaults
	envSeed, found := testcommon.LookupEnvInt(t, "SEED")
	if found {
		seed = envSeed
	}
	envRpcMode, found := testcommon.LookupRpcMode(t, "RPC_MODE")
	if found {
		rpcMode = envRpcMode
	}
	envRpcEndpoints, found := testcommon.LookupEnvStringArray(t, "RPC_URLS")
	if found {
		rpcEndpoints = envRpcEndpoints
	}
	envMaxIterations, found := testcommon.LookupEnvInt(t, "MAX_ITERATIONS")
	if found {
		maxIterations = envMaxIterations
	}
	envNumActors, found := testcommon.LookupEnvInt(t, "NUM_ACTORS")
	if found {
		numActors = envNumActors
	}
	envEpochLength, found := testcommon.LookupEnvInt(t, "EPOCH_LENGTH")
	if found {
		epochLength = envEpochLength
	}
	envMode, found := lookupEnvSimulationMode()
	if found {
		mode = envMode
	}
	envAlternateWeight, found := testcommon.LookupEnvInt(t, "ALTERNATE_WEIGHT")
	if found {
		alternateWeight = envAlternateWeight
	}

	testCommonConfig := testcommon.NewTestConfig(
		t,
		rpcMode,
		rpcEndpoints,
		"../localnet/genesis",
		seed,
	)

	// one last check that the transition weights sum to 100
	weightSum, success := IterateAndCheckSum(transitionWeights)
	if !success {
		t.Fatalf("FuzzConfig: Transition weights do not sum to 100: %d\n", weightSum)
	}

	return FuzzConfig{
		Seed:              seed,
		RpcMode:           rpcMode,
		RpcEndpoints:      rpcEndpoints,
		MaxIterations:     maxIterations,
		NumActors:         numActors,
		EpochLength:       epochLength,
		Mode:              mode,
		AlternateWeight:   alternateWeight,
		TestConfig:        &testCommonConfig,
		TransitionWeights: transitionWeights,
		InitialSetup:      initialSetup,
	}

}
