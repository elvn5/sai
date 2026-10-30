package audit_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/pkg/audit"
	"github.com/elvn5/sai/services/pkg/middleware"
)

type memStore struct{ entries []audit.Entry }

func (m *memStore) Insert(_ context.Context, e audit.Entry) error {
	m.entries = append(m.entries, e)
	return nil
}

func (m *memStore) List(context.Context, audit.Query) ([]audit.Entry, error) { return m.entries, nil }

func router(t *testing.T, l *audit.Log) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(l.Middleware())
	asAdmin := func(c *gin.Context) {
		if c.GetHeader("X-Admin") != "" {
			c.Set(middleware.ContextIsAdmin, true)
		}
	}
	echo := func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		c.Data(http.StatusCreated, "application/json", body)
	}
	r.POST("/blog", asAdmin, echo)
	r.PUT("/blog/:id", asAdmin, echo)
	r.PUT("/other/:id", asAdmin, echo)
	r.DELETE("/blog/:id", asAdmin, func(c *gin.Context) { c.Status(http.StatusNotFound) })
	return r
}

func send(r *gin.Engine, method, path, body string, admin bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if admin {
		req.Header.Set("X-Admin", "1")
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestMiddlewareRecordsAdminChanges(t *testing.T) {
	st := &memStore{}
	r := router(t, &audit.Log{Store: st, Modules: map[string]string{"/blog": "blog"}})

	body := `{"id":"p1","title":"` + strings.Repeat("я", 400) + `","adminPassword":"x","tags":[{"token":"t"}]}`
	if rec := send(r, "POST", "/blog", body, true); rec.Body.String() != body {
		t.Fatalf("the handler must read the body unchanged: %s", rec.Body.String())
	}
	send(r, "PUT", "/blog/p1", `{"title":"New"}`, false) // not an admin
	send(r, "PUT", "/other/p1", `{"title":"New"}`, true) // not a history module
	send(r, "DELETE", "/blog/p1", "", true)              // failed
	send(r, "PUT", "/blog/p1", `{"title":"Renamed"}`, true)

	if len(st.entries) != 2 {
		t.Fatalf("entries: %+v", st.entries)
	}
	created, updated := st.entries[0], st.entries[1]
	if created.Module != "blog" || created.Route != "/blog" || created.EntityID == nil || *created.EntityID != "p1" || created.Status != 201 {
		t.Errorf("created: %+v", created)
	}
	payload := string(created.Payload)
	if !strings.Contains(payload, `"adminPassword":"***"`) || !strings.Contains(payload, `"token":"***"`) ||
		strings.Count(payload, "я") != 300 {
		t.Errorf("payload: %s", payload)
	}
	if updated.Route != "/blog/:id" || *updated.EntityID != "p1" || string(updated.Payload) != `{"title":"Renamed"}` {
		t.Errorf("updated: %+v", updated)
	}
}

func TestNilLogDoesNothing(t *testing.T) {
	var l *audit.Log
	r := router(t, l)
	r.GET("/blog/admin/history", l.History("blog"))
	if rec := send(r, "PUT", "/blog/p1", `{"a":1}`, true); rec.Code != 201 {
		t.Errorf("put: %d", rec.Code)
	}
	if rec := send(r, "GET", "/blog/admin/history", "", true); rec.Body.String() != `[]` {
		t.Errorf("history: %s", rec.Body.String())
	}
}
