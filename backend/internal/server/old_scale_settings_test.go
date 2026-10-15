package server

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// pinUnset **يضبط مفتاحاً لمن لم يضبطه — ويمحوه بعد الاختبار.**
//
// # لماذا
//
// **صارت افتراضاتُ المال بالليرة الجديدة** (٢٠٢٦-١٠-١٥): سقفُ المحفظة ٥٠٠٠، وعتبةُ
// المصروف ٥٠٠٠، وسقفُ التعويض ٥٠٠، وحدّا أجرة الخاصّ ١٠ و١٠٠٠. **واختباراتٌ كثيرةٌ تكتب
// مبالغَها بالقديمة** (٧٠٠٠ و١٠٬٠٠٠…) وتقيس مساراً لا سقفاً — **فتُردّ بالسقف قبل أن
// تبلغ ما تقيسه.** فيُوسَّع السقفُ لها صراحةً.
//
// **ومن ضبط المفتاحَ بنفسه لا يُمسّ** — واختبارُ السقف نفسِه لا يناديها فيقيس الافتراض.
// **والمخزنُ يُبطَل** — فلقطتُه قد تكون أُخذت قبل الضبط.
func pinUnset(t *testing.T, pool *pgxpool.Pool, srv *Server, key, value string) {
	t.Helper()
	ctx := context.Background()
	tag, err := pool.Exec(ctx, `INSERT INTO app_settings (key, value) VALUES ($1, $2::jsonb)
		ON CONFLICT (key) DO NOTHING`, key, value)
	if err != nil {
		t.Fatalf("تعذّر ضبطُ %s: %v", key, err)
	}
	if tag.RowsAffected() == 1 {
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), `DELETE FROM app_settings WHERE key = $1`, key)
		})
	}
	if srv != nil && srv.settings != nil {
		srv.settings.Invalidate()
	}
}

// pinOldScaleMoney **السقوفُ الأربعةُ واسعةٌ لاختبارٍ يكتب مبالغَه بالليرة القديمة.**
func pinOldScaleMoney(t *testing.T, pool *pgxpool.Pool, srv *Server) {
	t.Helper()
	pinUnset(t, pool, srv, "finance.manual_wallet_max", "1000000")
	pinUnset(t, pool, srv, "finance.expense_approval_threshold", "1000000")
	for _, k := range []string{"compensations.cap_driver", "compensations.cap_merchant_goods",
		"compensations.cap_complaint"} {
		pinUnset(t, pool, srv, k, "1000000")
	}
	pinUnset(t, pool, srv, "delivery.custom_fee_min", "1000")
	pinUnset(t, pool, srv, "delivery.custom_fee_max", "1000000")
}
