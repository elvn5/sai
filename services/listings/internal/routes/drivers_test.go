package routes_test

import (
	"context"
	"strings"
	"testing"

	"github.com/elvn5/sai/services/pkg/telegram"
)

func TestDrivers(t *testing.T) {
	c, notifier, tg, _ := setupTrips(t, telegram.RU)
	notifier.SiteURL = "https://ermekten.test"
	ctx := context.Background()

	// Drivers keep their own cabinet; organizers' tools aren't theirs.
	if status, _ := c.do("POST", "/tours", tripTour, driver); status != 403 {
		t.Errorf("a driver creating a tour: %d", status)
	}
	if status, _ := c.do("GET", "/driver/summary", "", partner); status != 403 {
		t.Errorf("an organizer in the driver's cabinet: %d", status)
	}
	if status, _ := c.do("GET", "/crm/drivers", "", driver); status != 403 {
		t.Errorf("a driver in the catalog: %d", status)
	}
	if _, body := c.do("GET", "/driver/profile", "", driver); obj(t, body)["published"] != false {
		t.Errorf("no profile yet: %s", body)
	}

	// The card, a vehicle with its documents.
	if status, body := c.do("PUT", "/driver/profile", `{"name":" Бакыт ","phone":"+996555123456","region":"Бишкек","published":true}`, driver); status != 200 || obj(t, body)["name"] != "Бакыт" {
		t.Fatalf("profile: %d %s", status, body)
	}
	if status, _ := c.do("POST", "/driver/vehicles", `{"kind":"tank","model":"T-34","seats":4}`, driver); status != 400 {
		t.Errorf("an unknown kind: %d", status)
	}
	status, body := c.do("POST", "/driver/vehicles", `{"kind":"minibus","model":"Mercedes Sprinter","plate":"01KG123ABC","seats":18,"insuranceUntil":"2026-10-20"}`, driver)
	vehicle := obj(t, body)
	if status != 201 || vehicle["seats"] != float64(18) || vehicle["active"] != true {
		t.Fatalf("vehicle: %d %s", status, body)
	}
	vehicleID := vehicle["id"].(string)
	if _, body := c.do("GET", "/driver/summary", "", driver); !strings.Contains(body, `"documents":[{"vehicleId":"`+vehicleID+`","vehicle":"Mercedes Sprinter","kind":"insurance","until":"2026-10-20"}]`) ||
		!strings.Contains(body, `"vehicles":1`) || !strings.Contains(body, `"published":true`) {
		t.Errorf("summary with documents running out: %s", body)
	}

	// An organizer finds the driver and asks for transport for a tour.
	_, body = c.do("POST", "/tours", tripTour, partner)
	tourID := obj(t, body)["id"].(string)
	c.do("POST", "/tours/"+tourID+"/join", "", guest)
	notifier.Tick(ctx)
	tg.take()
	if _, body := c.do("GET", "/crm/drivers?date=2026-10-17&seats=10", "", partner); !strings.Contains(body, `"userId":"`+driver) || !strings.Contains(body, `"busy":false`) {
		t.Fatalf("catalog: %s", body)
	}
	if _, body := c.do("GET", "/crm/drivers?seats=30", "", partner); body != "[]" {
		t.Errorf("no vehicle big enough: %s", body)
	}
	request := `{"driverId":"` + driver + `","startsAt":"2026-10-17T00:30:00Z","routeFrom":"Бишкек, Южные ворота","routeTo":"Ак-Суу","passengers":12,"price":15000}`
	if status, _ := c.do("POST", "/crm/tours/"+tourID+"/transport", request, owner); status != 404 {
		t.Errorf("transport for someone else's tour: %d", status)
	}
	status, body = c.do("POST", "/crm/tours/"+tourID+"/transport", request, partner)
	if status != 201 || !strings.Contains(body, `"status":"requested"`) || !strings.Contains(body, `"name":"Бакыт"`) {
		t.Fatalf("request: %d %s", status, body)
	}
	notifier.Tick(ctx)
	sent := tg.take()
	if len(sent) != 1 || sent[0].chat != 1005 || !strings.Contains(sent[0].text, "<b>User 3333</b> просит транспорт для тура «Ала-Куль &lt;тест&gt;»") ||
		sent[0].button == nil || !strings.HasPrefix(sent[0].button.URL, "https://ermekten.test/partner/trips/") {
		t.Fatalf("request notification: %+v", sent)
	}
	tripID := strings.TrimPrefix(sent[0].button.URL, "https://ermekten.test/partner/trips/")

	// The driver sees the group's size and the meeting point, and accepts.
	_, body = c.do("GET", "/driver/trips/"+tripID, "", driver)
	if !strings.Contains(body, `"meetingPoint":"Южные ворота"`) || !strings.Contains(body, `"participants":1`) ||
		!strings.Contains(body, `"account":{"id":"`+partner) {
		t.Errorf("trip: %s", body)
	}
	if status, _ := c.do("PUT", "/driver/trips/"+tripID+"/status", `{"status":"done"}`, driver); status != 409 {
		t.Errorf("a request done before it's accepted: %d", status)
	}
	if status, body := c.do("PUT", "/driver/trips/"+tripID+"/status", `{"status":"accepted","vehicleId":"`+vehicleID+`"}`, driver); status != 200 ||
		!strings.Contains(body, `"status":"accepted"`) || !strings.Contains(body, `"due":15000`) || !strings.Contains(body, `"model":"Mercedes Sprinter"`) {
		t.Fatalf("accept: %d %s", status, body)
	}
	notifier.Tick(ctx)
	sent = tg.take()
	if len(sent) != 1 || sent[0].chat != 1003 || !strings.Contains(sent[0].text, "Водитель <b>Бакыт</b> принял заявку") ||
		sent[0].button.URL != "https://ermekten.test/partner/tours/"+tourID+"/participants" {
		t.Fatalf("accepted notification: %+v", sent)
	}
	if _, body := c.do("GET", "/crm/tours/"+tourID+"/transport", "", partner); !strings.Contains(body, `"status":"accepted"`) || !strings.Contains(body, `"plate":"01KG123ABC"`) {
		t.Errorf("transport on the group page: %s", body)
	}
	if _, body := c.do("GET", "/crm/drivers?date=2026-10-17", "", partner); !strings.Contains(body, `"busy":true`) {
		t.Errorf("busy that day: %s", body)
	}

	// The organizer is now the driver's client, and they can chat.
	if _, body := c.do("GET", "/crm/clients", "", driver); !strings.Contains(body, `"userId":"`+partner) {
		t.Errorf("the organizer among the driver's clients: %s", body)
	}
	if status, _ := c.do("POST", "/chats", `{"partnerId":"`+driver+`"}`, partner); status != 200 {
		t.Errorf("organizer writing to a driver: %d", status)
	}

	// Payments, expenses, a trip of their own, the report.
	if status, body := c.do("POST", "/driver/trips/"+tripID+"/payments", `{"amount":10000,"method":"cash","paidOn":"2026-10-17"}`, driver); status != 201 || !strings.Contains(body, `"due":5000`) {
		t.Errorf("payment: %d %s", status, body)
	}
	if status, _ := c.do("POST", "/driver/expenses", `{"category":"fuel","amount":3000,"spentOn":"2026-10-17","vehicleId":"`+vehicleID+`","tripId":"`+tripID+`"}`, driver); status != 201 {
		t.Errorf("expense: %d", status)
	}
	if status, _ := c.do("POST", "/driver/expenses", `{"category":"fuel","amount":1,"spentOn":"2026-10-17","vehicleId":"`+tourID+`"}`, driver); status != 404 {
		t.Errorf("expense of someone else's vehicle: %d", status)
	}
	status, body = c.do("POST", "/driver/trips", `{"startsAt":"2026-10-20T03:00:00Z","endsAt":"2026-10-19T03:00:00Z","routeFrom":"Бишкек","routeTo":"Манас"}`, driver)
	if status != 400 {
		t.Errorf("a trip ending before it starts: %d %s", status, body)
	}
	status, body = c.do("POST", "/driver/trips", `{"startsAt":"2026-10-20T03:00:00Z","routeFrom":"Бишкек","routeTo":"Аэропорт Манас","price":2000,"passengers":3}`, driver)
	own := obj(t, body)
	if status != 201 || own["status"] != "accepted" || own["tour"] != nil {
		t.Fatalf("own trip: %d %s", status, body)
	}
	c.do("PUT", "/driver/trips/"+tripID+"/status", `{"status":"done"}`, driver)
	if _, body := c.do("GET", "/driver/reports?from=2026-10-01&to=2026-10-31", "", driver); !strings.Contains(body, `"trips":1`) || !strings.Contains(body, `"income":10000`) ||
		!strings.Contains(body, `"expenses":3000`) || !strings.Contains(body, `"profit":7000`) || !strings.Contains(body, `"fuel":3000`) ||
		!strings.Contains(body, `"passengers":12`) ||
		!strings.Contains(body, `"byVehicle":[{"id":"`+vehicleID+`","model":"Mercedes Sprinter","trips":1,"income":10000,"expenses":3000},{"id":null`) {
		t.Errorf("report: %s", body)
	}
	if _, body := c.do("GET", "/driver/summary", "", driver); !strings.Contains(body, `"due":7000`) || !strings.Contains(body, `"monthExpenses":3000`) {
		t.Errorf("summary: %s", body)
	}
	if _, body := c.do("GET", "/driver/trips?status=accepted", "", driver); strings.Count(body, `"routeFrom"`) != 1 {
		t.Errorf("filtered trips: %s", body)
	}

	// Organizers' requests are declined or cancelled, not deleted; the
	// organizer calls one off.
	if status, _ := c.do("DELETE", "/driver/trips/"+tripID, "", driver); status != 404 {
		t.Errorf("deleting an organizer's request: %d", status)
	}
	if status, _ := c.do("DELETE", "/driver/trips/"+own["id"].(string), "", driver); status != 200 {
		t.Errorf("deleting an own trip: %d", status)
	}
	c.do("POST", "/crm/tours/"+tourID+"/transport", request, partner)
	_, body = c.do("GET", "/driver/trips?status=requested", "", driver)
	second := strings.Split(strings.SplitN(body, `"id":"`, 2)[1], `"`)[0]
	if status, body := c.do("DELETE", "/crm/transport/"+second, "", partner); status != 200 || !strings.Contains(body, `"status":"cancelled"`) {
		t.Errorf("call off: %d %s", status, body)
	}
	notifier.Tick(ctx)
	if sent := tg.take(); len(sent) != 2 || !strings.Contains(sent[1].text, "Организатор отменил заявку") {
		t.Errorf("called-off notification: %+v", sent)
	}
	if status, _ := c.do("DELETE", "/driver/vehicles/"+vehicleID, "", driver); status != 200 {
		t.Errorf("delete vehicle: %d", status)
	}
}
