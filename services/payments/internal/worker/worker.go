// Package worker settles pending orders against the payment gateway (it
// has no callbacks, so payments are polled) and tells the partner and the
// customer about paid bookings in Telegram.
package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/elvn5/sai/services/payments/internal/models"
	"github.com/elvn5/sai/services/payments/internal/provider"
	"github.com/elvn5/sai/services/payments/internal/store"
	"github.com/elvn5/sai/services/pkg/telegram"
)

const (
	batchSize = 50
	// A payment can settle a little after its link expires.
	expiryGrace = 10 * time.Minute
)

// Sender delivers Telegram messages (pkg/telegram); optional.
type Sender interface {
	SendMessage(ctx context.Context, chatID int64, text string, button *telegram.Button) error
}

// Chats resolves users' Telegram chats (the users service).
type Chats interface {
	TelegramChats(ctx context.Context, ids []string) (map[string]int64, error)
}

type Worker struct {
	Store    *store.Store
	Provider provider.Provider
	Sender   Sender
	Chats    Chats
	SiteURL  string
	Log      *slog.Logger
	Now      func() time.Time
	Interval time.Duration
}

func (w *Worker) now() time.Time {
	if w.Now != nil {
		return w.Now()
	}
	return time.Now()
}

func (w *Worker) Run(ctx context.Context) {
	interval := w.Interval
	if interval == 0 {
		interval = 15 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		w.Tick(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Tick is one pass: check pending orders, then send messages.
func (w *Worker) Tick(ctx context.Context) {
	pending, err := w.Store.PendingOrders(ctx, batchSize)
	if err != nil {
		w.Log.ErrorContext(ctx, "listing pending orders failed", slog.Any("error", err))
	}
	for _, o := range pending {
		if _, err := w.Check(ctx, o); err != nil {
			w.Log.WarnContext(ctx, "payment check failed", slog.String("order", o.ID), slog.Any("error", err))
		}
	}
	if err := w.notify(ctx); err != nil {
		w.Log.ErrorContext(ctx, "payment messages failed", slog.Any("error", err))
	}
}

// Check asks the gateway about a pending order and records the outcome.
func (w *Worker) Check(ctx context.Context, o models.Order) (models.Order, error) {
	now := w.now()
	expired := now.After(o.ExpiresAt.Add(expiryGrace))
	res, err := w.Provider.Status(ctx, o.ID)
	if err != nil {
		if expired {
			return w.resolve(ctx, o, models.StatusExpired, nil)
		}
		_ = w.Store.MarkChecked(ctx, o.ID, now)
		return o, err
	}
	switch {
	case res.State == provider.StatePaid && res.Amount > 0 && math.Abs(res.Amount-float64(o.Amount)) >= 1:
		msg := fmt.Sprintf("Paid %.2f instead of %d", res.Amount, o.Amount)
		w.Log.ErrorContext(ctx, "payment amount mismatch", slog.String("order", o.ID), slog.String("detail", msg))
		return w.resolve(ctx, o, models.StatusFailed, &msg)
	case res.State == provider.StatePaid:
		return w.resolve(ctx, o, models.StatusPaid, nil)
	case res.State == provider.StateFailed:
		msg := res.Message
		return w.resolve(ctx, o, models.StatusFailed, &msg)
	case expired:
		return w.resolve(ctx, o, models.StatusExpired, nil)
	default:
		return o, w.Store.MarkChecked(ctx, o.ID, now)
	}
}

func (w *Worker) resolve(ctx context.Context, o models.Order, status string, msg *string) (models.Order, error) {
	resolved, err := w.Store.Resolve(ctx, o.ID, status, msg, w.now())
	if errors.Is(err, store.ErrConflict) {
		return w.Store.Order(ctx, o.ID) // settled concurrently
	}
	if err == nil {
		w.Log.InfoContext(ctx, "order settled", slog.String("order", o.ID), slog.String("status", status))
	}
	return resolved, err
}

// notify sends the "paid" messages once per order (best effort: an
// unreachable chat doesn't block it).
func (w *Worker) notify(ctx context.Context) error {
	orders, err := w.Store.PaidUnnotified(ctx, batchSize)
	if err != nil || len(orders) == 0 {
		return err
	}
	if w.Sender == nil || w.Chats == nil {
		for _, o := range orders {
			if err := w.Store.MarkNotified(ctx, o.ID); err != nil {
				return err
			}
		}
		return nil
	}
	ids := make([]string, 0, 2*len(orders))
	for _, o := range orders {
		ids = append(ids, o.PayeeID, o.UserID)
	}
	chats, err := w.Chats.TelegramChats(ctx, ids)
	if err != nil {
		return err
	}
	for _, o := range orders {
		if chat, ok := chats[o.PayeeID]; ok {
			w.send(ctx, chat, partnerText(o), &telegram.Button{Text: "Мои бронирования", URL: w.SiteURL + "/profile?tab=listings"})
		}
		if chat, ok := chats[o.UserID]; ok {
			w.send(ctx, chat, customerText(o), &telegram.Button{Text: "Мои бронирования", URL: w.SiteURL + "/profile?tab=bookings"})
		}
		if err := w.Store.MarkNotified(ctx, o.ID); err != nil {
			return err
		}
	}
	return nil
}

func (w *Worker) send(ctx context.Context, chat int64, text string, button *telegram.Button) {
	if err := w.Sender.SendMessage(ctx, chat, text, button); err != nil {
		w.Log.WarnContext(ctx, "payment message not delivered", slog.Any("error", err))
	}
}

var months = []string{"янв.", "фев.", "мар.", "апр.", "мая", "июн.", "июл.", "авг.", "сен.", "окт.", "нояб.", "дек."}

func itemLine(o models.Order) string {
	line := "Тур «" + telegram.EscapeHTML(o.ItemTitle) + "»"
	if o.ItemDate != nil {
		if d, err := time.Parse(time.DateOnly, *o.ItemDate); err == nil {
			line += fmt.Sprintf(", %d %s", d.Day(), months[d.Month()-1])
		}
	}
	return line
}

// som formats an amount with thin spaces: 24 000 сом.
func som(n int) string {
	s := fmt.Sprint(n)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteRune(' ')
		}
		b.WriteRune(r)
	}
	return b.String() + " сом"
}

func partnerText(o models.Order) string {
	var b strings.Builder
	fmt.Fprintf(&b, "💳 <b>Новая оплаченная бронь</b>\n%s\nЧеловек: %d\nОплачено: %s", itemLine(o), o.Quantity, som(o.Amount))
	if o.Fee > 0 {
		fmt.Fprintf(&b, " (к выплате вам %s)", som(o.PayeeAmount))
	}
	fmt.Fprintf(&b, "\n\n👤 %s, %s", telegram.EscapeHTML(o.ContactName), telegram.EscapeHTML(o.ContactPhone))
	if o.Comment != nil && *o.Comment != "" {
		fmt.Fprintf(&b, "\n💬 %s", telegram.EscapeHTML(*o.Comment))
	}
	b.WriteString("\n\nСвяжитесь с гостем, чтобы подтвердить детали.")
	return b.String()
}

func customerText(o models.Order) string {
	return fmt.Sprintf("✅ <b>Оплата прошла</b>\n%s\nЧеловек: %d\nСумма: %s\n\nОрганизатор свяжется с вами по номеру %s.",
		itemLine(o), o.Quantity, som(o.Amount), telegram.EscapeHTML(o.ContactPhone))
}
