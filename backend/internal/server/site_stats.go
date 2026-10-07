package server

// ══════════════════════════════════════════════════════════════════════
// **زوّارُ الموقع وتحميلاتُ التطبيقات** (طلبُ المالك ٢٠٢٦-١٠-٠٧)
// ══════════════════════════════════════════════════════════════════════
//
// **كم شخصاً زار الموقع، وكم شخصاً نزّل كلَّ تطبيق — باليوم وبالمجموع.**
//
//   - الزيارةُ: `POST /public/visit` — يناديه الموقعُ مرّةً في الجلسة.
//   - التحميلُ: يُسجَّل في `GET /public/app/{key}` نفسِه.
//   - العرضُ: `GET /admin/site-stats` لرئيسيّة اللوحة.
//
// **والشخصُ بصمةٌ لا عنوان**: sha256 من «العنوان|المتصفّح» مقطوعةٌ إلى ٣٢
// خانة. **ويُعدّ مرّةً في اليوم لكلّ نوع** (`ON CONFLICT DO NOTHING`) —
// **فتنزيلٌ واحدٌ مقطّعٌ بطلباتِ مدىً لا يصير عشرة.**
//
// **والتسجيلُ لا يُبطئ التنزيلَ ولا يُسقطه** — في خلفيّةٍ بمهلةٍ قصيرة،
// **وخطؤه يُبلع.** واليومُ يومُ دمشق (`platform.Location`).

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/platform"
	"github.com/servacode/rahalgo/backend/internal/release"
)

const (
	hitVisit = "visit"
	// hitDownloadPrefix + مفتاحُ التطبيق (`release.Apps`).
	hitDownloadPrefix = "download:"
	// visitMaxPerHour **حدُّ نداءات الزيارة من عنوانٍ واحد** — والصفُّ
	// يتفرّد أصلاً، **والحدُّ يحمي القاعدةَ من الطَّرق لا العدَّ من التكرار.**
	visitMaxPerHour = 120
	// systemUserPhone **حسابُ النظام** (محفظةُ الاحتباس) — ليس زبوناً.
	systemUserPhone = "+000000000001"
)

// visitorOf **بصمةُ الشخص** — لا يُحفظ العنوانُ الخامّ.
func visitorOf(r *http.Request) string {
	sum := sha256.Sum256([]byte(clientIP(r) + "|" + r.UserAgent()))
	return hex.EncodeToString(sum[:])[:32]
}

// siteDay **يومُ دمشق** بصيغة `YYYY-MM-DD`.
func siteDay(t time.Time) string {
	return t.In(platform.Location()).Format("2006-01-02")
}

// recordHit **يُسجّل زيارةً أو تحميلاً — مرّةً في اليوم للشخص.**
func (s *Server) recordHit(ctx context.Context, kind, visitor string, at time.Time) error {
	_, err := s.pg.Exec(ctx, `
		INSERT INTO site_hits (day, kind, visitor) VALUES ($1::date, $2, $3)
		ON CONFLICT DO NOTHING`, siteDay(at), kind, visitor)
	return err
}

// recordHitAsync **في الخلفيّة وبمهلةٍ قصيرة** — ولا يُنتظر ولا يُبلَّغ خطؤه.
func (s *Server) recordHitAsync(r *http.Request, kind string) {
	if s.pg == nil {
		return
	}
	visitor, at := visitorOf(r), time.Now()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = s.recordHit(ctx, kind, visitor, at)
	}()
}

// handlePublicVisit **زيارةٌ للموقع** — بلا حساب وبلا جسم، ويردّ ٢٠٤.
func (s *Server) handlePublicVisit(w http.ResponseWriter, r *http.Request) {
	key := "visit:ip:" + clientIP(r)
	if n, err := s.incr(r.Context(), key); err == nil {
		if n == 1 && s.rdb != nil {
			s.rdb.Expire(r.Context(), key, time.Hour)
		}
		if n > visitMaxPerHour {
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	_ = s.recordHit(ctx, hitVisit, visitorOf(r), time.Now())
	w.WriteHeader(http.StatusNoContent)
}

type siteApps struct {
	Customer int64 `json:"customer"`
	Driver   int64 `json:"driver"`
	Merchant int64 `json:"merchant"`
	Rep      int64 `json:"rep"`
}

func (a *siteApps) set(key string, n int64) {
	switch key {
	case "customer":
		a.Customer = n
	case "driver":
		a.Driver = n
	case "merchant":
		a.Merchant = n
	case "rep":
		a.Rep = n
	}
}

type siteToday struct {
	Visits      int64    `json:"visits"`
	Downloads   siteApps `json:"downloads"`
	NewAccounts int64    `json:"new_accounts"`
}

type siteTotals struct {
	VisitsUniqueEver int64    `json:"visits_unique_ever"`
	VisitsDaysSum    int64    `json:"visits_days_sum"`
	Downloads        siteApps `json:"downloads"`
	Accounts         int64    `json:"accounts"`
}

type siteDayRow struct {
	Day         string `json:"day"`
	Visits      int64  `json:"visits"`
	Downloads   int64  `json:"downloads"`
	NewAccounts int64  `json:"new_accounts"`
}

type siteStats struct {
	Today  siteToday    `json:"today"`
	Totals siteTotals   `json:"totals"`
	Series []siteDayRow `json:"series"`
}

// accountsWhere **الحساباتُ الحقيقيّة** — بلا حساب النظام ولا المحذوف.
const accountsWhere = `phone <> '` + systemUserPhone + `' AND deleted_at IS NULL`

// handleSiteStats **زوّارُ الموقع والتحميلاتُ والحساباتُ الجديدة.**
//
// **والقدرةُ تُفحص هنا أيضاً** كرئيسيّة المدير — `platform.overview`.
func (s *Server) handleSiteStats(w http.ResponseWriter, r *http.Request) {
	if !hasCap(r, authz.PlatformOverview) {
		s.respondErr(w, errForbidden)
		return
	}
	days := 30
	if v, err := strconv.Atoi(r.URL.Query().Get("days")); err == nil {
		days = v
	}
	if days < 1 {
		days = 1
	}
	if days > 90 {
		days = 90
	}
	out, err := s.buildSiteStats(r.Context(), time.Now(), days)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (s *Server) buildSiteStats(ctx context.Context, now time.Time, days int) (siteStats, error) {
	today := siteDay(now)
	out := siteStats{Series: []siteDayRow{}}

	// ── اليوم والمجموع بالنوع ─────────────────────────────────────
	rows, err := s.pg.Query(ctx, `
		SELECT kind,
		       count(DISTINCT visitor) FILTER (WHERE day = $1::date),
		       count(DISTINCT visitor),
		       count(*)
		FROM site_hits GROUP BY kind`, today)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var kind string
		var nToday, nEver, nDays int64
		if err := rows.Scan(&kind, &nToday, &nEver, &nDays); err != nil {
			rows.Close()
			return out, err
		}
		if kind == hitVisit {
			out.Today.Visits, out.Totals.VisitsUniqueEver, out.Totals.VisitsDaysSum = nToday, nEver, nDays
			continue
		}
		if len(kind) > len(hitDownloadPrefix) && kind[:len(hitDownloadPrefix)] == hitDownloadPrefix {
			app := kind[len(hitDownloadPrefix):]
			out.Today.Downloads.set(app, nToday)
			out.Totals.Downloads.set(app, nEver)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return out, err
	}

	// ── الحسابات ─────────────────────────────────────────────────
	if err := s.pg.QueryRow(ctx, `
		SELECT count(*),
		       count(*) FILTER (WHERE (created_at AT TIME ZONE $2)::date = $1::date)
		FROM users WHERE `+accountsWhere, today, platform.TZ).
		Scan(&out.Totals.Accounts, &out.Today.NewAccounts); err != nil {
		return out, err
	}

	// ── السلسلة: كلُّ يومٍ ولو كان صفراً ─────────────────────────
	rows, err = s.pg.Query(ctx, `
		WITH d AS (
			SELECT generate_series($1::date - ($2::int - 1), $1::date, interval '1 day')::date AS day
		),
		h AS (
			SELECT day,
			       count(*) FILTER (WHERE kind = 'visit') AS visits,
			       count(*) FILTER (WHERE kind LIKE 'download:%') AS downloads
			FROM site_hits WHERE day >= $1::date - ($2::int - 1)
			GROUP BY day
		),
		u AS (
			SELECT (created_at AT TIME ZONE $3)::date AS day, count(*) AS n
			FROM users
			WHERE `+accountsWhere+`
			  AND created_at >= (($1::date - ($2::int - 1))::timestamp AT TIME ZONE $3) - interval '1 day'
			GROUP BY 1
		)
		SELECT to_char(d.day, 'YYYY-MM-DD'), COALESCE(h.visits, 0),
		       COALESCE(h.downloads, 0), COALESCE(u.n, 0)
		FROM d LEFT JOIN h USING (day) LEFT JOIN u USING (day)
		ORDER BY d.day`, today, days, platform.TZ)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var d siteDayRow
		if err := rows.Scan(&d.Day, &d.Visits, &d.Downloads, &d.NewAccounts); err != nil {
			return out, err
		}
		out.Series = append(out.Series, d)
	}
	return out, rows.Err()
}

// downloadHitKind **نوعُ التحميل لتطبيقٍ معروف.**
func downloadHitKind(app release.App) string { return hitDownloadPrefix + app.Key }
