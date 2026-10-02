package server

// ══════════════════════════════════════════════════════════════════════
// **بلاغُ المرحلة — «لدي مشكلة» لا تُغلق الطلب** (قرارُ المالك ٢٠٢٦-١٠-٠٢)
// ══════════════════════════════════════════════════════════════════════
//
// «زرّ لدي مشكلة له عملٌ معيّنٌ بكلّ مرحلة… **ما يصير يتسكّر الطلبُ بأيّ حالةٍ
// قبل ما يصل للزبون. لازم في حلّ لكلّ المشاكل.**»
//
// # ما يقع
//
//	المرحلةُ      ←  البلاغُ يخصّها وإلّا رُدّ (`report_wrong_stage`)
//	العملياتُ    ←  تُنبَّه فوراً بما قال وأين الطلب
//	الطلبُ        ←  **كما هو** — لا حالَ تتبدّل ولا مالَ يتحرّك
//	السجلُّ       ←  أثرٌ في المعاملة (`driver.stage_report`)
//
// # ولماذا البابُ نفسُه
//
// `POST /driver/orders/{id}/report` كان لبلاغ السائق على متجرٍ أو زبون (تذكرةُ
// دعمٍ بحقل `reason`). **والبلاغُ الجديدُ يُعرَف بحقل `code`** — فيبقى القديمُ يعمل
// كما هو لكلّ تطبيقٍ لم يُحدَّث، **ولا بابان لفعلٍ يسمّيه السائقُ باسمٍ واحد.**
//
// # والتكرارُ لا يُنبّه مرّتين
//
// سائقٌ عند بابٍ لا يُفتح يضغط ثانيةً بعد عشر ثوانٍ — **فلا يُزعَج المكتبُ
// بتنبيهين عن واقعةٍ واحدة.** البلاغُ نفسُه على الطلب نفسِه في دقيقةٍ ردٌّ ناجحٌ
// بلا أثرٍ ثانٍ (`duplicate: true`).

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

var errReportWrongStage = httpx.NewError(http.StatusConflict,
	"report_wrong_stage", "errors.report_wrong_stage")

// stageReportDedupSec **نافذةُ التكرار** — البلاغُ نفسُه فيها واحد.
const stageReportDedupSec = 60

// stageReportTitles عنوانُ التنبيه لكلّ بلاغ — **ما يقرؤه المكتبُ فيتصرّف.**
//
// **وبلاغاتُ الباب تطلب قراراً** (مساءَ ٢٠٢٦-١٠-٠٢): السائقُ لا يُنهي الطلب،
// **والمكتبُ يتّصل بالزبون ثمّ يأمر** — سلّم الآن أو عُد إلى المكتب
// (`/admin/orders/{id}/door-resolution`).
var stageReportTitles = map[string]string{
	"merchant_not_ready":          "الطلبُ غيرُ جاهز — السائقُ ينتظر عند المتجر",
	"customer_cancelled_by_phone": "الزبونُ يقول إنّه ألغى — والطلبُ مع السائق",
	"customer_no_answer":          "السائقُ عند باب الزبون — لا أحد يُجيب · اتّصل وقرّر",
	"customer_absent":             "الزبونُ غيرُ موجود عند الباب — اتّصل وقرّر",
	"customer_refused":            "الزبونُ رفض الاستلام — قرّر: سلّم أو عُد إلى المكتب",
	"customer_unreachable":        "الزبونُ لا يُوصَل إليه — اتّصل وقرّر",
	"address_wrong":               "العنوانُ خطأ — اتّصل بالزبون وقرّر",
	"driver_late":                 "السائقُ يُقرّ بتأخّره — راجِع الزبون وقرّر",
}

// handleDriverReportOrStage **بابٌ واحدٌ لبلاغين** — يُعرَف الجديدُ بـ`code`.
func (s *Server) handleDriverReportOrStage(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		s.respondErr(w, errValidation)
		return
	}
	var peek struct {
		Code string `json:"code"`
		Note string `json:"note"`
	}
	_ = json.Unmarshal(raw, &peek)
	if strings.TrimSpace(peek.Code) == "" {
		r.Body = io.NopCloser(bytes.NewReader(raw))
		s.handleDriverReport(w, r)
		return
	}
	s.driverStageReport(w, r, chi.URLParam(r, "id"), strings.TrimSpace(peek.Code), peek.Note)
}

// driverStageReport **يُنبّه العملياتِ ولا يمسّ الطلب.**
func (s *Server) driverStageReport(w http.ResponseWriter, r *http.Request, orderID, code, note string) {
	if !s.driverOwnsOrder(r, orderID) {
		s.respondErr(w, errNotYourOrder)
		return
	}
	ctx := r.Context()
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// **القفلُ على صفّ الطلب** — ضغطتان متزامنتان تُسجَّلان واحدة.
	var status string
	var number int64
	if err := tx.QueryRow(ctx,
		`SELECT status, number FROM orders WHERE id = $1 FOR UPDATE`, orderID).
		Scan(&status, &number); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if _, ok := orders.StageReportAt(code, status); !ok {
		s.respondErr(w, errReportWrongStage)
		return
	}

	var dup bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM audit_log
		              WHERE action = 'driver.stage_report' AND entity = 'order'
		                AND entity_id = $1 AND details->>'code' = $2
		                AND created_at > now() - make_interval(secs => $3))`,
		orderID, code, stageReportDedupSec).Scan(&dup); err != nil {
		s.respondErr(w, err)
		return
	}
	if dup {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"reported": true, "duplicate": true, "code": code, "status": status,
		})
		return
	}
	note = clip(strings.TrimSpace(note), 300)
	if err := s.auditTx(ctx, tx, r, "driver.stage_report", "order", orderID, map[string]any{
		"code": code, "status": status, "note": note,
	}); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		s.respondErr(w, err)
		return
	}

	// **والعملياتُ تُنبَّه بعد التثبيت** — تنبيهٌ عن بلاغٍ لم يُسجَّل يُبحث عنه فلا يوجد.
	body := "#" + strconv.FormatInt(number, 10)
	// **وكم وقف عند الباب** — من حدث الوصول، **فيعرف المكتبُ كم ينتظر رجلٌ في
	// الشارع** (مساءَ ٢٠٢٦-١٠-٠٢؛ بدل مفتاح «انتظار الباب» الذي حُذف).
	if status == orders.StAtDropoff {
		if m := s.doorWaitedMinutes(ctx, orderID); m >= 0 {
			body += " · ينتظر عند الباب منذ " + strconv.FormatInt(m, 10) + " د"
		}
	}
	if note != "" {
		body += " — " + note
	}
	title := stageReportTitles[code]
	if title == "" {
		title = "بلاغٌ من سائق"
	}
	s.notify.NotifyOps(ctx, notifications.Input{
		Kind: notifications.KindOrder, Title: title, Body: body,
		Entity: "order", EntityID: orderID, Href: "/dashboard/orders",
	})
	s.touch("order", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{
		"reported": true, "duplicate": false, "code": code, "status": status,
	})
}

// doorWaitedMinutes **كم دقيقةً مضت منذ وصل الباب** — وسالبٌ إن لم يُعرف.
func (s *Server) doorWaitedMinutes(ctx context.Context, orderID string) int64 {
	var m *float64
	if err := s.pg.QueryRow(ctx, `
		SELECT floor(EXTRACT(EPOCH FROM now() - max(created_at)) / 60)
		FROM order_events WHERE order_id = $1 AND to_status = 'at_dropoff'`,
		orderID).Scan(&m); err != nil || m == nil {
		return -1
	}
	return int64(*m)
}
