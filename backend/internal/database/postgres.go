package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPostgres ينشئ مجمع اتصالات PostgreSQL ويتحقق من الاتصال.
func NewPostgres(ctx context.Context, url string) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse config: %w", err)
	}
	// **أربعون لا عشرون** (مراقبةُ الإنتاج ٢٠٢٦-١٠-١٩): تطبيقٌ يُفتح يطلق عشرةَ نداءاتٍ
	// معاً، فيمتلئ العشرون في لحظةٍ وتنتظر البقيّة. **والقاعدةُ تقبل مئة** (`max_connections`).
	poolCfg.MaxConns = 40
	poolCfg.MinConns = 2
	poolCfg.MaxConnLifetime = time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: connect: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}
	return pool, nil
}
