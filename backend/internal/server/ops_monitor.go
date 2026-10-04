package server

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// ══════════════════════════════════════════════════════════════════════
// **شاشةُ المراقب — سيرُ الطلبات الآن** (قراراتُ المالك ٢٠٢٦-١٠-٠٤ — «مراقبة التشغيل»)
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان
//
// **الصفحةُ شاشةُ صحّةِ خادمٍ وحدَها** — وعلى التجهيز طلبان عالقان منذ خمس
// ساعاتٍ والصفحةُ تقول «سليم» ولا تذكرهما. **وموظّفُ العمليّات لا يراها أصلاً.**
//
// # وما صار — بابان بقدرتين (البند ٢)
//
//	GET /admin/ops/monitor   orders.read          سيرُ الطلبات + كلمةُ كلّ جزء
//	GET /admin/ops/health    observability.read   التفاصيلُ التقنيّة للمهندس
//	GET /admin/ops/status    أيُّ موظّف            حالُ الخادم للشريط الأحمر
//
// **والعالقُ من الدالّة الواحدة** (`orders.StuckReasonSQL` عبر `Alerts`) — هي
// نفسُها فلترُ «عالق» في لوحة الطلبات وإنذارُ الراصد. **فلا تقول هذه الصفحةُ
// رقماً وتعرض اللوحةُ غيرَه.**
//
// **ولا هاتفَ زبونٍ في الردّ** — المراقبُ يرى السيرَ لا الناس.

// monitorStuck **طلبٌ عالقٌ كما يراه المراقب.**
type monitorStuck struct {
	OrderID      string     `json:"order_id"`
	Number       int64      `json:"number"`
	Status       string     `json:"status"`
	MerchantName string     `json:"merchant_name"`
	Reason       string     `json:"reason"`
	Since        time.Time  `json:"since"`
	Minutes      float64    `json:"minutes"`
	AckedAt      *time.Time `json:"acked_at"`
	Reminders    int        `json:"reminders"`
}

type monitorDrivers struct {
	OnShift num `json:"on_shift"`
	Busy    num `json:"busy"`
	Free    num `json:"free"`
}

// monitorSystem **حالُ الخادم بكلمات** — بلا أرقامٍ ولا أسباب.
type monitorSystem struct {
	State string            `json:"state"`
	Parts map[string]string `json:"parts"`
	// Since **متى بدأت المشكلة** — والفارغُ لا مشكلة أو لم يرصدها الراصد بعد.
	Since *time.Time `json:"since"`
}

type opsMonitor struct {
	GeneratedAt time.Time `json:"generated_at"`
	// Flow **عدّاداتُ المراحل بشروط لوحة الطلبات** (`orders.BoardCounts`) —
	// وكلُّ مفتاحٍ فلترٌ يُفتح بـ`/dashboard/orders?filter=`.
	Flow            map[string]num `json:"flow"`
	Stuck           []monitorStuck `json:"stuck"`
	Drivers         monitorDrivers `json:"drivers"`
	EmergenciesOpen num            `json:"emergencies_open"`
	System          monitorSystem  `json:"system"`
	// ReminderMin **كلَّ كم دقيقةٍ يُعاد تذكيرُ العالق** — وصفرُه لا تكرار.
	ReminderMin int64 `json:"reminder_min"`
	// Missing **أقسامٌ لم تُقرأ** — ورقمُها `null` لا صفر.
	Missing []string `json:"missing"`
}

// handleOpsMonitor **شاشةُ المراقب** — `GET /admin/ops/monitor` بقدرة `orders.read`.
func (s *Server) handleOpsMonitor(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := timeoutCtx(r, 8*time.Second)
	defer cancel()
	httpx.JSON(w, http.StatusOK, s.buildOpsMonitor(ctx))
}

func (s *Server) buildOpsMonitor(ctx context.Context) opsMonitor {
	now := time.Now()
	out := opsMonitor{GeneratedAt: now, Flow: map[string]num{}, Stuck: []monitorStuck{}, Missing: []string{}}
	miss := func(section string, err error) bool {
		if err == nil {
			return false
		}
		s.logger.Error("مراقبة التشغيل: قسمٌ لم يُقرأ", "section", section, "error", err)
		out.Missing = append(out.Missing, section)
		return true
	}

	if c, err := s.orders.BoardCounts(ctx); !miss("flow", err) {
		for k, v := range c {
			out.Flow[k] = n64(int64(v))
		}
	}
	if al, err := s.orders.Alerts(ctx); !miss("stuck", err) {
		for _, a := range al {
			out.Stuck = append(out.Stuck, monitorStuck{
				OrderID: a.OrderID, Number: a.Number, Status: a.Status,
				MerchantName: a.MerchantName, Reason: a.Reason, Since: a.Since,
				Minutes: a.Minutes, AckedAt: a.AckedAt, Reminders: a.Reminders,
			})
		}
	}
	if onShift, busy, err := s.driverShiftCounts(ctx); !miss("drivers", err) {
		out.Drivers = monitorDrivers{OnShift: n64(onShift), Busy: n64(busy), Free: n64(onShift - busy)}
	}
	{
		var n int64
		err := s.pg.QueryRow(ctx, `SELECT count(*) FROM driver_emergencies WHERE status = 'open'`).Scan(&n)
		if !miss("emergencies", err) {
			out.EmergenciesOpen = n64(n)
		}
	}
	if s.settings != nil {
		out.ReminderMin = s.settings.GetInt(ctx, "ops.stuck_reminder_min")
	}
	out.System = s.systemWords(ctx, now)
	return out
}

// driverShiftCounts **السائقون على الدوام ومن معه طلبٌ مفتوح** — تقرؤه
// الرئيسيّةُ وشاشةُ المراقب بنصٍّ واحد.
func (s *Server) driverShiftCounts(ctx context.Context) (onShift, busy int64, err error) {
	err = s.pg.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE u.on_shift),
		       count(*) FILTER (WHERE u.on_shift AND EXISTS (
		           SELECT 1 FROM orders o WHERE o.driver_id = u.id AND o.closed_at IS NULL))
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id AND ur.role_code = 'driver'
		WHERE u.status = 'active'`).Scan(&onShift, &busy)
	return onShift, busy, err
}

// systemWords **حالُ الخادم بكلمات** — من آخر حكمٍ إن كان حديثاً، وإلّا يُقاس الآن.
func (s *Server) systemWords(ctx context.Context, now time.Time) monitorSystem {
	v, since, ok := s.opsT.recent(90*time.Second, now)
	if !ok {
		v = s.collectOps(ctx).Verdict
		_, since, _ = s.opsT.recent(time.Hour, now)
	}
	out := monitorSystem{State: v.State, Parts: v.plainParts()}
	if !since.IsZero() && v.State != stOK {
		out.Since = &since
	}
	return out
}

// handleOpsStatus **حالُ الخادم للشريط الأحمر** — `GET /admin/ops/status`.
//
// **لكلّ موظّفٍ في اللوحة** (`RequireAnyCapability` قبله): كلماتٌ بلا أرقام،
// **ومن يعمل في لوحةٍ خادمُها يتعثّر يحقّ له أن يعرف لماذا لا تستجيب.**
// (قرارُ المالك ٢٠٢٦-١٠-٠٤ — «مراقبة التشغيل»، البند ٣.)
func (s *Server) handleOpsStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := timeoutCtx(r, 5*time.Second)
	defer cancel()
	httpx.JSON(w, http.StatusOK, s.systemWords(ctx, time.Now()))
}

// handleAlertAck **«أنا عليه» على طلبٍ عالق** — `POST /admin/orders/{id}/alert-ack`.
//
// يوقف تكرارَ التذكير بسببه الحاليّ ولا يُخرجه من قائمة العالق (البند ٤).
func (s *Server) handleAlertAck(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	changed, err := s.orders.AckAlert(r.Context(), id, userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"acked": true, "changed": changed})
}

// ══════════════════════════════════════════════════════════════════════
// **إنذارُ تعطّل الخادم** (قرارُ المالك ٢٠٢٦-١٠-٠٤ — «مراقبة التشغيل»، البند ٣)
// ══════════════════════════════════════════════════════════════════════
//
// **كانت الحالةُ تُحسب حين يفتح أحدٌ الصفحة** — فقاعدةٌ تسقط ليلاً لا يعلم بها
// أحدٌ حتّى الصباح. **وصار راصدٌ يقيس كلَّ نصف دقيقة**:
//
//   - تبدّل الحالُ ⇒ بثٌّ «system» فيظهر الشريطُ الأحمرُ أو يختفي في كلّ لوحةٍ مفتوحة.
//   - طال العطبُ أكثرَ من `ops.outage_notify_min` ⇒ إشعارٌ للمالك والمدير، مرّةً للانقطاع.
//   - عاد سليماً بعد إشعار ⇒ إشعارُ «رجع كلُّ شيء».
//
// **وإن كانت القاعدةُ هي الواقعة فالإشعارُ لا يُكتب** — يُعاد في كلّ نبضةٍ حتّى
// يُكتب. **ولا يُغني هذا عن مراقبٍ من خارج الخادم**: خادمٌ مات لا يُرسل شيئاً.

// outageRoles **من يُشعَر بالتعطّل** — مالكُ المنصّة والمدير.
var outageRoles = []string{"owner_super_admin", "admin"}

// RunOpsWatch **راصدُ الخادم** — يُطلَق مرّةً عند الإقلاع.
func (s *Server) RunOpsWatch(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		h := s.collectOps(cctx)
		s.outageStep(cctx, h.Verdict, time.Now())
		cancel()
	}
}

// outageStep **خطوةٌ واحدةٌ من الراصد** — ويُرجع أأُرسل إشعارُ تعطّلٍ فيها.
// (مِعراضُ الفحص: الوقتُ والحكمُ يُمرَّران.)
func (s *Server) outageStep(ctx context.Context, v opsVerdict, now time.Time) bool {
	t := &s.opsT
	bad := v.State != stOK
	t.mu.Lock()
	prevBad := !t.badSince.IsZero()
	changed := prevBad != bad
	if !bad {
		wasNotified, since := t.notified, t.badSince
		t.badSince, t.notified = time.Time{}, false
		t.mu.Unlock()
		if changed {
			s.publishSystem(v.State)
		}
		if wasNotified {
			// **وعودةُ السلامة تُقال لمن أُخبر بالعطب** — وإلّا بقي ينتظر.
			s.sendOutage(ctx, "رجع كل شي شغّال",
				"المشكلة دامت "+strconv.Itoa(int(now.Sub(since).Minutes()))+" دقيقة")
		}
		return false
	}
	if !prevBad {
		t.badSince = now
	}
	since, notified := t.badSince, t.notified
	t.mu.Unlock()
	if changed {
		s.publishSystem(v.State)
	}
	limit := time.Duration(s.outageNotifyMin(ctx)) * time.Minute
	if notified || now.Sub(since) < limit {
		return false
	}
	mins := strconv.Itoa(int(now.Sub(since).Minutes()))
	if !s.sendOutage(ctx, "الخادم فيه مشكلة من "+mins+" دقيقة", outageBody(v)) {
		return false
	}
	t.mu.Lock()
	t.notified = true
	t.mu.Unlock()
	return true
}

func (s *Server) outageNotifyMin(ctx context.Context) int64 {
	if s.settings == nil {
		return 5
	}
	if n := s.settings.GetInt(ctx, "ops.outage_notify_min"); n > 0 {
		return n
	}
	return 5
}

// outagePartNames/outageStateWords **أسماءٌ عربيّةٌ لنصّ الإشعار** — والشاشةُ
// تقرأ معجمَها.
var outagePartNames = map[string]string{
	partAPI: "الخادم", partDB: "قاعدة البيانات", partCache: "الذاكرة السريعة",
	partPush: "الإشعارات", partRealtime: "الاتصال اللحظي",
}

var outageStateWords = map[string]string{stSlow: "بطيء", stDown: "واقف"}

func outageBody(v opsVerdict) string {
	var parts []string
	for _, k := range opsPartsOrder {
		if w, ok := outageStateWords[v.Parts[k].State]; ok {
			parts = append(parts, outagePartNames[k]+": "+w)
		}
	}
	return strings.Join(parts, " · ")
}

// sendOutage **يكتب الإشعارَ للمالك والمدير** — ويُرجع أكُتب.
func (s *Server) sendOutage(ctx context.Context, title, body string) bool {
	if s.notify == nil {
		return false
	}
	in := notifications.Input{
		Kind: notifications.KindAccount, Title: title, Body: body,
		Entity: "system", Href: "/dashboard/ops",
	}
	n, err := s.notify.NotifyRolesTx(ctx, s.pg, outageRoles, in)
	if err != nil {
		s.logger.Warn("مراقبة التشغيل: تعذّر إشعارُ التعطّل — يُعاد في النبضة التالية", "error", err)
		return false
	}
	if n == 0 {
		return false
	}
	s.notify.PublishToUsers(ctx, s.pg, outageRoles, in)
	return true
}

func (s *Server) publishSystem(state string) {
	if s.hub == nil {
		return
	}
	s.hub.Publish("ops", map[string]any{"type": "system", "state": state})
}
