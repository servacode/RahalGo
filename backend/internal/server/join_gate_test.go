package server

// **لا يُسجَّل متجرٌ بلا كودِ مندوبٍ فعّال.**
//
// (سياسةُ المالك ٢٠٢٦-٠٨-١٧: «ما يصير شخصٌ يفتح الرابطَ بشكلٍ خارجيّ —
//  ربّما حدا استطاع الوصولَ إليه أو خمّن الرابطَ ويصير يسجّل».)
//
// # ولماذا فحصٌ في المحرّك لا في الشاشة
//
// **الشاشةُ تُخفي النموذجَ والنقطةُ تبقى مفتوحة** — ومن نادى العنوانَ
// بأداةٍ سطريّةٍ سجّل متجراً بلا أن يفتح صفحة. **وحجبُ الزرّ ليس حجباً.**
//
// **وهذا الفحصُ ينادي النقطةَ كما يناديها الغريب**: بلا كود، وبكودٍ
// مخترَع، وبكودٍ صحيح.

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func TestPublicJoin_NeedsActiveRepCode(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()

	post := func(ref string) int {
		body, _ := json.Marshal(map[string]any{
			"ref":        ref,
			"store_name": "متجرُ فحص",
			"owner_name": "صاحبُه",
			"phone":      "+963900111222",
			"password":   "Rahal@2026",
		})
		req := httptest.NewRequest(http.MethodPost, "/x", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		f.srv.handlePublicJoin(w, req)
		return w.Code
	}

	// **بلا كودٍ أصلاً** — وهو حالُ من خمّن العنوان.
	if code := post(""); code != http.StatusForbidden {
		t.Fatalf("بلا كود ردَّ %d لا 403 — **والبابُ مفتوحٌ لمن خمّنه**", code)
	}

	// **وبكودٍ مخترَع** — ومن جرّب حروفاً لا يدخل.
	if code := post("RH-ZZZZZ"); code != http.StatusForbidden {
		t.Fatalf("بكودٍ مخترَعٍ ردَّ %d لا 403", code)
	}

	// ══════════════════════════════════════════════════════════════════
	// **وبكودِ مندوبٍ فعّالٍ يمرّ**
	// ══════════════════════════════════════════════════════════════════
	//
	// **وقفلٌ لا يُفتح بمفتاحه ليس قفلاً بل عطب** — فيُفحص الطرفان.
	// **والكودُ يُشتقّ من معرّف المندوب لا يُثبَّت.**
	//
	// **وثابتٌ يسقط في التشغيلة الثانية**: القيدُ فريدٌ، **وصفٌّ خلّفته
	// تشغيلةٌ سابقةٌ يحجز الكودَ فلا يُمنح لغيره.** (وقع في أوّل تشغيلةٍ
	// كاملة.)
	rep := testdb.NewUser(t, f.pool, "sales")
	var code string
	if err := f.pool.QueryRow(ctx, `
		UPDATE users SET invite_code = 'RH-' || upper(substr(replace($1::text,'-',''), 1, 6)),
		                 status = 'active'
		WHERE id = $1 RETURNING invite_code`, rep).Scan(&code); err != nil {
		t.Fatalf("تعذّر كودُ المندوب: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(),
			`DELETE FROM leads WHERE phone = '+963900111222'`)
	})
	if got := post(code); got == http.StatusForbidden {
		t.Fatalf("كودٌ فعّالٌ رُدّ بـ403 — **والقفلُ يمنع صاحبَ المفتاح**")
	}
}
