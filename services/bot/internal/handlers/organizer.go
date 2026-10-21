package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/keyboards"
	"github.com/elvn5/sai/services/bot/internal/screens"
	"github.com/elvn5/sai/services/bot/internal/session"
)

// Organizing is the public API as the organizer's room uses it
// (api.Client): marking arrivals, the organizer's tours, their numbers and
// bookings.
type Organizing interface {
	SetArrival(ctx context.Context, userID, outingID, participantID string, arrived bool) (api.Outing, error)
	MyTours(ctx context.Context, userID string) ([]api.Tour, error)
	ListingStats(ctx context.Context, userID string) (api.ListingStats, error)
	ReceivedOrders(ctx context.Context, userID string) ([]api.Order, error)
}

// Chats finds users' Telegram chats (users.Client): to link participants
// and send them the organizer's messages.
type Chats interface {
	TelegramChats(ctx context.Context, ids []string) (map[string]int64, error)
}

// arrivalErrors are the trails service's refusals to mark an arrival.
var arrivalErrors = map[string]string{
	"Check-in opens a day before the start": "Отмечать прибывших можно за сутки до сбора",
	"Check-in is over":                      "Отметка закрыта: прошла неделя после выхода",
	"This outing is closed":                 "Выход отменён — отмечать некого",
	"Participant not found":                 "Этот участник уже не записан",
	"Only the organizer can mark arrivals":  "Отмечать прибывших может только организатор",
}

const notOrganizer = "Это может только организатор выхода"

func (h *Handlers) organizing() bool { return h.sections() && h.Organizer != nil }

// orgHome gathers the organizer's room: the outings they run, their tours.
func (h *Handlers) orgHome(ctx context.Context, userID string) (screens.Screen, error) {
	list, err := h.Content.MyOutings(ctx, userID)
	if err != nil {
		return screens.Screen{}, err
	}
	outings := screens.OrganizedOutings(list, h.now())
	// The list doesn't carry the roster: load it where arrivals count.
	for i, o := range outings {
		if screens.CanCheckIn(o, h.now()) && o.Taken > 0 {
			if full, err := h.Content.Outing(ctx, userID, o.ID); err == nil {
				outings[i] = full
			}
		}
	}
	return screens.OrganizerHome(outings, h.myTours(ctx, userID), h.now()), nil
}

// myTours are the tours the user runs; none when they aren't a partner or
// listings is down (the outings still show).
func (h *Handlers) myTours(ctx context.Context, userID string) []api.Tour {
	tours, err := h.Organizer.MyTours(ctx, userID)
	if err != nil {
		h.Log.WarnContext(ctx, "organizer's tours", slog.Any("error", err))
		return nil
	}
	return tours
}

func (h *Handlers) organizerMenu(ctx *th.Context, msg telego.Message) error {
	if msg.From == nil {
		return nil
	}
	userID, err := h.userOf(ctx, msg.From)
	if err != nil {
		return err
	}
	s, err := h.orgHome(ctx, userID)
	if err != nil {
		return h.failed(ctx, msg.Chat.ID, err)
	}
	return h.send(ctx, msg.Chat.ID, s)
}

func (h *Handlers) organizerPress(ctx *th.Context, q telego.CallbackQuery) error {
	parts := strings.Split(strings.TrimPrefix(q.Data, keyboards.OrganizerPrefix), ":")
	userID, err := h.userOf(ctx, &q.From)
	if err != nil {
		return err
	}
	switch {
	case q.Data == screens.OrgHome:
		s, err := h.orgHome(ctx, userID)
		if err != nil {
			return h.failedPress(ctx, q, err)
		}
		if err := h.answer(ctx, q, ""); err != nil {
			return err
		}
		return h.show(ctx, q, s)
	case len(parts) == 2 && (parts[0] == screens.OrgRoster || parts[0] == screens.OrgRosterNotice):
		return h.roster(ctx, q, userID, parts[1], "", parts[0] == screens.OrgRosterNotice)
	case len(parts) == 3 && parts[0] == screens.OrgArrival:
		return h.markArrival(ctx, q, userID, parts[1], parts[2])
	case len(parts) == 2 && parts[0] == screens.OrgArrivedAll:
		return h.markAllArrived(ctx, q, userID, parts[1])
	case len(parts) == 2 && parts[0] == screens.OrgWrite:
		return h.startBroadcast(ctx, q, userID, parts[1])
	case len(parts) == 2 && parts[0] == screens.OrgSend:
		return h.sendBroadcast(ctx, q, userID, parts[1])
	case len(parts) == 2 && parts[0] == screens.OrgWriteAbort:
		h.Sessions.Cancel(chatOf(q))
		return h.roster(ctx, q, userID, parts[1], "Сообщение не отправлено", false)
	case len(parts) >= 2 && (parts[0] == screens.OrgTour || parts[0] == screens.OrgProgram || parts[0] == screens.OrgBookings):
		return h.orgTour(ctx, q, userID, parts)
	}
	return h.answer(ctx, q, "")
}

// organized loads an outing the user organizes, with its roster; false
// (the press answered) when it is gone or not theirs.
func (h *Handlers) organized(ctx *th.Context, q telego.CallbackQuery, userID, id string) (api.Outing, bool, error) {
	o, err := h.Content.Outing(ctx, userID, id)
	if api.StatusOf(err) == http.StatusNotFound || api.StatusOf(err) == http.StatusBadRequest {
		return o, false, h.answer(ctx, q, "Этого выхода больше нет")
	}
	if err != nil {
		return o, false, h.failedPress(ctx, q, err)
	}
	if !o.IsOrganizer {
		return o, false, h.answer(ctx, q, notOrganizer)
	}
	return o, true, nil
}

// chatsOf finds the participants' chats, best effort (names stay plain).
func (h *Handlers) chatsOf(ctx context.Context, o api.Outing) map[string]int64 {
	if h.Chats == nil || len(o.Roster) == 0 {
		return nil
	}
	ids := make([]string, 0, len(o.Roster))
	for _, p := range o.Roster {
		ids = append(ids, p.ID)
	}
	chats, err := h.Chats.TelegramChats(ctx, ids)
	if err != nil {
		h.Log.WarnContext(ctx, "participants' chats", slog.Any("error", err))
	}
	return chats
}

func (h *Handlers) showRoster(ctx *th.Context, q telego.CallbackQuery, o api.Outing, toast string, notice bool) error {
	if err := h.answer(ctx, q, toast); err != nil {
		return err
	}
	s := screens.Roster(o, h.chatsOf(ctx, o), h.Links, h.now())
	if notice {
		return h.send(ctx, chatOf(q), s)
	}
	return h.show(ctx, q, s)
}

func (h *Handlers) roster(ctx *th.Context, q telego.CallbackQuery, userID, id, toast string, notice bool) error {
	o, ok, err := h.organized(ctx, q, userID, id)
	if !ok {
		return err
	}
	return h.showRoster(ctx, q, o, toast, notice)
}

func (h *Handlers) markArrival(ctx *th.Context, q telego.CallbackQuery, userID, id, key string) error {
	o, ok, err := h.organized(ctx, q, userID, id)
	if !ok {
		return err
	}
	for i, p := range o.Roster {
		if screens.ParticipantKey(p.ID) != key {
			continue
		}
		arrived := p.ArrivedAt == nil
		updated, err := h.Organizer.SetArrival(ctx, userID, o.ID, p.ID, arrived)
		if msg, known := arrivalErrors[api.MessageOf(err)]; known {
			return h.showRoster(ctx, q, o, msg, false)
		}
		if err != nil {
			return h.failedPress(ctx, q, err)
		}
		name := p.Name()
		if name == "" {
			name = "Участник " + strconv.Itoa(i+1)
		}
		toast := "✅ " + name + " на месте"
		if !arrived {
			toast = name + ": отметка снята"
		}
		return h.showRoster(ctx, q, updated, toast, false)
	}
	return h.showRoster(ctx, q, o, "Этот участник уже не записан", false)
}

func (h *Handlers) markAllArrived(ctx *th.Context, q telego.CallbackQuery, userID, id string) error {
	o, ok, err := h.organized(ctx, q, userID, id)
	if !ok {
		return err
	}
	for _, p := range o.Roster {
		if p.ArrivedAt != nil {
			continue
		}
		updated, err := h.Organizer.SetArrival(ctx, userID, o.ID, p.ID, true)
		if msg, known := arrivalErrors[api.MessageOf(err)]; known && api.MessageOf(err) != "Participant not found" {
			return h.showRoster(ctx, q, o, msg, false)
		}
		if err != nil && api.StatusOf(err) != http.StatusNotFound {
			return h.failedPress(ctx, q, err)
		}
		if err == nil {
			o = updated
		}
	}
	return h.showRoster(ctx, q, o, "✅ Все на месте — хорошего похода!", false)
}

// startBroadcast asks the organizer for a message: the next text is it.
func (h *Handlers) startBroadcast(ctx *th.Context, q telego.CallbackQuery, userID, id string) error {
	o, ok, err := h.organized(ctx, q, userID, id)
	if !ok {
		return err
	}
	if o.Status != "open" || len(o.Participants) == 0 {
		return h.showRoster(ctx, q, o, "Некому писать: на выход никто не записан", false)
	}
	h.Sessions.StartWith(chatOf(q), session.FieldBroadcast, o.ID)
	if err := h.answer(ctx, q, ""); err != nil {
		return err
	}
	return h.show(ctx, q, screens.BroadcastPrompt(o))
}

// broadcastText takes the organizer's typed message and shows it before
// sending; typing again replaces it.
func (h *Handlers) broadcastText(ctx *th.Context, msg telego.Message, outingID string) error {
	text := strings.TrimSpace(msg.Text)
	if text == "" || msg.From == nil {
		h.Sessions.StartWith(msg.Chat.ID, session.FieldBroadcast, outingID)
		return h.reply(ctx, msg.Chat.ID, "Напишите сообщение текстом.", nil)
	}
	if utf8.RuneCountInString(text) > screens.BroadcastMaxLen {
		h.Sessions.StartWith(msg.Chat.ID, session.FieldBroadcast, outingID)
		return h.reply(ctx, msg.Chat.ID, "Слишком длинно: не больше "+strconv.Itoa(screens.BroadcastMaxLen)+
			" символов. Сократите и отправьте ещё раз.", nil)
	}
	userID, err := h.userOf(ctx, msg.From)
	if err != nil {
		return err
	}
	o, err := h.Content.Outing(ctx, userID, outingID)
	if err != nil {
		return h.failed(ctx, msg.Chat.ID, err)
	}
	if !o.IsOrganizer {
		return h.reply(ctx, msg.Chat.ID, notOrganizer, nil)
	}
	h.Sessions.StartWith(msg.Chat.ID, session.FieldBroadcastReady, outingID+"\n"+text)
	return h.send(ctx, msg.Chat.ID, screens.BroadcastPreview(o, text))
}

// sendBroadcast sends the written message to every participant with a
// Telegram chat and reports how many got it.
func (h *Handlers) sendBroadcast(ctx *th.Context, q telego.CallbackQuery, userID, id string) error {
	chatID := chatOf(q)
	field, arg, ok := h.Sessions.ConsumeWith(chatID)
	outingID, text, _ := strings.Cut(arg, "\n")
	if !ok || field != session.FieldBroadcastReady || outingID != id || text == "" {
		if ok {
			h.Sessions.StartWith(chatID, field, arg)
		}
		if err := h.answer(ctx, q, "Сообщение устарело — напишите его заново"); err != nil {
			return err
		}
		h.dropButtons(ctx, q)
		return nil
	}
	o, ok, err := h.organized(ctx, q, userID, id)
	if !ok {
		return err
	}
	ids := make([]string, 0, len(o.Participants))
	for _, p := range o.Participants {
		ids = append(ids, p.ID)
	}
	chats := map[string]int64{}
	if h.Chats != nil && len(ids) > 0 {
		if chats, err = h.Chats.TelegramChats(ctx, ids); err != nil {
			return h.failedPress(ctx, q, err)
		}
	}
	if err := h.answer(ctx, q, "Отправляем…"); err != nil {
		return err
	}
	message := screens.BroadcastMessage(o, text)
	sent := 0
	for _, id := range ids {
		chat, ok := chats[id]
		if !ok {
			continue
		}
		if err := h.send(ctx, chat, message); err != nil {
			h.Log.WarnContext(ctx, "organizer's message not delivered", slog.String("user", id), slog.Any("error", err))
			continue
		}
		sent++
	}
	return h.show(ctx, q, screens.BroadcastSent(o, sent, len(ids)))
}

// orgTour shows one of the organizer's tours ("t:<index>"), a day of its
// program ("tp:<index>:<day>") or its bookings ("tb:<index>").
func (h *Handlers) orgTour(ctx *th.Context, q telego.CallbackQuery, userID string, parts []string) error {
	index, err := strconv.Atoi(parts[1])
	if err != nil {
		return h.answer(ctx, q, "")
	}
	tours, err := h.Organizer.MyTours(ctx, userID)
	if err != nil {
		return h.failedPress(ctx, q, err)
	}
	if index < 0 || index >= len(tours) {
		if err := h.answer(ctx, q, "Этого тура больше нет"); err != nil {
			return err
		}
		s, err := h.orgHome(ctx, userID)
		if err != nil {
			return h.failed(ctx, chatOf(q), err)
		}
		return h.show(ctx, q, s)
	}
	t := tours[index]
	if parts[0] == screens.OrgProgram {
		day := 0
		if len(parts) == 3 {
			day, _ = strconv.Atoi(parts[2])
		}
		if err := h.answer(ctx, q, ""); err != nil {
			return err
		}
		return h.show(ctx, q, screens.OrganizerProgram(t, index, day))
	}

	var bookings []api.Order
	if orders, err := h.Organizer.ReceivedOrders(ctx, userID); err == nil {
		bookings = screens.TourBookings(orders, t.ID)
	} else {
		h.Log.WarnContext(ctx, "tour bookings", slog.Any("error", err))
	}
	if err := h.answer(ctx, q, ""); err != nil {
		return err
	}
	if parts[0] == screens.OrgBookings {
		return h.show(ctx, q, screens.OrganizerBookings(t, index, bookings))
	}
	var stats *api.StatCounts
	if all, err := h.Organizer.ListingStats(ctx, userID); err == nil {
		s := all.Of("tour", t.ID)
		stats = &s
	} else {
		h.Log.WarnContext(ctx, "tour stats", slog.Any("error", err))
	}
	return h.show(ctx, q, screens.OrganizerTour(t, index, stats, bookings, h.Links))
}
