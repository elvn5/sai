package routes_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/elvn5/sai/services/pkg/telegram"
)

type crmGroup struct {
	Participants []struct {
		ID       string  `json:"id"`
		ClientID string  `json:"clientId"`
		UserID   *string `json:"userId"`
		Name     *string `json:"name"`
		Status   string  `json:"status"`
		Due      int     `json:"due"`
		Paid     int     `json:"paid"`
		Payments []struct {
			ID string `json:"id"`
		} `json:"payments"`
	} `json:"participants"`
	Totals struct {
		Active, Waitlist, Due, Paid int
	} `json:"totals"`
}

func group(t *testing.T, body string) crmGroup {
	t.Helper()
	var g crmGroup
	if err := json.Unmarshal([]byte(body), &g); err != nil {
		t.Fatalf("not a group: %s", body)
	}
	return g
}

func TestCRM(t *testing.T) {
	c, notifier, tg, _ := setupTrips(t, telegram.RU)
	ctx := context.Background()
	_, body := c.do("POST", "/tours", strings.Replace(tripTour, `"capacity":2`, `"capacity":1,"price":5000`, 1), partner)
	id := obj(t, body)["id"].(string)
	c.do("POST", "/tours/"+id+"/join", "", guest)
	c.do("POST", "/tours/"+id+"/join", "", owner) // the waitlist
	notifier.Tick(ctx)
	tg.take()

	// Those who signed up are the partner's clients; others see nothing.
	_, body = c.do("GET", "/crm/clients", "", partner)
	if obj(t, body)["total"] != float64(2) || !strings.Contains(body, `"firstName":"User 2222"`) {
		t.Fatalf("clients: %s", body)
	}
	if status, _ := c.do("GET", "/crm/clients", "", regular); status != 403 {
		t.Errorf("clients as a non-partner: %d", status)
	}
	if status, _ := c.do("GET", "/crm/tours/"+id+"/participants", "", owner); status != 404 {
		t.Errorf("another partner's group: %d", status)
	}

	// A client added by hand.
	status, body := c.do("POST", "/crm/clients", `{"name":"  Айбек ","phone":"+996 555 123456","telegram":"@aibek_kg","tags":["vip"," vip ","горы"]}`, partner)
	manual := obj(t, body)
	if status != 201 || manual["name"] != "Айбек" || manual["telegram"] != "aibek_kg" || len(manual["tags"].([]any)) != 2 {
		t.Fatalf("create client: %d %s", status, body)
	}
	if status, body := c.do("POST", "/crm/clients", `{"phone":"+996 555 1"}`, partner); status != 400 || !strings.Contains(body, "name") {
		t.Errorf("client without a name: %d %s", status, body)
	}
	if _, body := c.do("GET", "/crm/clients?q=айб&tag=vip", "", partner); obj(t, body)["total"] != float64(1) {
		t.Errorf("search: %s", body)
	}
	if _, body := c.do("GET", "/crm/clients/tags", "", partner); body != `["vip","горы"]` {
		t.Errorf("tags: %s", body)
	}

	// The tour is full: the client goes on the waitlist.
	manualID := manual["id"].(string)
	if status, body := c.do("POST", "/crm/tours/"+id+"/participants", `{"clientId":"`+manualID+`"}`, partner); status != 409 || !strings.Contains(body, "No places left") {
		t.Errorf("add to a full tour: %d %s", status, body)
	}
	_, body = c.do("POST", "/crm/tours/"+id+"/participants", `{"clientId":"`+manualID+`","status":"waitlist"}`, partner)
	g := group(t, body)
	if len(g.Participants) != 3 || g.Totals.Active != 1 || g.Totals.Waitlist != 2 || g.Participants[0].Due != 5000 {
		t.Fatalf("group: %s", body)
	}
	guestRow, ownerRow, manualRow := g.Participants[0], g.Participants[1], g.Participants[2]
	if *guestRow.UserID != guest || *ownerRow.UserID != owner || manualRow.UserID != nil || *manualRow.Name != "Айбек" {
		t.Fatalf("group order: %s", body)
	}

	// Cancelling the guest gives the place to the first on the waitlist,
	// who hears of it.
	_, body = c.do("PATCH", "/crm/tours/"+id+"/participants/"+guestRow.ID, `{"status":"cancelled"}`, partner)
	if g := group(t, body); g.Participants[0].Status != "new" || *g.Participants[0].UserID != owner || g.Totals.Waitlist != 1 {
		t.Fatalf("promoted: %s", body)
	}
	notifier.Tick(ctx)
	if sent := tg.take(); len(sent) != 1 || sent[0].chat != 1001 || !strings.Contains(sent[0].text, "Освободилось место") {
		t.Fatalf("promoted message: %+v", sent)
	}
	if _, body := c.do("GET", "/tours/by-id/"+id, "", guest); obj(t, body)["isJoined"] != false || obj(t, body)["myStatus"] != nil {
		t.Errorf("cancelled participant's view: %s", body)
	}

	// What they owe and paid.
	c.do("PATCH", "/crm/tours/"+id+"/participants/"+ownerRow.ID, `{"amountDue":4500,"status":"confirmed","arrived":true}`, partner)
	_, body = c.do("POST", "/crm/tours/"+id+"/participants/"+ownerRow.ID+"/payments", `{"amount":2000,"method":"cash","paidOn":"2026-10-09","note":" аванс "}`, partner)
	g = group(t, body)
	if g.Totals.Due != 4500 || g.Totals.Paid != 2000 || len(g.Participants[0].Payments) != 1 || g.Participants[0].Status != "confirmed" {
		t.Fatalf("payment: %s", body)
	}
	if status, body := c.do("POST", "/crm/tours/"+id+"/participants/"+ownerRow.ID+"/payments", `{"amount":0,"method":"barter","paidOn":"2026-13-01"}`, partner); status != 400 ||
		!strings.Contains(body, "amount") || !strings.Contains(body, "method") || !strings.Contains(body, "paidOn") {
		t.Errorf("payment validation: %d %s", status, body)
	}
	_, body = c.do("GET", "/crm/summary", "", partner)
	summary := obj(t, body)
	if summary["clients"] != float64(3) || summary["paidThisMonth"] != float64(2000) || summary["outstanding"] != float64(2500) ||
		!strings.Contains(body, `"taken":1,"waitlist":1`) {
		t.Errorf("summary: %s", body)
	}
	_, body = c.do("GET", "/crm/clients/"+guestRow.ClientID, "", partner)
	if !strings.Contains(body, `"status":"cancelled"`) || !strings.Contains(body, `"title":"Ала-Куль <тест>"`) {
		t.Errorf("client card: %s", body)
	}
	if status, _ := c.do("DELETE", "/crm/payments/"+g.Participants[0].Payments[0].ID, "", owner); status != 404 {
		t.Errorf("delete another partner's payment: %d", status)
	}

	// Accounts can only be cancelled; clients added by hand can go.
	if status, _ := c.do("DELETE", "/crm/tours/"+id+"/participants/"+ownerRow.ID, "", partner); status != 409 {
		t.Errorf("delete an account's place: %d", status)
	}
	if status, _ := c.do("DELETE", "/crm/clients/"+guestRow.ClientID, "", partner); status != 409 {
		t.Errorf("delete an account client: %d", status)
	}
	if status, body := c.do("DELETE", "/crm/tours/"+id+"/participants/"+manualRow.ID, "", partner); status != 200 || len(group(t, body).Participants) != 2 {
		t.Errorf("delete a manual place: %d %s", status, body)
	}
	if status, _ := c.do("DELETE", "/crm/clients/"+manualID, "", partner); status != 204 {
		t.Errorf("delete a manual client: %d", status)
	}
}

func TestChats(t *testing.T) {
	c, notifier, tg, clock := setupTrips(t, telegram.EN)
	notifier.SiteURL = "https://ermekten.test"
	ctx := context.Background()
	_, body := c.do("POST", "/tours", tripTour, partner)
	id := obj(t, body)["id"].(string)
	c.do("POST", "/tours/"+id+"/join", "", guest)
	notifier.Tick(ctx)
	tg.take()

	// Partners write to their clients; travelers to organizers of tours.
	if status, _ := c.do("POST", "/chats", `{"userId":"`+regular+`"}`, partner); status != 404 {
		t.Errorf("chat with a stranger: %d", status)
	}
	if status, _ := c.do("POST", "/chats", `{"partnerId":"`+owner+`"}`, regular); status != 404 {
		t.Errorf("chat with someone without tours: %d", status)
	}
	status, body := c.do("POST", "/chats", `{"userId":"`+guest+`"}`, partner)
	chat := obj(t, body)
	if status != 200 || chat["role"] != "partner" || !strings.Contains(body, `"peer":{"id":"`+guest) {
		t.Fatalf("open: %d %s", status, body)
	}
	chatID := chat["id"].(string)
	if _, body := c.do("POST", "/chats", `{"partnerId":"`+partner+`"}`, guest); obj(t, body)["id"] != chatID || obj(t, body)["role"] != "client" {
		t.Errorf("the same chat from the other side: %s", body)
	}

	// A message reaches the client in the bot once per while, with a link
	// to the chat on the site.
	if status, body := c.do("POST", "/chats/"+chatID+"/messages", `{"body":"  Сбор в 6:00 <b>у ворот</b> "}`, partner); status != 201 || obj(t, body)["body"] != "Сбор в 6:00 <b>у ворот</b>" {
		t.Fatalf("send: %d %s", status, body)
	}
	*clock = clock.Add(time.Minute)
	c.do("POST", "/chats/"+chatID+"/messages", `{"body":"Возьмите воду"}`, partner)
	if status, _ := c.do("POST", "/chats/"+chatID+"/messages", `{"body":"   "}`, partner); status != 400 {
		t.Errorf("empty message: %d", status)
	}
	notifier.Tick(ctx)
	sent := tg.take()
	if len(sent) != 1 || sent[0].chat != 1002 || !strings.Contains(sent[0].text, "Новое сообщение от <b>User 3333</b>:\n\nСбор в 6:00 &lt;b&gt;у ворот&lt;/b&gt;") ||
		sent[0].button == nil || sent[0].button.URL != "https://ermekten.test/messages/"+chatID {
		t.Fatalf("message notification: %+v", sent)
	}
	if _, body := c.do("GET", "/chats/unread", "", guest); body != `{"chats":1}` {
		t.Errorf("unread: %s", body)
	}
	if _, body := c.do("GET", "/chats", "", guest); !strings.Contains(body, `"unread":2`) || !strings.Contains(body, `"body":"Возьмите воду"`) {
		t.Errorf("list: %s", body)
	}

	// Reading the chat marks it read; outsiders can't.
	_, body = c.do("GET", "/chats/"+chatID+"/messages", "", guest)
	if len(obj(t, body)["messages"].([]any)) != 2 {
		t.Errorf("messages: %s", body)
	}
	if _, body := c.do("GET", "/chats/unread", "", guest); body != `{"chats":0}` {
		t.Errorf("unread after reading: %s", body)
	}
	if _, body := c.do("GET", "/chats/"+chatID+"/messages", "", partner); obj(t, body)["peerReadAt"] == nil {
		t.Errorf("peer read: %s", body)
	}
	if status, _ := c.do("GET", "/chats/"+chatID+"/messages", "", regular); status != 404 {
		t.Errorf("outsider: %d", status)
	}

	// The partner reads replies in their language, with the partner's link.
	*clock = clock.Add(time.Minute)
	c.do("POST", "/chats/"+chatID+"/messages", `{"body":"Спасибо!"}`, guest)
	notifier.Tick(ctx)
	if sent := tg.take(); len(sent) != 1 || sent[0].chat != 1003 || !strings.Contains(sent[0].text, "New message from") ||
		sent[0].button.URL != "https://ermekten.test/en/partner/messages/"+chatID || sent[0].button.Text != "💬 Reply" {
		t.Fatalf("reply notification: %+v", sent)
	}
	if _, body := c.do("GET", "/crm/clients", "", partner); !strings.Contains(body, guest) {
		t.Errorf("clients: %s", body)
	}
	_, body = c.do("GET", "/crm/clients", "", partner)
	var page struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	_ = json.Unmarshal([]byte(body), &page)
	if _, body := c.do("GET", "/crm/clients/"+page.Items[0].ID, "", partner); obj(t, body)["chatId"] != chatID {
		t.Errorf("client's chat: %s", body)
	}
}

func TestParticipantForm(t *testing.T) {
	c, _, _, _ := setupTrips(t, telegram.RU)
	_, body := c.do("POST", "/tours", tripTour, partner)
	id := obj(t, body)["id"].(string)

	// A bad form refuses the sign-up as a whole.
	if status, body := c.do("POST", "/tours/"+id+"/join", `{"form":{"emergencyName":" ","emergencyPhone":"x","experience":"guru"},"acceptWaiver":true}`, guest); status != 400 ||
		!strings.Contains(body, "form") {
		t.Fatalf("bad form: %d %s", status, body)
	}
	if _, body := c.do("GET", "/tours/by-id/"+id, "", guest); obj(t, body)["myStatus"] != nil {
		t.Fatalf("joined despite a bad form: %s", body)
	}

	// The form and the rules come with the sign-up; the participant sees
	// their own.
	status, body := c.do("POST", "/tours/"+id+"/join", `{"form":{"emergencyName":"  Мама ","emergencyPhone":"+996 555 000111","experience":"hiking","health":" астма "},"acceptWaiver":true}`, guest)
	me := obj(t, body)
	if status != 200 || me["myWaiverAt"] == nil || !strings.Contains(body, `"myForm":{"emergencyName":"Мама","emergencyPhone":"+996 555 000111","experience":"hiking","health":"астма","notes":null}`) {
		t.Fatalf("join with a form: %d %s", status, body)
	}
	// The bot signs up without a body; the form comes later.
	c.do("POST", "/tours/"+id+"/join", "", owner)
	if _, body := c.do("GET", "/tours/by-id/"+id, "", owner); strings.Contains(body, "myForm") || strings.Contains(body, "myWaiverAt") {
		t.Errorf("no form yet: %s", body)
	}
	if status, body := c.do("PUT", "/tours/"+id+"/form", `{"acceptWaiver":true}`, owner); status != 200 || obj(t, body)["myWaiverAt"] == nil {
		t.Errorf("accept later: %d %s", status, body)
	}
	if status, _ := c.do("PUT", "/tours/"+id+"/form", `{"acceptWaiver":true}`, regular); status != 404 {
		t.Errorf("form without signing up: %d", status)
	}

	// The organizer sees the forms in the group, and fills one in for a
	// client added by hand.
	_, body = c.do("POST", "/crm/clients", `{"name":"Айбек"}`, partner)
	c.do("POST", "/crm/tours/"+id+"/participants", `{"clientId":"`+obj(t, body)["id"].(string)+`","status":"waitlist"}`, partner)
	_, body = c.do("GET", "/crm/tours/"+id+"/participants", "", partner)
	g := group(t, body)
	if !strings.Contains(body, `"health":"астма"`) || !strings.Contains(body, `"waiverVersion":"v1"`) || !strings.Contains(body, `"forms":1,"waivers":2`) {
		t.Fatalf("group forms: %s", body)
	}
	manual := g.Participants[2].ID
	if status, body := c.do("PUT", "/crm/tours/"+id+"/participants/"+manual+"/form", `{"form":{"emergencyName":"Брат","emergencyPhone":"+996 700 1","experience":"none"}}`, partner); status != 200 ||
		!strings.Contains(body, `"emergencyName":"Брат"`) {
		t.Errorf("partner fills a form: %d %s", status, body)
	}
	if status, _ := c.do("PUT", "/crm/tours/"+id+"/participants/"+manual+"/form", `{"form":null}`, owner); status != 404 {
		t.Errorf("another partner's participant: %d", status)
	}
}

func TestCRMTools(t *testing.T) {
	c, notifier, tg, clock := setupTrips(t, telegram.RU)
	ctx := context.Background()
	_, body := c.do("POST", "/tours", strings.Replace(tripTour, `"capacity":2`, `"capacity":3,"price":5000`, 1), partner)
	id := obj(t, body)["id"].(string)

	// A traveler who writes to the organizer becomes a client with a lead;
	// signing up wins it.
	c.do("POST", "/chats", `{"partnerId":"`+partner+`"}`, guest)
	c.do("POST", "/chats", `{"partnerId":"`+partner+`"}`, guest) // no second lead
	_, body = c.do("GET", "/crm/leads", "", partner)
	if strings.Count(body, `"stage":"new"`) != 1 || !strings.Contains(body, `"source":"ermekten"`) || !strings.Contains(body, `"firstName":"User 2222"`) {
		t.Fatalf("inquiry lead: %s", body)
	}
	c.do("POST", "/tours/"+id+"/join", "", guest)
	if _, body := c.do("GET", "/crm/leads", "", partner); !strings.Contains(body, `"stage":"won"`) || !strings.Contains(body, `"tour":{"id":"`+id) {
		t.Fatalf("won lead: %s", body)
	}

	// A lead by hand, moved through the stages; lost needs no tour.
	_, body = c.do("POST", "/crm/clients", `{"name":"Айбек","source":"instagram","tags":["vip"]}`, partner)
	manual := obj(t, body)["id"].(string)
	if obj(t, body)["source"] != "instagram" {
		t.Errorf("client source: %s", body)
	}
	status, body := c.do("POST", "/crm/leads", `{"clientId":"`+manual+`","tourId":"`+id+`","note":"звонил"}`, partner)
	lead := obj(t, body)
	if status != 201 || lead["stage"] != "new" || lead["note"] != "звонил" {
		t.Fatalf("create lead: %d %s", status, body)
	}
	if _, body := c.do("PATCH", "/crm/leads/"+lead["id"].(string), `{"stage":"lost","lostReason":"дорого"}`, partner); obj(t, body)["lostReason"] != "дорого" {
		t.Errorf("lose lead: %s", body)
	}
	if status, _ := c.do("POST", "/crm/leads", `{"clientId":"`+manual+`"}`, owner); status != 404 {
		t.Errorf("lead on another partner's client: %d", status)
	}

	// Tasks: open ones by due day; done ones drop out.
	c.do("POST", "/crm/tasks", `{"title":"Напомнить о доплате","dueOn":"2026-10-12","clientId":"`+manual+`"}`, partner)
	_, body = c.do("POST", "/crm/tasks", `{"title":"Купить газ","dueOn":"2026-10-11","tourId":"`+id+`"}`, partner)
	var tasks []map[string]any
	_ = json.Unmarshal([]byte(body), &tasks)
	if len(tasks) != 2 || tasks[0]["title"] != "Купить газ" || tasks[1]["client"].(map[string]any)["name"] != "Айбек" {
		t.Fatalf("tasks: %s", body)
	}
	if _, body := c.do("PATCH", "/crm/tasks/"+tasks[0]["id"].(string), `{"done":true}`, partner); strings.Contains(body, "Купить газ") {
		t.Errorf("done task still open: %s", body)
	}
	if _, body := c.do("GET", "/crm/clients/"+manual, "", partner); !strings.Contains(body, "Напомнить о доплате") || !strings.Contains(body, `"lostReason":"дорого"`) {
		t.Errorf("client card: %s", body)
	}

	// The tour's money: a payment, a refund, expenses; staff.
	_, body = c.do("GET", "/crm/tours/"+id+"/participants", "", partner)
	pid := group(t, body).Participants[0].ID
	c.do("POST", "/crm/tours/"+id+"/participants/"+pid+"/payments", `{"amount":5000,"method":"cash","paidOn":"2026-10-09"}`, partner)
	c.do("POST", "/crm/tours/"+id+"/participants/"+pid+"/payments", `{"kind":"refund","amount":1000,"method":"transfer","paidOn":"2026-10-10"}`, partner)
	c.do("POST", "/crm/tours/"+id+"/expenses", `{"category":"transport","amount":2500,"note":"Спринтер"}`, partner)
	_, body = c.do("POST", "/crm/tours/"+id+"/staff", `{"role":"guide","name":" Азамат ","phone":"+996 555 7"}`, partner)
	if !strings.Contains(body, `"paid":4000`) || !strings.Contains(body, `"expenses":2500,"margin":1500`) || !strings.Contains(body, `"name":"Азамат"`) {
		t.Fatalf("tour money: %s", body)
	}
	g := obj(t, body)
	expense := g["expenses"].([]any)[0].(map[string]any)["id"].(string)
	if status, _ := c.do("DELETE", "/crm/expenses/"+expense, "", owner); status != 404 {
		t.Errorf("another partner's expense: %d", status)
	}
	if _, body := c.do("DELETE", "/crm/expenses/"+expense, "", partner); !strings.Contains(body, `"expenses":[]`) {
		t.Errorf("delete expense: %s", body)
	}

	// Templates and a broadcast to clients with a tag: accounts get it in
	// their chat, clients added by hand are skipped.
	_, body = c.do("POST", "/crm/templates", `{"title":"Список вещей","body":"Ботинки, куртка, вода"}`, partner)
	if !strings.Contains(body, "Список вещей") {
		t.Errorf("template: %s", body)
	}
	_, body = c.do("GET", "/crm/clients?q=user%202222", "", partner)
	var page struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	_ = json.Unmarshal([]byte(body), &page)
	c.do("PUT", "/crm/clients/"+page.Items[0].ID, `{"tags":["vip"]}`, partner)
	*clock = clock.Add(time.Hour)
	notifier.Tick(ctx)
	tg.take()
	if status, body := c.do("POST", "/crm/broadcasts", `{"audience":"tag:vip","body":"Скидка 10% на зимние туры"}`, partner); status != 201 || body != `{"sent":1,"skipped":1}` {
		t.Fatalf("broadcast: %d %s", status, body)
	}
	notifier.Tick(ctx)
	if sent := tg.take(); len(sent) != 1 || sent[0].chat != 1002 || !strings.Contains(sent[0].text, "Скидка 10%") {
		t.Fatalf("broadcast message: %+v", sent)
	}
	if _, body := c.do("GET", "/crm/broadcasts", "", partner); !strings.Contains(body, `"audience":"tag:vip"`) {
		t.Errorf("broadcasts: %s", body)
	}
	if status, _ := c.do("POST", "/crm/broadcasts", `{"audience":"tour:`+id+`","body":"x"}`, owner); status != 404 {
		t.Errorf("broadcast to another partner's tour: %d", status)
	}

	// The report over the tour's dates.
	_, body = c.do("GET", "/crm/reports?from=2026-10-01&to=2026-10-31", "", partner)
	report := obj(t, body)
	if report["tours"] != float64(1) || report["participants"] != float64(1) || report["revenue"] != float64(4000) ||
		report["places"] != float64(3) || report["clients"] != float64(1) || !strings.Contains(body, `{"source":"instagram","total":1,"won":0,"lost":1}`) {
		t.Errorf("report: %s", body)
	}
	if status, _ := c.do("GET", "/crm/reports?from=2026-10-31&to=2026-10-01", "", partner); status != 400 {
		t.Errorf("bad period: %d", status)
	}
}
