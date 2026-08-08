package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// إدارة المتجر لتشغيله: الجاهزية، وساعات العمل، وإعداداته.

var (
	errNotReadyable   = httpx.NewError(http.StatusConflict, "not_readyable", "errors.not_readyable")
	errReasonRequired = httpx.NewError(http.StatusBadRequest, "reason_required", "errors.reason_required")
)

// readyableStatuses الحالات التي يصحّ فيها إعلان الجاهزية.
//
// من القبول حتى وقوف السائق عند الباب. وبعد الاستلام لا معنى للإعلان — الطعام
// في يد السائق أصلاً.
var readyableStatuses = map[string]bool{
	"accepted": true, "preparing": true, "dispatching": true,
	"assigned": true, "at_pickup": true,
}

// handleMerchantReady يرفع وسم الجاهزية عن طلب.
//
// **الوسم لا يغيّر الحالة**: التحضير وإسناد السائق مساران متوازيان لا متتاليان،
// فجعلُ الجاهزية حالةً يفرض تسلسلاً كاذباً ويمنع المتجر من قولها بينما الطلب
// «تم إسناد سائق». والسائق يقرؤها حقيقةً لا مرحلة.
func (s *Server) handleMerchantReady(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	if !s.merchantOwnsOrder(r, orderID) {
		s.respondErr(w, errForbidden)
		return
	}
	var status string
	var readyAt *time.Time
	if err := s.pg.QueryRow(r.Context(),
		`SELECT status, ready_at FROM orders WHERE id = $1`, orderID).Scan(&status, &readyAt); err != nil {
		s.respondErr(w, err)
		return
	}
	if !readyableStatuses[status] {
		s.respondErr(w, errNotReadyable)
		return
	}
	// إعلانٌ ثانٍ لا يُغيّر الوقت الأول: لحظة الجاهزية واحدة، وإعادة ضبطها
	// تُفسد قياس زمن التحضير الفعلي.
	if readyAt == nil {
		if _, err := s.pg.Exec(r.Context(),
			`UPDATE orders SET ready_at = now(), updated_at = now() WHERE id = $1`, orderID); err != nil {
			s.respondErr(w, err)
			return
		}
	}
	// العمليات والسائق يريان الجاهزية فوراً — قرارُ التحرّك مبنيٌّ عليها
	s.touch("order", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"ready": true})
}

// handleMerchantSetHours ساعات عمل متجره — يكتبها بنفسه.
//
// هو من يعرف متى يفتح في رمضان ومتى يقصّر يوم الجمعة. وجعلُه يتّصل بالمنصة
// لتغيير ساعة احتكاكٌ بلا مقابل.
// handleMerchantGetHours ساعاتُ عمل متجره — يقرؤها ليعدّلها.
//
// (كشفه فحصٌ شاملٌ ٢٠٢٦-٠٨-٠٨: شاشةُ الإعدادات تردّ ٤٠٥.)
//
// **كانت الكتابةُ وحدَها موجودة**: الشاشةُ تقرأ بـGET ثمّ تكتب بـPUT على
// المسار نفسِه — **والمسارُ لا يقبل إلّا الكتابة.**
//
// **فيُفتح الجدولُ فارغاً** ويكتب صاحبُ المتجر فوقَ ساعاته وهو لا يراها:
// **من أراد تعديلَ يومٍ واحدٍ محا الأسبوعَ كلَّه.**
func (s *Server) handleMerchantGetHours(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if !s.ownsMerchant(r, merchantID) {
		s.respondErr(w, errForbidden)
		return
	}
	hours, err := s.catalog.GetHours(r.Context(), merchantID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, hours)
}

func (s *Server) handleMerchantSetHours(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if !s.ownsMerchant(r, merchantID) {
		s.respondErr(w, errForbidden)
		return
	}
	req, err := decode[struct {
		Days []catalog.DayHours `json:"days"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.catalog.SetHours(r.Context(), userIDFrom(r), merchantID, req.Days, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
}

// handleMerchantSettings إعدادات تشغيله: وقت التحضير الافتراضي وحدّه الأدنى.
//
// **ما لا يملكه هنا مقصود**: نسبة العمولة وحالة المتجر بيد الإدارة — الأولى
// عقدٌ بين طرفين لا يعدّله طرف، والثانية قرار المنصة في من يعمل على منصّتها.
func (s *Server) handleMerchantSettings(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if !s.ownsMerchant(r, merchantID) {
		s.respondErr(w, errForbidden)
		return
	}
	req, err := decode[struct {
		DefaultPrepMinutes *int   `json:"default_prep_minutes"`
		MinOrder           *int64 `json:"min_order"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if req.DefaultPrepMinutes != nil && (*req.DefaultPrepMinutes < 1 || *req.DefaultPrepMinutes > 240) {
		s.respondErr(w, errValidation)
		return
	}
	if req.MinOrder != nil && *req.MinOrder < 0 {
		s.respondErr(w, errValidation)
		return
	}
	if _, err := s.pg.Exec(r.Context(), `
		UPDATE merchants SET
			default_prep_minutes = COALESCE($2, default_prep_minutes),
			min_order            = COALESCE($3, min_order),
			updated_at           = now()
		WHERE id = $1`, merchantID, req.DefaultPrepMinutes, req.MinOrder); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
}
