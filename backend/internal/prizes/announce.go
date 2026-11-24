package prizes

import (
	"context"
	"strconv"
	"time"
)

// Winner **رابحٌ بمرتبته وجائزته.**
type Winner struct {
	Standing
	Amount int64
}

// AnnounceDue **إعلانُ آخر شهرٍ حان إعلانُه ولم يُعلن** — ويردّ الرابحين (فارغٌ إن لم يقع شيء).
//
// الشهرُ الجاري إن بلغنا يومَ إعلانه (آخرَ يومٍ ١٢:٠٠)، وإلّا الشهرُ الماضي (إعلانٌ فاته خادمٌ
// متوقّف). **صفٌّ واحدٌ لكلّ شهر** (`prize_draws.month` مفتاحٌ أوّليّ) يُدرَج أوّلاً في المعاملة:
// خادمان معاً ينتظر ثانيهما تثبيتَ الأوّل ثمّ يجد الصفَّ فيخرج. **وشهرٌ نافذتُه كلُّها قبل يوم
// الإطلاق لا يُعلن عنه.**
func (s *Service) AnnounceDue(ctx context.Context) ([]Winner, error) {
	if !s.Enabled(ctx) {
		return nil, nil
	}
	now := s.now()
	month := MonthStart(now)
	if now.Before(AnnounceAt(month)) {
		month = month.AddDate(0, -1, 0)
	}
	_, launch := s.LaunchOn(ctx)
	if _, _, ok := countWindow(month, launch); !ok {
		return nil, nil
	}
	return s.announceMonth(ctx, month)
}

// announceMonth **يعلن شهراً بعينه** — مرّةً واحدة.
func (s *Service) announceMonth(ctx context.Context, month time.Time) ([]Winner, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	key := MonthStart(month).Format("2006-01-02")
	minPts := s.MinPoints(ctx)
	tag, err := tx.Exec(ctx, `
		INSERT INTO prize_draws (month, drawn_at, cutoff_at, min_points)
		VALUES ($1::date, $2, $3, $4) ON CONFLICT (month) DO NOTHING`,
		key, s.now(), CutoffAt(month), minPts)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, nil
	}
	// **والترتيبُ يُقرأ بعد حجز الشهر** — فما يُكتب هو ما رآه الإعلانُ نفسُه.
	standings, err := s.Standings(ctx, tx, month)
	if err != nil {
		return nil, err
	}
	qualified := 0
	for _, st := range standings {
		if st.Qualified {
			qualified++
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE prize_draws SET entrants = $2, qualified = $3 WHERE month = $1::date`,
		key, len(standings), qualified); err != nil {
		return nil, err
	}
	amounts := s.Amounts(ctx)
	picked := PickWinners(standings, len(amounts))
	winners := make([]Winner, 0, len(picked))
	for i, st := range picked {
		w := Winner{Standing: st, Amount: amounts[i]}
		// **المرتبةُ مرتبةُ الجائزة** (١..ن) — والمستثنى قبله لا يأخذ مكاناً.
		w.Rank = i + 1
		if _, err := tx.Exec(ctx, `
			INSERT INTO prize_winners (month, rank, user_id, points, reached_at, amount)
			VALUES ($1::date, $2, $3, $4, $5, $6)`,
			key, w.Rank, w.UserID, w.Points, w.ReachedAt, w.Amount); err != nil {
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
	s.logger.Info("prizes: announce", "month", MonthKey(month), "entrants", len(standings),
		"qualified", qualified, "winners", len(winners))
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

// Sweep **جولةٌ واحدة**: الانتهاءُ أوّلاً ثمّ الإعلانُ ثمّ التذكير — وكلُّها لا تتكرّر.
func (s *Service) Sweep(ctx context.Context) {
	if _, err := s.ExpireDue(ctx); err != nil {
		s.logger.Warn("prizes: expire", "err", err)
	}
	if _, err := s.AnnounceDue(ctx); err != nil {
		s.logger.Warn("prizes: announce", "err", err)
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
	textWonTitle    = "مبروك ربحت بمسابقة جوائز رحّال غو"
	textWonBody1    = "انضاف "
	textWonBody2    = " لرصيد جوائزك — استعمله من متجر الجوائز قبل آخر الشهر الجاي"
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
