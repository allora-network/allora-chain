package main

import (
	"fmt"
	"os"
	"reflect"

	fuzzcommon "github.com/allora-network/allora-chain/test/fuzz/common"
)

// this linter checks that the distribution of how often to pick
// state transitions, i.e. each transition's weight, sums to 100
func main() {
	transitionWeights := fuzzcommon.GetTransitionWeights()
	vw := reflect.ValueOf(transitionWeights)
	weightSum := uint64(0)
	fields := reflect.VisibleFields(reflect.TypeOf(transitionWeights))
	for _, field := range fields {
		fieldValue := vw.FieldByName(field.Name).Uint()
		weightSum += fieldValue
	}
	if weightSum != 100 {
		fmt.Printf("Weights of transitions do not sum to 100: %d\n", weightSum)
		os.Exit(2)
	}
}
