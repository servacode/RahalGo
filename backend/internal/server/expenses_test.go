package server

// **المصروفُ يخرج من الخزينة — ولا يُخلط بالخسائر.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٦: «يوجد مكتبٌ للشركة وموظّفون… يجب أن تُوثَّق»
//  · «نعم يخرج من خزينة المنصّة».)
//
// # وما يُحرَس
//
// **قاعدةُ المالك ٢٠٢٦-٠٨-٠٤**: «لا يُدفع لأحدٍ إلّا وخرج من الخزينة».
// **فصفٌّ بلا قيدٍ يجعل الشاشةَ تقول ما لا تقوله الخزينة** — ورصيدٌ يخالف
// دفترَه لا يُصدَّق أيُّهما.
//
// **ولا يُخلط بالخسائر**: `operating_expense` لا `platform_expense` —
// **وشاشةُ الخسائر تقرأ الثاني كلَّه**. **وإيجارُ المكتب ليس خسارة**، ولو
// خُلطا لَتضخّم تقريرُ الخسائر بالإيجار **فيبدو أداءُ المنصّة أسوأَ ممّا
// هو**، ولا يُعرف كم كلّف الفشلُ فعلاً.
//
// **والخطأُ يُلغى ولا يُمحى** — **وحذفُ الصفّ يترك قيدَه في الخزينة بلا
// صاحب**: مالٌ خرج ولا يُعرف لماذا. **ولا يُلغى مرّتين** وإلّا رُدّ المالُ
// ضِعفَه.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func TestExpenses_LeaveTreasuryAndStayOutOfLosses(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	actor, _, _ := twoCustomers(t, f)

	// ══════════════════════════════════════════════════════════════════
	// **والخزينةُ تُنشأ إن لم تكن — ولا يُتخطّى الفحص**
	// ══════════════════════════════════════════════════════════════════
	//
	// **وفحصٌ يتخطّى لا يُثبت شيئاً**: كتبتُه أوّلَ مرّةٍ بـ`t.Skip` فمرَّ
	// خضراءَ وهو لا يقيس، **ثمّ عطّلتُ نوعَ القيد عمداً فمرَّ أيضاً** —
	// وهو ما كشف أنّه لم يكن يعمل.
	//
	// **وأخضرُ لا يقيس أسوأُ من أحمرَ يقيس.**
	treasury := f.srv.orders.TreasuryID(ctx)
	if treasury == "" {
		treasury = testdb.NewUser(t, f.pool, "admin")
		// **ومحفظتُه تُنشأ مع الحساب** — فتُوسَم ولا تُدرَج.
		if _, err := f.pool.Exec(ctx, `
			INSERT INTO wallets (user_id, balance, is_treasury) VALUES ($1, 0, true)
			ON CONFLICT (user_id) DO UPDATE SET is_treasury = true`,
			treasury); err != nil {
			t.Fatalf("تعذّرت الخزينة: %v", err)
		}
		t.Cleanup(func() {
			_, _ = f.pool.Exec(context.Background(),
				`DELETE FROM wallets WHERE user_id = $1`, treasury)
		})
	}
	before, err := f.srv.wallet.Balance(ctx, treasury)
	if err != nil {
		t.Fatalf("تعذّر الرصيد: %v", err)
	}

	var catID string
	if err := f.pool.QueryRow(ctx,
		`SELECT id::text FROM expense_categories WHERE name = 'إيجار'`).Scan(&catID); err != nil {
		t.Fatalf("لا بابَ «إيجار» — والهجرةُ تزرعه: %v", err)
	}
	t.Cleanup(func() {
		c := context.Background()
		_, _ = f.pool.Exec(c, `DELETE FROM wallet_transactions WHERE kind = 'operating_expense'`)
		_, _ = f.pool.Exec(c, `DELETE FROM expenses`)
	})

	post := func(path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(body))
		rc := chi.NewRouteContext()
		if i := strings.Index(path, "|"); i >= 0 {
			rc.URLParams.Add("id", path[i+1:])
		}
		c := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
		c = context.WithValue(c, ctxUserID, actor)
		c = context.WithValue(c, ctxRoles, []string{"admin"})
		w := httptest.NewRecorder()
		if strings.HasPrefix(path, "void|") {
			f.srv.handleVoidExpense(w, req.WithContext(c))
		} else {
			f.srv.handleCreateExpense(w, req.WithContext(c))
		}
		return w
	}

	// ── ١ · يُسجَّل فينقص رصيدُ الخزينة ──────────────────────────────
	w := post("create", `{"category_id":"`+catID+`","amount":300000,"note":"إيجار آب","spent_at":"2026-08-01"}`)
	if w.Code != 201 {
		t.Fatalf("ردَّ %d — %s", w.Code, w.Body.String())
	}
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("ردٌّ لا يُفكّ: %v", err)
	}
	after, _ := f.srv.wallet.Balance(ctx, treasury)
	if after != before-300_000 {
		t.Fatalf("الرصيدُ %d وكان %d — **وصفٌّ بلا قيدٍ يجعل الشاشةَ تقول ما لا تقوله الخزينة**",
			after, before)
	}

	// ── ٢ · ولا يظهر في الخسائر ─────────────────────────────────────
	//
	// **وشاشةُ الخسائر تقرأ `platform_expense`** — **وإيجارُ المكتب ليس
	// خسارة.**
	var losses int64
	if err := f.pool.QueryRow(ctx, `
		SELECT COALESCE(sum(-t.amount), 0)
		FROM wallet_transactions t
		JOIN wallets w ON w.user_id = t.user_id
		WHERE w.is_treasury AND t.kind = 'platform_expense'`).Scan(&losses); err != nil {
		t.Fatalf("تعذّر عدُّ الخسائر: %v", err)
	}
	if losses != 0 {
		t.Fatalf("دخل المصروفُ تقريرَ الخسائر: %d — **فيتضخّم بالإيجار ولا يُعرف كم كلّف الفشل**",
			losses)
	}

	// ── ٣ · ويُلغى فيُردّ المال ─────────────────────────────────────
	if v := post("void|"+created.Data.ID, ""); v.Code != 200 {
		t.Fatalf("تعذّر الإلغاء: %d — %s", v.Code, v.Body.String())
	}
	back, _ := f.srv.wallet.Balance(ctx, treasury)
	if back != before {
		t.Fatalf("بعد الإلغاء %d وكان %d", back, before)
	}
	// **ولا يُلغى مرّتين** — **وإلّا رُدّ المالُ ضِعفَه إلى الخزينة.**
	if v := post("void|"+created.Data.ID, ""); v.Code == 200 {
		t.Fatal("أُلغيَ مرّتين — **فيُردّ المالُ ضِعفَه**")
	}
	twice, _ := f.srv.wallet.Balance(ctx, treasury)
	if twice != before {
		t.Fatalf("الرصيدُ بعد الإلغاء الثاني %d — **ومالٌ يُردّ مرّتين يُقرأ دخلاً**", twice)
	}

	// **والملغى لا يُعدّ في المجموع** — **وقيدٌ أُلغيَ ويُعدّ مالٌ يُحسب مرّتين.**
	req := httptest.NewRequest(http.MethodGet, "/x?from=2026-08-01&to=2026-08-31", nil)
	c := context.WithValue(req.Context(), ctxRoles, []string{"admin"})
	lw := httptest.NewRecorder()
	f.srv.handleListExpenses(lw, req.WithContext(c))
	// ══════════════════════════════════════════════════════════════════
	// **والحالةُ تُفحص قبل الأرقام**
	// ══════════════════════════════════════════════════════════════════
	//
	// **وبدونها كان هذا الفحصُ أخضرَ والقسمُ معطوب**: ردَّ الخادمُ خطأً
	// (`missing FROM-clause entry for table "u"`)، **فجاء `total` صفراً في
	// غلاف الخطأ — وهو ما ينتظره الفحصُ تماماً.**
	//
	// **فمرَّ على ردٍّ ٥٠٠**، وكشفه المالكُ على شاشته لا الفحصُ (٢٠٢٦-٠٨-١٦).
	if lw.Code != 200 {
		t.Fatalf("القائمةُ ردَّت %d — %s", lw.Code, lw.Body.String())
	}
	var list struct {
		Data struct {
			Total       int   `json:"total"`
			TotalAmount int64 `json:"total_amount"`
		} `json:"data"`
	}
	if err := json.Unmarshal(lw.Body.Bytes(), &list); err != nil {
		t.Fatalf("ردٌّ لا يُفكّ: %v", err)
	}
	if list.Data.Total != 0 || list.Data.TotalAmount != 0 {
		t.Fatalf("الملغى يُعدّ: %d صفّاً و%d مبلغاً",
			list.Data.Total, list.Data.TotalAmount)
	}
}
