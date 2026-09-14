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
	"strconv"
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

// ══════════════════════════════════════════════════════════════════════
// **استنتاجُ المكان — نصٌّ واحدٌ لا أربعة**
// ══════════════════════════════════════════════════════════════════════
//
// **والمُمرَّرُ يُقدَّم** (`COALESCE`) — **ومحرّكُ الإتاحة يعرف المدينةَ
// المُطفأةَ وهذا لا يعرفها**: **الاستنتاجُ يقرأ الفعّالةَ وحدَها كما
// كان يفعل البابُ القديم.**

// cityFrom استعلامُ المدينة من نقطةٍ بمعاملَي عرضٍ وطول.
func cityFrom(passed, latArg, lngArg string) string {
	return `COALESCE(` + passed + `::uuid, (
		SELECT c.id FROM cities c
		 WHERE c.active
		   AND ST_DWithin(c.center,
		        ST_SetSRID(ST_MakePoint(` + lngArg + `, ` + latArg + `), 4326)::geography,
		        c.radius_m)
		 ORDER BY ST_Distance(c.center,
		        ST_SetSRID(ST_MakePoint(` + lngArg + `, ` + latArg + `), 4326)::geography)
		 LIMIT 1))`
}

// govFrom محافظةُ تلك المدينة — تُستنتَج منها لا من النقطة.
func govFrom(passed, latArg, lngArg string) string {
	return `COALESCE(` + passed + `::uuid, (
		SELECT d.governorate_id FROM cities c
		  JOIN districts d ON d.id = c.district_id
		 WHERE c.id = ` + cityFrom(passed, latArg, lngArg) + `))`
}

// ══════════════════════════════════════════════════════════════════════
// **هويّةُ الهدف — ما وعد به الزرُّ** (`SI`، ٢٠٢٦-٠٩-١٤)
// ══════════════════════════════════════════════════════════════════════
//
// **والزرُّ قال «أخبرني عند توفّر الخدمة في دمشق»** — **لا «في هذه
// الخليّة».** **فمن ضغطها من عنوانين في دمشقَ هدفٌ واحد**، **ولو
// كانتا خليّتين متباعدتين.**
//
// **وطلبُ التغطية مكانيٌّ بطبعه** — **«أيُّ حيٍّ خارجَ النطاق عليه أكبرُ
// طلب؟» سؤالُ خلايا**، **ولو جُمع بالمدينة لصارت الرقّةُ صفّاً واحداً
// لا يقول أين يُوسَّع.**
//
// **وموضعٌ لا مدينةَ له هدفُه خليّتُه** — **ولا مكانَ يُسمّى ليكون
// هدفاً.**

// TargetKey **هويّةُ الهدف** — تُكتب مرّةً ولا تتبدّل.
func TargetKey(kind, cityID string, cy, cx float64) string {
	if kind == KindInterest && cityID != "" {
		return "city:" + cityID
	}
	return "cell:" + ftoa(cy) + "," + ftoa(cx)
}

// ftoa يكتب إحداثيّةَ خليّةٍ كما تكتبها القاعدة.
func ftoa(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }

// cityAt **أيُّ مدينةٍ تحوي هذه النقطة؟** — وفارغٌ إن لم تُعرَف.
//
// **وبالشرط عينِه الذي يصنّف به محرّكُ الإتاحة** — **فالمُطفأةُ تُعرَف
// كما تُعرَف الفعّالة**: **من اشترك في مدينةٍ لم تُطلَق يُلغي اشتراكَه
// منها.**
func cityAt(ctx context.Context, q Querier, lat, lng float64) string {
	var id string
	err := q.QueryRow(ctx, `
		SELECT c.id::text FROM cities c
		 WHERE ST_DWithin(c.center,
		        ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography, c.radius_m)
		 ORDER BY ST_Distance(c.center,
		        ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography)
		 LIMIT 1`, lat, lng).Scan(&id)
	if err != nil {
		return ""
	}
	return id
}

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

	// ══════════════════════════════════════════════════════════════════
	// **والمكانُ يُستنتَج حين لا يُمرَّر** (٢٠٢٦-٠٩-١٤)
	// ══════════════════════════════════════════════════════════════════
	//
	// **ومحرّكُ الإتاحة يحلّ المكانَ ويمرّره** — **ويعرف المدينةَ
	// المُطفأةَ أيضاً** (`city_not_supported`).
	//
	// **والبابُ القديمُ لا يمرّر شيئاً** — **وكان استعلامُه يستنتج
	// المدينةَ بنفسه**، **فلمّا مرّ بهذه الدالّة فقد الاستنتاجَ وسقط
	// فحصُه.** (قِيس: «لم تُستنتج مدينةُ الطلب».)
	//
	// **فيُستنتَج ما لم يُمرَّر** — **والمُمرَّرُ أدقُّ فيُقدَّم.**

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

	// **والهدفُ يُحسَب بما نعرفه الآن** — **ومدينةٌ لم تُمرَّر تُستنتَج
	// كما يستنتجها الاستعلامُ أدناه، فلا يفترق المفتاحُ عن الصفّ.**
	targetCity := in.CityID
	if targetCity == "" && in.Kind == KindInterest {
		targetCity = cityAt(ctx, e, in.Lat, in.Lng)
	}
	target := TargetKey(in.Kind, targetCity, float64(cy), float64(cx))

	cityExpr := cityFrom("$4", "$1", "$2")
	govExpr := govFrom("$5", "$1", "$2")
	cityExprU := cityFrom("$5", "$2", "$3")
	govExprU := govFrom("$6", "$2", "$3")

	// **ومن لا حساب له يُكتب صفّاً جديداً** — **ولا مفتاحَ تفرّدٍ له.**
	if uid == nil {
		var id string
		err := e.QueryRow(ctx, `
			INSERT INTO coverage_requests
			  (user_id, at, address_text, city_id, governorate_id, source,
			   kind, cell_y, cell_x, reason, target_key)
			VALUES (NULL, ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography,
			        $3, `+cityExpr+`, `+govExpr+`, $6, $7, $8, $9, $10, $11)
			RETURNING id::text`,
			in.Lat, in.Lng, strings.TrimSpace(in.Address), city, gov, src,
			in.Kind, cy, cx, in.Reason, target).Scan(&id)
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
		   kind, cell_y, cell_x, reason, target_key)
		VALUES ($1::uuid, ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography,
		        $4, `+cityExprU+`, `+govExprU+`, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (user_id, kind, target_key) WHERE user_id IS NOT NULL
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
		in.Kind, cy, cx, in.Reason, target).Scan(&id, &requests, &inserted)
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
	// **ويُلغى ما وعد به الزرُّ** — **ومن اشترك في دمشقَ يُلغي دمشق**،
	// **لا الخليّةَ التي وقف فيها يومَ ضغط.**
	//
	// **وحلبُ لا تُمَسّ** — **والهدفُ واحدٌ بعينه لا «كلُّ اشتراكاته».**
	cy, cx := snap(lat), snap(lng)
	target := TargetKey(KindInterest, cityAt(ctx, e, lat, lng), float64(cy), float64(cx))
	tag, err := e.Exec(ctx, `
		UPDATE coverage_requests
		   SET active = false, updated_at = now()
		 WHERE user_id = $1::uuid AND kind = $2 AND target_key = $3`,
		userID, KindInterest, target)
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
	city := ""
	if kind == KindInterest {
		city = cityAt(ctx, q, lat, lng)
	}
	var active bool
	err := q.QueryRow(ctx, `
		SELECT active FROM coverage_requests
		 WHERE user_id = $1::uuid AND kind = $2 AND target_key = $3`,
		userID, kind, TargetKey(kind, city, float64(cy), float64(cx))).Scan(&active)
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

// ══════════════════════════════════════════════════════════════════════
// **من يُخبَر يومَ تُطلَق مدينة** — سؤالُ الدفعة الثامنة (`SI-08`)
// ══════════════════════════════════════════════════════════════════════
//
// **ولا يُرسَل إشعارٌ اليوم** — **وإنّما يُثبَت أنّ السؤالَ يُجاب بلا
// تكرارٍ ولا تنقيةٍ عند الإرسال.**
//
// **والحسابُ مرّةً واحدةً ولو ضغط الزرَّ من عشرة عناوين** — **وهو ما
// كان يكسره التفرّدُ بالخليّة.**
//
// **والمُلغي لا يُستهدَف** — **ومن طلب ألّا يُخبَر لا يُخبَر.**

// InterestedInCity **الحساباتُ الساريةُ المشترِكةُ في هذه المدينة.**
func InterestedInCity(ctx context.Context, q Querier, cityID string) ([]string, error) {
	rows, err := q.Query(ctx, `
		SELECT user_id::text FROM coverage_requests
		 WHERE kind = $1 AND active AND user_id IS NOT NULL
		   AND target_key = $2
		 ORDER BY created_at`, KindInterest, "city:"+cityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
