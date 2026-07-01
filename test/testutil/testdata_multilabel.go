package testutil

import (
	_ "embed"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	alloraMath "github.com/allora-network/allora-chain/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// simulatorOutputMultilabelCSV is the raw output of the allora-simulator
// for a multi-label (multilabel) topic. It is the source of truth for
// multilabel inference-synthesis tests, the same role the inline
// simulatorEpochs map plays for the single-label regression tests.
//
// Layout: one header row, then one row per epoch. The first two columns are
// "epoch" (int) and "time" (timestamp); "target" is a bracketed label vector.
// Those three are non-numeric and are not exposed through the getters. Every
// other column is a scalar decimal. Per-class quantities are spread across
// columns suffixed _label_0, _label_1, _label_2.
//
//go:embed simulator_output_classification.csv
var simulatorOutputMultilabelCSV string

// Columns that are present in the CSV but are not numeric and therefore are
// not exposed through the value getter.
var multilabelNonNumericColumns = map[string]struct{}{
	"epoch":  {},
	"time":   {},
	"target": {},
}

// GetSimulatedValuesGetterForMultilabelEpochs parses the embedded
// multilabel simulator output and returns, for each epoch, a getter that
// resolves a column header to its decimal value for that epoch.
//
// It mirrors GetSimulatedValuesGetterForEpochs (the single-label regression
// fixture) but reads its data from an embedded CSV rather than from hardcoded
// string literals. The returned getter panics on an unknown or non-numeric
// header: asking for a column that does not exist is a test bug, consistent
// with the Must-style accessors used elsewhere in these tests.
//
// All parsing is done eagerly here under require, so a malformed CSV fails the
// calling test cleanly with row/column context instead of panicking deep
// inside a getter closure later.
func GetSimulatedValuesGetterForMultilabelEpochs(
	t *testing.T,
) map[int]func(header string) alloraMath.Dec {
	t.Helper()

	reader := csv.NewReader(strings.NewReader(simulatorOutputMultilabelCSV))
	// The CSV is a fixed, known-width grid; let the reader enforce that every
	// row matches the header width.
	reader.FieldsPerRecord = 0

	records, err := reader.ReadAll()
	require.NoError(t, err, "failed to parse embedded multilabel simulator CSV")
	require.GreaterOrEqual(t, len(records), 2,
		"multilabel simulator CSV must contain a header row and at least one epoch row")

	header := records[0]
	headerIndex := make(map[string]int, len(header))
	for i, name := range header {
		name = strings.TrimSpace(name)
		_, dup := headerIndex[name]
		require.False(t, dup, "duplicate column %q in multilabel simulator CSV header", name)
		headerIndex[name] = i
	}

	epochColIdx, ok := headerIndex["epoch"]
	require.True(t, ok, "multilabel simulator CSV header is missing the 'epoch' column")

	getters := make(map[int]func(header string) alloraMath.Dec, len(records)-1)

	for rowNum, row := range records[1:] {
		// rowNum is 0-based over data rows; +2 makes it the 1-based line number
		// in the file (1 header + 1-based data), which is what a human sees.
		lineNo := rowNum + 2

		epoch, err := strconv.Atoi(strings.TrimSpace(row[epochColIdx]))
		require.NoError(t, err,
			"line %d: failed to parse epoch id %q", lineNo, row[epochColIdx])

		// Eagerly parse every numeric cell so a bad value fails here, with
		// context, rather than inside a getter closure during a later test.
		values := make(map[string]alloraMath.Dec, len(header))
		for name, colIdx := range headerIndex { //nolint:maprange // test fixture parsing: each cell is parsed and stored under its own key, order does not affect the result
			if _, skip := multilabelNonNumericColumns[name]; skip {
				continue
			}
			raw := strings.TrimSpace(row[colIdx])
			dec, err := alloraMath.NewDecFromString(raw)
			require.NoError(t, err,
				"line %d: column %q: cannot parse decimal %q", lineNo, name, raw)
			values[name] = dec
		}

		_, dup := getters[epoch]
		require.False(t, dup,
			"line %d: duplicate epoch id %d in multilabel simulator CSV", lineNo, epoch)

		// Capture values for this epoch in the closure.
		epochValues := values
		getters[epoch] = func(header string) alloraMath.Dec {
			header = strings.TrimSpace(header)
			dec, ok := epochValues[header]
			if !ok {
				if _, isNonNumeric := multilabelNonNumericColumns[header]; isNonNumeric {
					panic("multilabel simulator getter: column '" + header +
						"' is non-numeric and not exposed")
				}
				panic("multilabel simulator getter: unknown column '" + header + "'")
			}
			return dec
		}
	}

	return getters
}

// GetMultilabelInferencesFromCsv builds per-inferer inferences where each
// inference carries one value per class label, pulled from the simulator's
// inference_<i>_label_<k> columns. It is the multilabel counterpart of
// GetInferencesFromCsv (which produces single-value inferences).
//
// labelCount is the number of class labels; the column for inferer i, label k
// is "inference_<i>_label_<k>". Values are ordered by label index, so
// Values[k] corresponds to label k.
func GetMultilabelInferencesFromCsv(
	topicId uint64,
	blockHeight int64,
	inferers []string,
	labelCount int,
	epochGet func(header string) alloraMath.Dec,
) (emissionstypes.Inferences, error) {
	if len(inferers) != 5 {
		return emissionstypes.Inferences{}, fmt.Errorf("expected 5 inferers, got %d", len(inferers))
	}
	if labelCount < 1 {
		return emissionstypes.Inferences{}, fmt.Errorf("expected at least 1 label, got %d", labelCount)
	}

	out := make([]*emissionstypes.Inference, 0, len(inferers))
	for i, inferer := range inferers {
		values := make([]alloraMath.Dec, 0, labelCount)
		for k := 0; k < labelCount; k++ {
			header := fmt.Sprintf("inference_%d_label_%d", i, k)
			values = append(values, epochGet(header))
		}
		out = append(out, &emissionstypes.Inference{
			Inferer:     inferer,
			Values:      values,
			TopicId:     topicId,
			BlockHeight: blockHeight,
			ExtraData:   nil,
			Proof:       "",
		})
	}
	return emissionstypes.Inferences{Inferences: out}, nil
}
