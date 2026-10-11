package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/elvn5/sai/services/pkg/auth"
	"github.com/elvn5/sai/services/pkg/middleware"
)

const jwtSecret = "test-secret-at-least-16-chars"

func token(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestRequireAdminOrUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/x", middleware.RequireAdminOrUser(auth.NewVerifier(jwtSecret), nil), func(c *gin.Context) {
		c.String(http.StatusOK, "%s|%v", c.GetString(middleware.ContextUserID), c.GetBool(middleware.ContextIsAdmin))
	})

	userTok := token(t, jwt.MapClaims{"sub": "u1", "role": "user"})
	adminTok := token(t, jwt.MapClaims{"role": "admin"})
	cases := map[string]struct {
		cookies    map[string]string
		wantStatus int
		wantBody   string
	}{
		"no cookies":    {nil, 401, `{"message":"Not authenticated","error":"Unauthorized","statusCode":401}`},
		"user":          {map[string]string{"user_token": userTok}, 200, "u1|false"},
		"admin":         {map[string]string{"admin_token": adminTok}, 200, "|true"},
		"garbage user":  {map[string]string{"user_token": "nope"}, 401, `{"message":"Invalid or expired session","error":"Unauthorized","statusCode":401}`},
		"admin as user": {map[string]string{"user_token": adminTok}, 401, `{"message":"Invalid session","error":"Unauthorized","statusCode":401}`},
		// NestJS accepted any signed token in admin_token; Go requires role admin.
		"user token in admin cookie": {map[string]string{"admin_token": userTok}, 401, `{"message":"Not authenticated","error":"Unauthorized","statusCode":401}`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/x", nil)
			for k, v := range tc.cookies {
				req.AddCookie(&http.Cookie{Name: k, Value: v})
			}
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus || rec.Body.String() != tc.wantBody {
				t.Fatalf("got %d %s, want %d %s", rec.Code, rec.Body.String(), tc.wantStatus, tc.wantBody)
			}
		})
	}
}

func TestRequireAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", middleware.RequireAdmin(auth.NewVerifier(jwtSecret)), func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	cases := map[string]struct {
		cookie   string
		status   int
		contains string
	}{
		"none":                {"", 401, "Not authenticated"},
		"admin":               {token(t, jwt.MapClaims{"role": "admin"}), 200, "ok"},
		"user token as admin": {token(t, jwt.MapClaims{"sub": "u1", "role": "user"}), 401, "Invalid session"},
		"garbage":             {"nope", 401, "Invalid or expired session"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			if tc.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "admin_token", Value: tc.cookie})
			}
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			if rec.Code != tc.status || !strings.Contains(rec.Body.String(), tc.contains) {
				t.Fatalf("got %d %s", rec.Code, rec.Body.String())
			}
		})
	}
}

type fakeLookup map[string]string // id → partner status

func (f fakeLookup) LookupUser(_ context.Context, id string) (middleware.SessionUser, error) {
	status, ok := f[id]
	if !ok {
		return middleware.SessionUser{}, middleware.ErrUserNotFound
	}
	return middleware.SessionUser{ID: id, PartnerStatus: status}, nil
}

func TestRequireUserWithLookupAndPartner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	lookup := fakeLookup{"partner": "approved", "regular": "none"}
	v := auth.NewVerifier(jwtSecret)
	r.POST("/tours", middleware.RequireUser(v, lookup), middleware.RequirePartner(), func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	cases := map[string]struct {
		sub      string
		status   int
		contains string
	}{
		"approved partner": {"partner", 200, "ok"},
		"not a partner":    {"regular", 403, "Only approved partners can do this"},
		"deleted user":     {"ghost", 401, "User not found"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/tours", nil)
			req.AddCookie(&http.Cookie{Name: "user_token", Value: token(t, jwt.MapClaims{"sub": tc.sub, "role": "user"})})
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			if rec.Code != tc.status || !strings.Contains(rec.Body.String(), tc.contains) {
				t.Fatalf("got %d %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestRequireAdminOrPartner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	lookup := fakeLookup{"partner": "approved", "regular": "pending"}
	r.POST("/blog", middleware.RequireAdminOrPartner(auth.NewVerifier(jwtSecret), lookup), func(c *gin.Context) {
		c.String(http.StatusOK, "%s|%v", c.GetString(middleware.ContextUserID), c.GetBool(middleware.ContextIsAdmin))
	})

	cases := map[string]struct {
		cookie   *http.Cookie
		status   int
		contains string
	}{
		"admin":       {&http.Cookie{Name: "admin_token", Value: token(t, jwt.MapClaims{"role": "admin"})}, 200, "|true"},
		"partner":     {&http.Cookie{Name: "user_token", Value: token(t, jwt.MapClaims{"sub": "partner", "role": "user"})}, 200, "partner|false"},
		"not partner": {&http.Cookie{Name: "user_token", Value: token(t, jwt.MapClaims{"sub": "regular", "role": "user"})}, 403, "Only approved partners"},
		"anonymous":   {nil, 401, "Not authenticated"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/blog", nil)
			if tc.cookie != nil {
				req.AddCookie(tc.cookie)
			}
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			if rec.Code != tc.status || !strings.Contains(rec.Body.String(), tc.contains) {
				t.Fatalf("got %d %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestOptionalUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", middleware.OptionalUser(auth.NewVerifier(jwtSecret)), func(c *gin.Context) {
		c.String(http.StatusOK, "%s|%v", c.GetString(middleware.ContextUserID), c.GetBool(middleware.ContextIsAdmin))
	})
	cases := map[string]struct {
		cookie *http.Cookie
		want   string
	}{
		"anonymous": {nil, "|false"},
		"user":      {&http.Cookie{Name: "user_token", Value: token(t, jwt.MapClaims{"sub": "u1", "role": "user"})}, "u1|false"},
		"garbage":   {&http.Cookie{Name: "user_token", Value: "nope"}, "|false"},
		"admin":     {&http.Cookie{Name: "admin_token", Value: token(t, jwt.MapClaims{"role": "admin"})}, "|true"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			if tc.cookie != nil {
				req.AddCookie(tc.cookie)
			}
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			if rec.Code != 200 || rec.Body.String() != tc.want {
				t.Fatalf("got %d %s, want %s", rec.Code, rec.Body.String(), tc.want)
			}
		})
	}
}
