// Package audit keeps the admin panel's history: every change an admin
// session makes through a service is written to that service's own
// database (table admin_history, created by each service's migrations)
// and read back by the panel per module.
package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/jsontime"
	"github.com/elvn5/sai/services/pkg/middleware"
	"github.com/elvn5/sai/services/pkg/respond"
)

const (
	// maxBody is the largest request body kept; a bigger one is recorded
	// as truncated.
	maxBody = 256 << 10
	// maxString is how many characters of each string value are kept.
	maxString = 300
	// maxResponse is how much of the answer is read for a created id.
	maxResponse = 64 << 10

	defaultLimit = 50
	maxLimit     = 200
)

// Entry is one change, as the panel receives it.
type Entry struct {
	ID       string  `json:"id"`
	Module   string  `json:"module"`
	Method   string  `json:"method"`
	Route    string  `json:"route"`
	Path     string  `json:"path"`
	EntityID *string `json:"entityId"`
	// What changed, in words, when the handler named it (SetSubject).
	Subject *string         `json:"subject"`
	Payload json.RawMessage `json:"payload"`
	Status  int             `json:"status"`
	// The admin panel has one shared password, so the client is what
	// tells sessions apart.
	IP        string        `json:"ip"`
	UserAgent string        `json:"userAgent"`
	CreatedAt jsontime.Time `json:"createdAt"`
}

// Store reads and writes admin_history.
type Store interface {
	Insert(ctx context.Context, e Entry) error
	List(ctx context.Context, q Query) ([]Entry, error)
}

// Query selects a page of a module's history, newest first.
type Query struct {
	Module   string
	EntityID string    // optional
	Before   time.Time // optional: only entries older than this
	Limit    int
}

// Log records admin changes. Modules maps a path's first segment ("/blog")
// to the panel module it belongs to ("blog"); paths outside it aren't
// recorded. A nil *Log records nothing and has no history.
type Log struct {
	Store   Store
	Modules map[string]string
	Logger  *slog.Logger
}

// New keeps the history in db.
func New(db *pgxpool.Pool, modules map[string]string, log *slog.Logger) *Log {
	return &Log{Store: PG{DB: db}, Modules: modules, Logger: log}
}

func (l *Log) module(path string) string {
	segment := path
	if i := strings.IndexByte(path[1:], '/'); i >= 0 {
		segment = path[:i+1]
	}
	return l.Modules[segment]
}

// Middleware records each successful POST, PUT, PATCH or DELETE made with
// an admin session (set by the auth middleware of the route). It goes
// first in the chain, so it sees the request body before any handler.
func (l *Log) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if l == nil || !mutating(c.Request.Method) {
			c.Next()
			return
		}
		module := l.module(c.Request.URL.Path)
		if module == "" {
			c.Next()
			return
		}
		body, truncated := peekBody(c.Request)
		capture := &responseCapture{ResponseWriter: c.Writer}
		c.Writer = capture
		c.Next()

		status := c.Writer.Status()
		if !c.GetBool(middleware.ContextIsAdmin) || status >= http.StatusBadRequest {
			return
		}
		entry := Entry{
			Module: module, Method: c.Request.Method, Route: c.FullPath(), Path: c.Request.URL.Path,
			EntityID: entityID(c, capture.buf.Bytes()), Subject: subject(c), Payload: payload(c.ContentType(), body, truncated),
			Status: status, IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		}
		ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 5*time.Second)
		defer cancel()
		if err := l.Store.Insert(ctx, entry); err != nil && l.Logger != nil {
			l.Logger.WarnContext(ctx, "admin history not recorded", slog.String("route", entry.Route), slog.Any("error", err))
		}
	}
}

// History serves a module's history to the panel (behind RequireAdmin):
// ?limit= (50, at most 200), ?before= (an entry's createdAt, for the next
// page), ?entityId=.
func (l *Log) History(module string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if l == nil {
			respond.JSON(c, http.StatusOK, []Entry{})
			return
		}
		q := Query{Module: module, EntityID: c.Query("entityId"), Limit: defaultLimit}
		if v := c.Query("limit"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 {
				httperr.Abort(c, http.StatusBadRequest, "limit must be a positive number")
				return
			}
			q.Limit = min(n, maxLimit)
		}
		if v := c.Query("before"); v != "" {
			t, err := time.Parse(time.RFC3339Nano, v)
			if err != nil {
				httperr.Abort(c, http.StatusBadRequest, "before must be an ISO date")
				return
			}
			q.Before = t
		}
		entries, err := l.Store.List(c.Request.Context(), q)
		if err != nil {
			_ = c.Error(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, httperr.InternalServerError())
			return
		}
		if entries == nil {
			entries = []Entry{}
		}
		respond.JSON(c, http.StatusOK, entries)
	}
}

const contextSubject = "audit.subject"

// SetSubject names what the request changes (a partner's name, say), for
// changes whose request body doesn't say.
func SetSubject(c *gin.Context, subject string) {
	if subject != "" {
		c.Set(contextSubject, subject)
	}
}

func mutating(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}

// peekBody reads up to maxBody bytes of the request and puts them back in
// front of the rest, so the handler reads the body unchanged.
func peekBody(r *http.Request) (body []byte, truncated bool) {
	if r.Body == nil || r.Body == http.NoBody {
		return nil, false
	}
	body, _ = io.ReadAll(io.LimitReader(r.Body, maxBody+1))
	r.Body = readCloser{io.MultiReader(bytes.NewReader(body), r.Body), r.Body}
	if len(body) > maxBody {
		return nil, true
	}
	return body, false
}

type readCloser struct {
	io.Reader
	io.Closer
}

// payload is the request's JSON with secrets hidden and long strings cut;
// null for other content types.
func payload(contentType string, body []byte, truncated bool) json.RawMessage {
	if truncated {
		return json.RawMessage(`{"_truncated":true}`)
	}
	if contentType != "application/json" || len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return nil
	}
	out, err := json.Marshal(sanitize(v))
	if err != nil {
		return nil
	}
	return out
}

func sanitize(v any) any {
	switch v := v.(type) {
	case map[string]any:
		for k, item := range v {
			if secret(k) {
				v[k] = "***"
			} else {
				v[k] = sanitize(item)
			}
		}
		return v
	case []any:
		for i, item := range v {
			v[i] = sanitize(item)
		}
		return v
	case string:
		if r := []rune(v); len(r) > maxString {
			return string(r[:maxString]) + "…"
		}
		return v
	}
	return v
}

func secret(key string) bool {
	k := strings.ToLower(key)
	for _, word := range []string{"password", "token", "secret"} {
		if strings.Contains(k, word) {
			return true
		}
	}
	return false
}

func subject(c *gin.Context) *string {
	if s := c.GetString(contextSubject); s != "" {
		return &s
	}
	return nil
}

// entityID is what the route changed: its path parameters ("page/locale"
// for two), else the id of what a create answered with.
func entityID(c *gin.Context, response []byte) *string {
	if len(c.Params) > 0 {
		values := make([]string, len(c.Params))
		for i, p := range c.Params {
			values[i] = p.Value
		}
		id := strings.Join(values, "/")
		return &id
	}
	var created struct {
		ID json.RawMessage `json:"id"`
	}
	if json.Unmarshal(response, &created) != nil || len(created.ID) == 0 {
		return nil
	}
	var id string
	if json.Unmarshal(created.ID, &id) != nil {
		id = string(created.ID) // a numeric id
	}
	return &id
}

// responseCapture keeps the start of the answer, to find a created id.
type responseCapture struct {
	gin.ResponseWriter
	buf bytes.Buffer
}

func (w *responseCapture) Write(b []byte) (int, error) {
	if room := maxResponse - w.buf.Len(); room > 0 {
		w.buf.Write(b[:min(len(b), room)])
	}
	return w.ResponseWriter.Write(b)
}

func (w *responseCapture) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

// PG is the Store on a service's PostgreSQL database.
type PG struct{ DB *pgxpool.Pool }

func (s PG) Insert(ctx context.Context, e Entry) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO admin_history
		(module, method, route, path, entity_id, subject, payload, status, ip, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		e.Module, e.Method, e.Route, e.Path, e.EntityID, e.Subject, nullJSON(e.Payload), e.Status, e.IP, e.UserAgent)
	return err
}

func (s PG) List(ctx context.Context, q Query) ([]Entry, error) {
	var before *time.Time
	if !q.Before.IsZero() {
		before = &q.Before
	}
	rows, err := s.DB.Query(ctx, `SELECT id::text, module, method, route, path, entity_id, subject, payload,
		status, ip, user_agent, created_at FROM admin_history
		WHERE module = $1 AND ($2 = '' OR entity_id = $2) AND ($3::timestamptz IS NULL OR created_at < $3)
		ORDER BY created_at DESC LIMIT $4`, q.Module, q.EntityID, before, q.Limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Entry, error) {
		var e Entry
		var payload []byte
		var created time.Time
		err := r.Scan(&e.ID, &e.Module, &e.Method, &e.Route, &e.Path, &e.EntityID, &e.Subject, &payload,
			&e.Status, &e.IP, &e.UserAgent, &created)
		e.Payload = payload
		e.CreatedAt = jsontime.New(created)
		return e, err
	})
}

func nullJSON(b json.RawMessage) any {
	if len(b) == 0 {
		return nil
	}
	return []byte(b)
}
