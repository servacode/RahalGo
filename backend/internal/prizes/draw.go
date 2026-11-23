package prizes

import (
	"context"
	"crypto/rand"
	"math/big"
	"strconv"
	"time"
)

// Entrant **مشتركٌ في سحب شهر** — بدعواته وفرصه.
type Entrant struct {
	UserID  string
	Invites int
	Entries int
}

// Winner **رابحٌ بمرتبته وجائزته.**
type Winner struct {
	Entrant
	Rank   int
	Amount int64
}

// PickWinners **سحبٌ مرجَّحٌ بالفرص بلا إعادة** — الشخصُ يربح مرّةً واحدةً بالسحب.
//
// `rnd(n)` يردّ عدداً في `[0, n)` — `crypto/rand` في التشغيل، ومثبَّتٌ في الاختبار.
func PickWinners(pool []Entrant, n int, rnd func(n int64) int64) []Entrant {
	left := append([]Entrant(nil), pool...)
	var out []Entrant
	for len(out) < n {
		var total int64
		for _, e := range left {
			if e.Entries > 0 {
				total += int64(e.Entries)
			}
		}
		if total == 0 {
			break
		}
		r := rnd(total)
		for i, e := range left {
			if e.Entries <= 0 {
				continue
			}
			if r < int64(e.Entries) {
				out = append(out, e)
				left = append(left[:i], left[i+1:]...)
				break
			}
			r -= int64(e.Entries)
		}
	}
	return out
}

// cryptoRand عددٌ في `[0, n)` من `crypto/rand`.
func cryptoRand(n int64) int64 {
	v, err := rand.Int(rand.Reader, big.NewInt(n))
	if err != nil {
		// **ولا سحبَ بعشوائيّةٍ معطوبة** — يُعاد في الدورة التالية.
		panic(err)
	}
	return v.Int64()
}

// Entrants **مشتركو شهرٍ** — من دعا فيه ما يكفي لفرصةٍ واحدةٍ على الأقلّ.
func (s *Service) Entrants(ctx context.Context, month time.Time) ([]Entrant, error) {
	_, launch := s.LaunchOn(ctx)
	from, to, ok := countWindow(month, launch)
	if !ok {
		return []Entrant{}, nil
	}
	per := s.PerEntry(ctx)
	rows, err := s.db.Query(ctx, `
		SELECT r.inviter_id::text, count(*)::int
		  FROM referrals r
		  JOIN users u ON u.id = r.inviter_id AND u.status = 'active'
		 WHERE r.created_at >= $1 AND r.created_at < $2
		 GROUP BY r.inviter_id
		HAVING count(*) >= $3
		 ORDER BY count(*) DESC, r.inviter_id`, from, to, per)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Entrant{}
	for rows.Next() {
		var e Entrant
		if err := rows.Scan(&e.UserID, &e.Invites); err != nil {
			return nil, err
		}
		e.Entries = EntriesFor(e.Invites, per)
		out = append(out, e)
	}
	return out, rows.Err()
}

// MonthInvites **دعواتُ الزبون في شهرٍ** — من يوم الإطلاق.
func (s *Service) MonthInvites(ctx context.Context, userID string, month time.Time) (int, error) {
	_, launch := s.LaunchOn(ctx)
	from, to, ok := countWindow(month, launch)
	if !ok {
		return 0, nil
	}
	var n int
	err := s.db.QueryRow(ctx, `
		SELECT count(*)::int FROM referrals
		 WHERE inviter_id = $1 AND created_at >= $2 AND created_at < $3`, userID, from, to).Scan(&n)
	return n, err
}

// DrawDue **سحبُ الشهر الماضي إن حان ولم يقع** — ويردّ الرابحين (فارغٌ إن لم يقع شيء).
//
// **صفٌّ واحدٌ لكلّ شهر** (`prize_draws.month` مفتاحٌ أوّليّ) يُدرَج أوّلاً في المعاملة:
// خادمان معاً ينتظر ثانيهما تثبيتَ الأوّل ثمّ يجد الصفَّ فيخرج. **وشهرٌ كلُّه قبل يوم
// الإطلاق لا يُسحب عليه.**
func (s *Service) DrawDue(ctx context.Context) ([]Winner, error) {
	if !s.Enabled(ctx) {
		return nil, nil
	}
	now := s.now()
	month := MonthStart(now).AddDate(0, -1, 0)
	if now.Before(DrawAt(month)) {
		return nil, nil
	}
	_, launch := s.LaunchOn(ctx)
	if _, _, ok := countWindow(month, launch); !ok {
		return nil, nil
	}
	return s.drawMonth(ctx, month, cryptoRand)
}

// drawMonth **يسحب على شهرٍ بعينه** — مرّةً واحدة.
func (s *Service) drawMonth(ctx context.Context, month time.Time, rnd func(int64) int64) ([]Winner, error) {
	entrants, err := s.Entrants(ctx, month)
	if err != nil {
		return nil, err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	total := 0
	for _, e := range entrants {
		total += e.Entries
	}
	key := month.Format("2006-01-02")
	tag, err := tx.Exec(ctx, `
		INSERT INTO prize_draws (month, drawn_at, entrants, total_entries)
		VALUES ($1::date, $2, $3, $4) ON CONFLICT (month) DO NOTHING`,
		key, s.now(), len(entrants), total)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, nil
	}
	amounts := s.Amounts(ctx)
	picked := PickWinners(entrants, len(amounts), rnd)
	winners := make([]Winner, 0, len(picked))
	for i, e := range picked {
		w := Winner{Entrant: e, Rank: i + 1, Amount: amounts[i]}
		if _, err := tx.Exec(ctx, `
			INSERT INTO prize_winners (month, rank, user_id, invites, entries, amount)
			VALUES ($1::date, $2, $3, $4, $5, $6)`,
			key, w.Rank, w.UserID, w.Invites, w.Entries, w.Amount); err != nil {
			return nil, err
		}
		if err := s.AwardTx(ctx, tx, w.UserID, w.Amount, month); err != nil {
			return nil, err
		}
		winners = append(winners, w)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	for _, w := range winners {
		if s.notify != nil && w.Amount > 0 {
			s.notify.NotifyWallet(ctx, w.UserID, textWonTitle,
				textWonBody1+formatMoney(w.Amount)+textWonBody2, prizesHref)
		}
	}
	s.logger.Info("prizes: draw", "month", MonthKey(month), "entrants", len(entrants),
		"entries", total, "winners", len(winners))
	return winners, nil
}

// RemindDue **«رصيد جوائزك بينتهي بعد ٣ أيام»** — مرّةً لكلّ انتهاء.
//
// **التحديثُ يكتب الختمَ ويردّ من خُتم في جملةٍ واحدة** — فخادمان لا يذكّران مرّتين.
func (s *Service) RemindDue(ctx context.Context) (int, error) {
	now := s.now()
	rows, err := s.db.Query(ctx, `
		UPDATE prize_balances SET reminded_for = expires_at
		 WHERE balance > 0 AND expires_at > $1
		   AND expires_at - interval '3 days' <= $1
		   AND reminded_for IS DISTINCT FROM expires_at
		RETURNING user_id::text, balance`, now)
	if err != nil {
		return 0, err
	}
	type hit struct {
		user    string
		balance int64
	}
	var hits []hit
	for rows.Next() {
		var h hit
		if err := rows.Scan(&h.user, &h.balance); err != nil {
			rows.Close()
			return 0, err
		}
		hits = append(hits, h)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for _, h := range hits {
		if s.notify != nil {
			s.notify.NotifyWallet(ctx, h.user, textRemindTitle,
				textRemindBody1+formatMoney(h.balance)+textRemindBody2, prizesHref)
		}
	}
	return len(hits), nil
}

// Sweep **جولةٌ واحدة**: الانتهاءُ أوّلاً ثمّ السحبُ ثمّ التذكير — وكلُّها لا تتكرّر.
func (s *Service) Sweep(ctx context.Context) {
	if _, err := s.ExpireDue(ctx); err != nil {
		s.logger.Warn("prizes: expire", "err", err)
	}
	if _, err := s.DrawDue(ctx); err != nil {
		s.logger.Warn("prizes: draw", "err", err)
	}
	if _, err := s.RemindDue(ctx); err != nil {
		s.logger.Warn("prizes: remind", "err", err)
	}
}

// Run **نبضةٌ دوريّة.**
func (s *Service) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Sweep(ctx)
		}
	}
}

// ── نصوصُ الإشعارات ─────────────────────────────────────────────────────

const (
	prizesHref      = "/portal/prizes"
	textWonTitle    = "مبروك ربحت بسحب جوائز رحّال غو"
	textWonBody1    = "انضاف "
	textWonBody2    = " لرصيد جوائزك — استعمله من متجر الجوائز قبل آخر الشهر"
	textRemindTitle = "رصيد جوائزك بينتهي بعد ٣ أيام"
	textRemindBody1 = "باقي "
	textRemindBody2 = " برصيد جوائزك — استعمله قبل آخر الشهر"
)

// formatMoney **مبلغٌ مجمَّعٌ بالفواصل** مع «ل.س».
func formatMoney(n int64) string {
	s := strconv.FormatInt(n, 10)
	neg := false
	if n < 0 {
		neg, s = true, s[1:]
	}
	var out []byte
	for i := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, s[i])
	}
	if neg {
		return "-" + string(out) + " ل.س"
	}
	return string(out) + " ل.س"
}
