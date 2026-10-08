// Command users serves account data to the other backend services over
// gRPC. Stage 4 of the migration adds its public HTTP API (sign-in,
// registration, partners) and moves accounts into its own database.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	pkgconfig "github.com/elvn5/sai/services/pkg/config"
	"github.com/elvn5/sai/services/pkg/logger"
	usersv1 "github.com/elvn5/sai/services/proto/gen/go/users/v1"
	"github.com/elvn5/sai/services/users/internal/config"
	"github.com/elvn5/sai/services/users/internal/grpcserver"
	"github.com/elvn5/sai/services/users/internal/store"
)

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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := pgxpool.New(ctx, cfg.DSN())
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer db.Close()
	if err := db.Ping(ctx); err != nil {
		return fmt.Errorf("database: %w", err)
	}

	srv := grpc.NewServer()
	usersv1.RegisterUsersServiceServer(srv, &grpcserver.Server{Store: store.New(db), Log: log})
	lis, err := (&net.ListenConfig{}).Listen(ctx, "tcp", ":"+cfg.GRPCPort)
	if err != nil {
		return fmt.Errorf("listen gRPC: %w", err)
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("users gRPC listening", slog.String("addr", lis.Addr().String()))
		errCh <- srv.Serve(lis)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}
	log.Info("shutting down")
	srv.GracefulStop()
	return nil
}
