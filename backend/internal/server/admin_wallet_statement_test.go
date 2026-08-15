package server

// **سجلُّ المحفظة في الملفّ — حركاتُ صاحبه وحدَه، وأحدثُها أوّلاً.**
//
// (سؤالُ المالك ٢٠٢٦-٠٨-١٥: «سجلّ المحفظة نتأكّد من عمله».)
//
// # وما يُحرَس
//
// **دفترُ مالٍ يُقرأ في ملفّ إنسان** — وشرطُ `user_id` هو ما يفصل دفتراً
// عن دفتر. **ومن خلطهما لم يُخطئ الجمعُ في الشاشة**: تظهر حركاتٌ أكثر،
// **فيُقرأ «هذا كثيرُ الشحن»** وهو مالُ غيره.
//
// **والترتيبُ بالمعرّف لا بالوقت** (`t.id` عدّادٌ متزايد) — **وحركتان في
// الثانية نفسِها يتأرجحان** لو رُتّبتا بالوقت، فيُقرأ الشحنُ بعد الدفع.
//
// # والسقفُ يُعلَن ولا يُقرأ
//
// **المحرّكُ يقصّ عند خمسين ويقول `truncated`** — «كشفٌ ناقصٌ يجب أن يقول
// إنّه ناقص»، هكذا كُتب فيه. **والشاشةُ تقرأ `transactions` وحدَها**
// فتُسقط الإعلان: **دفترٌ يَنقُص صامتاً.** (وهذا الفحصُ يثبّت أنّ المحرّك
// يقولها، ليُبنى عليه في الشاشة.)

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
)

// TestAdminWalletStatement_OwnRowsNewestFirstAndSaysWhenCut
// **دفترُه هو · أحدثُه أوّلاً · ويقول إن قُصّ.**
func TestAdminWalletStatement_OwnRowsNewestFirstAndSaysWhenCut(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	owner, other, _ := twoCustomers(t, f)

	mk := func(user string, amount int64, kind, note string) {
		if _, err := f.pool.Exec(ctx, `
			INSERT INTO wallet_transactions (user_id, amount, kind, note)
			VALUES ($1, $2, $3, $4)`, user, amount, kind, note); err != nil {
			t.Fatalf("تعذّرت الحركة %q: %v", note, err)
		}
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(),
			`DELETE FROM wallet_transactions WHERE user_id = ANY($1)`, []string{owner, other})
	})

	// **واحدةٌ لجاره أوّلاً** — فلو تسرّبت لَظهرت في آخر القائمة.
	mk(other, 999000, "topup", "شحنٌ لا يخصّ هذا الملفّ")
	mk(owner, 50000, "topup", "الأولى")
	mk(owner, -12000, "order_payment", "الثانية")
	mk(owner, 3000, "compensation", "الثالثة")

	w := asCustomer(f.srv.handleAdminWalletStatement, http.MethodGet, owner, owner)
	if w.Code != 200 {
		t.Fatalf("ردَّ %d — **ودفترٌ لا يُقرأ يُقرأ «لا حركةَ له»**", w.Code)
	}
	// **والردُّ في ظرفٍ** (`{"data":…}`) — وهو ما يفكّه عميلُ الويب،
	// **ومن قرأ الجذرَ قرأ فراغاً** ولا خطأ يدلّه.
	var env struct {
		Data struct {
			Balance      int64 `json:"balance"`
			Truncated    bool  `json:"truncated"`
			Total        int   `json:"total"`
			Page         int   `json:"page"`
			Opening      int64 `json:"opening"`
			Closing      int64 `json:"closing"`
			Transactions []struct {
				ID   int64  `json:"id"`
				Kind string `json:"kind"`
				Note string `json:"note"`
			} `json:"transactions"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("ردٌّ لا يُفكّ: %v — %s", err, w.Body.String())
	}

	if len(env.Data.Transactions) != 3 {
		t.Fatalf("%d حركةً لا 3 — **والزائدُ مالُ غيره والناقصُ نقصٌ صامت**",
			len(env.Data.Transactions))
	}
	// **ولا حركةَ لغيره** — وهو ما يحرسه شرطُ `user_id` وحدَه.
	for _, x := range env.Data.Transactions {
		if x.Note == "شحنٌ لا يخصّ هذا الملفّ" {
			t.Fatal("ظهرت حركةُ مستخدمٍ آخرَ في دفتر هذا الملفّ — **فيُقرأ مالُ غيره ماله**")
		}
	}
	// **وأحدثُها أوّلاً** — من فتح الدفتر يسأل «ماذا وقع الآن»، لا «ماذا وقع أوّلَ مرّة».
	if env.Data.Transactions[0].Note != "الثالثة" || env.Data.Transactions[2].Note != "الأولى" {
		t.Fatalf("الترتيبُ مقلوب: %q ثمّ %q — **فيُقرأ الشحنُ بعد الدفع**",
			env.Data.Transactions[0].Note, env.Data.Transactions[2].Note)
	}
	// **والنوعُ رمزٌ يُترجَم في الشاشة** — ومعجمُها يحمل السبعةَ كلَّها.
	if env.Data.Transactions[1].Kind != "order_payment" {
		t.Fatalf("النوعُ %q لا order_payment", env.Data.Transactions[1].Kind)
	}
	if env.Data.Truncated {
		t.Fatal("قال «قُصّ» وفيه ثلاثُ حركات — **وإنذارٌ كاذبٌ يُطفأ فيُفقد ما يحرسه**")
	}

	// **والمجموعُ ثلاثةٌ وصفحةٌ واحدة** — ولا شيءَ وراءها.
	if env.Data.Total != 3 || env.Data.Page != 1 {
		t.Fatalf("المجموعُ %d والصفحةُ %d — **والترقيمُ في الشاشة يُبنى عليهما**",
			env.Data.Total, env.Data.Page)
	}
	// **والافتتاحيُّ + المعروضُ = الختاميّ** — دفترٌ لا يتوازن يُقرأ خللاً
	// في المنصّة لا في القراءة.
	if env.Data.Opening+41000 != env.Data.Closing {
		t.Fatalf("لم يتوازن: افتتاحيٌّ %d وختاميٌّ %d ومعروضٌ 41000",
			env.Data.Opening, env.Data.Closing)
	}
}

// TestAdminWalletStatement_PagesInsteadOfCutting
// **يُقلَّب ولا يُقصّ — وكلُّ صفحةٍ تتوازن برصيدها هي.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٥: «أصلحها».)
//
// **وكان يردّ خمسين ويقول `truncated`** — **والشاشةُ لا تقرأ الحقل**، فمن
// راجع دفترَ زبونٍ اشتكى «رصيدي ناقص» رأى خمسين حركةً **وظنّها كلَّ ما وقع.**
func TestAdminWalletStatement_PagesInsteadOfCutting(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	owner, _, _ := twoCustomers(t, f)
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(),
			`DELETE FROM wallet_transactions WHERE user_id = $1`, owner)
	})

	// **مئةٌ وعشرون حركةً** — ثلاثُ صفحاتٍ وكسر، **وسبعون منها كانت تُحجب.**
	const n = 120
	for range n {
		if _, err := f.pool.Exec(ctx, `
			INSERT INTO wallet_transactions (user_id, amount, kind)
			VALUES ($1, 1000, 'topup')`, owner); err != nil {
			t.Fatalf("تعذّرت الحركة: %v", err)
		}
	}

	type page struct {
		Data struct {
			Total     int   `json:"total"`
			Page      int   `json:"page"`
			PerPage   int   `json:"per_page"`
			Opening   int64 `json:"opening"`
			Closing   int64 `json:"closing"`
			Truncated bool  `json:"truncated"`
			Rows      []struct {
				ID     int64 `json:"id"`
				Amount int64 `json:"amount"`
			} `json:"transactions"`
		} `json:"data"`
	}
	get := func(p int) page {
		req := httptest.NewRequest(http.MethodGet, "/x?page="+strconv.Itoa(p), nil)
		rc := chi.NewRouteContext()
		rc.URLParams.Add("id", owner)
		c := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
		c = context.WithValue(c, ctxUserID, owner)
		c = context.WithValue(c, ctxRoles, []string{"admin"})
		w := httptest.NewRecorder()
		f.srv.handleAdminWalletStatement(w, req.WithContext(c))
		var out page
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatalf("الصفحةُ %d لا تُفكّ: %v — %s", p, err, w.Body.String())
		}
		return out
	}

	seen := map[int64]bool{}
	got := 0
	for p := 1; p <= 3; p++ {
		pg := get(p)
		if pg.Data.Total != n {
			t.Fatalf("الصفحةُ %d تقول المجموعَ %d لا %d", p, pg.Data.Total, n)
		}
		want := 50
		if p == 3 {
			want = 20
		}
		if len(pg.Data.Rows) != want {
			t.Fatalf("الصفحةُ %d فيها %d حركةً لا %d — **والباقي بلا باب**",
				p, len(pg.Data.Rows), want)
		}
		// **ولا حركةَ تُقرأ مرّتين** — صفحتان تتداخلان تُريان مالاً لم يقع.
		for _, x := range pg.Data.Rows {
			if seen[x.ID] {
				t.Fatalf("الحركةُ %d قُرئت مرّتين — **ومالٌ يُعدّ مرّتين يُقرأ ضِعفَه**", x.ID)
			}
			seen[x.ID] = true
			got++
		}
		// ══════════════════════════════════════════════════════════════
		// **وكلُّ صفحةٍ تُقفل برصيدها هي**
		// ══════════════════════════════════════════════════════════════
		//
		// **والصفحةُ الثانيةُ لا تُقفل برصيد اليوم** — ولو فعلت لَخُرقت
		// المعادلة، **فيجمع من يراجعها فلا تُطابق** ويظنّ الخلل عندنا.
		var shown int64
		for _, x := range pg.Data.Rows {
			shown += x.Amount
		}
		if pg.Data.Opening+shown != pg.Data.Closing {
			t.Fatalf("الصفحةُ %d لم تتوازن: %d + %d ≠ %d",
				p, pg.Data.Opening, shown, pg.Data.Closing)
		}
		// **وآخرُ صفحةٍ لا تقول «بقيَ المزيد».**
		if pg.Data.Truncated != (p < 3) {
			t.Fatalf("الصفحةُ %d تقول truncated=%v — **وإنذارٌ كاذبٌ يُطفأ**",
				p, pg.Data.Truncated)
		}
	}
	if got != n {
		t.Fatalf("قُرئ %d من %d — **والنقصُ صامت**", got, n)
	}
}
