package catalog

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// TestCleanItemName **بادئةُ الرقم الزائفة تُشال بشرط أن يتبعها حرف.**
func TestCleanItemName(t *testing.T) {
	cases := map[string]string{
		"017057-طاقة بور هورس 250مل اورجينال": "طاقة بور هورس 250مل اورجينال",
		"0923-مجموعة سوم باي":                 "مجموعة سوم باي",
		"8230-بهارات كاري":                    "بهارات كاري",
		"3ب1 ارمغان 250غ":                     "3ب1 ارمغان 250غ",
		"2-3 قطع":                             "2-3 قطع",
		"  شاي  ":                             "شاي",
		"12345":                               "12345",
	}
	for in, want := range cases {
		if got := CleanItemName(in); got != want {
			t.Errorf("%q ⇒ %q والمطلوب %q", in, got, want)
		}
	}
}

func (f *menuFixture) countItems(t *testing.T) int {
	t.Helper()
	var n int
	if err := f.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM menu_items WHERE merchant_id = $1`, f.merchantID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// TestSyncPrices_CreateHidden **الخيارُ مُطفأٌ فلا إنشاء، ومشتعلٌ فأصنافٌ مخفيّة، وإعادةُ الرفع لا تُنشئ.**
func TestSyncPrices_CreateHidden(t *testing.T) {
	f := newMenuFixture(t)
	ctx := context.Background()
	f.priced(t, "شاي", 30)
	base := f.countItems(t)
	rows := []SyncRow{
		{Code: "017064", Name: "017057-طاقة بور هورس 250مل", Price: f64(1.5)},
		{Code: "017065", Name: "3ب1 ارمغان 250غ", Price: f64(0)},
		{Code: "017066", Name: "بلا سعر"},
		{Code: "", Name: "بلا رقم", Price: f64(2)},
		{Code: "017067", Name: "", Price: f64(2)},
		{Code: "017064", Name: "مكرر", Price: f64(9)},
	}
	opt := SyncOptions{Currency: CurrencySYP, Source: "file", ActorID: f.actorID}

	off, err := f.svc.SyncPrices(ctx, f.merchantID, rows, opt)
	if err != nil {
		t.Fatal(err)
	}
	if off.Created != 0 || f.countItems(t) != base || off.Unmatched != 6 {
		t.Fatalf("الخيارُ مُطفأٌ فلا إنشاء: %+v", off)
	}

	opt.CreateMissing = true
	opt.DryRun = true
	dry, err := f.svc.SyncPrices(ctx, f.merchantID, rows, opt)
	if err != nil {
		t.Fatal(err)
	}
	if dry.Created != 3 || dry.CreatedZeroPrice != 2 || dry.CreateSkipped != 2 || f.countItems(t) != base {
		t.Fatalf("«جرّب» قال %+v وكتب؟ %d", dry, f.countItems(t))
	}

	opt.DryRun = false
	rep, err := f.svc.SyncPrices(ctx, f.merchantID, rows, opt)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Created != 3 || rep.ZeroPrice != 2 || f.countItems(t) != base+3 {
		t.Fatalf("التطبيقُ قال %+v", rep)
	}
	var name, sec string
	var price int64
	var avail, secActive bool
	if err := f.pool.QueryRow(ctx, `
		SELECT i.name, i.merchant_price, i.available, ps.name, ps.active
		  FROM menu_items i JOIN platform_sections ps ON ps.id = i.platform_section_id
		 WHERE i.merchant_id = $1 AND i.external_code = '017064'`, f.merchantID).
		Scan(&name, &price, &avail, &sec, &secActive); err != nil {
		t.Fatal(err)
	}
	// **يذهب إلى قسمه من اسمه** (٢٠٢٦-١١-٠٧) — ويبقى مخفيّاً حتى يُفعَّل.
	if name != "طاقة بور هورس 250مل" || price != 2 || avail || sec != "مشروبات باردة" || !secActive {
		t.Fatalf("الصنفُ المنشأ: %q %d avail=%v %q active=%v", name, price, avail, sec, secActive)
	}

	// idempotent: الرفعُ ثانيةً يطابق بالرقم ولا يُنشئ.
	again, err := f.svc.SyncPrices(ctx, f.merchantID, rows, opt)
	if err != nil {
		t.Fatal(err)
	}
	if again.Created != 0 || f.countItems(t) != base+3 {
		t.Fatalf("إعادةُ الرفع أنشأت: %+v", again)
	}
}

// TestSyncPrices_CreateUSD **الدولارُ: سنتٌ وليرةٌ بسعر الصرف.**
func TestSyncPrices_CreateUSD(t *testing.T) {
	f := newMenuFixture(t)
	ctx := context.Background()
	if _, err := f.pool.Exec(ctx, `UPDATE merchants SET exchange_rate = 130 WHERE id = $1`, f.merchantID); err != nil {
		t.Fatal(err)
	}
	rep, err := f.svc.SyncPrices(ctx, f.merchantID,
		[]SyncRow{{Code: "U1", Name: "مادة", Price: f64(0.6405693950177936)}},
		SyncOptions{Currency: CurrencyUSD, Source: "file", CreateMissing: true})
	if err != nil || rep.Created != 1 {
		t.Fatalf("%+v %v", rep, err)
	}
	var cents *int64
	var price int64
	if err := f.pool.QueryRow(ctx, `SELECT price_usd_cents, merchant_price FROM menu_items
		WHERE merchant_id = $1 AND external_code = 'U1'`, f.merchantID).Scan(&cents, &price); err != nil {
		t.Fatal(err)
	}
	if cents == nil || *cents != 64 || price != USDToSYP(64, 130) {
		t.Fatalf("cents=%v price=%d", cents, price)
	}
}

// TestManualLink **الربطُ اليدويّ: يُحفظ، ويُردّ المكرَّر باسم الصنف الآخر، ويُفصل بالفارغ.**
func TestManualLink(t *testing.T) {
	f := newMenuFixture(t)
	ctx := context.Background()
	a := f.priced(t, "صنف أ", 10)
	b := f.priced(t, "صنف ب", 10)
	code := "017064"
	if err := f.svc.UpdateItem(ctx, f.actorID, a, MenuItemInput{ExternalCode: &code}, ""); err != nil {
		t.Fatal(err)
	}
	err := f.svc.UpdateItem(ctx, f.actorID, b, MenuItemInput{ExternalCode: &code}, "")
	var ae *httpx.AppError
	if !errors.As(err, &ae) || ae.Code != "external_code_taken" || ae.Details["name"] != "صنف أ" {
		t.Fatalf("المنتظَرُ external_code_taken باسم «صنف أ»: %v", err)
	}
	// نفسُ الصنف يعيد حفظ رقمه بلا اعتراض، والفارغُ يفصل.
	if err := f.svc.UpdateItem(ctx, f.actorID, a, MenuItemInput{ExternalCode: &code}, ""); err != nil {
		t.Fatal(err)
	}
	empty := ""
	if err := f.svc.UpdateItem(ctx, f.actorID, a, MenuItemInput{ExternalCode: &empty}, ""); err != nil {
		t.Fatal(err)
	}
	if st := f.state(t, a); st.code != nil {
		t.Fatalf("لم يُفصل: %v", *st.code)
	}
	if err := f.svc.UpdateItem(ctx, f.actorID, b, MenuItemInput{ExternalCode: &code}, ""); err != nil {
		t.Fatalf("بعد الفصل يُربط بغيره: %v", err)
	}
	// الإنشاءُ المكرَّر.
	_, err = f.svc.CreateItem(ctx, f.actorID, f.merchantID, MenuItemInput{
		Name: ptr("ج"), Price: ptr(int64(5)), PlatformSectionID: ptr(f.platformID), ExternalCode: &code,
	}, "")
	if !errors.As(err, &ae) || ae.Code != "external_code_taken" {
		t.Fatalf("الإنشاء: %v", err)
	}
	bad := "ab cd"
	if err := f.svc.UpdateItem(ctx, f.actorID, a, MenuItemInput{ExternalCode: &bad}, ""); err == nil {
		t.Fatal("رقمٌ فيه فراغ قُبل")
	}
}

// TestMallFile **ملفُّ مول الحقيقيّ** (يُتخطّى إن غاب): ٥٣٠٨ صفاً، والأرقام بأصفارها.
func TestMallFile(t *testing.T) {
	const path = "D:/tmp/ameen/mall2.xlsx"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("ملفُّ المول غائب")
	}
	sh, err := ReadSheet("mall2.xlsx", data)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := sh.SyncRows(SyncMapping{Code: "رمز المادة", Name: "اسم المادة", Price: "السعر", Currency: CurrencyUSD})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("headers=%v rows=%d first=%+v", sh.Headers, len(rows), rows[0])
	if len(rows) != 5308 {
		t.Fatalf("الصفوف %d", len(rows))
	}
	seen := map[string]bool{}
	zero := 0
	for _, r := range rows {
		if r.Code == "" || seen[r.Code] {
			t.Fatalf("رقمٌ فارغ أو مكرر: %+v", r)
		}
		seen[r.Code] = true
		if r.Price != nil && *r.Price == 0 {
			zero++
		}
	}
	f := newMenuFixture(t)
	ctx := context.Background()
	if _, err := f.pool.Exec(ctx, `UPDATE merchants SET exchange_rate = 130 WHERE id = $1`, f.merchantID); err != nil {
		t.Fatal(err)
	}
	opt := SyncOptions{Currency: CurrencyUSD, Source: "file", CreateMissing: true}
	opt.DryRun = true
	dry, err := f.svc.SyncPrices(ctx, f.merchantID, rows, opt)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("dry: rows=%d created=%d zero=%d skipped=%d(%d)", dry.Rows, dry.Created, dry.ZeroPrice, dry.CreateSkipped, len(dry.Skipped))
	opt.DryRun = false
	rep, err := f.svc.SyncPrices(ctx, f.merchantID, rows, opt)
	if err != nil {
		t.Fatal(err)
	}
	again, err := f.svc.SyncPrices(ctx, f.merchantID, rows, opt)
	if err != nil {
		t.Fatal(err)
	}
	var lead int
	_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM menu_items WHERE merchant_id=$1 AND external_code LIKE '0%'`, f.merchantID).Scan(&lead)
	t.Logf("real: created=%d zero=%d; again created=%d; codes with leading zero=%d; zero-price rows in file=%d", rep.Created, rep.ZeroPrice, again.Created, lead, zero)
	if rep.Created != dry.Created || again.Created != 0 || lead == 0 {
		t.Fatalf("created=%d dry=%d again=%d lead=%d", rep.Created, dry.Created, again.Created, lead)
	}
}
