package testcommon

import "github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"

type NameAccountAndAddress struct {
	Name string
	Aa   AccountAndAddress
}

// holder for account and address
type AccountAndAddress struct {
	Acc  cosmosaccount.Account
	Addr string
}

type SimulationPackage struct {
	GroundTruth    []float64
	InferenceError [][]float64
	InferenceBias  [][]float64
	ForecastError  []float64
	ForecastBias   []float64
	ReputeError    []float64
	ReputeBias     []float64
	OutperFormer   []int
}
