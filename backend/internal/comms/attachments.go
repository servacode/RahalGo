package comms

// ══════════════════════════════════════════════════════════════════════
// **صورةٌ ورسالةٌ صوتيّة في حديث الطلب** (قرارُ المالك ٢٠٢٦-١١-١٢، هجرة ٠٥٤٧)
// ══════════════════════════════════════════════════════════════════════
//
// **البابُ نفسُه والصلاحيةُ نفسُها** (`Permit`): طرفا الحديث يرسلان، والقناةُ المغلقةُ
// ترفض، وفاصلُ المعدّل نفسُه. **والملفُّ وسيطٌ محميٌّ برابطٍ موقَّع** يصدر مع قراءةٍ
// مُصرَّحٍ بها — فمن ليس طرفاً لا يملك رابطاً، والرابطُ العاري يردّ ٤٠٤.
//
// # والتطبيقُ القديمُ لا يرى فقاعةً فارغة
//
// **`body` إلزاميٌّ** (قيدُ ٠٠٨٧) فيُكتب فيه نصٌّ بديل: «📷 صورة» أو «🎤 رسالة صوتية»،
// وبعده التعليقُ إن كُتب. **والتطبيقُ الجديدُ يقرأ `attachment`** ويعرض التعليقَ وحده.

import (
	"context"
	"net/http"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
)

// نوعا المرفق.
const (
	AttachPhoto = "photo"
	AttachAudio = "audio"
)

// النصُّ البديلُ للتطبيق القديم — وهو نصُّ الإشعار أيضاً.
const (
	FallbackPhoto = "📷 صورة"
	FallbackAudio = "🎤 رسالة صوتية"
)

// MaxAttachments **سقفُ المرفقات لكلّ مرسِلٍ في الطلب الواحد.**
const MaxAttachments = 20

// MaxAudioMs **أطولُ رسالةٍ صوتيّة** — دقيقة.
const MaxAudioMs = 60_000

// ErrAttachmentLimit **بلغ سقفَ المرفقات في هذا الطلب.**
var ErrAttachmentLimit = httpx.NewError(http.StatusConflict,
	"chat_attachments_limit", "errors.chat_attachments_limit")

// Attachment **المرفقُ كما يُعرض** — والروابطُ موقَّعةٌ محدودةُ الأجل.
type Attachment struct {
	Kind       string `json:"kind"`
	URL        string `json:"url"`
	ThumbURL   string `json:"thumb_url,omitempty"`
	DurationMs int    `json:"duration_ms,omitempty"`
	// Caption **ما كُتب مع المرفق** — `body` بلا النصّ البديل.
	Caption string `json:"caption,omitempty"`
}

// AttachmentInput **ما حُفظ للتوّ** — صفُّ الوسيط ومساراه.
type AttachmentInput struct {
	MediaID    string
	Kind       string
	Path       string
	Thumb      string
	DurationMs int
}

// Fallback **النصُّ البديلُ لنوعٍ** — وفارغٌ لنوعٍ لا نعرفه.
func Fallback(kind string) string {
	switch kind {
	case AttachPhoto:
		return FallbackPhoto
	case AttachAudio:
		return FallbackAudio
	}
	return ""
}

// AttachmentBody **ما يُخزَّن في `body`** — البديلُ ثمّ التعليقُ في سطرٍ ثانٍ.
func AttachmentBody(kind, caption string) string {
	caption = strings.TrimSpace(caption)
	if caption == "" {
		return Fallback(kind)
	}
	return Fallback(kind) + "\n" + caption
}

type attachmentRow struct {
	kind, path, thumb string
	durationMs        int
}

// build **يبني المرفقَ للعرض** — وفارغٌ إن لم يكن، أو مُحي ملفُّه.
func (a attachmentRow) build(body string) *Attachment {
	if a.kind == "" || a.path == "" {
		return nil
	}
	out := &Attachment{Kind: a.kind, URL: media.SignedURL(stripMedia(a.path))}
	if a.thumb != "" {
		out.ThumbURL = media.SignedURL(stripMedia(a.thumb))
	}
	if a.kind == AttachAudio {
		out.DurationMs = a.durationMs
	}
	rest := strings.TrimPrefix(body, Fallback(a.kind))
	out.Caption = strings.TrimSpace(rest)
	return out
}

func stripMedia(p string) string { return strings.TrimPrefix(p, "/media/") }

// PreflightAttachment **أيُقبل مرفقٌ الآن؟** — قبل أن يُكتب ملفٌّ على القرص.
//
// القناةُ مفتوحة، والفاصلُ مضى، والسقفُ لم يُبلغ. **وفحصُها قبل الحفظ** يمنع ملفّاتٍ
// تُكتب ثمّ يُرفض صفُّها.
func (s *Service) PreflightAttachment(ctx context.Context, p *Permission) error {
	if !p.Open {
		return ErrChannelClosed
	}
	if err := s.checkGap(ctx, p); err != nil {
		return err
	}
	var n int
	if err := s.db.QueryRow(ctx, `
		SELECT count(*) FROM order_messages
		 WHERE order_id = $1 AND sender_id = $2 AND attachment_kind IS NOT NULL`,
		p.OrderID, p.SelfID).Scan(&n); err != nil {
		return err
	}
	if n >= MaxAttachments {
		return ErrAttachmentLimit
	}
	return nil
}

// SendAttachment **يكتب سطراً بمرفق** — بالصلاحية والفاصل نفسَيهما (`send`).
func (s *Service) SendAttachment(ctx context.Context, p *Permission, caption string, att AttachmentInput) (*Message, error) {
	if Fallback(att.Kind) == "" {
		return nil, ErrEmptyBody
	}
	return s.send(ctx, p, AttachmentBody(att.Kind, caption), &att)
}
