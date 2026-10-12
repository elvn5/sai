// Package grpcserver implements the users service's internal gRPC API for
// the other services and the Telegram bot.
package grpcserver

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	usersv1 "github.com/elvn5/sai/services/proto/gen/go/users/v1"
	"github.com/elvn5/sai/services/users/internal/handlers"
	"github.com/elvn5/sai/services/users/internal/models"
	"github.com/elvn5/sai/services/users/internal/store"
	"github.com/elvn5/sai/services/users/internal/verification"
)

const maxBatch = 500

type Server struct {
	usersv1.UnimplementedUsersServiceServer
	Store *store.Store
	Log   *slog.Logger
	Now   func() time.Time
}

func (s *Server) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Server) GetUser(ctx context.Context, req *usersv1.GetUserRequest) (*usersv1.GetUserResponse, error) {
	u, err := s.Store.ByID(ctx, req.GetId())
	if err != nil {
		return nil, s.storeError(ctx, err, "user not found")
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
	users, err := s.Store.ByIDs(ctx, req.GetIds())
	if err != nil {
		return nil, s.storeError(ctx, err, "")
	}
	out := make([]*usersv1.User, len(users))
	for i, u := range users {
		out[i] = toProto(u)
	}
	return &usersv1.GetUsersResponse{Users: out}, nil
}

func (s *Server) GetTelegramChats(ctx context.Context, req *usersv1.GetTelegramChatsRequest) (*usersv1.GetTelegramChatsResponse, error) {
	if len(req.GetUserIds()) > maxBatch {
		return nil, status.Errorf(codes.InvalidArgument, "at most %d ids per call", maxBatch)
	}
	chats := map[string]int64{}
	if len(req.GetUserIds()) > 0 {
		var err error
		if chats, err = s.Store.TelegramChats(ctx, req.GetUserIds()); err != nil {
			return nil, s.storeError(ctx, err, "")
		}
	}
	return &usersv1.GetTelegramChatsResponse{ChatIds: chats}, nil
}

func (s *Server) UpsertTelegramUser(ctx context.Context, req *usersv1.UpsertTelegramUserRequest) (*usersv1.UpsertTelegramUserResponse, error) {
	if req.GetTelegramId() == "" {
		return nil, status.Error(codes.InvalidArgument, "telegram_id is required")
	}
	u, err := handlers.UpsertTelegramUser(ctx, s.Store, handlers.TelegramProfile{
		TelegramID: req.GetTelegramId(), Username: req.Username, FirstName: req.FirstName, LastName: req.LastName,
	})
	if err != nil {
		return nil, s.storeError(ctx, err, "")
	}
	return &usersv1.UpsertTelegramUserResponse{Account: toAccount(u)}, nil
}

// ConfirmTelegramVerification: an unknown, expired or already-used token is
// NOT_FOUND alike — the bot answers all of them the same way.
func (s *Server) ConfirmTelegramVerification(ctx context.Context, req *usersv1.ConfirmTelegramVerificationRequest) (*usersv1.ConfirmTelegramVerificationResponse, error) {
	u, err := s.Store.ByVerificationHash(ctx, verification.Hash(req.GetToken()))
	if err != nil {
		return nil, s.storeError(ctx, err, "invalid or expired token")
	}
	if u.IsVerified || u.VerificationCodeExpiresAt == nil || u.VerificationCodeExpiresAt.Before(s.now()) {
		return nil, status.Error(codes.NotFound, "invalid or expired token")
	}
	store.ClearVerification(&u)
	saved, err := s.Store.Save(ctx, u)
	if err != nil {
		return nil, s.storeError(ctx, err, "")
	}
	return &usersv1.ConfirmTelegramVerificationResponse{Account: toAccount(saved)}, nil
}

func (s *Server) LinkTelegram(ctx context.Context, req *usersv1.LinkTelegramRequest) (*usersv1.LinkTelegramResponse, error) {
	if req.GetTelegramId() == "" {
		return nil, status.Error(codes.InvalidArgument, "telegram_id is required")
	}
	u, err := s.Store.ByTelegramLinkToken(ctx, verification.Hash(req.GetToken()), s.now())
	if err != nil {
		return nil, s.storeError(ctx, err, "invalid or expired token")
	}
	if u.TelegramID != nil && *u.TelegramID != req.GetTelegramId() {
		return nil, status.Error(codes.AlreadyExists, "the account is linked to another Telegram account")
	}

	moveFrom := ""
	other, err := s.Store.ByTelegramID(ctx, req.GetTelegramId())
	switch {
	case errors.Is(err, store.ErrNotFound):
	case err != nil:
		return nil, s.storeError(ctx, err, "")
	case other.ID != u.ID && (other.Email != nil || other.Phone != nil || other.PasswordHash != nil):
		return nil, status.Error(codes.AlreadyExists, "this Telegram account belongs to another Ermekten account")
	case other.ID != u.ID && !req.GetMoveFromOtherAccount():
		return nil, status.Error(codes.FailedPrecondition, "this Telegram account has its own Ermekten account")
	case other.ID != u.ID:
		moveFrom = other.ID
		s.Log.InfoContext(ctx, "moving a Telegram id to a site account", slog.String("from", other.ID), slog.String("to", u.ID))
	}

	linked, err := s.Store.LinkTelegram(ctx, u.ID, req.GetTelegramId(), req.Username, moveFrom)
	if err != nil {
		return nil, s.storeError(ctx, err, "")
	}
	return &usersv1.LinkTelegramResponse{Account: toAccount(linked)}, nil
}

func (s *Server) UpdateProfile(ctx context.Context, req *usersv1.UpdateProfileRequest) (*usersv1.UpdateProfileResponse, error) {
	u, err := s.Store.ByID(ctx, req.GetUserId())
	if err != nil {
		return nil, s.storeError(ctx, err, "user not found")
	}
	if req.FirstName != nil {
		u.FirstName = req.FirstName
	}
	if req.Phone != nil {
		u.Phone = req.Phone
	}
	saved, err := s.Store.Save(ctx, u)
	if errors.Is(err, store.ErrDuplicate) {
		return nil, status.Error(codes.AlreadyExists, "phone number is used by another account")
	}
	if err != nil {
		return nil, s.storeError(ctx, err, "")
	}
	return &usersv1.UpdateProfileResponse{Account: toAccount(saved)}, nil
}

func (s *Server) GetSupportInfo(ctx context.Context, _ *usersv1.GetSupportInfoRequest) (*usersv1.GetSupportInfoResponse, error) {
	info, err := s.Store.Support(ctx)
	if errors.Is(err, store.ErrNotFound) {
		return &usersv1.GetSupportInfoResponse{Support: &usersv1.SupportInfo{Message: store.DefaultSupportMessage}}, nil
	}
	if err != nil {
		return nil, s.storeError(ctx, err, "")
	}
	return &usersv1.GetSupportInfoResponse{Support: &usersv1.SupportInfo{
		Message: info.Message, Phone: info.Phone, TelegramContact: info.TelegramContact,
		Email: info.Email, WorkingHours: info.WorkingHours,
	}}, nil
}

func (s *Server) storeError(ctx context.Context, err error, notFound string) error {
	if errors.Is(err, store.ErrNotFound) && notFound != "" {
		return status.Error(codes.NotFound, notFound)
	}
	s.Log.ErrorContext(ctx, "users gRPC call failed", slog.Any("error", err))
	return status.Error(codes.Internal, "internal error")
}

var partnerStatuses = map[string]usersv1.PartnerStatus{
	"none":     usersv1.PartnerStatus_PARTNER_STATUS_NONE,
	"pending":  usersv1.PartnerStatus_PARTNER_STATUS_PENDING,
	"approved": usersv1.PartnerStatus_PARTNER_STATUS_APPROVED,
	"rejected": usersv1.PartnerStatus_PARTNER_STATUS_REJECTED,
}

func toProto(u models.User) *usersv1.User {
	return &usersv1.User{
		Id: u.ID, FirstName: u.FirstName, Username: u.Username, AvatarUrl: u.AvatarURL,
		PartnerStatus: partnerStatuses[u.PartnerStatus],
	}
}

func toAccount(u models.User) *usersv1.TelegramAccount {
	return &usersv1.TelegramAccount{Id: u.ID, FirstName: u.FirstName, Username: u.Username, Phone: u.Phone}
}
