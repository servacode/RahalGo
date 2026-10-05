package settings

// حرّاسُ قرارات المالك ٢٠٢٦-١٠-٠٤ في قسم الإعدادات — بلا قاعدة.

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// **الافتراضُ في الشيفرة = قرارُ المالك** (البند ٣) — القاعدةُ الجديدةُ تبدأ صحيحة.
func TestSETTINGS_CodeDefaultsAreOwnerDecisions(t *testing.T) {
	want := map[string]int64{
		"sales.commission_percent":            10,
		"drivers.failed_compensation_percent": 50,
		"merchants.commission_percent":        10,
		"delivery.platform_percent":           0,
	}
	for k, v := range want {
		if _, ok := Lookup(k); !ok {
			t.Errorf("المفتاحُ %q غائبٌ عن الفهرس", k)
			continue
		}
		if got := Default(k); got != v {
			t.Errorf("افتراضُ %q = %d والقرارُ %d", k, got, v)
		}
	}
}

// **المفاتيحُ الميّتةُ والملغاةُ خرجت من الفهرس** (البندان ٤ و٦).
func TestSETTINGS_DeadKeysRemoved(t *testing.T) {
	for _, k := range []string{
		"delivery.merchant_delivery_platform_percent",
		"sales.commission_source", "sales.activation_orders",
		"platform.invite_code", "platform.app_url", "platform.app_file",
		"merchants.require_menu_review",
	} {
		if _, ok := Lookup(k); ok {
			t.Errorf("المفتاحُ %q ما زال في الفهرس", k)
		}
	}
}

// **حصّةُ المنصّة لا تتجاوز تسعين** — كان الحفظُ يقبل ٩٥ والمحرّكُ يطبّق ٩٠ بصمت.
func TestSETTINGS_PercentRespectsCatalogMax(t *testing.T) {
	if _, err := Validate("delivery.platform_percent", float64(95)); err == nil {
		t.Fatal("قُبلت ٩٥٪ والحدُّ تسعون")
	}
	if v, err := Validate("delivery.platform_percent", float64(90)); err != nil || v != int64(90) {
		t.Fatalf("رُدّت ٩٠٪: %v %v", v, err)
	}
	// **ومفتاحٌ بلا حدٍّ مكتوبٍ يبقى على المئة.**
	if _, err := Validate("platform.background_dim", float64(100)); err != nil {
		t.Fatalf("رُدّت ١٠٠ لمفتاحٍ مداه مئة: %v", err)
	}
}

// **سقفُ نقد السائق لا يكون صفراً** (البند ٥) — وأقلُّه عشرةُ آلاف.
func TestSETTINGS_DriverCashLimitZeroForbidden(t *testing.T) {
	for _, bad := range []float64{0, 9999} {
		if _, err := Validate("drivers.cash_limit", bad); err == nil {
			t.Errorf("قُبل سقفُ نقدٍ %v", bad)
		}
	}
	if _, err := Validate("drivers.cash_limit", float64(10000)); err != nil {
		t.Errorf("رُدّ ١٠٬٠٠٠: %v", err)
	}
}

// **القوالبُ لا تُحفظ بلا نصوصها الإلزاميّة.**
func TestSETTINGS_TemplatesRequirePlaceholders(t *testing.T) {
	cases := []struct {
		key, val, missing string
	}{
		{"whatsapp.otp_template", "رمزك هنا", "{code}"},
		{"auth.sms_template", "رمز", "{code}"},
		{"accounts.welcome_template", "أهلاً {link}", "{password}"},
		{"accounts.welcome_template", "كلمتك {password}", "{link}"},
	}
	for _, c := range cases {
		_, err := Validate(c.key, c.val)
		var miss ErrMissingPlaceholder
		if !errors.As(err, &miss) || miss.Placeholder != c.missing {
			t.Errorf("%s=%q ⇒ %v — والمنتظَرُ نقصُ %s", c.key, c.val, err, c.missing)
		}
	}
	for _, k := range []string{"whatsapp.otp_template", "auth.sms_template", "accounts.welcome_template"} {
		d, _ := Lookup(k)
		if _, err := Validate(k, d.Default); err != nil {
			t.Errorf("افتراضُ %s لا يجتاز حارسَه: %v", k, err)
		}
	}
}

type fakeInts map[string]int64

func (f fakeInts) GetInt(_ context.Context, k string) int64 { return f[k] }

// **المفاتيحُ المرتبطةُ تُفحص معاً** (البند ١٦).
func TestSETTINGS_RelatedKeysChecked(t *testing.T) {
	ctx := context.Background()
	st := fakeInts{"delivery.custom_fee_min": 1000, "delivery.custom_fee_max": 100000,
		"orders.accept_timeout_min": 5, "orders.auto_accept_min": 10}
	bad := []struct {
		key string
		v   int64
	}{
		{"delivery.custom_fee_min", 200000},
		{"delivery.custom_fee_max", 500},
		{"orders.auto_accept_min", 5},
		{"orders.accept_timeout_min", 10},
	}
	for _, b := range bad {
		if err := CheckRelated(ctx, st, b.key, b.v); err == nil {
			t.Errorf("%s=%d قُبل وهو يتعارض", b.key, b.v)
		}
	}
	good := []struct {
		key string
		v   int64
	}{
		{"delivery.custom_fee_min", 100000},
		{"orders.auto_accept_min", 0},
		{"orders.auto_accept_min", 6},
		{"orders.accept_timeout_min", 9},
	}
	for _, g := range good {
		if err := CheckRelated(ctx, st, g.key, g.v); err != nil {
			t.Errorf("%s=%d رُدّ: %v", g.key, g.v, err)
		}
	}
}

// **كلُّ مفتاحٍ له بيتٌ في العمود الجانبيّ** — ومفاتيحُ الغد تسكن ببادئتها (البند ١٧).
func TestSETTINGS_EveryKeyHasATopic(t *testing.T) {
	known := map[Topic]bool{}
	for _, tp := range Topics {
		known[tp] = true
	}
	for _, d := range Catalog {
		p := PlacementOf(d)
		if !known[p.Topic] {
			t.Errorf("%s ⇒ موضوعٌ مجهول %q", d.Key, p.Topic)
		}
	}
	want := map[string]Topic{
		"pricing.margin_fixed":               TopicMoney,
		"delivery.platform_percent":          TopicMoney,
		"sales.commission_percent":           TopicMoney,
		"delivery.fee":                       TopicMoney,
		"support.late_reply_hours":           TopicSupport,
		"payouts.min_amount":                 TopicSupport,
		"whatsapp.otp_template":              TopicMessages,
		"accounts.welcome_template":          TopicMessages,
		"security.session_days":              TopicSecurity,
		"launch.customer_orders":             TopicLaunch,
		"release.driver.apk":                 TopicApps,
		"platform.location":                  TopicSite,
		"orders.accept_timeout_min":          TopicOrders,
		"drivers.assignment_mode":            TopicOrders,
		"drivers.monthly_target":             TopicDrivers,
		"customers.max_addresses":            TopicCustomers,
		"finance.expense_approval_threshold": TopicMoney,
		"compensations.cap_driver":           TopicMoney,
		"obligations.anything_new":           TopicMoney,
	}
	for k, tp := range want {
		if got := PlacementOf(Def{Key: k, Group: GroupPlatform}).Topic; got != tp {
			t.Errorf("%s ⇒ %q والمنتظَرُ %q", k, got, tp)
		}
	}
	// **وكلُّ إعدادٍ في مكانٍ واحد** (البند ١٢): المكرّرُ يُضبط في لوحه.
	for k, panel := range map[string]string{"launch.notice": "appStatus",
		"hours.platform_enforced": "hours", "app.min_version.driver": "release"} {
		if got := PlacementOf(Def{Key: k}).Panel; got != panel {
			t.Errorf("%s ⇒ لوح %q والمنتظَرُ %q", k, got, panel)
		}
	}
	// **وإعداداتُ الموقع العامّ مخفيّة** (البند ١٤) — والهويّةُ والتواصلُ ظاهرة.
	for _, k := range []string{"site.show_login", "platform.background", "shop.rail_auto"} {
		if !PlacementOf(Def{Key: k}).Hidden {
			t.Errorf("%s ظاهرٌ والقرارُ إخفاؤه", k)
		}
	}
	// **وتقليبُ سلايدر التطبيق رجع مع لوحه** (قرارُ المالك ٢٠٢٦-١٠-٠٥) — التطبيقُ يقرؤه.
	for _, k := range []string{"home.banner_auto", "home.banner_seconds"} {
		if p := PlacementOf(Def{Key: k}); p.Hidden || p.Panel != "slider" {
			t.Errorf("%s ⇒ %+v والمنتظَرُ لوحُ السلايدر ظاهراً", k, p)
		}
	}
	for _, k := range []string{"platform.name", "platform.support_phone", "page.terms_text"} {
		if PlacementOf(Def{Key: k}).Hidden {
			t.Errorf("%s مخفيٌّ والقرارُ إظهاره", k)
		}
	}
}

// **النصوصُ القانونيّةُ الافتراضيّةُ تطابق القرارات** (البند ١٣) — والمالكُ يراجعها.
func TestSETTINGS_DefaultLegalTextsMatchDecisions(t *testing.T) {
	for _, txt := range []string{DefaultDriverTerms, DefaultDriverHelp, DefaultDriverPrivacy} {
		if strings.Contains(txt, "تعويض") || strings.Contains(txt, "تعوّض") {
			t.Error("نصُّ السائق فيه «تعويض» — ممنوعٌ بقرار ٢٠٢٦-١٠-٠٣")
		}
	}
	if strings.Contains(DefaultMerchantTerms, "مراجعة الإدارة") {
		t.Error("شروطُ المتجر تَعِد بمراجعة الأصناف — أُلغيت")
	}
	if strings.Contains(DefaultMerchantTerms, "بنسبةٍ معلنة") {
		t.Error("شروطُ المتجر تَعِد بنسبة تعويض — أُلغيت")
	}
	if !strings.Contains(DefaultRepTerms, "ربح المنصّة") || strings.Contains(DefaultRepTerms, "من عمولة المنصّة") {
		t.Error("شروطُ المندوب لا تقول إنّ عمولتَه من ربح المنصّة")
	}
}
