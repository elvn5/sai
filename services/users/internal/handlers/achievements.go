package handlers

import (
	"errors"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/pkg/validation"
	"github.com/elvn5/sai/services/users/internal/models"
	"github.com/elvn5/sai/services/users/internal/store"
)

const (
	achievementNotFound = "Achievement not found"
	maxTitleLength      = 80
	maxDescLength       = 300
	hallOfFameSize      = 50
	recentAwardsSize    = 12
	givenAwardsSize     = 100
	adminAwardsSize     = 200
)

var achievementLocales = []string{"ru", "ky", "en", "de"}

// Iconify names as the Nuxt apps use them, e.g. "i-mdi-trophy".
var iconPattern = regexp.MustCompile(`^i-[a-z0-9]+(?:-[a-z0-9]+)+$`)

func init() {
	validation.RegisterPattern("iconname", iconPattern, "Icon must be an icon name like i-mdi-trophy")
}

type achievementRequest struct {
	Title       map[string]string `json:"title"`
	Description map[string]string `json:"description"`
	Icon        *string           `json:"icon" binding:"omitempty,iconname,max=100"`
	Rarity      *string           `json:"rarity" binding:"omitempty,oneof=common rare epic legendary"`
	IsActive    *bool             `json:"isActive"`
	SortOrder   *int              `json:"sortOrder" binding:"omitempty,min=0,max=100000"`
	creating    bool
}

// Refine checks the locale maps, which the validator tags can't describe
// with readable messages.
func (r achievementRequest) Refine() map[string]string {
	errs := map[string]string{}
	if r.creating && r.Icon == nil {
		errs["icon"] = "Required"
	}
	if r.Title != nil || r.creating {
		if msg := localizedTextError(r.Title, maxTitleLength, true); msg != "" {
			errs["title"] = msg
		}
	}
	if msg := localizedTextError(r.Description, maxDescLength, false); msg != "" {
		errs["description"] = msg
	}
	return errs
}

func localizedTextError(texts map[string]string, maxLen int, ruRequired bool) string {
	if ruRequired && strings.TrimSpace(texts["ru"]) == "" {
		return "Russian text is required"
	}
	for locale, text := range texts {
		if !slices.Contains(achievementLocales, locale) {
			return "Unknown locale: " + locale
		}
		if utf8.RuneCountInString(text) > maxLen {
			return "Must be at most " + strconv.Itoa(maxLen) + " characters"
		}
	}
	return ""
}

func trimTexts(texts map[string]string) map[string]string {
	out := make(map[string]string, len(texts))
	for locale, text := range texts {
		if text = strings.TrimSpace(text); text != "" {
			out[locale] = text
		}
	}
	return out
}

type awardRequest struct {
	Code          *string                  `json:"code" binding:"required,min=1,max=32"`
	AchievementID *string                  `json:"achievementId" binding:"required,uuid"`
	Note          validation.Field[string] `json:"note" binding:"omitempty,max=300"`
}

// Achievements serves the achievement catalog (admin-managed), awards
// (partners hand them to users by the user's code) and the hall of fame.
type Achievements struct{ Deps }

// Catalog lists the active achievements, for the site.
func (h *Achievements) Catalog(c *gin.Context) {
	list, err := h.Store.Achievements(c.Request.Context(), true)
	if err != nil {
		h.internalError(c, err)
		return
	}
	respond.JSON(c, http.StatusOK, list)
}

// HallOfFame is the public leaderboard plus the latest awards.
func (h *Achievements) HallOfFame(c *gin.Context) {
	ctx := c.Request.Context()
	leaders, err := h.Store.HallOfFame(ctx, hallOfFameSize)
	if err != nil {
		h.internalError(c, err)
		return
	}
	recent, err := h.Store.RecentAwards(ctx, recentAwardsSize)
	if err != nil {
		h.internalError(c, err)
		return
	}
	for i := range recent {
		recent[i].Note, recent[i].IsNew = nil, false
	}
	respond.JSON(c, http.StatusOK, gin.H{"leaders": leaders, "recent": recent})
}

// Mine is the signed-in user's trophy shelf and award code.
func (h *Achievements) Mine(c *gin.Context) {
	ctx := c.Request.Context()
	userID := currentUserID(c)
	code, err := h.Store.AwardCode(ctx, userID)
	if err != nil {
		h.internalError(c, err)
		return
	}
	awards, err := h.Store.AwardsOf(ctx, userID)
	if err != nil {
		h.internalError(c, err)
		return
	}
	points := 0
	for _, a := range awards {
		points += a.Achievement.Points
	}
	respond.JSON(c, http.StatusOK, gin.H{"code": code, "points": points, "awards": awards})
}

// MarkSeen clears the "new" badges once the user has seen them.
func (h *Achievements) MarkSeen(c *gin.Context) {
	if err := h.Store.MarkAwardsSeen(c.Request.Context(), currentUserID(c)); err != nil {
		h.internalError(c, err)
		return
	}
	respond.Empty(c, http.StatusOK)
}

// Recipient shows a partner whose code they entered, and what that user
// already has.
func (h *Achievements) Recipient(c *gin.Context) {
	user, ok := h.recipient(c, c.Param("code"))
	if !ok {
		return
	}
	earned, err := h.Store.EarnedAchievementIDs(c.Request.Context(), user.ID)
	if err != nil {
		h.internalError(c, err)
		return
	}
	respond.JSON(c, http.StatusOK, gin.H{"user": person(user), "earnedAchievementIds": earned})
}

// Award hands an achievement to the user behind a code.
func (h *Achievements) Award(c *gin.Context) {
	var req awardRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	user, ok := h.recipient(c, *req.Code)
	if !ok {
		return
	}
	partnerID := currentUserID(c)
	if user.ID == partnerID {
		httperr.Abort(c, http.StatusBadRequest, "You can't award yourself")
		return
	}
	ctx := c.Request.Context()
	achievement, err := h.Store.Achievement(ctx, *req.AchievementID)
	if errors.Is(err, store.ErrNotFound) || (err == nil && !achievement.IsActive) {
		httperr.Abort(c, http.StatusNotFound, achievementNotFound)
		return
	}
	if err != nil {
		h.internalError(c, err)
		return
	}
	award, err := h.Store.Award(ctx, user.ID, achievement.ID, partnerID, req.Note.Ptr())
	if errors.Is(err, store.ErrDuplicate) {
		httperr.Abort(c, http.StatusConflict, "This traveler already has this achievement")
		return
	}
	if err != nil {
		h.internalError(c, err)
		return
	}
	respond.JSON(c, http.StatusCreated, award)
}

// Given lists the awards the signed-in partner has handed out.
func (h *Achievements) Given(c *gin.Context) {
	awards, err := h.Store.AwardsGivenBy(c.Request.Context(), currentUserID(c), givenAwardsSize)
	if err != nil {
		h.internalError(c, err)
		return
	}
	respond.JSON(c, http.StatusOK, awards)
}

// AdminList lists the whole catalog, archived entries included.
func (h *Achievements) AdminList(c *gin.Context) {
	list, err := h.Store.Achievements(c.Request.Context(), false)
	if err != nil {
		h.internalError(c, err)
		return
	}
	respond.JSON(c, http.StatusOK, list)
}

func (h *Achievements) Create(c *gin.Context) {
	req := achievementRequest{creating: true}
	if !validation.BindJSON(c, &req) {
		return
	}
	a := models.Achievement{Rarity: "common", IsActive: true, Description: map[string]string{}}
	req.apply(&a)
	saved, err := h.Store.InsertAchievement(c.Request.Context(), a)
	if err != nil {
		h.internalError(c, err)
		return
	}
	respond.JSON(c, http.StatusCreated, saved)
}

func (h *Achievements) Update(c *gin.Context) {
	var req achievementRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	ctx := c.Request.Context()
	a, err := h.Store.Achievement(ctx, c.Param("id"))
	if errors.Is(err, store.ErrNotFound) {
		httperr.Abort(c, http.StatusNotFound, achievementNotFound)
		return
	}
	if err != nil {
		h.internalError(c, err)
		return
	}
	req.apply(&a)
	saved, err := h.Store.UpdateAchievement(ctx, a)
	if err != nil {
		h.internalError(c, err)
		return
	}
	respond.JSON(c, http.StatusOK, saved)
}

func (h *Achievements) Delete(c *gin.Context) {
	switch err := h.Store.DeleteAchievement(c.Request.Context(), c.Param("id")); {
	case errors.Is(err, store.ErrNotFound):
		httperr.Abort(c, http.StatusNotFound, achievementNotFound)
	case errors.Is(err, store.ErrInUse):
		httperr.Abort(c, http.StatusConflict, "This achievement has been awarded; archive it instead")
	case err != nil:
		h.internalError(c, err)
	default:
		respond.Empty(c, http.StatusOK)
	}
}

// AdminAwards lists the latest awards, for moderation.
func (h *Achievements) AdminAwards(c *gin.Context) {
	awards, err := h.Store.RecentAwards(c.Request.Context(), adminAwardsSize)
	if err != nil {
		h.internalError(c, err)
		return
	}
	respond.JSON(c, http.StatusOK, awards)
}

// Revoke takes an award back.
func (h *Achievements) Revoke(c *gin.Context) {
	switch err := h.Store.RevokeAward(c.Request.Context(), c.Param("id")); {
	case errors.Is(err, store.ErrNotFound):
		httperr.Abort(c, http.StatusNotFound, "Award not found")
	case err != nil:
		h.internalError(c, err)
	default:
		respond.Empty(c, http.StatusOK)
	}
}

func (r achievementRequest) apply(a *models.Achievement) {
	if r.Title != nil {
		a.Title = trimTexts(r.Title)
	}
	if r.Description != nil {
		a.Description = trimTexts(r.Description)
	}
	if r.Icon != nil {
		a.Icon = *r.Icon
	}
	if r.Rarity != nil {
		a.Rarity = *r.Rarity
	}
	if r.IsActive != nil {
		a.IsActive = *r.IsActive
	}
	if r.SortOrder != nil {
		a.SortOrder = *r.SortOrder
	}
}

// recipient finds the user behind an award code. Codes are typed by hand,
// so case, spaces and dashes are ignored.
func (h *Achievements) recipient(c *gin.Context, raw string) (models.User, bool) {
	code := normalizeAwardCode(raw)
	user, err := h.Store.ByAwardCode(c.Request.Context(), code)
	if errors.Is(err, store.ErrNotFound) || (err == nil && len(code) != store.AwardCodeLength) {
		httperr.Abort(c, http.StatusNotFound, "No traveler with this code")
		return models.User{}, false
	}
	if err != nil {
		h.internalError(c, err)
		return models.User{}, false
	}
	return user, true
}

func normalizeAwardCode(raw string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == ' ' || r == '-':
			return -1
		case r >= 'a' && r <= 'z':
			return r - 'a' + 'A'
		}
		return r
	}, raw)
}

func person(u models.User) models.Person {
	return models.Person{ID: u.ID, FirstName: u.FirstName, Username: u.Username, AvatarURL: u.AvatarURL}
}
