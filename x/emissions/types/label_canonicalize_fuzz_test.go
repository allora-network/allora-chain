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
//     second pass with the same caseSensitive flag. This is the property
//     that makes the lex-sort assignment of registry IDs deterministic:
//     two inputs that canonicalize to the same bytes are treated as the
//     same label everywhere.
//
// The seed corpus exercises ASCII, allowed separators, uppercase, invalid
// charset, control characters, invalid UTF-8, and both case modes to push
// branch coverage out of the gate.
func FuzzCanonicalLabelName(f *testing.F) {
	type seed struct {
		s  string
		cs bool
	}
	seeds := []seed{
		{s: "", cs: false},
		{s: "   ", cs: false},
		{s: "y", cs: false},
		{s: "label", cs: false},
		{s: "  label  ", cs: false},
		{s: "Cat", cs: false},
		{s: "Cat", cs: true},
		{s: "foo/bar.baz", cs: false},
		{s: "foo-bar_baz 0", cs: true},
		{s: "\u200bhidden", cs: false},
		{s: "\xff\xfe", cs: false},
		{s: "🚀", cs: false},
		{s: "a\x00b", cs: false},
		{s: "bad!", cs: false},
	}
	for _, s := range seeds {
		f.Add(s.s, s.cs)
	}
	f.Fuzz(func(t *testing.T, in string, labelCaseSensitive bool) {
		out, err := types.CanonicalLabelName(in, 64, labelCaseSensitive)
		if err != nil {
			if out != "" {
				t.Fatalf("CanonicalLabelName returned %q with error %v", out, err)
			}
			return
		}
		again, err := types.CanonicalLabelName(out, 64, labelCaseSensitive)
		if err != nil {
			t.Fatalf("idempotence pass failed on %q: %v", out, err)
		}
		if again != out {
			t.Fatalf("CanonicalLabelName not idempotent: %q -> %q", out, again)
		}
	})
}
