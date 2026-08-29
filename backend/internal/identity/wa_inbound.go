package identity

import (
	"context"
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ══════════════════════════════════════════════════════════════════════
// **التوثيقُ المقلوب — يراسلنا فنردّ**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك 2026-08-25: «يكون في رسالة جاهزة لازم يرسلها ع رقم
//
//	المنصة… يصله الكود يوثق حسابه».)
//
// # ولماذا قُلب
//
// **وواتساب قيّد رقمَ المنصّة مرّتين في يومٍ واحد** — الثانيةُ بعد
// رسالتَي تحقّقٍ اثنتين. **والسببُ الاتّجاه**: حسابٌ شخصيٌّ يبدأ محادثةً
// مع من لم يراسله. **ولا تُصلح ذلك مهلةٌ ولا صياغةُ نصّ.**
//
// **والردُّ مسموح** — فيبدأ الزبونُ ونردّ نحن.
//
// # ولماذا وسمٌ لا رقمٌ وحدَه
//
// **ومن سجّل برقمٍ قد يراسلنا من واتساب رقمٍ آخر** — هاتفُ زوجته أو
// رقمٌ ثانٍ في الجهاز. **فلو ربطنا بالرقم وحدَه ضاع نصفُهم.**
//
// **فيُولَّد وسمٌ قصيرٌ يُدسّ في الرسالة الجاهزة** — والوسمُ يعرف صاحبَه.
//
// # ولماذا سبعةُ محارف
//
// **والوسمُ يُقرأ بالعين ويُكتب باليد أحياناً** — فلا يطول. **وسبعةٌ من
// أبجديّةٍ من ٣٢ حرفاً تعطي أكثرَ من ثلاثةٍ وثلاثين مليارَ احتمال**،
// وعمرُه عشرُ دقائق. **فالتخمينُ مستحيلٌ عمليّاً.**
//
// **وحُذفت `0` و`O` و`1` و`I`** — تُقرأ خطأً فيشكو صاحبُها أنّ الرمز
// لا يعمل.
const waTagAlphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

const waTagLen = 7

// waTicketTTL **عمرُ التذكرة** — عشرُ دقائق تكفي لفتح واتساب وإرسال
// رسالة، **ولا تكفي لمن وجد الوسمَ في شاشةٍ متروكة.**
const waTicketTTL = 10 * time.Minute

// WAPurpose **ما طلبه صاحبُ الحساب.**
type WAPurpose string

const (
	// WAVerify **توثيقُ الحساب** — يُطلب عند أوّل طلب.
	WAVerify WAPurpose = "verify"
	// WAReset **استعادةُ كلمة المرور.**
	WAReset WAPurpose = "reset"
)

// waTagRe **يلتقط الوسمَ من أيّ نصّ** — والزبونُ قد يكتب قبله وبعده.
var waTagRe = regexp.MustCompile(`\b([23456789ABCDEFGHJKLMNPQRSTUVWXYZ]{7})\b`)

func newWATag() (string, error) {
	b := make([]byte, waTagLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	out := make([]byte, waTagLen)
	for i, v := range b {
		out[i] = waTagAlphabet[int(v)%len(waTagAlphabet)]
	}
	return string(out), nil
}

func waTicketKey(tag string) string { return "wa:ticket:" + tag }

// WATicket **يفتح تذكرةً ويردّ وسمَها.**
//
// **والهاتفُ يُخزَّن مع التذكرة لا يُستنتج من المرسِل** — انظر أعلاه.
func (s *Service) WATicket(ctx context.Context, rawPhone string, purpose WAPurpose) (string, error) {
	phone, ok := NormalizePhone(rawPhone)
	if !ok {
		return "", ErrInvalidPhone
	}
	tag, err := newWATag()
	if err != nil {
		return "", err
	}
	val := string(purpose) + "|" + phone
	if err := s.rdb.Set(ctx, waTicketKey(tag), val, waTicketTTL).Err(); err != nil {
		return "", err
	}
	return tag, nil
}

// HandleWAInbound **يقرأ رسالةً واردةً ويردّ بالرمز — أو لا يردّ.**
//
// # ولا يُردُّ على ما لا يعنينا
//
// **ورسالةٌ بلا وسمٍ صالحٍ ليست طلبَ توثيق** — وقد تكون سؤالاً من
// زبونٍ أو رقماً خاطئاً. **والردُّ الآليُّ عليها إزعاجٌ وبلاغُ سبام.**
//
// # والتذكرةُ تُستهلك
//
// **ووسمٌ يُقبل مرّتين يُقبل ألفاً** — فمن سرّبه أُرسل الرمزُ إلى كلّ
// من كتبه. **فتُحذف عند أوّل استعمال.**
func (s *Service) HandleWAInbound(ctx context.Context, from, text string) string {
	m := waTagRe.FindStringSubmatch(strings.ToUpper(text))
	if m == nil {
		return ""
	}
	tag := m[1]
	key := waTicketKey(tag)
	val, err := s.rdb.Get(ctx, key).Result()
	if err != nil || val == "" {
		// **ووسمٌ منتهٍ يُقال لصاحبه** — وإلّا انتظر رمزاً لن يأتي.
		return "انتهت صلاحية الطلب. أعد المحاولة من التطبيق."
	}
	parts := strings.SplitN(val, "|", 2)
	if len(parts) != 2 {
		return ""
	}
	purpose, phone := WAPurpose(parts[0]), parts[1]
	// **وتُحذف قبل التوليد لا بعده** — انظر أعلاه.
	s.rdb.Del(ctx, key)

	code, err := randomDigits(6)
	if err != nil {
		s.logger.Error("wa inbound: تعذّر توليدُ رمز", "error", err)
		return ""
	}
	// **والغرضُ يُحفظ مع الرمز** — فرمزُ التوثيق لا يستعيد كلمةَ مرور.
	p := "whatsapp"
	if purpose == WAReset {
		p = "reset"
	}
	if err := s.repo.CreateOTP(ctx, phone, s.hashOTP(phone, code), p, s.otpLifetime(ctx)); err != nil {
		s.logger.Error("wa inbound: تعذّر حفظُ الرمز", "error", err)
		return ""
	}
	s.logger.Info("wa inbound: رمزٌ أُرسل رداً", "purpose", p, "from", from)
	// **والرقمُ الذي راسلنا قد يخالف رقمَ الحساب** — ولا يُذكر هنا
	// رقمُ الحساب: **من قرأ الشاشةَ فوق كتفه لا يعرف لمن الرمز.**
	return fmt.Sprintf("رمز رحال غو: %s\nصالح لعشر دقائق.", code)
}
