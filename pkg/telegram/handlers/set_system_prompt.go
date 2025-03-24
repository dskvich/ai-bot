package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/dskvich/ai-bot/pkg/domain"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type SetSystemPromptChatProvider interface {
	Get(ctx context.Context, chatID int64, topicID int) (*domain.Chat, error)
	Save(ctx context.Context, chat *domain.Chat) error
}

type SetSystemPromptStateClearer interface {
	Clear(chatID int64, topicID int)
}

func SetSystemPrompt(
	chatProvider SetSystemPromptChatProvider,
	stateClearer SetSystemPromptStateClearer,
) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		chatID := update.Message.Chat.ID
		topicID := update.Message.MessageThreadID
		prompt := update.Message.Text

		chat, err := chatProvider.Get(ctx, chatID, topicID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				chat = domain.NewChat(chatID, topicID)
			} else {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:          update.Message.Chat.ID,
					MessageThreadID: update.Message.MessageThreadID,
					Text:            fmt.Sprintf("❌ Не удалось получить чат: %s", err),
				})
				return
			}
		}

		chat.SystemPrompt = prompt

		if err = chatProvider.Save(ctx, chat); err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          chatID,
				MessageThreadID: topicID,
				Text:            fmt.Sprintf("❌ Не удалось сохранить чат: %s", err),
			})
			return
		}

		stateClearer.Clear(chatID, topicID)

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:          chatID,
			MessageThreadID: topicID,
			Text:            "✅ Системная инструкция установлена: " + prompt,
		})
	}
}
