package server

// ══════════════════════════════════════════════════════════════════════
// **أدواتُ QA للشهادة الحيّة الكاملة لعقد الطلب المخصَّص** — Batch 2, staging-only
// ══════════════════════════════════════════════════════════════════════
//
// **كلُّها خلف `qaStagingEnabled` ⇒ ٤٠٤ في الإنتاج** (دفاعٌ في العمق في كلّ
// معالِج). **هويّاتٌ ثابتةٌ لا انتحالَ حرّ**: زبونُ QA القائم، وسائقُ QA ثابت،
// وأدمنُ QA ثابتٌ يُستعمل **فاعلَ تدقيقٍ فقط** (لا توكنَ يُصدَر له، وجلسةُ QA
// ترفض الأدمن). **وكلُّها تمرّ بالمسار الإنتاجيّ الحقيقيّ** (settings.Set
// بتحقّقه، AgreeCustom، AdminOverrideCustomQuote، Transition) — **فالبثُّ
// اللحظيّ والثوابتُ الماليّةُ كما في الإنتاج، ولا اختصارَ اختباريّ.**
//
// **وكلُّ أثرٍ عكوسٌ**: `custom-cleanup` يُلغي طلبات QA المفتوحة (فيُطلَق الحجز)،
// ويُفرِّغ محفظةَ QA، ويُعيد سياسةَ الأجرة إلى الافتراض.

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

const (
	// qaStagingDriverPhone **سائقُ QA الثابت** — لشهادة تطبيق السائق الحيّة.
	qaStagingDriverPhone = "+963900555003"
	// qaStagingAdminPhone **أدمنُ QA الثابت** — فاعلُ تدقيقٍ لمساعد التدخّل
	// وحدَه؛ **لا توكنَ يُصدَر له** (جلسةُ QA ترفض الأدمن)، ولا يُسجَّل دخولُه
	// (بلا كلمة). موجودٌ ليكون `actor_user_id` صادقاً في تدقيق التدخّل.
	qaStagingAdminPhone = "+963900555009"

	qaFeeSourceKey    = "delivery.custom_fee_source"
	qaFeeAmountKey    = "delivery.custom_fee"
	qaFeeMayChangeKey = "delivery.custom_driver_may_change_fee"
)

// qaFixedUser **يضمن هويّةَ QA ثابتةً بدورها** ويردّ معرّفَها — المسارُ الحقيقيّ.
func (s *Server) qaFixedUser(ctx context.Context, phone, role, name, ip string) (string, error) {
	u, err := s.identity.EnsureUserWithRole(ctx, "", phone, role, name, "", ip)
	if err != nil {
		return "", err
	}
	return u.ID, nil
}

// qaCarriesPrivileged **أيحمل الحسابُ دوراً مُمتازاً؟** (admin/ops/finance/owner).
//
// **دفاعٌ**: لا تُصدَر جلسةٌ لهويّةٍ صادفت دوراً خطيراً، ولو كانت هاتفَ QA.
func (s *Server) qaCarriesPrivileged(ctx context.Context, uid string) bool {
	rows, err := s.pg.Query(ctx, `SELECT role_code FROM user_roles WHERE user_id = $1::uuid`, uid)
	if err != nil {
		return true // فشلُ القراءة يُغلق البابَ لا يفتحه
	}
	defer rows.Close()
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return true
		}
		switch role {
		case "admin", "ops", "finance", "owner_super_admin":
			return true
		}
	}
	return rows.Err() != nil
}

// handleQACustomFeePolicy **يضبط سياسةَ أجرة المخصَّص عبر مسار الإعدادات الحقيقيّ**
// (بتحقّقه)، **ويردّ القيمةَ السابقةَ** لاستعادتها. — POST /qa/custom-fee-policy
func (s *Server) handleQACustomFeePolicy(w http.ResponseWriter, r *http.Request) {
	if !s.qaStagingEnabled() {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, err := decode[struct {
		Source          string `json:"source"`
		Fee             int64  `json:"fee"`
		DriverMayChange bool   `json:"driver_may_change"`
	}](r)
	if err != nil {
		s.respondErr(w, errValidation)
		return
	}
	if req.Source != "driver_defined" && req.Source != "admin_defined" {
		s.respondErr(w, errValidation)
		return
	}
	ctx := r.Context()
	prev := map[string]any{
		"source":            s.settings.GetString(ctx, qaFeeSourceKey),
		"fee":               s.settings.GetInt(ctx, qaFeeAmountKey),
		"driver_may_change": s.settings.GetBool(ctx, qaFeeMayChangeKey),
	}
	// **المسارُ الحقيقيّ** — `settings.Set` يمرّ بـ`Validate` (المعجم)، لا كتابةَ خام.
	if err := s.settings.Set(ctx, qaFeeSourceKey, req.Source, nil); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.settings.Set(ctx, qaFeeAmountKey, req.Fee, nil); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.settings.Set(ctx, qaFeeMayChangeKey, req.DriverMayChange, nil); err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA custom fee policy set (staging-only)",
		"source", req.Source, "fee", req.Fee, "may_change", req.DriverMayChange)
	httpx.JSON(w, http.StatusOK, map[string]any{"previous": prev, "set": req})
}

// handleQACustomDriverSession **يُصدر جلسةً لسائق QA الثابت وحدَه** — على التجهيز.
// POST /qa/driver-session
func (s *Server) handleQACustomDriverSession(w http.ResponseWriter, r *http.Request) {
	if !s.qaStagingEnabled() {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	ctx := r.Context()
	uid, err := s.qaFixedUser(ctx, qaStagingDriverPhone, "driver", "سائق الاختبار QA", clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **دفاعٌ**: لا جلسةَ سائقٍ لهويّةٍ صادفت دوراً مُمتازاً.
	if s.qaCarriesPrivileged(ctx, uid) {
		s.logger.Warn("QA driver session REFUSED — account carries a privileged role", "user", uid)
		s.respondErr(w, httpx.NewError(http.StatusForbidden, "qa_not_customer_only", "errors.forbidden"))
		return
	}
	// **وتُضبَط كلمتُه ليدخل تطبيقُ السائق بالهاتف+الكلمة** (نفسُها للزبون)
	// — الدخولُ بالكلمة لا يُقيَّد بقائمة، والهاتفُ ثابتٌ معزول.
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
	s.logger.Warn("QA driver session issued (staging-only)", "user", uid)
	httpx.JSON(w, http.StatusOK, res)
}

// handleQACustomAgree **يوثّق عرضَ السائق (بضاعة+أجرة) بلا تأكيدِ الزبون** —
// فيترك الطلبَ «بانتظار تأكيد الزبون». POST /qa/custom-agree
//
// **المسارُ الحقيقيّ** (`AgreeCustom` بفاعل السائق المُسنَد) — سلطةُ الأجرة
// من اللقطة، وتزايدُ النسخة، والبثّ اللحظيّ. **مقصورٌ على طلب زبون QA المفتوح.**
func (s *Server) handleQACustomAgree(w http.ResponseWriter, r *http.Request) {
	if !s.qaStagingEnabled() {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, err := decode[struct {
		Goods int64 `json:"goods"`
		Fee   int64 `json:"fee"`
	}](r)
	if err != nil {
		s.respondErr(w, errValidation)
		return
	}
	ctx := r.Context()
	cust, ok := s.qaCustomerUID(w, r)
	if !ok {
		return
	}
	orderID, err := s.qaResolveOpenOrder(ctx, cust)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **يوثّق بفاعل السائق المُسنَد** — لا سائقَ عشوائيّ.
	var driverID *string
	var kind string
	if err := s.pg.QueryRow(ctx,
		`SELECT driver_id::text, kind FROM orders WHERE id = $1`, orderID).Scan(&driverID, &kind); err != nil {
		s.respondErr(w, err)
		return
	}
	if kind != "custom" || driverID == nil {
		s.respondErr(w, httpx.NewError(http.StatusConflict, "qa_not_qa_order", "errors.conflict"))
		return
	}
	if err := s.orders.AgreeCustom(ctx, orderID, *driverID, req.Goods, req.Fee); err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA custom agree (staging-only, no auto-confirm)", "order", orderID, "goods", req.Goods, "fee", req.Fee)
	httpx.JSON(w, http.StatusOK, map[string]any{"order_id": orderID, "goods": req.Goods, "fee": req.Fee, "awaiting": "customer_confirmation"})
}

// handleQACustomOverride **يستدعي مسارَ تدخّل الأدمن الحقيقيّ** على طلب زبون QA —
// بفاعل أدمن QA الثابت. POST /qa/custom-override
//
// **لا يتجاوز منطقَ العمل**: `AdminOverrideCustomQuote` نفسُها التي يناديها
// معالِجُ الأدمن — تدقيقٌ وبثٌّ وقيودُ ما قبل/بعد الاستلام.
func (s *Server) handleQACustomOverride(w http.ResponseWriter, r *http.Request) {
	if !s.qaStagingEnabled() {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, err := decode[struct {
		Goods  int64  `json:"goods"`
		Fee    int64  `json:"fee"`
		Reason string `json:"reason"`
	}](r)
	if err != nil {
		s.respondErr(w, errValidation)
		return
	}
	ctx := r.Context()
	cust, ok := s.qaCustomerUID(w, r)
	if !ok {
		return
	}
	orderID, err := s.qaResolveOpenOrder(ctx, cust)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **أدمنُ QA الثابتُ فاعلَ تدقيقٍ** — لا توكنَ له، ولا يُسجَّل دخولُه.
	adminID, err := s.qaFixedUser(ctx, qaStagingAdminPhone, "admin", "أدمن الاختبار QA", clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	o, err := s.orders.AdminOverrideCustomQuote(ctx, orderID, adminID, req.Goods, req.Fee, req.Reason)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA admin custom-quote override (staging-only)", "order", orderID, "goods", req.Goods, "fee", req.Fee)
	view, verr := orderView("ops", o)
	if verr != nil {
		s.respondErr(w, verr)
		return
	}
	httpx.JSON(w, http.StatusOK, view)
}

// handleQACustomCleanup **مسارُ تنظيفٍ حتميٌّ واحدٌ للشهادة** — POST /qa/custom-cleanup.
//
// يُلغي كلَّ طلبات زبون QA المفتوحة (فيُطلَق الحجزُ عبر مسار التسوية النهائيّ)،
// ويُفرِّغ محفظةَ QA، ويُعيد سياسةَ الأجرة إلى الافتراض. **ويردّ ما بقي ليُتحقَّق:
// حجزٌ صفرٌ ورصيدٌ صفر.**
func (s *Server) handleQACustomCleanup(w http.ResponseWriter, r *http.Request) {
	if !s.qaStagingEnabled() {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	ctx := r.Context()
	cust, ok := s.qaCustomerUID(w, r)
	if !ok {
		return
	}
	// **١· تُلغى الطلباتُ المفتوحة** — الإلغاءُ يُطلق حجزَ المحفظة (settle terminal).
	rows, err := s.pg.Query(ctx,
		`SELECT id::text FROM orders WHERE customer_id = $1::uuid AND closed_at IS NULL`, cust)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			s.respondErr(w, err)
			return
		}
		ids = append(ids, id)
	}
	rows.Close()
	cancelled := 0
	for _, id := range ids {
		if _, err := s.orders.Transition(ctx, cust, []string{"ops"}, id, "cancelled", "QA cleanup"); err == nil {
			cancelled++
		}
	}
	// **٢· تُفرَّغ محفظةُ QA** (المسارُ المُعتمَد).
	drained := int64(0)
	if bal, berr := s.wallet.Balance(ctx, cust); berr == nil && bal > 0 {
		actor := cust
		if _, _, derr := s.wallet.ApplyTxID(ctx, s.pg, cust, -bal, "adjustment", "qa-drain", "QA cleanup drain (staging)", &actor); derr == nil {
			drained = bal
		}
	}
	// **٣· تُعاد سياسةُ الأجرة إلى الافتراض** (سائقٌ يحدّد، صفر، لا تعديل).
	_ = s.settings.Set(ctx, qaFeeSourceKey, "driver_defined", nil)
	_ = s.settings.Set(ctx, qaFeeAmountKey, int64(0), nil)
	_ = s.settings.Set(ctx, qaFeeMayChangeKey, false, nil)
	// **٤· ما بقي** — يُتحقَّق منه: حجزٌ صفرٌ ورصيدٌ صفر.
	var reserved, bal int64
	_ = s.pg.QueryRow(ctx,
		`SELECT COALESCE((SELECT reserved FROM wallets WHERE user_id = $1::uuid), 0),
		        COALESCE((SELECT balance  FROM wallets WHERE user_id = $1::uuid), 0)`, cust).
		Scan(&reserved, &bal)
	var custReserved int64
	_ = s.pg.QueryRow(ctx,
		`SELECT COALESCE(sum(custom_reserved_amount),0) FROM orders WHERE customer_id = $1::uuid`, cust).
		Scan(&custReserved)
	s.logger.Warn("QA custom cleanup (staging-only)", "cancelled", cancelled, "drained", drained)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"cancelled": cancelled, "drained": drained,
		"wallet_reserved": reserved, "wallet_balance": bal, "orders_custom_reserved": custReserved,
	})
}

// qaCustomerUID **يحلّ معرّفَ زبون QA الثابت** — أو يردّ ٤٠٤ إن لم يوجد بعد.
func (s *Server) qaCustomerUID(w http.ResponseWriter, r *http.Request) (string, bool) {
	var uid string
	if err := s.pg.QueryRow(r.Context(),
		`SELECT id::text FROM users WHERE phone = $1`, qaStagingPhone).Scan(&uid); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return "", false
	}
	return uid, true
}
