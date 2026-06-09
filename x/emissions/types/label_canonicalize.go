package types

import (
	"strings"
	"unicode/utf8"

	errorsmod "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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
// Case-insensitive canonicalization should apply ASCII-only lowercasing after
// this predicate has rejected non-ASCII input.
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

// CanonicalLabelName validates and canonicalizes a label for storage/key use:
// requires valid UTF-8, trims, enforces the ASCII-only label charset, optionally
// lowercases ASCII letters, then caps the byte length. (NFC is applied too but
// is inert for the ASCII charset; see the call site.)
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
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return "", errorsmod.Wrap(ErrInvalidLabelName, "label is empty after trimming")
	}
	for _, r := range trimmed {
		if !isAllowedLabelByte(r) {
			return "", errorsmod.Wrapf(ErrInvalidLabelName,
				"label contains a disallowed character: U+%04X", r)
		}
	}
	// NFC-normalize: a no-op for today's ASCII-only charset (the loop above
	// rejected all non-ASCII), kept so a future charset widening can't silently
	// skip normalization. Must run before the maxBytes check below.
	trimmed = norm.NFC.String(trimmed)
	if !labelCaseSensitive {
		trimmed = lowerASCII(trimmed)
	}
	if uint64(len(trimmed)) > maxBytes {
		return "", errorsmod.Wrapf(ErrInvalidLabelName,
			"label exceeds %d bytes (got %d)",
			maxBytes, len(trimmed))
	}
	return trimmed, nil
}

// EnsureCanonicalLabelName returns nil iff s is already in canonical form for
// the given limits. It is the validation-only counterpart to CanonicalLabelName
// for call sites that must reject (not rewrite) non-canonical input: it returns
// the underlying CanonicalLabelName error when s is invalid, and ErrLogic when
// s is valid UTF-8 but not canonical (needs trim/lowercase/NFC/length), which
// signals an upstream invariant break at registry/keeper call sites.
func EnsureCanonicalLabelName(s string, maxBytes uint64, labelCaseSensitive bool) error {
	canonical, err := CanonicalLabelName(s, maxBytes, labelCaseSensitive)
	if err != nil {
		return err
	}
	if canonical != s {
		return errorsmod.Wrapf(sdkerrors.ErrLogic, "label name must already be canonical: %q", s)
	}
	return nil
}

func lowerASCII(s string) string {
	out := []byte(s)
	for i, b := range out {
		if b >= 'A' && b <= 'Z' {
			out[i] = b + ('a' - 'A')
		}
	}
	return string(out)
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
