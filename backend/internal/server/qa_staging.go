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
	"github.com/servacode/rahalgo/backend/internal/support"
)

// qaStagingPhone **زبونُ QA الثابت** — رقمٌ محجوزٌ للاختبار على التجهيز.
//
// **ومختارٌ بعيداً عن أرقام البذور** (الأدمن `+963999…`، الطاقم `+963955…`،
// وأرقامُ اختبارِ الأطوار `+96390000000x`) — **فلا يُصدَر بالخطأ سِمةُ حسابٍ
// مُمتاز.** وحارسُ الدور أدناه يمنع ذلك على كلّ حال.
const qaStagingPhone = "+963900555001"

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

	phone, ok := identity.NormalizePhone(qaStagingPhone)
	if !ok {
		s.respondErr(w, errValidation)
		return
	}

	// **يُبحَث عنه أولاً** — **فلا يُعاد منحُ الدور على القائم** (يُدرج
	// `granted_by` فارغاً فيسقط)، ولا يُنشأ إلّا مرّةً.
	var uid string
	err := s.pg.QueryRow(r.Context(), `SELECT id::text FROM users WHERE phone = $1`, phone).Scan(&uid)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		user, cerr := s.identity.EnsureUserWithRole(r.Context(), "", qaStagingPhone, "customer", "زبون الاختبار QA", "", clientIP(r))
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
	for _, p := range []string{qaStagingPhone, "+963900000001"} { // الزبونُ + الرقمُ المُصطدَمُ في البناء المؤقّت
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
	"ticket_reply": true, // ردُّ أدمن على تذكرة زبون QA نفسِه (E-013)
	"warning":      true, // إنذارُ حسابٍ على زبون QA (D-corroboration)
	"offer":        true, // عرضُ خصمٍ حيٌّ قصيرُ الأجل على صنفٍ (A/ENG-005)
	"offer_off":    true, // إطفاءُ عرضٍ بذرناه (تنظيف)
}

// handleQAStagingSeed يبذر عتادَ اختبارٍ لزبون QA — على التجهيز وحدَه.
func (s *Server) handleQAStagingSeed(w http.ResponseWriter, r *http.Request) {
	if !s.qaStagingEnabled() {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, err := decode[struct {
		Kind    string `json:"kind"`
		OfferID string `json:"offer_id"`
	}](r)
	if err != nil {
		s.respondErr(w, errValidation)
		return
	}
	if !qaSeedAllowlist[req.Kind] {
		s.respondErr(w, httpx.NewError(http.StatusForbidden, "qa_seed_kind_not_allowed", "errors.forbidden"))
		return
	}

	// **العرضُ يُطفأ بمعرّفه ولا يلزمه زبونُ QA** — تنظيفٌ صريح.
	if req.Kind == "offer_off" {
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
	case "warning":
		s.qaSeedWarning(w, r, uid)
	case "offer":
		s.qaSeedOffer(w, r, uid)
	}
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
