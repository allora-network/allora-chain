package inference_synthesis_test

import (
	"encoding/csv"
	"reflect"
	"strings"
	"testing"

	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/stretchr/testify/assert"

	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// instantiate a AllWorkersAreNew struct
func NewWorkersAreNew(v bool) inferencesynthesis.AllWorkersAreNew {
	return inferencesynthesis.AllWorkersAreNew{
		AllInferersAreNew:    v,
		AllForecastersAreNew: v,
	}
}

// TestMakeMapFromWorkerToTheirWork tests the makeMapFromWorkerToTheirWork function for correctly mapping workers to their inferences.
func TestMakeMapFromWorkerToTheirWork(t *testing.T) {
	tests := []struct {
		name       string
		inferences []*emissionstypes.Inference
		expected   map[string]*emissionstypes.Inference
	}{
		{
			name: "multiple workers",
			inferences: []*emissionstypes.Inference{
				{
					TopicId: 101,
					Inferer: "worker1",
					Value:   alloraMath.MustNewDecFromString("10"),
				},
				{
					TopicId: 102,
					Inferer: "worker2",
					Value:   alloraMath.MustNewDecFromString("20"),
				},
				{
					TopicId: 103,
					Inferer: "worker3",
					Value:   alloraMath.MustNewDecFromString("30"),
				},
			},
			expected: map[string]*emissionstypes.Inference{
				"worker1": {
					TopicId: 101,
					Inferer: "worker1",
					Value:   alloraMath.MustNewDecFromString("10"),
				},
				"worker2": {
					TopicId: 102,
					Inferer: "worker2",
					Value:   alloraMath.MustNewDecFromString("20"),
				},
				"worker3": {
					TopicId: 103,
					Inferer: "worker3",
					Value:   alloraMath.MustNewDecFromString("30"),
				},
			},
		},
		{
			name:       "empty list",
			inferences: []*emissionstypes.Inference{},
			expected:   map[string]*emissionstypes.Inference{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := inferencesynthesis.MakeMapFromWorkerToTheirWork(tc.inferences)
			assert.True(t, reflect.DeepEqual(result, tc.expected), "Expected and actual maps should be equal")
		})
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcWeightedInferenceNormalOperation1() {
	topicId := inferencesynthesis.TopicId(1)

	// EPOCH 2
	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0": {Value: alloraMath.MustNewDecFromString("-0.230622933739544")},
		"worker1": {Value: alloraMath.MustNewDecFromString("-0.19693894066605602")},
		"worker2": {Value: alloraMath.MustNewDecFromString("0.048704500498029504")},
		"worker3": {Value: alloraMath.MustNewDecFromString("0.054145121711977245")},
		"worker4": {Value: alloraMath.MustNewDecFromString("0.22919548623217473")},
	}
	forecastImpliedInferenceByWorker := map[string]*emissionstypes.Inference{
		"worker5": {Value: alloraMath.MustNewDecFromString("0.05403102080389692")},
	}
	epsilon := alloraMath.MustNewDecFromString("0.0001")
	pNorm := alloraMath.MustNewDecFromString("3")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	infererNetworkRegrets := map[string]inferencesynthesis.Regret{ // FROM EPOCH 1
		"worker0": alloraMath.MustNewDecFromString("0.6244348454149327"),
		"worker1": alloraMath.MustNewDecFromString("0.692277772835371"),
		"worker2": alloraMath.MustNewDecFromString("0.5643643585741522"),
		"worker3": alloraMath.MustNewDecFromString("0.7350459785583141"),
		"worker4": alloraMath.MustNewDecFromString("0.6152990250605652"),
	}
	forecasterNetworkRegrets := map[string]inferencesynthesis.Regret{ // FROM EPOCH 1
		"worker5": alloraMath.MustNewDecFromString("0.9021613542996348"),
	}
	expectedNetworkCombinedInferenceValue := alloraMath.MustNewDecFromString("-0.006914290859920283")

	for inferer, regret := range infererNetworkRegrets {
		s.emissionsKeeper.SetInfererNetworkRegret(
			s.ctx,
			topicId,
			inferer,
			emissionstypes.TimestampedValue{BlockHeight: 0, Value: regret},
		)
	}

	for forecaster, regret := range forecasterNetworkRegrets {
		s.emissionsKeeper.SetForecasterNetworkRegret(
			s.ctx,
			topicId,
			forecaster,
			emissionstypes.TimestampedValue{BlockHeight: 0, Value: regret},
		)
	}

	sortedInferers := alloraMath.GetSortedKeys(inferenceByWorker)

	infererNormalizedRegrets, err := inferencesynthesis.GetInfererNormalizedRegretsWithMax(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		sortedInferers,
		epsilon,
	)
	s.Require().NoError(err)

	sortedForecasters := alloraMath.GetSortedKeys(forecastImpliedInferenceByWorker)

	// Get Forecaster normalized regrets and max regret
	forecastNormalizedRegrets, err := inferencesynthesis.GetForecasterNormalizedRegretsWithMax(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		sortedForecasters,
		epsilon,
	)
	s.Require().NoError(err)

	networkCombinedInferenceValue, err := inferencesynthesis.CalcWeightedInference(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		inferenceByWorker,
		sortedInferers,
		forecastImpliedInferenceByWorker,
		sortedForecasters,
		infererNormalizedRegrets,
		forecastNormalizedRegrets,
		NewWorkersAreNew(false),
		epsilon,
		pNorm,
		cNorm,
	)

	s.Require().NoError(err)

	s.Require().True(
		alloraMath.InDelta(
			expectedNetworkCombinedInferenceValue,
			networkCombinedInferenceValue,
			alloraMath.MustNewDecFromString("0.00001"),
		),
		"Network combined inference value should match expected value within epsilon",
		expectedNetworkCombinedInferenceValue.String(),
		networkCombinedInferenceValue.String(),
	)
}

func (s *InferenceSynthesisTestSuite) TestCalcWeightedInferenceNormalOperation2() {
	topicId := inferencesynthesis.TopicId(1)

	// EPOCH 3
	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0": {Value: alloraMath.MustNewDecFromString("-0.035995138925040600")},
		"worker1": {Value: alloraMath.MustNewDecFromString("-0.07333303938740420")},
		"worker2": {Value: alloraMath.MustNewDecFromString("-0.1495482917094790")},
		"worker3": {Value: alloraMath.MustNewDecFromString("-0.12952123274063800")},
		"worker4": {Value: alloraMath.MustNewDecFromString("-0.0703055329498285")},
	}
	forecastImpliedInferenceByWorker := map[string]*emissionstypes.Inference{
		"worker5": {Value: alloraMath.MustNewDecFromString("-0.1025675327315210")},
	}
	epsilon := alloraMath.MustNewDecFromString("0.0001")
	pNorm := alloraMath.MustNewDecFromString("3")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	infererNetworkRegrets := map[string]inferencesynthesis.Regret{ // FROM EPOCH 2
		"worker0": alloraMath.MustNewDecFromString("0.29086859544474400"),
		"worker1": alloraMath.MustNewDecFromString("0.4166836024936930"),
		"worker2": alloraMath.MustNewDecFromString("0.17509650556843800"),
		"worker3": alloraMath.MustNewDecFromString("0.49463613630101200"),
		"worker4": alloraMath.MustNewDecFromString("0.27842209239585900"),
	}
	forecasterNetworkRegrets := map[string]inferencesynthesis.Regret{ // FROM EPOCH 2
		"worker5": alloraMath.MustNewDecFromString("0.8145278779981100"),
	}
	expectedNetworkCombinedInferenceValue := alloraMath.MustNewDecFromString("-0.09101522746174310")

	for inferer, regret := range infererNetworkRegrets {
		s.emissionsKeeper.SetInfererNetworkRegret(
			s.ctx,
			topicId,
			inferer,
			emissionstypes.TimestampedValue{BlockHeight: 0, Value: regret},
		)
	}

	for forecaster, regret := range forecasterNetworkRegrets {
		s.emissionsKeeper.SetForecasterNetworkRegret(
			s.ctx,
			topicId,
			forecaster,
			emissionstypes.TimestampedValue{BlockHeight: 0, Value: regret},
		)
	}

	sortedInferers := alloraMath.GetSortedKeys(inferenceByWorker)

	infererNormalizedRegrets, err := inferencesynthesis.GetInfererNormalizedRegretsWithMax(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		sortedInferers,
		epsilon,
	)
	s.Require().NoError(err)

	sortedForecasters := alloraMath.GetSortedKeys(forecastImpliedInferenceByWorker)

	// Get Forecaster normalized regrets and max regret
	forecastNormalizedRegrets, err := inferencesynthesis.GetForecasterNormalizedRegretsWithMax(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		sortedForecasters,
		epsilon,
	)
	s.Require().NoError(err)

	networkCombinedInferenceValue, err := inferencesynthesis.CalcWeightedInference(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		inferenceByWorker,
		sortedInferers,
		forecastImpliedInferenceByWorker,
		sortedForecasters,
		infererNormalizedRegrets,
		forecastNormalizedRegrets,
		NewWorkersAreNew(false),
		epsilon,
		pNorm,
		cNorm,
	)
	s.Require().NoError(err)

	s.Require().True(
		alloraMath.InDelta(
			expectedNetworkCombinedInferenceValue,
			networkCombinedInferenceValue,
			alloraMath.MustNewDecFromString("0.01"),
		),
		"Network combined inference value should match expected value within epsilon",
		expectedNetworkCombinedInferenceValue.String(),
		networkCombinedInferenceValue.String(),
	)
}

func csvToSimulatedValuesMap(headers string, values string) map[string]string {
	r := csv.NewReader(strings.NewReader(headers + "\n" + values + "\n"))
	headersRead, err := r.Read()
	if err != nil {
		panic(err)
	}
	valuesRead, err := r.Read()
	if err != nil {
		panic(err)
	}
	simulatedValues := make(map[string]string)
	if len(headersRead) != len(valuesRead) {
		panic("Header and values length mismatch")
	}
	for i := 0; i < len(headersRead); i++ {
		simulatedValues[headersRead[i]] = valuesRead[i]
	}
	return simulatedValues
}

func (s *InferenceSynthesisTestSuite) TestCalcOneOutInferencesMultipleWorkers() {
	headers := "epoch,time,returns,inference_0,inference_1,inference_2,inference_3,inference_4,forecasted_loss_0_for_0,forecasted_loss_0_for_1,forecasted_loss_0_for_2,forecasted_loss_0_for_3,forecasted_loss_0_for_4,forecasted_loss_1_for_0,forecasted_loss_1_for_1,forecasted_loss_1_for_2,forecasted_loss_1_for_3,forecasted_loss_1_for_4,forecasted_loss_2_for_0,forecasted_loss_2_for_1,forecasted_loss_2_for_2,forecasted_loss_2_for_3,forecasted_loss_2_for_4,forecast_implied_inference_0,forecast_implied_inference_1,forecast_implied_inference_2,forecast_implied_inference_0_oneout_0,forecast_implied_inference_0_oneout_1,forecast_implied_inference_0_oneout_2,forecast_implied_inference_0_oneout_3,forecast_implied_inference_0_oneout_4,forecast_implied_inference_1_oneout_0,forecast_implied_inference_1_oneout_1,forecast_implied_inference_1_oneout_2,forecast_implied_inference_1_oneout_3,forecast_implied_inference_1_oneout_4,forecast_implied_inference_2_oneout_0,forecast_implied_inference_2_oneout_1,forecast_implied_inference_2_oneout_2,forecast_implied_inference_2_oneout_3,forecast_implied_inference_2_oneout_4,network_inference,network_naive_inference,network_inference_oneout_0,network_inference_oneout_1,network_inference_oneout_2,network_inference_oneout_3,network_inference_oneout_4,network_inference_oneout_5,network_inference_oneout_6,network_inference_oneout_7,network_naive_inference_onein_0,network_naive_inference_onein_1,network_naive_inference_onein_2,network_loss,reputer_stake_0,reputer_stake_1,reputer_stake_2,reputer_stake_3,reputer_stake_4,reputer_0_loss_inference_0,reputer_0_loss_inference_1,reputer_0_loss_inference_2,reputer_0_loss_inference_3,reputer_0_loss_inference_4,reputer_1_loss_inference_0,reputer_1_loss_inference_1,reputer_1_loss_inference_2,reputer_1_loss_inference_3,reputer_1_loss_inference_4,reputer_2_loss_inference_0,reputer_2_loss_inference_1,reputer_2_loss_inference_2,reputer_2_loss_inference_3,reputer_2_loss_inference_4,reputer_3_loss_inference_0,reputer_3_loss_inference_1,reputer_3_loss_inference_2,reputer_3_loss_inference_3,reputer_3_loss_inference_4,reputer_4_loss_inference_0,reputer_4_loss_inference_1,reputer_4_loss_inference_2,reputer_4_loss_inference_3,reputer_4_loss_inference_4,reputer_0_loss_forecast_implied_inference_0,reputer_0_loss_forecast_implied_inference_1,reputer_0_loss_forecast_implied_inference_2,reputer_1_loss_forecast_implied_inference_0,reputer_1_loss_forecast_implied_inference_1,reputer_1_loss_forecast_implied_inference_2,reputer_2_loss_forecast_implied_inference_0,reputer_2_loss_forecast_implied_inference_1,reputer_2_loss_forecast_implied_inference_2,reputer_3_loss_forecast_implied_inference_0,reputer_3_loss_forecast_implied_inference_1,reputer_3_loss_forecast_implied_inference_2,reputer_4_loss_forecast_implied_inference_0,reputer_4_loss_forecast_implied_inference_1,reputer_4_loss_forecast_implied_inference_2,inference_loss_0,inference_loss_1,inference_loss_2,inference_loss_3,inference_loss_4,forecast_implied_inference_loss_0,forecast_implied_inference_loss_1,forecast_implied_inference_loss_2,inference_regret_worker_0,inference_regret_worker_1,inference_regret_worker_2,inference_regret_worker_3,inference_regret_worker_4,inference_regret_worker_5,inference_regret_worker_6,inference_regret_worker_7,inference_regret_worker_0_onein_0,inference_regret_worker_1_onein_0,inference_regret_worker_2_onein_0,inference_regret_worker_3_onein_0,inference_regret_worker_4_onein_0,inference_regret_worker_5_onein_0,inference_regret_worker_0_onein_1,inference_regret_worker_1_onein_1,inference_regret_worker_2_onein_1,inference_regret_worker_3_onein_1,inference_regret_worker_4_onein_1,inference_regret_worker_5_onein_1,inference_regret_worker_0_onein_2,inference_regret_worker_1_onein_2,inference_regret_worker_2_onein_2,inference_regret_worker_3_onein_2,inference_regret_worker_4_onein_2,inference_regret_worker_5_onein_2,reputer_0_loss_network_inference,reputer_1_loss_network_inference,reputer_2_loss_network_inference,reputer_3_loss_network_inference,reputer_4_loss_network_inference,network_loss_reputers"
	values29 := "29,2020-12-30 00:00:00,0.06341995255869896,-0.06262282658758557,-0.08500102820757845,0.1083013406965205,0.13289465377115978,-0.0020728060388181396,-2.422236320166231,-0.9550590853619685,-2.6561235232710487,-3.992824396454848,-1.3539124887770135,-2.0431234533437306,-1.3049462512050782,-2.516728365710576,-2.2831238837514998,-2.4161858720575324,-1.7438228344282494,-1.237972718945092,-3.5749814295705225,-2.2295539861627254,-2.3928531254578873,0.12975349726041235,0.07529420218026293,0.10702750866768636,0.13174378501001557,0.13115344212350366,0.12895451648780984,0.06110891885060462,0.1297272967707113,0.0780031183660436,0.09067649885451183,0.03141943526549712,0.06519658466273871,0.1062000193152707,0.10686999536993497,0.10770465469690327,0.03152656148467134,0.10593349994233268,0.1081544730767232,0.10438540944589664,0.1276687789799889,0.10343577578111743,0.11151407656645225,0.05550377434717742,0.08026710963540105,0.11320915913040085,0.09142901134172764,0.11809058790109367,0.10325645526728987,0.12936327609140083,0.07541026807871372,0.10685190746623384,-4.648411655468548,225229.16522285485,228783.23029247465,168405.32077728235,418743.21152128006,242954.12805949515,-2.379091982254331,-2.6986244617833206,-1.9745779715150662,-3.151866665890307,-2.349999063676661,-2.338327816174232,-2.870994527824424,-2.0493458053693727,-3.3041969177528756,-2.1882495089861997,-2.6347834355050574,-2.494572740722586,-2.7287438173119583,-3.0312441801041126,-2.5286095438322036,-2.4990491033212052,-2.474511540253177,-2.4359496261654905,-3.015858026672051,-2.3496357576379374,-2.517049325503982,-2.6537993764967283,-2.3451088424879343,-3.0976847404413146,-2.419806332131773,-4.599757416923247,-4.722922865140425,-4.600459506813928,-4.738253353905239,-4.681297889651852,-4.6254898463174365,-4.088712724774769,-4.097757960454803,-4.174720726841145,-4.197887686208278,-4.172352853345479,-4.233839651700592,-4.4406432700049905,-4.438788441669191,-4.4532418186314215,-2.4705808089216754,-2.621011308206566,-2.307359227234206,-3.10858450211525,-2.35769401409519,-4.396259485214029,-4.4002233712107115,-4.401679130149749,-2.113359620837304,-1.8444197517192997,-2.3038543625669226,-1.4089558166932068,-2.2284865306421415,-0.0829717665704939,-0.10429837601342462,-0.08870102599961535,-1.9426733748851683,-1.6737335057671638,-2.1331681166147867,-1.2382695707410705,-2.057800284690005,0.08771447938164224,-1.9578219954396046,-1.6888821263215998,-2.148316737169223,-1.2534181912955071,-2.072948905244442,0.05123924938427502,-2.0274393228316687,-1.7584994537136645,-2.2179340645612875,-1.3230355186875713,-2.1425662326365056,-0.002780727993979862,-4.755363807454225,-4.8338102065662145,-4.502250715646263,-4.5408188288754765,-4.661431489307927,-4.648411655468548"
	values30 := "30,2020-12-31 00:00:00,-0.048661295981826246,-0.024383006893380528,0.04399633190425988,0.12696315824633808,-0.06222734684262579,-0.18544136371380426,-2.6492503602777937,-0.5743936470328908,-3.9064625801146042,-3.618414606693211,-2.5248666099131825,-2.2887881178533793,-1.783946273333383,-1.706371355285912,-3.8755284922835096,-2.0748194138264906,-2.9152626306451186,-1.674120828193758,-0.8357453545873608,-4.253336822227785,-1.4104783184173797,0.057942589626760244,-0.06217410949633049,-0.060647852586817576,0.055004591992690643,0.08967790690951748,-0.06550940521226968,0.11139206935025397,0.05880193202802593,-0.06229496160012768,-0.062213326593970025,-0.06223262890694136,-0.03480628188598846,-0.0618476113702539,-0.0620291321687006,-0.06046552604373999,-0.061057082073298005,-0.024253600520809285,-0.060165142996311746,-0.02074499588729473,-0.06006827066436784,-0.025559213291971458,-0.0028945451097527612,-0.06239408906368381,0.01042958639353237,-0.028578886859456525,-0.061193717439573345,-0.0012304119204419976,-0.0005535129944102879,0.05701819115919576,-0.062018476973073826,-0.06051063298493267,-4.580734991779338,225570.5681415406,229103.55218038158,168602.58035442166,419465.5443936611,244644.5672539627,-2.3710787653755685,-2.684750022648258,-1.9852985212678498,-3.1418125479078314,-2.351992812869071,-2.3344641301319826,-2.855730537188912,-2.061927488878044,-3.2904195339675595,-2.193702831619839,-2.5723239747504327,-2.4277222508417675,-2.7323362861550384,-2.9687082394702733,-2.5193021794883705,-2.4698159005560774,-2.440874398769401,-2.4418347532769347,-2.9732991498142955,-2.345096423577895,-2.498248978595617,-2.619734004871709,-2.3616127961144415,-3.0787434988685156,-2.42323381270577,-4.570615467734497,-4.711715940119585,-4.577882071601022,-4.70018171996458,-4.670486258131893,-4.5953423056176215,-3.9365817187758263,-4.105050381924702,-4.0568427343757625,-4.107989392629676,-4.155658543009392,-4.163814355156637,-4.3807099828507345,-4.418919805816289,-4.40247281125215,-2.4472565276385003,-2.5896996943310495,-2.317034811704988,-3.078696700838115,-2.357026202815353,-4.323812659092131,-4.388107570438046,-4.344503544731199,-2.1153715051676576,-1.8590813062921987,-2.2998389443176652,-1.4182640641180084,-2.228008756474326,-0.10036682318216522,-0.11313128054621141,-0.10345406810446772,-1.9296310911991135,-1.6733408923236546,-2.1140985303491213,-1.2325236501494639,-2.0422683425057815,0.0853735907863792,-1.9559915392295468,-1.6997013403540875,-2.1404589783795545,-1.2588840981798974,-2.068628790536215,0.046248685391899524,-2.0181773033801322,-1.7618871045046736,-2.2026447425301403,-1.321069862330483,-2.1308145546868,-0.006259866316942325,-4.72849229621568,-4.805343770098079,-4.367063631685487,-4.451868846475796,-4.602366710154445,-4.580734991779338"
	simulatedValues29 := csvToSimulatedValuesMap(headers, values29)
	simulatedValues30 := csvToSimulatedValuesMap(headers, values30)

	topicId := inferencesynthesis.TopicId(1)
	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0": {Value: alloraMath.MustNewDecFromString(simulatedValues30["inference_0"])},
		"worker1": {Value: alloraMath.MustNewDecFromString(simulatedValues30["inference_1"])},
		"worker2": {Value: alloraMath.MustNewDecFromString(simulatedValues30["inference_2"])},
		"worker3": {Value: alloraMath.MustNewDecFromString(simulatedValues30["inference_3"])},
		"worker4": {Value: alloraMath.MustNewDecFromString(simulatedValues30["inference_4"])},
	}
	forecastImpliedInferenceByWorker := map[string]*emissionstypes.Inference{
		"worker5": {Value: alloraMath.MustNewDecFromString(simulatedValues29["forecast_implied_inference_0"])},
	}
	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: "worker0",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker2", Value: alloraMath.MustNewDecFromString(simulatedValues29["forecasted_loss_0_for_2"])},
					{Inferer: "worker3", Value: alloraMath.MustNewDecFromString(simulatedValues29["forecasted_loss_0_for_3"])},
					{Inferer: "worker4", Value: alloraMath.MustNewDecFromString(simulatedValues29["forecasted_loss_0_for_4"])},
				},
			},
			{
				Forecaster: "worker1",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker2", Value: alloraMath.MustNewDecFromString(simulatedValues29["forecasted_loss_1_for_2"])},
					{Inferer: "worker3", Value: alloraMath.MustNewDecFromString(simulatedValues29["forecasted_loss_1_for_2"])},
					{Inferer: "worker4", Value: alloraMath.MustNewDecFromString(simulatedValues29["forecasted_loss_1_for_2"])},
				},
			},
			{
				Forecaster: "worker2",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker2", Value: alloraMath.MustNewDecFromString(simulatedValues29["forecasted_loss_2_for_2"])},
					{Inferer: "worker3", Value: alloraMath.MustNewDecFromString(simulatedValues29["forecasted_loss_2_for_3"])},
					{Inferer: "worker4", Value: alloraMath.MustNewDecFromString(simulatedValues29["forecasted_loss_2_for_4"])},
				},
			},
		},
	}
	infererNetworkRegrets := map[string]inferencesynthesis.Regret{
		"worker0": alloraMath.MustNewDecFromString(simulatedValues29["inference_regret_worker_0_onein_0"]),
		"worker1": alloraMath.MustNewDecFromString(simulatedValues29["inference_regret_worker_1_onein_0"]),
		"worker2": alloraMath.MustNewDecFromString(simulatedValues29["inference_regret_worker_2_onein_0"]),
		"worker3": alloraMath.MustNewDecFromString(simulatedValues29["inference_regret_worker_3_onein_0"]),
		"worker4": alloraMath.MustNewDecFromString(simulatedValues29["inference_regret_worker_4_onein_0"]),
	}
	forecasterNetworkRegrets := map[string]inferencesynthesis.Regret{
		"worker5": alloraMath.MustNewDecFromString(simulatedValues29["inference_regret_worker_5"]),
		"worker6": alloraMath.MustNewDecFromString(simulatedValues29["inference_regret_worker_6"]),
		"worker7": alloraMath.MustNewDecFromString(simulatedValues29["inference_regret_worker_7"]),
	}
	networkCombinedLoss := alloraMath.MustNewDecFromString(simulatedValues29["network_loss"])
	epsilon := alloraMath.MustNewDecFromString("0.0001")
	fTolerance := alloraMath.MustNewDecFromString("0.01")
	pNorm := alloraMath.MustNewDecFromString("3")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	expectedOneOutInferences := []struct {
		Worker string
		Value  string
	}{
		{Worker: "worker0", Value: simulatedValues30["network_inference_oneout_0"]},
		{Worker: "worker1", Value: simulatedValues30["network_inference_oneout_1"]},
		{Worker: "worker2", Value: simulatedValues30["network_inference_oneout_2"]},
		{Worker: "worker3", Value: simulatedValues30["network_inference_oneout_3"]},
		{Worker: "worker4", Value: simulatedValues30["network_inference_oneout_4"]},
		{Worker: "worker5", Value: simulatedValues30["network_inference_oneout_5"]},
		{Worker: "worker6", Value: simulatedValues30["network_inference_oneout_6"]},
		{Worker: "worker7", Value: simulatedValues30["network_inference_oneout_7"]},
	}
	expectedOneOutImpliedInferences := []struct {
		Worker string
		Value  string
	}{
		{Worker: "worker0", Value: simulatedValues30["forecast_implied_inference_0_oneout_0"]},
		{Worker: "worker1", Value: simulatedValues30["forecast_implied_inference_1_oneout_1"]},
		{Worker: "worker2", Value: simulatedValues30["forecast_implied_inference_2_oneout_2"]},
	}

	for inferer, regret := range infererNetworkRegrets {
		s.emissionsKeeper.SetInfererNetworkRegret(
			s.ctx,
			topicId,
			inferer,
			emissionstypes.TimestampedValue{BlockHeight: 0, Value: regret},
		)
	}

	for forecaster, regret := range forecasterNetworkRegrets {
		s.emissionsKeeper.SetForecasterNetworkRegret(
			s.ctx,
			topicId,
			forecaster,
			emissionstypes.TimestampedValue{BlockHeight: 0, Value: regret},
		)
	}

	sortedInferers := alloraMath.GetSortedKeys(inferenceByWorker)

	infererNormalizedRegrets, err := inferencesynthesis.GetInfererNormalizedRegretsWithMax(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		sortedInferers,
		epsilon,
	)
	s.Require().NoError(err)

	sortedForecasters := alloraMath.GetSortedKeys(forecastImpliedInferenceByWorker)

	// Get Forecaster normalized regrets and max regret
	forecastNormalizedRegrets, err := inferencesynthesis.GetForecasterNormalizedRegretsWithMax(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		sortedForecasters,
		epsilon,
	)
	s.Require().NoError(err)

	oneOutInfererValues, oneOutForecasterValues, err := inferencesynthesis.CalcOneOutInferences(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		inferenceByWorker,
		sortedInferers,
		forecastImpliedInferenceByWorker,
		sortedForecasters,
		forecasts,
		NewWorkersAreNew(false),
		infererNormalizedRegrets,
		forecastNormalizedRegrets,
		networkCombinedLoss,
		epsilon,
		fTolerance,
		pNorm,
		cNorm,
	)

	s.Require().NoError(err, "CalcOneOutInferences should not return an error")

	s.Require().Len(oneOutInfererValues, len(expectedOneOutInferences), "Unexpected number of one-out inferences")
	s.Require().Len(oneOutForecasterValues, len(expectedOneOutImpliedInferences), "Unexpected number of one-out implied inferences")

	for _, expected := range expectedOneOutInferences {
		found := false
		for _, oneOutInference := range oneOutInfererValues {
			if expected.Worker == oneOutInference.Worker {
				found = true
				s.inEpsilon2(oneOutInference.Value, expected.Value)
			}
		}
		if !found {
			s.FailNow("Matching worker not found", "Worker %s not found in returned inferences", expected.Worker)
		}
	}

	for _, expected := range expectedOneOutImpliedInferences {
		found := false
		for _, oneOutImpliedInference := range oneOutForecasterValues {
			if expected.Worker == oneOutImpliedInference.Worker {
				found = true
				s.inEpsilon3(oneOutImpliedInference.Value, expected.Value)
			}
		}
		if !found {
			s.FailNow("Matching worker not found", "Worker %s not found in returned inferences", expected.Worker)
		}
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcOneOutInferences5Workers3Forecasters() {
	require := s.Require()
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := inferencesynthesis.TopicId(1)
	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0": {Value: alloraMath.MustNewDecFromString("-0.035995138925040600")},
		"worker1": {Value: alloraMath.MustNewDecFromString("-0.07333303938740420")},
		"worker2": {Value: alloraMath.MustNewDecFromString("-0.1495482917094790")},
		"worker3": {Value: alloraMath.MustNewDecFromString("-0.12952123274063800")},
		"worker4": {Value: alloraMath.MustNewDecFromString("-0.0703055329498285")},
	}
	forecastImpliedInferenceByWorker := map[string]*emissionstypes.Inference{
		"forecaster0": {Value: alloraMath.MustNewDecFromString("-0.08944493117005920")},
		"forecaster1": {Value: alloraMath.MustNewDecFromString("-0.07333218290300560")},
		"forecaster2": {Value: alloraMath.MustNewDecFromString("-0.07756206109376570")},
	}
	// epoch 3
	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: "forecaster0",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("0.003305466418410120")},
					{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.0002788248228566030")},
					{Inferer: "worker2", Value: alloraMath.MustNewDecFromString(".0000240536828602367")},
					{Inferer: "worker3", Value: alloraMath.MustNewDecFromString("0.0008240378476798250")},
					{Inferer: "worker4", Value: alloraMath.MustNewDecFromString("0.0000186192181193532")},
				},
			},
			{
				Forecaster: "forecaster1",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("0.002308441286328890")},
					{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.0000214380788596749")},
					{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("0.012560171044167200")},
					{Inferer: "worker3", Value: alloraMath.MustNewDecFromString("0.017998563880697900")},
					{Inferer: "worker4", Value: alloraMath.MustNewDecFromString("0.00020024906252089700")},
				},
			},
			{
				Forecaster: "forecaster2",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("0.005369218152594270")},
					{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.0002578158768320300")},
					{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("0.0076008583603885900")},
					{Inferer: "worker3", Value: alloraMath.MustNewDecFromString("0.0076269073955871000")},
					{Inferer: "worker4", Value: alloraMath.MustNewDecFromString("0.00035670236460009500")},
				},
			},
		},
	}
	// epoch 2
	infererNetworkRegrets :=
		map[string]inferencesynthesis.Regret{
			"worker0": alloraMath.MustNewDecFromString("0.29240710390153500"),
			"worker1": alloraMath.MustNewDecFromString("0.4182220944854450"),
			"worker2": alloraMath.MustNewDecFromString("0.17663501719135000"),
			"worker3": alloraMath.MustNewDecFromString("0.49617463489106400"),
			"worker4": alloraMath.MustNewDecFromString("0.27996060999688600"),
		}
	forecasterNetworkRegrets := map[string]inferencesynthesis.Regret{
		"forecaster0": alloraMath.MustNewDecFromString("0.816066375505268"),
		"forecaster1": alloraMath.MustNewDecFromString("0.8234558901838660"),
		"forecaster2": alloraMath.MustNewDecFromString("0.8196673550408280"),
	}
	// epoch 2
	networkCombinedLoss := alloraMath.MustNewDecFromString(".0000127791308799785")
	epsilon := alloraMath.MustNewDecFromString("0.0001")
	fTolerance := alloraMath.MustNewDecFromString("0.01")
	pNorm := alloraMath.MustNewDecFromString("2.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	expectedOneOutInferences := []struct {
		Worker string
		Value  string
	}{
		{Worker: "worker0", Value: "-0.0878179883784"},
		{Worker: "worker1", Value: "-0.0834415833800"},
		{Worker: "worker2", Value: "-0.0760530852479"},
		{Worker: "worker3", Value: "-0.0769113408092"},
		{Worker: "worker4", Value: "-0.0977096283034"},
	}
	expectedOneOutImpliedInferences := []struct {
		Worker string
		Value  string
	}{
		{Worker: "forecaster0", Value: "-0.0847805342051"},
		{Worker: "forecaster1", Value: "-0.0882088249132"},
		{Worker: "forecaster2", Value: "-0.0872998460256"},
	}

	for inferer, regret := range infererNetworkRegrets {
		s.emissionsKeeper.SetInfererNetworkRegret(
			s.ctx,
			topicId,
			inferer,
			emissionstypes.TimestampedValue{BlockHeight: 0, Value: regret},
		)
	}

	for forecaster, regret := range forecasterNetworkRegrets {
		s.emissionsKeeper.SetForecasterNetworkRegret(
			s.ctx,
			topicId,
			forecaster,
			emissionstypes.TimestampedValue{BlockHeight: 0, Value: regret},
		)
	}

	sortedInferers := alloraMath.GetSortedKeys(inferenceByWorker)

	infererNormalizedRegrets, err := inferencesynthesis.GetInfererNormalizedRegretsWithMax(
		ctx,
		keeper,
		topicId,
		sortedInferers,
		epsilon,
	)
	require.NoError(err)

	sortedForecasters := alloraMath.GetSortedKeys(forecastImpliedInferenceByWorker)

	// Get Forecaster normalized regrets and max regret
	forecastNormalizedRegrets, err := inferencesynthesis.GetForecasterNormalizedRegretsWithMax(
		ctx,
		keeper,
		topicId,
		sortedForecasters,
		epsilon,
	)
	require.NoError(err)

	oneOutInfererValues, oneOutForecasterValues, err := inferencesynthesis.CalcOneOutInferences(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		inferenceByWorker,
		sortedInferers,
		forecastImpliedInferenceByWorker,
		sortedForecasters,
		forecasts,
		NewWorkersAreNew(false),
		infererNormalizedRegrets,
		forecastNormalizedRegrets,
		networkCombinedLoss,
		epsilon,
		fTolerance,
		pNorm,
		cNorm,
	)

	s.Require().NoError(err, "CalcOneOutInferences should not return an error")

	s.Require().Len(oneOutInfererValues, len(expectedOneOutInferences), "Unexpected number of one-out inferences")
	s.Require().Len(oneOutForecasterValues, len(expectedOneOutImpliedInferences), "Unexpected number of one-out implied inferences")

	for _, expected := range expectedOneOutInferences {
		found := false
		for _, oneOutInference := range oneOutInfererValues {
			if expected.Worker == oneOutInference.Worker {
				found = true
				s.inEpsilon2(oneOutInference.Value, expected.Value)
			}
		}
		if !found {
			s.FailNow("Matching worker not found", "Worker %s not found in returned inferences", expected.Worker)
		}
	}

	for _, expected := range expectedOneOutImpliedInferences {
		found := false
		for _, oneOutImpliedInference := range oneOutForecasterValues {
			if expected.Worker == oneOutImpliedInference.Worker {
				found = true
				s.inEpsilon3(oneOutImpliedInference.Value, expected.Value)
			}
		}
		if !found {
			s.FailNow("Matching worker not found", "Worker %s not found in returned inferences", expected.Worker)
		}
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcOneInInferences() {
	topicId := inferencesynthesis.TopicId(1)

	tests := []struct {
		name                        string
		inferenceByWorker           map[string]*emissionstypes.Inference
		forecastImpliedInferences   map[string]*emissionstypes.Inference
		maxRegretsByOneInForecaster map[string]inferencesynthesis.Regret
		epsilon                     alloraMath.Dec
		fTolerance                  alloraMath.Dec
		pNorm                       alloraMath.Dec
		cNorm                       alloraMath.Dec
		infererNetworkRegrets       inferencesynthesis.NormalizedRegrets
		forecasterNetworkRegrets    map[string]inferencesynthesis.Regret
		expectedOneInInferences     []*emissionstypes.WorkerAttributedValue
		expectedErr                 error
	}{
		{ // EPOCH 3
			name: "basic functionality",
			inferenceByWorker: map[string]*emissionstypes.Inference{
				"worker0": {Value: alloraMath.MustNewDecFromString("-0.0514234892489971")},
				"worker1": {Value: alloraMath.MustNewDecFromString("-0.0316532211989242")},
				"worker2": {Value: alloraMath.MustNewDecFromString("-0.1018014248041400")},
			},
			forecastImpliedInferences: map[string]*emissionstypes.Inference{
				"worker3": {Value: alloraMath.MustNewDecFromString("-0.0707517711518230")},
				"worker4": {Value: alloraMath.MustNewDecFromString("-0.0646463841210426")},
				"worker5": {Value: alloraMath.MustNewDecFromString("-0.0634099113416666")},
			},
			maxRegretsByOneInForecaster: map[string]inferencesynthesis.Regret{
				"worker3": alloraMath.MustNewDecFromString("0.9871536722074480"),
				"worker4": alloraMath.MustNewDecFromString("0.9871536722074480"),
				"worker5": alloraMath.MustNewDecFromString("0.9871536722074480"),
			},
			epsilon:    alloraMath.MustNewDecFromString("0.0001"),
			fTolerance: alloraMath.MustNewDecFromString("0.01"),
			pNorm:      alloraMath.MustNewDecFromString("2.0"),
			cNorm:      alloraMath.MustNewDecFromString("0.75"),
			infererNetworkRegrets: inferencesynthesis.NormalizedRegrets{
				Regrets: map[string]inferencesynthesis.Regret{
					"worker0": alloraMath.MustNewDecFromString("0.6975029322458370"),
					"worker1": alloraMath.MustNewDecFromString("0.9101744424126180"),
					"worker2": alloraMath.MustNewDecFromString("0.9871536722074480"),
				},
				MaxRegret: alloraMath.MustNewDecFromString("0.9871536722074480"),
			},
			forecasterNetworkRegrets: map[string]inferencesynthesis.Regret{
				"worker3": alloraMath.MustNewDecFromString("0.8308330665491310"),
				"worker4": alloraMath.MustNewDecFromString("0.8396961220162480"),
				"worker5": alloraMath.MustNewDecFromString("0.8017696138115460"),
			},
			expectedOneInInferences: []*emissionstypes.WorkerAttributedValue{
				{Worker: "worker3", Value: alloraMath.MustNewDecFromString("-0.06502630286365970")},
				{Worker: "worker4", Value: alloraMath.MustNewDecFromString("-0.06356081320547800")},
				{Worker: "worker5", Value: alloraMath.MustNewDecFromString("-0.06325114823960220")},
			},
			expectedErr: nil,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			for forecaster, regret := range tc.forecasterNetworkRegrets {
				err := s.emissionsKeeper.SetForecasterNetworkRegret(
					s.ctx,
					topicId,
					forecaster,
					emissionstypes.TimestampedValue{BlockHeight: 0, Value: regret},
				)
				s.Require().NoError(err)
			}

			oneInInferences, err := inferencesynthesis.CalcOneInInferences(
				s.ctx,
				s.emissionsKeeper,
				topicId,
				tc.inferenceByWorker,
				alloraMath.GetSortedKeys(tc.inferenceByWorker),
				tc.forecastImpliedInferences,
				alloraMath.GetSortedKeys(tc.forecastImpliedInferences),
				NewWorkersAreNew(false),
				tc.infererNetworkRegrets,
				tc.epsilon,
				tc.fTolerance,
				tc.pNorm,
				tc.cNorm,
			)

			if tc.expectedErr != nil {
				s.Require().ErrorIs(err, tc.expectedErr)
			} else {
				s.Require().NoError(err)
				s.Require().Len(oneInInferences, len(tc.expectedOneInInferences), "Unexpected number of one-in inferences")

				for _, expected := range tc.expectedOneInInferences {
					found := false
					for _, actual := range oneInInferences {
						if expected.Worker == actual.Worker {
							s.Require().True(
								alloraMath.InDelta(
									expected.Value,
									actual.Value,
									alloraMath.MustNewDecFromString("0.0001"),
								),
								"Mismatch in value for one-in inference of worker %s",
								expected.Worker,
							)
							found = true
							break
						}
					}
					if !found {
						s.FailNow("Matching worker not found", "Worker %s not found in actual inferences", expected.Worker)
					}
				}
			}
		})
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcNetworkInferences() {
	k := s.emissionsKeeper
	ctx := s.ctx
	topicId := uint64(1)

	worker1 := "worker1"
	worker2 := "worker2"
	worker3 := "worker3"
	worker4 := "worker4"

	// Set up input data
	inferences := &emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.5")},
			{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.7")},
		},
	}

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: worker3,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.1")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.1")},
				},
			},
			{
				Forecaster: worker4,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.1")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.1")},
				},
			},
		},
	}

	networkCombinedLoss := alloraMath.MustNewDecFromString("0.2")
	epsilon := alloraMath.MustNewDecFromString("0.001")
	fTolerance := alloraMath.MustNewDecFromString("0.01")
	pNorm := alloraMath.MustNewDecFromString("3")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	// Set inferer network regrets
	err := k.SetInfererNetworkRegret(ctx, topicId, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)

	// Set forecaster network regrets
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker4, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.5")})
	s.Require().NoError(err)

	// Set one-in forecaster network regrets
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker3, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker3, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker4, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.6")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker4, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)

	// Call the function
	valueBundle, err := inferencesynthesis.CalcNetworkInferences(ctx, k, topicId, inferences, forecasts, networkCombinedLoss, epsilon, fTolerance, pNorm, cNorm)
	s.Require().NoError(err)

	// Check the results
	s.Require().NotNil(valueBundle)
	s.Require().NotNil(valueBundle.CombinedValue)
	s.Require().NotNil(valueBundle.NaiveValue)
	s.Require().NotEmpty(valueBundle.OneOutInfererValues)
	s.Require().NotEmpty(valueBundle.OneOutForecasterValues)
	s.Require().NotEmpty(valueBundle.OneInForecasterValues)
}

func (s *InferenceSynthesisTestSuite) TestCalcNetworkInferencesSameInfererForecasters() {
	k := s.emissionsKeeper
	ctx := s.ctx
	topicId := uint64(1)
	blockHeightInferences := int64(390)
	blockHeightForecaster := int64(380)

	worker1 := "worker1"
	worker2 := "worker2"

	// Set up input data
	inferences := &emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.52")},
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.71")},
		},
	}

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				TopicId:     topicId,
				BlockHeight: blockHeightForecaster,
				Forecaster:  worker1,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.5")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.8")},
				},
			},
			{
				TopicId:     topicId,
				BlockHeight: blockHeightForecaster,
				Forecaster:  worker2,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.4")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.9")},
				},
			},
		},
	}

	networkCombinedLoss := alloraMath.MustNewDecFromString("1")
	epsilon := alloraMath.MustNewDecFromString("0.001")
	fTolerance := alloraMath.MustNewDecFromString("0.01")
	pNorm := alloraMath.MustNewDecFromString("2")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	// Call the function
	valueBundle, err := inferencesynthesis.CalcNetworkInferences(ctx, k, topicId, inferences, forecasts, networkCombinedLoss, epsilon, fTolerance, pNorm, cNorm)
	s.Require().NoError(err)
	s.Require().NotNil(valueBundle)
	s.Require().NotNil(valueBundle.CombinedValue)
	s.Require().NotNil(valueBundle.NaiveValue)
	s.Require().NotEmpty(valueBundle.OneOutInfererValues)
	s.Require().NotEmpty(valueBundle.OneOutForecasterValues)
	// s.Require().NotEmpty(valueBundle.OneInForecasterValues)

	// Set inferer network regrets
	err = k.SetInfererNetworkRegret(ctx, topicId, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)

	// Set forecaster network regrets
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.5")})
	s.Require().NoError(err)

	// Set one-in forecaster network regrets
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker1, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker1, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker2, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.6")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker2, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)

	// Call the function
	valueBundle, err = inferencesynthesis.CalcNetworkInferences(ctx, k, topicId, inferences, forecasts, networkCombinedLoss, epsilon, fTolerance, pNorm, cNorm)
	s.Require().NoError(err)

	// Check the results
	s.Require().NotNil(valueBundle)
	s.Require().NotNil(valueBundle.CombinedValue)
	s.Require().NotNil(valueBundle.NaiveValue)
	s.Require().NotEmpty(valueBundle.OneOutInfererValues)
	s.Require().NotEmpty(valueBundle.OneOutForecasterValues)
	s.Require().NotEmpty(valueBundle.OneInForecasterValues)
}

func (s *InferenceSynthesisTestSuite) TestCalcNetworkInferencesIncompleteData() {
	k := s.emissionsKeeper
	ctx := s.ctx
	topicId := uint64(1)
	blockHeightInferences := int64(390)
	blockHeightForecaster := int64(380)

	worker1 := "worker1"
	worker2 := "worker2"

	// Set up input data
	inferences := &emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.52")},
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.71")},
		},
	}

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				TopicId:     topicId,
				BlockHeight: blockHeightForecaster,
				Forecaster:  worker1,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.5")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.8")},
				},
			},
			{
				TopicId:     topicId,
				BlockHeight: blockHeightForecaster,
				Forecaster:  worker2,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.4")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.9")},
				},
			},
		},
	}

	networkCombinedLoss := alloraMath.MustNewDecFromString("1")
	epsilon := alloraMath.MustNewDecFromString("0.001")
	fTolerance := alloraMath.MustNewDecFromString("0.01")
	pNorm := alloraMath.MustNewDecFromString("2")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	// Call the function without setting regrets
	valueBundle, err := inferencesynthesis.CalcNetworkInferences(ctx, k, topicId, inferences, forecasts, networkCombinedLoss, epsilon, fTolerance, pNorm, cNorm)
	s.Require().NoError(err)
	s.Require().NotNil(valueBundle)
	s.Require().NotNil(valueBundle.CombinedValue)
	s.Require().NotNil(valueBundle.NaiveValue)
	s.Require().NotEmpty(valueBundle.OneOutInfererValues)
	s.Require().NotEmpty(valueBundle.OneOutForecasterValues)
	// OneInForecastValues come empty because regrets are epsilon
	s.Require().NotEmpty(valueBundle.OneInForecasterValues)
	s.Require().Len(valueBundle.OneInForecasterValues, 2)
}

func (s *InferenceSynthesisTestSuite) TestGetNetworkInferencesAtBlock() {
	require := s.Require()
	keeper := s.emissionsKeeper

	topicId := uint64(1)
	blockHeight := int64(3)
	require.True(blockHeight >= s.ctx.BlockHeight())
	s.ctx = s.ctx.WithBlockHeight(blockHeight)

	simpleNonce := emissionstypes.Nonce{BlockHeight: blockHeight}
	reputerRequestNonce := &emissionstypes.ReputerRequestNonce{
		ReputerNonce: &emissionstypes.Nonce{BlockHeight: blockHeight},
	}

	err := s.emissionsKeeper.SetTopic(s.ctx, topicId, emissionstypes.Topic{
		Id:              topicId,
		Creator:         "creator",
		Metadata:        "metadata",
		LossLogic:       "losslogic",
		LossMethod:      "lossmethod",
		InferenceLogic:  "inferencelogic",
		InferenceMethod: "inferencemethod",
		EpochLastEnded:  0,
		EpochLength:     100,
		GroundTruthLag:  10,
		DefaultArg:      "defaultarg",
		PNorm:           alloraMath.NewDecFromInt64(3),
		AlphaRegret:     alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:   false,
	})
	s.Require().NoError(err)

	reputer0 := "allo1m5v6rgjtxh4xszrrzqacwjh4ve6r0za2gxx9qr"
	reputer1 := "allo1e7cj9839ht2xm8urynqs5279hrvqd8neusvp2x"
	reputer2 := "allo1k9ss0xfer54nyack5678frl36e5g3rj2yzxtfj"
	reputer3 := "allo18ljxewge4vqrkk09tm5heldqg25yj8d9ekgkw5"
	reputer4 := "allo1k36ljvn8z0u49sagdg46p75psgreh23kdjn3l0"

	forecaster0 := "allo1pluvmvsmvecg2ccuqxa6ugzvc3a5udfyy0t76v"
	forecaster1 := "allo1e92saykj94jw3z55g4d3lfz098ppk0suwzc03a"
	forecaster2 := "allo1pk6mxny5p79t8zhkm23z7u3zmfuz2gn0snxkkt"

	// Set Loss bundles

	// EOOCH 2 reputer_x_loss_network_inference
	reputerLossBundles := emissionstypes.ReputerValueBundles{
		ReputerValueBundles: []*emissionstypes.ReputerValueBundle{
			{
				ValueBundle: &emissionstypes.ValueBundle{
					Reputer:             reputer0,
					CombinedValue:       alloraMath.MustNewDecFromString("-4.931794543987090"),
					ReputerRequestNonce: reputerRequestNonce,
					TopicId:             topicId,
				},
			},
			{
				ValueBundle: &emissionstypes.ValueBundle{
					Reputer:             reputer1,
					CombinedValue:       alloraMath.MustNewDecFromString("-5.01650814668254"),
					ReputerRequestNonce: reputerRequestNonce,
					TopicId:             topicId,
				},
			},
			{
				ValueBundle: &emissionstypes.ValueBundle{
					Reputer:             reputer2,
					CombinedValue:       alloraMath.MustNewDecFromString("-4.590153669869230"),
					ReputerRequestNonce: reputerRequestNonce,
					TopicId:             topicId,
				},
			},
			{
				ValueBundle: &emissionstypes.ValueBundle{
					Reputer:             reputer3,
					CombinedValue:       alloraMath.MustNewDecFromString("-4.906627167248600"),
					ReputerRequestNonce: reputerRequestNonce,
					TopicId:             topicId,
				},
			},
			{
				ValueBundle: &emissionstypes.ValueBundle{
					Reputer:             reputer4,
					CombinedValue:       alloraMath.MustNewDecFromString("-4.937932553142900"),
					ReputerRequestNonce: reputerRequestNonce,
					TopicId:             topicId,
				},
			},
		},
	}

	err = keeper.InsertReputerLossBundlesAtBlock(s.ctx, topicId, blockHeight, reputerLossBundles)
	require.NoError(err)

	// Set Stake

	// epoch 2 reputer_stake_x
	stake1, ok := cosmosMath.NewIntFromString("210535178682986000000000")
	s.Require().True(ok)
	err = keeper.AddStake(s.ctx, topicId, reputer0, stake1)
	require.NoError(err)
	stake2, ok := cosmosMath.NewIntFromString("216697599345612000000000")
	s.Require().True(ok)
	err = keeper.AddStake(s.ctx, topicId, reputer1, stake2)
	require.NoError(err)
	stake3, ok := cosmosMath.NewIntFromString("161740203770469000000000")
	s.Require().True(ok)
	err = keeper.AddStake(s.ctx, topicId, reputer2, stake3)
	require.NoError(err)
	stake4, ok := cosmosMath.NewIntFromString("394847513168064000000000")
	s.Require().True(ok)
	err = keeper.AddStake(s.ctx, topicId, reputer3, stake4)
	require.NoError(err)
	stake5, ok := cosmosMath.NewIntFromString("206170060245451000000000")
	s.Require().True(ok)
	err = keeper.AddStake(s.ctx, topicId, reputer4, stake5)
	require.NoError(err)

	// Set Inferences

	// epoch 3 inference_x
	inferences := emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{
				Inferer:     reputer0,
				Value:       alloraMath.MustNewDecFromString("-0.035995138925040600"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Inferer:     reputer1,
				Value:       alloraMath.MustNewDecFromString("-0.07333303938740420"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Inferer:     reputer2,
				Value:       alloraMath.MustNewDecFromString("-0.1495482917094790"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Inferer:     reputer3,
				Value:       alloraMath.MustNewDecFromString("-0.12952123274063800"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Inferer:     reputer4,
				Value:       alloraMath.MustNewDecFromString("-0.0703055329498285"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
		},
	}

	err = keeper.InsertInferences(s.ctx, topicId, simpleNonce, inferences)
	s.Require().NoError(err)

	// Set Forecasts

	// EPOCH 3 forecated_loss_x_for_y
	forecasts := emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: forecaster0,
				ForecastElements: []*emissionstypes.ForecastElement{
					{
						Inferer: reputer0,
						Value:   alloraMath.MustNewDecFromString("-2.480767250656480"),
					},
					{
						Inferer: reputer1,
						Value:   alloraMath.MustNewDecFromString("-3.5546685650440400"),
					},
					{
						Inferer: reputer2,
						Value:   alloraMath.MustNewDecFromString("-4.6188184193555700"),
					},
					{
						Inferer: reputer3,
						Value:   alloraMath.MustNewDecFromString("-3.084052840898730"),
					},
					{
						Inferer: reputer4,
						Value:   alloraMath.MustNewDecFromString("-4.73003856038905"),
					},
				},
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Forecaster: forecaster1,
				ForecastElements: []*emissionstypes.ForecastElement{
					{
						Inferer: reputer0,
						Value:   alloraMath.MustNewDecFromString("-2.6366811669641100"),
					},
					{
						Inferer: reputer1,
						Value:   alloraMath.MustNewDecFromString("-4.668814135864770"),
					},
					{
						Inferer: reputer2,
						Value:   alloraMath.MustNewDecFromString("-1.901004446344670"),
					},
					{
						Inferer: reputer3,
						Value:   alloraMath.MustNewDecFromString("-1.7447621462061600"),
					},
					{
						Inferer: reputer4,
						Value:   alloraMath.MustNewDecFromString("-3.6984295084170300"),
					},
				},
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Forecaster: forecaster2,
				ForecastElements: []*emissionstypes.ForecastElement{
					{
						Inferer: reputer0,
						Value:   alloraMath.MustNewDecFromString("-2.27008895019151"),
					},
					{
						Inferer: reputer1,
						Value:   alloraMath.MustNewDecFromString("-3.5886903414115300"),
					},
					{
						Inferer: reputer2,
						Value:   alloraMath.MustNewDecFromString("-2.119137360333620"),
					},
					{
						Inferer: reputer3,
						Value:   alloraMath.MustNewDecFromString("-2.1176515266976500"),
					},
					{
						Inferer: reputer4,
						Value:   alloraMath.MustNewDecFromString("-3.447694011689490"),
					},
				},
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
		},
	}

	err = keeper.InsertForecasts(s.ctx, topicId, simpleNonce, forecasts)
	s.Require().NoError(err)

	// Set inferer network regrets

	// EPOCH 2 inference_regret_worker_x
	infererNetworkRegrets := map[string]inferencesynthesis.Regret{
		reputer0: alloraMath.MustNewDecFromString("0.29240709744359700"),
		reputer1: alloraMath.MustNewDecFromString("0.41822210449254600"),
		reputer2: alloraMath.MustNewDecFromString("0.17663500756729100"),
		reputer3: alloraMath.MustNewDecFromString("0.4961746382998650"),
		reputer4: alloraMath.MustNewDecFromString("0.27996059439471200"),
	}

	for inferer, regret := range infererNetworkRegrets {
		s.emissionsKeeper.SetInfererNetworkRegret(
			s.ctx,
			topicId,
			inferer,
			emissionstypes.TimestampedValue{BlockHeight: blockHeight, Value: regret},
		)
	}

	// set forecaster network regrets

	// EPOCH 2 inference_regret_worker_5-7
	forecasterNetworkRegrets := map[string]inferencesynthesis.Regret{
		forecaster0: alloraMath.MustNewDecFromString("0.8160663799969630"),
		forecaster1: alloraMath.MustNewDecFromString("0.8234558968607070"),
		forecaster2: alloraMath.MustNewDecFromString("0.8196673491058460"),
	}

	for forecaster, regret := range forecasterNetworkRegrets {
		s.emissionsKeeper.SetForecasterNetworkRegret(
			s.ctx,
			topicId,
			forecaster,
			emissionstypes.TimestampedValue{BlockHeight: blockHeight, Value: regret},
		)
	}

	// Set one in forecaster network regrets

	setOneInForecasterNetworkRegret := func(forecaster string, inferer string, value string) {
		keeper.SetOneInForecasterNetworkRegret(
			s.ctx,
			topicId,
			forecaster,
			inferer,
			emissionstypes.TimestampedValue{
				BlockHeight: blockHeight,
				Value:       alloraMath.MustNewDecFromString(value),
			},
		)
	}

	/// Epoch 3 values inference_regret_worker_x_onein_y

	setOneInForecasterNetworkRegret(forecaster0, reputer0, "-0.0053184502499764600")
	setOneInForecasterNetworkRegret(forecaster0, reputer1, "0.17108317418750000")
	setOneInForecasterNetworkRegret(forecaster0, reputer2, "-0.15971590834870200")
	setOneInForecasterNetworkRegret(forecaster0, reputer3, "0.28707827253180900")
	setOneInForecasterNetworkRegret(forecaster0, reputer4, "-0.019305832050424900")

	setOneInForecasterNetworkRegret(forecaster0, forecaster0, "0.7377753950814930")

	setOneInForecasterNetworkRegret(forecaster1, reputer0, "-0.023527930610206600")
	setOneInForecasterNetworkRegret(forecaster1, reputer1, "0.152873693827269")
	setOneInForecasterNetworkRegret(forecaster1, reputer2, "-0.17792538870893200")
	setOneInForecasterNetworkRegret(forecaster1, reputer3, "0.26886879217157900")
	setOneInForecasterNetworkRegret(forecaster1, reputer4, "-0.03751531241065500")

	setOneInForecasterNetworkRegret(forecaster1, forecaster1, "0.7307092967948180")

	setOneInForecasterNetworkRegret(forecaster2, reputer0, "-0.0249860078679687")
	setOneInForecasterNetworkRegret(forecaster2, reputer1, "0.15141561656950700")
	setOneInForecasterNetworkRegret(forecaster2, reputer2, "-0.1793834659666940")
	setOneInForecasterNetworkRegret(forecaster2, reputer3, "0.26741071491381700")
	setOneInForecasterNetworkRegret(forecaster2, reputer4, "-0.0389733896684171")

	setOneInForecasterNetworkRegret(forecaster2, forecaster2, "0.7231311719266530")

	// Calculate

	valueBundle, err :=
		inferencesynthesis.GetNetworkInferencesAtBlock(
			s.ctx,
			s.emissionsKeeper,
			topicId,
			blockHeight,
			blockHeight,
		)
	require.NoError(err)

	// epoch 3 network_inference
	s.inEpsilon2(valueBundle.CombinedValue, "-0.08595551799339320")
	// epoch 3 network_naive_inference
	s.inEpsilon2(valueBundle.NaiveValue, "-0.0903194538050178")

	for _, inference := range inferences.Inferences {
		found := false
		for _, infererValue := range valueBundle.InfererValues {
			if string(inference.Inferer) == infererValue.Worker {
				found = true
				require.Equal(inference.Value, infererValue.Value)
			}
		}
		require.True(found, "Inference not found")
	}

	// epoch 3 forecast_implied_inference_x
	for _, forecasterValue := range valueBundle.ForecasterValues {
		switch string(forecasterValue.Worker) {
		case forecaster0:
			s.inEpsilon2(forecasterValue.Value, "-0.1025675327315210")
		case forecaster1:
			s.inEpsilon2(forecasterValue.Value, "-0.07302318589259440")
		case forecaster2:
			s.inEpsilon2(forecasterValue.Value, "-0.07233832253513270")
		default:
			require.Fail("Unexpected forecaster %v", forecasterValue.Worker)
		}
	}

	// epoch 3 network_inference_oneout_x
	for _, oneOutInfererValue := range valueBundle.OneOutInfererValues {
		switch string(oneOutInfererValue.Worker) {
		case reputer0:
			s.inEpsilon2(oneOutInfererValue.Value, "-0.09193875537568730")
		case reputer1:
			s.inEpsilon2(oneOutInfererValue.Value, "-0.08725723597921200")
		case reputer2:
			s.inEpsilon2(oneOutInfererValue.Value, "-0.07611975173450570")
		case reputer3:
			s.inEpsilon2(oneOutInfererValue.Value, "-0.07876098982982720")
		case reputer4:
			s.inEpsilon2(oneOutInfererValue.Value, "-0.09549195980225840")
		default:
			require.Fail("Unexpected worker %v", oneOutInfererValue.Worker)
		}
	}

	// epoch 3 network_naive_inference_onein_x
	for _, oneInForecasterValue := range valueBundle.OneInForecasterValues {
		switch string(oneInForecasterValue.Worker) {
		case forecaster0:
			s.inEpsilon2(oneInForecasterValue.Value, "-0.09101522746174310")
		case forecaster1:
			s.inEpsilon2(oneInForecasterValue.Value, "-0.08506554261579950")
		case forecaster2:
			s.inEpsilon2(oneInForecasterValue.Value, "-0.08491153277529740")
		default:
			require.Fail("Unexpected worker %v", oneInForecasterValue.Worker)
		}
	}

	// epoch 3 network_inference_oneout_5-7
	for _, oneOutForecasterValue := range valueBundle.OneOutForecasterValues {
		switch string(oneOutForecasterValue.Worker) {
		case forecaster0:
			s.inEpsilon2(oneOutForecasterValue.Value, "-0.0831086124651002")
		case forecaster1:
			s.inEpsilon2(oneOutForecasterValue.Value, "-0.0880928742443665")
		case forecaster2:
			s.inEpsilon2(oneOutForecasterValue.Value, "-0.08821296390705380")
		default:
			require.Fail("Unexpected worker %v", oneOutForecasterValue.Worker)
		}
	}
}

func (s *InferenceSynthesisTestSuite) TestFilterNoncesWithinEpochLength() {
	tests := []struct {
		name          string
		nonces        emissionstypes.Nonces
		blockHeight   int64
		epochLength   int64
		expectedNonce emissionstypes.Nonces
	}{
		{
			name: "Nonces within epoch length",
			nonces: emissionstypes.Nonces{
				Nonces: []*emissionstypes.Nonce{
					{BlockHeight: 10},
					{BlockHeight: 15},
				},
			},
			blockHeight: 20,
			epochLength: 10,
			expectedNonce: emissionstypes.Nonces{
				Nonces: []*emissionstypes.Nonce{
					{BlockHeight: 10},
					{BlockHeight: 15},
				},
			},
		},
		{
			name: "Nonces outside epoch length",
			nonces: emissionstypes.Nonces{
				Nonces: []*emissionstypes.Nonce{
					{BlockHeight: 5},
					{BlockHeight: 15},
				},
			},
			blockHeight: 20,
			epochLength: 10,
			expectedNonce: emissionstypes.Nonces{
				Nonces: []*emissionstypes.Nonce{
					{BlockHeight: 15},
				},
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			actual := inferencesynthesis.FilterNoncesWithinEpochLength(tc.nonces, tc.blockHeight, tc.epochLength)
			s.Require().Equal(tc.expectedNonce, actual, "Filter nonces do not match")
		})
	}
}

func (s *InferenceSynthesisTestSuite) TestSelectTopNReputerNonces() {
	// Define test cases
	tests := []struct {
		name                     string
		reputerRequestNonces     *emissionstypes.ReputerRequestNonces
		N                        int
		expectedTopNReputerNonce []*emissionstypes.ReputerRequestNonce
		currentBlockHeight       int64
		groundTruthLag           int64
	}{
		{
			name: "N greater than length of nonces, zero lag",
			reputerRequestNonces: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 1}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 2}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 3}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 4}},
				},
			},
			N: 5,
			expectedTopNReputerNonce: []*emissionstypes.ReputerRequestNonce{
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 3}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 4}},
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 1}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 2}},
			},
			currentBlockHeight: 10,
			groundTruthLag:     0,
		},
		{
			name: "N less than length of nonces, zero lag",
			reputerRequestNonces: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 1}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 2}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 3}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 4}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 5}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 6}},
				},
			},
			N: 2,
			expectedTopNReputerNonce: []*emissionstypes.ReputerRequestNonce{
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 5}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 6}},
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 3}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 4}},
			},
			currentBlockHeight: 10,
			groundTruthLag:     0,
		},
		{
			name: "Ground truth lag cutting selection midway",
			reputerRequestNonces: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 2}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 1}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 6}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 5}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 4}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 3}},
				},
			},
			N: 3,
			expectedTopNReputerNonce: []*emissionstypes.ReputerRequestNonce{
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 4}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 3}},
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 2}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 1}},
			},
			currentBlockHeight: 10,
			groundTruthLag:     6,
		},
		{
			name: "Big Ground truth lag, not selecting any nonces",
			reputerRequestNonces: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 2}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 1}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 6}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 5}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 4}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 3}},
				},
			},
			N:                        3,
			expectedTopNReputerNonce: []*emissionstypes.ReputerRequestNonce{},
			currentBlockHeight:       10,
			groundTruthLag:           10,
		},
		{
			name: "Small ground truth lag, selecting all nonces",
			reputerRequestNonces: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 6}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 5}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 5}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 4}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 4}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 3}},
				},
			},
			N: 3,
			expectedTopNReputerNonce: []*emissionstypes.ReputerRequestNonce{
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 6}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 5}},
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 5}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 4}},
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 4}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 3}},
			},
			currentBlockHeight: 10,
			groundTruthLag:     2,
		},
		{
			name: "Mid ground truth lag, selecting some nonces",
			reputerRequestNonces: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 6}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 5}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 5}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 4}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 4}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 3}},
				},
			},
			N: 3,
			expectedTopNReputerNonce: []*emissionstypes.ReputerRequestNonce{
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 5}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 4}},
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 4}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 3}},
			},
			currentBlockHeight: 10,
			groundTruthLag:     5,
		},
	}

	// Run test cases
	for _, tc := range tests {
		s.Run(tc.name, func() {
			actual := inferencesynthesis.SelectTopNReputerNonces(tc.reputerRequestNonces, tc.N, tc.currentBlockHeight, tc.groundTruthLag)
			s.Require().Equal(tc.expectedTopNReputerNonce, actual, "Reputer nonces do not match")
		})
	}
}

func (s *InferenceSynthesisTestSuite) TestSelectTopNWorkerNonces() {
	// Define test cases
	tests := []struct {
		name               string
		workerNonces       emissionstypes.Nonces
		N                  int
		expectedTopNNonces []*emissionstypes.Nonce
	}{
		{
			name: "N greater than length of nonces",
			workerNonces: emissionstypes.Nonces{
				Nonces: []*emissionstypes.Nonce{
					{BlockHeight: 1},
					{BlockHeight: 2},
				},
			},
			N: 5,
			expectedTopNNonces: []*emissionstypes.Nonce{
				{BlockHeight: 1},
				{BlockHeight: 2},
			},
		},
		{
			name: "N less than length of nonces",
			workerNonces: emissionstypes.Nonces{
				Nonces: []*emissionstypes.Nonce{
					{BlockHeight: 1},
					{BlockHeight: 2},
					{BlockHeight: 3},
				},
			},
			N: 2,
			expectedTopNNonces: []*emissionstypes.Nonce{
				{BlockHeight: 1},
				{BlockHeight: 2},
			},
		},
	}

	// Run test cases
	for _, tc := range tests {
		s.Run(tc.name, func() {
			actual := inferencesynthesis.SelectTopNWorkerNonces(tc.workerNonces, tc.N)
			s.Require().Equal(actual, tc.expectedTopNNonces, "Worker nonces to not match")
		})
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcNetworkInferencesTwoWorkerTwoForecasters() {
	k := s.emissionsKeeper
	ctx := s.ctx
	topicId := uint64(1)

	worker1 := "worker1"
	worker2 := "worker2"
	worker3 := "worker3"
	worker4 := "worker4"

	// Set up input data
	inferences := &emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.5")},
			{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.7")},
		},
	}

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: worker3,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.1")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.1")},
				},
			},
			{
				Forecaster: worker4,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.1")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.1")},
				},
			},
		},
	}

	networkCombinedLoss := alloraMath.MustNewDecFromString("0.2")
	epsilon := alloraMath.MustNewDecFromString("0.001")
	fTolerance := alloraMath.MustNewDecFromString("0.01")
	pNorm := alloraMath.MustNewDecFromString("3")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	// Set inferer network regrets
	err := k.SetInfererNetworkRegret(ctx, topicId, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)

	// Set forecaster network regrets
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker4, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.5")})
	s.Require().NoError(err)

	// Set one-in forecaster network regrets
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker3, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker3, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker4, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.6")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker4, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)

	// Call the function
	valueBundle, err := inferencesynthesis.CalcNetworkInferences(ctx, k, topicId, inferences, forecasts, networkCombinedLoss, epsilon, fTolerance, pNorm, cNorm)
	s.Require().NoError(err)

	// Check the results
	s.Require().NotNil(valueBundle)
	s.Require().NotNil(valueBundle.CombinedValue)
	s.Require().NotNil(valueBundle.NaiveValue)

	s.Require().Len(valueBundle.OneOutInfererValues, 2)
	s.Require().Len(valueBundle.OneOutForecasterValues, 2)
	s.Require().Len(valueBundle.OneInForecasterValues, 2)
}

func (s *InferenceSynthesisTestSuite) TestCalcNetworkInferencesThreeWorkerThreeForecasters() {
	k := s.emissionsKeeper
	ctx := s.ctx
	topicId := uint64(1)

	worker1 := "worker1"
	worker2 := "worker2"
	worker3 := "worker3"

	forecaster1 := "forecaster1"
	forecaster2 := "forecaster2"
	forecaster3 := "forecaster3"

	// Set up input data
	inferences := &emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.1")},
			{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.2")},
			{Inferer: worker3, Value: alloraMath.MustNewDecFromString("0.3")},
		},
	}

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: forecaster1,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.1")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.1")},
					{Inferer: worker3, Value: alloraMath.MustNewDecFromString("0.1")},
				},
			},
			{
				Forecaster: forecaster2,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.1")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.1")},
					{Inferer: worker3, Value: alloraMath.MustNewDecFromString("0.1")},
				},
			},
			{
				Forecaster: forecaster3,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.1")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.1")},
					{Inferer: worker3, Value: alloraMath.MustNewDecFromString("0.1")},
				},
			},
		},
	}

	networkCombinedLoss := alloraMath.MustNewDecFromString("0.3")
	epsilon := alloraMath.MustNewDecFromString("0.001")
	fTolerance := alloraMath.MustNewDecFromString("0.01")
	pNorm := alloraMath.MustNewDecFromString("3")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	// Set inferer network regrets
	err := k.SetInfererNetworkRegret(ctx, topicId, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.1")})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)

	// Set forecaster network regrets
	err = k.SetForecasterNetworkRegret(ctx, topicId, forecaster1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, forecaster2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.5")})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, forecaster3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.6")})
	s.Require().NoError(err)

	// Set one-in forecaster network regrets
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster1, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.7")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster1, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.8")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster1, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.9")})
	s.Require().NoError(err)

	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster2, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.1")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster2, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster2, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)

	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster3, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster3, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.5")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster3, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.6")})
	s.Require().NoError(err)

	// Call the function
	valueBundle, err := inferencesynthesis.CalcNetworkInferences(ctx, k, topicId, inferences, forecasts, networkCombinedLoss, epsilon, fTolerance, pNorm, cNorm)
	s.Require().NoError(err)

	// Check the results
	s.Require().NotNil(valueBundle)
	s.Require().NotNil(valueBundle.CombinedValue)
	s.Require().NotNil(valueBundle.NaiveValue)

	s.Require().Len(valueBundle.OneOutInfererValues, 3)
	s.Require().Len(valueBundle.OneOutForecasterValues, 3)
	s.Require().Len(valueBundle.OneInForecasterValues, 3)
}

func (s *InferenceSynthesisTestSuite) TestSortByBlockHeight() {
	// Create some test data
	tests := []struct {
		name   string
		input  *emissionstypes.ReputerRequestNonces
		output *emissionstypes.ReputerRequestNonces
	}{
		{
			name: "Sorted in descending order",
			input: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 5}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 2}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 7}},
				},
			},
			output: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 7}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 5}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 2}},
				},
			},
		},
		{
			name: "Already sorted",
			input: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 10}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 9}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 7}},
				},
			},
			output: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 10}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 9}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 7}},
				},
			},
		},
		{
			name: "Empty input",
			input: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{},
			},
			output: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{},
			},
		},
		{
			name: "Single element",
			input: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 3}},
				},
			},
			output: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 3}},
				},
			},
		},
	}

	for _, test := range tests {
		s.Run(test.name, func() {
			// Call the sorting function
			inferencesynthesis.SortByBlockHeight(test.input.Nonces)

			// Compare the sorted input with the expected output
			s.Require().Equal(test.input.Nonces, test.output.Nonces, "Sorting result mismatch.\nExpected: %v\nGot: %v")
		})
	}
}
