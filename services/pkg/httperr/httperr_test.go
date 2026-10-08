package httperr_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/pkg/httperr"
)

func TestAbortWritesNestShape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	httperr.Abort(c, http.StatusNotFound, "Hotel not found")

	want := `{"message":"Hotel not found","error":"Not Found","statusCode":404}`
	if rec.Code != http.StatusNotFound || rec.Body.String() != want {
		t.Fatalf("got %d %s, want 404 %s", rec.Code, rec.Body.String(), want)
	}
}
