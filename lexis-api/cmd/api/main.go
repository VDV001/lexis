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
	vocabInfra "github.com/lexis-app/lexis-api/internal/modules/vocabulary/infra"
	"github.com/lexis-app/lexis-api/internal/shared/config"
	"github.com/lexis-app/lexis-api/internal/shared/middleware"
)

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
	authHandler := handler.NewAuthHandler(authService, secureCookies)
	userHandler := handler.NewUserHandler(userRepo, settingsRepo)

	// ---- Tutor module ----
	registry := tutorInfra.NewDefaultRegistry(cfg.AnthropicAPIKey, cfg.OpenAIAPIKey, cfg.QwenAPIKey, cfg.GeminiAPIKey)
	chatService := tutorUsecase.NewChatService(registry, settingsRepo, userRepo)
	exerciseService := tutorUsecase.NewExerciseService(registry, settingsRepo)
	tutorH := tutorHandler.NewTutorHandler(chatService, exerciseService)

	// ---- Vocabulary + Progress modules ----
	wordRepo := vocabInfra.NewPostgresWordRepo(pool)
	snapshotRepo := vocabInfra.NewPostgresSnapshotRepo(pool)
	sessionRepo := progressInfra.NewPostgresSessionRepo(pool)
	roundRepo := progressInfra.NewPostgresRoundRepo(pool)
	goalRepo := progressInfra.NewPostgresGoalRepo(pool)

	progressService := progressUsecase.NewProgressService(roundRepo, sessionRepo, goalRepo, wordRepo, snapshotRepo, settingsRepo)
	progressH := progressHandler.NewProgressHandler(progressService)

	// ---- Router ----
	r := chi.NewRouter()

	// Global middleware
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(middleware.CORS(cfg.CORSAllowedOrigins))
	r.Use(middleware.RateLimit(redisClient, 60, time.Minute))

	// Health check (no auth)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Auth public routes (register, login, refresh)
		r.Route("/auth", func(r chi.Router) {
			r.With(middleware.LoginRateLimit(redisClient)).Post("/login", authHandler.Login)
			r.Mount("/", authHandler.PublicRoutes())
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth([]byte(cfg.JWTSecret)))

			// Auth protected routes (logout, logout-all)
			r.Mount("/auth", authHandler.ProtectedRoutes())

			r.Mount("/users", userHandler.Routes())
			r.Get("/ai/models", handler.HandleGetModels)
			r.Mount("/progress", progressH.Routes())
		})

		// Tutor routes (auth + stricter rate limit for AI endpoints)
		r.Route("/tutor", func(r chi.Router) {
			r.Use(middleware.Auth([]byte(cfg.JWTSecret)))
			r.Use(middleware.RateLimit(redisClient, 20, time.Minute))
			r.Mount("/", tutorH.Routes())
		})
	})

	// ---- HTTP Server ----
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.AppPort),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
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
