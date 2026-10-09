package routes_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/listings/internal/handlers"
	"github.com/elvn5/sai/services/listings/internal/purge"
	"github.com/elvn5/sai/services/listings/internal/routes"
	"github.com/elvn5/sai/services/listings/internal/store"
	"github.com/elvn5/sai/services/pkg/partnerpurge"
)

func TestOnlyPartnersList(t *testing.T) {
	c := setup(t)
	for path, body := range map[string]string{
		"/hotels":       `{"slug":"x","name":"X","type":"hotel","description":"d","destination":"karakol","cancellationPolicy":"free"}`,
		"/local-guides": `{"slug":"x","name":"X","bio":"b","languages":["ru"],"specializations":["trekking"]}`,
	} {
		if status, resp := c.do("POST", path, body, regular); status != 403 || !strings.Contains(resp, "Only approved partners can do this") {
			t.Errorf("non-partner POST %s: %d %s", path, status, resp)
		}
	}
}

func TestPurgeNonPartnerListings(t *testing.T) {
	db, _ := createDatabase(t, true)
	ctx := context.Background()
	const (
		partnerHotel = "aaaaaaaa-0000-0000-0000-000000000001"
		regularHotel = "aaaaaaaa-0000-0000-0000-000000000002"
		regularGuide = "bbbbbbbb-0000-0000-0000-000000000001"
	)
	for _, q := range []string{
		`INSERT INTO hotels (id, slug, name, type, description, destination, cancellation_policy, is_published, user_id) VALUES
			('` + partnerHotel + `', 'partner-inn', 'P', 'hotel', 'd', 'karakol', 'free', true, '` + partner + `'),
			('` + regularHotel + `', 'regular-inn', 'R', 'hotel', 'd', 'karakol', 'free', true, '` + regular + `')`,
		`INSERT INTO local_guides (id, slug, name, bio, languages, specializations, is_published, user_id) VALUES
			('` + regularGuide + `', 'regular-guide', 'R', 'b', '{ru}', '{trekking}', true, '` + regular + `')`,
		`INSERT INTO reviews (target_type, target_id, user_id, rating, text) VALUES
			('hotel', '` + regularHotel + `', '` + guest + `', 5, 'great'),
			('hotel', '` + partnerHotel + `', '` + guest + `', 4, 'good')`,
	} {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}

	log := slog.New(slog.DiscardHandler)
	partnerpurge.Run(ctx, db, testUsers, purge.Service(), log)

	gin.SetMode(gin.TestMode)
	c := client{t, routes.Setup(handlers.Deps{Store: store.New(db), Authors: testUsers, Log: log}, testUsers, secret)}
	if status, _ := c.do("GET", "/hotels/partner-inn", "", ""); status != 200 {
		t.Errorf("partner hotel removed: %d", status)
	}
	if status, _ := c.do("GET", "/hotels/regular-inn", "", ""); status != 404 {
		t.Errorf("regular hotel kept: %d", status)
	}
	if status, _ := c.do("GET", "/local-guides/regular-guide", "", ""); status != 404 {
		t.Errorf("regular guide kept: %d", status)
	}
	var reviews int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM reviews`).Scan(&reviews); err != nil || reviews != 1 {
		t.Errorf("reviews left: %d %v", reviews, err)
	}
}
