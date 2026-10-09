package settings_test

import (
	"context"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// **الذاكرةُ لا تُخفي ما حُفظ** (اختبارُ التحمّل ٢٠٢٦-١٠-٠٩): الكتابةُ من المخزن تُرى
// فوراً، والكتابةُ من خارجه تُرى بعد المهلة — **ولا تُرى قبلها** (وإلّا فلا ذاكرة).
func TestCache_FreshAfterWriteAndAfterTTL(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	const key = "orders.auto_transfer"
	st := settings.NewStore(pool)
	st.EnableCache(300 * time.Millisecond)
	t.Cleanup(func() { _ = st.Set(ctx, key, false, nil) })

	if err := st.Set(ctx, key, false, nil); err != nil {
		t.Fatal(err)
	}
	if st.GetBool(ctx, key) {
		t.Fatal("قيمةٌ أولى خاطئة")
	}
	// **من المخزن نفسِه: تُرى فوراً.**
	if err := st.Set(ctx, key, true, nil); err != nil {
		t.Fatal(err)
	}
	if !st.GetBool(ctx, key) {
		t.Fatal("**حُفظ «شغّال» وقُرئ «مطفي» — الذاكرةُ أخفت الحفظ**")
	}
	// **من خارجه (يدٌ في القاعدة): تبقى القديمة حتّى المهلة ثمّ تُرى.**
	if _, err := pool.Exec(ctx, `UPDATE app_settings SET value = 'false' WHERE key = $1`, key); err != nil {
		t.Fatal(err)
	}
	if !st.GetBool(ctx, key) {
		t.Fatal("الذاكرةُ لا تعمل — قُرئت القاعدةُ مباشرة")
	}
	time.Sleep(400 * time.Millisecond)
	if st.GetBool(ctx, key) {
		t.Fatal("**انقضت المهلةُ والقيمةُ القديمةُ باقية**")
	}
	// **والإبطالُ الصريحُ يُري الجديدَ فوراً.**
	if _, err := pool.Exec(ctx, `UPDATE app_settings SET value = 'true' WHERE key = $1`, key); err != nil {
		t.Fatal(err)
	}
	st.Invalidate()
	if !st.GetBool(ctx, key) {
		t.Fatal("**أُبطلت الذاكرةُ والقيمةُ القديمةُ باقية**")
	}
}
