package server

// ══════════════════════════════════════════════════════════════════════
// **ولا يعود النصُّ الحرُّ من بابٍ خلفيّ**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-٣٠.)
//
// **والقائمةُ المتدرّجةُ في الشاشة ليست حارساً**: من أرسل الطلبَ بأداةٍ
// سطريّةٍ يتخطّاها، **ونسخةٌ قديمةٌ من التطبيق في جهازِ مندوبٍ لم يحدّث
// ترسل `area` بلا `district_id`** — فتعود «وسط المدينة» صفّاً بلا
// انتماء.
//
// **وحقلٌ يُلزَم في الشاشة ولا يُلزَم في المحرّك ليس إلزاماً.**

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// postRepLead **ينادي بابَ المندوب كما يناديه الموجّه.**
func postRepLead(t *testing.T, f *driverFixture, rep, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/rep/leads", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), ctxUserID, rep)
	w := httptest.NewRecorder()
	f.srv.handleRepCreateLead(w, req.WithContext(ctx))
	return w
}

func TestRepLead_RequiresDistrict(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()

	rep := testdb.NewUser(t, f.pool, "sales")
	// **والتوثيقُ شرطُ الرفع** — يُضبَط كي يُفحص ما جئنا نفحصه لا ما قبله.
	if _, err := f.pool.Exec(ctx,
		`UPDATE users SET whatsapp_verified_at = now() WHERE id = $1`, rep); err != nil {
		t.Logf("تعذّر التوثيق (قد لا يكون العمودُ مطلوباً): %v", err)
	}
	var cat string
	if err := f.pool.QueryRow(ctx, `SELECT id::text FROM categories LIMIT 1`).Scan(&cat); err != nil {
		t.Fatalf("لا تصنيفات: %v", err)
	}
	dist := liveDistrict(t, f)

	base := `{"store_name":"متجرُ فحصٍ","owner_name":"صاحبُه",` +
		`"phone":"0999888777","password":"kalimatSirr9","category_id":"` + cat + `"`

	// ══════════════════════════════════════════════════════════════════
	// **بلا منطقةٍ يُردّ — ولو كتب نصّاً**
	// ══════════════════════════════════════════════════════════════════
	//
	// **و«وسط المدينة» في `area` لا تُغني**: هي عنوانٌ تفصيليٌّ لا موضعٌ
	// يُصنَّف. **ومن قبِلها بديلاً أعاد ما جئنا نمحوه.**
	w := postRepLead(t, f, rep, base+`,"area":"وسط المدينة"}`)
	if w.Code < 400 {
		t.Errorf("طلبٌ بلا منطقةٍ قُبل (%d) — **والنصُّ الحرُّ عاد صفّاً بلا انتماء**",
			w.Code)
		_, _ = f.pool.Exec(ctx, `DELETE FROM merchant_leads WHERE sales_rep_user_id = $1`, rep)
	}

	// **ومعرّفٌ مخترَعٌ يُردّ كذلك.**
	w = postRepLead(t, f, rep, base+`,"district_id":"وسط المدينة"}`)
	if w.Code < 400 {
		t.Error("نصٌّ حرٌّ مرّ في حقل المنطقة")
		_, _ = f.pool.Exec(ctx, `DELETE FROM merchant_leads WHERE sales_rep_user_id = $1`, rep)
	}

	// ══════════════════════════════════════════════════════════════════
	// **وبمنطقةٍ حيّةٍ يمرّ — وقفلٌ لا يُفتح بمفتاحه عطبٌ لا حماية**
	// ══════════════════════════════════════════════════════════════════
	w = postRepLead(t, f, rep, base+`,"district_id":"`+dist+`","area":"مقابل الجامع"}`)
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(),
			`DELETE FROM merchant_leads WHERE sales_rep_user_id = $1`, rep)
	})
	if w.Code >= 400 {
		t.Fatalf("طلبٌ بمنطقةٍ حيّةٍ رُدّ (%d) — %s", w.Code, w.Body.String())
	}

	// **والصفُّ يحمل الاثنين** — المنطقةَ والعنوانَ التفصيليّ.
	var gotDistrict *string
	var gotArea string
	if err := f.pool.QueryRow(ctx, `
		SELECT district_id::text, area FROM merchant_leads
		WHERE sales_rep_user_id = $1 ORDER BY created_at DESC LIMIT 1`, rep).
		Scan(&gotDistrict, &gotArea); err != nil {
		t.Fatalf("تعذّرت قراءةُ الطلب: %v", err)
	}
	if gotDistrict == nil || *gotDistrict != dist {
		t.Errorf("المنطقةُ لم تُقيَّد: %v", gotDistrict)
	}
	if gotArea != "مقابل الجامع" {
		t.Errorf("العنوانُ التفصيليُّ ضاع: %q", gotArea)
	}
}
