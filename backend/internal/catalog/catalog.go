// Package catalog الفئات الديناميكية والمتاجر.
package catalog

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/media"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/settings"
)

var (
	ErrCategoryInvalid = httpx.NewError(http.StatusBadRequest, "invalid_category", "errors.invalid_category")
	ErrNameRequired    = httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")

	// ══════════════════════════════════════════════════════════════════
	// **ولا متجرَ بلا موضعٍ على الأرض**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-١٢: «نسوي دبّوس المتجر إلزامي مو اختياري عند
	//  فتح الحساب».)
	//
	// **وكان العمودُ يقبل الفراغ** — فيُفتح متجرٌ ويستقبل طلباتٍ **وهو
	// بلا نقطةٍ على الخريطة.** ووقع فعلاً: طلبٌ حقيقيٌّ (#1003) لم تظهر
	// للسائق مسافتُه **لأنّ متجرَه بلا دبّوس**، والمحرّكُ يردّ `-1` أي
	// «لا يُعرف».
	//
	// **وعليه يقوم كلُّ شيء**: المسافةُ التي يقرّر بها السائق · الخريطةُ
	// التي يمشي عليها · وتوزيعُ «الأقرب» نفسُه.
	ErrLocationRequired = httpx.NewError(http.StatusBadRequest,
		"merchant_location_required", "errors.merchant_location_required")
)

type Category struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Icon      string `json:"icon"`
	SortOrder int    `json:"sort_order"`
	Active    bool   `json:"active"`
}

type Merchant struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	CategoryID    string   `json:"category_id"`
	CategoryName  string   `json:"category_name"`
	CategoryIcon  string   `json:"category_icon"`
	Phone         string   `json:"phone"`
	AddressText   string   `json:"address_text"`
	OwnerUserID   *string  `json:"owner_user_id"`
	OwnerPhone    *string  `json:"owner_phone"`
	SalesRepPhone *string  `json:"sales_rep_phone"`
	SalesRepCode  *string  `json:"sales_rep_code"`
	Lat           *float64 `json:"lat"`
	Lng           *float64 `json:"lng"`
	LogoURL       *string  `json:"logo_url"`
	LogoThumbURL  *string  `json:"logo_thumb_url"`
	Status        string   `json:"status"`
	// Violations إلغاءاتُ المتجر داخل نافذة الحظر وبعد آخر عفو.
	//
	// **في القائمة لا في صفحةٍ منفصلة**: متجرٌ على ٤ من ٥ تتّصل به العملياتُ
	// فتنقذ الطرفين — **وعدّادٌ لا يُرى إلا بفتح صفحةٍ عدّادٌ لا يُقرأ.**
	Violations int `json:"violations"`
	// CommissionPct **تجاوزُ عمولة هذا المتجر** — و`null` تعني «اتبع العامّ».
	//
	// كان رقماً دائماً يُنسخ لحظةَ الإنشاء، **فتغييرُ المفتاح لا يمسّ متجراً
	// قائماً.** (الترحيل ٠٠٦٧.)
	CommissionPct   *int64 `json:"commission_percent"`
	EmergencyClosed bool   `json:"emergency_closed"`
	// AcceptsReturns أيستردّ بضاعةَ طلبٍ تعذّر تسليمُه — **وعليه يظهر زرُّ
	// «رُدّت إلى المتجر» في شاشة العمليات.**
	AcceptsReturns bool      `json:"accepts_returns"`
	CreatedAt      time.Time `json:"created_at"`
}

type MerchantPage struct {
	Merchants []Merchant `json:"merchants"`
	Total     int        `json:"total"`
	Page      int        `json:"page"`
	PerPage   int        `json:"per_page"`
}

type Service struct {
	db       *pgxpool.Pool
	identity *identity.Service
	// settings مفاتيحُ الهامش — **تُقرأ عند كلّ عرضٍ لا تُخزَّن**، فتغييرُ
	// المالك يظهر في القائمة فوراً بلا إعادة حسابِ ألف صنف.
	settings *settings.Store
}

func NewService(db *pgxpool.Pool, identitySvc *identity.Service) *Service {
	return &Service{db: db, identity: identitySvc}
}

// SetSettings يحقن مخزن الإعدادات (يُنادى مرّة عند الإقلاع).
func (s *Service) SetSettings(st *settings.Store) { s.settings = st }

// ---------- الفئات ----------

func (s *Service) ListCategories(ctx context.Context, activeOnly bool) ([]Category, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, name, icon, sort_order, active FROM categories
		WHERE NOT $1 OR active
		ORDER BY sort_order, name`, activeOnly)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Icon, &c.SortOrder, &c.Active); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

type CategoryInput struct {
	Name      *string `json:"name"`
	Icon      *string `json:"icon"`
	SortOrder *int    `json:"sort_order"`
	Active    *bool   `json:"active"`
}

func (s *Service) CreateCategory(ctx context.Context, actorID string, in CategoryInput, ip string) (*Category, error) {
	if in.Name == nil || *in.Name == "" {
		return nil, ErrNameRequired
	}
	var c Category
	err := s.db.QueryRow(ctx, `
		INSERT INTO categories (name, icon, sort_order)
		VALUES ($1, COALESCE($2, ''), COALESCE($3, 0))
		RETURNING id, name, icon, sort_order, active`,
		*in.Name, in.Icon, in.SortOrder).
		Scan(&c.ID, &c.Name, &c.Icon, &c.SortOrder, &c.Active)
	if err != nil {
		return nil, err
	}
	s.audit(ctx, actorID, "admin.category_create", "category", c.ID, ip)
	return &c, nil
}

func (s *Service) UpdateCategory(ctx context.Context, actorID, id string, in CategoryInput, ip string) (*Category, error) {
	var c Category
	err := s.db.QueryRow(ctx, `
		UPDATE categories SET
			name       = COALESCE($2, name),
			icon       = COALESCE($3, icon),
			sort_order = COALESCE($4, sort_order),
			active     = COALESCE($5, active)
		WHERE id = $1
		RETURNING id, name, icon, sort_order, active`,
		id, in.Name, in.Icon, in.SortOrder, in.Active).
		Scan(&c.ID, &c.Name, &c.Icon, &c.SortOrder, &c.Active)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	s.audit(ctx, actorID, "admin.category_update", "category", id, ip)
	return &c, nil
}

// ---------- المتاجر ----------

// merchantSelect **دالّةٌ لا ثابت** — لأنّها تبني عدّادَ المخالفات من مصدره
// (`orders.ViolationsCountSQL`) بدل أن تكتبه من جديد، **ورقمُ معامل النافذة
// يختلف باختلاف الاستعلام**: القائمةُ تحمل مرشِّحاتٍ قبله، والمفردُ لا يحمل.
func merchantSelect(daysExpr string) string {
	return `
	SELECT m.id, m.name, m.description, m.category_id, c.name, c.icon,
	       m.phone, m.address_text, m.owner_user_id, u.phone, sr.phone, sr.invite_code,
	       ST_Y(m.location::geometry), ST_X(m.location::geometry),
	       lm.path, lm.thumb_path,
	       m.status,
	       -- **بشرط العدّ نفسه** الذي يحظر — لا بشرطٍ يشبهه.
	       --
	       -- وكان هنا عدّادٌ ثانٍ مكتوبٌ بيده: يعدّ الإلغاءَ والرفض **ويُغفل
	       -- الفشلَ بذنب المتجر والإنذاراتِ اليدوية**، ونافذتُه ثلاثون يوماً
	       -- ثابتةً لا الإعداد. **فيرى المالكُ في القائمة «٢» وفي الملفّ «٤»**
	       -- ولا يعرف أيّهما يُصدّق ولا أيّهما يحظر.
	       --
	       -- والتعليقُ القديم كان يقول «بشرط العدّ نفسه لا بشرطٍ يشبهه» —
	       -- **والوصفُ صحيحٌ والتنفيذُ خالفه.** فصار الشرطُ يأتي من مصدره.
	       ` + orders.ViolationsCountSQL("m.id", daysExpr) + `,
	       m.commission_percent, m.emergency_closed, m.accepts_returns, m.created_at
	FROM merchants m
	JOIN categories c ON c.id = m.category_id
	LEFT JOIN users u ON u.id = m.owner_user_id
	LEFT JOIN users sr ON sr.id = m.sales_rep_user_id
	LEFT JOIN media lm ON lm.id = m.logo_media_id`
}

// banDays نافذةُ عدّ المخالفات — **من الإعداد لا من رقمٍ ثابت.**
//
// **وثلاثون عند الجهل** كما في المحرّك، فلا يختلف عرضٌ عن قرار.
func (s *Service) banDays(ctx context.Context) int64 {
	if s.settings != nil {
		if v := s.settings.GetInt(ctx, "merchants.cancel_ban_days"); v > 0 {
			return v
		}
	}
	return 30
}

func scanMerchant(row pgx.Row) (*Merchant, error) {
	var m Merchant
	err := row.Scan(&m.ID, &m.Name, &m.Description, &m.CategoryID, &m.CategoryName, &m.CategoryIcon,
		&m.Phone, &m.AddressText, &m.OwnerUserID, &m.OwnerPhone, &m.SalesRepPhone, &m.SalesRepCode,
		&m.Lat, &m.Lng, &m.LogoURL, &m.LogoThumbURL,
		&m.Status, &m.Violations, &m.CommissionPct, &m.EmergencyClosed, &m.AcceptsReturns, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	m.LogoURL = media.URLForPtr(m.LogoURL)
	m.LogoThumbURL = media.URLForPtr(m.LogoThumbURL)
	return &m, nil
}

// ListMerchants متاجرُ المنصة بمرشِّحاتها.
//
// **و`repID` يجعلها تصلح لملفّ المندوب**: «أيُّ متاجرَ جلبها هذا؟» سؤالٌ يُسأل
// في ملفّه لا في قائمةٍ عامّةٍ تُبحث بالاسم. **ونقطةٌ ثانيةٌ تُبنى لأجله كانت
// ستُكرّر الاستعلامَ نفسَه بشرطٍ واحدٍ زائد** — وهي عائلةُ «قاعدةٌ مكتوبةٌ
// مرّتين» التي أتعبتنا.
func (s *Service) ListMerchants(ctx context.Context, query, categoryID, status, repID string, page, perPage int) (*MerchantPage, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	where := ` WHERE ($1 = '' OR m.name ILIKE '%'||$1||'%' OR m.phone ILIKE '%'||$1||'%')
	           AND ($2 = '' OR m.category_id::text = $2)
	           AND ($3 = '' OR m.status = $3)
	           AND ($4 = '' OR m.sales_rep_user_id::text = $4)`

	var total int
	if err := s.db.QueryRow(ctx, `SELECT count(*) FROM merchants m`+where,
		query, categoryID, status, repID).Scan(&total); err != nil {
		return nil, err
	}

	rows, err := s.db.Query(ctx, merchantSelect("$5")+where+`
		ORDER BY m.created_at DESC LIMIT $6 OFFSET $7`,
		query, categoryID, status, repID, s.banDays(ctx), perPage, (page-1)*perPage)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	merchants := []Merchant{}
	for rows.Next() {
		m, err := scanMerchant(rows)
		if err != nil {
			return nil, err
		}
		merchants = append(merchants, *m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &MerchantPage{Merchants: merchants, Total: total, Page: page, PerPage: perPage}, nil
}

type MerchantInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	CategoryID  *string `json:"category_id"`
	Phone       *string `json:"phone"`
	AddressText *string `json:"address_text"`
	Status      *string `json:"status"`
	// CommissionPct تجاوزُ عمولة هذا المتجر — **وسالبُ الواحدِ يمحوه**
	// فيعود إلى `merchants.commission_value` العامّ.
	CommissionPct   *int64 `json:"commission_percent"`
	EmergencyClosed *bool  `json:"emergency_closed"`
	// AcceptsReturns أيستردّ هذا المتجرُ بضاعةَ طلبٍ تعذّر تسليمُه.
	//
	// **بندٌ في الاتّفاق معه لا رأيٌ يُبديه ساعتَها** — وعليه يظهر زرُّ «رُدّت
	// إلى المتجر» في شاشة العمليات. **ومن يملك تغييرَه وحدَه يغلقه ساعةَ تُردّ
	// إليه بضاعة**، ولذلك موضعُه بطاقةُ المتجر عند الإدارة (قرارُ المالك
	// ٢٠٢٦-٠٨-٠٤).
	AcceptsReturns *bool    `json:"accepts_returns"`
	OwnerPhone     *string  `json:"owner_phone"`    // يربط/ينشئ حساب صاحب المتجر بدور merchant
	SalesRepCode   *string  `json:"sales_rep_code"` // كود دعوة المندوب — يُنسب له المتجر
	Lat            *float64 `json:"lat"`            // دبوس الموقع على الخريطة
	Lng            *float64 `json:"lng"`
	// معرف وسائط الشعار: غير مُرسل = بلا تغيير، "" = إزالة الشعار
	LogoMediaID *string `json:"logo_media_id"`
}

// requirePoint **يتحقّق أنّ الدبّوس موجودٌ وفي حدود الأرض.**
//
// **والصفرُ نقطةٌ صالحةٌ في البحر قرب غانا** — فلا يُقبل ضمناً على أنّه
// «فارغ»: من أرسل `0,0` أرسل موضعاً، **وهو ليس موضعَ متجرٍ في الرقّة.**
func requirePoint(lat, lng *float64) error {
	if lat == nil || lng == nil {
		return ErrLocationRequired
	}
	if *lat < -90 || *lat > 90 || *lng < -180 || *lng > 180 {
		return ErrLocationRequired
	}
	if *lat == 0 && *lng == 0 {
		return ErrLocationRequired
	}
	return nil
}

func (s *Service) CreateMerchant(ctx context.Context, actorID string, in MerchantInput, ip string) (*Merchant, error) {
	if in.Name == nil || *in.Name == "" || in.CategoryID == nil {
		return nil, ErrNameRequired
	}
	// **ويُفحص هنا لا في الواجهة** — الإنشاءُ يقع من لوحة الإدارة ومن
	// شاشة المندوب معاً، **وحارسٌ في واجهةٍ واحدةٍ يُلتفّ عليه من
	// الأخرى.**
	if err := requirePoint(in.Lat, in.Lng); err != nil {
		return nil, err
	}
	ownerID, err := s.resolveOwner(ctx, actorID, in.OwnerPhone, ip)
	if err != nil {
		return nil, err
	}
	repID, err := s.resolveRep(ctx, in.SalesRepCode)
	if err != nil {
		return nil, err
	}

	var id string
	err = s.db.QueryRow(ctx, `
		INSERT INTO merchants (name, description, category_id, phone, address_text, owner_user_id, sales_rep_user_id, location, logo_media_id, commission_percent, default_prep_minutes)
		VALUES ($1, COALESCE($2,''), $3, COALESCE($4,''), COALESCE($5,''), $6, $7,
		        CASE WHEN $8::float8 IS NOT NULL AND $9::float8 IS NOT NULL
		             THEN ST_SetSRID(ST_MakePoint($9::float8, $8::float8), 4326)::geography END,
		        NULLIF(COALESCE($10, ''), '')::uuid,
		        -- **ولا يُنسخ الافتراضُ في العمود.**
		        --
		        -- كان يُقرأ المفتاحُ هنا بـCOALESCE — فيولد كلُّ متجرٍ برقمٍ
		        -- خاصٍّ به يساوي العامَّ يومَ وُلد، **ثمّ لا يتحرّك حين يتحرّك
		        -- العامّ.** والفراغُ الآن يعني «اتبع العامّ» فعلاً.
		        $11,

		        -- وقت التحضير الافتراضي من اللوحة لا من افتراض العمود: كان ٢٠
		        -- مكتوباً في الترحيل 0035، يرثه كل متجرٍ جديد ولا يملك المالك
		        -- تغييره لمن يأتي بعده.
		        -- **ومن المخزن لا برقمٍ مكتوبٍ هنا** — الافتراضُ في الفهرس وحدَه.
		        $12)
		RETURNING id`,
		*in.Name, in.Description, *in.CategoryID, in.Phone, in.AddressText, ownerID,
		repID, in.Lat, in.Lng, in.LogoMediaID, in.CommissionPct,
		s.settings.GetInt(ctx, "merchants.default_prep_minutes")).Scan(&id)
	if isFKViolation(err) {
		return nil, ErrCategoryInvalid
	}
	if err != nil {
		return nil, err
	}
	s.audit(ctx, actorID, "admin.merchant_create", "merchant", id, ip)
	return s.merchantByID(ctx, id)
}

func (s *Service) UpdateMerchant(ctx context.Context, actorID, id string, in MerchantInput, ip string) (*Merchant, error) {
	// `suspended` تُقبل هنا للأدمن، ولها نقطتُها الخاصّة (merchant_violations.go)
	// التي تُسجّل السبب. **وقبولُها هنا يمنع حالةً لا تُرفع إلا بجراحةٍ في
	// القاعدة** لو أُغفلت.
	if in.Status != nil && *in.Status != "active" && *in.Status != "inactive" &&
		*in.Status != "suspended" {
		return nil, ErrNameRequired
	}
	ownerID, err := s.resolveOwner(ctx, actorID, in.OwnerPhone, ip)
	if err != nil {
		return nil, err
	}
	repID, err := s.resolveRep(ctx, in.SalesRepCode)
	if err != nil {
		return nil, err
	}

	tag, err := s.db.Exec(ctx, `
		UPDATE merchants SET
			name          = COALESCE($2, name),
			description   = COALESCE($3, description),
			category_id   = COALESCE($4, category_id),
			phone         = COALESCE($5, phone),
			address_text  = COALESCE($6, address_text),
			status        = COALESCE($7, status),
			-- **وسالبُ الواحدِ يمحو التجاوز.**
			--
			-- COALESCE وحدَه لا يفرّق بين «لم يُرسَل» و«أُرسل فارغاً» — وكلاهما
			-- NULL. **فمن أراد أن يعيد متجراً إلى العمولة العامّة لم يملك
			-- سبيلاً**: كلُّ إرسالٍ يُقرأ «بلا تغيير»، فيبقى رقمُه القديم
			-- يحكمه **بينما يظنّ المالكُ أنّ المفتاحَ العامَّ يحكمه.**
			commission_percent = CASE WHEN $13::bigint IS NULL THEN commission_percent
			                          WHEN $13 < 0 THEN NULL
			                          ELSE $13 END,
			emergency_closed = COALESCE($8, emergency_closed),
			accepts_returns = COALESCE($15, accepts_returns),
			owner_user_id = COALESCE($9, owner_user_id),
			sales_rep_user_id = COALESCE($10, sales_rep_user_id),
			location      = COALESCE(
				CASE WHEN $11::float8 IS NOT NULL AND $12::float8 IS NOT NULL
				     THEN ST_SetSRID(ST_MakePoint($12::float8, $11::float8), 4326)::geography END,
				location),
			logo_media_id = CASE WHEN $14::text IS NULL THEN logo_media_id
			                     ELSE NULLIF($14, '')::uuid END,
			updated_at    = now()
		WHERE id = $1`,
		id, in.Name, in.Description, in.CategoryID, in.Phone, in.AddressText, in.Status, in.EmergencyClosed, ownerID, repID, in.Lat, in.Lng, in.CommissionPct, in.LogoMediaID, in.AcceptsReturns)
	if isFKViolation(err) {
		return nil, ErrCategoryInvalid
	}
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, httpx.ErrNotFound
	}
	s.audit(ctx, actorID, "admin.merchant_update", "merchant", id, ip)
	return s.merchantByID(ctx, id)
}

// MerchantByID متجرٌ بعينه — **لملفّه المفرد.**
//
// **والمتجرُ كيانٌ لا شخص**: له قائمةٌ وساعاتٌ وعمولةٌ ومخالفاتٌ ومبيعات،
// **وصاحبُه حسابٌ آخر.** فيلزمه ملفٌّ كما لزم الأشخاصَ ملفُّهم — وكانت كلُّ
// هذه في نوافذَ منبثقةٍ داخل جدول، **تُفتح واحدةً وتُغلق لتُفتح أخرى.**
func (s *Service) MerchantByID(ctx context.Context, id string) (*Merchant, error) {
	return s.merchantByID(ctx, id)
}

func (s *Service) merchantByID(ctx context.Context, id string) (*Merchant, error) {
	m, err := scanMerchant(s.db.QueryRow(ctx,
		merchantSelect("$2")+` WHERE m.id = $1`, id, s.banDays(ctx)))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	return m, err
}

// resolveRep ينسب المتجر للمندوب صاحب كود الدعوة.
func (s *Service) resolveRep(ctx context.Context, repCode *string) (*string, error) {
	if repCode == nil || *repCode == "" {
		return nil, nil
	}
	user, err := s.identity.SalesRepByInviteCode(ctx, *repCode)
	if err != nil {
		return nil, err
	}
	return &user.ID, nil
}

// resolveOwner يجد/ينشئ حساب صاحب المتجر بدور merchant من رقم هاتفه.
func (s *Service) resolveOwner(ctx context.Context, actorID string, ownerPhone *string, ip string) (*string, error) {
	if ownerPhone == nil || *ownerPhone == "" {
		return nil, nil
	}
	user, err := s.identity.EnsureUserWithRole(ctx, actorID, *ownerPhone, "merchant", ip)
	if err != nil {
		return nil, err
	}
	return &user.ID, nil
}

func (s *Service) audit(ctx context.Context, actorID, action, entity, entityID, ip string) {
	_, _ = s.db.Exec(ctx, `
		INSERT INTO audit_log (actor_user_id, action, entity, entity_id, ip)
		VALUES ($1, $2, $3, $4, $5)`, actorID, action, entity, entityID, ip)
}

func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == "22P02")
}
