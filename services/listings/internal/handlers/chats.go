package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/listings/internal/models"
	"github.com/elvn5/sai/services/listings/internal/store"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/jsontime"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/pkg/validation"
)

const (
	chatNotFound = "Chat not found"
	// Messages per page of a chat.
	chatPage = 50
)

type openChatRequest struct {
	// A partner writes to one of their clients; a traveler to a partner
	// who runs tours, an organizer also to a driver.
	UserID    *string `json:"userId" binding:"omitempty,uuid"`
	PartnerID *string `json:"partnerId" binding:"omitempty,uuid"`
}

func (r openChatRequest) Refine() map[string]string {
	if (r.UserID == nil) == (r.PartnerID == nil) {
		return map[string]string{"userId": "Give userId or partnerId"}
	}
	return nil
}

type messageRequest struct {
	Body *validation.Trimmed `json:"body" binding:"required,min=1,max=4000"`
}

// Chats are conversations between a partner and a traveler: the partner
// writes to their clients from the CRM, travelers to the organizers of
// tours. Both read them on the site; the bot tells about new messages.
type Chats struct{ Deps }

// List is the visitor's chats with the other side's profile.
func (h *Chats) List(c *gin.Context) {
	ctx := c.Request.Context()
	me := currentUserID(c)
	chats, err := h.Store.Chats(ctx, me)
	if err != nil {
		h.fail(c, err, chatNotFound, "")
		return
	}
	ids := make([]string, 0, len(chats))
	for _, ch := range chats {
		ids = append(ids, store.Chat{PartnerID: ch.PartnerID, UserID: ch.UserID}.Other(me))
	}
	people := (&CRM{Deps: h.Deps}).accounts(ctx, ids)
	for i, ch := range chats {
		chats[i] = summarize(ch, me)
		chats[i].Peer = authorFor(people, ids[i])
	}
	if chats == nil {
		chats = []models.ChatSummary{}
	}
	respond.JSON(c, http.StatusOK, chats)
}

// summarize sets the visitor's role in the chat.
func summarize(ch models.ChatSummary, me string) models.ChatSummary {
	ch.Role = "client"
	if ch.PartnerID == me {
		ch.Role = "partner"
	}
	return ch
}

// Unread is how many chats have messages the visitor hasn't read.
func (h *Chats) Unread(c *gin.Context) {
	n, err := h.Store.UnreadChats(c.Request.Context(), currentUserID(c))
	if err != nil {
		h.fail(c, err, chatNotFound, "")
		return
	}
	respond.JSON(c, http.StatusOK, gin.H{"chats": n})
}

// Open starts (or finds) a chat and answers with it.
func (h *Chats) Open(c *gin.Context) {
	var req openChatRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	ctx := c.Request.Context()
	me := currentUserID(c)
	var partner, traveler string
	if req.UserID != nil {
		// Partners write only to their own clients.
		ok, err := h.Store.IsClient(ctx, me, *req.UserID)
		if err != nil {
			h.fail(c, err, clientNotFound, "")
			return
		}
		if !ok {
			httperr.Abort(c, http.StatusNotFound, clientNotFound)
			return
		}
		partner, traveler = me, *req.UserID
	} else {
		organizer, err := h.Store.OrganizesTours(ctx, *req.PartnerID)
		driver := false
		if err == nil && !organizer {
			// Organizers also write to the drivers in their catalog.
			driver, err = h.Store.IsPublishedDriver(ctx, *req.PartnerID)
		}
		if err != nil {
			h.fail(c, err, chatNotFound, "")
			return
		}
		if (!organizer && !driver) || *req.PartnerID == me {
			httperr.Abort(c, http.StatusNotFound, "Organizer not found")
			return
		}
		partner, traveler = *req.PartnerID, me
		// A traveler writing to an organizer is a lead.
		if organizer {
			err = h.Store.OpenInquiry(ctx, partner, traveler)
		} else {
			err = h.Store.AddAccountClient(ctx, partner, traveler)
		}
		if err != nil {
			h.fail(c, err, chatNotFound, "")
			return
		}
	}
	id, err := h.Store.OpenChat(ctx, partner, traveler)
	if err != nil {
		h.fail(c, err, chatNotFound, "")
		return
	}
	h.respondThread(c, http.StatusOK, store.Chat{ID: id, PartnerID: partner, UserID: traveler}, nil)
}

// load finds a chat the visitor is in.
func (h *Chats) load(c *gin.Context) (store.Chat, bool) {
	id := c.Param("id")
	if !validID(id) {
		httperr.Abort(c, http.StatusNotFound, chatNotFound)
		return store.Chat{}, false
	}
	ch, err := h.Store.Chat(c.Request.Context(), id)
	if err == nil && !ch.Has(currentUserID(c)) {
		err = store.ErrNotFound
	}
	if err != nil {
		h.fail(c, err, chatNotFound, "")
		return ch, false
	}
	return ch, true
}

// Messages is a chat with its latest messages, or those before ?before=.
// Reading the latest marks the chat read.
func (h *Chats) Messages(c *gin.Context) {
	q := validation.NewQuery(c)
	beforeRaw := q.String("before", 40)
	if !q.OK() {
		return
	}
	var before *time.Time
	if beforeRaw != nil {
		at, err := time.Parse(time.RFC3339Nano, *beforeRaw)
		if err != nil {
			httperr.Abort(c, http.StatusBadRequest, "Invalid before")
			return
		}
		before = &at
	}
	ch, ok := h.load(c)
	if !ok {
		return
	}
	if before == nil {
		if err := h.Store.MarkChatRead(c.Request.Context(), ch, currentUserID(c), h.now()); err != nil {
			h.fail(c, err, chatNotFound, "")
			return
		}
	}
	h.respondThread(c, http.StatusOK, ch, before)
}

func (h *Chats) respondThread(c *gin.Context, status int, ch store.Chat, before *time.Time) {
	ctx := c.Request.Context()
	me := currentUserID(c)
	other := ch.Other(me)
	messages, err := h.Store.Messages(ctx, ch.ID, before, chatPage)
	if err != nil {
		h.fail(c, err, chatNotFound, "")
		return
	}
	readAt, err := h.Store.ReadAt(ctx, ch, other)
	if err != nil {
		h.fail(c, err, chatNotFound, "")
		return
	}
	people := (&CRM{Deps: h.Deps}).accounts(ctx, []string{other})
	thread := models.ChatThread{
		ChatSummary: summarize(models.ChatSummary{ID: ch.ID, PartnerID: ch.PartnerID, UserID: ch.UserID}, me),
		PeerReadAt:  jsontime.Ptr(readAt),
		Messages:    messages,
	}
	thread.Peer = authorFor(people, other)
	if thread.Messages == nil {
		thread.Messages = []models.ChatMessage{}
	}
	if n := len(thread.Messages); n > 0 {
		thread.LastMessage = thread.Messages[n-1]
	}
	respond.JSON(c, status, thread)
}

// Send adds a message and tells the other side through the bot.
func (h *Chats) Send(c *gin.Context) {
	var req messageRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	ch, ok := h.load(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	me := currentUserID(c)
	body := strings.TrimSpace(string(*req.Body))
	msg, err := h.Store.AddMessage(ctx, ch, me, body, h.now())
	if err != nil {
		h.fail(c, err, chatNotFound, "")
		return
	}
	if h.Notify != nil {
		h.Notify.Message(ctx, ch, me, body)
	}
	respond.JSON(c, http.StatusCreated, msg)
}

// Read marks the chat read by the visitor.
func (h *Chats) Read(c *gin.Context) {
	ch, ok := h.load(c)
	if !ok {
		return
	}
	if err := h.Store.MarkChatRead(c.Request.Context(), ch, currentUserID(c), h.now()); err != nil {
		h.fail(c, err, chatNotFound, "")
		return
	}
	c.Status(http.StatusNoContent)
}
