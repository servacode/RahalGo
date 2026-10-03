package orders

// ══════════════════════════════════════════════════════════════════════
// **إرجاعُ البضاعة مشوارٌ يُرى لا إغلاقٌ صامت** (قرارُ المالك ٢٠٢٦-١٠-٠٣)
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان
//
// **«عُد إلى المكتب» يُنهي الطلبَ فشلاً ويخرج من يد السائق** — فيرى «الطلب
// لم يعد معك» **ولا طريقَ إلى المكتب ولا زرَّ يقول إنّ البضاعةَ وصلت.**
// والإدارةُ لا تعرف متى عادت ولا أين هي.
//
// # وما صار
//
// **الطلبُ يُنهى كما كان** — الحالُ والذنبُ والمالُ لا تتبدّل حرفاً. **ومعه
// مشوارُ إرجاع** (`return_to`): يبقى الطلبُ في قائمة السائق بوجهةٍ هي المكتبُ
// أو المتجرُ الذي يقبل الاسترداد، **بطريقٍ مرسومٍ وزرٍّ واحد: «سلّمت البضاعة».**
// ضغطُه يكتب وقتَه وموضعَه (`goods_handed_at`)، **فيخرج المشوارُ من يده وتراه
// الإدارةُ على بطاقة الطلب.**
//
//	المكتب   `platform.location` (إعداداتُ صفحة التواصل) — مصدرٌ واحد،
//	         **وفارغُها مشوارٌ بلا طريق** يقرأ فيه السائقُ العنوانَ (`platform.address`)
//	المتجر   موقعُ المتجر — **لمن يقبل الاسترداد وحدَه، ولا في الطلب الخاصّ**
//
// **والمالُ ليس هنا**: تسويةُ البضاعة قرارُ المكتب بعد أن تصل (`goods.go`).

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// وجهتا مشوار الإرجاع.
const (
	ReturnToOffice = "office"
	ReturnToStore  = "store"
)

// أخطاءُ مشوار الإرجاع.
var (
	// ErrReturnBadTarget **وجهةٌ لا تُعرف** — المكتبُ أو المتجرُ لا ثالث.
	ErrReturnBadTarget = httpx.NewError(http.StatusBadRequest, "return_bad_target", "errors.return_bad_target")
	// ErrReturnStoreRefused **المتجرُ لا يقبل الاسترداد** — أو الطلبُ خاصٌّ بلا متجر.
	ErrReturnStoreRefused = httpx.NewError(http.StatusConflict, "return_store_refused", "errors.return_store_refused")
	// ErrNoReturnTrip **لا مشوارَ إرجاعٍ قائمٌ على هذا الطلب.**
	ErrNoReturnTrip = httpx.NewError(http.StatusConflict, "no_return_trip", "errors.no_return_trip")
	// ErrGoodsAlreadyHanded **سُلّمت البضاعةُ سلفاً** — والوقتُ لا يُكتب مرّتين.
	ErrGoodsAlreadyHanded = httpx.NewError(http.StatusConflict, "goods_already_handed", "errors.goods_already_handed")
)

// normalizeReturnTo **وجهةُ الإرجاع كما طلبها المكتب** — وفارغُها المكتب.
func normalizeReturnTo(raw string) (string, error) {
	switch strings.TrimSpace(raw) {
	case "", ReturnToOffice:
		return ReturnToOffice, nil
	case ReturnToStore:
		return ReturnToStore, nil
	}
	return "", ErrReturnBadTarget
}

// storeTakesReturns **أيقبل كلُّ مصدرٍ في الطلب بضاعتَه؟** — القاعدةُ نفسُها التي
// تُظهر «رُدّت إلى المتجر» (`merchant_accepts_returns` في `queries.go`)، **ولا
// متجرَ للطلب الخاصّ.**
func storeTakesReturns(ctx context.Context, q dbtx.Querier, orderID string) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, `
		SELECT o.kind <> 'custom' AND o.merchant_id IS NOT NULL AND
		       COALESCE((SELECT bool_and(m2.accepts_returns) FROM merchants m2
		                 WHERE m2.id IN (SELECT COALESCE(oi.merchant_id, o.merchant_id)
		                                 FROM order_items oi WHERE oi.order_id = o.id
		                                 UNION SELECT o.merchant_id)), false)
		FROM orders o WHERE o.id = $1`, orderID).Scan(&ok)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, httpx.ErrNotFound
	}
	return ok, err
}

// ParseGeoSetting **يقرأ موقعاً مكتوباً `"lat,lng"`** — صيغةُ `KindGeo`. وفاسدُه «لا موقع».
func ParseGeoSetting(raw string) (lat, lng float64, ok bool) {
	a, b, found := strings.Cut(strings.TrimSpace(raw), ",")
	if !found {
		return 0, 0, false
	}
	la, err1 := strconv.ParseFloat(strings.TrimSpace(a), 64)
	ln, err2 := strconv.ParseFloat(strings.TrimSpace(b), 64)
	if err1 != nil || err2 != nil || la < -90 || la > 90 || ln < -180 || ln > 180 {
		return 0, 0, false
	}
	return la, ln, true
}

// ReturnPoint **وجهةُ مشوار الإرجاع** — ما يُرسم إليه الطريقُ وما يُقرأ على الشاشة.
type ReturnPoint struct {
	// To `office` أو `store`.
	To string
	// Label **اسمُ المتجر** — وفارغٌ للمكتب (التطبيقُ يقول «المكتب» بلغته).
	Label string
	// Address **العنوانُ المكتوب** — يُقرأ حين لا نقطة.
	Address string
	// Lat/Lng **النقطة** — وفارغةٌ إن لم يُضبط موقعُ المكتب أو لم يُدبَّس المتجر.
	Lat, Lng *float64
}

// ReturnPointOf **إلى أين يُرجع السائقُ بضاعةَ هذا الطلب** — وفارغٌ بلا مشوارٍ قائم.
//
// **والمكتبُ من الإعدادات لحظةَ السؤال** لا لقطةً على الطلب: موقعٌ صُحّح اليومَ
// يصل كلَّ مشوارٍ قائم.
func (s *Service) ReturnPointOf(ctx context.Context, orderID string) (*ReturnPoint, error) {
	var (
		to, name, addr string
		mLat, mLng     *float64
		pending        bool
	)
	err := s.db.QueryRow(ctx, `
		SELECT COALESCE(o.return_to, ''), COALESCE(m.name, ''), COALESCE(m.address_text, ''),
		       ST_Y(m.location::geometry), ST_X(m.location::geometry),
		       o.status = 'failed' AND o.return_to IS NOT NULL AND o.goods_handed_at IS NULL
		FROM orders o LEFT JOIN merchants m ON m.id = o.merchant_id
		WHERE o.id = $1`, orderID).Scan(&to, &name, &addr, &mLat, &mLng, &pending)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if !pending {
		return nil, nil
	}
	return s.returnPoint(ctx, to, name, addr, mLat, mLng), nil
}

// returnPoint **يبني الوجهةَ من وجهة المكتب وبيانات المتجر.**
func (s *Service) returnPoint(ctx context.Context, to, storeName, storeAddr string,
	storeLat, storeLng *float64) *ReturnPoint {
	if to == ReturnToStore {
		return &ReturnPoint{To: to, Label: storeName, Address: storeAddr, Lat: storeLat, Lng: storeLng}
	}
	p := &ReturnPoint{To: ReturnToOffice}
	if s.settings != nil {
		p.Address = s.settings.GetString(ctx, "platform.address")
		if la, ln, ok := ParseGeoSetting(s.settings.GetString(ctx, "platform.location")); ok {
			p.Lat, p.Lng = &la, &ln
		}
	}
	return p
}

// ReturnPointFor **كـ`ReturnPointOf` من بياناتٍ مقروءةٍ سلفاً** — لقائمة السائق،
// **فلا نداءَ لكلّ صفّ.**
func (s *Service) ReturnPointFor(ctx context.Context, to, storeName, storeAddr string,
	storeLat, storeLng *float64) *ReturnPoint {
	return s.returnPoint(ctx, to, storeName, storeAddr, storeLat, storeLng)
}

// HandGoods **«سلّمت البضاعة» — السائقُ أنهى مشوارَ الإرجاع.**
//
// **لسائق الطلب وحدَه**، على طلبٍ فشل بمشوارٍ قائم، **ومرّةً واحدة.** والموضعُ
// موضعُ الجهاز إن أرسله، **وإلّا آخرُ موضعٍ حديثٍ عند الخادم**، وإلّا لا موضع.
// و`hook` أثرُ التدقيق في المعاملة نفسِها (`XG-20`).
func (s *Service) HandGoods(ctx context.Context, orderID, driverID string, lat, lng *float64,
	hook func(context.Context, dbtx.Querier) error) (time.Time, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return time.Time{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var (
		status string
		owner  *string
		to     *string
		handed *time.Time
	)
	// **القفلُ داخل المعاملة** — ضغطتان متزامنتان لا تكتبان وقتين.
	if err := tx.QueryRow(ctx, `
		SELECT status, driver_id::text, return_to, goods_handed_at
		FROM orders WHERE id = $1 FOR UPDATE`, orderID).
		Scan(&status, &owner, &to, &handed); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, ErrNotDriversOrder
		}
		return time.Time{}, err
	}
	// **وطلبُ غيره كأنّه لم يكن** — لا يُقال له إنّ الطلبَ موجود.
	if owner == nil || *owner != driverID {
		return time.Time{}, ErrNotDriversOrder
	}
	if handed != nil {
		return time.Time{}, ErrGoodsAlreadyHanded
	}
	if status != StFailed || to == nil {
		return time.Time{}, ErrNoReturnTrip
	}

	var at time.Time
	if lat != nil && lng != nil {
		err = tx.QueryRow(ctx, `
			UPDATE orders SET goods_handed_at = now(),
			       goods_handed_point = ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography,
			       updated_at = now()
			WHERE id = $1 RETURNING goods_handed_at`, orderID, *lat, *lng).Scan(&at)
	} else {
		// **وموضعُ الخادم إن كان حديثاً** — ربعُ ساعةٍ كما في كلّ قياسٍ للسائق.
		err = tx.QueryRow(ctx, `
			UPDATE orders SET goods_handed_at = now(),
			       goods_handed_point = (SELECT u.last_location FROM users u
			                             WHERE u.id = $2::uuid
			                               AND u.last_location_at > now() - interval '15 minutes'),
			       updated_at = now()
			WHERE id = $1 RETURNING goods_handed_at`, orderID, driverID).Scan(&at)
	}
	if err != nil {
		return time.Time{}, err
	}
	if hook != nil {
		if err := hook(ctx, tx); err != nil {
			return time.Time{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return time.Time{}, err
	}
	s.pub.Publish("ops", map[string]any{"type": "order"})
	s.pub.Publish("driver:"+driverID, map[string]any{"type": "order"})
	return at, nil
}
