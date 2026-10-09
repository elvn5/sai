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

type attachment struct {
	URL  string `json:"url" binding:"min=1,max=2000"`
	Size int64  `json:"size" binding:"gte=0"`
}

type guideDTO struct {
	Excerpt     validation.Field[string] `json:"excerpt" binding:"omitempty,max=5"`
	Equipment   []string                 `json:"equipmentChecklist" binding:"omitempty,max=2,dive,min=1"`
	Attachments []attachment             `json:"attachments" binding:"omitempty,dive"`
}

func bindGuide(t *testing.T, body string) (bool, guideDTO, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	var dto guideDTO
	ok := validation.BindJSON(c, &dto)
	return ok, dto, rec.Body.String()
}

func TestNullableFieldStates(t *testing.T) {
	_, absent, _ := bindGuide(t, `{}`)
	_, null, _ := bindGuide(t, `{"excerpt":null}`)
	_, value, _ := bindGuide(t, `{"excerpt":"hi"}`)
	if absent.Excerpt.Set || !null.Excerpt.Set || null.Excerpt.Valid || !value.Excerpt.Valid || *value.Excerpt.Ptr() != "hi" {
		t.Fatalf("absent=%+v null=%+v value=%+v", absent.Excerpt, null.Excerpt, value.Excerpt)
	}
	if ok, _, body := bindGuide(t, `{"excerpt":"too long"}`); ok || body != `{"formErrors":[],"fieldErrors":{"excerpt":["String must contain at most 5 character(s)"]}}` {
		t.Fatalf("max on nullable: %v %s", ok, body)
	}
	if ok, _, body := bindGuide(t, `{"excerpt":5}`); ok || body != `{"formErrors":[],"fieldErrors":{"excerpt":["Expected string, received number"]}}` {
		t.Fatalf("type on nullable: %v %s", ok, body)
	}
}

// zod's flatten() files nested issues under the top-level property.
func TestNestedErrorsReportTopLevelField(t *testing.T) {
	cases := map[string]string{
		`{"equipmentChecklist":["a",""]}`:          `{"formErrors":[],"fieldErrors":{"equipmentChecklist":["String must contain at least 1 character(s)"]}}`,
		`{"equipmentChecklist":["a","b","c"]}`:     `{"formErrors":[],"fieldErrors":{"equipmentChecklist":["Array must contain at most 2 element(s)"]}}`,
		`{"attachments":[{"url":"","size":-1}]}`:   `{"formErrors":[],"fieldErrors":{"attachments":["String must contain at least 1 character(s)","Number must be greater than or equal to 0"]}}`,
		`{"attachments":[{"url":"x","size":1.5}]}`: `{"formErrors":[],"fieldErrors":{"attachments":["Expected integer, received float"]}}`,
	}
	for body, want := range cases {
		if ok, _, got := bindGuide(t, body); ok || got != want {
			t.Errorf("%s\n got: %v %s\nwant: %s", body, ok, got, want)
		}
	}
}

type ratingDTO struct {
	Stars validation.Field[int]    `json:"stars" binding:"omitempty,gte=1,lte=5"`
	Note  validation.Field[string] `json:"note" binding:"omitempty,min=1"`
}

// Zero values of nullable fields must still be validated.
func TestNullableZeroValuesAreValidated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for body, want := range map[string]string{
		`{"stars":0}`:                `{"formErrors":[],"fieldErrors":{"stars":["Number must be greater than or equal to 1"]}}`,
		`{"note":""}`:                `{"formErrors":[],"fieldErrors":{"note":["String must contain at least 1 character(s)"]}}`,
		`{"stars":null,"note":null}`: ``,
	} {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		var dto ratingDTO
		validation.BindJSON(c, &dto)
		if rec.Body.String() != want {
			t.Errorf("%s\n got: %s\nwant: %s", body, rec.Body.String(), want)
		}
	}
}

type multiDTO struct {
	Price validation.Field[int] `json:"price" binding:"omitempty,gte=0"`
	Name  *validation.Trimmed   `json:"name" binding:"required,min=1"`
	Stars validation.Field[int] `json:"stars" binding:"omitempty,gte=1"`
	Tags  []string              `json:"tags" binding:"omitempty,dive,oneof=a b"`
}

// Every mistyped property is reported, and properties after one are still
// decoded and validated (encoding/json alone would stop at the first).
func TestAllTypeErrorsReported(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"price":1.5,"name":5,"stars":0,"tags":["z"],"Price":3}`))
	c.Request.Header.Set("Content-Type", "application/json")
	var dto multiDTO
	validation.BindJSON(c, &dto)
	want := `{"formErrors":[],"fieldErrors":{"name":["Expected string, received number"],"price":["Expected integer, received float"],"stars":["Number must be greater than or equal to 1"],"tags":["Invalid enum value. Expected 'a' | 'b', received 'z'"]}}`
	if rec.Body.String() != want {
		t.Fatalf("got  %s\nwant %s", rec.Body.String(), want)
	}
}

type numberDTO struct {
	ID *json.Number `json:"id" binding:"required"`
}

func TestJSONNumberTypeMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"id":"x"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	var dto numberDTO
	validation.BindJSON(c, &dto)
	if want := `{"formErrors":[],"fieldErrors":{"id":["Expected number, received string"]}}`; rec.Body.String() != want {
		t.Fatalf("got %s", rec.Body.String())
	}
}
