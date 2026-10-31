package api_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/elvn5/sai/services/bot/internal/api"
)

func TestProfileCalls(t *testing.T) {
	ctx := context.Background()
	c, got := newClient(t, http.StatusOK, `{"user":{"id":"u1","email":"a@b.kg","partnerStatus":"none","partnerKind":"organizer","hasPassword":true}}`)
	me, err := c.Me(ctx, "u1")
	if err != nil || me.ID != "u1" || !me.HasPassword || *me.Email != "a@b.kg" {
		t.Fatalf("Me = %+v, %v", me, err)
	}
	if _, err := c.ApplyAsPartner(ctx, "u1", api.KindDriver); err != nil {
		t.Fatal(err)
	}
	current := "old-secret"
	if err := c.ChangePassword(ctx, "u1", &current, "new-secret"); err != nil {
		t.Fatal(err)
	}
	if err := c.ChangePassword(ctx, "u1", nil, "first-secret"); err != nil {
		t.Fatal(err)
	}
	health := "Астма"
	if err := c.SaveParticipantForm(ctx, "u1", api.ParticipantForm{EmergencyName: "Мама", EmergencyPhone: "+996 555 000111",
		Experience: "hiking", Health: &health}); err != nil {
		t.Fatal(err)
	}
	want := []request{
		{http.MethodGet, "/auth/user/me", "", "u1"},
		{http.MethodPost, "/partners/apply", `{"kind":"driver"}`, "u1"},
		{http.MethodPost, "/auth/user/password/change", `{"currentPassword":"old-secret","password":"new-secret"}`, "u1"},
		{http.MethodPost, "/auth/user/password/change", `{"password":"first-secret"}`, "u1"},
		{http.MethodPut, "/tours/participant-form",
			`{"form":{"emergencyName":"Мама","emergencyPhone":"+996 555 000111","experience":"hiking","health":"Астма","notes":null}}`, "u1"},
	}
	for i, w := range want {
		if (*got)[i] != w {
			t.Errorf("request %d: %+v, want %+v", i, (*got)[i], w)
		}
	}
}

func TestProfileReads(t *testing.T) {
	ctx := context.Background()
	c, got := newClient(t, http.StatusOK, `{"form":null,"updatedAt":null}`)
	if form, err := c.ParticipantForm(ctx, "u1"); err != nil || form != nil {
		t.Fatalf("no form = %+v, %v", form, err)
	}
	c, _ = newClient(t, http.StatusOK, `{"code":"AB12CD34","points":50,"awards":[{"id":"w1","achievement":{"id":"a1","title":{"ru":"Вершина"},"points":50},"awardedAt":"2026-10-25T18:29:23.017Z"}]}`)
	mine, err := c.MyAchievements(ctx, "u1")
	if err != nil || mine.Code != "AB12CD34" || len(mine.Awards) != 1 || mine.Awards[0].Achievement.Title["ru"] != "Вершина" {
		t.Fatalf("MyAchievements = %+v, %v", mine, err)
	}
	c, hall := newClient(t, http.StatusOK, `{"leaders":[{"rank":1,"user":{"id":"u7","firstName":"Айбек"},"points":300,"count":4}],"recent":[]}`)
	leaders, err := c.HallOfFame(ctx)
	if err != nil || len(leaders) != 1 || leaders[0].User.Name() != "Айбек" || (*hall)[0].userID != "" {
		t.Fatalf("HallOfFame = %+v, %v", leaders, err)
	}
	if r := (*got)[0]; r.uri != "/tours/participant-form" || r.userID != "u1" {
		t.Errorf("form request: %+v", r)
	}

	// A wrong current password is a 400 without field errors.
	c, _ = newClient(t, http.StatusBadRequest, `{"statusCode":400,"message":"Current password is incorrect"}`)
	err = c.ChangePassword(ctx, "u1", nil, "x")
	if api.StatusOf(err) != http.StatusBadRequest || api.FieldsOf(err) != nil {
		t.Errorf("wrong password: %v", err)
	}
}
