// Package grpcserver implements the media service's internal gRPC API.
package grpcserver

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/elvn5/sai/services/media/internal/storage"
	mediav1 "github.com/elvn5/sai/services/proto/gen/go/media/v1"
)

type Server struct {
	mediav1.UnimplementedMediaServiceServer
	R2  *storage.R2
	Log *slog.Logger
}

func (s *Server) DeleteObject(ctx context.Context, req *mediav1.DeleteObjectRequest) (*mediav1.DeleteObjectResponse, error) {
	key, ok := s.R2.KeyFromPublicURL(req.GetPublicUrl())
	if !ok {
		return &mediav1.DeleteObjectResponse{Deleted: false}, nil
	}
	err := s.R2.Delete(ctx, key)
	switch {
	case errors.Is(err, storage.ErrNotConfigured):
		return nil, status.Error(codes.FailedPrecondition, "storage service is not configured")
	case err != nil:
		s.Log.ErrorContext(ctx, "failed to delete R2 object", slog.String("key", key), slog.Any("error", err))
		return nil, status.Error(codes.Internal, "could not delete file")
	}
	return &mediav1.DeleteObjectResponse{Deleted: true}, nil
}
