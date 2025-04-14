package llm

import (
	"context"
	"fmt"
)

type ImageGenerator interface {
	GenerateImage(ctx context.Context, prompt string, model string) ([]byte, error)
}

type MultiProviderImageClient struct {
	providers map[string]ImageGenerator
}

func NewMultiProviderImageClient(providers map[string]ImageGenerator) *MultiProviderImageClient {
	return &MultiProviderImageClient{
		providers: providers,
	}
}

func (c *MultiProviderImageClient) GenerateImage(ctx context.Context, prompt string, model string) ([]byte, error) {
	provider, ok := c.providers[model]
	if !ok {
		return nil, fmt.Errorf("no provider found for model: %s", model)
	}

	return provider.GenerateImage(ctx, prompt, model)
}
