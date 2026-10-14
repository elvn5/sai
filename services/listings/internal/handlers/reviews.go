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

const reviewNotFound = "Review not found"

type createReviewRequest struct {
	TargetType *string             `json:"targetType" binding:"required,oneof=hotel local_guide tour"`
	TargetID   *string             `json:"targetId" binding:"required,uuid"`
	Rating     *int                `json:"rating" binding:"required,gte=1,lte=5"`
	Text       *validation.Trimmed `json:"text" binding:"required,min=10,max=3000"`
	Photos     []string            `json:"photos" binding:"omitempty,max=6,dive,max=2000,mediaurl"`
}

type replyRequest struct {
	Text *validation.Trimmed `json:"text" binding:"required,min=1,max=2000"`
}

type Reviews struct{ Deps }

// targetQuery reads the targetType/targetId/limit/offset query shared by
// reviews and posts.
func targetQuery(c *gin.Context) (targetType, targetID string, limit, offset int, ok bool) {
	q := validation.NewQuery(c)
	limit = q.IntDefault("limit", 20, validation.Bound(1), validation.Bound(50))
	offset = q.IntDefault("offset", 0, validation.Bound(0), nil)
	targetType = q.RequiredEnum("targetType", models.TargetTypes)
	targetID = q.RequiredUUID("targetId")
	return targetType, targetID, limit, offset, q.OK()
}

func (h *Reviews) List(c *gin.Context) {
	targetType, targetID, limit, offset, ok := targetQuery(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	reviews, total, err := h.Store.Reviews(ctx, targetType, targetID, limit, offset)
	if err != nil {
		h.fail(c, err, reviewNotFound, "")
		return
	}
	summary, err := h.Store.ReviewSummary(ctx, targetType, targetID)
	if err != nil {
		h.fail(c, err, reviewNotFound, "")
		return
	}
	if err := h.attachReviewAuthors(c, reviews); err != nil {
		h.fail(c, err, reviewNotFound, "")
		return
	}
	respond.JSON(c, http.StatusOK, models.ReviewsPage{Items: reviews, Total: total, Summary: summary})
}

func (h *Reviews) Create(c *gin.Context) {
	var req createReviewRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	ctx := c.Request.Context()
	userID := currentUserID(c)

	owner, err := h.Store.TargetOwner(ctx, *req.TargetType, *req.TargetID)
	switch {
	case errors.Is(err, store.ErrTargetMissing):
		httperr.Abort(c, http.StatusNotFound, "Listing not found")
		return
	case err != nil:
		h.fail(c, err, reviewNotFound, "")
		return
	case ownedBy(owner, userID):
		httperr.Abort(c, http.StatusForbidden, "You cannot review your own listing")
		return
	}

	const alreadyReviewed = "You have already reviewed this listing"
	if done, err := h.Store.HasReviewed(ctx, *req.TargetType, *req.TargetID, userID); err != nil || done {
		h.fail(c, orDuplicate(err), reviewNotFound, alreadyReviewed)
		return
	}
	review, err := h.Store.InsertReview(ctx, *req.TargetType, *req.TargetID, userID, *req.Rating, string(*req.Text), req.Photos)
	if err != nil {
		h.fail(c, err, reviewNotFound, alreadyReviewed)
		return
	}
	if err := h.Store.RefreshRating(ctx, *req.TargetType, *req.TargetID); err != nil {
		h.fail(c, err, reviewNotFound, "")
		return
	}
	reviews := []models.Review{review}
	if err := h.attachReviewAuthors(c, reviews); err != nil {
		h.fail(c, err, reviewNotFound, "")
		return
	}
	respond.JSON(c, http.StatusCreated, reviews[0])
}

func (h *Reviews) Delete(c *gin.Context) {
	id := c.Param("id")
	if !validID(id) {
		h.fail(c, store.ErrNotFound, reviewNotFound, "")
		return
	}
	ctx := c.Request.Context()
	author, targetType, targetID, err := h.Store.ReviewOwner(ctx, id)
	if err != nil {
		h.fail(c, err, reviewNotFound, "")
		return
	}
	if author != currentUserID(c) {
		httperr.Abort(c, http.StatusForbidden, "You can only delete your own reviews")
		return
	}
	if err := h.Store.DeleteReview(ctx, id); err != nil {
		h.fail(c, err, reviewNotFound, "")
		return
	}
	if err := h.Store.RefreshRating(ctx, targetType, targetID); err != nil {
		h.fail(c, err, reviewNotFound, "")
		return
	}
	respond.Empty(c, http.StatusOK)
}

func (h *Reviews) attachReviewAuthors(c *gin.Context, reviews []models.Review) error {
	ids := make([]string, len(reviews))
	for i, r := range reviews {
		ids[i] = r.UserID
	}
	authors, err := h.Authors.Authors(c.Request.Context(), ids)
	if err != nil {
		return err
	}
	for i := range reviews {
		reviews[i].Author = authorFor(authors, reviews[i].UserID)
	}
	return nil
}

// Mine lists reviews of the signed-in partner's listings, newest first
// (?unanswered=true: only those without a reply yet).
func (h *Reviews) Mine(c *gin.Context) {
	q := validation.NewQuery(c)
	limit := q.IntDefault("limit", 20, validation.Bound(1), validation.Bound(50))
	offset := q.IntDefault("offset", 0, validation.Bound(0), nil)
	unanswered := q.Bool("unanswered")
	if !q.OK() {
		return
	}
	items, total, err := h.Store.ReviewsOfOwner(c.Request.Context(), currentUserID(c), unanswered != nil && *unanswered, limit, offset)
	if err != nil {
		h.fail(c, err, reviewNotFound, "")
		return
	}
	reviews := make([]models.Review, len(items))
	for i := range items {
		reviews[i] = items[i].Review
	}
	if err := h.attachReviewAuthors(c, reviews); err != nil {
		h.fail(c, err, reviewNotFound, "")
		return
	}
	for i := range items {
		items[i].Review = reviews[i]
	}
	respond.JSON(c, http.StatusOK, models.Page[models.PartnerReview]{Items: items, Total: total})
}

// Reply sets the listing owner's public answer to a review.
func (h *Reviews) Reply(c *gin.Context) {
	var req replyRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	text := string(*req.Text)
	h.setReply(c, &text)
}

// DeleteReply removes the owner's answer.
func (h *Reviews) DeleteReply(c *gin.Context) {
	h.setReply(c, nil)
}

func (h *Reviews) setReply(c *gin.Context, text *string) {
	id := c.Param("id")
	if !validID(id) {
		h.fail(c, store.ErrNotFound, reviewNotFound, "")
		return
	}
	ctx := c.Request.Context()
	owner, err := h.Store.ReviewTargetOwner(ctx, id)
	if err != nil {
		h.fail(c, err, reviewNotFound, "")
		return
	}
	if !ownedBy(owner, currentUserID(c)) {
		httperr.Abort(c, http.StatusForbidden, "Only the listing owner can reply to its reviews")
		return
	}
	review, err := h.Store.SetReviewReply(ctx, id, text)
	if err != nil {
		h.fail(c, err, reviewNotFound, "")
		return
	}
	reviews := []models.Review{review}
	if err := h.attachReviewAuthors(c, reviews); err != nil {
		h.fail(c, err, reviewNotFound, "")
		return
	}
	respond.JSON(c, http.StatusOK, reviews[0])
}
