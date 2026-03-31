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
	tutorHandler "github.com/lexis-app/lexis-api/internal/modules/tutor/handler"
	tutorInfra "github.com/lexis-app/lexis-api/internal/modules/tutor/infra"
	tutorUsecase "github.com/lexis-app/lexis-api/internal/modules/tutor/usecase"
	"github.com/lexis-app/lexis-api/internal/shared/config"
	"github.com/lexis-app/lexis-api/internal/shared/middleware"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// ---- PostgreSQL connection pool ----
	ctx := context.Background()

	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to parse database URL: %v", err)
	}
	poolCfg.MaxConns = int32(cfg.DatabaseMaxConns)

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	log.Println("connected to PostgreSQL")

	// ---- Redis client ----
	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("failed to parse redis URL: %v", err)
	}
	if cfg.RedisPassword != "" {
		redisOpts.Password = cfg.RedisPassword
	}

	redisClient := redis.NewClient(redisOpts)
	defer func() { _ = redisClient.Close() }()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to ping redis: %v", err)
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
		// Auth routes (public)
		r.Mount("/auth", authHandler.Routes())

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth([]byte(cfg.JWTSecret)))

			r.Mount("/users", userHandler.Routes())
			r.Get("/ai/models", handler.HandleGetModels)
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

	go func() {
		log.Printf("lexis-api listening on :%d", cfg.AppPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// ---- Graceful shutdown ----
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	log.Println("server stopped")
}
