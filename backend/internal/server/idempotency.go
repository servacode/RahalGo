package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// errIdemReused **مفتاحٌ واحدٌ لجسمين مختلفين** (`CAF-02` · `13-029`).
//
// **إعادةُ محاولةٍ صادقةٌ تحمل الجسمَ نفسَه** — فبصمتُها تطابق. **ومفتاحٌ
// عاد بجسمٍ مختلفٍ ليس إعادة**: طلبٌ عُدّل ولم يُدوَّر مفتاحُه. **فيُرَدّ
// ولا يُنفَّذ ثانياً** — لا هو ولا ردُّ الأوّل يُنسَب إليه.
var errIdemReused = httpx.NewError(http.StatusConflict,
	"idempotency_key_reused", "errors.idempotency_key_reused")

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

	// ══════════════════════════════════════════════════════════════════
	// **مهلةُ الحيازة — ستّون ثانية**
	// ══════════════════════════════════════════════════════════════════
	//
	// **ومهلةُ الطلب في هذا الخادم ثلاثون ثانية** (`middleware.Timeout`
	// في `server.go`)، **ولا مسارَ من الستّة المحميّة مستثنىً منها**
	// (المستثنى: البثُّ والوسائطُ والإثباتُ ورفعُ الصور).
	//
	// **فضِعفُها حدٌّ لا يبلغه عاملٌ حيّ** — **ومن تجاوزه فقد مات معالجُه
	// أو قُتل.**
	//
	// **والسلامةُ لا تقوم عليها**: قفلُ الصفّ ورمزُ الملكيّة هما
	// الحارسان. **وهي تمنع البقاءَ إلى الأبد لا أكثر** — **وهي علّةُ
	// `C-06` بعينها: أربعٌ وعشرون ساعةَ حبسٍ لأنّ لا مهلةَ أصلاً.**
	//
	// **ولا تُجعل إعداداً تشغيليّاً**: **إعدادٌ يُغيَّر خطأً يفتح بابَ
	// السرقة أو الحبس**، ولا حاجةَ ميدانيّةً لتغييره.
	idempotencyLease = 60 * time.Second
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

		// ══════════════════════════════════════════════════════════
		// **بصمةُ الجسم — لبابَي الإنشاء وحدَهما** (`CAF-02`)
		// ══════════════════════════════════════════════════════════
		//
		// **ويُقرأ الجسمُ هنا ثمّ يُعادُ** — فالمعالجُ يقرؤه بعدَنا كما لو
		// لم يُمَسّ. **وما لا يُبصَم لا يُقرأ جسمُه** فلا يتغيّر مسارُه.
		var fp []byte
		if wantsFingerprint(endpoint) {
			// **بالحدّ نفسِه الذي يقرأ به المعالجُ** (`decode`: ‎1 MiB) —
			// **فما يتجاوزه يُرَدّ هنا كما يُرَدّ هناك**، ولا يُبصَم مقصوصاً.
			body, err := io.ReadAll(io.LimitReader(r.Body, (1<<20)+1))
			if err != nil || len(body) > (1<<20) {
				s.respondErr(w, errValidation)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			fp = fingerprintForEndpoint(endpoint, body)
		}

		// ══════════════════════════════════════════════════════════
		// **الحيازةُ على المَسبَح — مرئيّةٌ فوراً**
		// ══════════════════════════════════════════════════════════
		//
		// **ولو أُخّرت إلى داخل معاملة العمل لما رآها المتزامنُ**
		// (صفٌّ غيرُ مثبَّتٍ لا يُرى)، **فوقع تنفيذان.**
		claim, replay, err := s.acquireClaim(r.Context(), uid, endpoint, key, fp)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		if claim == nil {
			replay(w)
			return
		}

		ctx := context.WithValue(r.Context(), idemClaimKey, claim)
		cw := &captureWriter{ResponseWriter: w}
		next(cw, r.WithContext(ctx))

		// **والخطأُ يُطلق المفتاحَ محروساً** — انظر `releaseIdempotency`.
		if cw.status >= 400 {
			s.releaseIdempotency(r.Context(), claim)
		}
	}
}

// acquireClaim يحوز المطالبةَ أو يُرجع كيف تُعاد.
//
// **ثلاثُ نتائجَ لا رابع**: **حِيزت** (صفٌّ جديدٌ أو استردادٌ ذرّيّ) ·
// **ثبتت** فتُعاد نتيجتُها · **حيّةٌ لغيرنا** فـ`409`.
func (s *Server) acquireClaim(ctx context.Context, uid, endpoint, key string, fp []byte) (
	*idemClaim, func(http.ResponseWriter), error) {

	s.pruneIdempotency(ctx)

	token := uuid.New()
	tag, err := s.pg.Exec(ctx, `
		INSERT INTO idempotency_keys (user_id, endpoint, key, owner_token, lease_until, request_fingerprint)
		VALUES ($1, $2, $3, $4, now() + $5::interval, $6)
		ON CONFLICT (user_id, endpoint, key) DO NOTHING`,
		uid, endpoint, key, token, idempotencyLease.String(), fp)
	if err != nil {
		return nil, nil, err
	}
	if tag.RowsAffected() == 1 {
		return &idemClaim{uid: uid, endpoint: endpoint, key: key, token: token}, nil, nil
	}

	// ══════════════════════════════════════════════════════════════
	// **حرسُ المحتوى قبل أيّ إعادة أو استرداد** (`CAF-02`)
	// ══════════════════════════════════════════════════════════════
	//
	// **صفٌّ قائمٌ بالمفتاح نفسِه — أبصمتُه هي بصمتُنا؟** **فإن اختلفتا
	// فمفتاحٌ واحدٌ لجسمين**: لا استردادَ ولا إعادةَ نتيجةٍ ولا تنفيذ —
	// `409 idempotency_key_reused`.
	//
	// **والتركةُ `NULL` تمرّ**: صفٌّ بلا بصمة (من قبل الهجرة، أو جسمٌ لم
	// يُبصَم) **لا حكمَ عليه** — فيُعامَل كما كان. **وطلبُنا بلا بصمةٍ
	// (`fp == nil`) لا يحكم على أحد** — فلا مقارنة.
	if fp != nil {
		var stored []byte
		err := s.pg.QueryRow(ctx, `
			SELECT request_fingerprint
			  FROM idempotency_keys
			 WHERE user_id = $1 AND endpoint = $2 AND key = $3`,
			uid, endpoint, key).Scan(&stored)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			// **مُحيت بين الإدراج والقراءة** — يُعاد النداءُ فيحوزها.
			return nil, func(w http.ResponseWriter) {
				s.respondErr(w, errIdemLost)
			}, nil
		case err != nil:
			return nil, nil, err
		case stored != nil && !bytes.Equal(stored, fp):
			return nil, func(w http.ResponseWriter) {
				s.respondErr(w, errIdemReused)
			}, nil
		}
	}

	// ══════════════════════════════════════════════════════════════
	// **استردادٌ ذرّيٌّ بشرطٍ واحدٍ لا يُساوَم**
	// ══════════════════════════════════════════════════════════════
	//
	// **`committed_at IS NULL`** — **فما ثبت لا يُعاد تنفيذُه أبداً.**
	//
	// **و`owner_token IS NOT NULL`** — **صفوفُ ما قبل الهجرة لا
	// تُستردّ**: ليس فيها ما يقول أوقع عملُها أم لا، **وحبسٌ يوماً أهونُ
	// من تكرارِ دفعة.**
	//
	// **وهذا `UPDATE` يأخذ قفلَ الصفّ** — **فإن كانت معاملةُ عملٍ حيّةٌ
	// تمسكه انتظر حتّى تنتهي**، ثمّ قرأ حقيقتَها: **ثبتت فلا شرطَ يمرّ،
	// أو ارتدّت فيؤخذ.** **ولا سرقةَ من عاملٍ حيّ.**
	var got int
	err = s.pg.QueryRow(ctx, `
		UPDATE idempotency_keys
		   SET owner_token = $4, lease_until = now() + $5::interval,
		       request_fingerprint = COALESCE(request_fingerprint, $6)
		 WHERE user_id = $1 AND endpoint = $2 AND key = $3
		   AND committed_at IS NULL
		   AND owner_token IS NOT NULL
		   AND lease_until < now()
		RETURNING 1`,
		uid, endpoint, key, token, idempotencyLease.String(), fp).Scan(&got)
	if err == nil {
		return &idemClaim{uid: uid, endpoint: endpoint, key: key, token: token}, nil, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, err
	}

	// **لم تُحَز** — فتُقرأ حقيقتُها وتُعاد.
	var committed *string
	var status int
	var body string
	if err := s.pg.QueryRow(ctx, `
		SELECT committed_at::text, status_code, COALESCE(response, '')
		  FROM idempotency_keys
		 WHERE user_id = $1 AND endpoint = $2 AND key = $3`,
		uid, endpoint, key).Scan(&committed, &status, &body); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// **مُحيت بين المحاولتين** — يُعاد النداءُ فيحوزها.
			return nil, func(w http.ResponseWriter) {
				s.respondErr(w, errIdemLost)
			}, nil
		}
		return nil, nil, err
	}
	if committed != nil {
		return nil, func(w http.ResponseWriter) {
			s.writeReplay(w, status, body)
		}, nil
	}
	return nil, func(w http.ResponseWriter) {
		httpx.Error(w, &httpx.AppError{
			Status: http.StatusConflict, Code: "in_progress",
			MessageKey: "errors.in_progress",
		})
	}, nil
}

// pruneIdempotency **يقلّم ما شاخ ولا يمسّ حيّاً.**
//
// **والاحتفاظُ كما هو** — أربعٌ وعشرون ساعةً من الإنشاء (`idempotencyTTL`)،
// **عقدٌ قائمٌ لا يُبدَّل في هذه الدورة.**
//
// **وشرطٌ زائدٌ يحرس الحيّ**: **ما مهلتُه لم تنتهِ لا يُمَسّ** — ولو كان
// صفُّه قديماً لسببٍ ما. **ومطالبةٌ تُمحى تحت عاملٍ يشتغل فرصةُ تكرار.**
func (s *Server) pruneIdempotency(ctx context.Context) {
	if _, err := s.pg.Exec(ctx, `
		DELETE FROM idempotency_keys
		 WHERE created_at < now() - $1::interval
		   AND (lease_until IS NULL OR lease_until < now())`,
		idempotencyTTL.String()); err != nil {
		s.logger.Warn("منعُ التكرار: تعذّر التقليم", "error", err)
	}
}

// releaseIdempotency يُطلق مطالبةً انتهت بخطأ — **بحراسة**.
//
// **والعقدُ القائمُ يُحفَظ**: ردٌّ ‎≥400 يحذف الصفَّ فيُعاد الطلبُ
// بحرّيّة. **لكنّ الحذفَ يشترط الملكيّة** — **ومالكٌ قديمٌ فقد مطالبتَه
// لا يمحو ما استعاده غيرُه**، فيُنفَّذ العملُ مرّتين أو يُحبَس صاحبُه.
//
// **ولا يُحذَف ما ثبت** — النتيجةُ المحفوظةُ تُعاد لمن سأل.
func (s *Server) releaseIdempotency(ctx context.Context, c *idemClaim) {
	if c == nil {
		return
	}
	if _, err := s.pg.Exec(ctx, `
		DELETE FROM idempotency_keys
		 WHERE user_id = $1 AND endpoint = $2 AND key = $3
		   AND committed_at IS NULL AND owner_token = $4`,
		c.uid, c.endpoint, c.key, c.token); err != nil {
		s.logger.Warn("منعُ التكرار: تعذّر إطلاقُ مفتاحٍ فاشل", "error", err)
	}
}
