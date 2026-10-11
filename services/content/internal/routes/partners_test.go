package routes_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"

	"github.com/elvn5/sai/services/content/internal/purge"
	"github.com/elvn5/sai/services/pkg/partnerpurge"
)

func TestOnlyPartnersPublish(t *testing.T) {
	c, _, _ := setup(t)
	regular := cookie(t, "user_token", jwt.MapClaims{"sub": regularID, "role": "user"})
	for path, body := range map[string]string{
		"/guides":  `{"locale":"ru","slug":"g","title":"G","content":"x"}`,
		"/gallery": `{"imageUrl":"https://cdn.example.com/a.webp"}`,
		"/blog":    `{"locale":"ru","slug":"b","title":"B","content":"x"}`,
	} {
		if status, resp := c.do("POST", path, body, regular); status != 403 || !strings.Contains(resp, "Only approved partners can do this") {
			t.Errorf("non-partner POST %s: %d %s", path, status, resp)
		}
	}
}

func TestPartnerBlogPosts(t *testing.T) {
	c, _, _ := setup(t)
	admin := cookie(t, "admin_token", jwt.MapClaims{"role": "admin"})
	owner := cookie(t, "user_token", jwt.MapClaims{"sub": ownerID, "role": "user"})
	other := cookie(t, "user_token", jwt.MapClaims{"sub": otherID, "role": "user"})

	status, body := c.do("POST", "/blog", `{"locale":"ru","slug":"karakol-trip","title":"Karakol","content":"<p>Hi<script>x()</script></p>"}`, owner)
	if status != 201 || field(t, body, "isPublished") != true || field(t, body, "userId") != ownerID || field(t, body, "content") != "<p>Hi</p>" {
		t.Fatalf("partner create: %d %s", status, body)
	}
	id := field(t, body, "id").(string)
	if _, list := c.do("GET", "/blog?locale=ru", ""); !strings.Contains(list, id) {
		t.Errorf("partner post not public: %s", list)
	}
	status, body = c.do("POST", "/blog", `{"locale":"ru","slug":"draft","title":"Draft","content":"x","isPublished":false}`, owner)
	if status != 201 || field(t, body, "isPublished") != false {
		t.Errorf("partner draft: %d %s", status, body)
	}
	if _, mine := c.do("GET", "/blog/mine", "", owner); strings.Count(mine, `"id"`) != 2 {
		t.Errorf("mine: %s", mine)
	}

	if status, body := c.do("PUT", "/blog/"+id, `{"title":"Hijack"}`, other); status != 403 || !strings.Contains(body, "You can only manage your own posts") {
		t.Errorf("other partner edit: %d %s", status, body)
	}
	if status, _ := c.do("DELETE", "/blog/"+id, "", other); status != 403 {
		t.Errorf("other partner delete: %d", status)
	}
	if status, body := c.do("PUT", "/blog/"+id, `{"content":"<img src=x onerror=alert(1)>"}`, owner); status != 200 || strings.Contains(body, "onerror") {
		t.Errorf("owner edit not sanitized: %d %s", status, body)
	}

	// Partners can't touch posts written in the admin panel; the admin can
	// touch partners' posts.
	_, body = c.do("POST", "/blog", `{"locale":"ru","slug":"official","title":"Official","content":"x"}`, admin)
	adminPost := field(t, body, "id").(string)
	if field(t, body, "userId") != nil {
		t.Errorf("admin post has an author: %s", body)
	}
	if status, _ := c.do("PUT", "/blog/"+adminPost, `{"title":"Mine now"}`, owner); status != 403 {
		t.Errorf("partner editing admin post: %d", status)
	}
	if status, _ := c.do("DELETE", "/blog/"+id, "", admin); status != 200 {
		t.Errorf("admin deleting partner post: %d", status)
	}
}

func TestPurgeNonPartnerContent(t *testing.T) {
	c, media, db := setup(t)
	ctx := context.Background()
	unknownID := "44444444-4444-4444-4444-444444444444"
	for _, q := range []string{
		`INSERT INTO guides (locale, slug, title, content, is_published, user_id) VALUES
			('ru', 'partner', 'P', 'x', true, '` + ownerID + `'),
			('ru', 'regular', 'R', 'x', true, '` + regularID + `'),
			('ru', 'unknown', 'U', 'x', true, '` + unknownID + `'),
			('ru', 'editorial', 'E', 'x', true, NULL)`,
		`INSERT INTO gallery_items (image_url, user_id) VALUES
			('https://cdn.example.com/partner.webp', '` + ownerID + `'),
			('https://cdn.example.com/regular.webp', '` + regularID + `')`,
	} {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}

	partnerpurge.Run(ctx, db, testUsers, purge.Service(media, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))

	_, guides := c.do("GET", "/guides", "")
	for slug, want := range map[string]bool{"partner": true, "regular": false, "unknown": true, "editorial": true} {
		if got := strings.Contains(guides, `"slug":"`+slug+`"`); got != want {
			t.Errorf("guide %s kept = %v, want %v", slug, got, want)
		}
	}
	if _, gallery := c.do("GET", "/gallery", ""); !strings.Contains(gallery, "partner.webp") || strings.Contains(gallery, "regular.webp") {
		t.Errorf("gallery: %s", gallery)
	}
	if len(media.deleted) != 1 || media.deleted[0] != "https://cdn.example.com/regular.webp" {
		t.Errorf("storage deletes: %v", media.deleted)
	}

	// Runs once: content a later non-partner owns (e.g. a partner whose
	// status changed) stays.
	if _, err := db.Exec(ctx, `INSERT INTO guides (locale, slug, title, content, is_published, user_id)
		VALUES ('ru', 'later', 'L', 'x', true, '`+regularID+`')`); err != nil {
		t.Fatal(err)
	}
	partnerpurge.Run(ctx, db, testUsers, purge.Service(media, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))
	if _, guides := c.do("GET", "/guides", ""); !strings.Contains(guides, `"slug":"later"`) {
		t.Errorf("purge ran twice: %s", guides)
	}
}
