// Command listings serves what travelers book: hotels, local guides and
// tours, with their reviews and the owners' posts.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"github.com/elvn5/sai/services/listings/internal/alerts"
	"github.com/elvn5/sai/services/listings/internal/config"
	"github.com/elvn5/sai/services/listings/internal/grpcserver"
	"github.com/elvn5/sai/services/listings/internal/handlers"
	"github.com/elvn5/sai/services/listings/internal/legacyimport"
	"github.com/elvn5/sai/services/listings/internal/migrations"
	"github.com/elvn5/sai/services/listings/internal/purge"
	"github.com/elvn5/sai/services/listings/internal/routes"
	"github.com/elvn5/sai/services/listings/internal/store"
	pkgconfig "github.com/elvn5/sai/services/pkg/config"
	"github.com/elvn5/sai/services/pkg/database"
	"github.com/elvn5/sai/services/pkg/logger"
	"github.com/elvn5/sai/services/pkg/partnerpurge"
	"github.com/elvn5/sai/services/pkg/telegram"
	"github.com/elvn5/sai/services/pkg/usersclient"
	listingsv1 "github.com/elvn5/sai/services/proto/gen/go/listings/v1"
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
	// In the background: it waits for the users service.
	go partnerpurge.Run(ctx, db, users, purge.Service(), log)

	st := store.New(db)
	deps := handlers.Deps{Store: st, Authors: users, Log: log}
	if cfg.BotToken != "" {
		deps.Alerts = &alerts.Notifier{
			Store: st, Chats: users, Sender: telegram.New(cfg.BotToken, cfg.TelegramAPIURL), Log: log,
		}
	}
	grpcSrv := grpc.NewServer()
	listingsv1.RegisterListingsServiceServer(grpcSrv, &grpcserver.Server{Store: st, Log: log})
	grpcLis, err := (&net.ListenConfig{}).Listen(ctx, "tcp", ":"+cfg.GRPCPort)
	if err != nil {
		return fmt.Errorf("listen gRPC: %w", err)
	}

	gin.SetMode(gin.ReleaseMode)
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           routes.Setup(deps, users, cfg.JWTSecret),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	errCh := make(chan error, 2)
	go func() {
		log.Info("listings listening", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	go func() {
		log.Info("listings gRPC listening", slog.String("addr", grpcLis.Addr().String()))
		if err := grpcSrv.Serve(grpcLis); err != nil {
			errCh <- fmt.Errorf("gRPC server: %w", err)
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
	grpcSrv.GracefulStop()
	return srv.Shutdown(shutdownCtx)
}
