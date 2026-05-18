package types

import (
	"testing"
	"unicode/utf8"
)

func TestIsAllowedLabelByte(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		r    rune
		want bool
	}{
		// Lowercase a–z (first switch branch).
		{name: "lowercase_a", r: 'a', want: true},
		{name: "lowercase_m", r: 'm', want: true},
		{name: "lowercase_z", r: 'z', want: true},
		{name: "before_lowercase_a_grave", r: '`', want: false},
		{name: "after_lowercase_z_brace", r: '{', want: false},

		// Uppercase A–Z.
		{name: "uppercase_A", r: 'A', want: true},
		{name: "uppercase_N", r: 'N', want: true},
		{name: "uppercase_Z", r: 'Z', want: true},
		{name: "before_uppercase_A_at", r: '@', want: false},
		{name: "after_uppercase_Z_bracket", r: '[', want: false},

		// Digits 0–9.
		{name: "digit_0", r: '0', want: true},
		{name: "digit_5", r: '5', want: true},
		{name: "digit_9", r: '9', want: true},
		{name: "between_digit_9_and_uppercase_A_colon", r: ':', want: false},

		// Explicit separators (second switch).
		{name: "underscore", r: '_', want: true},
		{name: "hyphen_minus", r: '-', want: true},
		{name: "space", r: ' ', want: true},
		{name: "slash", r: '/', want: true},
		{name: "dot", r: '.', want: true},

		// ASCII punctuation / controls not in the charset.
		{name: "bang", r: '!', want: false},
		{name: "comma", r: ',', want: false},
		{name: "tab", r: '\t', want: false},
		{name: "newline", r: '\n', want: false},
		{name: "nul", r: 0, want: false},

		// Outside ASCII letters (unicode).
		{name: "unicode_e_acute", r: 'é', want: false},
		{name: "utf8_rune_error_sentinel", r: utf8.RuneError, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := isAllowedLabelByte(tc.r); got != tc.want {
				t.Fatalf("isAllowedLabelByte(U+%04X) = %v, want %v", tc.r, got, tc.want)
			}
		})
	}
}
