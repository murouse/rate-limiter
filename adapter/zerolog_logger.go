package adapter

import "github.com/rs/zerolog"

const prefix = "[RateLimiter] "

// ZeroLogLoggerAdapter adapts a logger
// to the rate limiter Logger interface.
type ZeroLogLoggerAdapter struct {
	logger *zerolog.Logger
}

// NewZerologLogger wraps a zerolog.Logger
// into a Logger-compatible adapter.
func NewZerologLogger(logger *zerolog.Logger) *ZeroLogLoggerAdapter {
	return &ZeroLogLoggerAdapter{logger: logger}
}

// Debugf logs a debug-level message.
func (z *ZeroLogLoggerAdapter) Debugf(msg string, args ...any) {
	if len(args) > 0 {
		z.logger.Debug().Msgf(prefix+msg, args...)
		return
	}
	z.logger.Debug().Msg(prefix + msg)
}

// Infof logs an info-level message.
func (z *ZeroLogLoggerAdapter) Infof(msg string, args ...any) {
	if len(args) > 0 {
		z.logger.Info().Msgf(prefix+msg, args...)
		return
	}
	z.logger.Info().Msg(prefix + msg)
}

// Warnf logs a warning-level message.
func (z *ZeroLogLoggerAdapter) Warnf(msg string, args ...any) {
	if len(args) > 0 {
		z.logger.Warn().Msgf(prefix+msg, args...)
		return
	}
	z.logger.Warn().Msg(prefix + msg)
}

// Errorf logs an error-level message.
func (z *ZeroLogLoggerAdapter) Errorf(msg string, args ...any) {
	if len(args) > 0 {
		z.logger.Error().Msgf(prefix+msg, args...)
		return
	}
	z.logger.Error().Msg(prefix + msg)
}
