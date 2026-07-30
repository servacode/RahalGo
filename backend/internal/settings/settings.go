// Package settings مخزن الإعدادات الديناميكية المركزية — تُدار من لوحة الأدمن
// وتُقرأ لحظياً؛ لا تتطلب أي نشر جديد (GROUND-RULES §1.3).
package settings

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store { return &Store{db: db} }

func (s *Store) Get(ctx context.Context, key string, out any) error {
	var raw []byte
	err := s.db.QueryRow(ctx, `SELECT value FROM app_settings WHERE key = $1`, key).Scan(&raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

// GetString يقرأ قيمة نصية ويعيد الاحتياطي عند غيابها — لا يفشل أبداً.
func (s *Store) GetString(ctx context.Context, key, fallback string) string {
	var v string
	if err := s.Get(ctx, key, &v); err != nil || v == "" {
		return fallback
	}
	return v
}

func (s *Store) Set(ctx context.Context, key string, value any, updatedBy *string) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(ctx, `
		INSERT INTO app_settings (key, value, updated_by) VALUES ($1, $2, $3)
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value, updated_at = now(), updated_by = EXCLUDED.updated_by`,
		key, raw, updatedBy)
	return err
}

var ErrNotFound = errors.New("settings: not found")

func (s *Store) GetRaw(ctx context.Context, key string) (json.RawMessage, error) {
	var raw []byte
	err := s.db.QueryRow(ctx, `SELECT value FROM app_settings WHERE key = $1`, key).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return raw, err
}
