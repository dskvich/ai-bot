package replicate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	apiURL                 = "https://api.replicate.com/v1"
	apiURLPredictions      = apiURL + "/predictions"
	apiURLModels           = apiURL + "/models"
	defaultPollingTimeout  = 60 * time.Second
	defaultPollingInterval = 1 * time.Second
)

type client struct {
	token string
	hc    *http.Client
}

func NewClient(token string) (*client, error) {
	if token == "" {
		return nil, errors.New("token cannot be empty")
	}
	return &client{
		token: token,
		hc:    &http.Client{},
	}, nil
}

func (c *client) GenerateImage(ctx context.Context, prompt string, model string) ([]byte, error) {
	// Map domain model to Replicate model
	replicateModel, ok := ModelToReplicateModel[model]
	if !ok {
		return nil, fmt.Errorf("unsupported model: %s", model)
	}

	predictionURL := fmt.Sprintf("%s/%s/predictions", apiURLModels, replicateModel)

	input := FluxInput{
		Prompt:      prompt,
		AspectRatio: DefaultAspectRatio,
	}

	reqBody, err := json.Marshal(CreatePredictionRequest{
		Input: map[string]interface{}{
			"prompt":       input.Prompt,
			"aspect_ratio": input.AspectRatio,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, predictionURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "wait") // Wait for the prediction to complete

	respBody, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create prediction: %w", err)
	}

	var prediction ReplicatePrediction
	if err := json.Unmarshal(respBody, &prediction); err != nil {
		return nil, fmt.Errorf("failed to parse prediction response: %w", err)
	}

	// If the prediction is not completed, poll for the result
	if prediction.Status != PredictionStatusSucceeded {
		prediction, err = c.pollPrediction(ctx, prediction.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to poll prediction: %w", err)
		}
	}

	if prediction.Status != PredictionStatusSucceeded {
		return nil, fmt.Errorf("prediction failed with status %s: %s", prediction.Status, prediction.Error)
	}

	if len(prediction.Output) == 0 {
		return nil, errors.New("no output returned")
	}

	imageData, err := c.downloadImage(ctx, prediction.Output)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}

	return imageData, nil
}

func (c *client) doRequest(req *http.Request) ([]byte, error) {
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return respBody, nil
}

func (c *client) pollPrediction(ctx context.Context, predictionID string) (ReplicatePrediction, error) {
	var prediction ReplicatePrediction

	// Create a timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, defaultPollingTimeout)
	defer cancel()

	ticker := time.NewTicker(defaultPollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return prediction, errors.New("polling timed out")
		case <-ticker.C:
			// Get the prediction status
			predictionURL := fmt.Sprintf("%s/%s", apiURLPredictions, predictionID)
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, predictionURL, nil)
			if err != nil {
				return prediction, fmt.Errorf("failed to create HTTP request: %w", err)
			}

			respBody, err := c.doRequest(req)
			if err != nil {
				return prediction, fmt.Errorf("failed to get prediction: %w", err)
			}

			if err := json.Unmarshal(respBody, &prediction); err != nil {
				return prediction, fmt.Errorf("failed to parse prediction response: %w", err)
			}

			// Check if the prediction is complete
			if prediction.Status == PredictionStatusSucceeded ||
				prediction.Status == PredictionStatusFailed ||
				prediction.Status == PredictionStatusCanceled {
				return prediction, nil
			}
		}
	}
}

// downloadImage downloads an image from a URL
func (c *client) downloadImage(ctx context.Context, imageURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	return imageData, nil
}
