package server

// ══════════════════════════════════════════════════════════════════════
// **عتادُ الإغلاق الأخير — على التجهيز وحدَه** (`qa_reconcile.go`)
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٩-٢٣: «الدفعةُ الأخيرة قبل الجلسة المرافقة».)
//
// شيئان ضيّقان لشهود الزبون، كلاهما يسقط مغلقاً في الإنتاج:
//
//	merchant_second        ·  متجرٌ ثانٍ صغيرٌ عكوسٌ لشهود سقفِ المصادر (CUST-11-036)
//	merchant_second_clear  ·  حذفُه (FK-safe)
//	GET /qa/reconcile      ·  مطابقةُ بيانات التجهيز — أعدادٌ وثوابتُ فقط (CUST-22-013)
//
// **ولا يُصدِر شيءٌ منها سرّاً ولا بياناتِ زبونٍ خامّةً ولا توكناً ولا SQL حرّاً**،
// و`reconcile` **قراءةٌ محضةٌ بلا أيّ تعديل**، و`merchant_second*` لا يمسّان مالاً.

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// qaSecondMerchantName اسمُ المتجر الثاني القابلِ للحذف — علامةٌ ثابتةٌ للتنظيف.
const qaSecondMerchantName = "QA متجر ثانٍ (اختبار القبول)"

// qaMerchantSecond **يبني متجراً ثانياً صغيراً على التجهيز** — مصدرٌ ثانٍ متمايزٌ
// (`SourcesOf` يجمع بـ`merchant_id`) لشهود سقفِ المصادر (`orders.max_sources`).
//
// عكوسٌ (`merchant_second_clear`)، بلا أثرٍ ماليّ، ولا يبدأ قبولَ المتجر: صنفٌ
// واحدٌ متاحٌ معتمَدٌ يكفي أن يُعدَّ مصدراً في تسعيرةٍ/طلب. يُعيد المعرّفَين.
func (s *Server) qaMerchantSecond(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// **متكرّرٌ آمن**: لو كان قائماً من شاهدٍ سابقٍ يُعاد لا يُضاعَف.
	var mid, itemID string
	err := s.pg.QueryRow(ctx, `
		SELECT m.id::text, mi.id::text
		FROM merchants m
		JOIN menu_items mi ON mi.merchant_id = m.id
		WHERE m.name = $1
		ORDER BY mi.id LIMIT 1`, qaSecondMerchantName).Scan(&mid, &itemID)
	if err == nil {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"merchant_id": mid, "item_id": itemID, "reused": true,
		})
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		s.respondErr(w, err)
		return
	}

	tx, err := s.pg.Begin(ctx)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := tx.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, commission_percent, status)
		VALUES ($1, (SELECT id FROM categories ORDER BY id LIMIT 1), 10, 'active')
		RETURNING id::text`, qaSecondMerchantName).Scan(&mid); err != nil {
		s.respondErr(w, err)
		return
	}
	var secID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO menu_sections (merchant_id, name)
		VALUES ($1::uuid, 'الرئيسية') RETURNING id::text`, mid).Scan(&secID); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO menu_items
		  (merchant_id, section_id, platform_section_id, name, merchant_price, price, available, approved)
		VALUES ($1::uuid, $2::uuid,
		        (SELECT id FROM platform_sections ORDER BY sort_order LIMIT 1),
		        'QA صنف المصدر الثاني', 10000, 10000, true, true)
		RETURNING id::text`, mid, secID).Scan(&itemID); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA second merchant seeded (staging-only)", "merchant", mid, "item", itemID)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"merchant_id": mid, "item_id": itemID, "reused": false,
	})
}

// qaMerchantSecondClear **يحذف المتجرَ الثاني وأصنافَه وأقسامَه** — FK-safe.
func (s *Server) qaMerchantSecondClear(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// **الأصنافُ ثمّ الأقسامُ ثمّ المتجر** — بالترتيب الذي تسمح به المفاتيح.
	if _, err := tx.Exec(ctx, `
		DELETE FROM menu_items WHERE merchant_id IN
		  (SELECT id FROM merchants WHERE name = $1)`, qaSecondMerchantName); err != nil {
		s.respondErr(w, err)
		return
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM menu_sections WHERE merchant_id IN
		  (SELECT id FROM merchants WHERE name = $1)`, qaSecondMerchantName); err != nil {
		s.respondErr(w, err)
		return
	}
	tag, err := tx.Exec(ctx, `DELETE FROM merchants WHERE name = $1`, qaSecondMerchantName)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA second merchant cleared (staging-only)", "deleted", tag.RowsAffected())
	httpx.JSON(w, http.StatusOK, map[string]any{"deleted_merchants": tag.RowsAffected()})
}

// handleQAStagingReconcile **مطابقةُ بيانات التجهيز بعد الاختبار** — على التجهيز
// وحدَه، **قراءةٌ محضة** (`CUST-22-013`).
//
// # ماذا يُرجع
//
//	money    ·  فحوصُ `fininv` التشغيليّةُ (نفسُ moneycheck): كم فُحص، كم سلِم،
//	            وأيُّها خُرق بعددِ صفوفه فقط — لا صفوفَ ولا بيانات.
//	residue  ·  أثرُ عتادِ QA: طلباتٌ جاريةٌ لزبونَي QA، رصيدُ محفظتيهما،
//	            أصنافُ الكثافة، عروضُ QA الفعّالة، والمتجرُ الثاني — كلُّها أعداد.
//
// **ولا سرَّ ولا هويّةَ زبونٍ ولا توكن**: أرقامٌ مجمَّعةٌ فقط. **ولا تعديلَ.**
func (s *Server) handleQAStagingReconcile(w http.ResponseWriter, r *http.Request) {
	if !s.qaStagingEnabled() {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	ctx := r.Context()

	// ── الثوابتُ الماليّة (moneycheck على التجهيز) ──────────────────────
	total, passed := 0, 0
	failed := []map[string]any{}
	for _, c := range fininv.Select() {
		if !c.Ops {
			continue // **ما لا يُشغَّل على قاعدةِ تشغيلٍ لا يُعَدّ** — كـmoneycheck.
		}
		total++
		vs, err := fininv.Run(ctx, s.pg, c.ID)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		if len(vs) == 0 {
			passed++
			continue
		}
		failed = append(failed, map[string]any{
			"id": c.ID, "name": c.Name, "violating_rows": len(vs[0].Rows),
		})
	}

	// ── أثرُ عتادِ QA ────────────────────────────────────────────────
	count := func(sql string, args ...any) int64 {
		var n int64
		_ = s.pg.QueryRow(ctx, sql, args...).Scan(&n)
		return n
	}
	qaOpenOrders := count(`
		SELECT count(*) FROM orders o JOIN users u ON u.id = o.customer_id
		WHERE u.phone IN ($1, $2) AND o.closed_at IS NULL`, qaStagingPhone, qaStagingPhone2)
	qaWalletBalance := count(`
		SELECT COALESCE(sum(w.balance), 0) FROM wallets w JOIN users u ON u.id = w.user_id
		WHERE u.phone IN ($1, $2)`, qaStagingPhone, qaStagingPhone2)
	qaDenseItems := count(`SELECT count(*) FROM menu_items WHERE name LIKE 'QA\_DENSE%' ESCAPE '\'`)
	qaActiveOffers := count(`SELECT count(*) FROM offers WHERE active AND title LIKE 'QA%'`)
	qaSecondMerchants := count(`SELECT count(*) FROM merchants WHERE name = $1`, qaSecondMerchantName)

	httpx.JSON(w, http.StatusOK, map[string]any{
		"money": map[string]any{
			"checks_total": total, "checks_passed": passed, "failed": failed,
		},
		"residue": map[string]any{
			"qa_open_orders":      qaOpenOrders,
			"qa_wallet_balance":   qaWalletBalance,
			"qa_dense_items":      qaDenseItems,
			"qa_active_offers":    qaActiveOffers,
			"qa_second_merchants": qaSecondMerchants,
		},
	})
}
