package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/validation"
	"github.com/elvn5/sai/services/users/internal/models"
	"github.com/elvn5/sai/services/users/internal/store"
	"github.com/elvn5/sai/services/users/internal/telegramauth"
)

const maxAuthAgeSeconds = 24 * 60 * 60

// widgetRequest is the Telegram Login Widget payload. Numbers are kept as
// their JSON text: the signature covers them as JavaScript prints them.
type widgetRequest struct {
	ID        *json.Number `json:"id" binding:"required"`
	FirstName *string      `json:"first_name" binding:"required"`
	LastName  *string      `json:"last_name"`
	Username  *string      `json:"username"`
	PhotoURL  *string      `json:"photo_url"`
	AuthDate  *json.Number `json:"auth_date" binding:"required"`
	Hash      *string      `json:"hash" binding:"required"`
}

type miniAppRequest struct {
	InitData *string `json:"initData" binding:"required,min=1"`
}

// TelegramProfile is the account data Telegram vouches for.
type TelegramProfile struct {
	TelegramID                              string
	Username, FirstName, LastName, PhotoURL *string
}

// Telegram serves /auth/telegram: the Login Widget on the site and the
// Mini App inside Telegram.
type Telegram struct{ Deps }

func (h *Telegram) Widget(c *gin.Context) {
	var req widgetRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	if h.BotToken == "" {
		httperr.Abort(c, http.StatusUnauthorized, "Bot token is not configured")
		return
	}

	// Only the fields the NestJS schema knew are signed (zod stripped the rest).
	fields := map[string]string{"id": jsNumber(*req.ID), "first_name": *req.FirstName, "auth_date": jsNumber(*req.AuthDate)}
	for k, v := range map[string]*string{"last_name": req.LastName, "username": req.Username, "photo_url": req.PhotoURL} {
		if v != nil {
			fields[k] = *v
		}
	}
	if !telegramauth.WidgetHashValid(fields, *req.Hash, h.BotToken) {
		httperr.Abort(c, http.StatusUnauthorized, "Invalid Telegram signature")
		return
	}
	authDate, _ := req.AuthDate.Float64()
	if !h.fresh(authDate) {
		httperr.Abort(c, http.StatusUnauthorized, "Telegram auth data has expired")
		return
	}
	h.signIn(c, TelegramProfile{
		TelegramID: jsNumber(*req.ID), Username: req.Username, FirstName: req.FirstName,
		LastName: req.LastName, PhotoURL: req.PhotoURL,
	})
}

func (h *Telegram) MiniApp(c *gin.Context) {
	var req miniAppRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	if h.BotToken == "" {
		httperr.Abort(c, http.StatusUnauthorized, "Bot token is not configured")
		return
	}
	params, ok := telegramauth.InitData(*req.InitData, h.BotToken)
	if !ok {
		httperr.Abort(c, http.StatusUnauthorized, "Invalid Telegram Mini App signature")
		return
	}
	authDate := math.NaN()
	if raw := params.Get("auth_date"); raw != "" {
		if n, err := strconv.ParseFloat(raw, 64); err == nil {
			authDate = n
		}
	}
	if !h.fresh(authDate) {
		httperr.Abort(c, http.StatusUnauthorized, "Telegram auth data has expired")
		return
	}
	rawUser := params.Get("user")
	if rawUser == "" {
		httperr.Abort(c, http.StatusUnauthorized, "Missing user in initData")
		return
	}
	var u struct {
		ID        json.Number `json:"id"`
		FirstName *string     `json:"first_name"`
		LastName  *string     `json:"last_name"`
		Username  *string     `json:"username"`
		PhotoURL  *string     `json:"photo_url"`
	}
	if err := json.Unmarshal([]byte(rawUser), &u); err != nil || u.ID == "" {
		httperr.Abort(c, http.StatusUnauthorized, "Malformed user in initData")
		return
	}
	h.signIn(c, TelegramProfile{
		TelegramID: jsNumber(u.ID), Username: u.Username, FirstName: u.FirstName,
		LastName: u.LastName, PhotoURL: u.PhotoURL,
	})
}

func (h *Telegram) fresh(authDate float64) bool {
	age := float64(h.now().UnixMilli())/1000 - authDate
	return !math.IsNaN(age) && !math.IsInf(age, 0) && age <= maxAuthAgeSeconds && age >= -60
}

func (h *Telegram) signIn(c *gin.Context, p TelegramProfile) {
	user, err := UpsertTelegramUser(c.Request.Context(), h.Store, p)
	if err != nil {
		h.internalError(c, err)
		return
	}
	(&Credentials{h.Deps}).startSession(c, user)
}

// UpsertTelegramUser finds the account for a Telegram user, creating it on
// first contact, and refreshes the provided profile fields. Shared with the
// bot's gRPC API.
func UpsertTelegramUser(ctx context.Context, s *store.Store, p TelegramProfile) (models.User, error) {
	existing, err := s.ByTelegramID(ctx, p.TelegramID)
	if errors.Is(err, store.ErrNotFound) {
		return s.Insert(ctx, models.User{
			TelegramID: &p.TelegramID, Username: p.Username, FirstName: p.FirstName,
			LastName: p.LastName, AvatarURL: p.PhotoURL, IsVerified: true,
		})
	}
	if err != nil {
		return models.User{}, err
	}
	keepUnlessSet(&existing.Username, p.Username)
	keepUnlessSet(&existing.FirstName, p.FirstName)
	keepUnlessSet(&existing.LastName, p.LastName)
	keepUnlessSet(&existing.AvatarURL, p.PhotoURL)
	return s.Save(ctx, existing)
}

func keepUnlessSet(dst **string, v *string) {
	if v != nil {
		*dst = v
	}
}

// jsNumber prints a JSON number as JavaScript's String(n) would for the
// integers Telegram sends (no exponent, no trailing ".0").
func jsNumber(n json.Number) string {
	if i, err := n.Int64(); err == nil {
		return strconv.FormatInt(i, 10)
	}
	f, err := n.Float64()
	if err != nil {
		return n.String()
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}
