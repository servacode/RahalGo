package notify

import (
	"context"

	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// ══════════════════════════════════════════════════════════════════════
// **«ردّ بموقعك»** (قرارُ المالك ٢٠٢٦-١١-٠٣)
// ══════════════════════════════════════════════════════════════════════
//
// الناسُ تخاف الروابط، وأزرارُ «أرسل الموقع» التفاعليّة لواجهة الأعمال الرسميّة وحدَها
// ونحن لا نستعملها. **فيردّ المستلِمُ على رسالة البوت بموقعه من مشبك واتساب نفسِه**
// ويلتقطه البوت هنا ويسلّمه للخادم (`SetInboundLocation`).

// InboundLocation **موقعٌ وصل برسالة**.
type InboundLocation struct {
	Lat, Lng  float64
	AccuracyM float64 // 0 = غير معروفة
	Live      bool    // موقعٌ مباشر
}

// LocationFromMessage **يقرأ نقطةً من رسالة موقعٍ ثابتٍ أو مباشر** — وإلّا `false`.
func LocationFromMessage(m *waE2E.Message) (InboundLocation, bool) {
	if m == nil {
		return InboundLocation{}, false
	}
	if l := m.GetLocationMessage(); l != nil {
		return InboundLocation{Lat: l.GetDegreesLatitude(), Lng: l.GetDegreesLongitude(),
			AccuracyM: float64(l.GetAccuracyInMeters())}, true
	}
	if l := m.GetLiveLocationMessage(); l != nil {
		return InboundLocation{Lat: l.GetDegreesLatitude(), Lng: l.GetDegreesLongitude(),
			AccuracyM: float64(l.GetAccuracyInMeters()), Live: true}, true
	}
	return InboundLocation{}, false
}

// lidResolver **ما نحتاجه من مخزن واتساب** — معرّفٌ خفيّ ⇒ رقم.
type lidResolver interface {
	GetPNForLID(ctx context.Context, lid types.JID) (types.JID, error)
}

// ResolveSenderPhone **رقمُ المرسِل بصيغة `+963…`** — أو سلسلةٌ فارغة إن تعذّر.
//
// **واتساب قد يخاطب المرسِلَ بمعرّفٍ خفيّ (LID) لا برقمه** — فيُؤخذ الرقمُ من العنوان
// البديل (`SenderAlt`) إن وُجد، وإلّا من جدول التطابق في المخزن. **ولا يُخمَّن رقمٌ من
// معرّفٍ خفيّ** (أرقامُه عشوائيّة).
func ResolveSenderPhone(ctx context.Context, info types.MessageInfo, lids lidResolver) string {
	isPN := func(j types.JID) bool { return j.Server == types.DefaultUserServer && j.User != "" }
	if isPN(info.Sender) {
		return "+" + info.Sender.User
	}
	if isPN(info.SenderAlt) {
		return "+" + info.SenderAlt.User
	}
	if info.Sender.Server == types.HiddenUserServer && lids != nil {
		if pn, err := lids.GetPNForLID(ctx, info.Sender.ToNonAD()); err == nil && isPN(pn) {
			return "+" + pn.User
		}
	}
	return ""
}

// SetInboundLocation **يربط معالِجَ المواقع الواردة**: يأخذ رقمَ المرسِل والنقطة ويردّ
// بنصٍّ يُرسَل، أو سلسلةً فارغةً فلا يُردّ.
func (s *WhatsAppSender) SetInboundLocation(f func(ctx context.Context, from string, loc InboundLocation) string) {
	s.hooksMu.Lock()
	s.inboundLoc = f
	s.hooksMu.Unlock()
}

// locationReply **يعالج رسالةَ موقعٍ واردة ويردّ بنصّ الجواب** — فارغٌ إن لم تكن موقعاً
// أو تعذّر رقمُ المرسِل أو لم يردّ المعالِج (لا طلبَ مفتوحاً).
func (s *WhatsAppSender) locationReply(ctx context.Context, msg *events.Message, lids lidResolver) (string, bool) {
	loc, ok := LocationFromMessage(msg.Message)
	if !ok {
		return "", false
	}
	s.hooksMu.RLock()
	h := s.inboundLoc
	s.hooksMu.RUnlock()
	if h == nil {
		return "", true
	}
	from := ResolveSenderPhone(ctx, msg.Info, lids)
	if from == "" {
		s.logger.Warn("whatsapp: موقعٌ وارد بلا رقمٍ معروف للمرسِل")
		return "", true
	}
	return h(ctx, from, loc), true
}
