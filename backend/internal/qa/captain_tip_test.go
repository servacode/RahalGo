package qa

// ══════════════════════════════════════════════════════════════════════
// **إكراميةُ الكابتن** — قرارُ المالك ٢٠٢٦-١١-٢٥ (`orders/tip.go`، هجرة ٠٥٥٩)
// ══════════════════════════════════════════════════════════════════════
//
// بتنحجز من المحفظة، بتنقيّد بالتسليم للي سلّم (ونسبةُ المنصّة للخزينة)، بترجع بالإلغاء
// وبتصحيح التسليم الخاطئ — **وسطورُها مجموعُها صفر**، والثوابتُ ما بتشوف خرق.

import (
	"net/http"
	"strings"
	"testing"
)

// tipSettings **الإكراميةُ شغّالة ونسبةُ المنصّة ١٠٪** — فإكراميةُ ٢٠: ١٨ للكابتن و٢ للخزينة.
func tipSettings(h *Harness, pct string) {
	mpsSettings(h)
	h.Setting("tips.enabled", "true")
	h.Setting("tips.platform_percent", pct)
}

// tipWalletOrder **طلبٌ «التوصيل من المحفظة» مع إكرامية** — ويردّ معرّفَه ونتيجةَ الإنشاء.
func tipWalletOrder(t *testing.T, h *Harness, cust *User, item *Item, scope string, tip int64) (string, Res) {
	t.Helper()
	body := orderBody(item, 1)
	if scope != "none" {
		body["payment_method"] = "wallet"
	}
	body["wallet_scope"] = scope
	if tip != 0 {
		body["tip"] = tip
	}
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("tip"), body)
	oid, _ := made.JSON()["id"].(string)
	return oid, made
}

type tipRow struct {
	amount, reserved, cut int64
	paid                  bool
}

func readTip(t *testing.T, h *Harness, oid string) tipRow {
	t.Helper()
	var r tipRow
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT tip_amount, tip_reserved, tip_platform_cut, tip_paid_at IS NOT NULL
		FROM orders WHERE id = $1::uuid`, oid).Scan(&r.amount, &r.reserved, &r.cut, &r.paid); err != nil {
		t.Fatal(err)
	}
	return r
}

func tipLine(t *testing.T, h *Harness, oid, userID string) int64 {
	t.Helper()
	var v int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		WHERE ref = $1 AND kind = 'tip' AND user_id = $2::uuid`, oid, userID).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

// TestTIP01_HeldThenPaidToDeliveringCaptain **الحجزُ ثمّ القيد**: ٢٠ بتنحجز، وبالتسليم ١٨ للكابتن
// و٢ للخزينة و−٢٠ عالزبون بنصّ «إكرامية الطلب #N»، والكابتنُ بيشوف `tip_driver` = ١٨.
func TestTIP01_HeldThenPaidToDeliveringCaptain(t *testing.T) {
	h := New(t)
	tid := treasury(t, h)
	tipSettings(h, "10")
	mpsNoCampaign(t, h)
	base := financialBaseline(t, h)
	cust := h.Customer()
	fundWallet(t, h, h.NewUser("admin"), cust.ID, 1000)
	item := h.NewItem(1000)

	oid, made := tipWalletOrder(t, h, cust, item, "delivery", 20)
	if made.Code >= 400 {
		t.Fatalf("الإنشاء: %s", made)
	}
	if r := readTip(t, h, oid); r.amount != 20 || r.reserved != 20 || r.paid {
		t.Fatalf("بعد الإنشاء: %+v — والمنتظَر محجوزة ٢٠", r)
	}
	if _, reserved, _ := walletLayers(t, h, cust.ID); reserved != 20 {
		t.Fatalf("محجوزُ المحفظة %d — والمنتظَر ٢٠", reserved)
	}
	if v := h.GET("/api/v1/my/orders/"+oid, cust.Token); v.JSON()["tip"] != float64(20) {
		t.Fatalf("**الزبونُ ما بيشوف إكراميته**: %s", v)
	}

	drv := h.driverOf(oid)
	// **والكابتنُ بيشوف حصّته على الطلب** قبل ما يسلّم — ١٨.
	if l := h.GET("/api/v1/driver/orders", drv.Token); l.Code >= 400 ||
		!strings.Contains(string(l.Body), `"tip_driver":18`) {
		t.Fatalf("**الكابتنُ ما بيشوف إكراميته**: %s", l)
	}
	deliverOrder(t, h, oid, drv)

	if r := readTip(t, h, oid); r.reserved != 0 || !r.paid || r.cut != 2 {
		t.Fatalf("بعد التسليم: %+v — والمنتظَر مقيّدة بنسبة ٢", r)
	}
	if c, d, tr := tipLine(t, h, oid, cust.ID), tipLine(t, h, oid, drv.ID), tipLine(t, h, oid, tid); c != -20 || d != 18 || tr != 2 {
		t.Fatalf("السطور: زبون %d · كابتن %d · خزينة %d — والمنتظَر −٢٠ · ١٨ · ٢", c, d, tr)
	}
	var note string
	var number int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT t.note, o.number FROM wallet_transactions t JOIN orders o ON o.id::text = t.ref
		WHERE t.ref = $1 AND t.kind = 'tip' AND t.user_id = $2::uuid`, oid, drv.ID).Scan(&note, &number); err != nil {
		t.Fatal(err)
	}
	if want := "إكرامية الطلب #" + itoa(int(number)); note != want {
		t.Fatalf("نصُّ السطر %q — والمنتظَر %q", note, want)
	}
	if _, reserved, _ := walletLayers(t, h, cust.ID); reserved != 0 {
		t.Fatalf("**حجزٌ باقي بعد التسليم**: %d", reserved)
	}
	assertNewViolations(t, h, base, "FI-03", "FI-04", "FI-05", "FI-06", "FI-11", "FI-17")
}

// TestTIP02_Rejections **مطفأة · كاش · مبلغٌ غلط · رصيدٌ ما بيكفّي** — ولا شي بيتحرّك.
func TestTIP02_Rejections(t *testing.T) {
	h := New(t)
	treasury(t, h)
	mpsSettings(h)
	mpsNoCampaign(t, h)
	cust := h.Customer()
	fundWallet(t, h, h.NewUser("admin"), cust.ID, 560)
	item := h.NewItem(1000)

	h.Setting("tips.enabled", "false")
	if _, r := tipWalletOrder(t, h, cust, item, "delivery", 10); r.Err() != "tip_not_allowed" {
		t.Fatalf("مطفأة: %s — والمنتظَر tip_not_allowed", r)
	}
	h.Setting("tips.enabled", "true")
	if _, r := tipWalletOrder(t, h, cust, item, "none", 10); r.Err() != "tip_not_allowed" {
		t.Fatalf("كاش: %s — والمنتظَر tip_not_allowed", r)
	}
	if _, r := tipWalletOrder(t, h, cust, item, "delivery", -5); r.Err() != "tip_invalid" {
		t.Fatalf("سالب: %s — والمنتظَر tip_invalid", r)
	}
	// **ولا سقفَ إلّا الرصيد** (قرارُ المالك ٢٠٢٦-١١-٢٥) — إكراميةٌ كبيرةٌ تُردّ لأنّ المحفظة ما بتكفّي، لا لسقف.
	if _, r := tipWalletOrder(t, h, cust, item, "delivery", 100000); r.Err() != "insufficient_balance" {
		t.Fatalf("أكبر من الرصيد: %s — والمنتظَر insufficient_balance", r)
	}
	// **أجرةٌ ٥٠٠ من ٥٦٠ وإكراميةُ ١٠٠ ما بتكفّي.**
	if _, r := tipWalletOrder(t, h, cust, item, "delivery", 100); r.Err() != "insufficient_balance" {
		t.Fatalf("ما بيكفّي: %s — والمنتظَر insufficient_balance", r)
	}
	if bal, reserved, _ := walletLayers(t, h, cust.ID); bal != 560 || reserved != 0 {
		t.Fatalf("**رفضٌ حرّك المحفظة**: رصيد %d محجوز %d", bal, reserved)
	}
	if r := h.GET("/api/v1/public/platform", ""); r.Code != http.StatusOK {
		t.Fatalf("المنصّة: %s", r)
	} else if tips, _ := r.JSON()["tips"].(map[string]any); tips["enabled"] != true {
		t.Fatalf("**`/public/platform` ما بيقول إنّ الإكرامية شغّالة**: %s", r)
	}
}

// TestTIP03_CancelReleases **الإلغاءُ بيفكّ الحجز** — ولا سطر بالدفتر.
func TestTIP03_CancelReleases(t *testing.T) {
	h := New(t)
	treasury(t, h)
	tipSettings(h, "0")
	mpsNoCampaign(t, h)
	base := financialBaseline(t, h)
	cust := h.Customer()
	fundWallet(t, h, h.NewUser("admin"), cust.ID, 1000)
	oid, made := tipWalletOrder(t, h, cust, h.NewItem(1000), "delivery", 10)
	if made.Code >= 400 {
		t.Fatalf("الإنشاء: %s", made)
	}
	if r := h.POST("/api/v1/orders/"+oid+"/cancel", cust.Token, map[string]any{"reason": "غيّرت رأيي"}); r.Code >= 400 {
		t.Fatalf("الإلغاء: %s", r)
	}
	if r := readTip(t, h, oid); r.reserved != 0 || r.paid {
		t.Fatalf("بعد الإلغاء: %+v", r)
	}
	if _, reserved, _ := walletLayers(t, h, cust.ID); reserved != 0 {
		t.Fatalf("**الحجزُ ما انفكّ**: %d", reserved)
	}
	if c := tipLine(t, h, oid, cust.ID); c != 0 {
		t.Fatalf("**سطرُ إكرامية لطلبٍ ملغى**: %d", c)
	}
	assertNewViolations(t, h, base, "FI-03", "FI-04", "FI-05", "FI-11")
}

// TestTIP04_WalletScopeSwitch **التبديلُ للمحفظة مع إكرامية بيحجزها، والرجوعُ للكاش بيفكّها.**
func TestTIP04_WalletScopeSwitch(t *testing.T) {
	h := New(t)
	treasury(t, h)
	tipSettings(h, "0")
	mpsNoCampaign(t, h)
	base := financialBaseline(t, h)
	cust := h.Customer()
	fundWallet(t, h, h.NewUser("admin"), cust.ID, 1000)
	oid, made := tipWalletOrder(t, h, cust, h.NewItem(1000), "none", 0)
	if made.Code >= 400 {
		t.Fatalf("الإنشاء: %s", made)
	}
	total, _ := made.JSON()["total"].(float64)

	sw := h.POSTKey("/api/v1/orders/"+oid+"/wallet-scope", cust.Token, uniq("ws"), map[string]any{
		"wallet_scope": "delivery", "expected_total": int64(total), "tip": 5,
	})
	if sw.Code >= 400 {
		t.Fatalf("التبديل للمحفظة: %s", sw)
	}
	if sw.JSON()["tip"] != float64(5) {
		t.Fatalf("الردّ ما فيه الإكرامية: %s", sw)
	}
	if r := readTip(t, h, oid); r.reserved != 5 {
		t.Fatalf("بعد التبديل: %+v — والمنتظَر محجوزة ٥", r)
	}
	// **نفسُ النطاق بإكراميةٍ تانية** — بتتبدّل وحدها.
	if r := h.POSTKey("/api/v1/orders/"+oid+"/wallet-scope", cust.Token, uniq("ws"), map[string]any{
		"wallet_scope": "delivery", "expected_total": int64(total), "tip": 10,
	}); r.Code >= 400 || readTip(t, h, oid).reserved != 10 {
		t.Fatalf("تبديلُ الإكرامية وحدها: %s · %+v", r, readTip(t, h, oid))
	}
	if r := h.POSTKey("/api/v1/orders/"+oid+"/wallet-scope", cust.Token, uniq("ws"), map[string]any{
		"wallet_scope": "none", "expected_total": int64(total),
	}); r.Code >= 400 {
		t.Fatalf("الرجوع للكاش: %s", r)
	}
	if r := readTip(t, h, oid); r.reserved != 0 || r.amount != 0 {
		t.Fatalf("**الكاشُ ما فكّ الإكرامية**: %+v", r)
	}
	if _, reserved, _ := walletLayers(t, h, cust.ID); reserved != 0 {
		t.Fatalf("محجوزٌ باقي: %d", reserved)
	}
	assertNewViolations(t, h, base, "FI-03", "FI-04", "FI-05", "FI-11")
}

// TestTIP05_UndeliverReturnsTip **تصحيحُ تسليمٍ خاطئ** — الإكراميةُ ترجع كاملةً والسطورُ صفر.
func TestTIP05_UndeliverReturnsTip(t *testing.T) {
	h := New(t)
	tid := treasury(t, h)
	tipSettings(h, "10")
	mpsNoCampaign(t, h)
	base := financialBaseline(t, h)
	cust := h.Customer()
	fundWallet(t, h, h.NewUser("admin"), cust.ID, 1000)
	oid, made := tipWalletOrder(t, h, cust, h.NewItem(1000), "delivery", 20)
	if made.Code >= 400 {
		t.Fatalf("الإنشاء: %s", made)
	}
	drv := h.driverOf(oid)
	deliverOrder(t, h, oid, drv)
	if r := undeliver(h, h.NewUser("admin"), oid, "الطلب انلغى فعلاً"); r.Code >= 400 {
		t.Fatalf("التصحيح: %s", r)
	}
	if c, d, tr := tipLine(t, h, oid, cust.ID), tipLine(t, h, oid, drv.ID), tipLine(t, h, oid, tid); c != 0 || d != 0 || tr != 0 {
		t.Fatalf("بعد التصحيح: زبون %d · كابتن %d · خزينة %d — والمنتظَر أصفار", c, d, tr)
	}
	if r := readTip(t, h, oid); r.paid || r.reserved != 0 {
		t.Fatalf("بعد التصحيح: %+v", r)
	}
	assertNewViolations(t, h, base, "FI-03", "FI-04", "FI-05", "FI-11")
}
