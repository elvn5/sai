// Command content serves the editorial content: blog posts, written
// guides, the community photo gallery and per-page SEO overrides.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/content/internal/config"
	"github.com/elvn5/sai/services/content/internal/handlers"
	"github.com/elvn5/sai/services/content/internal/legacyimport"
	"github.com/elvn5/sai/services/content/internal/mediaclient"
	"github.com/elvn5/sai/services/content/internal/migrations"
	"github.com/elvn5/sai/services/content/internal/routes"
	"github.com/elvn5/sai/services/content/internal/store"
	pkgconfig "github.com/elvn5/sai/services/pkg/config"
	"github.com/elvn5/sai/services/pkg/database"
	"github.com/elvn5/sai/services/pkg/logger"
	"github.com/elvn5/sai/services/pkg/middleware"
	"github.com/elvn5/sai/services/pkg/usersclient"
)

const shutdownTimeout = 15 * time.Second

func main() {
	log := logger.New("content")
	if err := run(log); err != nil {
		log.Error("content stopped", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := pkgconfig.Load[config.Config]()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := database.Connect(ctx, cfg.DSN(), migrations.FS, log)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer db.Close()

	// Before serving, so the first requests already see the imported data.
	if cfg.LegacyDatabase != "" {
		if err := legacyimport.Run(ctx, db, cfg.LegacyDSN(), log); err != nil {
			return fmt.Errorf("legacy import: %w", err)
		}
	}

	var media handlers.PhotoDeleter
	if cfg.MediaGRPCAddr != "" {
		client, err := mediaclient.New(cfg.MediaGRPCAddr)
		if err != nil {
			return err
		}
		defer func() { _ = client.Close() }()
		media = client
	} else {
		log.Warn("MEDIA_GRPC_ADDR not set: deleted gallery photos will stay in storage")
	}

	var users middleware.UserLookup
	if cfg.UsersGRPCAddr != "" {
		client, err := usersclient.New(cfg.UsersGRPCAddr)
		if err != nil {
			return err
		}
		defer func() { _ = client.Close() }()
		users = client
	}

	gin.SetMode(gin.ReleaseMode)
	srv := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: routes.Setup(routes.Deps{
			Store: store.New(db), Media: media, Users: users, JWTSecret: cfg.JWTSecret, Log: log,
		}),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("content listening", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
