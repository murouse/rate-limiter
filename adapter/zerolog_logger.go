package adapter

import "github.com/rs/zerolog"

const prefix = "[RateLimiter] "

type ZeroLogLoggerAdapter struct {
	logger *zerolog.Logger
}

func NewZerologLogger(logger *zerolog.Logger) *ZeroLogLoggerAdapter {
	return &ZeroLogLoggerAdapter{logger: logger}
}

func (z *ZeroLogLoggerAdapter) Debugf(msg string, args ...any) {
	if len(args) > 0 {
		z.logger.Debug().Msgf(prefix+msg, args...)
		return
	}
	z.logger.Debug().Msg(prefix + msg)
}

func (z *ZeroLogLoggerAdapter) Infof(msg string, args ...any) {
	if len(args) > 0 {
		z.logger.Info().Msgf(prefix+msg, args...)
		return
	}
	z.logger.Info().Msg(prefix + msg)
}

func (z *ZeroLogLoggerAdapter) Warnf(msg string, args ...any) {
	if len(args) > 0 {
		z.logger.Warn().Msgf(prefix+msg, args...)
		return
	}
	z.logger.Warn().Msg(prefix + msg)
}

func (z *ZeroLogLoggerAdapter) Errorf(msg string, args ...any) {
	if len(args) > 0 {
		z.logger.Error().Msgf(prefix+msg, args...)
		return
	}
	z.logger.Error().Msg(prefix + msg)
}
