package logger

import (
	"os"

	"github.com/charmbracelet/log"
)

func NewLogger(prefix string, level log.Level) *log.Logger {
	return log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		Prefix:          prefix,
		Level:           level,
	})
}
