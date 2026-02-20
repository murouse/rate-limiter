package extractor

import (
	"context"

	"google.golang.org/grpc"
)

type DefaultExtractor struct{}

func NewDefaultExtractor() *DefaultExtractor {
	return &DefaultExtractor{}
}

func (e *DefaultExtractor) ExtractRateKey(_ context.Context, _ interface{}, _ *grpc.UnaryServerInfo) (string, error) {
	return "rate-key", nil
}
