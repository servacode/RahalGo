// Package prizes **«جوائز رحّال غو»** — سحبٌ شهريٌّ على دعوات الزبائن ورصيدُ جوائز
// (قرارُ المالك ٢٠٢٦-١١-٢٣).
//
// # الأسئلةُ الخمسة
//
//	متى تُعدّ الدعوة؟  ←  لحظةَ إنشاء حساب المدعوّ (`referrals.created_at`)
//	كم فرصة؟         ←  كلُّ `prizes.invites_per_entry` دعوةً في الشهر فرصة
//	متى السحب؟        ←  أوّلَ الشهر ١٢ ظهراً بتوقيت دمشق، على الشهر الماضي
//	كم يربح؟         ←  بالمرتبة: `prizes.amount_1..5` (وما بعد الخامس كالخامس)
//	أين يُصرف؟        ←  متجرُ الجوائز وحدَه، **وينتهي آخرَ الشهر الذي مُنح فيه**
//
// # والمال
//
// **المنحُ والانتهاءُ استحقاقٌ لا حركةُ مال** — لا قيدَ خزينةٍ ولا دفتر. **والصرفُ وحدَه
// مال**: يُكتب على الطلب خصماً تتحمّله المنصّة (`orders.prize_discount` داخلَ `discount`)
// كخصم الكود تماماً، فحسابُ الخزينة الفرقيُّ يقرؤه بلا قيدٍ جديد.
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
	KeyEnabled        = "prizes.enabled"
	KeyLaunchOn       = "prizes.launch_on"
	KeyInvitesPerEntr = "prizes.invites_per_entry"
	KeyWinners        = "prizes.winners"
	// DefaultLaunchOn **أوّلُ يومٍ تُعدّ فيه الدعوات** — وما قبله لا يُعدّ أبداً.
	DefaultLaunchOn = "2026-12-01"
)

// Enabled **المفتاحُ الرئيسيّ.**
func (s *Service) Enabled(ctx context.Context) bool { return s.settings.GetBool(ctx, KeyEnabled) }

// PerEntry **كم دعوةً للفرصة الواحدة** — ولا يقلّ عن واحد.
func (s *Service) PerEntry(ctx context.Context) int {
	n := int(s.settings.GetInt(ctx, KeyInvitesPerEntr))
	if n < 1 {
		n = 1
	}
	return n
}

// WinnersCount **كم رابحاً في السحب** — ولا يقلّ عن واحد.
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

// LaunchOn **أوّلُ لحظةٍ تُعدّ فيها الدعوات** — منتصفُ ليل اليوم بتوقيت دمشق.
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

// DrawAt **موعدُ سحب الشهر** — أوّلُ الشهر التالي الساعةَ ١٢ ظهراً بتوقيت دمشق.
func DrawAt(month time.Time) time.Time { return NextMonth(month).Add(12 * time.Hour) }

// ExpiryFor **آخرُ ثانيةٍ من الشهر الذي فيه `t`** — انتهاءُ رصيدٍ مُنح فيه.
func ExpiryFor(t time.Time) time.Time { return NextMonth(t).Add(-time.Second) }

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

// countWindow **نافذةُ عدّ الشهر** — من أوّله أو من يوم الإطلاق أيّهما أبعد، إلى أوّل التالي.
// `ok=false` إن كان الشهرُ كلُّه قبل الإطلاق.
func countWindow(month, launch time.Time) (from, to time.Time, ok bool) {
	from, to = MonthStart(month), NextMonth(month)
	if !launch.Before(to) {
		return from, to, false
	}
	if launch.After(from) {
		from = launch
	}
	return from, to, true
}

// EntriesFor **فرصُ عددٍ من الدعوات** — ٥←١، ١٠←٢، ١٥←٣.
func EntriesFor(invites, perEntry int) int {
	if perEntry < 1 {
		perEntry = 1
	}
	if invites < 0 {
		return 0
	}
	return invites / perEntry
}

// ToNext **كم دعوةً بقيت للفرصة التالية.**
func ToNext(invites, perEntry int) int {
	if perEntry < 1 {
		perEntry = 1
	}
	if invites < 0 {
		invites = 0
	}
	return perEntry - invites%perEntry
}

// DisplayName **الاسمُ الأوّلُ وأوّلُ حرفٍ من الأخير ونقطة** — «أحمد م.». ولا رقمَ أبداً.
func DisplayName(full string) string {
	parts := strings.Fields(full)
	switch len(parts) {
	case 0:
		return "زبون"
	case 1:
		return parts[0]
	}
	last := []rune(parts[len(parts)-1])
	return parts[0] + " " + string(last[0]) + "."
}
