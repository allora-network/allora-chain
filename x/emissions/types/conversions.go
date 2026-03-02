package types

import (
	"cosmossdk.io/errors"
)

// NewForecastElementFromInput converts InputForecastElement to ForecastElement
func NewForecastElementFromInput(bfe *InputForecastElement) (*ForecastElement, error) {
	if bfe == nil {
		return nil, ErrInvalidValue
	}
	dec := bfe.Value.ToDec()
	forecastElement := &ForecastElement{
		Inferer: bfe.Inferer,
		Value:   dec,
	}
	err := forecastElement.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to validate forecast element")
	}
	return forecastElement, nil
}

// NewForecastFromInput converts InputForecast to Forecast
func NewForecastFromInput(bf *InputForecast) (*Forecast, error) {
	if bf == nil {
		return nil, ErrInvalidValue
	}
	elements := make([]*ForecastElement, len(bf.ForecastElements))
	for i, elem := range bf.ForecastElements {
		converted, err := NewForecastElementFromInput(elem)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert forecast element %d", i)
		}
		elements[i] = converted
	}
	forecast := &Forecast{
		TopicId:          bf.TopicId,
		BlockHeight:      bf.BlockHeight,
		Forecaster:       bf.Forecaster,
		ForecastElements: elements,
		ExtraData:        bf.ExtraData,
	}
	err := forecast.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to validate forecast")
	}
	return forecast, nil
}

// NewWorkerAttributedValueFromInput converts InputWorkerAttributedValue to WorkerAttributedValue
func NewWorkerAttributedValueFromInput(bwav *InputWorkerAttributedValue) (*WorkerAttributedValue, error) {
	if bwav == nil {
		return nil, ErrInvalidValue
	}
	dec := bwav.Value.ToDec()
	workerAttributedValue := &WorkerAttributedValue{
		Worker: bwav.Worker,
		Value:  dec,
	}
	err := workerAttributedValue.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to validate worker attributed value")
	}
	return workerAttributedValue, nil
}

// NewWithheldWorkerAttributedValueFromInput converts InputWithheldWorkerAttributedValue to WithheldWorkerAttributedValue
func NewWithheldWorkerAttributedValueFromInput(bwwav *InputWithheldWorkerAttributedValue) (*WithheldWorkerAttributedValue, error) {
	if bwwav == nil {
		return nil, ErrInvalidValue
	}
	dec := bwwav.Value.ToDec()
	withheldWorkerAttributedValue := &WithheldWorkerAttributedValue{
		Worker: bwwav.Worker,
		Value:  dec,
	}
	err := withheldWorkerAttributedValue.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to validate withheld worker attributed value")
	}
	return withheldWorkerAttributedValue, nil
}

// NewOneOutInfererForecasterValuesFromInput converts InputOneOutInfererForecasterValues to OneOutInfererForecasterValues
func NewOneOutInfererForecasterValuesFromInput(boifv *InputOneOutInfererForecasterValues) (*OneOutInfererForecasterValues, error) {
	if boifv == nil {
		return nil, ErrInvalidValue
	}
	values := make([]*WithheldWorkerAttributedValue, len(boifv.OneOutInfererValues))
	for i, val := range boifv.OneOutInfererValues {
		converted, err := NewWithheldWorkerAttributedValueFromInput(val)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert one out inferer value %d", i)
		}
		values[i] = converted
	}
	oneOutInfererForecasterValues := &OneOutInfererForecasterValues{
		Forecaster:          boifv.Forecaster,
		OneOutInfererValues: values,
	}
	err := oneOutInfererForecasterValues.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to validate one out inferer forecaster values")
	}
	return oneOutInfererForecasterValues, nil
}

// NewValueBundleFromInput converts InputValueBundle to ValueBundle
func NewValueBundleFromInput(bvb *InputValueBundle) (*ValueBundle, error) {
	if bvb == nil {
		return nil, ErrInvalidValue
	}

	combinedValue := bvb.CombinedValue.ToDec()
	naiveValue := bvb.NaiveValue.ToDec()

	infererValues := make([]*WorkerAttributedValue, len(bvb.InfererValues))
	for i, val := range bvb.InfererValues {
		converted, err := NewWorkerAttributedValueFromInput(val)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert inferer value %d", i)
		}
		infererValues[i] = converted
	}

	forecasterValues := make([]*WorkerAttributedValue, len(bvb.ForecasterValues))
	for i, val := range bvb.ForecasterValues {
		converted, err := NewWorkerAttributedValueFromInput(val)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert forecaster value %d", i)
		}
		forecasterValues[i] = converted
	}

	oneOutInfererValues := make([]*WithheldWorkerAttributedValue, len(bvb.OneOutInfererValues))
	for i, val := range bvb.OneOutInfererValues {
		converted, err := NewWithheldWorkerAttributedValueFromInput(val)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert one out inferer value %d", i)
		}
		oneOutInfererValues[i] = converted
	}

	oneOutForecasterValues := make([]*WithheldWorkerAttributedValue, len(bvb.OneOutForecasterValues))
	for i, val := range bvb.OneOutForecasterValues {
		converted, err := NewWithheldWorkerAttributedValueFromInput(val)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert one out forecaster value %d", i)
		}
		oneOutForecasterValues[i] = converted
	}

	oneInForecasterValues := make([]*WorkerAttributedValue, len(bvb.OneInForecasterValues))
	for i, val := range bvb.OneInForecasterValues {
		converted, err := NewWorkerAttributedValueFromInput(val)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert one in forecaster value %d", i)
		}
		oneInForecasterValues[i] = converted
	}

	oneOutInfererForecasterValues := make([]*OneOutInfererForecasterValues, len(bvb.OneOutInfererForecasterValues))
	for i, val := range bvb.OneOutInfererForecasterValues {
		converted, err := NewOneOutInfererForecasterValuesFromInput(val)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert one out inferer forecaster value %d", i)
		}
		oneOutInfererForecasterValues[i] = converted
	}

	valueBundle := &ValueBundle{
		TopicId:                       bvb.TopicId,
		ReputerRequestNonce:           bvb.ReputerRequestNonce,
		Reputer:                       bvb.Reputer,
		ExtraData:                     bvb.ExtraData,
		CombinedValue:                 combinedValue,
		InfererValues:                 infererValues,
		ForecasterValues:              forecasterValues,
		NaiveValue:                    naiveValue,
		OneOutInfererValues:           oneOutInfererValues,
		OneOutForecasterValues:        oneOutForecasterValues,
		OneInForecasterValues:         oneInForecasterValues,
		OneOutInfererForecasterValues: oneOutInfererForecasterValues,
	}
	err := valueBundle.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to validate value bundle")
	}
	return valueBundle, nil
}

// NewLossBundleFromInput converts InputReputerValueBundle to ReputerValueBundle
func NewLossBundleFromInput(brvb *InputReputerValueBundle) (*LossBundle, error) {
	if brvb == nil {
		return nil, ErrInvalidValue
	}
	if err := brvb.Validate(); err != nil {
		return nil, errors.Wrap(err, "failed to validate reputer value bundle")
	}
	valueBundle, err := NewValueBundleFromInput(brvb.ValueBundle)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert value bundle")
	}
	return valueBundle, nil
}
