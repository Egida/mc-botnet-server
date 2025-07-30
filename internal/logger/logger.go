package logger

import (
	"github.com/charmbracelet/log"
	"os"
)

func New(prefix string, level log.Level) *log.Logger {
	return log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		Prefix:          prefix,
		Level:           level,
	})
}
