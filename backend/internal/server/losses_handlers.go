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
// طلبٍ فشل. **وكلُّها قيودٌ في محفظة الخزينة.**
//
// **وجمعُها من الطلبات إعادةُ حسابٍ لما حُسب**: معادلةٌ ثانيةٌ تنحرف يوماً عن
// الأولى، **فيقول التقريرُ رقماً ويقول الدفترُ آخر.** والدفترُ هو الحقيقة —
// هو ما دُفع فعلاً.
//
// # وكلُّ خسارةٍ بالأرقام نفسِها التي في صفحة الأرباح (قرارُ المالك ٢٠٢٦-١٠-١٥)
//
// **كانت هذه الصفحةُ تعدّ `platform_expense` وحدَه** — وصفحةُ الأرباح تعدّ معه خسائرَ
// الطلبات الفاشلة والملغاة والمستردّة، **وتعدّ التسوياتِ اليدويّةَ «خارج الربح»**. فرقمان
// لخسارةٍ واحدة. **وصارت السطورُ من تصنيفٍ واحد** (`platformProfit` → `LossLines`):
//
//	خسائرُ الطلبات (فشل · إلغاء · استرداد)   سالبُ `platform_profit` لكلّ طلب
//	التعويضات                               `platform_expense`
//	نقصُ صندوق المكتب                       `platform_expense` بمرجع النقص
//	تسويةٌ يدويّةٌ إيداعاً في محفظة          `platform_expense` بمرجع طلب التسوية
//	ديونٌ مشطوبة                            طلباتُ الشطب الموافَق عليها بيوم الشطب
//
// **والمستردُّ مستردُّ صفحة الأرباح نفسُه** — نزاعاتٌ مخصومة، وديونٌ سُدّدت بالمكتب أو
// من الشحن، **وتسويةٌ يدويّةٌ خصماً من محفظة.** والصافي = الخسارة − المستردّ.

import (
	"net/http"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

type lossRow struct {
	// Kind **بابُ الخسارة** — `lost_failed` · `lost_cancelled` · `lost_refunded` ·
	// `compensation` · `shortfall` · `adjustment_credit`.
	Kind        string    `json:"kind"`
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

// lossRowsCTE **صفوفُ الخسارة من التصنيف الواحد** — يلي `treasuryEntriesCTE`.
//
// **خسارةُ الطلب صفٌّ واحدٌ للطلب** (مجموعُ قيوده في المدى، بيوم آخر قيدٍ له كما في
// الأرباح)، **وما سواها قيدٌ قيد.** والمشطوبُ رقمٌ للمعلومة لا صفّ خسارة — خسارتُه قُيّدت يومَ نشأ دينُه.
const lossRowsCTE = `,
	lr AS (
		SELECT bucket AS kind, order_id, -sum(amount) AS amount, max(at) AS at, ''::text AS note
		  FROM inr WHERE bucket IN ('lost_failed','lost_cancelled','lost_refunded')
		 GROUP BY bucket, order_id
		UNION ALL
		SELECT bucket, order_id, -amount, at, COALESCE(note, '')
		  FROM inr WHERE bucket IN ('compensation','shortfall','adjustment_credit')
	)`

// handlePlatformLosses خسائرُ المنصّة في المدى — واقعةً بواقعة ومجموعاً.
//
// # والأرقامُ جنباً إلى جنب (قرارُ المالك ٢٠٢٦-١٠-٠٤، «الخسائر والنزاعات» البند ٤)
//
//	lost       مجموعُ سطور الخسارة (`LossLines`) رقماً موجباً
//	recovered  المستردُّ في المدّة نفسِها (`Recovered` صفحة الأرباح)
//	net        الخسارةُ الصافية = lost − recovered
//	owed       ما بقي لنا عند الناس الآن (المفتوحُ من النزاعات بما بقي منه)
//	lines      السطورُ نفسُها بمفاتيح صفحة الأرباح — رقماً موجباً
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
	ctx := r.Context()

	// **الأرقامُ من حساب صفحة الأرباح نفسِه** — لا استعلامٌ ثانٍ ينحرف عنه.
	p, err := s.platformProfit(ctx, fromDay, toDay)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	total := p.LossTotal()
	lines := make([]profitLine, 0, len(p.LossLines()))
	for _, l := range p.LossLines() {
		lines = append(lines, profitLine{Key: l.Key, Amount: -l.Amount})
	}

	// **وما لنا عند الناس الآن** — لقطةٌ لا مدّة: الدَّينُ قائمٌ حتّى يُحسم.
	var owed int64
	if err := s.pg.QueryRow(ctx, `
		SELECT COALESCE(sum(amount - recovered), 0) FROM disputes WHERE status = 'open'`).
		Scan(&owed); err != nil {
		s.respondErr(w, err)
		return
	}

	// **وصفحةٌ محدودةٌ — والمجموعُ على المدّة كلِّها لا على الصفحة** (قرارُ
	// المالك ٢٠٢٦-٠٨-١٠). **رقمٌ يُشتقّ من صفحةٍ وهو عن الكلّ** يُنقص الخسارةَ
	// أمام عين من يقرأ كلّما قُلّب الترقيم.
	pg := pagingOf(r, 25)
	var count int
	if err := s.pg.QueryRow(ctx, treasuryEntriesCTE+lossRowsCTE+`
		SELECT count(*) FROM lr WHERE amount <> 0`, fromDay, toDay).Scan(&count); err != nil {
		s.respondErr(w, err)
		return
	}
	rows, err := s.pg.Query(ctx, treasuryEntriesCTE+lossRowsCTE+`
		SELECT lr.kind, o.id::text, o.number, lr.amount, lr.note, lr.at,
		       d.id::text, d.status, d.party_role,
		       COALESCE(d.merchant_id::text, d.party_user_id::text),
		       COALESCE(mm.name, uu.full_name)
		FROM lr
		LEFT JOIN orders o ON o.id = lr.order_id
		LEFT JOIN LATERAL (
		    SELECT x.* FROM disputes x WHERE x.order_id = o.id
		    ORDER BY x.created_at DESC LIMIT 1) d ON true
		LEFT JOIN merchants mm ON mm.id = d.merchant_id
		LEFT JOIN users uu     ON uu.id = d.party_user_id
		WHERE lr.amount <> 0
		ORDER BY lr.at DESC, lr.kind, o.number
		LIMIT $3 OFFSET $4`, fromDay, toDay, pg.PerPage, pg.Offset)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	out := []lossRow{}
	for rows.Next() {
		var x lossRow
		if err := rows.Scan(&x.Kind, &x.OrderID, &x.OrderNumber, &x.Amount, &x.Note, &x.CreatedAt,
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
	_ = s.pg.QueryRow(ctx,
		`SELECT COALESCE(balance, 0) FROM wallets WHERE is_treasury LIMIT 1`).Scan(&treasury)

	httpx.JSON(w, http.StatusOK, map[string]any{
		// **و`total` مجموعُ المال و`count` عددُ الصفوف** — اسمان لا يجتمعان.
		"losses": out, "total": total, "count": count,
		"recovered": p.Recovered, "net": total - p.Recovered, "owed": owed,
		"written_off": p.WrittenOff,
		"lines":       lines,
		"page":        pg.Page, "per_page": pg.PerPage,
		"treasury_balance": treasury,
		"from":             fromDay, "to": toDay,
	})
}
