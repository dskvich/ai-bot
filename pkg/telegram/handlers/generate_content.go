package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dskvich/ai-bot/pkg/domain"
	"github.com/dskvich/ai-bot/pkg/render"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/samber/lo"
)

type generateContentChatProvider interface {
	Get(ctx context.Context, chatID int64, topicID int) (*domain.Chat, error)
	Save(ctx context.Context, chat *domain.Chat) error
}

type generateContentAIService interface {
	GenerateImagePrompt(ctx context.Context, prompt string) (string, error)
	CreateChatCompletion(ctx context.Context, chat *domain.Chat) (*domain.Message, error)
}

type generateContentImageProvider interface {
	GenerateImage(ctx context.Context, prompt string, model string) ([]byte, error)
}

type generateContentPromptSaver interface {
	Save(ctx context.Context, prompt *domain.Prompt) error
}

type generateContentAudioConverter interface {
	ConvertToMP3(ctx context.Context, inputPath string) (string, error)
}

func GenerateContent(
	chatProvider generateContentChatProvider,
	promptSaver generateContentPromptSaver,
	aiService generateContentAIService,
	imageProvider generateContentImageProvider,
	audioConverter generateContentAudioConverter,
) bot.HandlerFunc {
	const maxTelegramMessageLength = 4096
	const moreButtonText = "Еще"

	findCutIndex := func(text string, maxLength int) int {
		if i := strings.LastIndex(text[:maxLength], "<pre>"); i > -1 {
			return i
		}
		if i := strings.LastIndex(text[:maxLength], "\n"); i > -1 {
			return i
		}
		return maxLength
	}

	getImageAsBytes := func(link string) ([]byte, error) {
		resp, err := http.Get(link)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, err
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return data, nil
	}

	saveTempVoiceFile := func(data []byte) (string, error) {
		const (
			voiceTempDir      = "tmp/voices"
			voiceTempFilePerm = 0o644
		)

		if err := os.MkdirAll(voiceTempDir, os.ModePerm); err != nil {
			return "", fmt.Errorf("unable to create temp directory: %w", err)
		}

		voiceFilePath := filepath.Join(voiceTempDir, fmt.Sprintf("voice-%d.ogg", time.Now().UnixNano()))
		if err := os.WriteFile(voiceFilePath, data, voiceTempFilePerm); err != nil {
			return "", fmt.Errorf("unable to write voice file: %w", err)
		}

		return voiceFilePath, nil
	}

	getVoiceAsMP3Bytes := func(ctx context.Context, voiceFileURL string) ([]byte, error) {
		voiceBytes, err := getImageAsBytes(voiceFileURL)
		if err != nil {
			return nil, fmt.Errorf("unable to download voice file: %w", err)
		}

		voiceFilePath, err := saveTempVoiceFile(voiceBytes)
		if err != nil {
			return nil, fmt.Errorf("unable to save temporary voice file: %w", err)
		}
		defer os.Remove(voiceFilePath)

		mp3Path, err := audioConverter.ConvertToMP3(ctx, voiceFilePath)
		if err != nil {
			return nil, fmt.Errorf("unable to convert voice file to mp3 file: %w", err)
		}
		defer os.Remove(mp3Path)

		file, err := os.Open(mp3Path)
		if err != nil {
			return nil, fmt.Errorf("opening mp3 file: %w", err)
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			return nil, fmt.Errorf("reading mp3 file: %w", err)
		}

		return data, nil
	}

	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		chatID := update.Message.Chat.ID
		topicID := update.Message.MessageThreadID
		prompt := &domain.Prompt{
			Text: lo.CoalesceOrEmpty(update.Message.Text, update.Message.Caption),
		}

		isImagePrompt := strings.Contains(strings.ToLower(prompt.Text), "рисуй") ||
			strings.Contains(strings.ToLower(prompt.Text), "draw")

		if isImagePrompt {
			newPrompt, err := aiService.GenerateImagePrompt(ctx, prompt.Text)
			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:          chatID,
					MessageThreadID: topicID,
					Text:            fmt.Sprintf("❌ Не удалось сгенерировать промпт: %s", err),
				})
				return
			}

			prompt.Text = newPrompt

			if err := promptSaver.Save(ctx, prompt); err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:          chatID,
					MessageThreadID: topicID,
					Text:            fmt.Sprintf("❌ Не удалось сохранить промпт: %s", err),
				})
				return
			}

			slog.InfoContext(ctx, "Prompt saved", "prompt", prompt)

			chat, err := chatProvider.Get(ctx, chatID, topicID)
			if err != nil && !errors.Is(err, domain.ErrNotFound) {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:          chatID,
					MessageThreadID: topicID,
					Text:            fmt.Sprintf("❌ Не удалось получить чат: %s", err),
				})
				return
			}

			model := ""
			if chat != nil {
				model = chat.ImageModel
			}

			imageData, err := imageProvider.GenerateImage(ctx, prompt.Text, model)
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
					{{Text: moreButtonText, CallbackData: domain.GenImageCallbackPrefix + strconv.Itoa(prompt.ID)}},
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
			return
		}

		// In case user sent an image
		if len(update.Message.Photo) > 0 {
			imageFile, err := b.GetFile(ctx, &bot.GetFileParams{
				FileID: update.Message.Photo[len(update.Message.Photo)-1].FileID,
			})
			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:          update.Message.Chat.ID,
					MessageThreadID: update.Message.MessageThreadID,
					Text:            fmt.Sprintf("❌ Не удалось получить метадату фото файла: %s", err),
				})
				return
			}

			imageFileURL, err := url.Parse(b.FileDownloadLink(imageFile))
			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:          update.Message.Chat.ID,
					MessageThreadID: update.Message.MessageThreadID,
					Text:            fmt.Sprintf("❌ Не удалось получить ссылку на фото файл: %s", err),
				})
				return
			}

			prompt.ImageBytes, err = getImageAsBytes(imageFileURL.String())
			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:          update.Message.Chat.ID,
					MessageThreadID: update.Message.MessageThreadID,
					Text:            fmt.Sprintf("❌ Не удалось получить фото файл: %s", err),
				})
				return
			}
		}

		if update.Message.Voice != nil {
			voiceFile, err := b.GetFile(ctx, &bot.GetFileParams{FileID: update.Message.Voice.FileID})
			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:          update.Message.Chat.ID,
					MessageThreadID: update.Message.MessageThreadID,
					Text:            fmt.Sprintf("❌ Не удалось получить метадату аудио файла: %s", err),
				})
				return
			}

			voiceFileURL, err := url.Parse(b.FileDownloadLink(voiceFile))
			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:          update.Message.Chat.ID,
					MessageThreadID: update.Message.MessageThreadID,
					Text:            fmt.Sprintf("❌ Не удалось получить ссылку на аудио файл: %s", err),
				})
				return
			}

			prompt.AudioBytes, err = getVoiceAsMP3Bytes(ctx, voiceFileURL.String())
			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:          update.Message.Chat.ID,
					MessageThreadID: update.Message.MessageThreadID,
					Text:            fmt.Sprintf("❌ Не удалось получить фото файл: %s", err),
				})
				return
			}
		}

		chat, err := chatProvider.Get(ctx, chatID, topicID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				chat = domain.NewChat(chatID, topicID)
			} else {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:          update.Message.Chat.ID,
					MessageThreadID: update.Message.MessageThreadID,
					Text:            fmt.Sprintf("❌ Не удалось получить чат: %s", err),
				})
				return
			}
		}

		if time.Now().After(chat.LastUpdate.Add(chat.TTL)) {
			chat.Messages = nil
		}

		content := []domain.ContentPart{{Type: domain.ContentPartTypeText, Data: prompt.Text}}

		if len(prompt.ImageBytes) > 0 {
			imageContent := domain.ContentPart{
				Type: domain.ContentPartTypeImage,
				Data: "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(prompt.ImageBytes),
			}

			if prompt.Text != "" {
				content = []domain.ContentPart{
					{Type: domain.ContentPartTypeText, Data: prompt.Text},
					imageContent,
				}
			} else {
				content = []domain.ContentPart{imageContent}
			}
		}

		chat.Messages = append(chat.Messages, domain.Message{
			Role:         domain.MessageRoleUser,
			ContentParts: content,
		})

		slog.InfoContext(ctx, "Calling AI for chat completion", "model", chat.TextModel, "messagesCount", len(chat.Messages))

		respMessage, err := aiService.CreateChatCompletion(ctx, chat)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          chatID,
				MessageThreadID: topicID,
				Text:            fmt.Sprintf("❌ Не удалось сгенерировать ответ: %s", err),
			})
			return
		}

		if respMessage == nil || len(respMessage.ContentParts) == 0 {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          chatID,
				MessageThreadID: topicID,
				Text:            "❌ Ответ пустой или отсутствует.",
			})
			return
		}

		chat.Messages = append(chat.Messages, *respMessage)

		err = chatProvider.Save(ctx, chat)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          chatID,
				MessageThreadID: topicID,
				Text:            fmt.Sprintf("❌ Не удалось сохранить чат: %+v", err),
			})
			return
		}

		part := respMessage.ContentParts[0] // Assume only one part for now
		if part.Type != domain.ContentPartTypeText {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          chatID,
				MessageThreadID: topicID,
				Text:            fmt.Sprintf("❌ Неожиданный тип ответа: %+v", part),
			})
			return
		}

		htmlText := render.ToHTML(part.Data)
		for htmlText != "" {
			if utf8.RuneCountInString(htmlText) <= maxTelegramMessageLength {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:          chatID,
					MessageThreadID: topicID,
					Text:            htmlText,
					ParseMode:       models.ParseModeHTML,
				})
				return
			}

			cutIndex := findCutIndex(htmlText, maxTelegramMessageLength)
			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          chatID,
				MessageThreadID: topicID,
				Text:            htmlText[:cutIndex],
				ParseMode:       models.ParseModeHTML,
			})
			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:          chatID,
					MessageThreadID: topicID,
					Text:            fmt.Sprintf("❌ Не удалось сгенерировать ответ: %s", err),
				})
			}
			htmlText = htmlText[cutIndex:]
			time.Sleep(time.Second) // Basic rate limit management
		}
	}
}
