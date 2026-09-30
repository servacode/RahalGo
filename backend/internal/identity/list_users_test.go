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

// ── ج · والبحثُ بالرقم كما يكتبه الإنسان ──────────────────────────────
//
// ══════════════════════════════════════════════════════════════════════
//
// **الأرقامُ تُخزَّن مطبَّعةً** (`+963…`)، **والبحثُ كان `ILIKE` خامّاً** —
// **فمن كتب `09…` كما هو على بطاقة الموظّف رُدّ بصفر نتائج.**
//
// **ونفيٌ كاذبٌ أسوأُ من خطأ**: يُقرأ «لا حسابَ بهذا الرقم» فيُنشأ الحسابُ
// مرّتين. (قِيس حيّاً ٢٠٢٦-٠٩-٣٠ على حساب موظّفٍ أُنشئ لتوّه. `BOOK-04`.)
func TestListUsers_FindsByLocalPhoneForm(t *testing.T) {
	pool := testdb.Pool(t)
	r := NewRepo(pool)
	ctx := context.Background()

	id := testdb.NewUser(t, pool, "customer")
	var stored string
	if err := pool.QueryRow(ctx,
		`SELECT phone::text FROM users WHERE id = $1::uuid`, id).Scan(&stored); err != nil {
		t.Fatalf("قراءةُ الرقم: %v", err)
	}
	// **والصيغةُ المحلّيّةُ تُشتقّ من المخزون** — لا رقمٌ مكتوبٌ بيد.
	if len(stored) < 5 || stored[:4] != "+963" {
		t.Skipf("الرقمُ المخزون %q ليس بصيغة +963 — لا يُقاس عليه", stored)
	}
	local := "0" + stored[4:]

	for _, form := range []string{stored, local} {
		users, total, err := r.ListUsers(ctx, form, "", false, "", 100, 0)
		if err != nil {
			t.Fatalf("ListUsers(%q): %v", form, err)
		}
		found := false
		for _, u := range users {
			if u.ID == id {
				found = true
			}
		}
		if !found {
			t.Errorf("**البحثُ بـ%q لم يجد الحساب** — total=%d · "+
				"**ونفيٌ كاذبٌ يُنشئ الحسابَ مرّتين** (BOOK-04)", form, total)
		}
	}
}

// ── د · و«الطاقم» صنفُ دورٍ محسوبٌ لا زوجٌ مكتوبٌ بيد ─────────────────
//
// **كان `role_code IN ('ops','finance')`** — **و`ops` مُحالٌ إلى الإرث،
// و`operations` و`customer_support` غائبان.** فمن ضغط «الموظّفون» رأى
// المالية وحدَها. (قِيس حيّاً: ثلاثةُ موظّفين و`role=staff` يردّ واحداً.)
func TestListUsers_StaffFilterCoversEveryStaffRole(t *testing.T) {
	pool := testdb.Pool(t)
	r := NewRepo(pool)
	ctx := context.Background()

	// **وموظّفو المنصّة الخمسة** (قرارُ المالك ٢٠٢٦-٠٩-٣٠): العمليّاتُ
	// والماليةُ وخدمةُ العملاء **ومديرُ المنصّة ومراقبُها** — **والمديرُ
	// صنفُه «مرتفع» لا «موظّف»**، فأوّلُ إصلاحٍ كان يُسقطه.
	roles := []string{"operations", "finance", "customer_support",
		"admin", "platform_monitor"}
	ids := make([]string, 0, len(roles))
	var tag string
	for i, role := range roles {
		id := testdb.NewUser(t, pool, role)
		if i == 0 {
			tag = "BOOK05-" + id[:8]
		}
		tagUser(t, pool, id, tag+"-"+role)
		ids = append(ids, id)
	}

	users, total, err := r.ListUsers(ctx, tag, "staff", false, "", 100, 0)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if total != len(roles) || len(users) != len(roles) {
		t.Fatalf("**مُنقّيةُ «الطاقم» ردّت %d وهم %d** — "+
			"total=%d · **ومن لا يظهر لا يُدار** (BOOK-05)",
			len(users), len(roles), total)
	}
	seen := map[string]bool{}
	for _, u := range users {
		seen[u.ID] = true
	}
	for i, id := range ids {
		if !seen[id] {
			t.Errorf("**موظّف %s لا يظهر في «الطاقم»**", roles[i])
		}
	}

	// **وشرطٌ سالب**: صاحبُ حسابٍ عاديٍّ ليس موظّفاً — **ومُنقّيةٌ تُظهر
	// الكلَّ ليست مُنقّية.**
	for _, acct := range []string{"customer", "driver", "merchant", "sales"} {
		id := testdb.NewUser(t, pool, acct)
		tagUser(t, pool, id, tag+"-"+acct)
		users2, _, err := r.ListUsers(ctx, tag, "staff", false, "", 100, 0)
		if err != nil {
			t.Fatalf("ListUsers: %v", err)
		}
		for _, u := range users2 {
			if u.ID == id {
				t.Errorf("**صاحبُ حساب %s ظهر في «الطاقم»** — وصفةُ الحساب ليست وظيفة", acct)
			}
		}
	}
}

// ── هـ · والحسابُ النظاميُّ يُوسَم ولا يُخفى ──────────────────────────
//
// ══════════════════════════════════════════════════════════════════════
//
// **سأل المالكُ ٢٠٢٦-٠٩-٣٠**: «لم أفهم ما عمل هذا الحساب ولماذا هو موجود
// إذا لا عمل له، احذفه» — **وقِيس قبل أن يُلمَس**:
//
//	orders/cash_settlement.go:44  يقرأ محفظة الاحتباس
//	                         :46  وغيابُها خطأُ XS-2 ⇒ **كلُّ تسويةٍ نقديّةٍ تسقط**
//	fininv/checks.go:654,718,723  ثلاثةُ فحوصٍ ماليّةٍ تقيسها
//
// **ورصيدُه = المستحقُّ النقديُّ القائم للمتاجر** — وصفرُه اليوم يقول
// «لا ديون» لا «لا عمل».
//
// **فالعيبُ في العرض لا في الحساب**: ظهر كأنّه مستخدمٌ غامض.
// **وإخفاؤه عينُ `BOOK-01`** — «حسابٌ يُعَدّ ولا يُعرَض شبح». **فيُوسَم.**
func TestListUsers_MarksSystemAccountWithoutHiding(t *testing.T) {
	pool := testdb.Pool(t)
	r := NewRepo(pool)
	ctx := context.Background()

	// **ومحفظةُ الاحتباس تُضمَن بدالّتها** — لا بصفٍّ مكتوبٍ بيد.
	var sysID string
	if err := pool.QueryRow(ctx,
		`SELECT user_id::text FROM wallets WHERE is_cash_holding LIMIT 1`).Scan(&sysID); err != nil {
		t.Skipf("لا محفظةَ احتباسٍ في قاعدة الاختبار — %v", err)
	}
	var sysPhone string
	if err := pool.QueryRow(ctx,
		`SELECT phone::text FROM users WHERE id = $1::uuid`, sysID).Scan(&sysPhone); err != nil {
		t.Fatalf("قراءةُ رقم الحساب النظاميّ: %v", err)
	}

	users, total, err := r.ListUsers(ctx, sysPhone, "", false, "", 100, 0)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	// **يُعرَض** — ولا يُخفى.
	var found *User
	for i := range users {
		if users[i].ID == sysID {
			found = &users[i]
		}
	}
	if found == nil {
		t.Fatalf("**الحسابُ النظاميُّ لا يظهر في القائمة** — total=%d · "+
			"**والإخفاءُ عينُ BOOK-01**", total)
	}
	// **ويُوسَم** — فلا يُقرأ مستخدماً غامضاً.
	if !found.IsSystem {
		t.Fatal("**الحسابُ النظاميُّ غيرُ موسوم** — فيُقرأ مستخدماً غامضاً، " +
			"وسأل المالكُ عنه فظنّه بلا عمل")
	}

	// **والإنسانُ ليس نظاماً** — ووسمٌ يُوسَم به الكلُّ ليس وسماً.
	human := testdb.NewUser(t, pool, "customer")
	tag := "BOOKSYS-" + human[:8]
	tagUser(t, pool, human, tag)
	hu, _, err := r.ListUsers(ctx, tag, "", false, "", 100, 0)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	for _, u := range hu {
		if u.ID == human && u.IsSystem {
			t.Fatal("**حسابُ زبونٍ وُسم نظاميّاً** — والوسمُ من محفظة الاحتباس وحدَها")
		}
	}
}
