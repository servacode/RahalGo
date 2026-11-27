package notify

// ══════════════════════════════════════════════════════════════════════
// **قائمة بوت الواتساب** (قرارُ المالك ٢٠٢٦-١١-٢٧)
// ══════════════════════════════════════════════════════════════════════
//
// `notify` لا تعرف حساباً ولا طلباً — **تسلّم كلَّ ما يصل (أو يخرج من هاتف صاحب الرقم) لدالّةٍ تُحقن**
// (`SetMenu`)، والخادمُ يقرّر (`server/wa_bot_menu.go`).
//
// **والأسبقيّةُ للمسارات القائمة**: طلبُ الموقع، وصورُ أصناف المتاجر، وأجوبةُ أسئلة البوت المعلّقة،
// ورموزُ التوثيق، وتقاريرُ المنصّة. **ما عالجه غيرُ القائمة يصلها بـ`Handled`** — لتعرف أنّ المحادثةَ
// نشطة (السكوتُ قبل إعادة القائمة يُحسب من آخر رسالةٍ في الاتّجاهين) **ولا تردّ عليه.**

import (
	"context"
	"strings"

	"go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// Menu kinds — نوعُ ما وصل.
const (
	MenuText     = "text"
	MenuLocation = "location"
	MenuMedia    = "media"
)

// MenuInbound **رسالةٌ للقائمة.**
type MenuInbound struct {
	// From **رقمُ صاحب المحادثة** بصيغة `+963…`.
	From string
	// Kind نصٌّ أو موقعٌ أو وسيط (صوت/صورة/فيديو/ملف).
	Kind string
	Text string
	Loc  InboundLocation
	// Handled **عالجها مسارٌ آخر** — للنشاط وحدَه، ولا ردّ.
	Handled bool
	// Outgoing **رسالةٌ كتبها صاحبُ الرقم (موظّف) من هاتفه في هذه المحادثة** — للنشاط وحدَه.
	Outgoing bool
}

// MenuReply **ردُّ القائمة** — نصٌّ (أو فارغ)، و`SendApp` يبعث ملفَّ تطبيق الزبون بعده.
type MenuReply struct {
	Text    string
	SendApp bool
}

// SetMenu **يربط معالِجَ القائمة.**
func (s *WhatsAppSender) SetMenu(f func(ctx context.Context, in MenuInbound) MenuReply) {
	s.hooksMu.Lock()
	s.menu = f
	s.hooksMu.Unlock()
}

func (s *WhatsAppSender) menuHook() func(ctx context.Context, in MenuInbound) MenuReply {
	s.hooksMu.RLock()
	defer s.hooksMu.RUnlock()
	return s.menu
}

// chatPhone **رقمُ الطرف الآخر في محادثةٍ فرديّة** — من معرّف المحادثة (رقماً أو معرّفاً خفيّاً).
func chatPhone(ctx context.Context, chat types.JID, lids lidResolver) string {
	switch {
	case chat.Server == types.DefaultUserServer && chat.User != "":
		return "+" + chat.User
	case chat.Server == types.HiddenUserServer && lids != nil:
		if pn, err := lids.GetPNForLID(ctx, chat.ToNonAD()); err == nil &&
			pn.Server == types.DefaultUserServer && pn.User != "" {
			return "+" + pn.User
		}
	}
	return ""
}

// menuDispatch **يسلّم الرسالةَ للقائمة في خيطٍ مستقلّ ويردّ في المحادثة نفسِها.**
//
// **وفي خيطٍ مستقلّ** — معالِجُ الأحداث يُعطّل ما بعده ما دام يعمل (انظر `onInbound`).
func (s *WhatsAppSender) menuDispatch(ctx context.Context, c *whatsmeow.Client, jid types.JID, in MenuInbound) {
	h := s.menuHook()
	if h == nil || in.From == "" {
		return
	}
	go s.menuRun(ctx, c, jid, h, in)
}

// menuRun **ينادي القائمةَ ويرسل ردَّها** — متزامنٌ (يُنادى من خيطٍ قائم).
func (s *WhatsAppSender) menuRun(ctx context.Context, c *whatsmeow.Client, jid types.JID,
	h func(ctx context.Context, in MenuInbound) MenuReply, in MenuInbound) {
	r := h(ctx, in)
	if in.Handled || in.Outgoing {
		return
	}
	if strings.TrimSpace(r.Text) != "" {
		if _, err := c.SendMessage(ctx, jid, &waE2E.Message{Conversation: proto.String(r.Text)}); err != nil {
			s.logger.Warn("whatsapp: تعذّر إرسالُ ردّ القائمة", "error", err)
			return
		}
	}
	if r.SendApp {
		s.sendAppFile(ctx, c, jid)
	}
}

// menuFromMessage **يرسل رسالةً واردةً لم يعالجها مسارٌ آخر إلى القائمة** — نصٌّ أو وسيطٌ أو موقع.
func (s *WhatsAppSender) menuFrom(ctx context.Context, c *whatsmeow.Client, msg *events.Message, in MenuInbound) {
	if s.menuHook() == nil {
		return
	}
	var lids lidResolver
	if c.Store != nil {
		lids = c.Store.LIDs
	}
	if in.From == "" {
		in.From = ResolveSenderPhone(ctx, msg.Info, lids)
	}
	jid := msg.Info.Sender.ToNonAD()
	if msg.Info.Chat.User != "" {
		jid = msg.Info.Chat.ToNonAD()
	}
	s.menuDispatch(ctx, c, jid, in)
}
