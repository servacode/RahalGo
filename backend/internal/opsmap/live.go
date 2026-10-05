package opsmap

import (
	"context"
	"fmt"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

// ══════════════════════════════════════════════════════════════════════
// **خريطةُ العمليّات الاحترافيّة — قرارُ المالك ٢٠٢٦-١٠-٠٥**
// ══════════════════════════════════════════════════════════════════════
//
// شريطٌ علويٌّ بمفاتيحَ وعدّادات، **ولونٌ واحدٌ لكلّ نوعٍ وحالٍ يُحسب هنا مرّةً**
// فتقرؤه الخريطةُ والشريطُ والبطاقةُ بحكمٍ واحد:
//
//	السائق   متاح (أخضر) · معه طلب (برتقاليّ) · خامل (رماديّ)
//	الطلب    ماشٍ (أزرق) · عالق (أحمر — بشرط لوحة الطلبات نفسِه)
//	المتجر   مفتوح (بنفسجيّ) · مغلق (رماديّ)
//	الزبائن  خلايا مجمَّعة (ورديّ) — لا بيتَ ولا اسمَ ولا هاتف
//	المكتب   شعارُ المنصّة في موضعها، ومن في اللوحة الآن

// حالُ السائق على الخريطة — **والحكمُ في الخادم لا في الشاشة.**
const (
	// DriverAvailable **متاح**: على الدوام، بلا طلبٍ مفتوح، وموضعُه صالحٌ للتوزيع.
	DriverAvailable = "available"
	// DriverBusy **معه طلبٌ مفتوح** — أيّاً كانت طزاجةُ موضعه.
	DriverBusy = "busy"
	// DriverIdle **خامل**: خارجَ الدوام أو شاخ موضعُه أو لا موضعَ له.
	DriverIdle = "idle"
)

// DriverTone **حالُ السائق في لون** — دالّةٌ صافيةٌ تُختبر بلا قاعدة.
//
// **ومعه طلبٌ يسبق كلَّ شيء**: سائقٌ يحمل طلباً وانقطع موضعُه ما زال «مشغولاً»
// — **ورسمُه رماديّاً يقول للمكتب إنّه فارغٌ وهو يحمل طعامَ أحد.**
//
// **و«متاح» بحدّ المحرّك نفسِه** (`FRESH` = ١٥ دقيقة) — **فلا تقول الخريطةُ
// «متاح» عن سائقٍ لا يوزّع عليه المحرّك.**
func DriverTone(onShift bool, activeOrders int, freshness string) string {
	if activeOrders > 0 {
		return DriverBusy
	}
	if onShift && (freshness == FreshnessLive || freshness == FreshnessFresh) {
		return DriverAvailable
	}
	return DriverIdle
}

// ══════════════════════════════════════════════════════════════════════
// **الزبائنُ خلايا لا بيوت**
// ══════════════════════════════════════════════════════════════════════
//
// # المصدر
//
// **نقطةٌ واحدةٌ لكلّ زبونٍ فعّال**: عنوانُه الافتراضيُّ المحفوظ، وإلّا نقطةُ
// تسليمِ آخرِ طلبٍ له في تسعين يوماً. **ولا يُعدّ الزبونُ مرّتين** في خليّتين.
//
// # والخصوصيّة
//
//   - **الخليّةُ لا النقطة**: ضلعُها `CustomerCellDeg` (≈ ٤٤٠ م شمالاً و٣٦٠ م
//     شرقاً عند الرقّة) — والمُرسَلُ مركزُها لا موضعُ أحد.
//   - **وحدٌّ أدنى** `CustomerMinCount`: خليّةٌ فيها أقلُّ من ثلاثة زبائن لا تُرسَل
//     — **فبيتٌ وحيدٌ في طرف المدينة لا يُعرف بأنّه «زبونٌ هنا».**
//   - **ولا معرّفَ ولا اسمَ ولا هاتف** — الردُّ أعدادٌ ومراكز.

// CustomerCellDeg **ضلعُ خليّة الزبائن بالدرجات.**
const CustomerCellDeg = 0.004

// CustomerMinCount **أقلُّ عددٍ تُرسَل به خليّة** — وما دونه يُحذف.
const CustomerMinCount = 3

// CustomerRecentDays **نافذةُ «آخر عنوان تسليم»** لمن لا عنوانَ محفوظاً له.
const CustomerRecentDays = 90

// CustomerCells **خلايا الزبائن في المشهد** — مركزٌ وعدٌّ لا غير.
func CustomerCells(ctx context.Context, q Querier, box *BBox) ([]Cell, error) {
	var args []any
	cond := ""
	if box != nil {
		if !box.Valid() {
			return nil, fmt.Errorf("مستطيلُ مشهدٍ غيرُ معقول")
		}
		cond = " AND " + box.SQL("p.g", 1)
		args = append(args, box.Args()...)
	}
	sql := fmt.Sprintf(`
		WITH p AS (
			SELECT COALESCE(
			         (SELECT a.location FROM user_addresses a
			           WHERE a.user_id = u.id AND a.is_default LIMIT 1),
			         (SELECT o.dropoff FROM orders o
			           WHERE o.customer_id = u.id
			             AND o.created_at > now() - make_interval(days => %[4]d)
			           ORDER BY o.created_at DESC LIMIT 1)) AS g
			FROM users u
			JOIN user_roles ur ON ur.user_id = u.id AND ur.role_code = 'customer'
			WHERE u.status = 'active')
		SELECT floor(ST_Y(p.g::geometry) / %[1]v) * %[1]v + %[2]v AS cy,
		       floor(ST_X(p.g::geometry) / %[1]v) * %[1]v + %[2]v AS cx,
		       count(*)
		FROM p
		WHERE p.g IS NOT NULL%[5]s
		GROUP BY 1, 2
		HAVING count(*) >= %[3]d
		ORDER BY 3 DESC
		LIMIT 3000`, CustomerCellDeg, CustomerCellDeg/2, CustomerMinCount,
		CustomerRecentDays, cond)
	return scanCells(ctx, q, sql, args...)
}

// ══════════════════════════════════════════════════════════════════════
// **المكتب — من في اللوحة الآن**
// ══════════════════════════════════════════════════════════════════════

// StaffOnline موظّفٌ ظاهرٌ في اللوحة — **اسمٌ ودورٌ لا غير.**
type StaffOnline struct {
	Name  string   `json:"name"`
	Roles []string `json:"roles"`
}

// OnlineStaff **من ظهر في اللوحة في آخر `windowMin` دقيقة** — بإشارة الحضور
// نفسِها التي يقرؤها القبولُ التلقائيُّ حين يفرغ المكتب
// (`orders.StaffPresent`: `users.last_seen_at` و`orders.staff_presence_min`).
//
// **والموظّفُ من يحمل دوراً غيرَ صفة الحساب** (`accountRoles`) — فلا يظهر
// زبونٌ ولا سائقٌ في «المكتب». **ولا هاتفَ ولا معرّف.**
func OnlineStaff(ctx context.Context, q Querier, accountRoles []string, windowMin int64) ([]StaffOnline, error) {
	if windowMin <= 0 {
		windowMin = 5
	}
	rows, err := q.Query(ctx, `
		SELECT u.full_name, array_agg(ur.role_code ORDER BY ur.role_code)
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		WHERE u.status = 'active'
		  AND u.last_seen_at > now() - make_interval(mins => $1::int)
		  AND NOT (ur.role_code = ANY($2::text[]))
		GROUP BY u.id, u.full_name
		ORDER BY u.full_name
		LIMIT 200`, windowMin, accountRoles)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StaffOnline{}
	for rows.Next() {
		var s StaffOnline
		if err := rows.Scan(&s.Name, &s.Roles); err != nil {
			return nil, err
		}
		s.Name = strings.TrimSpace(s.Name)
		out = append(out, s)
	}
	return out, rows.Err()
}

// ══════════════════════════════════════════════════════════════════════
// **شريطُ العدّادات**
// ══════════════════════════════════════════════════════════════════════

// DriverCounts عدّاداتُ السائقين بالحال.
type DriverCounts struct {
	Total     int `json:"total"`
	Available int `json:"available"`
	Busy      int `json:"busy"`
	Idle      int `json:"idle"`
}

// CountDrivers **يعدّ بالحكم نفسِه الذي يلوّن** — `DriverTone`.
func CountDrivers(list []Driver) DriverCounts {
	var c DriverCounts
	for _, d := range list {
		c.Total++
		switch d.Tone {
		case DriverAvailable:
			c.Available++
		case DriverBusy:
			c.Busy++
		default:
			c.Idle++
		}
	}
	return c
}

// FirstStuck **أوّلُ طلبٍ عالقٍ له موضع** — والقائمةُ مرتّبةٌ بالأحدث، فالأقدمُ
// عالقاً هو الأحقّ بالنظر أوّلاً.
func FirstStuck(list []Order) *Order {
	var best *Order
	for i := range list {
		o := &list[i]
		if o.StuckReason == nil {
			continue
		}
		if best == nil || o.CreatedAt.Before(best.CreatedAt) {
			best = o
		}
	}
	return best
}

// stuckColumn **سببُ العلوق أو NULL** — بشرط لوحة الطلبات نفسِه
// (`orders.StuckReasonSQL`)، **فلا تقول الخريطةُ «عالق» عن طلبٍ لا تعدّه اللوحة.**
func stuckColumn(l *orders.StuckLimits) string {
	if l == nil {
		return "NULL::text"
	}
	return orders.StuckReasonSQL(*l)
}
