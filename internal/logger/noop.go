package logger

// NoopLogger is a Logger implementation that discards all log messages.
//
// Useful as a default logger or in tests.
type NoopLogger struct{}

// NewNoopLogger returns a no-op Logger implementation.
func NewNoopLogger() *NoopLogger {
	return &NoopLogger{}
}

// Debugf implements Logger and does nothing.
func (n *NoopLogger) Debugf(_ string, _ ...any) {}

// Infof implements Logger and does nothing.
func (n *NoopLogger) Infof(_ string, _ ...any) {}

// Warnf implements Logger and does nothing.
func (n *NoopLogger) Warnf(_ string, _ ...any) {}

// Errorf implements Logger and does nothing.
func (n *NoopLogger) Errorf(_ string, _ ...any) {}
