package identity

// ══════════════════════════════════════════════════════════════════════
//  **حسابُ المالك لا يمسّه أحد**  `BOOK-02`
// ══════════════════════════════════════════════════════════════════════
//
// **قِيس ٢٠٢٦-٠٩-٣٠**: `users.status.manage` يملكها `admin` و
// `owner_super_admin` **و`trust_safety`**، **والحارسُ الوحيدُ «لا تفعلها
// بنفسك»** — **فموظّفُ الطاقم كان يستطيع حظرَ حسابِ المالك وإعادةَ تعيين
// كلمتِه وإخراجَه من كلّ جلساته.**
//
// **والسياسةُ كانت مكتوبةً ونصفَ مُطبَّقة**: `authz/roleclass.go` ينصّ أنّ
// المرتفعَ والمحميَّ **لا يُمنحان إلّا من مالك** — وطُبِّق على منح الأدوار
// وحدَه، **لا على إدارة الحساب الذي يحملها.**
//
// **وقرارُ المالك ٢٠٢٦-٠٩-٣٠**: «صاحبُ المنصّة لا أحدَ يستطيع تعديلَ أيِّ
// إجراءٍ يخصّه».

import (
	"context"
	"errors"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// ── أ · موظّفُ طاقمٍ لا يمسّ حسابَ المالك ─────────────────────────────
func TestAccountGuard_StaffCannotTouchOwner(t *testing.T) {
	pool := testdb.Pool(t)
	r := NewRepo(pool)
	ctx := context.Background()

	staff := testdb.NewUser(t, pool, "trust_safety")
	owner := testdb.NewUser(t, pool, "owner_super_admin")

	if err := guardAccountAdmin(ctx, pool, staff, owner); err == nil {
		t.Fatal("**موظّفُ طاقمٍ يمسّ حسابَ المالك** — وقرارُ المالك أنّ حسابَه " +
			"لا يمسّه أحد (BOOK-02)")
	} else if !errors.Is(err, ErrOwnerRoleProtected) {
		t.Fatalf("رُدّ بخطأٍ غيرِ المنتظَر: %v", err)
	}
	_ = r
}

// ── ب · ولا أدمنٌ يمسّ حسابَ المالك ───────────────────────────────────
//
// **وهذا أبعدُ من السياسة القائمة** — قرارُ المالك: **لا أحد**، ولا أدمن.
func TestAccountGuard_AdminCannotTouchOwner(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()

	admin := testdb.NewUser(t, pool, "admin")
	owner := testdb.NewUser(t, pool, "owner_super_admin")

	if err := guardAccountAdmin(ctx, pool, admin, owner); err == nil {
		t.Fatal("**أدمنٌ يمسّ حسابَ المالك** — والقرارُ «لا أحد» (BOOK-02)")
	}
}

// ── ج · والحسابُ المرتفعُ يمسّه المالكُ وحدَه ─────────────────────────
//
// **وهي سياسةُ `GrantByOwner` نفسُها** مطبَّقةً على إدارة الحساب.
func TestAccountGuard_ElevatedNeedsOwner(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()

	staff := testdb.NewUser(t, pool, "trust_safety")
	target := testdb.NewUser(t, pool, "admin")
	owner := testdb.NewUser(t, pool, "owner_super_admin")

	if err := guardAccountAdmin(ctx, pool, staff, target); err == nil {
		t.Error("**موظّفُ طاقمٍ يمسّ حسابَ أدمن** — والمرتفعُ للمالك وحدَه (BOOK-02)")
	}
	if err := guardAccountAdmin(ctx, pool, owner, target); err != nil {
		t.Errorf("**المالكُ مُنع من إدارة حسابِ أدمن**: %v — وهو صاحبُ السلطة", err)
	}
}

// ── د · والحسابُ العاديُّ يُدار كما كان ───────────────────────────────
//
// **وحارسٌ يمنع كلَّ شيءٍ حارسٌ يُنزَع** — فالزبونُ والسائقُ والمتجرُ
// يُدارون بالقدرة كما كانوا، **ولا يُشلّ مكتبُ العمليّات.**
func TestAccountGuard_OrdinaryAccountsUnaffected(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()

	staff := testdb.NewUser(t, pool, "trust_safety")
	for _, role := range []string{"customer", "driver", "merchant", "sales", "operations"} {
		target := testdb.NewUser(t, pool, role)
		if err := guardAccountAdmin(ctx, pool, staff, target); err != nil {
			t.Errorf("**مُنع مسُّ حسابِ %s**: %v — والحارسُ للمحميِّ والمرتفعِ وحدَهما", role, err)
		}
	}
}

// ── هـ · وفعلُ المرءِ بنفسِه ليس إدارةَ غيرِه ─────────────────────────
//
// **وإلّا أُقفل المالكُ عن حسابه هو** — ولا يُعيد كلمتَه ولا يُخرج جلساتَه.
func TestAccountGuard_SelfIsAllowed(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()

	owner := testdb.NewUser(t, pool, "owner_super_admin")
	if err := guardAccountAdmin(ctx, pool, owner, owner); err != nil {
		t.Fatalf("**المالكُ مُنع من حسابِ نفسِه**: %v — وحسابٌ لا يمسّه صاحبُه سجنٌ", err)
	}
}
