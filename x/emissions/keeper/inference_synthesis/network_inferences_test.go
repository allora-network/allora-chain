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

const headers = "epoch,time,returns,inference_0,inference_1,inference_2,inference_3,inference_4,forecasted_loss_0_for_0,forecasted_loss_0_for_1,forecasted_loss_0_for_2,forecasted_loss_0_for_3,forecasted_loss_0_for_4,forecasted_loss_1_for_0,forecasted_loss_1_for_1,forecasted_loss_1_for_2,forecasted_loss_1_for_3,forecasted_loss_1_for_4,forecasted_loss_2_for_0,forecasted_loss_2_for_1,forecasted_loss_2_for_2,forecasted_loss_2_for_3,forecasted_loss_2_for_4,forecast_implied_inference_0,forecast_implied_inference_1,forecast_implied_inference_2,forecast_implied_inference_0_oneout_0,forecast_implied_inference_0_oneout_1,forecast_implied_inference_0_oneout_2,forecast_implied_inference_0_oneout_3,forecast_implied_inference_0_oneout_4,forecast_implied_inference_1_oneout_0,forecast_implied_inference_1_oneout_1,forecast_implied_inference_1_oneout_2,forecast_implied_inference_1_oneout_3,forecast_implied_inference_1_oneout_4,forecast_implied_inference_2_oneout_0,forecast_implied_inference_2_oneout_1,forecast_implied_inference_2_oneout_2,forecast_implied_inference_2_oneout_3,forecast_implied_inference_2_oneout_4,network_inference,network_naive_inference,network_inference_oneout_0,network_inference_oneout_1,network_inference_oneout_2,network_inference_oneout_3,network_inference_oneout_4,network_inference_oneout_5,network_inference_oneout_6,network_inference_oneout_7,network_naive_inference_onein_0,network_naive_inference_onein_1,network_naive_inference_onein_2,network_loss,reputer_stake_0,reputer_stake_1,reputer_stake_2,reputer_stake_3,reputer_stake_4,reputer_0_loss_inference_0,reputer_0_loss_inference_1,reputer_0_loss_inference_2,reputer_0_loss_inference_3,reputer_0_loss_inference_4,reputer_1_loss_inference_0,reputer_1_loss_inference_1,reputer_1_loss_inference_2,reputer_1_loss_inference_3,reputer_1_loss_inference_4,reputer_2_loss_inference_0,reputer_2_loss_inference_1,reputer_2_loss_inference_2,reputer_2_loss_inference_3,reputer_2_loss_inference_4,reputer_3_loss_inference_0,reputer_3_loss_inference_1,reputer_3_loss_inference_2,reputer_3_loss_inference_3,reputer_3_loss_inference_4,reputer_4_loss_inference_0,reputer_4_loss_inference_1,reputer_4_loss_inference_2,reputer_4_loss_inference_3,reputer_4_loss_inference_4,reputer_0_loss_forecast_implied_inference_0,reputer_0_loss_forecast_implied_inference_1,reputer_0_loss_forecast_implied_inference_2,reputer_1_loss_forecast_implied_inference_0,reputer_1_loss_forecast_implied_inference_1,reputer_1_loss_forecast_implied_inference_2,reputer_2_loss_forecast_implied_inference_0,reputer_2_loss_forecast_implied_inference_1,reputer_2_loss_forecast_implied_inference_2,reputer_3_loss_forecast_implied_inference_0,reputer_3_loss_forecast_implied_inference_1,reputer_3_loss_forecast_implied_inference_2,reputer_4_loss_forecast_implied_inference_0,reputer_4_loss_forecast_implied_inference_1,reputer_4_loss_forecast_implied_inference_2,inference_loss_0,inference_loss_1,inference_loss_2,inference_loss_3,inference_loss_4,forecast_implied_inference_loss_0,forecast_implied_inference_loss_1,forecast_implied_inference_loss_2,inference_regret_worker_0,inference_regret_worker_1,inference_regret_worker_2,inference_regret_worker_3,inference_regret_worker_4,inference_regret_worker_5,inference_regret_worker_6,inference_regret_worker_7,inference_regret_worker_0_onein_0,inference_regret_worker_1_onein_0,inference_regret_worker_2_onein_0,inference_regret_worker_3_onein_0,inference_regret_worker_4_onein_0,inference_regret_worker_5_onein_0,inference_regret_worker_0_onein_1,inference_regret_worker_1_onein_1,inference_regret_worker_2_onein_1,inference_regret_worker_3_onein_1,inference_regret_worker_4_onein_1,inference_regret_worker_5_onein_1,inference_regret_worker_0_onein_2,inference_regret_worker_1_onein_2,inference_regret_worker_2_onein_2,inference_regret_worker_3_onein_2,inference_regret_worker_4_onein_2,inference_regret_worker_5_onein_2,reputer_0_loss_network_inference,reputer_1_loss_network_inference,reputer_2_loss_network_inference,reputer_3_loss_network_inference,reputer_4_loss_network_inference,network_loss_reputers"

func GetCsvToSimulatedValuesMap(
	headers string,
	values string,
) (
	simulatedValuesStr map[string]string,
	simulatedValuesDec map[string]alloraMath.Dec,
) {
	r := csv.NewReader(strings.NewReader(headers + "\n" + values + "\n"))
	headersRead, err := r.Read()
	if err != nil {
		panic(err)
	}
	valuesRead, err := r.Read()
	if err != nil {
		panic(err)
	}
	simulatedValuesStr = make(map[string]string)
	simulatedValuesDec = make(map[string]alloraMath.Dec)
	if len(headersRead) != len(valuesRead) {
		panic("Header and values length mismatch")
	}
	for i := 0; i < len(headersRead); i++ {
		simulatedValuesStr[headersRead[i]] = valuesRead[i]
		decVal, err := alloraMath.NewDecFromString(valuesRead[i])
		if err == nil {
			simulatedValuesDec[headersRead[i]] = decVal
		}
	}
	return simulatedValuesStr, simulatedValuesDec
}

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
	epoch1 := "1,2020-12-02 00:00:00,0.08290039236125041,0.02905393909123362,0.14451360970082316,0.24140254750009996,-0.024901742038025046,0.06783123836746127,-1.788924771500234,-4.933035320168331,-1.8802902124453127,-1.9836578582550644,-0.796936567997003,-2.667637909568831,-2.9152075223759946,-2.124289856934157,-2.4668422234226295,-2.974245167711838,-2.5209310106534204,-2.906120469109913,-0.7339391004380008,-2.0913144053608934,-2.8330167420021035,0.0915799185243186,0.0915799185243186,0.0915799185243186,0.0915799185243186,0.0915799185243186,0.0915799185243186,0.0915799185243186,0.0915799185243186,0.0915799185243186,0.0915799185243186,0.0915799185243186,0.0915799185243186,0.0915799185243186,0.0915799185243186,0.0915799185243186,0.0915799185243186,0.0915799185243186,0.0915799185243186,0.0915799185243186,0.09157991852431857,0.1005122013004736,0.0840179626419608,0.07017668581349269,0.10822015574751055,0.09497258711815537,0.0915799185243186,0.0915799185243186,0.0915799185243186,0.0915799185243186,0.0915799185243186,0.0915799185243186,-4.9196819027651495,210150.9321900143,216316.619889842,161451.73318335626,394151.55122023926,205705.2717820282,-2.2286806355560502,-2.7996519386759973,-1.543339291620877,-3.2359909515400442,-2.2471494166894765,-2.1156443518238133,-3.0416963274786792,-1.549281026385266,-3.4499788131793196,-1.9381930340847044,-2.139120638363718,-2.8077860035649795,-1.7658817069115689,-3.2006418063475492,-2.055424871031586,-2.1424866687998136,-2.778862230042374,-1.5354869413262946,-3.265820755056018,-2.0565779761476453,-2.2602868036272192,-2.87634248550558,-1.5434650590847583,-3.2293353040475563,-2.130806492365704,-4.877511659649782,-5.071175272791933,-4.84303511943602,-5.087731113530607,-5.0436118804321275,-4.928267908068103,-4.72236496146053,-5.055283638140658,-5.1098237661794945,-4.950048965819536,-4.887812939978951,-4.954365454280479,-5.0581185505230035,-5.014283008708997,-5.09540885532074,-2.172792010216437,-2.8512212844208205,-1.572087141808633,-3.278903341650252,-2.081433806672764,-4.950057099063459,-4.993295495836582,-4.975472865206469,0.6253110107451287,0.6931539381655671,0.5652405239043483,0.7359221438885103,0.6161751903907615,0.9030375196298309,0.9073613593071433,0.905579096244132,0.6244348454149327,0.692277772835371,0.5643643585741522,0.7350459785583141,0.6152990250605652,0.9021613542996348,0.6173870404932691,0.6852299679137074,0.5573165536524887,0.7279981736366505,0.6082512201389016,0.8994373890552836,0.6168416306591564,0.6846845580795947,0.556771143818376,0.7274527638025379,0.607705810304789,0.8971097161581596,-4.940603611195221,-5.029165147104658,-4.647875837737493,-4.936600201502898,-4.96409216331972,-4.9196819027651495"
	epoch2 := "2,2020-12-03 00:00:00,0.03178207880760021,-0.230622933739544,-0.19693894066605602,0.048704500498029504,0.054145121711977245,0.22919548623217473,-1.18172420646634,0.26621077264804827,-3.3897339254838474,-2.571846047295651,-2.0259184257783027,-0.5499700025611789,-1.7328740794514994,-4.338275221094591,-2.724483852551699,-2.1336429998512143,-0.5019942929743771,-0.6804897420817917,-3.792402810523422,-2.811890974392894,-1.4041161461317468,0.05403102080389692,0.04922834703615315,0.04954788276682784,0.05685567279701529,0.05000352194763739,0.08022256316427133,0.05481428931368454,0.047321301762175076,0.04889426405222525,0.050203045157962686,0.06563711044975742,0.04928523873520655,0.047878442353090994,0.04951209695386524,0.04968602947180613,0.055508789681751255,0.04913573002937966,0.04878697422828431,0.007161310580432416,-0.019103353192683702,0.04148117165417589,0.035902110182863385,0.00816388526204742,0.0005104814861249907,-0.025817933407434695,0.00046563769136606246,0.001151733943900887,0.0011060859823759312,-0.006914290859920277,-0.007714736487877573,-0.007661480532765125,-4.893498750410228,210535.17868298586,216697.59934561152,161740.20377046912,394847.51316806424,206170.06024545094,-2.2343006528249094,-2.792550873320775,-1.5440642777461833,-3.2179096445281856,-2.266891105877178,-2.121183041048061,-3.0327585803388817,-1.5537651318034542,-3.4287428212464928,-1.9692816101769068,-2.1920231037407203,-2.785811084367808,-1.753455977343409,-3.1195990096203157,-2.2110795616014465,-2.1571031771226314,-2.761718838699133,-1.5380057584990876,-3.19672982116354,-2.1369799918404144,-2.2771837795724736,-2.8628982847566364,-1.5463939783257883,-3.195013388928508,-2.1833305864744603,-4.867574982517031,-5.059471261766153,-4.834883795282288,-5.074813464108836,-5.028848891826424,-4.915862550163092,-4.6834319708821806,-4.979989953804598,-5.004204282733437,-4.921832064816302,-4.846165908785239,-4.9183980418244335,-5.032287470995465,-4.998807039058706,-5.063485917337664,-2.1897706281400366,-2.8373343518455867,-1.572684110944005,-3.231945838412287,-2.147527980840491,-4.926824873711378,-4.9618054852530085,-4.939960375271497,0.29240709744359666,0.41822210449254626,0.17663500756729117,0.4961746382998652,0.27996059439471166,0.8160663799969627,0.8234558968607071,0.8196673491058456,0.29086859544474397,0.4166836024936934,0.17509650556843842,0.49463613630101233,0.27842209239585874,0.8145278779981101,0.2780533480954255,0.4038683551443749,0.16228125821911993,0.48182088895169384,0.26560684504654025,0.8091021475125357,0.27692565343841374,0.40274066048736323,0.1611535635621082,0.48069319429468216,0.2644791503895285,0.8041859051006626,-4.931794543987086,-5.01650814668254,-4.590153669869233,-4.906627167248604,-4.937932553142897,-4.893498750410228"
	_, simulatedEpoch1 := GetCsvToSimulatedValuesMap(headers, epoch1)
	_, simulatedEpoch2 := GetCsvToSimulatedValuesMap(headers, epoch2)

	// EPOCH 2
	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0": {Value: simulatedEpoch2["inference_0"]},
		"worker1": {Value: simulatedEpoch2["inference_1"]},
		"worker2": {Value: simulatedEpoch2["inference_2"]},
		"worker3": {Value: simulatedEpoch2["inference_3"]},
		"worker4": {Value: simulatedEpoch2["inference_4"]},
	}

	forecastImpliedInferenceByWorker := map[string]*emissionstypes.Inference{
		"worker5": {Value: simulatedEpoch2["forecast_implied_inference_0"]},
	}

	epsilon := alloraMath.MustNewDecFromString("0.0001")
	pNorm := alloraMath.MustNewDecFromString("3")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	infererNetworkRegrets := map[string]inferencesynthesis.Regret{ // FROM EPOCH 1
		"worker0": simulatedEpoch1["inference_regret_worker_0_onein_0"],
		"worker1": simulatedEpoch1["inference_regret_worker_1_onein_0"],
		"worker2": simulatedEpoch1["inference_regret_worker_2_onein_0"],
		"worker3": simulatedEpoch1["inference_regret_worker_3_onein_0"],
		"worker4": simulatedEpoch1["inference_regret_worker_4_onein_0"],
	}

	forecasterNetworkRegrets := map[string]inferencesynthesis.Regret{ // FROM EPOCH 1
		"worker5": simulatedEpoch1["inference_regret_worker_5_onein_0"],
	}

	// FROM EPOCH 2
	expectedNetworkCombinedInferenceValue := simulatedEpoch2["network_naive_inference_onein_0"]

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
	epoch2 := "2,2020-12-03 00:00:00,0.03178207880760021,-0.230622933739544,-0.19693894066605602,0.048704500498029504,0.054145121711977245,0.22919548623217473,-1.18172420646634,0.26621077264804827,-3.3897339254838474,-2.571846047295651,-2.0259184257783027,-0.5499700025611789,-1.7328740794514994,-4.338275221094591,-2.724483852551699,-2.1336429998512143,-0.5019942929743771,-0.6804897420817917,-3.792402810523422,-2.811890974392894,-1.4041161461317468,0.05403102080389692,0.04922834703615315,0.04954788276682784,0.05685567279701529,0.05000352194763739,0.08022256316427133,0.05481428931368454,0.047321301762175076,0.04889426405222525,0.050203045157962686,0.06563711044975742,0.04928523873520655,0.047878442353090994,0.04951209695386524,0.04968602947180613,0.055508789681751255,0.04913573002937966,0.04878697422828431,0.007161310580432416,-0.019103353192683702,0.04148117165417589,0.035902110182863385,0.00816388526204742,0.0005104814861249907,-0.025817933407434695,0.00046563769136606246,0.001151733943900887,0.0011060859823759312,-0.006914290859920277,-0.007714736487877573,-0.007661480532765125,-4.893498750410228,210535.17868298586,216697.59934561152,161740.20377046912,394847.51316806424,206170.06024545094,-2.2343006528249094,-2.792550873320775,-1.5440642777461833,-3.2179096445281856,-2.266891105877178,-2.121183041048061,-3.0327585803388817,-1.5537651318034542,-3.4287428212464928,-1.9692816101769068,-2.1920231037407203,-2.785811084367808,-1.753455977343409,-3.1195990096203157,-2.2110795616014465,-2.1571031771226314,-2.761718838699133,-1.5380057584990876,-3.19672982116354,-2.1369799918404144,-2.2771837795724736,-2.8628982847566364,-1.5463939783257883,-3.195013388928508,-2.1833305864744603,-4.867574982517031,-5.059471261766153,-4.834883795282288,-5.074813464108836,-5.028848891826424,-4.915862550163092,-4.6834319708821806,-4.979989953804598,-5.004204282733437,-4.921832064816302,-4.846165908785239,-4.9183980418244335,-5.032287470995465,-4.998807039058706,-5.063485917337664,-2.1897706281400366,-2.8373343518455867,-1.572684110944005,-3.231945838412287,-2.147527980840491,-4.926824873711378,-4.9618054852530085,-4.939960375271497,0.29240709744359666,0.41822210449254626,0.17663500756729117,0.4961746382998652,0.27996059439471166,0.8160663799969627,0.8234558968607071,0.8196673491058456,0.29086859544474397,0.4166836024936934,0.17509650556843842,0.49463613630101233,0.27842209239585874,0.8145278779981101,0.2780533480954255,0.4038683551443749,0.16228125821911993,0.48182088895169384,0.26560684504654025,0.8091021475125357,0.27692565343841374,0.40274066048736323,0.1611535635621082,0.48069319429468216,0.2644791503895285,0.8041859051006626,-4.931794543987086,-5.01650814668254,-4.590153669869233,-4.906627167248604,-4.937932553142897,-4.893498750410228"
	epoch3 := "3,2020-12-04 00:00:00,-0.07990917965471393,-0.035995138925040554,-0.07333303938740415,-0.1495482917094787,-0.12952123274063815,-0.0703055329498285,-2.480767250656477,-3.5546685650440417,-4.6188184193555735,-3.084052840898731,-4.73003856038905,-2.6366811669641064,-4.668814135864767,-1.901004446344669,-1.7447621462061556,-3.6984295084170293,-2.2700889501915076,-3.5886903414115316,-2.119137360333619,-2.1176515266976472,-3.447694011689486,-0.1025675327315208,-0.07302318589259435,-0.07233832253513268,-0.10102068312719244,-0.10364793148967034,-0.0705067589574468,-0.10275234779836752,-0.14807539029121972,-0.07319270251459319,-0.0698427104144564,-0.0729888677103869,-0.07305532421316024,-0.07321629821257045,-0.07243222844764322,-0.07036013256420398,-0.07226434253912817,-0.07228409928815509,-0.0733821377291926,-0.08595551799339318,-0.0903194538050178,-0.09193875537568731,-0.08725723597921196,-0.07611975173450566,-0.07876098982982715,-0.09549195980225844,-0.0831086124651002,-0.0880928742443665,-0.08821296390705377,-0.09101522746174305,-0.08506554261579953,-0.08491153277529744,-4.83458359235066,210936.14772502996,217043.74857975644,162033.2054079192,395516.59487132856,206620.42964977756,-2.217852686051446,-2.7711015061857087,-1.5734375187707184,-3.2199574954128085,-2.254909150156216,-2.108880385172198,-3.007950683438181,-1.585144682537212,-3.4266914775613926,-1.960502548939211,-2.077444340504019,-2.6673314342959764,-1.900240284607539,-3.154663369533265,-2.142815319485575,-2.112662627942502,-2.6900735880222775,-1.633512574150124,-3.1988046047983345,-2.1048421628128926,-2.243965671368461,-2.811499693808028,-1.6115919229427993,-3.2021438541666454,-2.16498558347734,-4.847406907619317,-5.039758132164747,-4.817671740563477,-5.048913731669618,-5.006006193860714,-4.893551054567161,-4.575929170587921,-4.867617557617695,-4.874937265335567,-4.839456170853645,-4.786611853071776,-4.851181928372789,-4.974365559800201,-4.9547049150212645,-5.013835876725157,-2.1485564569077433,-2.780237637841959,-1.6465306848072383,-3.23861581701918,-2.1207011663432267,-4.866561367242139,-4.911489536203998,-4.884385989893725,-0.0054363258450547125,0.1709652985924215,-0.15983378394378017,0.28696039693673064,-0.019423707645502886,0.7376575194864143,0.7488009015599703,0.7426808539495675,-0.0053184502499764585,0.17108317418749958,-0.15971590834870197,0.28707827253180873,-0.019305832050424854,0.7377753950814926,-0.023527930610206638,0.15287369382726942,-0.17792538870893213,0.26886879217157855,-0.03751531241065495,0.7307092967948181,-0.0249860078679687,0.15141561656950742,-0.1793834659666942,0.26741071491381657,-0.0389733896684171,0.7231311719266534,-4.908209314287179,-4.990058152230002,-4.484965901600792,-4.827068818323691,-4.884659886876155,-4.83458359235066"

	_, simulatedEpoch2 := GetCsvToSimulatedValuesMap(headers, epoch2)
	_, simulatedEpoch3 := GetCsvToSimulatedValuesMap(headers, epoch3)

	// EPOCH 3
	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0": {Value: simulatedEpoch3["inference_0"]},
		"worker1": {Value: simulatedEpoch3["inference_1"]},
		"worker2": {Value: simulatedEpoch3["inference_2"]},
		"worker3": {Value: simulatedEpoch3["inference_3"]},
		"worker4": {Value: simulatedEpoch3["inference_4"]},
	}

	forecastImpliedInferenceByWorker := map[string]*emissionstypes.Inference{
		"worker5": {Value: simulatedEpoch3["forecast_implied_inference_0"]},
	}

	epsilon := alloraMath.MustNewDecFromString("0.0001")
	pNorm := alloraMath.MustNewDecFromString("3")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	infererNetworkRegrets := map[string]inferencesynthesis.Regret{ // FROM EPOCH 2
		"worker0": simulatedEpoch2["inference_regret_worker_0_onein_0"],
		"worker1": simulatedEpoch2["inference_regret_worker_1_onein_0"],
		"worker2": simulatedEpoch2["inference_regret_worker_2_onein_0"],
		"worker3": simulatedEpoch2["inference_regret_worker_3_onein_0"],
		"worker4": simulatedEpoch2["inference_regret_worker_4_onein_0"],
	}

	forecasterNetworkRegrets := map[string]inferencesynthesis.Regret{ // FROM EPOCH 2
		"worker5": simulatedEpoch2["inference_regret_worker_5_onein_0"],
	}

	// FROM EPOCH 3
	expectedNetworkCombinedInferenceValue := simulatedEpoch3["network_naive_inference_onein_0"]

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

func (s *InferenceSynthesisTestSuite) TestCalcOneOutInferencesMultipleWorkers() {
	topicId := inferencesynthesis.TopicId(1)
	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0": {Value: alloraMath.MustNewDecFromString("-0.0514234892489971")},
		"worker1": {Value: alloraMath.MustNewDecFromString("-0.0316532211989242")},
		"worker2": {Value: alloraMath.MustNewDecFromString("-0.1018014248041400")},
	}
	forecastImpliedInferenceByWorker := map[string]*emissionstypes.Inference{
		"worker3": {Value: alloraMath.MustNewDecFromString("-0.0707517711518230")},
		"worker4": {Value: alloraMath.MustNewDecFromString("-0.0646463841210426")},
		"worker5": {Value: alloraMath.MustNewDecFromString("-0.0634099113416666")},
	}
	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: "worker3",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("0.00011708024633613200")},
					{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.013382222402411400")},
					{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("3.82471429104471e-05")},
				},
			},
			{
				Forecaster: "worker4",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("0.00011486217283808300")},
					{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.0060528036329761000")},
					{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("0.0005337255825785730")},
				},
			},
			{
				Forecaster: "worker5",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("0.001810780808278390")},
					{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.0018544539679880700")},
					{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("0.001251454152216520")},
				},
			},
		},
	}
	infererNetworkRegrets := map[string]inferencesynthesis.Regret{
		"worker0": alloraMath.MustNewDecFromString("0.6975029322458370"),
		"worker1": alloraMath.MustNewDecFromString("0.9101744424126180"),
		"worker2": alloraMath.MustNewDecFromString("0.9871536722074480"),
	}
	forecasterNetworkRegrets := map[string]inferencesynthesis.Regret{
		"worker3": alloraMath.MustNewDecFromString("0.8308330665491310"),
		"worker4": alloraMath.MustNewDecFromString("0.8396961220162480"),
		"worker5": alloraMath.MustNewDecFromString("0.8017696138115460"),
	}
	networkCombinedLoss := alloraMath.MustNewDecFromString("0.0156937658327922")
	epsilon := alloraMath.MustNewDecFromString("0.0001")
	fTolerance := alloraMath.MustNewDecFromString("0.01")
	pNorm := alloraMath.MustNewDecFromString("2.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	expectedOneOutInferences := []struct {
		Worker string
		Value  string
	}{
		{Worker: "worker0", Value: "-0.0711130346780"},
		{Worker: "worker1", Value: "-0.077954217717"},
		{Worker: "worker2", Value: "-0.0423024599518"},
	}
	expectedOneOutImpliedInferences := []struct {
		Worker string
		Value  string
	}{
		{Worker: "worker3", Value: "-0.06351714496"},
		{Worker: "worker4", Value: "-0.06471822091"},
		{Worker: "worker5", Value: "-0.06495348528"},
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

	// epoch 3 inference_x
	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0": {Value: alloraMath.MustNewDecFromString("-0.035995138925040600")},
		"worker1": {Value: alloraMath.MustNewDecFromString("-0.07333303938740420")},
		"worker2": {Value: alloraMath.MustNewDecFromString("-0.1495482917094790")},
		"worker3": {Value: alloraMath.MustNewDecFromString("-0.12952123274063800")},
		"worker4": {Value: alloraMath.MustNewDecFromString("-0.0703055329498285")},
	}

	// epoch 3 forecast_implied_inference_x
	forecastImpliedInferenceByWorker := map[string]*emissionstypes.Inference{
		"forecaster0": {Value: alloraMath.MustNewDecFromString("-0.1025675327315210")},
		"forecaster1": {Value: alloraMath.MustNewDecFromString("-0.07302318589259440")},
		"forecaster2": {Value: alloraMath.MustNewDecFromString("-0.07233832253513270")},
	}

	// epoch 3 forecasted_loss_x_for_y
	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: "forecaster0",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("-2.480767250656480")},
					{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("-3.5546685650440400")},
					{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("-4.6188184193555700")},
					{Inferer: "worker3", Value: alloraMath.MustNewDecFromString("-3.084052840898730")},
					{Inferer: "worker4", Value: alloraMath.MustNewDecFromString("-4.73003856038905")},
				},
			},
			{
				Forecaster: "forecaster1",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("-2.6366811669641100")},
					{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("-4.668814135864770")},
					{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("-1.901004446344670")},
					{Inferer: "worker3", Value: alloraMath.MustNewDecFromString("-1.7447621462061600")},
					{Inferer: "worker4", Value: alloraMath.MustNewDecFromString("-3.6984295084170300")},
				},
			},
			{
				Forecaster: "forecaster2",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("-2.2700889501915100")},
					{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("-3.5886903414115300")},
					{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("-2.119137360333620")},
					{Inferer: "worker3", Value: alloraMath.MustNewDecFromString("-2.1176515266976500")},
					{Inferer: "worker4", Value: alloraMath.MustNewDecFromString("-3.447694011689490")},
				},
			},
		},
	}
	// epoch 2 inference_regret_worker_x
	infererNetworkRegrets :=
		map[string]inferencesynthesis.Regret{
			"worker0": alloraMath.MustNewDecFromString("0.29240709744359700"),
			"worker1": alloraMath.MustNewDecFromString("0.41822210449254600"),
			"worker2": alloraMath.MustNewDecFromString("0.17663500756729100"),
			"worker3": alloraMath.MustNewDecFromString("0.4961746382998650"),
			"worker4": alloraMath.MustNewDecFromString("0.27996059439471200"),
		}

	// epoch 2 inference_regret_worker_5-7
	forecasterNetworkRegrets := map[string]inferencesynthesis.Regret{
		"forecaster0": alloraMath.MustNewDecFromString("0.8160663799969630"),
		"forecaster1": alloraMath.MustNewDecFromString("0.8234558968607070"),
		"forecaster2": alloraMath.MustNewDecFromString("0.8196673491058460"),
	}
	// epoch 2 network_loss
	networkCombinedLoss := alloraMath.MustNewDecFromString("-4.89349875041023")
	epsilon := alloraMath.MustNewDecFromString("0.0001")
	fTolerance := alloraMath.MustNewDecFromString("0.01")
	pNorm := alloraMath.MustNewDecFromString("3")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	// epoch 3 network_inference_oneout_x ?
	expectedOneOutInferences := []struct {
		Worker string
		Value  string
	}{
		{Worker: "worker0", Value: "-0.09193875537568730"},
		{Worker: "worker1", Value: "-0.08725723597921200"},
		{Worker: "worker2", Value: "-0.07611975173450570"},
		{Worker: "worker3", Value: "-0.07876098982982720"},
		{Worker: "worker4", Value: "-0.09549195980225840"},
	}

	// epoch 3 network_inference_oneout_5-7
	expectedOneOutImpliedInferences := []struct {
		Worker string
		Value  string
	}{
		{Worker: "forecaster0", Value: "-0.0831086124651002"},
		{Worker: "forecaster1", Value: "-0.0880928742443665"},
		{Worker: "forecaster2", Value: "-0.08821296390705380"},
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
