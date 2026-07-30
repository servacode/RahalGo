package server

import (
	"net/http"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

type healthStatus struct {
	Status   string `json:"status"`
	Postgres string `json:"postgres"`
	Redis    string `json:"redis"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := timeoutCtx(r, 5*time.Second)
	defer cancel()

	st := healthStatus{Status: "ok", Postgres: "ok", Redis: "ok"}
	code := http.StatusOK

	if err := s.pg.Ping(ctx); err != nil {
		st.Status, st.Postgres = "degraded", err.Error()
		code = http.StatusServiceUnavailable
	}
	if err := s.rdb.Ping(ctx).Err(); err != nil {
		st.Status, st.Redis = "degraded", err.Error()
		code = http.StatusServiceUnavailable
	}

	httpx.JSON(w, code, st)
}
