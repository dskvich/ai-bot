package handlers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dskvich/ai-bot/pkg/domain"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type ChatClearer interface {
	DeleteMessages(ctx context.Context, chat *domain.Chat) error
}

func ClearChat(clearer ChatClearer) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		slog.InfoContext(ctx, "Clearing chat")

		chatID := update.Message.Chat.ID
		topicID := update.Message.MessageThreadID

		if err := clearer.DeleteMessages(ctx, &domain.Chat{
			ID:      chatID,
			TopicID: topicID,
		}); err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          chatID,
				MessageThreadID: topicID,
				Text:            fmt.Sprintf("❌ Не удалось очистить чат: %+v", err),
			})
			return
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:          chatID,
			MessageThreadID: topicID,
			Text:            "🧹 История очищена! Начните новый чат. 🚀",
		})
	}
}
