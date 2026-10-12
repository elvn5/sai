package grpcserver_test

import (
	"context"
	"log/slog"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	usersv1 "github.com/elvn5/sai/services/proto/gen/go/users/v1"
	"github.com/elvn5/sai/services/users/internal/grpcserver"
	"github.com/elvn5/sai/services/users/internal/legacyimport"
	"github.com/elvn5/sai/services/users/internal/models"
	"github.com/elvn5/sai/services/users/internal/store"
	"github.com/elvn5/sai/services/users/internal/testdb"
	"github.com/elvn5/sai/services/users/internal/verification"
)

func client(t *testing.T, st *store.Store) usersv1.UsersServiceClient {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := grpc.NewServer()
	usersv1.RegisterUsersServiceServer(srv, &grpcserver.Server{Store: st, Log: slog.New(slog.DiscardHandler)})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return usersv1.NewUsersServiceClient(conn)
}

func str(s string) *string { return &s }

func TestBotRPCs(t *testing.T) {
	db, _ := testdb.New(t, true)
	st := store.New(db)
	c := client(t, st)
	ctx := context.Background()

	// /start: first contact creates, later contact refreshes only given fields.
	first, err := c.UpsertTelegramUser(ctx, &usersv1.UpsertTelegramUserRequest{TelegramId: "42", FirstName: str("Aibek"), Username: str("aibek")})
	if err != nil {
		t.Fatal(err)
	}
	again, err := c.UpsertTelegramUser(ctx, &usersv1.UpsertTelegramUserRequest{TelegramId: "42", FirstName: str("Aibek K")})
	if err != nil || again.GetAccount().GetId() != first.GetAccount().GetId() || again.GetAccount().GetUsername() != "aibek" || again.GetAccount().GetFirstName() != "Aibek K" {
		t.Fatalf("upsert again: %v %v", again, err)
	}
	id := first.GetAccount().GetId()

	if res, err := c.GetUser(ctx, &usersv1.GetUserRequest{Id: id}); err != nil || res.GetUser().GetPartnerStatus() != usersv1.PartnerStatus_PARTNER_STATUS_NONE {
		t.Errorf("GetUser: %v %v", res, err)
	}
	if _, err := c.GetUser(ctx, &usersv1.GetUserRequest{Id: "nope"}); status.Code(err) != codes.NotFound {
		t.Errorf("GetUser unknown: %v", err)
	}

	// Profile edits from the bot; a phone owned by someone else is refused.
	if res, err := c.UpdateProfile(ctx, &usersv1.UpdateProfileRequest{UserId: id, Phone: str("+996555000001")}); err != nil || res.GetAccount().GetPhone() != "+996555000001" {
		t.Fatalf("UpdateProfile: %v %v", res, err)
	}
	other, _ := c.UpsertTelegramUser(ctx, &usersv1.UpsertTelegramUserRequest{TelegramId: "43"})
	if _, err := c.UpdateProfile(ctx, &usersv1.UpdateProfileRequest{UserId: other.GetAccount().GetId(), Phone: str("+996555000001")}); status.Code(err) != codes.AlreadyExists {
		t.Errorf("duplicate phone: %v", err)
	}

	// Phone registration confirmed through the bot's deep link.
	pending, err := st.Insert(ctx, models.User{Phone: str("+996700000000"), IsVerified: false})
	if err != nil {
		t.Fatal(err)
	}
	token, _ := verification.Token()
	store.SetVerification(&pending, verification.Hash(token), verification.CodeTTL, time.Now())
	if _, err := st.Save(ctx, pending); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ConfirmTelegramVerification(ctx, &usersv1.ConfirmTelegramVerificationRequest{Token: "wrong"}); status.Code(err) != codes.NotFound {
		t.Errorf("bad token: %v", err)
	}
	if res, err := c.ConfirmTelegramVerification(ctx, &usersv1.ConfirmTelegramVerificationRequest{Token: token}); err != nil || res.GetAccount().GetId() != pending.ID {
		t.Fatalf("confirm: %v %v", res, err)
	}
	if _, err := c.ConfirmTelegramVerification(ctx, &usersv1.ConfirmTelegramVerificationRequest{Token: token}); status.Code(err) != codes.NotFound {
		t.Errorf("token reused: %v", err)
	}

	if res, err := c.GetSupportInfo(ctx, &usersv1.GetSupportInfoRequest{}); err != nil || res.GetSupport().GetMessage() != store.DefaultSupportMessage {
		t.Errorf("default support: %v %v", res, err)
	}
}

// Accounts keep their ids, password hashes, verification state and partner
// status; the import runs once.
func TestLegacyImport(t *testing.T) {
	db, _ := testdb.New(t, true)
	legacy, legacyDSN := testdb.New(t, false)
	ctx := context.Background()
	_, err := legacy.Exec(ctx, `
		CREATE TYPE users_partnerstatus_enum AS ENUM ('none','pending','approved','rejected');
		CREATE TABLE users (id uuid PRIMARY KEY, "telegramId" bigint, email varchar, "passwordHash" varchar,
			username varchar, "firstName" varchar, "lastName" varchar, phone varchar, "avatarUrl" varchar,
			"isVerified" boolean, "verificationCodeHash" varchar, "verificationCodeExpiresAt" timestamp,
			"verificationCodeSentAt" timestamp, "verificationAttempts" int,
			"partnerStatus" users_partnerstatus_enum, "partnerRequestedAt" timestamp,
			"createdAt" timestamp, "updatedAt" timestamp);
		CREATE TABLE support_info (id uuid PRIMARY KEY, message text, phone varchar, "telegramContact" varchar,
			email varchar, "workingHours" varchar, "updatedAt" timestamp);
		INSERT INTO users VALUES
			('11111111-1111-1111-1111-111111111111', 7000000001, NULL, NULL, 'tg', 'Tg', NULL, NULL, NULL, true,
			 NULL, NULL, NULL, 0, 'approved', '2026-05-01 10:00:00', '2026-05-01 09:00:00', '2026-05-01 09:00:00'),
			('22222222-2222-2222-2222-222222222222', NULL, 'a@x.kg', '$2a$10$abcdefghijklmnopqrstuuJ0nq1bnDqN1yVqZ8Zr8JkV9oQm5k5e', NULL,
			 NULL, NULL, NULL, NULL, false, 'deadbeef', '2026-05-01 10:10:00', '2026-05-01 10:00:00', 2, 'none',
			 NULL, '2026-05-01 10:00:00', '2026-05-01 10:00:00');
		INSERT INTO support_info VALUES ('33333333-3333-3333-3333-333333333333', 'Call us', '+996', NULL, NULL, '9-18', '2026-05-01 10:00:00');`)
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.DiscardHandler)
	for range 2 {
		if err := legacyimport.Run(ctx, db, legacyDSN, log); err != nil {
			t.Fatal(err)
		}
	}

	st := store.New(db)
	tg, err := st.ByTelegramID(ctx, "7000000001")
	if err != nil || tg.ID != "11111111-1111-1111-1111-111111111111" || tg.PartnerStatus != "approved" || tg.Admin().TelegramID == nil || *tg.Admin().TelegramID != "7000000001" {
		t.Errorf("telegram user: %+v %v", tg, err)
	}
	pending, err := st.ByIdentifier(ctx, "a@x.kg")
	if err != nil || pending.IsVerified || pending.VerificationAttempts != 2 || pending.PasswordHash == nil || pending.VerificationCodeHash == nil {
		t.Errorf("pending user: %+v %v", pending, err)
	}
	support, err := st.Support(ctx)
	if err != nil || support.Message != "Call us" || support.WorkingHours == nil {
		t.Errorf("support: %+v %v", support, err)
	}
}

// A database still at the v1.0.3 NestJS schema (production before the
// migration) has no partner or verification columns; their defaults apply.
func TestLegacyImportFromOldSchema(t *testing.T) {
	db, _ := testdb.New(t, true)
	legacy, legacyDSN := testdb.New(t, false)
	ctx := context.Background()
	_, err := legacy.Exec(ctx, `
		CREATE TABLE users (id uuid PRIMARY KEY, "telegramId" bigint, username varchar, "firstName" varchar,
			"lastName" varchar, phone varchar, "createdAt" timestamp, "updatedAt" timestamp, "avatarUrl" varchar,
			email varchar, "passwordHash" varchar);
		INSERT INTO users VALUES ('11111111-1111-1111-1111-111111111111', 7000000001, 'aibek', 'Айбек', NULL,
			'+996555', '2026-09-01 10:00', '2026-09-01 10:00', NULL, NULL, NULL);`)
	if err != nil {
		t.Fatal(err)
	}
	if err := legacyimport.Run(ctx, db, legacyDSN, slog.New(slog.DiscardHandler)); err != nil {
		t.Fatal(err)
	}
	u, err := store.New(db).ByTelegramID(ctx, "7000000001")
	if err != nil || !u.IsVerified || u.PartnerStatus != "none" || u.VerificationAttempts != 0 || u.VerificationCodeHash != nil {
		t.Fatalf("imported: %+v %v", u, err)
	}
}

func TestGetTelegramChats(t *testing.T) {
	db, _ := testdb.New(t, true)
	st := store.New(db)
	c := client(t, st)
	ctx := context.Background()

	tg, err := c.UpsertTelegramUser(ctx, &usersv1.UpsertTelegramUserRequest{TelegramId: "777000123"})
	if err != nil {
		t.Fatal(err)
	}
	web, err := st.Insert(ctx, models.User{Email: str("a@b.kg"), IsVerified: true})
	if err != nil {
		t.Fatal(err)
	}
	res, err := c.GetTelegramChats(ctx, &usersv1.GetTelegramChatsRequest{UserIds: []string{tg.GetAccount().GetId(), web.ID, "not-a-uuid"}})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.GetChatIds(); len(got) != 1 || got[tg.GetAccount().GetId()] != 777000123 {
		t.Errorf("chats: %v", got)
	}
	if res, err := c.GetTelegramChats(ctx, &usersv1.GetTelegramChatsRequest{}); err != nil || len(res.GetChatIds()) != 0 {
		t.Errorf("empty: %v %v", res, err)
	}
}

func TestLinkTelegram(t *testing.T) {
	db, _ := testdb.New(t, true)
	st := store.New(db)
	c := client(t, st)
	ctx := context.Background()

	issue := func(userID string, ttl time.Duration) string {
		t.Helper()
		token, _ := verification.Token()
		if err := st.SetTelegramLinkToken(ctx, userID, verification.Hash(token), time.Now().Add(ttl)); err != nil {
			t.Fatal(err)
		}
		return token
	}
	pass := "hash"
	site, _ := st.Insert(ctx, models.User{Email: str("site@x.kg"), PasswordHash: &pass, IsVerified: true})
	if _, err := st.Insert(ctx, models.User{Email: str("full@x.kg"), PasswordHash: &pass, IsVerified: true, TelegramID: str("500")}); err != nil {
		t.Fatal(err)
	}
	shell, _ := c.UpsertTelegramUser(ctx, &usersv1.UpsertTelegramUserRequest{TelegramId: "600", FirstName: str("Bot user")})

	if _, err := c.LinkTelegram(ctx, &usersv1.LinkTelegramRequest{Token: "nope", TelegramId: "700"}); status.Code(err) != codes.NotFound {
		t.Errorf("unknown token: %v", err)
	}
	expired := issue(site.ID, -time.Minute)
	if _, err := c.LinkTelegram(ctx, &usersv1.LinkTelegramRequest{Token: expired, TelegramId: "700"}); status.Code(err) != codes.NotFound {
		t.Errorf("expired token: %v", err)
	}

	// A Telegram id owned by an account with its own sign-in is refused.
	token := issue(site.ID, time.Minute)
	if _, err := c.LinkTelegram(ctx, &usersv1.LinkTelegramRequest{Token: token, TelegramId: "500"}); status.Code(err) != codes.AlreadyExists {
		t.Errorf("full account: %v", err)
	}
	// A Telegram-only account needs confirmation, then loses the id.
	if _, err := c.LinkTelegram(ctx, &usersv1.LinkTelegramRequest{Token: token, TelegramId: "600"}); status.Code(err) != codes.FailedPrecondition {
		t.Errorf("shell without confirmation: %v", err)
	}
	res, err := c.LinkTelegram(ctx, &usersv1.LinkTelegramRequest{Token: token, TelegramId: "600", Username: str("aibek"), MoveFromOtherAccount: true})
	if err != nil || res.GetAccount().GetId() != site.ID || res.GetAccount().GetUsername() != "aibek" {
		t.Fatalf("move: %v %v", res, err)
	}
	if u, _ := st.ByID(ctx, site.ID); u.TelegramID == nil || *u.TelegramID != "600" {
		t.Errorf("site account telegram: %v", u.TelegramID)
	}
	if u, _ := st.ByID(ctx, shell.GetAccount().GetId()); u.TelegramID != nil {
		t.Errorf("shell kept telegram: %v", *u.TelegramID)
	}
	// The token is single-use.
	if _, err := c.LinkTelegram(ctx, &usersv1.LinkTelegramRequest{Token: token, TelegramId: "600"}); status.Code(err) != codes.NotFound {
		t.Errorf("token reused: %v", err)
	}
	// The bot now finds the site account for that Telegram user.
	if again, err := c.UpsertTelegramUser(ctx, &usersv1.UpsertTelegramUserRequest{TelegramId: "600"}); err != nil || again.GetAccount().GetId() != site.ID {
		t.Errorf("upsert after link: %v %v", again, err)
	}

	// A brand-new Telegram id links directly.
	other, _ := st.Insert(ctx, models.User{Phone: str("+996700111222"), PasswordHash: &pass, IsVerified: true})
	if res, err := c.LinkTelegram(ctx, &usersv1.LinkTelegramRequest{Token: issue(other.ID, time.Minute), TelegramId: "800"}); err != nil || res.GetAccount().GetId() != other.ID {
		t.Errorf("fresh link: %v %v", res, err)
	}
}
