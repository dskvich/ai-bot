package openai

type chatCompletionRequest struct {
	Model     string                  `json:"model"`
	Messages  []chatCompletionMessage `json:"messages"`
	MaxTokens int                     `json:"max_tokens"`
}

type chatCompletionResponse struct {
	Choices []chatCompletionChoice `json:"choices"`
}

type chatCompletionChoice struct {
	Message chatCompletionMessage `json:"message"`
}

type chatCompletionMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type chatMessagePartType string

const (
	chatMessagePartTypeText     chatMessagePartType = "text"
	chatMessagePartTypeImageURL chatMessagePartType = "image_url"
)

type chatMessagePart struct {
	Type     chatMessagePartType  `json:"type,omitempty"`
	Text     string               `json:"text,omitempty"`
	ImageURL *chatMessageImageURL `json:"image_url,omitempty"`
}

type chatMessageImageURL struct {
	URL string `json:"url,omitempty"`
}

const chatMessageRoleDeveloper = "developer"

type imageSize string

const (
	size256x256   imageSize = "256x256"
	size512x512   imageSize = "512x512"
	size1024x1024 imageSize = "1024x1024"
	size1024x1792 imageSize = "1024x1792"
	size1792x1024 imageSize = "1792x1024"
)

type imageQuality string

const (
	qualityStandard imageQuality = "standard"
	qualityHD       imageQuality = "hd"
)
