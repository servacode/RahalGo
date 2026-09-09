package server

// ══════════════════════════════════════════════════════════════════════
// **أقلُّ صلاحيّةٍ في الحقول لا في الأبواب** — `XG-42`
// ══════════════════════════════════════════════════════════════════════
//
// # المسألة
//
// **قسمةُ القدرات تحرس البابَ لا الحمولة** (`ADG-1`/`ADG-2`). **فمن
// ملك `users.read` ليجد حساباً لقيدٍ ماليٍّ نال هاتفَه أيضاً** —
// **ولا يتّصل بأحد.**
//
// **وإخفاءُ الحقل في الشاشة ليس حمايةً**: **الردُّ الخام هو الحقيقة**،
// ومن نادى النقطةَ بيده قرأ ما فيها.
//
// # ولماذا وسيطٌ واحدٌ لا شرطٌ في كلّ معالِج
//
// **`if role == "finance" { delete phone }` مبعثرةً تشيخ** — **ومسارٌ
// جديدٌ يُكتب غداً لا يعرفها.** **وثلاثةَ عشرَ معالِجاً تردّ هاتفاً
// اليوم**، ولكلٍّ بنيةُ ردٍّ خاصّةٌ به.
//
// **فالتشكيلُ عند حدّ الخروج**: **يُقرأ الجسمُ بعد كتابته ويُنقّى** —
// **فيشمل القوائمَ والتفاصيلَ والأعشاشَ معاً بلا أن يُعاد تصميمُ
// حمولةٍ واحدة.**
//
// # ولا اسمَ دورٍ في الشرط
//
// **قدرةٌ تُحسب من الحقيقة الموثوقة في كلّ طلب** (`ADG-1` · `R15`) —
// **فدورٌ جديدٌ يصنعه الإداريُّ يأخذ حكمَه من قدراته**، **ونزعُ القدرة
// يسري عند أوّل نداءٍ بلا خروجٍ ولا دخول.**
//
// # والحذفُ لا التفريغ
//
// **حقلٌ يُردّ فارغاً يكذب**: القارئُ لا يعرف أهو محجوبٌ أم لا رقمَ
// لصاحبه. **والحذفُ يقول الحقيقةَ ولا يُسرّب.**
//
// **وحقولُ الردّ في عقد الـAPI اختياريّةٌ في القارئ**: الويبُ
// `Record<string, unknown>` وأندرويد `@Serializable` بقيمٍ افتراضيّة —
// **فمفتاحٌ غائبٌ لا يكسر مُفكِّكاً.**

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/authz"
)

// shapeResponse **يُنقّي جسمَ الردّ بحسب قدرة قارئه.**
//
// **ويقع بعد التخويل** — فالقدراتُ في السياق من الحقيقة الموثوقة.
func (s *Server) shapeResponse(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// **وما يملك القدرةَ يمرّ بلا نسخٍ ولا فكِّ ترميز** — **ولا
		// يُدفَع ثمنُ الحراسة لمن لا يحتاجها.**
		missing := make([]string, 0, len(authz.FieldPolicy))
		for field, need := range authz.FieldPolicy {
			if !s.hasCapability(r, need) {
				missing = append(missing, field)
			}
		}
		if len(missing) == 0 {
			next.ServeHTTP(w, r)
			return
		}

		buf := &shapingWriter{ResponseWriter: w}
		next.ServeHTTP(buf, r)

		// **ولا يُمَسّ ما ليس JSON** — التصديرُ ملفٌّ وله حارسُه
		// (`users.export` · `finance.export`)، **والوسائطُ بايتات.**
		ct := w.Header().Get("Content-Type")
		if !strings.Contains(ct, "application/json") {
			buf.flushRaw()
			return
		}
		var body any
		if err := json.Unmarshal(buf.buf.Bytes(), &body); err != nil {
			buf.flushRaw()
			return
		}
		stripFields(body, missing)
		out, err := json.Marshal(body)
		if err != nil {
			buf.flushRaw()
			return
		}
		// **والسطرُ الجديدُ كما تكتبه `Encode`** — **فلا يفترق ردٌّ
		// مُشكَّلٌ عن غيره بمحرف** (وهو عقدُ `XG-33` في إعادة الردود).
		out = append(out, byte(10))
		w.Header().Set("Content-Length", "")
		w.Header().Del("Content-Length")
		w.WriteHeader(buf.status())
		_, _ = w.Write(out)
	})
}

// stripFields يمشي في الجسم كلِّه ويحذف المفاتيحَ المحميّة.
//
// **وفي العمق لا في السطح**: `order.customer_phone` كما `user.phone` —
// **وحقلٌ يُحذف من السطح ويبقى في عشٍّ داخليٍّ ليس محذوفاً.**
func stripFields(n any, fields []string) {
	switch v := n.(type) {
	case map[string]any:
		for _, f := range fields {
			delete(v, f)
		}
		for _, val := range v {
			stripFields(val, fields)
		}
	case []any:
		for _, e := range v {
			stripFields(e, fields)
		}
	}
}

// shapingWriter يحتجز الجسمَ ليُنقّى قبل أن يخرج.
//
// **ولا يكتب ترويسةً حتّى يُعرَف الجسم** — **فحالُ الردّ تُكتب مرّةً.**
type shapingWriter struct {
	http.ResponseWriter
	buf     bytes.Buffer
	code    int
	flushed bool
}

func (w *shapingWriter) WriteHeader(code int) { w.code = code }

func (w *shapingWriter) Write(b []byte) (int, error) {
	if w.flushed {
		return w.ResponseWriter.Write(b)
	}
	// **وما ليس JSON يمرّ بلا احتجاز** — **وملفُّ تصديرٍ يُحتجَز في
	// الذاكرة كلُّه عبءٌ لا حراسة**، وله حارسُه عند الباب.
	if ct := w.Header().Get("Content-Type"); ct != "" &&
		!strings.Contains(ct, "application/json") {
		w.flushed = true
		w.ResponseWriter.WriteHeader(w.status())
		return w.ResponseWriter.Write(b)
	}
	return w.buf.Write(b)
}

func (w *shapingWriter) status() int {
	if w.code == 0 {
		return http.StatusOK
	}
	return w.code
}

// flushRaw يُخرج ما احتُجز كما هو — **حين لا يُنقّى.**
func (w *shapingWriter) flushRaw() {
	if w.flushed {
		return
	}
	w.flushed = true
	w.ResponseWriter.WriteHeader(w.status())
	_, _ = w.ResponseWriter.Write(w.buf.Bytes())
}
