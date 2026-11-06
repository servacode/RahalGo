package server

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/pricing"
)

// تفاصيل عميل المندوب — شفافية العمولة.
//
// المندوب يقبض نسبةً من طلبات متجرٍ جلبه، ولم يكن يرى **من أين** جاءت: رقمٌ
// مجمَّع في لوحته وكفى. وثقةُ من يعمل بالعمولة تُبنى على أن يُراجع بنفسه، لا
// على أن يُصدّق. فهنا كل طلب: رقمه، قيمته، حالته، ونصيبه منه.
//
// **حدود ما يراه مقصودة**: قيمة الطلب وحالته وعمولته — لا اسم الزبون ولا عنوانه
// ولا هاتفه. المندوب طرفٌ في علاقته بالمتجر لا في طلبات زبائنه، وكشف بيانات
// الزبون له توسيعٌ للاطلاع بلا حاجة تُبرّره.

func (s *Server) handleRepMerchantDetail(w http.ResponseWriter, r *http.Request) {
	uid := userIDFrom(r)
	merchantID := chi.URLParam(r, "id")
	ctx := r.Context()

	// التحقق من النسبة **أولاً وفي الخادم**: بلا هذا يقرأ أي مندوب طلبات أي
	// متجر بتبديل المعرّف في الرابط.
	var head struct {
		Name         string  `json:"name"`
		CategoryIcon string  `json:"category_icon"`
		CategoryName string  `json:"category_name"`
		LogoThumbURL *string `json:"logo_thumb_url"`
		Status       string  `json:"status"`
		JoinedAt     string  `json:"joined_at"`
		OwnerPhone   *string `json:"owner_phone"`
		// ExchangeRate سعرُ صرف المتجر — يكتبه المندوبُ وهو يجهّزه (٢٠٢٦-١٠-١٢).
		ExchangeRate *float64 `json:"exchange_rate"`
		// **نسبةُ عمولة متجره وحدّا ما يضبطه منها** (قرارُ المالك ٢٠٢٦-١١-٠٤) — لا هامشَ المنصّة (`RQ-7`).
		repCommissionView
	}
	var override *pricing.Percent2
	err := s.pg.QueryRow(ctx, `
		SELECT m.name, c.icon, c.name, lm.thumb_path, m.status,
		       m.created_at::date::text,
		       NULLIF(ou.phone::text, ''), m.exchange_rate::float8, m.commission_percent
		FROM merchants m
		JOIN categories c ON c.id = m.category_id
		LEFT JOIN users ou ON ou.id = m.owner_user_id
		LEFT JOIN media lm ON lm.id = m.logo_media_id
		WHERE m.id = $1 AND m.sales_rep_user_id = $2`, merchantID, uid).
		Scan(&head.Name, &head.CategoryIcon, &head.CategoryName, &head.LogoThumbURL,
			&head.Status, &head.JoinedAt, &head.OwnerPhone, &head.ExchangeRate, &override)
	if errors.Is(err, pgx.ErrNoRows) {
		// نفس ردّ «غير موجود» لمتجرٍ ليس له: لا نكشف وجود متاجر الآخرين
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}
	head.LogoThumbURL = media.URLForPtr(head.LogoThumbURL)
	head.repCommissionView = s.repCommissionOf(ctx, override)

	// نسبةُ المندوب وعمولةُ المتجر — **من مخزن الإعدادات لا من SQL.**
	//
	// **والعمولتان تُقرآن هنا لا في الاستعلام**: الحسبةُ في `pricing` وحدَها،
	// **ولو كُتبت في SQL لَما عرفت النمطَ المقطوع** — فتعرض للمندوب نسبةً من
	// عمولةٍ لم تُحسب بها.
	repRule := pricing.RepCommission(ctx, s.settings)

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	// الفلترة بـ**دلوٍ** لا بحالة خام. المندوب لا شأن له بحالات التشغيل الأربع
	// عشرة — «السائق في المتجر» و«جارٍ إسناد سائق» تفاصيل عمليات لا تعنيه ولا
	// يملك تغييرها. يعنيه سؤالان: أتمّ الطلب فقبضتُ عمولته؟ أم ضاع؟
	//   delivered  → مُسلَّم
	//   cancelled  → كل نهاية غير التسليم
	//   ما عداهما  → لا فلترة (الكل)
	bucket := q.Get("status")
	// قيمة لا نعرفها تُعامَل كـ«الكل» لا كفلترةٍ لا تُطابق شيئاً: جدولٌ فارغ
	// بلا سبب ظاهر يوهم المندوب أن لا طلبات لعميله.
	if bucket != "delivered" && bucket != "cancelled" {
		bucket = ""
	}
	// **وطلبٌ بعدّة محطّات فيه قسمٌ حيٌّ لهذا المتجر طلبُه أيضاً** (C8) — لا متجرُه الأوّلُ وحدَه.
	where := `WHERE ` + orders.LiveStoreOrdersSQL("$1") + ` AND ($2 = '' OR
	          ($2 = 'delivered' AND o.status = 'delivered') OR
	          ($2 = 'cancelled' AND o.status IN ('rejected','cancelled','failed','refunded')))`

	var total int
	if err := s.pg.QueryRow(ctx,
		`SELECT count(*) FROM orders o `+where, merchantID, bucket).Scan(&total); err != nil {
		s.respondErr(w, err)
		return
	}

	// عمولة المندوب عن كل طلب: قيدُ العمولة وما عُكس منه، بمرجع الطلب نفسه.
	// الجمع لا الاختيار: الطلب المُسترجَع له قيدان يلغي أحدهما الآخر فيظهر صفراً
	// — وهو الصدق بعينه، لا إخفاءَ السطر ولا إظهارَ عمولةٍ سُحبت.
	rows, err := s.pg.Query(ctx, `
		-- **وفي طلبٍ بعدّة محطّات: بضاعةُ هذا المتجر وحدَها** (C8) — لا مجموعُ الطلب ولا توصيلُه،
		-- **وهامشُ بنوده هو، ونصيبُه عنه.**
		SELECT o.number, o.status, o.cancel_reason,
		       CASE WHEN o.stops > 1 THEN `+orders.StoreGoodsSQL("$1")+` ELSE o.total END,
		       CASE WHEN o.stops > 1 THEN `+orders.StoreGoodsSQL("$1")+` ELSE o.subtotal END,
		       CASE WHEN o.stops > 1 THEN 0 ELSE o.delivery_fee END,
		       -- **الهامشُ لا عمولةُ المتاجر.**
		       --
		       -- **ونصيبُ المندوب نسبةٌ منه** — فعرضُ عمولةِ المتاجر بجانبه
		       -- يجعل الرقمين لا يتّسقان: يرى «٢٬٦٠٠» و«٥٠٠» فيحسب فلا يخرج
		       -- ١٠٪. **ورقمان متجاوران لا تربطهما حسبةٌ يُفقدان الثقةَ في
		       -- الاثنين.**
		       --
		       -- (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «يجب أن تجلب هامشَ الربح بدل عمولة
		       -- المنصة، والاسمُ يبقى عمولة المنصة».)
		       CASE WHEN o.stops > 1 THEN `+orders.StoreMarginSQL("$1")+`
		            ELSE `+orders.OrderMarginSQL("o.id")+` END, o.created_at, o.delivered_at,
		       -- **والمشطوبُ ما كان سيُحتسب لولا الإلغاء.**
		       --
		       -- الدفترُ يُصفّر عمولةَ الطلب الملغى — وهو الصوابُ المحاسبيّ،
		       -- **لكنّ عرضَ صفرٍ للمندوب يخفي عنه حجمَ ما ضاع.** فيُحسب هنا
		       -- للعرض وحدَه، **ويقوله الشطبُ صراحةً: رقمٌ كان ولم يصر.**
		       --
		       -- **وهو الهامشُ نفسُه** — فبنودُ الطلب باقيةٌ ولو أُلغي.
		       `+orders.RepStoreShareSQL("$1", "$3")+`
		FROM orders o `+where+`
		ORDER BY o.number DESC LIMIT $4 OFFSET $5`,
		merchantID, bucket, uid, perPage, (page-1)*perPage)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	type repOrder struct {
		Number int64  `json:"number"`
		Status string `json:"status"`
		// سبب الإلغاء يُرسل مع الطلب الملغى: «ضاع طلبٌ» سؤال ناقص بلا جوابه
		CancelReason string `json:"cancel_reason"`
		Total        int64  `json:"total"`
		Subtotal     int64  `json:"subtotal"`
		DeliveryFee  int64  `json:"delivery_fee"`
		// **هامشُ المنصّة داخليٌّ لا يُكشف للمندوب** (RQ-7، قرارُ المالك ٢٠٢٦-٠٩-٢٦):
		// **يبقى محسوباً في الخادم** (منه يُشتقّ نصيبُ المندوب المشطوب أدناه)،
		// **ولا يُسلسَل في الجواب** (`json:"-"`). فيرى المندوبُ نصيبَه لا هامشَنا.
		Commission int64 `json:"-"`
		// ما كان سيُحتسب لولا الإلغاء — يبقى داخليّاً؛ المندوبُ يرى نصيبَه المشطوبَ وحدَه.
		Forfeited      int64      `json:"-"`
		ForfeitedShare int64      `json:"forfeited_share"`
		MyShare        int64      `json:"my_share"`
		CreatedAt      time.Time  `json:"created_at"`
		DeliveredAt    *time.Time `json:"delivered_at"`
	}
	list := []repOrder{}
	for rows.Next() {
		var o repOrder
		if err := rows.Scan(&o.Number, &o.Status, &o.CancelReason, &o.Total, &o.Subtotal, &o.DeliveryFee,
			&o.Commission, &o.CreatedAt, &o.DeliveredAt, &o.MyShare); err != nil {
			s.respondErr(w, err)
			return
		}
		// **والمشطوبُ هو الهامشُ نفسُه** — بنودُ الطلب باقيةٌ ولو أُلغي،
		// **فما كان سيُحتسب يُقرأ من مصدره لا من حسبةٍ ثانية.**
		o.Forfeited = o.Commission
		o.ForfeitedShare = repRule.Of(o.Forfeited)
		list = append(list, o)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}

	// الملخّص على **كل** طلبات المتجر لا على الصفحة المعروضة: ملخّصٌ يتغيّر
	// بتقليب الصفحات ليس ملخّصاً.
	var sum struct {
		Orders     int   `json:"orders"`
		Delivered  int   `json:"delivered"`
		Cancelled  int   `json:"cancelled"`
		Sales      int64 `json:"delivered_sales"`
		MyEarnings int64 `json:"my_earnings"`
	}
	if err := s.pg.QueryRow(ctx, `
		SELECT count(*),
		       count(*) FILTER (WHERE o.status = 'delivered'),
		       count(*) FILTER (WHERE o.status IN ('rejected','cancelled','failed','refunded')),
		       COALESCE(sum(CASE WHEN o.stops > 1 THEN `+orders.StoreGoodsSQL("$1")+` ELSE o.total END)
		                FILTER (WHERE o.status = 'delivered'), 0),
		       COALESCE(sum(`+orders.RepStoreShareSQL("$1", "$2")+`), 0)
		FROM orders o WHERE `+orders.LiveStoreOrdersSQL("$1"), merchantID, uid).
		Scan(&sum.Orders, &sum.Delivered, &sum.Cancelled, &sum.Sales, &sum.MyEarnings); err != nil {
		s.respondErr(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"merchant": head, "summary": sum, "orders": list,
		"total": total, "page": page, "per_page": perPage,
	})
}
