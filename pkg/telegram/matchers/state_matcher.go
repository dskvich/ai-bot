package matchers

import (
	"github.com/dskvich/ai-bot/pkg/domain"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type StateProvider interface {
	Get(chatID int64, topicID int) (domain.State, bool)
}

func IsEditingSystemPrompt(provider StateProvider) bot.MatchFunc {
	return func(update *models.Update) bool {
		if update.Message == nil {
			return false
		}
		chatID := update.Message.Chat.ID
		topicID := update.Message.MessageThreadID

		state, ok := provider.Get(chatID, topicID)
		if !ok {
			return false
		}

		return state == domain.StateEditSystemPrompt
	}
}
