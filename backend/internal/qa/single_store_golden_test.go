package qa

// ══════════════════════════════════════════════════════════════════════
// **الطلبُ من متجرٍ واحدٍ لا يتبدّل** — `T23` (طلبٌ بعدّة محطّات، ٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// **ميزةُ المحطّات تمسّ آلةَ الحالات والتسويةَ والحمولات** — **والطلبُ من
// متجرٍ واحدٍ هو كلُّ طلبات الإنتاج اليوم.** **فتُصوَّر دورتُه الكاملة**:
// أحداثُه وبنودُه وقيودُه وتسويتُه وصندوقُ سائقه وإشعاراتُ أطرافه وحمولةُ
// كلّ طرفٍ في كلّ خطوة، **في الوضعين** — **وتُقارَن الصورةُ بعد كلّ دفعة.**
// **وطابقت الرأسَ قبل الميزة (`ef5c2655`) حرفاً، ثمّ الإنتاجَ (`d4faf0d9`) بعد الدمج.**
//
// **وما يتبدّل من تشغيلٍ لتشغيل يُطبَّع** — المعرّفاتُ والأوقاتُ والأسماءُ
// والهواتفُ الفريدةُ ورقمُ الطلب — **وما سواه يُقارَن حرفاً.**
//
// # ومتى تُعاد الصورة
//
// **حين يتبدّل سلوكُ الطلب من متجرٍ واحدٍ بقرارٍ مقصود** (حقلٌ جديدٌ في حمولة،
// قيدٌ جديد) — **لا لتمرير فحصٍ سقط**:
//
//	QA_GOLDEN_UPDATE=1 go test -count=1 -run TestSingleStore_Unchanged ./internal/qa/
//
// **ويُراجَع الفرقُ في `git diff` قبل الالتزام.** **والإعادةُ تُشغّل الدورةَ
// مرّتين وتسقط إن افترقتا** — **فلا تُحفظ صورةٌ فيها ما يتبدّل وحدَه.**

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TestSingleStore_Unchanged **دورةُ طلبٍ من متجرٍ واحدٍ كما كانت على الرأس** — `T23`.
func TestSingleStore_Unchanged(t *testing.T) {
	for _, mode := range []string{"platform", "merchants"} {
		t.Run(mode, func(t *testing.T) {
			file := filepath.Join("testdata", "single_store_"+mode+".golden.json")
			got := singleStoreJourney(t, mode)
			if os.Getenv("QA_GOLDEN_UPDATE") != "" {
				// **ودورةٌ ثانيةٌ باسمٍ آخرَ تُثبت أنّ الصورةَ لا تحمل ما يتبدّل وحدَه.**
				var again string
				t.Run("again", func(t *testing.T) { again = singleStoreJourney(t, mode) })
				if again != got {
					t.Fatalf("**الدورةُ لا تُعيد نفسَها — في الصورة ما يتبدّل وحدَه**:\n%s",
						goldenDiff(got, again))
				}
				if err := os.MkdirAll("testdata", 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(file, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				t.Logf("كُتبت الصورة: %s", file)
				return
			}
			want, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("الصورةُ غيرُ موجودة (%v) — تُكتب بـ QA_GOLDEN_UPDATE=1", err)
			}
			if strings.ReplaceAll(string(want), "\r\n", "\n") != got {
				t.Fatalf("**تبدّل الطلبُ من متجرٍ واحد** (%s):\n%s", file, goldenDiff(string(want), got))
			}
		})
	}
}

// goldenDiff **كلُّ سطرٍ يفترق فيه النصّان** (حتّى أربعين) — وما قبل أوّلها.
//
// **وأوّلُ فرقٍ وحدَه لا يكفي**: من أصلحه ثمّ أعاد التشغيلَ وجد الثاني بعد
// نصف ساعة. **والسطورُ بعد الأوّل تُقارَن بموضعها** — فسطرٌ زائدٌ يُظهر ما بعده
// كلَّه فرقاً، **وهو نفسُه دليلٌ على موضع الزيادة.**
func goldenDiff(want, got string) string {
	a := strings.Split(strings.ReplaceAll(want, "\r\n", "\n"), "\n")
	b := strings.Split(got, "\n")
	var out []string
	diffs := 0
	for i := 0; i < len(a) || i < len(b); i++ {
		var x, y string
		if i < len(a) {
			x = a[i]
		}
		if i < len(b) {
			y = b[i]
		}
		if x == y {
			continue
		}
		if diffs == 0 {
			for j := max(0, i-4); j < i && j < len(a); j++ {
				out = append(out, "    "+a[j])
			}
		}
		if diffs++; diffs > 40 {
			out = append(out, "  …")
			break
		}
		out = append(out, fmt.Sprintf("السطر %d:\n  - %s\n  + %s", i+1, x, y))
	}
	if diffs == 0 {
		return "(لا فرق)"
	}
	return strings.Join(out, "\n")
}

// singleStoreJourney **دورةُ طلبٍ كاملةٌ من متجرٍ واحد** — وصورتُها المطبَّعة.
//
//	المنصّة   نقداً · متجرٌ بالمحفظة · مندوبٌ · عمولةُ ١٠٪ · المكتبُ يقبل ثمّ يُبلغ يدويّاً
//	المتاجر   بالمحفظة · متجرٌ نقديٌّ يدفع له السائق · المتجرُ يقبل بتحضيرِ ١٠ دقائق
//
// **ثمّ في الوضعين**: سائقٌ يقبل من الطابور · يصل · يستلم · في الطريق · عند
// الباب · يسلّم — **بالأبواب التي تناديها التطبيقات لا بكتابة الحال.**
func singleStoreJourney(t *testing.T, mode string) string {
	t.Helper()
	h := New(t)
	f := h.Factory()
	treasuryID := treasury(t, h)
	h.Setting("platform.orders_mode", `"`+mode+`"`)
	h.Setting("orders.auto_transfer", "false")
	h.Setting("orders.auto_accept_min", "0")
	h.Setting("orders.auto_dispatch", "true")
	h.Setting("orders.notify_merchant", "true")
	h.Setting("drivers.assignment_mode", `"queue"`)
	h.Setting("drivers.max_active_orders", "5")
	h.Setting("drivers.cash_limit", "9000000")
	h.Setting("delivery.fee", "400")
	h.Setting("delivery.by_distance", "false")
	// **وكودُ التسليم عشوائيٌّ** — **ومطفأٌ افتراضاً**، فيُثبَّت مطفأً لا يُترك للافتراض.
	h.Setting("delivery.code_required", "false")

	rep := f.RepAccount()
	m := f.Merchant(OwnedByRep(rep.ID), Commission(10))
	if mode == "merchants" {
		if _, err := h.Pool.Exec(ctxBG(),
			`UPDATE merchants SET settlement_method = 'cash' WHERE id = $1::uuid`, m.ID); err != nil {
			t.Fatal(err)
		}
	}
	item := h.NewItemPriced(m, 1000, 1200)
	cust := h.Customer()
	admin := h.NewUser("admin")
	ownerTok := h.TokenFor(m.Owner.ID, "merchant")

	views := map[string]any{}
	step := func(label string, r Res) map[string]any {
		t.Helper()
		if r.Code >= 400 {
			t.Fatalf("%s رُدّ: %s", label, r)
		}
		j := r.JSON()
		views[label] = j
		return j
	}

	body := orderBody(item, 2)
	if mode == "merchants" {
		f.Credit(cust.ID, 10000, "topup")
		body["payment_method"] = "wallet"
	}
	created := step("01 customer/create", h.POSTKey("/api/v1/orders", cust.Token, uniq("t23"), body))
	oid, _ := created["id"].(string)
	if oid == "" {
		t.Fatal("لا معرّفَ للطلب")
	}

	if mode == "platform" {
		step("02 ops/accept", h.POST("/api/v1/admin/orders/"+oid+"/transition", admin.Token,
			map[string]any{"to": "accepted"}))
		step("03 ops/send", h.POST("/api/v1/admin/orders/"+oid+"/whatsapp", admin.Token,
			map[string]any{"channel": "manual"}))
	} else {
		step("02 merchant/accept", h.POST("/api/v1/merchant/orders/"+oid+"/transition", ownerTok,
			map[string]any{"to": "accepted", "prep_minutes": 10}))
	}
	if got := h.statusOf(oid); got != "dispatching" {
		t.Fatalf("**لم ينزل الطلبُ إلى الطابور**: %q", got)
	}

	drv := f.Driver(OnShift())
	step("04 driver/accept", h.POST("/api/v1/driver/orders/"+oid+"/accept", drv.Token, nil))
	for i, to := range []string{"at_pickup", "picked_up", "on_the_way", "at_dropoff"} {
		step(fmt.Sprintf("%02d driver/%s", 5+i, to), h.POST("/api/v1/driver/orders/"+oid+"/transition",
			drv.Token, map[string]any{"to": to}))
	}
	h.authorizeProofExempt(oid)
	step("09 driver/delivered", h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
		map[string]any{"to": "delivered"}))

	step("10 customer/detail", h.GET("/api/v1/my/orders/"+oid, cust.Token))
	step("11 merchant/detail", h.GET("/api/v1/merchant/orders/"+oid, ownerTok))
	step("12 ops/detail", h.GET("/api/v1/admin/orders/"+oid, admin.Token))

	// ── ما في القاعدة ─────────────────────────────────────────────────
	q := func(sql string, args ...any) []map[string]any {
		t.Helper()
		rows, err := h.Pool.Query(ctxBG(), sql, args...)
		if err != nil {
			t.Fatalf("قراءة: %v\n%s", err, sql)
		}
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			vals, err := rows.Values()
			if err != nil {
				t.Fatal(err)
			}
			row := map[string]any{}
			for i, fd := range rows.FieldDescriptions() {
				row[fd.Name] = vals[i]
			}
			out = append(out, row)
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		return out
	}
	// **وإشعاراتُ الأطراف المعروفة وحدَها** — **والمكتبُ يُشعَر كلُّ موظّفيه، وعددُهم في
	// قاعدةٍ تتقاسمها اختباراتُ الحزمة يتبدّل.**
	parties := []string{cust.ID, m.Owner.ID, drv.ID, rep.ID}
	snap := map[string]any{
		"views": views,
		"order": q(`
			SELECT status, payment_method, subtotal, delivery_fee, discount, total, wallet_paid,
			       cash_due, driver_fee, platform_commission, sources_fee, prep_minutes,
			       promo_delivery_waived, merchant_id::text, driver_id::text, customer_id::text,
			       ended_by, fault, cancel_reason, goods_settled_to,
			       accepted_at IS NOT NULL AS accepted, sent_to_merchant_at IS NOT NULL AS sent,
			       picked_up_at IS NOT NULL AS picked, delivered_at IS NOT NULL AS delivered,
			       closed_at IS NOT NULL AS closed
			FROM orders WHERE id = $1::uuid`, oid),
		"events": q(`
			SELECT from_status, to_status, note, actor_id::text AS actor
			FROM order_events WHERE order_id = $1::uuid ORDER BY id`, oid),
		"items": q(`
			SELECT name, qty, unit_price, merchant_price, merchant_id::text, merchant_settlement_method,
			       options::text AS options, offer_cut
			FROM order_items WHERE order_id = $1::uuid ORDER BY name`, oid),
		"ledger": q(`
			SELECT kind, user_id::text AS party, amount, note
			FROM wallet_transactions WHERE ref = $1 ORDER BY id`, oid),
		"settlements": q(`
			SELECT merchant_id::text, method, amount, reversed_amount, adjusted_amount, state,
			       COALESCE(paid_via, '') AS paid_via, commission_waived, note
			FROM merchant_settlements WHERE order_id = $1::uuid ORDER BY merchant_id`, oid),
		"cash_box": q(`SELECT held FROM driver_cash_boxes WHERE driver_id = $1::uuid`, drv.ID),
		// **وترتيبُها بالطرف ثمّ بالنصّ لا بالوقت** — **إشعاران في معاملةٍ واحدةٍ
		// يتساويان وقتاً، ومعرّفُهما عشوائيّ.**
		"notifications": q(`
			SELECT user_id::text AS party, kind, title, body, entity, href
			FROM notifications WHERE entity_id = $1 AND user_id::text = ANY($2)
			ORDER BY array_position($2::text[], user_id::text), kind, title, body, href`,
			oid, parties),
	}

	// ── التطبيع ───────────────────────────────────────────────────────
	//
	// **و«قرب …» في عنوان السائق يقرؤه زنادُ القاعدة من أقرب معلمٍ أو متجر**
	// (`orders_near_landmark`، هجرة ٠٤٦٧) — **ومتاجرُ الاختبارات الأخرى تقف على
	// النقطة نفسِها**، فيتبدّل الاسمُ بما سبق في الحزمة لا بالشيفرة. **فيُطبَّع
	// بنصّه المحفوظ** — ويبقى أنّ السائقَ يراه ملصقاً بالعنوان.
	var number int64
	var mPhone, near string
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT o.number, COALESCE(m.phone, ''), COALESCE(o.near_landmark, '')
		FROM orders o JOIN merchants m ON m.id = o.merchant_id
		WHERE o.id = $1::uuid`, oid).Scan(&number, &mPhone, &near); err != nil {
		t.Fatal(err)
	}
	known := map[string]string{
		near: "<near_landmark>",
		oid:  "<order>", treasuryID: "<treasury>", fmt.Sprintf("#%d", number): "#<number>",
		cust.ID: "<customer>", cust.Phone: "<customer_phone>", cust.Name: "<customer_name>",
		m.ID: "<merchant>", m.Name: "<merchant_name>", mPhone: "<merchant_phone>",
		m.Owner.ID: "<owner>", m.Owner.Phone: "<owner_phone>", m.Owner.Name: "<owner_name>",
		item.ID: "<item>", item.Name: "<item_name>", item.SectionID: "<section>",
		rep.ID: "<rep>", rep.Phone: "<rep_phone>", rep.Name: "<rep_name>",
		drv.ID: "<driver>", drv.Phone: "<driver_phone>", drv.Name: "<driver_name>",
		admin.ID: "<admin>", admin.Phone: "<admin_phone>", admin.Name: "<admin_name>",
	}
	for i, r := range q(`SELECT id::text FROM order_items WHERE order_id = $1::uuid ORDER BY name`, oid) {
		known[r["id"].(string)] = fmt.Sprintf("<line%d>", i+1)
	}
	raw, err := json.Marshal(snap)
	if err != nil {
		t.Fatal(err)
	}
	var tree any
	if err := json.Unmarshal(raw, &tree); err != nil {
		t.Fatal(err)
	}
	out, err := json.MarshalIndent(goldenNormalize(tree, "", goldenReplacer(known)), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return string(out) + "\n"
}

var (
	goldenTS   = regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}(:?\d{2})?)?`)
	goldenUUID = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
)

// goldenVolatile **مفاتيحُ تتبدّل من تشغيلٍ لتشغيل** — تُكتب علامةً لا قيمة.
//
// **ورقمُ الطلب تسلسلٌ في قاعدةٍ تتقاسمها اختباراتُ الحزمة**، **والهاتفُ
// المقنَّعُ يحمل آخرَ أرقامِ هاتفٍ فريد.**
// goldenDriverAddr **عنوانُ الزبون في الدورة** — وما يُلصق به بعده «قرب …».
const goldenDriverAddr = "الرقة — شارع الاختبار"

var goldenVolatile = map[string]bool{
	"number":                true,
	"customer_phone_masked": true,
	// **واسمُ المنطقة يتبع ما بذرته اختباراتٌ قبله في القاعدة نفسِها** — والمعرّفُ مطبَّعٌ أصلاً.
	"zone_name": true,
}

// goldenReplacer **المعروفُ باسمه — والأطولُ أوّلاً** فلا يُستبدَل جزءُ قيمةٍ قبل كلِّها.
func goldenReplacer(known map[string]string) *strings.Replacer {
	keys := make([]string, 0, len(known))
	for k := range known {
		if k != "" {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		if len(keys[i]) != len(keys[j]) {
			return len(keys[i]) > len(keys[j])
		}
		return keys[i] < keys[j]
	})
	pairs := make([]string, 0, 2*len(keys))
	for _, k := range keys {
		pairs = append(pairs, k, known[k])
	}
	return strings.NewReplacer(pairs...)
}

// goldenNormalize **يطبّع الشجرة**: المعروفُ باسمه · الأوقاتُ · بقيّةُ المعرّفات.
func goldenNormalize(v any, key string, r *strings.Replacer) any {
	if goldenVolatile[key] && v != nil {
		return "<volatile>"
	}
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, e := range x {
			out[k] = goldenNormalize(e, k, r)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = goldenNormalize(e, key, r)
		}
		return out
	case string:
		// **و«قرب …» قد تحمل أرقاماً تُقنَّع في حمولة السائق** (اسمُ متجرٍ من اختبارٍ آخر)
		// **أو حيّاً بذره اختبارٌ قبله** — فلا يلتقطها نصُّها المحفوظ؛ **فيُطبَّع ما بعد العنوان.**
		if key == "address_text" && strings.HasPrefix(x, goldenDriverAddr+" · ") {
			return goldenDriverAddr + " · <near_landmark>"
		}
		s := goldenTS.ReplaceAllString(r.Replace(x), "<ts>")
		return goldenUUID.ReplaceAllString(s, "<uuid>")
	}
	return v
}
