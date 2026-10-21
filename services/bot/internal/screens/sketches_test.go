package screens_test

// Each artboard of the design sketches ("Ermekten — Telegram-бот" canvas),
// rendered from the data drawn on it: the text and the button layout must
// match the sketch exactly. Changing a screen means changing its sketch.

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/drafts"
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
		"🔎 Найти выход→o:find:0 | ➕ Собрать выход→d:new",
		"🧑‍✈️ Кабинет организатора→g:home",
	)
}

func TestSketchFindOutings(t *testing.T) {
	alaKolFull := alaKol
	alaKolFull.Taken, alaKolFull.IsJoined = 10, false
	komsomolets := api.Outing{ID: "o4", Title: "Пик Комсомолец", StartsAt: at(11, 7, 7, 0), Status: "open",
		Seats: 8, Taken: 2, Trail: &api.TrailRef{Name: "Ала-Арча"}}
	more := api.Outing{ID: "o5", Title: "Кёль-Тор", StartsAt: at(11, 8, 8, 0), Status: "open", Seats: 15, Taken: 9}
	s := screens.FindOutings([]api.Outing{uchitel, barskoon, alaKolFull, komsomolets, more}, 0)
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
		"➕ Собрать выход→d:new | ⬅️ Мои выходы→o:list:u",
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
		"🙅 Не смогу пойти→o:ask:u:o1",
		"📤 Позвать друзей→share:https://t.me/ermekten_bot?start=outing_o1",
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

	s := screens.Blog(posts, 0, false, links)
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
	caption, markup := screens.BlogPost(post)
	checkScreen(t, screens.Screen{Text: caption, Markup: markup},
		"📰 <b>Новая статья в блоге</b>\n\n"+
			"<b>Осенняя Ала-Арча: 5 маршрутов на один день</b>\n"+
			"От лёгкой прогулки к водопаду до подъёма на Рацек — с перепадами высот и временем в пути.",
		"📖 Читать→b:read:p9:0 | 🔕 Не присылать→n:blog_off",
	)
}

func TestSketchOrganizerCard(t *testing.T) {
	o := barskoon
	o.MeetingPoint, o.Transport, o.Level, o.CostPerPerson = "Южные ворота", "arranged", "beginner", num(800)
	o.Participants = []api.Person{{FirstName: str("Айбек")}, {FirstName: str("Нурлан")}, {Username: str("aidai_kg")}, {FirstName: str("Мээрим")}}
	checkScreen(t, screens.OutingCard(o, screens.FromMine, links, at(10, 20, 10, 6)),
		"🥾 <b>Водопад Барскоон</b>\n🏔 Барскоон · для новичков\n\n"+
			"📅 вс, 1 нояб., 07:30\n"+
			"📍 Сбор: Южные ворота\n"+
			"🚗 Заказной транспорт\n"+
			"💳 800 сом с человека\n"+
			"👥 ▰▰▰▱▱▱▱▱▱▱ 4 из 12 мест\n\n"+
			"👥 Идут: Айбек, Нурлан, @aidai_kg, Мээрим\n\n"+
			"⭐ Вы организатор",
		"✏️ Изменить→o:edit:u:o2 | ❌ Отменить выход→o:cask:u:o2",
		"📋 Участники и отметка→g:r:o2",
		"📤 Позвать друзей→share:https://t.me/ermekten_bot?start=outing_o2",
		"⬅️ К моим выходам→o:list:u",
	)
	checkScreen(t, screens.CancelConfirm(o, screens.FromMine),
		"Отменить выход «Водопад Барскоон»?\n\nУчастников: 4 — бот сообщит им об отмене.\n\nВернуть выход после отмены не получится.",
		"Да, отменить→o:cancel:u:o2 | Нет, оставить→o:view:u:o2",
	)
}

// The new outing's steps drawn on the sketches: the route, the day, the
// review.
func TestSketchNewOuting(t *testing.T) {
	now := at(10, 20, 10, 5)
	d := drafts.New("aibek", now)
	d.Trails = []api.TrailRef{{ID: "a", Name: "Пик Учитель"}, {ID: "b", Name: "Водопад Барскоон"}, {ID: "c", Name: "Озеро Ала-Кёль"},
		{ID: "d", Name: "Кёль-Тор"}, {ID: "e", Name: "Пик Комсомолец"}, {ID: "f", Name: "Сон-Кёль"}}
	checkScreen(t, screens.DraftScreen(d, now),
		"➕ <b>Новый выход</b> · шаг 1 из 12\n\n"+
			"🏔 <b>Куда идём?</b>\nВыберите маршрут из каталога или напишите его название — я поищу.",
		"Пик Учитель→d:trail:0 | Водопад Барскоон→d:trail:1",
		"Озеро Ала-Кёль→d:trail:2 | Кёль-Тор→d:trail:3",
		"Пик Комсомолец→d:trail:4 | Сон-Кёль→d:trail:5",
		"Без маршрута из каталога→d:trail:none",
		"✖️ Отмена→d:cancel",
	)

	d.TrailID, d.TrailName, d.Title, d.Step = "a", "Пик Учитель", "Пик Учитель за один день", drafts.StepDate
	checkScreen(t, screens.DraftScreen(d, now),
		"➕ <b>Новый выход</b> · шаг 3 из 12\n\n"+
			"📅 <b>Когда выходим?</b>\nВыберите день в календаре или напишите дату, например 24.10.",
		" →noop | Октябрь 2026→noop | ▶️→d:month:2026-11",
		"Пн→noop | Вт→noop | Ср→noop | Чт→noop | Пт→noop | Сб→noop | Вс→noop",
		"·→noop | 20→d:date:2026-10-20 | 21→d:date:2026-10-21 | 22→d:date:2026-10-22 | 23→d:date:2026-10-23 | 24→d:date:2026-10-24 | 25→d:date:2026-10-25",
		"26→d:date:2026-10-26 | 27→d:date:2026-10-27 | 28→d:date:2026-10-28 | 29→d:date:2026-10-29 | 30→d:date:2026-10-30 | 31→d:date:2026-10-31 |  →noop",
		"⬅️ Назад→d:back | ✖️ Отмена→d:cancel",
	)

	d.Date, d.Time, d.Returns = "2026-10-24", "06:00", "2026-10-24T19:00"
	d.MeetingPoint, d.Contact, d.Step, d.Reviewing = "ТЦ «Вефа», парковка", "@aibek", drafts.StepReview, true
	checkScreen(t, screens.DraftScreen(d, now),
		"👀 <b>Проверьте выход</b>\n\n"+
			"🥾 <b>Пик Учитель за один день</b>\n🏔 Пик Учитель · для новичков\n\n"+
			"📅 сб, 24 окт., 06:00 → 19:00\n"+
			"📍 Сбор: ТЦ «Вефа», парковка\n"+
			"🚗 На машине организатора\n"+
			"👥 Мест: 3\n"+
			"💬 Связь: @aibek\n\n"+
			"Всё верно? Выход появится в ближайших — в боте и на сайте, а бот будет сообщать вам о каждом, кто запишется.",
		"✅ Опубликовать→d:save",
		"✏️ Изменить→d:fields | ✖️ Отмена→d:cancel",
	)
}

func TestSketchTourMore(t *testing.T) {
	tour := api.Tour{ID: "t2", Slug: "son-kol", Title: "Сон-Кёль: ночь в юрте", Location: str("Нарын"), DurationDays: num(3),
		StartDate: "2027-06-12", EndDate: "2027-06-14", Price: num(18000), Capacity: num(12),
		Content: `<p>Три дня на высокогорном озере: перевал <strong>Калмак-Ашуу</strong>, табуны на джайлоо и ночь в юрте.</p>` +
			`<h3>Что включено</h3><ul><li>Транспорт из Бишкека</li><li>Юрта и питание</li></ul>`,
		Program: []api.ProgramDay{
			{Title: "Бишкек — Кочкор", Text: "Выезд в 8:00, обед в Кочкоре."},
			{Title: "Сон-Кёль", Text: "Конная прогулка, закат."},
		},
		Telegram: str("@sonkol_tours"), Phone: str("+996 555 123 456")}
	checkScreen(t, screens.TourMore(tour, screens.TabFavorites, 1, 0, true, links),
		"🧭 <b>Сон-Кёль: ночь в юрте</b>\n"+
			"📍 Нарын · 3 дня\n📅 12 — 14 июня\n💳 18 000 сом с человека\n\n"+
			"👥 В группе до 12 человек\n\n"+
			"Три дня на высокогорном озере: перевал <b>Калмак-Ашуу</b>, табуны на джайлоо и ночь в юрте.\n\n"+
			"<b>Что включено</b>\n\n"+
			"• Транспорт из Бишкека\n• Юрта и питание\n\n"+
			"🗓 <b>Программа</b>\n\n"+
			"<b>День 1. Бишкек — Кочкор</b>\nВыезд в 8:00, обед в Кочкоре.\n\n"+
			"<b>День 2. Сон-Кёль</b>\nКонная прогулка, закат.\n\n"+
			"📞 <b>Записаться и спросить</b>\nTelegram: @sonkol_tours\nТелефон: +996 555 123 456",
		"💔 Из избранного→t:drop:f:1:t2:more | ⬅️ К турам→t:f:1",
	)
}

func TestSketchArticle(t *testing.T) {
	published := at(10, 18, 9, 0)
	long := strings.Repeat("Литров на 30, с поясным ремнём и чехлом от дождя. ", 70)
	post := api.BlogPost{ID: "p1", Title: "Что взять в первый поход", PublishedAt: &published,
		Content: `<h2>Рюкзак</h2><p>` + long + `</p><h2>Одежда</h2><ul><li>Мембранная куртка</li><li>Флиска</li></ul>`}
	first := screens.Article(post, 0, 0, links)
	if !strings.HasPrefix(first.Text, "📰 <b>Что взять в первый поход</b> · 1/2\n<i>18 окт. 2026</i>\n\n<b>Рюкзак</b>\n\nЛитров на 30,") {
		t.Errorf("first page:\n%s", first.Text)
	}
	if got := strings.Join(layout(first), "\n"); got != "Дальше ▶️→b:read:p1:1\n⬅️ К статьям→b:0" {
		t.Errorf("first page buttons:\n%s", got)
	}
	second := screens.Article(post, 0, 1, links)
	if !strings.HasPrefix(second.Text, "📰 <b>Что взять в первый поход</b> · 2/2\n\nЛитров на 30,") ||
		!strings.HasSuffix(second.Text, "от дождя.\n\n<b>Одежда</b>\n\n• Мембранная куртка\n• Флиска") {
		t.Errorf("second page:\n%s", second.Text)
	}
	if got := strings.Join(layout(second), "\n"); got != "◀️ Назад→b:read:p1:0\n⬅️ К статьям→b:0" {
		t.Errorf("second page buttons:\n%s", got)
	}
}

func TestSketchCondition(t *testing.T) {
	o := alaKol
	o.Trail = &api.TrailRef{Slug: "ala-kol", Name: "Ала-Кёль"}
	checkScreen(t, screens.ConditionState(o),
		"📝 <b>Как маршрут «Ала-Кёль»?</b>\n\n"+
			"Расскажите в два нажатия — сводка появится на странице маршрута и поможет тем, кто пойдёт следом.\n\n"+
			"<b>Какой была тропа?</b>",
		"☀️ Сухо, тропа чистая→c:t:o3:0 | 💧 Мокро, грязь→c:t:o3:1",
		"🌨 Местами снег→c:t:o3:2 | ❄️ Снег→c:t:o3:3",
		"⛄ Глубокий снег→c:t:o3:4 | 🧊 Лёд→c:t:o3:5",
		"⬅️ К выходу→o:view:p:o3",
	)
	checkScreen(t, screens.ConditionTraction(o, 2),
		"📝 <b>Как маршрут «Ала-Кёль»?</b>\n\nТропа: 🌨 Местами снег\n\n<b>Что понадобилось на ноги?</b>",
		"👟 Хватило обычной обуви→c:d:o3:2:0",
		"🦶 Пригодились микрошипы→c:d:o3:2:1",
		"⛓ Нужны кошки→c:d:o3:2:2",
		"⬅️ Назад→c:s:o3",
	)
}

// The organizer's room: the outings they run and their tours.
var (
	barskoonDay = func() api.Outing {
		o := barskoon
		o.ID, o.StartsAt, o.MeetingPoint, o.Taken, o.Seats = "o2", at(10, 25, 7, 30), "Южные ворота", 4, 12
		joined := at(10, 18, 12, 0)
		arrived := at(10, 25, 7, 20)
		o.Participants = []api.Person{{ID: "u-aibek-1", FirstName: str("Айбек")}, {ID: "u-nurlan-2", FirstName: str("Нурлан"), Username: str("nurlan_kg")},
			{ID: "u-aidai-33", Username: str("aidai_kg")}, {ID: "u-meerim-4"}}
		for i, p := range o.Participants {
			e := api.RosterEntry{Person: p, JoinedAt: joined}
			if i < 2 {
				e.ArrivedAt = &arrived
			}
			o.Roster = append(o.Roster, e)
		}
		return o
	}()
	sonKolOwn = api.Tour{ID: "t1", Slug: "son-kol", Title: "Сон-Кёль на лошадях", StartDate: "2027-06-12", EndDate: "2027-06-14",
		DurationDays: num(3), Location: str("Нарын"), Price: num(18000), Capacity: num(12), IsPublished: true,
		RatingAverage: 4.8, ReviewCount: 12,
		Program: []api.ProgramDay{{Title: "Бишкек — Кочкор", Text: "Выезд в 8:00.\nОбед в Кочкоре."}, {Title: "Сон-Кёль", Text: "Конная прогулка."}, {Title: "Обратно"}}}
)

func TestSketchOrganizerHome(t *testing.T) {
	draft := api.Tour{ID: "t2", Title: "Ала-Кёль", StartDate: "2027-07-03", EndDate: "2027-07-05"}
	checkScreen(t, screens.OrganizerHome([]api.Outing{barskoonDay}, []api.Tour{sonKolOwn, draft}, at(10, 25, 7, 25)),
		"🧑‍✈️ <b>Кабинет организатора</b>\nУчастники, отметка на сборе и ваши туры.\n\n"+
			"🥾 <b>Ваши выходы</b>\n\n"+
			"<b>1. Водопад Барскоон</b>\nвс, 25 окт., 07:30 · 👥 4/12 · ✅ на месте 2 из 4\n\n"+
			"🧭 <b>Ваши туры</b>\n\n"+
			"<b>2. Сон-Кёль на лошадях</b>\n12 — 14 июня · 🟢 опубликован\n\n"+
			"<b>3. Ала-Кёль</b>\n3 — 5 июля · 📝 черновик",
		"1 · Водопад Барскоон→g:r:o2",
		"2 · Сон-Кёль на лошадях→g:t:0",
		"3 · Ала-Кёль→g:t:1",
		"➕ Собрать выход→d:new | ⬅️ Мои выходы→o:list:u",
	)
	if s := screens.OrganizerHome(nil, nil, at(10, 25, 7, 25)); !strings.Contains(s.Text, "Вы пока ничего не организуете") {
		t.Errorf("empty room:\n%s", s.Text)
	}
}

func TestSketchRoster(t *testing.T) {
	chats := map[string]int64{"u-aibek-1": 1001, "u-nurlan-2": 1002}
	checkScreen(t, screens.Roster(barskoonDay, chats, links, at(10, 25, 7, 25)),
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
		"📤 Позвать друзей→share:https://t.me/ermekten_bot?start=outing_o2",
		"🥾 Карточка выхода→o:view:u:o2 | ⬅️ Кабинет→g:home",
	)
	// A week ahead: no marks yet.
	early := screens.Roster(barskoonDay, nil, links, at(10, 18, 13, 0))
	if !strings.Contains(early.Text, "1. • Айбек · с 18 окт.") || !strings.HasSuffix(early.Text, "Отмечать прибывших можно будет за сутки до сбора.") ||
		strings.Contains(strings.Join(layout(early), "\n"), "g:a:") {
		t.Errorf("before check-in:\n%s\n%v", early.Text, layout(early))
	}
}

func TestSketchBroadcast(t *testing.T) {
	o := barskoonDay
	o.Organizer = api.Person{FirstName: str("Айгуль")}
	checkScreen(t, screens.BroadcastPrompt(o),
		"📣 <b>Сообщение участникам</b>\n«Водопад Барскоон» · 4 человека\n\n"+
			"Напишите сообщение — бот перешлёт его всем, кто записался. Например: "+
			"<i>«Сбор переносится на 06:30, ждём у южного входа»</i>.",
		"Отмена→g:mx:o2",
	)
	checkScreen(t, screens.BroadcastPreview(o, "Ждём <у ворот>"),
		"Отправить участникам «Водопад Барскоон» (4 человека)?\n\n<blockquote>Ждём &lt;у ворот&gt;</blockquote>\n\n"+
			"<i>Чтобы исправить, просто напишите сообщение заново.</i>",
		"📣 Отправить→g:ms:o2 | Отмена→g:mx:o2",
	)
	checkScreen(t, screens.BroadcastMessage(o, "Ждём у ворот"),
		"📣 <b>Сообщение от организатора</b> · Айгуль\nВыход «Водопад Барскоон», вс, 25 окт., 07:30\n\nЖдём у ворот",
		"Открыть выход→o:view:n:o2",
	)
	checkScreen(t, screens.BroadcastSent(o, 3, 4),
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
	checkScreen(t, screens.OrganizerTour(sonKolOwn, 0, &api.StatCounts{Views: 340, Contacts: 21}, mine, links),
		"🧭 <b>Сон-Кёль на лошадях</b>\n📅 12 — 14 июня · 3 дня\n📍 Нарын\n💳 18 000 сом · 👥 мест: 12\n🟢 Опубликован\n⭐ 4.8 · 12 отзывов\n\n"+
			"📊 <b>За 30 дней</b>: 👁 340 просмотров · 📞 21 обращение\n🎫 Онлайн-заявок: 1 · 2 человека",
		"📖 Программа · 3 дня→g:tp:0:0",
		"🎫 Заявки→g:tb:0",
		"⬅️ Кабинет→g:home",
	)
	checkScreen(t, screens.OrganizerProgram(sonKolOwn, 0, 0),
		"📖 <b>Программа · Сон-Кёль на лошадях</b>\n\n<b>День 1. Бишкек — Кочкор</b>\nВыезд в 8:00.\nОбед в Кочкоре.",
		"◀️→g:tp:0:2 | 1 / 3→noop | ▶️→g:tp:0:1",
		"⬅️ К туру→g:t:0",
	)
	checkScreen(t, screens.OrganizerBookings(sonKolOwn, 0, mine),
		"🎫 <b>Заявки · Сон-Кёль на лошадях</b>\n\n<b>12 июня</b>\n"+
			"1. Айбек · 2 человека · +996 555 123 456 · 36 000 сом\n<i>Вегетарианское меню</i>\n"+
			"2. Dana · 1 человек · +49 151 000 · 18 000 сом · ↩️ возврат",
		"⬅️ К туру→g:t:0",
	)
}

func TestSketchRating(t *testing.T) {
	o := uchitel
	o.Organizer = api.Person{FirstName: str("Айгуль")}
	o.CanRate = true
	checkScreen(t, screens.RateOuting(o),
		"⭐ <b>Как прошёл выход «Пик Учитель»?</b>\nсб, 24 окт., 06:00 · организатор Айгуль\n\n"+
			"Оцените от 1 до 5. Оценка появится на странице организатора — так другие поймут, с кем идти в горы.",
		"1 ★→r:v:o1:1 | 2 ★→r:v:o1:2 | 3 ★→r:v:o1:3 | 4 ★→r:v:o1:4 | 5 ★→r:v:o1:5",
	)
	o.MyRating = &api.RatingMine{Rating: 4}
	o.Trail = &api.TrailRef{Name: "Ала-Арча", Slug: "uchitel"}
	checkScreen(t, screens.RateThanks(o, at(10, 25, 10, 0)),
		"Спасибо! Ваша оценка выхода «Пик Учитель»: ★★★★☆\n\n"+
			"Добавьте пару слов — что понравилось, что можно улучшить. Отзыв увидят на странице организатора.",
		"✍️ Написать отзыв→r:w:o1",
		"📝 Рассказать, как маршрут→c:s:o1",
		"Изменить оценку→r:s:o1",
	)
	o.MyRating.Text = str("Отличный темп")
	if s := screens.RateThanks(o, at(10, 25, 10, 0)); !strings.HasSuffix(s.Text, "<i>Отличный темп</i>") || layout(s)[0] != "✍️ Изменить отзыв→r:w:o1" {
		t.Errorf("with a comment:\n%s\n%v", s.Text, layout(s))
	}
	checkScreen(t, screens.ReviewPrompt(o), "Напишите отзыв о выходе «Пик Учитель» одним сообщением.", "Отмена→r:v:o1:0")
}
