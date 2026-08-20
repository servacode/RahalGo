package qa

// ══════════════════════════════════════════════════════════════════════
// **الطبقةُ الثانية عشرة — بأيّ طريقٍ يعلم المتجرُ بطلبه**
// ══════════════════════════════════════════════════════════════════════
//
// المعرّفات: `MDIS-*` · الوسم: `@api @critical @release`
//
// (قرارُ المالك ٢٠٢٦-٠٨-٢٠: «من غير المعقول أن نرسل الطلبات عبر رسالة
//
//	SMS — إمّا بشكلٍ يدويٍّ على واتساب كما هي حاليّاً، أو بشكلٍ تلقائيٍّ
//	البوت يرسلها. لا أريد SMS بإرسال الطلبات».)
//
// # وما يُقاس
//
// **أنّ قناةً حُذفت لا تعود من بابٍ خلفيّ** — **وقناةٌ تُحذف من الشاشة
// وتبقى في المحرّك تعود يومَ يكتب أحدٌ نداءً بيده.**

import (
	"encoding/json"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/settings"
)

// TestMDIS_001_SMSChannelIsGone **ولا قناةَ رسائلَ للطلبات.**
//
// **وزرُّ اللوحة يرسل `whatsapp` دائماً منذ بُني** — **فبقيت قناةُ
// الرسائل شهوراً في المحرّك لا يناديها أحد.**
func TestMDIS_001_SMSChannelIsGone(t *testing.T) {
	h := New(t)
	ops := h.NewUser("ops")
	item := h.NewItem(1000)
	cust := h.Customer()
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("MDIS-001 تعذّر التجهيز: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)

	got := h.POST("/api/v1/admin/orders/"+oid+"/whatsapp", ops.Token,
		map[string]any{"channel": "sms"})
	if got.Code < 400 {
		t.Errorf("MDIS-001 **قناةُ الرسائل ما زالت تُقبل**: %s", got)
	}
}

// TestMDIS_010_MessageCarriesNoSMSFlag **ولا رايةَ لقناةٍ لا وجودَ لها.**
//
// **وحقلٌ يبقى في الردّ بعد أن حُذف معناه يجعل الشاشةَ تبني عليه** —
// ثمّ يُقرأ يوماً على أنّه حقيقة.
func TestMDIS_010_MessageCarriesNoSMSFlag(t *testing.T) {
	h := New(t)
	ops := h.NewUser("ops")
	item := h.NewItem(1000)
	cust := h.Customer()
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("MDIS-010 تعذّر التجهيز: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)

	res := h.GET("/api/v1/admin/orders/"+oid+"/message", ops.Token)
	if res.Code >= 400 {
		t.Fatalf("MDIS-010 نداءُ الرسالة رُدّ: %s", res)
	}
	var body struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &body); err != nil {
		t.Fatalf("MDIS-010 ردٌّ غيرُ مقروء: %v", err)
	}
	if _, has := body.Data["sms_ready"]; has {
		t.Error("MDIS-010 **`sms_ready` ما زالت في الردّ** — والقناةُ حُذفت")
	}
	// **ورابطُ واتساب يبقى** — هو الطريقُ اليدويُّ الذي أبقاه المالك.
	if link, _ := body.Data["wa_link"].(string); link == "" {
		t.Error("MDIS-010 **رابطُ واتساب غاب** — والإرسالُ اليدويُّ يقوم عليه")
	}
}

// TestMDIS_020_NoVerifiedMessage **ولا رسالةَ «وُثّق رقمك».**
//
// (قرارُ المالك ٢٠٢٦-٠٨-٢٠ — حُذفت.)
//
// **رسالةٌ ثانيةٌ إلى رقمٍ جديدٍ لم يُراسَل من قبل** — **وهذا بعينه
// نمطُ الإرسال الذي يُحظَر عليه البوت**، مقابل جملةٍ تقولها الشاشة.
func TestMDIS_020_NoVerifiedMessage(t *testing.T) {
	if _, ok := settings.Lookup("whatsapp.verified_template"); ok {
		t.Error("MDIS-020 **مفتاحُ قالبِ التوثيق ما زال في الفهرس** — " +
			"ويُعرض في اللوحة ولا يقرؤه أحد")
	}
}
