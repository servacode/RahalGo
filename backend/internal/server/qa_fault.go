package server

// ══════════════════════════════════════════════════════════════════════
// **حاقنُ الأعطال — على التجهيز وحدَه، لطلبات QA فقط** (المسار C)
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٩-٢٢: «ابنِ حاقنَ أعطالٍ» لشهود صفوف المسار C التي
//  تحتاج ٥xx محقونةً أو تأخيراً/مهلةً على نقطةٍ بعينها.)
//
// # حدودُه — صارمة
//
//   ١ · **على التجهيز وحدَه**: كلُّ فحصٍ يبدأ بـ`qaStagingEnabled()` — في
//       الإنتاج المِعمارُ يمرّ بلا أثرٍ البتّة (fail-closed). و`qa/seed` (بابُ
//       التسليح) غيرُ مسجَّلٍ أصلاً في الإنتاج.
//   ٢ · **مقصورٌ على QA**: لا يُحقَن إلّا حين يكون صاحبُ الطلب زبونَ QA
//       (بالجلسة) **أو** يحمل الطلبُ ترويسةَ `X-QA-Fault: 1` (عميلُ QA). فلا
//       أثرَ على مستخدمٍ آخر.
//   ٣ · **حتميٌّ ويُنظَّف نفسَه**: كلُّ عطبٍ مُسلَّحٌ بعددِ إصاباتٍ (`remaining`،
//       افتراضُه ١) يُنزَع تلقائيّاً حين ينفد — فلا يبقى مسلَّحاً بعد الشاهد.
//       والحالُ الافتراضيّةُ دائماً **مطفأة** (خريطةٌ فارغة).
//   ٤ · **مسموعٌ**: كلُّ تسليحٍ وحقنٍ يُسجَّل (`logger.Warn`).
//   ٥ · **لا مساسَ بقاعدةٍ ولا بمال**: يردّ ٥xx أو يؤخّر فحسب — لا كتابة.

import (
	"net/http"
	"sync"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
)

// errQAFault **العطبُ المحقون** — رمزٌ صريحٌ يُميَّز في السجلّ، ورسالتُه
// عائلةُ «غيرُ متاحٍ مؤقّتاً» (مفتاحٌ موجود، لا i18n جديد).
var errQAFault = httpx.NewError(http.StatusServiceUnavailable, "qa_fault_injected", "errors.temporarily_unavailable")

// qaFaultModes الأنماطُ المسموحة — اثنان لا غير (لا نبني ما لا يُستعمل).
const (
	qaFaultError5xx = "error_5xx" // ردٌّ ٥٠٣ فوريّ
	qaFaultLatency  = "latency"   // تأخيرٌ `ms` ثمّ يمضي (تباطؤٌ/مهلة)
)

type qaFault struct {
	mode      string
	ms        int
	remaining int
}

type qaFaultStore struct {
	mu sync.Mutex
	m  map[string]qaFault
}

// qaFaults **المخزنُ الوحيد** — على مستوى الحزمة (خادمٌ واحدٌ للعمليّة)،
// **افتراضُه فارغٌ = مطفأ**. لا يُسلَّح إلّا عبر `qa/seed` على التجهيز.
var qaFaults = &qaFaultStore{m: map[string]qaFault{}}

func (st *qaFaultStore) arm(path, mode string, ms, count int) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if count <= 0 {
		count = 1
	}
	st.m[path] = qaFault{mode: mode, ms: ms, remaining: count}
}

func (st *qaFaultStore) clear(path string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if path == "" {
		st.m = map[string]qaFault{}
		return
	}
	delete(st.m, path)
}

func (st *qaFaultStore) armed() bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	return len(st.m) > 0
}

// take يأخذ عطبَ المسار إن وُجد، ويُنقص عدّادَه (ويُنزعه عند النفاد).
func (st *qaFaultStore) take(path string) (qaFault, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	f, ok := st.m[path]
	if !ok || f.remaining <= 0 {
		return qaFault{}, false
	}
	f.remaining--
	if f.remaining <= 0 {
		delete(st.m, path)
	} else {
		st.m[path] = f
	}
	return f, true
}

func (st *qaFaultStore) snapshot() map[string]any {
	st.mu.Lock()
	defer st.mu.Unlock()
	out := map[string]any{}
	for k, v := range st.m {
		out[k] = map[string]any{"mode": v.mode, "ms": v.ms, "remaining": v.remaining}
	}
	return out
}

// qaUIDcache معرّفُ زبون QA — يُقرأ مرّةً ويُخبَّأ (لا استعلامَ لكلّ طلب).
var qaUIDcache struct {
	mu sync.Mutex
	id string
}

func (s *Server) qaUserID(r *http.Request) string {
	qaUIDcache.mu.Lock()
	defer qaUIDcache.mu.Unlock()
	if qaUIDcache.id != "" {
		return qaUIDcache.id
	}
	phone, ok := identity.NormalizePhone(qaStagingPhone)
	if !ok {
		return ""
	}
	_ = s.pg.QueryRow(r.Context(), `SELECT id::text FROM users WHERE phone = $1`, phone).Scan(&qaUIDcache.id)
	return qaUIDcache.id
}

// qaFaultScoped **أهو طلبُ QA؟** — على التجهيز، وصاحبُه زبونُ QA أو يحمل
// ترويسةَ العطب. **لا يُحقَن على غيره أبداً.**
func (s *Server) qaFaultScoped(r *http.Request) bool {
	if !s.qaStagingEnabled() {
		return false
	}
	if r.Header.Get("X-QA-Fault") == "1" {
		return true
	}
	uid := userIDFrom(r)
	return uid != "" && uid == s.qaUserID(r)
}

// qaMaybeFault يطبّق العطبَ المسلَّحَ للمسار إن كان الطلبُ مقصوراً على QA.
// يُرجع true إن ردّ الردَّ بنفسه (٥xx) فيتوقّف المنادي؛ والتأخيرُ يمضي.
func (s *Server) qaMaybeFault(w http.ResponseWriter, r *http.Request) bool {
	if !s.qaStagingEnabled() || !qaFaults.armed() || !s.qaFaultScoped(r) {
		return false
	}
	f, ok := qaFaults.take(r.URL.Path)
	if !ok {
		return false
	}
	switch f.mode {
	case qaFaultError5xx:
		s.logger.Warn("QA fault injected (staging-only)", "path", r.URL.Path, "mode", f.mode)
		s.respondErr(w, errQAFault)
		return true
	case qaFaultLatency:
		s.logger.Warn("QA fault injected (staging-only)", "path", r.URL.Path, "mode", f.mode, "ms", f.ms)
		time.Sleep(time.Duration(f.ms) * time.Millisecond)
		return false
	}
	return false
}

// qaFaultMW **حارسٌ للطرق المصادَق عليها** (مجموعةُ الزبون) — يعرف صاحبَ
// الطلب فيقصر الحقنَ على زبون QA. **يمرّ بلا أثرٍ في الإنتاج** (الشرطُ الأوّل).
func (s *Server) qaFaultMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.qaMaybeFault(w, r) {
			return
		}
		next.ServeHTTP(w, r)
	})
}
