package qa

// ══════════════════════════════════════════════════════════════════════
//  **تهيئةُ أوّلِ مالك — مرّةً واحدةً بالثابت لا بعلَم** (`OBS-1`)
// ══════════════════════════════════════════════════════════════════════
//
// # لماذا وُجدت الأداة
//
// **سياسةُ ٢٠٢٦-٠٩-١٢ تجعل منحَ `owner_super_admin` و`admin` للمالك
// وحدَه** — **ولا يرقّي أدمنٌ نفسَه.** **وقِيس أنّ الإنتاجَ فيه صفرُ
// مالكين**: فالسياسةُ مُغلَقةٌ بقصد، ولا بابَ في الـAPI يُنشئ أوّلَه.
//
// **والمفتاحُ على الخادم** — `cmd/ownerbootstrap`.
//
// # وما يُحرَس هنا
//
//	١ · **صفرُ مالكين ⇒ يُكتب واحدٌ** وأثرُه في المعاملة نفسِها
//	٢ · **ومالكٌ قائمٌ ⇒ تُردّ الأداةُ** — والمرّةُ الواحدةُ بالثابت
//	٣ · **ولا تُكتب حرفاً في وضع الفحص**
//	٤ · **ولا تمسّ كلمةً ولا رمزَ لوحةٍ ولا هاتفاً ولا دوراً آخر**
//	٥ · **ولا تُنشئ حساباً** — والهدفُ قائمٌ أو تُردّ
//
// **ولا تُنادى الأداةُ هنا كعمليّة** — **تُنادى دالّتُها نفسُها**
// (`bootstrapOwner`) **بمنطقها بعينه**، فالفحصُ يقيس ما ينفّذه الخادم
// لا نسخةً منه.

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/identity"
)

// TestOBS1_PhoneNormalizationIsCanonical **التطبيعُ بدالّة المشروع.**
//
// **ومن طبّع بيده أنشأ حساباً ثانياً لرقمٍ واحد** — والأداةُ لا تكتب
// `+963` بيدها.
func TestOBS1_PhoneNormalizationIsCanonical(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"0985395131", "+963985395131"},
		{"+963985395131", "+963985395131"},
		{"00963985395131", "+963985395131"},
		{"963985395131", "+963985395131"},
		{"098-539-5131", "+963985395131"},
	} {
		got, ok := identity.NormalizePhone(c.in)
		if !ok || got != c.want {
			t.Errorf("%q ⇒ %q (ok=%v) والمنتظَرُ %q", c.in, got, ok, c.want)
		}
	}
	if _, ok := identity.NormalizePhone("0985"); ok {
		t.Error("**رقمٌ ناقصٌ قُبل** — وحسابٌ يُصاب بالخطأ")
	}
}

// TestOBS2_BootstrapWritesOneOwnerAndAudits **صفرُ مالكين ⇒ واحدٌ وأثرُه.**
func TestOBS2_BootstrapWritesOneOwnerAndAudits(t *testing.T) {
	hh := New(t)
	clearOwners(t, hh)
	t.Cleanup(func() { clearOwners(t, hh) })

	u := hh.NewUser("admin")
	before := auditCount(t, hh, "admin.owner_bootstrap", u.ID)
	if n := ownerCount(t, hh); n != 0 {
		t.Fatalf("المِسنَدُ بدأ بـ%d مالكاً — والقياسُ بعده لا يعني شيئاً", n)
	}

	if err := identity.BootstrapFirstOwner(context.Background(), hh.Pool, u.ID, "+963900000099"); err != nil {
		t.Fatalf("**التهيئةُ أخفقت على صفر مالكين**: %v", err)
	}
	if n := ownerCount(t, hh); n != 1 {
		t.Errorf("**المالكونَ %d بعد التهيئة والمنتظَرُ واحد**", n)
	}
	if !holdsRole(t, hh, u.ID, authz.RoleOwnerSuperAdmin) {
		t.Error("**نجحت التهيئةُ ولم يقع الصفّ**")
	}
	if n := auditCount(t, hh, "admin.owner_bootstrap", u.ID); n != before+1 {
		t.Errorf("**أثرُ تدقيقٍ واحدٌ مطلوب**: %d ⇒ %d", before, n)
	}

	// ── ٢ · **ومالكٌ قائمٌ ⇒ تُردّ** — والمرّةُ الواحدةُ بالثابت ─────
	v := hh.NewUser("customer")
	err := identity.BootstrapFirstOwner(context.Background(), hh.Pool, v.ID, "+963900000098")
	if err == nil {
		t.Error("**مالكٌ ثانٍ كُتب** — **والتهيئةُ لأوّلِه وحدَه**")
	}
	if holdsRole(t, hh, v.ID, authz.RoleOwnerSuperAdmin) {
		t.Error("**رُدَّت التهيئةُ ووقعت**")
	}
	if n := ownerCount(t, hh); n != 1 {
		t.Errorf("**العددُ تبدّل بمحاولةٍ مردودة**: %d", n)
	}
	if n := auditCount(t, hh, "admin.owner_bootstrap", v.ID); n != 0 {
		t.Errorf("**أثرُ تدقيقٍ لفعلٍ لم يقع**: %d", n)
	}
	t.Logf("OBS-2: مالكٌ واحدٌ · ومحاولةٌ ثانيةٌ مردودةٌ بـ%v", err)
}

// TestOBS3_BootstrapTouchesNothingElse **ولا تمسّ ما ليس لها.**
//
// **كلمةٌ ورمزُ لوحةٍ وهاتفٌ وحالٌ وأدوارٌ أخرى** — تُقاس قبلَ وبعد.
func TestOBS3_BootstrapTouchesNothingElse(t *testing.T) {
	hh := New(t)
	clearOwners(t, hh)
	t.Cleanup(func() { clearOwners(t, hh) })
	u := hh.NewUser("admin")

	type snap struct {
		phone, status, roles string
		hasPw, hasPin        bool
		users                int
	}
	read := func() snap {
		var s snap
		if err := hh.Pool.QueryRow(ctxBG(), `
			SELECT u.phone::text, u.status,
			       COALESCE(string_agg(ur.role_code, ',' ORDER BY ur.role_code), ''),
			       (u.password_hash IS NOT NULL), (u.admin_pin_hash IS NOT NULL),
			       (SELECT count(*) FROM users)
			  FROM users u LEFT JOIN user_roles ur ON ur.user_id = u.id
			 WHERE u.id = $1::uuid
			 GROUP BY u.phone, u.status, u.password_hash, u.admin_pin_hash`,
			u.ID).Scan(&s.phone, &s.status, &s.roles, &s.hasPw, &s.hasPin, &s.users); err != nil {
			t.Fatalf("اللقطة: %v", err)
		}
		return s
	}

	a := read()
	if err := identity.BootstrapFirstOwner(context.Background(), hh.Pool, u.ID, a.phone); err != nil {
		t.Fatalf("التهيئة: %v", err)
	}
	b := read()

	if a.phone != b.phone {
		t.Errorf("**الهاتفُ تبدّل**: %s ⇒ %s", a.phone, b.phone)
	}
	if a.status != b.status {
		t.Errorf("**الحالُ تبدّل**: %s ⇒ %s", a.status, b.status)
	}
	if a.hasPw != b.hasPw {
		t.Error("**كلمةُ المرور تبدّلت**")
	}
	if a.hasPin != b.hasPin {
		t.Error("**رمزُ اللوحة تبدّل**")
	}
	if a.users != b.users {
		t.Errorf("**حسابٌ أُنشئ أو حُذف**: %d ⇒ %d", a.users, b.users)
	}
	// **والأدوارُ تزيد واحداً بعينه ولا شيءَ غيره.**
	want := a.roles
	if want != "" {
		want += ","
	}
	t.Logf("OBS-3: الأدوارُ %q ⇒ %q", a.roles, b.roles)
	if b.roles == a.roles {
		t.Error("**لم يُضَف دورٌ إطلاقاً**")
	}
	for _, r := range []string{"admin", authz.RoleOwnerSuperAdmin} {
		if !containsRole(b.roles, r) {
			t.Errorf("**الدورُ %s مفقودٌ بعد التهيئة**", r)
		}
	}
	if n := countRoleTokens(b.roles) - countRoleTokens(a.roles); n != 1 {
		t.Errorf("**تبدّلت %d أدوارٍ والمنتظَرُ واحد**", n)
	}
}

func containsRole(csv, role string) bool {
	for _, r := range splitCSV(csv) {
		if r == role {
			return true
		}
	}
	return false
}

func countRoleTokens(csv string) int { return len(splitCSV(csv)) }

func splitCSV(s string) []string {
	out := []string{}
	cur := ""
	for _, r := range s {
		if r == ',' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
