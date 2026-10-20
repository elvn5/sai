// Command bot is Ermekten's Telegram bot: greets users, links Telegram to
// site accounts, confirms phone registrations, edits profiles and shows
// support contacts; shows the user's outings, tours and the blog, and
// sends new blog posts to subscribers. Accounts live in the users service,
// outings, tours and posts behind the gateway.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/blogfeed"
	"github.com/elvn5/sai/services/bot/internal/config"
	"github.com/elvn5/sai/services/bot/internal/handlers"
	"github.com/elvn5/sai/services/bot/internal/migrations"
	"github.com/elvn5/sai/services/bot/internal/screens"
	"github.com/elvn5/sai/services/bot/internal/session"
	"github.com/elvn5/sai/services/bot/internal/store"
	"github.com/elvn5/sai/services/bot/internal/users"
	pkgconfig "github.com/elvn5/sai/services/pkg/config"
	"github.com/elvn5/sai/services/pkg/database"
	"github.com/elvn5/sai/services/pkg/logger"
)

const (
	// shutdownTimeout bounds waiting for updates in progress on shutdown.
	shutdownTimeout = 15 * time.Second
	// How often the blog feed looks for new posts.
	blogFeedInterval = 10 * time.Minute
)

// commands is the bot's command menu.
var commands = []telego.BotCommand{
	{Command: "outings", Description: "Мои выходы"},
	{Command: "tours", Description: "Туры: избранное и ближайшие"},
	{Command: "blog", Description: "Статьи блога"},
	{Command: "notifications", Description: "Что присылать"},
	{Command: "profile", Description: "Профиль"},
	{Command: "support", Description: "Поддержка"},
}

func main() {
	log := logger.New("bot")
	if err := run(log); err != nil {
		log.Error("bot stopped", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := pkgconfig.Load[config.Config]()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	accounts, err := users.New(cfg.UsersGRPCAddr)
	if err != nil {
		return err
	}
	defer func() { _ = accounts.Close() }()

	pool, err := database.Connect(ctx, cfg.DSN(), migrations.FS, log)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer pool.Close()
	db := store.New(pool)

	var content *api.Client
	if cfg.APIURL != "" && cfg.JWTSecret != "" {
		if content, err = api.New(cfg.APIURL, cfg.JWTSecret); err != nil {
			return err
		}
	} else {
		log.Warn("API_URL or JWT_SECRET unset: outings, tours and blog sections are off")
	}
	links := screens.Links{Site: strings.TrimRight(cfg.Site(), "/"), PublicAPI: strings.TrimRight(cfg.PublicAPIURL, "/")}

	options := []telego.BotOption{telego.WithLogger(botLogger{log: log, token: cfg.BotToken})}
	if cfg.TelegramAPIURL != "" {
		options = append(options, telego.WithAPIServer(cfg.TelegramAPIURL))
	}
	bot, err := telego.NewBot(cfg.BotToken, options...)
	if err != nil {
		return fmt.Errorf("telegram bot: %w", err)
	}

	if cfg.MiniAppURL != "" {
		err := bot.SetChatMenuButton(ctx, &telego.SetChatMenuButtonParams{
			MenuButton: &telego.MenuButtonWebApp{
				Type:   telego.ButtonTypeWebApp,
				Text:   "Ermekten",
				WebApp: telego.WebAppInfo{URL: cfg.MiniAppURL},
			},
		})
		if err != nil {
			log.Warn("could not set the Mini App menu button", slog.Any("error", err))
		}
	}

	if content != nil {
		if err := bot.SetMyCommands(ctx, &telego.SetMyCommandsParams{Commands: commands}); err != nil {
			log.Warn("could not set the command menu", slog.Any("error", err))
		}
	}

	updates, err := bot.UpdatesViaLongPolling(ctx, nil)
	if err != nil {
		return fmt.Errorf("long polling: %w", err)
	}
	bh, err := th.NewBotHandler(bot, updates)
	if err != nil {
		return fmt.Errorf("bot handler: %w", err)
	}
	h := &handlers.Handlers{Users: accounts, Sessions: session.New(), MiniAppURL: cfg.MiniAppURL, Log: log, Links: links}
	if content != nil {
		h.Content, h.Subscribers = content, db
		feed := &blogfeed.Feed{Posts: content, Store: db, Bot: bot, Links: links, Log: log, Interval: blogFeedInterval}
		go feed.Run(ctx)
	}
	h.Register(bh)

	errCh := make(chan error, 1)
	go func() { errCh <- bh.Start() }()
	log.Info("bot polling for updates")

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	log.Info("shutting down")
	// Long polling ends with ctx; wait for the updates already taken.
	stopCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := bh.StopWithContext(stopCtx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return nil
}

// botLogger routes Telego's logs to slog, without the bot token (it is
// part of every Bot API URL).
type botLogger struct {
	log   *slog.Logger
	token string
}

func (l botLogger) redact(format string, args []any) string {
	return strings.ReplaceAll(fmt.Sprintf(format, args...), l.token, "<token>")
}

func (l botLogger) Debugf(format string, args ...any) { l.log.Debug(l.redact(format, args)) }

func (l botLogger) Errorf(format string, args ...any) { l.log.Error(l.redact(format, args)) }
