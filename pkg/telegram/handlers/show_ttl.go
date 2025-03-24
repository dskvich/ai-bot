package handlers

import (
	"context"
	"strings"
	"time"

	"github.com/dskvich/ai-bot/pkg/domain"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/samber/lo"
)

func ShowTTL(supportedTTLOptions []time.Duration) bot.HandlerFunc {
	shortDuration := func(d time.Duration) string {
		s := d.String()
		s = lo.Ternary(strings.HasSuffix(s, "m0s"), s[:len(s)-2], s)
		s = lo.Ternary(strings.HasSuffix(s, "h0m"), s[:len(s)-2], s)
		return s
	}

	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		chatID := update.Message.Chat.ID
		topicID := update.Message.MessageThreadID

		ttlOptions := lo.Map(supportedTTLOptions, func(d time.Duration, _ int) string {
			return shortDuration(d)
		})

		buttons := lo.Map(ttlOptions, func(text string, _ int) models.InlineKeyboardButton {
			return models.InlineKeyboardButton{Text: text, CallbackData: domain.SetTTLCallbackPrefix + text}
		})

		kb := &models.InlineKeyboardMarkup{
			InlineKeyboard: lo.Chunk(buttons, 10), // 10 buttons in a row
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:          chatID,
			MessageThreadID: topicID,
			Text:            "⚙️ Выберите период хранения данных чата:",
			ReplyMarkup:     kb,
		})
	}
}
