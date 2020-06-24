package logger

import (
	"context"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ClientInfoGetter interface {
	GetUser() string
	GetIp() string
	GetPublicKeyFingerprint() string
	GetCommand() string

	GetBackendCommand() string
	GetBackendUser() string
	GetBackendHost() string
	GetBackendPort() int
}

//InitContextLogger attaches a logger to the context. This method should be called after the logger initialization
//as it inherits its settings.
func InitContextLogger(ctx context.Context) context.Context {
	l := logger.With().Logger()

	return l.WithContext(ctx)
}

//UpdateClientLogCtx updates the logger context with the client information.
func UpdateClientLogCtx(ctx context.Context, c ClientInfoGetter) context.Context {
	l := log.Ctx(ctx)

	l.UpdateContext(func(zCtx zerolog.Context) zerolog.Context {
		return zCtx.Str("user", c.GetUser()).
			Str("ip", c.GetIp()).
			Str("backendPublicKeyFingerprint", c.GetPublicKeyFingerprint()).
			Str("command", c.GetCommand()).
			Str("backendCommand", c.GetBackendCommand()).
			Str("backendUser", c.GetBackendUser()).
			Str("backendHost", c.GetBackendHost()).
			Int("backendPort", c.GetBackendPort())
	})

	return ctx
}

//Basic context logging

//TraceWithCtx logs a message at trace level.
func TraceWithCtx(ctx context.Context, msg string) {
	log.Ctx(ctx).Trace().Msg(msg)
}

//DebugWithCtx logs a message at debug level.
func DebugWithCtx(ctx context.Context, msg string) {
	log.Ctx(ctx).Debug().Msg(msg)
}

//InfoWithCtx logs a message at info level.
func InfoWithCtx(ctx context.Context, msg string) {
	log.Ctx(ctx).Info().Msg(msg)
}

//WarnWithCtx logs a message at warn level.
func WarnWithCtx(ctx context.Context, msg string) {
	log.Ctx(ctx).Warn().Msg(msg)
}

//WarnWithCtxWithErr logs a message at warn level.
func WarnWithCtxWithErr(ctx context.Context, err error, msg string) {
	log.Ctx(ctx).Warn().Err(err).Msg(msg)
}

//ErrorWithCtx logs a message at error level.
func ErrorWithCtx(ctx context.Context, msg string) {
	log.Ctx(ctx).Error().Msg(msg)
}

//ErrorWithCtxWithErr logs a message at error level.
func ErrorWithCtxWithErr(ctx context.Context, err error, msg string) {
	log.Ctx(ctx).Error().Err(err).Msg(msg)
}

//FatalWithCtx logs a message at fatal level.
func FatalWithCtx(ctx context.Context, msg string) {
	log.Ctx(ctx).Fatal().Msg(msg)
}

//FatalWithCtxWithErr logs a message at fatal level.
func FatalWithCtxWithErr(ctx context.Context, err error, msg string) {
	log.Ctx(ctx).Fatal().Err(err).Msg(msg)
}

//PanicWithCtx logs a message at panic level.
func PanicWithCtx(ctx context.Context, msg string) {
	log.Ctx(ctx).Panic().Msg(msg)
}

//PanicWithCtxWithErr logs a message at panic level.
func PanicWithCtxWithErr(ctx context.Context, err error, msg string) {
	log.Ctx(ctx).Panic().Err(err).Msg(msg)
}

//Formatted logging

//TracefWithCtx logs a formatted message at trace level.
func TracefWithCtx(ctx context.Context, msg string, a ...interface{}) {
	log.Ctx(ctx).Trace().Msgf(msg, a...)
}

//DebugfWithCtx logs a formatted message at debug level.
func DebugfWithCtx(ctx context.Context, msg string, a ...interface{}) {
	log.Ctx(ctx).Debug().Msgf(msg, a...)
}

//InfofWithCtx logs a formatted message at info level.
func InfofWithCtx(ctx context.Context, msg string, a ...interface{}) {
	log.Ctx(ctx).Info().Msgf(msg, a...)
}

//WarnfWithCtx logs a formatted message at warn level.
func WarnfWithCtx(ctx context.Context, msg string, a ...interface{}) {
	log.Ctx(ctx).Warn().Msgf(msg, a...)
}

//WarnfWithCtxWithErr logs a formatted message at warn level.
func WarnfWithCtxWithErr(ctx context.Context, err error, msg string, a ...interface{}) {
	log.Ctx(ctx).Warn().Err(err).Msgf(msg, a...)
}

//ErrorfWithCtx logs a formatted message at error level.
func ErrorfWithCtx(ctx context.Context, msg string, a ...interface{}) {
	log.Ctx(ctx).Error().Msgf(msg, a...)
}

//ErrorfWithCtxWithErr logs a formatted message at error level.
func ErrorfWithCtxWithErr(ctx context.Context, err error, msg string, a ...interface{}) {
	log.Ctx(ctx).Error().Err(err).Msgf(msg, a...)
}

//FatalfWithCtx logs a formatted message at fatal level.
func FatalfWithCtx(ctx context.Context, msg string, a ...interface{}) {
	log.Ctx(ctx).Fatal().Msgf(msg, a...)
}

//FatalfWithCtxWithErr logs a formatted message at fatal level.
func FatalfWithCtxWithErr(ctx context.Context, err error, msg string, a ...interface{}) {
	log.Ctx(ctx).Fatal().Err(err).Msgf(msg, a...)
}

//PanicfWithCtx logs a formatted message at panic level.
func PanicfWithCtx(ctx context.Context, msg string, a ...interface{}) {
	log.Ctx(ctx).Panic().Msgf(msg, a...)
}

//PanicfWithCtxWithErr logs a formatted message at panic level.
func PanicfWithCtxWithErr(ctx context.Context, err error, msg string, a ...interface{}) {
	log.Ctx(ctx).Panic().Err(err).Msgf(msg, a...)
}
