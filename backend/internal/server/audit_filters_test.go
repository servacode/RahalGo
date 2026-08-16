package server

// **سجلُّ الأحداث يُقلَّب ويُرشَّح بتاريخ — والتجديدُ لا يغرقه.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٦.)
//
// # وثلاثةٌ في أسرع جدولٍ نموّاً
//
// **كلُّ دخولٍ وتعديلِ إعدادٍ وإنذارٍ وقيدٍ يدويٍّ سطرٌ فيه.**
//
// **وكان يُقرأ منه آخرُ مئتين بلا عدٍّ ولا ترقيم** — والردُّ مصفوفةٌ
// مجرّدة، **لا حقلَ فيه يقول «هناك أكثر»**. **وسجلٌّ يصمت عن الباقي
// لمحةٌ باسم سجلّ.**
//
// **و`auth.refresh` يغرقه** — قِيس على خادم المالك: **أربعةٌ وأربعون من
// ثمانين سطراً**، خمسةٌ وخمسون بالمئة. **وتكتبه الساعةُ لا الإنسان.**
// **وأُصلح هذا بعينه في سجلّ نشاط الحساب أمس والطاولةُ نفسُها.**
//
// **ولا مدًى بالتاريخ** — **وسجلٌّ بلا تاريخٍ يُقلَّب لا يُبحَث**: من سأل
// «ماذا جرى الأسبوع الماضي؟» لم يكن له بابٌ إلّا التقليب.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// today **يومٌ بصيغة القاعدة** — والإزاحةُ بالأيام.
func today(offset int) string {
	return time.Now().AddDate(0, 0, offset).Format("2006-01-02")
}

func TestAdminAudit_PagesHidesRefreshAndFiltersByDate(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	actor, _, _ := twoCustomers(t, f)

	if _, err := f.pool.Exec(ctx, `DELETE FROM audit_log`); err != nil {
		t.Fatalf("تعذّر الإخلاء: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM audit_log`)
	})

	mk := func(action string, n, daysAgo int) {
		for range n {
			if _, err := f.pool.Exec(ctx, `
				INSERT INTO audit_log (actor_user_id, action, entity, entity_id, ip, details, created_at)
				VALUES ($1::uuid, $2, 'user', $1::text, '1.1.1.1', '{}'::jsonb,
				        now() - ($3::int || ' days')::interval)`,
				actor, action, daysAgo); err != nil {
				t.Fatalf("تعذّر السطر: %v", err)
			}
		}
	}
	mk("auth.refresh", 60, 0)
	mk("finance.wallet_apply", 3, 0)
	// **وقديمٌ خارجَ المدى** — يُقصى بالتاريخ لا بالتقليب.
	mk("finance.wallet_apply", 5, 30)

	type res struct {
		Data struct {
			Total   int `json:"total"`
			PerPage int `json:"per_page"`
			Entries []struct {
				Action string `json:"action"`
			} `json:"entries"`
		} `json:"data"`
	}
	get := func(q string) res {
		req := httptest.NewRequest(http.MethodGet, "/x?"+q, nil)
		c := context.WithValue(req.Context(), ctxRoles, []string{"admin"})
		w := httptest.NewRecorder()
		f.srv.handleAdminAudit(w, req.WithContext(c))
		if w.Code != 200 {
			t.Fatalf("ردَّ %d — %s", w.Code, w.Body.String())
		}
		var out res
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatalf("ردٌّ لا يُفكّ: %v", err)
		}
		return out
	}

	// ── ١ · التجديدُ مخفيٌّ افتراضاً ─────────────────────────────────
	def := get("limit=50&page=1")
	if def.Data.Total != 8 {
		t.Fatalf("المجموعُ %d لا 8 — **والتجديدُ يغرق السجلَّ الذي يُفتح لغيره**",
			def.Data.Total)
	}
	for _, e := range def.Data.Entries {
		if e.Action == "auth.refresh" {
			t.Fatal("ظهر التجديدُ بلا طلب — **تكتبه الساعةُ لا الإنسان**")
		}
	}

	// ── ٢ · ويُطلب فيظهر ────────────────────────────────────────────
	all := get("limit=50&page=1&refresh=true")
	if all.Data.Total != 68 {
		t.Fatalf("طُلب التجديدُ فردَّ %d لا 68 — **وأثرُ الأمان يُخفى ولا يُمحى**",
			all.Data.Total)
	}
	// **وصفحةٌ محدودةٌ** — والباقي بالتقليب لا بالصمت.
	if len(all.Data.Entries) != 50 {
		t.Fatalf("الصفحةُ %d لا 50", len(all.Data.Entries))
	}
	second := get("limit=50&page=2&refresh=true")
	if len(second.Data.Entries) != 18 {
		t.Fatalf("الصفحةُ الثانية %d لا 18 — **والباقي بلا باب**",
			len(second.Data.Entries))
	}

	// ── ٣ · والمدى بالتاريخ يُقصي القديم ────────────────────────────
	//
	// **وسجلٌّ بلا تاريخٍ يُقلَّب لا يُبحَث.**
	recent := get("limit=50&page=1&from=" + today(-1))
	if recent.Data.Total != 3 {
		t.Fatalf("مدى اليومين ردَّ %d لا 3 — **فيُقلَّب الشهرُ كلُّه لسؤالٍ عن أمس**",
			recent.Data.Total)
	}
}
