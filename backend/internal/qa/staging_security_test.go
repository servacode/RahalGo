package qa

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **أمنُ التكامل على بيئة التجهيز — `P-0` البنود ٣٠ · ٣١ · ٣٢ · ٣٣**
// ══════════════════════════════════════════════════════════════════════
//
// **وهذه لا تعمل إلّا بـ`RAHALGO_STAGING=1` وذاكرةٍ حقيقيّة** —
// **وتتخطّى بهدوءٍ في غيرها**، فلا تُحمَّر الحزمةُ لسببٍ بيئيّ.

// stagingOnly يتخطّى ما لم تكن بيئةُ التجهيز مهيّأة.
func stagingOnly(t *testing.T) {
	t.Helper()
	if os.Getenv("RAHALGO_STAGING") != "1" {
		t.Skip("اختبارُ تجهيزٍ — يحتاج RAHALGO_STAGING=1")
	}
	if os.Getenv("QA_REAL_REDIS_ADDR") == "" {
		t.Skip("يحتاج QA_REAL_REDIS_ADDR — و`miniredis` لا تُسقَط")
	}
}

// redisContainer اسمُ حاويةِ ذاكرة التجهيز — **من البيئة لا مكتوباً.**
func redisContainer() string {
	if c := os.Getenv("STAGING_REDIS_CONTAINER"); c != "" {
		return c
	}
	return "rahalgo-staging-redis"
}

// dockerRedis يوقف حاويةَ الذاكرة أو يُشعلها.
//
// **ويُشعِلها دائماً في التنظيف** — **واختبارٌ يترك الذاكرةَ ساقطةً
// يُسقط كلَّ ما بعده ويبدو عطباً في المنتج.**
func dockerRedis(t *testing.T, action string) {
	t.Helper()
	cmd := exec.Command("docker", action, redisContainer())
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("تعذّر %s على %s: %v · %s", action, redisContainer(), err, out)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **وحُذف `TestFAIL_R16_RedisDownFailsOpen`** — دورةُ إصلاحٍ ١٦
// ══════════════════════════════════════════════════════════════════════
//
// **كان يوثّق العيبَ وينجح ما دام قائماً**: جلسةٌ مُبطَلةٌ تُرفَض
// والذاكرةُ قائمة (401)، **ثمّ تُقبَل حين تسقط (200).**
//
// **وقد أُصلح** — **فصار يسقط**، وهو ما وعد به نصُّه: «يُقرأ سقوطُه
// أمراً بتحديث السجلّ لا عطباً».
//
// **وخلَفُه يقيس العقدَ الجديد**: `TestR16_STG_RedisOutageAuthorityHolds`
// في `session_authority_staging_test.go` — **بذاكرةٍ حقيقيّةٍ تُوقَف
// وتعود**، **ويزيد عليه ما لم يكن يُقاس**: **إبطالٌ يقع والذاكرةُ
// ساقطة، ثمّ تعود نظيفةً — والجلسةُ تبقى مرفوضة.**

// ══════════════════════════════════════════════════════════════════════
// **`R14` — جانبُ الخادم** (البند ٣١)
// ══════════════════════════════════════════════════════════════════════

func TestStaging_R14_SessionLifecycleServerSide(t *testing.T) {
	stagingOnly(t)
	h := New(t)
	user := h.NewUser("customer")

	// **وحسابٌ يُوقَف يُرفَض فوراً** — ولو بقي توكنُه صالحَ التوقيع.
	if res := h.GET("/api/v1/auth/me", user.Token); res.Code != http.StatusOK {
		t.Fatalf("توكنٌ جديدٌ لا يعمل — %d", res.Code)
	}
	if _, err := h.Pool.Exec(context.Background(),
		`UPDATE users SET status = 'suspended' WHERE id = $1::uuid`, user.ID); err != nil {
		t.Fatal(err)
	}
	// **والحالُ مخزَّنةٌ ثلاثين ثانية** — فيُمسَح المفتاحُ كما يمسحه المحرّك.
	h.Redis().Del(context.Background(), "ustatus:"+user.ID)

	if res := h.GET("/api/v1/auth/me", user.Token); res.Code != http.StatusForbidden {
		t.Errorf("حسابٌ موقوفٌ ما زال يعمل — %d", res.Code)
	} else {
		t.Log("✓ الإيقافُ يسري على توكنٍ قائم")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **خطُّ أساسٍ أمنيّ** (البند ٣٢)
// ══════════════════════════════════════════════════════════════════════

func TestStaging_SecurityBaseline(t *testing.T) {
	stagingOnly(t)
	h := New(t)
	cust := h.NewUser("customer")

	cases := []struct {
		name string
		path string
		tok  string
		want []int
	}{
		{"بلا توكن", "/api/v1/admin/users", "", []int{401}},
		{"دورٌ خاطئ", "/api/v1/admin/users", cust.Token, []int{403}},
		{"توكنٌ منتهٍ", "/api/v1/auth/me",
			h.ExpiredToken(cust.ID, "customer"), []int{401}},
		{"توكنٌ ملفَّق", "/api/v1/auth/me", "not-a-token", []int{401}},
		{"إدارةٌ بلا توكن", "/api/v1/admin/ops-map/meta", "", []int{401}},
		{"بثٌّ بلا توكن", "/api/v1/ws", "", []int{401, 400, 426}},
	}
	pass := 0
	for _, c := range cases {
		res := h.GET(c.path, c.tok)
		ok := false
		for _, w := range c.want {
			if res.Code == w {
				ok = true
			}
		}
		if !ok {
			t.Errorf("%s: الردّ %d وأُريد %v", c.name, res.Code, c.want)
			continue
		}
		pass++
	}
	t.Logf("خطُّ الأساس الأمنيّ = %d/%d", pass, len(cases))
}

// ══════════════════════════════════════════════════════════════════════
// **`D13` — سردُ الوسائط** (البند ٣٣)
// ══════════════════════════════════════════════════════════════════════
//
// **ولا يُصلَح** (البند ٥٢) — **يُسجَّل دليلُه.**
func TestFAIL_D13_MediaDirectoryListingOpen(t *testing.T) {
	stagingOnly(t)
	h := New(t)

	// ══════════════════════════════════════════════════════════════
	// **وانقلب هذا الرصد** — `D13` أُغلقت في دورةِ إصلاحٍ ١٠
	// ══════════════════════════════════════════════════════════════
	//
	// **كان يوثّق أنّ السردَ مفتوحٌ وينجح ما دام كذلك.** **وصار السردُ
	// مغلقاً مطلقاً والشخصيُّ لا يُخدَم إلّا برابطٍ موقَّع** — **فصار
	// يحرس الإغلاقَ لا يوثّق الفتح.**
	//
	// **وبُدّل لأنّ العقدَ تبدّل، لا ليمرّ.**
	listing := h.GET("/media/", "")
	anon := h.GET("/media/does-not-exist.jpg", "")
	t.Logf("D13 على التجهيز — سردُ /media/ = %d · ملفٌّ مجهولٌ = %d · جسمٌ %d بايت",
		listing.Code, anon.Code, len(listing.Body))

	if listing.Code == http.StatusOK && strings.Contains(string(listing.Body), "<pre>") {
		t.Errorf("**سردُ `/media/` ما زال مفتوحاً على التجهيز** — `D13`")
	}
	if anon.Code == http.StatusOK {
		t.Errorf("**ملفٌّ مجهولُ النسب خُدم** (%d) — والافتراضُ الحجب", anon.Code)
	}
}
