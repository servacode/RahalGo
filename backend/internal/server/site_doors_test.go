package server

// **بابا الموقع يصلان إلى الشاشة — ومفتوحان ما لم يُطفآ.**
//
// (طلبُ المالك ٢٠٢٦-٠٨-١٧: «أضف مفتاحاً لإخفاء زرّ تسجيل الدخول — بحيث
//  أستطيع الدخول عن طريق الرابط المباشر — وأيضاً زرّاً لإخفاء زرّ تسوّق».)
//
// # ولماذا فحصٌ لمفتاحٍ منطقيّ
//
// **إعدادٌ يُضاف إلى الفهرس ولا يُضمّ إلى الردّ يبقى ساكناً**: تُقلَب
// الشاشةُ في اللوحة **ولا يتبدّل شيءٌ في الموقع**، ولا خطأَ يقول لماذا.
// **وصاحبُه يظنّ المفتاحَ معطوباً** وهو لم يُوصَل أصلاً.
//
// # والافتراضُ يُفحص كما يُفحص القلب
//
// **`show_login !== false` في الشاشة** — فحقلٌ غائبٌ يعني «أظهِر».
// **ولو ردَّ المحرّكُ عدمَه لَظهر البابان دائماً** ولو أُطفئا: **مفتاحٌ
// يُقلب ولا يفعل.**

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSiteDoors_ReachThePublicPayload(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()

	// ══════════════════════════════════════════════════════════════════
	// **وما يُكتب في الإعدادات يُمحى بعده**
	// ══════════════════════════════════════════════════════════════════
	//
	// **قاعدةُ الفحص مشتركةٌ بين الحزم كلِّها** — **وفحصٌ يُطفئ مفتاحاً
	// ويتركه مُطفأً يُسقط نفسَه في التشغيلة التالية**، ويُسقط كلَّ من
	// يقرأ ذلك المفتاح. (وقع في أوّل تشغيلةٍ كاملةٍ لهذا الفحص:
	// «الافتراضُ مغلق» — **وكان الفحصُ نفسُه من أغلقه.**)
	//
	// **والمحوُ يسبق القياسَ ويتبعُه**: يسبقه لأنّ تشغيلةً سابقةً قد
	// خلّفت أثراً، **ويتبعه فلا نخلّف نحن.**
	wipe := func() {
		_, _ = f.pool.Exec(context.Background(),
			`DELETE FROM app_settings WHERE key IN ('site.show_login', 'site.show_shop')`)
	}
	wipe()
	t.Cleanup(wipe)

	read := func() (login, shop bool) {
		t.Helper()
		w := httptest.NewRecorder()
		f.srv.handlePublicPlatform(w, httptest.NewRequest(http.MethodGet, "/x", nil))
		if w.Code != 200 {
			t.Fatalf("ردَّ %d — %s", w.Code, w.Body.String())
		}
		var env struct {
			Data struct {
				ShowLogin *bool `json:"show_login"`
				ShowShop  *bool `json:"show_shop"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
			t.Fatalf("ردٌّ لا يُفكّ: %v", err)
		}
		// **ومؤشّرٌ لا قيمة**: الغائبُ والمُطفأُ يُقرآن `false` سواءً لو
		// قُرئا قيمةً — **فلا يُميَّز «لم يُرسَل» من «أُطفئ».**
		if env.Data.ShowLogin == nil || env.Data.ShowShop == nil {
			t.Fatalf("المفتاحان لم يصلا إلى الردّ أصلاً — %s", w.Body.String())
		}
		return *env.Data.ShowLogin, *env.Data.ShowShop
	}

	// **والافتراضُ الظهور** — منصّةٌ لم تُلمَس إعداداتُها لها بابٌ.
	if login, shop := read(); !login || !shop {
		t.Fatalf("الافتراضُ مغلق: دخول %v · تسوّق %v", login, shop)
	}

	// **ثمّ يُطفآن فيتبدّل الردّ** — **وهذا ما يفصل «موصولاً» عن «مكتوباً».**
	for _, key := range []string{"site.show_login", "site.show_shop"} {
		if err := f.srv.settings.Set(ctx, key, false, nil); err != nil {
			t.Fatalf("تعذّر إطفاءُ %q: %v", key, err)
		}
	}
	if login, shop := read(); login || shop {
		t.Fatalf("أُطفئا وبقيا: دخول %v · تسوّق %v", login, shop)
	}
}
