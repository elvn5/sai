package keyboards_test

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/mymmrac/telego"

	"github.com/elvn5/sai/services/bot/internal/keyboards"
)

func asJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// The menu as drawn on the "Старт и главное меню" sketch.
func TestMainMenu(t *testing.T) {
	m := keyboards.MainMenu("https://ermekten.kg", true)
	var rows []string
	for _, row := range m.Keyboard {
		var cells []string
		for _, b := range row {
			cells = append(cells, b.Text)
		}
		rows = append(rows, strings.Join(cells, " | "))
	}
	want := "🥾 Мои выходы | 🧭 Туры\n📰 Блог | 🔔 Уведомления\n👤 Профиль | 🆘 Поддержка\n🏔 Открыть Ermekten"
	if strings.Join(rows, "\n") != want {
		t.Errorf("menu:\n%s", strings.Join(rows, "\n"))
	}
	if m.Keyboard[3][0].WebApp == nil || m.Keyboard[3][0].WebApp.URL != "https://ermekten.kg" ||
		!m.ResizeKeyboard || m.InputFieldPlaceholder != "Выберите раздел" {
		t.Errorf("menu options: %s", asJSON(m))
	}

	// Without the API: only the profile and support; without a Mini App, no web app button.
	if got := asJSON(keyboards.MainMenu("", false).Keyboard); got != `[[{"text":"👤 Профиль"},{"text":"🆘 Поддержка"}]]` {
		t.Errorf("minimal menu: %s", got)
	}
}

func TestPager(t *testing.T) {
	data := func(page int) string { return "p" + strconv.Itoa(page) }
	for _, tc := range []struct {
		index, n int
		want     string
	}{
		{0, 3, "◀️→p2 | 1 / 3→noop | ▶️→p1"},
		{1, 3, "◀️→p0 | 2 / 3→noop | ▶️→p2"},
		{2, 3, "◀️→p1 | 3 / 3→noop | ▶️→p0"},
		{0, 12, "◀️→p11 | 1 / 12→noop | ▶️→p1"},
	} {
		var cells []string
		for _, b := range keyboards.Pager(tc.index, tc.n, data) {
			cells = append(cells, b.Text+"→"+b.CallbackData)
		}
		if got := strings.Join(cells, " | "); got != tc.want {
			t.Errorf("Pager(%d, %d) = %s, want %s", tc.index, tc.n, got, tc.want)
		}
	}
}

func TestTabs(t *testing.T) {
	row := keyboards.Tabs("b", [2]string{"A", "a"}, [2]string{"B", "b"})
	if got := asJSON(row); got != `[{"text":"A","callback_data":"a"},{"text":"• B •","callback_data":"b"}]` {
		t.Errorf("tabs: %s", got)
	}
}

func TestButtons(t *testing.T) {
	if got := asJSON([]telego.InlineKeyboardButton{keyboards.Button("x", "d"), keyboards.Link("y", "https://e.kg")}); got !=
		`[{"text":"x","callback_data":"d"},{"text":"y","url":"https://e.kg"}]` {
		t.Errorf("buttons: %s", got)
	}
	if got := asJSON(keyboards.LinkMove("tok")); got !=
		`{"inline_keyboard":[[{"text":"Привязать","callback_data":"link_move:tok"},{"text":"Отмена","callback_data":"link_cancel"}]]}` {
		t.Errorf("link move: %s", got)
	}
}
