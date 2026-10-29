package server

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ══════════════════════════════════════════════════════════════════════
// **حذفُ متجرٍ نهائيّاً — لمتجرٍ بلا طلب** (قرارُ المالك ٢٠٢٦-١٠-٢٠)
// ══════════════════════════════════════════════════════════════════════
//
// «فطاير الفاخر» أُنشئ مرّتين برقمين — «نحذف هذا بشكل نهائي مشان نعمله موظف».
//
// **ولا يُحذف متجرٌ له طلبٌ أو تسوية** — تاريخُ الطلبات والمال مربوطٌ به (`orders`
// و`merchant_settlements` بلا حذفٍ متتابع)، **ومحوُه يمحو دفتراً.** فذاك يُوقَف لا يُحذف.
//
// **وما يُحذف معه** بقيده المتتابع: ساعاتُه وأصنافُه وأقسامُه ومشروباتُه وإنذاراتُه.
// **ومرشَّحُ المندوب يبقى** ويُفكّ عنه — تاريخُ عمل المندوب لا يُمحى.
// **وصاحبُه إن لم يبقَ له متجرٌ** تُنزع عنه صفةُ المتجر — فيصير رقمُه صالحاً
// لموظّفٍ عند متجرٍ آخر (`identity.AddStaff`)، ويبقى زبوناً كما كان.

var errMerchantHasOrders = httpx.NewError(http.StatusConflict, "merchant_has_orders", "errors.conflict")

func (s *Server) handleDeleteMerchant(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ctx := r.Context()
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var name string
	var number int64
	var owner *string
	var busy bool
	err = tx.QueryRow(ctx, `
		SELECT m.name, m.number, m.owner_user_id::text,
		       -- **والطلبُ الذي قُلب «من متجرٍ غير تابع» يبقى أثرُه على متجره الأوّل** (٢٠٢٦-١٠-٢٩).
		       EXISTS (SELECT 1 FROM orders o WHERE o.merchant_id = m.id OR o.partner_merchant_id = m.id
		                                         OR o.ext_store_merchant_id = m.id)
		       OR EXISTS (SELECT 1 FROM merchant_settlements ms WHERE ms.merchant_id = m.id)
		FROM merchants m WHERE m.id::text = $1 FOR UPDATE`, id).Scan(&name, &number, &owner, &busy)
	if errors.Is(err, pgx.ErrNoRows) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if busy {
		s.respondErr(w, errMerchantHasOrders)
		return
	}
	if _, err := tx.Exec(ctx, `UPDATE merchant_leads SET merchant_id = NULL WHERE merchant_id::text = $1`, id); err != nil {
		s.respondErr(w, err)
		return
	}
	if _, err := tx.Exec(ctx, `DELETE FROM merchants WHERE id::text = $1`, id); err != nil {
		s.respondErr(w, err)
		return
	}
	ownerFreed := false
	if owner != nil {
		tag, err := tx.Exec(ctx, `
			DELETE FROM user_roles WHERE user_id = $1::uuid AND role_code = 'merchant'
			   AND NOT EXISTS (SELECT 1 FROM merchants WHERE owner_user_id = $1::uuid)`, *owner)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		ownerFreed = tag.RowsAffected() > 0
	}
	if err := tx.Commit(ctx); err != nil {
		s.respondErr(w, err)
		return
	}
	// **وموظّفو صاحبٍ لم يبقَ له متجرٌ يُحذفون** — لا يدخلون إلى لا شيء.
	if ownerFreed {
		if list, err := s.identity.ListStaff(ctx, *owner); err == nil {
			for _, st := range list {
				_ = s.identity.RemoveStaff(ctx, userIDFrom(r), *owner, st.ID, clientIP(r))
			}
		}
	}
	s.auditCtx(ctx, userIDFrom(r), clientIP(r), "admin.merchant_deleted", "merchant", id,
		map[string]any{"name": name, "number": number, "owner_freed": ownerFreed})
	s.touch("merchant", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"deleted": true, "owner_freed": ownerFreed})
}
