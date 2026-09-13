package qa

import (
	"net/http"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **هويّةُ الرقم عبرَ المسالك الحيّة** (`PID`)
// ══════════════════════════════════════════════════════════════════════
//
// # ولماذا لا تكفي دالّةُ التطبيع
//
// **و`TestNormalizePhone` تقيس الدالّةَ وحدَها** — **وهذه تقيس أنّ
// المسالكَ تناديها**: **مسلكٌ يُنشئ حساباً بنداءٍ مباشرٍ إلى القاعدة
// يتجاوزها بلا أن يصرخ شيء.**
//
// # وما يُقاس
//
//	١ · الأجنبيُّ يُردّ قبل أن يُرسَل رمز
//	٢ · والصيغتان حسابٌ واحد — ولا مكرَّرَ لفرقِ كتابة
//	٣ · وبابُ إنشاء الموظّفين يردّ الأجنبيَّ كذلك
//	٤ · ولا رمزَ يُخزَّن لرقمٍ مردود

// TestPID1_ForeignNumberRejectedBeforeOTP **ولا رمزَ لرقمٍ يُردّ بعده.**
//
// **وشرطُ المالك**: **لا يُرسَل الرمزُ أوّلاً ثمّ يُرفَض** — والتكلفةُ
// والانتظارُ يقعان على من لن يُقبَل.
func TestPID1_ForeignNumberRejectedBeforeOTP(t *testing.T) {
	hh := New(t)
	launchOn(hh, "launch.customer_signup")

	for _, phone := range []string{
		"+905321234567",  // تركيّ
		"+14155552671",   // أمريكيّ
		"+4915112345678", // ألمانيّ
		"+96650123456",   // سعوديّ — وبادئتُه تشبه سوريا
		"0112345678",     // أرضيُّ دمشق
		"093212345",      // ناقصٌ رقماً
		"09321234567",    // زائدٌ رقماً
		"abc",            // مشوَّه
	} {
		for _, path := range []string{
			"/api/v1/auth/signup/request",
			"/api/v1/auth/otp/request",
			"/api/v1/auth/reset/request",
		} {
			r := hh.POST(path, "", map[string]any{"phone": phone})
			if r.Code < 400 {
				t.Errorf("**%s قَبِل %q** (%d) — **ورمزٌ أُرسل إلى رقمٍ لا يُقبَل.**",
					path, phone, r.Code)
			}
		}
		// **ولا أثرَ لرمزٍ في القاعدة** — لا صفَّ تحقّقٍ لرقمٍ مردود.
		var n int
		_ = hh.Pool.QueryRow(ctxBG(),
			`SELECT count(*) FROM otp_codes WHERE phone::text LIKE '%' || $1 || '%'`,
			phone).Scan(&n)
		if n > 0 {
			t.Errorf("**رمزٌ خُزّن لرقمٍ مردود**: %q ⇒ %d صفّاً", phone, n)
		}
	}
}

// TestPID2_OneIdentityForBothForms **وصيغتان حسابٌ واحد.**
//
// **وشرطُ المالك**: **لا حسابٌ مكرَّرٌ لفرقٍ في الكتابة وحدَه.**
func TestPID2_OneIdentityForBothForms(t *testing.T) {
	hh := New(t)
	u := hh.NewUser("customer")

	var stored string
	if err := hh.Pool.QueryRow(ctxBG(),
		`SELECT phone::text FROM users WHERE id = $1`, u.ID).Scan(&stored); err != nil {
		t.Fatalf("قراءةُ الرقم: %v", err)
	}
	// **والمخزَّنُ بالصيغة الواحدة** — `+9639XXXXXXXX`.
	if len(stored) != 13 || stored[:5] != "+9639" {
		t.Fatalf("**المخزَّنُ ليس بالصيغة الكانونيّة**: %q", stored)
	}
	local := "0" + stored[4:] // 09XXXXXXXX

	// **والصيغتان تجدان الحسابَ نفسَه** — يُقاس ببابِ الإدارة.
	_, admin := roleUser(t, hh, "admin")
	for _, form := range []string{stored, local, "00963" + stored[4:], stored[1:]} {
		r := hh.POST("/api/v1/admin/users", admin, map[string]any{
			"phone": form, "full_name": "قياسُ التكرار", "password": "Pid#2026Test",
		})
		// **و٤٠٩ «قائمٌ» هو الجواب الصحيح** — **و٢٠١ يعني حساباً ثانياً.**
		if r.Code == http.StatusCreated {
			t.Errorf("**حسابٌ ثانٍ أُنشئ بصيغةٍ أخرى للرقم نفسِه**: %q", form)
		}
	}

	var count int
	_ = hh.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM users WHERE phone = $1::citext`, stored).Scan(&count)
	if count != 1 {
		t.Errorf("**%d حساباً لرقمٍ واحد** — والهويّةُ واحدة.", count)
	}
}

// TestPID3_StaffCreationRejectsForeign **وبابُ الموظّفين كذلك.**
//
// **ولا بابَ يُستثنى**: **من أنشأ موظّفاً برقمٍ أجنبيٍّ أدخل هويّةً لا
// تصلها رسالةٌ ولا واتساب.**
func TestPID3_StaffCreationRejectsForeign(t *testing.T) {
	hh := New(t)
	_, admin := roleUser(t, hh, "admin")

	for _, phone := range []string{"+905321234567", "+14155552671", "0112345678"} {
		r := hh.POST("/api/v1/admin/users", admin, map[string]any{
			"phone": phone, "full_name": "أجنبيّ", "password": "Pid#2026Test",
		})
		if r.Code < 400 {
			t.Errorf("**موظّفٌ أُنشئ برقمٍ لا يُقبَل**: %q (%d)", phone, r.Code)
		}
	}
}

// TestPID4_SyrianMobileIsAccepted **وما يُقبَل يُقبَل فعلاً.**
//
// **ومنعٌ يمنع الصحيحَ أسوأُ من لا منع** — فيُقاس القبولُ لا الردُّ وحدَه.
func TestPID4_SyrianMobileIsAccepted(t *testing.T) {
	hh := New(t)
	_, admin := roleUser(t, hh, "admin")

	// **وأرقامٌ من مولّد المعمل** — فلا تتزاحم جولتان على رقمٍ واحد.
	base := hh.Factory().NS.Phone() // +9639XXXXXXXX
	local := "0" + base[4:]
	for i, phone := range []string{
		local,
		base,
		"00963" + base[4:],
		base[1:],
	} {
		r := hh.POST("/api/v1/admin/users", admin, map[string]any{
			"phone": phone, "full_name": "سوريٌّ صالح", "password": "Pid#2026Test",
		})
		if r.Code >= 400 && r.Err() == "invalid_phone" {
			t.Errorf("**رقمٌ سوريٌّ صالحٌ رُدّ** [%d]: %q ⇒ %s", i, phone, r.Err())
		}
	}
}
