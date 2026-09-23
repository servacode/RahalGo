package server

// ══════════════════════════════════════════════════════════════════════
// **بابُ جلسةٍ للاختبار الحيّ — على التجهيز وحدَه، ويسقط مغلقاً في الإنتاج**
// ══════════════════════════════════════════════════════════════════════
//
// (لتمكين الاختبار الآليّ الحيّ لتطبيق الزبون على التجهيز بلا OTP يدويّ —
//  قرارُ المالك: قدرةٌ QA دائمةٌ على التجهيز.)
//
// # لماذا وكيف يُؤمَّن
//
// **لا يُسجَّل مسارُه إلّا على التجهيز** (`APP_ENV=staging` و`RAHALGO_STAGING=1`)
// — فلا وجودَ له في الإنتاج أصلاً (`server.go`). **ويُتحقَّق ثانيةً وقتَ
// الطلب** دفاعاً في العمق: بيئةٌ غيرُ التجهيز ⇒ `404` كأنّه غيرُ موجود.
//
// **ولا يكشف سرّاً**: يُصدر جلسةَ زبونٍ عاديّةً كأيّ دخولٍ ناجح، لزبون QA
// ثابتٍ مبذور — **بلا كلمةِ إنتاج، ولا سرِّ توقيع، ولا بابِ أدمن، ولا رمز.**
// **والإصدارُ يُسجَّل** (سطرُ تحذيرٍ) فيبقى مسموعاً.

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/offers"
	"github.com/servacode/rahalgo/backend/internal/platform"
	"github.com/servacode/rahalgo/backend/internal/support"
)

// qaStagingPhone **زبونُ QA الثابت** — رقمٌ محجوزٌ للاختبار على التجهيز.
//
// **ومختارٌ بعيداً عن أرقام البذور** (الأدمن `+963999…`، الطاقم `+963955…`،
// وأرقامُ اختبارِ الأطوار `+96390000000x`) — **فلا يُصدَر بالخطأ سِمةُ حسابٍ
// مُمتاز.** وحارسُ الدور أدناه يمنع ذلك على كلّ حال.
const qaStagingPhone = "+963900555001"

// qaStagingPhone2 **زبونُ QA الثاني** — لشهود «لا تسريبَ بين الحسابات»
// (CUST-15-011). زبونٌ محضٌ كالأوّل، بعيدٌ عن أرقام البذور، وحارسُ الدور
// يمنع أيَّ امتيازٍ عنه أيضاً.
const qaStagingPhone2 = "+963900555002"

// qaStagingEnabled **أعلى التجهيز نحن؟** — الشرطان معاً، لا أحدُهما.
func (s *Server) qaStagingEnabled() bool {
	return s.cfg.Env == "staging" && os.Getenv("RAHALGO_STAGING") == "1"
}

// handleQAStagingSession يُصدر جلسةَ زبون QA — على التجهيز وحدَه.
func (s *Server) handleQAStagingSession(w http.ResponseWriter, r *http.Request) {
	// **دفاعٌ في العمق**: ولو سُجّل المسارُ خطأً في غير التجهيز، يسقط مغلقاً.
	if !s.qaStagingEnabled() {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}

	// **زبونٌ ثانٍ عند الطلب** (`{"second":true}`) — لشهود عزلِ الحسابات.
	sel := qaStagingPhone
	if req, derr := decode[struct {
		Second bool `json:"second"`
	}](r); derr == nil && req.Second {
		sel = qaStagingPhone2
	}
	phone, ok := identity.NormalizePhone(sel)
	if !ok {
		s.respondErr(w, errValidation)
		return
	}
	name := "زبون الاختبار QA"
	if sel == qaStagingPhone2 {
		name = "زبون الاختبار QA الثاني"
	}

	// **يُبحَث عنه أولاً** — **فلا يُعاد منحُ الدور على القائم** (يُدرج
	// `granted_by` فارغاً فيسقط)، ولا يُنشأ إلّا مرّةً.
	var uid string
	err := s.pg.QueryRow(r.Context(), `SELECT id::text FROM users WHERE phone = $1`, phone).Scan(&uid)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		user, cerr := s.identity.EnsureUserWithRole(r.Context(), "", sel, "customer", name, "", clientIP(r))
		if cerr != nil {
			s.respondErr(w, cerr)
			return
		}
		uid = user.ID
	case err != nil:
		s.respondErr(w, err)
		return
	}

	// ══════════════════════════════════════════════════════════════════
	// **حارسُ الدور — لا تُصدَر جلسةٌ إلّا لزبونٍ محض** (لا أدمن ولا طاقم)
	// ══════════════════════════════════════════════════════════════════
	//
	// **دفاعٌ لو اصطدم رقمُ QA بحسابٍ مُمتازٍ مبذور** — **فلا يُصنَع توكنُ
	// أدمنٍ من بابِ الاختبار.** أيُّ دورٍ غيرِ `customer` ⇒ رفضٌ مغلق.
	rows, rerr := s.pg.Query(r.Context(), `SELECT role_code FROM user_roles WHERE user_id = $1::uuid`, uid)
	if rerr != nil {
		s.respondErr(w, rerr)
		return
	}
	privileged := false
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			rows.Close()
			s.respondErr(w, err)
			return
		}
		if role != "customer" {
			privileged = true
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	if privileged {
		s.logger.Warn("QA staging session REFUSED — account carries a privileged role", "user", uid)
		s.respondErr(w, httpx.NewError(http.StatusForbidden, "qa_not_customer_only", "errors.forbidden"))
		return
	}

	// **جلسةٌ عاديّة** — عائلةُ الجلسة القائمةُ إن وُجدت وإلّا جديدة.
	sid := s.identity.ActiveSessionID(r.Context(), uid)
	if sid == "" {
		sid = uuid.NewString()
	}
	res, err := s.identity.IssueForUserID(r.Context(), uid, sid, r.UserAgent(), clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA staging session issued (staging-only)", "user", uid, "ip", clientIP(r))
	httpx.JSON(w, http.StatusOK, res)
}

// handleQAStagingRevoke يُبطل جلساتِ رقمٍ من قائمةِ QA المسموحة — على التجهيز وحدَه.
//
// **لتنظيفِ أثرِ جلسةٍ من بناءٍ سابق** (كجلسةٍ صدرت خطأً لحسابٍ مُمتاز).
// **ولا يمسّ إلّا رقمَين QA مسموحَين** — لا حسابَ إنتاجٍ ولا سواه، فلا يصير
// بابَ تعطيلٍ لأحد.
func (s *Server) handleQAStagingRevoke(w http.ResponseWriter, r *http.Request) {
	if !s.qaStagingEnabled() {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, err := decode[struct {
		Phone string `json:"phone"`
	}](r)
	if err != nil {
		s.respondErr(w, errValidation)
		return
	}
	phone, ok := identity.NormalizePhone(req.Phone)
	if !ok {
		s.respondErr(w, errValidation)
		return
	}
	allowed := map[string]bool{}
	for _, p := range []string{qaStagingPhone, qaStagingPhone2, "+963900000001"} { // زبونا QA + الرقمُ المُصطدَمُ في البناء المؤقّت
		if n, nok := identity.NormalizePhone(p); nok {
			allowed[n] = true
		}
	}
	if !allowed[phone] {
		s.respondErr(w, httpx.NewError(http.StatusForbidden, "qa_phone_not_allowed", "errors.forbidden"))
		return
	}
	var uid string
	err = s.pg.QueryRow(r.Context(), `SELECT id::text FROM users WHERE phone = $1`, phone).Scan(&uid)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.JSON(w, http.StatusOK, map[string]any{"revoked": 0, "note": "no such user"})
		return
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}
	tag, err := s.pg.Exec(r.Context(),
		`UPDATE refresh_tokens SET revoked_at = now()
		  WHERE user_id = $1::uuid AND revoked_at IS NULL AND expires_at > now()`, uid)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA staging sessions revoked (staging-only)", "user", uid, "count", tag.RowsAffected())
	httpx.JSON(w, http.StatusOK, map[string]any{"revoked": tag.RowsAffected()})
}

// qaFlagAllowlist **مفاتيحُ الرايات المسموحُ قلبُها في اختبار القبول** —
// **نفسُ ما يقلبه `stagingctl`** (رايات الإطلاق ودوامُ المنصّة): كلُّها
// منطقيّةٌ (`bool`)، وكلُّها لازمةٌ لشهود A–F (فتحُ الطلب، إلخ).
var qaFlagAllowlist = map[string]bool{
	"launch.customer_signup":        true,
	"launch.customer_browse":        true,
	"launch.customer_orders":        true,
	"launch.customer_custom_orders": true,
	"launch.merchant_orders":        true,
	"hours.platform_enforced":       true,
}

// handleQAStagingSetting يقرأ رايةً مسموحةً ويقلبها — على التجهيز وحدَه.
//
// **يُرجع القيمةَ السابقةَ** (لـ`RESTORE`) ثمّ يضع الجديدة. **قائمةُ سماحٍ
// صارمة** (رايات الإطلاق/الدوام فقط) — لا مفتاحَ ماليّ ولا سواه. **يسقط
// مغلقاً في الإنتاج** (المسارُ غيرُ مسجَّل + حارسٌ ثانٍ).
func (s *Server) handleQAStagingSetting(w http.ResponseWriter, r *http.Request) {
	if !s.qaStagingEnabled() {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, err := decode[struct {
		Key   string `json:"key"`
		Value bool   `json:"value"`
	}](r)
	if err != nil {
		s.respondErr(w, errValidation)
		return
	}
	if !qaFlagAllowlist[req.Key] {
		s.respondErr(w, httpx.NewError(http.StatusForbidden, "qa_flag_not_allowed", "errors.forbidden"))
		return
	}
	prev := s.settings.GetBool(r.Context(), req.Key)
	if err := s.settings.Set(r.Context(), req.Key, req.Value, nil); err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA staging flag set (staging-only)", "key", req.Key, "previous", prev, "set", req.Value)
	httpx.JSON(w, http.StatusOK, map[string]any{"key": req.Key, "previous": prev, "set": req.Value})
}

// ══════════════════════════════════════════════════════════════════════
// **بذّارُ عتادِ الاختبار — على التجهيز وحدَه، على بيانات زبون QA فقط**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٩-٢٢: «ابنِ بذّاراتٍ ضيّقةً على التجهيز» — لشهود
//  A/D/E التي تحتاج عتادَ أدمن/متجرٍ لا يصنعه بابُ QA الزبونيّ.)
//
// # حدودُه
//
// **يسقط مغلقاً في الإنتاج** (المسارُ غيرُ مسجَّلٍ + حارسٌ ثانٍ). **ولا
// يُصدِر أيَّ توكن أدمن**: معرّفُ الموظّف يُستعمل مفتاحاً خارجيّاً (كاتبَ
// الردّ ومُصدِرَ الإنذار) لا جلسةً. **وكلُّ ما يبذره يخصّ زبونَ QA
// وحدَه** — تذكرتُه وإنذارُه؛ **إلّا العرضَ فعامٌّ بطبعه**، فيُبذَر قصيرَ
// الأجل (٤٥ دقيقة) فلا يبقى معلّقاً، ويُطفأ صراحةً بـ`offer_off`.

// qaSeedAllowlist أنواعُ العتاد المسموحةُ بذرُها.
var qaSeedAllowlist = map[string]bool{
	"ticket_reply":   true, // ردُّ أدمن على تذكرة زبون QA نفسِه (E-013)
	"resolve_ticket": true, // حلُّ تذكرة زبون QA — لشهود «المحلولةُ تُغلَق» (SUP-014، بلا تعويضٍ فلا مساسَ ماليّ)
	"warning":        true, // إنذارُ حسابٍ على زبون QA (D-corroboration)
	"push":           true, // دفعةٌ حتميّةٌ لزبون QA (المسار E) — kind/entity للوجهة
	"wallet_fund":    true, // شحنُ محفظة زبون QA بمبلغٍ حتميّ (المسار D) — عبر wallet.ApplyTx
	"wallet_drain":   true, // تصفيرُ محفظة زبون QA (تنظيفٌ ماليّ) — عبر wallet.ApplyTx
	"offer":          true, // عرضُ خصمٍ حيٌّ قصيرُ الأجل على صنفٍ (A/ENG-005)
	"offer_off":      true, // إطفاءُ عرضٍ بذرناه (تنظيف)
	// ── حالاتُ العتاد (المسار B) — كلُّها تُرجع القيمةَ السابقةَ للاستعادة ──
	"item_available": true, // إتاحةُ/إيقافُ صنف (menu_items.available)
	"item_price":     true, // تغييرُ سعرِ صنف (menu_items.price)
	"item_name":      true, // تغييرُ اسمِ صنف (menu_items.name) — شهودُ التفاف الاسم الطويل، عكوسٌ

	"section_active":     true, // تفعيلُ/تعطيلُ قسم (platform_sections.active)
	"zone_active":        true, // فتحُ/إغلاقُ منطقةِ تغطية (delivery_zones.active)
	"platform_pause":     true, // إيقافٌ مؤقّتٌ للمنصّة (service_closure → temporarily_unavailable)
	"merchant_emergency": true, // إغلاقُ/فتحُ متجرِ صنفٍ طارئاً (merchants.emergency_closed)
	// ── حاقنُ الأعطال (المسار C) — انظر `qa_fault.go` ──
	"fault_arm":    true, // تسليحُ عطبٍ على مسار (error_5xx | latency)
	"fault_clear":  true, // نزعُ عطبٍ (أو الكلّ)
	"fault_status": true, // قراءةُ المسلَّح
	// ── عتادُ كثافةٍ للأداء (المسار G) ──
	"fixture_dense":       true, // بذرُ أصنافٍ كثيفةٍ (يستنسخ مراجعَ صنفٍ قالب)
	"fixture_dense_clear": true, // حذفُ كلّ أصناف QA_DENSE
	// ── نظيرُ السائق (المسار B تكملة) — طلبٌ مخصّصٌ نقديٌّ لزبون QA فقط، محايدٌ ماليّاً ──
	"order_advance":   true, // سوقُ طلبِ زبون QA المخصّصِ النقديّ عبر الحالات (يُسنِد سائقاً فعليّاً)
	"order_chat_send": true, // رسالةُ سائقٍ على طلبِ زبون QA (SUP-002) — عبر comms.Send
	// ── دوامُ المنطقة (zone_closed_now) + الحدُّ الأدنى للنسخة (426) — عكوسان ──
	"zone_close":  true, // إغلاقُ منطقةٍ الآن حتميّاً (hours_enforced + جدولٌ فارغ)، يحفظ السابق
	"zone_reopen": true, // إعادةُ جدول المنطقة المحفوظ
	"min_version": true, // ضبطُ app.min_version.customer (يُرجع السابق) لشهود update_required
	// فتحُ متجرِ QA الآن حتميّاً (لطلبٍ عاديٍّ خارجَ الدوام) — عكوسٌ، بلا أثرٍ ماليّ:
	"merchant_open":      true, // حذفُ merchant_hours + رفعُ الطارئ (يحفظ السابق)، بمعرّف صنفٍ
	"merchant_restore":   true, // إعادةُ جدول المتجر والإغلاق الطارئ المحفوظَين
	"gov_active":         true, // قلبُ فعّاليّة محافظةِ نقطةٍ (province_not_supported، 08-007) — عكوسٌ
	"merchant_hours_set": true, // ضبطُ جدول دوامِ متجرٍ من قائمةٍ صريحة (استعادةٌ دقيقة)
	// متجرٌ ثانٍ عكوسٌ لشهود سقفِ المصادر (CUST-11-036) — بلا أثرٍ ماليّ، لا قبولَ متجر:
	"merchant_second":       true, // إنشاءُ متجرٍ ثانٍ صغيرٍ (مصدرٌ ثانٍ)
	"merchant_second_clear": true, // حذفُه (FK-safe)
	"option_available":      true, // قلبُ إتاحةِ خيارِ إضافة (10-014) — عكوسٌ
	"item_image":            true, // تبديلُ صورةِ صنف (09-011) — عكوسٌ
	"customer_suspend":      true, // إيقافُ زبون QA (06-031) — عكوسٌ، بلا إبطالِ جلسة
	"customer_set_password": true, // ضبطُ كلمةِ مرورِ زبون QA لدخول الواجهة
	"customer_restore":      true, // إعادةُ زبون QA إلى active
	// ── الجهازُ التجريبيّ لحذف الحساب (CUST-06-026) + روابطُ التواصل (CUST-ENG-011) ──
	"disposable_create":      true, // زبونٌ منفصلٌ يُستهلك لشهود الحذف (لا يمسّ QA1)
	"disposable_delete_code": true, // رمزُ حذفٍ حقيقيٌّ للجهاز التجريبيّ (يُدعى بعد طلب التطبيق)
	"contact_set":            true, // ضبطُ إعداداتِ التواصل لشهودها (يحفظ السابق) — لا بابَ أدمن
	"contact_clear":          true, // استعادةُ إعداداتِ التواصل السابقة
	"otp_code":               true, // إصدارُ رمزِ OTP لرقمِ QA (04/05/06) — dev يطبع لا يرسل واتساب
}

// qaStateSeed أنواعُ الحالة التي لا تلزمها هويّةُ زبون QA (تُعالَج قبل استخراجه).
var qaStateSeed = map[string]bool{
	"offer_off": true, "item_available": true, "item_price": true,
	"section_active": true, "zone_active": true, "platform_pause": true,
	"merchant_emergency": true, "item_name": true,
	"fault_arm": true, "fault_clear": true, "fault_status": true,
	"fixture_dense": true, "fixture_dense_clear": true,
	// دوامُ المنطقة والحدُّ الأدنى للنسخة لا تلزمها هويّةُ زبون QA:
	"zone_close": true, "zone_reopen": true, "min_version": true,
	"merchant_open": true, "merchant_restore": true, "gov_active": true, "merchant_hours_set": true,
	"merchant_second": true, "merchant_second_clear": true,
	"option_available": true, "item_image": true,
	"customer_suspend": true, "customer_restore": true, "customer_set_password": true,
	"disposable_create": true, "disposable_delete_code": true,
	"contact_set": true, "contact_clear": true, "otp_code": true,
}

// handleQAStagingSeed يبذر عتادَ اختبارٍ لزبون QA — على التجهيز وحدَه.
func (s *Server) handleQAStagingSeed(w http.ResponseWriter, r *http.Request) {
	if !s.qaStagingEnabled() {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, err := decode[struct {
		Kind      string `json:"kind"`
		OfferID   string `json:"offer_id"`
		ItemID    string `json:"item_id"`
		SectionID string `json:"section_id"`
		ZoneID    string `json:"zone_id"`
		ValueBool bool   `json:"value_bool"`
		ValueInt  int64  `json:"value_int"`
		// حاقنُ الأعطال:
		Path  string `json:"path"`  // نقطةُ النهاية (r.URL.Path) المستهدفة
		Mode  string `json:"mode"`  // error_5xx | latency
		Ms    int    `json:"ms"`    // للتأخير
		Count int    `json:"count"` // عددُ الإصابات (افتراضُه ١)
		// دفعةُ QA (push):
		NotifKind string `json:"notif_kind"` // order | chat | offer | account | ticket …
		Title     string `json:"title"`
		Body      string `json:"body"`
		Entity    string `json:"entity"`
		EntityID  string `json:"entity_id"`
		// النظيرُ/السائق (المسار B تكملة) + دوامُ المنطقة + الحدُّ الأدنى للنسخة:
		OrderID   string  `json:"order_id"` // طلبُ زبون QA (order_advance / order_chat_send)
		Target    string  `json:"target"`   // الحالةُ الهدف (order_advance)
		Name      string  `json:"name"`     // اسمُ صنفٍ (item_name — شهودُ التفاف الاسم الطويل 20-005/10-013)
		Lat       float64 `json:"lat"`      // نقطةٌ (gov_active — حلُّ المحافظة، 08-007)
		Lng       float64 `json:"lng"`
		HoursJSON string  `json:"hours_json"` // جدولُ دوامِ متجرٍ صريح (merchant_hours_set — استعادةٌ دقيقة)
		OptionID  string  `json:"option_id"`  // خيارُ إضافةٍ (option_available — 10-014)
		MediaID   string  `json:"media_id"`   // صورةُ صنفٍ للاستعادة (item_image — 09-011)
		Phone     string  `json:"phone"`      // رقمُ QA لإصدار رمزِ OTP (otp_code — 04/05/06)
		Purpose   string  `json:"purpose"`    // signup | reset | whatsapp | delete (otp_code)
	}](r)
	if err != nil {
		s.respondErr(w, errValidation)
		return
	}
	if !qaSeedAllowlist[req.Kind] {
		s.respondErr(w, httpx.NewError(http.StatusForbidden, "qa_seed_kind_not_allowed", "errors.forbidden"))
		return
	}

	// ── حالاتُ العتاد (لا تلزمها هويّةُ زبون QA) — كلُّها تُرجع `previous` ──
	if qaStateSeed[req.Kind] {
		switch req.Kind {
		case "offer_off":
			// **العرضُ يُطفأ بمعرّفه** — تنظيفٌ صريح.
			if req.OfferID == "" {
				s.respondErr(w, errValidation)
				return
			}
			if _, err := s.offers.SetActive(r.Context(), req.OfferID, false, s.saleOf(r)); err != nil {
				s.respondErr(w, err)
				return
			}
			s.logger.Warn("QA staging offer deactivated (staging-only)", "offer", req.OfferID)
			httpx.JSON(w, http.StatusOK, map[string]any{"offer_id": req.OfferID, "active": false})
		case "item_available":
			s.qaSetBool(w, r, "menu_items", "available", req.ItemID, req.ValueBool)
		case "section_active":
			s.qaSetBool(w, r, "platform_sections", "active", req.SectionID, req.ValueBool)
		case "zone_active":
			s.qaSetBool(w, r, "delivery_zones", "active", req.ZoneID, req.ValueBool)
		case "item_price":
			s.qaSetItemPrice(w, r, req.ItemID, req.ValueInt)
		case "item_name":
			s.qaSetItemName(w, r, req.ItemID, req.Name)
		case "platform_pause":
			s.qaPlatformPause(w, r, req.ValueBool)
		case "merchant_emergency":
			s.qaSetMerchantEmergency(w, r, req.ItemID, req.ValueBool)
		case "fault_arm":
			if req.Path == "" || (req.Mode != qaFaultError5xx && req.Mode != qaFaultLatency) {
				s.respondErr(w, errValidation)
				return
			}
			qaFaults.arm(req.Path, req.Mode, req.Ms, req.Count)
			s.logger.Warn("QA fault armed (staging-only)", "path", req.Path, "mode", req.Mode, "ms", req.Ms, "count", req.Count)
			httpx.JSON(w, http.StatusOK, map[string]any{"armed": req.Path, "mode": req.Mode, "ms": req.Ms, "count": req.Count})
		case "fault_clear":
			qaFaults.clear(req.Path) // فارغٌ ⇒ الكلّ
			s.logger.Warn("QA fault cleared (staging-only)", "path", req.Path)
			httpx.JSON(w, http.StatusOK, map[string]any{"cleared": req.Path, "armed_now": qaFaults.snapshot()})
		case "fault_status":
			httpx.JSON(w, http.StatusOK, map[string]any{"armed": qaFaults.snapshot()})
		case "fixture_dense":
			s.qaSeedDense(w, r, req.ItemID, req.Count)
		case "fixture_dense_clear":
			s.qaClearDense(w, r)
		case "zone_close":
			s.qaZoneClose(w, r, req.ZoneID)
		case "zone_reopen":
			s.qaZoneReopen(w, r, req.ZoneID)
		case "min_version":
			s.qaMinVersion(w, r, req.ValueInt)
		case "merchant_open":
			s.qaMerchantOpen(w, r, req.ItemID)
		case "merchant_restore":
			s.qaMerchantRestore(w, r, req.ItemID)
		case "gov_active":
			s.qaGovActive(w, r, req.Lat, req.Lng, req.ValueBool)
		case "merchant_hours_set":
			s.qaMerchantHoursSet(w, r, req.ItemID, req.HoursJSON)
		case "merchant_second":
			s.qaMerchantSecond(w, r)
		case "merchant_second_clear":
			s.qaMerchantSecondClear(w, r)
		case "option_available":
			s.qaOptionAvailable(w, r, req.OptionID, req.ItemID, req.ValueBool)
		case "item_image":
			s.qaItemImage(w, r, req.ItemID, req.MediaID)
		case "customer_suspend":
			s.qaCustomerSuspend(w, r, "suspended")
		case "customer_restore":
			s.qaCustomerSuspend(w, r, "active")
		case "customer_set_password":
			s.qaCustomerSetPassword(w, r)
		case "disposable_create":
			s.qaDisposableCreate(w, r)
		case "disposable_delete_code":
			s.qaDisposableDeleteCode(w, r)
		case "contact_set":
			s.qaContactSet(w, r)
		case "contact_clear":
			s.qaContactClear(w, r)
		case "otp_code":
			s.qaOTPCode(w, r, req.Phone, req.Purpose)
		}
		return
	}

	phone, ok := identity.NormalizePhone(qaStagingPhone)
	if !ok {
		s.respondErr(w, errValidation)
		return
	}
	var uid string
	if err := s.pg.QueryRow(r.Context(), `SELECT id::text FROM users WHERE phone = $1`, phone).Scan(&uid); err != nil {
		s.respondErr(w, err)
		return
	}

	switch req.Kind {
	case "ticket_reply":
		s.qaSeedTicketReply(w, r, uid)
	case "resolve_ticket":
		s.qaResolveTicket(w, r, uid)
	case "warning":
		s.qaSeedWarning(w, r, uid)
	case "offer":
		s.qaSeedOffer(w, r, uid)
	case "push":
		// **دفعةٌ حتميّةٌ لزبون QA** — عبر نفس مسار الإشعارات (`notify` ⇒ الدافع)،
		// فتحمل kind/entity/entity_id للوجهة (deep-link). لا سرَّ، لا موضوعَ إنتاج.
		title := req.Title
		if title == "" {
			title = "إشعار اختبار QA"
		}
		s.notify.Notify(r.Context(), notifications.Input{
			UserID: uid, Kind: req.NotifKind, Title: title, Body: req.Body,
			Entity: req.Entity, EntityID: req.EntityID,
		})
		s.logger.Warn("QA push sent (staging-only)", "user", uid, "kind", req.NotifKind, "entity", req.Entity)
		httpx.JSON(w, http.StatusOK, map[string]any{"sent": true, "kind": req.NotifKind, "entity": req.Entity, "entity_id": req.EntityID})
	case "wallet_fund":
		s.qaWalletFund(w, r, uid, req.ValueInt)
	case "wallet_drain":
		s.qaWalletDrain(w, r, uid)
	case "order_advance":
		s.qaOrderAdvance(w, r, uid, req.OrderID, req.Target)
	case "order_chat_send":
		s.qaOrderChatSend(w, r, uid, req.OrderID, req.Body)
	}
}

// qaWalletFund يشحن محفظةَ زبون QA بمبلغٍ حتميّ **عبر المسار المُعتمَد**
// (`wallet.ApplyTx` — يكتب قيدَ `wallet_transactions` ويفرض CHECK، لا تعديلَ
// قاعدةٍ خام). يُرجع الرصيدَ ومعرّفَ القيد.
func (s *Server) qaWalletFund(w http.ResponseWriter, r *http.Request, uid string, amount int64) {
	if amount <= 0 || amount > 100_000_000 { // حتميٌّ وموجب، بسقفٍ يحرس من خطأٍ عرضيّ
		s.respondErr(w, errValidation)
		return
	}
	actor := uid
	bal, txID, err := s.wallet.ApplyTxID(r.Context(), s.pg, uid, amount, "topup", "qa-fund", "QA wallet funding (staging)", &actor)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **يُبثّ حدثُ المحفظة اللحظيّ كما يفعل مسارُ الأدمن الحقيقيّ**
	// (`handleAdminWalletApply` ⇒ `touchUser(id, "wallet")`): البذّارُ كان يكتب
	// القيدَ فقط بلا بثٍّ، فبدا رصيدُ الواجهة جامداً — وهو أثرُ عتادٍ لا عطبُ
	// منتَج. **فيُطابَق المسارُ الإنتاجيّ** ليُشهَد التحديثُ اللحظيّ (CUST-WAL-010).
	s.touchUser(uid, "wallet")
	s.logger.Warn("QA wallet funded (staging-only)", "user", uid, "amount", amount, "balance", bal, "tx", txID)
	httpx.JSON(w, http.StatusOK, map[string]any{"funded": amount, "balance": bal, "tx_id": txID, "kind": "topup"})
}

// qaWalletDrain يُصفّر محفظةَ زبون QA (تنظيفٌ ماليّ) بقيدِ تسويةٍ سالبٍ
// بمقدار الرصيد — عبر المسار المُعتمَد، لا تعديلَ خام.
func (s *Server) qaWalletDrain(w http.ResponseWriter, r *http.Request, uid string) {
	bal, err := s.wallet.Balance(r.Context(), uid)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if bal == 0 {
		httpx.JSON(w, http.StatusOK, map[string]any{"drained": 0, "balance": 0})
		return
	}
	actor := uid
	newBal, txID, derr := s.wallet.ApplyTxID(r.Context(), s.pg, uid, -bal, "adjustment", "qa-drain", "QA wallet drain (staging cleanup)", &actor)
	if derr != nil {
		s.respondErr(w, derr)
		return
	}
	s.touchUser(uid, "wallet") // بثٌّ لحظيٌّ كمسار الأدمن — الرصيدُ تغيّر
	s.logger.Warn("QA wallet drained (staging-only)", "user", uid, "removed", bal, "balance", newBal, "tx", txID)
	httpx.JSON(w, http.StatusOK, map[string]any{"drained": bal, "balance": newBal, "tx_id": txID})
}

// qaResolveTicket يحلّ أحدثَ تذكرةٍ مفتوحةٍ لزبون QA — **بلا تعويضٍ فلا يُمسّ
// دفترُ المال** — لشهود «المحلولةُ تُغلَق ولا تُردّ» (SUP-014، `ticket_resolved`).
func (s *Server) qaResolveTicket(w http.ResponseWriter, r *http.Request, uid string) {
	var ticketID string
	err := s.pg.QueryRow(r.Context(),
		`SELECT id::text FROM tickets WHERE customer_id = $1::uuid AND status IN ('open','in_progress')
		 ORDER BY created_at DESC LIMIT 1`, uid).Scan(&ticketID)
	if errors.Is(err, pgx.ErrNoRows) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	} else if err != nil {
		s.respondErr(w, err)
		return
	}
	actor, aerr := s.qaNonCustomerAuthor(r.Context(), uid)
	if aerr != nil {
		s.respondErr(w, aerr)
		return
	}
	if _, err := s.support.Resolve(r.Context(), actor, ticketID,
		"عولجت — عتادُ اختبار QA", 0, clientIP(r)); err != nil { // تعويض 0 — لا قيدَ ماليّ
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA staging ticket resolved (staging-only)", "ticket", ticketID)
	httpx.JSON(w, http.StatusOK, map[string]any{"ticket_id": ticketID, "kind": "resolve_ticket", "status": "resolved"})
}

// qaNonCustomerAuthor **معرّفُ موظّفٍ يصلح كاتبَ ردٍّ أو مُصدِرَ إنذار** —
// **معرّفٌ لا توكن.** أيُّ حسابٍ يحمل دوراً غيرَ `customer`، فإن لم يوجد
// فأيُّ حسابٍ سوى زبون QA (ليصير `mine=false` في عرض الزبون).
func (s *Server) qaNonCustomerAuthor(ctx context.Context, excludeUID string) (string, error) {
	var id string
	err := s.pg.QueryRow(ctx, `
		SELECT u.id::text FROM users u
		WHERE u.id <> $1::uuid
		  AND EXISTS (SELECT 1 FROM user_roles ur WHERE ur.user_id = u.id AND ur.role_code <> 'customer')
		ORDER BY u.created_at LIMIT 1`, excludeUID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		err = s.pg.QueryRow(ctx,
			`SELECT id::text FROM users WHERE id <> $1::uuid ORDER BY created_at LIMIT 1`, excludeUID).Scan(&id)
	}
	return id, err
}

// qaSeedTicketReply يضمن تذكرةً مفتوحةً لزبون QA ثمّ يضيف ردَّ موظّفٍ عليها.
func (s *Server) qaSeedTicketReply(w http.ResponseWriter, r *http.Request, uid string) {
	var ticketID string
	err := s.pg.QueryRow(r.Context(),
		`SELECT id::text FROM tickets WHERE customer_id = $1::uuid AND status IN ('open','in_progress')
		 ORDER BY created_at DESC LIMIT 1`, uid).Scan(&ticketID)
	if errors.Is(err, pgx.ErrNoRows) {
		if ierr := s.pg.QueryRow(r.Context(), `
			INSERT INTO tickets (customer_id, subject, reason, created_by, opened_by_customer)
			VALUES ($1::uuid, $2, $3, $1::uuid, true) RETURNING id::text`,
			uid, "استفسار اختبار QA", "other").Scan(&ticketID); ierr != nil {
			s.respondErr(w, ierr)
			return
		}
	} else if err != nil {
		s.respondErr(w, err)
		return
	}
	author, aerr := s.qaNonCustomerAuthor(r.Context(), uid)
	if aerr != nil {
		s.respondErr(w, aerr)
		return
	}
	if _, err := s.support.Reply(r.Context(), author, ticketID,
		"شكراً لتواصلك مع رحّال غو — استلمنا رسالتك ونعمل عليها. (ردُّ عتادِ اختبار QA)"); err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA staging ticket reply seeded (staging-only)", "ticket", ticketID, "author", author)
	httpx.JSON(w, http.StatusOK, map[string]any{"ticket_id": ticketID, "kind": "ticket_reply"})
}

// qaSeedWarning يُنشئ إنذارَ حسابٍ على زبون QA ويُشعره — كبابِ الأدمن تماماً.
func (s *Server) qaSeedWarning(w http.ResponseWriter, r *http.Request, uid string) {
	const reason = "wrong_address" // سببٌ زبونيٌّ صالحٌ في `WarnReasonsFor("customer")`
	issuedBy, aerr := s.qaNonCustomerAuthor(r.Context(), uid)
	if aerr != nil {
		s.respondErr(w, aerr)
		return
	}
	var id string
	if err := s.pg.QueryRow(r.Context(), `
		INSERT INTO warnings (user_id, role_code, reason, note, issued_by)
		VALUES ($1::uuid, 'customer', $2, $3, $4::uuid) RETURNING id::text`,
		uid, reason, "إنذارُ عتادِ اختبار QA", issuedBy).Scan(&id); err != nil {
		s.respondErr(w, err)
		return
	}
	body := support.WarnReasonAr(reason) + " — إنذارُ عتادِ اختبار QA"
	s.notify.Notify(r.Context(), notifications.Input{
		UserID: uid, Kind: notifications.KindAccount,
		Title: notifTitles.warningOnYou, Body: clip(body, 200),
		Entity: "user", EntityID: uid,
	})
	s.logger.Warn("QA staging warning seeded (staging-only)", "user", uid, "warning", id)
	httpx.JSON(w, http.StatusOK, map[string]any{"warning_id": id, "kind": "warning"})
}

// qaSeedOffer يُنشئ عرضَ خصمٍ حيّاً قصيرَ الأجل على صنفٍ بلا عرضٍ قائم.
func (s *Server) qaSeedOffer(w http.ResponseWriter, r *http.Request, uid string) {
	var itemID string
	err := s.pg.QueryRow(r.Context(), `
		SELECT i.id::text FROM menu_items i
		WHERE i.approved AND i.available
		  AND NOT EXISTS (SELECT 1 FROM offers o WHERE o.menu_item_id = i.id AND o.active)
		ORDER BY i.id LIMIT 1`).Scan(&itemID)
	if errors.Is(err, pgx.ErrNoRows) {
		s.respondErr(w, httpx.NewError(http.StatusConflict, "qa_no_free_item", "errors.conflict"))
		return
	} else if err != nil {
		s.respondErr(w, err)
		return
	}
	author, aerr := s.qaNonCustomerAuthor(r.Context(), uid)
	if aerr != nil {
		s.respondErr(w, aerr)
		return
	}
	pct := 20
	borne := offers.ByPlatform
	ends := time.Now().Add(45 * time.Minute) // **قصيرُ الأجل** — عرضٌ عامٌّ لا يبقى معلّقاً
	o, err := s.offers.Create(r.Context(), author, offers.Input{
		Title:           "QA اختبار — عرض تجريبي",
		MenuItemID:      &itemID,
		DiscountPercent: &pct,
		BorneBy:         &borne,
		EndsAt:          &ends,
	}, s.saleOf(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA staging offer seeded (staging-only)", "offer", o.ID, "item", itemID)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"offer_id": o.ID, "item_id": itemID, "kind": "offer", "expires_at": ends.Format(time.RFC3339),
	})
}

// ══════════════════════════════════════════════════════════════════════
// **بذّارُ حالةِ العتاد — إتاحةٌ/سعرٌ/قسمٌ/منطقةٌ/إيقافٌ مؤقّت** (المسار B)
// ══════════════════════════════════════════════════════════════════════
//
// **كلُّها تُرجع `previous`** فيستعيدها المُختبِرُ بندائه ثانيةً بالقيمة السابقة.
// **الجدولُ والعمودُ حرفان ثابتان من `switch`** لا من الطلب، **والمعرّفُ يُتحقَّق
// أنّه `UUID`** — فلا حقنَ. **على التجهيز وحدَه** (الحارسُ في المنادي).

// qaSetBool يقلب عموداً منطقيّاً في صفٍّ بمعرّفه، ويُرجع القيمةَ السابقة.
func (s *Server) qaSetBool(w http.ResponseWriter, r *http.Request, table, col, id string, val bool) {
	if !isUUID(id) {
		s.respondErr(w, errValidation)
		return
	}
	var prev bool
	// **الجدولُ/العمودُ من قائمةِ `switch` الثابتة لا من المستخدم** — لا حقن.
	// **نقرأ السابقَ ثمّ نكتب** — أبسطُ من CTE وكافٍ على التجهيز.
	if err := s.pg.QueryRow(r.Context(),
		`SELECT `+col+` FROM `+table+` WHERE id = $1::uuid`, id).Scan(&prev); err != nil {
		s.respondErr(w, err)
		return
	}
	if _, err := s.pg.Exec(r.Context(),
		`UPDATE `+table+` SET `+col+` = $2 WHERE id = $1::uuid`, id, val); err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA staging state set (staging-only)", "table", table, "col", col, "id", id, "previous", prev, "set", val)
	httpx.JSON(w, http.StatusOK, map[string]any{"table": table, "col": col, "id": id, "previous": prev, "set": val})
}

// qaSetItemPrice يضبط سعرَ صنفٍ ويُرجع السعرَ السابق.
//
// **العمودُ `merchant_price`** — وهو ما يقرؤه المحرّكُ والعرضُ (`SalePrice`)،
// **لا `price`** القديمَ المهمَل. (تصحيحُ ٢٠٢٦-٠٩-٢٢.)
func (s *Server) qaSetItemPrice(w http.ResponseWriter, r *http.Request, id string, price int64) {
	if !isUUID(id) || price < 0 {
		s.respondErr(w, errValidation)
		return
	}
	var prev int64
	if err := s.pg.QueryRow(r.Context(),
		`SELECT merchant_price FROM menu_items WHERE id = $1::uuid`, id).Scan(&prev); err != nil {
		s.respondErr(w, err)
		return
	}
	if _, err := s.pg.Exec(r.Context(),
		`UPDATE menu_items SET merchant_price = $2 WHERE id = $1::uuid`, id, price); err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA staging item price set (staging-only)", "id", id, "previous", prev, "set", price)
	httpx.JSON(w, http.StatusOK, map[string]any{"item_id": id, "previous": prev, "set": price})
}

// qaSetItemName **يضبط اسمَ صنفٍ** (menu_items.name) لشهود التفاف/قصّ الاسم
// الطويل في الواجهة (20-005/10-013). **يُرجع الاسمَ السابقَ للاستعادة** — نداءٌ
// ثانٍ بقيمته يُعيد الحال. **على التجهيز، على صنفٍ بمعرّفه، لا أثرَ ماليّ.**
func (s *Server) qaSetItemName(w http.ResponseWriter, r *http.Request, id, name string) {
	// **حدٌّ سخيٌّ يحرس من نصٍّ لا نهائيّ** — والطولُ المقصودُ للالتفاف نحو ٨٠–١٥٠ حرفاً.
	if !isUUID(id) || name == "" || len([]rune(name)) > 300 {
		s.respondErr(w, errValidation)
		return
	}
	var prev string
	if err := s.pg.QueryRow(r.Context(),
		`SELECT name FROM menu_items WHERE id = $1::uuid`, id).Scan(&prev); err != nil {
		s.respondErr(w, err)
		return
	}
	if _, err := s.pg.Exec(r.Context(),
		`UPDATE menu_items SET name = $2 WHERE id = $1::uuid`, id, name); err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA staging item name set (staging-only)", "id", id, "prev_len", len([]rune(prev)), "set_len", len([]rune(name)))
	httpx.JSON(w, http.StatusOK, map[string]any{"item_id": id, "previous": prev, "set": name})
}

// qaSetMerchantEmergency يغلق/يفتح متجرَ صنفٍ طارئاً (merchants.emergency_closed)،
// ويُرجع الحالَ السابقة — لشهود «المتجر مغلق حالياً» (08-012/11-026).
func (s *Server) qaSetMerchantEmergency(w http.ResponseWriter, r *http.Request, itemID string, closed bool) {
	if !isUUID(itemID) {
		s.respondErr(w, errValidation)
		return
	}
	var mid string
	if err := s.pg.QueryRow(r.Context(),
		`SELECT merchant_id::text FROM menu_items WHERE id = $1::uuid`, itemID).Scan(&mid); err != nil {
		s.respondErr(w, err)
		return
	}
	var prev bool
	if err := s.pg.QueryRow(r.Context(),
		`SELECT emergency_closed FROM merchants WHERE id = $1::uuid`, mid).Scan(&prev); err != nil {
		s.respondErr(w, err)
		return
	}
	if _, err := s.pg.Exec(r.Context(),
		`UPDATE merchants SET emergency_closed = $2 WHERE id = $1::uuid`, mid, closed); err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA staging merchant emergency set (staging-only)", "merchant", mid, "previous", prev, "set", closed)
	httpx.JSON(w, http.StatusOK, map[string]any{"merchant_id": mid, "item_id": itemID, "previous": prev, "set": closed})
}

// qaPlatformPause يضبط الإيقافَ المؤقّت للمنصّة، ويُرجع الحالَ السابقة.
func (s *Server) qaPlatformPause(w http.ResponseWriter, r *http.Request, active bool) {
	prev, err := s.platform.Closure(r.Context(), s.pg)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	msg := ""
	if active {
		msg = "إيقافٌ مؤقّتٌ للاختبار — عتادُ QA"
	}
	if _, err := s.platform.SetClosure(r.Context(), "", platform.Closure{Active: active, Message: msg}); err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA staging platform pause set (staging-only)", "previous", prev.Active, "set", active)
	httpx.JSON(w, http.StatusOK, map[string]any{"previous_active": prev.Active, "previous_message": prev.Message, "set": active})
}

// ══════════════════════════════════════════════════════════════════════
// **عتادُ كثافةٍ للأداء — أصنافٌ كثيرةٌ في قسمٍ واحد** (المسار G)
// ══════════════════════════════════════════════════════════════════════
//
// (عقدُ P-8: jank التمرير/التنقّل يلزمه ٣٠–٥٠ صنفاً.) **يستنسخ مراجعَ صنفٍ
// قالبٍ قائم** (متجرُه وقسمُه ومنصّةُ قسمه) فلا يخترع بنيةً — أصنافُ QA_DENSE
// معتمَدةٌ ومتاحةٌ لتظهر في السوق. **تُحذَف كلُّها بـ`fixture_dense_clear`.**

// qaSeedDense يبذر `count` صنفاً باستنساخ مراجع صنفٍ قالب.
func (s *Server) qaSeedDense(w http.ResponseWriter, r *http.Request, template string, count int) {
	if !isUUID(template) {
		s.respondErr(w, errValidation)
		return
	}
	if count <= 0 || count > 60 { // سقفٌ يحرس من كثافةٍ عرضيّة
		count = 40
	}
	var psid *string
	if err := s.pg.QueryRow(r.Context(),
		`SELECT platform_section_id::text FROM menu_items WHERE id = $1::uuid`, template).Scan(&psid); err != nil {
		s.respondErr(w, err)
		return
	}
	tag, err := s.pg.Exec(r.Context(), `
		INSERT INTO menu_items
		    (merchant_id, section_id, platform_section_id, name, price, merchant_price, available, approved, sort_order)
		SELECT t.merchant_id, t.section_id, t.platform_section_id,
		       'QA_DENSE_' || g::text, t.price, t.merchant_price, true, true, 9000 + g
		  FROM menu_items t, generate_series(1, $2) g
		 WHERE t.id = $1::uuid`, template, count)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA dense fixture seeded (staging-only)", "count", tag.RowsAffected(), "template", template)
	httpx.JSON(w, http.StatusOK, map[string]any{"seeded": tag.RowsAffected(), "platform_section_id": psid, "template": template})
}

// qaClearDense يحذف كلَّ أصناف QA_DENSE — تنظيفٌ تامّ.
func (s *Server) qaClearDense(w http.ResponseWriter, r *http.Request) {
	tag, err := s.pg.Exec(r.Context(), `DELETE FROM menu_items WHERE name LIKE 'QA_DENSE_%'`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA dense fixture cleared (staging-only)", "deleted", tag.RowsAffected())
	httpx.JSON(w, http.StatusOK, map[string]any{"deleted": tag.RowsAffected()})
}
