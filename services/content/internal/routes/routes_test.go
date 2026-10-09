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

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/elvn5/sai/services/content/internal/legacyimport"
	"github.com/elvn5/sai/services/content/internal/migrations"
	"github.com/elvn5/sai/services/content/internal/routes"
	"github.com/elvn5/sai/services/content/internal/store"
	"github.com/elvn5/sai/services/pkg/database"
	"github.com/elvn5/sai/services/pkg/middleware"
)

const secret = "test-secret-at-least-16-chars"

const (
	// ownerID and otherID are approved partners; regularID isn't.
	ownerID   = "11111111-1111-1111-1111-111111111111"
	otherID   = "22222222-2222-2222-2222-222222222222"
	regularID = "33333333-3333-3333-3333-333333333333"
)

// fakeUsers stands in for the users service: id → partner status.
type fakeUsers map[string]string

var testUsers = fakeUsers{ownerID: "approved", otherID: "approved", regularID: "none"}

func (f fakeUsers) LookupUser(_ context.Context, id string) (middleware.SessionUser, error) {
	status, ok := f[id]
	if !ok {
		return middleware.SessionUser{}, middleware.ErrUserNotFound
	}
	return middleware.SessionUser{ID: id, PartnerStatus: status}, nil
}

func (f fakeUsers) PartnerStatuses(_ context.Context, ids []string) (map[string]string, error) {
	out := map[string]string{}
	for _, id := range ids {
		if status, ok := f[id]; ok {
			out[id] = status
		}
	}
	return out, nil
}

// newDatabase creates a throwaway, migrated content database on the server
// in TEST_POSTGRES_DSN (any database name; it's replaced). Skips without it.
func newDatabase(t *testing.T) (*pgxpool.Pool, string) {
	return createDatabase(t, true)
}

// newLegacyDatabase creates an empty throwaway database.
func newLegacyDatabase(t *testing.T) (*pgxpool.Pool, string) {
	return createDatabase(t, false)
}

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
	name := "content_test_" + hex.EncodeToString(suffix)
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

func jsonNumber(n int) string { return strconv.Itoa(n) }

type fakeMedia struct {
	mu      sync.Mutex
	deleted []string
}

func (f *fakeMedia) DeleteObject(_ context.Context, url string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleted = append(f.deleted, url)
	return nil
}

type client struct {
	t      *testing.T
	router *gin.Engine
}

func (c client) do(method, path, body string, cookies ...*http.Cookie) (int, string) {
	c.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, ck := range cookies {
		req.AddCookie(ck)
	}
	rec := httptest.NewRecorder()
	c.router.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

func cookie(t *testing.T, name string, claims jwt.MapClaims) *http.Cookie {
	t.Helper()
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: name, Value: tok}
}

func setup(t *testing.T) (client, *fakeMedia, *pgxpool.Pool) {
	t.Helper()
	db, _ := newDatabase(t)
	media := &fakeMedia{}
	gin.SetMode(gin.TestMode)
	r := routes.Setup(routes.Deps{Store: store.New(db), Media: media, Users: testUsers, JWTSecret: secret, Log: slog.New(slog.DiscardHandler)})
	return client{t, r}, media, db
}

func field(t *testing.T, body, key string) any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("not an object: %s", body)
	}
	return m[key]
}

func TestBlogPublishingLifecycle(t *testing.T) {
	c, _, _ := setup(t)
	admin := cookie(t, "admin_token", jwt.MapClaims{"role": "admin"})

	status, body := c.do("POST", "/blog", `{"locale":"ru","slug":"first","title":"First","content":"<p>x</p>"}`, admin)
	if status != 201 || field(t, body, "isPublished") != false || field(t, body, "publishedAt") != nil {
		t.Fatalf("create draft: %d %s", status, body)
	}
	if !strings.Contains(body, `"content":"<p>x</p>"`) {
		t.Errorf("HTML was escaped in JSON: %s", body)
	}
	id := field(t, body, "id").(string)

	if _, list := c.do("GET", "/blog?locale=ru", ""); list != "[]" {
		t.Errorf("draft visible publicly: %s", list)
	}
	if status, _ := c.do("POST", "/blog", `{"locale":"ru","slug":"first","title":"Dup","content":"x"}`, admin); status != 409 {
		t.Errorf("duplicate slug: %d", status)
	}

	status, body = c.do("PUT", "/blog/"+id, `{"isPublished":true,"excerpt":"Short"}`, admin)
	if status != 200 || field(t, body, "publishedAt") == nil || field(t, body, "excerpt") != "Short" {
		t.Fatalf("publish: %d %s", status, body)
	}
	// null clears; absent fields stay.
	status, body = c.do("PUT", "/blog/"+id, `{"excerpt":null}`, admin)
	if status != 200 || field(t, body, "excerpt") != nil || field(t, body, "title") != "First" || field(t, body, "publishedAt") == nil {
		t.Fatalf("clear excerpt: %d %s", status, body)
	}

	if status, body := c.do("GET", "/blog/ru/first", ""); status != 200 || field(t, body, "id") != id {
		t.Errorf("by slug: %d %s", status, body)
	}
	if _, list := c.do("GET", "/blog", ""); !strings.Contains(list, id) {
		t.Errorf("no locale should list every locale: %s", list)
	}
	status, body = c.do("PUT", "/blog/"+id, `{"isPublished":false}`, admin)
	if status != 200 || field(t, body, "publishedAt") != nil {
		t.Fatalf("unpublish: %d %s", status, body)
	}

	if status, body := c.do("DELETE", "/blog/"+id, "", admin); status != 200 || body != "" {
		t.Errorf("delete: %d %q", status, body)
	}
	if status, body := c.do("GET", "/blog/admin/"+id, "", admin); status != 404 || !strings.Contains(body, "Post not found") {
		t.Errorf("after delete: %d %s", status, body)
	}
	if status, _ := c.do("GET", "/blog/admin/not-a-uuid", "", admin); status != 404 {
		t.Errorf("malformed id: %d", status)
	}
	user := cookie(t, "user_token", jwt.MapClaims{"sub": regularID, "role": "user"})
	if status, body := c.do("POST", "/blog", `{}`, user); status != 403 || !strings.Contains(body, "Only approved partners") {
		t.Errorf("non-partner creating blog post: %d %s", status, body)
	}
	if status, body := c.do("POST", "/blog", `{}`); status != 401 || !strings.Contains(body, "Not authenticated") {
		t.Errorf("anonymous creating blog post: %d %s", status, body)
	}
}

func TestGuidesOwnershipAndSanitizing(t *testing.T) {
	c, _, _ := setup(t)
	owner := cookie(t, "user_token", jwt.MapClaims{"sub": ownerID, "role": "user"})
	other := cookie(t, "user_token", jwt.MapClaims{"sub": otherID, "role": "user"})

	status, body := c.do("POST", "/guides", `{"locale":"en","slug":"ala-kul","title":"Ala-Kul",
		"content":"<p onclick=\"x()\">Go <script>bad()</script><a href=\"https://x.com\">map</a></p>",
		"equipmentChecklist":["tent","boots"],"attachments":[{"url":"/uploads/a.gpx","name":"track.gpx","size":12}]}`, owner)
	if status != 201 {
		t.Fatalf("create: %d %s", status, body)
	}
	if got := field(t, body, "content"); got != `<p>Go <a href="https://x.com" target="_blank" rel="noopener noreferrer">map</a></p>` {
		t.Errorf("content not sanitized like sanitize-html: %v", got)
	}
	if field(t, body, "difficulty") != "easy" || field(t, body, "season") != "all_year" || field(t, body, "isPublished") != true {
		t.Errorf("defaults: %s", body)
	}
	id := field(t, body, "id").(string)

	if status, body := c.do("PUT", "/guides/"+id, `{"title":"Hijack"}`, other); status != 403 || !strings.Contains(body, "You can only manage your own guides") {
		t.Errorf("other user edit: %d %s", status, body)
	}
	status, body = c.do("PUT", "/guides/"+id, `{"difficulty":"hard","equipmentChecklist":[]}`, owner)
	if status != 200 || field(t, body, "difficulty") != "hard" || len(field(t, body, "equipmentChecklist").([]any)) != 0 || len(field(t, body, "attachments").([]any)) != 1 {
		t.Fatalf("owner edit: %d %s", status, body)
	}
	if _, mine := c.do("GET", "/guides/mine", "", owner); !strings.Contains(mine, id) {
		t.Errorf("mine: %s", mine)
	}
	if _, mine := c.do("GET", "/guides/mine", "", other); mine != "[]" {
		t.Errorf("other's mine: %s", mine)
	}

	status, body = c.do("POST", "/guides", `{"locale":"xx","slug":"Bad Slug","title":"","content":"x","attachments":[{"url":"","name":"n","size":-1}]}`, owner)
	want := map[string]bool{"locale": true, "slug": true, "title": true, "attachments": true}
	var errs struct {
		FieldErrors map[string][]string `json:"fieldErrors"`
	}
	_ = json.Unmarshal([]byte(body), &errs)
	for k := range want {
		if len(errs.FieldErrors[k]) == 0 {
			t.Errorf("validation missing %s: %s", k, body)
		}
	}
	if status != 400 {
		t.Errorf("validation status %d", status)
	}
}

func TestGalleryFeedAndDelete(t *testing.T) {
	c, media, _ := setup(t)
	owner := cookie(t, "user_token", jwt.MapClaims{"sub": ownerID, "role": "user"})
	var ids []string
	for i, loc := range []string{"Karakol", "Song-Kol", "Karakol"} {
		_, body := c.do("POST", "/gallery", `{"imageUrl":"https://cdn.example.com/gallery/`+jsonNumber(i)+`.webp","location":"`+loc+`","caption":"photo `+jsonNumber(i)+`"}`, owner)
		ids = append(ids, field(t, body, "id").(string))
	}

	if _, body := c.do("GET", "/gallery/locations", ""); body != `["Karakol","Song-Kol"]` {
		t.Errorf("locations: %s", body)
	}
	if _, body := c.do("GET", "/gallery?location=Karakol", ""); strings.Count(body, `"id"`) != 2 {
		t.Errorf("location filter: %s", body)
	}
	if _, body := c.do("GET", "/gallery?search=PHOTO%201", ""); strings.Count(body, `"id"`) != 1 {
		t.Errorf("search: %s", body)
	}
	// JS parseInt semantics: "1abc" → 1, "0" → default 20, "-5" → 1.
	for query, want := range map[string]int{"limit=1abc": 1, "limit=0": 3, "limit=-5": 1, "limit=x": 3, "offset=2": 1, "offset=-3": 3} {
		if _, body := c.do("GET", "/gallery?"+query, ""); strings.Count(body, `"id"`) != want {
			t.Errorf("%s: got %d items", query, strings.Count(body, `"id"`))
		}
	}

	if status, body := c.do("PUT", "/gallery/"+ids[0], `{"caption":null}`, owner); status != 200 || field(t, body, "caption") != nil {
		t.Errorf("clear caption: %d %s", status, body)
	}
	other := cookie(t, "user_token", jwt.MapClaims{"sub": otherID, "role": "user"})
	if status, _ := c.do("DELETE", "/gallery/"+ids[0], "", other); status != 403 {
		t.Errorf("other delete: %d", status)
	}
	if status, body := c.do("DELETE", "/gallery/"+ids[0], "", owner); status != 200 || body != "" {
		t.Errorf("delete: %d %s", status, body)
	}
	if len(media.deleted) != 1 || media.deleted[0] != "https://cdn.example.com/gallery/0.webp" {
		t.Errorf("media delete calls: %v", media.deleted)
	}
}

func TestPageSeo(t *testing.T) {
	c, _, _ := setup(t)
	admin := cookie(t, "admin_token", jwt.MapClaims{"role": "admin"})

	if status, body := c.do("GET", "/page-seo/home/ru", ""); status != 200 || body != "" {
		t.Errorf("missing override: %d %q", status, body)
	}
	status, body := c.do("PUT", "/page-seo/home/ru", `{"title":"Ermekten","description":"Desc"}`, admin)
	if status != 200 || field(t, body, "title") != "Ermekten" || field(t, body, "ogImageUrl") != nil {
		t.Fatalf("create: %d %s", status, body)
	}
	status, body = c.do("PUT", "/page-seo/home/ru", `{"title":null}`, admin)
	if status != 200 || field(t, body, "title") != nil || field(t, body, "description") != "Desc" {
		t.Fatalf("partial update: %d %s", status, body)
	}
	if _, body := c.do("GET", "/page-seo/admin/all", "", admin); strings.Count(body, `"id"`) != 1 {
		t.Errorf("admin list: %s", body)
	}
	if status, _ := c.do("GET", "/page-seo/admin/all", ""); status != 401 {
		t.Errorf("anonymous admin list: %d", status)
	}
}

// The legacy tables use TypeORM's camelCase columns, enums, simple-array
// and simple-json; the import must convert them and run only once.
func TestLegacyImport(t *testing.T) {
	db, _ := newDatabase(t)
	legacyDB, legacyDSN := newLegacyDatabase(t)
	ctx := context.Background()
	_, err := legacyDB.Exec(ctx, `
		CREATE TYPE locale_enum AS ENUM ('ru','ky','en','de');
		CREATE TABLE blog_posts (id uuid PRIMARY KEY, locale locale_enum, slug varchar, title varchar, excerpt text,
			content text, "coverImageUrl" varchar, "seoTitle" varchar, "seoDescription" varchar, "isPublished" boolean,
			"publishedAt" timestamp, "createdAt" timestamp, "updatedAt" timestamp);
		CREATE TABLE guides (id uuid PRIMARY KEY, locale locale_enum, slug varchar, title varchar, excerpt text,
			content text, difficulty varchar, season varchar, "equipmentChecklist" text, "coverImageUrl" varchar,
			attachments text, "seoTitle" varchar, "seoDescription" varchar, "isPublished" boolean,
			"publishedAt" timestamp, "userId" uuid, "createdAt" timestamp, "updatedAt" timestamp);
		INSERT INTO blog_posts VALUES ('aaaaaaaa-0000-0000-0000-000000000001','ru','hello','Hello',NULL,'<p>x</p>',
			NULL,NULL,NULL,true,'2026-05-01 10:00:00.123','2026-05-01 09:00:00','2026-05-02 09:00:00');
		INSERT INTO guides VALUES ('bbbbbbbb-0000-0000-0000-000000000001','en','trek','Trek',NULL,'<p>y</p>',
			'hard','summer','tent,boots',NULL,'[{"url":"/uploads/t.gpx","name":"t.gpx","size":5}]',NULL,NULL,true,
			'2026-05-01 10:00:00',NULL,'2026-05-01 10:00:00','2026-05-01 10:00:00'),
			('bbbbbbbb-0000-0000-0000-000000000002','en','empty','Empty',NULL,'<p>z</p>',
			'easy','all_year','',NULL,'[]',NULL,NULL,true,'2026-05-01 10:00:00','`+ownerID+`','2026-05-01 10:00:00','2026-05-01 10:00:00');`)
	if err != nil {
		t.Fatal(err)
	}

	log := slog.New(slog.DiscardHandler)
	if err := legacyimport.Run(ctx, db, legacyDSN, log); err != nil {
		t.Fatal(err)
	}
	// Second run is a no-op even if legacy data changed.
	_, _ = legacyDB.Exec(ctx, `UPDATE blog_posts SET title = 'Changed'`)
	if err := legacyimport.Run(ctx, db, legacyDSN, log); err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	c := client{t, routes.Setup(routes.Deps{Store: store.New(db), JWTSecret: secret, Log: log})}
	_, body := c.do("GET", "/blog/ru/hello", "")
	if field(t, body, "title") != "Hello" || field(t, body, "publishedAt") != "2026-05-01T10:00:00.123Z" || field(t, body, "createdAt") != "2026-05-01T09:00:00.000Z" {
		t.Errorf("blog post: %s", body)
	}
	_, body = c.do("GET", "/guides/en/trek", "")
	if !strings.Contains(body, `"equipmentChecklist":["tent","boots"]`) || !strings.Contains(body, `"attachments":[{"url":"/uploads/t.gpx","name":"t.gpx","size":5}]`) || field(t, body, "userId") != nil {
		t.Errorf("guide: %s", body)
	}
	_, body = c.do("GET", "/guides/en/empty", "")
	if !strings.Contains(body, `"equipmentChecklist":[]`) || field(t, body, "userId") != ownerID {
		t.Errorf("guide with empty list: %s", body)
	}
}

// v1.0.3 (production before the migration) had gallery "region" instead of
// "location" and no tours/listing tables; the import still works.
func TestLegacyImportFromOldSchema(t *testing.T) {
	db, _ := newDatabase(t)
	legacyDB, legacyDSN := newLegacyDatabase(t)
	ctx := context.Background()
	_, err := legacyDB.Exec(ctx, `
		CREATE TYPE region AS ENUM ('issyk_kul', 'chuy');
		CREATE TABLE gallery_items (id uuid PRIMARY KEY, "imageUrl" varchar, caption varchar, "altText" varchar,
			"sortOrder" int, "isPublished" boolean, "createdAt" timestamp, "updatedAt" timestamp, "userId" uuid, region region);
		INSERT INTO gallery_items VALUES ('cccccccc-0000-0000-0000-000000000001', '/uploads/lake.jpg', 'Озеро', NULL, 0,
			true, '2026-09-05 10:00', '2026-09-05 10:00', NULL, 'issyk_kul');`)
	if err != nil {
		t.Fatal(err)
	}
	if err := legacyimport.Run(ctx, db, legacyDSN, slog.New(slog.DiscardHandler)); err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	c := client{t, routes.Setup(routes.Deps{Store: store.New(db), JWTSecret: secret, Log: slog.New(slog.DiscardHandler)})}
	if _, body := c.do("GET", "/gallery", ""); !strings.Contains(body, `"imageUrl":"/uploads/lake.jpg","location":null,"caption":"Озеро"`) {
		t.Errorf("gallery: %s", body)
	}
}
