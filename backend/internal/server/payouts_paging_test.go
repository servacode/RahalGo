package server

// **طلباتُ السحب تُقلَّب — ومجموعُ المعلَّق يُقال.**
//
// (كشفه فحصُ المالك ٢٠٢٦-٠٨-١٦.)
//
// # مئتان صامتةٌ في المال
//
// **كان `LIMIT 200` بلا عدٍّ ولا ترقيمٍ ولا كلمة** — والردُّ مصفوفةٌ
// مجرّدة، **لا حقلَ فيه يقول «هناك أكثر».**
//
// **والمعلَّقُ يتصدّر فيخفّ الأثر** — لكنّه يبقى في التاريخ: من رشّح
// «مدفوع» ليراجع ما صُرف **يرى آخرَ مئتين ويظنّها كلَّ ما دُفع.**
// **وهذا مالٌ خرج، ومراجعتُه ناقصةً أسوأُ من عدمها.**
//
// # ومجموعٌ لا يتبع الترشيح
//
// **سؤالُه «كم عليّ الآن؟»** — **ومجموعٌ يتبع مُرشِّحاً يقول صفراً لمن
// يقرأ المرفوض**، وهو لا يخصّه.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func TestAdminPayouts_PagesAndSumsPending(t *testing.T) {
	f := newDriverFixture(t, 1)
	ctx := context.Background()
	driver := f.drivers[0]

	if _, err := f.pool.Exec(ctx, `DELETE FROM payout_requests`); err != nil {
		t.Fatalf("تعذّر الإخلاء: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM payout_requests`)
	})

	ask := func(user string, amount int64, status string) {
		if _, err := f.pool.Exec(ctx, `
			INSERT INTO payout_requests (user_id, amount, status)
			VALUES ($1, $2, $3)`, user, amount, status); err != nil {
			t.Fatalf("تعذّر الطلب: %v", err)
		}
	}
	// **ثلاثون معلَّقاً بألفٍ** — أكثرُ من صفحةٍ واحدة (خمسٌ وعشرون).
	//
	// **ولكلٍّ صاحبُه**: القاعدةُ تمنع معلَّقين لشخصٍ واحد
	// (`payout_requests_one_pending`) — **وهي حارسٌ صحيحٌ لا يُلتفّ عليه
	// في فحص.**
	for range 30 {
		ask(testdb.NewUser(t, f.pool, "driver"), 1000, "pending")
	}
	// **ومرفوضٌ ومدفوعٌ لا يدخلان المجموع** — «كم عليّ الآن» لا «كم كان».
	ask(driver, 500_000, "rejected")
	ask(driver, 700_000, "paid")

	type res struct {
		Data struct {
			Total        int   `json:"total"`
			PerPage      int   `json:"per_page"`
			PendingTotal int64 `json:"pending_total"`
			Payouts      []struct {
				Status string `json:"status"`
			} `json:"payouts"`
		} `json:"data"`
	}
	get := func(q string) res {
		req := httptest.NewRequest(http.MethodGet, "/x?"+q, nil)
		c := context.WithValue(req.Context(), ctxRoles, []string{"admin"})
		w := httptest.NewRecorder()
		f.srv.handleAdminPayouts(w, req.WithContext(c))
		if w.Code != 200 {
			t.Fatalf("ردَّ %d — %s", w.Code, w.Body.String())
		}
		var out res
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatalf("ردٌّ لا يُفكّ: %v", err)
		}
		return out
	}

	// ── ١ · يُقلَّب ويُعدّ ────────────────────────────────────────────
	all := get("page=1")
	if all.Data.Total != 32 {
		t.Fatalf("المجموعُ %d لا 32 — **والعدُّ يقول كم هناك لا كم عُرض**",
			all.Data.Total)
	}
	if len(all.Data.Payouts) != all.Data.PerPage {
		t.Fatalf("الصفحةُ %d والحدُّ %d", len(all.Data.Payouts), all.Data.PerPage)
	}
	// **والمعلَّقُ يتصدّر** — ما يحتاج عملاً في الأعلى.
	if all.Data.Payouts[0].Status != "pending" {
		t.Fatalf("تصدّر %q لا المعلَّق", all.Data.Payouts[0].Status)
	}

	// ── ٢ · ومجموعُ المعلَّق وحدَه ────────────────────────────────────
	if all.Data.PendingTotal != 30_000 {
		t.Fatalf("المعلَّقُ %d لا 30000 — **ومالٌ لا يُرى مجموعاً لا يُخطَّط له**",
			all.Data.PendingTotal)
	}

	// ── ٣ · ولا يتبع الترشيح ─────────────────────────────────────────
	//
	// **ومجموعٌ يتبعه يقول صفراً لمن يقرأ المرفوض** وهو لا يخصّه.
	rej := get("page=1&status=rejected")
	if rej.Data.Total != 1 {
		t.Fatalf("المرفوضُ %d لا 1", rej.Data.Total)
	}
	if rej.Data.PendingTotal != 30_000 {
		t.Fatalf("المجموعُ تبع الترشيح: %d — **وسؤالُه «كم عليّ الآن»**",
			rej.Data.PendingTotal)
	}
}
