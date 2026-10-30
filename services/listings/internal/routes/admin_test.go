package routes_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/listings/internal/handlers"
	"github.com/elvn5/sai/services/listings/internal/routes"
	"github.com/elvn5/sai/services/listings/internal/store"
	"github.com/elvn5/sai/services/pkg/audit"
	"github.com/elvn5/sai/services/pkg/auth"
)

// asAdmin makes a request with an admin-panel session.
func (c client) asAdmin(method, path, body string) (int, string) {
	c.t.Helper()
	signer, err := auth.NewSigner(secret, "1h")
	if err != nil {
		c.t.Fatal(err)
	}
	tok, err := signer.AdminToken()
	if err != nil {
		c.t.Fatal(err)
	}
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.AddCookie(&http.Cookie{Name: auth.AdminCookie, Value: tok})
	rec := httptest.NewRecorder()
	c.router.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

// participantOf is the id of userID's row in a group.
func participantOf(t *testing.T, group map[string]any, userID string) string {
	t.Helper()
	for _, p := range group["participants"].([]any) {
		if p := p.(map[string]any); p["userId"] == userID {
			return p["id"].(string)
		}
	}
	t.Fatalf("no participant %s in %v", userID, group["participants"])
	return ""
}

func TestAdminTours(t *testing.T) {
	db, _ := createDatabase(t, true)
	log := slog.New(slog.DiscardHandler)
	clock := time.Date(2026, 10, 10, 6, 0, 0, 0, time.UTC)
	gin.SetMode(gin.TestMode)
	deps := handlers.Deps{Store: store.New(db), Authors: testUsers, Log: log,
		History: audit.New(db, routes.HistoryModules, log), Now: func() time.Time { return clock }}
	c := client{t, routes.Setup(deps, testUsers, secret)}

	_, body := c.do("POST", "/tours", tripTour, partner)
	id := obj(t, body)["id"].(string)
	_, body = c.do("POST", "/tours", strings.Replace(strings.Replace(tripTour, "ala-kul", "son-kul", 1), "2026-10-1", "2026-11-1", 2), partner)
	other := obj(t, body)["id"].(string)
	c.do("POST", "/tours/"+id+"/join", "", guest)
	c.do("POST", "/tours/"+id+"/join", "", owner)
	c.do("POST", "/tours/"+id+"/join", "", regular) // the waitlist
	c.do("POST", "/tours/"+other+"/join", "", guest)

	// Only the panel reads it.
	if status, _ := c.do("GET", "/tours/admin", "", partner); status != 401 {
		t.Errorf("as a partner: %d", status)
	}

	// The list: each tour's phase and group, with counts per phase.
	status, body := c.asAdmin("GET", "/tours/admin", "")
	list := obj(t, body)
	if status != 200 || !strings.Contains(body, `"counts":{"current":0,"past":0,"upcoming":2}`) || len(list["tours"].([]any)) != 2 {
		t.Fatalf("list: %d %s", status, body)
	}
	first := list["tours"].([]any)[1].(map[string]any) // latest departure first
	counts := first["counts"].(map[string]any)
	if first["phase"] != "upcoming" || first["tour"].(map[string]any)["organizer"] == nil ||
		counts["active"] != float64(2) || counts["waitlist"] != float64(1) || counts["due"] != float64(0) {
		t.Errorf("tour row: %v", first)
	}
	if _, body := c.asAdmin("GET", "/tours/admin?phase=upcoming&q=SON-", ""); !strings.Contains(body, other) || strings.Contains(body, id) {
		t.Errorf("search: %s", body)
	}
	if _, body := c.asAdmin("GET", "/tours/admin?phase=current", ""); !strings.Contains(body, `"tours":[]`) {
		t.Errorf("current: %s", body)
	}

	// One tour: its group and organizer.
	status, body = c.asAdmin("GET", "/tours/admin/"+id, "")
	group := obj(t, body)
	if status != 200 || group["phase"] != "upcoming" || group["organizer"].(map[string]any)["id"] != partner ||
		len(group["participants"].([]any)) != 3 || group["totals"].(map[string]any)["waitlist"] != float64(1) {
		t.Fatalf("tour: %d %s", status, body)
	}

	// Cancelling a place gives it to the waitlist; check-in; a participant
	// of another tour isn't this one's.
	if _, body := c.asAdmin("PATCH", "/tours/admin/"+id+"/participants/"+participantOf(t, group, guest), `{"status":"cancelled"}`); !strings.Contains(body, `"waitlist":0`) {
		t.Errorf("cancel: %s", body)
	}
	if _, body := c.asAdmin("PATCH", "/tours/admin/"+id+"/participants/"+participantOf(t, group, regular), `{"arrived":true}`); !strings.Contains(body, `"arrived":1`) {
		t.Errorf("check-in: %s", body)
	}
	_, body = c.asAdmin("GET", "/tours/admin/"+other, "")
	if status, _ := c.asAdmin("PATCH", "/tours/admin/"+id+"/participants/"+participantOf(t, obj(t, body), guest), `{"arrived":true}`); status != 404 {
		t.Errorf("another tour's participant: %d", status)
	}
	if status, _ := c.asAdmin("PATCH", "/tours/admin/"+id+"/participants/"+participantOf(t, group, owner), `{"status":"gone"}`); status != 400 {
		t.Errorf("bad status: %d", status)
	}

	// Hidden from the catalog, and back.
	if _, body := c.asAdmin("PUT", "/tours/admin/"+id+"/published", `{"published":false}`); !strings.Contains(body, `"isPublished":false`) {
		t.Errorf("hide: %s", body)
	}
	if status, _ := c.do("GET", "/tours/ru/ala-kul", "", ""); status != 404 {
		t.Errorf("hidden tour in the catalog: %d", status)
	}
	c.asAdmin("PUT", "/tours/admin/"+id+"/published", `{"published":true}`)
	if status, _ := c.do("GET", "/tours/ru/ala-kul", "", ""); status != 200 {
		t.Errorf("shown again: %d", status)
	}

	// On the way: the group met, the tour isn't over.
	clock = time.Date(2026, 10, 17, 5, 0, 0, 0, time.UTC)
	if _, body := c.asAdmin("GET", "/tours/admin?phase=current", ""); !strings.Contains(body, `"counts":{"current":1,"past":0,"upcoming":1}`) || !strings.Contains(body, id) {
		t.Errorf("on the way: %s", body)
	}

	// Every change is in the panel's history, named by the tour.
	_, body = c.asAdmin("GET", "/tours/admin/history", "")
	if strings.Count(body, `"module":"tours"`) != 4 || !strings.Contains(body, `"subject":"Ала-Куль <тест>"`) {
		t.Errorf("history: %s", body)
	}
	// A tour's history takes in its participants' changes.
	if _, body := c.asAdmin("GET", "/tours/admin/history?entityId="+other, ""); body != "[]" {
		t.Errorf("another tour's history: %s", body)
	}
	if _, body := c.asAdmin("GET", "/tours/admin/history?entityId="+id, ""); strings.Count(body, `"module":"tours"`) != 4 {
		t.Errorf("the tour's history: %s", body)
	}
}
