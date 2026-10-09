// Package session writes the session cookies, byte-for-byte as Express's
// res.cookie / res.clearCookie did, so browsers treat them identically.
package session

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/pkg/auth"
)

const (
	userMaxAge  = 30 * 24 * time.Hour
	adminMaxAge = 12 * time.Hour
)

type Cookies struct {
	Secure bool
	now    func() time.Time
}

func NewCookies(secure bool) *Cookies {
	return &Cookies{Secure: secure, now: time.Now}
}

func (c *Cookies) SetUser(ctx *gin.Context, token string) {
	c.set(ctx, auth.UserCookie, token, userMaxAge)
}

func (c *Cookies) SetAdmin(ctx *gin.Context, token string) {
	c.set(ctx, auth.AdminCookie, token, adminMaxAge)
}

// set: name=value; Max-Age=N; Path=/; Expires=...; HttpOnly; [Secure; ]SameSite=Lax
func (c *Cookies) set(ctx *gin.Context, name, value string, maxAge time.Duration) {
	expires := c.now().Add(maxAge).UTC().Format(http.TimeFormat)
	cookie := name + "=" + value +
		"; Max-Age=" + strconv.Itoa(int(maxAge/time.Second)) +
		"; Path=/; Expires=" + expires + "; HttpOnly"
	if c.Secure {
		cookie += "; Secure"
	}
	ctx.Writer.Header().Add("Set-Cookie", cookie+"; SameSite=Lax")
}

// Clear: name=; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT
func Clear(ctx *gin.Context, name string) {
	ctx.Writer.Header().Add("Set-Cookie", name+"=; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT")
}
