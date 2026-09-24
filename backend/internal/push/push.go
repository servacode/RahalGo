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

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/servacode/rahalgo/backend/internal/obs"
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
	// ══════════════════════════════════════════════════════════════════
	// **Apps التطبيقاتُ التي يخصّها هذا الإشعار — وفارغةٌ تعني كلَّها**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-١١: «لازم يكون هناك فصلٌ بين العمليات
	//  والإشعارات».)
	//
	// **صاحبُ المتجر يحمل تطبيقين** — تطبيقَ متجرِه وتطبيقَ الزبون.
	// **و«طلبٌ جديد» يخصّ متجرَه وحدَه**، ولو ذهب إلى الاثنين لَرنّ في
	// تطبيقٍ لا علاقةَ له به.
	//
	// **والفارغةُ تعني كلَّ التطبيقات لا لا شيء**: أكثرُ الإشعارات لا
	// تُصرّح، **وافتراضُ الصمت يجعلها تختفي كلَّها دفعةً واحدةً بلا أن
	// يظهر شيءٌ في سجلّ.**
	Apps []string
}

// Service **الوجهُ الوحيدُ للمنصّة** — تسجيلٌ وإرسالٌ وتنظيف.
type Service struct {
	db     *pgxpool.Pool
	logger *slog.Logger
	// **ناقلٌ لكلّ منصّة** — فارغةٌ تعني «لا دفعَ مهيّأ».
	transports map[string]Transport

	warnOnce sync.Once

	// kick **خانةٌ واحدةٌ توقظ جولةَ نقل** — `PF-09`.
	//
	// **ولا طابور**: **ألفُ إشعارٍ في ثانيةٍ لا يُولّد ألفَ جولة**،
	// والجولةُ الواحدةُ تأخذ الدفعةَ كلَّها.
	kick chan struct{}
}

func New(db *pgxpool.Pool, logger *slog.Logger, transports ...Transport) *Service {
	m := make(map[string]Transport, len(transports))
	for _, t := range transports {
		if t != nil {
			m[t.Platform()] = t
		}
	}
	return &Service{db: db, logger: logger, transports: m,
		kick: make(chan struct{}, 1)}
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
// **وتسجيلٌ بجلسةٍ أُبطلت لا يُكتب** — `D12`.
//
// # السباقُ الذي قِيس
//
// **خروجٌ يتزامن مع إعادة تسجيل**: **قِيس ١٧ من ١٠٠ تبقى فيها
// الوجهةُ حيّةً بعد خروجٍ تامّ** — **يكتبها تسجيلٌ سبق الحذفَ
// بجزءٍ من الثانية.** **وخروجٌ ثانٍ لا يمحوها**: **رمزُ التجديد
// أُنفق، فيردّ البابُ نجاحاً صامتاً بلا حذف.**
//
// **فتبقى وجهةُ دفعٍ حيّةٌ لحسابٍ خرج** — **وهو `D12` بعينه في
// نافذةٍ ضيّقة.**
//
// # والقفلُ على صفوف الجلسة نفسِها
//
// **`FOR UPDATE` على عائلة هذه الجلسة** — **لا قفلَ عامّ**:
//
//	سبق التسجيلُ ⇒ الخروجُ ينتظر ثمّ يحذف ما كُتب
//	سبق الخروجُ  ⇒ التسجيلُ يرى `revoked_at` فيمتنع
//
// **وكلتا الحالتين تنتهيان بلا وجهةٍ لحسابٍ خرج.**
//
// **وجلسةٌ فارغةٌ تمرّ كما كانت** — **نسخةٌ قديمةٌ أو رمزٌ بلا
// عائلة**: **ولا يُكسَر تسجيلُها لأجل حارسٍ يُضاف.**
func (s *Service) Register(ctx context.Context, userID, token, platform, app, appVersion string) error {
	return s.RegisterForSession(ctx, userID, "", token, platform, app, appVersion)
}

// RegisterForSession **تسجيلٌ مربوطٌ بحياة عائلة الجلسة** — `D12`.
func (s *Service) RegisterForSession(ctx context.Context, userID, sessionID, token, platform, app, appVersion string) error {
	if s == nil || token == "" || userID == "" {
		return nil
	}
	if sessionID == "" {
		return s.insert(ctx, s.db, userID, "", token, platform, app, appVersion)
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var alive int
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM (
			SELECT 1 FROM refresh_tokens
			 WHERE session_id = $1::uuid AND revoked_at IS NULL AND expires_at > now()
			 FOR UPDATE
		) t`, sessionID).Scan(&alive); err != nil {
		return err
	}
	if alive == 0 {
		// **جلسةٌ أُبطلت** — **ولا يُكتب لها هدف**، **ولا يُردّ
		// خطأً**: النداءُ نفسُه سيُردّ ٤٠١ في مرّته التالية،
		// **والصمتُ هنا لا يُخفي شيئاً.**
		return tx.Commit(ctx)
	}
	if err := s.insert(ctx, tx, userID, sessionID, token, platform, app, appVersion); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func (s *Service) insert(ctx context.Context, q execer, userID, sessionID, token, platform, app, appVersion string) error {
	_, err := q.Exec(ctx, `
		INSERT INTO device_tokens (token, user_id, session_id, platform, app, app_version)
		VALUES ($1, $2, nullif($3, '')::uuid, $4, $5, $6)
		ON CONFLICT (token) DO UPDATE
		SET user_id = EXCLUDED.user_id,
		    -- والمِلكيّةُ تنتقل مع الحساب (D12): هاتفٌ سُلّم لغيره
		    -- يُسجّله باسمٍ جديد وعائلةٍ جديدة، فلا يبقى هدفاً
		    -- لمن سبق.
		    session_id = EXCLUDED.session_id,
		    platform = EXCLUDED.platform,
		    app = EXCLUDED.app,
		    app_version = EXCLUDED.app_version,
		    last_seen_at = now()`,
		token, userID, sessionID, normalizePlatform(platform), app, appVersion)
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

// SendToUser **دفعٌ فوريٌّ غيرُ دائم** — **وليس مسارَ التسليم.**
//
// ══════════════════════════════════════════════════════════════════════
// **ولا يُنادى من `notifications` بعد اليوم** — `PF-09`
// ══════════════════════════════════════════════════════════════════════
//
// **مرّةً واحدةً بلا إعادةٍ ولا أثر**: **سقطت الشبكةُ فضاع التنبيهُ
// ولا سطرَ يقول ذلك.** **وذاك `PF-09` بعينه.**
//
// **ومسارُ التسليم اليوم**: علامةٌ في صفّ الإشعار ⇒ صفوفُ نقلٍ لكلّ
// هدف ⇒ عاملٌ يُعيد بتراجعٍ محدود (`delivery.go`).
//
// **وتبقى هذه لقراءة الأجهزة وترشيح التطبيقات وحذف الميّت** — **وهي
// مقيسةٌ بفحوصها**، **ولا تُنادى في مسارِ إشعار.**
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

	byPlatform, err := s.tokensOf(ctx, userID, msg.Apps)
	if err != nil {
		s.logger.Error("الدفع: تعذّرت قراءةُ الأجهزة", "user", userID, "error", err)
		return
	}
	// ══════════════════════════════════════════════════════════════
	// **وإشعارٌ لا يصل لا يشتكي منه أحد** — عدُّ الحصيلة (دورة ٧٠أ)
	// ══════════════════════════════════════════════════════════════
	//
	// **والسائقُ يظنّ أنّه لا طلبات، والمكتبُ يظنّه كسولاً.** **وفحصُ
	// `Diagnose` يقول عن حسابٍ بعينه**، **ولا شيءَ يقول عن المنصّة:
	// «كم حاولنا وكم وصل».**
	//
	// **وصفرُ أجهزةٍ ليس عطباً** — حسابٌ لم يُفتح تطبيقُه بعد الدخول،
	// **فيُعدّ على حدة ولا يُخلَط بالفشل.**
	if len(byPlatform) == 0 {
		obs.Push(obs.PushNoDevice, 1)
		return
	}
	for platform, tokens := range byPlatform {
		t, ok := s.transports[platform]
		if !ok {
			continue
		}
		obs.Push(obs.PushAttempted, len(tokens))
		dead, err := t.Send(ctx, tokens, msg)
		if err != nil {
			obs.Push(obs.PushFailed, len(tokens))
			s.logger.Error("الدفع: تعذّر الإرسال", "platform", platform,
				"devices", len(tokens), "error", err)
		} else {
			// **والمقبولُ ما لم يُردّ رمزُه** — **وغوغل تقبل الدفعةَ
			// وتردّ فيها رموزاً بعينها.** **ولا تقول أكثرَ من ذلك**،
			// فلا يُدَّعى «وصل إلى الجهاز»: **القبولُ عند المزوّد
			// ليس رنيناً في جيب.**
			obs.Push(obs.PushSent, len(tokens)-len(dead))
			obs.Push(obs.PushDeadToken, len(dead))
		}
		s.dropDead(ctx, dead)
	}
}

// SendToTokens **يرسل إلى رموزٍ بعينها مباشرةً** — لا عبر جدول الأجهزة (Obs 3.1).
//
// **لإشعارِ أمانٍ لمرّةٍ واحدةٍ إلى رموزٍ أُزيلت توّاً**: الرمزُ حُذف من
// `device_tokens` فلا يُختار بالمستخدم في دفعٍ خاصٍّ بعد الآن، **ويُرسَل إليه هنا
// بالسلسلةِ مباشرةً** — **ولا يُعاد إدراجُه** (لا كتابةَ جدول). الرموزُ لأجهزةِ
// أندرويد (نوعُ العميل المُزاح `android-*`). **الفشلُ يُسجَّل ولا يُرمى** فلا
// يعرقل نداءَ الإبطال/الدخول.
func (s *Service) SendToTokens(ctx context.Context, tokens []string, msg Message) {
	if s == nil || len(tokens) == 0 || !s.Enabled() {
		return
	}
	t, ok := s.transports[PlatformAndroid]
	if !ok {
		return
	}
	obs.Push(obs.PushAttempted, len(tokens))
	if _, err := t.Send(ctx, tokens, msg); err != nil {
		obs.Push(obs.PushFailed, len(tokens))
		s.logger.Error("الدفع: تعذّر إشعارُ الإزاحة المباشر", "devices", len(tokens), "error", err)
		return
	}
	obs.Push(obs.PushSent, len(tokens))
}

// tokensOf أجهزةُ الحساب مجموعةً بمنصّتها — **والميّتُ بالزمن لا يُقرأ.**
//
// **و`apps` فارغةٌ تعني كلَّ الأجهزة.** وإن حُدّدت، **يبقى الجهازُ المجهولُ
// تطبيقُه (`app = ”`) داخلاً في كلّ حال**: جهازٌ سُجّل من نسخةٍ قديمةٍ لا
// تُصرّح **يجب أن يستقبل لا أن يصمت** — إشعارٌ زائدٌ أهونُ من إشعارٍ مفقود.
func (s *Service) tokensOf(ctx context.Context, userID string, apps []string) (map[string][]string, error) {
	rows, err := s.db.Query(ctx, `
		SELECT platform, token FROM device_tokens
		WHERE user_id = $1 AND last_seen_at > now() - $2::interval
		  AND ($3::text[] IS NULL OR app = '' OR app = ANY($3::text[]))`,
		userID, staleAfter.String(), nilIfEmpty(apps))
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

// nilIfEmpty **قائمةٌ فارغةٌ تصير `NULL`** — لتقرأها الجملةُ «بلا ترشيح».
//
// **ومصفوفةٌ فارغةٌ في `= ANY` لا تطابق شيئاً** — فلو مُرّرت كما هي
// **لَصمتت كلُّ الإشعارات التي لا تُصرّح بتطبيقها**، وهي أكثرُها.
func nilIfEmpty(v []string) []string {
	if len(v) == 0 {
		return nil
	}
	return v
}
