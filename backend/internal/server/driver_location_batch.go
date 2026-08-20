package server

import (
	"context"
	"net/http"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ══════════════════════════════════════════════════════════════════════
//  **الموقعُ بالجملة — لِما جُمع بلا شبكة**
// ══════════════════════════════════════════════════════════════════════
//
// (خطّةُ تطبيق أندرويد، البند الثالث.)
//
// # ولماذا نقطةٌ واحدةٌ لا تكفي الهاتف
//
// **النقطةُ المفردة تصميمُ متصفّحٍ مفتوح**: التبويبُ حيٌّ والنبضةُ تصل في
// ثانيتها، **فإن سقطت واحدةٌ جاءت التالية بعد دقيقة.**
//
// **والهاتفُ في جيبِ سائقٍ دخل قبواً** — تنقطع الشبكةُ عشرين دقيقة،
// **ويجمع الجهازُ في تلك المدّة عشراتِ النقاط في قاعدته المحلّيّة.**
//
// **فإمّا تُفقد كلُّها، وإمّا تُرسل دفعةً بأزمنتها.** والثانيةُ هي الصواب،
// **لأنّ `driver_track` تُقرأ لتحديد الاتّجاه لا للعرض** — ومن فقد أثرَه
// عشرين دقيقةً لا يُعرف أقادمٌ هو أم خارج، **فيُخطئ الإسنادُ ولا يُخطئ
// أحدٌ في الشيفرة.**
//
// # وليست نداءً مكرّراً للمفرد
//
// **عشرون نداءً متتالياً على شبكةٍ عائدةٍ للتوّ يسقط نصفُها**، وكلُّ واحدٍ
// منها يفتح معاملةً ويُقلّم الجدولَ من جديد. **ونداءٌ واحدٌ يصل أو لا يصل**
// — وإعادتُه لا تُضاعف شيئاً (انظر القيدَ الفريد في هجرة `0101`).

const (
	// maxBatchPoints **سقفُ الدفعة.**
	//
	// **ونبضةٌ كلَّ عشرين ثانيةً تعني ثلاثاً في الدقيقة** — فمئتان تغطّي
	// انقطاعاً يزيد على ساعة. **وما فوقها إمّا عطبٌ في التطبيق وإمّا
	// عبثٌ**، وكلاهما لا يُخدَم.
	maxBatchPoints = 200

	// futureSkew **ما يُحتمل من تقدّمِ ساعةِ الجهاز.**
	//
	// **وساعاتُ الهواتف تسبق وتتأخّر دقائق** — ورفضُ نقطةٍ سبقت بثانيتين
	// يفقد أثراً صحيحاً. **أمّا نقطةٌ في الغد فليست تفاوتَ ساعة**، وقبولُها
	// يجعلها آخرَ ما يُقرأ إلى الأبد.
	futureSkew = 5 * time.Minute

	// batchMaxAge **وأقدمُ ما يُقبَل.**
	//
	// **والأثرُ يُقلَّم بعد ساعتين أصلاً** — فنقطةٌ عمرُها يومٌ تُكتب لتُحذف
	// في النداء نفسِه. **وقبولُها إنفاقٌ بلا أثر.**
	batchMaxAge = 2 * time.Hour
)

// trackPoint نقطةٌ واحدةٌ كما يرسلها الجهاز.
type trackPoint struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
	// At **متى سجّلها الجهازُ** — لا متى وصلت. **وهي جوهرُ الدفعة**:
	// بلاها تُسجَّل عشرون نقطةً في ثانيةٍ واحدة، **فيبدو من وقف عشرين
	// دقيقةً كأنّه قطع المدينةَ في لحظة.**
	At        time.Time `json:"at"`
	SpeedMps  *float64  `json:"speed_mps"`
	AccuracyM *float64  `json:"accuracy_m"`
	// ══════════════════════════════════════════════════════════════════
	// **والاتّجاهُ — أُكمل في الدفعة ٢٠٢٦-٠٨-٢٠**
	// ══════════════════════════════════════════════════════════════════
	//
	// (تصحيحُ المالك: «الموقع المفرد يحفظ `bearing_deg`، لكن نقاط
	//  `location/batch` لا تحمله، **وبالتالي أيّ نقاط تُجمع أثناء
	//  انقطاع الشبكة تفقد الاتجاه نهائيًا**».)
	//
	// **وهو النقصُ بعينه**: الانقطاعُ في الشارع كثير — نفقٌ أو حيٌّ بلا
	// تغطية — **فأطولُ المسارات وأغناها بالمنعطفات هي التي كانت تصل
	// بلا اتّجاه.**
	//
	// **ومؤشّرٌ لا قيمة**: النسخةُ المنشورةُ لا ترسله، **وحقلٌ إلزاميٌّ
	// يجعل طابورَ من كان بلا شبكةٍ يُرفض إلى الأبد.**
	BearingDeg *float64 `json:"bearing_deg"`
}

// valid **نقطةٌ يُعتدّ بها.**
//
// **وصفرٌ صفرٌ ليس موضعاً** — هو ما يرسله جهازٌ لم يجد إشارة، **ونقطةٌ في
// المحيط الأطلسيّ تجعل كلَّ مسافةٍ آلافَ الكيلومترات** فيسقط الترتيبُ كلُّه
// بهدوء.
func (p trackPoint) valid(now time.Time) bool {
	if p.Lat == 0 && p.Lng == 0 {
		return false
	}
	if p.Lat < -90 || p.Lat > 90 || p.Lng < -180 || p.Lng > 180 {
		return false
	}
	if p.At.IsZero() || p.At.After(now.Add(futureSkew)) || p.At.Before(now.Add(-batchMaxAge)) {
		return false
	}
	return true
}

// handleDriverLocationBatch يستقبل نقاطاً جُمعت بلا شبكة.
//
// **ويعيد كم قُبل وكم رُفض** — **وتطبيقٌ يمسح طابورَه على ردٍّ ناجحٍ بلا
// عددٍ يمسح ما لم يُقبَل**، فيضيع الأثرُ وهو يظنّ أنّه سلّمه.
func (s *Server) handleDriverLocationBatch(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Points []trackPoint `json:"points"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if len(req.Points) == 0 || len(req.Points) > maxBatchPoints {
		s.respondErr(w, errValidation)
		return
	}

	now := time.Now()
	good := make([]trackPoint, 0, len(req.Points))
	for _, p := range req.Points {
		if p.valid(now) {
			good = append(good, p)
		}
	}
	if len(good) == 0 {
		// **ونجاحٌ بلا قبول** — لا خطأ: **التطبيقُ الذي يتلقّى خطأً يعيد
		// المحاولةَ إلى الأبد بنقاطٍ لن تُقبل أبداً.** والعددُ يقول له ذلك.
		httpx.JSON(w, http.StatusOK, map[string]any{"accepted": 0, "rejected": len(req.Points)})
		return
	}

	// **وتُرتَّب بالزمن** — **فآخرُ نقطةٍ في الدفعة ليست بالضرورة أحدثَها**:
	// التطبيقُ يُفرغ طابورَه بترتيبٍ لا يضمنه أحد، **ومن كتب `last_location`
	// بآخرِ ما وصله قد يكتب موضعاً قديماً فوق حديث.**
	sort.Slice(good, func(i, j int) bool { return good[i].At.Before(good[j].At) })

	if err := s.saveTrackBatch(r.Context(), userIDFrom(r), good); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"accepted": len(good),
		"rejected": len(req.Points) - len(good),
	})
}

// saveTrackBatch يكتب الدفعةَ في معاملةٍ واحدة.
//
// **ومعاملةٌ واحدةٌ لا عشرون نداءً**: كلُّ نداءٍ منفردٍ يقفل الصفَّ ويكتب
// سجلَّ الكتابة، **وعشرون منها على شبكةٍ عائدةٍ للتوّ تُبطئ كلَّ سائقٍ آخر.**
func (s *Server) saveTrackBatch(ctx context.Context, driverID string, pts []trackPoint) error {
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	batch := &pgx.Batch{}
	for _, p := range pts {
		// **و`ON CONFLICT DO NOTHING` هي مفاتيحُ التكرار للموقع**: دفعةٌ
		// وصلت وانقطع ردُّها **يعيدها التطبيقُ لأنّه لا يعلم**، ولولا
		// القيدُ لَتضاعف الأثرُ بلا أن يظهر — **النقاطُ صحيحةٌ ومكرّرة،
		// فيبدو السائقُ أكثفَ حركةً ممّا كان.**
		// **والاتّجاهُ يُنظَّف قبل أن يُكتب** — كما في النقطة المفردة:
		// **قيدُ القاعدة يرفض ما خرج عن الدائرة، ورفضُه يُسقط الدفعةَ
		// كلَّها** — فتضيع عشرون نقطةً لأنّ واحدةً منها شاذّة.
		batch.Queue(`
			INSERT INTO driver_track (driver_id, at, recorded_at, speed_mps, accuracy_m, bearing_deg)
			VALUES ($1, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography, $4, $5, $6, $7)
			ON CONFLICT (driver_id, recorded_at) DO NOTHING`,
			driverID, p.Lng, p.Lat, p.At, p.SpeedMps, p.AccuracyM, cleanBearing(p.BearingDeg))
	}
	// **وأحدثُ نقطةٍ وحدَها تكتب الموضعَ الحاليّ** — وبعد الفرز هي الأخيرة.
	//
	// **والشرطُ على الزمن يمنع ارتداداً**: دفعةٌ قديمةٌ تصل بعد نبضةٍ حديثة
	// **لا يجوز أن تعيد السائقَ إلى حيث كان قبل ربع ساعة.**
	last := pts[len(pts)-1]
	batch.Queue(`
		UPDATE users
		SET last_location = ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography,
		    last_location_at = $4
		WHERE id = $1 AND (last_location_at IS NULL OR last_location_at < $4)`,
		driverID, last.Lng, last.Lat, last.At)

	// **والتقليمُ مرّةً في الدفعة لا مرّةً في النقطة** — انظر التعليقَ في
	// `driver_location.go`: **التقليمُ مع الكتابة يبقى ما دامت الكتابةُ
	// باقية**، ومهمّةٌ ليليّةٌ تُنسى فينتفخ الجدولُ بصمت.
	batch.Queue(`
		DELETE FROM driver_track
		WHERE driver_id = $1 AND recorded_at < now() - interval '2 hours'`, driverID)

	res := tx.SendBatch(ctx, batch)
	for range batch.Len() {
		if _, err := res.Exec(); err != nil {
			_ = res.Close()
			return err
		}
	}
	if err := res.Close(); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
