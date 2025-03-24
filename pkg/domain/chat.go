package domain

import "time"

const DefaultTTL = 15 * time.Minute

type Chat struct {
	ID           int64
	TopicID      int
	TextModel    string
	ImageModel   string
	TTL          time.Duration
	SystemPrompt string
	Messages     []Message
	LastUpdate   time.Time
}

func NewChat(chatID int64, topicID int) *Chat {
	return &Chat{
		ID:         chatID,
		TopicID:    topicID,
		TextModel:  Gpt4oMiniModel,
		ImageModel: DallE2Model,
		TTL:        DefaultTTL,
	}
}

type Message struct {
	Role         string
	ContentParts []ContentPart
}

const MessageRoleUser = "user"

type ContentPart struct {
	Type ContentPartType
	Data string
}

type ContentPartType string

const (
	ContentPartTypeText  ContentPartType = "text"
	ContentPartTypeImage ContentPartType = "image"
)
