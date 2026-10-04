package orders_test

// **سجلُّ الطلبات — قراراتُ المالك ٢٠٢٦-١٠-٠٤**
//
//	البحثُ يفهم الموظّف (09… · 00963… · ١٤١٦ · #1416 · اسم · % و_)   البند ٥
//	المتجرُ أيُّ مصدرٍ شارك في الطلب                                   البند ٧
//	أعدادُ الحالات تتبع البحثَ والفلاتر                                 المشكلة ٨
//	مدىً مقلوبٌ يُردّ · ونوعُ الطلب يُرشِّح                             المشكلتان ١٣ و٢٠

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

func TestNormalizeSearch(t *testing.T) {
	for _, c := range []struct {
		in    string
		want  orders.SearchTerms
		label string
	}{
		{"1416", orders.SearchTerms{Number: 1416, Phone: "1416"}, "رقمُ طلب"},
		{"١٤١٦", orders.SearchTerms{Number: 1416, Phone: "1416"}, "أرقامٌ هنديّة"},
		{"#1416", orders.SearchTerms{Number: 1416, Phone: "1416"}, "علامةُ #"},
		{" # 14 16 ", orders.SearchTerms{Number: 1416, Phone: "1416"}, "مسافات"},
		{"0912345678", orders.SearchTerms{Phone: "963912345678"}, "09…"},
		{"00963912345678", orders.SearchTerms{Phone: "963912345678"}, "00963…"},
		{"+963 912 345 678", orders.SearchTerms{Phone: "963912345678"}, "+963 بمسافات"},
		{"٠٩١٢٣٤٥٦٧٨", orders.SearchTerms{Phone: "963912345678"}, "هاتفٌ بأرقامٍ هنديّة"},
		{"أحمد", orders.SearchTerms{Name: "أحمد"}, "اسم"},
		{"%", orders.SearchTerms{Name: `\%`}, "% حرفٌ لا أداة"},
		{"a_b", orders.SearchTerms{Name: `a\_b`}, "_ حرفٌ لا أداة"},
		{"", orders.SearchTerms{}, "فارغ"},
	} {
		if got := orders.NormalizeSearch(c.in); got != c.want {
			t.Errorf("%s: %q ⇒ %+v — والمطلوب %+v", c.label, c.in, got, c.want)
		}
	}
}

func TestHistoryFilters_SearchSourcesCountsKind(t *testing.T) {
	pool := testdb.Pool(t)
	store := settings.NewStore(pool)
	svc := orders.NewService(pool, nil, wallet.NewService(pool),
		cashbox.NewService(pool, store), nil,
		slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
	svc.SetSettings(store)
	ctx := context.Background()
	customer := testdb.NewUser(t, pool, "customer")
	var phone string
	if _, err := pool.Exec(ctx, `UPDATE users SET full_name = 'زبونُ السجلّ الفريد' WHERE id = $1`, customer); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT phone::text FROM users WHERE id = $1`, customer).Scan(&phone); err != nil {
		t.Fatal(err)
	}

	var categoryID string
	if err := pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatalf("لا تصنيفات في القاعدة: %v", err)
	}
	mkMerchant := func(name string) string {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO merchants (name, category_id, commission_percent)
			VALUES ($1, $2, 10) RETURNING id::text`, name, categoryID).Scan(&id); err != nil {
			t.Fatalf("تعذّر إنشاءُ متجر: %v", err)
		}
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, id)
		})
		return id
	}
	mA := mkMerchant("متجرُ السجلّ أ")
	mB := mkMerchant("متجرُ السجلّ ب")

	mk := func(kind string, merchant any, status string) (string, int64) {
		var id string
		var number int64
		if err := pool.QueryRow(ctx, `
			INSERT INTO orders (kind, customer_id, merchant_id, status, address_text, dropoff,
				payment_method, subtotal, delivery_fee, total, cash_due, closed_at,
				snap_merchant_commission_percent, snap_rep_commission_percent,
				snap_commission_source, snap_activation_orders)
			VALUES ($1, $2, $3, $4, 'عنوان اختبار',
				ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
				'cash', 0, 0, 0, 0, now(),
			`+qaSnapSQLX()+`)
			RETURNING id::text, number`, kind, customer, merchant, status).Scan(&id, &number); err != nil {
			t.Fatalf("تعذّر إنشاءُ الطلب: %v", err)
		}
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, id)
		})
		return id, number
	}
	aID, aNum := mk("standard", mA, "delivered")
	bID, _ := mk("standard", mB, "cancelled")
	cID, _ := mk("custom", nil, "delivered")
	// **طلبُ «أ» فيه صنفٌ من «ب»** — متعدّدُ المصادر (البند ٧).
	if _, err := pool.Exec(ctx, `
		INSERT INTO order_items (order_id, merchant_id, name, unit_price, merchant_price, qty, options)
		VALUES ($1, $2, 'صنفٌ من ب', 100, 90, 1, '[]')`, aID, mB); err != nil {
		t.Fatalf("صنفُ المصدر الثاني: %v", err)
	}

	list := func(f orders.ListFilter) (map[string]bool, *orders.OrderPage) {
		t.Helper()
		f.CustomerID = customer
		f.PerPage = 50
		p, err := svc.List(ctx, f)
		if err != nil {
			t.Fatalf("تعذّرت القراءة (%+v): %v", f, err)
		}
		out := map[string]bool{}
		for _, o := range p.Orders {
			out[o.ID] = true
		}
		return out, p
	}

	// ── البحث (البند ٥) ─────────────────────────────────────────────
	local := "0" + phone[4:] // +9639xxxxxxxx ⇒ 09xxxxxxxx
	intl := "00" + phone[1:] // ⇒ 00963…
	for _, q := range []string{phone, local, intl, "زبونُ السجلّ"} {
		got, _ := list(orders.ListFilter{Query: q})
		if !got[aID] || !got[bID] || !got[cID] {
			t.Errorf("البحثُ %q لم يجد طلباتِ الزبون: %v", q, got)
		}
	}
	arabic := ""
	for _, r := range strconv.FormatInt(aNum, 10) {
		arabic += string('٠' + (r - '0'))
	}
	for _, q := range []string{arabic, "#" + strconv.FormatInt(aNum, 10)} {
		got, _ := list(orders.ListFilter{Query: q})
		if !got[aID] {
			t.Errorf("البحثُ %q لم يجد الطلب #%d", q, aNum)
		}
	}
	for _, q := range []string{"%", "_"} {
		if got, _ := list(orders.ListFilter{Query: q}); len(got) != 0 {
			t.Errorf("**%q طابق %d طلباً** — والمطلوبُ حرفٌ لا أداة", q, len(got))
		}
	}

	// ── المتجرُ أيُّ مصدر (البند ٧) ─────────────────────────────────
	got, _ := list(orders.ListFilter{MerchantID: mB, AnySource: true})
	if !got[aID] || !got[bID] || got[cID] {
		t.Errorf("متجرُ ب بأيّ مصدر: %v — **وطلبُ أ فيه صنفٌ من ب**", got)
	}
	got, _ = list(orders.ListFilter{MerchantID: mB})
	if got[aID] || !got[bID] {
		t.Errorf("متجرُ ب صاحباً: %v — **وبوّابةُ المتجر ما باعه هو**", got)
	}

	// ── الأعدادُ تتبع البحثَ والفلاتر (المشكلة ٨) ───────────────────
	_, p := list(orders.ListFilter{Query: "#" + strconv.FormatInt(aNum, 10), WithCounts: true})
	if p.Counts == nil || p.Counts["delivered"] != 1 || p.Counts["cancelled"] != 0 {
		t.Errorf("أعدادُ البحث: %v — **والمطلوبُ مسلَّمٌ واحدٌ كالقائمة**", p.Counts)
	}
	_, p = list(orders.ListFilter{ClosedOnly: true, Status: "cancelled", MerchantID: mA, AnySource: true})
	if p.Counts["delivered"] != 1 || p.Total != 0 {
		t.Errorf("أعدادُ المتجر: %v وعددُ القائمة %d — **والحالُ لا يدخل العدّ**", p.Counts, p.Total)
	}

	// ── نوعُ الطلب (المشكلة ٢٠) ─────────────────────────────────────
	got, _ = list(orders.ListFilter{Kind: "custom"})
	if len(got) != 1 || !got[cID] {
		t.Errorf("الطلباتُ الخاصّة: %v", got)
	}
	_, err := svc.List(ctx, orders.ListFilter{CustomerID: customer, Kind: "x"})
	var appErr *httpx.AppError
	if !errors.As(err, &appErr) || appErr.Status != http.StatusBadRequest {
		t.Errorf("نوعٌ لا يُعرف: %v", err)
	}

	// ── مدىً مقلوب (المشكلة ١٣) ─────────────────────────────────────
	_, err = svc.List(ctx, orders.ListFilter{CustomerID: customer, From: "2026-02-01", To: "2026-01-01"})
	if !errors.As(err, &appErr) || appErr.Status != http.StatusBadRequest {
		t.Errorf("مدىً مقلوب ردّ %v — **وسجلٌّ فارغٌ صامتٌ يُقرأ «لا طلبات»**", err)
	}
}
