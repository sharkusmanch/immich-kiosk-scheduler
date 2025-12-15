// Package server implements the HTTP server with redirect, health, and metrics endpoints.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/sharkusmanch/immich-kiosk-scheduler/internal/config"
	"github.com/sharkusmanch/immich-kiosk-scheduler/internal/scheduler"
)

// Metrics for Prometheus
var (
	redirectsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "immich_kiosk_scheduler_redirects_total",
			Help: "Total number of redirects served",
		},
		[]string{"schedule"},
	)

	currentSchedule = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "immich_kiosk_scheduler_current_schedule",
			Help: "Currently active schedule (1 = active)",
		},
		[]string{"schedule"},
	)
)

func init() {
	prometheus.MustRegister(redirectsTotal)
	prometheus.MustRegister(currentSchedule)
}

// Server is the HTTP server for immich-kiosk-scheduler.
type Server struct {
	router            chi.Router
	scheduler         *scheduler.Scheduler
	kioskURL          string
	passthroughParams map[string]bool
	port              int
	logger            *slog.Logger
}

// New creates a new Server instance.
func New(cfg *config.Config, sched *scheduler.Scheduler) (*Server, error) {
	// Build passthrough params map for O(1) lookup
	passthroughMap := make(map[string]bool)
	for _, p := range cfg.PassthroughParams {
		sanitized, valid := config.SanitizeParam(p)
		if valid {
			passthroughMap[sanitized] = true
		}
	}

	s := &Server{
		scheduler:         sched,
		kioskURL:          cfg.KioskURL,
		passthroughParams: passthroughMap,
		port:              cfg.Port,
		logger:            slog.Default(),
	}

	s.setupRoutes()
	return s, nil
}

// setupRoutes configures the HTTP routes.
func (s *Server) setupRoutes() {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(s.loggingMiddleware)

	// Routes
	r.Get("/", s.handleRedirect)
	r.Get("/healthz", s.handleHealth)
	r.Get("/metrics", promhttp.Handler().ServeHTTP)

	s.router = r
}

// loggingMiddleware logs HTTP requests.
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		s.logger.Info("http request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", ww.Status()),
			slog.Duration("duration", time.Since(start)),
			slog.String("remote", r.RemoteAddr),
		)
	})
}

// handleRedirect redirects to the kiosk URL with the appropriate album.
func (s *Server) handleRedirect(w http.ResponseWriter, r *http.Request) {
	album := s.scheduler.GetCurrentAlbum()
	scheduleName := s.scheduler.GetCurrentScheduleName()

	// Build redirect URL
	redirectURL, err := s.buildRedirectURL(r, album)
	if err != nil {
		s.logger.Error("failed to build redirect URL", slog.Any("error", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Update metrics
	redirectsTotal.WithLabelValues(scheduleName).Inc()
	s.updateCurrentScheduleMetric(scheduleName)

	s.logger.Info("redirecting",
		slog.String("schedule", scheduleName),
		slog.String("album", album),
		slog.String("redirect_url", redirectURL),
	)

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// buildRedirectURL constructs the redirect URL with album and passthrough params.
func (s *Server) buildRedirectURL(r *http.Request, album string) (string, error) {
	u, err := url.Parse(s.kioskURL)
	if err != nil {
		return "", fmt.Errorf("invalid kiosk URL: %w", err)
	}

	q := u.Query()
	q.Set("album", album)

	// Add passthrough params from the original request
	for param := range s.passthroughParams {
		if value := r.URL.Query().Get(param); value != "" {
			// URL encoding happens automatically when we call q.Encode()
			q.Set(param, value)
		}
	}

	u.RawQuery = q.Encode()
	return u.String(), nil
}

// updateCurrentScheduleMetric updates the current_schedule gauge.
func (s *Server) updateCurrentScheduleMetric(active string) {
	// Reset all to 0
	currentSchedule.Reset()
	// Set active to 1
	currentSchedule.WithLabelValues(active).Set(1)
}

// handleHealth returns a simple health check response.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]any{
		"status":   "ok",
		"schedule": s.scheduler.GetCurrentScheduleName(),
		"album":    s.scheduler.GetCurrentAlbum(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// Start begins listening for HTTP requests.
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	s.logger.Info("starting server", slog.String("addr", addr))
	return http.ListenAndServe(addr, s.router)
}

// StartWithContext begins listening for HTTP requests with graceful shutdown support.
func (s *Server) StartWithContext(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", s.port)
	srv := &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("starting server", slog.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		s.logger.Info("shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

// Router returns the chi router for testing.
func (s *Server) Router() chi.Router {
	return s.router
}
