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
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/lexis-app/lexis-api/internal/modules/auth/handler"
	progressHandler "github.com/lexis-app/lexis-api/internal/modules/progress/handler"
	tutorHandler "github.com/lexis-app/lexis-api/internal/modules/tutor/handler"
	vocabHandler "github.com/lexis-app/lexis-api/internal/modules/vocabulary/handler"
	"github.com/lexis-app/lexis-api/internal/shared/config"
	"github.com/lexis-app/lexis-api/internal/shared/middleware"
	"github.com/lexis-app/lexis-api/migrations"
)

// setupDatabase opens the pgx pool, pings it, and runs embedded migrations.
func setupDatabase(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}
	poolCfg.MaxConns = int32(cfg.DatabaseMaxConns)

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	log.Println("connected to PostgreSQL")

	if err := runMigrations(cfg.DatabaseURL); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func runMigrations(databaseURL string) error {
	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", source, databaseURL)
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
	return nil
}

// setupRedis opens the redis client and pings it.
func setupRedis(ctx context.Context, cfg *config.Config) (*redis.Client, error) {
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}
	if cfg.RedisPassword != "" {
		opts.Password = cfg.RedisPassword
	}

	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}
	log.Println("connected to Redis")
	return client, nil
}

// routerDeps bundles everything buildRouter needs. Keeping the dependency
// surface explicit at the call site is preferable to a big argument list.
type routerDeps struct {
	cfg         *config.Config
	redisClient *redis.Client
	blacklist   middleware.Blacklist
	auth        *handler.AuthHandler
	user        *handler.UserHandler
	vocab       *vocabHandler.VocabHandler
	progress    *progressHandler.ProgressHandler
	tutor       *tutorHandler.TutorHandler
}

func buildRouter(d routerDeps) http.Handler {
	r := chi.NewRouter()

	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(middleware.CORS(d.cfg.CORSAllowedOrigins))
	r.Use(middleware.MaxBodySize(1 << 20))
	r.Use(middleware.RateLimit(d.redisClient, "global", 60, time.Minute))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"status":"ok","version":%q}`, version)
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.RequireJSON)

		r.Route("/auth", func(r chi.Router) {
			r.With(middleware.LoginRateLimit(d.redisClient)).Post("/login", d.auth.Login)
			r.Post("/register", d.auth.Register)
			r.Post("/refresh", d.auth.Refresh)

			r.Group(func(r chi.Router) {
				r.Use(middleware.Auth([]byte(d.cfg.JWTSecret), d.blacklist))
				r.Post("/logout", d.auth.Logout)
				r.Post("/logout-all", d.auth.LogoutAll)
			})
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth([]byte(d.cfg.JWTSecret), d.blacklist))
			r.Use(func(next http.Handler) http.Handler {
				return http.TimeoutHandler(next, 30*time.Second, `{"type":"about:blank","title":"Request Timeout","status":503,"detail":"request took too long"}`)
			})

			r.Mount("/users", d.user.Routes())
			r.Get("/ai/models", handler.HandleGetModels)
			r.Mount("/vocabulary", d.vocab.Routes())
			r.Mount("/progress", d.progress.Routes())
		})

		r.Route("/tutor", func(r chi.Router) {
			r.Use(middleware.Auth([]byte(d.cfg.JWTSecret), d.blacklist))
			r.Use(middleware.RateLimit(d.redisClient, "tutor", 20, time.Minute))
			r.Mount("/", d.tutor.Routes())
		})
	})

	return r
}

// runHTTPServer starts the server and blocks until SIGINT/SIGTERM, then
// gracefully shuts it down with a 10s deadline.
func runHTTPServer(srv *http.Server) error {
	errCh := make(chan error, 1)
	go func() {
		log.Printf("lexis-api listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("server error: %w", err)
		}
	}()

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
