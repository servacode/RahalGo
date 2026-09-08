package qa

// ══════════════════════════════════════════════════════════════════════
// **«لا أعرف» تصل إلى صاحبها كما هي** — `XG-41B` · `R16`
// ══════════════════════════════════════════════════════════════════════
//
// **والمعجمُ وحدَه لا يكفي**: **مدخلٌ صحيحٌ في المعجم ومفتاحٌ آخرُ على
// السلك يعطي الرسالةَ العامّةَ نفسَها.** **فيُقاس السلك.**
//
// **والحالُ تُصنَع بالمسار الحقيقيّ لا بنداءٍ داخليّ**: **رمزٌ يحمل
// معرّفَ جلسةٍ لا تقرؤه الحقيقةُ الموثوقة** ⇒ `SessionRows` تسقط ⇒
// `ErrSessionCheckUnavailable` ⇒ **٥٠٣.**
//
// **ولا يُمَسّ سلوكُ التوثيق**: الرمزُ والحالُ كما كانا — **المبدَّلُ
// نصٌّ يقرؤه إنسان.**

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestXG41B_AuthUnavailableReachesTheReaderAsItself(t *testing.T) {
	h := New(t)
	u := h.Factory().NewUserWith("customer")

	// **معرّفُ جلسةٍ لا يقرؤه الجدول** — **فالحقيقةُ الموثوقةُ تسقط
	// ولا تكذب.** (وهو النمطُ الثالثُ في `R16`: «لا أعرف».)
	tok := h.TokenWithSession(u.ID, "لا-جلسة-بهذا-المعرّف", "customer")

	got := h.GET("/api/v1/auth/me", tok)
	t.Logf("XG-41B REAL PATH: status=%d · code=%q · key=%q",
		got.Code, got.Err(), messageKeyOf(got))

	// **والحالُ والرمزُ لا يتبدّلان** — **هذه دورةُ نصٍّ لا دورةُ عقد.**
	if got.Code != 503 {
		t.Fatalf("**الحالُ %d لا ٥٠٣** — **و«لا أعرف» ليست «رمزُك مُبطَل»**", got.Code)
	}
	if got.Err() != "auth_unavailable" {
		t.Errorf("**رمزُ الخطأ %q** — والعقدُ `auth_unavailable`", got.Err())
	}
	key := messageKeyOf(got)
	if key != "errors.auth_unavailable" {
		t.Fatalf("**مفتاحُ الرسالة %q** — والمعجمُ يعرف `errors.auth_unavailable`", key)
	}

	// **ثمّ يُترجَم كما تترجمه الواجهة** — **بالمفتاح الذي وصل فعلاً.**
	raw, err := os.ReadFile(filepath.Join("..", "..", "..",
		"web", "packages", "i18n", "src", "locales", "ar.json"))
	if err != nil {
		t.Skipf("المعجمُ غيرُ متاح: %v", err)
	}
	var dict struct {
		Errors map[string]string `json:"errors"`
	}
	if err := json.Unmarshal(raw, &dict); err != nil {
		t.Fatalf("المعجمُ لا يُقرأ: %v", err)
	}
	msg := dict.Errors[strings.TrimPrefix(key, "errors.")]
	t.Logf("XG-41B READS: %q", msg)
	if strings.TrimSpace(msg) == "" || msg == dict.Errors["internal"] {
		t.Errorf("**ما يقرؤه صاحبُ الجلسة**: %q — "+
			"**والعامّةُ تقول «في المنصّة عطب» ولا عطبَ فيها**", msg)
	}
}

// messageKeyOf مفتاحُ رسالةِ الخطأ كما يصل الواجهة.
func messageKeyOf(r Res) string {
	var e struct {
		Error struct {
			MessageKey string `json:"message_key"`
		} `json:"error"`
	}
	_ = json.Unmarshal(r.Body, &e)
	return e.Error.MessageKey
}
