package cabinet

var optionLabels = map[string]string{
	"upcoming": "Предстоящие", "current": "Сейчас в пути", "past": "Прошедшие",
	"soonest": "Сначала ближайшие", "price_asc": "Сначала дешевле", "price_desc": "Сначала дороже", "rating": "По рейтингу",
	"ru": "Русский", "ky": "Кыргызча", "en": "English", "de": "Deutsch",
	"issyk_kul": "Иссык-Куль", "karakol": "Каракол", "bishkek": "Бишкек", "ala_archa": "Ала-Арча", "song_kol": "Сон-Куль", "naryn": "Нарын", "tash_rabat": "Таш-Рабат", "kel_suu": "Кель-Суу", "osh": "Ош", "arslanbob": "Арсланбоб", "sary_chelek": "Сары-Челек", "talas": "Талас", "batken": "Баткен", "chuy": "Чуй",
	"free": "Бесплатная отмена", "flexible": "Гибкие условия", "strict": "Строгие условия", "non_refundable": "Без возврата",
	"none": "Нет опыта", "hiking": "Походы выходного дня", "trekking": "Многодневные походы", "mountaineering": "Альпинизм",
	"cash": "Наличные", "transfer": "Перевод", "card": "Карта", "other": "Другое", "payment": "Оплата", "refund": "Возврат",
	"new": "Новая", "working": "В работе", "offer": "Предложение", "won": "Успешно", "lost": "Отказ", "confirmed": "Подтверждён", "waitlist": "Лист ожидания", "cancelled": "Отменено", "requested": "Запрошено", "accepted": "Принято", "declined": "Отклонено", "done": "Выполнено", "pending": "Ожидает оплаты", "paid": "Оплачено", "failed": "Ошибка", "expired": "Истекло", "refunded": "Возвращено",
	"transport": "Транспорт", "guides": "Гиды", "food": "Питание", "accommodation": "Проживание", "permits": "Разрешения", "gear": "Снаряжение", "guide": "Гид", "assistant": "Помощник", "driver": "Водитель", "cook": "Повар",
	"fuel": "Топливо", "repair": "Ремонт", "maintenance": "Обслуживание", "wash": "Мойка", "parking": "Парковка", "fines": "Штрафы", "insurance": "Страховка", "inspection": "Техосмотр",
	"sedan": "Седан", "minivan": "Минивэн", "minibus": "Микроавтобус", "bus": "Автобус", "offroad": "Внедорожник",
	"ermekten": "Ermekten", "instagram": "Instagram", "whatsapp": "WhatsApp", "telegram": "Telegram", "phone": "Телефон", "referral": "Рекомендация", "all": "Все клиенты",
}

func (v *view) option(value string) string {
	if label, ok := optionLabels[value]; ok {
		return v.l.T(label)
	}
	return value
}
