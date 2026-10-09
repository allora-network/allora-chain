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

// LabelNamesByID returns the registry's label names in compact-id order:
// element i is the name of the label with id i+1. It fails unless the ids are
// exactly 1 through len(Labels), each once.
func (r EpochLabelRegistry) LabelNamesByID() ([]string, error) {
	names := make([]string, len(r.Labels))
	seen := make([]bool, len(r.Labels))
	for _, lbl := range r.Labels {
		if lbl == nil {
			return nil, errors.Wrap(sdkerrors.ErrLogic, "epoch label registry has a nil label")
		}
		if lbl.Id < 1 || int(lbl.Id) > len(r.Labels) {
			return nil, errors.Wrapf(sdkerrors.ErrLogic, "epoch label registry id %d out of range 1..%d", lbl.Id, len(r.Labels))
		}
		if seen[lbl.Id-1] {
			return nil, errors.Wrapf(sdkerrors.ErrLogic, "epoch label registry repeats id %d", lbl.Id)
		}
		seen[lbl.Id-1] = true
		names[lbl.Id-1] = lbl.Name
	}
	return names, nil
}
