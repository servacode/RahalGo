package prizes

// ══════════════════════════════════════════════════════════════════════
// **نقاطُ المسابقة وترتيبُها** (قرارُ المالك ٢٠٢٦-١١-٢٥)
// ══════════════════════════════════════════════════════════════════════
//
// **كلُّ نقطةٍ حدثٌ له لحظة** — حضورٌ، متابعةٌ، طلبٌ مُسلَّم، دعوةٌ، أوّلُ طلبٍ لمدعوّ — **وتُجمع
// من مصادرها في كلّ قراءة** داخلَ نافذة الشهر (`countWindow`). فلا عدّادَ يُزاد ولا يُنقص:
// **إلغاءُ التسليم يمحو `delivered_at` فتسقط نقاطُ الطلب وحدَها**، وإن كان أوّلَ طلبٍ لمدعوٍّ
// صار الطلبُ المُسلَّمُ التالي أوّلَه.
//
// **ولحظةُ بلوغ المجموع = آخرُ حدثٍ عُدّ** (كلُّ النقاط موجبة) — وبها يُكسر التعادل.

import (
	"context"
	"net/http"
	"sort"
	"time"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/offers"
)

// Standing **مشتركٌ بمجموعه وترتيبه.**
type Standing struct {
	UserID    string
	Points    int
	ReachedAt time.Time
	// Breakdown **من أين جاءت النقاط** — للوحة وحدَها.
	Breakdown map[string]int
	// Rank **ترتيبُه بين غير المستثنَين** — وصفرٌ للمستثنى (رابحُ الشهر الماضي).
	Rank      int
	Excluded  bool
	Qualified bool
}

// مصادرُ النقاط كما تُكتب في التفصيل.
const (
	SrcCheckin          = "checkin"
	SrcOrder            = "order"
	SrcInviteSignup     = "invite_signup"
	SrcInviteFirstOrder = "invite_first_order"
	SrcFollow           = "follow"
)

// RankStandings **يرتّب ويُعلِّم** — الأكثرُ نقاطاً أوّلاً، **والتعادلُ لمن بلغه أبكر**، ثمّ
// بالمعرّف (ثباتٌ لا عدل). **والمستثنى بلا رتبة** ولا يتأهّل، ومن بلغ `minPoints` متأهّل.
func RankStandings(in []Standing, excluded map[string]bool, minPoints int) []Standing {
	out := append([]Standing(nil), in...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Points != b.Points {
			return a.Points > b.Points
		}
		if !a.ReachedAt.Equal(b.ReachedAt) {
			return a.ReachedAt.Before(b.ReachedAt)
		}
		return a.UserID < b.UserID
	})
	rank := 0
	for i := range out {
		out[i].Excluded = excluded[out[i].UserID]
		out[i].Rank = 0
		out[i].Qualified = false
		if out[i].Excluded {
			continue
		}
		rank++
		out[i].Rank = rank
		out[i].Qualified = out[i].Points >= minPoints && out[i].Points > 0
	}
	return out
}

// PickWinners **أوّلُ `n` متأهّلين بالترتيب** — والمستثنى ومن دون الحدّ لا يفوزان.
func PickWinners(ranked []Standing, n int) []Standing {
	var out []Standing
	for _, s := range ranked {
		if len(out) >= n {
			break
		}
		if s.Rank > 0 && s.Qualified {
			out = append(out, s)
		}
	}
	return out
}

// standingsSQL **أحداثُ النقاط في النافذة** ثمّ مجموعُ كلّ زبونٍ ولحظةُ بلوغه.
//
// الطلبُ: مُسلَّمٌ للزبون نفسِه، عاديٌّ أو خاصّ (والأمانةُ خاصّ)، **لا طلبُ متجرٍ ولا طلبُ
// جوائز**. وأوّلُ طلبٍ لمدعوٍّ: **أوّلُ طلبٍ مُسلَّمٍ له في عمره**، يُحسب للداعي إن وقع في النافذة.
const standingsSQL = `
WITH ev AS (
	SELECT c.user_id, 'checkin'::text AS src, $3::int AS pts, c.created_at AS at
	  FROM prize_checkins c
	 WHERE c.created_at >= $1 AND c.created_at < $2
	UNION ALL
	SELECT f.user_id, 'follow', CASE f.network WHEN 'facebook' THEN $4::int ELSE $5::int END, f.created_at
	  FROM prize_follows f
	 WHERE f.created_at >= $1 AND f.created_at < $2
	UNION ALL
	SELECT o.customer_id, 'order', $6::int, o.delivered_at
	  FROM orders o
	 WHERE o.status = 'delivered' AND o.kind IN ('standard', 'custom') AND NOT o.prize_order
	   AND o.delivered_at >= $1 AND o.delivered_at < $2
	UNION ALL
	SELECT r.inviter_id, 'invite_signup', $7::int, r.created_at
	  FROM referrals r
	 WHERE r.created_at >= $1 AND r.created_at < $2
	UNION ALL
	SELECT r.inviter_id, 'invite_first_order', $8::int, fo.first_at
	  FROM referrals r
	  JOIN LATERAL (
		SELECT min(o.delivered_at) AS first_at
		  FROM orders o
		 WHERE o.customer_id = r.invitee_id AND o.status = 'delivered'
		   AND o.kind IN ('standard', 'custom') AND NOT o.prize_order
	  ) fo ON true
	 WHERE fo.first_at >= $1 AND fo.first_at < $2
)
SELECT ev.user_id::text, sum(ev.pts)::int, max(ev.at),
       COALESCE(sum(ev.pts) FILTER (WHERE ev.src = 'checkin'), 0)::int,
       COALESCE(sum(ev.pts) FILTER (WHERE ev.src = 'order'), 0)::int,
       COALESCE(sum(ev.pts) FILTER (WHERE ev.src = 'invite_signup'), 0)::int,
       COALESCE(sum(ev.pts) FILTER (WHERE ev.src = 'invite_first_order'), 0)::int,
       COALESCE(sum(ev.pts) FILTER (WHERE ev.src = 'follow'), 0)::int
  FROM ev
  JOIN users u ON u.id = ev.user_id AND u.status = 'active'
 WHERE ev.pts > 0
 GROUP BY ev.user_id`

// Standings **ترتيبُ شهرٍ كاملاً** — كلُّ من له نقطةٌ فأكثر، مرتّباً ومُعلَّماً.
func (s *Service) Standings(ctx context.Context, q dbtx.Querier, month time.Time) ([]Standing, error) {
	_, launch := s.LaunchOn(ctx)
	from, to, ok := countWindow(month, launch)
	if !ok {
		return []Standing{}, nil
	}
	pv := s.Points(ctx)
	rows, err := q.Query(ctx, standingsSQL, from, to,
		pv.Checkin, pv.FollowFacebook, pv.FollowInstagram, pv.Order, pv.InviteSignup, pv.InviteFirstOrder)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []Standing{}
	for rows.Next() {
		var st Standing
		var c, o, is, ifo, f int
		if err := rows.Scan(&st.UserID, &st.Points, &st.ReachedAt, &c, &o, &is, &ifo, &f); err != nil {
			return nil, err
		}
		st.Breakdown = map[string]int{
			SrcCheckin: c, SrcOrder: o, SrcInviteSignup: is, SrcInviteFirstOrder: ifo, SrcFollow: f,
		}
		list = append(list, st)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	excluded, err := s.excludedFor(ctx, q, month)
	if err != nil {
		return nil, err
	}
	return RankStandings(list, excluded, s.MinPoints(ctx)), nil
}

// excludedFor **رابحو الشهر الماضي** — خارجَ ترتيب هذا الشهر وحدَه.
func (s *Service) excludedFor(ctx context.Context, q dbtx.Querier, month time.Time) (map[string]bool, error) {
	prev := MonthStart(month).AddDate(0, -1, 0).Format("2006-01-02")
	rows, err := q.Query(ctx, `SELECT user_id::text FROM prize_winners WHERE month = $1::date`, prev)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

// ── أفعالُ الزبون ────────────────────────────────────────────────────────

// ErrNotStarted **المسابقةُ لم تبدأ** — مطفأةٌ أو قبل يوم الإطلاق. فالحضورُ والمتابعةُ قبلها
// لا تُكتب (المتابعةُ مرّةً في العمر، **فلو كُتبت قبل الإطلاق لضاعت نقاطُها للأبد**).
var ErrNotStarted = httpx.NewError(http.StatusConflict, "prizes_not_started", "errors.prizes_not_started")

// ErrBadNetwork **شبكةٌ غيرُ فيسبوك وإنستغرام.**
var ErrBadNetwork = httpx.NewError(http.StatusBadRequest, "prizes_bad_network", "errors.validation")

// Networks **الصفحتان المتابَعتان.**
var Networks = []string{"facebook", "instagram"}

func (s *Service) started(ctx context.Context) bool {
	if !s.Enabled(ctx) {
		return false
	}
	_, launch := s.LaunchOn(ctx)
	return !s.now().Before(launch)
}

// DamascusDay **يومُ `t` بتوقيت دمشق** `YYYY-MM-DD` — وحدةُ «مرّةً باليوم».
func DamascusDay(t time.Time) string { return t.In(offers.Damascus).Format("2006-01-02") }

// CheckIn **«سجّل حضوري»** — مرّةً باليوم؛ والضغطةُ الثانيةُ في اليوم نفسِه لا تضيف شيئاً.
// يردّ `true` إن كُتب حضورٌ جديد.
func (s *Service) CheckIn(ctx context.Context, userID string) (bool, error) {
	if !s.started(ctx) {
		return false, ErrNotStarted
	}
	now := s.now()
	tag, err := s.db.Exec(ctx, `
		INSERT INTO prize_checkins (user_id, day, created_at) VALUES ($1, $2::date, $3)
		ON CONFLICT (user_id, day) DO NOTHING`, userID, DamascusDay(now), now)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// Follow **«تابعنا على فيسبوك/إنستغرام»** — بالأمانة بلا تحقّق: الزرُّ يفتح الصفحةَ ويُعلِّم.
// **مرّةً في عمر الحساب** وتُحسب في شهر الضغط. يردّ `true` إن كُتبت متابعةٌ جديدة.
func (s *Service) Follow(ctx context.Context, userID, network string) (bool, error) {
	valid := false
	for _, n := range Networks {
		if n == network {
			valid = true
		}
	}
	if !valid {
		return false, ErrBadNetwork
	}
	if !s.started(ctx) {
		return false, ErrNotStarted
	}
	tag, err := s.db.Exec(ctx, `
		INSERT INTO prize_follows (user_id, network, created_at) VALUES ($1, $2, $3)
		ON CONFLICT (user_id, network) DO NOTHING`, userID, network, s.now())
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}
