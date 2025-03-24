package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dskvich/ai-bot/pkg/domain"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/samber/lo"
)

type SetTextModelChatProvider interface {
	Get(ctx context.Context, chatID int64, topicID int) (*domain.Chat, error)
	Save(ctx context.Context, chat *domain.Chat) error
	DeleteMessages(ctx context.Context, chat *domain.Chat) error
}

func SetTextModel(chatProvider SetTextModelChatProvider, supportedTextModels []string) bot.HandlerFunc {
	parseTextModel := func(modelRaw string) (string, error) {
		if !strings.HasPrefix(modelRaw, domain.SetTextModelCallbackPrefix) {
			return "", fmt.Errorf("invalid format, expected prefix '%s'", domain.SetTextModelCallbackPrefix)
		}

		model := strings.TrimPrefix(modelRaw, domain.SetTextModelCallbackPrefix)

		if lo.Contains(supportedTextModels, model) {
			return model, nil
		}

		return "", errors.New("unsupported model")
	}

	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		chatID := update.CallbackQuery.Message.Message.Chat.ID
		topicID := update.CallbackQuery.Message.Message.MessageThreadID

		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			ShowAlert:       false,
		})

		model, err := parseTextModel(update.CallbackQuery.Data)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          chatID,
				MessageThreadID: topicID,
				Text:            fmt.Sprintf("❌ Не удалось извлечь текстовую модель: %s", err),
			})
			return
		}

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

		chat.TextModel = model

		if err = chatProvider.Save(ctx, chat); err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          chatID,
				MessageThreadID: topicID,
				Text:            fmt.Sprintf("❌ Не удалось сохранить чат: %s", err),
			})
			return
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:          chatID,
			MessageThreadID: topicID,
			Text:            "✅ Модель установлена: " + model,
		})

		if err = chatProvider.DeleteMessages(ctx, chat); err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          chatID,
				MessageThreadID: topicID,
				Text:            fmt.Sprintf("❌ Не удалось очистить историю: %s", err),
			})
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:          chatID,
			MessageThreadID: topicID,
			Text:            "🧹 История очищена! Начните новый чат. 🚀",
		})
	}
}
