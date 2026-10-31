package screens_test

// Each artboard of the design sketches ("Ermekten — Telegram-бот" canvas),
// rendered from the data drawn on it: the text and the button layout must
// match the sketch exactly. Changing a screen means changing its sketch.

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/i18n"
	"github.com/elvn5/sai/services/bot/internal/screens"
)

// at is a time in Bishkek (UTC+6) in 2026.
func at(month time.Month, day, hour, minute int) time.Time {
	return time.Date(2026, month, day, hour-6, minute, 0, 0, time.UTC)
}

// layout is the keyboard row by row, "text→data|url" joined with " | "; a
// share dialog shows as "share:<the link shared>".
func layout(s screens.Screen) []string {
	var out []string
	if s.Markup == nil {
		return out
	}
	for _, row := range s.Markup.InlineKeyboard {
		var cells []string
		for _, b := range row {
			target := b.CallbackData + b.URL
			if b.WebApp != nil {
				target = "webapp:" + b.WebApp.URL
			}
			if q, ok := strings.CutPrefix(b.URL, "https://t.me/share/url?"); ok {
				v, _ := url.ParseQuery(q)
				target = "share:" + v.Get("url")
			}
			cells = append(cells, b.Text+"→"+target)
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

// The tours drawn on the sketches: two the user signed up for, one they
// run.
var (
	uchitel = api.Tour{ID: "o1", Title: "Пик Учитель", StartDate: "2026-10-24", EndDate: "2026-10-24", MeetingTime: str("06:00"),
		MeetingPoint: str("ТЦ «Вефа», парковка"), Capacity: num(10), Taken: 7, IsJoined: true,
		MyForm: &filledForm, MyWaiverAt: str("2026-10-10T00:00:00Z")}
	barskoon = api.Tour{ID: "o2", Title: "Водопад Барскоон", StartDate: "2026-11-01", EndDate: "2026-11-01", MeetingTime: str("07:30"),
		Capacity: num(12), Taken: 4, IsOrganizer: true}
	alaKol = api.Tour{ID: "o3", Title: "Озеро Ала-Кёль", StartDate: "2026-11-07", EndDate: "2026-11-08", MeetingTime: str("05:30"),
		Taken: 9, IsJoined: true, MyForm: &filledForm, MyWaiverAt: str("2026-10-10T00:00:00Z")}
)

// The participant's form as the API returns it (its fields don't matter to
// the bot).
var filledForm = json.RawMessage(`{}`)

func TestSketchMain(t *testing.T) {
	next := uchitel
	want := "🏔 <b>Здравствуйте, Айбек!</b>\n\n" +
		"Я бот Ermekten — помогу не пропустить туры по Кыргызстану и записаться на них прямо в чате.\n\n" +
		"🎒 <b>Мои туры</b> — куда вы едете и когда сбор\n" +
		"🧭 <b>Туры</b> — избранное и ближайшие даты\n" +
		"📰 <b>Блог</b> — свежие статьи и маршруты\n" +
		"🔔 <b>Уведомления</b> — что присылать\n\n" +
		"Ближайший тур: <b>Пик Учитель</b>, сб, 24 окт., 06:00\n\n" +
		"Выберите раздел в меню ниже."
	if got := screens.Welcome(i18n.RU, "Айбек", &next); got != want {
		t.Errorf("greeting:\n%s\n--- want (sketch):\n%s", got, want)
	}
	if got := screens.Welcome(i18n.RU, "Айбек", nil); strings.Contains(got, "Ближайший тур") {
		t.Errorf("greeting without tours mentions one:\n%s", got)
	}
}

func TestSketchMyTours(t *testing.T) {
	done := api.Tour{ID: "o9", Title: "Сон-Кёль", StartDate: "2026-09-12", EndDate: "2026-09-14", IsJoined: true}
	s := screens.MyTours(i18n.RU, []api.Tour{alaKol, done, uchitel}, screens.FromMine, at(10, 20, 10, 5))
	checkScreen(t, s,
		"🎒 <b>Мои туры</b>\n"+
			"За всё время: 1 тур\n\n"+
			"<b>Предстоящие</b>\n\n"+
			"<b>1. Пик Учитель</b>\nсб, 24 окт., 06:00 · 👥 7 из 10 мест\n\n"+
			"<b>2. Озеро Ала-Кёль</b>\nсб, 7 нояб., 05:30 · 👥 записались: 9",
		"1 · Пик Учитель→o:view:u:o1",
		"2 · Озеро Ала-Кёль→o:view:u:o3",
		"• Предстоящие •→o:list:u | Прошедшие→o:list:p",
		"🧭 Все туры→t:n:0",
		"🧑‍✈️ Кабинет организатора→g:home",
	)
}

func TestSketchTourCard(t *testing.T) {
	o := uchitel
	o.Price, o.Organizer, o.Telegram = num(500), &api.Person{FirstName: str("Айгуль")}, str("aigul_trek")
	o.Roster = nil

	checkScreen(t, screens.TourCard(i18n.RU, o, screens.FromMine, links, at(10, 20, 10, 6)),
		"🧭 <b>Пик Учитель</b>\n📅 24 октября\n\n"+
			"🕕 Сбор: сб, 24 окт., 06:00\n"+
			"📍 ТЦ «Вефа», парковка\n"+
			"💳 500 сом с человека\n"+
			"👥 ▰▰▰▰▰▰▰▱▱▱ 7 из 10 мест\n"+
			"🧑 Организатор: Айгуль\n\n"+
			"📞 <b>Связь с организатором</b>\nTelegram: @aigul_trek\n\n"+
			"✅ Вы записаны",
		"🙅 Отменить запись→o:ask:u:o1",
		"📤 Позвать друзей→share:https://t.me/ermekten_bot?start=tour_o1",
		"⬅️ К моим турам→o:list:u",
	)
	checkScreen(t, screens.LeaveConfirm(i18n.RU, o, screens.FromMine),
		"Точно отменить запись на «Пик Учитель»?\n\nМесто освободится, организатор получит уведомление.",
		"Да, отменить→o:leave:u:o1 | Нет, еду→o:view:u:o1",
	)
}

func TestSketchOrganizerCard(t *testing.T) {
	o := barskoon
	o.MeetingPoint, o.Price = str("Южные ворота"), num(800)
	o.Roster = []api.RosterEntry{{Person: api.Person{FirstName: str("Айбек")}}, {Person: api.Person{FirstName: str("Нурлан")}},
		{Person: api.Person{Username: str("aidai_kg")}}, {Person: api.Person{FirstName: str("Мээрим")}}}
	checkScreen(t, screens.TourCard(i18n.RU, o, screens.FromMine, links, at(10, 20, 10, 6)),
		"🧭 <b>Водопад Барскоон</b>\n📅 1 ноября\n\n"+
			"🕕 Сбор: вс, 1 нояб., 07:30\n"+
			"📍 Южные ворота\n"+
			"💳 800 сом с человека\n"+
			"👥 ▰▰▰▱▱▱▱▱▱▱ 4 из 12 мест\n\n"+
			"👥 Едут: Айбек, Нурлан, @aidai_kg, Мээрим\n\n"+
			"⭐ Вы организатор",
		"📋 Участники и отметка→g:r:o2",
		"📤 Позвать друзей→share:https://t.me/ermekten_bot?start=tour_o2",
		"⬅️ К моим турам→o:list:u",
	)
}

func TestSketchTourMore(t *testing.T) {
	tour := api.Tour{ID: "t2", Slug: "son-kol", Title: "Сон-Кёль: ночь в юрте", Location: str("Нарын"), DurationDays: num(3),
		StartDate: "2027-06-12", EndDate: "2027-06-14", Price: num(18000), Capacity: num(12), Taken: 5, MeetingTime: str("08:00"),
		Content: `<p>Три дня на высокогорном озере: перевал <strong>Калмак-Ашуу</strong>, табуны на джайлоо и ночь в юрте.</p>` +
			`<h3>Что включено</h3><ul><li>Транспорт из Бишкека</li><li>Юрта и питание</li></ul>`,
		Program: []api.ProgramDay{
			{Title: "Бишкек — Кочкор", Text: "Выезд в 8:00, обед в Кочкоре."},
			{Title: "Сон-Кёль", Text: "Конная прогулка, закат."},
		},
		Telegram: str("@sonkol_tours"), Phone: str("+996 555 123 456")}
	checkScreen(t, screens.TourMore(i18n.RU, tour, screens.TabFavorites, 1, 0, true, links, at(10, 20, 10, 0)),
		"🧭 <b>Сон-Кёль: ночь в юрте</b>\n"+
			"📍 Нарын · 3 дня\n📅 12 — 14 июня\n💳 18 000 сом с человека\n🕕 Сбор: сб, 12 июн., 08:00 · 👥 5 из 12 мест\n\n"+
			"👥 В группе до 12 человек\n\n"+
			"Три дня на высокогорном озере: перевал <b>Калмак-Ашуу</b>, табуны на джайлоо и ночь в юрте.\n\n"+
			"<b>Что включено</b>\n\n"+
			"• Транспорт из Бишкека\n• Юрта и питание\n\n"+
			"🗓 <b>Программа</b>\n\n"+
			"<b>День 1. Бишкек — Кочкор</b>\nВыезд в 8:00, обед в Кочкоре.\n\n"+
			"<b>День 2. Сон-Кёль</b>\nКонная прогулка, закат.\n\n"+
			"📞 <b>Связь с организатором</b>\nTelegram: @sonkol_tours\nТелефон: +996 555 123 456",
		"✋ Записаться→o:join:t:t2 | 📤 Поделиться→share:https://t.me/ermekten_bot?start=tour_t2",
		"💔 Из избранного→t:drop:f:1:t2:more | ⬅️ К турам→t:f:1",
	)
	tour.IsJoined = true
	if got := layout(screens.TourMore(i18n.RU, tour, screens.TabFavorites, 1, 0, true, links, at(10, 20, 10, 0))); got[0] != "🎒 Моя запись→o:view:t:t2 | 📤 Поделиться→share:https://t.me/ermekten_bot?start=tour_t2" {
		t.Errorf("joined: %v", got)
	}
}

func TestSketchTours(t *testing.T) {
	sonKol := api.Tour{ID: "t2", Slug: "son-kol", Locale: "ru", Title: "Сон-Кёль: ночь в юрте", Location: str("Нарын"),
		DurationDays: num(3), StartDate: "2027-06-12", EndDate: "2027-06-14", Price: num(18000),
		RatingAverage: 4.8, ReviewCount: 23, CoverImageURL: str("/uploads/tours/son-kol.jpg"),
		Excerpt: str("Перевал Калмак-Ашуу, табуны на джайлоо и звёзды над озером на высоте 3 016 м.")}
	tours := []api.Tour{{ID: "t1"}, sonKol, {ID: "t3"}, {ID: "t4"}, {ID: "t5"}}
	favorites := map[string]bool{"t1": true, "t2": true, "t3": true, "t4": true, "t5": true}

	s := screens.Tours(i18n.RU, screens.ToursView{Tab: screens.TabFavorites, Index: 1, Tours: tours, Favorites: favorites}, links)
	checkScreen(t, s,
		"🧭 <b>Туры</b> · Избранное 2 из 5\n\n"+
			"<b>Сон-Кёль: ночь в юрте</b>\n"+
			"📍 Нарын · 3 дня\n"+
			"📅 12 — 14 июня\n"+
			"💳 18 000 сом с человека\n"+
			"⭐ 4.8 · 23 отзыва\n\n"+
			"<i>Перевал Калмак-Ашуу, табуны на джайлоо и звёзды над озером на высоте 3 016 м.</i>",
		"◀️→t:f:0 | 2 / 5→noop | ▶️→t:f:2",
		"✋ Записаться→o:join:t:t2 | 📤 Поделиться→share:https://t.me/ermekten_bot?start=tour_t2",
		"📖 Подробнее→t:more:f:1:0 | 💔 Из избранного→t:drop:f:1:t2",
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

	s := screens.Blog(i18n.RU, posts, 0, false, links)
	checkScreen(t, s,
		"📰 <b>Блог Ermekten</b> · 1 из 8\n\n"+
			"<b>Что взять в первый поход: чек-лист на выходные</b>\n"+
			"<i>18 окт. 2026</i>\n\n"+
			"Рюкзак на 30 литров, треккинговые палки и термос — разбираем, без чего не стоит выходить в горы осенью и что можно оставить дома.",
		"◀️→b:7 | 1 / 8→noop | ▶️→b:1",
		"📖 Читать статью→b:read:p1:0",
		"🔔 Присылать новые статьи→b:sub:0",
	)
	if p := s.Preview; p == nil || !p.ShowAboveText || p.URL != "https://api.ermekten.kg/images/1280/uploads/blog/checklist.jpg" {
		t.Errorf("cover: %+v", p)
	}
}

func TestSketchNotifications(t *testing.T) {
	checkScreen(t, screens.Notifications(i18n.RU, screens.NotifySettings{ToursAll: true}),
		"🔔 <b>Уведомления</b>\n\nВыберите, о чём вам писать.\n\n"+
			"🎒 <b>Мои туры</b> — всегда\nКто записался, изменения, отмена, напоминание перед сбором, просьба оценить тур.\n\n"+
			"🧭 <b>Новые туры</b> — ✅ включено\nКогда на сайте появляется новый тур.\n\n"+
			"📰 <b>Статьи блога</b> — выключено\nНовая статья — одним сообщением с обложкой.",
		"🧭 Новые туры: ✅ вкл→n:tours",
		"📰 Статьи блога: ⬜ выкл→n:blog",
	)
}

func TestSketchBlogNotification(t *testing.T) {
	post := api.BlogPost{ID: "p9", Slug: "autumn-ala-archa", Title: "Осенняя Ала-Арча: 5 маршрутов на один день",
		Excerpt: str("От лёгкой прогулки к водопаду до подъёма на Рацек — с перепадами высот и временем в пути.")}
	caption, markup := screens.BlogPost(i18n.RU, post)
	checkScreen(t, screens.Screen{Text: caption, Markup: markup},
		"📰 <b>Новая статья в блоге</b>\n\n"+
			"<b>Осенняя Ала-Арча: 5 маршрутов на один день</b>\n"+
			"От лёгкой прогулки к водопаду до подъёма на Рацек — с перепадами высот и временем в пути.",
		"📖 Читать→b:read:p9:0 | 🔕 Не присылать→n:blog_off",
	)
}

func TestSketchArticle(t *testing.T) {
	published := at(10, 18, 9, 0)
	long := strings.Repeat("Литров на 30, с поясным ремнём и чехлом от дождя. ", 70)
	post := api.BlogPost{ID: "p1", Title: "Что взять в первый поход", PublishedAt: &published,
		Content: `<h2>Рюкзак</h2><p>` + long + `</p><h2>Одежда</h2><ul><li>Мембранная куртка</li><li>Флиска</li></ul>`}
	first := screens.Article(i18n.RU, post, 0, 0, links)
	if !strings.HasPrefix(first.Text, "📰 <b>Что взять в первый поход</b> · 1/2\n<i>18 окт. 2026</i>\n\n<b>Рюкзак</b>\n\nЛитров на 30,") {
		t.Errorf("first page:\n%s", first.Text)
	}
	if got := strings.Join(layout(first), "\n"); got != "Дальше ▶️→b:read:p1:1\n⬅️ К статьям→b:0" {
		t.Errorf("first page buttons:\n%s", got)
	}
	second := screens.Article(i18n.RU, post, 0, 1, links)
	if !strings.HasPrefix(second.Text, "📰 <b>Что взять в первый поход</b> · 2/2\n\nЛитров на 30,") ||
		!strings.HasSuffix(second.Text, "от дождя.\n\n<b>Одежда</b>\n\n• Мембранная куртка\n• Флиска") {
		t.Errorf("second page:\n%s", second.Text)
	}
	if got := strings.Join(layout(second), "\n"); got != "◀️ Назад→b:read:p1:0\n⬅️ К статьям→b:0" {
		t.Errorf("second page buttons:\n%s", got)
	}
}

// The organizer's room: the tours they run.
var (
	barskoonDay = func() api.Tour {
		o := barskoon
		o.StartDate, o.EndDate, o.MeetingPoint, o.Taken, o.IsPublished = "2026-10-25", "2026-10-25", str("Южные ворота"), 4, true
		joined := at(10, 18, 12, 0)
		arrived := at(10, 25, 7, 20)
		people := []api.Person{{ID: "u-aibek-1", FirstName: str("Айбек")}, {ID: "u-nurlan-2", FirstName: str("Нурлан"), Username: str("nurlan_kg")},
			{ID: "u-aidai-33", Username: str("aidai_kg")}, {ID: "u-meerim-4"}}
		for i, p := range people {
			e := api.RosterEntry{Person: p, JoinedAt: joined}
			if i < 2 {
				e.ArrivedAt = &arrived
			}
			o.Roster = append(o.Roster, e)
		}
		return o
	}()
	sonKolOwn = api.Tour{ID: "t1", Slug: "son-kol", Title: "Сон-Кёль на лошадях", StartDate: "2027-06-12", EndDate: "2027-06-14",
		DurationDays: num(3), Location: str("Нарын"), Price: num(18000), Capacity: num(12), Taken: 3, IsPublished: true,
		RatingAverage: 4.8, ReviewCount: 12,
		Program: []api.ProgramDay{{Title: "Бишкек — Кочкор", Text: "Выезд в 8:00.\nОбед в Кочкоре."}, {Title: "Сон-Кёль", Text: "Конная прогулка."}, {Title: "Обратно"}}}
)

func TestSketchOrganizerHome(t *testing.T) {
	draft := api.Tour{ID: "t2", Title: "Ала-Кёль", StartDate: "2027-07-03", EndDate: "2027-07-05"}
	checkScreen(t, screens.OrganizerHome(i18n.RU, []api.Tour{barskoonDay, sonKolOwn, draft}, at(10, 25, 7, 25)),
		"🧑‍✈️ <b>Кабинет организатора</b>\nУчастники, отметка на сборе и ваши туры.\n\n"+
			"🧭 <b>Ваши туры</b>\n\n"+
			"<b>1. Водопад Барскоон</b>\n25 октября · 👥 4 из 12 мест · ✅ на месте 2 из 4\n\n"+
			"<b>2. Сон-Кёль на лошадях</b>\n12 — 14 июня · 👥 3 из 12 мест\n\n"+
			"<b>3. Ала-Кёль</b>\n3 — 5 июля · 👥 записались: 0 · 📝 черновик",
		"1 · Водопад Барскоон→g:t:0",
		"2 · Сон-Кёль на лошадях→g:t:1",
		"3 · Ала-Кёль→g:t:2",
		"⬅️ Мои туры→o:list:u",
	)
	if s := screens.OrganizerHome(i18n.RU, nil, at(10, 25, 7, 25)); !strings.Contains(s.Text, "У вас пока нет туров") {
		t.Errorf("empty room:\n%s", s.Text)
	}
}

func TestSketchRoster(t *testing.T) {
	chats := map[string]int64{"u-aibek-1": 1001, "u-nurlan-2": 1002}
	checkScreen(t, screens.Roster(i18n.RU, barskoonDay, chats, links, at(10, 25, 7, 25)),
		"📋 <b>Участники · Водопад Барскоон</b>\nвс, 25 окт., 07:30 · 📍 Южные ворота\n"+
			"👥 4 из 12 мест · ✅ на месте 2 из 4\n\n"+
			"1. ✅ <a href=\"tg://user?id=1001\">Айбек</a> · с 18 окт.\n"+
			"2. ✅ <a href=\"tg://user?id=1002\">Нурлан</a> · @nurlan_kg · с 18 окт.\n"+
			"3. ⬜ @aidai_kg · с 18 окт.\n"+
			"4. ⬜ Участник 4 · с 18 окт.\n\n"+
			"Нажмите на участника, когда он придёт на сбор.",
		"✅ Айбек→g:a:o2:u-aibek- | ✅ Нурлан→g:a:o2:u-nurlan",
		"⬜ @aidai_kg→g:a:o2:u-aidai- | ⬜ Участник 4→g:a:o2:u-meerim",
		"✅ Все на месте→g:all:o2",
		"📣 Написать участникам→g:m:o2",
		"📤 Позвать друзей→share:https://t.me/ermekten_bot?start=tour_o2",
		"🧭 Карточка тура→o:view:u:o2 | ⬅️ Кабинет→g:home",
	)
	// A week ahead: no marks yet.
	early := screens.Roster(i18n.RU, barskoonDay, nil, links, at(10, 18, 13, 0))
	if !strings.Contains(early.Text, "1. • Айбек · с 18 окт.") || !strings.HasSuffix(early.Text, "Отмечать прибывших можно будет за сутки до сбора.") ||
		strings.Contains(strings.Join(layout(early), "\n"), "g:a:") {
		t.Errorf("before check-in:\n%s\n%v", early.Text, layout(early))
	}
}

func TestSketchBroadcast(t *testing.T) {
	o := barskoonDay
	o.Organizer = &api.Person{FirstName: str("Айгуль")}
	checkScreen(t, screens.BroadcastPrompt(i18n.RU, o),
		"📣 <b>Сообщение участникам</b>\n«Водопад Барскоон» · 4 человека\n\n"+
			"Напишите сообщение — бот перешлёт его всем, кто записался. Например: "+
			"<i>«Сбор переносится на 06:30, ждём у южного входа»</i>.",
		"Отмена→g:mx:o2",
	)
	checkScreen(t, screens.BroadcastPreview(i18n.RU, o, "Ждём <у ворот>"),
		"Отправить участникам «Водопад Барскоон» (4 человека)?\n\n<blockquote>Ждём &lt;у ворот&gt;</blockquote>\n\n"+
			"<i>Чтобы исправить, просто напишите сообщение заново.</i>",
		"📣 Отправить→g:ms:o2 | Отмена→g:mx:o2",
	)
	checkScreen(t, screens.BroadcastMessage(i18n.RU, o, "Ждём у ворот"),
		"📣 <b>Сообщение от организатора</b> · Айгуль\nТур «Водопад Барскоон», вс, 25 окт., 07:30\n\nЖдём у ворот",
		"Открыть тур→o:view:n:o2",
	)
	checkScreen(t, screens.BroadcastSent(i18n.RU, o, 3, 4),
		"✅ Сообщение отправлено: 3 из 4.\n\nНе дошло до 1 человека — у них не подключён Telegram или бот заблокирован. "+
			"Свяжитесь с ними сами.",
		"⬅️ К участникам→g:r:o2",
	)
}

func TestSketchOrganizerTour(t *testing.T) {
	bookings := []api.Order{
		{Kind: "tour", ItemID: "t1", ItemDate: str("2027-06-12"), Quantity: 2, Amount: 36000, ContactName: "Айбек", ContactPhone: "+996 555 123 456", Status: "paid", Comment: str("Вегетарианское меню")},
		{Kind: "tour", ItemID: "t1", ItemDate: str("2027-06-12"), Quantity: 1, Amount: 18000, ContactName: "Dana", ContactPhone: "+49 151 000", Status: "refunded"},
		{Kind: "tour", ItemID: "t9", Quantity: 5, Status: "paid"},
	}
	mine := screens.TourBookings(bookings, "t1")
	checkScreen(t, screens.OrganizerTour(i18n.RU, sonKolOwn, 0, &api.StatCounts{Views: 340, Contacts: 21}, mine, links),
		"🧭 <b>Сон-Кёль на лошадях</b>\n📅 12 — 14 июня · 3 дня\n📍 Нарын\n💳 18 000 сом · 👥 3 из 12 мест\n🟢 Опубликован\n⭐ 4.8 · 12 отзывов\n\n"+
			"📊 <b>За 30 дней</b>: 👁 340 просмотров · 📞 21 обращение\n🎫 Онлайн-заявок: 1 · 2 человека",
		"📋 Участники · 3→g:r:t1",
		"📖 Программа · 3 дня→g:tp:0:0",
		"🎫 Заявки→g:tb:0",
		"⬅️ Кабинет→g:home",
	)
	checkScreen(t, screens.OrganizerProgram(i18n.RU, sonKolOwn, 0, 0),
		"📖 <b>Программа · Сон-Кёль на лошадях</b>\n\n<b>День 1. Бишкек — Кочкор</b>\nВыезд в 8:00.\nОбед в Кочкоре.",
		"◀️→g:tp:0:2 | 1 / 3→noop | ▶️→g:tp:0:1",
		"⬅️ К туру→g:t:0",
	)
	checkScreen(t, screens.OrganizerBookings(i18n.RU, sonKolOwn, 0, mine),
		"🎫 <b>Заявки · Сон-Кёль на лошадях</b>\n\n<b>12 июня</b>\n"+
			"1. Айбек · 2 человека · +996 555 123 456 · 36 000 сом\n<i>Вегетарианское меню</i>\n"+
			"2. Dana · 1 человек · +49 151 000 · 18 000 сом · ↩️ возврат",
		"⬅️ К туру→g:t:0",
	)
}

func TestSketchRating(t *testing.T) {
	o := uchitel
	o.Organizer = &api.Person{FirstName: str("Айгуль")}
	o.CanRate = true
	checkScreen(t, screens.RateTour(i18n.RU, o),
		"⭐ <b>Как прошёл тур «Пик Учитель»?</b>\nсб, 24 окт., 06:00 · организатор Айгуль\n\n"+
			"Оцените от 1 до 5. Оценка станет вашим отзывом о туре — так другие поймут, с кем ехать в горы.",
		"1 ★→r:v:o1:1 | 2 ★→r:v:o1:2 | 3 ★→r:v:o1:3 | 4 ★→r:v:o1:4 | 5 ★→r:v:o1:5",
	)
	o.MyRating = &api.RatingMine{Rating: 4}
	checkScreen(t, screens.RateThanks(i18n.RU, o),
		"Спасибо! Ваша оценка тура «Пик Учитель»: ★★★★☆\n\n"+
			"Добавьте пару слов — что понравилось, что можно улучшить. Отзыв увидят на странице тура и организатора.",
		"✍️ Написать отзыв→r:w:o1",
		"Изменить оценку→r:s:o1",
	)
	o.MyRating.Text = str("Отличный темп")
	if s := screens.RateThanks(i18n.RU, o); !strings.HasSuffix(s.Text, "<i>Отличный темп</i>") || layout(s)[0] != "✍️ Изменить отзыв→r:w:o1" {
		t.Errorf("with a comment:\n%s\n%v", s.Text, layout(s))
	}
	checkScreen(t, screens.ReviewPrompt(i18n.RU, o), "Напишите отзыв о туре «Пик Учитель» одним сообщением.", "Отмена→r:v:o1:0")
}

// The profile, as on the site: the account, the participant's form, the
// password, achievements and partnership.
func TestSketchProfile(t *testing.T) {
	me := api.Me{ID: "u1", FirstName: str("Айбек"), Username: str("aibek"), Phone: str("+996 555 123 456"),
		PartnerStatus: api.PartnerApproved, PartnerKind: api.KindOrganizer, HasPassword: true}
	v := screens.ProfileView{Me: me, Joined: []api.Tour{uchitel, alaKol}, JoinedKnown: true,
		Awards: &api.MyAchievements{Points: 80, Awards: make([]api.Award, 2)}, FormKnown: true, Form: &profileForm, Organizing: true}
	checkScreen(t, screens.Profile(i18n.RU, v, at(10, 20, 12, 0)),
		"👤 <b>Айбек</b> · @aibek\n\n"+
			"📞 Телефон: +996 555 123 456\n✉️ Email: не указан\n🔑 Пароль: задан\n\n"+
			"🎒 Туров пройдено: 0 · впереди: 2\n🏆 Достижений: 2 · очков: 80\n"+
			"📝 Анкета участника: ✅ заполнена\n🤝 Партнёрство: ✅ организатор туров",
		"✏️ Изменить имя→profile_edit_name | ✏️ Изменить телефон→profile_edit_phone",
		"📝 Анкета участника→p:form | 🔑 Сменить пароль→p:pw",
		"🏆 Достижения→p:aw | 🤝 Партнёрство→p:pa",
		"🎒 Мои туры→o:list:u | ❤️ Избранное→t:f:0",
		"🧑‍✈️ Кабинет организатора→g:home",
		"🌐 Язык · 🇷🇺 Русский→l:menu",
	)

	// What didn't load is left out; a traveler has no organizer's room.
	traveler := screens.ProfileView{Me: api.Me{ID: "u2", PartnerStatus: api.PartnerPending, PartnerKind: api.KindDriver}, Organizing: true}
	checkScreen(t, screens.Profile(i18n.RU, traveler, at(10, 20, 12, 0)),
		"👤 <b>Без имени</b>\n\n"+
			"📞 Телефон: не указан\n✉️ Email: не указан\n🔑 Пароль: не задан\n\n"+
			"🤝 Партнёрство: ⏳ заявка водителя на рассмотрении",
		"✏️ Изменить имя→profile_edit_name | ✏️ Изменить телефон→profile_edit_phone",
		"📝 Анкета участника→p:form | 🔑 Задать пароль→p:pw",
		"🏆 Достижения→p:aw | 🤝 Партнёрство→p:pa",
		"🎒 Мои туры→o:list:u | ❤️ Избранное→t:f:0",
		"🌐 Язык · 🇷🇺 Русский→l:menu",
	)
}

var profileForm = api.ParticipantForm{EmergencyName: "Мама", EmergencyPhone: "+996 700 000 111", Experience: "trekking",
	Health: str("Аллергия на пыльцу")}

func TestSketchParticipantForm(t *testing.T) {
	why := "Заполните один раз — анкету увидят организаторы туров, на которые вы записываетесь: так они подготовятся и будут знать, кому звонить в экстренном случае."
	checkScreen(t, screens.FormScreen(i18n.RU, &profileForm, ""),
		"📝 <b>Анкета участника</b>\n\n"+why+"\n\n"+
			"🆘 <b>Экстренный контакт:</b> Мама, +996 700 000 111\n"+
			"🧗 <b>Опыт в горах:</b> Многодневные треки\n"+
			"❤️ <b>Здоровье:</b> Аллергия на пыльцу",
		"✏️ Изменить анкету→p:fe",
		"◀️ Профиль→p:home",
	)
	checkScreen(t, screens.FormScreen(i18n.RU, nil, ""),
		"📝 <b>Анкета участника</b>\n\n"+why+"\n\n"+
			"Анкета пока не заполнена. Это необязательно, но организатору важно знать это до выезда.",
		"📝 Заполнить анкету→p:fe",
		"◀️ Профиль→p:home",
	)
	checkScreen(t, screens.FormStep(i18n.RU, screens.StepExperience, "trekking", ""),
		"📝 <b>Анкета участника</b> · 3 из 5\n\nКакой у вас опыт в горах?",
		"Нет опыта→p:fx:none",
		"Однодневные походы→p:fx:hiking",
		"✅ Многодневные треки→p:fx:trekking",
		"Альпинизм→p:fx:mountaineering",
		"Отмена→p:fc",
	)
	checkScreen(t, screens.FormStep(i18n.RU, screens.StepHealth, "Аллергия на пыльцу", ""),
		"📝 <b>Анкета участника</b> · 4 из 5\n\n"+
			"Что о здоровье важно знать организатору — аллергии, хронические болезни, лекарства? Напишите или пропустите.",
		"➡️ Оставить: Аллергия на пыльцу→p:fk",
		"⏭ Пропустить→p:fs",
		"Отмена→p:fc",
	)
}

func TestSketchPassword(t *testing.T) {
	withPassword := api.Me{Phone: str("+996 555 123 456"), HasPassword: true}
	checkScreen(t, screens.PasswordScreen(i18n.RU, withPassword, ""),
		"🔑 <b>Пароль</b>\n\n"+
			"Пароль задан — им можно входить на сайт по email или телефону. После смены пароля выйдем из аккаунта на всех других устройствах.",
		"🔑 Сменить пароль→p:pc",
		"◀️ Профиль→p:home",
	)
	checkScreen(t, screens.PasswordScreen(i18n.RU, api.Me{}, ""),
		"🔑 <b>Пароль</b>\n\n"+
			"Вы входите через Telegram. Чтобы входить на сайт по паролю, к аккаунту нужен email или телефон — добавьте телефон, и пароль можно будет задать.",
		"✏️ Изменить телефон→profile_edit_phone",
		"◀️ Профиль→p:home",
	)
	checkScreen(t, screens.PasswordStep(i18n.RU, screens.StepNew, ""),
		"🔑 <b>Пароль</b>\n\nВведите новый пароль — не короче 8 символов.\n\n🔒 Сообщение с паролем я сразу удалю из чата.",
		"Отмена→p:px",
	)
}

func TestSketchPartner(t *testing.T) {
	checkScreen(t, screens.Partner(i18n.RU, api.Me{PartnerStatus: api.PartnerRejected}, true, ""),
		"🤝 <b>Партнёрство</b>\n\n"+
			"❌ Заявка была отклонена. Вы можете подать её снова.\n\n"+
			"Станьте партнёром Ermekten. Кем вы хотите стать?\n\n"+
			"🧭 <b>Организатор туров</b> — публикуйте туры, ведите группы, клиентов и оплаты.\n"+
			"🚐 <b>Водитель</b> — возите туристические группы: заявки от организаторов, рейсы, транспорт и доходы.",
		"🧭 Организатор туров→p:ap:organizer | 🚐 Водитель→p:ap:driver",
		"◀️ Профиль→p:home",
	)
	checkScreen(t, screens.Partner(i18n.RU, api.Me{PartnerStatus: api.PartnerApproved, PartnerKind: api.KindOrganizer}, true, ""),
		"🤝 <b>Партнёрство</b>\n\n"+
			"✅ Вы организатор туров Ermekten: публикуйте туры, ведите группы и вручайте путешественникам достижения.\n\n"+
			"Создать тур можно на сайте, в «🏔 Открыть Ermekten».",
		"🧑‍✈️ Кабинет организатора→g:home",
		"◀️ Профиль→p:home",
	)
	checkScreen(t, screens.Partner(i18n.RU, api.Me{PartnerStatus: api.PartnerApproved, PartnerKind: api.KindDriver}, true, ""),
		"🤝 <b>Партнёрство</b>\n\n"+
			"✅ Вы водитель-партнёр Ermekten: организаторы туров находят вас в каталоге и присылают заявки на перевозку групп.\n\n"+
			"Заявки, рейсы и транспорт — в кабинете на сайте, в «🏔 Открыть Ermekten».",
		"◀️ Профиль→p:home",
	)
}
