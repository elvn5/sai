package screens

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mymmrac/telego"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/keyboards"
)

// Blog callback actions: "b:<index>" shows a post, "b:sub:<index>" and
// "b:unsub:<index>" change the subscription and show it again.
const (
	ActSubscribe   = "sub"
	ActUnsubscribe = "unsub"
)

// Notification settings callbacks.
const (
	NotifyTours = keyboards.NotifyPrefix + "tours"
	NotifyBlog  = keyboards.NotifyPrefix + "blog"
	// "Don't send posts" under a post that came as a notification.
	NotifyBlogOff = keyboards.NotifyPrefix + "blog_off"
)

func BlogData(index int) string { return keyboards.BlogPrefix + strconv.Itoa(index) }

func blogAction(action string, index int) string {
	return keyboards.BlogPrefix + action + ":" + strconv.Itoa(index)
}

func postPath(p api.BlogPost) string {
	path := "/blog/" + p.Slug
	if p.Locale != "" && p.Locale != "ru" {
		path = "/" + p.Locale + path
	}
	return path
}

// Blog shows one post of the blog, newest first, with its cover.
func Blog(posts []api.BlogPost, index int, subscribed bool, links Links) Screen {
	n := len(posts)
	var b strings.Builder
	b.WriteString("📰 <b>Блог Ermekten</b>")
	var buttons [][]telego.InlineKeyboardButton
	picture := ""
	if n == 0 {
		index = 0
		b.WriteString("\n\nСтатей пока нет. Подпишитесь — пришлём первую, как только она выйдет.")
	} else {
		index = min(max(index, 0), n-1)
		p := posts[index]
		picture = links.Picture(p.CoverImageURL)
		fmt.Fprintf(&b, " · %d из %d\n\n<b>%s</b>", index+1, n, esc(p.Title))
		if p.PublishedAt != nil {
			b.WriteString("\n<i>" + day(*p.PublishedAt) + "</i>")
		}
		if p.Excerpt != nil && *p.Excerpt != "" {
			b.WriteString("\n\n" + esc(clip(*p.Excerpt, 400)))
		}
		if n > 1 {
			buttons = append(buttons, keyboards.Pager(index, n, BlogData))
		}
		buttons = append(buttons, []telego.InlineKeyboardButton{links.Open("📖 Читать статью", postPath(p))})
	}
	sub := keyboards.Button("🔔 Присылать новые статьи", blogAction(ActSubscribe, index))
	if subscribed {
		sub = keyboards.Button("🔕 Не присылать новые статьи", blogAction(ActUnsubscribe, index))
	}
	buttons = append(buttons, []telego.InlineKeyboardButton{sub})
	return Screen{Text: b.String(), Markup: rows(buttons...), Preview: coverPreview(picture)}
}

// BlogPost is a new post as a notification: the caption of its cover
// photo (or a message with a preview when it has none) and its buttons.
func BlogPost(p api.BlogPost, links Links) (caption string, markup *telego.InlineKeyboardMarkup) {
	var b strings.Builder
	b.WriteString("📰 <b>Новая статья в блоге</b>\n\n<b>" + esc(p.Title) + "</b>")
	if p.Excerpt != nil && *p.Excerpt != "" {
		// A photo caption holds 1024 characters.
		b.WriteString("\n" + esc(clip(*p.Excerpt, 600)))
	}
	return b.String(), rows([]telego.InlineKeyboardButton{
		links.Open("📖 Читать", postPath(p)),
		keyboards.Button("🔕 Не присылать", NotifyBlogOff),
	})
}

// NotifySettings is what the notification settings show.
type NotifySettings struct {
	// Every new tour.
	ToursAll bool
	// Destinations followed on the site (without "any").
	ToursDestinations int
	Blog              bool
}

// Notifications lets the user pick what the bot writes about.
func Notifications(s NotifySettings) Screen {
	tours, toursButton := "выключено", "🧭 Новые туры: ⬜ выкл"
	switch {
	case s.ToursAll:
		tours, toursButton = "✅ включено", "🧭 Новые туры: ✅ вкл"
	case s.ToursDestinations > 0:
		tours = fmt.Sprintf("по %d %s, выбранным на сайте", s.ToursDestinations,
			plural(s.ToursDestinations, "направлению", "направлениям", "направлениям"))
		toursButton = "🧭 Новые туры: включить все"
	}
	blog, blogButton := "выключено", "📰 Статьи блога: ⬜ выкл"
	if s.Blog {
		blog, blogButton = "✅ включено", "📰 Статьи блога: ✅ вкл"
	}
	text := "🔔 <b>Уведомления</b>\n\nВыберите, о чём вам писать.\n\n" +
		"🥾 <b>Мои выходы</b> — всегда\nКто записался, изменения, отмена, напоминание перед сбором.\n\n" +
		"🧭 <b>Новые туры</b> — " + tours + "\nКогда на сайте появляется новый тур.\n\n" +
		"📰 <b>Статьи блога</b> — " + blog + "\nНовая статья — одним сообщением с обложкой."
	return Screen{Text: text, Preview: noPreview(), Markup: rows(
		[]telego.InlineKeyboardButton{keyboards.Button(toursButton, NotifyTours)},
		[]telego.InlineKeyboardButton{keyboards.Button(blogButton, NotifyBlog)},
	)}
}

// Welcome greets the user; next is their nearest outing, if any.
func Welcome(name string, next *api.Outing) string {
	var b strings.Builder
	b.WriteString("🏔 <b>Здравствуйте, " + esc(name) + "!</b>\n\n" +
		"Я бот Ermekten — помогу не пропустить выходы в горы и туры по Кыргызстану.\n\n" +
		"🥾 <b>Мои выходы</b> — куда вы идёте и когда сбор\n" +
		"🧭 <b>Туры</b> — избранное и ближайшие даты\n" +
		"📰 <b>Блог</b> — свежие статьи и маршруты\n" +
		"🔔 <b>Уведомления</b> — что присылать")
	if next != nil {
		b.WriteString("\n\nБлижайший выход: <b>" + esc(next.Title) + "</b>, " + When(next.StartsAt))
	}
	b.WriteString("\n\nВыберите раздел в меню ниже.")
	return b.String()
}
