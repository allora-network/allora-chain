package types

import (
	"strings"
	"unicode"
	"unicode/utf8"

	errorsmod "cosmossdk.io/errors"
	"golang.org/x/text/unicode/norm"
)

// MaxCanonicalLabelByteLength bounds a canonical label name at 64 UTF-8 bytes
// after NFC normalization and trimming. The bound is byte-level (not
// rune-level) so staged InputInference labels and every EpochLabelRegistry
// entry have a deterministic, modest upper bound on serialized size,
// regardless of how many codepoints the label contains.
const MaxCanonicalLabelByteLength = 64

// CanonicalLabelName returns the canonical form of a user-supplied label
// name. The canonical form is:
//
//  1. NFC-normalized (Unicode Normalization Form C), so that visually
//     identical sequences always compare byte-equal.
//  2. Trimmed of leading and trailing Unicode whitespace (strings.TrimSpace).
//  3. Checked to be non-empty after trimming.
//  4. Checked to contain no control (Cc) or format (Cf) runes — these are
//     invisible characters (e.g. zero-width space, bidi marks, NULs) that
//     would otherwise allow two visually-identical labels to differ by a
//     stealthy invisible rune and thus break deterministic registry identity.
//  5. Bounded at MaxCanonicalLabelByteLength bytes after normalization.
//  6. Verified to be valid UTF-8.
//
// The function is idempotent by construction: CanonicalLabelName(c) == c for
// every c already in canonical form, because NFC is idempotent, TrimSpace is
// idempotent on trimmed input, and the Cc/Cf / byte-length / UTF-8 checks
// are pure validations that do not mutate the string. This invariant is
// exercised by FuzzCanonicalLabelName.
//
// Canonicalization is applied at two sites:
//   - InputInference.ValidateWithLimits (worker payload submission-time), so
//     that every label registered in the temporary EpochLabelRegistry is
//     canonical before close-time registry construction.
//   - TopicKeeper.SetTopic / UpdateTopic (persisted Topic.LabelWhitelist), so
//     that whitelist lookups are pure byte-equality against already-canonical
//     names built by the msgserver.
func CanonicalLabelName(s string) (string, error) {
	if !utf8.ValidString(s) {
		return "", errorsmod.Wrap(ErrInvalidLabelName, "label is not valid UTF-8")
	}
	normalized := norm.NFC.String(s)
	trimmed := strings.TrimSpace(normalized)
	if trimmed == "" {
		return "", errorsmod.Wrap(ErrInvalidLabelName, "label is empty after trimming")
	}
	for _, r := range trimmed {
		if unicode.In(r, unicode.Cc, unicode.Cf) {
			return "", errorsmod.Wrapf(ErrInvalidLabelName,
				"label contains a disallowed control/format rune: U+%04X", r)
		}
	}
	if len(trimmed) > MaxCanonicalLabelByteLength {
		return "", errorsmod.Wrapf(ErrInvalidLabelName,
			"label exceeds %d UTF-8 bytes after normalization (got %d)",
			MaxCanonicalLabelByteLength, len(trimmed))
	}
	return trimmed, nil
}

// CanonicalizeLabelList canonicalizes each entry in the input slice and
// returns a new slice with the canonical forms in the same order. It rejects
// (returns an error for) any entry that fails CanonicalLabelName and, when
// rejectDuplicates is true, any post-canonicalization duplicate.
//
// The duplicate check is exact byte-equality on the canonical form, which is
// what downstream consumers such as whitelist membership and ELR registration
// rely on.
func CanonicalizeLabelList(labels []string, rejectDuplicates bool) ([]string, error) {
	if len(labels) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(labels))
	var seen map[string]struct{}
	if rejectDuplicates {
		seen = make(map[string]struct{}, len(labels))
	}
	for i, raw := range labels {
		c, err := CanonicalLabelName(raw)
		if err != nil {
			return nil, errorsmod.Wrapf(err, "label at index %d", i)
		}
		if rejectDuplicates {
			if _, dup := seen[c]; dup {
				return nil, errorsmod.Wrapf(ErrInvalidLabelName,
					"duplicate label after canonicalization at index %d: %q", i, c)
			}
			seen[c] = struct{}{}
		}
		out = append(out, c)
	}
	return out, nil
}

// CanonicalLabelSet builds a lookup set over a slice of already-canonical
// labels. Helper for the msgserver whitelist check, which constructs the
// set once per payload and reuses it across all submitted labels.
func CanonicalLabelSet(labels []string) map[string]struct{} {
	if len(labels) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(labels))
	for _, l := range labels {
		set[l] = struct{}{}
	}
	return set
}
