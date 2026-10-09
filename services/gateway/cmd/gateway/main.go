// Command gateway is the public entry point of the backend: it serves the
// REST API the Nuxt apps call and routes each request to the service that
// owns it.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/gateway/internal/config"
	"github.com/elvn5/sai/services/gateway/internal/routes"
	pkgconfig "github.com/elvn5/sai/services/pkg/config"
	"github.com/elvn5/sai/services/pkg/logger"
)

const shutdownTimeout = 15 * time.Second

func main() {
	log := logger.New("gateway")
	if err := run(log); err != nil {
		log.Error("gateway stopped", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := pkgconfig.Load[config.Config]()
	if err != nil {
		return err
	}

	gin.SetMode(gin.ReleaseMode)
	router, err := routes.Setup(cfg, log)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		log.Info("gateway listening", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
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
