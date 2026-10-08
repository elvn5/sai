package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/pkg/auth"
	"github.com/elvn5/sai/services/pkg/httperr"
)

const (
	// ContextUserID holds the signed-in site user's id (empty for admins).
	ContextUserID = "auth.userID"
	// ContextIsAdmin is true when the request carries an admin session.
	ContextIsAdmin = "auth.isAdmin"
)

// RequireUser admits requests with a valid site-user session cookie,
// failing with the same 401 messages as the NestJS UserJwtAuthGuard.
//
// TODO(stage 4): NestJS also checks the user still exists in the database;
// add that check over gRPC once the users service exists.
func RequireUser(v *auth.Verifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, msg := checkUser(c, v)
		if msg != "" {
			httperr.Abort(c, http.StatusUnauthorized, msg)
			return
		}
		c.Set(ContextUserID, userID)
		c.Next()
	}
}

// RequireAdminOrUser admits either an admin session or a site-user session,
// like the NestJS AnyAuthGuard (admin is checked first; if that fails, the
// user check's error is what the client sees).
//
// Unlike the NestJS JwtAuthGuard, the admin cookie must carry role "admin" —
// NestJS accepted any validly signed token there.
func RequireAdminOrUser(v *auth.Verifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		if token, err := c.Cookie(auth.AdminCookie); err == nil && token != "" && v.Admin(token) == nil {
			c.Set(ContextIsAdmin, true)
			c.Next()
			return
		}
		userID, msg := checkUser(c, v)
		if msg != "" {
			httperr.Abort(c, http.StatusUnauthorized, msg)
			return
		}
		c.Set(ContextUserID, userID)
		c.Next()
	}
}

func checkUser(c *gin.Context, v *auth.Verifier) (userID, errMessage string) {
	token, err := c.Cookie(auth.UserCookie)
	if err != nil || token == "" {
		return "", "Not authenticated"
	}
	userID, err = v.UserID(token)
	switch {
	case errors.Is(err, auth.ErrInvalidSession):
		return "", "Invalid session"
	case err != nil:
		return "", "Invalid or expired session"
	}
	return userID, ""
}

// RequireAdmin admits only admin-panel sessions, failing with the same 401
// messages as the NestJS JwtAuthGuard.
func RequireAdmin(v *auth.Verifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(auth.AdminCookie)
		if err != nil || token == "" {
			httperr.Abort(c, http.StatusUnauthorized, "Not authenticated")
			return
		}
		switch err := v.Admin(token); {
		case errors.Is(err, auth.ErrInvalidSession):
			httperr.Abort(c, http.StatusUnauthorized, "Invalid session")
			return
		case err != nil:
			httperr.Abort(c, http.StatusUnauthorized, "Invalid or expired session")
			return
		}
		c.Set(ContextIsAdmin, true)
		c.Next()
	}
}
