// اختباراتُ المصنع — **الأداةُ تُثبَت لا تُوصَف.**
//
// **وتنقسم قسمين**: **ما لا يحتاج قاعدةً فيُشغَّل دائماً** — الحتميّةُ
// والنطاقاتُ وحارسُ الإنتاج — **وما يحتاجها فيُتخطّى بصدقٍ إن غابت.**
package qa

import (
	"github.com/servacode/rahalgo/backend/internal/identity"
	"os"
	"strings"
	"testing"
	"time"
)

// ══════════════════════════════════════════════════════════════════════
// **١ · الحتميّة — ولا وقتٌ ولا عشوائيّة**
// ══════════════════════════════════════════════════════════════════════

func TestFactoryNamespaceIsDeterministic(t *testing.T) {
	a := NewNamespace("scenario/one")
	b := NewNamespace("scenario/one")

	// **نفسُ السيناريو ⇒ نفسُ التسلسل.**
	for i := 0; i < 20; i++ {
		if pa, pb := a.Phone(), b.Phone(); pa != pb {
			t.Fatalf("الهاتفُ %d اختلف: %s ≠ %s — النطاقُ ليس حتميّاً", i, pa, pb)
		}
	}

	// ══════════════════════════════════════════════════════════════════
	// **وشكلُ الهاتف شكلُ المحمول السوريّ** (٢٠٢٦-٠٩-١٣)
	// ══════════════════════════════════════════════════════════════════
	//
	// **وكان `+9639` وعشرةَ أرقامٍ = ثلاثةَ عشرَ حرفاً بعد الزائد** —
	// **أقصى ما تسمح به E.164، وهو رقمٌ صالحٌ في العالم وليس سوريّاً.**
	//
	// **والمحمولُ السوريُّ `+963` وتسعةٌ تبدأ بـ9** — **ومُعامَلٌ لا
	// يقبله المنتَجُ لا يقيس المنتَج**، بل بابَه الخلفيّ.
	//
	// **وتُقاس صحّتُه بالتطبيع نفسِه** — **ولا قاعدةَ ثانيةً تُكتب هنا
	// فتفترق عن التي تحرس المنتَج.**
	p := NewNamespace("shape").Phone()
	if !identity.IsSyrianMobile(p) {
		t.Fatalf("**هاتفُ المعمل ليس محمولاً سوريّاً**: %q", p)
	}
	if !strings.HasPrefix(p, "+9639") || len(p) != 13 {
		t.Fatalf("شكلُ الهاتف %q لا يطابق +9639 وثمانيةَ أرقام", p)
	}
}

func TestFactoryNamespacesDoNotCollide(t *testing.T) {
	a := NewNamespace("scenario/alpha")
	b := NewNamespace("scenario/beta")

	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		seen[a.Phone()] = true
	}
	for i := 0; i < 200; i++ {
		if seen[b.Phone()] {
			t.Fatalf("تصادمٌ بين نطاقين عند الرقم %d — التوازي غيرُ آمن", i)
		}
	}
}

func TestFactoryNamesCarryScenario(t *testing.T) {
	n := NewNamespace("TestSomething/deep/case")
	got := n.Name("store")
	if !strings.Contains(got, "case") || !strings.HasPrefix(got, "QA/") {
		t.Fatalf("الاسمُ %q لا يقول أيُّ سيناريوٍ أنشأه", got)
	}
}

// **والبذرةُ تُبدَّل عمداً لا عشوائيّاً.**
func TestFactorySeedChangesOutput(t *testing.T) {
	before := NewNamespace("same").Phone()
	t.Setenv("RAHALGO_TEST_SEED", "other")
	after := NewNamespace("same").Phone()
	if before == after {
		t.Fatal("تبديلُ البذرة لم يبدّل الناتج")
	}
	t.Setenv("RAHALGO_TEST_SEED", "")
	if NewNamespace("same").Phone() != before {
		t.Fatal("إعادةُ البذرة لم تُعِد الناتجَ الأوّل")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٢ · الساعة**
// ══════════════════════════════════════════════════════════════════════

func TestFactoryClockIsFixed(t *testing.T) {
	c := NewClock()
	if c.Now() != NewClock().Now() {
		t.Fatal("الساعةُ ليست ثابتة")
	}
	if !c.Ago(time.Hour).Before(c.Now()) {
		t.Fatal("Ago لا يسبق")
	}
	if !c.Ahead(time.Hour).After(c.Now()) {
		t.Fatal("Ahead لا يلحق")
	}
	// **٢٥ ساعةً تتجاوز مهلةَ مفتاح التكرار** — وهي ما يحتاجه `R8`.
	if c.Now().Sub(c.Ago(25*time.Hour)) < 24*time.Hour {
		t.Fatal("الإزاحةُ لا تكفي لتجاوز مهلة اليوم")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٣ · حارسُ الإنتاج — والبرهانُ بالتجربة**
// ══════════════════════════════════════════════════════════════════════
//
// (البند ١٩: **رفضٌ قاطعٌ بلا تخطٍّ سهل.**)

func TestProductionDatabaseGuard(t *testing.T) {
	rejected := []struct {
		name string
		url  string
	}{
		{"قاعدةُ الإنتاج بعينها", "postgres://u:p@localhost:5432/rahalgo?sslmode=disable"},
		{"بلا لاحقة", "postgres://u:p@localhost:5432/rahalgo_prod"},
		{"مضيفٌ بعيدٌ بالاسم", "postgres://u:p@db.rahalgo.com:5432/rahalgo_test"},
		{"مضيفٌ بعيدٌ برقمه", "postgres://u:p@195.201.141.130:5432/rahalgo_test"},
		{"بلا اسمِ قاعدة", "postgres://u:p@localhost:5432/"},
		{"فارغ", ""},
		{"لا يُقرأ", ":://???"},
	}
	for _, c := range rejected {
		if err := checkTestDatabase(c.url); err == nil {
			t.Errorf("الحارسُ قَبِل ما يجب أن يرفض — %s: %s", c.name, c.url)
		}
	}

	accepted := []string{
		"postgres://rahalgo:x@localhost:5434/rahalgo_test?sslmode=disable",
		"postgres://u:p@127.0.0.1:5432/qa_test",
		"postgres://u:p@postgres:5432/rahalgo_test",
	}
	for _, u := range accepted {
		if err := checkTestDatabase(u); err != nil {
			t.Errorf("الحارسُ رفض قاعدةَ اختبارٍ صالحة: %s — %v", u, err)
		}
	}
	t.Log("PRODUCTION DATABASE GUARD = PROVEN — رُفض ٧ ونُفّذ ٣")
}

// **وقاعدةُ الإنتاج الحقيقيّةُ مرفوضةٌ بالاسم** — **وهي التي على السيرفر.**
func TestGuardRejectsActualProductionTarget(t *testing.T) {
	// **مصدرُه `deploy/compose.yml`**: `postgres://rahalgo:…@postgres:5432/rahalgo`.
	const prod = "postgres://rahalgo:secret@postgres:5432/rahalgo?sslmode=disable"
	if err := checkTestDatabase(prod); err == nil {
		t.Fatal("الحارسُ قَبِل قاعدةَ الإنتاج الفعليّة — وهذا أخطرُ ما يمكن")
	} else {
		t.Logf("رُفضت قاعدةُ الإنتاج الفعليّة: %v", err)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٤ · ما يحتاج قاعدة — ويُتخطّى بصدقٍ إن غابت**
// ══════════════════════════════════════════════════════════════════════

func TestFactoryBuildsUserStates(t *testing.T) {
	h := New(t) // **يتخطّى إن لم تُضبط TEST_DATABASE_URL**
	f := h.Factory()

	u := f.NewUserWith("customer", Suspended(), MustChangePassword())
	var status string
	var must bool
	if err := h.Pool.QueryRow(f.ctx(),
		`SELECT status, must_change_password FROM users WHERE id = $1::uuid`,
		u.ID).Scan(&status, &must); err != nil {
		t.Fatalf("تعذّرت قراءةُ المستخدم: %v", err)
	}
	if status != "suspended" || !must {
		t.Fatalf("الحالُ %q · وإجبارُ التبديل %v — والمنتظَرُ suspended/true", status, must)
	}
}

func TestFactoryDriverStates(t *testing.T) {
	h := New(t)
	f := h.Factory()

	d := f.Driver(OnShift(), CashHeld(50_000),
		LocationAt(35.95, 39.01, f.CK.Ago(2*time.Minute)))

	var onShift bool
	var held int64
	if err := h.Pool.QueryRow(f.ctx(), `
		SELECT u.on_shift, COALESCE(b.held, 0)
		FROM users u LEFT JOIN driver_cash_boxes b ON b.driver_id = u.id
		WHERE u.id = $1::uuid`, d.ID).Scan(&onShift, &held); err != nil {
		t.Fatalf("تعذّرت قراءةُ السائق: %v", err)
	}
	if !onShift || held != 50_000 {
		t.Fatalf("ورديّةٌ %v · نقدٌ %d — والمنتظَرُ true/50000", onShift, held)
	}

	// **والصندوقُ يطابق دفترَه** — **وهو ما يجعل الفكسچرَ متّسقاً.**
	var entries int64
	_ = h.Pool.QueryRow(f.ctx(),
		`SELECT COALESCE(sum(amount), 0) FROM driver_cash_entries WHERE driver_id = $1::uuid`,
		d.ID).Scan(&entries)
	if entries != held {
		t.Fatalf("الصندوقُ %d ودفترُه %d — الفكسچرُ غيرُ متّسق", held, entries)
	}
}

func TestFactoryFinanciallyConsistent(t *testing.T) {
	h := New(t)
	f := h.Factory()

	u := f.NewUserWith("customer")
	// **والأنواعُ هنا لا تشترط مرجعاً** — `order_payment` و`refund`
	// تشترطان طلباً (عقدُ `P-4`)، **وقيدٌ منهما بلا طلبٍ حالٌ لا تقع في
	// الواقع** — **وفكسچرٌ يبني المستحيلَ يختبر المستحيل.**
	// والمقصودُ هنا: **الرصيدُ يتبع الدفتر**، وهو يُثبَت بأيّ نوع.
	f.Credit(u.ID, 25_000, "topup")
	f.Credit(u.ID, -5_000, "adjustment")

	bal, sum := f.Balance(u.ID), f.LedgerSum(u.ID)
	if bal != sum {
		t.Fatalf("الرصيدُ %d ودفترُه %d — الفكسچرُ الطبيعيُّ كسر الاتّساق", bal, sum)
	}
	if bal != 20_000 {
		t.Fatalf("الرصيدُ %d لا ٢٠٬٠٠٠", bal)
	}
}

// **والإفسادُ يقع بابٍ مسمّىً وحدَه.**
func TestUnsafeFixtureBreaksConsistencyOnPurpose(t *testing.T) {
	h := New(t)
	f := h.Factory()

	u := f.NewUserWith("customer")
	f.Credit(u.ID, 10_000, "topup")
	f.UnsafeCorruptBalance(u.ID, 999_999)

	if f.Balance(u.ID) == f.LedgerSum(u.ID) {
		t.Fatal("UnsafeCorruptBalance لم تكسر الاتّساق — والحارسُ لن يُختبَر بها")
	}
	t.Log("UnsafeCorruptBalance تكسر عمداً — ولاختبار الحارس وحدَه")
}

func TestFactoryMerchantAndRep(t *testing.T) {
	h := New(t)
	f := h.Factory()

	rep := f.RepAccount()
	m := f.Merchant(OwnedByRep(rep.ID), Commission(15))

	var repID, status *string
	var pct *int64
	if err := h.Pool.QueryRow(f.ctx(), `
		SELECT sales_rep_user_id::text, status, commission_percent
		FROM merchants WHERE id = $1::uuid`, m.ID).Scan(&repID, &status, &pct); err != nil {
		t.Fatalf("تعذّرت قراءةُ المتجر: %v", err)
	}
	if repID == nil || *repID != rep.ID {
		t.Fatal("المتجرُ لم يُنسَب إلى المندوب")
	}
	if pct == nil || *pct != 15 {
		t.Fatal("نسبةُ العمولة لم تُضبط")
	}
	// **والمصنعُ لا ينفّذ نقلاً** — **النقلُ فعلٌ يختبره الاختبار.**
}

func TestFactoryCleansUpAfterItself(t *testing.T) {
	h := New(t)
	f := h.Factory()

	var id string
	t.Run("scenario", func(sub *testing.T) {
		sh := &Harness{T: sub, Pool: h.Pool, Srv: h.Srv, tokens: h.tokens}
		sf := sh.Factory()
		id = sf.NewUserWith("customer").ID
	})

	var n int
	if err := h.Pool.QueryRow(f.ctx(),
		`SELECT count(*) FROM users WHERE id = $1::uuid`, id).Scan(&n); err != nil {
		t.Fatalf("تعذّر العدّ: %v", err)
	}
	if n != 0 {
		t.Fatalf("بقي %d صفّاً بعد التنظيف — السيناريو يلوّث ما بعده", n)
	}
}

// **ونوعُ قيدٍ لا يكتبه أحدٌ يُرفض** — `XOB-9` مؤجَّلٌ إلى `P-4`،
// **والمصنعُ لا يفتح له باباً في الأثناء.**
func TestFactoryRejectsUnusedLedgerKinds(t *testing.T) {
	for _, k := range []string{"penalty", "platform_expense", "platform_profit", "reward"} {
		if allowedLedgerKind(k) {
			t.Errorf("المصنعُ يقبل نوعاً غيرَ مستعمَل: %q — انظر XOB-9", k)
		}
	}
	for _, k := range []string{"topup", "commission", "payout", "refund"} {
		if !allowedLedgerKind(k) {
			t.Errorf("المصنعُ يرفض نوعاً مستعمَلاً: %q", k)
		}
	}
}

// حالُ القاعدة — **يُطبع ليُعرَف ما نُفِّذ وما تُخطّي.**
func TestDatabaseAvailability(t *testing.T) {
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Log("TEST_DATABASE_URL غيرُ مضبوط — اختباراتُ القاعدة تُتخطّى، ولا تُعدّ ناجحة")
		return
	}
	t.Log("قاعدةُ اختبارٍ متاحة — اختباراتُ القاعدة تُنفَّذ")
}
