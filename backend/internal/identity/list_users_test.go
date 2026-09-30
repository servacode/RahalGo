package identity

// ══════════════════════════════════════════════════════════════════════
//  **قائمةُ الحسابات — والعدّادُ يقول ما يعرضه الجدول**  `BOOK-01`
// ══════════════════════════════════════════════════════════════════════
//
// **قِيس حيّاً ٢٠٢٦-٠٩-٣٠ على أوّل شاشةٍ في تجربة المنصّة**:
//
//	GET /admin/users  ⇒  total=1  ·  وصفرُ صفوف
//
// **والسبب استعلامان يستثنيان الأدمن بصيغتين غير مكافئتين**: المجموعُ
// بـ`NOT EXISTS`، والصفوفُ بـ`HAVING NOT bool_or(... = 'admin')`.
//
// **ولحسابٍ بلا أيِّ دور**: `NOT EXISTS` تردّ `true` فيُعَدّ، **و`bool_or`
// على مجموعةٍ فارغةٍ تردّ `NULL`** فـ`NOT NULL` = `NULL`، **و`HAVING` تطرح
// ما ليس `true`** — **فيُعَدّ ولا يُعرَض.**
//
// # ولماذا هذا أكبرُ من شاشةٍ فارغة
//
// **كلُّ حسابٍ بلا دورٍ كان شبحاً**: حسابٌ سُحب دورُه، أو أُنشئ ولم يُكمل،
// أو حسابٌ نظاميّ — **يزيد العدّادَ ولا يظهر، فلا يجده الأدمنُ ولا يُديره.**
//
// # وقرارُ المالك
//
// **والأدمنُ يُعرَض كغيره** (٢٠٢٦-٠٩-٣٠): «لا يجوز إخفاؤه ويبدو كعطب».
// **والإخفاءُ لم يكن حمايةً**: بابُ الكتابة مفتوحٌ لمن يملك القدرةَ ويعرف
// المعرّف — **والحمايةُ موضعُها حرّاسُ الكتابة لا حجبُ القراءة.**
//
// # وينطَّق على حساباتِه وحدَها
//
// **قاعدةُ الاختبار لا تُنظَّف** وفيها عشراتُ الآلاف. **فالشرطُ على الكلّ
// يقيس تاريخَ الجهاز لا هذا الاختبار** — فيُوسَم الاسمُ بوسمٍ فريدٍ وتُنقّى
// القائمةُ به.

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// tagUser يضع وسماً فريداً في اسم الحساب — **فتُنقّى القائمةُ به.**
func tagUser(t *testing.T, pool *pgxpool.Pool, id, name string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`UPDATE users SET full_name = $2 WHERE id = $1::uuid`, id, name); err != nil {
		t.Fatalf("وسمُ الحساب: %v", err)
	}
}

// ── أ · العدّادُ يساوي ما يُعرَض ───────────────────────────────────────
//
// **وهو الثابتُ الذي انكسر**: صفحةٌ واحدةٌ تتّسع للكلّ ⇒ `total` = عددُ
// الصفوف. **ولا يُقبل عدّادٌ يقول ما لا يُرى.**
func TestListUsers_TotalEqualsRowsOnOnePage(t *testing.T) {
	pool := testdb.Pool(t)
	r := NewRepo(pool)
	ctx := context.Background()

	// **حسابٌ بلا أيِّ دور** — وهو الحالةُ التي كانت تُطرَح.
	roleless := testdb.NewUser(t, pool, "")
	// **والوسمُ من معرّفٍ تولّده القاعدة** — فريدٌ بالقاعدة لا بالرجاء.
	tag := "BOOK01-" + roleless[:8]
	tagUser(t, pool, roleless, tag+"-بلا-دور")
	// **وحسابٌ بدورِ زبون** — الحالةُ العاديّة.
	customer := testdb.NewUser(t, pool, "customer")
	tagUser(t, pool, customer, tag+"-زبون")
	// **وحسابُ أدمن** — يُعرَض بقرار المالك.
	admin := testdb.NewUser(t, pool, "admin")
	tagUser(t, pool, admin, tag+"-أدمن")

	users, total, err := r.ListUsers(ctx, tag, "", false, "", 100, 0)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if total != len(users) {
		t.Fatalf("**العدّادُ %d وعددُ الصفوف %d** — وصفحةٌ واحدةٌ تتّسع للكلّ: "+
			"**عدّادٌ يقول ما لا يُرى** (BOOK-01)", total, len(users))
	}
	if total != 3 {
		t.Fatalf("أُنشئت ثلاثةُ حساباتٍ موسومةٍ وردّت القائمةُ %d — "+
			"**وحسابٌ لا يظهر لا يُدار**", total)
	}

	seen := map[string]bool{}
	for _, u := range users {
		seen[u.ID] = true
	}
	if !seen[roleless] {
		t.Error("**حسابٌ بلا دورٍ لا يظهر** — شبحٌ يزيد العدّادَ ولا يُدار (BOOK-01)")
	}
	if !seen[customer] {
		t.Error("حسابُ زبونٍ لا يظهر — والقائمةُ قائمةُ الحسابات")
	}
	if !seen[admin] {
		t.Error("**حسابُ الأدمن لا يظهر** — وقرارُ المالك ٢٠٢٦-٠٩-٣٠ أن يُعرَض كغيره")
	}
}

// ── ب · وتنقيةُ الدور تجد من يحمله ────────────────────────────────────
//
// **وكان `?role=admin` يردّ صفراً دائماً** — لأنّ المجموعَ يستثني الأدمن،
// **ومُنقِّيةٌ لا تجد شيئاً أبداً أسوأُ من غيابها**: تُقرأ «لا أدمن».
func TestListUsers_RoleFilterFindsAdmin(t *testing.T) {
	pool := testdb.Pool(t)
	r := NewRepo(pool)
	ctx := context.Background()

	admin := testdb.NewUser(t, pool, "admin")
	tag := "BOOK01R-" + admin[:8]
	tagUser(t, pool, admin, tag+"-أدمن")
	other := testdb.NewUser(t, pool, "customer")
	tagUser(t, pool, other, tag+"-زبون")

	users, total, err := r.ListUsers(ctx, tag, "admin", false, "", 100, 0)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if total == 0 || len(users) == 0 {
		t.Fatalf("**تنقيةُ role=admin ردّت صفراً والأدمنُ موجود** — "+
			"total=%d صفوف=%d (BOOK-01)", total, len(users))
	}
	if total != len(users) {
		t.Fatalf("العدّادُ %d والصفوفُ %d في تنقيةِ دور", total, len(users))
	}
	for _, u := range users {
		has := false
		for _, rc := range u.Roles {
			if rc == "admin" {
				has = true
			}
		}
		if !has {
			t.Errorf("تنقيةُ role=admin ردّت حساباً بلا الدور: %s", u.Phone)
		}
	}
}
