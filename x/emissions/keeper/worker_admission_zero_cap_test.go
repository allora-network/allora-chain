package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// An effective cap of 0 is reachable only when the global floor is 0.
// This test covers the case where the global floor is 0 and the topic cap is 0,
// and how it affects the admission of new inferers.
//
// Nobody is admitted into an empty active set; an already over-full set
// still allows the usual net-zero swap.
func (s *KeeperTestSuite) TestPlanInferenceAdmissionZeroCap() {
	s.Require().Equal(uint64(0), types.EffectiveMaxTopInferersToReward(0, 0, 32))

	type tc struct {
		name     string
		setup    func(ctx sdk.Context, topicId uint64, inferer string)
		wantKind keeper.InferenceAdmissionKind
	}
	cases := []tc{
		{
			name:     "first_submission_empty_set_not_admitted",
			setup:    func(sdk.Context, uint64, string) {},
			wantKind: keeper.InferenceAdmissionNotAdmitted,
		},
		{
			name: "experienced_worker_empty_set_not_admitted",
			setup: func(ctx sdk.Context, topicId uint64, inferer string) {
				score := types.Score{TopicId: topicId, BlockHeight: 1, Address: inferer, Score: alloraMath.NewDecFromInt64(50)}
				s.Require().NoError(s.ScoresKeeper().SetInfererScoreEma(ctx, topicId, inferer, score))
			},
			wantKind: keeper.InferenceAdmissionNotAdmitted,
		},
		{
			name: "over_full_set_and_higher_score_plans_eviction",
			setup: func(ctx sdk.Context, topicId uint64, inferer string) {
				active := s.AddrsStr(1)
				activeScore := types.Score{TopicId: topicId, BlockHeight: 1, Address: active, Score: alloraMath.NewDecFromInt64(10)}
				candidateScore := types.Score{TopicId: topicId, BlockHeight: 1, Address: inferer, Score: alloraMath.NewDecFromInt64(100)}
				s.Require().NoError(s.ScoresKeeper().SetInfererScoreEma(ctx, topicId, active, activeScore))
				s.Require().NoError(s.ScoresKeeper().SetInfererScoreEma(ctx, topicId, inferer, candidateScore))
				s.Require().NoError(s.ScoresKeeper().SetLowestInfererScoreEma(ctx, topicId, activeScore))
				s.Require().NoError(s.WorkerKeeper().AddActiveInferer(ctx, topicId, active))
			},
			wantKind: keeper.InferenceAdmissionEvictLowest,
		},
	}

	for _, c := range cases {
		s.Run(c.name, func() {
			s.SetupTest()
			ctx := s.Ctx()
			topicId := s.CreateTopic()
			topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
			s.Require().NoError(err)
			inferer := s.AddrsStr(0)
			c.setup(ctx, topicId, inferer)

			plan, err := s.WorkerKeeper().PlanInferenceAdmission(ctx, topic, types.BlockHeight(10), inferer, 0)
			s.Require().NoError(err)
			s.Require().Equal(c.wantKind, plan.Kind)
		})
	}
}
