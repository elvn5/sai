// Command users owns accounts: sign-in (password, Telegram, admin),
// registration, partner applications and support contacts over HTTP, and
// account lookups for the other services and the bot over gRPC.
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

	pkgauth "github.com/elvn5/sai/services/pkg/auth"
	pkgconfig "github.com/elvn5/sai/services/pkg/config"
	"github.com/elvn5/sai/services/pkg/database"
	"github.com/elvn5/sai/services/pkg/logger"
	usersv1 "github.com/elvn5/sai/services/proto/gen/go/users/v1"
	"github.com/elvn5/sai/services/users/internal/config"
	"github.com/elvn5/sai/services/users/internal/email"
	"github.com/elvn5/sai/services/users/internal/grpcserver"
	"github.com/elvn5/sai/services/users/internal/handlers"
	"github.com/elvn5/sai/services/users/internal/legacyimport"
	"github.com/elvn5/sai/services/users/internal/migrations"
	"github.com/elvn5/sai/services/users/internal/routes"
	"github.com/elvn5/sai/services/users/internal/session"
	"github.com/elvn5/sai/services/users/internal/store"
)

const shutdownTimeout = 15 * time.Second

func main() {
	log := logger.New("users")
	if err := run(log); err != nil {
		log.Error("users stopped", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := pkgconfig.Load[config.Config]()
	if err != nil {
		return err
	}
	signer, err := pkgauth.NewSigner(cfg.JWTSecret, cfg.JWTExpiresIn)
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
	if cfg.ResendAPIKey == "" {
		log.Warn("RESEND_API_KEY not set: email registrations will fail")
	}

	st := store.New(db)
	gin.SetMode(gin.ReleaseMode)
	srv := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: routes.Setup(handlers.Deps{
			Store: st, Signer: signer, Cookies: session.NewCookies(cfg.CookieSecure),
			Email:         email.NewResend(cfg.ResendAPIKey, cfg.EmailFrom, cfg.ResendBaseURL),
			AdminPassword: cfg.AdminPassword, BotToken: cfg.BotToken, Log: log,
		}, cfg.JWTSecret),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	grpcSrv := grpc.NewServer()
	usersv1.RegisterUsersServiceServer(grpcSrv, &grpcserver.Server{Store: st, Log: log})
	lis, err := (&net.ListenConfig{}).Listen(ctx, "tcp", ":"+cfg.GRPCPort)
	if err != nil {
		return fmt.Errorf("listen gRPC: %w", err)
	}

	errCh := make(chan error, 2)
	go func() {
		log.Info("users listening", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	go func() {
		log.Info("users gRPC listening", slog.String("addr", lis.Addr().String()))
		if err := grpcSrv.Serve(lis); err != nil {
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
