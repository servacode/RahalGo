package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/support"
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
		// ══════════════════════════════════════════════════════════
		// **واسمُ المتجر بيده** — (طلبُ المالك ٢٠٢٦-٠٨-٢٣)
		// ══════════════════════════════════════════════════════════
		//
		// **كان للأدمن وحدَه** (`/admin/merchants/{id}`) — **فمن أخطأ
		// حرفاً في اسمه يومَ سُجّل لا يملك تصحيحَه**، ويبقى الخطأُ في
		// كلّ إشعارٍ يصل سائقَه.
		//
		// **ولا يراه زبونٌ على كلّ حال** (`customer_privacy.go`) —
		// فتبديلُه لا يمسّ تجربةَ السوق.
		Name               *string `json:"name"`
		DefaultPrepMinutes *int    `json:"default_prep_minutes"`
		MinOrder           *int64  `json:"min_order"`
		// ══════════════════════════════════════════════════════════════
		// **وعنوانُه ودبّوسُه بيده هو**
		// ══════════════════════════════════════════════════════════════
		//
		// (قرارُ المالك ٢٠٢٦-٠٨-١٢: «يجب إضافة عنوان المتجر بالإعدادات
		//  لأنّه غير موجود بالويب».)
		//
		// **وكانا يُضبطان من لوحة الإدارة وحدَها** — فصاحبُ المتجر ينتقل
		// أو يجد دبّوسَه في الشارع المجاور **ولا يملك أن يصلحه**، وينتظر
		// من يفتح له اللوحة.
		//
		// **والسائقُ هو من يدفع الثمن**: يقف على بابٍ ليس بابَ المتجر.
		AddressText *string  `json:"address_text"`
		Lat         *float64 `json:"lat"`
		Lng         *float64 `json:"lng"`
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
	// **واسمٌ فارغٌ يُردّ** — **ومتجرٌ بلا اسمٍ يظهر فراغاً في كلّ
	// شاشةٍ وإشعار**، ولا يعرف السائقُ إلى أين يمضي.
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
			s.respondErr(w, errValidation)
			return
		}
		req.Name = &trimmed
	}
	// **والنقطةُ تُرسَل كاملةً أو لا تُرسَل** — نصفُها يكتب موضعاً على
	// خطِّ الاستواء: **من أرسل `lat` وحدَه نقل متجرَه إلى البحر.**
	if (req.Lat == nil) != (req.Lng == nil) {
		s.respondErr(w, errValidation)
		return
	}
	if req.Lat != nil {
		if *req.Lat < -90 || *req.Lat > 90 || *req.Lng < -180 || *req.Lng > 180 ||
			(*req.Lat == 0 && *req.Lng == 0) {
			s.respondErr(w, errValidation)
			return
		}
	}
	if _, err := s.pg.Exec(r.Context(), `
		UPDATE merchants SET
			name                 = COALESCE($7, name),
			default_prep_minutes = COALESCE($2, default_prep_minutes),
			min_order            = COALESCE($3, min_order),
			address_text         = COALESCE($4, address_text),
			-- **ولا يُمحى الدبّوسُ حين لا يُرسَل** — الشاشةُ قد تحفظ
			-- مدّةَ التحضير وحدَها، **ومن كتب نقطةً من قيمٍ فارغة
			-- أضاع موضعَ المتجر بحفظِ حقلٍ آخر.**
			location = CASE
				WHEN $5::float8 IS NOT NULL
				THEN ST_SetSRID(ST_MakePoint($6::float8, $5::float8), 4326)::geography
				ELSE location END,
			updated_at           = now()
		WHERE id = $1`, merchantID, req.DefaultPrepMinutes, req.MinOrder,
		req.AddressText, req.Lat, req.Lng, req.Name); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
}

// handleMerchantReportReasons **ما يملك المتجرُ الإبلاغَ عنه.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٦.)
//
// **وكلُّها على السائق** — فلا عمودَ جهةٍ كما في بلاغ السائق: **جهةٌ واحدةٌ
// لا تُسأل عنها.**
func (s *Server) handleMerchantReportReasons(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{"reasons": support.MerchantReportReasons})
}

// handleMerchantReport **بلاغُ متجرٍ على سائقِ طلبٍ من طلباته.**
//
// **وطرفٌ يُشتكى عليه ولا يشتكي طرفٌ ناقص**: المتجرُ يُنذَر ويُحظَر بعدّاد
// مخالفات، **ولا يُسمع منه.**
func (s *Server) handleMerchantReport(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Reason string `json:"reason"`
		Note   string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	t, err := s.support.MerchantReport(r.Context(), userIDFrom(r),
		chi.URLParam(r, "id"), req.Reason, req.Note)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("ticket", "ops")
	httpx.JSON(w, http.StatusCreated, t)
}
