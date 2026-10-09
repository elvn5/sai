package handlers

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/pkg/auth"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/pkg/validation"
	"github.com/elvn5/sai/services/users/internal/session"
)

type adminLoginRequest struct {
	Password *string `json:"password" binding:"required,adminpassword"`
}

// AdminAuth serves /auth: the admin panel's single-password sign-in.
type AdminAuth struct{ Deps }

func (h *AdminAuth) Login(c *gin.Context) {
	var req adminLoginRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	if h.AdminPassword == "" {
		httperr.Abort(c, http.StatusUnauthorized, "Admin password is not configured")
		return
	}
	if subtle.ConstantTimeCompare([]byte(*req.Password), []byte(h.AdminPassword)) != 1 {
		httperr.Abort(c, http.StatusUnauthorized, "Invalid password")
		return
	}
	token, err := h.Signer.AdminToken()
	if err != nil {
		h.internalError(c, err)
		return
	}
	h.Cookies.SetAdmin(c, token)
	respond.JSON(c, http.StatusOK, gin.H{"success": true})
}

func (h *AdminAuth) Logout(c *gin.Context) {
	session.Clear(c, auth.AdminCookie)
	respond.JSON(c, http.StatusOK, gin.H{"success": true})
}

// Me runs behind RequireAdmin.
func (h *AdminAuth) Me(c *gin.Context) {
	respond.JSON(c, http.StatusOK, gin.H{"authenticated": true})
}
