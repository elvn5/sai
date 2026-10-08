package sanitize_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/elvn5/sai/services/pkg/sanitize"
)

// testdata/golden.json is sanitize-html's output (the NestJS config) for
// each input; regenerate it with Node if that config changes.
func TestMatchesSanitizeHTML(t *testing.T) {
	data, err := os.ReadFile("testdata/golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct{ Input, Want string }
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		if got := sanitize.HTML(tc.Input); got != tc.Want {
			t.Errorf("HTML(%q)\n got: %q\nwant: %q", tc.Input, got, tc.Want)
		}
	}
}
