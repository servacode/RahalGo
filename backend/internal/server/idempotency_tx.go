package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ══════════════════════════════════════════════════════════════════════
// **منعُ التكرار: عملٌ وعلامةُ تثبيتٍ في معاملةٍ واحدة**
// ══════════════════════════════════════════════════════════════════════
//
// (`XG-33` · `R8` · `C-06` — دورةُ إصلاحٍ ٩.)
//
// # ما كان
//
// **حجزٌ ثمّ عملٌ ثمّ ختمٌ في ثلاثِ عمليّات.** **فحين تبقى المطالبةُ
// غيرَ منتهيةٍ لا يُعرَف من الصفّ وحدَه** أمات المنفّذُ قبل العمل أم
// بعده — **والفرقُ مالٌ**: طلبٌ ثانٍ أو دفعةٌ ثانية.
//
// # وما صار
//
//	IDEMPOTENCY DURABLE COMMIT STATE MUST BE ATOMIC WITH BUSINESS COMMIT
//
// **طَورٌ أوّل**: تُحاز المطالبةُ على المَسبَح **فتصير مرئيّةً لغيرها
// فوراً** — **ولو أُخّرت إلى داخل المعاملة لما رآها المتزامنُ فوقع
// تنفيذان.**
//
// **وطَورٌ ثانٍ**: معاملةٌ واحدةٌ تقفل الصفَّ وتتحقّق من الملكيّة ثمّ
// تكتب العملَ والعلامةَ والنتيجةَ **وتُثبَّت مرّةً واحدة.**
//
// # ولماذا القفلُ لا المهلةُ وحدَها
//
// **ساعةٌ تسبق ساعة، وعمليّةٌ بطيئةٌ ليست ميّتة.** **فمن أراد أن يسرق
// مطالبةً يعمل صاحبُها اصطدم بقفل صفِّها فانتظر** — ثمّ وجدها **مثبَّتةً
// فيُعيد نتيجتَها، أو مرتدّةً فيأخذها.**
//
// # والسياجُ يحمي من العكس
//
// **صاحبٌ استُرِدّت منه ثمّ استيقظ** — **يجد رمزَه لا يطابق فلا يكتب
// حرفاً.**

// idemClaim مطالبةٌ مملوكةٌ في هذا الطلب.
type idemClaim struct {
	uid, endpoint, key string
	token              uuid.UUID
}

type idemClaimKeyType struct{}

var idemClaimKey idemClaimKeyType

// claimFrom المطالبةُ المرافقةُ للطلب — إن وُجدت.
func claimFrom(ctx context.Context) *idemClaim {
	c, _ := ctx.Value(idemClaimKey).(*idemClaim)
	return c
}

// errClaimLost **فُقدت الملكيّةُ** — استردّها غيرُنا فلا كتابةَ لنا.
var errClaimLost = errors.New("idempotency: fenced out")

// errAlreadyCommitted **ثبت العملُ من قبل** — تُعاد نتيجتُه ولا يُعاد.
var errAlreadyCommitted = errors.New("idempotency: already committed")

// IdempotentBody ما يُرجعه العملُ ليُحفَظ ويُعاد.
type IdempotentBody struct {
	Status  int
	Payload any
	// AfterCommit **ما لا يقع إلّا بعد التثبيت** — إشعارٌ أو بثّ.
	//
	// **ولا يقع داخلَ المعاملة**: **إشعارٌ خرج ثمّ ارتدّت المعاملةُ
	// كذبٌ لا يُسحَب.** **وهو أفضلُ جهدٍ بعد التثبيت** — فشلُه لا
	// يُسقط عملاً ثبت.
	AfterCommit func()
}

// WithIdempotentTx يُجري عملاً في معاملةٍ واحدةٍ مسيَّجةٍ ويكتب ردَّه.
//
// **وهو المدخلُ الوحيدُ للمسارات المحميّة** — **ومن كتب عملاً خارجَه
// أعاد فجوةَ `XG-33`.**
//
// # ترتيبٌ مُلزَم
//
//  1. `SELECT … FOR UPDATE` **قبل أيّ كتابةِ عمل.**
//  2. **ثبت من قبلُ؟** ⇒ تُعاد نتيجتُه ولا يُنفَّذ شيء.
//  3. **رمزُ الملكيّة لا يطابق؟** ⇒ **صفرُ كتابات.**
//  4. العملُ · ثمّ **ترميزُ الردّ** · ثمّ العلامةُ والنتيجة.
//  5. **تثبيتٌ واحد.**
//
// **والترميزُ قبل التثبيت عمداً**: **ردٌّ تعذّر ترميزُه بعد تثبيت العمل
// يترك منعَ التكرار مجهولاً** — فيُرتَدّ العملُ كلُّه.
//
// **وبلا مفتاحٍ في الترويسة** تُجرى العمليّةُ في معاملةٍ أيضاً — **فحدُّ
// المعاملة ليس امتيازاً لمن أرسل مفتاحاً.**
func (s *Server) WithIdempotentTx(w http.ResponseWriter, r *http.Request,
	do func(ctx context.Context, q dbtx.Querier) (IdempotentBody, error)) {
	ctx := r.Context()
	claim := claimFrom(ctx)

	tx, err := s.pg.Begin(ctx)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if claim != nil {
		// ── ١ ── قفلُ الصفّ قبل أيّ كتابة ─────────────────────────
		var committed *string
		var owner *uuid.UUID
		var status int
		var body string
		err := tx.QueryRow(ctx, `
			SELECT committed_at::text, owner_token, status_code, COALESCE(response, '')
			  FROM idempotency_keys
			 WHERE user_id = $1 AND endpoint = $2 AND key = $3
			 FOR UPDATE`,
			claim.uid, claim.endpoint, claim.key).
			Scan(&committed, &owner, &status, &body)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			// **مُحيت تحتنا** — تقليمٌ أو إطلاقُ خطأ. **ولا نكتب على
			// مطالبةٍ لا نملكها.**
			s.respondErr(w, errIdemLost)
			return
		case err != nil:
			s.respondErr(w, err)
			return
		case committed != nil:
			// ── ٢ ── ثبت من قبلُ ⇒ تُعاد نتيجتُه ────────────────
			_ = tx.Rollback(ctx)
			s.writeReplay(w, status, body)
			return
		case owner == nil || *owner != claim.token:
			// ── ٣ ── السياج: صفرُ كتابات ─────────────────────────
			_ = tx.Rollback(ctx)
			s.logger.Warn("منعُ التكرار: مالكٌ فقد مطالبتَه",
				"endpoint", claim.endpoint)
			s.respondErr(w, errIdemLost)
			return
		}
	}

	// ── ٤ ── العملُ ثمّ ترميزُ الردّ ثمّ العلامة ─────────────────
	out, err := do(ctx, tx)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **والمحفوظُ هو المكتوبُ حرفاً بحرف** — **و`httpx.JSON` تستعمل
	// `Encode` فتُلحق سطراً جديداً**، **فمحرفٌ واحدٌ يجعل الردَّ المُعادَ
	// غيرَ الأصل** — ومن قارنهما ظنّ أنّ شيئاً تبدّل.
	//
	// **والترميزُ قبل التثبيت عمداً**: **ردٌّ تعذّر ترميزُه بعد تثبيت
	// العمل يترك منعَ التكرار مجهولاً** — فيُرتَدّ العملُ كلُّه.
	encoded, err := json.Marshal(map[string]any{"data": out.Payload})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	encoded = append(encoded, byte(10)) // سطرٌ جديدٌ كما تكتبه `Encode`
	if claim != nil {
		tag, err := tx.Exec(ctx, `
			UPDATE idempotency_keys
			   SET committed_at = now(), done = true,
			       status_code = $4, response = $5
			 WHERE user_id = $1 AND endpoint = $2 AND key = $3
			   AND owner_token = $6 AND committed_at IS NULL`,
			claim.uid, claim.endpoint, claim.key,
			out.Status, string(encoded), claim.token)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		if tag.RowsAffected() != 1 {
			// **تبدّلت الملكيّةُ بين القفل والكتابة** — لا يقع عمليّاً
			// **لأنّ القفلَ قائم**، **وحارسٌ لا يُختبَر لا يُوثَق به**
			// فيبقى مكتوباً.
			s.respondErr(w, errIdemLost)
			return
		}
	}

	// ── ٥ ── تثبيتٌ واحد ─────────────────────────────────────────
	if err := tx.Commit(ctx); err != nil {
		s.respondErr(w, err)
		return
	}
	if out.AfterCommit != nil {
		out.AfterCommit()
	}
	httpx.JSON(w, out.Status, out.Payload)
}

// errIdemLost **مطالبةٌ لم تعُد لنا** — يُعاد النداءُ فيقرأ الحقيقة.
var errIdemLost = httpx.NewError(http.StatusConflict,
	"idempotency_reclaimed", "errors.in_progress")

// writeReplay يُعيد النتيجةَ المحفوظةَ كما كانت.
func (s *Server) writeReplay(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Idempotent-Replay", "true")
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}
