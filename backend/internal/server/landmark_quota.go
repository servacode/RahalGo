package server

// ══════════════════════════════════════════════════════════════════════
// **معالمُ السائقين: خمسةٌ جديدةٌ كلَّ يوم** (قرارُ المالك ٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// «لازم كل سائق باليوم يرسل اقل شي 5 معالم، واذا ارسلو شي قديم طالعه، والي ما يبعث
// معالم 5 باليوم يصله تنبيه انه لم تكتمل المعالم ال5 الخاصة بك بعد».
//
//  ١ · **«موجود من قبل»** — معلمٌ بالاسم والعائلة نفسِهما (`maplearn.SameLandmark`) ضمن
//      المسافة (`drivers.landmarks_duplicate_radius_m`): على الخريطة، أو متجرٌ لنا، أو أرسله
//      سائقٌ — هو أو غيرُه — ولم يُرفض. **فيُقال له ما هو** (٤٠٩ `landmark_exists`)
//      **ويُحفظ صفُّه `duplicate`**: لا يُعدّ، ولا يظهر للموافقة ولا على خريطة العمليّات.
//  ٢ · **العدُّ بيوم المنصّة** (`platform_day`، من ساعة فتح الدوام) — ما ليس مكرّراً ولا
//      مرفوضاً، **ورفضُ الإدارة يُنقصه**: فالمعلمُ الوهميُّ لا يفيد صاحبَه.
//  ٣ · **والموقعُ من نبضة السائق** — التطبيقُ يأخذ بين طلبين آخرَ قراءةٍ للملاحة،
//      **أي مكانَ آخر تسليم**. فإن ابتعدت عن نبضةٍ حديثةٍ دقيقة حُفظ المعلمُ على النبضة.
//
// **ونسخةُ الميدان 0.7.0 تعمل بلا تحديث**: النجاحُ يُفكّ `JsonObject` فتُتجاهل حقولُه
// الجديدة، **ونصُّ الـ٤٠٩ في `details.notice`** — والتطبيقُ يعرضه على أيّ رمز (`ApiErrors.kt`).
// **وكلُّ قيم `details` نصوص**: التطبيقُ يقرؤها `Map<String,String>`، ورقمٌ فيها يُسقط فكَّ
// الردّ كلَّه فيقرأ السائقُ «تعذّر إتمام الطلب».

import (
	"context"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/maplearn"
)

const (
	// landmarkPingMaxAge **نبضةٌ أقدمُ لا تقول أين هو الآن** — كما في «موقعك الصح هون؟».
	landmarkPingMaxAge = 2 * time.Minute
	// landmarkPingMaxAccM **ونبضةٌ بدقّةٍ أسوأَ لا تصحّح شيئاً.**
	landmarkPingMaxAccM = 50.0
	// landmarkPingFixAfterM **أبعدُ من هذا عن النبضة ⇒ نقطةُ التطبيق قديمة** (آخرُ تسليم).
	// **وأقربُ منه لا يُبدَّل**: النبضةُ كلَّ دقيقةٍ والسائقُ يمشي بينها.
	landmarkPingFixAfterM = 150.0
	// landmarkNearLimit **ما يُقرأ حول النقطة** — ضمن ثلاثمئة مترٍ لا يكون أكثر.
	landmarkNearLimit = 500
)

// errLandmarkExists **المعلمُ موجودٌ من قبل** — والنصُّ والعددُ في التفصيل.
var errLandmarkExists = httpx.NewError(http.StatusConflict, "landmark_exists", "errors.conflict")

// landmarkText **يملأ نصّاً من `notifTitles`** — `{today}` وأخواتُها.
func landmarkText(tpl string, kv ...string) string {
	return strings.NewReplacer(kv...).Replace(tpl)
}

// landmarkClassFamily **عائلةُ صنفِ البلاطة بأنواع السائق السبعة** — «other» لصنفٍ معروفٍ لا
// نوعَ له عندنا (حديقة، مخفر…)، **وفارغٌ لما لا يُرسم** (أسماءُ أحياءٍ ومقابر ومكاتب): **لا
// يُقارَن إلّا بالاسم كاملاً بحرفه** (`maplearn.SameLandmark`).
func landmarkClassFamily(class string) string {
	switch class {
	case "":
		return ""
	case "place_of_worship":
		return "mosque"
	case "pharmacy":
		return "pharmacy"
	case "school", "kindergarten", "college", "university":
		return "school"
	case "hospital", "doctors", "dentist":
		return "hospital"
	case "shop", "grocery", "bakery", "restaurant", "cafe", "bank", "fuel", "lodging":
		return "shop"
	case "attraction":
		return "roundabout"
	}
	return "other"
}

// storeFamily **عائلةُ متجرٍ لنا** — محلٌّ، **إلّا أن يقول اسمُه غيرَ ذلك** («صيدلية الشفاء»).
func storeFamily(k maplearn.LandmarkKey) string {
	if k.Kind != "" && k.Kind != "other" {
		return k.Kind
	}
	return "shop"
}

// landmarkSubmit **ما أرسله السائق.**
type landmarkSubmit struct {
	Name, Kind string
	Lat, Lng   float64
}

// landmarkMatch **ما طابقه** — `map` · `store` · `suggestion` (سائقٌ آخر) · `self` (هو نفسُه).
type landmarkMatch struct {
	Source   string
	Ref      string
	Name     string
	Distance float64
}

// landmarkOutcome **نتيجةُ الإرسال.**
type landmarkOutcome struct {
	ID int64
	// Day يومُ المنصّة المختومُ على الصفّ.
	Day string
	// Today معالمُه المحسوبةُ اليوم — بعد هذا الإرسال.
	Today int
	// Quota المطلوبُ باليوم (صفرٌ: العدُّ مطفأ).
	Quota int
	// Dup **ما طابقه** — وفارغٌ: معلمٌ جديد.
	Dup *landmarkMatch
	// FromPing **حُفظ على نبضة السائق** لا على نقطة التطبيق.
	FromPing bool
}

// landmarkQuota **المطلوبُ من كلّ سائقٍ باليوم** — وصفرٌ يطفئ العدَّ والتذكير.
func (s *Server) landmarkQuota(ctx context.Context) int {
	return max(int(s.settings.GetInt(ctx, "drivers.landmarks_daily_quota")), 0)
}

// landmarkCountedOn **معالمُه المحسوبةُ بيومٍ ما** — بانتظار الموافقة أو موافَقٌ عليها.
func landmarkCountedOn(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, driverID, day string) (int, error) {
	var n int
	err := q.QueryRow(ctx, `
		SELECT count(*) FROM landmark_suggestions
		 WHERE suggested_by = $1::uuid AND platform_day = $2::date
		   AND status IN ('pending', 'approved')`, driverID, day).Scan(&n)
	return n, err
}

// submitLandmark **يحفظ ما أرسله السائق** — جديداً بانتظار الموافقة أو مكرّراً — ويعدّ يومَه.
//
// **و`now` يُمرَّر** فيُقاس يومُ المنصّة وحداثةُ النبضة بلا ساعةٍ حقيقيّة.
func (s *Server) submitLandmark(ctx context.Context, uid string, in landmarkSubmit, now time.Time) (landmarkOutcome, error) {
	out := landmarkOutcome{Day: siteDay(now, s.dayStartHour(ctx)), Quota: s.landmarkQuota(ctx)}
	radius := float64(s.settings.GetInt(ctx, "drivers.landmarks_duplicate_radius_m"))
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// **ومرّةً للسائق في اللحظة** — نقرتان، أو إعادةُ إرسالٍ بعد ضياع الردّ (لا مفتاحَ تفرّدٍ
	// في التطبيق)، **ترى الثانيةُ الأولى** فتُقال «بعتّه قبل» ولا يُكتب صفّان.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('landmark:' || $1, 0))`, uid); err != nil {
		return out, err
	}

	lat, lng := in.Lat, in.Lng
	var gap *int
	var pLat, pLng, d float64
	err = tx.QueryRow(ctx, `
		SELECT ST_Y(t.at::geometry), ST_X(t.at::geometry),
		       ST_Distance(t.at, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography)
		  FROM driver_track t
		 WHERE t.driver_id = $1::uuid AND t.recorded_at > $4
		   AND COALESCE(t.accuracy_m, 0) <= $5 AND NOT t.mocked
		 ORDER BY t.recorded_at DESC LIMIT 1`,
		uid, in.Lng, in.Lat, now.Add(-landmarkPingMaxAge), landmarkPingMaxAccM).Scan(&pLat, &pLng, &d)
	switch {
	case err == nil:
		g := int(math.Round(d))
		gap = &g
		if d > landmarkPingFixAfterM {
			lat, lng, out.FromPing = pLat, pLng, true
		}
	case errors.Is(err, pgx.ErrNoRows):
		// **خارجَ الدوام لا نبضة** — يُحفظ كما جاء، وتقول اللوحةُ «بلا تحقّق».
	default:
		return out, err
	}

	match, err := landmarkDuplicate(ctx, tx, uid, in, lat, lng, radius)
	if err != nil {
		return out, err
	}
	// **والضغطةُ الثانيةُ على المكرَّر نفسِه لا تكتب صفّاً** (مراجعة ٢٠٢٦-١١-٠٥) — في 0.7.0 تبقى
	// الصفحةُ مفتوحةً بعد الرسالة الحمراء فيضغط ثانيةً، **وكان كلُّ ضغطةٍ صفّاً يزيد «أرسل»
	// و«مكرّر» في اللوحة والملخّص.** فالاسمُ نفسُه على ما طابقه نفسِه في يومه ⇒ الردُّ نفسُه.
	if match != nil {
		var prev int64
		err := tx.QueryRow(ctx, `
			SELECT id FROM landmark_suggestions
			 WHERE suggested_by = $1::uuid AND platform_day = $2::date AND status = 'duplicate'
			   AND dup_source = $3 AND dup_ref = $4 AND btrim(name) = btrim($5)
			 ORDER BY id DESC LIMIT 1`, uid, out.Day, match.Source, match.Ref, in.Name).Scan(&prev)
		switch {
		case err == nil:
			out.ID, out.Dup = prev, match
			out.Today, err = landmarkCountedOn(ctx, tx, uid, out.Day)
			return out, err
		case !errors.Is(err, pgx.ErrNoRows):
			return out, err
		}
	}
	status := "pending"
	var src, ref, dname *string
	var ddist *int
	if match != nil {
		status = "duplicate"
		dm := int(math.Round(match.Distance))
		src, ref, dname, ddist = &match.Source, &match.Ref, &match.Name, &dm
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO landmark_suggestions (name, kind, point, suggested_by, status, platform_day,
		            sent_point, point_from_ping, ping_gap_m, dup_source, dup_ref, dup_name, dup_distance_m)
		VALUES ($1, $2, ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography, $5::uuid, $6, $7::date,
		        CASE WHEN $8 THEN ST_SetSRID(ST_MakePoint($9, $10), 4326)::geography END,
		        $8, $11, $12, $13, $14, $15)
		RETURNING id`,
		in.Name, in.Kind, lng, lat, uid, status, out.Day,
		out.FromPing, in.Lng, in.Lat, gap, src, ref, dname, ddist).Scan(&out.ID); err != nil {
		return out, err
	}
	if out.Today, err = landmarkCountedOn(ctx, tx, uid, out.Day); err != nil {
		return out, err
	}
	if err := tx.Commit(ctx); err != nil {
		return out, err
	}
	out.Dup = match
	return out, nil
}

// landmarkDuplicate **أهو معلمٌ نعرفه؟** — المعالمُ ومتاجرُنا واقتراحاتُ السائقين ضمن المسافة.
//
// **وأدقُّ الأخبار أوّلاً**: «بعتّه أنت» ثمّ «على الخريطة» ثمّ «بعتو سائقٌ قبلك» — وفي كلٍّ
// أقربُها. **والمرفوضُ والمكرَّرُ ليسا موجودَين.**
func landmarkDuplicate(ctx context.Context, tx pgx.Tx, uid string, in landmarkSubmit, lat, lng, radius float64) (*landmarkMatch, error) {
	rows, err := tx.Query(ctx, `
		WITH p AS (SELECT ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography AS g)
		SELECT 'map', l.id::text, l.name, l.kind, '', '', ST_Distance(l.point, p.g) AS d
		  FROM map_landmarks l, p
		 WHERE ST_DWithin(l.point, p.g, $3) AND btrim(l.name) <> ''
		UNION ALL
		SELECT 'store', m.id::text, m.name, '', '', '', ST_Distance(m.location, p.g)
		  FROM merchants m, p
		 WHERE m.status = 'active' AND m.location IS NOT NULL AND ST_DWithin(m.location, p.g, $3)
		UNION ALL
		SELECT 'suggestion', x.id::text, x.name, x.kind, COALESCE(x.suggested_by::text, ''), x.status,
		       ST_Distance(x.point, p.g)
		  FROM landmark_suggestions x, p
		 WHERE x.status IN ('pending', 'approved') AND ST_DWithin(x.point, p.g, $3)
		 ORDER BY d
		 LIMIT `+strconv.Itoa(landmarkNearLimit), lng, lat, radius)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	key := maplearn.LandmarkKeyOf(in.Name)
	fam := maplearn.DriverFamily(in.Kind, key)
	var self, onMap, other *landmarkMatch
	for rows.Next() {
		var src, id, name, kind, by, status string
		var dist float64
		if err := rows.Scan(&src, &id, &name, &kind, &by, &status, &dist); err != nil {
			return nil, err
		}
		ek := maplearn.LandmarkKeyOf(name)
		var efam string
		switch src {
		case "map":
			efam = landmarkClassFamily(landmarkClass(kind))
		case "store":
			efam = storeFamily(ek)
		default:
			efam = maplearn.DriverFamily(kind, ek)
		}
		// **وما أرسله هو نفسُه يُطابَق بالاسم ولو بدّل النوع** (مراجعة ٢٠٢٦-١١-٠٥) — «جامع النور»
		// بنوع جامع ثمّ بنوع محلّ كان يُحسب مرّتين.
		own := src == "suggestion" && by == uid
		if !maplearn.SameLandmark(key, fam, ek, efam) && !(own && maplearn.SameOwnLandmark(key, fam, ek, efam)) {
			continue
		}
		m := &landmarkMatch{Source: src, Ref: id, Name: strings.TrimSpace(name), Distance: dist}
		switch {
		case own:
			if self == nil {
				m.Source = "self"
				self = m
			}
		// **وما وافقت عليه الإدارةُ على الخريطة** — لا «بانتظار الموافقة».
		case src == "suggestion" && status == "pending":
			if other == nil {
				other = m
			}
		default:
			if onMap == nil {
				if src == "suggestion" {
					m.Source = "map"
				}
				onMap = m
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, m := range []*landmarkMatch{self, onMap, other} {
		if m != nil {
			return m, nil
		}
	}
	return nil, nil
}

// todaySuffix **«. معالمك اليوم: ٢ من ٥»** — وفارغٌ والعدُّ مطفأ.
func (o landmarkOutcome) todaySuffix() string {
	if o.Quota <= 0 {
		return ""
	}
	return landmarkText(notifTitles.landmarkToday,
		"{today}", strconv.Itoa(o.Today), "{quota}", strconv.Itoa(o.Quota))
}

// notice **ما يُقال للسائق** — انحسب، أو موجودٌ من قبل وما هو.
func (o landmarkOutcome) notice() string {
	if o.Dup == nil {
		if o.Quota <= 0 {
			return notifTitles.landmarkSent
		}
		return landmarkText(notifTitles.landmarkCounted,
			"{today}", strconv.Itoa(o.Today), "{quota}", strconv.Itoa(o.Quota))
	}
	var head string
	switch o.Dup.Source {
	case "self":
		head = notifTitles.landmarkExistsSelf
	case "suggestion":
		head = notifTitles.landmarkExistsPending
	default:
		head = notifTitles.landmarkExistsMap
	}
	return landmarkText(head, "{name}", o.Dup.Name,
		"{distance}", strconv.Itoa(int(math.Round(o.Dup.Distance)))) + o.todaySuffix()
}

// existsError **الـ٤٠٩ بتفصيله — نصوصاً كلُّها.**
func (o landmarkOutcome) existsError() error {
	e := *errLandmarkExists
	e.Details = map[string]any{
		"notice":        o.notice(),
		"today":         strconv.Itoa(o.Today),
		"quota":         strconv.Itoa(o.Quota),
		"day":           o.Day,
		"existing_name": o.Dup.Name,
		"distance_m":    strconv.Itoa(int(math.Round(o.Dup.Distance))),
		"source":        o.Dup.Source,
	}
	return &e
}

// handleDriverLandmarksToday **معالمُه اليوم من المطلوب** — سطرُ «معالمك اليوم: ٢ من ٥» فوق صفحة
// «أضف معلماً» (قرارُ المالك ٢٠٢٦-١١-٠٥). GET /driver/landmarks/today
func (s *Server) handleDriverLandmarksToday(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	day := siteDay(time.Now(), s.dayStartHour(ctx))
	n, err := landmarkCountedOn(ctx, s.pg, userIDFrom(r), day)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"day": day, "today": n, "quota": s.landmarkQuota(ctx)})
}

// errLandmarkBadPoint **لا موقعَ صالحاً** — بنصٍّ يقول ما يفعل، لا «موقع التوصيل».
func errLandmarkBadPoint() error {
	e := *httpx.NewError(http.StatusUnprocessableEntity, "bad_point", "errors.validation")
	e.Details = map[string]any{"notice": notifTitles.landmarkBadPoint}
	return &e
}
