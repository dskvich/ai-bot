package middleware

import (
	"context"
	"log/slog"
	"slices"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func Auth(authorizedIDs []int64) bot.Middleware {
	return func(next bot.HandlerFunc) bot.HandlerFunc {
		return func(ctx context.Context, b *bot.Bot, update *models.Update) {
			slog.InfoContext(ctx, "Auth middleware started")

			var userID int64
			switch {
			case update.Message != nil:
				userID = update.Message.From.ID
			case update.CallbackQuery != nil:
				userID = update.CallbackQuery.From.ID
			default:
				slog.WarnContext(ctx, "Received unknown update type", "update", update)
				return
			}

			if slices.Contains(authorizedIDs, userID) {
				next(ctx, b, update)
				return
			}

			slog.WarnContext(ctx, "Unauthorized access attempt", "userID", userID)

			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          update.Message.Chat.ID,
				MessageThreadID: update.Message.MessageThreadID,
				Text:            "❌ Not authorized",
			})
		}
	}
}
