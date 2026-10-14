package users_test

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/elvn5/sai/services/bot/internal/users"
	usersv1 "github.com/elvn5/sai/services/proto/gen/go/users/v1"
)

// fakeService answers the users gRPC calls the bot makes.
type fakeService struct {
	usersv1.UsersServiceClient
	err      error
	upserted *usersv1.UpsertTelegramUserRequest
	linked   *usersv1.LinkTelegramRequest
}

func str(s string) *string { return &s }

func (f *fakeService) UpsertTelegramUser(_ context.Context, in *usersv1.UpsertTelegramUserRequest, _ ...grpc.CallOption) (*usersv1.UpsertTelegramUserResponse, error) {
	f.upserted = in
	return &usersv1.UpsertTelegramUserResponse{Account: &usersv1.TelegramAccount{Id: "u1", FirstName: in.FirstName}}, f.err
}

func (f *fakeService) ConfirmTelegramVerification(context.Context, *usersv1.ConfirmTelegramVerificationRequest, ...grpc.CallOption) (*usersv1.ConfirmTelegramVerificationResponse, error) {
	return &usersv1.ConfirmTelegramVerificationResponse{Account: &usersv1.TelegramAccount{Id: "u1"}}, f.err
}

func (f *fakeService) LinkTelegram(_ context.Context, in *usersv1.LinkTelegramRequest, _ ...grpc.CallOption) (*usersv1.LinkTelegramResponse, error) {
	f.linked = in
	return &usersv1.LinkTelegramResponse{Account: &usersv1.TelegramAccount{Id: "u1", FirstName: str("Айгерим")}}, f.err
}

func (f *fakeService) UpdateProfile(context.Context, *usersv1.UpdateProfileRequest, ...grpc.CallOption) (*usersv1.UpdateProfileResponse, error) {
	return &usersv1.UpdateProfileResponse{}, f.err
}

func TestFindOrCreateLeavesOutMissingFields(t *testing.T) {
	svc := &fakeService{}
	c := users.NewWithClient(svc)
	a, err := c.FindOrCreate(context.Background(), users.Profile{TelegramID: 123456789012, FirstName: "Айгерим"})
	if err != nil || a.ID != "u1" {
		t.Fatalf("find or create: %+v %v", a, err)
	}
	if svc.upserted.GetTelegramId() != "123456789012" || svc.upserted.Username != nil || svc.upserted.LastName != nil ||
		svc.upserted.GetFirstName() != "Айгерим" {
		t.Errorf("request: %+v", svc.upserted)
	}
}

func TestErrorMapping(t *testing.T) {
	ctx := context.Background()
	for code, want := range map[codes.Code]users.LinkStatus{
		codes.OK:                 users.Linked,
		codes.NotFound:           users.LinkInvalid,
		codes.AlreadyExists:      users.LinkTaken,
		codes.FailedPrecondition: users.LinkNeedsMove,
	} {
		svc := &fakeService{err: status.Error(code, "x")}
		if code == codes.OK {
			svc.err = nil
		}
		res, err := users.NewWithClient(svc).Link(ctx, "tok", 42, "aigerim", true)
		if err != nil || res.Status != want {
			t.Errorf("%s: %+v %v", code, res, err)
		}
		if !svc.linked.GetMoveFromOtherAccount() || svc.linked.GetUsername() != "aigerim" {
			t.Errorf("%s request: %+v", code, svc.linked)
		}
	}
	if _, err := users.NewWithClient(&fakeService{err: status.Error(codes.Unavailable, "down")}).Link(ctx, "t", 1, "", false); err == nil {
		t.Error("unavailable is an error")
	}

	if a, err := users.NewWithClient(&fakeService{err: status.Error(codes.NotFound, "x")}).ConfirmVerification(ctx, "t"); a != nil || err != nil {
		t.Errorf("unknown token: %+v %v", a, err)
	}
	if err := users.NewWithClient(&fakeService{err: status.Error(codes.AlreadyExists, "x")}).UpdatePhone(ctx, "u1", "+996"); !errors.Is(err, users.ErrPhoneTaken) {
		t.Errorf("phone taken: %v", err)
	}
}
