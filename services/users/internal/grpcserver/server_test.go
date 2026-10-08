package grpcserver_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	usersv1 "github.com/elvn5/sai/services/proto/gen/go/users/v1"
	"github.com/elvn5/sai/services/users/internal/grpcserver"
	"github.com/elvn5/sai/services/users/internal/store"
)

// newUsersDB creates a throwaway database with the NestJS-shaped users
// table. Skips without TEST_POSTGRES_DSN.
func newUsersDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	base := os.Getenv("TEST_POSTGRES_DSN")
	if base == "" {
		t.Skip("TEST_POSTGRES_DSN not set")
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	suffix := make([]byte, 4)
	_, _ = rand.Read(suffix)
	name := "users_test_" + hex.EncodeToString(suffix)
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	cfg, _ := pgxpool.ParseConfig(base)
	cfg.ConnConfig.Database = name
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
		_, _ = admin.Exec(ctx, "DROP DATABASE "+name+" WITH (FORCE)")
		_ = admin.Close(ctx)
	})
	_, err = db.Exec(ctx, `
		CREATE TYPE users_partnerstatus_enum AS ENUM ('none','pending','approved','rejected');
		CREATE TABLE users (id uuid PRIMARY KEY, "firstName" varchar, username varchar, "avatarUrl" varchar,
			email varchar, phone varchar, "partnerStatus" users_partnerstatus_enum NOT NULL DEFAULT 'none');
		INSERT INTO users VALUES
			('11111111-1111-1111-1111-111111111111','Aibek','aibek','https://t.me/a.jpg','a@x.kg','+996555','approved'),
			('22222222-2222-2222-2222-222222222222',NULL,NULL,NULL,'b@x.de',NULL,'none');`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestUsersService(t *testing.T) {
	db := newUsersDB(t)
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := grpc.NewServer()
	usersv1.RegisterUsersServiceServer(srv, &grpcserver.Server{Store: store.New(db), Log: slog.New(slog.DiscardHandler)})
	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	client := usersv1.NewUsersServiceClient(conn)
	ctx := context.Background()

	res, err := client.GetUser(ctx, &usersv1.GetUserRequest{Id: "11111111-1111-1111-1111-111111111111"})
	if err != nil {
		t.Fatal(err)
	}
	u := res.GetUser()
	if u.GetFirstName() != "Aibek" || u.GetUsername() != "aibek" || u.GetPartnerStatus() != usersv1.PartnerStatus_PARTNER_STATUS_APPROVED {
		t.Errorf("GetUser = %v", u)
	}

	if _, err := client.GetUser(ctx, &usersv1.GetUserRequest{Id: "33333333-3333-3333-3333-333333333333"}); status.Code(err) != codes.NotFound {
		t.Errorf("unknown user: %v", err)
	}
	if _, err := client.GetUser(ctx, &usersv1.GetUserRequest{Id: "not-a-uuid"}); status.Code(err) != codes.NotFound {
		t.Errorf("malformed id: %v", err)
	}

	batch, err := client.GetUsers(ctx, &usersv1.GetUsersRequest{Ids: []string{
		"11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222", "33333333-3333-3333-3333-333333333333",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.GetUsers()) != 2 {
		t.Fatalf("GetUsers returned %d users", len(batch.GetUsers()))
	}
	for _, u := range batch.GetUsers() {
		if u.GetId() == "22222222-2222-2222-2222-222222222222" && (u.FirstName != nil || u.GetPartnerStatus() != usersv1.PartnerStatus_PARTNER_STATUS_NONE) {
			t.Errorf("user without profile: %v", u)
		}
	}
}
