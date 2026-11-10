package cabinet

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mymmrac/telego"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/i18n"
	"github.com/elvn5/sai/services/bot/internal/keyboards"
	"github.com/elvn5/sai/services/bot/internal/screens"
)

type API interface {
	CabinetRequest(context.Context, string, string, string, any, any) error
}
type record = map[string]any
type action func(*view) (screens.Screen, error)
type state struct {
	mu      sync.Mutex
	user    string
	touched time.Time
	actions map[string]action
	form    *form
	pending *field
}
type form struct {
	resource     resource
	vars         map[string]string
	values       record
	names        map[string]string
	method, path string
	filter       bool
	back         action
}
type Engine struct {
	API       API
	Links     screens.Links
	mu        sync.Mutex
	states    map[int64]*state
	resources map[string]resource
	Now       func() time.Time
}
type view struct {
	e    *Engine
	s    *state
	ctx  context.Context
	user string
	l    i18n.Lang
	rows [][]telego.InlineKeyboardButton
}

const lifetime = 30 * time.Minute
const pageSize = 6

func New(client API, links screens.Links) *Engine {
	return &Engine{API: client, Links: links, states: map[int64]*state{}, resources: registry(), Now: time.Now}
}
func (e *Engine) state(chat int64) *state {
	e.mu.Lock()
	defer e.mu.Unlock()
	if s := e.states[chat]; s != nil {
		return s
	}
	// Bound abandoned conversations without retaining drafts indefinitely.
	if len(e.states) > 0 {
		for id, s := range e.states {
			if s.mu.TryLock() {
				if !s.touched.IsZero() && e.Now().Sub(s.touched) > lifetime {
					delete(e.states, id)
				}
				s.mu.Unlock()
			}
		}
	}
	s := &state{actions: map[string]action{}}
	e.states[chat] = s
	return s
}
func (e *Engine) Cancel(chat int64) {
	s := e.state(chat)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending = nil
	s.form = nil
	s.actions = map[string]action{}
}

// Handle accepts public entry points or an opaque action token. User identity
// is obtained from Telegram by the bridge, never from callback contents.
func (e *Engine) Handle(ctx context.Context, user string, chat int64, l i18n.Lang, data string) (screens.Screen, error) {
	s := e.state(chat)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.user != user || e.Now().Sub(s.touched) > lifetime {
		s.actions = map[string]action{}
		s.form = nil
		s.pending = nil
	}
	s.user = user
	s.touched = e.Now()
	v := &view{e: e, s: s, ctx: ctx, user: user, l: l}
	var a action
	switch {
	case data == "home":
		a = func(v *view) (screens.Screen, error) { return v.home() }
	case data == "catalog":
		a = v.listAction("catalog", nil, nil, 0)
	case data == "chats":
		a = v.listAction("chats", nil, nil, 0)
	case strings.HasPrefix(data, "tour:"):
		id := strings.TrimPrefix(data, "tour:")
		if _, err := uuid.Parse(id); err == nil {
			a = func(v *view) (screens.Screen, error) {
				return v.card(v.e.resources["catalog"], map[string]string{"id": id}, record{"id": id}, v.listAction("catalog", nil, nil, 0), 0)
			}
		}
	case strings.HasPrefix(data, "trip:"):
		id := strings.TrimPrefix(data, "trip:")
		if _, err := uuid.Parse(id); err == nil {
			a = func(v *view) (screens.Screen, error) {
				return v.card(v.e.resources["trips"], map[string]string{"id": id}, record{"id": id}, v.listAction("trips", nil, nil, 0), 0)
			}
		}
	case strings.HasPrefix(data, "group:"):
		id := strings.TrimPrefix(data, "group:")
		if _, err := uuid.Parse(id); err == nil {
			a = v.listAction("group", map[string]string{"tour": id}, nil, 0)
		}
	case strings.HasPrefix(data, "chat:"):
		id := strings.TrimPrefix(data, "chat:")
		if _, err := uuid.Parse(id); err == nil {
			a = func(v *view) (screens.Screen, error) {
				return v.card(v.e.resources["chats"], map[string]string{"id": id}, record{"id": id}, v.listAction("chats", nil, nil, 0), 0)
			}
		}
	default:
		a = s.actions[data]
	}
	if a == nil {
		v.button("Открыть кабинет", func(v *view) (screens.Screen, error) { return v.home() })
		return v.screen(l.T("Кнопка устарела. Откройте раздел заново.")), nil
	}
	// Invalidate all buttons from the previous screen before doing anything.
	// Together with the per-chat lock this prevents double submissions.
	s.actions = map[string]action{}
	s.pending = nil
	screen, err := a(v)
	if err != nil {
		v.button("Открыть кабинет", func(v *view) (screens.Screen, error) { return v.home() })
		return v.errorScreen(err), nil
	}
	return screen, nil
}
func (e *Engine) Text(ctx context.Context, user string, chat int64, l i18n.Lang, text string) (screens.Screen, bool, error) {
	s := e.state(chat)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.user != user || s.pending == nil || s.form == nil {
		return screens.Screen{}, false, nil
	}
	v := &view{e: e, s: s, ctx: ctx, user: user, l: l}
	if e.Now().Sub(s.touched) > lifetime {
		s.pending = nil
		s.form = nil
		v.button("Открыть кабинет", func(v *view) (screens.Screen, error) { return v.home() })
		return v.screen(l.T("Черновик истёк. Начните заново.")), true, nil
	}
	s.touched = e.Now()
	s.actions = map[string]action{}
	f := *s.pending
	value, err := parseField(f, text)
	if err != nil {
		screen, e2 := v.prompt(f, 0)
		screen.Text = l.T("Проверьте значение и попробуйте ещё раз.") + "\n\n" + screen.Text
		return screen, true, e2
	}
	if f.append {
		old := s.form.values[f.key]
		switch f.kind {
		case "program":
			a, _ := old.([]any)
			b, _ := value.([]any)
			if len(a)+len(b) > f.max {
				screen, e2 := v.prompt(f, 0)
				screen.Text = l.T("Проверьте значение и попробуйте ещё раз.") + "\n\n" + screen.Text
				return screen, true, e2
			}
			value = append(a, b...)
		default:
			value = str(old) + "\n\n" + str(value)
			if len([]rune(str(value))) > f.max {
				screen, e2 := v.prompt(f, 0)
				screen.Text = l.T("Проверьте значение и попробуйте ещё раз.") + "\n\n" + screen.Text
				return screen, true, e2
			}
		}
	}
	if f.kind == "photos" {
		old, _ := s.form.values[f.key].([]any)
		add, _ := value.([]any)
		if len(old)+len(add) > f.max {
			screen, e2 := v.prompt(f, 0)
			screen.Text = l.T("Слишком много фотографий.") + "\n\n" + screen.Text
			return screen, true, e2
		}
		value = append(old, add...)
	}
	s.form.values[f.key] = value
	s.pending = nil
	screen, err := v.edit(s.form, 0)
	return screen, true, err
}
func (v *view) request(method, path string, body any) (any, error) {
	var result any
	// DELETE and read markers may respond with 204 and no JSON.
	if method == "DELETE" || strings.HasSuffix(path, "/read") {
		return nil, v.e.API.CabinetRequest(v.ctx, v.user, method, path, body, nil)
	}
	err := v.e.API.CabinetRequest(v.ctx, v.user, method, path, body, &result)
	return result, err
}
func (v *view) button(label string, a action) { v.rawButton(v.l.T(label), a) }
func (v *view) rawButton(label string, a action) {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	token := hex.EncodeToString(b[:])
	v.s.actions[token] = a
	v.rows = append(v.rows, []telego.InlineKeyboardButton{keyboards.Button(short(label, 60), "k:"+token)})
}
func (v *view) screen(text string) screens.Screen {
	return screens.Screen{Text: text, Markup: &telego.InlineKeyboardMarkup{InlineKeyboard: v.rows}, Preview: &telego.LinkPreviewOptions{IsDisabled: true}}
}
func (v *view) home() (screens.Screen, error) {
	v.s.form = nil
	v.s.pending = nil
	value, err := v.request("GET", "/auth/user/me", nil)
	if err != nil {
		return screens.Screen{}, err
	}
	me := object(object(value)["user"])
	keys := []string{"catalog", "chats"}
	if str(me["partnerStatus"]) == "approved" {
		if str(me["partnerKind"]) == "driver" {
			keys = append(keys, "driverSummary", "driverProfile", "vehicles", "trips", "driverExpenses", "driverReports")
		} else {
			keys = append(keys, "summary", "tours", "drivers", "reports", "leads", "reviews", "stats", "bookings", "finances")
		}
		keys = append(keys, "clients", "tasks", "templates", "broadcasts", "awards")
	}
	for _, key := range keys {
		v.button(v.e.resources[key].title, v.listAction(key, nil, nil, 0))
	}
	v.rows = append(v.rows, []telego.InlineKeyboardButton{keyboards.Button(v.l.T("👤 Профиль"), "p:home"), keyboards.Button(v.l.T("🧑‍✈️ Комната организатора"), screens.OrgHome)})
	return v.screen("<b>" + v.l.T("Туры и партнёры") + "</b>\n\n" + v.l.T("Выберите раздел. Изменения и сообщения подтверждаются перед отправкой.")), nil
}
func (v *view) authorize(r resource) error {
	if r.role == "" {
		return nil
	}
	value, err := v.request("GET", "/auth/user/me", nil)
	if err != nil {
		return err
	}
	me := object(object(value)["user"])
	if str(me["partnerStatus"]) != "approved" || (r.role != "partner" && (r.role == "driver" && str(me["partnerKind"]) != "driver" || r.role == "organizer" && str(me["partnerKind"]) == "driver")) {
		return &api.Error{Status: 403, Message: "Partner access required"}
	}
	return nil
}
func pathFor(pattern string, vars map[string]string) string {
	for k, val := range vars {
		pattern = strings.ReplaceAll(pattern, "{"+k+"}", url.PathEscape(val))
	}
	return pattern
}
func cloneVars(vars map[string]string) map[string]string {
	out := map[string]string{}
	for k, val := range vars {
		out[k] = val
	}
	return out
}
func object(a any) record {
	m, _ := a.(map[string]any)
	if m == nil {
		return record{}
	}
	return m
}
func str(a any) string {
	if a == nil {
		return ""
	}
	return fmt.Sprint(a)
}
func short(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}
func title(r record) string {
	for _, key := range []string{"title", "name", "model", "routeTo", "text", "body", "category", "paidOn", "startDate"} {
		if s := str(r[key]); s != "" {
			return short(s, 65)
		}
	}
	for _, key := range []string{"account", "peer", "client", "listing", "driver"} {
		if nested := object(r[key]); len(nested) > 0 {
			for _, k := range []string{"firstName", "username", "name", "title"} {
				if s := str(nested[k]); s != "" {
					return short(s, 65)
				}
			}
		}
	}
	return "•"
}
func (v *view) listAction(key string, vars map[string]string, query record, page int) action {
	return func(v *view) (screens.Screen, error) { return v.list(v.e.resources[key], vars, query, page) }
}
func queryPath(path string, query record) string {
	u, err := url.Parse(path)
	if err != nil {
		return path
	}
	q := u.Query()
	for k, val := range query {
		if val != nil && str(val) != "" {
			q.Set(k, str(val))
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}
func (v *view) load(r resource, vars map[string]string, query record, page int) ([]record, int, record, error) {
	p := pathFor(r.list, vars)
	q := record{}
	for k, val := range query {
		q[k] = val
	}
	if r.key == "catalog" {
		q["locale"] = string(v.l)
	}
	paged := r.key == "clients" || r.key == "reviews" || r.key == "tourReviews" || r.key == "posts"
	if paged {
		q["limit"] = pageSize
		q["offset"] = page * pageSize
	}
	value, err := v.request("GET", queryPath(p, q), nil)
	if err != nil {
		return nil, 0, nil, err
	}
	meta := object(value)
	raw := value
	if r.extract != "" {
		raw = meta[r.extract]
		if r.key == "payments" {
			for _, p := range array(meta["participants"]) {
				if str(p["id"]) == vars["participant"] {
					raw = p["payments"]
					break
				}
			}
		}
	} else if items, ok := meta["items"]; ok {
		raw = items
	}
	if r.single {
		return []record{meta}, 1, meta, nil
	}
	items := array(raw)
	for _, item := range items {
		localize(item, string(v.l))
	}
	if search := strings.ToLower(str(query["q"])); search != "" && (r.key == "catalog" || r.key == "drivers") {
		filtered := []record{}
		for _, item := range items {
			if strings.Contains(strings.ToLower(title(item)+" "+str(item["location"])+" "+str(item["excerpt"])), search) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	if !paged && r.key != "catalog" {
		filtered := []record{}
		for _, item := range items {
			if stage := str(query["stage"]); stage != "" && str(item["stage"]) != stage {
				continue
			}
			if status := str(query["status"]); status != "" && str(item["status"]) != status {
				continue
			}
			if q := strings.ToLower(str(query["q"])); q != "" && !strings.Contains(strings.ToLower(title(item)+" "+str(item["phone"])+" "+str(item["note"])), q) {
				continue
			}
			if phase := str(query["phase"]); phase != "" {
				raw, _ := json.Marshal(item)
				var t api.Tour
				_ = json.Unmarshal(raw, &t)
				actual := "current"
				if v.e.Now().Before(t.StartsAt()) {
					actual = "upcoming"
				} else if !v.e.Now().Before(t.EndsAt()) {
					actual = "past"
				}
				if phase != actual {
					continue
				}
			}
			filtered = append(filtered, item)
		}
		items = filtered
	}
	if r.key == "catalog" {
		items = filterTours(items, query)
	}
	total := len(items)
	if paged {
		if n, ok := meta["total"].(float64); ok {
			total = int(n)
		}
	} else {
		start := min(max(0, page)*pageSize, len(items))
		items = items[start:min(start+pageSize, len(items))]
	}
	return items, total, meta, nil
}
func array(a any) []record {
	raw, _ := a.([]any)
	out := make([]record, 0, len(raw))
	for _, r := range raw {
		out = append(out, object(r))
	}
	return out
}
func (v *view) list(r resource, vars map[string]string, query record, page int) (screens.Screen, error) {
	v.s.form = nil
	if err := v.authorize(r); err != nil {
		return screens.Screen{}, err
	}
	items, total, meta, err := v.load(r, vars, query, page)
	if err != nil {
		return screens.Screen{}, err
	}
	back := v.listAction(r.key, vars, query, page)
	if r.single {
		r.list = queryPath(r.list, query)
		return v.card(r, vars, meta, func(v *view) (screens.Screen, error) { return v.home() }, 0)
	}
	for _, item := range items {
		iv := cloneVars(vars)
		iv["id"] = str(item["id"])
		v.rawButton(title(item), func(v *view) (screens.Screen, error) { return v.card(r, iv, item, back, 0) })
	}
	if page > 0 {
		v.button("◀️ Назад", v.listAction(r.key, vars, query, page-1))
	}
	if (page+1)*pageSize < total {
		v.button("Дальше ▶️", v.listAction(r.key, vars, query, page+1))
	}
	if r.create != "" && r.key != "posts" {
		v.button("➕ Добавить", func(v *view) (screens.Screen, error) { return v.begin(r, vars, nil, false, back) })
	}
	if r.key == "catalog" {
		if destination := str(query["destination"]); destination != "" {
			v.button("Уведомления по направлению", func(v *view) (screens.Screen, error) { return v.destinationAlert(destination, back) })
		}
	}
	if len(r.filters) > 0 {
		v.button("🔎 Фильтры", func(v *view) (screens.Screen, error) { return v.filter(r, vars, query, back) })
	}
	v.button("Обновить", back)
	v.button("Открыть кабинет", func(v *view) (screens.Screen, error) { return v.home() })
	text := "<b>" + v.l.T(r.title) + "</b> · " + strconv.Itoa(total)
	if total == 0 {
		text += "\n\n" + v.l.T("Пока ничего нет.")
	}
	if len(query) > 0 {
		text += "\n\n" + v.describe(query)
	}
	if totals := object(meta["totals"]); len(totals) > 0 {
		text += "\n\n" + v.describe(totals)
	}
	if r.key == "posts" { // Only the owner gets the publishing action.
		tour, e := v.request("GET", "/tours/by-id/"+url.PathEscape(vars["tour"]), nil)
		if e == nil && object(tour)["isOrganizer"] == true {
			v.button("➕ Добавить", func(v *view) (screens.Screen, error) { return v.begin(r, vars, nil, false, back) })
		}
	}
	return v.screen(text), nil
}

func localize(item record, lang string) {
	for _, key := range []string{"title", "description"} {
		if values, ok := item[key].(map[string]any); ok {
			if text := str(values[lang]); text != "" {
				item[key] = text
			} else {
				item[key] = values["ru"]
			}
		}
	}
}
