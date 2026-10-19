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
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/pkg/database"
	"github.com/elvn5/sai/services/pkg/middleware"
	"github.com/elvn5/sai/services/pkg/usersclient"
	"github.com/elvn5/sai/services/trails/internal/handlers"
	"github.com/elvn5/sai/services/trails/internal/migrations"
	"github.com/elvn5/sai/services/trails/internal/routes"
	"github.com/elvn5/sai/services/trails/internal/store"
)

const (
	secret  = "test-secret-at-least-16-chars"
	partner = "11111111-1111-1111-1111-111111111111"
	other   = "22222222-2222-2222-2222-222222222222"
	hiker   = "33333333-3333-3333-3333-333333333333"
	friend  = "44444444-4444-4444-4444-444444444444"
)

// fakeUsers stands in for the users service: id → partner status.
type fakeUsers map[string]string

func (f fakeUsers) LookupUser(_ context.Context, id string) (middleware.SessionUser, error) {
	status, ok := f[id]
	if !ok {
		return middleware.SessionUser{}, middleware.ErrUserNotFound
	}
	return middleware.SessionUser{ID: id, PartnerStatus: status}, nil
}

func (f fakeUsers) Authors(_ context.Context, ids []string) (map[string]usersclient.Author, error) {
	out := map[string]usersclient.Author{}
	for _, id := range ids {
		if _, ok := f[id]; ok {
			name := "User " + id[:4]
			out[id] = usersclient.Author{ID: id, FirstName: &name}
		}
	}
	return out, nil
}

var users = fakeUsers{partner: "approved", other: "approved", hiker: "none", friend: "none"}

// now is a fixed clock: 2026-10-11 12:00 in Bishkek.
var now = time.Date(2026, 10, 11, 6, 0, 0, 0, time.UTC)

type client struct {
	t      *testing.T
	router *gin.Engine
}

func setup(t *testing.T) client {
	t.Helper()
	gin.SetMode(gin.TestMode)
	return client{t, routes.Setup(newDeps(t), users, secret)}
}

// newDeps gives handler dependencies on a fresh, migrated database.
func newDeps(t *testing.T) handlers.Deps {
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
	name := "trails_test_" + hex.EncodeToString(suffix)
	db, err := database.Connect(context.Background(),
		database.DSN(cfg.Host, strconv.Itoa(int(cfg.Port)), cfg.User, cfg.Password, name),
		migrations.FS, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
		admin := cfg.Copy()
		admin.Database = "postgres"
		if conn, err := pgx.ConnectConfig(context.Background(), admin); err == nil {
			_, _ = conn.Exec(context.Background(), "DROP DATABASE "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)")
			_ = conn.Close(context.Background())
		}
	})
	return handlers.Deps{Store: store.New(db), Authors: users, Log: slog.New(slog.DiscardHandler),
		Now: func() time.Time { return now }}
}

// do sends a request as userID ("admin" for the admin panel, "" for
// anonymous).
func (c client) do(method, path, body, userID string) (int, string) {
	c.t.Helper()
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
	c.router.ServeHTTP(rec, req)
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

const komsomolets = `{"slug":"komsomolets","name":" Пик Комсомолец ","summary":"Классика Ала-Арчи","description":"<p onclick=\"x()\">Через хижину Рацека</p>",
	"area":"ala_archa","kind":"climb","grade":"1b","seasons":["summer","autumn"],"gear":["helmet","crampons","ice_axe"],
	"summitElevation":4150,"days":2,"segments":[{"from":"КПП","to":"Альплагерь","hours":1},{"from":"Альплагерь","to":"Хижина Рацека","hours":3.5}],
	"gettingThere":"Автобус №1 от Старой площади до КПП",
	"track":[[42.5641,74.4822,2140],[42.5500,74.4900,2600],[42.5300,74.5000,3370],[42.5200,74.5100,3360]]}`

func TestTrailCatalog(t *testing.T) {
	c := setup(t)
	if status, body := c.do("POST", "/trails", komsomolets, hiker); status != 403 || !strings.Contains(body, "Only approved partners") {
		t.Errorf("non-partner create: %d %s", status, body)
	}
	if status, body := c.do("POST", "/trails", `{"name":"x","area":"moon","track":[[95,0,null]]}`, partner); status != 400 ||
		!strings.Contains(body, `"slug":["Required"]`) || !strings.Contains(body, `"area":["Invalid enum value`) || !strings.Contains(body, `"track":["Invalid track point"]`) {
		t.Errorf("validation: %d %s", status, body)
	}

	status, body := c.do("POST", "/trails", komsomolets, partner)
	tr := obj(t, body)
	if status != 201 || tr["name"] != "Пик Комсомолец" || tr["userId"] != partner || tr["description"] != "<p>Через хижину Рацека</p>" {
		t.Fatalf("create: %d %s", status, body)
	}
	stats := tr["trackStats"].(map[string]any)
	if stats["elevationGain"] != float64(1230) || stats["maxElevation"] != float64(3370) || stats["distanceKm"].(float64) < 5 {
		t.Errorf("track stats: %v", stats)
	}
	if tr["startLatitude"] != 42.5641 {
		t.Errorf("start from track: %v", tr["startLatitude"])
	}
	id := tr["id"].(string)
	if status, _ := c.do("POST", "/trails", komsomolets, other); status != 409 {
		t.Errorf("slug taken: %d", status)
	}
	if status, _ := c.do("PUT", "/trails/"+id, `{"grade":"2a"}`, other); status != 403 {
		t.Errorf("other partner edit: %d", status)
	}
	if status, body := c.do("PUT", "/trails/"+id, `{"grade":"1a","summary":null}`, partner); status != 200 || obj(t, body)["grade"] != "1a" || obj(t, body)["summary"] != nil {
		t.Errorf("owner edit: %d %s", status, body)
	}
	_, body = c.do("POST", "/trails", `{"slug":"draft","name":"Draft","area":"karakol","kind":"hike","grade":"trail","isPublished":false}`, "admin")
	draftID := obj(t, body)["id"].(string)

	if _, body := c.do("GET", "/trails?area=ala_archa", "", ""); strings.Count(body, `"slug"`) != 1 || strings.Contains(body, `"track"`) {
		t.Errorf("list (no tracks): %s", body)
	}
	if _, body := c.do("GET", "/trails?grade=1a,1b&search=комсо", "", ""); !strings.Contains(body, "komsomolets") {
		t.Errorf("grade+search: %s", body)
	}
	if _, body := c.do("GET", "/trails?season=winter", "", ""); body != "[]" {
		t.Errorf("season filter: %s", body)
	}
	if _, body := c.do("GET", "/trails/areas", "", ""); body != `{"ala_archa":1}` {
		t.Errorf("areas: %s", body)
	}
	if status, _ := c.do("GET", "/trails/draft", "", ""); status != 404 {
		t.Errorf("draft public: %d", status)
	}
	if status, _ := c.do("GET", "/trails/draft", "", "admin"); status != 200 {
		t.Errorf("draft to admin: %d", status)
	}
	if status, body := c.do("GET", "/trails/komsomolets", "", ""); status != 200 || !strings.Contains(body, `"track":[[42.5641,74.4822,2140]`) {
		t.Errorf("detail: %d %s", status, body)
	}
	if _, body := c.do("GET", "/trails/mine", "", partner); strings.Count(body, `"slug"`) != 1 {
		t.Errorf("mine: %s", body)
	}
	if status, _ := c.do("DELETE", "/trails/"+draftID, "", partner); status != 403 {
		t.Errorf("partner deleting admin route: %d", status)
	}
	if status, _ := c.do("DELETE", "/trails/"+draftID, "", "admin"); status != 200 {
		t.Errorf("admin delete: %d", status)
	}
}

func TestConditionReports(t *testing.T) {
	c := setup(t)
	_, body := c.do("POST", "/trails", komsomolets, partner)
	id := obj(t, body)["id"].(string)

	report := `{"observedOn":"2026-10-10","trailState":"patchy_snow","traction":"microspikes","snowLine":3400,"reached":"Хижина Рацека","text":"Выше хижины лёд"}`
	if status, _ := c.do("POST", "/trails/"+id+"/conditions", report, ""); status != 401 {
		t.Errorf("anonymous: %d", status)
	}
	for date, want := range map[string]int{"2026-10-12": 400, "2026-09-01": 400} {
		if status, body := c.do("POST", "/trails/"+id+"/conditions", strings.Replace(report, "2026-10-10", date, 1), hiker); status != want {
			t.Errorf("date %s: %d %s", date, status, body)
		}
	}
	status, body := c.do("POST", "/trails/"+id+"/conditions", report, hiker)
	if status != 201 || !strings.Contains(body, `"firstName":"User 3333"`) || !strings.Contains(body, `"slug":"komsomolets"`) {
		t.Fatalf("create: %d %s", status, body)
	}
	reportID := obj(t, body)["id"].(string)
	c.do("POST", "/trails/komsomolets/conditions", `{"observedOn":"2026-10-03","trailState":"clear","traction":"none"}`, friend)

	if _, body := c.do("GET", "/trails/komsomolets/conditions", "", ""); strings.Index(body, "2026-10-10") > strings.Index(body, "2026-10-03") {
		t.Errorf("order: %s", body)
	}
	if _, body := c.do("GET", "/trails?area=ala_archa", "", ""); !strings.Contains(body, `"latestCondition":{"observedOn":"2026-10-10","trailState":"patchy_snow","traction":"microspikes"}`) {
		t.Errorf("summary on card: %s", body)
	}
	if _, body := c.do("GET", "/conditions?limit=1", "", ""); strings.Count(body, `"observedOn"`) != 1 {
		t.Errorf("recent: %s", body)
	}
	if status, _ := c.do("DELETE", "/conditions/"+reportID, "", friend); status != 403 {
		t.Errorf("other user delete: %d", status)
	}
	if status, _ := c.do("DELETE", "/conditions/"+reportID, "", hiker); status != 200 {
		t.Errorf("author delete: %d", status)
	}
}

func TestOutings(t *testing.T) {
	c := setup(t)
	_, body := c.do("POST", "/trails", komsomolets, partner)
	trailID := obj(t, body)["id"].(string)

	outing := `{"trailId":"` + trailID + `","title":"Комсомолец за выходные","startsAt":"2026-10-17T06:00:00+06:00","returnsAt":"2026-10-18T20:00:00+06:00",
		"meetingPoint":"Южные ворота","transport":"car","seats":1,"costPerPerson":400,"level":"intermediate","contact":"@hiker_kg"}`
	if status, body := c.do("POST", "/outings", `{"title":"x","startsAt":"2026-10-01T06:00:00+06:00"}`, hiker); status != 400 ||
		!strings.Contains(body, `"meetingPoint":["Required"]`) {
		t.Errorf("validation: %d %s", status, body)
	}
	if status, body := c.do("POST", "/outings", strings.Replace(outing, "2026-10-17", "2026-10-01", 1), hiker); status != 400 || !strings.Contains(body, "startsAt") {
		t.Errorf("past start: %d %s", status, body)
	}
	status, body := c.do("POST", "/outings", outing, hiker)
	o := obj(t, body)
	if status != 201 || o["isOrganizer"] != true || o["contact"] != "@hiker_kg" || o["trail"].(map[string]any)["slug"] != "komsomolets" {
		t.Fatalf("create: %d %s", status, body)
	}
	id := o["id"].(string)

	if _, body := c.do("GET", "/outings", "", ""); !strings.Contains(body, id) || strings.Contains(body, "@hiker_kg") {
		t.Errorf("public list leaks contact or misses outing: %s", body)
	}
	if status, _ := c.do("POST", "/outings/"+id+"/join", "", hiker); status != 400 {
		t.Errorf("organizer joining: %d", status)
	}
	status, body = c.do("POST", "/outings/"+id+"/join", "", friend)
	if status != 200 || !strings.Contains(body, `"isJoined":true`) || !strings.Contains(body, "@hiker_kg") || !strings.Contains(body, `"taken":1`) {
		t.Fatalf("join: %d %s", status, body)
	}
	if status, body := c.do("POST", "/outings/"+id+"/join", "", other); status != 409 || !strings.Contains(body, "No seats left") {
		t.Errorf("full: %d %s", status, body)
	}
	if status, body := c.do("GET", "/outings/"+id, "", other); status != 200 || strings.Contains(body, "@hiker_kg") || !strings.Contains(body, `"participants":[{"id":"`+friend) {
		t.Errorf("detail for outsider: %d %s", status, body)
	}
	if _, body := c.do("GET", "/outings?trail=komsomolets", "", ""); !strings.Contains(body, id) {
		t.Errorf("by trail: %s", body)
	}
	if _, body := c.do("GET", "/trails/komsomolets", "", ""); !strings.Contains(body, `"upcomingOutings":1`) {
		t.Errorf("trail count: %s", body)
	}
	if status, body := c.do("PUT", "/outings/"+id, `{"seats":0}`, hiker); status != 400 {
		t.Errorf("seats below taken: %d %s", status, body)
	}
	if status, _ := c.do("PUT", "/outings/"+id, `{"seats":3}`, friend); status != 403 {
		t.Errorf("participant edit: %d", status)
	}
	if status, body := c.do("PUT", "/outings/"+id, `{"seats":3,"returnsAt":null}`, hiker); status != 200 || obj(t, body)["returnsAt"] != nil {
		t.Errorf("organizer edit: %d %s", status, body)
	}
	if _, body := c.do("GET", "/outings/mine", "", friend); !strings.Contains(body, id) {
		t.Errorf("mine (joined): %s", body)
	}
	if status, body := c.do("DELETE", "/outings/"+id+"/join", "", friend); status != 200 || !strings.Contains(body, `"taken":0`) {
		t.Errorf("leave: %d %s", status, body)
	}
	if status, body := c.do("POST", "/outings/"+id+"/cancel", "", hiker); status != 200 || obj(t, body)["status"] != "cancelled" {
		t.Errorf("cancel: %d %s", status, body)
	}
	if status, _ := c.do("POST", "/outings/"+id+"/join", "", friend); status != 409 {
		t.Errorf("join cancelled: %d", status)
	}
	if _, body := c.do("GET", "/outings", "", ""); body != "[]" {
		t.Errorf("cancelled still listed: %s", body)
	}
}

func TestOutingsSummary(t *testing.T) {
	deps := newDeps(t)
	c := client{t, routes.Setup(deps, users, secret)}
	_, body := c.do("POST", "/trails", komsomolets, partner)
	trailID := obj(t, body)["id"].(string)
	_, body = c.do("POST", "/outings", `{"trailId":"`+trailID+`","title":"Комсомолец","startsAt":"2026-10-17T06:00:00+06:00",
		"meetingPoint":"Южные ворота","transport":"car","seats":3,"level":"intermediate"}`, hiker)
	id := obj(t, body)["id"].(string)
	c.do("POST", "/outings/"+id+"/join", "", friend)

	if status, body := c.do("GET", "/outings/mine/summary", "", friend); status != 200 || !strings.Contains(body, `"outings":0`) {
		t.Errorf("before the start nothing counts: %d %s", status, body)
	}
	if status, _ := c.do("GET", "/outings/mine/summary", "", ""); status != 401 {
		t.Errorf("anonymous: %d", status)
	}

	// A week later the outing is behind them.
	later := deps
	later.Now = func() time.Time { return now.Add(7 * 24 * time.Hour) }
	c = client{t, routes.Setup(later, users, secret)}
	for _, who := range []string{friend, hiker} {
		_, body := c.do("GET", "/outings/mine/summary", "", who)
		sum := obj(t, body)
		if sum["outings"] != float64(1) || sum["distanceKm"].(float64) <= 0 || sum["elevationGain"].(float64) <= 0 {
			t.Errorf("summary for %s: %s", who, body)
		}
	}
	if _, body := c.do("GET", "/outings/mine/summary", "", other); !strings.Contains(body, `"outings":0`) {
		t.Errorf("outsider: %s", body)
	}
}
