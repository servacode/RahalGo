// Package cartreminder **«سلّتك ناطرتك» — تذكيرُ السلّة المتروكة** (قرارُ المالك ٢٠٢٦-١١-٢٥).
//
// # القاعدة
//
// زبونٌ في سلّته أغراضٌ ولم يطلب ⇒ **بعد ساعةٍ من آخر تغييرٍ في السلّة** إشعارٌ واحد
// («كمّل طلبك»). **مرّةً لكلّ سلّة**: لا تذكيرَ جديدَ إلّا إن تغيّرت السلّةُ بعد التذكير.
// والطلبُ أو إفراغُ السلّة يلغيه.
//
// # ولماذا خادمٌ ومكنسة
//
// **السلّةُ عند العميل فقط** (التطبيقُ والمتصفّح) — فالعميلُ يُبلغ حالَها (`Report`) وتُفرَّغ
// بـ`Clear`، **والمكنسةُ** (`Sweep`) تلتقط المستحقَّ وترسل بمحرّك الإشعارات القائم: صندوقٌ
// ودفعُ تطبيقٍ ودفعُ متصفّح. **والصفُّ في القاعدة** لا في الذاكرة: إعادةُ تشغيل الخادم لا تُضيّع
// مؤقّتاً ولا تكرّر تذكيراً.
//
// # وساعةُ الهدوء
//
// **تُحترَم** (`notify.quiet_from/to` — سياسةُ الإزعاج نفسُها في `campaigns.Quiet`): تذكيرٌ يحلّ
// وقتُه ليلاً **يُؤجَّل إلى انتهائها ولا يسقط** — فالمكنسةُ تتخطّى الصفَّ والصفُّ ينتظر.
// ولا أفضليّاتِ إشعاراتٍ لكلّ زبون في المنصّة (قِيس: لا جدولَ ولا عمود)، فالمفتاحُ العامُّ
// `cart_reminder.enabled` هو الباب.
package cartreminder

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/campaigns"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// مفاتيحُ الإعدادات — في `settings/catalog.go`.
const (
	KeyEnabled      = "cart_reminder.enabled"
	KeyAfterMinutes = "cart_reminder.after_minutes"
	// DefaultAfterMinutes **ساعةٌ** — قرارُ المالك.
	DefaultAfterMinutes = 60
)

// Entity **وجهةُ الإشعار** — والمعرّفُ معرّفُ المتجر. يعرفها التطبيقُ (`Engagement.DEST_CART`)
// والمتصفّحُ (`push.WebURL`) فيفتحان السلّة.
const Entity = "cart"

const (
	// textTitle و textBody **بلهجةٍ بسيطة** — `{store}` اسمُ المتجر.
	textTitle = "سلّتك ناطرتك 🛒"
	textBody1 = "في أغراض بسلّتك من "
	textBody2 = " — كمّل طلبك"
	// staleAfter **سلّةٌ أقدمُ من يومٍ لا تُذكَّر** — انقطاعُ المكنسة أو تفعيلُ المفتاح بعد أيّامٍ
	// لا يُحيي سلالَ الأسبوع الماضي. **وتتّسع لتأجيل ساعة الهدوء** (أطولُه نحوُ ١٢ ساعة).
	staleAfter = 24 * time.Hour
	// batch **سقفُ الجولة** — والباقي في التالية.
	batch = 500
	// maxSignature **سقفُ بصمة السلّة الواردة** — ما فوقه يُقصّ قبل التجزئة.
	maxSignature = 8192
)

// Settings ما تقرؤه الحزمة.
type Settings interface {
	GetBool(ctx context.Context, key string) bool
	GetInt(ctx context.Context, key string) int64
}

// Notifier **محرّكُ الإشعارات القائم.**
type Notifier interface {
	Notify(ctx context.Context, in notifications.Input)
}

// ErrInvalid **بلاغٌ مرفوض** — متجرٌ غيرُ معروف أو عدٌّ غيرُ موجب.
var ErrInvalid = errors.New("cartreminder: invalid state")

// Service خدمةُ التذكير.
type Service struct {
	db       *pgxpool.Pool
	settings Settings
	notify   Notifier
	logger   *slog.Logger
	// now **ساعةٌ تُبدَّل في الاختبار.**
	now func() time.Time
}

// New ينشئ الخدمة.
func New(db *pgxpool.Pool, st Settings, n Notifier, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{db: db, settings: st, notify: n, logger: logger, now: time.Now}
}

// SetClock **للاختبار وحدَه.**
func (s *Service) SetClock(now func() time.Time) {
	if now == nil {
		now = time.Now
	}
	s.now = now
}

// Enabled **المفتاحُ الرئيسيّ** — مطفأٌ افتراضاً، والمالكُ يشغّله.
func (s *Service) Enabled(ctx context.Context) bool { return s.settings.GetBool(ctx, KeyEnabled) }

// After **كم دقيقةً بعد آخر تغيير** — ولا تقلّ عن دقيقة.
func (s *Service) After(ctx context.Context) time.Duration {
	m := s.settings.GetInt(ctx, KeyAfterMinutes)
	if m < 1 {
		m = DefaultAfterMinutes
	}
	return time.Duration(m) * time.Minute
}

// quiet **ساعةُ الهدوء من الإعدادات** — سياسةُ الحملات نفسُها.
func (s *Service) quiet(ctx context.Context) campaigns.Quiet {
	return campaigns.Quiet{
		From: int(s.settings.GetInt(ctx, "notify.quiet_from")),
		To:   int(s.settings.GetInt(ctx, "notify.quiet_to")),
	}
}

// fingerprint **بصمةُ محتوى السلّة** — تُجزَّأ فلا يخزَّن ما يرسله العميلُ نصّاً.
func fingerprint(sig string) string {
	if len(sig) > maxSignature {
		sig = sig[:maxSignature]
	}
	sum := sha256.Sum256([]byte(sig))
	return hex.EncodeToString(sum[:16])
}

// Report **العميلُ يُبلغ حالَ سلّته** — `merchantID` متجرُها و`count` عددُ القطع و`signature`
// بصمةُ محتواها.
//
// **بلاغٌ بالمحتوى نفسِه لا يلمس الصفَّ** (إعادةُ فتح التطبيق): لا يُعيد المؤقّتَ ولا يمحو ختمَ
// التذكير. **وتغييرُ المحتوى** يُرجع `changed_at` إلى الآن ويمحو `reminded_at` — فتذكيرٌ جديدٌ ممكن.
// وعدٌّ غيرُ موجب إفراغٌ (`Clear`).
func (s *Service) Report(ctx context.Context, userID, merchantID string, count int, signature string) error {
	if count <= 0 {
		return s.Clear(ctx, userID)
	}
	merchantID = strings.TrimSpace(merchantID)
	if userID == "" || merchantID == "" {
		return ErrInvalid
	}
	fp := fingerprint(signature)
	_, err := s.db.Exec(ctx, `
		INSERT INTO cart_reminders (user_id, merchant_id, items_count, signature, changed_at)
		SELECT $1::uuid, m.id, $3, $4, now() FROM merchants m WHERE m.id = $2::uuid
		ON CONFLICT (user_id) DO UPDATE
		   SET merchant_id = EXCLUDED.merchant_id,
		       items_count = EXCLUDED.items_count,
		       signature   = EXCLUDED.signature,
		       changed_at  = now(),
		       reminded_at = NULL
		 WHERE cart_reminders.signature IS DISTINCT FROM EXCLUDED.signature
		    OR cart_reminders.merchant_id IS DISTINCT FROM EXCLUDED.merchant_id`,
		userID, merchantID, count, fp)
	if err != nil {
		// معرّفٌ ليس uuid ⇒ بلاغٌ مرفوض لا عطبٌ في الخادم.
		if strings.Contains(err.Error(), "22P02") {
			return ErrInvalid
		}
		return err
	}
	return nil
}

// Clear **سلّةٌ فُرّغت أو صارت طلباً** — يُحذف الصفُّ فلا تذكير.
func (s *Service) Clear(ctx context.Context, userID string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM cart_reminders WHERE user_id = $1::uuid`, userID)
	return err
}

// hit **تذكيرٌ التُقط وسيُرسَل.**
type hit struct {
	user, merchant, store string
}

// Due **يلتقط المستحقَّ ويختمه ويردّ ما يُرسَل** — ذرّيٌّ بجملةٍ واحدة.
//
// **المستحقّ:** لم يُذكَّر · مرّت `After` منذ آخر تغيير · لم يمرّ عليه أكثرُ من `staleAfter` ·
// **لم يطلب الزبونُ بعد ذلك التغيير** (ولو من عميلٍ لم يُبلغ) · حسابُه فعّالٌ · متجرُه فعّالٌ.
func (s *Service) Due(ctx context.Context) ([]hit, error) {
	now := s.now()
	rows, err := s.db.Query(ctx, `
		WITH due AS (
			SELECT r.user_id FROM cart_reminders r
			  JOIN users u ON u.id = r.user_id AND u.status = 'active'
			  JOIN merchants m ON m.id = r.merchant_id AND m.status = 'active'
			 WHERE r.reminded_at IS NULL
			   AND r.changed_at <= $1::timestamptz - make_interval(secs => $2::double precision)
			   AND r.changed_at >  $1::timestamptz - make_interval(secs => $3::double precision)
			   AND NOT EXISTS (SELECT 1 FROM orders o
			                    WHERE o.customer_id = r.user_id AND o.created_at >= r.changed_at)
			 ORDER BY r.changed_at
			 LIMIT $4
			 FOR UPDATE OF r SKIP LOCKED
		)
		UPDATE cart_reminders r SET reminded_at = $1
		  FROM due, merchants m
		 WHERE r.user_id = due.user_id AND m.id = r.merchant_id
		RETURNING r.user_id::text, r.merchant_id::text, m.name`,
		now, s.After(ctx).Seconds(), staleAfter.Seconds(), batch)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []hit
	for rows.Next() {
		var h hit
		if err := rows.Scan(&h.user, &h.merchant, &h.store); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// Sweep **جولةٌ واحدة** — وتردّ كم تذكيراً أُرسل.
//
// **مطفأةٌ ⇒ لا شيء** (ولا تلمس القاعدة). **وساعةُ الهدوء ⇒ لا شيء والصفوفُ تنتظر.**
func (s *Service) Sweep(ctx context.Context) int {
	if !s.Enabled(ctx) {
		return 0
	}
	if s.quiet(ctx).InQuiet(s.now()) {
		return 0
	}
	hits, err := s.Due(ctx)
	if err != nil {
		s.logger.Warn("cart reminder: due", "err", err)
		return 0
	}
	for _, h := range hits {
		s.notify.Notify(ctx, notifications.Input{
			UserID: h.user, Kind: campaigns.KindPromo,
			Title: textTitle, Body: textBody1 + h.store + textBody2,
			Entity: Entity, EntityID: h.merchant,
			Apps: []string{notifications.AppCustomer},
		})
	}
	return len(hits)
}

// Run **نبضةٌ دوريّة.**
func (s *Service) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Sweep(ctx)
		}
	}
}
