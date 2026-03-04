package web

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	"podpal/internal/downloader"
)

func TestIndexHandler(t *testing.T) {
	tmpl, err := template.ParseGlob("../templates/*.html")
	if err != nil {
		t.Fatal(err)
	}

	dl := downloader.New(t.TempDir())
	h := New(tmpl, dl)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if len(body) < 100 {
		t.Error("response body too short")
	}

	// Should contain model options
	if !containsString(body, "ipodvideo") {
		t.Error("response missing ipodvideo model")
	}
	if !containsString(body, "iPod Video") {
		t.Error("response missing iPod Video description")
	}
}

func TestDownloadExpired(t *testing.T) {
	tmpl, err := template.ParseGlob("../templates/*.html")
	if err != nil {
		t.Fatal(err)
	}

	dl := downloader.New(t.TempDir())
	h := New(tmpl, dl)

	req := httptest.NewRequest("GET", "/download/nonexistent", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func containsString(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSubstr(s, sub))
}

func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
