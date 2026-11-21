package notify

// ══════════════════════════════════════════════════════════════════════
// **«ارسل التطبيق»** (قرارُ المالك ٢٠٢٦-١١-٢١)
// ══════════════════════════════════════════════════════════════════════
//
// «بدي البوت يرسل تطبيق رحال غو الي بالموقع كملف apk للزبون وقت اكتبله ارسل التطبيق.»
//
//	صاحبُ رقم البوت يكتب «ارسل التطبيق» بمحادثة زبون  ←  البوتُ يبعت ملفَّ تطبيق الزبون
//	                                                     (آخرُ نسخة مرفوعة باللوحة) بنفس المحادثة.
//
// **ومن الرقم نفسِه بس** — الزبونُ إذا كتبها ما بيصير شي (`IsFromMe`). **والأمرُ يضلّ ظاهر**
// (قراره: «خليها تبين عادي»). **وما في ملف** ← سطرٌ بمحادثة صاحب الرقم مع نفسه، لا للزبون.

import (
	"context"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// OutgoingFile **ملفٌّ يبعته البوت** — المحتوى واسمُه ونوعُه وسطرٌ تحته.
type OutgoingFile struct {
	Data     []byte
	FileName string
	Mime     string
	Caption  string
}

// SendAppCaption **السطرُ تحت الملفّ.**
const SendAppCaption = "هاد تطبيق رحّال غو، اضغط عليه لتنزّله 📲"

// sendAppMissing **ما في ملف مرفوع** — لصاحب الرقم بمحادثته مع نفسه.
const sendAppMissing = "❌ ما في ملف تطبيق زبون مرفوع — ارفعه من اللوحة (الإعدادات ← الإصدار)"

var sendAppVerbs = map[string]bool{"ارسل": true, "ابعت": true, "ابعث": true, "بعت": true, "ارسال": true}
var sendAppNouns = map[string]bool{"التطبيق": true, "تطبيق": true}

// IsSendAppCommand **«ارسل التطبيق» وأخواتُها** — كلمتان بالضبط (أ/إ/آ متل ا)، وإشارةُ ترقيمٍ بآخرها عادي.
func IsSendAppCommand(text string) bool {
	t := strings.NewReplacer("أ", "ا", "إ", "ا", "آ", "ا").Replace(strings.TrimSpace(text))
	t = strings.TrimRight(t, " .،,!؟?")
	w := strings.Fields(t)
	return len(w) == 2 && sendAppVerbs[w[0]] && sendAppNouns[w[1]]
}

// messageText **نصُّ الرسالة** — عاديّةً أو مطوَّلة.
func messageText(m *waE2E.Message) string {
	if m == nil {
		return ""
	}
	if t := m.GetConversation(); t != "" {
		return t
	}
	if e := m.GetExtendedTextMessage(); e != nil {
		return e.GetText()
	}
	return ""
}

// SetAppFile **يربط مصدرَ ملفّ التطبيق** — انظر حقل `appFile`.
func (s *WhatsAppSender) SetAppFile(f func(ctx context.Context) (OutgoingFile, error)) {
	s.hooksMu.Lock()
	s.appFile = f
	s.hooksMu.Unlock()
}

// documentMessage **رسالةُ ملفّ من رفعٍ تمّ.**
func documentMessage(up whatsmeow.UploadResponse, f OutgoingFile) *waE2E.Message {
	return &waE2E.Message{DocumentMessage: &waE2E.DocumentMessage{
		URL:           proto.String(up.URL),
		DirectPath:    proto.String(up.DirectPath),
		MediaKey:      up.MediaKey,
		FileEncSHA256: up.FileEncSHA256,
		FileSHA256:    up.FileSHA256,
		FileLength:    proto.Uint64(up.FileLength),
		Mimetype:      proto.String(f.Mime),
		FileName:      proto.String(f.FileName),
		Title:         proto.String(f.FileName),
		Caption:       proto.String(f.Caption),
	}}
}

// sendAppFile **يرفع الملفّ ويبعته للمحادثة** — وبلا ملفٍّ يخبر صاحبَ الرقم.
func (s *WhatsAppSender) sendAppFile(ctx context.Context, c *whatsmeow.Client, to types.JID) {
	s.hooksMu.RLock()
	src := s.appFile
	s.hooksMu.RUnlock()
	if src == nil {
		return
	}
	f, err := src(ctx)
	if err != nil || len(f.Data) == 0 {
		s.logger.Warn("whatsapp: «ارسل التطبيق» بلا ملف", "error", err)
		if c.Store != nil && c.Store.ID != nil {
			_, _ = c.SendMessage(ctx, c.Store.ID.ToNonAD(), &waE2E.Message{Conversation: proto.String(sendAppMissing)})
		}
		return
	}
	up, err := c.Upload(ctx, f.Data, whatsmeow.MediaDocument)
	if err != nil {
		s.logger.Warn("whatsapp: تعذّر رفعُ ملفّ التطبيق", "error", err)
		return
	}
	if _, err := c.SendMessage(ctx, to, documentMessage(up, f)); err != nil {
		s.logger.Warn("whatsapp: تعذّر إرسالُ ملفّ التطبيق", "error", err)
	}
}
