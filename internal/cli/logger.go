package cli

import (
	"os"

	"github.com/kamichidu/go-hariti"
	"github.com/kamichidu/go-hariti/internal/logger"
)

func NewCLILogger(verbose bool) hariti.Logger {
	level := hariti.LevelInfo
	if verbose {
		level = hariti.LevelDebug
	}
	return logger.NewLogger(os.Stderr, logger.LoggerOptions{
		Level: level,
	})
}
