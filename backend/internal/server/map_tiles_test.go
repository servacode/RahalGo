package server

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/config"
)

// **وسيطُ البلاطات يجلب مرّةً ويحفظ** (٢٠٢٦-١٠-٠٩) — والثانيةُ من القرص لا من المصدر.
func TestMapTile_FetchOnceThenServeFromDisk(t *testing.T) {
	var hits atomic.Int32
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("\x89PNG-fake-" + r.URL.Path))
	}))
	defer up.Close()
	t.Setenv("MAP_TILE_UPSTREAM", up.URL+"/{z}/{x}/{y}.png")

	s := &Server{cfg: &config.Config{UploadsDir: t.TempDir()}, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	r := chi.NewRouter()
	r.Get("/t/{z}/{x}/{y}", s.handleMapTile)

	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest("GET", "/t/15/19935/12855.png", nil))
		if rec.Code != 200 || rec.Body.Len() == 0 {
			t.Fatalf("الطلب %d: %d", i, rec.Code)
		}
	}
	if hits.Load() != 1 {
		t.Fatalf("**طُلب المصدرُ %d مرّات — والمطلوبُ مرّةٌ واحدة**", hits.Load())
	}
	// **وبلاطةٌ خارج الشبكة لا تُطلب من المصدر.**
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("GET", "/t/3/99/1.png", nil))
	if rec.Code != 404 || hits.Load() != 1 {
		t.Fatalf("بلاطةٌ خارج الشبكة: %d · طلباتُ المصدر %d", rec.Code, hits.Load())
	}
}

// **والنمطُ يشير إلى وسيطنا** حين لا يُضبط `MAP_TILE_URL`.
func TestMapStyle_PointsToOurTiles(t *testing.T) {
	t.Setenv("MAP_TILE_URL", "")
	req := httptest.NewRequest("GET", "/api/v1/public/map-style.json", nil)
	req.Host = "api.rahalgo.com"
	if got := tileURL(req); got != "https://api.rahalgo.com/api/v1/public/tiles/{z}/{x}/{y}.png" {
		t.Fatalf("%q", got)
	}
}
