package jsontime_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/elvn5/sai/services/pkg/jsontime"
)

func TestMatchesJavaScriptToJSON(t *testing.T) {
	loc := time.FixedZone("Bishkek", 6*3600)
	cases := map[time.Time]string{
		time.Date(2026, 10, 8, 14, 15, 26, 64_123_456, loc): `"2026-10-08T08:15:26.064Z"`,
		time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC):         `"2026-01-02T03:04:05.000Z"`,
	}
	for in, want := range cases {
		got, _ := json.Marshal(jsontime.New(in))
		if string(got) != want {
			t.Errorf("got %s, want %s", got, want)
		}
	}
	null, _ := json.Marshal(struct {
		At *jsontime.Time `json:"at"`
	}{jsontime.Ptr(nil)})
	if string(null) != `{"at":null}` {
		t.Errorf("nil = %s", null)
	}
}
