package log

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"cosmossdk.io/log"
	"github.com/rs/zerolog"
)

//nolint:exhaustruct
func NewTestLogger(t zerolog.TestingLog) log.Logger {
	lvl := zerolog.DebugLevel
	cw := zerolog.ConsoleWriter{
		Out:           os.Stderr,
		TimeFormat:    time.RFC3339Nano,
		NoColor:       true,
		FieldsExclude: []string{"module"},
	}

	// Clean up formatter: raw message only
	cw.FormatLevel = func(i interface{}) string { return "" }
	cw.FormatCaller = func(i interface{}) string {
		if fullPath, ok := i.(string); ok {
			if cwd, err := os.Getwd(); err == nil {
				if relPath, err := filepath.Rel(cwd, fullPath); err == nil {
					return fmt.Sprintf("[%s]", relPath)
				}
			}
			return fmt.Sprintf("[%s]", fullPath)
		}
		return "[no-caller]"
	}
	cw.FormatTimestamp = func(i interface{}) string {
		return fmt.Sprintf("[%s]", i)
	}
	zerolog.TimestampFieldName = "time"
	return log.NewCustomLogger(zerolog.New(cw).With().CallerWithSkipFrameCount(4).Timestamp().Logger().Level(lvl))
}
