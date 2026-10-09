package routes_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

// signUp registers and verifies an email account, returning its session
// cookie and id.
func (e *env) signUp(email string) (*http.Cookie, string) {
	e.t.Helper()
	e.do("POST", "/auth/user/register", `{"email":"`+email+`","password":"secret123","firstName":"`+strings.Split(email, "@")[0]+`"}`)
	r := e.do("POST", "/auth/user/verify", `{"identifier":"`+email+`","code":"`+e.email.codes[email]+`"}`)
	session := r.cookie(e.t, "user_token")
	var me struct{ User struct{ ID string } }
	_ = json.Unmarshal([]byte(e.do("GET", "/auth/user/me", "", session).body), &me)
	return session, me.User.ID
}

func adminCookie(t *testing.T) *http.Cookie {
	t.Helper()
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"role": "admin"}).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: "admin_token", Value: tok}
}

func decode[T any](t *testing.T, body string) T {
	t.Helper()
	var v T
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		t.Fatalf("decode %s: %v", body, err)
	}
	return v
}

func TestAchievements(t *testing.T) {
	e := setup(t)
	admin := adminCookie(t)
	partner, partnerID := e.signUp("guide@x.kg")
	e.do("POST", "/partners/apply", "", partner)
	if r := e.do("POST", "/partners/"+partnerID+"/approve", "", admin); r.status != 201 {
		t.Fatalf("approve %q: %d %s", partnerID, r.status, r.body)
	}
	traveler, travelerID := e.signUp("aiza@x.kg")

	// The admin builds the catalog.
	if r := e.do("POST", "/achievements", `{"title":{"en":"Peak"},"icon":"trophy","rarity":"mythic"}`, admin); r.status != 400 ||
		r.body != `{"formErrors":[],"fieldErrors":{"icon":["Icon must be an icon name like i-mdi-trophy"],"rarity":["Invalid enum value. Expected 'common' | 'rare' | 'epic' | 'legendary', received 'mythic'"],"title":["Russian text is required"]}}` {
		t.Errorf("invalid achievement: %d %s", r.status, r.body)
	}
	if r := e.do("POST", "/achievements", `{"title":{"ru":"Пик"},"icon":"i-mdi-trophy"}`, partner); r.status != 401 {
		t.Errorf("partner creating catalog: %d", r.status)
	}
	r := e.do("POST", "/achievements", `{"title":{"ru":" Покоритель Ала-Куля ","en":"Ala-Kul conqueror","de":""},"description":{"ru":"Дошёл до озера"},"icon":"i-mdi-image-filter-hdr","rarity":"epic"}`, admin)
	if r.status != 201 || !strings.Contains(r.body, `"title":{"en":"Ala-Kul conqueror","ru":"Покоритель Ала-Куля"}`) || !strings.Contains(r.body, `"points":50`) {
		t.Fatalf("create: %d %s", r.status, r.body)
	}
	alaKul := decode[struct{ ID string }](t, r.body).ID
	legendID := decode[struct{ ID string }](t, e.do("POST", "/achievements", `{"title":{"ru":"Легенда"},"icon":"i-mdi-crown","rarity":"legendary","sortOrder":1}`, admin).body).ID
	archivedID := decode[struct{ ID string }](t, e.do("POST", "/achievements", `{"title":{"ru":"Старое"},"icon":"i-mdi-star","isActive":false}`, admin).body).ID

	if r := e.do("GET", "/achievements", ""); strings.Contains(r.body, archivedID) || !strings.Contains(r.body, legendID) {
		t.Errorf("public catalog shows archived or misses active: %s", r.body)
	}
	if r := e.do("GET", "/achievements/admin/all", "", admin); !strings.Contains(r.body, archivedID) {
		t.Errorf("admin catalog misses archived: %s", r.body)
	}

	// The traveler's code; the partner looks it up and awards.
	r = e.do("GET", "/achievements/me", "", traveler)
	mine := decode[struct {
		Code   string
		Points int
		Awards []any
	}](t, r.body)
	if len(mine.Code) != 8 || mine.Points != 0 || len(mine.Awards) != 0 {
		t.Fatalf("me before: %s", r.body)
	}
	if again := decode[struct{ Code string }](t, e.do("GET", "/achievements/me", "", traveler).body); again.Code != mine.Code {
		t.Errorf("code changed: %s → %s", mine.Code, again.Code)
	}
	typed := strings.ToLower(mine.Code[:4]) + "-" + mine.Code[4:]

	if r := e.do("GET", "/achievements/recipients/"+mine.Code, "", traveler); r.status != 403 {
		t.Errorf("non-partner lookup: %d", r.status)
	}
	if r := e.do("GET", "/achievements/recipients/NOPE2345", "", partner); r.status != 404 {
		t.Errorf("unknown code: %d", r.status)
	}
	r = e.do("GET", "/achievements/recipients/"+typed, "", partner)
	if r.status != 200 || !strings.Contains(r.body, `"firstName":"aiza"`) || !strings.Contains(r.body, `"earnedAchievementIds":[]`) || strings.Contains(r.body, "@") {
		t.Errorf("recipient: %d %s", r.status, r.body)
	}

	award := func(code, achievementID string) response {
		return e.do("POST", "/achievements/awards", `{"code":"`+code+`","achievementId":"`+achievementID+`","note":"Молодец!"}`, partner)
	}
	if r := award(typed, alaKul); r.status != 201 || !strings.Contains(r.body, `"awardedBy":{"id":"`+partnerID+`"`) || !strings.Contains(r.body, `"isNew":true`) {
		t.Fatalf("award: %d %s", r.status, r.body)
	}
	if r := award(mine.Code, alaKul); r.status != 409 {
		t.Errorf("award twice: %d %s", r.status, r.body)
	}
	if r := award(mine.Code, archivedID); r.status != 404 {
		t.Errorf("award archived: %d", r.status)
	}
	partnerCode := decode[struct{ Code string }](t, e.do("GET", "/achievements/me", "", partner).body).Code
	if r := award(partnerCode, alaKul); r.status != 400 {
		t.Errorf("self award: %d", r.status)
	}
	if r := award(mine.Code, legendID); r.status != 201 {
		t.Fatalf("award legend: %d %s", r.status, r.body)
	}
	if r := e.do("DELETE", "/achievements/"+alaKul, "", admin); r.status != 409 {
		t.Errorf("delete awarded: %d", r.status)
	}
	if r := e.do("DELETE", "/achievements/"+archivedID, "", admin); r.status != 200 {
		t.Errorf("delete unused: %d", r.status)
	}

	// The traveler sees them as new, once.
	r = e.do("GET", "/achievements/me", "", traveler)
	if !strings.Contains(r.body, `"points":150`) || strings.Count(r.body, `"isNew":true`) != 2 {
		t.Errorf("me after: %s", r.body)
	}
	e.do("POST", "/achievements/me/seen", "", traveler)
	if r := e.do("GET", "/achievements/me", "", traveler); strings.Contains(r.body, `"isNew":true`) {
		t.Errorf("still new after seen: %s", r.body)
	}

	// Hall of fame: ranked, public fields only, no notes.
	r = e.do("GET", "/hall-of-fame", "")
	hall := decode[struct {
		Leaders []struct {
			Rank         int
			Points       int
			Count        int
			User         struct{ ID string }
			Achievements []struct{ Rarity string }
		}
		Recent []map[string]any
	}](t, r.body)
	if len(hall.Leaders) != 1 || hall.Leaders[0].User.ID != travelerID || hall.Leaders[0].Points != 150 ||
		hall.Leaders[0].Rank != 1 || hall.Leaders[0].Achievements[0].Rarity != "legendary" || len(hall.Recent) != 2 {
		t.Errorf("hall: %s", r.body)
	}
	if strings.Contains(r.body, "Молодец") || strings.Contains(r.body, "@") || strings.Contains(r.body, mine.Code) {
		t.Errorf("hall leaks private fields: %s", r.body)
	}

	if r := e.do("GET", "/achievements/awards/given", "", partner); strings.Count(r.body, `"id":"`+travelerID+`"`) != 2 {
		t.Errorf("given: %s", r.body)
	}
	awardID := decode[[]struct{ ID string }](t, e.do("GET", "/achievements/admin/awards", "", admin).body)[0].ID
	if r := e.do("DELETE", "/achievements/awards/"+awardID, "", admin); r.status != 200 {
		t.Errorf("revoke: %d", r.status)
	}
	if r := e.do("GET", "/achievements/me", "", traveler); strings.Count(r.body, `"achievement":`) != 1 {
		t.Errorf("after revoke: %s", r.body)
	}
	if r := e.do("PUT", "/achievements/"+legendID, `{"isActive":false,"title":{"ru":"Легенда гор"}}`, admin); r.status != 200 || !strings.Contains(r.body, `"isActive":false`) || !strings.Contains(r.body, "Легенда гор") {
		t.Errorf("update: %d %s", r.status, r.body)
	}
}
