// Package testdb يهيّئ قاعدة بيانات حقيقية للاختبارات.
//
// الخدمات المالية منطقها في SQL: قيود CHECK ومعاملات ذرّية وقيود فريدة. اختبارها
// بمحاكاة (mock) يختبر المحاكاة لا النظام — فنستعمل قاعدة حقيقية بالهجرات نفسها.
//
// يتخطّى الاختبار بهدوء إن لم تُضبط TEST_DATABASE_URL، فلا يُفشل بناء من لا
// قاعدة لديه — وCI يضبطها فتعمل الاختبارات كاملة.
package testdb

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/migrate"
)

// Pool مجمّع اتصالات لقاعدة الاختبار، أو تخطٍّ إن لم تكن مهيّأة.
func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL غير مضبوط — تُتخطّى اختبارات القاعدة")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("تعذّر الاتصال بقاعدة الاختبار: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("قاعدة الاختبار لا تستجيب: %v", err)
	}
	if _, err := migrate.Up(ctx, pool); err != nil {
		t.Fatalf("فشل تطبيق الهجرات: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// NewUser ينشئ مستخدماً بهاتف فريد ويعيد معرّفه — لكل اختبار بياناته.
func NewUser(t *testing.T, pool *pgxpool.Pool, role string) string {
	t.Helper()
	ctx := context.Background()
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO users (phone, full_name)
		VALUES ('+9639' || lpad((floor(random() * 100000000))::text, 8, '0'), $1)
		RETURNING id`, "اختبار "+role).Scan(&id)
	if err != nil {
		t.Fatalf("تعذّر إنشاء مستخدم: %v", err)
	}
	if role != "" {
		if _, err := pool.Exec(ctx,
			`INSERT INTO user_roles (user_id, role_code) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			id, role); err != nil {
			t.Fatalf("تعذّر منح الدور: %v", err)
		}
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id)
	})
	return id
}
