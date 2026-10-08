package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/pkg/auth"
	"github.com/elvn5/sai/services/pkg/httperr"
)

const (
	// ContextUserID holds the signed-in site user's id (empty for admins).
	ContextUserID = "auth.userID"
	// ContextPartnerStatus holds the signed-in user's partner status, when
	// a UserLookup is configured.
	ContextPartnerStatus = "auth.partnerStatus"
	// ContextIsAdmin is true when the request carries an admin session.
	ContextIsAdmin = "auth.isAdmin"
)

// ErrUserNotFound is what a UserLookup returns for an unknown id.
var ErrUserNotFound = errors.New("user not found")

// SessionUser is what the auth middleware needs to know about the account
// behind a session.
type SessionUser struct {
	ID            string
	PartnerStatus string // "none", "pending", "approved", "rejected"
}

// UserLookup confirms a session's user still exists (the users service).
type UserLookup interface {
	LookupUser(ctx context.Context, id string) (SessionUser, error)
}

// RequireUser admits requests with a valid site-user session cookie whose
// account still exists, failing with the same messages as the NestJS
// UserJwtAuthGuard. A nil lookup skips the existence check.
func RequireUser(v *auth.Verifier, lookup UserLookup) gin.HandlerFunc {
	return func(c *gin.Context) {
		if authenticateUser(c, v, lookup) {
			c.Next()
		}
	}
}

// RequirePartner admits approved partners only; it runs after RequireUser
// with a lookup, like the NestJS PartnerGuard.
func RequirePartner() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetString(ContextPartnerStatus) != "approved" {
			httperr.Abort(c, http.StatusForbidden, "Only approved partners can do this")
			return
		}
		c.Next()
	}
}

// authenticateUser validates the user cookie (and, with a lookup, the
// account), records the user in the context, and aborts on failure.
func authenticateUser(c *gin.Context, v *auth.Verifier, lookup UserLookup) bool {
	userID, msg := checkUser(c, v)
	if msg != "" {
		httperr.Abort(c, http.StatusUnauthorized, msg)
		return false
	}
	if lookup != nil {
		user, err := lookup.LookupUser(c.Request.Context(), userID)
		switch {
		case errors.Is(err, ErrUserNotFound):
			httperr.Abort(c, http.StatusUnauthorized, "User not found")
			return false
		case err != nil:
			_ = c.Error(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, httperr.InternalServerError())
			return false
		}
		c.Set(ContextPartnerStatus, user.PartnerStatus)
	}
	c.Set(ContextUserID, userID)
	return true
}

// RequireAdminOrUser admits either an admin session or a site-user session,
// like the NestJS AnyAuthGuard (admin is checked first; if that fails, the
// user check's error is what the client sees).
//
// Unlike the NestJS JwtAuthGuard, the admin cookie must carry role "admin" —
// NestJS accepted any validly signed token there.
func RequireAdminOrUser(v *auth.Verifier, lookup UserLookup) gin.HandlerFunc {
	return func(c *gin.Context) {
		if token, err := c.Cookie(auth.AdminCookie); err == nil && token != "" && v.Admin(token) == nil {
			c.Set(ContextIsAdmin, true)
			c.Next()
			return
		}
		if authenticateUser(c, v, lookup) {
			c.Next()
		}
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
