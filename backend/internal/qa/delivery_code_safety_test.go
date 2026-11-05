package qa

// **واتسابُ كود الاستلام يحمل سطرَ الحماية — وإشعارُ التطبيق بلاه** (قرارُ المالك
// ٢٠٢٦-١١-٠٥، البند ٦): الكودُ يصل لحظةَ يستلم السائقُ الطلبَ وهو في طريقه إلى الزبون،
// **فيُذكَّر الزبونُ ألّا يتعامل معه برّا التطبيق.**

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

// dcodeTextBot **بوتٌ يحفظ ما أُرسل** — ليُقرأ النصُّ نفسُه.
type dcodeTextBot struct {
	mu    sync.Mutex
	texts []string
	calls atomic.Int32
}

func (b *dcodeTextBot) Ready() bool { return true }
func (b *dcodeTextBot) SendText(_ context.Context, _ string, text string) error {
	b.mu.Lock()
	b.texts = append(b.texts, text)
	b.mu.Unlock()
	b.calls.Add(1)
	return nil
}

func (b *dcodeTextBot) find(sub string) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, x := range b.texts {
		if strings.Contains(x, sub) {
			return x
		}
	}
	return ""
}

func TestDCODE_WhatsAppCarriesSafetyNotice(t *testing.T) {
	h := New(t)
	h.Setting("delivery.code_required", "true")
	h.Setting("delivery.code_channel", `"both"`)
	t.Cleanup(func() { h.API.SetMerchantNotifier(nil) })

	bot := &dcodeTextBot{}
	oid, cust, _ := dcodeOrderWith(t, h, func() { h.API.SetMerchantNotifier(bot) })
	code, _ := dcodeOf(t, h, oid)
	if code == nil {
		t.Fatal("DCODE لا كود")
	}
	var wa string
	deadline := time.Now().Add(5 * time.Second)
	for wa == "" {
		if time.Now().After(deadline) {
			t.Fatal("لم يصل واتسابُ الكود")
		}
		wa = bot.find(*code)
		time.Sleep(20 * time.Millisecond)
	}
	if !strings.Contains(wa, orders.SafetyNotice) {
		t.Fatalf("**واتسابُ الكود بلا سطر الحماية**: %q", wa)
	}

	var body string
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT body FROM notifications
		WHERE user_id = $1::uuid AND entity_id = $2::uuid AND body LIKE '%' || $3 || '%'
		ORDER BY created_at DESC LIMIT 1`, cust.ID, oid, *code).Scan(&body); err != nil {
		t.Fatalf("إشعارُ الكود في التطبيق: %v", err)
	}
	if strings.Contains(body, orders.SafetyNotice) {
		t.Fatalf("**إشعارُ التطبيق يحمل سطرَ الحماية** — قصيرٌ يُقرأ على شاشةٍ مقفلة: %q", body)
	}
}
