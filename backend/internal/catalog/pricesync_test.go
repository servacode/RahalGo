package catalog

import (
	"bytes"
	"context"
	"testing"

	"github.com/xuri/excelize/v2"
	"golang.org/x/text/encoding/charmap"
)

func f64(v float64) *float64 { return &v }

func (f *menuFixture) priced(t *testing.T, name string, price int64) string {
	t.Helper()
	id, err := f.svc.CreateItem(context.Background(), f.actorID, f.merchantID, MenuItemInput{
		SectionID: ptr(f.sectionID), Name: ptr(name), Price: ptr(price), PlatformSectionID: ptr(f.platformID),
	}, "127.0.0.1")
	if err != nil {
		t.Fatalf("إنشاءُ الصنف: %v", err)
	}
	return id
}

type itemState struct {
	price     int64
	cents     *int64
	available bool
	code      *string
	barcode   *string
}

func (f *menuFixture) state(t *testing.T, id string) itemState {
	t.Helper()
	var st itemState
	if err := f.pool.QueryRow(context.Background(), `
		SELECT merchant_price, price_usd_cents, available, external_code, barcode
		  FROM menu_items WHERE id = $1`, id).Scan(&st.price, &st.cents, &st.available, &st.code, &st.barcode); err != nil {
		t.Fatal(err)
	}
	return st
}

// TestSyncPrices_MatchAndUpdate **قائمةُ الأمين تطابق وتحدّث** (قرارُ المالك ٢٠٢٦-١٠-١٩) —
// برقم المادّة ثمّ الباركود ثمّ الاسم، **والكميّةُ تحكم التوفّر، و«جرّب» لا يكتب.**
func TestSyncPrices_MatchAndUpdate(t *testing.T) {
	f := newMenuFixture(t)
	ctx := context.Background()
	rice := f.priced(t, "رز مصري ١ كغ", 100)
	sugar := f.priced(t, "سكر  أبيض", 50)
	oil := f.priced(t, "زيت دوار الشمس", 200)
	tea := f.priced(t, "شاي", 30)
	if _, err := f.pool.Exec(ctx, `UPDATE menu_items SET external_code = 'A-1' WHERE id = $1`, rice); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE menu_items SET barcode = '6281000' WHERE id = $1`, oil); err != nil {
		t.Fatal(err)
	}
	rows := []SyncRow{
		{Code: "A-1", Name: "أيّ اسم", Price: f64(120), Qty: f64(5)},         // برقم المادّة
		{Code: "B-7", Name: "سكر ابيض", Price: f64(55), Qty: f64(0)},         // بالاسم ⇒ يُربط ويصير غير متوفر
		{Code: "C-3", Barcode: "6281000", Price: f64(200), Qty: f64(9)},      // بالباركود، سعرٌ كما هو
		{Code: "Z-9", Name: "مادّة ليست عندنا", Price: f64(10), Qty: f64(1)}, // لا مقابل
		{Code: "T-1", Name: "شاي", Price: f64(-3)},                           // سعرٌ فاسد ⇒ يُترك
	}

	dry, err := f.svc.SyncPrices(ctx, f.merchantID, rows, SyncOptions{Currency: CurrencySYP, DryRun: true, Source: "file", ActorID: f.actorID})
	if err != nil {
		t.Fatal(err)
	}
	if dry.Matched != 4 || dry.PriceChanged != 2 || dry.MadeUnavailable != 1 || dry.Unmatched != 1 || dry.LinkedByName != 1 || len(dry.Skipped) != 1 {
		t.Fatalf("«جرّب» قال %+v", dry)
	}
	if f.state(t, rice).price != 100 {
		t.Fatal("**«جرّب» كتب في القاعدة**")
	}

	rep, err := f.svc.SyncPrices(ctx, f.merchantID, rows, SyncOptions{Currency: CurrencySYP, Source: "file", ActorID: f.actorID})
	if err != nil {
		t.Fatal(err)
	}
	if rep.PriceChanged != 2 || rep.MadeUnavailable != 1 {
		t.Fatalf("التطبيقُ قال %+v", rep)
	}
	if st := f.state(t, rice); st.price != 120 || !st.available {
		t.Fatalf("الرز: %+v — **والمطلوب ١٢٠ متوفر**", st)
	}
	if st := f.state(t, sugar); st.price != 55 || st.available || st.code == nil || *st.code != "B-7" {
		t.Fatalf("السكر: %+v — **والمطلوب ٥٥ غير متوفر ومربوطٌ بـ B-7**", st)
	}
	if st := f.state(t, oil); st.price != 200 || st.code == nil || *st.code != "C-3" {
		t.Fatalf("الزيت: %+v — **والمطلوب ٢٠٠ ومربوطٌ بـ C-3**", st)
	}
	if st := f.state(t, tea); st.price != 30 {
		t.Fatalf("الشاي تبدّل بسعرٍ فاسد: %+v", st)
	}

	// **والمرّةُ الثانية بالرقم** — ولو تبدّل الاسمُ في الأمين، والكميّةُ تعيده متوفراً.
	rep2, err := f.svc.SyncPrices(ctx, f.merchantID, []SyncRow{{Code: "B-7", Name: "اسمٌ آخر", Qty: f64(3)}},
		SyncOptions{Currency: CurrencySYP, Source: "agent"})
	if err != nil {
		t.Fatal(err)
	}
	if rep2.Matched != 1 || rep2.MadeAvailable != 1 || rep2.LinkedByName != 0 {
		t.Fatalf("المرّةُ الثانية: %+v", rep2)
	}
	if st := f.state(t, sugar); !st.available || st.price != 55 {
		t.Fatalf("السكر بعد الكميّة: %+v", st)
	}

	// **والدولارُ بسعر صرف المتجر.**
	if _, err := f.pool.Exec(ctx, `UPDATE merchants SET exchange_rate = 130 WHERE id = $1`, f.merchantID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.SyncPrices(ctx, f.merchantID, []SyncRow{{Code: "A-1", Price: f64(1.5)}},
		SyncOptions{Currency: CurrencyUSD, Source: "file"}); err != nil {
		t.Fatal(err)
	}
	if st := f.state(t, rice); st.price != 195 || st.cents == nil || *st.cents != 150 {
		t.Fatalf("الرز بالدولار: %+v — **والمطلوب ١٫٥$ = ١٩٥ ل.س**", st)
	}
	var runs int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM price_sync_runs WHERE merchant_id = $1`, f.merchantID).Scan(&runs); err != nil || runs != 3 {
		t.Fatalf("سجلُّ المزامنة %d — **والمطلوب ٣ (لا يُسجَّل «جرّب»)**", runs)
	}
}

// TestReadSheet_ExcelAndArabicCSV **ملفُّ الأمين يُقرأ** — xlsx، وCSV بترميز ويندوز العربيّ،
// **والأرقامُ كما يكتبها محاسبٌ عربيّ.**
func TestReadSheet_ExcelAndArabicCSV(t *testing.T) {
	x := excelize.NewFile()
	_ = x.SetSheetRow("Sheet1", "A1", &[]any{"تقرير المواد"})
	_ = x.SetSheetRow("Sheet1", "A3", &[]any{"رقم المادة", "اسم المادة", "سعر البيع", "الكمية"})
	_ = x.SetSheetRow("Sheet1", "A4", &[]any{"A-1", "رز", "1,250", "٣"})
	_ = x.SetSheetRow("Sheet1", "A5", &[]any{"A-2", "سكر", "٧٥٠٫٥ ل.س", "0"})
	var buf bytes.Buffer
	if err := x.Write(&buf); err != nil {
		t.Fatal(err)
	}
	sh, err := ReadSheet("مواد.xlsx", buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(sh.Headers) != 4 || sh.Headers[2] != "سعر البيع" || len(sh.Rows) != 2 {
		t.Fatalf("الرأس %v الصفوف %d — **والعنوانُ فوق الرأس لا يُعدّ رأساً**", sh.Headers, len(sh.Rows))
	}
	rows, err := sh.SyncRows(SyncMapping{Code: "رقم المادة", Name: "اسم المادة", Price: "سعر البيع", Qty: "الكمية"})
	if err != nil {
		t.Fatal(err)
	}
	if *rows[0].Price != 1250 || *rows[0].Qty != 3 || *rows[1].Price != 750.5 || *rows[1].Qty != 0 {
		t.Fatalf("الأرقام: %+v %+v", *rows[0].Price, *rows[1].Price)
	}
	if _, err := sh.SyncRows(SyncMapping{Code: "عمودٌ غائب", Price: "سعر البيع"}); err != ErrSyncMapping {
		t.Fatalf("عمودٌ غائبٌ قُبل: %v", err)
	}

	csv := "رقم المادة;السعر;الكمية\r\nA-1;100;2\r\n"
	enc, err := charmap.Windows1256.NewEncoder().String(csv)
	if err != nil {
		t.Fatal(err)
	}
	sh2, err := ReadSheet("list.csv", []byte(enc))
	if err != nil {
		t.Fatal(err)
	}
	if sh2.Headers[0] != "رقم المادة" || len(sh2.Rows) != 1 {
		t.Fatalf("CSV العربيّ: %v", sh2.Headers)
	}
	if _, err := ReadSheet("old.xls", []byte("x")); err != ErrSyncFileType {
		t.Fatalf("‎.xls قُبل: %v", err)
	}
}
