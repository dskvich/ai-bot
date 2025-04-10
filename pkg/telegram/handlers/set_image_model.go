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

type SetImageModelChatProvider interface {
	Get(ctx context.Context, chatID int64, topicID int) (*domain.Chat, error)
	Save(ctx context.Context, chat *domain.Chat) error
}

func SetImageModel(chatProvider SetImageModelChatProvider, supportedImageModels []string) bot.HandlerFunc {
	parseImageModel := func(modelRaw string) (string, error) {
		if !strings.HasPrefix(modelRaw, domain.SetImageModelCallbackPrefix) {
			return "", fmt.Errorf("invalid format, expected prefix '%s'", domain.SetImageModelCallbackPrefix)
		}

		model := strings.TrimPrefix(modelRaw, domain.SetImageModelCallbackPrefix)

		if lo.Contains(supportedImageModels, model) {
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

		model, err := parseImageModel(update.CallbackQuery.Data)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          chatID,
				MessageThreadID: topicID,
				Text:            fmt.Sprintf("❌ Не удалось извлечь модель изображения: %s", err),
			})
			return
		}

		chat, err := chatProvider.Get(ctx, chatID, topicID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				chat = domain.NewChat(chatID, topicID)
			} else {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:          chatID,
					MessageThreadID: topicID,
					Text:            fmt.Sprintf("❌ Не удалось получить чат: %s", err),
				})
				return
			}
		}

		chat.ImageModel = model

		if err = chatProvider.Save(ctx, chat); err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          chatID,
				MessageThreadID: topicID,
				Text:            fmt.Sprintf("❌ Не удалось сохранить чат: %s", err),
			})
			return
		}

		var displayName string
		switch model {
		case domain.DallE2Model:
			displayName = "DALL-E 2"
		case domain.DallE3Model:
			displayName = "DALL-E 3"
		default:
			displayName = model
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:          chatID,
			MessageThreadID: topicID,
			Text:            "✅ Модель для генерации изображений установлена: " + displayName,
		})
	}
}
