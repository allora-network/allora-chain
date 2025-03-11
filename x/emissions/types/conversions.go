package types

import (
	"cosmossdk.io/errors"
)

// Convert converts BoundedInference to Inference
func (bi *BoundedInference) Convert() (*Inference, error) {
	if bi == nil {
		return nil, nil
	}
	dec, err := bi.Value.ToDec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert value")
	}
	return &Inference{
		TopicId:     bi.TopicId,
		BlockHeight: bi.BlockHeight,
		Inferer:     bi.Inferer,
		Value:       dec,
		ExtraData:   bi.ExtraData,
		Proof:       bi.Proof,
	}, nil
}

// Convert converts BoundedForecastElement to ForecastElement
func (bfe *BoundedForecastElement) Convert() (*ForecastElement, error) {
	if bfe == nil {
		return nil, nil
	}
	dec, err := bfe.Value.ToDec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert value")
	}
	return &ForecastElement{
		Inferer: bfe.Inferer,
		Value:   dec,
	}, nil
}

// Convert converts BoundedForecast to Forecast
func (bf *BoundedForecast) Convert() (*Forecast, error) {
	if bf == nil {
		return nil, nil
	}
	elements := make([]*ForecastElement, len(bf.ForecastElements))
	for i, elem := range bf.ForecastElements {
		converted, err := elem.Convert()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert forecast element %d", i)
		}
		elements[i] = converted
	}
	return &Forecast{
		TopicId:          bf.TopicId,
		BlockHeight:      bf.BlockHeight,
		Forecaster:       bf.Forecaster,
		ForecastElements: elements,
		ExtraData:        bf.ExtraData,
	}, nil
}

// Convert converts BoundedInferenceForecastBundle to InferenceForecastBundle
func (bifb *BoundedInferenceForecastBundle) Convert() (*InferenceForecastBundle, error) {
	if bifb == nil {
		return nil, nil
	}
	inference, err := bifb.Inference.Convert()
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert inference")
	}
	forecast, err := bifb.Forecast.Convert()
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert forecast")
	}
	return &InferenceForecastBundle{
		Inference: inference,
		Forecast:  forecast,
	}, nil
}

// Convert converts BoundedWorkerDataBundle to WorkerDataBundle
func (bwdb *BoundedWorkerDataBundle) Convert() (*WorkerDataBundle, error) {
	if bwdb == nil {
		return nil, nil
	}
	bundle, err := bwdb.InferenceForecastsBundle.Convert()
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert inference forecasts bundle")
	}
	return &WorkerDataBundle{
		Worker:                             bwdb.Worker,
		Nonce:                              bwdb.Nonce,
		TopicId:                            bwdb.TopicId,
		InferenceForecastsBundle:           bundle,
		InferencesForecastsBundleSignature: bwdb.InferencesForecastsBundleSignature,
		Pubkey:                             bwdb.Pubkey,
	}, nil
}

// Convert converts BoundedWorkerAttributedValue to WorkerAttributedValue
func (bwav *BoundedWorkerAttributedValue) Convert() (*WorkerAttributedValue, error) {
	if bwav == nil {
		return nil, nil
	}
	dec, err := bwav.Value.ToDec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert value")
	}
	return &WorkerAttributedValue{
		Worker: bwav.Worker,
		Value:  dec,
	}, nil
}

// Convert converts BoundedWithheldWorkerAttributedValue to WithheldWorkerAttributedValue
func (bwwav *BoundedWithheldWorkerAttributedValue) Convert() (*WithheldWorkerAttributedValue, error) {
	if bwwav == nil {
		return nil, nil
	}
	dec, err := bwwav.Value.ToDec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert value")
	}
	return &WithheldWorkerAttributedValue{
		Worker: bwwav.Worker,
		Value:  dec,
	}, nil
}

// Convert converts BoundedOneOutInfererForecasterValues to OneOutInfererForecasterValues
func (boifv *BoundedOneOutInfererForecasterValues) Convert() (*OneOutInfererForecasterValues, error) {
	if boifv == nil {
		return nil, nil
	}
	values := make([]*WithheldWorkerAttributedValue, len(boifv.OneOutInfererValues))
	for i, val := range boifv.OneOutInfererValues {
		converted, err := val.Convert()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert one out inferer value %d", i)
		}
		values[i] = converted
	}
	return &OneOutInfererForecasterValues{
		Forecaster:          boifv.Forecaster,
		OneOutInfererValues: values,
	}, nil
}

// Convert converts BoundedValueBundle to ValueBundle
func (bvb *BoundedValueBundle) Convert() (*ValueBundle, error) {
	if bvb == nil {
		return nil, nil
	}

	combinedValue, err := bvb.CombinedValue.ToDec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert combined value")
	}

	naiveValue, err := bvb.NaiveValue.ToDec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert naive value")
	}

	infererValues := make([]*WorkerAttributedValue, len(bvb.InfererValues))
	for i, val := range bvb.InfererValues {
		converted, err := val.Convert()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert inferer value %d", i)
		}
		infererValues[i] = converted
	}

	forecasterValues := make([]*WorkerAttributedValue, len(bvb.ForecasterValues))
	for i, val := range bvb.ForecasterValues {
		converted, err := val.Convert()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert forecaster value %d", i)
		}
		forecasterValues[i] = converted
	}

	oneOutInfererValues := make([]*WithheldWorkerAttributedValue, len(bvb.OneOutInfererValues))
	for i, val := range bvb.OneOutInfererValues {
		converted, err := val.Convert()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert one out inferer value %d", i)
		}
		oneOutInfererValues[i] = converted
	}

	oneOutForecasterValues := make([]*WithheldWorkerAttributedValue, len(bvb.OneOutForecasterValues))
	for i, val := range bvb.OneOutForecasterValues {
		converted, err := val.Convert()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert one out forecaster value %d", i)
		}
		oneOutForecasterValues[i] = converted
	}

	oneInForecasterValues := make([]*WorkerAttributedValue, len(bvb.OneInForecasterValues))
	for i, val := range bvb.OneInForecasterValues {
		converted, err := val.Convert()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert one in forecaster value %d", i)
		}
		oneInForecasterValues[i] = converted
	}

	oneOutInfererForecasterValues := make([]*OneOutInfererForecasterValues, len(bvb.OneOutInfererForecasterValues))
	for i, val := range bvb.OneOutInfererForecasterValues {
		converted, err := val.Convert()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert one out inferer forecaster value %d", i)
		}
		oneOutInfererForecasterValues[i] = converted
	}

	return &ValueBundle{
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
	}, nil
}

// Convert converts BoundedReputerValueBundle to ReputerValueBundle
func (brvb *BoundedReputerValueBundle) Convert() (*ReputerValueBundle, error) {
	if brvb == nil {
		return nil, nil
	}
	valueBundle, err := brvb.ValueBundle.Convert()
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert value bundle")
	}
	return &ReputerValueBundle{
		ValueBundle: valueBundle,
		Signature:   brvb.Signature,
		Pubkey:      brvb.Pubkey,
	}, nil
}
