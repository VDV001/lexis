package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

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
	"github.com/lexis-app/lexis-api/internal/shared/middleware"
	"github.com/lexis-app/lexis-api/migrations"
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

	// ---- PostgreSQL connection pool ----
	ctx := context.Background()

	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to parse database URL: %w", err)
	}
	poolCfg.MaxConns = int32(cfg.DatabaseMaxConns)

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	log.Println("connected to PostgreSQL")

	// ---- Run migrations ----
	migrationSource, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", migrationSource, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	srcErr, dbErr := m.Close()
	if srcErr != nil {
		return fmt.Errorf("failed to close migration source: %w", srcErr)
	}
	if dbErr != nil {
		return fmt.Errorf("failed to close migration db: %w", dbErr)
	}
	log.Println("migrations applied")

	// ---- Redis client ----
	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return fmt.Errorf("failed to parse redis URL: %w", err)
	}
	if cfg.RedisPassword != "" {
		redisOpts.Password = cfg.RedisPassword
	}

	redisClient := redis.NewClient(redisOpts)
	defer func() { _ = redisClient.Close() }()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to ping redis: %w", err)
	}
	log.Println("connected to Redis")

	// ---- Repositories ----
	userRepo := infra.NewPostgresUserRepo(pool)
	tokenRepo := infra.NewPostgresTokenRepo(pool)
	settingsRepo := infra.NewPostgresSettingsRepo(pool)
	blacklist := infra.NewRedisBlacklist(redisClient)

	// ---- Services ----
	authService := usecase.NewAuthService(
		userRepo,
		tokenRepo,
		settingsRepo,
		blacklist,
		cfg.JWTSecret,
		cfg.JWTAccessTTL,
		cfg.JWTRefreshTTL,
	)

	// ---- Handlers ----
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

	// ---- Background contexts ----
	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()

	eventCtx, eventCancel := context.WithCancel(ctx)
	defer eventCancel()

	// ---- Snapshot worker ----
	snapshotWorker := vocabUsecase.NewVocabSnapshotWorker(wordRepo, snapshotRepo)
	go snapshotWorker.Run(workerCtx)

	// ---- Event subscriptions ----
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

	// ---- Router ----
	r := chi.NewRouter()

	// Global middleware
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(middleware.CORS(cfg.CORSAllowedOrigins))
	r.Use(middleware.MaxBodySize(1 << 20)) // 1 MB global limit
	r.Use(middleware.RateLimit(redisClient, "global", 60, time.Minute))

	// Health check (no auth)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"status":"ok","version":%q}`, version)
	})

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.RequireJSON)
		// Auth routes (public + protected in one sub-router)
		r.Route("/auth", func(r chi.Router) {
			// Public
			r.With(middleware.LoginRateLimit(redisClient)).Post("/login", authHandler.Login)
			r.Post("/register", authHandler.Register)
			r.Post("/refresh", authHandler.Refresh)

			// Protected
			r.Group(func(r chi.Router) {
				r.Use(middleware.Auth([]byte(cfg.JWTSecret), blacklist))
				r.Post("/logout", authHandler.Logout)
				r.Post("/logout-all", authHandler.LogoutAll)
			})
		})

		// Protected routes (with write timeout for non-streaming endpoints)
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth([]byte(cfg.JWTSecret), blacklist))
			r.Use(func(next http.Handler) http.Handler {
				return http.TimeoutHandler(next, 30*time.Second, `{"type":"about:blank","title":"Request Timeout","status":503,"detail":"request took too long"}`)
			})

			r.Mount("/users", userHandler.Routes())
			r.Get("/ai/models", handler.HandleGetModels)
			r.Mount("/vocabulary", vocabH.Routes())
			r.Mount("/progress", progressH.Routes())
		})

		// Tutor routes (auth + stricter rate limit for AI endpoints)
		r.Route("/tutor", func(r chi.Router) {
			r.Use(middleware.Auth([]byte(cfg.JWTSecret), blacklist))
			r.Use(middleware.RateLimit(redisClient, "tutor", 20, time.Minute))
			r.Mount("/", tutorH.Routes())
		})
	})

	// ---- HTTP Server ----
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.AppPort),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0, // disabled — SSE streams need long-lived connections; per-request timeouts via context
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("lexis-api listening on :%d", cfg.AppPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("server error: %w", err)
		}
	}()

	// ---- Graceful shutdown ----
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
	case err := <-errCh:
		return err
	}

	log.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("forced shutdown: %w", err)
	}
	log.Println("server stopped")
	return nil
}
