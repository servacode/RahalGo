package identity

// **دورٌ واحدٌ ومعه الزبون — ولا ثالث.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٠: «ما يصير سائقاً ومندوباً وصاحبَ متجرٍ معاً —
//  يعني دورين فقط».)
//
// # ولماذا يُحرَس في المحرّك لا في القائمة
//
// **قائمةُ اختيارٍ في الشاشة تمنع لا تمنع شيئاً** — من نادى الواجهةَ
// البرمجيّة مباشرةً تجاوزها. **وحجبٌ في العرض وحدَه وعدٌ بحجب.**
//
// # وضررُ الجمع لا يُكشف في مراجعة
//
// **مندوبٌ هو صاحبُ متجرٍ يمنح متجرَه عمولةَ نفسِه**، **وسائقٌ هو مندوبٌ
// يُسند لنفسه.** **وكلُّ فعلٍ منهما مشروعٌ وحدَه** — فلا يقف عليه من يراجع.
//
// # وبابان لا واحد
//
// **يُحرَس المنحُ ويُترك الإنشاء فيدخل من هناك ما مُنع من هنا.** فيُفحص
// الحارسان معاً: `checkOnePrimary` لِما يُنشأ، و`ensureOnePrimary` لِما
// يُمنح — **وهما اللذان تناديهما `AdminCreateUser` و`AdminGrantRole`.**

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestOneRole_CreateRefusesTwoPrimaries **لا يُنشأ حسابٌ بدورين أساسيّين.**
func TestOneRole_CreateRefusesTwoPrimaries(t *testing.T) {
	for _, c := range []struct {
		name  string
		roles []string
		ok    bool
	}{
		{"سائقٌ وحدَه", []string{"driver"}, true},
		{"سائقٌ وزبون", []string{"driver", "customer"}, true},
		{"زبونٌ وحدَه", []string{"customer"}, true},
		{"أدمنُ وزبون", []string{"admin", "customer"}, true},
		// **وهذه الثلاثةُ هي ما شكا منه المالك.**
		{"سائقٌ ومندوب", []string{"driver", "sales"}, false},
		{"متجرٌ وسائق", []string{"merchant", "driver"}, false},
		{"عملياتٌ وماليّة", []string{"ops", "finance"}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := checkOnePrimary(c.roles)
			if c.ok && err != nil {
				t.Fatalf("%v رُدّت وهي مسموحة: %v", c.roles, err)
			}
			if !c.ok && err == nil {
				t.Fatalf("%v قُبلت — **ومن يُسند لنفسه لا يُكشف في مراجعة**", c.roles)
			}
		})
	}
}

// TestOneRole_GrantRefusesSecondPrimary **ولا يُمنح دورٌ ثانٍ لمن له دور.**
func TestOneRole_GrantRefusesSecondPrimary(t *testing.T) {
	pool := testdb.Pool(t)
	repo := NewRepo(pool)
	ctx := context.Background()
	driver := testdb.NewUser(t, pool, "driver")

	if err := repo.ensureOnePrimary(ctx, driver, "sales"); err == nil {
		t.Fatalf("سائقٌ يُمنح دورَ المندوب — **وهو يُسند لنفسه**")
	}

	// **والزبونُ يُمنح له** — هو الدورُ الذي يجتمع مع كلّ شيء.
	if err := repo.ensureOnePrimary(ctx, driver, "customer"); err != nil {
		t.Fatalf("سائقٌ لا يستطيع أن يكون زبوناً: %v", err)
	}

	// **ومنحُ ما هو ممنوحٌ لا شيء** — ولا يُقرأ تعارضاً مع نفسِه.
	if err := repo.ensureOnePrimary(ctx, driver, "driver"); err != nil {
		t.Fatalf("منحُ الدور نفسِه رُدّ: %v", err)
	}
}

// TestOneRole_EveryRoleBringsCustomer **وكلُّ دورٍ يجلب الزبونَ معه.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٠: «وصاحبُ المنصّة أيضاً، والموظّفون أيضاً».)
//
// **ومن لا يستطيع أن يطلب من منصّته لا يرى ما يراه زبائنُه** — وهو أوّلُ من
// يجب أن يراه.
func TestOneRole_EveryRoleBringsCustomer(t *testing.T) {
	pool := testdb.Pool(t)
	repo := NewRepo(pool)
	ctx := context.Background()

	for _, role := range []string{"driver", "sales", "merchant", "ops", "finance", "admin"} {
		t.Run(role, func(t *testing.T) {
			u := testdb.NewUser(t, pool, "customer")
			// **يُنزع دورُ الزبون أوّلاً** ليُقاس أنّ المنحَ هو من أعاده.
			if err := repo.RevokeRole(ctx, u, "customer"); err != nil {
				t.Fatalf("تعذّر نزعُ دور الزبون: %v", err)
			}
			if err := repo.GrantRole(ctx, u, role, nil); err != nil {
				t.Fatalf("تعذّر منحُ %s: %v", role, err)
			}
			var isCustomer bool
			if err := pool.QueryRow(ctx, `
				SELECT EXISTS (SELECT 1 FROM user_roles
				               WHERE user_id = $1 AND role_code = 'customer')`,
				u).Scan(&isCustomer); err != nil {
				t.Fatalf("تعذّرت القراءة: %v", err)
			}
			if !isCustomer {
				t.Fatalf("%s لم يأخذ دورَ الزبون معه — **فلا يرى منصّتَه كما يراها زبائنُه**", role)
			}
		})
	}
}

// TestMerchantRole_NotGrantableByHand **ولا حسابَ متجرٍ بلا متجر.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٠: «أصلاً لا يمكن إنشاءُ حسابٍ بدون متجر».)
//
// **وصاحبُ متجرٍ بلا متجرٍ يقع على شاشةٍ ميّتة** — رسالةٌ بلا شريطٍ ولا
// خروج. **والحالُ تُبلَغ بضغطةٍ واحدةٍ من لوحة الحسابات.**
//
// **والمسارُ الشرعيُّ لا يمرّ من هنا**: `catalog.CreateMerchant` ينشئ
// المتجرَ وصاحبَه معاً بـ`EnsureUserWithRole` — **فالمنعُ يغلق البابَ
// اليدويَّ وحدَه.**
func TestMerchantRole_NotGrantableByHand(t *testing.T) {
	if err := checkGrantable(RoleMerchant); err == nil {
		t.Fatalf("دورُ المتجر يُمنح بيد — **وصاحبُه يقع على شاشةٍ لا يخرج منها**")
	}
	// **وبقيّةُ الأدوار تُمنح** — المنعُ لهذا وحدَه.
	for _, r := range []string{"driver", "sales", "ops", "finance", "admin", RoleCustomer} {
		if err := checkGrantable(r); err != nil {
			t.Fatalf("دورُ %s مُنع بلا سبب: %v", r, err)
		}
	}
}
