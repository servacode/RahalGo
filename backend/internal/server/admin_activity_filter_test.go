package server

// **سجلُّ النشاط يُرشَّح — وتجديدُ الجلسة لا يغرقه.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٥ بعد قياسٍ على الخادم: **ثلاثون من ثمانيةٍ
//  وخمسين سطراً `auth.refresh`** — أكثرُ من نصف السجلّ.)
//
// # ولماذا يُخفى ولا يُحذف
//
// **`auth.refresh` هو الفعلُ الوحيدُ الذي تكتبه الساعةُ لا الإنسان** —
// هاتفٌ تُرك مفتوحاً يكتب سطراً كلَّ دورة تجديدٍ إلى الأبد. **وخمسةٌ
// وعشرون في الصفحة**، فبعد يومٍ تُدفَع الأفعالُ الحقيقيّةُ خارجَ الصفحة
// الأولى — **وسجلٌّ يُفتح لتجد فيه شيئاً واحداً يملؤه المؤقّت.**
//
// **وهو أثرُ أمانٍ بعنوانٍ ووقت** — فيُخفى ويُطلب، ولا يُمحى.
//
// # وجردُ الأفعال قبل الترشيح
//
// **وقائمةٌ تُبنى من المعروض تخسر خياراتِها كلَّما رُشِّحت** — فلا يُرجَع
// منها إلى ما قبلها، **ويعلق من رشَّح مرّةً في ترشيحه.**

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// TestActivity_HidesRefreshAndFilters **يُخفى · ويُطلب · ويُرشَّح.**
func TestActivity_HidesRefreshAndFilters(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	user, _, _ := twoCustomers(t, f)
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(),
			`DELETE FROM audit_log WHERE actor_user_id = $1`, user)
	})

	mk := func(action string, n int) {
		for range n {
			if _, err := f.pool.Exec(ctx, `
				INSERT INTO audit_log (actor_user_id, action, entity, entity_id, ip, details)
				VALUES ($1::uuid, $2, 'user', $1::text, '1.1.1.1', '{}'::jsonb)`,
				user, action); err != nil {
				t.Fatalf("تعذّر سطرُ %q: %v", action, err)
			}
		}
	}
	mk("auth.refresh", 30)
	mk("auth.password_login", 2)
	mk("user.rename_self", 1)

	type res struct {
		Data struct {
			Total    int `json:"total"`
			Activity []struct {
				Action string `json:"action"`
			} `json:"activity"`
			Kinds []struct {
				Action string `json:"action"`
				Count  int    `json:"count"`
			} `json:"kinds"`
		} `json:"data"`
	}
	get := func(query string) res {
		req := httptest.NewRequest(http.MethodGet, "/x?"+query, nil)
		rc := chi.NewRouteContext()
		rc.URLParams.Add("id", user)
		c := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
		c = context.WithValue(c, ctxUserID, user)
		c = context.WithValue(c, ctxRoles, []string{"admin"})
		w := httptest.NewRecorder()
		f.srv.handleAdminUserActivity(w, req.WithContext(c))
		var out res
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatalf("ردٌّ لا يُفكّ: %v — %s", err, w.Body.String())
		}
		return out
	}

	// ── مخفيٌّ افتراضاً ───────────────────────────────────────────────
	def := get("page=1")
	if def.Data.Total != 3 {
		t.Fatalf("المجموعُ %d لا 3 — **والتجديدُ يغرق السجلَّ الذي يُفتح لغيره**",
			def.Data.Total)
	}
	for _, a := range def.Data.Activity {
		if a.Action == "auth.refresh" {
			t.Fatal("ظهر تجديدُ الجلسة بلا طلب — **تكتبه الساعةُ لا الإنسان**")
		}
	}

	// **وجردُ الأفعال يذكره** — **وما لا يُذكر لا يُطلب**، فيُخفى بلا باب.
	seen := map[string]int{}
	for _, k := range def.Data.Kinds {
		seen[k.Action] = k.Count
	}
	if seen["auth.refresh"] != 30 {
		t.Fatalf("الجردُ يقول %d تجديداً لا 30 — **والقائمةُ تُبنى منه**",
			seen["auth.refresh"])
	}
	if seen["auth.password_login"] != 2 || seen["user.rename_self"] != 1 {
		t.Fatalf("جردٌ ناقص: %v", seen)
	}

	// ── ويُطلب فيظهر ─────────────────────────────────────────────────
	all := get("page=1&refresh=true")
	if all.Data.Total != 33 {
		t.Fatalf("طُلب التجديدُ فردَّ %d لا 33 — **وأثرُ الأمان يُخفى ولا يُمحى**",
			all.Data.Total)
	}

	// ── ويُرشَّح بالفعل ───────────────────────────────────────────────
	one := get("page=1&action=auth.password_login")
	if one.Data.Total != 2 {
		t.Fatalf("الترشيحُ ردَّ %d لا 2", one.Data.Total)
	}
	for _, a := range one.Data.Activity {
		if a.Action != "auth.password_login" {
			t.Fatalf("مرَّ فعلٌ آخرُ في الترشيح: %s", a.Action)
		}
	}
	// **والجردُ لا يتبع الترشيح** — **وإلّا علق من رشَّح مرّةً في ترشيحه.**
	if len(one.Data.Kinds) != 3 {
		t.Fatalf("الجردُ صار %d بعد الترشيح — **فلا يُرجَع منه إلى ما قبله**",
			len(one.Data.Kinds))
	}
}
