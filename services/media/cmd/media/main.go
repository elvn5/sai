// Command media serves uploads: presigned direct-to-R2 uploads, image
// uploads re-encoded to WebP, guide attachments, and the local /uploads
// files.
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

	"github.com/elvn5/sai/services/media/internal/config"
	"github.com/elvn5/sai/services/media/internal/routes"
	pkgconfig "github.com/elvn5/sai/services/pkg/config"
	"github.com/elvn5/sai/services/pkg/logger"
)

const shutdownTimeout = 15 * time.Second

func main() {
	log := logger.New("media")
	if err := run(log); err != nil {
		log.Error("media stopped", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := pkgconfig.Load[config.Config]()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cfg.UploadsDir, 0o755); err != nil { //nolint:gosec // public files
		return fmt.Errorf("create uploads dir: %w", err)
	}

	gin.SetMode(gin.ReleaseMode)
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           routes.Setup(cfg, log),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		log.Info("media listening", slog.String("addr", srv.Addr), slog.String("uploads_dir", cfg.UploadsDir))
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
