// Command listings serves what travelers book: hotels, local guides and
// tours, with their reviews and the owners' posts.
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

	"github.com/elvn5/sai/services/listings/internal/config"
	"github.com/elvn5/sai/services/listings/internal/handlers"
	"github.com/elvn5/sai/services/listings/internal/legacyimport"
	"github.com/elvn5/sai/services/listings/internal/migrations"
	"github.com/elvn5/sai/services/listings/internal/routes"
	"github.com/elvn5/sai/services/listings/internal/store"
	pkgconfig "github.com/elvn5/sai/services/pkg/config"
	"github.com/elvn5/sai/services/pkg/database"
	"github.com/elvn5/sai/services/pkg/logger"
	"github.com/elvn5/sai/services/pkg/usersclient"
)

const shutdownTimeout = 15 * time.Second

func main() {
	log := logger.New("listings")
	if err := run(log); err != nil {
		log.Error("listings stopped", slog.Any("error", err))
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

	if cfg.LegacyDatabase != "" {
		if err := legacyimport.Run(ctx, db, cfg.LegacyDSN(), log); err != nil {
			return fmt.Errorf("legacy import: %w", err)
		}
	}

	users, err := usersclient.New(cfg.UsersGRPCAddr)
	if err != nil {
		return err
	}
	defer func() { _ = users.Close() }()

	gin.SetMode(gin.ReleaseMode)
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           routes.Setup(handlers.Deps{Store: store.New(db), Authors: users, Log: log}, users, cfg.JWTSecret),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("listings listening", slog.String("addr", srv.Addr))
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
