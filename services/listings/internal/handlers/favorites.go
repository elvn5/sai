package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/listings/internal/models"
	"github.com/elvn5/sai/services/listings/internal/store"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/pkg/validation"
)

const favoriteNotFound = "Listing not found"

// Favorites serves /favorites: listings a traveler saved.
type Favorites struct{ Deps }

// IDs lists the saved ids by type, for the hearts on cards.
func (h *Favorites) IDs(c *gin.Context) {
	ids, err := h.Store.FavoriteIDs(c.Request.Context(), currentUserID(c))
	if err != nil {
		h.fail(c, err, favoriteNotFound, "")
		return
	}
	respond.JSON(c, http.StatusOK, ids)
}

// Mine lists the saved listings themselves (published ones).
func (h *Favorites) Mine(c *gin.Context) {
	ctx, userID := c.Request.Context(), currentUserID(c)
	tours, err := h.Store.FavoriteTours(ctx, userID)
	if err != nil {
		h.fail(c, err, favoriteNotFound, "")
		return
	}
	hotels, err := h.Store.FavoriteHotels(ctx, userID)
	if err != nil {
		h.fail(c, err, favoriteNotFound, "")
		return
	}
	guides, err := h.Store.FavoriteLocalGuides(ctx, userID)
	if err != nil {
		h.fail(c, err, favoriteNotFound, "")
		return
	}
	respond.JSON(c, http.StatusOK, gin.H{
		"tours": nonNilList(tours), "hotels": nonNilList(hotels), "localGuides": nonNilList(guides),
	})
}

// Add saves a published listing; Remove forgets it. Both are idempotent.
func (h *Favorites) Add(c *gin.Context) {
	targetType, targetID, ok := h.target(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	if _, err := h.Store.TargetOwner(ctx, targetType, targetID); errors.Is(err, store.ErrTargetMissing) {
		httperr.Abort(c, http.StatusNotFound, favoriteNotFound)
		return
	} else if err != nil {
		h.fail(c, err, favoriteNotFound, "")
		return
	}
	if err := h.Store.AddFavorite(ctx, currentUserID(c), targetType, targetID); err != nil {
		h.fail(c, err, favoriteNotFound, "")
		return
	}
	respond.JSON(c, http.StatusOK, gin.H{"saved": true})
}

func (h *Favorites) Remove(c *gin.Context) {
	targetType, targetID, ok := h.target(c)
	if !ok {
		return
	}
	if err := h.Store.RemoveFavorite(c.Request.Context(), currentUserID(c), targetType, targetID); err != nil {
		h.fail(c, err, favoriteNotFound, "")
		return
	}
	respond.JSON(c, http.StatusOK, gin.H{"saved": false})
}

func (h *Favorites) target(c *gin.Context) (string, string, bool) {
	targetType, targetID := c.Param("type"), c.Param("id")
	if !isTargetType(targetType) || !validID(targetID) {
		httperr.Abort(c, http.StatusNotFound, favoriteNotFound)
		return "", "", false
	}
	return targetType, targetID, true
}

func isTargetType(t string) bool {
	return t == "hotel" || t == "local_guide" || t == "tour"
}

func nonNilList[T any](items []T) []T {
	if items == nil {
		return []T{}
	}
	return items
}

// TourAlerts serves /tour-alerts: "tell me about new tours" by Telegram.
type TourAlerts struct{ Deps }

type tourAlertRequest struct {
	// A destination, or "" for any.
	Destination *string `json:"destination" binding:"required"`
}

func (h *TourAlerts) Mine(c *gin.Context) {
	destinations, err := h.Store.TourAlerts(c.Request.Context(), currentUserID(c))
	if err != nil {
		h.fail(c, err, favoriteNotFound, "")
		return
	}
	respond.JSON(c, http.StatusOK, gin.H{"destinations": nonNilList(destinations)})
}

// Add follows a destination. It needs the bot (503 without it) and the
// visitor's Telegram linked to the account (409 otherwise).
func (h *TourAlerts) Add(c *gin.Context) {
	destination, ok := h.destination(c)
	if !ok {
		return
	}
	if h.Alerts == nil {
		httperr.Abort(c, http.StatusServiceUnavailable, "Telegram notifications are off")
		return
	}
	ctx, userID := c.Request.Context(), currentUserID(c)
	linked, err := h.Alerts.Linked(ctx, userID)
	if err != nil {
		h.fail(c, err, favoriteNotFound, "")
		return
	}
	if !linked {
		httperr.Abort(c, http.StatusConflict, "Link Telegram to your account first")
		return
	}
	if err := h.Store.AddTourAlert(ctx, userID, destination); err != nil {
		h.fail(c, err, favoriteNotFound, "")
		return
	}
	h.Mine(c)
}

func (h *TourAlerts) Remove(c *gin.Context) {
	destination, ok := h.destination(c)
	if !ok {
		return
	}
	if err := h.Store.RemoveTourAlert(c.Request.Context(), currentUserID(c), destination); err != nil {
		h.fail(c, err, favoriteNotFound, "")
		return
	}
	h.Mine(c)
}

func (h *TourAlerts) destination(c *gin.Context) (string, bool) {
	var req tourAlertRequest
	if !validation.BindJSON(c, &req) {
		return "", false
	}
	if *req.Destination != "" && !isDestination(*req.Destination) {
		c.AbortWithStatusJSON(http.StatusBadRequest, validation.Flattened{
			FormErrors: []string{}, FieldErrors: map[string][]string{"destination": {"Invalid destination"}},
		})
		return "", false
	}
	return *req.Destination, true
}

func isDestination(d string) bool {
	for _, known := range models.Destinations {
		if known == d {
			return true
		}
	}
	return false
}
