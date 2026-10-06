package server

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/auth"
	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// طلبات انضمام المتاجر — يسجّلها المندوب من تطبيقه (`POST /rep/leads`).

// **وما حُوِّل لا يُردّ** — والرسالةُ تقول السبب لا «غير موجود».
var errLeadConverted = httpx.NewError(http.StatusConflict,
	"lead_already_converted", "errors.lead_already_converted")

// نصوص إشعارات هذا القسم — مجمّعة كي لا تتناثر في الكود.
var m = struct{ leadNewOps, leadApproved, leadNeedsInfo string }{
	leadNewOps:    "طلب انضمام متجر جديد",
	leadApproved:  "تمت الموافقة على عميلك",
	leadNeedsInfo: "طلب انضمام بحاجة معلومات",
}

// حدود طول الحقول — نقطة عامة بلا حساب، نمنع تخزين حمولات ضخمة لكل صف.
const (
	leadMaxShort = 120  // اسم المتجر/المالك/المنطقة/الهاتف
	leadMaxNote  = 1000 // الملاحظات
)

// clip يقصّ ويهذّب نصاً إلى حدٍّ أقصى.
func clip(v string, max int) string {
	v = strings.TrimSpace(v)
	if len(v) > max {
		return v[:max]
	}
	return v
}

// incr **يعدّ نداءً في ريدس** — **ويردّ صفراً حين لا ريدس** فلا يسقط
// المسارُ في فحصٍ لا يشغّله. (والحدُّ أفضليّةٌ لا شرط.)
func (s *Server) incr(ctx context.Context, key string) (int64, error) {
	if s.rdb == nil {
		return 0, nil
	}
	return s.rdb.Incr(ctx, key).Result()
}

type lead struct {
	ID        string `json:"id"`
	StoreName string `json:"store_name"`
	OwnerName string `json:"owner_name"`
	Phone     string `json:"phone"`
	Area      string `json:"area"`
	// District **«منطقة، محافظة»** — يقرؤها من يوافق على الطلب.
	//
	// **وكان «وسط المدينة» وحدَه** — فمن راجع الطلبَ لا يعرف أفي
	// الرقّة هو أم في حلب.
	District     string    `json:"district"`
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
		if err := rows.Scan(&l.ID, &l.StoreName, &l.OwnerName, &l.Phone, &l.Area, &l.District,
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
	       COALESCE(d.name || '، ' || g.name, ''),
	       c.name, c.icon, l.lat, l.lng,
	       NULLIF(COALESCE(u.full_name, u.phone::text), ''), u.invite_code, l.status, l.created_at,
	       l.note, l.decision_note
	FROM merchant_leads l
	LEFT JOIN users u ON u.id = l.sales_rep_user_id
	LEFT JOIN categories c ON c.id = l.category_id
	LEFT JOIN districts d ON d.id = l.district_id
	LEFT JOIN governorates g ON g.id = d.governorate_id`

// adminLead **طلبٌ كما يراه المكتب** — الطلبُ نفسُه وتحذيراتُه.
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٤.) **والتحذيراتُ للمكتب وحدَه** — لا تُرسَل إلى
// المندوب: من عرف أنّ رقماً له حسابٌ عندنا عرف شيئاً عن صاحبه.
type adminLead struct {
	lead
	GovernorateID *string `json:"governorate_id"`
	// DuplicatePhone **أسماءُ متاجرَ أو طلباتٍ مفتوحةٍ بالرقم نفسِه.**
	DuplicatePhone []string `json:"duplicate_phone"`
	// NearbySameName **متاجرُ قائمةٌ بالاسم نفسِه قريبةٌ منه** — ضمن
	// ألفِ مترٍ من نقطته (`adminLeadSelect`)، أو في منطقته إن لم تكن له نقطة.
	NearbySameName []string `json:"nearby_same_name"`
	// ExistingAccount **للرقم حسابٌ قائم** — فلا كلمةَ سرٍّ تُولَّد له:
	// تصله «صار عندك متجر». يُقال قبل الموافقة.
	ExistingAccount bool      `json:"existing_account"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// leadStatuses **الحالاتُ الأربع** — و`needs_info` تعود للمندوب بملاحظة.
var leadStatuses = map[string]bool{"new": true, "needs_info": true, "converted": true, "rejected": true}

// handleAdminLeads طلباتُ الانضمام — **للمتاجر التي يضيفها المندوبون وحدَها.**
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٤: «وصفحةُ طلبات الانضمام للمتاجر التي يضيفها
// المندوبون وحدَها».) **وما أنشأته الإدارةُ بيدها لا يمرّ من هنا أصلاً**،
// **وصفٌّ بلا مندوبٍ بقيّةُ بابِ الانضمام العامّ الذي رُفع** (`JOIN-0`).
//
// **والترشيحُ كلُّه في الخادم** — بحثٌ ومندوبٌ ومحافظةٌ وتصنيفٌ وتاريخ —
// فالعددُ والصفحاتُ صادقةٌ لما يُعرض.
func (s *Server) handleAdminLeads(w http.ResponseWriter, r *http.Request) {
	qp := r.URL.Query()
	status := qp.Get("status")
	if status != "" && !leadStatuses[status] {
		s.respondErr(w, errValidation)
		return
	}
	// **وصفحةٌ محدودةٌ بعدٍّ** — (قرارُ المالك ٢٠٢٦-٠٨-١٠).
	//
	// **ومئتان بلا كلمةٍ تُقرأ «هذا كلُّ من طلب الانضمام»** — فيُظنّ أنّ
	// الطلباتِ نضبت وهي في الصفحة الثانية.
	pg := pagingOf(r, 20)
	// **ومندوبُه مُرشِّحٌ** — (قرارُ المالك ٢٠٢٦-٠٨-١٦): **عملاؤه المحتملون
	// في ملفّه.**
	repID := qp.Get("rep_id")
	govID := qp.Get("governorate_id")
	catID := qp.Get("category_id")
	for _, id := range []string{repID, govID, catID} {
		if id != "" && !isUUID(id) {
			s.respondErr(w, errValidation)
			return
		}
	}
	// **والتاريخُ يومٌ بتوقيت دمشق** — من «من» إلى «إلى» شاملَين.
	var from, to *time.Time
	for _, p := range []struct {
		raw string
		dst **time.Time
	}{{qp.Get("from"), &from}, {qp.Get("to"), &to}} {
		if p.raw == "" {
			continue
		}
		d, err := time.Parse("2006-01-02", p.raw)
		if err != nil {
			s.respondErr(w, errValidation)
			return
		}
		*p.dst = &d
	}
	q := clip(qp.Get("q"), leadMaxShort)
	qPhone := ""
	if ph, ok := identity.NormalizePhone(q); ok {
		qPhone = ph
	}
	const leadWhere = ` WHERE l.sales_rep_user_id IS NOT NULL
		AND ($1 = '' OR l.status = $1)
		AND ($2 = '' OR l.sales_rep_user_id::text = $2)
		AND ($3 = '' OR d.governorate_id::text = $3)
		AND ($4 = '' OR l.category_id::text = $4)
		AND ($5::date IS NULL OR (l.created_at AT TIME ZONE 'Asia/Damascus')::date >= $5::date)
		AND ($6::date IS NULL OR (l.created_at AT TIME ZONE 'Asia/Damascus')::date <= $6::date)
		AND ($7 = '' OR strpos(lower(l.store_name), lower($7)) > 0
		             OR strpos(lower(l.owner_name), lower($7)) > 0
		             OR strpos(l.phone, $7) > 0
		             OR ($8 <> '' AND l.phone = $8))`
	args := []any{status, repID, govID, catID, from, to, q, qPhone}
	var count int
	if err := s.pg.QueryRow(r.Context(),
		`SELECT count(*) FROM merchant_leads l
		 LEFT JOIN districts d ON d.id = l.district_id`+leadWhere,
		args...).Scan(&count); err != nil {
		s.respondErr(w, err)
		return
	}
	rows, err := s.pg.Query(r.Context(),
		adminLeadSelect+leadWhere+`
		ORDER BY l.created_at DESC LIMIT $9 OFFSET $10`,
		append(args, pg.PerPage, pg.Offset)...)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []adminLead{}
	for rows.Next() {
		var a adminLead
		l := &a.lead
		if err := rows.Scan(&l.ID, &l.StoreName, &l.OwnerName, &l.Phone, &l.Area, &l.District,
			&l.CategoryName, &l.CategoryIcon, &l.Lat, &l.Lng,
			&l.RepName, &l.RepCode, &l.Status, &l.CreatedAt,
			&l.Note, &l.DecisionNote,
			&a.GovernorateID, &a.DuplicatePhone, &a.NearbySameName, &a.ExistingAccount,
			&a.UpdatedAt); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	res := paged("leads", out, count, pg)
	// **ومندوبو المرشِّح من أصحاب الطلبات أنفسِهم** — لا قائمةُ كلّ مندوب.
	reps := []map[string]string{}
	if rr, err := s.pg.Query(r.Context(), `
		SELECT DISTINCT u.id::text, COALESCE(NULLIF(u.full_name, ''), u.phone::text)
		  FROM merchant_leads l JOIN users u ON u.id = l.sales_rep_user_id
		 ORDER BY 2`); err == nil {
		for rr.Next() {
			var id, name string
			if rr.Scan(&id, &name) == nil {
				reps = append(reps, map[string]string{"id": id, "name": name})
			}
		}
		rr.Close()
	}
	res["reps"] = reps
	httpx.JSON(w, http.StatusOK, res)
}

// adminLeadSelect **الطلبُ وتحذيراتُه** — `leadSelect` نفسُه وأعمدةٌ بعده.
const adminLeadSelect = `
	SELECT l.id, l.store_name, l.owner_name, l.phone, l.area,
	       COALESCE(d.name || '، ' || g.name, ''),
	       c.name, c.icon, l.lat, l.lng,
	       NULLIF(COALESCE(u.full_name, u.phone::text), ''), u.invite_code, l.status, l.created_at,
	       l.note, l.decision_note,
	       g.id::text,
	       COALESCE((SELECT array_agg(x.name) FROM (
	           SELECT m2.name FROM merchants m2
	             LEFT JOIN users o ON o.id = m2.owner_user_id
	            WHERE m2.id IS DISTINCT FROM l.merchant_id
	              AND (m2.phone = l.phone OR o.phone::text = l.phone)
	           UNION
	           SELECT l2.store_name FROM merchant_leads l2
	            WHERE l2.phone = l.phone AND l2.id <> l.id
	              AND l2.status IN ('new', 'needs_info')
	           LIMIT 5) x), '{}'),
	       COALESCE((SELECT array_agg(n.name) FROM (
	           SELECT m3.name FROM merchants m3
	            WHERE m3.id IS DISTINCT FROM l.merchant_id
	              AND lower(btrim(m3.name)) = lower(btrim(l.store_name))
	              AND ((l.lat IS NOT NULL AND l.lng IS NOT NULL AND m3.location IS NOT NULL
	                    AND ST_DWithin(m3.location,
	                        ST_SetSRID(ST_MakePoint(l.lng, l.lat), 4326)::geography, 1000))
	                   OR (l.district_id IS NOT NULL AND m3.district_id = l.district_id))
	           LIMIT 5) n), '{}'),
	       EXISTS (SELECT 1 FROM users ux WHERE ux.phone::text = l.phone),
	       l.updated_at
	FROM merchant_leads l
	LEFT JOIN users u ON u.id = l.sales_rep_user_id
	LEFT JOIN categories c ON c.id = l.category_id
	LEFT JOIN districts d ON d.id = l.district_id
	LEFT JOIN governorates g ON g.id = d.governorate_id`

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
	// **وضمُّ المتاجر** — وإغلاقُه يُبقي المرشَّحين القائمين كما هم.
	//
	// **وقبل قراءةِ الجسم** — فلا يُستهلك مفتاحُ تفرّدٍ لبابٍ مغلق.
	if !s.requireLaunch(w, r, launchRepAcquisition) {
		return
	}
	// التحديد **بالمندوب لا بعنوانه**: المناديب يعملون من شبكات مشتركة (مقهى،
	// مكتب) فحدُّ العنوان يوقف زملاءه معه، وهو مصادَق أصلاً فهويّته معروفة.
	// **ولا يسجّل متجراً من لم يوثّق رقمَه** — (قرارُ المالك ٢٠٢٦-٠٨-١٣).
	//
	// **والمفتاحُ العامُّ يعلوه**: مطفأً لا يُسأل أحد.
	if s.settings.RequireWhatsApp(r.Context(), "sales.require_whatsapp") {
		var verified bool
		if err := s.pg.QueryRow(r.Context(),
			`SELECT whatsapp_verified_at IS NOT NULL FROM users WHERE id = $1`,
			userIDFrom(r)).Scan(&verified); err != nil {
			s.respondErr(w, err)
			return
		}
		if !verified {
			s.respondErr(w, errWhatsAppRequired)
			return
		}
	}

	// **وحدُّ المعدّل بالمساعد الآمن لا بنداءٍ مباشر.**
	//
	// **كان `s.rdb.Incr` عارياً** — **وينهار بمؤشّرٍ فارغٍ حين لا ريدس**
	// (`nil pointer dereference`). **وهو الحارسُ نفسُه المكتوبُ في
	// `incr` أعلاه** ويستعمله بابُ الويب منذ زمن: **مساعدٌ يُكتب ثمّ
	// يُنسى بابٌ لا يناديه.**
	//
	// **والحدُّ أفضليّةٌ لا شرط** — أخطاؤه مُبتلَعةٌ هنا كما هناك.
	key := "rep:lead:" + userIDFrom(r)
	if n, err := s.incr(r.Context(), key); err == nil {
		if n == 1 && s.rdb != nil {
			s.rdb.Expire(r.Context(), key, time.Hour)
		}
		if n > repLeadsPerHour {
			s.respondErr(w, httpx.NewError(http.StatusTooManyRequests, "rate_limited", "errors.rate_limited"))
			return
		}
	}
	req, err := decode[struct {
		StoreName string `json:"store_name"`
		OwnerName string `json:"owner_name"`
		Phone     string `json:"phone"`
		// Area **عنوانٌ تفصيليٌّ اختياريّ** — «مقابل الجامع».
		//
		// **والمنطقةُ تقول أين، وهذا يقول كيف تصل.**
		Area string `json:"area"`
		// DistrictID **المنطقةُ الإداريّةُ تُختار من قائمة.**
		//
		// (قرارُ المالك ٢٠٢٦-٠٨-٣٠.) **وكانت نصّاً حرّاً** — ثلاثةُ
		// نصوصٍ لموضعٍ واحدٍ لا تُصنَّف ولا تُصفّى.
		DistrictID string   `json:"district_id"`
		CategoryID string   `json:"category_id"`
		Password   string   `json:"password"`
		Lat        *float64 `json:"lat"`
		Lng        *float64 `json:"lng"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **ولا كلمةَ من المندوب** (قرارُ المالك ٢٠٢٦-١٠-٠٦: «السيرفر سيقوم بتوليدها… نلغي بالفورم كلمة
	// السر») — عند التحويل تُولَّد كلمةٌ مؤقّتةٌ وتُرسَل لصاحب المتجر. **فالفارغُ يأخذ كلمةً عشوائيّةً لا
	// يعرفها أحد**، والنسخُ القديمةُ التي ما زالت ترسلها تُفحَص كما كانت.
	if req.Password == "" {
		gen, gerr := identity.GenerateTempPassword(24)
		if gerr != nil {
			s.respondErr(w, gerr)
			return
		}
		req.Password = gen
	} else if len(req.Password) < s.minPasswordLen(r.Context()) {
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

	// **والمنطقةُ إلزاميّةٌ عند المندوب** — هو في الميدان يراها بعينه،
	// **ومن لم يعرف منطقةَ متجرٍ يقف أمامه لا يعرف شيئاً عنه.**
	district, err := s.validDistrict(r, req.DistrictID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if district == nil {
		s.respondErr(w, errDivNeedsGovernorate)
		return
	}

	repID := userIDFrom(r)
	storeName := req.StoreName

	// ══════════════════════════════════════════════════════════════
	// **وإنشاءُ المرشَّح يمرّ بمنعِ التكرار الدائم**
	// ══════════════════════════════════════════════════════════════
	//
	// المسارُ ملفوفٌ بـ`s.idempotent` (server.go) الذي يحوز المطالبةَ
	// ويضعها في السياق؛ و`WithIdempotentTx` يقفل صفَّ المفتاح ويُدرج
	// المرشَّحَ ويكتب الجوابَ في معاملةٍ واحدة. **فالنقرةُ المزدوجةُ
	// وإعادةُ المهلة والمتزامنُ بالمفتاح نفسِه ⇒ مرشَّحٌ واحدٌ بعينه** —
	// إعادةٌ لاحقةٌ بالمفتاح نفسِه تُعيد المُثبَّتَ الأوّلَ لا صفّاً ثانياً.
	s.WithIdempotentTx(w, r, func(ctx context.Context, q dbtx.Querier) (IdempotentBody, error) {
		var leadID string
		if err := q.QueryRow(ctx, `
			INSERT INTO merchant_leads
				(store_name, owner_name, phone, area, district_id, category_id, lat, lng, owner_password_hash, sales_rep_user_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			RETURNING id`,
			storeName, req.OwnerName, phone, req.Area, district,
			req.CategoryID, req.Lat, req.Lng, pwHash, repID).Scan(&leadID); err != nil {
			// ══════════════════════════════════════════════════════════════
			// **وتصنيفٌ حُذف من اللوحة يُقال لا يُردّ خاماً**
			// ══════════════════════════════════════════════════════════════
			//
			// **كان يُردّ ٥٠٠ «خطأ غير متوقّع»** — والإدراجُ يفشل بانتهاك
			// مفتاحٍ أجنبيّ (`23503`) أو بمعرّفٍ لا يُقرأ (`22P02`)،
			// **فيُمرَّر كما هو.**
			//
			// **و`CreateMerchant` تترجمه منذ زمن** — وهذا الباب لا:
			// **معالجةٌ تُكتب في موضعٍ وتُنسى في نظيره.**
			//
			// **وهي حالٌ تقع فعلاً**: يفتح المندوبُ النموذجَ في السوق،
			// **وتُحذف فئةٌ من اللوحة قبل أن يضغط «أرسل»** — فيقرأ «خطأ غير
			// متوقّع» ولا يعرف أنّ عليه اختيارَ تصنيفٍ آخر.
			//
			// (كشفه اختبارُ الميدان ٢٠٢٦-٠٨-٣٠.)
			if isFKViolation(err) {
				return IdempotentBody{}, catalog.ErrCategoryInvalid
			}
			return IdempotentBody{}, err
		}
		return IdempotentBody{
			Status:  http.StatusCreated,
			Payload: map[string]any{"id": leadID},
			AfterCommit: func() {
				// مكتب المنصة يعرف فوراً أن عميلاً ينتظر الموافقة
				s.notify.NotifyOps(r.Context(), notifications.Input{
					Kind: notifications.KindLead, Title: m.leadNewOps,
					Body: storeName, Entity: "lead", EntityID: leadID, Href: "/dashboard/leads",
				})
				s.touch("lead", "ops", "sales:"+repID)
			},
		}, nil
	})
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
	if err != nil || !leadStatuses[req.Status] {
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
	// **و«بحاجة معلومات» كالردّ** — ملاحظةٌ تقول للمندوب ما الناقص
	// (قرارُ المالك ٢٠٢٦-١٠-٠٤)، **وإلّا عاد إليه طلبٌ لا يعرف ما يصلح فيه.**
	if (req.Status == "rejected" || req.Status == "needs_info") && note == "" {
		s.respondErr(w, errReasonRequired)
		return
	}
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if req.Status == "converted" {
		welcome, err := s.convertLead(r.Context(), userIDFrom(r), id, clientIP(r))
		if err != nil {
			s.respondErr(w, err)
			return
		}
		// **ورسالةُ الدخول تُقال في جواب الموافقة نفسِه** — أوصلت أم لا، فإن لم
		// تصل ضغط المكتبُ «إعادة إرسال» من ملفّ صاحب المتجر.
		httpx.JSON(w, http.StatusOK, map[string]any{"updated": true, "welcome": welcome})
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
	// **ومَن ردّ الفرصةَ أو أعادها يُكتب** (قرارُ المالك ٢٠٢٦-١٠-٠٤، الرابع).
	//
	// كان التحديثُ بلا أثر — **والمندوبُ يسأل «مين رفض متجري؟» ولا جواب.**
	leadAction := "ops.lead_rejected"
	switch req.Status {
	case "new":
		leadAction = "ops.lead_reopened"
	case "needs_info":
		leadAction = "ops.lead_needs_info"
	}
	leadMeta := map[string]any{"name": storeName}
	if note != "" {
		leadMeta["note"] = note
	}
	if repID != nil {
		leadMeta["sales_rep_id"] = *repID
	}
	s.audit(r, leadAction, "lead", id, leadMeta)
	// الرفض يخصّ المندوب بقدر ما تخصّه الموافقة — وإلا بقي يلاحق عميلاً ميتاً.
	// **والسببُ في متن الإشعار** — لا في صفحةٍ يُطلب منه أن يفتحها.
	if req.Status == "rejected" && repID != nil {
		s.notify.Notify(r.Context(), notifications.Input{
			UserID: *repID, Kind: notifications.KindLead,
			Title: notifTitles.leadRejected, Body: storeName + " — " + note,
			Entity: "lead", EntityID: id, Href: "/portal/leads",
			// **إلى تطبيق المندوب وحدَه** (OBS-R7) — لا يرنّ على تطبيق زبونه.
			Apps: []string{notifications.AppRep},
		})
	}
	// **و«بحاجة معلومات» تصل المندوبَ بملاحظتها** — والطلبُ يبقى في قائمته
	// (`GET /rep/leads`) بحالته `needs_info` وملاحظتِه `decision_note`.
	if req.Status == "needs_info" && repID != nil {
		s.notify.Notify(r.Context(), notifications.Input{
			UserID: *repID, Kind: notifications.KindLead,
			Title: m.leadNeedsInfo, Body: storeName + " — " + note,
			Entity: "lead", EntityID: id, Href: "/portal/leads",
			Apps: []string{notifications.AppRep},
		})
	}
	s.touch("lead", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
}

// convertLead يحوّل طلب انضمام إلى متجر فعلي: ينشئ/يربط حساب صاحب المتجر (بكلمة
// مروره المحفوظة إن كان جديداً) والمتجر بتصنيفه وموقعه منسوباً للمندوب.
// ══════════════════════════════════════════════════════════════════════
// **grantSalesTargetIfAny — هدفُ المندوب يُدفع حيث يقع فعلُه**
// ══════════════════════════════════════════════════════════════════════
//
// **وفعلُ المندوب فتحُ متجر** — **وينتهي يومَ يوقّع العميل.** (قرارُ
// المالك ٢٠٢٦-٠٨-٣١: «الهدفُ الشهريّ هو عددُ العملاء المسجَّلين».)
//
// **وكان يُدفع عند تسليم طلبٍ من متاجره** — **فمندوبٌ فتح عشرةً في
// أسبوعٍ عدّادُه صفرٌ حتّى يشتري الناس.** وذلك يقيس السوقَ لا المندوب.
//
// # ولماذا بابان يناديانها
//
// **المتجرُ يُفتح من طريقين**: تحويلُ طلبِ مندوب، **وإنشاءٌ من لوحة
// الإدارة برمز مندوب.** **ومن نادى واحداً وترك الآخر ترك مندوباً
// يفتح عملاءَ ولا يُكافأ**، ولا خطأَ يظهر.
//
// **ولا تُنادى في `CreateMerchant`**: تلك تُنشئ متاجرَ بلا مندوبٍ
// أيضاً، **ودالّةُ كتالوجٍ تصرف مالاً تُفاجئ من يناديها.**
//
// **وخطؤها لا يُسقط الإنشاء**: المتجرُ فُتح، **ومكافأةٌ تأخّرت أهونُ
// من عميلٍ ضاع.** والقاعدةُ تمنع التكرار — فهرسٌ فريدٌ لكلّ شهر.
//
// **وكان خطؤها يُبلَع** (قرارُ المالك ٢٠٢٦-١٠-٠٤، قسمُ الأهداف): لا سطرَ في
// السجلّ، **ومن وقف عند الهدف بالضبط لا يقبضها أبداً.** والآن بمعاملتها،
// وعثرتُها تُكتب في السجلّ وفي `incentive_grant_failures` فتُعاد دوريّاً.
func (s *Server) grantSalesTargetIfAny(ctx context.Context, merchantID string) {
	if s.incentives == nil || merchantID == "" {
		return
	}
	rep := s.salesOpenerOf(ctx, s.pg, merchantID)
	if rep == "" {
		return
	}
	paid := s.incentives.GrantTargetIfReached(ctx, rep, "sales")
	s.notifySalesTargetPaid(ctx, merchantID, paid)
}

// salesOpenerOf **المندوبُ الذي فتح المتجر** — فعّالاً؛ وإلّا فراغ.
//
// **ويُحسب له للأبد** (`opened_by_rep_id`، قرارُ المالك ٢٠٢٦-١٠-٠٤): النقلُ
// ينقل العمولةَ القادمةَ وحدَها لا رصيدَ الهدف.
func (s *Server) salesOpenerOf(ctx context.Context, q dbtx.Querier, merchantID string) string {
	var repID *string
	// **والمندوبُ غيرُ الفعّال لا يُحسب له هدفٌ ولا مكافأة** (قرارُ المالك ٢٠٢٦-١٠-٠٤).
	if err := q.QueryRow(ctx,
		`SELECT m.opened_by_rep_id::text FROM merchants m
		   JOIN users u ON u.id = m.opened_by_rep_id AND u.status = 'active'
		  WHERE m.id = $1`,
		merchantID).Scan(&repID); err != nil || repID == nil {
		return ""
	}
	return *repID
}

// grantSalesTargetTx يمنح المكافأةَ **في معاملةٍ مُمرَّرة** ويُرجع ما دُفع.
//
// **ولا يُشعِر** — **الإشعارُ بعد التثبيت** (`PF-01`): **إشعارٌ خرج ثمّ
// ارتدّت المعاملةُ كذبٌ لا يُسحَب.**
func (s *Server) grantSalesTargetTx(ctx context.Context, q dbtx.Querier,
	merchantID string) (int64, error) {
	if s.incentives == nil || merchantID == "" {
		return 0, nil
	}
	rep := s.salesOpenerOf(ctx, q, merchantID)
	if rep == "" {
		return 0, nil
	}
	return s.incentives.GrantTargetIfReachedTx(ctx, q, rep, "sales")
}

// notifySalesTargetPaid يخبر المندوبَ ببلوغ هدفه.
//
// ══════════════════════════════════════════════════════════════════════
// **ومن نال مكافأتَه يُبشَّر بها**
// ══════════════════════════════════════════════════════════════════════
//
// (قِيس 2026-09-02: المكافأةُ تُقيَّد في محفظته آليّاً **ولا شيءَ يقول
//
//	له** — فيراها رقماً زاد بلا سبب.)
//
// **وصفرٌ يعني «لم يبلغ أو نالها من قبل»** — **ولا يُبشَّر أحدٌ بمالٍ
// لم يُقيَّد.**
//
// **وحزمةُ الحوافز بلا مُشعِر عن قصد**: تُدفع في الخلفيّة بلا فاعلٍ
// بشريّ، **والمنادي هو من يعرف جمهورَه.**
func (s *Server) notifySalesTargetPaid(ctx context.Context, merchantID string, paid int64) {
	if paid <= 0 {
		return
	}
	var repID *string
	if err := s.pg.QueryRow(ctx,
		`SELECT opened_by_rep_id::text FROM merchants WHERE id = $1`,
		merchantID).Scan(&repID); err != nil || repID == nil || *repID == "" {
		return
	}
	s.notify.Notify(ctx, notifications.Input{
		UserID: *repID, Kind: notifications.KindWallet,
		Title:  notifTitles.targetReached,
		Body:   strconv.FormatInt(paid, 10) + " " + currencyWord,
		Entity: "wallet", Href: "/portal/wallet",
		Apps: []string{notifications.AppRep},
	})
}

func (s *Server) convertLead(ctx context.Context, actorID, leadID, ip string) (map[string]any, error) {
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
		return nil, httpx.ErrNotFound
	}
	if merchantID != nil {
		return nil, nil // محوّل مسبقاً — لا تكرار
	}
	if categoryID == nil {
		return nil, errValidation // لا متجر بلا تصنيف
	}
	// ══════════════════════════════════════════════════════════════════
	// **والعنوانُ يُبنى من المنطقة والتفصيل معاً**
	// ══════════════════════════════════════════════════════════════════
	//
	// **وكان النصَّ الحرَّ وحدَه**: «وسط المدينة» عنوانُ متجرٍ لا يُوصَل
	// إليه — **لا مدينةَ فيه ولا محافظة.** ومن قرأه في لوحةٍ لا يعرف
	// أفي الرقّة هو أم في حلب.
	//
	// **والترتيبُ من العامّ إلى الخاصّ**: «مركز الرقة، الرقة — مقابل
	// الجامع». **وعكسُه يجعل أوّلَ ما يُقرأ أغمضَ ما فيه.**
	addr := area
	if label := s.districtLabelByLead(ctx, leadID); label != "" {
		if addr == "" {
			addr = label
		} else {
			addr = label + " — " + addr
		}
	}
	// ══════════════════════════════════════════════════════════════════
	// **وكلمةُ الدخول تُولَّد قبل المعاملة وتُكتب فيها** (قرارُ المالك ٢٠٢٦-١٠-٠٤)
	// ══════════════════════════════════════════════════════════════════
	//
	// **كانت خطوةً ثانيةً بعد التثبيت** (`issueWelcomeFor`) — معاملةٌ أخرى
	// تكتب الكلمةَ ومهلتَها. **فإن سقطت بقي صاحبُ متجرٍ جديدٌ بحسابٍ بلا
	// كلمةٍ يعرفها أحد**، والمتجرُ قائمٌ والطلبُ محوَّل، وجوابُ الموافقة لا
	// يقول أوصلت الرسالةُ أم لا.
	//
	// **والآن الحسابُ والكلمةُ المؤقّتةُ ومهلتُها والمتجرُ والتحويلُ معاملةٌ
	// واحدة** — كما في إنشاء المتجر من اللوحة. **والرسالةُ وحدَها بعد
	// التثبيت في النداء نفسِه**: رسالةٌ خرجت ثمّ ارتدّت المعاملةُ كذبٌ لا يُسحَب.
	//
	// **ولا كلمةَ من المندوب** — بصمةُ مرشَّحٍ قديمٍ تُهمَل.
	_ = pwHash
	plain, err := identity.GenerateTempPassword(s.minPasswordLen(ctx))
	if err != nil {
		return nil, err
	}
	// **والمهلةُ تُقرأ قبل المعاملة** — قراءةُ إعدادٍ من المَسبَح وهي مفتوحةٌ تحجز
	// اتّصالاً ثانياً، **ونداءان متزاحمان على مَسبَحٍ ضيّقٍ يحبس كلٌّ منهما الآخر.**
	tempHours := s.identity.TempPasswordHours(ctx)
	// **والبصمةُ قبلها أيضاً** — وتُكتب بيدنا داخل المعاملة لا بـ`EnsureUserWithRoleTx`
	// (ذاك يقرأ حدَّ الطول من المَسبَح وهو داخلَها).
	tempHash, err := auth.HashPassword(plain)
	if err != nil {
		return nil, err
	}
	in := catalog.MerchantInput{
		Name:        &storeName,
		CategoryID:  categoryID,
		Phone:       &phone,
		AddressText: &addr,
		OwnerPhone:  &phone,
		Lat:         lat,
		Lng:         lng,
	}
	if repCode != nil {
		in.SalesRepCode = repCode
	}

	// ══════════════════════════════════════════════════════════════
	// **التحويلُ يقع كلُّه أو لا يقع** — `PF-01` · `D2` · `D25` · `XG-18`
	// ══════════════════════════════════════════════════════════════
	//
	// **كانت ستَّ كتاباتٍ متتاليةً بلا معاملة، وأربعٌ منها خطؤها
	// مُهمَلٌ بـ`_, _ =`.** **فسقوطُ التثبيتِ الأخيرِ يترك**: متجراً
	// مُنشأً · **ومكافأةً مدفوعةً** · ومرشَّحاً ما يزال `new` · وردّاً
	// `500` بلا بيان.
	//
	// **وأخطرُ من ذلك**: **الإعادةُ تُنشئ متجراً ثانياً** —
	// `RETRY SAFETY = BROKEN`. (أُثبت بالحقن: متاجرُ=1 · المرشَّحُ `new`.)
	//
	// **والأخطاءُ لم تعُد تُهمَل**: **معاملةٌ تُثبَّت وفيها كتابةٌ سقطت
	// أسوأُ من لا معاملة** — تُخفي العطبَ وتدّعي الذرّيّة.
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// ══════════════════════════════════════════════════════════════
	// **ويُقفَل صفُّ المرشَّح ثمّ يُعاد الفحصُ تحته** — `XG-18` · `C-03`
	// ══════════════════════════════════════════════════════════════
	//
	// **الفحصُ أعلاه وقع خارجَ المعاملة** — **وبين القراءة والكتابة
	// فجوةٌ يدخل منها نداءٌ ثانٍ.** **فيريان `merchant_id` فارغاً معاً
	// ويُنشئ كلٌّ منهما متجراً.**
	//
	// **وكانت الحمايةُ عرَضيّةً**: يصطدمان بتفرّد `users.phone` حين
	// يُنشئان صاحبَ المتجر. **ومن كان له حسابٌ من قبلُ لا اصطدامَ
	// فيه** — **فوقع متجران في ستّ جولاتٍ من ست.**
	//
	// **والقفلُ يُسلسلهما**: الثاني ينتظر ثمّ يقرأ ما ثبّته الأوّل.
	var lockedMerchant *string
	if err := tx.QueryRow(ctx,
		`SELECT merchant_id::text FROM merchant_leads WHERE id = $1 FOR UPDATE`,
		leadID).Scan(&lockedMerchant); err != nil {
		return nil, err
	}
	if lockedMerchant != nil {
		// **ونتيجةٌ حتميّةٌ لا خطأ**: **العقدُ القائمُ يعدّ التحويلَ
		// المكرَّرَ لا شيءَ يُفعَل** — والمرشَّحُ محوَّلٌ فعلاً.
		return nil, nil
	}

	// هل يملك الرقم حساباً مسبقاً؟ حرج أمنياً: الكلمةُ المولَّدة يجب ألّا تُطبَّق
	// على حسابٍ قائم (وإلا أمكن لمهاجمٍ «التسجيلُ» برقم ضحيّة ثمّ الاستيلاءُ على
	// حسابها عند الموافقة). **و`EnsureUserWithRoleTx` لا يمسّ كلمةَ القائم** —
	// والسؤالُ هنا داخلَ المعاملة ليقول الجوابُ ما وقع فعلاً.
	normPhone, _ := identity.NormalizePhone(phone)
	var ownerExisted bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE phone = $1)`,
		normPhone).Scan(&ownerExisted); err != nil {
		return nil, err
	}

	merchantNewID, err := s.catalog.CreateMerchantTx(ctx, tx, actorID, in, ip)
	if err != nil {
		return nil, err
	}
	// الاسم يُملأ إن كان فارغاً (غير حسّاس).
	var ownerID *string
	if err := tx.QueryRow(ctx, `
		UPDATE users SET
			full_name  = CASE WHEN full_name = '' THEN $2 ELSE full_name END,
			updated_at = now()
		WHERE id = (SELECT owner_user_id FROM merchants WHERE id = $1)
		RETURNING id::text`,
		merchantNewID, ownerName).Scan(&ownerID); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	// **ومهلةُ الكلمة المؤقّتة في المعاملة نفسِها** — ٧٢ ساعةً من الإعدادات.
	// للحساب الجديد وحدَه: **القائمُ لم تُكتب له كلمة.** **و`created_at = now()`
	// حارسٌ ثانٍ**: لا تُكتب الكلمةُ إلّا لحسابٍ وُلد في هذه المعاملة نفسِها.
	var expires *time.Time
	if ownerID != nil && !ownerExisted {
		var exp time.Time
		err := tx.QueryRow(ctx, `
			UPDATE users SET password_hash = $3, must_change_password = true,
			       temp_password_expires_at = now() + ($2::int * interval '1 hour')
			 WHERE id = $1::uuid AND created_at = now()
			RETURNING temp_password_expires_at`,
			*ownerID, tempHours, tempHash).Scan(&exp)
		switch {
		case err == nil:
			expires = &exp
		case errors.Is(err, pgx.ErrNoRows):
			ownerExisted = true // وُجد قبلنا — فرسالتُه «صار عندك متجر»
		default:
			return nil, err
		}
	}
	// ══════════════════════════════════════════════════════════════════
	// **والمنطقةُ تنتقل إلى المتجر مع الموافقة**
	// ══════════════════════════════════════════════════════════════════
	//
	// **وهنا لا في `CreateMerchant`**: تلك تُنشئ متاجرَ من اللوحة أيضاً،
	// **ولا منطقةَ في نموذجها.** ومن أضاف حقلاً إليها لأجل هذا الباب
	// جعل كلَّ من يناديها يمرّ بحقلٍ لا يعنيه.
	//
	// **والمتجرُ يقول من أيّ مرشَّحٍ جاء** — **وعليه فهرسٌ فريد**
	// (`merchants_lead_uq`)، **فثانٍ للمرشَّح نفسِه يُرفض في لحظة
	// إدخاله ولو سقط القفلُ أعلاه.** **حارسان لا واحد.**
	if _, err := tx.Exec(ctx, `
		UPDATE merchants SET
			lead_id = $2,
			district_id = COALESCE(district_id,
				(SELECT district_id FROM merchant_leads WHERE id = $2))
		WHERE id = $1`, merchantNewID, leadID); err != nil {
		return nil, err
	}

	// **وهدفُ المندوب الشهريّ يُحسب لحظةَ إنشاء المتجر** (قرارُ المالك
	// ٢٠٢٦-١٠-٠٤: «مو ذنب المندوب إذا ما صار طلب») — **والعمولةُ لا**:
	// تُقيَّد عند تسليم أوّل طلبٍ ناجح (`settleRep`). ويحرسهما
	// `TestLeadConvert_TargetAtCreationCommissionAtDelivery`.
	paid, err := s.grantSalesTargetTx(ctx, tx, merchantNewID)
	if err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE merchant_leads SET status = 'converted', merchant_id = $2, updated_at = now()
		WHERE id = $1`, leadID, merchantNewID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// ── وبعد التثبيت تُرسَل الإشعارات ──────────────────────────
	//
	// **ولا تُرسَل من داخل المعاملة**: **إشعارٌ خرج ثمّ ارتدّت المعاملةُ
	// كذبٌ لا يُسحَب** — يقرأ المندوبُ «قُبل متجرُك» ولا متجرَ.
	s.notifySalesTargetPaid(ctx, merchantNewID, paid)
	// **ورسالةُ الدخول لصاحب المتجر في النداء نفسِه** — جديدٌ: كلمتُه
	// المؤقّتةُ التي كُتبت في المعاملة ورابطُ تطبيق المتجر · قائمٌ: «صار
	// عندك متجر» بلا كلمة.
	welcome := map[string]any{"sent": false, "existing_owner": ownerExisted}
	if ownerID != nil {
		welcome["user_id"] = *ownerID
		if ownerExisted {
			welcome["sent"] = s.notifyNewStoreOwner(ctx, actorID, *ownerID, storeName, ip)
		} else {
			sent := s.sendWelcome(ctx, actorID, *ownerID, ip, plain, "merchant", false)
			welcome["sent"] = sent
			welcome["temp_password"] = s.revealTemp(sent, plain)
			if expires != nil {
				welcome["expires_at"] = *expires
			}
		}
	}
	// المندوب يعرف فوراً أن عميله اعتُمد (مصدر عمولته)
	var repID *string
	_ = s.pg.QueryRow(ctx, `SELECT sales_rep_user_id FROM merchant_leads WHERE id = $1`, leadID).Scan(&repID)
	if repID != nil {
		s.notify.Notify(ctx, notifications.Input{
			UserID: *repID, Kind: notifications.KindLead,
			Title: m.leadApproved, Body: storeName,
			Entity: "merchant", EntityID: merchantNewID, Href: "/portal/merchants",
		})
	}
	return welcome, nil
}
