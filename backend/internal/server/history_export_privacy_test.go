package server

// **سجلُّ الطلبات في أبواب الخادم — قراراتُ المالك ٢٠٢٦-١٠-٠٤**
//
//	التصديرُ يشمل الخاصَّ · بيوم دمشق · بالعربيّة · والهاتفُ لمن يملكه · بلا حقن   البند ٢
//	هاتفُ الزبون وموقعُه وصورتُه لثلاثةٍ وحدَهم · والباقي مخفيٌّ جزئيّاً           البند ٣

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// callCaps يستدعي باباً بقدراتٍ محسوبةٍ في السياق — كما يضعها وسيطُ التخويل.
func callCaps(h http.HandlerFunc, url, userID string, caps []string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	ctx := context.WithValue(req.Context(), ctxUserID, userID)
	ctx = context.WithValue(ctx, ctxCaps, caps)
	w := httptest.NewRecorder()
	h(w, req.WithContext(ctx))
	return w
}

// historyOrder طلبٌ منتهٍ بوقتٍ بتوقيت دمشق — وبلا متجرٍ إن كان خاصّاً.
func historyOrder(t *testing.T, f *driverFixture, customer, kind string, merchant any, status, damascusAt string) int64 {
	t.Helper()
	var id string
	var number int64
	if err := f.pool.QueryRow(context.Background(), `
		INSERT INTO orders (kind, customer_id, merchant_id, status, address_text, dropoff,
			payment_method, subtotal, delivery_fee, total, wallet_paid, cash_due, created_at, closed_at,
			snap_merchant_commission_percent, snap_rep_commission_percent,
			snap_commission_source, snap_activation_orders)
		VALUES ($1, $2, $3, $4, 'عنوان اختبار',
			ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
			'cash', 1000, 500, 1500, 0, 1500,
			($5::timestamp AT TIME ZONE 'Asia/Damascus'), now(),
			`+qaSnapSQL()+`)
		RETURNING id, number`, kind, customer, merchant, status, damascusAt).Scan(&id, &number); err != nil {
		t.Fatalf("تعذّر إنشاءُ الطلب: %v", err)
	}
	t.Cleanup(func() { _, _ = f.pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, id) })
	return number
}

func TestOrdersExport_CustomDamascusArabicPhoneGate(t *testing.T) {
	f := newDriverFixture(t, 0)
	admin := testdb.NewUser(t, f.pool, "finance")
	cust := testdb.NewUser(t, f.pool, "customer")
	evil := testdb.NewUser(t, f.pool, "customer")
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE users SET full_name = '=HYPERLINK("http://x")' WHERE id = $1`, evil); err != nil {
		t.Fatal(err)
	}
	var phone string
	_ = f.pool.QueryRow(context.Background(), `SELECT phone::text FROM users WHERE id = $1`, cust).Scan(&phone)

	// **الواحدةُ ليلاً بدمشق يومَ ٥** — وهي ما زالت يومَ ٤ في غرينتش.
	custom := historyOrder(t, f, cust, "custom", nil, "delivered", "2026-03-05 01:00:00")
	std := historyOrder(t, f, evil, "standard", f.merchantID, "cancelled", "2026-03-05 12:00:00")
	prevDay := historyOrder(t, f, cust, "standard", f.merchantID, "delivered", "2026-03-04 23:30:00")

	url := "/admin/orders/export?from=2026-03-05&to=2026-03-05&closed=1"
	w := callCaps(f.srv.handleOrdersExport, url, admin, []string{"finance.export"})
	if w.Code != http.StatusOK {
		t.Fatalf("التصدير ردّ %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	lines := map[int64]string{}
	for _, l := range strings.Split(body, "\n") {
		if i := strings.IndexByte(l, ','); i > 0 {
			if n, err := strconv.ParseInt(l[:i], 10, 64); err == nil {
				lines[n] = l
			}
		}
	}
	if lines[custom] == "" {
		t.Fatalf("**الطلبُ الخاصُّ #%d غاب عن الملف** — ضمٌّ صلبٌ بالمتجر يُسقطه:\n%s", custom, body)
	}
	if !strings.Contains(lines[custom], "2026-03-05 01:00") || !strings.Contains(lines[custom], "طلب خاص") {
		t.Errorf("سطرُ الخاصّ بغير يوم دمشق أو بلا نوعه: %s", lines[custom])
	}
	if lines[prevDay] != "" {
		t.Errorf("طلبُ يوم ٤ بدمشق دخل ملفَّ يوم ٥: %s", lines[prevDay])
	}
	if !strings.Contains(lines[custom], "مُسلَّم") || strings.Contains(body, ",delivered,") ||
		!strings.Contains(lines[custom], "نقداً") {
		t.Errorf("القيمُ ليست بالعربيّة: %s", lines[custom])
	}
	if strings.Contains(body, "هاتف الزبون") || strings.Contains(body, phone) {
		t.Errorf("**الهاتفُ خرج لمن لا يملك `users.contact.read`**")
	}
	if !strings.Contains(lines[std], `'=HYPERLINK`) {
		t.Errorf("**معادلةٌ في خليّة** — حقنُ CSV: %s", lines[std])
	}

	w = callCaps(f.srv.handleOrdersExport, url, admin, []string{"finance.export", "users.contact.read"})
	if !strings.Contains(w.Body.String(), "هاتف الزبون") || !strings.Contains(w.Body.String(), phone) {
		t.Errorf("صاحبُ `users.contact.read` لم ينل عمودَ الهاتف")
	}
	if strings.Contains(w.Body.String(), "'"+phone) {
		t.Errorf("الهاتفُ سُبق بفاصلةٍ عليا — **رقمٌ لا معادلة**")
	}

	// **ومدىً مقلوبٌ يُردّ** — لا ملفٌّ فارغٌ صامت.
	w = callCaps(f.srv.handleOrdersExport, "/admin/orders/export?from=2026-03-06&to=2026-03-05",
		admin, []string{"finance.export"})
	if w.Code != http.StatusBadRequest {
		t.Errorf("مدىً مقلوب ردّ %d", w.Code)
	}
}

func TestOrdersList_CustomerDetailsByCapability(t *testing.T) {
	f := newDriverFixture(t, 0)
	reader := testdb.NewUser(t, f.pool, "finance")
	cust := testdb.NewUser(t, f.pool, "customer")
	var phone string
	_ = f.pool.QueryRow(context.Background(), `SELECT phone::text FROM users WHERE id = $1`, cust).Scan(&phone)
	historyOrder(t, f, cust, "standard", f.merchantID, "delivered", "2026-03-05 12:00:00")

	get := func(caps []string) map[string]any {
		t.Helper()
		w := callCaps(f.srv.handleListOrders, "/admin/orders?closed=1&customer_id="+cust, reader, caps)
		if w.Code != http.StatusOK {
			t.Fatalf("القائمة ردّت %d: %s", w.Code, w.Body.String())
		}
		var page struct {
			Data struct {
				Orders []map[string]any `json:"orders"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil || len(page.Data.Orders) != 1 {
			t.Fatalf("قائمةٌ غيرُ متوقّعة: %s", w.Body.String())
		}
		return page.Data.Orders[0]
	}

	// **الماليّةُ ومراقبُ المنصّة** — مخفيٌّ جزئيّاً، بلا إحداثيّاتٍ ولا صورة.
	o := get([]string{"orders.read"})
	for _, k := range []string{"customer_phone", "lat", "lng", "proof_url"} {
		if _, leak := o[k]; leak {
			t.Errorf("**`%s` خرج لمن لا يملك `orders.customer_details.read`**", k)
		}
	}
	masked, _ := o["customer_phone_masked"].(string)
	if masked == "" || strings.Contains(masked, phone[4:]) || !strings.HasSuffix(masked, phone[len(phone)-3:]) ||
		!strings.HasPrefix(masked, "+963 9") {
		t.Errorf("الهاتفُ المخفيّ %q — والمطلوبُ +963 9•• ••• وآخرُ ثلاثة", masked)
	}

	// **والمالكُ والعمليّاتُ والدعم** — كاملاً.
	o = get([]string{"orders.read", "orders.customer_details.read"})
	if o["customer_phone"] != phone || o["lat"] == nil {
		t.Errorf("صاحبُ القدرة لم ينل الهاتفَ والموقع: %v", o)
	}
	if _, ok := o["customer_phone_masked"]; ok {
		t.Errorf("مخفيٌّ لمن يرى الكامل")
	}

	// **والهجرةُ تمنحها للثلاثة وحدَهم.**
	for _, c := range []struct {
		role string
		want bool
	}{
		{"owner_super_admin", true}, {"admin", true}, {"operations", true}, {"customer_support", true},
		{"finance", false}, {"platform_monitor", false}, {"trust_safety", false},
	} {
		var ok bool
		_ = f.pool.QueryRow(context.Background(), `SELECT EXISTS (SELECT 1 FROM role_capabilities
			WHERE role_code = $1 AND capability_code = 'orders.customer_details.read')`, c.role).Scan(&ok)
		if ok != c.want {
			t.Errorf("%s ⇐ orders.customer_details.read = %v — والقرارُ %v", c.role, ok, c.want)
		}
	}
}

func TestMaskPhone(t *testing.T) {
	for in, want := range map[string]string{
		"+963912345123": "+963 9•• ••• 123",
		"":              "",
		"12":            "••",
		"0612345":       "••••345",
	} {
		if got := MaskPhone(in); got != want {
			t.Errorf("MaskPhone(%q) = %q — والمطلوب %q", in, got, want)
		}
	}
}
