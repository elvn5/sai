package routes_test

import (
	"strings"
	"testing"

	"github.com/elvn5/sai/services/payments/internal/provider"
)

func TestPartnerFinances(t *testing.T) {
	e := setup(t, &provider.Fake{SiteURL: "https://ermekten.kg"})
	pay := func(qty int) {
		_, body := e.do("POST", "/orders", order(tourID, qty), tourist)
		id := obj(t, body)["id"].(string)
		e.do("POST", "/payments/test/"+id, `{"success":true}`, tourist)
		e.do("POST", "/orders/"+id+"/check", "", tourist)
	}

	if status, _ := e.do("GET", "/payouts/mine", "", tourist); status != 403 {
		t.Errorf("non-partner: %d", status)
	}
	if _, body := e.do("GET", "/payouts/mine", "", partner); body !=
		`{"owed":0,"owedOrders":0,"paidOut":0,"month":{"orders":0,"guests":0,"revenue":0,"fee":0,"earned":0},"payouts":[]}` {
		t.Errorf("empty: %s", body)
	}

	// 12 000 per guest, 10% commission.
	pay(2)
	if status, body := e.do("POST", "/payouts", `{"payeeId":"`+partner+`","note":"Наличными"}`, "admin"); status != 201 {
		t.Fatalf("payout: %d %s", status, body)
	}
	pay(1)
	_, body := e.do("GET", "/payouts/mine", "", partner)
	for _, want := range []string{
		`"owed":10800,"owedOrders":1,"paidOut":21600`,
		`"month":{"orders":2,"guests":3,"revenue":36000,"fee":3600,"earned":32400}`,
		`"amount":21600,"orders":1,"note":"Наличными"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("finances missing %s: %s", want, body)
		}
	}
}
