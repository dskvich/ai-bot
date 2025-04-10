package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/caarlos0/env/v9"
	"github.com/dskvich/ai-bot/pkg/converter"
	"github.com/dskvich/ai-bot/pkg/database"
	"github.com/dskvich/ai-bot/pkg/domain"
	"github.com/dskvich/ai-bot/pkg/llm/openai"
	"github.com/dskvich/ai-bot/pkg/logger"
	"github.com/dskvich/ai-bot/pkg/repository"
	"github.com/dskvich/ai-bot/pkg/services"
	"github.com/dskvich/ai-bot/pkg/telegram/handlers"
	"github.com/dskvich/ai-bot/pkg/telegram/matchers"
	"github.com/dskvich/ai-bot/pkg/telegram/middleware"
	"github.com/go-telegram/bot"
)

type Config struct {
	OpenAIToken               string  `env:"OPEN_AI_TOKEN,required"`
	TelegramBotToken          string  `env:"TELEGRAM_BOT_TOKEN,required"`
	TelegramAuthorizedUserIDs []int64 `env:"TELEGRAM_AUTHORIZED_USER_IDS" envSeparator:" "`
	PgURL                     string  `env:"DATABASE_URL"`
	PgHost                    string  `env:"DB_HOST" envDefault:"localhost:61234"`
	BunDebug                  int     `env:"BUNDEBUG" envDefault:"0"`
}

func main() {
	slog.SetDefault(slog.New(logger.NewHandler(os.Stderr, logger.DefaultOptions)))

	if err := runMain(); err != nil {
		slog.Error("shutting down due to error", logger.Err(err))
		os.Exit(1)
	}
	slog.Info("shutdown complete")
}

func runMain() error {
	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()

	svcGroup, err := setupServices(ctx)
	if err != nil {
		return err
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGHUP)
		select {
		case s := <-sigCh:
			slog.Info("shutting down due to signal", "signal", s.String())
			cancelFn()
		case <-ctx.Done():
		}
	}()

	return svcGroup.Start(ctx)
}

func setupServices(ctx context.Context) (services.Group, error) {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parsing env config: %w", err)
	}

	var svc services.Service
	var svcGroup services.Group

	db, err := database.NewDB(cfg.PgURL, cfg.PgHost)
	if err != nil {
		return nil, fmt.Errorf("initializing database: %w", err)
	}

	openAIClient, err := openai.NewClient(cfg.OpenAIToken)
	if err != nil {
		return nil, fmt.Errorf("creating open ai client: %w", err)
	}

	chatRepository := repository.NewChatRepository(db)
	stateRepository := repository.NewStateRepository()
	promptRepository := repository.NewPromptRepository(db)

	// Price per 1M tokens (Input/Output)
	// https://platform.openai.com/docs/pricing
	supportedTextModels := []string{
		"gpt-4o-mini",   // $0.15/$0.60
		"gpt-3.5-turbo", // $0.50/$1.50
		"o3-mini",       // $1.10/$4.40
		// "gpt-4o",        // $2.50/$10.00
		// "gpt-4-turbo",   // $10.00/$30.00
	}

	supportedImageModels := []string{
		domain.DallE2Model, // DALL-E 2
		domain.DallE3Model, // DALL-E 3
	}

	supportedTTLOptions := []time.Duration{
		30 * time.Second,
		15 * time.Minute,
		time.Hour,
		8 * time.Hour,
		24 * time.Hour,
		7 * 24 * time.Hour,
	}

	opts := []bot.Option{
		bot.WithMiddlewares(
			middleware.RequestID,
			middleware.Auth(cfg.TelegramAuthorizedUserIDs),
			middleware.Typing,
			middleware.VoiceToText(&converter.VoiceToMP3{}, openAIClient),
		),

		bot.WithDefaultHandler(handlers.GenerateContent(chatRepository, promptRepository, openAIClient, &converter.VoiceToMP3{})),
		bot.WithMessageTextHandler("/start", bot.MatchTypePrefix, handlers.Start()),
		bot.WithMessageTextHandler("/new", bot.MatchTypePrefix, handlers.ClearChat(chatRepository)),
		bot.WithMessageTextHandler("/text_models", bot.MatchTypePrefix, handlers.ShowTextModels(supportedTextModels)),
		bot.WithMessageTextHandler("/image_models", bot.MatchTypePrefix, handlers.ShowImageModels(supportedImageModels)),
		bot.WithMessageTextHandler("/system_prompt", bot.MatchTypePrefix, handlers.ShowSystemPrompt(chatRepository)),
		bot.WithMessageTextHandler("/ttl", bot.MatchTypePrefix, handlers.ShowTTL(supportedTTLOptions)),

		bot.WithCallbackQueryDataHandler(domain.SetImageModelCallbackPrefix, bot.MatchTypePrefix, handlers.SetImageModel(chatRepository, supportedImageModels)),
		bot.WithCallbackQueryDataHandler(domain.SetTTLCallbackPrefix, bot.MatchTypePrefix, handlers.SetTTL(chatRepository, supportedTTLOptions)),
		bot.WithCallbackQueryDataHandler(domain.SetTextModelCallbackPrefix, bot.MatchTypePrefix, handlers.SetTextModel(chatRepository, supportedTextModels)),
		bot.WithCallbackQueryDataHandler(domain.SetSystemPromptCallbackPrefix, bot.MatchTypePrefix, handlers.RequestSystemPrompt(stateRepository)),
		bot.WithCallbackQueryDataHandler(domain.GenImageCallbackPrefix, bot.MatchTypePrefix, handlers.RegenerateImage(promptRepository, openAIClient, chatRepository)),
	}

	b, err := bot.New(cfg.TelegramBotToken, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating telegram bot: %w", err)
	}

	b.RegisterHandlerMatchFunc(matchers.IsEditingSystemPrompt(stateRepository), handlers.SetSystemPrompt(chatRepository, stateRepository))

	if svc, err = services.NewTelegramBot(b); err == nil {
		svcGroup = append(svcGroup, svc)
	} else {
		return nil, err
	}

	return svcGroup, nil
}
