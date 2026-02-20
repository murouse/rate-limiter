package extender

import (
	"context"

	"google.golang.org/grpc"
)

type DefaultRateKeyExtender struct{}

func NewDefaultRateKeyExtender() *DefaultRateKeyExtender {
	return &DefaultRateKeyExtender{}
}

func (e *DefaultRateKeyExtender) ExtendRateKey(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo) (string, error) {
	return "custom-rate-key", nil
}
