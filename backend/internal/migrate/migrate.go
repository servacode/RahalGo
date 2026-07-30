// Package migrate مشغّل هجرات بسيط وصارم:
// ملفات SQL مرقمة مضمّنة في الملف التنفيذي، تُطبَّق بالترتيب داخل معاملات،
// ويُسجَّل المطبَّق منها في جدول schema_migrations.
// قاعدة ملزمة (GROUND-RULES): ممنوع تعديل هجرة بعد دمجها — أضف هجرة جديدة.
package migrate

import (
	"context"
	"embed"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Up يطبّق كل الهجرات غير المطبقة بالترتيب ويعيد عددها.
func Up(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    text PRIMARY KEY,
			applied_at timestamptz NOT NULL DEFAULT now()
		)`); err != nil {
		return 0, fmt.Errorf("migrate: create schema_migrations: %w", err)
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return 0, fmt.Errorf("migrate: read dir: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)

	applied := map[string]bool{}
	rows, err := pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return 0, fmt.Errorf("migrate: read applied: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return 0, err
		}
		applied[v] = true
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	count := 0
	for _, name := range names {
		if applied[name] {
			continue
		}
		sqlBytes, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return count, fmt.Errorf("migrate: read %s: %w", name, err)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return count, fmt.Errorf("migrate: begin %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
			_ = tx.Rollback(ctx)
			return count, fmt.Errorf("migrate: apply %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
			_ = tx.Rollback(ctx)
			return count, fmt.Errorf("migrate: record %s: %w", name, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return count, fmt.Errorf("migrate: commit %s: %w", name, err)
		}
		count++
	}
	return count, nil
}
