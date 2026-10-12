package grpcserver_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/elvn5/sai/services/listings/internal/grpcserver"
	"github.com/elvn5/sai/services/listings/internal/migrations"
	"github.com/elvn5/sai/services/listings/internal/store"
	"github.com/elvn5/sai/services/pkg/database"
	listingsv1 "github.com/elvn5/sai/services/proto/gen/go/listings/v1"
)

func TestGetTour(t *testing.T) {
	base := os.Getenv("TEST_POSTGRES_DSN")
	if base == "" {
		t.Skip("TEST_POSTGRES_DSN not set")
	}
	cfg, err := pgx.ParseConfig(base)
	if err != nil {
		t.Fatal(err)
	}
	suffix := make([]byte, 4)
	_, _ = rand.Read(suffix)
	name := "listings_grpc_test_" + hex.EncodeToString(suffix)
	ctx := context.Background()
	db, err := database.Connect(ctx, database.DSN(cfg.Host, strconv.Itoa(int(cfg.Port)), cfg.User, cfg.Password, name),
		migrations.FS, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
		admin := cfg.Copy()
		admin.Database = "postgres"
		if conn, err := pgx.ConnectConfig(ctx, admin); err == nil {
			_, _ = conn.Exec(ctx, "DROP DATABASE "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)")
			_ = conn.Close(ctx)
		}
	})
	const id = "aaaaaaaa-0000-0000-0000-000000000001"
	if _, err := db.Exec(ctx, `INSERT INTO tours (id, locale, slug, title, content, price, start_date, end_date, is_published, user_id)
		VALUES ($1, 'ru', 'ala-kul', 'Ала-Куль', 'x', 12000, '2027-06-10', '2027-06-12', true, '11111111-1111-1111-1111-111111111111')`, id); err != nil {
		t.Fatal(err)
	}

	srv := &grpcserver.Server{Store: store.New(db), Log: slog.New(slog.DiscardHandler)}
	res, err := srv.GetTour(ctx, &listingsv1.GetTourRequest{Id: id})
	if err != nil {
		t.Fatal(err)
	}
	if tour := res.GetTour(); tour.GetPrice() != 12000 || tour.GetStartDate() != "2027-06-10" || tour.GetUserId() != "11111111-1111-1111-1111-111111111111" || !tour.GetIsPublished() {
		t.Errorf("tour: %v", tour)
	}
	for _, bad := range []string{"nope", "bbbbbbbb-0000-0000-0000-000000000001"} {
		if _, err := srv.GetTour(ctx, &listingsv1.GetTourRequest{Id: bad}); status.Code(err) != codes.NotFound {
			t.Errorf("%s: %v", bad, err)
		}
	}
}
