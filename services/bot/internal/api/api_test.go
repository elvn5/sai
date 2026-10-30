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
		`[{"id":"t1","title":"Сон-Кёль","startDate":"2026-10-24","meetingTime":"06:30","capacity":10,"taken":7,"isJoined":true,"organizer":{"id":"u2","firstName":"Айгуль"}}]`)
	list, err := c.JoinedTours(context.Background(), "u1")
	if err != nil || len(list) != 1 || list[0].Organizer.Name() != "Айгуль" || !list[0].IsJoined ||
		!list[0].StartsAt().Equal(time.Date(2026, 10, 24, 0, 30, 0, 0, time.UTC)) || list[0].Full() {
		t.Fatalf("JoinedTours = %+v, %v", list, err)
	}
	if r := (*got)[0]; r.method != http.MethodGet || r.uri != "/tours/joined" || r.userID != "u1" {
		t.Errorf("request: %+v", r)
	}
}

// A tour without a meeting time meets at 09:00 in Bishkek; a full one has
// no places left.
func TestTourStart(t *testing.T) {
	ten := 10
	tour := api.Tour{StartDate: "2026-10-24", Capacity: &ten, Taken: 10}
	if !tour.StartsAt().Equal(time.Date(2026, 10, 24, 3, 0, 0, 0, time.UTC)) || !tour.Full() {
		t.Errorf("start %v, full %v", tour.StartsAt(), tour.Full())
	}
	if (api.Tour{StartDate: "2026-10-24", Taken: 99}).Full() {
		t.Error("a tour without a limit is never full")
	}
}

func TestWrites(t *testing.T) {
	c, got := newClient(t, http.StatusOK, `[]`)
	ctx := context.Background()
	for _, call := range []func() error{
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
		{http.MethodPut, "/favorites/tour/t1", "", "u1"},
		{http.MethodDelete, "/favorites/tour/t1", "", "u1"},
		{http.MethodPut, "/tour-alerts", `{"destination":""}`, "u1"},
		{http.MethodGet, "/tours?from=2026-09-20&locale=ru", "", ""},
	}
	for i, w := range want {
		if (*got)[i] != w {
			t.Errorf("request %d: %+v, want %+v", i, (*got)[i], w)
		}
	}
}

// Upcoming tours keep those on the way (sign-up stays open), not those
// over.
func TestUpcomingTours(t *testing.T) {
	c, _ := newClient(t, http.StatusOK, `[{"id":"on-the-way","startDate":"2026-10-19","endDate":"2026-10-21"},
		{"id":"over","startDate":"2026-10-18","endDate":"2026-10-19"},{"id":"today","startDate":"2026-10-20","endDate":"2026-10-20"},
		{"id":"next","startDate":"2026-11-01","endDate":"2026-11-02"}]`)
	tours, err := c.UpcomingTours(context.Background(), "ru") // 18:00 Bishkek on 2026-10-20
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, tour := range tours {
		ids = append(ids, tour.ID)
	}
	if strings.Join(ids, ",") != "on-the-way,next" {
		t.Errorf("upcoming: %v", ids)
	}
}

func TestErrors(t *testing.T) {
	c, _ := newClient(t, http.StatusConflict, `{"statusCode":409,"message":"No places left","error":"Conflict"}`)
	_, err := c.JoinTour(context.Background(), "u1", "t1")
	if api.StatusOf(err) != http.StatusConflict || api.MessageOf(err) != "No places left" {
		t.Errorf("error: %v", err)
	}
}

func TestReads(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		answer, uri, userID string
		call                func(c *api.Client) (any, error)
		check               func(v any) bool
	}{
		{`{"id":"t 1","isJoined":true,"canRate":true,"myRating":{"rating":4,"text":null},"roster":[{"id":"u2","joinedAt":"2026-10-18T03:00:00Z"}]}`, "/tours/by-id/t%201", "u1",
			func(c *api.Client) (any, error) { return c.TourByID(ctx, "u1", "t 1") },
			func(v any) bool {
				o := v.(api.Tour)
				return o.IsJoined && o.CanRate && o.MyRating.Rating == 4 && o.Roster[0].ID == "u2" && o.Roster[0].ArrivedAt == nil
			}},
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

func TestParticipationWrites(t *testing.T) {
	c, got := newClient(t, http.StatusOK, `{"id":"t9","title":"Сон-Кёль","isOrganizer":true}`)
	ctx := context.Background()
	if o, err := c.SetArrival(ctx, "u1", "t9", "u2", true); err != nil || o.ID != "t9" || !o.IsOrganizer {
		t.Fatalf("arrival: %+v %v", o, err)
	}
	if _, err := c.SetArrival(ctx, "u1", "t9", "u2", false); err != nil {
		t.Fatal(err)
	}
	text := "Отлично"
	if _, err := c.RateTour(ctx, "u3", "t9", 5, &text); err != nil {
		t.Fatal(err)
	}
	if _, err := c.JoinTour(ctx, "u3", "t9"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.LeaveTour(ctx, "u3", "t9"); err != nil {
		t.Fatal(err)
	}
	want := []request{
		{http.MethodPut, "/tours/t9/participants/u2/arrival", "", "u1"},
		{http.MethodDelete, "/tours/t9/participants/u2/arrival", "", "u1"},
		{http.MethodPut, "/tours/t9/rating", `{"rating":5,"text":"Отлично"}`, "u3"},
		{http.MethodPost, "/tours/t9/join", "", "u3"},
		{http.MethodDelete, "/tours/t9/join", "", "u3"},
	}
	for i, w := range want {
		if (*got)[i] != w {
			t.Errorf("request %d:\n%+v\nwant\n%+v", i, (*got)[i], w)
		}
	}
}

func TestDetailReads(t *testing.T) {
	ctx := context.Background()
	c, got := newClient(t, http.StatusOK, `{"id":"t1","content":"<p>x</p>","program":[{"title":"День","text":"Т"}],"capacity":12,"telegram":"@a"}`)
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
