package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/content/internal/models"
	"github.com/elvn5/sai/services/content/internal/store"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/jsontime"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/pkg/sanitize"
	"github.com/elvn5/sai/services/pkg/validation"
)

const postNotFound = "Post not found"

type createBlogPostRequest struct {
	Locale         *string                  `json:"locale" binding:"required,oneof=ru ky en de"`
	Slug           *string                  `json:"slug" binding:"required,min=1,max=255,slug"`
	Title          *string                  `json:"title" binding:"required,min=1,max=255"`
	Excerpt        validation.Field[string] `json:"excerpt" binding:"omitempty,max=500"`
	Content        *string                  `json:"content" binding:"required,min=1"`
	CoverImageURL  validation.Field[string] `json:"coverImageUrl" binding:"omitempty,max=2000"`
	SeoTitle       validation.Field[string] `json:"seoTitle" binding:"omitempty,max=255"`
	SeoDescription validation.Field[string] `json:"seoDescription" binding:"omitempty,max=500"`
	Difficulty     validation.Field[string] `json:"difficulty" binding:"omitempty,oneof=any easy moderate hard expert"`
	Duration       validation.Field[string] `json:"duration" binding:"omitempty,max=40"`
	Season         validation.Field[string] `json:"season" binding:"omitempty,oneof=spring summer autumn winter all_year"`
	IsPublished    *bool                    `json:"isPublished"`
}

type updateBlogPostRequest struct {
	Locale         *string                  `json:"locale" binding:"omitempty,oneof=ru ky en de"`
	Slug           *string                  `json:"slug" binding:"omitempty,min=1,max=255,slug"`
	Title          *string                  `json:"title" binding:"omitempty,min=1,max=255"`
	Excerpt        validation.Field[string] `json:"excerpt" binding:"omitempty,max=500"`
	Content        *string                  `json:"content" binding:"omitempty,min=1"`
	CoverImageURL  validation.Field[string] `json:"coverImageUrl" binding:"omitempty,max=2000"`
	SeoTitle       validation.Field[string] `json:"seoTitle" binding:"omitempty,max=255"`
	SeoDescription validation.Field[string] `json:"seoDescription" binding:"omitempty,max=500"`
	Difficulty     validation.Field[string] `json:"difficulty" binding:"omitempty,oneof=any easy moderate hard expert"`
	Duration       validation.Field[string] `json:"duration" binding:"omitempty,max=40"`
	Season         validation.Field[string] `json:"season" binding:"omitempty,oneof=spring summer autumn winter all_year"`
	IsPublished    *bool                    `json:"isPublished"`
}

// Blog serves /blog: public reading; the admin writes any post, approved
// partners write and manage their own.
type Blog struct {
	Store *store.Store
	Log   *slog.Logger
}

func (h *Blog) ListPublished(c *gin.Context) {
	locale, ok := optionalEnumQuery(c, "locale", models.Locales)
	if !ok {
		respond.JSON(c, http.StatusOK, []models.BlogPost{})
		return
	}
	posts, err := h.Store.PublishedBlogPosts(c.Request.Context(), locale)
	if err != nil {
		fail(c, h.Log, err, postNotFound)
		return
	}
	respond.JSON(c, http.StatusOK, posts)
}

func (h *Blog) ListAll(c *gin.Context) {
	locale, ok := optionalEnumQuery(c, "locale", models.Locales)
	if !ok {
		respond.JSON(c, http.StatusOK, []models.BlogPost{})
		return
	}
	posts, err := h.Store.AllBlogPosts(c.Request.Context(), locale)
	if err != nil {
		fail(c, h.Log, err, postNotFound)
		return
	}
	respond.JSON(c, http.StatusOK, posts)
}

// ListMine lists the signed-in partner's posts.
func (h *Blog) ListMine(c *gin.Context) {
	posts, err := h.Store.BlogPostsByUser(c.Request.Context(), currentUserID(c))
	if err != nil {
		fail(c, h.Log, err, postNotFound)
		return
	}
	respond.JSON(c, http.StatusOK, posts)
}

func (h *Blog) GetAdmin(c *gin.Context) {
	post, err := h.load(c, c.Param("id"))
	if err != nil {
		fail(c, h.Log, err, postNotFound)
		return
	}
	respond.JSON(c, http.StatusOK, post)
}

func (h *Blog) GetBySlug(c *gin.Context) {
	post, err := h.Store.PublishedBlogPostBySlug(c.Request.Context(), c.Param("locale"), c.Param("slug"))
	if err != nil {
		fail(c, h.Log, err, postNotFound)
		return
	}
	respond.JSON(c, http.StatusOK, post)
}

func (h *Blog) Create(c *gin.Context) {
	var req createBlogPostRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	ctx := c.Request.Context()
	if taken, err := h.Store.BlogSlugTaken(ctx, *req.Locale, *req.Slug, ""); err != nil || taken {
		fail(c, h.Log, orSlugTaken(err), postNotFound)
		return
	}

	post := models.BlogPost{
		Locale: *req.Locale, Slug: *req.Slug, Title: *req.Title, Content: *req.Content,
		Excerpt: req.Excerpt.Ptr(), CoverImageURL: req.CoverImageURL.Ptr(),
		SeoTitle: req.SeoTitle.Ptr(), SeoDescription: req.SeoDescription.Ptr(),
		Difficulty: req.Difficulty.Ptr(), Duration: req.Duration.Ptr(), Season: req.Season.Ptr(),
		IsPublished: req.IsPublished != nil && *req.IsPublished,
	}
	if !isAdmin(c) {
		// A partner's post: their HTML is untrusted, and it goes live
		// unless they save it as a draft.
		userID := currentUserID(c)
		post.UserID = &userID
		post.Content = sanitize.HTML(post.Content)
		post.IsPublished = req.IsPublished == nil || *req.IsPublished
	}
	if post.IsPublished {
		now := jsontime.New(time.Now())
		post.PublishedAt = &now
	}
	saved, err := h.Store.InsertBlogPost(ctx, post)
	if err != nil {
		fail(c, h.Log, err, postNotFound)
		return
	}
	respond.JSON(c, http.StatusCreated, saved)
}

func (h *Blog) Update(c *gin.Context) {
	var req updateBlogPostRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	ctx := c.Request.Context()
	post, ok := h.editable(c)
	if !ok {
		return
	}

	locale, slug := post.Locale, post.Slug
	if req.Locale != nil {
		locale = *req.Locale
	}
	if req.Slug != nil {
		slug = *req.Slug
	}
	if locale != post.Locale || slug != post.Slug {
		if taken, err := h.Store.BlogSlugTaken(ctx, locale, slug, post.ID); err != nil || taken {
			fail(c, h.Log, orSlugTaken(err), postNotFound)
			return
		}
	}

	wasPublished := post.IsPublished
	post.Locale, post.Slug = locale, slug
	if req.Title != nil {
		post.Title = *req.Title
	}
	if req.Content != nil {
		post.Content = *req.Content
		if !isAdmin(c) {
			post.Content = sanitize.HTML(post.Content)
		}
	}
	applyField(&post.Excerpt, req.Excerpt)
	applyField(&post.CoverImageURL, req.CoverImageURL)
	applyField(&post.SeoTitle, req.SeoTitle)
	applyField(&post.SeoDescription, req.SeoDescription)
	applyField(&post.Difficulty, req.Difficulty)
	applyField(&post.Duration, req.Duration)
	applyField(&post.Season, req.Season)
	if req.IsPublished != nil {
		post.IsPublished = *req.IsPublished
	}
	switch {
	case !post.IsPublished:
		post.PublishedAt = nil
	case !wasPublished:
		now := jsontime.New(time.Now())
		post.PublishedAt = &now
	}

	saved, err := h.Store.UpdateBlogPost(ctx, post)
	if err != nil {
		fail(c, h.Log, err, postNotFound)
		return
	}
	respond.JSON(c, http.StatusOK, saved)
}

func (h *Blog) Delete(c *gin.Context) {
	post, ok := h.editable(c)
	if !ok {
		return
	}
	if err := h.Store.DeleteBlogPost(c.Request.Context(), post.ID); err != nil {
		fail(c, h.Log, err, postNotFound)
		return
	}
	respond.Empty(c, http.StatusOK)
}

// editable loads the post in the path; a partner may only touch their own.
func (h *Blog) editable(c *gin.Context) (models.BlogPost, bool) {
	post, err := h.load(c, c.Param("id"))
	if err != nil {
		fail(c, h.Log, err, postNotFound)
		return models.BlogPost{}, false
	}
	if !isAdmin(c) && (post.UserID == nil || *post.UserID != currentUserID(c)) {
		httperr.Abort(c, http.StatusForbidden, "You can only manage your own posts")
		return models.BlogPost{}, false
	}
	return post, true
}

func (h *Blog) load(c *gin.Context, id string) (models.BlogPost, error) {
	if !validID(id) {
		return models.BlogPost{}, store.ErrNotFound
	}
	return h.Store.BlogPost(c.Request.Context(), id)
}

// orSlugTaken turns a successful "taken" check into ErrSlugTaken.
func orSlugTaken(err error) error {
	if err != nil {
		return err
	}
	return store.ErrSlugTaken
}
