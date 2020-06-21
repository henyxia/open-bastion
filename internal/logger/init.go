package logger

import (
	"os"

	"github.com/rs/zerolog"
)

// initLogger initialize the logger with the passed config
func initLogger(config LogConfigGetter) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	if config.IsJSON() {
		logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	} else {
		logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
	}

	lvl := config.Level()

	if lvl < -1 {
		lvl = -1
	}

	if lvl > 5 {
		lvl = 5
	}

	zerolog.SetGlobalLevel(zerolog.Level(lvl))

	if config.ReportCaller() {
		logger = logger.With().Caller().Logger()
	}
}
