// Command media serves uploads: presigned direct-to-R2 uploads, image
// uploads re-encoded to WebP, guide attachments, and the local /uploads
// files.
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

	"github.com/elvn5/sai/services/media/internal/config"
	"github.com/elvn5/sai/services/media/internal/grpcserver"
	"github.com/elvn5/sai/services/media/internal/routes"
	"github.com/elvn5/sai/services/media/internal/storage"
	pkgconfig "github.com/elvn5/sai/services/pkg/config"
	"github.com/elvn5/sai/services/pkg/logger"
	"github.com/elvn5/sai/services/pkg/middleware"
	"github.com/elvn5/sai/services/pkg/usersclient"
	mediav1 "github.com/elvn5/sai/services/proto/gen/go/media/v1"
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

	r2 := storage.NewR2(cfg)

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
		Addr:              ":" + cfg.Port,
		Handler:           routes.Setup(cfg, r2, users, log),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	grpcSrv := grpc.NewServer()
	mediav1.RegisterMediaServiceServer(grpcSrv, &grpcserver.Server{R2: r2, Log: log})
	grpcLis, err := (&net.ListenConfig{}).Listen(ctx, "tcp", ":"+cfg.GRPCPort)
	if err != nil {
		return fmt.Errorf("listen gRPC: %w", err)
	}

	errCh := make(chan error, 2)
	go func() {
		log.Info("media listening", slog.String("addr", srv.Addr), slog.String("uploads_dir", cfg.UploadsDir))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	go func() {
		log.Info("media gRPC listening", slog.String("addr", grpcLis.Addr().String()))
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
