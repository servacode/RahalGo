package server

// ══════════════════════════════════════════════════════════════════════
// **جسورُ QA للشهادة الحيّة الكاملة لتطبيق المندوب** — Rep E2E, staging-only
// ══════════════════════════════════════════════════════════════════════
//
// **كلُّها خلف `qaStagingEnabled` ⇒ ٤٠٤ في الإنتاج** (المسارُ غيرُ مسجَّلٍ أصلاً،
// ودفاعٌ ثانٍ في المعالِج/الوسيط). **ولا منطقَ عملٍ بديلٌ ولا SQL خامّ**: كلُّ
// جسرٍ يمرّ بالمعالِج الإنتاجيِّ الحقيقيِّ عينِه — نفسُ التحقّقِ والمعاملةِ
// والتدقيقِ والإشعارِ وحجزِ المال والبثِّ اللحظيّ — **ولا يقدّم إلّا التوثيقَ
// والتنسيقَ (auth/orchestration).**
//
// # جسران بطبيعتين
//
//  1. **جلسةُ مندوب** (`/qa/rep-session`): تُصدر توكنَ مندوبٍ (دور `sales`) لهويّةٍ
//     ثابتةٍ عبر مسار الهويّة الحقيقيّ — كجسرِ جلسةِ السائق تماماً.
//  2. **أفعالُ الأدمن** (المجموعةُ خلف `qaAdminActor`): تُحقن هويّةُ أدمن QA
//     الثابتةِ فاعلاً (`ctxUserID`) ثمّ يُشغَّل المعالِجُ الإنتاجيُّ نفسُه. **ولا
//     توكنَ أدمنٍ يُصدَر أبداً** — الحقنُ في السياق لهذا الطلب وحدَه، فلا يخرج
//     امتيازٌ من باب الاختبار.
//
// **وكلُّ أثرٍ عكوسٌ ويُنظَّف في خاتمة الشهادة** (إعداداتُ المال تُستعاد،
// وكياناتُ QA تُجهَّل، ولا رصيدَ ولا طلبَ معلَّقٌ يبقى).

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

const (
	// qaStagingRepPhoneA **مندوبُ QA الأوّل** — للجلسة والاكتساب والإدارة.
	// **بعيدٌ عن أرقام البذور** (كبقيّة هويّات QA)، وحارسُ الدور يمنع أيَّ امتياز.
	qaStagingRepPhoneA = "+963900555010"
	// qaStagingRepPhoneB **مندوبُ QA الثاني** — لشاهد نقلِ المتجر (أ ⇒ ب).
	qaStagingRepPhoneB = "+963900555011"
)

// qaRepIdentityForSlot **هويّةُ مندوب QA بحسب الخانة** — «أ» افتراضاً، «ب» للنقل.
func qaRepIdentityForSlot(slot string) (phone, name string) {
	if slot == "b" {
		return qaStagingRepPhoneB, "مندوب الاختبار QA-ب"
	}
	return qaStagingRepPhoneA, "مندوب الاختبار QA-أ"
}

// handleQARepSession **يُصدر جلسةً لمندوب QA الثابت** عبر مسار الهويّة الحقيقيّ.
// POST /qa/rep-session   body: {"slot":"a"|"b"}
//
// **يُنشئ الهويّةَ إن غابت (EnsureUserWithRole ⇒ منحُ دور `sales` ⇒ رمزُ دعوة)،
// ويضبط كلمتَها ليدخل التطبيقُ بالهاتف+الكلمة، ثمّ يُصدر التوكن.** ويردّ رمزَ
// الدعوة والحالةَ ووجوبَ تبديل الكلمة — ليقودها الاختبارُ الحيّ.
func (s *Server) handleQARepSession(w http.ResponseWriter, r *http.Request) {
	// **دفاعٌ في العمق**: ولو سُجّل المسارُ خطأً في غير التجهيز، يسقط مغلقاً.
	if !s.qaStagingEnabled() {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	slot := "a"
	if req, derr := decode[struct {
		Slot string `json:"slot"`
	}](r); derr == nil && req.Slot == "b" {
		slot = "b"
	}
	phone, name := qaRepIdentityForSlot(slot)
	ctx := r.Context()

	uid, err := s.qaFixedUser(ctx, phone, "sales", name, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **حارسُ الدور**: لا جلسةَ مندوبٍ لهويّةٍ صادفت دوراً مُمتازاً — فلا يُصنَع
	// توكنُ أدمنٍ من باب الاختبار ولو اصطدم رقمُ QA بحسابٍ خطير.
	if s.qaCarriesPrivileged(ctx, uid) {
		s.logger.Warn("QA rep session REFUSED — account carries a privileged role", "user", uid)
		s.respondErr(w, httpx.NewError(http.StatusForbidden, "qa_not_rep_only", "errors.forbidden"))
		return
	}
	// **وتُضبَط كلمتُه ليدخل تطبيقُ المندوب بالهاتف+الكلمة** — نفسُ كلمةِ QA.
	if err := s.identity.QASetPassword(ctx, uid, qaCustomerPassword); err != nil {
		s.respondErr(w, err)
		return
	}
	sid := s.identity.ActiveSessionID(ctx, uid)
	if sid == "" {
		sid = uuid.NewString()
	}
	res, err := s.identity.IssueForUserID(ctx, uid, sid, r.UserAgent(), clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **قراءةُ هويّةٍ محضةٌ** — رمزُ الدعوة والحالةُ ووجوبُ التبديل، ليقودها
	// الاختبار (نقلُ المتجر يلزمه رمزُ دعوةِ المندوب ب). لا كتابةَ عمل.
	var invite *string
	var status string
	var mustChange bool
	_ = s.pg.QueryRow(ctx,
		`SELECT invite_code, status, must_change_password FROM users WHERE id = $1::uuid`,
		uid).Scan(&invite, &status, &mustChange)

	s.logger.Warn("QA rep session issued (staging-only)", "user", uid, "slot", slot)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"session":              res,
		"user_id":              uid,
		"phone":                phone,
		"invite_code":          invite,
		"status":               status,
		"must_change_password": mustChange,
		"slot":                 slot,
	})
}

// qaRepMoneyKeys **مفاتيحُ مالِ المندوب المسموحُ ضبطُها مؤقّتاً في الشهادة** —
// **قائمةٌ صارمةٌ**: العمولةُ والهامشُ والمصدرُ والتفعيلُ والهدفُ والمكافأةُ وحدُّ
// السحب. مفتاحٌ واحدٌ نصّيٌّ (`sales.commission_source`، اختيار)، والبقيّةُ عدديّة.
// **لا مفتاحَ خارجَها** — فلا يتّسع البابُ إلى إعدادٍ آخر.
var qaRepMoneyKeys = map[string]bool{
	"sales.commission_percent":     true, // نسبةُ المندوب (عدد)
	"merchants.commission_percent": true, // نسبةُ المنصّة من المتجر (عدد)
	"pricing.margin_fixed":         true, // الهامشُ الثابت (مال)
	"sales.commission_source":      true, // مصدرُ العمولة (اختيار، نصّ)
	"sales.activation_orders":      true, // طلباتُ تفعيلِ المتجر (عدد)
	"sales.monthly_target":         true, // هدفُ المندوب الشهريّ (عدد)
	"sales.target_reward":          true, // مكافأةُ بلوغِ الهدف (مال)
	"payouts.min_amount":           true, // أدنى مبلغِ سحب (عدد)
}

// qaRepMoneySet **يضبط مفتاحَ مالِ مندوبٍ مسموحاً مؤقّتاً عبر مسار الإعدادات
// الحقيقيّ** (`settings.Set` بتحقّقه) **ويُرجع السابقَ للاستعادة** — نداءٌ ثانٍ
// بالقيمة السابقة يُعيد الحال. **لا مفتاحَ خارجَ `qaRepMoneyKeys`، ولا كتابةَ
// خام.** المفتاحُ النصّيُّ الوحيدُ `sales.commission_source` يأخذ `value_str`،
// والبقيّةُ `value_int`. — kind=rep_money_set (على التجهيز وحدَه).
func (s *Server) qaRepMoneySet(w http.ResponseWriter, r *http.Request, key string, valueInt int64, valueStr string) {
	if !qaRepMoneyKeys[key] {
		s.respondErr(w, httpx.NewError(http.StatusForbidden, "qa_money_key_not_allowed", "errors.forbidden"))
		return
	}
	ctx := r.Context()
	if key == "sales.commission_source" {
		prev := s.settings.GetString(ctx, key)
		if err := s.settings.Set(ctx, key, valueStr, nil); err != nil {
			s.respondErr(w, err)
			return
		}
		s.logger.Warn("QA rep money set (staging-only)", "key", key, "previous", prev, "set", valueStr)
		httpx.JSON(w, http.StatusOK, map[string]any{"key": key, "previous": prev, "set": valueStr})
		return
	}
	prev := s.settings.GetInt(ctx, key)
	if err := s.settings.Set(ctx, key, int(valueInt), nil); err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA rep money set (staging-only)", "key", key, "previous", prev, "set", valueInt)
	httpx.JSON(w, http.StatusOK, map[string]any{"key": key, "previous": prev, "set": valueInt})
}

// qaRepWalletFund **يشحن محفظةَ مندوب QA (خانة أ) عبر مسار الدفتر الحقيقيّ**
// (`wallet.ApplyTxID` نوع `topup`) **ليُشهَد تدفّقُ السحب حيّاً** — لا حقنَ خامّ.
//
// **ولماذا شحنٌ لا عمولة**: رصيدُ المندوب في الإنتاج عمولةٌ من طلبٍ سُلِّم، وذاك
// يلزمه طورُ السائق (تبعيّةٌ عبر التطبيقات). **فالشحنُ هنا يُمكّن شهادةَ آلةِ
// السحب** (طلبٌ ⇒ حجزٌ ⇒ قرارٌ ⇒ دفترٌ ⇒ إشعار) **بمصدرِ رصيدٍ صريحٍ مؤقّت**،
// وسحبٌ مدفوعٌ بكامل المبلغ يُعيد الرصيدَ إلى صفرٍ فلا يبقى أثر. — kind=rep_wallet_fund.
func (s *Server) qaRepWalletFund(w http.ResponseWriter, r *http.Request, amount int64) {
	if amount <= 0 || amount > 100_000_000 { // حتميٌّ موجبٌ بسقفٍ يحرس من خطأٍ عرضيّ
		s.respondErr(w, errValidation)
		return
	}
	ctx := r.Context()
	uid, err := s.qaFixedUser(ctx, qaStagingRepPhoneA, "sales", "مندوب الاختبار QA-أ", clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	actor := uid
	bal, txID, err := s.wallet.ApplyTxID(ctx, s.pg, uid, amount, "topup", "qa-rep-fund",
		"QA rep wallet funding (staging)", &actor)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.touchUser(uid, "wallet") // بثٌّ لحظيٌّ كمسار الأدمن الحقيقيّ
	s.logger.Warn("QA rep wallet funded (staging-only)", "user", uid, "amount", amount, "balance", bal, "tx", txID)
	httpx.JSON(w, http.StatusOK, map[string]any{"funded": amount, "balance": bal, "tx_id": txID, "user_id": uid})
}

// qaReleaseKeys **مفاتيحُ قنواتِ الإصدار المسموحُ ضبطُها في الشهادة** — اسمُ
// النسخة (نصّ) وحدُّها الأدنى (عدد)، **للمندوب والمتجر**. **قائمةٌ صارمةٌ**:
// لا مفتاحَ إصدارٍ لتطبيقٍ خارجَها، ولا مفتاحَ سواه.
var qaReleaseKeys = map[string]bool{
	"release.rep.version":      true, // اسمُ نسخةِ المندوب المعروض في /public/releases (نصّ)
	"app.min_version.rep":      true, // الحدُّ الأدنى لنسخةِ المندوب (عدد) — لشاهد التحديث الإلزاميّ 426
	"release.merchant.version": true, // اسمُ نسخةِ المتجر المعروض في /public/releases (نصّ)
	"app.min_version.merchant": true, // الحدُّ الأدنى لنسخةِ المتجر (عدد) — لشاهد التحديث الإلزاميّ 426
}

// qaReleaseSet **يضبط مفتاحَ قناةِ إصدارٍ مؤقّتاً عبر مسار الإعدادات الحقيقيّ**
// (`settings.Set`) **ويُرجع السابقَ للاستعادة**. مفاتيحُ `release.*.version`
// نصّيّةٌ (`value_str`)، و`app.min_version.*` عدديّةٌ (`value_int`). **لا مفتاحَ
// خارجَ `qaReleaseKeys`.** — kind=rep_release_set / merchant_release_set (على
// التجهيز وحدَه).
func (s *Server) qaReleaseSet(w http.ResponseWriter, r *http.Request, key string, valueInt int64, valueStr string) {
	if !qaReleaseKeys[key] {
		s.respondErr(w, httpx.NewError(http.StatusForbidden, "qa_release_key_not_allowed", "errors.forbidden"))
		return
	}
	ctx := r.Context()
	if key == "release.rep.version" || key == "release.merchant.version" {
		prev := s.settings.GetString(ctx, key)
		if err := s.settings.Set(ctx, key, valueStr, nil); err != nil {
			s.respondErr(w, err)
			return
		}
		s.logger.Warn("QA release set (staging-only)", "key", key, "previous", prev, "set", valueStr)
		httpx.JSON(w, http.StatusOK, map[string]any{"key": key, "previous": prev, "set": valueStr})
		return
	}
	prev := s.settings.GetInt(ctx, key)
	if err := s.settings.Set(ctx, key, int(valueInt), nil); err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA release set (staging-only)", "key", key, "previous", prev, "set", valueInt)
	httpx.JSON(w, http.StatusOK, map[string]any{"key": key, "previous": prev, "set": valueInt})
}

// qaAdminActor **وسيطٌ يحقن أدمنَ QA الثابتَ فاعلاً ثمّ يشغّل المعالِجَ الإنتاجيّ.**
//
// **على التجهيز وحدَه** (يسقط ٤٠٤ في غيره). **يخطّي حرّاسَ الأدمن (التوثيق/القدرة/
// الخطوة) وحدَها** — فالمسارُ الطافرُ هو مسارُ الإنتاجِ إلّا التخويلَ. **ولا توكنَ
// أدمنٍ يُصدَر**: الحقنُ في سياق هذا الطلب فقط، فيصير `actor_user_id` في التدقيق
// صادقاً (أدمن QA الثابت) دون أن يملك أحدٌ توكنَه.
func (s *Server) qaAdminActor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.qaStagingEnabled() {
			s.respondErr(w, httpx.ErrNotFound)
			return
		}
		adminID, err := s.qaFixedUser(r.Context(), qaStagingAdminPhone, "admin",
			"أدمن الاختبار QA", clientIP(r))
		if err != nil {
			s.respondErr(w, err)
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserID, adminID)
		ctx = context.WithValue(ctx, ctxRoles, []string{"admin"})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
