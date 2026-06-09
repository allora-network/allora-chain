package types

import (
	"cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	alloraMath "github.com/allora-network/allora-chain/math"
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
func NewLossBundleFromInput(brvb *InputReputerValueBundle) (*ReputerValueBundle, error) {
	if brvb == nil {
		return nil, ErrInvalidValue
	}
	valueBundle, err := NewValueBundleFromInput(brvb.ValueBundle)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert value bundle")
	}
	reputerValueBundle := &ReputerValueBundle{
		ValueBundle: valueBundle,
		Signature:   brvb.Signature,
		Pubkey:      brvb.Pubkey,
	}
	return reputerValueBundle, nil
}

// TODO: remove once the system completely moves to using NetworkInferenceBundle
func ValueBundleToNetworkInferenceBundle(vb *ValueBundle) *NetworkInferenceBundle {
	if vb == nil {
		return nil
	}

	const (
		label0Id   uint32 = 1
		label0Name string = "y"
	)

	var nonce int64
	if vb.ReputerRequestNonce != nil && vb.ReputerRequestNonce.ReputerNonce != nil {
		nonce = vb.ReputerRequestNonce.ReputerNonce.BlockHeight
	}

	//nolint:exhaustruct
	out := &NetworkInferenceBundle{
		TopicId: vb.TopicId,
		Nonce:   nonce,
		CombinedValue: []*LabeledValue{
			{LabelId: label0Id, LabelName: label0Name, Value: vb.CombinedValue},
		},
		NaiveValue: []*LabeledValue{
			{LabelId: label0Id, LabelName: label0Name, Value: vb.NaiveValue},
		},
	}

	// InfererValues: []*WorkerAttributedValue -> []*WorkerInference
	if n := len(vb.InfererValues); n > 0 {
		out.InfererValues = make([]*WorkerInference, n)
		for i, v := range vb.InfererValues {
			out.InfererValues[i] = &WorkerInference{
				Worker: v.Worker,
				Values: []*LabeledValue{
					{LabelId: label0Id, LabelName: label0Name, Value: v.Value},
				},
			}
		}
	}

	// ForecasterValues: []*WorkerAttributedValue -> []*WorkerInference
	if n := len(vb.ForecasterValues); n > 0 {
		out.ForecasterValues = make([]*WorkerInference, n)
		for i, v := range vb.ForecasterValues {
			out.ForecasterValues[i] = &WorkerInference{
				Worker: v.Worker,
				Values: []*LabeledValue{
					{LabelId: label0Id, LabelName: label0Name, Value: v.Value},
				},
			}
		}
	}

	// OneOutInfererValues: []*WithheldWorkerAttributedValue -> []*OneOutInfererValue
	if n := len(vb.OneOutInfererValues); n > 0 {
		out.OneOutInfererValues = make([]*OneOutInfererValue, n)
		for i, v := range vb.OneOutInfererValues {
			out.OneOutInfererValues[i] = &OneOutInfererValue{
				WithheldInferer: v.Worker,
				CombinedInference: []*LabeledValue{
					{LabelId: label0Id, LabelName: label0Name, Value: v.Value},
				},
			}
		}
	}

	// OneOutForecasterValues: []*WithheldWorkerAttributedValue -> []*OneOutForecasterValue
	if n := len(vb.OneOutForecasterValues); n > 0 {
		out.OneOutForecasterValues = make([]*OneOutForecasterValue, n)
		for i, v := range vb.OneOutForecasterValues {
			out.OneOutForecasterValues[i] = &OneOutForecasterValue{
				WithheldForecaster: v.Worker,
				CombinedInference: []*LabeledValue{
					{LabelId: label0Id, LabelName: label0Name, Value: v.Value},
				},
			}
		}
	}

	// OneInForecasterValues: []*WorkerAttributedValue -> []*OneInForecasterValue
	if n := len(vb.OneInForecasterValues); n > 0 {
		out.OneInForecasterValues = make([]*OneInForecasterValue, n)
		for i, v := range vb.OneInForecasterValues {
			out.OneInForecasterValues[i] = &OneInForecasterValue{
				Forecaster: v.Worker,
				CombinedInference: []*LabeledValue{
					{LabelId: label0Id, LabelName: label0Name, Value: v.Value},
				},
			}
		}
	}

	// OneOutInfererForecasterValues: []*OneOutInfererForecasterValues -> []*OneOutInfererForecasterValue
	// Old structure: per forecaster -> list of withheld-inferer values (but no explicit withheld-inferer in output).
	// Here we emit one record per (forecaster, withheldInferer) pair.
	if n := len(vb.OneOutInfererForecasterValues); n > 0 {
		// pre-size approximately: sum of per-forecaster rows
		total := 0
		for _, row := range vb.OneOutInfererForecasterValues {
			total += len(row.OneOutInfererValues)
		}
		if total > 0 {
			out.OneOutInfererForecasterValues = make([]*OneOutInfererForecasterValue, 0, total)
			for _, row := range vb.OneOutInfererForecasterValues {
				fc := row.Forecaster
				for _, cell := range row.OneOutInfererValues {
					out.OneOutInfererForecasterValues = append(out.OneOutInfererForecasterValues,
						&OneOutInfererForecasterValue{
							Forecaster:      fc,
							WithheldInferer: cell.Worker,
							CombinedInference: []*LabeledValue{
								{LabelId: label0Id, LabelName: label0Name, Value: cell.Value},
							},
						},
					)
				}
			}
		}
	}

	return out
}

// ConvertInferenceValuesFromProto converts a stored Inference proto into the internal
// InferenceValues representation used by math code.
func ConvertInferenceValuesFromProto(
	topicArity TopicOutputArity,
	labels []*TopicLabel,
	inf *Inference,
) (InferenceValues, error) {
	if inf == nil {
		return InferenceValues{}, errors.Wrap(sdkerrors.ErrInvalidRequest, "inference is nil")
	}

	switch topicArity {
	case TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE:
		if len(inf.Values) != 1 {
			return InferenceValues{}, errors.Wrap(sdkerrors.ErrInvalidRequest, "single-arity inference accepts exactly one value")
		}

		dec := inf.Values[0]

		if dec.IsNaN() || !dec.IsFinite() {
			return InferenceValues{}, errors.Wrap(sdkerrors.ErrInvalidRequest, "invalid scalar inference value")
		}

		return alloraMath.DecArray{dec}, nil
	case TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI:
		regLen := len(labels)
		if regLen == 0 {
			return InferenceValues{}, errors.Wrap(sdkerrors.ErrLogic, "epoch label registry is empty for multi-arity")
		}
		if len(inf.Values) == 0 {
			return InferenceValues{}, errors.Wrap(sdkerrors.ErrInvalidRequest, "multi-arity inference requires values")
		}
		if len(inf.Values) > regLen {
			return InferenceValues{}, errors.Wrapf(
				sdkerrors.ErrLogic,
				"multi-arity inference length exceeds registry: got=%d reg=%d",
				len(inf.Values), regLen,
			)
		}

		zero := alloraMath.ZeroDec()
		out := make(alloraMath.DecArray, regLen)
		for i := range out {
			out[i] = zero
		}
		copy(out, inf.Values)
		if err := ValidateInferenceValues(out, labels); err != nil {
			return InferenceValues{}, err
		}
		return out, nil
	default:
		return InferenceValues{}, errors.Wrap(sdkerrors.ErrInvalidRequest, "output_arity is invalid")
	}
}

// ConvertInferenceValuesToLabeledValues converts the internal InferenceValues representation into
// a slice of LabeledValue suitable for RPC responses or event emission.
func ConvertInferenceValuesToLabeledValues(iv InferenceValues, reg *EpochLabelRegistry) ([]*LabeledValue, error) {
	if reg == nil {
		return nil, errors.Wrap(sdkerrors.ErrLogic, "label registry can not be nil")
	}
	want := len(reg.GetLabels())
	if len(iv) != want {
		return nil, errors.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"inference values length mismatch: got=%d want=%d",
			len(iv), want,
		)
	}
	out := make([]*LabeledValue, 0, len(iv))
	for i, v := range iv {
		lbl := reg.GetLabels()[i]
		if lbl == nil {
			return nil, errors.Wrapf(sdkerrors.ErrLogic, "nil label in registry at idx=%d", i)
		}
		out = append(out, &LabeledValue{
			LabelId:   lbl.Id,
			LabelName: lbl.Name,
			Value:     v,
		})
	}
	return out, nil
}

func ConvertLabeledValuesToDecArray(in []*LabeledValue) alloraMath.DecArray {
	out := make(alloraMath.DecArray, len(in))
	for i := range in {
		out[i] = in[i].Value
	}
	return out
}
