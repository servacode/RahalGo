package server

// ══════════════════════════════════════════════════════════════════════
// **إنهاءُ الإدارة عند باب الزبون** (قرارُ المالك مساءَ ٢٠٢٦-١٠-٠٢، البند ١)
// ══════════════════════════════════════════════════════════════════════
//
//	POST /admin/orders/{id}/door-resolution
//	{action: "deliver_now" | "return_to_office", fault, reason, note}
//
// **السائقُ لا يُنهي الطلبَ عند الباب** — يُبلّغ (`/driver/orders/{id}/report`)
// وينتظر. **والمكتبُ يتّصل بالزبون ثمّ يأمر**: «سلّم الآن» أو «عُد إلى المكتب».
// **والحسابُ كلُّه في المحرّك** (`orders/door.go`) — هذا البابُ يقرأ ويُدقّق.
//
// **والأثرُ في المعاملة نفسِها** (`XG-20`): أمرٌ وقع بلا من ولا لماذا لا يُراجَع.

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

func (s *Server) handleDoorResolution(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Action string `json:"action"`
		Fault  string `json:"fault"`
		Reason string `json:"reason"`
		Note   string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	oid := chi.URLParam(r, "id")
	in := orders.DoorResolution{
		Action: strings.TrimSpace(req.Action),
		Fault:  strings.TrimSpace(req.Fault),
		Reason: strings.TrimSpace(req.Reason),
		Note:   clip(strings.TrimSpace(req.Note), 300),
	}
	o, err := s.orders.ResolveDoor(r.Context(), userIDFrom(r), rolesFrom(r), oid, in,
		func(ctx context.Context, q dbtx.Querier) error {
			return s.auditTx(ctx, q, r, "ops.door_resolution", "order", oid, map[string]any{
				"action": in.Action, "fault": in.Fault, "reason": in.Reason, "note": in.Note,
			})
		})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("order", "ops")
	httpx.JSON(w, http.StatusOK, o)
}
