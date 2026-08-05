package orders

// تعقّبُ السائق — **أواقفٌ عند المتجر، وإلى أين يتّجه.**
//
// # سؤالان لا يجيبهما موضعٌ واحد
//
//	أواقفٌ أم عابر؟  ←  من لم يبرح دائرةً صغيرةً دقائقَ فهو واقف
//	إلى أين يتّجه؟   ←  أيقترب من نقطةٍ أم يبتعد عنها
//
// **ونقطةٌ واحدةٌ لا تقول شيئاً من ذلك**: من هو على بُعد أربعمئة مترٍ من المتجر
// قد يكون قادماً إليه أو خارجاً منه، **والفرقُ بينهما هو الفرقُ بين إسنادٍ
// صائبٍ وطلبٍ يضيع.**
//
// # ولماذا لا نستعمل الاتّجاه الذي يعطيه الجهاز
//
// **بوصلةُ الهاتف تقول أين يشير رأسُه لا إلى أين يمضي**: هاتفٌ في جيبٍ يدور
// مع صاحبه، **ودرّاجةٌ تنعطف في زقاقٍ تُقرأ ذاهبةً إلى الجهة الخطأ.**
//
// **والاقترابُ يُقاس بالمسافة لا بالزاوية**: نقطتان في وقتين، ومسافتاهما إلى
// الهدف — **وإن قصرت الثانيةُ فهو قادم.** ولا يكذب هذا مهما دار الهاتف.

import (
	"context"
	"time"
)

// TrackWindow المدّةُ التي تُقرأ منها الحركة.
//
// **وخمسُ دقائقَ تكفي**: أقصرُ منها تجعل إشارةَ وقوفٍ من دقيقتين تُقرأ توقّفاً،
// **وأطولُ منها تجعل من غادر قبل ثلاثِ دقائقَ يبدو واقفاً بعد.**
const TrackWindow = 5 * time.Minute

// StillRadiusM نصفُ قطر «الوقوف» — **بالأمتار.**
//
// **ودقّةُ الـGPS في المدن تتراوح عشراتِ الأمتار**، فمن وقف تماماً تتراقص
// نقطتُه في دائرةٍ من ثلاثين متراً. **وحدٌّ أضيقُ يجعل الواقفَ متحرّكاً دائماً**،
// وأوسعُ يجعل من مشى في الشارع واقفاً.
const StillRadiusM = 60

// Movement ما يُقرأ من أثر السائق.
type Movement struct {
	// Known **أثمّة أثرٌ يكفي للحكم؟** — ونقطةٌ واحدةٌ لا تكفي.
	//
	// **والجهلُ يُقال ولا يُخمَّن**: من لا أثرَ له لا يُحكم عليه بأنّه واقف
	// ولا بأنّه ماضٍ، **وحكمٌ على غير بيّنةٍ أسوأُ من لا حكم.**
	Known bool
	// Still لم يبرح دائرةً صغيرةً في النافذة.
	Still bool
	// SpreadM أوسعُ مسافةٍ بين نقطتين في النافذة — **مقياسُ الحركة.**
	SpreadM float64
	// Points كم نقطةً في النافذة.
	Points int
}

// MovementOf يقرأ حركةَ سائقٍ في النافذة الأخيرة.
func (s *Service) MovementOf(ctx context.Context, driverID string) Movement {
	var m Movement
	// **والانتشارُ يُحسب في القاعدة لا في Go** — جلبُ مئة نقطةٍ لحساب مسافةٍ
	// بينها نقلُ بياناتٍ بلا سبب.
	err := s.db.QueryRow(ctx, `
		WITH pts AS (
			SELECT at FROM driver_track
			WHERE driver_id = $1 AND created_at > now() - make_interval(secs => $2)
			ORDER BY created_at DESC
			LIMIT 200
		)
		SELECT count(*),
		       COALESCE((SELECT max(ST_Distance(a.at, b.at))
		                 FROM pts a CROSS JOIN pts b), 0)
		FROM pts`, driverID, TrackWindow.Seconds()).Scan(&m.Points, &m.SpreadM)
	if err != nil || m.Points < 2 {
		return m
	}
	m.Known = true
	m.Still = m.SpreadM <= StillRadiusM
	return m
}

// Approach هل يقترب من نقطةٍ أم يبتعد.
type Approach struct {
	Known bool
	// Closing يقترب — **أقصرُ آخرُ مسافةٍ من أقدمِها في النافذة.**
	Closing bool
	// NowM بُعدُه الآن، و ThenM بُعدُه في أوّل النافذة.
	NowM  float64
	ThenM float64
}

// ApproachTo يقرأ أيقترب السائقُ من نقطةٍ أم يبتعد عنها.
//
// **والقياسُ بطرفَي النافذة لا بكلّ نقطة**: من دار حول مبنًى ثمّ اتّجه إلينا
// **يُقرأ مقترباً وهو كذلك**، ومن تعرّج في الطريق لا يُقرأ متردّداً.
func (s *Service) ApproachTo(ctx context.Context, driverID string, lat, lng float64) Approach {
	var a Approach
	err := s.db.QueryRow(ctx, `
		WITH pts AS (
			SELECT at, created_at FROM driver_track
			WHERE driver_id = $1 AND created_at > now() - make_interval(secs => $2)
			ORDER BY created_at DESC
			LIMIT 200
		), target AS (
			SELECT ST_SetSRID(ST_MakePoint($4, $3), 4326)::geography AS g
		)
		SELECT
			(SELECT ST_Distance(p.at, t.g) FROM pts p, target t
			 ORDER BY p.created_at DESC LIMIT 1),
			(SELECT ST_Distance(p.at, t.g) FROM pts p, target t
			 ORDER BY p.created_at ASC LIMIT 1)
		FROM pts LIMIT 1`, driverID, TrackWindow.Seconds(), lat, lng).
		Scan(&a.NowM, &a.ThenM)
	if err != nil {
		return a
	}
	a.Known = true
	// **والاقترابُ بفرقٍ يُعتدّ به** — لا بمترٍ واحد: **ترقّصُ الـGPS يجعل
	// الواقفَ يُقرأ مقترباً ثمّ مبتعداً كلَّ ثانية.**
	a.Closing = a.ThenM-a.NowM > 50
	return a
}
