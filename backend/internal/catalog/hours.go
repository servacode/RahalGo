package catalog

import (
	"context"
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// أوقات الدوام: 7 أيام دائماً (0=الأحد … 6=السبت) — تُعاد بقيم افتراضية إن لم تُضبط.

var ErrBadHours = httpx.NewError(http.StatusBadRequest, "invalid_hours", "errors.invalid_hours")

type DayHours struct {
	DayOfWeek int    `json:"day_of_week"`
	Closed    bool   `json:"closed"`
	OpenTime  string `json:"open_time"`  // "HH:MM"
	CloseTime string `json:"close_time"` // "HH:MM"
}

func (s *Service) GetHours(ctx context.Context, merchantID string) ([]DayHours, error) {
	if _, err := s.merchantByID(ctx, merchantID); err != nil {
		return nil, err
	}
	// أيام مضبوطة
	set := map[int]DayHours{}
	rows, err := s.db.Query(ctx, `
		SELECT day_of_week, closed, to_char(open_time,'HH24:MI'), to_char(close_time,'HH24:MI')
		FROM merchant_hours WHERE merchant_id = $1`, merchantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var d DayHours
		if err := rows.Scan(&d.DayOfWeek, &d.Closed, &d.OpenTime, &d.CloseTime); err != nil {
			return nil, err
		}
		set[d.DayOfWeek] = d
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]DayHours, 7)
	for i := 0; i < 7; i++ {
		if d, ok := set[i]; ok {
			out[i] = d
		} else {
			out[i] = DayHours{DayOfWeek: i, OpenTime: "09:00", CloseTime: "23:00"}
		}
	}
	return out, nil
}

func (s *Service) SetHours(ctx context.Context, actorID, merchantID string, days []DayHours, ip string) error {
	if len(days) != 7 {
		return ErrBadHours
	}
	if _, err := s.merchantByID(ctx, merchantID); err != nil {
		return err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, d := range days {
		if d.DayOfWeek < 0 || d.DayOfWeek > 6 {
			return ErrBadHours
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO merchant_hours (merchant_id, day_of_week, closed, open_time, close_time)
			VALUES ($1, $2, $3, $4::time, $5::time)
			ON CONFLICT (merchant_id, day_of_week) DO UPDATE
			SET closed = EXCLUDED.closed, open_time = EXCLUDED.open_time, close_time = EXCLUDED.close_time`,
			merchantID, d.DayOfWeek, d.Closed, d.OpenTime, d.CloseTime); err != nil {
			return ErrBadHours
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	s.audit(ctx, actorID, "merchant.hours_update", "merchant", merchantID, ip)
	return nil
}
