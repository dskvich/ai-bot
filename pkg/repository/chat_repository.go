package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/dskvich/ai-bot/pkg/domain"
	"github.com/uptrace/bun"
)

type chatRepository struct {
	db *bun.DB
}

func NewChatRepository(db *bun.DB) *chatRepository {
	return &chatRepository{db: db}
}

func (c *chatRepository) Save(ctx context.Context, chat *domain.Chat) error {
	chat.LastUpdate = time.Now()

	_, err := c.db.NewInsert().
		Model(chat).
		On("CONFLICT (id, topic_id) DO UPDATE").
		Set("text_model = EXCLUDED.text_model").
		Set("image_model = EXCLUDED.image_model").
		Set("ttl = EXCLUDED.ttl").
		Set("system_prompt = EXCLUDED.system_prompt").
		Set("messages = EXCLUDED.messages").
		Set("last_update = EXCLUDED.last_update").
		Exec(ctx)
	return err
}

func (c *chatRepository) Get(ctx context.Context, chatID int64, topicID int) (*domain.Chat, error) {
	var chat domain.Chat

	err := c.db.NewSelect().
		Model(&chat).
		Where("id = ?", chatID).
		Where("topic_id = ?", topicID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("fetching chat: %w", err)
	}

	return &chat, nil
}

func (c *chatRepository) DeleteMessages(ctx context.Context, chat *domain.Chat) error {
	_, err := c.db.NewUpdate().
		Model((*domain.Chat)(nil)).
		Set("messages = null").
		Where("id = ?", chat.ID).
		Where("topic_id = ?", chat.TopicID).
		Exec(ctx)
	return err
}
