package extractor

import (
	"context"

	"google.golang.org/grpc"
)

type DefaultExtractor struct{}

func NewDefaultExtractor() *DefaultExtractor {
	return &DefaultExtractor{}
}

func (e *DefaultExtractor) ExtractRateKey(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, attrs map[string]string) (string, error) {
	return "rate-key", nil
}
