package offers

// العروضُ والخصومات — **ما تُنزله المنصةُ بنفسها.**
//
// # الفرقُ عن أكواد الخصم
//
// **الكودُ يُكتب والعرضُ يُرى.** ومن لم يسمع بالكود لا يستفيد منه **ولا يعلم
// أنّه فاته.** والعرضُ في شاشته: يفتح الأيقونةَ فيرى.
//
// # والخصمُ يمرّ في الدفتر لا في العرض وحدَه
//
// **سعرٌ مشطوبٌ في الشاشة ثمنُه صفرٌ في الدفتر خدعة.** فيُطبَّق الخصمُ لحظةَ
// بناء الطلب على **لقطة البند** — وهي ما تقرؤه التسويةُ كلُّها.
//
//	تتحمّله المنصة  ←  سعرُ البيع ينزل، وسعرُ الشراء كما هو
//	                   **فالهامشُ يضيق** — والمتجرُ يقبض كاملاً
//	يتحمّله المتجر  ←  ينزلان معاً بالمقدار نفسِه
//	                   **فالهامشُ كما هو** — والمتجرُ يقبض أقلّ
//
// **ولا حسبةَ ثانيةً في التسوية**: هي تقرأ اللقطةَ كما تقرؤها دائماً.

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
)

var (
	ErrBadKind    = httpx.NewError(http.StatusBadRequest, "bad_offer_kind", "errors.validation")
	ErrNeedsTitle = httpx.NewError(http.StatusBadRequest, "offer_needs_title", "errors.validation")
	// ErrBadDiscount خصمٌ ناقص — صنفٌ ونسبةٌ ومن يتحمّل، ثلاثةٌ معاً.
	ErrBadDiscount = httpx.NewError(http.StatusBadRequest, "bad_offer_discount", "errors.validation")
	// ErrBadWindow **نهايةٌ قبل بدايةٍ — مدّةٌ لا تقع أبداً.**
	//
	// **وعرضٌ ينتهي قبل أن يبدأ يُقبَل صامتاً ثمّ لا يُسعَّر به قطّ** —
	// **فيُسأل «لماذا لا يعمل عرضي؟» ولا شيءَ في الشاشة يقول.**
	ErrBadWindow = httpx.NewError(http.StatusBadRequest, "bad_offer_window", "errors.validation")
	// ErrItemHasOffer **وخصمان على صنفٍ واحدٍ سؤالٌ بلا جواب.**
	ErrItemHasOffer = httpx.NewError(http.StatusConflict, "item_already_discounted", "errors.item_already_discounted")
)

// KindDiscount النوعُ الوحيدُ الباقي — **واللافتاتُ في جدولها** (`banners`).
//
// **وشيءٌ واحدٌ في مكانين يفترق**: كانت لافتةٌ هنا ولافتةٌ هناك، فتُضاف في
// أحدهما ولا تظهر في عرض الآخر. (انظر الترحيل ٠٠٧٥.)
const (
	KindDiscount = "discount"

	ByPlatform = "platform"
	ByMerchant = "merchant"
)

type Service struct{ db *pgxpool.Pool }

func New(db *pgxpool.Pool) *Service { return &Service{db: db} }

// Offer عرضٌ كما يُقرأ في الشاشتين.
type Offer struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Title string `json:"title"`
	Body  string `json:"body"`

	ImageURL *string `json:"image_url"`
	MediaID  *string `json:"media_id"`
	Href     string  `json:"href"`

	MenuItemID *string `json:"menu_item_id"`
	// ItemName و MerchantName و السعران — **تُقرأ مع العرض لا بنداءٍ لكلّ
	// بطاقة**: عشرةُ عروضٍ تعني عشرةَ نداءات، **وشاشةٌ بطيئةٌ لا تُفتح.**
	ItemName     string  `json:"item_name"`
	MerchantName string  `json:"merchant_name"`
	MerchantID   *string `json:"merchant_id"`
	ItemImageURL *string `json:"item_image_url"`
	// PriceBefore سعرُ البيع بلا خصم، و PriceAfter بعده.
	//
	// **ويُحسبان في الخادم**: الشاشةُ تعرض ما يُقال لها، **وحسبةٌ في متصفّحٍ
	// تفترق عمّا يُقيَّد في الطلب** — فيرى سعراً ويُحاسَب بآخر.
	PriceBefore int64 `json:"price_before"`
	PriceAfter  int64 `json:"price_after"`

	DiscountPercent *int    `json:"discount_percent"`
	BorneBy         *string `json:"borne_by"`

	StartsAt *time.Time `json:"starts_at"`
	EndsAt   *time.Time `json:"ends_at"`
	Active   bool       `json:"active"`
	// Live سارٍ الآن — **يقوله الخادمُ ولا يُستنتج في الشاشة**: شرطٌ يُحسب
	// في موضعين يفترق يوماً، **فتُعرض على الزبون عروضٌ انتهت.**
	Live bool `json:"live"`
	// Status الحالُ المشتقّة — **يقولها الخادمُ ولا تُستنتج في الشاشة.**
	//
	// **وشاشةٌ تحسبها بساعة الجهاز تقول «سارٍ» لعرضٍ انتهى** — **ومن
	// قدّم ساعتَه رأى عرضاً مجدولاً ساريا.**
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`

	// HasOptions للصنف خياراتٌ تُختار قبل الطلب — **حجمٌ أو إضافات.**
	//
	// **وشاشةُ العروض تُدخل الصنفَ السلّةَ بضغطةٍ واحدة** — فصنفٌ بحجمٍ
	// إلزاميٍّ يدخل بلا اختيار **ويُردّ الطلبُ كلُّه عند الإرسال**
	// (`orders/service.go`)، ولا يعرف الزبونُ أيَّ صنفٍ سبّبه.
	HasOptions bool `json:"has_options"`
}

// LiveCond شرطُ السريان — **تعبيرٌ واحدٌ يُعاد استعماله.**
//
// **ومكتوبٌ مرّةً**: الشاشةُ تقرؤه والزبونُ يقرؤه والمحرّكُ يقرؤه —
// **وثلاثُ نسخٍ تفترق فيُعرض ما لا يُطبَّق.**
//
// **وصُدِّر حين احتاجته شاشاتُ التصفّح**: بطاقةُ القسم ونافذةُ الصنف
// **كانتا تعرضان السعرَ كاملاً** والعرضُ يقول غيرَه — **ونسخُ الشرط هناك
// كان يفتح البابَ الذي أُغلق هنا.**
const LiveCond = `(o.active
	AND (o.starts_at IS NULL OR o.starts_at <= now())
	AND (o.ends_at IS NULL OR o.ends_at > now()))`

const offerSelect = `
	SELECT o.id::text, o.kind, o.title, o.body,
	       o.media_id::text, mm.path, o.href,
	       o.menu_item_id::text, COALESCE(mi.name, ''), COALESCE(mr.name, ''),
	       mr.id::text, im.path,
	       COALESCE(mi.price, 0),
	       o.discount_percent, o.borne_by,
	       o.starts_at, o.ends_at, o.active, ` + LiveCond + `, o.created_at,
	       EXISTS (SELECT 1 FROM modifier_groups g WHERE g.item_id = mi.id)
	FROM offers o
	LEFT JOIN media mm ON mm.id = o.media_id
	LEFT JOIN menu_items mi ON mi.id = o.menu_item_id
	LEFT JOIN merchants mr ON mr.id = mi.merchant_id
	LEFT JOIN media im ON im.id = mi.image_media_id`

func scan(rows interface {
	Scan(dest ...any) error
}, marginOf func(cost int64) int64) (*Offer, error) {
	var o Offer
	var cost int64
	if err := rows.Scan(&o.ID, &o.Kind, &o.Title, &o.Body,
		&o.MediaID, &o.ImageURL, &o.Href,
		&o.MenuItemID, &o.ItemName, &o.MerchantName, &o.MerchantID, &o.ItemImageURL,
		&cost, &o.DiscountPercent, &o.BorneBy,
		&o.StartsAt, &o.EndsAt, &o.Active, &o.Live, &o.CreatedAt,
		&o.HasOptions); err != nil {
		return nil, err
	}
	// **والحالُ تُشتقّ من الحقول نفسِها التي يقرؤها `LiveCond`** —
	// **ووقتُ الخادم**: `time.Now()` في العمليّة التي تقرأ القاعدة.
	o.Status = StatusAt(o.Active, o.StartsAt, o.EndsAt, time.Now())
	o.ImageURL = media.URLForPtr(o.ImageURL)
	o.ItemImageURL = media.URLForPtr(o.ItemImageURL)
	if o.DiscountPercent != nil {
		o.PriceBefore = marginOf(cost)
		o.PriceAfter = AfterDiscount(o.PriceBefore, *o.DiscountPercent)
	}
	return &o, nil
}

// ══════════════════════════════════════════════════════════════════════
// الحالُ المشتقّة — **حرفٌ واحدٌ يُشتقّ ولا يُخزَّن** (`OF`، ٢٠٢٦-٠٩-١٥)
// ══════════════════════════════════════════════════════════════════════
//
// **وعمودُ حالٍ مخزَّنٌ يفترق عن الحقيقة لحظةَ تمضي النهاية**: **صفٌّ
// مكتوبٌ فيه «سارٍ» وقد انتهى أمسِ** — **إلّا أن يمرّ عليه مُجدوِلٌ
// يصحّحه، فتصير سلامةُ السعر معلّقةً بمهمّةٍ مؤجّلة.**
//
// **والحقولُ الثلاثةُ تكفي** — `active` و`starts_at` و`ends_at`:
// **ووقتُ الخادم هو الحَكَم** (`now()`)، **ولا سلطانَ لساعة الجهاز.**
//
//	انتهت مدّتُه            ⇒ EXPIRED   — **ولا يعود**
//	أُنزل قبل نهايته        ⇒ STOPPED
//	لم يبدأ بعد            ⇒ SCHEDULED
//	وما سوى ذلك            ⇒ ACTIVE
//
// **والانتهاءُ يسبق الإنزال في القراءة**: **ومن أنزل عرضاً بعد انتهائه
// لم يُنزله — انتهى وحدَه.**
const (
	StatusScheduled = "scheduled"
	StatusActive    = "active"
	StatusStopped   = "stopped"
	StatusExpired   = "expired"
)

// StatusAt الحالُ عند لحظةٍ بعينها — **والزمنُ يُمرَّر ليُقاس.**
//
// **ودالّةٌ خالصةٌ تُقاس في آلةٍ بلا قاعدة** — **وحارسٌ يبني صفَّ
// بياناتٍ بيده لا يقيس قراراً** (درسُ الدفعة السادسة).
func StatusAt(active bool, startsAt, endsAt *time.Time, now time.Time) string {
	if endsAt != nil && !endsAt.After(now) {
		return StatusExpired
	}
	if !active {
		return StatusStopped
	}
	if startsAt != nil && startsAt.After(now) {
		return StatusScheduled
	}
	return StatusActive
}

// LiveAt **أيُسعَّر به الآن؟** — **والحالُ العاملةُ واحدةٌ لا اثنتان.**
//
// **وهي شرطُ `LiveCond` نفسُه مكتوباً في غُو** — **ومن أراد أن يتأكّد
// أنّهما لا يفترقان فليقرأ `TestOF_LiveMatchesStatus`.**
func LiveAt(active bool, startsAt, endsAt *time.Time, now time.Time) bool {
	return StatusAt(active, startsAt, endsAt, now) == StatusActive
}

// AfterDiscount السعرُ بعد الخصم — **ولا يُقرَّب.**
//
// **وهو المصدرُ الواحد**: تقرؤه الشاشةُ ويقرؤه بناءُ الطلب. **وحسبةٌ في
// موضعين تفترق يوماً** فيرى سعراً ويُحاسَب بآخر.
func AfterDiscount(price int64, percent int) int64 {
	if percent <= 0 || price <= 0 {
		return price
	}
	return price - price*int64(percent)/100
}

// List العروضُ كلُّها — للإدارة. و`liveOnly` للزبون.
func (s *Service) List(ctx context.Context, liveOnly bool, marginOf func(int64) int64) ([]Offer, error) {
	q := offerSelect
	if liveOnly {
		// **والخصمُ على صنفٍ غائبٍ أو غيرِ متاحٍ لا يُعرض** — يفتحه الزبونُ
		// فلا يجده، **ووعدٌ لا يُوفى أسوأُ من صمت.**
		q += ` WHERE ` + LiveCond + `
			AND mi.id IS NOT NULL AND mi.available`
	}
	q += ` ORDER BY o.created_at DESC`
	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Offer{}
	for rows.Next() {
		o, err := scan(rows, marginOf)
		if err != nil {
			return nil, err
		}
		out = append(out, *o)
	}
	return out, rows.Err()
}

// Input ما يُرسله من ينشئ عرضاً.
type Input struct {
	Kind            string     `json:"kind"`
	Title           string     `json:"title"`
	Body            string     `json:"body"`
	MediaID         *string    `json:"media_id"`
	Href            string     `json:"href"`
	MenuItemID      *string    `json:"menu_item_id"`
	DiscountPercent *int       `json:"discount_percent"`
	BorneBy         *string    `json:"borne_by"`
	StartsAt        *time.Time `json:"starts_at"`
	EndsAt          *time.Time `json:"ends_at"`
	Active          *bool      `json:"active"`
}

// Create ينشئ عرضاً — **والتحقّقُ هنا لا في الشاشة.**
func (s *Service) Create(ctx context.Context, actorID string, in Input,
	marginOf func(int64) int64) (*Offer, error) {
	// **والنوعُ يُفترض ولا يُسأل** — لم يبقَ إلّا واحد.
	in.Kind = KindDiscount
	if strings.TrimSpace(in.Title) == "" {
		return nil, ErrNeedsTitle
	}
	if in.MenuItemID == nil || in.DiscountPercent == nil ||
		*in.DiscountPercent < 1 || *in.DiscountPercent > 90 ||
		in.BorneBy == nil || (*in.BorneBy != ByPlatform && *in.BorneBy != ByMerchant) {
		return nil, ErrBadDiscount
	}

	// **والمدّةُ تُفحص في الخادم** — **ولا يُقبَل ما لا يقع.**
	//
	// **ويُقبَل الفارغُ**: **بلا بدايةٍ يعني «من الآن»، وبلا نهايةٍ
	// «بلا حدّ»** — وهو عقدُ الجدول منذ ٠٠٧٤.
	if in.StartsAt != nil && in.EndsAt != nil && !in.EndsAt.After(*in.StartsAt) {
		return nil, ErrBadWindow
	}
	// **ونهايةٌ مضت حينَ يُنشأ عرضٌ جديد** — **وُلد منتهياً.**
	if in.EndsAt != nil && !in.EndsAt.After(time.Now()) {
		return nil, ErrBadWindow
	}

	active := true
	if in.Active != nil {
		active = *in.Active
	}
	var id string
	err := s.db.QueryRow(ctx, `
		INSERT INTO offers (kind, title, body, media_id, href, menu_item_id,
		                    discount_percent, borne_by, starts_at, ends_at, active, created_by)
		VALUES ($1, $2, $3, NULLIF($4,'')::uuid, $5, NULLIF($6,'')::uuid,
		        $7, $8, $9, $10, $11, $12)
		RETURNING id::text`,
		in.Kind, strings.TrimSpace(in.Title), in.Body, ptrStr(in.MediaID), in.Href,
		ptrStr(in.MenuItemID), in.DiscountPercent, in.BorneBy,
		in.StartsAt, in.EndsAt, active, actorID).Scan(&id)
	if err != nil && strings.Contains(err.Error(), "offers_one_live_per_item") {
		return nil, ErrItemHasOffer
	}
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id, marginOf)
}

// SetActive يرفع العرضَ أو ينزله.
//
// **ولا حذف**: عرضٌ حُذف لا يُقرأ في تقريرٍ لاحق — **ومن سأل «كم خسرنا على
// عروض رمضان؟» لم يجد ما يقرؤه.**
func (s *Service) SetActive(ctx context.Context, id string, active bool,
	marginOf func(int64) int64) (*Offer, error) {
	_, err := s.db.Exec(ctx,
		`UPDATE offers SET active = $2, updated_at = now() WHERE id = $1`, id, active)
	if err != nil && strings.Contains(err.Error(), "offers_one_live_per_item") {
		return nil, ErrItemHasOffer
	}
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id, marginOf)
}

func (s *Service) Get(ctx context.Context, id string, marginOf func(int64) int64) (*Offer, error) {
	return scan(s.db.QueryRow(ctx, offerSelect+` WHERE o.id = $1`, id), marginOf)
}

// ══════════════════════════════════════════════════════════════════════
// **بابُ صاحب المتجر ومندوبِه — على المحرّك نفسِه** (`OF`، ٢٠٢٦-٠٩-١٥)
// ══════════════════════════════════════════════════════════════════════
//
// **ولا محرّكَ ثانٍ**: **الجدولُ نفسُه، و`LiveCond` نفسُه، و`LiveDiscount`
// نفسُها التي يناديها بناءُ الطلب.** **ومحرّكٌ ثانٍ للعروض يعني سعرين.**
//
// **والفرقُ كلُّه في النطاق**: **من يملك الصنفَ يملك عرضَه.**

// ErrNotYours **صنفٌ ليس في قائمة من يطلب.**
//
// **ويُردّ كما يُردّ الغائب** — **ولا يُقال «هذا الصنفُ لمتجرٍ آخر»**:
// **وجوابٌ يفرّق بين «ليس لك» و«لا وجود له» يُعدّ المعرّفاتِ عدّاً.**
var ErrNotYours = httpx.NewError(http.StatusForbidden, "forbidden", "errors.forbidden")

// ListForMerchant عروضُ متجرٍ بعينه — **كلُّها، بحالها المشتقّة.**
//
// **ولا يُقرأ منها متجرٌ آخر**: **الشرطُ على `mi.merchant_id` لا على ما
// يرسله الجهاز.**
func (s *Service) ListForMerchant(ctx context.Context, merchantID string,
	marginOf func(int64) int64) ([]Offer, error) {
	rows, err := s.db.Query(ctx, offerSelect+`
		WHERE mi.merchant_id = $1
		ORDER BY o.created_at DESC`, merchantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Offer{}
	for rows.Next() {
		o, err := scan(rows, marginOf)
		if err != nil {
			return nil, err
		}
		out = append(out, *o)
	}
	return out, rows.Err()
}

// OwnerOf **متجرُ العرض** — **يُسأل قبل كلّ فعلٍ على عرضٍ بمعرّفه.**
//
// **ومن أنزل عرضاً بمعرّفه وحدَه بلا هذا السؤال أنزل عرضَ أيّ متجر** —
// **والمعرّفاتُ تُقرأ من ردٍّ سابقٍ أو تُخمَّن** (وهو درسُ `repOwnsItem`).
func (s *Service) OwnerOf(ctx context.Context, offerID string) (string, error) {
	var merchantID *string
	err := s.db.QueryRow(ctx, `
		SELECT mi.merchant_id::text FROM offers o
		JOIN menu_items mi ON mi.id = o.menu_item_id
		WHERE o.id = $1`, offerID).Scan(&merchantID)
	if err != nil || merchantID == nil {
		return "", ErrNotYours
	}
	return *merchantID, nil
}

// CreateScoped ينشئ عرضاً **في نطاق متجرٍ مأذونٍ فيه سلفاً.**
//
// # ولا يُصدَّق معرّفُ المتجر من الحمولة
//
// **والنطاقُ يجيء من المسار المحروس** — **والصنفُ يُسأل: أهو في هذا
// المتجر؟** **ومن صدّق معرّفاً في الجسد فتح قائمةَ كلّ متجر.**
//
// # ومن يتحمّل الخصمَ ليس خياراً له
//
// **وصاحبُ المتجر لا يقرّر أن تتحمّله المنصّة** — **وإلّا أنفق من
// هامشِ غيره بضغطة.** **والقرارُ بيد الإدارة وحدَها كما كان** (الهجرة
// ٠٠٧٤: «أقرّر لكلّ عرضٍ على حدة»).
//
// # والمنتهي الباقي على `active` يُطوى
//
// **والفهرسُ يمنع عرضين فاعلين على صنف** — **ومنتهٍ لم يُنزَل يشغل
// الموضعَ وهو لا يُسعَّر به.** **فيُطوى في المعاملة نفسِها**: **ولا
// يُحذف** (تقريرُ «كم خسرنا على عروض رمضان؟»).
func (s *Service) CreateScoped(ctx context.Context, actorID, merchantID string, in Input,
	marginOf func(int64) int64) (*Offer, error) {
	if in.MenuItemID == nil || strings.TrimSpace(*in.MenuItemID) == "" {
		return nil, ErrBadDiscount
	}
	var ok bool
	if err := s.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM menu_items WHERE id = $1 AND merchant_id = $2)`,
		*in.MenuItemID, merchantID).Scan(&ok); err != nil || !ok {
		return nil, ErrNotYours
	}
	// **ويتحمّله المتجرُ حتماً** — **ولا يُقرأ ما أرسله الجهاز.**
	borne := ByMerchant
	in.BorneBy = &borne
	if _, err := s.db.Exec(ctx, `
		UPDATE offers SET active = false, updated_at = now()
		WHERE menu_item_id = $1 AND active AND ends_at IS NOT NULL AND ends_at <= now()`,
		*in.MenuItemID); err != nil {
		return nil, err
	}
	return s.Create(ctx, actorID, in, marginOf)
}

// LiveDiscount خصمُ صنفٍ سارٍ الآن — **يُنادى لحظةَ بناء الطلب.**
//
// **ويُقرأ من القاعدة لا من ذاكرةٍ محمّلة**: عرضٌ يُنزَل وطلبٌ يُبنى في اللحظة
// نفسِها — **والذاكرةُ تُعطي سعراً انتهى.**
func (s *Service) LiveDiscount(ctx context.Context, q dbtx.Querier, menuItemID string) (percent int, borneBy string) {
	var p *int
	var b *string
	_ = q.QueryRow(ctx, `
		SELECT o.discount_percent, o.borne_by FROM offers o
		WHERE o.menu_item_id = $1 AND o.kind = 'discount' AND `+LiveCond,
		menuItemID).Scan(&p, &b)
	if p == nil || b == nil {
		return 0, ""
	}
	return *p, *b
}

func ptrStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
