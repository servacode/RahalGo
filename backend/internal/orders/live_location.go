package orders

// ══════════════════════════════════════════════════════════════════════
// **موقعُ الزبون المباشر** (قرارُ المالك ٢٠٢٦-١٠-١٦)
// ══════════════════════════════════════════════════════════════════════
//
// **دبابيسُ الخريطة في الرقّة كثيراً ما تخطئ** — فيصل الكابتنُ الدبّوسَ
// والزبونُ في شارعٍ آخر. **فحين يصير الطلبُ «بالطريق» يُسأل الزبونُ أن
// يشارك موقعَه**، فيرسله تطبيقُه كلَّ عشر ثوانٍ ويراه السائقُ على خريطته.
//
// # وما يُحفظ
//
// **آخرُ نقطةٍ ووقتُها لا غير** (`0450_order_live_location.sql`) — **ولا
// أثرَ يُحفظ**: مواضعُ الزبون الماضية ليست شأنَ أحد.
//
// # ومن يراه
//
// **السائقُ المُسنَدُ والمكتبُ وحدَهما** (`privacy_contract.go`) — **ولا يقرؤه
// استعلامُ الطلب العامّ**: يُلحَق في ردّيهما بعينهما (`LiveFor`).

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// LiveFreshFor **بعدها تُعامَل النقطةُ كأنّها غيرُ موجودة** (قرارُ المالك: دقيقتان).
//
// **ونقطةٌ بلا وقتٍ كذبةٌ تشيخ** — زبونٌ أطفأ هاتفَه قبل ربع ساعةٍ يبقى موضعُه
// مكتوباً، **فيُقاد السائقُ إلى حيث كان لا إلى حيث هو.**
const LiveFreshFor = 2 * time.Minute

// SettingLiveSwitchM **كم يبعد الزبونُ عن الدبّوس قبل أن يتحوّل الطريقُ إليه** — بالمتر.
const SettingLiveSwitchM = "orders.live_location_switch_m"

// SettingLiveMaxAccuracyM **أسوأُ دقّةٍ تُعدّ قراءةً دقيقة** — بالمتر (قرارُ المالك ٢٠٢٦-١٠-١٦).
//
// **ما فوقها يُرى ولا يُقاد إليه**: نقطةٌ بدقّةِ ستّين متراً تقول «هو في الحارة»
// لا «هو عند الباب»، **وطريقٌ يتحوّل إليها يأخذ السائقَ إلى بيت الجيران.**
const SettingLiveMaxAccuracyM = "orders.live_location_max_accuracy_m"

// SettingNearNoticeM **كم يقترب الكابتنُ قبل أن يُنبَّه الزبون** — بالمتر (قرارُ المالك ٢٠٢٦-١٠-١٦).
const SettingNearNoticeM = "orders.near_notice_m"

// LiveRule **حدّا الحكم معاً** — بُعدُ التحويل وأسوأُ دقّةٍ مقبولة، من اللوحة.
//
// **يُقرآن مرّةً ويُمرَّران معاً** — فلا يقرأ بابٌ حدّاً من اللوحة وآخرُ افتراضَه.
type LiveRule struct {
	SwitchM      int64
	MaxAccuracyM int64
}

// LiveStatuses **متى يُقبل الموقعُ المباشر** — من الإسناد إلى الباب.
//
// **وما قبل الإسناد لا سائقَ يراه**، **وما بعد التسليم لا طريقَ يُقاد عليه.**
var LiveStatuses = []string{StAssigned, StAtPickup, StPickedUp, StOnTheWay, StAtDropoff}

// liveKinds **الطلبُ العاديّ والخاصّ بنوعيه** (مشترياتٌ وأمانة) — **و«لدي توصيلة»
// خارجه**: مستلِمُها ليس على التطبيق.
var liveKinds = []string{"standard", "custom"}

// ErrLiveInactive **الطلبُ ليس في طريقه** — انتهى أو لم يُسنَد بعد.
var ErrLiveInactive = httpx.NewError(http.StatusConflict,
	"live_location_inactive", "errors.live_location_inactive")

// LivePoint **آخرُ ما أرسله الزبون** — كما يصل السائقَ والمكتب.
type LivePoint struct {
	Lat       float64  `json:"lat"`
	Lng       float64  `json:"lng"`
	AccuracyM *float64 `json:"accuracy_m"`
	// At **متى وصلت** — والتطبيقُ يقيس عمرَها بنفسه بين قراءتين.
	At time.Time `json:"at"`
	// PinGapM **بُعدُها عن دبّوس العنوان بالمتر** — ومنه يُعرف أيُّهما الوجهة.
	PinGapM float64 `json:"pin_gap_m"`
	// SwitchM **الحدُّ الذي يتحوّل بعده الطريقُ إلى الزبون** — من الإعدادات.
	SwitchM int64 `json:"switch_m"`
	// Precise **أدقيقةٌ هي؟** — دقّتُها معروفةٌ ودون `orders.live_location_max_accuracy_m`
	// (قرارُ المالك ٢٠٢٦-١٠-١٦). **والتقريبيّةُ تُرى على الخريطة ولا يتحوّل إليها طريق.**
	Precise bool `json:"precise"`
}

// precise **الدقّةُ دون الحدّ** — **ودقّةٌ مجهولةٌ ليست دقيقة**: جهازٌ لم يقل كم
// يخطئ لا يُصدَّق أنّه عند الباب.
func precise(accuracyM *float64, maxAccuracyM int64) bool {
	return accuracyM != nil && *accuracyM <= float64(maxAccuracyM)
}

// Switch **أيُقاد السائقُ إلى الزبون لا إلى الدبّوس؟**
//
// **شرطان معاً** (قرارُ المالك ٢٠٢٦-١٠-١٦): **بعيدٌ عن الدبّوس فوق الحدّ، ودقيقٌ.**
// **فنقطةٌ داخلَ بيتٍ بدقّةِ أربعين متراً لا تسحب الطريقَ** عن دبّوسٍ قد يكون صحيحاً.
//
// **حكمٌ واحدٌ يقرؤه الطريقُ في الخادم** (`driver_route.go`) **وتنبيهُ القرب**
// (`NearNoticeDue`) **وتقرؤه شاشةُ السائق بالحقول نفسِها** (`pin_gap_m` · `switch_m`
// · `precise`) — فلا يرسم التطبيقُ طريقاً والخادمُ يحسب غيرَه.
func (p *LivePoint) Switch() bool {
	return p != nil && p.Precise && p.PinGapM > float64(p.SwitchM)
}

// SaveLive **يكتب آخرَ نقطةٍ للزبون** — ويردّ سائقَ الطلب ليُنبَّه.
//
// **وفي جملةٍ واحدة**: لا يُكتب إلّا لصاحب الطلب وهو في طريقه، **فلا نافذةَ
// بين الفحص والكتابة يُغلق فيها الطلب.**
func (s *Service) SaveLive(ctx context.Context, orderID, customerID string,
	lat, lng float64, accuracyM *float64) (string, error) {
	var driverID *string
	err := s.db.QueryRow(ctx, `
		WITH o AS (
			SELECT id, customer_id, driver_id FROM orders
			WHERE id = $1::uuid AND customer_id = $2::uuid AND closed_at IS NULL
			  AND status = ANY($6::text[]) AND kind = ANY($7::text[])
		), up AS (
			INSERT INTO order_live_locations (order_id, customer_id, point, accuracy_m, at)
			SELECT o.id, o.customer_id, ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography, $5, now()
			FROM o
			ON CONFLICT (order_id) DO UPDATE
			SET point = EXCLUDED.point, accuracy_m = EXCLUDED.accuracy_m, at = EXCLUDED.at
			RETURNING order_id
		)
		SELECT o.driver_id::text FROM o JOIN up ON up.order_id = o.id`,
		orderID, customerID, lng, lat, accuracyM, LiveStatuses, liveKinds).Scan(&driverID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", s.liveRefusal(ctx, orderID, customerID)
	}
	if err != nil {
		return "", err
	}
	if driverID == nil {
		return "", nil
	}
	return *driverID, nil
}

// liveRefusal **لماذا لم يُكتب** — **طلبُ غيره لا يُعرَف أنّه موجود** (٤٠٤)،
// وطلبُه خارجَ طريقه «ليس في طريقه» (٤٠٩).
func (s *Service) liveRefusal(ctx context.Context, orderID, customerID string) error {
	var mine bool
	err := s.db.QueryRow(ctx,
		`SELECT customer_id = $2::uuid FROM orders WHERE id = $1::uuid`,
		orderID, customerID).Scan(&mine)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !mine) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return ErrLiveInactive
}

// StopLive **«وقّف المشاركة»** — يمحو النقطةَ ويردّ سائقَ الطلب.
//
// **ولا يُردّ طلبٌ انتهى**: الإيقافُ بعد النهاية لا شيءَ يمحوه، **ونجاحٌ صامتٌ
// خيرٌ من خطأٍ يقرؤه من ضغط «وقّف» متأخّراً.**
func (s *Service) StopLive(ctx context.Context, orderID, customerID string) (string, error) {
	var driverID *string
	err := s.db.QueryRow(ctx,
		`SELECT driver_id::text FROM orders WHERE id = $1::uuid AND customer_id = $2::uuid`,
		orderID, customerID).Scan(&driverID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", httpx.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if _, err := s.db.Exec(ctx,
		`DELETE FROM order_live_locations WHERE order_id = $1::uuid`, orderID); err != nil {
		return "", err
	}
	if driverID == nil {
		return "", nil
	}
	return *driverID, nil
}

// LiveFor **النقاطُ الحديثةُ لطلباتٍ بأعيانها** — لطلبٍ في طريقه وحدَه.
//
// **وما شاخ لا يُردّ** — **وما انتهى طلبُه لا يُردّ ولو بقي صفُّه** (المحفّزُ
// يمحوه، **وهذا حارسٌ ثانٍ لا يتّكل على الأوّل**).
//
// **و`driverID` غيرُ الفارغ يشترط أنّ الطلبَ طلبُه** — **فسائقٌ آخرُ لا يراه
// ولو مرّ الطلبُ بقائمته** (عرضٌ أو طابور). **والفارغُ للمكتب وحدَه.**
func (s *Service) LiveFor(ctx context.Context, orderIDs []string, driverID string,
	rule LiveRule) (map[string]*LivePoint, error) {
	out := map[string]*LivePoint{}
	if len(orderIDs) == 0 {
		return out, nil
	}
	rows, err := s.db.Query(ctx, `
		SELECT l.order_id::text, ST_Y(l.point::geometry), ST_X(l.point::geometry),
		       l.accuracy_m::float8, l.at,
		       CASE WHEN o.dropoff IS NULL THEN -1 ELSE ST_Distance(l.point, o.dropoff) END
		FROM order_live_locations l
		JOIN orders o ON o.id = l.order_id
		WHERE l.order_id = ANY($1::uuid[])
		  AND o.closed_at IS NULL AND o.status = ANY($2::text[])
		  AND l.at > now() - make_interval(secs => $3::int)
		  AND ($4::text = '' OR o.driver_id::text = $4::text)`,
		orderIDs, LiveStatuses, int(LiveFreshFor/time.Second), driverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		p := &LivePoint{SwitchM: rule.SwitchM}
		if err := rows.Scan(&id, &p.Lat, &p.Lng, &p.AccuracyM, &p.At, &p.PinGapM); err != nil {
			return nil, err
		}
		p.Precise = precise(p.AccuracyM, rule.MaxAccuracyM)
		out[id] = p
	}
	return out, rows.Err()
}
