package screens

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mymmrac/telego"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/i18n"
	"github.com/elvn5/sai/services/bot/internal/keyboards"
)

// Blog callback actions: "b:<index>" shows a post, "b:sub:<index>" and
// "b:unsub:<index>" change the subscription and show it again,
// "b:read:<post id>:<page>" reads a post.
const (
	ActSubscribe   = "sub"
	ActUnsubscribe = "unsub"
	ActRead        = "read"
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

// ReadData opens a page of a post.
func ReadData(postID string, page int) string {
	return keyboards.BlogPrefix + ActRead + ":" + postID + ":" + strconv.Itoa(page)
}

// Blog shows one post of the blog, newest first, with its cover.
func Blog(l i18n.Lang, posts []api.BlogPost, index int, subscribed bool, links Links) Screen {
	n := len(posts)
	var b strings.Builder
	b.WriteString(l.T("📰 <b>Блог Ermekten</b>"))
	var buttons [][]telego.InlineKeyboardButton
	picture := ""
	if n == 0 {
		index = 0
		b.WriteString("\n\n" + l.T("Статей пока нет. Подпишитесь — пришлём первую, как только она выйдет."))
	} else {
		index = min(max(index, 0), n-1)
		p := posts[index]
		picture = links.Picture(p.CoverImageURL)
		b.WriteString(" · " + l.F("%d из %d", index+1, n) + "\n\n<b>" + esc(p.Title) + "</b>")
		if p.PublishedAt != nil {
			b.WriteString("\n<i>" + day(l, *p.PublishedAt) + "</i>")
		}
		if p.Excerpt != nil && *p.Excerpt != "" {
			b.WriteString("\n\n" + esc(clip(*p.Excerpt, 400)))
		}
		if n > 1 {
			buttons = append(buttons, keyboards.Pager(index, n, BlogData))
		}
		buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button(l.T("📖 Читать статью"), ReadData(p.ID, 0))})
	}
	sub := keyboards.Button(l.T("🔔 Присылать новые статьи"), blogAction(ActSubscribe, index))
	if subscribed {
		sub = keyboards.Button(l.T("🔕 Не присылать новые статьи"), blogAction(ActUnsubscribe, index))
	}
	buttons = append(buttons, []telego.InlineKeyboardButton{sub})
	return Screen{Text: b.String(), Markup: rows(buttons...), Preview: coverPreview(picture)}
}

// BlogPost is a new post as a notification: the caption of its cover
// photo (or a message with a preview when it has none) and its buttons.
func BlogPost(l i18n.Lang, p api.BlogPost) (caption string, markup *telego.InlineKeyboardMarkup) {
	var b strings.Builder
	b.WriteString(l.T("📰 <b>Новая статья в блоге</b>") + "\n\n<b>" + esc(p.Title) + "</b>")
	if p.Excerpt != nil && *p.Excerpt != "" {
		// A photo caption holds 1024 characters.
		b.WriteString("\n" + esc(clip(*p.Excerpt, 600)))
	}
	return b.String(), rows([]telego.InlineKeyboardButton{
		keyboards.Button(l.T("📖 Читать"), ReadData(p.ID, 0)),
		keyboards.Button(l.T("🔕 Не присылать"), NotifyBlogOff),
	})
}

// Article is a page of a post, read in the chat: the cover above the
// first page, pages turned with buttons. index is the post's place in the
// blog, for the way back.
func Article(l i18n.Lang, p api.BlogPost, index, page int, links Links) Screen {
	pages := Pages(Paragraphs(p.Content), pageLimit)
	if len(pages) == 0 && p.Excerpt != nil {
		pages = Pages(PlainParagraphs(*p.Excerpt), pageLimit)
	}
	n := max(len(pages), 1)
	page = min(max(page, 0), n-1)

	var b strings.Builder
	b.WriteString("📰 <b>" + esc(p.Title) + "</b>")
	if n > 1 {
		fmt.Fprintf(&b, " · %d/%d", page+1, n)
	}
	picture := ""
	if page == 0 {
		picture = links.Picture(p.CoverImageURL)
		if p.PublishedAt != nil {
			b.WriteString("\n<i>" + day(l, *p.PublishedAt) + "</i>")
		}
	}
	if len(pages) > 0 {
		b.WriteString("\n\n" + pages[page])
	} else {
		b.WriteString("\n\n" + l.T("У статьи пока нет текста."))
	}

	var buttons [][]telego.InlineKeyboardButton
	if n > 1 {
		var row []telego.InlineKeyboardButton
		if page > 0 {
			row = append(row, keyboards.Button(l.T("◀️ Назад"), ReadData(p.ID, page-1)))
		}
		if page < n-1 {
			row = append(row, keyboards.Button(l.T("Дальше ▶️"), ReadData(p.ID, page+1)))
		}
		buttons = append(buttons, row)
	}
	buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button(l.T("⬅️ К статьям"), BlogData(index))})
	return Screen{Text: b.String(), Markup: rows(buttons...), Preview: coverPreview(picture)}
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
func Notifications(l i18n.Lang, s NotifySettings) Screen {
	tours, toursButton := l.T("выключено"), l.T("🧭 Новые туры: ⬜ выкл")
	switch {
	case s.ToursAll:
		tours, toursButton = l.T("✅ включено"), l.T("🧭 Новые туры: ✅ вкл")
	case s.ToursDestinations > 0:
		tours = l.F("по %d %s, выбранным на сайте", s.ToursDestinations,
			l.N(s.ToursDestinations, "направлению", "направлениям", "направлениям"))
		toursButton = l.T("🧭 Новые туры: включить все")
	}
	blog, blogButton := l.T("выключено"), l.T("📰 Статьи блога: ⬜ выкл")
	if s.Blog {
		blog, blogButton = l.T("✅ включено"), l.T("📰 Статьи блога: ✅ вкл")
	}
	text := l.T("🔔 <b>Уведомления</b>\n\nВыберите, о чём вам писать.") + "\n\n" +
		l.T("🎒 <b>Мои туры</b> — всегда\nКто записался, изменения, отмена, напоминание перед сбором, просьба оценить тур.") + "\n\n" +
		l.F("🧭 <b>Новые туры</b> — %s\nКогда на сайте появляется новый тур.", tours) + "\n\n" +
		l.F("📰 <b>Статьи блога</b> — %s\nНовая статья — одним сообщением с обложкой.", blog)
	return Screen{Text: text, Preview: noPreview(), Markup: rows(
		[]telego.InlineKeyboardButton{keyboards.Button(toursButton, NotifyTours)},
		[]telego.InlineKeyboardButton{keyboards.Button(blogButton, NotifyBlog)},
	)}
}

// Welcome greets the user; next is their nearest tour, if any.
func Welcome(l i18n.Lang, name string, next *api.Tour) string {
	var b strings.Builder
	b.WriteString(l.F("🏔 <b>Здравствуйте, %s!</b>", esc(name)) + "\n\n" +
		l.T("Я бот Ermekten — помогу не пропустить туры по Кыргызстану и записаться на них прямо в чате.") + "\n\n" +
		l.T("🎒 <b>Мои туры</b> — куда вы едете и когда сбор") + "\n" +
		l.T("🧭 <b>Туры</b> — избранное и ближайшие даты") + "\n" +
		l.T("📰 <b>Блог</b> — свежие статьи и маршруты") + "\n" +
		l.T("🔔 <b>Уведомления</b> — что присылать"))
	if next != nil {
		b.WriteString("\n\n" + l.F("Ближайший тур: <b>%s</b>, %s", esc(next.Title), When(l, next.StartsAt())))
	}
	b.WriteString("\n\n" + l.T("Выберите раздел в меню ниже."))
	return b.String()
}
