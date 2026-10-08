// Package grpcserver implements the users service's internal gRPC API.
package grpcserver

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	usersv1 "github.com/elvn5/sai/services/proto/gen/go/users/v1"
	"github.com/elvn5/sai/services/users/internal/store"
)

const maxBatch = 500

type Server struct {
	usersv1.UnimplementedUsersServiceServer
	Store *store.Store
	Log   *slog.Logger
}

func (s *Server) GetUser(ctx context.Context, req *usersv1.GetUserRequest) (*usersv1.GetUserResponse, error) {
	u, err := s.Store.User(ctx, req.GetId())
	switch {
	case errors.Is(err, store.ErrNotFound):
		return nil, status.Error(codes.NotFound, "user not found")
	case err != nil:
		s.Log.ErrorContext(ctx, "get user failed", slog.Any("error", err))
		return nil, status.Error(codes.Internal, "could not load user")
	}
	return &usersv1.GetUserResponse{User: toProto(u)}, nil
}

func (s *Server) GetUsers(ctx context.Context, req *usersv1.GetUsersRequest) (*usersv1.GetUsersResponse, error) {
	if len(req.GetIds()) > maxBatch {
		return nil, status.Errorf(codes.InvalidArgument, "at most %d ids per call", maxBatch)
	}
	if len(req.GetIds()) == 0 {
		return &usersv1.GetUsersResponse{}, nil
	}
	users, err := s.Store.Users(ctx, req.GetIds())
	if err != nil {
		s.Log.ErrorContext(ctx, "get users failed", slog.Any("error", err))
		return nil, status.Error(codes.Internal, "could not load users")
	}
	out := make([]*usersv1.User, len(users))
	for i, u := range users {
		out[i] = toProto(u)
	}
	return &usersv1.GetUsersResponse{Users: out}, nil
}

var partnerStatuses = map[string]usersv1.PartnerStatus{
	"none":     usersv1.PartnerStatus_PARTNER_STATUS_NONE,
	"pending":  usersv1.PartnerStatus_PARTNER_STATUS_PENDING,
	"approved": usersv1.PartnerStatus_PARTNER_STATUS_APPROVED,
	"rejected": usersv1.PartnerStatus_PARTNER_STATUS_REJECTED,
}

func toProto(u store.User) *usersv1.User {
	return &usersv1.User{
		Id:            u.ID,
		FirstName:     u.FirstName,
		Username:      u.Username,
		AvatarUrl:     u.AvatarURL,
		PartnerStatus: partnerStatuses[u.PartnerStatus],
	}
}
