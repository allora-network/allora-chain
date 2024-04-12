package rewards_test

import (
	"testing"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type MathTestSuite struct {
	suite.Suite
}

func (s *MathTestSuite) SetupTest() {
}

func TestMathTestSuite(t *testing.T) {
	suite.Run(t, new(MathTestSuite))
}

func (s *MathTestSuite) TestPhiSimple() {
	x := alloraMath.MustNewDecFromString("7.9997")
	p := alloraMath.NewDecFromInt64(2)
	// we expect a value very very close to 64
	result, err := rewards.Phi(p, x)
	s.Require().NoError(err)
	s.Require().True(alloraMath.InDelta(alloraMath.NewDecFromInt64(64), result, alloraMath.MustNewDecFromString("0.001")))
}

func (s *MathTestSuite) TestAdjustedStakeSimple() {
	// for this example we use
	// 3 reputers with stakes of 50_000, 100_000, 150_000
	// listening coefficients of 0.25, 0.18, 0.63 for those reputers
	// and we calculate the adjusted stake for reputer 2 (the 100_000)

	stake := alloraMath.NewDecFromInt64(100000)
	allStakes := []alloraMath.Dec{alloraMath.NewDecFromInt64(50000), stake, alloraMath.NewDecFromInt64(150000)}
	listeningCoefficient := alloraMath.MustNewDecFromString("0.18")
	allListeningCoefficients := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("0.25"),
		listeningCoefficient,
		alloraMath.MustNewDecFromString("0.63"),
	}
	numReputers := alloraMath.NewDecFromInt64(3)

	// use wolfram alpha to calculate the expected result
	// https://www.wolframalpha.com/input?i2d=true&i=1-%5C%2840%29%5C%2840%29Power%5B%5C%2840%29Power%5B%5C%2840%29ln%5C%2840%291%2BPower%5Be%2C20%5D%5C%2841%29%5C%2841%29%2C1%5D%5C%2841%29%2C-1%5D%5C%2841%29*Power%5B%5C%2840%29ln%5C%2840%291%2BPower%5Be%2C%5C%2840%29-20%5C%2840%29Divide%5B3*0.18*100%5C%2844%29000%2C0.18*100%5C%2844%29000+%2B+0.25*50%5C%2844%29000+%2B+0.63*150%5C%2844%29000%5D+-+1%5C%2841%29%5C%2841%29%5D%5C%2841%29%5C%2841%29%2C1%5D%5C%2841%29
	expected := alloraMath.MustNewDecFromString("0.4319994174428689223916439092220111693737492607160554179509")

	result, err := rewards.GetAdjustedStake(
		stake,
		allStakes,
		listeningCoefficient,
		allListeningCoefficients,
		numReputers,
	)
	s.Require().NoError(err)
	s.Require().True(alloraMath.InDelta(expected, result, alloraMath.MustNewDecFromString("0.0001")))
}

func (s *MathTestSuite) TestNormalizeAgainstSlice() {
	a := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("2.0"),
		alloraMath.MustNewDecFromString("3.0"),
		alloraMath.MustNewDecFromString("5.0"),
	}
	expected := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("0.2"),
		alloraMath.MustNewDecFromString("0.3"),
		alloraMath.MustNewDecFromString("0.5"),
	}

	result, err := rewards.ModifiedRewardFractions(a)

	s.Require().NoError(err)
	for i := range expected {
		s.Require().True(alloraMath.InDelta(expected[i], result[i], alloraMath.MustNewDecFromString("0.0001")))
	}
}

func (s *MathTestSuite) TestEntropySimple() {
	f_ij := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("0.2"),
		alloraMath.MustNewDecFromString("0.3"),
		alloraMath.MustNewDecFromString("0.5"),
	}
	N_i_eff := alloraMath.MustNewDecFromString("0.75")
	N_i := alloraMath.MustNewDecFromString("3.0")
	beta := alloraMath.MustNewDecFromString("0.25")

	// using wolfram alpha to get a sample result
	// https://www.wolframalpha.com/input?i2d=true&i=-Power%5B%5C%2840%29Divide%5B0.75%2C3%5D%5C%2841%29%2C0.25%5D*%5C%2840%290.2*ln%5C%2840%290.2%5C%2841%29+%2B+0.3*ln%5C%2840%290.3%5C%2841%29+%2B+0.5*ln%5C%2840%290.5%5C%2841%29%5C%2841%29
	expected := alloraMath.MustNewDecFromString("0.7280746285142275338742683350155248011115920866691059016669")
	result, err := rewards.Entropy(f_ij, N_i_eff, N_i, beta)
	s.Require().NoError(err)
	s.Require().True(alloraMath.InDelta(expected, result, alloraMath.MustNewDecFromString("0.0001")))
}

func (s *MathTestSuite) TestNumberRatio() {
	rewardFractions := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("0.2"),
		alloraMath.MustNewDecFromString("0.3"),
		alloraMath.MustNewDecFromString("0.4"),
		alloraMath.MustNewDecFromString("0.5"),
		alloraMath.MustNewDecFromString("0.6"),
		alloraMath.MustNewDecFromString("0.7"),
	}

	// 1 / (0.2 *0.2 + 0.3*0.3 + 0.4*0.4 + 0.5*0.5 + 0.6*0.6 + 0.7*0.7)
	// 1 / (0.04 + 0.09 + 0.16 + 0.25 + 0.36 + 0.49)
	// 1 / 1.39 = 0.719424460431654676259005145797598627787307032590051458
	expected := alloraMath.MustNewDecFromString("0.719424460431654676259005145797598627787307032590051458")

	result, err := rewards.NumberRatio(rewardFractions)
	s.Require().NoError(err, "Error calculating number ratio")
	s.Require().True(alloraMath.InDelta(expected, result, alloraMath.MustNewDecFromString("0.0001")))
}

func (s *MathTestSuite) TestNumberRatioZeroFractions() {
	zeroFractions := []alloraMath.Dec{alloraMath.ZeroDec()}

	_, err := rewards.NumberRatio(zeroFractions)
	s.Require().ErrorIs(err, emissions.ErrNumberRatioDivideByZero)
}

func (s *MathTestSuite) TestNumberRatioEmptyList() {
	emptyFractions := []alloraMath.Dec{}

	_, err := rewards.NumberRatio(emptyFractions)
	s.Require().ErrorIs(err, emissions.ErrNumberRatioInvalidSliceLength)
}

func (s *MathTestSuite) TestInferenceRewardsSimple() {
	// T_i = log L naive - log L
	// 1 = 2 - 1
	//
	// X = 0.1 + 0.4 * sigma(a * T_i - b)
	// 0.5 = 0.1 + 0.4 * sigma(8 * 1 - 7.5)
	//
	// U_i = ((1 - 0.5) * 2 * 2 * 2 ) / (2 + 2 + 4)
	// U_i = 0.5 * 8 / 8
	// U_i = 0.5
	infRewards, err := rewards.GetRewardForInferenceTaskInTopic(
		alloraMath.MustNewDecFromString("2"),   // log10(L_i- (naive))
		alloraMath.MustNewDecFromString("1"),   // log10(L_i (network))
		alloraMath.MustNewDecFromString("2.0"), // F_i
		alloraMath.MustNewDecFromString("2.0"), // G_i
		alloraMath.MustNewDecFromString("4.0"), // H_i
		alloraMath.MustNewDecFromString("2.0"), // E_i
		alloraMath.NewDecFromInt64(8),          // a
		alloraMath.MustNewDecFromString("7.5"), // b
	)
	s.Require().NoError(err)
	s.Require().True(alloraMath.InDelta(alloraMath.MustNewDecFromString("0.5"), infRewards, alloraMath.MustNewDecFromString("0.0001")))
}

func (s *MathTestSuite) TestInferenceRewardsZero() {
	result, err := rewards.GetRewardForInferenceTaskInTopic(
		alloraMath.MustNewDecFromString("2"),   // log10(L_i- (naive))
		alloraMath.MustNewDecFromString("1"),   // log10(L_i (network))
		alloraMath.MustNewDecFromString("2.0"), // F_i
		alloraMath.MustNewDecFromString("2.0"), // G_i
		alloraMath.MustNewDecFromString("4.0"), // H_i
		alloraMath.MustNewDecFromString("2.0"), // E_i
		alloraMath.NewDecFromInt64(8),          // a
		alloraMath.MustNewDecFromString("7.5"), // b
	)
	s.Require().NoError(err)
	s.Require().True(alloraMath.InDelta(alloraMath.ZeroDec(), result, alloraMath.MustNewDecFromString("0.0001")))
}

func (s *MathTestSuite) TestForecastRewardsSimple() {
	// V_i = (2 * 3 * 4 * 5) / (6 + 4 + 10)
	// V_i = 120 / 20
	// V_i = 6
	result, err := rewards.GetRewardForForecastingTaskInTopic(
		alloraMath.MustNewDecFromString("2"),   // log10(L_i- (naive))
		alloraMath.MustNewDecFromString("1"),   // log10(L_i (network))
		alloraMath.MustNewDecFromString("2.0"), // F_i
		alloraMath.MustNewDecFromString("2.0"), // G_i
		alloraMath.MustNewDecFromString("4.0"), // H_i
		alloraMath.MustNewDecFromString("2.0"), // E_i
		alloraMath.NewDecFromInt64(8),          // a
		alloraMath.MustNewDecFromString("7.5"), // b
	)
	s.Require().NoError(err)
	s.Require().True(alloraMath.InDelta(alloraMath.NewDecFromInt64(6.0), result, alloraMath.MustNewDecFromString("0.0001")))
}

func (s *MathTestSuite) TestForecastRewardsZero() {
	result, err := rewards.GetRewardForForecastingTaskInTopic(
		alloraMath.MustNewDecFromString("2"),   // log10(L_i- (naive))
		alloraMath.MustNewDecFromString("1"),   // log10(L_i (network))
		alloraMath.MustNewDecFromString("2.0"), // F_i
		alloraMath.MustNewDecFromString("2.0"), // G_i
		alloraMath.MustNewDecFromString("4.0"), // H_i
		alloraMath.MustNewDecFromString("2.0"), // E_i
		alloraMath.NewDecFromInt64(8),          // a
		alloraMath.MustNewDecFromString("7.5"), // b
	)
	s.Require().NoError(err)
	s.Require().True(alloraMath.InDelta(alloraMath.ZeroDec(), result, alloraMath.MustNewDecFromString("0.0001")))
}

func (s *MathTestSuite) TestReputerRewardSimple() {
	// W_i = (2 * 2) / (4 + 2 + 2)
	// W_i = 4 / 8
	// W_i = 0.5
	result, err := rewards.GetRewardForReputerTaskInTopic(
		alloraMath.MustNewDecFromString("4.0"),
		alloraMath.MustNewDecFromString("2.0"),
		alloraMath.MustNewDecFromString("2.0"),
		alloraMath.MustNewDecFromString("2.0"),
	)
	s.Require().NoError(err)
	s.Require().True(alloraMath.InDelta(alloraMath.MustNewDecFromString("0.5"), result, alloraMath.MustNewDecFromString("0.0001")))
}

func (s *MathTestSuite) TestReputerRewardZero() {
	result, err := rewards.GetRewardForReputerTaskInTopic(
		alloraMath.MustNewDecFromString("2"),
		alloraMath.MustNewDecFromString("2.0"),
		alloraMath.MustNewDecFromString("2.0"),
		alloraMath.ZeroDec(),
	)
	s.Require().NoError(err)
	s.Require().True(alloraMath.InDelta(alloraMath.ZeroDec(), result, alloraMath.MustNewDecFromString("0.0001")))
}

func (s *MathTestSuite) TestForecastingPerformanceScoreSimple() {
	networkInferenceLoss := alloraMath.MustNewDecFromString("100.0")
	naiveNetworkInferenceLoss := alloraMath.MustNewDecFromString("1000.0")
	score, err := rewards.ForecastingPerformanceScore(naiveNetworkInferenceLoss, networkInferenceLoss)
	s.Require().NoError(err)
	s.Require().True(alloraMath.InDelta(alloraMath.OneDec(), score, alloraMath.MustNewDecFromString("0.0001")))
}

func (s *MathTestSuite) TestSigmoidSimple() {
	x := alloraMath.MustNewDecFromString("0.5")
	result, err := rewards.Sigmoid(x)
	s.Require().NoError(err)
	s.Require().True(alloraMath.InDelta(alloraMath.MustNewDecFromString("0.6224593312018546"), result, alloraMath.MustNewDecFromString("0.0001")))
}

func (s *MathTestSuite) TestForecastingUtilitySimple() {
	a := alloraMath.MustNewDecFromString("8.0")
	b := alloraMath.MustNewDecFromString("0.5")
	forecastingPerformanceScore := alloraMath.MustNewDecFromString(".125")
	// 0.1 + 0.4 * sigma(8 * .125 - 0.5)
	// 0.1 + 0.4 * sigma(0.5)
	// 0.1 + 0.4 * 0.6224593312018546
	// 0.34898373248074184

	ret, err := rewards.ForecastingUtility(forecastingPerformanceScore, a, b)
	s.Require().NoError(err)
	s.Require().True(alloraMath.InDelta(alloraMath.MustNewDecFromString("0.34898373248074184"), ret, alloraMath.MustNewDecFromString("0.0001")))
}

func (s *MathTestSuite) TestNormalizationFactorSimple() {
	entropyInference := alloraMath.MustNewDecFromString("4.0")
	entropyForecasting := alloraMath.MustNewDecFromString("6.0")
	chi := alloraMath.MustNewDecFromString("0.5")

	// (4+6) / (1-0.5)*4 + 0.5*6
	// 10 / 2 + 3
	// 10 / 5
	// 2

	result, err := rewards.NormalizationFactor(entropyInference, entropyForecasting, chi)
	s.Require().NoError(err)

	s.Require().True(alloraMath.InDelta(alloraMath.NewDecFromInt64(2), result, alloraMath.MustNewDecFromString("0.0001")))
}

func TestStdDev(t *testing.T) {
	tests := []struct {
		name string
		data []alloraMath.Dec
		want alloraMath.Dec
	}{
		{
			name: "basic",
			data: []alloraMath.Dec{
				alloraMath.MustNewDecFromString("-0.00675"),
				alloraMath.MustNewDecFromString("-0.00622"),
				alloraMath.MustNewDecFromString("-0.01502"),
				alloraMath.MustNewDecFromString("-0.01214"),
				alloraMath.MustNewDecFromString("0.00392"),
				alloraMath.MustNewDecFromString("0.00559"),
				alloraMath.MustNewDecFromString("0.0438"),
				alloraMath.MustNewDecFromString("0.04304"),
				alloraMath.MustNewDecFromString("0.09719"),
				alloraMath.MustNewDecFromString("0.09675"),
			},
			want: alloraMath.MustNewDecFromString("0.041014924273483966"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rewards.StdDev(tt.data)
			require.NoError(t, err)
			require.True(t, alloraMath.InDelta(tt.want, got, alloraMath.MustNewDecFromString("0.0001")))
		})
	}
}

func TestGetWorkerPortionOfRewards(t *testing.T) {
	tests := []struct {
		name            string
		scores          [][]alloraMath.Dec
		preward         alloraMath.Dec
		totalRewards    alloraMath.Dec
		workerAddresses []sdk.AccAddress
		want            []alloraMath.Dec
		wantErr         bool
	}{
		{
			name: "basic",
			scores: [][]alloraMath.Dec{
				{alloraMath.MustNewDecFromString("-0.00675"), alloraMath.MustNewDecFromString("-0.00622"), alloraMath.MustNewDecFromString("-0.00388")},
				{alloraMath.MustNewDecFromString("-0.01502"), alloraMath.MustNewDecFromString("-0.01214"), alloraMath.MustNewDecFromString("-0.01554")},
				{alloraMath.MustNewDecFromString("0.00392"), alloraMath.MustNewDecFromString("0.00559"), alloraMath.MustNewDecFromString("0.00545")},
				{alloraMath.MustNewDecFromString("0.0438"), alloraMath.MustNewDecFromString("0.04304"), alloraMath.MustNewDecFromString("0.03906")},
				{alloraMath.MustNewDecFromString("0.09719"), alloraMath.MustNewDecFromString("0.09675"), alloraMath.MustNewDecFromString("0.09418")},
			},
			preward:      alloraMath.MustNewDecFromString("1.5"),
			totalRewards: alloraMath.MustNewDecFromString("1000"),
			workerAddresses: []sdk.AccAddress{
				[]byte("addr1"),
				[]byte("addr2"),
				[]byte("addr3"),
				[]byte("addr4"),
				[]byte("addr5"),
			},
			want:    []alloraMath.Dec{alloraMath.MustNewDecFromString("76.71471224853309"), alloraMath.MustNewDecFromString("55.310145462117234"), alloraMath.MustNewDecFromString("98.29388639227018"), alloraMath.MustNewDecFromString("215.38198445289035"), alloraMath.MustNewDecFromString("554.2992714441891")},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rewards.GetWorkerPortionOfRewards(tt.scores, tt.preward, tt.totalRewards, tt.workerAddresses)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetWorkerPortionOfRewards() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			for i := range tt.want {
				if !(alloraMath.InDelta(tt.want[i], got[i].Reward, alloraMath.MustNewDecFromString("0.00001"))) {
					t.Errorf(
						"GetWorkerPortionOfRewards() got = %s, want %s",
						got[i].Reward.String(),
						tt.want[i].String(),
					)
					return
				}
			}
		})
	}
}

func TestGetReputerRewardFractions(t *testing.T) {
	tests := []struct {
		name    string
		stakes  []alloraMath.Dec
		scores  []alloraMath.Dec
		preward alloraMath.Dec
		want    []alloraMath.Dec
		wantErr bool
	}{
		{
			name:    "basic",
			stakes:  []alloraMath.Dec{alloraMath.MustNewDecFromString("1178377.89152"), alloraMath.MustNewDecFromString("385287.87376"), alloraMath.MustNewDecFromString("395488.13091"), alloraMath.MustNewDecFromString("208201.11762"), alloraMath.MustNewDecFromString("369044.55988")},
			scores:  []alloraMath.Dec{alloraMath.MustNewDecFromString("17.53839"), alloraMath.MustNewDecFromString("22.63517"), alloraMath.MustNewDecFromString("26.28035"), alloraMath.MustNewDecFromString("13.51383"), alloraMath.MustNewDecFromString("15.08629")},
			preward: alloraMath.OneDec(),
			want:    []alloraMath.Dec{alloraMath.MustNewDecFromString("0.42911"), alloraMath.MustNewDecFromString("0.18108"), alloraMath.MustNewDecFromString("0.2158"), alloraMath.MustNewDecFromString("0.05842"), alloraMath.MustNewDecFromString("0.1156")},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rewards.GetReputerRewardFractions(tt.stakes, tt.scores, tt.preward)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetReputerRewardFractions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !alloraMath.SlicesInDelta(got, tt.want, alloraMath.MustNewDecFromString("0.00001")) {
				t.Errorf("GetReputerRewardFractions() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetStakeWeightedLossMatrix(t *testing.T) {
	tests := []struct {
		name                   string
		reputersAdjustedStakes []alloraMath.Dec
		reputersReportedLosses [][]alloraMath.Dec
		want                   []alloraMath.Dec
		wantErr                bool
	}{
		{
			name: "basic",
			reputersAdjustedStakes: []alloraMath.Dec{
				alloraMath.MustNewDecFromString("1.0"),
				alloraMath.MustNewDecFromString("0.76188"),
				alloraMath.MustNewDecFromString("0.7816"),
				alloraMath.MustNewDecFromString("0.40664"),
				alloraMath.MustNewDecFromString("0.71687"),
			},
			reputersReportedLosses: [][]alloraMath.Dec{
				{alloraMath.MustNewDecFromString("0.0112"), alloraMath.MustNewDecFromString("0.00231"), alloraMath.MustNewDecFromString("0.02274"), alloraMath.MustNewDecFromString("0.01299"), alloraMath.MustNewDecFromString("0.02515"), alloraMath.MustNewDecFromString("0.0185"), alloraMath.MustNewDecFromString("0.01018"), alloraMath.MustNewDecFromString("0.02105"), alloraMath.MustNewDecFromString("0.01041"), alloraMath.MustNewDecFromString("0.0183"), alloraMath.MustNewDecFromString("0.01022"), alloraMath.MustNewDecFromString("0.01333"), alloraMath.MustNewDecFromString("0.01298"), alloraMath.MustNewDecFromString("0.01023"), alloraMath.MustNewDecFromString("0.01268"), alloraMath.MustNewDecFromString("0.01381"), alloraMath.MustNewDecFromString("0.01731"), alloraMath.MustNewDecFromString("0.01238"), alloraMath.MustNewDecFromString("0.01168"), alloraMath.MustNewDecFromString("0.00929"), alloraMath.MustNewDecFromString("0.01212"), alloraMath.MustNewDecFromString("0.01806"), alloraMath.MustNewDecFromString("0.01901"), alloraMath.MustNewDecFromString("0.01828"), alloraMath.MustNewDecFromString("0.01522"), alloraMath.MustNewDecFromString("0.01833"), alloraMath.MustNewDecFromString("0.0101"), alloraMath.MustNewDecFromString("0.01224"), alloraMath.MustNewDecFromString("0.01226"), alloraMath.MustNewDecFromString("0.01474"), alloraMath.MustNewDecFromString("0.01218"), alloraMath.MustNewDecFromString("0.01604"), alloraMath.MustNewDecFromString("0.01149"), alloraMath.MustNewDecFromString("0.02075"), alloraMath.MustNewDecFromString("0.00818"), alloraMath.MustNewDecFromString("0.0116"), alloraMath.MustNewDecFromString("0.01127"), alloraMath.MustNewDecFromString("0.01495"), alloraMath.MustNewDecFromString("0.00689"), alloraMath.MustNewDecFromString("0.0108"), alloraMath.MustNewDecFromString("0.01417"), alloraMath.MustNewDecFromString("0.0124"), alloraMath.MustNewDecFromString("0.01588"), alloraMath.MustNewDecFromString("0.01012"), alloraMath.MustNewDecFromString("0.01467"), alloraMath.MustNewDecFromString("0.0128"), alloraMath.MustNewDecFromString("0.01234"), alloraMath.MustNewDecFromString("0.0148"), alloraMath.MustNewDecFromString("0.01046"), alloraMath.MustNewDecFromString("0.01192"), alloraMath.MustNewDecFromString("0.01381"), alloraMath.MustNewDecFromString("0.01687"), alloraMath.MustNewDecFromString("0.01136"), alloraMath.MustNewDecFromString("0.01185"), alloraMath.MustNewDecFromString("0.01568"), alloraMath.MustNewDecFromString("0.00949"), alloraMath.MustNewDecFromString("0.01339")},
				{alloraMath.MustNewDecFromString("0.01635"), alloraMath.MustNewDecFromString("0.00179"), alloraMath.MustNewDecFromString("0.03396"), alloraMath.MustNewDecFromString("0.0153"), alloraMath.MustNewDecFromString("0.01988"), alloraMath.MustNewDecFromString("0.00962"), alloraMath.MustNewDecFromString("0.01191"), alloraMath.MustNewDecFromString("0.01616"), alloraMath.MustNewDecFromString("0.01417"), alloraMath.MustNewDecFromString("0.01216"), alloraMath.MustNewDecFromString("0.01292"), alloraMath.MustNewDecFromString("0.01564"), alloraMath.MustNewDecFromString("0.01323"), alloraMath.MustNewDecFromString("0.01261"), alloraMath.MustNewDecFromString("0.01145"), alloraMath.MustNewDecFromString("0.0163"), alloraMath.MustNewDecFromString("0.014"), alloraMath.MustNewDecFromString("0.01373"), alloraMath.MustNewDecFromString("0.01453"), alloraMath.MustNewDecFromString("0.01207"), alloraMath.MustNewDecFromString("0.01641"), alloraMath.MustNewDecFromString("0.01601"), alloraMath.MustNewDecFromString("0.01114"), alloraMath.MustNewDecFromString("0.01259"), alloraMath.MustNewDecFromString("0.01589"), alloraMath.MustNewDecFromString("0.01229"), alloraMath.MustNewDecFromString("0.01309"), alloraMath.MustNewDecFromString("0.0138"), alloraMath.MustNewDecFromString("0.01162"), alloraMath.MustNewDecFromString("0.01145"), alloraMath.MustNewDecFromString("0.01013"), alloraMath.MustNewDecFromString("0.01208"), alloraMath.MustNewDecFromString("0.0111"), alloraMath.MustNewDecFromString("0.0118"), alloraMath.MustNewDecFromString("0.01374"), alloraMath.MustNewDecFromString("0.01428"), alloraMath.MustNewDecFromString("0.01791"), alloraMath.MustNewDecFromString("0.01288"), alloraMath.MustNewDecFromString("0.01161"), alloraMath.MustNewDecFromString("0.01151"), alloraMath.MustNewDecFromString("0.01148"), alloraMath.MustNewDecFromString("0.01284"), alloraMath.MustNewDecFromString("0.01239"), alloraMath.MustNewDecFromString("0.01023"), alloraMath.MustNewDecFromString("0.01712"), alloraMath.MustNewDecFromString("0.0116"), alloraMath.MustNewDecFromString("0.01639"), alloraMath.MustNewDecFromString("0.01043"), alloraMath.MustNewDecFromString("0.01308"), alloraMath.MustNewDecFromString("0.01455"), alloraMath.MustNewDecFromString("0.01607"), alloraMath.MustNewDecFromString("0.01205"), alloraMath.MustNewDecFromString("0.01357"), alloraMath.MustNewDecFromString("0.01108"), alloraMath.MustNewDecFromString("0.01633"), alloraMath.MustNewDecFromString("0.01208"), alloraMath.MustNewDecFromString("0.01278")},
				{alloraMath.MustNewDecFromString("0.01345"), alloraMath.MustNewDecFromString("0.00209"), alloraMath.MustNewDecFromString("0.03249"), alloraMath.MustNewDecFromString("0.01688"), alloraMath.MustNewDecFromString("0.02126"), alloraMath.MustNewDecFromString("0.01338"), alloraMath.MustNewDecFromString("0.0116"), alloraMath.MustNewDecFromString("0.01605"), alloraMath.MustNewDecFromString("0.0133"), alloraMath.MustNewDecFromString("0.01407"), alloraMath.MustNewDecFromString("0.01367"), alloraMath.MustNewDecFromString("0.01244"), alloraMath.MustNewDecFromString("0.0145"), alloraMath.MustNewDecFromString("0.01262"), alloraMath.MustNewDecFromString("0.01348"), alloraMath.MustNewDecFromString("0.01684"), alloraMath.MustNewDecFromString("0.01148"), alloraMath.MustNewDecFromString("0.01705"), alloraMath.MustNewDecFromString("0.01714"), alloraMath.MustNewDecFromString("0.0124"), alloraMath.MustNewDecFromString("0.0125"), alloraMath.MustNewDecFromString("0.01462"), alloraMath.MustNewDecFromString("0.01274"), alloraMath.MustNewDecFromString("0.01407"), alloraMath.MustNewDecFromString("0.01667"), alloraMath.MustNewDecFromString("0.01316"), alloraMath.MustNewDecFromString("0.01628"), alloraMath.MustNewDecFromString("0.01373"), alloraMath.MustNewDecFromString("0.01409"), alloraMath.MustNewDecFromString("0.01603"), alloraMath.MustNewDecFromString("0.01378"), alloraMath.MustNewDecFromString("0.01143"), alloraMath.MustNewDecFromString("0.013"), alloraMath.MustNewDecFromString("0.01644"), alloraMath.MustNewDecFromString("0.01528"), alloraMath.MustNewDecFromString("0.01441"), alloraMath.MustNewDecFromString("0.01404"), alloraMath.MustNewDecFromString("0.01402"), alloraMath.MustNewDecFromString("0.01479"), alloraMath.MustNewDecFromString("0.01417"), alloraMath.MustNewDecFromString("0.01244"), alloraMath.MustNewDecFromString("0.0116"), alloraMath.MustNewDecFromString("0.01419"), alloraMath.MustNewDecFromString("0.01497"), alloraMath.MustNewDecFromString("0.01629"), alloraMath.MustNewDecFromString("0.01514"), alloraMath.MustNewDecFromString("0.01133"), alloraMath.MustNewDecFromString("0.01339"), alloraMath.MustNewDecFromString("0.01053"), alloraMath.MustNewDecFromString("0.01424"), alloraMath.MustNewDecFromString("0.01428"), alloraMath.MustNewDecFromString("0.01446"), alloraMath.MustNewDecFromString("0.01805"), alloraMath.MustNewDecFromString("0.01229"), alloraMath.MustNewDecFromString("0.01586"), alloraMath.MustNewDecFromString("0.01234"), alloraMath.MustNewDecFromString("0.01513")},
				{alloraMath.MustNewDecFromString("0.01675"), alloraMath.MustNewDecFromString("0.00318"), alloraMath.MustNewDecFromString("0.02623"), alloraMath.MustNewDecFromString("0.02734"), alloraMath.MustNewDecFromString("0.03526"), alloraMath.MustNewDecFromString("0.02733"), alloraMath.MustNewDecFromString("0.01697"), alloraMath.MustNewDecFromString("0.01619"), alloraMath.MustNewDecFromString("0.01925"), alloraMath.MustNewDecFromString("0.02018"), alloraMath.MustNewDecFromString("0.01735"), alloraMath.MustNewDecFromString("0.01922"), alloraMath.MustNewDecFromString("0.02225"), alloraMath.MustNewDecFromString("0.0189"), alloraMath.MustNewDecFromString("0.01923"), alloraMath.MustNewDecFromString("0.03193"), alloraMath.MustNewDecFromString("0.01956"), alloraMath.MustNewDecFromString("0.01763"), alloraMath.MustNewDecFromString("0.01975"), alloraMath.MustNewDecFromString("0.01466"), alloraMath.MustNewDecFromString("0.02021"), alloraMath.MustNewDecFromString("0.01803"), alloraMath.MustNewDecFromString("0.01438"), alloraMath.MustNewDecFromString("0.01929"), alloraMath.MustNewDecFromString("0.02305"), alloraMath.MustNewDecFromString("0.02223"), alloraMath.MustNewDecFromString("0.02445"), alloraMath.MustNewDecFromString("0.01967"), alloraMath.MustNewDecFromString("0.02292"), alloraMath.MustNewDecFromString("0.01878"), alloraMath.MustNewDecFromString("0.01751"), alloraMath.MustNewDecFromString("0.02695"), alloraMath.MustNewDecFromString("0.01849"), alloraMath.MustNewDecFromString("0.01658"), alloraMath.MustNewDecFromString("0.01948"), alloraMath.MustNewDecFromString("0.01594"), alloraMath.MustNewDecFromString("0.02318"), alloraMath.MustNewDecFromString("0.01906"), alloraMath.MustNewDecFromString("0.01607"), alloraMath.MustNewDecFromString("0.01369"), alloraMath.MustNewDecFromString("0.01686"), alloraMath.MustNewDecFromString("0.01314"), alloraMath.MustNewDecFromString("0.01936"), alloraMath.MustNewDecFromString("0.01518"), alloraMath.MustNewDecFromString("0.018"), alloraMath.MustNewDecFromString("0.02212"), alloraMath.MustNewDecFromString("0.02259"), alloraMath.MustNewDecFromString("0.01674"), alloraMath.MustNewDecFromString("0.02944"), alloraMath.MustNewDecFromString("0.01796"), alloraMath.MustNewDecFromString("0.02187"), alloraMath.MustNewDecFromString("0.01895"), alloraMath.MustNewDecFromString("0.01637"), alloraMath.MustNewDecFromString("0.01594"), alloraMath.MustNewDecFromString("0.01608"), alloraMath.MustNewDecFromString("0.02203"), alloraMath.MustNewDecFromString("0.01486")},
				{alloraMath.MustNewDecFromString("0.02093"), alloraMath.MustNewDecFromString("0.00213"), alloraMath.MustNewDecFromString("0.02462"), alloraMath.MustNewDecFromString("0.0203"), alloraMath.MustNewDecFromString("0.03115"), alloraMath.MustNewDecFromString("0.01"), alloraMath.MustNewDecFromString("0.01545"), alloraMath.MustNewDecFromString("0.01785"), alloraMath.MustNewDecFromString("0.01662"), alloraMath.MustNewDecFromString("0.01156"), alloraMath.MustNewDecFromString("0.02284"), alloraMath.MustNewDecFromString("0.01475"), alloraMath.MustNewDecFromString("0.01331"), alloraMath.MustNewDecFromString("0.01592"), alloraMath.MustNewDecFromString("0.01462"), alloraMath.MustNewDecFromString("0.02333"), alloraMath.MustNewDecFromString("0.01836"), alloraMath.MustNewDecFromString("0.01465"), alloraMath.MustNewDecFromString("0.0186"), alloraMath.MustNewDecFromString("0.01566"), alloraMath.MustNewDecFromString("0.01506"), alloraMath.MustNewDecFromString("0.01678"), alloraMath.MustNewDecFromString("0.01423"), alloraMath.MustNewDecFromString("0.01658"), alloraMath.MustNewDecFromString("0.01741"), alloraMath.MustNewDecFromString("0.03491"), alloraMath.MustNewDecFromString("0.01408"), alloraMath.MustNewDecFromString("0.01191"), alloraMath.MustNewDecFromString("0.01572"), alloraMath.MustNewDecFromString("0.01355"), alloraMath.MustNewDecFromString("0.01477"), alloraMath.MustNewDecFromString("0.01662"), alloraMath.MustNewDecFromString("0.01128"), alloraMath.MustNewDecFromString("0.02581"), alloraMath.MustNewDecFromString("0.01718"), alloraMath.MustNewDecFromString("0.01705"), alloraMath.MustNewDecFromString("0.01251"), alloraMath.MustNewDecFromString("0.02158"), alloraMath.MustNewDecFromString("0.01187"), alloraMath.MustNewDecFromString("0.01504"), alloraMath.MustNewDecFromString("0.0135"), alloraMath.MustNewDecFromString("0.02432"), alloraMath.MustNewDecFromString("0.01602"), alloraMath.MustNewDecFromString("0.01194"), alloraMath.MustNewDecFromString("0.0153"), alloraMath.MustNewDecFromString("0.0199"), alloraMath.MustNewDecFromString("0.01673"), alloraMath.MustNewDecFromString("0.01049"), alloraMath.MustNewDecFromString("0.02068"), alloraMath.MustNewDecFromString("0.01573"), alloraMath.MustNewDecFromString("0.01487"), alloraMath.MustNewDecFromString("0.02639"), alloraMath.MustNewDecFromString("0.01981"), alloraMath.MustNewDecFromString("0.02123"), alloraMath.MustNewDecFromString("0.02134"), alloraMath.MustNewDecFromString("0.0217"), alloraMath.MustNewDecFromString("0.01177")},
			},
			want: []alloraMath.Dec{
				alloraMath.MustNewDecFromString("0.01489"), alloraMath.MustNewDecFromString("0.00219"), alloraMath.MustNewDecFromString("0.02752"), alloraMath.MustNewDecFromString("0.01684"), alloraMath.MustNewDecFromString("0.02502"), alloraMath.MustNewDecFromString("0.01395"), alloraMath.MustNewDecFromString("0.01242"), alloraMath.MustNewDecFromString("0.01769"), alloraMath.MustNewDecFromString("0.01372"),
				alloraMath.MustNewDecFromString("0.01469"), alloraMath.MustNewDecFromString("0.01416"), alloraMath.MustNewDecFromString("0.01442"), alloraMath.MustNewDecFromString("0.01423"), alloraMath.MustNewDecFromString("0.01304"), alloraMath.MustNewDecFromString("0.01354"), alloraMath.MustNewDecFromString("0.01813"), alloraMath.MustNewDecFromString("0.01556"), alloraMath.MustNewDecFromString("0.01455"),
				alloraMath.MustNewDecFromString("0.0154"), alloraMath.MustNewDecFromString("0.01215"), alloraMath.MustNewDecFromString("0.01435"), alloraMath.MustNewDecFromString("0.01659"), alloraMath.MustNewDecFromString("0.01431"), alloraMath.MustNewDecFromString("0.01579"), alloraMath.MustNewDecFromString("0.01683"), alloraMath.MustNewDecFromString("0.01821"), alloraMath.MustNewDecFromString("0.01389"),
				alloraMath.MustNewDecFromString("0.01348"), alloraMath.MustNewDecFromString("0.01405"), alloraMath.MustNewDecFromString("0.01439"), alloraMath.MustNewDecFromString("0.01301"), alloraMath.MustNewDecFromString("0.01501"), alloraMath.MustNewDecFromString("0.0123"), alloraMath.MustNewDecFromString("0.01788"), alloraMath.MustNewDecFromString("0.01325"), alloraMath.MustNewDecFromString("0.01417"),
				alloraMath.MustNewDecFromString("0.01438"), alloraMath.MustNewDecFromString("0.01578"), alloraMath.MustNewDecFromString("0.01104"), alloraMath.MustNewDecFromString("0.0127"), alloraMath.MustNewDecFromString("0.01332"), alloraMath.MustNewDecFromString("0.01414"), alloraMath.MustNewDecFromString("0.01508"), alloraMath.MustNewDecFromString("0.01191"), alloraMath.MustNewDecFromString("0.01598"),
				alloraMath.MustNewDecFromString("0.01506"), alloraMath.MustNewDecFromString("0.01459"), alloraMath.MustNewDecFromString("0.01277"), alloraMath.MustNewDecFromString("0.01406"), alloraMath.MustNewDecFromString("0.01426"), alloraMath.MustNewDecFromString("0.01532"), alloraMath.MustNewDecFromString("0.01683"), alloraMath.MustNewDecFromString("0.0151"), alloraMath.MustNewDecFromString("0.01364"),
				alloraMath.MustNewDecFromString("0.01688"), alloraMath.MustNewDecFromString("0.01362"), alloraMath.MustNewDecFromString("0.01342"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := rewards.GetStakeWeightedLossMatrix(tt.reputersAdjustedStakes, tt.reputersReportedLosses)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetStakeWeightedLossMatrix() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !alloraMath.SlicesInDelta(got, tt.want, alloraMath.MustNewDecFromString("1e-5")) {
				t.Errorf("GetStakeWeightedLossMatrix() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetStakeWeightedLossMatrixWithMissingLosses(t *testing.T) {
	tests := []struct {
		name                   string
		reputersAdjustedStakes []alloraMath.Dec
		reputersReportedLosses [][]alloraMath.Dec
		want                   []alloraMath.Dec
		wantErr                bool
	}{
		{
			name: "basic",
			reputersAdjustedStakes: []alloraMath.Dec{
				alloraMath.MustNewDecFromString("1.0"),
				alloraMath.MustNewDecFromString("1.0"),
				alloraMath.MustNewDecFromString("1.0"),
			},
			reputersReportedLosses: [][]alloraMath.Dec{
				{alloraMath.MustNewDecFromString("1.0"), alloraMath.MustNewDecFromString("2.0"), alloraMath.MustNewDecFromString("3.0"), alloraMath.MustNewDecFromString("4.0")},
				{alloraMath.MustNewDecFromString("2.0"), alloraMath.NewNaN(), alloraMath.MustNewDecFromString("5.0"), alloraMath.MustNewDecFromString("3.0")},
				{alloraMath.NewNaN(), alloraMath.NewNaN(), alloraMath.MustNewDecFromString("1.0"), alloraMath.MustNewDecFromString("2.0")},
			},
			want: []alloraMath.Dec{
				alloraMath.MustNewDecFromString("1.41421"), alloraMath.MustNewDecFromString("2.00000"), alloraMath.MustNewDecFromString("2.46621"), alloraMath.MustNewDecFromString("2.88449"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := rewards.GetStakeWeightedLossMatrix(tt.reputersAdjustedStakes, tt.reputersReportedLosses)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStakeWeightedLossMatrix() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !alloraMath.SlicesInDelta(got, tt.want, alloraMath.MustNewDecFromString("1e-5")) {
				t.Errorf("GetStakeWeightedLossMatrix() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetStakeWeightedLoss(t *testing.T) {
	tests := []struct {
		name                   string
		reputersStakes         []alloraMath.Dec
		reputersReportedLosses []alloraMath.Dec
		want                   alloraMath.Dec
		wantErr                bool
	}{
		{
			name:                   "simple average",
			reputersStakes:         []alloraMath.Dec{alloraMath.MustNewDecFromString("1176644.37627"), alloraMath.MustNewDecFromString("384623.3607"), alloraMath.MustNewDecFromString("394676.13226"), alloraMath.MustNewDecFromString("207999.66194"), alloraMath.MustNewDecFromString("368582.76542")},
			reputersReportedLosses: []alloraMath.Dec{alloraMath.MustNewDecFromString("0.0112"), alloraMath.MustNewDecFromString("0.01635"), alloraMath.MustNewDecFromString("0.01345"), alloraMath.MustNewDecFromString("0.01675"), alloraMath.MustNewDecFromString("0.02093")},
			want:                   alloraMath.MustNewDecFromString("0.01381883491416319"),
			wantErr:                false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rewards.GetStakeWeightedLoss(tt.reputersStakes, tt.reputersReportedLosses)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStakeWeightedLoss() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !(alloraMath.InDelta(tt.want, got, alloraMath.MustNewDecFromString("0.00001"))) {
				t.Errorf("GetStakeWeightedLoss() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetWorkerScore(t *testing.T) {
	tests := []struct {
		name         string
		losses       alloraMath.Dec
		lossesOneOut alloraMath.Dec
		want         alloraMath.Dec
	}{
		{"basic", alloraMath.MustNewDecFromString("0.011411892282242868"), alloraMath.MustNewDecFromString("0.01344474872292"), alloraMath.MustNewDecFromString("0.07119502617735574")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rewards.GetWorkerScore(tt.losses, tt.lossesOneOut)
			require.NoError(t, err)
			if !(alloraMath.InDelta(tt.want, got, alloraMath.MustNewDecFromString("0.00000000000001"))) {
				t.Errorf("GetWorkerScore() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetFinalWorkerScoreForecastTask(t *testing.T) {
	tests := []struct {
		name        string
		scoreOneIn  alloraMath.Dec
		scoreOneOut alloraMath.Dec
		fUniqueAgg  alloraMath.Dec
		want        alloraMath.Dec
	}{
		{
			"basic",
			alloraMath.MustNewDecFromString("0.07300629674057668"),
			alloraMath.MustNewDecFromString("-0.009510726019112292"),
			alloraMath.MustNewDecFromString("0.0625"),
			alloraMath.MustNewDecFromString("-0.004353412096631731"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rewards.GetFinalWorkerScoreForecastTask(tt.scoreOneIn, tt.scoreOneOut, tt.fUniqueAgg)
			require.NoError(t, err)
			if !alloraMath.InDelta(tt.want, got, alloraMath.MustNewDecFromString("0.00001")) {
				t.Errorf("GetFinalWorkerScoreForecastTask() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAllConsensusScores(t *testing.T) {
	allLosses := [][]alloraMath.Dec{
		{alloraMath.MustNewDecFromString("0.0112"), alloraMath.MustNewDecFromString("0.00231"), alloraMath.MustNewDecFromString("0.02274"), alloraMath.MustNewDecFromString("0.01299"), alloraMath.MustNewDecFromString("0.02515"), alloraMath.MustNewDecFromString("0.0185"), alloraMath.MustNewDecFromString("0.01018"), alloraMath.MustNewDecFromString("0.02105"), alloraMath.MustNewDecFromString("0.01041"), alloraMath.MustNewDecFromString("0.0183"), alloraMath.MustNewDecFromString("0.01022"), alloraMath.MustNewDecFromString("0.01333"), alloraMath.MustNewDecFromString("0.01298"), alloraMath.MustNewDecFromString("0.01023"), alloraMath.MustNewDecFromString("0.01268"), alloraMath.MustNewDecFromString("0.01381"), alloraMath.MustNewDecFromString("0.01731"), alloraMath.MustNewDecFromString("0.01238"), alloraMath.MustNewDecFromString("0.01168"), alloraMath.MustNewDecFromString("0.00929"), alloraMath.MustNewDecFromString("0.01212"), alloraMath.MustNewDecFromString("0.01806"), alloraMath.MustNewDecFromString("0.01901"), alloraMath.MustNewDecFromString("0.01828"), alloraMath.MustNewDecFromString("0.01522"), alloraMath.MustNewDecFromString("0.01833"), alloraMath.MustNewDecFromString("0.0101"), alloraMath.MustNewDecFromString("0.01224"), alloraMath.MustNewDecFromString("0.01226"), alloraMath.MustNewDecFromString("0.01474"), alloraMath.MustNewDecFromString("0.01218"), alloraMath.MustNewDecFromString("0.01604"), alloraMath.MustNewDecFromString("0.01149"), alloraMath.MustNewDecFromString("0.02075"), alloraMath.MustNewDecFromString("0.00818"), alloraMath.MustNewDecFromString("0.0116"), alloraMath.MustNewDecFromString("0.01127"), alloraMath.MustNewDecFromString("0.01495"), alloraMath.MustNewDecFromString("0.00689"), alloraMath.MustNewDecFromString("0.0108"), alloraMath.MustNewDecFromString("0.01417"), alloraMath.MustNewDecFromString("0.0124"), alloraMath.MustNewDecFromString("0.01588"), alloraMath.MustNewDecFromString("0.01012"), alloraMath.MustNewDecFromString("0.01467"), alloraMath.MustNewDecFromString("0.0128"), alloraMath.MustNewDecFromString("0.01234"), alloraMath.MustNewDecFromString("0.0148"), alloraMath.MustNewDecFromString("0.01046"), alloraMath.MustNewDecFromString("0.01192"), alloraMath.MustNewDecFromString("0.01381"), alloraMath.MustNewDecFromString("0.01687"), alloraMath.MustNewDecFromString("0.01136"), alloraMath.MustNewDecFromString("0.01185"), alloraMath.MustNewDecFromString("0.01568"), alloraMath.MustNewDecFromString("0.00949"), alloraMath.MustNewDecFromString("0.01339")},
		{alloraMath.MustNewDecFromString("0.01635"), alloraMath.MustNewDecFromString("0.00179"), alloraMath.MustNewDecFromString("0.03396"), alloraMath.MustNewDecFromString("0.0153"), alloraMath.MustNewDecFromString("0.01988"), alloraMath.MustNewDecFromString("0.00962"), alloraMath.MustNewDecFromString("0.01191"), alloraMath.MustNewDecFromString("0.01616"), alloraMath.MustNewDecFromString("0.01417"), alloraMath.MustNewDecFromString("0.01216"), alloraMath.MustNewDecFromString("0.01292"), alloraMath.MustNewDecFromString("0.01564"), alloraMath.MustNewDecFromString("0.01323"), alloraMath.MustNewDecFromString("0.01261"), alloraMath.MustNewDecFromString("0.01145"), alloraMath.MustNewDecFromString("0.0163"), alloraMath.MustNewDecFromString("0.014"), alloraMath.MustNewDecFromString("0.01373"), alloraMath.MustNewDecFromString("0.01453"), alloraMath.MustNewDecFromString("0.01207"), alloraMath.MustNewDecFromString("0.01641"), alloraMath.MustNewDecFromString("0.01601"), alloraMath.MustNewDecFromString("0.01114"), alloraMath.MustNewDecFromString("0.01259"), alloraMath.MustNewDecFromString("0.01589"), alloraMath.MustNewDecFromString("0.01229"), alloraMath.MustNewDecFromString("0.01309"), alloraMath.MustNewDecFromString("0.0138"), alloraMath.MustNewDecFromString("0.01162"), alloraMath.MustNewDecFromString("0.01145"), alloraMath.MustNewDecFromString("0.01013"), alloraMath.MustNewDecFromString("0.01208"), alloraMath.MustNewDecFromString("0.0111"), alloraMath.MustNewDecFromString("0.0118"), alloraMath.MustNewDecFromString("0.01374"), alloraMath.MustNewDecFromString("0.01428"), alloraMath.MustNewDecFromString("0.01791"), alloraMath.MustNewDecFromString("0.01288"), alloraMath.MustNewDecFromString("0.01161"), alloraMath.MustNewDecFromString("0.01151"), alloraMath.MustNewDecFromString("0.01148"), alloraMath.MustNewDecFromString("0.01284"), alloraMath.MustNewDecFromString("0.01239"), alloraMath.MustNewDecFromString("0.01023"), alloraMath.MustNewDecFromString("0.01712"), alloraMath.MustNewDecFromString("0.0116"), alloraMath.MustNewDecFromString("0.01639"), alloraMath.MustNewDecFromString("0.01043"), alloraMath.MustNewDecFromString("0.01308"), alloraMath.MustNewDecFromString("0.01455"), alloraMath.MustNewDecFromString("0.01607"), alloraMath.MustNewDecFromString("0.01205"), alloraMath.MustNewDecFromString("0.01357"), alloraMath.MustNewDecFromString("0.01108"), alloraMath.MustNewDecFromString("0.01633"), alloraMath.MustNewDecFromString("0.01208"), alloraMath.MustNewDecFromString("0.01278")},
		{alloraMath.MustNewDecFromString("0.01345"), alloraMath.MustNewDecFromString("0.00209"), alloraMath.MustNewDecFromString("0.03249"), alloraMath.MustNewDecFromString("0.01688"), alloraMath.MustNewDecFromString("0.02126"), alloraMath.MustNewDecFromString("0.01338"), alloraMath.MustNewDecFromString("0.0116"), alloraMath.MustNewDecFromString("0.01605"), alloraMath.MustNewDecFromString("0.0133"), alloraMath.MustNewDecFromString("0.01407"), alloraMath.MustNewDecFromString("0.01367"), alloraMath.MustNewDecFromString("0.01244"), alloraMath.MustNewDecFromString("0.0145"), alloraMath.MustNewDecFromString("0.01262"), alloraMath.MustNewDecFromString("0.01348"), alloraMath.MustNewDecFromString("0.01684"), alloraMath.MustNewDecFromString("0.01148"), alloraMath.MustNewDecFromString("0.01705"), alloraMath.MustNewDecFromString("0.01714"), alloraMath.MustNewDecFromString("0.0124"), alloraMath.MustNewDecFromString("0.0125"), alloraMath.MustNewDecFromString("0.01462"), alloraMath.MustNewDecFromString("0.01274"), alloraMath.MustNewDecFromString("0.01407"), alloraMath.MustNewDecFromString("0.01667"), alloraMath.MustNewDecFromString("0.01316"), alloraMath.MustNewDecFromString("0.01628"), alloraMath.MustNewDecFromString("0.01373"), alloraMath.MustNewDecFromString("0.01409"), alloraMath.MustNewDecFromString("0.01603"), alloraMath.MustNewDecFromString("0.01378"), alloraMath.MustNewDecFromString("0.01143"), alloraMath.MustNewDecFromString("0.013"), alloraMath.MustNewDecFromString("0.01644"), alloraMath.MustNewDecFromString("0.01528"), alloraMath.MustNewDecFromString("0.01441"), alloraMath.MustNewDecFromString("0.01404"), alloraMath.MustNewDecFromString("0.01402"), alloraMath.MustNewDecFromString("0.01479"), alloraMath.MustNewDecFromString("0.01417"), alloraMath.MustNewDecFromString("0.01244"), alloraMath.MustNewDecFromString("0.0116"), alloraMath.MustNewDecFromString("0.01419"), alloraMath.MustNewDecFromString("0.01497"), alloraMath.MustNewDecFromString("0.01629"), alloraMath.MustNewDecFromString("0.01514"), alloraMath.MustNewDecFromString("0.01133"), alloraMath.MustNewDecFromString("0.01339"), alloraMath.MustNewDecFromString("0.01053"), alloraMath.MustNewDecFromString("0.01424"), alloraMath.MustNewDecFromString("0.01428"), alloraMath.MustNewDecFromString("0.01446"), alloraMath.MustNewDecFromString("0.01805"), alloraMath.MustNewDecFromString("0.01229"), alloraMath.MustNewDecFromString("0.01586"), alloraMath.MustNewDecFromString("0.01234"), alloraMath.MustNewDecFromString("0.01513")},
		{alloraMath.MustNewDecFromString("0.01675"), alloraMath.MustNewDecFromString("0.00318"), alloraMath.MustNewDecFromString("0.02623"), alloraMath.MustNewDecFromString("0.02734"), alloraMath.MustNewDecFromString("0.03526"), alloraMath.MustNewDecFromString("0.02733"), alloraMath.MustNewDecFromString("0.01697"), alloraMath.MustNewDecFromString("0.01619"), alloraMath.MustNewDecFromString("0.01925"), alloraMath.MustNewDecFromString("0.02018"), alloraMath.MustNewDecFromString("0.01735"), alloraMath.MustNewDecFromString("0.01922"), alloraMath.MustNewDecFromString("0.02225"), alloraMath.MustNewDecFromString("0.0189"), alloraMath.MustNewDecFromString("0.01923"), alloraMath.MustNewDecFromString("0.03193"), alloraMath.MustNewDecFromString("0.01956"), alloraMath.MustNewDecFromString("0.01763"), alloraMath.MustNewDecFromString("0.01975"), alloraMath.MustNewDecFromString("0.01466"), alloraMath.MustNewDecFromString("0.02021"), alloraMath.MustNewDecFromString("0.01803"), alloraMath.MustNewDecFromString("0.01438"), alloraMath.MustNewDecFromString("0.01929"), alloraMath.MustNewDecFromString("0.02305"), alloraMath.MustNewDecFromString("0.02223"), alloraMath.MustNewDecFromString("0.02445"), alloraMath.MustNewDecFromString("0.01967"), alloraMath.MustNewDecFromString("0.02292"), alloraMath.MustNewDecFromString("0.01878"), alloraMath.MustNewDecFromString("0.01751"), alloraMath.MustNewDecFromString("0.02695"), alloraMath.MustNewDecFromString("0.01849"), alloraMath.MustNewDecFromString("0.01658"), alloraMath.MustNewDecFromString("0.01948"), alloraMath.MustNewDecFromString("0.01594"), alloraMath.MustNewDecFromString("0.02318"), alloraMath.MustNewDecFromString("0.01906"), alloraMath.MustNewDecFromString("0.01607"), alloraMath.MustNewDecFromString("0.01369"), alloraMath.MustNewDecFromString("0.01686"), alloraMath.MustNewDecFromString("0.01314"), alloraMath.MustNewDecFromString("0.01936"), alloraMath.MustNewDecFromString("0.01518"), alloraMath.MustNewDecFromString("0.018"), alloraMath.MustNewDecFromString("0.02212"), alloraMath.MustNewDecFromString("0.02259"), alloraMath.MustNewDecFromString("0.01674"), alloraMath.MustNewDecFromString("0.02944"), alloraMath.MustNewDecFromString("0.01796"), alloraMath.MustNewDecFromString("0.02187"), alloraMath.MustNewDecFromString("0.01895"), alloraMath.MustNewDecFromString("0.01637"), alloraMath.MustNewDecFromString("0.01594"), alloraMath.MustNewDecFromString("0.01608"), alloraMath.MustNewDecFromString("0.02203"), alloraMath.MustNewDecFromString("0.01486")},
		{alloraMath.MustNewDecFromString("0.02093"), alloraMath.MustNewDecFromString("0.00213"), alloraMath.MustNewDecFromString("0.02462"), alloraMath.MustNewDecFromString("0.0203"), alloraMath.MustNewDecFromString("0.03115"), alloraMath.MustNewDecFromString("0.01"), alloraMath.MustNewDecFromString("0.01545"), alloraMath.MustNewDecFromString("0.01785"), alloraMath.MustNewDecFromString("0.01662"), alloraMath.MustNewDecFromString("0.01156"), alloraMath.MustNewDecFromString("0.02284"), alloraMath.MustNewDecFromString("0.01475"), alloraMath.MustNewDecFromString("0.01331"), alloraMath.MustNewDecFromString("0.01592"), alloraMath.MustNewDecFromString("0.01462"), alloraMath.MustNewDecFromString("0.02333"), alloraMath.MustNewDecFromString("0.01836"), alloraMath.MustNewDecFromString("0.01465"), alloraMath.MustNewDecFromString("0.0186"), alloraMath.MustNewDecFromString("0.01566"), alloraMath.MustNewDecFromString("0.01506"), alloraMath.MustNewDecFromString("0.01678"), alloraMath.MustNewDecFromString("0.01423"), alloraMath.MustNewDecFromString("0.01658"), alloraMath.MustNewDecFromString("0.01741"), alloraMath.MustNewDecFromString("0.03491"), alloraMath.MustNewDecFromString("0.01408"), alloraMath.MustNewDecFromString("0.01191"), alloraMath.MustNewDecFromString("0.01572"), alloraMath.MustNewDecFromString("0.01355"), alloraMath.MustNewDecFromString("0.01477"), alloraMath.MustNewDecFromString("0.01662"), alloraMath.MustNewDecFromString("0.01128"), alloraMath.MustNewDecFromString("0.02581"), alloraMath.MustNewDecFromString("0.01718"), alloraMath.MustNewDecFromString("0.01705"), alloraMath.MustNewDecFromString("0.01251"), alloraMath.MustNewDecFromString("0.02158"), alloraMath.MustNewDecFromString("0.01187"), alloraMath.MustNewDecFromString("0.01504"), alloraMath.MustNewDecFromString("0.0135"), alloraMath.MustNewDecFromString("0.02432"), alloraMath.MustNewDecFromString("0.01602"), alloraMath.MustNewDecFromString("0.01194"), alloraMath.MustNewDecFromString("0.0153"), alloraMath.MustNewDecFromString("0.0199"), alloraMath.MustNewDecFromString("0.01673"), alloraMath.MustNewDecFromString("0.01049"), alloraMath.MustNewDecFromString("0.02068"), alloraMath.MustNewDecFromString("0.01573"), alloraMath.MustNewDecFromString("0.01487"), alloraMath.MustNewDecFromString("0.02639"), alloraMath.MustNewDecFromString("0.01981"), alloraMath.MustNewDecFromString("0.02123"), alloraMath.MustNewDecFromString("0.02134"), alloraMath.MustNewDecFromString("0.0217"), alloraMath.MustNewDecFromString("0.01177")},
	}
	stakes := []alloraMath.Dec{alloraMath.MustNewDecFromString("1176644.37627"), alloraMath.MustNewDecFromString("384623.3607"), alloraMath.MustNewDecFromString("394676.13226"), alloraMath.MustNewDecFromString("207999.66194"), alloraMath.MustNewDecFromString("368582.76542")}
	allListeningCoefficients := []alloraMath.Dec{alloraMath.MustNewDecFromString("1.0"), alloraMath.MustNewDecFromString("1.0"), alloraMath.MustNewDecFromString("1.0"), alloraMath.MustNewDecFromString("1.0"), alloraMath.MustNewDecFromString("1.0")}
	var numReputers int64 = 5
	want := []alloraMath.Dec{alloraMath.MustNewDecFromString("17.4346"), alloraMath.MustNewDecFromString("20.13897"), alloraMath.MustNewDecFromString("24.08276"), alloraMath.MustNewDecFromString("11.41393"), alloraMath.MustNewDecFromString("15.33319")}
	wantErr := false

	got, err := rewards.GetAllConsensusScores(allLosses, stakes, allListeningCoefficients, numReputers)
	if (err != nil) != wantErr {
		t.Errorf("GetAllConsensusScores() error = %v, wantErr %v", err, wantErr)
		return
	}

	if !alloraMath.SlicesInDelta(got, want, alloraMath.MustNewDecFromString("0.01")) {
		t.Errorf("GetAllConsensusScores() got = %v, want %v", got, want)
	}
}

// something about this test takes too long and hangs
// must investigate further
/*
func TestGetAllReputersOutput(t *testing.T) {
	allLosses := [][]alloraMath.Dec{
		{alloraMath.MustNewDecFromString("0.0112"), alloraMath.MustNewDecFromString("0.00231"), alloraMath.MustNewDecFromString("0.02274"), alloraMath.MustNewDecFromString("0.01299"), alloraMath.MustNewDecFromString("0.02515"), alloraMath.MustNewDecFromString("0.0185"), alloraMath.MustNewDecFromString("0.01018"), alloraMath.MustNewDecFromString("0.02105"), alloraMath.MustNewDecFromString("0.01041"), alloraMath.MustNewDecFromString("0.0183"), alloraMath.MustNewDecFromString("0.01022"), alloraMath.MustNewDecFromString("0.01333"), alloraMath.MustNewDecFromString("0.01298"), alloraMath.MustNewDecFromString("0.01023"), alloraMath.MustNewDecFromString("0.01268"), alloraMath.MustNewDecFromString("0.01381"), alloraMath.MustNewDecFromString("0.01731"), alloraMath.MustNewDecFromString("0.01238"), alloraMath.MustNewDecFromString("0.01168"), alloraMath.MustNewDecFromString("0.00929"), alloraMath.MustNewDecFromString("0.01212"), alloraMath.MustNewDecFromString("0.01806"), alloraMath.MustNewDecFromString("0.01901"), alloraMath.MustNewDecFromString("0.01828"), alloraMath.MustNewDecFromString("0.01522"), alloraMath.MustNewDecFromString("0.01833"), alloraMath.MustNewDecFromString("0.0101"), alloraMath.MustNewDecFromString("0.01224"), alloraMath.MustNewDecFromString("0.01226"), alloraMath.MustNewDecFromString("0.01474"), alloraMath.MustNewDecFromString("0.01218"), alloraMath.MustNewDecFromString("0.01604"), alloraMath.MustNewDecFromString("0.01149"), alloraMath.MustNewDecFromString("0.02075"), alloraMath.MustNewDecFromString("0.00818"), alloraMath.MustNewDecFromString("0.0116"), alloraMath.MustNewDecFromString("0.01127"), alloraMath.MustNewDecFromString("0.01495"), alloraMath.MustNewDecFromString("0.00689"), alloraMath.MustNewDecFromString("0.0108"), alloraMath.MustNewDecFromString("0.01417"), alloraMath.MustNewDecFromString("0.0124"), alloraMath.MustNewDecFromString("0.01588"), alloraMath.MustNewDecFromString("0.01012"), alloraMath.MustNewDecFromString("0.01467"), alloraMath.MustNewDecFromString("0.0128"), alloraMath.MustNewDecFromString("0.01234"), alloraMath.MustNewDecFromString("0.0148"), alloraMath.MustNewDecFromString("0.01046"), alloraMath.MustNewDecFromString("0.01192"), alloraMath.MustNewDecFromString("0.01381"), alloraMath.MustNewDecFromString("0.01687"), alloraMath.MustNewDecFromString("0.01136"), alloraMath.MustNewDecFromString("0.01185"), alloraMath.MustNewDecFromString("0.01568"), alloraMath.MustNewDecFromString("0.00949"), alloraMath.MustNewDecFromString("0.01339")},
		{alloraMath.MustNewDecFromString("0.01635"), alloraMath.MustNewDecFromString("0.00179"), alloraMath.MustNewDecFromString("0.03396"), alloraMath.MustNewDecFromString("0.0153"), alloraMath.MustNewDecFromString("0.01988"), alloraMath.MustNewDecFromString("0.00962"), alloraMath.MustNewDecFromString("0.01191"), alloraMath.MustNewDecFromString("0.01616"), alloraMath.MustNewDecFromString("0.01417"), alloraMath.MustNewDecFromString("0.01216"), alloraMath.MustNewDecFromString("0.01292"), alloraMath.MustNewDecFromString("0.01564"), alloraMath.MustNewDecFromString("0.01323"), alloraMath.MustNewDecFromString("0.01261"), alloraMath.MustNewDecFromString("0.01145"), alloraMath.MustNewDecFromString("0.0163"), alloraMath.MustNewDecFromString("0.014"), alloraMath.MustNewDecFromString("0.01373"), alloraMath.MustNewDecFromString("0.01453"), alloraMath.MustNewDecFromString("0.01207"), alloraMath.MustNewDecFromString("0.01641"), alloraMath.MustNewDecFromString("0.01601"), alloraMath.MustNewDecFromString("0.01114"), alloraMath.MustNewDecFromString("0.01259"), alloraMath.MustNewDecFromString("0.01589"), alloraMath.MustNewDecFromString("0.01229"), alloraMath.MustNewDecFromString("0.01309"), alloraMath.MustNewDecFromString("0.0138"), alloraMath.MustNewDecFromString("0.01162"), alloraMath.MustNewDecFromString("0.01145"), alloraMath.MustNewDecFromString("0.01013"), alloraMath.MustNewDecFromString("0.01208"), alloraMath.MustNewDecFromString("0.0111"), alloraMath.MustNewDecFromString("0.0118"), alloraMath.MustNewDecFromString("0.01374"), alloraMath.MustNewDecFromString("0.01428"), alloraMath.MustNewDecFromString("0.01791"), alloraMath.MustNewDecFromString("0.01288"), alloraMath.MustNewDecFromString("0.01161"), alloraMath.MustNewDecFromString("0.01151"), alloraMath.MustNewDecFromString("0.01148"), alloraMath.MustNewDecFromString("0.01284"), alloraMath.MustNewDecFromString("0.01239"), alloraMath.MustNewDecFromString("0.01023"), alloraMath.MustNewDecFromString("0.01712"), alloraMath.MustNewDecFromString("0.0116"), alloraMath.MustNewDecFromString("0.01639"), alloraMath.MustNewDecFromString("0.01043"), alloraMath.MustNewDecFromString("0.01308"), alloraMath.MustNewDecFromString("0.01455"), alloraMath.MustNewDecFromString("0.01607"), alloraMath.MustNewDecFromString("0.01205"), alloraMath.MustNewDecFromString("0.01357"), alloraMath.MustNewDecFromString("0.01108"), alloraMath.MustNewDecFromString("0.01633"), alloraMath.MustNewDecFromString("0.01208"), alloraMath.MustNewDecFromString("0.01278")},
		{alloraMath.MustNewDecFromString("0.01345"), alloraMath.MustNewDecFromString("0.00209"), alloraMath.MustNewDecFromString("0.03249"), alloraMath.MustNewDecFromString("0.01688"), alloraMath.MustNewDecFromString("0.02126"), alloraMath.MustNewDecFromString("0.01338"), alloraMath.MustNewDecFromString("0.0116"), alloraMath.MustNewDecFromString("0.01605"), alloraMath.MustNewDecFromString("0.0133"), alloraMath.MustNewDecFromString("0.01407"), alloraMath.MustNewDecFromString("0.01367"), alloraMath.MustNewDecFromString("0.01244"), alloraMath.MustNewDecFromString("0.0145"), alloraMath.MustNewDecFromString("0.01262"), alloraMath.MustNewDecFromString("0.01348"), alloraMath.MustNewDecFromString("0.01684"), alloraMath.MustNewDecFromString("0.01148"), alloraMath.MustNewDecFromString("0.01705"), alloraMath.MustNewDecFromString("0.01714"), alloraMath.MustNewDecFromString("0.0124"), alloraMath.MustNewDecFromString("0.0125"), alloraMath.MustNewDecFromString("0.01462"), alloraMath.MustNewDecFromString("0.01274"), alloraMath.MustNewDecFromString("0.01407"), alloraMath.MustNewDecFromString("0.01667"), alloraMath.MustNewDecFromString("0.01316"), alloraMath.MustNewDecFromString("0.01628"), alloraMath.MustNewDecFromString("0.01373"), alloraMath.MustNewDecFromString("0.01409"), alloraMath.MustNewDecFromString("0.01603"), alloraMath.MustNewDecFromString("0.01378"), alloraMath.MustNewDecFromString("0.01143"), alloraMath.MustNewDecFromString("0.013"), alloraMath.MustNewDecFromString("0.01644"), alloraMath.MustNewDecFromString("0.01528"), alloraMath.MustNewDecFromString("0.01441"), alloraMath.MustNewDecFromString("0.01404"), alloraMath.MustNewDecFromString("0.01402"), alloraMath.MustNewDecFromString("0.01479"), alloraMath.MustNewDecFromString("0.01417"), alloraMath.MustNewDecFromString("0.01244"), alloraMath.MustNewDecFromString("0.0116"), alloraMath.MustNewDecFromString("0.01419"), alloraMath.MustNewDecFromString("0.01497"), alloraMath.MustNewDecFromString("0.01629"), alloraMath.MustNewDecFromString("0.01514"), alloraMath.MustNewDecFromString("0.01133"), alloraMath.MustNewDecFromString("0.01339"), alloraMath.MustNewDecFromString("0.01053"), alloraMath.MustNewDecFromString("0.01424"), alloraMath.MustNewDecFromString("0.01428"), alloraMath.MustNewDecFromString("0.01446"), alloraMath.MustNewDecFromString("0.01805"), alloraMath.MustNewDecFromString("0.01229"), alloraMath.MustNewDecFromString("0.01586"), alloraMath.MustNewDecFromString("0.01234"), alloraMath.MustNewDecFromString("0.01513")},
		{alloraMath.MustNewDecFromString("0.01675"), alloraMath.MustNewDecFromString("0.00318"), alloraMath.MustNewDecFromString("0.02623"), alloraMath.MustNewDecFromString("0.02734"), alloraMath.MustNewDecFromString("0.03526"), alloraMath.MustNewDecFromString("0.02733"), alloraMath.MustNewDecFromString("0.01697"), alloraMath.MustNewDecFromString("0.01619"), alloraMath.MustNewDecFromString("0.01925"), alloraMath.MustNewDecFromString("0.02018"), alloraMath.MustNewDecFromString("0.01735"), alloraMath.MustNewDecFromString("0.01922"), alloraMath.MustNewDecFromString("0.02225"), alloraMath.MustNewDecFromString("0.0189"), alloraMath.MustNewDecFromString("0.01923"), alloraMath.MustNewDecFromString("0.03193"), alloraMath.MustNewDecFromString("0.01956"), alloraMath.MustNewDecFromString("0.01763"), alloraMath.MustNewDecFromString("0.01975"), alloraMath.MustNewDecFromString("0.01466"), alloraMath.MustNewDecFromString("0.02021"), alloraMath.MustNewDecFromString("0.01803"), alloraMath.MustNewDecFromString("0.01438"), alloraMath.MustNewDecFromString("0.01929"), alloraMath.MustNewDecFromString("0.02305"), alloraMath.MustNewDecFromString("0.02223"), alloraMath.MustNewDecFromString("0.02445"), alloraMath.MustNewDecFromString("0.01967"), alloraMath.MustNewDecFromString("0.02292"), alloraMath.MustNewDecFromString("0.01878"), alloraMath.MustNewDecFromString("0.01751"), alloraMath.MustNewDecFromString("0.02695"), alloraMath.MustNewDecFromString("0.01849"), alloraMath.MustNewDecFromString("0.01658"), alloraMath.MustNewDecFromString("0.01948"), alloraMath.MustNewDecFromString("0.01594"), alloraMath.MustNewDecFromString("0.02318"), alloraMath.MustNewDecFromString("0.01906"), alloraMath.MustNewDecFromString("0.01607"), alloraMath.MustNewDecFromString("0.01369"), alloraMath.MustNewDecFromString("0.01686"), alloraMath.MustNewDecFromString("0.01314"), alloraMath.MustNewDecFromString("0.01936"), alloraMath.MustNewDecFromString("0.01518"), alloraMath.MustNewDecFromString("0.018"), alloraMath.MustNewDecFromString("0.02212"), alloraMath.MustNewDecFromString("0.02259"), alloraMath.MustNewDecFromString("0.01674"), alloraMath.MustNewDecFromString("0.02944"), alloraMath.MustNewDecFromString("0.01796"), alloraMath.MustNewDecFromString("0.02187"), alloraMath.MustNewDecFromString("0.01895"), alloraMath.MustNewDecFromString("0.01637"), alloraMath.MustNewDecFromString("0.01594"), alloraMath.MustNewDecFromString("0.01608"), alloraMath.MustNewDecFromString("0.02203"), alloraMath.MustNewDecFromString("0.01486")},
		{alloraMath.MustNewDecFromString("0.02093"), alloraMath.MustNewDecFromString("0.00213"), alloraMath.MustNewDecFromString("0.02462"), alloraMath.MustNewDecFromString("0.0203"), alloraMath.MustNewDecFromString("0.03115"), alloraMath.MustNewDecFromString("0.01"), alloraMath.MustNewDecFromString("0.01545"), alloraMath.MustNewDecFromString("0.01785"), alloraMath.MustNewDecFromString("0.01662"), alloraMath.MustNewDecFromString("0.01156"), alloraMath.MustNewDecFromString("0.02284"), alloraMath.MustNewDecFromString("0.01475"), alloraMath.MustNewDecFromString("0.01331"), alloraMath.MustNewDecFromString("0.01592"), alloraMath.MustNewDecFromString("0.01462"), alloraMath.MustNewDecFromString("0.02333"), alloraMath.MustNewDecFromString("0.01836"), alloraMath.MustNewDecFromString("0.01465"), alloraMath.MustNewDecFromString("0.0186"), alloraMath.MustNewDecFromString("0.01566"), alloraMath.MustNewDecFromString("0.01506"), alloraMath.MustNewDecFromString("0.01678"), alloraMath.MustNewDecFromString("0.01423"), alloraMath.MustNewDecFromString("0.01658"), alloraMath.MustNewDecFromString("0.01741"), alloraMath.MustNewDecFromString("0.03491"), alloraMath.MustNewDecFromString("0.01408"), alloraMath.MustNewDecFromString("0.01191"), alloraMath.MustNewDecFromString("0.01572"), alloraMath.MustNewDecFromString("0.01355"), alloraMath.MustNewDecFromString("0.01477"), alloraMath.MustNewDecFromString("0.01662"), alloraMath.MustNewDecFromString("0.01128"), alloraMath.MustNewDecFromString("0.02581"), alloraMath.MustNewDecFromString("0.01718"), alloraMath.MustNewDecFromString("0.01705"), alloraMath.MustNewDecFromString("0.01251"), alloraMath.MustNewDecFromString("0.02158"), alloraMath.MustNewDecFromString("0.01187"), alloraMath.MustNewDecFromString("0.01504"), alloraMath.MustNewDecFromString("0.0135"), alloraMath.MustNewDecFromString("0.02432"), alloraMath.MustNewDecFromString("0.01602"), alloraMath.MustNewDecFromString("0.01194"), alloraMath.MustNewDecFromString("0.0153"), alloraMath.MustNewDecFromString("0.0199"), alloraMath.MustNewDecFromString("0.01673"), alloraMath.MustNewDecFromString("0.01049"), alloraMath.MustNewDecFromString("0.02068"), alloraMath.MustNewDecFromString("0.01573"), alloraMath.MustNewDecFromString("0.01487"), alloraMath.MustNewDecFromString("0.02639"), alloraMath.MustNewDecFromString("0.01981"), alloraMath.MustNewDecFromString("0.02123"), alloraMath.MustNewDecFromString("0.02134"), alloraMath.MustNewDecFromString("0.0217"), alloraMath.MustNewDecFromString("0.01177")},
	}
	stakes := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("1176644.37627"),
		alloraMath.MustNewDecFromString("384623.3607"),
		alloraMath.MustNewDecFromString("394676.13226"),
		alloraMath.MustNewDecFromString("207999.66194"),
		alloraMath.MustNewDecFromString("368582.76542"),
	}
	initialCoefficients := []alloraMath.Dec{
		alloraMath.OneDec(),
		alloraMath.OneDec(),
		alloraMath.OneDec(),
		alloraMath.OneDec(),
		alloraMath.OneDec(),
	}
	var numReputers int64 = 5
	wantScores := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("17.536755245164326"),
		alloraMath.MustNewDecFromString("20.302662649273707"),
		alloraMath.MustNewDecFromString("24.278413872561256"),
		alloraMath.MustNewDecFromString("11.365030585937692"),
		alloraMath.MustNewDecFromString("15.211816727558011"),
	}
	wantCoefficients := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("0.99942"),
		alloraMath.OneDec(),
		alloraMath.OneDec(),
		alloraMath.MustNewDecFromString("0.96574"),
		alloraMath.MustNewDecFromString("0.95346"),
	}
	gotScores, gotCoefficients, err := rewards.GetAllReputersOutput(allLosses, stakes, initialCoefficients, numReputers)
	require.NoError(t, err, "GetAllReputersOutput() error = %v, wantErr %v", err, false)

	if !alloraMath.SlicesInDelta(gotScores, wantScores, alloraMath.MustNewDecFromString("0.00001")) {
		t.Errorf("GetAllReputersOutput() gotScores = %v, want %v", gotScores, wantScores)
	}

	if !alloraMath.SlicesInDelta(gotCoefficients, wantCoefficients, alloraMath.MustNewDecFromString("0.0001")) {
		t.Errorf("GetAllReputersOutput() gotCoefficients = %v, want %v", gotCoefficients, wantCoefficients)
	}
}
*/
