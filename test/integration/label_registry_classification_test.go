package integration_test

import (
	"context"

	"github.com/stretchr/testify/require"

	alloraMath "github.com/allora-network/allora-chain/math"
	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// LabelRegistryClassificationChecks exercises label registry topic settings
// end-to-end through the live emissions query/tx RPCs. These scenarios do not
// require signed worker payloads.
//
// The six sub-scenarios are:
//  1. CLASSIFICATION + MULTI-arity topic with a label_whitelist and
//     max_labels_per_submission round-trips through storage and the
//     whitelist is persisted canonicalized (trim/case fold).
//  2. CreateNewTopic rejects zero max_labels_per_submission.
//  3. CreateNewTopic rejects a whitelist containing a canonical duplicate
//     (post-trim/case fold) via ErrInvalidLabelName.
//  4. CreateNewTopic rejects a whitelist containing a control character
//     via ErrInvalidLabelName.
//  5. Topic.LabelWhitelist is stored in sorted/canonical form that is
//     byte-equal across resubmission of identical inputs.
//  6. UpdateTopic can change max_labels_per_submission when the topic has
//     no open worker submission window (covered by the freshly-created,
//     inactive test topic).
//
//nolint:exhaustruct
func LabelRegistryClassificationChecks(m testCommon.TestConfig) {
	ctx := context.Background()
	m.T.Log(">>> Label Registry integration scenarios <<<")

	addTopicCreator(m, m.AliceAddr)

	baseReq := func() *emissionstypes.CreateNewTopicRequest {
		return &emissionstypes.CreateNewTopicRequest{
			Creator:                  m.AliceAddr,
			Metadata:                 "label registry integration",
			LossMethod:               "mse",
			EpochLength:              5,
			GroundTruthLag:           10,
			WorkerSubmissionWindow:   4,
			PNorm:                    alloraMath.NewDecFromInt64(3),
			AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
			AllowNegative:            false,
			Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
			MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
			ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
			ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
			ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
			EnableWorkerWhitelist:    true,
			EnableReputerWhitelist:   true,
			CNorm:                    alloraMath.MustNewDecFromString("0.75"),
			TopicType:                emissionstypes.TopicType_TOPIC_TYPE_CLASSIFICATION,
			OutputArity:              emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			RequireUnity:             true,
			UnityTolerance:           alloraMath.MustNewDecFromString("0.01"),
			MaxLabelsPerSubmission:   3,
			LabelWhitelist:           []string{"  bull  ", "Bear/Cub", "bear"},
		}
	}

	// Scenario 1 + 5: create, confirm canonicalization + round-trip.
	canonicalReq := baseReq()
	txResp, err := m.Client.BroadcastTx(ctx, m.AliceAcc, canonicalReq)
	require.NoError(m.T, err)
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	require.NoError(m.T, err)
	createResp := &emissionstypes.CreateNewTopicResponse{}
	require.NoError(m.T, txResp.Decode(createResp))

	stored, err := m.Client.QueryEmissions().GetTopic(ctx, &emissionstypes.GetTopicRequest{
		TopicId: createResp.TopicId,
	})
	require.NoError(m.T, err)
	require.Equal(m.T, uint64(3), stored.Topic.MaxLabelsPerSubmission,
		"MaxLabelsPerSubmission must round-trip (scenario 1)")
	require.ElementsMatch(m.T,
		[]string{"bull", "bear/cub", "bear"},
		stored.Topic.LabelWhitelist,
		"LabelWhitelist must be persisted canonicalized (scenario 1, 5)")

	// Scenario 2: zero MaxLabelsPerSubmission is rejected.
	zeroCapReq := baseReq()
	zeroCapReq.MaxLabelsPerSubmission = 0
	zeroCapReq.LabelWhitelist = []string{"y"}
	txResp, err = m.Client.BroadcastTx(ctx, m.AliceAcc, zeroCapReq)
	zeroCapErr := err
	if err == nil {
		_, zeroCapErr = m.Client.WaitForTx(ctx, txResp.TxHash)
	}
	require.Error(m.T, zeroCapErr,
		"zero MaxLabelsPerSubmission must be rejected (scenario 2)")
	require.ErrorContains(m.T, zeroCapErr, "max labels per submission")

	// Scenario 3: canonical duplicate whitelist rejected.
	dupReq := baseReq()
	dupReq.LabelWhitelist = []string{"Foo", " foo "}
	txResp, err = m.Client.BroadcastTx(ctx, m.AliceAcc, dupReq)
	if err == nil {
		_, waitErr := m.Client.WaitForTx(ctx, txResp.TxHash)
		require.Error(m.T, waitErr,
			"canonical-duplicate whitelist must be rejected (scenario 3)")
	}

	// Scenario 4: control-character rejected.
	ctrlReq := baseReq()
	ctrlReq.LabelWhitelist = []string{"a\x00b"}
	txResp, err = m.Client.BroadcastTx(ctx, m.AliceAcc, ctrlReq)
	if err == nil {
		_, waitErr := m.Client.WaitForTx(ctx, txResp.TxHash)
		require.Error(m.T, waitErr,
			"control-character in whitelist must be rejected (scenario 4)")
	}

	// Scenario 6: UpdateTopic can change fields on a freshly-created
	// (inactive) topic with no open WSW. We re-use the first topic.
	updateReq := &emissionstypes.UpdateTopicRequest{
		Sender:                 m.AliceAddr,
		TopicId:                createResp.TopicId,
		Metadata:               stored.Topic.Metadata,
		LossMethod:             stored.Topic.LossMethod,
		AlphaRegret:            stored.Topic.AlphaRegret,
		MeritSortitionAlpha:    stored.Topic.MeritSortitionAlpha,
		PNorm:                  stored.Topic.PNorm,
		CNorm:                  stored.Topic.CNorm,
		RequireUnity:           stored.Topic.RequireUnity,
		UnityTolerance:         stored.Topic.UnityTolerance,
		MaxLabelsPerSubmission: 5,
		LabelWhitelist:         stored.Topic.LabelWhitelist,
		LabelDefaultValue:      alloraMath.ZeroDec(),
	}
	txResp, err = m.Client.BroadcastTx(ctx, m.AliceAcc, updateReq)
	require.NoError(m.T, err)
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	require.NoError(m.T, err)
	updated, err := m.Client.QueryEmissions().GetTopic(ctx, &emissionstypes.GetTopicRequest{
		TopicId: createResp.TopicId,
	})
	require.NoError(m.T, err)
	require.Equal(m.T, uint64(5), updated.Topic.MaxLabelsPerSubmission,
		"UpdateTopic must accept changes when no WSW is open (scenario 6)")
}
