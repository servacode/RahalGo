package notify

import (
	"context"
	"strings"

	"go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// ══════════════════════════════════════════════════════════════════════
// **«صورة + اسم وسعر»** (قرارُ المالك 2026-11-09) — صاحبُ المتجر أو موظّفه يبعت صورةً بتعليق
// لرقم البوت فينضاف الصنف. `notify` لا تعرف متجراً ولا صنفاً: تلتقط الصورةَ وتنزّلها وتسلّمها
// لدالّة تُحقن (`SetInboundImage`) — كما في الموقع. وردودُ النصّ على أسئلة البوت تمرّ
// بـ`SetInboundReply` قبل معالِج النصّ العامّ.
// ══════════════════════════════════════════════════════════════════════

// InboundImage **صورةٌ (أو وسيطٌ غيرُ صورة) وصلت برسالة**.
type InboundImage struct {
	// Caption التعليق (اسم وسعر عادةً).
	Caption string
	// Mime نوع الصورة (image/jpeg …).
	Mime string
	// Data بايتات الصورة منزَّلةً — فارغة إن NotImage.
	Data []byte
	// MsgID معرّف الرسالة (لمنع المعالجة المزدوجة ولربط الردود بها).
	MsgID string
	// QuotedID ونصُّه: الرسالة التي ردّت عليها هذه (إن وُجدت).
	QuotedID   string
	QuotedText string
	// NotImage فيديو/ملف/صوت — يُردّ عليه «ابعت صورة».
	NotImage bool
}

// SetInboundImage يربط معالِجَ الصور الواردة: رقمُ المرسِل والصورة ⇒ نصُّ الردّ أو فارغ.
func (s *WhatsAppSender) SetInboundImage(f func(ctx context.Context, from string, img InboundImage) string) {
	s.hooksMu.Lock()
	s.inboundImg = f
	s.hooksMu.Unlock()
}

// SetInboundImageGate **بوابةُ الميزة** — false ⇒ الصورة لا تُسلَّم لمعالِج الأصناف وتكمل
// لمسار الواتساب العاديّ (مفتاح الميزة مطفأ عامّاً أو للمتجر، قرار المالك 2026-11-10). فارغ ⇒ مفتوحة.
func (s *WhatsAppSender) SetInboundImageGate(f func(ctx context.Context, from string) bool) {
	s.hooksMu.Lock()
	s.inboundGate = f
	s.hooksMu.Unlock()
}

// SetInboundReply يربط معالِجَ نصوصٍ تُجيب عن أسئلة البوت المعلّقة. يردّ (نصّ، هل عالج)؛
// ولا يُعالج ⇒ يكمل النصُّ إلى المعالِج العامّ.
func (s *WhatsAppSender) SetInboundReply(f func(ctx context.Context, from, text, quotedID string) (string, bool)) {
	s.hooksMu.Lock()
	s.inboundReply = f
	s.hooksMu.Unlock()
}

// ImageFromMessage **يقرأ الصورةَ من رسالة** — بلا تنزيل. kind: "image" أو "other" (فيديو/ملف/صوت) أو "".
func ImageFromMessage(m *waE2E.Message) (kind string, img *waE2E.ImageMessage) {
	if m == nil {
		return "", nil
	}
	if i := m.GetImageMessage(); i != nil {
		return "image", i
	}
	if m.GetVideoMessage() != nil || m.GetDocumentMessage() != nil || m.GetAudioMessage() != nil {
		return "other", nil
	}
	return "", nil
}

// mediaReply **يعالج صورةً واردة (أو وسيطاً غير صورة) ويردّ** — true إن كانت الرسالةُ وسيطاً
// والتقطها المعالِج. التنزيلُ والمعالجةُ في خيطٍ مستقلّ فلا يتجمّد الوارد.
func (s *WhatsAppSender) mediaReply(ctx context.Context, c *whatsmeow.Client, msg *events.Message) bool {
	kind, im := ImageFromMessage(msg.Message)
	if kind == "" {
		return false
	}
	s.hooksMu.RLock()
	h := s.inboundImg
	gate := s.inboundGate
	s.hooksMu.RUnlock()
	if h == nil {
		return false
	}
	var lids lidResolver
	if c.Store != nil {
		lids = c.Store.LIDs
	}
	from := ResolveSenderPhone(ctx, msg.Info, lids)
	if from == "" {
		s.logger.Warn("whatsapp: صورةٌ واردة بلا رقمٍ معروف للمرسِل")
		return true
	}
	if gate != nil && !gate(ctx, from) {
		return false
	}
	in := InboundImage{MsgID: msg.Info.ID, NotImage: kind != "image"}
	if im != nil {
		in.Caption, in.Mime = im.GetCaption(), im.GetMimetype()
		if ci := im.GetContextInfo(); ci != nil {
			in.QuotedID = ci.GetStanzaID()
			in.QuotedText = quotedText(ci.GetQuotedMessage())
		}
	}
	jid := msg.Info.Sender.ToNonAD()
	if msg.Info.Chat.User != "" {
		jid = msg.Info.Chat.ToNonAD()
	}
	go func() {
		if im != nil {
			data, err := c.Download(ctx, im)
			if err != nil {
				s.logger.Warn("whatsapp: تعذّر تنزيل صورة واردة", "error", err)
				return
			}
			in.Data = data
		}
		reply := h(ctx, from, in)
		if strings.TrimSpace(reply) == "" {
			return
		}
		if _, err := c.SendMessage(ctx, jid, &waE2E.Message{Conversation: proto.String(reply)}); err != nil {
			s.logger.Warn("whatsapp: تعذّر الردّ على صورة", "error", err)
		}
	}()
	return true
}

func quotedText(q *waE2E.Message) string {
	if q == nil {
		return ""
	}
	if t := q.GetConversation(); t != "" {
		return t
	}
	if e := q.GetExtendedTextMessage(); e != nil {
		return e.GetText()
	}
	if i := q.GetImageMessage(); i != nil {
		return i.GetCaption()
	}
	return ""
}

// replyHook **يعرض النصَّ على معالِج الأجوبة المعلّقة** — (فارغ، false) إن لم يُربط أو لم يعالج.
func (s *WhatsAppSender) replyHook(ctx context.Context, from, text, quotedID string) (string, bool) {
	s.hooksMu.RLock()
	h := s.inboundReply
	s.hooksMu.RUnlock()
	if h == nil {
		return "", false
	}
	return h(ctx, from, text, quotedID)
}
