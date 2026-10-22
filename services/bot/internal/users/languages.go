package users

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// LanguageStore keeps the language of each chat (store.Store).
type LanguageStore interface {
	Language(ctx context.Context, chatID int64, fallback string) (string, error)
	KnownLanguage(ctx context.Context, chatID int64) (string, error)
	SetLanguage(ctx context.Context, chatID int64, lang string) error
}

// LanguageReporter passes a chat's language on (Client.SetLanguage).
type LanguageReporter interface {
	SetLanguage(ctx context.Context, telegramID int64, lang string) error
}

// Languages keeps the chats' languages in the bot's store and reports each
// one to the users service, where trails and listings read it for their
// notifications: once per chat while the bot runs (so chats that predate
// the reporting catch up on their next message), and again on a change.
type Languages struct {
	Store  LanguageStore
	Users  LanguageReporter
	Log    *slog.Logger
	mu     sync.Mutex
	synced map[int64]string
}

// Language is the chat's language (fallback for a new chat, kept); its
// report goes out in the background, so a slow users service doesn't
// hold up the answer.
func (l *Languages) Language(ctx context.Context, chatID int64, fallback string) (string, error) {
	lang, err := l.Store.Language(ctx, chatID, fallback)
	if err != nil || !l.claim(chatID, lang) {
		return lang, err
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		l.report(ctx, chatID, lang)
	}()
	return lang, nil
}

func (l *Languages) KnownLanguage(ctx context.Context, chatID int64) (string, error) {
	return l.Store.KnownLanguage(ctx, chatID)
}

// SetLanguage keeps the language the user picked and reports it.
func (l *Languages) SetLanguage(ctx context.Context, chatID int64, lang string) error {
	if err := l.Store.SetLanguage(ctx, chatID, lang); err != nil {
		return err
	}
	if l.claim(chatID, lang) {
		l.report(ctx, chatID, lang)
	}
	return nil
}

// claim marks lang as reported for the chat; false when it already was.
func (l *Languages) claim(chatID int64, lang string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.synced[chatID] == lang {
		return false
	}
	if l.synced == nil {
		l.synced = map[int64]string{}
	}
	l.synced[chatID] = lang
	return true
}

// report tells the users service, best effort: a failed report is tried
// again on the chat's next message.
func (l *Languages) report(ctx context.Context, chatID int64, lang string) {
	if err := l.Users.SetLanguage(ctx, chatID, lang); err != nil {
		l.Log.WarnContext(ctx, "could not report the chat's language", slog.Int64("chat", chatID), slog.Any("error", err))
		l.mu.Lock()
		if l.synced[chatID] == lang {
			delete(l.synced, chatID)
		}
		l.mu.Unlock()
	}
}
