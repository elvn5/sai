package routes_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/listings/internal/alerts"
	"github.com/elvn5/sai/services/listings/internal/handlers"
	"github.com/elvn5/sai/services/listings/internal/routes"
	"github.com/elvn5/sai/services/listings/internal/store"
	"github.com/elvn5/sai/services/pkg/telegram"
)

// Only the guest has Telegram linked.
type fakeChats struct{}

func (fakeChats) TelegramChats(_ context.Context, ids []string) (map[string]int64, error) {
	out := map[string]int64{}
	for _, id := range ids {
		if id == guest {
			out[id] = 42
		}
	}
	return out, nil
}

type alertMessage struct {
	chat   int64
	text   string
	button *telegram.Button
}

type fakeSender chan alertMessage

func (f fakeSender) SendMessage(_ context.Context, chat int64, text string, button *telegram.Button) error {
	f <- alertMessage{chat, text, button}
	return nil
}

func setupWithAlerts(t *testing.T) (client, fakeSender) {
	t.Helper()
	db, _ := createDatabase(t, true)
	st := store.New(db)
	sent := make(fakeSender, 4)
	log := slog.New(slog.DiscardHandler)
	gin.SetMode(gin.TestMode)
	return client{t, routes.Setup(handlers.Deps{Store: st, Authors: testUsers, Log: log, Alerts: &alerts.Notifier{
		Store: st, Chats: fakeChats{}, Sender: sent, Log: log,
	}}, testUsers, secret)}, sent
}

const detailedTour = `{"locale":"ru","slug":"son-kul","title":"Сон-Куль","content":"<p>Юрты</p>","startDate":"2027-07-01",
	"endDate":"2027-07-03","destination":"naryn","cancellationPolicy":"free","price":15000,"capacity":12,
	"photos":["/uploads/a.webp","/uploads/b.webp"],
	"program":[{"title":"Бишкек — Кочкор","text":"Выезд утром"},{"title":"Озеро","text":""}]}`

func TestTourDetailsAndDates(t *testing.T) {
	c := setup(t)
	status, body := c.do("POST", "/tours", detailedTour, partner)
	tr := obj(t, body)
	if status != 201 || tr["capacity"] != float64(12) || len(tr["photos"].([]any)) != 2 || len(tr["program"].([]any)) != 2 {
		t.Fatalf("create: %d %s", status, body)
	}
	if status, body := c.do("POST", "/tours", strings.Replace(strings.Replace(detailedTour, "son-kul", "x", 1),
		`"title":"Озеро"`, `"title":""`, 1), partner); status != 400 || !strings.Contains(body, "program") {
		t.Errorf("empty day title: %d %s", status, body)
	}
	if status, body := c.do("POST", "/tours", strings.Replace(strings.Replace(detailedTour, "son-kul", "y", 1),
		`"capacity":12`, `"capacity":0`, 1), partner); status != 400 || !strings.Contains(body, "capacity") {
		t.Errorf("zero capacity: %d %s", status, body)
	}

	_, body = c.do("GET", "/tours/ru/son-kul", "", "")
	if org, _ := obj(t, body)["organizer"].(map[string]any); org == nil || org["id"] != partner {
		t.Errorf("organizer: %s", body)
	}
	for query, want := range map[string]bool{
		"from=2027-06-01&to=2027-07-31": true, "from=2027-07-02": false, "to=2027-06-30": false, "from=nonsense": true,
	} {
		if _, body := c.do("GET", "/tours?locale=ru&"+query, "", ""); strings.Contains(body, "son-kul") != want {
			t.Errorf("%s: %s", query, body)
		}
	}
}

func TestFavorites(t *testing.T) {
	c := setup(t)
	_, body := c.do("POST", "/tours", detailedTour, partner)
	tourID := obj(t, body)["id"].(string)
	hotelID := createHotel(c, "saved-inn", "")

	if status, _ := c.do("PUT", "/favorites/tour/"+tourID, "", ""); status != 401 {
		t.Errorf("anonymous: %d", status)
	}
	if status, _ := c.do("PUT", "/favorites/tour/bbbbbbbb-0000-0000-0000-000000000009", "", guest); status != 404 {
		t.Errorf("unknown tour: %d", status)
	}
	if status, _ := c.do("PUT", "/favorites/trail/"+tourID, "", guest); status != 404 {
		t.Errorf("unknown type: %d", status)
	}
	for range 2 {
		if status, body := c.do("PUT", "/favorites/tour/"+tourID, "", guest); status != 200 || body != `{"saved":true}` {
			t.Fatalf("save: %d %s", status, body)
		}
	}
	c.do("PUT", "/favorites/hotel/"+hotelID, "", guest)
	if _, body := c.do("GET", "/favorites/mine/ids", "", guest); body != `{"hotel":["`+hotelID+`"],"local_guide":[],"tour":["`+tourID+`"]}` {
		t.Errorf("ids: %s", body)
	}
	_, body = c.do("GET", "/favorites/mine", "", guest)
	if mine := obj(t, body); len(mine["tours"].([]any)) != 1 || len(mine["hotels"].([]any)) != 1 || len(mine["localGuides"].([]any)) != 0 {
		t.Errorf("mine: %s", body)
	}
	if _, body := c.do("GET", "/favorites/mine/ids", "", owner); body != `{"hotel":[],"local_guide":[],"tour":[]}` {
		t.Errorf("someone else's: %s", body)
	}
	c.do("DELETE", "/favorites/tour/"+tourID, "", guest)
	if _, body := c.do("GET", "/favorites/mine/ids", "", guest); strings.Contains(body, tourID) {
		t.Errorf("after remove: %s", body)
	}
}

func TestTourAlerts(t *testing.T) {
	c := setup(t)
	if status, _ := c.do("PUT", "/tour-alerts", `{"destination":""}`, guest); status != 503 {
		t.Errorf("without the bot: %d", status)
	}

	c, sent := setupWithAlerts(t)
	if status, body := c.do("PUT", "/tour-alerts", `{"destination":""}`, owner); status != 409 || !strings.Contains(body, "Link Telegram") {
		t.Errorf("not linked: %d %s", status, body)
	}
	if status, _ := c.do("PUT", "/tour-alerts", `{"destination":"atlantis"}`, guest); status != 400 {
		t.Errorf("bad destination: %d", status)
	}
	if status, body := c.do("PUT", "/tour-alerts", `{"destination":"naryn"}`, guest); status != 200 || body != `{"destinations":["naryn"]}` {
		t.Fatalf("follow: %d %s", status, body)
	}

	// A tour elsewhere says nothing; one in Naryn messages the follower.
	c.do("POST", "/tours", strings.Replace(strings.Replace(detailedTour, "son-kul", "karakol-tour", 1), "naryn", "karakol", 1), partner)
	c.do("POST", "/tours", detailedTour, partner)
	select {
	case m := <-sent:
		if m.chat != 42 || !strings.Contains(m.text, "Сон-Куль") || m.button == nil ||
			!strings.HasPrefix(m.button.Data, "t:open:") || m.button.URL != "" {
			t.Errorf("message: %+v", m)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no message")
	}
	select {
	case m := <-sent:
		t.Errorf("extra message: %+v", m)
	case <-time.After(200 * time.Millisecond):
	}

	if _, body := c.do("DELETE", "/tour-alerts", `{"destination":"naryn"}`, guest); body != `{"destinations":[]}` {
		t.Errorf("unfollow: %s", body)
	}
}
