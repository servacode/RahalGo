package orders

// الإسنادُ بنفس المسار — **طلبٌ يقع في طريقِ سائقٍ ماضٍ فيه.**
//
// # الفكرة
//
// سائقٌ ذاهبٌ إلى مطعمٍ في المنصور، وطلبٌ جديدٌ من مطعمٍ يبعد عنه مئتَي متر
// **وزبونُه في الاتّجاه نفسِه**. **إسنادُه إليه رحلةٌ واحدةٌ بدل رحلتين** —
// ولمن ينتظر دقائقُ أقلّ، وللمنصة مشوارٌ أقلّ.
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٥: «أيُّ طلبٍ قريبٍ من المتجر الذي يتوجّه به سائق،
// بشرط أن يكون الطلبان بنفس الخطّ، يُسنَد مباشرةً إلى نفس السائق... ويأتيه
// الطلبُ بشكلٍ خاصٍّ وليس للجميع».)
//
// # وأربعةُ شروطٍ لا واحد
//
//  1. **المتجران متجاوران** — وإلّا صارت وقفةً ثانيةً في آخر المدينة.
//  2. **والزبونان في الجهة نفسِها** — مطعمان متلاصقان وزبونان متعاكسان
//     **رحلتان لا رحلة**، والثاني يبرد طعامُه في الطريق إلى الأوّل.
//  3. **ولم يتجاوز المتجرَ الثاني بعد** — من مرّ عليه ومضى **يرجع القهقرى**،
//     وهو أطولُ من رحلةٍ مستقلّة.
//  4. **ويحتمله سقفُه** — نقداً وعددَ طلبات: **الحارسُ نفسُه الذي في الدور**،
//     ولا يُلتفّ عليه باسم الذكاء.
//
// # ولماذا لا يُعرض على غيره
//
// **الطلبُ يُسنَد إليه خاصّةً** لأنّ قيمتَه في اجتماعه مع ما بيده: **من لا
// يحمل الأوّلَ لا يربح شيئاً من قرب الثاني**، فعرضُه على الجميع يُبدّده.

import (
	"context"
	"strconv"

	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// SameRouteRadiusM كم بين المتجرين ليُعدّا متجاورين.
//
// **وثمانمئةُ مترٍ وقفةٌ بدقيقتين على درّاجة**، وأبعدُ منها رحلةٌ ثانيةٌ
// تُسمّى نفسَها اقتصاداً وهي ليست كذلك.
const SameRouteRadiusM = 800

// SameRouteSpreadM كم بين الزبونين ليُعدّا في الجهة نفسِها.
//
// **وأوسعُ من المتجرين عمداً**: نقطتا تسليمٍ في حيٍّ واحدٍ قد تبعدان كيلومتراً
// **وهما على الخطّ نفسِه**، بينما مطعمان متباعدان وقفتان مستقلّتان.
const SameRouteSpreadM = 2000

// SameRouteCandidate سائقٌ يصلح لطلبٍ بنفس مساره — **ومعه سببُ صلاحه.**
//
// **والسببُ يُعاد لا يُخفى**: العملياتُ تسأل «لماذا هذا؟» — **وجوابٌ مثل
// «الخوارزميةُ اختارته» ليس جواباً.**
type SameRouteCandidate struct {
	DriverID string
	// AnchorOrderID الطلبُ الذي بيده والذي جعله مرشّحاً.
	AnchorOrderID string
	// BetweenPickupsM بين المتجرين، و BetweenDropoffsM بين الزبونين.
	BetweenPickupsM  float64
	BetweenDropoffsM float64
}

// SameRouteDriver يجد سائقاً في مسار هذا الطلب — **أو لا يجد.**
//
// **ولا يُخترع مرشّحٌ عند الشكّ**: من لا أثرَ له لا يُحكم عليه، **والطلبُ ينزل
// إلى الطابور كما كان.** والطابورُ يعمل، **والذكاءُ زيادةٌ لا بديل.**
func (s *Service) SameRouteDriver(ctx context.Context, orderID string) *SameRouteCandidate {
	limit := s.settingInt(ctx, "drivers.cash_limit")
	// ══════════════════════════════════════════════════════════════════
	// **وسقفُ هذا الطريق أعلى بواحد — بقرار المالك**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-١٣.)
	//
	// **السقفُ العامُّ واحد**: لا يحمل السائقُ طلبين متباعدَين، **فزبونٌ
	// ينتظر بينما سائقُه في حيٍّ آخر** هو ما أراد منعَه.
	//
	// **وهذا الطريقُ استثناؤه**: الشروطُ الأربعةُ فوق تضمن أنّ الثاني
	// **في طريقه أصلا** — متجرٌ على بعد ثمانمئة متر، وزبونٌ في الجهة
	// نفسِها، **ولم يتجاوز المتجرَ بعد.**
	//
	// **فالزيادةُ لا تعني تكديسا** — تعني وقفةً ثانيةً على طريقٍ يسلكه.
	maxActive := s.settingInt(ctx, "drivers.max_active_orders") +
		s.settingInt(ctx, "drivers.same_route_extra")

	var c SameRouteCandidate
	// **والشروطُ في الاستعلام لا بعده**: جلبُ كلّ سائقٍ ثمّ غربلتُه في Go
	// يعني قراءةَ المنصة كلِّها لاختيار واحد.
	err := s.db.QueryRow(ctx, `
		WITH nw AS (
			SELECT o.id, COALESCE(o.pickup_override, m.location) AS pick, o.dropoff,
			       o.cash_due
			FROM orders o JOIN merchants m ON m.id = o.merchant_id
			WHERE o.id = $1
		)
		SELECT a.driver_id::text, a.id::text,
		       ST_Distance(a.pick, nw.pick), ST_Distance(a.dropoff, nw.dropoff)
		FROM (
			SELECT o.id, o.driver_id, o.status,
			       COALESCE(o.pickup_override, m.location) AS pick, o.dropoff
			FROM orders o JOIN merchants m ON m.id = o.merchant_id
			WHERE o.driver_id IS NOT NULL AND o.closed_at IS NULL
			  -- **وقبل الاستلام وحدَه**: من استلم بضاعتَه ومضى إلى الزبون
			  -- **لا يعود إلى مطعمٍ ثانٍ والطعامُ يبرد في صندوقه.**
			  AND o.status IN ('assigned', 'at_pickup')
		) a
		CROSS JOIN nw
		JOIN users u ON u.id = a.driver_id
		WHERE u.on_shift AND u.status = 'active'
		  -- ١ · المتجران متجاوران
		  AND ST_Distance(a.pick, nw.pick) <= $2
		  -- ٢ · والزبونان في الجهة نفسِها
		  AND ST_Distance(a.dropoff, nw.dropoff) <= $3
		  -- ٣ · ولم يتجاوز المتجرَ الثاني: **موضعُه أقربُ إليه ممّا هو إلى
		  --     زبونه** — ومن جاوزه صار الزبونُ أقربَ إليه من المطعم.
		  --
		  --     **وموضعٌ شاخ لا يُقاس عليه** — من أطفأ التطبيقَ قبل ساعةٍ
		  --     يبقى موضعُه مكتوباً.
		  AND u.last_location IS NOT NULL
		  AND u.last_location_at > now() - interval '5 minutes'
		  AND ST_Distance(u.last_location, nw.pick) <
		      ST_Distance(u.last_location, nw.dropoff)
		  -- ٤ · ويحتمله سقفُه — **الحارسُ نفسُه الذي في الدور.**
		  AND COALESCE((SELECT b.held FROM driver_cash_boxes b
		                WHERE b.driver_id = u.id), 0) + nw.cash_due <= $4
		  AND (SELECT count(*) FROM orders o2
		       WHERE o2.driver_id = u.id AND o2.closed_at IS NULL) < $5
		-- **والأقربُ أوّلاً** — وقفتان متلاصقتان خيرٌ من متباعدتين.
		ORDER BY ST_Distance(a.pick, nw.pick)
		LIMIT 1`,
		orderID, SameRouteRadiusM, SameRouteSpreadM, limit, maxActive).
		Scan(&c.DriverID, &c.AnchorOrderID, &c.BetweenPickupsM, &c.BetweenDropoffsM)
	if err != nil {
		return nil
	}
	return &c
}

// sameRouteNote نصُّ حدثِ الإسناد بنفس المسار — **يُقرأ في سجلّ الطلب.**
const sameRouteNote = "إسنادٌ بنفس المسار — سائقٌ في الطريق"

// TrySameRoute يحاول إسنادَ الطلب إلى سائقٍ في مساره — **ويُرجع أوقع أم لا.**
//
// **ولا يُنادى إلّا حين ينزل الطلبُ إلى الطابور**: قبلها لا مسارَ له، وبعدها
// صار لأحد.
func (s *Service) TrySameRoute(ctx context.Context, orderID string) bool {
	c := s.SameRouteDriver(ctx, orderID)
	if c == nil {
		return false
	}
	// **والسببُ في نصّ الحدث نفسِه لا في حدثٍ ثانٍ.**
	//
	// **وحدثان لانتقالٍ واحدٍ يُقرآن انتقالين**: من يمسح السجلَّ يرى الطلبَ
	// أُسند مرّتين، **ويسأل عن الثانية.**
	if err := s.assignDirectly(ctx, orderID, c.DriverID, sameRouteNote); err != nil {
		s.logger.Warn("نفسُ المسار: تعذّر الإسناد", "order", orderID, "error", err)
		return false
	}
	s.logger.Info("نفسُ المسار: أُسند",
		"order", orderID, "driver", c.DriverID, "anchor", c.AnchorOrderID,
		"pickups_m", int(c.BetweenPickupsM), "dropoffs_m", int(c.BetweenDropoffsM))

	// **وتنبيهٌ يقول لماذا** — لا «طلبٌ جديد» وحدَها.
	//
	// **والسائقُ في الطريق إلى مطعمٍ حين يصله**: تنبيهٌ لا يشرح يجعله يظنّ
	// أنّ الطلبَ الأوّلَ أُلغي أو أنّ شيئاً اختلط، **فيقف ليقرأ ويسأل.**
	//
	// **و«بنفس مسارك» تُغني عن الوقوف**: يعرف أنّهما رحلةٌ واحدة، ويكمل.
	// (قرارُ المالك ٢٠٢٦-٠٨-٠٥: «يأتيه الطلبُ مع تنبيه: لديك طلبٌ جديدٌ
	// بنفس المسار».)
	if s.notify != nil {
		var number int64
		var merchant string
		if err := s.db.QueryRow(ctx, `
			SELECT o.number, m.name FROM orders o
			JOIN merchants m ON m.id = o.merchant_id WHERE o.id = $1`,
			orderID).Scan(&number, &merchant); err == nil {
			s.notify.Notify(ctx, notifications.Input{
				UserID:   c.DriverID,
				Kind:     "order",
				Title:    "طلبٌ جديدٌ بنفس مسارك",
				Body:     merchant + " — على بُعد " + metersText(c.BetweenPickupsM) + " من وجهتك",
				Entity:   "order",
				EntityID: orderID,
				Href:     "/portal",
			})
		}
	}
	return true
}

// metersText مسافةٌ مقروءة — **بالمتر تحت الكيلو وبالكيلو فوقه.**
//
// **و«٨٤٧ متراً» تُقرأ بلمحة، و«٠٫٨٤٧ كم» تُقرأ بتفكير** — والسائقُ على درّاجة.
func metersText(m float64) string {
	if m < 1000 {
		return strconv.Itoa(int(m)) + " م"
	}
	return strconv.FormatFloat(m/1000, 'f', 1, 64) + " كم"
}
