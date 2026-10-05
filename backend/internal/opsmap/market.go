package opsmap

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

// ══════════════════════════════════════════════════════════════════════
// **المتاجرُ والطلباتُ النشطة — `MAP-2`**
// ══════════════════════════════════════════════════════════════════════
//
// # ولا نصَّ «مفتوحٌ الآن» ثانٍ
//
// **`orders.OpenNowSQL` هو مصدرُ الحقيقة** — يستعمله العرضُ وإنشاءُ
// الطلب. **ولو نُسخ هنا لَافترقا يوماً**: الخريطةُ تقول مفتوحٌ والزبونُ
// يُردّ، **أو العكس** — وهي عينُ العلّة التي أُغلقت بتصدير هذا النصّ.

// Merchant متجرٌ على الخريطة.
//
// **ولا دَين، ولا هاتفَ إلّا لمن يملك `users.contact.read`** (البند ٣٣) —
// **معلوماتٌ تشغيليّةٌ مختصرة** (البند ٩)، ومن أراد الملفَّ فتحه.
type Merchant struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	Status   string  `json:"status"`
	OpenNow  bool    `json:"open_now"`
	CityName *string `json:"city,omitempty"`
	AreaText *string `json:"area,omitempty"`
	ActiveN  int     `json:"active_orders"`

	// RepName **مندوبُه** — **ولمن يملك `VIEW_REP_ACTIVITY` وحدَه**
	// (البند ٩).
	RepName *string `json:"rep,omitempty"`

	// Phone **هاتفُ المتجر — لمن يملك `users.contact.read` وحدَه** (زرُّ
	// الاتّصال في بطاقته).
	Phone *string `json:"phone,omitempty"`
}

// MerchantFilter مُرشِّحاتُ طبقة المتاجر.
type MerchantFilter struct {
	CityID  string
	Status  string
	OpenNow *bool
	// HasActive متجرٌ عليه طلبٌ جارٍ.
	HasActive *bool
	Search    string

	// WithPhone **أيُرسَل هاتفُ المتجر؟** — خيارُ ردٍّ بقدرة `users.contact.read`.
	WithPhone bool
}

// Merchants متاجرُ المشهد.
func Merchants(ctx context.Context, q Querier, box *BBox, f MerchantFilter,
	withRep bool, limit int) ([]Merchant, error) {

	var where []string
	var args []any
	add := func(cond string, vals ...any) {
		where = append(where, cond)
		args = append(args, vals...)
	}

	// **ومتجرٌ بلا دبّوسٍ لا يُرسَم** — ولا يُخترَع له موضع.
	add("m.location IS NOT NULL")
	if box != nil {
		if !box.Valid() {
			return nil, fmt.Errorf("مستطيلُ مشهدٍ غيرُ معقول")
		}
		add(box.SQL("m.location", len(args)+1), box.Args()...)
	}
	if f.Status != "" {
		add(fmt.Sprintf("m.status = $%d", len(args)+1), f.Status)
	}
	if f.CityID != "" {
		add(fmt.Sprintf("m.city_id = $%d", len(args)+1), f.CityID)
	}
	if f.Search != "" {
		add(fmt.Sprintf("m.name ILIKE $%d", len(args)+1), "%"+f.Search+"%")
	}
	if f.OpenNow != nil {
		if *f.OpenNow {
			add(orders.OpenNowSQL)
		} else {
			add("NOT " + orders.OpenNowSQL)
		}
	}
	if f.HasActive != nil {
		cond := "EXISTS (SELECT 1 FROM orders o WHERE o.merchant_id = m.id AND o.closed_at IS NULL)"
		if !*f.HasActive {
			cond = "NOT " + cond
		}
		add(cond)
	}
	if limit <= 0 || limit > 3000 {
		limit = 3000
	}

	sql := `
		SELECT m.id::text, m.name, m.status,
		       ST_Y(m.location::geometry), ST_X(m.location::geometry),
		       ` + orders.OpenNowSQL + `,
		       c.name, m.address_text,
		       (SELECT count(*) FROM orders o
		         WHERE o.merchant_id = m.id AND o.closed_at IS NULL),
		       rep.full_name, m.phone::text
		FROM merchants m
		LEFT JOIN cities c ON c.id = m.city_id
		LEFT JOIN users rep ON rep.id = m.sales_rep_user_id
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY m.name
		LIMIT ` + fmt.Sprint(limit)

	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Merchant{}
	for rows.Next() {
		var x Merchant
		var rep *string
		var phone string
		if err := rows.Scan(&x.ID, &x.Name, &x.Status, &x.Lat, &x.Lng,
			&x.OpenNow, &x.CityName, &x.AreaText, &x.ActiveN, &rep, &phone); err != nil {
			return nil, err
		}
		if f.WithPhone && phone != "" {
			x.Phone = &phone
		}
		if withRep {
			x.RepName = rep
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// ══════════════════════════════════════════════════════════════════════
// **الطلباتُ النشطة**
// ══════════════════════════════════════════════════════════════════════

// Order طلبٌ حيٌّ على الخريطة.
//
// # ولماذا نقطةُ التسليم لا عنوانُ الزبون
//
// **الخريطةُ تحتاج «أين يذهب»** — **ولا تحتاج اسمَ الزبون ولا هاتفَه**
// (البند ٣٣). **والعنوانُ النصّيُّ يُرسَل لمن يملك رؤيةَ الطلبات**،
// وهو ما تراه صفحةُ الطلبات أصلاً.
type Order struct {
	ID     string `json:"id"`
	Number int64  `json:"number"`
	Status string `json:"status"`
	Kind   string `json:"kind"`

	// Drop **نقطةُ التسليم** — وهي موضعُ الطلب على الخريطة.
	DropLat float64 `json:"drop_lat"`
	DropLng float64 `json:"drop_lng"`
	Address *string `json:"address,omitempty"`

	MerchantID   *string  `json:"merchant_id,omitempty"`
	MerchantName *string  `json:"merchant,omitempty"`
	PickLat      *float64 `json:"pick_lat,omitempty"`
	PickLng      *float64 `json:"pick_lng,omitempty"`

	DriverID   *string  `json:"driver_id,omitempty"`
	DriverName *string  `json:"driver,omitempty"`
	DriverLat  *float64 `json:"driver_lat,omitempty"`
	DriverLng  *float64 `json:"driver_lng,omitempty"`

	CreatedAt  time.Time  `json:"created_at"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	PickedUpAt *time.Time `json:"picked_up_at,omitempty"`

	// Money **الإجمالُ وطريقةُ الدفع** — **لمن يملك صلاحيّتَه وحدَه**
	// (البند ١٠).
	Total         *int64  `json:"total,omitempty"`
	PaymentMethod *string `json:"payment_method,omitempty"`
	DeliveryFee   *int64  `json:"delivery_fee,omitempty"`

	// StuckReason **سببُ العلوق أو غيابُه** — بشرط لوحة الطلبات
	// (`orders.StuckReasonSQL`): `emergency` · `no_accept` · `not_sent` ·
	// `no_driver` · `too_long`. **والأحمرُ على الخريطة منه وحدَه.**
	StuckReason *string `json:"stuck_reason,omitempty"`

	// CustomerName **اسمُ الزبون — لمن يملك `orders.customer_details.read`
	// وحدَه.** ولا هاتفَ هنا لأحد.
	CustomerName *string `json:"customer_name,omitempty"`
}

// OrderFilter مُرشِّحاتُ طبقة الطلبات.
type OrderFilter struct {
	Status string
	// Unassigned طلبٌ بلا سائق.
	Unassigned *bool
	MerchantID string
	DriverID   string
	// Search رقمُ طلبٍ أو جزءٌ منه.
	Search string

	// Stuck **مهلُ العلوق** — وفارغُها لا حكمَ علوق (خيارُ ردّ لا مرشِّح).
	Stuck *orders.StuckLimits
	// WithCustomer **أيُرسَل اسمُ الزبون؟** — بقدرة `orders.customer_details.read`.
	WithCustomer bool
}

// ActiveOrders طلباتُ المشهد الحيّة.
//
// **والنشطُ افتراضاً** (البند ١٠) — `closed_at IS NULL`، **وهو تعريفُ
// المحرّك نفسِه** لا تعريفٌ ثانٍ.
func ActiveOrders(ctx context.Context, q Querier, box *BBox, f OrderFilter,
	withMoney bool, limit int) ([]Order, error) {

	var where []string
	var args []any
	add := func(cond string, vals ...any) {
		where = append(where, cond)
		args = append(args, vals...)
	}

	add("o.closed_at IS NULL")
	if box != nil {
		if !box.Valid() {
			return nil, fmt.Errorf("مستطيلُ مشهدٍ غيرُ معقول")
		}
		add(box.SQL("o.dropoff", len(args)+1), box.Args()...)
	}
	if f.Status != "" {
		add(fmt.Sprintf("o.status = $%d", len(args)+1), f.Status)
	}
	if f.MerchantID != "" {
		add(fmt.Sprintf("o.merchant_id = $%d", len(args)+1), f.MerchantID)
	}
	if f.DriverID != "" {
		add(fmt.Sprintf("o.driver_id = $%d", len(args)+1), f.DriverID)
	}
	if f.Unassigned != nil {
		if *f.Unassigned {
			add("o.driver_id IS NULL")
		} else {
			add("o.driver_id IS NOT NULL")
		}
	}
	if f.Search != "" {
		add(fmt.Sprintf("o.number::text ILIKE $%d", len(args)+1), "%"+f.Search+"%")
	}
	if limit <= 0 || limit > 2000 {
		limit = 2000
	}

	sql := `
		SELECT o.id::text, o.number, o.status, o.kind,
		       ST_Y(o.dropoff::geometry), ST_X(o.dropoff::geometry), o.address_text,
		       m.id::text, m.name,
		       ST_Y(COALESCE(o.pickup_override, m.location)::geometry),
		       ST_X(COALESCE(o.pickup_override, m.location)::geometry),
		       d.id::text, d.full_name,
		       ST_Y(d.last_location::geometry), ST_X(d.last_location::geometry),
		       o.created_at, o.accepted_at, o.picked_up_at,
		       o.total, o.payment_method, o.delivery_fee,
		       ` + stuckColumn(f.Stuck) + `, cu.full_name
		FROM orders o
		LEFT JOIN merchants m ON m.id = o.merchant_id
		LEFT JOIN users d ON d.id = o.driver_id
		LEFT JOIN users cu ON cu.id = o.customer_id
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY o.created_at DESC
		LIMIT ` + fmt.Sprint(limit)

	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Order{}
	for rows.Next() {
		var x Order
		var total, fee *int64
		var pay, cust *string
		if err := rows.Scan(&x.ID, &x.Number, &x.Status, &x.Kind,
			&x.DropLat, &x.DropLng, &x.Address,
			&x.MerchantID, &x.MerchantName, &x.PickLat, &x.PickLng,
			&x.DriverID, &x.DriverName, &x.DriverLat, &x.DriverLng,
			&x.CreatedAt, &x.AcceptedAt, &x.PickedUpAt,
			&total, &pay, &fee, &x.StuckReason, &cust); err != nil {
			return nil, err
		}
		if f.WithCustomer {
			x.CustomerName = cust
		}
		if withMoney {
			x.Total, x.PaymentMethod, x.DeliveryFee = total, pay, fee
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
