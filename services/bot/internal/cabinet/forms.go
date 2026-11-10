package cabinet

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/screens"
)

func fieldsFor(r resource, editing bool) []field {
	out := append([]field{}, r.fields...)
	if r.key == "group" {
		if editing {
			out = out[1:]
		} else {
			out = out[:2]
			out[1].options = []string{"new", "confirmed", "waitlist"}
		}
	}
	for i := range out {
		if out[i].key == "content" || out[i].key == "startsAt" || out[i].key == "audience" {
			out[i].required = true
		}
		if out[i].key == "text" && r.key == "tourReviews" {
			out[i].min = 10
		}
	}
	return out
}
func (v *view) begin(r resource, vars map[string]string, item record, editing bool, back action) (screens.Screen, error) {
	if err := v.authorize(r); err != nil {
		return screens.Screen{}, err
	}
	r.fields = fieldsFor(r, editing)
	values := record{}
	names := map[string]string{}
	for _, f := range r.fields {
		if value, ok := item[f.key]; ok && value != nil {
			values[f.key] = value
		}
		if f.kind == "ref" {
			if nested := object(item[strings.TrimSuffix(f.key, "Id")]); nested["id"] != nil {
				values[f.key] = nested["id"]
				names[f.key] = title(nested)
			}
		}
	}
	path := r.create
	if editing {
		path = r.update
	}
	method := "POST"
	if m, p, ok := strings.Cut(path, " "); ok {
		method, path = m, p
	}
	if r.key == "tours" && !editing {
		values["slug"] = "tour-" + uuid.NewString()
		if values["locale"] == nil {
			values["locale"] = string(v.l)
		}
		if values["cancellationPolicy"] == nil {
			values["cancellationPolicy"] = "flexible"
		}
	}
	if r.key == "posts" || r.key == "tourReviews" {
		values["targetType"] = "tour"
		values["targetId"] = vars["tour"]
	}
	if r.key == "vehicles" && !editing {
		values["active"] = true
	}
	if r.key == "group" && editing {
		values["arrived"] = item["arrivedAt"] != nil
	}
	f := &form{resource: r, vars: cloneVars(vars), values: values, names: names, method: method, path: pathFor(path, vars), back: back}
	return v.edit(f, 0)
}
func (v *view) filter(r resource, vars map[string]string, query record, back action) (screens.Screen, error) {
	values := record{}
	for k, val := range query {
		values[k] = val
	}
	r.fields = r.filters
	f := &form{resource: r, vars: cloneVars(vars), values: values, filter: true, back: back}
	return v.edit(f, 0)
}
func (v *view) edit(f *form, page int) (screens.Screen, error) {
	v.s.form = f
	v.s.pending = nil
	fields := f.resource.fields
	start := min(max(0, page)*8, len(fields))
	end := min(start+8, len(fields))
	var lines []string
	for _, field := range fields[start:end] {
		value := f.values[field.key]
		marker := ""
		if field.required && empty(value) {
			marker = " *"
		}
		label := v.l.T(field.label) + marker
		lines = append(lines, "<b>"+html.EscapeString(label)+"</b>: "+html.EscapeString(short(v.formValue(f, field, value), 130)))
		v.button(label, func(v *view) (screens.Screen, error) { v.s.form = f; return v.prompt(field, 0) })
	}
	if start > 0 {
		v.button("◀️ Назад", func(v *view) (screens.Screen, error) { return v.edit(f, page-1) })
	}
	if end < len(fields) {
		v.button("Дальше ▶️", func(v *view) (screens.Screen, error) { return v.edit(f, page+1) })
	}
	v.button("Проверить и сохранить", func(v *view) (screens.Screen, error) { return v.review(f, 0) })
	v.button("Отмена", func(v *view) (screens.Screen, error) { v.s.form = nil; v.s.pending = nil; return f.back(v) })
	return v.screen("<b>" + v.l.T(f.resource.title) + "</b>\n" + v.l.T("Выберите поле. Звёздочкой отмечены обязательные.") + "\n\n" + strings.Join(lines, "\n\n")), nil
}
func (v *view) formValue(form *form, f field, value any) string {
	if name := form.names[f.key]; name != "" {
		return name
	}
	return v.previewValue(f, value)
}
func empty(a any) bool { return a == nil || str(a) == "" }
func (v *view) previewValue(f field, value any) string {
	if value == nil {
		return "—"
	}
	if f.kind == "html" {
		return strings.Join(screens.Paragraphs(str(value)), " ")
	}
	if f.kind == "ref" {
		return v.l.T("Выбрано")
	}
	if f.kind == "bool" {
		if value == true {
			return v.l.T("Да")
		}
		return v.l.T("Нет")
	}
	return v.option(str(value))
}
func (v *view) prompt(f field, page int) (screens.Screen, error) {
	form := v.s.form
	if form == nil {
		return v.home()
	}
	v.s.pending = &f
	back := func(v *view) (screens.Screen, error) { return v.edit(form, 0) }
	set := func(value any) action {
		return func(v *view) (screens.Screen, error) {
			form.values[f.key] = value
			v.s.pending = nil
			return v.edit(form, 0)
		}
	}
	if !f.required {
		v.button("Очистить поле", func(v *view) (screens.Screen, error) {
			if f.kind == "photos" || f.kind == "tags" || f.kind == "program" {
				form.values[f.key] = []any{}
			} else {
				form.values[f.key] = nil
				delete(form.names, f.key)
			}
			return v.edit(form, 0)
		})
	}
	if !f.append && !empty(form.values[f.key]) && (f.kind == "html" || f.kind == "program" || (f.kind == "text" && f.max >= 2000)) {
		v.button("Дополнить текст или программу", func(v *view) (screens.Screen, error) { next := f; next.append = true; return v.prompt(next, 0) })
	}
	prompt := v.l.T("Отправьте значение следующим сообщением.")
	switch f.kind {
	case "choice":
		v.s.pending = nil
		for _, value := range f.options {
			v.button(v.option(value), set(value))
		}
		prompt = v.l.T("Выберите значение кнопкой.")
	case "bool":
		v.s.pending = nil
		v.button("Да", set(true))
		v.button("Нет", set(false))
		prompt = v.l.T("Выберите значение кнопкой.")
	case "ref":
		v.s.pending = nil
		r := v.e.resources[f.ref]
		if err := v.authorize(r); err != nil {
			return screens.Screen{}, err
		}
		items, total, _, err := v.load(r, nil, nil, page)
		if err != nil {
			return screens.Screen{}, err
		}
		for _, item := range items {
			id := str(item["id"])
			if f.ref == "drivers" {
				id = str(item["userId"])
			}
			v.rawButton(title(item), func(v *view) (screens.Screen, error) {
				if form.names == nil {
					form.names = map[string]string{}
				}
				form.names[f.key] = title(item)
				return set(id)(v)
			})
		}
		if page > 0 {
			v.button("◀️ Назад", func(v *view) (screens.Screen, error) { return v.prompt(f, page-1) })
		}
		if (page+1)*pageSize < total {
			v.button("Дальше ▶️", func(v *view) (screens.Screen, error) { return v.prompt(f, page+1) })
		}
		prompt = v.l.T("Выберите запись из списка. Если список пуст, сначала добавьте запись в соответствующем разделе.")
	case "audience":
		v.button("Все клиенты", set("all"))
		v.button("Группа тура", func(v *view) (screens.Screen, error) { return v.audienceTours(form, f, 0) })
		prompt = v.l.T("Выберите всех клиентов или группу тура. Для рассылки по метке отправьте название метки текстом.")
	case "date":
		prompt = v.l.T("Введите дату в формате ГГГГ-ММ-ДД.")
	case "time":
		prompt = v.l.T("Введите время ЧЧ:ММ (Бишкек).")
	case "datetime":
		prompt = v.l.T("Введите дату и время: ГГГГ-ММ-ДД ЧЧ:ММ (Бишкек).")
	case "int":
		prompt = v.l.F("Введите целое число от %d до %d.", f.min, f.max)
	case "photos", "photo":
		prompt = v.l.T("Отправьте фотографию или ссылку https://. Фотографии добавляются по одной; очистка удалит весь список.")
	case "program":
		prompt = v.l.T("Каждый день: заголовок в первой строке, описание ниже. Между днями оставьте пустую строку.")
	case "tags":
		prompt = v.l.T("Введите метки через запятую.")
	}
	if f.key == "body" && form.resource.key != "templates" {
		v.button("Шаблоны сообщений", func(v *view) (screens.Screen, error) { return v.templatePicker(form, f, 0) })
	}
	v.button("Назад", back)
	return v.screen("<b>" + v.l.T(f.label) + "</b>\n\n" + prompt), nil
}
func (v *view) audienceTours(form *form, f field, page int) (screens.Screen, error) {
	v.s.pending = nil
	items, total, _, err := v.load(v.e.resources["tours"], nil, nil, page)
	if err != nil {
		return screens.Screen{}, err
	}
	for _, item := range items {
		v.rawButton(title(item), func(v *view) (screens.Screen, error) {
			form.values[f.key] = "tour:" + str(item["id"])
			if form.names == nil {
				form.names = map[string]string{}
			}
			form.names[f.key] = title(item)
			return v.edit(form, 0)
		})
	}
	if page > 0 {
		v.button("◀️ Назад", func(v *view) (screens.Screen, error) { return v.audienceTours(form, f, page-1) })
	}
	if (page+1)*pageSize < total {
		v.button("Дальше ▶️", func(v *view) (screens.Screen, error) { return v.audienceTours(form, f, page+1) })
	}
	v.button("Назад", func(v *view) (screens.Screen, error) { return v.prompt(f, 0) })
	return v.screen(v.l.T("Получатели")), nil
}
func (v *view) templatePicker(form *form, f field, page int) (screens.Screen, error) {
	v.s.pending = nil
	items, total, _, err := v.load(v.e.resources["templates"], nil, nil, page)
	if err != nil {
		return screens.Screen{}, err
	}
	for _, item := range items {
		v.rawButton(title(item), func(v *view) (screens.Screen, error) { form.values[f.key] = item["body"]; return v.edit(form, 0) })
	}
	if page > 0 {
		v.button("◀️ Назад", func(v *view) (screens.Screen, error) { return v.templatePicker(form, f, page-1) })
	}
	if (page+1)*pageSize < total {
		v.button("Дальше ▶️", func(v *view) (screens.Screen, error) { return v.templatePicker(form, f, page+1) })
	}
	v.button("Назад", func(v *view) (screens.Screen, error) { return v.prompt(f, 0) })
	return v.screen(v.l.T("Шаблоны сообщений")), nil
}
func validateForm(f *form) error {
	for _, field := range f.resource.fields {
		value := f.values[field.key]
		if field.required && empty(value) {
			return fmt.Errorf("%s", field.key)
		}
	}
	for _, pair := range [][2]string{{"startDate", "endDate"}, {"from", "to"}, {"startsAt", "endsAt"}} {
		a, b := str(f.values[pair[0]]), str(f.values[pair[1]])
		if a != "" && b != "" && b < a {
			return fmt.Errorf("%s", pair[1])
		}
	}
	if f.resource.key == "posts" && empty(f.values["text"]) && len(array(f.values["photos"])) == 0 {
		if p, ok := f.values["photos"].([]any); !ok || len(p) == 0 {
			return fmt.Errorf("text")
		}
	}
	if f.resource.key == "leads" && f.values["stage"] == "lost" && empty(f.values["lostReason"]) {
		return fmt.Errorf("lostReason")
	}
	return nil
}
func (v *view) review(f *form, page int) (screens.Screen, error) {
	if err := validateForm(f); err != nil {
		screen, _ := v.edit(f, 0)
		screen.Text = v.l.T("Заполните обязательные поля и проверьте порядок дат.") + "\n\n" + screen.Text
		return screen, nil
	}
	if f.filter {
		return v.list(v.e.resources[f.resource.key], f.vars, f.values, 0)
	}
	display := record{}
	for k, value := range f.values {
		display[k] = value
	}
	for _, field := range f.resource.fields {
		if field.kind == "ref" && !empty(f.values[field.key]) {
			display[field.key] = v.formValue(f, field, f.values[field.key])
		}
	}
	if name := f.names["audience"]; name != "" {
		display["audience"] = name
	}
	if f.resource.key == "awards" {
		recipient, err := v.request("GET", "/achievements/recipients/"+url.PathEscape(str(f.values["code"])), nil)
		if err != nil {
			v.button("Вернуться к черновику", func(v *view) (screens.Screen, error) { return v.edit(f, 0) })
			return v.errorScreen(err), nil
		}
		display["recipient"] = recipient
	}
	pages := screens.Pages(v.paragraphs(display, 0), 1800)
	if len(pages) == 0 {
		pages = []string{v.l.T("Сохранить изменения?")}
	}
	page = min(max(page, 0), len(pages)-1)
	if page > 0 {
		v.button("◀️ Назад", func(v *view) (screens.Screen, error) { return v.review(f, page-1) })
	}
	if page+1 < len(pages) {
		v.button("Дальше ▶️", func(v *view) (screens.Screen, error) { return v.review(f, page+1) })
	}
	v.button("Подтвердить", func(v *view) (screens.Screen, error) {
		if err := v.authorize(f.resource); err != nil {
			return screens.Screen{}, err
		}
		body := any(f.values)
		if f.resource.key == "groupForm" || f.resource.key == "myForm" {
			body = record{"form": f.values}
		}
		value, err := v.request(f.method, f.path, body)
		if err != nil {
			v.button("Вернуться к черновику", func(v *view) (screens.Screen, error) { return v.edit(f, 0) })
			v.button("Проверить список", f.back)
			return v.errorScreen(err), nil
		}
		v.s.form = nil
		v.s.pending = nil
		v.button("Назад к списку", f.back)
		v.button("Открыть кабинет", func(v *view) (screens.Screen, error) { return v.home() })
		text := v.l.T("Сохранено.")
		if f.resource.key == "broadcasts" {
			text += "\n\n" + v.describe(object(value))
		}
		return v.screen(text), nil
	})
	v.button("Изменить", func(v *view) (screens.Screen, error) { return v.edit(f, 0) })
	v.button("Отмена", f.back)
	text := v.l.T("Проверьте данные перед подтверждением.")
	if f.resource.key == "broadcasts" || f.resource.key == "message" || f.resource.key == "transport" {
		text = v.l.T("После подтверждения сообщение или заявка будут отправлены получателю.")
	}
	return v.screen(text + "\n\n" + pages[page]), nil
}
func (v *view) errorScreen(err error) screens.Screen {
	text := v.l.T("Не удалось выполнить действие. Проверьте список перед повтором: запрос мог сохраниться.")
	if api.StatusOf(err) >= 400 && api.StatusOf(err) < 500 {
		text = v.l.T("Действие отклонено. Проверьте права и введённые данные.")
	}
	if fields := api.FieldsOf(err); len(fields) > 0 {
		keys := []string{}
		for key := range fields {
			if label := v.label(key); label != "" {
				keys = append(keys, label)
			}
		}
		sort.Strings(keys)
		text += "\n" + html.EscapeString(strings.Join(keys, ", "))
	}
	return v.screen(text)
}

var phoneRE = regexp.MustCompile(`^\+?[0-9\s\-()]{5,32}$`)
var telegramRE = regexp.MustCompile(`^@?[a-zA-Z0-9_]{5,32}$`)

func parseField(f field, text string) (any, error) {
	text = strings.TrimSpace(text)
	bad := fmt.Errorf("invalid %s", f.key)
	if text == "" {
		return nil, bad
	}
	switch f.kind {
	case "int":
		n, err := strconv.Atoi(text)
		if err != nil || n < f.min || n > f.max {
			return nil, bad
		}
		return n, nil
	case "date":
		if _, err := time.Parse(time.DateOnly, text); err != nil || len(text) != 10 {
			return nil, bad
		}
	case "time":
		if _, err := time.Parse("15:04", text); err != nil || len(text) != 5 {
			return nil, bad
		}
	case "datetime":
		t, err := time.ParseInLocation("2006-01-02 15:04", text, time.FixedZone("Bishkek", 6*3600))
		if err != nil {
			return nil, bad
		}
		return t.Format(time.RFC3339), nil
	case "choice":
		if !slices.Contains(f.options, text) {
			return nil, bad
		}
	case "bool", "ref":
		return nil, bad // only buttons may supply these
	case "tags":
		out := []any{}
		for _, tag := range strings.Split(text, ",") {
			tag = strings.TrimSpace(tag)
			if tag == "" || utf8.RuneCountInString(tag) > 40 {
				return nil, bad
			}
			out = append(out, tag)
		}
		if len(out) > f.max {
			return nil, bad
		}
		return out, nil
	case "audience":
		if utf8.RuneCountInString(text) > 40 {
			return nil, bad
		}
		return "tag:" + text, nil
	case "program":
		out := []any{}
		for _, block := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n\n") {
			heading, body, _ := strings.Cut(strings.TrimSpace(block), "\n")
			if heading == "" || utf8.RuneCountInString(heading) > 120 || utf8.RuneCountInString(body) > 2000 {
				return nil, bad
			}
			out = append(out, record{"title": heading, "text": body})
		}
		if len(out) > f.max {
			return nil, bad
		}
		return out, nil
	case "photos", "photo":
		out := []any{}
		for _, line := range strings.Fields(text) {
			u, err := url.Parse(line)
			if err != nil || ((u.Scheme != "https" || u.Host == "") && !strings.HasPrefix(line, "/uploads/")) || len(line) > 2000 {
				return nil, bad
			}
			out = append(out, line)
		}
		if len(out) > f.max {
			return nil, bad
		}
		if f.kind == "photo" {
			return out[0], nil
		}
		return out, nil
	case "html":
		if utf8.RuneCountInString(text) > f.max {
			return nil, bad
		}
		return "<p>" + strings.ReplaceAll(html.EscapeString(text), "\n", "<br>") + "</p>", nil
	default:
		if utf8.RuneCountInString(text) > f.max || utf8.RuneCountInString(text) < f.min {
			return nil, bad
		}
	}
	if (f.key == "phone" || f.key == "whatsapp" || f.key == "emergencyPhone") && !phoneRE.MatchString(text) {
		return nil, bad
	}
	if f.key == "telegram" && !telegramRE.MatchString(text) {
		return nil, bad
	}
	return text, nil
}
