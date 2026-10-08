package validation_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/pkg/validation"
)

type uploadDTO struct {
	Filename    *string `json:"filename" binding:"required,min=1,max=255"`
	ContentType *string `json:"contentType" binding:"required,oneof=image/jpeg image/png image/webp"`
	Folder      *string `json:"folder" binding:"omitempty,max=100,testfolder"`
}

func init() {
	validation.RegisterPattern("testfolder", regexp.MustCompile(`^[a-z0-9-]+$`),
		"folder must be lowercase alphanumeric with hyphens")
}

func bind(t *testing.T, contentType, body string) (int, map[string]any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", contentType)
	var dto uploadDTO
	if validation.BindJSON(c, &dto) {
		return http.StatusOK, nil
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("bad JSON %q: %v", rec.Body.String(), err)
	}
	return rec.Code, out
}

// Expected bodies are what the NestJS ZodValidationPipe returned for the
// same input.
func TestZodCompatibleErrors(t *testing.T) {
	cases := map[string]struct {
		contentType, body string
		want              string
	}{
		"missing fields": {"application/json", `{}`,
			`{"formErrors":[],"fieldErrors":{"contentType":["Required"],"filename":["Required"]}}`},
		"empty body": {"application/json", ``,
			`{"formErrors":[],"fieldErrors":{"contentType":["Required"],"filename":["Required"]}}`},
		"not json": {"text/plain", `hello`,
			`{"formErrors":[],"fieldErrors":{"contentType":["Required"],"filename":["Required"]}}`},
		"bad enum and empty string": {"application/json", `{"filename":"","contentType":"image/gif"}`,
			`{"formErrors":[],"fieldErrors":{"contentType":["Invalid enum value. Expected 'image/jpeg' | 'image/png' | 'image/webp', received 'image/gif'"],"filename":["String must contain at least 1 character(s)"]}}`},
		"wrong type": {"application/json", `{"filename":42,"contentType":"image/png"}`,
			`{"formErrors":[],"fieldErrors":{"filename":["Expected string, received number"]}}`},
		"custom pattern": {"application/json", `{"filename":"a.png","contentType":"image/png","folder":"Bad Folder"}`,
			`{"formErrors":[],"fieldErrors":{"folder":["folder must be lowercase alphanumeric with hyphens"]}}`},
		"array body": {"application/json", `[]`,
			`{"formErrors":["Expected object, received array"],"fieldErrors":{}}`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			status, got := bind(t, tc.contentType, tc.body)
			var want map[string]any
			_ = json.Unmarshal([]byte(tc.want), &want)
			if status != http.StatusBadRequest || !reflect.DeepEqual(got, want) {
				t.Fatalf("got %d %v\nwant 400 %v", status, got, want)
			}
		})
	}
}

func TestValidBody(t *testing.T) {
	status, _ := bind(t, "application/json", `{"filename":"a.png","contentType":"image/png","folder":"gallery"}`)
	if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
}
