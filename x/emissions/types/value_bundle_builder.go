package types

// Can abstract this out to different kinds of builders later
type ValueBundleBuilder interface {
	setCombinedValue()
	setInfererValues()
	setForecasterValues()
	setNaiveValue()
	setOneOutInfererValues()
	setOneOutForecasterValues()
	setOneInValues()
	getNetworkValues() (ValueBundle, error)
}
