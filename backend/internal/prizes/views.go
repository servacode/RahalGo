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

// Me **حالُ الزبون في الجوائز** — `GET /prizes/me`.
type Me struct {
	Enabled         bool       `json:"enabled"`
	LaunchOn        string     `json:"launch_on"`
	Month           string     `json:"month"`
	MonthInvites    int        `json:"month_invites"`
	InvitesPerEntry int        `json:"invites_per_entry"`
	Entries         int        `json:"entries"`
	InvitesToNext   int        `json:"invites_to_next"`
	DrawAt          time.Time  `json:"draw_at"`
	Amounts         []int64    `json:"amounts"`
	WinnersCount    int        `json:"winners_count"`
	PrizeBalance    int64      `json:"prize_balance"`
	PrizeExpiresAt  *time.Time `json:"prize_expires_at"`
	StoreID         *string    `json:"store_id"`
}

// MeFor يبني حالَ الزبون.
func (s *Service) MeFor(ctx context.Context, userID string) (*Me, error) {
	now := s.now()
	month := MonthStart(now)
	launchRaw, launch := s.LaunchOn(ctx)
	per := s.PerEntry(ctx)
	invites, err := s.MonthInvites(ctx, userID, month)
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
		Enabled:         s.Enabled(ctx),
		LaunchOn:        launchRaw,
		Month:           MonthKey(month),
		MonthInvites:    invites,
		InvitesPerEntry: per,
		Entries:         EntriesFor(invites, per),
		InvitesToNext:   ToNext(invites, per),
		DrawAt:          NextDrawAt(now, launch).UTC(),
		Amounts:         amounts,
		WinnersCount:    len(amounts),
		PrizeBalance:    bal.Amount,
		StoreID:         store,
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

// PublicWinner **رابحٌ كما يُعرض للزبائن** — بلا رقمٍ أبداً.
type PublicWinner struct {
	Rank   int    `json:"rank"`
	Name   string `json:"name"`
	Amount int64  `json:"amount"`
}

// LastWinners **رابحو آخر سحبٍ وقع.**
func (s *Service) LastWinners(ctx context.Context) (*string, []PublicWinner, error) {
	var month time.Time
	err := s.db.QueryRow(ctx, `SELECT month FROM prize_draws ORDER BY month DESC LIMIT 1`).Scan(&month)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, []PublicWinner{}, nil
	}
	if err != nil {
		return nil, nil, err
	}
	key := month.Format("2006-01")
	rows, err := s.db.Query(ctx, `
		SELECT w.rank, COALESCE(u.full_name, ''), w.amount
		  FROM prize_winners w JOIN users u ON u.id = w.user_id
		 WHERE w.month = $1 ORDER BY w.rank`, month)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	out := []PublicWinner{}
	for rows.Next() {
		var w PublicWinner
		var full string
		if err := rows.Scan(&w.Rank, &full, &w.Amount); err != nil {
			return nil, nil, err
		}
		w.Name = DisplayName(full)
		out = append(out, w)
	}
	return &key, out, rows.Err()
}

// AdminEntrant **مشتركٌ كما تراه اللوحة.**
type AdminEntrant struct {
	UserID  string `json:"user_id"`
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Invites int    `json:"invites"`
	Entries int    `json:"entries"`
}

// AdminEntries **مشتركو شهرٍ** — بأسمائهم وأرقامهم، الأكثرُ فرصاً أوّلاً.
func (s *Service) AdminEntries(ctx context.Context, month time.Time) ([]AdminEntrant, error) {
	es, err := s.Entrants(ctx, month)
	if err != nil {
		return nil, err
	}
	out := make([]AdminEntrant, 0, len(es))
	for _, e := range es {
		a := AdminEntrant{UserID: e.UserID, Invites: e.Invites, Entries: e.Entries}
		if err := s.db.QueryRow(ctx, `
			SELECT COALESCE(full_name, ''), COALESCE(phone::text, '') FROM users WHERE id = $1`,
			e.UserID).Scan(&a.Name, &a.Phone); err != nil {
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
	Amount  int64  `json:"amount"`
	Spent   int64  `json:"spent"`
	Expired int64  `json:"expired"`
}

// AdminDraw **سحبٌ وقع** — وكلفتُه: ما مُنح وما صُرف فعلاً (الكلفة) وما انتهى.
type AdminDraw struct {
	Month        string        `json:"month"`
	DrawnAt      time.Time     `json:"drawn_at"`
	Entrants     int           `json:"entrants"`
	TotalEntries int           `json:"total_entries"`
	Awarded      int64         `json:"awarded"`
	Spent        int64         `json:"spent"`
	Expired      int64         `json:"expired"`
	Winners      []AdminWinner `json:"winners"`
}

// AdminDraws **السحوباتُ السابقة** — الأحدثُ أوّلاً.
//
// **والصرفُ والانتهاءُ يُنسبان إلى رصيدٍ مُنح في شهر السحب** — بانتهائه (`expires_at`)،
// فرصيدُ كلّ سحبٍ ينتهي في شهرٍ غيرِ شهر غيره.
func (s *Service) AdminDraws(ctx context.Context) ([]AdminDraw, error) {
	rows, err := s.db.Query(ctx, `
		SELECT d.month, d.drawn_at, d.entrants, d.total_entries
		  FROM prize_draws d ORDER BY d.month DESC LIMIT 36`)
	if err != nil {
		return nil, err
	}
	var draws []AdminDraw
	var months []time.Time
	for rows.Next() {
		var d AdminDraw
		var m time.Time
		if err := rows.Scan(&m, &d.DrawnAt, &d.Entrants, &d.TotalEntries); err != nil {
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
			       w.amount,
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
			if err := wr.Scan(&w.Rank, &w.UserID, &w.Name, &w.Phone, &w.Amount, &w.Spent, &w.Expired); err != nil {
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
