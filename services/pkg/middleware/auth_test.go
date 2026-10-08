package middleware_test

import (
	"net/http"
	"net/http/httptest"
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
	r.POST("/x", middleware.RequireAdminOrUser(auth.NewVerifier(jwtSecret)), func(c *gin.Context) {
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
