package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/listings/internal/models"
	"github.com/elvn5/sai/services/listings/internal/store"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/pkg/sanitize"
	"github.com/elvn5/sai/services/pkg/validation"
)

const (
	hotelNotFound = "Hotel not found"
	slugTaken     = "Slug is already taken"
)

// hotelRequest is updateHotelSchema (every field optional).
type hotelRequest struct {
	Slug               *string                   `json:"slug" binding:"omitempty,min=1,max=255,slug"`
	Name               *validation.Trimmed       `json:"name" binding:"omitempty,min=1,max=255"`
	Type               *string                   `json:"type" binding:"omitempty,oneof=hotel guesthouse hostel yurt_camp resort apartment"`
	Description        *string                   `json:"description" binding:"omitempty,min=1,max=20000"`
	Destination        *string                   `json:"destination" binding:"omitempty,oneof=issyk_kul karakol bishkek ala_archa song_kol naryn tash_rabat kel_suu osh arslanbob sary_chelek talas batken chuy"`
	Address            validation.Field[string]  `json:"address" binding:"omitempty,max=255"`
	Latitude           validation.Field[float64] `json:"latitude" binding:"omitempty,gte=-90,lte=90"`
	Longitude          validation.Field[float64] `json:"longitude" binding:"omitempty,gte=-180,lte=180"`
	PriceFrom          validation.Field[int]     `json:"priceFrom" binding:"omitempty,gte=0"`
	Stars              validation.Field[int]     `json:"stars" binding:"omitempty,gte=1,lte=5"`
	Amenities          []string                  `json:"amenities" binding:"omitempty,max=16,dive,oneof=wifi breakfast parking airport_transfer hot_water heating kitchen restaurant sauna pool beach laundry family_friendly pets_allowed tours_desk horse_riding"`
	CancellationPolicy *string                   `json:"cancellationPolicy" binding:"omitempty,oneof=free flexible strict non_refundable"`
	CheckInTime        validation.Field[string]  `json:"checkInTime" binding:"omitempty,hhmm"`
	CheckOutTime       validation.Field[string]  `json:"checkOutTime" binding:"omitempty,hhmm"`
	CoverImageURL      validation.Field[string]  `json:"coverImageUrl" binding:"omitempty,max=2000,mediaurl"`
	Photos             []string                  `json:"photos" binding:"omitempty,max=20,dive,max=2000,mediaurl"`
	Website            validation.Field[string]  `json:"website" binding:"omitempty,url,max=500"`
	Phone              validation.Field[string]  `json:"phone" binding:"omitempty,max=32,phone"`
	Whatsapp           validation.Field[string]  `json:"whatsapp" binding:"omitempty,max=32,phone"`
	Telegram           validation.Field[string]  `json:"telegram" binding:"omitempty,max=33,telegram"`
}

// createHotelRequest is createHotelSchema: hotelRequest with the required
// fields marked.
type createHotelRequest struct {
	Slug               *string                   `json:"slug" binding:"required,min=1,max=255,slug"`
	Name               *validation.Trimmed       `json:"name" binding:"required,min=1,max=255"`
	Type               *string                   `json:"type" binding:"required,oneof=hotel guesthouse hostel yurt_camp resort apartment"`
	Description        *string                   `json:"description" binding:"required,min=1,max=20000"`
	Destination        *string                   `json:"destination" binding:"required,oneof=issyk_kul karakol bishkek ala_archa song_kol naryn tash_rabat kel_suu osh arslanbob sary_chelek talas batken chuy"`
	Address            validation.Field[string]  `json:"address" binding:"omitempty,max=255"`
	Latitude           validation.Field[float64] `json:"latitude" binding:"omitempty,gte=-90,lte=90"`
	Longitude          validation.Field[float64] `json:"longitude" binding:"omitempty,gte=-180,lte=180"`
	PriceFrom          validation.Field[int]     `json:"priceFrom" binding:"omitempty,gte=0"`
	Stars              validation.Field[int]     `json:"stars" binding:"omitempty,gte=1,lte=5"`
	Amenities          []string                  `json:"amenities" binding:"omitempty,max=16,dive,oneof=wifi breakfast parking airport_transfer hot_water heating kitchen restaurant sauna pool beach laundry family_friendly pets_allowed tours_desk horse_riding"`
	CancellationPolicy *string                   `json:"cancellationPolicy" binding:"required,oneof=free flexible strict non_refundable"`
	CheckInTime        validation.Field[string]  `json:"checkInTime" binding:"omitempty,hhmm"`
	CheckOutTime       validation.Field[string]  `json:"checkOutTime" binding:"omitempty,hhmm"`
	CoverImageURL      validation.Field[string]  `json:"coverImageUrl" binding:"omitempty,max=2000,mediaurl"`
	Photos             []string                  `json:"photos" binding:"omitempty,max=20,dive,max=2000,mediaurl"`
	Website            validation.Field[string]  `json:"website" binding:"omitempty,url,max=500"`
	Phone              validation.Field[string]  `json:"phone" binding:"omitempty,max=32,phone"`
	Whatsapp           validation.Field[string]  `json:"whatsapp" binding:"omitempty,max=32,phone"`
	Telegram           validation.Field[string]  `json:"telegram" binding:"omitempty,max=33,telegram"`
}

func (r createHotelRequest) asUpdate() hotelRequest {
	// Same fields, different validation tags.
	return hotelRequest(r)
}

// apply copies the fields present in the request onto h.
func (r hotelRequest) apply(h *models.Hotel) {
	if r.Slug != nil {
		h.Slug = *r.Slug
	}
	if r.Name != nil {
		h.Name = string(*r.Name)
	}
	if r.Type != nil {
		h.Type = *r.Type
	}
	if r.Description != nil {
		h.Description = sanitize.HTML(*r.Description)
	}
	if r.Destination != nil {
		h.Destination = *r.Destination
	}
	if r.CancellationPolicy != nil {
		h.CancellationPolicy = *r.CancellationPolicy
	}
	if r.Amenities != nil {
		h.Amenities = r.Amenities
	}
	if r.Photos != nil {
		h.Photos = r.Photos
	}
	applyField(&h.Address, r.Address)
	applyField(&h.Latitude, r.Latitude)
	applyField(&h.Longitude, r.Longitude)
	applyField(&h.PriceFrom, r.PriceFrom)
	applyField(&h.Stars, r.Stars)
	applyField(&h.CheckInTime, r.CheckInTime)
	applyField(&h.CheckOutTime, r.CheckOutTime)
	applyField(&h.CoverImageURL, r.CoverImageURL)
	applyField(&h.Website, r.Website)
	contactFields{r.Phone, r.Whatsapp, r.Telegram}.apply(&h.Phone, &h.Whatsapp, &h.Telegram)
}

type Hotels struct{ Deps }

func (h *Hotels) List(c *gin.Context) {
	q := validation.NewQuery(c)
	f := store.HotelFilter{
		Limit:            q.IntDefault("limit", 20, validation.Bound(1), validation.Bound(50)),
		Offset:           q.IntDefault("offset", 0, validation.Bound(0), nil),
		Destination:      q.Enum("destination", models.Destinations),
		Type:             q.Enum("type", models.HotelTypes),
		Amenities:        q.List("amenities", models.HotelAmenities),
		Search:           q.String("search", 100),
		Sort:             q.EnumDefault("sort", models.HotelSorts, "rating"),
		FreeCancellation: boolValue(q.Bool("freeCancellation")),
	}
	if n := q.Number("maxPrice", validation.NumberRule{Integer: true, Min: validation.Bound(0)}); n != nil {
		v := int(*n)
		f.MaxPrice = &v
	}
	f.MinRating = q.Number("minRating", validation.NumberRule{Min: validation.Bound(0), Max: validation.Bound(5)})
	if !q.OK() {
		return
	}

	page, err := h.Store.PublishedHotels(c.Request.Context(), f)
	if err != nil {
		h.fail(c, err, hotelNotFound, "")
		return
	}
	respond.JSON(c, http.StatusOK, page)
}

func (h *Hotels) ListMine(c *gin.Context) {
	hotels, err := h.Store.HotelsByUser(c.Request.Context(), currentUserID(c))
	if err != nil {
		h.fail(c, err, hotelNotFound, "")
		return
	}
	respond.JSON(c, http.StatusOK, hotels)
}

func (h *Hotels) GetBySlug(c *gin.Context) {
	hotel, err := h.Store.PublishedHotelBySlug(c.Request.Context(), c.Param("slug"))
	if err != nil {
		h.fail(c, err, hotelNotFound, "")
		return
	}
	respond.JSON(c, http.StatusOK, hotel)
}

func (h *Hotels) Create(c *gin.Context) {
	var req createHotelRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	ctx := c.Request.Context()
	if taken, err := h.Store.HotelSlugTaken(ctx, *req.Slug, ""); err != nil || taken {
		h.fail(c, orDuplicate(err), hotelNotFound, slugTaken)
		return
	}
	userID := currentUserID(c)
	hotel := models.Hotel{UserID: &userID, IsPublished: true, Amenities: []string{}, Photos: []string{}}
	req.asUpdate().apply(&hotel)

	saved, err := h.Store.SaveHotel(ctx, hotel)
	if err != nil {
		h.fail(c, err, hotelNotFound, slugTaken)
		return
	}
	respond.JSON(c, http.StatusCreated, saved)
}

func (h *Hotels) Update(c *gin.Context) {
	var req hotelRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	hotel, ok := h.owned(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	if req.Slug != nil && *req.Slug != hotel.Slug {
		if taken, err := h.Store.HotelSlugTaken(ctx, *req.Slug, hotel.ID); err != nil || taken {
			h.fail(c, orDuplicate(err), hotelNotFound, slugTaken)
			return
		}
	}
	req.apply(&hotel)

	saved, err := h.Store.SaveHotel(ctx, hotel)
	if err != nil {
		h.fail(c, err, hotelNotFound, slugTaken)
		return
	}
	respond.JSON(c, http.StatusOK, saved)
}

func (h *Hotels) Delete(c *gin.Context) {
	hotel, ok := h.owned(c)
	if !ok {
		return
	}
	if err := h.Store.DeleteListing(c.Request.Context(), "hotel", hotel.ID); err != nil {
		h.fail(c, err, hotelNotFound, "")
		return
	}
	respond.Empty(c, http.StatusOK)
}

func (h *Hotels) owned(c *gin.Context) (models.Hotel, bool) {
	id := c.Param("id")
	if !validID(id) {
		h.fail(c, store.ErrNotFound, hotelNotFound, "")
		return models.Hotel{}, false
	}
	hotel, err := h.Store.Hotel(c.Request.Context(), id)
	if err != nil {
		h.fail(c, err, hotelNotFound, "")
		return models.Hotel{}, false
	}
	if !ownedBy(hotel.UserID, currentUserID(c)) {
		httperr.Abort(c, http.StatusForbidden, "You can only manage your own hotels")
		return models.Hotel{}, false
	}
	return hotel, true
}

// orDuplicate turns a successful "taken" check into ErrDuplicate.
func orDuplicate(err error) error {
	if err != nil {
		return err
	}
	return store.ErrDuplicate
}

func boolValue(b *bool) bool { return b != nil && *b }
