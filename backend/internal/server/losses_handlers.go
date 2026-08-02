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
	OrderNumber *int64    `json:"order_number"`
	Amount      int64     `json:"amount"`
	Note        string    `json:"note"`
	CreatedAt   time.Time `json:"created_at"`
}

// handlePlatformLosses ما خرج من الخزينة في المدى — واقعةً بواقعة ومجموعاً.
func (s *Server) handlePlatformLosses(w http.ResponseWriter, r *http.Request) {
	to, err := time.Parse("2006-01-02", r.URL.Query().Get("to"))
	if err != nil {
		to = time.Now()
	}
	from, err := time.Parse("2006-01-02", r.URL.Query().Get("from"))
	if err != nil {
		from = to.AddDate(0, 0, -29)
	}
	toEnd := to.AddDate(0, 0, 1)

	// **قيودُ المصروف تُعرض موجبةً** — تُقرأ «خسرنا كذا» لا «−كذا»،
	// والإشارةُ في الدفتر لا في الشاشة.
	rows, err := s.pg.Query(r.Context(), `
		SELECT o.number, -t.amount, t.note, t.created_at
		FROM wallet_transactions t
		JOIN wallets w ON w.id = t.wallet_id
		LEFT JOIN orders o ON o.id::text = t.ref
		WHERE w.is_treasury AND t.kind = 'platform_expense'
		  AND t.created_at >= $1 AND t.created_at < $2
		ORDER BY t.created_at DESC LIMIT 500`, from, toEnd)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	out := []lossRow{}
	var total int64
	for rows.Next() {
		var x lossRow
		if err := rows.Scan(&x.OrderNumber, &x.Amount, &x.Note, &x.CreatedAt); err != nil {
			s.respondErr(w, err)
			return
		}
		total += x.Amount
		out = append(out, x)
	}

	// **والربحُ بجانبها** — خسارةٌ بلا ما يقابلها رقمٌ يُفزع بلا معنى.
	//
	// **ورصيدُ الخزينة لا مجموعُ قيود**: هو الصافي بعد كلّ ما دخل وخرج،
	// **ولا يحتاج جمعاً ثانياً يُخطئ.**
	var treasury int64
	_ = s.pg.QueryRow(r.Context(),
		`SELECT COALESCE(balance, 0) FROM wallets WHERE is_treasury LIMIT 1`).Scan(&treasury)

	httpx.JSON(w, http.StatusOK, map[string]any{
		"losses": out, "total": total, "treasury_balance": treasury,
		"from": from.Format("2006-01-02"), "to": to.Format("2006-01-02"),
	})
}
