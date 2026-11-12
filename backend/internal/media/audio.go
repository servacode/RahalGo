package media

// ══════════════════════════════════════════════════════════════════════
// **الرسالةُ الصوتيّة بحديث الطلب** (قرارُ المالك ٢٠٢٦-١١-١٢، هجرة ٠٥٤٧)
// ══════════════════════════════════════════════════════════════════════
//
// **لا فكَّ ترميزٍ ولا إعادةَ ترميز** — الصوتُ يُحفظ كما وصل بعد فحص بايتاته الأولى
// (لا الامتداد ولا الترويسة). **والاسمُ uuid** والامتدادُ من النوع المكتشف، فلا اسمَ من
// العميل يصل القرص. **ولا مصغَّرةَ** (`thumb_path` فارغ) ولا أبعاد.

import (
	"bytes"
	"context"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// MaxAudioBytes **سقفُ الرسالة الصوتيّة** — دقيقةٌ بـAAC أحاديّ لا تبلغ نصفَ ميغا.
const MaxAudioBytes = 1536 << 10

var (
	// ErrBadAudio **ليس ملفّاً صوتيّاً** نعرفه.
	ErrBadAudio = httpx.NewError(http.StatusBadRequest, "invalid_audio", "errors.invalid_audio")
	// ErrAudioTooLarge **أكبرُ من السقف أو أطولُ من دقيقة.**
	ErrAudioTooLarge = httpx.NewError(http.StatusBadRequest, "audio_too_large", "errors.audio_too_large")
)

func init() {
	// **ونوعُ الملف عند الخدمة من امتداده** — فيعرفه المتصفّحُ ومشغّلُ أندرويد.
	for ext, typ := range map[string]string{
		".m4a": "audio/mp4", ".ogg": "audio/ogg", ".webm": "audio/webm",
		".aac": "audio/aac", ".mp3": "audio/mpeg",
	} {
		_ = mime.AddExtensionType(ext, typ)
	}
}

// AudioExt **امتدادُ الصوت من بايتاته الأولى** — وفارغٌ: ليس صوتاً نعرفه.
func AudioExt(b []byte) string {
	switch {
	case len(b) >= 12 && bytes.Equal(b[4:8], []byte("ftyp")):
		return ".m4a" // MP4/M4A (AAC) — أندرويد وسفاري
	case len(b) >= 4 && bytes.HasPrefix(b, []byte("OggS")):
		return ".ogg"
	case len(b) >= 4 && bytes.HasPrefix(b, []byte{0x1A, 0x45, 0xDF, 0xA3}):
		return ".webm" // EBML — كروم وفايرفوكس
	case len(b) >= 3 && bytes.HasPrefix(b, []byte("ID3")):
		return ".mp3"
	case len(b) >= 2 && b[0] == 0xFF && (b[1] == 0xF1 || b[1] == 0xF9):
		return ".aac" // ADTS
	case len(b) >= 2 && b[0] == 0xFF && b[1]&0xE0 == 0xE0:
		return ".mp3" // إطارُ MPEG بلا ID3
	}
	return ""
}

// SaveAudio **يحفظ رسالةً صوتيّة** — صنفُها `chat_audio` دائماً.
func (s *Service) SaveAudio(ctx context.Context, actorID string, raw []byte) (*Media, error) {
	if int64(len(raw)) > MaxAudioBytes {
		return nil, ErrAudioTooLarge
	}
	ext := AudioExt(raw)
	if ext == "" {
		return nil, ErrBadAudio
	}
	now := time.Now().UTC()
	base := filepath.Join(now.Format("2006"), now.Format("01"))
	if err := os.MkdirAll(filepath.Join(s.dir, base), 0o755); err != nil {
		return nil, err
	}
	rel := filepath.ToSlash(filepath.Join(base, uuid.NewString()+ext))
	full := filepath.Join(s.dir, filepath.FromSlash(rel))
	if err := os.WriteFile(full, raw, 0o644); err != nil {
		return nil, err
	}
	m := &Media{Kind: "chat_audio", Bytes: int64(len(raw)), URL: URLFor(rel)}
	if err := s.db.QueryRow(ctx, `
		INSERT INTO media (kind, path, thumb_path, width, height, bytes, created_by)
		VALUES ('chat_audio', $1, '', 0, 0, $2, $3) RETURNING id`,
		rel, m.Bytes, actorID).Scan(&m.ID); err != nil {
		_ = os.Remove(full)
		return nil, err
	}
	return m, nil
}
