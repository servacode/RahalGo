package qa

import (
	"context"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **عمليّةُ الإدارةِ تقع كلُّها أو لا تقع** — `PF-01` · `PF-02` · `PF-03`
// ══════════════════════════════════════════════════════════════════════
//
// # السببُ الواحد
//
// **ثلاثُ عمليّاتٍ إداريّةٍ تكتب مرّاتٍ متتاليةً على `s.pg` مباشرةً بلا
// معاملة** — **وبعضُ أخطائها مُهمَلٌ بـ`_, _ =`.** فإن سقطت الكتابةُ
// الثالثةُ بقيت الأولى والثانية.
//
// **ولا يراه زبونٌ ولا إدارة**: الردُّ 500 بلا بيان، **والقاعدةُ فيها
// حالٌ لا يصفها أيُّ عقد.**
//
// # والفرقُ بين هذه وبين حرّاسِ `TestFAIL_*`
//
// **تلك توثّق العطبَ بالسجلّ وتنجح وهو قائم.** **وهذه تسقط ما دام
// قائماً** — **فلا يُغلَق العيبُ بقراءةِ شيفرة.**

// countRows عدّادٌ صغيرٌ يُقرأ بعد الحقن.
func countRows(t *testing.T, h *Harness, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := h.Pool.QueryRow(ctxBG(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("العدّ: %v", err)
	}
	return n
}

// ══════════════════════════════════════════════════════════════════════
// **`PF-02` — مصروفٌ بلا خصمِ خزينة**
// ══════════════════════════════════════════════════════════════════════
//
// **وتقريرُ الأرباح يقول ربحاً لم يقع.**
func TestATOMIC_ExpenseAndTreasuryAreOneUnit(t *testing.T) {
	h := New(t)
	treasury(t, h)
	admin := h.NewUser("admin")

	var catID string
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO expense_categories (name, sort_order, active)
		VALUES ($1, 1, true) RETURNING id::text`,
		uniq("بندُ ذرّيّةٍ ")).Scan(&catID); err != nil {
		t.Fatalf("بندُ المصروف: %v", err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = h.Pool.Exec(ctx, `DELETE FROM expenses WHERE category_id = $1::uuid`, catID)
		_, _ = h.Pool.Exec(ctx, `DELETE FROM expense_categories WHERE id = $1::uuid`, catID)
	})

	// **يُسقَط خصمُ الخزينة** — والمصروفُ كُتب قبلَه.
	fp := h.Arm("D5/step-2-treasury-debit", "wallet_transactions", "INSERT",
		1, "kind", "operating_expense")
	got := h.POST("/api/v1/admin/expenses", admin.Token,
		map[string]any{"category_id": catID, "amount": 33_000, "note": "PF-02"})
	fp.MustFire(t)

	expenses := countRows(t, h,
		`SELECT count(*) FROM expenses WHERE category_id = $1::uuid`, catID)
	t.Logf("الردُّ %d · مصاريفُ باقيةٌ %d", got.Code, expenses)

	if expenses != 0 {
		t.Errorf("**مصروفٌ بقي بلا خصمِ خزينة** (%d صفّاً) — "+
			"**والعمليّةُ تقع كلُّها أو لا تقع.**", expenses)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`PF-01` — متجرٌ ومكافأةٌ ومرشَّحٌ ما يزال `new`**
// ══════════════════════════════════════════════════════════════════════
//
// **وأخطرُها**: **الإعادةُ تُنشئ متجراً ثانياً** — `RETRY SAFETY = BROKEN`.
func TestATOMIC_LeadConversionIsOneUnit(t *testing.T) {
	h := New(t)
	treasury(t, h)
	admin := h.NewUser("admin")

	leadID, phone := seedLeadForConversion(t, h)

	// **يُسقَط التثبيتُ الأخير** — والمتجرُ والمكافأةُ سبقاه.
	fp := h.Arm("D2/step-6-commit-conversion", "merchant_leads", "UPDATE",
		1, "status", "converted")
	got := h.POST("/api/v1/admin/leads/"+leadID+"/status", admin.Token,
		map[string]any{"status": "converted", "note": "ذرّيّة"})
	fp.MustFire(t)

	merchants := countRows(t, h,
		`SELECT count(*) FROM merchants WHERE phone = $1`, phone)
	var status string
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT status FROM merchant_leads WHERE id = $1::uuid`, leadID).Scan(&status)
	t.Logf("الردُّ %d · متاجرُ %d · حالُ المرشَّح %q", got.Code, merchants, status)

	if merchants != 0 {
		t.Errorf("**متجرٌ أُنشئ والتحويلُ لم يثبت** (%d) — "+
			"**والإعادةُ تُنشئ ثانياً.**", merchants)
	}
	if status == "converted" {
		t.Error("**المرشَّحُ صار محوَّلاً والتثبيتُ سقط**")
	}
}

// seedLeadForConversion مرشَّحٌ جاهزٌ للتحويل — بتصنيفٍ ونقطةٍ وهاتف.
func seedLeadForConversion(t *testing.T, h *Harness) (leadID, phone string) {
	t.Helper()
	f := h.Factory()
	catID := f.category()
	phone = uniqPhone()
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO merchant_leads
		       (store_name, owner_name, phone, area, category_id, lat, lng, status)
		VALUES ($1, $2, $3, 'الرقّة', $4::uuid, 35.95, 39.01, 'new')
		RETURNING id::text`,
		uniq("متجرُ ذرّيّةٍ "), "صاحبُه", phone, catID).Scan(&leadID); err != nil {
		t.Skipf("تعذّر تجهيزُ مرشَّح: %v", err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = h.Pool.Exec(ctx, `DELETE FROM merchant_leads WHERE id = $1::uuid`, leadID)
	})
	return leadID, phone
}

// ══════════════════════════════════════════════════════════════════════
// **`PF-03` — مستخدِمٌ بلا دورٍ يقفل حسابَه**
// ══════════════════════════════════════════════════════════════════════
//
// **والتعافي مسدود**: **الإعادةُ تردّ `phone_taken`** — **فلا يُنشأ ولا
// يُصلَح.**
func TestATOMIC_AdminUserCreationIsOneUnit(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	phone := uniqPhone()

	// **تُسقَط كتابةُ الكلمة** — والمستخدِمُ ودورُه سبقاها.
	//
	// **وهي الحالةُ الأدقّ**: **`CreateUserWithRole` ذرّيّةٌ وحدَها**
	// (قِيس: سقوطُها يَردّ الإنشاءَ كلَّه)، **والانكسارُ فيما بعدها** —
	// `GrantRole` ثمّ `SetTempPassword` **كلٌّ في عمليّةٍ على حدة.**
	//
	// **فيبقى مستخدِمٌ بلا كلمة**: **لا يدخل، ورقمُه محجوزٌ فلا يُنشأ
	// ثانيةً** — `phone_taken`. **والتعافي مسدود.**
	fp := h.Arm("D15/set-temp-password", "users", "UPDATE", 1, "must_change_password", "true")
	got := h.POST("/api/v1/admin/users", admin.Token, map[string]any{
		// **والدورُ عارضٌ في هذا الفحص لا مقصود** — يقيس ذرّيّةَ
		// الإنشاء لا سياسةَ الأدوار. **و`ops` صار إرثاً لا يُمنَح**
		// (٢٠٢٦-٠٩-١٢)، **فبقاؤه يُسقط الفحصَ بسببٍ ليس سببَه.**
		"phone": phone, "full_name": "ذرّيّةُ الإنشاء", "roles": []string{"driver"},
		"password": "Qwerty!2345",
	})
	t.Logf("ردُّ الإنشاء: %d %s", got.Code, got.Err())
	fp.MustFire(t)

	users := countRows(t, h, `SELECT count(*) FROM users WHERE phone = $1`, phone)
	roles := countRows(t, h, `
		SELECT count(*) FROM user_roles r JOIN users u ON u.id = r.user_id
		 WHERE u.phone = $1`, phone)
	t.Logf("الردُّ %d · مستخدِمون باقون %d · أدوارُهم %d", got.Code, users, roles)

	if users != 0 {
		t.Errorf("**مستخدِمٌ بقي بلا كلمة** (%d مستخدِماً · %d دوراً) — "+
			"**لا يدخل، ورقمُه محجوزٌ فلا يُنشأ ثانيةً.**", users, roles)
	}
}
