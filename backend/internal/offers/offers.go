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
)

var (
	ErrBadKind    = httpx.NewError(http.StatusBadRequest, "bad_offer_kind", "errors.validation")
	ErrNeedsTitle = httpx.NewError(http.StatusBadRequest, "offer_needs_title", "errors.validation")
	// ErrBadDiscount خصمٌ ناقص — صنفٌ ونسبةٌ ومن يتحمّل، ثلاثةٌ معاً.
	ErrBadDiscount = httpx.NewError(http.StatusBadRequest, "bad_offer_discount", "errors.validation")
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
	Live      bool      `json:"live"`
	CreatedAt time.Time `json:"created_at"`
}

// liveCond شرطُ السريان — **تعبيرٌ واحدٌ يُعاد استعماله.**
//
// **ومكتوبٌ مرّةً**: الشاشةُ تقرؤه والزبونُ يقرؤه والمحرّكُ يقرؤه —
// **وثلاثُ نسخٍ تفترق فيُعرض ما لا يُطبَّق.**
const liveCond = `(o.active
	AND (o.starts_at IS NULL OR o.starts_at <= now())
	AND (o.ends_at IS NULL OR o.ends_at > now()))`

const offerSelect = `
	SELECT o.id::text, o.kind, o.title, o.body,
	       o.media_id::text, mm.path, o.href,
	       o.menu_item_id::text, COALESCE(mi.name, ''), COALESCE(mr.name, ''),
	       mr.id::text, im.path,
	       COALESCE(mi.price, 0),
	       o.discount_percent, o.borne_by,
	       o.starts_at, o.ends_at, o.active, ` + liveCond + `, o.created_at
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
		&o.StartsAt, &o.EndsAt, &o.Active, &o.Live, &o.CreatedAt); err != nil {
		return nil, err
	}
	o.ImageURL = media.URLForPtr(o.ImageURL)
	o.ItemImageURL = media.URLForPtr(o.ItemImageURL)
	if o.DiscountPercent != nil {
		o.PriceBefore = marginOf(cost)
		o.PriceAfter = AfterDiscount(o.PriceBefore, *o.DiscountPercent)
	}
	return &o, nil
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
		q += ` WHERE ` + liveCond + `
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

// LiveDiscount خصمُ صنفٍ سارٍ الآن — **يُنادى لحظةَ بناء الطلب.**
//
// **ويُقرأ من القاعدة لا من ذاكرةٍ محمّلة**: عرضٌ يُنزَل وطلبٌ يُبنى في اللحظة
// نفسِها — **والذاكرةُ تُعطي سعراً انتهى.**
func (s *Service) LiveDiscount(ctx context.Context, menuItemID string) (percent int, borneBy string) {
	var p *int
	var b *string
	_ = s.db.QueryRow(ctx, `
		SELECT o.discount_percent, o.borne_by FROM offers o
		WHERE o.menu_item_id = $1 AND o.kind = 'discount' AND `+liveCond,
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
