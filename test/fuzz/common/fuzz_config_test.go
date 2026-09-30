package fuzzcommon

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeInitialSetupFillsDefaultNumTopics(t *testing.T) {
	normalized := NormalizeInitialSetup(InitialSetup{NumAdminWhitelist: 1}) //nolint:exhaustruct // a config file omitting fields
	require.Equal(t, GetHardCodedInitialSetup().NumTopics, normalized.NumTopics)
	require.Equal(t, 1, normalized.NumAdminWhitelist)

	kept := NormalizeInitialSetup(InitialSetup{NumTopics: 5}) //nolint:exhaustruct // only the field under test
	require.Equal(t, 5, kept.NumTopics)
}

func TestInitialSetupJsonFields(t *testing.T) {
	raw := `{"numAdminWhitelist":2,"numGlobalWhitelist":4,"numTopicCreatorWhitelist":2,
		"numTopics":3,"topicsInSameBlock":true,"unevenTopicWeights":true,"expectEpochEndRefusal":true}`
	var setup InitialSetup
	require.NoError(t, json.Unmarshal([]byte(raw), &setup))
	require.Equal(t, 3, setup.NumTopics)
	require.True(t, setup.TopicsInSameBlock)
	require.True(t, setup.UnevenTopicWeights)
	require.True(t, setup.ExpectEpochEndRefusal)
}

func TestApplyInitialSetupEnvOverrides(t *testing.T) {
	t.Setenv("NUM_SETUP_TOPICS", "4")
	t.Setenv("TOPICS_IN_SAME_BLOCK", "true")
	t.Setenv("UNEVEN_TOPIC_WEIGHTS", "1")
	t.Setenv("EXPECT_EPOCH_END_REFUSAL", "false")
	setup := ApplyInitialSetupEnvOverrides(t, GetHardCodedInitialSetup())
	require.Equal(t, 4, setup.NumTopics)
	require.True(t, setup.TopicsInSameBlock)
	require.True(t, setup.UnevenTopicWeights)
	require.False(t, setup.ExpectEpochEndRefusal)
	// untouched fields keep the hardcoded values
	require.Equal(t, GetHardCodedInitialSetup().NumAdminWhitelist, setup.NumAdminWhitelist)
}

func TestInitialSetupValidate(t *testing.T) {
	tests := []struct {
		name    string
		setup   InitialSetup
		wantErr string
	}{
		{name: "defaults", setup: GetHardCodedInitialSetup(), wantErr: ""},
		{name: "zero topics", setup: InitialSetup{NumTopics: 0}, wantErr: "numTopics"}, //nolint:exhaustruct // only the field under test
		{
			name:    "refusal check with one topic",
			setup:   InitialSetup{NumTopics: 1, TopicsInSameBlock: true, ExpectEpochEndRefusal: true}, //nolint:exhaustruct // only the fields under test
			wantErr: "at least 2 setup topics",
		},
		{
			name:    "refusal check without same block",
			setup:   InitialSetup{NumTopics: 3, TopicsInSameBlock: false, ExpectEpochEndRefusal: true}, //nolint:exhaustruct // only the fields under test
			wantErr: "topicsInSameBlock",
		},
		{
			name:    "collision config",
			setup:   InitialSetup{NumTopics: 3, TopicsInSameBlock: true, UnevenTopicWeights: true, ExpectEpochEndRefusal: true}, //nolint:exhaustruct // only the fields under test
			wantErr: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.setup.Validate()
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}
