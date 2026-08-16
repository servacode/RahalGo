package server

/*
**الأرباح — من أين جاء المالُ وأين ذهب، ولكلِّ إنسانٍ نصيبُه.**

(قرارُ المالك ٢٠٢٦-٠٨-١٦.)

# ولماذا رقمان لا رقم

**طلب المالكُ معادلةً**: «الربحُ مجموعُ الهامش والعمولة ناقص مجموع المصاريف
والخسائر».

**والمعنى صحيح** — **والحسابُ المباشرُ لها يخطئ**: `platform_profit` في الدفتر
ليس «العمولة»، هو **ما دفعه الزبونُ ناقصَ ما رُدَّ له ناقصَ ما قُيّد للمتجر
والسائق والمندوب**. **فهو يحوي الهامشَ والعمولةَ معاً، وقد طُرح منه الخصمُ
والكوبون أصلاً** لأنّ الزبونَ دفع أقلّ.

**فلو جُمع الهامشُ والعمولةُ من أعمدة الطلبات ثمّ طُرح الخصمُ مرّةً أخرى
لَحُسب مرّتين** — **ولَخالف الناتجُ رصيدَ الخزينة**، ولا يُعرف أيُّهما يُصدَّق.

**فيُعرض الاثنان**:

  · **التفصيل** — هامشٌ وعمولةٌ وخصومٌ ودعواتٌ وخسائرُ ومصاريف: **يقول من
    أين جاء المالُ وأين ذهب.**
  · **والصافي من الدفتر** — مجموعُ ما دخل الخزينةَ وخرج منها في المدّة:
    **يطابق رصيدَها دائماً**، فلا يُقرأ رقمان متناقضان.

# والمدى من أوّل يومٍ إلى آخره

(«فنعرف مثلاً سجلَّ الربح والخسارة من أوّل يومٍ إلى تاريخ آخر يوم».)
*/

import (
	"net/http"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// profitRow **سطرُ شخصٍ في تبويبه** — أعمدتُه تختلف بالدور.
type profitRow struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	Phone  string `json:"phone"`
	// A وB وC **ثلاثةُ أرقامٍ يُسمّيها التبويبُ في الشاشة** — ولا يُكتب
	// اسمُها هنا: **معجمُ الشاشة يترجم، والمحرّكُ لا يعرف لغةَ من يقرأ.**
	A int64 `json:"a"`
	B int64 `json:"b"`
	C int64 `json:"c"`
}

// handleProfits **الأرباحُ بتبويباتها الخمسة.**
func (s *Server) handleProfits(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, to := q.Get("from"), q.Get("to")
	// **ومدًى مفتوحٌ يعني «من أوّل يوم»** — (قرارُ المالك)، فلا يُجبَر أحدٌ
	// على ملء حقلٍ ليرى الكلّ.
	if _, err := time.Parse("2006-01-02", from); from != "" && err != nil {
		s.respondErr(w, errValidation)
		return
	}
	if _, err := time.Parse("2006-01-02", to); to != "" && err != nil {
		s.respondErr(w, errValidation)
		return
	}

	switch q.Get("tab") {
	case "", "platform":
		s.profitsPlatform(w, r, from, to)
	case "customers":
		s.profitsParties(w, r, from, to, "customers")
	case "reps":
		s.profitsParties(w, r, from, to, "reps")
	case "drivers":
		s.profitsParties(w, r, from, to, "drivers")
	case "merchants":
		s.profitsMerchants(w, r, from, to)
	default:
		s.respondErr(w, httpx.ErrNotFound)
	}
}

// ══════════════════════════════════════════════════════════════════════
//
//	**١ · المنصّة**
//
// ══════════════════════════════════════════════════════════════════════
func (s *Server) profitsPlatform(w http.ResponseWriter, r *http.Request, from, to string) {
	ctx := r.Context()

	// **والطلباتُ المسلَّمةُ وحدَها** — **وطلبٌ أُلغيَ لا هامشَ فيه ولا
	// عمولة**، وعدُّه يجعل الشاشةَ تَعِد بمالٍ لم يُقبض.
	//
	// **والمدى على يوم التسليم** — لا على يوم الإنشاء: **طلبٌ أُنشئ في
	// آخر الشهر وسُلّم في أوّل الذي يليه ربحُ الثاني.**
	const orderScope = `
		FROM orders o
		WHERE o.status = 'delivered'
		  AND ($1 = '' OR o.delivered_at >= $1::date)
		  AND ($2 = '' OR o.delivered_at < ($2::date + 1))`

	var ordersN int
	var margin, commission, discount, sales int64
	if err := s.pg.QueryRow(ctx, `
		SELECT count(*),
		       COALESCE(sum(`+orders.OrderMarginSQL("o.id")+`), 0),
		       COALESCE(sum(o.platform_commission), 0),
		       COALESCE(sum(o.discount), 0),
		       COALESCE(sum(o.total), 0)`+orderScope, from, to).
		Scan(&ordersN, &margin, &commission, &discount, &sales); err != nil {
		s.respondErr(w, err)
		return
	}

	// **وما خرج من الخزينة بأنواعه** — يُقرأ من الدفتر لا من معادلة:
	// **قيدٌ رُفض أو صُحّح يظهر أثرُه هنا فوراً.**
	//
	// **والمدى على وقت القيد** — وهو وقتُ خروج المال.
	const ledgerScope = `
		FROM wallet_transactions t
		JOIN wallets wl ON wl.user_id = t.user_id
		WHERE wl.is_treasury
		  AND ($1 = '' OR t.created_at >= $1::date)
		  AND ($2 = '' OR t.created_at < ($2::date + 1))`

	var losses, opex, referrals, penalties, net int64
	if err := s.pg.QueryRow(ctx, `
		SELECT
			COALESCE(sum(-t.amount) FILTER (WHERE t.kind = 'platform_expense'), 0),
			COALESCE(sum(-t.amount) FILTER (WHERE t.kind = 'operating_expense'), 0),
			COALESCE(sum(-t.amount) FILTER (WHERE t.kind = 'reward'), 0),
			COALESCE(sum(t.amount)  FILTER (WHERE t.kind = 'penalty'), 0),
			-- ══════════════════════════════════════════════════════════
			-- **والصافي مجموعُ الحركة كلِّها — لا معادلةٌ تُحاكيها**
			-- ══════════════════════════════════════════════════════════
			--
			-- **فيطابق رصيدَ الخزينة دائماً**، ولا يُقرأ رقمان متناقضان
			-- في شاشةٍ واحدة. **ومعادلةٌ تُكتب بيدٍ تنسى نوعاً يُضاف غدا.**
			COALESCE(sum(t.amount), 0)`+ledgerScope, from, to).
		Scan(&losses, &opex, &referrals, &penalties, &net); err != nil {
		s.respondErr(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"orders": ordersN, "sales": sales,
		"margin": margin, "commission": commission, "discount": discount,
		"losses": losses, "opex": opex, "referrals": referrals,
		"penalties": penalties,
		// **والدخلُ المحصَّلُ من الطلبات** — ما بقي للخزينة بعد أنصبة
		// الأطراف، **وهو الرقمُ الذي يُطرح منه ما خرج.**
		"net": net,
	})
}

// ══════════════════════════════════════════════════════════════════════
//
//	**٢ و٣ و٤ · الزبائنُ والمندوبون والسائقون**
//
// ══════════════════════════════════════════════════════════════════════
//
// **والمصدرُ الدفترُ لا الطلبات** — «ماذا كسب فعلاً» سؤالٌ عن مالٍ قُيّد،
// **وحسابٌ من الطلبات يقول ما كان ينبغي أن يُقيَّد** وقد يختلفان.
func (s *Server) profitsParties(w http.ResponseWriter, r *http.Request, from, to, tab string) {
	pg := pagingOf(r, 25)

	// **ولكلّ دورٍ نوعُ قيده**: الزبونُ يكسب بالدعوة، والمندوبُ بالعمولة،
	// والسائقُ بأجرة التوصيل. **والمكافأةُ والعقوبةُ تخصّان الاثنين.**
	var earnKinds, role string
	switch tab {
	case "customers":
		earnKinds, role = `'reward'`, "customer"
	case "reps":
		earnKinds, role = `'commission','reward'`, "sales"
	default:
		earnKinds, role = `'driver_earning','reward'`, "driver"
	}

	// ══════════════════════════════════════════════════════════════════
	// **وجسمٌ واحدٌ يحمل الجداولَ والشرطَ معاً — للعدّ وللقائمة**
	// ══════════════════════════════════════════════════════════════════
	//
	// **وأوّلُ كتابةٍ فصلت الجداولَ عن الوصلات**: كان `scope` يحمل
	// `FROM users u` وحدَه، **والقائمةُ تشير إلى `e` و`sp` وليسا فيه** —
	// فردَّ الخادمُ `missing FROM-clause entry for table "e"` عند كلّ نداء.
	// (كشفه المالكُ على شاشته ٢٠٢٦-٠٨-١٦.)
	//
	// **والعدُّ يحمل الشرطَ نفسَه** — **وعدٌّ يقول ألفاً وقائمةٌ تعرض
	// ثلاثةً يجعل التنقّلَ يعد بصفحاتٍ فارغة.**
	ctes := `
		WITH earned AS (
			SELECT t.user_id,
			       COALESCE(sum(t.amount) FILTER (WHERE t.kind IN (` + earnKinds + `)), 0) AS got,
			       COALESCE(sum(-t.amount) FILTER (WHERE t.kind = 'penalty'), 0) AS lost
			FROM wallet_transactions t
			WHERE ($1 = '' OR t.created_at >= $1::date)
			  AND ($2 = '' OR t.created_at < ($2::date + 1))
			GROUP BY t.user_id
		), spent AS (
			-- **وما أنفقه عبر المنصّة** — المسلَّمُ وحدَه: **طلبٌ أُلغيَ لم
			-- يُنفَق فيه شيء.**
			SELECT o.customer_id AS uid, COALESCE(sum(o.total), 0) AS paid
			FROM orders o
			WHERE o.status = 'delivered'
			  AND ($1 = '' OR o.delivered_at >= $1::date)
			  AND ($2 = '' OR o.delivered_at < ($2::date + 1))
			GROUP BY o.customer_id
		)`

	// **ومن لم يكسب ولم ينفق لا يُعرض** — **وقائمةٌ فيها ألفُ صفرٍ لا
	// تُقرأ**، ويضيع فيها من كسب.
	body := `
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id AND ur.role_code = '` + role + `'
		LEFT JOIN earned e ON e.user_id = u.id
		LEFT JOIN spent sp ON sp.uid = u.id
		WHERE u.deleted_at IS NULL
		  AND (COALESCE(e.got, 0) <> 0 OR COALESCE(e.lost, 0) <> 0
		       OR COALESCE(sp.paid, 0) <> 0)`

	var count int
	if err := s.pg.QueryRow(r.Context(),
		ctes+` SELECT count(*)`+body, from, to).Scan(&count); err != nil {
		s.respondErr(w, err)
		return
	}

	rows, err := s.pg.Query(r.Context(), ctes+`
		SELECT u.id::text, COALESCE(NULLIF(u.full_name, ''), ''), u.phone::text,
		       COALESCE(e.got, 0), COALESCE(e.lost, 0), COALESCE(sp.paid, 0)`+body+`
		ORDER BY COALESCE(e.got, 0) DESC, COALESCE(sp.paid, 0) DESC
		LIMIT $3 OFFSET $4`, from, to, pg.PerPage, pg.Offset)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []profitRow{}
	for rows.Next() {
		var x profitRow
		if err := rows.Scan(&x.UserID, &x.Name, &x.Phone, &x.A, &x.B, &x.C); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, x)
	}
	httpx.JSON(w, http.StatusOK, paged("rows", out, count, pg))
}

// ══════════════════════════════════════════════════════════════════════
//
//	**٥ · المتاجر**
//
// ══════════════════════════════════════════════════════════════════════
//
// («كلُّ صاحب متجرٍ ماذا كسب منّا وماذا كسبنا منه عمولةً وهامشاً وإجماليُّ
//
//	مبيعاته لدينا».)
//
// **والصفُّ للمتجر لا لصاحبه** — **من ملك متجرين يخلط رقمَيهما في سطرٍ
// واحدٍ فلا يُعرف أيُّهما يربح.**
func (s *Server) profitsMerchants(w http.ResponseWriter, r *http.Request, from, to string) {
	pg := pagingOf(r, 25)
	const scope = `
		FROM merchants m
		LEFT JOIN orders o ON o.merchant_id = m.id AND o.status = 'delivered'
		  AND ($1 = '' OR o.delivered_at >= $1::date)
		  AND ($2 = '' OR o.delivered_at < ($2::date + 1))`

	var count int
	if err := s.pg.QueryRow(r.Context(),
		`SELECT count(*) FROM merchants`).Scan(&count); err != nil {
		s.respondErr(w, err)
		return
	}
	rows, err := s.pg.Query(r.Context(), `
		SELECT m.id::text, m.name, COALESCE(u.phone::text, ''),
		       -- **ما كسبه** — سعرُ الشراء: **ما قُيّد له فعلاً.**
		       COALESCE(sum(`+orders.OrderMarginSQL("o.id")+`), 0) AS our_margin,
		       COALESCE(sum(o.platform_commission), 0) AS our_commission,
		       COALESCE(sum(o.subtotal), 0) AS gross`+scope+`
		LEFT JOIN users u ON u.id = m.owner_user_id
		GROUP BY m.id, m.name, u.phone
		ORDER BY sum(o.subtotal) DESC NULLS LAST, m.name
		LIMIT $3 OFFSET $4`, from, to, pg.PerPage, pg.Offset)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	type store struct {
		ID    string `json:"user_id"`
		Name  string `json:"name"`
		Phone string `json:"phone"`
		// A **هامشُنا** · B **عمولتُنا** · C **إجماليُّ مبيعاته**
		A int64 `json:"a"`
		B int64 `json:"b"`
		C int64 `json:"c"`
		// Earned **ما كسبه هو** — المبيعاتُ ناقصَ ما أخذناه.
		Earned int64 `json:"earned"`
	}
	out := []store{}
	for rows.Next() {
		var x store
		if err := rows.Scan(&x.ID, &x.Name, &x.Phone, &x.A, &x.B, &x.C); err != nil {
			s.respondErr(w, err)
			return
		}
		// **وما كسبه هو = مبيعاتُه ناقصَ هامشِنا وعمولتِنا** — **ورقمٌ
		// يُقرأ دخلاً وهو مبيعاتٌ يجعل المتجرَ يظنّ نفسَه أربح ممّا هو.**
		x.Earned = x.C - x.A - x.B
		out = append(out, x)
	}
	httpx.JSON(w, http.StatusOK, paged("rows", out, count, pg))
}
