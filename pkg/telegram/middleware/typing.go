package middleware

import (
	"context"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func Typing(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		slog.InfoContext(ctx, "Typing middleware started")

		var (
			chatID  int64
			topicID int
		)

		switch {
		case update.Message != nil:
			chatID, topicID = update.Message.Chat.ID, update.Message.MessageThreadID
		case update.CallbackQuery != nil:
			chatID, topicID = update.CallbackQuery.Message.Message.Chat.ID, update.CallbackQuery.Message.Message.MessageThreadID
		default:
			slog.WarnContext(ctx, "Received unknown update type", "update", update)
		}

		if chatID != 0 {
			b.SendChatAction(ctx, &bot.SendChatActionParams{
				ChatID:          chatID,
				MessageThreadID: topicID,
				Action:          models.ChatActionTyping,
			})
		}

		next(ctx, b, update)
	}
}
