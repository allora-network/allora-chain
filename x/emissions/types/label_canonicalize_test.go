package types_test

import (
	"strings"
	"testing"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

// testMaxBytes mirrors the module-initial 64-byte cap so tests do not
// depend on what the operator configured via
// Params.MaxCanonicalLabelByteLength; every call site below uses this
// constant to keep the focus on the label canonicalizer semantics rather
// than the cap value itself. Keep in sync with DefaultParams.
const testMaxBytes uint64 = 64

// TestCanonicalLabelName_Accepts exercises the acceptance cases for the
// default labelCaseSensitive=false mode (the worker-path default). Every
// input must canonicalize to the expected form, without error, and the
// result must be idempotent under a second pass through CanonicalLabelName.
func TestCanonicalLabelName_Accepts(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain ascii", in: "y", want: "y"},
		{name: "trims leading/trailing spaces", in: "   label  ", want: "label"},
		{name: "trims tabs and newlines", in: "\t\nfoo\r\n", want: "foo"},
		{name: "preserves internal spaces", in: "two words", want: "two words"},
		{name: "digits", in: "abc123", want: "abc123"},
		{name: "hyphen", in: "foo-bar", want: "foo-bar"},
		{name: "underscore", in: "foo_bar", want: "foo_bar"},
		{name: "hierarchy slash", in: "a/b/c", want: "a/b/c"},
		{name: "hierarchy dot", in: "a.b.c", want: "a.b.c"},
		{name: "lowercases uppercase", in: "Cat", want: "cat"},
		{name: "lowercases all caps", in: "CAT", want: "cat"},
		{name: "mixed separators", in: "Group/Sub-Topic_v1", want: "group/sub-topic_v1"},
		{
			name: "max length boundary exact",
			in:   strings.Repeat("a", int(testMaxBytes)),
			want: strings.Repeat("a", int(testMaxBytes)),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := types.CanonicalLabelName(tc.in, testMaxBytes, false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("CanonicalLabelName(%q, %d, false) = %q, want %q", tc.in, testMaxBytes, got, tc.want)
			}
			again, err := types.CanonicalLabelName(got, testMaxBytes, false)
			if err != nil {
				t.Fatalf("idempotence pass errored: %v", err)
			}
			if again != got {
				t.Fatalf("CanonicalLabelName not idempotent: %q -> %q", got, again)
			}
		})
	}
}

// TestCanonicalLabelName_CaseSensitive verifies that
// labelCaseSensitive=true preserves uppercase ASCII letters and that the
// function remains idempotent in that mode.
func TestCanonicalLabelName_CaseSensitive(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "preserves uppercase", in: "Cat", want: "Cat"},
		{name: "preserves all caps", in: "CAT", want: "CAT"},
		{name: "still trims", in: "  Cat  ", want: "Cat"},
		{name: "mixed separators", in: "Group/Sub-Topic_v1", want: "Group/Sub-Topic_v1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := types.CanonicalLabelName(tc.in, testMaxBytes, true)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("CanonicalLabelName(%q, %d, true) = %q, want %q", tc.in, testMaxBytes, got, tc.want)
			}
			again, err := types.CanonicalLabelName(got, testMaxBytes, true)
			if err != nil {
				t.Fatalf("idempotence pass errored: %v", err)
			}
			if again != got {
				t.Fatalf("CanonicalLabelName not idempotent: %q -> %q", got, again)
			}
		})
	}
}

// TestCanonicalLabelName_Rejects exercises the rejection paths. Every case
// must return a non-nil error; no case should return a partial result.
func TestCanonicalLabelName_Rejects(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name               string
		in                 string
		labelCaseSensitive bool
	}{
		{name: "empty", in: "", labelCaseSensitive: false},
		{name: "whitespace only", in: "   \t\n", labelCaseSensitive: false},
		{name: "zero width space", in: "fo\u200bo", labelCaseSensitive: false},
		{name: "left-to-right mark", in: "\u200efoo", labelCaseSensitive: false},
		{name: "nul byte", in: "fo\x00o", labelCaseSensitive: false},
		{name: "bell control", in: "\afoo", labelCaseSensitive: false},
		{name: "exclamation", in: "bad!", labelCaseSensitive: false},
		{name: "at sign", in: "bad@good", labelCaseSensitive: false},
		{name: "hash", in: "bad#good", labelCaseSensitive: false},
		{name: "unicode letter", in: "é", labelCaseSensitive: false},
		{name: "unicode kelvin sign folds to ascii k", in: "\u212A", labelCaseSensitive: false},
		{name: "emoji", in: "good 🚀", labelCaseSensitive: false},
		{name: "nfd e-acute", in: "e\u0301", labelCaseSensitive: false},
		{name: "nfc e-acute", in: "\u00e9", labelCaseSensitive: false},
		{name: "uppercase when case sensitive false", in: "Á", labelCaseSensitive: false},
		{
			name:               "one byte over",
			in:                 strings.Repeat("a", int(testMaxBytes)+1),
			labelCaseSensitive: false,
		},
		{name: "invalid utf8", in: "\xff\xfe", labelCaseSensitive: false},
		{name: "uppercase retained when case sensitive true rejects charset", in: "BAD!", labelCaseSensitive: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := types.CanonicalLabelName(tc.in, testMaxBytes, tc.labelCaseSensitive)
			if err == nil {
				t.Fatalf("expected error for %q (cs=%v), got %q", tc.in, tc.labelCaseSensitive, got)
			}
			if got != "" {
				t.Fatalf("expected empty result on error, got %q", got)
			}
		})
	}
}

// TestCanonicalLabelName_RejectsReplacementCharacter documents that a literal
// Unicode replacement character is valid UTF-8 but outside the ASCII-only label
// charset, so it must be rejected by the charset check rather than the UTF-8
// validity gate.
func TestCanonicalLabelName_RejectsReplacementCharacter(t *testing.T) {
	t.Parallel()
	got, err := types.CanonicalLabelName("\uFFFD", testMaxBytes, false)
	if err == nil {
		t.Fatalf("expected error for literal replacement character, got %q", got)
	}
	if got != "" {
		t.Fatalf("expected empty result on error, got %q", got)
	}
	if !strings.Contains(err.Error(), "disallowed character: U+FFFD") {
		t.Fatalf("expected disallowed-character error for U+FFFD, got %v", err)
	}
}

// TestCanonicalLabelName_RejectsZeroMaxBytes covers the defensive check for
// a zero cap. Params.Validate rejects zero so this path is not reachable
// through normal load, but the canonicalizer must still reject it cleanly.
func TestCanonicalLabelName_RejectsZeroMaxBytes(t *testing.T) {
	t.Parallel()
	got, err := types.CanonicalLabelName("y", 0, false)
	if err == nil {
		t.Fatalf("expected error for zero max bytes, got %q", got)
	}
	if got != "" {
		t.Fatalf("expected empty result on error, got %q", got)
	}
}

// TestCanonicalizeLabelList_DedupesPostCanon covers the case where two input
// labels are byte-different pre-canonicalization but byte-equal after: the
// helper must flag the duplicate when rejectDuplicates is true, and must
// pass silently when it is false (with both canonical forms returned).
func TestCanonicalizeLabelList_DedupesPostCanon(t *testing.T) {
	t.Parallel()
	in := []string{"Cat", " cat "}
	if _, err := types.CanonicalizeLabelList(in, true, testMaxBytes, false); err == nil {
		t.Fatalf("expected duplicate-post-canon error, got nil")
	}
	got, err := types.CanonicalizeLabelList(in, false, testMaxBytes, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != "cat" || got[1] != "cat" {
		t.Fatalf("unexpected result: %q", got)
	}

	// Under labelCaseSensitive=true the two entries are distinct, so no
	// dedupe.
	cs, err := types.CanonicalizeLabelList(in, true, testMaxBytes, true)
	if err != nil {
		t.Fatalf("unexpected error under labelCaseSensitive: %v", err)
	}
	if len(cs) != 2 || cs[0] != "Cat" || cs[1] != "cat" {
		t.Fatalf("unexpected case-sensitive result: %q", cs)
	}
}

// TestCanonicalizeLabelList_RejectsInner ensures the index-reporting
// behaviour: if a middle-of-list entry fails canonicalization, the error
// must reference its index and the overall result must be empty.
func TestCanonicalizeLabelList_RejectsInner(t *testing.T) {
	t.Parallel()
	in := []string{"ok", "", "also_ok"}
	got, err := types.CanonicalizeLabelList(in, true, testMaxBytes, false)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "index 1") {
		t.Fatalf("expected error to include failing index 1, got %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil result, got %v", got)
	}
}

// TestCanonicalLabelSet_NilEmpty documents the nil-vs-empty semantics relied
// on by the msgserver whitelist check (empty whitelist means "unrestricted"
// and must build a nil set, which callers interpret as "skip").
func TestCanonicalLabelSet_NilEmpty(t *testing.T) {
	t.Parallel()
	if got := types.CanonicalLabelSet(nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
	if got := types.CanonicalLabelSet([]string{}); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
	got := types.CanonicalLabelSet([]string{"a", "b"})
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
	if _, ok := got["a"]; !ok {
		t.Fatalf("missing key a")
	}
}
