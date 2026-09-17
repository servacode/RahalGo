package qa

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// ══════════════════════════════════════════════════════════════════════
// **إيقافُ المنصّة يُسأل عنه** (`PCA`، ٢٠٢٦-٠٩-١٧)
// ══════════════════════════════════════════════════════════════════════
//
// # العطبُ المقيس
//
// **و`PUT /admin/platform/closure` يمنع الطلبَ عن البلد كلِّه** —
// **ولم يكن يكتب سطراً في سجلّ الأحداث.**
//
// **و`service_closure` صفٌّ واحدٌ يُكتب فوقه**: **`updated_by` يحفظ
// آخرَ فاعلٍ لا تاريخَ الأفعال.** **فمن أوقف المنصّةَ أمسِ ثمّ أعادها
// غيرُه اليومَ ذهب أوّلُهما بلا أثر** — **ولا يُعرَف أنّه كان.**
//
// **وأخوهُ `admin.zone_hours_set` مسجَّلٌ منذ زمن** — فكان الغيابُ
// سهواً لا قصدا.
//
// **والسجلُّ يُلحَق ولا يُكتب فوقه** — **وهذا ما يفصل «من فعل» عن
// «ما الحالُ الآن».**

// closureVia **يضبط الإيقافَ من باب الأدمن المعتمَد** — لا بيدٍ في القاعدة.
func closureVia(t *testing.T, hh *Harness, tok string, active bool, msg string) {
	t.Helper()
	r := hh.Call("PUT", "/api/v1/admin/platform/closure", tok,
		map[string]any{"active": active, "message": msg}, nil)
	if r.Code != http.StatusOK {
		t.Fatalf("**بابُ الإيقاف ردّ %d %s**", r.Code, r.Err())
	}
}

// closureAudits **عدُّ سطور الإيقاف في السجلّ.**
func closureAudits(t *testing.T, hh *Harness) int {
	t.Helper()
	var n int
	if err := hh.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM audit_log WHERE action = 'admin.platform_closure'`).Scan(&n); err != nil {
		t.Fatalf("قراءةُ السجلّ: %v", err)
	}
	return n
}

// waitClosureAudits **ينتظر بلوغَ العدد المنتظَر** — **والكتابةُ في
// السجلّ تقع خارجَ نداء الطلب** (`s.audit` تُطلق خيطاً)، **وهي عادةُ
// المشروع لكلّ فعلٍ بلا معاملة** — وأخوهُ `admin.zone_hours_set` كذلك.
//
// **والانتظارُ لا يستر سقوطاً**: **من لم يكتب لا يبلغ العددَ أبداً**،
// **وتنتهي المهلةُ فيحمرّ الفحص.**
func waitClosureAudits(t *testing.T, hh *Harness, want int) int {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	got := closureAudits(t, hh)
	for got < want && time.Now().Before(deadline) {
		time.Sleep(60 * time.Millisecond)
		got = closureAudits(t, hh)
	}
	return got
}

// TestPCA1_ClosingAndReopeningAreBothRecorded **الإغلاقُ والفتحُ كلاهما
// يُكتب** — **ولا يمحو الثاني الأوّل.**
func TestPCA1_ClosingAndReopeningAreBothRecorded(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	tok := ownerToken(t, hh)
	// **ويُعاد الحالُ مهما سقط الفحص** — وصفُّ الإيقاف مشتركٌ للمنصّة.
	t.Cleanup(func() { closure(t, hh, false, "", nil) })

	before := closureAudits(t, hh)

	closureVia(t, hh, tok, true, "صيانةٌ مؤقّتة")
	afterClose := waitClosureAudits(t, hh, before+1)
	if afterClose != before+1 {
		t.Fatalf("**إيقافُ المنصّة لم يُسجَّل**: %d ⇐ %d — "+
			"**ومن منع الطلبَ عن البلد كلِّه لا يُعرَف.**", before, afterClose)
	}

	closureVia(t, hh, tok, false, "")
	afterOpen := waitClosureAudits(t, hh, afterClose+1)
	if afterOpen != afterClose+1 {
		t.Errorf("**إعادةُ الفتح لم تُسجَّل**: %d ⇐ %d", afterClose, afterOpen)
	}

	// **والأوّلُ باقٍ بعد الثاني** — **والسجلُّ يُلحَق ولا يُكتب فوقه.**
	var closes int
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM audit_log
		 WHERE action = 'admin.platform_closure'
		   AND details->>'active' = 'true'`).Scan(&closes); err != nil {
		t.Fatalf("قراءةُ تفصيل السجلّ: %v", err)
	}
	if closes == 0 {
		t.Error("**ذهب سطرُ الإغلاق بعد سطر الفتح** — " +
			"**وصفٌّ يُكتب فوقه ليس سجلّا.**")
	}
}

// TestPCA2_ActorAndShapeAreTyped **والفاعلُ باسمه والهدفُ بنوعه.**
func TestPCA2_ActorAndShapeAreTyped(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	tok := ownerToken(t, hh)
	t.Cleanup(func() { closure(t, hh, false, "", nil) })

	closureVia(t, hh, tok, true, "صيانةٌ مؤقّتة")
	if waitClosureAudits(t, hh, 1) == 0 {
		t.Fatal("**لا سطرَ إيقافٍ في السجلّ**")
	}

	var entity string
	var actor *string
	var details string
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT entity, actor_user_id::text, details::text
		  FROM audit_log
		 WHERE action = 'admin.platform_closure'
		 ORDER BY created_at DESC LIMIT 1`).Scan(&entity, &actor, &details); err != nil {
		t.Fatalf("قراءةُ آخر سطر: %v", err)
	}
	// **والنوعُ كأخيه `admin.zone_hours_set`** — لا تسميةٌ ثانية.
	if entity != "platform" {
		t.Errorf("**نوعُ الهدف %q لا `platform`**", entity)
	}
	if actor == nil || *actor == "" {
		t.Error("**سطرٌ بلا فاعل** — **ومن فعلها هو السؤال كلُّه.**")
	}
	// **ولا سرَّ في التفصيل** — الرسالةُ إعلانٌ للناس لا كلمةَ مرور.
	for _, bad := range []string{"password", "pin", "token", "secret", "otp"} {
		if strings.Contains(strings.ToLower(details), bad) {
			t.Errorf("**تفصيلُ السجلّ يذكر %q**", bad)
		}
	}
}
