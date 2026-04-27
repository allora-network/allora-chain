package types_test

import (
	"strings"
	"testing"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

// TestCanonicalLabelName_Accepts exercises the acceptance cases: the
// canonical form is byte-equal to the expected, no error is returned, and
// the result is idempotent under a second pass through CanonicalLabelName.
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
		{name: "nfc composed stays composed", in: "\u00e9", want: "\u00e9"},
		{name: "nfd gets normalized to nfc", in: "e\u0301", want: "\u00e9"},
		{name: "nfc after trim", in: "  e\u0301  ", want: "\u00e9"},
		{
			name: "max length boundary exact",
			in:   strings.Repeat("a", types.MaxCanonicalLabelByteLength),
			want: strings.Repeat("a", types.MaxCanonicalLabelByteLength),
		},
		{
			name: "multibyte rune under byte cap",
			in:   strings.Repeat("é", 32),
			want: strings.Repeat("é", 32),
		},
		{name: "replacement character ok", in: "valid \uFFFD", want: "valid \uFFFD"},
		{name: "emoji ok", in: "good 🚀", want: "good 🚀"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := types.CanonicalLabelName(tc.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("CanonicalLabelName(%q) = %q, want %q", tc.in, got, tc.want)
			}
			// Idempotence: running the canonicalizer again on the canonical
			// form must be a no-op. This invariant is what the rest of the
			// pipeline (map keys, sort.Strings) relies on.
			again, err := types.CanonicalLabelName(got)
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
	// Cross the byte cap by one two-byte rune to exercise the boundary.
	overlongMultibyte := strings.Repeat("é", types.MaxCanonicalLabelByteLength/len("é")+1)
	cases := []struct {
		name string
		in   string
	}{
		{name: "empty", in: ""},
		{name: "whitespace only", in: "   \t\n"},
		{name: "zero width space", in: "fo\u200bo"},
		{name: "left-to-right mark", in: "\u200efoo"},
		{name: "nul byte", in: "fo\x00o"},
		{name: "bell control", in: "\afoo"},
		{
			name: "one byte over",
			in:   strings.Repeat("a", types.MaxCanonicalLabelByteLength+1),
		},
		{name: "multibyte over", in: overlongMultibyte},
		{name: "invalid utf8", in: "\xff\xfe"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := types.CanonicalLabelName(tc.in)
			if err == nil {
				t.Fatalf("expected error for %q, got %q", tc.in, got)
			}
			if got != "" {
				t.Fatalf("expected empty result on error, got %q", got)
			}
		})
	}
}

// TestCanonicalizeLabelList_DedupesPostCanon covers the case where two input
// labels are byte-different pre-canonicalization but byte-equal after — the
// helper must flag the duplicate when rejectDuplicates is true, and must
// pass silently when it is false (with both canonical forms returned).
func TestCanonicalizeLabelList_DedupesPostCanon(t *testing.T) {
	t.Parallel()
	in := []string{"e\u0301", " \u00e9 "}
	if _, err := types.CanonicalizeLabelList(in, true); err == nil {
		t.Fatalf("expected duplicate-post-canon error, got nil")
	}
	got, err := types.CanonicalizeLabelList(in, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != "\u00e9" || got[1] != "\u00e9" {
		t.Fatalf("unexpected result: %q", got)
	}
}

// TestCanonicalizeLabelList_RejectsInner ensures the index-reporting
// behaviour: if a middle-of-list entry fails canonicalization, the error
// must reference its index and the overall result must be empty.
func TestCanonicalizeLabelList_RejectsInner(t *testing.T) {
	t.Parallel()
	in := []string{"ok", "", "also_ok"}
	got, err := types.CanonicalizeLabelList(in, true)
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
