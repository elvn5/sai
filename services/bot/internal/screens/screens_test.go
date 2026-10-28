package screens_test

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/i18n"
	"github.com/elvn5/sai/services/bot/internal/screens"
)

var (
	now   = time.Date(2026, 10, 20, 4, 0, 0, 0, time.UTC) // 10:00 in Bishkek
	links = screens.Links{PublicAPI: "https://api.ermekten.kg", Bot: "ermekten_bot"}
)

func str(s string) *string { return &s }
func num(n int) *int       { return &n }

// buttons flattens a screen's keyboard to "text→data|url" lines.
func buttons(s screens.Screen) []string {
	var out []string
	if s.Markup == nil {
		return out
	}
	for _, row := range s.Markup.InlineKeyboard {
		for _, b := range row {
			target := b.CallbackData + b.URL
			if b.WebApp != nil {
				target = "webapp:" + b.WebApp.URL
			}
			out = append(out, b.Text+"→"+target)
		}
	}
	return out
}

func has(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// trip is a tour starting on day (YYYY-MM-DD) at 10:00 in Bishkek, with
// 7 of 10 places taken.
func trip(id, title, day string) api.Tour {
	return api.Tour{ID: id, Title: title, StartDate: day, EndDate: day, MeetingTime: str("10:00"),
		MeetingPoint: str("ТЦ «Вефа»"), Capacity: num(10), Taken: 7}
}

func TestMyToursSplitsUpcomingAndPast(t *testing.T) {
	soon := trip("t1", "Пик Учитель", "2026-10-24")
	later := trip("t2", "Водопад <Барскоон>", "2026-11-01")
	later.Capacity = nil
	done := trip("t3", "Ала-Кёль", "2026-10-10")
	list := []api.Tour{later, done, soon}

	s := screens.MyTours(i18n.RU, list, screens.FromMine, now)
	for _, want := range []string{
		"🎒 <b>Мои туры</b>\nЗа всё время: 1 тур\n\n<b>Предстоящие</b>",
		"<b>1. Пик Учитель</b>\nсб, 24 окт., 10:00 · 👥 7 из 10 мест",
		"<b>2. Водопад &lt;Барскоон&gt;</b>\nвс, 1 нояб., 10:00 · 👥 записались: 7",
	} {
		if !strings.Contains(s.Text, want) {
			t.Errorf("text lacks %q:\n%s", want, s.Text)
		}
	}
	if strings.Contains(s.Text, "Ала-Кёль") {
		t.Errorf("a past tour among upcoming:\n%s", s.Text)
	}
	got := buttons(s)
	for _, want := range []string{
		"1 · Пик Учитель→o:view:u:t1", "2 · Водопад <Барскоон>→o:view:u:t2",
		"• Предстоящие •→o:list:u", "Прошедшие→o:list:p", "🧭 Все туры→t:n:0", "🧑‍✈️ Кабинет организатора→g:home",
	} {
		if !has(got, want) {
			t.Errorf("buttons lack %q: %v", want, got)
		}
	}

	past := screens.MyTours(i18n.RU, list, screens.FromPast, now)
	if !strings.Contains(past.Text, "<b>1. Ала-Кёль</b>\nсб, 10 окт., 10:00") || !has(buttons(past), "1 · Ала-Кёль→o:view:p:t3") {
		t.Errorf("past: %v\n%s", buttons(past), past.Text)
	}
}

func TestMyToursEmpty(t *testing.T) {
	s := screens.MyTours(i18n.RU, nil, screens.FromMine, now)
	if !strings.Contains(s.Text, "Вы пока не записаны ни на один тур.") || strings.Contains(s.Text, "За всё время") {
		t.Errorf("empty:\n%s", s.Text)
	}
	if past := screens.MyTours(i18n.RU, nil, screens.FromPast, now); !strings.HasSuffix(past.Text, "Прошедших туров пока нет.") {
		t.Errorf("no past:\n%s", past.Text)
	}
}

func TestTourCardActions(t *testing.T) {
	o := trip("t1", "Пик Учитель", "2026-10-24")
	o.EndDate, o.Price, o.Telegram = "2026-10-25", num(500), str("aigul_trek")
	o.Organizer = &api.Person{FirstName: str("Айгуль")}
	o.IsJoined = true

	s := screens.TourCard(i18n.RU, o, screens.FromTours, links, now)
	want := "🧭 <b>Пик Учитель</b>\n📅 24 — 25 октября\n\n🕕 Сбор: сб, 24 окт., 10:00\n📍 ТЦ «Вефа»\n" +
		"💳 500 сом с человека\n👥 ▰▰▰▰▰▰▰▱▱▱ 7 из 10 мест\n🧑 Организатор: Айгуль\n\n📞 <b>Связь с организатором</b>\nTelegram: @aigul_trek\n\n✅ Вы записаны"
	if s.Text != want {
		t.Errorf("card:\n%s\nwant:\n%s", s.Text, want)
	}
	if got := buttons(s); strings.Join(got, ",") != "🙅 Отменить запись→o:ask:t:t1,"+
		"📤 Позвать друзей→https://t.me/share/url?url="+url.QueryEscape("https://t.me/ermekten_bot?start=tour_t1")+
		"&text="+url.QueryEscape("Поехали в тур: «Пик Учитель», сб, 24 окт., 10:00")+","+
		"⬅️ К турам→t:n:0" {
		t.Errorf("joined buttons: %v", got)
	}

	o.IsJoined = false
	stranger := screens.TourCard(i18n.RU, o, screens.FromMine, links, now)
	if !has(buttons(stranger), "✋ Записаться→o:join:u:t1") || strings.Contains(stranger.Text, "aigul_trek") {
		t.Errorf("not joined: %v\n%s", buttons(stranger), stranger.Text)
	}
	o.Taken = 10
	full := screens.TourCard(i18n.RU, o, screens.FromMine, links, now)
	if got := strings.Join(buttons(full), ","); !strings.Contains(got, "⏳ Встать в лист ожидания→o:join:u:t1") || strings.Contains(got, "Позвать") ||
		!strings.HasSuffix(full.Text, "Мест больше нет — можно встать в лист ожидания") {
		t.Errorf("full: %v\n%s", buttons(full), full.Text)
	}
	o.MyStatus = "waitlist"
	waiting := screens.TourCard(i18n.RU, o, screens.FromMine, links, now)
	if got := strings.Join(buttons(waiting), ","); !strings.Contains(got, "🙅 Выйти из листа ожидания→o:leave:u:t1") ||
		!strings.HasSuffix(waiting.Text, "⏳ Вы в листе ожидания — напишем, если освободится место") {
		t.Errorf("waiting: %v\n%s", buttons(waiting), waiting.Text)
	}
	o.MyStatus = ""
	o.Taken, o.IsOrganizer = 3, true
	o.Roster = []api.RosterEntry{{Person: api.Person{FirstName: str("Аня")}}, {Person: api.Person{Username: str("bob")}}, {}}
	own := screens.TourCard(i18n.RU, o, screens.FromMine, links, now)
	if got := buttons(own); len(got) != 3 || got[0] != "📋 Участники и отметка→g:r:t1" ||
		!strings.HasSuffix(own.Text, "👥 Едут: Аня, @bob и ещё 1\n\n⭐ Вы организатор") {
		t.Errorf("organizer: %v\n%s", got, own.Text)
	}

	o.IsOrganizer = false
	confirm := screens.LeaveConfirm(i18n.RU, o, screens.FromMine)
	if got := buttons(confirm); strings.Join(got, ",") != "Да, отменить→o:leave:u:t1,Нет, еду→o:view:u:t1" {
		t.Errorf("confirm: %v", got)
	}
}

func TestTourCardAfter(t *testing.T) {
	// Past: nothing to do but look, and rate it when it may be rated.
	past := trip("t2", "Ала-Кёль", "2026-10-18")
	past.IsJoined = true
	ps := screens.TourCard(i18n.RU, past, screens.FromPast, links, now)
	if !strings.HasSuffix(ps.Text, "🏁 Тур уже прошёл") {
		t.Errorf("past:\n%s", ps.Text)
	}
	if got := buttons(ps); strings.Join(got, ",") != "⬅️ К моим турам→o:list:p" {
		t.Errorf("past buttons: %v", got)
	}
	past.CanRate = true
	if got := buttons(screens.TourCard(i18n.RU, past, screens.FromPast, links, now)); !has(got, "⭐ Оценить тур→r:s:t2") {
		t.Errorf("rate button: %v", got)
	}
	// Met less than 3 hours ago: still upcoming.
	late := trip("t3", "X", "2026-10-20")
	late.MeetingTime = str("08:00")
	if s := screens.TourCard(i18n.RU, late, screens.FromMine, links, now); !has(buttons(s), "✋ Записаться→o:join:u:t3") {
		t.Errorf("grace period: %v", buttons(s))
	}
}

func TestMyToursMoreThanShown(t *testing.T) {
	var list []api.Tour
	for i := range 10 {
		list = append(list, trip("t"+string(rune('a'+i)), "Тур", "2026-11-0"+strconv.Itoa(i)))
	}
	list[0].StartDate = "2026-11-10"
	s := screens.MyTours(i18n.RU, list, screens.FromMine, now)
	if !strings.HasSuffix(s.Text, "…и ещё 2.") || len(s.Markup.InlineKeyboard) != 8+3 {
		t.Errorf("10 tours: %d rows\n%s", len(s.Markup.InlineKeyboard), s.Text)
	}
}

func TestToursCarousel(t *testing.T) {
	tours := []api.Tour{
		{ID: "t1", Slug: "a", Title: "A"},
		{ID: "t2", Slug: "son-kol", Locale: "en", Title: "Сон-Кёль: ночь в юрте", Location: str("Нарын"),
			DurationDays: num(3), StartDate: "2027-06-12", EndDate: "2027-06-14", Price: num(18000),
			RatingAverage: 4.8, ReviewCount: 23, CoverImageURL: str("/uploads/tours/son-kol.jpg")},
	}
	s := screens.Tours(i18n.RU, screens.ToursView{Tab: screens.TabFavorites, Index: 1, Tours: tours, Favorites: map[string]bool{"t2": true}}, links)
	want := "🧭 <b>Туры</b> · Избранное 2 из 2\n\n<b>Сон-Кёль: ночь в юрте</b>\n📍 Нарын · 3 дня\n📅 12 — 14 июня\n" +
		"💳 18 000 сом с человека\n⭐ 4.8 · 23 отзыва"
	if s.Text != want {
		t.Errorf("tour:\n%s", s.Text)
	}
	if s.Preview == nil || s.Preview.URL != "https://api.ermekten.kg/images/1280/uploads/tours/son-kol.jpg" || !s.Preview.ShowAboveText {
		t.Errorf("preview: %+v", s.Preview)
	}
	got := buttons(s)
	for _, w := range []string{
		"◀️→t:f:0", "2 / 2→noop", "📖 Подробнее→t:more:f:1:0", "💔 Из избранного→t:drop:f:1:t2",
		"• ❤️ Избранное •→t:f:0", "🆕 Ближайшие→t:n:0", "🔔 Сообщать о новых турах→t:alert:f:1",
	} {
		if !has(got, w) {
			t.Errorf("buttons lack %q: %v", w, got)
		}
	}

	first := screens.Tours(i18n.RU, screens.ToursView{Tab: screens.TabUpcoming, Tours: tours, AlertOn: true}, links)
	if g := buttons(first); !has(g, "❤️ В избранное→t:save:n:0:t1") || !has(g, "🔕 Не сообщать о новых турах→t:alert:n:0") {
		t.Errorf("upcoming: %v", g)
	}
	if !first.Preview.IsDisabled {
		t.Errorf("no cover, yet a preview: %+v", first.Preview)
	}
	empty := screens.Tours(i18n.RU, screens.ToursView{Tab: screens.TabFavorites}, links)
	if !strings.Contains(empty.Text, "В избранном пока пусто") {
		t.Errorf("empty:\n%s", empty.Text)
	}
}

func TestBlog(t *testing.T) {
	published := time.Date(2026, 10, 18, 6, 0, 0, 0, time.UTC)
	posts := []api.BlogPost{{ID: "p1", Slug: "checklist", Title: "Что взять в поход", Excerpt: str("Рюкзак & палки"),
		PublishedAt: &published, CoverImageURL: str("https://cdn.example.com/a.jpg")}}
	s := screens.Blog(i18n.RU, posts, 0, false, links)
	if s.Text != "📰 <b>Блог Ermekten</b> · 1 из 1\n\n<b>Что взять в поход</b>\n<i>18 окт. 2026</i>\n\nРюкзак &amp; палки" {
		t.Errorf("blog:\n%s", s.Text)
	}
	if s.Preview.URL != "https://cdn.example.com/a.jpg" {
		t.Errorf("preview: %+v", s.Preview)
	}
	if got := buttons(s); strings.Join(got, ",") != "📖 Читать статью→b:read:p1:0,🔔 Присылать новые статьи→b:sub:0" {
		t.Errorf("buttons: %v", got)
	}
	if !has(buttons(screens.Blog(i18n.RU, posts, 0, true, links)), "🔕 Не присылать новые статьи→b:unsub:0") {
		t.Error("no unsubscribe button")
	}

	caption, markup := screens.BlogPost(i18n.RU, posts[0])
	raw, _ := json.Marshal(markup)
	if caption != "📰 <b>Новая статья в блоге</b>\n\n<b>Что взять в поход</b>\nРюкзак &amp; палки" ||
		!strings.Contains(string(raw), `"callback_data":"n:blog_off"`) {
		t.Errorf("post: %q %s", caption, raw)
	}
}

func TestNotifications(t *testing.T) {
	s := screens.Notifications(i18n.RU, screens.NotifySettings{ToursAll: true})
	if got := buttons(s); strings.Join(got, ",") != "🧭 Новые туры: ✅ вкл→n:tours,📰 Статьи блога: ⬜ выкл→n:blog" {
		t.Errorf("buttons: %v", got)
	}
	partial := screens.Notifications(i18n.RU, screens.NotifySettings{ToursDestinations: 2, Blog: true})
	if !strings.Contains(partial.Text, "по 2 направлениям, выбранным на сайте") ||
		!has(buttons(partial), "🧭 Новые туры: включить все→n:tours") || !has(buttons(partial), "📰 Статьи блога: ✅ вкл→n:blog") {
		t.Errorf("partial: %v\n%s", buttons(partial), partial.Text)
	}
}

func TestPicture(t *testing.T) {
	for src, want := range map[string]string{
		"":                       "",
		"/uploads/a.jpg":         "https://api.ermekten.kg/images/1280/uploads/a.jpg",
		"https://r2.example/a":   "https://r2.example/a",
		"data:image/png;base64,": "",
	} {
		if got := links.Picture(&src); got != want {
			t.Errorf("Picture(%q) = %q, want %q", src, got, want)
		}
	}
	if got := (screens.Links{}).Picture(str("/uploads/a.jpg")); got != "" {
		t.Errorf("without a public API: %q", got)
	}
}

func TestToursEdges(t *testing.T) {
	one := []api.Tour{{ID: "t1", Slug: "a", Title: "A", StartDate: "bad"}}
	// Past the end: the last tour; one tour: no pager; no dates line for a bad date.
	s := screens.Tours(i18n.RU, screens.ToursView{Tab: screens.TabUpcoming, Index: 9, Tours: one}, links)
	if s.Text != "🧭 <b>Туры</b> · Ближайшие 1 из 1\n\n<b>A</b>" || has(buttons(s), "1 / 1→noop") {
		t.Errorf("one tour: %q %v", s.Text, buttons(s))
	}
	up := screens.Tours(i18n.RU, screens.ToursView{Tab: screens.TabUpcoming}, links)
	if !strings.Contains(up.Text, "Ближайших туров пока нет") || len(up.Markup.InlineKeyboard) != 2 {
		t.Errorf("no upcoming: %v\n%s", buttons(up), up.Text)
	}
}

func TestBlogEdges(t *testing.T) {
	s := screens.Blog(i18n.RU, nil, 3, true, links)
	if !strings.Contains(s.Text, "Статей пока нет") || strings.Join(buttons(s), ",") != "🔕 Не присылать новые статьи→b:unsub:0" {
		t.Errorf("empty blog: %v\n%s", buttons(s), s.Text)
	}
	en := api.BlogPost{ID: "p", Slug: "x", Locale: "en", Title: "T"}
	if caption, _ := screens.BlogPost(i18n.RU, en); caption != "📰 <b>Новая статья в блоге</b>\n\n<b>T</b>" {
		t.Errorf("no excerpt: %q", caption)
	}
	if !has(buttons(screens.Blog(i18n.RU, []api.BlogPost{en}, 0, false, links)), "📖 Читать статью→b:read:p:0") {
		t.Error("no read button")
	}
}

// Everything is done in the chat: no section button opens the Mini App
// or the site — only Telegram's own share dialog.
func TestNoMiniAppButtons(t *testing.T) {
	o := trip("o1", "Пик", "2026-10-22")
	own := o
	own.IsOrganizer = true
	tour := api.Tour{ID: "t1", Slug: "son-kol", Locale: "ky", Title: "Сон-Кёль", Content: `<p>Текст</p>`, StartDate: "2026-11-01"}
	post := api.BlogPost{ID: "p1", Slug: "checklist", Title: "Чек-лист", Content: `<p>Текст</p>`}
	_, pushMarkup := screens.BlogPost(i18n.RU, post)

	for _, s := range []screens.Screen{
		screens.MyTours(i18n.RU, []api.Tour{o}, screens.FromMine, now),
		screens.TourCard(i18n.RU, o, screens.FromMine, links, now), screens.TourCard(i18n.RU, own, screens.FromMine, links, now),
		screens.Tours(i18n.RU, screens.ToursView{Tab: screens.TabUpcoming, Tours: []api.Tour{tour}}, links),
		screens.TourMore(i18n.RU, tour, screens.TabUpcoming, 0, 0, false, links, now),
		screens.Blog(i18n.RU, []api.BlogPost{post}, 0, false, links), screens.Article(i18n.RU, post, 0, 0, links),
		{Markup: pushMarkup},
	} {
		for _, row := range s.Markup.InlineKeyboard {
			for _, b := range row {
				if b.WebApp != nil || (b.URL != "" && !strings.HasPrefix(b.URL, "https://t.me/share/url?")) {
					t.Errorf("%q leaves the chat: %+v", b.Text, b)
				}
			}
		}
	}
}

// Telegram takes up to 64 bytes of callback data; ids are UUIDs.
func TestCallbackDataFits(t *testing.T) {
	const id = "0f8fad5b-d9cb-469f-a165-70867728950e"
	o := trip(id, "Пик", "2026-10-22")
	o.IsOrganizer = true
	past := trip(id, "Пик", "2026-10-18")
	past.IsJoined, past.CanRate = true, true
	tours := []api.Tour{{ID: id, Slug: "x"}}
	for i := range 20 {
		tours = append(tours, api.Tour{ID: id, Slug: "x", Title: strconv.Itoa(i)})
	}
	post := api.BlogPost{ID: id, Content: strings.Repeat("<p>"+strings.Repeat("слово ", 600)+"</p>", 5)}
	for _, s := range []screens.Screen{
		screens.TourCard(i18n.RU, o, screens.FromTours, links, now), screens.LeaveConfirm(i18n.RU, o, screens.FromNotice),
		screens.TourCard(i18n.RU, past, screens.FromPast, links, now), screens.Roster(i18n.RU, o, nil, links, now),
		screens.Tours(i18n.RU, screens.ToursView{Tab: screens.TabFavorites, Index: 19, Tours: tours, Favorites: map[string]bool{id: true}}, links),
		screens.TourMore(i18n.RU, tours[19], screens.TabFavorites, 19, 0, true, links, now),
		screens.Article(i18n.RU, post, 99, 3, links),
	} {
		for _, row := range s.Markup.InlineKeyboard {
			for _, b := range row {
				if len(b.CallbackData) > 64 {
					t.Errorf("%q: %d bytes: %s", b.Text, len(b.CallbackData), b.CallbackData)
				}
			}
		}
	}
}
