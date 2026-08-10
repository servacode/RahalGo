package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/auth"
	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// طلبات انضمام المتاجر عبر رابط المندوب.

// **وما حُوِّل لا يُردّ** — والرسالةُ تقول السبب لا «غير موجود».
var errLeadConverted = httpx.NewError(http.StatusConflict,
	"lead_already_converted", "errors.lead_already_converted")

// نصوص إشعارات هذا القسم — مجمّعة كي لا تتناثر في الكود.
var m = struct{ leadNew, leadNewOps, leadApproved string }{
	leadNew:      "طلب انضمام جديد عبر رابطك",
	leadNewOps:   "طلب انضمام متجر جديد",
	leadApproved: "تمت الموافقة على عميلك",
}

// حدود طول الحقول — نقطة عامة بلا حساب، نمنع تخزين حمولات ضخمة لكل صف.
const (
	leadMaxShort = 120  // اسم المتجر/المالك/المنطقة/الهاتف
	leadMaxNote  = 1000 // الملاحظات
	joinMaxPerIP = 10   // طلبات لكل عنوان خلال النافذة
)

// clip يقصّ ويهذّب نصاً إلى حدٍّ أقصى.
func clip(v string, max int) string {
	v = strings.TrimSpace(v)
	if len(v) > max {
		return v[:max]
	}
	return v
}

// handlePublicJoin التقاط طلب انضمام متجر عبر رابط/باركود مندوب (عام، بلا حساب).
func (s *Server) handlePublicJoin(w http.ResponseWriter, r *http.Request) {
	// تحديد المعدل حسب العنوان — نقطة عامة قابلة للإغراق (نفس نمط طلب الرمز).
	// نعزل المضيف عن المنفذ العابر كي يكون المفتاح لكل عنوان لا لكل اتصال.
	ip := clientIP(r)
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	key := "join:req:" + ip
	if n, err := s.rdb.Incr(r.Context(), key).Result(); err == nil {
		if n == 1 {
			s.rdb.Expire(r.Context(), key, time.Hour)
		}
		if n > joinMaxPerIP {
			s.respondErr(w, httpx.NewError(http.StatusTooManyRequests, "rate_limited", "errors.rate_limited"))
			return
		}
	}

	req, err := decode[struct {
		Ref        string   `json:"ref"` // كود دعوة المندوب
		StoreName  string   `json:"store_name"`
		OwnerName  string   `json:"owner_name"`
		Phone      string   `json:"phone"`
		Area       string   `json:"area"`
		CategoryID string   `json:"category_id"` // تصنيف المتجر
		Password   string   `json:"password"`    // كلمة مرور صاحب المتجر
		Lat        *float64 `json:"lat"`         // موقع المتجر (اختياري)
		Lng        *float64 `json:"lng"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	req.StoreName = clip(req.StoreName, leadMaxShort)
	req.OwnerName = clip(req.OwnerName, leadMaxShort)
	req.Area = clip(req.Area, leadMaxShort)
	if req.StoreName == "" {
		s.respondErr(w, errValidation)
		return
	}
	// الرقم يجب أن يكون رقم موبايل صالح (سيصله رمز الدخول والإشعارات عبر واتساب).
	phone, ok := identity.NormalizePhone(req.Phone)
	if !ok {
		s.respondErr(w, identity.ErrInvalidPhone)
		return
	}
	// كلمة المرور إلزامية — يدخل بها صاحب المتجر بعد الموافقة.
	if len(req.Password) < s.minPasswordLen(r.Context()) {
		s.respondErr(w, httpx.NewError(http.StatusBadRequest, "weak_password", "errors.weak_password"))
		return
	}
	pwHash, err := auth.HashPassword(req.Password)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// معرّف التصنيف — اختياري لكنه إن وُجد يجب أن يكون UUID صالحاً.
	var categoryID *string
	if req.CategoryID != "" {
		if !isUUID(req.CategoryID) {
			s.respondErr(w, errValidation)
			return
		}
		categoryID = &req.CategoryID
	}
	// الإسناد: كود مندوب صالح وفعّال → يُنسب له. غير ذلك (لا كود/كود المنصة/كود
	// خاطئ) → تسجيل مباشر منسوب للمنصة (لا رفض) — الإنشاء الفعلي عند موافقة الإدارة.
	var repID *string
	if rep, err := s.identity.SalesRepByInviteCode(r.Context(), req.Ref); err == nil && rep.Status == "active" {
		repID = &rep.ID
	}
	// **لا نسبة لمتجرٍ على المنصة أصلاً**.
	//
	// المندوب يُكافأ على **جلب** متجر، ومتجرٌ يعمل عندنا لم يُجلَب. وبلا هذا
	// الفحص يستطيع من يعرف متاجر المنصة أن يدعو متجراً قائماً برقمٍ آخر
	// فيَنسبه لنفسه ويقبض عن مبيعاته. (وهو ما تمنعه DoorDash صراحةً في شروط
	// إحالتها: لا مكافأة لمتجرٍ له حساب سابق.)
	//
	// والطلب لا يُرفض — يمضي بلا نسبة. فالمتجر قد يكون فرعاً جديداً بحقّ،
	// وحرمانُه من التسجيل عقوبةٌ على المندوب لا عليه.
	if repID != nil {
		var exists bool
		if err := s.pg.QueryRow(r.Context(), `
			SELECT EXISTS(
				SELECT 1 FROM merchants m
				JOIN users u ON u.id = m.owner_user_id
				WHERE u.phone = $1)`, phone).Scan(&exists); err == nil && exists {
			repID = nil
		}
	}
	if _, err := s.pg.Exec(r.Context(), `
		INSERT INTO merchant_leads
			(store_name, owner_name, phone, area, category_id, lat, lng, owner_password_hash, sales_rep_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		req.StoreName, req.OwnerName, phone, req.Area,
		categoryID, req.Lat, req.Lng, pwHash, repID); err != nil {
		s.respondErr(w, err)
		return
	}
	// إشعار فوري: المندوب صاحب الكود والإدارة يعرفان بالطلب بلا تحديث صفحة.
	if repID != nil {
		s.notify.Notify(r.Context(), notifications.Input{
			UserID: *repID, Kind: notifications.KindLead,
			Title: m.leadNew, Body: req.StoreName,
			// **والرابطُ إلى صفحةٍ قائمة.**
			//
			// كان يشير إلى `/portal/leads` **ولا وجودَ لها في لوحة المندوب** —
			// فيُضغط الإشعارُ فيصل إلى لا شيء. **وإشعارٌ يفتح صفحةً غيرَ موجودة
			// أسوأُ من إشعارٍ بلا رابط**: يُقرأ عطباً في المنصة.
			//
			// والفرصُ تُعرض في «متاجري» مع المتاجر — **رحلةُ المتجر واحدةٌ من
			// فرصةٍ إلى متجرٍ يعمل**، وفصلُها بابين يجعل المندوبَ يتنقّل بينهما.
			//
			// وهي علّةُ `N-21` نفسُها في لوحةٍ أخرى.
			Entity: "lead", Href: "/portal/merchants",
		})
	}
	s.notify.NotifyOps(r.Context(), notifications.Input{
		Kind: notifications.KindLead, Title: m.leadNewOps,
		Body: req.StoreName, Entity: "lead", Href: "/dashboard/leads",
	})
	httpx.JSON(w, http.StatusCreated, map[string]any{"received": true})
}

// handlePublicInvite يعيد كود الدعوة الذي يُعرض في نموذج التسجيل (للقراءة فقط):
// كود المندوب إن كان صالحاً وفعّالاً، وإلا كود المنصة الافتراضي (تسجيل مباشر).
func (s *Server) handlePublicInvite(w http.ResponseWriter, r *http.Request) {
	// تحديد معدل حسب العنوان — نقطة عامة قد تُستغل لتعداد أكواد المندوبين.
	ip := clientIP(r)
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	key := "invite:req:" + ip
	if n, err := s.rdb.Incr(r.Context(), key).Result(); err == nil {
		if n == 1 {
			s.rdb.Expire(r.Context(), key, time.Hour)
		}
		if n > 60 {
			s.respondErr(w, httpx.NewError(http.StatusTooManyRequests, "rate_limited", "errors.rate_limited"))
			return
		}
	}

	ref := r.URL.Query().Get("ref")
	if ref != "" {
		if rep, err := s.identity.SalesRepByInviteCode(r.Context(), ref); err == nil && rep.Status == "active" {
			// **واسمُه معه.**
			//
			// كانت النقطة تعرف المندوب — تجلب صفَّه وتتحقق من نشاطه — ثم تردّ
			// «by: rep» بلا اسم. فيصل صاحبُ المتجر إلى صفحة تسجيلٍ لا يعرف من
			// دعاه إليها، ويُطلب منه أن يكتب اسمه وهاتفه وكلمة مروره لمجهول.
			//
			// **ورابطُ الإحالة كلُّه قائمٌ على أن يُعرف صاحبه**: المندوب يشاركه
			// عبر واتساب بعد لقاءٍ في السوق، والصفحة التي لا تذكر اسمه تنقض
			// ذلك اللقاء.
			//
			// والاسم ليس تسريباً: المندوب موظّفٌ يعمل علناً باسمه، ويقوله بفمه
			// لكل متجرٍ يزوره. والتعداد محروسٌ بحدّ المعدّل أعلاه.
			httpx.JSON(w, http.StatusOK, map[string]any{
				"code": ref, "by": "rep", "rep_name": rep.FullName,
			})
			return
		}
	}
	code := s.settings.GetString(r.Context(), "platform.invite_code")
	httpx.JSON(w, http.StatusOK, map[string]any{"code": code, "by": "platform"})
}

type lead struct {
	ID           string    `json:"id"`
	StoreName    string    `json:"store_name"`
	OwnerName    string    `json:"owner_name"`
	Phone        string    `json:"phone"`
	Area         string    `json:"area"`
	CategoryName *string   `json:"category_name"`
	CategoryIcon *string   `json:"category_icon"`
	Lat          *float64  `json:"lat"`
	Lng          *float64  `json:"lng"`
	RepName      *string   `json:"rep_name"`
	RepCode      *string   `json:"rep_code"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	// Note ما كتبه المندوبُ حين أرسل — **صوتُه هو.**
	Note string `json:"note"`
	// DecisionNote سببُ ردّ الإدارة — **صوتٌ آخرُ في حقلٍ آخر.**
	//
	// **وخلطُهما يمحو ما كتبه صاحبُ الفرصة**، ويجعل حقلاً واحداً يحمل صوتين.
	DecisionNote string `json:"decision_note"`
}

func scanLeads(rows interface {
	Next() bool
	Scan(...any) error
}) ([]lead, error) {
	out := []lead{}
	for rows.Next() {
		var l lead
		if err := rows.Scan(&l.ID, &l.StoreName, &l.OwnerName, &l.Phone, &l.Area,
			&l.CategoryName, &l.CategoryIcon, &l.Lat, &l.Lng,
			&l.RepName, &l.RepCode, &l.Status, &l.CreatedAt,
			&l.Note, &l.DecisionNote); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, nil
}

const leadSelect = `
	SELECT l.id, l.store_name, l.owner_name, l.phone, l.area,
	       c.name, c.icon, l.lat, l.lng,
	       NULLIF(COALESCE(u.full_name, u.phone::text), ''), u.invite_code, l.status, l.created_at,
	       l.note, l.decision_note
	FROM merchant_leads l
	LEFT JOIN users u ON u.id = l.sales_rep_user_id
	LEFT JOIN categories c ON c.id = l.category_id`

// handleAdminLeads كل طلبات الانضمام (ترشيح بالحالة اختياري).
func (s *Server) handleAdminLeads(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	// **وصفحةٌ محدودةٌ بعدٍّ** — (قرارُ المالك ٢٠٢٦-٠٨-١٠).
	//
	// **ومئتان بلا كلمةٍ تُقرأ «هذا كلُّ من طلب الانضمام»** — فيُظنّ أنّ
	// الطلباتِ نضبت وهي في الصفحة الثانية.
	pg := pagingOf(r, 20)
	var count int
	if err := s.pg.QueryRow(r.Context(),
		`SELECT count(*) FROM merchant_leads l WHERE ($1 = '' OR l.status = $1)`,
		status).Scan(&count); err != nil {
		s.respondErr(w, err)
		return
	}
	rows, err := s.pg.Query(r.Context(),
		leadSelect+` WHERE ($1 = '' OR l.status = $1)
		ORDER BY l.created_at DESC LIMIT $2 OFFSET $3`, status, pg.PerPage, pg.Offset)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out, err := scanLeads(rows)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, paged("leads", out, count, pg))
}

// handleRepCreateLead تسجيل عميل جديد من بوابة المندوب مباشرة.
//
// المندوب واقف في المحل: أن يملأ النموذج بنفسه في نصف دقيقة أنجع من إرسال رابط
// يُنسى. الطلب يُنسب له تلقائياً بهويّته (لا كود دعوة ولا انتحال)، ويبقى
// **معلّقاً** حتى موافقة الإدارة — لا يُنشئ متجراً ولا حساباً (قرار حوكمة الضمّ).
//
// كلمة المرور يضعها المندوب ويسلّمها لصاحب المتجر — لكنها **مؤقتة**: يُجبَر
// المالك على تبديلها عند أول دخول، فلا تبقى كلمة مرور يعرفها غير صاحبها.
// repLeadsPerHour سقف تسجيل العملاء للمندوب الواحد في الساعة.
//
// السقف سخيّ عمداً: مندوبٌ نشط قد يسجّل عدّة متاجر في جولة ميدانية واحدة، فحدٌّ
// ضيّق يعاقب المجتهد. لكنه موجود لأن النقطة تُنشئ **حسابات وكلمات مرور** —
// وأي نقطة تُنشئ حسابات بلا سقف هي أداة إغراق جاهزة (GROUND-RULES §5).
const repLeadsPerHour = 30

func (s *Server) handleRepCreateLead(w http.ResponseWriter, r *http.Request) {
	// التحديد **بالمندوب لا بعنوانه**: المناديب يعملون من شبكات مشتركة (مقهى،
	// مكتب) فحدُّ العنوان يوقف زملاءه معه، وهو مصادَق أصلاً فهويّته معروفة.
	key := "rep:lead:" + userIDFrom(r)
	if n, err := s.rdb.Incr(r.Context(), key).Result(); err == nil {
		if n == 1 {
			s.rdb.Expire(r.Context(), key, time.Hour)
		}
		if n > repLeadsPerHour {
			s.respondErr(w, httpx.NewError(http.StatusTooManyRequests, "rate_limited", "errors.rate_limited"))
			return
		}
	}
	req, err := decode[struct {
		StoreName  string   `json:"store_name"`
		OwnerName  string   `json:"owner_name"`
		Phone      string   `json:"phone"`
		Area       string   `json:"area"`
		CategoryID string   `json:"category_id"`
		Password   string   `json:"password"`
		Lat        *float64 `json:"lat"`
		Lng        *float64 `json:"lng"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if len(req.Password) < s.minPasswordLen(r.Context()) {
		s.respondErr(w, httpx.NewError(http.StatusBadRequest, "weak_password", "errors.weak_password"))
		return
	}
	pwHash, err := auth.HashPassword(req.Password)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	req.StoreName = clip(req.StoreName, leadMaxShort)
	req.OwnerName = clip(req.OwnerName, leadMaxShort)
	req.Area = clip(req.Area, leadMaxShort)
	if req.StoreName == "" || req.CategoryID == "" {
		s.respondErr(w, errValidation)
		return
	}
	if !isUUID(req.CategoryID) {
		s.respondErr(w, errValidation)
		return
	}
	phone, ok := identity.NormalizePhone(req.Phone)
	if !ok {
		s.respondErr(w, identity.ErrInvalidPhone)
		return
	}

	repID := userIDFrom(r)
	var leadID string
	if err := s.pg.QueryRow(r.Context(), `
		INSERT INTO merchant_leads
			(store_name, owner_name, phone, area, category_id, lat, lng, owner_password_hash, sales_rep_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`,
		req.StoreName, req.OwnerName, phone, req.Area,
		req.CategoryID, req.Lat, req.Lng, pwHash, repID).Scan(&leadID); err != nil {
		s.respondErr(w, err)
		return
	}

	// مكتب المنصة يعرف فوراً أن عميلاً ينتظر الموافقة
	s.notify.NotifyOps(r.Context(), notifications.Input{
		Kind: notifications.KindLead, Title: m.leadNewOps,
		Body: req.StoreName, Entity: "lead", EntityID: leadID, Href: "/dashboard/leads",
	})
	s.touch("lead", "ops", "sales:"+repID)
	httpx.JSON(w, http.StatusCreated, map[string]any{"id": leadID})
}

// handleRepLeads طلبات انضمام المندوب نفسه (بوابة المندوب).
func (s *Server) handleRepLeads(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(),
		leadSelect+` WHERE l.sales_rep_user_id = $1 ORDER BY l.created_at DESC LIMIT 200`,
		userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out, err := scanLeads(rows)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}

// handleAdminLeadStatus تحديث حالة طلب. "converted" ينشئ المتجر فعلياً (موافقة
// الإدارة هي لحظة الإنشاء — لا متجر قبلها). "rejected"/"new" مجرد وسم.
func (s *Server) handleAdminLeadStatus(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Status string `json:"status"`
		Note   string `json:"note"`
	}](r)
	if err != nil || (req.Status != "converted" && req.Status != "rejected" && req.Status != "new") {
		s.respondErr(w, errValidation)
		return
	}
	// **والردُّ يلزمه كلمة — كما كلُّ ردٍّ في هذه المنصة.**
	//
	// المندوبُ الذي رُدّت فرصتُه بلا سببٍ **يلاحق عميلاً ميتاً أو يعيد إرسالَ
	// الفرصة نفسِها** — فتُردّ ثانيةً، ويدور هو والمكتبُ في حلقة.
	//
	// **والقاعدةُ مفروضةٌ في كلّ موضعٍ سواه**: الرفضُ والإلغاءُ والفشلُ
	// والاسترجاعُ وحسمُ النزاع وردُّ صنفٍ في المراجعة. **والفرصةُ وحدَها كانت
	// تُردّ صامتة.**
	note := strings.TrimSpace(req.Note)
	if req.Status == "rejected" && note == "" {
		s.respondErr(w, errReasonRequired)
		return
	}
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if req.Status == "converted" {
		if err := s.convertLead(r.Context(), userIDFrom(r), id, clientIP(r)); err != nil {
			s.respondErr(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
		return
	}
	// ══════════════════════════════════════════════════════════════════
	// **وما حُوِّل لا يُردّ — المتجرُ قائمٌ يبيع.**
	// ══════════════════════════════════════════════════════════════════
	//
	// (شهده المالك ٢٠٢٦-٠٨-٠٨: «لا يجوز أن يبقى زرُّ الرفض بعد قبول».)
	//
	// **كان الشرطُ `WHERE id = $1` وحدَه** — بلا ذكرٍ لحالتها السابقة. فطلبٌ
	// حُوِّل، وصاحبُ المتجر يدخل بوّابتَه ويستقبل طلبات، **يُوسَم «مرفوضاً»
	// بضغطة.**
	//
	// **والأثرُ يخرج من الشاشة**: يصل المندوبَ إشعارٌ «رُدّ طلبُ الانضمام»
	// بسببٍ كتبه أحد — **فيقرأ أنّ فرصتَه ضاعت وهي لم تضِع**: المتجرُ يعمل
	// والنسبةُ له. **ويُخصَم من عدّ «سُجّل» في لوحته** فيظنّ نفسَه أقلَّ
	// إنجازاً ممّا هو.
	//
	// **وإخفاءُ الزرّ لا يكفي**: الشاشةُ تمنع اليدَ والخادمُ يمنع الفعل —
	// ومن فتح لوحتين وضغط في القديمة قبل أن تُحدَّث مرّ.
	//
	// **والاستئنافُ يبقى**: `new` مسموحةٌ من `rejected` — من رُدَّ خطأً
	// يُعاد. **والممنوعُ الخروجُ من `converted` وحدَه.**
	var repID *string
	var storeName string
	err = s.pg.QueryRow(r.Context(),
		`UPDATE merchant_leads SET status = $2, decision_note = $3, updated_at = now()
		 WHERE id = $1 AND status <> 'converted'
		 RETURNING sales_rep_user_id, store_name`,
		id, req.Status, note).Scan(&repID, &storeName)
	if errors.Is(err, pgx.ErrNoRows) {
		// **وصفٌّ لم يتبدّل إمّا غائبٌ أو محوَّل** — ويُفرَّق بينهما، فرسالةُ
		// «غير موجود» على متجرٍ يعمل تُقرأ عطباً.
		var cur string
		if e := s.pg.QueryRow(r.Context(),
			`SELECT status FROM merchant_leads WHERE id = $1`, id).Scan(&cur); e != nil {
			s.respondErr(w, httpx.ErrNotFound)
			return
		}
		s.respondErr(w, errLeadConverted)
		return
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// الرفض يخصّ المندوب بقدر ما تخصّه الموافقة — وإلا بقي يلاحق عميلاً ميتاً.
	// **والسببُ في متن الإشعار** — لا في صفحةٍ يُطلب منه أن يفتحها.
	if req.Status == "rejected" && repID != nil {
		s.notify.Notify(r.Context(), notifications.Input{
			UserID: *repID, Kind: notifications.KindLead,
			Title: notifTitles.leadRejected, Body: storeName + " — " + note,
			Entity: "lead", EntityID: id, Href: "/portal/leads",
		})
	}
	s.touch("lead", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
}

// convertLead يحوّل طلب انضمام إلى متجر فعلي: ينشئ/يربط حساب صاحب المتجر (بكلمة
// مروره المحفوظة إن كان جديداً) والمتجر بتصنيفه وموقعه منسوباً للمندوب.
func (s *Server) convertLead(ctx context.Context, actorID, leadID, ip string) error {
	var (
		storeName, ownerName, phone, area string
		categoryID                        *string
		lat, lng                          *float64
		pwHash                            string
		repCode                           *string
		merchantID                        *string
	)
	err := s.pg.QueryRow(ctx, `
		SELECT l.store_name, l.owner_name, l.phone, l.area, l.category_id, l.lat, l.lng,
		       l.owner_password_hash, u.invite_code, l.merchant_id
		FROM merchant_leads l
		LEFT JOIN users u ON u.id = l.sales_rep_user_id
		WHERE l.id = $1`, leadID).
		Scan(&storeName, &ownerName, &phone, &area, &categoryID, &lat, &lng, &pwHash, &repCode, &merchantID)
	if err != nil {
		return httpx.ErrNotFound
	}
	if merchantID != nil {
		return nil // محوّل مسبقاً — لا تكرار
	}
	if categoryID == nil {
		return errValidation // لا متجر بلا تصنيف
	}
	in := catalog.MerchantInput{
		Name:        &storeName,
		CategoryID:  categoryID,
		Phone:       &phone,
		AddressText: &area,
		OwnerPhone:  &phone,
		Lat:         lat,
		Lng:         lng,
	}
	if repCode != nil {
		in.SalesRepCode = repCode
	}
	// هل يملك الرقم حساباً مسبقاً؟ حرج أمنياً: كلمة المرور من نموذج التسجيل يجب ألّا
	// تُطبَّق على حساب قائم (وإلا يمكن لمهاجم "التسجيل" برقم ضحية بلا كلمة مرور ثم
	// يستولي على حسابها عند الموافقة). نطبّق كلمة المرور على الحسابات الجديدة فقط.
	var ownerExisted bool
	_ = s.pg.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE phone = $1)`, phone).Scan(&ownerExisted)

	mrch, err := s.catalog.CreateMerchant(ctx, actorID, in, ip)
	if err != nil {
		return err
	}
	// الاسم يُملأ إن كان فارغاً (غير حسّاس). كلمة المرور للحساب الجديد حصراً.
	_, _ = s.pg.Exec(ctx, `
		UPDATE users SET
			full_name  = CASE WHEN full_name = '' THEN $2 ELSE full_name END,
			updated_at = now()
		WHERE id = (SELECT owner_user_id FROM merchants WHERE id = $1)`,
		mrch.ID, ownerName)
	// كلمة المرور وضعها طرف ثالث (المندوب أو نموذج التسجيل) — مؤقتة يُجبَر
	// صاحب المتجر على تبديلها عند أول دخول قبل الوصول إلى بوابته.
	if !ownerExisted && pwHash != "" {
		_, _ = s.pg.Exec(ctx, `
			UPDATE users SET password_hash = $2, must_change_password = true, updated_at = now()
			WHERE id = (SELECT owner_user_id FROM merchants WHERE id = $1)
			  AND COALESCE(password_hash,'') = ''`,
			mrch.ID, pwHash)
	}
	_, err = s.pg.Exec(ctx, `
		UPDATE merchant_leads SET status = 'converted', merchant_id = $2, updated_at = now()
		WHERE id = $1`, leadID, mrch.ID)
	// المندوب يعرف فوراً أن عميله اعتُمد (مصدر عمولته)
	var repID *string
	_ = s.pg.QueryRow(ctx, `SELECT sales_rep_user_id FROM merchant_leads WHERE id = $1`, leadID).Scan(&repID)
	if repID != nil {
		s.notify.Notify(ctx, notifications.Input{
			UserID: *repID, Kind: notifications.KindLead,
			Title: m.leadApproved, Body: storeName,
			Entity: "merchant", EntityID: mrch.ID, Href: "/portal/merchants",
		})
	}
	return err
}
