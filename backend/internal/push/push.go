// Package push **الدفعُ إلى الأجهزة — لِما يصل والشاشةُ مقفلة.**
//
// ══════════════════════════════════════════════════════════════════════
// **ولماذا حزمةٌ مستقلّةٌ عن `notifications`**
// ══════════════════════════════════════════════════════════════════════
//
// **`notifications` تقرّر مَن يُبلَّغ وبماذا** — وهي منطقُ عمل. **وهذه
// تقرّر كيف يصل** — وهي نقل. **ومن خلطهما ربط منطقَ المنصّة بجوجل**، فصار
// تبديلُ ناقلٍ يمسّ كلَّ موضعٍ يُنشئ إشعاراً.
//
// **والفصلُ ليس نظريّاً**: آيفون يحتاج APNs، وقد نضيف ناقلاً ذاتيّاً
// لأجهزةٍ بلا خدمات جوجل. **وكلُّها تدخل من هذا الباب وحدَه.**
//
// # وعهدُ هذه الحزمة: لا تُفشل شيئاً أبداً
//
// **هو عهدُ `notifications` نفسُه.** فشلُ الدفع يُسجَّل ولا يُرجَع خطأً:
// **وذعرٌ يُسقط النداءَ كلَّه أشدُّ من إشعارٍ لم يصل** — يُبطل الطلبَ الذي
// أُنشئ والانتقالَ الذي وقع، **ويردّ خمسمئة على فعلٍ نجح.**
//
// # وبلا إعدادٍ تصمت ولا تسقط
//
// **المنصّةُ تعمل اليوم بلا دفع**، وستعمل غداً على جهاز مطوّرٍ بلا مفتاح.
// **فالخدمةُ غيرُ المهيّأة تُسجّل مرّةً واحدةً وتمرّ** — ولا تملأ السجلَّ
// بسطرٍ عند كلّ حدث.
package push

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// المنصّاتُ المعروفة — **والقائمةُ مغلقةٌ** كما نوعُ الجلسة.
const (
	PlatformAndroid = "android"
	PlatformIOS     = "ios"
)

// staleAfter **رمزٌ لم يُرَ منذ هذه المدّة ميّتٌ عمليّاً.**
//
// **وشهران لا أسبوع**: سائقٌ في إجازةٍ أو هاتفٌ في التصليح يعود، **ومن
// حُذف رمزُه لا يصله شيءٌ حتّى يفتح التطبيق** — وهو لن يفتحه لأنّه لم
// يصله شيء.
const staleAfter = 60 * 24 * time.Hour

// Transport **ناقلٌ إلى منصّةٍ واحدة** — FCM اليوم، وAPNs غداً.
//
// **يعيد الرموزَ الميّتة** ليحذفها المستدعي: **رمزٌ مرفوضٌ يبقى في الجدول
// يُنادى عليه في كلّ حدثٍ إلى الأبد.**
type Transport interface {
	// Send يرسل إلى رموزٍ من منصّته، ويعيد ما رفضته المنصّةُ نهائيّاً.
	Send(ctx context.Context, tokens []string, msg Message) (dead []string, err error)
	// Platform المنصّةُ التي يخدمها.
	Platform() string
}

// Message **ما يُدفع** — عنوانٌ ونصٌّ وحمولةٌ صغيرة.
type Message struct {
	Title string
	Body  string
	// Data **حمولةٌ يقرؤها التطبيقُ ليعرف أين يذهب** — وهي مفاتيحُ لا نصوص
	// (`kind` و`entity` و`entity_id`). **ولا تحمل سرّاً**: تمرّ بخوادم
	// جوجل وتُخزَّن حتّى يستيقظ الجهاز.
	Data map[string]string
	// Urgent **يوقظ الجهازَ من سباته.**
	//
	// **وليست زينة**: أندرويد في `Doze` يؤجّل الرسائلَ العاديّةَ إلى نافذةِ
	// صيانةٍ قد تبعد ساعة. **والأولويّةُ العاليةُ تخترق ذلك** — وهي
	// المخصّصةُ لطلبٍ ينتظر سائقاً.
	//
	// **ولا تُرفع لكلّ شيء**: جوجل تخفض حصّةَ من يُسيء استعمالها،
	// **فتتأخّر رسائلُه كلُّها** — بما فيها العاجل.
	Urgent bool
}

// Service **الوجهُ الوحيدُ للمنصّة** — تسجيلٌ وإرسالٌ وتنظيف.
type Service struct {
	db     *pgxpool.Pool
	logger *slog.Logger
	// **ناقلٌ لكلّ منصّة** — فارغةٌ تعني «لا دفعَ مهيّأ».
	transports map[string]Transport

	warnOnce sync.Once
}

func New(db *pgxpool.Pool, logger *slog.Logger, transports ...Transport) *Service {
	m := make(map[string]Transport, len(transports))
	for _, t := range transports {
		if t != nil {
			m[t.Platform()] = t
		}
	}
	return &Service{db: db, logger: logger, transports: m}
}

// Enabled **أثمّة ناقلٌ مهيّأ؟** — تقرؤها الصحّةُ والإقلاع.
func (s *Service) Enabled() bool { return s != nil && len(s.transports) > 0 }

// normalizePlatform يردّ المجهولَ إلى أندرويد — **ولا يرفض التسجيل.**
//
// **ونسخةٌ من التطبيق تكتب اسماً تبدّل خيرٌ لها أن تُسجَّل خطأً من ألّا
// تُسجَّل**: الأولى تُرسل إلى ناقلٍ لا يعرفها فيردّ الرمزَ ميّتاً فيُحذف،
// **والثانيةُ تترك السائقَ بلا إشعارٍ ولا أحدَ يعلم.**
func normalizePlatform(p string) string {
	if p == PlatformIOS {
		return PlatformIOS
	}
	return PlatformAndroid
}

// Register يُسجّل رمزَ جهازٍ لصاحبه — **ويُنادى في كلّ إقلاعٍ للتطبيق.**
//
// **والرمزُ ينتقل إلى آخرِ من سجّله**: هاتفٌ يُسلَّم لسائقٍ آخرَ يجب ألّا
// يبقى يستقبل إشعاراتِ الأوّل. **و`ON CONFLICT (token)` تفعل ذلك بنداءٍ
// واحدٍ بلا سباق.**
func (s *Service) Register(ctx context.Context, userID, token, platform, appVersion string) error {
	if s == nil || token == "" || userID == "" {
		return nil
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO device_tokens (token, user_id, platform, app_version)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (token) DO UPDATE
		SET user_id = EXCLUDED.user_id,
		    platform = EXCLUDED.platform,
		    app_version = EXCLUDED.app_version,
		    last_seen_at = now()`,
		token, userID, normalizePlatform(platform), appVersion)
	return err
}

// Unregister يحذف رمزاً — **عند الخروج من التطبيق.**
//
// **ومقيَّدٌ بصاحبه**: لولا `user_id` لَاستطاع أيُّ داخلٍ أن يُسكت إشعاراتِ
// غيره برمزٍ يعرفه.
func (s *Service) Unregister(ctx context.Context, userID, token string) error {
	if s == nil || token == "" {
		return nil
	}
	_, err := s.db.Exec(ctx,
		`DELETE FROM device_tokens WHERE token = $1 AND user_id = $2`, token, userID)
	return err
}

// SendToUser يدفع إلى كلّ أجهزة الحساب — **ولا يُفشل شيئاً أبداً.**
func (s *Service) SendToUser(ctx context.Context, userID string, msg Message) {
	if s == nil || userID == "" {
		return
	}
	if !s.Enabled() {
		// **مرّةً واحدةً في عمر العمليّة** — سطرٌ عند كلّ حدثٍ يُغرق السجلّ
		// حتّى يختفي فيه ما يُقرأ.
		s.warnOnce.Do(func() {
			s.logger.Info("الدفع: غيرُ مهيّأ — الإشعاراتُ تصل عبر البثّ الحيّ وحدَه")
		})
		return
	}

	byPlatform, err := s.tokensOf(ctx, userID)
	if err != nil {
		s.logger.Error("الدفع: تعذّرت قراءةُ الأجهزة", "user", userID, "error", err)
		return
	}
	for platform, tokens := range byPlatform {
		t, ok := s.transports[platform]
		if !ok {
			continue
		}
		dead, err := t.Send(ctx, tokens, msg)
		if err != nil {
			s.logger.Error("الدفع: تعذّر الإرسال", "platform", platform,
				"devices", len(tokens), "error", err)
		}
		s.dropDead(ctx, dead)
	}
}

// tokensOf أجهزةُ الحساب مجموعةً بمنصّتها — **والميّتُ بالزمن لا يُقرأ.**
func (s *Service) tokensOf(ctx context.Context, userID string) (map[string][]string, error) {
	rows, err := s.db.Query(ctx, `
		SELECT platform, token FROM device_tokens
		WHERE user_id = $1 AND last_seen_at > now() - $2::interval`,
		userID, staleAfter.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]string{}
	for rows.Next() {
		var platform, token string
		if rows.Scan(&platform, &token) == nil {
			out[platform] = append(out[platform], token)
		}
	}
	return out, rows.Err()
}

// dropDead يحذف ما رفضته المنصّةُ نهائيّاً.
//
// **ورمزٌ مرفوضٌ يبقى يُنادى عليه في كلّ حدثٍ إلى الأبد** — يستهلك حصّةً
// ويبطئ كلَّ إشعارٍ لصاحبه.
func (s *Service) dropDead(ctx context.Context, dead []string) {
	if len(dead) == 0 {
		return
	}
	if _, err := s.db.Exec(ctx,
		`DELETE FROM device_tokens WHERE token = ANY($1)`, dead); err != nil {
		s.logger.Warn("الدفع: تعذّر حذفُ رموزٍ ميّتة", "count", len(dead), "error", err)
		return
	}
	s.logger.Info("الدفع: حُذفت رموزٌ ميّتة", "count", len(dead))
}
