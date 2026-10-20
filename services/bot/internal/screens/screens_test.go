package screens_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/screens"
)

var (
	now   = time.Date(2026, 10, 20, 4, 0, 0, 0, time.UTC) // 10:00 in Bishkek
	links = screens.Links{Site: "https://ermekten.kg", PublicAPI: "https://api.ermekten.kg"}
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

func outing(id, title string, starts time.Time, status string) api.Outing {
	return api.Outing{ID: id, Title: title, StartsAt: starts, Status: status, Seats: 10, Taken: 7, MeetingPoint: "ТЦ «Вефа»"}
}

func TestMyOutingsSplitsUpcomingAndPast(t *testing.T) {
	soon := outing("o1", "Пик Учитель", now.Add(4*24*time.Hour), "open")
	soon.IsJoined = true
	later := outing("o2", "Водопад <Барскоон>", now.Add(12*24*time.Hour), "open")
	later.IsOrganizer = true
	done := outing("o3", "Ала-Кёль", now.Add(-10*24*time.Hour), "open")
	cancelled := outing("o4", "Сон-Кёль", now.Add(2*24*time.Hour), "cancelled")
	list := []api.Outing{later, done, soon, cancelled}
	sum := &api.OutingsSummary{Outings: 12, DistanceKm: 84.4, ElevationGain: 5200}

	s := screens.MyOutings(list, sum, screens.FromMine, now)
	for _, want := range []string{
		"🥾 <b>Мои выходы</b>\nЗа всё время: 12 выходов · 84 км · ↗ 5 200 м\n\n<b>Предстоящие</b>",
		"<b>1. Пик Учитель</b>\nсб, 24 окт., 10:00 · 👥 7/10 · вы идёте",
		"<b>2. Водопад &lt;Барскоон&gt;</b>\nвс, 1 нояб., 10:00 · 👥 7/10 · ⭐ вы организатор",
	} {
		if !strings.Contains(s.Text, want) {
			t.Errorf("text lacks %q:\n%s", want, s.Text)
		}
	}
	if strings.Contains(s.Text, "Ала-Кёль") || strings.Contains(s.Text, "Сон-Кёль") {
		t.Errorf("past outings among upcoming:\n%s", s.Text)
	}
	got := buttons(s)
	for _, want := range []string{
		"1 · Пик Учитель→o:view:u:o1", "2 · Водопад <Барскоон>→o:view:u:o2",
		"• Предстоящие •→o:list:u", "Прошедшие→o:list:p", "🔎 Найти выход→o:find:0",
	} {
		if !has(got, want) {
			t.Errorf("buttons lack %q: %v", want, got)
		}
	}

	past := screens.MyOutings(list, nil, screens.FromPast, now)
	if !strings.Contains(past.Text, "<b>1. Сон-Кёль</b>\nчт, 22 окт., 10:00 · ❌ отменён") ||
		!strings.Contains(past.Text, "<b>2. Ала-Кёль</b>") || strings.Contains(past.Text, "За всё время") {
		t.Errorf("past:\n%s", past.Text)
	}
	if !has(buttons(past), "1 · Сон-Кёль→o:view:p:o4") {
		t.Errorf("past buttons: %v", buttons(past))
	}
}

func TestMyOutingsEmpty(t *testing.T) {
	s := screens.MyOutings(nil, &api.OutingsSummary{}, screens.FromMine, now)
	if !strings.Contains(s.Text, "Вы пока не записаны ни на один выход.") || strings.Contains(s.Text, "За всё время") {
		t.Errorf("empty:\n%s", s.Text)
	}
}

func TestOutingCardActions(t *testing.T) {
	o := outing("o1", "Пик Учитель", now.Add(4*24*time.Hour), "open")
	returns := o.StartsAt.Add(9 * time.Hour)
	o.ReturnsAt, o.Trail, o.Level, o.Transport = &returns, &api.TrailRef{Name: "Ала-Арча"}, "intermediate", "car"
	o.CostPerPerson, o.Organizer, o.Contact = num(500), api.Person{FirstName: str("Айгуль")}, str("@aigul_trek")
	o.IsJoined = true

	s := screens.OutingCard(o, screens.FromFind, links, now)
	want := "🥾 <b>Пик Учитель</b>\n🏔 Ала-Арча · нужен опыт\n\n📅 сб, 24 окт., 10:00 → 19:00\n📍 Сбор: ТЦ «Вефа»\n" +
		"🚗 На машине организатора\n💳 500 сом с человека\n👥 ▰▰▰▰▰▰▰▱▱▱ 7 из 10 мест\n🧑 Организатор: Айгуль\n💬 Связь: @aigul_trek\n\n✅ Вы идёте"
	if s.Text != want {
		t.Errorf("card:\n%s\nwant:\n%s", s.Text, want)
	}
	if got := buttons(s); strings.Join(got, ",") != "🌐 Открыть в Ermekten→https://ermekten.kg/outings/o1,"+
		"🙅 Не смогу пойти→o:ask:f:o1,⬅️ К ближайшим выходам→o:find:0" {
		t.Errorf("joined buttons: %v", got)
	}

	o.IsJoined = false
	if !has(buttons(screens.OutingCard(o, screens.FromMine, links, now)), "✋ Пойду→o:join:u:o1") {
		t.Error("no join button for a free seat")
	}
	o.Taken = 10
	full := screens.OutingCard(o, screens.FromMine, links, now)
	if strings.Contains(strings.Join(buttons(full), ","), "o:join") || !strings.HasSuffix(full.Text, "Мест больше нет") {
		t.Errorf("full: %v\n%s", buttons(full), full.Text)
	}
	o.Taken, o.IsOrganizer = 3, true
	own := screens.OutingCard(o, screens.FromMine, links, now)
	if len(buttons(own)) != 2 || !strings.HasSuffix(own.Text, "⭐ Вы организатор") {
		t.Errorf("organizer: %v", buttons(own))
	}
	o.IsOrganizer, o.Status = false, "cancelled"
	if gone := screens.OutingCard(o, screens.FromPast, links, now); len(buttons(gone)) != 2 ||
		!strings.HasSuffix(gone.Text, "❌ Выход отменён организатором") {
		t.Errorf("cancelled: %v", buttons(gone))
	}

	confirm := screens.LeaveConfirm(o, screens.FromMine)
	if got := buttons(confirm); strings.Join(got, ",") != "Да, не пойду→o:leave:u:o1,Нет, иду→o:view:u:o1" {
		t.Errorf("confirm: %v", got)
	}
}

func TestFindOutingsPages(t *testing.T) {
	var list []api.Outing
	for i := range 7 {
		list = append(list, outing("id"+string(rune('a'+i)), "Выход", now.Add(time.Duration(i+1)*24*time.Hour), "open"))
	}
	s := screens.FindOutings(list, 1, links)
	got := buttons(s)
	if !has(got, "6 · Выход→o:view:f:idf") || !has(got, "2 / 2→noop") || !has(got, "◀️→o:find:0") ||
		!has(got, "▶️→o:find:0") || has(got, "1 · Выход→o:view:f:ida") {
		t.Errorf("page 2: %v", got)
	}
	if !strings.Contains(s.Text, "👥 свободно 3 из 10") {
		t.Errorf("text:\n%s", s.Text)
	}
	empty := screens.FindOutings(nil, 0, links)
	if !has(buttons(empty), "➕ Собрать выход→https://ermekten.kg/outings/new") {
		t.Errorf("empty: %v", buttons(empty))
	}
}

func TestToursCarousel(t *testing.T) {
	tours := []api.Tour{
		{ID: "t1", Slug: "a", Title: "A"},
		{ID: "t2", Slug: "son-kol", Locale: "en", Title: "Сон-Кёль: ночь в юрте", Location: str("Нарын"),
			DurationDays: num(3), StartDate: "2027-06-12", EndDate: "2027-06-14", Price: num(18000),
			RatingAverage: 4.8, ReviewCount: 23, CoverImageURL: str("/uploads/tours/son-kol.jpg")},
	}
	s := screens.Tours(screens.ToursView{Tab: screens.TabFavorites, Index: 1, Tours: tours, Favorites: map[string]bool{"t2": true}}, links)
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
		"◀️→t:f:0", "2 / 2→noop", "🌐 Подробнее→https://ermekten.kg/en/tours/son-kol", "💔 Из избранного→t:drop:f:1:t2",
		"• ❤️ Избранное •→t:f:0", "🆕 Ближайшие→t:n:0", "🔔 Сообщать о новых турах→t:alert:f:1",
	} {
		if !has(got, w) {
			t.Errorf("buttons lack %q: %v", w, got)
		}
	}

	first := screens.Tours(screens.ToursView{Tab: screens.TabUpcoming, Tours: tours, AlertOn: true}, links)
	if g := buttons(first); !has(g, "❤️ В избранное→t:save:n:0:t1") || !has(g, "🔕 Не сообщать о новых турах→t:alert:n:0") {
		t.Errorf("upcoming: %v", g)
	}
	if !first.Preview.IsDisabled {
		t.Errorf("no cover, yet a preview: %+v", first.Preview)
	}
	empty := screens.Tours(screens.ToursView{Tab: screens.TabFavorites}, links)
	if !strings.Contains(empty.Text, "В избранном пока пусто") {
		t.Errorf("empty:\n%s", empty.Text)
	}
}

func TestBlog(t *testing.T) {
	published := time.Date(2026, 10, 18, 6, 0, 0, 0, time.UTC)
	posts := []api.BlogPost{{ID: "p1", Slug: "checklist", Title: "Что взять в поход", Excerpt: str("Рюкзак & палки"),
		PublishedAt: &published, CoverImageURL: str("https://cdn.example.com/a.jpg")}}
	s := screens.Blog(posts, 0, false, links)
	if s.Text != "📰 <b>Блог Ermekten</b> · 1 из 1\n\n<b>Что взять в поход</b>\n<i>18 окт. 2026</i>\n\nРюкзак &amp; палки" {
		t.Errorf("blog:\n%s", s.Text)
	}
	if s.Preview.URL != "https://cdn.example.com/a.jpg" {
		t.Errorf("preview: %+v", s.Preview)
	}
	if got := buttons(s); strings.Join(got, ",") != "📖 Читать статью→https://ermekten.kg/blog/checklist,🔔 Присылать новые статьи→b:sub:0" {
		t.Errorf("buttons: %v", got)
	}
	if !has(buttons(screens.Blog(posts, 0, true, links)), "🔕 Не присылать новые статьи→b:unsub:0") {
		t.Error("no unsubscribe button")
	}

	caption, markup := screens.BlogPost(posts[0], links)
	raw, _ := json.Marshal(markup)
	if caption != "📰 <b>Новая статья в блоге</b>\n\n<b>Что взять в поход</b>\nРюкзак &amp; палки" ||
		!strings.Contains(string(raw), `"callback_data":"n:blog_off"`) {
		t.Errorf("post: %q %s", caption, raw)
	}
}

func TestNotifications(t *testing.T) {
	s := screens.Notifications(screens.NotifySettings{ToursAll: true})
	if got := buttons(s); strings.Join(got, ",") != "🧭 Новые туры: ✅ вкл→n:tours,📰 Статьи блога: ⬜ выкл→n:blog" {
		t.Errorf("buttons: %v", got)
	}
	partial := screens.Notifications(screens.NotifySettings{ToursDestinations: 2, Blog: true})
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

func TestOutingCardDetails(t *testing.T) {
	o := outing("o1", "Траверс <Ала-Арчи>", now.Add(48*time.Hour), "open")
	returns := o.StartsAt.Add(30 * time.Hour)
	o.ReturnsAt, o.CostPerPerson, o.Transport, o.Level = &returns, num(0), "arranged", "advanced"
	o.Organizer = api.Person{Username: str("guide")}
	o.Description = str("Берите кошки & ледоруб.")
	s := screens.OutingCard(o, screens.FromMine, links, now)
	for _, want := range []string{
		"🥾 <b>Траверс &lt;Ала-Арчи&gt;</b>\nдля опытных\n\n",
		"📅 чт, 22 окт., 10:00 → пт, 23 окт., 16:00",
		"🚗 Заказной транспорт\n💳 Бесплатно",
		"🧑 Организатор: @guide",
		"<i>Берите кошки &amp; ледоруб.</i>",
		"Есть свободные места — присоединяйтесь!",
	} {
		if !strings.Contains(s.Text, want) {
			t.Errorf("card lacks %q:\n%s", want, s.Text)
		}
	}

	// Past: nothing to do but look; no organizer name, no line.
	past := outing("o2", "Ала-Кёль", now.Add(-48*time.Hour), "open")
	past.IsJoined = true
	ps := screens.OutingCard(past, screens.FromPast, links, now)
	if !strings.HasSuffix(ps.Text, "🏁 Выход уже прошёл") || strings.Contains(ps.Text, "Организатор") {
		t.Errorf("past:\n%s", ps.Text)
	}
	if got := buttons(ps); strings.Join(got, ",") != "🌐 Открыть в Ermekten→https://ermekten.kg/outings/o2,⬅️ К моим выходам→o:list:p" {
		t.Errorf("past buttons: %v", got)
	}
	// Started less than 3 hours ago: still upcoming, as on the site.
	if late := screens.OutingCard(outing("o3", "X", now.Add(-2*time.Hour), "open"), screens.FromMine, links, now); !has(buttons(late), "✋ Пойду→o:join:u:o3") {
		t.Errorf("grace period: %v", buttons(late))
	}
}

func TestMyOutingsMoreThanShown(t *testing.T) {
	var list []api.Outing
	for i := range 10 {
		o := outing("o"+string(rune('a'+i)), "Выход", now.Add(time.Duration(i+1)*time.Hour), "open")
		o.IsJoined = true
		list = append(list, o)
	}
	s := screens.MyOutings(list, nil, screens.FromMine, now)
	if !strings.HasSuffix(s.Text, "…и ещё 2 — все на сайте.") || len(s.Markup.InlineKeyboard) != 8+2 {
		t.Errorf("10 outings: %d rows\n%s", len(s.Markup.InlineKeyboard), s.Text)
	}
	if empty := screens.MyOutings(nil, nil, screens.FromPast, now); !strings.HasSuffix(empty.Text, "Прошедших выходов пока нет.") {
		t.Errorf("no past:\n%s", empty.Text)
	}
}

func TestToursEdges(t *testing.T) {
	one := []api.Tour{{ID: "t1", Slug: "a", Title: "A", StartDate: "bad"}}
	// Past the end: the last tour; one tour: no pager; no dates line for a bad date.
	s := screens.Tours(screens.ToursView{Tab: screens.TabUpcoming, Index: 9, Tours: one}, links)
	if s.Text != "🧭 <b>Туры</b> · Ближайшие 1 из 1\n\n<b>A</b>" || has(buttons(s), "1 / 1→noop") {
		t.Errorf("one tour: %q %v", s.Text, buttons(s))
	}
	up := screens.Tours(screens.ToursView{Tab: screens.TabUpcoming}, links)
	if !strings.Contains(up.Text, "Ближайших туров пока нет") || len(up.Markup.InlineKeyboard) != 2 {
		t.Errorf("no upcoming: %v\n%s", buttons(up), up.Text)
	}
}

func TestBlogEdges(t *testing.T) {
	s := screens.Blog(nil, 3, true, links)
	if !strings.Contains(s.Text, "Статей пока нет") || strings.Join(buttons(s), ",") != "🔕 Не присылать новые статьи→b:unsub:0" {
		t.Errorf("empty blog: %v\n%s", buttons(s), s.Text)
	}
	en := api.BlogPost{ID: "p", Slug: "x", Locale: "en", Title: "T"}
	if caption, _ := screens.BlogPost(en, links); caption != "📰 <b>Новая статья в блоге</b>\n\n<b>T</b>" {
		t.Errorf("no excerpt: %q", caption)
	}
	if !has(buttons(screens.Blog([]api.BlogPost{en}, 0, false, links)), "📖 Читать статью→https://ermekten.kg/en/blog/x") {
		t.Error("locale prefix in the post link")
	}
}

// With a Mini App every site button opens the page inside Telegram.
func TestMiniAppButtons(t *testing.T) {
	app := screens.Links{Site: "https://ermekten.kg", MiniApp: "https://app.ermekten.kg"}
	o := outing("o1", "Пик", now.Add(48*time.Hour), "open")
	tour := api.Tour{ID: "t1", Slug: "son-kol", Locale: "ky", Title: "Сон-Кёль"}
	post := api.BlogPost{ID: "p1", Slug: "checklist", Title: "Чек-лист"}
	_, pushMarkup := screens.BlogPost(post, app)

	for _, tc := range []struct {
		s    screens.Screen
		want string
	}{
		{screens.FindOutings(nil, 0, app), "➕ Собрать выход→webapp:https://app.ermekten.kg/outings/new"},
		{screens.OutingCard(o, screens.FromMine, app, now), "🌐 Открыть в Ermekten→webapp:https://app.ermekten.kg/outings/o1"},
		{screens.Tours(screens.ToursView{Tab: screens.TabUpcoming, Tours: []api.Tour{tour}}, app), "🌐 Подробнее→webapp:https://app.ermekten.kg/ky/tours/son-kol"},
		{screens.Blog([]api.BlogPost{post}, 0, false, app), "📖 Читать статью→webapp:https://app.ermekten.kg/blog/checklist"},
		{screens.Screen{Markup: pushMarkup}, "📖 Читать→webapp:https://app.ermekten.kg/blog/checklist"},
	} {
		got := buttons(tc.s)
		if !has(got, tc.want) {
			t.Errorf("no %q in %v", tc.want, got)
		}
		for _, b := range got {
			if strings.Contains(b, "→https://") {
				t.Errorf("a button still opens the browser: %q", b)
			}
		}
	}
}
