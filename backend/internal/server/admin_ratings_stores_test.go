package server

// **والمتاجرُ في التقييمات — كما يراها السائقون.**
//
// (كشفه فحصُ المالك ٢٠٢٦-٠٨-١٦.)
//
// **جدولُ `merchant_ratings` كان يُكتب فيه ولا يُقرأ منه أبداً** — لا شاشةَ
// إدارةٍ ولا بوّابةَ متجرٍ ولا تقرير.
//
// **والتبويبُ وُضع لسؤال «من يشكو منه الناس؟»** وكان يجيبه عن السائقين
// وحدَهم، **فيبقى السؤالُ عن المتاجر بلا جوابٍ إلّا بفتح عشرين ملفّاً** —
// وهو ما بُني ليمنعه.
//
// # ونجمتان لا متوسّطٌ واحد
//
// **«بطيءٌ في التجهيز» و«سيّئُ التعامل» عيبان لا يُعالجان بشيءٍ واحد** —
// **ومتوسّطُهما يخفي أيَّهما هو**: متجرٌ سريعٌ فظٌّ ومتجرٌ لطيفٌ بطيءٌ
// يخرجان برقمٍ واحد.
//
// # والحدُّ الأدنى هو حدُّ السائقين نفسُه
//
// **وشرطان مختلفان لقائمتين في شاشةٍ واحدةٍ يُقرآن رقماً واحداً** — فيُظنّ
// المتجرُ أسوأَ من سائقٍ وهو قُيّم مرّةً وذاك مئة.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminRatings_IncludesStores(t *testing.T) {
	f := newDriverFixture(t, 1)
	ctx := context.Background()
	customer, _, _ := twoCustomers(t, f)
	driver := f.drivers[0]

	mkOrder := func() string {
		var id string
		if err := f.pool.QueryRow(ctx, `
			INSERT INTO orders (customer_id, merchant_id, driver_id, status, address_text,
				dropoff, payment_method, subtotal, delivery_fee, total, cash_due)
			VALUES ($1, $2, $3, 'delivered', 'عنوان اختبار',
				ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
				'cash', 1000, 0, 1000, 1000)
			RETURNING id::text`, customer, f.merchantID, driver).Scan(&id); err != nil {
			t.Fatalf("تعذّر الطلب: %v", err)
		}
		t.Cleanup(func() {
			c := context.Background()
			_, _ = f.pool.Exec(c, `DELETE FROM merchant_ratings WHERE order_id = $1`, id)
			_, _ = f.pool.Exec(c, `DELETE FROM orders WHERE id = $1`, id)
		})
		return id
	}
	rate := func(orderID string, speed, conduct int) {
		if _, err := f.pool.Exec(ctx, `
			INSERT INTO merchant_ratings (order_id, driver_id, merchant_id, speed_stars, conduct_stars)
			VALUES ($1, $2, $3, $4, $5)`,
			orderID, driver, f.merchantID, speed, conduct); err != nil {
			t.Fatalf("تعذّر التقييم: %v", err)
		}
	}
	// **سريعٌ فظّ** — ٥ في التجهيز و١ في التعامل، **ومتوسّطُهما ثلاثٌ يخفي
	// العيبَ الحادّ.**
	rate(mkOrder(), 5, 1)
	rate(mkOrder(), 5, 1)

	get := func(min string) map[string]any {
		req := httptest.NewRequest(http.MethodGet, "/x?min="+min, nil)
		c := context.WithValue(req.Context(), ctxUserID, driver)
		c = context.WithValue(c, ctxRoles, []string{"admin"})
		w := httptest.NewRecorder()
		f.srv.handleAdminRatings(w, req.WithContext(c))
		if w.Code != 200 {
			t.Fatalf("ردَّ %d — %s", w.Code, w.Body.String())
		}
		var env struct {
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
			t.Fatalf("ردٌّ لا يُفكّ: %v", err)
		}
		return env.Data
	}

	d := get("1")
	stores, _ := d["stores"].([]any)
	var mine map[string]any
	for _, x := range stores {
		row, _ := x.(map[string]any)
		if row["id"] == f.merchantID {
			mine = row
		}
	}
	if mine == nil {
		t.Fatalf("المتجرُ لا يظهر في التقييمات: %v — **وجدولٌ يُكتب فيه ولا يُقرأ منه**",
			stores)
	}
	// **والنجمتان تفترقان** — ولو جُمعتا في متوسّطٍ لَقُرئ «ثلاثٌ» ولا يُعرف أيُّهما.
	if mine["speed"] != float64(5) || mine["conduct"] != float64(1) {
		t.Fatalf("التجهيزُ %v والتعاملُ %v — **ومتوسّطُهما يخفي أيَّهما العيب**",
			mine["speed"], mine["conduct"])
	}
	if mine["count"] != float64(2) || mine["low"] != float64(2) {
		t.Fatalf("العدُّ %v والمنخفضُ %v", mine["count"], mine["low"])
	}

	// ── والحدُّ الأدنى يحكمها كما يحكم السائقين ───────────────────────
	//
	// **ومن لم يبلغه لا يتصدّر** — وقُيّم مرّتين، فحدُّ الخمسة يُخفيه.
	d5 := get("5")
	for _, x := range d5["stores"].([]any) {
		if row, _ := x.(map[string]any); row["id"] == f.merchantID {
			t.Fatal("تصدّر متجرٌ لم يبلغ الحدَّ الأدنى — **فيُظنّ أسوأَ من سائقٍ قُيّم مئة**")
		}
	}
}
