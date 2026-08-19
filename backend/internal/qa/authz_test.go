package qa

// **الطبقةُ الثانية — التخويلُ بين حسابين: IDOR.**
//
// المعرّفات: `SEC-IDOR-*` · الوسم: `@api @security @critical @release`
//
// # ولماذا لا يكفي ٤٠١
//
// **حارسُ «بلا توكن يُردّ» يمرّ في نظامٍ مكسورٍ تماما**: التوكنُ صحيحٌ
// **والمورد ليس لصاحبه.** وهذه أشيعُ ثغرةٍ في واجهاتِ REST، **ولا
// يكشفها إلّا حسابان حقيقيّان.**
//
// **وحسابان يحتاجان رمزَي واتساب على هاتفين** — وهو ما جعلها
// `NOT VERIFIED` في تدقيق ٢٠٢٦-٠٨-١٩. **والمصنعُ يصنعهما في مللي ثانية.**

import (
	"fmt"
	"net/http"
	"testing"
)

// TestSECIDOR_Addresses **عنوانُ زيدٍ لا يقرؤه عمرو ولا يعدّله ولا
// يحذفه.**
func TestSECIDOR_Addresses(t *testing.T) {
	h := New(t)
	victim, attacker := h.Customer(), h.Customer()

	made := h.POST("/api/v1/my/addresses", victim.Token, map[string]any{
		"area_building": "الأمين — بناء ٣", "street": "شارع الاختبار",
		"floor": "2", "kind": "home", "lat": 35.9506, "lng": 39.0094,
	})
	if made.Code != http.StatusCreated && made.Code != http.StatusOK {
		t.Fatalf("SEC-IDOR: تعذّر تجهيزُ عنوانِ الضحيّة: %s", made)
	}
	id, _ := made.JSON()["id"].(string)
	if id == "" {
		t.Fatalf("SEC-IDOR: الردُّ بلا معرّف: %s", made)
	}

	// **والضحيّةُ يقرأ عنوانَه** — وإلّا كان ما بعده بلا معنى.
	if got := h.GET("/api/v1/my/addresses", victim.Token); got.Code != http.StatusOK {
		t.Fatalf("SEC-IDOR-000 صاحبُ العنوان لا يقرؤه: %s", got)
	}

	for _, c := range []struct{ ID, Method, Path string }{
		{"SEC-IDOR-001", "PATCH", "/api/v1/my/addresses/" + id},
		{"SEC-IDOR-002", "DELETE", "/api/v1/my/addresses/" + id},
		{"SEC-IDOR-003", "POST", "/api/v1/my/addresses/" + id + "/default"},
	} {
		t.Run(c.ID, func(t *testing.T) {
			body := map[string]any{"area_building": "مسروق", "lat": 1.0, "lng": 1.0}
			if c.Method != "PATCH" {
				body = nil
			}
			got := h.Call(c.Method, c.Path, attacker.Token, body, nil)
			if got.Code < 400 {
				t.Errorf("%s %s على عنوانِ غيرِه: يُنتظر رفضٌ ووقع %s",
					c.Method, c.Path, got)
			}
		})
	}

	// **والعنوانُ ما زال للضحيّة** — رفضٌ في الردّ وتغييرٌ في القاعدة
	// يُقرأ نجاحاً.
	var owner string
	_ = h.Pool.QueryRow(t.Context(),
		`SELECT user_id::text FROM user_addresses WHERE id = $1::uuid`, id).Scan(&owner)
	if owner != "" && owner != victim.ID {
		t.Errorf("SEC-IDOR-004 تغيّر مالكُ العنوان في القاعدة: %s ≠ %s", owner, victim.ID)
	}
}

// TestSECIDOR_Orders **طلبُ زيدٍ لا يفتحه عمرو.**
func TestSECIDOR_Orders(t *testing.T) {
	h := New(t)
	victim, attacker := h.Customer(), h.Customer()
	item := h.NewItem(1000)

	made := h.POSTKey("/api/v1/orders", victim.Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Skipf("SEC-IDOR: تعذّر تجهيزُ طلبِ الضحيّة (%s) — يُراجَع مصنعُ الطلب", made)
	}
	id, _ := made.JSON()["id"].(string)

	for _, c := range []struct{ ID, Path string }{
		{"SEC-IDOR-010", "/api/v1/my/orders/" + id},
		{"SEC-IDOR-012", "/api/v1/orders/" + id + "/messages"},
	} {
		t.Run(c.ID, func(t *testing.T) {
			got := h.GET(c.Path, attacker.Token)
			if got.Code < 400 {
				t.Errorf("GET %s بحسابٍ آخر: يُنتظر رفضٌ ووقع %s", c.Path, got)
			}
		})
	}

	// ══════════════════════════════════════════════════════════════════
	// **SEC-IDOR-011 — والشكوى: لا تسريبَ ولو ردَّ ٢٠٠**
	// ══════════════════════════════════════════════════════════════════
	//
	// **والاستعلامُ مقيَّدٌ بصاحبه** (`AND o.customer_id = $2`) فيردّ
	// `null` للغريب. **فالخاصّيّةُ التي تُقاس هي ألّا يُسرَّب شيء** — لا
	// أن يكون الرمزُ ٤٠٣.
	//
	// **وحارسٌ يشترط ٤٠٣ يسقط على نظامٍ سليم** — ثمّ يُطفأ لأنّه يكذب.
	t.Run("SEC-IDOR-011", func(t *testing.T) {
		got := h.GET("/api/v1/my/orders/"+id+"/complaint", attacker.Token)
		if got.Code < 400 {
			if tk := got.JSON()["ticket"]; tk != nil {
				t.Errorf("SEC-IDOR-011 **تسريب**: شكوى غيرِه ظهرت: %v", tk)
			}
		}
	})
}

// TestVAL_040_ForeignOrderComplaintCode **وطلبٌ ليس لك يُردّ ٤٠٤.**
//
// المعرّف: `VAL-040` · الوسم: `@api @critical @release`
//
// # وكانت ملاحظةً حمراءَ ثمّ أُصلحت
//
// **سُجّلت `OBS-001` يوم ٢٠٢٦-٠٨-١٩** حين كشفها `SEC-IDOR-011`: شكوى
// طلبِ غيرِه كانت تردّ ٢٠٠ بـ`ticket: null`.
//
// **ولم تكن تسريباً** — الاستعلامُ مقيَّدٌ بصاحبه فلا يخرج منه شيء.
// **لكنّ «لا شكوى» و«ليس طلبَك» جوابان لمعنيين**، وهي عائلة `BUG-005`:
// **رابطٌ قديمٌ يبقى صالحاً في الظاهر ولا شيءَ يقول لصاحبه أن يعود.**
//
// **وأُصلحت ٢٠٢٦-٠٨-٢٠ فانتقلت من `OBS` إلى `VAL`** — وما صار يحرس
// سلوكاً قائماً **يحجب الإصدارَ إن انكسر.**
func TestVAL_040_ForeignOrderComplaintCode(t *testing.T) {
	h := New(t)
	victim, attacker := h.Customer(), h.Customer()
	item := h.NewItem(1000)
	made := h.POSTKey("/api/v1/orders", victim.Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Skipf("VAL-040 تعذّر التجهيز: %s", made)
	}
	id, _ := made.JSON()["id"].(string)
	if got := h.GET("/api/v1/my/orders/"+id+"/complaint", attacker.Token); got.Code == 200 {
		t.Errorf("VAL-040 شكوى طلبٍ ليس له: يُنتظر 404 ووقع %s", got)
	}
}

// TestSECIDOR_WalletIsOwn **والمحفظةُ محفظةُ صاحبها.**
//
// **ولا معرّفَ في المسار** — فالخطرُ ليس IDOR بل خلطُ الحسابات: **من
// يقرأ المحفظةَ من التوكن يقرأ الصحيحة**، ومن يقرؤها من حقلٍ في الجسم
// يُخدَع.
func TestSECIDOR_WalletIsOwn(t *testing.T) {
	h := New(t)
	a, b := h.Customer(), h.Customer()
	seed(t, h, a.ID, 5000)

	ra, rb := h.GET("/api/v1/my/wallet", a.Token), h.GET("/api/v1/my/wallet", b.Token)
	if ra.Code != http.StatusOK || rb.Code != http.StatusOK {
		t.Fatalf("SEC-IDOR-020 تعذّرت قراءةُ المحفظة: %s · %s", ra, rb)
	}
	if fmt.Sprint(ra.JSON()["balance"]) == fmt.Sprint(rb.JSON()["balance"]) &&
		fmt.Sprint(ra.JSON()["balance"]) != "0" {
		t.Errorf("SEC-IDOR-020 حسابان يريان الرصيدَ نفسَه: %v", ra.JSON()["balance"])
	}
}

// seed **يودع رصيداً مباشرةً في الدفتر** — ولا يمرّ ببابِ إدارة.
//
// **ودفترُ المال لا يُمسّ إلّا بقرارٍ صريح** (CLAUDE.md) — **وهذه قاعدةُ
// اختبارٍ لا الإنتاج**، و`testdb` يرفض غيرَها.
func seed(t *testing.T, h *Harness, userID string, amount int64) {
	t.Helper()
	_, err := h.Pool.Exec(t.Context(), `
		INSERT INTO wallets (user_id, balance) VALUES ($1::uuid, $2)
		ON CONFLICT (user_id) DO UPDATE SET balance = EXCLUDED.balance`, userID, amount)
	if err != nil {
		t.Skipf("qa: تعذّر تجهيزُ المحفظة (%v) — يُراجَع مخطّطُ المحافظ", err)
	}
}

// orderBody **أصغرُ طلبٍ صحيح** — ويُستعمل في كلّ اختبارٍ يحتاج طلباً.
func orderBody(it *Item, qty int) map[string]any {
	return map[string]any{
		"items":          []map[string]any{{"menu_item_id": it.ID, "qty": qty}},
		"address_text":   "الرقة — شارع الاختبار",
		"lat":            35.9506,
		"lng":            39.0094,
		"payment_method": "cash",
	}
}
