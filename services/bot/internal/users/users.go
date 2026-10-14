// Package users is the bot's view of the users service: accounts of
// Telegram users, linking Telegram to site accounts and support contacts,
// over its internal gRPC API.
package users

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	usersv1 "github.com/elvn5/sai/services/proto/gen/go/users/v1"
)

const callTimeout = 10 * time.Second

// ErrPhoneTaken: another account already uses the phone number.
var ErrPhoneTaken = errors.New("phone is taken")

// Account is a user as the bot shows it. Nil fields are unset.
type Account struct {
	ID        string
	FirstName *string
	Username  *string
	Phone     *string
}

// Profile is what Telegram tells about a user.
type Profile struct {
	TelegramID int64
	Username   string
	FirstName  string
	LastName   string
}

// LinkStatus is how an attempt to link Telegram to a site account ended.
type LinkStatus int

const (
	Linked LinkStatus = iota
	// Unknown, expired or already used token.
	LinkInvalid
	// The Telegram account belongs to another account with its own sign-in.
	LinkTaken
	// It belongs to a Telegram-only account; linking moves it from there.
	LinkNeedsMove
)

type LinkResult struct {
	Status  LinkStatus
	Account Account // when Linked
}

type Support struct {
	Message         string
	Phone           *string
	TelegramContact *string
	Email           *string
	WorkingHours    *string
}

type Client struct {
	conn   *grpc.ClientConn
	client usersv1.UsersServiceClient
}

// New connects lazily to addr (host:port) over the private network.
func New(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("users gRPC client: %w", err)
	}
	return &Client{conn: conn, client: usersv1.NewUsersServiceClient(conn)}, nil
}

// NewWithClient wraps an existing client (tests).
func NewWithClient(client usersv1.UsersServiceClient) *Client {
	return &Client{client: client}
}

func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func account(a *usersv1.TelegramAccount) Account {
	return Account{ID: a.GetId(), FirstName: a.FirstName, Username: a.Username, Phone: a.Phone}
}

// optional is a proto optional string: unset when empty, as Telegram
// leaves out what a user doesn't have.
func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// FindOrCreate returns the account of a Telegram user, creating it on
// first contact.
func (c *Client) FindOrCreate(ctx context.Context, p Profile) (Account, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	res, err := c.client.UpsertTelegramUser(ctx, &usersv1.UpsertTelegramUserRequest{
		TelegramId: strconv.FormatInt(p.TelegramID, 10),
		Username:   optional(p.Username),
		FirstName:  optional(p.FirstName),
		LastName:   optional(p.LastName),
	})
	if err != nil {
		return Account{}, fmt.Errorf("upsert telegram user: %w", err)
	}
	return account(res.GetAccount()), nil
}

// ConfirmVerification completes a phone registration started on the
// site, which hands the user a t.me deep link. Nil for an unknown,
// expired or already used token.
func (c *Client) ConfirmVerification(ctx context.Context, token string) (*Account, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	res, err := c.client.ConfirmTelegramVerification(ctx, &usersv1.ConfirmTelegramVerificationRequest{Token: token})
	if status.Code(err) == codes.NotFound {
		return nil, nil //nolint:nilnil // nil account: the link is invalid
	}
	if err != nil {
		return nil, fmt.Errorf("confirm telegram verification: %w", err)
	}
	a := account(res.GetAccount())
	return &a, nil
}

// Link links the Telegram user to the site account that issued token (the
// site's "Link Telegram" deep link). move takes Telegram away from its
// Telegram-only account.
func (c *Client) Link(ctx context.Context, token string, telegramID int64, username string, move bool) (LinkResult, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	res, err := c.client.LinkTelegram(ctx, &usersv1.LinkTelegramRequest{
		Token:                token,
		TelegramId:           strconv.FormatInt(telegramID, 10),
		Username:             optional(username),
		MoveFromOtherAccount: move,
	})
	switch status.Code(err) {
	case codes.OK:
		return LinkResult{Status: Linked, Account: account(res.GetAccount())}, nil
	case codes.NotFound:
		return LinkResult{Status: LinkInvalid}, nil
	case codes.AlreadyExists:
		return LinkResult{Status: LinkTaken}, nil
	case codes.FailedPrecondition:
		return LinkResult{Status: LinkNeedsMove}, nil
	default:
		return LinkResult{}, fmt.Errorf("link telegram: %w", err)
	}
}

func (c *Client) UpdateFirstName(ctx context.Context, userID, firstName string) error {
	return c.updateProfile(ctx, &usersv1.UpdateProfileRequest{UserId: userID, FirstName: &firstName})
}

// UpdatePhone fails with ErrPhoneTaken if another account uses the number.
func (c *Client) UpdatePhone(ctx context.Context, userID, phone string) error {
	return c.updateProfile(ctx, &usersv1.UpdateProfileRequest{UserId: userID, Phone: &phone})
}

func (c *Client) updateProfile(ctx context.Context, req *usersv1.UpdateProfileRequest) error {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	_, err := c.client.UpdateProfile(ctx, req)
	if status.Code(err) == codes.AlreadyExists {
		return ErrPhoneTaken
	}
	if err != nil {
		return fmt.Errorf("update profile: %w", err)
	}
	return nil
}

// Support returns the support contacts edited in the admin panel (the
// users service falls back to a default message).
func (c *Client) Support(ctx context.Context) (Support, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	res, err := c.client.GetSupportInfo(ctx, &usersv1.GetSupportInfoRequest{})
	if err != nil {
		return Support{}, fmt.Errorf("get support info: %w", err)
	}
	s := res.GetSupport()
	return Support{
		Message:         s.GetMessage(),
		Phone:           s.Phone,
		TelegramContact: s.TelegramContact,
		Email:           s.Email,
		WorkingHours:    s.WorkingHours,
	}, nil
}
