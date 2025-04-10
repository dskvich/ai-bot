package handlers

import (
	"context"

	"github.com/dskvich/ai-bot/pkg/domain"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/samber/lo"
)

func ShowImageModels(supportedImageModels []string) bot.HandlerFunc {

	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		chatID := update.Message.Chat.ID
		topicID := update.Message.MessageThreadID

		buttons := lo.Map(supportedImageModels, func(model string, _ int) models.InlineKeyboardButton {
			var displayName string
			switch model {
			case domain.DallE2Model:
				displayName = "DALL-E 2"
			case domain.DallE3Model:
				displayName = "DALL-E 3"
			default:
				displayName = model
			}
			return models.InlineKeyboardButton{Text: displayName, CallbackData: domain.SetImageModelCallbackPrefix + model}
		})

		kb := &models.InlineKeyboardMarkup{
			InlineKeyboard: lo.Chunk(buttons, 2), // 2 buttons in a row
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:          chatID,
			MessageThreadID: topicID,
			Text:            "🎨 Выберите модель для генерации изображений:",
			ReplyMarkup:     kb,
		})
	}
}
