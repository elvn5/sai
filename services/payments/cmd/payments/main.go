// Command payments takes online payments for bookings through Bakai Bank
// Open Banking and keeps the books of what's owed to partners.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/payments/internal/config"
	"github.com/elvn5/sai/services/payments/internal/handlers"
	"github.com/elvn5/sai/services/payments/internal/listingsclient"
	"github.com/elvn5/sai/services/payments/internal/migrations"
	"github.com/elvn5/sai/services/payments/internal/provider"
	"github.com/elvn5/sai/services/payments/internal/routes"
	"github.com/elvn5/sai/services/payments/internal/store"
	"github.com/elvn5/sai/services/payments/internal/worker"
	pkgconfig "github.com/elvn5/sai/services/pkg/config"
	"github.com/elvn5/sai/services/pkg/database"
	"github.com/elvn5/sai/services/pkg/logger"
	"github.com/elvn5/sai/services/pkg/telegram"
	"github.com/elvn5/sai/services/pkg/usersclient"
)

const shutdownTimeout = 15 * time.Second

func main() {
	log := logger.New("payments")
	if err := run(log); err != nil {
		log.Error("payments stopped", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := pkgconfig.Load[config.Config]()
	if err != nil {
		return err
	}
	siteURL := strings.TrimRight(cfg.SiteURL, "/")

	var gateway provider.Provider
	var fake *provider.Fake
	switch cfg.Provider {
	case "bakai":
		if cfg.BakaiLogin == "" || cfg.BakaiPassword == "" || cfg.BakaiQRType == "" {
			return errors.New("PAYMENTS_PROVIDER=bakai needs BAKAI_LOGIN, BAKAI_PASSWORD and BAKAI_QR_TYPE")
		}
		gateway = &provider.Bakai{BaseURL: cfg.BakaiAPIURL, Login: cfg.BakaiLogin, Password: cfg.BakaiPassword, QRType: cfg.BakaiQRType}
	case "fake":
		fake = &provider.Fake{SiteURL: siteURL}
		gateway = fake
		log.Warn("PAYMENTS_PROVIDER=fake: payments go to a test page, no money moves")
	default:
		return fmt.Errorf("unknown PAYMENTS_PROVIDER %q", cfg.Provider)
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
	listings, err := listingsclient.New(cfg.ListingsGRPCAddr)
	if err != nil {
		return err
	}
	defer func() { _ = listings.Close() }()

	st := store.New(db)
	w := &worker.Worker{Store: st, Provider: gateway, Chats: users, SiteURL: siteURL, Log: log}
	if cfg.BotToken != "" {
		w.Sender = telegram.New(cfg.BotToken, "")
	}
	go w.Run(ctx)

	deps := handlers.Deps{
		Store: st, Provider: gateway, Tours: listings, Authors: users, Checker: w,
		FeePercent: cfg.FeePercent, PaymentTTL: time.Duration(cfg.PaymentTTLMinutes) * time.Minute,
		SiteURL: siteURL, Log: log,
	}
	gin.SetMode(gin.ReleaseMode)
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           routes.Setup(deps, users, cfg.JWTSecret, fake),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("payments listening", slog.String("addr", srv.Addr), slog.String("provider", gateway.Name()))
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
