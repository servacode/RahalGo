// عيّنةُ نقلٍ إلى المصنع — **خمسةٌ لا خمسُمئة.**
//
// (البند ٢١ من طلب المالك: **إثباتُ أنّ الواجهةَ صالحةٌ للاستعمال، لا
// مشروعُ نقلٍ ضخم.**)
//
// **وما تختبره لم يتبدّل** — **التركيبُ وحدَه تبدّل.** **والأصلُ يبقى في
// موضعه** حتّى يُقرَّر النقلُ الكامل في مرحلةٍ لاحقة.
package qa

import (
	"net/http"
	"testing"
	"time"
)

// ══════════════════════════════════════════════════════════════════════
// **١ · هويّةٌ — كان `h.Customer()`**
// ══════════════════════════════════════════════════════════════════════
//
// **والفرقُ أنّ الهاتفَ صار حتميّاً** — **فسقوطٌ يُعاد بالأرقام نفسِها.**

func TestSampleFactory_ValidToken(t *testing.T) {
	h := New(t)
	u := h.Factory().NewUserWith("customer")

	got := h.GET("/api/v1/auth/me", u.Token)
	if got.Code != http.StatusOK {
		t.Fatalf("توكنٌ صحيح: يُنتظر 200 ووقع %s", got)
	}
	if id, _ := got.JSON()["id"].(string); id != u.ID {
		t.Errorf("/auth/me أعاد حساباً آخر: %v ≠ %s", got.JSON()["id"], u.ID)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٢ · حالُ حسابٍ — لم يكن ممكناً بالمِسنَد وحدَه**
// ══════════════════════════════════════════════════════════════════════
//
// **والمِسنَدُ ينشئ نشطاً وحدَه** — **فمن أراد موقوفاً كتب `UPDATE` بيده
// في اختباره.** **وهذا ما يمحوه المصنع.**

func TestSampleFactory_SuspendedIsRefused(t *testing.T) {
	h := New(t)
	u := h.Factory().NewUserWith("customer", Suspended())

	got := h.GET("/api/v1/auth/me", u.Token)
	if got.Code == http.StatusOK {
		t.Fatalf("حسابٌ موقوفٌ مرّ — يُنتظر ردٌّ ووقع %s", got)
	}
	t.Logf("الموقوفُ رُدَّ بـ%d — `middleware.go:40`", got.Code)
}

// ══════════════════════════════════════════════════════════════════════
// **٣ · سائقٌ بحالٍ مركّبة**
// ══════════════════════════════════════════════════════════════════════
//
// **ثلاثةُ خيارٍ سطراً واحداً** — **وكان يحتاج ثلاثةَ استعلاماتٍ يدويّة.**

func TestSampleFactory_DriverOnShiftWithCash(t *testing.T) {
	h := New(t)
	f := h.Factory()
	d := f.Driver(OnShift(), CashHeld(120_000),
		LocationAt(35.95, 39.01, f.CK.Ago(30*time.Second)))

	got := h.GET("/api/v1/driver/me", d.Token)
	if got.Code != http.StatusOK {
		t.Fatalf("بابُ السائق: يُنتظر 200 ووقع %s", got)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٤ · تخويلٌ — كان `h.Customer(), h.Customer()`**
// ══════════════════════════════════════════════════════════════════════
//
// **ونطاقان مختلفان يضمنان ألّا يتشابه الهاتفان** — **وهو ما كان يعتمد
// على الوقت.**

func TestSampleFactory_ForeignRoleDenied(t *testing.T) {
	h := New(t)
	f := h.Factory()
	cust := f.NewUserWith("customer")

	got := h.GET("/api/v1/driver/me", cust.Token)
	if got.Code == http.StatusOK {
		t.Fatalf("زبونٌ فتح بابَ سائق — يُنتظر ردٌّ ووقع %s", got)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٥ · مالٌ متّسق — شرطٌ سابقٌ لـ`P-4`**
// ══════════════════════════════════════════════════════════════════════
//
// **والمصنعُ يقيّد بالدفتر فيتبعه الرصيد** — **فلا فكسچرَ ماليٌّ مكسورٌ
// يدخل اختباراً ماليّاً.**

func TestSampleFactory_WalletMatchesLedger(t *testing.T) {
	h := New(t)
	f := h.Factory()
	u := f.NewUserWith("customer")
	f.Credit(u.ID, 40_000, "topup")

	got := h.GET("/api/v1/my/wallet", u.Token)
	if got.Code != http.StatusOK {
		t.Fatalf("بابُ المحفظة: يُنتظر 200 ووقع %s", got)
	}
	if f.Balance(u.ID) != f.LedgerSum(u.ID) {
		t.Fatalf("الرصيدُ %d ودفترُه %d", f.Balance(u.ID), f.LedgerSum(u.ID))
	}
}
