package extractor

import (
	"context"

	"google.golang.org/grpc"
)

type Extractor struct{}

func New() *Extractor {
	return &Extractor{}
}

func (e *Extractor) ExtractRateKey(_ context.Context, _ interface{}, _ *grpc.UnaryServerInfo) (string, error) {
	return "rate-key", nil
}
