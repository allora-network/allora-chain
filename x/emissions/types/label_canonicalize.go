package types

import (
	"strings"
	"unicode/utf8"

	errorsmod "cosmossdk.io/errors"
	"golang.org/x/text/unicode/norm"
)

// isAllowedLabelByte reports whether r belongs to the canonical label charset.
//
// The canonical charset is ASCII only:
//   - letters: a-z (and A-Z when the topic is label-case-sensitive)
//   - digits: 0-9
//   - separators: underscore, hyphen-minus, space
//   - hierarchy: forward slash, dot
//
// Callers that want case-insensitive canonicalization MUST lowercase before
// calling this predicate on each rune.
func isAllowedLabelByte(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z':
		return true
	case r >= 'A' && r <= 'Z':
		return true
	case r >= '0' && r <= '9':
		return true
	}
	switch r {
	case '_', '-', ' ', '/', '.':
		return true
	}
	return false
}

// CanonicalLabelName validates and canonicalizes a label for storage/key use.
// It requires valid UTF-8, NFC-normalizes, trims, optionally lowercases, then
// enforces the ASCII label charset and max byte length.
// The result is idempotent for the same maxBytes and labelCaseSensitive values.
func CanonicalLabelName(s string, maxBytes uint64, labelCaseSensitive bool) (string, error) {
	if maxBytes == 0 {
		// Defensive: Params.Validate rejects zero; a zero here means the
		// caller threaded an un-loaded / zero-value params struct.
		return "", errorsmod.Wrap(ErrInvalidLabelName,
			"max canonical label byte length must be >= 1")
	}
	if !utf8.ValidString(s) {
		return "", errorsmod.Wrap(ErrInvalidLabelName, "label is not valid UTF-8")
	}
	normalized := norm.NFC.String(s)
	trimmed := strings.TrimSpace(normalized)
	if trimmed == "" {
		return "", errorsmod.Wrap(ErrInvalidLabelName, "label is empty after trimming")
	}
	if !labelCaseSensitive {
		// Using strings.ToLower on ASCII-only input produces ASCII-only
		// output; non-ASCII runes would be rejected by the charset check
		// that follows regardless. Lowercasing before the charset check
		// keeps the rejection message accurate even when the input had
		// uppercase letters that would have been legal under
		// label_case_sensitive.
		trimmed = strings.ToLower(trimmed)
	}
	// Enforce byte cap before iterating so we do not spend work on inputs
	// that the byte-length check will reject anyway.
	if uint64(len(trimmed)) > maxBytes {
		return "", errorsmod.Wrapf(ErrInvalidLabelName,
			"label exceeds %d bytes after normalization (got %d)",
			maxBytes, len(trimmed))
	}
	for _, r := range trimmed {
		if !isAllowedLabelByte(r) {
			return "", errorsmod.Wrapf(ErrInvalidLabelName,
				"label contains a disallowed character: U+%04X", r)
		}
	}
	return trimmed, nil
}

// CanonicalizeLabelList canonicalizes each entry in the input slice and
// returns a new slice with the canonical forms in the same order. It rejects
// (returns an error for) any entry that fails CanonicalLabelName and, when
// rejectDuplicates is true, any post-canonicalization duplicate.
//
// maxBytes and labelCaseSensitive are threaded through to CanonicalLabelName;
// callers should pass Params.MaxCanonicalLabelByteLength and the topic's
// LabelCaseSensitive field so that whitelist canonicalization matches
// submission-time canonicalization byte-for-byte.
//
// The duplicate check is exact byte-equality on the canonical form, which is
// what downstream consumers (whitelist membership, registry lex-sort,
// registry keys) rely on.
func CanonicalizeLabelList(labels []string, rejectDuplicates bool, maxBytes uint64, labelCaseSensitive bool) ([]string, error) {
	if len(labels) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(labels))
	var seen map[string]struct{}
	if rejectDuplicates {
		seen = make(map[string]struct{}, len(labels))
	}
	for i, raw := range labels {
		c, err := CanonicalLabelName(raw, maxBytes, labelCaseSensitive)
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
