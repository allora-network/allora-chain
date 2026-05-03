package types_test

import (
	"testing"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

// FuzzCanonicalLabelName asserts two invariants that the rest of the epoch
// label registry pipeline relies on:
//
//  1. The function never panics, regardless of input.
//  2. When canonicalization succeeds, the result is idempotent under a
//     second pass. This is the property that makes registry identity
//     deterministic: two inputs that canonicalize to the same bytes are
//     treated as the same label everywhere.
//
// The seed corpus exercises ASCII, NFC/NFD equivalents, composed emoji,
// control characters, and invalid UTF-8 to push the branch coverage of the
// canonicalizer broadly out of the gate.
func FuzzCanonicalLabelName(f *testing.F) {
	seeds := []string{
		"",
		"   ",
		"y",
		"label",
		"  label  ",
		"e\u0301",
		"\u00e9",
		"\u200bhidden",
		"\xff\xfe",
		"🚀",
		"a\x00b",
		"mixed\t space",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		out, err := types.CanonicalLabelName(in)
		if err != nil {
			if out != "" {
				t.Fatalf("CanonicalLabelName returned %q with error %v", out, err)
			}
			return
		}
		again, err := types.CanonicalLabelName(out)
		if err != nil {
			t.Fatalf("idempotence pass failed on %q: %v", out, err)
		}
		if again != out {
			t.Fatalf("CanonicalLabelName not idempotent: %q -> %q", out, again)
		}
	})
}
