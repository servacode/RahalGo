package waitems

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/media"
	"github.com/servacode/rahalgo/backend/internal/notify"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

type fx struct {
	t        *testing.T
	pool     *pgxpool.Pool
	svc      *Service
	ctx      context.Context
	owner    string // معرّف المستخدم
	phone    string // رقم المالك
	merchant string
	catID    string
	dairy    string // معرّف قسم «ألبان وأجبان»
	mu       sync.Mutex
	sent     []string
	seq      int
}

func png1() []byte {
	im := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for x := 0; x < 64; x++ {
		for y := 0; y < 64; y++ {
			im.Set(x, y, color.RGBA{uint8(x * 4), uint8(y * 4), 90, 255})
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, im)
	return b.Bytes()
}

func newFx(t *testing.T) *fx {
	t.Helper()
	pool := testdb.Pool(t)
	f := &fx{t: t, pool: pool, ctx: context.Background()}
	md, err := media.NewService(pool, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f.svc = New(pool, catalog.NewService(pool, nil), md, nil)
	f.svc.Quiet = time.Hour // الإرسال بـFlushAll فقط
	f.svc.Send = func(_ context.Context, phone, text string) error {
		f.mu.Lock()
		f.sent = append(f.sent, text)
		f.mu.Unlock()
		return nil
	}
	f.svc.SiteURL = func() string { return "https://site.test" }

	f.owner = testdb.NewUser(t, pool, "merchant")
	must(t, pool.QueryRow(f.ctx, `SELECT phone FROM users WHERE id = $1`, f.owner).Scan(&f.phone))
	must(t, pool.QueryRow(f.ctx, `INSERT INTO categories (name, icon, sort_order, has_drinks)
		VALUES ('فئة واتساب', '', 901, false) RETURNING id`).Scan(&f.catID))
	f.merchant = f.newStore(f.owner, "متجر الاختبار", "SYP", 0)
	err = pool.QueryRow(f.ctx, `SELECT id::text FROM platform_sections WHERE name = 'ألبان وأجبان' AND active LIMIT 1`).Scan(&f.dairy)
	if err != nil {
		must(t, pool.QueryRow(f.ctx, `INSERT INTO platform_sections (name, sort_order) VALUES ('ألبان وأجبان', 22) RETURNING id::text`).Scan(&f.dairy))
	}
	// المتجر الرئيسيّ أعلن قسماً واحداً ⇒ يُستعمل بلا سؤال (الأسئلة لها اختباراتها أدناه).
	f.declare(f.merchant, f.dairy)
	return f
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func (f *fx) newStore(owner, name, cur string, rate float64) string {
	var id string
	var r *float64
	if rate > 0 {
		r = &rate
	}
	must(f.t, f.pool.QueryRow(f.ctx, `
		INSERT INTO merchants (name, category_id, owner_user_id, display_currency, exchange_rate)
		VALUES ($1, $2, $3, $4, $5) RETURNING id::text`, name, f.catID, owner, cur, r).Scan(&id))
	return id
}

func (f *fx) img(phone, caption string) string {
	f.seq++
	return f.svc.HandleImage(f.ctx, phone, notify.InboundImage{
		Caption: caption, Mime: "image/png", Data: png1(), MsgID: fmt.Sprintf("%s-m%d", f.owner, f.seq)})
}

func (f *fx) imgID(phone, caption, id string) string {
	return f.svc.HandleImage(f.ctx, phone, notify.InboundImage{Caption: caption, Mime: "image/png", Data: png1(), MsgID: f.owner + "-" + id})
}

func (f *fx) flush() []string {
	f.svc.FlushAll()
	f.mu.Lock()
	defer f.mu.Unlock()
	out := f.sent
	f.sent = nil
	return out
}

func (f *fx) count(merchant string) int {
	var n int
	must(f.t, f.pool.QueryRow(f.ctx, `SELECT count(*) FROM menu_items WHERE merchant_id = $1::uuid`, merchant).Scan(&n))
	return n
}

func one(t *testing.T, got []string) string {
	t.Helper()
	if len(got) != 1 {
		t.Fatalf("المتوقع رسالة واحدة، وصل %d: %q", len(got), got)
	}
	return got[0]
}

func has(t *testing.T, s string, subs ...string) {
	t.Helper()
	for _, x := range subs {
		if !strings.Contains(s, x) {
			t.Fatalf("الردّ بلا %q:\n%s", x, s)
		}
	}
}

func TestStrangerIgnored(t *testing.T) {
	f := newFx(t)
	if r := f.img("+963900000999", "جبنة بيضاء 25000"); r != "" {
		t.Fatalf("رُدّ على غريب: %q", r)
	}
	if got := f.flush(); len(got) != 0 {
		t.Fatalf("أُرسل لغريب: %q", got)
	}
	if r, ok := f.svc.HandleText(f.ctx, "+963900000999", "فرع", ""); ok || r != "" {
		t.Fatalf("غريب عولج نصُّه: %q %v", r, ok)
	}
	if f.count(f.merchant) != 0 {
		t.Fatal("أُضيف صنف لغريب")
	}
}

func TestOwnerCreatesItem(t *testing.T) {
	f := newFx(t)
	if r := f.img(f.phone, "جبنة بيضاء 25000"); r != "" {
		t.Fatalf("ردّ فوريّ غير متوقع: %q", r)
	}
	msg := one(t, f.flush())
	has(t, msg, "انضاف (جبنة بيضاء)", "٢٥٬٠٠٠ ل.س", "ألبان وأجبان", "✓", "https://site.test/store/menu/")

	var id, via, src, sec string
	var avail bool
	var price int64
	var media *string
	must(t, f.pool.QueryRow(f.ctx, `
		SELECT id::text, created_via, section_source, platform_section_id::text, available, merchant_price, image_media_id::text
		  FROM menu_items WHERE merchant_id = $1::uuid`, f.merchant).Scan(&id, &via, &src, &sec, &avail, &price, &media))
	if via != "whatsapp" || src != "manual" || sec != f.dairy || !avail || price != 25000 || media == nil {
		t.Fatalf("صفّ خاطئ: via=%s src=%s sec=%s avail=%v price=%d media=%v", via, src, sec, avail, price, media)
	}
	if !strings.HasSuffix(msg, id) {
		t.Fatalf("الرابط ليس رابط الصنف: %s", msg)
	}
	var actor, details string
	must(t, f.pool.QueryRow(f.ctx, `SELECT actor_user_id::text, details::text FROM audit_log
		WHERE action = 'menu.item_create' AND entity_id = $1`, id).Scan(&actor, &details))
	if actor != f.owner || !strings.Contains(details, "whatsapp") {
		t.Fatalf("تدقيق خاطئ: %s %s", actor, details)
	}
}

func TestRedeliveredMessageIgnored(t *testing.T) {
	f := newFx(t)
	f.imgID(f.phone, "لبنة 8000", "same-id")
	f.imgID(f.phone, "لبنة 8000", "same-id")
	f.flush()
	if n := f.count(f.merchant); n != 1 {
		t.Fatalf("التسليم المعاد كرّر الصنف: %d", n)
	}
}

func TestStaffCreatesItem(t *testing.T) {
	f := newFx(t)
	staff := "+963944123456"
	must(t, func() error {
		_, err := f.pool.Exec(f.ctx, `INSERT INTO merchant_staff (owner_user_id, phone, name, password_hash)
			VALUES ($1, $2, 'علي', 'x')`, f.owner, staff)
		return err
	}())
	f.img(staff, "لبنة 8000")
	msg := one(t, f.flush())
	has(t, msg, "انضاف (لبنة)")
	var id, actor, details string
	must(t, f.pool.QueryRow(f.ctx, `SELECT id::text FROM menu_items WHERE merchant_id = $1::uuid`, f.merchant).Scan(&id))
	must(t, f.pool.QueryRow(f.ctx, `SELECT actor_user_id::text, details::text FROM audit_log
		WHERE action = 'menu.item_create' AND entity_id = $1`, id).Scan(&actor, &details))
	if actor != f.owner || !strings.Contains(details, "staff_id") || !strings.Contains(details, "علي") {
		t.Fatalf("الموظف غير مسجّل بالتدقيق: %s %s", actor, details)
	}
}

func TestMissingPriceThenReply(t *testing.T) {
	f := newFx(t)
	f.img(f.phone, "جبنة بيضاء")
	msg := one(t, f.flush())
	if msg != "شو سعر (جبنة بيضاء)؟" {
		t.Fatalf("سؤال السعر: %q", msg)
	}
	if f.count(f.merchant) != 0 {
		t.Fatal("أُضيف صنف بلا سعر")
	}
	// ردّ غير مفهوم: يعيد السؤال ويبقى معلّقاً
	r, ok := f.svc.HandleText(f.ctx, f.phone, "مش عارف", "")
	if !ok || !strings.Contains(r, "ما فهمت السعر") {
		t.Fatalf("ردّ غامض: %q %v", r, ok)
	}
	r, ok = f.svc.HandleText(f.ctx, f.phone, "٢٥ ألف", "")
	if !ok {
		t.Fatal("الجواب لم يُعالَج")
	}
	has(t, r, "انضاف (جبنة بيضاء)", "٢٥٬٠٠٠ ل.س")
	if f.count(f.merchant) != 1 {
		t.Fatal("لم ينضف الصنف بعد السعر")
	}
	// لا سؤال معلّق الآن ⇒ النص يمرّ للمعالِج العام
	if _, ok := f.svc.HandleText(f.ctx, f.phone, "مرحبا", ""); ok {
		t.Fatal("نصّ عادي عولج بلا سؤال")
	}
}

func TestMissingName(t *testing.T) {
	f := newFx(t)
	f.img(f.phone, "25000")
	if msg := one(t, f.flush()); msg != "شو اسم الصنف؟" {
		t.Fatalf("سؤال الاسم: %q", msg)
	}
	r, _ := f.svc.HandleText(f.ctx, f.phone, "جبنة بيضاء", "")
	has(t, r, "انضاف (جبنة بيضاء)", "٢٥٬٠٠٠")
}

func TestMultiStoreAsksOnceAndRemembers(t *testing.T) {
	f := newFx(t)
	second := f.newStore(f.owner, "الفرع الثاني", "SYP", 0)
	f.declare(second, f.dairy)
	f.img(f.phone, "جبنة بيضاء 25000")
	q := one(t, f.flush())
	has(t, q, "لأي فرع؟", "١ - متجر الاختبار", "٢ - الفرع الثاني")
	if f.count(f.merchant)+f.count(second) != 0 {
		t.Fatal("أُضيف قبل اختيار الفرع")
	}
	// جواب غير مفهوم
	r, _ := f.svc.HandleText(f.ctx, f.phone, "٧", "")
	has(t, r, "ما فهمت", "لأي فرع؟")
	r, ok := f.svc.HandleText(f.ctx, f.phone, "٢", "")
	if !ok {
		t.Fatal("لم يعالج الاختيار")
	}
	has(t, r, "انضاف (جبنة بيضاء)")
	if f.count(second) != 1 || f.count(f.merchant) != 0 {
		t.Fatalf("الصنف بالفرع الخطأ: %d/%d", f.count(f.merchant), f.count(second))
	}
	// بعدها بلا سؤال
	f.img(f.phone, "لبنة 8000")
	has(t, one(t, f.flush()), "انضاف (لبنة)")
	if f.count(second) != 2 {
		t.Fatal("لم يتذكر الفرع")
	}
	// «فرع» يعيد السؤال
	r, ok = f.svc.HandleText(f.ctx, f.phone, "فرع", "")
	if !ok {
		t.Fatal("«فرع» لم يعالَج")
	}
	has(t, r, "لأي فرع؟")
	r, _ = f.svc.HandleText(f.ctx, f.phone, "1", "")
	has(t, r, "تمام", "متجر الاختبار")
	f.img(f.phone, "حليب 5000")
	f.flush()
	if f.count(f.merchant) != 1 {
		t.Fatal("لم يتبدّل الفرع")
	}
}

func TestMultiStoreBatchWaits(t *testing.T) {
	f := newFx(t)
	second := f.newStore(f.owner, "الفرع الثاني", "SYP", 0)
	f.declare(second, f.dairy)
	f.img(f.phone, "جبنة بيضاء 25000")
	f.img(f.phone, "لبنة 8000")
	f.img(f.phone, "حليب 5000")
	has(t, one(t, f.flush()), "لأي فرع؟")
	r, _ := f.svc.HandleText(f.ctx, f.phone, "2", "")
	has(t, r, "انضاف ٣ أصناف")
	if f.count(second) != 3 {
		t.Fatalf("الصور المنتظرة لم تُضف كلها: %d", f.count(second))
	}
}

func TestDuplicateName(t *testing.T) {
	f := newFx(t)
	f.img(f.phone, "جبنة بيضاء 25000")
	f.flush()
	f.img(f.phone, "جبنة  بيضاء 26000")
	q := one(t, f.flush())
	has(t, q, "في صنف بنفس الاسم", "١ حدّث صورته وسعره", "٢ أضفه صنف جديد")
	if f.count(f.merchant) != 1 {
		t.Fatal("أُضيف مكرّر قبل الجواب")
	}
	r, _ := f.svc.HandleText(f.ctx, f.phone, "٢", "")
	has(t, r, "انضاف (جبنة بيضاء)")
	if f.count(f.merchant) != 2 {
		t.Fatal("خيار ٢ لم يضف صنفاً جديداً")
	}
	// خيار ١: تحديث الموجود
	f.img(f.phone, "جبنة بيضاء 30000")
	f.flush()
	r, _ = f.svc.HandleText(f.ctx, f.phone, "1", "")
	has(t, r, "حدّثت صورة وسعر (جبنة بيضاء)", "٣٠٬٠٠٠")
	if f.count(f.merchant) != 2 {
		t.Fatal("التحديث أضاف صنفاً")
	}
	var n int
	must(t, f.pool.QueryRow(f.ctx, `SELECT count(*) FROM menu_items WHERE merchant_id = $1::uuid AND merchant_price = 30000`, f.merchant).Scan(&n))
	if n != 1 {
		t.Fatalf("السعر لم يتحدّث: %d", n)
	}
	must(t, f.pool.QueryRow(f.ctx, `SELECT count(*) FROM audit_log WHERE action = 'menu.item_wa_update' AND actor_user_id = $1`, f.owner).Scan(&n))
	if n != 1 {
		t.Fatal("تدقيق التحديث ناقص")
	}
}

func TestSanityConfirm(t *testing.T) {
	f := newFx(t)
	for i := 0; i < 3; i++ {
		_, err := f.pool.Exec(f.ctx, `INSERT INTO menu_items (merchant_id, name, merchant_price, price, platform_section_id)
			VALUES ($1::uuid, $2, 1000, 1000, $3::uuid)`, f.merchant, "صنف قديم "+string(rune('أ'+i)), f.dairy)
		must(t, err)
	}
	f.img(f.phone, "جبنة بيضاء 50000")
	q := one(t, f.flush())
	has(t, q, "متأكد؟", "٥٠٬٠٠٠ ل.س", "نعم")
	if f.count(f.merchant) != 3 {
		t.Fatal("حُفظ قبل التأكيد")
	}
	r, _ := f.svc.HandleText(f.ctx, f.phone, "نعم", "")
	has(t, r, "انضاف (جبنة بيضاء)")
	if f.count(f.merchant) != 4 {
		t.Fatal("لم يُحفظ بعد التأكيد")
	}
	// «لا» ⇒ يسأل السعر من جديد
	f.img(f.phone, "لبنة 70000")
	f.flush()
	r, _ = f.svc.HandleText(f.ctx, f.phone, "لا", "")
	if r != "شو سعر (لبنة)؟" {
		t.Fatalf("بعد لا: %q", r)
	}
	r, _ = f.svc.HandleText(f.ctx, f.phone, "1500", "")
	has(t, r, "انضاف (لبنة)", "١٬٥٠٠")
}

func TestRateLimit(t *testing.T) {
	f := newFx(t)
	f.svc.RatePerMin = 2
	f.img(f.phone, "جبنة بيضاء 25000")
	f.img(f.phone, "لبنة 8000")
	f.img(f.phone, "حليب 5000")
	got := f.flush()
	msg := one(t, got)
	has(t, msg, "انضاف", "ما انضافوا (سرعة)")
	if f.count(f.merchant) != 2 {
		t.Fatalf("الحدّ لم يُطبَّق: %d", f.count(f.merchant))
	}
}

func TestBatchSummary(t *testing.T) {
	f := newFx(t)
	names := []string{"جبنة بيضاء", "لبنة", "حليب", "زبدة"}
	for i, n := range names {
		f.img(f.phone, n+" "+string(rune('1'+i))+"000")
	}
	f.img(f.phone, "كشك")
	f.img(f.phone, "قشطة")
	msg := one(t, f.flush())
	has(t, msg, "انضاف ٤ أصناف، و٢ ناقصهن سعر", "ردّ على صورة")
	// وجوابٌ بالاقتباس يطابق صورته
	r, _ := f.svc.HandleText(f.ctx, f.phone, "3000", "")
	has(t, r, "أكتر من صنف")
}

func TestQuotedReplyMatchesPhoto(t *testing.T) {
	f := newFx(t)
	f.imgID(f.phone, "كشك", "photo-A")
	f.imgID(f.phone, "قشطة", "photo-B")
	f.flush()
	r, _ := f.svc.HandleText(f.ctx, f.phone, "٣ آلاف", f.owner+"-photo-B")
	has(t, r, "انضاف (قشطة)", "٣٬٠٠٠")
	r, _ = f.svc.HandleText(f.ctx, f.phone, "4000", "")
	has(t, r, "انضاف (كشك)", "٤٬٠٠٠") // بقي سؤال واحد فيجيبه الرد بلا اقتباس
}

func TestNotImage(t *testing.T) {
	f := newFx(t)
	r := f.svc.HandleImage(f.ctx, f.phone, notify.InboundImage{NotImage: true, MsgID: f.owner + "-vid-1"})
	has(t, r, "ابعت صورة")
	if r := f.svc.HandleImage(f.ctx, "+963900000998", notify.InboundImage{NotImage: true, MsgID: "vid-2"}); r != "" {
		t.Fatalf("رُدّ على غريب: %q", r)
	}
}

func TestUSDStore(t *testing.T) {
	f := newFx(t)
	usd := f.newStore(f.owner, "متجر دولار", "USD", 15000)
	// فرع واحد فقط للمالك الثاني
	owner2 := testdb.NewUser(t, f.pool, "merchant")
	var phone2 string
	must(t, f.pool.QueryRow(f.ctx, `SELECT phone FROM users WHERE id = $1`, owner2).Scan(&phone2))
	_ = usd
	st := f.newStore(owner2, "متجر دولار ٢", "USD", 15000)
	f.declare(st, f.dairy)
	f.img(phone2, "لبنة 5")
	has(t, one(t, f.flush()), "انضاف (لبنة)", "٥ $")
	var cents, syp int64
	must(t, f.pool.QueryRow(f.ctx, `SELECT price_usd_cents, merchant_price FROM menu_items WHERE merchant_id = $1::uuid`, st).Scan(&cents, &syp))
	if cents != 500 || syp != 75000 {
		t.Fatalf("الدولار: %d سنت و%d ل.س", cents, syp)
	}
}
