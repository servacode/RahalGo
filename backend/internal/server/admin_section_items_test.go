package server

// **بحثُ القسم وترشيحُه وعدّاده — في المحرّك لا في الشاشة.**
//
// (كشفه فحصُ المالك ٢٠٢٦-٠٨-١٦.)
//
// # ثلاثةُ أعطابٍ من عائلةٍ واحدة
//
// **البحثُ كان يعمل على الصفحة المعروضة وحدَها** — فمن بحث عن صنفٍ في
// الصفحة الثالثة **قرأ «لا أصناف»**. **وبحثٌ يقول «غيرُ موجود» عمّا هو
// موجودٌ أخطرُ من رقمٍ يكذب**: فيُضاف الصنفُ مرّتين.
//
// **والترشيحُ بالحال مثلُه** — «أرِني ما ينتظر المراجعة» يردّ ما في
// الصفحة الحاليّة وحدَها.
//
// **والبطاقاتُ كانت تعدّ الصفحة**: قسمٌ فيه ثلاثمئة يقول «الكلّ: ٥٠»،
// **والترقيمُ أسفلَه يقول «١ / ٦»** — رقمان متناقضان في شاشةٍ واحدة.
//
// # والبطاقتان لا تتبعان مُرشِّحَ الحال
//
// **وبطاقةٌ تتبعه تقول «المعروضُ صفر» لمن رشّح «ينتظر المراجعة»** — وهي
// لا تخصّه: **سؤالُها عن القسم لا عن الترشيح.**

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestSectionItems_ServerSideSearchFilterAndCounts(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()

	var sectionID string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO platform_sections (name, sort_order) VALUES ('قسمُ الفحص', 900)
		RETURNING id::text`).Scan(&sectionID); err != nil {
		t.Fatalf("تعذّر القسم: %v", err)
	}
	var menuSectionID string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO menu_sections (merchant_id, name) VALUES ($1, 'قائمةُ الفحص')
		RETURNING id::text`, f.merchantID).Scan(&menuSectionID); err != nil {
		t.Fatalf("تعذّرت قائمةُ المتجر: %v", err)
	}
	t.Cleanup(func() {
		c := context.Background()
		_, _ = f.pool.Exec(c, `DELETE FROM menu_items WHERE platform_section_id = $1`, sectionID)
		_, _ = f.pool.Exec(c, `DELETE FROM menu_sections WHERE id = $1`, menuSectionID)
		_, _ = f.pool.Exec(c, `DELETE FROM platform_sections WHERE id = $1`, sectionID)
	})

	// **سبعون صنفاً** — أكثرُ من صفحةٍ واحدة (خمسون)، **فالصفحةُ الثانيةُ
	// هي ما كان يختفي عن البحث.**
	//
	// **وواحدٌ فيها ينتظر المراجعة** — واسمُه فريدٌ ليُبحث عنه.
	for i := range 70 {
		name := fmt.Sprintf("صنفٌ %02d", i)
		approved := true
		if i == 65 {
			name, approved = "شاورما عربي", false
		}
		if _, err := f.pool.Exec(ctx, `
			INSERT INTO menu_items (merchant_id, section_id, platform_section_id,
				name, price, merchant_price, approved, available)
			VALUES ($1, $2, $3, $4, 1000, 1000, $5, true)`,
			f.merchantID, menuSectionID, sectionID, name, approved); err != nil {
			t.Fatalf("تعذّر الصنف %d: %v", i, err)
		}
	}

	type res struct {
		Data struct {
			Count int `json:"count"`
			All   int `json:"all"`
			Live  int `json:"live"`
			Items []struct {
				Name string `json:"name"`
			} `json:"items"`
		} `json:"data"`
	}
	get := func(query string) res {
		req := httptest.NewRequest(http.MethodGet, "/x?"+query, nil)
		rc := chi.NewRouteContext()
		rc.URLParams.Add("id", sectionID)
		c := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
		c = context.WithValue(c, ctxRoles, []string{"admin"})
		w := httptest.NewRecorder()
		f.srv.handleSectionItems(w, req.WithContext(c))
		if w.Code != 200 {
			t.Fatalf("ردَّ %d — %s", w.Code, w.Body.String())
		}
		var out res
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatalf("ردٌّ لا يُفكّ: %v", err)
		}
		return out
	}

	// ── ١ · البطاقاتُ تعدّ القسمَ لا الصفحة ──────────────────────────
	first := get("page=1")
	if len(first.Data.Items) != 50 {
		t.Fatalf("الصفحةُ %d صفّاً لا 50", len(first.Data.Items))
	}
	if first.Data.All != 70 {
		t.Fatalf("«الكلّ» %d لا 70 — **ورقمان متناقضان في شاشةٍ واحدة**", first.Data.All)
	}
	if first.Data.Live != 69 {
		t.Fatalf("«المعروض» %d لا 69 — **والمعروضُ فعلاً لا المسجَّل**", first.Data.Live)
	}

	// ── ٢ · والبحثُ يبلغ الصفحةَ الثانية ────────────────────────────
	//
	// **وهو ما كان يردّ «لا أصناف»** — والصنفُ في الصفّ ٦٥.
	found := get("page=1&q=" + "شاورما")
	if found.Data.Count != 1 || len(found.Data.Items) != 1 ||
		found.Data.Items[0].Name != "شاورما عربي" {
		t.Fatalf("البحثُ ردَّ %d — **فيُقال «غيرُ موجود» عمّا هو موجودٌ فيُضاف مرّتين**",
			found.Data.Count)
	}
	// **والبطاقتان لا تتبعان الترشيحَ بالحال** — لكنّهما تتبعان البحث:
	// **سؤالُ «كم في القسم؟» عن القسم، وسؤالُ «كم وجدتُ؟» عن البحث.**
	if found.Data.All != 1 {
		t.Fatalf("«الكلّ» مع بحثٍ %d لا 1", found.Data.All)
	}

	// ── ٣ · والترشيحُ بالحال يعمل على القسم كلِّه ────────────────────
	pending := get("page=1&state=pending")
	if pending.Data.Count != 1 || pending.Data.Items[0].Name != "شاورما عربي" {
		t.Fatalf("«ينتظر المراجعة» ردَّ %d — **وترشيحٌ فوق صفحةٍ وعدٌ بترشيح**",
			pending.Data.Count)
	}
	// **وبطاقتُه لا تتبعه** — **وإلّا قالت «المعروضُ صفر» وهي لا تخصّ ترشيحَه.**
	if pending.Data.All != 70 || pending.Data.Live != 69 {
		t.Fatalf("البطاقتان تبعتا الترشيح: %d/%d — **وسؤالُهما عن القسم**",
			pending.Data.All, pending.Data.Live)
	}

	live := get("page=1&state=live")
	if live.Data.Count != 69 {
		t.Fatalf("«معروض» ردَّ %d لا 69", live.Data.Count)
	}
}
