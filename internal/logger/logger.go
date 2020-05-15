package logs

import (
	"os"

	"github.com/rs/zerolog"
)

var logger zerolog.Logger

//LogConfigGetter represents the logger configuration
type LogConfigGetter interface {
	IsJSON() bool
	Level() int
	ReportCaller() bool
}

//InitLogger initialize the logger with the passed config
func InitLogger(config LogConfigGetter) {
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

//Basic logging

//Trace logs a message at trace level.
func Trace(msg string) {
	logger.Trace().Msg(msg)
}

//Debug logs a message at debug level.
func Debug(msg string) {
	logger.Debug().Msg(msg)
}

//Info logs a message at info level.
func Info(msg string) {
	logger.Info().Msg(msg)
}

//Warn logs a message at warn level.
func Warn(msg string) {
	logger.Warn().Msg(msg)
}

//Error logs a message at error level.
func Error(msg string) {
	logger.Error().Msg(msg)
}

//Fatal logs a message at fatal level.
func Fatal(msg string) {
	logger.Fatal().Msg(msg)
}

//Panic logs a message at panic level.
func Panic(msg string) {
	logger.Panic().Msg(msg)
}

//Formatted logging

//Tracef logs a formatted message at trace level.
func Tracef(msg string, a ...interface{}) {
	logger.Trace().Msgf(msg, a...)
}

//Debugf logs a formatted message at debug level.
func Debugf(msg string, a ...interface{}) {
	logger.Debug().Msgf(msg, a...)
}

//Infof logs a formatted message at info level.
func Infof(msg string, a ...interface{}) {
	logger.Info().Msgf(msg, a...)
}

//Warnf logs a formatted message at warn level.
func Warnf(msg string, a ...interface{}) {
	logger.Warn().Msgf(msg, a...)
}

//Errorf logs a formatted message at error level.
func Errorf(msg string, a ...interface{}) {
	logger.Error().Msgf(msg, a...)
}

//Fatalf logs a formatted message at fatal level.
func Fatalf(msg string, a ...interface{}) {
	logger.Fatal().Msgf(msg, a...)
}

//Panicf logs a formatted message at panic level.
func Panicf(msg string, a ...interface{}) {
	logger.Panic().Msgf(msg, a...)
}
