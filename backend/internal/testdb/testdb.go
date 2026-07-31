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
	"net/url"
	"os"
	"strings"
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
	// **حارس**: الاختبارات تُنشئ مستخدمين ومتاجر وطلبات وقيوداً مالية. تشغيلها
	// على قاعدة التطوير يلوّثها ببيانات تبدو حقيقية — وقد حدث فعلاً: خمس عشرة
	// نسخة من متجر اختبار غرقت واجهة الزبون، وقيود عمولة لطلبات لم تقع.
	// وأخطر منه احتمال تشغيلها على قاعدة إنتاج.
	//
	// فلا تقبل هذه الحزمة إلا قاعدة يقول اسمها إنها للاختبار. الشرط في الاسم لا
	// في العنوان لأن الاسم هو ما يكتبه الإنسان ويخطئ فيه.
	if !isTestDatabase(url) {
		t.Fatalf("TEST_DATABASE_URL لا يشير إلى قاعدة اختبار (يجب أن ينتهي اسمها بـ _test): %s",
			redactURL(url))
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

// isTestDatabase يتحقق أن اسم القاعدة ينتهي بـ_test.
func isTestDatabase(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	name := strings.TrimPrefix(u.Path, "/")
	return strings.HasSuffix(name, "_test")
}

// redactURL يحجب كلمة المرور قبل طباعة العنوان في رسالة خطأ.
func redactURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "(عنوان غير صالح)"
	}
	if u.User != nil {
		u.User = url.User(u.User.Username())
	}
	return u.Redacted()
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
