// Package usersclient calls the users service's internal gRPC API: session
// checks for the auth middleware, and public author profiles.
package usersclient

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/elvn5/sai/services/pkg/middleware"
	"github.com/elvn5/sai/services/pkg/telegram"
	usersv1 "github.com/elvn5/sai/services/proto/gen/go/users/v1"
)

const callTimeout = 5 * time.Second

// Author is the public part of a profile shown next to user content —
// never email or phone (the NestJS PublicAuthor shape).
type Author struct {
	ID        string  `json:"id"`
	FirstName *string `json:"firstName"`
	Username  *string `json:"username"`
	AvatarURL *string `json:"avatarUrl"`
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

func (c *Client) Close() error { return c.conn.Close() }

var partnerStatuses = map[usersv1.PartnerStatus]string{
	usersv1.PartnerStatus_PARTNER_STATUS_NONE:     "none",
	usersv1.PartnerStatus_PARTNER_STATUS_PENDING:  "pending",
	usersv1.PartnerStatus_PARTNER_STATUS_APPROVED: "approved",
	usersv1.PartnerStatus_PARTNER_STATUS_REJECTED: "rejected",
}

// LookupUser implements middleware.UserLookup.
func (c *Client) LookupUser(ctx context.Context, id string) (middleware.SessionUser, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	res, err := c.client.GetUser(ctx, &usersv1.GetUserRequest{Id: id})
	if status.Code(err) == codes.NotFound {
		return middleware.SessionUser{}, middleware.ErrUserNotFound
	}
	if err != nil {
		return middleware.SessionUser{}, fmt.Errorf("get user: %w", err)
	}
	return middleware.SessionUser{ID: res.GetUser().GetId(), PartnerStatus: partnerStatuses[res.GetUser().GetPartnerStatus()]}, nil
}

// Authors returns the public profiles for ids; unknown ids are absent.
func (c *Client) Authors(ctx context.Context, ids []string) (map[string]Author, error) {
	out := make(map[string]Author, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	res, err := c.client.GetUsers(ctx, &usersv1.GetUsersRequest{Ids: ids})
	if err != nil {
		return nil, fmt.Errorf("get users: %w", err)
	}
	for _, u := range res.GetUsers() {
		out[u.GetId()] = Author{ID: u.GetId(), FirstName: u.FirstName, Username: u.Username, AvatarURL: u.AvatarUrl}
	}
	return out, nil
}

// PartnerStatuses returns the partner status ("none", "pending",
// "approved", "rejected") of each known id; unknown ids are absent.
func (c *Client) PartnerStatuses(ctx context.Context, ids []string) (map[string]string, error) {
	out := make(map[string]string, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	res, err := c.client.GetUsers(ctx, &usersv1.GetUsersRequest{Ids: ids})
	if err != nil {
		return nil, fmt.Errorf("get users: %w", err)
	}
	for _, u := range res.GetUsers() {
		out[u.GetId()] = partnerStatuses[u.GetPartnerStatus()]
	}
	return out, nil
}

// TelegramChats returns the Telegram chat id of each user who has a
// linked Telegram account; others are absent.
func (c *Client) TelegramChats(ctx context.Context, ids []string) (map[string]int64, error) {
	if len(ids) == 0 {
		return map[string]int64{}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	res, err := c.client.GetTelegramChats(ctx, &usersv1.GetTelegramChatsRequest{UserIds: ids})
	if err != nil {
		return nil, fmt.Errorf("get telegram chats: %w", err)
	}
	return res.GetChatIds(), nil
}

// TelegramChat is a user's private chat with the bot and the language it
// reads the bot in (Russian until the bot reports one).
type TelegramChat struct {
	ID   int64
	Lang telegram.Lang
}

// TelegramRecipients returns the Telegram chat of each user who has a
// linked Telegram account, with its language; others are absent.
func (c *Client) TelegramRecipients(ctx context.Context, ids []string) (map[string]TelegramChat, error) {
	out := make(map[string]TelegramChat, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	res, err := c.client.GetTelegramChats(ctx, &usersv1.GetTelegramChatsRequest{UserIds: ids})
	if err != nil {
		return nil, fmt.Errorf("get telegram chats: %w", err)
	}
	for id, chat := range res.GetChatIds() {
		out[id] = TelegramChat{ID: chat, Lang: telegram.ParseLang(res.GetLanguages()[id])}
	}
	return out, nil
}
