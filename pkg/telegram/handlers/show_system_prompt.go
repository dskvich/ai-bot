package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/dskvich/ai-bot/pkg/domain"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type ShowSystemPromptChatProvider interface {
	Get(ctx context.Context, chatID int64, topicID int) (*domain.Chat, error)
}

func ShowSystemPrompt(chatProvider ShowSystemPromptChatProvider) bot.HandlerFunc {
	const editButtonText = "Редактировать"

	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		chatID := update.Message.Chat.ID
		topicID := update.Message.MessageThreadID

		chat, err := chatProvider.Get(ctx, chatID, topicID)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          chatID,
				MessageThreadID: topicID,
				Text:            fmt.Sprintf("❌ Не удалось получить чат: %s", err),
			})
			return
		}

		prompt := "Отсутсвует"
		if chat != nil && chat.SystemPrompt != "" {
			prompt = chat.SystemPrompt
		}

		kb := &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: editButtonText, CallbackData: domain.SetSystemPromptCallbackPrefix + editButtonText},
				},
			},
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:          chatID,
			MessageThreadID: topicID,
			Text:            fmt.Sprintf("🧠 Текущая системная инструкция:\n%s", prompt),
			ReplyMarkup:     kb,
		})
	}
}
