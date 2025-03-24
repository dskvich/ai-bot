package middleware

import (
	"context"

	"github.com/dskvich/ai-bot/pkg/logger"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func RequestID(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		ctx = logger.ContextWithRequestID(ctx, update.ID)

		next(ctx, b, update)
	}
}
