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

const postNotFound = "Post not found"

type createPostRequest struct {
	TargetType *string             `json:"targetType" binding:"required,oneof=hotel local_guide tour"`
	TargetID   *string             `json:"targetId" binding:"required,uuid"`
	Text       *validation.Trimmed `json:"text" binding:"omitempty,max=3000"`
	Photos     []string            `json:"photos" binding:"omitempty,max=10,dive,max=2000,mediaurl"`
}

// Posts serves /listing-posts: news and photos a listing's owner shares.
type Posts struct{ Deps }

func (h *Posts) List(c *gin.Context) {
	targetType, targetID, limit, offset, ok := targetQuery(c)
	if !ok {
		return
	}
	posts, total, err := h.Store.Posts(c.Request.Context(), targetType, targetID, limit, offset)
	if err != nil {
		h.fail(c, err, postNotFound, "")
		return
	}
	if err := h.attachPostAuthors(c, posts); err != nil {
		h.fail(c, err, postNotFound, "")
		return
	}
	respond.JSON(c, http.StatusOK, models.Page[models.ListingPost]{Items: posts, Total: total})
}

func (h *Posts) Create(c *gin.Context) {
	var req createPostRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	text := ""
	if req.Text != nil {
		text = string(*req.Text)
	}
	// The schema's refine.
	if text == "" && len(req.Photos) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, validation.Flattened{
			FormErrors:  []string{},
			FieldErrors: map[string][]string{"text": {"A post needs text or at least one photo"}},
		})
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
		h.fail(c, err, postNotFound, "")
		return
	case !ownedBy(owner, userID):
		httperr.Abort(c, http.StatusForbidden, "Only the listing owner can post here")
		return
	}

	post, err := h.Store.InsertPost(ctx, *req.TargetType, *req.TargetID, userID, text, req.Photos)
	if err != nil {
		h.fail(c, err, postNotFound, "")
		return
	}
	posts := []models.ListingPost{post}
	if err := h.attachPostAuthors(c, posts); err != nil {
		h.fail(c, err, postNotFound, "")
		return
	}
	respond.JSON(c, http.StatusCreated, posts[0])
}

func (h *Posts) Delete(c *gin.Context) {
	id := c.Param("id")
	if !validID(id) {
		h.fail(c, store.ErrNotFound, postNotFound, "")
		return
	}
	ctx := c.Request.Context()
	author, err := h.Store.PostOwner(ctx, id)
	if err != nil {
		h.fail(c, err, postNotFound, "")
		return
	}
	if author != currentUserID(c) {
		httperr.Abort(c, http.StatusForbidden, "You can only delete your own posts")
		return
	}
	if err := h.Store.DeletePost(ctx, id); err != nil {
		h.fail(c, err, postNotFound, "")
		return
	}
	respond.Empty(c, http.StatusOK)
}

func (h *Posts) attachPostAuthors(c *gin.Context, posts []models.ListingPost) error {
	ids := make([]string, len(posts))
	for i, p := range posts {
		ids[i] = p.UserID
	}
	authors, err := h.Authors.Authors(c.Request.Context(), ids)
	if err != nil {
		return err
	}
	for i := range posts {
		posts[i].Author = authorFor(authors, posts[i].UserID)
	}
	return nil
}
