package server

// ══════════════════════════════════════════════════════════════════════
// **طلباتُ التوسّع — قسمٌ مستقلٌّ وإبلاغٌ يدويّ** (قرارُ المالك ٢٠٢٦-١٠-٠٤)
// ══════════════════════════════════════════════════════════════════════
//
// **وكانت مخفيّةً تحت الخريطة** — فصارت قسماً له أرقامُه وجدولُه:
// محافظة ← مدينة ← منطقة، بعدد الأشخاص والطلبات وأوّلِ وآخرِ طلبٍ والحال.
//
// # والملغاةُ لا تُحسب
//
// **من ألغى «أخبرني» خرج من كلّ رقمٍ هنا** (`active = false`) — والصفُّ
// باقٍ في الكثافة التاريخيّة وحدَها.
//
// # والإبلاغُ بزرٍّ لا آليّاً
//
// **قد تُرسم التغطيةُ قبل تجهيز السائقين** — فلا يُبلَّغ أحدٌ عند حفظ
// منطقةٍ أو إطلاق مدينة، **بل حين يضغط المكتبُ «بلّغ المنتظرين الآن»**
// وقد رأى عددَهم قبل الضغط. **والتذكيرُ** عددُ من صاروا مغطَّين ولم
// يُبلَّغوا بعد.
//
// **والحكمُ بالتغطية عينُ حكم الإتاحة** (`CoverableAt`،
// `CityEffectivelyLaunched`) — فلا يفترق ما يقوله الجدولُ عمّا يُرسَل.

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/opsmap"
)

// أحوالُ صفّ التوسّع.
const (
	// expUncovered **خارجَ التغطية الفعليّة** — يُرسم له نطاق.
	expUncovered = "uncovered"
	// expReady **مُغطّى وفيه منتظرون لم يُبلَّغوا** — وهو التذكير.
	expReady = "ready"
	// expCovered **مُغطّى ولا منتظرَ يُبلَّغ.**
	expCovered = "covered"
)

// expansionRow صفٌّ في جدول التوسّع.
type expansionRow struct {
	// Key **هويّةُ الصفّ** — `zone:<id>` · `city:<id>` · `cell:<kind>:<y>,<x>`.
	// **والإبلاغُ يقبل الأوّلين وحدَهما.**
	Key  string `json:"key"`
	Kind string `json:"kind"`

	GovernorateID   string `json:"governorate_id"`
	GovernorateName string `json:"governorate_name"`
	CityID          string `json:"city_id"`
	CityName        string `json:"city_name"`
	ZoneID          string `json:"zone_id"`
	ZoneName        string `json:"zone_name"`
	// Label **آخرُ عنوانٍ كتبه طالب** — لما لا منطقةَ له.
	Label string `json:"label"`

	People   int       `json:"people"`
	Requests int       `json:"requests"`
	FirstAt  time.Time `json:"first_at"`
	LastAt   time.Time `json:"last_at"`

	// Lat/Lng **وسطُ الطلبات** — لـ«اعرض على الخريطة» و«ارسم تغطيةً هنا».
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`

	// Waiting **صفوفٌ لم تُبلَّغ** · Notified **صفوفٌ بُلّغت.**
	Waiting  int `json:"waiting"`
	Notified int `json:"notified"`
	// Notifiable **كم يُبلَّغ لو ضُغط الزرُّ الآن** — يُعرض قبل الضغط.
	Notifiable int    `json:"notifiable"`
	Status     string `json:"status"`
}

// expansionSummary الأرقامُ والجدول.
type expansionSummary struct {
	People      int `json:"people"`
	NewThisWeek int `json:"new_this_week"`
	Cities      int `json:"cities"`
	// WaitingNotNotified **طلباتُ «أضف منطقتي» السارية التي لم يُبلَّغ
	// أصحابُها** — **وهو عينُ رقم الصفحة الرئيسة** (`expansion_waiting`).
	WaitingNotNotified int `json:"waiting_not_notified"`
	// PendingNotify **التذكير** — من صار مغطّىً ولم يُبلَّغ.
	PendingNotify int            `json:"pending_notify"`
	ReadyPlaces   int            `json:"ready_places"`
	Rows          []expansionRow `json:"rows"`
}

// expRaw صفٌّ خامٌّ من القاعدة.
type expRaw struct {
	id, kind, addr               string
	uid                          *string
	requests                     int
	created, lastSeen            time.Time
	notified                     *time.Time
	lat, lng, cy, cx             float64
	cityID, govID, target        string
	zoneID, zoneName, zoneCityID string
}

// expZoneJoin **المنطقةُ التي تحكم النقطة** — بترتيب `ZoneAt` عينِه.
const expZoneJoin = `
	LEFT JOIN LATERAL (
		SELECT dz.id, dz.name, dz.city_id
		  FROM delivery_zones dz
		 WHERE dz.active AND (
		        (dz.shape = 'radius' AND ST_DWithin(dz.center, r.at, dz.radius_m))
		     OR (dz.shape = 'polygon' AND dz.area IS NOT NULL AND ST_Covers(dz.area, r.at)))
		 ORDER BY dz.sort_order, ST_Distance(dz.center, r.at), dz.id
		 LIMIT 1) z ON r.kind = 'coverage_request'`

func (s *Server) expansionRaw(ctx context.Context) ([]expRaw, error) {
	rows, err := s.pg.Query(ctx, `
		SELECT r.id::text, r.user_id::text, r.kind, r.requests, r.created_at,
		       r.last_seen_at, r.notified_at, r.address_text,
		       ST_Y(r.at::geometry), ST_X(r.at::geometry),
		       COALESCE(r.cell_y, 0), COALESCE(r.cell_x, 0),
		       COALESCE(r.city_id::text, ''), COALESCE(r.governorate_id::text, ''),
		       r.target_key,
		       COALESCE(z.id::text, ''), COALESCE(z.name, ''), COALESCE(z.city_id::text, '')
		  FROM coverage_requests r`+expZoneJoin+`
		 WHERE r.active
		 ORDER BY r.created_at
		 LIMIT 50000`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []expRaw
	for rows.Next() {
		var x expRaw
		if err := rows.Scan(&x.id, &x.uid, &x.kind, &x.requests, &x.created,
			&x.lastSeen, &x.notified, &x.addr, &x.lat, &x.lng, &x.cy, &x.cx,
			&x.cityID, &x.govID, &x.target, &x.zoneID, &x.zoneName, &x.zoneCityID); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

type expPlace struct{ name, govID, govName string }

func (s *Server) expansionPlaces(ctx context.Context) (map[string]expPlace, map[string]string, error) {
	cities := map[string]expPlace{}
	rows, err := s.pg.Query(ctx, `
		SELECT c.id::text, c.name, COALESCE(g.id::text, ''), COALESCE(g.name, '')
		  FROM cities c
		  LEFT JOIN districts d    ON d.id = c.district_id
		  LEFT JOIN governorates g ON g.id = d.governorate_id`)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var id string
		var p expPlace
		if err := rows.Scan(&id, &p.name, &p.govID, &p.govName); err != nil {
			rows.Close()
			return nil, nil, err
		}
		cities[id] = p
	}
	rows.Close()
	govs := map[string]string{}
	grows, err := s.pg.Query(ctx, `SELECT id::text, name FROM governorates`)
	if err != nil {
		return nil, nil, err
	}
	defer grows.Close()
	for grows.Next() {
		var id, name string
		if err := grows.Scan(&id, &name); err != nil {
			return nil, nil, err
		}
		govs[id] = name
	}
	return cities, govs, grows.Err()
}

// expKey هويّةُ الصفّ الذي ينتمي إليه طلب.
func expKey(x expRaw) string {
	switch {
	case x.kind == opsmap.KindCoverage && x.zoneID != "":
		return "zone:" + x.zoneID
	case x.kind == opsmap.KindInterest && strings.HasPrefix(x.target, "city:"):
		return x.target
	default:
		return fmt.Sprintf("cell:%s:%g,%g", x.kind, x.cy, x.cx)
	}
}

// expPending **أينتظر الإبلاغَ ويمكن إبلاغُه؟** — حسابٌ لم يُختَم.
func expPending(x expRaw) bool { return x.uid != nil && x.notified == nil }

// expansion **يبني الأرقامَ والجدول.**
func (s *Server) expansion(ctx context.Context) (expansionSummary, error) {
	raws, err := s.expansionRaw(ctx)
	if err != nil {
		return expansionSummary{}, err
	}
	cities, govs, err := s.expansionPlaces(ctx)
	if err != nil {
		return expansionSummary{}, err
	}

	weekAgo := time.Now().Add(-7 * 24 * time.Hour)
	people := map[string]bool{}
	newPeople := map[string]bool{}
	cityIDs := map[string]bool{}
	sum := expansionSummary{Rows: []expansionRow{}}

	type group struct {
		row     *expansionRow
		members []expRaw
		people  map[string]bool
		latest  time.Time
	}
	groups := map[string]*group{}
	var order []string

	for _, x := range raws {
		who := x.id
		if x.uid != nil {
			who = *x.uid
		}
		people[who] = true
		if x.created.After(weekAgo) {
			newPeople[who] = true
		}
		if x.kind == opsmap.KindCoverage && x.notified == nil {
			sum.WaitingNotNotified++
		}

		k := expKey(x)
		g := groups[k]
		if g == nil {
			cityID := x.cityID
			if x.zoneCityID != "" {
				cityID = x.zoneCityID
			}
			row := &expansionRow{Key: k, Kind: x.kind, CityID: cityID,
				ZoneID: x.zoneID, ZoneName: x.zoneName, FirstAt: x.created}
			if p, ok := cities[cityID]; ok {
				row.CityName, row.GovernorateID, row.GovernorateName = p.name, p.govID, p.govName
			}
			if row.GovernorateID == "" && x.govID != "" {
				row.GovernorateID, row.GovernorateName = x.govID, govs[x.govID]
			}
			g = &group{row: row, people: map[string]bool{}}
			groups[k] = g
			order = append(order, k)
		}
		r := g.row
		if r.CityID != "" {
			cityIDs[r.CityID] = true
		}
		g.members = append(g.members, x)
		g.people[who] = true
		r.Requests += x.requests
		if x.created.Before(r.FirstAt) {
			r.FirstAt = x.created
		}
		if x.lastSeen.After(r.LastAt) {
			r.LastAt = x.lastSeen
		}
		if x.created.After(r.LastAt) {
			r.LastAt = x.created
		}
		if a := strings.TrimSpace(x.addr); a != "" && !x.created.Before(g.latest) {
			r.Label, g.latest = a, x.created
		}
		if x.notified == nil {
			r.Waiting++
		} else {
			r.Notified++
		}
	}

	for _, k := range order {
		g := groups[k]
		r := g.row
		r.People = len(g.people)
		for _, m := range g.members {
			r.Lat += m.lat
			r.Lng += m.lng
		}
		r.Lat /= float64(len(g.members))
		r.Lng /= float64(len(g.members))

		covered, notifiable, err := s.expansionCoverage(ctx, k, g.members)
		if err != nil {
			return expansionSummary{}, err
		}
		r.Notifiable = notifiable
		switch {
		case !covered:
			r.Status = expUncovered
		case notifiable > 0:
			r.Status = expReady
			sum.ReadyPlaces++
			sum.PendingNotify += notifiable
		default:
			r.Status = expCovered
		}
		sum.Rows = append(sum.Rows, *r)
	}

	sum.People, sum.NewThisWeek, sum.Cities = len(people), len(newPeople), len(cityIDs)
	sort.SliceStable(sum.Rows, func(i, j int) bool {
		a, b := sum.Rows[i], sum.Rows[j]
		if a.GovernorateName != b.GovernorateName {
			return a.GovernorateName < b.GovernorateName
		}
		if a.CityName != b.CityName {
			return a.CityName < b.CityName
		}
		if a.People != b.People {
			return a.People > b.People
		}
		return a.Key < b.Key
	})
	return sum, nil
}

// expansionCoverage **أمُغطّى هذا الصفُّ؟ وكم يُبلَّغ لو ضُغط الزرّ؟**
//
// **وبالدالّتين اللتين يقرؤهما الإبلاغُ نفسُه** — فالعددُ المعروضُ قبل
// الضغط هو ما يُرسَل.
func (s *Server) expansionCoverage(ctx context.Context, key string, members []expRaw) (bool, int, error) {
	if s.orders == nil {
		return false, 0, nil
	}
	switch {
	case strings.HasPrefix(key, "zone:"):
		ok, err := s.orders.CoverableAt(ctx, s.pg, members[0].lat, members[0].lng)
		if err != nil || !ok {
			return false, 0, err
		}
		n := 0
		for _, m := range members {
			if !expPending(m) {
				continue
			}
			c, err := s.orders.CoverableAt(ctx, s.pg, m.lat, m.lng)
			if err != nil {
				return false, 0, err
			}
			if c {
				n++
			}
		}
		return true, n, nil
	case strings.HasPrefix(key, "city:"):
		ok, err := s.orders.CityEffectivelyLaunched(ctx, s.pg, strings.TrimPrefix(key, "city:"))
		if err != nil || !ok {
			return false, 0, err
		}
		n := 0
		for _, m := range members {
			if expPending(m) {
				n++
			}
		}
		return true, n, nil
	}
	return false, 0, nil
}

// notifyZoneWaiting **يُبلّغ منتظري منطقةٍ بعينها** — ويُرجع كم بُلّغ.
//
// **ومن تحكمه هذه المنطقةُ بترتيب `ZoneAt`** — عينُ صفّه في الجدول.
func (s *Server) notifyZoneWaiting(ctx context.Context, zoneID string) (int, error) {
	if s.notify == nil || s.orders == nil {
		return 0, nil
	}
	rows, err := s.pg.Query(ctx, `
		SELECT r.id::text, r.user_id::text, ST_Y(r.at::geometry), ST_X(r.at::geometry)
		  FROM coverage_requests r`+expZoneJoin+`
		 WHERE r.kind = 'coverage_request' AND r.active AND r.user_id IS NOT NULL
		   AND r.notified_at IS NULL AND z.id = $1::uuid`, zoneID)
	if err != nil {
		return 0, err
	}
	var list []pendingCoverage
	for rows.Next() {
		var p pendingCoverage
		if err := rows.Scan(&p.id, &p.uid, &p.lat, &p.lng); err != nil {
			rows.Close()
			return 0, err
		}
		list = append(list, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	return s.sendAreaCoverage(ctx, list), nil
}

// ErrNotCovered **المكانُ لم يُغطَّ بعد** — فلا يُبلَّغ أحد.
var ErrNotCovered = httpx.NewError(http.StatusConflict, "not_covered", "errors.not_covered")

// expansionNotify **«بلّغ المنتظرين الآن»** لصفٍّ بعينه.
func (s *Server) expansionNotify(ctx context.Context, key string) (int, error) {
	switch {
	case strings.HasPrefix(key, "zone:"):
		id := strings.TrimPrefix(key, "zone:")
		if !uuidLike(id) {
			return 0, ErrBadExpansionKey
		}
		return s.notifyZoneWaiting(ctx, id)
	case strings.HasPrefix(key, "city:"):
		id := strings.TrimPrefix(key, "city:")
		if !uuidLike(id) {
			return 0, ErrBadExpansionKey
		}
		launched, err := s.orders.CityEffectivelyLaunched(ctx, s.pg, id)
		if err != nil {
			return 0, err
		}
		if !launched {
			return 0, ErrNotCovered
		}
		return s.notifyCityLaunch(ctx, id), nil
	}
	return 0, ErrBadExpansionKey
}

// ErrBadExpansionKey **صفٌّ لا يُبلَّغ** — خليّةٌ خارجَ التغطية.
var ErrBadExpansionKey = httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")

func uuidLike(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
				return false
			}
		}
	}
	return true
}

// handleExpansion **قسمُ «طلبات التوسّع».**
func (s *Server) handleExpansion(w http.ResponseWriter, r *http.Request) {
	sum, err := s.expansion(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, sum)
}

// handleExpansionReminder **التذكير وحدَه** — تقرؤه الصفحةُ الرئيسة.
//
// **مغطّىً ولم يُبلَّغ**: `pending_notify` أشخاصٌ في `places` صفّاً.
func (s *Server) handleExpansionReminder(w http.ResponseWriter, r *http.Request) {
	sum, err := s.expansion(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"pending_notify":       sum.PendingNotify,
		"places":               sum.ReadyPlaces,
		"waiting_not_notified": sum.WaitingNotNotified,
	})
}

// handleExpansionNotify **«بلّغ المنتظرين الآن»** — يدويٌّ ويُقيَّد.
func (s *Server) handleExpansionNotify(w http.ResponseWriter, r *http.Request) {
	in, err := decode[struct {
		Key string `json:"key"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	n, err := s.expansionNotify(r.Context(), strings.TrimSpace(in.Key))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	entity, id, _ := strings.Cut(in.Key, ":")
	s.audit(r, "expansion.notify", entity, id, map[string]any{"notified": n})
	httpx.JSON(w, http.StatusOK, map[string]any{"notified": n})
}
