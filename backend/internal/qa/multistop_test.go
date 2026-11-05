package qa

// ══════════════════════════════════════════════════════════════════════
// **طلبٌ واحدٌ بعدّة محطّات** — قرارُ المالك ٢٠٢٦-١١-٠٥ («١ موافق»)
// ══════════════════════════════════════════════════════════════════════
//
// **الدفعةُ الأولى (P1)**: بوّابةُ نسخة السائق · صفوفُ الأقسام عند الإنشاء ·
// كلُّ المصادر مفتوحة. **ولا شيءَ يقرأ الأقسامَ بعد** — القبولُ والاستلامُ
// لكلّ محطّةٍ في الدفعات التالية.

import (
	"net/http"
	"testing"
)

// multiStopBuild **رقمُ نسخة السائق «التي تعرف المحطّات» في الاختبار** —
// أعلى من كلّ نسخةٍ منشورة، **والمِسنَدُ لا يرسل ترويسةَ النسخة** فلا يُردّ
// نداءُ سائقٍ في الاختبار بـ٤٢٦.
const multiStopBuild = "9000"

// OpenMultiStop **يفتح بوّابةَ الطلب من عدّة متاجر** — نسخةُ المحطّات معلومةٌ
// (`orders.multistop_driver_build`) **ومفروضةٌ على كلّ السائقين**
// (`app.min_version.driver`). **ويُعاد الاثنان بعد الاختبار.**
func (h *Harness) OpenMultiStop() {
	h.T.Helper()
	h.Setting("orders.multistop_driver_build", multiStopBuild)
	h.Setting("app.min_version.driver", multiStopBuild)
}

// twoStoreBody **طلبٌ صحيحٌ بصنفٍ من كلّ متجر** — بترتيب السلّة.
func twoStoreBody(items ...*Item) map[string]any {
	body := orderBody(items[0], 1)
	lines := []map[string]any{}
	for _, it := range items {
		lines = append(lines, map[string]any{"menu_item_id": it.ID, "qty": 1})
	}
	body["items"] = lines
	return body
}

// quoteSources **سقفُ المصادر كما تقوله التسعيرةُ للسلّة.**
func quoteSources(t *testing.T, h *Harness, tok string, items ...*Item) (max float64, tooMany bool) {
	t.Helper()
	lines := []map[string]any{}
	for _, it := range items {
		lines = append(lines, map[string]any{"menu_item_id": it.ID, "qty": 1})
	}
	r := h.POST("/api/v1/public/quote", tok, map[string]any{
		"items": lines, "lat": raqqaLat, "lng": raqqaLng,
	})
	if r.Code != http.StatusOK {
		t.Fatalf("التسعيرة: %s", r)
	}
	j := r.JSON()
	max, _ = j["max_sources"].(float64)
	tooMany, _ = j["too_many_sources"].(bool)
	return max, tooMany
}

// TestMultiStop_RuntimeGate **سقفُ المصادر فوق الواحد ينتظر نسخةَ السائق** — `T36`.
//
// **وفي وقت التشغيل لا عند الحفظ**: **إنزالُ أدنى نسخةٍ للسائق يُغلق السلّةَ
// في اللحظة**، والتسعيرةُ والإنشاءُ يقولان الشيءَ نفسَه.
//
// **وكان السقفُ يسري وحدَه** — فسلّةٌ من متجرين تُقبل وسائقٌ بتطبيقٍ لا يعرف
// إلّا محطّةً واحدةً يأخذها.
func TestMultiStop_RuntimeGate(t *testing.T) {
	h := New(t)
	treasury(t, h)
	u := h.Customer()
	a := h.NewItem(1000)
	b := h.NewItem(1000)
	h.Setting("orders.max_sources", "2")
	h.Setting("orders.multistop_driver_build", "22")
	h.Setting("app.min_version.driver", "21")

	expect := func(step string, wantMax float64, wantTooMany bool) {
		t.Helper()
		max, tooMany := quoteSources(t, h, u.Token, a, b)
		if max != wantMax || tooMany != wantTooMany {
			t.Errorf("%s: التسعيرةُ تقول السقفَ %v وتجاوزَه %v — يُنتظر %v و%v",
				step, max, tooMany, wantMax, wantTooMany)
		}
	}

	// ١ · أدنى نسخةٍ للسائق دون نسخة المحطّات ⇒ متجرٌ واحد.
	expect("أدنى نسخةٍ ٢١ ونسخةُ المحطّات ٢٢", 1, true)
	r := h.POSTKey("/api/v1/orders", u.Token, uniq("ms-gate"), twoStoreBody(a, b))
	if r.Code != http.StatusConflict || r.Err() != "too_many_sources" {
		t.Fatalf("**طلبٌ من متجرين والبوّابةُ مغلقة**: %s — يُنتظر 409 too_many_sources", r)
	}

	// ٢ · فُرضت النسخة ⇒ السقفُ المضبوط.
	h.Setting("app.min_version.driver", "22")
	expect("أدنى نسخةٍ ٢٢", 2, false)
	if r := h.POSTKey("/api/v1/orders", u.Token, uniq("ms-gate"), twoStoreBody(a, b)); r.Code >= 400 {
		t.Fatalf("**البوّابةُ مفتوحةٌ والطلبُ من متجرين رُدّ**: %s", r)
	}

	// ٣ · رجوعٌ يُنزل أدنى نسخة ⇒ متجرٌ واحدٌ في اللحظة.
	h.Setting("app.min_version.driver", "21")
	expect("رجوعٌ إلى ٢١", 1, true)

	// ٤ · نسخةُ المحطّات صفر (الافتراض) ⇒ مغلقةٌ مهما علت أدنى نسخة.
	h.Setting("orders.multistop_driver_build", "0")
	h.Setting("app.min_version.driver", "99")
	expect("نسخةُ المحطّات صفر", 1, true)

	// ٥ · ورقمُ الملفّ المرفوع مع الفرض التلقائيّ يفتحها كالحدّ اليدويّ —
	// **وهو ما تقرؤه بوّابةُ التحديث نفسُها** (`effectiveMinVersion`).
	h.Setting("orders.multistop_driver_build", "22")
	h.Setting("app.min_version.driver", "0")
	h.Setting("release.driver.apk", `"driver-22.apk"`)
	h.Setting("release.driver.version_code", "22")
	expect("ملفٌّ مرفوعٌ برقم ٢٢ والفرضُ تلقائيّ", 2, false)
	h.Setting("release.driver.auto_force", "false")
	expect("ملفٌّ مرفوعٌ والفرضُ مطفأ", 1, true)

	// ٦ · والسقفُ واحدٌ يبقى واحداً ولو فُتحت البوّابة.
	h.Setting("release.driver.auto_force", "true")
	h.Setting("orders.max_sources", "1")
	expect("السقفُ واحد", 1, true)
}

// TestMultiStop_MaxSourcesSettingWaitsForDriverBuild **ولا يُحفظ سقفٌ لا يقع أثرُه.**
//
// **كان يُحفظ فلا يتبدّل شيءٌ في السلّة** — **فيرفع المالكُ السقفَ ويبقى
// «متجرٌ واحد» ولا يعرف لماذا.** **فالردُّ يقول السبب** (`multistop_driver_not_forced`)،
// **والإنزالُ إلى الواحد يُحفظ دائماً.**
func TestMultiStop_MaxSourcesSettingWaitsForDriverBuild(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	h.Setting("orders.max_sources", "1")
	h.Setting("orders.multistop_driver_build", "22")
	h.Setting("app.min_version.driver", "21")

	put := func(v int) Res {
		t.Helper()
		return h.Call("PUT", "/api/v1/admin/settings/orders.max_sources",
			admin.Token, map[string]any{"value": v}, nil)
	}
	stored := func() string {
		t.Helper()
		var v string
		if err := h.Pool.QueryRow(ctxBG(),
			`SELECT value::text FROM app_settings WHERE key = 'orders.max_sources'`).Scan(&v); err != nil {
			t.Fatalf("قراءةُ السقف: %v", err)
		}
		return v
	}

	if r := put(2); r.Code != http.StatusConflict || r.Err() != "multistop_driver_not_forced" {
		t.Fatalf("**سقفٌ فوق الواحد حُفظ والنسخةُ لم تُفرض**: %s — يُنتظر 409 multistop_driver_not_forced", r)
	}
	if got := stored(); got != "1" {
		t.Fatalf("**الرفضُ كتب**: السقفُ المحفوظ %s — يُنتظر 1", got)
	}

	// **وبعد الفرض يُحفظ.**
	h.Setting("app.min_version.driver", "22")
	if r := put(2); r.Code >= 400 {
		t.Fatalf("**النسخةُ مفروضةٌ والسقفُ رُدّ**: %s", r)
	}
	if got := stored(); got != "2" {
		t.Fatalf("السقفُ المحفوظ %s — يُنتظر 2", got)
	}

	// **والإنزالُ إلى الواحد لا يُمنع أبداً** — ولو أُغلقت البوّابة.
	h.Setting("app.min_version.driver", "21")
	if r := put(1); r.Code >= 400 {
		t.Fatalf("**إنزالُ السقف إلى الواحد رُدّ**: %s", r)
	}
}
