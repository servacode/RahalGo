// إثباتاتُ القاعدة — **المرحلةُ `P-3V`.**
//
// **وثلاثةُ ادّعاءاتٍ بقيت بلا برهانٍ في `P-3`**: **العزلُ · والتنظيفُ ·
// والاتّساقُ الماليّ.** **وتُثبَت هنا على بوستغرس حقيقيّةٍ أو تُتخطّى
// بصدق** — **ولا تُكتب لها `PASS` بلا تشغيل.**
package qa

import (
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **١ · هويّةُ القاعدة — تُثبَت قبل أوّل كتابة**
// ══════════════════════════════════════════════════════════════════════
//
// (البند ٣ من طلب المالك: **إن لم تُثبَت، لا يُكتب شيء.**)

func TestDatabaseIdentityProof(t *testing.T) {
	raw := os.Getenv("TEST_DATABASE_URL")
	if raw == "" {
		t.Skip("TEST_DATABASE_URL غير مضبوط — لا هويّةَ تُثبَت")
	}

	// **الحارسُ أوّلاً** — **فما لم يُثبَت أنّه ليس إنتاجاً لا يُلمَس.**
	if err := checkTestDatabase(raw); err != nil {
		t.Fatalf("رُفضت القاعدة قبل أيّ كتابة: %v", err)
	}

	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("عنوانٌ لا يُقرأ")
	}
	name := strings.TrimPrefix(u.Path, "/")

	h := New(t)
	var pgVersion, postgis string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT version()`).Scan(&pgVersion); err != nil {
		t.Fatalf("تعذّرت قراءةُ نسخة بوستغرس: %v", err)
	}
	// **وPostGIS شرطٌ**: الهجراتُ تستعمل `geography(Point,4326)`.
	if err := h.Pool.QueryRow(ctxBG(), `SELECT postgis_lib_version()`).Scan(&postgis); err != nil {
		t.Fatalf("PostGIS غيرُ مثبَّتة — والهجراتُ تشترطها: %v", err)
	}

	var current string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT current_database()`).Scan(&current)

	t.Logf("HOST     = %s", u.Hostname())
	t.Logf("PORT     = %s", u.Port())
	t.Logf("DATABASE = %s  (current_database = %s)", name, current)
	t.Logf("POSTGRES = %s", firstLine(pgVersion))
	t.Logf("POSTGIS  = %s", postgis)
	t.Logf("URL      = %s", redact(raw))
	t.Log("THIS IS NOT PRODUCTION — اللاحقةُ _test · والمضيفُ محلّيّ · والاسمُ ليس إنتاجيّاً")

	if current != name {
		t.Errorf("اسمُ القاعدة في العنوان %q وفي الاتّصال %q — لا تطابق", name, current)
	}
}

func firstLine(s string) string {
	if i := strings.IndexAny(s, "\n("); i > 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

// redact يُخفي كلمةَ المرور — **ولا تُطبع في سجلٍّ أبداً.**
func redact(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "***"
	}
	if u.User != nil {
		u.User = url.UserPassword(u.User.Username(), "***")
	}
	return u.String()
}

// ══════════════════════════════════════════════════════════════════════
// **٢ · العزل — سيناريوان لا يرى أحدُهما الآخر**
// ══════════════════════════════════════════════════════════════════════
//
// (البند ٧: **إثباتٌ لا وصف.**)

func TestDatabaseIsolationBetweenScenarios(t *testing.T) {
	h := New(t)

	var idA, idB, phoneA, phoneB string
	t.Run("alpha", func(sub *testing.T) {
		f := (&Harness{T: sub, Pool: h.Pool, Srv: h.Srv, tokens: h.tokens}).Factory()
		u := f.NewUserWith("customer")
		idA, phoneA = u.ID, u.Phone
		// **ويرى صفَّه هو.**
		if n := countUser(sub, h, idA); n != 1 {
			sub.Fatalf("alpha لا يرى صفَّه: %d", n)
		}
	})
	t.Run("beta", func(sub *testing.T) {
		f := (&Harness{T: sub, Pool: h.Pool, Srv: h.Srv, tokens: h.tokens}).Factory()
		u := f.NewUserWith("customer")
		idB, phoneB = u.ID, u.Phone
		// **ولا يرى صفَّ alpha** — **لأنّ ذاك نُظّف عند انتهائه.**
		if n := countUser(sub, h, idA); n != 0 {
			sub.Errorf("beta يرى صفَّ alpha — العزلُ مكسور (%d)", n)
		}
	})

	if phoneA == phoneB {
		t.Fatalf("النطاقان أعطيا الهاتفَ نفسَه: %s — التصادمُ واقع", phoneA)
	}
	if idA == idB {
		t.Fatal("المعرّفان متطابقان")
	}
	t.Logf("DATABASE ISOLATION SELF-TEST = PROVEN — %s ≠ %s", phoneA, phoneB)
}

func countUser(t *testing.T, h *Harness, id string) int {
	t.Helper()
	var n int
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM users WHERE id = $1::uuid`, id).Scan(&n); err != nil {
		t.Fatalf("تعذّر العدّ: %v", err)
	}
	return n
}

// ══════════════════════════════════════════════════════════════════════
// **٣ · التنظيف — يمحو سيناريوه ولا يمسّ غيرَه**
// ══════════════════════════════════════════════════════════════════════
//
// (البند ٨.)

func TestCleanupRemovesOnlyItsOwnScenario(t *testing.T) {
	h := New(t)
	f := h.Factory()

	// **باقٍ** — أُنشئ في السيناريو الخارجيّ.
	survivor := f.NewUserWith("customer")

	var doomed string
	t.Run("inner", func(sub *testing.T) {
		sf := (&Harness{T: sub, Pool: h.Pool, Srv: h.Srv, tokens: h.tokens}).Factory()
		u := sf.NewUserWith("driver")
		doomed = u.ID
		if countUser(sub, h, doomed) != 1 {
			sub.Fatal("الصفُّ لم يُنشَأ")
		}
	}) // ← `t.Cleanup` الداخليُّ يقع هنا

	if n := countUser(t, h, doomed); n != 0 {
		t.Errorf("بقي %d صفّاً بعد التنظيف", n)
	}
	if n := countUser(t, h, survivor.ID); n != 1 {
		t.Errorf("التنظيفُ محا صفَّ سيناريوٍ آخر — بقي %d", n)
	}
	t.Log("CLEANUP SELF-TEST = PROVEN — الداخليُّ مُحي والخارجيُّ باقٍ")
}

// ══════════════════════════════════════════════════════════════════════
// **٤ · الاتّساقُ الماليُّ على قاعدةٍ حقيقيّة**
// ══════════════════════════════════════════════════════════════════════
//
// (البند ٩: **إثباتُ اتّساقِ الفكسچر لا بناءُ محرّك الثوابت.**)

func TestFinancialFixtureConsistencyOnDatabase(t *testing.T) {
	h := New(t)
	f := h.Factory()

	u := f.NewUserWith("customer")
	// **والأنواعُ هنا لا تشترط مرجعاً** — `order_payment` و`refund`
	// تشترطان طلباً (عقدُ `P-4`)، **وقيدٌ منهما بلا طلبٍ حالٌ لا تقع في
	// الواقع** — **وفكسچرٌ يبني المستحيلَ يختبر المستحيل.**
	// والمقصودُ هنا: **الرصيدُ يتبع الدفتر**، وهو يُثبَت بأيّ نوع.
	f.Credit(u.ID, 100_000, "topup")
	f.Credit(u.ID, -30_000, "adjustment")
	f.Credit(u.ID, 5_000, "compensation")

	bal, sum := f.Balance(u.ID), f.LedgerSum(u.ID)
	if bal != sum {
		t.Fatalf("المحفظةُ %d ودفترُها %d — الفكسچرُ الطبيعيُّ كسر الاتّساق", bal, sum)
	}
	if bal != 75_000 {
		t.Fatalf("الرصيدُ %d لا ٧٥٬٠٠٠", bal)
	}

	// **وعددُ القيود ثلاثةٌ لا أقلّ** — **فالرصيدُ لم يُكتب وحدَه.**
	var n int
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM wallet_transactions WHERE user_id = $1::uuid`, u.ID).Scan(&n)
	if n != 3 {
		t.Fatalf("القيودُ %d لا ٣ — الرصيدُ كُتب دون دفتره", n)
	}

	// **وصندوقُ السائق دفترٌ ثانٍ — يُثبَت مثلَه.**
	d := f.Driver(CashHeld(60_000))
	var held, entries int64
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(held,0) FROM driver_cash_boxes WHERE driver_id = $1::uuid`, d.ID).Scan(&held)
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(sum(amount),0) FROM driver_cash_entries WHERE driver_id = $1::uuid`, d.ID).Scan(&entries)
	if held != entries || held != 60_000 {
		t.Fatalf("الصندوقُ %d ودفترُه %d — والمنتظَرُ ٦٠٬٠٠٠ في الاثنين", held, entries)
	}

	t.Log("FACTORY DEFAULT FINANCIAL FIXTURE IS INTERNALLY CONSISTENT")
}

// **وحدُّ الإفساد** — **لا يقع من المسار الطبيعيّ.**
func TestCorruptFixtureIsExplicitOnly(t *testing.T) {
	h := New(t)
	f := h.Factory()

	clean := f.NewUserWith("customer")
	f.Credit(clean.ID, 10_000, "topup")
	if f.Balance(clean.ID) != f.LedgerSum(clean.ID) {
		t.Fatal("المسارُ الطبيعيُّ كسر الاتّساق — وهذا ما يجب ألّا يقع")
	}

	broken := f.NewUserWith("customer")
	f.Credit(broken.ID, 10_000, "topup")
	f.UnsafeCorruptBalance(broken.ID, 999_999)
	if f.Balance(broken.ID) == f.LedgerSum(broken.ID) {
		t.Fatal("UnsafeCorruptBalance لم تكسر شيئاً — فلا تصلح لاختبار حارس")
	}
	t.Log("الإفسادُ لا يقع إلّا بابٍ مسمّىً — والمسارُ الطبيعيُّ متّسقٌ دائماً")
}

// ══════════════════════════════════════════════════════════════════════
// **٥ · التوازي**
// ══════════════════════════════════════════════════════════════════════
//
// (البند ١١.)

func TestParallelFactoryScenarios(t *testing.T) {
	h := New(t)

	const n = 6
	ids := make([]string, n)
	phones := make([]string, n)
	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			// **نطاقٌ لكلّ خيطٍ باسمه** — **وهو ما يمنع التصادم.**
			ns := NewNamespace(t.Name() + "/worker" + string(rune('A'+i)))
			phones[i] = ns.Phone()
			var id string
			err := h.Pool.QueryRow(ctxBG(), `
				INSERT INTO users (phone, full_name, status)
				VALUES ($1, $2, 'active') RETURNING id::text`,
				phones[i], ns.Name("par")).Scan(&id)
			if err != nil {
				t.Errorf("الخيطُ %d تعذّر: %v", i, err)
				return
			}
			ids[i] = id
		}(i)
	}
	close(start)
	wg.Wait()

	t.Cleanup(func() {
		for _, id := range ids {
			if id != "" {
				_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM users WHERE id = $1::uuid`, id)
			}
		}
	})

	seen := map[string]bool{}
	for i, p := range phones {
		if p == "" {
			continue
		}
		if seen[p] {
			t.Fatalf("تصادمٌ في الهاتف عند الخيط %d: %s", i, p)
		}
		seen[p] = true
	}
	for i, id := range ids {
		if id == "" {
			t.Fatalf("الخيطُ %d لم يُنشئ صفّاً", i)
		}
	}
	t.Logf("PARALLEL DB FACTORY PROOF = PASS — %d خيطاً بلا تصادم", n)
}

// ══════════════════════════════════════════════════════════════════════
// **٦ · الهجرات**
// ══════════════════════════════════════════════════════════════════════
//
// **و`testdb.Pool` يطبّقها عند الإقلاع** — **فالإثباتُ أنّ الجداولَ قائمة.**

func TestMigrationsApplied(t *testing.T) {
	h := New(t)
	tables := []string{
		"users", "user_roles", "merchants", "orders", "order_events",
		"wallets", "wallet_transactions", "driver_cash_boxes", "driver_cash_entries",
		"idempotency_keys", "audit_log", "tickets", "app_settings",
		"payout_requests", "merchant_leads", "delivery_zones", "cities",
	}
	var missing []string
	for _, tbl := range tables {
		var ok bool
		if err := h.Pool.QueryRow(ctxBG(),
			`SELECT to_regclass($1) IS NOT NULL`, "public."+tbl).Scan(&ok); err != nil || !ok {
			missing = append(missing, tbl)
		}
	}
	if len(missing) > 0 {
		t.Errorf("جداولُ ناقصةٌ بعد الهجرات: %v", missing)
	}
	var applied int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM schema_migrations`).Scan(&applied)
	t.Logf("MIGRATIONS = PASS — %d هجرةً مطبَّقة · و%d جدولاً فُحص", applied, len(tables))
}
