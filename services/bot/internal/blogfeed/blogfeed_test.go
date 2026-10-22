package blogfeed_test

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegoapi"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/blogfeed"
	"github.com/elvn5/sai/services/bot/internal/screens"
	"github.com/elvn5/sai/services/bot/internal/store"
)

// fakePosts has Russian posts and, by language, the others.
type fakePosts struct {
	posts  []api.BlogPost
	byLang map[string][]api.BlogPost
}

func (f *fakePosts) BlogPosts(_ context.Context, locale string) ([]api.BlogPost, error) {
	if locale == "ru" {
		return f.posts, nil
	}
	return f.byLang[locale], nil
}

type fakeStore struct {
	mu   sync.Mutex
	seen map[string]bool
	subs map[int64]bool
	// Chats' languages (Russian when not set).
	langs map[int64]string
}

func (s *fakeStore) SeenPosts(context.Context) (map[string]bool, error) {
	out := map[string]bool{}
	for k := range s.seen {
		out[k] = true
	}
	return out, nil
}

func (s *fakeStore) MarkSeen(_ context.Context, ids []string) error {
	for _, id := range ids {
		s.seen[id] = true
	}
	return nil
}

func (s *fakeStore) BlogSubscribers(context.Context) ([]store.Subscriber, error) {
	var out []store.Subscriber
	for id := range s.subs {
		lang := s.langs[id]
		if lang == "" {
			lang = "ru"
		}
		out = append(out, store.Subscriber{ChatID: id, Lang: lang})
	}
	return out, nil
}

func (s *fakeStore) SetBlogSubscriber(_ context.Context, chatID int64, on bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if on {
		s.subs[chatID] = true
	} else {
		delete(s.subs, chatID)
	}
	return nil
}

// fakeBot records sends; chat 13 blocked the bot, chat 7 is flaky.
type fakeBot struct {
	mu        sync.Mutex
	sent      []string
	chats     []int64
	badPhotos bool
}

func (b *fakeBot) SendMessage(_ context.Context, p *telego.SendMessageParams) (*telego.Message, error) {
	if p.ChatID.ID == 13 {
		return nil, &telegoapi.Error{ErrorCode: 403, Description: "Forbidden: bot was blocked by the user"}
	}
	if p.ChatID.ID == 7 {
		return nil, &telegoapi.Error{ErrorCode: 500, Description: "Internal Server Error"}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sent = append(b.sent, "text "+p.Text)
	b.chats = append(b.chats, p.ChatID.ID)
	return &telego.Message{}, nil
}

func (b *fakeBot) SendPhoto(_ context.Context, p *telego.SendPhotoParams) (*telego.Message, error) {
	if p.ChatID.ID == 13 {
		return nil, &telegoapi.Error{ErrorCode: 403, Description: "Forbidden: bot was blocked by the user"}
	}
	if b.badPhotos {
		return nil, &telegoapi.Error{ErrorCode: 400, Description: "Bad Request: wrong file identifier/HTTP URL specified"}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sent = append(b.sent, "photo "+p.Photo.URL+" "+p.Caption)
	b.chats = append(b.chats, p.ChatID.ID)
	return &telego.Message{}, nil
}

func cover(s string) *string { return &s }

func TestFeed(t *testing.T) {
	posts := &fakePosts{posts: []api.BlogPost{{ID: "old", Title: "Старая"}}}
	store := &fakeStore{seen: map[string]bool{}, subs: map[int64]bool{42: true, 13: true}}
	bot := &fakeBot{}
	feed := &blogfeed.Feed{Posts: posts, Store: store, Bot: bot, Log: slog.New(slog.DiscardHandler),
		Links: screens.Links{}}
	ctx := context.Background()

	// The first pass only remembers what is there.
	if err := feed.Pass(ctx); err != nil {
		t.Fatal(err)
	}
	if len(bot.sent) != 0 || !store.seen["old"] {
		t.Fatalf("first pass sent %v, seen %v", bot.sent, store.seen)
	}

	posts.posts = []api.BlogPost{
		{ID: "b", Title: "Вторая", CoverImageURL: cover("https://cdn.example.com/b.jpg")},
		{ID: "a", Title: "Первая"},
		{ID: "old", Title: "Старая"},
	}
	if err := feed.Pass(ctx); err != nil {
		t.Fatal(err)
	}
	if len(bot.sent) != 2 || !strings.HasPrefix(bot.sent[0], "text 📰 <b>Новая статья в блоге</b>\n\n<b>Первая</b>") ||
		!strings.HasPrefix(bot.sent[1], "photo https://cdn.example.com/b.jpg 📰") {
		t.Errorf("sent: %q", bot.sent)
	}
	if store.subs[13] || !store.subs[42] {
		t.Errorf("blocked chat still subscribed: %v", store.subs)
	}

	// Nothing new, nothing sent.
	if err := feed.Pass(ctx); err != nil || len(bot.sent) != 2 {
		t.Errorf("repeat pass: %v %q", err, bot.sent)
	}
}

func TestFeedOnEmptyBlog(t *testing.T) {
	posts := &fakePosts{}
	store := &fakeStore{seen: map[string]bool{}, subs: map[int64]bool{42: true}}
	bot := &fakeBot{}
	feed := &blogfeed.Feed{Posts: posts, Store: store, Bot: bot, Log: slog.New(slog.DiscardHandler)}
	_ = feed.Pass(context.Background())
	// The first post after an empty start goes out.
	posts.posts = []api.BlogPost{{ID: "first", Title: "Первая"}}
	_ = feed.Pass(context.Background())
	if len(bot.sent) != 1 {
		t.Errorf("sent: %q", bot.sent)
	}
}

func newFeed(posts *fakePosts, subs ...int64) (*blogfeed.Feed, *fakeStore, *fakeBot) {
	store := &fakeStore{seen: map[string]bool{"__start__": true, "__start__:ky": true, "__start__:en": true, "__start__:de": true},
		subs: map[int64]bool{}}
	for _, id := range subs {
		store.subs[id] = true
	}
	bot := &fakeBot{}
	return &blogfeed.Feed{Posts: posts, Store: store, Bot: bot, Log: slog.New(slog.DiscardHandler),
		Links: screens.Links{}, Interval: time.Hour}, store, bot
}

// Posts in each language go to the chats reading the bot in it; a language
// the feed meets for the first time (it learnt languages later) starts
// quietly.
func TestFeedLanguages(t *testing.T) {
	posts := &fakePosts{
		posts:  []api.BlogPost{{ID: "ru1", Title: "Новая"}},
		byLang: map[string][]api.BlogPost{"en": {{ID: "en1", Title: "Fresh"}}, "de": {{ID: "de-old", Title: "Alt"}}},
	}
	feed, store, bot := newFeed(posts, 42, 8, 9)
	delete(store.seen, "__start__:de")
	store.langs = map[int64]string{8: "en", 9: "de"}
	if err := feed.Pass(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(bot.sent) != 2 || bot.chats[0] != 42 || bot.chats[1] != 8 ||
		!strings.HasPrefix(bot.sent[1], "text 📰 <b>New blog post</b>\n\n<b>Fresh</b>") {
		t.Errorf("sent %q to %v", bot.sent, bot.chats)
	}
	if !store.seen["de-old"] || !store.seen["__start__:de"] {
		t.Errorf("German posts not remembered: %v", store.seen)
	}
}

// Telegram couldn't fetch the cover: the post goes as text.
func TestFeedPhotoFallback(t *testing.T) {
	feed, _, bot := newFeed(&fakePosts{posts: []api.BlogPost{{ID: "a", Title: "А", CoverImageURL: cover("https://x/a.jpg")}}}, 42)
	bot.badPhotos = true
	if err := feed.Pass(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(bot.sent) != 1 || !strings.HasPrefix(bot.sent[0], "text 📰") {
		t.Errorf("sent: %q", bot.sent)
	}
}

// A burst of posts: the latest three go out, all are marked seen.
func TestFeedLimit(t *testing.T) {
	var list []api.BlogPost
	for _, id := range []string{"e", "d", "c", "b", "a"} { // newest first
		list = append(list, api.BlogPost{ID: id, Title: strings.ToUpper(id)})
	}
	feed, store, bot := newFeed(&fakePosts{posts: list}, 42)
	if err := feed.Pass(context.Background()); err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, m := range bot.sent {
		titles = append(titles, strings.Split(m, "<b>")[2][:1])
	}
	if strings.Join(titles, "") != "CDE" || !store.seen["a"] || !store.seen["b"] {
		t.Errorf("sent %v, seen %v", titles, store.seen)
	}
}

// A failure that isn't a block keeps the subscription.
func TestFeedKeepsFlakyChats(t *testing.T) {
	feed, store, _ := newFeed(&fakePosts{posts: []api.BlogPost{{ID: "a", Title: "А"}}}, 7)
	if err := feed.Pass(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !store.subs[7] {
		t.Error("a chat with a temporary error was unsubscribed")
	}
}

type failingPosts struct{}

func (failingPosts) BlogPosts(context.Context, string) ([]api.BlogPost, error) {
	return nil, errors.New("gateway down")
}

func TestFeedRun(t *testing.T) {
	feed, _, _ := newFeed(nil)
	feed.Posts = failingPosts{}
	if err := feed.Pass(context.Background()); err == nil {
		t.Error("an API failure was swallowed")
	}
	// Run makes a pass at once and stops with its context.
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { feed.Run(ctx); close(done) }()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run didn't stop")
	}
}
