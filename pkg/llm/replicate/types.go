package replicate

import "time"

type ReplicatePrediction struct {
	ID          string                 `json:"id"`
	Version     string                 `json:"version"`
	Logs        string                 `json:"logs"`
	Error       string                 `json:"error"`
	Status      string                 `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	CompletedAt time.Time              `json:"completed_at,omitempty"`
	URLs        map[string]string      `json:"urls"`
	Metrics     map[string]interface{} `json:"metrics"`
	Input       map[string]interface{} `json:"input"`
	Output      string                 `json:"output"`
	StartedAt   time.Time              `json:"started_at,omitempty"`
}

type CreatePredictionRequest struct {
	Input map[string]interface{} `json:"input"`
}

type FluxInput struct {
	Prompt      string `json:"prompt"`
	AspectRatio string `json:"aspect_ratio"`
}

const (
	PredictionStatusStarting   = "starting"
	PredictionStatusProcessing = "processing"
	PredictionStatusSucceeded  = "succeeded"
	PredictionStatusFailed     = "failed"
	PredictionStatusCanceled   = "canceled"
)
