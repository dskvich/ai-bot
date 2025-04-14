package replicate

import "github.com/dskvich/ai-bot/pkg/domain"

const (
	FluxProUltra11Model = "black-forest-labs/flux-1.1-pro-ultra"
)

var ModelToReplicateModel = map[string]string{
	domain.FluxProUltra11: FluxProUltra11Model,
}

const DefaultAspectRatio = "3:2"
