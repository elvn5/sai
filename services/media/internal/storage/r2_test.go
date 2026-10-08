package storage

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/elvn5/sai/services/media/internal/config"
)

func fullConfig() config.Config {
	return config.Config{
		R2AccountID: "acc123", R2AccessKeyID: "key", R2SecretAccessKey: "secret",
		R2BucketName: "photos", R2PublicURL: "https://cdn.example.com/",
	}
}

func TestPresignUpload(t *testing.T) {
	up, err := NewR2(fullConfig()).PresignUpload(context.Background(), "Mountain.JPG", "image/jpeg", "gallery")
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^gallery/[0-9a-f-]{36}\.jpg$`).MatchString(up.Key) {
		t.Errorf("key = %q", up.Key)
	}
	if up.PublicURL != "https://cdn.example.com/"+up.Key {
		t.Errorf("publicUrl = %q", up.PublicURL)
	}
	u, _ := url.Parse(up.UploadURL)
	if u.Host != "photos.acc123.r2.cloudflarestorage.com" || u.Path != "/"+up.Key {
		t.Errorf("uploadUrl = %q", up.UploadURL)
	}
	q := u.Query()
	if q.Get("X-Amz-Expires") != "600" || !strings.Contains(q.Get("X-Amz-SignedHeaders"), "content-type") {
		t.Errorf("presign params = %v", q)
	}
}

func TestPresignErrorsMatchNest(t *testing.T) {
	noBucket := fullConfig()
	noBucket.R2BucketName = ""
	if _, err := NewR2(noBucket).PresignUpload(context.Background(), "a.png", "image/png", ""); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("no bucket: %v", err)
	}

	noCreds := fullConfig()
	noCreds.R2SecretAccessKey = ""
	if _, err := NewR2(noCreds).PresignUpload(context.Background(), "a.png", "image/png", ""); err == nil || errors.Is(err, ErrNotConfigured) {
		t.Errorf("no creds should be a signing failure, got %v", err)
	}

	noPublic := fullConfig()
	noPublic.R2PublicURL = ""
	if _, err := NewR2(noPublic).PresignUpload(context.Background(), "a.png", "image/png", ""); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("no public url: %v", err)
	}
}

func TestDeleteCallsBucket(t *testing.T) {
	var method, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	cfg := fullConfig()
	cfg.R2Endpoint = srv.URL
	if err := NewR2(cfg).Delete(context.Background(), "gallery/x.webp"); err != nil {
		t.Fatal(err)
	}
	if method != http.MethodDelete || path != "/photos/gallery/x.webp" {
		t.Fatalf("S3 saw %s %s", method, path)
	}
}
