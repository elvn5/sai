package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/pkg/validation"
	"github.com/elvn5/sai/services/users/internal/email"
	"github.com/elvn5/sai/services/users/internal/models"
	"github.com/elvn5/sai/services/users/internal/store"
	"github.com/elvn5/sai/services/users/internal/verification"
)

const passwordCost = 10

// PendingVerification tells the site how an account is to be confirmed:
// a typed email code, or (phones) a Telegram deep link carrying Token.
type PendingVerification struct {
	Identifier string `json:"identifier"`
	Method     string `json:"method"`
	Token      string `json:"token,omitempty"`
}

type registerRequest struct {
	Email     *string `json:"email" binding:"omitnil,zodemail,max=255"`
	Phone     *string `json:"phone" binding:"omitnil,min=5,max=30"`
	Password  *string `json:"password" binding:"required,min=8,bcryptpassword"`
	FirstName *string `json:"firstName" binding:"omitnil,min=1,max=100"`
}

func (r *registerRequest) Refine() map[string]string {
	if (r.Email == nil || *r.Email == "") && (r.Phone == nil || *r.Phone == "") {
		return map[string]string{"email": "Email or phone is required"}
	}
	return nil
}

type loginRequest struct {
	Identifier *string `json:"identifier" binding:"required,min=1,max=255"`
	Password   *string `json:"password" binding:"required,min=1,max=100"`
}

type verifyRequest struct {
	Identifier *string `json:"identifier" binding:"required,min=1,max=255"`
	Code       *string `json:"code" binding:"required,sixdigits"`
}

type resetPasswordRequest struct {
	Identifier *string `json:"identifier" binding:"required,min=1,max=255"`
	Code       *string `json:"code" binding:"required,sixdigits"`
	Password   *string `json:"password" binding:"required,min=8,bcryptpassword"`
}

type identifierRequest struct {
	Identifier *string `json:"identifier" binding:"required,min=1,max=255"`
}

// Credentials serves /auth/user: email/phone + password accounts.
type Credentials struct{ Deps }

func (h *Credentials) Register(c *gin.Context) {
	var req registerRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	ctx := c.Request.Context()
	if req.Email != nil {
		if taken, err := h.Store.EmailTaken(ctx, *req.Email); err != nil {
			h.internalError(c, err)
			return
		} else if taken {
			httperr.Abort(c, http.StatusConflict, "Email is already registered")
			return
		}
	}
	if req.Phone != nil {
		if taken, err := h.Store.PhoneTaken(ctx, *req.Phone); err != nil {
			h.internalError(c, err)
			return
		} else if taken {
			httperr.Abort(c, http.StatusConflict, "Phone is already registered")
			return
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), passwordCost)
	if err != nil {
		h.internalError(c, err)
		return
	}
	user, err := h.Store.Insert(ctx, models.User{
		Email: req.Email, Phone: req.Phone, PasswordHash: ptr(string(hash)), FirstName: req.FirstName,
		IsVerified: false,
	})
	if errors.Is(err, store.ErrDuplicate) {
		httperr.Abort(c, http.StatusConflict, "Email is already registered")
		return
	} else if err != nil {
		h.internalError(c, err)
		return
	}

	pending, ok := h.issueVerification(c, user)
	if !ok {
		// Don't leave an unverifiable account holding the email/phone hostage.
		if err := h.Store.Delete(ctx, user.ID); err != nil {
			h.Log.ErrorContext(ctx, "failed to remove unverifiable account", "error", err)
		}
		return
	}
	respond.JSON(c, http.StatusOK, gin.H{"verification": pending})
}

func (h *Credentials) Login(c *gin.Context) {
	var req loginRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	user, err := h.Store.ByIdentifier(c.Request.Context(), *req.Identifier)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		h.internalError(c, err)
		return
	}
	if err != nil || user.PasswordHash == nil ||
		bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(*req.Password)) != nil {
		httperr.Abort(c, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	if !user.IsVerified {
		method := "telegram"
		if user.Email != nil {
			method = "email"
		}
		respond.JSON(c, http.StatusOK, gin.H{"verification": PendingVerification{Identifier: *req.Identifier, Method: method}})
		return
	}
	h.startSession(c, user)
}

func (h *Credentials) Verify(c *gin.Context) {
	var req verifyRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	ctx := c.Request.Context()
	user, ok := h.pendingAccount(c, *req.Identifier)
	if !ok {
		return
	}
	switch {
	case user.VerificationCodeHash == nil || user.VerificationCodeExpiresAt == nil:
		httperr.Abort(c, http.StatusBadRequest, "No verification code pending. Please request a new one.")
		return
	case user.VerificationCodeExpiresAt.Before(h.now()):
		httperr.Abort(c, http.StatusBadRequest, "Code has expired. Please request a new one.")
		return
	case user.VerificationAttempts >= verification.MaxAttempts:
		httperr.Abort(c, http.StatusBadRequest, "Too many attempts. Please request a new code.")
		return
	}

	if !verification.Matches(*req.Code, *user.VerificationCodeHash) {
		user.VerificationAttempts++
		if _, err := h.Store.Save(ctx, user); err != nil {
			h.internalError(c, err)
			return
		}
		httperr.Abort(c, http.StatusUnauthorized, "Invalid code")
		return
	}

	store.ClearVerification(&user)
	saved, err := h.Store.Save(ctx, user)
	if err != nil {
		h.internalError(c, err)
		return
	}
	h.startSession(c, saved)
}

func (h *Credentials) ResendCode(c *gin.Context) {
	var req identifierRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	user, ok := h.pendingAccount(c, *req.Identifier)
	if !ok {
		return
	}
	if user.VerificationCodeSentAt != nil && h.now().Sub(*user.VerificationCodeSentAt) < verification.ResendCooldown {
		httperr.AbortPlain(c, http.StatusTooManyRequests, "Please wait before requesting another code")
		return
	}
	pending, ok := h.issueVerification(c, user)
	if !ok {
		return
	}
	respond.JSON(c, http.StatusOK, gin.H{"verification": pending})
}

// ForgotPassword emails a code for setting a new password. Only email
// accounts can: a phone account signs in through Telegram instead (422).
// An unknown email gets the same answer as a known one, so the form can't
// be used to find out who has an account.
func (h *Credentials) ForgotPassword(c *gin.Context) {
	var req identifierRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	ctx := c.Request.Context()
	identifier := strings.TrimSpace(*req.Identifier)
	sent := gin.H{"verification": PendingVerification{Identifier: identifier, Method: "email"}}
	if !strings.Contains(identifier, "@") {
		httperr.Abort(c, http.StatusUnprocessableEntity, "Password reset is only available for email accounts")
		return
	}
	user, err := h.Store.ByIdentifier(ctx, identifier)
	if errors.Is(err, store.ErrNotFound) || (err == nil && user.Email == nil) {
		respond.JSON(c, http.StatusOK, sent)
		return
	} else if err != nil {
		h.internalError(c, err)
		return
	}
	if user.VerificationCodeSentAt != nil && h.now().Sub(*user.VerificationCodeSentAt) < verification.ResendCooldown {
		httperr.AbortPlain(c, http.StatusTooManyRequests, "Please wait before requesting another code")
		return
	}
	code, err := verification.Code()
	if err != nil {
		h.internalError(c, err)
		return
	}
	store.SetVerification(&user, verification.Hash(code), verification.CodeTTL, h.now())
	if _, err := h.Store.Save(ctx, user); err != nil {
		h.internalError(c, err)
		return
	}
	if err := h.Email.SendVerificationCode(ctx, *user.Email, code); err != nil {
		h.Log.ErrorContext(ctx, "password reset email failed", "error", err)
		httperr.Abort(c, http.StatusInternalServerError, "Could not send verification email")
		return
	}
	respond.JSON(c, http.StatusOK, sent)
}

// ResetPassword sets a new password with the emailed code and signs in. The
// code proves the email, so an unverified account becomes verified.
func (h *Credentials) ResetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	ctx := c.Request.Context()
	user, err := h.Store.ByIdentifier(ctx, strings.TrimSpace(*req.Identifier))
	if errors.Is(err, store.ErrNotFound) {
		httperr.Abort(c, http.StatusUnauthorized, "Invalid code")
		return
	} else if err != nil {
		h.internalError(c, err)
		return
	}
	switch {
	case user.VerificationCodeHash == nil || user.VerificationCodeExpiresAt == nil:
		httperr.Abort(c, http.StatusBadRequest, "No verification code pending. Please request a new one.")
		return
	case user.VerificationCodeExpiresAt.Before(h.now()):
		httperr.Abort(c, http.StatusBadRequest, "Code has expired. Please request a new one.")
		return
	case user.VerificationAttempts >= verification.MaxAttempts:
		httperr.Abort(c, http.StatusBadRequest, "Too many attempts. Please request a new code.")
		return
	}
	if !verification.Matches(*req.Code, *user.VerificationCodeHash) {
		user.VerificationAttempts++
		if _, err := h.Store.Save(ctx, user); err != nil {
			h.internalError(c, err)
			return
		}
		httperr.Abort(c, http.StatusUnauthorized, "Invalid code")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), passwordCost)
	if err != nil {
		h.internalError(c, err)
		return
	}
	if err := h.Store.SetPassword(ctx, user.ID, string(hash)); err != nil {
		h.internalError(c, err)
		return
	}
	store.ClearVerification(&user)
	saved, err := h.Store.Save(ctx, user)
	if err != nil {
		h.internalError(c, err)
		return
	}
	h.startSession(c, saved)
}

// VerificationStatus is polled by the site while the Telegram bot confirms
// a phone registration.
func (h *Credentials) VerificationStatus(c *gin.Context) {
	identifier, present := c.GetQuery("identifier")
	var problem string
	switch {
	case !present:
		problem = "Required"
	case len([]rune(identifier)) < 1:
		problem = "String must contain at least 1 character(s)"
	case len([]rune(identifier)) > 255:
		problem = "String must contain at most 255 character(s)"
	}
	if problem != "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, validation.Flattened{
			FormErrors: []string{}, FieldErrors: map[string][]string{"identifier": {problem}},
		})
		return
	}
	user, err := h.Store.ByIdentifier(c.Request.Context(), identifier)
	if errors.Is(err, store.ErrNotFound) {
		httperr.Abort(c, http.StatusNotFound, "Account not found")
		return
	} else if err != nil {
		h.internalError(c, err)
		return
	}
	respond.JSON(c, http.StatusOK, gin.H{"verified": user.IsVerified})
}

// pendingAccount loads an account still awaiting verification, or aborts
// with NestJS's 404 / 409.
func (h *Credentials) pendingAccount(c *gin.Context, identifier string) (models.User, bool) {
	user, err := h.Store.ByIdentifier(c.Request.Context(), identifier)
	switch {
	case errors.Is(err, store.ErrNotFound):
		httperr.Abort(c, http.StatusNotFound, "Account not found")
		return user, false
	case err != nil:
		h.internalError(c, err)
		return user, false
	case user.IsVerified:
		httperr.Abort(c, http.StatusConflict, "Account is already verified")
		return user, false
	}
	return user, true
}

// issueVerification stores a fresh code and delivers it: emailed for email
// accounts; for phone accounts returned as a token for the Telegram deep
// link (the bot confirms it). Aborts and returns false on failure.
func (h *Credentials) issueVerification(c *gin.Context, user models.User) (PendingVerification, bool) {
	ctx := c.Request.Context()
	if user.Email != nil {
		code, err := verification.Code()
		if err != nil {
			h.internalError(c, err)
			return PendingVerification{}, false
		}
		store.SetVerification(&user, verification.Hash(code), verification.CodeTTL, h.now())
		if _, err := h.Store.Save(ctx, user); err != nil {
			h.internalError(c, err)
			return PendingVerification{}, false
		}
		if err := h.Email.SendVerificationCode(ctx, *user.Email, code); err != nil {
			message := "Could not send verification email"
			if errors.Is(err, email.ErrNotConfigured) {
				message = "Email service is not configured"
			}
			h.Log.ErrorContext(ctx, "verification email failed", "error", err)
			httperr.Abort(c, http.StatusInternalServerError, message)
			return PendingVerification{}, false
		}
		return PendingVerification{Identifier: *user.Email, Method: "email"}, true
	}

	token, err := verification.Token()
	if err != nil {
		h.internalError(c, err)
		return PendingVerification{}, false
	}
	store.SetVerification(&user, verification.Hash(token), verification.CodeTTL, h.now())
	if _, err := h.Store.Save(ctx, user); err != nil {
		h.internalError(c, err)
		return PendingVerification{}, false
	}
	return PendingVerification{Identifier: *user.Phone, Method: "telegram", Token: token}, true
}

func (h *Credentials) startSession(c *gin.Context, user models.User) {
	token, err := h.Signer.UserToken(user.ID)
	if err != nil {
		h.internalError(c, err)
		return
	}
	h.Cookies.SetUser(c, token)
	respond.JSON(c, http.StatusOK, gin.H{"user": user.Public()})
}
