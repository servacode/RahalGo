// Package obs **عدّاداتُ التشغيل — تجميعٌ لا تجسّس** (دورة ٧٠أ).
//
// ══════════════════════════════════════════════════════════════════════
// **ولماذا عدّادٌ في الذاكرة لا مكدّسُ رصدٍ كامل**
// ══════════════════════════════════════════════════════════════════════
//
// **والمستودعُ لا يستعمل Prometheus ولا OpenTelemetry ولا Grafana** —
// **وإدخالُ مكدّسٍ كاملٍ لأجل أوّل قياسٍ داخليٍّ يبني بنيةً تحتيّةً قبل
// أن يُعرف ما يُقاس.** **وعدّادٌ ذرّيٌّ يُقرأ بـ`curl` واحدة يصلح
// منبعاً لـSentinel لاحقاً بلا تبديل.**
//
// # وما لا يدخل هنا أبداً
//
// **ولا معرِّفَ إنسانٍ في هذه الحزمة** — **لا مستخدمٌ ولا جلسةٌ ولا
// هاتفٌ ولا رمزٌ ولا نصُّ خطأٍ خام.** **ونصُّ الخطأ هو ما يُهرّب
// الأسماءَ والعناوين**، فالأسبابُ **مفاتيحُ معجمٍ مغلقٍ** لا نصوصٌ
// حرّة.
//
// **وعدّادٌ لا يُقسَّم على إنسان**: **رقمٌ واحدٌ لكلّ سببٍ للمنصّة
// كلِّها** — **ومن قسّمه على المستخدم بنى تتبّعاً باسم الرصد.**
//
// # والعدُّ يُصفَّر بإقلاع المحرّك
//
// **وهذا مقصود**: **العدّادُ يقول «كم منذ الإقلاع»**، ومعه `uptime`
// **فيُشتقّ المعدّل.** **ولا يُدَّعى تاريخٌ لا يملكه.**
package obs

import (
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// WSReason **سببُ حصيلة مصافحة البثّ** — معجمٌ مغلق.
//
// **وثلاثةُ أسبابٍ مختلفةٍ كانت تردّ ٤٠١ واحدةً** (دورةُ ٦٩ج)، **ولا
// يُميَّز بينها إلّا باستنتاجٍ من إيقاعِ إعادةِ الوصل.** **وهذه
// المفاتيحُ تُنهي الاستنتاج.**
type WSReason string

const (
	// WSSuccess **مصافحةٌ قُبلت** — وبها يُعرف المقام لا العدد وحدَه.
	WSSuccess WSReason = "ws_auth_success"
	// WSExpiredAccess **رمزٌ انقضى عمرُه** — **وهو وحدَه ما يشفيه
	// تجديد.** **وتكرارُ الرمز نفسِه لا يشفيه أبداً.**
	WSExpiredAccess WSReason = "ws_auth_expired_access"
	// WSInvalidToken **مشوَّهٌ أو توقيعٌ باطلٌ أو غائب.**
	WSInvalidToken WSReason = "ws_auth_invalid_token"
	// WSRevokedSession **عائلةٌ أُبطلت** — خروجٌ أو تبديلُ حساب.
	WSRevokedSession WSReason = "ws_auth_revoked_session"
	// WSForbidden **حسابٌ محظورٌ أو محذوف** — **وردُّه ٤٠٣ لا ٤٠١**
	// (`D14`)، فيُعدّ على حدة.
	WSForbidden WSReason = "ws_auth_forbidden"
	// WSBackendUnavailable **تعذّر سؤالُ الجلسة** — ٥٠٣، **وهي وحدَها
	// عرَضُ بنيةٍ لا عرَضُ عميل.**
	WSBackendUnavailable WSReason = "ws_auth_backend_unavailable"
)

// wsReasons **ترتيبُ العرض** — ثابتٌ فلا يرقص الردّ بين نداءين.
var wsReasons = []WSReason{
	WSSuccess, WSExpiredAccess, WSInvalidToken,
	WSRevokedSession, WSForbidden, WSBackendUnavailable,
}

var wsCounters = func() map[WSReason]*atomic.Int64 {
	m := make(map[WSReason]*atomic.Int64, len(wsReasons))
	for _, r := range wsReasons {
		m[r] = &atomic.Int64{}
	}
	return m
}()

// WSAuth **يعدّ حصيلةَ مصافحة.**
//
// **ولا يُنادى بسببٍ خارج المعجم** — **ومن أضاف سبباً نسي تسجيلَه
// هنا يعدّ في العدم.** فيُهمَل المجهولُ صراحةً ولا يُخترَع له صفّ.
func WSAuth(r WSReason) {
	if c, ok := wsCounters[r]; ok {
		c.Add(1)
	}
}

// PushOutcome **حصيلةُ محاولة دفع.**
type PushOutcome string

const (
	// PushAttempted **رسالةٌ عُرضت على الناقل.**
	PushAttempted PushOutcome = "push_attempted"
	// PushSent **قبلها المزوّد.**
	PushSent PushOutcome = "push_sent"
	// PushFailed **ردَّها المزوّد أو تعذّر بلوغُه.**
	PushFailed PushOutcome = "push_failed"
	// PushDeadToken **رمزٌ رُفض نهائيّاً** — جهازٌ حُذف منه التطبيق.
	PushDeadToken PushOutcome = "push_dead_token"
	// PushNoDevice **لا جهازَ مسجَّلاً لهذا الحساب** — **وهي ليست
	// عطباً**: حسابٌ لم يُفتح تطبيقُه بعد الدخول.
	PushNoDevice PushOutcome = "push_no_device"
)

var pushOutcomes = []PushOutcome{
	PushAttempted, PushSent, PushFailed, PushDeadToken, PushNoDevice,
}

var pushCounters = func() map[PushOutcome]*atomic.Int64 {
	m := make(map[PushOutcome]*atomic.Int64, len(pushOutcomes))
	for _, o := range pushOutcomes {
		m[o] = &atomic.Int64{}
	}
	return m
}()

// Push **يعدّ حصيلةَ دفعٍ** — بمقدارٍ، فالإرسالُ لجمعٍ لا لواحد.
func Push(o PushOutcome, n int) {
	if n <= 0 {
		return
	}
	if c, ok := pushCounters[o]; ok {
		c.Add(int64(n))
	}
}

var started = time.Now()

// Snapshot **لقطةُ العدّادات** — أرقامٌ لا نصوص.
type Snapshot struct {
	UptimeSeconds int64            `json:"uptime_seconds"`
	WSAuth        map[string]int64 `json:"ws_auth"`
	Push          map[string]int64 `json:"push"`
	// Clients **نداءاتٌ بحسب «نوعُ التطبيق:نسخته»** منذ الإقلاع.
	Clients map[string]int64 `json:"clients"`
	// ClientsDropped **كم مفتاحاً أُهمل بعد السقف** — **ورقمٌ غيرُ
	// صفرٍ يعني أنّ الخريطةَ ناقصة، فلا تُقرأ على أنّها تامّة.**
	ClientsDropped int64 `json:"clients_dropped"`
}

// Take **يقرأ اللقطة.**
//
// **ولا قفلَ**: **كلُّ عدّادٍ ذرّيٌّ بذاته**، **واللقطةُ ليست لحظةً
// واحدةً بالضبط** — وهذا مقبولٌ في عدّادٍ تشغيليّ، **ولا يُدَّعى غيرُه.**
func Take() Snapshot {
	s := Snapshot{
		UptimeSeconds: int64(time.Since(started).Seconds()),
		WSAuth:        make(map[string]int64, len(wsReasons)),
		Push:          make(map[string]int64, len(pushOutcomes)),
	}
	for _, r := range wsReasons {
		s.WSAuth[string(r)] = wsCounters[r].Load()
	}
	for _, o := range pushOutcomes {
		s.Push[string(o)] = pushCounters[o].Load()
	}
	s.Clients = clientsSnapshot()
	s.ClientsDropped = clientDropped.Load()
	return s
}

// Reset **يُصفّر العدّادات** — **للاختبار وحدَه.**
//
// **ولا مسارَ إنتاجٍ يناديها**: **عدّادٌ يُصفَّر بنداءٍ من خارج يصير
// رقماً لا يُصدَّق.**
func Reset() {
	for _, r := range wsReasons {
		wsCounters[r].Store(0)
	}
	for _, o := range pushOutcomes {
		pushCounters[o].Store(0)
	}
	clientMu.Lock()
	clientCounts = map[string]int64{}
	clientMu.Unlock()
	clientDropped.Store(0)
}

// ══════════════════════════════════════════════════════════════════════
// **نسخُ العملاء — تجميعاً بلا جهازٍ ولا إنسان** (دورة ٧٠أ)
// ══════════════════════════════════════════════════════════════════════
//
// **ودورةُ ٦٩ج لم تستطع أن تقول أيُّ تطبيقٍ يطرق البابَ** — **لأنّ
// عقدَ سجلّ الوصول يحذف خريطةَ الترويسات جملةً، وهو صواب.** **فحُسم
// الاستدلالُ من إيقاع إعادةِ الوصل**، وذاك استنتاجٌ لا قياس.
//
// **والمحرّكُ يرى الترويستين أصلاً** (`X-RahalGo-Client` و
// `X-RahalGo-Version`) — **ويقرؤهما في موضعٍ واحدٍ لكلّ نداء**
// (`min_version.go`). **فالعدُّ هناك لا يفتح باباً جديداً.**
//
// # ولا يُوسَّع سجلُّ البوّابة
//
// **ولا تُعاد ترويسةٌ إلى سجلّ Caddy** — **الخصوصيّةُ المُثبَتة في
// ٦٩أ-ر١ لا تُضعَّف لأجل إحصاء.** **والعدُّ في المحرّك حيث الترويسةُ
// مقروءةٌ أصلاً.**
//
// # والتعدادُ محدود
//
// **ومفتاحٌ يأتي من عميلٍ يُصدَّق يفجّر الذاكرة**: **من أرسل نسخةً
// عشوائيّةً في كلّ نداءٍ بنى خريطةً بلا قعر.** **فالنوعُ من معجمٍ
// مغلق، والنسخةُ رقمٌ موجب، والخريطةُ لها سقف** — **وما بعده يُهمَل
// ولا يُكذَب فيه: `clients_dropped` يقول كم أُهمل.**

// maxClientKeys **سقفُ خريطة النسخ** — أربعةُ تطبيقاتٍ × نسخٍ معقولة.
const maxClientKeys = 64

var (
	clientMu      sync.Mutex
	clientCounts  = map[string]int64{}
	clientDropped atomic.Int64
)

// knownClientKind **معجمُ أنواع العملاء المغلق** — **ومجهولُه لا
// يُعدّ.** (الترويسةُ تصل `rahalgo-customer` وأخواتِها.)
var knownClientKind = map[string]bool{
	"customer": true, "driver": true, "merchant": true, "rep": true,
}

// Client **يعدّ نداءً من تطبيقٍ بنوعه ونسخته.**
//
// **ولا شيءَ يميّز جهازاً ولا إنساناً** — **«زبونٌ نسخةُ ١١» رقمٌ
// للمنصّة كلِّها.**
func Client(kind string, version int) {
	if !knownClientKind[kind] || version <= 0 {
		return
	}
	key := kind + ":" + strconv.Itoa(version)
	clientMu.Lock()
	defer clientMu.Unlock()
	if _, ok := clientCounts[key]; !ok && len(clientCounts) >= maxClientKeys {
		clientDropped.Add(1)
		return
	}
	clientCounts[key]++
}

// clientsSnapshot **نسخةٌ من الخريطة** — **ولا يُعار المرجعُ نفسُه**:
// خريطةٌ تُقرأ وتُكتب معاً تُسقط العمليّة.
func clientsSnapshot() map[string]int64 {
	clientMu.Lock()
	defer clientMu.Unlock()
	out := make(map[string]int64, len(clientCounts))
	for k, v := range clientCounts {
		out[k] = v
	}
	return out
}
