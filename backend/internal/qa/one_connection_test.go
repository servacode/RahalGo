package qa

// ══════════════════════════════════════════════════════════════════════
// **وحدةُ إنشاءِ الطلب تكفيها وصلةٌ واحدة** — `XG-46`
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان يقع
//
// **`WithIdempotentTx` يفتح المعاملةَ ثمّ يُجري العملَ داخلها** (عقدُ
// `XG-33`). **وداخلَه كانت قراءاتٌ تذهب إلى المَسبَح لا إلى المعاملة**:
// المتجرُ أمفتوح · وتحقّقُ واتساب · `SourcesOf` · `priceItems` ·
// `DeliveryAt` · وإعداداتُ التسعير واللقطة.
//
// **فالطلبُ يمسك وصلةً ويطلب ثانيةً وهو ممسكٌ بالأولى.**
//
// # ولماذا هذا عطبُ منتَجٍ لا بطءَ جهاز
//
// **سقفُ المَسبَح في الإنتاج عشرون** (`database/postgres.go`).
// **فعشرون طلبَ إنشاءٍ متزامنةً تمسك العشرين، ثمّ يطلب كلٌّ منها
// ثانيةً** — **ولا تُفكّ إلّا بانتهاء واحدةٍ لا تستطيع أن تنتهي.**
// **جمودٌ تامٌّ حتّى تقتلها مهلةُ الثلاثين ثانية.**
//
// **وقيس في دورةِ ٣٨** أنّ هذا هو ما ضخّم تعثّرَ وصلٍ بيئيّاً إلى
// `500` — **والبابُ نفسُه يُفتَح بالازدحام بلا أيّ عطبِ بيئة.**
//
// # وهذا الفحصُ يقطع الشكّ
//
// **مَسبَحٌ سقفُه واحد**: **إن احتاجت الوحدةُ وصلةً ثانيةً جمدت
// حتماً** — لا احتمالاً. **وإن اكتفت بواحدةٍ مضت.**
//
// **ولا نومَ ولا مهلةٌ تُرفَع**: **المقيسُ أن يعود الردُّ.**

import (
	"context"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// oneConnHarness مِسنَدٌ بمَسبَحٍ سقفُه وصلةٌ واحدة.
//
// **والهجراتُ والقفلُ من `testdb` كما هي** — **ثمّ مَسبَحٌ ضيّقٌ
// لهذا الفحص وحدَه**، ولا يُمَسّ سقفُ الإنتاج.
func oneConnHarness(t *testing.T, max int32) *Harness {
	t.Helper()
	_ = testdb.Pool(t) // **الهجراتُ والقفل** — ثمّ نبني مَسبَحَنا.

	cfg, err := pgxpool.ParseConfig(os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Skipf("وصلةُ الاختبار: %v", err)
	}
	cfg.MaxConns = max
	cfg.MinConns = 0
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("مَسبَحٌ ضيّق: %v", err)
	}
	t.Cleanup(pool.Close)
	return build(t, pool)
}

func TestXG46_OrderCreateNeedsOneConnection(t *testing.T) {
	h := oneConnHarness(t, 1)
	treasury(t, h)
	cust := h.Customer()
	item := h.NewItem(1000)

	type out struct {
		code int
		body string
	}
	done := make(chan out, 1)
	start := time.Now()
	go func() {
		got := h.POSTKey("/api/v1/orders", cust.Token, uniq("xg46"), orderBody(item, 1))
		done <- out{got.Code, string(got.Body)}
	}()

	select {
	case r := <-done:
		took := time.Since(start)
		t.Logf("XG-46: وصلةٌ واحدة ⇒ %d بعد %s", r.code, took.Round(time.Millisecond))
		if r.code >= 400 {
			t.Errorf("**أُنشئ الطلبُ بردٍّ %d على وصلةٍ واحدة**: %s", r.code, r.body)
		}
	case <-time.After(20 * time.Second):
		buf := make([]byte, 2<<20)
		nb := runtime.Stack(buf, true)
		for _, g := range strings.Split(string(buf[:nb]), "\n\n") {
			if strings.Contains(g, "puddle") && strings.Contains(g, "rahalgo") {
				var keep []string
				for _, f := range strings.Split(g, "\n") {
					if strings.Contains(f, "rahalgo") || strings.Contains(f, "puddle") {
						keep = append(keep, strings.TrimSpace(f))
					}
				}
				t.Logf("BLOCKED\n    %s", strings.Join(keep, "\n    "))
			}
		}
		st := h.Pool.Stat()
		t.Fatalf("**جمدت وحدةُ الإنشاء على وصلةٍ واحدة** — "+
			"**تمسك المعاملةَ وتطلب وصلةً ثانيةً لا وجودَ لها.** "+
			"(المَسبَح: محجوزٌ=%d خاملٌ=%d سقفٌ=%d · انتظارٌ=%s) (`XG-46`)",
			st.AcquiredConns(), st.IdleConns(), st.MaxConns(),
			st.AcquireDuration().Round(time.Millisecond))
	}
}
