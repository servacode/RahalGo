package server

// خسائرُ المنصة — **من الدفتر لا من تقدير.**
//
// # المسألة
//
// قاعدةُ المالك: «**نحسب الخسارة الفعلية فقط وليس الخسارة الافتراضية**».
//
// **والفرقُ ليس لفظياً**: طلبٌ أُلغي قبل التحضير خسارتُه صفر — لم يُطبخ طعام
// ولم يقد سائق. **والتقريرُ الذي يعدّه خسارةً يجعل المنصةَ تبدو خاسرةً وهي لم
// تدفع شيئاً**، فيُتّخذ قرارٌ على رقمٍ لا وجود له.
//
// # ولماذا من الخزينة لا من الطلبات
//
// الخسارةُ الفعلية **مالٌ خرج**: بضاعةٌ لم يستردّها متجر، وتعويضُ سائقٍ عن
// طلبٍ فشل. **وكلُّها قيودٌ في محفظة الخزينة** — `platform_expense`.
//
// **وجمعُها من الطلبات إعادةُ حسابٍ لما حُسب**: معادلةٌ ثانيةٌ تنحرف يوماً عن
// الأولى، **فيقول التقريرُ رقماً ويقول الدفترُ آخر.** والدفترُ هو الحقيقة —
// هو ما دُفع فعلاً.

import (
	"net/http"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

type lossRow struct {
	OrderID     *string   `json:"order_id"`
	OrderNumber *int64    `json:"order_number"`
	Amount      int64     `json:"amount"`
	Note        string    `json:"note"`
	CreatedAt   time.Time `json:"created_at"`
	// **نزاعُ الخسارة إن كان لها نزاع** (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ٤):
	// مفتوح · انخصم · انسقط — **وفارغٌ = لا نزاع.** والأحدثُ على الطلب يُقرأ.
	DisputeID     *string `json:"dispute_id"`
	DisputeStatus *string `json:"dispute_status"`
	PartyRole     *string `json:"party_role"`
	PartyID       *string `json:"party_id"`
	PartyName     *string `json:"party_name"`
}

// handlePlatformLosses ما خرج من الخزينة في المدى — واقعةً بواقعة ومجموعاً.
//
// # وأربعةُ أرقامٍ جنباً إلى جنب (قرارُ المالك ٢٠٢٦-١٠-٠٤، «الخسائر والنزاعات» البند ٤)
//
//	lost       ما خرج من الخزينة (`platform_expense`)
//	recovered  ما عاد إليها من النزاعات المخصومة في المدّة نفسِها
//	net        الخسارةُ الصافية = lost − recovered
//	owed       ما بقي لنا عند الناس الآن (المفتوحُ من النزاعات بما بقي منه)
//
// **وكان المجموعُ إجماليّاً وحدَه** — فالمالكُ لا يعرف كم خسرنا فعلاً بعد الاسترجاع.
//
// # والأيّامُ بتوقيت دمشق لا غرينتش (المشكلة ١٦)
//
// خسارةٌ بين منتصف الليل والثالثة فجراً كانت تُحسب لليوم السابق. **والحدّان
// يُحسبان في القاعدة** (`AT TIME ZONE 'Asia/Damascus'`) كما في التقارير — لا
// بمنطقةٍ يحمّلها الخادمُ فتسقط إلى غرينتش على جهازٍ بلا جدول مناطق.
func (s *Server) handlePlatformLosses(w http.ResponseWriter, r *http.Request) {
	to, err := time.Parse("2006-01-02", r.URL.Query().Get("to"))
	if err != nil {
		to = time.Now().UTC().Add(3 * time.Hour) // يومُ دمشق
	}
	from, err := time.Parse("2006-01-02", r.URL.Query().Get("from"))
	if err != nil {
		from = to.AddDate(0, 0, -29)
	}
	fromDay, toDay := from.Format("2006-01-02"), to.Format("2006-01-02")

	// **قيودُ المصروف تُعرض موجبةً** — تُقرأ «خسرنا كذا» لا «−كذا».
	//
	// **وصفحةٌ محدودةٌ — والمجموعُ على المدّة كلِّها لا على الصفحة** (قرارُ
	// المالك ٢٠٢٦-٠٨-١٠). **رقمٌ يُشتقّ من صفحةٍ وهو عن الكلّ** يُنقص الخسارةَ
	// أمام عين من يقرأ كلّما قُلّب الترقيم.
	pg := pagingOf(r, 25)
	const rangeSQL = `
		  AND t.created_at >= ($1::date::timestamp AT TIME ZONE 'Asia/Damascus')
		  AND t.created_at <  (($2::date + 1)::timestamp AT TIME ZONE 'Asia/Damascus')`
	const lossFilter = `
		FROM wallet_transactions t
		JOIN wallets w ON w.user_id = t.user_id
		WHERE w.is_treasury AND t.kind = 'platform_expense'` + rangeSQL

	var count int
	var total int64
	if err := s.pg.QueryRow(r.Context(),
		`SELECT count(*), COALESCE(sum(-t.amount), 0)`+lossFilter, fromDay, toDay).
		Scan(&count, &total); err != nil {
		s.respondErr(w, err)
		return
	}

	// **والمسترَدُّ ما عاد إلى الخزينة بمرجع نزاع** — خصمٌ من طرفٍ يُقيَّد ربحاً
	// للخزينة بمعرّف النزاع (`CreditTreasuryDirect`).
	var recovered int64
	if err := s.pg.QueryRow(r.Context(), `
		SELECT COALESCE(sum(t.amount), 0)
		FROM wallet_transactions t
		JOIN wallets w ON w.user_id = t.user_id
		WHERE w.is_treasury AND t.kind = 'platform_profit'
		  AND t.ref IN (SELECT id::text FROM disputes)`+rangeSQL, fromDay, toDay).
		Scan(&recovered); err != nil {
		s.respondErr(w, err)
		return
	}
	// **وما لنا عند الناس الآن** — لقطةٌ لا مدّة: الدَّينُ قائمٌ حتّى يُحسم.
	var owed int64
	if err := s.pg.QueryRow(r.Context(), `
		SELECT COALESCE(sum(amount - recovered), 0) FROM disputes WHERE status = 'open'`).
		Scan(&owed); err != nil {
		s.respondErr(w, err)
		return
	}

	rows, err := s.pg.Query(r.Context(), `
		SELECT o.id::text, o.number, -t.amount, t.note, t.created_at,
		       d.id::text, d.status, d.party_role,
		       COALESCE(d.merchant_id::text, d.party_user_id::text),
		       COALESCE(mm.name, uu.full_name)
		FROM wallet_transactions t
		JOIN wallets w ON w.user_id = t.user_id
		LEFT JOIN orders o ON o.id::text = t.ref
		LEFT JOIN LATERAL (
		    SELECT x.* FROM disputes x WHERE x.order_id = o.id
		    ORDER BY x.created_at DESC LIMIT 1) d ON true
		LEFT JOIN merchants mm ON mm.id = d.merchant_id
		LEFT JOIN users uu     ON uu.id = d.party_user_id
		WHERE w.is_treasury AND t.kind = 'platform_expense'`+rangeSQL+`
		ORDER BY t.created_at DESC LIMIT $3 OFFSET $4`, fromDay, toDay, pg.PerPage, pg.Offset)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	out := []lossRow{}
	for rows.Next() {
		var x lossRow
		if err := rows.Scan(&x.OrderID, &x.OrderNumber, &x.Amount, &x.Note, &x.CreatedAt,
			&x.DisputeID, &x.DisputeStatus, &x.PartyRole, &x.PartyID, &x.PartyName); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}

	// **ورصيدُ الخزينة يبقى في الردّ** — لمن يقرؤه؛ والصفحةُ تعرض الأربعة.
	var treasury int64
	_ = s.pg.QueryRow(r.Context(),
		`SELECT COALESCE(balance, 0) FROM wallets WHERE is_treasury LIMIT 1`).Scan(&treasury)

	httpx.JSON(w, http.StatusOK, map[string]any{
		// **و`total` مجموعُ المال و`count` عددُ الصفوف** — اسمان لا يجتمعان.
		"losses": out, "total": total, "count": count,
		"recovered": recovered, "net": total - recovered, "owed": owed,
		"page": pg.Page, "per_page": pg.PerPage,
		"treasury_balance": treasury,
		"from":             fromDay, "to": toDay,
	})
}
