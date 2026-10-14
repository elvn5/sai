package routes_test

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

// event posts a stats event as a browser at ip (with an optional session).
func (c client) event(targetType, id, event, ip, userAgent, userID string) int {
	c.t.Helper()
	body := `{"targetType":"` + targetType + `","targetId":"` + id + `","event":"` + event + `"}`
	if userID != "" {
		status, _ := c.do("POST", "/listing-stats/events", body, userID)
		return status
	}
	req := httptest.NewRequest("POST", "/listing-stats/events", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", ip)
	req.Header.Set("User-Agent", userAgent)
	rec := httptest.NewRecorder()
	c.router.ServeHTTP(rec, req)
	return rec.Code
}

type statsReport struct {
	Totals   map[string]int `json:"totals"`
	Previous map[string]int `json:"previous"`
	Daily    []struct {
		Day      string `json:"day"`
		Views    int    `json:"views"`
		Contacts int    `json:"contacts"`
	} `json:"daily"`
	Listings []struct {
		Listing struct {
			TargetType string `json:"targetType"`
			ID         string `json:"id"`
			Title      string `json:"title"`
		} `json:"listing"`
		Views    int `json:"views"`
		Contacts int `json:"contacts"`
		Whatsapp int `json:"whatsapp"`
	} `json:"listings"`
}

func TestListingStats(t *testing.T) {
	c := setup(t)
	hotel := createHotel(c, "stats-inn", "")
	quiet := createHotel(c, "quiet-inn", "")
	const browser = "Mozilla/5.0 (iPhone)"

	for _, ip := range []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"} {
		if status := c.event("hotel", hotel, "view", ip, browser, ""); status != 204 {
			t.Fatalf("view: %d", status)
		}
	}
	// Repeats the same day, crawlers and the owner's own visits don't count.
	c.event("hotel", hotel, "view", "10.0.0.1", browser, "")
	c.event("hotel", hotel, "view", "10.0.0.9", "Googlebot/2.1", "")
	c.event("hotel", hotel, "view", "", "", owner)
	c.event("hotel", hotel, "whatsapp", "10.0.0.1", browser, "")
	c.event("hotel", hotel, "phone", "10.0.0.2", browser, "")
	c.event("hotel", hotel, "phone", "10.0.0.2", browser, "")
	// Unknown listings are ignored but still answered with 204.
	if status := c.event("tour", hotel, "view", "10.0.0.1", browser, ""); status != 204 {
		t.Errorf("unknown listing: %d", status)
	}
	if status, body := c.do("POST", "/listing-stats/events", `{"targetType":"hotel","targetId":"`+hotel+`","event":"hack"}`, ""); status != 400 ||
		!strings.Contains(body, `"event"`) {
		t.Errorf("bad event: %d %s", status, body)
	}

	if status, _ := c.do("GET", "/listing-stats/mine", "", regular); status != 403 {
		t.Errorf("non-partner stats: %d", status)
	}
	status, body := c.do("GET", "/listing-stats/mine?days=7", "", owner)
	if status != 200 {
		t.Fatalf("stats: %d %s", status, body)
	}
	var r statsReport
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		t.Fatal(err)
	}
	if r.Totals["views"] != 3 || r.Totals["contacts"] != 2 || r.Totals["phone"] != 1 || r.Totals["whatsapp"] != 1 {
		t.Errorf("totals: %v", r.Totals)
	}
	if len(r.Daily) != 7 || r.Daily[6].Views != 3 || r.Daily[6].Contacts != 2 || r.Daily[0].Views != 0 {
		t.Errorf("daily: %+v", r.Daily)
	}
	if len(r.Listings) != 2 {
		t.Fatalf("listings: %+v", r.Listings)
	}
	for _, l := range r.Listings {
		switch l.Listing.ID {
		case hotel:
			if l.Views != 3 || l.Whatsapp != 1 || l.Contacts != 2 || l.Listing.Title != "stats-inn Inn" {
				t.Errorf("hotel stats: %+v", l)
			}
		case quiet:
			if l.Views != 0 {
				t.Errorf("quiet stats: %+v", l)
			}
		}
	}
	// Another partner sees only their own (none).
	if _, body := c.do("GET", "/listing-stats/mine", "", partner); !strings.Contains(body, `"listings":[]`) || !strings.Contains(body, `"views":0`) {
		t.Errorf("other partner: %s", body)
	}
}

func TestReviewReplies(t *testing.T) {
	c := setup(t)
	hotel := createHotel(c, "reply-inn", "")
	_, body := c.do("POST", "/reviews", `{"targetType":"hotel","targetId":"`+hotel+`","rating":3,"text":"Fine, but the shower was cold"}`, guest)
	review := obj(t, body)["id"].(string)

	if status, body := c.do("PUT", "/reviews/"+review+"/reply", `{"text":"Sorry!"}`, partner); status != 403 ||
		!strings.Contains(body, "Only the listing owner can reply to its reviews") {
		t.Errorf("non-owner reply: %d %s", status, body)
	}
	if status, body := c.do("PUT", "/reviews/"+review+"/reply", `{"text":"   "}`, owner); status != 400 || !strings.Contains(body, `"text"`) {
		t.Errorf("empty reply: %d %s", status, body)
	}

	if _, body := c.do("GET", "/reviews/mine?unanswered=true", "", owner); !strings.Contains(body, `"total":1`) ||
		!strings.Contains(body, `"listing":{"targetType":"hotel","id":"`+hotel+`","title":"reply-inn Inn"`) ||
		!strings.Contains(body, `"firstName":"User 2222"`) {
		t.Errorf("unanswered: %s", body)
	}
	status, body := c.do("PUT", "/reviews/"+review+"/reply", `{"text":"  Fixed the boiler, come back!  "}`, owner)
	if status != 200 || !strings.Contains(body, `"reply":{"text":"Fixed the boiler, come back!","repliedAt":`) {
		t.Fatalf("reply: %d %s", status, body)
	}
	if _, body := c.do("GET", "/reviews?targetType=hotel&targetId="+hotel, "", ""); !strings.Contains(body, `"text":"Fixed the boiler, come back!"`) {
		t.Errorf("public reply: %s", body)
	}
	if _, body := c.do("GET", "/reviews/mine?unanswered=true", "", owner); !strings.Contains(body, `"total":0`) {
		t.Errorf("unanswered after reply: %s", body)
	}
	if _, body := c.do("GET", "/reviews/mine", "", owner); !strings.Contains(body, `"total":1`) {
		t.Errorf("all reviews: %s", body)
	}
	if status, body := c.do("DELETE", "/reviews/"+review+"/reply", "", owner); status != 200 || !strings.Contains(body, `"reply":null`) {
		t.Errorf("delete reply: %d %s", status, body)
	}
	if status, _ := c.do("GET", "/reviews/mine", "", regular); status != 403 {
		t.Errorf("non-partner reviews: %d", status)
	}
}
