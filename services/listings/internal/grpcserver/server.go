// Package grpcserver implements the listings service's internal gRPC API
// (services/proto/listings/v1), for other backend services.
package grpcserver

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/elvn5/sai/services/listings/internal/store"
	listingsv1 "github.com/elvn5/sai/services/proto/gen/go/listings/v1"
)

type Server struct {
	listingsv1.UnimplementedListingsServiceServer
	Store *store.Store
	Log   *slog.Logger
}

func (s *Server) GetTour(ctx context.Context, req *listingsv1.GetTourRequest) (*listingsv1.GetTourResponse, error) {
	if uuid.Validate(req.GetId()) != nil {
		return nil, status.Error(codes.NotFound, "tour not found")
	}
	t, err := s.Store.Tour(ctx, req.GetId())
	if errors.Is(err, store.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "tour not found")
	}
	if err != nil {
		s.Log.ErrorContext(ctx, "listings gRPC call failed", slog.Any("error", err))
		return nil, status.Error(codes.Internal, "internal error")
	}
	tour := &listingsv1.Tour{
		Id: t.ID, Slug: t.Slug, Locale: t.Locale, Title: t.Title, StartDate: t.StartDate, EndDate: t.EndDate,
		UserId: t.UserID, IsPublished: t.IsPublished, CoverImageUrl: t.CoverImageURL,
	}
	if t.Capacity != nil {
		capacity := int32(*t.Capacity) //nolint:gosec // capacity is validated to at most 500
		tour.Capacity = &capacity
	}
	if t.Price != nil {
		price := int32(*t.Price) //nolint:gosec // prices are validated far below 2^31
		tour.Price = &price
	}
	return &listingsv1.GetTourResponse{Tour: tour}, nil
}
