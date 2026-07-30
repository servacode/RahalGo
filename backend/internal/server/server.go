// Package server يبني موجّه HTTP وطبقة الوسطاء.
// البنية الطبقية الملزمة: handlers → services → repositories (GROUND-RULES §3).
package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/servacode/rahalgo/backend/internal/config"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

type Server struct {
	cfg    *config.Config
	logger *slog.Logger
	pg     *pgxpool.Pool
	rdb    *redis.Client
}

func New(cfg *config.Config, logger *slog.Logger, pg *pgxpool.Pool, rdb *redis.Client) *Server {
	return &Server{cfg: cfg, logger: logger, pg: pg, rdb: rdb}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   s.allowedOrigins(),
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/healthz", s.handleHealth)

	r.Route("/api/v1", func(r chi.Router) {
		// نقاط الـAPI تُسجَّل هنا مع تقدم المراحل.
	})

	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		httpx.Error(w, httpx.ErrNotFound)
	})

	return r
}

func (s *Server) allowedOrigins() []string {
	if s.cfg.Env == "development" {
		return []string{"http://localhost:*", "http://127.0.0.1:*"}
	}
	// تُضبط نطاقات الإنتاج عند النشر (المرحلة 5).
	return []string{"https://*.rahalgo.com"}
}
