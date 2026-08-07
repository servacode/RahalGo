package server

// **طلبُ السحب — مدخلُ المال من جهة صاحبه.**
//
// (فحصُ التغطية ٢٠٢٦-٠٨-٠٨، بقرار المالك: «نعم أكمل».)
//
// # لماذا هنا بعينه
//
// حزمةُ `server` كانت عند **٧٫٧٪** رغم ٢٠٣ نقاطِ نهاية. **والعطبان اللذان
// وُجدا في المال والأمان كانا فيها** — ولم يظهرا حتّى كُتب لهما اختبار.
//
// **وقرأتُ هذا المسلكَ فوجدتُه محكماً** — والقراءةُ ليست إثباتاً. فهذه
// شهادةُ ما يفعله فعلاً، تسقط يومَ يتبدّل بلا قصد.
//
// # وما يحرسه
//
//	الدور        ← الزبونُ لا يسحب: رصيدُه للشراء لا للصرف
//	المبلغ       ← لا صفرَ ولا سالبَ ولا أكثرَ من الرصيد
//	الحدُّ الأدنى ← تحويلٌ صغيرٌ تكلفتُه أكبرُ منه
//	الاستثناء    ← **ومن رصيدُه أقلُّ من الحدّ يسحبه كاملاً** وإلّا حُبس ماله
//	طلبٌ واحد    ← فهرسٌ فريدٌ يمنع إغراقَ المالية

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/realtime"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

type payoutFixture struct {
	pool *pgxpool.Pool
	srv  *Server
	min  int64
}

func newPayoutFixture(t *testing.T) *payoutFixture {
	t.Helper()
	pool := testdb.Pool(t)
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	hub := realtime.NewHub(quiet)
	store := settings.NewStore(pool)
	return &payoutFixture{
		pool: pool,
		srv: &Server{
			pg:       pool,
			logger:   quiet,
			hub:      hub,
			wallet:   wallet.NewService(pool),
			settings: store,
			notify:   notifications.New(pool, hub, quiet),
		},
		min: store.GetInt(context.Background(), "payouts.min_amount"),
	}
}

// user ينشئ حساباً بدورٍ ورصيد.
func (f *payoutFixture) user(t *testing.T, role string, balance int64) string {
	t.Helper()
	id := testdb.NewUser(t, f.pool, role)
	if balance > 0 {
		if _, err := f.srv.wallet.Apply(context.Background(), id, balance,
			"topup", "", "رصيدُ اختبار", nil); err != nil {
			t.Fatalf("تعذّر شحنُ الرصيد: %v", err)
		}
	}
	return id
}

func (f *payoutFixture) request(uid, role string, amount int64) *httptest.ResponseRecorder {
	body := `{"amount":` + itoa64(amount) + `,"note":"طلبُ اختبار"}`
	req := httptest.NewRequest(http.MethodPost, "/my/payouts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), ctxUserID, uid)
	ctx = context.WithValue(ctx, ctxRoles, []string{role})
	ctx = context.WithValue(ctx, chi.RouteCtxKey, chi.NewRouteContext())
	w := httptest.NewRecorder()
	f.srv.handleCreatePayout(w, req.WithContext(ctx))
	return w
}

func (f *payoutFixture) pendingCount(t *testing.T, uid string) int {
	t.Helper()
	var n int
	if err := f.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM payout_requests WHERE user_id = $1 AND status = 'pending'`,
		uid).Scan(&n); err != nil {
		t.Fatalf("تعذّر عدُّ الطلبات: %v", err)
	}
	return n
}

func TestCreatePayoutRefusesWhatItMust(t *testing.T) {
	f := newPayoutFixture(t)
	if f.min <= 0 {
		t.Fatalf("`payouts.min_amount` = %d — **والاختبارُ يقيس الحدَّ**، فبلا حدٍّ لا يقيس شيئاً.", f.min)
	}

	t.Run("الزبونُ لا يسحب", func(t *testing.T) {
		uid := f.user(t, "customer", 10*f.min)
		if code := f.request(uid, "customer", f.min).Code; code != http.StatusForbidden {
			t.Fatalf("الزبونُ سحب برمز %d — **ورصيدُه للشراء لا للصرف.**", code)
		}
		if n := f.pendingCount(t, uid); n != 0 {
			t.Fatalf("أُنشئ %d طلباً لمن لا يملك السحب", n)
		}
	})

	t.Run("لا صفرَ ولا سالب", func(t *testing.T) {
		uid := f.user(t, "driver", 10*f.min)
		for _, amount := range []int64{0, -f.min} {
			if code := f.request(uid, "driver", amount).Code; code < 400 {
				t.Fatalf("قُبل مبلغُ %d برمز %d", amount, code)
			}
		}
		if n := f.pendingCount(t, uid); n != 0 {
			t.Fatalf("أُنشئ %d طلباً بمبلغٍ غيرِ صالح", n)
		}
	})

	t.Run("لا أكثرَ من الرصيد", func(t *testing.T) {
		uid := f.user(t, "driver", 2*f.min)
		if code := f.request(uid, "driver", 2*f.min+1).Code; code < 400 {
			t.Fatalf("قُبل سحبٌ يتجاوز الرصيدَ برمز %d — **وهو مالٌ لا وجودَ له.**", code)
		}
	})

	t.Run("لا أقلَّ من الحدّ", func(t *testing.T) {
		uid := f.user(t, "driver", 10*f.min)
		if code := f.request(uid, "driver", f.min-1).Code; code < 400 {
			t.Fatalf("قُبل ما دون الحدّ برمز %d", code)
		}
	})

	// **ومن رصيدُه أقلُّ من الحدّ يسحبه كاملاً** — وإلّا حُبس ماله إلى الأبد.
	t.Run("والرصيدُ الصغيرُ يُسحب كاملاً", func(t *testing.T) {
		small := f.min / 2
		uid := f.user(t, "driver", small)
		if code := f.request(uid, "driver", small).Code; code != http.StatusCreated {
			t.Fatalf("مُنع سحبُ رصيدٍ كاملٍ دون الحدّ برمز %d — **ومالُه محبوسٌ إلى الأبد.**", code)
		}
	})

	t.Run("طلبٌ معلَّقٌ واحد", func(t *testing.T) {
		uid := f.user(t, "merchant", 10*f.min)
		if code := f.request(uid, "merchant", f.min).Code; code != http.StatusCreated {
			t.Fatalf("رُفض الطلبُ الأوّل برمز %d", code)
		}
		if code := f.request(uid, "merchant", f.min).Code; code < 400 {
			t.Fatalf("قُبل طلبٌ ثانٍ والأوّلُ معلَّقٌ برمز %d", code)
		}
		if n := f.pendingCount(t, uid); n != 1 {
			t.Fatalf("طلباتٌ معلّقة: %d — والمنتظَر واحد", n)
		}
	})
}

// **والمتزامنون لا يفتحون طلبين** — الفهرسُ الفريدُ هو ما يحرسه، لا فحصٌ
// في الشيفرة. **وفحصٌ بلا قيدٍ يمرّ عليه اثنان معاً.**
func TestCreatePayoutIsSingleUnderConcurrency(t *testing.T) {
	f := newPayoutFixture(t)
	uid := f.user(t, "driver", 20*f.min)

	const n = 8
	var start, done sync.WaitGroup
	start.Add(1)
	codes := make([]int, n)
	for i := 0; i < n; i++ {
		done.Add(1)
		go func(i int) {
			defer done.Done()
			start.Wait()
			codes[i] = f.request(uid, "driver", f.min).Code
		}(i)
	}
	start.Done()
	done.Wait()

	created := 0
	for _, c := range codes {
		if c == http.StatusCreated {
			created++
		}
	}
	if got := f.pendingCount(t, uid); got != 1 || created != 1 {
		t.Fatalf("**أُنشئ %d طلباً معلّقاً و%d ردٍّ بالإنشاء** — والمنتظَر واحدٌ وواحد.\n"+
			"الردود: %v\n**والفهرسُ الفريدُ هو ما يحرس هذا** — لا الفحصُ في الشيفرة.",
			got, created, codes)
	}
}

func itoa64(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b []byte
	for v > 0 {
		b = append([]byte{byte('0' + v%10)}, b...)
		v /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}
