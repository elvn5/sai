package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
)

func (h *Handlers) cabinetMenu(ctx *th.Context, msg telego.Message) error {
	if msg.From == nil || msg.Chat.ID != msg.From.ID {
		return nil
	}
	user, err := h.userOf(ctx, msg.From)
	if err != nil {
		return err
	}
	h.Sessions.Cancel(msg.Chat.ID)
	l := h.lang(ctx, msg.From)
	s, err := h.Cabinet.Handle(ctx, user, msg.Chat.ID, l, "home")
	if err != nil {
		return h.failed(ctx, l, msg.Chat.ID, err)
	}
	return h.send(ctx, msg.Chat.ID, s)
}
func (h *Handlers) cabinetPress(ctx *th.Context, q telego.CallbackQuery) error {
	if chatOf(q) != q.From.ID {
		return h.answer(ctx, q, "")
	}
	l := h.lang(ctx, &q.From)
	user, err := h.userOf(ctx, &q.From)
	if err != nil {
		return h.failedPress(ctx, l, q, err)
	}
	if err := h.answer(ctx, q, ""); err != nil {
		return err
	}
	h.Sessions.Cancel(chatOf(q))
	s, err := h.Cabinet.Handle(ctx, user, chatOf(q), l, strings.TrimPrefix(q.Data, "k:"))
	if err != nil {
		return h.failed(ctx, l, chatOf(q), err)
	}
	return h.show(ctx, q, s)
}

func (h *Handlers) cabinetPhoto(ctx *th.Context, msg telego.Message) error {
	if msg.From == nil || msg.Chat.ID != msg.From.ID || len(msg.Photo) == 0 {
		return nil
	}
	user, err := h.userOf(ctx, msg.From)
	if err != nil {
		return err
	}
	l := h.lang(ctx, msg.From)
	screen, handled, err := h.Cabinet.Photo(ctx, user, msg.Chat.ID, l, func() (string, error) {
		uploader, ok := h.Content.(interface {
			UploadImage(context.Context, string, []byte) (string, error)
		})
		if !ok {
			return "", fmt.Errorf("upload unavailable")
		}
		photo := msg.Photo[len(msg.Photo)-1]
		file, err := ctx.Bot().GetFile(ctx, &telego.GetFileParams{FileID: photo.FileID})
		if err != nil {
			return "", fmt.Errorf("telegram photo unavailable")
		}
		if file.FileSize > 8<<20 {
			return "", fmt.Errorf("photo exceeds upload limit")
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, ctx.Bot().FileDownloadURL(file.FilePath), nil)
		if err != nil {
			return "", fmt.Errorf("photo request failed")
		}
		client := &http.Client{Timeout: 20 * time.Second}
		res, err := client.Do(req)
		if err != nil {
			return "", fmt.Errorf("photo download failed")
		}
		defer func() { _ = res.Body.Close() }()
		if res.StatusCode != http.StatusOK {
			return "", fmt.Errorf("photo download rejected")
		}
		data, err := io.ReadAll(io.LimitReader(res.Body, (8<<20)+1))
		if err != nil {
			return "", fmt.Errorf("photo download incomplete")
		}
		return uploader.UploadImage(ctx, user, data)
	})
	if err != nil {
		return h.failed(ctx, l, msg.Chat.ID, err)
	}
	if handled {
		return h.send(ctx, msg.Chat.ID, screen)
	}
	return nil
}

// recordTourView feeds the partner statistics for the existing tour screens.
func (h *Handlers) recordTourView(ctx context.Context, userID, tourID string) {
	if h.Cabinet == nil {
		return
	}
	err := h.Cabinet.API.CabinetRequest(ctx, userID, "POST", "/listing-stats/events", map[string]any{"targetType": "tour", "targetId": tourID, "event": "view"}, nil)
	if err != nil {
		h.Log.DebugContext(ctx, "tour view not recorded")
	}
}
