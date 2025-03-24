package handlers

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/dskvich/ai-bot/pkg/domain"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type regenerateImagePromptProvider interface {
	GetByID(ctx context.Context, id int64) (*domain.Prompt, error)
}

type regenerateImageProvider interface {
	GenerateImage(ctx context.Context, prompt string) ([]byte, error)
}

func RegenerateImage(
	promptProvider regenerateImagePromptProvider,
	imageProvider regenerateImageProvider,
) bot.HandlerFunc {
	const moreButtonText = "Еще"

	parsePromptID := func(promptIDRaw string) (int64, error) {
		idStr := strings.TrimPrefix(promptIDRaw, domain.GenImageCallbackPrefix)

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid promptID: %s", promptIDRaw)
		}

		return id, nil
	}

	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		chatID := update.CallbackQuery.Message.Message.Chat.ID
		topicID := update.CallbackQuery.Message.Message.MessageThreadID

		defer b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			ShowAlert:       false,
		})

		promptID, err := parsePromptID(update.CallbackQuery.Data)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          chatID,
				MessageThreadID: topicID,
				Text:            fmt.Sprintf("❌ Не удалось прочитать промпт ID: %s", err),
			})
			return
		}

		slog.InfoContext(ctx, "PromptID parsed", "id", promptID)

		prompt, err := promptProvider.GetByID(ctx, promptID)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          chatID,
				MessageThreadID: topicID,
				Text:            fmt.Sprintf("❌ Не удалось извлечь промпт: %s", err),
			})
			return
		}

		slog.InfoContext(ctx, "Prompt fetched", "prompt", prompt)

		imageData, err := imageProvider.GenerateImage(ctx, prompt.Text)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          chatID,
				MessageThreadID: topicID,
				Text:            fmt.Sprintf("❌ Не удалось сгенерировать изображение: %s", err),
			})
			return
		}

		slog.InfoContext(ctx, "Image generated", "size", len(imageData))

		kb := &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: moreButtonText, CallbackData: domain.GenImageCallbackPrefix + strconv.FormatInt(promptID, 10)},
				},
			},
		}

		b.SendPhoto(ctx, &bot.SendPhotoParams{
			ChatID:          chatID,
			MessageThreadID: topicID,
			Photo: &models.InputFileUpload{
				Data: bytes.NewReader(imageData),
			},
			ReplyMarkup: kb,
		})
	}
}
