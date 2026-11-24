// Package prizes **«جوائز رحّال غو»** — مسابقةُ نقاطٍ شهريّةٌ ورصيدُ جوائز.
//
// كانت سحباً عشوائيّاً على الدعوات (قرارُ المالك ٢٠٢٦-١١-٢٣)، **وصارت لوحةَ نقاطٍ بلا حظّ**
// (قرارُ المالك ٢٠٢٦-١١-٢٥: «صاحب أكثر تفاعل يستاهل»).
//
// # النقاط (كلُّ قيمةٍ من الإعدادات `prizes.points_*`)
//
//	حضورٌ يوميّ «سجّل حضوري»            ←  مرّةً باليوم (يومُ دمشق)
//	طلبٌ مُسلَّمٌ للزبون نفسِه           ←  لكلّ طلب (عاديّ/خاصّ/أمانة)، ويسقط إن أُلغي التسليم
//	صديقٌ يفتح حساباً برابط دعوته         ←  لحظةَ إنشاء الحساب (`referrals.created_at`)
//	أوّلُ طلبٍ مُسلَّمٍ لذلك الصديق        ←  للداعي
//	متابعةُ فيسبوك / إنستغرام            ←  مرّةً في عمر الحساب، تُحسب في شهر الضغط
//
// **النقاطُ تُحسب من مصادرها في كلّ قراءة** (`points.go`) لا من عدّادٍ يُزاد — فإلغاءُ تسليمٍ
// (`delivered_at` يُمحى) يُسقط نقاطَه وحدَه بلا قيدٍ عكسيّ.
//
// # الشهرُ والحسم والإعلان (بتوقيت دمشق)
//
//	يومُ الحسم   = min(29, آخرُ يومٍ − 1) الساعةَ ١٢:٠٠  (فبراير ٢٨ يوماً ⇒ ٢٧، وإلّا ٢٩)
//	الإعلان      = آخرُ يومٍ في الشهر الساعةَ ١٢:٠٠
//	نافذةُ الشهر = من حسم الشهر الماضي إلى حسم هذا الشهر — فما بعد الحسم يُحسب للشهر التالي
//
// فالشهرُ الأوّلُ يبدأ فعليّاً أوّلَه ٠٠:٠٠ (يومُ الإطلاق `prizes.launch_on`)، **وكلُّ نشاطٍ بين
// الحسم وآخر الشهر يُعدّ للشهر الجاي.**
//
// # الفوز
//
// الأعلى نقاطاً أوّلاً، **والتعادلُ لمن وصل مجموعَه أبكر** (كلُّ النقاط موجبة، فلحظةُ الوصول
// = آخرُ حدثٍ عُدّ له). **ولا يفوز إلّا من بلغ `prizes.min_points`**، ورابحو الشهر الماضي
// خارجَ ترتيب هذا الشهر وحدَه. وأوّلُ `prizes.winners` بالجوائز `prizes.amount_1..5`.
//
// # والمال
//
// **المنحُ والانتهاءُ استحقاقٌ لا حركةُ مال** — لا قيدَ خزينةٍ ولا دفتر. **والصرفُ وحدَه
// مال**: يُكتب على الطلب خصماً تتحمّله المنصّة (`orders.prize_discount` داخلَ `discount`)
// كخصم الكود تماماً، فحسابُ الخزينة الفرقيُّ يقرؤه بلا قيدٍ جديد. **والرصيدُ ينتهي آخرَ الشهر
// الذي يلي يومَ الإعلان** (الإعلانُ آخرَ يومٍ في الشهر، فلا يُمنح رصيدٌ لساعات).
package prizes

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/offers"
	"github.com/servacode/rahalgo/backend/internal/pricing"
)

// Settings ما تقرؤه الحزمة.
//
// **وهي مخزنُ التسعير نفسُه** (`pricing.Store`) — فسعرُ صنف الجائزة يُحسب بالمعادلة التي تحسبه
// عند الطلب.
type Settings interface {
	pricing.Store
}

// Notifier **إبلاغُ الرابح وصاحب الرصيد** — بإشعار المحفظة القائم.
type Notifier interface {
	NotifyWallet(ctx context.Context, userID, title, body, href string)
}

// Service خدمةُ الجوائز.
type Service struct {
	db       *pgxpool.Pool
	settings Settings
	notify   Notifier
	logger   *slog.Logger
	// now **ساعةٌ تُبدَّل في الاختبار** — السحبُ مربوطٌ بأوّل الشهر.
	now func() time.Time
}

// New ينشئ الخدمة.
func New(db *pgxpool.Pool, st Settings, n Notifier, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{db: db, settings: st, notify: n, logger: logger, now: time.Now}
}

// SetClock **للاختبار وحدَه** — يثبّت «الآن».
func (s *Service) SetClock(now func() time.Time) {
	if now == nil {
		now = time.Now
	}
	s.now = now
}

// مفاتيحُ الإعدادات — في `settings/catalog.go`.
const (
	KeyEnabled   = "prizes.enabled"
	KeyLaunchOn  = "prizes.launch_on"
	KeyWinners   = "prizes.winners"
	KeyMinPoints = "prizes.min_points"
	// نقاطُ كلّ فعل.
	KeyPtsCheckin          = "prizes.points_checkin"
	KeyPtsOrder            = "prizes.points_order"
	KeyPtsInviteSignup     = "prizes.points_invite_signup"
	KeyPtsInviteFirstOrder = "prizes.points_invite_first_order"
	KeyPtsFollowFacebook   = "prizes.points_follow_facebook"
	KeyPtsFollowInstagram  = "prizes.points_follow_instagram"
	// **روابطُ الصفحتين من إعدادات «تواصل معنا» القائمة** — لا مفتاحَ ثانٍ لمعنًى واحد.
	KeyFacebookURL  = "platform.facebook"
	KeyInstagramURL = "platform.instagram"
	// DefaultLaunchOn **أوّلُ يومٍ تُعدّ فيه النقاط** — وما قبله لا يُعدّ أبداً.
	DefaultLaunchOn = "2026-12-01"
)

// PointValues **نقاطُ كلّ فعل** كما في الإعدادات — ولا سالبَ.
type PointValues struct {
	Checkin          int `json:"checkin"`
	Order            int `json:"order"`
	InviteSignup     int `json:"invite_signup"`
	InviteFirstOrder int `json:"invite_first_order"`
	FollowFacebook   int `json:"follow_facebook"`
	FollowInstagram  int `json:"follow_instagram"`
}

// Points **نقاطُ الأفعال الآن.**
func (s *Service) Points(ctx context.Context) PointValues {
	get := func(k string) int {
		n := int(s.settings.GetInt(ctx, k))
		if n < 0 {
			n = 0
		}
		return n
	}
	return PointValues{
		Checkin:          get(KeyPtsCheckin),
		Order:            get(KeyPtsOrder),
		InviteSignup:     get(KeyPtsInviteSignup),
		InviteFirstOrder: get(KeyPtsInviteFirstOrder),
		FollowFacebook:   get(KeyPtsFollowFacebook),
		FollowInstagram:  get(KeyPtsFollowInstagram),
	}
}

// MinPoints **أقلُّ مجموعٍ للتأهّل** — ولا سالب.
func (s *Service) MinPoints(ctx context.Context) int {
	n := int(s.settings.GetInt(ctx, KeyMinPoints))
	if n < 0 {
		n = 0
	}
	return n
}

// Enabled **المفتاحُ الرئيسيّ.**
func (s *Service) Enabled(ctx context.Context) bool { return s.settings.GetBool(ctx, KeyEnabled) }

// WinnersCount **كم رابحاً في الشهر** — ولا يقلّ عن واحد.
func (s *Service) WinnersCount(ctx context.Context) int {
	n := int(s.settings.GetInt(ctx, KeyWinners))
	if n < 1 {
		n = 1
	}
	return n
}

// Amounts **جائزةُ كلّ مرتبة** — وما بعد الخامسة جائزةُ الخامسة.
func (s *Service) Amounts(ctx context.Context) []int64 {
	n := s.WinnersCount(ctx)
	base := make([]int64, 5)
	for i := range base {
		base[i] = s.settings.GetInt(ctx, amountKey(i+1))
	}
	return amountsFor(base, n)
}

func amountKey(rank int) string {
	return "prizes.amount_" + string(rune('0'+rank))
}

// amountsFor **جوائزُ المراتب الأولى حتّى `n`** — وما بعد الخامسة كالخامسة.
func amountsFor(base []int64, n int) []int64 {
	out := make([]int64, n)
	for i := range out {
		if i < len(base) {
			out[i] = base[i]
		} else {
			out[i] = base[len(base)-1]
		}
	}
	return out
}

// LaunchOn **أوّلُ لحظةٍ تُعدّ فيها النقاط** — منتصفُ ليل اليوم بتوقيت دمشق.
func (s *Service) LaunchOn(ctx context.Context) (string, time.Time) {
	raw := strings.TrimSpace(s.settings.GetString(ctx, KeyLaunchOn))
	t, err := time.ParseInLocation("2006-01-02", raw, offers.Damascus)
	if err != nil {
		raw = DefaultLaunchOn
		t, _ = time.ParseInLocation("2006-01-02", raw, offers.Damascus)
	}
	return raw, t
}

// ── حسابُ الأشهر بتوقيت دمشق ────────────────────────────────────────────

// MonthStart **أوّلُ لحظةٍ من شهر `t`** بتوقيت دمشق.
func MonthStart(t time.Time) time.Time {
	d := t.In(offers.Damascus)
	return time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, offers.Damascus)
}

// NextMonth **أوّلُ الشهر التالي.**
func NextMonth(month time.Time) time.Time { return MonthStart(month).AddDate(0, 1, 0) }

// LastDay **آخرُ يومٍ في شهر `month`** (٢٨..٣١).
func LastDay(month time.Time) int { return NextMonth(month).AddDate(0, 0, -1).Day() }

// CutoffDay **يومُ الحسم** = min(29, آخرُ يومٍ − 1): ٢٩ عادةً، و٢٧ في فبراير ٢٨ يوماً،
// و٢٨ في فبراير الكبيس — فبين الحسم والإعلان يومٌ على الأقلّ.
func CutoffDay(month time.Time) int {
	d := LastDay(month) - 1
	if d > 29 {
		d = 29
	}
	return d
}

// CutoffAt **حسمُ الشهر** — يومُ الحسم الساعةَ ١٢ ظهراً بتوقيت دمشق. **ما قبله لهذا الشهر،
// وما بعده للشهر التالي.**
func CutoffAt(month time.Time) time.Time {
	m := MonthStart(month)
	return time.Date(m.Year(), m.Month(), CutoffDay(m), 12, 0, 0, 0, offers.Damascus)
}

// AnnounceAt **إعلانُ الشهر** — آخرُ يومٍ فيه الساعةَ ١٢ ظهراً بتوقيت دمشق.
func AnnounceAt(month time.Time) time.Time {
	m := MonthStart(month)
	return time.Date(m.Year(), m.Month(), LastDay(m), 12, 0, 0, 0, offers.Damascus)
}

// CompetitionMonth **أيُّ مسابقةٍ تُحسب لها لحظةُ `t`** — شهرُها، إلّا بعد حسمه فالتالي.
func CompetitionMonth(t time.Time) time.Time {
	m := MonthStart(t)
	if !t.Before(CutoffAt(m)) {
		return NextMonth(m)
	}
	return m
}

// CurrentMonth **المسابقةُ التي يراها الزبون** — الجاريةُ الآن، **وقبل الإطلاق مسابقةُ شهر
// الإطلاق** (فلا يُقال حسمُ شهرٍ لن يُعدّ).
func CurrentMonth(now, launch time.Time) time.Time {
	m := CompetitionMonth(now)
	if l := CompetitionMonth(launch); l.After(m) {
		m = l
	}
	return m
}

// ExpiryFor **آخرُ ثانيةٍ من الشهر الذي فيه `t`**.
func ExpiryFor(t time.Time) time.Time { return NextMonth(t).Add(-time.Second) }

// AwardExpiry **انتهاءُ جائزةٍ مُنحت في `t`** — آخرُ الشهر التالي: الإعلانُ آخرَ يومٍ في الشهر،
// **فلو انتهت آخرَ شهرها لعاشت ساعات.**
func AwardExpiry(t time.Time) time.Time { return ExpiryFor(NextMonth(t)) }

// MonthKey `YYYY-MM`.
func MonthKey(month time.Time) string { return month.In(offers.Damascus).Format("2006-01") }

// ParseMonth يقرأ `YYYY-MM` بتوقيت دمشق.
func ParseMonth(s string) (time.Time, bool) {
	t, err := time.ParseInLocation("2006-01", strings.TrimSpace(s), offers.Damascus)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// countWindow **نافذةُ نقاط الشهر** — من حسم الشهر الماضي (أو يوم الإطلاق، أيّهما أبعد) إلى
// حسم هذا الشهر. `ok=false` إن كانت النافذةُ كلُّها قبل الإطلاق.
func countWindow(month, launch time.Time) (from, to time.Time, ok bool) {
	m := MonthStart(month)
	from, to = CutoffAt(m.AddDate(0, -1, 0)), CutoffAt(m)
	if !launch.Before(to) {
		return from, to, false
	}
	if launch.After(from) {
		from = launch
	}
	return from, to, true
}
