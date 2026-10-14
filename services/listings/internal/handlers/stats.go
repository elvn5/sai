package handlers

import (
	"crypto/sha256"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/listings/internal/models"
	"github.com/elvn5/sai/services/listings/internal/store"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/pkg/validation"
)

// Bishkek: stats are counted per Bishkek calendar day.
var bishkek = time.FixedZone("Asia/Bishkek", 6*60*60)

// crawlers don't count as visitors.
var crawlers = regexp.MustCompile(`(?i)bot|crawl|spider|slurp|preview|headless|lighthouse`)

// maxSeen caps the dedup memory; past it the day starts over (a few
// repeats get counted rather than the process growing).
const maxSeen = 200_000

type recordEventRequest struct {
	TargetType *string `json:"targetType" binding:"required,oneof=hotel local_guide tour"`
	TargetID   *string `json:"targetId" binding:"required,uuid"`
	Event      *string `json:"event" binding:"required,oneof=view phone whatsapp telegram website"`
}

// Stats counts visits to listings and clicks on their contact buttons,
// and reports them to the listing owners.
type Stats struct {
	Deps
	// Now is overridable in tests.
	Now func() time.Time

	mu      sync.Mutex
	seenDay string
	// Visitor+listing+event hashes counted today: one of each per day.
	seen map[[32]byte]struct{}
}

func (h *Stats) now() time.Time {
	if h.Now != nil {
		return h.Now().In(bishkek)
	}
	return time.Now().In(bishkek)
}

// firstToday reports whether key wasn't seen yet today, and remembers it.
func (h *Stats) firstToday(day string, key [32]byte) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.seenDay != day || len(h.seen) >= maxSeen {
		h.seenDay, h.seen = day, map[[32]byte]struct{}{}
	}
	if _, ok := h.seen[key]; ok {
		return false
	}
	h.seen[key] = struct{}{}
	return true
}

// Record counts a view or a contact click. It always answers 204 for a
// valid body: the browser fires it and forgets. Owners' own visits,
// crawlers, unpublished listings and repeats on the same day don't count.
func (h *Stats) Record(c *gin.Context) {
	var req recordEventRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	defer respond.Empty(c, http.StatusNoContent)
	if crawlers.MatchString(c.Request.UserAgent()) {
		return
	}
	ctx := c.Request.Context()
	owner, err := h.Store.TargetOwner(ctx, *req.TargetType, *req.TargetID)
	if errors.Is(err, store.ErrTargetMissing) {
		return
	}
	if err != nil {
		h.Log.ErrorContext(ctx, "stats: owner lookup failed", slog.Any("error", err))
		return
	}
	if ownedBy(owner, currentUserID(c)) {
		return
	}
	now := h.now()
	day := now.Format(time.DateOnly)
	key := sha256.Sum256([]byte(c.ClientIP() + "\x00" + c.Request.UserAgent() + "\x00" + currentUserID(c) +
		"\x00" + *req.TargetType + "\x00" + *req.TargetID + "\x00" + *req.Event))
	if !h.firstToday(day, key) {
		return
	}
	if err := h.Store.RecordEvent(ctx, *req.TargetType, *req.TargetID, *req.Event, now); err != nil {
		h.Log.ErrorContext(ctx, "stats: record failed", slog.Any("error", err))
	}
}

// Mine reports the signed-in partner's listings stats over the last
// `days` days (today included) and the period before.
func (h *Stats) Mine(c *gin.Context) {
	q := validation.NewQuery(c)
	days := q.IntDefault("days", 30, validation.Bound(1), validation.Bound(365))
	if !q.OK() {
		return
	}
	ctx := c.Request.Context()
	userID := currentUserID(c)
	to := h.now()
	from := to.AddDate(0, 0, -(days - 1))
	prevTo := from.AddDate(0, 0, -1)
	prevFrom := prevTo.AddDate(0, 0, -(days - 1))

	listings, err := h.Store.OwnedListings(ctx, userID)
	if err != nil {
		h.fail(c, err, "", "")
		return
	}
	byListing, err := h.Store.OwnerStatsByListing(ctx, userID, from, to)
	if err != nil {
		h.fail(c, err, "", "")
		return
	}
	daily, err := h.Store.OwnerDaily(ctx, userID, from, to)
	if err != nil {
		h.fail(c, err, "", "")
		return
	}
	previous, err := h.Store.OwnerTotals(ctx, userID, prevFrom, prevTo)
	if err != nil {
		h.fail(c, err, "", "")
		return
	}

	report := models.StatsReport{
		From:     from.Format(time.DateOnly),
		To:       to.Format(time.DateOnly),
		Previous: previous,
		Daily:    make([]models.DailyStats, 0, days),
		Listings: make([]models.ListingStats, 0, len(listings)),
	}
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		key := d.Format(time.DateOnly)
		counts := daily[key]
		report.Daily = append(report.Daily, models.DailyStats{Day: key, Views: counts.Views, Contacts: counts.Contacts})
	}
	for _, l := range listings {
		counts := byListing[[2]string{l.TargetType, l.ID}]
		report.Listings = append(report.Listings, models.ListingStats{Listing: l, StatCounts: counts})
		report.Totals.Views += counts.Views
		report.Totals.Phone += counts.Phone
		report.Totals.Whatsapp += counts.Whatsapp
		report.Totals.Telegram += counts.Telegram
		report.Totals.Website += counts.Website
		report.Totals.Contacts += counts.Contacts
	}
	respond.JSON(c, http.StatusOK, report)
}
