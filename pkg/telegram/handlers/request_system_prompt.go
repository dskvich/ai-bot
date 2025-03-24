package handlers

import (
	"context"

	"github.com/dskvich/ai-bot/pkg/domain"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type RequestSystemPromptStateProvider interface {
	Save(chatID int64, topicID int, state domain.State)
}

func RequestSystemPrompt(provider RequestSystemPromptStateProvider) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		chatID := update.CallbackQuery.Message.Message.Chat.ID
		topicID := update.CallbackQuery.Message.Message.MessageThreadID

		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			ShowAlert:       false,
		})

		provider.Save(chatID, topicID, domain.StateEditSystemPrompt)

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:          chatID,
			MessageThreadID: topicID,
			Text:            "📝 Пожалуйста, отправьте новую системную инструкцию:",
		})
	}
}
