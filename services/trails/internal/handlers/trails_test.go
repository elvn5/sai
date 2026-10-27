package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/trails/internal/models"
)

func TestValidTrailExtras(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pdf, wrong := "/uploads/check.pdf", "/uploads/track.gpx"
	for _, tc := range []struct {
		name  string
		trail models.Trail
		want  bool
	}{
		{"empty", models.Trail{}, true},
		{"pdf", models.Trail{ChecklistURL: &pdf}, true},
		{"wrong file", models.Trail{ChecklistURL: &wrong}, false},
		{"missing point", models.Trail{Markers: []models.TrailMarker{{PointIndex: 0, Label: "A"}}}, false},
		{"valid point", models.Trail{Track: make([]models.TrackPoint, 1), Markers: []models.TrailMarker{{PointIndex: 0, Label: "A"}}}, true},
		{"blank label", models.Trail{Track: make([]models.TrackPoint, 1), Markers: []models.TrailMarker{{PointIndex: 0, Label: " "}}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			if got := validTrailExtras(c, tc.trail); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
