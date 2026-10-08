package validation_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/pkg/validation"
)

func runQuery(t *testing.T, rawQuery string, read func(q *validation.Query)) (bool, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/?"+rawQuery, nil)
	q := validation.NewQuery(c)
	read(q)
	return q.OK(), rec.Body.String()
}

// Expected bodies match the NestJS zod schemas' output.
func TestQueryCoercionAndErrors(t *testing.T) {
	read := func(q *validation.Query) {
		q.IntDefault("limit", 20, validation.Bound(1), validation.Bound(50))
		q.Number("minRating", validation.NumberRule{Min: validation.Bound(0), Max: validation.Bound(5)})
		q.List("amenities", []string{"wifi", "pool"})
		q.Bool("freeCancellation")
		q.RequiredEnum("locale", []string{"ru", "en"})
	}
	cases := map[string]string{
		"locale=ru&limit=abc":            `{"formErrors":[],"fieldErrors":{"limit":["Expected number, received nan"]}}`,
		"locale=ru&limit=2.5":            `{"formErrors":[],"fieldErrors":{"limit":["Expected integer, received float"]}}`,
		"locale=ru&limit=":               `{"formErrors":[],"fieldErrors":{"limit":["Number must be greater than or equal to 1"]}}`,
		"locale=ru&limit=51&minRating=6": `{"formErrors":[],"fieldErrors":{"limit":["Number must be less than or equal to 50"],"minRating":["Number must be less than or equal to 5"]}}`,
		"locale=ru&amenities=wifi,spa":   `{"formErrors":[],"fieldErrors":{"amenities":["Invalid enum value. Expected 'wifi' | 'pool', received 'spa'"]}}`,
		"locale=ru&freeCancellation=yes": `{"formErrors":[],"fieldErrors":{"freeCancellation":["Invalid enum value. Expected 'true' | 'false', received 'yes'"]}}`,
		"limit=10":                       `{"formErrors":[],"fieldErrors":{"locale":["Required"]}}`,
	}
	for query, want := range cases {
		if ok, body := runQuery(t, query, read); ok || body != want {
			t.Errorf("%s\n got: %v %s\nwant: %s", query, ok, body, want)
		}
	}
	for _, query := range []string{"locale=ru", "locale=en&limit=0x10&minRating=4.5&amenities=wifi,,pool&freeCancellation=true", "locale=ru&limit=%207%20"} {
		if ok, body := runQuery(t, query, read); !ok {
			t.Errorf("%s rejected: %s", query, body)
		}
	}
}
