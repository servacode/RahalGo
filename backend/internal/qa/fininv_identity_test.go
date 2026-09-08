package qa

// ══════════════════════════════════════════════════════════════════════
// **هويّةُ الثابتِ الماليّ اسمٌ مستقرٌّ لا اسمٌ معروض** — `XG-43`
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان يقع
//
// **ثابتان يجمعان بـ`u.full_name`** — `FI-02.a` (رصيدُ المحفظة) و`FI-10.a`
// (محتجَزُ الصندوق). **والاسمُ المعروضُ ليس هويّة**: هو نصٌّ يتبدّل،
// **ولا يمنع أحدٌ شخصين من حمله.**
//
// # وهو يكذب في الاتّجاهين
//
// **كذبةُ الإنذار**: سليمان متشابها الاسمِ يُدمجان في صفٍّ واحد، فيصير
// مجموعُ قيودِهما مقابلَ رصيدِ أحدهما — **فيُنذَر على صحيح.**
//
// **وكذبةُ السكوت** — وهي الأخطر: **من له رصيدٌ بلا قيدٍ واحد** (مالٌ
// من عدم) **يُدمَج بمن رصيدُه مثلُه وقيودُه سليمة، فيتساوى الطرفان
// ويسكت الحارس.** **وخرقٌ حقيقيٌّ سُتر أسوأُ من إنذارٍ كاذب.**
//
// # وهذا الملفّ يقيس الكذبتين
//
// **والاستعلامان القديمان محفوظان هنا نصّاً** — **فلا يُقاس الإصلاحُ
// بقولي إنّه أُصلح، بل بأنّ القديمَ ما يزال يكذب والجديدَ يصدق على
// المعطياتِ نفسِها.**
//
// **ولا قيدَ على الأسماء في المنتَج**: تشابُهُ الأسماء **معطىً صحيح**،
// والعلّةُ في سؤال الحارس لا في الناس.

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// ── الاستعلامان كما كانا قبل الإصلاح ──────────────────────────────────
//
// **يُحفظان ليُقاس بهما، لا ليُستعملا.**

const brokenFI02a = `
	SELECT u.full_name, w.balance, COALESCE(SUM(t.amount), 0)::bigint AS مجموع_القيود
	FROM wallets w
	JOIN users u ON u.id = w.user_id
	LEFT JOIN wallet_transactions t ON t.user_id = w.user_id
	GROUP BY u.full_name, w.balance
	HAVING w.balance <> COALESCE(SUM(t.amount), 0)`

const brokenFI10a = `
	SELECT u.full_name, b.held, COALESCE(SUM(e.amount), 0)::bigint AS مجموع_القيود
	FROM driver_cash_boxes b
	JOIN users u ON u.id = b.driver_id
	LEFT JOIN driver_cash_entries e ON e.driver_id = b.driver_id
	GROUP BY u.full_name, b.held
	HAVING b.held <> COALESCE(SUM(e.amount), 0)`

// xg43Rows عددُ صفوفِ استعلامٍ خام — **وصفٌّ واحدٌ يعني إنذاراً.**
func xg43Rows(t *testing.T, pool *pgxpool.Pool, sql string) int {
	t.Helper()
	rows, err := pool.Query(context.Background(), sql)
	if err != nil {
		t.Fatalf("استعلامٌ خام: %v", err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		n++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("قراءةُ الصفوف: %v", err)
	}
	return n
}

// xg43User حسابٌ باسمٍ يُعطى — **والاسمُ يتكرّر عمداً.**
func xg43User(t *testing.T, pool *pgxpool.Pool, phone, name string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO users (phone, full_name, status)
		VALUES ($1, $2, 'active') RETURNING id::text`, phone, name).Scan(&id); err != nil {
		t.Fatalf("إنشاءُ حساب: %v", err)
	}
	return id
}

// xg43Subjects معرّفاتُ الأطرافِ في خروقِ ثابتٍ — **العمودُ الأوّل.**
func xg43Subjects(vs []fininv.Violation, id string) []string {
	var out []string
	for _, v := range vs {
		if v.Check.ID != id {
			continue
		}
		for _, r := range v.Rows {
			if len(r) > 0 {
				out = append(out, strings.TrimSpace(toStr(r[0])))
			}
		}
	}
	return out
}

func toStr(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// ══════════════════════════════════════════════════════════════════════
// **متشابها الاسمِ طرفان اثنان لا طرفٌ واحد**
// ══════════════════════════════════════════════════════════════════════
func TestXG43_FinancialIdentityIsStableNotDisplayName(t *testing.T) {
	pool := fixtureDB(t, "rahalgo_xg43_identity")
	ctx := context.Background()

	// **والخزينةُ تُنشأ كما ينشئها الإقلاع** — وإلّا خُرقت `FI-02.c`
	// بلا ذنبِ سيناريو. (وهي علّةُ `fininv_fixture_test` بعينها.)
	if _, err := pool.Exec(ctx, `
		WITH u AS (
		  INSERT INTO users (phone, full_name, status)
		  VALUES ('0900430000', 'إداريُّ الأساس', 'active') RETURNING id)
		INSERT INTO user_roles (user_id, role_code)
		SELECT id, 'admin' FROM u`); err != nil {
		t.Fatalf("إداريُّ الأساس: %v", err)
	}
	if _, err := wallet.EnsureTreasury(ctx, pool); err != nil {
		t.Fatalf("EnsureTreasury: %v", err)
	}

	// **اسمٌ عربيٌّ واحدٌ بحرفه** — لا اختلافَ في الرسم ولا في التشكيل.
	// **فالهويّةُ لا تُحلّ بتطبيعِ نصّ، بل بالمعرّف.**
	const sameName = "محمّد العليّ"

	// **وكلُّ عبارةٍ وحدَها** — pgx لا يجمع عبارتين في بيانٍ محضَّر.
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("زرعُ الحالة: %v", err)
		}
	}

	reset := func(ids ...string) {
		t.Helper()
		if _, err := pool.Exec(ctx,
			`DELETE FROM users WHERE id::text = ANY($1)`, ids); err != nil {
			t.Fatalf("تنظيف: %v", err)
		}
	}

	// ── B1 · كذبةُ الإنذار ────────────────────────────────────────
	t.Run("B1_متشابهان_سليمان_لا_يُنذَر_عليهما", func(t *testing.T) {
		a := xg43User(t, pool, "0900430001", sameName)
		b := xg43User(t, pool, "0900430002", sameName)
		defer reset(a, b)

		// **رصيدان متساويان وقيودُهما مطابقة** — كلٌّ سليمٌ وحدَه.
		exec(`INSERT INTO wallets (user_id, balance) VALUES ($1::uuid, 5000), ($2::uuid, 5000)
			ON CONFLICT (user_id) DO UPDATE SET balance = EXCLUDED.balance`, a, b)
		exec(`INSERT INTO wallet_transactions (user_id, amount, kind)
			VALUES ($1::uuid, 5000, 'adjustment'), ($2::uuid, 5000, 'adjustment')`, a, b)
		exec(`INSERT INTO driver_cash_boxes (driver_id, held) VALUES ($1::uuid, 4000), ($2::uuid, 4000)`, a, b)
		exec(`INSERT INTO driver_cash_entries (driver_id, amount, kind)
			VALUES ($1::uuid, 4000, 'adjustment'), ($2::uuid, 4000, 'adjustment')`, a, b)

		// **القديمُ يُنذر** — والطرفان سليمان.
		oldW := xg43Rows(t, pool, brokenFI02a)
		oldB := xg43Rows(t, pool, brokenFI10a)
		t.Logf("BROKEN AGGREGATE = (full_name, balance) · BROKEN RESULT: FI-02.a=%d صفّاً · FI-10.a=%d صفّاً",
			oldW, oldB)
		if oldW == 0 || oldB == 0 {
			t.Fatal("**القديمُ لم يكذب** — فالحالةُ المزروعةُ لا تُعيد إنتاج `XG-43`")
		}

		vs, err := fininv.Run(ctx, pool, "FI-02.a", "FI-10.a")
		if err != nil {
			t.Fatalf("محرّك: %v", err)
		}
		if len(vs) != 0 {
			t.Errorf("**إنذارٌ كاذبٌ بعد الإصلاح** — %v", vs)
		}
		t.Logf("FALSE POSITIVE: A=%s B=%s · نفسُ الاسم · بعد الإصلاح = صفرُ خروق",
			first8(a), first8(b))
	})

	// ── B2 · كذبةُ السكوت ─────────────────────────────────────────
	t.Run("B2_خرقٌ_حقيقيٌّ_لا_يستره_متشابهُ_اسم", func(t *testing.T) {
		a := xg43User(t, pool, "0900430003", sameName)
		b := xg43User(t, pool, "0900430004", sameName)
		defer reset(a, b)

		// **A مخروقٌ حقّاً**: رصيدٌ ومحتجَزٌ بلا قيدٍ واحد — **مالٌ من عدم.**
		// **وB سليمٌ برصيدٍ مساوٍ** — فيُكمل النقصَ في الجمع بالاسم.
		exec(`INSERT INTO wallets (user_id, balance) VALUES ($1::uuid, 5000), ($2::uuid, 5000)
			ON CONFLICT (user_id) DO UPDATE SET balance = EXCLUDED.balance`, a, b)
		exec(`INSERT INTO wallet_transactions (user_id, amount, kind)
			VALUES ($1::uuid, 5000, 'adjustment')`, b)
		exec(`INSERT INTO driver_cash_boxes (driver_id, held) VALUES ($1::uuid, 4000), ($2::uuid, 4000)`, a, b)
		exec(`INSERT INTO driver_cash_entries (driver_id, amount, kind)
			VALUES ($1::uuid, 4000, 'adjustment')`, b)

		// **والقديمُ يسكت** — وهذا هو المقصود.
		oldW := xg43Rows(t, pool, brokenFI02a)
		oldB := xg43Rows(t, pool, brokenFI10a)
		t.Logf("A REAL VIOLATION = رصيدٌ ٥٠٠٠ ومحتجَزٌ ٤٠٠٠ بلا قيد · B OFFSET = ٥٠٠٠ و٤٠٠٠ بقيودها · "+
			"BROKEN RESULT: FI-02.a=%d · FI-10.a=%d", oldW, oldB)
		if oldW != 0 || oldB != 0 {
			t.Fatal("**القديمُ لم يستر** — فالحالةُ لا تُثبت السترَ المدّعى")
		}

		vs, err := fininv.Run(ctx, pool, "FI-02.a", "FI-10.a")
		if err != nil {
			t.Fatalf("محرّك: %v", err)
		}
		w := xg43Subjects(vs, "FI-02.a")
		c := xg43Subjects(vs, "FI-10.a")
		if len(w) != 1 || w[0] != a {
			t.Errorf("FI-02.a: الأطرافُ المخروقةُ %v والمنتظَرُ [%s] وحدَه", w, a)
		}
		if len(c) != 1 || c[0] != a {
			t.Errorf("FI-10.a: الأطرافُ المخروقةُ %v والمنتظَرُ [%s] وحدَه", c, a)
		}
		t.Logf("MASKING: بعد الإصلاح = A (%s) وحدَه، وB سليمٌ لم يُتّهم", first8(a))
	})

	// ── تبديلُ الاسمِ لا يبدّل الحقيقةَ الماليّة ────────────────────
	t.Run("تبديلُ_الاسمِ_لا_يبدّل_الحقيقة", func(t *testing.T) {
		a := xg43User(t, pool, "0900430005", "اسمٌ أوّل")
		defer reset(a)

		exec(`INSERT INTO wallets (user_id, balance) VALUES ($1::uuid, 7000)
			ON CONFLICT (user_id) DO UPDATE SET balance = EXCLUDED.balance`, a)
		exec(`INSERT INTO driver_cash_boxes (driver_id, held) VALUES ($1::uuid, 3000)`, a)

		before, err := fininv.Run(ctx, pool, "FI-02.a", "FI-10.a")
		if err != nil {
			t.Fatalf("محرّك: %v", err)
		}
		if _, err := pool.Exec(ctx,
			`UPDATE users SET full_name = $2 WHERE id::text = $1`, a, "اسمٌ آخرُ تماماً"); err != nil {
			t.Fatalf("تبديلُ الاسم: %v", err)
		}
		after, err := fininv.Run(ctx, pool, "FI-02.a", "FI-10.a")
		if err != nil {
			t.Fatalf("محرّك: %v", err)
		}

		// **والمقارنةُ بالطرفِ والأرقام** — **والاسمُ وصفٌ لا حكم**، فيُطرح.
		fw, fc := xg43Fingerprint(before), xg43Fingerprint(after)
		if fw != fc {
			t.Errorf("**تبدّلت الحقيقةُ الماليّةُ بتبديل الاسم**:\nقبل: %s\nبعد: %s", fw, fc)
		}
		if len(xg43Subjects(after, "FI-02.a")) != 1 {
			t.Error("**ضاع الخرقُ بعد التبديل** — والاسمُ لا يملك ذلك")
		}
		t.Logf("NAME CHANGE: الطرفُ %s · الخرقانِ باقيانِ بالأرقام نفسِها", first8(a))
	})

	// ── والأساسُ نظيفٌ حين يُنظَّف ──────────────────────────────────
	t.Run("P4_نظيفٌ_على_أساسٍ_صحيح", func(t *testing.T) {
		all, err := fininv.Run(ctx, pool)
		if err != nil {
			t.Fatalf("محرّك: %v", err)
		}
		t.Logf("P-4: %d ثابتاً · %d خرقاً", len(fininv.Select()), len(all))
		for _, v := range all {
			t.Errorf("**خرقٌ على أساسٍ صحيح** — %s", v)
		}
	})
}

// xg43Fingerprint بصمةُ الخروق بلا الاسمِ المعروض.
//
// **العمودُ الأوّلُ معرّفٌ والثاني اسمٌ** — **فيُسقَط الثاني**، ويبقى ما
// يُحكَم به.
func xg43Fingerprint(vs []fininv.Violation) string {
	var b strings.Builder
	for _, v := range vs {
		b.WriteString(v.Check.ID)
		for _, r := range v.Rows {
			b.WriteString("|")
			for i, cell := range r {
				if i == 1 {
					continue
				}
				fmt.Fprintf(&b, "%v,", cell)
			}
		}
		b.WriteString(";")
	}
	return b.String()
}

// ══════════════════════════════════════════════════════════════════════
// **حارسٌ دائم: لا ثابتَ ماليٌّ يُعرّف طرفَه باسمه المعروض**
// ══════════════════════════════════════════════════════════════════════
//
// **والحارسُ على معنى الاستعلام لا على شكله**: تُطوى المسافاتُ
// والأسطرُ أوّلاً، **فتنسيقُ SQL لا يُسقطه ولا يُمرّره.**
//
// **ويشمل السجلَّ كلَّه** — لا `FI-02.a` و`FI-10.a` وحدَهما، **فالعلّةُ
// قاعدةٌ لا موضعان.**
func TestXG43_NoInvariantUsesDisplayIdentity(t *testing.T) {
	// **حقولُ العرض** — نصوصٌ يكتبها الناسُ وتتبدّل ولا تُلزَم بتفرّد.
	display := regexp.MustCompile(`(?i)(full_name|display_name|[a-z_]*\.name\b|\bphone\b)`)
	// **الهويّاتُ المستقرّة** — مفتاحٌ أو مرجعٌ فريدٌ في الجدول.
	stable := regexp.MustCompile(`(?i)(\bid\b|[a-z_]+_id\b|\.id\b|\bnumber\b|\bref\b|\bperiod\b)`)

	flat := regexp.MustCompile(`\s+`)
	checked, flagged := 0, 0
	for _, c := range fininv.All {
		if strings.TrimSpace(c.SQL) == "" {
			continue
		}
		checked++
		sql := flat.ReplaceAllString(c.SQL, " ")

		// ── ١ ── لا تجميعَ بحقلِ عرضٍ بلا هويّةٍ مستقرّة ──────────
		rest := sql
		for {
			i := strings.Index(strings.ToUpper(rest), "GROUP BY")
			if i < 0 {
				break
			}
			rest = rest[i+len("GROUP BY"):]
			keys := rest
			for _, stop := range []string{"HAVING", "ORDER BY", "LIMIT", "UNION", ")"} {
				if j := strings.Index(strings.ToUpper(keys), stop); j >= 0 {
					keys = keys[:j]
				}
			}
			if display.MatchString(keys) && !stable.MatchString(keys) {
				flagged++
				t.Errorf("**%s يجمع بحقلِ عرضٍ بلا هويّةٍ مستقرّة** — `GROUP BY%s`\n"+
					"  **والاسمُ ليس هويّة**: يتبدّل، ويحمله اثنان. (`XG-43`)", c.ID, keys)
			}
		}

		// ── ٢ ── ولا مطابقةَ ولا وصلَ بحقلِ عرض ───────────────────
		//
		// **الجمعُ ليس البابَ الوحيد**: `JOIN … ON u.full_name = …`
		// يدمج الأطرافَ كما يدمجها التجميع.
		cmp := regexp.MustCompile(`(?i)(full_name|display_name|\bphone\b)\s*(=|<>|!=|\bIN\b|\bLIKE\b)` +
			`|(=|<>|!=)\s*[a-z_]*\.?(full_name|display_name|phone)\b`)
		if cmp.MatchString(sql) {
			flagged++
			t.Errorf("**%s يطابق بحقلِ عرض** — والمطابقةُ بالاسم هويّةٌ بالاسم. (`XG-43`)", c.ID)
		}
	}
	t.Logf("IDENTITY GUARD: %d ثابتاً فُحصت · %d مخالفاً", checked, flagged)
	if checked < 50 {
		t.Errorf("**فُحص %d ثابتاً فقط** — والحارسُ الذي لا يرى السجلَّ لا يحرسه", checked)
	}
}
