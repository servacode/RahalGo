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
	// **والمحفظةُ تُربط بصاحبها لا بمعرّفٍ لها.**
	//
	// (كشفه فحصُ المشروع ٢٠٢٦-٠٨-٠٨: صفحةُ الخسائر تردّ ٥٠٠ دائماً.)
	//
	// كان الوصلُ على عمودين لا وجودَ لهما: جدولُ المحافظ مفتاحُه صاحبُها،
	// وجدولُ الحركات يحمل صاحبَها كذلك — **ولا معرّفَ محفظةٍ في أيٍّ منهما.**
	// **فالاستعلامُ يسقط قبل أن يقرأ صفّاً**، والصفحةُ لم تعمل قطّ.
	// ══════════════════════════════════════════════════════════════════
	// **وصفحةٌ محدودةٌ — والمجموعُ على المدّة كلِّها لا على الصفحة**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-١٠: «لا تنسَ إضافة الباجينيشن».)
	//
	// **والمجموعُ كان يُجمَع من الأسطر المعروضة** — وكان صحيحاً حين تُعرض
	// كلُّها. **ومع الترقيم يصير مجموعَ عشرين سطراً لا خسائرَ الشهر**:
	// يُقلَّب إلى الصفحة الثانية **فينقص ما خسرته المنصّةُ أمام عين من
	// يقرأ**، ثمّ يُبنى عليه قرارُ تسعير.
	//
	// **وهي عائلةُ العطب نفسِها في النزاعات وتقييمات الحساب**: **رقمٌ يُشتقّ
	// من صفحةٍ وهو عن الكلّ.**
	pg := pagingOf(r, 25)
	const lossFilter = `
		FROM wallet_transactions t
		JOIN wallets w ON w.user_id = t.user_id
		WHERE w.is_treasury AND t.kind = 'platform_expense'
		  AND t.created_at >= $1 AND t.created_at < $2`

	var count int
	var total int64
	if err := s.pg.QueryRow(r.Context(),
		`SELECT count(*), COALESCE(sum(-t.amount), 0)`+lossFilter, from, toEnd).
		Scan(&count, &total); err != nil {
		s.respondErr(w, err)
		return
	}

	rows, err := s.pg.Query(r.Context(), `
		SELECT o.number, -t.amount, t.note, t.created_at
		FROM wallet_transactions t
		JOIN wallets w ON w.user_id = t.user_id
		LEFT JOIN orders o ON o.id::text = t.ref
		WHERE w.is_treasury AND t.kind = 'platform_expense'
		  AND t.created_at >= $1 AND t.created_at < $2
		ORDER BY t.created_at DESC LIMIT $3 OFFSET $4`, from, toEnd, pg.PerPage, pg.Offset)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	out := []lossRow{}
	for rows.Next() {
		var x lossRow
		if err := rows.Scan(&x.OrderNumber, &x.Amount, &x.Note, &x.CreatedAt); err != nil {
			s.respondErr(w, err)
			return
		}
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
		// **و`total` مجموعُ المال و`count` عددُ الصفوف** — **اسمان لا
		// يجتمعان في مفتاحٍ واحد.**
		"losses": out, "total": total, "count": count,
		"page": pg.Page, "per_page": pg.PerPage,
		"treasury_balance": treasury,
		"from":             from.Format("2006-01-02"), "to": to.Format("2006-01-02"),
	})
}
