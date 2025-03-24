package middleware

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type audioConverter interface {
	ConvertToMP3(ctx context.Context, inputPath string) (string, error)
}

type audioTranscriber interface {
	TranscribeAudio(ctx context.Context, audioFilePath string) (string, error)
}

func VoiceToText(converter audioConverter, transcriber audioTranscriber) bot.Middleware {
	return func(next bot.HandlerFunc) bot.HandlerFunc {
		downloadFileToBuffer := func(link string) ([]byte, error) {
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

		processVoiceMessage := func(ctx context.Context, b *bot.Bot, update *models.Update) (string, error) {
			voiceFile, err := b.GetFile(ctx, &bot.GetFileParams{FileID: update.Message.Voice.FileID})
			if err != nil {
				return "", fmt.Errorf("unable to get voice file metadata: %w", err)
			}

			voiceFileURL, err := url.Parse(b.FileDownloadLink(voiceFile))
			if err != nil {
				return "", fmt.Errorf("invalid voice file URL: %w", err)
			}

			voiceBytes, err := downloadFileToBuffer(voiceFileURL.String())
			if err != nil {
				return "", fmt.Errorf("unable to download voice file: %w", err)
			}

			voiceFilePath, err := saveTempVoiceFile(voiceBytes)
			if err != nil {
				return "", fmt.Errorf("unable to save temporary voice file: %w", err)
			}
			defer os.Remove(voiceFilePath)

			mp3Path, err := converter.ConvertToMP3(ctx, voiceFilePath)
			if err != nil {
				return "", fmt.Errorf("unable to convert voice file to MP3: %w", err)
			}
			defer os.Remove(mp3Path)

			transcribedText, err := transcriber.TranscribeAudio(ctx, mp3Path)
			if err != nil {
				return "", fmt.Errorf("unable to transcribe MP3 file: %w", err)
			}

			return transcribedText, nil
		}

		return func(ctx context.Context, b *bot.Bot, update *models.Update) {
			slog.InfoContext(ctx, "Voice to text middleware started")

			if update.Message == nil || update.Message.Voice == nil {
				next(ctx, b, update)
				return
			}

			transcribedText, err := processVoiceMessage(ctx, b, update)
			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:          update.Message.Chat.ID,
					MessageThreadID: update.Message.MessageThreadID,
					Text:            fmt.Sprintf("❌ Ошибка при обработке голосового сообщения: %s", err),
				})
				return
			}

			update.Message.Text = transcribedText

			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          update.Message.Chat.ID,
				MessageThreadID: update.Message.MessageThreadID,
				Text:            fmt.Sprintf("🎤 %s", transcribedText),
			})

			next(ctx, b, update)
		}
	}
}
