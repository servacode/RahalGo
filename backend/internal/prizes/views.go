package prizes

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
	"github.com/servacode/rahalgo/backend/internal/pricing"
)

// StoreID **معرّفُ متجر الجوائز** — و`nil` إن لم يُعيَّن بعد.
func StoreID(ctx context.Context, q dbtx.Querier) (*string, error) {
	var id string
	err := q.QueryRow(ctx, `SELECT id::text FROM merchants WHERE status = 'prize' LIMIT 1`).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// AdminStore **متجرُ الجوائز كما تراه اللوحة** — اسمُه وعددُ أصنافه (قرارُ المالك ٢٠٢٦-١١-٢٤).
type AdminStore struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Items     int    `json:"items"`
	Available int    `json:"available"`
}

// StoreForAdmin **متجرُ الجوائز وأصنافُه** — و`nil` إن لم يُختر بعد.
func StoreForAdmin(ctx context.Context, q dbtx.Querier) (*AdminStore, error) {
	var st AdminStore
	err := q.QueryRow(ctx, `
		SELECT m.id::text, m.name,
		       (SELECT count(*) FROM menu_items i WHERE i.merchant_id = m.id),
		       (SELECT count(*) FROM menu_items i WHERE i.merchant_id = m.id AND i.available AND i.approved)
		  FROM merchants m WHERE m.status = 'prize' LIMIT 1`).Scan(&st.ID, &st.Name, &st.Items, &st.Available)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &st, nil
}

// ErrStoreTaken **في متجر جوائز قبله** — متجرٌ واحدٌ لا غير، فيُرجَع الأوّلُ قبل اختيار غيره.
var ErrStoreTaken = httpx.NewError(http.StatusConflict, "prize_store_taken", "errors.prize_store_taken")

// SetStore **يختار متجرَ الجوائز أو يرجّعه متجراً عاديّاً (مُطفأً)** — `merchantID` فارغ = إرجاع.
//
// **والمحظورُ لا يصير متجرَ جوائز** — الحظرُ بابُه `/suspend` وحدَه.
func SetStore(ctx context.Context, q dbtx.Querier, merchantID string) error {
	if merchantID == "" {
		_, err := q.Exec(ctx, `UPDATE merchants SET status = 'inactive', updated_at = now() WHERE status = 'prize'`)
		return err
	}
	cur, err := StoreID(ctx, q)
	if err != nil {
		return err
	}
	if cur != nil && *cur != merchantID {
		return ErrStoreTaken
	}
	tag, err := q.Exec(ctx, `
		UPDATE merchants SET status = 'prize', updated_at = now()
		 WHERE id = $1::uuid AND status IN ('active', 'inactive', 'prize')`, merchantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}
	return nil
}

// Me **حالُ الزبون في المسابقة** — `GET /prizes/me`.
//
// **نقاطُه وترتيبُه هو وحدَه** (قرارُ المالك ٢٠٢٦-١١-٢٥): لا اسمَ ولا نقاطَ لزبونٍ آخر في هذا
// الردّ ولا في غيره من أبواب الزبون — الترتيبُ الكاملُ باللوحة وحدَها.
type Me struct {
	Enabled  bool   `json:"enabled"`
	LaunchOn string `json:"launch_on"`
	// Started **بدأت المسابقة** — مفعّلةٌ وبلغنا يومَ الإطلاق؛ قبلها لا حضورَ ولا متابعة.
	Started bool   `json:"started"`
	Month   string `json:"month"`
	// CutoffAt **حسمُ هذه المسابقة** — بعده يُحسب كلُّ شيءٍ للشهر التالي.
	CutoffAt   time.Time `json:"cutoff_at"`
	AnnounceAt time.Time `json:"announce_at"`
	Points     int       `json:"points"`
	// Rank **ترتيبُه** — وصفرٌ إن لم تكن له نقطةٌ بعد أو كان مستثنًى.
	Rank      int  `json:"rank"`
	MinPoints int  `json:"min_points"`
	ToQualify int  `json:"to_qualify"`
	Qualified bool `json:"qualified"`
	// Excluded **ربح الشهر الماضي** — فهذا الشهرُ استراحة.
	Excluded          bool        `json:"excluded"`
	CheckedInToday    bool        `json:"checked_in_today"`
	FollowedFacebook  bool        `json:"followed_facebook"`
	FollowedInstagram bool        `json:"followed_instagram"`
	FacebookURL       string      `json:"facebook_url"`
	InstagramURL      string      `json:"instagram_url"`
	PointValues       PointValues `json:"point_values"`
	Amounts           []int64     `json:"amounts"`
	WinnersCount      int         `json:"winners_count"`
	// LastMonth/LastRank/LastAmount **نتيجتُه هو في آخر إعلان** — لا أسماءُ الرابحين.
	LastMonth      *string    `json:"last_month"`
	LastRank       int        `json:"last_rank"`
	LastAmount     int64      `json:"last_amount"`
	PrizeBalance   int64      `json:"prize_balance"`
	PrizeExpiresAt *time.Time `json:"prize_expires_at"`
	StoreID        *string    `json:"store_id"`
}

// MeFor يبني حالَ الزبون.
func (s *Service) MeFor(ctx context.Context, userID string) (*Me, error) {
	now := s.now()
	launchRaw, launch := s.LaunchOn(ctx)
	month := CurrentMonth(now, launch)
	minPts := s.MinPoints(ctx)
	standings, err := s.Standings(ctx, s.db, month)
	if err != nil {
		return nil, err
	}
	excluded, err := s.excludedFor(ctx, s.db, month)
	if err != nil {
		return nil, err
	}
	bal, err := s.BalanceOf(ctx, s.db, userID)
	if err != nil {
		return nil, err
	}
	store, err := StoreID(ctx, s.db)
	if err != nil {
		return nil, err
	}
	amounts := s.Amounts(ctx)
	out := &Me{
		Enabled:      s.Enabled(ctx),
		LaunchOn:     launchRaw,
		Started:      s.started(ctx),
		Month:        MonthKey(month),
		CutoffAt:     CutoffAt(month).UTC(),
		AnnounceAt:   AnnounceAt(month).UTC(),
		MinPoints:    minPts,
		Excluded:     excluded[userID],
		FacebookURL:  s.settings.GetString(ctx, KeyFacebookURL),
		InstagramURL: s.settings.GetString(ctx, KeyInstagramURL),
		PointValues:  s.Points(ctx),
		Amounts:      amounts,
		WinnersCount: len(amounts),
		PrizeBalance: bal.Amount,
		StoreID:      store,
	}
	for _, st := range standings {
		if st.UserID == userID {
			out.Points, out.Rank, out.Qualified = st.Points, st.Rank, st.Qualified
			break
		}
	}
	if !out.Excluded && out.Points < minPts {
		out.ToQualify = minPts - out.Points
	}
	if err := s.db.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM prize_checkins WHERE user_id = $1 AND day = $2::date),
		       EXISTS (SELECT 1 FROM prize_follows WHERE user_id = $1 AND network = 'facebook'),
		       EXISTS (SELECT 1 FROM prize_follows WHERE user_id = $1 AND network = 'instagram')`,
		userID, DamascusDay(now)).Scan(&out.CheckedInToday, &out.FollowedFacebook, &out.FollowedInstagram); err != nil {
		return nil, err
	}
	var last time.Time
	err = s.db.QueryRow(ctx, `SELECT month FROM prize_draws ORDER BY month DESC LIMIT 1`).Scan(&last)
	if err == nil {
		k := last.Format("2006-01")
		out.LastMonth = &k
		err = s.db.QueryRow(ctx, `SELECT rank, amount FROM prize_winners WHERE month = $1 AND user_id = $2`,
			last, userID).Scan(&out.LastRank, &out.LastAmount)
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if bal.Amount > 0 && bal.ExpiresAt != nil {
		t := bal.ExpiresAt.UTC()
		out.PrizeExpiresAt = &t
	}
	return out, nil
}

// Item **صنفٌ من متجر الجوائز.**
type Item struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       int64   `json:"price"`
	ImageURL    *string `json:"image_url"`
	ThumbURL    *string `json:"thumb_url"`
	Available   bool    `json:"available"`
}

// Items **أصنافُ متجر الجوائز** بسعر البيع نفسِه الذي يُحسب عند الطلب — مرتّبةً بالسعر.
func (s *Service) Items(ctx context.Context) (*string, []Item, error) {
	store, err := StoreID(ctx, s.db)
	if err != nil || store == nil {
		return store, []Item{}, err
	}
	rule := pricing.RuleFrom(ctx, s.settings)
	rows, err := s.db.Query(ctx, `
		SELECT i.id::text, i.name, COALESCE(i.description, ''), i.merchant_price,
		       i.margin_override, ps.margin_override, im.path, im.thumb_path,
		       i.available AND i.approved
		  FROM menu_items i
		  LEFT JOIN platform_sections ps ON ps.id = i.platform_section_id
		  LEFT JOIN media im ON im.id = i.image_media_id
		 WHERE i.merchant_id = $1::uuid
		 LIMIT 500`, *store)
	if err != nil {
		return store, nil, err
	}
	defer rows.Close()
	out := []Item{}
	for rows.Next() {
		var it Item
		var cost int64
		var im, sm *int64
		if err := rows.Scan(&it.ID, &it.Name, &it.Description, &cost, &im, &sm,
			&it.ImageURL, &it.ThumbURL, &it.Available); err != nil {
			return store, nil, err
		}
		it.Price = rule.SalePrice(cost, im, sm)
		it.ImageURL = media.URLForPtr(it.ImageURL)
		it.ThumbURL = media.URLForPtr(it.ThumbURL)
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return store, nil, err
	}
	sortItems(out)
	return store, out, nil
}

func sortItems(items []Item) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && (items[j].Price < items[j-1].Price ||
			(items[j].Price == items[j-1].Price && items[j].Name < items[j-1].Name)); j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}

// AdminStanding **مشتركٌ كما تراه اللوحة** — باسمه ورقمه ونقاطه ومصادرها.
type AdminStanding struct {
	Rank      int            `json:"rank"`
	UserID    string         `json:"user_id"`
	Name      string         `json:"name"`
	Phone     string         `json:"phone"`
	Points    int            `json:"points"`
	ReachedAt time.Time      `json:"reached_at"`
	Excluded  bool           `json:"excluded"`
	Qualified bool           `json:"qualified"`
	Breakdown map[string]int `json:"breakdown"`
}

// AdminLeaderboard **ترتيبُ شهرٍ كاملاً بالأسماء الحقيقيّة** — للموظّفين وحدَهم.
func (s *Service) AdminLeaderboard(ctx context.Context, month time.Time) ([]AdminStanding, error) {
	list, err := s.Standings(ctx, s.db, month)
	if err != nil {
		return nil, err
	}
	out := make([]AdminStanding, 0, len(list))
	for _, st := range list {
		a := AdminStanding{
			Rank: st.Rank, UserID: st.UserID, Points: st.Points, ReachedAt: st.ReachedAt,
			Excluded: st.Excluded, Qualified: st.Qualified, Breakdown: st.Breakdown,
		}
		if err := s.db.QueryRow(ctx, `
			SELECT COALESCE(full_name, ''), COALESCE(phone::text, '') FROM users WHERE id = $1`,
			st.UserID).Scan(&a.Name, &a.Phone); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

// AdminWinner **رابحٌ بما صرف وما انتهى.**
type AdminWinner struct {
	Rank    int    `json:"rank"`
	UserID  string `json:"user_id"`
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Points  int    `json:"points"`
	Amount  int64  `json:"amount"`
	Spent   int64  `json:"spent"`
	Expired int64  `json:"expired"`
}

// AdminDraw **إعلانٌ وقع** — وكلفتُه: ما مُنح وما صُرف فعلاً (الكلفة) وما انتهى.
type AdminDraw struct {
	Month     string        `json:"month"`
	DrawnAt   time.Time     `json:"drawn_at"`
	CutoffAt  *time.Time    `json:"cutoff_at"`
	MinPoints int           `json:"min_points"`
	Entrants  int           `json:"entrants"`
	Qualified int           `json:"qualified"`
	Awarded   int64         `json:"awarded"`
	Spent     int64         `json:"spent"`
	Expired   int64         `json:"expired"`
	Winners   []AdminWinner `json:"winners"`
}

// AdminDraws **النتائجُ السابقة** — الأحدثُ أوّلاً.
//
// **والصرفُ والانتهاءُ يُنسبان إلى رصيدٍ مُنح في شهر السحب** — بانتهائه (`expires_at`)،
// فرصيدُ كلّ سحبٍ ينتهي في شهرٍ غيرِ شهر غيره.
func (s *Service) AdminDraws(ctx context.Context) ([]AdminDraw, error) {
	rows, err := s.db.Query(ctx, `
		SELECT d.month, d.drawn_at, d.cutoff_at, d.min_points, d.entrants, d.qualified
		  FROM prize_draws d ORDER BY d.month DESC LIMIT 36`)
	if err != nil {
		return nil, err
	}
	var draws []AdminDraw
	var months []time.Time
	for rows.Next() {
		var d AdminDraw
		var m time.Time
		if err := rows.Scan(&m, &d.DrawnAt, &d.CutoffAt, &d.MinPoints, &d.Entrants, &d.Qualified); err != nil {
			rows.Close()
			return nil, err
		}
		d.Month = m.Format("2006-01")
		d.Winners = []AdminWinner{}
		draws = append(draws, d)
		months = append(months, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range draws {
		wr, err := s.db.Query(ctx, `
			SELECT w.rank, w.user_id::text, COALESCE(u.full_name, ''), COALESCE(u.phone::text, ''),
			       w.points, w.amount,
			       COALESCE((SELECT -sum(e.amount) FROM prize_entries e
			                  WHERE e.user_id = w.user_id AND e.kind IN ('spend', 'refund')
			                    AND e.expires_at = a.expires_at), 0),
			       COALESCE((SELECT -sum(e.amount) FROM prize_entries e
			                  WHERE e.user_id = w.user_id AND e.kind = 'expire'
			                    AND e.expires_at = a.expires_at), 0)
			  FROM prize_winners w
			  JOIN users u ON u.id = w.user_id
			  LEFT JOIN LATERAL (SELECT expires_at FROM prize_entries
			                      WHERE user_id = w.user_id AND kind = 'award' AND draw_month = w.month
			                      LIMIT 1) a ON true
			 WHERE w.month = $1 ORDER BY w.rank`, months[i])
		if err != nil {
			return nil, err
		}
		for wr.Next() {
			var w AdminWinner
			if err := wr.Scan(&w.Rank, &w.UserID, &w.Name, &w.Phone, &w.Points, &w.Amount, &w.Spent, &w.Expired); err != nil {
				wr.Close()
				return nil, err
			}
			draws[i].Winners = append(draws[i].Winners, w)
			draws[i].Awarded += w.Amount
			draws[i].Spent += w.Spent
			draws[i].Expired += w.Expired
		}
		wr.Close()
		if err := wr.Err(); err != nil {
			return nil, err
		}
	}
	if draws == nil {
		draws = []AdminDraw{}
	}
	return draws, nil
}
