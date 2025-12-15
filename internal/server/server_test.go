package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sharkusmanch/immich-kiosk-scheduler/internal/config"
	"github.com/sharkusmanch/immich-kiosk-scheduler/internal/scheduler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T, cfg *config.Config) *Server {
	t.Helper()
	sched, err := scheduler.New(cfg)
	require.NoError(t, err)

	srv, err := New(cfg, sched)
	require.NoError(t, err)

	return srv
}

func TestServer_Redirect(t *testing.T) {
	cfg := &config.Config{
		KioskURL:          "https://kiosk.example.com",
		DefaultAlbum:      "default-album-id",
		Port:              8080,
		PassthroughParams: []string{},
		Schedule:          []config.ScheduleEntry{},
	}

	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "https://kiosk.example.com?album=default-album-id", rec.Header().Get("Location"))
}

func TestServer_RedirectWithPassthroughParams(t *testing.T) {
	cfg := &config.Config{
		KioskURL:          "https://kiosk.example.com",
		DefaultAlbum:      "default-album-id",
		Port:              8080,
		PassthroughParams: []string{"transition", "duration"},
		Schedule:          []config.ScheduleEntry{},
	}

	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/?transition=fade&duration=30", nil)
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusFound, rec.Code)

	location := rec.Header().Get("Location")
	assert.Contains(t, location, "album=default-album-id")
	assert.Contains(t, location, "transition=fade")
	assert.Contains(t, location, "duration=30")
}

func TestServer_RedirectFiltersUnallowedParams(t *testing.T) {
	cfg := &config.Config{
		KioskURL:          "https://kiosk.example.com",
		DefaultAlbum:      "default-album-id",
		Port:              8080,
		PassthroughParams: []string{"transition"},
		Schedule:          []config.ScheduleEntry{},
	}

	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/?transition=fade&evil=<script>", nil)
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusFound, rec.Code)

	location := rec.Header().Get("Location")
	assert.Contains(t, location, "transition=fade")
	assert.NotContains(t, location, "evil")
	assert.NotContains(t, location, "script")
}

func TestServer_RedirectSanitizesParamValues(t *testing.T) {
	cfg := &config.Config{
		KioskURL:          "https://kiosk.example.com",
		DefaultAlbum:      "default-album-id",
		Port:              8080,
		PassthroughParams: []string{"transition"},
		Schedule:          []config.ScheduleEntry{},
	}

	srv := newTestServer(t, cfg)

	// Attempt to inject via param value
	req := httptest.NewRequest(http.MethodGet, "/?transition=fade%22%3E%3Cscript%3E", nil)
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusFound, rec.Code)

	location := rec.Header().Get("Location")
	// URL encoding should prevent injection
	assert.NotContains(t, location, "<script>")
}

func TestServer_HealthCheck(t *testing.T) {
	cfg := &config.Config{
		KioskURL:          "https://kiosk.example.com",
		DefaultAlbum:      "default-album-id",
		Port:              8080,
		PassthroughParams: []string{},
		Schedule:          []config.ScheduleEntry{},
	}

	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "ok")
}

func TestServer_Metrics(t *testing.T) {
	cfg := &config.Config{
		KioskURL:          "https://kiosk.example.com",
		DefaultAlbum:      "default-album-id",
		Port:              8080,
		PassthroughParams: []string{},
		Schedule:          []config.ScheduleEntry{},
	}

	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	// Should contain Prometheus metrics format
	assert.Contains(t, rec.Body.String(), "# HELP")
}

func TestServer_RedirectIncrementsMetrics(t *testing.T) {
	cfg := &config.Config{
		KioskURL:          "https://kiosk.example.com",
		DefaultAlbum:      "default-album-id",
		Port:              8080,
		PassthroughParams: []string{},
		Schedule:          []config.ScheduleEntry{},
	}

	srv := newTestServer(t, cfg)

	// Make a redirect request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)

	// Check metrics
	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec = httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)

	assert.Contains(t, rec.Body.String(), "immich_kiosk_scheduler_redirects_total")
}

func TestServer_NotFound(t *testing.T) {
	cfg := &config.Config{
		KioskURL:          "https://kiosk.example.com",
		DefaultAlbum:      "default-album-id",
		Port:              8080,
		PassthroughParams: []string{},
		Schedule:          []config.ScheduleEntry{},
	}

	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}
