package nodepath_test

import (
	"testing"

	"github.com/elvn5/sai/services/media/internal/nodepath"
)

func TestExtname(t *testing.T) {
	cases := map[string]string{"a.png": ".png", "a.tar.gz": ".gz", ".env": "", "noext": "", "dir/x.WEBP": ".WEBP", "file.": "."}
	for in, want := range cases {
		if got := nodepath.Extname(in); got != want {
			t.Errorf("Extname(%q) = %q, want %q", in, got, want)
		}
	}
}
