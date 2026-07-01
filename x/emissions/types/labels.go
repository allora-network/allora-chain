package types

import (
	"cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	// SingleArityCanonicalLabel is the only non-empty label a SINGLE-arity topic
	// submission may carry.
	SingleArityCanonicalLabel = "y"

	// SingleArityCanonicalLabelID is the fixed registry id for that label.
	SingleArityCanonicalLabelID uint32 = 1
)

// ValidateSingleArityLabel enforces that a SINGLE-arity submission carries
// either no label (legacy scalar path) or the canonical label "y".
func ValidateSingleArityLabel(label string) error {
	if label != "" && label != SingleArityCanonicalLabel {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest,
			"single-arity label must be %q, got %q", SingleArityCanonicalLabel, label)
	}
	return nil
}
