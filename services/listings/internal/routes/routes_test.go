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

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/elvn5/sai/services/listings/internal/handlers"
	"github.com/elvn5/sai/services/listings/internal/legacyimport"
	"github.com/elvn5/sai/services/listings/internal/migrations"
	"github.com/elvn5/sai/services/listings/internal/routes"
	"github.com/elvn5/sai/services/listings/internal/store"
	"github.com/elvn5/sai/services/pkg/database"
	"github.com/elvn5/sai/services/pkg/middleware"
	"github.com/elvn5/sai/services/pkg/usersclient"
)

const (
	secret  = "test-secret-at-least-16-chars"
	owner   = "11111111-1111-1111-1111-111111111111"
	guest   = "22222222-2222-2222-2222-222222222222"
	partner = "33333333-3333-3333-3333-333333333333"
)

func createDatabase(t *testing.T, migrate bool) (*pgxpool.Pool, string) {
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
	name := "listings_test_" + hex.EncodeToString(suffix)
	dsn := database.DSN(cfg.Host, strconv.Itoa(int(cfg.Port)), cfg.User, cfg.Password, name)
	admin := cfg.Copy()
	admin.Database = "postgres"

	var db *pgxpool.Pool
	if migrate {
		db, err = database.Connect(context.Background(), dsn, migrations.FS, slog.New(slog.DiscardHandler))
	} else {
		var conn *pgx.Conn
		if conn, err = pgx.ConnectConfig(context.Background(), admin); err == nil {
			_, err = conn.Exec(context.Background(), "CREATE DATABASE "+pgx.Identifier{name}.Sanitize())
			_ = conn.Close(context.Background())
		}
		if err == nil {
			db, err = pgxpool.New(context.Background(), dsn)
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
		if conn, err := pgx.ConnectConfig(context.Background(), admin); err == nil {
			_, _ = conn.Exec(context.Background(), "DROP DATABASE "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)")
			_ = conn.Close(context.Background())
		}
	})
	return db, dsn
}

// fakeUsers stands in for the users service.
type fakeUsers map[string]string // id → partner status

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
		if _, ok := f[id]; ok {
			name := "User " + id[:4]
			out[id] = usersclient.Author{ID: id, FirstName: &name}
		}
	}
	return out, nil
}

type client struct {
	t      *testing.T
	router *gin.Engine
}

func (c client) do(method, path, body, userID string) (int, string) {
	c.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if userID != "" {
		tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": userID, "role": "user"}).SignedString([]byte(secret))
		req.AddCookie(&http.Cookie{Name: "user_token", Value: tok})
	}
	rec := httptest.NewRecorder()
	c.router.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

func setup(t *testing.T) client {
	t.Helper()
	db, _ := createDatabase(t, true)
	users := fakeUsers{owner: "none", guest: "none", partner: "approved"}
	gin.SetMode(gin.TestMode)
	return client{t, routes.Setup(handlers.Deps{Store: store.New(db), Authors: users, Log: slog.New(slog.DiscardHandler)}, users, secret)}
}

func obj(t *testing.T, body string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("not an object: %s", body)
	}
	return m
}

func createHotel(c client, slug, extra string) string {
	_, body := c.do("POST", "/hotels", `{"slug":"`+slug+`","name":"  `+slug+` Inn  ","type":"guesthouse",
		"description":"<p onclick=\"x()\">Cozy</p>","destination":"karakol","cancellationPolicy":"free"`+extra+`}`, owner)
	var m map[string]any
	_ = json.Unmarshal([]byte(body), &m)
	id, _ := m["id"].(string)
	return id
}

func TestHotels(t *testing.T) {
	c := setup(t)
	status, body := c.do("POST", "/hotels", `{"slug":"kok-jaiyk","name":"  Kok-Jaiyk  ","type":"guesthouse",
		"description":"<p onclick=\"x()\">Cozy <script>x()</script>house</p>","destination":"karakol",
		"cancellationPolicy":"free","amenities":["wifi","breakfast"],"priceFrom":2500,"telegram":"@kokjaiyk",
		"photos":["https://cdn.example.com/a.webp"]}`, owner)
	h := obj(t, body)
	if status != 201 || h["name"] != "Kok-Jaiyk" || h["description"] != "<p>Cozy house</p>" || h["telegram"] != "kokjaiyk" || h["ratingAverage"] != float64(0) {
		t.Fatalf("create: %d %s", status, body)
	}
	id := h["id"].(string)
	createHotel(c, "yurt-camp", `,"amenities":["horse_riding"],"priceFrom":3000`)

	for query, want := range map[string]int{
		"amenities=wifi,breakfast&freeCancellation=true&maxPrice=3000": 1,
		"amenities=pool":  0,
		"search=yurt":     1,
		"destination=osh": 0,
		"sort=price_desc": 2,
		"limit=1":         1,
	} {
		_, body := c.do("GET", "/hotels?"+query, "", "")
		var page struct {
			Items []any `json:"items"`
			Total int   `json:"total"`
		}
		_ = json.Unmarshal([]byte(body), &page)
		if len(page.Items) != want {
			t.Errorf("%s: %d items (%s)", query, len(page.Items), body)
		}
	}
	if _, body := c.do("GET", "/hotels?sort=price_desc", "", ""); !strings.Contains(body, `"total":2`) || strings.Index(body, "yurt-camp") > strings.Index(body, "kok-jaiyk") {
		t.Errorf("price_desc order: %s", body)
	}
	if status, body := c.do("GET", "/hotels?amenities=spa&limit=abc", "", ""); status != 400 || !strings.Contains(body, `"limit":["Expected number, received nan"]`) || !strings.Contains(body, "received 'spa'") {
		t.Errorf("query validation: %d %s", status, body)
	}

	status, body = c.do("POST", "/hotels", `{"slug":"kok-jaiyk","name":"x","type":"hotel","description":"d","destination":"karakol","cancellationPolicy":"free"}`, owner)
	if status != 409 || !strings.Contains(body, "Slug is already taken") {
		t.Errorf("dup slug: %d %s", status, body)
	}
	status, body = c.do("POST", "/hotels", `{"slug":"Bad","name":"   ","type":"castle","stars":0,"latitude":91,"website":"nope","telegram":"@ab","photos":["ftp://x"]}`, owner)
	errs := obj(t, body)["fieldErrors"].(map[string]any)
	for _, f := range []string{"slug", "name", "type", "stars", "latitude", "website", "telegram", "photos", "description", "destination", "cancellationPolicy"} {
		if errs[f] == nil {
			t.Errorf("missing validation error for %s: %s", f, body)
		}
	}
	if status != 400 {
		t.Errorf("validation status %d", status)
	}

	if status, body := c.do("PUT", "/hotels/"+id, `{"stars":3,"telegram":null}`, guest); status != 403 || !strings.Contains(body, "You can only manage your own hotels") {
		t.Errorf("other user update: %d %s", status, body)
	}
	status, body = c.do("PUT", "/hotels/"+id, `{"stars":3,"telegram":null,"amenities":[]}`, owner)
	if h := obj(t, body); status != 200 || h["stars"] != float64(3) || h["telegram"] != nil || len(h["amenities"].([]any)) != 0 || h["priceFrom"] != float64(2500) {
		t.Errorf("owner update: %d %s", status, body)
	}

	// Deleting a hotel removes its reviews and posts.
	c.do("POST", "/reviews", `{"targetType":"hotel","targetId":"`+id+`","rating":5,"text":"Wonderful stay, thanks!"}`, guest)
	c.do("POST", "/listing-posts", `{"targetType":"hotel","targetId":"`+id+`","text":"Season open"}`, owner)
	if status, _ := c.do("DELETE", "/hotels/"+id, "", owner); status != 200 {
		t.Fatalf("delete: %d", status)
	}
	if _, body := c.do("GET", "/reviews?targetType=hotel&targetId="+id, "", ""); !strings.Contains(body, `"total":0`) {
		t.Errorf("reviews survived delete: %s", body)
	}
	if status, _ := c.do("GET", "/hotels/mine", "", "44444444-4444-4444-4444-444444444444"); status != 401 {
		t.Errorf("unknown user: %d", status)
	}
}

func TestLocalGuides(t *testing.T) {
	c := setup(t)
	if _, body := c.do("GET", "/local-guides/mine", "", owner); body != `{"guide":null}` {
		t.Errorf("mine before create: %s", body)
	}
	payload := `{"slug":"anna","name":"Anna","bio":"<p>Guide</p>","languages":["de","en"],"specializations":["trekking"],"destinations":["karakol","song_kol"],"hasVehicle":true,"pricePerDay":9000}`
	if status, body := c.do("POST", "/local-guides", payload, owner); status != 201 || obj(t, body)["hasVehicle"] != true {
		t.Fatalf("create: %d %s", status, body)
	}
	if status, body := c.do("POST", "/local-guides", strings.Replace(payload, `"anna"`, `"anna-2"`, 1), owner); status != 409 || !strings.Contains(body, "You already have a guide profile") {
		t.Errorf("second profile: %d %s", status, body)
	}
	if status, body := c.do("POST", "/local-guides", `{"slug":"x","name":"X","bio":"b","languages":[],"specializations":["ski"]}`, guest); status != 400 ||
		!strings.Contains(body, `"languages":["Array must contain at least 1 element(s)"]`) || !strings.Contains(body, `"destinations":["Required"]`) {
		t.Errorf("validation: %d %s", status, body)
	}
	for query, want := range map[string]string{
		"language=de&destination=song_kol&hasVehicle=true": `"total":1`,
		"language=ru":        `"total":0`,
		"specialization=ski": `"total":0`,
	} {
		if _, body := c.do("GET", "/local-guides?"+query, "", ""); !strings.Contains(body, want) {
			t.Errorf("%s: %s", query, body)
		}
	}
	if _, body := c.do("GET", "/local-guides/mine", "", owner); !strings.Contains(body, `"slug":"anna"`) {
		t.Errorf("mine after create: %s", body)
	}
}

func TestToursRequirePartnerAndDates(t *testing.T) {
	c := setup(t)
	tour := `{"locale":"ru","slug":"ala-kul","title":"Ала-Куль","content":"<p>3 дня</p>","startDate":"2027-06-10","endDate":"2027-06-12","destination":"karakol","cancellationPolicy":"free"}`
	if status, body := c.do("POST", "/tours", tour, owner); status != 403 || !strings.Contains(body, "Only approved partners can do this") {
		t.Errorf("non-partner: %d %s", status, body)
	}
	status, body := c.do("POST", "/tours", tour, partner)
	tr := obj(t, body)
	if status != 201 || tr["startDate"] != "2027-06-10" || tr["isPublished"] != true || tr["publishedAt"] == nil {
		t.Fatalf("partner create: %d %s", status, body)
	}
	if status, body := c.do("POST", "/tours", strings.Replace(strings.Replace(tour, "ala-kul", "back", 1), "2027-06-12", "2027-06-01", 1), partner); status != 400 ||
		body != `{"formErrors":[],"fieldErrors":{"endDate":["endDate must be on or after startDate"]}}` {
		t.Errorf("date refine: %d %s", status, body)
	}
	if status, body := c.do("POST", "/tours", strings.Replace(tour, "2027-06-10", "2027-02-30", 1), partner); status != 400 || !strings.Contains(body, `"startDate":["Invalid date"]`) {
		t.Errorf("invalid date: %d %s", status, body)
	}
	if status, body := c.do("GET", "/tours", "", ""); status != 400 || body != `{"formErrors":[],"fieldErrors":{"locale":["Required"]}}` {
		t.Errorf("tours without locale: %d %s", status, body)
	}
	if _, body := c.do("GET", "/tours?locale=ru&destination=karakol", "", ""); !strings.Contains(body, "ala-kul") {
		t.Errorf("list: %s", body)
	}
	if status, body := c.do("GET", "/tours/ru/ala-kul", "", ""); status != 200 || obj(t, body)["endDate"] != "2027-06-12" {
		t.Errorf("by slug: %d %s", status, body)
	}
}

func TestReviewsAndPosts(t *testing.T) {
	c := setup(t)
	hotel := createHotel(c, "lake-house", "")
	review := func(user string, rating int) (int, string) {
		return c.do("POST", "/reviews", `{"targetType":"hotel","targetId":"`+hotel+`","rating":`+strconv.Itoa(rating)+`,"text":"   Lovely place, great views   "}`, user)
	}

	if status, body := review(owner, 5); status != 403 || !strings.Contains(body, "You cannot review your own listing") {
		t.Errorf("self review: %d %s", status, body)
	}
	status, body := review(guest, 4)
	r := obj(t, body)
	if status != 201 || r["text"] != "Lovely place, great views" || r["author"].(map[string]any)["firstName"] != "User 2222" {
		t.Fatalf("review: %d %s", status, body)
	}
	if status, body := review(guest, 3); status != 409 || !strings.Contains(body, "You have already reviewed this listing") {
		t.Errorf("duplicate: %d %s", status, body)
	}
	review(partner, 5)
	if _, body := c.do("GET", "/hotels/lake-house", "", ""); !strings.Contains(body, `"ratingAverage":4.5`) || !strings.Contains(body, `"reviewCount":2`) {
		t.Errorf("rating not refreshed: %s", body)
	}
	_, body = c.do("GET", "/reviews?targetType=hotel&targetId="+hotel, "", "")
	if !strings.Contains(body, `"summary":{"average":4.5,"count":2,"distribution":{"1":0,"2":0,"3":0,"4":1,"5":1}}`) {
		t.Errorf("summary: %s", body)
	}
	if status, body := c.do("POST", "/reviews", `{"targetType":"hotel","targetId":"`+hotel+`","rating":6,"text":"short"}`, guest); status != 400 ||
		!strings.Contains(body, `"rating":["Number must be less than or equal to 5"]`) || !strings.Contains(body, `"text":["String must contain at least 10 character(s)"]`) {
		t.Errorf("review validation: %d %s", status, body)
	}
	if status, body := c.do("POST", "/reviews", `{"targetType":"tour","targetId":"`+hotel+`","rating":5,"text":"Nothing here at all"}`, guest); status != 404 || !strings.Contains(body, "Listing not found") {
		t.Errorf("missing target: %d %s", status, body)
	}
	if status, _ := c.do("DELETE", "/reviews/"+r["id"].(string), "", partner); status != 403 {
		t.Errorf("delete other's review: %d", status)
	}
	if status, _ := c.do("DELETE", "/reviews/"+r["id"].(string), "", guest); status != 200 {
		t.Errorf("delete own review: %d", status)
	}
	if _, body := c.do("GET", "/hotels/lake-house", "", ""); !strings.Contains(body, `"ratingAverage":5`) || !strings.Contains(body, `"reviewCount":1`) {
		t.Errorf("rating after delete: %s", body)
	}

	if status, body := c.do("POST", "/listing-posts", `{"targetType":"hotel","targetId":"`+hotel+`","text":"spam"}`, guest); status != 403 || !strings.Contains(body, "Only the listing owner can post here") {
		t.Errorf("non-owner post: %d %s", status, body)
	}
	if status, body := c.do("POST", "/listing-posts", `{"targetType":"hotel","targetId":"`+hotel+`","text":"   "}`, owner); status != 400 ||
		body != `{"formErrors":[],"fieldErrors":{"text":["A post needs text or at least one photo"]}}` {
		t.Errorf("empty post: %d %s", status, body)
	}
	if status, body := c.do("POST", "/listing-posts", `{"targetType":"hotel","targetId":"`+hotel+`","photos":["/uploads/x.webp"]}`, owner); status != 201 || obj(t, body)["text"] != "" {
		t.Errorf("photo post: %d %s", status, body)
	}
	if _, body := c.do("GET", "/listing-posts?targetType=hotel&targetId="+hotel, "", ""); !strings.Contains(body, `"total":1`) || !strings.Contains(body, `"firstName":"User 1111"`) {
		t.Errorf("posts list: %s", body)
	}
	if status, body := c.do("GET", "/listing-posts?targetType=hotel&targetId=nope", "", ""); status != 400 || !strings.Contains(body, `"targetId":["Invalid uuid"]`) {
		t.Errorf("posts query validation: %d %s", status, body)
	}
}

func TestLegacyImport(t *testing.T) {
	db, _ := createDatabase(t, true)
	legacy, legacyDSN := createDatabase(t, false)
	ctx := context.Background()
	_, err := legacy.Exec(ctx, `
		CREATE TABLE hotels (id uuid PRIMARY KEY, slug varchar, name varchar, type varchar, description text,
			destination varchar, address varchar, latitude double precision, longitude double precision,
			"priceFrom" int, stars smallint, amenities text, "cancellationPolicy" varchar, "checkInTime" varchar,
			"checkOutTime" varchar, "coverImageUrl" varchar, photos text, phone varchar, whatsapp varchar,
			telegram varchar, website varchar, "ratingAverage" real, "reviewCount" int, "isPublished" boolean,
			"userId" uuid, "createdAt" timestamp, "updatedAt" timestamp);
		CREATE TABLE tours (id uuid PRIMARY KEY, locale varchar, slug varchar, title varchar, excerpt text, content text,
			price int, "durationDays" int, location varchar, "startDate" date, "endDate" date, destination varchar,
			"cancellationPolicy" varchar, "coverImageUrl" varchar, phone varchar, whatsapp varchar, telegram varchar,
			"ratingAverage" real, "reviewCount" int, "isPublished" boolean, "publishedAt" timestamp, "userId" uuid,
			"createdAt" timestamp, "updatedAt" timestamp);
		CREATE TABLE reviews (id uuid PRIMARY KEY, "targetType" varchar, "targetId" uuid, rating smallint, text text,
			photos text, "userId" uuid, "createdAt" timestamp, "updatedAt" timestamp);
		INSERT INTO hotels VALUES ('aaaaaaaa-0000-0000-0000-000000000001','old-inn','Old Inn','hotel','<p>x</p>','osh',
			NULL,40.5,72.8,1500,NULL,'wifi,parking','strict','14:00',NULL,NULL,'["https://cdn.example.com/1.webp"]',
			NULL,NULL,'oldinn',NULL,4.5,2,true,'`+owner+`','2026-05-01 10:00:00.5','2026-05-01 10:00:00.5');
		INSERT INTO tours VALUES ('bbbbbbbb-0000-0000-0000-000000000001','en','old-tour','Old tour',NULL,'<p>t</p>',
			100,2,NULL,'2027-01-02','2027-01-04',NULL,NULL,NULL,NULL,NULL,NULL,0,0,true,'2026-05-01 10:00:00',
			'`+partner+`','2026-05-01 10:00:00','2026-05-01 10:00:00');
		INSERT INTO reviews VALUES ('cccccccc-0000-0000-0000-000000000001','hotel','aaaaaaaa-0000-0000-0000-000000000001',
			4,'Good old inn here','[]','`+guest+`','2026-05-02 10:00:00','2026-05-02 10:00:00');`)
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.DiscardHandler)
	for range 2 { // the second run must be a no-op
		if err := legacyimport.Run(ctx, db, legacyDSN, log); err != nil {
			t.Fatal(err)
		}
	}

	users := fakeUsers{owner: "none", guest: "none", partner: "approved"}
	gin.SetMode(gin.TestMode)
	c := client{t, routes.Setup(handlers.Deps{Store: store.New(db), Authors: users, Log: log}, users, secret)}
	_, body := c.do("GET", "/hotels/old-inn", "", "")
	if !strings.Contains(body, `"amenities":["wifi","parking"]`) || !strings.Contains(body, `"photos":["https://cdn.example.com/1.webp"]`) ||
		!strings.Contains(body, `"ratingAverage":4.5`) || !strings.Contains(body, `"createdAt":"2026-05-01T10:00:00.500Z"`) {
		t.Errorf("hotel: %s", body)
	}
	_, body = c.do("GET", "/tours/en/old-tour", "", "")
	if !strings.Contains(body, `"startDate":"2027-01-02"`) || !strings.Contains(body, `"endDate":"2027-01-04"`) {
		t.Errorf("tour: %s", body)
	}
	_, body = c.do("GET", "/reviews?targetType=hotel&targetId=aaaaaaaa-0000-0000-0000-000000000001", "", "")
	if !strings.Contains(body, `"total":1`) || !strings.Contains(body, `"photos":[]`) {
		t.Errorf("reviews: %s", body)
	}
}
