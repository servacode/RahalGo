package server

// ══════════════════════════════════════════════════════════════════════
// **لقطةُ اقتصادٍ لمِسنَدٍ يُنشئ طلباً بيده** — `XQ-2`
// ══════════════════════════════════════════════════════════════════════
//
// **المنتجُ يكتب اللقطةَ مع كلّ طلب** — **ومِسنَدٌ يُدخل صفّاً بيده
// يصنع حالاً لا ينتجها المنتج**، **وتسويتُها تُردّ بحقّ.**
//
// **والقيمُ من مخزن الإعدادات لا من قراءةٍ خامّة** — **ولا يُكتب
// افتراضٌ مرّتين.**

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/pricing"
	"github.com/servacode/rahalgo/backend/internal/settings"
)

// qaSnapStore يُفتح مرّةً على قاعدة الاختبار — **ولا يُبنى مسبحٌ لكلّ نداء.**
var (
	qaSnapStoreOnce sync.Once
	qaSnapStore     *settings.Store
)

func qaSnapStoreOf() *settings.Store {
	qaSnapStoreOnce.Do(func() {
		dsn := os.Getenv("TEST_DATABASE_URL")
		if dsn == "" {
			return
		}
		if pool, err := pgxpool.New(context.Background(), dsn); err == nil {
			qaSnapStore = settings.NewStore(pool)
		}
	})
	return qaSnapStore
}

// qaSnapSQL أربعُ قيمٍ حرفيّةٍ تُدرَج في `VALUES` — **بقيم اللحظة.**
func qaSnapSQL() string {
	st := qaSnapStoreOf()
	if st == nil {
		return fmt.Sprintf("0, 0, '%s', 1", pricing.SourcePricingMargin)
	}
	c, err := st.ReadCoherent(context.Background(),
		"merchants.commission_percent", "sales.commission_percent",
		pricing.CommissionSourceKey, "sales.activation_orders")
	if err != nil {
		return fmt.Sprintf("0, 0, '%s', 1", pricing.SourcePricingMargin)
	}
	src := c.String(pricing.CommissionSourceKey)
	if src == "" {
		src = string(pricing.SourcePricingMargin)
	}
	return fmt.Sprintf("%d, %d, '%s', %d",
		c.Int("merchants.commission_percent"), c.Int("sales.commission_percent"),
		src, c.Int("sales.activation_orders"))
}
