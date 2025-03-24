package handlers

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func Start() bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		greeting := `👋 Привет! Я твой ChatGPT Telegram-бот. Вот что я умею:

🆕 <b>/new</b> — Начать новый чат
⏳ <b>/ttl</b> — Установить время жизни чата
📝 <b>/text_models</b> — Выбрать модель для текста
🖼️ <b>/image_models</b> — Выбрать модель для картинок
⚙️ <b>/system_prompt</b> — Настроить системную инструкцию

🖊️ Просто задай мне вопрос — я помогу!
🎨 Напиши "нарисуй ..." и я создам картинку.
🎙 Отправь голосовое сообщение — я пойму.
📷 Отправь картинку — я опишу её или отвечу на твои вопросы о ней.

Начнем? 🚀`

		b.SendMessage(ctx, &bot.SendMessageParams{
			MessageThreadID: update.Message.MessageThreadID,
			ChatID:          update.Message.Chat.ID,
			Text:            greeting,
			ParseMode:       models.ParseModeHTML,
		})
	}
}
