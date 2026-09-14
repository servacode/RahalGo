package opsmap

// ══════════════════════════════════════════════════════════════════════
// **إشارةُ الطلب الجغرافيّ — نيّتان لا واحدة** (`CR`، ٢٠٢٦-٠٩-١٤)
// ══════════════════════════════════════════════════════════════════════
//
//	KindCoverage  **الخدمةُ في مدينتي وعنواني خارجَ الشكل** ⇒ أضِف منطقتي
//	KindInterest  **لم تصل الخدمةُ إلى مكاني بعد**          ⇒ أخبرني
//
// **والرسالتان لا تتبادلان** — **ومن قيل له «سنأخذ منطقتك بعين
// الاعتبار» ونحن لم نصل مدينتَه أصلاً وُعد بما لا يقع.**
//
// # والتفرّدُ بالخليّة لا بالتساوي العشريّ
//
// **وضغطتان على النقطة نفسِها تختلفان في الخانة السابعة** — **فيُولَد
// طلبان لمكانٍ واحد.** **والخليّةُ شبكةُ `opsmap` عينُها** (`CellSizeDeg`
// ≈ ١٫١ كم) — **وهي التي تُجمَّع بها الخريطةُ الإداريّة أصلاً**، **فلا
// حدٌّ ثانٍ يُخترَع.**
//
// **والتفرّدُ لا يمحو شدّةَ الطلب** — **`requests` يُعَدّ.**

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// أنواعُ الإشارة.
const (
	// KindCoverage **أضِف منطقتي إلى نطاق التوصيل.**
	KindCoverage = "coverage_request"
	// KindInterest **أخبرني عند توفّر الخدمة هنا.**
	KindInterest = "service_interest"
)

// ValidKind **أنوعٌ نعرفه؟**
func ValidKind(k string) bool { return k == KindCoverage || k == KindInterest }

// SignalResult **ما وقع** — وتقرؤه الشاشةُ لتقول الحال.
type SignalResult struct {
	ID string `json:"id"`
	// Outcome **`created` أو `already_registered`** — **ولا يُردّ عطبٌ
	// على ضغطةٍ مكرّرة**: **زرٌّ يبدو معطوباً يُضغط ثالثةً ورابعة.**
	Outcome string `json:"outcome"`
	// Requests **كم مرّةً طُلب هذا المكان من هذا الحساب.**
	Requests int `json:"requests"`
}

// أحوالُ النتيجة.
const (
	OutcomeCreated    = "created"
	OutcomeRegistered = "already_registered"
	OutcomeReactived  = "reactivated"
)

// Signal **ما يُرسَل** — والإحداثيّةُ وحدَها تحكم الموضع.
//
// **ولا يُقرأ نصُّ العنوان جغرافيا** — **يُحفَظ ليقرأه المكتبُ لا
// ليُستنتَج منه مكان.**
type Signal struct {
	Kind    string
	Lat     float64
	Lng     float64
	Address string
	Source  string
	// Reason **سببُ الإتاحة كما قضاه الخادمُ لحظتَها** — لا كما أرسله
	// العميل.
	Reason string
	// CityID و GovID **ما حُلّ منهما** — وفارغٌ يبقى فارغاً.
	//
	// **ولا يُخترَع مكانٌ لنقطةٍ لا تعرفها المنصّة** (`AG-08`).
	CityID string
	GovID  string
}

// ErrBadSignal **إشارةٌ لا تُقبَل.**
var ErrBadSignal = httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")

// Record **يسجّل الإشارةَ أو يدمجها في سابقتها.**
//
// **والدمجُ لصاحب حسابٍ فقط** — **ومن لا حساب له لا هويّةَ تجمع
// ضغطتيه**، **ولا تُخترَع بصمةُ جهازٍ لأجل ذلك** (نهيُ المالك).
//
// **والإحياءُ يُعاد**: **من ألغى «أخبرني» ثمّ عاد فضغطها يُعاد
// اشتراكُه** — **ولا يُحبَس على إلغاءٍ قديم.**
func Record(ctx context.Context, e Execer, userID string, in Signal) (SignalResult, error) {
	if !ValidKind(in.Kind) {
		return SignalResult{}, ErrBadSignal
	}
	if in.Lat < -90 || in.Lat > 90 || in.Lng < -180 || in.Lng > 180 {
		return SignalResult{}, httpx.NewError(http.StatusBadRequest, "bad_point", "errors.bad_point")
	}
	src := strings.TrimSpace(in.Source)
	if src == "" {
		src = "customer_app"
	}
	cy, cx := snap(in.Lat), snap(in.Lng)

	var (
		uid  *string
		city *string
		gov  *string
	)
	if userID != "" {
		uid = &userID
	}
	if in.CityID != "" {
		city = &in.CityID
	}
	if in.GovID != "" {
		gov = &in.GovID
	}

	// **ومن لا حساب له يُكتب صفّاً جديداً** — **ولا مفتاحَ تفرّدٍ له.**
	if uid == nil {
		var id string
		err := e.QueryRow(ctx, `
			INSERT INTO coverage_requests
			  (user_id, at, address_text, city_id, governorate_id, source,
			   kind, cell_y, cell_x, reason)
			VALUES (NULL, ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography,
			        $3, $4::uuid, $5::uuid, $6, $7, $8, $9, $10)
			RETURNING id::text`,
			in.Lat, in.Lng, strings.TrimSpace(in.Address), city, gov, src,
			in.Kind, cy, cx, in.Reason).Scan(&id)
		if err != nil {
			return SignalResult{}, err
		}
		return SignalResult{ID: id, Outcome: OutcomeCreated, Requests: 1}, nil
	}

	// **ولصاحب الحساب دمجٌ ذرّيٌّ** — **ولا قراءةٌ ثمّ كتابةٌ يتسلّل
	// بينهما نداءٌ ثانٍ فيُولَد صفّان.**
	var (
		id       string
		requests int
		inserted bool
	)
	err := e.QueryRow(ctx, `
		INSERT INTO coverage_requests
		  (user_id, at, address_text, city_id, governorate_id, source,
		   kind, cell_y, cell_x, reason)
		VALUES ($1::uuid, ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography,
		        $4, $5::uuid, $6::uuid, $7, $8, $9, $10, $11)
		ON CONFLICT (user_id, kind, cell_y, cell_x) WHERE user_id IS NOT NULL
		DO UPDATE SET
		    requests     = coverage_requests.requests + 1,
		    last_seen_at = now(),
		    updated_at   = now(),
		    -- **ويُحيا المُلغى** — ومن عاد فضغطها أراد الاشتراك.
		    active       = true,
		    -- **وما حُلّ من المكان يُحدَّث** — فقد تُبذَر مدينةٌ لاحقاً.
		    city_id        = COALESCE(EXCLUDED.city_id, coverage_requests.city_id),
		    governorate_id = COALESCE(EXCLUDED.governorate_id, coverage_requests.governorate_id),
		    reason         = EXCLUDED.reason
		RETURNING id::text, requests, (xmax = 0)`,
		userID, in.Lat, in.Lng, strings.TrimSpace(in.Address), city, gov, src,
		in.Kind, cy, cx, in.Reason).Scan(&id, &requests, &inserted)
	if err != nil {
		return SignalResult{}, err
	}
	out := SignalResult{ID: id, Requests: requests, Outcome: OutcomeRegistered}
	if inserted {
		out.Outcome = OutcomeCreated
	}
	return out, nil
}

// CancelInterest **يُلغي اشتراكَ «أخبرني» لهذا المكان.**
//
// **ولا يُمحى صفٌّ** — **والطلبُ التاريخيُّ يبقى محسوباً في الكثافة**،
// **وإنّما يخرج من دائرة من يُخبَر.**
//
// **ولا يُلغى طلبُ التغطية بهذا** — **وهو واقعةٌ لا اشتراك.**
func CancelInterest(ctx context.Context, e Execer, userID string, lat, lng float64) error {
	if userID == "" {
		return httpx.NewError(http.StatusUnauthorized, "unauthorized", "errors.unauthorized")
	}
	cy, cx := snap(lat), snap(lng)
	tag, err := e.Exec(ctx, `
		UPDATE coverage_requests
		   SET active = false, updated_at = now()
		 WHERE user_id = $1::uuid AND kind = $2
		   AND cell_y = $3 AND cell_x = $4`,
		userID, KindInterest, cy, cx)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}
	return nil
}

// InterestActive **أهذا المكانُ مشترَكٌ فيه الآن لهذا الحساب؟**
//
// **وتقرؤه الشاشةُ لترسم حالَ الزرّ** — **وزرٌّ لا يعرف حالَه يُضغط
// مرّتين.**
func InterestActive(ctx context.Context, q Querier, userID, kind string, lat, lng float64) (bool, error) {
	if userID == "" {
		return false, nil
	}
	cy, cx := snap(lat), snap(lng)
	var active bool
	err := q.QueryRow(ctx, `
		SELECT active FROM coverage_requests
		 WHERE user_id = $1::uuid AND kind = $2 AND cell_y = $3 AND cell_x = $4`,
		userID, kind, cy, cx).Scan(&active)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return active, err
}

// ══════════════════════════════════════════════════════════════════════
// **الكثافةُ بالمكان الإداريّ — «كم طلباً في دمشق؟»**
// ══════════════════════════════════════════════════════════════════════
//
// **وتجميعُ الخلايا يقول «أين»** — **وهذا يقول «في أيّ مدينةٍ
// ومحافظة»**، **وهو سؤالُ التوسّع الأوّل.**

// PlaceDemand **صفٌّ في جدول الكثافة الإداريّة.**
type PlaceDemand struct {
	// GovernorateID و GovernorateName فارغان لما لم يُحَلّ.
	GovernorateID   string `json:"governorate_id,omitempty"`
	GovernorateName string `json:"governorate_name,omitempty"`
	CityID          string `json:"city_id,omitempty"`
	CityName        string `json:"city_name,omitempty"`
	Kind            string `json:"kind"`
	// People عددُ الحسابات المتميّزة · Signals مجموعُ المرّات.
	People  int `json:"people"`
	Signals int `json:"signals"`
}

// DemandByPlace **الكثافةُ مجموعةً بالمدينة والمحافظة.**
//
// **والمجهولُ إداريّاً يبقى صفّاً بلا اسم** — **ولا يُنسَب إلى محافظةٍ
// لم تُحَلّ**: **وطلبٌ في البادية لا يُحسَب على دمشق.**
func DemandByPlace(ctx context.Context, q Querier, kind string) ([]PlaceDemand, error) {
	rows, err := q.Query(ctx, `
		SELECT COALESCE(r.governorate_id::text, ''), COALESCE(g.name, ''),
		       COALESCE(r.city_id::text, ''),        COALESCE(c.name, ''),
		       r.kind,
		       count(DISTINCT COALESCE(r.user_id::text, r.id::text)),
		       COALESCE(sum(r.requests), 0)
		FROM coverage_requests r
		LEFT JOIN governorates g ON g.id = r.governorate_id
		LEFT JOIN cities c       ON c.id = r.city_id
		WHERE ($1 = '' OR r.kind = $1)
		GROUP BY 1, 2, 3, 4, 5
		ORDER BY 7 DESC
		LIMIT 500`, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PlaceDemand{}
	for rows.Next() {
		var p PlaceDemand
		if err := rows.Scan(&p.GovernorateID, &p.GovernorateName,
			&p.CityID, &p.CityName, &p.Kind, &p.People, &p.Signals); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
