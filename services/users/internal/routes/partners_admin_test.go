package routes_test

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPartnerCRUDAndHistory(t *testing.T) {
	e := setup(t)
	admin := adminCookie(t)
	e.do("POST", "/auth/user/register", `{"email":"d@x.kg","password":"secret123"}`)
	r := e.do("POST", "/auth/user/verify", `{"identifier":"d@x.kg","code":"`+e.email.codes["d@x.kg"]+`"}`)
	user := r.cookie(t, "user_token")
	var me struct{ User struct{ ID string } }
	_ = json.Unmarshal([]byte(e.do("GET", "/auth/user/me", "", user).body), &me)
	id := me.User.ID

	if r := e.do("GET", "/partners", "", user); r.status != 401 {
		t.Errorf("partners as a user: %d", r.status)
	}
	if r := e.do("GET", "/partners", "", admin); r.body != `[]` {
		t.Errorf("no partners yet: %s", r.body)
	}
	if r := e.do("PUT", "/partners/"+id, `{"kind":"driver"}`, admin); r.status != 404 {
		t.Errorf("edit a non-partner: %d", r.status)
	}
	if r := e.do("POST", "/partners", `{"userId":"`+id+`","kind":"bus"}`, admin); r.status != 400 {
		t.Errorf("create with an unknown kind: %d", r.status)
	}
	if r := e.do("POST", "/partners", `{"userId":"`+id+`","kind":"driver"}`, admin); r.status != 201 ||
		!strings.Contains(r.body, `"partnerStatus":"approved"`) || !strings.Contains(r.body, `"partnerKind":"driver"`) {
		t.Fatalf("create: %d %s", r.status, r.body)
	}
	if r := e.do("POST", "/partners", `{"userId":"`+id+`"}`, admin); r.status != 409 {
		t.Errorf("create twice: %d", r.status)
	}
	r = e.do("PUT", "/partners/"+id, `{"kind":"organizer","status":"rejected","firstName":"Азамат","password":"x"}`, admin)
	if r.status != 200 || !strings.Contains(r.body, `"partnerStatus":"rejected"`) || !strings.Contains(r.body, `"firstName":"Азамат"`) {
		t.Fatalf("update: %d %s", r.status, r.body)
	}
	if r := e.do("GET", "/partners?status=approved", "", admin); r.body != `[]` {
		t.Errorf("approved only: %s", r.body)
	}
	if r := e.do("GET", "/partners?status=rejected", "", admin); !strings.Contains(r.body, id) {
		t.Errorf("rejected: %s", r.body)
	}
	if r := e.do("DELETE", "/partners/"+id, "", admin); r.status != 200 || !strings.Contains(r.body, `"partnerStatus":"none"`) {
		t.Errorf("delete: %d %s", r.status, r.body)
	}
	if r := e.do("GET", "/partners", "", admin); r.body != `[]` {
		t.Errorf("after delete: %s", r.body)
	}

	// Every successful admin change is in the partners' history, newest
	// first, with secrets hidden; failed ones and the user's own aren't.
	if r := e.do("GET", "/partners/admin/history", "", user); r.status != 401 {
		t.Errorf("history as a user: %d", r.status)
	}
	var history []struct {
		Method, Route string
		EntityID      *string
		Subject       *string
		Payload       map[string]any
		CreatedAt     string
	}
	_ = json.Unmarshal([]byte(e.do("GET", "/partners/admin/history", "", admin).body), &history)
	if len(history) != 3 || history[0].Method != "DELETE" || history[1].Route != "/partners/:userId" ||
		history[2].Route != "/partners" || *history[2].EntityID != id || history[1].Payload["password"] != "***" ||
		history[0].Subject == nil || *history[0].Subject != "Азамат" {
		t.Fatalf("history: %+v", history)
	}
	var page []any
	_ = json.Unmarshal([]byte(e.do("GET", "/partners/admin/history?limit=1&before="+history[0].CreatedAt, "", admin).body), &page)
	if len(page) != 1 {
		t.Errorf("next page: %v", page)
	}
	if r := e.do("GET", "/users/admin/history", "", admin); r.body != `[]` {
		t.Errorf("users module history: %s", r.body)
	}
	if r := e.do("GET", "/partners/admin/history?before=yesterday", "", admin); r.status != 400 {
		t.Errorf("bad cursor: %d", r.status)
	}
}
