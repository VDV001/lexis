package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"

	"github.com/lexis-app/lexis-api/internal/modules/auth/handler"
	"github.com/lexis-app/lexis-api/internal/modules/auth/infra"
	"github.com/lexis-app/lexis-api/internal/modules/auth/usecase"
	progressHandler "github.com/lexis-app/lexis-api/internal/modules/progress/handler"
	progressInfra "github.com/lexis-app/lexis-api/internal/modules/progress/infra"
	progressUsecase "github.com/lexis-app/lexis-api/internal/modules/progress/usecase"
	tutorHandler "github.com/lexis-app/lexis-api/internal/modules/tutor/handler"
	tutorInfra "github.com/lexis-app/lexis-api/internal/modules/tutor/infra"
	tutorUsecase "github.com/lexis-app/lexis-api/internal/modules/tutor/usecase"
	vocabHandler "github.com/lexis-app/lexis-api/internal/modules/vocabulary/handler"
	vocabInfra "github.com/lexis-app/lexis-api/internal/modules/vocabulary/infra"
	vocabUsecase "github.com/lexis-app/lexis-api/internal/modules/vocabulary/usecase"
	"github.com/lexis-app/lexis-api/internal/shared/config"
	"github.com/lexis-app/lexis-api/internal/shared/eventbus"
)

// Version is set at build time via ldflags:
//
//	go build -ldflags "-X main.version=$(cat ../../VERSION)" ./cmd/api
var version = "dev"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ctx := context.Background()

	pool, err := setupDatabase(ctx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()

	redisClient, err := setupRedis(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() { _ = redisClient.Close() }()

	// ---- Repositories ----
	userRepo := infra.NewPostgresUserRepo(pool)
	tokenRepo := infra.NewPostgresTokenRepo(pool)
	settingsRepo := infra.NewPostgresSettingsRepo(pool)
	blacklist := infra.NewRedisBlacklist(redisClient)

	// ---- Auth services + handlers ----
	authService := usecase.NewAuthService(
		userRepo, tokenRepo, settingsRepo, blacklist,
		cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL,
	)
	secureCookies := cfg.AppEnv == "production"
	authHandler := handler.NewAuthHandler(authService, secureCookies, cfg.JWTRefreshTTL)
	userService := usecase.NewUserService(userRepo, settingsRepo)
	userHandler := handler.NewUserHandler(userService)

	// ---- Event bus ----
	bus := eventbus.New()

	// ---- Tutor module ----
	registry := tutorInfra.NewDefaultRegistry(cfg.AnthropicAPIKey, cfg.OpenAIAPIKey, cfg.QwenAPIKey, cfg.GeminiAPIKey)
	chatService := tutorUsecase.NewChatService(registry, tutorSettingsAdapter{inner: settingsRepo}, tutorUserAdapter{inner: userRepo})
	exerciseService := tutorUsecase.NewExerciseService(registry, tutorSettingsAdapter{inner: settingsRepo}, bus)
	tutorH := tutorHandler.NewTutorHandler(chatService, exerciseService)

	// ---- Vocabulary + Progress modules ----
	wordRepo := vocabInfra.NewPostgresWordRepo(pool)
	snapshotRepo := vocabInfra.NewPostgresSnapshotRepo(pool)
	vocabService := vocabUsecase.NewVocabService(wordRepo, vocabSettingsAdapter{inner: settingsRepo})
	vocabH := vocabHandler.NewVocabHandler(vocabService)

	sessionRepo := progressInfra.NewPostgresSessionRepo(pool)
	roundRepo := progressInfra.NewPostgresRoundRepo(pool)
	goalRepo := progressInfra.NewPostgresGoalRepo(pool)
	progressService := progressUsecase.NewProgressService(
		roundRepo, sessionRepo, goalRepo, wordRepo,
		progressSnapshotAdapter{inner: snapshotRepo},
		progressSettingsAdapter{inner: settingsRepo},
	)
	progressH := progressHandler.NewProgressHandler(progressService)

	// ---- Background workers + event subscriptions ----
	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()
	eventCtx, eventCancel := context.WithCancel(ctx)
	defer eventCancel()

	snapshotWorker := vocabUsecase.NewVocabSnapshotWorker(wordRepo, snapshotRepo)
	go snapshotWorker.Run(workerCtx)

	bus.Subscribe(eventbus.EventWordsDiscovered, func(e eventbus.Event) {
		payload, ok := e.Payload.(eventbus.WordsDiscoveredPayload)
		if !ok {
			log.Printf("eventbus: unexpected payload type for %s", e.Type)
			return
		}
		if err := vocabService.AddDiscoveredWords(eventCtx, payload.UserID, payload.Language, payload.Words, payload.Context); err != nil {
			log.Printf("eventbus: failed to add discovered words: %v", err)
		}
	})

	// ---- Router + HTTP server ----
	r := buildRouter(routerDeps{
		cfg:         cfg,
		redisClient: redisClient,
		blacklist:   blacklist,
		auth:        authHandler,
		user:        userHandler,
		vocab:       vocabH,
		progress:    progressH,
		tutor:       tutorH,
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.AppPort),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0, // disabled — SSE streams need long-lived connections; per-request timeouts via context
		IdleTimeout:  60 * time.Second,
	}

	return runHTTPServer(srv)
}
