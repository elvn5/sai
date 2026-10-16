package routes_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/payments/internal/handlers"
	"github.com/elvn5/sai/services/payments/internal/listingsclient"
	"github.com/elvn5/sai/services/payments/internal/migrations"
	"github.com/elvn5/sai/services/payments/internal/provider"
	"github.com/elvn5/sai/services/payments/internal/routes"
	"github.com/elvn5/sai/services/payments/internal/store"
	"github.com/elvn5/sai/services/payments/internal/worker"
	"github.com/elvn5/sai/services/pkg/database"
	"github.com/elvn5/sai/services/pkg/middleware"
	"github.com/elvn5/sai/services/pkg/telegram"
	"github.com/elvn5/sai/services/pkg/usersclient"
)

const (
	secret   = "test-secret-at-least-16-chars"
	partner  = "11111111-1111-1111-1111-111111111111"
	tourist  = "22222222-2222-2222-2222-222222222222"
	tourID   = "aaaaaaaa-0000-0000-0000-000000000001"
	freeTour = "aaaaaaaa-0000-0000-0000-000000000002"
	oldTour  = "aaaaaaaa-0000-0000-0000-000000000003"
)

var now = time.Date(2026, 10, 12, 6, 0, 0, 0, time.UTC)

type fakeUsers map[string]string

func (f fakeUsers) LookupUser(_ context.Context, id string) (middleware.SessionUser, error) {
	s, ok := f[id]
	if !ok {
		return middleware.SessionUser{}, middleware.ErrUserNotFound
	}
	return middleware.SessionUser{ID: id, PartnerStatus: s}, nil
}

func (f fakeUsers) Authors(_ context.Context, ids []string) (map[string]usersclient.Author, error) {
	out := map[string]usersclient.Author{}
	for _, id := range ids {
		name := "User " + id[:4]
		out[id] = usersclient.Author{ID: id, FirstName: &name}
	}
	return out, nil
}

func (f fakeUsers) TelegramChats(_ context.Context, ids []string) (map[string]int64, error) {
	return map[string]int64{partner: 1, tourist: 2}, nil
}

var users = fakeUsers{partner: "approved", tourist: "none"}

type fakeTours map[string]listingsclient.Tour

func (f fakeTours) Tour(_ context.Context, id string) (listingsclient.Tour, error) {
	t, ok := f[id]
	if !ok {
		return t, listingsclient.ErrNotFound
	}
	return t, nil
}

func intp(n int) *int       { return &n }
func strp(s string) *string { return &s }

var tours = fakeTours{
	tourID:   {ID: tourID, Slug: "ala-kul", Locale: "ru", Title: "Ала-Куль", Price: intp(12000), StartDate: "2026-10-20", UserID: strp(partner), IsPublished: true},
	freeTour: {ID: freeTour, Slug: "free", Locale: "ru", Title: "Free", StartDate: "2026-10-20", UserID: strp(partner), IsPublished: true},
	oldTour:  {ID: oldTour, Slug: "old", Locale: "ru", Title: "Old", Price: intp(5000), StartDate: "2026-10-01", UserID: strp(partner), IsPublished: true},
}

type sent struct {
	chat int64
	text string
}

type fakeTelegram struct {
	mu   sync.Mutex
	msgs []sent
}

func (f *fakeTelegram) SendMessage(_ context.Context, chat int64, text string, _ *telegram.Button) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.msgs = append(f.msgs, sent{chat, text})
	return nil
}

type env struct {
	t      *testing.T
	router *gin.Engine
	fake   *provider.Fake
	worker *worker.Worker
	tg     *fakeTelegram
	clock  *time.Time
}

func setup(t *testing.T, gateway provider.Provider) *env {
	t.Helper()
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
	name := "payments_test_" + hex.EncodeToString(suffix)
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
	clock := now
	e := &env{t: t, tg: &fakeTelegram{}, clock: &clock}
	fake, _ := gateway.(*provider.Fake)
	e.fake = fake
	st := store.New(db)
	nowFn := func() time.Time { return *e.clock }
	e.worker = &worker.Worker{Store: st, Provider: gateway, Sender: e.tg, Chats: users, SiteURL: "https://ermekten.kg",
		Log: slog.New(slog.DiscardHandler), Now: nowFn}
	gin.SetMode(gin.TestMode)
	e.router = routes.Setup(handlers.Deps{Store: st, Provider: gateway, Tours: tours, Authors: users, Checker: e.worker,
		FeePercent: 10, PaymentTTL: 30 * time.Minute, SiteURL: "https://ermekten.kg", Log: slog.New(slog.DiscardHandler),
		Now: nowFn}, users, secret, fake)
	return e
}

func (e *env) do(method, path, body, userID string) (int, string) {
	e.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	switch userID {
	case "":
	case "admin":
		tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"role": "admin"}).SignedString([]byte(secret))
		req.AddCookie(&http.Cookie{Name: "admin_token", Value: tok})
	default:
		tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": userID, "role": "user"}).SignedString([]byte(secret))
		req.AddCookie(&http.Cookie{Name: "user_token", Value: tok})
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

func obj(t *testing.T, body string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("not an object: %s", body)
	}
	return m
}

func order(item string, qty int) string {
	return `{"kind":"tour","itemId":"` + item + `","quantity":` + strconv.Itoa(qty) + `,"contactName":"Айгерим","contactPhone":"+996 555 123 456","comment":"Вегетарианка"}`
}

func TestTourBookingAndPayment(t *testing.T) {
	e := setup(t, &provider.Fake{SiteURL: "https://ermekten.kg"})

	if status, _ := e.do("POST", "/orders", order(tourID, 2), ""); status != 401 {
		t.Errorf("anonymous: %d", status)
	}
	if status, body := e.do("POST", "/orders", `{"kind":"tour","itemId":"x","quantity":0}`, tourist); status != 400 ||
		!strings.Contains(body, `"contactPhone":["Required"]`) {
		t.Errorf("validation: %d %s", status, body)
	}
	for item, want := range map[string]int{freeTour: 409, oldTour: 409, "bbbbbbbb-0000-0000-0000-000000000009": 404} {
		if status, body := e.do("POST", "/orders", order(item, 1), tourist); status != want {
			t.Errorf("%s: %d %s", item, status, body)
		}
	}
	if status, _ := e.do("POST", "/orders", order(tourID, 1), partner); status != 400 {
		t.Errorf("own tour: %d", status)
	}

	// Priced from the tour, 10% commission; the client can't set amounts.
	status, body := e.do("POST", "/orders", strings.Replace(order(tourID, 2), `"kind"`, `"amount":1,"kind"`, 1), tourist)
	o := obj(t, body)
	if status != 201 || o["amount"] != float64(24000) || o["fee"] != float64(2400) || o["payeeAmount"] != float64(21600) ||
		o["status"] != "pending" || !strings.HasPrefix(o["paymentUrl"].(string), "https://ermekten.kg/payments/test?order=") {
		t.Fatalf("create: %d %s", status, body)
	}
	id := o["id"].(string)

	// The partner doesn't see unpaid orders; others can't see it at all.
	if status, _ := e.do("GET", "/orders/"+id, "", partner); status != 404 {
		t.Errorf("partner sees unpaid order: %d", status)
	}
	if _, body := e.do("GET", "/orders/received", "", partner); body != "[]" {
		t.Errorf("received before payment: %s", body)
	}

	// Paying on the test page, then checking on return.
	if status, _ := e.do("POST", "/payments/test/"+id, `{"success":true}`, partner); status != 404 {
		t.Errorf("someone else paying: %d", status)
	}
	if status, body := e.do("POST", "/payments/test/"+id, `{"success":true}`, tourist); status != 200 || !strings.Contains(body, "/payments/return?order="+id) {
		t.Fatalf("test pay: %d %s", status, body)
	}
	if status, body := e.do("POST", "/orders/"+id+"/check", "", tourist); status != 200 || obj(t, body)["status"] != "paid" || obj(t, body)["paidAt"] == nil {
		t.Fatalf("check: %d %s", status, body)
	}

	// Messages go out once, to the partner and the customer.
	e.worker.Tick(context.Background())
	e.worker.Tick(context.Background())
	if len(e.tg.msgs) != 2 || !strings.Contains(e.tg.msgs[0].text, "Новая оплаченная бронь") || !strings.Contains(e.tg.msgs[0].text, "24 000 сом (к выплате вам 21 600 сом)") ||
		!strings.Contains(e.tg.msgs[0].text, "+996 555 123 456") || !strings.Contains(e.tg.msgs[1].text, "Оплата прошла") {
		t.Errorf("messages: %+v", e.tg.msgs)
	}

	if _, body := e.do("GET", "/orders/received", "", partner); !strings.Contains(body, id) || !strings.Contains(body, `"payer":{"id":"`+tourist) {
		t.Errorf("received: %s", body)
	}
	if _, body := e.do("GET", "/orders/mine", "", tourist); !strings.Contains(body, id) {
		t.Errorf("mine: %s", body)
	}

	// Payout: the balance, then nothing left; refunds only before payout.
	if _, body := e.do("GET", "/payouts/balances", "", "admin"); !strings.Contains(body, `"amount":21600,"orders":1`) {
		t.Errorf("balances: %s", body)
	}
	if status, _ := e.do("GET", "/payouts/balances", "", tourist); status != 401 {
		t.Errorf("balances as user: %d", status)
	}
	if status, body := e.do("POST", "/payouts", `{"payeeId":"`+partner+`","note":"Перевод на карту"}`, "admin"); status != 201 || obj(t, body)["amount"] != float64(21600) {
		t.Fatalf("payout: %d %s", status, body)
	}
	if status, _ := e.do("POST", "/payouts", `{"payeeId":"`+partner+`"}`, "admin"); status != 409 {
		t.Errorf("empty payout: %d", status)
	}
	if status, _ := e.do("POST", "/orders/"+id+"/refund", `{}`, "admin"); status != 409 {
		t.Errorf("refund after payout: %d", status)
	}
}

func TestDeclinedExpiredAndRefunded(t *testing.T) {
	e := setup(t, &provider.Fake{SiteURL: "https://ermekten.kg"})
	ctx := context.Background()
	create := func() string {
		_, body := e.do("POST", "/orders", order(tourID, 1), tourist)
		return obj(t, body)["id"].(string)
	}

	declined := create()
	e.do("POST", "/payments/test/"+declined, `{"success":false}`, tourist)
	expired := create()
	paid := create()
	e.do("POST", "/payments/test/"+paid, `{"success":true}`, tourist)

	*e.clock = now.Add(45 * time.Minute) // past the link's life and grace
	e.worker.Tick(ctx)
	for id, want := range map[string]string{declined: "failed", expired: "expired", paid: "paid"} {
		if _, body := e.do("GET", "/orders/"+id, "", tourist); obj(t, body)["status"] != want {
			t.Errorf("%s: %s", want, body)
		}
	}
	if status, body := e.do("POST", "/orders/"+paid+"/refund", `{"note":"Тур отменён"}`, "admin"); status != 200 || obj(t, body)["status"] != "refunded" {
		t.Errorf("refund: %d %s", status, body)
	}
	if _, body := e.do("GET", "/payouts/balances", "", "admin"); body != "[]" {
		t.Errorf("refunded still owed: %s", body)
	}
	if _, body := e.do("GET", "/orders/admin?status=expired", "", "admin"); strings.Count(body, `"kind"`) != 1 {
		t.Errorf("admin filter: %s", body)
	}
}

// stubGateway reports a fixed result.
type stubGateway struct {
	res     provider.Result
	linkErr error
}

func (s *stubGateway) Name() string { return "stub" }
func (s *stubGateway) CreatePayment(context.Context, provider.Payment) (string, error) {
	return "https://pay.example/x", s.linkErr
}
func (s *stubGateway) Status(context.Context, string) (provider.Result, error) { return s.res, nil }

func TestGatewayEdgeCases(t *testing.T) {
	gw := &stubGateway{res: provider.Result{State: provider.StatePaid, Amount: 100}}
	e := setup(t, gw)

	// The bank reporting a different amount doesn't count as paid.
	_, body := e.do("POST", "/orders", order(tourID, 1), tourist)
	id := obj(t, body)["id"].(string)
	if _, body := e.do("POST", "/orders/"+id+"/check", "", tourist); obj(t, body)["status"] != "failed" {
		t.Errorf("amount mismatch: %s", body)
	}

	// The bank being down when creating the link fails the order.
	gw.linkErr = context.DeadlineExceeded
	if status, body := e.do("POST", "/orders", order(tourID, 1), tourist); status != 502 {
		t.Errorf("gateway down: %d %s", status, body)
	}
	if _, body := e.do("GET", "/orders/mine", "", tourist); strings.Count(body, `"status":"failed"`) != 2 {
		t.Errorf("failed orders: %s", body)
	}
}

func TestOnlinePaymentOff(t *testing.T) {
	e := setup(t, provider.Disabled{})

	if status, body := e.do("GET", "/payments/config", "", ""); status != 200 || body != `{"online":false}` {
		t.Errorf("config: %d %s", status, body)
	}
	if status, body := e.do("POST", "/orders", order(tourID, 1), tourist); status != 503 ||
		!strings.Contains(body, "Online payment is not available") {
		t.Errorf("order: %d %s", status, body)
	}
	if status, body := e.do("GET", "/orders/mine", "", tourist); status != 200 || body != "[]" {
		t.Errorf("no order was created: %d %s", status, body)
	}

	on := setup(t, &provider.Fake{SiteURL: "https://ermekten.kg"})
	if status, body := on.do("GET", "/payments/config", "", ""); status != 200 || body != `{"online":true}` {
		t.Errorf("config with a gateway: %d %s", status, body)
	}
}
