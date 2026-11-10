package waitems

import (
	"fmt"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// ══════════════════════════════════════════════════════════════════════
// اختيار القسم بالسؤال (قرار المالك 2026-11-09): البوت يعرض أقسام المتجر مرقّمةً والمرسل يردّ برقم.
// ══════════════════════════════════════════════════════════════════════

// declare يُعلن أقسام متجر (store_sections).
func (f *fx) declare(store string, sections ...string) {
	f.t.Helper()
	for _, s := range sections {
		_, err := f.pool.Exec(f.ctx, `INSERT INTO store_sections (store_id, section_id) VALUES ($1::uuid, $2::uuid)
			ON CONFLICT DO NOTHING`, store, s)
		must(f.t, err)
	}
}

// newSection قسمٌ جديد بترتيب مرتفع (يأتي بعد أقسام السوق).
func (f *fx) newSection(name string, order int, active bool) string {
	var id string
	must(f.t, f.pool.QueryRow(f.ctx, `INSERT INTO platform_sections (name, sort_order, active)
		VALUES ($1, $2, $3) RETURNING id::text`, name, order, active).Scan(&id))
	return id
}

// freshStore متجرٌ بمالكٍ جديد (رقمه، معرّفه) لاختبار إعلان أقسامٍ مستقلّ.
func (f *fx) freshStore(secs ...string) (phone, store string) {
	f.t.Helper()
	o := f.newOwner()
	store = f.newStore(o.id, "متجر أقسام", "SYP", 0)
	f.declare(store, secs...)
	return o.phone, store
}

type owner struct{ id, phone string }

func (f *fx) newOwner() owner {
	f.t.Helper()
	id := testdb.NewUser(f.t, f.pool, "merchant")
	var ph string
	must(f.t, f.pool.QueryRow(f.ctx, `SELECT phone FROM users WHERE id = $1`, id).Scan(&ph))
	return owner{id, ph}
}

func (f *fx) sectionOf(store string) (name, src string) {
	f.t.Helper()
	must(f.t, f.pool.QueryRow(f.ctx, `SELECT ps.name, m.section_source FROM menu_items m
		JOIN platform_sections ps ON ps.id = m.platform_section_id WHERE m.merchant_id = $1::uuid`, store).Scan(&name, &src))
	return
}

func TestSectionIsAskedAndNumberCreatesItem(t *testing.T) {
	f := newFx(t)
	a := f.newSection("قسم اختبار أ", 9001, true)
	b := f.newSection("قسم اختبار ب", 9002, true)
	c := f.newSection("قسم اختبار ج", 9003, true)
	phone, store := f.freshStore(a, b, c)

	f.img(phone, "جبنة بيضاء 25000")
	q := one(t, f.flush())
	want := "بأي قسم أحط (جبنة بيضاء)؟\n١ - قسم اختبار أ\n٢ - قسم اختبار ب\n٣ - قسم اختبار ج\nردّ برقم القسم"
	if q != want {
		t.Fatalf("سؤال القسم:\n%s\nالمتوقع:\n%s", q, want)
	}
	if f.count(store) != 0 {
		t.Fatal("أُضيف الصنف قبل اختيار القسم")
	}
	// رقم خارج القائمة وكلام غير مفهوم ⇒ إعادة قصيرة والصنف معلّق
	for _, bad := range []string{"٩", "٠", "مش عارف"} {
		r, ok := f.svc.HandleText(f.ctx, phone, bad, "")
		if !ok || r != "ما فهمت 🙏 ردّ برقم القسم من ١ لـ ٣" {
			t.Fatalf("ردّ على %q: %q %v", bad, r, ok)
		}
	}
	if f.count(store) != 0 {
		t.Fatal("أُضيف الصنف بجواب غلط")
	}
	r, ok := f.svc.HandleText(f.ctx, phone, "٢", "")
	if !ok {
		t.Fatal("لم يعالَج الجواب")
	}
	has(t, r, "انضاف (جبنة بيضاء)", "٢٥٬٠٠٠ ل.س", "بقسم قسم اختبار ب", "✓")
	if n, src := f.sectionOf(store); n != "قسم اختبار ب" || src != "manual" {
		t.Fatalf("القسم %q المصدر %q", n, src)
	}
	// لا سؤال معلّق بعدها
	if _, ok := f.svc.HandleText(f.ctx, phone, "٢", ""); ok {
		t.Fatal("بقي سؤال معلّق")
	}
}

func TestSectionByNameAndAsciiDigit(t *testing.T) {
	f := newFx(t)
	a := f.newSection("خضار اختبار", 9011, true)
	b := f.newSection("فواكه اختبار", 9012, true)
	phone, store := f.freshStore(a, b)
	f.img(phone, "تفاح 5000")
	f.flush()
	r, _ := f.svc.HandleText(f.ctx, phone, "فواكه", "")
	has(t, r, "انضاف (تفاح)", "بقسم فواكه اختبار")
	f.img(phone, "خيار 2000")
	f.flush()
	r, _ = f.svc.HandleText(f.ctx, phone, "1", "")
	has(t, r, "انضاف (خيار)", "بقسم خضار اختبار")
	if f.count(store) != 2 {
		t.Fatal("العدد")
	}
}

func TestSectionManyIsCappedButNameStillWorks(t *testing.T) {
	f := newFx(t)
	var ids []string
	for i := 1; i <= 30; i++ {
		ids = append(ids, f.newSection(fmt.Sprintf("صنف رقم%02d اختبار", i), 9100+i, true))
	}
	phone, store := f.freshStore(ids...)
	f.img(phone, "عصير 3000")
	q := one(t, f.flush())
	has(t, q, "٢٥ - صنف رقم25 اختبار", "اكتب اسم القسم إذا مو بالقائمة")
	if strings.Contains(q, "صنف رقم26") {
		t.Fatalf("القائمة لم تُقصّ:\n%s", q)
	}
	r, _ := f.svc.HandleText(f.ctx, phone, "٢٦", "")
	has(t, r, "ما فهمت", "٢٥")
	r, _ = f.svc.HandleText(f.ctx, phone, "رقم28", "")
	has(t, r, "انضاف (عصير)", "صنف رقم28 اختبار")
	if f.count(store) != 1 {
		t.Fatal("لم يُضف")
	}
}

func TestSectionAmbiguousNameReasks(t *testing.T) {
	f := newFx(t)
	a := f.newSection("حلويات شرقية اختبار", 9021, true)
	b := f.newSection("حلويات غربية اختبار", 9022, true)
	phone, _ := f.freshStore(a, b)
	f.img(phone, "كنافة 9000")
	f.flush()
	r, _ := f.svc.HandleText(f.ctx, phone, "حلويات", "")
	has(t, r, "ما فهمت")
}

func TestSectionSingleOptionNoQuestion(t *testing.T) {
	f := newFx(t)
	a := f.newSection("قسم وحيد اختبار", 9031, true)
	phone, store := f.freshStore(a)
	f.img(phone, "حليب 5000")
	has(t, one(t, f.flush()), "انضاف (حليب)", "بقسم قسم وحيد اختبار")
	if f.count(store) != 1 {
		t.Fatal("لم يُضف")
	}
}

func TestSectionNoActiveSectionsFallsBackToPending(t *testing.T) {
	f := newFx(t)
	dead := f.newSection("قسم موقوف اختبار", 9041, false)
	phone, store := f.freshStore(dead) // أعلن قسماً غير فعّال فقط ⇒ لا خيارات
	f.img(phone, "قطعة غريبة جدا اسمها كذا 1200")
	msg := one(t, f.flush())
	has(t, msg, "انضاف", "بانتظار التصنيف")
	if _, src := f.sectionOf(store); src != "pending" {
		t.Fatalf("المصدر %s", src)
	}
}

func TestSectionStaffAnswersForStore(t *testing.T) {
	f := newFx(t)
	a := f.newSection("قسم موظف أ", 9051, true)
	b := f.newSection("قسم موظف ب", 9052, true)
	o := f.newOwner()
	store := f.newStore(o.id, "متجر موظف", "SYP", 0)
	f.declare(store, a, b)
	staff := "+963944765432"
	_, err := f.pool.Exec(f.ctx, `INSERT INTO merchant_staff (owner_user_id, phone, name, password_hash)
		VALUES ($1, $2, 'سامر', 'x')`, o.id, staff)
	must(t, err)
	f.img(staff, "لبنة 8000")
	has(t, one(t, f.flush()), "بأي قسم أحط (لبنة)؟", "١ - قسم موظف أ")
	// المالك نفسه لا يرى سؤال الموظف: السؤال مرتبط برقم المرسل
	if _, ok := f.svc.HandleText(f.ctx, o.phone, "١", ""); ok {
		t.Fatal("سؤال الموظف عولج برقم المالك")
	}
	r, _ := f.svc.HandleText(f.ctx, staff, "2", "")
	has(t, r, "انضاف (لبنة)", "قسم موظف ب")
	var actor, details string
	must(t, f.pool.QueryRow(f.ctx, `SELECT a.actor_user_id::text, a.details::text FROM audit_log a
		JOIN menu_items m ON m.id::text = a.entity_id WHERE a.action = 'menu.item_create' AND m.merchant_id = $1::uuid`, store).Scan(&actor, &details))
	if actor != o.id || !strings.Contains(details, "سامر") {
		t.Fatalf("تدقيق الموظف: %s %s", actor, details)
	}
}

func TestSectionAbandonedExpiresAndCancels(t *testing.T) {
	f := newFx(t)
	a := f.newSection("قسم مهلة أ", 9061, true)
	b := f.newSection("قسم مهلة ب", 9062, true)
	phone, store := f.freshStore(a, b)
	f.img(phone, "تمر 4000")
	f.flush()
	// انتهت المهلة ⇒ الجواب يمرّ للمعالج العام ولا يُضاف شيء
	_, err := f.pool.Exec(f.ctx, `UPDATE wa_item_pending SET expires_at = now() - interval '1 minute' WHERE phone = $1`, phone)
	must(t, err)
	if _, ok := f.svc.HandleText(f.ctx, phone, "1", ""); ok {
		t.Fatal("عولج جواب بعد انتهاء المهلة")
	}
	if f.count(store) != 0 {
		t.Fatal("أُضيف بعد المهلة")
	}
	// «إلغاء» يمسح سؤال القسم
	f.img(phone, "رطب 4000")
	f.flush()
	r, _ := f.svc.HandleText(f.ctx, phone, "الغاء", "")
	has(t, r, "لغيت")
	if _, ok := f.svc.HandleText(f.ctx, phone, "1", ""); ok || f.count(store) != 0 {
		t.Fatal("بقي السؤال بعد الإلغاء")
	}
}

func TestSectionBatchSummaryShowsListOnce(t *testing.T) {
	f := newFx(t)
	a := f.newSection("قسم دفعة أ", 9071, true)
	b := f.newSection("قسم دفعة ب", 9072, true)
	phone, store := f.freshStore(a, b)
	f.imgID(phone, "تمر 4000", "ph-A")
	f.imgID(phone, "رطب 5000", "ph-B")
	msg := one(t, f.flush())
	has(t, msg, "٢ ناقصهن قسم", "ردّ على صورة", "الأقسام:", "١ - قسم دفعة أ", "٢ - قسم دفعة ب")
	if strings.Count(msg, "١ - قسم دفعة أ") != 1 {
		t.Fatalf("القائمة تكرّرت:\n%s", msg)
	}
	r, _ := f.svc.HandleText(f.ctx, phone, "2", f.owner+"-ph-B")
	has(t, r, "انضاف (رطب)", "قسم دفعة ب")
	r, _ = f.svc.HandleText(f.ctx, phone, "1", "")
	has(t, r, "انضاف (تمر)", "قسم دفعة أ")
	if f.count(store) != 2 {
		t.Fatal("العدد")
	}
}

func TestSectionDuplicateUpdateSkipsQuestion(t *testing.T) {
	f := newFx(t)
	a := f.newSection("قسم مكرر أ", 9081, true)
	b := f.newSection("قسم مكرر ب", 9082, true)
	phone, store := f.freshStore(a, b)
	f.img(phone, "جبنة بيضاء 25000")
	f.flush()
	f.svc.HandleText(f.ctx, phone, "1", "")
	f.img(phone, "جبنة بيضاء 30000")
	f.flush()
	r, _ := f.svc.HandleText(f.ctx, phone, "1", "") // ١ = حدّث الموجود
	has(t, r, "حدّثت صورة وسعر (جبنة بيضاء)", "٣٠٬٠٠٠")
	if f.count(store) != 1 {
		t.Fatal("التحديث أضاف صنفاً")
	}
}

func TestPickSection(t *testing.T) {
	names := []string{"خضار", "فواكه", "حلويات شرقية", "حلويات غربية"}
	cases := []struct {
		in string
		i  int
		ok bool
	}{
		{"1", 0, true}, {"٢", 1, true}, {"رقم 4", 3, true}, {"5", 0, false}, {"0", 0, false},
		{"فواكه", 1, true}, {"خضار ", 0, true}, {"شرقية", 2, true}, {"حلويات", 0, false}, {"", 0, false}, {"xyz", 0, false},
	}
	for _, c := range cases {
		i, ok := pickSection(ToASCIIDigits(c.in), names)
		if ok != c.ok || (ok && i != c.i) {
			t.Errorf("%q → %d %v، المتوقع %d %v", c.in, i, ok, c.i, c.ok)
		}
	}
}
