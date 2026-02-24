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

var (
	// Pre-computed boundary values
	minBoundValue Dec
	maxBoundValue Dec
)

// Initialize the boundary values for performance
func init() {
	minBoundValue = MustNewDecFromString(fmt.Sprintf("1e-%d", DataLimitExponent))
	maxBoundValue = MustNewDecFromString(fmt.Sprintf("1e%d", DataLimitExponent))
}

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

	// Check magnitude boundaries
	if absValue.Lt(minBoundValue) {
		return errorsmod.Wrapf(ErrOutOfRange, "value %s cannot have higher precision than %d", d, DataLimitExponent)
	}
	if absValue.Gt(maxBoundValue) {
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

// MustNewBoundedExp40DecFromString creates a new BoundedExp40Dec from a string.
// It panics if the string cannot be parsed or if the resulting decimal is out of bounds.
func MustNewBoundedExp40DecFromString(s string) BoundedExp40Dec {
	dec, err := NewBoundedExp40DecFromString(s)
	if err != nil {
		panic(err)
	}
	return dec
}

// NewBoundedExp40DecFromString creates a new BoundedExp40Dec from a string
func NewBoundedExp40DecFromString(s string) (BoundedExp40Dec, error) {
	d, err := NewDecFromString(s)
	if err != nil {
		return BoundedExp40Dec{}, errorsmod.Wrap(err, "failed to parse decimal")
	}
	return NewBoundedExp40Dec(d)
}

// NewCappedBoundedExp40Dec creates a BoundedExp40Dec from a Dec,
// capping the value to the allowed bounds if it's out of range.
func NewCappedBoundedExp40Dec(d Dec) (BoundedExp40Dec, error) {
	if !d.IsFinite() {
		return BoundedExp40Dec{}, errorsmod.Wrap(ErrOutOfRange, "cannot cap non-finite decimal")
	}

	if d.IsZero() {
		return BoundedExp40Dec{dec: d}, nil
	}

	absValue, err := d.Abs()
	if err != nil {
		return BoundedExp40Dec{}, errorsmod.Wrap(err, "failed to compute absolute value")
	}

	var capped Dec
	switch {
	case absValue.Lt(minBoundValue):
		// Too small → cap to min bound with correct sign
		if d.IsNegative() {
			capped, err = minBoundValue.Neg()
			if err != nil {
				return BoundedExp40Dec{}, errorsmod.Wrap(err, "failed to negate min bound value")
			}
		} else {
			capped = minBoundValue
		}
	case absValue.Gt(maxBoundValue):
		// Too large → cap to max bound with correct sign
		if d.IsNegative() {
			capped, err = maxBoundValue.Neg()
			if err != nil {
				return BoundedExp40Dec{}, errorsmod.Wrap(err, "failed to negate max bound value")
			}
		} else {
			capped = maxBoundValue
		}
	default:
		capped = d
	}

	return BoundedExp40Dec{dec: capped}, nil
}

// NewCappedBoundedExp40DecFromString creates a BoundedExp40Dec from a string,
// capping the value to the allowed bounds if it's out of range.
func NewCappedBoundedExp40DecFromString(s string) (BoundedExp40Dec, error) {
	d, err := NewDecFromString(s)
	if err != nil {
		return BoundedExp40Dec{}, errorsmod.Wrap(err, "failed to parse decimal")
	}
	return NewCappedBoundedExp40Dec(d)
}

// MustNewCappedBoundedExp40Dec creates a BoundedExp40Dec from a Dec,
// capping the value to the allowed bounds if it's out of range.
// It panics on error.
func MustNewCappedBoundedExp40Dec(d Dec) BoundedExp40Dec {
	dec, err := NewCappedBoundedExp40Dec(d)
	if err != nil {
		panic(err)
	}
	return dec
}

// MustNewCappedBoundedExp40DecFromString creates a BoundedExp40Dec from a string,
// capping the value to the allowed bounds if it's out of range.
// It panics on parse or boundary errors.
func MustNewCappedBoundedExp40DecFromString(s string) BoundedExp40Dec {
	dec, err := NewCappedBoundedExp40DecFromString(s)
	if err != nil {
		panic(err)
	}
	return dec
}

// Returns the maximum value that can be represented by a BoundedExp40Dec
func GetMaxPositiveBoundaryExp40Dec() BoundedExp40Dec {
	return BoundedExp40Dec{dec: maxBoundValue}
}

// Returns the minimum value that can be represented by a BoundedExp40Dec
func GetMinPositiveBoundaryExp40Dec() BoundedExp40Dec {
	return BoundedExp40Dec{dec: minBoundValue}
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

func (bd BoundedExp40Dec) ToDec() Dec {
	return bd.dec
}

func (bd BoundedExp40Dec) Equal(y BoundedExp40Dec) bool {
	return bd.dec.Equal(y.dec)
}

func (bd BoundedExp40Dec) String() string {
	return bd.dec.String()
}

func (bd BoundedExp40Dec) IsNaN() bool {
	return bd.dec.IsNaN()
}

func (bd BoundedExp40Dec) IsFinite() bool {
	return bd.dec.IsFinite()
}
