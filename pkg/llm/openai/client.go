package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/dskvich/ai-bot/pkg/domain"
)

const (
	apiURLChatCompletions = "https://api.openai.com/v1/chat/completions"
	apiURLAudioTranscribe = "https://api.openai.com/v1/audio/transcriptions"
	apiURLImageGeneration = "https://api.openai.com/v1/images/generations"

	modelWhisper       = "whisper-1"
	defaultMaxTokens   = 4096
	defaultResponseFmt = "b64_json"
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

func (c *client) CreateChatCompletion(ctx context.Context, chat *domain.Chat) (*domain.Message, error) {
	messages := make([]chatCompletionMessage, 0, len(chat.Messages)+1)

	if chat.SystemPrompt != "" {
		messages = append(messages, chatCompletionMessage{
			Role:    chatMessageRoleDeveloper,
			Content: []chatMessagePart{{Type: chatMessagePartTypeText, Text: chat.SystemPrompt}},
		})
	}

	for _, msg := range chat.Messages {
		if len(msg.ContentParts) == 1 && msg.ContentParts[0].Type == domain.ContentPartTypeText {
			// Simple text-only case
			messages = append(messages, chatCompletionMessage{Role: msg.Role, Content: msg.ContentParts[0].Data})
		} else {
			// Complex content case (multiple parts)
			var parts []chatMessagePart
			for _, content := range msg.ContentParts {
				switch content.Type {
				case domain.ContentPartTypeText:
					parts = append(parts, chatMessagePart{Type: chatMessagePartTypeText, Text: content.Data})
				case domain.ContentPartTypeImage:
					parts = append(parts, chatMessagePart{
						Type:     chatMessagePartTypeImageURL,
						ImageURL: &chatMessageImageURL{URL: content.Data},
					})
				default:
					return nil, errors.New("unsupported content type")
				}
			}
			messages = append(messages, chatCompletionMessage{Role: msg.Role, Content: parts})
		}
	}

	reqBody, err := json.Marshal(chatCompletionRequest{
		Model:     chat.TextModel,
		Messages:  messages,
		MaxTokens: defaultMaxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURLChatCompletions, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	respBody, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send chat completion request: %w", err)
	}

	var parsedResp chatCompletionResponse
	if err := json.Unmarshal(respBody, &parsedResp); err != nil {
		return nil, fmt.Errorf("failed to parse chat completion response: %w", err)
	}

	if len(parsedResp.Choices) == 0 {
		return nil, errors.New("no choices returned in response")
	}

	return &domain.Message{
		Role:         parsedResp.Choices[0].Message.Role,
		ContentParts: []domain.ContentPart{{Type: "text", Data: fmt.Sprint(parsedResp.Choices[0].Message.Content)}},
	}, nil
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

func (c *client) TranscribeAudio(ctx context.Context, audioFilePath string) (string, error) {
	body, contentType, err := createMultipartForm(audioFilePath, modelWhisper)
	if err != nil {
		return "", fmt.Errorf("failed to create multipart form: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURLAudioTranscribe, body)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)

	respBody, err := c.doRequest(req)
	if err != nil {
		return "", fmt.Errorf("failed to transcribe audio: %w", err)
	}

	var parsedResp struct {
		Text string `json:"text"`
	}

	if err := json.Unmarshal(respBody, &parsedResp); err != nil {
		return "", fmt.Errorf("failed to parse transcription response: %w", err)
	}

	return parsedResp.Text, nil
}

func createMultipartForm(filePath, model string) (*bytes.Buffer, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	fileWriter, err := writer.CreateFormFile("file", filePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(fileWriter, file); err != nil {
		return nil, "", fmt.Errorf("failed to copy file: %w", err)
	}

	if err := writer.WriteField("model", model); err != nil {
		return nil, "", fmt.Errorf("failed to write model field: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close writer: %w", err)
	}

	return &body, writer.FormDataContentType(), nil
}

func (c *client) GenerateImage(ctx context.Context, prompt string, model string) ([]byte, error) {
	if model == "" {
		model = domain.DallE2Model
	}

	size := size256x256
	quality := qualityStandard

	if model == domain.DallE3Model {
		size = size1024x1024
		quality = qualityHD
	}

	reqBody, err := json.Marshal(map[string]interface{}{
		"model":           model,
		"prompt":          prompt,
		"n":               1,
		"size":            size,
		"quality":         quality,
		"response_format": defaultResponseFmt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURLImageGeneration, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	respBody, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate image: %w", err)
	}

	var parsedResp struct {
		Data []struct {
			B64Json []byte `json:"b64_json"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &parsedResp); err != nil {
		return nil, fmt.Errorf("failed to parse image generation response: %w", err)
	}

	if len(parsedResp.Data) == 0 {
		return nil, errors.New("no image data returned")
	}

	return parsedResp.Data[0].B64Json, nil
}

func (c *client) GenerateImagePrompt(ctx context.Context, prompt string) (string, error) {
	prompt = fmt.Sprintf("User described an image idea. Write a detailed English prompt for AI image generation: %s", prompt)

	reqBody, err := json.Marshal(chatCompletionRequest{
		Model:     domain.Gpt4oMiniModel,
		Messages:  []chatCompletionMessage{{Role: "user", Content: prompt}},
		MaxTokens: defaultMaxTokens,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURLChatCompletions, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	respBody, err := c.doRequest(req)
	if err != nil {
		return "", fmt.Errorf("failed to send chat completion request: %w", err)
	}

	var parsedResp chatCompletionResponse
	if err := json.Unmarshal(respBody, &parsedResp); err != nil {
		return "", fmt.Errorf("failed to parse chat completion response: %w", err)
	}

	if len(parsedResp.Choices) == 0 {
		return "", errors.New("no choices returned in response")
	}

	return fmt.Sprint(parsedResp.Choices[0].Message.Content), nil
}
