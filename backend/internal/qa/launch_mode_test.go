package qa

import (
	"net/http"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/authz"
)

// ══════════════════════════════════════════════════════════════════════
// **وضعُ الإطلاق — ما هو مفتوحٌ للناس الآن** (`LM`)
// ══════════════════════════════════════════════════════════════════════
//
// # المسألة
//
// **والمنصّةُ تُنزَّل قبل أن تُفتح**: يُثبِّت الناسُ التطبيقَ ويُسجّلون
// ويتصفّحون، **والطلبُ لا يُستقبَل** حتّى تمتلئ السوقُ بالمتاجر.
//
// # وما يُقاس هنا
//
//	١ · كلُّ بابٍ يُفتح ويُغلق مستقلّاً
//	٢ · والمنعُ في المحرّك — **فنداءٌ مباشرٌ لا يتجاوز زرّاً مخفيّاً**
//	٣ · وغيابُ الإعداد يُغلق ولا يفتح
//	٤ · والرمزُ `launch_closed` لا `forbidden` — «ليس الآن» لا «لستَ أهلاً»
//	٥ · ونصُّ المالك يصل العميلَ إن ضبطه

// launchOff **يُغلق باباً لمدّة الاختبار** — و`Setting` تُرجعه بعده.
func launchOff(h *Harness, key string) { h.Setting(key, "false") }

// launchOn **يفتحه صراحةً** — ولا يُترَك للافتراض.
func launchOn(h *Harness, key string) { h.Setting(key, "true") }

// isLaunchClosed **أهذا ردُّ بابٍ مغلق؟** — ٥٠٣ ورمزُه.
func isLaunchClosed(r Res) bool {
	return r.Code == http.StatusServiceUnavailable && r.Err() == "launch_closed"
}

// TestLM1_OrderingClosedWhileBrowsingOpen **الحالُ التي طلبها المالك.**
//
// **يُنزَّل التطبيقُ ويُسجَّل ويُتصفَّح** — **والطلبُ يُردّ.**
func TestLM1_OrderingClosedWhileBrowsingOpen(t *testing.T) {
	hh := New(t)
	launchOn(hh, "launch.customer_signup")
	launchOn(hh, "launch.customer_browse")
	launchOff(hh, "launch.customer_orders")

	// ── التصفّحُ مفتوح ───────────────────────────────────────────
	if r := hh.GET("/api/v1/public/home", ""); r.Code != http.StatusOK {
		t.Errorf("**التصفّحُ مفتوحٌ ومع ذلك رُدّ**: %d / %s", r.Code, r.Err())
	}

	// ── والطلبُ مردود ────────────────────────────────────────────
	u, tok := capUser(t, hh, "customer")
	_ = u
	r := hh.POST("/api/v1/orders", tok, map[string]any{
		"merchant_id": "00000000-0000-0000-0000-000000000001",
		"items":       []any{},
	})
	if !isLaunchClosed(r) {
		t.Errorf("**الطلبُ مرّ وبابُه مغلق**: %d / %s — **وزرٌّ مخفيٌّ ليس منعاً.**",
			r.Code, r.Err())
	}
}

// TestLM2_EachDoorIsIndependent **ولا بابٌ يفتح غيرَه.**
//
// **ومفتاحٌ واحدٌ «مفتوح/مغلق» لا يكفي** — **التصفّحُ يُفتح والطلبُ
// يُغلق، وقد يُفتح العاديُّ ويُغلق المخصَّص.**
func TestLM2_EachDoorIsIndependent(t *testing.T) {
	hh := New(t)
	_, tok := capUser(t, hh, "customer")

	// **العاديُّ مغلقٌ والمخصَّصُ مفتوح.**
	launchOff(hh, "launch.customer_orders")
	launchOn(hh, "launch.merchant_orders")
	launchOn(hh, "launch.customer_custom_orders")

	normal := hh.POST("/api/v1/orders", tok, map[string]any{"items": []any{}})
	if !isLaunchClosed(normal) {
		t.Errorf("**العاديُّ مرّ وبابُه مغلق**: %d / %s", normal.Code, normal.Err())
	}
	custom := hh.POST("/api/v1/orders/custom", tok, map[string]any{
		"request": "قياس", "address_text": "الرقة", "lat": 35.95, "lng": 39.01,
	})
	if isLaunchClosed(custom) {
		t.Errorf("**المخصَّصُ رُدّ ببابِ غيرِه** — والبابان مستقلّان.")
	}

	// **والعكسُ كذلك.**
	launchOn(hh, "launch.customer_orders")
	launchOff(hh, "launch.customer_custom_orders")
	custom2 := hh.POST("/api/v1/orders/custom", tok, map[string]any{
		"request": "قياس", "address_text": "الرقة", "lat": 35.95, "lng": 39.01,
	})
	if !isLaunchClosed(custom2) {
		t.Errorf("**المخصَّصُ مرّ وبابُه مغلق**: %d / %s", custom2.Code, custom2.Err())
	}
}

// TestLM3_MerchantDoorGatesNormalOrdersOnly **وبابُ المتاجر يعني شيئاً.**
//
// **ويُفتح المكتبُ وتبقى السوقُ مغلقة**: المخصَّصُ يمشي، **والطلبُ من
// متجرٍ يُردّ.**
func TestLM3_MerchantDoorGatesNormalOrdersOnly(t *testing.T) {
	hh := New(t)
	_, tok := capUser(t, hh, "customer")
	launchOn(hh, "launch.customer_orders")
	launchOff(hh, "launch.merchant_orders")
	launchOn(hh, "launch.customer_custom_orders")

	if r := hh.POST("/api/v1/orders", tok, map[string]any{"items": []any{}}); !isLaunchClosed(r) {
		t.Errorf("**الطلبُ من متجرٍ مرّ وبابُ المتاجر مغلق**: %d / %s", r.Code, r.Err())
	}
	c := hh.POST("/api/v1/orders/custom", tok, map[string]any{
		"request": "قياس", "address_text": "الرقة", "lat": 35.95, "lng": 39.01,
	})
	if isLaunchClosed(c) {
		t.Error("**المخصَّصُ رُدّ ببابِ المتاجر** — ولا متجرَ فيه.")
	}
}

// TestLM4_SignupAndDriverAndRepDoors **والأبوابُ الثلاثةُ الأخرى.**
func TestLM4_SignupAndDriverAndRepDoors(t *testing.T) {
	hh := New(t)

	// ── إنشاءُ الحسابات ─────────────────────────────────────────
	launchOff(hh, "launch.customer_signup")
	if r := hh.POST("/api/v1/auth/signup/request", "",
		map[string]any{"phone": "0933111222"}); !isLaunchClosed(r) {
		t.Errorf("**التسجيلُ مرّ وبابُه مغلق**: %d / %s", r.Code, r.Err())
	}
	launchOn(hh, "launch.customer_signup")
	if r := hh.POST("/api/v1/auth/signup/request", "",
		map[string]any{"phone": "0933111222"}); isLaunchClosed(r) {
		t.Error("**التسجيلُ رُدّ وبابُه مفتوح.**")
	}

	// ── دوامُ السائق ────────────────────────────────────────────
	launchOff(hh, "launch.driver_work")
	_, dtok := capUser(t, hh, "driver")
	if r := hh.POST("/api/v1/driver/shift", dtok,
		map[string]any{"on": true}); !isLaunchClosed(r) {
		t.Errorf("**بدءُ الدوام مرّ وبابُه مغلق**: %d / %s", r.Code, r.Err())
	}

	// ── وضمُّ المتاجر ───────────────────────────────────────────
	launchOff(hh, "launch.rep_acquisition")
	_, rtok := capUser(t, hh, "sales")
	if r := hh.POST("/api/v1/rep/leads", rtok, map[string]any{
		"name": "متجرُ قياس", "phone": "0933222333",
	}); !isLaunchClosed(r) {
		t.Errorf("**ضمُّ المتاجر مرّ وبابُه مغلق**: %d / %s", r.Code, r.Err())
	}
}

// TestLM5_MissingConfigFailsClosed **وغيابُ الإعداد يُغلق ولا يفتح.**
//
// **وشرطُ المالك**: **قراءةٌ تعذّرت لا تفتح عملاً مُغلقاً.** **فيُمحى
// الصفُّ ويُقاس أنّ الافتراضَ مغلق.**
func TestLM5_MissingConfigFailsClosed(t *testing.T) {
	hh := New(t)
	_, tok := capUser(t, hh, "customer")

	for _, key := range []string{
		"launch.customer_orders", "launch.customer_custom_orders",
		"launch.customer_browse", "launch.customer_signup",
	} {
		var old *string
		_ = hh.Pool.QueryRow(ctxBG(),
			`SELECT value::text FROM app_settings WHERE key = $1`, key).Scan(&old)
		if _, err := hh.Pool.Exec(ctxBG(),
			`DELETE FROM app_settings WHERE key = $1`, key); err != nil {
			t.Fatalf("محوُ %s: %v", key, err)
		}
		k := key
		o := old
		t.Cleanup(func() {
			if o != nil {
				_, _ = hh.Pool.Exec(ctxBG(), `
					INSERT INTO app_settings (key, value) VALUES ($1, $2::jsonb)
					ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, k, *o)
			}
		})
	}

	if r := hh.POST("/api/v1/orders", tok, map[string]any{"items": []any{}}); !isLaunchClosed(r) {
		t.Errorf("**بلا إعدادٍ انفتح الطلب**: %d / %s — **والإخفاقُ يجب أن يُغلق.**",
			r.Code, r.Err())
	}
	if r := hh.GET("/api/v1/public/home", ""); r.Code != http.StatusServiceUnavailable {
		t.Errorf("**بلا إعدادٍ انفتح التصفّح**: %d", r.Code)
	}
	if r := hh.POST("/api/v1/auth/signup/request", "",
		map[string]any{"phone": "0933444555"}); !isLaunchClosed(r) {
		t.Errorf("**بلا إعدادٍ انفتح التسجيل**: %d / %s", r.Code, r.Err())
	}
}

// TestLM6_NoticeReachesTheClient **ونصُّ المالك يصل، ولا يُكتب في شيفرة.**
func TestLM6_NoticeReachesTheClient(t *testing.T) {
	hh := New(t)
	_, tok := capUser(t, hh, "customer")
	launchOff(hh, "launch.customer_orders")

	// ── بلا نصٍّ: الرمزُ وحدَه، وتُترجمه الشاشة ─────────────────
	hh.Setting("launch.notice", `""`)
	r := hh.POST("/api/v1/orders", tok, map[string]any{"items": []any{}})
	if !isLaunchClosed(r) {
		t.Fatalf("لم يُردَّ الطلب: %d / %s", r.Code, r.Err())
	}
	if strings.Contains(string(r.Body), `"notice"`) {
		t.Error("**نصٌّ فارغٌ أُرسل** — وفارغُه لا يُعرَض.")
	}

	// ── وبنصٍّ: يصل كما ضُبط ────────────────────────────────────
	const notice = "نفتح استقبال الطلبات بعد العيد إن شاء الله."
	hh.Setting("launch.notice", `"`+notice+`"`)
	r2 := hh.POST("/api/v1/orders", tok, map[string]any{"items": []any{}})
	if !isLaunchClosed(r2) {
		t.Fatalf("لم يُردَّ الطلب: %d / %s", r2.Code, r2.Err())
	}
	if !strings.Contains(string(r2.Body), notice) {
		t.Errorf("**نصُّ المالك لم يصل**: %s", string(r2.Body))
	}
	// **والمفتاحُ باقٍ** — عقدٌ تقرؤه الواجهةُ، ولا يُبدَّل بنصّ.
	if !strings.Contains(string(r2.Body), "errors.launch_closed") {
		t.Error("**مفتاحُ الرسالة ضاع** — فكلُّ عميلٍ لا يعرف النصَّ يفقد ترجمتَه.")
	}
}

// TestLM7_NotForbiddenButNotYet **«ليس الآن» لا «لستَ أهلاً».**
//
// **و٤٠٣ يُقرأ منعَ صلاحيّةٍ** — **فيظنُّ الزبونُ أنّ حسابَه ناقص.**
func TestLM7_NotForbiddenButNotYet(t *testing.T) {
	hh := New(t)
	_, tok := capUser(t, hh, "customer")
	launchOff(hh, "launch.customer_orders")

	r := hh.POST("/api/v1/orders", tok, map[string]any{"items": []any{}})
	if r.Code == http.StatusForbidden {
		t.Error("**البابُ المغلقُ يردّ ٤٠٣** — ويُقرأ «لستَ أهلاً» وهو «ليس الآن».")
	}
	if r.Code != http.StatusServiceUnavailable {
		t.Errorf("**رمزُ الحال %d** — والمنتظَرُ ٥٠٣.", r.Code)
	}
	if r.Err() == "forbidden" {
		t.Error("**رمزُ الخطأ `forbidden`** — ولا علاقةَ للصلاحيّة بالحال.")
	}
}

// TestLM8_OnlySettingsOwnerMayChangeLaunch **ولا سلطةَ إعداداتٍ جديدة.**
//
// **وشرطُ المالك (بندُ ٥/ب)**: **لا يُعطى العمليّاتُ ولا الماليّةُ
// سلطةَ إعداداتٍ جديدة** — **ومصفوفةُ القدرات المنشورةُ تُحترَم.**
//
// **وقدرةُ مفاتيح الإطلاق `settings.general.manage`** — يملكها الأدمنُ
// والمالكُ وحدَهما.
func TestLM8_OnlySettingsOwnerMayChangeLaunch(t *testing.T) {
	hh := New(t)
	const key = "/api/v1/admin/settings/launch.customer_orders"

	// ── من لا يملكها يُردّ ──────────────────────────────────────
	// **ولا `observability` هنا**: **ليس دوراً مبذوراً في قاعدة الاختبار**
	// — **ودورٌ لا وجودَ له يسقط على قيدِ مفتاحٍ أجنبيٍّ لا على تخويل**،
	// **فيُقرأ منعاً وهو غياب.** ويُقاس بدورٍ مصنوعٍ بقدرته وحدَها.
	capRole(t, hh, "lm_obs", authz.ObservabilityRead)
	for _, role := range []string{"operations", "finance", "lm_obs", "customer_support"} {
		_, tok := roleUser(t, hh, role)
		r := hh.Call("PUT", key, tok, map[string]any{"value": false}, nil)
		if r.Code < 400 {
			t.Errorf("**`%s` بدّل بابَ إطلاق** (%d) — **ولا سلطةَ إعداداتٍ جديدة له.**",
				role, r.Code)
		}
	}

	// ── والمالكُ يملكها ─────────────────────────────────────────
	//
	// **و٤٠٣ منعُ صلاحيّةٍ، وسواها جوابُ منطقٍ** — والمقيسُ التخويلُ
	// وحدَه.
	_, owner := roleUser(t, hh, "owner_super_admin")
	r := hh.Call("PUT", key, owner, map[string]any{"value": true}, nil)
	if r.Code == http.StatusForbidden {
		t.Errorf("**المالكُ مُنع من بابِ إطلاق**: %s", r.Err())
	}
}
