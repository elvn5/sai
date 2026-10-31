package screens

import (
	"strconv"
	"strings"
	"time"

	"github.com/mymmrac/telego"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/i18n"
	"github.com/elvn5/sai/services/bot/internal/keyboards"
)

// The profile's callback data ("p:<action>[:<arg>]"), the same section
// as on the site's profile: the account, the participant's form, the
// password, achievements and partnership.
const (
	ProfileHome     = keyboards.ProfilePrefix + "home"
	ProfileForm     = keyboards.ProfilePrefix + "form"
	ProfilePassword = keyboards.ProfilePrefix + "pw"
	ProfileAwards   = keyboards.ProfilePrefix + "aw"
	ProfilePartner  = keyboards.ProfilePrefix + "pa"
	// Filling in the form: start, an experience level, keep the value
	// asked for, skip an optional one, stop.
	FormEdit   = keyboards.ProfilePrefix + "fe"
	FormLevel  = keyboards.ProfilePrefix + "fx:"
	FormKeep   = keyboards.ProfilePrefix + "fk"
	FormSkip   = keyboards.ProfilePrefix + "fs"
	FormCancel = keyboards.ProfilePrefix + "fc"
	// Changing the password, and stopping.
	PasswordStart  = keyboards.ProfilePrefix + "pc"
	PasswordCancel = keyboards.ProfilePrefix + "px"
	// A partner application: "p:ap:<kind>".
	PartnerApply = keyboards.ProfilePrefix + "ap:"
)

// Experience levels of the participant's form, as the site names them.
var Levels = []string{"none", "hiking", "trekking", "mountaineering"}

func levelTitle(l i18n.Lang, level string) string {
	switch level {
	case "none":
		return l.T("Нет опыта")
	case "hiking":
		return l.T("Однодневные походы")
	case "trekking":
		return l.T("Многодневные треки")
	case "mountaineering":
		return l.T("Альпинизм")
	}
	return level
}

// ProfileView is what the profile shows; nil parts failed to load and
// are left out.
type ProfileView struct {
	Me api.Me
	// The tours signed up for (nil: unknown).
	Joined []api.Tour
	// Whether the list of tours loaded.
	JoinedKnown bool
	Awards      *api.MyAchievements
	// Whether the form loaded, and the form itself (nil: not filled in).
	FormKnown bool
	Form      *api.ParticipantForm
	// The organizer's room is on (a button to it for organizers).
	Organizing bool
}

func orDash(l i18n.Lang, s *string, missing string) string {
	if s == nil || strings.TrimSpace(*s) == "" {
		return l.T(missing)
	}
	return esc(*s)
}

// HasLogin: the account can sign in with a password (email or phone).
func HasLogin(me api.Me) bool {
	return (me.Email != nil && *me.Email != "") || (me.Phone != nil && *me.Phone != "")
}

// IsOrganizer: an approved partner running tours.
func IsOrganizer(me api.Me) bool {
	return me.PartnerStatus == api.PartnerApproved && me.PartnerKind != api.KindDriver
}

func partnerLine(l i18n.Lang, me api.Me) string {
	driver := me.PartnerKind == api.KindDriver
	switch me.PartnerStatus {
	case api.PartnerApproved:
		if driver {
			return l.T("✅ водитель")
		}
		return l.T("✅ организатор туров")
	case api.PartnerPending:
		if driver {
			return l.T("⏳ заявка водителя на рассмотрении")
		}
		return l.T("⏳ заявка организатора на рассмотрении")
	}
	return l.T("нет")
}

// Profile is the profile section's first screen.
func Profile(l i18n.Lang, v ProfileView, now time.Time) Screen {
	me := v.Me
	name := l.T("Без имени")
	if me.FirstName != nil && *me.FirstName != "" {
		name = esc(*me.FirstName)
	}
	head := "👤 <b>" + name + "</b>"
	if me.Username != nil && *me.Username != "" {
		head += " · @" + esc(*me.Username)
	}
	password := l.T("не задан")
	if me.HasPassword {
		password = l.T("задан")
	}
	lines := []string{
		head, "",
		l.F("📞 Телефон: %s", orDash(l, me.Phone, "не указан")),
		l.F("✉️ Email: %s", orDash(l, me.Email, "не указан")),
		l.F("🔑 Пароль: %s", password),
		"",
	}
	if v.JoinedKnown {
		upcoming, past := SplitTours(v.Joined, now)
		lines = append(lines, l.F("🎒 Туров пройдено: %d · впереди: %d", len(past), len(upcoming)))
	}
	if v.Awards != nil {
		lines = append(lines, l.F("🏆 Достижений: %d · очков: %d", len(v.Awards.Awards), v.Awards.Points))
	}
	if v.FormKnown {
		form := l.T("не заполнена")
		if v.Form != nil {
			form = l.T("✅ заполнена")
		}
		lines = append(lines, l.F("📝 Анкета участника: %s", form))
	}
	lines = append(lines, l.F("🤝 Партнёрство: %s", partnerLine(l, me)))

	passwordButton := l.T("🔑 Задать пароль")
	if me.HasPassword {
		passwordButton = l.T("🔑 Сменить пароль")
	}
	kb := [][]telego.InlineKeyboardButton{
		{keyboards.Button(l.T("✏️ Изменить имя"), keyboards.EditName), keyboards.Button(l.T("✏️ Изменить телефон"), keyboards.EditPhone)},
		{keyboards.Button(l.T("📝 Анкета участника"), ProfileForm), keyboards.Button(passwordButton, ProfilePassword)},
		{keyboards.Button(l.T("🏆 Достижения"), ProfileAwards), keyboards.Button(l.T("🤝 Партнёрство"), ProfilePartner)},
		{keyboards.Button(l.T("🎒 Мои туры"), ListData(FromMine)), keyboards.Button(l.T("❤️ Избранное"), TourData(TabFavorites, 0))},
	}
	if v.Organizing && IsOrganizer(me) {
		kb = append(kb, []telego.InlineKeyboardButton{keyboards.Button(l.T("🧑‍✈️ Кабинет организатора"), OrgHome)})
	}
	kb = append(kb, []telego.InlineKeyboardButton{keyboards.Button("🌐 "+l.T("Язык")+" · "+l.Name(), keyboards.LanguageMenu)})
	return Screen{Text: strings.Join(lines, "\n"), Preview: noPreview(), Markup: rows(kb...)}
}

func backToProfile(l i18n.Lang) []telego.InlineKeyboardButton {
	return []telego.InlineKeyboardButton{keyboards.Button(l.T("◀️ Профиль"), ProfileHome)}
}

// FormScreen shows the participant's form (nil: not filled in); note is
// an optional first line ("saved").
func FormScreen(l i18n.Lang, f *api.ParticipantForm, note string) Screen {
	var b strings.Builder
	if note != "" {
		b.WriteString(note + "\n\n")
	}
	b.WriteString(l.T("📝 <b>Анкета участника</b>") + "\n\n")
	b.WriteString(l.T("Заполните один раз — анкету увидят организаторы туров, на которые вы записываетесь: так они подготовятся и будут знать, кому звонить в экстренном случае.") + "\n\n")
	button := l.T("📝 Заполнить анкету")
	if f == nil {
		b.WriteString(l.T("Анкета пока не заполнена. Это необязательно, но организатору важно знать это до выезда."))
	} else {
		button = l.T("✏️ Изменить анкету")
		b.WriteString(l.F("🆘 <b>Экстренный контакт:</b> %s, %s", esc(f.EmergencyName), esc(f.EmergencyPhone)) + "\n")
		b.WriteString(l.F("🧗 <b>Опыт в горах:</b> %s", levelTitle(l, f.Experience)))
		if f.Health != nil && *f.Health != "" {
			b.WriteString("\n" + l.F("❤️ <b>Здоровье:</b> %s", esc(*f.Health)))
		}
		if f.Notes != nil && *f.Notes != "" {
			b.WriteString("\n" + l.F("💬 <b>Что ещё важно знать организатору:</b> %s", esc(*f.Notes)))
		}
	}
	return Screen{Text: b.String(), Preview: noPreview(), Markup: rows(
		[]telego.InlineKeyboardButton{keyboards.Button(button, FormEdit)},
		backToProfile(l),
	)}
}

// Steps of filling in the form, in order.
const (
	StepName       = "name"
	StepPhone      = "phone"
	StepExperience = "experience"
	StepHealth     = "health"
	StepNotes      = "notes"
)

var formSteps = []string{StepName, StepPhone, StepExperience, StepHealth, StepNotes}

// FormStep asks for one answer of the form; current is the answer kept
// so far ("" for none), and an optional problem with the last answer
// comes first.
func FormStep(l i18n.Lang, step, current, problem string) Screen {
	n := 1
	for i, s := range formSteps {
		if s == step {
			n = i + 1
		}
	}
	var b strings.Builder
	if problem != "" {
		b.WriteString(problem + "\n\n")
	}
	b.WriteString(l.F("📝 <b>Анкета участника</b> · %d из %d", n, len(formSteps)) + "\n\n")
	var kb [][]telego.InlineKeyboardButton
	keep := func() {
		if current != "" {
			kb = append(kb, []telego.InlineKeyboardButton{keyboards.Button(l.F("➡️ Оставить: %s", clip(current, 40)), FormKeep)})
		}
	}
	switch step {
	case StepName:
		b.WriteString(l.T("Кому позвонить в экстренном случае? Напишите имя."))
		keep()
	case StepPhone:
		b.WriteString(l.T("Телефон этого человека, например +996 555 123 456."))
		keep()
	case StepExperience:
		b.WriteString(l.T("Какой у вас опыт в горах?"))
		for _, level := range Levels {
			label := levelTitle(l, level)
			if level == current {
				label = "✅ " + label
			}
			kb = append(kb, []telego.InlineKeyboardButton{keyboards.Button(label, FormLevel+level)})
		}
	case StepHealth:
		b.WriteString(l.T("Что о здоровье важно знать организатору — аллергии, хронические болезни, лекарства? Напишите или пропустите."))
		keep()
		kb = append(kb, []telego.InlineKeyboardButton{keyboards.Button(l.T("⏭ Пропустить"), FormSkip)})
	case StepNotes:
		b.WriteString(l.T("Что ещё важно знать организатору? Напишите или пропустите."))
		keep()
		kb = append(kb, []telego.InlineKeyboardButton{keyboards.Button(l.T("⏭ Пропустить"), FormSkip)})
	}
	kb = append(kb, []telego.InlineKeyboardButton{keyboards.Button(l.T("Отмена"), FormCancel)})
	return Screen{Text: b.String(), Preview: noPreview(), Markup: rows(kb...)}
}

// PasswordScreen tells about the password and offers to change or set it.
func PasswordScreen(l i18n.Lang, me api.Me, note string) Screen {
	var b strings.Builder
	if note != "" {
		b.WriteString(note + "\n\n")
	}
	b.WriteString(l.T("🔑 <b>Пароль</b>") + "\n\n")
	switch {
	case !me.HasPassword && !HasLogin(me):
		b.WriteString(l.T("Вы входите через Telegram. Чтобы входить на сайт по паролю, к аккаунту нужен email или телефон — добавьте телефон, и пароль можно будет задать."))
		return Screen{Text: b.String(), Preview: noPreview(), Markup: rows(
			[]telego.InlineKeyboardButton{keyboards.Button(l.T("✏️ Изменить телефон"), keyboards.EditPhone)},
			backToProfile(l),
		)}
	case me.HasPassword:
		b.WriteString(l.T("Пароль задан — им можно входить на сайт по email или телефону. После смены пароля выйдем из аккаунта на всех других устройствах."))
	default:
		b.WriteString(l.T("Задайте пароль, чтобы входить на сайт по email или телефону."))
	}
	button := l.T("🔑 Задать пароль")
	if me.HasPassword {
		button = l.T("🔑 Сменить пароль")
	}
	return Screen{Text: b.String(), Preview: noPreview(), Markup: rows(
		[]telego.InlineKeyboardButton{keyboards.Button(button, PasswordStart)},
		backToProfile(l),
	)}
}

// Steps of changing the password.
const (
	StepCurrent = "current"
	StepNew     = "new"
	StepConfirm = "confirm"
)

// PasswordStep asks for a password; problem says what was wrong with the
// last one.
func PasswordStep(l i18n.Lang, step, problem string) Screen {
	var b strings.Builder
	if problem != "" {
		b.WriteString(problem + "\n\n")
	}
	b.WriteString(l.T("🔑 <b>Пароль</b>") + "\n\n")
	switch step {
	case StepCurrent:
		b.WriteString(l.T("Введите текущий пароль."))
	case StepNew:
		b.WriteString(l.T("Введите новый пароль — не короче 8 символов."))
	default:
		b.WriteString(l.T("Повторите новый пароль."))
	}
	b.WriteString("\n\n" + l.T("🔒 Сообщение с паролем я сразу удалю из чата."))
	return Screen{Text: b.String(), Preview: noPreview(), Markup: rows(
		[]telego.InlineKeyboardButton{keyboards.Button(l.T("Отмена"), PasswordCancel)},
	)}
}

// Leaders shown under one's own achievements.
const leadersShown = 3

func achievementTitle(l i18n.Lang, a api.Achievement) string {
	for _, lang := range []string{string(l), string(i18n.RU)} {
		if t := a.Title[lang]; t != "" {
			return t
		}
	}
	return a.ID
}

// Awards is the user's hall of fame: points, medals, their code and the
// leaders.
func Awards(l i18n.Lang, mine api.MyAchievements, leaders []api.HallEntry, userID string) Screen {
	rank := "—"
	for _, e := range leaders {
		if e.User.ID == userID {
			rank = "#" + strconv.Itoa(e.Rank)
		}
	}
	var b strings.Builder
	b.WriteString(l.T("🏆 <b>Достижения</b>") + "\n\n")
	b.WriteString(l.F("⭐ Очков: %d · достижений: %d · место: %s", mine.Points, len(mine.Awards), rank) + "\n\n")
	if len(mine.Awards) == 0 {
		b.WriteString(l.T("Пока нет достижений — впереди много маршрутов!"))
	}
	for i, a := range mine.Awards {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("🏅 " + esc(achievementTitle(l, a.Achievement)) + " · " +
			strconv.Itoa(a.Achievement.Points) + " " + l.N(a.Achievement.Points, "очко", "очка", "очков"))
	}
	b.WriteString("\n\n" + l.F("🎟 Ваш код путешественника: <code>%s</code>", esc(mine.Code)) + "\n")
	b.WriteString(l.T("Покажите код партнёру, чтобы получить достижение. Партнёры вручают их за пройденные маршруты, поездки и вклад в сообщество."))
	if len(leaders) > 0 {
		b.WriteString("\n\n" + l.T("👑 <b>Лидеры</b>"))
		medals := []string{"🥇", "🥈", "🥉"}
		for i, e := range leaders[:min(leadersShown, len(leaders))] {
			b.WriteString("\n" + medals[i] + " " + esc(e.User.Name()) + " · " + strconv.Itoa(e.Points))
		}
	}
	return Screen{Text: b.String(), Preview: noPreview(), Markup: rows(backToProfile(l))}
}

// Partner is the partnership: the status, or the offer to apply as an
// organizer or a driver.
func Partner(l i18n.Lang, me api.Me, organizing bool, note string) Screen {
	var b strings.Builder
	if note != "" {
		b.WriteString(note + "\n\n")
	}
	b.WriteString(l.T("🤝 <b>Партнёрство</b>") + "\n\n")
	driver := me.PartnerKind == api.KindDriver
	switch me.PartnerStatus {
	case api.PartnerApproved:
		if driver {
			b.WriteString(l.T("✅ Вы водитель-партнёр Ermekten: организаторы туров находят вас в каталоге и присылают заявки на перевозку групп.") + "\n\n" +
				l.T("Заявки, рейсы и транспорт — в кабинете на сайте, в «🏔 Открыть Ermekten»."))
			return Screen{Text: b.String(), Preview: noPreview(), Markup: rows(backToProfile(l))}
		}
		b.WriteString(l.T("✅ Вы организатор туров Ermekten: публикуйте туры, ведите группы и вручайте путешественникам достижения.") + "\n\n" +
			l.T("Создать тур можно на сайте, в «🏔 Открыть Ermekten»."))
		kb := [][]telego.InlineKeyboardButton{}
		if organizing {
			kb = append(kb, []telego.InlineKeyboardButton{keyboards.Button(l.T("🧑‍✈️ Кабинет организатора"), OrgHome)})
		}
		return Screen{Text: b.String(), Preview: noPreview(), Markup: rows(append(kb, backToProfile(l))...)}
	case api.PartnerPending:
		kind := l.T("организатора туров")
		if driver {
			kind = l.T("водителя")
		}
		b.WriteString(l.F("⏳ Ваша заявка %s на рассмотрении у администратора.", kind))
		return Screen{Text: b.String(), Preview: noPreview(), Markup: rows(backToProfile(l))}
	}
	if me.PartnerStatus == api.PartnerRejected {
		b.WriteString(l.T("❌ Заявка была отклонена. Вы можете подать её снова.") + "\n\n")
	}
	b.WriteString(l.T("Станьте партнёром Ermekten. Кем вы хотите стать?") + "\n\n" +
		l.T("🧭 <b>Организатор туров</b> — публикуйте туры, ведите группы, клиентов и оплаты.") + "\n" +
		l.T("🚐 <b>Водитель</b> — возите туристические группы: заявки от организаторов, рейсы, транспорт и доходы."))
	return Screen{Text: b.String(), Preview: noPreview(), Markup: rows(
		[]telego.InlineKeyboardButton{
			keyboards.Button(l.T("🧭 Организатор туров"), PartnerApply+api.KindOrganizer),
			keyboards.Button(l.T("🚐 Водитель"), PartnerApply+api.KindDriver),
		},
		backToProfile(l),
	)}
}
