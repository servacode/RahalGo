package server

// ══════════════════════════════════════════════════════════════════════
// **إنهاءُ الإدارة عند باب الزبون** (قرارُ المالك مساءَ ٢٠٢٦-١٠-٠٢، البند ١)
// ══════════════════════════════════════════════════════════════════════
//
//	POST /admin/orders/{id}/door-resolution
//	{action: "deliver_now" | "return_to_office", fault, reason, note, return_to,
//	 driver_share, driver_percent, customer_action}
//
// **و`driver_share`/`customer_action`** (قرارُ المالك ٢٠٢٦-١٠-١٥): من يدفع خسارةَ الطلب
// بذنب السائق، وما يقع على الزبون بذنبه — انظر `orders/door.go`.
//
// **و`return_to`** (قرارُ المالك ٢٠٢٦-١٠-٠٣): `office` (وفارغُه) أو `store` لمتجرٍ يقبل
// الاسترداد — **وجهةُ مشوار الإرجاع** الذي يراه السائق (`orders/return_trip.go`).
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

	"github.com/servacode/rahalgo/backend/internal/authz"
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
		// ReturnTo **إلى أين يُرجع البضاعة** — `office` أو `store`.
		ReturnTo string `json:"return_to"`
		// DriverShare **من يتحمّل الخسارة بذنب السائق** (قرارُ المالك ٢٠٢٦-١٠-١٥) —
		// `platform` (وفارغُه) · `full` · `percent`.
		DriverShare string `json:"driver_share"`
		// DriverPercent **نسبةُ السائق** مع `percent` — من ١ إلى ٩٩.
		DriverPercent int `json:"driver_percent"`
		// CustomerAction **ما يقع على الزبون بذنبه** — `cash_ban` (وفارغُه) · `block` · `none`.
		CustomerAction string `json:"customer_action"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	oid := chi.URLParam(r, "id")
	in := orders.DoorResolution{
		Action:         strings.TrimSpace(req.Action),
		Fault:          strings.TrimSpace(req.Fault),
		Reason:         strings.TrimSpace(req.Reason),
		Note:           clip(strings.TrimSpace(req.Note), 300),
		ReturnTo:       strings.TrimSpace(req.ReturnTo),
		DriverShare:    strings.TrimSpace(req.DriverShare),
		DriverPercent:  req.DriverPercent,
		CustomerAction: strings.TrimSpace(req.CustomerAction),
	}
	// ══════════════════════════════════════════════════════════════════
	// **وحظرُ الحساب قدرتُه قدرةُ صفحة الحسابات** (`users.status.manage`)
	// ══════════════════════════════════════════════════════════════════
	//
	// **من لا يحظر حساباً من صفحته لا يحظره من باب الطلب** — والتدخّلُ في الطلب
	// (`orders.intervene`) لا يحمله وحدَه.
	block := in.Action == orders.DoorReturnToOffice && in.Fault == orders.FaultCustomer &&
		in.CustomerAction == orders.CustomerBlock
	if block && !s.hasCapability(r, authz.UsersStatusManage) {
		s.respondErr(w, errForbiddenCap)
		return
	}
	var blocked string
	o, err := s.orders.ResolveDoor(r.Context(), userIDFrom(r), rolesFrom(r), oid, in,
		func(ctx context.Context, q dbtx.Querier) error {
			if err := s.auditTx(ctx, q, r, "ops.door_resolution", "order", oid, map[string]any{
				"action": in.Action, "fault": in.Fault, "reason": in.Reason, "note": in.Note,
				"return_to": in.ReturnTo, "driver_share": in.DriverShare,
				"driver_percent": in.DriverPercent, "customer_action": in.CustomerAction,
			}); err != nil {
				return err
			}
			if !block {
				return nil
			}
			// **والحظرُ في معاملة الإنهاء** — بالحظر القائم نفسِه وتدقيقِه.
			var customer string
			if err := q.QueryRow(ctx,
				`SELECT COALESCE(customer_id::text, '') FROM orders WHERE id = $1`, oid).
				Scan(&customer); err != nil {
				return err
			}
			blocked = customer
			return s.identity.BlockForOrderTx(ctx, q, userIDFrom(r), customer, oid,
				in.Note, clientIP(r))
		})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if blocked != "" {
		s.identity.AfterBlock(r.Context(), blocked)
		s.touch("user", "ops")
	}
	s.touch("order", "ops")
	httpx.JSON(w, http.StatusOK, o)
}
