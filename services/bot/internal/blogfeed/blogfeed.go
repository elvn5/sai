// Package blogfeed sends new blog posts to the chats subscribed to the
// blog: it checks the blog now and then and posts what it hasn't seen,
// with the cover as a photo. Posts are marked seen before they go out, so
// a crash never repeats one (it may skip one instead).
package blogfeed

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegoapi"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/screens"
)

const (
	// Posts sent per pass at most: a burst of publishing isn't a flood.
	maxPerPass = 3
	// Pause between messages: well under Telegram's per-bot limit.
	sendGap = 50 * time.Millisecond
	// Marks that the first pass ran, even on an empty blog.
	startMark = "__start__"
)

// Posts lists published posts, newest first (api.Client).
type Posts interface {
	BlogPosts(ctx context.Context) ([]api.BlogPost, error)
}

// Store is the bot's database (store.Store).
type Store interface {
	SeenPosts(ctx context.Context) (map[string]bool, error)
	MarkSeen(ctx context.Context, ids []string) error
	BlogSubscribers(ctx context.Context) ([]int64, error)
	SetBlogSubscriber(ctx context.Context, chatID int64, on bool) error
}

// Bot sends messages (telego.Bot).
type Bot interface {
	SendMessage(ctx context.Context, params *telego.SendMessageParams) (*telego.Message, error)
	SendPhoto(ctx context.Context, params *telego.SendPhotoParams) (*telego.Message, error)
}

type Feed struct {
	Posts    Posts
	Store    Store
	Bot      Bot
	Links    screens.Links
	Log      *slog.Logger
	Interval time.Duration
}

// Run checks the blog every Interval until ctx ends.
func (f *Feed) Run(ctx context.Context) {
	ticker := time.NewTicker(f.Interval)
	defer ticker.Stop()
	for {
		if err := f.Pass(ctx); err != nil && ctx.Err() == nil {
			f.Log.ErrorContext(ctx, "blog feed", slog.Any("error", err))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Pass sends the posts published since the last pass. The first pass only
// remembers what is there.
func (f *Feed) Pass(ctx context.Context) error {
	posts, err := f.Posts.BlogPosts(ctx)
	if err != nil {
		return err
	}
	seen, err := f.Store.SeenPosts(ctx)
	if err != nil {
		return err
	}
	if len(seen) == 0 {
		ids := []string{startMark}
		for _, p := range posts {
			ids = append(ids, p.ID)
		}
		return f.Store.MarkSeen(ctx, ids)
	}

	// Oldest first, so the chat reads in publishing order.
	var fresh []api.BlogPost
	for i := len(posts) - 1; i >= 0; i-- {
		if !seen[posts[i].ID] {
			fresh = append(fresh, posts[i])
		}
	}
	if len(fresh) == 0 {
		return nil
	}
	ids := make([]string, len(fresh))
	for i, p := range fresh {
		ids[i] = p.ID
	}
	if err := f.Store.MarkSeen(ctx, ids); err != nil {
		return err
	}
	if len(fresh) > maxPerPass {
		fresh = fresh[len(fresh)-maxPerPass:]
	}

	chats, err := f.Store.BlogSubscribers(ctx)
	if err != nil {
		return err
	}
	for _, p := range fresh {
		for _, chat := range chats {
			if err := f.send(ctx, chat, p); err != nil {
				f.dropped(ctx, chat, err)
			}
			time.Sleep(sendGap)
		}
	}
	return nil
}

func (f *Feed) send(ctx context.Context, chat int64, p api.BlogPost) error {
	caption, markup := screens.BlogPost(p)
	if picture := f.Links.Picture(p.CoverImageURL); picture != "" {
		_, err := f.Bot.SendPhoto(ctx, tu.Photo(tu.ID(chat), tu.FileFromURL(picture)).
			WithCaption(caption).WithParseMode(telego.ModeHTML).WithReplyMarkup(markup))
		if err == nil || undeliverable(err) {
			return err
		}
		// Telegram couldn't fetch the picture: send the text alone.
		f.Log.WarnContext(ctx, "blog post photo not sent", slog.Any("error", err))
	}
	_, err := f.Bot.SendMessage(ctx, tu.Message(tu.ID(chat), caption).WithParseMode(telego.ModeHTML).
		WithLinkPreviewOptions(&telego.LinkPreviewOptions{IsDisabled: true}).WithReplyMarkup(markup))
	return err
}

// undeliverable: the user blocked the bot or the chat is gone.
func undeliverable(err error) bool {
	var apiErr *telegoapi.Error
	return errors.As(err, &apiErr) && apiErr.ErrorCode == http.StatusForbidden
}

// dropped logs a failed message and unsubscribes chats that can't
// receive any.
func (f *Feed) dropped(ctx context.Context, chat int64, err error) {
	if !undeliverable(err) {
		f.Log.WarnContext(ctx, "blog post not sent", slog.Int64("chat", chat), slog.Any("error", err))
		return
	}
	if err := f.Store.SetBlogSubscriber(ctx, chat, false); err != nil {
		f.Log.WarnContext(ctx, "could not unsubscribe a blocked chat", slog.Int64("chat", chat), slog.Any("error", err))
	}
}
