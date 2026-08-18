package referrals_test

// **المكافأةُ مرّةً واحدةً للرقم — والحذفُ لا يُعيدها.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٨: «مرّةً واحدةً الهديّة، ولو حذف حسابَه
//
//	وأعاد تسجيلَ جديدٍ لا يكسب شيئاً. وأيضاً الدعوة نفسُ الشيء».)
//
// # ما يُقاس
//
// **دورةُ الاحتيال كاملةً**: حسابٌ برقمٍ يحجز، ثمّ يُحذف كما يحذفه
// المحرّكُ حرفاً (`phone = 'deleted-'||id`)، ثمّ حسابٌ جديدٌ **بالرقم
// نفسِه** يحاول الحجز.
//
// **ولا يُقاس الحجزُ بمعرّفين مختلفين وحدَه** — ذاك يمرّ في التصميم
// القديم أيضاً: **العطبُ كان أنّ المعرّفَ يتبدّل والرقمَ لا.**
//
// # ولماذا لا تُنادى `GrantSignupBonus` نفسُها
//
// **تحتاج محفظةً وخزينةً وإعدادات** — وتركيبُها هنا يقيس ثلاثةَ أشياءَ
// ويُسقط الاختبارَ لأسبابٍ لا تخصّ ما نحرسه. **والمحروسُ هو الحجز.**

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// claim يحجز كما تفعل `claimPhoneOnce` حرفاً — **والاستعلامُ نفسُه**،
// **ونسختان منه تفترقان يوما.**
const claim = `
	INSERT INTO phone_claims (phone_hash, kind)
	SELECT encode(sha256(((SELECT value FROM app_secrets WHERE key = 'phone_pepper')
	                      || u.phone)::bytea), 'hex'), $2
	FROM users u WHERE u.id = $1
	ON CONFLICT DO NOTHING`

func TestBonusAndReferral_OncePerPhone(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL غير مضبوط")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("تعذّر الاتّصال: %v", err)
	}
	t.Cleanup(db.Close)

	const phone = "+963900000777"
	var first, second string

	t.Cleanup(func() {
		_, _ = db.Exec(ctx, `DELETE FROM phone_claims WHERE phone_hash IN (
			SELECT encode(sha256(((SELECT value FROM app_secrets WHERE key='phone_pepper')
			                      || u.phone)::bytea),'hex')
			FROM users u WHERE u.id::text = ANY($1))`, []string{first, second})
		_, _ = db.Exec(ctx, `DELETE FROM users WHERE id::text = ANY($1)`,
			[]string{first, second})
	})

	// **ويُنظَّف ما خلّفته دورةٌ سابقة** — **واختبارٌ يتخطّى لأنّ رقماً
	// باقٍ من مرّةٍ ماضيةٍ لا يقيس شيئا.**
	for _, p := range []string{phone, "+963900000778"} {
		_, _ = db.Exec(ctx, `DELETE FROM phone_claims WHERE phone_hash =
			encode(sha256(((SELECT value FROM app_secrets WHERE key='phone_pepper')
			               || $1::text)::bytea),'hex')`, p)
		_, _ = db.Exec(ctx, `DELETE FROM users WHERE phone = $1`, p)
	}

	mk := func(p string) string {
		var id string
		if err := db.QueryRow(ctx, `
			INSERT INTO users (phone, full_name, status)
			VALUES ($1, 'فحصُ الرقم', 'active') RETURNING id::text`, p).Scan(&id); err != nil {
			t.Skipf("تعذّر إنشاءُ حساب: %v", err)
		}
		return id
	}
	claimed := func(id, kind string) bool {
		tag, err := db.Exec(ctx, claim, id, kind)
		if err != nil {
			t.Fatalf("تعذّر الحجز: %v", err)
		}
		return tag.RowsAffected() == 1
	}

	// ── ١ · أوّلُ حسابٍ بالرقم يحجز الاثنين ───────────────────────────
	first = mk(phone)
	if !claimed(first, "signup_bonus") {
		t.Fatal("أوّلُ حسابٍ بالرقم مُنع من الهديّة — والحجزُ فارغ")
	}
	if !claimed(first, "referral") {
		t.Fatal("أوّلُ حسابٍ بالرقم مُنع من الدعوة")
	}

	// ── ٢ · ويُحذف كما يحذفه المحرّك — الرقمُ يُحرَّر ────────────────
	if _, err := db.Exec(ctx,
		`UPDATE users SET status='deleted', phone='deleted-'||id::text WHERE id::text=$1`,
		first); err != nil {
		t.Fatalf("تعذّر الحذف: %v", err)
	}

	// ── ٣ · وحسابٌ جديدٌ بالرقم نفسِه لا يكسب شيئا ───────────────────
	second = mk(phone)
	if second == first {
		t.Fatal("المعرّفُ لم يتبدّل — والاختبارُ لا يقيس شيئا")
	}
	if claimed(second, "signup_bonus") {
		t.Error("حسابٌ جديدٌ بالرقم نفسِه أخذ الهديّةَ ثانيةً — **الخزينةُ تُستنزف بحذفٍ وتسجيل**")
	}
	if claimed(second, "referral") {
		t.Error("حسابٌ جديدٌ بالرقم نفسِه نُسب لداعٍ ثانيةً — **ومن دعاه يُكافأ مرّتين**")
	}

	// ── ٤ · ورقمٌ آخرُ لا يُمنع ─────────────────────────────────────
	other := mk("+963900000778")
	t.Cleanup(func() { _, _ = db.Exec(ctx, `DELETE FROM users WHERE id::text=$1`, other) })
	if !claimed(other, "signup_bonus") {
		t.Error("رقمٌ لم يقبض قطُّ مُنع — **الحاجزُ يمنع الجميع لا المحتال**")
	}
	_, _ = db.Exec(ctx, `DELETE FROM phone_claims WHERE phone_hash IN (
		SELECT encode(sha256(((SELECT value FROM app_secrets WHERE key='phone_pepper')
		                      || u.phone)::bytea),'hex')
		FROM users u WHERE u.id::text = $1)`, other)
}
