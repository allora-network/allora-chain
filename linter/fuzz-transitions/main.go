package main

import (
	"encoding/json"
	"fmt"
	"os"

	fuzzcommon "github.com/allora-network/allora-chain/test/fuzz/common"
)

// this linter checks that the distribution of how often to pick
// state transitions, i.e. each transition's weight, sums to 100
func main() {
	// check hard coded transition weights first
	hardcodedTransitionWeights := fuzzcommon.GetHardCodedTransitionWeights()
	weightSum, success := fuzzcommon.IterateAndCheckSum(hardcodedTransitionWeights)
	if !success {
		fmt.Printf("Hardcoded transition weights do not sum to 100: %d\n", weightSum)
		os.Exit(2)
	} else {
		fmt.Printf("Hardcoded transition weights sum to 100: %d\n", weightSum)
	}

	// now check if the json exists
	// if it doesn't exist don't worry about checking it
	// but if it does exist, check that the sum of the weights is 100

	// get values from config.json
	configJsonFile, err := os.ReadFile("test/fuzz/.config.json")
	if err == nil {
		unmarshalJson := fuzzcommon.FuzzConfigJson{} // nolint: exhaustruct // this is how unmarshalling works
		err = json.Unmarshal(configJsonFile, &unmarshalJson)
		if err != nil {
			fmt.Printf("Error unmarshalling config.json: %s\n", err)
			os.Exit(2)
		} else {
			weightSum, success = fuzzcommon.IterateAndCheckSum(unmarshalJson.TransitionWeights)
			if !success {
				fmt.Printf("Transition weights do not sum to 100: %d\n", weightSum)
				os.Exit(2)
			} else {
				fmt.Printf("Fuzzer config.json transition weights sum to 100: %d\n", weightSum)
			}
		}
	} else {
		fmt.Printf("No fuzzer config.json found, proceeding without config.json values\n")
	}
}
