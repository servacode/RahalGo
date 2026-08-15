package server

// **ملفُّ صاحب المتجر يرى متجرَه — لا حسابَه وحدَه.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٦: «ابدأ بملفّ صاحب المتجر».)
//
// # خمسةُ أشياءَ كانت خارجَه
//
// **طلباتُ متجره** — والملفُّ يعرض ما اشتراه لنفسه وحدَه، **وهو كلُّ
// عمله.**
//
// **وإنذاراتُ متجره** — `merchant_warnings` جدولٌ آخرُ غيرُ `warnings`.
// **ومتجرٌ يُحظر بعد أربع مخالفات**، فمن راجع صاحبَه ليقرّر **قرأ صفراً**
// وهو على ثلاثٍ من أربع.
//
// **ونزاعاتُ متجره** — تُكتب بـ`merchant_id` لا بشخصٍ قصداً: «المتجرُ
// كيانٌ لا شخص». **فكانت قائمتُه فارغةً دائماً.**
//
// **وتقييمُ السائقين لمتجره** — `merchant_ratings` **يُكتب فيه ولا يُقرأ
// منه أبداً**، لا في إدارةٍ ولا بوّابةٍ ولا تقرير. **والسائقُ يُسأل بعد
// كلّ تسليم** فيُنفَق وقتُه على رأيٍ لا يبلغ أحداً.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func TestMerchantProfile_SeesHisStore(t *testing.T) {
	f := newDriverFixture(t, 1)
	ctx := context.Background()
	customer, _, _ := twoCustomers(t, f)
	driver := f.drivers[0]
	owner := testdb.NewUser(t, f.pool, "merchant")

	// **ومتجرُ العُدّة يصير له** — فيصير كلُّ ما تحته في ملفّه.
	if _, err := f.pool.Exec(ctx,
		`UPDATE merchants SET owner_user_id = $2 WHERE id = $1`, f.merchantID, owner); err != nil {
		t.Fatalf("تعذّر ربطُ المتجر: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(),
			`UPDATE merchants SET owner_user_id = NULL WHERE id = $1`, f.merchantID)
	})

	var orderID string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO orders (customer_id, merchant_id, driver_id, status, address_text,
			dropoff, payment_method, subtotal, delivery_fee, total, cash_due)
		VALUES ($1, $2, $3, 'delivered', 'عنوان اختبار',
			ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
			'cash', 10000, 3000, 13000, 13000)
		RETURNING id::text`, customer, f.merchantID, driver).Scan(&orderID); err != nil {
		t.Fatalf("تعذّر الطلب: %v", err)
	}
	exec := func(what, q string, args ...any) {
		if _, err := f.pool.Exec(ctx, q, args...); err != nil {
			t.Fatalf("تعذّر %s: %v", what, err)
		}
	}
	exec("إنذارَ المتجر", `INSERT INTO merchant_warnings (merchant_id, reason, note)
		VALUES ($1, 'late_prep', 'تأخّرٌ في التجهيز')`, f.merchantID)
	exec("نزاعَ المتجر", `INSERT INTO disputes (party_role, merchant_id, order_id, reason, amount)
		VALUES ('merchant', $1, $2, 'بضاعةٌ ناقصة', 5000)`, f.merchantID, orderID)
	exec("تقييمَ السائق", `INSERT INTO merchant_ratings
		(order_id, driver_id, merchant_id, speed_stars, conduct_stars, comment)
		VALUES ($1, $2, $3, 4, 5, 'تجهيزٌ سريع')`, orderID, driver, f.merchantID)
	t.Cleanup(func() {
		c := context.Background()
		_, _ = f.pool.Exec(c, `DELETE FROM merchant_ratings WHERE order_id = $1`, orderID)
		_, _ = f.pool.Exec(c, `DELETE FROM disputes WHERE merchant_id = $1`, f.merchantID)
		_, _ = f.pool.Exec(c, `DELETE FROM merchant_warnings WHERE merchant_id = $1`, f.merchantID)
		_, _ = f.pool.Exec(c, `DELETE FROM orders WHERE id = $1`, orderID)
	})

	// ── ١ · طلباتُ متجره ──────────────────────────────────────────────
	//
	// **و`owner_id` يجمع متاجرَه كلَّها** — و`merchant_id` يخاطب متجراً
	// بعينه، **والملفُّ يخاطب إنسانا.**
	page, err := f.srv.orders.List(ctx, orders.ListFilter{OwnerID: owner, Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("تعذّرت قراءةُ الطلبات: %v", err)
	}
	// **وطلبُ العُدّة على المتجر نفسِه** — فالمعيارُ حضورُ طلبنا لا عددٌ ثابت.
	seen := false
	for _, o := range page.Orders {
		if o.ID == orderID {
			seen = true
		}
		if o.MerchantID != f.merchantID {
			t.Fatalf("مرَّ طلبُ متجرٍ آخر: %s", o.MerchantID)
		}
	}
	if !seen {
		t.Fatalf("طلبُ متجره لا يظهر (%d صفّاً) — **والملفُّ كان يعرض ما اشتراه لنفسه وحدَه**",
			page.Total)
	}
	// **ومن ليس مالكاً لا يرى شيئاً** — الشرطُ يفصل لا يزيّن.
	empty, err := f.srv.orders.List(ctx, orders.ListFilter{OwnerID: customer, Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	if empty.Total != 0 {
		t.Fatalf("زبونٌ يرى %d من طلبات متجرٍ ليس له", empty.Total)
	}

	// ── ٢ · إنذاراتُ متجره ───────────────────────────────────────────
	var warns struct {
		Data struct {
			Warnings []struct {
				Reason string `json:"reason"`
				Role   string `json:"role_code"`
			} `json:"warnings"`
		} `json:"data"`
	}
	decodeAdmin(t, f, f.srv.handleAdminUserWarnings, owner, &warns)
	found := false
	for _, x := range warns.Data.Warnings {
		if x.Reason == "late_prep" && x.Role == "store" {
			found = true
		}
	}
	if !found {
		t.Fatalf("إنذارُ المتجر لا يظهر في ملفّ صاحبه: %+v — **فيُقرأ صفراً وهو على ثلاثٍ من أربع**",
			warns.Data.Warnings)
	}

	// ── ٣ · نزاعاتُ متجره ────────────────────────────────────────────
	var fin struct {
		Data struct {
			DisputesN int `json:"disputes_count"`
		} `json:"data"`
	}
	decodeAdmin(t, f, f.srv.handleAdminUserFinancials, owner, &fin)
	if fin.Data.DisputesN != 1 {
		t.Fatalf("نزاعاتُ متجره %d — **وتُكتب بـmerchant_id لا بشخص، فكانت قائمتُه فارغةً دائماً**",
			fin.Data.DisputesN)
	}

	// ── ٤ · وتقييمُ السائقين — يُقرأ لأوّل مرّة ──────────────────────
	var fb struct {
		Data struct {
			ByDriversN int      `json:"by_drivers_count"`
			AvgSpeed   *float64 `json:"avg_speed"`
			AvgConduct *float64 `json:"avg_conduct"`
			ByDrivers  []struct {
				Driver  string `json:"driver"`
				Speed   int    `json:"speed_stars"`
				Conduct int    `json:"conduct_stars"`
			} `json:"by_drivers"`
		} `json:"data"`
	}
	decodeAdmin(t, f, f.srv.handleAdminUserFeedback, owner, &fb)
	if fb.Data.ByDriversN != 1 || len(fb.Data.ByDrivers) != 1 {
		t.Fatalf("تقييمُ السائقين %d — **وجدولٌ يُكتب فيه ولا يُقرأ منه أبداً**",
			fb.Data.ByDriversN)
	}
	if fb.Data.AvgSpeed == nil || *fb.Data.AvgSpeed != 4 ||
		fb.Data.AvgConduct == nil || *fb.Data.AvgConduct != 5 {
		t.Fatalf("المتوسّطان %v و%v", fb.Data.AvgSpeed, fb.Data.AvgConduct)
	}
	// **وقائلُه باسمه** — **وتقييمٌ بلا قائلٍ لا يُراجَع.**
	if fb.Data.ByDrivers[0].Driver == "" {
		t.Fatal("تقييمٌ بلا اسمِ سائق — **فلا يُعرف أسائقٌ واحدٌ كرّرها أم عشرة**")
	}
}

// decodeAdmin **نداءُ معالِجٍ إداريٍّ بمعرّفٍ في المسار.**
func decodeAdmin(t *testing.T, f *driverFixture,
	h func(http.ResponseWriter, *http.Request), id string, into any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", id)
	c := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	c = context.WithValue(c, ctxUserID, id)
	c = context.WithValue(c, ctxRoles, []string{"admin"})
	w := httptest.NewRecorder()
	h(w, req.WithContext(c))
	if w.Code != 200 {
		t.Fatalf("ردَّ %d — %s", w.Code, w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), into); err != nil {
		t.Fatalf("ردٌّ لا يُفكّ: %v — %s", err, w.Body.String())
	}
}
