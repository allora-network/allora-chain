package keeper_test

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *KeeperTestSuite) TestInferenceValuesFromProto() {
	s.SetupTest()

	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	creator := s.Addrs(0)
	topicId := s.SetupTopic(creator)
	nonce := int64(1)

	w1 := s.AddrsStr(0)
	w2 := s.AddrsStr(1)

	mustDec := func(x string) alloraMath.Dec { return alloraMath.MustNewDecFromString(x) }

	setArity := func(arity types.TopicOutputArity) types.Topic {
		topic, err := k.GetTopic(ctx, topicId)
		s.Require().NoError(err)
		topic.OutputArity = arity
		topic.RequireUnity = false
		topic.UnityTolerance = alloraMath.ZeroDec()
		err = k.SetTopic(ctx, topicId, topic)
		s.Require().NoError(err)

		updated, err := k.GetTopic(ctx, topicId)
		s.Require().NoError(err)
		return updated
	}

	registerLabels := func(labels ...string) {
		for _, l := range labels {
			_, err := k.RegisterEpochLabel(ctx, topicId, nonce, l)
			s.Require().NoError(err)
		}
	}

	type tc struct {
		name      string
		arity     types.TopicOutputArity
		setup     func()
		inf       func() *types.Inference
		wantErrIs error
		wantVals  []string
	}

	cases := []tc{
		{
			name:      "nil_inference_rejected",
			arity:     types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			setup:     nil,
			inf:       func() *types.Inference { return nil },
			wantErrIs: sdkerrors.ErrInvalidRequest,
		},
		{
			name:  "SINGLE_scalar_only_ok",
			arity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			inf: func() *types.Inference {
				return &types.Inference{
					TopicId:     topicId,
					BlockHeight: nonce,
					Inferer:     w1,
					Value:       mustDec("42"),
					Values:      nil,
				}
			},
			wantVals: []string{"42"},
		},
		{
			name:  "SINGLE_values_len1_equal_scalar_ok",
			arity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			inf: func() *types.Inference {
				return &types.Inference{
					TopicId:     topicId,
					BlockHeight: nonce,
					Inferer:     w1,
					Value:       mustDec("7"),
					Values:      []alloraMath.Dec{mustDec("7")},
				}
			},
			wantVals: []string{"7"},
		},
		{
			name:  "SINGLE_values_len1_mismatch_rejected",
			arity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			inf: func() *types.Inference {
				return &types.Inference{
					TopicId:     topicId,
					BlockHeight: nonce,
					Inferer:     w1,
					Value:       mustDec("1"),
					Values:      []alloraMath.Dec{mustDec("2")},
				}
			},
			wantErrIs: sdkerrors.ErrInvalidRequest,
		},
		{
			name:  "SINGLE_values_len_gt_1_rejected",
			arity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			inf: func() *types.Inference {
				return &types.Inference{
					TopicId:     topicId,
					BlockHeight: nonce,
					Inferer:     w1,
					Value:       mustDec("1"),
					Values:      []alloraMath.Dec{mustDec("1"), mustDec("2")},
				}
			},
			wantErrIs: sdkerrors.ErrInvalidRequest,
		},
		{
			name:  "MULTI_empty_registry_rejected",
			arity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			setup: func() {
				// do NOT register labels => regLen=0
			},
			inf: func() *types.Inference {
				return &types.Inference{
					TopicId:     topicId,
					BlockHeight: nonce,
					Inferer:     w1,
					Value:       alloraMath.ZeroDec(),
					Values:      []alloraMath.Dec{mustDec("1")},
				}
			},
			wantErrIs: sdkerrors.ErrLogic,
		},
		{
			name:  "MULTI_empty_values_rejected",
			arity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			setup: func() {
				registerLabels("a")
			},
			inf: func() *types.Inference {
				return &types.Inference{
					TopicId:     topicId,
					BlockHeight: nonce,
					Inferer:     w1,
					Value:       alloraMath.ZeroDec(),
					Values:      nil,
				}
			},
			wantErrIs: sdkerrors.ErrInvalidRequest,
		},
		{
			name:  "MULTI_values_len_gt_registry_rejected",
			arity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			setup: func() {
				registerLabels("a", "b")
			},
			inf: func() *types.Inference {
				return &types.Inference{
					TopicId:     topicId,
					BlockHeight: nonce,
					Inferer:     w1,
					Value:       alloraMath.ZeroDec(),
					Values:      []alloraMath.Dec{mustDec("1"), mustDec("2"), mustDec("3")},
				}
			},
			wantErrIs: sdkerrors.ErrLogic,
		},
		{
			name:  "MULTI_exact_len_ok_no_padding",
			arity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			setup: func() {
				registerLabels("a", "b", "c")
			},
			inf: func() *types.Inference {
				return &types.Inference{
					TopicId:     topicId,
					BlockHeight: nonce,
					Inferer:     w2,
					Value:       alloraMath.ZeroDec(),
					Values:      []alloraMath.Dec{mustDec("10"), mustDec("20"), mustDec("30")},
				}
			},
			wantVals: []string{"10", "20", "30"},
		},
		{
			name:  "MULTI_shorter_len_pads_to_registry_len",
			arity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			setup: func() {
				registerLabels("a", "b", "c", "d", "e")
			},
			inf: func() *types.Inference {
				return &types.Inference{
					TopicId:     topicId,
					BlockHeight: nonce,
					Inferer:     w1,
					Value:       alloraMath.ZeroDec(),
					Values:      []alloraMath.Dec{mustDec("9"), mustDec("8")}, // => [9,8,0,0,0]
				}
			},
			wantVals: []string{"9", "8", "0", "0", "0"},
		},
		{
			name:  "MULTI_rejects_invalid_value_in_values",
			arity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			setup: func() {
				registerLabels("a", "b", "c")
			},
			inf: func() *types.Inference {
				return &types.Inference{
					TopicId:     topicId,
					BlockHeight: nonce,
					Inferer:     w1,
					Value:       alloraMath.ZeroDec(),
					Values:      []alloraMath.Dec{mustDec("1"), alloraMath.NewNaN(), mustDec("3")},
				}
			},
			wantErrIs: sdkerrors.ErrInvalidRequest,
		},
	}

	for _, c := range cases {
		s.Run(c.name, func() {
			s.SetupTest()

			nonce = int64(1)

			registerLabels = func(labels ...string) {
				for _, l := range labels {
					_, err := k.RegisterEpochLabel(ctx, topicId, nonce, l)
					s.Require().NoError(err)
				}
			}

			topic := setArity(c.arity)

			if c.setup != nil {
				c.setup()
			}

			inf := (*types.Inference)(nil)
			if c.inf != nil {
				inf = c.inf()
			}

			reg, err := k.GetEpochLabelRegistry(ctx, topicId, nonce)
			s.Require().NoError(err)

			got, err := keeper.InferenceValuesFromProto(topic, reg, inf)

			if c.wantErrIs != nil {
				s.Require().ErrorIs(err, c.wantErrIs)
				return
			}
			s.Require().NoError(err)

			s.Require().Equal(len(c.wantVals), len(got))
			for i := range c.wantVals {
				s.Require().Equal(c.wantVals[i], got[i].String())
			}
		})
	}
}
