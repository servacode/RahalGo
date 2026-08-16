package server

// **الأرباحُ تُقرأ من الدفتر — والصافي يطابق الخزينة.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٦.)
//
// # ولماذا رقمان
//
// **`platform_profit` ليس «العمولة»** — هو ما دفعه الزبونُ ناقصَ ما رُدَّ
// ناقصَ ما قُيّد للأطراف: **يحوي الهامشَ والعمولةَ معاً وقد طُرح منه
// الخصمُ أصلاً.**
//
// **فلو جُمع الهامشُ والعمولةُ ثمّ طُرح الخصمُ مرّةً أخرى لَحُسب مرّتين** —
// **ولَخالف الناتجُ رصيدَ الخزينة**، ولا يُعرف أيُّهما يُصدَّق.
//
// **فالتفصيلُ يقول من أين جاء المال، والصافي يُقرأ من الدفتر** — وهذا ما
// يُحرَس هنا: **أنّ الصافيَ = مجموعُ حركة الخزينة**، مهما أُضيف من أنواع.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func TestProfits_NetMatchesTreasuryLedger(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	actor, _, _ := twoCustomers(t, f)

	treasury := f.srv.orders.TreasuryID(ctx)
	if treasury == "" {
		treasury = testdb.NewUser(t, f.pool, "admin")
		if _, err := f.pool.Exec(ctx, `
			INSERT INTO wallets (user_id, balance, is_treasury) VALUES ($1, 0, true)
			ON CONFLICT (user_id) DO UPDATE SET is_treasury = true`, treasury); err != nil {
			t.Fatalf("تعذّرت الخزينة: %v", err)
		}
		t.Cleanup(func() {
			_, _ = f.pool.Exec(context.Background(),
				`DELETE FROM wallets WHERE user_id = $1`, treasury)
		})
	}
	// **وتُخلى حركةُ الخزينة قبل القياس** — **وفحصٌ يقيس ما خلّفه غيرُه لا
	// يقيس شيئا.**
	if _, err := f.pool.Exec(ctx,
		`DELETE FROM wallet_transactions WHERE user_id = $1`, treasury); err != nil {
		t.Fatalf("تعذّر الإخلاء: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(),
			`DELETE FROM wallet_transactions WHERE user_id = $1`, treasury)
	})

	// **ما دخل الخزينةَ وما خرج** — بأنواعه الخمسة.
	//
	// **ويمرّ بالمحفظة لا بإدراجٍ في الجدول** — **وإدراجٌ مباشرٌ يكتب القيدَ
	// ولا يحرّك الرصيد**، فيقيس الفحصُ شيئاً لا يقع في التشغيل. (وقع في
	// أوّل كتابةٍ لهذا الفحص: الصافي ٤٧٠٠٠ والرصيدُ صفر.)
	mk := func(amount int64, kind string) {
		if _, err := f.srv.wallet.Apply(ctx, treasury, amount, kind, "",
			"قيدُ فحص", &actor); err != nil {
			t.Fatalf("تعذّر القيدُ %q: %v", kind, err)
		}
	}
	mk(100_000, "platform_profit")   // دخلٌ من الطلبات
	mk(-20_000, "platform_expense")  // خسارةٌ: تعويضُ سائق
	mk(-30_000, "operating_expense") // مصروفُ مكتب
	mk(-5_000, "reward")             // مكافأةُ دعوة
	mk(2_000, "penalty")             // عقوبةٌ تدخل الخزينة

	req := httptest.NewRequest(http.MethodGet, "/x?tab=platform", nil)
	c := context.WithValue(req.Context(), ctxUserID, actor)
	c = context.WithValue(c, ctxRoles, []string{"admin"})
	w := httptest.NewRecorder()
	f.srv.handleProfits(w, req.WithContext(c))
	if w.Code != 200 {
		t.Fatalf("ردَّ %d — %s", w.Code, w.Body.String())
	}
	var env struct {
		Data struct {
			Losses    int64 `json:"losses"`
			Opex      int64 `json:"opex"`
			Referrals int64 `json:"referrals"`
			Penalties int64 `json:"penalties"`
			Net       int64 `json:"net"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("ردٌّ لا يُفكّ: %v", err)
	}
	d := env.Data

	// **وكلُّ بابٍ بمقداره** — **ومجموعٌ واحدٌ لا يقول أين ذهب المال.**
	if d.Losses != 20_000 || d.Opex != 30_000 || d.Referrals != 5_000 || d.Penalties != 2_000 {
		t.Fatalf("التفصيل: خسائر %d · مصاريف %d · دعوات %d · عقوبات %d",
			d.Losses, d.Opex, d.Referrals, d.Penalties)
	}

	// ══════════════════════════════════════════════════════════════════
	// **والصافي يطابق رصيدَ الخزينة**
	// ══════════════════════════════════════════════════════════════════
	//
	// **وهو ما يجعل الشاشةَ تُصدَّق**: رقمان متناقضان في شاشةٍ واحدةٍ لا
	// يُعرف أيُّهما يُقرأ.
	balance, err := f.srv.wallet.Balance(ctx, treasury)
	if err != nil {
		t.Fatalf("تعذّر الرصيد: %v", err)
	}
	if d.Net != balance {
		t.Fatalf("الصافي %d ورصيدُ الخزينة %d — **ورقمان متناقضان لا يُصدَّق أيُّهما**",
			d.Net, balance)
	}
	if d.Net != 47_000 {
		t.Fatalf("الصافي %d لا 47000 (100000-20000-30000-5000+2000)", d.Net)
	}
}

// ══════════════════════════════════════════════════════════════════════
//
//	**وكلُّ تبويبٍ يُنادى — لا الأوّلُ وحدَه**
//
// ══════════════════════════════════════════════════════════════════════
//
// **وهذا ما لم يكن**: فحصُ الصافي أعلاه ينادي `tab=platform` وحدَه، **وكانت
// تبويباتُ الزبائن والمناديب والسائقين تردّ ٥٠٠** — استعلامُها يشير إلى
// `earned` و`spent` ولم يكونا موصولين في الجسم:
//
//	missing FROM-clause entry for table "e" (SQLSTATE 42P01)
//
// **وكشفه المالكُ على شاشته بعد الرفع لا الفحص** (٢٠٢٦-٠٨-١٦).
//
// # ولماذا فحصٌ لا انتباه
//
// **خمسةُ استعلاماتٍ خلفَ خمس ضغطاتٍ لا يجمعها بناءٌ ولا تحقّقُ أنواع** —
// **وخطأُ SQL لا يظهر إلّا وقتَ النداء.** فمن غيّر عموداً غداً يعرف هنا.
func TestProfits_EveryTabAnswers(t *testing.T) {
	f := newDriverFixture(t, 0)
	actor, _, _ := twoCustomers(t, f)

	for _, tab := range []string{"platform", "customers", "reps", "drivers", "merchants"} {
		req := httptest.NewRequest(http.MethodGet,
			"/x?tab="+tab+"&from=2026-08-01&to=2026-08-31", nil)
		c := context.WithValue(req.Context(), ctxUserID, actor)
		c = context.WithValue(c, ctxRoles, []string{"admin"})
		w := httptest.NewRecorder()
		f.srv.handleProfits(w, req.WithContext(c))
		if w.Code != 200 {
			t.Fatalf("تبويبُ %q ردَّ %d — %s", tab, w.Code, w.Body.String())
		}
		// **ومدًى مفتوحٌ يُجرَّب أيضاً** — **وشرطُ التاريخ يخفي خطأً في
		// فرعٍ لا يُسلك.**
		req2 := httptest.NewRequest(http.MethodGet, "/x?tab="+tab, nil)
		c2 := context.WithValue(req2.Context(), ctxUserID, actor)
		c2 = context.WithValue(c2, ctxRoles, []string{"admin"})
		w2 := httptest.NewRecorder()
		f.srv.handleProfits(w2, req2.WithContext(c2))
		if w2.Code != 200 {
			t.Fatalf("تبويبُ %q بمدًى مفتوحٍ ردَّ %d — %s", tab, w2.Code, w2.Body.String())
		}
	}
}
