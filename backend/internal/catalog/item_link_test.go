package catalog

// **رقمُ المادّة بالأمين والباركود — ثلاثةُ عيوبٍ من مراجعة ما قبل الإصدار** (٢٠٢٦-١١-٠٥)
//
//  1. **الأرقامُ الهنديّةُ تُحفظ لاتينيّة** — الكيبوردُ العربيّ يكتب «٠١٧٠٦٤» وملفُّ الأمين
//     «017064»، فكان الصنفُ المربوطُ بيده لا يُطابَق أبداً. **ورقمٌ يتشاركه صنفان بعد
//     التحويل يذهب للأقدم** — صنفِ المتجر لا نسخته المخفيّة.
//  2. **رقمٌ قديمٌ لم يُلمس لا يمنع الحفظ** — التطبيقُ يرسل الحقلين مع كلّ حفظ.
//  3. **والمزامنةُ لا تكتب رقماً لا يقبله النموذج** — بقواعد `cleanLink` نفسِها، **وتطبّق
//     سعرَ الصفّ وكميّتَه** فلا يتوقّف تحديثُ الصنف بسبب باركود.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

func linkOf(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func appCode(err error) string {
	var ae *httpx.AppError
	if errors.As(err, &ae) {
		return ae.Code
	}
	return ""
}

// TestLinkText **الرقمُ كما يُقارَن ويُحفظ** — مشذَّباً وبأرقامٍ لاتينيّة، وما سواها كما هو.
func TestLinkText(t *testing.T) {
	cases := map[string]string{
		" ٠١٧٠٦٤ ":   "017064",
		"۶۲۸۱۰۰۰۱۲۳": "6281000123",
		"A-١٢":       "A-12",
		"017064":     "017064",
		"":           "",
	}
	for in, want := range cases {
		if got := linkText(in); got != want {
			t.Errorf("%q ⇒ %q والمطلوب %q", in, got, want)
		}
	}
	for v, ok := range map[string]bool{"": true, "017064": true, "AB 12": false, strings.Repeat("9", maxLinkLen): true, strings.Repeat("9", maxLinkLen+1): false} {
		if validLink(v) != ok {
			t.Errorf("validLink(%q) = %v والمطلوب %v", v, !ok, ok)
		}
	}
}

// TestItemLink_WesternDigits **رقمٌ بأرقامٍ هنديّةٍ أو فارسيّةٍ يُحفظ لاتينيّاً ويطابق ملفَّ الأمين.**
//
// **وكان يُحفظ كما كُتب**: المزامنةُ لا تجده بالرقم، و«أنشئ غير الموجود» تصنع له صنفاً
// مخفيّاً ثانياً، **و«017064» و«٠١٧٠٦٤» يُربطان بصنفين** بلا اعتراض.
func TestItemLink_WesternDigits(t *testing.T) {
	f := newMenuFixture(t)
	ctx := context.Background()
	syp := SyncOptions{Currency: CurrencySYP, Source: "file"}
	a := f.priced(t, "صنف أ", 10)
	b := f.priced(t, "صنف ب", 10)

	code, barcode := "٠١٧٠٦٤", "۶۲۸۱۰۰۰۱۲۳"
	if err := f.svc.UpdateItem(ctx, f.actorID, a, MenuItemInput{ExternalCode: &code, Barcode: &barcode}, ""); err != nil {
		t.Fatal(err)
	}
	if st := f.state(t, a); linkOf(st.code) != "017064" || linkOf(st.barcode) != "6281000123" {
		t.Errorf("حُفظ %q / %q — والمطلوب 017064 / 6281000123", linkOf(st.code), linkOf(st.barcode))
	}
	western := "017064"
	if err := f.svc.UpdateItem(ctx, f.actorID, b, MenuItemInput{ExternalCode: &western}, ""); appCode(err) != "external_code_taken" {
		t.Errorf("الرقمُ نفسُه بأرقامٍ لاتينيّةٍ رُبط بصنفٍ آخر: %v", err)
	}

	// **وملفُّ الأمين يطابقه بالرقم وبالباركود** — بأيّ شكلٍ كُتبت أرقامُه، ولا يُنشئ له نسخة.
	base := f.countItems(t)
	for i, r := range []SyncRow{
		{Code: "017064", Name: "اسمٌ آخر في الأمين", Price: f64(120)},
		{Barcode: "6281000123", Price: f64(130)},
		{Code: "٠١٧٠٦٤", Name: "مادة", Price: f64(140)},
	} {
		opt := syp
		opt.CreateMissing = true
		rep, err := f.svc.SyncPrices(ctx, f.merchantID, []SyncRow{r}, opt)
		if err != nil {
			t.Fatal(err)
		}
		if got := f.state(t, a).price; rep.Matched != 1 || rep.LinkedByName != 0 || rep.Created != 0 || got != int64(*r.Price) {
			t.Errorf("الصفّ %d: %+v والسعر %d — والمطلوب مطابقةٌ بالرقم وسعرٌ %v", i+1, rep, got, *r.Price)
		}
	}
	if n := f.countItems(t); n != base {
		t.Errorf("أُنشئت أصنافٌ مكرّرة: %d والمطلوب %d", n, base)
	}

	// **وما حُفظ هنديّاً قبل الإصلاح** يطابقه الملفّ كذلك، **ولا يُربط رقمُه بصنفٍ آخر.**
	old := f.priced(t, "صنف قديم", 10)
	if _, err := f.pool.Exec(ctx, `UPDATE menu_items SET external_code = '٠١٧٠٧٠', barcode = '٦٢٨٩' WHERE id = $1`, old); err != nil {
		t.Fatal(err)
	}
	for i, r := range []SyncRow{{Code: "017070", Price: f64(55)}, {Barcode: "6289", Price: f64(56)}} {
		rep, err := f.svc.SyncPrices(ctx, f.merchantID, []SyncRow{r}, syp)
		if err != nil {
			t.Fatal(err)
		}
		if got := f.state(t, old).price; rep.Matched != 1 || got != int64(*r.Price) {
			t.Errorf("الصنفُ القديم، الصفّ %d: %+v والسعر %d", i+1, rep, got)
		}
	}
	w := "017070"
	if err := f.svc.UpdateItem(ctx, f.actorID, b, MenuItemInput{ExternalCode: &w}, ""); appCode(err) != "external_code_taken" {
		t.Errorf("رقمٌ محفوظٌ بأرقامٍ هنديّةٍ رُبط بصنفٍ آخر: %v", err)
	}
}

// TestUpdateItem_UnchangedLinkSaves **رقمٌ قديمٌ لا يقبله النموذج لا يمنع حفظَ الصنف ما لم يُلمس.**
//
// التطبيقُ يرسل رقمَ الأمين والباركود مع كلّ حفظ (`MenuViewModel.saveItem`) — **ورقمٌ
// كتبته المزامنةُ بفراغٍ في وسطه كان يردّ كلَّ حفظٍ** بـ`external_code_invalid`، ولو غيّر
// صاحبُه السعرَ وحدَه.
func TestUpdateItem_UnchangedLinkSaves(t *testing.T) {
	f := newMenuFixture(t)
	ctx := context.Background()
	a := f.priced(t, "صنف من المزامنة", 10)
	const oldCode, oldBarcode = "AB 12", "6281000123456 6281000123457"
	if _, err := f.pool.Exec(ctx, `UPDATE menu_items SET external_code = $2, barcode = $3 WHERE id = $1`, a, oldCode, oldBarcode); err != nil {
		t.Fatal(err)
	}
	code, barcode := oldCode, oldBarcode
	if err := f.svc.UpdateItem(ctx, f.actorID, a, MenuItemInput{Price: ptr(int64(99)), ExternalCode: &code, Barcode: &barcode}, ""); err != nil {
		t.Fatalf("حفظُ صنفٍ لم يُلمس رقمُه رُفض: %v", err)
	}
	if st := f.state(t, a); st.price != 99 || linkOf(st.code) != oldCode || linkOf(st.barcode) != oldBarcode {
		t.Fatalf("بعد الحفظ: %d %q %q", st.price, linkOf(st.code), linkOf(st.barcode))
	}
	// **ورقمٌ جديدٌ لا يصلح يُردّ كما كان** — والباركودُ القديمُ معه لا يُفحص.
	bad := "AB 13"
	if err := f.svc.UpdateItem(ctx, f.actorID, a, MenuItemInput{ExternalCode: &bad, Barcode: &barcode}, ""); appCode(err) != "external_code_invalid" {
		t.Fatalf("رقمٌ جديدٌ فيه فراغ قُبل: %v", err)
	}
	good := "AB12"
	if err := f.svc.UpdateItem(ctx, f.actorID, a, MenuItemInput{ExternalCode: &good, Barcode: &barcode}, ""); err != nil {
		t.Fatalf("تصحيحُ الرقم رُفض: %v", err)
	}
	if st := f.state(t, a); linkOf(st.code) != "AB12" || linkOf(st.barcode) != oldBarcode {
		t.Fatalf("بعد التصحيح: %q %q", linkOf(st.code), linkOf(st.barcode))
	}
}

// TestSyncPrices_NeverStoresInvalidLink **المزامنةُ لا تكتب رقماً أو باركوداً لا يقبله النموذج — وتطبّق صفَّه.**
//
// **وكانت تشذّب الطرفين وحدَهما**: عمودُ الاسم يُختار رقماً، أو باركودان في خانة، فيُكتب
// ما فيه فراغ — **صنفٌ لا يُحفظ تعديلُه بعدها.** فلا يُكتب ويُذكر (`LinkRejected`)،
// **والسعرُ والكميّةُ يُطبَّقان**: تركُ الصفّ كلِّه كان يُبقي صنفاً خلص بالأمين «متوفراً»
// بسعرٍ قديمٍ في كلّ مزامنة، ولا يراه أحد. **ولا يُنشأ صنفٌ برقمٍ لا يصلح** (`bad_code`).
func TestSyncPrices_NeverStoresInvalidLink(t *testing.T) {
	f := newMenuFixture(t)
	ctx := context.Background()
	oil := f.priced(t, "زيت", 100)
	sugar := f.priced(t, "سكر", 50)
	tea := f.priced(t, "شاي", 30)
	if _, err := f.pool.Exec(ctx, `UPDATE menu_items SET external_code = 'T-1' WHERE id = $1`, tea); err != nil {
		t.Fatal(err)
	}
	long := strings.Repeat("9", maxLinkLen+1)
	rows := []SyncRow{
		{Code: "عصير برتقال 1 لتر", Name: "زيت", Price: f64(110), Qty: f64(0)},             // عمودُ الاسم اختير رقماً، وخلص
		{Code: "S-1", Barcode: "6281000123456 6281000123457", Name: "سكر", Price: f64(55)}, // باركودان بخانةٍ واحدة
		{Code: "T-1", Barcode: long, Price: f64(35)},                                       // باركودٌ أطولُ من الحدّ
		{Code: long, Name: "مادة جديدة", Price: f64(5)},                                    // لا يُنشأ صنفٌ برقمٍ طويل
		{Code: "N 1", Name: "مادة ثانية", Price: f64(5)},                                   // ولا برقمٍ فيه فراغ
		{Code: "N-2", Name: "مادة سليمة", Price: f64(5)},                                   // وهذه تُنشأ
	}
	base := f.countItems(t)
	for _, dry := range []bool{true, false} {
		rep, err := f.svc.SyncPrices(ctx, f.merchantID, rows,
			SyncOptions{Currency: CurrencySYP, Source: "file", ActorID: f.actorID, CreateMissing: true, DryRun: dry})
		if err != nil {
			t.Fatal(err)
		}
		skipped, rejected := map[string]int{}, map[string]int{}
		for _, s := range rep.Skipped {
			skipped[s.Reason]++
		}
		for _, s := range rep.LinkRejectedSample {
			rejected[s.Reason]++
		}
		if rep.Matched != 3 || rep.PriceChanged != 3 || rep.MadeUnavailable != 1 || rep.Created != 1 ||
			len(rep.Skipped) != 2 || skipped["bad_code"] != 2 ||
			rep.LinkRejected != 3 || rejected["bad_code"] != 1 || rejected["bad_barcode"] != 2 ||
			rep.LinkedByName != 1 {
			t.Errorf("dry=%v: %+v — والمطلوب 3 مطابقة بأسعارها، وصنفٌ جديدٌ واحد، و2 «bad_code» متروكان من الإنشاء وحدَه، "+
				"و3 روابط لم تُحفظ (1 رقم و2 باركود)، و«رُبطت بالاسم» 1 (السكر) لا الزيت الذي لم يُحفظ له شيء", dry, rep)
		}
	}
	var bad int
	if err := f.pool.QueryRow(ctx, `
		SELECT count(*) FROM menu_items WHERE merchant_id = $1
		   AND (external_code ~ '\s' OR barcode ~ '\s'
		        OR char_length(external_code) > 64 OR char_length(barcode) > 64)`, f.merchantID).Scan(&bad); err != nil {
		t.Fatal(err)
	}
	if bad != 0 {
		t.Errorf("كُتب %d رقماً أو باركوداً لا يقبله النموذج", bad)
	}
	if n := f.countItems(t); n != base+1 {
		t.Errorf("الأصناف %d والمطلوب %d", n, base+1)
	}
	// **والصفوفُ الثلاثةُ طُبّقت** — والرابطُ الذي يصلح منها حُفظ، والذي لا يصلح لم يُكتب.
	for _, c := range []struct {
		name, id, code string
		price          int64
		available      bool
	}{{"الزيت", oil, "", 110, false}, {"السكر", sugar, "S-1", 55, true}, {"الشاي", tea, "T-1", 35, true}} {
		st := f.state(t, c.id)
		if st.price != c.price || st.available != c.available || linkOf(st.code) != c.code || st.barcode != nil {
			t.Errorf("%s: السعر %d متوفر=%v الرقم %q الباركود %q — والمطلوب %d متوفر=%v الرقم %q بلا باركود",
				c.name, st.price, st.available, linkOf(st.code), linkOf(st.barcode), c.price, c.available, c.code)
		}
	}
}

// TestSyncPrices_SharedCodeGoesToOldest **رقمٌ أو باركودٌ يتشاركه صنفان بعد التحويل يذهب للأقدم — دائماً.**
//
// صنفُ المتجر حُفظ رقمُه «٠١٧٠٦٤» بيده، **و«أنشئ غير الموجود» صنعت قبل التحويل نسخةً مخفيّةً
// برقم «017064»** — رقمٌ واحدٌ بعد `linkText`. **وكان الاستعلامُ بلا ترتيبٍ والأخيرُ يربح**،
// فتتحدّث النسخةُ الأحدثُ لا صنفُ المتجر. **وبالترتيبين في الجدول** — فلا يمرّ الفحصُ لأنّ
// الصفوفَ وقعت بترتيبٍ موافق.
func TestSyncPrices_SharedCodeGoesToOldest(t *testing.T) {
	f := newMenuFixture(t)
	ctx := context.Background()
	link := func(id, code, barcode string, ageDays int) {
		t.Helper()
		if _, err := f.pool.Exec(ctx, `
			UPDATE menu_items SET external_code = $2, barcode = NULLIF($3, ''),
			       created_at = now() - make_interval(days => $4::int)
			 WHERE id = $1`, id, code, barcode, ageDays); err != nil {
			t.Fatal(err)
		}
	}
	// **صنفُ المتجر أوّلاً في الجدول ونسختُه بعده** — وهو ما كان يخسر.
	mine := f.priced(t, "صنف المتجر", 10)
	link(mine, "٠١٧٠٦٤", "٦٢٨١٠٠٠", 30)
	twin := f.priced(t, "نسخة مخفية", 10)
	link(twin, "017064", "6281000", 1)
	// **والعكسُ**: النسخةُ أوّلاً وصنفُ المتجر بعدها.
	twin2 := f.priced(t, "نسخة مخفية ثانية", 10)
	link(twin2, "017065", "", 1)
	mine2 := f.priced(t, "صنف المتجر الثاني", 10)
	link(mine2, "٠١٧٠٦٥", "", 30)

	for i, c := range []struct {
		row         SyncRow
		owner, twin string
	}{
		{SyncRow{Code: "017064", Price: f64(77)}, mine, twin},
		{SyncRow{Code: "٠١٧٠٦٤", Price: f64(78)}, mine, twin},
		{SyncRow{Barcode: "6281000", Price: f64(79)}, mine, twin},
		{SyncRow{Code: "017065", Price: f64(80)}, mine2, twin2},
	} {
		rep, err := f.svc.SyncPrices(ctx, f.merchantID, []SyncRow{c.row}, SyncOptions{Currency: CurrencySYP, Source: "file"})
		if err != nil {
			t.Fatal(err)
		}
		got, other := f.state(t, c.owner).price, f.state(t, c.twin).price
		if rep.Matched != 1 || got != int64(*c.row.Price) || other != 10 {
			t.Errorf("الصفّ %d: %+v — الأقدم %d والنسخة %d، والمطلوب %v للأقدم و10 للنسخة", i+1, rep, got, other, *c.row.Price)
		}
	}
}
