package server

// ══════════════════════════════════════════════════════════════════════
// **عينُ الأمان — الراصدُ يرى من يطرق الباب** (مراقبةُ المنصّة، الدفعةُ الثانية ٢٠٢٦-١٠-١٧)
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان
//
// **الحماياتُ تعمل ولا يعلم بها أحد**: قفلُ الرقم بعد محاولات كلمة السر، وحدُّ
// الرموز، وقفلُ كود التسليم، ورفضُ صورة التسليم بموقعٍ مزيَّف، ورفضُ من يفتح
// طلبَ غيره. **كلُّها تردّ بصمت** — فمن يجرّب مئةَ كلمةٍ على حساب متجرٍ أو
// يقلّب أرقامَ الطلبات **لا يصل المالكَ منه شيء.**
//
// # وما صار
//
// **الحماياتُ كما هي حرفاً بحرف** — لا حدَّ تبدّل ولا ردَّ. **والجديدُ عدٌّ
// وتنبيه**:
//
//	login_failed   كلمةُ سرٍّ خاطئة (أو رقمٌ لا حسابَ له)        ← `identity`
//	login_locked   رقمٌ انقفل الآن بعد `security.login_max_attempts` ← `identity`
//	otp_limited    طلبُ رمزٍ رُدّ لأنّه فوق الحدّ                   ← `identity`
//	code_wrong     كودُ تسليمٍ خاطئ                               ← `delivery_code.go`
//	code_locked    كودُ تسليمٍ انقفل الآن                          ← `delivery_code.go`
//	gps_mocked     صورةُ تسليمٍ بموقعٍ مزيَّف                       ← `delivery_proof.go`
//	forbidden      محاولةُ فتح ما ليس له                          ← وسيطُ `countErrors`
//
// **والعدُّ في الذاكرة** (كعدّاد الأخطاء) **ويُكتب مجمَّعاً باليوم** في
// `security_events_daily` كلَّ دقيقة — **فتقريرُ الصباح يقرأ «مبارح» ولو أُعيد
// تشغيلُ الخادم في الليل.**
//
// # والتنبيهات
//
// **فوريّةٌ لما يقع مرّةً** — حسابٌ انقفل، كودٌ انقفل، موقعٌ مزيَّف: رسالةٌ
// واحدةٌ لكلّ صاحبٍ في الساعة (`secAlertRepeat`)، **تُرسَل حالاً** لا بعد
// دقيقة (`kick` يوقظ الراصد).
//
// **وقفزةٌ لما يتكاثر** — محاولاتُ الدخول الغلط وفتحُ بيانات الغير في عشر
// دقائق، **بحدَّين في اللوحة** (`monitoring.login_fail_spike` و
// `monitoring.forbidden_spike`)، **وتمرّ بـ`alertBook`**: مرّةً، ثمّ «رجع طبيعي».
//
// **ومفتاحٌ واحدٌ يُطفئها كلَّها** (`monitoring.security_alerts`) — **والعدُّ
// يبقى** ليقرأه التقرير.
//
// # ولا سرَّ في رسالة
//
// **لا رقمَ هاتفٍ كامل** — آخرُ ثلاثة أرقامٍ وحدَها. **ولا رمزَ ولا كلمةَ ولا
// كودَ ولا توكن.** واسمُ السائق مسموحٌ (من الطاقم)، ورقمُ الطلب وحدَه للطلب.
//
// # ولماذا «محاولةُ فتح بيانات الغير» تُعدّ هكذا
//
// **٤٠٣ برمز `forbidden` أو `not_your_order` على نداءٍ يحمل توكناً** — ذاك
// رفضُ التخويل بعينه. **وما سواه من ٤٠٣ قرارُ عمل** (متجرٌ معلَّق، كلمةٌ
// يجب تبديلُها، تأكيدُ خطوة) **فلا يُعدّ.**
//
// **و٤٠٤ يُعدّ بشرطين** — لأنّ المنصّة تُخفي طلبَ غيرك بـ٤٠٤ لا ٤٠٣
// (`handleMyOrder`: «ليس لك» = «غيرُ موجود»، كي لا يُعرف أنّه موجود). **فلو
// لم يُعدّ لَما رُئي من يقلّب أرقامَ طلبات الزبائن أبداً.** والشرطان:
//
//  1. **مسارٌ فيه معرّف** (`{id}` وأخواتُه في نمط الموجّه) **وردٌّ برمز
//     `not_found`** — لا رابطٌ مجهولٌ ولا صفحةٌ ليست موجودة.
//  2. **والمعرّفُ المختلفُ وحدَه يُعدّ** — بصمةُ (التوكن + الطريقة + الرابط)
//     تُعدّ مرّةً في عشر دقائق. **فتطبيقٌ يسأل كلَّ خمس ثوانٍ عن طلبٍ حُذف
//     يُعدّ مرّة** لا مئةً وعشرين، **ومن يقلّب الأرقامَ يُعدّ لكلّ رقم.**
//
// **والشرطُ الثاني نفسُه على ٤٠٣** — فحسابٌ محظورٌ يعيد تطبيقُه المحاولةَ
// على البابين نفسَيهما لا يصنع قفزةً كاذبة.

import (
	"context"
	"hash/fnv"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/platform"
)

// أنواعُ أحداث الأمان في الخادم — وأنواعُ الهويّة الثلاثة بأسمائها هناك.
const (
	secLoginFailed = identity.SecLoginFailed
	secLoginLocked = identity.SecLoginLocked
	secOTPLimited  = identity.SecOTPLimited
	secCodeWrong   = "code_wrong"
	secCodeLocked  = "code_locked"
	secGPSMocked   = "gps_mocked"
	secForbidden   = "forbidden"
)

const (
	// secAlertRepeat **التنبيهُ الفوريُّ لصاحبٍ واحدٍ مرّةً في الساعة.**
	secAlertRepeat = time.Hour
	// secRingSize **كم وقتاً تحفظ حلقةُ كلّ قفزة.**
	secRingSize = 4096
	// secSeenCap **سقفُ بصمات «المختلف وحدَه»** — وما زاد يُنظَّف من القديم.
	secSeenCap = 20000
	// secQueueCap **سقفُ التنبيهات المنتظرة** — والقناةُ مقطوعةٌ طويلاً.
	secQueueCap = 50
	// securityKeep **كم يبقى عدُّ الأمان اليوميّ في الجدول.**
	securityKeep = 90 * 24 * time.Hour
)

type secDayKey struct {
	Day  string // يومُ دمشق `2006-01-02`
	Kind string
}

// secRing **حلقةُ أوقاتٍ لقفزة** — كحلقة الأخطاء.
type secRing struct {
	at   [secRingSize]time.Time
	next int
}

func (g *secRing) add(now time.Time) {
	g.at[g.next] = now
	g.next = (g.next + 1) % secRingSize
}

func (g *secRing) since(t time.Time) int {
	n := 0
	for _, at := range g.at {
		if !at.IsZero() && !at.Before(t) {
			n++
		}
	}
	return n
}

// secAlert **تنبيهٌ فوريٌّ ينتظر الإرسال.**
type secAlert struct {
	Subject string
	Text    string
}

// securityTracker **عدّادُ الأمان في الذاكرة** — وصفرُه صالح.
type securityTracker struct {
	mu      sync.Mutex
	pending map[secDayKey]int64
	login   secRing
	forbid  secRing
	// seen **بصماتُ ما عُدّ من الرفض** — ومتى. انظر «المختلفُ وحدَه» أعلاه.
	seen map[uint64]time.Time
	// sentAt **متى نُبّه عن كلّ صاحب** — منعُ التكرار في الساعة.
	sentAt map[string]time.Time
	queue  []secAlert
	kick   chan struct{}
}

// kickCh **جرسُ الراصد** — يُنشأ مرّة.
func (t *securityTracker) kickCh() chan struct{} {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.kick == nil {
		t.kick = make(chan struct{}, 1)
	}
	return t.kick
}

// record **يعدّ حدثاً واحداً** — قفلٌ وزيادة.
func (t *securityTracker) record(kind string, now time.Time) {
	k := secDayKey{Day: now.In(platform.Location()).Format("2006-01-02"), Kind: kind}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.pending == nil {
		t.pending = map[secDayKey]int64{}
	}
	t.pending[k]++
	switch kind {
	case secLoginFailed:
		t.login.add(now)
	case secForbidden:
		t.forbid.add(now)
	}
}

// recordDistinct **يعدّ الرفضَ إن لم تُرَ بصمتُه في عشر دقائق** — ويُرجع أعَدَّ.
func (t *securityTracker) recordDistinct(kind string, fp uint64, now time.Time) bool {
	t.mu.Lock()
	if t.seen == nil {
		t.seen = map[uint64]time.Time{}
	}
	if at, ok := t.seen[fp]; ok && now.Sub(at) < spikeWindow {
		t.mu.Unlock()
		return false
	}
	if len(t.seen) >= secSeenCap {
		for k, at := range t.seen {
			if now.Sub(at) >= spikeWindow {
				delete(t.seen, k)
			}
		}
	}
	if len(t.seen) < secSeenCap {
		t.seen[fp] = now
	}
	t.mu.Unlock()
	t.record(kind, now)
	return true
}

// countSince **كم حدثاً من نوعٍ له حلقةٌ منذ لحظة.**
func (t *securityTracker) countSince(kind string, since time.Time) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	switch kind {
	case secLoginFailed:
		return t.login.since(since)
	case secForbidden:
		return t.forbid.since(since)
	}
	return 0
}

// enqueue **تنبيهٌ فوريٌّ لصاحبه** — ويُرمى إن نُبّه عنه في الساعة.
func (t *securityTracker) enqueue(subject, text string, now time.Time) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.sentAt == nil {
		t.sentAt = map[string]time.Time{}
	}
	if at, ok := t.sentAt[subject]; ok && now.Sub(at) < secAlertRepeat {
		return false
	}
	for _, q := range t.queue {
		if q.Subject == subject {
			return false
		}
	}
	if len(t.queue) >= secQueueCap {
		return false
	}
	t.queue = append(t.queue, secAlert{Subject: subject, Text: text})
	if t.kick != nil {
		select {
		case t.kick <- struct{}{}:
		default:
		}
	}
	return true
}

// drain **يرسل المنتظر** — وما لم يصل يبقى للنبضة التالية.
func (t *securityTracker) drain(now time.Time, send func(string) bool) {
	t.mu.Lock()
	q := t.queue
	t.queue = nil
	t.mu.Unlock()
	var rest []secAlert
	for _, a := range q {
		if send(a.Text) {
			t.mu.Lock()
			if t.sentAt == nil {
				t.sentAt = map[string]time.Time{}
			}
			t.sentAt[a.Subject] = now
			t.mu.Unlock()
			continue
		}
		rest = append(rest, a)
	}
	if len(rest) > 0 {
		t.mu.Lock()
		t.queue = append(rest, t.queue...)
		t.mu.Unlock()
	}
}

// dropQueue **يرمي المنتظر** — والتنبيهاتُ مُطفأة: لا يُرسَل ما فات بعد تشغيلها.
func (t *securityTracker) dropQueue() {
	t.mu.Lock()
	t.queue = nil
	t.mu.Unlock()
}

func (t *securityTracker) take() map[secDayKey]int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	p := t.pending
	t.pending = nil
	return p
}

func (t *securityTracker) putBack(p map[secDayKey]int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.pending == nil {
		t.pending = map[secDayKey]int64{}
	}
	for k, n := range p {
		t.pending[k] += n
	}
}

// ── نصوصُ التنبيه ────────────────────────────────────────────────────

// lockAlertText **حسابٌ انقفل** — بآخر ثلاثة أرقامٍ وحدَها.
func lockAlertText(tail string) string {
	if tail == "" {
		tail = "؟"
	}
	return "🔐 حساب انقفل من كتر محاولات كلمة السر: رقم ينتهي بـ …" + tail + "\n" +
		"حدا جرّب كلمات سر غلط لحد ما انقفل. بينفتح لحاله بعد ربع ساعة، وإذا تكرّر شوف مين صاحب الحساب."
}

// codeLockAlertText **كودُ التسليم انقفل** — برقم الطلب وحدَه.
func codeLockAlertText(number int64) string {
	return "🔢 كود التسليم انقفل على الطلب #" + strconv.FormatInt(number, 10) + "\n" +
		"السائق كتب الكود غلط كذا مرة. العمليات بتفكّه من صفحة الطلب بعد ما تتأكّد من الزبون بالتلفون."
}

// mockedAlertText **سائقٌ زيّف موقعه** — باسمه (من الطاقم) ورقم الطلب.
func mockedAlertText(driverName string, number int64) string {
	if strings.TrimSpace(driverName) == "" {
		driverName = "سائق بلا اسم"
	}
	return "📍 سائق حاول يزيّف موقعه: " + driverName + " — الطلب #" + strconv.FormatInt(number, 10) + "\n" +
		"انرفضت صورة التسليم لأنّ الموقع مزيّف. شوف شو صار مع هالطلب."
}

// ── منافذُ العدّ ────────────────────────────────────────────────────

// securityAlertsOn **أتُرسَل تنبيهاتُ الأمان؟** — الراصدُ ومفتاحُ الأمان معاً.
func (s *Server) securityAlertsOn(ctx context.Context) bool {
	return s.monSetting(ctx, "monitoring.enabled") && s.monSetting(ctx, "monitoring.security_alerts")
}

// noteIdentitySecurity **مستمعُ الهويّة** — يُحقن في `server.New`.
func (s *Server) noteIdentitySecurity(ctx context.Context, ev identity.SecurityEvent) {
	now := time.Now()
	s.sec.record(ev.Kind, now)
	if ev.Kind == secLoginLocked && s.securityAlertsOn(ctx) {
		s.sec.enqueue("lock:"+ev.Subject, lockAlertText(ev.PhoneTail), now)
	}
}

// noteCodeWrong **كودُ تسليمٍ خاطئ** — و`locked` إن انقفل به الآن.
func (s *Server) noteCodeWrong(ctx context.Context, orderID string, locked bool) {
	now := time.Now()
	s.sec.record(secCodeWrong, now)
	if !locked {
		return
	}
	s.sec.record(secCodeLocked, now)
	if !s.securityAlertsOn(ctx) || s.pg == nil {
		return
	}
	var number int64
	if err := s.pg.QueryRow(ctx, `SELECT number FROM orders WHERE id = $1`, orderID).Scan(&number); err != nil {
		return
	}
	s.sec.enqueue("code:"+orderID, codeLockAlertText(number), now)
}

// noteMockedProof **صورةُ تسليمٍ بموقعٍ مزيَّف.**
func (s *Server) noteMockedProof(ctx context.Context, orderID, driverID string) {
	now := time.Now()
	s.sec.record(secGPSMocked, now)
	if !s.securityAlertsOn(ctx) || s.pg == nil {
		return
	}
	var number int64
	var name string
	if err := s.pg.QueryRow(ctx, `
		SELECT o.number, COALESCE(u.full_name, '')
		FROM orders o LEFT JOIN users u ON u.id = $2::uuid
		WHERE o.id = $1`, orderID, driverID).Scan(&number, &name); err != nil {
		return
	}
	s.sec.enqueue("gps:"+driverID+":"+orderID, mockedAlertText(name, number), now)
}

// authzRefusalCodes **رموزُ رفض التخويل** — وما سواها من ٤٠٣ قرارُ عمل.
var authzRefusalCodes = []string{"forbidden", "not_your_order"}

// noteRefusal **يعدّ محاولةَ فتح ما ليس له** — يُنادى من `countErrors`.
func (s *Server) noteRefusal(r *http.Request, code int, head []byte, route string) {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || token == "" {
		return
	}
	switch code {
	case http.StatusForbidden:
		hit := false
		for _, c := range authzRefusalCodes {
			if strings.Contains(string(head), `"code":"`+c+`"`) {
				hit = true
				break
			}
		}
		if !hit {
			return
		}
	case http.StatusNotFound:
		if !strings.Contains(route, "{") || !strings.Contains(string(head), `"code":"not_found"`) {
			return
		}
	default:
		return
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(token))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(r.Method + " " + r.URL.Path))
	s.sec.recordDistinct(secForbidden, h.Sum64(), time.Now())
}

// ── الراصد ──────────────────────────────────────────────────────────

// securityChecks **قفزتا الأمان للنبضة** — وتمرّان بـ`alertBook`.
func (s *Server) securityChecks(ctx context.Context, now time.Time) []monCheck {
	if !s.monSetting(ctx, "monitoring.security_alerts") {
		return nil
	}
	login := monCheck{Key: "sec_login",
		OKText: "✅ رجع طبيعي: محاولات الدخول الغلط خفّت."}
	if limit := s.monInt(ctx, "monitoring.login_fail_spike", 20); limit > 0 {
		c := s.sec.countSince(secLoginFailed, now.Add(-spikeWindow))
		login.Known, login.Bad = true, c >= int(limit)
		login.Text = "🚨 محاولات دخول غلط كتير: " + arNum(int64(c)) + " بعشر دقايق\n" +
			"في حدا عم يجرّب كلمات سر. الأرقام اللي بتوصل للحد بتنقفل لحالها."
	}
	forbid := monCheck{Key: "sec_forbidden",
		OKText: "✅ رجع طبيعي: محاولات فتح بيانات الغير خفّت."}
	if limit := s.monInt(ctx, "monitoring.forbidden_spike", 30); limit > 0 {
		c := s.sec.countSince(secForbidden, now.Add(-spikeWindow))
		forbid.Known, forbid.Bad = true, c >= int(limit)
		forbid.Text = "🚨 محاولات فتح بيانات الغير كتير: " + arNum(int64(c)) + " بعشر دقايق\n" +
			"في حساب عم يحاول يفتح طلبات أو بيانات مو إلو. المنصة عم ترفض، بس لازم ينشاف مين."
	}
	return []monCheck{login, forbid}
}

// drainSecurityAlerts **يرسل التنبيهاتِ الفوريّة المنتظرة.**
func (s *Server) drainSecurityAlerts(ctx context.Context, now time.Time) {
	if !s.securityAlertsOn(ctx) {
		s.sec.dropQueue()
		return
	}
	s.sec.drain(now, func(text string) bool { return s.alertOwner(ctx, text) })
}

// flushSecurity **يكتب العدَّ اليوميّ في الجدول** — دفعةً واحدة.
func (s *Server) flushSecurity(ctx context.Context) error {
	p := s.sec.take()
	if len(p) == 0 {
		return nil
	}
	if s.pg == nil {
		s.sec.putBack(p)
		return nil
	}
	for k, n := range p {
		if _, err := s.pg.Exec(ctx, `
			INSERT INTO security_events_daily (day, kind, count, updated_at)
			VALUES ($1::date, $2, $3, now())
			ON CONFLICT (day, kind) DO UPDATE SET
				count = security_events_daily.count + EXCLUDED.count,
				updated_at = now()`, k.Day, k.Kind, n); err != nil {
			s.sec.putBack(p)
			return err
		}
		delete(p, k)
	}
	return nil
}

// securityDay **عدُّ يومٍ بتوقيت دمشق** — من الجدول بعد كتابة المعلَّق.
func (s *Server) securityDay(ctx context.Context, day string) (map[string]int64, error) {
	if err := s.flushSecurity(ctx); err != nil {
		return nil, err
	}
	rows, err := s.pg.Query(ctx,
		`SELECT kind, count FROM security_events_daily WHERE day = $1::date`, day)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var k string
		var n int64
		if err := rows.Scan(&k, &n); err != nil {
			return nil, err
		}
		out[k] = n
	}
	return out, rows.Err()
}

// securityMorningLine **سطرُ الأمان في التقرير** — من عدّ مبارح.
//
// **و«ولا محاولة مشبوهة» حين لا شيءَ مطلقاً** — ولو كان الأربعةُ صفراً
// وفي «فتح بيانات الغير» شيء، **يُذكر هو** ولا يُقال «ولا شي».
func securityMorningLine(c map[string]int64) string {
	failed, locked, code, gps, forbid := c[secLoginFailed], c[secLoginLocked], c[secCodeLocked], c[secGPSMocked], c[secForbidden]
	if failed == 0 && locked == 0 && code == 0 && gps == 0 && forbid == 0 {
		return "الأمان مبارح: ولا محاولة مشبوهة ✅"
	}
	line := "الأمان مبارح: " + arNum(failed) + " محاولة دخول غلط · " + arNum(locked) + " حساب انقفل · " +
		arNum(code) + " كود تسليم انقفل · " + arNum(gps) + " موقع مزيّف"
	if forbid > 0 {
		line += " · " + arNum(forbid) + " محاولة فتح بيانات الغير"
	}
	return line
}

// securityMorning **سطرُ الأمان لصباح `now`** — يومُ دمشق السابق.
func (s *Server) securityMorning(ctx context.Context, now time.Time) (string, bool) {
	if s.pg == nil {
		return "", false
	}
	l := now.In(platform.Location())
	y := time.Date(l.Year(), l.Month(), l.Day()-1, 12, 0, 0, 0, platform.Location()).Format("2006-01-02")
	c, err := s.securityDay(ctx, y)
	if err != nil {
		return "الأمان مبارح: ما قدرت أقرأ العدّ.", true
	}
	return securityMorningLine(c), true
}

// routePattern **نمطُ مسار الطلب** — فارغٌ إن لم يُطابَق.
func routePattern(r *http.Request) string {
	if rc := chi.RouteContext(r.Context()); rc != nil {
		return rc.RoutePattern()
	}
	return ""
}

// ── لفحوص `qa` وحدَها ──────────────────────────────────────────────

// MonitorTickForTest **نبضةُ راصدٍ واحدة** — مع إرسال تنبيهات الأمان المنتظرة.
func (s *Server) MonitorTickForTest(ctx context.Context, now time.Time) {
	s.monitorTick(ctx, now)
}

// MorningReportForTest **نصُّ تقرير الصباح** — كما يُرسَل.
func (s *Server) MorningReportForTest(ctx context.Context, now time.Time) string {
	return s.morningReport(ctx, now)
}
