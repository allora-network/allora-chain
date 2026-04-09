package actorutils_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	alloraMath "github.com/allora-network/allora-chain/math"
	actorutils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

type ActorUtilsTestSuite struct {
	testutil.TestSuite
}

func TestModuleTestSuite(t *testing.T) {
	suite.Run(t, &ActorUtilsTestSuite{
		testutil.NewTestSuite("actor_utils_losses"),
	})
}

func (a *ActorUtilsTestSuite) TestFilterUnacceptedWorkersFromReputerValueBundle() {
	workerNonce := emissionstypes.Nonce{
		BlockHeight: 1,
	}

	inferer1 := a.AddrsStr(0)
	inferer2 := a.AddrsStr(1)
	inferer4 := a.AddrsStr(3)
	inferer5 := a.AddrsStr(4)

	forecaster1 := a.AddrsStr(5)
	forecaster2 := a.AddrsStr(6)
	forecaster3 := a.AddrsStr(7)
	forecaster4 := a.AddrsStr(8)
	forecaster5 := a.AddrsStr(9)

	infererLossBundle := emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{
				TopicId:     1,
				BlockHeight: 1,
				Inferer:     inferer1,
				Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(1)},
			},
			{
				TopicId:     1,
				BlockHeight: 1,
				Inferer:     inferer2,
				Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(2)},
			},
			{
				TopicId:     1,
				BlockHeight: 1,
				Inferer:     inferer4,
				Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(3)},
			},
		},
	}
	err := a.WorkerKeeper().InsertActiveInferences(a.Ctx(), 1, workerNonce.BlockHeight, infererLossBundle)
	a.Require().NoError(err)

	forecasterLossBundle := emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				TopicId:     1,
				BlockHeight: 1,
				Forecaster:  forecaster1,
				ForecastElements: []*emissionstypes.ForecastElement{
					{
						Inferer: inferer1,
						Value:   alloraMath.NewDecFromInt64(1),
					},
					{
						Inferer: inferer2,
						Value:   alloraMath.NewDecFromInt64(2),
					},
				},
			},
			{
				TopicId:     1,
				BlockHeight: 1,
				Forecaster:  forecaster2,
				ForecastElements: []*emissionstypes.ForecastElement{
					{
						Inferer: inferer1,
						Value:   alloraMath.NewDecFromInt64(3),
					},
				},
			},
		},
	}
	err = a.WorkerKeeper().InsertActiveForecasts(a.Ctx(), 1, workerNonce.BlockHeight, forecasterLossBundle)
	a.Require().NoError(err)

	// Prepare a sample ReputerValueBundle
	valueBundle := &emissionstypes.ValueBundle{
		TopicId: 1,
		ReputerRequestNonce: &emissionstypes.ReputerRequestNonce{
			ReputerNonce: &workerNonce,
		},
		Reputer:       a.AddrsStr(40),
		ExtraData:     nil,
		CombinedValue: alloraMath.NewDecFromInt64(1000),
		InfererValues: []*emissionstypes.WorkerAttributedValue{
			{Worker: inferer1, Value: alloraMath.NewDecFromInt64(100)},
			{Worker: inferer2, Value: alloraMath.NewDecFromInt64(100)},
			{Worker: inferer5, Value: alloraMath.NewDecFromInt64(200)}, // Should be filtered out
		},
		ForecasterValues: []*emissionstypes.WorkerAttributedValue{
			{Worker: forecaster1, Value: alloraMath.NewDecFromInt64(300)},
			{Worker: forecaster3, Value: alloraMath.NewDecFromInt64(400)}, // Should be filtered out
		},
		NaiveValue: alloraMath.NewDecFromInt64(1000),
		OneOutInfererValues: []*emissionstypes.WithheldWorkerAttributedValue{
			{Worker: inferer1, Value: alloraMath.NewDecFromInt64(500)},
			{Worker: inferer5, Value: alloraMath.NewDecFromInt64(600)}, // Should be filtered out
		},
		OneOutForecasterValues: []*emissionstypes.WithheldWorkerAttributedValue{
			{Worker: forecaster1, Value: alloraMath.NewDecFromInt64(700)},
			{Worker: forecaster4, Value: alloraMath.NewDecFromInt64(800)}, // Should be filtered out
		},
		OneOutInfererForecasterValues: []*emissionstypes.OneOutInfererForecasterValues{
			{
				Forecaster: forecaster1,
				OneOutInfererValues: []*emissionstypes.WithheldWorkerAttributedValue{
					{Worker: inferer1, Value: alloraMath.NewDecFromInt64(900)},
					{Worker: inferer5, Value: alloraMath.NewDecFromInt64(1000)}, // Should be filtered out
				},
			},
			{
				Forecaster: forecaster3, // Should be filtered out
				OneOutInfererValues: []*emissionstypes.WithheldWorkerAttributedValue{
					{Worker: inferer1, Value: alloraMath.NewDecFromInt64(1100)},
				},
			},
		},
		OneInForecasterValues: []*emissionstypes.WorkerAttributedValue{
			{Worker: forecaster1, Value: alloraMath.NewDecFromInt64(1200)},
			{Worker: forecaster5, Value: alloraMath.NewDecFromInt64(1300)}, // Should be filtered out
		},
	}

	acceptedBundle, err := actorutils.FilterUnacceptedWorkersFromReputerValueBundle(a.EmissionsKeeper(), a.Ctx(), 1, emissionstypes.ReputerRequestNonce{ReputerNonce: &workerNonce}, valueBundle)
	a.Require().NoError(err)

	// Validate the bundle
	a.Require().Len(acceptedBundle.InfererValues, 2)
	a.Require().Len(acceptedBundle.ForecasterValues, 1)
	a.Require().Len(acceptedBundle.OneOutInfererValues, 1)
	a.Require().Len(acceptedBundle.OneOutForecasterValues, 1)
	a.Require().Len(acceptedBundle.OneOutInfererForecasterValues, 1)
	a.Require().Len(acceptedBundle.OneOutInfererForecasterValues[0].OneOutInfererValues, 1)
	a.Require().Len(acceptedBundle.OneInForecasterValues, 1)
}
