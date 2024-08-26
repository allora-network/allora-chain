package actorutils_test

import (
	"log"
	"strconv"
	"testing"

	alloraMath "github.com/allora-network/allora-chain/math"
	actorutils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	"github.com/stretchr/testify/require"
)

func (s *ActorUtilsTestSuite) TestAdjustedStakeSimple() {
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

	result, err := actorutils.GetAdjustedStake(
		stake,
		allStakes,
		listeningCoefficient,
		allListeningCoefficients,
		numReputers,
	)
	s.Require().NoError(err)
	s.Require().True(alloraMath.InDelta(expected, result, alloraMath.MustNewDecFromString("0.0001")))
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
			got, err := actorutils.GetFinalWorkerScoreForecastTask(tt.scoreOneIn, tt.scoreOneOut, tt.fUniqueAgg)
			require.NoError(t, err)
			if !alloraMath.InDelta(tt.want, got, alloraMath.MustNewDecFromString("0.00001")) {
				t.Errorf("GetFinalWorkerScoreForecastTask() got = %v, want %v", got, tt.want)
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
				alloraMath.MustNewDecFromString("0.0152671"), alloraMath.MustNewDecFromString("0.002216"), alloraMath.MustNewDecFromString("0.02790"), alloraMath.MustNewDecFromString("0.017319"), alloraMath.MustNewDecFromString("0.025520"), alloraMath.MustNewDecFromString("0.0148812"), alloraMath.MustNewDecFromString("0.012625"), alloraMath.MustNewDecFromString("0.01780378"), alloraMath.MustNewDecFromString("0.0140014"),
				alloraMath.MustNewDecFromString("0.015013"), alloraMath.MustNewDecFromString("0.014774"), alloraMath.MustNewDecFromString("0.014550"), alloraMath.MustNewDecFromString("0.0144484"), alloraMath.MustNewDecFromString("0.0133076"), alloraMath.MustNewDecFromString("0.0137"), alloraMath.MustNewDecFromString("0.018843"), alloraMath.MustNewDecFromString("0.0158344"), alloraMath.MustNewDecFromString("0.014681"),
				alloraMath.MustNewDecFromString("0.015683"), alloraMath.MustNewDecFromString("0.012371"), alloraMath.MustNewDecFromString("0.014564"), alloraMath.MustNewDecFromString("0.0166473"), alloraMath.MustNewDecFromString("0.0145905"), alloraMath.MustNewDecFromString("0.01598"), alloraMath.MustNewDecFromString("0.01696"), alloraMath.MustNewDecFromString("0.01964"), alloraMath.MustNewDecFromString("0.0144"),
				alloraMath.MustNewDecFromString("0.0136411"), alloraMath.MustNewDecFromString("0.014375"), alloraMath.MustNewDecFromString("0.0145467"), alloraMath.MustNewDecFromString("0.01319248"), alloraMath.MustNewDecFromString("0.01555"), alloraMath.MustNewDecFromString("0.01246"), alloraMath.MustNewDecFromString("0.01849"), alloraMath.MustNewDecFromString("0.01386"), alloraMath.MustNewDecFromString("0.01430"),
				alloraMath.MustNewDecFromString("0.0148031"), alloraMath.MustNewDecFromString("0.016073"), alloraMath.MustNewDecFromString("0.01154604"), alloraMath.MustNewDecFromString("0.01281518"), alloraMath.MustNewDecFromString("0.01340968"), alloraMath.MustNewDecFromString("0.014733235"), alloraMath.MustNewDecFromString("0.01520795"), alloraMath.MustNewDecFromString("0.012093517"), alloraMath.MustNewDecFromString("0.0160167"),
				alloraMath.MustNewDecFromString("0.01547095"), alloraMath.MustNewDecFromString("0.01496103"), alloraMath.MustNewDecFromString("0.01296408"), alloraMath.MustNewDecFromString("0.0151219369"), alloraMath.MustNewDecFromString("0.014375538"), alloraMath.MustNewDecFromString("0.01548074"), alloraMath.MustNewDecFromString("0.017446629"), alloraMath.MustNewDecFromString("0.015452587"), alloraMath.MustNewDecFromString("0.01407107"),
				alloraMath.MustNewDecFromString("0.01700426"), alloraMath.MustNewDecFromString("0.014413132"), alloraMath.MustNewDecFromString("0.013480447"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := actorutils.GetStakeWeightedLossMatrix(tt.reputersAdjustedStakes, tt.reputersReportedLosses)
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
				alloraMath.MustNewDecFromString("1.5"), alloraMath.MustNewDecFromString("2.00000"), alloraMath.MustNewDecFromString("3.0"), alloraMath.MustNewDecFromString("2.999999"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := actorutils.GetStakeWeightedLossMatrix(tt.reputersAdjustedStakes, tt.reputersReportedLosses)
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
			want:                   alloraMath.MustNewDecFromString("0.0142047230098813"),
			wantErr:                false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := actorutils.GetStakeWeightedLoss(tt.reputersStakes, tt.reputersReportedLosses)
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
	reputerEpsilon := alloraMath.MustNewDecFromString("1e-2")
	epsilon := alloraMath.MustNewDecFromString("1e-4")
	want := []alloraMath.Dec{alloraMath.MustNewDecFromString("5.114259531"), alloraMath.MustNewDecFromString("5.339287075"), alloraMath.MustNewDecFromString("6.538380081"), alloraMath.MustNewDecFromString("2.5952235325"), alloraMath.MustNewDecFromString("3.5870524743")}
	wantErr := false

	got, err := actorutils.GetAllConsensusScores(allLosses, stakes, allListeningCoefficients, numReputers, reputerEpsilon, epsilon)
	if (err != nil) != wantErr {
		t.Errorf("GetAllConsensusScores() error = %v, wantErr %v", err, wantErr)
		return
	}

	if !alloraMath.SlicesInDelta(got, want, alloraMath.MustNewDecFromString("0.01")) {
		t.Errorf("GetAllConsensusScores() got = %v, want %v", got, want)
	}
}

func (s *ActorUtilsTestSuite) TestGetAllReputersOutput() {
	require := s.Require()

	params, err := s.emissionsKeeper.GetParams(s.ctx)
	require.NoError(err)

	epsilon := alloraMath.MustNewDecFromString("0.01")

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
		alloraMath.MustNewDecFromString("0.016983"),
		alloraMath.MustNewDecFromString("0.017068"),
		alloraMath.MustNewDecFromString("0.016047"),
		alloraMath.MustNewDecFromString("0.011649"),
		alloraMath.MustNewDecFromString("0.013453"),
	}
	gotScores0, gotCoefficients0, err := actorutils.GetAllReputersOutput(
		allLosses,
		stakes,
		initialCoefficients,
		numReputers,
		params.LearningRate,
		0,
		params.EpsilonReputer,
		epsilon,
		params.MinStakeFraction,
		params.MaxGradientThreshold,
	)
	require.NoError(err)

	gotScores1, gotCoefficients1, err := actorutils.GetAllReputersOutput(
		allLosses,
		stakes,
		initialCoefficients,
		numReputers,
		params.LearningRate,
		2,
		params.EpsilonReputer,
		epsilon,
		params.MinStakeFraction,
		params.MaxGradientThreshold,
	)
	require.NoError(err)

	gotScores2, gotCoefficients2, err := actorutils.GetAllReputersOutput(
		allLosses,
		stakes,
		initialCoefficients,
		numReputers,
		params.LearningRate,
		5,
		params.EpsilonReputer,
		epsilon,
		params.MinStakeFraction,
		params.MaxGradientThreshold,
	)
	require.NoError(err)

	gotScores3, gotCoefficients3, err := actorutils.GetAllReputersOutput(
		allLosses,
		stakes,
		initialCoefficients,
		numReputers,
		params.LearningRate,
		20,
		params.EpsilonReputer,
		epsilon,
		params.MinStakeFraction,
		params.MaxGradientThreshold,
	)
	require.NoError(err)

	// Assumes that the inputs are of the same length
	getAdjustedStakes := func(coefficients []alloraMath.Dec) ([]alloraMath.Dec, error) {
		N_r := alloraMath.NewDecFromInt64(int64(len(stakes)))
		adjustedStakes := make([]alloraMath.Dec, len(stakes))
		adjustedStakeNumerators := make([]alloraMath.Dec, len(stakes))
		sumAdjustedStakes := alloraMath.ZeroDec()
		for i, stake := range stakes {
			adjustedStake, err := stake.Mul(coefficients[i])
			require.NoError(err)
			adjustedStake, err = adjustedStake.Mul(N_r)
			require.NoError(err)
			adjustedStakeNumerators[i] = adjustedStake
			sumAdjustedStakes, err = sumAdjustedStakes.Add(adjustedStake)
			require.NoError(err)
		}
		for i, adjustedStakeNumerator := range adjustedStakeNumerators {
			adjustedStake, err := adjustedStakeNumerator.Quo(sumAdjustedStakes)
			require.NoError(err)
			adjustedStakes[i] = alloraMath.Min(alloraMath.OneDec(), adjustedStake)
		}
		return adjustedStakes, nil
	}

	// Assumes that the inputs are the same length as the `stakes` array
	getTotalConsensusScore := func(scores []alloraMath.Dec, coefficients []alloraMath.Dec) (float64, error) {
		adjustedStakes, err := getAdjustedStakes(coefficients)
		require.NoError(err)
		require.Len(adjustedStakes, len(stakes))
		totalScore := alloraMath.ZeroDec()
		sumStake := alloraMath.ZeroDec()
		for i, score := range scores {
			stakeTimesScore, err := score.Mul(adjustedStakes[i])
			require.NoError(err)
			totalScore, err = totalScore.Add(stakeTimesScore)
			require.NoError(err)
			sumStake, err = sumStake.Add(adjustedStakes[i])
			require.NoError(err)
		}
		totalScore, err = totalScore.Quo(sumStake)
		require.NoError(err)
		output, err := strconv.ParseFloat(totalScore.String(), 64)
		require.NoError(err)
		return output, nil
	}

	startCoefficients := []alloraMath.Dec{
		alloraMath.OneDec(),
		alloraMath.OneDec(),
		alloraMath.OneDec(),
		alloraMath.OneDec(),
		alloraMath.OneDec(),
	}

	require.True(len(gotCoefficients0) == len(stakes))
	require.True(len(gotCoefficients1) == len(stakes))
	require.True(len(gotCoefficients2) == len(stakes))
	require.True(len(gotCoefficients3) == len(stakes))

	// Check that the total consensus score improves with successive invocations of the function with more iterations
	totalScore0, _ := getTotalConsensusScore(gotScores0, startCoefficients)
	totalScore1, _ := getTotalConsensusScore(gotScores1, gotCoefficients1)
	totalScore2, _ := getTotalConsensusScore(gotScores2, gotCoefficients2)
	totalScore3, _ := getTotalConsensusScore(gotScores3, gotCoefficients3)

	require.LessOrEqual(totalScore0, totalScore1)
	require.LessOrEqual(totalScore1, totalScore2)
	require.LessOrEqual(totalScore2, totalScore3)

	// Some simple checks of the scores
	require.True(len(gotScores1) == len(wantScores))
	require.True(len(gotScores2) == len(wantScores))
	require.True(len(gotScores3) == len(wantScores))

	// Verify score output matches that of GetAllConsensusScores()
	wantScores3, err := actorutils.GetAllConsensusScores(allLosses, stakes, gotCoefficients3, numReputers, params.EpsilonReputer, epsilon)
	require.NoError(err)
	if !alloraMath.SlicesInDelta(gotScores3, wantScores3, alloraMath.MustNewDecFromString("0.01")) {
		log.Println("GetAllConsensusScores() got", gotScores3, "want", wantScores3)
	}
}
