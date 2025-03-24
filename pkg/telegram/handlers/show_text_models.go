package handlers

import (
	"context"

	"github.com/dskvich/ai-bot/pkg/domain"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/samber/lo"
)

func ShowTextModels(supportedTextModels []string) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		chatID := update.Message.Chat.ID
		topicID := update.Message.MessageThreadID

		buttons := lo.Map(supportedTextModels, func(text string, _ int) models.InlineKeyboardButton {
			return models.InlineKeyboardButton{Text: text, CallbackData: domain.SetTextModelCallbackPrefix + text}
		})

		kb := &models.InlineKeyboardMarkup{
			InlineKeyboard: lo.Chunk(buttons, 2), // 2 button in a row
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:          chatID,
			MessageThreadID: topicID,
			Text:            "⚙️ Выберите текстовую модель GPT:",
			ReplyMarkup:     kb,
		})
	}
}
