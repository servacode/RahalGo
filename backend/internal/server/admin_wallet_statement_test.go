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
	"testing"
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

	// ══════════════════════════════════════════════════════════════════
	// **وفوق الخمسين يقول إنّه قُصّ**
	// ══════════════════════════════════════════════════════════════════
	//
	// **والشاشةُ لا تقرأ هذا الحقل اليوم** — فالدفترُ يَنقُص عندها صامتاً.
	// **وهذا يثبّت أنّ المحرّك يقولها** كي يُبنى عليه.
	for i := range 50 {
		mk(owner, 1000, "topup", "حشوٌ للسقف")
		_ = i
	}
	w2 := asCustomer(f.srv.handleAdminWalletStatement, http.MethodGet, owner, owner)
	var env2 struct {
		Data struct {
			Truncated    bool       `json:"truncated"`
			Transactions []struct{} `json:"transactions"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &env2); err != nil {
		t.Fatalf("ردٌّ لا يُفكّ: %v", err)
	}
	if len(env2.Data.Transactions) != 50 {
		t.Fatalf("عُرضت %d حركةً والسقفُ خمسون", len(env2.Data.Transactions))
	}
	if !env2.Data.Truncated {
		t.Fatal("قُصّ الدفترُ ولم يقل — **ونقصٌ لا يُعلَن يُقرأ كمالاً**")
	}
}
