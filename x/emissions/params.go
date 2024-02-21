package emissions

// DefaultParams returns default module parameters.
func DefaultParams() Params {
	return Params{
		// Set default values here.
		Version: "0.0.1",
	}
}

// Validate does the sanity check on the params.
func (p Params) Validate() error {
	// Sanity check goes here.
	return nil
}
