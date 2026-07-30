package server

import (
	"context"
	"net/http"
	"time"
)

func timeoutCtx(r *http.Request, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), d)
}
