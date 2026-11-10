package cabinet

import (
	"html"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/mymmrac/telego"

	"github.com/elvn5/sai/services/bot/internal/keyboards"
	"github.com/elvn5/sai/services/bot/internal/screens"
)

// Keys not listed here or in the field registry stay internal (UUIDs, tokens,
// ownership columns). Nested records use the same readable labels.
var labels = map[string]string{
	"user": "Получатель", "description": "Описание", "owed": "Задолженность", "owedOrders": "Количество", "orders": "Количество", "guests": "Участники", "fee": "Комиссия", "held": "Проведено туров", "average": "Средняя оценка", "count": "Количество отзывов", "reviews": "Отзывы", "repliedAt": "Дата ответа", "sender": "Автор", "achievement": "Достижение", "recipient": "Получатель", "points": "Баллы",
	"totals": "Итого", "stats": "Статистика", "views": "Просмотры", "contacts": "Обращения", "previous": "Предыдущий период", "daily": "По дням", "day": "Дата", "listings": "Объявления", "listing": "Объявление",
	"clients": "Клиенты", "paidThisMonth": "Оплачено за месяц", "outstanding": "Задолженность", "upcoming": "Ближайшие туры", "taken": "Занято мест", "waitlist": "Лист ожидания", "due": "К оплате, сом", "paid": "Оплачено, сом", "active": "Активен", "arrived": "Прибыл", "forms": "Анкеты", "waivers": "Согласия", "expenses": "Расходы", "margin": "Остаток, сом",
	"participants": "Участники", "account": "Аккаунт", "firstName": "Имя", "username": "Telegram", "form": "Анкета участника", "formAt": "Анкета заполнена", "waiverAt": "Правила приняты", "arrivedAt": "Время прибытия", "joinedAt": "Дата записи", "history": "История туров", "lastTour": "Последний тур", "leads": "Заявки", "tasks": "Задачи", "tour": "Тур", "client": "Клиент", "payments": "Оплаты", "staff": "Команда тура",
	"from": "С даты", "to": "По дату", "tours": "Туры", "places": "Количество мест", "noShows": "Не пришли", "cancelled": "Отменено", "repeatClients": "Повторные клиенты", "revenue": "Выручка, сом", "byTour": "По турам", "total": "Всего", "won": "Успешно", "lost": "Отказ", "createdAt": "Создано", "updatedAt": "Обновлено", "doneAt": "Выполнено",
	"peer": "Собеседник", "lastMessage": "Последнее сообщение", "unread": "Непрочитанные", "messages": "Сообщения", "peerReadAt": "Прочитано собеседником", "sent": "Доставлено", "skipped": "Пропущено", "reply": "Ответ", "author": "Автор", "ratingAverage": "Средняя оценка", "reviewCount": "Количество отзывов", "summary": "Сводка", "photos": "Фотографии",
	"vehicle": "Автомобиль", "vehicles": "Автомобили", "driver": "Водитель", "requests": "Заявки", "monthIncome": "Доход за месяц", "monthExpenses": "Расходы за месяц", "documents": "Документы", "until": "Действует до", "hasProfile": "Профиль заполнен", "trips": "Поездки", "income": "Доход, сом", "profit": "Прибыль, сом", "byMonth": "По месяцам", "month": "Месяц", "byCategory": "По категориям", "byVehicle": "По автомобилям", "busy": "Занят", "expenseList": "Расходы",
	"isPublished": "Опубликован", "isJoined": "Вы записаны", "myStatus": "Ваша запись", "myForm": "Анкета участника", "myWaiverAt": "Правила приняты", "organizer": "Организатор", "earned": "Заработано", "balance": "Баланс, сом", "paidOut": "Выплачено", "payouts": "Выплаты", "available": "Доступно", "monthEarned": "Заработано за месяц", "quantity": "Количество", "amount": "Сумма, сом", "status": "Статус", "itemDate": "Дата", "paidAt": "Дата оплаты", "method": "Способ оплаты", "note": "Заметка",
}

func (v *view) label(key string) string {
	if l, ok := labels[key]; ok {
		return v.l.T(l)
	}
	for _, r := range v.e.resources {
		for _, f := range append(append([]field{}, r.fields...), r.filters...) {
			if f.key == key {
				return v.l.T(f.label)
			}
		}
	}
	return ""
}
func (v *view) describe(data record) string { return strings.Join(v.paragraphs(data, 0), "\n") }
func (v *view) paragraphs(data record, depth int) []string {
	if depth > 5 {
		return nil
	}
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := []string{}
	for _, key := range keys {
		value := data[key]
		if value == nil {
			continue
		}
		label := v.label(key)
		if label == "" {
			continue
		}
		switch value := value.(type) {
		case map[string]any:
			sub := v.paragraphs(value, depth+1)
			if len(sub) > 0 {
				out = append(out, "<b>"+html.EscapeString(label)+"</b>\n"+strings.Join(sub, "\n"))
			}
		case []any:
			if len(value) == 0 {
				continue
			}
			out = append(out, "<b>"+html.EscapeString(label)+"</b>")
			for _, item := range value {
				if m, ok := item.(map[string]any); ok {
					out = append(out, v.paragraphs(m, depth+1)...)
				} else {
					if key == "photos" {
						src := str(item)
						link := v.e.Links.Picture(&src)
						if link != "" {
							out = append(out, "<a href=\""+html.EscapeString(link)+"\">"+v.l.T("Фотографии")+"</a>")
						}
					} else {
						out = append(out, html.EscapeString(str(item)))
					}
				}
			}
		case bool:
			text := "Нет"
			if value {
				text = "Да"
			}
			out = append(out, html.EscapeString(label)+": "+v.l.T(text))
		default:
			text := str(value)
			if text == "" {
				continue
			}
			if key == "content" {
				out = append(out, screens.Paragraphs(text)...)
			} else {
				out = append(out, html.EscapeString(label)+": "+html.EscapeString(v.option(text)))
			}
		}
	}
	return out
}
func (v *view) card(r resource, vars map[string]string, item record, back action, page int) (screens.Screen, error) {
	if err := v.authorize(r); err != nil {
		return screens.Screen{}, err
	}
	// Refresh mutable cards after saves and changes from other devices.
	if r.single {
		fresh, err := v.request("GET", pathFor(r.list, vars), nil)
		if err != nil {
			return screens.Screen{}, err
		}
		item = object(fresh)
	}
	if r.update != "" && !r.single && r.detail == "" {
		fresh, err := v.request("GET", pathFor(r.list, vars), nil)
		if err != nil {
			return screens.Screen{}, err
		}
		raw := fresh
		if r.extract != "" {
			raw = object(fresh)[r.extract]
		} else if items, ok := object(fresh)["items"]; ok {
			raw = items
		}
		found := false
		for _, candidate := range array(raw) {
			if str(candidate["id"]) == vars["id"] {
				item = candidate
				found = true
				break
			}
		}
		if !found {
			return back(v)
		}
	}
	if r.key == "chats" {
		v.s.form = nil
	}
	if r.detail != "" {
		data, err := v.request("GET", pathFor(r.detail, vars), nil)
		if err != nil {
			return screens.Screen{}, err
		}
		item = object(data)
	}
	if r.key == "chats" {
		for _, m := range array(item["messages"]) {
			if str(m["senderId"]) == v.user {
				m["sender"] = v.l.T("Вы")
			} else {
				m["sender"] = title(object(item["peer"]))
			}
		}
	}
	localize(item, string(v.l))
	pages := screens.Pages(v.paragraphs(item, 0), 1800)
	if len(pages) == 0 {
		pages = []string{v.l.T("Пока ничего нет.")}
	}
	page = min(max(page, 0), len(pages)-1)
	again := func(v *view) (screens.Screen, error) { return v.card(r, vars, item, back, page) }
	if page > 0 {
		v.button("◀️ Назад", func(v *view) (screens.Screen, error) { return v.card(r, vars, item, back, page-1) })
	}
	if page+1 < len(pages) {
		v.button("Дальше ▶️", func(v *view) (screens.Screen, error) { return v.card(r, vars, item, back, page+1) })
	}
	if r.update != "" {
		v.button("Изменить", func(v *view) (screens.Screen, error) { return v.begin(r, vars, item, true, again) })
	}
	if r.remove != "" && (r.key != "posts" || str(object(item["author"])["id"]) == v.user) {
		v.button("Удалить", func(v *view) (screens.Screen, error) {
			return v.confirmDelete(pathFor(r.remove, vars), title(item), back)
		})
	}
	switch r.key {
	case "tours":
		tv := map[string]string{"tour": str(item["id"])}
		for _, key := range []string{"group", "expenses", "staff", "transport", "posts", "tourReviews"} {
			v.button(v.e.resources[key].title, v.listAction(key, tv, nil, 0))
		}
		v.button("Скопировать тур", func(v *view) (screens.Screen, error) {
			copy := record{}
			for k, x := range item {
				copy[k] = x
			}
			delete(copy, "startDate")
			delete(copy, "endDate")
			return v.begin(r, nil, copy, false, back)
		})
	case "catalog":
		_ = v.e.API.CabinetRequest(v.ctx, v.user, "POST", "/listing-stats/events", record{"targetType": "tour", "targetId": item["id"], "event": "view"}, nil)
		id := str(item["id"])
		tv := map[string]string{"tour": id}
		if item["isJoined"] == true || item["myStatus"] == "waitlist" {
			v.button("Анкета участника", func(v *view) (screens.Screen, error) {
				return v.begin(v.e.resources["myForm"], tv, object(item["myForm"]), false, again)
			})
			v.button("Правила участия и риски", func(v *view) (screens.Screen, error) { return v.waiver(id, again) })
		}
		v.rows = append(v.rows, []telego.InlineKeyboardButton{keyboards.Button(v.l.T("🎒 Моя запись"), "o:view:n:"+id), keyboards.Button(v.l.T("❤️ В избранное"), "t:open:"+id)})
		for _, contact := range []string{"phone", "whatsapp", "telegram"} {
			if value := str(item[contact]); value != "" {
				v.button(v.label(contact), func(v *view) (screens.Screen, error) {
					_ = v.e.API.CabinetRequest(v.ctx, v.user, "POST", "/listing-stats/events", record{"targetType": "tour", "targetId": id, "event": contact}, nil)
					v.button("Назад", again)
					return v.screen(html.EscapeString(value)), nil
				})
			}
		}
		v.button("Отзывы", v.listAction("tourReviews", tv, nil, 0))
		v.button("Новости тура", v.listAction("posts", tv, nil, 0))
		if owner := str(object(item["organizer"])["id"]); owner != "" && owner != v.user {
			v.button("Написать организатору", func(v *view) (screens.Screen, error) { return v.openChat("partnerId", owner, again) })
			v.button("Организатор", func(v *view) (screens.Screen, error) { return v.organizer(owner, again, 0) })
		}
	case "group":
		pv := cloneVars(vars)
		pv["participant"] = str(item["id"])
		v.button("Оплаты участника", v.listAction("payments", pv, nil, 0))
		v.button("Анкета участника", func(v *view) (screens.Screen, error) {
			return v.begin(v.e.resources["groupForm"], vars, object(item["form"]), false, again)
		})
	case "reviews":
		v.button("Ответить", func(v *view) (screens.Screen, error) {
			return v.begin(v.e.resources["reply"], vars, record{"text": object(item["reply"])["text"]}, false, again)
		})
		if item["reply"] != nil {
			v.button("Удалить ответ", func(v *view) (screens.Screen, error) {
				return v.confirmDelete("/reviews/"+url.PathEscape(vars["id"])+"/reply", title(item), again)
			})
		}
	case "tourReviews":
		if str(object(item["author"])["id"]) == v.user {
			v.button("Удалить отзыв", func(v *view) (screens.Screen, error) {
				return v.confirmDelete("/reviews/"+url.PathEscape(vars["id"]), title(item), back)
			})
		}
	case "clients":
		if id := str(item["userId"]); id != "" {
			v.button("Написать сообщение", func(v *view) (screens.Screen, error) { return v.openChat("userId", id, again) })
		}
	case "trips":
		v.button("Статус поездки", func(v *view) (screens.Screen, error) {
			return v.begin(v.e.resources["tripStatus"], vars, item, false, again)
		})
		v.button("Оплаты поездки", v.listAction("driverPayments", map[string]string{"trip": vars["id"]}, nil, 0))
	case "drivers":
		if id := str(item["userId"]); id != "" {
			v.button("Написать сообщение", func(v *view) (screens.Screen, error) { return v.openChat("partnerId", id, again) })
		}
	case "chats":
		v.button("Написать сообщение", func(v *view) (screens.Screen, error) {
			return v.begin(v.e.resources["message"], vars, nil, false, again)
		})
		if messages := array(item["messages"]); len(messages) >= 50 {
			oldest := str(messages[0]["createdAt"])
			v.button("Предыдущие сообщения", func(v *view) (screens.Screen, error) {
				older := r
				older.detail = queryPath(pathFor(r.detail, vars), record{"before": oldest})
				return v.card(older, vars, nil, again, 0)
			})
		}
	}
	if len(r.filters) > 0 {
		v.button("🔎 Фильтры", func(v *view) (screens.Screen, error) { return v.filter(r, vars, nil, back) })
	}
	v.button("Обновить", again)
	v.button("Назад к списку", back)
	v.button("Открыть кабинет", func(v *view) (screens.Screen, error) { return v.home() })
	screen := v.screen("<b>" + v.l.T(r.title) + "</b> · " + strconv.Itoa(page+1) + "/" + strconv.Itoa(len(pages)) + "\n\n" + pages[page])
	if page == 0 {
		src := str(item["coverImageUrl"])
		if src == "" {
			if photos, ok := item["photos"].([]any); ok && len(photos) > 0 {
				src = str(photos[0])
			}
		}
		if picture := v.e.Links.Picture(&src); picture != "" {
			screen.Preview = &telego.LinkPreviewOptions{URL: picture, PreferLargeMedia: true, ShowAboveText: true}
		}
	}
	return screen, nil
}
func (v *view) confirmDelete(path, name string, back action) (screens.Screen, error) {
	v.button("Да, удалить", func(v *view) (screens.Screen, error) {
		_, err := v.request("DELETE", path, nil)
		if err != nil {
			v.button("Назад к списку", back)
			return v.errorScreen(err), nil
		}
		return back(v)
	})
	v.button("Отмена", back)
	return v.screen(v.l.T("Удалить запись? Это действие нельзя отменить.") + "\n\n" + html.EscapeString(name)), nil
}
func (v *view) openChat(key, id string, back action) (screens.Screen, error) {
	value, err := v.request("POST", "/chats", record{key: id})
	if err != nil {
		return screens.Screen{}, err
	}
	item := object(value)
	return v.card(v.e.resources["chats"], map[string]string{"id": str(item["id"])}, item, back, 0)
}
func (v *view) organizer(id string, back action, page int) (screens.Screen, error) {
	value, err := v.request("GET", queryPath("/tours/organizers/"+url.PathEscape(id), record{"locale": string(v.l)}), nil)
	if err != nil {
		return screens.Screen{}, err
	}
	pages := screens.Pages(v.paragraphs(object(value), 0), 1800)
	if len(pages) == 0 {
		pages = []string{v.l.T("Пока ничего нет.")}
	}
	page = min(max(page, 0), len(pages)-1)
	if page > 0 {
		v.button("◀️ Назад", func(v *view) (screens.Screen, error) { return v.organizer(id, back, page-1) })
	}
	if page+1 < len(pages) {
		v.button("Дальше ▶️", func(v *view) (screens.Screen, error) { return v.organizer(id, back, page+1) })
	}
	v.button("Назад к списку", back)
	return v.screen(pages[page]), nil
}
