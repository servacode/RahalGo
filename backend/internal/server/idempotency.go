package server

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ══════════════════════════════════════════════════════════════════════
//  **منعُ التكرار — فعلٌ واحدٌ مهما أُعيد إرساله**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١١، البند الرابع من خطّة تطبيق أندرويد.)
//
// # ولماذا لم يُحتَج إليه قبل الهاتف
//
// **المتصفّحُ يُعطّل الزرَّ حتّى يعود الرد** — والشبكةُ ثابتةٌ غالباً.
// **والهاتفُ يفقد الشبكةَ في منتصف النداء**، والتطبيقُ لا يعلم أوصل
// الطلبُ أم لا، **فيعيد الإرسال** — وينشأ طلبان وخصمان.
//
// # وما هو محميٌّ أصلاً — فلا يُلَفّ بهذا
//
// **انتقالاتُ الحالة**: `مُسلَّم` لا ينتقل إلى `مُسلَّم`، والصفُّ مقفولٌ
// بـ`FOR UPDATE`. **فالنداءُ الثاني يُردّ بخطأٍ ولا يكتب مالاً.**
//
// **وحمايةٌ فوق حماية تُخفي عطباً**: من رأى المفتاحَ ظنّ الانتقالَ محميّاً
// به، **فلو كُسر الحارسُ الحقيقيُّ يوماً لم ينتبه أحد.**
//
// # وما ليس محميّاً — وهو ما يُلَفّ
//
// **إنشاءُ الطلب** (أهمُّ فعلٍ في تطبيق الزبون)، **وإضافةُ الرصيد
// اليدويّة**، **وقرارُ السحب**، **وتسويةُ نقد السائق**، **ومنحُ المكافأة.**

// idempotencyHeader الترويسةُ التي يرسلها العميل.
//
// **والاسمُ معياريّ** (`Idempotency-Key`) لا خاصٌّ بنا — تعرفه كلُّ مكتبةٍ
// وكلُّ مطوّرٍ يقرأ الشيفرة، **واسمٌ مخترَعٌ يحتاج شرحاً في كلّ مرّة.**
const idempotencyHeader = "Idempotency-Key"

const (
	// maxIdempotencyKey **سقفُ طول المفتاح** — المعياريُّ `UUID` بستّةٍ
	// وثلاثين حرفاً، **ومئتان سقفٌ سخيٌّ يمنع أن يُملأ الجدولُ بنصٍّ حرّ.**
	maxIdempotencyKey = 200

	// idempotencyTTL **عمرُ المفتاح.**
	//
	// **ويوم لا ساعة**: هاتفٌ فقد الشبكةَ ليلاً قد لا يعود إلّا صباحاً،
	// **وطابورٌ يُفرَّغ بعد اثنتي عشرة ساعةً يجب ألّا يُنشئ طلباتٍ ثانيةً.**
	//
	// **ولا شهر**: الجدولُ ينتفخ بلا فائدة، **ومن أعاد فعلاً بعد يومٍ
	// يقصده فعلاً.**
	idempotencyTTL = 24 * time.Hour
)

// captureWriter يلتقط الردَّ ليُحفظ — **ويُمرّره كما هو إلى صاحبه.**
type captureWriter struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (w *captureWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *captureWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// idempotent يلفّ نقطةً تكتب مالاً فتصير آمنةً من الإعادة.
//
// **وبلا ترويسةٍ يمرّ النداءُ كما كان** — الويبُ لا يرسلها، **ونقطةٌ ترفض
// من لا يرسل مفتاحاً تكسر كلَّ شاشةٍ تعمل اليوم.** والحمايةُ لمن طلبها.
func (s *Server) idempotent(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimSpace(r.Header.Get(idempotencyHeader))
		uid := userIDFrom(r)
		if key == "" || uid == "" {
			next(w, r)
			return
		}
		if len(key) > maxIdempotencyKey {
			s.respondErr(w, errValidation)
			return
		}
		endpoint := r.Method + " " + r.URL.Path

		// **الحجزُ أوّلاً ثمّ التنفيذ** — لا العكس.
		//
		// **ولو نُفّذ ثمّ حُجز** لَمرّ نداءان متزامنان معاً: كلاهما لا يجد
		// مفتاحاً فينفّذان، **ثمّ يتصادمان عند الحفظ وقد كُتب المالُ مرّتين.**
		claimed, err := s.claimIdempotency(r.Context(), uid, endpoint, key)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		if !claimed {
			s.replayIdempotency(w, r.Context(), uid, endpoint, key)
			return
		}

		cw := &captureWriter{ResponseWriter: w}
		next(cw, r)

		// ══════════════════════════════════════════════════════════════
		// **والفاشلُ يُطلَق لا يُحفَظ**
		// ══════════════════════════════════════════════════════════════
		//
		// **ردٌّ بخطأٍ محفوظٌ يعني أنّ المحاولةَ الثانية تتلقّى الخطأَ
		// نفسَه إلى الأبد** — ومن فشل طلبُه لانقطاع قاعدةٍ لحظةً **لا
		// يستطيع أن يعيد أبداً.**
		//
		// **فيُحذف الحجزُ فيصير الطريقُ مفتوحاً**، والإعادةُ تُنفَّذ من
		// جديد. **وهذا هو المقصود: الحمايةُ من تكرار ما نجح لا من إعادة
		// ما فشل.**
		if cw.status >= 400 {
			s.releaseIdempotency(r.Context(), uid, endpoint, key)
			return
		}
		s.storeIdempotency(r.Context(), uid, endpoint, key, cw.status, cw.body.Bytes())
	}
}

// claimIdempotency يحجز المفتاح — **ويعيد `false` إن كان محجوزاً.**
func (s *Server) claimIdempotency(ctx context.Context, uid, endpoint, key string) (bool, error) {
	// **والتقليمُ مع الكتابة** — **ومهمّةٌ ليليّةٌ تُنسى أو تتعطّل فينتفخ
	// الجدولُ بصمت.** وتعثّرُه لا يمنع الحجز.
	if _, err := s.pg.Exec(ctx,
		`DELETE FROM idempotency_keys WHERE created_at < now() - $1::interval`,
		idempotencyTTL.String()); err != nil {
		s.logger.Warn("منعُ التكرار: تعذّر التقليم", "error", err)
	}
	tag, err := s.pg.Exec(ctx, `
		INSERT INTO idempotency_keys (user_id, endpoint, key)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, endpoint, key) DO NOTHING`, uid, endpoint, key)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// replayIdempotency يردّ ما حُفظ — **أو يقول «ما زال يُنفَّذ».**
func (s *Server) replayIdempotency(w http.ResponseWriter, ctx context.Context, uid, endpoint, key string) {
	var done bool
	var status int
	var body string
	// **و`COALESCE` لأنّ الردَّ فارغٌ ما دام الحجزُ لم يكتمل** — ونداءان
	// متزامنان يقع أحدُهما على ذلك، **فيسقط المسحُ بخمسمئة بدل أن يقول
	// «قيد التنفيذ»**، ويظنّ صاحبُه المنصّةَ معطوبة.
	err := s.pg.QueryRow(ctx, `
		SELECT done, status_code, COALESCE(response, '') FROM idempotency_keys
		WHERE user_id = $1 AND endpoint = $2 AND key = $3`, uid, endpoint, key).
		Scan(&done, &status, &body)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if !done {
		// **نداءان متزامنان بالمفتاح نفسِه** — الثاني يجد حجزاً لم يكتمل.
		//
		// **و409 لا 500**: الحالةُ سليمةٌ ومؤقّتة، **ومن رأى خمسمئة ظنّ
		// المنصّةَ معطوبةً وأعاد بمفتاحٍ جديد** — وذاك بالضبط ما نمنعه.
		httpx.Error(w, &httpx.AppError{
			Status: http.StatusConflict, Code: "in_progress",
			MessageKey: "errors.in_progress",
		})
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	// **وترويسةٌ تقول إنّه معاد** — يقرؤها من يشخّص، **ولا تُغيّر شيئاً عند
	// من لا يعرفها.**
	w.Header().Set("Idempotent-Replay", "true")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func (s *Server) storeIdempotency(ctx context.Context, uid, endpoint, key string, status int, body []byte) {
	if _, err := s.pg.Exec(ctx, `
		UPDATE idempotency_keys SET done = true, status_code = $4, response = $5
		WHERE user_id = $1 AND endpoint = $2 AND key = $3`,
		uid, endpoint, key, status, string(body)); err != nil {
		// **ولا يُسقط الردَّ** — الفعلُ وقع ونجح، **والفشلُ هنا يعني أنّ
		// إعادةً لاحقةً ستُنفّذ ثانيةً**، وهو أهونُ من إبطال ما نجح.
		s.logger.Error("منعُ التكرار: تعذّر حفظُ الرد", "endpoint", endpoint, "error", err)
	}
}

func (s *Server) releaseIdempotency(ctx context.Context, uid, endpoint, key string) {
	if _, err := s.pg.Exec(ctx, `
		DELETE FROM idempotency_keys
		WHERE user_id = $1 AND endpoint = $2 AND key = $3 AND done = false`,
		uid, endpoint, key); err != nil {
		s.logger.Warn("منعُ التكرار: تعذّر إطلاقُ مفتاحٍ فاشل", "error", err)
	}
}
