package qa

// ══════════════════════════════════════════════════════════════════════
// **عائلةُ العروض والخصومات — من يربح ومن يخسر بكلّ عرض** (`MPS`)
// ══════════════════════════════════════════════════════════════════════
//
// الكوبونُ بأنواعه، وكودُ «توصيل مجاني» مع حملة التوصيل المجاني، وخصمُ الصنف
// (على المنصّة أو على المتجر)، ومكافأةُ الدعوة. **وكلُّ رقمٍ يُقرأ من الدفتر
// بعد طلبٍ مرّ بدورة حياته كاملةً** — لا من عمودٍ كُتب بيد.
//
// والإعداداتُ ثابتةٌ في كلّ اختبار: عمولةُ المتجر ١٠٪، نصيبُ المندوب ١٠٪،
// هامشٌ صفر، أجرةُ توصيلٍ ٥٠٠ ثابتة، وحصّةُ المنصّة من الأجرة ١٠٪ (السائقُ ٤٥٠).

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/referrals"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// mpsSettings **الإعداداتُ الثابتةُ للعائلة** — فالأرقامُ تُحسب سلفاً بلا تخمين.
func mpsSettings(h *Harness) {
	h.Setting("merchants.commission_percent", "10")
	h.Setting("sales.commission_percent", "10")
	h.Setting("pricing.margin_fixed", "0")
	h.Setting("delivery.fee", "500")
	h.Setting("delivery.by_distance", "false")
	h.Setting("delivery.share_standard", "10")
	h.Setting("delivery.share_standard_mode", `"percent"`)
}

// mpsNoCampaign **لا حملةَ توصيلٍ مجانيٍّ شغّالةٌ من اختبارٍ آخر** — وإلّا أكلت الأجرة.
func mpsNoCampaign(t *testing.T, h *Harness) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE free_delivery_campaigns SET active = false`); err != nil {
		t.Fatal(err)
	}
}

// mpsCampaign **حملةُ توصيلٍ مجانيٍّ للعاديّ شغّالةٌ الآن** — تُطفأ وتُمحى بعد الاختبار.
func mpsCampaign(t *testing.T, h *Harness) string {
	t.Helper()
	mpsNoCampaign(t, h)
	now := time.Now().UTC()
	var id string
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO free_delivery_campaigns (active, starts_at, ends_at, kinds)
		VALUES (true, $1, $2, ARRAY['standard']) RETURNING id::text`,
		now.Add(-time.Hour), now.Add(24*time.Hour)).Scan(&id); err != nil {
		t.Fatalf("تعذّر إنشاءُ الحملة: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `UPDATE free_delivery_campaigns SET active = false WHERE id = $1::uuid`, id)
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM free_delivery_campaigns WHERE id = $1::uuid`, id)
	})
	return id
}

// mpsCode **كودُ خصمٍ فعّال** — ويُمحى مع استعمالاته بعد الاختبار.
func mpsCode(t *testing.T, h *Harness, kind string, value int64, oncePerUser bool) string {
	t.Helper()
	code := strings.ToUpper(uniq("MPS"))
	var id string
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO promo_codes (code, kind, value, once_per_user, active)
		VALUES ($1, $2, $3, $4, true) RETURNING id::text`, code, kind, value, oncePerUser).Scan(&id); err != nil {
		t.Fatalf("تعذّر إنشاءُ الكود: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM promo_redemptions WHERE promo_id = $1::uuid`, id)
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM promo_codes WHERE id = $1::uuid`, id)
	})
	return code
}

// mpsMoney **حركاتُ مال الطلب كما يقولها الدفتر** — لكلّ طرفٍ ما قُيّد له بمرجع الطلب.
type mpsMoney struct {
	delivery, driverFee, waived, discount, total, cashDue int64
	merchant, driver, rep, platform, collected            int64
	claimed                                               bool
}

func mpsRead(t *testing.T, h *Harness, oid string) mpsMoney {
	t.Helper()
	var m mpsMoney
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT o.delivery_fee, o.driver_fee, o.promo_delivery_waived, o.discount, o.total, o.cash_due,
		       COALESCE((SELECT sum(amount) FROM wallet_transactions WHERE ref = $1 AND kind = 'merchant_earning'), 0),
		       COALESCE((SELECT sum(amount) FROM wallet_transactions WHERE ref = $1 AND kind = 'driver_earning'), 0),
		       COALESCE((SELECT sum(amount) FROM wallet_transactions WHERE ref = $1 AND kind = 'commission'), 0),
		       COALESCE((SELECT sum(amount) FROM wallet_transactions WHERE ref = $1 AND kind = 'platform_profit'), 0),
		       COALESCE((SELECT sum(amount) FROM driver_cash_entries WHERE ref = $1 AND kind = 'order_collection'), 0),
		       EXISTS (SELECT 1 FROM free_delivery_claims c WHERE c.order_id = o.id)
		  FROM orders o WHERE o.id = $1::uuid`, oid).
		Scan(&m.delivery, &m.driverFee, &m.waived, &m.discount, &m.total, &m.cashDue,
			&m.merchant, &m.driver, &m.rep, &m.platform, &m.collected, &m.claimed); err != nil {
		t.Fatalf("قراءةُ مال الطلب: %v", err)
	}
	return m
}

// mpsPlace **طلبٌ نقديٌّ بصنفٍ واحد** — وبكودٍ إن مُرّر.
func mpsPlace(t *testing.T, h *Harness, cust *User, item *Item, code string) string {
	t.Helper()
	body := orderBody(item, 1)
	if code != "" {
		body["promo_code"] = code
	}
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("mps"), body)
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	if oid == "" {
		t.Fatalf("طلبٌ بلا معرّف: %s", made)
	}
	return oid
}

// TestMPS01_FixedCouponMoneyMovements **كوبونٌ ثابتٌ بخمسين — من حصّة المنصّة وحدَها.**
//
// بضاعةٌ ١٠٠٠ · أجرةٌ ٥٠٠ · كوبونٌ ٥٠ ⇒ الزبونُ يدفع ١٤٥٠ نقداً، المتجرُ ٩٠٠ (١٠٠٠ − عمولة ١٠٠)،
// السائقُ ٤٥٠، المندوبُ ١٠٪ من (١٠٠ − ٥٠) = ٥، والمنصّةُ ١٠٠ + ٥٠ − ٥٠ − ٥ = ٩٥.
func TestMPS01_FixedCouponMoneyMovements(t *testing.T) {
	h := New(t)
	treasury(t, h)
	mpsSettings(h)
	mpsNoCampaign(t, h)
	base := financialBaseline(t, h)

	f := h.Factory()
	r := f.RepAccount()
	m := f.Merchant(OwnedByRep(r.ID))
	item := h.NewItemFor(m, 1000)
	code := mpsCode(t, h, "fixed", 50, true)

	oid := mpsPlace(t, h, h.Customer(), item, code)
	deliverOrder(t, h, oid, h.driverOf(oid))

	got := mpsRead(t, h, oid)
	if got.delivery != 500 || got.discount != 50 || got.total != 1450 || got.cashDue != 1450 {
		t.Fatalf("الفاتورة: أجرة %d · خصم %d · مجموع %d · نقد %d — والمنتظَر 500 · 50 · 1450 · 1450",
			got.delivery, got.discount, got.total, got.cashDue)
	}
	if got.collected != 1450 {
		t.Errorf("صندوقُ السائق قبض %d — والمنتظَر 1450", got.collected)
	}
	if got.merchant != 900 {
		t.Errorf("مستحقُّ المتجر %d — والمنتظَر 900 (**الكوبونُ لا يمسّ المتجر**)", got.merchant)
	}
	if got.driver != 450 || got.driverFee != 450 {
		t.Errorf("أجرُ السائق %d (المثبَّت %d) — والمنتظَر 450", got.driver, got.driverFee)
	}
	if got.rep != 5 {
		t.Errorf("نصيبُ المندوب %d — والمنتظَر 5 (١٠٪ من العمولة ناقصَ الكوبون)", got.rep)
	}
	if got.platform != 95 {
		t.Errorf("ربحُ المنصّة %d — والمنتظَر 95", got.platform)
	}
	assertNewViolations(t, h, base, "FI-06", "FI-10", "FI-04", "FI-03")
}

// TestMPS02_FreeDeliveryCodeVsCampaign **كودُ «توصيل مجاني» يأكل نصيبَ المندوب ولا يستهلك حقَّ الحملة.**
//
// الطلبُ الأوّلُ بالكود: الزبونُ يدفع ١٠٠٠، السائقُ ٤٥٠ كاملاً، المندوبُ صفر (العمولةُ ١٠٠ والإعفاءُ ٥٠٠
// من حصّة المنصّة — قرارُ ٢٠٢٦-١٠-١٥)، والمنصّةُ ١٠٠ + ٥٠ − ٥٠٠ = −٣٥٠. **ولا صفَّ في الحملة** —
// الأجرةُ صارت صفراً قبلها. الطلبُ الثاني بلا كود: الحملةُ تُعفيه، والمندوبُ يأخذ ١٠ كاملةً
// (قرارُ ٢٠٢٦-١٠-٢٦)، والمنصّةُ −٣٦٠.
func TestMPS02_FreeDeliveryCodeVsCampaign(t *testing.T) {
	h := New(t)
	treasury(t, h)
	mpsSettings(h)
	mpsCampaign(t, h)
	base := financialBaseline(t, h)

	f := h.Factory()
	r := f.RepAccount()
	m := f.Merchant(OwnedByRep(r.ID))
	item := h.NewItemFor(m, 1000)
	code := mpsCode(t, h, "free_delivery", 0, false)
	cust := h.Customer()

	first := mpsPlace(t, h, cust, item, code)
	deliverOrder(t, h, first, h.driverOf(first))
	a := mpsRead(t, h, first)
	if a.delivery != 0 || a.waived != 500 || a.total != 1000 || a.driverFee != 450 {
		t.Fatalf("بالكود: أجرة %d · معفى %d · مجموع %d · السائق %d — والمنتظَر 0 · 500 · 1000 · 450",
			a.delivery, a.waived, a.total, a.driverFee)
	}
	if a.claimed {
		t.Fatalf("**الكودُ استهلك حقَّ الحملة** — والأجرةُ صُفّرت قبلها فلا شيءَ تُعفيه")
	}
	if a.merchant != 900 || a.driver != 450 || a.rep != 0 || a.platform != -350 {
		t.Errorf("بالكود: متجر %d · سائق %d · مندوب %d · منصّة %d — والمنتظَر 900 · 450 · 0 · -350",
			a.merchant, a.driver, a.rep, a.platform)
	}

	second := mpsPlace(t, h, cust, item, "")
	deliverOrder(t, h, second, h.driverOf(second))
	b := mpsRead(t, h, second)
	if !b.claimed || b.delivery != 0 || b.waived != 500 || b.total != 1000 {
		t.Fatalf("بالحملة: حقّ %v · أجرة %d · معفى %d · مجموع %d — **الحملةُ لم تُعطِ حقَّها بعد الكود**",
			b.claimed, b.delivery, b.waived, b.total)
	}
	if b.merchant != 900 || b.driver != 450 || b.rep != 10 || b.platform != -360 {
		t.Errorf("بالحملة: متجر %d · سائق %d · مندوب %d · منصّة %d — والمنتظَر 900 · 450 · 10 · -360",
			b.merchant, b.driver, b.rep, b.platform)
	}
	assertNewViolations(t, h, base, "FI-06", "FI-10", "FI-04", "FI-03")
}

// TestMPS03_ItemOfferWhoBears **خصمُ الصنف بمبلغٍ ٢٠٠ — على المنصّة أو على المتجر.**
//
// صنفٌ كلفتُه ١٠٠٠ ويُباع بـ١٢٠٠ (هامشٌ ٢٠٠)، والزبونُ يدفع ١٠٠٠ + ٥٠٠ = ١٥٠٠ في الحالين.
//
//	على المنصّة: المتجرُ ٩٠٠ · الهامشُ صفر · المندوبُ ١٠٪ من ١٠٠ = ١٠ · المنصّةُ ١٤٠
//	على المتجر:  الشراءُ ينزل إلى ٨٠٠ ⇒ المتجرُ ٧٢٠ · العمولةُ ٨٠ · الهامشُ ٢٠٠ ·
//	             المندوبُ ١٠٪ من ٢٨٠ = ٢٨ · المنصّةُ ٣٠٢
func TestMPS03_ItemOfferWhoBears(t *testing.T) {
	cases := []struct {
		by                          string
		merchant, rep, platform, mp int64
	}{
		{"platform", 900, 10, 140, 1000},
		{"merchant", 720, 28, 302, 800},
	}
	for _, c := range cases {
		t.Run(c.by, func(t *testing.T) {
			h := New(t)
			treasury(t, h)
			mpsSettings(h)
			mpsNoCampaign(t, h)
			base := financialBaseline(t, h)

			f := h.Factory()
			r := f.RepAccount()
			m := f.Merchant(OwnedByRep(r.ID))
			item := h.NewItemPriced(m, 1000, 1200)
			// **سعرُ البيع في الطلب يُشتقّ من الشراء والهامش لا من `menu_items.price`**
			// (`rule.SalePrice` في orders/service.go) — والعائلةُ تضبط الهامشَ صفراً،
			// **فبلا هامشٍ للصنف نفسِه يُباع بـ١٠٠٠ لا بـ١٢٠٠** ويسقط فرضُ الاختبار.
			if _, err := h.Pool.Exec(ctxBG(),
				`UPDATE menu_items SET margin_override = 200 WHERE id = $1::uuid`, item.ID); err != nil {
				t.Fatalf("هامشُ الصنف: %v", err)
			}
			admin := h.NewUser("admin")
			var offerID string
			if err := h.Pool.QueryRow(ctxBG(), `
				INSERT INTO offers (kind, title, menu_item_id, discount_amount, borne_by,
				                    starts_at, ends_at, active, created_by)
				VALUES ('discount', 'عرضُ فحص', $1::uuid, 200, $2, now() - interval '1 hour',
				        now() + interval '1 day', true, $3::uuid) RETURNING id::text`,
				item.ID, c.by, admin.ID).Scan(&offerID); err != nil {
				t.Fatalf("تعذّر إنشاءُ العرض: %v", err)
			}
			t.Cleanup(func() { _, _ = h.Pool.Exec(ctxBG(), `DELETE FROM offers WHERE id = $1::uuid`, offerID) })

			oid := mpsPlace(t, h, h.Customer(), item, "")
			var unit, mp, cut int64
			var by string
			if err := h.Pool.QueryRow(ctxBG(), `
				SELECT unit_price, merchant_price, offer_cut, COALESCE(offer_borne_by, '')
				  FROM order_items WHERE order_id = $1::uuid`, oid).Scan(&unit, &mp, &cut, &by); err != nil {
				t.Fatal(err)
			}
			if unit != 1000 || mp != c.mp || cut != 200 || by != c.by {
				t.Fatalf("البند: بيع %d · شراء %d · خصم %d · على %q — والمنتظَر 1000 · %d · 200 · %q",
					unit, mp, cut, by, c.mp, c.by)
			}
			deliverOrder(t, h, oid, h.driverOf(oid))
			got := mpsRead(t, h, oid)
			if got.total != 1500 || got.collected != 1500 {
				t.Fatalf("الزبونُ دفع %d (قُبض %d) — والمنتظَر 1500", got.total, got.collected)
			}
			if got.merchant != c.merchant || got.driver != 450 || got.rep != c.rep || got.platform != c.platform {
				t.Errorf("متجر %d · سائق %d · مندوب %d · منصّة %d — والمنتظَر %d · 450 · %d · %d",
					got.merchant, got.driver, got.rep, got.platform, c.merchant, c.rep, c.platform)
			}
			assertNewViolations(t, h, base, "FI-06", "FI-10", "FI-04", "FI-03")
		})
	}
}

// TestMPS04_CancelReleasesOncePerUserCode **طلبٌ أُلغي يُرجع الكودَ لصاحبه** — «مرّةً لكلّ مستخدم»
// تُحسب على طلبٍ حيّ، **والعدّادُ ينزل.**
func TestMPS04_CancelReleasesOncePerUserCode(t *testing.T) {
	h := New(t)
	treasury(t, h)
	mpsSettings(h)
	mpsNoCampaign(t, h)
	item := h.NewItem(1000)
	code := mpsCode(t, h, "fixed", 50, true)
	cust := h.Customer()

	first := mpsPlace(t, h, cust, item, code)
	if got := h.POST("/api/v1/orders/"+first+"/cancel", cust.Token,
		map[string]any{"reason": "غيّرتُ رأيي"}); got.Code >= 400 {
		t.Fatalf("الإلغاء: %s", got)
	}
	var used int
	if err := h.Pool.QueryRow(ctxBG(), `SELECT used_count FROM promo_codes WHERE code = $1`, code).Scan(&used); err != nil {
		t.Fatal(err)
	}
	if used != 0 {
		t.Errorf("العدّادُ %d بعد الإلغاء — والمنتظَر 0", used)
	}
	again := mpsPlace(t, h, cust, item, code)
	if got := mpsRead(t, h, again); got.discount != 50 {
		t.Fatalf("بعد الإلغاء الخصم %d — **والكودُ لم يرجع لصاحبه**", got.discount)
	}
	// **والثالثُ يُردّ** — الطلبُ الثاني حيٌّ ويحمل الكود.
	body := orderBody(item, 1)
	body["promo_code"] = code
	if third := h.POSTKey("/api/v1/orders", cust.Token, uniq("mps"), body); third.Code < 400 {
		t.Fatalf("**الكودُ صلح مرّةً ثالثة** وهو «مرّةً لكلّ مستخدم»: %s", third)
	}
}

// mpsRefSettings **إعداداتُ الدعوة في الذاكرة** — مكافأةُ الأولى ٥٠٠٠ وعند التسجيل.
type mpsRefSettings struct{}

func (mpsRefSettings) GetInt(_ context.Context, k string) int64 {
	if k == "referral.reward_1" {
		return 5000
	}
	return 0
}

func (mpsRefSettings) GetString(_ context.Context, k string) string {
	if k == "referral.reward_on" {
		return referrals.OnSignup
	}
	return ""
}

// TestMPS05_ReferralRewardPairsWithTreasury **مكافأةُ الدعوة من الخزينة — طرفان متقابلان ومرّةً واحدة.**
//
// الداعي +٥٠٠٠ في محفظته، والخزينةُ −٥٠٠٠ بالنوع نفسِه (`reward`)، **وصرفٌ ثانٍ لا يقع.**
func TestMPS05_ReferralRewardPairsWithTreasury(t *testing.T) {
	h := New(t)
	tid := treasury(t, h)
	inviter := h.Customer()
	invitee := h.Customer()
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO referrals (inviter_id, invitee_id, rank) VALUES ($1::uuid, $2::uuid, 1)`,
		inviter.ID, invitee.ID); err != nil {
		t.Fatalf("تعذّرت الدعوة: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM referrals WHERE invitee_id = $1::uuid`, invitee.ID)
	})
	sum := func(user string) int64 {
		var n int64
		if err := h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
			WHERE user_id = $1::uuid AND kind = 'reward'`, user).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	before := sum(tid)
	svc := referrals.New(h.Pool, wallet.NewService(h.Pool), mpsRefSettings{},
		func(context.Context) string { return tid }, nil, nil)

	svc.SettleOnSignup(ctxBG(), invitee.ID, invitee.ID)
	if got := sum(inviter.ID); got != 5000 {
		t.Fatalf("مكافأةُ الداعي %d — والمنتظَر 5000", got)
	}
	if got := sum(tid) - before; got != -5000 {
		t.Fatalf("الخزينةُ تحرّكت %d — والمنتظَر -5000 (**دفترٌ لا يتوازن**)", got)
	}
	svc.SettleOnSignup(ctxBG(), invitee.ID, invitee.ID)
	if got := sum(inviter.ID); got != 5000 {
		t.Fatalf("**صُرفت المكافأةُ مرّتين**: %d", got)
	}
}
