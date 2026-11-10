package cabinet

import (
	"context"

	"github.com/elvn5/sai/services/bot/internal/i18n"
	"github.com/elvn5/sai/services/bot/internal/screens"
)

// Photo keeps the pending field locked across upload, so a slow download cannot
// attach its result to a different form opened in the meantime.
func (e *Engine) Photo(ctx context.Context, user string, chat int64, l i18n.Lang, upload func() (string, error)) (screens.Screen, bool, error) {
	s := e.state(chat)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.user != user || s.pending == nil || s.form == nil || (s.pending.kind != "photo" && s.pending.kind != "photos") || e.Now().Sub(s.touched) > lifetime {
		return screens.Screen{}, false, nil
	}
	f := *s.pending
	v := &view{e: e, s: s, ctx: ctx, user: user, l: l}
	old, _ := s.form.values[f.key].([]any)
	if f.kind == "photos" && len(old) >= f.max {
		screen, err := v.prompt(f, 0)
		screen.Text = l.T("Слишком много фотографий.") + "\n" + screen.Text
		return screen, true, err
	}
	link, err := upload()
	if err != nil {
		return screens.Screen{}, true, err
	}
	s.actions = map[string]action{}
	s.touched = e.Now()
	if f.kind == "photos" {
		s.form.values[f.key] = append(old, link)
	} else {
		s.form.values[f.key] = link
	}
	s.pending = nil
	screen, err := v.edit(s.form, 0)
	return screen, true, err
}
