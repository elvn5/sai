// Command trails serves the route catalog, trail condition reports and
// group outings.
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

	pkgconfig "github.com/elvn5/sai/services/pkg/config"
	"github.com/elvn5/sai/services/pkg/database"
	"github.com/elvn5/sai/services/pkg/logger"
	"github.com/elvn5/sai/services/pkg/telegram"
	"github.com/elvn5/sai/services/pkg/usersclient"
	"github.com/elvn5/sai/services/trails/internal/config"
	"github.com/elvn5/sai/services/trails/internal/handlers"
	"github.com/elvn5/sai/services/trails/internal/migrations"
	"github.com/elvn5/sai/services/trails/internal/notify"
	"github.com/elvn5/sai/services/trails/internal/routes"
	"github.com/elvn5/sai/services/trails/internal/store"
)

const shutdownTimeout = 15 * time.Second

func main() {
	log := logger.New("trails")
	if err := run(log); err != nil {
		log.Error("trails stopped", slog.Any("error", err))
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

	users, err := usersclient.New(cfg.UsersGRPCAddr)
	if err != nil {
		return err
	}
	defer func() { _ = users.Close() }()

	st := store.New(db)
	deps := handlers.Deps{Store: st, Authors: users, Log: log}
	if cfg.BotToken != "" {
		notifier := &notify.Notifier{
			Store: st, Users: users, Sender: telegram.New(cfg.BotToken, cfg.TelegramAPIURL), Log: log,
		}
		deps.Notify = notifier
		go notifier.Run(ctx)
	} else {
		log.Warn("BOT_TOKEN not set: Telegram notifications about outings are off")
	}

	gin.SetMode(gin.ReleaseMode)
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           routes.Setup(deps, users, cfg.JWTSecret),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("trails listening", slog.String("addr", srv.Addr))
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
