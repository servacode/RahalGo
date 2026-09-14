package orders

// ══════════════════════════════════════════════════════════════════════
// **وقتُ المنطقة — طبقةٌ فوقَ الجغرافيا لا بديلٌ عنها** (`ZH`)
// ══════════════════════════════════════════════════════════════════════
//
// # ولمَ رمزٌ جديدٌ لا `out_of_zone`
//
// **وعنوانٌ في قلب الحيّ المخدوم ليس خارجَ التغطية** — **والساعةُ هي
// المانع.** **ومن قرأ «عنوانُك خارجَ منطقة التوصيل» حكم على المنصّة
// أنّها لا تصله أبداً فحذف التطبيق** — **ومن قرأ «يعود التوصيل الثامنةَ
// صباحاً» عاد.**
//
// **والرمزُ عقدٌ تقرؤه الشاشة** — **فرمزٌ واحدٌ لحالين يمنعها أن
// تفرّق.**
//
// # وترتيبُ الأسبقيّة محفوظ
//
//	وضعُ الإطلاق      ⇒ `launch_closed`             (المعالِج)
//	إيقافٌ مؤقّت       ⇒ `temporarily_unavailable`   (المعالِج)
//	دوامُ المنصّة      ⇒ `platform_closed_now`       (المعالِج)
//	نقطةٌ مشوَّهة       ⇒ `bad_point`                 (`RequireServiceable`)
//	لا تغطيةَ صالحة   ⇒ `coverage_unavailable`      (`ZoneAt`)
//	خارجَ الأشكال     ⇒ `out_of_zone`               (`ZoneAt`)
//	**ثمّ** وقتُ المنطقة ⇒ `zone_closed_now`           **هنا**
//
// **وهذا آخرُها بقصد** — **فلا يحجب وقتُ المنطقة سبباً أسبقَ منه**:
// **من كان خارجَ التغطية يُقال له ذلك، لا «يعود التوصيل الثامنة» إلى
// منطقةٍ ليس فيها.**
//
// # ولمَ يُسأل عن معرّف المنطقة التي سُعِّرت
//
// **و`ZoneAt` تختار منطقةً بعينها عند التداخل** (أقربُها مركزاً) —
// **ونداءٌ ثانٍ يسأل «أيُّ منطقةٍ تحوي النقطة» قد يقع على غيرها.**
// **فيُسعَّر بمنطقةٍ ويُقاس وقتُ أخرى** — **ويُردّ الزبونُ بجدولٍ لا
// يخصّه.**
//
// **فالرايةُ تُقرأ في الصفّ نفسِه** (`ZoneCharge.HoursEnforced`).

import (
	"context"
	"net/http"
	"time"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ErrZoneClosedNow **المنطقةُ مُغطّاةٌ ولا يُوصَّل إليها الآن.**
//
// **و٥٠٣ لا ٤٠٠**: **«ليس الآن» لا «طلبُك خطأ»** — **وهو رمزُ
// `coverage_unavailable` و`platform_closed_now` نفسُه**: عائلةُ «الخدمةُ
// غيرُ متاحةٍ مؤقّتاً». **و`out_of_zone` يبقى ٤٠٠** لأنّه حكمٌ على
// العنوان لا على الوقت.
var ErrZoneClosedNow = httpx.NewError(http.StatusServiceUnavailable,
	"zone_closed_now", "errors.zone_closed_now")

// ZoneHours **ما تحتاجه بوّابةُ القبول من آلة الجداول.**
//
// **وواجهةٌ ضيّقةٌ لا حزمةٌ تُستورَد** — **ووحدةٌ تأخذ كلَّ شيءٍ تستطيع
// أن تفعل كلَّ شيء.**
type ZoneHours interface {
	// ZoneOpen **أيُوصَّل إلى هذه المنطقة الآن؟**
	//
	// **ويُعاد موعدُ أوّلِ لحظةٍ يُقبَل فيها طلبٌ إليها** — **تقاطعَ
	// المنصّةِ والمنطقة**، **لا موعدَ المنطقة وحدَها**: **ومن قيل له
	// «يعود التوصيل الثالثة» والمنصّةُ لا تستقبل حتّى الخامسة عاد
	// فوجد البابَ مغلقاً، ولا يعود ثالثةً.**
	//
	// **ولا يُحسَب الموعدُ إلّا عند المنع** — **فمسارُ القبول لا يدفع
	// ثمنَ سؤالٍ لا يُعرَض.**
	ZoneOpen(ctx context.Context, q dbtx.Querier, zoneID string) (open bool, nextAt *time.Time, err error)
}

// SetZoneHours **يركّب آلةَ الجداول** — **ونظيرُ `SetOffers` و`SetRouter`.**
func (s *Service) SetZoneHours(h ZoneHours) { s.zoneHours = h }

// requireZoneOpen **يردّ الطلبَ إن كانت المنطقةُ خارجَ وقتها.**
//
// **ولا تُسأل منطقةٌ لا جدولَ ساريَ لها** — **وهي كلُّ منطقةٍ قائمةٍ
// اليوم**: **فلا رحلةَ إلى القاعدة ولا تبدّلَ في سلوكٍ قائم.**
//
// **ومعرّفٌ فارغٌ حالُ «لم تُرسم خريطةٌ بعد»** — **ولا جدولَ لما ليس
// منطقةً.**
//
// **وآلةٌ غيرُ مركَّبةٍ لا تمنع** — **وفحصٌ يبني الخدمةَ بلا هذه الآلة
// يبقى يقيس ما كان يقيس**، **ولا يُردّ طلبٌ لأنّ تركيباً نقص.**
func (s *Service) requireZoneOpen(ctx context.Context, q dbtx.Querier, z ZoneCharge) error {
	if !z.HoursEnforced || z.ID == "" || s.zoneHours == nil {
		return nil
	}
	open, nextAt, err := s.zoneHours.ZoneOpen(ctx, q, z.ID)
	if err != nil {
		// **وعطبُ قراءةٍ ليس حكماً** — **ولا يُردّ طلبٌ لأنّ السؤالَ
		// تعذّر.** **والجغرافيا حُسمت قبله، والمنصّةُ مفتوحةٌ فوقَه.**
		return nil
	}
	if open {
		return nil
	}
	e := *ErrZoneClosedNow
	if nextAt != nil {
		// **والموعدُ يُرسَل ليُعرَض** — **و«غيرُ متاح» طريقٌ مسدودٌ،
		// و«يعود الثامنة» موعدٌ يُعاد إليه.**
		e.Details = map[string]any{"next_available_at": nextAt.Format(time.RFC3339)}
	}
	return &e
}

// quoteZoneTime **يكتب حالَ وقتِ المنطقة في التسعيرة.**
//
// **والتسعيرةُ تُخبِر ولا تنهار** — **وسلّةٌ تسقط لأنّ الوقتَ انتهى
// سلّةٌ لا تُستعمل.** **فيُعطَّل زرُّ الإتمام ويُقال السبب، والرسمُ
// يبقى معروضاً لأنّ العنوانَ مُغطّىً والرقمَ صحيح.**
//
// **وعطبُ قراءةٍ لا يُغلق شيئاً هنا** — **والمحرّكُ يردّ الطلبَ عند
// الإنشاء إن كان مغلقاً على كلّ حال.**
func (s *Service) quoteZoneTime(ctx context.Context, out *QuoteResult, z ZoneCharge) {
	if !z.HoursEnforced || z.ID == "" || s.zoneHours == nil {
		return
	}
	open, nextAt, err := s.zoneHours.ZoneOpen(ctx, s.db, z.ID)
	if err != nil || open {
		return
	}
	out.Serviceable = false
	out.ZoneClosed = true
	out.ServiceableReason = "zone_closed_now"
	if nextAt != nil {
		out.NextAvailableAt = nextAt.Format(time.RFC3339)
	}
}
