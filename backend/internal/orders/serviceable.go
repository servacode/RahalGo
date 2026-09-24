package orders

import (
	"context"
	"math"
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ══════════════════════════════════════════════════════════════════════
// **نقطةُ التسليم — وهي وحدَها تقرّر** (`SRV`، ٢٠٢٦-٠٩-١٣)
// ══════════════════════════════════════════════════════════════════════
//
// # العقد
//
// **وإحداثيّةُ العنوان المختارِ لهذا الطلب هي الحاكمة** — **لا موضعُ
// الجهاز، ولا مدينةُ الحساب، ولا مدينةُ المتجر، ولا نتيجةُ تصفّحٍ
// قديمة.**
//
// # ولماذا يُعاد الفحصُ عند الإنشاء
//
// **والتغطيةُ تتبدّل** · **والعنوانُ يتبدّل** · **والسلّةُ تشيخ** ·
// **والعميلُ يُعدَّل** · **ونسخةٌ قديمةٌ تلتفّ على الشاشة.**
//
// **فما أظهرته الشاشةُ قبل دقيقةٍ ليس حكماً** — **والحكمُ يُعاد في
// معاملةِ الإنشاء نفسِها.**
//
// # وموضعُ القرار واحد
//
// **و`ZoneAt` هي الحاكمة** (`sources.go`): **الدائرةُ بـ`ST_DWithin`
// والمضلَّعُ بـ`ST_Covers`**، **وجدولٌ فارغٌ يُقرأ «لم تُرسم خريطةٌ
// بعد» لا «لا نُوصّل إلى أحد»** (قرارُ المالك ٢٠٢٦-٠٨-١٨).
//
// **ولا فحصَ مضلَّعٍ ثانٍ في معالِج** — **وقاعدتان تفترقان يومَ تُبدَّل
// إحداهما.** **وقِيس ٢٠٢٦-٠٩-١٣: `catalog.ZoneForPoint` كانت تفحص
// الدائرةَ وحدَها ولا تقرأ المضلَّع** — **فلو نُوديت لَقالت «خارجَ
// التغطية» لنقطةٍ داخلَ مضلَّعٍ مرسوم.** (حُذفت، ولم يكن لها مستعمل.)

// ErrBadPoint **إحداثيّةٌ ليست إحداثيّة.**
//
// **وهي غيرُ `no_address`**: **ذاك عنوانٌ غائب، وهذا عنوانٌ حاضرٌ
// بنقطةٍ لا معنى لها** — صفرٌ صفرٌ، أو خطُّ عرضٍ فوق التسعين.
//
// **وصفرٌ صفرٌ نقطةٌ في المحيط الأطلسيّ** — **ولو مرّت لَقُبل طلبٌ
// إلى ماء.**
var ErrBadPoint = httpx.NewError(http.StatusBadRequest,
	"bad_point", "errors.bad_point")

// ValidPoint **أهذه إحداثيّةٌ يصلح أن يُوصَّل إليها؟**
//
// **وموضعٌ واحدٌ يقرؤه الجميع** — **وكان الفحصُ مكتوباً في خمسة
// مواضعَ حرفاً حرفاً** (مدينةٌ · موضعُ سائقٍ · حزمتُه · مسارُه) —
// **ولا واحدٌ منها في إنشاء الطلب.**
//
// **والصفرُ المزدوجُ يُردّ صراحةً**: **هو ما يُرسله عميلٌ لم يُحدَّد
// موضعُه بعد**، **وهو نقطةٌ صالحةٌ حسابيّاً في خليج غينيا** — فيمرّ
// فحصَ المدى ويُقبل طلبٌ إلى ماء.
func ValidPoint(lat, lng float64) bool {
	if math.IsNaN(lat) || math.IsNaN(lng) ||
		math.IsInf(lat, 0) || math.IsInf(lng, 0) {
		return false
	}
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return false
	}
	// **وصفرٌ صفرٌ ليس موضعاً** — بل غيابُ موضع.
	if lat == 0 && lng == 0 {
		return false
	}
	return true
}

// RequireServiceable **بوّابةُ القبول — نقطةٌ صالحةٌ داخلَ تغطيةٍ فعّالة.**
//
// **وتُنادى داخلَ معاملةِ الإنشاء** — **فما بين الفحص والكتابة لا
// تتبدّل التغطيةُ في عينِ هذه المعاملة.** (`XG-46`: والمُنفّذُ يُمرَّر
// ولا يُؤخَذ من المَسبَح، **فوصلةٌ ثانيةٌ تُجمّد طلباً خارجَ التغطية
// حين يمتلئ المَسبَح.**)
//
// **والترتيبُ مقصود**: **صحّةُ النقطة قبل سؤال القاعدة** — **فنقطةٌ
// مشوَّهةٌ تُردّ برسالةٍ تقول ما بها، لا بـ«خارج نطاق التوصيل».**
func (s *Service) RequireServiceable(ctx context.Context, q dbtx.Querier,
	lat, lng float64) (ZoneCharge, error) {
	if !ValidPoint(lat, lng) {
		return ZoneCharge{}, ErrBadPoint
	}
	return s.ZoneAt(ctx, q, lat, lng)
}

// requirePlaceLaunched **سلطةُ الجغرافيا الإداريّة عند الإنشاء** (Batch 3a).
//
// **يُنادى بعد `ValidPoint` وقبل `ZoneAt`** — بترتيب سُلّم الإتاحة نفسِه
// (`AvailabilityAt` §٤ قبل §٥-٦) — **فيُطابق الشرحُ الإنشاءَ حرفاً**: يقرأ
// `classifyPlace` عينَها ويردّ خطأً برمز سببها. **والحكمُ للأبِ لا للابن**:
// محافظةٌ أو مدينةٌ مُطفأةٌ تردّ الطلبَ ولو بقيت منطقةٌ ابنةٌ نشطة. **ولا يُطفَأ
// شيءٌ في القاعدة** — القرارُ لحظيٌّ من حالة الأب.
func (s *Service) requirePlaceLaunched(ctx context.Context, q dbtx.Querier, lat, lng float64) error {
	place, err := s.classifyPlace(ctx, q, lat, lng)
	if err != nil {
		return err
	}
	switch place.Reason {
	case ReasonProvinceNotSupported:
		return ErrProvinceNotSupported
	case ReasonCityNotSupported:
		return ErrCityNotSupported
	case ReasonAreaNotSupported:
		return ErrAreaNotSupported
	}
	return nil
}

// CoverableAt **أهذه النقطةُ مُغطّاةٌ فعليّاً الآن، بصرف النظر عن الساعة؟**
// (Batch 3d — إشعارُ تغطية المنطقة).
//
// **بالسلطةِ الكاملةِ للهرم**: المحافظةُ نشطةٌ (`classifyPlace`) والمدينةُ نشطةٌ
// ومنطقةٌ نشطةٌ تحوي النقطةَ هندسيّاً (`ZoneAt`). **ووقتُ المنطقة يُتجاهَل عمداً**:
// نافذةُ ساعةٍ مغلقةٍ ليست «لا تغطية». **فتُشتَقّ من دالّتَي القراءة/الإنشاء
// نفسِهما** فلا تفترق الإتاحةُ عن الإنشاء عن الإشعار.
func (s *Service) CoverableAt(ctx context.Context, q dbtx.Querier, lat, lng float64) (bool, error) {
	if !ValidPoint(lat, lng) {
		return false, nil
	}
	place, err := s.classifyPlace(ctx, q, lat, lng)
	if err != nil {
		return false, err
	}
	if place.Reason != "" {
		return false, nil
	}
	_, zerr := s.ZoneAt(ctx, q, lat, lng)
	switch {
	case zerr == ErrOutOfZone, zerr == ErrCoverageUnavailable:
		return false, nil
	case zerr != nil:
		return false, zerr
	}
	return true, nil
}

// CityEffectivelyLaunched **أُطلقت المدينةُ فعليّاً تحت الهرم؟** (Batch 3d —
// إشعارُ إطلاق المدينة). **المدينةُ نشطةٌ ومحافظتُها نشطة** — فمدينةٌ نشطةٌ تحت
// محافظةٍ مُطفأةٍ ليست مُطلَقةً بعد. (بلا محافظةٍ مرتبطةٍ ⇒ حكمُ المدينة وحدَه.)
func (s *Service) CityEffectivelyLaunched(ctx context.Context, q dbtx.Querier, cityID string) (bool, error) {
	var cityOn bool
	var govOn *bool
	err := q.QueryRow(ctx, `
		SELECT c.active, g.active
		FROM cities c
		LEFT JOIN districts d    ON d.id = c.district_id
		LEFT JOIN governorates g ON g.id = d.governorate_id
		WHERE c.id = $1::uuid`, cityID).Scan(&cityOn, &govOn)
	if err != nil {
		return false, err
	}
	return cityOn && (govOn == nil || *govOn), nil
}
