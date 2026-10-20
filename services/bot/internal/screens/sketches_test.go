package screens_test

// Each artboard of the design sketches ("Ermekten — Telegram-бот" canvas),
// rendered from the data drawn on it: the text and the button layout must
// match the sketch exactly. Changing a screen means changing its sketch.

import (
	"strings"
	"testing"
	"time"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/screens"
)

// at is a time in Bishkek (UTC+6) in 2026.
func at(month time.Month, day, hour, minute int) time.Time {
	return time.Date(2026, month, day, hour-6, minute, 0, 0, time.UTC)
}

// layout is the keyboard row by row, "text→data|url" joined with " | ".
func layout(s screens.Screen) []string {
	var out []string
	if s.Markup == nil {
		return out
	}
	for _, row := range s.Markup.InlineKeyboard {
		var cells []string
		for _, b := range row {
			cells = append(cells, b.Text+"→"+b.CallbackData+b.URL)
		}
		out = append(out, strings.Join(cells, " | "))
	}
	return out
}

func checkScreen(t *testing.T, s screens.Screen, text string, rows ...string) {
	t.Helper()
	if s.Text != text {
		t.Errorf("text:\n%s\n--- want (sketch):\n%s", s.Text, text)
	}
	if got := layout(s); strings.Join(got, "\n") != strings.Join(rows, "\n") {
		t.Errorf("buttons:\n%s\n--- want (sketch):\n%s", strings.Join(got, "\n"), strings.Join(rows, "\n"))
	}
}

// The outings drawn on the sketches.
var (
	uchitel = api.Outing{ID: "o1", Title: "Пик Учитель", StartsAt: at(10, 24, 6, 0), Status: "open",
		Seats: 10, Taken: 7, IsJoined: true, MeetingPoint: "ТЦ «Вефа», парковка", Trail: &api.TrailRef{Name: "Ала-Арча"}}
	barskoon = api.Outing{ID: "o2", Title: "Водопад Барскоон", StartsAt: at(11, 1, 7, 30), Status: "open",
		Seats: 12, Taken: 4, IsOrganizer: true, Trail: &api.TrailRef{Name: "Барскоон"}}
	alaKol = api.Outing{ID: "o3", Title: "Озеро Ала-Кёль", StartsAt: at(11, 7, 5, 30), Status: "open",
		Seats: 10, Taken: 9, IsJoined: true, Trail: &api.TrailRef{Name: "Каракол"}}
)

func TestSketchMain(t *testing.T) {
	next := uchitel
	want := "🏔 <b>Здравствуйте, Айбек!</b>\n\n" +
		"Я бот Ermekten — помогу не пропустить выходы в горы и туры по Кыргызстану.\n\n" +
		"🥾 <b>Мои выходы</b> — куда вы идёте и когда сбор\n" +
		"🧭 <b>Туры</b> — избранное и ближайшие даты\n" +
		"📰 <b>Блог</b> — свежие статьи и маршруты\n" +
		"🔔 <b>Уведомления</b> — что присылать\n\n" +
		"Ближайший выход: <b>Пик Учитель</b>, сб, 24 окт., 06:00\n\n" +
		"Выберите раздел в меню ниже."
	if got := screens.Welcome("Айбек", &next); got != want {
		t.Errorf("greeting:\n%s\n--- want (sketch):\n%s", got, want)
	}
	if got := screens.Welcome("Айбек", nil); strings.Contains(got, "Ближайший выход") {
		t.Errorf("greeting without outings mentions one:\n%s", got)
	}
}

func TestSketchMyOutings(t *testing.T) {
	sum := &api.OutingsSummary{Outings: 12, DistanceKm: 84.2, ElevationGain: 5200}
	s := screens.MyOutings([]api.Outing{alaKol, barskoon, uchitel}, sum, screens.FromMine, at(10, 20, 10, 5))
	checkScreen(t, s,
		"🥾 <b>Мои выходы</b>\n"+
			"За всё время: 12 выходов · 84 км · ↗ 5 200 м\n\n"+
			"<b>Предстоящие</b>\n\n"+
			"<b>1. Пик Учитель</b>\nсб, 24 окт., 06:00 · 👥 7/10 · вы идёте\n\n"+
			"<b>2. Водопад Барскоон</b>\nвс, 1 нояб., 07:30 · 👥 4/12 · ⭐ вы организатор\n\n"+
			"<b>3. Озеро Ала-Кёль</b>\nсб, 7 нояб., 05:30 · 👥 9/10 · вы идёте",
		"1 · Пик Учитель→o:view:u:o1",
		"2 · Водопад Барскоон→o:view:u:o2",
		"3 · Озеро Ала-Кёль→o:view:u:o3",
		"• Предстоящие •→o:list:u | Прошедшие→o:list:p",
		"🔎 Найти выход→o:find:0",
	)
}

func TestSketchFindOutings(t *testing.T) {
	alaKolFull := alaKol
	alaKolFull.Taken, alaKolFull.IsJoined = 10, false
	komsomolets := api.Outing{ID: "o4", Title: "Пик Комсомолец", StartsAt: at(11, 7, 7, 0), Status: "open",
		Seats: 8, Taken: 2, Trail: &api.TrailRef{Name: "Ала-Арча"}}
	more := api.Outing{ID: "o5", Title: "Кёль-Тор", StartsAt: at(11, 8, 8, 0), Status: "open", Seats: 15, Taken: 9}
	s := screens.FindOutings([]api.Outing{uchitel, barskoon, alaKolFull, komsomolets, more}, 0, links)
	checkScreen(t, s,
		"🔎 <b>Ближайшие выходы</b>\n\n"+
			"<b>1. Пик Учитель</b>\nсб, 24 окт., 06:00 · 👥 свободно 3 из 10 · 🏔 Ала-Арча · вы идёте\n\n"+
			"<b>2. Водопад Барскоон</b>\nвс, 1 нояб., 07:30 · 👥 свободно 8 из 12 · 🏔 Барскоон · ⭐ вы организатор\n\n"+
			"<b>3. Озеро Ала-Кёль</b>\nсб, 7 нояб., 05:30 · мест нет · 🏔 Каракол\n\n"+
			"<b>4. Пик Комсомолец</b>\nсб, 7 нояб., 07:00 · 👥 свободно 6 из 8 · 🏔 Ала-Арча",
		"1 · Пик Учитель→o:view:f:o1",
		"2 · Водопад Барскоон→o:view:f:o2",
		"3 · Озеро Ала-Кёль→o:view:f:o3",
		"4 · Пик Комсомолец→o:view:f:o4",
		"◀️→o:find:1 | 1 / 2→noop | ▶️→o:find:1",
		"⬅️ Мои выходы→o:list:u",
	)
}

func TestSketchOutingCard(t *testing.T) {
	o := uchitel
	returns := at(10, 24, 19, 0)
	o.ReturnsAt, o.Level, o.Transport, o.CostPerPerson = &returns, "intermediate", "car", num(500)
	o.Organizer, o.Contact = api.Person{FirstName: str("Айгуль")}, str("@aigul_trek")

	checkScreen(t, screens.OutingCard(o, screens.FromMine, links, at(10, 20, 10, 6)),
		"🥾 <b>Пик Учитель</b>\n🏔 Ала-Арча · нужен опыт\n\n"+
			"📅 сб, 24 окт., 06:00 → 19:00\n"+
			"📍 Сбор: ТЦ «Вефа», парковка\n"+
			"🚗 На машине организатора\n"+
			"💳 500 сом с человека\n"+
			"👥 ▰▰▰▰▰▰▰▱▱▱ 7 из 10 мест\n"+
			"🧑 Организатор: Айгуль\n"+
			"💬 Связь: @aigul_trek\n\n"+
			"✅ Вы идёте",
		"🌐 Открыть на сайте→https://ermekten.kg/outings/o1",
		"🙅 Не смогу пойти→o:ask:u:o1",
		"⬅️ К моим выходам→o:list:u",
	)
	checkScreen(t, screens.LeaveConfirm(o, screens.FromMine),
		"Точно не пойдёте на «Пик Учитель»?\n\nМесто освободится, организатор получит уведомление.",
		"Да, не пойду→o:leave:u:o1 | Нет, иду→o:view:u:o1",
	)
}

func TestSketchTours(t *testing.T) {
	sonKol := api.Tour{ID: "t2", Slug: "son-kol", Locale: "ru", Title: "Сон-Кёль: ночь в юрте", Location: str("Нарын"),
		DurationDays: num(3), StartDate: "2027-06-12", EndDate: "2027-06-14", Price: num(18000),
		RatingAverage: 4.8, ReviewCount: 23, CoverImageURL: str("/uploads/tours/son-kol.jpg"),
		Excerpt: str("Перевал Калмак-Ашуу, табуны на джайлоо и звёзды над озером на высоте 3 016 м.")}
	tours := []api.Tour{{ID: "t1"}, sonKol, {ID: "t3"}, {ID: "t4"}, {ID: "t5"}}
	favorites := map[string]bool{"t1": true, "t2": true, "t3": true, "t4": true, "t5": true}

	s := screens.Tours(screens.ToursView{Tab: screens.TabFavorites, Index: 1, Tours: tours, Favorites: favorites}, links)
	checkScreen(t, s,
		"🧭 <b>Туры</b> · Избранное 2 из 5\n\n"+
			"<b>Сон-Кёль: ночь в юрте</b>\n"+
			"📍 Нарын · 3 дня\n"+
			"📅 12 — 14 июня\n"+
			"💳 18 000 сом с человека\n"+
			"⭐ 4.8 · 23 отзыва\n\n"+
			"<i>Перевал Калмак-Ашуу, табуны на джайлоо и звёзды над озером на высоте 3 016 м.</i>",
		"◀️→t:f:0 | 2 / 5→noop | ▶️→t:f:2",
		"🌐 Подробнее→https://ermekten.kg/tours/son-kol | 💔 Из избранного→t:drop:f:1:t2",
		"• ❤️ Избранное •→t:f:0 | 🆕 Ближайшие→t:n:0",
		"🔔 Сообщать о новых турах→t:alert:f:1",
	)
	// The cover sits above the text, as drawn.
	if p := s.Preview; p == nil || !p.ShowAboveText || !p.PreferLargeMedia ||
		p.URL != "https://api.ermekten.kg/images/1280/uploads/tours/son-kol.jpg" {
		t.Errorf("cover: %+v", p)
	}
}

func TestSketchBlog(t *testing.T) {
	published := at(10, 18, 9, 0)
	post := api.BlogPost{ID: "p1", Slug: "first-hike-checklist", Title: "Что взять в первый поход: чек-лист на выходные",
		PublishedAt: &published, CoverImageURL: str("/uploads/blog/checklist.jpg"),
		Excerpt: str("Рюкзак на 30 литров, треккинговые палки и термос — разбираем, без чего не стоит выходить в горы осенью и что можно оставить дома.")}
	posts := []api.BlogPost{post, {ID: "p2"}, {ID: "p3"}, {ID: "p4"}, {ID: "p5"}, {ID: "p6"}, {ID: "p7"}, {ID: "p8"}}

	s := screens.Blog(posts, 0, false, links)
	checkScreen(t, s,
		"📰 <b>Блог Ermekten</b> · 1 из 8\n\n"+
			"<b>Что взять в первый поход: чек-лист на выходные</b>\n"+
			"<i>18 окт. 2026</i>\n\n"+
			"Рюкзак на 30 литров, треккинговые палки и термос — разбираем, без чего не стоит выходить в горы осенью и что можно оставить дома.",
		"◀️→b:7 | 1 / 8→noop | ▶️→b:1",
		"📖 Читать статью→https://ermekten.kg/blog/first-hike-checklist",
		"🔔 Присылать новые статьи→b:sub:0",
	)
	if p := s.Preview; p == nil || !p.ShowAboveText || p.URL != "https://api.ermekten.kg/images/1280/uploads/blog/checklist.jpg" {
		t.Errorf("cover: %+v", p)
	}
}

func TestSketchNotifications(t *testing.T) {
	checkScreen(t, screens.Notifications(screens.NotifySettings{ToursAll: true}),
		"🔔 <b>Уведомления</b>\n\nВыберите, о чём вам писать.\n\n"+
			"🥾 <b>Мои выходы</b> — всегда\nКто записался, изменения, отмена, напоминание перед сбором.\n\n"+
			"🧭 <b>Новые туры</b> — ✅ включено\nКогда на сайте появляется новый тур.\n\n"+
			"📰 <b>Статьи блога</b> — выключено\nНовая статья — одним сообщением с обложкой.",
		"🧭 Новые туры: ✅ вкл→n:tours",
		"📰 Статьи блога: ⬜ выкл→n:blog",
	)
}

func TestSketchBlogNotification(t *testing.T) {
	post := api.BlogPost{ID: "p9", Slug: "autumn-ala-archa", Title: "Осенняя Ала-Арча: 5 маршрутов на один день",
		Excerpt: str("От лёгкой прогулки к водопаду до подъёма на Рацек — с перепадами высот и временем в пути.")}
	caption, markup := screens.BlogPost(post, links)
	checkScreen(t, screens.Screen{Text: caption, Markup: markup},
		"📰 <b>Новая статья в блоге</b>\n\n"+
			"<b>Осенняя Ала-Арча: 5 маршрутов на один день</b>\n"+
			"От лёгкой прогулки к водопаду до подъёма на Рацек — с перепадами высот и временем в пути.",
		"📖 Читать→https://ermekten.kg/blog/autumn-ala-archa | 🔕 Не присылать→n:blog_off",
	)
}
