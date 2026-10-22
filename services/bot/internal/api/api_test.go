package api_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/pkg/auth"
)

const secret = "test-secret-at-least-16-chars"

type request struct {
	method, uri, body, userID string
}

func newClient(t *testing.T, status int, answer string) (*api.Client, *[]request) {
	t.Helper()
	var got []request
	verifier := auth.NewVerifier(secret)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		req := request{method: r.Method, uri: r.URL.RequestURI(), body: string(body)}
		if c, err := r.Cookie(auth.UserCookie); err == nil {
			req.userID, _ = verifier.UserID(c.Value)
		}
		got = append(got, req)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, answer)
	}))
	t.Cleanup(srv.Close)
	c, err := api.New(srv.URL+"/", secret)
	if err != nil {
		t.Fatal(err)
	}
	c.Now = func() time.Time { return time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC) }
	return c, &got
}

func TestCallsAsTheUser(t *testing.T) {
	c, got := newClient(t, http.StatusOK,
		`[{"id":"o1","title":"Пик","startsAt":"2026-10-24T00:00:00.000Z","seats":10,"taken":7,"status":"open","organizer":{"id":"u2","firstName":"Айгуль"}}]`)
	list, err := c.MyOutings(context.Background(), "u1")
	if err != nil || len(list) != 1 || list[0].Organizer.Name() != "Айгуль" || list[0].StartsAt.Day() != 24 {
		t.Fatalf("MyOutings = %+v, %v", list, err)
	}
	if r := (*got)[0]; r.method != http.MethodGet || r.uri != "/outings/mine" || r.userID != "u1" {
		t.Errorf("request: %+v", r)
	}
}

func TestWrites(t *testing.T) {
	c, got := newClient(t, http.StatusOK, `[]`)
	ctx := context.Background()
	for _, call := range []func() error{
		func() error { return c.JoinOuting(ctx, "u1", "o1") },
		func() error { return c.LeaveOuting(ctx, "u1", "o1") },
		func() error { return c.SetFavoriteTour(ctx, "u1", "t1", true) },
		func() error { return c.SetFavoriteTour(ctx, "u1", "t1", false) },
		func() error { return c.SetAllToursAlert(ctx, "u1", true) },
		func() error { _, err := c.UpcomingTours(ctx, "ru"); return err },
	} {
		if err := call(); err != nil {
			t.Fatal(err)
		}
	}
	want := []request{
		{http.MethodPost, "/outings/o1/join", "", "u1"},
		{http.MethodDelete, "/outings/o1/join", "", "u1"},
		{http.MethodPut, "/favorites/tour/t1", "", "u1"},
		{http.MethodDelete, "/favorites/tour/t1", "", "u1"},
		{http.MethodPut, "/tour-alerts", `{"destination":""}`, "u1"},
		{http.MethodGet, "/tours?from=2026-10-20&locale=ru", "", ""},
	}
	for i, w := range want {
		if (*got)[i] != w {
			t.Errorf("request %d: %+v, want %+v", i, (*got)[i], w)
		}
	}
}

func TestErrors(t *testing.T) {
	c, _ := newClient(t, http.StatusConflict, `{"statusCode":409,"message":"No seats left","error":"Conflict"}`)
	err := c.JoinOuting(context.Background(), "u1", "o1")
	if api.StatusOf(err) != http.StatusConflict || api.MessageOf(err) != "No seats left" {
		t.Errorf("error: %v", err)
	}
}

// /outings/mine says only who organizes: the rest are joined.
func TestMyOutingsJoined(t *testing.T) {
	c, _ := newClient(t, http.StatusOK, `[{"id":"a","isOrganizer":true},{"id":"b","isOrganizer":false}]`)
	list, err := c.MyOutings(context.Background(), "u1")
	if err != nil || list[0].IsJoined || !list[1].IsJoined {
		t.Errorf("MyOutings = %+v, %v", list, err)
	}
}

func TestReads(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		answer, uri, userID string
		call                func(c *api.Client) (any, error)
		check               func(v any) bool
	}{
		{`{"outings":3,"distanceKm":12.5,"elevationGain":900}`, "/outings/mine/summary", "u1",
			func(c *api.Client) (any, error) { return c.OutingsSummary(ctx, "u1") },
			func(v any) bool {
				return v.(api.OutingsSummary) == api.OutingsSummary{Outings: 3, DistanceKm: 12.5, ElevationGain: 900}
			}},
		{`[{"id":"o1"}]`, "/outings?limit=50", "u1",
			func(c *api.Client) (any, error) { return c.UpcomingOutings(ctx, "u1") },
			func(v any) bool { return len(v.([]api.Outing)) == 1 }},
		{`{"id":"o 1","isJoined":true,"contact":"@a"}`, "/outings/o%201", "u1",
			func(c *api.Client) (any, error) { return c.Outing(ctx, "u1", "o 1") },
			func(v any) bool { o := v.(api.Outing); return o.IsJoined && *o.Contact == "@a" }},
		{`{"tours":[{"id":"t1"}],"hotels":[],"localGuides":[]}`, "/favorites/mine", "u1",
			func(c *api.Client) (any, error) { return c.FavoriteTours(ctx, "u1") },
			func(v any) bool { return len(v.([]api.Tour)) == 1 }},
		{`{"destinations":["","karakol"]}`, "/tour-alerts/mine", "u1",
			func(c *api.Client) (any, error) { return c.TourAlerts(ctx, "u1") },
			func(v any) bool { d := v.([]string); return len(d) == 2 && d[0] == "" }},
		{`[{"id":"p1","publishedAt":"2026-10-18T03:00:00.000Z"}]`, "/blog?locale=ru", "",
			func(c *api.Client) (any, error) { return c.BlogPosts(ctx, "ru") },
			func(v any) bool { p := v.([]api.BlogPost); return p[0].PublishedAt.Hour() == 3 }},
	} {
		c, got := newClient(t, http.StatusOK, tc.answer)
		v, err := tc.call(c)
		if err != nil || !tc.check(v) {
			t.Errorf("%s: %+v, %v", tc.uri, v, err)
		}
		if r := (*got)[0]; r.method != http.MethodGet || r.uri != tc.uri || r.userID != tc.userID {
			t.Errorf("request: %+v, want GET %s as %q", r, tc.uri, tc.userID)
		}
	}
}

func TestDeleteAlert(t *testing.T) {
	c, got := newClient(t, http.StatusOK, `{}`)
	if err := c.SetAllToursAlert(context.Background(), "u1", false); err != nil {
		t.Fatal(err)
	}
	if r := (*got)[0]; r.method != http.MethodDelete || r.body != `{"destination":""}` {
		t.Errorf("request: %+v", r)
	}
}

func TestOtherErrors(t *testing.T) {
	// Validation errors carry an object, not a message string.
	c, _ := newClient(t, http.StatusBadRequest, `{"formErrors":[],"fieldErrors":{"destination":["Invalid","Other"],"x":[]}}`)
	err := c.SetAllToursAlert(context.Background(), "u1", true)
	if api.StatusOf(err) != http.StatusBadRequest || api.MessageOf(err) != "" || err.Error() != "api: 400 " ||
		len(api.FieldsOf(err)) != 1 || api.FieldsOf(err)["destination"] != "Invalid" {
		t.Errorf("400: %v %v", err, api.FieldsOf(err))
	}
	if api.FieldsOf(context.Canceled) != nil {
		t.Error("fields of another error")
	}

	c, _ = newClient(t, http.StatusOK, `not json`)
	if _, err := c.BlogPosts(context.Background(), "ru"); err == nil || api.StatusOf(err) != 0 || api.MessageOf(err) != "" {
		t.Errorf("bad JSON: %v", err)
	}

	unreachable, err := api.New("http://127.0.0.1:1", secret)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := unreachable.BlogPosts(context.Background(), "ru"); err == nil || api.StatusOf(err) != 0 {
		t.Errorf("unreachable: %v", err)
	}
}

func TestPersonName(t *testing.T) {
	name, user := "Айгуль", "aigul"
	for _, tc := range []struct {
		p    api.Person
		want string
	}{
		{api.Person{FirstName: &name, Username: &user}, "Айгуль"},
		{api.Person{Username: &user}, "@aigul"},
		{api.Person{FirstName: new(string)}, ""},
	} {
		if got := tc.p.Name(); got != tc.want {
			t.Errorf("Name() = %q, want %q", got, tc.want)
		}
	}
}

func TestOutingWrites(t *testing.T) {
	c, got := newClient(t, http.StatusOK, `{"id":"o9","title":"Пик","isOrganizer":true}`)
	ctx := context.Background()
	trail := "t1"
	in := api.OutingInput{TrailID: &trail, Title: "Пик", StartsAt: "2026-10-24T06:00:00+06:00", MeetingPoint: "Вефа",
		Transport: "car", Seats: 4, Level: "beginner"}
	if o, err := c.CreateOuting(ctx, "u1", in); err != nil || o.ID != "o9" || !o.IsOrganizer {
		t.Fatalf("create: %+v %v", o, err)
	}
	in.TrailID = nil
	if _, err := c.UpdateOuting(ctx, "u1", "o9", in); err != nil {
		t.Fatal(err)
	}
	if err := c.CancelOuting(ctx, "u1", "o9"); err != nil {
		t.Fatal(err)
	}
	if err := c.ReportCondition(ctx, "u1", "ala-kol", api.ConditionInput{ObservedOn: "2026-10-18", TrailState: "wet", Traction: "none"}); err != nil {
		t.Fatal(err)
	}
	body := `{"trailId":"t1","title":"Пик","startsAt":"2026-10-24T06:00:00+06:00","returnsAt":null,"meetingPoint":"Вефа",` +
		`"transport":"car","seats":4,"costPerPerson":null,"level":"beginner","description":null,"contact":null}`
	want := []request{
		{http.MethodPost, "/outings", body, "u1"},
		{http.MethodPut, "/outings/o9", strings.Replace(body, `"trailId":"t1"`, `"trailId":null`, 1), "u1"},
		{http.MethodPost, "/outings/o9/cancel", "", "u1"},
		{http.MethodPost, "/trails/ala-kol/conditions", `{"observedOn":"2026-10-18","trailState":"wet","traction":"none"}`, "u1"},
	}
	for i, w := range want {
		if (*got)[i] != w {
			t.Errorf("request %d:\n%+v\nwant\n%+v", i, (*got)[i], w)
		}
	}
}

func TestDetailReads(t *testing.T) {
	ctx := context.Background()
	c, got := newClient(t, http.StatusOK, `[{"id":"t1","slug":"uchitel","name":"Пик Учитель","description":"…"}]`)
	trails, err := c.Trails(ctx, "пик у")
	if err != nil || len(trails) != 1 || trails[0].ID != "t1" || trails[0].Name != "Пик Учитель" {
		t.Errorf("trails: %+v %v", trails, err)
	}
	if _, err := c.Trails(ctx, ""); err != nil {
		t.Fatal(err)
	}
	if (*got)[0].uri != "/trails?search=%D0%BF%D0%B8%D0%BA+%D1%83&sort=popular" || (*got)[1].uri != "/trails?sort=popular" {
		t.Errorf("trails requests: %+v", *got)
	}

	c, got = newClient(t, http.StatusOK, `{"id":"t1","content":"<p>x</p>","program":[{"title":"День","text":"Т"}],"capacity":12,"telegram":"@a"}`)
	tour, err := c.Tour(ctx, "ru", "son-kol")
	if err != nil || tour.Content != "<p>x</p>" || tour.Program[0].Title != "День" || *tour.Capacity != 12 || *tour.Telegram != "@a" {
		t.Errorf("tour: %+v %v", tour, err)
	}
	if (*got)[0].uri != "/tours/ru/son-kol" {
		t.Errorf("tour request: %+v", (*got)[0])
	}

	c, got = newClient(t, http.StatusOK, `{"id":"p1","content":"<p>y</p>"}`)
	post, err := c.BlogPost(ctx, "ru", "checklist")
	if err != nil || post.Content != "<p>y</p>" || (*got)[0].uri != "/blog/ru/checklist" {
		t.Errorf("post: %+v %v %+v", post, err, *got)
	}
}
