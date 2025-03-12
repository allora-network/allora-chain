package math

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
)

const (
	// DataLimitExponent defines the default exponent limit for bounded decimals
	// Built-in in the type definition for strict API contract
	DataLimitExponent = 40
)

// BoundedExp40Dec represents a decimal with enforced value boundaries.
// It wraps the base Dec type and enforces limits during marshaling/unmarshaling.
type BoundedExp40Dec struct {
	dec Dec
}

// validateBounds checks if the decimal is within the allowed bounds
func validateBounds(d Dec) error {
	if !d.IsFinite() {
		return errorsmod.Wrap(ErrOutOfRange, "value must be finite")
	}

	if d.IsZero() {
		return nil
	}

	// Get absolute value
	absValue, err := d.Abs()
	if err != nil {
		return errorsmod.Wrap(err, "failed to get absolute value")
	}

	// Create boundary values for the magnitude
	baseString := "1e"
	minValue, _ := NewDecFromString(baseString + fmt.Sprintf("-%d", DataLimitExponent))
	maxValue, _ := NewDecFromString(baseString + fmt.Sprintf("%d", DataLimitExponent))

	// Check magnitude boundaries
	if absValue.Lt(minValue) {
		return errorsmod.Wrapf(ErrOutOfRange, "value %s cannot have higher precision than %d", d, DataLimitExponent)
	}
	if absValue.Gt(maxValue) {
		return errorsmod.Wrapf(ErrOutOfRange, "value %s cannot be greater than ±1e%d", d, DataLimitExponent)
	}

	return nil
}

// MustNewBoundedExp40Dec returns a new BoundedExp40Dec from a given Dec. It panics if the Dec
// is not valid.
func MustNewBoundedExp40Dec(d Dec) BoundedExp40Dec {
	dec, err := NewBoundedExp40Dec(d)
	if err != nil {
		panic(err)
	}
	return dec
}

// NewBoundedExp40Dec creates a new BoundedExp40Dec from a Dec
func NewBoundedExp40Dec(d Dec) (BoundedExp40Dec, error) {
	if err := validateBounds(d); err != nil {
		return BoundedExp40Dec{}, err
	}
	return BoundedExp40Dec{dec: d}, nil
}

// NewBoundedExp40DecFromString creates a new BoundedExp40Dec from a string
func NewBoundedExp40DecFromString(s string) (BoundedExp40Dec, error) {
	d, err := NewDecFromString(s)
	if err != nil {
		return BoundedExp40Dec{}, errorsmod.Wrap(err, "failed to parse decimal")
	}
	return NewBoundedExp40Dec(d)
}

// Marshal implements the gogo proto custom type interface
func (bd BoundedExp40Dec) Marshal() ([]byte, error) {
	if err := validateBounds(bd.dec); err != nil {
		return nil, err
	}
	return bd.dec.Marshal()
}

// Unmarshal implements the gogo proto custom type interface
func (bd *BoundedExp40Dec) Unmarshal(data []byte) error {
	if err := bd.dec.Unmarshal(data); err != nil {
		return err
	}
	if err := validateBounds(bd.dec); err != nil {
		return err
	}
	return nil
}

// MarshalJSON implements json.Marshaler
func (bd BoundedExp40Dec) MarshalJSON() ([]byte, error) {
	if err := validateBounds(bd.dec); err != nil {
		return nil, err
	}
	return bd.dec.MarshalJSON()
}

// UnmarshalJSON implements json.Unmarshaler
func (bd *BoundedExp40Dec) UnmarshalJSON(bz []byte) error {
	if err := bd.dec.UnmarshalJSON(bz); err != nil {
		return err
	}
	if err := validateBounds(bd.dec); err != nil {
		return err
	}
	return nil
}

// Size returns the encoded size of BoundedExp40Dec
func (bd BoundedExp40Dec) Size() int {
	return bd.dec.Size()
}

// MarshalTo serializes BoundedExp40Dec into a given buffer
func (bd BoundedExp40Dec) MarshalTo(data []byte) (int, error) {
	if err := validateBounds(bd.dec); err != nil {
		return 0, err
	}
	return bd.dec.MarshalTo(data)
}

func (bd BoundedExp40Dec) ToDec() (Dec, error) {
	return bd.dec, nil
}

func (bd BoundedExp40Dec) Equal(y BoundedExp40Dec) bool {
	return bd.dec.Equal(y.dec)
}

func (bd BoundedExp40Dec) String() string {
	return bd.dec.String()
}
