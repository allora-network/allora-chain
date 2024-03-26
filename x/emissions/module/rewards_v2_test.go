package module_test

import (
	"math"
	"testing"
	"time"

	cosmosMath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/allora-network/allora-chain/x/emissions/module"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// UNIT TESTS INTERNAL

func TestStdDev(t *testing.T) {
	tests := []struct {
		name string
		data []float64
		want float64
	}{
		{
			name: "basic",
			data: []float64{-0.00675, -0.00622, -0.01502, -0.01214, 0.00392, 0.00559, 0.0438, 0.04304, 0.09719, 0.09675},
			want: 0.041014924273483966,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := module.StdDev(tt.data); math.Abs(got-tt.want) > 1e-5 {
				t.Errorf("StdDev() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetWorkerRewardFractions(t *testing.T) {
	tests := []struct {
		name    string
		scores  [][]float64
		preward float64
		want    []float64
		wantErr bool
	}{
		{
			name: "basic",
			scores: [][]float64{
				{-0.00675, -0.00622, -0.00388},
				{-0.01502, -0.01214, -0.01554},
				{0.00392, 0.00559, 0.00545},
				{0.0438, 0.04304, 0.03906},
				{0.09719, 0.09675, 0.09418},
			},
			preward: 1.5,
			want:    []float64{0.07671, 0.05531, 0.09829, 0.21537, 0.55432},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := module.GetWorkerRewardFractions(tt.scores, tt.preward)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetWorkerRewardFractions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !slicesAreApproxEqual(got, tt.want, 1e-4) {
				t.Errorf("GetWorkerRewardFractions() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetReputerRewardFractions(t *testing.T) {
	tests := []struct {
		name    string
		stakes  []float64
		scores  []float64
		preward float64
		want    []float64
		wantErr bool
	}{
		{
			name:    "basic",
			stakes:  []float64{1178377.89152, 385287.87376, 395488.13091, 208201.11762, 369044.55988},
			scores:  []float64{17.53839, 22.63517, 26.28035, 13.51383, 15.08629},
			preward: 1,
			want:    []float64{0.42911, 0.18108, 0.2158, 0.05842, 0.1156},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := module.GetReputerRewardFractions(tt.stakes, tt.scores, tt.preward)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetReputerRewardFractions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !slicesAreApproxEqual(got, tt.want, 1e-5) {
				t.Errorf("GetReputerRewardFractions() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetStakeWeightedLossMatrix(t *testing.T) {
	tests := []struct {
		name                   string
		reputersAdjustedStakes []float64
		reputersReportedLosses [][]float64
		want                   []float64
		wantErr                bool
	}{
		{
			name:                   "basic",
			reputersAdjustedStakes: []float64{1.0, 0.76188, 0.7816, 0.40664, 0.71687},
			reputersReportedLosses: [][]float64{
				{0.0112, 0.00231, 0.02274, 0.01299, 0.02515, 0.0185, 0.01018, 0.02105, 0.01041, 0.0183, 0.01022, 0.01333, 0.01298, 0.01023, 0.01268, 0.01381, 0.01731, 0.01238, 0.01168, 0.00929, 0.01212, 0.01806, 0.01901, 0.01828, 0.01522, 0.01833, 0.0101, 0.01224, 0.01226, 0.01474, 0.01218, 0.01604, 0.01149, 0.02075, 0.00818, 0.0116, 0.01127, 0.01495, 0.00689, 0.0108, 0.01417, 0.0124, 0.01588, 0.01012, 0.01467, 0.0128, 0.01234, 0.0148, 0.01046, 0.01192, 0.01381, 0.01687, 0.01136, 0.01185, 0.01568, 0.00949, 0.01339},
				{0.01635, 0.00179, 0.03396, 0.0153, 0.01988, 0.00962, 0.01191, 0.01616, 0.01417, 0.01216, 0.01292, 0.01564, 0.01323, 0.01261, 0.01145, 0.0163, 0.014, 0.01373, 0.01453, 0.01207, 0.01641, 0.01601, 0.01114, 0.01259, 0.01589, 0.01229, 0.01309, 0.0138, 0.01162, 0.01145, 0.01013, 0.01208, 0.0111, 0.0118, 0.01374, 0.01428, 0.01791, 0.01288, 0.01161, 0.01151, 0.01148, 0.01284, 0.01239, 0.01023, 0.01712, 0.0116, 0.01639, 0.01043, 0.01308, 0.01455, 0.01607, 0.01205, 0.01357, 0.01108, 0.01633, 0.01208, 0.01278},
				{0.01345, 0.00209, 0.03249, 0.01688, 0.02126, 0.01338, 0.0116, 0.01605, 0.0133, 0.01407, 0.01367, 0.01244, 0.0145, 0.01262, 0.01348, 0.01684, 0.01148, 0.01705, 0.01714, 0.0124, 0.0125, 0.01462, 0.01274, 0.01407, 0.01667, 0.01316, 0.01628, 0.01373, 0.01409, 0.01603, 0.01378, 0.01143, 0.013, 0.01644, 0.01528, 0.01441, 0.01404, 0.01402, 0.01479, 0.01417, 0.01244, 0.0116, 0.01419, 0.01497, 0.01629, 0.01514, 0.01133, 0.01339, 0.01053, 0.01424, 0.01428, 0.01446, 0.01805, 0.01229, 0.01586, 0.01234, 0.01513},
				{0.01675, 0.00318, 0.02623, 0.02734, 0.03526, 0.02733, 0.01697, 0.01619, 0.01925, 0.02018, 0.01735, 0.01922, 0.02225, 0.0189, 0.01923, 0.03193, 0.01956, 0.01763, 0.01975, 0.01466, 0.02021, 0.01803, 0.01438, 0.01929, 0.02305, 0.02223, 0.02445, 0.01967, 0.02292, 0.01878, 0.01751, 0.02695, 0.01849, 0.01658, 0.01948, 0.01594, 0.02318, 0.01906, 0.01607, 0.01369, 0.01686, 0.01314, 0.01936, 0.01518, 0.018, 0.02212, 0.02259, 0.01674, 0.02944, 0.01796, 0.02187, 0.01895, 0.01637, 0.01594, 0.01608, 0.02203, 0.01486},
				{0.02093, 0.00213, 0.02462, 0.0203, 0.03115, 0.01, 0.01545, 0.01785, 0.01662, 0.01156, 0.02284, 0.01475, 0.01331, 0.01592, 0.01462, 0.02333, 0.01836, 0.01465, 0.0186, 0.01566, 0.01506, 0.01678, 0.01423, 0.01658, 0.01741, 0.03491, 0.01408, 0.01191, 0.01572, 0.01355, 0.01477, 0.01662, 0.01128, 0.02581, 0.01718, 0.01705, 0.01251, 0.02158, 0.01187, 0.01504, 0.0135, 0.02432, 0.01602, 0.01194, 0.0153, 0.0199, 0.01673, 0.01049, 0.02068, 0.01573, 0.01487, 0.02639, 0.01981, 0.02123, 0.02134, 0.0217, 0.01177},
			},
			want: []float64{
				0.01489, 0.00219, 0.02752, 0.01684, 0.02502, 0.01395, 0.01242, 0.01769, 0.01372,
				0.01469, 0.01416, 0.01442, 0.01423, 0.01304, 0.01354, 0.01813, 0.01556, 0.01455,
				0.0154, 0.01215, 0.01435, 0.01659, 0.01431, 0.01579, 0.01683, 0.01821, 0.01389,
				0.01348, 0.01405, 0.01439, 0.01301, 0.01501, 0.0123, 0.01788, 0.01325, 0.01417,
				0.01438, 0.01578, 0.01104, 0.0127, 0.01332, 0.01414, 0.01508, 0.01191, 0.01598,
				0.01506, 0.01459, 0.01277, 0.01406, 0.01426, 0.01532, 0.01683, 0.0151, 0.01364,
				0.01688, 0.01362, 0.01342,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := module.GetStakeWeightedLossMatrix(tt.reputersAdjustedStakes, tt.reputersReportedLosses)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStakeWeightedLossMatrix() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// convert to 10^x
			for i, v := range got {
				got[i] = math.Pow(10, v)
			}
			if !slicesAreApproxEqual(got, tt.want, 1e-5) {
				t.Errorf("GetStakeWeightedLossMatrix() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetStakeWeightedLoss(t *testing.T) {
	tests := []struct {
		name                   string
		reputersStakes         []float64
		reputersReportedLosses []float64
		want                   float64
		wantErr                bool
	}{
		{
			name:                   "simple average",
			reputersStakes:         []float64{1176644.37627, 384623.3607, 394676.13226, 207999.66194, 368582.76542},
			reputersReportedLosses: []float64{0.0112, 0.01635, 0.01345, 0.01675, 0.02093},
			want:                   0.01381883491416319,
			wantErr:                false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := module.GetStakeWeightedLoss(tt.reputersStakes, tt.reputersReportedLosses)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStakeWeightedLoss() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !(math.Abs(math.Pow(10, got)-tt.want) <= 1e-5) {
				t.Errorf("GetStakeWeightedLoss() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetWorkerScore(t *testing.T) {
	tests := []struct {
		name         string
		losses       float64
		lossesOneOut float64
		want         float64
	}{
		{"basic", 0.011411892282242868, 0.01344474872292, 0.07119502617735574},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := module.GetWorkerScore(tt.losses, tt.lossesOneOut)
			if !(math.Abs(got-tt.want) <= 1e-14) {
				t.Errorf("GetWorkerScore() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetFinalWorkerScoreForecastTask(t *testing.T) {
	tests := []struct {
		name        string
		scoreOneIn  float64
		scoreOneOut float64
		fUniqueAgg  float64
		want        float64
	}{
		{"basic", 0.07300629674057668, -0.009510726019112292, 0.0625, -0.004353412096631731},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := module.GetFinalWorkerScoreForecastTask(tt.scoreOneIn, tt.scoreOneOut, tt.fUniqueAgg)
			if got != tt.want {
				t.Errorf("GetFinalWorkerScoreForecastTask() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAllConsensusScores(t *testing.T) {
	tests := []struct {
		name                     string
		allLosses                [][]float64
		stakes                   []float64
		allListeningCoefficients []float64
		numReputers              int
		want                     []float64
		wantErr                  bool
	}{
		{
			name: "basic",
			allLosses: [][]float64{
				{0.0112, 0.00231, 0.02274, 0.01299, 0.02515, 0.0185, 0.01018, 0.02105, 0.01041, 0.0183, 0.01022, 0.01333, 0.01298, 0.01023, 0.01268, 0.01381, 0.01731, 0.01238, 0.01168, 0.00929, 0.01212, 0.01806, 0.01901, 0.01828, 0.01522, 0.01833, 0.0101, 0.01224, 0.01226, 0.01474, 0.01218, 0.01604, 0.01149, 0.02075, 0.00818, 0.0116, 0.01127, 0.01495, 0.00689, 0.0108, 0.01417, 0.0124, 0.01588, 0.01012, 0.01467, 0.0128, 0.01234, 0.0148, 0.01046, 0.01192, 0.01381, 0.01687, 0.01136, 0.01185, 0.01568, 0.00949, 0.01339},
				{0.01635, 0.00179, 0.03396, 0.0153, 0.01988, 0.00962, 0.01191, 0.01616, 0.01417, 0.01216, 0.01292, 0.01564, 0.01323, 0.01261, 0.01145, 0.0163, 0.014, 0.01373, 0.01453, 0.01207, 0.01641, 0.01601, 0.01114, 0.01259, 0.01589, 0.01229, 0.01309, 0.0138, 0.01162, 0.01145, 0.01013, 0.01208, 0.0111, 0.0118, 0.01374, 0.01428, 0.01791, 0.01288, 0.01161, 0.01151, 0.01148, 0.01284, 0.01239, 0.01023, 0.01712, 0.0116, 0.01639, 0.01043, 0.01308, 0.01455, 0.01607, 0.01205, 0.01357, 0.01108, 0.01633, 0.01208, 0.01278},
				{0.01345, 0.00209, 0.03249, 0.01688, 0.02126, 0.01338, 0.0116, 0.01605, 0.0133, 0.01407, 0.01367, 0.01244, 0.0145, 0.01262, 0.01348, 0.01684, 0.01148, 0.01705, 0.01714, 0.0124, 0.0125, 0.01462, 0.01274, 0.01407, 0.01667, 0.01316, 0.01628, 0.01373, 0.01409, 0.01603, 0.01378, 0.01143, 0.013, 0.01644, 0.01528, 0.01441, 0.01404, 0.01402, 0.01479, 0.01417, 0.01244, 0.0116, 0.01419, 0.01497, 0.01629, 0.01514, 0.01133, 0.01339, 0.01053, 0.01424, 0.01428, 0.01446, 0.01805, 0.01229, 0.01586, 0.01234, 0.01513},
				{0.01675, 0.00318, 0.02623, 0.02734, 0.03526, 0.02733, 0.01697, 0.01619, 0.01925, 0.02018, 0.01735, 0.01922, 0.02225, 0.0189, 0.01923, 0.03193, 0.01956, 0.01763, 0.01975, 0.01466, 0.02021, 0.01803, 0.01438, 0.01929, 0.02305, 0.02223, 0.02445, 0.01967, 0.02292, 0.01878, 0.01751, 0.02695, 0.01849, 0.01658, 0.01948, 0.01594, 0.02318, 0.01906, 0.01607, 0.01369, 0.01686, 0.01314, 0.01936, 0.01518, 0.018, 0.02212, 0.02259, 0.01674, 0.02944, 0.01796, 0.02187, 0.01895, 0.01637, 0.01594, 0.01608, 0.02203, 0.01486},
				{0.02093, 0.00213, 0.02462, 0.0203, 0.03115, 0.01, 0.01545, 0.01785, 0.01662, 0.01156, 0.02284, 0.01475, 0.01331, 0.01592, 0.01462, 0.02333, 0.01836, 0.01465, 0.0186, 0.01566, 0.01506, 0.01678, 0.01423, 0.01658, 0.01741, 0.03491, 0.01408, 0.01191, 0.01572, 0.01355, 0.01477, 0.01662, 0.01128, 0.02581, 0.01718, 0.01705, 0.01251, 0.02158, 0.01187, 0.01504, 0.0135, 0.02432, 0.01602, 0.01194, 0.0153, 0.0199, 0.01673, 0.01049, 0.02068, 0.01573, 0.01487, 0.02639, 0.01981, 0.02123, 0.02134, 0.0217, 0.01177},
			},
			stakes:                   []float64{1176644.37627, 384623.3607, 394676.13226, 207999.66194, 368582.76542},
			allListeningCoefficients: []float64{1.0, 1.0, 1.0, 1.0, 1.0},
			numReputers:              5,
			want:                     []float64{17.4346, 20.13897, 24.08276, 11.41393, 15.33319},
			wantErr:                  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := module.GetAllConsensusScores(tt.allLosses, tt.stakes, tt.allListeningCoefficients, tt.numReputers)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetAllConsensusScores() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !slicesAreApproxEqual(got, tt.want, 1e-2) {
				t.Errorf("GetAllConsensusScores() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAllReputersOutput(t *testing.T) {
	tests := []struct {
		name                string
		allLosses           [][]float64
		stakes              []float64
		consensusScores     []float64
		initialCoefficients []float64
		numReputers         int
		wantScores          []float64
		wantCoefficients    []float64
		wantErr             bool
	}{
		{
			name: "basic",
			allLosses: [][]float64{
				{0.0112, 0.00231, 0.02274, 0.01299, 0.02515, 0.0185, 0.01018, 0.02105, 0.01041, 0.0183, 0.01022, 0.01333, 0.01298, 0.01023, 0.01268, 0.01381, 0.01731, 0.01238, 0.01168, 0.00929, 0.01212, 0.01806, 0.01901, 0.01828, 0.01522, 0.01833, 0.0101, 0.01224, 0.01226, 0.01474, 0.01218, 0.01604, 0.01149, 0.02075, 0.00818, 0.0116, 0.01127, 0.01495, 0.00689, 0.0108, 0.01417, 0.0124, 0.01588, 0.01012, 0.01467, 0.0128, 0.01234, 0.0148, 0.01046, 0.01192, 0.01381, 0.01687, 0.01136, 0.01185, 0.01568, 0.00949, 0.01339},
				{0.01635, 0.00179, 0.03396, 0.0153, 0.01988, 0.00962, 0.01191, 0.01616, 0.01417, 0.01216, 0.01292, 0.01564, 0.01323, 0.01261, 0.01145, 0.0163, 0.014, 0.01373, 0.01453, 0.01207, 0.01641, 0.01601, 0.01114, 0.01259, 0.01589, 0.01229, 0.01309, 0.0138, 0.01162, 0.01145, 0.01013, 0.01208, 0.0111, 0.0118, 0.01374, 0.01428, 0.01791, 0.01288, 0.01161, 0.01151, 0.01148, 0.01284, 0.01239, 0.01023, 0.01712, 0.0116, 0.01639, 0.01043, 0.01308, 0.01455, 0.01607, 0.01205, 0.01357, 0.01108, 0.01633, 0.01208, 0.01278},
				{0.01345, 0.00209, 0.03249, 0.01688, 0.02126, 0.01338, 0.0116, 0.01605, 0.0133, 0.01407, 0.01367, 0.01244, 0.0145, 0.01262, 0.01348, 0.01684, 0.01148, 0.01705, 0.01714, 0.0124, 0.0125, 0.01462, 0.01274, 0.01407, 0.01667, 0.01316, 0.01628, 0.01373, 0.01409, 0.01603, 0.01378, 0.01143, 0.013, 0.01644, 0.01528, 0.01441, 0.01404, 0.01402, 0.01479, 0.01417, 0.01244, 0.0116, 0.01419, 0.01497, 0.01629, 0.01514, 0.01133, 0.01339, 0.01053, 0.01424, 0.01428, 0.01446, 0.01805, 0.01229, 0.01586, 0.01234, 0.01513},
				{0.01675, 0.00318, 0.02623, 0.02734, 0.03526, 0.02733, 0.01697, 0.01619, 0.01925, 0.02018, 0.01735, 0.01922, 0.02225, 0.0189, 0.01923, 0.03193, 0.01956, 0.01763, 0.01975, 0.01466, 0.02021, 0.01803, 0.01438, 0.01929, 0.02305, 0.02223, 0.02445, 0.01967, 0.02292, 0.01878, 0.01751, 0.02695, 0.01849, 0.01658, 0.01948, 0.01594, 0.02318, 0.01906, 0.01607, 0.01369, 0.01686, 0.01314, 0.01936, 0.01518, 0.018, 0.02212, 0.02259, 0.01674, 0.02944, 0.01796, 0.02187, 0.01895, 0.01637, 0.01594, 0.01608, 0.02203, 0.01486},
				{0.02093, 0.00213, 0.02462, 0.0203, 0.03115, 0.01, 0.01545, 0.01785, 0.01662, 0.01156, 0.02284, 0.01475, 0.01331, 0.01592, 0.01462, 0.02333, 0.01836, 0.01465, 0.0186, 0.01566, 0.01506, 0.01678, 0.01423, 0.01658, 0.01741, 0.03491, 0.01408, 0.01191, 0.01572, 0.01355, 0.01477, 0.01662, 0.01128, 0.02581, 0.01718, 0.01705, 0.01251, 0.02158, 0.01187, 0.01504, 0.0135, 0.02432, 0.01602, 0.01194, 0.0153, 0.0199, 0.01673, 0.01049, 0.02068, 0.01573, 0.01487, 0.02639, 0.01981, 0.02123, 0.02134, 0.0217, 0.01177},
			},
			stakes:              []float64{1176644.37627, 384623.3607, 394676.13226, 207999.66194, 368582.76542},
			initialCoefficients: []float64{1.0, 1.0, 1.0, 1.0, 1.0},
			numReputers:         5,
			wantScores:          []float64{17.53436, 20.29489, 24.26994, 11.36754, 15.21749},
			wantCoefficients:    []float64{0.99942, 1.0, 1.0, 0.96574, 0.95346},
			wantErr:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotScores, gotCoefficients, err := module.GetAllReputersOutput(tt.allLosses, tt.stakes, tt.initialCoefficients, tt.numReputers)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAllReputersOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !slicesAreApproxEqual(gotScores, tt.wantScores, 1e-4) {
				t.Errorf("GetAllReputersOutput() gotScores = %v, want %v", gotScores, tt.wantScores)
			}

			if !slicesAreApproxEqual(gotCoefficients, tt.wantCoefficients, 1e-4) {
				t.Errorf("GetAllReputersOutput() gotCoefficients = %v, want %v", gotCoefficients, tt.wantCoefficients)
			}
		})
	}
}

// Helper function to compare two slices of float64 within a tolerance
func slicesAreApproxEqual(a, b []float64, tolerance float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if math.Abs(a[i]-b[i]) > tolerance {
			return false
		}
	}
	return true
}

// UNIT TESTS
// TODO: Use the tests below as reference when new unit tests are added - probably half of each test will be converted into functions.

func (s *ModuleTestSuite) TestGetReputerScore() {
	// Mock data with 2 reputers reporting loss of 2 workers
	// [0] - Reputer 1
	// [1] - Reputer 2
	reputersReportedCombinedLosses := []float64{85, 80}
	reputersReportedNaiveLosses := []float64{100, 90}
	reputersWorker1ReportedInferenceLosses := []float64{90, 90}
	reputersWorker2ReportedInferenceLosses := []float64{100, 100}
	reputersWorker1ReportedForecastLosses := []float64{90, 85}
	reputersWorker2ReportedForecastLosses := []float64{100, 100}
	reputersWorker1ReportedOneOutLosses := []float64{115, 120}
	reputersWorker2ReportedOneOutLosses := []float64{100, 100}
	reputersWorker1ReportedOneInNaiveLosses := []float64{90, 85}
	reputersWorker2ReportedOneInNaiveLosses := []float64{100, 100}

	reputer1AllReportedLosses := []float64{
		reputersReportedCombinedLosses[0],
		reputersReportedNaiveLosses[0],
		reputersWorker1ReportedInferenceLosses[0],
		reputersWorker2ReportedInferenceLosses[0],
		reputersWorker1ReportedForecastLosses[0],
		reputersWorker2ReportedForecastLosses[0],
		reputersWorker1ReportedOneOutLosses[0],
		reputersWorker2ReportedOneOutLosses[0],
		reputersWorker1ReportedOneInNaiveLosses[0],
		reputersWorker2ReportedOneInNaiveLosses[0],
	}
	reputer2AllReportedLosses := []float64{
		reputersReportedCombinedLosses[1],
		reputersReportedNaiveLosses[1],
		reputersWorker1ReportedInferenceLosses[1],
		reputersWorker2ReportedInferenceLosses[1],
		reputersWorker1ReportedForecastLosses[1],
		reputersWorker2ReportedForecastLosses[1],
		reputersWorker1ReportedOneOutLosses[1],
		reputersWorker2ReportedOneOutLosses[1],
		reputersWorker1ReportedOneInNaiveLosses[1],
		reputersWorker2ReportedOneInNaiveLosses[1],
	}

	allReputersStakes := []float64{50, 150}

	// Get listening coefficients
	listeningCoefficient := 0.18
	allListeningCoefficients := []float64{listeningCoefficient, 0.63}

	// Get adjusted stakes
	var adjustedStakes []float64
	for _, reputerStake := range allReputersStakes {
		adjustedStake, err := module.GetAdjustedStake(reputerStake, allReputersStakes, listeningCoefficient, allListeningCoefficients, float64(2))
		s.NoError(err, "Error getting adjustedStake")
		adjustedStakes = append(adjustedStakes, adjustedStake)
	}

	// Get consensus loss vector
	consensus, err := module.GetStakeWeightedLossMatrix(adjustedStakes, [][]float64{reputer1AllReportedLosses, reputer2AllReportedLosses})
	s.NoError(err, "Error getting consensus")

	// Get reputer scores
	reputer1Score, err := module.GetConsensusScore(reputer1AllReportedLosses, consensus)
	s.NoError(err, "Error getting reputer1Score")
	s.NotEqual(0, reputer1Score, "Expected reputer1Score to be non-zero")

	reputer2Score, err := module.GetConsensusScore(reputer2AllReportedLosses, consensus)
	s.NoError(err, "Error getting reputer2Score")
	s.NotEqual(0, reputer2Score, "Expected reputer2Score to be non-zero")
}

func (s *ModuleTestSuite) TestGetWorkerScoreForecastTask() {
	timeNow := uint64(time.Now().UTC().Unix())

	// Create a topic
	topicIds, err := mockCreateTopics(s, 1)
	s.NoError(err, "Error creating topic")
	topicId := topicIds[0]

	// Create and register 2 reputers in topic
	reputers, err := mockSomeReputers(s, topicId)
	s.NoError(err, "Error creating reputers")

	// Create and register 2 workers in topic
	workers, err := mockSomeWorkers(s, topicId)
	s.NoError(err, "Error creating workers")

	// Add a lossBundle for each reputer
	var reputersLossBundles []*types.LossBundle
	reputer1LossBundle := types.LossBundle{
		TopicId:      topicId,
		Reputer:      reputers[0].String(),
		CombinedLoss: cosmosMath.NewUint(85),
		ForecasterLosses: []*types.WorkerAttributedLoss{
			{
				Worker: workers[0].String(),
				Value:  cosmosMath.NewUint(90),
			},
			{
				Worker: workers[1].String(),
				Value:  cosmosMath.NewUint(100),
			},
		},
		NaiveLoss: cosmosMath.NewUint(100),
		// Increased loss when removing for worker 1
		OneOutLosses: []*types.WorkerAttributedLoss{
			{
				Worker: workers[0].String(),
				Value:  cosmosMath.NewUint(115),
			},
			{
				Worker: workers[1].String(),
				Value:  cosmosMath.NewUint(100),
			},
		},
		OneInNaiveLosses: []*types.WorkerAttributedLoss{
			{
				Worker: workers[0].String(),
				Value:  cosmosMath.NewUint(90),
			},
			{
				Worker: workers[1].String(),
				Value:  cosmosMath.NewUint(100),
			},
		},
	}
	reputersLossBundles = append(reputersLossBundles, &reputer1LossBundle)
	reputer2LossBundle := types.LossBundle{
		TopicId:      topicId,
		Reputer:      reputers[1].String(),
		CombinedLoss: cosmosMath.NewUint(80),
		ForecasterLosses: []*types.WorkerAttributedLoss{
			{
				Worker: workers[0].String(),
				Value:  cosmosMath.NewUint(85),
			},
			{
				Worker: workers[1].String(),
				Value:  cosmosMath.NewUint(100),
			},
		},
		NaiveLoss: cosmosMath.NewUint(90),
		// Increased loss when removing for worker 1
		OneOutLosses: []*types.WorkerAttributedLoss{
			{
				Worker: workers[0].String(),
				Value:  cosmosMath.NewUint(120),
			},
			{
				Worker: workers[1].String(),
				Value:  cosmosMath.NewUint(100),
			},
		},
		OneInNaiveLosses: []*types.WorkerAttributedLoss{
			{
				Worker: workers[0].String(),
				Value:  cosmosMath.NewUint(85),
			},
			{
				Worker: workers[1].String(),
				Value:  cosmosMath.NewUint(100),
			},
		},
	}
	reputersLossBundles = append(reputersLossBundles, &reputer2LossBundle)

	err = s.emissionsKeeper.InsertLossBundles(s.ctx, topicId, timeNow, types.LossBundles{LossBundles: reputersLossBundles})
	s.NoError(err, "Error adding lossBundles")

	// Get LossBundles
	lossBundles, err := s.emissionsKeeper.GetLossBundles(s.ctx, topicId, timeNow)
	s.NoError(err, "Error getting lossBundles")

	// Get reputers stakes and reported losses for each worker
	var reputersStakes []float64
	var reputersReportedCombinedLosses []float64
	var reputersNaiveReportedLosses []float64

	var reputersWorker1ReportedOneOutLosses []float64
	var reputersWorker2ReportedOneOutLosses []float64

	var reputersWorker1ReportedOneInNaiveLosses []float64
	var reputersWorker2ReportedOneInNaiveLosses []float64

	for _, lossBundle := range lossBundles.LossBundles {
		reputerAddr, err := sdk.AccAddressFromBech32(lossBundle.Reputer)
		s.NoError(err, "Error getting reputerAddr")

		reputerStake, err := s.emissionsKeeper.GetStakeOnTopicFromReputer(s.ctx, topicId, reputerAddr)
		s.NoError(err, "Error getting reputerStake")

		reputerStakeFloat := float64(reputerStake.BigInt().Int64())
		reputersStakes = append(reputersStakes, reputerStakeFloat)
		reputersReportedCombinedLosses = append(reputersReportedCombinedLosses, float64(lossBundle.CombinedLoss.BigInt().Int64()))
		reputersNaiveReportedLosses = append(reputersNaiveReportedLosses, float64(lossBundle.NaiveLoss.BigInt().Int64()))

		// Add OneOutLosses
		for _, workerLoss := range lossBundle.OneOutLosses {
			if workerLoss.Worker == workers[0].String() {
				reputersWorker1ReportedOneOutLosses = append(reputersWorker1ReportedOneOutLosses, float64(workerLoss.Value.BigInt().Int64()))
			} else if workerLoss.Worker == workers[1].String() {
				reputersWorker2ReportedOneOutLosses = append(reputersWorker2ReportedOneOutLosses, float64(workerLoss.Value.BigInt().Int64()))
			}
		}

		// Add OneInNaiveLosses
		for _, workerLoss := range lossBundle.OneInNaiveLosses {
			if workerLoss.Worker == workers[0].String() {
				reputersWorker1ReportedOneInNaiveLosses = append(reputersWorker1ReportedOneInNaiveLosses, float64(workerLoss.Value.BigInt().Int64()))
			} else if workerLoss.Worker == workers[1].String() {
				reputersWorker2ReportedOneInNaiveLosses = append(reputersWorker2ReportedOneInNaiveLosses, float64(workerLoss.Value.BigInt().Int64()))
			}
		}
	}

	// Get Stake Weighted Loss - Network Inference Loss - (L_i)
	networkStakeWeightedLoss, err := module.GetStakeWeightedLoss(reputersStakes, reputersReportedCombinedLosses)
	s.NoError(err, "Error getting stakeWeightedLoss")
	s.NotEqual(0, networkStakeWeightedLoss, "Expected worker1StakeWeightedLoss to be non-zero")

	// Get Stake Weighted Loss - Naive Loss - (L^-_i)
	networkStakeWeightedNaiveLoss, err := module.GetStakeWeightedLoss(reputersStakes, reputersNaiveReportedLosses)
	s.NoError(err, "Error getting stakeWeightedLoss")
	s.NotEqual(0, networkStakeWeightedNaiveLoss, "Expected worker1StakeWeightedNaiveLoss to be non-zero")

	// Get Stake Weighted Loss - OneOut Loss - (L^-_i)
	worker1StakeWeightedOneOutLoss, err := module.GetStakeWeightedLoss(reputersStakes, reputersWorker1ReportedOneOutLosses)
	s.NoError(err, "Error getting stakeWeightedLoss")
	s.NotEqual(0, worker1StakeWeightedOneOutLoss, "Expected worker1StakeWeightedOneOutLoss to be non-zero")

	worker2StakeWeightedOneOutLoss, err := module.GetStakeWeightedLoss(reputersStakes, reputersWorker2ReportedOneOutLosses)
	s.NoError(err, "Error getting stakeWeightedLoss")
	s.NotEqual(0, worker2StakeWeightedOneOutLoss, "Expected worker2StakeWeightedOneOutLoss to be non-zero")

	// Get Stake Weighted Loss - OneInNaive Loss - (L^+_ki)
	worker1StakeWeightedOneInNaiveLoss, err := module.GetStakeWeightedLoss(reputersStakes, reputersWorker1ReportedOneInNaiveLosses)
	s.NoError(err, "Error getting stakeWeightedLoss")
	s.NotEqual(0, worker1StakeWeightedOneInNaiveLoss, "Expected worker1StakeWeightedOneInNaiveLoss to be non-zero")

	worker2StakeWeightedOneInNaiveLoss, err := module.GetStakeWeightedLoss(reputersStakes, reputersWorker2ReportedOneInNaiveLosses)
	s.NoError(err, "Error getting stakeWeightedLoss")
	s.NotEqual(0, worker2StakeWeightedOneInNaiveLoss, "Expected worker2StakeWeightedOneInNaiveLoss to be non-zero")

	// Get Worker Score - OneOut Score (L^-_ki - L_i)
	worker1OneOutScore := module.GetWorkerScore(networkStakeWeightedLoss, worker1StakeWeightedOneOutLoss)
	s.NotEqual(0, worker1OneOutScore, "Expected worker1Score to be non-zero")

	worker2OneOutScore := module.GetWorkerScore(networkStakeWeightedLoss, worker2StakeWeightedOneOutLoss)
	s.NotEqual(0, worker2OneOutScore, "Expected worker2Score to be non-zero")

	// Get Worker Score - OneIn Score (L^-_i - L^+_ki)
	worker1ScoreOneInNaive := module.GetWorkerScore(networkStakeWeightedNaiveLoss, worker1StakeWeightedOneInNaiveLoss)
	s.NotEqual(0, worker1ScoreOneInNaive, "Expected worker1ScoreOneInNaive to be non-zero")

	worker2ScoreOneInNaive := module.GetWorkerScore(networkStakeWeightedNaiveLoss, worker2StakeWeightedOneInNaiveLoss)
	s.NotEqual(0, worker2ScoreOneInNaive, "Expected worker2ScoreOneInNaive to be non-zero")

	// Get Worker 1 Final Score - Forecast Task (T_ik)
	worker1FinalScore := module.GetFinalWorkerScoreForecastTask(worker1ScoreOneInNaive, worker1OneOutScore, module.GetfUniqueAgg(2))
	s.NotEqual(0, worker1FinalScore, "Expected worker1FinalScore to be non-zero")

	// Get Worker 2 Final Score - Forecast Task (T_ik)
	worker2FinalScore := module.GetFinalWorkerScoreForecastTask(worker2ScoreOneInNaive, worker2OneOutScore, module.GetfUniqueAgg(2))
	s.NotEqual(0, worker2FinalScore, "Expected worker2FinalScore to be non-zero")
}

func (s *ModuleTestSuite) TestGetWorkerScoreInferenceTask() {
	timeNow := uint64(time.Now().UTC().Unix())

	// Create a topic
	topicIds, err := mockCreateTopics(s, 1)
	s.NoError(err, "Error creating topic")
	topicId := topicIds[0]

	// Create and register 2 reputers in topic
	reputers, err := mockSomeReputers(s, topicId)
	s.NoError(err, "Error creating reputers")

	// Create and register 2 workers in topic
	workers, err := mockSomeWorkers(s, topicId)
	s.NoError(err, "Error creating workers")

	// Add a lossBundle for each reputer
	var reputersLossBundles []*types.LossBundle
	reputer1LossBundle := types.LossBundle{
		TopicId:      topicId,
		Reputer:      reputers[0].String(),
		CombinedLoss: cosmosMath.NewUint(85),
		// Increased loss when removing for worker 1
		OneOutLosses: []*types.WorkerAttributedLoss{
			{
				Worker: workers[0].String(),
				Value:  cosmosMath.NewUint(115),
			},
			{
				Worker: workers[1].String(),
				Value:  cosmosMath.NewUint(100),
			},
		},
	}
	reputersLossBundles = append(reputersLossBundles, &reputer1LossBundle)
	reputer2LossBundle := types.LossBundle{
		TopicId:      topicId,
		Reputer:      reputers[1].String(),
		CombinedLoss: cosmosMath.NewUint(80),
		// Increased loss when removing for worker 1
		OneOutLosses: []*types.WorkerAttributedLoss{
			{
				Worker: workers[0].String(),
				Value:  cosmosMath.NewUint(120),
			},
			{
				Worker: workers[1].String(),
				Value:  cosmosMath.NewUint(100),
			},
		},
	}
	reputersLossBundles = append(reputersLossBundles, &reputer2LossBundle)

	err = s.emissionsKeeper.InsertLossBundles(s.ctx, topicId, timeNow, types.LossBundles{LossBundles: reputersLossBundles})
	s.NoError(err, "Error adding lossBundle for worker")

	// Get LossBundles
	lossBundles, err := s.emissionsKeeper.GetLossBundles(s.ctx, topicId, timeNow)
	s.NoError(err, "Error getting lossBundles")

	// Get reputers stakes and reported losses for each worker
	var reputersStakes []float64
	var reputersCombinedReportedLosses []float64

	var reputersWorker1ReportedOneOutLosses []float64
	var reputersWorker2ReportedOneOutLosses []float64

	for _, lossBundle := range lossBundles.LossBundles {
		reputerAddr, err := sdk.AccAddressFromBech32(lossBundle.Reputer)
		s.NoError(err, "Error getting reputerAddr")

		reputerStake, err := s.emissionsKeeper.GetStakeOnTopicFromReputer(s.ctx, topicId, reputerAddr)
		s.NoError(err, "Error getting reputerStake")

		reputerStakeFloat := float64(reputerStake.BigInt().Int64())
		reputersStakes = append(reputersStakes, reputerStakeFloat)
		reputersCombinedReportedLosses = append(reputersCombinedReportedLosses, float64(lossBundle.CombinedLoss.BigInt().Int64()))

		// Add OneOutLosses
		for _, workerLoss := range lossBundle.OneOutLosses {
			if workerLoss.Worker == workers[0].String() {
				reputersWorker1ReportedOneOutLosses = append(reputersWorker1ReportedOneOutLosses, float64(workerLoss.Value.BigInt().Int64()))
			} else if workerLoss.Worker == workers[1].String() {
				reputersWorker2ReportedOneOutLosses = append(reputersWorker2ReportedOneOutLosses, float64(workerLoss.Value.BigInt().Int64()))
			}
		}
	}

	// Get Stake Weighted Loss - Network Inference Loss - (L_i)
	networkStakeWeightedLoss, err := module.GetStakeWeightedLoss(reputersStakes, reputersCombinedReportedLosses)
	s.NoError(err, "Error getting stakeWeightedLoss")
	s.NotEqual(0, networkStakeWeightedLoss, "Expected worker1StakeWeightedLoss to be non-zero")

	// Get Stake Weighted Loss - OneOut Loss - (L^-_ji)
	worker1StakeWeightedOneOutLoss, err := module.GetStakeWeightedLoss(reputersStakes, reputersWorker1ReportedOneOutLosses)
	s.NoError(err, "Error getting stakeWeightedLoss")
	s.NotEqual(0, worker1StakeWeightedOneOutLoss, "Expected worker1StakeWeightedOneOutLoss to be non-zero")

	worker2StakeWeightedOneOutLoss, err := module.GetStakeWeightedLoss(reputersStakes, reputersWorker2ReportedOneOutLosses)
	s.NoError(err, "Error getting stakeWeightedLoss")
	s.NotEqual(0, worker2StakeWeightedOneOutLoss, "Expected worker2StakeWeightedOneOutLoss to be non-zero")

	// Get Worker Score - OneOut Score - (Tij)
	worker1Score := module.GetWorkerScore(networkStakeWeightedLoss, worker1StakeWeightedOneOutLoss)
	s.NotEqual(0, worker1Score, "Expected worker1Score to be non-zero")

	worker2Score := module.GetWorkerScore(networkStakeWeightedLoss, worker2StakeWeightedOneOutLoss)
	s.NotEqual(0, worker2Score, "Expected worker2Score to be non-zero")
}

func (s *ModuleTestSuite) TestGetStakeWeightedLoss() {
	timeNow := uint64(time.Now().UTC().Unix())

	// Create a topic
	topicIds, err := mockCreateTopics(s, 1)
	s.NoError(err, "Error creating topic")
	topicId := topicIds[0]
	// Create and register 2 reputers in topic
	reputers, err := mockSomeReputers(s, topicId)
	s.NoError(err, "Error creating reputers")

	// Add a lossBundle for each reputer
	losses := []cosmosMath.Uint{cosmosMath.NewUint(150), cosmosMath.NewUint(250)}

	var newLossBundles []*types.LossBundle
	for i, reputer := range reputers {
		lossBundle := types.LossBundle{
			Reputer:      reputer.String(),
			CombinedLoss: losses[i],
		}
		newLossBundles = append(newLossBundles, &lossBundle)
	}

	err = s.emissionsKeeper.InsertLossBundles(s.ctx, topicId, timeNow, types.LossBundles{LossBundles: newLossBundles})
	s.NoError(err, "Error adding lossBundle for reputer")

	var reputersStakes []float64
	var reputersReportedLosses []float64

	// Get LossBundles
	lossBundles, err := s.emissionsKeeper.GetLossBundles(s.ctx, topicId, timeNow)
	s.NoError(err, "Error getting lossBundles")

	// Get stakes and reported losses
	for _, lossBundle := range lossBundles.LossBundles {
		reputerAddr, err := sdk.AccAddressFromBech32(lossBundle.Reputer)
		s.NoError(err, "Error getting reputerAddr")

		reputerStake, err := s.emissionsKeeper.GetStakeOnTopicFromReputer(s.ctx, topicId, reputerAddr)
		s.NoError(err, "Error getting reputerStake")

		reputerStakeFloat, _ := reputerStake.BigInt().Float64()
		reputersStakes = append(reputersStakes, reputerStakeFloat)
		reputersReportedLosses = append(reputersReportedLosses, float64(lossBundle.CombinedLoss.BigInt().Int64()))
	}

	expectedStakeWeightedLoss, err := module.GetStakeWeightedLoss(reputersStakes, reputersReportedLosses)
	s.NoError(err, "Error getting stakeWeightedLoss")
	s.NotEqual(0, expectedStakeWeightedLoss, "Expected stakeWeightedLoss to be non-zero")
}
