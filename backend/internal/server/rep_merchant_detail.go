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
	}
	err := s.pg.QueryRow(ctx, `
		SELECT m.name, c.icon, c.name, lm.thumb_path, m.status,
		       m.created_at::date::text,
		       NULLIF(COALESCE(ou.whatsapp_phone::text, ou.phone::text), '')
		FROM merchants m
		JOIN categories c ON c.id = m.category_id
		LEFT JOIN users ou ON ou.id = m.owner_user_id
		LEFT JOIN media lm ON lm.id = m.logo_media_id
		WHERE m.id = $1 AND m.sales_rep_user_id = $2`, merchantID, uid).
		Scan(&head.Name, &head.CategoryIcon, &head.CategoryName, &head.LogoThumbURL,
			&head.Status, &head.JoinedAt, &head.OwnerPhone)
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

	// نسبة المندوب من عمولة المنصة — تُقرأ مرّة لحساب المشطوب المعروض
	var repPct float64
	if err := s.pg.QueryRow(ctx, `
		SELECT COALESCE((SELECT (value#>>'{}')::float8 FROM app_settings
		                 WHERE key = 'sales.commission_percent'), 10)`).Scan(&repPct); err != nil {
		s.respondErr(w, err)
		return
	}

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
	where := `WHERE o.merchant_id = $1 AND ($2 = '' OR
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
		SELECT o.number, o.status, o.cancel_reason, o.total, o.subtotal, o.delivery_fee,
		       o.platform_commission, o.created_at, o.delivered_at,
		       -- **العمولة المشطوبة**: ما كان سيُحتسب لولا الإلغاء.
		       -- الدفتر يُصفّر عمولة الطلب الملغى — وهو الصواب المحاسبي، لكن عرض
		       -- صفرٍ للمندوب يخفي عنه حجم ما ضاع. فنحسبها هنا للعرض وحده،
		       -- ويقولها الشطب صراحةً: رقمٌ كان ولم يصر.
		       (o.subtotal * mm.commission_percent / 100),
		       COALESCE((SELECT sum(t.amount) FROM wallet_transactions t
		                 WHERE t.user_id = $3 AND t.ref = o.id::text
		                   AND t.kind IN ('commission', 'adjustment')), 0)
		FROM orders o JOIN merchants mm ON mm.id = o.merchant_id `+where+`
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
		Commission   int64  `json:"platform_commission"`
		// ما كان سيُحتسب لولا الإلغاء — للعرض مشطوباً لا للحساب
		Forfeited      int64      `json:"forfeited_commission"`
		ForfeitedShare int64      `json:"forfeited_share"`
		MyShare        int64      `json:"my_share"`
		CreatedAt      time.Time  `json:"created_at"`
		DeliveredAt    *time.Time `json:"delivered_at"`
	}
	list := []repOrder{}
	for rows.Next() {
		var o repOrder
		if err := rows.Scan(&o.Number, &o.Status, &o.CancelReason, &o.Total, &o.Subtotal, &o.DeliveryFee,
			&o.Commission, &o.CreatedAt, &o.DeliveredAt, &o.Forfeited, &o.MyShare); err != nil {
			s.respondErr(w, err)
			return
		}
		// نصيب المندوب من المشطوبة — بالنسبة نفسها المعتمدة وقت العرض
		o.ForfeitedShare = o.Forfeited * int64(repPct) / 100
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
		       COALESCE(sum(o.total) FILTER (WHERE o.status = 'delivered'), 0),
		       COALESCE((SELECT sum(t.amount) FROM wallet_transactions t
		                 JOIN orders o2 ON t.ref <> '' AND o2.id::text = t.ref
		                 WHERE t.user_id = $2 AND o2.merchant_id = $1
		                   AND t.kind IN ('commission', 'adjustment')), 0)
		FROM orders o WHERE o.merchant_id = $1`, merchantID, uid).
		Scan(&sum.Orders, &sum.Delivered, &sum.Cancelled, &sum.Sales, &sum.MyEarnings); err != nil {
		s.respondErr(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"merchant": head, "summary": sum, "orders": list,
		"total": total, "page": page, "per_page": perPage,
	})
}
