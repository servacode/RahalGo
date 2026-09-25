package server

// ══════════════════════════════════════════════════════════════════════
// **عتادُ الإغلاق الأخير — على التجهيز وحدَه** (`qa_reconcile.go`)
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٩-٢٣: «الدفعةُ الأخيرة قبل الجلسة المرافقة».)
//
// شيئان ضيّقان لشهود الزبون، كلاهما يسقط مغلقاً في الإنتاج:
//
//	merchant_second        ·  متجرٌ ثانٍ صغيرٌ عكوسٌ لشهود سقفِ المصادر (CUST-11-036)
//	merchant_second_clear  ·  حذفُه (FK-safe)
//	GET /qa/reconcile      ·  مطابقةُ بيانات التجهيز — أعدادٌ وثوابتُ فقط (CUST-22-013)
//
// **ولا يُصدِر شيءٌ منها سرّاً ولا بياناتِ زبونٍ خامّةً ولا توكناً ولا SQL حرّاً**،
// و`reconcile` **قراءةٌ محضةٌ بلا أيّ تعديل**، و`merchant_second*` لا يمسّان مالاً.

import (
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
)

// qaSecondMerchantName اسمُ المتجر الثاني القابلِ للحذف — علامةٌ ثابتةٌ للتنظيف.
const qaSecondMerchantName = "QA متجر ثانٍ (اختبار القبول)"

// qaMerchantSecond **يبني متجراً ثانياً صغيراً على التجهيز** — مصدرٌ ثانٍ متمايزٌ
// (`SourcesOf` يجمع بـ`merchant_id`) لشهود سقفِ المصادر (`orders.max_sources`).
//
// عكوسٌ (`merchant_second_clear`)، بلا أثرٍ ماليّ، ولا يبدأ قبولَ المتجر: صنفٌ
// واحدٌ متاحٌ معتمَدٌ يكفي أن يُعدَّ مصدراً في تسعيرةٍ/طلب. يُعيد المعرّفَين.
func (s *Server) qaMerchantSecond(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// **متكرّرٌ آمن**: لو كان قائماً من شاهدٍ سابقٍ يُعاد لا يُضاعَف.
	var mid, itemID string
	err := s.pg.QueryRow(ctx, `
		SELECT m.id::text, mi.id::text
		FROM merchants m
		JOIN menu_items mi ON mi.merchant_id = m.id
		WHERE m.name = $1
		ORDER BY mi.id LIMIT 1`, qaSecondMerchantName).Scan(&mid, &itemID)
	if err == nil {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"merchant_id": mid, "item_id": itemID, "reused": true,
		})
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		s.respondErr(w, err)
		return
	}

	tx, err := s.pg.Begin(ctx)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := tx.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, commission_percent, status)
		VALUES ($1, (SELECT id FROM categories ORDER BY id LIMIT 1), 10, 'active')
		RETURNING id::text`, qaSecondMerchantName).Scan(&mid); err != nil {
		s.respondErr(w, err)
		return
	}
	var secID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO menu_sections (merchant_id, name)
		VALUES ($1::uuid, 'الرئيسية') RETURNING id::text`, mid).Scan(&secID); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO menu_items
		  (merchant_id, section_id, platform_section_id, name, merchant_price, price, available, approved)
		VALUES ($1::uuid, $2::uuid,
		        (SELECT id FROM platform_sections ORDER BY sort_order LIMIT 1),
		        'QA صنف المصدر الثاني', 10000, 10000, true, true)
		RETURNING id::text`, mid, secID).Scan(&itemID); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA second merchant seeded (staging-only)", "merchant", mid, "item", itemID)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"merchant_id": mid, "item_id": itemID, "reused": false,
	})
}

// qaMerchantSecondClear **يحذف المتجرَ الثاني وأصنافَه وأقسامَه** — FK-safe.
func (s *Server) qaMerchantSecondClear(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// **الأصنافُ ثمّ الأقسامُ ثمّ المتجر** — بالترتيب الذي تسمح به المفاتيح.
	if _, err := tx.Exec(ctx, `
		DELETE FROM menu_items WHERE merchant_id IN
		  (SELECT id FROM merchants WHERE name = $1)`, qaSecondMerchantName); err != nil {
		s.respondErr(w, err)
		return
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM menu_sections WHERE merchant_id IN
		  (SELECT id FROM merchants WHERE name = $1)`, qaSecondMerchantName); err != nil {
		s.respondErr(w, err)
		return
	}
	tag, err := tx.Exec(ctx, `DELETE FROM merchants WHERE name = $1`, qaSecondMerchantName)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA second merchant cleared (staging-only)", "deleted", tag.RowsAffected())
	httpx.JSON(w, http.StatusOK, map[string]any{"deleted_merchants": tag.RowsAffected()})
}

// qaOptionAvailable **يقلب إتاحةَ خيارِ إضافةٍ** (`modifier_options.available`) —
// لشهود «خيارٌ غيرُ متاحٍ مُعطَّلٌ في الورقة» (`CUST-10-014`). عكوسٌ، يُرجع السابق.
//
// **بلا optionID**: أوّلُ خيارٍ لأوّلِ مجموعةِ الصنف — فيكفي معرّفُ الصنف.
func (s *Server) qaOptionAvailable(w http.ResponseWriter, r *http.Request, optionID, itemID string, avail bool) {
	ctx := r.Context()
	if optionID == "" {
		if itemID == "" {
			s.respondErr(w, errValidation)
			return
		}
		if err := s.pg.QueryRow(ctx, `
			SELECT o.id::text FROM modifier_options o
			JOIN modifier_groups g ON g.id = o.group_id
			WHERE g.item_id = $1::uuid ORDER BY o.sort_order LIMIT 1`, itemID).Scan(&optionID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				s.respondErr(w, httpx.NewError(http.StatusConflict, "qa_no_option", "errors.conflict"))
				return
			}
			s.respondErr(w, err)
			return
		}
	}
	var prev bool
	var name string
	if err := s.pg.QueryRow(ctx,
		`SELECT available, name FROM modifier_options WHERE id = $1::uuid`, optionID).Scan(&prev, &name); err != nil {
		s.respondErr(w, err)
		return
	}
	if _, err := s.pg.Exec(ctx,
		`UPDATE modifier_options SET available = $2 WHERE id = $1::uuid`, optionID, avail); err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA option availability set (staging-only)", "option", optionID, "previous", prev, "set", avail)
	httpx.JSON(w, http.StatusOK, map[string]any{"option_id": optionID, "name": name, "previous": prev, "set": avail})
}

// qaItemImage **يبدّل صورةَ صنفٍ** (`menu_items.image_media_id`) — لشهود «صورةٌ
// ناقصةٌ/معطوبةٌ لا تكسر الشاشة» (`CUST-09-011`). عكوسٌ: يُرجع السابقَ ويُعاد به.
//
// **mediaID فارغٌ ⇒ إزالةُ الصورة** (NULL = ناقصة ⇒ بديلٌ). وغيرُ الفارغِ يُعيدها.
func (s *Server) qaItemImage(w http.ResponseWriter, r *http.Request, itemID, mediaID string) {
	ctx := r.Context()
	if itemID == "" {
		s.respondErr(w, errValidation)
		return
	}
	var prev *string
	if err := s.pg.QueryRow(ctx,
		`SELECT image_media_id::text FROM menu_items WHERE id = $1::uuid`, itemID).Scan(&prev); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.respondErr(w, httpx.ErrNotFound)
			return
		}
		s.respondErr(w, err)
		return
	}
	if mediaID == "" {
		if _, err := s.pg.Exec(ctx, `UPDATE menu_items SET image_media_id = NULL WHERE id = $1::uuid`, itemID); err != nil {
			s.respondErr(w, err)
			return
		}
	} else {
		if _, err := s.pg.Exec(ctx, `UPDATE menu_items SET image_media_id = $2::uuid WHERE id = $1::uuid`, itemID, mediaID); err != nil {
			s.respondErr(w, err)
			return
		}
	}
	prevOut := ""
	if prev != nil {
		prevOut = *prev
	}
	s.logger.Warn("QA item image set (staging-only)", "item", itemID, "previous", prevOut, "set", mediaID)
	httpx.JSON(w, http.StatusOK, map[string]any{"item_id": itemID, "previous": prevOut, "set": mediaID})
}

// qaCustomerSuspend **يوقف/يُعيد زبونَ QA وحدَه** — لشهود «موقوفٌ وله طلبٌ حيّ»
// (`CUST-06-031`). عكوسٌ، **مقصورٌ على رقم QA** (لا رقمَ آخرَ يُمَسّ)، **بلا
// إبطالِ جلسة** — مطابقةً لعقد المالك (٢٠٢٦-٠٩-٠٧): الإيقافُ يمنع نشاطاً جديداً
// ولا يترك طلباً حيّاً معلَّقاً، **فالجلسةُ تبقى (توثيقٌ لا تخويل).**
//
// **ولا يُصدِر جلسةَ أدمنٍ ولا يقبل رقماً من الطلب**: الرقمُ ثابتٌ في الشيفرة،
// والحالةُ محصورةٌ في `suspended`/`active`.
func (s *Server) qaCustomerSuspend(w http.ResponseWriter, r *http.Request, status string) {
	if status != "suspended" && status != "active" {
		s.respondErr(w, errValidation)
		return
	}
	ctx := r.Context()
	var uid, prev string
	if err := s.pg.QueryRow(ctx,
		`SELECT id::text, status FROM users WHERE phone = $1`, qaStagingPhone).Scan(&uid, &prev); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.respondErr(w, httpx.ErrNotFound)
			return
		}
		s.respondErr(w, err)
		return
	}
	// **رقمُ QA وحدَه** — والحالةُ من `switch` لا من الطلب. **ولا إبطالَ للجلسات.**
	tag, err := s.pg.Exec(ctx,
		`UPDATE users SET status = $2 WHERE phone = $1 AND status IN ('active','suspended')`,
		qaStagingPhone, status)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **ويُبطَل كاشُ الحالة فوراً** — كالمسار الإداريّ (`invalidateStatusCache`)،
	// **وإلّا رأى الوسيطُ الحالةَ القديمةَ ٣٠ث فلا يُنفَّذ التعليقُ حتميّاً.**
	s.identity.InvalidateStatusCache(ctx, uid)
	s.logger.Warn("QA customer status set (staging-only)", "phone", "QA", "previous", prev, "set", status, "rows", tag.RowsAffected())
	httpx.JSON(w, http.StatusOK, map[string]any{"previous": prev, "set": status, "rows": tag.RowsAffected()})
}

// qaCustomerPassword كلمةُ مرورِ زبون QA الثابتة — لدخولِ الواجهة على التجهيز
// (بلا OTP، بلا حقنِ توكن). **رقمُ QA اختباريّ على التجهيز فقط.**
const qaCustomerPassword = "RahalQA@2026"

// qaCustomerSetPassword **يضبط كلمةَ مرورِ زبون QA** — ليُسجَّل الدخولُ بالواجهة
// على الجهاز (رقم QA + هذه الكلمة). **رقمُ QA وحدَه**، staging-only، عكوسٌ عمليّاً
// (يُعاد ضبطُها أو يُبقى — حسابُ اختبار). **لا يفتح بابَ أدمن ولا يمسّ رقماً آخر.**
func (s *Server) qaCustomerSetPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var uid string
	if err := s.pg.QueryRow(ctx,
		`SELECT id::text FROM users WHERE phone = $1`, qaStagingPhone).Scan(&uid); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.respondErr(w, httpx.ErrNotFound)
			return
		}
		s.respondErr(w, err)
		return
	}
	if err := s.identity.QASetPassword(ctx, uid, qaCustomerPassword); err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA customer password set (staging-only)", "phone", "QA")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "phone": qaStagingPhone})
}

// ── الجهازُ التجريبيّ لحذف الحساب (CUST-06-026) ─────────────────────────
// **رقمٌ منفصلٌ يُستهلك** — لا يُمسّ به زبونُ QA الأساسيّ. staging-only، عكوسٌ
// (الحذفُ يحرّر الرقمَ فيُعاد إنشاؤُه).
const (
	qaDisposablePhone = "+963900555999"
	qaDisposableName  = "زبون تجريبي للحذف (QA)"
)

// qaDisposableCreate يُنشئ (أو يُرجع) زبونَ الحذف التجريبيّ بكلمةٍ معلومة.
func (s *Server) qaDisposableCreate(w http.ResponseWriter, r *http.Request) {
	id, err := s.identity.QACreateOrGetCustomer(r.Context(), qaDisposablePhone, qaDisposableName, qaCustomerPassword)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA disposable customer ready (staging-only)", "id", id)
	httpx.JSON(w, http.StatusOK, map[string]any{"id": id, "phone": qaDisposablePhone, "password": qaCustomerPassword})
}

// qaDisposableDeleteCode يُصدر رمزَ حذفٍ حقيقيّاً للجهاز التجريبيّ ويُعيده —
// يُدعى **بعد** أن يطلب التطبيقُ الرمزَ (الأحدثُ يُستهلك في `ConsumeOTP`).
func (s *Server) qaDisposableDeleteCode(w http.ResponseWriter, r *http.Request) {
	code, err := s.identity.QAIssueDeleteCode(r.Context(), qaDisposablePhone)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA disposable delete code issued (staging-only)")
	httpx.JSON(w, http.StatusOK, map[string]any{"code": code, "phone": qaDisposablePhone})
}

// ── رموزُ OTP على التجهيز (04/05/06) ────────────────────────────────────
// **المزوّدُ dev يطبع الرمزَ في السجلّ لا يرسله واتساب** — فلا سبيلَ لشهود
// مسارات OTP على التطبيق إلّا إصدارُ الرمز وإعادتُه. **مقصورٌ على أرقام QA**
// (نطاقٌ محجوز)، staging-only، **ولا يتجاوز التحقّق**: `ConsumeOTP` يبقى كما هو
// (مهلة/محاولات/الأحدث)، فتبقى حالاتُ القبول حقيقيّة.
const (
	qaSignupPhone = "+963900555998" // تسجيلٌ جديد (04-*)
	qaNewPhone    = "+963900555997" // هدفُ تغيير الرقم (06-024)
)

// qaSignupWitnessPhones **أرقامُ تسجيلٍ إضافيّةٌ ثابتةٌ** لشاهدِ «ضاع الرد»
// (Batch 4): كلُّ محاولةٍ تستهلك رقماً (يصير له حساب)، فيلزم عدّةٌ ثابتةٌ صغيرة.
var qaSignupWitnessPhones = []string{
	"+963900555990", "+963900555991", "+963900555992", "+963900555993", "+963900555994",
}

var qaOTPPhones = func() map[string]bool {
	m := map[string]bool{
		qaStagingPhone:    true, // QA1 — reset/whatsapp/change-from
		qaDisposablePhone: true, // زبونُ الحذف
		qaSignupPhone:     true, // تسجيلٌ جديد
		qaNewPhone:        true, // هدفُ تغيير الرقم
	}
	for _, p := range qaSignupWitnessPhones {
		m[p] = true // أرقامُ شاهدِ «ضاع الرد» — تسجيلٌ ثابتٌ متعدّد
	}
	return m
}()
var qaOTPPurposes = map[string]bool{
	"login":        true, // مسارُ OTP للدخول/التسجيل (/auth/otp/request+verify) — RequestOTP
	"signup":       true, // مسارُ تسجيلٍ منفصلٍ إن وُجد
	"reset":        true, // استعادةُ كلمة المرور
	"whatsapp":     true, // توثيقُ واتساب
	"phone_change": true, // تغييرُ الرقم
	"delete":       true, // حذفُ الحساب
}

// qaMaybeSignupLatency **مِعطارُ تأخيرٍ ضيّقٌ (CUST-04-010)** — إن سُلّح تأخيرٌ
// على مسار طلبِ الرمز `/auth/otp/request` **ورقمُ الطالبِ رقمُ QA للتسجيل**،
// نام المقدارَ المسلَّح ثمّ مضى. staging-only، مقصورٌ على رقم QA، يُستهلك مرّةً —
// **فيُشهَد «تسجيلٌ بطيء» بلا مسٍّ لمستخدمٍ آخر ولا للإنتاج.**
func (s *Server) qaMaybeSignupLatency(rawPhone string) {
	if !s.qaStagingEnabled() {
		return
	}
	phone, ok := identity.NormalizePhone(rawPhone)
	if !ok || phone != qaSignupPhone {
		return
	}
	if f, ok := qaFaults.take("/api/v1/auth/signup/request"); ok && f.mode == qaFaultLatency {
		s.logger.Warn("QA signup latency injected (staging-only)", "ms", f.ms)
		time.Sleep(time.Duration(f.ms) * time.Millisecond)
	}
}

// qaMaybeLoginLatency **مِعطارُ تأخيرٍ ضيّقٌ لدخول QA (CUST-06-006)** — نظيرُ
// معطارِ التسجيل: `/auth/login` غيرُ مصادَقٍ فلا يبلغه حاقنُ العطب العاديّ (المقصورُ
// على uid)؛ فهذا يُطبّق التأخيرَ المسلَّح **لرقم QA وحدَه** على التجهيز، افتراضُه
// مطفأ (لا عطبَ مسلَّح ⇒ لا أثر). لا يمسّ رقماً آخرَ ولا الإنتاج.
func (s *Server) qaMaybeLoginLatency(rawPhone string) {
	if !s.qaStagingEnabled() {
		return
	}
	phone, ok := identity.NormalizePhone(rawPhone)
	if !ok || phone != qaStagingPhone {
		return
	}
	if f, ok := qaFaults.take("/api/v1/auth/login"); ok && f.mode == qaFaultLatency {
		s.logger.Warn("QA login latency injected (staging-only)", "ms", f.ms)
		time.Sleep(time.Duration(f.ms) * time.Millisecond)
	}
}

// qaOTPCode يُصدر رمزَ OTP لرقمِ QA وغرضٍ محدَّدين ويُعيده.
func (s *Server) qaOTPCode(w http.ResponseWriter, r *http.Request, rawPhone, purpose string) {
	phone, ok := identity.NormalizePhone(rawPhone)
	if !ok || !qaOTPPhones[phone] {
		s.respondErr(w, httpx.NewError(http.StatusForbidden, "qa_phone_not_allowed", "errors.forbidden"))
		return
	}
	if !qaOTPPurposes[purpose] {
		s.respondErr(w, errValidation)
		return
	}
	code, err := s.identity.QAIssueCode(r.Context(), phone, purpose)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA OTP code issued (staging-only)", "purpose", purpose)
	httpx.JSON(w, http.StatusOK, map[string]any{"code": code, "phone": phone, "purpose": purpose})
}

// qaSignupBonusReverse **يعكس هديّةَ تسجيلِ حسابِ `qaSignupPhone`** (CUST-22-016،
// تنظيفُ الدخان الحيّ).
//
// **لماذا مِعطارٌ ودَورُه**: كلُّ تسجيلٍ جديدٍ ينال هديّةً عامّة
// (`customers.signup_bonus`) بقيدٍ **مزدوج**: +المبلغ للحساب و−المبلغ للخزينة
// (`GrantSignupBonus`). فحذفُ الحساب يرفضه العقدُ ما دامت محفظتُه غيرَ صفر
// (`ErrWalletNotEmpty`)، **ولا مِعطارَ قائمٌ يعكس الطرفين**: `qaWalletDrain`
// مقصورٌ على QA1 وقيدُه **مفردٌ** (لا مقابلَ خزينة) فيُخلّ التوازن.
//
// **فالعكسُ هنا مزدوجٌ عبر المسار المعتمد** `wallet.ApplyTx` — **لا تعديلَ قاعدةٍ
// خام**: −الرصيد من الحساب و+الرصيد على الخزينة في معاملةٍ واحدة، فيعود الحسابُ
// **صفراً** والخزينةُ إلى ما كانت **قبل الهديّة** بالضبط. ثمّ يُحذف الحسابُ عبر
// المسار الحقيقيّ `/auth/account/delete`. **staging فقط** (الموزّعُ خلفَ
// `qaStagingEnabled` ⇒ ٤٠٤ في الإنتاج)، **ومقصورٌ على رقمِ تسجيل QA وحدَه**،
// **وعكوسٌ** (رصيدٌ صفرٌ ⇒ لا قيد).
func (s *Server) qaSignupBonusReverse(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	phone, ok := identity.NormalizePhone(qaSignupPhone)
	if !ok {
		s.respondErr(w, errValidation)
		return
	}
	var uid string
	err := s.pg.QueryRow(ctx, `SELECT id::text FROM users WHERE phone = $1`, phone).Scan(&uid)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.JSON(w, http.StatusOK, map[string]any{"reversed": 0, "note": "no_account"})
		return
	} else if err != nil {
		s.respondErr(w, err)
		return
	}
	bal, err := s.wallet.Balance(ctx, uid)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if bal == 0 {
		httpx.JSON(w, http.StatusOK, map[string]any{"reversed": 0, "account_balance": 0})
		return
	}
	var tid string
	if err := s.pg.QueryRow(ctx, `SELECT user_id::text FROM wallets WHERE is_treasury LIMIT 1`).Scan(&tid); err != nil {
		s.respondErr(w, err)
		return
	}
	// **قيدٌ مزدوجٌ في معاملةٍ واحدة** — إمّا كلاهما أو لا شيء.
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	actor := uid
	accBal, _, e1 := s.wallet.ApplyTxID(ctx, tx, uid, -bal, "adjustment", "qa-signup-bonus-reverse", "عكسُ هديّة التسجيل (تنظيف QA)", &actor)
	if e1 != nil {
		s.respondErr(w, e1)
		return
	}
	treBal, _, e2 := s.wallet.ApplyTxID(ctx, tx, tid, bal, "adjustment", "qa-signup-bonus-reverse", "عكسُ هديّة التسجيل — استعادةُ الخزينة (تنظيف QA)", &actor)
	if e2 != nil {
		s.respondErr(w, e2)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		s.respondErr(w, err)
		return
	}
	s.touchUser(uid, "wallet")
	s.logger.Warn("QA signup bonus reversed (staging-only)", "user", uid, "amount", bal, "account_balance", accBal, "treasury_balance", treBal)
	httpx.JSON(w, http.StatusOK, map[string]any{"reversed": bal, "account_balance": accBal, "treasury_balance": treBal})
}

// ── روابطُ التواصل (CUST-ENG-011) ───────────────────────────────────────
// **يضبط إعداداتِ التواصل لشهودها على الجهاز ثمّ يستعيدها.** staging-only، عكوس،
// **لا بابَ أدمن**: يكتب مفاتيحَ التواصل وحدَها ويحفظ سابقَها.
var qaContactKeys = []string{"platform.support_phone", "platform.whatsapp", "platform.facebook"}
var qaContactValues = map[string]string{
	"platform.support_phone": "0912345678",
	"platform.whatsapp":      "0912345678",
	"platform.facebook":      "https://facebook.com/rahalgo",
}
var qaContactSaved map[string]string

func (s *Server) qaContactSet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// **`updated_by` عمودُ UUID** — تمريرُ نصٍّ غيرِ UUID يرمي 22P02. البذّارُ
	// لا مستخدمَ له فيُترك NULL.
	saved := map[string]string{}
	for _, k := range qaContactKeys {
		saved[k] = s.settings.GetString(ctx, k)
		if err := s.settings.Set(ctx, k, qaContactValues[k], nil); err != nil {
			s.respondErr(w, err)
			return
		}
	}
	qaContactSaved = saved
	s.logger.Warn("QA contact links set (staging-only)")
	httpx.JSON(w, http.StatusOK, map[string]any{"set": qaContactValues})
}

func (s *Server) qaContactClear(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	for _, k := range qaContactKeys {
		prev := ""
		if qaContactSaved != nil {
			prev = qaContactSaved[k]
		}
		if err := s.settings.Set(ctx, k, prev, nil); err != nil {
			s.respondErr(w, err)
			return
		}
	}
	s.logger.Warn("QA contact links restored (staging-only)")
	httpx.JSON(w, http.StatusOK, map[string]any{"restored": true})
}

// handleQAStagingReconcile **مطابقةُ بيانات التجهيز بعد الاختبار** — على التجهيز
// وحدَه، **قراءةٌ محضة** (`CUST-22-013`).
//
// # ماذا يُرجع
//
//	money    ·  فحوصُ `fininv` التشغيليّةُ (نفسُ moneycheck): كم فُحص، كم سلِم،
//	            وأيُّها خُرق بعددِ صفوفه فقط — لا صفوفَ ولا بيانات.
//	residue  ·  أثرُ عتادِ QA: طلباتٌ جاريةٌ لزبونَي QA، رصيدُ محفظتيهما،
//	            أصنافُ الكثافة، عروضُ QA الفعّالة، والمتجرُ الثاني — كلُّها أعداد.
//
// **ولا سرَّ ولا هويّةَ زبونٍ ولا توكن**: أرقامٌ مجمَّعةٌ فقط. **ولا تعديلَ.**
func (s *Server) handleQAStagingReconcile(w http.ResponseWriter, r *http.Request) {
	if !s.qaStagingEnabled() {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	ctx := r.Context()

	// ── الثوابتُ الماليّة (moneycheck على التجهيز) ──────────────────────
	total, passed := 0, 0
	failed := []map[string]any{}
	for _, c := range fininv.Select() {
		if !c.Ops {
			continue // **ما لا يُشغَّل على قاعدةِ تشغيلٍ لا يُعَدّ** — كـmoneycheck.
		}
		total++
		vs, err := fininv.Run(ctx, s.pg, c.ID)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		if len(vs) == 0 {
			passed++
			continue
		}
		entry := map[string]any{
			"id": c.ID, "name": c.Name, "violating_rows": len(vs[0].Rows),
		}
		// **تصنيفُ الخرق بحالةِ الطلب فقط** — عددٌ لكلّ حالة، بلا معرّفٍ ولا
		// مبلغٍ ولا بيانةِ زبون. **فيُعرَف أهو طلبٌ ملغىً (أثرٌ حميد) أم جارٍ.**
		statusIdx := -1
		for i, col := range vs[0].Cols {
			if col == "status" {
				statusIdx = i
				break
			}
		}
		if statusIdx >= 0 {
			byStatus := map[string]int{}
			for _, row := range vs[0].Rows {
				if statusIdx < len(row) {
					if st, ok := row[statusIdx].(string); ok {
						byStatus[st]++
					}
				}
			}
			entry["by_status"] = byStatus
		}
		failed = append(failed, entry)
	}

	// ── أثرُ عتادِ QA ────────────────────────────────────────────────
	count := func(sql string, args ...any) int64 {
		var n int64
		_ = s.pg.QueryRow(ctx, sql, args...).Scan(&n)
		return n
	}
	qaOpenOrders := count(`
		SELECT count(*) FROM orders o JOIN users u ON u.id = o.customer_id
		WHERE u.phone IN ($1, $2) AND o.closed_at IS NULL`, qaStagingPhone, qaStagingPhone2)
	qaWalletBalance := count(`
		SELECT COALESCE(sum(w.balance), 0) FROM wallets w JOIN users u ON u.id = w.user_id
		WHERE u.phone IN ($1, $2)`, qaStagingPhone, qaStagingPhone2)
	qaDenseItems := count(`SELECT count(*) FROM menu_items WHERE name LIKE 'QA\_DENSE%' ESCAPE '\'`)
	qaActiveOffers := count(`SELECT count(*) FROM offers WHERE active AND title LIKE 'QA%'`)
	qaSecondMerchants := count(`SELECT count(*) FROM merchants WHERE name = $1`, qaSecondMerchantName)

	// ── مراقبةُ #1050 قراءةً فقط (CUST-14-020) — حالتُه وعددُ أحداثه بصمةٌ ──
	// **تتغيّر لو تقدّم**. لا تعديلَ، أرقامٌ فقط.
	var o1050Status string
	var o1050Exists bool
	var o1050Events int64
	if err := s.pg.QueryRow(ctx, `SELECT status FROM orders WHERE number = 1050`).Scan(&o1050Status); err == nil {
		o1050Exists = true
		o1050Events = count(`SELECT count(*) FROM order_events e JOIN orders o ON o.id = e.order_id WHERE o.number = 1050`)
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"money": map[string]any{
			"checks_total": total, "checks_passed": passed, "failed": failed,
		},
		"residue": map[string]any{
			"qa_open_orders":      qaOpenOrders,
			"qa_wallet_balance":   qaWalletBalance,
			"qa_dense_items":      qaDenseItems,
			"qa_active_offers":    qaActiveOffers,
			"qa_second_merchants": qaSecondMerchants,
		},
		"order_1050": map[string]any{
			"exists": o1050Exists, "status": o1050Status, "events": o1050Events,
		},
	})
}
