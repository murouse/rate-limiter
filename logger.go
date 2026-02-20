package ratelimiter

type noopLogger struct{}

func (n *noopLogger) Debugf(_ string, _ ...any) {}

func (n *noopLogger) Infof(_ string, _ ...any) {}

func (n *noopLogger) Warnf(_ string, _ ...any) {}

func (n *noopLogger) Errorf(_ string, _ ...any) {}
