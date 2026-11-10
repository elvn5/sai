// Package cabinet implements the former site's tour and partner workflows as
// native Telegram cards and forms. The registry is the allowlist of operations;
// callback data contains only opaque, short-lived, user-bound action tokens.
package cabinet

type field struct {
	append           bool
	key, label, kind string
	required         bool
	min, max         int
	options          []string
	ref              string
}

type resource struct {
	key, title, list, create, update, remove, detail, extract string
	fields                                                    []field
	filters                                                   []field
	role                                                      string // empty: any account; partner, organizer or driver
	single                                                    bool
}

func textField(key, label string, required bool, max int) field {
	return field{key: key, label: label, kind: "text", required: required, max: max}
}
func number(key, label string, required bool, min, max int) field {
	return field{key: key, label: label, kind: "int", required: required, min: min, max: max}
}
func choice(key, label string, required bool, options ...string) field {
	return field{key: key, label: label, kind: "choice", required: required, options: options}
}
func date(key, label string, required bool) field {
	return field{key: key, label: label, kind: "date", required: required}
}
func ref(key, label, target string, required bool) field {
	return field{key: key, label: label, kind: "ref", ref: target, required: required}
}
func boolean(key, label string) field { return field{key: key, label: label, kind: "bool"} }
func special(key, label, kind string, max int) field {
	return field{key: key, label: label, kind: kind, max: max}
}

var destinations = []string{"issyk_kul", "karakol", "bishkek", "ala_archa", "song_kol", "naryn", "tash_rabat", "kel_suu", "osh", "arslanbob", "sary_chelek", "talas", "batken", "chuy"}
var contacts = []field{textField("phone", "Телефон", false, 32), textField("whatsapp", "WhatsApp", false, 32), textField("telegram", "Telegram", false, 33)}
var period = []field{date("from", "С даты", false), date("to", "По дату", false)}
var tourFields = append([]field{
	textField("title", "Название", true, 255), choice("locale", "Язык", true, "ru", "ky", "en", "de"),
	textField("excerpt", "Краткое описание", false, 500), special("content", "Описание", "html", 20000),
	date("startDate", "Дата начала", true), date("endDate", "Дата окончания", true),
	number("price", "Цена, сом", false, 0, 100000000), number("durationDays", "Длительность, дней", false, 1, 365),
	number("capacity", "Количество мест", false, 1, 500), textField("location", "Место", false, 255),
	choice("destination", "Направление", false, destinations...),
	choice("cancellationPolicy", "Условия отмены", false, "free", "flexible", "strict", "non_refundable"),
	textField("meetingPoint", "Место сбора", false, 200), special("meetingTime", "Время сбора", "time", 5),
	special("coverImageUrl", "Обложка", "photo", 1), special("photos", "Фотографии", "photos", 20),
	special("program", "Программа по дням", "program", 30),
}, contacts...)
var clientFields = []field{
	textField("name", "Имя", true, 120), textField("phone", "Телефон", false, 32), textField("telegram", "Telegram", false, 33),
	textField("note", "Заметка", false, 4000), special("tags", "Метки", "tags", 20),
	choice("source", "Источник", false, "ermekten", "instagram", "whatsapp", "telegram", "phone", "referral", "other"),
}
var paymentFields = []field{
	number("amount", "Сумма, сом", true, 1, 100000000), choice("method", "Способ оплаты", true, "cash", "transfer", "card", "other"),
	date("paidOn", "Дата оплаты", true), textField("note", "Заметка", false, 500),
}
var formFields = []field{
	textField("emergencyName", "Контакт на случай ЧП", true, 120), textField("emergencyPhone", "Телефон контакта", true, 32),
	choice("experience", "Опыт", true, "none", "hiking", "trekking", "mountaineering"),
	textField("health", "Здоровье", false, 2000), textField("notes", "Примечания", false, 2000),
}
var tripFields = []field{
	ref("vehicleId", "Автомобиль", "vehicles", false), ref("clientId", "Клиент", "clients", false),
	special("startsAt", "Отправление", "datetime", 0), special("endsAt", "Возвращение", "datetime", 0),
	textField("routeFrom", "Откуда", true, 200), textField("routeTo", "Куда", true, 200),
	number("passengers", "Пассажиры", false, 1, 80), number("price", "Цена, сом", false, 0, 100000000), textField("note", "Заметка", false, 2000),
}

func registry() map[string]resource {
	resources := []resource{
		{key: "achievements", title: "Достижения", list: "/achievements"},
		{key: "awards", title: "Выданные достижения", role: "partner", list: "/achievements/awards/given", create: "/achievements/awards", fields: []field{textField("code", "Код путешественника", true, 32), ref("achievementId", "Достижение", "achievements", true), textField("note", "Заметка", false, 300)}},
		{key: "tours", title: "Мои объявления", role: "organizer", list: "/tours/mine", create: "/tours", update: "PUT /tours/{id}", remove: "/tours/{id}", fields: tourFields},
		{key: "catalog", title: "Каталог туров", list: "/tours", detail: "/tours/by-id/{id}", filters: append([]field{textField("q", "Поиск", false, 100), choice("destination", "Направление", false, destinations...), choice("duration", "Длительность", false, "1", "2-3", "4+"), boolean("free", "Бесплатная отмена"), choice("sort", "Сортировка", false, "soonest", "price_asc", "price_desc", "rating")}, period...)},
		{key: "clients", title: "Клиенты", role: "partner", list: "/crm/clients", create: "/crm/clients", update: "PUT /crm/clients/{id}", remove: "/crm/clients/{id}", detail: "/crm/clients/{id}", fields: clientFields, filters: []field{textField("q", "Поиск", false, 100), textField("tag", "Метка", false, 40)}},
		{key: "leads", title: "Заявки", role: "partner", list: "/crm/leads", create: "/crm/leads", update: "PATCH /crm/leads/{id}", remove: "/crm/leads/{id}", fields: []field{ref("clientId", "Клиент", "clients", true), ref("tourId", "Тур", "tours", false), choice("stage", "Этап", true, "new", "working", "offer", "won", "lost"), textField("lostReason", "Причина отказа", false, 300), textField("note", "Заметка", false, 2000)}},
		{key: "tasks", title: "Задачи", role: "partner", list: "/crm/tasks?all=true", create: "/crm/tasks", update: "PATCH /crm/tasks/{id}", remove: "/crm/tasks/{id}", fields: []field{textField("title", "Название", true, 200), date("dueOn", "Срок", true), ref("clientId", "Клиент", "clients", false), ref("tourId", "Тур", "tours", false), boolean("done", "Выполнено")}},
		{key: "templates", title: "Шаблоны сообщений", role: "partner", list: "/crm/templates", create: "/crm/templates", update: "PUT /crm/templates/{id}", remove: "/crm/templates/{id}", fields: []field{textField("title", "Название", true, 100), textField("body", "Сообщение", true, 4000)}},
		{key: "broadcasts", title: "Рассылки", role: "partner", list: "/crm/broadcasts", create: "/crm/broadcasts", fields: []field{special("audience", "Получатели", "audience", 80), textField("body", "Сообщение", true, 4000)}},
		{key: "summary", title: "Сводка", role: "organizer", list: "/crm/summary", single: true},
		{key: "reports", title: "Отчёты", role: "organizer", list: "/crm/reports", single: true, filters: period},
		{key: "stats", title: "Просмотры и обращения", role: "partner", list: "/listing-stats/mine", single: true, filters: []field{choice("days", "Период, дней", true, "7", "30", "90")}},
		{key: "reviews", title: "Отзывы о моих турах", role: "partner", list: "/reviews/mine", filters: []field{boolean("unanswered", "Без ответа")}},
		{key: "tourReviews", title: "Отзывы", list: "/reviews?targetType=tour&targetId={tour}", create: "/reviews", fields: []field{number("rating", "Оценка", true, 1, 5), textField("text", "Отзыв", true, 3000), special("photos", "Фотографии", "photos", 6)}},
		{key: "reply", title: "Ответ на отзыв", role: "partner", create: "PUT /reviews/{id}/reply", fields: []field{textField("text", "Ответ", true, 2000)}},
		{key: "posts", title: "Новости тура", list: "/listing-posts?targetType=tour&targetId={tour}", create: "/listing-posts", remove: "/listing-posts/{id}", fields: []field{textField("text", "Текст", false, 3000), special("photos", "Фотографии", "photos", 10)}},
		{key: "group", title: "Группа тура", role: "organizer", list: "/crm/tours/{tour}/participants", extract: "participants", create: "/crm/tours/{tour}/participants", update: "PATCH /crm/tours/{tour}/participants/{id}", remove: "/crm/tours/{tour}/participants/{id}", fields: []field{ref("clientId", "Клиент", "clients", true), choice("status", "Статус", false, "new", "confirmed", "waitlist", "cancelled"), number("amountDue", "К оплате, сом", false, 0, 100000000), boolean("arrived", "Прибыл")}},
		{key: "myForm", title: "Анкета участника", create: "PUT /tours/{tour}/form", fields: formFields},
		{key: "groupForm", title: "Анкета участника", role: "organizer", create: "PUT /crm/tours/{tour}/participants/{id}/form", fields: formFields},
		{key: "payments", title: "Оплаты участника", role: "organizer", list: "/crm/tours/{tour}/participants", extract: "payments", create: "/crm/tours/{tour}/participants/{participant}/payments", remove: "/crm/payments/{id}", fields: append([]field{choice("kind", "Тип операции", true, "payment", "refund")}, paymentFields...)},
		{key: "expenses", title: "Расходы тура", role: "organizer", list: "/crm/tours/{tour}/participants", extract: "expenses", create: "/crm/tours/{tour}/expenses", remove: "/crm/expenses/{id}", fields: []field{choice("category", "Категория", true, "transport", "guides", "food", "accommodation", "permits", "gear", "other"), number("amount", "Сумма, сом", true, 1, 100000000), textField("note", "Заметка", false, 500)}},
		{key: "staff", title: "Команда тура", role: "organizer", list: "/crm/tours/{tour}/participants", extract: "staff", create: "/crm/tours/{tour}/staff", remove: "/crm/staff/{id}", fields: []field{choice("role", "Роль", true, "guide", "assistant", "driver", "cook", "other"), textField("name", "Имя", true, 120), textField("phone", "Телефон", false, 32), textField("note", "Заметка", false, 500)}},
		{key: "drivers", title: "Каталог водителей", role: "organizer", list: "/crm/drivers", filters: []field{textField("q", "Поиск", false, 100), date("date", "Дата", false), number("seats", "Количество мест", false, 1, 80)}},
		{key: "transport", title: "Транспорт тура", role: "organizer", list: "/crm/tours/{tour}/transport", create: "/crm/tours/{tour}/transport", remove: "/crm/transport/{id}", fields: append([]field{ref("driverId", "Водитель", "drivers", true)}, tripFields[2:]...)},
		{key: "driverSummary", title: "Сводка водителя", role: "driver", list: "/driver/summary", single: true},
		{key: "driverProfile", title: "Профиль водителя", role: "driver", list: "/driver/profile", single: true, update: "PUT /driver/profile", fields: []field{textField("name", "Имя", true, 120), textField("phone", "Телефон", false, 32), textField("region", "Регион", false, 120), textField("about", "О себе", false, 2000), boolean("published", "В каталоге")}},
		{key: "vehicles", title: "Автомобили", role: "driver", list: "/driver/vehicles", create: "/driver/vehicles", update: "PUT /driver/vehicles/{id}", remove: "/driver/vehicles/{id}", fields: []field{choice("kind", "Тип автомобиля", true, "sedan", "minivan", "minibus", "bus", "offroad", "other"), textField("model", "Модель", true, 120), textField("plate", "Госномер", false, 20), number("seats", "Количество мест", true, 1, 80), number("year", "Год", false, 1950, 2100), date("insuranceUntil", "Страховка до", false), date("inspectionUntil", "Техосмотр до", false), textField("note", "Заметка", false, 500), boolean("active", "Активен")}},
		{key: "trips", title: "Поездки водителя", role: "driver", list: "/driver/trips", detail: "/driver/trips/{id}", create: "/driver/trips", update: "PUT /driver/trips/{id}", remove: "/driver/trips/{id}", fields: tripFields, filters: append([]field{choice("status", "Статус", false, "requested", "accepted", "declined", "done", "cancelled")}, period...)},
		{key: "tripStatus", title: "Статус поездки", role: "driver", create: "PUT /driver/trips/{id}/status", fields: []field{choice("status", "Статус", true, "accepted", "declined", "done", "cancelled"), ref("vehicleId", "Автомобиль", "vehicles", false), number("price", "Цена, сом", false, 0, 100000000), textField("declineReason", "Причина отказа", false, 300)}},
		{key: "driverPayments", title: "Оплаты поездки", role: "driver", list: "/driver/trips/{trip}", extract: "payments", create: "/driver/trips/{trip}/payments", remove: "/driver/payments/{id}", fields: paymentFields},
		{key: "driverExpenses", title: "Расходы водителя", role: "driver", list: "/driver/expenses", create: "/driver/expenses", remove: "/driver/expenses/{id}", fields: []field{choice("category", "Категория", true, "fuel", "repair", "maintenance", "wash", "parking", "fines", "insurance", "other"), number("amount", "Сумма, сом", true, 1, 100000000), date("spentOn", "Дата расхода", true), ref("vehicleId", "Автомобиль", "vehicles", false), ref("tripId", "Поездка", "trips", false), textField("note", "Заметка", false, 500)}, filters: period},
		{key: "driverReports", title: "Отчёты водителя", role: "driver", list: "/driver/reports", single: true, filters: period},
		{key: "chats", title: "Переписка", list: "/chats", detail: "/chats/{id}/messages"},
		{key: "message", title: "Написать сообщение", create: "/chats/{id}/messages", fields: []field{textField("body", "Сообщение", true, 4000)}},
		{key: "finances", title: "Баланс и выплаты", role: "partner", list: "/payouts/mine", single: true},
		{key: "bookings", title: "Оплаченные бронирования", role: "partner", list: "/orders/received"},
	}
	out := make(map[string]resource, len(resources))
	for _, r := range resources {
		switch r.key {
		case "leads":
			r.filters = []field{choice("stage", "Этап", false, "new", "working", "offer", "won", "lost"), ref("clientId", "Клиент", "clients", false)}
		case "tasks":
			r.filters = []field{boolean("all", "Включая выполненные"), ref("clientId", "Клиент", "clients", false)}
		case "group":
			r.filters = []field{choice("status", "Статус", false, "new", "confirmed", "waitlist", "cancelled"), textField("q", "Поиск", false, 100)}
		case "tours":
			r.filters = []field{textField("q", "Поиск", false, 100), choice("phase", "Период тура", false, "upcoming", "current", "past")}
		}
		out[r.key] = r
	}
	return out
}
