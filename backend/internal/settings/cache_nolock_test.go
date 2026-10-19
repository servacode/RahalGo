package settings

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// slowQuerier **قاعدةٌ يعلق سؤالُها الجماعيّ متى شئنا** — والباقي يمرّ إليها كما هو.
type slowQuerier struct {
	dbtx.Querier
	block   atomic.Bool
	entered chan struct{}
	release chan struct{}
}

func (q *slowQuerier) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if q.block.Load() {
		q.entered <- struct{}{}
		<-q.release
	}
	return q.Querier.Query(ctx, sql, args...)
}

func (q *slowQuerier) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return q.Querier.Exec(ctx, sql, args...)
}

// TestCache_RefreshDoesNotBlockReaders **التجديدُ لا يوقف القرّاء** (مراقبةُ الإنتاج
// ٢٠٢٦-١٠-١٩) — كان يمسك القفلَ وهو ينتظر القاعدة، **فيقف كلُّ قارئٍ خلفه** ومعه
// وصلتُه، حتّى يعلق المسبح. **فالقارئُ الثاني يُجاب من الذاكرة والأوّلُ يجدّد.**
func TestCache_RefreshDoesNotBlockReaders(t *testing.T) {
	pool := testdb.Pool(t)
	q := &slowQuerier{Querier: pool, entered: make(chan struct{}, 1), release: make(chan struct{})}
	st := &Store{db: q}
	st.EnableCache(30 * time.Millisecond)
	ctx := context.Background()

	_ = st.GetBool(ctx, "orders.auto_transfer") // يملأ الذاكرة
	time.Sleep(50 * time.Millisecond)           // تنقضي المهلة

	q.block.Store(true)
	done := make(chan struct{})
	go func() {
		_ = st.GetBool(ctx, "orders.auto_transfer") // يجدّد ويعلق في القاعدة
		close(done)
	}()
	<-q.entered
	q.block.Store(false)

	answered := make(chan struct{})
	go func() {
		_ = st.GetString(ctx, "platform.orders_mode")
		close(answered)
	}()
	select {
	case <-answered:
	case <-time.After(time.Second):
		close(q.release)
		t.Fatal("**القارئُ الثاني وقف خلف التجديد** — والقاعدةُ معلّقة: هذا ما يعلّق المسبح")
	}
	close(q.release)
	<-done
}
